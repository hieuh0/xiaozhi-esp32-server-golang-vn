package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"xiaozhi-esp32-server-golang/internal/app/server/auth"
	"xiaozhi-esp32-server-golang/internal/app/server/types"
	"xiaozhi-esp32-server-golang/internal/domain/mcp"
	"xiaozhi-esp32-server-golang/internal/domain/openclaw"
	log "xiaozhi-esp32-server-golang/logger"
)

// WebSocketServer represents the WebSocket server
type WebSocketServer struct {
	// WebSocket upgrader
	upgrader websocket.Upgrader
	// client states, concurrency-safe via sync.Map
	clientStates sync.Map
	// authentication manager
	authManager *auth.AuthManager
	// port
	port int
	// MCP manager
	globalMCPManager *mcp.GlobalMCPManager

	onNewConnection    types.OnNewConnection
	onOpenClawResponse func(event openclaw.ResponseDelivery) bool
	onInjectMessage    func(deviceID, message string, skipLlm bool, autoListen bool) error
}

// WebSocketServerOption is a functional option for configuring WebSocketServer
type WebSocketServerOption func(*WebSocketServer)

// WithAuthManager sets the authentication manager
func WithAuthManager(authManager *auth.AuthManager) WebSocketServerOption {
	return func(s *WebSocketServer) {
		s.authManager = authManager
	}
}

// WithMCPManager sets the MCP manager
func WithMCPManager(mcpManager *mcp.GlobalMCPManager) WebSocketServerOption {
	return func(s *WebSocketServer) {
		s.globalMCPManager = mcpManager
	}
}

func WithOnNewConnection(onNewConnection types.OnNewConnection) WebSocketServerOption {
	return func(s *WebSocketServer) {
		s.onNewConnection = onNewConnection
	}
}

func WithOnOpenClawResponse(handler func(event openclaw.ResponseDelivery) bool) WebSocketServerOption {
	return func(s *WebSocketServer) {
		s.onOpenClawResponse = handler
	}
}

func WithOnInjectMessage(handler func(deviceID, message string, skipLlm bool, autoListen bool) error) WebSocketServerOption {
	return func(s *WebSocketServer) {
		s.onInjectMessage = handler
	}
}

// NewWebSocketServer creates a new WebSocket server using functional options
func NewWebSocketServer(port int, opts ...WebSocketServerOption) *WebSocketServer {
	s := &WebSocketServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // allow connections from all origins
			},
		},
		// defaults
		authManager:      auth.A(),
		port:             port,
		globalMCPManager: mcp.GetGlobalMCPManager(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start starts the WebSocket server
func (s *WebSocketServer) Start() error {
	// start all MCP managers via unified manager
	if err := mcp.StartMCPManagers(); err != nil {
		log.Errorf("Failed to start MCP manager cluster: %v", err)
		return err
	}

	// start session cleanup
	go s.cleanupSessions()

	// register route handlers
	http.HandleFunc("/xiaozhi/mqtt_udp/v1/", s.handleMqttUdpChat)
	http.HandleFunc("/xiaozhi/v1/", s.handleChat)
	http.HandleFunc("/xiaozhi/ota/", s.handleOta)
	http.HandleFunc("/xiaozhi/ota/activate", s.handleOtaActivate)
	http.HandleFunc("/mcp", s.handleMCPWebSocket)
	http.HandleFunc("/ws/openclaw", s.handleOpenClawWebSocket)
	http.HandleFunc("/xiaozhi/api/mcp/tools/", s.handleMCPAPI)
	http.HandleFunc("/xiaozhi/api/vision", s.handleVisionAPI) // image recognition API

	http.HandleFunc("/admin/inject_msg", s.handleInjectMsg)

	listenAddr := fmt.Sprintf("0.0.0.0:%d", s.port)
	log.Infof("WebSocket server started at ws://%s/xiaozhi/v1/", listenAddr)
	log.Infof("MCP WebSocket endpoint: ws://%s/mcp?token=xxx", listenAddr)
	log.Infof("OpenClaw WebSocket endpoint: ws://%s/ws/openclaw?token=xxx", listenAddr)
	log.Infof("MCP API endpoint: http://%s/xiaozhi/api/mcp/tools/{deviceId}", listenAddr)

	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Log().Fatalf("WebSocket server failed to start: %v", err)
		return err
	}
	return nil
}

