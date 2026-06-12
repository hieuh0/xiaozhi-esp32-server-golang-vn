package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"xiaozhi-esp32-server-golang/internal/app/mqtt_server"
	"xiaozhi-esp32-server-golang/internal/app/server/chat"
	"xiaozhi-esp32-server-golang/internal/app/server/mqtt_udp"
	"xiaozhi-esp32-server-golang/internal/app/server/types"
	"xiaozhi-esp32-server-golang/internal/app/server/websocket"
	"xiaozhi-esp32-server-golang/internal/data/history"
	user_config "xiaozhi-esp32-server-golang/internal/domain/config"
	config_types "xiaozhi-esp32-server-golang/internal/domain/config/types"
	"xiaozhi-esp32-server-golang/internal/domain/mcp"
	"xiaozhi-esp32-server-golang/internal/domain/openclaw"
	"xiaozhi-esp32-server-golang/internal/pool"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/spf13/viper"
)

// App manages all protocol services and ChatManager instances.

type App struct {
	wsServer       *websocket.WebSocketServer
	mqttUdpAdapter *mqtt_udp.MqttUdpAdapter
	mqttUdpMu      sync.RWMutex

	// Manage ChatManager instances with a concurrent map.
	chatManagers cmap.ConcurrentMap[string, *chat.ChatManager]
}

func NewApp() *App {
	var err error
	app := &App{
		chatManagers: cmap.New[*chat.ChatManager](),
	}
	mcp.RegisterCurrentDeviceTransportResolver(func(deviceID string) string {
		chatManager, exists := app.GetChatManager(deviceID)
		if !exists || chatManager == nil {
			return ""
		}
		return chatManager.GetTransportType()
	})
	app.wsServer = app.newWebSocketServer()
	app.mqttUdpAdapter, err = app.newMqttUdpAdapter()
	if err != nil {
		log.Errorf("newMqttUdpAdapter err: %+v", err)
		return nil
	}
	if app.mqttUdpAdapter != nil {
		mqtt_udp.RegisterGlobalAdapter(app.mqttUdpAdapter)
	}
	return app
}

func (a *App) Run() {
	go a.wsServer.Start()
	log.Infof("enter Run, mqtt_server.enable: %v", viper.GetBool("mqtt_server.enable"))
	if viper.GetBool("mqtt_server.enable") {
		go func() {
			err := a.startMqttServer()
			if err != nil {
				log.Errorf("startMqttServer err: %+v", err)
			}
		}()
	}
	a.mqttUdpMu.RLock()
	adapter := a.mqttUdpAdapter
	a.mqttUdpMu.RUnlock()
	if adapter != nil {
		go adapter.Start() // Non-blocking; the adapter handles connections and retries in the background.
	}

	// Register local chat-related MCP tools.
	a.registerChatMCPTools()

	a.registerHandler()

	a.initEventHandle()

	// Start resource pool statistics monitoring, logging every five minutes.
	ctx := context.Background()
	pool.StartStatsMonitor(ctx, 5*time.Minute)

	// Start reporting resource pool statistics to the manager backend every five seconds.
	pool.StartStatsReporter(ctx)

	select {} // Block the main goroutine.
}

func (app *App) initEventHandle() {
	eventHandle, err := NewEventHandle(app)
	if err != nil {
		log.Errorf("failed to initialize EventHandle: %v", err)
		return
	}
	if err := eventHandle.Start(); err != nil {
		log.Errorf("failed to start EventHandle: %v", err)
		return
	}

	// Initialize the always-enabled message worker for Redis, MemoryProvider, and History.
	historyCfg := history.HistoryClientConfig{
		BaseURL:   util.GetBackendURL(),
		AuthToken: util.GetManagerAuthToken(),
		Timeout:   viper.GetDuration("manager.history_timeout"),
		Enabled:   true, // Always enabled.
	}
	NewMessageWorker(historyCfg)
	log.Info("message worker initialized")
}

