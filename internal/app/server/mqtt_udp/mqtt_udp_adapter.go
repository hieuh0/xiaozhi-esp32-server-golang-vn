package mqtt_udp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"xiaozhi-esp32-server-golang/internal/app/server/types"
	"xiaozhi-esp32-server-golang/internal/data/client"
	. "xiaozhi-esp32-server-golang/internal/data/client"
	msgdata "xiaozhi-esp32-server-golang/internal/data/msg"
	. "xiaozhi-esp32-server-golang/logger"
	log "xiaozhi-esp32-server-golang/logger"
)

type MqttConfig struct {
	Broker   string
	Type     string
	Port     int
	ClientID string
	Username string
	Password string
}

// MqttUdpAdapter is an MQTT-UDP adapter.
type MqttUdpAdapter struct {
	client             mqtt.Client
	udpServer          *UdpServer
	mqttConfig         *MqttConfig
	deviceId2Conn      *sync.Map
	lifecycleStates    *sync.Map
	msgChan            chan mqtt.Message
	onNewConnection    types.OnNewConnection
	onDeviceOnline     func(deviceID string)
	onDeviceOffline    func(deviceID string)
	onTransportReady   func(deviceID string)
	offlineGracePeriod time.Duration
	stopCtx            context.Context
	stopCancel         context.CancelFunc
	sync.RWMutex
}

type mqttDeviceLifecycleState struct {
	mu             sync.Mutex
	brokerOnline   bool
	lastEventTs    int64
	cleanupTimer   *time.Timer
	cleanupVersion uint64
}

const defaultOfflineGracePeriod = 2 * time.Minute

// MqttUdpAdapterOption configures optional adapter parameters.
type MqttUdpAdapterOption func(*MqttUdpAdapter)

// WithUdpServer sets the UDP server.
func WithUdpServer(udpServer *UdpServer) MqttUdpAdapterOption {
	return func(s *MqttUdpAdapter) {
		s.udpServer = udpServer
	}
}

func WithOnNewConnection(onNewConnection types.OnNewConnection) MqttUdpAdapterOption {
	return func(s *MqttUdpAdapter) {
		s.onNewConnection = onNewConnection
	}
}

func WithOnDeviceOnline(onDeviceOnline func(deviceID string)) MqttUdpAdapterOption {
	return func(s *MqttUdpAdapter) {
		s.onDeviceOnline = onDeviceOnline
	}
}

func WithOnDeviceOffline(onDeviceOffline func(deviceID string)) MqttUdpAdapterOption {
	return func(s *MqttUdpAdapter) {
		s.onDeviceOffline = onDeviceOffline
	}
}

func WithOnTransportReady(onTransportReady func(deviceID string)) MqttUdpAdapterOption {
	return func(s *MqttUdpAdapter) {
		s.onTransportReady = onTransportReady
	}
}

func WithOfflineGracePeriod(gracePeriod time.Duration) MqttUdpAdapterOption {
	return func(s *MqttUdpAdapter) {
		s.offlineGracePeriod = gracePeriod
	}
}

// NewMqttUdpAdapter creates an MQTT-UDP adapter. Config is required; other parameters use options.
func NewMqttUdpAdapter(config *MqttConfig, opts ...MqttUdpAdapterOption) *MqttUdpAdapter {
	ctx, cancel := context.WithCancel(context.Background())
	s := &MqttUdpAdapter{
		mqttConfig:         config,
		deviceId2Conn:      &sync.Map{},
		lifecycleStates:    &sync.Map{},
		msgChan:            make(chan mqtt.Message, 10000),
		offlineGracePeriod: defaultOfflineGracePeriod,
		stopCtx:            ctx,
		stopCancel:         cancel,
	}
	for _, opt := range opts {
		opt(s)
	}

	go s.processMessage()
	return s
}

func (s *MqttUdpAdapter) getClient() mqtt.Client {
	s.RLock()
	client := s.client
	s.RUnlock()
	return client
}

