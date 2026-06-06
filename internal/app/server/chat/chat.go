package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"

	"xiaozhi-esp32-server-golang/constants"
	"xiaozhi-esp32-server-golang/internal/app/server/auth"
	"xiaozhi-esp32-server-golang/internal/app/server/chat/plugins"
	types_conn "xiaozhi-esp32-server-golang/internal/app/server/types"
	types_audio "xiaozhi-esp32-server-golang/internal/data/audio"
	. "xiaozhi-esp32-server-golang/internal/data/client"
	. "xiaozhi-esp32-server-golang/internal/data/msg"
	chathooks "xiaozhi-esp32-server-golang/internal/domain/chat/hooks"
	"xiaozhi-esp32-server-golang/internal/domain/chat/streamtransform"
	userconfig "xiaozhi-esp32-server-golang/internal/domain/config"
	"xiaozhi-esp32-server-golang/internal/domain/mcp"
	"xiaozhi-esp32-server-golang/internal/domain/openclaw"
	pkghooks "xiaozhi-esp32-server-golang/internal/pkg/hooks"
	log "xiaozhi-esp32-server-golang/logger"
)

type ChatManager struct {
	DeviceID  string
	transport types_conn.IConn

	clientState       *ClientState
	serverTransport   *ServerTransport
	mcpTransport      *McpTransport
	hookHub           *chathooks.Hub
	transformRegistry *streamtransform.Registry

	sessionMu sync.RWMutex
	session   *ChatSession

	startingSession     *ChatSession
	startingSessionDone chan struct{}

	ctx    context.Context
	cancel context.CancelFunc

	helloMu      sync.Mutex
	helloInited  bool
	mcpInitState chatMcpInitState

	speakRequestMu      sync.Mutex
	pendingSpeakRequest *pendingSpeakRequest
	lastSpeakPathWarmAt atomic.Int64
	speakReadyTimeout   time.Duration

	// Close protection to prevent multiple closes
	closeOnce      sync.Once
	managerClosing atomic.Bool
	needFreshHello bool

	mqttRebootstrapPending atomic.Bool

	retainedSessionCleanupMu     sync.Mutex
	retainedSessionCleanupTimer  *time.Timer
	retainedSessionCleanupTarget *ChatSession
	retainedSessionIdleTimeout   time.Duration
}

type pendingSpeakRequest struct {
	sessionID string
	done      chan struct{}
	timer     *time.Timer

	once sync.Once
	mu   sync.Mutex
	err  error
}

func (p *pendingSpeakRequest) resolve(err error) {
	if p == nil {
		return
	}
	p.once.Do(func() {
		if p.timer != nil {
			p.timer.Stop()
		}
		p.mu.Lock()
		p.err = err
		p.mu.Unlock()
		close(p.done)
	})
}

func (p *pendingSpeakRequest) Err() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

type chatMcpInitState uint8

const (
	chatMcpInitStateIdle chatMcpInitState = iota
	chatMcpInitStateInFlight
	chatMcpInitStateReady
)

const (
	defaultSpeakRequestReuseWindow = 60 * time.Second
	defaultSpeakReadyTimeout       = 5 * time.Second
	defaultRetainedSessionIdleTTL  = 10 * time.Minute
)

type brokerOnlineAwareTransport interface {
	IsBrokerOnline() bool
}

type ChatManagerOption func(*ChatManager)

var (
	chatHookAsyncExecutorOnce sync.Once
	chatHookAsyncExecutor     *pkghooks.AsyncExecutor
)

func sharedChatHookAsyncExecutor() *pkghooks.AsyncExecutor {
	chatHookAsyncExecutorOnce.Do(func() {
		asyncCfg := pkghooks.AsyncConfig{
			QueueSize:    viper.GetInt("chat_hooks.async.queue_size"),
			WorkerCount:  viper.GetInt("chat_hooks.async.worker_count"),
			DropWhenFull: viper.GetBool("chat_hooks.async.drop_when_full"),
			Timeout:      time.Duration(viper.GetInt("chat_hooks.async.timeout_ms")) * time.Millisecond,
		}
		chatHookAsyncExecutor = pkghooks.NewAsyncExecutor(context.Background(), asyncCfg)
		log.Infof("Initialize global shared chat hook observer executor: queue_size=%d worker_count=%d drop_when_full=%v timeout=%s", asyncCfg.QueueSize, asyncCfg.WorkerCount, asyncCfg.DropWhenFull, asyncCfg.Timeout)
	})
	return chatHookAsyncExecutor
}

func newChatHookHub(parent context.Context) *chathooks.Hub {
	asyncCfg := pkghooks.AsyncConfig{
		QueueSize:    viper.GetInt("chat_hooks.async.queue_size"),
		WorkerCount:  viper.GetInt("chat_hooks.async.worker_count"),
		DropWhenFull: viper.GetBool("chat_hooks.async.drop_when_full"),
		Timeout:      time.Duration(viper.GetInt("chat_hooks.async.timeout_ms")) * time.Millisecond,
	}
	hub := chathooks.NewHub(parent, pkghooks.WithAsyncConfig(asyncCfg), pkghooks.WithAsyncExecutor(sharedChatHookAsyncExecutor()))
	stats := hub.Stats()
	log.Infof("Initialize chat hook hub: queue_size=%d worker_count=%d drop_when_full=%v timeout=%s dropped_async=%d", asyncCfg.QueueSize, asyncCfg.WorkerCount, asyncCfg.DropWhenFull, asyncCfg.Timeout, stats.DroppedAsync)
	return hub
}

func chatHookBuiltinOverrides() map[string]chathooks.BuiltinPluginConfig {
	overrides := map[string]chathooks.BuiltinPluginConfig{}
	for _, reg := range chathooks.BuiltinRegistrations() {
		path := "chat_hooks.plugins." + reg.Meta.Name
		cfg := chathooks.BuiltinPluginConfig{}
		if viper.IsSet(path + ".enabled") {
			enabled := viper.GetBool(path + ".enabled")
			cfg.Enabled = &enabled
		}
		if viper.IsSet(path + ".priority") {
			cfg.Priority = viper.GetInt(path + ".priority")
		}
		overrides[reg.Meta.Name] = cfg
	}
	return overrides
}

