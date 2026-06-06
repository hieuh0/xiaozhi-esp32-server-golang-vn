package mcp

import (
	"fmt"
	"sync"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/getkin/kin-openapi/openapi3"

	mcp_protocol "github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// LocalMCPManager Local MCP Tool Manager
type LocalMCPManager struct {
	tools map[string]*McpTool //Tool name -> Tool definition
	mu    sync.RWMutex        //Read-write locks protect concurrent access
}

var (
	localManager *LocalMCPManager
	localOnce    sync.Once
)

// GetLocalMCPManager Gets the local MCP manager singleton
func GetLocalMCPManager() *LocalMCPManager {
	localOnce.Do(func() {
		localManager = &LocalMCPManager{
			tools: make(map[string]*McpTool),
		}
		//Initialize default local tools
		localManager.initDefaultTools()
	})
	return localManager
}

// initDefaultTools initializes default local tools
func (l *LocalMCPManager) initDefaultTools() {

	log.Info("The local MCP manager default tool initialization is completed.")
}

// RegisterTool Register local tool
func (l *LocalMCPManager) RegisterTool(tool *McpTool) error {
	if tool == nil {
		return fmt.Errorf("Tool cannot be empty")
	}

	if tool.info.Name == "" {
		return fmt.Errorf("Tool name cannot be empty")
	}

	if !tool.isLocal || tool.localHandler == nil {
		return fmt.Errorf("Tool processing function cannot be empty")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	//Check if the tool already exists
	if _, exists := l.tools[tool.info.Name]; exists {
		log.Warnf("Local tool %s already exists and will be overwritten", tool.info.Name)
	}

	l.tools[tool.info.Name] = tool
	log.Infof("Successfully registered local tools: %s - %s", tool.info.Name, tool.info.Desc)
	return nil
}

func (l *LocalMCPManager) convertStructToOpenaipi3Schema(inputParams any) (*openapi3.Schema, error) {
	//Use github.com/ThinkInAIXYZ/go-mcp to generate tool through struct, and then convert it to openapi3.Schema
	toolInstance, err := mcp_protocol.NewTool("get_system_info", "Get basic system information", inputParams)
	if err != nil {
		return nil, err
	}

	marshaledInputSchema, err := sonic.Marshal(toolInstance.InputSchema)
	if err != nil {
		return nil, err
	}

	inputSchema := &openapi3.Schema{}
	err = sonic.Unmarshal(marshaledInputSchema, inputSchema)
	if err != nil {
		return nil, err
	}
	return inputSchema, nil
}

// RegisterToolFunc register tool function (simplified version)
func (l *LocalMCPManager) RegisterToolFunc(name, description string, inputParams any, handler LocalToolHandler) error {
	inputSchema, err := l.convertStructToOpenaipi3Schema(inputParams)
	if err != nil {
		log.Errorf("Failed to convert struct to openapi3 schema: %v", err)
		return err
	}
	tool := &McpTool{
		info: &schema.ToolInfo{
			Name:        name,
			Desc:        description,
			ParamsOneOf: schema.NewParamsOneOfByOpenAPIV3(inputSchema),
		},
		isLocal:      true,
		localHandler: handler,
	}
	return l.RegisterTool(tool)
}

// UnregisterTool Unregister tool
func (l *LocalMCPManager) UnregisterTool(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.tools[name]; !exists {
		return fmt.Errorf("Tool %s does not exist", name)
	}

	delete(l.tools, name)
	log.Infof("Successfully logged out of local tool: %s", name)
	return nil
}

// GetAllTools gets all local tools and returns the Eino tool interface format
func (l *LocalMCPManager) GetAllTools() map[string]tool.InvokableTool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make(map[string]tool.InvokableTool)
	for name, mcpTool := range l.tools {
		result[name] = mcpTool
	}
	return result
}

// GetToolByName Gets a tool by name
func (l *LocalMCPManager) GetToolByName(name string) (tool.InvokableTool, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	mcpTool, exists := l.tools[name]
	if !exists {
		return nil, false
	}

	return mcpTool, true
}

// GetToolNames Gets a list of all tool names
func (l *LocalMCPManager) GetToolNames() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.tools))
	for name := range l.tools {
		names = append(names, name)
	}
	return names
}

// GetToolCount Gets the number of tools
func (l *LocalMCPManager) GetToolCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.tools)
}

// Start starts the local manager (reserved interface)
func (l *LocalMCPManager) Start() error {
	log.Info("Local MCP manager started")
	return nil
}

// Stop Stop the local manager (reserved interface)
func (l *LocalMCPManager) Stop() error {
	//Note: We do not clear the tools because the local manager's tools should remain available throughout the application lifecycle
	//If you need to clear the tool, you should explicitly call the UnregisterTool method
	log.Info("The local MCP manager has stopped")
	return nil
}