func (app *App) currentMqttConfig() *mqtt_udp.MqttConfig {
	if !viper.GetBool("mqtt.enable") {
		return nil
	}
	return &mqtt_udp.MqttConfig{
		Broker:   viper.GetString("mqtt.broker"),
		Type:     viper.GetString("mqtt.type"),
		Port:     viper.GetInt("mqtt.port"),
		ClientID: viper.GetString("mqtt.client_id"),
		Username: viper.GetString("mqtt.username"),
		Password: viper.GetString("mqtt.password"),
	}
}

func (app *App) newMqttUdpAdapter() (*mqtt_udp.MqttUdpAdapter, error) {
	mqttConfig := app.currentMqttConfig()
	if mqttConfig == nil {
		return nil, nil
	}

	udpServer, err := app.newUdpServer()
	if err != nil {
		return nil, err
	}

	return mqtt_udp.NewMqttUdpAdapter(
		mqttConfig,
		mqtt_udp.WithUdpServer(udpServer),
		mqtt_udp.WithOnNewConnection(app.OnNewConnection),
		mqtt_udp.WithOnDeviceOnline(app.DeviceOnline),
		mqtt_udp.WithOnDeviceOffline(app.DeviceOffline),
		mqtt_udp.WithOnTransportReady(app.onMqttTransportReady),
		mqtt_udp.WithOfflineGracePeriod(app.mqttOfflineGracePeriod()),
	), nil
}

func (app *App) mqttOfflineGracePeriod() time.Duration {
	if duration := viper.GetDuration("mqtt.transport_offline_grace_period"); duration > 0 {
		return duration
	}
	if seconds := viper.GetInt("mqtt.transport_offline_grace_period_seconds"); seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return 2 * time.Minute
}

func (app *App) newUdpServer() (*mqtt_udp.UdpServer, error) {
	udpPort := viper.GetInt("udp.listen_port")
	externalHost := viper.GetString("udp.external_host")
	externalPort := viper.GetInt("udp.external_port")

	udpServer := mqtt_udp.NewUDPServer(udpPort, externalHost, externalPort)
	err := udpServer.Start()
	if err != nil {
		log.Fatalf("udpServer.Start err: %+v", err)
		return nil, err
	}
	return udpServer, nil
}

func (app *App) newWebSocketServer() *websocket.WebSocketServer {
	port := viper.GetInt("websocket.port")
	return websocket.NewWebSocketServer(
		port,
		websocket.WithOnNewConnection(app.OnNewConnection),
		websocket.WithOnOpenClawResponse(app.OnOpenClawResponse),
		websocket.WithOnInjectMessage(func(deviceID, message string, skipLlm bool, autoListen bool) error {
			chatManager, exists := app.GetChatManager(deviceID)
			if !exists || chatManager == nil {
				return fmt.Errorf("device %s not found or offline", deviceID)
			}
			return chatManager.InjectMessage(message, skipLlm, autoListen)
		}),
	)
}

func (app *App) startMqttServer() error {
	return mqtt_server.StartMqttServer()
}

// ReloadMqttServer stops the MQTT server, then restarts it when mqtt_server.enable is true.
func (app *App) ReloadMqttServer() {
	_ = mqtt_server.StopMqttServer()
	if !viper.GetBool("mqtt_server.enable") {
		return
	}
	if err := app.startMqttServer(); err != nil {
		log.Errorf("ReloadMqttServer start: %v", err)
	}
}

// ReloadMqttUdp stops the old MQTT+UDP adapter, then recreates it when mqtt.enable is true.
func (app *App) ReloadMqttUdp() {
	app.mqttUdpMu.Lock()
	old := app.mqttUdpAdapter
	app.mqttUdpAdapter = nil
	app.mqttUdpMu.Unlock()
	if old != nil {
		old.Stop()
	}
	if !viper.GetBool("mqtt.enable") {
		return
	}
	adapter, err := app.newMqttUdpAdapter()
	if err != nil {
		log.Errorf("ReloadMqttUdp newMqttUdpAdapter: %v", err)
		return
	}
	app.mqttUdpMu.Lock()
	app.mqttUdpAdapter = adapter
	app.mqttUdpMu.Unlock()
	mqtt_udp.RegisterGlobalAdapter(adapter)
	time.Sleep(500 * time.Millisecond)
	go adapter.Start()
}

