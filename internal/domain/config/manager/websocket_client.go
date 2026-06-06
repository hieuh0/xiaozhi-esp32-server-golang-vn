package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	einoschema "github.com/cloudwego/eino/schema"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	cmap "github.com/orcaman/concurrent-map/v2"

	"xiaozhi-esp32-server-golang/internal/domain/config/types"
	"xiaozhi-esp32-server-golang/internal/domain/mcp"
	"xiaozhi-esp32-server-golang/internal/domain/openclaw"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

type MessageHandleFunc func(*WebSocketRequest) (string, error)

type WebSocketClient struct {
	conn           *websocket.Conn
	baseURL        string
	requestTimeout time.Duration
	responseChans  map[string]chan *WebSocketResponse
	callbacks      map[string]func(*WebSocketResponse)
	requestHandler func(*WebSocketRequest) //Handle incoming requests
	mu             sync.RWMutex
	writeMu        sync.Mutex //Protect WebSocket write operations from concurrent writes
	isConnected    bool
	connectMu      sync.Mutex
	messageQueue   chan *WebSocketRequest
	workers        sync.WaitGroup

	messageHandle cmap.ConcurrentMap[string, MessageHandleFunc]
	uuid          string

	//Reconnect related fields
	retryStopChan  chan struct{}  //Reconnect coroutine stop signal
	retryWg        sync.WaitGroup //Reconnect coroutine waiting group
	retryMu        sync.Mutex     //Protect reconnection related operations
	isRetrying     bool           //Is reconnecting
	isShuttingDown bool           //Is it shutting down (actively disconnecting, not reconnecting)
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

type managerWSClientClaims struct {
	Purpose string `json:"purpose"`
	UUID    string `json:"uuid"`
	jwt.RegisteredClaims
}

var (
	defaultClient           *WebSocketClient
	clientOnce              sync.Once
	systemConfigPushHandler func(map[string]interface{})
)

// SetSystemConfigPushHandler sets the callback when system_config push is received (the main program is used to merge into viper, etc.), injected by user_config during Init
func SetSystemConfigPushHandler(fn func(map[string]interface{})) {
	systemConfigPushHandler = fn
}

func GetDefaultClient() *WebSocketClient {
	clientOnce.Do(func() {
		defaultClient = NewWebSocketClient()
	})
	return defaultClient
}

func NewWebSocketClient() *WebSocketClient {
	//Get it from the environment variable first, if the environment variable does not exist, get it from the configuration
	baseURL := util.GetBackendURL()
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &WebSocketClient{
		baseURL:        baseURL,
		requestTimeout: 30 * time.Second,
		responseChans:  make(map[string]chan *WebSocketResponse),
		callbacks:      make(map[string]func(*WebSocketResponse)),
		messageQueue:   make(chan *WebSocketRequest, 100),
		messageHandle:  cmap.New[MessageHandleFunc](),
		uuid:           uuid.New().String(),
		retryStopChan:  make(chan struct{}),
		isRetrying:     false,
	}
}

func NewWebSocketClientWithHandler(requestHandler func(*WebSocketRequest)) *WebSocketClient {
	client := NewWebSocketClient()
	client.requestHandler = requestHandler
	return client
}

func (c *WebSocketClient) Connect(ctx context.Context) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	if c.isConnected {
		return nil
	}

	//Convert HTTP URL to WebSocket URL
	wsURL := "ws://" + c.baseURL[7:] + "/ws" //Remove "http://" and add "/ws"
	wsToken, err := c.generateWSToken()
	if err != nil {
		return fmt.Errorf("Failed to generate WebSocket authentication token: %v", err)
	}

	//Establish a WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Origin": []string{c.baseURL},
		"UUID":   []string{c.uuid},
		"Authorization": []string{
			"Bearer " + wsToken,
		},
	})
	if err != nil {
		return fmt.Errorf("WebSocket connection failed: %v", err)
	}

	c.conn = conn
	c.isConnected = true

	//Set ping handler
	conn.SetPongHandler(func(appData string) error {
		log.Debugf("received pong message")
		return nil
	})

	//Start message processing loop
	go c.handleMessages()

	//Start message sending worker thread
	c.startWorkers()

	//Start heartbeat detection
	go c.startHeartbeat()

	log.Debugf("WebSocket client connected to: %s", wsURL)
	return nil
}

func (c *WebSocketClient) generateWSToken() (string, error) {
	claims := managerWSClientClaims{
		Purpose: "manager-ws-client",
		UUID:    c.uuid,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := []byte(util.GetManagerEndpointAuthToken())
	return token.SignedString(secret)
}

func (c *WebSocketClient) Disconnect() error {
	return c.disconnect(false)
}

// disconnect internal disconnect method
// manualDisconnect: true means active disconnection (does not trigger reconnection), false means error disconnection (triggers reconnection)
func (c *WebSocketClient) disconnect(manualDisconnect bool) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	if !c.isConnected {
		return nil
	}

	if manualDisconnect {
		c.isShuttingDown = true
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Debugf("Error closing WebSocket connection: %v", err)
		}
		c.conn = nil
	}

	c.isConnected = false
	c.mu.Lock()
	//Close all response channels
	for _, ch := range c.responseChans {
		close(ch)
	}
	c.responseChans = make(map[string]chan *WebSocketResponse)
	c.callbacks = make(map[string]func(*WebSocketResponse))
	c.mu.Unlock()

	//Stop worker thread
	close(c.messageQueue)
	c.workers.Wait()
	//Re-create the message queue
	c.messageQueue = make(chan *WebSocketRequest, 100)

	log.Debugf("WebSocket connection disconnected")
	return nil
}

func (c *WebSocketClient) IsConnected() bool {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()
	return c.isConnected
}

