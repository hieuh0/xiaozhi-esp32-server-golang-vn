package mcp

import (
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type McpClientPool struct {
	device2McpClient cmap.ConcurrentMap[string, *DeviceMcpSession]
	createMu         sync.Mutex
}

var mcpClientPool *McpClientPool
var currentDeviceTransportResolver func(deviceID string) string
var currentDeviceTransportResolverMu sync.RWMutex

func init() {
	mcpClientPool = &McpClientPool{
		device2McpClient: cmap.New[*DeviceMcpSession](),
	}
	go mcpClientPool.checkOffline()
}

func RegisterCurrentDeviceTransportResolver(resolver func(deviceID string) string) {
	currentDeviceTransportResolverMu.Lock()
	defer currentDeviceTransportResolverMu.Unlock()
	currentDeviceTransportResolver = resolver
}

func ResolveCurrentDeviceTransport(deviceID string) (string, bool) {
	currentDeviceTransportResolverMu.RLock()
	resolver := currentDeviceTransportResolver
	currentDeviceTransportResolverMu.RUnlock()
	if resolver == nil {
		return "", false
	}
	transportType := strings.TrimSpace(resolver(deviceID))
	if transportType == "" {
		return "", false
	}
	transportType = normalizeDeviceTransportType(transportType)
	if transportType == "unknown" {
		return "", false
	}
	return transportType, true
}

func (p *McpClientPool) GetMcpClient(deviceID string) *DeviceMcpSession {
	client, ok := p.device2McpClient.Get(deviceID)
	if !ok {
		return nil
	}
	return client
}

func (p *McpClientPool) GetOrCreateMcpClient(deviceID string) *DeviceMcpSession {
	if deviceID = strings.TrimSpace(deviceID); deviceID == "" {
		return nil
	}
	if client := p.GetMcpClient(deviceID); client != nil {
		return client
	}

	p.createMu.Lock()
	defer p.createMu.Unlock()

	if client := p.GetMcpClient(deviceID); client != nil {
		return client
	}

	client := NewDeviceMCPSession(deviceID)
	p.device2McpClient.Set(deviceID, client)
	return client
}

func (p *McpClientPool) RemoveMcpClient(deviceID string) {
	p.device2McpClient.Remove(deviceID)
}

func (p *McpClientPool) AddMcpClient(deviceID string, client *DeviceMcpSession) {
	p.device2McpClient.Set(deviceID, client)
}

func (p *McpClientPool) GetToolByDeviceId(deviceId string, toolsName string) (tool.InvokableTool, bool) {
	client := p.GetMcpClient(deviceId)
	if client == nil {
		return nil, false
	}
	return client.GetToolByName(toolsName)
}

func (p *McpClientPool) GetAllToolsByDeviceIdAndAgentId(deviceId string, agentId string) (map[string]tool.InvokableTool, error) {
	retTools := make(map[string]tool.InvokableTool)
	deviceClient := p.GetMcpClient(deviceId)
	if deviceClient != nil {
		deviceTools := deviceClient.GetTools()
		for toolName, tool := range deviceTools {
			retTools[toolName] = tool
		}
	}
	agentClient := p.GetMcpClient(agentId)
	if agentClient != nil {
		agentTools := agentClient.GetTools()
		for toolName, tool := range agentTools {
			retTools[toolName] = tool
		}
	}
	return retTools, nil
}

func (p *McpClientPool) GetWsEndpointMcpTools(agentId string) (map[string]tool.InvokableTool, error) {
	retTools := make(map[string]tool.InvokableTool)
	agentClient := p.GetMcpClient(agentId)
	if agentClient == nil {
		return retTools, nil
	}
	agentTools := agentClient.GetWsEndpointMcpTools()
	for toolName, tool := range agentTools {
		retTools[toolName] = tool
	}
	return retTools, nil
}

func (p *McpClientPool) checkOffline() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		p.sweepOfflineClients()
		<-ticker.C
	}
}

func (p *McpClientPool) sweepOfflineClients() {
	now := time.Now()
	for _, client := range p.device2McpClient.Items() {
		//Check WebSocket endpoint MCP connection
		hasActiveWsConnections := false
		staleWsConnections := make([]*McpClientInstance, 0)
		client.wsEndPointMcp.Range(func(_, value interface{}) bool {
			wsInstance := value.(*McpClientInstance)
			if now.Sub(wsInstance.LastPing()) > mcpConnectionOfflineTimeout {
				staleWsConnections = append(staleWsConnections, wsInstance)
			} else {
				hasActiveWsConnections = true
			}
			return true //continue
		})
		for _, wsInstance := range staleWsConnections {
			wsInstance.closeWithReason("offline_timeout")
		}

		//Check IoT over MCP connection (by transportType)
		hasActiveIotConnection := false
		staleIotConnections := make([]*McpClientInstance, 0)
		client.iotMux.Lock()
		for transportType, iotClient := range client.iotOverMcpByTransport {
			if now.Sub(iotClient.LastPing()) > mcpConnectionOfflineTimeout {
				staleIotConnections = append(staleIotConnections, iotClient)
				delete(client.iotOverMcpByTransport, transportType)
			} else {
				hasActiveIotConnection = true
			}
		}
		client.iotMux.Unlock()
		for _, iotClient := range staleIotConnections {
			iotClient.closeWithReason("offline_timeout")
		}

		//If there are no active connections, remove the client
		if !hasActiveWsConnections && !hasActiveIotConnection {
			p.RemoveMcpClient(client.deviceID)
		}
	}
}
