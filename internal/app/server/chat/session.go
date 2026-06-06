package chat

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"

	. "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/data/history"
	. "xiaozhi-esp32-server-golang/internal/data/msg"
	chathooks "xiaozhi-esp32-server-golang/internal/domain/chat/hooks"
	"xiaozhi-esp32-server-golang/internal/domain/chat/streamtransform"
	user_config "xiaozhi-esp32-server-golang/internal/domain/config"
	"xiaozhi-esp32-server-golang/internal/domain/config/types"
	"xiaozhi-esp32-server-golang/internal/domain/eventbus"
	"xiaozhi-esp32-server-golang/internal/domain/llm"
	llm_common "xiaozhi-esp32-server-golang/internal/domain/llm/common"
	"xiaozhi-esp32-server-golang/internal/domain/mcp"
	"xiaozhi-esp32-server-golang/internal/domain/memory"
	"xiaozhi-esp32-server-golang/internal/domain/memory/llm_memory"
	"xiaozhi-esp32-server-golang/internal/domain/openclaw"
	"xiaozhi-esp32-server-golang/internal/domain/speaker"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

type AsrResponseChannelItem struct {
	ctx           context.Context
	text          string
	speakerResult *speaker.IdentifyResult
}

const detectLLMDebounceDuration = 300 * time.Millisecond

type detectAction string

const (
	detectActionSilent  detectAction = "silent"
	detectActionWelcome detectAction = "welcome"
	detectActionLLM     detectAction = "llm"
)

type welcomePlaybackResult struct {
	natural bool
}

const (
	chatSessionCloseReasonManagerShutdown     = "manager_shutdown"
	chatSessionCloseReasonExplicitExit        = "explicit_exit"
	chatSessionCloseReasonFatalError          = "fatal_error"
	chatSessionCloseReasonAudioIdleTimeout    = "audio_idle_timeout"
	chatSessionCloseReasonRetainedIdleTimeout = "retained_idle_timeout"
)

type ChatSession struct {
	clientState     *ClientState
	asrManager      *ASRManager
	ttsManager      *TTSManager
	llmManager      *LLMManager
	speakerManager  *SpeakerManager
	mediaPlayer     *SessionMediaPlayer
	serverTransport *ServerTransport

	ctx    context.Context
	cancel context.CancelFunc

	chatTextQueue *util.Queue[AsrResponseChannelItem]

	// Temporary storage of voiceprint recognition results (with lock protection)
	speakerResultMu        sync.RWMutex
	pendingSpeakerResult   *speaker.IdentifyResult
	speakerResultReady     chan struct{} //Only used for notification of readiness, no data is transmitted
	turnSpeakerInterrupted atomic.Bool

	vadLoopStarted              bool
	listenStartSeq              atomic.Uint64
	realtimeListenSessionActive atomic.Bool

	// When an inactive device triggers a high-frequency trigger, the most recent "inactivated" determination will be reused in a short period of time to avoid frequent interface calls.
	activationCheckMu     sync.Mutex
	lastActivationFalseAt time.Time

	// Close protection to prevent multiple closes
	closeOnce sync.Once
	closing   atomic.Bool

	// stopSpeaking protection to prevent concurrency conflicts with AddAsrResultToQueue/HandleWelcome
	stopSpeakingMu sync.Mutex

	welcomePlaybackMu     sync.Mutex
	welcomePlaybackDoneCh chan welcomePlaybackResult

	detectLLMDebounceMu    sync.Mutex
	detectLLMDebounceTimer *time.Timer

	openClawStreamMu sync.Mutex
	openClawStreams  map[string]chan llm_common.LLMResponseStruct

	openClawWarmupMu sync.Mutex
	openClawWarmup   *openClawWarmupTask

	hookHub      *chathooks.Hub
	closeHandler func(session *ChatSession, reason string)
}

type ChatSessionOption func(*ChatSession)

func WithChatSessionCloseHandler(handler func(session *ChatSession, reason string)) ChatSessionOption {
	return func(s *ChatSession) {
		s.closeHandler = handler
	}
}

func NewChatSession(clientState *ClientState, serverTransport *ServerTransport, hookHub *chathooks.Hub, transformRegistry *streamtransform.Registry, opts ...ChatSessionOption) *ChatSession {
	s := &ChatSession{
		clientState:        clientState,
		serverTransport:    serverTransport,
		chatTextQueue:      util.NewQueue[AsrResponseChannelItem](10),
		speakerResultReady: make(chan struct{}, 1), //The buffer is 1 to avoid blocking
		openClawStreams:    make(map[string]chan llm_common.LLMResponseStruct),
		hookHub:            hookHub,
	}
	for _, opt := range opts {
		opt(s)
	}

	s.asrManager = NewASRManager(clientState, serverTransport)
	s.asrManager.session = s //Set session reference
	s.ttsManager = NewTTSManager(clientState, serverTransport, s)
	s.mediaPlayer = NewSessionMediaPlayer(s)
	s.llmManager = NewLLMManager(clientState, serverTransport, s.ttsManager, s, transformRegistry)

	clientState.OnVoiceSilenceMetricCallback = func(ctx context.Context, ts int64) {
		s.TraceVoiceSilence(ctx, ts)
	}

	// If voiceprint recognition is enabled, create a voiceprint manager
	if clientState.IsSpeakerEnabled() {
		// Get the voiceprint service address from the system configuration (viper)
		baseURL := viper.GetString("voice_identify.base_url")
		if baseURL != "" {
			// Set service address and threshold to configuration
			speakerConfig := map[string]interface{}{
				"base_url": baseURL,
			}
			// Read the threshold configuration, if not configured use the default value of 0.6
			if viper.IsSet("voice_identify.threshold") {
				threshold := viper.GetFloat64("voice_identify.threshold")
				speakerConfig["threshold"] = threshold
			}

			provider, err := speaker.GetSpeakerProvider(speakerConfig)
			if err != nil {
				log.Warnf("Failed to create voiceprint recognition provider: %v", err)
			} else {
				clientState.SpeakerProvider = provider
				s.speakerManager = NewSpeakerManager(provider)
				log.Debugf("Device %s enables voiceprint recognition", clientState.DeviceID)

				// Set a callback for asynchronously obtaining voiceprint results
				clientState.OnVoiceSilenceSpeakerCallback = func(ctx context.Context) {
					log.Debugf("[Voiceprint Recognition] OnVoiceSilenceSpeakerCallback is called, deviceID: %s", clientState.DeviceID)

					// Get voiceprint results asynchronously
					go func() {
						log.Debugf("[Voiceprint Recognition] Start asynchronously obtaining voiceprint recognition results, deviceID: %s", clientState.DeviceID)

						// Check if speakerManager is activated
						if !s.speakerManager.IsActive() {
							// log.Warnf("[Voiceprint Recognition] speakerManager is not activated and the recognition result cannot be obtained")
							return
						}
						// Clear previous results
						s.speakerResultMu.Lock()
						oldResult := s.pendingSpeakerResult
						s.pendingSpeakerResult = nil
						s.speakerResultMu.Unlock()
						if oldResult != nil {
							log.Debugf("[Voiceprint Recognition] Clear previous recognition results: identified=%v, speaker_id=%s", oldResult.Identified, oldResult.SpeakerID)
						}

						// Clear readiness notification (non-blocking)
						select {
						case <-s.speakerResultReady:
							log.Debugf("[Voiceprint Recognition] Clear the ready notification channel")
						default:
							log.Debugf("[Voiceprint Recognition] Ready notification channel is empty")
						}

						result, err := s.speakerManager.FinishAndIdentify(ctx)
						if err != nil {
							log.Warnf("[Voiceprint Recognition] Failed to obtain voiceprint recognition results: %v, deviceID: %s", err, clientState.DeviceID)
							// Failure of voiceprint recognition does not affect the main process and stores nil results.
							s.speakerResultMu.Lock()
							s.pendingSpeakerResult = nil
							s.speakerResultMu.Unlock()
							log.Debugf("[Voiceprint Recognition] nil result has been stored (recognition failed)")
						} else if result != nil && result.Identified {
							log.Infof("[Voiceprint Recognition] Recognized speaker: %s (Confidence: %.4f, Threshold: %.4f), deviceID: %s",
								result.SpeakerName, result.Confidence, result.Threshold, clientState.DeviceID)
							log.Debugf("[Voiceprint Recognition] Recognition result details: speaker_id=%s, speaker_name=%s, confidence=%.4f, threshold=%.4f",
								result.SpeakerID, result.SpeakerName, result.Confidence, result.Threshold)
							s.speakerResultMu.Lock()
							s.pendingSpeakerResult = result
							s.speakerResultMu.Unlock()
							log.Debugf("[Voiceprint Recognition] Recognition results have been stored (recognized)")
						} else {
							// The speaker is not recognized and the result is also stored.
							if result != nil {
								log.Debugf("[Voiceprint Recognition] Speaker not identified: identified=%v, confidence=%.4f, threshold=%.4f, deviceID: %s",
									result.Identified, result.Confidence, result.Threshold, clientState.DeviceID)
							} else {
								log.Debugf("[Voiceprint recognition] The recognition result is nil, deviceID: %s", clientState.DeviceID)
							}
							s.speakerResultMu.Lock()
							s.pendingSpeakerResult = result
							s.speakerResultMu.Unlock()
							log.Debugf("[Voiceprint Recognition] Recognition results have been stored (not recognized)")
						}

						// Notification result is ready
						select {
						case s.speakerResultReady <- struct{}{}:
							log.Debugf("[Voiceprint Recognition] Result ready notification sent, deviceID: %s", clientState.DeviceID)
						default:
							log.Warnf("[Voiceprint Recognition] The result ready notification channel is full and the notification cannot be sent, deviceID: %s", clientState.DeviceID)
						}
					}()
				}
			}
		}
	}

	// Set the callback for the first character returned by ASR
	clientState.OnAsrFirstTextCallback = func(text string, isFinal bool) {
		clientState.Asr.MarkTextReceived()
		clientState.ClearAudioIdleTimeoutPending()
		clientState.PauseAudioIdleWindow(time.Now())
		log.Debugf("ASR returns characters for the first time: device=%s, text=%s, isFinal=%v", clientState.DeviceID, text, isFinal)
		clientState.MarkAsrFirstText()
		s.TraceAsrFirstText(clientState.Ctx, time.Now().UnixMilli())
		if clientState.IsRealTime() && viper.GetInt("chat.realtime_mode") == 4 {
			if s.isRealtimeMcpAudioGateActive() {
				log.Debugf("Device %s realtime media playback gate control is activated, skipping ASR first word interruption: text=%s", clientState.DeviceID, text)
				return
			}
			s.StopAssistantOutputAfterAsrWithReason(true, "ChatSession.OnAsrFirstTextCallback realtime_mode=4")
		}
	}

	return s
}

