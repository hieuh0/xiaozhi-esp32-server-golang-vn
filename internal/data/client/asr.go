package client

import (
	"bytes"
	"context"
	"strings"
	"sync"
	asr_types "xiaozhi-esp32-server-golang/internal/domain/asr/types"
	log "xiaozhi-esp32-server-golang/logger"
)

type Asr struct {
	lock sync.RWMutex
	// ASR context and channels
	Ctx              context.Context
	Cancel           context.CancelFunc
	AsrEnd           chan bool
	AsrAudioChannel  chan []float32                 // Channel for streaming audio input
	AsrResultChannel chan asr_types.StreamingResult // Channel for streaming ASR result fragments
	AsrResult        bytes.Buffer                   // Stores the final recognized text for this session
	Statue           int                            // 0: initializing  1: recognizing  2: recognition ended
	AutoEnd          bool                           // auto_end means ASR automatically determines end, no longer using the VAD module

	// ASR type and mode
	AsrType string // ASR type, e.g. "funasr", "doubao"
	Mode    string // ASR mode, e.g. "online", "offline"

	// ClientState reference for callback notifications
	ClientState *ClientState

	// Chat history audio buffer: continuously accumulates audio data sent to ASR
	HistoryAudioBuffer []float32

	// Whether the first non-empty text has been received in the current ASR turn
	ReceivedTextInTurn bool
}

func (a *Asr) Reset() {
	a.AsrResult.Reset()
}

func (a *Asr) CancelWithReason(reason string) {
	a.lock.RLock()
	cancel := a.Cancel
	a.lock.RUnlock()

	if cancel != nil {
		log.Debugf("Asr.CancelWithReason: reason=%s", reason)
		cancel()
	}
}

func (a *Asr) RetireAsrResult(ctx context.Context) (asr_types.StreamingResult, bool, error) {
	defer func() {
		a.Reset()
	}()

	log.Log().Debugf("asr type: %s, mode: %s", a.AsrType, a.Mode)

	// Use local variable to track whether the first-text event has been sent
	firstTextSent := false
	var emptyResult asr_types.StreamingResult

	for {
		select {
		case <-ctx.Done():
			log.Debugf("RetireAsrResult: ctx done, exit")
			return emptyResult, false, nil
		default:
			// Avoid probabilistically selecting a channel when ctx is cancelled,
			// which would cause results from an already-cancelled context to be used
			select {
			case result, ok := <-a.AsrResultChannel:
				log.Debugf("asr result: %s, ok: %+v, isFinal: %+v, emptyReason: %s, error: %+v", result.Text, ok, result.IsFinal, result.EmptyReason, result.Error)
				if result.Error != nil {
					if result.RetryReason != "" {
						log.Warnf("ASR returned recoverable error (%s), passing to upper layer for recovery: %v", result.RetryReason, result.Error)
						return result, true, nil
					}
					return emptyResult, false, result.Error
				}

				// Detect first text return (text is non-empty and has not been sent yet)
				if result.Text != "" && !firstTextSent && a.ClientState != nil && a.ClientState.OnAsrFirstTextCallback != nil {
					firstTextSent = true
					// Call callback to notify first text
					a.ClientState.OnAsrFirstTextCallback(result.Text, result.IsFinal)
				}

				if a.AsrType == "funasr" &&
					strings.EqualFold(a.Mode, "2pass") &&
					strings.EqualFold(result.Mode, "2pass-online") {
					if result.IsFinal {
						log.Debugf("funasr 2pass-online result incorrectly marked as final, waiting for 2pass-offline final result")
					}
					continue
				}

				if result.IsFinal {
					return result, true, nil
				}

				if !ok {
					log.Debugf("asr result channel closed")
					return emptyResult, true, nil
				}
			}
		}
	}
}

func (a *Asr) MarkTextReceived() {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.ReceivedTextInTurn = true
}

func (a *Asr) HasReceivedText() bool {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.ReceivedTextInTurn
}

func (a *Asr) ResetReceivedText() {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.ReceivedTextInTurn = false
}

func (a *Asr) StopWithReason(reason string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.AsrAudioChannel != nil {
		log.Debugf("Asr.StopWithReason: reason=%s", reason)
		close(a.AsrAudioChannel) // Close the ASR audio input channel to signal ASR to stop and return results
		a.AsrAudioChannel = nil  // Set to nil since it has been closed
	}
}

