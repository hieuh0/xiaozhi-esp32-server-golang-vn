package mcp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	deviceMCPPingInterval          = 2 * time.Minute
	wsEndpointToolsRefreshInterval = 10 * time.Minute
	heartbeatRefreshFailureLimit   = 5
	mcpConnectionOfflineTimeout    = 3 * deviceMCPPingInterval
)

// DeviceMcpSession represents the MCP session of a device and aggregates multiple MCP connections.
type DeviceMcpSession struct {
	deviceID              string
	Ctx                   context.Context
	cancel                context.CancelFunc
	wsEndPointMcp         sync.Map
	iotOverMcpByTransport map[string]*McpClientInstance
	iotMux                sync.RWMutex
}

type mcpClientInitState uint32

const (
	mcpClientInitStateIdle mcpClientInitState = iota
	mcpClientInitStateInitializing
	mcpClientInitStateReady
)

func normalizeDeviceTransportType(transportType string) string {
	transportType = strings.TrimSpace(transportType)
	if transportType == "" {
		return "unknown"
	}
	return transportType
}

func buildIotServerName(deviceID, transportType string) string {
	return fmt.Sprintf("iot_over_mcp_%s_%s", deviceID, normalizeDeviceTransportType(transportType))
}

func (dcs *DeviceMcpSession) AddWsEndPointMcp(mcpClient *McpClientInstance) {
	dcs.wsEndPointMcp.Store(mcpClient.serverName, mcpClient)

	//Set close callback
	mcpClient.SetOnCloseHandler(dcs.handleMcpClientClose)

	mcpClient.refreshTools()
}

func (dcs *DeviceMcpSession) SetIotOverMcp(transportType string, mcpClient *McpClientInstance) {
	transportType = normalizeDeviceTransportType(transportType)
	if mcpClient != nil {
		mcpClient.SetOnCloseHandler(dcs.handleMcpClientClose)
	}

	var old *McpClientInstance
	dcs.iotMux.Lock()
	//Keep single instance with device + transportType
	if existing := dcs.iotOverMcpByTransport[transportType]; existing != nil && existing != mcpClient {
		old = existing
	}
	dcs.iotOverMcpByTransport[transportType] = mcpClient
	dcs.iotMux.Unlock()

	//Close the old instance outside the lock to avoid executing cancellation logic in the session lock.
	if old != nil {
		old.closeWithReason("iot_transport_replaced")
	}
}

func (dcs *DeviceMcpSession) RemoveWsEndPointMcp(mcpClient *McpClientInstance) {
	dcs.wsEndPointMcp.Delete(mcpClient.serverName)
}

// GetDeviceID Get device ID
func (dcs *DeviceMcpSession) GetDeviceID() string {
	return dcs.deviceID
}

// handleMcpClientClose handles MCP client closing events
func (dcs *DeviceMcpSession) handleMcpClientClose(instance *McpClientInstance, reason string) {
	logger.Infof("MCP client %s for device %s has been closed due to: %s", dcs.deviceID, instance.serverName, reason)

	//Remove a closed client from a session
	dcs.RemoveWsEndPointMcp(instance)
	dcs.removeIotOverMcpByInstance(instance)

	if !dcs.hasAnyClient() {
		logger.Infof("All MCP connections for device %s have been closed, clean up sessions", dcs.deviceID)
		dcs.cancel()
		mcpClientPool.RemoveMcpClient(dcs.deviceID)
	}
}

func (dcs *DeviceMcpSession) removeIotOverMcpByInstance(instance *McpClientInstance) {
	dcs.iotMux.Lock()
	defer dcs.iotMux.Unlock()
	for transportType, iotClient := range dcs.iotOverMcpByTransport {
		if iotClient == instance {
			delete(dcs.iotOverMcpByTransport, transportType)
		}
	}
}

func (dcs *DeviceMcpSession) hasAnyClient() bool {
	hasWsClient := false
	dcs.wsEndPointMcp.Range(func(_, _ interface{}) bool {
		hasWsClient = true
		return false
	})
	if hasWsClient {
		return true
	}

	dcs.iotMux.RLock()
	defer dcs.iotMux.RUnlock()
	return len(dcs.iotOverMcpByTransport) > 0
}

