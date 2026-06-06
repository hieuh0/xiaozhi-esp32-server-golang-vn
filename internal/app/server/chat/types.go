package chat

import (
	"context"

	config_types "xiaozhi-esp32-server-golang/internal/domain/config/types"
)

// ChatSessionOperator defines the ChatSession operation interface required by the local mcp tool
// This interface is used to decouple LLMManager and ChatSession to avoid circular dependencies
type ChatSessionOperator interface {
	// LocalMcpCloseChat Close chat session
	LocalMcpCloseChat() error

	// LocalMcpClearHistory Clear historical conversations
	LocalMcpClearHistory() error

	// LocalMcpPlayMusic plays music
	LocalMcpPlayMusic(ctx context.Context, params *PlayMusicParams) error

	// LocalMcpSwitchDeviceRole switches device roles by role name (supports fuzzy matching)
	LocalMcpSwitchDeviceRole(ctx context.Context, roleName string) (string, error)

	// LocalMcpRestoreDeviceDefaultRole restores the device default role
	LocalMcpRestoreDeviceDefaultRole(ctx context.Context) error

	// LocalMcpSearchKnowledge retrieves the knowledge base associated with the current agent
	LocalMcpSearchKnowledge(ctx context.Context, query string, topK int, knowledgeBaseIDs []uint) ([]config_types.KnowledgeSearchHit, error)

	// LocalMcpControlMusicPlayback controls current session-level media playback
	LocalMcpControlMusicPlayback(ctx context.Context, params *MusicPlaybackControlParams) (*MusicPlaybackControlResult, error)

	// Other operations can be added in the future as needed
	// GetDeviceID() string
	// IsActive() bool
}
