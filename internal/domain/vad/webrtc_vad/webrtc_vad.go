package webrtc_vad

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/domain/vad/inter"

	"github.com/hackers365/go-webrtcvad"
)

const (
	// DefaultSampleRate supported sample rates for WebRTC VAD (8000, 16000, 32000, 48000)
	DefaultSampleRate = 16000
	// DefaultMode VAD sensitivity mode (0: least sensitive, 3: most sensitive)
	DefaultMode = 2
	// FrameDuration frame duration (ms), WebRTC VAD supports 10ms, 20ms, 30ms
	FrameDuration = 20
)

// WebRTCVAD WebRTC VAD implementation, implements the Resource interface
type WebRTCVAD struct {
	webrtcVad      *webrtcvad.VAD
	sampleRate     int          // sample rate
	mode           int          // VAD mode
	frameSize      int          // samples per frame
	frameSizeBytes int          // bytes per frame
	initialized    bool         // whether initialized
	lastUsed       time.Time    // last used time
	mu             sync.RWMutex // read-write lock
}

// AcquireVAD creates and returns a WebRTC VAD instance (managed by global resource pool)
func AcquireVAD(config map[string]interface{}) (inter.VAD, error) {
	vadConfig := getVadConfigFromMap(config)

	vad := &WebRTCVAD{
		sampleRate: vadConfig.SampleRate,
		mode:       vadConfig.Mode,
		lastUsed:   time.Now(),
	}

	// Initialize instance
	if err := vad.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize WebRTC VAD: %w", err)
	}

	return vad, nil
}

// ReleaseVAD releases a VAD instance
func ReleaseVAD(vad inter.VAD) error {
	if vad != nil {
		return vad.Close()
	}
	return nil
}

// NewWebRTCVAD creates a new WebRTC VAD instance
func NewWebRTCVAD() inter.VAD {
	return &WebRTCVAD{
		sampleRate: DefaultSampleRate,
		mode:       DefaultMode,
		lastUsed:   time.Now(),
	}
}

// NewWebRTCVADWithConfig creates a WebRTC VAD instance with the specified configuration
func NewWebRTCVADWithConfig(sampleRate, mode int) (inter.VAD, error) {
	if !isValidSampleRate(sampleRate) {
		return nil, fmt.Errorf("unsupported sample rate: %d, supported rates: 8000, 16000, 32000, 48000", sampleRate)
	}
	if mode < 0 || mode > 3 {
		return nil, fmt.Errorf("invalid VAD mode: %d, must be 0-3", mode)
	}

	vad := &WebRTCVAD{
		sampleRate: sampleRate,
		mode:       mode,
		lastUsed:   time.Now(),
	}

	err := vad.init()
	if err != nil {
		return nil, err
	}

	return vad, nil
}

// init initializes WebRTC VAD
func (w *WebRTCVAD) init() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.initialized {
		return nil
	}

	// Calculate frame size
	w.frameSize = w.sampleRate / 1000 * FrameDuration
	w.frameSizeBytes = w.frameSize * 2 // 16-bit PCM

	// Create VAD instance
	var err error
	w.webrtcVad, err = webrtcvad.New()
	if w.webrtcVad == nil {
		return fmt.Errorf("failed to create WebRTC VAD instance")
	}

	err = w.webrtcVad.SetMode(w.mode)
	if err != nil {
		webrtcvad.Free(w.webrtcVad)
		return fmt.Errorf("failed to set WebRTC VAD mode: %+v", err)
	}

	w.initialized = true
	w.lastUsed = time.Now()
	return nil
}

func (w *WebRTCVAD) IsVAD(pcmData []float32) (bool, error) {
	return w.isVad(pcmData, w.sampleRate, w.frameSize)
}

