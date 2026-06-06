package chat

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
	. "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/domain/asr"
	asr_types "xiaozhi-esp32-server-golang/internal/domain/asr/types"
	"xiaozhi-esp32-server-golang/internal/domain/audio"
	chathooks "xiaozhi-esp32-server-golang/internal/domain/chat/hooks"
	"xiaozhi-esp32-server-golang/internal/domain/speaker"
	"xiaozhi-esp32-server-golang/internal/domain/vad/inter"
	"xiaozhi-esp32-server-golang/internal/pool"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"
)

type ASRManagerOption func(*ASRManager)

const maxFirstSpeechPreAudioMs = 200

// AsrMessageSaveCallback persists a recognized user message and its audio.
type AsrMessageSaveCallback func(userMsg *schema.Message, messageID string, audioData []float32)

type ASRManager struct {
	clientState     *ClientState
	serverTransport *ServerTransport
	session         *ChatSession // Provides access to speakerManager.

	// Manage the ASR resource privately.
	asrResource *pool.ResourceWrapper[asr.AsrProvider]
	resourceMu  sync.RWMutex // Protect resource access.
}

func NewASRManager(clientState *ClientState, serverTransport *ServerTransport, opts ...ASRManagerOption) *ASRManager {
	asr := &ASRManager{
		clientState:     clientState,
		serverTransport: serverTransport,
		session:         nil, // Set later through SetSession.
	}
	for _, opt := range opts {
		opt(asr)
	}
	return asr
}

func (a *ASRManager) runAudioIdleTimeoutWatchdog(ctx context.Context) {
	state := a.clientState
	if state == nil {
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !state.UsesAudioIdleClock() || !state.AudioIdleStarted() || state.AudioIdlePaused() {
				continue
			}
			if !state.ShouldCountAudioIdleTimeout() || state.Asr.HasReceivedText() {
				continue
			}
			if state.GetClientVoiceStop() || state.AudioIdleTimeoutPending() {
				continue
			}

			elapsed := state.GetAudioIdleElapsed(time.Now())
			threshold := time.Duration(state.GetMaxIdleDuration()) * time.Millisecond
			if elapsed < threshold {
				continue
			}
			if !state.MarkAudioIdleTimeoutPending() {
				continue
			}

			if !state.Asr.HasOpenAudioInput() {
				log.Infof(
					"audio idle timeout with no active ASR stream; closing session: device=%s, mode=%s, elapsed=%dms, threshold=%dms",
					state.DeviceID,
					state.ListenMode,
					elapsed.Milliseconds(),
					state.GetMaxIdleDuration(),
				)
				if a.session != nil {
					a.session.CloseWithReason(chatSessionCloseReasonAudioIdleTimeout)
				} else {
					state.ClearAudioIdleTimeoutPending()
				}
				continue
			}

			log.Infof(
				"audio idle timeout; finalizing ASR: device=%s, mode=%s, elapsed=%dms, threshold=%dms",
				state.DeviceID,
				state.ListenMode,
				elapsed.Milliseconds(),
				state.GetMaxIdleDuration(),
			)
			state.OnVoiceSilence()
		}
	}
}

