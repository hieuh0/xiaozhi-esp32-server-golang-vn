package user_config

import (
	"context"
	"xiaozhi-esp32-server-golang/internal/domain/config/types"
)

// UserConfigProvider User configuration provider interface
// This is an extended interface that supports more operations and is different from the original UserConfig interface.
type UserConfigProvider interface {
	//auth
	//Get activation information based on deviceId and clientId
	IsDeviceActivated(ctx context.Context, deviceId string, clientId string) (bool, error)
	GetActivationInfo(ctx context.Context, deviceId string, clientId string) (string, string, string, int)
	VerifyChallenge(ctx context.Context, deviceId string, clientId string, activationPayload types.ActivationPayload) (bool, error)

	//llm memory

	//GetUserConfig gets user configuration (compatible with original interface)
	GetUserConfig(ctx context.Context, userID string) (types.UConfig, error)

	//SwitchDeviceRoleByName switches the device role by role name (supports fuzzy matching)
	SwitchDeviceRoleByName(ctx context.Context, deviceID string, roleName string) (string, error)

	//RestoreDeviceDefaultRole Restores the device's default role (clears device-bound roles)
	RestoreDeviceDefaultRole(ctx context.Context, deviceID string) error

	//Get mqtt, mqtt_server, udp, ota, vision configuration
	GetSystemConfig(ctx context.Context) (string, error)

	//Register uplink event processing function (such as device online and offline, etc.)
	NotifyDeviceEvent(ctx context.Context, eventType string, eventData map[string]interface{})
	//Register downstream event handling functions (such as message injection, etc.)
	RegisterMessageEventHandler(ctx context.Context, eventType string, eventHandler types.EventHandler)
}