// ReloadMqttUdpWithFlags reloads MQTT+UDP according to the change flags.
func (app *App) ReloadMqttUdpWithFlags(doMqttReload, doUdpReload bool) {
	if !doMqttReload && !doUdpReload {
		return
	}
	if !viper.GetBool("mqtt.enable") {
		log.Infof("ReloadMqttUdpWithFlags: mqtt disabled, stopping mqtt+udp")
		app.ReloadMqttUdp()
		return
	}

	app.mqttUdpMu.RLock()
	adapter := app.mqttUdpAdapter
	app.mqttUdpMu.RUnlock()

	if adapter == nil {
		log.Infof("ReloadMqttUdpWithFlags: mqtt enabled but adapter is nil, starting mqtt+udp")
		newAdapter, err := app.newMqttUdpAdapter()
		if err != nil {
			log.Errorf("ReloadMqttUdpWithFlags newMqttUdpAdapter: %v", err)
			return
		}
		if newAdapter == nil {
			return
		}
		app.mqttUdpMu.Lock()
		app.mqttUdpAdapter = newAdapter
		app.mqttUdpMu.Unlock()
		mqtt_udp.RegisterGlobalAdapter(newAdapter)
		time.Sleep(500 * time.Millisecond)
		go newAdapter.Start()
		return
	}

	if doMqttReload && doUdpReload {
		log.Infof("ReloadMqttUdpWithFlags: mqtt & udp config changed, reloading mqtt+udp")
		app.ReloadMqttUdp()
		return
	}
	if doMqttReload {
		log.Infof("ReloadMqttUdpWithFlags: mqtt config changed, reloading mqtt only")
		mqttConfig := app.currentMqttConfig()
		if mqttConfig == nil {
			app.ReloadMqttUdp()
			return
		}
		adapter.ReloadMqttClient(mqttConfig)
		return
	}
	if doUdpReload {
		log.Infof("ReloadMqttUdpWithFlags: udp listen changed, reloading udp only")
		udpServer, err := app.newUdpServer()
		if err != nil {
			log.Errorf("ReloadMqttUdpWithFlags newUdpServer: %v", err)
			return
		}
		adapter.ReloadUdpServer(udpServer)
	}
}

// ReloadMCP stops global MCP when disabled, or restarts or starts MCP when enabled.
func (app *App) ReloadMCP() error {
	if !viper.GetBool("mcp.global.enabled") {
		// When disabled, stop without restarting to avoid relying on Start checks or timing.
		if err := mcp.GetGlobalMCPManager().Stop(); err != nil {
			return err
		}
		return nil
	}
	mgr := mcp.GetMCPManager()
	if mgr.IsStarted() {
		if err := mgr.RestartManager("global"); err != nil {
			return err
		}
		return nil
	}
	if err := mcp.StartMCPManagers(); err != nil {
		return err
	}
	return nil
}

