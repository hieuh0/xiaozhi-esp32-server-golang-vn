package mqtt_server

import (
	"fmt"
	"strings"
	"time"

	mqttServer "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"

	client "xiaozhi-esp32-server-golang/internal/data/msg"
	log "xiaozhi-esp32-server-golang/logger"
)

// DeviceHook enforces device permissions and automatic subscriptions.
// Regular users may publish only to the designated topic and are automatically
// subscribed to /p2p/device_sub/{mac} when connected.
type DeviceHook struct {
	mqttServer.HookBase
	server           *mqttServer.Server
	publishLifecycle func(event client.MqttLifecycleEvent) error
}

func (h *DeviceHook) ID() string {
	return "custom-device-hook"
}

func (h *DeviceHook) Provides(b byte) bool {
	return b == mqttServer.OnDisconnect || b == mqttServer.OnACLCheck || b == mqttServer.OnSessionEstablished || b == mqttServer.OnSubscribe || b == mqttServer.OnPublish
}

// OnACLCheck controls publish and subscribe permissions.
func (h *DeviceHook) OnACLCheck(cl *mqttServer.Client, topic string, write bool) bool {
	isAdmin := isAdminUser(cl)

	if isAdmin {
		return true // Administrators are unrestricted.
	}

	if write {
		// Regular users may publish only to "device-server".
		if topic == client.MDeviceMockPubTopicPrefix {
			return true
		}
		log.Warnf("Regular user publish denied for topic %s", topic)
		return false
	}

	mac := parseMacFromClientId(cl.ID)
	if mac == "" {
		log.Warnf("Regular user subscription denied for %s: cannot parse MAC from client ID, clientID=%s", topic, cl.ID)
		return false
	}

	allowedTopic := deviceSubTopic(mac)
	if topic == allowedTopic {
		return true
	}

	log.Warnf("Regular user subscription denied for %s: only own topic %s is allowed", topic, allowedTopic)
	return false
}

func (h *DeviceHook) OnConnect(cl *mqttServer.Client, pk packets.Packet) error {
	isAdmin := isAdminUser(cl)
	if isAdmin {
		return nil
	}
	pk.Connect.Clean = true
	return nil
}

func (h *DeviceHook) OnDisconnect(cl *mqttServer.Client, err error, ok bool) {
	if cl == nil {
		log.Warnf("OnDisconnect: client is nil, err=%v, ok=%v", err, ok)
		return
	}
	isAdmin := isAdminUser(cl)
	mac := parseMacFromClientId(cl.ID)
	deviceID := deviceIDFromClientId(cl.ID)
	takenOver := cl.IsTakenOver()

	log.Infof("OnDisconnect: clientID=%s, deviceID=%s, mac=%s, ok=%v, err=%v, takenOver=%v, isAdmin=%v",
		cl.ID, deviceID, mac, ok, err, takenOver, isAdmin)

	if isAdmin {
		return
	}
	if takenOver {
		log.Infof("Client %s was taken over by a new connection with the same ID; skipping unsubscribe and offline lifecycle event", cl.ID)
		return
	}
	if mac == "" {
		log.Infof("OnDisconnect: cannot parse MAC address from client ID, clientID=%s, err=%v, ok=%v", cl.ID, err, ok)
		return
	}

	log.Infof("OnDisconnect: publishing offline lifecycle event, clientID=%s, deviceID=%s", cl.ID, deviceID)
	h.publishLifecycleEvent(cl.ID, client.MqttLifecycleStateOffline)
	topic := deviceSubTopic(mac)

	action := h.server.Topics.Unsubscribe(topic, cl.ID)
	log.Infof("OnDisconnect: unsubscribed client %s from topic %s, action=%v", cl.ID, topic, action)

	return
}

// OnSessionEstablished subscribes the device after the connection is established.
func (h *DeviceHook) OnSessionEstablished(cl *mqttServer.Client, pk packets.Packet) {
	isAdmin := isAdminUser(cl)
	mac := parseMacFromClientId(cl.ID)
	deviceID := deviceIDFromClientId(cl.ID)
	if isAdmin {
		return // Administrators are unrestricted.
	}
	if mac == "" {
		log.Info("Warning: cannot parse MAC address from client ID:", cl.ID)
		return
	}
	log.Infof("OnSessionEstablished: clientID=%s, deviceID=%s, mac=%s, clean=%v", cl.ID, deviceID, mac, pk.Connect.Clean)
	h.publishLifecycleEvent(cl.ID, client.MqttLifecycleStateOnline)

	topic := deviceSubTopic(mac)

	// Subscribe through the server API instead of injecting a packet.
	clientID := cl.ID
	exists := h.server.Topics.Subscribe(clientID, packets.Subscription{
		Filter: topic,
		Qos:    0,
	})

	log.Infof("Subscribed client %s to topic %s, exists: %v", clientID, topic, exists)
}

