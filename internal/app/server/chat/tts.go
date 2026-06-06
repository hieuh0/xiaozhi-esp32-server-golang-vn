package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	. "xiaozhi-esp32-server-golang/internal/data/client"
	chathooks "xiaozhi-esp32-server-golang/internal/domain/chat/hooks"
	llm_common "xiaozhi-esp32-server-golang/internal/domain/llm/common"
	"xiaozhi-esp32-server-golang/internal/domain/tts"
	ttsstream "xiaozhi-esp32-server-golang/internal/domain/tts/streaming"
	"xiaozhi-esp32-server-golang/internal/pool"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

// Session-wide audio queue element kinds.
const (
	AudioQueueKindFrame         = 0
	AudioQueueKindSentenceStart = 1
	AudioQueueKindSentenceEnd   = 2
	AudioQueueKindTtsStart      = 3
	AudioQueueKindTtsStop       = 4
	AudioQueueKindMediaFrame    = 5
)

// AudioQueueElem is a session-wide queue element for TTS/media frames and control messages.
type AudioQueueElem struct {
	Kind        int    // AudioQueueKindFrame / MediaFrame / SentenceStart / SentenceEnd / TtsStart / TtsStop
	Data        []byte // Copied into the queue for Frame or MediaFrame.
	Text        string // Used for SentenceStart and SentenceEnd.
	Err         error  // Optional segment error for SentenceEnd.
	IsStart     bool   // Whether SentenceStart is the first packet, used for metrics.
	Generation  uint64 // Old generations are discarded after interruption.
	MetricCycle uint64 // Associates the first TTS audio frame with its metrics cycle.
	DebugReason string
	OnStart     func()
	OnEnd       func(error)
	OnError     func(error)
}

type delayedSentenceTask struct {
	Elem      AudioQueueElem
	ExecuteAt time.Time
}

type interruptRequest struct {
	done   chan struct{}
	reason string
}

// SessionAudioQueueCap absorbs prefetch without blocking the session audio queue.
const SessionAudioQueueCap = 150

type TTSQueueItem struct {
	ctx         context.Context
	llmResponse llm_common.LLMResponseStruct        // Used in single-item mode.
	StreamChan  <-chan llm_common.LLMResponseStruct // Preferred in streaming mode when non-nil.
	enqueueSeq  uint64
	generation  uint64
	metricCycle uint64
	onStartFunc func()
	onEndFunc   func(err error)
}

type ttsMetricState struct {
	cycleID          uint64
	pendingItems     int
	activeRequests   int
	turnEndRequested bool
	started          bool
	firstAudio       bool
	ttsStopped       bool
	turnEnded        bool
	turnEndPolicy    ttsTurnEndPolicy
}

// TTSManager handles TTS processing.

type TTSManagerOption func(*TTSManager)

type TTSManager struct {
	clientState               *ClientState
	session                   *ChatSession
	serverTransport           *ServerTransport
	ttsQueue                  *util.Queue[TTSQueueItem]
	sessionAudioQueue         chan AudioQueueElem // Session-wide queue for frames and control messages.
	delayedSentenceQueue      chan delayedSentenceTask
	delayedSentenceReadyQueue chan AudioQueueElem
	interruptCh               chan interruptRequest // Interrupt signal that drains the queue and resumes sender processing.
	audioGeneration           atomic.Uint64         // Incremented on interruption so the sender drops old elements.
	audioInterruptMu          sync.RWMutex
	audioInterruptCh          chan struct{}
	ttsActive                 atomic.Bool // Whether a TTS segment has started but not ended.
	senderLoopActive          atomic.Bool
	senderLoopDone            chan struct{} // Closed on sender exit so synchronous interruption returns quickly.

	ttsQueueSeq   atomic.Uint64
	droppedTTSSeq atomic.Uint64

	mediaPlaybackMu     sync.RWMutex
	mediaPlaybackActive bool
	mediaPlaybackWaitCh chan struct{}

	interruptStopMu          sync.Mutex
	interruptStopPending     bool
	interruptStopSendTtsStop bool
	interruptStopErr         error
	interruptStopReason      string

	// Chat history audio cache containing accumulated TTS Opus frames.
	audioHistoryBuffer [][]byte
	audioMutex         sync.Mutex

	// Internal dual-stream channel created on IsStart and closed on IsEnd.
	dualStreamChan     chan llm_common.LLMResponseStruct
	dualStreamDone     chan struct{} // onEndFunc signal used by synchronous dual-stream waits.
	dualStreamOwnerCtx context.Context
	dualStreamMu       sync.Mutex
	dualStreamEpoch    atomic.Uint64

	ttsMetricMu    sync.Mutex
	ttsMetricState ttsMetricState
}

// NewTTSManager creates a TTS manager for the supplied client state.
func NewTTSManager(clientState *ClientState, serverTransport *ServerTransport, session *ChatSession, opts ...TTSManagerOption) *TTSManager {
	t := &TTSManager{
		clientState:               clientState,
		session:                   session,
		serverTransport:           serverTransport,
		ttsQueue:                  util.NewQueue[TTSQueueItem](10),
		sessionAudioQueue:         make(chan AudioQueueElem, SessionAudioQueueCap),
		delayedSentenceQueue:      make(chan delayedSentenceTask, SessionAudioQueueCap),
		delayedSentenceReadyQueue: make(chan AudioQueueElem, SessionAudioQueueCap),
		interruptCh:               make(chan interruptRequest, 1),
		senderLoopDone:            make(chan struct{}),
		audioInterruptCh:          make(chan struct{}),
	}
	for _, opt := range opts {
		opt(t)
	}
	t.audioGeneration.Store(1)
	return t
}

func normalizeTTSReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "unspecified"
	}
	return reason
}

func (t *TTSManager) debugState() string {
	if t == nil || t.clientState == nil {
		return "tts_manager=nil"
	}
	return fmt.Sprintf(
		"status=%s phase=%s ttsActive=%v ttsStart=%v welcomeSpeaking=%v welcomePlaying=%v",
		t.clientState.GetStatus(),
		t.clientState.GetListenPhase(),
		t.ttsActive.Load(),
		t.clientState.GetTtsStart(),
		t.clientState.IsWelcomeSpeaking,
		t.clientState.IsWelcomePlaying,
	)
}

// Start launches the TTS queue consumer and the unified session audio sender.
func (t *TTSManager) Start(ctx context.Context) {
	go t.runDelayedSentenceLoop(ctx)
	go t.runSenderLoop(ctx)
	t.processTTSQueue(ctx)
}