// OnNewConnection handles new connections from every protocol.
func (a *App) OnNewConnection(transport types.IConn) {
	deviceID := transport.GetDeviceID()
	transportType := transport.GetTransportType()
	notifyLifecycleOnManager := transportType != types.TransportTypeMqttUdp

	// Check whether the device already has a ChatManager.
	if existingManager, exists := a.chatManagers.Get(deviceID); exists {
		log.Infof("device %s already has a ChatManager; closing the old connection first", deviceID)
		// Close the old ChatManager.
		existingManager.Close()
		a.chatManagers.Remove(deviceID)
	}

	// Create a new ChatManager.
	chatManager, err := chat.NewChatManager(deviceID, transport)
	if err != nil {
		log.Errorf("failed to create ChatManager: %v", err)
		return
	}

	// Store the ChatManager.
	a.chatManagers.Set(deviceID, chatManager)

	if notifyLifecycleOnManager {
		a.DeviceOnline(deviceID)
	}

	log.Infof("created and stored ChatManager for device %s", deviceID)

	// Replay OpenClaw offline messages with retries while the new session initializes.
	go a.replayOpenClawOfflineMessages(deviceID)

	// Start the ChatManager.
	go func() {
		defer func() {
			// Remove the ChatManager from the map when it exits.
			if storedManager, exists := a.chatManagers.Get(deviceID); exists && storedManager == chatManager {
				a.chatManagers.Remove(deviceID)
				log.Infof("removed ChatManager for device %s from the map", deviceID)
				if notifyLifecycleOnManager {
					a.DeviceOffline(deviceID)
				}
			}
		}()

		if err := chatManager.Start(); err != nil {
			log.Errorf("failed to start ChatManager: %v", err)
		}
	}()
}

func (a *App) onMqttTransportReady(deviceID string) {
	chatManager, exists := a.GetChatManager(deviceID)
	if !exists || chatManager == nil {
		return
	}
	chatManager.HandleMqttTransportReady()
	chatManager.WarmupMcp()
}

// OnOpenClawResponse delivers real-time OpenClaw responses.
func (a *App) OnOpenClawResponse(event openclaw.ResponseDelivery) bool {
	deviceID := strings.TrimSpace(event.DeviceID)
	if deviceID == "" {
		return false
	}
	chatManager, exists := a.GetChatManager(deviceID)
	if !exists || chatManager == nil {
		return false
	}
	if err := chatManager.InjectOpenClawResponse(event); err != nil {
		log.Warnf(
			"failed to inject real-time OpenClaw message, device=%s correlation_id=%s start=%v end=%v err=%v",
			deviceID,
			strings.TrimSpace(event.CorrelationID),
			event.IsStart,
			event.IsEnd,
			err,
		)
		return false
	}
	return true
}

func (a *App) replayOpenClawOfflineMessages(deviceID string) {
	manager := openclaw.GetManager()
	const maxRetry = 10
	for i := 0; i < maxRetry; i++ {
		time.Sleep(1 * time.Second)
		delivered, remaining := manager.ReplayOfflineMessages(deviceID, func(msg openclaw.OfflineMessage) error {
			chatManager, exists := a.GetChatManager(deviceID)
			if !exists || chatManager == nil {
				return fmt.Errorf("chat manager not ready")
			}
			if strings.TrimSpace(msg.Text) == "" {
				return nil
			}
			return chatManager.InjectMessage(msg.Text, true, false)
		})
		if delivered > 0 {
			log.Infof("replayed OpenClaw offline messages, device=%s delivered=%d remaining=%d", deviceID, delivered, remaining)
		}
		if remaining == 0 {
			return
		}
	}
}

// GetChatManager returns the ChatManager for a device.
func (a *App) GetChatManager(deviceID string) (*chat.ChatManager, bool) {
	return a.chatManagers.Get(deviceID)
}

// CloseChatManager closes the ChatManager for a device.
func (a *App) CloseChatManager(deviceID string) bool {
	if manager, exists := a.chatManagers.Get(deviceID); exists {
		manager.Close()
		a.chatManagers.Remove(deviceID)
		log.Infof("closed and removed ChatManager for device %s", deviceID)
		return true
	}
	return false
}

// GetAllChatManagers returns a copy of all ChatManager entries.
func (a *App) GetAllChatManagers() map[string]*chat.ChatManager {
	// Return a copy to avoid concurrent access issues.
	managers := make(map[string]*chat.ChatManager)
	for tuple := range a.chatManagers.IterBuffered() {
		managers[tuple.Key] = tuple.Val
	}
	return managers
}

// GetChatManagerCount returns the number of active ChatManager instances.
func (a *App) GetChatManagerCount() int {
	return a.chatManagers.Count()
}

