package mcp

import (
	"context"
	"fmt"
	"strings"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/components/tool"
	mcp_go "github.com/mark3labs/mcp-go/mcp"
)

func parseSelectedMCPServiceNames(raw string) map[string]struct{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	selected := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		selected[name] = struct{}{}
	}
	if len(selected) == 0 {
		return nil
	}
	return selected
}

func isGlobalToolAllowed(toolKey string, selected map[string]struct{}) bool {
	if len(selected) == 0 {
		return true
	}
	for serviceName := range selected {
		if strings.HasPrefix(toolKey, serviceName+"_") {
			return true
		}
	}
	return false
}

func filterGlobalToolsBySelectedServices(globalTools map[string]tool.InvokableTool, selectedNames string) map[string]tool.InvokableTool {
	selected := parseSelectedMCPServiceNames(selectedNames)
	if len(selected) == 0 {
		result := make(map[string]tool.InvokableTool, len(globalTools))
		for name, invokable := range globalTools {
			result[name] = invokable
		}
		return result
	}

	result := make(map[string]tool.InvokableTool)
	for toolKey, invokable := range globalTools {
		if isGlobalToolAllowed(toolKey, selected) {
			result[toolKey] = invokable
		}
	}
	return result
}

func GetToolByName(deviceId string, agentId string, toolName string, selectedMCPServiceNames string) (tool.InvokableTool, bool) {
	return GetToolByNameWithTransport(deviceId, agentId, "", toolName, selectedMCPServiceNames)
}

func GetToolByNameWithTransport(deviceId string, agentId string, transportType string, toolName string, selectedMCPServiceNames string) (tool.InvokableTool, bool) {
	//Get it from the local manager first
	localManager := GetLocalMCPManager()
	tool, ok := localManager.GetToolByName(toolName)
	if ok {
		return tool, ok
	}

	//Secondly, get it from the global manager
	selected := parseSelectedMCPServiceNames(selectedMCPServiceNames)
	globalManager := GetGlobalMCPManager()
	if len(selected) == 0 {
		tool, ok = globalManager.GetToolByName(toolName)
		if ok {
			return tool, ok
		}
	} else {
		globalTools := globalManager.GetAllTools()

		//Compatible with scenarios where "server_tool" is directly passed in
		if invokable, exists := globalTools[toolName]; exists && isGlobalToolAllowed(toolName, selected) {
			return invokable, true
		}

		for serviceName := range selected {
			candidate := serviceName + "_" + toolName
			if invokable, exists := globalTools[candidate]; exists {
				return invokable, true
			}
		}
	}

	//Finally, it is obtained from the device MCP client pool, giving priority to the tools reported by the current transport.
	if transportType = strings.TrimSpace(transportType); transportType != "" {
		deviceClient := mcpClientPool.GetMcpClient(deviceId)
		if deviceClient != nil {
			tool, ok = deviceClient.GetIotToolByTransportAndName(transportType, toolName)
			if ok {
				return tool, true
			}
		}
		if agentId != "" && agentId != deviceId {
			return mcpClientPool.GetToolByDeviceId(agentId, toolName)
		}
		return nil, false
	}

	tool, ok = mcpClientPool.GetToolByDeviceId(deviceId, toolName)
	if !ok && agentId != "" && agentId != deviceId {
		tool, ok = mcpClientPool.GetToolByDeviceId(agentId, toolName)
	}
	return tool, ok
}

func GetDeviceMcpClient(deviceId string) *DeviceMcpSession {
	return mcpClientPool.GetMcpClient(deviceId)
}

func GetOrCreateDeviceMcpClient(deviceId string) *DeviceMcpSession {
	return mcpClientPool.GetOrCreateMcpClient(deviceId)
}

func AddDeviceMcpClient(deviceId string, mcpClient *DeviceMcpSession) error {
	mcpClientPool.AddMcpClient(deviceId, mcpClient)
	return nil
}

func RemoveDeviceMcpClient(deviceId string) error {
	mcpClientPool.RemoveMcpClient(deviceId)
	return nil
}

func ShouldScheduleDeviceIotOverMcp(deviceId string, conn ConnInterface) bool {
	if deviceId = strings.TrimSpace(deviceId); deviceId == "" || conn == nil {
		return false
	}
	transportType := strings.TrimSpace(conn.GetMcpTransportType())
	if transportType == "" {
		return false
	}

	session := GetDeviceMcpClient(deviceId)
	if session == nil {
		return true
	}
	return session.ShouldScheduleIotInit(transportType, conn)
}

