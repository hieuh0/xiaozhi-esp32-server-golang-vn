package mcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/viper"

	log "xiaozhi-esp32-server-golang/logger"
)

const (
	globalMCPPingInterval                 = 60 * time.Second
	globalMCPPingTimeout                  = 5 * time.Second
	globalMCPPeriodicToolsRefreshInterval = 2 * time.Minute
)

// MCPServerConfig MCP server configuration
type MCPServerConfig struct {
	Name         string            `json:"name" mapstructure:"name"`
	Type         string            `json:"type" mapstructure:"type"`
	Url          string            `json:"url" mapstructure:"url"`
	SSEUrl       string            `json:"sse_url" mapstructure:"sse_url"` //Backwards compatible with sse_url field
	Enabled      bool              `json:"enabled" mapstructure:"enabled"`
	Provider     string            `json:"provider,omitempty" mapstructure:"provider"`
	ServiceID    string            `json:"service_id,omitempty" mapstructure:"service_id"`
	AuthRef      string            `json:"auth_ref,omitempty" mapstructure:"auth_ref"`
	Headers      map[string]string `json:"headers,omitempty" mapstructure:"headers"`
	AllowedTools []string          `json:"allowed_tools,omitempty" mapstructure:"allowed_tools"`
}

// GlobalMCPManager GlobalMCPManager
type GlobalMCPManager struct {
	servers       map[string]*MCPServerConnection
	tools         map[string]tool.InvokableTool
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	reconnectConf ReconnectConfig
	httpClient    *http.Client
}

// ReconnectConfig reconnection configuration
type ReconnectConfig struct {
	Interval    time.Duration
	MaxAttempts int
}

// MCPServerConnection MCP server connection
type MCPServerConnection struct {
	config        MCPServerConfig
	client        *client.Client
	tools         map[string]tool.InvokableTool
	connected     bool
	refreshing    bool
	refreshQueued bool
	mu            sync.RWMutex
	lastError     error
	retryCount    int
	lastPing      time.Time
	reconnecting  bool
	reconnectWait chan struct{}
}

var (
	globalManager *GlobalMCPManager
	once          sync.Once
)

var buildGlobalMCPTransport = buildMCPTransport

// GetGlobalMCPManager Gets the global MCP manager singleton
func GetGlobalMCPManager() *GlobalMCPManager {
	once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		globalManager = &GlobalMCPManager{
			servers: make(map[string]*MCPServerConnection),
			tools:   make(map[string]tool.InvokableTool),
			ctx:     ctx,
			cancel:  cancel,
			reconnectConf: ReconnectConfig{
				Interval:    time.Duration(viper.GetInt("mcp.global.reconnect_interval")) * time.Second,
				MaxAttempts: viper.GetInt("mcp.global.max_reconnect_attempts"),
			},
			httpClient: &http.Client{
				Timeout: 600 * time.Second,
			},
		}
	})
	return globalManager
}

// Start starts the global MCP manager
func (g *GlobalMCPManager) Start() error {
	//Hot update scenario: ctx has been canceled after Stop and needs to be rebuilt so that monitoring and reconnection can be normal after restarting.
	if g.ctx != nil && g.ctx.Err() != nil {
		g.ctx, g.cancel = context.WithCancel(context.Background())
		g.reconnectConf = ReconnectConfig{
			Interval:    time.Duration(viper.GetInt("mcp.global.reconnect_interval")) * time.Second,
			MaxAttempts: viper.GetInt("mcp.global.max_reconnect_attempts"),
		}
	}

	//First check the configuration
	CheckMCPConfig()

	if !viper.GetBool("mcp.global.enabled") {
		log.Info("Global MCP Manager is disabled")
		return nil
	}

	var serverConfigs []MCPServerConfig
	if err := viper.UnmarshalKey("mcp.global.servers", &serverConfigs); err != nil {
		log.Errorf("Failed to parse MCP server configuration: %v", err)
		return fmt.Errorf("Failed to parse MCP server configuration: %v", err)
	}

	log.Infof("Read %d MCP server configurations from the configuration", len(serverConfigs))

	//Detailed documentation of each server configuration
	for i, config := range serverConfigs {
		log.Infof("MCP server [%d]: Type=%s, Name=%s, Url=%s, SSEUrl=%s, Enabled=%v",
			i+1, config.Type, config.Name, config.Url, config.SSEUrl, config.Enabled)
	}

	//Connect to enabled server
	connectedCount := 0
	for _, config := range serverConfigs {
		if config.Enabled {
			if err := g.connectToServer(config); err != nil {
				log.Errorf("Failed to connect to MCP server %s: %v", config.Name, err)
			} else {
				connectedCount++
			}
		} else {
			log.Infof("MCP server %s disabled, skipping connection", config.Name)
		}
	}

	log.Infof("Successfully connected %d MCP servers", connectedCount)

	//Start monitoring goroutine
	go g.monitorConnections()

	log.Info("Global MCP Manager started")
	return nil
}

