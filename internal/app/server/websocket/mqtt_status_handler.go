package websocket

import (
	"encoding/json"
	"net/http"
	"strings"

	"xiaozhi-esp32-server-golang/internal/app/mqtt_server"
	"xiaozhi-esp32-server-golang/internal/app/server/mqtt_udp"
	"xiaozhi-esp32-server-golang/internal/util"
)

type mqttInternalStatusResponse struct {
	Broker mqttInternalBrokerStatus `json:"broker"`
	Client mqttInternalClientStatus `json:"client"`
}

type mqttInternalBrokerStatus struct {
	Running          bool `json:"running"`
	ConnectedClients int  `json:"connected_clients"`
}

type mqttInternalClientStatus struct {
	Connected bool   `json:"connected"`
	BrokerURL string `json:"broker_url"`
}

// handleMqttInternalStatus returns runtime MQTT status for the manager backend.
func (s *WebSocketServer) handleMqttInternalStatus(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token != util.GetManagerAuthToken() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	brokerSt := mqtt_server.GetServerStatus()
	adapterSt := mqtt_udp.GetAdapterStatus()

	resp := mqttInternalStatusResponse{
		Broker: mqttInternalBrokerStatus{
			Running:          brokerSt.Running,
			ConnectedClients: brokerSt.ConnectedClients,
		},
		Client: mqttInternalClientStatus{
			Connected: adapterSt.Connected,
			BrokerURL: adapterSt.BrokerURL,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}