// EnsureDeviceIotOverMcp Ensures that the device-side IotOverMcp runtime is bound to the transport.
// Reuse existing connections; replace old connections when the transport changes.
func EnsureDeviceIotOverMcp(deviceId string, conn ConnInterface) error {
	if deviceId == "" || conn == nil {
		return fmt.Errorf("deviceId or conn is empty")
	}
	transportType := strings.TrimSpace(conn.GetMcpTransportType())
	if transportType == "" {
		return fmt.Errorf("transportType is empty")
	}

	mcpClientSession := GetOrCreateDeviceMcpClient(deviceId)
	if mcpClientSession == nil {
		return fmt.Errorf("Failed to obtain or create device MCP session")
	}

	transportType = normalizeDeviceTransportType(transportType)

	mcpClientSession.iotMux.Lock()
	existing := mcpClientSession.iotOverMcpByTransport[transportType]
	if existing != nil && existing.conn == conn {
		if existing.IsInitializing() || existing.IsReady() {
			mcpClientSession.iotMux.Unlock()
			return nil
		}
	}

	iotOverMcpClient := NewIotOverMcpClient(deviceId, transportType, conn)
	if iotOverMcpClient == nil {
		mcpClientSession.iotMux.Unlock()
		return fmt.Errorf("Failed to create IotOverMcp client")
	}
	var old *McpClientInstance
	if existing := mcpClientSession.iotOverMcpByTransport[transportType]; existing != nil && existing != iotOverMcpClient {
		old = existing
	}
	mcpClientSession.iotOverMcpByTransport[transportType] = iotOverMcpClient
	iotOverMcpClient.SetOnCloseHandler(mcpClientSession.handleMcpClientClose)
	mcpClientSession.iotMux.Unlock()
	if old != nil {
		old.closeWithReason("iot_transport_replaced")
	}

	if err := iotOverMcpClient.startIotOverMcp(); err != nil {
		iotOverMcpClient.setInitState(mcpClientInitStateIdle)
		CloseDeviceIotOverMcp(deviceId, conn)
		return fmt.Errorf("Failed to initialize IotOverMcp client: %w", err)
	}
	iotOverMcpClient.setInitState(mcpClientInitStateReady)

	return nil
}

func HandleDeviceIotMcpMessage(deviceId string, transportType string, payload []byte) error {
	mcpClientSession := GetDeviceMcpClient(deviceId)
	if mcpClientSession == nil {
		return nil
	}
	transportType = strings.TrimSpace(transportType)
	if transportType == "" {
		return fmt.Errorf("transportType is empty")
	}

	mcpClientSession.iotMux.RLock()
	iotClient := mcpClientSession.iotOverMcpByTransport[normalizeDeviceTransportType(transportType)]
	mcpClientSession.iotMux.RUnlock()
	if iotClient == nil {
		return nil
	}
	if iotClient.iotTransport != nil {
		//The inbound MCP message on the device side has been routed to the current runtime according to device + transportType.
		//Directly inject the current transport to avoid competing with historical runtime for consumption on the shared conn queue.
		iotClient.iotTransport.handleMessage(payload)
		return nil
	}
	if iotClient.conn == nil {
		return nil
	}
	return iotClient.conn.HandleMcpMessage(payload)
}

func CloseDeviceIotOverMcp(deviceId string, conn ConnInterface) {
	mcpClientSession := GetDeviceMcpClient(deviceId)
	if mcpClientSession == nil {
		return
	}
	if conn == nil {
		return
	}

	mcpClientSession.iotMux.Lock()
	transportType := normalizeDeviceTransportType(conn.GetMcpTransportType())
	iotClient := mcpClientSession.iotOverMcpByTransport[transportType]
	if iotClient == nil {
		mcpClientSession.iotMux.Unlock()
		return
	}
	if conn != nil && iotClient.conn != conn {
		mcpClientSession.iotMux.Unlock()
		return
	}
	delete(mcpClientSession.iotOverMcpByTransport, transportType)
	mcpClientSession.iotMux.Unlock()

	iotClient.closeWithReason("device_iot_closed")
}

func GetToolsByDeviceId(deviceId string, agentId string, selectedMCPServiceNames string) (map[string]tool.InvokableTool, error) {
	return GetToolsByDeviceIdWithTransport(deviceId, agentId, "", selectedMCPServiceNames)
}