// ProcessVadAudio starts VAD audio processing.
func (a *ASRManager) ProcessVadAudio(ctx context.Context) {
	state := a.clientState
	go func() {
		hasTriggeredCancel := true // Tracks whether cancellation has already been triggered.
		hasLoggedFirstTextExtendedWait := false
		speakerInterruptTriggered := atomic.Bool{}
		speakerPeekInFlight := atomic.Bool{}
		lastSpeakerPeekDoneAt := atomic.Int64{}
		var speakerPeekAudioMs int64
		var speakerPeekRequestSeq uint64
		const speakerPeekInterval = 200 * time.Millisecond
		const firstSpeakerPeekAudioThresholdMs int64 = 400
		audioFormat := state.InputAudioFormat
		// Allocate for a maximum assumed frame duration of 120 ms.
		maxFrameSize := audioFormat.SampleRate * audioFormat.Channels * 120 / 1000
		audioProcesser, err := audio.GetAudioProcesser(audioFormat.SampleRate, audioFormat.Channels, 20) // Use a default frame duration to create the decoder.
		if err != nil {
			log.Errorf("failed to get audio decoder: %v", err)
			return
		}

		// Derive frame size and duration from the first decoded frame.
		var frameSize int
		var frameDurationMs int
		var vadNeedGetCount int // Required VAD frame count, calculated after the first frame.

		// Lazily acquire VAD and release it when idle to avoid holding a pooled instance.
		var vadWrapper *pool.ResourceWrapper[inter.VAD]
		var vadProvider inter.VAD
		var vadLastUseAt time.Time
		const vadIdleReleaseTimeout = 2 * time.Second
		vadIdleTicker := time.NewTicker(time.Second)
		defer vadIdleTicker.Stop()
		needVad := !(state.Asr.AutoEnd || state.ListenMode == "manual")
		vadProviderName := state.DeviceConfig.Vad.Provider
		vadProviderConfig := state.DeviceConfig.Vad.Config
		effectiveVadProviderName := vadProviderName
		if configProvider, ok := vadProviderConfig["provider"].(string); ok && configProvider != "" {
			effectiveVadProviderName = configProvider
		}
		isSileroVAD := effectiveVadProviderName == "silero_vad"
		releaseVad := func(reason string) {
			if vadWrapper == nil {
				return
			}
			pool.Release(vadWrapper)
			vadWrapper = nil
			vadProvider = nil
			vadLastUseAt = time.Time{}
			log.Debugf("released VAD resource: device=%s, reason=%s", state.DeviceID, reason)
		}
		defer releaseVad("process_exit")
		ensureVad := func() bool {
			if !needVad {
				return false
			}
			if vadProvider != nil {
				return true
			}

			// Warn when the provider is empty and configuration fallback is attempted.
			if vadProviderName == "" {
				log.Warnf("VAD provider is empty; attempting to read it from config")
			} else {
				log.Debugf("acquiring VAD resource: provider=%s", vadProviderName)
			}

			wrapper, err := pool.Acquire[inter.VAD](
				"vad",
				vadProviderName,
				vadProviderConfig,
			)
			if err != nil {
				log.Errorf("failed to acquire VAD resource: provider=%s, config=%+v, error=%v", vadProviderName, vadProviderConfig, err)
				return false
			}
			vadWrapper = wrapper
			vadProvider = wrapper.GetProvider()
			vadLastUseAt = time.Now()
			return true
		}
		for {
			// Decode into a maximum-sized buffer and use the actual decoded size.
			pcmFrame := make([]float32, maxFrameSize)

			select {
			case <-vadIdleTicker.C:
				if vadWrapper != nil && !vadLastUseAt.IsZero() && time.Since(vadLastUseAt) >= vadIdleReleaseTimeout {
					releaseVad("idle_timeout")
				}
				continue
			case opusFrame, ok := <-state.OpusAudioBuffer:
				//log.Debugf("processAsrAudio received audio data, len: %d", len(opusFrame))
				if !ok {
					log.Debugf("processAsrAudio audio channel closed")
					return
				}

				var skipVad bool
				var haveVoice bool
				clientHaveVoice := state.GetClientHaveVoice()
				if state.ListenMode == "manual" {
					skipVad = true         // Skip VAD.
					clientHaveVoice = true // Voice was already detected.
					haveVoice = true       // Current frame has voice.
				} else if state.Asr.AutoEnd {
					skipVad = true   // Provider still controls stop without changing idle semantics.
					haveVoice = true // Send current audio directly to ASR.
				}

				if state.GetClientVoiceStop() { // Do not accept audio after the client stops speaking.
					//log.Infof("client stopped speaking; skipping audio data")
					continue
				}

				//log.Debugf("clientVoiceStop: %+v, asrDataSize: %d, listenMode: %s, isSkipVad: %v\n", state.GetClientVoiceStop(), state.AsrAudioBuffer.GetAsrDataSize(), state.ListenMode, skipVad)

				n, err := audioProcesser.DecoderFloat32(opusFrame, pcmFrame)
				if err != nil {
					log.Errorf("audio decode failed: %v", err)
					continue
				}

				// Calculate frame size and duration from decoded data.
				if frameSize == 0 {
					// Calculate frame metadata from the first decoded frame.
					frameSize = n
					samplesPerChannel := n / audioFormat.Channels
					frameDurationMs = samplesPerChannel * 1000 / audioFormat.SampleRate
					audioFormat.FrameDuration = frameDurationMs

					// Calculate the frame count required by VAD.
					vadNeedGetCount = 1
					log.Debugf("calculated frame metadata from decoded audio: frameSize=%d, frameDurationMs=%d, vadNeedGetCount=%d", frameSize, frameDurationMs, vadNeedGetCount)
				}

				var vadPcmData []float32
				pcmData := pcmFrame[:n]
				speakerPcmData := pcmFrame[:n]

				// Use the actual size when frame sizes are inconsistent.
				if n != frameSize {
					log.Debugf("frame size mismatch: expected=%d, actual=%d; using actual size", frameSize, n)
					// Recalculate this frame's duration.
					samplesPerChannel := n / audioFormat.Channels
					currentFrameDurationMs := samplesPerChannel * 1000 / audioFormat.SampleRate
					frameSize = n
					frameDurationMs = currentFrameDurationMs
					audioFormat.FrameDuration = frameDurationMs
				}

				if !skipVad && needVad {
					if !ensureVad() {
						continue
					}
					//decode opus to pcm
					state.AsrAudioBuffer.AddAsrAudioData(pcmData)

					// Calculate the minimum data required by VAD.
					vadNeedMinSize := frameSize

					if state.AsrAudioBuffer.GetAsrDataSize() >= vadNeedMinSize {
						if isSileroVAD {
							vadPcmData = pcmData
						} else {
							vadPcmData = state.AsrAudioBuffer.GetAsrData(vadNeedGetCount, frameSize)
						}

						// Use the VAD resource acquired outside the loop.
						if !isSileroVAD {
							vadLastUseAt = time.Now()
							if err := vadProvider.Reset(); err != nil {
								log.Errorf("failed to reset VAD: %v", err)
								continue
							}
						}

						// Run VAD.
						vadLastUseAt = time.Now()
						haveVoice, err = vadProvider.IsVADExt(vadPcmData, audioFormat.SampleRate, frameSize)
						if err != nil {
							log.Errorf("processAsrAudio VAD detection failed: %v", err)
							continue
						}

						// Preserve leading audio when speech is first detected.
						if haveVoice && !clientHaveVoice {
							// Keep at most 200 ms of leading silence.
							currentFrameSamples := len(pcmData)
							allData := state.AsrAudioBuffer.GetAndClearAllData()
							pcmData = trimFirstSpeechAudio(allData, currentFrameSamples, audioFormat.SampleRate, audioFormat.Channels)
						}
					}
					//log.Debugf("isVad, pcmData len: %d, vadPcmData len: %d, haveVoice: %v", len(pcmData), len(vadPcmData), haveVoice)
				}

				if haveVoice {
					hasLoggedFirstTextExtendedWait = false
					//log.Infof("speech detected, len: %d", len(pcmData))
					state.SetClientHaveVoice(true)
					state.SetClientHaveVoiceLastTime(time.Now().UnixMilli())
					state.Vad.ResetIdleDuration()
					// Accumulate detected speech duration.
					state.Vad.AddVoiceDuration(int64(frameDurationMs))

					continuousVoiceDuration := state.Vad.GetVoiceContinuousDuration()
					if state.IsRealTime() && viper.GetInt("chat.realtime_mode") == 1 && continuousVoiceDuration > 360 {
						// Trigger only once.
						if !hasTriggeredCancel {
							if a.session != nil && a.session.isRealtimeMcpAudioGateActive() {
								log.Debugf("device %s realtime media playback gate active; skipping VAD interruption", state.DeviceID)
								hasTriggeredCancel = true
							} else {
								// Cancel active LLM and TTS work after a realtime VAD interruption.
								log.Debugf("realtime VAD interruption after %d ms of speech; canceling active LLM and TTS", continuousVoiceDuration)
								if a.session != nil {
									a.session.StopAssistantOutputAfterAsrWithReason(true, "ASRManager.ProcessVadAudio realtime_mode=1 VAD interrupt")
								} else {
									state.AfterAsrSessionCtx.CancelWithReason("ASRManager.ProcessVadAudio: realtime_mode=1 VAD interrupt")
								}
								hasTriggeredCancel = true // Mark as triggered.
							}
						}
					}
				} else {
					state.Vad.AddIdleDuration(int64(frameDurationMs))
					state.Vad.ResetVoiceContinuousDuration()

					// Reset accumulated duration only when no prior speech was detected.
					if !clientHaveVoice {
						speakerInterruptTriggered.Store(false)
						lastSpeakerPeekDoneAt.Store(0)
						speakerPeekAudioMs = 0
						// Keep the most recent frames.
						/*
							if state.AsrAudioBuffer.GetFrameCount(frameSize) > vadNeedGetCount*3 {
								state.AsrAudioBuffer.RemoveAsrAudioData(1, frameSize)
							}*/
						continue
					}
				}

				if clientHaveVoice || haveVoice {
					// Forward the buffered frame immediately on first speech to preserve very short utterances.

					// VAD detected speech; send data to the ASR audio channel.
					//log.Infof("VAD detected speech; sending data to ASR audio channel, len: %d", len(pcmData))
					state.Asr.AddAudioData(pcmData)

					// Send only voiced frames to speaker recognition.
					if haveVoice &&
						state.IsSpeakerEnabled() && state.HasSpeakerGroups() &&
						a.session != nil && a.session.speakerManager != nil {
						// Start streaming speaker recognition on first speech.
						if !a.session.speakerManager.IsActive() {
							sampleRate := audioFormat.SampleRate
							agentId := a.session.clientState.AgentID
							if err := a.session.speakerManager.StartStreaming(ctx, sampleRate, agentId); err != nil {
								log.Warnf("failed to start speaker recognition stream: %v", err)
							} else {
								speakerInterruptTriggered.Store(false)
								lastSpeakerPeekDoneAt.Store(0)
								speakerPeekAudioMs = 0
							}
						}

						// Send the audio chunk.
						if err := a.session.speakerManager.SendAudioChunk(ctx, speakerPcmData); err != nil {
							log.Warnf("failed to send audio chunk to speaker recognition service: %v", err)
						} else if a.session.speakerManager.IsActive() {
							if audioFormat.Channels > 0 && audioFormat.SampleRate > 0 {
								speakerPeekAudioMs += int64(len(speakerPcmData)/audioFormat.Channels) * 1000 / int64(audioFormat.SampleRate)
							}

							if state.IsRealTime() &&
								viper.GetInt("chat.realtime_mode") == 3 &&
								!speakerInterruptTriggered.Load() &&
								speakerPeekAudioMs >= firstSpeakerPeekAudioThresholdMs {
								now := time.Now()
								lastDoneAt := lastSpeakerPeekDoneAt.Load()
								if (lastDoneAt <= 0 || now.Sub(time.Unix(0, lastDoneAt)) >= speakerPeekInterval) &&
									speakerPeekInFlight.CompareAndSwap(false, true) {
									reqSeq := atomic.AddUint64(&speakerPeekRequestSeq, 1)
									requestID := fmt.Sprintf("peek_%d_%d", now.UnixMilli(), reqSeq)

									go func(reqID string) {
										defer func() {
											lastSpeakerPeekDoneAt.Store(time.Now().UnixNano())
											speakerPeekInFlight.Store(false)
										}()

										if a.session == nil || a.session.speakerManager == nil || !a.session.speakerManager.IsActive() {
											return
										}

										peekCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
										defer cancel()

										peekResult, throttled, err := a.session.speakerManager.PeekAndIdentify(peekCtx, reqID)
										if err != nil {
											if ctx.Err() == nil {
												log.Debugf("speaker peek failed: device=%s, request_id=%s, err=%v", state.DeviceID, reqID, err)
											}
											return
										}
										if throttled {
											return
										}
										if peekResult == nil || !peekResult.Identified {
											return
										}
										if !speakerInterruptTriggered.CompareAndSwap(false, true) {
											return
										}

										log.Infof(
											"realtime speaker peek matched; interrupting immediately: device=%s, speaker=%s, confidence=%.4f, threshold=%.4f",
											state.DeviceID,
											peekResult.SpeakerName,
											peekResult.Confidence,
											peekResult.Threshold,
										)
										if a.session != nil && a.session.isRealtimeMcpAudioGateActive() {
											log.Debugf("device %s realtime media playback gate active; skipping speaker peek interruption", state.DeviceID)
											return
										}
										a.session.MarkTurnSpeakerInterrupted()
										if a.session != nil {
											a.session.StopAssistantOutputAfterAsrWithReason(true, "ASRManager.ProcessVadAudio realtime_mode=3 speaker peek interrupt")
										} else {
											state.AfterAsrSessionCtx.CancelWithReason("ASRManager.ProcessVadAudio: realtime_mode=3 speaker peek interrupt")
										}
									}(requestID)
								}
							}
						}
					}
				}

				// Determine whether speech ended after a voiced period.
				lastHaveVoiceTime := state.GetClientHaveVoiceLastTime()

				if clientHaveVoice && lastHaveVoiceTime > 0 && !haveVoice {
					// Reset short speech to avoid false positives.
					voiceDurationInSession := state.Vad.GetVoiceDurationInSession()
					if voiceDurationInSession < 100 {
						log.Debugf("speech duration too short (%dms < 300ms); resetting clientHaveVoice", voiceDurationInSession)
						state.SetClientHaveVoice(false)
						state.Vad.ResetVoiceDuration()
						speakerInterruptTriggered.Store(false)
						lastSpeakerPeekDoneAt.Store(0)
						speakerPeekAudioMs = 0
						continue
					}

					idleDuration := state.Vad.GetIdleDuration()
					if state.IsRealTime() && !state.Asr.HasReceivedText() {
						preTextSilenceDuration := state.GetPreAsrTextSilenceDuration()
						if idleDuration <= preTextSilenceDuration {
							log.Debugf(
								"realtime mode has not received initial ASR text; delaying finalization to silence threshold: status=%s, idle=%dms, pre_text_timeout=%dms, voice_duration=%dms, voice_duration_in_session=%dms, history_audio_samples=%d",
								state.Status,
								idleDuration,
								preTextSilenceDuration,
								state.Vad.GetVoiceDuration(),
								voiceDurationInSession,
								state.Asr.GetHistoryAudioLen(),
							)
							continue
						}

						if !hasLoggedFirstTextExtendedWait {
							log.Debugf(
								"realtime silence timeout without ASR text; keeping stream open and forwarding audio: status=%s, idle=%dms, pre_text_timeout=%dms, voice_duration=%dms, voice_duration_in_session=%dms, history_audio_samples=%d",
								state.Status,
								idleDuration,
								preTextSilenceDuration,
								state.Vad.GetVoiceDuration(),
								voiceDurationInSession,
								state.Asr.GetHistoryAudioLen(),
							)
							hasLoggedFirstTextExtendedWait = true
						}
						continue
					}

					if state.IsSilence(idleDuration) { // Transition from speech to silence.
						log.Debugf(
							"speech ended; preparing to stop ASR: status=%s, idle=%dms, voice_duration=%dms, voice_duration_in_session=%dms, history_audio_samples=%d, pending_restart=%v",
							state.Status,
							idleDuration,
							state.Vad.GetVoiceDuration(),
							state.Vad.GetVoiceDurationInSession(),
							state.Asr.GetHistoryAudioLen(),
							state.AudioIdleTimeoutPending(),
						)
						// Reset before OnVoiceSilence so the next turn can trigger again.
						hasTriggeredCancel = false
						speakerInterruptTriggered.Store(false)
						lastSpeakerPeekDoneAt.Store(0)
						speakerPeekAudioMs = 0
						state.OnVoiceSilence()
						//state.VoiceStatus.Reset()
						continue
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()
}

// releaseResource releases the ASR resource.
func (a *ASRManager) releaseResource() {
	a.resourceMu.Lock()
	defer a.resourceMu.Unlock()
	if a.asrResource != nil {
		pool.Release(a.asrResource)
		a.asrResource = nil
		log.Debugf("ASR resource returned")
	}
}

// Cleanup releases ASR resources.
func (a *ASRManager) Cleanup() {
	a.releaseResource()
}

// RestartAsrRecognition restarts ASR recognition.
func (a *ASRManager) RestartAsrRecognition(ctx context.Context) error {
	state := a.clientState
	log.Debugf("starting ASR recognition restart")
	if a.session != nil {
		a.session.ResetTurnSpeakerInterrupted()
	}

	// Cancel the current ASR context.
	state.Asr.CancelWithReason("ASRManager.RestartAsrRecognition: cancel previous ASR context before restart")

	state.Asr.ResetReceivedText()
	state.VoiceStatus.Reset()
	state.AsrAudioBuffer.ClearAsrAudioData()
	state.Asr.ClearHistoryAudio() // Clear historical audio.

	// Acquire a resource when none is held.
	a.resourceMu.Lock()
	var asrProvider asr.AsrProvider
	if a.asrResource == nil {
		// Acquire a new resource.
		a.resourceMu.Unlock()

		asrWrapper, err := pool.Acquire[asr.AsrProvider](
			"asr",
			state.DeviceConfig.Asr.Provider,
			state.DeviceConfig.Asr.Config,
		)
		if err != nil {
			log.Errorf("failed to acquire ASR resource: %v", err)
			return fmt.Errorf("failed to acquire ASR resource: %w", err)
		}

		// Store the private resource reference.
		a.resourceMu.Lock()
		a.asrResource = asrWrapper
		asrProvider = asrWrapper.GetProvider()
		a.resourceMu.Unlock()
		log.Debugf("acquired new ASR resource")
	} else {
		// Reuse the existing resource.
		asrProvider = a.asrResource.GetProvider()
		a.resourceMu.Unlock()
		log.Debugf("reusing existing ASR resource")
	}

	// Recreate the ASR context and channel.
	state.Asr.Ctx, state.Asr.Cancel = context.WithCancel(ctx)
	state.Asr.AsrAudioChannel = make(chan []float32, 100)

	// Restart streaming recognition.
	asrResultChannel, err := asrProvider.StreamingRecognize(state.Asr.Ctx, state.Asr.AsrAudioChannel)
	if err != nil {
		// Return the resource after recognition failure because it may be invalid.
		a.releaseResource()
		log.Errorf("failed to restart ASR streaming recognition: %v", err)
		return fmt.Errorf("failed to restart ASR streaming recognition: %w", err)
	}

	state.AsrResultChannel = asrResultChannel
	// Reset metrics for the current turn.
	state.MarkTurnStart()
	if a.session != nil {
		a.session.TraceTurnStart(state.Asr.Ctx, state.Statistic.TurnStartTs)
	}
	log.Debugf("ASR recognition restarted successfully")
	return nil
}

// StartAsrRecognitionLoop starts the ASR result processing loop.
// onMessageSave persists messages; onError handles failures such as session closure.
func (a *ASRManager) StartAsrRecognitionLoop(
	ctx context.Context,
	onMessageSave AsrMessageSaveCallback,
	onError func(error),
) {
	state := a.clientState

	// Start a goroutine to process ASR results.
	go func() {
		// Ensure ASR resources are released when the goroutine exits.
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("ASR result processing goroutine panic: %v, stack: %s", r, string(debug.Stack()))
			}
			// Release resources on normal exit or panic.
			a.releaseResource()
		}()

		// Maximum idle time is 60 seconds.
		var startIdleTime, maxIdleTime int64
		startIdleTime = time.Now().Unix()
		maxIdleTime = 60

		// Count waits while restart is disallowed to prevent an infinite loop.
		var invalidStatusWaitCount int64
		maxInvalidStatusWaitCount := int64(10) // Wait at most 10 times, about one second.

		// Protect against repeated empty ASR results causing a tight loop.
		const emptyResultProtectWindow = 3 * time.Second
		const maxEmptyResultInWindow = 3
		emptyResultWindowStart := time.Now()
		emptyResultCount := 0

		// Protect against endless reconnects on repeated recoverable upstream errors.
		const recoverableErrorProtectWindow = 10 * time.Second
		const maxRecoverableErrorInWindow = 3
		recoverableErrorWindowStart := time.Now()
		recoverableErrorCount := 0

		isAllowedToRestart := func() bool {
			allowed := state.Status == ClientStatusListening || state.Status == ClientStatusListenStop
			if state.IsRealTime() {
				allowed = state.Status != ClientStatusInit
			}
			return allowed
		}
		resumeAudioIdle := func() {
			state.ResumeAudioIdleWindow(time.Now())
		}
		startAudioIdle := func() {
			state.StartAudioIdleWindow(time.Now())
		}
		closeAudioIdleTimeout := func(reason string) {
			if !state.AudioIdleTimeoutPending() {
				return
			}

			state.ClearAudioIdleTimeoutPending()
			log.Infof("audio idle timeout finalization completed: device=%s, reason=%s", state.DeviceID, reason)
			if a.session != nil {
				a.session.CloseWithReason(chatSessionCloseReasonAudioIdleTimeout)
				return
			}
			if onError != nil {
				onError(fmt.Errorf("audio idle timeout: %s", reason))
			}
		}

		for {
			select {
			case <-ctx.Done():
				log.Debugf("asr ctx done")
				return
			default:
			}

			result, isRetry, err := state.RetireAsrResult(ctx)
			if err != nil {
				if ctx.Err() != nil || errors.Is(err, context.Canceled) {
					log.Debugf("failed to process ASR result after cancellation: %v", err)
				} else {
					log.Errorf("failed to process ASR result: %v", err)
				}
				if onError != nil {
					onError(err)
				}
				return
			}
			if !isRetry {
				log.Debugf("asrResult is not retry, return")
				return
			}
			text := result.Text

			if result.RetryReason != "" {
				if state.AudioIdleTimeoutPending() {
					closeAudioIdleTimeout(result.RetryReason)
					return
				}

				now := time.Now()
				if now.Sub(recoverableErrorWindowStart) > recoverableErrorProtectWindow {
					recoverableErrorWindowStart = now
					recoverableErrorCount = 0
				}
				recoverableErrorCount++
				log.Warnf(
					"recoverable ASR error: reason=%s, count=%d/%d, status=%s",
					result.RetryReason,
					recoverableErrorCount,
					maxRecoverableErrorInWindow,
					state.Status,
				)

				if recoverableErrorCount >= maxRecoverableErrorInWindow {
					err := fmt.Errorf("ASR triggered %d recoverable errors within %s; stopping retries and disconnecting", recoverableErrorCount, recoverableErrorProtectWindow)
					log.Errorf("%v", err)
					if onError != nil {
						onError(err)
					}
					return
				}

				switch result.RetryReason {
				case asr_types.RetryReasonDoubaoResponseCode45000081, asr_types.RetryReasonXunfeiServiceInstanceInvalid, asr_types.RetryReasonAliyunQwen3ConnectionClosed:
					a.releaseResource()
					if isAllowedToRestart() {
						invalidStatusWaitCount = 0
						if restartErr := a.RestartAsrRecognition(ctx); restartErr != nil {
							log.Errorf("failed to restart recognition after recoverable ASR error: reason=%s, err=%v", result.RetryReason, restartErr)
							if onError != nil {
								onError(restartErr)
							}
							return
						}
						resumeAudioIdle()
						continue
					}

					log.Warnf("current state does not allow immediate restart after recoverable ASR error: reason=%s, status=%s, realtime=%v", result.RetryReason, state.Status, state.IsRealTime())
					state.Asr.CancelWithReason("ASRManager.StartAsrRecognitionLoop: recoverable error but restart not allowed yet")
					resumeAudioIdle()
					continue
				case asr_types.RetryReasonDoubaoWaitingNextPacketTimeout:
					log.Warnf("Doubao ASR session idle timeout; suspending stream until the next utterance")
					state.Asr.CancelWithReason("ASRManager.StartAsrRecognitionLoop: doubao waiting next packet timeout")
					resumeAudioIdle()
					continue
				}
			}

			if text != "" {
				asrFinalTs := time.Now().UnixMilli()
				state.MarkAsrFinalTextAt(asrFinalTs)
				if a.session != nil {
					a.session.TraceAsrFinalText(ctx, asrFinalTs)
				}
				log.Debugf("processed ASR result: %s, duration: %d ms", text, state.GetAsrDuration())

				state.ClearAudioIdleTimeoutPending()
				// Reset empty-result counters after successful recognition.
				emptyResultWindowStart = time.Now()
				emptyResultCount = 0
				recoverableErrorWindowStart = time.Now()
				recoverableErrorCount = 0

				// Stop current LLM and TTS work in realtime mode.
				if state.IsRealTime() && viper.GetInt("chat.realtime_mode") == 2 {
					shouldInterrupt := true
					if a.session != nil && a.session.isRealtimeMcpAudioGateActive() {
						shouldInterrupt = false
						log.Debugf("device %s realtime media playback gate active; deferring interruption to final ASR gate", state.DeviceID)
					}
					if shouldInterrupt {
						log.Debugf("OnListenStart in realtime mode; stopping current LLM and TTS")
						if a.session != nil {
							a.session.StopAssistantOutputAfterAsrWithReason(true, "ASRManager.StartAsrRecognitionLoop realtime_mode=2 ASR result interrupt")
						} else {
							state.AfterAsrSessionCtx.CancelWithReason("ASRManager.StartAsrRecognitionLoop: realtime_mode=2 ASR result interrupt")
						}
					}
				}

				// Reset retry counters.
				startIdleTime = time.Now().Unix()

				// End voice input after receiving an ASR result.
				state.OnVoiceSilence()

				// Get the pending speaker result with a timeout.
				speakerResult := a.getSpeakerResult()
				speakerInterrupted := false
				if a.session != nil {
					speakerInterrupted = a.session.ConsumeTurnSpeakerInterrupted()
				}

				if a.session != nil {
					payload, stop, hookErr := a.session.hookHub.EmitASROutput(a.session.hookContext(ctx), chathooks.ASROutputData{Text: text, SpeakerResult: speakerResult})
					if hookErr != nil {
						log.Warnf("ASR_OUTPUT hook failed: %v", hookErr)
					}
					text = payload.Text
					speakerResult = payload.SpeakerResult
					if stop {
						log.Infof("ASR_OUTPUT hook requested the current flow to stop")
						state.Asr.ClearHistoryAudio()
						if state.UsesAudioIdleClock() {
							startAudioIdle()
						} else {
							state.ResetAudioIdleWindow()
						}
						continue
					}
				}

				if a.session != nil {
					allowChat, denyReason := a.session.ShouldAllowSpeakerChat(speakerResult, speakerInterrupted)
					if !allowChat {
						log.Infof(
							"dropping ASR result and skipping STT/LLM: device=%s, reason=%s, speaker_interrupted=%v, speaker_result=%+v, text=%q",
							state.DeviceID,
							denyReason,
							speakerInterrupted,
							speakerResult,
							text,
						)
						state.Asr.ClearHistoryAudio()

						if !state.IsRealTime() {
							startAudioIdle()
							return
						}
						if restartErr := a.RestartAsrRecognition(ctx); restartErr != nil {
							log.Errorf("failed to restart recognition after dropping ASR result: %v", restartErr)
							if onError != nil {
								onError(restartErr)
							}
							return
						}
						startAudioIdle()
						continue
					}
				}

				// Create the user message from hook-rewritten text.
				userMsg := &schema.Message{
					Role:    schema.User,
					Content: text,
				}

				// Generate a compact MessageID with MD5 to stay within varchar(64).
				// Source format: {SessionID}-{Role}-{Timestamp}.
				rawMessageID := fmt.Sprintf("%s-%s-%d",
					state.SessionID,
					userMsg.Role,
					time.Now().UnixMilli())
				// Generate a fixed 32-character hexadecimal string.
				hash := md5.Sum([]byte(rawMessageID))
				messageID := hex.EncodeToString(hash[:])

				// Add synchronously to memory for LLM context.
				state.AddMessage(userMsg)

				// Get historical ASR audio.
				audioData := state.Asr.GetHistoryAudio()
				state.Asr.ClearHistoryAudio()

				// Persist the message through the callback.
				if onMessageSave != nil {
					onMessageSave(userMsg, messageID, audioData)
				}

				// Send the hook-rewritten ASR result to the client.
				err = a.serverTransport.SendAsrResult(text)
				if err != nil {
					log.Errorf("failed to send ASR message: %v", err)
					if onError != nil {
						onError(err)
					}
					return
				}

				if a.session != nil {
					handledByRealtimeGate, gateErr := a.session.tryHandleRealtimeMcpAudioASR(ctx, text)
					if gateErr != nil {
						log.Warnf("realtime media playback fast control failed: device=%s text=%q err=%v", state.DeviceID, text, gateErr)
					}
					if handledByRealtimeGate {
						if !state.IsRealTime() {
							return
						}
						if restartErr := a.RestartAsrRecognition(ctx); restartErr != nil {
							log.Errorf("failed to restart ASR after realtime media control: %v", restartErr)
							if onError != nil {
								onError(restartErr)
							}
							return
						}
						startAudioIdle()
						continue
					}
				}

				// Add to the queue managed by ASRManager.
				if err := a.addAsrResultToQueue(text, speakerResult); err != nil {
					log.Errorf("failed to start conversation: %v", err)
					if onError != nil {
						onError(err)
					}
					return
				}

				// Return after non-realtime recognition; realtime resource rotation is automatic.
				if !state.IsRealTime() {
					return
				}

				// Restart ASR in realtime mode.
				if restartErr := a.RestartAsrRecognition(ctx); restartErr != nil {
					log.Errorf("failed to restart ASR recognition: %v", restartErr)
					if onError != nil {
						onError(restartErr)
					}
					return
				}
				// Continue processing the next ASR result in realtime mode.
				continue
			} else {
				log.Debugf(
					"empty ASR result details: status=%s, emptyReason=%s, client_voice_stop=%v, history_audio_samples=%d, voice_duration=%dms, voice_duration_in_session=%dms, idle_duration=%dms, realtime=%v",
					state.Status,
					result.EmptyReason,
					state.GetClientVoiceStop(),
					state.Asr.GetHistoryAudioLen(),
					state.Vad.GetVoiceDuration(),
					state.Vad.GetVoiceDurationInSession(),
					state.Vad.GetIdleDuration(),
					state.IsRealTime(),
				)
				if state.AudioIdleTimeoutPending() {
					closeAudioIdleTimeout(result.EmptyReason)
					return
				}
				if result.EmptyReason != "" {
					log.Debugf("empty ASR result classified: reason=%s, status=%s", result.EmptyReason, state.Status)
					emptyResultWindowStart = time.Now()
					emptyResultCount = 0

					if result.EmptyReason == asr_types.EmptyReasonNoServerResponse ||
						result.EmptyReason == asr_types.EmptyReasonProviderEmptyFinal {
						state.Asr.CancelWithReason("ASRManager.StartAsrRecognitionLoop: empty final result from provider")
						resumeAudioIdle()
						continue
					}
				}

				now := time.Now()
				if now.Sub(emptyResultWindowStart) > emptyResultProtectWindow {
					emptyResultWindowStart = now
					emptyResultCount = 0
				}
				emptyResultCount++
				if emptyResultCount >= maxEmptyResultInWindow {
					err := fmt.Errorf("ASR returned %d empty results within %s; triggering protection and disconnecting", emptyResultCount, emptyResultProtectWindow)
					log.Errorf("%v", err)
					if onError != nil {
						onError(err)
					}
					return
				}

				// Handle an empty text result.
				select {
				case <-ctx.Done():
					log.Debugf("asr ctx done")
					return
				default:
				}

				log.Debugf("ready Restart Asr, state.Status: %s", state.Status)
				// Realtime mode may restart ASR during LLMStart or TTSStart.
				// Non-realtime mode may restart only in Listening or ListenStop.
				if isAllowedToRestart() {
					// Reset wait count when restart is allowed.
					invalidStatusWaitCount = 0
					// Check whether empty text requires an ASR restart.
					diffTs := time.Now().Unix() - startIdleTime
					if startIdleTime > 0 && diffTs <= maxIdleTime {
						log.Warnf("ASR result is empty; attempting restart, diff ts: %d", diffTs)
						if restartErr := a.RestartAsrRecognition(ctx); restartErr != nil {
							log.Errorf("failed to restart ASR recognition: %v", restartErr)
							if onError != nil {
								onError(restartErr)
							}
							return
						}
						resumeAudioIdle()
						continue
					} else {
						log.Warnf("ASR result is empty and maximum idle time was reached: %d", maxIdleTime)
						if onError != nil {
							onError(fmt.Errorf("ASR result is empty and maximum idle time was reached: %d", maxIdleTime))
						}
						return
					}
				} else {
					// Briefly wait for state recovery when restart is disallowed.
					invalidStatusWaitCount++
					if invalidStatusWaitCount >= maxInvalidStatusWaitCount {
						// Wait timed out; exit the loop.
						log.Debugf("status %s, realtime: %v, unchanged after %d waits; exiting ASR recognition loop", state.Status, state.IsRealTime(), maxInvalidStatusWaitCount)
						return
					}
					// Continue after a short wait for state recovery.
					log.Debugf("status %s, realtime: %v, restart not allowed; waiting for recovery (%d/%d)", state.Status, state.IsRealTime(), invalidStatusWaitCount, maxInvalidStatusWaitCount)
					time.Sleep(200 * time.Millisecond) // Wait briefly.
					continue
				}
			}
		}
	}()
}

func trimFirstSpeechAudio(allData []float32, currentFrameSamples, sampleRate, channels int) []float32 {
	if len(allData) == 0 {
		return nil
	}
	if currentFrameSamples <= 0 || currentFrameSamples > len(allData) || sampleRate <= 0 || channels <= 0 {
		return allData
	}

	maxPreSpeechSamples := sampleRate * channels * maxFirstSpeechPreAudioMs / 1000
	keepSamples := currentFrameSamples + maxPreSpeechSamples
	if keepSamples >= len(allData) {
		return allData
	}

	audio := make([]float32, keepSamples)
	copy(audio, allData[len(allData)-keepSamples:])
	return audio
}

// getSpeakerResult returns the pending speaker result with a timeout.
func (a *ASRManager) getSpeakerResult() *speaker.IdentifyResult {
	if a.session == nil || a.session.speakerManager == nil {
		return nil
	}

	log.Debugf("speakerManager: %+v, IsActive: %+v", a.session.speakerManager, a.session.speakerManager.IsActive())

	timeout := time.NewTimer(200 * time.Millisecond)
	defer timeout.Stop()

	var speakerResult *speaker.IdentifyResult
	select {
	case <-a.session.speakerResultReady:
		a.session.speakerResultMu.RLock()
		speakerResult = a.session.pendingSpeakerResult
		a.session.speakerResultMu.RUnlock()
	case <-timeout.C:
		// Read the current result after timeout; it may be nil.
		a.session.speakerResultMu.RLock()
		speakerResult = a.session.pendingSpeakerResult
		a.session.speakerResultMu.RUnlock()
		log.Debugf("speaker recognition result timed out; using current result")
	}
	log.Debugf("speaker recognition result: %+v", speakerResult)
	return speakerResult
}

// addAsrResultToQueue adds the ASR result to the ASRManager-owned queue.
func (a *ASRManager) addAsrResultToQueue(text string, speakerResult *speaker.IdentifyResult) error {
	if a.session == nil {
		return fmt.Errorf("session is nil")
	}
	return a.session.AddAsrResultToQueue(text, speakerResult)
}
