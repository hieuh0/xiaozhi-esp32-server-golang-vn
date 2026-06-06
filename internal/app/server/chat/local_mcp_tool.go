package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	mcp_manager "xiaozhi-esp32-server-golang/internal/domain/mcp"
	log "xiaozhi-esp32-server-golang/logger"

	//"github.com/scroot/music-sd/pkg/netease"
	//"github.com/scroot/music-sd/pkg/qq"
	"github.com/spf13/viper"
)

type LocalMcpTool struct {
	Name        string
	Description string
	Params      any
	Handle      mcp_manager.LocalToolHandler
}

// InitChatLocalMCPTools initializes chat-related local MCP tools
func InitChatLocalMCPTools() {
	manager := mcp_manager.GetLocalMCPManager()

	log.Info("Initialize chat-related local MCP tools...")

	localTools := map[string]LocalMcpTool{
		/*"get_current_datetime": {
			Name:        "get_current_datetime",
			Description: "Get current date and time information",
			Params:      struct{}{},
			Handle:      getCurrentDateTimeHandler,
		},*/
		"exit_conversation": {
			Name:        "exit_conversation",
			Description: "Use when the user explicitly indicates they want to end the conversation, exit the system, or say goodbye - gracefully closes the current chat session",
			Params:      struct{}{},
			Handle:      exitConversationHandler,
		},
		"clear_conversation_history": {
			Name:        "clear_conversation_history",
			Description: "Use when the user requests to clear, erase or reset the conversation history - clears all history in the current session",
			Params:      struct{}{},
			Handle:      clearConversationHistoryHandler,
		},
		"switch_device_role": {
			Name:        "switch_device_role",
			Description: "Use when the user wants to switch the current device to a specific role; the role_name parameter supports fuzzy matching (matched against global roles and user roles for this device)",
			Params:      SwitchDeviceRoleParams{},
			Handle:      switchDeviceRoleHandler,
		},
		"restore_device_default_role": {
			Name:        "restore_device_default_role",
			Description: "Use when the user wants to restore the device's default role or cancel the current role override",
			Params:      struct{}{},
			Handle:      restoreDeviceDefaultRoleHandler,
		},
		"search_knowledge": {
			Name:        "search_knowledge",
			Description: "Use when user questions require factual basis, process rules, parameter details, or document clauses - searches the agent's linked knowledge bases and returns relevant excerpts; optionally pass knowledge_base_ids to search specific bases; do not call for casual chat or purely creative tasks",
			Params:      SearchKnowledgeParams{},
			Handle:      searchKnowledgeHandler,
		},
		/*"play_music": {
			Name:        "play_music",
			Description: "Use when the user wants to listen to music or relax - plays music by name. When user wants any random music, recommend a specific song title. Prefer this tool when multiple music tools exist. **This tool call takes longer, return a friendly transition message first**",
			Params:      PlayMusicParams{},
			Handle:      playMusicHandler,
		},*/
	}

	for toolName, localTool := range localTools {
		//Only skipped if the configuration is explicitly set to false, enabled if the configuration does not exist or is true
		if viper.IsSet("local_mcp."+toolName) && !viper.GetBool("local_mcp."+toolName) {
			continue
		}
		err := manager.RegisterToolFunc(
			localTool.Name,
			localTool.Description,
			localTool.Params,
			localTool.Handle,
		)
		if err != nil {
			log.Errorf("Failed to register local MCP tool %s: %+v", toolName, err)
		}
	}

	log.Info("Chat-related local MCP tool initialization completed")
}

func RegisterLocalMcpFunc(name string, description string, params any, handle mcp_manager.LocalToolHandler) error {
	manager := mcp_manager.GetLocalMCPManager()

	err := manager.RegisterToolFunc(
		name,
		description,
		params,
		handle,
	)
	if err != nil {
		log.Errorf("Failed to register local MCP tool %s: %+v", name, err)
		return err
	}
	return nil
}

type SwitchDeviceRoleParams struct {
	RoleName string `json:"role_name" description:"Target role name, supports fuzzy matching" required:"true"`
}

type SearchKnowledgeParams struct {
	Query            string `json:"query" description:"Query content to search" required:"true"`
	TopK             int    `json:"top_k,omitempty" description:"Number of results to return, default 5"`
	KnowledgeBaseIDs []uint `json:"knowledge_base_ids,omitempty" description:"Optional: search only within these knowledge base IDs (linked to current agent)"`
}