// Stop Stop the global MCP manager
func (g *GlobalMCPManager) Stop() error {
	g.cancel()

	g.mu.Lock()
	type serverEntry struct {
		name string
		conn *MCPServerConnection
	}
	servers := make([]serverEntry, 0, len(g.servers))
	for name, conn := range g.servers {
		if conn != nil {
			servers = append(servers, serverEntry{name: name, conn: conn})
		}
	}
	g.servers = make(map[string]*MCPServerConnection)
	g.tools = make(map[string]tool.InvokableTool)
	g.mu.Unlock()

	for _, server := range servers {
		if err := server.conn.disconnect(); err != nil {
			log.Errorf("Disconnect MCP server %s connection failed: %v", server.name, err)
		}
	}

	log.Info("Global MCP Manager has stopped")
	return nil
}

// createFailedConnection creates a failed connection object for subsequent reconnection
func (g *GlobalMCPManager) createFailedConnection(config MCPServerConfig) {
	conn := &MCPServerConnection{
		config:     config,
		tools:      make(map[string]tool.InvokableTool),
		connected:  false,
		lastError:  fmt.Errorf("Failed to initialize connection"),
		retryCount: 0,
	}

	g.mu.Lock()
	g.servers[config.Name] = conn
	g.mu.Unlock()

	log.Infof("Connection object created for failed MCP server: %s", config.Name)
}

// connectToServer connects to the MCP server
func (g *GlobalMCPManager) connectToServer(config MCPServerConfig) error {
	//Verify configuration
	if config.Name == "" {
		return fmt.Errorf("MCP server name cannot be empty")
	}

	if !config.Enabled {
		log.Infof("MCP server %s disabled, skipping connection", config.Name)
		return nil
	}

	_, endpoint, endpointErr := endpointForConfig(config)
	if endpointErr != nil {
		return endpointErr
	}
	log.Infof("Connecting to MCP server: %s (URL: %s)", config.Name, endpoint)

	conn := &MCPServerConnection{
		config: config,
		tools:  make(map[string]tool.InvokableTool),
	}

	g.mu.Lock()
	g.servers[config.Name] = conn
	g.mu.Unlock()

	//Connect to server
	if err := conn.connect(); err != nil {
		return fmt.Errorf("Failed to connect to MCP server: %v", err)
	}

	log.Infof("Connected to MCP server: %s", config.Name)
	return nil
}

// connect connect to MCP server
func (conn *MCPServerConnection) connect() (retErr error) {
	//Use background context and do not set a timeout to keep the SSE connection alive for a long time
	ctx := context.Background()

	transportInstance, endpoint, err := buildGlobalMCPTransport(conn.config)
	if err != nil {
		return err
	}

	//Create an MCP client using client.NewClient
	mcpClient := client.NewClient(transportInstance)
	serverName := conn.config.Name
	defer func() {
		if retErr == nil {
			return
		}

		conn.mu.Lock()
		conn.client = nil
		conn.connected = false
		conn.refreshing = false
		conn.refreshQueued = false
		conn.tools = make(map[string]tool.InvokableTool)
		conn.lastError = retErr
		conn.mu.Unlock()

		if globalManager != nil {
			globalManager.removeGlobalTools(serverName)
		}

		if closeErr := mcpClient.Close(); closeErr != nil {
			log.Errorf("Failed to close MCP client: %v", closeErr)
		}
	}()

	mcpClient.OnNotification(conn.handleJSONRPCNotification)
	conn.mu.Lock()
	conn.client = mcpClient
	conn.mu.Unlock()

	log.Infof("Start connecting to MCP server: %s, %s URL: %s", conn.config.Name, conn.config.Type, endpoint)

	//Start client
	if err := mcpClient.Start(ctx); err != nil {
		log.Errorf("Failed to start MCP client, server: %s, error: %v", conn.config.Name, err)
		retErr = fmt.Errorf("Failed to start client: %v", err)
		return retErr
	}

	log.Infof("MCP client started successfully: %s", conn.config.Name)

	//Initialize client
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "xiaozhi-esp32-server",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{
				Experimental: make(map[string]any),
			},
		},
	}

	log.Infof("Initializing MCP server: %s", conn.config.Name)
	initResult, err := mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		log.Errorf("Failed to initialize MCP server, server: %s, error: %v", conn.config.Name, err)
		retErr = fmt.Errorf("Initialization failed: %v", err)
		return retErr
	}

	log.Infof("MCP server initialization successful: %s, result: %+v", conn.config.Name, initResult)

	//Get a list of tools
	if refreshErr := conn.refreshTools(ctx); refreshErr != nil {
		log.Errorf("Failed to obtain tool list: %v", refreshErr)
		retErr = fmt.Errorf("Failed to obtain tool list: %v", refreshErr)
		return retErr
	}

	conn.mu.Lock()
	conn.connected = true
	conn.lastError = nil
	conn.retryCount = 0
	conn.mu.Unlock()

	log.Infof("MCP server connection establishment completed: %s", conn.config.Name)
	return nil
}

