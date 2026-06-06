package speaker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sync"
	"time"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"
)

// StreamingClient WebSocket streaming recognition client
type StreamingClient struct {
	wsURL      string
	conn       *websocket.Conn
	sampleRate int
	mutex      sync.Mutex
	writeMu    sync.Mutex
	peekMu     sync.Mutex
	finishWait chan finishResponse
	peekWaits  map[string]chan peekResponse
	lastPeekAt time.Time
}

type finishResponse struct {
	result *IdentifyResult
	err    error
}

type peekResponse struct {
	result    *IdentifyResult
	throttled bool
	err       error
}

// NewStreamingClient creates a streaming recognition client
func NewStreamingClient(baseURL string) *StreamingClient {
	wsURL := deriveWebSocketURL(baseURL)
	return &StreamingClient{
		wsURL: wsURL,
	}
}

// deriveWebSocketURL derives a WebSocket URL from HTTP base_url
func deriveWebSocketURL(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		log.Errorf("Failed to parse base_url: %v, use default value", err)
		return "ws://localhost:8080/api/v1/speaker/identify_ws"
	}

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	return fmt.Sprintf("%s://%s/api/v1/speaker/identify_ws", scheme, u.Host)
}

// Connect Connect to the WebSocket of the voiceprint recognition service
func (sc *StreamingClient) Connect(sampleRate int, agentId string, threshold float32) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.sampleRate = sampleRate

	//If a connection already exists, use Ping to check if the connection is still valid
	if sc.conn != nil {
		if sc.pingConnectionLocked() {
			//The connection is valid, reuse the existing connection
			return nil
		}
		//The connection has been disconnected. Close the old connection and prepare to reconnect.
		log.Debugf("An old connection has been detected as being disconnected and will be re-established")
		sc.closeConnectionLocked()
	}

	//Construct a WebSocket URL, including sampling rate, agent_id and threshold parameters
	wsURL := fmt.Sprintf("%s?sample_rate=%d", sc.wsURL, sampleRate)
	if agentId != "" {
		wsURL += fmt.Sprintf("&agent_id=%s", url.QueryEscape(agentId))
	}
	//If the threshold is greater than 0, pass the threshold parameter
	if threshold > 0 {
		wsURL += fmt.Sprintf("&threshold=%.6f", threshold)
	}

	//Connect WebSockets
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("WebSocket connection failed: %v", err)
	}

	sc.conn = conn
	sc.finishWait = nil
	sc.peekWaits = make(map[string]chan peekResponse)

	//Set read timeout
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	//Receive connection confirmation message
	var connectionMsg map[string]interface{}
	if err := conn.ReadJSON(&connectionMsg); err != nil {
		conn.Close()
		sc.conn = nil
		return fmt.Errorf("Failed to read connection confirmation message: %v", err)
	}

	if msgType, ok := connectionMsg["type"].(string); !ok || msgType != "connection" {
		conn.Close()
		sc.conn = nil
		return fmt.Errorf("Unexpected connection message: %v", connectionMsg)
	}
	conn.SetReadDeadline(time.Time{})

	log.Debugf("Voiceprint recognition WebSocket connection successful, sampling rate: %d Hz, agent_id: %s, threshold: %.4f", sampleRate, agentId, threshold)
	go sc.readLoop(conn)
	return nil
}

// SendAudioChunk sends audio data chunks
func (sc *StreamingClient) SendAudioChunk(audioData []float32) error {
	conn := sc.getConn()
	if conn == nil {
		return fmt.Errorf("not connected")
	}

	//Convert float32 array to binary bytes
	chunkBytes := float32ToBytes(audioData)

	//Send binary message
	sc.writeMu.Lock()
	err := conn.WriteMessage(websocket.BinaryMessage, chunkBytes)
	sc.writeMu.Unlock()
	if err != nil {
		//Close connection when sending fails
		sc.failConnection(conn, fmt.Errorf("Failed to send audio data: %v", err))
		return fmt.Errorf("Failed to send audio data: %v", err)
	}

	return nil
}