func (c *WebSocketClient) SendRequest(ctx context.Context, method, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	if !c.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return nil, fmt.Errorf("Connection failed: %v", err)
		}
	}

	//Generate UUID as request ID
	requestID := uuid.New().String()

	request := WebSocketRequest{
		ID:     requestID,
		Method: method,
		Path:   path,
		Body:   body,
	}

	//Create response channel
	responseChan := make(chan *WebSocketResponse, 1)
	c.mu.Lock()
	c.responseChans[requestID] = responseChan
	c.mu.Unlock()

	//Clear response channel
	defer func() {
		c.mu.Lock()
		delete(c.responseChans, requestID)
		c.mu.Unlock()
		close(responseChan)
	}()

	//Send request (protected with write lock)
	c.writeMu.Lock()
	err := c.conn.WriteJSON(request)
	c.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("Failed to send request: %v", err)
	}

	//Waiting for response
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(c.requestTimeout):
		return nil, fmt.Errorf("Request timeout")
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancellation")
	}
}

// Convenience method - native ping using WebSocket
func (c *WebSocketClient) Ping() error {
	if !c.IsConnected() {
		return fmt.Errorf("WebSocket not connected")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
}

func (c *WebSocketClient) GetStatus(ctx context.Context) (*WebSocketResponse, error) {
	return c.SendRequest(ctx, "GET", "/api/ws/status", nil)
}

func (c *WebSocketClient) Echo(ctx context.Context, message string) (*WebSocketResponse, error) {
	return c.SendRequest(ctx, "POST", "/api/ws/echo", map[string]interface{}{
		"message": message,
	})
}

// Global convenience methods
func ConnectManagerWebSocket(ctx context.Context) error {
	return GetDefaultClient().Connect(ctx)
}

func DisconnectManagerWebSocket() error {
	client := GetDefaultClient()
	client.StopReconnect()
	return client.disconnect(true) //Actively disconnect without triggering reconnection
}

func SendManagerRequest(ctx context.Context, method, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	return GetDefaultClient().SendRequest(ctx, method, path, body)
}

func ManagerWebSocketPing(ctx context.Context) error {
	return GetDefaultClient().Ping()
}

func ManagerWebSocketStatus(ctx context.Context) (*WebSocketResponse, error) {
	return GetDefaultClient().GetStatus(ctx)
}

func ManagerWebSocketEcho(ctx context.Context, message string) (*WebSocketResponse, error) {
	return GetDefaultClient().Echo(ctx, message)
}

func IsManagerWebSocketConnected() bool {
	return GetDefaultClient().IsConnected()
}

func SendDeviceRequest(ctx context.Context, path string, body map[string]interface{}) (*WebSocketResponse, error) {
	return GetDefaultClient().SendRequest(ctx, "POST", path, body)
}

// startWorkers starts the message sending worker thread
func (c *WebSocketClient) startWorkers() {
	workerCount := 3 //Start 3 worker threads

	for i := 0; i < workerCount; i++ {
		c.workers.Add(1)
		go func(workerID int) {
			defer c.workers.Done()

			log.Debugf("Manager WebSocket worker thread %d started", workerID)

			for request := range c.messageQueue {
				if !c.IsConnected() {
					log.Debugf("Worker thread %d: WebSocket is not connected, discarding the request", workerID)
					continue
				}

				//Send request (protected with write lock)
				c.writeMu.Lock()
				err := c.conn.WriteJSON(request)
				c.writeMu.Unlock()
				if err != nil {
					log.Debugf("Worker thread %d: Failed to send request: %v", workerID, err)
					//The connection may have been disconnected, triggering reconnection
					c.handleConnectionError()
					continue
				}

				log.Debugf("Worker thread %d: Request %s sent", workerID, request.ID)
			}

			log.Debugf("Manager WebSocket worker thread %d stopped", workerID)
		}(i)
	}
}

// handleConnectionError handles connection errors
func (c *WebSocketClient) handleConnectionError() {
	if c.IsConnected() {
		log.Warn("WebSocket connection error detected, disconnecting...")
		c.disconnect(false) //Error disconnection will trigger reconnection
		//Trigger reconnection
		c.triggerReconnect()
	}
}

// startHeartbeat starts heartbeat detection
func (c *WebSocketClient) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second) //Send a ping every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !c.IsConnected() {
				return
			}

			//Send ping message
			c.writeMu.Lock()
			err := c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
			c.writeMu.Unlock()

			if err != nil {
				log.Warnf("Failed to send ping, connection may be down: %v", err)
				c.disconnect(false) //Error disconnection will trigger reconnection
				//Trigger reconnection
				c.triggerReconnect()
				return
			}
			log.Debugf("Sending ping message successfully")

		case <-c.retryStopChan:
			return
		}
	}
}

// triggerReconnect triggers reconnection (non-blocking)
func (c *WebSocketClient) triggerReconnect() {
	c.retryMu.Lock()
	defer c.retryMu.Unlock()

	//If it is shutting down, reconnection will not be triggered.
	if c.isShuttingDown {
		log.Debug("Closing, does not trigger reconnection")
		return
	}

	//If it is already reconnecting, it will not be triggered again.
	if c.isRetrying {
		return
	}

	c.isRetrying = true
	//Start the reconnection coroutine
	c.retryWg.Add(1)
	go c.startReconnectLoop()
}

// startReconnectLoop starts a reconnect loop (using exponential backoff algorithm)
func (c *WebSocketClient) startReconnectLoop() {
	defer func() {
		c.retryMu.Lock()
		c.isRetrying = false
		c.retryMu.Unlock()
		c.retryWg.Done()
	}()

	//Hardcoded backoff algorithm parameters
	initialDelay := 3 * time.Second //Initial delay 3 seconds
	maxDelay := 1 * time.Minute     //Maximum delay 1 minute
	backoffMultiplier := 2.0        //Backoff multiplier

	delay := initialDelay
	retryCount := 0

	log.Infof("Manager WebSocket connection retry coroutine started")

	for {
		//Check if reconnection should be stopped
		select {
		case <-c.retryStopChan:
			log.Info("Stop reconnection when receiving stop signal")
			return
		default:
		}

		//If closing, stop reconnecting
		c.retryMu.Lock()
		shuttingDown := c.isShuttingDown
		c.retryMu.Unlock()
		if shuttingDown {
			log.Info("Closing, stop reconnecting")
			return
		}

		//If already connected, stop reconnecting
		if c.IsConnected() {
			log.Info("Manager WebSocket connection has been restored, stop reconnecting")
			return
		}

		retryCount++
		log.Warnf("Manager WebSocket connection failed (%d time), wait for %v and then retry the connection...", retryCount, delay)

		//Wait delay time
		select {
		case <-time.After(delay):
			//Continue to reconnect
		case <-c.retryStopChan:
			log.Info("Stop reconnection when receiving stop signal")
			return
		}

		//try to connect
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := c.Connect(ctx)
		cancel()

		if err != nil {
			log.Warnf("Manager WebSocket connection failed (%d time): %v", retryCount, err)
			//Calculate the next delay time (exponential backoff)
			delay = time.Duration(float64(delay) * backoffMultiplier)
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}

		//Connection successful
		log.Info("Manager WebSocket connection successful")
		return
	}
}