func (conn *MCPServerConnection) handleJSONRPCNotification(notification mcp.JSONRPCNotification) {
	switch notification.Method {
	case mcp.MethodNotificationToolsListChanged, "notifications/tools/updated":
		log.Infof("MCP server %s received the tool list update notification and prepared to refresh the tool list.", conn.config.Name)
		conn.scheduleToolsRefresh()
	}
}

func (conn *MCPServerConnection) scheduleToolsRefresh() {
	conn.scheduleToolsRefreshWithReason("notification")
}

func (conn *MCPServerConnection) schedulePeriodicToolsRefresh() {
	conn.scheduleToolsRefreshWithReason("periodic refresh")
}

func (conn *MCPServerConnection) scheduleToolsRefreshWithReason(reason string) {
	conn.mu.Lock()
	if conn.refreshing {
		conn.refreshQueued = true
		conn.mu.Unlock()
		return
	}
	conn.refreshing = true
	conn.mu.Unlock()

	go conn.runScheduledToolsRefresh(reason)
}

func (conn *MCPServerConnection) runScheduledToolsRefresh(reason string) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := conn.refreshTools(ctx)
		cancel()
		if err != nil {
			log.Warnf("MCP server %s %s Failed to refresh tool list: %v", conn.config.Name, reason, err)
		}

		conn.mu.Lock()
		if err != nil {
			conn.lastError = err
		} else {
			conn.lastError = nil
		}

		if conn.refreshQueued {
			conn.refreshQueued = false
			conn.mu.Unlock()
			continue
		}

		conn.refreshing = false
		conn.mu.Unlock()
		return
	}
}

func normalizeMCPTransportType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "sse":
		return "sse"
	case "streamable_http", "streamable-http", "http":
		return "streamablehttp"
	default:
		return strings.ToLower(strings.TrimSpace(t))
	}
}

func endpointForConfig(config MCPServerConfig) (string, string, error) {
	transportType := normalizeMCPTransportType(config.Type)
	if transportType == "" {
		if strings.TrimSpace(config.SSEUrl) != "" {
			transportType = "sse"
		} else if strings.TrimSpace(config.Url) != "" {
			transportType = "streamablehttp"
		}
	}

	switch transportType {
	case "sse":
		if strings.TrimSpace(config.SSEUrl) != "" {
			return transportType, strings.TrimSpace(config.SSEUrl), nil
		}
		if strings.TrimSpace(config.Url) != "" {
			return transportType, strings.TrimSpace(config.Url), nil
		}
		return "", "", fmt.Errorf("MCP server %s missing SSE URL", config.Name)
	case "streamablehttp":
		if strings.TrimSpace(config.Url) != "" {
			return transportType, strings.TrimSpace(config.Url), nil
		}
		if strings.TrimSpace(config.SSEUrl) != "" {
			return transportType, strings.TrimSpace(config.SSEUrl), nil
		}
		return "", "", fmt.Errorf("MCP server %s missing StreamableHTTP URL", config.Name)
	default:
		return "", "", fmt.Errorf("MCP server %s type is not supported: %s", config.Name, config.Type)
	}
}