// runSenderLoop is the sole sender. It dispatches queue elements by kind, owns flow control,
// exits only when ctx is canceled, and drains then continues on session cancellation or TurnAbort.
func (t *TTSManager) runSenderLoop(ctx context.Context) {
	t.senderLoopActive.Store(true)
	defer func() {
		t.senderLoopActive.Store(false)
		close(t.senderLoopDone)
	}()

	frameDuration := time.Duration(t.clientState.OutputAudioFormat.FrameDuration) * time.Millisecond
	cacheFrameCount := 120 / t.clientState.OutputAudioFormat.FrameDuration
	totalFrames := 0
	currentSentenceFrames := 0
	playbackTail := time.Time{}

	handleDelayedSentence := func(elem AudioQueueElem) {
		if elem.Generation != t.currentAudioGeneration() {
			if elem.OnEnd != nil {
				elem.OnEnd(context.Canceled)
			}
			return
		}
		switch elem.Kind {
		case AudioQueueKindSentenceStart:
			if elem.OnStart != nil {
				elem.OnStart()
			}
			if elem.Text != "" {
				if err := t.serverTransport.SendSentenceStart(elem.Text); err != nil {
					log.Errorf("failed to send TTS text: %s, %v", elem.Text, err)
					if elem.OnError != nil {
						elem.OnError(err)
					}
					if elem.OnEnd != nil {
						elem.OnEnd(err)
					}
				}
			}
		case AudioQueueKindSentenceEnd:
			callbackErr := elem.Err
			if elem.Text != "" {
				if err := t.serverTransport.SendSentenceEnd(elem.Text); err != nil {
					log.Errorf("failed to send TTS text: %s, %v", elem.Text, err)
					if elem.OnError != nil {
						elem.OnError(err)
					}
					if callbackErr == nil {
						callbackErr = err
					}
				}
			}
			currentSentenceFrames = 0
			if elem.OnEnd != nil {
				elem.OnEnd(callbackErr)
			}
		}
	}

	handleInterrupt := func(reason string) {
		t.drainSessionAudioQueue()
		t.drainDelayedSentenceReadyQueue()
		if pending, sendTtsStop, stopErr, stopReason := t.consumePendingInterruptStop(); pending {
			t.finishTtsStopWithReason(t.clientState.Ctx, sendTtsStop, stopErr, stopReason)
		}
		totalFrames = 0
		currentSentenceFrames = 0
		playbackTail = time.Time{}
		log.Infof("runSenderLoop interrupt, drained queue and continue: device=%s reason=%s state={%s}", t.clientState.DeviceID, normalizeTTSReason(reason), t.debugState())
	}

	for {
		select {
		case elem := <-t.delayedSentenceReadyQueue:
			handleDelayedSentence(elem)
			continue
		default:
		}

		select {
		case <-ctx.Done():
			t.drainSessionAudioQueue()
			t.drainDelayedSentenceReadyQueue()
			t.finishTtsStopWithReason(t.clientState.Ctx, true, ctx.Err(), "runSenderLoop: ctx done")
			log.Debugf("runSenderLoop ctx done, drained queue and exit")
			return
		case req := <-t.interruptCh:
			handleInterrupt(req.reason)
			if req.done != nil {
				close(req.done)
			}
			continue
		case elem := <-t.delayedSentenceReadyQueue:
			handleDelayedSentence(elem)
		case elem, ok := <-t.sessionAudioQueue:
			if !ok {
				return
			}
			if elem.Generation != t.currentAudioGeneration() {
				if elem.OnEnd != nil {
					elem.OnEnd(context.Canceled)
				}
				continue
			}
			switch elem.Kind {
			case AudioQueueKindSentenceStart:
				currentSentenceFrames = 0
				if !t.enqueueDelayedSentenceTask(ctx, elem) && elem.OnEnd != nil {
					elem.OnEnd(ctx.Err())
				}
			case AudioQueueKindFrame, AudioQueueKindMediaFrame:
				now := time.Now()
				if playbackTail.IsZero() || now.After(playbackTail) {
					playbackTail = now
				}
				allowedAhead := time.Duration(cacheFrameCount) * frameDuration
				sendAt := playbackTail.Add(-allowedAhead)
				if now.Before(sendAt) {
					waitResult, interruptReq := t.waitUntilSenderDeadline(ctx, sendAt, handleDelayedSentence)
					switch waitResult {
					case senderWaitContextDone:
						t.drainSessionAudioQueue()
						t.drainDelayedSentenceReadyQueue()
						t.finishTtsStopWithReason(t.clientState.Ctx, true, ctx.Err(), "runSenderLoop: frame send wait ctx done")
						return
					case senderWaitInterrupted:
						handleInterrupt(interruptReq.reason)
						if interruptReq.done != nil {
							close(interruptReq.done)
						}
						continue
					}
					now = time.Now()
					if now.After(playbackTail) {
						playbackTail = now
					}
				}
				if err := t.serverTransport.SendAudio(elem.Data); err != nil {
					audioType := "TTS"
					if elem.Kind == AudioQueueKindMediaFrame {
						audioType = "media"
					}
					log.Errorf("failed to send %s audio: len: %d, %v", audioType, len(elem.Data), err)
					if elem.OnError != nil {
						elem.OnError(err)
					}
					continue
				}
				if elem.Kind == AudioQueueKindFrame {
					t.markTtsMetricFirstAudio(t.clientState.Ctx, elem.MetricCycle)
					t.audioMutex.Lock()
					frameCopy := make([]byte, len(elem.Data))
					copy(frameCopy, elem.Data)
					t.audioHistoryBuffer = append(t.audioHistoryBuffer, frameCopy)
					t.audioMutex.Unlock()
				}
				totalFrames++
				currentSentenceFrames++
				playbackTail = playbackTail.Add(frameDuration)
			case AudioQueueKindSentenceEnd:
				if !t.enqueueDelayedSentenceTask(ctx, elem) && elem.OnEnd != nil {
					elem.OnEnd(ctx.Err())
				}
			case AudioQueueKindTtsStart:
				if t.session != nil {
					hookErr := t.session.hookHub.EmitTTSOutputStart(t.session.hookContext(ctx))
					if hookErr != nil {
						log.Warnf("TTS_OUTPUT_START hook failed: %v", hookErr)
					}
				}
				t.ttsActive.Store(true)
				if err := t.serverTransport.SendTtsStart(); err != nil {
					log.Errorf("failed to send TtsStart: %v", err)
				}
				// Reset frame count and playback tail for the new speech segment.
				totalFrames = 0
				playbackTail = time.Time{}
			case AudioQueueKindTtsStop:
				log.Infof("runSenderLoop processing tts stop: device=%s reason=%s state={%s}", t.clientState.DeviceID, normalizeTTSReason(elem.DebugReason), t.debugState())
				// Wait for the playback tail to reach the final frame before TtsStop.
				if !playbackTail.IsZero() {
					waitResult, interruptReq := t.waitUntilSenderDeadline(ctx, playbackTail, handleDelayedSentence)
					switch waitResult {
					case senderWaitContextDone:
						t.drainSessionAudioQueue()
						t.drainDelayedSentenceReadyQueue()
						t.finishTtsStopWithReason(t.clientState.Ctx, true, ctx.Err(), "runSenderLoop: tts stop wait playback tail ctx done")
						return
					case senderWaitInterrupted:
						handleInterrupt(interruptReq.reason)
						if interruptReq.done != nil {
							close(interruptReq.done)
						}
						continue
					}
				}
				// Allow a short playback grace period before turn-end finalization.
				waitResult, interruptReq := t.waitUntilSenderDeadline(ctx, time.Now().Add(ttsPlaybackCompletionGrace), handleDelayedSentence)
				switch waitResult {
				case senderWaitContextDone:
					t.drainSessionAudioQueue()
					t.drainDelayedSentenceReadyQueue()
					t.finishTtsStopWithReason(t.clientState.Ctx, true, ctx.Err(), "runSenderLoop: tts stop grace wait ctx done")
					return
				case senderWaitInterrupted:
					handleInterrupt(interruptReq.reason)
					if interruptReq.done != nil {
						close(interruptReq.done)
					}
					continue
				}
				t.finishTtsStopWithReason(withTTSTurnPlaybackSettled(t.clientState.Ctx), true, nil, elem.DebugReason)
				playbackTail = time.Time{}
				totalFrames = 0
				currentSentenceFrames = 0
			}
		}
	}
}

// drainSessionAudioQueue discards unsent elements after context cancellation.
func (t *TTSManager) drainSessionAudioQueue() {
	for {
		select {
		case elem, ok := <-t.sessionAudioQueue:
			if !ok {
				return
			}
			if elem.OnEnd != nil {
				elem.OnEnd(context.Canceled)
			}
		default:
			return
		}
	}
}

func (t *TTSManager) drainDelayedSentenceReadyQueue() {
	for {
		select {
		case elem := <-t.delayedSentenceReadyQueue:
			if elem.OnEnd != nil {
				elem.OnEnd(context.Canceled)
			}
		default:
			return
		}
	}
}

// ClearSessionAudioQueue drains the session audio queue.
func (t *TTSManager) ClearSessionAudioQueue() {
	t.drainSessionAudioQueue()
}

func (t *TTSManager) currentAudioGeneration() uint64 {
	return t.audioGeneration.Load()
}

func (t *TTSManager) nextAudioGeneration() uint64 {
	return t.audioGeneration.Add(1)
}

func (t *TTSManager) currentAudioInterruptCh() <-chan struct{} {
	t.audioInterruptMu.RLock()
	defer t.audioInterruptMu.RUnlock()
	return t.audioInterruptCh
}

func (t *TTSManager) rotateAudioInterruptCh() {
	t.audioInterruptMu.Lock()
	defer t.audioInterruptMu.Unlock()

	oldCh := t.audioInterruptCh
	t.audioInterruptCh = make(chan struct{})
	if oldCh == nil {
		return
	}

	select {
	case <-oldCh:
	default:
		close(oldCh)
	}
}

func (t *TTSManager) withAudioInterruptContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}

	interruptCtx, cancel := context.WithCancel(ctx)
	interruptCh := t.currentAudioInterruptCh()
	if interruptCh == nil {
		return interruptCtx, cancel
	}

	go func() {
		select {
		case <-interruptCtx.Done():
		case <-interruptCh:
			cancel()
		}
	}()

	return interruptCtx, cancel
}

func (t *TTSManager) startTtsMetricCycle() uint64 {
	t.ttsMetricMu.Lock()
	defer t.ttsMetricMu.Unlock()

	t.ttsMetricState = ttsMetricState{
		cycleID: t.ttsMetricState.cycleID + 1,
	}
	return t.ttsMetricState.cycleID
}

func (t *TTSManager) currentTtsMetricCycle() uint64 {
	t.ttsMetricMu.Lock()
	defer t.ttsMetricMu.Unlock()
	return t.ttsMetricState.cycleID
}

func (t *TTSManager) registerTTSTurnEndPolicy(ctx context.Context, cycleID uint64) {
	if cycleID == 0 {
		return
	}
	policy := ttsTurnEndPolicyFromContext(ctx)
	if policy == ttsTurnEndPolicyNone {
		return
	}

	t.ttsMetricMu.Lock()
	defer t.ttsMetricMu.Unlock()

	if t.ttsMetricState.cycleID != cycleID || t.ttsMetricState.turnEnded {
		return
	}
	t.ttsMetricState.turnEndPolicy = policy
}

func (t *TTSManager) consumeTTSTurnEndPolicy() ttsTurnEndPolicy {
	t.ttsMetricMu.Lock()
	defer t.ttsMetricMu.Unlock()

	policy := t.ttsMetricState.turnEndPolicy
	t.ttsMetricState.turnEndPolicy = ttsTurnEndPolicyNone
	return policy
}

