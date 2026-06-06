package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the server configuration
type Config struct {
	Server struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"server"`
	MQTT struct {
		Broker   string `json:"broker"`
		ClientID string `json:"client_id"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"mqtt"`
	// Wake word configuration
	WakeupWords    []string `json:"wakeup_words"`
	EnableGreeting bool     `json:"enable_greeting"`
}

// ServerAddress returns the server address
func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// LoadConfig loads configuration from a file
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves configuration to a file
func (c *Config) SaveConfig(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
