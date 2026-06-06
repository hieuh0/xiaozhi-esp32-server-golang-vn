package chat

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	mcp_go "github.com/mark3labs/mcp-go/mcp"
)

// MCPResponseType defines the type of MCP response
type MCPResponseType string

const (
	// Action class: specific actions need to be performed, usually terminating subsequent processing
	MCPResponseTypeAction MCPResponseType = "action"
	// Audio resource class: specific actions need to be performed, and subsequent processing is usually terminated, and there is no need to return to stop.
	MCPResponseTypeAudio MCPResponseType = "audio"

	// Content class: Return information content, allowing subsequent processing
	MCPResponseTypeContent MCPResponseType = "content"
	// Error classes: handling error situations
	MCPResponseTypeError MCPResponseType = "error"
)

// MCPResponseBase The base structure for all MCP responses
type MCPResponseBase struct {
	Type      MCPResponseType `json:"type"`
	Success   bool            `json:"success"`
	Timestamp int64           `json:"timestamp"`
	ToolName  string          `json:"tool_name"`
}

// MCPActionResponse action response - used for playing music, exiting conversations and other scenarios where actions need to be performed.
type MCPActionResponse struct {
	MCPResponseBase
	Action   string            `json:"action"`
	Message  string            `json:"message"`
	Status   string            `json:"status"`
	Metadata map[string]string `json:"metadata,omitempty"`
	// control flag
	FinalAction       bool   `json:"final_action"`
	NoFurtherResponse bool   `json:"no_further_response"`
	SilenceLLM        bool   `json:"silence_llm"`
	UserState         string `json:"user_state"`
	Instruction       string `json:"instruction,omitempty"`
}

