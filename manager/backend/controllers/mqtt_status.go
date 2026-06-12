package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"xiaozhi/manager/backend/models"
)

type mqttServerStatusResponse struct {
	Running          bool    `json:"running"`
	ListenHost       string  `json:"listen_host"`
	ListenPort       int     `json:"listen_port"`
	ConnectedClients *int    `json:"connected_clients"` // nil when main app unavailable
}

type mqttClientStatusResponse struct {
	Configured bool    `json:"configured"`
	Broker     string  `json:"broker"`
	Type       string  `json:"type"`
	Port       int     `json:"port"`
	Enable     bool    `json:"enable"`
	Connected  *bool   `json:"connected"`   // nil when main app unavailable
	BrokerURL  *string `json:"broker_url"`  // runtime broker URL from main app
}

type mqttInternalStatus struct {
	Broker struct {
		Running          bool `json:"running"`
		ConnectedClients int  `json:"connected_clients"`
	} `json:"broker"`
	Client struct {
		Connected bool   `json:"connected"`
		BrokerURL string `json:"broker_url"`
	} `json:"client"`
}

func (ac *AdminController) fetchInternalMqttStatus() (*mqttInternalStatus, error) {
	url := ac.AppServerURL + "/internal/mqtt/status"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+ac.InternalAuthToken)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var status mqttInternalStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

// GetMqttServerStatus probes the configured MQTT broker TCP port and enriches with runtime data.
func (ac *AdminController) GetMqttServerStatus(c *gin.Context) {
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "mqtt_server").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	resp := mqttServerStatusResponse{}
	if len(configs) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(configs[0].JsonData), &data); err == nil {
			resp.ListenHost, _ = data["listen_host"].(string)
			if p, ok := data["listen_port"].(float64); ok {
				resp.ListenPort = int(p)
			}
		}
	}
	if resp.ListenPort == 0 {
		resp.ListenPort = 1883
	}

	probeHost := resp.ListenHost
	if probeHost == "" || probeHost == "0.0.0.0" || probeHost == "::" {
		probeHost = "127.0.0.1"
	}

	addr := fmt.Sprintf("%s:%d", probeHost, resp.ListenPort)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err == nil {
		conn.Close()
		resp.Running = true
	}

	if internal, err := ac.fetchInternalMqttStatus(); err == nil {
		n := internal.Broker.ConnectedClients
		resp.ConnectedClients = &n
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// GetMqttClientStatus reads mqtt config from DB and enriches with runtime connection state.
func (ac *AdminController) GetMqttClientStatus(c *gin.Context) {
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "mqtt").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	resp := mqttClientStatusResponse{}
	if len(configs) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(configs[0].JsonData), &data); err == nil {
			resp.Broker, _ = data["broker"].(string)
			resp.Type, _ = data["type"].(string)
			if p, ok := data["port"].(float64); ok {
				resp.Port = int(p)
			}
			resp.Enable, _ = data["enable"].(bool)
		}
		resp.Configured = resp.Broker != ""
	}

	if internal, err := ac.fetchInternalMqttStatus(); err == nil {
		connected := internal.Client.Connected
		brokerURL := internal.Client.BrokerURL
		resp.Connected = &connected
		resp.BrokerURL = &brokerURL
		// adapter running from yaml config (not saved to DB yet) — still considered configured
		if !resp.Configured && brokerURL != "" {
			resp.Configured = true
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