func (s *ChatSession) Start(pctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(pctx)

	if s.clientState.InputAudioFormat.SampleRate <= 0 || s.clientState.InputAudioFormat.Channels <= 0 {
		return fmt.Errorf("The input audio format is not initialized, please complete the hello handshake first")
	}

	err := s.InitAsrLlmTts()
	if err != nil {
		log.Errorf("Failed to initialize ASR/LLM/TTS: %v", err)
		return err
	}

	// Asynchronously load historical messages without blocking session startup
	go func() {
		err := s.initHistoryMessages()
		if err != nil {
			log.Errorf("Failed to initialize conversation history: %v", err)
		}
	}()

	if !s.vadLoopStarted {
		// Session-level idle watchdog needs to exist independently of a single ASR loop life cycle.
		// In this way, auto mode can continue to count the connection idle time after a round ends successfully.
		go s.asrManager.runAudioIdleTimeoutWatchdog(s.ctx)
		s.asrManager.ProcessVadAudio(s.ctx)
		s.vadLoopStarted = true
	}

	go s.processChatText(s.ctx)  //Process the conversation message after asr
	go s.llmManager.Start(s.ctx) //Process a series of return messages after llm
	go s.ttsManager.Start(s.ctx) //Processing tts message queue
	if s.mediaPlayer != nil {
		s.mediaPlayer.AttachSession()
	}

	return nil
}

// Initialize historical conversation records to memory
func (s *ChatSession) initHistoryMessages() error {
	var historyMessages []*schema.Message
	var err error

	if s.clientState.GetMemoryMode() == MemoryModeNone {
		log.Debugf("Device %s memory mode=none, skip historical message loading", s.clientState.DeviceID)
		return nil
	}

	// Select the data source according to the configuration (no priority relationship, select directly)
	useRedis := s.shouldUseRedis()
	useManager := s.shouldUseManager()

	// Verify required fields: DeviceID cannot be empty
	if s.clientState.DeviceID == "" {
		log.Debugf("DeviceID is empty, skip historical message loading (may be called before hello message)")
		return nil
	}

	// Select the data source according to the configuration (no priority relationship, select directly)
	if useRedis {
		// Load from Redis
		historyMessages, err = llm_memory.Get().GetMessages(
			s.ctx,
			s.clientState.DeviceID,
			s.clientState.AgentID,
			20)
		if err != nil {
			log.Warnf("Failed to load historical messages from Redis: %v", err)
			return err
		}
		log.Infof("Loaded %d historical messages from Redis", len(historyMessages))
	} else if useManager {
		// Load from Manager
		historyMessages, err = s.loadFromManager()
		if err != nil {
			log.Warnf("Failed to load historical messages from Manager: %v", err)
			return err
		}
		log.Infof("Loaded %d historical messages from Manager", len(historyMessages))
	} else {
		// Both data sources are not configured and historical messages are not loaded.
		log.Debugf("Neither Redis nor Manager is configured, skipping historical message loading")
		return nil
	}

	if len(historyMessages) > 0 {
		s.clientState.InitMessages(historyMessages)
		log.Infof("Successfully loaded %d historical messages", len(historyMessages))
	} else {
		log.Debugf("Not loaded into history messages (maybe no history)")
	}

	return nil
}

// shouldUseRedis determines whether to use Redis as a data source
func (s *ChatSession) shouldUseRedis() bool {
	// Determine based on config_provider.type
	providerType := viper.GetString("config_provider.type")
	return providerType == "redis"
}

// shouldUseManager determines whether to use Manager as the data source
func (s *ChatSession) shouldUseManager() bool {
	// Determine based on config_provider.type
	providerType := viper.GetString("config_provider.type")
	return providerType == "manager"
}

// loadFromManager loads historical messages from the Manager database
func (s *ChatSession) loadFromManager() ([]*schema.Message, error) {
	// Create HistoryClient
	historyCfg := history.HistoryClientConfig{
		BaseURL:   util.GetBackendURL(),
		AuthToken: util.GetManagerAuthToken(),
		Timeout:   viper.GetDuration("manager.history_timeout"),
		Enabled:   true,
	}
	client := history.NewHistoryClient(historyCfg)

	if s.clientState.DeviceID == "" || s.clientState.AgentID == "" {
		return []*schema.Message{}, nil
	}

	req := &history.GetMessagesRequest{
		DeviceID:  s.clientState.DeviceID,
		AgentID:   s.clientState.AgentID,
		SessionID: s.clientState.SessionID,
		Limit:     20,
	}

	resp, err := client.GetMessages(s.ctx, req)
	if err != nil {
		return nil, err
	}

	// Convert to schema.Message format
	messages := make([]*schema.Message, 0, len(resp.Messages))
	for _, item := range resp.Messages {
		var msg *schema.Message
		switch item.Role {
		case "user":
			msg = schema.UserMessage(item.Content)
		case "assistant":
			msg = schema.AssistantMessage(item.Content, item.ToolCalls)
		case "tool":
			msg = schema.ToolMessage(item.Content, item.ToolCallID)
		case "system":
			msg = schema.SystemMessage(item.Content)
		default:
			log.Warnf("Unknown messaging role: %s", item.Role)
			continue
		}

		messages = append(messages, msg)
	}

	for _, msg := range messages {
		log.Debugf("Historical message: %+v", msg)
	}

	return messages, nil
}

// Performed after mqtt receives type: listen, state: start
func (c *ChatSession) InitAsrLlmTts() error {
	// Initialize asr structure
	c.clientState.InitAsr()

	// Initialize memory (memory is not in the resource pool)
	memoryMode := c.clientState.GetMemoryMode()
	memoryConfig := c.clientState.DeviceConfig.Memory
	memoryType := memory.MemoryType(memoryConfig.Provider)
	if memoryMode != MemoryModeLong {
		memoryType = memory.MemoryTypeNone
	}

	memoryProvider, err := memory.GetProvider(memoryType, memoryConfig.Config)
	if err != nil {
		return fmt.Errorf("Failed to create Memory provider: %v", err)
	}
	c.clientState.MemoryProvider = memoryProvider

	if memoryMode == MemoryModeLong {
		// Initialize memory context (long memory mode only)
		context, err := memoryProvider.GetContext(c.ctx, c.clientState.GetDeviceIDOrAgentID(), 500)
		if err != nil {
			log.Warnf("Failed to initialize memory context: %v", err)
		}
		c.clientState.MemoryContext = context
	} else {
		c.clientState.MemoryContext = ""
	}

	return nil
}

// HandleAudioMessage handles audio messages
func (c *ChatSession) HandleAudioMessage(data []byte) bool {
	select {
	case c.clientState.OpusAudioBuffer <- data:
		return true
	default:
		log.Warnf("Audio buffer is full, discarding audio data")
	}
	return false
}

// handleListenMessage handles listening messages
func (s *ChatSession) HandleListenMessage(msg *ClientMessage) error {
	// Process according to status
	switch msg.State {
	case MessageStateStart:
		s.HandleListenStart(msg)
	case MessageStateStop:
		s.HandleListenStop()
	case MessageStateDetect:
		s.HandleListenDetect(msg)
	}

	// logging
	log.Infof("Device %s updates audio listening status: %s", msg.DeviceID, msg.State)
	return nil
}

