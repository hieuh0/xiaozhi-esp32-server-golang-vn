package speaker

import (
	"context"
)

// SpeakerProvider voiceprint recognition provider interface
type SpeakerProvider interface {
	//StartStreaming starts streaming recognition
	StartStreaming(ctx context.Context, sampleRate int, agentId string) error

	//SendAudioChunk sends audio data chunks
	SendAudioChunk(ctx context.Context, audioData []float32) error

	//FinishAndIdentify completes input and obtains identification results
	FinishAndIdentify(ctx context.Context) (*IdentifyResult, error)

	//IsActive checks whether it is active
	IsActive() bool

	//Close closes the connection
	Close() error
}

// GetSpeakerProvider Gets the voiceprint recognition provider
func GetSpeakerProvider(config map[string]interface{}) (SpeakerProvider, error) {
	return NewAsrServerProvider(config)
}