func (t *TTSManager) dispatchTTSTurnEndPolicy(ctx context.Context, stopErr error) {
	policy := t.consumeTTSTurnEndPolicy()
	if policy == ttsTurnEndPolicyNone {
		return
	}

	handler := ttsTurnEndPolicyHandlerFromContext(ctx)
	if handler == nil && t.clientState != nil {
		handler = ttsTurnEndPolicyHandlerFromContext(t.clientState.Ctx)
	}
	if handler == nil {
		log.Debugf("TTS turn end policy dropped: no handler, policy=%d", policy)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("TTS turn end policy handler panic: %v", r)
			}
		}()

		// A synchronous session reset here can re-enter through stopSpeaking or Interrupt.
		// Dispatch asynchronously to ChatManager to avoid trapping runSenderLoop.
		handler.handleTTSTurnEndPolicy(ctx, policy, stopErr)
	}()
}

type ttsMetricCompletion struct {
	err          error
	ttsStopTs    int64
	turnEndTs    int64
	traceTtsStop bool
	traceTurnEnd bool
}

func (t *TTSManager) emitTtsMetricCompletion(ctx context.Context, completion ttsMetricCompletion) {
	if t.session == nil {
		return
	}
	if completion.traceTtsStop {
		t.session.TraceTtsStop(ctx, completion.ttsStopTs, completion.err)
	}
	if completion.traceTurnEnd {
		t.session.TraceTurnEnd(ctx, completion.turnEndTs, completion.err)
	}
}

func (t *TTSManager) registerTtsMetricItem(cycleID uint64) {
	if cycleID == 0 {
		return
	}

	t.ttsMetricMu.Lock()
	defer t.ttsMetricMu.Unlock()

	if t.ttsMetricState.cycleID != cycleID || t.ttsMetricState.turnEnded {
		return
	}
	t.ttsMetricState.pendingItems++
}

func (t *TTSManager) finishTtsMetricItem(ctx context.Context, cycleID uint64, err error) {
	t.emitTtsMetricCompletion(ctx, t.finishTtsMetricItemLocked(cycleID, err))
}

func (t *TTSManager) finishTtsMetricItemLocked(cycleID uint64, err error) ttsMetricCompletion {
	t.ttsMetricMu.Lock()
	defer t.ttsMetricMu.Unlock()

	if t.ttsMetricState.cycleID != cycleID || cycleID == 0 {
		return ttsMetricCompletion{}
	}
	if t.ttsMetricState.pendingItems > 0 {
		t.ttsMetricState.pendingItems--
	}
	return t.maybeFinalizeTtsMetricLocked(err)
}

func (t *TTSManager) markTtsMetricRequestStart(ctx context.Context, cycleID uint64) {
	if cycleID == 0 {
		return
	}

	var startTs int64

	t.ttsMetricMu.Lock()
	if t.ttsMetricState.cycleID == cycleID && !t.ttsMetricState.turnEnded {
		t.ttsMetricState.activeRequests++
		if !t.ttsMetricState.started {
			t.clientState.MarkTtsStart()
			startTs = t.clientState.Statistic.TtsStartTs
			t.ttsMetricState.started = true
		}
	}
	t.ttsMetricMu.Unlock()

	if startTs > 0 && t.session != nil {
		t.session.TraceTtsStart(ctx, startTs)
	}
}

func (t *TTSManager) markTtsMetricFirstAudio(ctx context.Context, cycleID uint64) {
	if cycleID == 0 {
		return
	}

	var firstAudioTs int64

	t.ttsMetricMu.Lock()
	if t.ttsMetricState.cycleID == cycleID && t.ttsMetricState.started && !t.ttsMetricState.firstAudio && !t.ttsMetricState.turnEnded {
		t.clientState.MarkTtsFirstFrame()
		firstAudioTs = t.clientState.Statistic.TtsFirstFrameTs
		t.ttsMetricState.firstAudio = true
	}
	t.ttsMetricMu.Unlock()

	if firstAudioTs > 0 && t.session != nil {
		t.session.TraceTtsFirstFrame(ctx, firstAudioTs)
	}
}

func (t *TTSManager) finishTtsMetricRequest(ctx context.Context, cycleID uint64, err error) {
	t.emitTtsMetricCompletion(ctx, t.finishTtsMetricRequestLocked(cycleID, err))
}

func (t *TTSManager) finishTtsMetricRequestLocked(cycleID uint64, err error) ttsMetricCompletion {
	t.ttsMetricMu.Lock()
	defer t.ttsMetricMu.Unlock()

	if t.ttsMetricState.cycleID != cycleID || cycleID == 0 {
		return ttsMetricCompletion{}
	}
	if t.ttsMetricState.activeRequests > 0 {
		t.ttsMetricState.activeRequests--
	}
	return t.maybeFinalizeTtsMetricLocked(err)
}

func (t *TTSManager) requestTurnEndLocked(err error) ttsMetricCompletion {
	t.ttsMetricMu.Lock()
	defer t.ttsMetricMu.Unlock()

	if t.ttsMetricState.cycleID == 0 {
		return ttsMetricCompletion{}
	}
	t.ttsMetricState.turnEndRequested = true
	return t.maybeFinalizeTtsMetricLocked(err)
}

func (t *TTSManager) forceStopTtsMetric(ctx context.Context, err error) {
	t.emitTtsMetricCompletion(ctx, t.forceStopTtsMetricLocked(err))
}

func (t *TTSManager) forceStopTtsMetricLocked(err error) ttsMetricCompletion {
	t.ttsMetricMu.Lock()
	defer t.ttsMetricMu.Unlock()

	if t.ttsMetricState.cycleID == 0 || t.ttsMetricState.turnEnded {
		return ttsMetricCompletion{}
	}
	t.ttsMetricState.turnEndRequested = true
	return t.finalizeTtsMetricLocked(err)
}

func (t *TTSManager) maybeFinalizeTtsMetricLocked(err error) ttsMetricCompletion {
	if !t.ttsMetricState.turnEndRequested || t.ttsMetricState.turnEnded {
		return ttsMetricCompletion{}
	}
	if t.ttsMetricState.pendingItems > 0 || t.ttsMetricState.activeRequests > 0 {
		return ttsMetricCompletion{}
	}
	return t.finalizeTtsMetricLocked(err)
}

func (t *TTSManager) finalizeTtsMetricLocked(err error) ttsMetricCompletion {
	if t.ttsMetricState.turnEnded {
		return ttsMetricCompletion{}
	}

	completion := ttsMetricCompletion{
		err:          err,
		traceTurnEnd: true,
	}
	if t.ttsMetricState.started && !t.ttsMetricState.ttsStopped {
		t.clientState.MarkTtsStop()
		completion.ttsStopTs = t.clientState.Statistic.TtsStopTs
		completion.turnEndTs = completion.ttsStopTs
		completion.traceTtsStop = true
		t.ttsMetricState.ttsStopped = true
	} else {
		completion.turnEndTs = time.Now().UnixMilli()
	}
	t.ttsMetricState.turnEnded = true
	return completion
}

func (t *TTSManager) pushTTSQueueItem(item TTSQueueItem) error {
	item.enqueueSeq = t.ttsQueueSeq.Add(1)
	if err := t.ttsQueue.Push(item); err != nil {
		return err
	}
	t.registerTtsMetricItem(item.metricCycle)
	return nil
}

func (t *TTSManager) shouldDropTTSQueueItem(item TTSQueueItem) bool {
	if item.enqueueSeq == 0 {
		return false
	}
	return item.enqueueSeq <= t.droppedTTSSeq.Load()
}

func (t *TTSManager) dismissTTSQueueItem(item TTSQueueItem, err error) {
	if item.onEndFunc != nil {
		item.onEndFunc(err)
	}
	t.finishTtsMetricItem(item.ctx, item.metricCycle, err)
}

func (t *TTSManager) waitForMediaPlaybackRelease(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		t.mediaPlaybackMu.RLock()
		active := t.mediaPlaybackActive
		waitCh := t.mediaPlaybackWaitCh
		t.mediaPlaybackMu.RUnlock()

		if !active || waitCh == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-waitCh:
		}
	}
}

func (t *TTSManager) BeginExclusiveMediaPlayback(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	waitCh := make(chan struct{})

	t.mediaPlaybackMu.Lock()
	if t.mediaPlaybackActive {
		t.mediaPlaybackMu.Unlock()
		return fmt.Errorf("media playback is already in exclusive mode")
	}
	t.mediaPlaybackActive = true
	t.mediaPlaybackWaitCh = waitCh
	t.mediaPlaybackMu.Unlock()

	t.ClearTTSQueue()
	// Media takeover interrupts and clears current TTS without immediately sending tts_stop.
	// The outer response sends tts_stop during unified finalization after playback completes.
	if err := t.InterruptAndClearQueueSync(ctx); err != nil {
		t.EndExclusiveMediaPlayback()
		return err
	}

	return nil
}