// CloseAllChatManagers closes all ChatManager instances.
func (a *App) CloseAllChatManagers() {
	for tuple := range a.chatManagers.IterBuffered() {
		tuple.Val.Close()
		log.Infof("closed ChatManager for device %s", tuple.Key)
	}

	// Clear the map.
	a.chatManagers.Clear()
	log.Info("all ChatManager instances closed")
}

// registerChatMCPTools registers local chat-related MCP tools.
func (s *App) registerChatMCPTools() {
	// Call the registration function in the chat package.
	chat.RegisterChatMCPTools()

	log.Info("registered local chat-related MCP tools")
}

func (s *App) DeviceOnline(deviceID string) {
	eventData := map[string]interface{}{
		"device_id": deviceID,
	}
	providerType := viper.GetString("config_provider.type")
	provider, err := user_config.GetProvider(providerType)
	if err != nil {
		log.Errorf("GetProvider err: %+v", err)
		return
	}
	provider.NotifyDeviceEvent(context.Background(), config_types.EventDeviceOnline, eventData)
}

func (s *App) DeviceOffline(deviceID string) {
	eventData := map[string]interface{}{
		"device_id": deviceID,
	}
	providerType := viper.GetString("config_provider.type")
	provider, err := user_config.GetProvider(providerType)
	if err != nil {
		log.Errorf("GetProvider err: %+v", err)
		return
	}
	provider.NotifyDeviceEvent(context.Background(), config_types.EventDeviceOffline, eventData)
}

func (a *App) registerHandler() {
	providerType := viper.GetString("config_provider.type")
	log.Infof("registerHandler: config_provider.type=%s", providerType)
	provider, err := user_config.GetProvider(providerType)
	if err != nil {
		log.Errorf("GetProvider err: %+v", err)
		return
	}
	provider.RegisterMessageEventHandler(context.Background(), config_types.EventHandleMessageInject, a.HandleInjectMsg)
	log.Infof("registerHandler: registered paths=[%s]", config_types.EventHandleMessageInject)
}

// HandleInjectMsg injects a message into a client.
func (a *App) HandleInjectMsg(ctx context.Context, eventType string, eventData map[string]interface{}) (string, error) {
	type InjectMsg struct {
		SkipLlm    bool   `json:"skip_llm"`
		AutoListen *bool  `json:"auto_listen"`
		DeviceId   string `json:"device_id"`
		Message    string `json:"message"`
	}
	bodyBytes, _ := json.Marshal(eventData)
	var msg InjectMsg
	err := json.Unmarshal(bodyBytes, &msg)
	if err != nil {
		log.Errorf("HandleInjectMsg error: %+v", err)
		return "", fmt.Errorf("HandleInjectMsg error")
	}

	// Validate required parameters.
	if msg.DeviceId == "" {
		log.Errorf("HandleInjectMsg: device_id is required")
		return "", fmt.Errorf("device_id is required")
	}
	if msg.Message == "" {
		log.Errorf("HandleInjectMsg: message is required")
		return "", fmt.Errorf("message is required")
	}

	// Get the ChatManager for the specified device.
	chatManager, exists := a.GetChatManager(msg.DeviceId)
	if !exists {
		log.Errorf("HandleInjectMsg: device %s not found or offline", msg.DeviceId)
		return "", fmt.Errorf("device %s not found or offline", msg.DeviceId)
	}

	autoListen := true
	if msg.AutoListen != nil {
		autoListen = *msg.AutoListen
	}

	log.Debugf("HandleInjectMsg: injecting message to device %s, skip_llm: %v, auto_listen: %v, message: %s",
		msg.DeviceId, msg.SkipLlm, autoListen, msg.Message)

	// Inject the message through the public ChatManager method.
	err = chatManager.InjectMessage(msg.Message, msg.SkipLlm, autoListen)
	if err != nil {
		log.Errorf("HandleInjectMsg: failed to inject message to device %s: %v", msg.DeviceId, err)
		return "", fmt.Errorf("failed to inject message: %v", err)
	}

	return "message injected successfully", nil
}