func NewChatManager(deviceID string, transport types_conn.IConn, options ...ChatManagerOption) (*ChatManager, error) {
	cm := &ChatManager{
		DeviceID:          deviceID,
		transport:         transport,
		speakReadyTimeout: defaultSpeakReadyTimeout,
	}

	for _, option := range options {
		option(cm)
	}

	ctx := context.WithValue(context.Background(), "chat_session_operator", ChatSessionOperator(cm))
	ctx = withTTSTurnEndPolicyHandler(ctx, cm)
	cm.ctx, cm.cancel = context.WithCancel(ctx)

	clientState, err := GenClientState(cm.ctx, cm.DeviceID)
	if err != nil {
		log.Errorf("Failed to initialize client state: %v", err)
		_ = cm.transport.Close()
		return nil, err
	}
	cm.clientState = clientState

	cm.serverTransport = NewServerTransport(cm.transport, clientState)
	cm.mcpTransport = &McpTransport{
		Client:          clientState,
		ServerTransport: cm.serverTransport,
	}

	cm.transport.OnClose(cm.OnClose)

	cm.hookHub = newChatHookHub(cm.ctx)
	if !viper.IsSet("chat_hooks.enabled") || viper.GetBool("chat_hooks.enabled") {
		if err := chathooks.RegisterBuiltinPlugins(cm.hookHub, chatHookBuiltinOverrides()); err != nil {
			log.Errorf("Failed to register chat hook builtin plugins: %v", err)
			_ = cm.transport.Close()
			return nil, err
		}
		log.Infof("Chat hook plugins loaded: %+v", cm.hookHub.PluginMetas())
	}

	cm.transformRegistry = streamtransform.NewRegistry()
	plugins.Init(cm.transformRegistry)

	return cm, nil
}

func GenClientState(pctx context.Context, deviceID string) (*ClientState, error) {
	configProvider, err := userconfig.GetProvider(viper.GetString("config_provider.type"))
	if err != nil {
		log.Errorf("Failed to get user configuration provider: %+v", err)
		return nil, err
	}
	deviceConfig, err := configProvider.GetUserConfig(pctx, deviceID)
	if err != nil {
		log.Errorf("Failed to get configuration for device %s: %+v", deviceID, err)
		return nil, err
	}
	deviceConfig.MemoryMode = NormalizeMemoryMode(deviceConfig.MemoryMode)
	deviceConfig.SpeakerChatMode = NormalizeSpeakerChatMode(deviceConfig.SpeakerChatMode)

	ctx, cancel := context.WithCancel(pctx)

	maxSilenceDuration := viper.GetInt64("chat.chat_max_silence_duration")
	if !viper.IsSet("chat.chat_max_silence_duration") {
		maxSilenceDuration = 400
	}

	isDeviceActivated, err := configProvider.IsDeviceActivated(ctx, deviceID, "")
	if err != nil {
		log.Errorf("Failed to check device activation status: %v", err)
	}

	clientState := &ClientState{
		IsActivated:       isDeviceActivated,
		Dialogue:          &Dialogue{},
		Abort:             false,
		ListenMode:        "auto",
		ListenPhase:       ListenPhaseIdle,
		DeviceID:          deviceID,
		AgentID:           deviceConfig.AgentId,
		Ctx:               ctx,
		Cancel:            cancel,
		SystemPrompt:      deviceConfig.SystemPrompt,
		DeviceConfig:      deviceConfig,
		OutputAudioFormat: types_audio.AudioFormat{},
		OpusAudioBuffer:   make(chan []byte, 100),
		AsrAudioBuffer: &AsrAudioBuffer{
			PcmData:          make([]float32, 0),
			AudioBufferMutex: sync.RWMutex{},
		},
		VoiceStatus: VoiceStatus{
			HaveVoice:            false,
			HaveVoiceLastTime:    0,
			VoiceStop:            false,
			SilenceThresholdTime: maxSilenceDuration,
		},
		SessionCtx: Ctx{},
	}
	applyOutputAudioFormatForTTS(clientState)

	return clientState, nil
}

func applyOutputAudioFormatForTTS(clientState *ClientState) {
	clientState.OutputAudioFormat = types_audio.AudioFormat{
		SampleRate:    types_audio.SampleRate,
		Channels:      types_audio.Channels,
		FrameDuration: types_audio.FrameDuration,
		Format:        types_audio.Format,
	}
	ttsType := clientState.DeviceConfig.Tts.Provider
	if ttsType == constants.TtsTypeXiaozhi {
		clientState.OutputAudioFormat.SampleRate = 24000
		clientState.OutputAudioFormat.FrameDuration = 20
	}
}

func (c *ChatManager) ReloadDeviceConfig(ctx context.Context) error {
	configProvider, err := userconfig.GetProvider(viper.GetString("config_provider.type"))
	if err != nil {
		return fmt.Errorf("Failed to get configuration provider: %w", err)
	}

	deviceConfig, err := configProvider.GetUserConfig(ctx, c.DeviceID)
	if err != nil {
		return fmt.Errorf("Failed to obtain device configuration: %w", err)
	}
	deviceConfig.MemoryMode = NormalizeMemoryMode(deviceConfig.MemoryMode)
	deviceConfig.SpeakerChatMode = NormalizeSpeakerChatMode(deviceConfig.SpeakerChatMode)

	oldAgentID := c.clientState.AgentID
	c.clientState.AgentID = deviceConfig.AgentId
	c.clientState.DeviceConfig = deviceConfig
	c.clientState.SystemPrompt = deviceConfig.SystemPrompt
	c.clientState.SpeakerTTSConfig = nil
	openclaw.GetManager().ExitMode(oldAgentID, c.DeviceID)
	openclaw.GetManager().ExitMode(c.clientState.AgentID, c.DeviceID)
	applyOutputAudioFormatForTTS(c.clientState)
	log.Infof("Device %s configuration has been refreshed, current agent=%s", c.DeviceID, deviceConfig.AgentId)
	return nil
}

func (c *ChatManager) Start() error {
	go c.cmdMessageLoop(c.ctx)
	go c.audioMessageLoop(c.ctx)

	<-c.ctx.Done()
	return nil
}

func (c *ChatManager) handleLoopExit(loopName string, ctx context.Context) {
	if r := recover(); r != nil {
		log.Errorf("Device %s %s loop panic: %v\n%s", c.DeviceID, loopName, r, string(debug.Stack()))
	}
	if ctx == nil || ctx.Err() != nil {
		return
	}
	if c.serverTransport != nil && c.serverTransport.IsClosed() {
		return
	}
	log.Warnf("Device %s %s loop exited abnormally, close ChatManager", c.DeviceID, loopName)
	if err := c.Close(); err != nil {
		log.Warnf("Failed to close ChatManager after device %s %s loop exited: %v", c.DeviceID, loopName, err)
	}
}

func (c *ChatManager) cmdMessageLoop(ctx context.Context) {
	defer c.handleLoopExit("cmd", ctx)

	recvFailCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Infof("Device %s recvCmd context cancel", c.DeviceID)
			return
		default:
		}

		if recvFailCount > 3 {
			log.Errorf("Device %s recv cmd timeout exceeds threshold", c.DeviceID)
			return
		}

		message, err := c.serverTransport.RecvCmd(ctx, 120)
		if err != nil {
			log.Errorf("recv cmd error: %v", err)
			recvFailCount++
			continue
		}
		if message == nil {
			continue
		}

		recvFailCount = 0
		log.Infof("Text message received: %s", string(message))
		if err := c.handleTextMessage(message); err != nil {
			log.Errorf("Failed to process text message: %v, message content: %s", err, string(message))
		}
	}
}

