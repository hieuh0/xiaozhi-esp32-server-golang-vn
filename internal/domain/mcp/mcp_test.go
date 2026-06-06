package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func markTestClientConnected(client *McpClientInstance, lastPing time.Time) *McpClientInstance {
	client.storeToolsSnapshot(client.tools)
	client.setConnected(true)
	client.setLastPing(lastPing)
	return client
}

func newTestMcpClientPool() *McpClientPool {
	return &McpClientPool{
		device2McpClient: cmap.New[*DeviceMcpSession](),
	}
}

func TestGlobalMCPManager_Singleton(t *testing.T) {
	// 测试单例模式
	manager1 := GetGlobalMCPManager()
	manager2 := GetGlobalMCPManager()

	assert.Equal(t, manager1, manager2, "应该返回同一个实例")
}

func TestDeviceMCPManager_Singleton(t *testing.T) {
	t.Skip("GetDeviceMCPManager function not implemented yet")
	// // 测试单例模式
	// manager1 := GetDeviceMCPManager()
	// manager2 := GetDeviceMCPManager()
	//
	// assert.Equal(t, manager1, manager2, "应该返回同一个实例")
}

func TestGlobalMCPManager_StartStop(t *testing.T) {
	// 设置测试配置
	viper.Set("mcp.global.enabled", false)

	manager := GetGlobalMCPManager()

	// 测试启动（禁用状态）
	err := manager.Start()
	assert.NoError(t, err)

	// 测试停止
	err = manager.Stop()
	assert.NoError(t, err)
}

func TestMCPTool_Info(t *testing.T) {
	tool := &McpTool{
		info: &schema.ToolInfo{
			Name: "test_tool",
			Desc: "测试工具",
		},
		serverName: "test_server",
		client:     nil, // 测试中不需要真实客户端
	}

	info, err := tool.Info(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "test_tool", info.Name)
	assert.Equal(t, "测试工具", info.Desc)
}

func TestMCPTool_InvokableRun(t *testing.T) {
	tool := &McpTool{
		info: &schema.ToolInfo{
			Name: "test_tool",
			Desc: "测试工具",
		},
		serverName: "test_server",
		client:     nil, // 测试中不需要真实客户端
	}

	// 这个测试会失败，因为客户端为nil
	// 但可以验证方法签名和基本逻辑
	_, err := tool.InvokableRun(context.Background(), `{"query": "test"}`)
	assert.Error(t, err)                                       // 预期会有错误，因为客户端为nil
	assert.Contains(t, err.Error(), "Failed to call MCP tool") // Verify the expected error message.
}

func TestDeviceMCPManager_GetDeviceTools(t *testing.T) {
	t.Skip("GetDeviceMCPManager function not implemented yet")
	// manager := GetDeviceMCPManager()
	//
	// // 测试获取不存在设备的工具
	// tools := manager.GetDeviceTools("non_existent_device")
	// assert.Empty(t, tools)
}

func TestGlobalMCPManager_GetAllTools(t *testing.T) {
	manager := GetGlobalMCPManager()

	// 测试获取所有工具（初始状态应该为空）
	tools := manager.GetAllTools()
	assert.NotNil(t, tools)
}

func TestGlobalMCPManager_GetToolByName(t *testing.T) {
	manager := GetGlobalMCPManager()

	// 测试获取不存在的工具
	tool, exists := manager.GetToolByName("non_existent_tool")
	assert.False(t, exists)
	assert.Nil(t, tool)
}

func TestGlobalMCPManager_GetToolByNameRejectsAmbiguousMatches(t *testing.T) {
	manager := GetGlobalMCPManager()

	manager.mu.Lock()
	originalTools := manager.tools
	manager.tools = map[string]einotool.InvokableTool{
		"server_a_shared_tool": &McpTool{
			info:       &schema.ToolInfo{Name: "shared_tool"},
			serverName: "server_a",
		},
		"server_b_shared_tool": &McpTool{
			info:       &schema.ToolInfo{Name: "shared_tool"},
			serverName: "server_b",
		},
	}
	manager.mu.Unlock()

	t.Cleanup(func() {
		manager.mu.Lock()
		manager.tools = originalTools
		manager.mu.Unlock()
	})

	tool, exists := manager.GetToolByName("shared_tool")
	assert.False(t, exists)
	assert.Nil(t, tool)

	tool, exists = manager.GetToolByName("server_a_shared_tool")
	assert.True(t, exists)
	assert.NotNil(t, tool)
}

func TestMCPServerConfig_Structure(t *testing.T) {
	config := MCPServerConfig{
		Name:    "test_server",
		SSEUrl:  "http://localhost:3001/sse",
		Enabled: true,
	}

	assert.Equal(t, "test_server", config.Name)
	assert.Equal(t, "http://localhost:3001/sse", config.SSEUrl)
	assert.True(t, config.Enabled)
}

func TestReconnectConfig_Structure(t *testing.T) {
	config := ReconnectConfig{
		Interval:    5 * time.Second,
		MaxAttempts: 10,
	}

	assert.Equal(t, 5*time.Second, config.Interval)
	assert.Equal(t, 10, config.MaxAttempts)
}

func TestGlobalMCPConnectionRefreshesToolsOnStandardNotification(t *testing.T) {
	manager := GetGlobalMCPManager()
	transportInstance := &mockGlobalTransport{
		listResponses: []mockGlobalTransportListResponse{
			{
				result: &mcp.ListToolsResult{
					Tools: []mcp.Tool{
						mcp.NewTool("weather", mcp.WithDescription("weather v1")),
					},
				},
			},
			{
				result: &mcp.ListToolsResult{
					Tools: []mcp.Tool{
						mcp.NewTool("weather", mcp.WithDescription("weather v1")),
						mcp.NewTool("forecast", mcp.WithDescription("forecast v2")),
					},
				},
			},
		},
	}

	originalBuildTransport := buildGlobalMCPTransport
	buildGlobalMCPTransport = func(config MCPServerConfig) (transport.Interface, string, error) {
		return transportInstance, "mock://global", nil
	}
	t.Cleanup(func() {
		buildGlobalMCPTransport = originalBuildTransport
	})

	manager.mu.Lock()
	originalTools := manager.tools
	manager.tools = make(map[string]einotool.InvokableTool)
	manager.mu.Unlock()
	t.Cleanup(func() {
		manager.mu.Lock()
		manager.tools = originalTools
		manager.mu.Unlock()
	})

	conn := &MCPServerConnection{
		config: MCPServerConfig{
			Name:    "mock_server",
			Type:    "streamablehttp",
			Url:     "mock://global",
			Enabled: true,
		},
		tools: make(map[string]einotool.InvokableTool),
	}

	require.NoError(t, conn.connect())

	initialTools := manager.GetAllTools()
	require.Contains(t, initialTools, "mock_server_weather")
	require.NotContains(t, initialTools, "mock_server_forecast")

	transportInstance.emitNotification(mcp.JSONRPCNotification{
		JSONRPC: mcp.JSONRPC_VERSION,
		Notification: mcp.Notification{
			Method: mcp.MethodNotificationToolsListChanged,
		},
	})

	require.Eventually(t, func() bool {
		tools := manager.GetAllTools()
		_, exists := tools["mock_server_forecast"]
		return exists
	}, time.Second, 20*time.Millisecond)
}

func TestGlobalMCPManagerSchedulePeriodicToolsRefreshUpdatesGlobalTools(t *testing.T) {
	manager := GetGlobalMCPManager()
	transportInstance := &mockGlobalTransport{
		listResponses: []mockGlobalTransportListResponse{
			{
				result: &mcp.ListToolsResult{
					Tools: []mcp.Tool{
						mcp.NewTool("weather", mcp.WithDescription("weather v1")),
					},
				},
			},
			{
				result: &mcp.ListToolsResult{
					Tools: []mcp.Tool{
						mcp.NewTool("weather", mcp.WithDescription("weather v1")),
						mcp.NewTool("forecast", mcp.WithDescription("forecast v2")),
					},
				},
			},
		},
	}

	originalBuildTransport := buildGlobalMCPTransport
	buildGlobalMCPTransport = func(config MCPServerConfig) (transport.Interface, string, error) {
		return transportInstance, "mock://global", nil
	}
	t.Cleanup(func() {
		buildGlobalMCPTransport = originalBuildTransport
	})

	manager.mu.Lock()
	originalServers := manager.servers
	originalTools := manager.tools
	manager.servers = make(map[string]*MCPServerConnection)
	manager.tools = make(map[string]einotool.InvokableTool)
	manager.mu.Unlock()
	t.Cleanup(func() {
		manager.mu.Lock()
		manager.servers = originalServers
		manager.tools = originalTools
		manager.mu.Unlock()
	})

	conn := &MCPServerConnection{
		config: MCPServerConfig{
			Name:    "periodic_server",
			Type:    "streamablehttp",
			Url:     "mock://global",
			Enabled: true,
		},
		tools: make(map[string]einotool.InvokableTool),
	}

	require.NoError(t, conn.connect())

	manager.mu.Lock()
	manager.servers[conn.config.Name] = conn
	manager.mu.Unlock()

	initialTools := manager.GetAllTools()
	require.Contains(t, initialTools, "periodic_server_weather")
	require.NotContains(t, initialTools, "periodic_server_forecast")

	manager.schedulePeriodicToolsRefresh()

	require.Eventually(t, func() bool {
		tools := manager.GetAllTools()
		_, exists := tools["periodic_server_forecast"]
		return exists
	}, time.Second, 20*time.Millisecond)
}

