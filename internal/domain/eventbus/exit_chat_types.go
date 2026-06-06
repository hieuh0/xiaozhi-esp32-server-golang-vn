package eventbus

import (
	"time"

	. "xiaozhi-esp32-server-golang/internal/data/client"
)

// ExitChatEvent exit chat event
type ExitChatEvent struct {
	//client status
	ClientState *ClientState

	//Reason for exit
	Reason string //"User actively exits", "Tool call exit", "Timeout exit", etc.

	//Exit trigger mode
	TriggerType string //"exit_words" (exit word detection), "tool_call" (tool call), "timeout" (timeout), etc.

	//The original text entered by the user (if any)
	UserText string

	//Timestamp
	Timestamp time.Time
}
