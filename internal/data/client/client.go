package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"sync"

	utypes "xiaozhi-esp32-server-golang/internal/domain/config/types"
	"xiaozhi-esp32-server-golang/internal/domain/llm"
	llm_common "xiaozhi-esp32-server-golang/internal/domain/llm/common"
	"xiaozhi-esp32-server-golang/internal/domain/memory"
	"xiaozhi-esp32-server-golang/internal/domain/speaker"
	"xiaozhi-esp32-server-golang/internal/domain/tts"

	. "xiaozhi-esp32-server-golang/internal/data/audio"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"
)

// Dialogue represents the conversation history
type Dialogue struct {
	mu       sync.RWMutex // Read-write lock protecting Messages
	Messages []*schema.Message
}

const (
	ClientStatusInit       = "init"
	ClientStatusListening  = "listening"
	ClientStatusListenStop = "listenStop"
	ClientStatusLLMStart   = "llmStart"
	ClientStatusTTSStart   = "ttsStart"

	ListenPhaseIdle      = "idle"
	ListenPhaseStarting  = "starting"
	ListenPhaseListening = "listening"

	CommandTypeDetect      = "detect"
	CommandTypeListenStart = "listen_start"
	CommandTypeListenStop  = "listen_stop"

	MemoryModeNone  = "none"
	MemoryModeShort = "short"
	MemoryModeLong  = "long"

	SpeakerChatModeOff            = "off"
	SpeakerChatModeIdentifiedOnly = "identified_only"
)

func NormalizeMemoryMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case MemoryModeNone:
		return MemoryModeNone
	case MemoryModeLong:
		return MemoryModeLong
	default:
		return MemoryModeShort
	}
}

func NormalizeSpeakerChatMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case SpeakerChatModeIdentifiedOnly:
		return SpeakerChatModeIdentifiedOnly
	default:
		return SpeakerChatModeOff
	}
}

type SendAudioData func(audioData []byte) error

// ClientState represents the client state
type ClientState struct {
	cmdMu sync.Mutex

	IsActivated bool
	// Conversation history
	Dialogue *Dialogue
	// Abort state
	Abort bool
	// Listen mode
	ListenMode string
	// listen start flow state: idle / starting / listening
	ListenPhase string
	// Device ID
	DeviceID string
	AgentID  string
	// Session ID
	SessionID string

	// Device configuration
	DeviceConfig utypes.UConfig

	Vad
	Asr
	Llm

	// TTS provider
	TTSProvider      tts.TTSProvider        // Default TTS provider
	SpeakerTTSConfig map[string]interface{} // TTS config for voice fingerprint recognition (full config, takes priority)
	// Memory provider
	MemoryProvider memory.MemoryProvider
	MemoryContext  string // Memory context

	// Context control
	Ctx    context.Context
	Cancel context.CancelFunc

	SessionCtx         Ctx // Context for one conversation session
	AfterAsrSessionCtx Ctx // Context for post-ASR flow

	// System prompt
	SystemPrompt string

	InputAudioFormat  AudioFormat // Input audio format
	OutputAudioFormat AudioFormat // Output audio format

	// Audio data buffer for opus input
	OpusAudioBuffer chan []byte

	// Audio data buffer for PCM input
	AsrAudioBuffer *AsrAudioBuffer

	VoiceStatus
	AudioIdle AudioIdleClock

	UdpSendAudioData SendAudioData // Send audio data
	Statistic        Statistic     // Timing statistics
	MqttLastActiveTs int64         // Last active timestamp
	VadLastActiveTs  int64         // VAD last active timestamp; disconnect if >60s and not in TTS

	Status string // State: listening, llmStart, ttsStart

	IsTtsStart        bool // Whether TTS has started
	IsWelcomeSpeaking bool // Whether welcome message has been played
	IsWelcomePlaying  bool // Whether welcome message is currently playing

	LastCmdType string
	LastCmdAt   time.Time

	// Voice fingerprint recognition
	SpeakerProvider speaker.SpeakerProvider // Voice fingerprint provider (initialized in session)

	// Async callback for retrieving voice fingerprint result (set in session)
	OnVoiceSilenceSpeakerCallback func(ctx context.Context)

	// Voice silence event metric callback (set in session)
	OnVoiceSilenceMetricCallback func(ctx context.Context, ts int64)

	// Callback for first ASR text return (set in session)
	OnAsrFirstTextCallback func(text string, isFinal bool)
}

