package types

import "context"

// IConn is the protocol-independent connection interface implemented by adapters such as websocket and mqtt_udp.
// Extend it when additional transport operations are required.

const (
	TransportTypeWebsocket = "websocket"
	TransportTypeMqttUdp   = "udp"
)

type IConn interface {
	// Send command or signaling data.
	SendCmd(msg []byte) error
	// Receive command or signaling data.
	RecvCmd(ctx context.Context, timeout int) ([]byte, error)
	// Send audio data.
	SendAudio(audio []byte) error
	// Receive audio data.
	RecvAudio(ctx context.Context, timeout int) ([]byte, error)

	GetDeviceID() string

	Close() error
	OnClose(func(deviceId string))

	CloseAudioChannel() error

	GetTransportType() string

	// Get private data.
	GetData(key string) (interface{}, error)
}

type OnNewConnection func(conn IConn)
