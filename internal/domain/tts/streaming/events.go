package streaming

// SentenceSignalType represents the type of sentence-level control signal that needs to be sent before a piece of audio.
type SentenceSignalType string

const (
	SentenceSignalStart SentenceSignalType = "sentence_start"
	SentenceSignalEnd   SentenceSignalType = "sentence_end"
)

// SentenceSignal represents an ordered sentence boundary signal bound to the current audio block.
type SentenceSignal struct {
	Type SentenceSignalType
	Text string
}

// SynthesisEvent represents a piece of dual-stream TTS output.
// Audio is the current audio block; SentenceSignals represents the sentence boundary signals that need to be sent before sending this audio block.
type SynthesisEvent struct {
	Audio           []byte
	SentenceSignals []SentenceSignal
}
