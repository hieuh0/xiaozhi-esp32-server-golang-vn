package mqtt_udp

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
	"xiaozhi-esp32-server-golang/internal/app/server/types"

	log "xiaozhi-esp32-server-golang/logger"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	MaxIdleDuration = 300 // Disconnect after 300 seconds without inbound or outbound data.
)

// MqttUdpConn implements types.IConn for MQTT-UDP connections.
// Extend its methods and fields as needed.

type MqttUdpConn struct {
	ctx    context.Context
	cancel context.CancelFunc

	DeviceId string

	PubTopic   string
	MqttClient mqtt.Client
	udpServer  *UdpServer

	UdpSession *UdpSession

	recvCmdChan chan []byte
	sync.RWMutex

	data sync.Map

	onCloseCbList []func(deviceId string)

	lastActiveTs    int64 // Updated by inbound and outbound signaling and audio data.
	retainedUntilTs int64
	brokerOnline    atomic.Bool
}

// NewMqttUdpConn creates a MqttUdpConn instance.
func NewMqttUdpConn(deviceID string, pubTopic string, mqttClient mqtt.Client, udpServer *UdpServer, udpSession *UdpSession) *MqttUdpConn {
	ctx, cancel := context.WithCancel(context.Background())
	log.Log().Debugf("NewMqttUdpConn pubTopic: %s", pubTopic)
	return &MqttUdpConn{
		ctx:      ctx,
		cancel:   cancel,
		DeviceId: deviceID,

		PubTopic:   pubTopic,
		MqttClient: mqttClient,
		udpServer:  udpServer,
		UdpSession: udpSession,

		recvCmdChan: make(chan []byte, 100),

		data: sync.Map{},

		lastActiveTs: time.Now().Unix(),
	}
}

// SendCmd sends a command through MQTT-UDP.
func (c *MqttUdpConn) SendCmd(msg []byte) error {
	//log.Debugf("mqtt udp conn send cmd, topic: %s, msg: %s", c.PubTopic, string(msg))
	atomic.StoreInt64(&c.lastActiveTs, time.Now().Unix())
	c.RLock()
	client := c.MqttClient
	c.RUnlock()
	if client == nil {
		return errors.New("mqtt client is nil")
	}
	token := client.Publish(c.PubTopic, 0, false, msg)
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c *MqttUdpConn) PushMsgToRecvCmd(msg []byte) error {
	select {
	case c.recvCmdChan <- msg:
		atomic.StoreInt64(&c.lastActiveTs, time.Now().Unix())
		return nil
	default:
		return errors.New("recvCmdChan is full")
	}
}

// RecvCmd receives command or signaling data.
func (c *MqttUdpConn) RecvCmd(ctx context.Context, timeout int) ([]byte, error) {
	select {
	case <-ctx.Done():
		log.Debugf("mqtt udp conn recv cmd context done")
		return nil, ctx.Err()
	case msg := <-c.recvCmdChan:
		return msg, nil
	case <-time.After(time.Duration(timeout) * time.Second):
		log.Debugf("mqtt udp conn recv cmd timeout")
		return nil, nil
	}
}

// SendAudio sends audio through MQTT-UDP.
func (c *MqttUdpConn) SendAudio(audio []byte) error {
	udpSession := c.GetUdpSession()
	if udpSession == nil {
		return nil
	}
	ok, err := udpSession.SendAudioData(audio)
	if err != nil {
		return err
	}
	if !ok {
		if udpSession.IsClosed() {
			return nil
		}
		return errors.New("sendAudioChan is full")
	}
	return nil
	/*
		select {
		case c.UdpSession.SendChannel <- audio:
			c.lastActiveTs = time.Now().Unix()
			return nil
		default:
			return errors.New("sendAudioChan is full")
		}*/
}