func (s *ChatSession) beginListenStart() uint64 {
	startSeq := s.listenStartSeq.Add(1)
	if s.clientState.IsRealTime() {
		s.realtimeListenSessionActive.Store(true)
	}
	s.clientState.SetListenPhase(ListenPhaseStarting)
	return startSeq
}

func (s *ChatSession) invalidateListenStart() {
	s.listenStartSeq.Add(1)
	s.realtimeListenSessionActive.Store(false)
	s.clientState.SetListenPhase(ListenPhaseIdle)
}

func (s *ChatSession) isCurrentListenStart(startSeq uint64) bool {
	return startSeq == s.listenStartSeq.Load()
}

func (s *ChatSession) isRealtimeListenSessionActive() bool {
	return s.realtimeListenSessionActive.Load()
}

func (s *ChatSession) shouldIgnoreListenStartError(startSeq uint64, ctx context.Context, err error) bool {
	if !s.isCurrentListenStart(startSeq) {
		return true
	}
	if ctx != nil && ctx.Err() != nil {
		return true
	}
	if s.clientState.Ctx.Err() != nil {
		return true
	}
	return errors.Is(err, context.Canceled)
}

func (s *ChatSession) shouldIgnoreAsrLoopError(startSeq uint64, ctx context.Context, err error) bool {
	if !s.isCurrentListenStart(startSeq) {
		return true
	}
	if ctx != nil && ctx.Err() != nil {
		return true
	}
	if s.clientState.Ctx.Err() != nil {
		return true
	}
	return errors.Is(err, context.Canceled)
}

func isAutoListenActive(state *ClientState) bool {
	if state == nil || state.ListenMode != "auto" {
		return false
	}
	phase := state.GetListenPhase()
	return phase == ListenPhaseStarting || phase == ListenPhaseListening
}

func shouldIgnoreListenStartDuringWelcome(mode string, welcomePlaying bool) bool {
	return mode != "realtime" && welcomePlaying
}

func shouldWaitRealtimeListenStartDuringWelcome(mode string, welcomePlaying bool) bool {
	return false
}

func shouldInterruptOutputOnListenStart(mode string, welcomePlaying bool) bool {
	if mode == "realtime" && welcomePlaying {
		return false
	}
	return true
}

func completeWelcomePlaybackWaitCh(ch chan welcomePlaybackResult, natural bool) {
	if ch == nil {
		return
	}
	select {
	case ch <- welcomePlaybackResult{natural: natural}:
	default:
	}
	close(ch)
}

func (s *ChatSession) beginWelcomePlaybackWait() {
	if s == nil {
		return
	}

	s.welcomePlaybackMu.Lock()
	staleCh := s.welcomePlaybackDoneCh
	s.welcomePlaybackDoneCh = make(chan welcomePlaybackResult, 1)
	s.welcomePlaybackMu.Unlock()

	if staleCh != nil {
		completeWelcomePlaybackWaitCh(staleCh, false)
	}
}

func (s *ChatSession) completeWelcomePlaybackWait(natural bool) {
	if s == nil {
		return
	}

	s.welcomePlaybackMu.Lock()
	ch := s.welcomePlaybackDoneCh
	s.welcomePlaybackDoneCh = nil
	s.welcomePlaybackMu.Unlock()

	completeWelcomePlaybackWaitCh(ch, natural)
}

func (s *ChatSession) currentWelcomePlaybackWaitCh() <-chan welcomePlaybackResult {
	if s == nil {
		return nil
	}

	s.welcomePlaybackMu.Lock()
	ch := s.welcomePlaybackDoneCh
	s.welcomePlaybackMu.Unlock()
	return ch
}

func (s *ChatSession) waitForWelcomePlaybackCompletion() bool {
	if s == nil {
		return true
	}

	doneCh := s.currentWelcomePlaybackWaitCh()
	if doneCh == nil {
		return true
	}

	var sessionDone <-chan struct{}
	if s.ctx != nil {
		sessionDone = s.ctx.Done()
	}

	log.Infof("Device %s realtime listen start waiting for the welcome TTS to end", s.clientState.DeviceID)

	select {
	case result, ok := <-doneCh:
		if !ok {
			log.Infof("Device %s welcome message waiting channel has been closed, cancel realtime listen start", s.clientState.DeviceID)
			return false
		}
		if !result.natural {
			log.Infof("The welcome message of device %s was interrupted, cancel this realtime listen start", s.clientState.DeviceID)
			return false
		}
		log.Infof("The greeting playback of device %s is completed, continue realtime listen start", s.clientState.DeviceID)
		return true
	case <-s.clientState.Ctx.Done():
		log.Debugf("Device %s client ctx canceled, terminated realtime listen start wait", s.clientState.DeviceID)
		return false
	case <-sessionDone:
		log.Debugf("Device %s session ctx canceled, terminated realtime listen start wait", s.clientState.DeviceID)
		return false
	}
}

func resolveDetectAction(text string, enableGreeting bool, welcomeAlreadySpoken bool, autoListenActive bool) detectAction {
	if text == "" {
		return detectActionSilent
	}
	if enableGreeting && isWakeupWord(text) {
		if !welcomeAlreadySpoken {
			return detectActionWelcome
		}
		if autoListenActive {
			return detectActionSilent
		}
		return detectActionLLM
	}
	return detectActionLLM
}

func (s *ChatSession) cancelPendingDetectLLM() {
	if s == nil {
		return
	}

	s.detectLLMDebounceMu.Lock()
	timer := s.detectLLMDebounceTimer
	s.detectLLMDebounceTimer = nil
	s.detectLLMDebounceMu.Unlock()

	if timer != nil {
		timer.Stop()
	}
}