// McpClientInstance represents a specific MCP client connection
type McpClientInstance struct {
	serverName       string
	mcpClient        *client.Client //It is the mcp server connected from ws endpoint.
	tools            map[string]tool.InvokableTool
	toolsState       atomic.Value //map[string]tool.InvokableTool, replace the whole when refreshing, and take a snapshot of the reading path
	serverInfo       *mcp.InitializeResult
	Ctx              context.Context
	cancel           context.CancelFunc
	conn             ConnInterface
	iotTransport     *IotOverMcpTransport
	initState        uint32
	lastPing         atomic.Int64
	lastToolsRefresh atomic.Int64
	refreshFailures  atomic.Int32
	connected        atomic.Bool

	//Add close callback
	onCloseHandler func(instance *McpClientInstance, reason string)
	closed         atomic.Bool
}

// NewDeviceMCPClient creates a new MCP client
func NewDeviceMCPSession(deviceID string) *DeviceMcpSession {
	ctx, cancel := context.WithCancel(context.Background())

	deviceMcpClient := &DeviceMcpSession{
		deviceID:              deviceID,
		Ctx:                   ctx,
		cancel:                cancel,
		iotOverMcpByTransport: make(map[string]*McpClientInstance),
		iotMux:                sync.RWMutex{},
		// wsEndPointMcp: make(map[string]*McpClientInstance),
	}

	go deviceMcpClient.refreshToolsAndPing()

	return deviceMcpClient
}

func NewWsEndPointMcpClient(ctx context.Context, deviceID string, conn *websocket.Conn) *McpClientInstance {
	ctx, cancel := context.WithCancel(ctx)

	wsTransport, err := NewWebsocketTransport(conn)
	if err != nil {
		logger.Errorf("Failed to create MCP client: %v", err)
		return nil
	}
	mcpClient := client.NewClient(wsTransport)

	wsEndPointMcp := &McpClientInstance{
		serverName: fmt.Sprintf("ws_endpoint_mcp_%s_%s", deviceID, conn.RemoteAddr().String()),
		mcpClient:  mcpClient,
		Ctx:        ctx,
		cancel:     cancel,
		initState:  uint32(mcpClientInitStateReady),
	}
	wsEndPointMcp.storeToolsSnapshot(make(map[string]tool.InvokableTool))
	wsEndPointMcp.setConnected(true)
	wsEndPointMcp.setLastPing(time.Now())
	mcpClient.OnNotification(wsEndPointMcp.handleJSONRPCNotification)

	//Set the transport shutdown callback
	wsTransport.SetOnCloseHandler(wsEndPointMcp.handleTransportClose)

	wsEndPointMcp.sendInitlize(ctx)
	wsEndPointMcp.mcpClient.Start(ctx)
	return wsEndPointMcp
}

func NewIotOverMcpClient(deviceID string, transportType string, conn ConnInterface) *McpClientInstance {
	ctx, cancel := context.WithCancel(context.Background())

	iotTransport, err := NewIotOverMcpTransport(conn)
	if err != nil {
		logger.Errorf("Failed to create MCP client: %v", err)
		return nil
	}
	mcpClient := client.NewClient(iotTransport)

	iotOverMcp := &McpClientInstance{
		serverName:   buildIotServerName(deviceID, transportType),
		mcpClient:    mcpClient,
		Ctx:          ctx,
		cancel:       cancel,
		conn:         conn,
		iotTransport: iotTransport,
		initState:    uint32(mcpClientInitStateInitializing),
	}
	iotOverMcp.storeToolsSnapshot(make(map[string]tool.InvokableTool))
	iotOverMcp.setConnected(true)
	iotOverMcp.setLastPing(time.Now())
	iotTransport.SetNotificationHandler(iotOverMcp.handleJSONRPCNotification)
	iotTransport.SetActivityHandler(func() {
		iotOverMcp.setLastPing(time.Now())
	})

	//Set the transport shutdown callback
	iotTransport.SetOnCloseHandler(iotOverMcp.handleTransportClose)

	return iotOverMcp
}

func (dc *McpClientInstance) startIotOverMcp() error {
	if err := dc.sendInitlize(dc.Ctx); err != nil {
		return err
	}
	dc.mcpClient.Start(dc.Ctx)
	return dc.refreshTools()
}

// refreshToolsCommon common tool list refresh logic
func (dc *McpClientInstance) refreshTools() error {
	_, err := dc.refreshToolsWithPolicy(false)
	return err
}

func (dc *McpClientInstance) refreshToolsStrict() (map[string]tool.InvokableTool, error) {
	return dc.refreshToolsWithPolicy(true)
}