// IsSpeakerEnabled checks whether voice fingerprint recognition is enabled (reads from global config)
func (c *ClientState) IsSpeakerEnabled() bool {
	// Get enable field from global config (viper)
	enabled := viper.GetBool("voice_identify.enable")
	return enabled
}

// HasSpeakerGroups checks whether the device config has voice fingerprint groups
func (c *ClientState) HasSpeakerGroups() bool {
	// Check whether the device config has voice fingerprint group configuration
	return len(c.DeviceConfig.VoiceIdentify) > 0
}

func (c *ClientState) IsRealTime() bool {
	return c.ListenMode == "realtime"
}

func (c *ClientState) GetMemoryMode() string {
	return NormalizeMemoryMode(c.DeviceConfig.MemoryMode)
}

func (c *ClientState) GetSpeakerChatMode() string {
	return NormalizeSpeakerChatMode(c.DeviceConfig.SpeakerChatMode)
}

func (c *ClientState) RequireMatchedSpeakerForChat() bool {
	return c.HasSpeakerGroups() && c.GetSpeakerChatMode() == SpeakerChatModeIdentifiedOnly
}

func (c *ClientState) HasMatchedConfiguredSpeaker(result *speaker.IdentifyResult) bool {
	if result == nil || !result.Identified {
		return false
	}
	_, ok := c.DeviceConfig.VoiceIdentify[result.SpeakerName]
	return ok
}

func (c *ClientState) GetDeviceIDOrAgentID() string {
	if c.AgentID != "" {
		return c.AgentID
	}
	return c.DeviceID
}

// History message methods start
func (c *ClientState) AddMessage(msg *schema.Message) {
	if msg == nil {
		log.Warnf("attempted to add nil message to conversation history")
		return
	}
	c.Dialogue.mu.Lock()
	defer c.Dialogue.mu.Unlock()
	c.Dialogue.Messages = append(c.Dialogue.Messages, msg)
}

func (c *ClientState) GetMessages(count int) []*schema.Message {
	c.Dialogue.mu.RLock()
	defer c.Dialogue.mu.RUnlock()

	// Add bounds check to prevent array out of bounds
	if len(c.Dialogue.Messages) == 0 {
		return []*schema.Message{}
	}

	// Calculate start index, ensuring no out-of-bounds
	startIndex := len(c.Dialogue.Messages) - count
	if startIndex < 0 {
		startIndex = 0
	}

	return AlignToolMessages(c.Dialogue.Messages[startIndex:])
}

/*
func AlignMessage(messages []*schema.Message) []*schema.Message {
	findMsgTypeUser := false
	// To ensure message completeness, iterate and find messages after the first User message
	for i := 0; i < len(messages); i++ {
		msg := messages[i]
		if msg == nil {
			continue
		}
		if !findMsgTypeUser {
			if msg.Role == schema.User {
				return messages[i:]
			}
			continue
		}
	}
	return messages
}
*/
// AlignToolMessages ensures that tool_call_id in role:tool messages corresponds to tool_calls id in role:assistant messages.
// Removes mismatched tool messages and handles reverse mismatch scenarios.
func AlignToolMessages(messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return messages
	}

	// Collect all tool_calls ids from assistant messages
	validToolCallIDs := make(map[string]bool)
	// Collect all tool_call_id from tool messages
	usedToolCallIDs := make(map[string]bool)

	// First pass: collect tool_calls ids from assistant messages and tool_call_id from tool messages
	for _, msg := range messages {
		if msg == nil {
			continue
		}

		if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				if toolCall.ID != "" {
					validToolCallIDs[toolCall.ID] = true
				}
			}
		}

		if msg.Role == schema.Tool && msg.ToolCallID != "" {
			usedToolCallIDs[msg.ToolCallID] = true
		}
	}

	// Filter messages, handling bidirectional mismatches
	var alignedMessages []*schema.Message
	for _, msg := range messages {
		if msg == nil {
			continue
		}

		// If it is a tool message, check whether tool_call_id is valid
		if msg.Role == schema.Tool {
			if msg.ToolCallID != "" && validToolCallIDs[msg.ToolCallID] {
				alignedMessages = append(alignedMessages, msg)
			}
		} else if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			// Handle assistant messages, check for unused tool_calls
			for _, toolCall := range msg.ToolCalls {
				if toolCall.ID != "" {
					if usedToolCallIDs[toolCall.ID] {
						alignedMessages = append(alignedMessages, msg)
					} else {
						continue
					}
				}
			}
		} else {
			// Keep all other message types
			alignedMessages = append(alignedMessages, msg)
		}
	}

	return alignedMessages
}