func buildMCPTransport(config MCPServerConfig) (transport.Interface, string, error) {
	transportType, endpoint, err := endpointForConfig(config)
	if err != nil {
		return nil, "", err
	}

	headers := make(map[string]string)
	for k, v := range config.Headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		headers[strings.TrimSpace(k)] = v
	}

	switch transportType {
	case "sse":
		opts := make([]transport.ClientOption, 0)
		if len(headers) > 0 {
			opts = append(opts, transport.WithHeaders(headers))
		}
		sseTransport, err := transport.NewSSE(endpoint, opts...)
		if err != nil {
			return nil, "", fmt.Errorf("Failed to create SSE transport layer: %v", err)
		}
		return sseTransport, endpoint, nil
	case "streamablehttp":
		opts := make([]transport.StreamableHTTPCOption, 0)
		if len(headers) > 0 {
			opts = append(opts, transport.WithHTTPHeaders(headers))
		}
		httpTransport, err := transport.NewStreamableHTTP(endpoint, opts...)
		if err != nil {
			return nil, "", fmt.Errorf("Failed to create StreamableHTTP transport layer: %v", err)
		}
		return httpTransport, endpoint, nil
	default:
		return nil, "", fmt.Errorf("Unsupported MCP transfer type: %s", transportType)
	}
}