func (t *TTSManager) EndExclusiveMediaPlayback() {
	t.mediaPlaybackMu.Lock()
	waitCh := t.mediaPlaybackWaitCh
	t.mediaPlaybackActive = false
	t.mediaPlaybackWaitCh = nil
	t.mediaPlaybackMu.Unlock()

	if waitCh == nil {
		return
	}

	select {
	case <-waitCh:
	default:
		close(waitCh)
	}
}

func (t *TTSManager) sentenceControlDelay() time.Duration {
	frameDurationMs := t.clientState.OutputAudioFormat.FrameDuration
	if frameDurationMs <= 0 {
		return 0
	}
	cacheFrameCount := 120 / frameDurationMs
	return time.Duration(cacheFrameCount*frameDurationMs) * time.Millisecond
}

func insertDelayedSentenceTask(tasks []delayedSentenceTask, task delayedSentenceTask) []delayedSentenceTask {
	insertAt := len(tasks)
	for insertAt > 0 && task.ExecuteAt.Before(tasks[insertAt-1].ExecuteAt) {
		insertAt--
	}
	tasks = append(tasks, delayedSentenceTask{})
	copy(tasks[insertAt+1:], tasks[insertAt:])
	tasks[insertAt] = task
	return tasks
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func (t *TTSManager) enqueueDelayedSentenceTask(ctx context.Context, elem AudioQueueElem) bool {
	task := delayedSentenceTask{
		Elem:      elem,
		ExecuteAt: time.Now().Add(t.sentenceControlDelay()),
	}
	if ctx == nil {
		t.delayedSentenceQueue <- task
		return true
	}
	if ctx.Err() != nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case t.delayedSentenceQueue <- task:
		return true
	}
}

func (t *TTSManager) runDelayedSentenceLoop(ctx context.Context) {
	var (
		pending []delayedSentenceTask
		timer   *time.Timer
		timerCh <-chan time.Time
	)

	resetTimer := func() {
		stopTimer(timer)
		timer = nil
		timerCh = nil
		if len(pending) == 0 {
			return
		}
		waitDuration := time.Until(pending[0].ExecuteAt)
		if waitDuration < 0 {
			waitDuration = 0
		}
		timer = time.NewTimer(waitDuration)
		timerCh = timer.C
	}

	for {
		select {
		case <-ctx.Done():
			stopTimer(timer)
			return
		case task := <-t.delayedSentenceQueue:
			pending = insertDelayedSentenceTask(pending, task)
			resetTimer()
		case <-timerCh:
			timer = nil
			timerCh = nil
			if len(pending) == 0 {
				continue
			}
			task := pending[0]
			pending = pending[1:]
			if task.Elem.Generation != t.currentAudioGeneration() {
				if task.Elem.OnEnd != nil {
					task.Elem.OnEnd(context.Canceled)
				}
				resetTimer()
				continue
			}
			select {
			case <-ctx.Done():
				if task.Elem.OnEnd != nil {
					task.Elem.OnEnd(ctx.Err())
				}
				return
			case t.delayedSentenceReadyQueue <- task.Elem:
			}
			resetTimer()
		}
	}
}

type senderWaitResult int

const (
	senderWaitReached senderWaitResult = iota
	senderWaitContextDone
	senderWaitInterrupted
)

func (t *TTSManager) waitUntilSenderDeadline(ctx context.Context, deadline time.Time, handleDelayed func(AudioQueueElem)) (senderWaitResult, interruptRequest) {
	for {
		now := time.Now()
		if !now.Before(deadline) {
			return senderWaitReached, interruptRequest{}
		}

		timer := time.NewTimer(deadline.Sub(now))
		select {
		case <-ctx.Done():
			stopTimer(timer)
			return senderWaitContextDone, interruptRequest{}
		case req := <-t.interruptCh:
			stopTimer(timer)
			return senderWaitInterrupted, req
		case elem := <-t.delayedSentenceReadyQueue:
			stopTimer(timer)
			handleDelayed(elem)
		case <-timer.C:
			return senderWaitReached, interruptRequest{}
		}
	}
}

func (t *TTSManager) enqueueSessionElem(ctx context.Context, generation uint64, elem AudioQueueElem) bool {
	elem.Generation = generation
	if ctx == nil {
		t.sessionAudioQueue <- elem
		return true
	}
	if ctx.Err() != nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case t.sessionAudioQueue <- elem:
		return true
	}
}

// InterruptAndClearQueue asynchronously asks runSenderLoop to drain the queue and continue.
func (t *TTSManager) InterruptAndClearQueue() {
	t.InterruptAndClearQueueWithReason("InterruptAndClearQueue")
}

func (t *TTSManager) InterruptAndClearQueueWithReason(reason string) {
	reason = normalizeTTSReason(reason)
	t.nextAudioGeneration()
	t.rotateAudioInterruptCh()
	if !t.senderLoopActive.Load() {
		return
	}
	select {
	case t.interruptCh <- interruptRequest{reason: reason}:
	default:
	}
}

// InterruptAndStop records an immediate TTS stop request.
// runSenderLoop emits the stop and finalizes metrics after draining the queue.
func (t *TTSManager) InterruptAndStop(ctx context.Context, sendTtsStop bool, stopErr error) {
	t.InterruptAndStopWithReason(ctx, sendTtsStop, stopErr, "InterruptAndStop")
}

func (t *TTSManager) InterruptAndStopWithReason(ctx context.Context, sendTtsStop bool, stopErr error, reason string) {
	reason = normalizeTTSReason(reason)
	t.recordPendingInterruptStop(sendTtsStop, stopErr, reason)
	t.InterruptAndClearQueueWithReason(reason)
	t.finishPendingInterruptStopIfSenderLoopExited(ctx)
}

// InterruptAndStopSync synchronously interrupts while keeping finalization in runSenderLoop.
func (t *TTSManager) InterruptAndStopSync(ctx context.Context, sendTtsStop bool, stopErr error) error {
	return t.InterruptAndStopSyncWithReason(ctx, sendTtsStop, stopErr, "InterruptAndStopSync")
}

func (t *TTSManager) InterruptAndStopSyncWithReason(ctx context.Context, sendTtsStop bool, stopErr error, reason string) error {
	reason = normalizeTTSReason(reason)
	t.recordPendingInterruptStop(sendTtsStop, stopErr, reason)
	if err := t.InterruptAndClearQueueSyncWithReason(ctx, reason); err != nil {
		t.finishPendingInterruptStopIfSenderLoopExited(ctx)
		return err
	}
	t.finishPendingInterruptStopIfSenderLoopExited(ctx)
	return nil
}

func (t *TTSManager) recordPendingInterruptStop(sendTtsStop bool, stopErr error, reason string) {
	t.interruptStopMu.Lock()
	defer t.interruptStopMu.Unlock()

	reason = normalizeTTSReason(reason)
	if t.interruptStopPending {
		t.interruptStopSendTtsStop = t.interruptStopSendTtsStop || sendTtsStop
		if t.interruptStopErr == nil {
			t.interruptStopErr = stopErr
		}
		if t.interruptStopReason == "" || t.interruptStopReason == "unspecified" {
			t.interruptStopReason = reason
		}
	} else {
		t.interruptStopPending = true
		t.interruptStopSendTtsStop = sendTtsStop
		t.interruptStopErr = stopErr
		t.interruptStopReason = reason
	}
	log.Infof("record pending tts interrupt stop: device=%s reason=%s sendTtsStop=%v stopErr=%v state={%s}", t.clientState.DeviceID, reason, sendTtsStop, stopErr, t.debugState())
}

func (t *TTSManager) consumePendingInterruptStop() (bool, bool, error, string) {
	t.interruptStopMu.Lock()
	defer t.interruptStopMu.Unlock()

	if !t.interruptStopPending {
		return false, false, nil, ""
	}

	sendTtsStop := t.interruptStopSendTtsStop
	stopErr := t.interruptStopErr
	reason := normalizeTTSReason(t.interruptStopReason)
	t.interruptStopPending = false
	t.interruptStopSendTtsStop = false
	t.interruptStopErr = nil
	t.interruptStopReason = ""
	return true, sendTtsStop, stopErr, reason
}

func (t *TTSManager) finishPendingInterruptStopIfSenderLoopExited(ctx context.Context) {
	if t.senderLoopActive.Load() {
		return
	}
	if pending, sendTtsStop, stopErr, reason := t.consumePendingInterruptStop(); pending {
		t.finishTtsStopWithReason(ctx, sendTtsStop, stopErr, reason)
	}
}

// InterruptAndClearQueueSync waits for runSenderLoop to finish draining the queue.
func (t *TTSManager) InterruptAndClearQueueSync(ctx context.Context) error {
	return t.InterruptAndClearQueueSyncWithReason(ctx, "InterruptAndClearQueueSync")
}

func (t *TTSManager) InterruptAndClearQueueSyncWithReason(ctx context.Context, reason string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	reason = normalizeTTSReason(reason)
	t.nextAudioGeneration()
	t.rotateAudioInterruptCh()
	if !t.senderLoopActive.Load() {
		return nil
	}

	req := interruptRequest{
		done:   make(chan struct{}),
		reason: reason,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.senderLoopDone:
		return nil
	case t.interruptCh <- req:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.senderLoopDone:
		return nil
	case <-req.done:
		return nil
	}
}

