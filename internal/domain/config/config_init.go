package user_config

import (
	"context"
	"fmt"
	log "xiaozhi-esp32-server-golang/logger"

	"xiaozhi-esp32-server-golang/internal/domain/config/manager"
	"xiaozhi-esp32-server-golang/internal/domain/config/memory"
	redis_config "xiaozhi-esp32-server-golang/internal/domain/config/redis"

	"github.com/spf13/viper"
)

var (
	//managerSystemConfigHandlers callback list when receiving WebSocket system_config push, the main program can be registered multiple times (such as merged into viper, hot update service)
	managerSystemConfigHandlers []func(map[string]interface{})
)

// RegisterManagerSystemConfigHandler registers the callback for system configuration push in manager mode. It should be called before InitConfigSystem; it can be called multiple times to append multiple callbacks.
func RegisterManagerSystemConfigHandler(fn func(map[string]interface{})) {
	managerSystemConfigHandlers = append(managerSystemConfigHandlers, fn)
}

// InitConfigSystem initialization configuration system
// Call the Init method of the corresponding configuration package according to the value of config_provider.type
func InitConfigSystem(ctx context.Context) error {
	//Get configuration provider type
	providerType := viper.GetString("config_provider.type")
	if providerType == "" {
		providerType = "redis" //Use redis by default
		log.Infof("config_provider.type not set, using default: redis")
	}

	log.Infof("Initializing config system with provider: %s", providerType)

	//Call the corresponding Init method according to the configuration provider type
	switch providerType {
	case "manager":
		manager.SetSystemConfigPushHandler(func(data map[string]interface{}) {
			for _, h := range managerSystemConfigHandlers {
				h(data)
			}
		})
		return manager.Init(ctx)
	case "redis":
		return redis_config.Init(ctx)
	case "memory":
		return memory.Init(ctx)
	default:
		return fmt.Errorf("unsupported config provider type: %s", providerType)
	}
}