func (dc *McpClientInstance) refreshToolsWithPolicy(clearOnFailure bool) (map[string]tool.InvokableTool, error) {
	emptyTools := make(map[string]tool.InvokableTool)
	if dc == nil || dc.mcpClient == nil {
		err := fmt.Errorf("mcp client not initialized")
		if clearOnFailure {
			dc.clearToolsSnapshot()
		}
		return emptyTools, err
	}
	if dc.serverInfo == nil {
		err := fmt.Errorf("client not initialized")
		if clearOnFailure {
			dc.clearToolsSnapshot()
		}
		return emptyTools, err
	}

	tools, err := dc.mcpClient.ListTools(dc.Ctx, mcp.ListToolsRequest{})
	if err != nil {
		logger.Errorf("Failed to refresh tool list: %v", err)
		if clearOnFailure {
			dc.clearToolsSnapshot()
			logger.Warnf("Failed to refresh the tool list, the memory tool snapshot of %s has been cleared", dc.serverName)
		}
		return emptyTools, err
	}

	//Tool conversion may be heavy, so it should be completed outside the lock first to avoid blocking the path for reading the tool list.
	convertedTools := ConvertMcpToolListToInvokableToolList(tools.Tools, dc.serverName, dc.mcpClient)

	dc.storeToolsSnapshot(convertedTools)
	dc.setLastPing(time.Now())
	dc.setLastToolsRefresh(time.Now())
	dc.resetRefreshFailures()

	logger.Infof("Refresh tool list successfully: %s obtained %d tools", dc.serverName, len(convertedTools))
	return convertedTools, nil
}

func (dc *McpClientInstance) GetServerName() string {
	return dc.serverName
}

func (dc *McpClientInstance) IsInitialized() bool {
	return dc != nil && dc.serverInfo != nil
}

func (dc *McpClientInstance) storeToolsSnapshot(tools map[string]tool.InvokableTool) {
	if dc == nil {
		return
	}
	if tools == nil {
		tools = make(map[string]tool.InvokableTool)
	}
	dc.tools = tools
	dc.toolsState.Store(tools)
}

func (dc *McpClientInstance) clearToolsSnapshot() {
	if dc == nil {
		return
	}
	dc.storeToolsSnapshot(make(map[string]tool.InvokableTool))
	dc.setLastToolsRefresh(time.Time{})
}

func (dc *McpClientInstance) loadToolsSnapshot() map[string]tool.InvokableTool {
	if dc == nil {
		return nil
	}
	if snapshot := dc.toolsState.Load(); snapshot != nil {
		if tools, ok := snapshot.(map[string]tool.InvokableTool); ok {
			return tools
		}
	}
	return dc.tools
}

func (dc *McpClientInstance) copyToolsInto(dst map[string]tool.InvokableTool) {
	if dc == nil {
		return
	}
	for name, invokable := range dc.loadToolsSnapshot() {
		dst[name] = invokable
	}
}

func (dc *McpClientInstance) toolCount() int {
	return len(dc.loadToolsSnapshot())
}

func (dc *McpClientInstance) getToolByName(toolName string) (tool.InvokableTool, bool) {
	tools := dc.loadToolsSnapshot()
	return findInvokableToolByName(tools, toolName)
}

func (dc *McpClientInstance) setConnected(connected bool) {
	if dc == nil {
		return
	}
	dc.connected.Store(connected)
}

func (dc *McpClientInstance) setLastPing(ts time.Time) {
	if dc == nil {
		return
	}
	if ts.IsZero() {
		dc.lastPing.Store(0)
		return
	}
	dc.lastPing.Store(ts.UnixNano())
}

func (dc *McpClientInstance) setLastToolsRefresh(ts time.Time) {
	if dc == nil {
		return
	}
	if ts.IsZero() {
		dc.lastToolsRefresh.Store(0)
		return
	}
	dc.lastToolsRefresh.Store(ts.UnixNano())
}

func (dc *McpClientInstance) incrementRefreshFailures() int32 {
	if dc == nil {
		return 0
	}
	return dc.refreshFailures.Add(1)
}

func (dc *McpClientInstance) resetRefreshFailures() {
	if dc == nil {
		return
	}
	dc.refreshFailures.Store(0)
}

func (dc *McpClientInstance) RefreshFailureCount() int32 {
	if dc == nil {
		return 0
	}
	return dc.refreshFailures.Load()
}