// playMusicHandler processing function for playing music
func playMusicHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Execute play music tool")

	//Parse parameters
	var params PlayMusicParams

	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			response := NewErrorResponse("play_music", "Parameter parsing failed", "PARSE_ERROR", "Please check parameter format")
			return response.ToJSON()
		}
	}

	log.Infof("Found ChatSessionOperator, calling LocalMcpPlayMusic method to play music: %s", params.Name)
	audioData, realMusicName, err := GetMusicAudioData(ctx, &params)
	if err != nil {
		log.Errorf("Failed to obtain music data: %v", err)
		response := NewErrorResponse("play_music", fmt.Sprintf("Failed to get music data: %v", err), "PLAYBACK_ERROR", "Please check the music name or network connection")
		return response.ToJSON()
	} else {
		//Successful playback - action response, terminating subsequent processing
		response := NewAudioResponse("play_music", "play_music", fmt.Sprintf("Playing music: %s", realMusicName), true, audioData)
		response.MusicName = realMusicName
		return response.ToJSON()
	}

}

/*
// getCurrentDateTimeHandler handles current date and time requests.
func getCurrentDateTimeHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Execute current date and time tool")

	// Parse parameters.
	var params map[string]interface{}
	timezone := "Local" // Default time zone.

	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err == nil {
			if tz, ok := params["timezone"].(string); ok && tz != "" {
				timezone = tz
			}
		}
	}

	now := time.Now()

	// Try to parse the requested time zone.
	if timezone != "Local" {
		if loc, err := time.LoadLocation(timezone); err == nil {
			now = now.In(loc)
		} else {
			log.Warnf("Failed to load time zone %s; using local time zone", timezone)
		}
	}

	// Build the response data.
	data := map[string]interface{}{
		"datetime": map[string]interface{}{
			"formatted":     now.Format("2006-01-02 15:04:05"),
			"iso8601":       now.Format(time.RFC3339),
			"chinese":       formatChineseDateTime(now),
			"unix":          now.Unix(),
			"year":          now.Year(),
			"month":         int(now.Month()),
			"day":           now.Day(),
			"hour":          now.Hour(),
			"minute":        now.Minute(),
			"second":        now.Second(),
			"weekday":       now.Weekday().String(),
			"weekday_en":    getWeekdayEnglish(now.Weekday()),
			"week_number":   getWeekNumber(now),
			"timezone":      timezone,
			"timezone_name": now.Location().String(),
		},
	}

	// Create a content response.
	response := NewContentResponse("get_current_datetime", data, fmt.Sprintf("Current time: %s", formatDateTime(now)))
	// response.Format = "datetime"
	// response.DisplayHint = "Can be used to display current date and time information"

	log.Infof("Current date and time retrieved: %s", now.Format("2006-01-02 15:04:05"))
	return response.ToJSON(),nil
}
*/
//exitConversationHandler exit conversation processing function
func exitConversationHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Execute exit dialog tool")

	//Parse parameters
	var params map[string]interface{}
	reason := "User initiated exit" // Default reason

	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err == nil {
			if r, ok := params["reason"].(string); ok && r != "" {
				reason = r
			}
		}
	}

	//Create action-like responses - terminal operations
	response := NewActionResponse("exit_conversation", "exit_conversation", "Conversation ending, thank you for using!", "exiting", true)
	response.UserState = "conversation_ended"
	response.Instruction = "Conversation ended, do not generate additional text responses"
	response.Metadata = map[string]string{
		"reason":    reason,
		"exit_code": "0",
		"farewell":  "Goodbye! Looking forward to our next conversation.",
	}

	log.Infof("Exit dialog processing completed, reason: %s", reason)

	//Get the ChatSessionOperator from context and call the Close method
	if chatSessionOperatorValue := ctx.Value("chat_session_operator"); chatSessionOperatorValue != nil {
		if chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator); ok {
			log.Info("The ChatSessionOperator is found and the Close method is being called to close the session.")
			defer chatSessionOperator.LocalMcpCloseChat()
		} else {
			log.Warn("The chat_session_operator obtained from context is not of type ChatSessionOperator")
		}
	} else {
		log.Warn("chat_session_operator not found from context")
	}

	responseStr, err := response.ToJSON()
	if err != nil {
		return "", err
	}

	return responseStr, nil
}