// isVad detects voice activity in audio data
func (w *WebRTCVAD) isVad(pcmData []float32, sampleRate int, frameSize int) (bool, error) {
	if len(pcmData) == 0 {
		return false, nil
	}

	//log.Debugf("isVad, pcmData len: %d, frameSize: %d", len(pcmData), frameSize)

	// Update last used time
	w.lastUsed = time.Now()

	//pcmBytes := pcmData
	// Convert float32 data to int16 PCM data
	pcmBytes := w.float32ToPCMBytes(pcmData)

	// If data length is insufficient for one frame, return false
	if len(pcmBytes) < frameSize {
		return false, nil
	}

	// Process multiple frames, use result of last frame
	var isActive bool
	var err error

	activityCount := 0
	for i := 0; i+frameSize <= len(pcmBytes); i += frameSize {
		frameData := pcmBytes[i : i+frameSize]

		isActive, err = w.webrtcVad.Process(sampleRate, frameData)
		if err != nil {
			return false, fmt.Errorf("WebRTC VAD process error: %w", err)
		}
		if isActive {
			activityCount++
		}
	}

	frameCount := len(pcmBytes) / frameSize
	isActive = activityCount >= frameCount/2

	//log.Debugf("isVad, isActive: %v, activityCount: %d", isActive, activityCount)
	return isActive, nil
}

func (w *WebRTCVAD) IsVADExt(pcmData []float32, sampleRate int, frameSize int) (bool, error) {
	return w.isVad(pcmData, sampleRate, frameSize)
}

// Reset resets the detector state
func (w *WebRTCVAD) Reset() error {
	return nil
}

// Close closes and releases resources (implements Resource interface)
func (w *WebRTCVAD) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.initialized && w.webrtcVad != nil {
		webrtcvad.Free(w.webrtcVad)
		w.initialized = false
	}
	return nil
}

// IsValid checks if the resource is valid (implements Resource interface)
func (w *WebRTCVAD) IsValid() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.initialized && w.webrtcVad != nil
}

// float32ToPCMBytes converts float32 array to 16-bit PCM byte array
func (w *WebRTCVAD) float32ToPCMBytes(samples []float32) []byte {
	pcmBytes := make([]byte, len(samples)*2)

	for i, sample := range samples {
		// Convert float32 (-1.0 to 1.0) to int16 (-32768 to 32767)
		var intSample int16
		if sample > 1.0 {
			intSample = 32767
		} else if sample < -1.0 {
			intSample = -32768
		} else {
			intSample = int16(sample * 32767)
		}

		// Write in little-endian byte order
		binary.LittleEndian.PutUint16(pcmBytes[i*2:], uint16(intSample))
	}

	return pcmBytes
}

// isValidSampleRate checks if the sample rate is supported by WebRTC VAD
func isValidSampleRate(sampleRate int) bool {
	validRates := []int{8000, 16000, 32000, 48000}
	for _, rate := range validRates {
		if rate == sampleRate {
			return true
		}
	}
	return false
}

// SetMode sets the VAD sensitivity mode
func (w *WebRTCVAD) SetMode(mode int) error {
	if mode < 0 || mode > 3 {
		return fmt.Errorf("invalid VAD mode: %d, must be 0-3", mode)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.mode = mode

	if w.initialized {
		return w.webrtcVad.SetMode(mode)
	}

	return nil
}

// SetSampleRate sets the sample rate
func (w *WebRTCVAD) SetSampleRate(sampleRate int) error {
	if !isValidSampleRate(sampleRate) {
		return fmt.Errorf("unsupported sample rate: %d, supported rates: 8000, 16000, 32000, 48000", sampleRate)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// If already initialized, need to re-initialize
	if w.initialized {
		w.Close()
	}

	w.sampleRate = sampleRate
	return nil
}

// GetSampleRate returns the current sample rate
func (w *WebRTCVAD) GetSampleRate() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.sampleRate
}

// GetMode returns the current VAD mode
func (w *WebRTCVAD) GetMode() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.mode
}

// GetLastUsed returns the last used time
func (w *WebRTCVAD) GetLastUsed() time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastUsed
}