func (c *ClientState) InitMessages(messages []*schema.Message) error {
	c.Dialogue.mu.Lock()
	defer c.Dialogue.mu.Unlock()
	c.Dialogue.Messages = AlignToolMessages(messages)
	return nil
}

// History message methods end

func (c *ClientState) SetTtsStart(isStart bool) {
	c.IsTtsStart = isStart
}

func (c *ClientState) GetTtsStart() bool {
	return c.IsTtsStart
}

func (c *ClientState) GetMaxIdleDuration() int64 {
	if !viper.IsSet("chat.max_idle_duration") {
		return 30000
	}

	maxIdleDuration := viper.GetInt64("chat.max_idle_duration")
	if maxIdleDuration <= 0 {
		return math.MaxInt64
	}
	return maxIdleDuration
}

func (c *ClientState) UsesAudioIdleClock() bool {
	if c == nil {
		return false
	}
	return c.ListenMode == "auto" || c.IsRealTime()
}

func (c *ClientState) ShouldCountAudioIdleTimeout() bool {
	if c == nil || !c.IsRealTime() {
		return true
	}
	if c.GetTtsStart() {
		return false
	}
	switch c.GetStatus() {
	case ClientStatusLLMStart, ClientStatusTTSStart:
		return false
	default:
		return true
	}
}

func (c *ClientState) StartAudioIdleWindow(now time.Time) {
	if c == nil || !c.UsesAudioIdleClock() {
		return
	}
	c.AudioIdle.Start(now)
	c.SetClientVoiceStop(false)
}

func (c *ClientState) PauseAudioIdleWindow(now time.Time) {
	if c == nil || !c.UsesAudioIdleClock() {
		return
	}
	c.AudioIdle.Pause(now)
}

func (c *ClientState) ResumeAudioIdleWindow(now time.Time) {
	if c == nil || !c.UsesAudioIdleClock() {
		return
	}
	c.AudioIdle.Resume(now)
	c.SetClientVoiceStop(false)
}

func (c *ClientState) ResetAudioIdleWindow() {
	if c == nil {
		return
	}
	c.AudioIdle.Reset()
}

func (c *ClientState) GetAudioIdleElapsed(now time.Time) time.Duration {
	if c == nil {
		return 0
	}
	return c.AudioIdle.Elapsed(now)
}

func (c *ClientState) AudioIdleStarted() bool {
	if c == nil {
		return false
	}
	return c.AudioIdle.Started()
}

func (c *ClientState) AudioIdlePaused() bool {
	if c == nil {
		return false
	}
	return c.AudioIdle.Paused()
}

func (c *ClientState) MarkAudioIdleTimeoutPending() bool {
	if c == nil {
		return false
	}
	return c.AudioIdle.MarkTimeoutPending()
}

func (c *ClientState) ClearAudioIdleTimeoutPending() {
	if c == nil {
		return
	}
	c.AudioIdle.ClearTimeoutPending()
}

func (c *ClientState) AudioIdleTimeoutPending() bool {
	if c == nil {
		return false
	}
	return c.AudioIdle.TimeoutPending()
}

func (c *ClientState) GetPreAsrTextSilenceDuration() int64 {
	if viper.IsSet("chat.pre_asr_text_silence_duration") {
		preTextSilenceDuration := viper.GetInt64("chat.pre_asr_text_silence_duration")
		if preTextSilenceDuration <= 0 {
			return math.MaxInt64
		}
		return preTextSilenceDuration
	}

	base := c.VoiceStatus.SilenceThresholdTime
	if base <= 0 {
		base = 400
	}
	preTextSilenceDuration := base * 4
	if preTextSilenceDuration < 1000 {
		preTextSilenceDuration = 1000
	}
	return preTextSilenceDuration
}