func (c *ChatManager) audioMessageLoop(ctx context.Context) {
	defer c.handleLoopExit("audio", ctx)

	for {
		select {
		case <-ctx.Done():
			log.Debugf("Device %s recvAudio context cancel", c.DeviceID)
			return
		default:
		}

		message, err := c.serverTransport.RecvAudio(ctx, 600)
		if err != nil {
			log.Errorf("recv audio error: %v", err)
			return
		}
		if message == nil {
			continue
		}

		session := c.GetSession()
		if session == nil {
			log.Debugf("Device %s currently has no active ChatSession, discarding audio data", c.DeviceID)
			continue
		}
		if c.hasPendingSpeakRequest() {
			log.Debugf("Device %s currently has a pending speak_request, discarding audio data", c.DeviceID)
			continue
		}

		log.Debugf("Audio data received, size: %d bytes", len(message))
		isAuth := viper.GetBool("auth.enable")
		if isAuth && !c.clientState.IsActivated {
			log.Debugf("Device %s is not activated, skipping audio data", c.clientState.DeviceID)
			continue
		}
		if c.clientState.GetClientVoiceStop() {
			log.Debug("Client stops talking, skips audio data")
			continue
		}

		if ok := session.HandleAudioMessage(message); !ok {
			log.Warnf("Audio buffer full, discarding audio data")
		}
	}
}

func (c *ChatManager) handleTextMessage(message []byte) error {
	var clientMsg ClientMessage
	if err := json.Unmarshal(message, &clientMsg); err != nil {
		log.Errorf("Failed to parse message: %v", err)
		return fmt.Errorf("Failed to parse message: %v", err)
	}

	if clientMsg.Type != MessageTypeGoodBye {
		c.cancelRetainedSessionCleanup("received device activity message")
	}

	switch clientMsg.Type {
	case MessageTypeHello:
		return c.HandleHelloMessage(&clientMsg)
	case MessageTypeSpeakReady:
		return c.HandleSpeakReadyMessage(&clientMsg)
	case MessageTypeListen:
		return c.HandleListenMessage(&clientMsg)
	case MessageTypeAbort:
		return c.HandleAbortMessage(&clientMsg)
	case MessageTypeIot:
		return c.HandleIoTMessage(&clientMsg)
	case MessageTypeMcp:
		return c.HandleMcpMessage(&clientMsg)
	case MessageTypeGoodBye:
		return c.HandleGoodByeMessage(&clientMsg)
	default:
		return fmt.Errorf("Unknown message type: %s", clientMsg.Type)
	}
}

func (c *ChatManager) HandleHelloMessage(msg *ClientMessage) error {
	if msg.AudioParams == nil {
		return fmt.Errorf("hello message is missing audio_params")
	}
	transportType := strings.TrimSpace(msg.Transport)
	if transportType == "" && c.serverTransport != nil {
		transportType = c.serverTransport.GetTransportType()
	}
	switch transportType {
	case types_conn.TransportTypeWebsocket, types_conn.TransportTypeMqttUdp:
	default:
		return fmt.Errorf("Unsupported transfer type: %s", transportType)
	}

	c.helloMu.Lock()
	defer c.helloMu.Unlock()

	clientState := c.clientState
	clientState.InputAudioFormat = *msg.AudioParams
	isFirstHello := !c.helloInited
	requiresFreshHello := c.requiresFreshHello()
	isDuplicateMqttHello := !isFirstHello && !requiresFreshHello &&
		c.serverTransport != nil && c.serverTransport.GetTransportType() == types_conn.TransportTypeMqttUdp
	if c.helloInited {
		prevAgentID := clientState.AgentID
		if err := c.refreshDeviceConfigOnHello(); err != nil {
			log.Warnf("Device %s duplicate hello failed to refresh configuration, downgrade continues: %v", clientState.DeviceID, err)
		}
		c.resetOpenClawModeOnHello(prevAgentID, clientState.AgentID)
	} else {
		c.resetOpenClawModeOnHello(clientState.AgentID)
	}

	if isFirstHello || requiresFreshHello {
		preferredSessionID := ""
		if isFirstHello {
			preferredSessionID = strings.TrimSpace(clientState.SessionID)
		}
		session, err := auth.A().EnsureSession(msg.DeviceID, preferredSessionID)
		if err != nil {
			return fmt.Errorf("Failed to create session: %v", err)
		}
		clientState.SessionID = session.ID
		c.helloInited = true
	}
	if isDuplicateMqttHello {
		log.Infof("Device %s received duplicate_hello and performed hello renegotiation", clientState.DeviceID)
		c.markMqttConversationStateStale("duplicate_hello")
	}

	chatSession, err := c.ensureSessionForHello()
	if err != nil {
		if isFirstHello || requiresFreshHello {
			c.setNeedFreshHello(true)
		}
		return err
	}
	if err := c.sendHelloResponse(msg); err != nil {
		if isFirstHello || requiresFreshHello {
			c.setNeedFreshHello(true)
			if chatSession != nil {
				chatSession.CloseWithReason(chatSessionCloseReasonFatalError)
			}
		}
		return err
	}
	c.refreshSpeakPathWarmFromTransport()
	c.scheduleMcpInitOnHelloLocked(msg)
	if !isFirstHello && !requiresFreshHello {
		log.Infof("Device %s duplicate_hello processing completed, hello renegotiation refreshed", clientState.DeviceID)
	}
	return nil
}

func (c *ChatManager) scheduleMcpInitOnHelloLocked(msg *ClientMessage) {
	if !c.hasMcpFeature(msg) {
		return
	}
	c.scheduleMcpInitLocked()
}

func (c *ChatManager) scheduleMcpInitLocked() {
	if c.mcpTransport == nil {
		return
	}
	if c.mcpInitState == chatMcpInitStateInFlight {
		return
	}
	if !shouldScheduleDeviceMcpRuntimeInit(c.clientState.DeviceID, c.mcpTransport) {
		return
	}
	if c.mcpInitState == chatMcpInitStateReady {
		log.Warnf(
			"Device %s MCP status drift: ChatManager is ready, but transport=%s needs to be reinitialized",
			c.DeviceID,
			strings.TrimSpace(c.mcpTransport.GetMcpTransportType()),
		)
	}

	c.mcpInitState = chatMcpInitStateInFlight
	deviceID := c.clientState.DeviceID
	transportType := strings.TrimSpace(c.mcpTransport.GetMcpTransportType())
	go func() {
		err := initMcp(deviceID, c.mcpTransport)
		c.finishMcpInit(transportType, err)
	}()
}

func (c *ChatManager) finishMcpInit(transportType string, err error) {
	c.helloMu.Lock()
	defer c.helloMu.Unlock()

	if c.ctx.Err() != nil || c.managerClosing.Load() {
		return
	}
	if c.mcpTransport == nil {
		c.mcpInitState = chatMcpInitStateIdle
		return
	}
	currentTransportType := strings.TrimSpace(c.mcpTransport.GetMcpTransportType())
	if currentTransportType != strings.TrimSpace(transportType) {
		return
	}

	if err != nil {
		c.mcpInitState = chatMcpInitStateIdle
		log.Warnf("Device %s MCP initialization failed, waiting for subsequent hello retry: %v", c.DeviceID, err)
		return
	}

	c.mcpInitState = chatMcpInitStateReady
}