// StopReconnect stops the reconnection coroutine
func (c *WebSocketClient) StopReconnect() {
	c.retryMu.Lock()
	c.isShuttingDown = true
	shouldClose := c.retryStopChan != nil
	c.retryMu.Unlock()

	if shouldClose {
		//Use select to avoid closing channels repeatedly
		select {
		case <-c.retryStopChan:
			//Channel has been closed
		default:
			close(c.retryStopChan)
		}
		c.retryWg.Wait()
		log.Info("Manager WebSocket reconnection coroutine has been gracefully closed")
	}
}

// SendRequestWithCallback sends a request and uses a callback to handle the response
func (c *WebSocketClient) SendRequestWithCallback(ctx context.Context, method, path string, body map[string]interface{}, callback func(*WebSocketResponse)) error {
	if !c.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return fmt.Errorf("Connection failed: %v", err)
		}
	}

	//Generate UUID as request ID
	requestID := uuid.New().String()

	request := WebSocketRequest{
		ID:     requestID,
		Method: method,
		Path:   path,
		Body:   body,
	}

	//Register callback
	c.mu.Lock()
	c.callbacks[requestID] = callback
	c.mu.Unlock()

	//Cleanup callback
	defer func() {
		c.mu.Lock()
		delete(c.callbacks, requestID)
		c.mu.Unlock()
	}()

	//Queue the request
	select {
	case c.messageQueue <- &request:
		log.Debugf("Request %s has been queued", requestID)
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("The message queue is full and the request timed out")
	case <-ctx.Done():
		return fmt.Errorf("context cancellation")
	}
}

// SendRequestAsync sends a request asynchronously
func (c *WebSocketClient) SendRequestAsync(ctx context.Context, method, path string, body map[string]interface{}) (string, error) {
	if !c.IsConnected() {
		if err := c.Connect(ctx); err != nil {
			return "", fmt.Errorf("Connection failed: %v", err)
		}
	}

	//Generate UUID as request ID
	requestID := uuid.New().String()

	request := WebSocketRequest{
		ID:     requestID,
		Method: method,
		Path:   path,
		Body:   body,
	}

	//Queue the request
	select {
	case c.messageQueue <- &request:
		log.Debugf("Asynchronous request %s has been queued", requestID)
		return requestID, nil
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("The message queue is full and the request timed out")
	case <-ctx.Done():
		return "", fmt.Errorf("context cancellation")
	}
}

// GetResponse Gets the response of the specified request ID (for asynchronous requests)
func (c *WebSocketClient) GetResponse(requestID string, timeout time.Duration) (*WebSocketResponse, error) {
	responseChan := make(chan *WebSocketResponse, 1)

	//Register a temporary callback
	c.mu.Lock()
	c.callbacks[requestID] = func(response *WebSocketResponse) {
		responseChan <- response
	}
	c.mu.Unlock()

	//Cleanup callback
	defer func() {
		c.mu.Lock()
		delete(c.callbacks, requestID)
		c.mu.Unlock()
		close(responseChan)
	}()

	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("Timeout waiting for response")
	}
}

// handleSystemConfigPush handles system configuration changes pushed by the server and calls registered callbacks asynchronously
func (c *WebSocketClient) handleSystemConfigPush(data map[string]interface{}) {
	if systemConfigPushHandler == nil {
		log.Debugf("System_config push received, but no handling callback registered")
		return
	}
	go systemConfigPushHandler(data)
}

// handleMessages handles received WebSocket messages
func (c *WebSocketClient) handleMessages() {
	for {
		if !c.isConnected {
			return
		}

		//Read message type
		messageType, reader, err := c.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Debugf("WebSocket read error: %v", err)
			}
			c.disconnect(false) //Error disconnection will trigger reconnection
			//Trigger reconnection
			c.triggerReconnect()
			return
		}

		//Handle different types of messages
		switch messageType {
		case websocket.TextMessage:
			//Processing JSON messages
			var rawMessage map[string]interface{}
			if err := json.NewDecoder(reader).Decode(&rawMessage); err != nil {
				log.Errorf("Failed to parse JSON message: %v", err)
				continue
			}

			//Judgment based on message type: server push (system_config), request, response
			if msgType, _ := rawMessage["type"].(string); msgType == "system_config" {
				if data, ok := rawMessage["data"].(map[string]interface{}); ok {
					c.handleSystemConfigPush(data)
				} else {
					log.Warnf("Received system_config push but data format is invalid")
				}
			} else if method, exists := rawMessage["method"]; exists && method != nil {
				//This is the request received
				c.handleIncomingRequest(rawMessage)
			} else if status, exists := rawMessage["status"]; exists && status != nil {
				//This is the response received
				c.handleIncomingResponse(rawMessage)
			} else {
				log.Warnf("Unrecognized WebSocket message received: %+v", rawMessage)
			}

		case websocket.PingMessage:
			//Handle ping messages, automatically reply to pong (protected with write lock)
			log.Debugf("When a ping message is received, it will automatically reply with pong")
			c.writeMu.Lock()
			err := c.conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(10*time.Second))
			c.writeMu.Unlock()
			if err != nil {
				log.Errorf("Failed to send pong: %v", err)
			}

		case websocket.PongMessage:
			//Handling pong messages
			log.Debugf("received pong message")

		case websocket.CloseMessage:
			//Handle closing messages
			log.Debugf("Received shutdown message")
			c.disconnect(false) //Error disconnection will trigger reconnection
			//Trigger reconnection
			c.triggerReconnect()
			return

		default:
			log.Warnf("Received WebSocket message of unknown type: %d", messageType)
		}
	}
}