func (s *MqttUdpAdapter) setClient(client mqtt.Client) {
	s.Lock()
	s.client = client
	s.Unlock()
	s.updateSessionsClient(client)
}

func (s *MqttUdpAdapter) getUdpServer() *UdpServer {
	s.RLock()
	udpServer := s.udpServer
	s.RUnlock()
	return udpServer
}

func (s *MqttUdpAdapter) setUdpServer(udpServer *UdpServer) {
	s.Lock()
	s.udpServer = udpServer
	s.Unlock()
}

func (s *MqttUdpAdapter) updateSessionsClient(client mqtt.Client) {
	s.deviceId2Conn.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*MqttUdpConn); ok {
			conn.SetMqttClient(client)
		}
		return true
	})
}

func (s *MqttUdpAdapter) clearDeviceSessions() {
	s.deviceId2Conn.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*MqttUdpConn); ok {
			conn.Destroy()
		}
		s.deviceId2Conn.Delete(key)
		return true
	})
}

func (s *MqttUdpAdapter) clearLifecycleStates() {
	if s.lifecycleStates == nil {
		return
	}
	s.lifecycleStates.Range(func(key, value interface{}) bool {
		if state, ok := value.(*mqttDeviceLifecycleState); ok {
			state.mu.Lock()
			if state.cleanupTimer != nil {
				state.cleanupTimer.Stop()
				state.cleanupTimer = nil
			}
			state.mu.Unlock()
		}
		s.lifecycleStates.Delete(key)
		return true
	})
}

// Start launches the MQTT client without blocking and retries connections in a background goroutine.
func (s *MqttUdpAdapter) Start() error {
	Infof("starting MqttUdpAdapter; connecting to MQTT server in background Broker=%s:%d ClientID=%s", s.mqttConfig.Broker, s.mqttConfig.Port, s.mqttConfig.ClientID)
	go s.connectAndRetry()
	return nil
}

// connectAndRetry connects to MQTT in the background and retries without blocking the main flow.
func (s *MqttUdpAdapter) connectAndRetry() {
	const retryInterval = 5 * time.Second

	s.RLock()
	cfg := s.mqttConfig
	s.RUnlock()
	if cfg == nil {
		return
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", cfg.Type, cfg.Broker, cfg.Port))
	opts.SetClientID(cfg.ClientID)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		Errorf("MQTT connection lost: %v", err)
	})

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		Info("MQTT connected")
		topic := ServerSubTopicPrefix
		if token := client.Subscribe(topic, 0, s.handleMessage); token.Wait() && token.Error() != nil {
			Errorf("failed to subscribe to topic: %v", token.Error())
		}
	})

	var retryCount int
	for {
		select {
		case <-s.stopCtx.Done():
			return
		default:
		}
		client := mqtt.NewClient(opts)
		s.setClient(client)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			retryCount++
			Errorf("failed to connect to MQTT server (attempt %d): %v; retrying in %d seconds", retryCount, token.Error(), int(retryInterval.Seconds()))
			select {
			case <-s.stopCtx.Done():
				return
			case <-time.After(retryInterval):
				continue
			}
		}
		break
	}

	_ = s.checkClientActive()
}

func (s *MqttUdpAdapter) checkClientActive() error {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCtx.Done():
				return
			case <-ticker.C:
				s.deviceId2Conn.Range(func(key, value interface{}) bool {
					conn := value.(*MqttUdpConn)
					if !conn.IsActive() {
						conn.Destroy()
					}
					return true
				})
			}
		}
	}()
	return nil
}

func (s *MqttUdpAdapter) SetDeviceSession(deviceId string, conn *MqttUdpConn) {
	Debugf("SetDeviceSession, deviceId: %s", deviceId)
	s.deviceId2Conn.Store(deviceId, conn)
}

