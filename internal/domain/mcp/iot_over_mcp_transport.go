package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	log "xiaozhi-esp32-server-golang/logger"
)

/**
// Interface for the transport layer.
type Interface interface {
	// Start the connection. Start should only be called once.
	Start(ctx context.Context) error

	// SendRequest sends a json RPC request and returns the response synchronously.
	SendRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error)

	// SendNotification sends a json RPC Notification to the server.
	SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error

	// SetNotificationHandler sets the handler for notifications.
	// Any notification before the handler is set will be discarded.
	SetNotificationHandler(handler func(notification mcp.JSONRPCNotification))

	// Close the connection.
	Close() error
}
*/

type ConnInterface interface {
	SendMcpMsg(payload []byte) error
	RecvMcpMsg(ctx context.Context, timeOut int) ([]byte, error)
	HandleMcpMessage(payload []byte) error
	GetMcpTransportType() string
}

type IotOverMcpTransport struct {
	conn ConnInterface

	notifyHandler   func(notification mcp.JSONRPCNotification)
	activityHandler func()
	//Add close callback
	onCloseHandler func(reason string)

	respChans    map[string]*pendingResponse
	respChansMux sync.RWMutex
	readDone     chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
	closed       bool
	closedMux    sync.RWMutex
	writeMux     sync.Mutex

	requestTimeout time.Duration
	closeTimeout   time.Duration
}

func (t *IotOverMcpTransport) Send(ctx context.Context, msg []byte) error {
	return t.conn.SendMcpMsg(msg)
}

func NewIotOverMcpTransport(conn ConnInterface) (*IotOverMcpTransport, error) {
	ctx, cancel := context.WithCancel(context.Background())
	transportInstance := &IotOverMcpTransport{
		conn:           conn,
		respChans:      make(map[string]*pendingResponse),
		readDone:       make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
		requestTimeout: DefaultRequestTimeout,
		closeTimeout:   DefaultCloseTimeout,
	}
	go transportInstance.readMessages()
	return transportInstance, nil
}

// Implement the Interface interface
func (t *IotOverMcpTransport) Start(ctx context.Context) error {
	//TODO: Start connection/monitor messages, etc.

	return nil
}

func (t *IotOverMcpTransport) popPending(id string) *pendingResponse {
	t.respChansMux.Lock()
	defer t.respChansMux.Unlock()

	pending := t.respChans[id]
	if pending != nil {
		delete(t.respChans, id)
	}
	return pending
}

func (t *IotOverMcpTransport) failAllPending(err error) {
	t.respChansMux.Lock()
	pending := make([]*pendingResponse, 0, len(t.respChans))
	for id, pendingResp := range t.respChans {
		pending = append(pending, pendingResp)
		delete(t.respChans, id)
	}
	t.respChansMux.Unlock()

	for _, pendingResp := range pending {
		pendingResp.resolve(nil, err)
	}
}

func (t *IotOverMcpTransport) readMessages() {
	defer close(t.readDone)

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			message, err := t.conn.RecvMcpMsg(t.ctx, 1000)
			if err != nil {
				if t.ctx.Err() != nil {
					return
				}
				if isTransportTimeoutErr(err) {
					continue
				}

				t.closedMux.Lock()
				t.closed = true
				t.closedMux.Unlock()
				t.failAllPending(fmt.Errorf("connection is closed"))

				if t.onCloseHandler != nil {
					t.onCloseHandler("connection_closed")
				}
				return
			}

			t.handleMessage(message)
		}
	}
}

func (t *IotOverMcpTransport) handleMessage(message []byte) {
	if t.activityHandler != nil {
		t.activityHandler()
	}

	method, hasID, err := classifyJSONRPCMessage(message)
	if err != nil {
		log.Warnf("Received unrecognized IoT MCP message: %s", string(message))
		return
	}

	if method != "" {
		if hasID {
			log.Warnf("Received unsupported IoT JSON-RPC request: %s", method)
			return
		}

		var notification mcp.JSONRPCNotification
		if err := json.Unmarshal(message, &notification); err != nil {
			log.Warnf("Received malformed IoT JSON-RPC notification: %s", string(message))
			return
		}
		if t.notifyHandler != nil {
			t.notifyHandler(notification)
		}
		return
	}

	if hasID {
		var response transport.JSONRPCResponse
		if err := json.Unmarshal(message, &response); err != nil {
			log.Warnf("Received malformed IoT JSON-RPC response: %s", string(message))
			return
		}

		pending := t.popPending(requestIDKey(response.ID))
		if pending == nil {
			log.Warnf("No IoT response channel found for ID: %s", response.ID.String())
			return
		}
		pending.resolve(&response, nil)
		return
	}

	log.Warnf("Received unrecognized IoT MCP message: %s", string(message))
}

func (t *IotOverMcpTransport) SendRequest(ctx context.Context, request transport.JSONRPCRequest) (*transport.JSONRPCResponse, error) {
	t.closedMux.RLock()
	if t.closed {
		t.closedMux.RUnlock()
		return nil, fmt.Errorf("connection is closed")
	}
	t.closedMux.RUnlock()

	idStr := requestIDKey(request.ID)
	pending := newPendingResponse()

	t.respChansMux.Lock()
	t.respChans[idStr] = pending
	t.respChansMux.Unlock()

	payload, err := json.Marshal(request)
	if err != nil {
		t.popPending(idStr)
		return nil, err
	}

	t.writeMux.Lock()
	err = t.conn.SendMcpMsg(payload)
	t.writeMux.Unlock()
	if err != nil {
		t.popPending(idStr)
		return nil, err
	}

	select {
	case result := <-pending.resultCh:
		if result.err != nil {
			return nil, result.err
		}
		return result.response, nil
	case <-ctx.Done():
		t.popPending(idStr)
		return nil, ctx.Err()
	case <-time.After(t.requestTimeout):
		t.popPending(idStr)
		return nil, fmt.Errorf("request timeout")
	}
}

func (t *IotOverMcpTransport) SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	t.closedMux.RLock()
	if t.closed {
		t.closedMux.RUnlock()
		return fmt.Errorf("connection is closed")
	}
	t.closedMux.RUnlock()

	payload, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	t.writeMux.Lock()
	err = t.conn.SendMcpMsg(payload)
	t.writeMux.Unlock()
	return err
}

func (t *IotOverMcpTransport) SetNotificationHandler(handler func(notification mcp.JSONRPCNotification)) {
	t.notifyHandler = handler
}

func (t *IotOverMcpTransport) SetActivityHandler(handler func()) {
	t.activityHandler = handler
}

// SetOnCloseHandler sets the connection close callback
func (t *IotOverMcpTransport) SetOnCloseHandler(handler func(reason string)) {
	t.onCloseHandler = handler
}

func (t *IotOverMcpTransport) Close() error {
	t.closedMux.Lock()
	t.closed = true
	t.closedMux.Unlock()
	t.failAllPending(fmt.Errorf("connection is closed"))

	//Notify the client layer that the connection is about to be closed
	if t.onCloseHandler != nil {
		t.onCloseHandler("manual_close")
	}

	t.cancel()

	select {
	case <-t.readDone:
	case <-time.After(t.closeTimeout):
		log.Warnf("Timeout waiting for IoT MCP read goroutine to finish")
	}
	return nil
}

func (t *IotOverMcpTransport) GetSessionId() string {
	return ""
}
