package user_config

import (
	"fmt"

	"xiaozhi-esp32-server-golang/internal/domain/config/manager"
	userconfig_redis "xiaozhi-esp32-server-golang/internal/domain/config/redis"
	"xiaozhi-esp32-server-golang/internal/util"
)

// Config user configuration provider configuration structure
type Config struct {
	Type       string                 `json:"type"`       //Storage type: "redis", "memory", "file"
	Parameters map[string]interface{} `json:"parameters"` //Storage related configuration parameters
}

func GetProvider(sType string) (UserConfigProvider, error) {
	config := make(map[string]interface{})
	if sType == "manager" {
		//The backend address is first obtained from the environment variable. If the environment variable does not exist, it is obtained from the configuration.
		backendUrl := util.GetBackendURL()
		config = map[string]interface{}{
			"backend_url": backendUrl,
			"auth_token":  util.GetManagerAuthToken(),
		}
	}

	provider, err := GetUserConfigProvider(sType, config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// GetUserConfigProvider creates a user configuration provider
// Create the corresponding provider instance based on the incoming storage type and configuration parameters
// providerType: provider type, supports "redis", "memory", "file"
// config: provider configuration parameters
// Returns the UserConfigProvider interface, supporting complete CRUD operations
func GetUserConfigProvider(providerType string, config map[string]interface{}) (UserConfigProvider, error) {
	if config == nil {
		config = make(map[string]interface{})
	}

	switch providerType {
	case "redis":
		//Create a Redis user configuration provider
		provider, err := userconfig_redis.NewRedisUserConfigProvider(config)
		if err != nil {
			return nil, fmt.Errorf("Failed to create Redis user configuration provider: %v", err)
		}
		return provider, nil
	case "manager":
		//Create a backend management system user configuration provider
		provider, err := manager.NewManagerUserConfigProvider(config)
		if err != nil {
			return nil, fmt.Errorf("Failed to create backend management system user configuration provider: %v", err)
		}
		return provider, nil
	default:
		return nil, fmt.Errorf("Unsupported user configuration provider: %s", providerType)
	}
}