func GetToolsByDeviceIdWithTransport(deviceId string, agentId string, transportType string, selectedMCPServiceNames string) (map[string]tool.InvokableTool, error) {
	retTools := make(map[string]tool.InvokableTool)

	//Get it from the local manager first
	localManager := GetLocalMCPManager()
	localTools := localManager.GetAllTools()
	for toolName, tool := range localTools {
		retTools[toolName] = tool
	}
	log.Infof("Get %d tools from local manager", len(localTools))

	//Secondly, get it from the global manager
	globalTools := GetGlobalMCPManager().GetAllTools()
	filteredGlobalTools := filterGlobalToolsBySelectedServices(globalTools, selectedMCPServiceNames)
	for toolName, tool := range filteredGlobalTools {
		//Local tools take precedence. If a tool with the same name already exists, it will not be overwritten.
		if _, exists := retTools[toolName]; !exists {
			retTools[toolName] = tool
		}
	}
	log.Infof("Obtained %d tools from the global manager (after filtering)", len(filteredGlobalTools))

	if transportType = strings.TrimSpace(transportType); transportType != "" && deviceId != "" {
		deviceClient := mcpClientPool.GetMcpClient(deviceId)
		if deviceClient != nil {
			for toolName, tool := range deviceClient.GetIotToolsByTransport(transportType) {
				if _, exists := retTools[toolName]; !exists {
					retTools[toolName] = tool
				}
			}
		}
	}

	if transportType == "" {
		deviceTools, err := mcpClientPool.GetAllToolsByDeviceIdAndAgentId(deviceId, agentId)
		if err != nil {
			log.Errorf("Failed to get tool for device %s: %v", deviceId, err)
			return retTools, nil
		}
		for toolName, tool := range deviceTools {
			if _, exists := retTools[toolName]; !exists {
				retTools[toolName] = tool
			}
		}
		log.Infof("Obtained %d tools from device %s", deviceId, len(deviceTools))
	} else if agentId != "" && agentId != deviceId {
		log.Debugf("Start getting ws endpoint MCP tool from agent %s, device=%s, transport=%s", agentId, deviceId, transportType)
		agentTools, err := mcpClientPool.GetWsEndpointMcpTools(agentId)
		if err != nil {
			log.Errorf("Failed to get tool for agent %s: %v", agentId, err)
			return retTools, nil
		}
		log.Debugf("Obtain %d ws endpoint MCP tools from agent %s, device=%s", agentId, len(agentTools), deviceId)
		for toolName, tool := range agentTools {
			if _, exists := retTools[toolName]; !exists {
				retTools[toolName] = tool
			}
		}
	}
	log.Infof("Device %s obtained a total of %d tools", deviceId, len(retTools))

	return retTools, nil
}

func GetWsEndpointMcpTools(agentId string) (map[string]tool.InvokableTool, error) {
	return mcpClientPool.GetWsEndpointMcpTools(agentId)
}

func GetWsEndpointConnectionStatus(agentId string) (bool, int) {
	if strings.TrimSpace(agentId) == "" {
		return false, 0
	}
	client := mcpClientPool.GetMcpClient(agentId)
	if client == nil {
		return false, 0
	}
	return client.GetWsEndpointConnectionStatus()
}

// GetReportedToolsByDeviceID Gets the tools reported by the device through Iot over MCP.
// The console device dimension only returns tools under the websocket / mqtt_udp(udp) transport, and does not mix in other types such as ws endpoint.
func GetReportedToolsByDeviceID(deviceId string) (map[string]tool.InvokableTool, error) {
	retTools := make(map[string]tool.InvokableTool)
	if deviceId == "" {
		return retTools, nil
	}

	client := mcpClientPool.GetMcpClient(deviceId)
	if client == nil {
		return retTools, nil
	}

	transportType, resolved := ResolveCurrentDeviceTransport(deviceId)
	if !resolved || transportType == "" {
		return retTools, nil
	}

	for toolName, invokable := range client.GetIotToolsByTransport(transportType) {
		retTools[toolName] = invokable
	}

	return retTools, nil
}

// RefreshReportedToolsByDeviceID forces a tools/list launch to the current online transport.
// When the refresh fails, an empty list is returned, and the memory tool snapshot of the corresponding runtime is cleared.
func RefreshReportedToolsByDeviceID(deviceId string) (map[string]tool.InvokableTool, error) {
	retTools := make(map[string]tool.InvokableTool)
	if deviceId == "" {
		return retTools, nil
	}

	client := mcpClientPool.GetMcpClient(deviceId)
	if client == nil {
		return retTools, nil
	}

	transportType, resolved := ResolveCurrentDeviceTransport(deviceId)
	if !resolved || transportType == "" {
		return retTools, nil
	}

	return client.RefreshIotToolsByTransport(transportType)
}