func (dc *McpClientInstance) LastPing() time.Time {
	if dc == nil {
		return time.Time{}
	}
	unixNano := dc.lastPing.Load()
	if unixNano == 0 {
		return time.Time{}
	}
	return time.Unix(0, unixNano)
}

func (dc *McpClientInstance) LastToolsRefresh() time.Time {
	if dc == nil {
		return time.Time{}
	}
	unixNano := dc.lastToolsRefresh.Load()
	if unixNano == 0 {
		return time.Time{}
	}
	return time.Unix(0, unixNano)
}

func (dc *McpClientInstance) getInitState() mcpClientInitState {
	if dc == nil {
		return mcpClientInitStateIdle
	}
	return mcpClientInitState(atomic.LoadUint32(&dc.initState))
}

func (dc *McpClientInstance) setInitState(state mcpClientInitState) {
	if dc == nil {
		return
	}
	atomic.StoreUint32(&dc.initState, uint32(state))
}

func (dc *McpClientInstance) IsInitializing() bool {
	return dc.getInitState() == mcpClientInitStateInitializing
}

func (dc *McpClientInstance) IsReady() bool {
	return dc.getInitState() == mcpClientInitStateReady
}

func (dc *McpClientInstance) closeWithReason(reason string) {
	if dc == nil {
		return
	}
	if !dc.closed.CompareAndSwap(false, true) {
		return
	}
	logger.Infof("MCP client %s closed, reason: %s", dc.serverName, reason)

	dc.setConnected(false)
	dc.setInitState(mcpClientInitStateIdle)
	if dc.cancel != nil {
		dc.cancel()
	}
	if dc.mcpClient != nil && reason != "connection_closed" && reason != "manual_close" {
		if err := dc.mcpClient.Close(); err != nil {
			logger.Warnf("Close MCP client %s transport failed: %v", dc.serverName, err)
		}
	}

	if dc.onCloseHandler != nil {
		dc.onCloseHandler(dc, reason)
	}
}

func (dc *DeviceMcpSession) snapshotWsEndpointClients() []*McpClientInstance {
	clients := make([]*McpClientInstance, 0)
	dc.wsEndPointMcp.Range(func(_, value interface{}) bool {
		mcpInstance, ok := value.(*McpClientInstance)
		if ok && mcpInstance != nil {
			clients = append(clients, mcpInstance)
		}
		return true
	})
	return clients
}

func (dc *DeviceMcpSession) GetWsEndpointConnectionStatus() (bool, int) {
	connectedCount := 0
	for _, mcpInstance := range dc.snapshotWsEndpointClients() {
		if mcpInstance != nil && mcpInstance.IsConnected() && mcpInstance.IsInitialized() {
			connectedCount++
		}
	}
	return connectedCount > 0, connectedCount
}

func (dc *DeviceMcpSession) snapshotIotClients() []*McpClientInstance {
	dc.iotMux.RLock()
	defer dc.iotMux.RUnlock()

	clients := make([]*McpClientInstance, 0, len(dc.iotOverMcpByTransport))
	for _, instance := range dc.iotOverMcpByTransport {
		if instance != nil {
			clients = append(clients, instance)
		}
	}
	return clients
}

type iotTransportClientSnapshot struct {
	transportType string
	client        *McpClientInstance
}

func (dc *DeviceMcpSession) snapshotIotTransports() []iotTransportClientSnapshot {
	dc.iotMux.RLock()
	defer dc.iotMux.RUnlock()

	clients := make([]iotTransportClientSnapshot, 0, len(dc.iotOverMcpByTransport))
	for transportType, instance := range dc.iotOverMcpByTransport {
		if instance != nil {
			clients = append(clients, iotTransportClientSnapshot{
				transportType: transportType,
				client:        instance,
			})
		}
	}
	return clients
}

