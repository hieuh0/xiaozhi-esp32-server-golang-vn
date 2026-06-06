package mcp

import (
	"fmt"
	"sync"

	log "xiaozhi-esp32-server-golang/logger"
)

// MCPManager Unified MCP manager, responsible for coordinating all sub-managers
type MCPManager struct {
	localManager  *LocalMCPManager
	globalManager *GlobalMCPManager
	//deviceManager can manage device manager pools here in the future

	mu      sync.RWMutex
	started bool
}

var (
	mcpManager *MCPManager
	mcpOnce    sync.Once
)

// GetMCPManager Gets the unified MCP manager singleton
func GetMCPManager() *MCPManager {
	mcpOnce.Do(func() {
		mcpManager = &MCPManager{
			localManager:  GetLocalMCPManager(),
			globalManager: GetGlobalMCPManager(),
			started:       false,
		}
	})
	return mcpManager
}

// Start starts all MCP managers
func (m *MCPManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		log.Warn("MCP manager has been started")
		return nil
	}

	log.Info("=== Start MCP Manager Cluster ===")

	//1. First start the local manager
	log.Info("Start the local MCP manager...")
	if err := m.localManager.Start(); err != nil {
		log.Errorf("Failed to start local MCP manager: %v", err)
		return fmt.Errorf("Failed to start local MCP manager: %v", err)
	}

	//2. Then start the global manager
	log.Info("Starting the global MCP manager...")
	if err := m.globalManager.Start(); err != nil {
		log.Errorf("Failed to start global MCP manager: %v", err)
		return fmt.Errorf("Failed to start global MCP manager: %v", err)
	}

	//3. The device manager is dynamically created when connecting, and does not need to be started here.
	log.Info("The device MCP manager will be created dynamically based on the connection")

	m.started = true
	log.Info("=== MCP manager cluster startup completed ===")

	//Output startup status statistics
	m.printStartupStats()

	return nil
}

// Stop Stop all MCP managers
func (m *MCPManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		log.Info("MCP manager is not started, no need to stop")
		return nil
	}

	log.Info("=== Stop MCP Manager Cluster ===")

	//Stop managers in reverse order
	//1. Stop the global manager
	log.Info("Stop global MCP manager...")
	if err := m.globalManager.Stop(); err != nil {
		log.Errorf("Failed to stop global MCP manager: %v", err)
	}

	//2. Stop the local manager
	log.Info("Stop the local MCP manager...")
	if err := m.localManager.Stop(); err != nil {
		log.Errorf("Failed to stop local MCP manager: %v", err)
	}

	//3. Device Manager automatically cleans through disconnection
	log.Info("Device MCP connections will be automatically cleared")

	m.started = false
	log.Info("=== MCP Manager Cluster Stopped ===")
	return nil
}

// IsStarted checks whether the manager has been started
func (m *MCPManager) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

// GetLocalManager Gets the local manager
func (m *MCPManager) GetLocalManager() *LocalMCPManager {
	return m.localManager
}

// GetGlobalManager Gets the global manager
func (m *MCPManager) GetGlobalManager() *GlobalMCPManager {
	return m.globalManager
}

// printStartupStats outputs startup status statistics
func (m *MCPManager) printStartupStats() {
	localToolCount := m.localManager.GetToolCount()
	globalToolCount := len(m.globalManager.GetAllTools())

	log.Infof("MCP manager startup statistics:")
	log.Infof("- Number of local tools: %d", localToolCount)
	log.Infof("- Number of global tools: %d", globalToolCount)
	log.Infof("- Device Manager: Dynamic Management")
	log.Infof("- Total number of tools: %d", localToolCount+globalToolCount)
}

// GetAllManagersStatus Gets the status information of all managers
func (m *MCPManager) GetAllManagersStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := map[string]interface{}{
		"mcp_manager": map[string]interface{}{
			"started": m.started,
		},
		"local_manager": map[string]interface{}{
			"tool_count": m.localManager.GetToolCount(),
			"tool_names": m.localManager.GetToolNames(),
		},
		"global_manager": map[string]interface{}{
			"tool_count": len(m.globalManager.GetAllTools()),
		},
		"device_manager": map[string]interface{}{
			"active_devices": mcpClientPool.device2McpClient.Count(),
		},
	}

	return status
}

// RestartManager restarts the specified manager
func (m *MCPManager) RestartManager(managerType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("MCP manager cluster not started")
	}

	switch managerType {
	case "local":
		log.Info("Restart the local MCP manager...")
		if err := m.localManager.Stop(); err != nil {
			log.Errorf("Failed to stop local manager: %v", err)
		}
		if err := m.localManager.Start(); err != nil {
			return fmt.Errorf("Failed to restart local manager: %v", err)
		}
		log.Info("Local MCP manager restart completed")

	case "global":
		log.Info("Restart global MCP manager...")
		if err := m.globalManager.Stop(); err != nil {
			log.Errorf("Failed to stop global manager: %v", err)
		}
		if err := m.globalManager.Start(); err != nil {
			return fmt.Errorf("Failed to restart global manager: %v", err)
		}
		log.Info("Global MCP Manager restart completed")

	default:
		return fmt.Errorf("Unsupported manager type: %s", managerType)
	}

	return nil
}

//For backward compatibility, convenience functions are provided

// StartMCPManagers starts all MCP managers (convenience function)
func StartMCPManagers() error {
	return GetMCPManager().Start()
}

// StopMCPManagers stops all MCP managers (convenience function)
func StopMCPManagers() error {
	return GetMCPManager().Stop()
}

// GetMCPManagerStatus Gets the MCP manager status (convenience function)
func GetMCPManagerStatus() map[string]interface{} {
	return GetMCPManager().GetAllManagersStatus()
}
