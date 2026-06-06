package types

import "context"

type EventHandler func(ctx context.Context, eventType string, eventData map[string]interface{}) (string, error)

// Upstream push event main program => management internal control
const (
	EventDeviceOnline  = "/api/device/active"   //Device online
	EventDeviceOffline = "/api/device/inactive" //Equipment offline
)

// Downward pull event management internal control => main program
const (
	EventHandleMessageInject = "/api/device/inject_msg" //Handle message injection
)