func TestGlobalMCPConnectionFailsWhenInitialToolsListFails(t *testing.T) {
	manager := GetGlobalMCPManager()
	transportInstance := &mockGlobalTransport{
		listResponses: []mockGlobalTransportListResponse{
			{
				err: fmt.Errorf("tools/list failed"),
			},
		},
	}

	originalBuildTransport := buildGlobalMCPTransport
	buildGlobalMCPTransport = func(config MCPServerConfig) (transport.Interface, string, error) {
		return transportInstance, "mock://global", nil
	}
	t.Cleanup(func() {
		buildGlobalMCPTransport = originalBuildTransport
	})

	manager.mu.Lock()
	originalTools := manager.tools
	manager.tools = make(map[string]einotool.InvokableTool)
	manager.mu.Unlock()
	t.Cleanup(func() {
		manager.mu.Lock()
		manager.tools = originalTools
		manager.mu.Unlock()
	})

	conn := &MCPServerConnection{
		config: MCPServerConfig{
			Name:    "broken_server",
			Type:    "streamablehttp",
			Url:     "mock://global",
			Enabled: true,
		},
		tools: make(map[string]einotool.InvokableTool),
	}

	err := conn.connect()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取工具列表失败")
	assert.False(t, conn.connected)
	assert.Nil(t, conn.client)
	assert.Empty(t, manager.GetAllTools())
	conn.mu.RLock()
	assert.Error(t, conn.lastError)
	conn.mu.RUnlock()
}

// TestMCPGoStructures 测试 mcp-go 库结构体的使用
func TestMCPGoStructures(t *testing.T) {
	t.Run("InitializeRequest", func(t *testing.T) {
		initRequest := mcp.InitializeRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodInitialize),
			},
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				ClientInfo: mcp.Implementation{
					Name:    "test-client",
					Version: "1.0.0",
				},
				Capabilities: mcp.ClientCapabilities{
					Experimental: make(map[string]any),
				},
			},
		}

		assert.Equal(t, string(mcp.MethodInitialize), initRequest.Request.Method)
		assert.Equal(t, "test-client", initRequest.Params.ClientInfo.Name)
	})

	t.Run("JSONRPCRequest", func(t *testing.T) {
		request := mcp.JSONRPCRequest{
			JSONRPC: mcp.JSONRPC_VERSION,
			ID:      mcp.NewRequestId(1),
			Request: mcp.Request{
				Method: string(mcp.MethodToolsList),
			},
		}

		assert.Equal(t, mcp.JSONRPC_VERSION, request.JSONRPC)
		assert.Equal(t, string(mcp.MethodToolsList), request.Request.Method)
	})

	t.Run("Tool", func(t *testing.T) {
		tool := mcp.NewTool(
			"test-tool",
			mcp.WithDescription("A test tool"),
		)

		assert.Equal(t, "test-tool", tool.Name)
		assert.Equal(t, "A test tool", tool.Description)
	})
}

// 创建测试工具
func TestMCPTool_InvokableRun_NewTool(t *testing.T) {
	testTool := &McpTool{
		info: &schema.ToolInfo{
			Name: "test_tool",
			Desc: "测试工具",
		},
		serverName: "test_server",
		client:     nil, // 测试中不需要真实客户端
	}

	// 这个测试会失败，因为没有真实的MCP服务器
	// 但可以验证方法签名和基本逻辑
	_, err := testTool.InvokableRun(context.Background(), `{"query": "test"}`)
	assert.Error(t, err) // 预期会有网络错误
}

func TestIsRetryableRemoteCallError(t *testing.T) {
	assert.True(t, isRetryableRemoteCallError(fmt.Errorf("session closed")))
	assert.True(t, isRetryableRemoteCallError(fmt.Errorf("failed to unmarshal response: unexpected end of JSON input")))
	assert.True(t, isRetryableRemoteCallError(fmt.Errorf("upstream bad gateway: 502")))
	assert.False(t, isRetryableRemoteCallError(fmt.Errorf("tool validation failed")))
}

func TestMCPToolInvokableRunReconnectsOnRetryableRemoteCallError(t *testing.T) {
	originalCallRemoteMCPTool := callRemoteMCPTool
	originalReconnectGlobalMCPServer := reconnectGlobalMCPServer
	t.Cleanup(func() {
		callRemoteMCPTool = originalCallRemoteMCPTool
		reconnectGlobalMCPServer = originalReconnectGlobalMCPServer
	})

	initialClient := new(client.Client)
	reconnectedClient := new(client.Client)
	callCount := 0
	reconnectCount := 0

	callRemoteMCPTool = func(ctx context.Context, cli *client.Client, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		callCount++
		assert.Equal(t, "maps_weather", request.Params.Name)
		if callCount == 1 {
			assert.Same(t, initialClient, cli)
			return nil, fmt.Errorf("failed to unmarshal response: unexpected end of JSON input")
		}

		assert.Same(t, reconnectedClient, cli)
		require.Equal(t, map[string]interface{}{"city": "北京"}, request.Params.Arguments)
		return mcp.NewToolResultText("weather ok"), nil
	}

	reconnectGlobalMCPServer = func(serverName string) (*client.Client, error) {
		reconnectCount++
		assert.Equal(t, "高德地图", serverName)
		return reconnectedClient, nil
	}

	testTool := &McpTool{
		info:       &schema.ToolInfo{Name: "maps_weather", Desc: "天气查询"},
		serverName: "高德地图",
		client:     initialClient,
	}

	result, err := testTool.InvokableRun(context.Background(), `{"city":"北京"}`)
	require.NoError(t, err)
	assert.Contains(t, result, "weather ok")
	assert.Equal(t, 2, callCount)
	assert.Equal(t, 1, reconnectCount)
	assert.Same(t, reconnectedClient, testTool.client)
}

func TestMCPToolInvokableRunUsesOriginNameForRemoteCall(t *testing.T) {
	originalCallRemoteMCPTool := callRemoteMCPTool
	t.Cleanup(func() {
		callRemoteMCPTool = originalCallRemoteMCPTool
	})

	var calledName string
	callRemoteMCPTool = func(ctx context.Context, cli *client.Client, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		calledName = request.Params.Name
		return mcp.NewToolResultText("ok"), nil
	}

	testTool := &McpTool{
		info:       &schema.ToolInfo{Name: "browser_click", Desc: "click"},
		originName: "browser.click",
		serverName: "browser",
		client:     new(client.Client),
	}

	_, err := testTool.InvokableRun(context.Background(), `{}`)
	require.NoError(t, err)
	assert.Equal(t, "browser.click", calledName)
}

func TestConvertMcpToolListToInvokableToolListSanitizesInvalidNames(t *testing.T) {
	tools := []mcp.Tool{
		mcp.NewTool("browser.click", mcp.WithDescription("click")),
		mcp.NewTool("中文工具", mcp.WithDescription("unicode")),
		mcp.NewTool("browser click", mcp.WithDescription("space")),
	}

	converted := ConvertMcpToolListToInvokableToolList(tools, "test_server", new(client.Client))

	require.Contains(t, converted, "browser_click")
	require.Contains(t, converted, "tool_f14843ab")
	require.Contains(t, converted, "browser_click_29a678e7")

	info, err := converted["browser_click"].(*McpTool).Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "browser_click", info.Name)
	assert.Equal(t, "browser.click", converted["browser_click"].(*McpTool).originName)
}

func TestMcpClientInstanceGetToolByNameMatchesOriginName(t *testing.T) {
	sanitizedTool := &McpTool{
		info:       &schema.ToolInfo{Name: "browser_click"},
		originName: "browser.click",
	}
	instance := &McpClientInstance{
		tools: map[string]einotool.InvokableTool{
			"browser_click": sanitizedTool,
		},
	}
	instance.storeToolsSnapshot(instance.tools)

	invokable, ok := instance.getToolByName("browser_click")
	require.True(t, ok)
	assert.Same(t, sanitizedTool, invokable)

	invokable, ok = instance.getToolByName("browser.click")
	require.True(t, ok)
	assert.Same(t, sanitizedTool, invokable)
}