func (s *MqttUdpAdapter) getDeviceSession(deviceId string) *MqttUdpConn {
	Debugf("getDeviceSession, deviceId: %s", deviceId)
	if conn, ok := s.deviceId2Conn.Load(deviceId); ok {
		return conn.(*MqttUdpConn)
	}
	return nil
}

// handleMessage enqueues a message for processing.
func (s *MqttUdpAdapter) handleMessage(client mqtt.Client, msg mqtt.Message) {
	select {
	case s.msgChan <- msg:
		return
	default:
		Debugf("handleMessage msg chan is full, topic: %s, payload: %s", msg.Topic(), string(msg.Payload()))
	}
}

// Disconnect after a timeout or an explicit goodbye.
func (s *MqttUdpAdapter) handleDisconnect(deviceId string, closingConn *MqttUdpConn) {
	Debugf("handleDisconnect, deviceId: %s", deviceId)

	conn := s.getDeviceSession(deviceId)
	if closingConn != nil && conn != nil && conn != closingConn {
		Debugf("handleDisconnect, ignore stale conn for deviceId: %s", deviceId)
		return
	}

	notifyOffline := false
	if state := s.getLifecycleState(deviceId); state != nil {
		shouldDeleteState := false
		state.mu.Lock()
		if state.brokerOnline {
			state.brokerOnline = false
			state.cleanupVersion++
			if state.cleanupTimer != nil {
				state.cleanupTimer.Stop()
				state.cleanupTimer = nil
			}
			notifyOffline = true
		}
		shouldDeleteState = !state.brokerOnline && state.cleanupTimer == nil
		state.mu.Unlock()
		if shouldDeleteState {
			s.lifecycleStates.Delete(deviceId)
		}
	}

	if conn == nil {
		Debugf("handleDisconnect, deviceId: %s not found", deviceId)
		if notifyOffline && s.onDeviceOffline != nil {
			s.onDeviceOffline(deviceId)
		}
		return
	}
	conn.ReleaseUdpSession()
	s.deviceId2Conn.Delete(deviceId)
	if notifyOffline && s.onDeviceOffline != nil {
		s.onDeviceOffline(deviceId)
	}
}

// Stop cancels the context, disconnects MQTT, closes UDP, and cleans up sessions before hot reload.
func (s *MqttUdpAdapter) Stop() {
	Debugf("enter MqttUdpAdapter Stop ")
	defer Debugf("exit MqttUdpAdapter Stop ")
	s.stopCancel()
	client := s.getClient()
	if client != nil && client.IsConnected() {
		Debugf("MqttUdpAdapter Stop, disconnect mqtt client")
		client.Disconnect(250)
	}
	udpServer := s.getUdpServer()
	Debugf("MqttUdpAdapter Stop, udpServer: %v", udpServer)
	if udpServer != nil {
		Debugf("MqttUdpAdapter Stop, close udpServer")
		_ = udpServer.Close()
	}
	s.clearLifecycleStates()
	s.clearDeviceSessions()
}

// ReloadMqttClient reconnects only MQTT while preserving the UDP server instance.
func (s *MqttUdpAdapter) ReloadMqttClient(newConfig *MqttConfig) {
	if newConfig == nil {
		return
	}
	s.Lock()
	s.mqttConfig = newConfig
	oldClient := s.client
	s.Unlock()
	if oldClient != nil && oldClient.IsConnected() {
		oldClient.Disconnect(250)
	}
	s.clearLifecycleStates()
	s.clearDeviceSessions()
	go s.connectAndRetry()
}

// ReloadUdpServer restarts only UDP while preserving the MQTT connection.
func (s *MqttUdpAdapter) ReloadUdpServer(newUdpServer *UdpServer) {
	if newUdpServer == nil {
		return
	}
	oldUdp := s.getUdpServer()
	s.clearLifecycleStates()
	s.clearDeviceSessions()
	s.setUdpServer(newUdpServer)
	if oldUdp != nil {
		_ = oldUdp.Close()
	}
}

