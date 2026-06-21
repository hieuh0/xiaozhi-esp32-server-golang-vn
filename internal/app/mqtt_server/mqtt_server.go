package mqtt_server

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	mqttServer "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/spf13/viper"

	msg "xiaozhi-esp32-server-golang/internal/data/msg"
	log "xiaozhi-esp32-server-golang/logger"
)

var (
	currentServer *mqttServer.Server
	serverMu      sync.Mutex
	brokerReady   chan struct{} // closed when the port is confirmed bound
)

// WaitBrokerReady returns a channel that is closed once the broker has
// successfully bound its port. Returns nil if StartMqttServer has not been called.
func WaitBrokerReady() <-chan struct{} {
	serverMu.Lock()
	defer serverMu.Unlock()
	return brokerReady
}

// StartMqttServer starts the MQTT server and may be called again after
// StopMqttServer to apply configuration changes.
func StartMqttServer() error {
	serverMu.Lock()
	defer serverMu.Unlock()
	if currentServer != nil {
		return errors.New("mqtt_server is already running; call StopMqttServer first")
	}
	srv := mqttServer.New(&mqttServer.Options{
		InlineClient: true,
	})

	if err := srv.AddHook(&AuthHook{}, nil); err != nil {
		log.Errorf("Failed to add AuthHook: %v", err)
		return err
	}
	deviceHook := &DeviceHook{
		server: srv,
		publishLifecycle: func(event msg.MqttLifecycleEvent) error {
			payload, err := json.Marshal(event)
			if err != nil {
				return err
			}
			return srv.Publish(msg.MDeviceLifecycleTopic, payload, false, 0)
		},
	}
	if err := srv.AddHook(deviceHook, nil); err != nil {
		log.Errorf("Failed to add DeviceHook: %v", err)
		return err
	}

	if viper.GetBool("mqtt_server.tls.enable") {
		pemFile := viper.GetString("mqtt_server.tls.pem")
		keyFile := viper.GetString("mqtt_server.tls.key")
		cert, err := tls.LoadX509KeyPair(pemFile, keyFile)
		if err != nil {
			log.Errorf("Failed to load certificate: %v", err)
			return err
		}
		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
		ssltcp := listeners.NewTCP(listeners.Config{
			ID:        "ssl",
			Address:   fmt.Sprintf(":%d", viper.GetInt("mqtt_server.tls.port")),
			TLSConfig: tlsConfig,
		})
		if err := srv.AddListener(ssltcp); err != nil {
			return err
		}
	}

	host := viper.GetString("mqtt_server.listen_host")
	port := viper.GetInt("mqtt_server.listen_port")
	if port == 0 {
		return errors.New("invalid mqtt_server.port; check the configuration file")
	}
	address := fmt.Sprintf("%s:%d", host, port)
	tcp := listeners.NewTCP(listeners.Config{Type: "tcp", ID: "t1", Address: address})
	if err := srv.AddListener(tcp); err != nil {
		return err
	}

	ready := make(chan struct{})
	brokerReady = ready
	currentServer = srv

	go func() {
		// Serve starts listener goroutines internally and returns immediately, so
		// currentServer must remain set here.
		if err := srv.Serve(); err != nil {
			log.Warnf("MQTT server Serve exited: %v", err)
		}
	}()

	// mochi-mqtt's Serve() returns nil even when the listener goroutine fails
	// silently. Poll the port to confirm it is actually bound before signalling ready.
	go func() {
		if waitForPort(address, 10*time.Second) {
			log.Infof("MQTT broker confirmed listening on %s", address)
			close(ready)
		} else {
			log.Errorf("MQTT broker failed to bind %s within 10s — check for port conflict", address)
		}
	}()

	return nil
}

// waitForPort polls address with net.DialTimeout until accepting connections or timeout.
func waitForPort(address string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// StopMqttServer stops the current MQTT server so it can be restarted with
// updated configuration.
func StopMqttServer() error {
	log.Infof("enter StopMqttServer ")
	defer log.Infof("exit StopMqttServer ")
	serverMu.Lock()
	defer serverMu.Unlock()
	srv := currentServer
	if srv == nil {
		return nil
	}
	// Keep Close in the same critical section to prevent concurrent stop calls
	// from closing the same server instance more than once.
	if err := srv.Close(); err != nil {
		log.Warnf("StopMqttServer Close: %v", err)
		return err
	}
	currentServer = nil
	brokerReady = nil
	log.Info("MQTT server stopped")
	return nil
}