func (a *Asr) Stop() {
	a.StopWithReason("Asr.Stop")
}

func (a *Asr) HasOpenAudioInput() bool {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.AsrAudioChannel != nil
}

func (a *Asr) AddAudioData(pcmFrameData []float32) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.AsrAudioChannel != nil {
		// Use select for non-blocking send to avoid deadlock when channel is full
		select {
		case a.AsrAudioChannel <- pcmFrameData:
			// Successfully sent; sync audio data to history buffer for chat history recording
			a.HistoryAudioBuffer = append(a.HistoryAudioBuffer, pcmFrameData...)
		default:
			// Channel is full; skip this data to avoid blocking deadlock
			log.Warnf("AsrAudioChannel is full, skipping audio data")
		}
	}
	return nil
}

// GetHistoryAudio returns the history audio buffer (returns a copy, does not clear original data)
func (a *Asr) GetHistoryAudio() []float32 {
	a.lock.Lock()
	defer a.lock.Unlock()
	if len(a.HistoryAudioBuffer) == 0 {
		return nil
	}
	// Return a copy to avoid external modifications affecting original data
	result := make([]float32, len(a.HistoryAudioBuffer))
	copy(result, a.HistoryAudioBuffer)
	return result
}

// GetHistoryAudioLen returns the length of the history audio buffer (number of samples)
func (a *Asr) GetHistoryAudioLen() int {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return len(a.HistoryAudioBuffer)
}

// ClearHistoryAudio clears the history audio buffer
func (a *Asr) ClearHistoryAudio() {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.HistoryAudioBuffer = nil
}

type AsrAudioBuffer struct {
	PcmData          []float32
	AudioBufferMutex sync.RWMutex
}

func (a *AsrAudioBuffer) AddAsrAudioData(pcmFrameData []float32) error {
	a.AudioBufferMutex.Lock()
	defer a.AudioBufferMutex.Unlock()
	a.PcmData = append(a.PcmData, pcmFrameData...)
	return nil
}

func (a *AsrAudioBuffer) GetAsrDataSize() int {
	a.AudioBufferMutex.RLock()
	defer a.AudioBufferMutex.RUnlock()
	return len(a.PcmData)
}

// GetFrameCount returns the number of frames (requires frame size for calculation)
func (a *AsrAudioBuffer) GetFrameCount(frameSize int) int {
	a.AudioBufferMutex.RLock()
	defer a.AudioBufferMutex.RUnlock()
	if frameSize == 0 {
		return 0
	}
	return len(a.PcmData) / frameSize
}

func (a *AsrAudioBuffer) GetAndClearAllData() []float32 {
	a.AudioBufferMutex.Lock()
	defer a.AudioBufferMutex.Unlock()
	pcmData := make([]float32, len(a.PcmData))
	copy(pcmData, a.PcmData)
	a.PcmData = []float32{}
	return pcmData
}

// GetAsrData retrieves data using a sliding window (requires frame size for calculation)
func (a *AsrAudioBuffer) GetAsrData(frameCount int, frameSize int) []float32 {
	a.AudioBufferMutex.Lock()
	defer a.AudioBufferMutex.Unlock()
	pcmDataLen := len(a.PcmData)
	retSize := frameCount * frameSize
	if pcmDataLen < retSize {
		retSize = pcmDataLen
	}
	pcmData := make([]float32, retSize)
	copy(pcmData, a.PcmData[pcmDataLen-retSize:])
	return pcmData
}

// RemoveAsrAudioData removes the specified number of frames of audio data (requires frame size for calculation)
func (a *AsrAudioBuffer) RemoveAsrAudioData(frameCount int, frameSize int) {
	a.AudioBufferMutex.Lock()
	defer a.AudioBufferMutex.Unlock()
	removeSize := frameCount * frameSize
	if removeSize > len(a.PcmData) {
		removeSize = len(a.PcmData)
	}
	a.PcmData = a.PcmData[removeSize:]
}

func (a *AsrAudioBuffer) ClearAsrAudioData() {
	a.AudioBufferMutex.Lock()
	defer a.AudioBufferMutex.Unlock()
	a.PcmData = nil
}