// handleIncomingRequest handles the received request
func (c *WebSocketClient) handleIncomingRequest(rawMessage map[string]interface{}) {
	var request WebSocketRequest
	if err := mapToStruct(rawMessage, &request); err != nil {
		log.Errorf("Failed to parse WebSocket request: %v", err)
		return
	}

	log.Debugf("Request received: ID=%s, Method=%s, Path=%s", request.ID, request.Method, request.Path)

	//If there is a registered request handler, call it
	if c.requestHandler != nil {
		go c.requestHandler(&request)
	} else {
		//If no handler is registered, the default handler is used for known paths.
		c.handleDefaultRequest(&request)
	}
}

func (c *WebSocketClient) RegisterMessageHandler(ctx context.Context, path string, handler types.EventHandler) {
	f := func(request *WebSocketRequest) (string, error) {
		return handler(ctx, request.Path, request.Body)
	}
	c.messageHandle.Set(path, f)
}

// handleDefaultRequest default request handler
func (c *WebSocketClient) handleDefaultRequest(request *WebSocketRequest) {
	switch request.Path {
	case "/api/config/test":
		//Configuration tests may be time-consuming (VAD/ASR/LLM/TTS serial execution), put in independent goroutine to avoid blocking read loops, and support multi-request concurrency
		go c.handleConfigTestRequest(request)

	case "/api/mcp/tools":
		//Handle MCP tool list requests
		c.handleMcpToolListRequest(request)

	case "/api/mcp/status":
		c.handleMcpStatusRequest(request)

	case "/api/mcp/call":
		//Handle MCP tool call requests
		c.handleMcpToolCallRequest(request)

	case "/api/openclaw/status":
		c.handleOpenClawStatusRequest(request)

	case "/api/openclaw/chat":
		c.handleOpenClawChatRequest(request)

	case "/api/server/info":
		//Return server information
		response := map[string]interface{}{
			"server_name": "xiaozhi-server",
			"version":     "1.0.0",
			"uptime":      time.Now().Format(time.RFC3339),
			"request_id":  request.ID,
		}

		if err := c.SendResponse(request.ID, 200, response, ""); err != nil {
			log.Errorf("Failed to send server information response: %v", err)
		}

	case "/api/server/ping":
		//simple ping response
		response := map[string]interface{}{
			"message": "pong from server",
			"time":    time.Now().Format(time.RFC3339),
		}

		if err := c.SendResponse(request.ID, 200, response, ""); err != nil {
			log.Errorf("Failed to send ping response: %v", err)
		}
	default:
		handler, exists := c.messageHandle.Get(request.Path)
		if exists {
			//Call the processor and handle the return value
			result, err := handler(request)
			if err != nil {
				log.Errorf("Processing request %s failed: %v", request.Path, err)
				//Send error response
				if err := c.SendResponse(request.ID, 500, nil, err.Error()); err != nil {
					log.Errorf("Failed to send error response: %v", err)
				}
			} else {
				//Send successful response
				response := map[string]interface{}{
					"result": result,
				}
				if err := c.SendResponse(request.ID, 200, response, ""); err != nil {
					log.Errorf("Failed to send successful response: %v", err)
				}
			}
		} else {
			log.Warnf("Received unknown WebSocket request path: %s, ID: %s", request.Path, request.ID)

			//Send 404 response
			if err := c.SendResponse(request.ID, 404, nil, "Unknown endpoint"); err != nil {
				log.Errorf("Failed to send error response: %v", err)
			}
		}
	}
}

// configTestTotalTimeout configures the overall test timeout (total of VAD+ASR+LLM+TTS)
const configTestTotalTimeout = 90 * time.Second

// handleConfigTestRequest handles configuration test requests: VAD/ASR/LLM/TTS uses the delivered configuration and fixed WAV/text to perform lightweight testing
func (c *WebSocketClient) handleConfigTestRequest(request *WebSocketRequest) {
	data, _ := request.Body["data"].(map[string]interface{})
	if data == nil {
		log.Debugf("[config_test] Request ID=%s missing data field", request.ID)
		_ = c.SendResponse(request.ID, 400, nil, "Missing data field")
		return
	}
	testText, _ := request.Body["test_text"].(string)
	//debug: Number of configurations of each type in the request (excluding provider)
	log.Debugf("[config_test] Request ID=%s test_text=%q data Number of entries of each type: vad=%d asr=%d llm=%d tts=%d",
		request.ID, testText,
		countConfigKeys(data["vad"]), countConfigKeys(data["asr"]),
		countConfigKeys(data["llm"]), countConfigKeys(data["tts"]))

	type configTestResult struct {
		vad, asr, llm, tts map[string]interface{}
	}
	done := make(chan configTestResult, 1)
	go func() {
		vadR, asrR, llmR, ttsR := RunConfigTest(data, testText)
		done <- configTestResult{vadR, asrR, llmR, ttsR}
	}()

	var vadR, asrR, llmR, ttsR map[string]interface{}
	select {
	case res := <-done:
		vadR, asrR, llmR, ttsR = res.vad, res.asr, res.llm, res.tts
	case <-time.After(configTestTotalTimeout):
		log.Warnf("[config_test] Request ID=%s Overall timeout %v", request.ID, configTestTotalTimeout)
		body := map[string]interface{}{
			"vad": map[string]interface{}{"_error": map[string]interface{}{"ok": false, "message": "Config test overall timeout"}},
			"asr": map[string]interface{}{"_error": map[string]interface{}{"ok": false, "message": "Config test overall timeout"}},
			"llm": map[string]interface{}{"_error": map[string]interface{}{"ok": false, "message": "Config test overall timeout"}},
			"tts": map[string]interface{}{"_error": map[string]interface{}{"ok": false, "message": "Config test overall timeout"}},
		}
		_ = c.SendResponse(request.ID, 200, body, "")
		return
	}

	//When a certain type is included in the request but there is no testable configuration, _none is returned to facilitate the front-end display of the reason.
	fillEmptyConfigTestResult(data, "vad", vadR)
	fillEmptyConfigTestResult(data, "asr", asrR)
	fillEmptyConfigTestResult(data, "llm", llmR)
	fillEmptyConfigTestResult(data, "tts", ttsR)
	body := map[string]interface{}{
		"vad": vadR,
		"asr": asrR,
		"llm": llmR,
		"tts": ttsR,
	}
	log.Debugf("[config_test] Response ID=%s Number of results of each type: vad=%d asr=%d llm=%d tts=%d",
		request.ID, len(vadR), len(asrR), len(llmR), len(ttsR))
	_ = c.SendResponse(request.ID, 200, body, "")
}