func (s *ChatSession) scheduleDetectLLM(text string) {
	if s == nil {
		return
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	s.cancelPendingDetectLLM()

	var timer *time.Timer
	timer = time.AfterFunc(detectLLMDebounceDuration, func() {
		s.detectLLMDebounceMu.Lock()
		if s.detectLLMDebounceTimer != timer {
			s.detectLLMDebounceMu.Unlock()
			return
		}
		s.detectLLMDebounceTimer = nil
		s.detectLLMDebounceMu.Unlock()

		if s.IsClosing() || s.clientState == nil {
			return
		}
		if s.clientState.Ctx != nil && s.clientState.Ctx.Err() != nil {
			return
		}

		if phase := s.clientState.GetListenPhase(); phase != ListenPhaseIdle {
			log.Debugf("Detect LLM debounce skipped because listen phase=%s", phase)
			return
		}

		if err := s.AddAsrResultToQueue(text, nil); err != nil {
			log.Errorf("Detect LLM debounce enqueue failed: %v", err)
		}
	})

	s.detectLLMDebounceMu.Lock()
	s.detectLLMDebounceTimer = timer
	s.detectLLMDebounceMu.Unlock()
}

func (s *ChatSession) HandleListenDetect(msg *ClientMessage) error {
	// When a new detect arrives, first cancel the previous detect->LLM debounce that has not yet been triggered.
	// Prevent old leading text from being re-queued to the LLM queue at a later time.
	s.cancelPendingDetectLLM()

	// Check device activation status
	isActivated, err := s.CheckDeviceActivated()
	if err != nil {
		log.Errorf("Failed to check device activation status: %v", err)
		return err
	}
	if !isActivated {
		return nil
	}

	// First take a snapshot of the command history "before this detect arrives", and then record the current detect.
	// In this way, the history seen in subsequent logs is the previous command, not the current detect itself.
	now := time.Now()
	prevHistory := s.clientState.GetCommandHistorySnapshot()
	s.clientState.RecordCommandArrival(CommandTypeDetect, now)

	// listen detect means "the device detected a potentially available leading text",
	// We do not directly enter the formal monitoring here, but first determine whether it should trigger a welcome message, silently ignore it, or delay entering LLM.
	if msg.Text != "" {
		text := removePunctuation(msg.Text)
		enableGreeting := viper.GetBool("enable_greeting")
		autoListenActive := isAutoListenActive(s.clientState)
		// The processing of wake-up words is divided into three categories:
		// 1. Wake up for the first time and allow the welcome message: play the welcome message;
		// 2. The welcome message has been played and is currently in auto listen: repeated wake-ups are ignored;
		// 3. In other cases: follow the buffer path of detect -> LLM as normal text.
		action := resolveDetectAction(text, enableGreeting, s.clientState.IsWelcomeSpeaking, autoListenActive)

		log.Debugf(
			"Detect recv: device=%s text=%q action=%s autoListenActive=%v history={%s} welcomeSpeaking=%v welcomePlaying=%v",
			msg.DeviceID,
			text,
			action,
			autoListenActive,
			prevHistory.DebugString(now),
			s.clientState.IsWelcomeSpeaking,
			s.clientState.IsWelcomePlaying,
		)

		if action == detectActionSilent {
			return nil
		}

		// When detect decides to play the welcome message or take over the conversation, it must first stop the current residual output.
		// Avoid the intersection of the old round of TTS/LLM and the new round of detect actions.
		s.StopSpeakingWithReason(true, fmt.Sprintf("HandleListenDetect action=%s text=%q", action, text))

		if action == detectActionWelcome {
			s.HandleWelcome()
			return nil
		}

		if action == detectActionLLM {
			// The text in the detect stage first undergoes a short debounce;
			// If listen start is received immediately later, the formal listening will take over.
			s.scheduleDetectLLM(text)
		}
	}
	return nil
}

func (s *ChatSession) HandleNotActivated() {
	configProvider, err := user_config.GetProvider(viper.GetString("config_provider.type"))
	if err != nil {
		log.Errorf("Failed to get configuration provider: %v", err)
		return
	}

	code, challenge, message, timeoutMs := configProvider.GetActivationInfo(s.clientState.Ctx, s.clientState.DeviceID, "client_id")
	if code == "" {
		log.Errorf("Failed to obtain activation information: %v", err)
		return
	}

	log.Infof("Activation code: %s, Challenge code: %s, Message: %s, Timeout: %d", code, challenge, message, timeoutMs)

	s.ttsManager.EnqueueTtsStartWithReason(s.clientState.Ctx, "HandleNotActivated")
	defer s.ttsManager.EnqueueTtsStopWithReason(s.clientState.Ctx, "HandleNotActivated")

	sessionCtx := s.clientState.SessionCtx.Get(s.clientState.Ctx)
	ctx := s.clientState.AfterAsrSessionCtx.Get(sessionCtx)
	err = s.ttsManager.handleTextResponse(ctx, llm_common.LLMResponseStruct{
		Text: fmt.Sprintf("Please add the device in the admin panel, activation code: %s", code),
	}, false)
	s.ttsManager.RequestTurnEnd(ctx, err)

}

func (s *ChatSession) HandleWelcome() {
	greetingText := s.GetRandomGreeting()

	s.stopSpeakingMu.Lock()
	defer s.stopSpeakingMu.Unlock()

	if s.clientState.Ctx.Err() != nil {
		log.Debugf("HandleWelcome client ctx canceled, skipping welcome message")
		return
	}

	sessionCtx := s.clientState.SessionCtx.Get(s.clientState.Ctx)
	ctx := s.clientState.AfterAsrSessionCtx.Get(sessionCtx)
	if ctx.Err() != nil {
		log.Debugf("HandleWelcome afterAsr ctx canceled, skipping welcome message")
		return
	}

	s.clientState.IsWelcomeSpeaking = true
	s.clientState.IsWelcomePlaying = true
	s.beginWelcomePlaybackWait()

	go func(ctx context.Context, greetingText string) {
		if ctx.Err() != nil || s.clientState.Ctx.Err() != nil {
			s.completeWelcomePlaybackWait(false)
			return
		}

		s.ttsManager.EnqueueTtsStartWithReason(s.clientState.Ctx, "HandleWelcome")
		err := s.ttsManager.handleTextResponse(ctx, llm_common.LLMResponseStruct{Text: greetingText}, true)
		s.ttsManager.EnqueueTtsStopWithReason(s.clientState.Ctx, "HandleWelcome natural end")
		s.ttsManager.RequestTurnEnd(ctx, err)
	}(ctx, greetingText)
}

func (a *ChatSession) checkExitWords(text string) bool {
	exitWords := []string{"goodbye", "farewell", "exit", "exit conversation", "see you"}
	for _, word := range exitWords {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func normalizeOpenClawKeywordText(text string) string {
	return removePunctuation(strings.ToLower(strings.TrimSpace(text)))
}

func containsOpenClawKeyword(text string, keywords []string) bool {
	normalizedText := normalizeOpenClawKeywordText(text)
	if normalizedText == "" {
		return false
	}
	for _, keyword := range keywords {
		normalizedKeyword := normalizeOpenClawKeywordText(keyword)
		if normalizedKeyword == "" {
			continue
		}
		if strings.Contains(normalizedText, normalizedKeyword) {
			return true
		}
	}
	return false
}

func (s *ChatSession) isOpenClawEnterKeyword(text string) bool {
	return containsOpenClawKeyword(text, s.clientState.DeviceConfig.OpenClaw.EnterKeywords)
}

func (s *ChatSession) isOpenClawExitKeyword(text string) bool {
	return containsOpenClawKeyword(text, s.clientState.DeviceConfig.OpenClaw.ExitKeywords)
}

func openClawLogSnippet(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

func (s *ChatSession) GetRandomGreeting() string {
	greetingList := viper.GetStringSlice("greeting_list")
	if len(greetingList) == 0 {
		return "Hello, what's on your mind today?"
	}
	rand.Seed(time.Now().UnixNano())
	return greetingList[rand.Intn(len(greetingList))]
}

func (s *ChatSession) AddTextToTTSQueue(text string) error {
	return s.llmManager.AddTextToTTSQueue(text)
}

func (s *ChatSession) AddTextToTTSQueueWithOptions(text string, options llmResponseChannelOptions) error {
	return s.llmManager.AddTextToTTSQueueWithOptions(text, options)
}

func (s *ChatSession) IsTTSActive() bool {
	if s == nil || s.ttsManager == nil {
		return false
	}
	return s.ttsManager.ttsActive.Load()
}

func (s *ChatSession) getOrCreateOpenClawStream(correlationID string) (chan llm_common.LLMResponseStruct, bool, error) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return nil, false, fmt.Errorf("missing correlation_id")
	}

	s.openClawStreamMu.Lock()
	if existing, ok := s.openClawStreams[correlationID]; ok {
		s.openClawStreamMu.Unlock()
		return existing, false, nil
	}
	streamChan := make(chan llm_common.LLMResponseStruct, 16)
	s.openClawStreams[correlationID] = streamChan
	s.openClawStreamMu.Unlock()

	sessionCtx := s.clientState.SessionCtx.Get(s.clientState.Ctx)
	ctx := s.clientState.AfterAsrSessionCtx.Get(sessionCtx)
	options := llmResponseChannelOptions{}
	hasWarmup := s.getOpenClawWarmupTask(correlationID) != nil
	if hasWarmup {
		options.disableTTSCommands = true
		options.onEndFunc = func(err error, args ...any) {
			// The warm-up takes over the start, and the stop needs to be added here when the official OpenClaw reply is finished;
			// It cannot be sent at the warm-up switching point, otherwise the main reply will be cut off midway.
			if !s.clientState.IsRealTime() {
				s.ttsManager.EnqueueTtsStopWithReason(ctx, fmt.Sprintf("OpenClaw stream end correlation_id=%s", correlationID))
			}
			s.ttsManager.RequestTurnEnd(ctx, err)
			s.finishOpenClawWarmup(correlationID, false)
		}
	}
	log.Infof("OpenClaw stream created: device=%s correlation_id=%s warmup_attached=%v", s.clientState.DeviceID, correlationID, hasWarmup)
	if err := s.llmManager.HandleLLMResponseChannelAsyncWithOptions(ctx, nil, streamChan, options); err != nil {
		if hasWarmup && !s.clientState.IsRealTime() {
			s.ttsManager.EnqueueTtsStopWithReason(ctx, fmt.Sprintf("OpenClaw stream setup failed correlation_id=%s", correlationID))
		}
		if hasWarmup {
			s.ttsManager.RequestTurnEnd(ctx, err)
		}
		s.openClawStreamMu.Lock()
		delete(s.openClawStreams, correlationID)
		s.openClawStreamMu.Unlock()
		close(streamChan)
		return nil, false, err
	}

	return streamChan, true, nil
}

func (s *ChatSession) closeOpenClawStream(correlationID string) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return
	}
	s.openClawStreamMu.Lock()
	delete(s.openClawStreams, correlationID)
	s.openClawStreamMu.Unlock()
}

func (s *ChatSession) clearOpenClawStreams() {
	s.openClawStreamMu.Lock()
	s.openClawStreams = make(map[string]chan llm_common.LLMResponseStruct)
	s.openClawStreamMu.Unlock()
}

func (s *ChatSession) clearPendingSpeakerResult() {
	if s == nil {
		return
	}

	s.speakerResultMu.Lock()
	s.pendingSpeakerResult = nil
	s.speakerResultMu.Unlock()

	for {
		select {
		case <-s.speakerResultReady:
		default:
			return
		}
	}
}