func (dc *DeviceMcpSession) heartbeatMcpInstance(mcpInstance *McpClientInstance) {
	if mcpInstance == nil || !mcpInstance.IsInitialized() {
		return
	}
	if mcpInstance.conn != nil {
		if err := mcpInstance.refreshTools(); err != nil {
			dc.handleHeartbeatRefreshFailure(mcpInstance, err)
			return
		}
		logger.Debugf("Device %s keeps IoT MCP alive through tools/list heartbeat", mcpInstance.serverName)
		return
	}
	err := mcpInstance.mcpClient.Ping(mcpInstance.Ctx)
	if err == nil {
		mcpInstance.setLastPing(time.Now())
		logger.Debugf("Device %s pinged successfully", mcpInstance.serverName)
	} else {
		logger.Warnf("Device %s ping failed: %v", mcpInstance.serverName, err)
	}
	if lastRefresh := mcpInstance.LastToolsRefresh(); lastRefresh.IsZero() || time.Since(lastRefresh) >= wsEndpointToolsRefreshInterval {
		if err := mcpInstance.refreshTools(); err != nil {
			dc.handleHeartbeatRefreshFailure(mcpInstance, err)
			return
		}
	}
}

func (dc *DeviceMcpSession) handleHeartbeatRefreshFailure(mcpInstance *McpClientInstance, err error) {
	failures := mcpInstance.incrementRefreshFailures()
	if failures < heartbeatRefreshFailureLimit {
		logger.Warnf(
			"Device %s heartbeat tool refresh failed (%d/%d); keeping runtime active: %v",
			mcpInstance.serverName,
			failures,
			heartbeatRefreshFailureLimit,
			err,
		)
		return
	}

	logger.Warnf(
		"Device %s heartbeat tool refresh failed %d consecutive times; destroying runtime: %v",
		mcpInstance.serverName,
		failures,
		err,
	)
	mcpInstance.closeWithReason("refresh_tools_failed")
}

func (dc *DeviceMcpSession) refreshToolsAndPing() {
	//Get the tool list only once on initialization
	findTools := func(mcpInstance *McpClientInstance) {
		if mcpInstance == nil || !mcpInstance.IsInitialized() {
			return
		}
		mcpInstance.refreshTools()
	}

	//Get the tool list during initialization
	for _, instance := range dc.snapshotWsEndpointClients() {
		findTools(instance)
	}

	for _, instance := range dc.snapshotIotClients() {
		findTools(instance)
	}

	//A heartbeat every 2 minutes
	pingTick := time.NewTicker(deviceMCPPingInterval)
	defer pingTick.Stop()

	for {
		select {
		case <-dc.Ctx.Done():
			logger.Infof("Device %s session canceled, stop pinging", dc.deviceID)
			return
		case <-pingTick.C:
			for _, instance := range dc.snapshotWsEndpointClients() {
				dc.heartbeatMcpInstance(instance)
			}
			for _, instance := range dc.snapshotIotClients() {
				dc.heartbeatMcpInstance(instance)
			}
		}
	}
}

func (dc *McpClientInstance) sendInitlize(ctx context.Context) error {
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "mcp-go",
				Version: "0.1.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	}

	serverInfo, err := dc.mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		fmt.Printf("Failed to initialize: %v", err)
		return err
	}
	dc.serverInfo = serverInfo
	return nil
}

func (dc *McpClientInstance) findTools() (*mcp.ListToolsResult, error) {
	tools, err := dc.mcpClient.ListTools(dc.Ctx, mcp.ListToolsRequest{})
	if err != nil {
		logger.Errorf("Failed to obtain tool list: %v", err)
		return nil, err
	}
	return tools, nil
}

// handleJSONRPCNotification handles JSON-RPC notifications
func (dc *McpClientInstance) handleJSONRPCNotification(notification mcp.JSONRPCNotification) {
	switch notification.Method {
	case "notifications/progress":
		//handleProgressNotification(notification)
	case "notifications/message":
		//handleMessageNotification(notification)
	case "notifications/resources/updated":
		//handleResourceUpdateNotification(notification)
	case "notifications/tools/updated":
		//Receive tool update notification and refresh tool list
		logger.Infof("Receive tool update notification and refresh tool list")
		go dc.refreshToolsOnNotification()
	default:
		log.Printf("Unknown notification: %s", notification.Method)
	}
}

// refreshToolsOnNotification refreshes the tool list based on notifications
func (dc *McpClientInstance) refreshToolsOnNotification() {
	//Add a short delay to avoid frequent refreshes
	time.Sleep(100 * time.Millisecond)
	dc.refreshTools()
}

// handleJSONRPCError handles JSON-RPC errors
func (dc *McpClientInstance) handleJSONRPCError(errMsg mcp.JSONRPCError) error {
	logger.Errorf("Received MCP server error: %+v", errMsg.Error)
	return nil
}

