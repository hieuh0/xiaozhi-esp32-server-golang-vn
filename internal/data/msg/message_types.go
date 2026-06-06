package msg

import (
	"encoding/json"

	types_audio "xiaozhi-esp32-server-golang/internal/data/audio"
)

const (
	MDeviceMockPubTopicPrefix = "device-server"
	MDeviceMockSubTopicPrefix = "null"
	MDeviceSubTopicPrefix     = "/p2p/device_sub/"
	MDevicePubTopicPrefix     = "/p2p/device_public/"
	MDeviceLifecycleTopic     = MDevicePubTopicPrefix + "_server/lifecycle"
	MServerSubTopicPrefix     = "/p2p/device_public/#"
	MServerPubTopicPrefix     = MDeviceSubTopicPrefix
)

const (
	MqttLifecycleType         = "mqtt_lifecycle"
	MqttLifecycleStateOnline  = "online"
	MqttLifecycleStateOffline = "offline"
)

// Message type constants
const (
	MessageTypeHello      = "hello"       // Handshake message
	MessageTypeAbort      = "abort"       // Abort message
	MessageTypeListen     = "listen"      // Listen message
	MessageTypeIot        = "iot"         // IoT message
	MessageTypeMcp        = "mcp"         // MCP message
	MessageTypeGoodBye    = "goodbye"     // Goodbye message
	MessageTypeSpeakReady = "speak_ready" // Device is ready to receive proactive broadcast
)

// Server message type constants
const (
	ServerMessageTypeHello        = "hello"         // Handshake message
	ServerMessageTypeStt          = "stt"           // Speech-to-text
	ServerMessageTypeTts          = "tts"           // Text-to-speech
	ServerMessageTypeIot          = "iot"           // IoT message
	ServerMessageTypeLlm          = "llm"           // Large language model
	ServerMessageTypeText         = "text"          // Text message
	ServerMessageTypeGoodBye      = "goodbye"       // Goodbye message
	ServerMessageTypeSpeakRequest = "speak_request" // Proactive broadcast request
)

// Message state constants
const (
	MessageStateStart         = "start"          // Start state
	MessageStateSentenceStart = "sentence_start" // Sentence start state
	MessageStateSentenceEnd   = "sentence_end"   // Sentence end state
	MessageStateStop          = "stop"           // Stop state
	MessageStateDetect        = "detect"         // Detect state
	MessageStateAbort         = "abort"          // Abort state
	MessageStateSuccess       = "success"        // Success state
	MessageStateReady         = "ready"          // Device ready state
)

type UdpConfig struct {
	Server string `json:"server"`
	Port   int    `json:"port"`
	Key    string `json:"key"`
	Nonce  string `json:"nonce"`
}

type MqttLifecycleEvent struct {
	Type     string `json:"type"`
	DeviceID string `json:"device_id"`
	State    string `json:"state"`
	ClientID string `json:"client_id,omitempty"`
	Ts       int64  `json:"ts"`
}

// ServerMessage represents a server message
type ServerMessage struct {
	Type        string                   `json:"type"`
	Text        string                   `json:"text,omitempty"`
	SessionID   string                   `json:"session_id,omitempty"`
	Version     int                      `json:"version"`
	State       string                   `json:"state,omitempty"`
	Transport   string                   `json:"transport,omitempty"`
	AudioFormat *types_audio.AudioFormat `json:"audio_params,omitempty"`
	Emotion     string                   `json:"emotion,omitempty"`
	AutoListen  *bool                    `json:"auto_listen,omitempty"`
	Udp         *UdpConfig               `json:"udp,omitempty"`
	PayLoad     json.RawMessage          `json:"payload,omitempty"`
}