func (s *ChatSession) InjectOpenClawResponse(event openclaw.ResponseDelivery) error {
	correlationID := strings.TrimSpace(event.CorrelationID)
	text := strings.TrimSpace(event.Text)

	// Non-streaming: when there is no correlation_id, directly inject as a single sentence.
	if correlationID == "" {
		if text == "" {
			return nil
		}
		return s.AddTextToTTSQueue(text)
	}

	// The empty fragments in the middle are meaningless and are skipped directly; the empty fragments at the end are reserved for closing.
	if text == "" && !event.IsEnd {
		return nil
	}

	streamChan, created, err := s.getOrCreateOpenClawStream(correlationID)
	if err != nil {
		return err
	}

	isStart := event.IsStart
	if created && !isStart {
		// If the first arriving fragment is not marked start, the first segment will be pulled up.
		isStart = true
	}
	if isStart {
		if task := s.getOpenClawWarmupTask(correlationID); task != nil {
			if text != "" {
				// Only stop the warm-up when the first segment of the actual playable text arrives to avoid premature preemption by too short leading segments.
				// The first paragraph mark of the warm-up is only used for the warm-up TTS, and the IsStart of the first paragraph of the official reply cannot be swallowed.
				// Otherwise, the formal reply will be downgraded to a single-sentence TTS, and subsequent snapshots will be broadcast again as the second sentence.
				s.cancelOpenClawWarmup(correlationID, false)
				s.beginOpenClawSpeech(task)
			} else {
				isStart = false
			}
		}
	} else if event.IsEnd {
		s.cancelOpenClawWarmup(correlationID, false)
	}

	resp := llm_common.LLMResponseStruct{
		Text:    text,
		IsStart: isStart,
		IsEnd:   event.IsEnd,
	}

	select {
	case <-s.ctx.Done():
		return fmt.Errorf("chat session closed")
	case streamChan <- resp:
	}

	if event.IsEnd {
		s.closeOpenClawStream(correlationID)
	}

	return nil
}

// InterruptAndClearTTSQueue triggers TTS interruption and clears the sending queue (called for realtime mode VAD interruption and other scenarios)
func (s *ChatSession) InterruptAndClearTTSQueue() {
	s.InterruptAndClearTTSQueueWithReason("ChatSession.InterruptAndClearTTSQueue")
}

func (s *ChatSession) InterruptAndClearTTSQueueWithReason(reason string) {
	log.Infof("interrupt and clear tts queue requested: device=%s reason=%s state={%s}", s.clientState.DeviceID, normalizeTTSReason(reason), s.ttsManager.debugState())
	if s.mediaPlayer != nil {
		if err := s.mediaPlayer.Suspend(); err != nil && !errors.Is(err, context.Canceled) {
			log.Warnf("Suspended media playback failed: %v", err)
		}
	}
	s.ttsManager.ClearTTSQueue()
	s.ttsManager.InterruptAndStopWithReason(s.clientState.Ctx, true, context.Canceled, reason)
}

// handleAbortMessage handles the abort message
func (s *ChatSession) HandleAbortMessage(msg *ClientMessage) error {
	s.cancelPendingDetectLLM()

	// Set interruption status
	s.clientState.Abort = true

	if s.clientState.IsRealTime() {
		s.StopAssistantOutputAfterAsrWithReason(true, "HandleAbortMessage realtime")
	} else {
		s.StopSpeakingWithReason(true, "HandleAbortMessage auto")
	}

	// logging
	log.Infof("Device %s abort session", msg.DeviceID)
	return nil
}

func (s *ChatSession) CheckDeviceActivated() (bool, error) {
	if viper.GetBool("auth.enable") {
		if !s.clientState.IsActivated {
			const falseCheckThrottle = time.Second
			s.activationCheckMu.Lock()
			lastFalseAt := s.lastActivationFalseAt
			s.activationCheckMu.Unlock()
			if !lastFalseAt.IsZero() && time.Since(lastFalseAt) < falseCheckThrottle {
				log.Debugf("The activation status of device %s is still inactive, skipping repeated real-time verification", s.clientState.DeviceID)
				return false, nil
			}

			configProvider, err := user_config.GetProvider(viper.GetString("config_provider.type"))
			if err != nil {
				log.Errorf("Failed to get configuration provider: %v", err)
				return false, err
			}
			// Call the interface to confirm the activation status again
			isActivated, err := configProvider.IsDeviceActivated(s.clientState.Ctx, s.clientState.DeviceID, "client_id")
			if err != nil {
				log.Errorf("Failed to get activation status: %v", err)
				return false, err
			}
			if isActivated {
				s.clientState.IsActivated = true
				s.activationCheckMu.Lock()
				s.lastActivationFalseAt = time.Time{}
				s.activationCheckMu.Unlock()
			} else {
				s.activationCheckMu.Lock()
				s.lastActivationFalseAt = time.Now()
				s.activationCheckMu.Unlock()
				s.HandleNotActivated()
				return false, nil
			}
		}
	}
	return true, nil
}

func (s *ChatSession) HandleListenStart(msg *ClientMessage) error {
	s.cancelPendingDetectLLM()

	// Check activation status first
	isActivated, err := s.CheckDeviceActivated()
	if err != nil {
		log.Errorf("Failed to check device activation status: %v", err)
		return err
	}
	if !isActivated {
		return nil
	}

	now := time.Now()
	prevHistory := s.clientState.GetCommandHistorySnapshot()

	// In auto/manual mode, the device may automatically reissue listen start during the welcome message playback;
	// Such packets should not preempt the greeting, so they are simply ignored while the greeting is still playing.
	if shouldIgnoreListenStartDuringWelcome(msg.Mode, s.clientState.IsWelcomePlaying) {
		log.Infof("The greeting message of device %s is playing, ignore listen start: history={%s}", msg.DeviceID, prevHistory.DebugString(now))
		return nil
	}

	log.Debugf(
		"ListenStart recv: device=%s mode=%s history={%s} welcomeSpeaking=%v welcomePlaying=%v phase=%s",
		msg.DeviceID,
		msg.Mode,
		prevHistory.DebugString(now),
		s.clientState.IsWelcomeSpeaking,
		s.clientState.IsWelcomePlaying,
		s.clientState.GetListenPhase(),
	)

	// The processing goals of realtime and auto/manual are different:
	// realtime is more like a "permanent listening session", trying not to interrupt the current link;
	// auto/manual is more like "start a new round of official pickup", which will reset the current output and restart ASR.
	if msg.Mode == "realtime" {
		// When the current realtime listen session has not reached listen stop / session cancel / close,
		// Repeated listen start packets are uniformly and silently ignored to avoid interrupting the current link.
		if s.clientState.IsRealTime() && s.isRealtimeListenSessionActive() {
			return nil
		}

		// When realtime is entered for the first time, if the welcome message is still playing, wait for it to end naturally;
		// Only when the welcome message is completely played, can you continue to enter realtime listen.
		if shouldWaitRealtimeListenStartDuringWelcome(msg.Mode, s.clientState.IsWelcomePlaying) {
			if !s.waitForWelcomePlaybackCompletion() {
				return nil
			}
		}

		s.clientState.RecordCommandArrival(CommandTypeListenStart, now)
		if shouldInterruptOutputOnListenStart(msg.Mode, s.clientState.IsWelcomePlaying) {
			// In non-greeting protection scenarios, listen start represents a new round of listening takeover.
			// It is necessary to actively stop the current TTS/LLM to avoid speaking and listening at the same time.
			s.StopSpeakingWithReason(true, fmt.Sprintf("HandleListenStart mode=%s", msg.Mode))
		}

		s.clientState.ListenMode = msg.Mode
		log.Infof("Device %s pickup pattern: %s", msg.DeviceID, msg.Mode)

		shouldStartAudioIdleWindow := s.clientState.GetListenPhase() != ListenPhaseListening
		startSeq := s.beginListenStart()
		go func() {
			if err := s.OnListenStart(startSeq, shouldStartAudioIdleWindow); err != nil {
				log.Errorf("Device %s listen start failed to start: %v", msg.DeviceID, err)
			}
		}()
		return nil
	}

	if s.clientState.GetListenPhase() == ListenPhaseStarting {
		log.Infof("Device %s listen start is starting, ignore repeated listen start", msg.DeviceID)
		return nil
	}

	s.clientState.RecordCommandArrival(CommandTypeListenStart, now)

	// When the auto/manual mode is entered here, it is regarded as explicitly starting a new round of pickup process:
	// Update the mode, stop the old output, and then asynchronously raise OnListenStart to do ASR initialization.
	s.clientState.ListenMode = msg.Mode
	log.Infof("Device %s pickup pattern: %s", msg.DeviceID, msg.Mode)
	s.StopSpeakingWithReason(true, fmt.Sprintf("HandleListenStart mode=%s", msg.Mode))

	startSeq := s.beginListenStart()
	go func() {
		if err := s.OnListenStart(startSeq, true); err != nil {
			log.Errorf("Device %s listen start failed to start: %v", msg.DeviceID, err)
		}
	}()

	return nil
}

func (s *ChatSession) HandleListenStop() error {
	s.cancelPendingDetectLLM()
	s.clientState.RecordCommandArrival(CommandTypeListenStop, time.Now())
	/*if s.clientState.ListenMode == "auto" {
		s.clientState.CancelSessionCtx()
	}*/

	// call
	if s.clientState.IsRealTime() {
		s.invalidateListenStart()
	}
	s.clientState.OnManualStop()

	return nil
}

