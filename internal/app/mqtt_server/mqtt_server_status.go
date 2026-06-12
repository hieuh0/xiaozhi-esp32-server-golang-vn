package mqtt_server

// ServerStatus holds broker running state and connected client count.
type ServerStatus struct {
	Running          bool `json:"running"`
	ConnectedClients int  `json:"connected_clients"`
}

// GetServerStatus returns broker running state and client count.
func GetServerStatus() ServerStatus {
	serverMu.Lock()
	defer serverMu.Unlock()
	if currentServer == nil {
		return ServerStatus{}
	}
	clients := currentServer.Clients.GetAll()
	return ServerStatus{Running: true, ConnectedClients: len(clients)}
}
