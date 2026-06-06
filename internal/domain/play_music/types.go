package play_music

import (
	"context"
)

// MusicPlayerInterface music player interface
type MusicPlayerInterface interface {
	//PlayMusicStream plays music from URL and returns audio stream channel
	PlayMusicStream(ctx context.Context, url string) (chan []byte, error)

	//GetPlayerInfo gets player information
	GetPlayerInfo() map[string]interface{}

	//Stop Stop the player
	Stop() error
}

// MusicPlayerConfig music player configuration
type MusicPlayerConfig struct {
	FrameDuration int    `json:"frame_duration"` //Frame duration (ms), default 20ms
	AudioFormat   string `json:"audio_format"`   //Audio format, default "mp3"
}

// DefaultMusicPlayerConfig default music player configuration
func DefaultMusicPlayerConfig() *MusicPlayerConfig {
	return &MusicPlayerConfig{
		FrameDuration: 20,    // 20ms
		AudioFormat:   "mp3", //MP3 format
	}
}

// ToMap converts configuration to map
func (c *MusicPlayerConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"frame_duration": c.FrameDuration,
		"audio_format":   c.AudioFormat,
	}
}

// AudioStreamInfo audio stream information
type AudioStreamInfo struct {
	URL           string `json:"url"`
	Format        string `json:"format"`         //Audio format, such as "mp3", "wav"
	SampleRate    int    `json:"sample_rate"`    //Sampling rate
	Channels      int    `json:"channels"`       //Number of channels
	Duration      int64  `json:"duration"`       //Duration (milliseconds)
	ContentLength int64  `json:"content_length"` //Content length (bytes)
}

// PlaybackStatus playback status
type PlaybackStatus int

const (
	StatusIdle PlaybackStatus = iota
	StatusPlaying
	StatusPaused
	StatusStopped
	StatusError
)

// String Returns the string representation of the status
func (s PlaybackStatus) String() string {
	switch s {
	case StatusIdle:
		return "idle"
	case StatusPlaying:
		return "playing"
	case StatusPaused:
		return "paused"
	case StatusStopped:
		return "stopped"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// PlaybackEvent playback event
type PlaybackEvent struct {
	Type      string      `json:"type"`      //Event types: "started", "progress", "finished", "error"
	Timestamp int64       `json:"timestamp"` //Timestamp
	Message   string      `json:"message"`   //event message
	Data      interface{} `json:"data"`      //extra data
}

// StreamingStats streaming statistics
type StreamingStats struct {
	BytesDownloaded int64          `json:"bytes_downloaded"` //Number of bytes downloaded
	BytesDecoded    int64          `json:"bytes_decoded"`    //Number of bytes decoded
	FramesGenerated int64          `json:"frames_generated"` //Number of frames generated
	StartTime       int64          `json:"start_time"`       //start time
	FirstFrameTime  int64          `json:"first_frame_time"` //First frame time
	Status          PlaybackStatus `json:"status"`           //Current status
	ErrorCount      int            `json:"error_count"`      //number of errors
}