// GetReportedToolsByAgentID only gets the MCP tools reported by the agent (WebSocket endpoint)
func GetReportedToolsByAgentID(agentId string) (map[string]tool.InvokableTool, error) {
	retTools := make(map[string]tool.InvokableTool)
	if agentId == "" {
		return retTools, nil
	}

	return mcpClientPool.GetWsEndpointMcpTools(agentId)
}

// RefreshReportedToolsByAgentID forces a tools/list request to the ws endpoint of the agent.
// When the refresh fails, an empty list is returned, and the memory tool snapshot of the corresponding runtime is cleared.
func RefreshReportedToolsByAgentID(agentId string) (map[string]tool.InvokableTool, error) {
	retTools := make(map[string]tool.InvokableTool)
	if agentId == "" {
		return retTools, nil
	}

	client := mcpClientPool.GetMcpClient(agentId)
	if client == nil {
		return retTools, nil
	}

	return client.RefreshWsEndpointTools()
}

// GetReportedToolByDeviceIDAndName is only found in the device reporting tool
func GetReportedToolByDeviceIDAndName(deviceId, toolName string) (tool.InvokableTool, bool) {
	if deviceId == "" {
		return nil, false
	}

	client := mcpClientPool.GetMcpClient(deviceId)
	if client == nil {
		return nil, false
	}

	transportType, resolved := ResolveCurrentDeviceTransport(deviceId)
	if !resolved || transportType == "" {
		return nil, false
	}

	invokable, ok := client.GetIotToolByTransportAndName(transportType, toolName)
	return invokable, ok
}

// GetReportedToolByAgentIDAndName is only found in the agent reporting tool
func GetReportedToolByAgentIDAndName(agentId, toolName string) (tool.InvokableTool, bool) {
	reportedTools, err := GetReportedToolsByAgentID(agentId)
	if err != nil {
		log.Errorf("Failed to obtain the MCP tool reported by the agent: agent=%s err=%v", agentId, err)
		return nil, false
	}

	return findInvokableToolByName(reportedTools, toolName)
}

func RawCallReportedToolByDeviceID(deviceId, toolName string, arguments map[string]interface{}) (string, bool, error) {
	if deviceId == "" {
		return "", false, nil
	}

	client := mcpClientPool.GetMcpClient(deviceId)
	if client == nil {
		return "", false, nil
	}

	transportType, resolved := ResolveCurrentDeviceTransport(deviceId)
	if !resolved || transportType == "" {
		return "", false, nil
	}

	return client.RawCallIotToolByTransport(context.Background(), transportType, toolName, arguments)
}

func RawCallReportedToolByAgentID(agentId, toolName string, arguments map[string]interface{}) (string, bool, error) {
	if agentId == "" {
		return "", false, nil
	}

	client := mcpClientPool.GetMcpClient(agentId)
	if client == nil {
		return "", false, nil
	}

	return client.RawCallWsEndpointTool(context.Background(), toolName, arguments)
}

// GetReportedToolsByDeviceIdAndAgentId compatible method: clearly shunt device/agent query, no longer mixed
func GetReportedToolsByDeviceIdAndAgentId(deviceId string, agentId string) (map[string]tool.InvokableTool, error) {
	if deviceId != "" {
		return GetReportedToolsByDeviceID(deviceId)
	}
	if agentId != "" {
		return GetReportedToolsByAgentID(agentId)
	}
	return make(map[string]tool.InvokableTool), nil
}

// GetReportedToolByName compatible method: split by dimension, no longer mixed
func GetReportedToolByName(deviceId string, agentId string, toolName string) (tool.InvokableTool, bool) {
	if deviceId != "" {
		return GetReportedToolByDeviceIDAndName(deviceId, toolName)
	}
	if agentId != "" {
		return GetReportedToolByAgentIDAndName(agentId, toolName)
	}
	return nil, false
}

func GetAudioResourceByTool(tool McpTool, resourceLink mcp_go.ResourceLink) (mcp_go.ReadResourceResult, error) {
	/*client := tool.GetClient()
	resourceRequest := mcp_go.ReadResourceRequest{
		Request: mcp_go.Request{
			Params: mcp_go.ReadResourceParams{
				URI: resourceLink.URL,
			},
		},
	}
	client.ReadResource(context.Background(), resourceRequest)*/
	return mcp_go.ReadResourceResult{}, nil
}