// handleGetDeviceTools retrieves the tool list for a device
func (s *WebSocketServer) handleGetDeviceTools(w http.ResponseWriter, r *http.Request, deviceID string) {

}

// cleanupSessions periodically removes expired sessions
func (s *WebSocketServer) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.authManager.CleanupSessions(30 * time.Minute)
	}
}

// handleChat handles WebSocket connections
func (s *WebSocketServer) handleChat(w http.ResponseWriter, r *http.Request) {
	s.internalHandleChat(w, r, false)
}

// handleMqttUdpChat handles MQTT-UDP bridge WebSocket connections
func (s *WebSocketServer) handleMqttUdpChat(w http.ResponseWriter, r *http.Request) {
	s.internalHandleChat(w, r, true)
}

// internalHandleChat handles WebSocket connections
func (s *WebSocketServer) internalHandleChat(w http.ResponseWriter, r *http.Request, isMqttUdp bool) {
	deviceID, clientID := extractDeviceAndClientID(r)
	if deviceID == "" {
		log.Warn("Missing device-id, please pass via Header or URL parameter")
		http.Error(w, "Missing device-id (supported via Header or URL parameter)", http.StatusBadRequest)
		return
	}
	if clientID == "" {
		log.Debugf("Connection did not provide client-id, device_id: %s", deviceID)
	}

	/*isAuth := viper.GetBool("auth.enable")
	if isAuth {
		token := r.Header.Get("Authorization")
		if token == "" {
			log.Warn("Missing Authorization header")
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// validate token
		if !s.authManager.ValidateToken(token) {
			log.Warnf("Invalid token: %s", token)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	}*/

	// upgrade HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	// adapt to IConn interface
	wsConn := NewWebSocketConn(conn, deviceID, isMqttUdp)
	if s.onNewConnection != nil {
		s.onNewConnection(wsConn)
	}

}

func extractDeviceAndClientID(r *http.Request) (string, string) {
	deviceKeys := []string{"Device-Id", "device-id", "DEVICE-ID", "device_id", "Device_Id", "deviceId"}
	clientKeys := []string{"Client-Id", "client-id", "CLIENT-ID", "client_id", "Client_Id", "clientId"}

	headerDeviceID, headerDeviceKey := findHeaderValue(r.Header, deviceKeys)
	queryDeviceID, queryDeviceKey := findQueryValue(r.URL.Query(), deviceKeys)
	headerClientID, headerClientKey := findHeaderValue(r.Header, clientKeys)
	queryClientID, queryClientKey := findQueryValue(r.URL.Query(), clientKeys)

	deviceID := headerDeviceID
	if deviceID == "" {
		deviceID = queryDeviceID
	} else if queryDeviceID != "" && queryDeviceID != headerDeviceID {
		log.Warnf("device-id mismatch: Header(%s) vs URL param(%s), using Header value", headerDeviceKey, queryDeviceKey)
	}

	clientID := headerClientID
	if clientID == "" {
		clientID = queryClientID
	} else if queryClientID != "" && queryClientID != headerClientID {
		log.Warnf("client-id mismatch: Header(%s) vs URL param(%s), using Header value", headerClientKey, queryClientKey)
	}

	return deviceID, clientID
}

func findHeaderValue(header http.Header, keys []string) (string, string) {
	for _, key := range keys {
		if value := header.Get(key); value != "" {
			return value, key
		}
	}
	return "", ""
}

func findQueryValue(values url.Values, keys []string) (string, string) {
	for _, key := range keys {
		if value := values.Get(key); value != "" {
			return value, key
		}
	}
	return "", ""
}

func (s *WebSocketServer) handleInjectMsg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.onInjectMessage == nil {
		http.Error(w, "inject message handler unavailable", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		DeviceID   string `json:"device_id"`
		Message    string `json:"message"`
		SkipLlm    bool   `json:"skip_llm"`
		AutoListen *bool  `json:"auto_listen"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	autoListen := true
	if req.AutoListen != nil {
		autoListen = *req.AutoListen
	}
	if err := s.onInjectMessage(req.DeviceID, req.Message, req.SkipLlm, autoListen); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"device_id":   req.DeviceID,
		"message":     req.Message,
		"skip_llm":    req.SkipLlm,
		"auto_listen": autoListen,
	})
}
