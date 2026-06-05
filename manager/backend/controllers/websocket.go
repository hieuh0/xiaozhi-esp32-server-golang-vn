package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	cmap "github.com/orcaman/concurrent-map/v2"
	"gorm.io/gorm"

	"xiaozhi/manager/backend/models"
)

type WebSocketController struct {
	DB                *gorm.DB
	endpointAuthToken string
	upgrader          websocket.Upgrader
	clientsMap        cmap.ConcurrentMap[string, *WebSocketClient]
}

type WSClientClaims struct {
	Purpose string `json:"purpose"`
	UUID    string `json:"uuid"`
	jwt.RegisteredClaims
}

// WebSocketClient represents a client connected to the Manager Backend.
type WebSocketClient struct {
	ID           string
	conn         *websocket.Conn
	controller   *WebSocketController
	requestChans map[string]chan *WebSocketResponse
	callbacks    map[string]func(*WebSocketResponse)
	mu           sync.RWMutex
	isConnected  bool
	stopChan     chan struct{} // stop signal channel
}

type WebSocketRequest struct {
	ID      string                 `json:"id"`
	Method  string                 `json:"method"`
	Path    string                 `json:"path"`
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body,omitempty"`
}

type WebSocketResponse struct {
	ID      string                 `json:"id"`
	Status  int                    `json:"status"`
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Schema      bool                   `json:"schema"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}

const (
	defaultBroadcastRequestTimeout = 30 * time.Second
	defaultMcpStatusRequestTimeout = 3 * time.Second
	openClawChatDefaultTimeoutMs   = 10 * 60 * 1000
	openClawChatMinTimeoutMs       = 1000
	openClawChatMaxTimeoutMs       = 10 * 60 * 1000
)

// NewWebSocketController creates a new WebSocket controller.
func NewWebSocketController(db *gorm.DB, endpointAuthToken string) *WebSocketController {
	return &WebSocketController{
		DB:                db,
		endpointAuthToken: strings.TrimSpace(endpointAuthToken),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // allow all origins; restrict in production
			},
		},
		clientsMap: cmap.New[*WebSocketClient](),
	}
}

// HandleWebSocket upgrades the HTTP connection to a WebSocket connection.
func (ctrl *WebSocketController) HandleWebSocket(c *gin.Context) {
	tokenString := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(tokenString), "bearer ") {
		tokenString = strings.TrimSpace(tokenString[7:])
	}
	if tokenString == "" {
		tokenString = strings.TrimSpace(c.Query("token"))
	}
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing websocket auth token"})
		return
	}

	claims, err := ctrl.parseWSClientToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid websocket auth token"})
		return
	}
	if claims.Purpose != "manager-ws-client" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid websocket token purpose"})
		return
	}

	// Read UUID header.
	clientUUID := c.GetHeader("UUID")
	if clientUUID == "" {
		log.Printf("websocket connection missing UUID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing UUID header"})
		return
	}
	if strings.TrimSpace(claims.UUID) != "" && strings.TrimSpace(claims.UUID) != strings.TrimSpace(clientUUID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "UUID does not match token"})
		return
	}

	// Upgrade HTTP connection to WebSocket.
	conn, err := ctrl.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	// Close any existing connection with the same UUID.
	if existingClient, exists := ctrl.clientsMap.Get(clientUUID); exists {
		log.Printf("disconnecting existing connection: %s", clientUUID)
		existingClient.conn.Close()
		existingClient.isConnected = false
	}

	// Create new client.
	client := &WebSocketClient{
		ID:           clientUUID,
		conn:         conn,
		controller:   ctrl,
		requestChans: make(map[string]chan *WebSocketResponse),
		callbacks:    make(map[string]func(*WebSocketResponse)),
		isConnected:  true,
		stopChan:     make(chan struct{}),
	}

	// Store in clientsMap.
	ctrl.clientsMap.Set(clientUUID, client)

	log.Printf("new websocket client connected: %s", clientUUID)

	// Start client message handling.
	go client.handleMessages()

	// Start heartbeat.
	go client.heartbeat()
}

func (ctrl *WebSocketController) parseWSClientToken(tokenString string) (*WSClientClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &WSClientClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(ctrl.endpointAuthToken), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*WSClientClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrInvalidKey
	}
	return claims, nil
}

// removeClient removes a client from the map and signals its heartbeat to stop.
func (ctrl *WebSocketController) removeClient(clientID string) {
	if client, exists := ctrl.clientsMap.Get(clientID); exists {
		// Send stop signal to heartbeat goroutine.
		select {
		case client.stopChan <- struct{}{}:
			log.Printf("stop signal sent to client: %s", clientID)
		default:
			// Channel may be full or already closed; ignore.
		}

		// Ensure client state is correctly set.
		client.isConnected = false
		// Remove from map.
		ctrl.clientsMap.Remove(clientID)
		log.Printf("websocket client disconnected: %s", clientID)
	}
}

// GetClient returns the client with the given UUID.
func (ctrl *WebSocketController) GetClient(uuid string) *WebSocketClient {
	if client, exists := ctrl.clientsMap.Get(uuid); exists {
		return client
	}
	return nil
}

// IsClientConnected reports whether the client with the given UUID is connected.
func (ctrl *WebSocketController) IsClientConnected(uuid string) bool {
	if client, exists := ctrl.clientsMap.Get(uuid); exists {
		return client.isConnected
	}
	return false
}

// GetFirstConnectedClientUUID returns the UUID of the first connected client, useful for config testing.
func (ctrl *WebSocketController) GetFirstConnectedClientUUID() string {
	for item := range ctrl.clientsMap.IterBuffered() {
		if client := item.Val; client.isConnected {
			return client.ID
		}
	}
	return ""
}

// SendToClient sends a message to the client with the given UUID.
func (ctrl *WebSocketController) SendToClient(uuid string, message interface{}) error {
	if client, exists := ctrl.clientsMap.Get(uuid); exists && client.isConnected {
		return client.conn.WriteJSON(message)
	}
	return fmt.Errorf("client %s not connected", uuid)
}

// Broadcast sends a message to all connected clients.
func (ctrl *WebSocketController) Broadcast(message interface{}) {
	for item := range ctrl.clientsMap.IterBuffered() {
		if client := item.Val; client.isConnected {
			if err := client.conn.WriteJSON(message); err != nil {
				log.Printf("failed to broadcast message to client %s: %v", client.ID, err)
			}
		}
	}
}

// BroadcastSystemConfig pushes a system config change to all connected clients
// in the same format as GET /api/system/configs: {"type":"system_config","data":{...}}.
func (ctrl *WebSocketController) BroadcastSystemConfig(data gin.H) {
	ctrl.Broadcast(gin.H{"type": "system_config", "data": data})
}

// handleMessages processes incoming messages from a client.
func (client *WebSocketClient) handleMessages() {
	defer func() {
		client.conn.Close()
		client.isConnected = false
		client.controller.removeClient(client.ID)
	}()

	for {
		if !client.isConnected {
			return
		}

		// Read next message.
		messageType, reader, err := client.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket read error: %v", err)
			}
			return
		}

		// Handle by message type.
		switch messageType {
		case websocket.TextMessage:
			// Handle JSON message.
			var rawMessage map[string]interface{}
			if err := json.NewDecoder(reader).Decode(&rawMessage); err != nil {
				log.Printf("failed to parse JSON message: %v", err)
				continue
			}
			client.handleMessage(rawMessage)

		case websocket.PingMessage:
			// Reply with pong.
			log.Printf("received ping message, sending pong")
			if err := client.conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				log.Printf("failed to send pong: %v", err)
			}

		case websocket.PongMessage:
			log.Printf("received pong message")

		case websocket.CloseMessage:
			log.Printf("received close message")
			return

		default:
			log.Printf("received unknown websocket message type: %d", messageType)
		}
	}
}

// handleMessage dispatches an incoming raw message.
func (client *WebSocketClient) handleMessage(rawMessage map[string]interface{}) {
	// Check if it is a request message.
	if method, exists := rawMessage["method"]; exists && method != nil {
		client.handleRequest(rawMessage)
		return
	}

	// Check if it is a response message.
	if status, exists := rawMessage["status"]; exists && status != nil {
		client.handleResponse(rawMessage)
		return
	}

	log.Printf("received unrecognized message: %+v", rawMessage)
}

// handleRequest processes an incoming request message.
func (client *WebSocketClient) handleRequest(rawMessage map[string]interface{}) {
	var request WebSocketRequest
	if err := mapToStruct(rawMessage, &request); err != nil {
		log.Printf("failed to parse request: %v", err)
		return
	}

	log.Printf("received request: ID=%s, Method=%s, Path=%s", request.ID, request.Method, request.Path)

	// Process and respond.
	client.processRequest(&request)
}

// handleResponse processes an incoming response message.
func (client *WebSocketClient) handleResponse(rawMessage map[string]interface{}) {
	var response WebSocketResponse
	if err := mapToStruct(rawMessage, &response); err != nil {
		log.Printf("failed to parse response: %v", err)
		return
	}

	log.Printf("received response: ID=%s, Status=%d", response.ID, response.Status)

	// Find the matching response channel.
	client.mu.RLock()
	responseChan, exists := client.requestChans[response.ID]
	callback, callbackExists := client.callbacks[response.ID]
	client.mu.RUnlock()

	if exists {
		select {
		case responseChan <- &response:
		default:
			log.Printf("response channel full, dropping response: %s", response.ID)
		}
	}

	if callbackExists {
		go callback(&response)
	}

	if !exists && !callbackExists {
		log.Printf("received response for unknown ID: %s", response.ID)
	}
}

// processRequest routes a request to the appropriate handler.
func (client *WebSocketClient) processRequest(request *WebSocketRequest) {
	switch request.Path {
	case "/api/server/info":
		client.handleServerInfoRequest(request)

	case "/api/server/ping":
		client.handlePingRequest(request)

	case "/api/device/active":
		client.handleDeviceActiveRequest(request)

	case "/api/device/inactive":
		client.handleDeviceInactiveRequest(request)

	default:
		log.Printf("unknown request path: %s", request.Path)
		client.sendResponse(request.ID, 404, nil, "Unknown endpoint")
	}
}

// handleServerInfoRequest handles a server info request.
func (client *WebSocketClient) handleServerInfoRequest(request *WebSocketRequest) {
	response := map[string]interface{}{
		"server_name": "xiaozhi-manager-backend",
		"version":     "1.0.0",
		"uptime":      time.Now().Format(time.RFC3339),
		"request_id":  request.ID,
		"client_id":   client.ID,
	}

	client.sendResponse(request.ID, 200, response, "")
}

// handlePingRequest handles a ping request.
func (client *WebSocketClient) handlePingRequest(request *WebSocketRequest) {
	response := map[string]interface{}{
		"message":   "pong from manager backend",
		"time":      time.Now().Format(time.RFC3339),
		"client_id": client.ID,
	}

	client.sendResponse(request.ID, 200, response, "")
}

// handleDeviceActiveRequest handles a device last-active-time update request.
func (client *WebSocketClient) handleDeviceActiveRequest(request *WebSocketRequest) {
	// Extract device_id from request body.
	deviceID := ""
	if request.Body != nil {
		if id, ok := request.Body["device_id"].(string); ok {
			deviceID = id
		}
	}

	if deviceID == "" {
		log.Printf("received device active request but device_id is missing")
		client.sendResponse(request.ID, 400, nil, "missing device_id parameter")
		return
	}

	log.Printf("handling device active time update request, device_id: %s", deviceID)

	// Update device last active time.
	now := time.Now()
	result := client.controller.DB.Model(&models.Device{}).
		Where("device_name = ?", deviceID).
		Update("last_active_at", now)

	if result.Error != nil {
		log.Printf("failed to update device active time: %v", result.Error)
		client.sendResponse(request.ID, 500, nil, fmt.Sprintf("failed to update device active time: %v", result.Error))
		return
	}

	if result.RowsAffected == 0 {
		log.Printf("device not found: %s", deviceID)
		client.sendResponse(request.ID, 404, nil, "device not found")
		return
	}

	// Build success response.
	response := map[string]interface{}{
		"device_id":      deviceID,
		"last_active_at": now.Format(time.RFC3339),
		"message":        "device active time updated successfully",
	}

	client.sendResponse(request.ID, 200, response, "")
	log.Printf("device %s active time updated to: %s", deviceID, now.Format(time.RFC3339))
}

// handleDeviceInactiveRequest handles a device offline request.
func (client *WebSocketClient) handleDeviceInactiveRequest(request *WebSocketRequest) {
	// Extract device_id from request body.
	deviceID := ""
	if request.Body != nil {
		if id, ok := request.Body["device_id"].(string); ok {
			deviceID = id
		}
	}

	if deviceID == "" {
		log.Printf("received device inactive request but device_id is missing")
		client.sendResponse(request.ID, 400, nil, "missing device_id parameter")
		return
	}

	log.Printf("handling device inactive request, device_id: %s", deviceID)

	// Set device last active time to NULL (offline state).
	result := client.controller.DB.Model(&models.Device{}).
		Where("device_name = ?", deviceID).
		Update("last_active_at", nil)

	if result.Error != nil {
		log.Printf("failed to update device offline status: %v", result.Error)
		client.sendResponse(request.ID, 500, nil, fmt.Sprintf("failed to update device offline status: %v", result.Error))
		return
	}

	if result.RowsAffected == 0 {
		log.Printf("device not found: %s", deviceID)
		client.sendResponse(request.ID, 404, nil, "device not found")
		return
	}

	// Build success response.
	response := map[string]interface{}{
		"device_id":      deviceID,
		"last_active_at": nil, // offline state
		"message":        "device offline status updated successfully",
	}

	client.sendResponse(request.ID, 200, response, "")
	log.Printf("device %s set to offline", deviceID)
}

// sendResponse sends a WebSocket response to the client.
func (client *WebSocketClient) sendResponse(requestID string, status int, body map[string]interface{}, errorMsg string) {
	response := WebSocketResponse{
		ID:     requestID,
		Status: status,
		Body:   body,
		Error:  errorMsg,
	}

	if err := client.conn.WriteJSON(response); err != nil {
		log.Printf("failed to send response: %v", err)
	} else {
		log.Printf("response sent: ID=%s, Status=%d", requestID, status)
	}
}

// heartbeat sends periodic WebSocket native pings to keep the connection alive.
func (client *WebSocketClient) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Consecutive ping failure counter.
	pingFailCount := 0
	maxPingFailCount := 3 // allow up to 3 consecutive failures

	for {
		select {
		case <-client.stopChan:
			log.Printf("stop signal received, stopping heartbeat")
			return
		case <-ticker.C:
			if !client.isConnected {
				return
			}

			// Check whether connection is still valid.
			if client.conn == nil {
				log.Printf("websocket connection is nil, stopping heartbeat")
				return
			}

			// Send WebSocket native ping.
			if err := client.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				pingFailCount++
				log.Printf("ping failed (attempt %d): %v", pingFailCount, err)

				// Only disconnect after exceeding the threshold.
				if pingFailCount >= maxPingFailCount {
					log.Printf("closing websocket connection after %d consecutive ping failures", maxPingFailCount)
					client.conn.Close()
					return
				}
			} else {
				// Ping succeeded; reset failure counter.
				if pingFailCount > 0 {
					log.Printf("ping recovered, resetting failure counter")
					pingFailCount = 0
				}
			}
		}
	}
}

// SendRequest sends a request to the client (fire-and-forget).
func (client *WebSocketClient) SendRequest(method, path string, body map[string]interface{}) error {
	request := WebSocketRequest{
		ID:     uuid.New().String(),
		Method: method,
		Path:   path,
		Body:   body,
	}

	return client.conn.WriteJSON(request)
}

// SendRequestWithResponse sends a request and waits for the response.
func (client *WebSocketClient) SendRequestWithResponse(ctx context.Context, method, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	requestID := uuid.New().String()

	request := WebSocketRequest{
		ID:     requestID,
		Method: method,
		Path:   path,
		Body:   body,
	}

	// Create response channel.
	responseChan := make(chan *WebSocketResponse, 1)
	client.mu.Lock()
	client.requestChans[requestID] = responseChan
	client.mu.Unlock()

	// Clean up response channel on return.
	defer func() {
		client.mu.Lock()
		delete(client.requestChans, requestID)
		client.mu.Unlock()
		close(responseChan)
	}()

	// Send request.
	if err := client.conn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// Wait for response.
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timed out")
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled")
	}
}

// mapToStruct converts a map to a struct via JSON round-trip.
func mapToStruct(data map[string]interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}

// SendRequestToClient sends a request to the specified client UUID and waits for the response.
func (ctrl *WebSocketController) SendRequestToClient(ctx context.Context, uuid string, method, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	if client, exists := ctrl.clientsMap.Get(uuid); exists && client.isConnected {
		return client.SendRequestWithResponse(ctx, method, path, body)
	}
	return nil, fmt.Errorf("client %s not connected", uuid)
}

// RequestMcpToolsFromClient requests the MCP tool list from the client (broadcast, waits for first non-empty response).
func (ctrl *WebSocketController) RequestMcpToolsFromClient(ctx context.Context, agentID string) ([]string, error) {
	toolDetails, err := ctrl.RequestMcpToolDetailsFromClient(ctx, agentID)
	if err != nil {
		return nil, err
	}

	toolNames := make([]string, 0, len(toolDetails))
	for _, detail := range toolDetails {
		toolNames = append(toolNames, detail.Name)
	}

	return toolNames, nil
}

func (ctrl *WebSocketController) RequestMcpToolDetailsFromClient(ctx context.Context, agentID string) ([]MCPTool, error) {
	log.Printf("requesting MCP tool list from client, agentID: %s", agentID)
	return ctrl.requestMcpToolsByBody(ctx, map[string]interface{}{"agent_id": agentID})
}

func (ctrl *WebSocketController) RequestMcpEndpointStatusFromClient(ctx context.Context, agentID string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"agent_id": agentID,
	}

	return ctrl.broadcastMcpStatusRequest(ctx, body)
}

// RequestDeviceMcpToolsFromClient requests the MCP tool list at device scope (broadcast, waits for first non-empty response).
func (ctrl *WebSocketController) RequestDeviceMcpToolsFromClient(ctx context.Context, deviceID string) ([]string, error) {
	toolDetails, err := ctrl.RequestDeviceMcpToolDetailsFromClient(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	toolNames := make([]string, 0, len(toolDetails))
	for _, detail := range toolDetails {
		toolNames = append(toolNames, detail.Name)
	}

	return toolNames, nil
}

func (ctrl *WebSocketController) RequestDeviceMcpToolDetailsFromClient(ctx context.Context, deviceID string) ([]MCPTool, error) {
	log.Printf("requesting device MCP tool list, deviceID: %s", deviceID)
	return ctrl.requestMcpToolsByBody(ctx, map[string]interface{}{"device_id": deviceID})
}

func (ctrl *WebSocketController) requestMcpToolsByBody(ctx context.Context, body map[string]interface{}) ([]MCPTool, error) {
	response, err := ctrl.broadcastRequestAndWaitFirstSuccess(ctx, "GET", "/api/mcp/tools", body)
	if err != nil {
		return nil, err
	}

	toolsData, ok := response.Body["tools"]
	if !ok {
		return []MCPTool{}, nil
	}

	tools := make([]MCPTool, 0)
	switch v := toolsData.(type) {
	case []interface{}:
		for _, item := range v {
			if toolStr, ok := item.(string); ok {
				tools = append(tools, MCPTool{Name: toolStr, Description: fmt.Sprintf("MCP tool: %s", toolStr), Schema: true})
				continue
			}

			toolMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := toolMap["name"].(string)
			if name == "" {
				continue
			}

			description, _ := toolMap["description"].(string)
			if description == "" {
				description = fmt.Sprintf("MCP tool: %s", name)
			}

			parsed := MCPTool{Name: name, Description: description, Schema: true}
			if inputSchema, ok := toolMap["input_schema"].(map[string]interface{}); ok {
				parsed.InputSchema = inputSchema
			} else if inputSchema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
				// compatibility: some clients return camelCase field names
				parsed.InputSchema = inputSchema
			}
			tools = append(tools, parsed)
		}
	case []string:
		for _, name := range v {
			tools = append(tools, MCPTool{Name: name, Description: fmt.Sprintf("MCP tool: %s", name), Schema: true})
		}
	}

	return tools, nil
}

// CallMcpToolFromClient asks the client to execute an MCP tool call.
func (ctrl *WebSocketController) CallMcpToolFromClient(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	response, err := ctrl.broadcastRequestAndWaitFirstSuccess(ctx, "POST", "/api/mcp/call", body)
	if err != nil {
		return nil, err
	}

	if response.Body == nil {
		return map[string]interface{}{}, nil
	}

	return response.Body, nil
}

// RequestOpenClawStatusFromClient asks the client for the OpenClaw connection status.
func (ctrl *WebSocketController) RequestOpenClawStatusFromClient(ctx context.Context, agentID string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"agent_id": agentID,
	}

	response, err := ctrl.broadcastRequestAndWaitFirstSuccess(ctx, "GET", "/api/openclaw/status", body)
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return map[string]interface{}{}, nil
	}

	return response.Body, nil
}

// CallOpenClawChatFromClient asks the client to perform an OpenClaw chat test.
func (ctrl *WebSocketController) CallOpenClawChatFromClient(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	if body == nil {
		body = map[string]interface{}{}
	}
	timeoutMs := normalizeOpenClawChatTimeoutMs(body["timeout_ms"])
	body["timeout_ms"] = timeoutMs
	waitTimeout := time.Duration(timeoutMs)*time.Millisecond + 5*time.Second

	response, err := ctrl.broadcastRequestAndWaitFirstSuccessWithTimeout(ctx, "POST", "/api/openclaw/chat", body, waitTimeout)
	if err != nil {
		return nil, err
	}
	if response.Body == nil {
		return map[string]interface{}{}, nil
	}

	return response.Body, nil
}

type wsClientResponse struct {
	clientID string
	response *WebSocketResponse
}

// CallOpenClawChatStreamFromClient asks the client to perform an OpenClaw chat test with streaming callbacks.
func (ctrl *WebSocketController) CallOpenClawChatStreamFromClient(
	ctx context.Context,
	body map[string]interface{},
	onResponse func(*WebSocketResponse) error,
) (map[string]interface{}, error) {
	if body == nil {
		body = map[string]interface{}{}
	}
	timeoutMs := normalizeOpenClawChatTimeoutMs(body["timeout_ms"])
	body["timeout_ms"] = timeoutMs
	body["stream_events"] = true
	waitTimeout := time.Duration(timeoutMs)*time.Millisecond + 5*time.Second

	responseChan := make(chan wsClientResponse, 64)
	requestID := uuid.New().String()
	callbacksRegistered := 0

	for item := range ctrl.clientsMap.IterBuffered() {
		client := item.Val
		if !client.isConnected {
			continue
		}

		clientID := client.ID
		responseHandler := func(response *WebSocketResponse) {
			select {
			case responseChan <- wsClientResponse{clientID: clientID, response: response}:
			default:
				log.Printf("OpenClaw streaming response channel full, dropping response: %s", requestID)
			}
		}

		client.mu.Lock()
		client.callbacks[requestID] = responseHandler
		client.mu.Unlock()
		callbacksRegistered++

		request := WebSocketRequest{
			ID:     requestID,
			Method: "POST",
			Path:   "/api/openclaw/chat",
			Body:   body,
		}
		if err := client.conn.WriteJSON(request); err != nil {
			log.Printf("failed to send OpenClaw streaming request to client %s: %v", client.ID, err)
		}
	}

	if callbacksRegistered == 0 {
		return nil, fmt.Errorf("no connected clients")
	}

	defer func() {
		for item := range ctrl.clientsMap.IterBuffered() {
			client := item.Val
			client.mu.Lock()
			delete(client.callbacks, requestID)
			client.mu.Unlock()
		}
	}()

	selectedClientID := ""
	failedClients := map[string]bool{}
	firstError := ""
	timeout := time.After(waitTimeout)

	for {
		select {
		case event := <-responseChan:
			resp := event.response
			if resp == nil {
				continue
			}

			if selectedClientID == "" {
				if resp.Status >= http.StatusBadRequest {
					failedClients[event.clientID] = true
					if firstError == "" {
						msg := strings.TrimSpace(resp.Error)
						if msg != "" {
							firstError = msg
						}
					}
					if len(failedClients) >= callbacksRegistered {
						if firstError != "" {
							return nil, fmt.Errorf("%s", firstError)
						}
						return nil, fmt.Errorf("all clients returned failure")
					}
					continue
				}
				selectedClientID = event.clientID
			}

			if event.clientID != selectedClientID {
				continue
			}

			if onResponse != nil {
				if err := onResponse(resp); err != nil {
					return nil, err
				}
			}

			if resp.Status == http.StatusOK {
				if resp.Body == nil {
					return map[string]interface{}{}, nil
				}
				return resp.Body, nil
			}

			if resp.Status >= http.StatusBadRequest {
				msg := strings.TrimSpace(resp.Error)
				if msg == "" {
					msg = fmt.Sprintf("OpenClaw streaming request failed: status=%d", resp.Status)
				}
				return nil, fmt.Errorf("%s", msg)
			}
		case <-timeout:
			return nil, fmt.Errorf("request timed out")
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		}
	}
}

func (ctrl *WebSocketController) broadcastRequestAndWaitFirstSuccess(ctx context.Context, method, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	return ctrl.broadcastRequestAndWaitFirstSuccessWithTimeout(ctx, method, path, body, defaultBroadcastRequestTimeout)
}

func isMcpStatusOnline(body map[string]interface{}) bool {
	if body == nil {
		return false
	}
	if connected, ok := body["connected"].(bool); ok && connected {
		return true
	}
	status, _ := body["status"].(string)
	return strings.EqualFold(strings.TrimSpace(status), "online")
}

func mcpStatusClientCount(body map[string]interface{}) int {
	if body == nil {
		return 0
	}
	switch v := body["client_count"].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func (ctrl *WebSocketController) broadcastMcpStatusRequest(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	requestID := uuid.New().String()

	clients := make([]*WebSocketClient, 0)
	for item := range ctrl.clientsMap.IterBuffered() {
		client := item.Val
		if client != nil && client.isConnected {
			clients = append(clients, client)
		}
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("no connected clients")
	}

	responseChan := make(chan *WebSocketResponse, len(clients))
	responseHandler := func(response *WebSocketResponse) {
		select {
		case responseChan <- response:
		default:
			log.Printf("MCP status response channel full, dropping response: %s", response.ID)
		}
	}

	for _, client := range clients {
		client.mu.Lock()
		client.callbacks[requestID] = responseHandler
		client.mu.Unlock()
	}
	defer func() {
		for _, client := range clients {
			client.mu.Lock()
			delete(client.callbacks, requestID)
			client.mu.Unlock()
		}
	}()

	sentCount := 0
	for _, client := range clients {
		request := WebSocketRequest{ID: requestID, Method: "GET", Path: "/api/mcp/status", Body: body}
		if err := client.conn.WriteJSON(request); err != nil {
			log.Printf("failed to send MCP status request to client %s: %v", client.ID, err)
			continue
		}
		sentCount++
	}
	if sentCount == 0 {
		return nil, fmt.Errorf("no available clients")
	}

	offline := map[string]interface{}{
		"connected":    false,
		"status":       "offline",
		"client_count": 0,
	}
	responsesReceived := 0
	successResponses := 0
	firstError := ""
	timeout := time.After(defaultMcpStatusRequestTimeout)
	for {
		select {
		case response := <-responseChan:
			responsesReceived++
			if response != nil && response.Status == http.StatusOK {
				successResponses++
				if isMcpStatusOnline(response.Body) {
					return response.Body, nil
				}
				offline["client_count"] = mcpStatusClientCount(offline) + mcpStatusClientCount(response.Body)
			} else if response != nil && firstError == "" {
				firstError = strings.TrimSpace(response.Error)
			}

			if responsesReceived >= sentCount {
				if successResponses > 0 {
					return offline, nil
				}
				if firstError != "" {
					return nil, fmt.Errorf("%s", firstError)
				}
				return nil, fmt.Errorf("all clients returned failure")
			}
		case <-timeout:
			if successResponses > 0 {
				return offline, nil
			}
			if firstError != "" {
				return nil, fmt.Errorf("%s", firstError)
			}
			return nil, fmt.Errorf("request timed out")
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		}
	}
}

func normalizeOpenClawChatTimeoutMs(v interface{}) int {
	timeout := openClawChatDefaultTimeoutMs
	switch x := v.(type) {
	case int:
		timeout = x
	case int32:
		timeout = int(x)
	case int64:
		timeout = int(x)
	case float32:
		timeout = int(x)
	case float64:
		timeout = int(x)
	}

	if timeout < openClawChatMinTimeoutMs {
		timeout = openClawChatMinTimeoutMs
	}
	if timeout > openClawChatMaxTimeoutMs {
		timeout = openClawChatMaxTimeoutMs
	}
	return timeout
}

func (ctrl *WebSocketController) broadcastRequestAndWaitFirstSuccessWithTimeout(
	ctx context.Context,
	method, path string,
	body map[string]interface{},
	waitTimeout time.Duration,
) (*WebSocketResponse, error) {
	if waitTimeout <= 0 {
		waitTimeout = defaultBroadcastRequestTimeout
	}

	responseChan := make(chan *WebSocketResponse, 10)
	requestID := uuid.New().String()

	responseHandler := func(response *WebSocketResponse) {
		select {
		case responseChan <- response:
		default:
			log.Printf("response channel full, dropping response: %s", response.ID)
		}
	}

	callbacksRegistered := 0
	for item := range ctrl.clientsMap.IterBuffered() {
		client := item.Val
		if !client.isConnected {
			continue
		}

		client.mu.Lock()
		client.callbacks[requestID] = responseHandler
		client.mu.Unlock()
		callbacksRegistered++

		request := WebSocketRequest{ID: requestID, Method: method, Path: path, Body: body}
		if err := client.conn.WriteJSON(request); err != nil {
			log.Printf("failed to send request to client %s: %v", client.ID, err)
		}
	}

	if callbacksRegistered == 0 {
		return nil, fmt.Errorf("no connected clients")
	}

	defer func() {
		for item := range ctrl.clientsMap.IterBuffered() {
			client := item.Val
			client.mu.Lock()
			delete(client.callbacks, requestID)
			client.mu.Unlock()
		}
	}()

	responsesReceived := 0
	firstError := ""
	timeout := time.After(waitTimeout)
	for {
		select {
		case response := <-responseChan:
			responsesReceived++
			if response != nil && response.Status == http.StatusOK {
				return response, nil
			}
			if response != nil && firstError == "" {
				msg := strings.TrimSpace(response.Error)
				if msg != "" {
					firstError = msg
				}
			}
			if responsesReceived >= callbacksRegistered {
				if firstError != "" {
					return nil, fmt.Errorf("%s", firstError)
				}
				return nil, fmt.Errorf("all clients returned failure")
			}
		case <-timeout:
			return nil, fmt.Errorf("request timed out")
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		}
	}
}

// RequestServerInfoFromClient requests server info from the specified client.
func (ctrl *WebSocketController) RequestServerInfoFromClient(ctx context.Context, uuid string) (*WebSocketResponse, error) {
	return ctrl.SendRequestToClient(ctx, uuid, "GET", "/api/server/info", nil)
}

func (ctrl *WebSocketController) RequestDeviceActivation(ctx context.Context, uuid, deviceID string) (*WebSocketResponse, error) {
	return ctrl.SendRequestToClient(ctx, uuid, "GET", "/api/device/activation", map[string]interface{}{
		"device_id": deviceID,
	})
}

// RequestPingFromClient sends a ping request to the specified client.
func (ctrl *WebSocketController) RequestPingFromClient(ctx context.Context, uuid string) (*WebSocketResponse, error) {
	return ctrl.SendRequestToClient(ctx, uuid, "GET", "/api/server/ping", nil)
}

// InjectMessageToDevice injects a message into a device (broadcast).
func (ctrl *WebSocketController) InjectMessageToDevice(ctx context.Context, deviceID, message string, skipLlm bool, autoListen bool) error {
	body := map[string]interface{}{
		"device_id":   deviceID,
		"message":     message,
		"skip_llm":    skipLlm,
		"auto_listen": autoListen,
	}

	// Build request.
	request := WebSocketRequest{
		ID:     uuid.New().String(),
		Method: "POST",
		Path:   "/api/device/inject_msg",
		Body:   body,
	}

	// Broadcast to all connected clients.
	var lastError error
	clientCount := 0

	for item := range ctrl.clientsMap.IterBuffered() {
		client := item.Val
		if client.isConnected {
			clientCount++
			if err := client.conn.WriteJSON(request); err != nil {
				log.Printf("failed to broadcast inject message to client %s: %v", client.ID, err)
				lastError = err
			} else {
				log.Printf("inject message broadcast to client %s succeeded", client.ID)
			}
		}
	}

	if clientCount == 0 {
		return fmt.Errorf("no connected clients")
	}

	return lastError
}

// SendRequestToClientAsync sends a request to the specified client without waiting for a response.
func (ctrl *WebSocketController) SendRequestToClientAsync(uuid string, method, path string, body map[string]interface{}) error {
	if client, exists := ctrl.clientsMap.Get(uuid); exists && client.isConnected {
		return client.SendRequest(method, path, body)
	}
	return fmt.Errorf("client %s not connected", uuid)
}

// GetClientConnectionStatus returns the connection status of all clients.
func (ctrl *WebSocketController) GetClientConnectionStatus() map[string]interface{} {
	clients := make([]map[string]interface{}, 0)
	for item := range ctrl.clientsMap.IterBuffered() {
		client := item.Val
		clients = append(clients, map[string]interface{}{
			"uuid":      client.ID,
			"connected": client.isConnected,
		})
	}

	return map[string]interface{}{
		"clients": clients,
		"count":   len(clients),
	}
}

// GetClientStatus returns the connection status of a specific client.
func (ctrl *WebSocketController) GetClientStatus(uuid string) map[string]interface{} {
	if client, exists := ctrl.clientsMap.Get(uuid); exists {
		return map[string]interface{}{
			"uuid":      client.ID,
			"connected": client.isConnected,
			"message":   "client connected",
		}
	}

	return map[string]interface{}{
		"uuid":      uuid,
		"connected": false,
		"message":   "client not connected",
	}
}