func (t *TTSManager) finishTtsStop(ctx context.Context, sendTtsStop bool, stopErr error) bool {
	return t.finishTtsStopWithReason(ctx, sendTtsStop, stopErr, "finishTtsStop")
}

func (t *TTSManager) finishTtsStopWithReason(ctx context.Context, sendTtsStop bool, stopErr error, reason string) bool {
	reason = normalizeTTSReason(reason)
	log.Infof("finish tts stop: device=%s reason=%s sendTtsStop=%v stopErr=%v state={%s}", t.clientState.DeviceID, reason, sendTtsStop, stopErr, t.debugState())
	if !t.ttsActive.Swap(false) {
		if t.clientState != nil {
			if sendTtsStop && t.clientState.IsRealTime() {
				t.clientState.SetStatus(ClientStatusListenStop)
				t.clientState.SetTtsStart(false)
			}
			if t.clientState.UsesAudioIdleClock() {
				t.clientState.StartAudioIdleWindow(time.Now())
			}
		}
		return false
	}

	if ctx == nil {
		ctx = context.Background()
	}

	sentTtsStop := false
	if sendTtsStop {
		if stopErr != nil && t.serverTransport != nil {
			t.serverTransport.DrainPendingAudio()
		}
		if err := t.serverTransport.SendTtsStop(); err != nil {
			if stopErr == nil {
				stopErr = err
			}
			log.Errorf("failed to send TtsStop: %v", err)
		} else {
			sentTtsStop = true
		}
	}
	if !sentTtsStop && t.clientState != nil {
		t.clientState.SetStatus(ClientStatusListenStop)
		t.clientState.SetTtsStart(false)
	}
	if t.clientState != nil && t.clientState.UsesAudioIdleClock() {
		t.clientState.StartAudioIdleWindow(time.Now())
	}
	if t.session != nil {
		t.session.completeWelcomePlaybackWait(stopErr == nil)
	}
	if t.session != nil {
		hookErr := t.session.hookHub.EmitTTSOutputStop(t.session.hookContext(ctx), chathooks.TTSOutputStopData{Err: stopErr})
		if hookErr != nil {
			log.Warnf("TTS_OUTPUT_STOP hook failed: %v", hookErr)
		}
	}

	t.forceStopTtsMetric(ctx, stopErr)
	t.dispatchTTSTurnEndPolicy(ctx, stopErr)

	return true
}

func (t *TTSManager) FinishTtsWithoutProtocolStop(ctx context.Context, stopErr error) bool {
	return t.finishTtsStop(ctx, false, stopErr)
}

// EnqueueTtsStart queues TtsStart for runSenderLoop, blocking until queued or ctx.Done.
func (t *TTSManager) EnqueueTtsStart(ctx context.Context) {
	t.EnqueueTtsStartWithReason(ctx, "EnqueueTtsStart")
}

func (t *TTSManager) EnqueueTtsStartWithReason(ctx context.Context, reason string) {
	cycleID := t.startTtsMetricCycle()
	t.registerTTSTurnEndPolicy(ctx, cycleID)
	reason = normalizeTTSReason(reason)
	log.Debugf("enqueue tts start: device=%s reason=%s generation=%d state={%s}", t.clientState.DeviceID, reason, t.currentAudioGeneration(), t.debugState())
	t.enqueueSessionElem(ctx, t.currentAudioGeneration(), AudioQueueElem{Kind: AudioQueueKindTtsStart, DebugReason: reason})
}

// RequestTurnEnd marks logical output complete; turn_end is emitted after all TTS audio.
func (t *TTSManager) RequestTurnEnd(ctx context.Context, err error) {
	t.emitTtsMetricCompletion(ctx, t.requestTurnEndLocked(err))
}

// EnqueueTtsStop queues TtsStop for runSenderLoop, blocking until queued or ctx.Done.
func (t *TTSManager) EnqueueTtsStop(ctx context.Context) {
	t.EnqueueTtsStopWithReason(ctx, "EnqueueTtsStop")
}

func (t *TTSManager) EnqueueTtsStopWithReason(ctx context.Context, reason string) {
	reason = normalizeTTSReason(reason)
	log.Infof("enqueue tts stop: device=%s reason=%s generation=%d state={%s}", t.clientState.DeviceID, reason, t.currentAudioGeneration(), t.debugState())
	t.enqueueSessionElem(ctx, t.currentAudioGeneration(), AudioQueueElem{Kind: AudioQueueKindTtsStop, DebugReason: reason})
}