// MCPActionResponse action response - used for playing music, exiting conversations and other scenarios where actions need to be performed.
type MCPAudioResponse struct {
	MCPResponseBase
	Data      []byte            `json:"data"`
	MusicName string            `json:"music_name"`
	Action    string            `json:"action"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	// control flag
	FinalAction bool `json:"final_action"`
}

// MCPContentResponse content class response - used to obtain time, query information and other return data scenarios
type MCPContentResponse struct {
	MCPResponseBase
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// MCPErrorResponse error class response - unified error handling
type MCPErrorResponse struct {
	MCPResponseBase
	Error      string `json:"error"`
	ErrorCode  string `json:"error_code,omitempty"`
	Details    string `json:"details,omitempty"`
	Suggestion string `json:"suggestion,omitempty"` //Advice to users
}

// MCPResponse unified MCP response interface
type MCPResponse interface {
	GetType() MCPResponseType
	GetSuccess() bool
	IsTerminal() bool //Is it a terminal operation?
	ToJSON() (string, error)
	GetContent() []mcp_go.Content
	GetAction() string //Get action type
}

// Implement MCPResponse interface
func (r *MCPActionResponse) GetType() MCPResponseType { return MCPResponseTypeAction }
func (r *MCPActionResponse) GetSuccess() bool         { return r.Success }
func (r *MCPActionResponse) IsTerminal() bool         { return r.FinalAction || r.NoFurtherResponse }
func (r *MCPActionResponse) GetAction() string        { return r.Action }
func (r *MCPActionResponse) GetContent() []mcp_go.Content {
	return []mcp_go.Content{
		mcp_go.TextContent{
			Type: "text",
			Text: r.Message,
		},
	}
}

// Add interface method implementation for MCPAudioResponse
func (r *MCPAudioResponse) GetType() MCPResponseType { return MCPResponseTypeAudio }
func (r *MCPAudioResponse) GetSuccess() bool         { return r.Success }
func (r *MCPAudioResponse) IsTerminal() bool         { return r.FinalAction }
func (r *MCPAudioResponse) GetAction() string        { return r.Action }
func (r *MCPAudioResponse) GetContent() []mcp_go.Content {
	return []mcp_go.Content{
		mcp_go.TextContent{
			Type: "text",
			Text: r.MusicName,
		},
		mcp_go.AudioContent{
			Type:     "audio",
			Data:     base64.StdEncoding.EncodeToString(r.Data),
			MIMEType: "audio/mpeg",
		},
	}
}

func (r *MCPContentResponse) GetType() MCPResponseType { return MCPResponseTypeContent }
func (r *MCPContentResponse) GetSuccess() bool         { return r.Success }
func (r *MCPContentResponse) IsTerminal() bool         { return false } //Content classes usually do not terminate
func (r *MCPContentResponse) GetAction() string        { return "" }    //Content class has no action
func (r *MCPContentResponse) GetContent() []mcp_go.Content {
	return []mcp_go.Content{
		mcp_go.TextContent{
			Type: "text",
			Text: r.Message,
		},
	}
}

func (r *MCPErrorResponse) GetType() MCPResponseType { return MCPResponseTypeError }
func (r *MCPErrorResponse) GetSuccess() bool         { return r.Success }
func (r *MCPErrorResponse) IsTerminal() bool         { return false } //Error classes allow subsequent processing
func (r *MCPErrorResponse) GetAction() string        { return "" }    //Error class has no action
func (r *MCPErrorResponse) GetContent() []mcp_go.Content {
	return []mcp_go.Content{
		mcp_go.TextContent{
			Type: "text",
			Text: r.Error,
		},
	}
}

// ToJSON method implementation
func (r *MCPActionResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	return string(data), err
}

// Add ToJSON method to MCPAudioResponse
func (r *MCPAudioResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	return string(data), err
}

func (r *MCPContentResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	return string(data), err
}

func (r *MCPErrorResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	return string(data), err
}

// convenience constructor

// NewActionResponse creates an action class response
func NewActionResponse(toolName, action, message, status string, terminal bool) *MCPActionResponse {
	return &MCPActionResponse{
		MCPResponseBase: MCPResponseBase{
			Type:      MCPResponseTypeAction,
			Success:   true,
			Timestamp: time.Now().Unix(),
			ToolName:  toolName,
		},
		Action:            action,
		Message:           message,
		Status:            status,
		FinalAction:       terminal,
		NoFurtherResponse: terminal,
		SilenceLLM:        terminal,
	}
}

// NewAudioResponse creates audio class response - corrected return type
func NewAudioResponse(toolName, action, status string, terminal bool, data []byte) *MCPAudioResponse {
	return &MCPAudioResponse{
		MCPResponseBase: MCPResponseBase{
			Type:      MCPResponseTypeAudio,
			Success:   true,
			Timestamp: time.Now().Unix(),
			ToolName:  toolName,
		},
		Data:        data,
		Action:      action,
		Status:      status,
		FinalAction: terminal,
	}
}

// NewContentResponse creates a content class response
func NewContentResponse(toolName string, data interface{}, message string) *MCPContentResponse {
	return &MCPContentResponse{
		MCPResponseBase: MCPResponseBase{
			Type:      MCPResponseTypeContent,
			Success:   true,
			Timestamp: time.Now().Unix(),
			ToolName:  toolName,
		},
		Data:    data,
		Message: message,
	}
}

// NewErrorResponse creates an error class response
func NewErrorResponse(toolName, error, errorCode, suggestion string) *MCPErrorResponse {
	return &MCPErrorResponse{
		MCPResponseBase: MCPResponseBase{
			Type:      MCPResponseTypeError,
			Success:   false,
			Timestamp: time.Now().Unix(),
			ToolName:  toolName,
		},
		Error:      error,
		ErrorCode:  errorCode,
		Suggestion: suggestion,
	}
}

// ParseMCPResponse Parse MCP response from JSON string
func ParseMCPResponse(jsonStr string) (MCPResponse, error) {
	var base MCPResponseBase
	if err := json.Unmarshal([]byte(jsonStr), &base); err != nil {
		return nil, err
	}

	switch base.Type {
	case MCPResponseTypeAction:
		var response MCPActionResponse
		if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
			return nil, err
		}
		return &response, nil
	case MCPResponseTypeAudio:
		var response MCPAudioResponse
		if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
			return nil, err
		}
		return &response, nil
	case MCPResponseTypeContent:
		var response MCPContentResponse
		if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
			return nil, err
		}
		return &response, nil
	case MCPResponseTypeError:
		var response MCPErrorResponse
		if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
			return nil, err
		}
		return &response, nil
	default:
		return NewErrorResponse("unknown", "Unknown response type", "INVALID_TYPE", "Please check tool implementation"), fmt.Errorf("Unknown response type: %s", base.Type)
	}
}
