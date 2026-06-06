package aliyun_qwen3

import (
	"encoding/base64"
	"encoding/json"

	log "xiaozhi-esp32-server-golang/logger"
)

// ClientEvent client sends event infrastructure
type ClientEvent struct {
	EventID string   `json:"event_id,omitempty"`
	Type    string   `json:"type"`
	Session *Session `json:"session,omitempty"`
	Audio   string   `json:"audio,omitempty"` //Base64 encoded audio data
}

// Session configuration in session session.update event
type Session struct {
	Modalities              []string                 `json:"modalities"`
	InputAudioFormat        string                   `json:"input_audio_format,omitempty"`
	SampleRate              int                      `json:"sample_rate,omitempty"`
	InputAudioTranscription *InputAudioTranscription `json:"input_audio_transcription,omitempty"`
	TurnDetection           *TurnDetection           `json:"turn_detection"`
}

// InputAudioTranscription audio transcription configuration
type InputAudioTranscription struct {
	Language string `json:"language,omitempty"`
}

// TurnDetection VAD configuration
type TurnDetection struct {
	Type              string  `json:"type,omitempty"`                //"server_vad" or not set
	Threshold         float64 `json:"threshold,omitempty"`           //VAD threshold
	SilenceDurationMs int     `json:"silence_duration_ms,omitempty"` //Silence duration (milliseconds)
}

// ServerEvent server response event infrastructure
type ServerEvent struct {
	Type            string     `json:"type"`
	EventID         string     `json:"event_id,omitempty"`
	PreviousEventID string     `json:"previous_event_id,omitempty"`
	Session         *Session   `json:"session,omitempty"`
	Item            *Item      `json:"item,omitempty"`
	Text            string     `json:"text,omitempty"`
	Stash           string     `json:"stash,omitempty"`
	Transcript      string     `json:"transcript,omitempty"`
	Error           *ErrorInfo `json:"error,omitempty"`
}

// Item session item (such as input audio transcription results)
type Item struct {
	ID            string         `json:"id,omitempty"`
	Type          string         `json:"type,omitempty"`
	Status        string         `json:"status,omitempty"`
	Transcription *Transcription `json:"transcription,omitempty"`
}

// Transcription Transcription results
type Transcription struct {
	Text     string `json:"text,omitempty"`
	Language string `json:"language,omitempty"`
}

// ErrorInfo error message
type ErrorInfo struct {
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// NewSessionUpdateEvent creates session.update event
func NewSessionUpdateEvent(config Config) *ClientEvent {
	session := &Session{
		Modalities:              []string{"text"},
		InputAudioFormat:        config.Format,
		SampleRate:              config.SampleRate,
		InputAudioTranscription: &InputAudioTranscription{Language: config.Language},
	}

	if config.AutoEnd {
		session.TurnDetection = &TurnDetection{
			Type:              "server_vad",
			Threshold:         config.VADThreshold,
			SilenceDurationMs: config.VADSilenceMs,
		}
	} else {
		session.TurnDetection = nil
	}

	event := &ClientEvent{
		EventID: "session_update",
		Type:    "session.update",
		Session: session,
	}

	//Debugging: Print session.update events
	if jsonBytes, err := json.Marshal(event); err == nil {
		log.Debugf("[aliyun_qwen3] session.update JSON: %s", string(jsonBytes))
	}

	return event
}

// NewAudioAppendEvent creates input_audio_buffer.append event
func NewAudioAppendEvent(audioData []byte) *ClientEvent {
	encoded := base64.StdEncoding.EncodeToString(audioData)
	return &ClientEvent{
		Type:  "input_audio_buffer.append",
		Audio: encoded,
	}
}

// NewAudioCommitEvent creates input_audio_buffer.commit event
func NewAudioCommitEvent() *ClientEvent {
	return &ClientEvent{
		EventID: "audio_commit",
		Type:    "input_audio_buffer.commit",
	}
}

// NewSessionFinishEvent creates session.finish event
func NewSessionFinishEvent() *ClientEvent {
	return &ClientEvent{
		EventID: "session_finish",
		Type:    "session.finish",
	}
}

// IsTranscriptionEvent determines whether it is a transcription event
func IsTranscriptionEvent(event *ServerEvent) bool {
	return event.Type == "conversation.item.input_audio_transcription.text" ||
		event.Type == "conversation.item.input_audio_transcription.completed"
}

// IsFinalTranscription determines whether it is the final transcription result
func IsFinalTranscription(event *ServerEvent) bool {
	return event.Type == "conversation.item.input_audio_transcription.completed"
}

// GetTranscriptionText Gets the transcript text
func GetTranscriptionText(event *ServerEvent) string {
	if event == nil {
		return ""
	}
	if event.Item != nil && event.Item.Transcription != nil && event.Item.Transcription.Text != "" {
		return event.Item.Transcription.Text
	}
	if event.Transcript != "" {
		return event.Transcript
	}
	if event.Text != "" {
		return event.Text
	}
	if event.Stash != "" {
		return event.Stash
	}
	return ""
}
