package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	log "xiaozhi-esp32-server-golang/logger"
)

const localMcpMusicControlToolName = "control_music_playback"

func init() {
	if err := RegisterLocalMcpFunc(
		localMcpMusicControlToolName,
		"Must be used when the user wants to control the music or audio currently playing on the device. For commands like 'resume playing', 'continue listening', 'pause', 'stop', 'next song', 'previous song', 'play playlist', 'add current to playlist', this tool must be called - do not just reply with text. Only do NOT use this tool when the user wants to play a new song, search for songs, or request specific music.",
		MusicPlaybackControlParams{},
		musicPlaybackControlHandler,
	); err != nil {
		log.Errorf("Failed to register media control local MCP tool: %v", err)
	}
}

func musicPlaybackControlHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Infof("Execute media control tool, args=%s", argumentsInJSON)

	var params MusicPlaybackControlParams
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			response := NewErrorResponse(localMcpMusicControlToolName, "Parameter parsing failed", "PARSE_ERROR", "Please check action parameter format")
			return response.ToJSON()
		}
	}

	chatSessionOperatorValue := ctx.Value("chat_session_operator")
	if chatSessionOperatorValue == nil {
		return "", fmt.Errorf("chat_session_operator not found from context")
	}

	chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator)
	if !ok {
		return "", fmt.Errorf("The chat_session_operator obtained from context is not of type ChatSessionOperator")
	}

	result, err := chatSessionOperator.LocalMcpControlMusicPlayback(ctx, &params)
	if err != nil {
		log.Errorf("Media control failed: %v", err)
		response := NewErrorResponse(localMcpMusicControlToolName, fmt.Sprintf("Media control failed: %v", err), "MEDIA_CONTROL_FAILED", "Please check current playback status and retry")
		return response.ToJSON()
	}
	if result == nil {
		result = &MusicPlaybackControlResult{
			Action:          normalizeMusicPlaybackAction(params.Action),
			Status:          "unknown",
			SilenceResponse: true,
		}
	}

	action := normalizeMusicPlaybackAction(params.Action)
	if result != nil && result.Action != "" {
		action = result.Action
	}

	response := NewActionResponse(
		localMcpMusicControlToolName,
		action,
		buildMusicPlaybackControlMessage(result),
		result.Status,
		false,
	)
	response.NoFurtherResponse = result.SilenceResponse
	response.SilenceLLM = result.SilenceResponse
	response.Metadata = buildMusicPlaybackControlMetadata(result)

	return response.ToJSON()
}

func buildMusicPlaybackControlMessage(result *MusicPlaybackControlResult) string {
	if result == nil {
		return "Media control completed"
	}

	switch result.Action {
	case "resume":
		if result.CurrentTitle != "" {
			return fmt.Sprintf("Resumed: %s", result.CurrentTitle)
		}
		return "Resumed playback"
	case "pause":
		if result.CurrentTitle != "" {
			return fmt.Sprintf("Paused: %s", result.CurrentTitle)
		}
		return "Playback paused"
	case "stop":
		if result.CurrentTitle != "" {
			return fmt.Sprintf("Stopped: %s", result.CurrentTitle)
		}
		return "Playback stopped"
	case "prev":
		if result.CurrentTitle != "" {
			return fmt.Sprintf("Switched to previous: %s", result.CurrentTitle)
		}
		return "Switched to previous track"
	case "next":
		if result.CurrentTitle != "" {
			return fmt.Sprintf("Switched to next: %s", result.CurrentTitle)
		}
		return "Switched to next track"
	case "play_playlist":
		if result.CurrentTitle != "" {
			return fmt.Sprintf("Started playing playlist: %s", result.CurrentTitle)
		}
		return "Started playing playlist"
	case "enqueue_current":
		if result.AddedTitle != "" {
			return fmt.Sprintf("Added current to playlist: %s", result.AddedTitle)
		}
		return "Added current to playlist"
	default:
		return "Media control completed"
	}
}

func buildMusicPlaybackControlMetadata(result *MusicPlaybackControlResult) map[string]string {
	if result == nil {
		return nil
	}

	metadata := map[string]string{
		"action":          result.Action,
		"status":          result.Status,
		"current_title":   result.CurrentTitle,
		"current_index":   strconv.Itoa(result.CurrentIndex),
		"playlist_length": strconv.Itoa(result.PlaylistLength),
		"current_source":  result.CurrentSource,
		"position_ms":     strconv.FormatInt(result.PositionMs, 10),
	}
	if result.AddedTitle != "" {
		metadata["added_title"] = result.AddedTitle
	}
	return metadata
}