func (c *ClientState) UpdateLastActiveTs() {
	c.MqttLastActiveTs = time.Now().Unix()
}

func (c *ClientState) IsActive() bool {
	diff := time.Now().Unix() - c.MqttLastActiveTs
	return c.MqttLastActiveTs > 0 && diff <= ClientActiveTs
}

func (c *ClientState) SetStatus(status string) {
	c.Status = status
}

func (c *ClientState) GetStatus() string {
	return c.Status
}

func (c *ClientState) SetListenPhase(phase string) {
	c.ListenPhase = phase
}

func (c *ClientState) GetListenPhase() string {
	return c.ListenPhase
}

type Ctx struct {
	sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *Ctx) Reset() {
	c.ResetWithReason("Ctx.Reset")
}

func (c *Ctx) ResetWithReason(reason string) {
	c.Lock()
	defer c.Unlock()
	if c.ctx != nil {
		log.Debugf("Ctx.ResetWithReason: reason=%s", reason)
		c.cancel()
		c.ctx = nil
		c.cancel = nil
	}
}

func (c *Ctx) Get(parentCtx context.Context) context.Context {
	c.Lock()
	defer c.Unlock()
	if c.ctx == nil || c.ctx.Err() != nil {
		if c.ctx != nil {
			c.cancel()
		}
		c.ctx, c.cancel = context.WithCancel(parentCtx)
	}
	return c.ctx
}

func (c *Ctx) Cancel() {
	c.CancelWithReason("Ctx.Cancel")
}

func (c *Ctx) CancelWithReason(reason string) {
	c.Lock()
	defer c.Unlock()
	if c.ctx != nil {
		log.Debugf("Ctx.CancelWithReason: reason=%s", reason)
		c.cancel()
		c.ctx = nil
		c.cancel = nil
	}
}

func (s *ClientState) getLLMProvider() (llm.LLMProvider, error) {
	llmConfig := s.DeviceConfig.Llm
	providerName := llmConfig.Provider
	if providerName == "" {
		providerName = "openai"
	}
	llmProvider, err := llm.GetLLMProvider(providerName, llmConfig.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %v", err)
	}
	return llmProvider, nil
}

func (s *ClientState) InitLlm() error {
	ctx, cancel := context.WithCancel(s.Ctx)

	llmProvider, err := s.getLLMProvider()
	if err != nil {
		log.Errorf("failed to create LLM provider: %v", err)
		return err
	}

	s.Llm = Llm{
		Ctx:         ctx,
		Cancel:      cancel,
		LLMProvider: llmProvider,
	}
	return nil
}

func (s *ClientState) InitAsr() error {
	asrConfig := s.DeviceConfig.Asr

	log.Infof("initializing ASR, asrConfig: %+v", asrConfig)

	// Initialize ASR (no longer directly creating AsrProvider; use resource pool instead)
	ctx, cancel := context.WithCancel(s.Ctx)
	s.Asr = Asr{
		Ctx:             ctx,
		Cancel:          cancel,
		AsrAudioChannel: make(chan []float32, 100),
		AsrEnd:          make(chan bool, 1),
		AsrResult:       bytes.Buffer{},
		AsrType:         asrConfig.Provider,
		ClientState:     s, // Set ClientState reference
	}

	// Set ASR mode
	if mode, ok := asrConfig.Config["mode"].(string); ok {
		s.Asr.Mode = mode
	}

	if rawAutoEnd, ok := asrConfig.Config["auto_end"]; ok {
		if autoEnd, ok := rawAutoEnd.(bool); ok {
			s.Asr.AutoEnd = autoEnd
		}
	}
	return nil
}

func (c *ClientState) Destroy() {
	c.Asr.StopWithReason("ClientState.Destroy")
	c.Vad.Reset()
	c.ResetAudioIdleWindow()
	c.ClearAudioIdleTimeoutPending()

	// Return ASR resources (if any)
	// Note: importing the pool package here would create a circular dependency,
	// so resource return is handled at the call site (ChatSession.Close)

	c.VoiceStatus.Reset()
	c.AsrAudioBuffer.ClearAsrAudioData()

	c.SessionCtx.ResetWithReason("ClientState.Destroy: session_ctx")
	c.AfterAsrSessionCtx.ResetWithReason("ClientState.Destroy: after_asr_ctx")

	c.Statistic.Reset()
	c.SetStatus(ClientStatusInit)
	c.SetListenPhase(ListenPhaseIdle)
	c.SetTtsStart(false)
}