// handleTransportClose handles the transport layer closing event
func (dc *McpClientInstance) handleTransportClose(reason string) {
	dc.closeWithReason(reason)
}

// SetOnCloseHandler sets the close callback
func (dc *McpClientInstance) SetOnCloseHandler(handler func(instance *McpClientInstance, reason string)) {
	dc.onCloseHandler = handler
}

// IsConnected checks if the connection is still active
func (dc *McpClientInstance) IsConnected() bool {
	if dc == nil {
		return false
	}
	return dc.connected.Load()
}

func (dc *DeviceMcpSession) ShouldScheduleIotInit(transportType string, conn ConnInterface) bool {
	transportType = normalizeDeviceTransportType(transportType)
	if transportType == "unknown" || conn == nil {
		return false
	}

	dc.iotMux.RLock()
	existing := dc.iotOverMcpByTransport[transportType]
	dc.iotMux.RUnlock()
	if existing == nil {
		return true
	}
	if existing.conn != conn {
		return true
	}

	switch existing.getInitState() {
	case mcpClientInitStateReady, mcpClientInitStateInitializing:
		return false
	default:
		return true
	}
}

// GetConnectionStatus Gets connection status information
func (dc *McpClientInstance) GetConnectionStatus() map[string]interface{} {
	toolsCount := dc.toolCount()

	initState := "idle"
	switch dc.getInitState() {
	case mcpClientInitStateInitializing:
		initState = "initializing"
	case mcpClientInitStateReady:
		initState = "ready"
	}

	return map[string]interface{}{
		"server_name": dc.serverName,
		"connected":   dc.IsConnected(),
		"init_state":  initState,
		"last_ping":   dc.LastPing(),
		"tools_count": toolsCount,
	}
}

func (dc *McpClientInstance) RawCallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	if dc == nil || dc.mcpClient == nil {
		return "", fmt.Errorf("MCP client not initialized")
	}
	if !dc.IsConnected() || !dc.IsInitialized() {
		return "", fmt.Errorf("MCP client not ready")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := dc.mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	})
	if err != nil {
		return "", fmt.Errorf("Failed to call tool: %v", err)
	}

	resultBytes, err := result.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("Tool call returns content conversion failure: %v", err)
	}
	return string(resultBytes), nil
}

// GetTools gets the list of tools
func (dc *DeviceMcpSession) GetTools() map[string]tool.InvokableTool {
	tools := make(map[string]tool.InvokableTool)
	for _, mcpInstance := range dc.snapshotWsEndpointClients() {
		mcpInstance.copyToolsInto(tools)
	}

	for _, iotClient := range dc.snapshotIotClients() {
		iotClient.copyToolsInto(tools)
	}
	return tools
}

func (dc *DeviceMcpSession) GetWsEndpointMcpTools() map[string]tool.InvokableTool {
	tools := make(map[string]tool.InvokableTool)
	for _, mcpInstance := range dc.snapshotWsEndpointClients() {
		mcpInstance.copyToolsInto(tools)
	}
	return tools
}

func (dc *DeviceMcpSession) RefreshWsEndpointTools() (map[string]tool.InvokableTool, error) {
	tools := make(map[string]tool.InvokableTool)
	for _, mcpInstance := range dc.snapshotWsEndpointClients() {
		refreshedTools, err := mcpInstance.refreshToolsStrict()
		if err != nil {
			for _, cleanupTarget := range dc.snapshotWsEndpointClients() {
				cleanupTarget.clearToolsSnapshot()
			}
			return map[string]tool.InvokableTool{}, err
		}
		for name, invokable := range refreshedTools {
			tools[name] = invokable
		}
	}
	return tools, nil
}