func (s *MqttUdpAdapter) getLifecycleState(deviceID string) *mqttDeviceLifecycleState {
	if s.lifecycleStates == nil || deviceID == "" {
		return nil
	}
	if state, ok := s.lifecycleStates.Load(deviceID); ok {
		if lifecycleState, ok := state.(*mqttDeviceLifecycleState); ok {
			return lifecycleState
		}
	}
	return nil
}

func (s *MqttUdpAdapter) getOrCreateLifecycleState(deviceID string) *mqttDeviceLifecycleState {
	if s.lifecycleStates == nil || deviceID == "" {
		return nil
	}
	if state := s.getLifecycleState(deviceID); state != nil {
		return state
	}
	newState := &mqttDeviceLifecycleState{}
	actual, _ := s.lifecycleStates.LoadOrStore(deviceID, newState)
	return actual.(*mqttDeviceLifecycleState)
}

func (s *MqttUdpAdapter) markDeviceOnline(deviceID string, eventTs int64) (bool, bool) {
	state := s.getOrCreateLifecycleState(deviceID)
	if state == nil {
		return false, false
	}
	if eventTs <= 0 {
		eventTs = time.Now().UnixMilli()
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.lastEventTs > 0 && eventTs < state.lastEventTs {
		return false, false
	}

	if eventTs > state.lastEventTs {
		state.lastEventTs = eventTs
	}

	wasOnline := state.brokerOnline
	state.brokerOnline = true
	state.cleanupVersion++
	if state.cleanupTimer != nil {
		state.cleanupTimer.Stop()
		state.cleanupTimer = nil
	}
	return !wasOnline, true
}

func (s *MqttUdpAdapter) markDeviceOffline(deviceID string, eventTs int64) (bool, uint64) {
	state := s.getOrCreateLifecycleState(deviceID)
	if state == nil {
		return false, 0
	}
	if eventTs <= 0 {
		eventTs = time.Now().UnixMilli()
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.lastEventTs > 0 && eventTs < state.lastEventTs {
		return false, 0
	}

	if eventTs > state.lastEventTs {
		state.lastEventTs = eventTs
	}

	wasOnline := state.brokerOnline
	state.brokerOnline = false
	state.cleanupVersion++
	version := state.cleanupVersion
	if state.cleanupTimer != nil {
		state.cleanupTimer.Stop()
	}
	state.cleanupTimer = time.AfterFunc(s.offlineGracePeriod, func() {
		s.cleanupOfflineTransport(deviceID, version)
	})
	return wasOnline, version
}

func (s *MqttUdpAdapter) cleanupOfflineTransport(deviceID string, version uint64) {
	state := s.getLifecycleState(deviceID)
	if state == nil {
		return
	}

	state.mu.Lock()
	if state.brokerOnline || state.cleanupVersion != version {
		state.mu.Unlock()
		return
	}
	state.cleanupTimer = nil
	state.mu.Unlock()

	conn := s.getDeviceSession(deviceID)
	if conn == nil {
		s.lifecycleStates.Delete(deviceID)
		return
	}
	conn.Destroy()
}

func (s *MqttUdpAdapter) EnsureDeviceTransport(deviceId string) (*MqttUdpConn, error) {
	deviceSession := s.getDeviceSession(deviceId)
	if deviceSession != nil {
		deviceSession.MarkBrokerOnline()
		return deviceSession, nil
	}

	udpServer, udpSession, err := s.createUdpSession(deviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to create udpSession, deviceId: %s, err: %w", deviceId, err)
	}

	topicMacAddr := strings.ReplaceAll(deviceId, ":", "_")
	publicTopic := fmt.Sprintf("%s%s", client.ServerPubTopicPrefix, topicMacAddr)

	mqttClient := s.getClient()
	if mqttClient == nil {
		udpServer.CloseSessionByRef(udpSession)
		return nil, fmt.Errorf("mqtt client is nil, deviceId: %s", deviceId)
	}

	deviceSession = NewMqttUdpConn(deviceId, publicTopic, mqttClient, udpServer, udpSession)
	s.bindUdpSessionData(deviceSession, udpSession)
	deviceSession.MarkBrokerOnline()

	s.SetDeviceSession(deviceId, deviceSession)
	deviceSession.OnClose(func(closedDeviceID string) {
		s.handleDisconnect(closedDeviceID, deviceSession)
	})

	if s.onNewConnection != nil {
		s.onNewConnection(deviceSession)
	}
	return deviceSession, nil
}

func (s *MqttUdpAdapter) promoteDeviceOnline(deviceID string, eventTs int64) (*MqttUdpConn, bool, bool, error) {
	notifyOnline, acceptedOnline := s.markDeviceOnline(deviceID, eventTs)
	if !acceptedOnline {
		return s.getDeviceSession(deviceID), false, false, nil
	}
	conn, err := s.EnsureDeviceTransport(deviceID)
	if err != nil {
		return nil, notifyOnline, true, err
	}
	if conn != nil {
		conn.MarkBrokerOnline()
	}
	return conn, notifyOnline, true, nil
}

func (s *MqttUdpAdapter) handleLifecycleMessage(payload []byte) {
	var lifecycleEvent msgdata.MqttLifecycleEvent
	if err := json.Unmarshal(payload, &lifecycleEvent); err != nil {
		Errorf("failed to parse MQTT lifecycle message: %v", err)
		return
	}
	deviceID := strings.TrimSpace(lifecycleEvent.DeviceID)
	if deviceID == "" {
		Errorf("MQTT lifecycle message is missing device_id: %s", string(payload))
		return
	}

	switch strings.TrimSpace(lifecycleEvent.State) {
	case msgdata.MqttLifecycleStateOnline:
		_, notifyOnline, acceptedOnline, err := s.promoteDeviceOnline(deviceID, lifecycleEvent.Ts)
		if err != nil {
			Errorf("failed to process MQTT online event: device=%s err=%v", deviceID, err)
			return
		}
		if !acceptedOnline {
			return
		}
		if notifyOnline && s.onDeviceOnline != nil {
			s.onDeviceOnline(deviceID)
		}
		// Even if broker presence does not change after a device restart, an online broadcast can mean
		// the device runtime was rebuilt, so trigger transport readiness to reset and reinitialize IoT MCP.
		if s.onTransportReady != nil {
			s.onTransportReady(deviceID)
		}
	case msgdata.MqttLifecycleStateOffline:
		notifyOffline, cleanupVersion := s.markDeviceOffline(deviceID, lifecycleEvent.Ts)
		if cleanupVersion == 0 {
			return
		}
		conn := s.getDeviceSession(deviceID)
		if conn != nil {
			conn.MarkBrokerOffline(s.offlineGracePeriod)
		}
		if notifyOffline && s.onDeviceOffline != nil {
			s.onDeviceOffline(deviceID)
		}
	default:
		Warnf("ignoring unknown MQTT lifecycle state: device=%s state=%s", deviceID, lifecycleEvent.State)
	}
}

// processMessage processes queued MQTT messages.
func (s *MqttUdpAdapter) processMessage() {
	for {
		select {
		case <-s.stopCtx.Done():
			return
		case mqttMsg := <-s.msgChan:
			Debugf("mqtt handleMessage, topic: %s, payload: %s", mqttMsg.Topic(), string(mqttMsg.Payload()))
			if mqttMsg.Topic() == msgdata.MDeviceLifecycleTopic {
				s.handleLifecycleMessage(mqttMsg.Payload())
				continue
			}
			var clientMsg ClientMessage
			if err := json.Unmarshal(mqttMsg.Payload(), &clientMsg); err != nil {
				Errorf("failed to parse JSON: %v", err)
				continue
			}
			_, deviceId := s.getDeviceIdByTopic(mqttMsg.Topic())
			if deviceId == "" {
				Errorf("failed to parse mac_addr: %v", mqttMsg.Topic())
				continue
			}

			existingSession := s.getDeviceSession(deviceId)
			deviceSession, notifyOnline, _, err := s.promoteDeviceOnline(deviceId, time.Now().UnixMilli())
			if err != nil {
				Errorf("failed to ensure MQTT transport is online: device=%s err=%v", deviceId, err)
				continue
			}
			if notifyOnline && s.onDeviceOnline != nil {
				s.onDeviceOnline(deviceId)
			}
			if notifyOnline && s.onTransportReady != nil {
				s.onTransportReady(deviceId)
			}
			if existingSession != nil && clientMsg.Type == "hello" {
				newUdpSession, err := s.rotateDeviceUdpSession(deviceSession, deviceId)
				if err != nil {
					Errorf("hello failed to rebuild udpSession, deviceId: %s, err: %v", deviceId, err)
					continue
				}
				Debugf("hello rebuilt udpSession successfully, deviceId: %s, connID: %s", deviceId, newUdpSession.ConnId)
			}

			if err := deviceSession.PushMsgToRecvCmd(mqttMsg.Payload()); err != nil {
				Errorf("InternalRecvCmd failed: %v", err)
				continue
			}
		}
	}
}

func (s *MqttUdpAdapter) createUdpSession(deviceId string) (*UdpServer, *UdpSession, error) {
	udpServer := s.getUdpServer()
	if udpServer == nil {
		return nil, nil, fmt.Errorf("udpServer is nil")
	}
	udpSession := udpServer.CreateSession(deviceId, "")
	if udpSession == nil {
		return nil, nil, fmt.Errorf("udpSession is nil")
	}
	return udpServer, udpSession, nil
}

func (s *MqttUdpAdapter) bindUdpSessionData(deviceSession *MqttUdpConn, udpSession *UdpSession) {
	if deviceSession == nil || udpSession == nil {
		return
	}
	deviceSession.SetUdpSession(udpSession)
	strAesKey, strFullNonce := udpSession.GetAesKeyAndNonce()
	deviceSession.SetData("aes_key", strAesKey)
	deviceSession.SetData("full_nonce", strFullNonce)
}

func (s *MqttUdpAdapter) rotateDeviceUdpSession(deviceSession *MqttUdpConn, deviceId string) (*UdpSession, error) {
	if deviceSession == nil {
		return nil, fmt.Errorf("deviceSession is nil")
	}
	udpServer, udpSession, err := s.createUdpSession(deviceId)
	if err != nil {
		return nil, err
	}
	oldSession := deviceSession.GetUdpSession()
	s.bindUdpSessionData(deviceSession, udpSession)
	if oldSession != nil {
		udpServer.CloseSessionByRef(oldSession)
	}
	return udpSession, nil
}

func (s *MqttUdpAdapter) getDeviceIdByTopic(topic string) (string, string) {
	var topicMacAddr, deviceId string
	// Parse mac_addr from topic (/p2p/device_public/mac_addr).
	strList := strings.Split(topic, "/")
	if len(strList) == 4 {
		topicMacAddr = strList[3]

		// Check for the new format: "GID_test@@@ba_8f_17_de_94_94@@@e4b0c442-98fc-4e1b-8c3d-6a5b6a5b6a6d".
		if strings.Contains(topicMacAddr, "@@@") {
			parts := strings.Split(topicMacAddr, "@@@")
			if len(parts) >= 2 {
				// Extract the middle component as the MAC address.
				macAddr := parts[1]
				deviceId = strings.ReplaceAll(macAddr, "_", ":")
			}
		} else {
			deviceId = strings.ReplaceAll(topicMacAddr, "_", ":")
		}
	}

	log.Log().Debugf("topicMacAddr: %s, deviceId: %s", topicMacAddr, deviceId)
	return topicMacAddr, deviceId
}