// fillEmptyConfigTestResult Writes a _none entry when the request contains this type but the test result is empty
func fillEmptyConfigTestResult(data map[string]interface{}, typ string, result map[string]interface{}) {
	if _, has := data[typ]; !has || len(result) > 0 {
		return
	}
	msg := strings.ToUpper(typ) + " not configured or not enabled"
	result["_none"] = map[string]interface{}{"ok": false, "message": msg}
	log.Debugf("[config_test] Type %s No results, written _none: %s", typ, msg)
}

// countConfigKeys counts the number of config entries in data except provider, used for debugging
func countConfigKeys(v interface{}) int {
	m, ok := v.(map[string]interface{})
	if !ok {
		return 0
	}
	n := 0
	for k := range m {
		if k != "provider" {
			n++
		}
	}
	return n
}

// handleIncomingResponse handles the received response
func (c *WebSocketClient) handleIncomingResponse(rawMessage map[string]interface{}) {
	var response WebSocketResponse
	if err := mapToStruct(rawMessage, &response); err != nil {
		log.Errorf("Failed to parse WebSocket response: %v", err)
		return
	}

	log.Debugf("Response received: ID=%s, Status=%d", response.ID, response.Status)

	//Find the corresponding response channel and callback
	c.mu.RLock()
	responseChan, exists := c.responseChans[response.ID]
	callback, callbackExists := c.callbacks[response.ID]
	c.mu.RUnlock()

	if exists {
		select {
		case responseChan <- &response:
		default:
			log.Debugf("The response channel is full, discarding the response: %s", response.ID)
		}
	}

	if callbackExists {
		go callback(&response)
	}

	if !exists && !callbackExists {
		log.Debugf("Received unknown response ID: %s", response.ID)
	}
}

// SendResponse sends a response to a received request
func (c *WebSocketClient) SendResponse(requestID string, status int, body map[string]interface{}, errorMsg string) error {
	if !c.IsConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	response := WebSocketResponse{
		ID:     requestID,
		Status: status,
		Body:   body,
		Error:  errorMsg,
	}

	//Use write lock protection
	c.writeMu.Lock()
	err := c.conn.WriteJSON(response)
	c.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("Failed to send response: %v", err)
	}

	log.Debugf("Response sent: ID=%s, Status=%d", requestID, status)
	return nil
}

// SetRequestHandler sets the request handler
func (c *WebSocketClient) SetRequestHandler(handler func(*WebSocketRequest)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestHandler = handler
}

// mapToStruct helper function: convert map to struct
func mapToStruct(data map[string]interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}

func toolInfoToSchemaMap(paramsOneOf interface{}) map[string]interface{} {
	if paramsOneOf == nil {
		return nil
	}

	//ParamsOneOf internal fields are not exported, direct json.Marshal may get {}.
	//Prioritize using the official ToOpenAPIV3() to ensure that you can get the real parameter schema.
	if p, ok := paramsOneOf.(*einoschema.ParamsOneOf); ok && p != nil {
		if openAPISchema, err := p.ToOpenAPIV3(); err == nil && openAPISchema != nil {
			raw, err := json.Marshal(openAPISchema)
			if err == nil {
				decoded := map[string]interface{}{}
				if err = json.Unmarshal(raw, &decoded); err == nil {
					if len(decoded) > 0 {
						return decoded
					}
				}
			}
		}
	}

	raw, err := json.Marshal(paramsOneOf)
	if err != nil {
		return nil
	}

	decoded := map[string]interface{}{}
	if err = json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}

	if openAPIV3, ok := decoded["openAPIV3"].(map[string]interface{}); ok {
		return openAPIV3
	}
	if openAPIV3, ok := decoded["open_api_v3"].(map[string]interface{}); ok {
		return openAPIV3
	}
	if len(decoded) == 0 {
		return nil
	}
	return decoded
}

func convertReportedToolsToToolList(reportedTools map[string]tool.InvokableTool) ([]map[string]interface{}, error) {
	toolList := make([]map[string]interface{}, 0)

	names := make([]string, 0, len(reportedTools))
	for name := range reportedTools {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		invokable := reportedTools[name]
		toolInfo := map[string]interface{}{
			"name":        name,
			"description": fmt.Sprintf("MCP tool: %s", name),
			"schema":      true,
		}

		if info, err := invokable.Info(context.Background()); err == nil && info != nil {
			if info.Desc != "" {
				toolInfo["description"] = info.Desc
			}
			inputSchema := toolInfoToSchemaMap(info.ParamsOneOf)
			if inputSchema != nil {
				toolInfo["input_schema"] = inputSchema
			}
		}

		toolList = append(toolList, toolInfo)
	}

	return toolList, nil
}

func getDeviceMcpTools(deviceID string) ([]map[string]interface{}, error) {
	reportedTools, err := mcp.RefreshReportedToolsByDeviceID(deviceID)
	if err != nil {
		log.Errorf("Failed to refresh the MCP tool list reported by the device: %v", err)
		return nil, err
	}

	return convertReportedToolsToToolList(reportedTools)
}

func getAgentMcpTools(agentID string) ([]map[string]interface{}, error) {
	reportedTools, err := mcp.RefreshReportedToolsByAgentID(agentID)
	if err != nil {
		log.Errorf("Failed to refresh the MCP tool list reported by the agent: %v", err)
		return nil, err
	}

	return convertReportedToolsToToolList(reportedTools)
}

