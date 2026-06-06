package memory

import (
	"context"
	"fmt"
	"sync"

	"xiaozhi-esp32-server-golang/internal/domain/config/types"
	log "xiaozhi-esp32-server-golang/logger"
)

// MemoryUserConfigProvider memory user configuration provider
// Implement the UserConfigProvider interface and store the configuration in memory
// Note: Data will be lost after restarting, suitable for testing or temporary storage scenarios
type MemoryUserConfigProvider struct {
	mu         sync.RWMutex
	configs    map[string]types.UConfig
	maxEntries int
}

// MemoryConfig memory configuration structure
type MemoryConfig struct {
	MaxEntries int `json:"max_entries"` //Maximum number of storage entries
}

// NewMemoryUserConfigProvider creates a memory user configuration provider
// config: Configuration parameter map, including max_entries, etc.
func NewMemoryUserConfigProvider(config map[string]interface{}) (*MemoryUserConfigProvider, error) {
	//Parse configuration parameters
	memoryConfig := &MemoryConfig{
		MaxEntries: 1000, //Default maximum 1000 configurations
	}

	if maxEntries, ok := config["max_entries"].(int); ok && maxEntries > 0 {
		memoryConfig.MaxEntries = maxEntries
	} else if maxEntriesFloat, ok := config["max_entries"].(float64); ok && maxEntriesFloat > 0 {
		memoryConfig.MaxEntries = int(maxEntriesFloat)
	}

	provider := &MemoryUserConfigProvider{
		configs:    make(map[string]types.UConfig),
		maxEntries: memoryConfig.MaxEntries,
	}

	log.Log().Infof("Memory user configuration provider initialization successful, maximum number of entries: %d", memoryConfig.MaxEntries)
	return provider, nil
}

// GetUserConfig gets user configuration
func (m *MemoryUserConfigProvider) GetUserConfig(ctx context.Context, userID string) (types.UConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.configs[userID]
	if !exists {
		log.Log().Debugf("User %s configuration does not exist, return empty configuration", userID)
		return types.UConfig{}, nil
	}

	return config, nil
}

// SetUserConfig sets user configuration
func (m *MemoryUserConfigProvider) SetUserConfig(ctx context.Context, userID string, config types.UConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	//Check if the maximum number of entries is exceeded
	if len(m.configs) >= m.maxEntries && !m.configExists(userID) {
		return fmt.Errorf("Maximum number of storage entries %d reached, new configuration cannot be added", m.maxEntries)
	}

	m.configs[userID] = config
	log.Log().Infof("User %s configuration set successfully (memory storage)", userID)
	return nil
}

// DeleteUserConfig Delete user configuration
func (m *MemoryUserConfigProvider) DeleteUserConfig(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.configs[userID]; !exists {
		log.Log().Warnf("User %s configuration does not exist and does not need to be deleted.", userID)
		return nil
	}

	delete(m.configs, userID)
	log.Log().Infof("User %s configuration deleted successfully (memory storage)", userID)
	return nil
}

// Close closes the provider (no special cleanup is required for the memory provider)
func (m *MemoryUserConfigProvider) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	//Clear all configuration
	m.configs = make(map[string]types.UConfig)
	log.Log().Info("The memory user configuration provider has been closed and all configurations have been cleared")
	return nil
}

// configExists checks whether the configuration exists (internal method, needs to hold a lock when calling)
func (m *MemoryUserConfigProvider) configExists(userID string) bool {
	_, exists := m.configs[userID]
	return exists
}

// GetStats Gets storage statistics (additional utility method)
func (m *MemoryUserConfigProvider) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_configs": len(m.configs),
		"max_entries":   m.maxEntries,
		"usage_percent": float64(len(m.configs)) / float64(m.maxEntries) * 100,
	}
}

// ListUserIDs List all user IDs (extra utility method)
func (m *MemoryUserConfigProvider) ListUserIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userIDs := make([]string, 0, len(m.configs))
	for userID := range m.configs {
		userIDs = append(userIDs, userID)
	}
	return userIDs
}

// GetSystemConfig Gets system configuration
func (m *MemoryUserConfigProvider) GetSystemConfig(ctx context.Context) (string, error) {
	//Memory configuration provider does not provide system configuration
	return "", nil
}

// Init initializes the Memory configuration provider
func Init(ctx context.Context) error {
	log.Log().Info("Memory config provider initialized successfully")
	return nil
}

// Close Closes the Memory configuration provider and cleans up resources
func Close() error {
	log.Log().Info("Memory config provider closed")
	return nil
}

// IsConnected checks whether the Memory configuration provider is connected
func IsConnected() bool {
	//The memory configuration provider is always in "connected" state
	return true
}