func (c *ChatManager) hasMcpFeature(msg *ClientMessage) bool {
	if msg == nil || msg.Features == nil {
		return false
	}
	isMcp, ok := msg.Features["mcp"]
	return ok && isMcp
}

func (c *ChatManager) sendHelloResponse(msg *ClientMessage) error {
	transportType := strings.TrimSpace(msg.Transport)
	if transportType == "" {
		transportType = c.serverTransport.GetTransportType()
	}

	switch transportType {
	case types_conn.TransportTypeWebsocket:
		return c.serverTransport.SendHello(types_conn.TransportTypeWebsocket, &c.clientState.OutputAudioFormat, nil)
	case types_conn.TransportTypeMqttUdp:
		udpConfig, err := c.buildMqttHelloUdpConfig()
		if err != nil {
			return err
		}
		return c.serverTransport.SendHello(types_conn.TransportTypeMqttUdp, &c.clientState.OutputAudioFormat, udpConfig)
	default:
		return fmt.Errorf("Unsupported transfer type: %s", transportType)
	}
}

func (c *ChatManager) buildMqttHelloUdpConfig() (*UdpConfig, error) {
	udpExternalHost := viper.GetString("udp.external_host")
	udpExternalPort := viper.GetInt("udp.external_port")

	aesKey, err := c.serverTransport.GetData("aes_key")
	if err != nil {
		return nil, fmt.Errorf("Failed to obtain aes_key: %v", err)
	}
	fullNonce, err := c.serverTransport.GetData("full_nonce")
	if err != nil {
		return nil, fmt.Errorf("Failed to get full_nonce: %v", err)
	}

	strAesKey, ok := aesKey.(string)
	if !ok {
		return nil, fmt.Errorf("aes_key is not a string")
	}
	strFullNonce, ok := fullNonce.(string)
	if !ok {
		return nil, fmt.Errorf("full_nonce is not a string")
	}

	return &UdpConfig{
		Server: udpExternalHost,
		Port:   udpExternalPort,
		Key:    strAesKey,
		Nonce:  strFullNonce,
	}, nil
}

func (c *ChatManager) HandleListenMessage(msg *ClientMessage) error {
	if c.requiresHelloBootstrapForSession() {
		log.Infof(
			"Device %s current session is closed or has not completed hello, ignore listen %s, wait for new hello",
			c.DeviceID,
			strings.TrimSpace(msg.State),
		)
		return nil
	}
	session, err := c.ensureSession()
	if err != nil {
		return err
	}
	c.clearMqttRebootstrapPending("listen_message")
	return session.HandleListenMessage(msg)
}

func (c *ChatManager) HandleAbortMessage(msg *ClientMessage) error {
	session := c.GetSession()
	if session == nil {
		log.Debugf("Device %s currently has no active ChatSession, abort ignored", c.DeviceID)
		return nil
	}
	return session.HandleAbortMessage(msg)
}

func (c *ChatManager) HandleIoTMessage(msg *ClientMessage) error {
	if err := c.serverTransport.SendIot(msg); err != nil {
		return fmt.Errorf("Failed to send response: %v", err)
	}
	log.Infof("Device %s IoT command: %s", msg.DeviceID, msg.Text)
	return nil
}

func (c *ChatManager) HandleMcpMessage(msg *ClientMessage) error {
	return mcp.HandleDeviceIotMcpMessage(c.clientState.DeviceID, c.mcpTransport.GetMcpTransportType(), msg.PayLoad)
}

func (c *ChatManager) HandleGoodByeMessage(msg *ClientMessage) error {
	session := c.GetSession()
	if session != nil {
		log.Infof("Device %s received device-side goodbye, retained ChatSession and reset to silent state", c.DeviceID)
		session.ResetToSilentState()
		c.scheduleRetainedSessionCleanup(session, "peer_goodbye")
	} else {
		log.Infof("Device %s received device-side goodbye, but there is currently no active ChatSession, only clearing the audio link", c.DeviceID)
	}

	c.resetSpeakPathAfterGoodbye()
	return c.transport.CloseAudioChannel()
}

func (c *ChatManager) HandleSpeakReadyMessage(msg *ClientMessage) error {
	if msg == nil {
		return nil
	}
	if c.serverTransport == nil || c.serverTransport.GetTransportType() != types_conn.TransportTypeMqttUdp {
		return nil
	}
	if msg.State != "" && msg.State != MessageStateReady {
		log.Debugf("Device %s speak_ready status is not ready, ignore: %+v", c.DeviceID, msg)
		return nil
	}
	if msg.SpeakUDPConfig != nil && !msg.SpeakUDPConfig.Ready {
		log.Warnf("Device %s speak_ready udp_config.ready=false, ignored", c.DeviceID)
		return nil
	}

	c.speakRequestMu.Lock()
	pending := c.pendingSpeakRequest
	c.speakRequestMu.Unlock()
	if pending == nil {
		log.Debugf("Device %s received speak_ready with no pending requests, ignored", c.DeviceID)
		return nil
	}
	if pending.sessionID != "" && strings.TrimSpace(msg.SessionID) != pending.sessionID {
		log.Warnf("Device %s speak_ready session_id does not match: got=%s want=%s", c.DeviceID, msg.SessionID, pending.sessionID)
		return nil
	}

	c.markSpeakPathWarm(time.Now())
	c.clearMqttRebootstrapPending("speak_ready")
	c.finishPendingSpeakRequest(pending, nil)

	reuseExisting := false
	if msg.SpeakUDPConfig != nil {
		reuseExisting = msg.SpeakUDPConfig.ReuseExisting
	}
	log.Infof("Device %s speak_ready is ready, reuse_existing=%v", c.DeviceID, reuseExisting)
	return nil
}

func (c *ChatManager) ensureSession() (*ChatSession, error) {
	return c.ensureSessionInternal(false)
}

func (c *ChatManager) ensureSessionForHello() (*ChatSession, error) {
	return c.ensureSessionInternal(true)
}