func buildAllowedToolSet(allowedTools []string) map[string]struct{} {
	if len(allowedTools) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(allowedTools))
	for _, toolName := range allowedTools {
		toolName = strings.TrimSpace(toolName)
		if toolName == "" {
			continue
		}
		set[toolName] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

func filterMCPToolsByAllowList(tools []mcp.Tool, allowedTools []string) []mcp.Tool {
	allowedSet := buildAllowedToolSet(allowedTools)
	if len(allowedSet) == 0 {
		return tools
	}

	filtered := make([]mcp.Tool, 0, len(tools))
	for _, item := range tools {
		if _, ok := allowedSet[strings.TrimSpace(item.Name)]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// refreshTools refresh tool list
func (conn *MCPServerConnection) refreshTools(ctx context.Context) error {
	conn.mu.RLock()
	serverName := conn.config.Name
	allowedTools := append([]string(nil), conn.config.AllowedTools...)
	mcpClient := conn.client
	conn.mu.RUnlock()
	if mcpClient == nil {
		return fmt.Errorf("MCP client not initialized")
	}

	//Get a list of tools
	listRequest := mcp.ListToolsRequest{}
	toolsResult, err := mcpClient.ListTools(ctx, listRequest)
	if err != nil {
		return fmt.Errorf("Failed to obtain tool list: %v", err)
	}

	tools := filterMCPToolsByAllowList(toolsResult.Tools, allowedTools)
	convertedTools := ConvertMcpToolListToInvokableToolList(tools, serverName, mcpClient)

	conn.mu.Lock()
	conn.tools = convertedTools
	conn.mu.Unlock()

	//Updates to the global tool table are placed outside conn.mu to avoid lock order reversal with g.mu.
	globalManager.updateGlobalTools(serverName, convertedTools)

	log.Infof("MCP server %s tool list has been updated, with a total of %d tools", serverName, len(convertedTools))
	return nil
}

func ConvertMcpToolListToInvokableToolList(tools []mcp.Tool, serverName string, client *client.Client) map[string]tool.InvokableTool {
	invokeTools := make(map[string]tool.InvokableTool)
	usedNames := make(map[string]string, len(tools))
	for _, tool := range tools {
		originName := tool.Name
		if strings.TrimSpace(originName) == "" {
			log.Warnf("Skip empty name MCP tool, server=%s", serverName)
			continue
		}
		llmName := uniqueLLMToolName(sanitizeLLMToolName(originName), originName, usedNames)
		if llmName != originName {
			log.Debugf("The MCP tool name %q does not comply with the OpenAI tool name specification and has been converted to %q, server=%s", originName, llmName, serverName)
		}

		marshaledInputSchema, err := sonic.Marshal(tool.InputSchema)
		if err != nil {
			log.Errorf("convert mcp tool to invokeable tool err: %+v", err)
			continue
		}
		inputSchema := &openapi3.Schema{}
		err = sonic.Unmarshal(marshaledInputSchema, inputSchema)
		if err != nil {
			log.Errorf("convert mcp tool to invokeable tool err: %+v", err)
			continue
		}

		mcpToolInstance := &McpTool{
			info: &schema.ToolInfo{
				Name:        llmName,
				Desc:        tool.Description,
				ParamsOneOf: schema.NewParamsOneOfByOpenAPIV3(inputSchema),
			},
			originName: originName,
			serverName: serverName,
			client:     client,
		}
		invokeTools[llmName] = mcpToolInstance
	}
	return invokeTools
}

// disconnect disconnect
func (conn *MCPServerConnection) disconnect() error {
	conn.mu.Lock()
	serverName := conn.config.Name
	mcpClient := conn.client
	conn.client = nil
	conn.connected = false
	conn.tools = make(map[string]tool.InvokableTool)
	conn.mu.Unlock()

	if globalManager != nil {
		globalManager.removeGlobalTools(serverName)
	}

	if mcpClient != nil {
		//Close the client and place it outside the lock to avoid locking the fast path.
		if err := mcpClient.Close(); err != nil {
			log.Errorf("Failed to close MCP client: %v", err)
		}
	}

	return nil
}

func (g *GlobalMCPManager) removeGlobalTools(serverName string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for name, mcpToolInterface := range g.tools {
		if mt, ok := mcpToolInterface.(*McpTool); ok && mt.serverName == serverName {
			delete(g.tools, name)
		}
	}
}

// updateGlobalTools updates the global tool list
func (g *GlobalMCPManager) updateGlobalTools(serverName string, tools map[string]tool.InvokableTool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	//Remove old tools from this server
	for name, mcpToolInterface := range g.tools {
		if mt, ok := mcpToolInterface.(*McpTool); ok && mt.serverName == serverName {
			delete(g.tools, name)
		}
	}

	//Add new tool
	for name, mcpToolInterface := range tools {
		g.tools[fmt.Sprintf("%s_%s", serverName, name)] = mcpToolInterface
	}
}

// GetAllTools Get all available tools
func (g *GlobalMCPManager) GetAllTools() map[string]tool.InvokableTool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[string]tool.InvokableTool)
	for name, mcpToolInterface := range g.tools {
		result[name] = mcpToolInterface
	}
	return result
}

// GetToolByName Gets a tool by name
func (g *GlobalMCPManager) GetToolByName(name string) (tool.InvokableTool, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if invokable, exists := g.tools[name]; exists {
		return invokable, true
	}

	var matched tool.InvokableTool
	matchCount := 0
	for _, invokable := range g.tools {
		if !mcpToolMatchesName(invokable, name) {
			continue
		}
		matchCount++
		if matchCount == 1 {
			matched = invokable
			continue
		}

		log.Warnf("Global MCP tool name %s There are multiple providers with the same name. Please specify the server name explicitly.", name)
		return nil, false
	}
	return matched, matchCount == 1
}

func GetServerClientByName(serverName string) *client.Client {
	return GetGlobalMCPManager().GetServerClientByName(serverName)
}

func (g *GlobalMCPManager) GetServerClientByName(serverName string) *client.Client {
	g.mu.RLock()
	conn, ok := g.servers[serverName]
	g.mu.RUnlock()
	if !ok || conn == nil {
		return nil
	}

	conn.mu.RLock()
	defer conn.mu.RUnlock()
	return conn.client
}

func GetServerEndpointSnapshotByName(serverName string) string {
	return GetGlobalMCPManager().GetServerEndpointSnapshotByName(serverName)
}

func (g *GlobalMCPManager) GetServerEndpointSnapshotByName(serverName string) string {
	g.mu.RLock()
	conn, ok := g.servers[serverName]
	g.mu.RUnlock()
	if !ok || conn == nil {
		return ""
	}

	conn.mu.RLock()
	config := conn.config
	conn.mu.RUnlock()

	_, endpoint, err := endpointForConfig(config)
	if err != nil {
		if strings.TrimSpace(config.Url) != "" {
			return strings.TrimSpace(config.Url)
		}
		return strings.TrimSpace(config.SSEUrl)
	}
	return endpoint
}

func ReconnectServerByName(serverName string) (*client.Client, error) {
	return GetGlobalMCPManager().reconnectServer(serverName)
}

// isSessionClosedError determines whether it is a session closed error
func isSessionClosedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "session closed")
}

