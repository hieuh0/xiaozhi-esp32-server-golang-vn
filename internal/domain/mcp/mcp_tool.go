package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var callRemoteMCPTool = func(ctx context.Context, cli *client.Client, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return cli.CallTool(ctx, request)
}

var reconnectGlobalMCPServer = func(serverName string) (*client.Client, error) {
	return GetGlobalMCPManager().reconnectServer(serverName)
}

// LocalToolHandler local tool processing function type
type LocalToolHandler func(ctx context.Context, argumentsInJSON string) (string, error)

// mcpTool MCP tool implementation, supporting remote and local tools
type McpTool struct {
	info       *schema.ToolInfo
	originName string
	serverName string
	client     *client.Client

	//Native tool support
	isLocal      bool
	localHandler LocalToolHandler
}

// Info obtains tool information and implements the BaseTool interface
func (t *McpTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *McpTool) callName() string {
	if t.originName != "" {
		return t.originName
	}
	if t.info != nil {
		return t.info.Name
	}
	return ""
}

func mcpToolMatchesName(invokable tool.InvokableTool, name string) bool {
	mcpTool, ok := invokable.(*McpTool)
	if !ok || mcpTool == nil {
		return false
	}
	if mcpTool.info != nil && mcpTool.info.Name == name {
		return true
	}
	return mcpTool.originName != "" && mcpTool.originName == name
}

func findInvokableToolByName(tools map[string]tool.InvokableTool, name string) (tool.InvokableTool, bool) {
	if invokable, ok := tools[name]; ok {
		return invokable, true
	}
	for _, invokable := range tools {
		if mcpToolMatchesName(invokable, name) {
			return invokable, true
		}
	}
	return nil, false
}

func remoteCallNameForTool(invokable tool.InvokableTool, fallback string) string {
	if mcpTool, ok := invokable.(*McpTool); ok && mcpTool != nil {
		if name := mcpTool.callName(); name != "" {
			return name
		}
	}
	return fallback
}

func (t *McpTool) InvokeableLocalRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	toolInfo := t.info
	if t.localHandler == nil {
		return "", fmt.Errorf("The handler function for local tool %s is undefined", toolInfo.Name)
	}

	log.Infof("Execute local tool: %s, parameters: %s", toolInfo.Name, argumentsInJSON)

	resultStr, err := t.localHandler(ctx, argumentsInJSON)
	if err != nil {
		log.Errorf("Local tool %s failed to execute: %v", toolInfo.Name, err)
		return "", fmt.Errorf("Local tool execution failed: %v", err)
	}
	if len(resultStr) > 2048 {
		log.Infof("Local tool %s executed successfully, result length: %d", toolInfo.Name, len(resultStr))
	} else {
		log.Infof("Local tool %s was executed successfully, result: %+s", toolInfo.Name, resultStr)
	}

	return resultStr, nil
}

// InvokableRun calls the tool and implements the InvokableTool interface
func (t *McpTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	//If it is a local tool, call the local processing function directly
	if t.isLocal {
		return t.InvokeableLocalRun(ctx, argumentsInJSON, opts...)
	}

	retContent := ""

	//Remote MCP tool calling logic
	//Check if the client is available
	if t.client == nil {
		return retContent, fmt.Errorf("Failed to call MCP tool: MCP client not initialized")
	}

	//Parse parameters
	var arguments map[string]interface{}
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &arguments); err != nil {
			return retContent, fmt.Errorf("Failed to parse tool parameters: %v", err)
		}
	}

	//Prepare to call request
	toolName := t.callName()
	callRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	}

	result, err := callRemoteMCPTool(ctx, t.client, callRequest)
	if err != nil {
		if !isRetryableRemoteCallError(err) {
			return retContent, fmt.Errorf("Failed to call tool: %v", err)
		}

		log.Warnf("The tool %s failed to call, prepare to reconnect to the server %s and try again: %v", t.info.Name, t.serverName, err)

		newClient, reconnectErr := reconnectGlobalMCPServer(t.serverName)
		if reconnectErr != nil {
			return retContent, fmt.Errorf("Failed to call the tool: %v, and failed to reconnect to the server: %v", err, reconnectErr)
		}

		t.client = newClient
		result, err = callRemoteMCPTool(ctx, t.client, callRequest)
		if err != nil {
			return retContent, fmt.Errorf("The call still fails after reconnecting: %v", err)
		}
	}

	if err != nil {
		return retContent, fmt.Errorf("Failed to call tool: %v", err)
	}

	resultStr, err := result.MarshalJSON()
	if err != nil {
		return retContent, fmt.Errorf("Tool call returns content conversion failure: %v", err)
	}

	return string(resultStr), nil
}

func (t *McpTool) GetClient() *client.Client {
	return t.client
}

func (t *McpTool) GetServerName() string {
	return t.serverName
}
