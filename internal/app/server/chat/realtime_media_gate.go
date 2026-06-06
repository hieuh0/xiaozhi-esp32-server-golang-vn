package chat

import (
	"context"
	"strings"
	"time"

	"xiaozhi-esp32-server-golang/internal/domain/eventbus"
	"xiaozhi-esp32-server-golang/internal/domain/play_music"
	log "xiaozhi-esp32-server-golang/logger"
)

type realtimeMusicControlRule struct {
	action   string
	keywords []string
}

var realtimeMcpAudioControlRules = []realtimeMusicControlRule{
	{
		action: "play_playlist",
		keywords: []string{
			"play playlist",
			"play the playlist",
			"play songs in playlist",
		},
	},
	{
		action: "enqueue_current",
		keywords: []string{
			"add to playlist",
			"add to queue",
			"enqueue current",
		},
	},
	{
		action: "resume",
		keywords: []string{
			"continue playing",
			"resume playing",
			"resume music",
			"keep playing",
			"play on",
		},
	},
	{
		action: "pause",
		keywords: []string{
			"pause",
			"pause music",
			"hold on",
		},
	},
	{
		action: "stop",
		keywords: []string{
			"stop playing",
			"stop music",
			"stop",
		},
	},
	{
		action: "next",
		keywords: []string{
			"next song",
			"next track",
			"skip",
			"skip song",
		},
	},
	{
		action: "prev",
		keywords: []string{
			"previous song",
			"previous track",
			"go back",
		},
	},
}

var realtimeMcpAudioExitKeywords = []string{
	"goodbye",
	"bye bye",
	"bye",
	"see you",
	"exit",
	"exit conversation",
	"farewell",
}

func normalizeRealtimeMcpAudioText(text string) string {
	return removePunctuation(strings.ToLower(strings.TrimSpace(text)))
}

func detectRealtimeMcpAudioControlAction(text string) string {
	normalizedText := normalizeRealtimeMcpAudioText(text)
	if normalizedText == "" {
		return ""
	}

	for _, rule := range realtimeMcpAudioControlRules {
		for _, keyword := range rule.keywords {
			normalizedKeyword := normalizeRealtimeMcpAudioText(keyword)
			if normalizedKeyword == "" {
				continue
			}
			if strings.Contains(normalizedText, normalizedKeyword) {
				return rule.action
			}
		}
	}

	return ""
}

func isRealtimeMcpAudioExitCommand(text string) bool {
	normalizedText := normalizeRealtimeMcpAudioText(text)
	if normalizedText == "" {
		return false
	}

	for _, keyword := range realtimeMcpAudioExitKeywords {
		normalizedKeyword := normalizeRealtimeMcpAudioText(keyword)
		if normalizedKeyword == "" {
			continue
		}
		if strings.Contains(normalizedText, normalizedKeyword) {
			return true
		}
	}

	return false
}

func isRealtimeMcpAudioSourceType(sourceType MediaSourceType) bool {
	return sourceType == MediaSourceTypeMCPResource || sourceType == MediaSourceTypeInlineAudio
}

func isRealtimeMcpAudioPlaybackState(state MediaPlayerState) bool {
	if !isRealtimeMcpAudioSourceType(state.CurrentSourceType) {
		return false
	}

	return state.Status == play_music.StatusPlaying
}

func (s *ChatSession) hasRealtimeMcpAudioControlContext() bool {
	if s == nil || s.clientState == nil || !s.clientState.IsRealTime() || s.mediaPlayer == nil {
		return false
	}

	return s.mediaPlayer.HasRealtimeMcpAudioControlContext()
}

func (s *ChatSession) isRealtimeMcpAudioGateActive() bool {
	if s == nil || s.clientState == nil || !s.clientState.IsRealTime() || s.mediaPlayer == nil {
		return false
	}

	return s.mediaPlayer.ShouldGateRealtimeMcpAudioASR()
}

func (s *ChatSession) tryHandleRealtimeMcpAudioASR(ctx context.Context, text string) (bool, error) {
	if !s.hasRealtimeMcpAudioControlContext() {
		return false, nil
	}

	if isRealtimeMcpAudioExitCommand(text) {
		eventbus.Get().Publish(eventbus.TopicExitChat, &eventbus.ExitChatEvent{
			ClientState: s.clientState,
			Reason:      "User exited during realtime media playback",
			TriggerType: "realtime_media_exit_words",
			UserText:    text,
			Timestamp:   time.Now(),
		})
		log.Infof("Device %s realtime media playback gate hit exit command: %s", s.clientState.DeviceID, text)
		return true, nil
	}

	action := detectRealtimeMcpAudioControlAction(text)
	if action != "" {
		_, err := controlMusicPlayback(ctx, s, &MusicPlaybackControlParams{Action: action})
		if err != nil {
			log.Warnf("Device %s realtime media playback gate failed to execute control action: action=%s, text=%s, err=%v", s.clientState.DeviceID, action, text, err)
			return true, nil
		}
		log.Infof("Device %s realtime media playback gate execution control action: action=%s, text=%s", s.clientState.DeviceID, action, text)
		return true, nil
	}

	if !s.isRealtimeMcpAudioGateActive() {
		return false, nil
	}

	log.Debugf("Device %s realtime media playback gate ignores ASR text: %s", s.clientState.DeviceID, text)
	return true, nil
}