// FinishAndIdentify completes input and obtains identification results
func (sc *StreamingClient) FinishAndIdentify(ctx context.Context) (*IdentifyResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	resultCh := make(chan finishResponse, 1)

	sc.mutex.Lock()
	if sc.conn == nil {
		sc.mutex.Unlock()
		return nil, fmt.Errorf("not connected")
	}
	if sc.finishWait != nil {
		sc.mutex.Unlock()
		return nil, fmt.Errorf("finish already in progress")
	}
	sc.finishWait = resultCh
	conn := sc.conn
	sc.mutex.Unlock()

	//Send completion command
	finishCmd := map[string]interface{}{
		"action": "finish",
	}
	sc.writeMu.Lock()
	err := conn.WriteJSON(finishCmd)
	sc.writeMu.Unlock()
	if err != nil {
		sc.clearFinishWait(resultCh)
		sc.failConnection(conn, fmt.Errorf("Failed to send completion command: %v", err))
		return nil, fmt.Errorf("Failed to send completion command: %v", err)
	}

	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()

	select {
	case resp := <-resultCh:
		return resp.result, resp.err
	case <-ctx.Done():
		sc.clearFinishWait(resultCh)
		return nil, ctx.Err()
	case <-timer.C:
		sc.clearFinishWait(resultCh)
		return nil, fmt.Errorf("Timeout waiting for final recognition result")
	}
}

// PeekAndIdentify gets the intermediate identification results (without ending the current round)
// Returns: recognition result, whether it is stabilized by the server, error
func (sc *StreamingClient) PeekAndIdentify(ctx context.Context, requestID string) (*IdentifyResult, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if requestID == "" {
		requestID = fmt.Sprintf("peek_%d", time.Now().UnixNano())
	}

	sc.peekMu.Lock()
	peekStarted := false
	defer func() {
		if peekStarted {
			sc.lastPeekAt = time.Now()
		}
		sc.peekMu.Unlock()
	}()
	if !sc.lastPeekAt.IsZero() && time.Since(sc.lastPeekAt) < 200*time.Millisecond {
		return nil, true, nil
	}
	peekStarted = true

	respCh := make(chan peekResponse, 1)

	sc.mutex.Lock()
	if sc.conn == nil {
		sc.mutex.Unlock()
		return nil, false, fmt.Errorf("not connected")
	}
	if sc.peekWaits == nil {
		sc.peekWaits = make(map[string]chan peekResponse)
	}
	sc.peekWaits[requestID] = respCh
	conn := sc.conn
	sc.mutex.Unlock()

	peekCmd := map[string]interface{}{
		"action": "peek",
	}
	peekCmd["request_id"] = requestID
	sc.writeMu.Lock()
	err := conn.WriteJSON(peekCmd)
	sc.writeMu.Unlock()
	if err != nil {
		sc.removePeekWait(requestID, respCh)
		sc.failConnection(conn, fmt.Errorf("Failed to send peek command: %v", err))
		return nil, false, fmt.Errorf("Failed to send peek command: %v", err)
	}

	timer := time.NewTimer(1500 * time.Millisecond)
	defer timer.Stop()

	select {
	case resp := <-respCh:
		return resp.result, resp.throttled, resp.err
	case <-ctx.Done():
		sc.removePeekWait(requestID, respCh)
		return nil, false, ctx.Err()
	case <-timer.C:
		sc.removePeekWait(requestID, respCh)
		return nil, false, fmt.Errorf("Timeout waiting for peek results")
	}
}

// Close closes the connection
func (sc *StreamingClient) Close() error {
	sc.mutex.Lock()
	conn := sc.conn
	sc.conn = nil
	finishWait, peekWaits := sc.takePendingLocked()
	sc.mutex.Unlock()

	if conn != nil {
		if err := conn.Close(); err != nil {
			sc.signalPending(finishWait, peekWaits, fmt.Errorf("Connection closed: %v", err))
			return err
		}
	}
	sc.signalPending(finishWait, peekWaits, fmt.Errorf("connection closed"))
	return nil
}

// closeConnectionLocked closes the connection (must be called when mutex is already held)
func (sc *StreamingClient) closeConnectionLocked() error {
	if sc.conn != nil {
		err := sc.conn.Close()
		sc.conn = nil
		return err
	}
	return nil
}

// IsConnected checks whether it is connected
func (sc *StreamingClient) IsConnected() bool {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	return sc.conn != nil
}

// pingConnectionLocked uses Ping to detect whether the connection is valid (must be called when mutex is already held)
func (sc *StreamingClient) pingConnectionLocked() bool {
	if sc.conn == nil {
		return false
	}

	//Use Ping messages to detect connection liveness
	sc.writeMu.Lock()
	sc.conn.SetWriteDeadline(time.Now().Add(1000 * time.Millisecond))
	err := sc.conn.WriteMessage(websocket.PingMessage, nil)
	sc.conn.SetWriteDeadline(time.Time{})
	sc.writeMu.Unlock()

	return err == nil
}

func (sc *StreamingClient) getConn() *websocket.Conn {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	return sc.conn
}

func (sc *StreamingClient) clearFinishWait(waitCh chan finishResponse) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if sc.finishWait == waitCh {
		sc.finishWait = nil
	}
}

