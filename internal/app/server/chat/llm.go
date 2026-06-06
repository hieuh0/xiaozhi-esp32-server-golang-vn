package chat

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	. "xiaozhi-esp32-server-golang/internal/data/client"
	chathooks "xiaozhi-esp32-server-golang/internal/domain/chat/hooks"
	"xiaozhi-esp32-server-golang/internal/domain/chat/streamtransform"
	config_types "xiaozhi-esp32-server-golang/internal/domain/config/types"
	"xiaozhi-esp32-server-golang/internal/domain/eventbus"
	"xiaozhi-esp32-server-golang/internal/domain/llm"
	llm_common "xiaozhi-esp32-server-golang/internal/domain/llm/common"
	"xiaozhi-esp32-server-golang/internal/domain/speaker"
	"xiaozhi-esp32-server-golang/internal/pool"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"
)

const (
	MaxMessageCount = 10

	McpReadResourcePageSize       = 100 * 1024
	McpReadResourceStreamDoneFlag = "[DONE]"
)

// Use a dedicated context key type to avoid collisions.
type contextKey int

const (
	ttsPlaybackCompletionGrace time.Duration = 150 * time.Millisecond
	fullTextKey                contextKey    = iota
	toolRoundMessagesKey
	ttsTurnTrackerKey
	ttsPlaybackStartHookKey
	ttsTurnEndPolicyKey
	ttsTurnEndPolicyHandlerKey
	ttsTurnPlaybackSettledKey
)

const (
	interruptExtraKey      = "interrupt"
	interruptByExtraKey    = "interrupt_by"
	interruptStageExtraKey = "interrupt_stage"
	interruptContentSuffix = " [user interrupted]"
)

// GetLastMessageID returns the most recently saved MessageID for two-phase persistence.
func (l *LLMManager) GetLastMessageID(role string) (string, bool) {
	l.lastMessageIDMu.RLock()
	defer l.lastMessageIDMu.RUnlock()
	id, ok := l.lastMessageID[role]
	return id, ok
}

type LLMResponseChannelItem struct {
	ctx          context.Context
	userMessage  *schema.Message
	responseChan chan llm_common.LLMResponseStruct
	onStartFunc  func(args ...any)
	onEndFunc    func(err error, args ...any)
}

type llmHandleResult struct {
	ok                      bool
	suppressProtocolTtsStop bool
}

func llmHandleResultFromArgs(args []any) llmHandleResult {
	if len(args) == 0 {
		return llmHandleResult{}
	}
	result, ok := args[0].(llmHandleResult)
	if !ok {
		return llmHandleResult{}
	}
	return result
}

func (l *LLMManager) finishTTSTurn(ctx context.Context, stopErr error, result llmHandleResult) {
	l.finishTTSTurnWithReason(ctx, stopErr, result, "LLMManager.finishTTSTurn")
}

func (l *LLMManager) finishTTSTurnWithReason(ctx context.Context, stopErr error, result llmHandleResult, reason string) {
	if l == nil || l.ttsManager == nil {
		return
	}

	if result.suppressProtocolTtsStop {
		// Media tools return here after playback completes. The protocol-level
		// tts_stop is still required or the client remains in the speaking state.
		log.Debugf("media output completed; sending tts stop through the standard TTS finalization path")
	}

	l.ttsManager.EnqueueTtsStopWithReason(ctx, reason)
	l.ttsManager.RequestTurnEnd(ctx, stopErr)
}

type llmResponseChannelOptions struct {
	disableTTSCommands bool
	onStartFunc        func(args ...any)
	onEndFunc          func(err error, args ...any)
	onTTSPlaybackStart func()
	ttsTurnEndPolicy   ttsTurnEndPolicy
}

type ttsPlaybackStartHook func()

type ttsTurnEndPolicy uint8

const (
	ttsTurnEndPolicyNone ttsTurnEndPolicy = iota
	ttsTurnEndPolicyGoodbyeAndIdle
)

type ttsTurnEndPolicyHandler interface {
	handleTTSTurnEndPolicy(ctx context.Context, policy ttsTurnEndPolicy, stopErr error)
}

func withTTSPlaybackStartHook(ctx context.Context, hook func()) context.Context {
	if ctx == nil || hook == nil {
		return ctx
	}

	var once sync.Once
	return context.WithValue(ctx, ttsPlaybackStartHookKey, ttsPlaybackStartHook(func() {
		once.Do(hook)
	}))
}

func ttsPlaybackStartHookFromContext(ctx context.Context) func() {
	if ctx == nil {
		return nil
	}
	hook, ok := ctx.Value(ttsPlaybackStartHookKey).(ttsPlaybackStartHook)
	if !ok || hook == nil {
		return nil
	}
	return func() {
		hook()
	}
}

func withTTSTurnEndPolicy(ctx context.Context, policy ttsTurnEndPolicy) context.Context {
	if ctx == nil || policy == ttsTurnEndPolicyNone {
		return ctx
	}
	return context.WithValue(ctx, ttsTurnEndPolicyKey, policy)
}

func ttsTurnEndPolicyFromContext(ctx context.Context) ttsTurnEndPolicy {
	if ctx == nil {
		return ttsTurnEndPolicyNone
	}
	policy, ok := ctx.Value(ttsTurnEndPolicyKey).(ttsTurnEndPolicy)
	if !ok {
		return ttsTurnEndPolicyNone
	}
	return policy
}

func withTTSTurnEndPolicyHandler(ctx context.Context, handler ttsTurnEndPolicyHandler) context.Context {
	if ctx == nil || handler == nil {
		return ctx
	}
	return context.WithValue(ctx, ttsTurnEndPolicyHandlerKey, handler)
}

func withTTSTurnPlaybackSettled(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ttsTurnPlaybackSettledKey, true)
}

func ttsTurnPlaybackSettledFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	settled, ok := ctx.Value(ttsTurnPlaybackSettledKey).(bool)
	return ok && settled
}

func ttsTurnEndPolicyHandlerFromContext(ctx context.Context) ttsTurnEndPolicyHandler {
	if ctx == nil {
		return nil
	}
	handler, ok := ctx.Value(ttsTurnEndPolicyHandlerKey).(ttsTurnEndPolicyHandler)
	if !ok {
		return nil
	}
	return handler
}