// handleMcpToolListRequest handles MCP tool list requests
func (c *WebSocketClient) handleMcpToolListRequest(request *WebSocketRequest) {
	//Get agent_id/device_id from request body
	agentID := ""
	deviceID := ""
	if request.Body != nil {
		if id, ok := request.Body["agent_id"].(string); ok {
			agentID = id
		}
		if id, ok := request.Body["device_id"].(string); ok {
			deviceID = id
		}
	}

	if agentID == "" && deviceID == "" {
		log.Warnf("MCP tool list request received but agent_id/device_id is missing")
		if err := c.SendResponse(request.ID, 400, nil, "Missing agent_id or device_id parameter"); err != nil {
			log.Errorf("Failed to send error response: %v", err)
		}
		return
	}

	log.Infof("Process MCP tool list request, agent_id: %s, device_id: %s", agentID, deviceID)

	if agentID != "" && deviceID != "" {
		if err := c.SendResponse(request.ID, 400, nil, "agent_id and device_id cannot both be provided"); err != nil {
			log.Errorf("Failed to send error response: %v", err)
		}
		return
	}

	var (
		toolList []map[string]interface{}
		err      error
	)
	if deviceID != "" {
		toolList, err = getDeviceMcpTools(deviceID)
	} else {
		toolList, err = getAgentMcpTools(agentID)
	}
	if err != nil {
		log.Errorf("Failed to obtain MCP tool list: %v", err)
		if err := c.SendResponse(request.ID, 500, nil, fmt.Sprintf("Failed to get tool list: %v", err)); err != nil {
			log.Errorf("Failed to send error response: %v", err)
		}
		return
	}

	//construct response
	response := map[string]interface{}{
		"agent_id":  agentID,
		"device_id": deviceID,
		"tools":     toolList,
		"count":     len(toolList),
	}

	//Send response
	if err := c.SendResponse(request.ID, 200, response, ""); err != nil {
		log.Errorf("Failed to send MCP tool list response: %v", err)
	}
}

func (c *WebSocketClient) handleMcpStatusRequest(request *WebSocketRequest) {
	agentID := ""
	if request.Body != nil {
		if id, ok := request.Body["agent_id"].(string); ok {
			agentID = strings.TrimSpace(id)
		}
	}

	if agentID == "" {
		_ = c.SendResponse(request.ID, 400, nil, "Missing agent_id parameter")
		return
	}

	connected, clientCount := mcp.GetWsEndpointConnectionStatus(agentID)
	status := "offline"
	if connected {
		status = "online"
	}

	response := map[string]interface{}{
		"agent_id":     agentID,
		"connected":    connected,
		"status":       status,
		"client_count": clientCount,
	}
	_ = c.SendResponse(request.ID, 200, response, "")
}

// Global convenience methods (async version)
func SendManagerRequestAsync(ctx context.Context, method, path string, body map[string]interface{}) (string, error) {
	return GetDefaultClient().SendRequestAsync(ctx, method, path, body)
}

func SendManagerRequestWithCallback(ctx context.Context, method, path string, body map[string]interface{}, callback func(*WebSocketResponse)) error {
	return GetDefaultClient().SendRequestWithCallback(ctx, method, path, body, callback)
}

func GetManagerResponse(requestID string, timeout time.Duration) (*WebSocketResponse, error) {
	return GetDefaultClient().GetResponse(requestID, timeout)
}

// Two-way communication support method
func SetManagerRequestHandler(handler func(*WebSocketRequest)) {
	GetDefaultClient().SetRequestHandler(handler)
}

func SendManagerResponse(requestID string, status int, body map[string]interface{}, errorMsg string) error {
	return GetDefaultClient().SendResponse(requestID, status, body, errorMsg)
}

// Create a client with a request handler
func NewManagerClientWithHandler(handler func(*WebSocketRequest)) *WebSocketClient {
	return NewWebSocketClientWithHandler(handler)
}

// SendMcpToolListRequest Send MCP tool list request
func SendMcpToolListRequest(ctx context.Context, agentID string) (*WebSocketResponse, error) {
	body := map[string]interface{}{
		"agent_id": agentID,
	}
	return SendManagerRequest(ctx, "GET", "/api/mcp/tools", body)
}

// SendMcpToolListRequestAsync sends MCP tool list request asynchronously
func SendMcpToolListRequestAsync(ctx context.Context, agentID string) (string, error) {
	body := map[string]interface{}{
		"agent_id": agentID,
	}
	return SendManagerRequestAsync(ctx, "GET", "/api/mcp/tools", body)
}

// SendMcpToolListRequestWithCallback uses callback to send MCP tool list request
func SendMcpToolListRequestWithCallback(ctx context.Context, agentID string, callback func(*WebSocketResponse)) error {
	body := map[string]interface{}{
		"agent_id": agentID,
	}
	return SendManagerRequestWithCallback(ctx, "GET", "/api/mcp/tools", body, callback)
}

// Init initializes the Manager configuration provider
// Including the initialization and reconnection mechanism of WebSocket connection
func Init(ctx context.Context) error {
	log.Infof("Initializing Manager config provider with WebSocket client")

	//Create WebSocket client
	client := GetDefaultClient()

	//Try to connect to WebSocket server
	if err := client.Connect(ctx); err != nil {
		log.Warnf("Initial connection to Manager WebSocket failed: %v, the reconnection mechanism will be started.", err)
		//Even if the initial connection fails, start the reconnection mechanism
		client.triggerReconnect()
	} else {
		log.Infof("Manager config provider initialized successfully")
	}

	return nil
}

// Close closes the Manager configuration provider and cleans up resources
func Close() error {
	log.Infof("Closing Manager config provider")

	//Stop the reconnection coroutine
	client := GetDefaultClient()
	client.StopReconnect()

	//Actively disconnect (do not trigger reconnection)
	client.disconnect(true)

	return nil
}

// IsConnected checks whether the Manager configuration provider is connected
func IsConnected() bool {
	return IsManagerWebSocketConnected()
}