func (c *ChatManager) ensureSessionInternal(allowFreshHello bool) (*ChatSession, error) {
	for {
		c.sessionMu.Lock()
		if c.session != nil {
			session := c.session
			c.sessionMu.Unlock()
			if session.IsClosing() {
				return nil, fmt.Errorf("ChatSession is closing; try again later")
			}
			return session, nil
		}
		if c.startingSession != nil {
			waitCh := c.startingSessionDone
			c.sessionMu.Unlock()
			if waitCh == nil {
				return nil, fmt.Errorf("ChatSession is starting; try again later")
			}
			<-waitCh
			continue
		}
		if !c.helloInited {
			c.sessionMu.Unlock()
			return nil, fmt.Errorf("hello is not initialized; cannot create ChatSession")
		}
		if c.needFreshHello && !allowFreshHello {
			c.sessionMu.Unlock()
			return nil, fmt.Errorf("ChatSession has exited; resend hello first")
		}

		session := NewChatSession(
			c.clientState,
			c.serverTransport,
			c.hookHub,
			c.transformRegistry,
			WithChatSessionCloseHandler(c.handleSessionClosed),
		)
		c.startingSession = session
		c.startingSessionDone = make(chan struct{})
		c.sessionMu.Unlock()

		err := session.Start(c.ctx)
		if err != nil {
			session.CloseWithReason(chatSessionCloseReasonFatalError)
		}
		c.finishSessionStart(session, allowFreshHello, err)
		if err != nil {
			return nil, err
		}
		if session.IsClosing() {
			return nil, fmt.Errorf("ChatSession is closing; try again later")
		}
		return session, nil
	}
}

func (c *ChatManager) requiresFreshHello() bool {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	return c.needFreshHello
}

func (c *ChatManager) setNeedFreshHello(required bool) {
	c.sessionMu.Lock()
	c.needFreshHello = required
	c.sessionMu.Unlock()
}

func (c *ChatManager) needsMqttRebootstrap() bool {
	if c == nil {
		return false
	}
	return c.mqttRebootstrapPending.Load()
}

func (c *ChatManager) clearMqttRebootstrapPending(reason string) {
	if c == nil {
		return
	}
	if c.mqttRebootstrapPending.Swap(false) {
		log.Debugf("Device %s clears MQTT session rebuild flag: reason=%s", c.DeviceID, reason)
	}
}

func (c *ChatManager) hasPendingSpeakRequest() bool {
	if c == nil {
		return false
	}
	c.speakRequestMu.Lock()
	defer c.speakRequestMu.Unlock()
	return c.pendingSpeakRequest != nil
}

func (c *ChatManager) resetSpeakPathAfterMqttRebootstrap(reason string) {
	if c == nil {
		return
	}
	c.lastSpeakPathWarmAt.Store(0)

	c.speakRequestMu.Lock()
	pending := c.pendingSpeakRequest
	c.speakRequestMu.Unlock()
	if pending != nil {
		c.finishPendingSpeakRequest(pending, fmt.Errorf("MQTT link reestablishment leads to active reporting that the link has been reset: %s", reason))
	}
}

func (c *ChatManager) markMqttConversationStateStale(reason string) {
	if c == nil || c.serverTransport == nil || c.serverTransport.GetTransportType() != types_conn.TransportTypeMqttUdp {
		return
	}
	if c.managerClosing.Load() {
		return
	}
	if reason == "duplicate_hello" && c.hasPendingSpeakRequest() {
		log.Infof("Device %s received duplicate_hello, detected the pending speak_request, processed the handshake renegotiation according to the active broadcast, and skipped session reconstruction normalization.", c.DeviceID)
		return
	}

	alreadyPending := c.mqttRebootstrapPending.Swap(true)
	if alreadyPending {
		log.Debugf("Device %s MQTT session rebuild flag already exists, skip duplicate normalization: reason=%s", c.DeviceID, reason)
		return
	}

	session := c.GetSession()
	if session != nil && !session.IsClosing() {
		log.Infof("Device %s MQTT link reestablishment, reset current session status: reason=%s", c.DeviceID, reason)
		session.ResetToSilentState()
		c.scheduleRetainedSessionCleanup(session, "mqtt_transport_rebootstrap")
	} else if c.clientState != nil {
		log.Infof("Device %s MQTT link reestablishment, clearing residual client status: reason=%s", c.DeviceID, reason)
		c.clientState.Destroy()
		c.clientState.Abort = false
		c.clientState.IsWelcomeSpeaking = false
		c.clientState.IsWelcomePlaying = false
	}

	c.resetSpeakPathAfterMqttRebootstrap(reason)
}

func (c *ChatManager) resetMcpRuntimeOnMqttTransportReady() {
	if c == nil {
		return
	}

	c.helloMu.Lock()
	defer c.helloMu.Unlock()

	c.mcpInitState = chatMcpInitStateIdle
	if c.clientState == nil || c.mcpTransport == nil {
		return
	}

	log.Infof("Device %s received MQTT online broadcast, destroyed the existing IoT MCP runtime, and prepared to re-initialize", c.DeviceID)
	closeDeviceMcpRuntime(c.clientState.DeviceID, c.mcpTransport)
}

func (c *ChatManager) HandleMqttTransportReady() {
	c.markMqttConversationStateStale("transport_ready")
	c.resetMcpRuntimeOnMqttTransportReady()
}

func (c *ChatManager) resetSpeakPathAfterGoodbye() {
	c.resetSpeakPathAfterSessionReset(fmt.Errorf("Goodbye on the device side causes active reporting that the link has been reset."))
}

func (c *ChatManager) resetSpeakPathAfterServerSessionClose(reason string) {
	c.resetSpeakPathAfterSessionReset(fmt.Errorf("The server closed the session and actively reported that the link has been reset: %s", reason))
}

func (c *ChatManager) resetSpeakPathAfterSessionReset(err error) {
	if c == nil {
		return
	}
	c.lastSpeakPathWarmAt.Store(0)

	c.speakRequestMu.Lock()
	pending := c.pendingSpeakRequest
	c.speakRequestMu.Unlock()
	if pending != nil {
		c.finishPendingSpeakRequest(pending, err)
	}
}

func (c *ChatManager) getRetainedSessionIdleTimeout() time.Duration {
	if c != nil && c.retainedSessionIdleTimeout > 0 {
		return c.retainedSessionIdleTimeout
	}
	if !viper.IsSet("chat.retained_session_idle_timeout_ms") {
		return defaultRetainedSessionIdleTTL
	}
	ms := viper.GetInt64("chat.retained_session_idle_timeout_ms")
	if ms <= 0 {
		return defaultRetainedSessionIdleTTL
	}
	return time.Duration(ms) * time.Millisecond
}

func (c *ChatManager) cancelRetainedSessionCleanup(reason string) {
	if c == nil {
		return
	}
	c.retainedSessionCleanupMu.Lock()
	timer := c.retainedSessionCleanupTimer
	c.retainedSessionCleanupTimer = nil
	c.retainedSessionCleanupTarget = nil
	c.retainedSessionCleanupMu.Unlock()

	if timer != nil {
		timer.Stop()
		log.Debugf("Device %s canceled ChatSession retention cleanup: reason=%s", c.DeviceID, reason)
	}
}

func (c *ChatManager) scheduleRetainedSessionCleanup(session *ChatSession, reason string) {
	if c == nil || session == nil || session.IsClosing() {
		return
	}

	timeout := c.getRetainedSessionIdleTimeout()
	c.cancelRetainedSessionCleanup("reschedule")

	c.retainedSessionCleanupMu.Lock()
	c.retainedSessionCleanupTarget = session
	c.retainedSessionCleanupTimer = time.AfterFunc(timeout, func() {
		c.runRetainedSessionCleanup(session, timeout)
	})
	c.retainedSessionCleanupMu.Unlock()

	log.Infof(
		"Device %s ChatSession has entered the reserved state: timeout=%s reason=%s",
		c.DeviceID,
		timeout,
		reason,
	)
}

