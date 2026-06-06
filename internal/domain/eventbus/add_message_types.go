package eventbus

import (
	"time"
	. "xiaozhi-esp32-server-golang/internal/data/client"

	"github.com/cloudwego/eino/schema"
)

// AddMessageEvent unified message addition event
type AddMessageEvent struct {
	//client status
	ClientState *ClientState

	//Message content (use schema.Message uniformly)
	//schema.Message is a standard LLM message format, including:
	//- Role: Message role (User/Assistant/System/Tool)
	//- Content: message text content
	//- ToolCalls: Tool call list (optional)
	//- ToolCallID: tool call ID (used by Tool role)
	Msg schema.Message

	//Message ID (used to associate two-stage saves)
	MessageID string

	//Audio data (optional, not part of the schema.Message standard format)
	//First stage: AudioData = nil (only save text)
	//Second stage: AudioData != nil (update audio)
	AudioData [][]byte //TTS/ASR audio frame array (Opus format or PCM format)
	AudioSize int      //Audio size (bytes)

	//Audio format information (not part of the schema.Message standard format)
	SampleRate int //Sampling rate
	Channels   int //Number of channels

	//Metadata (not part of the schema.Message standard format)
	Timestamp   time.Time
	TTSDuration int //TTS time taken (milliseconds)

	//stage identification
	IsUpdate bool //true=update audio, false=add new message
}