func (s *ChatSession) OnListenStart(startSeq uint64, shouldStartAudioIdleWindow bool) error {
	log.Debugf("OnListenStart start")
	defer log.Debugf("OnListenStart end")

	if !s.isCurrentListenStart(startSeq) {
		log.Debugf("OnListenStart stale before init, skip")
		return nil
	}

	select {
	case <-s.clientState.Ctx.Done():
		log.Debugf("OnListenStart Ctx done, return")
		if s.isCurrentListenStart(startSeq) {
			s.clientState.SetListenPhase(ListenPhaseIdle)
		}
		return nil
	default:
	}

	// realtime mode: skip Destroy, keep ASR running, but clear AudioBuffer
	var ctx context.Context
	if s.clientState.IsRealTime() {
		s.clientState.AsrAudioBuffer.ClearAsrAudioData()
	} else {
		s.stopSpeakingMu.Lock()
		if !s.isCurrentListenStart(startSeq) {
			s.stopSpeakingMu.Unlock()
			log.Debugf("OnListenStart stale before destroy, skip")
			return nil
		}
		s.clientState.Destroy()
		if !s.isCurrentListenStart(startSeq) {
			s.stopSpeakingMu.Unlock()
			log.Debugf("OnListenStart stale after destroy, skip")
			return nil
		}

		s.clientState.SetListenPhase(ListenPhaseStarting)
		s.clientState.SetStatus(ClientStatusListening)
		ctx = s.clientState.SessionCtx.Get(s.clientState.Ctx)

		// Initializing ASR related state needs to be consistent with session context reconstruction.
		if s.clientState.ListenMode == "manual" {
			s.clientState.VoiceStatus.SetClientHaveVoice(true)
		}
		s.stopSpeakingMu.Unlock()
	}

	if s.clientState.IsRealTime() {
		s.clientState.SetListenPhase(ListenPhaseStarting)
		s.clientState.SetStatus(ClientStatusListening)
		ctx = s.clientState.SessionCtx.Get(s.clientState.Ctx)

		// Initialization asr related
		if s.clientState.ListenMode == "manual" {
			s.clientState.VoiceStatus.SetClientHaveVoice(true)
		}
	}

	// Start asr streaming recognition and reuse the restartAsrRecognition function
	if !s.isCurrentListenStart(startSeq) {
		log.Debugf("OnListenStart stale before ASR restart, skip")
		return nil
	}
	err := s.asrManager.RestartAsrRecognition(ctx)
	if err != nil {
		if s.shouldIgnoreListenStartError(startSeq, ctx, err) {
			log.Infof("OnListenStart interrupted during ASR restart, ignore err: %v", err)
			if s.isCurrentListenStart(startSeq) {
				s.clientState.SetListenPhase(ListenPhaseIdle)
			}
			return nil
		}

		log.Errorf("asr streaming recognition failed: %v", err)
		if s.isCurrentListenStart(startSeq) {
			s.clientState.SetListenPhase(ListenPhaseIdle)
		}
		s.CloseWithReason(chatSessionCloseReasonFatalError)
		return err
	}

	if !s.isCurrentListenStart(startSeq) {
		log.Debugf("OnListenStart stale after ASR restart, cancel current start")
		s.clientState.Asr.CancelWithReason("ChatSession.OnListenStart: stale listen start after ASR restart")
		return nil
	}

	s.clientState.SetListenPhase(ListenPhaseListening)
	if shouldStartAudioIdleWindow {
		s.clientState.StartAudioIdleWindow(time.Now())
	}

	// Define message save callback
	onMessageSave := func(userMsg *schema.Message, messageID string, audioData []float32) {
		// ASR text and audio are obtained at the same time and saved at one time (no need for two stages)
		eventbus.Get().Publish(eventbus.TopicAddMessage, &eventbus.AddMessageEvent{
			ClientState: s.clientState,
			Msg:         *userMsg,
			MessageID:   messageID,
			AudioData:   [][]byte{util.Float32SliceToBytes(audioData)}, //Convert to byte array
			AudioSize:   len(audioData) * 4,                            // float32 = 4 bytes
			SampleRate:  s.clientState.InputAudioFormat.SampleRate,
			Channels:    s.clientState.InputAudioFormat.Channels,
			IsUpdate:    false, //Save once (text + audio)
			Timestamp:   time.Now(),
		})
	}

	// Define error handling callbacks
	onError := func(err error) {
		if s.shouldIgnoreAsrLoopError(startSeq, ctx, err) {
			log.Infof("ASR recognition loop ends in reset/exit, ignoring err: %v", err)
			return
		}
		log.Errorf("ASR recognition loop error: %v", err)
		s.CloseWithReason(chatSessionCloseReasonFatalError)
	}

	// Start the ASR recognition result processing loop (resource management is inside ASRManager)
	s.asrManager.StartAsrRecognitionLoop(ctx, onMessageSave, onError)

	return nil
}

// startChat Start a conversation
func (s *ChatSession) AddAsrResultToQueue(text string, speakerResult *speaker.IdentifyResult) error {
	return s.AddAsrResultToQueueWithOptions(text, speakerResult, llmResponseChannelOptions{})
}

func (s *ChatSession) AddAsrResultToQueueWithOptions(text string, speakerResult *speaker.IdentifyResult, options llmResponseChannelOptions) error {
	log.Debugf("AddAsrResultToQueue text: %s", text)
	if speakerResult != nil && speakerResult.Identified {
		log.Debugf("AddAsrResultToQueue speaker: %s (confidence: %.2f)", speakerResult.SpeakerName, speakerResult.Confidence)
	}

	// Check if the session has been stopped (determined by trying to acquire the lock)
	// If StopSpeaking is being executed, it will wait here; if the execution is completed, tryLock will return immediately.
	if !s.stopSpeakingMu.TryLock() {
		log.Debugf("AddAsrResultToQueue is executing StopSpeaking, discarding the message")
		return nil
	}
	s.stopSpeakingMu.Unlock()

	sessionCtx := s.clientState.SessionCtx.Get(s.clientState.Ctx)
	// Check if sessionCtx has been canceled
	if sessionCtx.Err() != nil {
		log.Debugf("AddAsrResultToQueue sessionCtx canceled, message discarded")
		return nil
	}
	ctx := s.clientState.AfterAsrSessionCtx.Get(sessionCtx)
	ctx = withTTSPlaybackStartHook(ctx, options.onTTSPlaybackStart)
	ctx = withTTSTurnEndPolicy(ctx, options.ttsTurnEndPolicy)

	item := AsrResponseChannelItem{
		ctx:           ctx,
		text:          text,
		speakerResult: speakerResult,
	}
	err := s.chatTextQueue.Push(item)
	if err != nil {
		log.Warnf("chatTextQueue is full or closed, discarding message")
	}
	return nil
}

func (s *ChatSession) processChatText(ctx context.Context) {
	log.Debugf("processChatText start")
	defer log.Debugf("processChatText end")

	for {
		item, err := s.chatTextQueue.Pop(ctx, 0)
		if err != nil {
			if err == util.ErrQueueCtxDone {
				return
			}
			continue
		}

		err = s.actionDoChat(item.ctx, item.text, item.speakerResult)
		if err != nil {
			log.Errorf("Failed to process conversation: %v", err)
			continue
		}
	}
}

func (s *ChatSession) ClearChatTextQueue() {
	s.chatTextQueue.Clear()
}

// DoExitChat performs exit chat logic (sends goodbye message and closes conversation)
func (s *ChatSession) DoExitChat() {
	// friendly goodbye
	goodbyeText := "Alright, goodbye! Looking forward to chatting again~"

	// Save a message from the assistant role
	goodbyeMsg := schema.AssistantMessage(goodbyeText, nil)
	if err := s.llmManager.AddLlmMessage(s.clientState.Ctx, goodbyeMsg); err != nil {
		log.Errorf("Failed to save goodbye message: %v", err)
	}

	// Get context
	sessionCtx := s.clientState.SessionCtx.Get(s.clientState.Ctx)
	ctx := s.clientState.AfterAsrSessionCtx.Get(sessionCtx)

	// Send TTS goodbye message
	s.ttsManager.EnqueueTtsStartWithReason(ctx, "ChatSession.processGoodbye")

	err := s.ttsManager.handleTextResponse(ctx, llm_common.LLMResponseStruct{
		Text:    goodbyeText,
		IsStart: true,
		IsEnd:   true,
	}, true) //Synchronous processing, waiting for TTS to complete

	if err != nil {
		log.Errorf("Failed to send goodbye message: %v", err)
	}

	s.ttsManager.RequestTurnEnd(ctx, err)
	s.ttsManager.EnqueueTtsStopWithReason(ctx, "ChatSession.processGoodbye")
	// Close session
	s.CloseWithReason(chatSessionCloseReasonExplicitExit)
}

func (s *ChatSession) Close() {
	s.CloseWithReason(chatSessionCloseReasonManagerShutdown)
}