// handleMcpToolCallRequest handles the MCP tool call request
func (c *WebSocketClient) handleMcpToolCallRequest(request *WebSocketRequest) {
	agentID := ""
	deviceID := ""
	toolName := ""
	arguments := map[string]interface{}{}
	if request.Body != nil {
		if id, ok := request.Body["agent_id"].(string); ok {
			agentID = id
		}
		if id, ok := request.Body["device_id"].(string); ok {
			deviceID = id
		}
		if t, ok := request.Body["tool_name"].(string); ok {
			toolName = t
		}
		if args, ok := request.Body["arguments"].(map[string]interface{}); ok {
			arguments = args
		}
	}

	if toolName == "" || (agentID == "" && deviceID == "") {
		_ = c.SendResponse(request.ID, 400, nil, "Missing tool_name or agent_id/device_id parameter")
		return
	}

	if agentID != "" && deviceID != "" {
		_ = c.SendResponse(request.ID, 400, nil, "agent_id and device_id cannot both be provided")
		return
	}

	var (
		invokable tool.InvokableTool
		ok        bool
	)
	if deviceID != "" {
		invokable, ok = mcp.GetReportedToolByDeviceIDAndName(deviceID, toolName)
	} else {
		invokable, ok = mcp.GetReportedToolByAgentIDAndName(agentID, toolName)
	}
	if !ok {
		var (
			result    string
			rawCalled bool
			err       error
		)
		if deviceID != "" {
			result, rawCalled, err = mcp.RawCallReportedToolByDeviceID(deviceID, toolName, arguments)
		} else {
			result, rawCalled, err = mcp.RawCallReportedToolByAgentID(agentID, toolName, arguments)
		}
		if rawCalled {
			if err != nil {
				_ = c.SendResponse(request.ID, 500, nil, fmt.Sprintf("Tool call failed (raw call): %v", err))
				return
			}
			log.Warnf("The tool %s does not appear in the tool list and has been passed through raw call: device=%s agent=%s", toolName, deviceID, agentID)
			_ = c.SendResponse(request.ID, 200, map[string]interface{}{
				"agent_id":  agentID,
				"device_id": deviceID,
				"tool_name": toolName,
				"result":    result,
			}, "")
			return
		}
		_ = c.SendResponse(request.ID, 404, nil, fmt.Sprintf("Tool not found: %s", toolName))
		return
	}

	argBytes, _ := json.Marshal(arguments)
	result, err := invokable.InvokableRun(context.Background(), string(argBytes))
	if err != nil {
		_ = c.SendResponse(request.ID, 500, nil, fmt.Sprintf("Tool call failed: %v", err))
		return
	}

	_ = c.SendResponse(request.ID, 200, map[string]interface{}{
		"agent_id":  agentID,
		"device_id": deviceID,
		"tool_name": toolName,
		"result":    result,
	}, "")
}

func (c *WebSocketClient) handleOpenClawStatusRequest(request *WebSocketRequest) {
	agentID := ""
	if request.Body != nil {
		if id, ok := request.Body["agent_id"].(string); ok {
			agentID = strings.TrimSpace(id)
		}
	}
	if agentID == "" {
		_ = c.SendResponse(request.ID, 400, nil, "missing agent_id")
		return
	}

	manager := openclaw.GetManager()
	connected := manager.GetAgentSession(agentID) != nil
	status := "offline"
	if connected {
		status = "online"
	}

	_ = c.SendResponse(request.ID, 200, map[string]interface{}{
		"agent_id":  agentID,
		"connected": connected,
		"status":    status,
	}, "")
}

const (
	defaultOpenClawChatTimeoutMs = 10 * 60 * 1000
	minOpenClawChatTimeoutMs     = 1000
	maxOpenClawChatTimeoutMs     = 10 * 60 * 1000
	openClawChatTestSessionID    = "openclaw-chat-test-global"
)

func buildOpenClawTestDeviceID(agentID string) string {
	trimmed := strings.TrimSpace(agentID)
	if trimmed == "" {
		trimmed = "unknown"
	}
	return "__openclaw_test__:" + trimmed
}

func buildOpenClawTestSessionID() string {
	return openClawChatTestSessionID
}

func parseOpenClawTimeoutMs(v interface{}) int {
	timeout := defaultOpenClawChatTimeoutMs
	switch x := v.(type) {
	case int:
		timeout = x
	case int32:
		timeout = int(x)
	case int64:
		timeout = int(x)
	case float64:
		timeout = int(x)
	case float32:
		timeout = int(x)
	}
	if timeout < minOpenClawChatTimeoutMs {
		timeout = minOpenClawChatTimeoutMs
	}
	if timeout > maxOpenClawChatTimeoutMs {
		timeout = maxOpenClawChatTimeoutMs
	}
	return timeout
}

func parseOpenClawStreamEvents(v interface{}) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "1", "true", "yes", "on":
			return true
		}
	case int:
		return x != 0
	case int32:
		return x != 0
	case int64:
		return x != 0
	case float32:
		return x != 0
	case float64:
		return x != 0
	}
	return false
}

