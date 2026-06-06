package websocket

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"time"
	"xiaozhi-esp32-server-golang/internal/app/server/types"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"
)

// WebSocketConn implements the types.IConn interface for WebSocket connections
type WebSocketConn struct {
	ctx    context.Context
	cancel context.CancelFunc

	onCloseCbList []func(deviceId string)

	conn     *websocket.Conn
	deviceID string

	isMqttUdpBridge bool
	recvCmdChan     chan []byte
	recvAudioChan   chan []byte

	closed bool
	sync.RWMutex
}

// NewWebSocketConn creates a new WebSocketConn instance
func NewWebSocketConn(conn *websocket.Conn, deviceID string, isMqttUdpBridge bool) *WebSocketConn {
	ctx, cancel := context.WithCancel(context.Background())
	instance := &WebSocketConn{
		ctx:             ctx,
		cancel:          cancel,
		conn:            conn,
		deviceID:        deviceID,
		isMqttUdpBridge: isMqttUdpBridge,
		recvCmdChan:     make(chan []byte, 100),
		recvAudioChan:   make(chan []byte, 100),
	}

	// Set pong handler
	conn.SetPongHandler(func(appData string) error {
		log.Debugf("Received pong message, deviceID: %s", deviceID)
		return nil
	})

	// Start heartbeat goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second) // send ping every 30 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := instance.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
					log.Errorf("Failed to send ping message, deviceID: %s, error: %v", deviceID, err)
					// heartbeat failed, close connection
					for _, cb := range instance.onCloseCbList {
						cb(instance.deviceID)
					}
					return
				}
				log.Debugf("Ping message sent successfully, deviceID: %s", deviceID)
			case <-instance.ctx.Done():
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-instance.ctx.Done():
				return
			default:
				msgType, audio, err := instance.conn.ReadMessage()
				if err != nil {
					log.Errorf("read message error: %v", err)
					for _, cb := range instance.onCloseCbList {
						cb(instance.deviceID) // notify registered listeners to exit
					}
					return
				}

				if msgType == websocket.TextMessage {
					select {
					case instance.recvCmdChan <- audio:
					default:
						log.Errorf("recv cmd channel is full")
					}
				} else if msgType == websocket.BinaryMessage {
					if instance.isMqttUdpBridge {
						audio = instance.tryUnpackUdpBridgeAudioPacket(audio)
					}
					select {
					case instance.recvAudioChan <- audio:
					default:
						log.Errorf("recv audio channel is full")
					}
				}
			}
		}
	}()

	return instance
}

// Adapts the MQTT UDP bridge data format.
// First 8 bytes are 0, bytes 12-16 are audio data length, bytes after 16 are audio data.
func (c *WebSocketConn) tryUnpackUdpBridgeAudioPacket(buffer []byte) []byte {
	if len(buffer) < 16 {
		return buffer
	}
	// check if first 8 bytes are all zero
	for i := 0; i < 8; i++ {
		if buffer[i] != 0 {
			return buffer
		}
	}
	dataLen := binary.BigEndian.Uint32(buffer[12:16])
	if int(dataLen) != len(buffer)-16 {
		return buffer
	}
	audioData := buffer[16:]
	return audioData
}

func (c *WebSocketConn) packUdpBridgeAudioPacket(buffer []byte) []byte {
	header := make([]byte, 16)
	// first 8 bytes are all zero, already initialized
	// bytes 9-12: current Unix timestamp (seconds)
	timestamp := uint32(time.Now().Unix())
	binary.BigEndian.PutUint32(header[8:12], timestamp)
	// bytes 13-16: audio data length
	binary.BigEndian.PutUint32(header[12:16], uint32(len(buffer)))
	// concatenate header and audio data
	return append(header, buffer...)
}

func (w *WebSocketConn) SendCmd(msg []byte) error {
	w.Lock()
	defer w.Unlock()

	if w.closed {
		return errors.New("connection is closed")
	}

	log.Debugf("send cmd: %s", string(msg))

	err := w.conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Errorf("send cmd error: %v", err)
		return err
	}
	return nil
}

func (w *WebSocketConn) SendAudio(audio []byte) error {
	w.Lock()
	defer w.Unlock()

	if w.closed {
		return errors.New("connection is closed")
	}

	if w.isMqttUdpBridge {
		audio = w.packUdpBridgeAudioPacket(audio)
	}
	err := w.conn.WriteMessage(websocket.BinaryMessage, audio)
	if err != nil {
		log.Errorf("send audio error: %v", err)
		return err
	}
	return nil
}

func (w *WebSocketConn) RecvCmd(ctx context.Context, timeout int) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			log.Debugf("recv cmd context done")
			return nil, ctx.Err()
		case msg, ok := <-w.recvCmdChan:
			if !ok {
				return nil, errors.New("connection is closed")
			}
			return msg, nil
		case <-time.After(time.Duration(timeout) * time.Second):
			return nil, errors.New("timeout")
		}
	}
}

func (w *WebSocketConn) RecvAudio(ctx context.Context, timeout int) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			log.Debugf("recv audio context done")
			return nil, ctx.Err()
		case audio, ok := <-w.recvAudioChan:
			if !ok {
				return nil, errors.New("connection is closed")
			}
			return audio, nil
		case <-time.After(time.Duration(timeout) * time.Second):
			return nil, errors.New("timeout")
		}
	}
}

func (w *WebSocketConn) Close() error {
	w.Lock()
	defer w.Unlock()

	if w.closed {
		return nil // Already closed
	}

	w.closed = true
	w.cancel()
	w.conn.Close()
	close(w.recvCmdChan)
	close(w.recvAudioChan)
	return nil
}

func (w *WebSocketConn) OnClose(cb func(deviceId string)) {
	w.onCloseCbList = append(w.onCloseCbList, cb)
}

func (w *WebSocketConn) GetDeviceID() string {
	return w.deviceID
}

func (w *WebSocketConn) GetTransportType() string {
	return types.TransportTypeWebsocket
}

func (w *WebSocketConn) GetData(key string) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (w *WebSocketConn) CloseAudioChannel() error {
	return nil
}