func (s *ChatSession) IsClosing() bool {
	if s == nil {
		return true
	}
	return s.closing.Load()
}

func (s *ChatSession) CloseWithReason(reason string) {
	s.closing.Store(true)
	s.closeOnce.Do(func() {
		// Clean up ASR resources (resource management is inside ASRManager)
		if s.asrManager != nil {
			s.asrManager.Cleanup()
		}
		deviceID := ""
		if s.clientState != nil {
			deviceID = s.clientState.DeviceID
		}
		log.Debugf("ChatSession.Close() starts cleaning up session resources, device %s", deviceID)

		if s.mediaPlayer != nil {
			s.mediaPlayer.DetachSession(true)
		}

		s.cancelPendingDetectLLM()

		// Cancel session level context
		if s.cancel != nil {
			s.cancel()
		}
		s.finishOpenClawWarmup("", false)

		// Clear chat text queue
		s.ClearChatTextQueue()
		s.clearOpenClawStreams()

		// Stop talking and clean up audio related resources. The Close path has DetachSession(true) in front of it,
		// Do not Suspend the media again here, otherwise resumeOnAttach will be cleared.
		s.stopSpeakingWithLock(true, true, false, "ChatSession.Close")

		if s.speakerManager != nil {
			s.speakerManager.Close()
		}

		if s.clientState != nil {
			eventbus.Get().Publish(eventbus.TopicSessionEnd, s.clientState)
		}

		log.Debugf("ChatSession.Close() Session resource cleanup completed, device %s", deviceID)

		if s.closeHandler != nil {
			s.closeHandler(s, reason)
		}
	})
}

func (s *ChatSession) actionDoChat(ctx context.Context, text string, speakerResult *speaker.IdentifyResult) error {
	select {
	case <-ctx.Done():
		log.Debugf("actionDoChat ctx done, return")
		return nil
	default:
	}

	agentID := strings.TrimSpace(s.clientState.AgentID)
	deviceID := strings.TrimSpace(s.clientState.DeviceID)
	openclawSessionID := strings.TrimSpace(s.clientState.SessionID)
	trimmedText := strings.TrimSpace(text)

	handledByRealtimeGate, gateErr := s.tryHandleRealtimeMcpAudioASR(ctx, trimmedText)
	if handledByRealtimeGate {
		return gateErr
	}

	openclawManager := openclaw.GetManager()
	if s.clientState.DeviceConfig.OpenClaw.Allowed {
		isOpenClawMode := openclawManager.IsModeEnabled(agentID, deviceID)
		isEnterKeyword := s.isOpenClawEnterKeyword(text)
		isExitKeyword := false
		if isOpenClawMode {
			isExitKeyword = s.isOpenClawExitKeyword(text)
		}
		log.Debugf(
			"OpenClaw routing determination: agent=%s device=%s session=%s allowed=%v mode=%v enter_keyword=%v exit_keyword=%v text_len=%d text_trim_len=%d text_snippet=%q",
			agentID,
			deviceID,
			openclawSessionID,
			s.clientState.DeviceConfig.OpenClaw.Allowed,
			isOpenClawMode,
			isEnterKeyword,
			isExitKeyword,
			len(text),
			len(trimmedText),
			openClawLogSnippet(trimmedText, 64),
		)
		if isOpenClawMode {
			if isExitKeyword {
				s.finishOpenClawWarmup("", true)
				exited := openclawManager.ExitMode(agentID, deviceID)
				_ = s.AddTextToTTSQueue("Exited OpenClaw mode")
				log.Infof("Device %s exited OpenClaw mode: agent=%s exited=%v", deviceID, agentID, exited)
				return nil
			}

			log.Infof(
				"OpenClaw sends STT: agent=%s device=%s session=%s text_len=%d text_snippet=%q",
				agentID,
				deviceID,
				openclawSessionID,
				len(trimmedText),
				openClawLogSnippet(trimmedText, 64),
			)
			s.finishOpenClawWarmup("", true)
			messageID, err := openclawManager.SendMessage(
				agentID,
				deviceID,
				text,
				openclawSessionID,
			)
			if err != nil {
				log.Warnf(
					"Device %s failed to send OpenClaw message and has fallen back to normal mode: agent=%s session=%s text_snippet=%q err=%v",
					deviceID,
					agentID,
					openclawSessionID,
					openClawLogSnippet(trimmedText, 64),
					err,
				)
				openclawManager.ExitMode(agentID, deviceID)
				_ = s.AddTextToTTSQueue("OpenClaw is unavailable, exited OpenClaw mode")
			} else {
				s.startOpenClawWarmup(messageID, text)
				log.Infof("OpenClaw sends STT successfully: agent=%s device=%s session=%s message_id=%s", agentID, deviceID, openclawSessionID, messageID)
			}
			return nil
		}

		if isEnterKeyword {
			if !openclawManager.EnterMode(agentID, deviceID) {
				_ = s.AddTextToTTSQueue("OpenClaw is unavailable, please try again later")
				log.Warnf("Device %s failed to enter OpenClaw mode: agent=%s agent session not ready", deviceID, agentID)
				return nil
			}
			_ = s.AddTextToTTSQueue("Entered OpenClaw mode, please continue speaking")
			log.Infof("Device %s enters OpenClaw mode: agent=%s trigger=%q", deviceID, agentID, openClawLogSnippet(trimmedText, 32))
			return nil
		}
		log.Debugf(
			"OpenClaw has not taken over the current STT: agent=%s device=%s mode=%v enter_keyword=%v text_snippet=%q",
			agentID,
			deviceID,
			isOpenClawMode,
			isEnterKeyword,
			openClawLogSnippet(trimmedText, 64),
		)
	} else {
		s.finishOpenClawWarmup("", false)
		if openclawManager.ExitMode(agentID, deviceID) {
			log.Debugf("OpenClaw configuration is not enabled and has been forced out of mode: agent=%s device=%s", agentID, deviceID)
		}
	}

	if s.checkExitWords(text) {
		// Post exit chat event
		eventbus.Get().Publish(eventbus.TopicExitChat, &eventbus.ExitChatEvent{
			ClientState: s.clientState,
			Reason:      "User initiated exit",
			TriggerType: "exit_words",
			UserText:    text,
			Timestamp:   time.Now(),
		})
		return nil
	}

	clientState := s.clientState

	sessionID := clientState.SessionID

	// Dynamically switch TTS after voiceprint recognition (restore to default TTS when not recognized)
	if err := s.switchTTSForSpeaker(speakerResult); err != nil {
		log.Warnf("Failed to switch TTS: %v", err)
		// Do not interrupt the process and continue to use the current TTS
	}

	// Create Eino native messages directly
	userMessage := &schema.Message{
		Role:    schema.User,
		Content: text,
	}

	// Get global MCP tool list
	mcpTools, err := mcp.GetToolsByDeviceIdWithTransport(
		clientState.DeviceID,
		clientState.AgentID,
		s.serverTransport.GetTransportType(),
		clientState.DeviceConfig.MCPServiceNames,
	)
	if err != nil {
		log.Errorf("Failed to get tool for device %s: %v", clientState.DeviceID, err)
		mcpTools = make(map[string]tool.InvokableTool)
	}
	if !hasAvailableKnowledgeBase(clientState.DeviceConfig.KnowledgeBases) {
		if _, ok := mcpTools["search_knowledge"]; ok {
			delete(mcpTools, "search_knowledge")
			log.Infof("Device %s is not associated with an available knowledge base and the tool search_knowledge has been removed", clientState.DeviceID)
		}
	}

	// Convert MCP tool to interface format for passing to conversion function
	mcpToolsInterface := make(map[string]interface{})
	for name, tool := range mcpTools {
		mcpToolsInterface[name] = tool
	}

	// Convert MCP tools to Eino ToolInfo format
	einoTools, err := llm.ConvertMCPToolsToEinoTools(ctx, mcpToolsInterface)
	if err != nil {
		log.Errorf("Convert MCP tool failed: %v", err)
		einoTools = nil
	}

	toolNameList := make([]string, 0)
	for _, tool := range einoTools {
		toolNameList = append(toolNameList, tool.Name)
	}

	// Send LLM request with tool
	log.Infof("Use %d MCP tools to send LLM requests, tools: %+v", len(einoTools), toolNameList)

	err = s.llmManager.DoLLmRequest(ctx, userMessage, einoTools, true, speakerResult)
	if err != nil {
		log.Errorf("Sending LLM request with tools failed, sessionID: %s, error: %v", sessionID, err)
		return fmt.Errorf("Sending LLM request with tools failed: %v", err)
	}
	return nil
}

func hasAvailableKnowledgeBase(knowledgeBases []types.KnowledgeBaseRef) bool {
	for _, kb := range knowledgeBases {
		if strings.EqualFold(strings.TrimSpace(kb.Status), "inactive") {
			continue
		}
		if strings.TrimSpace(kb.ExternalKBID) == "" {
			continue
		}
		return true
	}
	return false
}