func (c *ChatManager) runRetainedSessionCleanup(session *ChatSession, timeout time.Duration) {
	if c == nil || session == nil {
		return
	}

	c.retainedSessionCleanupMu.Lock()
	if c.retainedSessionCleanupTarget != session {
		c.retainedSessionCleanupMu.Unlock()
		return
	}
	c.retainedSessionCleanupTimer = nil
	c.retainedSessionCleanupTarget = nil
	c.retainedSessionCleanupMu.Unlock()

	c.sessionMu.RLock()
	currentSession := c.session
	c.sessionMu.RUnlock()
	if currentSession != session || session.IsClosing() {
		return
	}

	log.Infof("Device %s ChatSession has been idle for more than %s, perform a thorough cleanup", c.DeviceID, timeout)
	session.CloseWithReason(chatSessionCloseReasonRetainedIdleTimeout)
}

func (c *ChatManager) finishSessionStart(session *ChatSession, allowFreshHello bool, startErr error) {
	var waitCh chan struct{}

	c.sessionMu.Lock()
	if c.startingSession == session {
		waitCh = c.startingSessionDone
		c.startingSession = nil
		c.startingSessionDone = nil
		if startErr == nil && !session.IsClosing() {
			c.session = session
			if allowFreshHello {
				c.needFreshHello = false
			}
		}
	}
	c.sessionMu.Unlock()

	if waitCh != nil {
		close(waitCh)
	}
}

func (c *ChatManager) handleSessionClosed(session *ChatSession, reason string) {
	var waitCh chan struct{}

	c.cancelRetainedSessionCleanup("session_closed")

	c.sessionMu.Lock()
	switch {
	case c.session == session:
		c.session = nil
	case c.startingSession == session:
		waitCh = c.startingSessionDone
		c.startingSession = nil
		c.startingSessionDone = nil
	default:
		c.sessionMu.Unlock()
		log.Debugf("Device %s received an expired ChatSession close callback and ignored subsequent cleanup", c.DeviceID)
		return
	}
	c.sessionMu.Unlock()

	if waitCh != nil {
		close(waitCh)
		log.Debugf("Device %s ChatSession closed during startup phase, startup state cleared", c.DeviceID)
		return
	}

	if reason == chatSessionCloseReasonManagerShutdown {
		// manager_shutdown may come from the server's active shutdown(true) or from the underlying link
		// Resource cleanup after disconnection shutdown(false). Active shutdown(true) scenario is still controlled by
		// ServerTransport.Close() sends goodbye to avoid accidental transmission on the passive disconnection path.
		return
	}

	if c.serverTransport == nil {
		return
	}

	switch c.serverTransport.GetTransportType() {
	case types_conn.TransportTypeWebsocket:
		if err := c.shutdown(true); err != nil {
			log.Warnf("Failed to close websocket transport: %v", err)
		}
	case types_conn.TransportTypeMqttUdp:
		if reason == chatSessionCloseReasonRetainedIdleTimeout {
			c.setNeedFreshHello(true)
			c.resetSpeakPathAfterServerSessionClose(reason)
			return
		}
		if !shouldSendMqttGoodbyeOnSessionClose(reason) {
			return
		}
		c.setNeedFreshHello(true)
		c.resetSpeakPathAfterServerSessionClose(reason)
		if err := c.serverTransport.SendMqttGoodbye(); err != nil {
			log.Warnf("Sending mqtt goodbye failed: %v", err)
		}
	}
}

func shouldSendMqttGoodbyeOnSessionClose(reason string) bool {
	switch reason {
	case chatSessionCloseReasonExplicitExit,
		chatSessionCloseReasonFatalError,
		chatSessionCloseReasonAudioIdleTimeout:
		return true
	default:
		return false
	}
}

func (c *ChatManager) shutdown(closeTransport bool) error {
	var shutdownErr error

	c.closeOnce.Do(func() {
		c.cancelRetainedSessionCleanup("manager_shutdown")

		if c.clientState != nil {
			log.Infof("Close ChatManager, device %s", c.clientState.DeviceID)
		}

		c.sessionMu.RLock()
		session := c.session
		startingSession := c.startingSession
		c.sessionMu.RUnlock()

		if session != nil {
			session.CloseWithReason(chatSessionCloseReasonManagerShutdown)
		}
		if startingSession != nil && startingSession != session {
			startingSession.CloseWithReason(chatSessionCloseReasonManagerShutdown)
		}

		if c.clientState != nil && c.mcpTransport != nil {
			mcp.CloseDeviceIotOverMcp(c.clientState.DeviceID, c.mcpTransport)
		}

		if c.hookHub != nil {
			c.hookHub.Close()
		}

		if closeTransport {
			c.managerClosing.Store(true)
			defer c.managerClosing.Store(false)

			if c.serverTransport != nil {
				shutdownErr = c.serverTransport.Close()
			} else if c.transport != nil {
				shutdownErr = c.transport.Close()
			}
		} else if c.serverTransport != nil {
			if err := c.serverTransport.CloseWithoutTransport(); err != nil {
				log.Warnf("Failed to close server transport wrapper: %v", err)
			}
		}

		if c.cancel != nil {
			c.cancel()
		}
	})

	return shutdownErr
}

func (c *ChatManager) Close() error {
	return c.shutdown(true)
}

func (c *ChatManager) OnClose(deviceId string) {
	log.Infof("Device %s disconnected", deviceId)
	if c.managerClosing.Load() {
		return
	}
	if err := c.shutdown(false); err != nil {
		log.Warnf("Resource cleanup failed after connection closed: %v", err)
	}
}

func (c *ChatManager) GetClientState() *ClientState {
	return c.clientState
}

func (c *ChatManager) GetDeviceId() string {
	return c.clientState.DeviceID
}

func (c *ChatManager) GetTransportType() string {
	if c == nil || c.serverTransport == nil {
		return ""
	}
	if c.ctx != nil && c.ctx.Err() != nil {
		return ""
	}
	if c.serverTransport.IsClosed() {
		return ""
	}
	if awareTransport, ok := c.transport.(brokerOnlineAwareTransport); ok && !awareTransport.IsBrokerOnline() {
		return ""
	}
	return c.serverTransport.GetTransportType()
}

func (c *ChatManager) WarmupMcp() {
	c.helloMu.Lock()
	defer c.helloMu.Unlock()
	c.scheduleMcpInitLocked()
}

func (c *ChatManager) GetSession() *ChatSession {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()
	return c.session
}