func openClawStreamSnippet(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

func (c *WebSocketClient) handleOpenClawChatRequest(request *WebSocketRequest) {
	agentID := ""
	message := ""
	sessionID := ""
	timeoutMs := defaultOpenClawChatTimeoutMs
	streamEvents := false

	if request.Body != nil {
		if id, ok := request.Body["agent_id"].(string); ok {
			agentID = strings.TrimSpace(id)
		}
		if msg, ok := request.Body["message"].(string); ok {
			message = strings.TrimSpace(msg)
		}
		if rawSessionID, ok := request.Body["session_id"].(string); ok && strings.TrimSpace(rawSessionID) != "" {
			sessionID = strings.TrimSpace(rawSessionID)
		}
		timeoutMs = parseOpenClawTimeoutMs(request.Body["timeout_ms"])
		streamEvents = parseOpenClawStreamEvents(request.Body["stream_events"])
	}

	if agentID == "" {
		_ = c.SendResponse(request.ID, 400, nil, "missing agent_id")
		return
	}
	if message == "" {
		_ = c.SendResponse(request.ID, 400, nil, "missing message")
		return
	}
	if sessionID == "" {
		sessionID = buildOpenClawTestSessionID()
	}

	manager := openclaw.GetManager()
	if manager.GetAgentSession(agentID) == nil {
		_ = c.SendResponse(request.ID, 409, nil, fmt.Sprintf("openclaw session not connected for agent %s", agentID))
		return
	}

	testDeviceID := buildOpenClawTestDeviceID(agentID)
	//Clear the test device history cache to avoid linking to the previous round of test results.
	manager.ReplayOfflineMessages(testDeviceID, func(msg openclaw.OfflineMessage) error {
		return nil
	})

	start := time.Now()
	messageID, err := manager.SendMessage(agentID, testDeviceID, message, sessionID)
	if err != nil {
		errMsg := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.Contains(errMsg, "session not found") {
			_ = c.SendResponse(request.ID, 409, nil, fmt.Sprintf("openclaw session not connected for agent %s", agentID))
			return
		}
		_ = c.SendResponse(request.ID, 500, nil, fmt.Sprintf("openclaw send failed: %v", err))
		return
	}
	if streamEvents {
		log.Infof(
			"openclaw chat stream started: request_id=%s agent=%s message_id=%s session=%s timeout_ms=%d",
			request.ID,
			agentID,
			messageID,
			sessionID,
			timeoutMs,
		)
	}

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	var replyBuilder strings.Builder
	chunks := make([]string, 0, 8)
	done := false
	firstChunkLatencyMs := -1
	for time.Now().Before(deadline) {
		manager.ReplayOfflineMessages(testDeviceID, func(msg openclaw.OfflineMessage) error {
			correlationID := strings.TrimSpace(msg.CorrelationID)
			if correlationID != "" && correlationID != messageID {
				return nil
			}
			chunk := strings.TrimSpace(msg.Text)
			if chunk != "" {
				replyBuilder.WriteString(chunk)
				chunks = append(chunks, chunk)
				if firstChunkLatencyMs < 0 {
					firstChunkLatencyMs = int(time.Since(start).Milliseconds())
				}
				if streamEvents {
					log.Infof(
						"openclaw chat stream chunk received: request_id=%s agent=%s message_id=%s chunk_index=%d chunk_len=%d chunk_snippet=%q",
						request.ID,
						agentID,
						messageID,
						len(chunks),
						len(chunk),
						openClawStreamSnippet(chunk, 64),
					)
				}
				if streamEvents {
					partialBody := map[string]interface{}{
						"agent_id":    agentID,
						"message_id":  messageID,
						"chunk":       chunk,
						"chunk_index": len(chunks),
						"reply":       strings.TrimSpace(replyBuilder.String()),
						"latency_ms":  int(time.Since(start).Milliseconds()),
						"done":        false,
					}
					if firstChunkLatencyMs >= 0 {
						partialBody["first_chunk_latency_ms"] = firstChunkLatencyMs
					}
					if err := c.SendResponse(request.ID, http.StatusPartialContent, partialBody, ""); err != nil {
						log.Warnf("openclaw chat stream partial response send failed: request_id=%s, err=%v", request.ID, err)
					}
				}
			}
			if msg.IsEnd {
				if streamEvents {
					log.Infof(
						"openclaw chat stream end marker received: request_id=%s agent=%s message_id=%s chunk_count=%d partial_reply_len=%d elapsed_ms=%d",
						request.ID,
						agentID,
						messageID,
						len(chunks),
						len(strings.TrimSpace(replyBuilder.String())),
						int(time.Since(start).Milliseconds()),
					)
				}
				done = true
			}
			return nil
		})
		if done {
			break
		}
		time.Sleep(120 * time.Millisecond)
	}
	reply := strings.TrimSpace(replyBuilder.String())

	if !done {
		//Clean the test device offline cache to avoid accumulation.
		manager.ReplayOfflineMessages(testDeviceID, func(msg openclaw.OfflineMessage) error {
			return nil
		})
		if reply == "" {
			if streamEvents {
				log.Warnf(
					"openclaw chat stream timeout without reply: request_id=%s agent=%s message_id=%s timeout_ms=%d",
					request.ID,
					agentID,
					messageID,
					timeoutMs,
				)
			}
			_ = c.SendResponse(request.ID, 504, nil, "openclaw response timeout")
			return
		}
		if streamEvents {
			log.Warnf(
				"openclaw chat stream timeout with partial reply: request_id=%s agent=%s message_id=%s chunk_count=%d reply_len=%d elapsed_ms=%d",
				request.ID,
				agentID,
				messageID,
				len(chunks),
				len(reply),
				int(time.Since(start).Milliseconds()),
			)
		}
		_ = c.SendResponse(request.ID, 504, map[string]interface{}{
			"agent_id":               agentID,
			"message_id":             messageID,
			"reply":                  reply,
			"chunks":                 chunks,
			"chunk_count":            len(chunks),
			"latency_ms":             int(time.Since(start).Milliseconds()),
			"first_chunk_latency_ms": firstChunkLatencyMs,
			"timeout_ms":             timeoutMs,
			"finished":               false,
		}, "openclaw response timeout (partial reply received)")
		return
	}

	latencyMs := int(time.Since(start).Milliseconds())
	if streamEvents {
		log.Infof(
			"openclaw chat stream completed: request_id=%s agent=%s message_id=%s chunk_count=%d reply_len=%d latency_ms=%d",
			request.ID,
			agentID,
			messageID,
			len(chunks),
			len(reply),
			latencyMs,
		)
	}
	var firstChunkLatency interface{}
	if firstChunkLatencyMs >= 0 {
		firstChunkLatency = firstChunkLatencyMs
	}
	_ = c.SendResponse(request.ID, 200, map[string]interface{}{
		"agent_id":               agentID,
		"message_id":             messageID,
		"reply":                  reply,
		"chunks":                 chunks,
		"chunk_count":            len(chunks),
		"latency_ms":             latencyMs,
		"first_chunk_latency_ms": firstChunkLatency,
		"timeout_ms":             timeoutMs,
		"finished":               true,
	}, "")
}