func (s *ChatSession) MarkTurnSpeakerInterrupted() {
	if s == nil {
		return
	}
	s.turnSpeakerInterrupted.Store(true)
}

func (s *ChatSession) ConsumeTurnSpeakerInterrupted() bool {
	if s == nil {
		return false
	}
	return s.turnSpeakerInterrupted.Swap(false)
}

func (s *ChatSession) ResetTurnSpeakerInterrupted() {
	if s == nil {
		return
	}
	s.turnSpeakerInterrupted.Store(false)
}

func (s *ChatSession) ShouldAllowSpeakerChat(speakerResult *speaker.IdentifyResult, speakerInterrupted bool) (bool, string) {
	if s == nil || s.clientState == nil {
		return true, ""
	}

	matchedConfiguredSpeaker := s.clientState.HasMatchedConfiguredSpeaker(speakerResult)
	if speakerInterrupted && !matchedConfiguredSpeaker {
		return false, "speaker_interrupt_without_identify"
	}

	if s.clientState.RequireMatchedSpeakerForChat() && !matchedConfiguredSpeaker {
		return false, "speaker_chat_mode_identified_only_not_matched"
	}

	return true, ""
}

// switchTTSForSpeaker switches TTS for the identified speaker
func (s *ChatSession) switchTTSForSpeaker(speakerResult *speaker.IdentifyResult) error {
	s.clientState.SpeakerTTSConfig = nil

	// 1. Check if speakerResult is nil
	if speakerResult == nil {
		log.Debug("speakerResult is nil, clear the voiceprint TTS configuration")
		return nil
	}

	// 2. Find the voiceprint group configuration
	speakerGroupInfo, found := s.clientState.DeviceConfig.VoiceIdentify[speakerResult.SpeakerName]
	if !found {
		// Configuration not found, clear voiceprint TTS configuration
		log.Debugf("The configuration of voiceprint group %s was not found. Clear the voiceprint TTS configuration.", speakerResult.SpeakerName)
		return nil
	}

	// 3. Check whether a custom tone is configured
	if speakerGroupInfo.TTSConfigID == nil || *speakerGroupInfo.TTSConfigID == "" {
		// No custom timbre is configured, clear the voiceprint TTS configuration
		log.Debugf("Voiceprint group %s is not configured with custom TTS. Clear the voiceprint TTS configuration.", speakerResult.SpeakerName)
		return nil
	}

	// 4. Find the corresponding TTS configuration from the system configuration (viper)
	var targetTTSConfig *types.TtsConfigItem
	ttsConfigsRaw := viper.Get("tts")
	if ttsConfigsRaw == nil {
		return fmt.Errorf("tts not found in system configuration")
	}

	// Parse tts configuration (now a map, key is config_id)
	if ttsConfigsMap, ok := ttsConfigsRaw.(map[string]interface{}); ok {
		// Find matching config_id
		if configItem, exists := ttsConfigsMap[*speakerGroupInfo.TTSConfigID]; exists {
			if configMap, ok := configItem.(map[string]interface{}); ok {
				// Parse configuration items
				ttsItem := &types.TtsConfigItem{
					ConfigID: *speakerGroupInfo.TTSConfigID,
				}
				if name, ok := configMap["name"].(string); ok {
					ttsItem.Name = name
				}
				if provider, ok := configMap["provider"].(string); ok {
					ttsItem.Provider = provider
				}
				if isDefault, ok := configMap["is_default"].(bool); ok {
					ttsItem.IsDefault = isDefault
				}
				// Other fields of the configuration item are directly used as config
				ttsItem.Config = make(map[string]interface{})
				for k, v := range configMap {
					if k != "name" && k != "provider" && k != "is_default" && k != "config_id" {
						ttsItem.Config[k] = v
					}
				}
				targetTTSConfig = ttsItem
			}
		}
	}

	if targetTTSConfig == nil {
		return fmt.Errorf("TTS configuration %s not found", *speakerGroupInfo.TTSConfigID)
	}

	// 5. Copy the TTS configuration to avoid modifying the original configuration
	ttsConfig := make(map[string]interface{})
	for k, v := range targetTTSConfig.Config {
		ttsConfig[k] = v
	}

	// 6. If the timbre value is configured, overwrite it in the TTS configuration.
	if speakerGroupInfo.Voice != nil && *speakerGroupInfo.Voice != "" {
		// Set the corresponding timbre field according to the provider
		if targetTTSConfig.Provider == "cosyvoice" {
			ttsConfig["spk_id"] = *speakerGroupInfo.Voice
		} else {
			ttsConfig["voice"] = *speakerGroupInfo.Voice
		}
		log.Debugf("Set timbre for speaker %s: %s", speakerResult.SpeakerName, *speakerGroupInfo.Voice)
	}
	if targetTTSConfig.Provider == "aliyun_qwen" &&
		speakerGroupInfo.VoiceModelOverride != nil &&
		strings.TrimSpace(*speakerGroupInfo.VoiceModelOverride) != "" {
		overrideModel := strings.TrimSpace(*speakerGroupInfo.VoiceModelOverride)
		ttsConfig["model"] = overrideModel
		log.Debugf("Override Qianwen model for speaker %s: %s", speakerResult.SpeakerName, overrideModel)
	}

	// 7. Save the complete TTS configuration (deep copy)
	s.clientState.SpeakerTTSConfig = make(map[string]interface{})
	for k, v := range ttsConfig {
		s.clientState.SpeakerTTSConfig[k] = v
	}
	// Make sure provider is in config
	s.clientState.SpeakerTTSConfig["provider"] = targetTTSConfig.Provider

	log.Infof("✅ Successfully switched TTS configuration for speaker %s - Provider: %s, ConfigID: %s, Voice: %v",
		speakerResult.SpeakerName,
		targetTTSConfig.Provider,
		targetTTSConfig.ConfigID,
		speakerGroupInfo.Voice)

	return nil
}

func (s *ChatSession) hookContext(ctx context.Context) chathooks.Context {
	sessionID := ""
	deviceID := ""
	if s != nil && s.clientState != nil {
		sessionID = s.clientState.SessionID
		deviceID = s.clientState.DeviceID
	}

	return chathooks.Context{
		Ctx:       ctx,
		SessionID: sessionID,
		DeviceID:  deviceID,
	}
}

func (s *ChatSession) emitMetricStage(ctx context.Context, stage chathooks.MetricStage, ts int64, err error) {
	if s == nil {
		return
	}

	hookErr := s.hookHub.EmitMetric(s.hookContext(ctx), chathooks.MetricData{Stage: stage, Ts: ts, Err: err})
	if hookErr != nil {
		log.Warnf("METRIC hook execution failed: stage=%s err=%v", stage, hookErr)
	}
}

func (s *ChatSession) TraceTurnStart(ctx context.Context, ts int64) {
	s.emitMetricStage(ctx, chathooks.MetricTurnStart, ts, nil)
}

func (s *ChatSession) TraceTurnEnd(ctx context.Context, ts int64, err error) {
	s.emitMetricStage(ctx, chathooks.MetricTurnEnd, ts, err)
}

func (s *ChatSession) TraceVoiceSilence(ctx context.Context, ts int64) {
	s.emitMetricStage(ctx, chathooks.MetricVoiceSilence, ts, nil)
}

func (s *ChatSession) TraceAsrFirstText(ctx context.Context, ts int64) {
	s.emitMetricStage(ctx, chathooks.MetricAsrFirstText, ts, nil)
}

func (s *ChatSession) TraceAsrFinalText(ctx context.Context, ts int64) {
	s.emitMetricStage(ctx, chathooks.MetricAsrFinalText, ts, nil)
}

func (s *ChatSession) TraceLlmStart(ctx context.Context, ts int64) {
	s.emitMetricStage(ctx, chathooks.MetricLlmStart, ts, nil)
}

func (s *ChatSession) TraceLlmFirstToken(ctx context.Context, ts int64) {
	s.emitMetricStage(ctx, chathooks.MetricLlmFirstToken, ts, nil)
}

func (s *ChatSession) TraceLlmFirstSentence(ctx context.Context, ts int64) {
	s.emitMetricStage(ctx, chathooks.MetricLlmFirstSentence, ts, nil)
}

func (s *ChatSession) TraceLlmEnd(ctx context.Context, ts int64, err error) {
	s.emitMetricStage(ctx, chathooks.MetricLlmEnd, ts, err)
}

func (s *ChatSession) TraceTtsStart(ctx context.Context, ts int64) {
	s.emitMetricStage(ctx, chathooks.MetricTtsStart, ts, nil)
}

func (s *ChatSession) TraceTtsFirstFrame(ctx context.Context, ts int64) {
	s.emitMetricStage(ctx, chathooks.MetricTtsFirstFrame, ts, nil)
}

func (s *ChatSession) TraceTtsStop(ctx context.Context, ts int64, err error) {
	s.emitMetricStage(ctx, chathooks.MetricTtsStop, ts, err)
}
