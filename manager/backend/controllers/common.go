package controllers

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// WebSocketControllerInterface defines the interface for WebSocket controllers
type WebSocketControllerInterface interface {
	RequestMcpToolDetailsFromClient(ctx context.Context, agentID string) ([]MCPTool, error)
}

// GetAgentMcpToolsCommon is a shared function for retrieving an agent's MCP tool list,
// usable by both admin and regular user controllers
func GetAgentMcpToolsCommon(
	c *gin.Context,
	agentID string,
	webSocketController WebSocketControllerInterface,
	agentValidator func(agentID string) error, // function to validate agent access
) {
	log.Printf("GetAgentMcpToolsCommon started, agentID: %s", agentID)

	if agentID == "" {
		log.Printf("error: agent_id parameter is empty")
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id parameter is required"})
		return
	}

	// Validate agent access (validation logic provided by caller)
	if err := agentValidator(agentID); err != nil {
		log.Printf("agent validation failed: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	log.Printf("agent validation succeeded, checking WebSocket controller")

	// Check if the WebSocket controller is available
	if webSocketController == nil {
		// Return empty list instead of error when WebSocket controller is unavailable
		log.Printf("WebSocket controller not initialized, returning empty tool list")
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"tools": []interface{}{}}})
		return
	}

	log.Printf("WebSocket controller available, requesting MCP tool list")

	// Create context
	ctx := context.Background()

	// Fetch tool details (including schema and examples)
	tools, err := webSocketController.RequestMcpToolDetailsFromClient(ctx, agentID)
	if err != nil {
		log.Printf("failed to get MCP tool list: %v", err)
		// Return empty list instead of error on failure
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"tools": []interface{}{}}})
		return
	}

	log.Printf("successfully retrieved MCP tool list: count=%d", len(tools))
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"tools": tools}})
}

func newMcpEndpointData(endpoint string) gin.H {
	return gin.H{
		"endpoint":     endpoint,
		"status":       "unknown",
		"connected":    false,
		"tools_count":  0,
		"client_count": 0,
	}
}

func applyMcpEndpointStatus(data gin.H, statusResult map[string]interface{}) {
	if data == nil || statusResult == nil {
		return
	}

	connected, _ := statusResult["connected"].(bool)
	status, _ := statusResult["status"].(string)
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		if connected {
			status = "online"
		} else {
			status = "offline"
		}
	}

	data["connected"] = connected
	data["status"] = status
	if clientCount, ok := statusResult["client_count"]; ok {
		data["client_count"] = clientCount
	}
	if toolsCount, ok := statusResult["tools_count"]; ok {
		data["tools_count"] = toolsCount
	}
	if statusMessage, ok := statusResult["status_message"].(string); ok && strings.TrimSpace(statusMessage) != "" {
		data["status_message"] = statusMessage
	}
}
