package inter

// VAD voice activity detection interface
type VAD interface {
	// IsVAD detects voice activity in audio data
	IsVAD(pcmData []float32) (bool, error)

	IsVADExt(pcmData []float32, sampleRate int, frameSize int) (bool, error)
	// Reset resets the detector state
	Reset() error
	// Close closes and releases resources
	Close() error
	// IsValid checks if the resource is valid
	IsValid() bool
}