func (t *TTSManager) enqueueSessionElemWithError(ctx context.Context, generation uint64, elem AudioQueueElem) error {
	if t.enqueueSessionElem(ctx, generation, elem) {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return context.Canceled
}

func (t *TTSManager) EnqueueMediaSentenceStart(ctx context.Context, text string, onError func(error)) error {
	return t.enqueueSessionElemWithError(ctx, t.currentAudioGeneration(), AudioQueueElem{
		Kind:    AudioQueueKindSentenceStart,
		Text:    text,
		OnError: onError,
	})
}

func (t *TTSManager) EnqueueMediaSentenceEnd(ctx context.Context, text string, onError func(error), onEnd func(error)) error {
	err := t.enqueueSessionElemWithError(ctx, t.currentAudioGeneration(), AudioQueueElem{
		Kind:    AudioQueueKindSentenceEnd,
		Text:    text,
		OnEnd:   onEnd,
		OnError: onError,
	})
	if err != nil && onEnd != nil {
		onEnd(err)
	}
	return err
}

func (t *TTSManager) EnqueueMediaFrame(ctx context.Context, frame []byte, onError func(error)) error {
	frameCopy := make([]byte, len(frame))
	copy(frameCopy, frame)
	return t.enqueueSessionElemWithError(ctx, t.currentAudioGeneration(), AudioQueueElem{
		Kind:    AudioQueueKindMediaFrame,
		Data:    frameCopy,
		OnError: onError,
	})
}

func (t *TTSManager) processTTSQueue(ctx context.Context) {
	for {
		item, err := t.ttsQueue.Pop(ctx, 0) // Blocking.
		if err != nil {
			if err == util.ErrQueueCtxDone {
				return
			}
			continue
		}

		if t.shouldDropTTSQueueItem(item) {
			t.dismissTTSQueueItem(item, context.Canceled)
			continue
		}

		itemCtx, cancel := t.withAudioInterruptContext(item.ctx)
		item.ctx = itemCtx
		waitErr := t.waitForMediaPlaybackRelease(item.ctx)
		if waitErr != nil {
			cancel()
			t.dismissTTSQueueItem(item, waitErr)
			continue
		}
		if t.shouldDropTTSQueueItem(item) {
			cancel()
			t.dismissTTSQueueItem(item, context.Canceled)
			continue
		}

		itemErr := error(nil)
		if item.StreamChan != nil {
			log.Debugf("processTTSQueue start, stream mode")
			itemErr = t.handleStreamTts(item)
			t.finishTtsMetricItem(item.ctx, item.metricCycle, itemErr)
			cancel()
			log.Debugf("processTTSQueue end, stream mode")
			continue
		}

		// Non-streaming: handleTts emits SentenceStart, frames, then SentenceEnd.
		log.Debugf("processTTSQueue start, text: %s", item.llmResponse.Text)
		itemErr = t.handleTts(item.ctx, item.generation, item.metricCycle, item.llmResponse, item.onStartFunc, item.onEndFunc)
		t.finishTtsMetricItem(item.ctx, item.metricCycle, itemErr)
		cancel()
		log.Debugf("processTTSQueue end, text: %s (pushed)", item.llmResponse.Text)
	}
}

func (t *TTSManager) ClearTTSQueue() {
	t.droppedTTSSeq.Store(t.ttsQueueSeq.Load())
	t.dualStreamEpoch.Add(1)

	t.dualStreamMu.Lock()
	dualStreamChan := t.dualStreamChan
	t.dualStreamChan = nil
	t.dualStreamDone = nil
	t.dualStreamOwnerCtx = nil
	t.dualStreamMu.Unlock()
	safeCloseLLMResponseStream(dualStreamChan)

	drained := t.ttsQueue.ClearAndDrain()
	for _, item := range drained {
		t.dismissTTSQueueItem(item, context.Canceled)
	}
}

// handleTts generates one TTS item and queues SentenceStart, frames, then SentenceEnd.
func (t *TTSManager) handleTts(ctx context.Context, generation uint64, metricCycle uint64, llmResponse llm_common.LLMResponseStruct, onStartFunc func(), onEndFunc func(error)) error {
	if strings.TrimSpace(llmResponse.Text) == "" {
		if onEndFunc != nil {
			onEndFunc(nil)
		}
		return nil
	}
	outChan, release, genErr := t.generateTtsOnly(ctx, metricCycle, llmResponse)
	if genErr != nil {
		log.Errorf("handleTts gen err, text: %s, err: %v", llmResponse.Text, genErr)
		if onEndFunc != nil {
			onEndFunc(genErr)
		}
		return genErr
	}
	if outChan == nil {
		if release != nil {
			release()
		}
		if onEndFunc != nil {
			onEndFunc(nil)
		}
		return nil
	}
	requestActive := true
	firstAudioReported := false
	finishRequest := func(err error) {
		if requestActive {
			t.finishTtsMetricRequest(ctx, metricCycle, err)
			requestActive = false
		}
	}
	if !t.enqueueSessionElem(ctx, generation, AudioQueueElem{
		Kind:    AudioQueueKindSentenceStart,
		Text:    llmResponse.Text,
		IsStart: llmResponse.IsStart,
		OnStart: onStartFunc,
	}) {
		if release != nil {
			release()
		}
		finishRequest(ctx.Err())
		if onEndFunc != nil {
			onEndFunc(ctx.Err())
		}
		return ctx.Err()
	}
	for {
		select {
		case <-ctx.Done():
			if release != nil {
				release()
			}
			finishRequest(ctx.Err())
			if onEndFunc != nil {
				onEndFunc(ctx.Err())
			}
			return ctx.Err()
		case frame, ok := <-outChan:
			if !ok {
				if release != nil {
					release()
				}
				finishRequest(nil)
				if !t.enqueueSessionElem(ctx, generation, AudioQueueElem{
					Kind:  AudioQueueKindSentenceEnd,
					Text:  llmResponse.Text,
					OnEnd: onEndFunc,
				}) && onEndFunc != nil {
					onEndFunc(ctx.Err())
				}
				return ctx.Err()
			}
			if !firstAudioReported {
				t.markTtsMetricFirstAudio(ctx, metricCycle)
				firstAudioReported = true
			}
			frameCopy := make([]byte, len(frame))
			copy(frameCopy, frame)
			if !t.enqueueSessionElem(ctx, generation, AudioQueueElem{Kind: AudioQueueKindFrame, Data: frameCopy, MetricCycle: metricCycle}) {
				if release != nil {
					release()
				}
				finishRequest(ctx.Err())
				if onEndFunc != nil {
					onEndFunc(ctx.Err())
				}
				return ctx.Err()
			}
		}
	}
}

const ttsSyncWaitTimeout = 30 * time.Second

// signalDone sends one completion signal to the buffered channel.
func signalDone(done chan<- struct{}) {
	select {
	case done <- struct{}{}:
	default:
	}
}

func safeCloseLLMResponseStream(ch chan llm_common.LLMResponseStruct) {
	if ch == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	close(ch)
}

func sendLLMResponseToDualStream(ctx context.Context, ch chan llm_common.LLMResponseStruct, llmResponse llm_common.LLMResponseStruct) (err error) {
	if ch == nil {
		return nil
	}

	defer func() {
		if recover() != nil {
			err = nil
		}
	}()

	select {
	case ch <- llmResponse:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("TTS processing context canceled")
	}
}

func chainTTSOnEndFuncs(funcs ...func(error)) func(error) {
	filtered := make([]func(error), 0, len(funcs))
	for _, fn := range funcs {
		if fn != nil {
			filtered = append(filtered, fn)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	return func(err error) {
		for _, fn := range filtered {
			fn(err)
		}
	}
}

// waitForSync waits for completion, context cancellation, or timeout.
func (t *TTSManager) waitForSync(ctx context.Context, done <-chan struct{}) error {
	timer := time.NewTimer(ttsSyncWaitTimeout)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("TTS processing context canceled")
	case <-timer.C:
		return fmt.Errorf("TTS processing timed out")
	}
}

// handleTextResponse asynchronously queues text responses for TTS.
// Without dual streaming, each call queues one item. With dual streaming, IsStart
// creates the internal stream, later calls write to it, and IsEnd closes it.
func (t *TTSManager) handleTextResponse(ctx context.Context, llmResponse llm_common.LLMResponseStruct, isSync bool) error {
	return t.handleTextResponseWithHooks(ctx, llmResponse, isSync, nil, nil)
}

func (t *TTSManager) handleTextResponseWithHooks(ctx context.Context, llmResponse llm_common.LLMResponseStruct, isSync bool, onTTSItemEnqueued func() func(error), onTTSPlaybackStart func()) error {
	hasText := strings.TrimSpace(llmResponse.Text) != ""
	if !hasText && !llmResponse.IsEnd && !llmResponse.IsStart {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		log.Debugf("handleTextResponse: ignore response because ctx already canceled")
		return nil
	}

	// TTS_INPUT hook
	if t.session != nil {
		payload, stop, hookErr := t.session.hookHub.EmitTTSInput(t.session.hookContext(ctx), chathooks.TTSInputData{Text: llmResponse.Text, IsStart: llmResponse.IsStart, IsEnd: llmResponse.IsEnd})
		if hookErr != nil {
			log.Warnf("TTS_INPUT hook failed: %v", hookErr)
		}
		llmResponse.Text = payload.Text
		llmResponse.IsStart = payload.IsStart
		llmResponse.IsEnd = payload.IsEnd
		if stop {
			log.Infof("TTS_INPUT hook requested the current flow to stop")
			return nil
		}
	}

	// Recheck hasText because the hook may modify the text.
	hasText = strings.TrimSpace(llmResponse.Text) != ""

	if !t.SupportsDualStream() {
		if !hasText {
			return nil
		}
		gen := t.currentAudioGeneration()
		metricCycle := t.currentTtsMetricCycle()
		var done chan struct{}
		onEndFunc := func(error) {}
		if onTTSItemEnqueued != nil {
			onEndFunc = onTTSItemEnqueued()
		}
		if isSync {
			done = make(chan struct{}, 1)
			onEndFunc = chainTTSOnEndFuncs(onEndFunc, func(error) { signalDone(done) })
		}
		if err := t.pushTTSQueueItem(TTSQueueItem{
			ctx:         ctx,
			llmResponse: llmResponse,
			generation:  gen,
			metricCycle: metricCycle,
			onStartFunc: onTTSPlaybackStart,
			onEndFunc:   onEndFunc,
		}); err != nil {
			if onEndFunc != nil {
				onEndFunc(err)
			}
			return err
		}
		if isSync {
			return t.waitForSync(ctx, done)
		}
		return nil
	}

	// Dual-stream mode.
	var streamChan chan llm_common.LLMResponseStruct
	var streamOwnerCtx context.Context
	if llmResponse.IsStart {
		streamEpoch := t.dualStreamEpoch.Load()
		t.dualStreamMu.Lock()
		oldStreamChan := t.dualStreamChan
		t.dualStreamChan = nil
		t.dualStreamDone = nil
		t.dualStreamOwnerCtx = nil
		t.dualStreamMu.Unlock()
		safeCloseLLMResponseStream(oldStreamChan)

		streamChan = make(chan llm_common.LLMResponseStruct, 16)
		var done chan struct{}
		var onEndFunc func(error)
		if onTTSItemEnqueued != nil {
			onEndFunc = onTTSItemEnqueued()
		}
		if isSync {
			done = make(chan struct{}, 1)
			onEndFunc = chainTTSOnEndFuncs(onEndFunc, func(error) { signalDone(done) })
		}
		if err := t.pushTTSQueueItem(TTSQueueItem{
			ctx:         ctx,
			StreamChan:  streamChan,
			generation:  t.currentAudioGeneration(),
			metricCycle: t.currentTtsMetricCycle(),
			onStartFunc: onTTSPlaybackStart,
			onEndFunc:   onEndFunc,
		}); err != nil {
			safeCloseLLMResponseStream(streamChan)
			if onEndFunc != nil {
				onEndFunc(err)
			}
			return err
		}
		if t.dualStreamEpoch.Load() != streamEpoch {
			safeCloseLLMResponseStream(streamChan)
			return nil
		}
		t.dualStreamMu.Lock()
		if t.dualStreamEpoch.Load() != streamEpoch {
			t.dualStreamMu.Unlock()
			safeCloseLLMResponseStream(streamChan)
			return nil
		}
		t.dualStreamChan = streamChan
		t.dualStreamDone = done
		t.dualStreamOwnerCtx = ctx
		t.dualStreamMu.Unlock()
		log.Debugf("handleTextResponse: dual stream, created StreamChan and pushed item")
	} else {
		t.dualStreamMu.Lock()
		streamChan = t.dualStreamChan
		streamOwnerCtx = t.dualStreamOwnerCtx
		t.dualStreamMu.Unlock()
		if streamChan != nil && streamOwnerCtx != ctx {
			log.Debugf("handleTextResponse: dual stream fragment ignored because active stream belongs to another context")
			return nil
		}
	}

	if streamChan != nil && hasText {
		if err := sendLLMResponseToDualStream(ctx, streamChan, llmResponse); err != nil {
			return err
		}
	} else if streamChan == nil && hasText {
		// Fallback: queue as a single item when data arrives before IsStart.
		gen := t.currentAudioGeneration()
		var done chan struct{}
		var onEndFunc func(error)
		if onTTSItemEnqueued != nil {
			onEndFunc = onTTSItemEnqueued()
		}
		if isSync {
			done = make(chan struct{}, 1)
			onEndFunc = chainTTSOnEndFuncs(onEndFunc, func(error) { signalDone(done) })
		}
		if err := t.pushTTSQueueItem(TTSQueueItem{
			ctx:         ctx,
			llmResponse: llmResponse,
			generation:  gen,
			metricCycle: t.currentTtsMetricCycle(),
			onStartFunc: onTTSPlaybackStart,
			onEndFunc:   onEndFunc,
		}); err != nil {
			if onEndFunc != nil {
				onEndFunc(err)
			}
			return err
		}
		log.Debugf("handleTextResponse: dual stream fallback, no active stream, pushed single item")
		if isSync {
			return t.waitForSync(ctx, done)
		}
	}

	if llmResponse.IsEnd && streamChan != nil {
		var done chan struct{}
		t.dualStreamMu.Lock()
		if t.dualStreamChan == streamChan && t.dualStreamOwnerCtx == ctx {
			done = t.dualStreamDone
			t.dualStreamChan = nil
			t.dualStreamDone = nil
			t.dualStreamOwnerCtx = nil
		}
		t.dualStreamMu.Unlock()
		safeCloseLLMResponseStream(streamChan)
		if isSync && done != nil {
			return t.waitForSync(ctx, done)
		}
	}

	return nil
}

// getEffectiveTTSConfig returns speaker-specific config when available, otherwise device defaults.
func (t *TTSManager) getEffectiveTTSConfig() map[string]interface{} {
	if t.clientState.SpeakerTTSConfig != nil && len(t.clientState.SpeakerTTSConfig) > 0 {
		config := make(map[string]interface{})
		for k, v := range t.clientState.SpeakerTTSConfig {
			config[k] = v
		}
		return config
	}
	return t.clientState.DeviceConfig.Tts.Config
}

// SupportsDualStream reports whether the configured TTS provider streams both input and output.
func (t *TTSManager) SupportsDualStream() bool {
	config := t.getEffectiveTTSConfig()
	if config == nil {
		return false
	}
	v, ok := config["double_stream"]
	if !ok {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		return s == "true" || s == "1"
	}
	return false
}

// getTTSProviderInstance acquires a provider using provider and voice as the pool identity.
func (t *TTSManager) getTTSProviderInstance() (*pool.ResourceWrapper[tts.TTSProvider], error) {
	// Resolve TTS configuration and provider.
	var ttsConfig map[string]interface{}
	var ttsProvider string

	if t.clientState.SpeakerTTSConfig != nil && len(t.clientState.SpeakerTTSConfig) > 0 {
		// Use speaker-specific TTS configuration.
		if provider, ok := t.clientState.SpeakerTTSConfig["provider"].(string); ok {
			ttsProvider = provider
		} else {
			log.Warnf("speaker TTS configuration has no provider; using default configuration")
			ttsProvider = t.clientState.DeviceConfig.Tts.Provider
			ttsConfig = t.clientState.DeviceConfig.Tts.Config
		}
		// Deep-copy the configuration map.
		ttsConfig = make(map[string]interface{})
		for k, v := range t.clientState.SpeakerTTSConfig {
			ttsConfig[k] = v
		}
	} else {
		// Use default TTS configuration.
		ttsProvider = t.clientState.DeviceConfig.Tts.Provider
		ttsConfig = t.clientState.DeviceConfig.Tts.Config
	}

	// Logical label for logging and fingerprinting: provider or provider:voiceID.
	voiceID := extractVoiceID(ttsConfig)
	providerLabel := ttsProvider
	if voiceID != "" {
		providerLabel = fmt.Sprintf("%s:%s", ttsProvider, voiceID)
	}

	// Acquire from the pool; configuration fingerprints rotate pools when host or voice changes.
	ttsWrapper, err := pool.Acquire[tts.TTSProvider]("tts", providerLabel, ttsConfig)
	if err != nil {
		log.Errorf("failed to acquire TTS resource: %v", err)
		return nil, fmt.Errorf("failed to acquire TTS resource: %v", err)
	}

	return ttsWrapper, nil
}

// extractVoiceID extracts the voice ID from configuration.
func extractVoiceID(config map[string]interface{}) string {
	if config == nil {
		return ""
	}

	// Read the provider type from configuration.
	provider, _ := config["provider"].(string)

	// CosyVoice uses spk_id.
	if provider == "cosyvoice" {
		if spkID, ok := config["spk_id"].(string); ok && spkID != "" {
			return spkID
		}
		return ""
	}

	// MiniMax and other providers use voice.
	if voice, ok := config["voice"].(string); ok && voice != "" {
		return voice
	}

	return ""
}

// generateTtsOnly generates audio without sending it and returns a release callback.
func (t *TTSManager) generateTtsOnly(ctx context.Context, metricCycle uint64, llmResponse llm_common.LLMResponseStruct) (outputChan <-chan []byte, releaseFunc func(), err error) {
	if strings.TrimSpace(llmResponse.Text) == "" {
		return nil, nil, nil
	}
	ttsWrapper, err := t.getTTSProviderInstance()
	if err != nil {
		log.Errorf("failed to acquire TTS provider instance: %v", err)
		return nil, nil, err
	}
	ttsProviderInstance := ttsWrapper.GetProvider()
	t.markTtsMetricRequestStart(ctx, metricCycle)
	ch, err := ttsProviderInstance.TextToSpeechStream(ctx, llmResponse.Text, t.clientState.OutputAudioFormat.SampleRate, t.clientState.OutputAudioFormat.Channels, t.clientState.OutputAudioFormat.FrameDuration)
	if err != nil {
		pool.Release(ttsWrapper)
		t.finishTtsMetricRequest(ctx, metricCycle, err)
		log.Errorf("failed to generate TTS audio: %v", err)
		return nil, nil, fmt.Errorf("failed to generate TTS audio: %v", err)
	}
	return ch, func() { pool.Release(ttsWrapper) }, nil
}

// handleDualStreamTts streams text from StreamChan into the provider while streaming audio out.
// It returns false when the provider does not support dual streaming and fallback is required.
func (t *TTSManager) handleDualStreamTts(item TTSQueueItem) (bool, error) {
	ttsWrapper, err := t.getTTSProviderInstance()
	if err != nil {
		log.Errorf("failed to get provider for dual-stream TTS: %v", err)
		return false, nil
	}
	defer pool.Release(ttsWrapper)

	provider := ttsWrapper.GetProvider()
	adapter, ok := provider.(*tts.ContextTTSAdapter)
	if !ok {
		return false, nil
	}
	dp, ok := adapter.Provider.(tts.DualStreamProvider)
	if !ok {
		return false, nil
	}

	textChan := make(chan string, 16)
	t.markTtsMetricRequestStart(item.ctx, item.metricCycle)
	eventChan, err := dp.StreamingSynthesize(item.ctx, textChan,
		t.clientState.OutputAudioFormat.SampleRate,
		t.clientState.OutputAudioFormat.Channels,
		t.clientState.OutputAudioFormat.FrameDuration)
	if err != nil {
		close(textChan)
		t.finishTtsMetricRequest(item.ctx, item.metricCycle, err)
		log.Errorf("dual-stream TTS StreamingSynthesize failed: %v", err)
		return false, nil
	}
	requestActive := true
	firstAudioReported := false
	finishRequest := func(err error) {
		if requestActive {
			t.finishTtsMetricRequest(item.ctx, item.metricCycle, err)
			requestActive = false
		}
	}

	// Feed LLM response text from StreamChan into the TTS provider.
	go func() {
		defer close(textChan)
		for {
			select {
			case <-item.ctx.Done():
				return
			case resp, ok := <-item.StreamChan:
				if !ok {
					return
				}
				text := strings.TrimSpace(resp.Text)
				if text == "" {
					continue
				}
				select {
				case textChan <- text:
				case <-item.ctx.Done():
					return
				}
			}
		}
	}()

	firstSentence := true
	for event := range eventChan {
		for _, signal := range event.SentenceSignals {
			switch signal.Type {
			case ttsstream.SentenceSignalEnd:
				if !t.enqueueSessionElem(item.ctx, item.generation, AudioQueueElem{
					Kind: AudioQueueKindSentenceEnd,
					Text: signal.Text,
				}) {
					finishRequest(item.ctx.Err())
					if item.onEndFunc != nil {
						item.onEndFunc(item.ctx.Err())
					}
					return true, item.ctx.Err()
				}
			case ttsstream.SentenceSignalStart:
				startElem := AudioQueueElem{
					Kind:    AudioQueueKindSentenceStart,
					Text:    signal.Text,
					IsStart: firstSentence,
				}
				if firstSentence {
					startElem.OnStart = item.onStartFunc
					firstSentence = false
				}
				if !t.enqueueSessionElem(item.ctx, item.generation, startElem) {
					finishRequest(item.ctx.Err())
					if item.onEndFunc != nil {
						item.onEndFunc(item.ctx.Err())
					}
					return true, item.ctx.Err()
				}
			}
		}

		if len(event.Audio) > 0 {
			if !firstAudioReported {
				t.markTtsMetricFirstAudio(item.ctx, item.metricCycle)
				firstAudioReported = true
			}
			frameCopy := make([]byte, len(event.Audio))
			copy(frameCopy, event.Audio)
			if !t.enqueueSessionElem(item.ctx, item.generation, AudioQueueElem{Kind: AudioQueueKindFrame, Data: frameCopy, MetricCycle: item.metricCycle}) {
				finishRequest(item.ctx.Err())
				if item.onEndFunc != nil {
					item.onEndFunc(item.ctx.Err())
				}
				return true, item.ctx.Err()
			}
		}
	}

	finishRequest(nil)
	if !t.enqueueSessionElem(item.ctx, item.generation, AudioQueueElem{Kind: AudioQueueKindSentenceEnd, OnEnd: item.onEndFunc}) && item.onEndFunc != nil {
		item.onEndFunc(nil)
	}
	return true, item.ctx.Err()
}

// handleStreamTts reads item.StreamChan, generates each segment, and queues its control messages and frames.
func (t *TTSManager) handleStreamTts(item TTSQueueItem) error {
	if t.SupportsDualStream() {
		handled, err := t.handleDualStreamTts(item)
		if handled {
			return err
		}
	}

	firstSegment := true
	for {
		select {
		case <-item.ctx.Done():
			if !t.enqueueSessionElem(item.ctx, item.generation, AudioQueueElem{Kind: AudioQueueKindSentenceEnd, OnEnd: item.onEndFunc, Err: item.ctx.Err()}) && item.onEndFunc != nil {
				item.onEndFunc(item.ctx.Err())
			}
			return item.ctx.Err()
		case resp, ok := <-item.StreamChan:
			if !ok {
				if !t.enqueueSessionElem(item.ctx, item.generation, AudioQueueElem{Kind: AudioQueueKindSentenceEnd, OnEnd: item.onEndFunc}) && item.onEndFunc != nil {
					item.onEndFunc(nil)
				}
				return item.ctx.Err()
			}
			outChan, release, genErr := t.generateTtsOnly(item.ctx, item.metricCycle, resp)
			if genErr != nil {
				if firstSegment {
					if !t.enqueueSessionElem(item.ctx, item.generation, AudioQueueElem{Kind: AudioQueueKindSentenceStart, OnStart: item.onStartFunc}) {
						if item.onEndFunc != nil {
							item.onEndFunc(item.ctx.Err())
						}
						return item.ctx.Err()
					}
				}
				if !t.enqueueSessionElem(item.ctx, item.generation, AudioQueueElem{Kind: AudioQueueKindSentenceEnd, OnEnd: item.onEndFunc, Err: genErr}) && item.onEndFunc != nil {
					item.onEndFunc(genErr)
				}
				return genErr
			}
			if outChan == nil {
				if release != nil {
					release()
				}
				continue
			}
			requestActive := true
			firstAudioReported := false
			finishRequest := func(err error) {
				if requestActive {
					t.finishTtsMetricRequest(item.ctx, item.metricCycle, err)
					requestActive = false
				}
			}
			startElem := AudioQueueElem{
				Kind:    AudioQueueKindSentenceStart,
				Text:    resp.Text,
				IsStart: resp.IsStart,
			}
			if firstSegment {
				startElem.OnStart = item.onStartFunc
				firstSegment = false
			}
			if !t.enqueueSessionElem(item.ctx, item.generation, startElem) {
				if release != nil {
					release()
				}
				finishRequest(item.ctx.Err())
				if item.onEndFunc != nil {
					item.onEndFunc(item.ctx.Err())
				}
				return item.ctx.Err()
			}
			for {
				select {
				case <-item.ctx.Done():
					if release != nil {
						release()
					}
					finishRequest(item.ctx.Err())
					if item.onEndFunc != nil {
						item.onEndFunc(item.ctx.Err())
					}
					return item.ctx.Err()
				case frame, ok := <-outChan:
					if !ok {
						if release != nil {
							release()
						}
						finishRequest(nil)
						if !t.enqueueSessionElem(item.ctx, item.generation, AudioQueueElem{Kind: AudioQueueKindSentenceEnd, Text: resp.Text}) && item.onEndFunc != nil {
							item.onEndFunc(item.ctx.Err())
						}
						goto nextResp
					}
					if !firstAudioReported {
						t.markTtsMetricFirstAudio(item.ctx, item.metricCycle)
						firstAudioReported = true
					}
					frameCopy := make([]byte, len(frame))
					copy(frameCopy, frame)
					if !t.enqueueSessionElem(item.ctx, item.generation, AudioQueueElem{Kind: AudioQueueKindFrame, Data: frameCopy, MetricCycle: item.metricCycle}) {
						if release != nil {
							release()
						}
						finishRequest(item.ctx.Err())
						if item.onEndFunc != nil {
							item.onEndFunc(item.ctx.Err())
						}
						return item.ctx.Err()
					}
				}
			}
		nextResp:
		}
	}
}

// getAlignedDuration rounds elapsed time up to frameDuration.
func getAlignedDuration(startTime time.Time, frameDuration time.Duration) time.Duration {
	elapsed := time.Since(startTime)
	// Round up to frameDuration.
	alignedMs := ((elapsed.Milliseconds() + frameDuration.Milliseconds() - 1) / frameDuration.Milliseconds()) * frameDuration.Milliseconds()
	return time.Duration(alignedMs) * time.Millisecond
}

func (t *TTSManager) sendAudioStream(ctx context.Context, audioChan <-chan []byte, isStart bool, recordHistory bool) error {
	totalFrames := 0 // Track total frames sent.

	isStatistic := true
	// Calculate initial buffered audio from outputAudioFormat.FrameDuration.
	cacheFrameCount := 120 / t.clientState.OutputAudioFormat.FrameDuration
	/*if cacheFrameCount > 20 || cacheFrameCount < 3 {
		cacheFrameCount = 5
	}*/

	// Record the send start time.
	startTime := time.Now()

	// Use absolute time for precise flow control.
	frameDuration := time.Duration(t.clientState.OutputAudioFormat.FrameDuration) * time.Millisecond

	log.Debugf("SendTTSAudio started, buffered frames: %d, frame duration: %v", cacheFrameCount, frameDuration)

	// Maintain cacheFrameCount buffered frames at the peer.
	for {
		// Calculate when the next frame should be sent.
		nextFrameTime := startTime.Add(time.Duration(totalFrames-cacheFrameCount) * frameDuration)
		now := time.Now()

		// Wait until the next frame deadline.
		if now.Before(nextFrameTime) {
			sleepDuration := nextFrameTime.Sub(now)
			//log.Debugf("SendTTSAudio flow-control wait: %v", sleepDuration)
			time.Sleep(sleepDuration)
		}

		// Receive and send the next frame.
		select {
		case <-ctx.Done():
			log.Debugf("SendTTSAudio context done, exit")
			return nil
		case frame, ok := <-audioChan:
			if !ok {
				// The channel is closed. Wait for the remaining playback duration.
				elapsed := time.Since(startTime)
				totalDuration := time.Duration(totalFrames) * frameDuration
				if totalDuration > elapsed {
					waitDuration := totalDuration - elapsed
					log.Debugf("SendTTSAudio waiting for client playback buffer: %v (totalFrames=%d, frameDuration=%v)", waitDuration, totalFrames, frameDuration)
					time.Sleep(waitDuration)
				}

				log.Debugf("SendTTSAudio audioChan closed, exit, total frames sent: %d", totalFrames)
				return nil
			}
			// Send the current frame.
			if err := t.serverTransport.SendAudio(frame); err != nil {
				log.Errorf("failed to send TTS audio frame %d, len: %d, error: %v", totalFrames, len(frame), err)
				return fmt.Errorf("failed to send TTS audio, len: %d: %v", len(frame), err)
			}

			if recordHistory {
				// Append each audio frame independently to the history cache.
				t.audioMutex.Lock()
				frameCopy := make([]byte, len(frame))
				copy(frameCopy, frame)
				t.audioHistoryBuffer = append(t.audioHistoryBuffer, frameCopy)
				t.audioMutex.Unlock()
			}

			totalFrames++
			if totalFrames%100 == 0 {
				log.Debugf("SendTTSAudio sent %d frames", totalFrames)
			}

			// Record latency once at startup.
			if isStart && isStatistic && totalFrames == 1 {
				log.Debugf("end-of-input to ASR/LLM/TTS first-frame latency: %d ms", t.clientState.GetAsrLlmTtsDuration())
				isStatistic = false
			}
		}
	}
}

func (t *TTSManager) SendTTSAudio(ctx context.Context, audioChan <-chan []byte, isStart bool) error {
	if err := t.waitForMediaPlaybackRelease(ctx); err != nil {
		return err
	}
	return t.sendAudioStream(ctx, audioChan, isStart, true)
}

func (t *TTSManager) SendMediaAudio(ctx context.Context, audioChan <-chan []byte) error {
	return t.sendAudioStream(ctx, audioChan, false, false)
}

// ClearAudioHistory clears the TTS audio history cache.
func (t *TTSManager) ClearAudioHistory() {
	t.audioMutex.Lock()
	defer t.audioMutex.Unlock()
	t.audioHistoryBuffer = nil
}

// GetAndClearAudioHistory returns and clears the TTS audio history cache.
func (t *TTSManager) GetAndClearAudioHistory() [][]byte {
	t.audioMutex.Lock()
	defer t.audioMutex.Unlock()
	data := t.audioHistoryBuffer
	t.audioHistoryBuffer = nil
	return data
}