func (c *ChatManager) InjectMessage(message string, skipLlm bool, autoListen bool) error {
	c.cancelRetainedSessionCleanup("inject_message")
	if err := c.prepareSpeakPathForInjectedSpeech(message, autoListen); err != nil {
		return err
	}
	session, err := c.ensureSession()
	if err != nil {
		return err
	}
	options := llmResponseChannelOptions{
		onTTSPlaybackStart: c.newInjectedSpeechStartHook(),
		ttsTurnEndPolicy:   injectedSpeechTTSTurnEndPolicy(autoListen),
	}
	if skipLlm {
		return session.AddTextToTTSQueueWithOptions(message, options)
	}
	return session.AddAsrResultToQueueWithOptions(message, nil, options)
}

func (c *ChatManager) prepareSpeakPathForInjectedSpeech(previewText string, autoListen bool) error {
	if c == nil || c.serverTransport == nil {
		return nil
	}
	if c.serverTransport.GetTransportType() != types_conn.TransportTypeMqttUdp {
		log.Debugf("Device %s injected message skipping speak_request: transport=%s", c.DeviceID, c.serverTransport.GetTransportType())
		return nil
	}
	if !c.shouldSendSpeakRequest(time.Now()) {
		log.Debugf("Device %s injects messages to reuse the existing broadcast link and skip speak_request", c.DeviceID)
		return nil
	}
	if _, err := c.ensureClientSessionID(); err != nil {
		return err
	}

	needSessionBootstrap := c.requiresHelloBootstrapForSession()
	pending, created := c.getOrCreatePendingSpeakRequest()
	if created {
		if err := c.serverTransport.SendSpeakRequest(previewText, autoListen); err != nil {
			c.finishPendingSpeakRequest(pending, err)
			return err
		}
		log.Infof("Device %s has sent speak_request, session_id=%s", c.DeviceID, pending.sessionID)
	}

	waitCtx := c.ctx
	if waitCtx == nil {
		waitCtx = context.Background()
	}
	if err := c.waitPendingSpeakRequest(waitCtx, pending); err != nil {
		return err
	}
	if needSessionBootstrap {
		if err := c.waitForInjectedSpeechSession(waitCtx); err != nil {
			return err
		}
	}
	return nil
}

func (c *ChatManager) shouldSendSpeakRequest(now time.Time) bool {
	if c == nil || c.serverTransport == nil {
		return false
	}
	if c.serverTransport.GetTransportType() != types_conn.TransportTypeMqttUdp {
		log.Debugf("Device %s speak_request determination: transport=%s, no need to send", c.DeviceID, c.serverTransport.GetTransportType())
		return false
	}
	if c.requiresHelloBootstrapForSession() {
		log.Debugf("Device %s speak_request determination: ChatSession relies on new hello to be established and needs to be sent", c.DeviceID)
		return true
	}
	if c.needsMqttRebootstrap() {
		log.Debugf("Device %s speak_request judgment: MQTT link needs to be reestablished and needs to be sent", c.DeviceID)
		return true
	}
	if c.isConversationActive() {
		log.Debugf("Device %s speak_request determination: currently in session, skip sending", c.DeviceID)
		return false
	}

	warmAt := c.currentSpeakPathWarmAt()
	if warmAt <= 0 {
		log.Debugf("Device %s speak_request judgment: There is no reusable hot link and needs to be sent", c.DeviceID)
		return true
	}
	reuseWindow := speakRequestReuseWindow()
	idleFor := now.Sub(time.UnixMilli(warmAt))
	if idleFor <= reuseWindow {
		log.Debugf("Device %s speak_request judgment: hot link is still valid idle_for=%s reuse_window=%s, skip sending", c.DeviceID, idleFor, reuseWindow)
		return false
	}
	log.Debugf("Device %s speak_request judgment: hot link has expired idle_for=%s reuse_window=%s, needs to be sent", c.DeviceID, idleFor, reuseWindow)
	return true
}

func (c *ChatManager) requiresHelloBootstrapForSession() bool {
	if c == nil {
		return false
	}
	if c.GetSession() != nil {
		return false
	}
	if !c.helloInited {
		return true
	}
	return c.requiresFreshHello()
}

func (c *ChatManager) ensureClientSessionID() (string, error) {
	if c == nil || c.clientState == nil {
		return "", fmt.Errorf("clientState is not initialized")
	}
	sessionID := strings.TrimSpace(c.clientState.SessionID)
	if sessionID != "" {
		return sessionID, nil
	}
	session, err := auth.A().CreateSession(c.DeviceID)
	if err != nil {
		return "", fmt.Errorf("Failed to create session: %v", err)
	}
	c.clientState.SessionID = session.ID
	return session.ID, nil
}

func (c *ChatManager) waitForInjectedSpeechSession(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if _, err := c.ensureSession(); err == nil {
		return nil
	} else if !shouldRetryInjectedSpeechSessionWait(err) {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	timeout := c.speakReadyTimeout
	if timeout <= 0 {
		timeout = defaultSpeakReadyTimeout
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		if _, err := c.ensureSession(); err == nil {
			return nil
		} else {
			lastErr = err
			if !shouldRetryInjectedSpeechSessionWait(err) {
				return err
			}
		}

		select {
		case <-waitCtx.Done():
			if waitCtx.Err() == context.DeadlineExceeded {
				if lastErr != nil {
					return fmt.Errorf("Timeout waiting for ChatSession to be established: %w", lastErr)
				}
				return fmt.Errorf("Timeout waiting for ChatSession to be established")
			}
			return waitCtx.Err()
		case <-ticker.C:
		}
	}
}

func shouldRetryInjectedSpeechSessionWait(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "hello is not initialized") ||
		strings.Contains(msg, "resend hello") ||
		strings.Contains(msg, "is starting") ||
		strings.Contains(msg, "is closing")
}

func (c *ChatManager) isConversationActive() bool {
	if c == nil || c.clientState == nil {
		return false
	}
	if phase := c.clientState.GetListenPhase(); phase != "" && phase != ListenPhaseIdle {
		return true
	}
	switch c.clientState.GetStatus() {
	case ClientStatusListening, ClientStatusLLMStart, ClientStatusTTSStart:
		return true
	}
	session := c.GetSession()
	return session != nil && session.IsTTSActive()
}

func (c *ChatManager) getOrCreatePendingSpeakRequest() (*pendingSpeakRequest, bool) {
	c.speakRequestMu.Lock()
	defer c.speakRequestMu.Unlock()

	if c.pendingSpeakRequest != nil {
		return c.pendingSpeakRequest, false
	}

	sessionID := ""
	if c.clientState != nil {
		sessionID = strings.TrimSpace(c.clientState.SessionID)
	}
	pending := &pendingSpeakRequest{
		sessionID: sessionID,
		done:      make(chan struct{}),
	}
	timeout := c.speakReadyTimeout
	if timeout <= 0 {
		timeout = defaultSpeakReadyTimeout
	}
	pending.timer = time.AfterFunc(timeout, func() {
		c.finishPendingSpeakRequest(pending, fmt.Errorf("Waiting for speak_ready timeout"))
	})
	c.pendingSpeakRequest = pending
	return pending, true
}