// RecvAudio receives audio data.
func (c *MqttUdpConn) RecvAudio(ctx context.Context, timeout int) ([]byte, error) {
	udpSession := c.GetUdpSession()
	if udpSession == nil {
		waitDuration := time.Second
		if timeout > 0 {
			timeoutDuration := time.Duration(timeout) * time.Second
			if timeoutDuration < waitDuration {
				waitDuration = timeoutDuration
			}
		}
		select {
		case <-ctx.Done():
			log.Debugf("mqtt udp conn recv audio context done")
			return nil, ctx.Err()
		case <-time.After(waitDuration):
			return nil, nil
		}
	}
	select {
	case <-ctx.Done():
		log.Debugf("mqtt udp conn recv audio context done")
		return nil, ctx.Err()
	case audio, ok := <-udpSession.RecvChannel:
		if ok {
			atomic.StoreInt64(&c.lastActiveTs, time.Now().Unix())
			return audio, nil
		}
		return nil, nil
	case <-time.After(time.Duration(timeout) * time.Second):
		log.Debugf("mqtt udp conn recv audio timeout")
		return nil, nil
	}
}

// GetDeviceID returns the device ID.
func (c *MqttUdpConn) GetDeviceID() string {
	return c.DeviceId
}

// Close closes the connection.
func (c *MqttUdpConn) Close() error {
	//c.cancel()
	c.Destroy()
	return nil
}

func (c *MqttUdpConn) OnClose(closeCb func(deviceId string)) {
	c.onCloseCbList = append(c.onCloseCbList, closeCb)
}

func (c *MqttUdpConn) SetMqttClient(client mqtt.Client) {
	c.Lock()
	c.MqttClient = client
	c.Unlock()
}

func (c *MqttUdpConn) GetUdpSession() *UdpSession {
	c.RLock()
	defer c.RUnlock()
	return c.UdpSession
}

func (c *MqttUdpConn) SetUdpSession(session *UdpSession) {
	c.Lock()
	c.UdpSession = session
	c.Unlock()
}

func (c *MqttUdpConn) ReleaseUdpSession() {
	c.Lock()
	udpSession := c.UdpSession
	c.UdpSession = nil
	c.Unlock()
	if udpSession == nil {
		return
	}
	if c.udpServer != nil {
		c.udpServer.CloseSessionByRef(udpSession)
		return
	}
	udpSession.Destroy()
}

func (c *MqttUdpConn) GetTransportType() string {
	return types.TransportTypeMqttUdp
}

func (c *MqttUdpConn) SetData(key string, value interface{}) {
	c.data.Store(key, value)
}

func (c *MqttUdpConn) GetData(key string) (interface{}, error) {
	value, ok := c.data.Load(key)
	if !ok {
		return nil, errors.New("key not found")
	}
	return value, nil
}

func (c *MqttUdpConn) IsActive() bool {
	now := time.Now()
	if c.brokerOnline.Load() {
		return true
	}
	retainedUntil := atomic.LoadInt64(&c.retainedUntilTs)
	if retainedUntil > 0 && now.Before(time.Unix(0, retainedUntil)) {
		return true
	}
	if retainedUntil > 0 {
		return false
	}
	return now.Unix()-atomic.LoadInt64(&c.lastActiveTs) < MaxIdleDuration
}

// Destroy releases the connection resources.
func (c *MqttUdpConn) Destroy() {
	c.brokerOnline.Store(false)
	atomic.StoreInt64(&c.retainedUntilTs, 0)
	c.cancel()
	for _, cb := range c.onCloseCbList {
		cb(c.DeviceId)
	}
}

func (c *MqttUdpConn) CloseAudioChannel() error {
	c.ReleaseUdpSession()
	return nil
}

func (c *MqttUdpConn) MarkBrokerOnline() {
	c.brokerOnline.Store(true)
	atomic.StoreInt64(&c.retainedUntilTs, 0)
	atomic.StoreInt64(&c.lastActiveTs, time.Now().Unix())
}

func (c *MqttUdpConn) MarkBrokerOffline(gracePeriod time.Duration) {
	c.brokerOnline.Store(false)
	atomic.StoreInt64(&c.retainedUntilTs, time.Now().Add(gracePeriod).UnixNano())
}

func (c *MqttUdpConn) IsBrokerOnline() bool {
	return c.brokerOnline.Load()
}