func TestFilterMCPToolsByAllowList(t *testing.T) {
	tools := []mcp.Tool{
		{Name: "alerts"},
		{Name: "forecast"},
		{Name: "history"},
	}

	filtered := filterMCPToolsByAllowList(tools, []string{"forecast", "alerts"})
	require.Len(t, filtered, 2)
	assert.Equal(t, "alerts", filtered[0].Name)
	assert.Equal(t, "forecast", filtered[1].Name)

	unfiltered := filterMCPToolsByAllowList(tools, nil)
	require.Len(t, unfiltered, 3)
}

type testIotConn struct {
	transportType   string
	deviceID        string
	autoRespond     bool
	sent            chan []byte
	recv            chan []byte
	onRequest       func(capturedJSONRPCRequest, []byte)
	sendErrByMethod map[string]error
}

type closeAwareIotConn struct {
	transportType string
	recvEntered   chan struct{}
	ctxCanceled   chan struct{}
	enterOnce     sync.Once
	cancelOnce    sync.Once
}

type dropInboundIotConn struct {
	transportType string
	sent          chan []byte
	handleCount   atomic.Int32
}

type capturedJSONRPCRequest struct {
	Method string          `json:"method"`
	ID     json.RawMessage `json:"id"`
}

type mockGlobalTransportListResponse struct {
	result *mcp.ListToolsResult
	err    error
}

type mockGlobalTransport struct {
	mu                  sync.Mutex
	notificationHandler func(notification mcp.JSONRPCNotification)
	listResponses       []mockGlobalTransportListResponse
	defaultListResult   *mcp.ListToolsResult
	requestMethods      []string
}

func (t *mockGlobalTransport) Start(ctx context.Context) error {
	return nil
}

func (t *mockGlobalTransport) SendRequest(ctx context.Context, request transport.JSONRPCRequest) (*transport.JSONRPCResponse, error) {
	t.mu.Lock()
	t.requestMethods = append(t.requestMethods, request.Method)
	t.mu.Unlock()

	switch request.Method {
	case string(mcp.MethodInitialize):
		return buildJSONRPCTransportSuccessResponse(request.ID, mcp.InitializeResult{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ServerCapabilities{},
			ServerInfo: mcp.Implementation{
				Name:    "mock-global-mcp",
				Version: "1.0.0",
			},
		})
	case string(mcp.MethodToolsList):
		t.mu.Lock()
		var response mockGlobalTransportListResponse
		if len(t.listResponses) > 0 {
			response = t.listResponses[0]
			t.listResponses = t.listResponses[1:]
		} else {
			response = mockGlobalTransportListResponse{result: t.defaultListResult}
		}
		t.mu.Unlock()

		if response.err != nil {
			return nil, response.err
		}
		if response.result == nil {
			response.result = &mcp.ListToolsResult{}
		}
		return buildJSONRPCTransportSuccessResponse(request.ID, response.result)
	case string(mcp.MethodPing):
		return buildJSONRPCTransportSuccessResponse(request.ID, map[string]any{})
	default:
		return nil, fmt.Errorf("unexpected method: %s", request.Method)
	}
}

func (t *mockGlobalTransport) drainRequestMethods() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	methods := append([]string(nil), t.requestMethods...)
	t.requestMethods = t.requestMethods[:0]
	return methods
}

func (t *mockGlobalTransport) SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	return nil
}

func (t *mockGlobalTransport) SetNotificationHandler(handler func(notification mcp.JSONRPCNotification)) {
	t.mu.Lock()
	t.notificationHandler = handler
	t.mu.Unlock()
}

func (t *mockGlobalTransport) Close() error {
	return nil
}

func (t *mockGlobalTransport) GetSessionId() string {
	return "mock-session"
}

func (t *mockGlobalTransport) emitNotification(notification mcp.JSONRPCNotification) {
	t.mu.Lock()
	handler := t.notificationHandler
	t.mu.Unlock()
	if handler != nil {
		handler(notification)
	}
}

func newTestIotConn(transportType string) *testIotConn {
	return &testIotConn{
		transportType:   transportType,
		sent:            make(chan []byte, 8),
		recv:            make(chan []byte, 8),
		sendErrByMethod: make(map[string]error),
	}
}

func newCloseAwareIotConn(transportType string) *closeAwareIotConn {
	return &closeAwareIotConn{
		transportType: transportType,
		recvEntered:   make(chan struct{}),
		ctxCanceled:   make(chan struct{}),
	}
}

func newDropInboundIotConn(transportType string) *dropInboundIotConn {
	return &dropInboundIotConn{
		transportType: transportType,
		sent:          make(chan []byte, 8),
	}
}

func (c *closeAwareIotConn) SendMcpMsg(payload []byte) error {
	return nil
}

func (c *closeAwareIotConn) RecvMcpMsg(ctx context.Context, timeout int) ([]byte, error) {
	c.enterOnce.Do(func() {
		close(c.recvEntered)
	})

	select {
	case <-ctx.Done():
		c.cancelOnce.Do(func() {
			close(c.ctxCanceled)
		})
		return nil, ctx.Err()
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return nil, fmt.Errorf("timeout")
	}
}

func (c *closeAwareIotConn) HandleMcpMessage(payload []byte) error {
	return nil
}

func (c *closeAwareIotConn) GetMcpTransportType() string {
	return c.transportType
}

func (c *dropInboundIotConn) SendMcpMsg(payload []byte) error {
	c.sent <- append([]byte(nil), payload...)
	return nil
}

func (c *dropInboundIotConn) RecvMcpMsg(ctx context.Context, timeout int) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return nil, fmt.Errorf("timeout")
	}
}

func (c *dropInboundIotConn) HandleMcpMessage(payload []byte) error {
	c.handleCount.Add(1)
	return nil
}

func (c *dropInboundIotConn) GetMcpTransportType() string {
	return c.transportType
}

func (c *testIotConn) SendMcpMsg(payload []byte) error {
	var request capturedJSONRPCRequest
	if err := json.Unmarshal(payload, &request); err == nil {
		if sendErr := c.sendErrByMethod[request.Method]; sendErr != nil {
			return sendErr
		}
	}

	c.sent <- append([]byte(nil), payload...)
	if c.autoRespond {
		if err := json.Unmarshal(payload, &request); err == nil {
			if c.onRequest != nil {
				c.onRequest(request, append([]byte(nil), payload...))
			}
			switch request.Method {
			case string(mcp.MethodInitialize):
				if c.deviceID != "" {
					if session := GetDeviceMcpClient(c.deviceID); session != nil {
						session.iotMux.RLock()
						_, ok := session.iotOverMcpByTransport[normalizeDeviceTransportType(c.transportType)]
						session.iotMux.RUnlock()
						if ok {
							c.recv <- buildJSONRPCSuccessResponsePayload(request.ID, mcp.InitializeResult{
								ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
								Capabilities:    mcp.ServerCapabilities{},
								ServerInfo: mcp.Implementation{
									Name:    "test-mcp-server",
									Version: "1.0.0",
								},
							})
						}
					}
				}
			case string(mcp.MethodToolsList):
				c.recv <- buildJSONRPCSuccessResponsePayload(request.ID, mcp.ListToolsResult{
					Tools: []mcp.Tool{
						mcp.NewTool("demo_tool", mcp.WithDescription("demo")),
					},
				})
			}
		}
	} else {
		if err := json.Unmarshal(payload, &request); err == nil && c.onRequest != nil {
			c.onRequest(request, append([]byte(nil), payload...))
		}
	}
	return nil
}

func (c *testIotConn) RecvMcpMsg(ctx context.Context, timeout int) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case payload := <-c.recv:
		return append([]byte(nil), payload...), nil
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return nil, fmt.Errorf("timeout")
	}
}

func (c *testIotConn) GetMcpTransportType() string {
	return c.transportType
}

func (c *testIotConn) HandleMcpMessage(payload []byte) error {
	c.recv <- append([]byte(nil), payload...)
	return nil
}

func buildJSONRPCSuccessResponsePayload(id json.RawMessage, result any) []byte {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}

	payload, err := json.Marshal(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  json.RawMessage `json:"result"`
	}{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultBytes,
	})
	if err != nil {
		panic(err)
	}
	return payload
}

func TestWebsocketTransportHandleMessageRoutesNotification(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wsTransport := &WebsocketTransport{
		respChans:      make(map[string]*pendingResponse),
		readDone:       make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
		requestTimeout: DefaultRequestTimeout,
		closeTimeout:   DefaultCloseTimeout,
	}

	notified := make(chan string, 1)
	wsTransport.SetNotificationHandler(func(notification mcp.JSONRPCNotification) {
		notified <- notification.Method
	})

	wsTransport.handleMessage([]byte(`{"jsonrpc":"2.0","method":"notifications/tools/updated","params":{}}`))

	select {
	case method := <-notified:
		assert.Equal(t, "notifications/tools/updated", method)
	case <-time.After(time.Second):
		t.Fatal("expected notification to be routed to handler")
	}
}