func (sc *StreamingClient) removePeekWait(requestID string, waitCh chan peekResponse) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if existing, ok := sc.peekWaits[requestID]; ok && existing == waitCh {
		delete(sc.peekWaits, requestID)
	}
}

func (sc *StreamingClient) takePendingLocked() (chan finishResponse, []chan peekResponse) {
	finishWait := sc.finishWait
	sc.finishWait = nil

	peekWaits := make([]chan peekResponse, 0, len(sc.peekWaits))
	for requestID, waitCh := range sc.peekWaits {
		peekWaits = append(peekWaits, waitCh)
		delete(sc.peekWaits, requestID)
	}
	return finishWait, peekWaits
}

func (sc *StreamingClient) signalPending(finishWait chan finishResponse, peekWaits []chan peekResponse, err error) {
	if finishWait != nil {
		select {
		case finishWait <- finishResponse{err: err}:
		default:
		}
	}
	for _, waitCh := range peekWaits {
		if waitCh == nil {
			continue
		}
		select {
		case waitCh <- peekResponse{err: err}:
		default:
		}
	}
}

func (sc *StreamingClient) failConnection(conn *websocket.Conn, err error) {
	sc.mutex.Lock()
	if sc.conn != conn {
		sc.mutex.Unlock()
		return
	}
	_ = sc.closeConnectionLocked()
	finishWait, peekWaits := sc.takePendingLocked()
	sc.mutex.Unlock()
	sc.signalPending(finishWait, peekWaits, err)
}

func (sc *StreamingClient) readLoop(conn *websocket.Conn) {
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			sc.failConnection(conn, fmt.Errorf("Failed to read message: %v", err))
			return
		}
		if messageType != websocket.TextMessage {
			continue
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Warnf("Failed to parse voiceprint message: %v", err)
			continue
		}

		if !sc.dispatchMessage(msg) {
			sc.failConnection(conn, parseServerError(msg))
			return
		}
	}
}

func (sc *StreamingClient) dispatchMessage(msg map[string]interface{}) bool {
	msgType, _ := msg["type"].(string)
	switch msgType {
	case "partial_result":
		requestID := getString(msg, "request_id")
		throttled := getBool(msg, "throttled")

		sc.mutex.Lock()
		waitCh := sc.peekWaits[requestID]
		if waitCh != nil {
			delete(sc.peekWaits, requestID)
		}
		sc.mutex.Unlock()
		if waitCh == nil {
			return true
		}

		var result *IdentifyResult
		if resultData, ok := msg["result"].(map[string]interface{}); ok && resultData != nil {
			result = identifyResultFromMap(resultData)
		}
		select {
		case waitCh <- peekResponse{result: result, throttled: throttled}:
		default:
		}
		return true
	case "result":
		sc.mutex.Lock()
		waitCh := sc.finishWait
		sc.finishWait = nil
		sc.mutex.Unlock()
		if waitCh == nil {
			return true
		}

		var result *IdentifyResult
		if resultData, ok := msg["result"].(map[string]interface{}); ok && resultData != nil {
			result = identifyResultFromMap(resultData)
		}
		select {
		case waitCh <- finishResponse{result: result}:
		default:
		}
		return true
	case "error":
		return false
	default:
		//Messages such as audio_received/connection/ready/cancelled/closing are only used for status prompts and are ignored here.
		return true
	}
}

func parseServerError(msg map[string]interface{}) error {
	if errMsg, ok := msg["message"].(string); ok && errMsg != "" {
		return fmt.Errorf("Server error: %s", errMsg)
	}
	return fmt.Errorf("Server error: %v", msg)
}

// float32ToBytes Converts a float32 array to binary bytes (little endian)
func float32ToBytes(samples []float32) []byte {
	buf := make([]byte, len(samples)*4)
	for i, sample := range samples {
		bits := math.Float32bits(sample)
		binary.LittleEndian.PutUint32(buf[i*4:], bits)
	}
	return buf
}

// Helper function: safely get values from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getFloat32(m map[string]interface{}, key string) float32 {
	if v, ok := m[key].(float64); ok {
		return float32(v)
	}
	return 0.0
}

func identifyResultFromMap(resultData map[string]interface{}) *IdentifyResult {
	return &IdentifyResult{
		Identified:  getBool(resultData, "identified"),
		SpeakerID:   getString(resultData, "speaker_id"),
		SpeakerName: getString(resultData, "speaker_name"),
		Confidence:  getFloat32(resultData, "confidence"),
		Threshold:   getFloat32(resultData, "threshold"),
	}
}