// GetPreferredIotTransportType Returns the most suitable transport for the current device for device dimension MCP queries/calls.
// Priority is given to the transport that is still connected and has a recent heartbeat; if neither is active, the last existing transport is returned.
func (dc *DeviceMcpSession) GetPreferredIotTransportType() string {
	preferredTransport := ""
	var preferredClient *McpClientInstance
	isSupportedTransport := func(transportType string) bool {
		switch normalizeDeviceTransportType(transportType) {
		case "websocket", "udp", "mqtt_udp":
			return true
		default:
			return false
		}
	}

	selectPreferred := func(connectedOnly bool) string {
		preferredTransport = ""
		preferredClient = nil
		for _, snapshot := range dc.snapshotIotTransports() {
			transportType := snapshot.transportType
			iotClient := snapshot.client
			transportType = normalizeDeviceTransportType(transportType)
			if iotClient == nil {
				continue
			}
			if !isSupportedTransport(transportType) {
				continue
			}
			if connectedOnly && !iotClient.IsConnected() {
				continue
			}
			if preferredClient == nil {
				preferredTransport = transportType
				preferredClient = iotClient
				continue
			}
			currentLastPing := iotClient.LastPing()
			preferredLastPing := preferredClient.LastPing()
			if currentLastPing.After(preferredLastPing) {
				preferredTransport = transportType
				preferredClient = iotClient
				continue
			}
			if currentLastPing.Equal(preferredLastPing) && transportType < preferredTransport {
				preferredTransport = transportType
				preferredClient = iotClient
			}
		}
		return preferredTransport
	}

	if transportType := selectPreferred(true); transportType != "" {
		return transportType
	}
	return selectPreferred(false)
}

func (dc *DeviceMcpSession) GetIotToolsByTransport(transportType string) map[string]tool.InvokableTool {
	transportType = strings.TrimSpace(transportType)
	tools := make(map[string]tool.InvokableTool)
	if transportType == "" {
		return tools
	}

	dc.iotMux.RLock()
	iotClient := dc.iotOverMcpByTransport[transportType]
	dc.iotMux.RUnlock()
	if iotClient == nil {
		return tools
	}

	iotClient.copyToolsInto(tools)

	return tools
}

func (dc *DeviceMcpSession) RefreshIotToolsByTransport(transportType string) (map[string]tool.InvokableTool, error) {
	transportType = normalizeDeviceTransportType(transportType)
	tools := make(map[string]tool.InvokableTool)
	if transportType == "unknown" {
		return tools, nil
	}

	dc.iotMux.RLock()
	iotClient := dc.iotOverMcpByTransport[transportType]
	dc.iotMux.RUnlock()
	if iotClient == nil {
		return tools, nil
	}

	return iotClient.refreshToolsStrict()
}

func (dc *DeviceMcpSession) GetIotToolByTransportAndName(transportType, toolName string) (tool.InvokableTool, bool) {
	transportType = strings.TrimSpace(transportType)
	if transportType == "" {
		return nil, false
	}

	dc.iotMux.RLock()
	iotClient := dc.iotOverMcpByTransport[transportType]
	dc.iotMux.RUnlock()
	if iotClient == nil {
		return nil, false
	}

	return iotClient.getToolByName(toolName)
}

func (dc *DeviceMcpSession) RawCallIotToolByTransport(ctx context.Context, transportType, toolName string, arguments map[string]interface{}) (string, bool, error) {
	transportType = strings.TrimSpace(transportType)
	if transportType == "" {
		return "", false, nil
	}

	dc.iotMux.RLock()
	iotClient := dc.iotOverMcpByTransport[transportType]
	dc.iotMux.RUnlock()
	if iotClient == nil || !iotClient.IsConnected() || !iotClient.IsInitialized() {
		return "", false, nil
	}

	if invokable, ok := iotClient.getToolByName(toolName); ok {
		toolName = remoteCallNameForTool(invokable, toolName)
	}
	result, err := iotClient.RawCallTool(ctx, toolName, arguments)
	return result, true, err
}

func (dc *DeviceMcpSession) RawCallWsEndpointTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, bool, error) {
	var selected *McpClientInstance
	for _, mcpInstance := range dc.snapshotWsEndpointClients() {
		if mcpInstance == nil || !mcpInstance.IsConnected() || !mcpInstance.IsInitialized() {
			continue
		}
		selected = mcpInstance
		break
	}
	if selected == nil {
		return "", false, nil
	}

	if invokable, ok := selected.getToolByName(toolName); ok {
		toolName = remoteCallNameForTool(invokable, toolName)
	}
	result, err := selected.RawCallTool(ctx, toolName, arguments)
	return result, true, err
}

func (dc *DeviceMcpSession) GetToolByName(toolName string) (tool tool.InvokableTool, ok bool) {
	for _, mcpInstance := range dc.snapshotWsEndpointClients() {
		if tool, ok = mcpInstance.getToolByName(toolName); ok {
			return tool, true
		}
	}
	if ok {
		return tool, true
	}

	for _, iotClient := range dc.snapshotIotClients() {
		if tool, ok = iotClient.getToolByName(toolName); ok {
			return tool, true
		}
	}
	return nil, false
}