type CommandHistorySnapshot struct {
	LastCmdType string
	LastCmdAt   time.Time
}

func (s CommandHistorySnapshot) DebugString(now time.Time) string {
	formatAt := func(at time.Time) string {
		if at.IsZero() {
			return "zero"
		}
		return at.Format(time.RFC3339Nano)
	}
	formatAge := func(at time.Time) string {
		if at.IsZero() {
			return "n/a"
		}
		return now.Sub(at).Truncate(time.Millisecond).String()
	}

	return fmt.Sprintf(
		"lastCmd=%q lastCmdAt=%s lastCmdAge=%s",
		s.LastCmdType,
		formatAt(s.LastCmdAt),
		formatAge(s.LastCmdAt),
	)
}

func (c *ClientState) RecordCommandArrival(cmdType string, at time.Time) {
	c.cmdMu.Lock()
	c.LastCmdType = cmdType
	c.LastCmdAt = at
	c.cmdMu.Unlock()
}

func (c *ClientState) GetCommandHistorySnapshot() CommandHistorySnapshot {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()
	return CommandHistorySnapshot{
		LastCmdType: c.LastCmdType,
		LastCmdAt:   c.LastCmdAt,
	}
}

func (state *ClientState) OnManualStop() {
	state.ClearAudioIdleTimeoutPending()
	state.OnVoiceSilence()
}

func (state *ClientState) OnVoiceSilence() {
	silenceTs := time.Now().UnixMilli()
	log.Debugf("OnVoiceSilence, voiceDuration: %d, voiceDurationInSession: %d", state.Vad.GetVoiceDuration(), state.Vad.GetVoiceDurationInSession())
	if state.MarkVoiceSilenceAt(silenceTs) && state.OnVoiceSilenceMetricCallback != nil {
		state.OnVoiceSilenceMetricCallback(state.Ctx, silenceTs)
	}
	state.Asr.ResetReceivedText()
	state.SetClientVoiceStop(true) // Set speech-stop flag; audio received after this will not enter VAD
	// Client stopped speaking
	state.Asr.StopWithReason("ClientState.OnVoiceSilence") // Stop ASR and get result, then proceed to LLM
	// Release VAD
	state.Vad.Reset() // Release VAD instance

	state.SetStatus(ClientStatusListenStop)
	state.SetListenPhase(ListenPhaseIdle)

	// If async voice fingerprint callback is set, call it
	if state.OnVoiceSilenceSpeakerCallback != nil {
		state.OnVoiceSilenceSpeakerCallback(state.Ctx)
	}
}

type Llm struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	// LLM provider
	LLMProvider llm.LLMProvider
	// Channel for receiving ASR-to-text output
	LLmRecvChannel chan llm_common.LLMResponseStruct
}

type SpeakReadyUDPConfig struct {
	Ready         bool `json:"ready"`
	ReuseExisting bool `json:"reuse_existing,omitempty"`
}

// ClientMessage represents a client message
type ClientMessage struct {
	Type           string               `json:"type"`
	DeviceID       string               `json:"device_id,omitempty"`
	SessionID      string               `json:"session_id,omitempty"`
	Text           string               `json:"text,omitempty"`
	Mode           string               `json:"mode,omitempty"`
	State          string               `json:"state,omitempty"`
	Token          string               `json:"token,omitempty"`
	DeviceMac      string               `json:"device_mac,omitempty"`
	Version        int                  `json:"version,omitempty"`
	Transport      string               `json:"transport,omitempty"`
	Features       map[string]bool      `json:"features,omitempty"`
	AudioParams    *AudioFormat         `json:"audio_params,omitempty"`
	SpeakUDPConfig *SpeakReadyUDPConfig `json:"udp_config,omitempty"`
	PayLoad        json.RawMessage      `json:"payload,omitempty"`
}