func (c *ChatManager) waitPendingSpeakRequest(ctx context.Context, pending *pendingSpeakRequest) error {
	if pending == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-pending.done:
		return pending.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *ChatManager) finishPendingSpeakRequest(pending *pendingSpeakRequest, err error) {
	if pending == nil {
		return
	}
	c.speakRequestMu.Lock()
	if c.pendingSpeakRequest == pending {
		c.pendingSpeakRequest = nil
	}
	c.speakRequestMu.Unlock()
	pending.resolve(err)
}

func (c *ChatManager) refreshSpeakPathWarmFromTransport() {
	if c == nil || c.serverTransport == nil || !c.serverTransport.HasActiveUDPBinding() {
		return
	}
	if c.hasPendingSpeakRequest() {
		log.Debugf("Device %s currently has a pending speak_request, skip refreshing the hot link", c.DeviceID)
		return
	}
	if c.needsMqttRebootstrap() {
		log.Debugf("Device %s currently has MQTT session rebuild flag, skip refreshing hot link", c.DeviceID)
		return
	}
	if ts := c.serverTransport.GetUDPLastActiveTs(); ts > 0 {
		c.updateSpeakPathWarmAt(ts)
		return
	}
	c.markSpeakPathWarm(time.Now())
}

func (c *ChatManager) currentSpeakPathWarmAt() int64 {
	if c == nil {
		return 0
	}
	latest := c.lastSpeakPathWarmAt.Load()
	if c.serverTransport != nil {
		if transportTs := c.serverTransport.GetUDPLastActiveTs(); transportTs > latest {
			latest = transportTs
		}
	}
	return latest
}

func (c *ChatManager) markSpeakPathWarm(ts time.Time) {
	if ts.IsZero() {
		ts = time.Now()
	}
	c.updateSpeakPathWarmAt(ts.UnixMilli())
}

func (c *ChatManager) updateSpeakPathWarmAt(ts int64) {
	if c == nil || ts <= 0 {
		return
	}
	for {
		current := c.lastSpeakPathWarmAt.Load()
		if current >= ts {
			return
		}
		if c.lastSpeakPathWarmAt.CompareAndSwap(current, ts) {
			return
		}
	}
}

func speakRequestReuseWindow() time.Duration {
	if !viper.IsSet("chat.speak_request_reuse_window_ms") {
		return defaultSpeakRequestReuseWindow
	}
	ms := viper.GetInt64("chat.speak_request_reuse_window_ms")
	if ms <= 0 {
		return defaultSpeakRequestReuseWindow
	}
	return time.Duration(ms) * time.Millisecond
}

func (c *ChatManager) newInjectedSpeechStartHook() func() {
	if c == nil {
		return nil
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			if c.serverTransport == nil || c.serverTransport.GetTransportType() != types_conn.TransportTypeMqttUdp {
				return
			}
			c.markSpeakPathWarm(time.Now())
		})
	}
}

func injectedSpeechTTSTurnEndPolicy(autoListen bool) ttsTurnEndPolicy {
	if autoListen {
		return ttsTurnEndPolicyNone
	}
	return ttsTurnEndPolicyGoodbyeAndIdle
}

func (c *ChatManager) handleTTSTurnEndPolicy(ctx context.Context, policy ttsTurnEndPolicy, stopErr error) {
	if c == nil || policy == ttsTurnEndPolicyNone {
		return
	}
	if stopErr != nil {
		log.Debugf("Device %s TTS turn end policy skipped: policy=%d err=%v", c.DeviceID, policy, stopErr)
		return
	}

	switch policy {
	case ttsTurnEndPolicyGoodbyeAndIdle:
		if c.serverTransport == nil || c.serverTransport.GetTransportType() != types_conn.TransportTypeMqttUdp {
			return
		}
		if !ttsTurnPlaybackSettledFromContext(ctx) {
			timer := time.NewTimer(ttsPlaybackCompletionGrace)
			defer stopTimer(timer)

			select {
			case <-timer.C:
			case <-c.ctx.Done():
				log.Debugf("Device %s TTS turn end policy delayed goodbye canceled: %v", c.DeviceID, c.ctx.Err())
				return
			}
		}
		if c.managerClosing.Load() || c.serverTransport == nil || c.serverTransport.GetTransportType() != types_conn.TransportTypeMqttUdp {
			return
		}

		session := c.GetSession()
		if session == nil || session.IsClosing() {
			log.Debugf("Device %s TTS turn end policy skipped: session already closed", c.DeviceID)
			return
		}
		session.CloseWithReason(chatSessionCloseReasonExplicitExit)
	}
}

func (c *ChatManager) InjectOpenClawResponse(event openclaw.ResponseDelivery) error {
	c.cancelRetainedSessionCleanup("openclaw_response")
	session, err := c.ensureSession()
	if err != nil {
		return err
	}
	return session.InjectOpenClawResponse(event)
}

func (c *ChatManager) ExitChat() error {
	session := c.GetSession()
	if session == nil {
		return nil
	}
	session.DoExitChat()
	return nil
}

func (c *ChatManager) resetOpenClawModeOnHello(agentIDs ...string) {
	deviceID := strings.TrimSpace(c.clientState.DeviceID)
	if deviceID == "" {
		return
	}

	openclawManager := openclaw.GetManager()
	seen := make(map[string]struct{}, len(agentIDs))
	for _, agentID := range agentIDs {
		agentID = strings.TrimSpace(agentID)
		if agentID == "" {
			continue
		}
		if _, exists := seen[agentID]; exists {
			continue
		}
		seen[agentID] = struct{}{}
		if openclawManager.ExitMode(agentID, deviceID) {
			log.Infof("Device %s reset OpenClaw mode after hello: agent=%s", deviceID, agentID)
		}
	}
}

func (c *ChatManager) refreshDeviceConfigOnHello() error {
	configProvider, err := userconfig.GetProvider(viper.GetString("config_provider.type"))
	if err != nil {
		return fmt.Errorf("Failed to get configuration provider: %w", err)
	}

	deviceConfig, err := configProvider.GetUserConfig(c.clientState.Ctx, c.clientState.DeviceID)
	if err != nil {
		return fmt.Errorf("Failed to obtain device configuration: %w", err)
	}
	deviceConfig.MemoryMode = NormalizeMemoryMode(deviceConfig.MemoryMode)
	deviceConfig.SpeakerChatMode = NormalizeSpeakerChatMode(deviceConfig.SpeakerChatMode)

	prevAgentID := c.clientState.AgentID
	c.clientState.AgentID = deviceConfig.AgentId
	c.clientState.DeviceConfig = deviceConfig
	c.clientState.SystemPrompt = deviceConfig.SystemPrompt
	c.clientState.SpeakerTTSConfig = nil
	applyOutputAudioFormatForTTS(c.clientState)

	log.Infof("Device %s hello refreshed configuration successfully, agent: %s -> %s", c.clientState.DeviceID, prevAgentID, deviceConfig.AgentId)
	return nil
}
