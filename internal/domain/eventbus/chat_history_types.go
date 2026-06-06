package eventbus

import (
	"context"
	"time"
)

// UserMessageEvent User message event
// Deprecated: Use AddMessageEvent instead, and use TopicAddMessage event uniformly
type UserMessageEvent struct {
	Ctx       context.Context
	SessionID string
	DeviceID  string
	AgentID   string

	//ASR results
	Text      string
	AudioData []byte //Raw audio data (PCM float32 to bytes)
	AudioSize int    //Number of audio samples

	//Audio format information (for conversion to WAV)
	SampleRate int //Sampling rate
	Channels   int //Number of channels

	//metadata
	Timestamp time.Time
}

// AssistantMessageEvent robot reply event
// Deprecated: Use AddMessageEvent instead, and use TopicAddMessage event uniformly
type AssistantMessageEvent struct {
	Ctx       context.Context
	SessionID string
	DeviceID  string
	AgentID   string

	//LLM results
	Text string

	//TTS results
	AudioData [][]byte //Synthesized audio data (Opus format, audio frame array)
	AudioSize int      //Audio size (bytes)

	//Audio format information (for conversion to WAV)
	SampleRate int //Sampling rate
	Channels   int //Number of channels

	//metadata
	TTSDuration int //milliseconds
	Timestamp   time.Time
}