type ttsTurnTracker struct {
	mu      sync.Mutex
	pending int
	doneCh  chan struct{}
}

func newTTSTurnTracker() *ttsTurnTracker {
	doneCh := make(chan struct{})
	close(doneCh)
	return &ttsTurnTracker{doneCh: doneCh}
}

func (t *ttsTurnTracker) Add() func(error) {
	if t == nil {
		return func(error) {}
	}

	t.mu.Lock()
	if t.pending == 0 {
		t.doneCh = make(chan struct{})
	}
	t.pending++
	t.mu.Unlock()

	var once sync.Once
	return func(error) {
		once.Do(func() {
			t.mu.Lock()
			defer t.mu.Unlock()
			if t.pending == 0 {
				return
			}
			t.pending--
			if t.pending == 0 {
				close(t.doneCh)
			}
		})
	}
}

func (t *ttsTurnTracker) Wait(ctx context.Context) error {
	if t == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	t.mu.Lock()
	pending := t.pending
	doneCh := t.doneCh
	t.mu.Unlock()

	if pending == 0 {
		return nil
	}

	select {
	case <-doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type LLMManager struct {
	clientState       *ClientState
	session           *ChatSession
	serverTransport   *ServerTransport
	ttsManager        *TTSManager
	transformRegistry *streamtransform.Registry

	einoTools []*schema.ToolInfo

	llmResponseQueue *util.Queue[LLMResponseChannelItem]

	// Store the most recently saved MessageID for two-phase persistence.
	// key: role (user/assistant), value: MessageID
	lastMessageID   map[string]string
	lastMessageIDMu sync.RWMutex // Protect concurrent access to lastMessageID.
}

func NewLLMManager(clientState *ClientState, serverTransport *ServerTransport, ttsManager *TTSManager, session *ChatSession, transformRegistry *streamtransform.Registry) *LLMManager {
	return &LLMManager{
		clientState:       clientState,
		session:           session,
		serverTransport:   serverTransport,
		ttsManager:        ttsManager,
		transformRegistry: transformRegistry,
		llmResponseQueue:  util.NewQueue[LLMResponseChannelItem](10),
		lastMessageID:     make(map[string]string),
	}
}

func (l *LLMManager) openOutputPipeline(ctx context.Context) (*streamtransform.Pipeline, error) {
	if l == nil || l.transformRegistry == nil {
		return &streamtransform.Pipeline{}, nil
	}

	sessionID := ""
	deviceID := ""
	if l.clientState != nil {
		sessionID = l.clientState.SessionID
		deviceID = l.clientState.DeviceID
	}

	return l.transformRegistry.Open(streamtransform.Context{
		Ctx:       ctx,
		SessionID: sessionID,
		DeviceID:  deviceID,
		RequestID: fmt.Sprintf("%s-%d", sessionID, time.Now().UnixNano()),
	})
}

func (l *LLMManager) emitLLMOutputRaw(ctx context.Context, data chathooks.LLMOutputRawData) (chathooks.LLMOutputRawData, bool, error) {
	if l == nil || l.session == nil || l.session.hookHub == nil {
		return data, false, nil
	}
	return l.session.hookHub.EmitLLMOutputRaw(l.session.hookContext(ctx), data)
}

// handleLLMWithContextAndTools processes LLM responses with context control,
// with or without tools, and manages the LLM resource lifecycle internally.
func (l *LLMManager) handleLLMWithContextAndTools(
	ctx context.Context,
	dialogue []*schema.Message,
	tools []*schema.ToolInfo,
) (chan llm_common.LLMResponseStruct, error) {
	// Acquire an LLM resource.
	llmWrapper, err := pool.Acquire[llm.LLMProvider](
		"llm",
		l.clientState.DeviceConfig.Llm.Provider,
		l.clientState.DeviceConfig.Llm.Config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire LLM resource: %w", err)
	}

	// Get the provider.
	llmProvider := llmWrapper.GetProvider()

	// Invoke the LLM provider.
	msgChan := llmProvider.ResponseWithContext(ctx, l.clientState.SessionID, dialogue, tools)

	pipeline, err := l.openOutputPipeline(ctx)
	if err != nil {
		pool.Release(llmWrapper)
		return nil, fmt.Errorf("failed to create LLM output stream transformation pipeline: %w", err)
	}

	// Create the response channel.
	responseChannel := make(chan llm_common.LLMResponseStruct, 2)
	startTs := time.Now().UnixMilli()
	var firstSegment bool
	var rawFullText strings.Builder

	// Start a goroutine to process responses.
	go func() {
		defer func() {
			log.Debugf("full Response with %d tools, fullText: %s", len(tools), rawFullText.String())
			close(responseChannel)
			if closeErr := pipeline.Close(); closeErr != nil {
				log.Warnf("failed to close LLM output stream transformation pipeline: %v", closeErr)
			}
			// Release the resource.
			pool.Release(llmWrapper)
			log.Debugf("LLM resource released")
		}()

		isFirstOutput := true
		llmFirstTokenMarked := false

		emitResponse := func(item streamtransform.Item) bool {
			response := llm_common.LLMResponseStruct{
				IsEnd: item.IsEnd,
			}

			switch item.Kind {
			case streamtransform.ItemKindToolCalls:
				response.ToolCalls = item.ToolCalls
				if len(item.ToolCalls) > 0 {
					response.IsStart = isFirstOutput
				}
			case streamtransform.ItemKindTextDelta, streamtransform.ItemKindTextSegment:
				response.Text = item.Text
				if strings.TrimSpace(item.Text) != "" {
					response.IsStart = isFirstOutput
					if !firstSegment {
						firstSegment = true
						firstSentenceTs := time.Now().UnixMilli()
						if l.clientState.MarkLlmFirstSentenceAt(firstSentenceTs) && l.session != nil {
							l.session.TraceLlmFirstSentence(ctx, firstSentenceTs)
						}
						log.Infof("latency: LLM first sentence: %d ms", firstSentenceTs-startTs)
					}
					if isFirstOutput {
						isFirstOutput = false
					}
				}
			default:
				return true
			}

			if strings.TrimSpace(response.Text) == "" && len(response.ToolCalls) == 0 && !response.IsEnd {
				return true
			}

			select {
			case <-ctx.Done():
				log.Infof("context canceled; stopping LLM response processing: %v, context done, exit", ctx.Err())
				return false
			case responseChannel <- response:
				return true
			}
		}

		pushToPipeline := func(item streamtransform.Item) (bool, error) {
			items, stop, err := pipeline.Push(item)
			if err != nil {
				return false, err
			}
			for _, out := range items {
				if !emitResponse(out) {
					return true, nil
				}
			}
			return stop, nil
		}

		pushRawText := func(delta string, isEnd bool, errVal error) (bool, error) {
			payload, stop, hookErr := l.emitLLMOutputRaw(ctx, chathooks.LLMOutputRawData{
				Delta:    delta,
				FullText: rawFullText.String(),
				IsEnd:    isEnd,
				Err:      errVal,
			})
			if hookErr != nil {
				log.Warnf("LLM_OUTPUT_RAW hook failed: %v", hookErr)
			}
			if stop {
				log.Infof("LLM_OUTPUT_RAW hook requested the current flow to stop")
				return true, nil
			}
			if payload.Delta != "" {
				rawFullText.WriteString(payload.Delta)
			}
			return pushToPipeline(streamtransform.Item{
				Kind:  streamtransform.ItemKindTextDelta,
				Text:  payload.Delta,
				IsEnd: payload.IsEnd,
			})
		}

		pushRawToolCalls := func(toolCalls []schema.ToolCall) (bool, error) {
			payload, stop, hookErr := l.emitLLMOutputRaw(ctx, chathooks.LLMOutputRawData{
				FullText:  rawFullText.String(),
				ToolCalls: toolCalls,
			})
			if hookErr != nil {
				log.Warnf("LLM_OUTPUT_RAW hook failed: %v", hookErr)
			}
			if stop {
				log.Infof("LLM_OUTPUT_RAW hook requested the current flow to stop")
				return true, nil
			}
			if len(payload.ToolCalls) == 0 {
				return false, nil
			}
			return pushToPipeline(streamtransform.Item{
				Kind:      streamtransform.ItemKindToolCalls,
				ToolCalls: payload.ToolCalls,
			})
		}

		for {
			select {
			case <-ctx.Done():
				log.Infof("context canceled; stopping LLM response processing: %v, context done, exit", ctx.Err())
				return
			case message, ok := <-msgChan:
				if !ok {
					stop, pushErr := pushRawText("", true, nil)
					if pushErr != nil {
						log.Errorf("failed to process LLM end-of-stream: %v", pushErr)
					}
					if stop || pushErr != nil {
						return
					}
					return
				}
				if message == nil {
					continue
				}
				if llm.IsLLMErrorMessage(message) {
					errMsg := llm.LLMErrorMessage(message)
					log.Warnf("LLM returned an error: %s", errMsg)
					stop, pushErr := pushRawText(errMsg, true, nil)
					if pushErr != nil {
						log.Errorf("failed to process LLM error output: %v", pushErr)
					}
					if stop || pushErr != nil {
						return
					}
					return
				}
				if message.Content != "" {
					if !llmFirstTokenMarked {
						firstTokenTs := time.Now().UnixMilli()
						l.clientState.MarkLlmFirstToken()
						if l.session != nil {
							l.session.TraceLlmFirstToken(ctx, firstTokenTs)
						}
						llmFirstTokenMarked = true
					}
					stop, pushErr := pushRawText(message.Content, false, nil)
					if pushErr != nil {
						log.Errorf("failed to process LLM text stream: %v", pushErr)
						return
					}
					if stop {
						return
					}
				}
				if len(message.ToolCalls) > 0 {
					log.Infof("processing tool calls: %+v", message.ToolCalls)
					stop, pushErr := pushRawToolCalls(message.ToolCalls)
					if pushErr != nil {
						log.Errorf("failed to process LLM tool stream: %v", pushErr)
						return
					}
					if stop {
						return
					}
				}
			}
		}
	}()

	return responseChannel, nil
}

func (l *LLMManager) Start(ctx context.Context) {
	l.processLLMResponseQueue(ctx)
}

func (l *LLMManager) processLLMResponseQueue(ctx context.Context) {
	for {
		item, err := l.llmResponseQueue.Pop(ctx, 0) // Blocking.
		if err != nil {
			if err == util.ErrQueueCtxDone {
				return
			}
			// Other errors.
			continue
		}

		log.Debugf("processLLMResponseQueue item: %+v", item)
		if item.onStartFunc != nil {
			item.onStartFunc()
		}

		// handleLLMResponse retrieves and populates fullText and toolCalls from the context.
		result, err := l.handleLLMResponse(item.ctx, item.userMessage, item.responseChan)
		if waitErr := waitForTTSTurnDrainIfRoot(item.ctx); err == nil && waitErr != nil {
			err = waitErr
		}

		if item.onEndFunc != nil {
			item.onEndFunc(err, result)
		}
	}
}

func (l *LLMManager) ClearLLMResponseQueue() {
	l.llmResponseQueue.Clear()
}

func (l *LLMManager) AddTextToTTSQueue(text string) error {
	return l.AddTextToTTSQueueWithOptions(text, llmResponseChannelOptions{})
}

func (l *LLMManager) AddTextToTTSQueueWithOptions(text string, options llmResponseChannelOptions) error {
	log.Debugf("AddTextToTTSQueue text: %s", text)
	msg := &schema.Message{
		Role:    schema.User,
		Content: text,
	}
	llmResponseChan := make(chan llm_common.LLMResponseStruct, 10)
	llmResponseChan <- llm_common.LLMResponseStruct{
		IsStart: true,
		IsEnd:   true,
		Text:    text,
	}
	close(llmResponseChan)

	sessionCtx := l.clientState.SessionCtx.Get(l.clientState.Ctx)
	ctx := l.clientState.AfterAsrSessionCtx.Get(sessionCtx)
	ctx = withTTSPlaybackStartHook(ctx, options.onTTSPlaybackStart)
	ctx = withTTSTurnEndPolicy(ctx, options.ttsTurnEndPolicy)
	if err := l.HandleLLMResponseChannelAsyncWithOptions(ctx, msg, llmResponseChan, options); err != nil {
		log.Warnf("AddTextToTTSQueue enqueue failed: %v", err)
		return err
	}

	return nil
}

func chainLLMResponseStartHooks(hooks ...func(args ...any)) func(args ...any) {
	filtered := make([]func(args ...any), 0, len(hooks))
	for _, hook := range hooks {
		if hook != nil {
			filtered = append(filtered, hook)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return func(args ...any) {
		for _, hook := range filtered {
			hook(args...)
		}
	}
}

func chainLLMResponseEndHooks(hooks ...func(err error, args ...any)) func(err error, args ...any) {
	filtered := make([]func(err error, args ...any), 0, len(hooks))
	for _, hook := range hooks {
		if hook != nil {
			filtered = append(filtered, hook)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return func(err error, args ...any) {
		for _, hook := range filtered {
			hook(err, args...)
		}
	}
}

func (l *LLMManager) HandleLLMResponseChannelAsync(ctx context.Context, userMessage *schema.Message, responseChan chan llm_common.LLMResponseStruct) error {
	return l.handleLLMResponseChannelAsync(ctx, userMessage, responseChan, llmResponseChannelOptions{})
}

func (l *LLMManager) HandleLLMResponseChannelAsyncWithOptions(ctx context.Context, userMessage *schema.Message, responseChan chan llm_common.LLMResponseStruct, options llmResponseChannelOptions) error {
	return l.handleLLMResponseChannelAsync(ctx, userMessage, responseChan, options)
}

func (l *LLMManager) handleLLMResponseChannelAsync(ctx context.Context, userMessage *schema.Message, responseChan chan llm_common.LLMResponseStruct, options llmResponseChannelOptions) error {
	ctx = ensureTTSTurnTrackerInContext(ctx)
	ctx = withTTSPlaybackStartHook(ctx, options.onTTSPlaybackStart)
	ctx = withTTSTurnEndPolicy(ctx, options.ttsTurnEndPolicy)

	needSendTtsCmd := true
	val := ctx.Value("nest")
	nest := 0
	log.Debugf("AddLLMResponseChannel nest: %+v", val)
	if n, ok := val.(int); ok {
		nest = n
		if nest > 1 {
			needSendTtsCmd = false
		}
	}
	if options.disableTTSCommands {
		needSendTtsCmd = false
	}

	// Initialize or reuse fullText in the context for chat history.
	// Reuse it when an LLM request continues after a tool call; otherwise create it.
	var fullText *strings.Builder
	if existingFullText, ok := ctx.Value(fullTextKey).(*strings.Builder); ok && existingFullText != nil {
		fullText = existingFullText
		log.Debugf("reusing fullText, current length: %d", fullText.Len())
	} else {
		fullText = &strings.Builder{}
		ctx = context.WithValue(ctx, fullTextKey, fullText)
		log.Debugf("created new fullText")
	}

	var onStartFunc func(...any)
	var onEndFunc func(err error, args ...any)

	if needSendTtsCmd {
		onStartFunc = func(...any) {
			// Clear the TTS audio cache only for the first LLM call, based on context nest.
			val := ctx.Value("nest")
			if nest, ok := val.(int); !ok || nest <= 1 {
				// First call or no nest value: clear the TTS audio cache.
				l.ttsManager.ClearAudioHistory()
				log.Debugf("onStartFunc first call; cleared TTS audio cache")
			}
			l.ttsManager.EnqueueTtsStartWithReason(ctx, "LLMManager.handleLLMResponseChannelAsync onStart")
		}
		onEndFunc = func(err error, args ...any) {
			handleResult := llmHandleResultFromArgs(args)
			l.clientState.MarkLlmEnd()
			if l.session != nil {
				l.session.TraceLlmEnd(ctx, time.Now().UnixMilli(), err)
			}
			strFullText := fullText.String()

			l.finishTTSTurnWithReason(ctx, err, handleResult, "LLMManager.handleLLMResponseChannelAsync onEnd")

			// Read fullText from the closure.
			audioData := l.ttsManager.GetAndClearAudioHistory()

			// Calculate total audio size across all frames.
			audioSize := 0
			for _, frame := range audioData {
				audioSize += len(frame)
			}

			// Publish the event only for the first call (nest <= 1).
			if nest <= 1 {
				// Get the assistant MessageID from LLMManager.
				// Skip phase two when phase-one persistence has not completed.
				messageID, ok := l.GetLastMessageID(string(schema.Assistant))
				if !ok {
					log.Warnf("MessageID not found when TTS completed; skipping phase-two audio update")
					return
				}

				// Publish phase two: update audio.
				assistantMsg := schema.AssistantMessage(strFullText, nil)
				eventbus.Get().Publish(eventbus.TopicAddMessage, &eventbus.AddMessageEvent{
					ClientState: l.clientState,
					Msg:         *assistantMsg,
					MessageID:   messageID,
					AudioData:   audioData, // Phase two includes audio.
					AudioSize:   audioSize,
					SampleRate:  l.clientState.OutputAudioFormat.SampleRate,
					Channels:    l.clientState.OutputAudioFormat.Channels,
					Timestamp:   time.Now(),
					IsUpdate:    true, // Update the message.
				})
			}
		}
	}

	onStartFunc = chainLLMResponseStartHooks(onStartFunc, options.onStartFunc)
	onEndFunc = chainLLMResponseEndHooks(onEndFunc, options.onEndFunc)

	item := LLMResponseChannelItem{
		ctx:          ctx,
		userMessage:  userMessage,
		responseChan: responseChan,
		onStartFunc:  onStartFunc,
		onEndFunc:    onEndFunc,
	}

	err := l.llmResponseQueue.Push(item)
	if err != nil {
		log.Warnf("llmResponseQueue is full or closed; dropping message")
		return fmt.Errorf("llmResponseQueue is full or closed; message dropped")
	}
	return nil
}

func (l *LLMManager) HandleLLMResponseChannelSync(ctx context.Context, userMessage *schema.Message, llmResponseChannel chan llm_common.LLMResponseStruct, einoTools []*schema.ToolInfo) (bool, error) {
	ctx = ensureTTSTurnTrackerInContext(ctx)

	needSendTtsCmd := true
	val := ctx.Value("nest")
	nest := 0
	log.Debugf("AddLLMResponseChannel nest: %+v", val)
	if n, ok := val.(int); ok {
		nest = n
		if nest > 1 {
			needSendTtsCmd = false
		}
	}

	// Initialize or reuse fullText in the context for chat history.
	// Reuse it when an LLM request continues after a tool call; otherwise create it.
	var fullText *strings.Builder
	if existingFullText, ok := ctx.Value(fullTextKey).(*strings.Builder); ok && existingFullText != nil {
		fullText = existingFullText
		log.Debugf("reusing fullText, current length: %d", fullText.Len())
	} else {
		fullText = &strings.Builder{}
		ctx = context.WithValue(ctx, fullTextKey, fullText)
		log.Debugf("created new fullText")
	}

	if needSendTtsCmd {
		// Clear the TTS audio cache only for the first LLM call, based on context nest.
		if nest <= 1 {
			// First call or no nest value: clear the TTS audio cache.
			l.ttsManager.ClearAudioHistory()
			log.Debugf("HandleLLMResponseChannelSync first call; cleared TTS audio cache")
		}
		l.ttsManager.EnqueueTtsStartWithReason(ctx, "LLMManager.HandleLLMResponseChannelSync start")
	}

	result, err := l.handleLLMResponse(ctx, userMessage, llmResponseChannel)
	if waitErr := waitForTTSTurnDrainIfRoot(ctx); err == nil && waitErr != nil {
		err = waitErr
	}
	l.clientState.MarkLlmEnd()
	if l.session != nil {
		l.session.TraceLlmEnd(ctx, time.Now().UnixMilli(), err)
	}
	strFullText := fullText.String()

	if needSendTtsCmd {
		l.finishTTSTurnWithReason(ctx, err, result, "LLMManager.HandleLLMResponseChannelSync end")

		// Collect TTS audio and publish the chat history event.
		// LLM responses after tool calls (nest > 1) also accumulate audio without clearing it.
		// Clear the cache and publish only for the first call (nest <= 1).
		audioData := l.ttsManager.GetAndClearAudioHistory()

		// Calculate total audio size across all frames.
		audioSize := 0
		for _, frame := range audioData {
			audioSize += len(frame)
		}

		// Publish the event only for the first call (nest <= 1).
		if nest <= 1 {
			// Get the assistant MessageID from LLMManager.
			// Skip phase two when phase-one persistence has not completed.
			messageID, ok := l.GetLastMessageID(string(schema.Assistant))
			if !ok {
				log.Warnf("MessageID not found when TTS completed; skipping phase-two audio update")
				return result.ok, err
			}

			// Publish phase two: update audio.
			assistantMsg := schema.AssistantMessage(strFullText, nil)
			eventbus.Get().Publish(eventbus.TopicAddMessage, &eventbus.AddMessageEvent{
				ClientState: l.clientState,
				Msg:         *assistantMsg,
				MessageID:   messageID,
				AudioData:   audioData, // Phase two includes audio.
				AudioSize:   audioSize,
				SampleRate:  l.clientState.OutputAudioFormat.SampleRate,
				Channels:    l.clientState.OutputAudioFormat.Channels,
				Timestamp:   time.Now(),
			})
		}
	} else {
		// For nest > 1, audio still accumulates even though no TTS command is sent.
		// It is collected when the root response (nest <= 1) completes.
		log.Debugf("LLM response after tool call (nest=%d); audio will accumulate in the cache", nest)
	}

	return result.ok, err
}

// handleLLMResponse processes an LLM response.
func (l *LLMManager) handleLLMResponse(ctx context.Context, userMessage *schema.Message, llmResponseChannel chan llm_common.LLMResponseStruct) (llmHandleResult, error) {
	log.Debugf("handleLLMResponse start")
	defer log.Debugf("handleLLMResponse end")

	// Get fullText from the context for chat history.
	fullText := ctx.Value(fullTextKey).(*strings.Builder)
	state := l.clientState
	// Keep toolCalls local because internal tool execution does not affect chat history.
	var toolCalls []schema.ToolCall
	toolExecCtx := context.WithValue(ctx, "nest", 2)
	toolExecCtx = context.WithValue(toolExecCtx, fullTextKey, fullText)
	if speechStartHook := ttsPlaybackStartHookFromContext(ctx); speechStartHook != nil {
		toolExecCtx = withTTSPlaybackStartHook(toolExecCtx, speechStartHook)
	}
	if l.clientState.GetMemoryMode() == MemoryModeNone && userMessage != nil {
		toolExecCtx = appendToolRoundMessagesToContext(toolExecCtx, []*schema.Message{userMessage})
	}
	ttsTracker := ttsTurnTrackerFromContext(ctx)
	var onTTSItemEnqueued func() func(error)
	onTTSPlaybackStart := ttsPlaybackStartHookFromContext(ctx)
	if ttsTracker != nil {
		onTTSItemEnqueued = ttsTracker.Add
	}
	toolExecutor := newToolCallExecutor(l, toolExecCtx)
	assistantSaved := false
	result := llmHandleResult{}

	saveInterruptedAssistant := func() {
		if assistantSaved {
			return
		}
		if ctx.Err() == nil {
			return
		}
		text := strings.TrimSpace(fullText.String())
		if text == "" {
			return
		}
		msg := schema.AssistantMessage(text, nil)
		msg.Extra = map[string]any{
			interruptExtraKey:      true,
			interruptByExtraKey:    "user",
			interruptStageExtraKey: "llm",
		}
		if err := l.AddLlmMessage(ctx, msg); err != nil {
			log.Errorf("failed to save interrupted assistant message: %v", err)
			return
		}
		assistantSaved = true
	}

	select {
	case <-ctx.Done():
		saveInterruptedAssistant()
		log.Debugf("handleLLMResponse ctx done, return")
		return result, nil
	default:
	}

	for {
		select {
		case <-ctx.Done():
			// Prioritize cancellation handling.
			saveInterruptedAssistant()
			log.Infof("%s context canceled; stopping LLM response processing, context done, exit", state.DeviceID)
			return result, nil
		default:
			// Continue with a non-blocking check while ctx is active.
			select {
			case llmResponse, ok := <-llmResponseChannel:
				if !ok {
					// The channel is closed; exit the goroutine.
					log.Infof("LLM response channel closed; exiting goroutine")
					result.ok = true
					return result, nil
				}
				if ctx.Err() != nil {
					saveInterruptedAssistant()
					log.Infof("%s context canceled before LLM chunk arrived; dropping late response and exiting", state.DeviceID)
					return result, nil
				}

				log.Debugf("LLM response: %+v", llmResponse)

				if len(llmResponse.ToolCalls) > 0 {
					log.Debugf("received tools: %+v", llmResponse.ToolCalls)
					toolCalls = append(toolCalls, llmResponse.ToolCalls...)
					toolExecutor.Submit(llmResponse.ToolCalls)
				}

				hasText := strings.TrimSpace(llmResponse.Text) != ""
				if hasText || llmResponse.IsStart || llmResponse.IsEnd {
					// Dual-stream finalization depends on IsEnd even when text is empty.
					if err := l.ttsManager.handleTextResponseWithHooks(ctx, llmResponse, false, onTTSItemEnqueued, onTTSPlaybackStart); err != nil {
						result.ok = true
						return result, err
					}
				}
				if hasText {
					fullText.WriteString(llmResponse.Text)
				}

				if llmResponse.IsEnd {
					if len(toolCalls) == 0 {
						// Persist to Redis.
						if userMessage != nil {
							if userMessage.Role == schema.User {
								// Check whether ASR processing already saved the user message.
								// Compare the last message role and content.
								/*messages := l.clientState.GetMessages(1)
								shouldSave := true
								if len(messages) > 0 {
									lastMsg := messages[len(messages)-1]
									if lastMsg.Role == schema.User && lastMsg.Content == userMessage.Content {
										// ASR already saved the user message; skip it.
										shouldSave = false
										log.Debugf("user message already saved during ASR processing; skipping duplicate: %s", userMessage.Content)
									}
								}
								if shouldSave {
									if err := l.AddLlmMessage(ctx, userMessage); err != nil {
										log.Errorf("failed to save user message: %v", err)
									}
								}*/
							}
						}
						strFullText := fullText.String()
						if strings.TrimSpace(strFullText) != "" || len(toolCalls) > 0 {
							if err := l.AddLlmMessage(ctx, schema.AssistantMessage(strFullText, toolCalls)); err != nil {
								log.Errorf("failed to save assistant message: %v", err)
							} else {
								assistantSaved = true
							}
						}
					}
					if len(toolCalls) > 0 {
						toolSummary, err := l.handleToolCallResponse(toolExecCtx, schema.AssistantMessage(fullText.String(), toolCalls), toolCalls, toolExecutor)
						if err != nil {
							log.Errorf("failed to process tool call response: %v", err)
							result.ok = true
							return result, fmt.Errorf("failed to process tool call response: %v", err)
						}
						result.suppressProtocolTtsStop = toolSummary.hasMediaOutput
						if !toolSummary.invokeToolSuccess && strings.TrimSpace(llmResponse.Text) != "" {
							if err := l.ttsManager.handleTextResponseWithHooks(ctx, llmResponse, false, nil, onTTSPlaybackStart); err != nil {
								result.ok = true
								return result, err
							}
							fullText.WriteString(llmResponse.Text)
						}
					}

					result.ok = true
					return result, nil
				}
			case <-ctx.Done():
				// Context canceled; exit the goroutine.
				saveInterruptedAssistant()
				log.Infof("%s context canceled; stopping LLM response processing, context done, exit", state.DeviceID)
				return result, nil
			}
		}
	}
}

func (l *LLMManager) DoLLmRequest(ctx context.Context, userMessage *schema.Message, einoTools []*schema.ToolInfo, isSync bool, speakerResult *speaker.IdentifyResult) error {
	log.Debugf("sending LLM request with tools, seesionID: %s, requestEinoMessages: %+v", l.clientState.SessionID, userMessage)
	clientState := l.clientState

	l.einoTools = einoTools

	// Assemble history and the current user message.
	requestMessages := l.GetMessages(ctx, userMessage, MaxMessageCount, speakerResult)

	if l.session != nil {
		payload, stop, hookErr := l.session.hookHub.EmitLLMInput(l.session.hookContext(ctx), chathooks.LLMInputData{
			UserMessage:     userMessage,
			RequestMessages: requestMessages,
			Tools:           einoTools,
		})
		if hookErr != nil {
			log.Warnf("LLM_INPUT hook failed: %v", hookErr)
		}
		userMessage = payload.UserMessage
		requestMessages = payload.RequestMessages
		einoTools = payload.Tools
		if stop {
			log.Infof("LLM_INPUT hook requested the current flow to stop")
			return nil
		}
	}

	clientState.SetStartLlmTs()
	if l.session != nil {
		l.session.TraceLlmStart(ctx, time.Now().UnixMilli())
	}
	clientState.SetStatus(ClientStatusLLMStart)

	// Process the LLM response; the helper manages resource ownership.
	responseSentences, err := l.handleLLMWithContextAndTools(
		ctx,
		requestMessages,
		einoTools,
	)
	if err != nil {
		log.Errorf("failed to send LLM request with tools, seesionID: %s, error: %v", l.clientState.SessionID, err)
		return fmt.Errorf("failed to send LLM request with tools: %v", err)
	}

	log.Debugf("DoLLmRequest goroutine started - SessionID: %s, context state: %v", l.clientState.SessionID, ctx.Err())

	if isSync {
		// Synchronous processing; defer in handleLLMWithContextAndTools releases the resource.
		_, err := l.HandleLLMResponseChannelSync(ctx, userMessage, responseSentences, einoTools)
		if err != nil {
			log.Errorf("failed to process LLM response, seesionID: %s, error: %v", l.clientState.SessionID, err)
			return err
		}
	} else {
		// Asynchronous processing; defer in handleLLMWithContextAndTools releases the resource.
		err = l.HandleLLMResponseChannelAsync(ctx, userMessage, responseSentences)
		if err != nil {
			log.Errorf("failed to process LLM response, seesionID: %s, error: %v", l.clientState.SessionID, err)
		}
	}

	log.Debugf("DoLLmRequest finished - SessionID: %s", l.clientState.SessionID)

	return nil
}

// AddMessage adds any message type to chat history through the shared entry point.
func (l *LLMManager) AddMessage(ctx context.Context, msg *schema.Message) error {
	if msg == nil {
		log.Warnf("attempted to add a nil message to chat history")
		return fmt.Errorf("message cannot be nil")
	}

	// Generate a compact MessageID with MD5 to stay within varchar(64).
	// Source format: {SessionID}-{Role}-{Timestamp}.
	rawMessageID := fmt.Sprintf("%s-%s-%d",
		l.clientState.SessionID,
		msg.Role,
		time.Now().UnixMilli())
	// Generate a fixed 32-character hexadecimal string.
	hash := md5.Sum([]byte(rawMessageID))
	messageID := hex.EncodeToString(hash[:])

	// Add to memory synchronously.
	l.clientState.AddMessage(msg)

	// Tool messages are saved directly because they have no audio.
	if msg.Role == schema.Tool {
		eventbus.Get().Publish(eventbus.TopicAddMessage, &eventbus.AddMessageEvent{
			ClientState: l.clientState,
			Msg:         *msg,
			MessageID:   messageID,
			AudioData:   nil, // Tool messages have no audio.
			AudioSize:   0,
			SampleRate:  0,
			Channels:    0,
			Timestamp:   time.Now(),
			IsUpdate:    false, // Single-phase save.
		})
		return nil
	}

	// User and assistant messages use two-phase persistence.
	// Store MessageID for the later audio update.
	if msg.Role == schema.User || msg.Role == schema.Assistant {
		l.lastMessageIDMu.Lock()
		l.lastMessageID[string(msg.Role)] = messageID
		l.lastMessageIDMu.Unlock()
	}

	// Publish phase one with text only.
	eventbus.Get().Publish(eventbus.TopicAddMessage, &eventbus.AddMessageEvent{
		ClientState: l.clientState,
		Msg:         *msg,
		MessageID:   messageID,
		AudioData:   nil, // Phase one has no audio.
		AudioSize:   0,
		SampleRate:  0,
		Channels:    0,
		Timestamp:   time.Now(),
		IsUpdate:    false, // Insert a new message.
	})

	return nil
}

// AddLlmMessage preserves backward compatibility by delegating to AddMessage.
func (l *LLMManager) AddLlmMessage(ctx context.Context, msg *schema.Message) error {
	return l.AddMessage(ctx, msg)
}

func (l *LLMManager) GetMessages(ctx context.Context, userMessage *schema.Message, count int, speakerResult *speaker.IdentifyResult) []*schema.Message {
	memoryMode := l.clientState.GetMemoryMode()
	includeHistory := memoryMode != MemoryModeNone

	// Load context from dialogue; none mode only carries temporary messages from the current tool chain.
	messageList := make([]*schema.Message, 0)
	if includeHistory {
		messageList = l.clientState.GetMessages(count)
		if userMessage != nil {
			messageList = trimTrailingUserMessages(messageList)
		}
	} else if toolRoundMessages := toolRoundMessagesFromContext(ctx); len(toolRoundMessages) > 0 {
		messageList = toolRoundMessages
	}

	// Build the system prompt.
	systemPrompt := l.clientState.SystemPrompt
	globalSystemPrompt := strings.TrimSpace(viper.GetString("chat.global_system_prompt"))
	if globalSystemPrompt != "" {
		if systemPrompt != "" {
			systemPrompt = globalSystemPrompt + "\n\n" + systemPrompt
		} else {
			systemPrompt = globalSystemPrompt
		}
	}

	// Add the current date and time.
	now := time.Now()
	systemPrompt += fmt.Sprintf("\nCurrent date and time: %s %s", now.Format("2006-01-02 15:04:05"), now.Format("Monday"))

	if memoryMode == MemoryModeLong && l.clientState.MemoryContext != "" {
		systemPrompt += fmt.Sprintf("\nUser personalization info: \n%s", l.clientState.MemoryContext)
	}

	log.Debugf("speakerResult: %+v, voiceIdentify: %+v", speakerResult, l.clientState.DeviceConfig.VoiceIdentify)

	// Merge speaker identification results into systemPrompt.
	if speakerResult != nil && speakerResult.Identified {
		// Match speakerResult to speakerGroup data in userConfig.
		if l.clientState.DeviceConfig.VoiceIdentify != nil {
			// Prefer SpeakerName because VoiceIdentify is keyed by speakerGroup.Name.
			if speakerGroupInfo, found := l.clientState.DeviceConfig.VoiceIdentify[speakerResult.SpeakerName]; found {
				// Merge the matching speakerGroup description into systemPrompt.
				if speakerGroupInfo.Prompt != "" {
					systemPrompt += fmt.Sprintf("\nSpeaker info from voice recognition: \n%s", speakerGroupInfo.Prompt)
				}
			}
		}
	}

	//search memory
	if memoryMode == MemoryModeLong && l.clientState.MemoryProvider != nil && userMessage != nil {
		memoryContext, err := l.clientState.MemoryProvider.Search(ctx, l.clientState.GetDeviceIDOrAgentID(), userMessage.Content, 10, 180)
		if err != nil {
			log.Errorf("memory search failed: %v", err)
		}
		log.Debugf("memory search succeeded, input: %s, memory: %s", userMessage.Content, memoryContext)
		if memoryContext != "" {
			systemPrompt += fmt.Sprintf("\nHistorical context: \n%s", memoryContext)
		}
	}

	systemPrompt += buildKnowledgeSearchRoutingPolicy(l.clientState.DeviceConfig.KnowledgeBases)

	retMessage := make([]*schema.Message, 0)
	retMessage = append(retMessage, &schema.Message{
		Role:    schema.System,
		Content: systemPrompt,
	})
	// Filter empty assistant messages to avoid LLM API 400 errors.
	// An assistant message with empty Content and ToolCalls is invalid.
	for _, msg := range messageList {
		if msg != nil && msg.Role == schema.Assistant && msg.Content == "" && len(msg.ToolCalls) == 0 {
			log.Debugf("filtered empty assistant message to avoid an LLM API error")
			continue
		}
		msgCopy := cloneMessageForRequest(msg)
		if isInterruptedMessage(msgCopy) {
			msgCopy.Content = decorateInterruptedContent(msgCopy.Content)
		}
		retMessage = append(retMessage, msgCopy)
	}
	if userMessage != nil {
		// Avoid adding a duplicate when retMessage already ends with the same user message.
		shouldAdd := true
		if len(retMessage) > 0 {
			lastMsg := retMessage[len(retMessage)-1]
			if lastMsg.Role == schema.User && lastMsg.Content == userMessage.Content {
				// The last message is already the same user message; skip it.
				shouldAdd = false
				//log.Debugf("last message is the same user message; skipping duplicate: %s", userMessage.Content)
			}
		}
		if shouldAdd {
			retMessage = append(retMessage, userMessage)
		}
	}
	return retMessage
}

func buildKnowledgeSearchRoutingPolicy(knowledgeBases []config_types.KnowledgeBaseRef) string {
	if len(knowledgeBases) == 0 {
		return ""
	}

	availableKBs := make([]string, 0, len(knowledgeBases))
	for _, kb := range knowledgeBases {
		if strings.EqualFold(strings.TrimSpace(kb.Status), "inactive") {
			continue
		}
		if strings.TrimSpace(kb.ExternalKBID) == "" {
			continue
		}
		name := strings.TrimSpace(kb.Name)
		if name == "" {
			name = strings.TrimSpace(kb.ExternalKBID)
		}
		if name == "" {
			continue
		}
		if kb.ID == 0 {
			continue
		}
		desc := strings.TrimSpace(kb.Description)
		if desc == "" {
			desc = "No description"
		}
		availableKBs = append(availableKBs, fmt.Sprintf("%d: Name=%s; Description=%s", kb.ID, name, desc))
		if len(availableKBs) >= 8 {
			break
		}
	}
	if len(availableKBs) == 0 {
		return ""
	}

	return fmt.Sprintf(
		“\nKnowledge base search rules (tool: search_knowledge):\nAvailable knowledge bases (id:name+description): %s\n”+
			“1. Trigger condition: user asks questions requiring factual basis, process rules, parameters, definitions, clauses, comparisons, or explicitly requests 'answer from knowledge base/document'.\n”+
			“2. Do not trigger: casual chat, emotional support, pure creative tasks, purely subjective suggestions.\n”+
			“3. Usage: call at most once per turn, query distills core keywords from user question, top_k defaults to 5; if a specific knowledge base can be identified, pass knowledge_base_ids (multiple allowed).\n”+
			“4. Selection rule: only pass knowledge base IDs most semantically relevant to the current question; if uncertain, omit knowledge_base_ids.\n”+
			“5. Insufficient info: if evidence is insufficient, do not fabricate - ask user for more specific keywords.\n”+
			“6. Output requirement: do not mention 'knowledge base', 'search', 'MCP', 'tool call', 'hit results' or any source/process information in the response.”,
		strings.Join(availableKBs, “, “),
	)
}

func trimTrailingUserMessages(messages []*schema.Message) []*schema.Message {
	end := len(messages)
	for end > 0 {
		msg := messages[end-1]
		if msg == nil || msg.Role != schema.User {
			break
		}
		end--
	}
	return messages[:end]
}

func isInterruptedMessage(msg *schema.Message) bool {
	if msg == nil || msg.Extra == nil {
		return false
	}
	v, ok := msg.Extra[interruptExtraKey]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	default:
		return false
	}
}

func decorateInterruptedContent(content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	if strings.HasSuffix(content, interruptContentSuffix) {
		return content
	}
	return content + interruptContentSuffix
}

func cloneMessagesForRequest(messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return nil
	}

	cloned := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		cloned = append(cloned, cloneMessageForRequest(msg))
	}

	return cloned
}

func toolRoundMessagesFromContext(ctx context.Context) []*schema.Message {
	if ctx == nil {
		return nil
	}

	messages, ok := ctx.Value(toolRoundMessagesKey).([]*schema.Message)
	if !ok || len(messages) == 0 {
		return nil
	}

	return cloneMessagesForRequest(messages)
}

func ttsTurnTrackerFromContext(ctx context.Context) *ttsTurnTracker {
	if ctx == nil {
		return nil
	}

	tracker, ok := ctx.Value(ttsTurnTrackerKey).(*ttsTurnTracker)
	if !ok {
		return nil
	}

	return tracker
}

func ensureTTSTurnTrackerInContext(ctx context.Context) context.Context {
	if ttsTurnTrackerFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, ttsTurnTrackerKey, newTTSTurnTracker())
}

func waitForTTSTurnDrainIfRoot(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if nest, ok := ctx.Value("nest").(int); ok && nest > 1 {
		return nil
	}

	tracker := ttsTurnTrackerFromContext(ctx)
	if tracker == nil {
		return nil
	}

	return tracker.Wait(ctx)
}

func appendToolRoundMessagesToContext(ctx context.Context, messages []*schema.Message) context.Context {
	if len(messages) == 0 {
		return ctx
	}

	combined := toolRoundMessagesFromContext(ctx)
	combined = append(combined, cloneMessagesForRequest(messages)...)
	if len(combined) == 0 {
		return ctx
	}

	return context.WithValue(ctx, toolRoundMessagesKey, combined)
}

func cloneMessageForRequest(msg *schema.Message) *schema.Message {
	if msg == nil {
		return nil
	}
	msgCopy := *msg

	if msg.ToolCalls != nil {
		msgCopy.ToolCalls = append([]schema.ToolCall(nil), msg.ToolCalls...)
	}
	if msg.MultiContent != nil {
		msgCopy.MultiContent = append([]schema.ChatMessagePart(nil), msg.MultiContent...)
	}
	if msg.Extra != nil {
		msgCopy.Extra = make(map[string]any, len(msg.Extra))
		for k, v := range msg.Extra {
			msgCopy.Extra[k] = v
		}
	}
	if msg.ResponseMeta != nil {
		respMetaCopy := *msg.ResponseMeta
		msgCopy.ResponseMeta = &respMetaCopy
	}

	return &msgCopy
}
