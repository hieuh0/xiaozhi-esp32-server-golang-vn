package mqtt_udp

import (
	"fmt"
	"sync"
)

var (
	globalAdapter   *MqttUdpAdapter
	globalAdapterMu sync.RWMutex
)

// AdapterStatus holds connection state of the MQTT client adapter.
type AdapterStatus struct {
	Connected bool   `json:"connected"`
	BrokerURL string `json:"broker_url"`
}

// RegisterGlobalAdapter stores the active adapter for status queries.
func RegisterGlobalAdapter(a *MqttUdpAdapter) {
	globalAdapterMu.Lock()
	defer globalAdapterMu.Unlock()
	globalAdapter = a
}

// GetAdapterStatus returns connection state of the MQTT client adapter.
func GetAdapterStatus() AdapterStatus {
	globalAdapterMu.RLock()
	defer globalAdapterMu.RUnlock()
	if globalAdapter == nil {
		return AdapterStatus{}
	}
	globalAdapter.RLock()
	cfg := globalAdapter.mqttConfig
	client := globalAdapter.client
	globalAdapter.RUnlock()

	connected := client != nil && client.IsConnected()
	brokerURL := ""
	if cfg != nil && cfg.Broker != "" {
		brokerURL = fmt.Sprintf("%s://%s:%d", cfg.Type, cfg.Broker, cfg.Port)
	}
	return AdapterStatus{Connected: connected, BrokerURL: brokerURL}
}