// clearConversationHistoryHandler clears the processing function of historical conversations
func clearConversationHistoryHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Execute Clear History Conversation Tool")

	//Parse parameters
	var params map[string]interface{}
	reason := "User cleared history" // Default reason

	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err == nil {
			if r, ok := params["reason"].(string); ok && r != "" {
				reason = r
			}
		}
	}

	//Get the ChatSessionOperator from context and call the LocalMcpClearHistory method
	if chatSessionOperatorValue := ctx.Value("chat_session_operator"); chatSessionOperatorValue != nil {
		if chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator); ok {
			log.Info("Found ChatSessionOperator, calling LocalMcpClearHistory method to clear history")
			if err := chatSessionOperator.LocalMcpClearHistory(); err != nil {
				log.Errorf("Failed to clear conversation history: %v", err)
				return "", err
			} else {
				//Cleared successfully - Action-like response, but does not terminate the conversation
				response := NewActionResponse("clear_conversation_history", "clear_history", "Conversation history cleared, you can start a new conversation.", "completed", false)
				response.Metadata = map[string]string{
					"reason": reason,
					"status": "cleared",
				}
				log.Info("History conversation cleared successfully")

				return response.ToJSON()
			}
		} else {
			log.Warn("The chat_session_operator obtained from context is not of type ChatSessionOperator")
			return "", fmt.Errorf("The chat_session_operator obtained from context is not of type ChatSessionOperator")
		}
	}
	log.Warn("chat_session_operator not found from context")
	return "", fmt.Errorf("chat_session_operator not found from context")
}

// switchDeviceRoleHandler handler function for switching device roles
func switchDeviceRoleHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Execute the Switch Device Role tool")

	var params SwitchDeviceRoleParams
	if argumentsInJSON == "" {
		response := NewErrorResponse("switch_device_role", "Missing parameter role_name", "MISSING_ROLE_NAME", "Please provide the role name to switch to")
		return response.ToJSON()
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		response := NewErrorResponse("switch_device_role", "Parameter parsing failed", "PARSE_ERROR", "Please check role_name parameter format")
		return response.ToJSON()
	}
	params.RoleName = strings.TrimSpace(params.RoleName)
	if params.RoleName == "" {
		response := NewErrorResponse("switch_device_role", "Role name cannot be empty", "INVALID_ROLE_NAME", "Please provide a valid role_name")
		return response.ToJSON()
	}

	if chatSessionOperatorValue := ctx.Value("chat_session_operator"); chatSessionOperatorValue != nil {
		if chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator); ok {
			matchedRoleName, err := chatSessionOperator.LocalMcpSwitchDeviceRole(ctx, params.RoleName)
			if err != nil {
				log.Errorf("Failed to switch device roles: %v", err)
				response := NewErrorResponse("switch_device_role", fmt.Sprintf("Failed to switch role: %v", err), "SWITCH_ROLE_FAILED", "Please try a different role name or retry later")
				return response.ToJSON()
			}

			response := NewActionResponse(
				"switch_device_role",
				"switch_device_role",
				fmt.Sprintf("Switched to role: %s", matchedRoleName),
				"completed",
				false,
			)
			response.Metadata = map[string]string{
				"requested_role_name": params.RoleName,
				"matched_role_name":   matchedRoleName,
			}
			return response.ToJSON()
		}
		return "", fmt.Errorf("The chat_session_operator obtained from context is not of type ChatSessionOperator")
	}

	return "", fmt.Errorf("chat_session_operator not found from context")
}

// restoreDeviceDefaultRoleHandler handler function to restore the device's default role
func restoreDeviceDefaultRoleHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Execute the restore device default role tool")

	if chatSessionOperatorValue := ctx.Value("chat_session_operator"); chatSessionOperatorValue != nil {
		if chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator); ok {
			if err := chatSessionOperator.LocalMcpRestoreDeviceDefaultRole(ctx); err != nil {
				log.Errorf("Failed to restore device default role: %v", err)
				response := NewErrorResponse("restore_device_default_role", fmt.Sprintf("Failed to restore default role: %v", err), "RESTORE_ROLE_FAILED", "Please try again later")
				return response.ToJSON()
			}

			response := NewActionResponse(
				"restore_device_default_role",
				"restore_device_default_role",
				"Device default role restored",
				"completed",
				false,
			)
			return response.ToJSON()
		}
		return "", fmt.Errorf("The chat_session_operator obtained from context is not of type ChatSessionOperator")
	}

	return "", fmt.Errorf("chat_session_operator not found from context")
}