func TestRequestIDKeyNormalizesUint64AndInt64ToSameKey(t *testing.T) {
	requestID := mcp.NewRequestId(uint64(1))
	assert.Equal(t, "1", requestIDKey(requestID))

	var response transport.JSONRPCResponse
	require.NoError(t, json.Unmarshal([]byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`), &response))
	assert.Equal(t, "1", requestIDKey(response.ID))
}

func TestIotOverMcpTransportSendRequestMatchesResponseByID(t *testing.T) {
	conn := newTestIotConn("udp")
	iotTransport, err := NewIotOverMcpTransport(conn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, iotTransport.Close())
	})

	var (
		mu       sync.Mutex
		requests []capturedJSONRPCRequest
	)
	conn.onRequest = func(request capturedJSONRPCRequest, payload []byte) {
		mu.Lock()
		requests = append(requests, request)
		if len(requests) != 2 {
			mu.Unlock()
			return
		}
		first := requests[0]
		second := requests[1]
		mu.Unlock()

		conn.recv <- buildJSONRPCSuccessResponsePayload(second.ID, map[string]any{"request": "second"})
		conn.recv <- buildJSONRPCSuccessResponsePayload(first.ID, map[string]any{"request": "first"})
	}

	type requestResult struct {
		response *transport.JSONRPCResponse
		err      error
	}

	req1 := transport.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      mcp.NewRequestId("req-1"),
		Method:  string(mcp.MethodPing),
	}
	req2 := transport.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      mcp.NewRequestId("req-2"),
		Method:  string(mcp.MethodToolsList),
	}

	resultCh1 := make(chan requestResult, 1)
	resultCh2 := make(chan requestResult, 1)

	go func() {
		response, err := iotTransport.SendRequest(context.Background(), req1)
		resultCh1 <- requestResult{response: response, err: err}
	}()
	go func() {
		response, err := iotTransport.SendRequest(context.Background(), req2)
		resultCh2 <- requestResult{response: response, err: err}
	}()

	var result1, result2 requestResult
	select {
	case result1 = <-resultCh1:
	case <-time.After(2 * time.Second):
		t.Fatal("first request timed out")
	}
	select {
	case result2 = <-resultCh2:
	case <-time.After(2 * time.Second):
		t.Fatal("second request timed out")
	}

	require.NoError(t, result1.err)
	require.NoError(t, result2.err)
	require.NotNil(t, result1.response)
	require.NotNil(t, result2.response)
	assert.Equal(t, "string:req-1", result1.response.ID.String())
	assert.Equal(t, "string:req-2", result2.response.ID.String())
}

func TestIotOverMcpTransportSendRequestMatchesUint64ResponseID(t *testing.T) {
	conn := newTestIotConn("udp")
	iotTransport, err := NewIotOverMcpTransport(conn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, iotTransport.Close())
	})

	req := transport.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      mcp.NewRequestId(uint64(1)),
		Method:  string(mcp.MethodInitialize),
	}

	go func() {
		payload := <-conn.sent
		var captured capturedJSONRPCRequest
		require.NoError(t, json.Unmarshal(payload, &captured))
		conn.recv <- buildJSONRPCSuccessResponsePayload(captured.ID, map[string]any{"ok": true})
	}()

	response, err := iotTransport.SendRequest(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "int64:1", response.ID.String())
}

func TestIotOverMcpTransportSendNotificationSendsPayload(t *testing.T) {
	conn := newTestIotConn("udp")
	iotTransport, err := NewIotOverMcpTransport(conn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, iotTransport.Close())
	})

	notification := mcp.JSONRPCNotification{
		JSONRPC: "2.0",
		Notification: mcp.Notification{
			Method: string(mcp.MethodNotificationToolsListChanged),
		},
	}
	require.NoError(t, iotTransport.SendNotification(context.Background(), notification))

	select {
	case payload := <-conn.sent:
		var actual mcp.JSONRPCNotification
		require.NoError(t, json.Unmarshal(payload, &actual))
		assert.Equal(t, notification.Method, actual.Method)
	case <-time.After(time.Second):
		t.Fatal("expected notification payload to be sent")
	}
}

func TestGlobalMCPManagerDisconnectRemovesGlobalTools(t *testing.T) {
	manager := GetGlobalMCPManager()

	manager.mu.Lock()
	originalServers := manager.servers
	originalTools := manager.tools
	manager.servers = make(map[string]*MCPServerConnection)
	manager.tools = make(map[string]einotool.InvokableTool)
	conn := &MCPServerConnection{
		config: MCPServerConfig{Name: "test-global-server"},
		tools:  make(map[string]einotool.InvokableTool),
	}
	manager.servers["test-global-server"] = conn
	manager.tools["test-global-server_demo"] = &McpTool{
		info:       &schema.ToolInfo{Name: "demo"},
		serverName: "test-global-server",
	}
	manager.mu.Unlock()

	t.Cleanup(func() {
		manager.mu.Lock()
		manager.servers = originalServers
		manager.tools = originalTools
		manager.mu.Unlock()
	})

	require.NoError(t, conn.disconnect())

	manager.mu.RLock()
	_, exists := manager.tools["test-global-server_demo"]
	manager.mu.RUnlock()
	assert.False(t, exists)
}

func buildJSONRPCSuccessResponse(t *testing.T, id json.RawMessage, result any) []byte {
	t.Helper()
	return buildJSONRPCSuccessResponsePayload(id, result)
}

func TestEnsureDeviceIotOverMcp_RegistersTransportBeforeInitialize(t *testing.T) {
	deviceID := fmt.Sprintf("test-device-%d", time.Now().UnixNano())
	conn := newTestIotConn("websocket")
	conn.deviceID = deviceID
	conn.autoRespond = true

	t.Cleanup(func() {
		CloseDeviceIotOverMcp(deviceID, conn)
		if session := GetDeviceMcpClient(deviceID); session != nil && session.cancel != nil {
			session.cancel()
		}
		_ = RemoveDeviceMcpClient(deviceID)
	})

	require.NoError(t, EnsureDeviceIotOverMcp(deviceID, conn))

	session := GetDeviceMcpClient(deviceID)
	require.NotNil(t, session)

	tool, ok := session.GetIotToolByTransportAndName(conn.GetMcpTransportType(), "demo_tool")
	require.True(t, ok)
	require.NotNil(t, tool)
}

func TestEnsureDeviceIotOverMcp_DeduplicatesConcurrentInitialize(t *testing.T) {
	deviceID := fmt.Sprintf("dedupe-device-%d", time.Now().UnixNano())
	conn := newTestIotConn("websocket")
	conn.deviceID = deviceID

	var initializeRequests int32
	initStarted := make(chan struct{})
	releaseInit := make(chan struct{})
	conn.onRequest = func(request capturedJSONRPCRequest, _ []byte) {
		switch request.Method {
		case string(mcp.MethodInitialize):
			if atomic.AddInt32(&initializeRequests, 1) == 1 {
				close(initStarted)
				<-releaseInit
			}
			conn.recv <- buildJSONRPCSuccessResponsePayload(request.ID, mcp.InitializeResult{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				Capabilities:    mcp.ServerCapabilities{},
				ServerInfo: mcp.Implementation{
					Name:    "test-mcp-server",
					Version: "1.0.0",
				},
			})
		case string(mcp.MethodToolsList):
			conn.recv <- buildJSONRPCSuccessResponsePayload(request.ID, mcp.ListToolsResult{
				Tools: []mcp.Tool{
					mcp.NewTool("demo_tool", mcp.WithDescription("demo")),
				},
			})
		}
	}

	t.Cleanup(func() {
		select {
		case <-releaseInit:
		default:
			close(releaseInit)
		}
		CloseDeviceIotOverMcp(deviceID, conn)
		if session := GetDeviceMcpClient(deviceID); session != nil && session.cancel != nil {
			session.cancel()
		}
		_ = RemoveDeviceMcpClient(deviceID)
	})

	firstErrCh := make(chan error, 1)
	go func() {
		firstErrCh <- EnsureDeviceIotOverMcp(deviceID, conn)
	}()

	select {
	case <-initStarted:
	case <-time.After(time.Second):
		t.Fatal("expected first initialize request to start")
	}

	secondErrCh := make(chan error, 1)
	go func() {
		secondErrCh <- EnsureDeviceIotOverMcp(deviceID, conn)
	}()

	select {
	case err := <-secondErrCh:
		require.NoError(t, err)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected duplicate ensure to return without waiting for initialize")
	}

	close(releaseInit)
	require.NoError(t, <-firstErrCh)
	assert.Equal(t, int32(1), atomic.LoadInt32(&initializeRequests))
}

func TestHandleDeviceIotMcpMessage_RoutesPayloadToCurrentTransport(t *testing.T) {
	deviceID := fmt.Sprintf("route-device-%d", time.Now().UnixNano())
	conn := newTestIotConn("websocket")

	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}
	session.iotOverMcpByTransport[normalizeDeviceTransportType(conn.transportType)] = markTestClientConnected(&McpClientInstance{
		serverName: buildIotServerName(deviceID, conn.transportType),
		conn:       conn,
	}, time.Now())
	require.NoError(t, AddDeviceMcpClient(deviceID, session))

	t.Cleanup(func() {
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	payload := []byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`)
	require.NoError(t, HandleDeviceIotMcpMessage(deviceID, conn.transportType, payload))

	select {
	case routed := <-conn.recv:
		assert.Equal(t, payload, routed)
	case <-time.After(time.Second):
		t.Fatal("expected payload to be routed to current transport")
	}
}

func TestShouldScheduleDeviceIotOverMcp_UsesTransportInitState(t *testing.T) {
	deviceID := fmt.Sprintf("schedule-device-%d", time.Now().UnixNano())
	conn := newTestIotConn("websocket")
	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}

	instance := &McpClientInstance{
		serverName: buildIotServerName(deviceID, conn.transportType),
		conn:       conn,
	}
	instance.setConnected(true)
	instance.setInitState(mcpClientInitStateInitializing)
	session.iotOverMcpByTransport[normalizeDeviceTransportType(conn.transportType)] = instance
	require.NoError(t, AddDeviceMcpClient(deviceID, session))

	t.Cleanup(func() {
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	assert.False(t, ShouldScheduleDeviceIotOverMcp(deviceID, conn))

	instance.setInitState(mcpClientInitStateReady)
	assert.False(t, ShouldScheduleDeviceIotOverMcp(deviceID, conn))

	instance.setInitState(mcpClientInitStateIdle)
	assert.True(t, ShouldScheduleDeviceIotOverMcp(deviceID, conn))

	otherConn := newTestIotConn("websocket")
	assert.True(t, ShouldScheduleDeviceIotOverMcp(deviceID, otherConn))
}

func TestHandleDeviceIotMcpMessage_DirectlyInjectsIntoActiveTransport(t *testing.T) {
	deviceID := fmt.Sprintf("inject-device-%d", time.Now().UnixNano())
	conn := newDropInboundIotConn("udp")
	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}
	instance := NewIotOverMcpClient(deviceID, conn.transportType, conn)
	require.NotNil(t, instance)
	instance.SetOnCloseHandler(session.handleMcpClientClose)
	session.iotOverMcpByTransport[normalizeDeviceTransportType(conn.transportType)] = instance
	require.NoError(t, AddDeviceMcpClient(deviceID, session))

	t.Cleanup(func() {
		cancel()
		instance.closeWithReason("test_cleanup")
		_ = RemoveDeviceMcpClient(deviceID)
	})

	reqCtx, reqCancel := context.WithTimeout(context.Background(), time.Second)
	defer reqCancel()

	resultCh := make(chan error, 1)
	go func() {
		_, err := instance.iotTransport.SendRequest(reqCtx, transport.JSONRPCRequest{
			JSONRPC: mcp.JSONRPC_VERSION,
			ID:      mcp.NewRequestId(uint64(2)),
			Method:  string(mcp.MethodToolsList),
			Params:  map[string]any{},
		})
		resultCh <- err
	}()

	select {
	case <-conn.sent:
	case <-time.After(time.Second):
		t.Fatal("expected transport to send tools/list request")
	}

	require.NoError(t, HandleDeviceIotMcpMessage(deviceID, conn.transportType, []byte(`{"jsonrpc":"2.0","id":2,"result":{"tools":[]}}`)))

	select {
	case err := <-resultCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("expected MCP response to resolve the pending request")
	}

	assert.Zero(t, conn.handleCount.Load(), "direct injection should not fall back to conn.HandleMcpMessage")
}

func TestCloseDeviceIotOverMcpClosesUnderlyingTransport(t *testing.T) {
	deviceID := fmt.Sprintf("close-iot-device-%d", time.Now().UnixNano())
	conn := newCloseAwareIotConn("websocket")
	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}
	instance := NewIotOverMcpClient(deviceID, conn.transportType, conn)
	require.NotNil(t, instance)
	instance.SetOnCloseHandler(session.handleMcpClientClose)
	session.iotOverMcpByTransport[normalizeDeviceTransportType(conn.transportType)] = instance
	require.NoError(t, AddDeviceMcpClient(deviceID, session))

	t.Cleanup(func() {
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	select {
	case <-conn.recvEntered:
	case <-time.After(time.Second):
		t.Fatal("expected IoT transport read loop to start")
	}

	CloseDeviceIotOverMcp(deviceID, conn)

	select {
	case <-conn.ctxCanceled:
	case <-time.After(time.Second):
		t.Fatal("expected closing device IoT MCP to cancel the transport read loop")
	}
}

func TestHeartbeatRefreshToolsFailure_DestroysRuntimeAfterFiveFailures(t *testing.T) {
	deviceID := fmt.Sprintf("heartbeat-refresh-fail-%d", time.Now().UnixNano())
	conn := newTestIotConn("websocket")
	conn.deviceID = deviceID
	conn.autoRespond = true

	t.Cleanup(func() {
		if session := GetDeviceMcpClient(deviceID); session != nil && session.cancel != nil {
			session.cancel()
		}
		_ = RemoveDeviceMcpClient(deviceID)
	})

	require.NoError(t, EnsureDeviceIotOverMcp(deviceID, conn))

	session := GetDeviceMcpClient(deviceID)
	require.NotNil(t, session)

	session.iotMux.RLock()
	instance := session.iotOverMcpByTransport[normalizeDeviceTransportType(conn.transportType)]
	session.iotMux.RUnlock()
	require.NotNil(t, instance)

	conn.sendErrByMethod[string(mcp.MethodToolsList)] = fmt.Errorf("list tools failed")
	for i := 1; i < heartbeatRefreshFailureLimit; i++ {
		session.heartbeatMcpInstance(instance)

		session = GetDeviceMcpClient(deviceID)
		require.NotNil(t, session)
		assert.EqualValues(t, i, instance.RefreshFailureCount())
		assert.True(t, instance.IsConnected())
	}

	session.heartbeatMcpInstance(instance)

	session = GetDeviceMcpClient(deviceID)
	assert.Nil(t, session)
}

func TestHeartbeatRefreshToolsFailure_SuccessResetsCounter(t *testing.T) {
	deviceID := fmt.Sprintf("heartbeat-refresh-reset-%d", time.Now().UnixNano())
	conn := newTestIotConn("websocket")
	conn.deviceID = deviceID
	conn.autoRespond = true

	t.Cleanup(func() {
		if session := GetDeviceMcpClient(deviceID); session != nil && session.cancel != nil {
			session.cancel()
		}
		_ = RemoveDeviceMcpClient(deviceID)
	})

	require.NoError(t, EnsureDeviceIotOverMcp(deviceID, conn))

	session := GetDeviceMcpClient(deviceID)
	require.NotNil(t, session)

	session.iotMux.RLock()
	instance := session.iotOverMcpByTransport[normalizeDeviceTransportType(conn.transportType)]
	session.iotMux.RUnlock()
	require.NotNil(t, instance)

	conn.sendErrByMethod[string(mcp.MethodToolsList)] = fmt.Errorf("list tools failed")
	for i := 1; i < heartbeatRefreshFailureLimit; i++ {
		session.heartbeatMcpInstance(instance)
		assert.EqualValues(t, i, instance.RefreshFailureCount())
	}

	delete(conn.sendErrByMethod, string(mcp.MethodToolsList))
	session.heartbeatMcpInstance(instance)
	assert.Zero(t, instance.RefreshFailureCount())

	conn.sendErrByMethod[string(mcp.MethodToolsList)] = fmt.Errorf("list tools failed again")
	for i := 1; i < heartbeatRefreshFailureLimit; i++ {
		session.heartbeatMcpInstance(instance)
		session = GetDeviceMcpClient(deviceID)
		require.NotNil(t, session)
		assert.EqualValues(t, i, instance.RefreshFailureCount())
	}
}

func buildJSONRPCTransportSuccessResponse(id mcp.RequestId, result any) (*transport.JSONRPCResponse, error) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &transport.JSONRPCResponse{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      id,
		Result:  resultBytes,
	}, nil
}

func TestHeartbeatIotRuntimeSkipsPingAndRefreshesLastPing(t *testing.T) {
	deviceID := fmt.Sprintf("heartbeat-iot-skip-ping-%d", time.Now().UnixNano())
	conn := newTestIotConn("websocket")
	conn.deviceID = deviceID
	conn.autoRespond = true

	var methodsMu sync.Mutex
	methods := make([]string, 0, 4)
	conn.onRequest = func(request capturedJSONRPCRequest, _ []byte) {
		methodsMu.Lock()
		methods = append(methods, request.Method)
		methodsMu.Unlock()
	}

	t.Cleanup(func() {
		if session := GetDeviceMcpClient(deviceID); session != nil && session.cancel != nil {
			session.cancel()
		}
		_ = RemoveDeviceMcpClient(deviceID)
	})

	require.NoError(t, EnsureDeviceIotOverMcp(deviceID, conn))

	session := GetDeviceMcpClient(deviceID)
	require.NotNil(t, session)

	session.iotMux.RLock()
	instance := session.iotOverMcpByTransport[normalizeDeviceTransportType(conn.transportType)]
	session.iotMux.RUnlock()
	require.NotNil(t, instance)

	oldPing := time.Unix(100, 0)
	instance.setLastPing(oldPing)

	methodsMu.Lock()
	methods = methods[:0]
	methodsMu.Unlock()

	session.heartbeatMcpInstance(instance)

	assert.True(t, instance.LastPing().After(oldPing))

	methodsMu.Lock()
	defer methodsMu.Unlock()
	assert.Contains(t, methods, string(mcp.MethodToolsList))
	assert.NotContains(t, methods, string(mcp.MethodPing))
}

func TestHeartbeatWsEndpointRuntimeSkipsToolsRefreshBeforeTenMinutesAndPings(t *testing.T) {
	transportInstance := &mockGlobalTransport{
		defaultListResult: &mcp.ListToolsResult{
			Tools: []mcp.Tool{
				mcp.NewTool("weather", mcp.WithDescription("weather")),
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	instance := &McpClientInstance{
		serverName: "ws_endpoint_mcp_test",
		mcpClient:  client.NewClient(transportInstance),
		Ctx:        ctx,
		cancel:     cancel,
	}
	instance.storeToolsSnapshot(make(map[string]einotool.InvokableTool))
	instance.setConnected(true)
	require.NoError(t, instance.sendInitlize(context.Background()))

	oldPing := time.Unix(100, 0)
	instance.setLastPing(oldPing)
	instance.setLastToolsRefresh(time.Now().Add(-5 * time.Minute))

	session := &DeviceMcpSession{}
	transportInstance.drainRequestMethods()

	session.heartbeatMcpInstance(instance)

	assert.True(t, instance.LastPing().After(oldPing))
	methods := transportInstance.drainRequestMethods()
	assert.Contains(t, methods, string(mcp.MethodPing))
	assert.NotContains(t, methods, string(mcp.MethodToolsList))
}

func TestHeartbeatWsEndpointRuntimeRefreshesToolsAfterTenMinutesAndPings(t *testing.T) {
	transportInstance := &mockGlobalTransport{
		defaultListResult: &mcp.ListToolsResult{
			Tools: []mcp.Tool{
				mcp.NewTool("weather", mcp.WithDescription("weather")),
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	instance := &McpClientInstance{
		serverName: "ws_endpoint_mcp_test",
		mcpClient:  client.NewClient(transportInstance),
		Ctx:        ctx,
		cancel:     cancel,
	}
	instance.storeToolsSnapshot(make(map[string]einotool.InvokableTool))
	instance.setConnected(true)
	require.NoError(t, instance.sendInitlize(context.Background()))

	oldPing := time.Unix(100, 0)
	oldRefresh := time.Now().Add(-11 * time.Minute)
	instance.setLastPing(oldPing)
	instance.setLastToolsRefresh(oldRefresh)

	session := &DeviceMcpSession{}
	transportInstance.drainRequestMethods()

	session.heartbeatMcpInstance(instance)

	assert.True(t, instance.LastPing().After(oldPing))
	assert.True(t, instance.LastToolsRefresh().After(oldRefresh))
	methods := transportInstance.drainRequestMethods()
	assert.Equal(t, []string{string(mcp.MethodPing), string(mcp.MethodToolsList)}, methods)
}

func TestHeartbeatWsEndpointRuntimePingsBeforeToolsRefreshFailure(t *testing.T) {
	transportInstance := &mockGlobalTransport{
		listResponses: []mockGlobalTransportListResponse{
			{err: fmt.Errorf("list tools failed")},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	instance := &McpClientInstance{
		serverName: "ws_endpoint_mcp_test",
		mcpClient:  client.NewClient(transportInstance),
		Ctx:        ctx,
		cancel:     cancel,
	}
	instance.storeToolsSnapshot(make(map[string]einotool.InvokableTool))
	instance.setConnected(true)
	require.NoError(t, instance.sendInitlize(context.Background()))

	oldPing := time.Unix(100, 0)
	oldRefresh := time.Now().Add(-11 * time.Minute)
	instance.setLastPing(oldPing)
	instance.setLastToolsRefresh(oldRefresh)

	session := &DeviceMcpSession{}
	transportInstance.drainRequestMethods()

	session.heartbeatMcpInstance(instance)

	assert.True(t, instance.LastPing().After(oldPing))
	assert.EqualValues(t, 1, instance.RefreshFailureCount())
	assert.True(t, instance.IsConnected())
	assert.Equal(t, []string{string(mcp.MethodPing), string(mcp.MethodToolsList)}, transportInstance.drainRequestMethods())
}

func TestSweepOfflineClientsKeepsWsEndpointWithinOfflineTimeout(t *testing.T) {
	pool := newTestMcpClientPool()
	deviceID := fmt.Sprintf("ws-endpoint-offline-window-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientCtx, clientCancel := context.WithCancel(ctx)
	defer clientCancel()

	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}
	instance := markTestClientConnected(&McpClientInstance{
		serverName: "ws_endpoint_mcp_test",
		Ctx:        clientCtx,
		cancel:     clientCancel,
	}, time.Now().Add(-(deviceMCPPingInterval + 30*time.Second)))
	instance.SetOnCloseHandler(session.handleMcpClientClose)
	session.wsEndPointMcp.Store(instance.serverName, instance)
	pool.AddMcpClient(deviceID, session)

	pool.sweepOfflineClients()

	assert.True(t, instance.IsConnected())
	assert.NoError(t, clientCtx.Err())
	assert.NotNil(t, pool.GetMcpClient(deviceID))
}

func TestSweepOfflineClientsClosesWsEndpointAfterOfflineTimeout(t *testing.T) {
	pool := newTestMcpClientPool()
	deviceID := fmt.Sprintf("ws-endpoint-offline-timeout-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientCtx, clientCancel := context.WithCancel(ctx)
	defer clientCancel()

	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}
	instance := markTestClientConnected(&McpClientInstance{
		serverName: "ws_endpoint_mcp_test",
		Ctx:        clientCtx,
		cancel:     clientCancel,
	}, time.Now().Add(-(mcpConnectionOfflineTimeout + time.Second)))
	instance.SetOnCloseHandler(session.handleMcpClientClose)
	session.wsEndPointMcp.Store(instance.serverName, instance)
	pool.AddMcpClient(deviceID, session)

	pool.sweepOfflineClients()

	assert.False(t, instance.IsConnected())
	assert.ErrorIs(t, clientCtx.Err(), context.Canceled)
	assert.Nil(t, pool.GetMcpClient(deviceID))
}

func TestGetToolByNameWithTransport_PrefersCurrentTransport(t *testing.T) {
	deviceID := fmt.Sprintf("transport-device-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}

	wsTool := &McpTool{
		info: &schema.ToolInfo{Name: "shared_tool"},
	}
	udpTool := &McpTool{
		info: &schema.ToolInfo{Name: "shared_tool"},
	}

	session.iotOverMcpByTransport["websocket"] = markTestClientConnected(&McpClientInstance{
		serverName: buildIotServerName(deviceID, "websocket"),
		tools: map[string]einotool.InvokableTool{
			"shared_tool": wsTool,
		},
	}, time.Now())
	session.iotOverMcpByTransport["mqtt_udp"] = markTestClientConnected(&McpClientInstance{
		serverName: buildIotServerName(deviceID, "mqtt_udp"),
		tools: map[string]einotool.InvokableTool{
			"shared_tool": udpTool,
		},
	}, time.Now())
	require.NoError(t, AddDeviceMcpClient(deviceID, session))

	t.Cleanup(func() {
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	invokable, ok := GetToolByNameWithTransport(deviceID, "", "websocket", "shared_tool", "")
	require.True(t, ok)
	assert.Same(t, wsTool, invokable)

	invokable, ok = GetToolByNameWithTransport(deviceID, "", "mqtt_udp", "shared_tool", "")
	require.True(t, ok)
	assert.Same(t, udpTool, invokable)

	tools, err := GetToolsByDeviceIdWithTransport(deviceID, "", "websocket", "")
	require.NoError(t, err)
	require.Contains(t, tools, "shared_tool")
	assert.Same(t, wsTool, tools["shared_tool"])
}

func TestGetReportedToolsByDeviceID_RequiresCurrentOnlineTransport(t *testing.T) {
	deviceID := fmt.Sprintf("reported-tools-no-resolver-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}

	wsTool := &McpTool{info: &schema.ToolInfo{Name: "shared_tool"}}
	session.iotOverMcpByTransport["websocket"] = markTestClientConnected(&McpClientInstance{
		serverName: buildIotServerName(deviceID, "websocket"),
		tools: map[string]einotool.InvokableTool{
			"shared_tool": wsTool,
		},
	}, time.Unix(300, 0))
	require.NoError(t, AddDeviceMcpClient(deviceID, session))
	RegisterCurrentDeviceTransportResolver(nil)

	t.Cleanup(func() {
		RegisterCurrentDeviceTransportResolver(nil)
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	reportedTools, err := GetReportedToolsByDeviceID(deviceID)
	require.NoError(t, err)
	assert.Empty(t, reportedTools)

	invokable, ok := GetReportedToolByDeviceIDAndName(deviceID, "shared_tool")
	require.False(t, ok)
	assert.Nil(t, invokable)
}

func TestGetReportedToolsByDeviceID_ReturnsEmptyWhenResolverReturnsEmpty(t *testing.T) {
	deviceID := fmt.Sprintf("reported-tools-empty-resolver-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}

	udpTool := &McpTool{info: &schema.ToolInfo{Name: "shared_tool"}}
	session.iotOverMcpByTransport["udp"] = markTestClientConnected(&McpClientInstance{
		serverName: buildIotServerName(deviceID, "udp"),
		tools: map[string]einotool.InvokableTool{
			"shared_tool": udpTool,
		},
	}, time.Unix(100, 0))
	require.NoError(t, AddDeviceMcpClient(deviceID, session))
	RegisterCurrentDeviceTransportResolver(func(id string) string {
		return ""
	})

	t.Cleanup(func() {
		RegisterCurrentDeviceTransportResolver(nil)
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	reportedTools, err := GetReportedToolsByDeviceID(deviceID)
	require.NoError(t, err)
	assert.Empty(t, reportedTools)

	invokable, ok := GetReportedToolByDeviceIDAndName(deviceID, "shared_tool")
	require.False(t, ok)
	assert.Nil(t, invokable)
}

func TestGetReportedToolsByDeviceID_UsesResolvedCurrentTransport(t *testing.T) {
	deviceID := fmt.Sprintf("reported-tools-current-transport-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}

	wsOnlyTool := &McpTool{info: &schema.ToolInfo{Name: "ws_only"}}
	wsSharedTool := &McpTool{info: &schema.ToolInfo{Name: "shared_tool"}}
	udpSharedTool := &McpTool{info: &schema.ToolInfo{Name: "shared_tool"}}
	udpOnlyTool := &McpTool{info: &schema.ToolInfo{Name: "udp_only"}}

	session.wsEndPointMcp.Store("ws-endpoint", &McpClientInstance{
		serverName: "ws-endpoint",
		tools: map[string]einotool.InvokableTool{
			"ws_only":     wsOnlyTool,
			"shared_tool": wsSharedTool,
		},
	})
	session.iotOverMcpByTransport["websocket"] = markTestClientConnected(&McpClientInstance{
		serverName: buildIotServerName(deviceID, "websocket"),
		tools: map[string]einotool.InvokableTool{
			"shared_tool": wsSharedTool,
		},
	}, time.Unix(300, 0))
	session.iotOverMcpByTransport["udp"] = markTestClientConnected(&McpClientInstance{
		serverName: buildIotServerName(deviceID, "udp"),
		tools: map[string]einotool.InvokableTool{
			"shared_tool": udpSharedTool,
			"udp_only":    udpOnlyTool,
		},
	}, time.Unix(100, 0))
	require.NoError(t, AddDeviceMcpClient(deviceID, session))
	RegisterCurrentDeviceTransportResolver(func(id string) string {
		if id == deviceID {
			return "udp"
		}
		return ""
	})

	t.Cleanup(func() {
		RegisterCurrentDeviceTransportResolver(nil)
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	reportedTools, err := GetReportedToolsByDeviceID(deviceID)
	require.NoError(t, err)
	require.Contains(t, reportedTools, "shared_tool")
	assert.Same(t, udpSharedTool, reportedTools["shared_tool"])
	require.Contains(t, reportedTools, "udp_only")
	assert.Same(t, udpOnlyTool, reportedTools["udp_only"])
	assert.NotContains(t, reportedTools, "ws_only")

	invokable, ok := GetReportedToolByDeviceIDAndName(deviceID, "shared_tool")
	require.True(t, ok)
	assert.Same(t, udpSharedTool, invokable)

	invokable, ok = GetReportedToolByDeviceIDAndName(deviceID, "ws_only")
	require.False(t, ok)
	assert.Nil(t, invokable)
}

func TestGetReportedToolsByDeviceID_IgnoresUnsupportedIotTransport(t *testing.T) {
	deviceID := fmt.Sprintf("reported-tools-ignore-unsupported-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}

	serialTool := &McpTool{info: &schema.ToolInfo{Name: "serial_only"}}
	udpTool := &McpTool{info: &schema.ToolInfo{Name: "udp_only"}}

	session.iotOverMcpByTransport["serial"] = markTestClientConnected(&McpClientInstance{
		serverName: buildIotServerName(deviceID, "serial"),
		tools: map[string]einotool.InvokableTool{
			"serial_only": serialTool,
		},
	}, time.Unix(300, 0))
	session.iotOverMcpByTransport["udp"] = markTestClientConnected(&McpClientInstance{
		serverName: buildIotServerName(deviceID, "udp"),
		tools: map[string]einotool.InvokableTool{
			"udp_only": udpTool,
		},
	}, time.Unix(100, 0))
	require.NoError(t, AddDeviceMcpClient(deviceID, session))
	RegisterCurrentDeviceTransportResolver(func(id string) string {
		if id == deviceID {
			return "udp"
		}
		return ""
	})

	t.Cleanup(func() {
		RegisterCurrentDeviceTransportResolver(nil)
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	reportedTools, err := GetReportedToolsByDeviceID(deviceID)
	require.NoError(t, err)
	require.Contains(t, reportedTools, "udp_only")
	assert.Same(t, udpTool, reportedTools["udp_only"])
	assert.NotContains(t, reportedTools, "serial_only")

	invokable, ok := GetReportedToolByDeviceIDAndName(deviceID, "serial_only")
	require.False(t, ok)
	assert.Nil(t, invokable)
}

func TestRefreshReportedToolsByDeviceID_UsesRemoteToolsListAndUpdatesSnapshot(t *testing.T) {
	deviceID := fmt.Sprintf("refresh-reported-tools-device-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())

	transportInstance := &mockGlobalTransport{
		defaultListResult: &mcp.ListToolsResult{
			Tools: []mcp.Tool{
				mcp.NewTool("fresh_tool", mcp.WithDescription("fresh tool from remote")),
			},
		},
	}

	instance := &McpClientInstance{
		serverName: buildIotServerName(deviceID, "udp"),
		mcpClient:  client.NewClient(transportInstance),
		Ctx:        ctx,
		cancel:     cancel,
		tools: map[string]einotool.InvokableTool{
			"stale_tool": &McpTool{info: &schema.ToolInfo{Name: "stale_tool"}},
		},
	}
	instance.storeToolsSnapshot(instance.tools)
	instance.setConnected(true)
	instance.setInitState(mcpClientInitStateReady)
	require.NoError(t, instance.sendInitlize(context.Background()))
	transportInstance.drainRequestMethods()

	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: map[string]*McpClientInstance{"udp": instance},
	}
	require.NoError(t, AddDeviceMcpClient(deviceID, session))
	RegisterCurrentDeviceTransportResolver(func(id string) string {
		if id == deviceID {
			return "udp"
		}
		return ""
	})

	t.Cleanup(func() {
		RegisterCurrentDeviceTransportResolver(nil)
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	reportedTools, err := RefreshReportedToolsByDeviceID(deviceID)
	require.NoError(t, err)
	require.Contains(t, reportedTools, "fresh_tool")
	assert.NotContains(t, reportedTools, "stale_tool")
	assert.Contains(t, instance.loadToolsSnapshot(), "fresh_tool")
	assert.NotContains(t, instance.loadToolsSnapshot(), "stale_tool")
	assert.Contains(t, transportInstance.drainRequestMethods(), string(mcp.MethodToolsList))
}

func TestRefreshReportedToolsByDeviceID_ClearsSnapshotOnFailure(t *testing.T) {
	deviceID := fmt.Sprintf("refresh-reported-tools-device-fail-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())

	transportInstance := &mockGlobalTransport{
		listResponses: []mockGlobalTransportListResponse{
			{err: fmt.Errorf("tools/list failed")},
		},
	}

	instance := &McpClientInstance{
		serverName: buildIotServerName(deviceID, "udp"),
		mcpClient:  client.NewClient(transportInstance),
		Ctx:        ctx,
		cancel:     cancel,
		tools: map[string]einotool.InvokableTool{
			"stale_tool": &McpTool{info: &schema.ToolInfo{Name: "stale_tool"}},
		},
	}
	instance.storeToolsSnapshot(instance.tools)
	instance.setConnected(true)
	instance.setInitState(mcpClientInitStateReady)
	require.NoError(t, instance.sendInitlize(context.Background()))

	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: map[string]*McpClientInstance{"udp": instance},
	}
	require.NoError(t, AddDeviceMcpClient(deviceID, session))
	RegisterCurrentDeviceTransportResolver(func(id string) string {
		if id == deviceID {
			return "udp"
		}
		return ""
	})

	t.Cleanup(func() {
		RegisterCurrentDeviceTransportResolver(nil)
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	reportedTools, err := RefreshReportedToolsByDeviceID(deviceID)
	require.Error(t, err)
	assert.Empty(t, reportedTools)
	assert.Empty(t, instance.loadToolsSnapshot())
}

func TestRefreshReportedToolsByAgentID_ClearsSnapshotOnFailure(t *testing.T) {
	agentID := fmt.Sprintf("refresh-reported-tools-agent-fail-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())

	transportInstance := &mockGlobalTransport{
		listResponses: []mockGlobalTransportListResponse{
			{err: fmt.Errorf("tools/list failed")},
		},
	}

	instance := &McpClientInstance{
		serverName: "ws_endpoint_mcp_test",
		mcpClient:  client.NewClient(transportInstance),
		Ctx:        ctx,
		cancel:     cancel,
		tools: map[string]einotool.InvokableTool{
			"stale_tool": &McpTool{info: &schema.ToolInfo{Name: "stale_tool"}},
		},
	}
	instance.storeToolsSnapshot(instance.tools)
	instance.setConnected(true)
	instance.setInitState(mcpClientInitStateReady)
	require.NoError(t, instance.sendInitlize(context.Background()))

	session := &DeviceMcpSession{
		deviceID:              agentID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}
	session.wsEndPointMcp.Store(instance.serverName, instance)
	require.NoError(t, AddDeviceMcpClient(agentID, session))

	t.Cleanup(func() {
		cancel()
		_ = RemoveDeviceMcpClient(agentID)
	})

	reportedTools, err := RefreshReportedToolsByAgentID(agentID)
	require.Error(t, err)
	assert.Empty(t, reportedTools)
	assert.Empty(t, instance.loadToolsSnapshot())
}

func TestGetWsEndpointConnectionStatusRequiresInitializedClient(t *testing.T) {
	agentID := fmt.Sprintf("ws-endpoint-status-agent-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())

	session := &DeviceMcpSession{
		deviceID:              agentID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}

	notReady := &McpClientInstance{serverName: "ws_endpoint_mcp_not_ready"}
	notReady.setConnected(true)
	session.wsEndPointMcp.Store(notReady.serverName, notReady)

	require.NoError(t, AddDeviceMcpClient(agentID, session))
	t.Cleanup(func() {
		cancel()
		_ = RemoveDeviceMcpClient(agentID)
	})

	connected, count := GetWsEndpointConnectionStatus(agentID)
	require.False(t, connected)
	require.Equal(t, 0, count)

	ready := &McpClientInstance{
		serverName: "ws_endpoint_mcp_ready",
		serverInfo: &mcp.InitializeResult{},
	}
	ready.setConnected(true)
	session.wsEndPointMcp.Store(ready.serverName, ready)

	connected, count = GetWsEndpointConnectionStatus(agentID)
	require.True(t, connected)
	require.Equal(t, 1, count)
}

func TestRawCallReportedToolByDeviceID_UsesResolvedTransport(t *testing.T) {
	deviceID := fmt.Sprintf("raw-call-device-%d", time.Now().UnixNano())
	conn := newTestIotConn("udp")
	conn.onRequest = func(request capturedJSONRPCRequest, _ []byte) {
		switch request.Method {
		case string(mcp.MethodInitialize):
			conn.recv <- buildJSONRPCSuccessResponsePayload(request.ID, mcp.InitializeResult{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				Capabilities:    mcp.ServerCapabilities{},
				ServerInfo: mcp.Implementation{
					Name:    "test-mcp-server",
					Version: "1.0.0",
				},
			})
		case string(mcp.MethodToolsCall):
			conn.recv <- buildJSONRPCSuccessResponsePayload(request.ID, *mcp.NewToolResultText("raw device call ok"))
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}
	instance := NewIotOverMcpClient(deviceID, "udp", conn)
	require.NotNil(t, instance)
	require.NoError(t, instance.sendInitlize(context.Background()))
	session.iotOverMcpByTransport["udp"] = instance
	require.NoError(t, AddDeviceMcpClient(deviceID, session))
	RegisterCurrentDeviceTransportResolver(func(id string) string {
		if id == deviceID {
			return "udp"
		}
		return ""
	})

	t.Cleanup(func() {
		RegisterCurrentDeviceTransportResolver(nil)
		cancel()
		_ = RemoveDeviceMcpClient(deviceID)
	})

	_, ok := GetReportedToolByDeviceIDAndName(deviceID, "hidden_tool")
	require.False(t, ok)

	result, rawCalled, err := RawCallReportedToolByDeviceID(deviceID, "hidden_tool", map[string]interface{}{"mode": "debug"})
	require.True(t, rawCalled)
	require.NoError(t, err)
	assert.Contains(t, result, "raw device call ok")
}

func TestHandleDeviceIotMcpMessageMarksIotRuntimeActive(t *testing.T) {
	deviceID := fmt.Sprintf("iot-message-activity-%d", time.Now().UnixNano())
	conn := newTestIotConn("websocket")
	conn.deviceID = deviceID
	conn.autoRespond = true

	t.Cleanup(func() {
		if session := GetDeviceMcpClient(deviceID); session != nil && session.cancel != nil {
			session.cancel()
		}
		_ = RemoveDeviceMcpClient(deviceID)
	})

	require.NoError(t, EnsureDeviceIotOverMcp(deviceID, conn))

	session := GetDeviceMcpClient(deviceID)
	require.NotNil(t, session)

	session.iotMux.RLock()
	instance := session.iotOverMcpByTransport[normalizeDeviceTransportType(conn.transportType)]
	session.iotMux.RUnlock()
	require.NotNil(t, instance)

	oldPing := time.Unix(100, 0)
	instance.setLastPing(oldPing)

	err := HandleDeviceIotMcpMessage(deviceID, conn.transportType, []byte(`{"jsonrpc":"2.0","id":999,"result":{"ok":true}}`))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return instance.LastPing().After(oldPing)
	}, time.Second, 20*time.Millisecond)
}

func TestRawCallReportedToolByAgentID_UsesWsEndpointRuntime(t *testing.T) {
	agentID := fmt.Sprintf("raw-call-agent-%d", time.Now().UnixNano())
	conn := newTestIotConn("websocket")
	conn.onRequest = func(request capturedJSONRPCRequest, _ []byte) {
		switch request.Method {
		case string(mcp.MethodInitialize):
			conn.recv <- buildJSONRPCSuccessResponsePayload(request.ID, mcp.InitializeResult{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				Capabilities:    mcp.ServerCapabilities{},
				ServerInfo: mcp.Implementation{
					Name:    "test-mcp-server",
					Version: "1.0.0",
				},
			})
		case string(mcp.MethodToolsCall):
			conn.recv <- buildJSONRPCSuccessResponsePayload(request.ID, *mcp.NewToolResultText("raw agent call ok"))
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &DeviceMcpSession{
		deviceID:              agentID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
	}
	instance := NewIotOverMcpClient(agentID, "websocket", conn)
	require.NotNil(t, instance)
	require.NoError(t, instance.sendInitlize(context.Background()))
	session.wsEndPointMcp.Store("ws-endpoint", instance)
	require.NoError(t, AddDeviceMcpClient(agentID, session))

	t.Cleanup(func() {
		cancel()
		_ = RemoveDeviceMcpClient(agentID)
	})

	_, ok := GetReportedToolByAgentIDAndName(agentID, "hidden_tool")
	require.False(t, ok)

	result, rawCalled, err := RawCallReportedToolByAgentID(agentID, "hidden_tool", map[string]interface{}{"mode": "debug"})
	require.True(t, rawCalled)
	require.NoError(t, err)
	assert.Contains(t, result, "raw agent call ok")
}