func isRetryableRemoteCallError(err error) bool {
	if err == nil {
		return false
	}
	if isSessionClosedError(err) {
		return true
	}

	message := strings.ToLower(err.Error())
	retryableIndicators := []string{
		"unexpected end of json input",
		"invalid character",
		"eof",
		"broken pipe",
		"connection reset",
		"connection refused",
		"connection aborted",
		"timeout",
		"bad gateway",
		"502",
		"temporarily unavailable",
	}
	for _, indicator := range retryableIndicators {
		if strings.Contains(message, indicator) {
			return true
		}
	}
	return false
}

func (g *GlobalMCPManager) schedulePeriodicToolsRefresh() {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, conn := range g.servers {
		if conn == nil {
			continue
		}

		conn.mu.RLock()
		connected := conn.connected
		hasClient := conn.client != nil
		conn.mu.RUnlock()
		if !connected || !hasClient {
			continue
		}

		conn.schedulePeriodicToolsRefresh()
	}
}

// monitorConnections monitors connection status
func (g *GlobalMCPManager) monitorConnections() {
	pingTicker := time.NewTicker(globalMCPPingInterval) //ping every 60 seconds
	defer pingTicker.Stop()
	toolsRefreshTicker := time.NewTicker(globalMCPPeriodicToolsRefreshInterval)
	defer toolsRefreshTicker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-pingTicker.C:
			//Perform ping detection
			g.mu.RLock()
			for name, conn := range g.servers {
				go func(name string, conn *MCPServerConnection) {
					ctx, cancel := context.WithTimeout(context.Background(), globalMCPPingTimeout)
					defer cancel()

					if err := conn.ping(ctx); err != nil {
						log.Warnf("MCP server %s ping failed and started to reconnect: %v", name, err)
						//When ping fails, it is directly marked as disconnected and triggers reconnection.
						conn.mu.Lock()
						conn.connected = false
						conn.lastError = err
						conn.mu.Unlock()

						//Directly trigger reconnection
						go g.reconnectServer(name)
					} else {
						//log.Debugf("MCMCP server %s ping successful successful", name)
					}
				}(name, conn)
			}
			g.mu.RUnlock()
		case <-toolsRefreshTicker.C:
			g.schedulePeriodicToolsRefresh()
		}
	}
}

// reconnectServer reconnects to the server and returns a new client
func (g *GlobalMCPManager) reconnectServer(serverName string) (*client.Client, error) {
	g.mu.RLock()
	var conn *MCPServerConnection
	for _, c := range g.servers {
		if c.config.Name == serverName {
			conn = c
			break
		}
	}
	g.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("Server connection not found: %s", serverName)
	}

	conn.mu.Lock()
	if conn.reconnecting {
		wait := conn.reconnectWait
		conn.mu.Unlock()
		if wait != nil {
			<-wait
		}

		conn.mu.RLock()
		mcpClient := conn.client
		connected := conn.connected
		lastErr := conn.lastError
		conn.mu.RUnlock()
		if mcpClient != nil && connected {
			return mcpClient, nil
		}
		if lastErr != nil {
			return nil, fmt.Errorf("Reconnection failed: %v", lastErr)
		}
		return nil, fmt.Errorf("Reconnect failed: client is not ready")
	}
	wait := make(chan struct{})
	conn.reconnecting = true
	conn.reconnectWait = wait
	conn.mu.Unlock()

	defer func() {
		conn.mu.Lock()
		conn.reconnecting = false
		if conn.reconnectWait == wait {
			close(wait)
			conn.reconnectWait = nil
		}
		conn.mu.Unlock()
	}()

	//Disconnect
	if err := conn.disconnect(); err != nil {
		log.Errorf("Failed to disconnect: %v", err)
	}

	//Wait a short period of time to ensure resources are released
	time.Sleep(time.Second)

	//Reconnect
	if err := conn.connect(); err != nil {
		conn.mu.Lock()
		conn.lastError = err
		conn.mu.Unlock()
		return nil, fmt.Errorf("Reconnection failed: %v", err)
	}

	conn.mu.RLock()
	mcpClient := conn.client
	conn.mu.RUnlock()
	return mcpClient, nil
}

// ping sends a ping request to check the connection status
func (conn *MCPServerConnection) ping(ctx context.Context) error {
	conn.mu.RLock()
	mcpClient := conn.client
	conn.mu.RUnlock()
	if mcpClient == nil {
		return fmt.Errorf("client not initialized")
	}

	//Use empty Ping request as ping
	err := mcpClient.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Ping failed: %v", err)
	}

	conn.mu.Lock()
	conn.lastPing = time.Now()
	conn.mu.Unlock()

	return nil
}