func searchKnowledgeHandler(ctx context.Context, argumentsInJSON string) (string, error) {
	log.Info("Execute knowledge base search tool")

	var params SearchKnowledgeParams
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			response := NewErrorResponse("search_knowledge", "Parameter parsing failed", "PARSE_ERROR", "Please check query parameter format")
			return response.ToJSON()
		}
	}
	params.Query = strings.TrimSpace(params.Query)
	if params.Query == "" {
		response := NewErrorResponse("search_knowledge", "query cannot be empty", "INVALID_QUERY", "Please provide content to search")
		return response.ToJSON()
	}
	if params.TopK <= 0 {
		params.TopK = 5
	}

	chatSessionOperatorValue := ctx.Value("chat_session_operator")
	if chatSessionOperatorValue == nil {
		return "", fmt.Errorf("chat_session_operator not found from context")
	}
	chatSessionOperator, ok := chatSessionOperatorValue.(ChatSessionOperator)
	if !ok {
		return "", fmt.Errorf("The chat_session_operator obtained from context is not of type ChatSessionOperator")
	}

	hits, err := chatSessionOperator.LocalMcpSearchKnowledge(ctx, params.Query, params.TopK, params.KnowledgeBaseIDs)
	if err != nil {
		response := NewErrorResponse("search_knowledge", fmt.Sprintf("Knowledge search failed: %v", err), "SEARCH_FAILED", "Please try again later")
		return response.ToJSON()
	}

	data := map[string]interface{}{
		"query": params.Query,
		"hits":  hits,
		"count": len(hits),
	}
	if len(hits) == 0 {
		response := NewContentResponse("search_knowledge", data, "No sufficient relevant information found")
		return response.ToJSON()
	}

	var builder strings.Builder
	for i, hit := range hits {
		content := strings.TrimSpace(hit.Content)
		if content == "" {
			continue
		}
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, content))
	}
	msg := strings.TrimSpace(builder.String())
	if msg == "" {
		msg = "Retrieved relevant information"
	}
	response := NewContentResponse("search_knowledge", data, msg)
	return response.ToJSON()
}

// getWeekNumber gets the week number
func getWeekNumber(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

// formatDateTime formats date and time in English
func formatDateTime(t time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d %s %02d:%02d:%02d",
		t.Year(), int(t.Month()), t.Day(),
		t.Weekday().String(),
		t.Hour(), t.Minute(), t.Second(),
	)
}

// getWeekdayEnglish returns the English day of the week
func getWeekdayEnglish(weekday time.Weekday) string {
	weekdays := map[time.Weekday]string{
		time.Sunday:    "Sunday",
		time.Monday:    "Monday",
		time.Tuesday:   "Tuesday",
		time.Wednesday: "Wednesday",
		time.Thursday:  "Thursday",
		time.Friday:    "Friday",
		time.Saturday:  "Saturday",
	}
	return weekdays[weekday]
}

// RegisterChatMCPTools public function for external calls to register chat MCP tools
func RegisterChatMCPTools() {
	InitChatLocalMCPTools()
}

// play music
func GetMusicAudioData(ctx context.Context, musicParams *PlayMusicParams) ([]byte, string, error) {
	musicName := musicParams.Name
	//welcome := musicParams.Welcome
	welcome := ""
	log.Infof("Search music: %s, welcome: %s", musicName, welcome)
	//Here you can get the music URL based on the music name
	//The implementation is currently simplified, assuming that musicName is the URL or obtained from the configuration
	musicURL, realMusicName, ierr := getMusicURL(musicName)
	if ierr != nil {
		log.Errorf("Failed to get music URL: %v", ierr)
		return nil, "", fmt.Errorf("Failed to get music URL: %v", ierr)
	}

	log.Infof("Music search successful URL: %s, music name: %s", musicURL, realMusicName)

	client := getHTTPClient()
	req, err := http.NewRequest("GET", musicURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("Create request failed: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to read response: %v", err)
	}

	log.Infof("Acquisition of music %s data successful, audio data length: %d", realMusicName, len(audioData))

	return audioData, realMusicName, nil
}

/*
func GetMusicAudioData(ctx context.Context, musicParams *PlayMusicParams) ([]byte, string, error) {
	musicName := musicParams.Name
	//welcome := musicParams.Welcome
	welcome := ""
	log.Infof("Search music: %s, welcome: %s", musicName, welcome)
	// A music URL can be resolved from the music name here.
	// The current simplified implementation treats musicName as a URL or loads it from configuration.
	musicList := netease.Search(musicName)
	musicList = append(musicList, qq.Search(musicName)...)
	for id, music := range musicList {
		log.Infof("[%2d] %7s | %s %5sMB - %s - %s - %s\n", id, music.Source, music.Duration, music.Size, music.Title, music.Singer, music.Album)
	}

	if len(musicList) <= 0 {
		return nil, "", fmt.Errorf("no music found")
	}
	m := musicList[0]
	m.ParseMusic()
	rc, err := m.ReadCloser()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get music data: %v", err)
	}
	defer rc.Close()

	audioData, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %v", err)
	}

	log.Infof("Music data retrieved for %s, audio data length: %d", m.Name, len(audioData))

	return audioData, m.Name, nil

}
*/