// OnSubscribe logs subscribe packets.
func (h *DeviceHook) OnSubscribe(cl *mqttServer.Client, pk packets.Packet) packets.Packet {
	log.Info("=== Subscribe packet received ===")
	log.Infof("Client ID: %s", cl.ID)
	log.Infof("Packet type: %v", pk.FixedHeader.Type)
	log.Infof("Packet ID: %d", pk.PacketID)

	if len(pk.Filters) > 0 {
		log.Info("Subscriptions:")
		for i, sub := range pk.Filters {
			log.Infof("  %d. Topic: %s, QoS: %d", i+1, sub.Filter, sub.Qos)
		}
	}

	log.Info("==================")
	return pk
}

// OnPublish logs publish packets.
func (h *DeviceHook) OnPublish(cl *mqttServer.Client, pk packets.Packet) (packets.Packet, error) {
	if cl == nil {
		return pk, nil
	}

	log.Info("=== Publish packet received ===")
	log.Infof("Client ID: %s", cl.ID)
	log.Infof("Packet type: %v", pk.FixedHeader.Type)
	log.Infof("Packet ID: %d", pk.PacketID)
	log.Infof("Topic: %s", pk.TopicName)

	if isAdminUser(cl) {
		return pk, nil
	}

	if len(pk.Payload) > 0 {
		if len(pk.Payload) > 100 {
			// Log only the first 100 bytes of long payloads.
			log.Infof("Payload (first 100 bytes): %s...", pk.Payload[:100])
		} else {
			log.Infof("Payload: %s", pk.Payload)
		}
	} else {
		log.Info("Payload: <empty>")
	}

	// Extract the MAC address from the client.
	mac := parseMacFromClientId(cl.ID)
	if mac == "" {
		log.Info("Warning: cannot parse MAC address from client ID:", cl.ID)
		return pk, nil
	}
	forwardTopic := fmt.Sprintf("%s%s", client.MDevicePubTopicPrefix, mac)

	pk.TopicName = forwardTopic

	log.Info("==================")
	return pk, nil
}

// isAdminUser reports whether the client is an administrator.
func isAdminUser(cl *mqttServer.Client) bool {
	if cl == nil {
		return false
	}
	return string(cl.Properties.Username) == configuredAdminUsername()
}

// parseMacFromClientId extracts the MAC address from a client ID.
func parseMacFromClientId(clientId string) string {
	parts := strings.Split(clientId, "@@@")
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}

func deviceIDFromClientId(clientID string) string {
	mac := parseMacFromClientId(clientID)
	if mac == "" {
		return ""
	}
	return strings.ReplaceAll(mac, "_", ":")
}

func (h *DeviceHook) publishLifecycleEvent(clientID string, state string) {
	if h == nil || h.publishLifecycle == nil {
		return
	}
	deviceID := deviceIDFromClientId(clientID)
	if deviceID == "" {
		log.Warnf("Skipping MQTT lifecycle event: cannot parse deviceID, clientID=%s, state=%s", clientID, state)
		return
	}
	event := client.MqttLifecycleEvent{
		Type:     client.MqttLifecycleType,
		DeviceID: deviceID,
		State:    state,
		ClientID: clientID,
		Ts:       time.Now().UnixMilli(),
	}
	log.Infof("Publishing MQTT lifecycle event: device=%s, clientID=%s, state=%s, ts=%d", deviceID, clientID, state, event.Ts)
	if err := h.publishLifecycle(event); err != nil {
		log.Warnf("Failed to publish MQTT lifecycle event: device=%s state=%s err=%v", deviceID, state, err)
	}
}

func deviceSubTopic(mac string) string {
	return fmt.Sprintf("%s%s", client.MDeviceSubTopicPrefix, mac)
}

// StartPeriodicSubscriptionPrinter periodically logs client subscriptions.
func (h *DeviceHook) StartPeriodicSubscriptionPrinter(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			h.PrintAllClientSubscriptions()
		}
	}()
}

// PrintAllClientSubscriptions logs all client subscriptions.
func (h *DeviceHook) PrintAllClientSubscriptions() {
	log.Info("=== Client subscriptions ===")
	clients := h.server.Clients.GetAll()
	if len(clients) == 0 {
		log.Info("No clients are connected")
		return
	}

	for clientID, _ := range clients {
		log.Infof("Topics subscribed by client %s: ", clientID)

		// Get subscribers for all single-level topics, then select this client.
		allSubs := h.server.Topics.Subscribers("+")
		foundTopics := false

		// Check the client's subscription.
		if subs, ok := allSubs.Subscriptions[clientID]; ok {
			log.Infof("  - %s (QoS: %d)", subs.Filter, subs.Qos)
			foundTopics = true
		}

		// Check additional wildcard subscriptions.
		allSubs = h.server.Topics.Subscribers("#")
		if subs, ok := allSubs.Subscriptions[clientID]; ok {
			log.Infof("  - %s (QoS: %d)", subs.Filter, subs.Qos)
			foundTopics = true
		}

		// Check the device-specific topic.
		mac := parseMacFromClientId(clientID)
		if mac != "" {
			topic := deviceSubTopic(mac)
			topicSubs := h.server.Topics.Subscribers(topic)
			if subs, ok := topicSubs.Subscriptions[clientID]; ok {
				log.Infof("  - %s (QoS: %d)", subs.Filter, subs.Qos)
				foundTopics = true
			}
		}

		if !foundTopics {
			log.Info("  No subscriptions found or available")
		}
	}
	log.Info("=====================")
}
