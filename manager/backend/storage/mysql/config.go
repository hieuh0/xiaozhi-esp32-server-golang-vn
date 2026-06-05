package mysql

import (
	"fmt"
	"xiaozhi/manager/backend/config"
)

// Config MySQL configuration
type Config struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	MaxIdleConns    int    `json:"max_idle_conns"`
	MaxOpenConns    int    `json:"max_open_conns"`
	ConnMaxLifetime int    `json:"conn_max_lifetime"`
}

// NewConfigFromDatabase creates a MySQL config from a database config
func NewConfigFromDatabase(cfg *config.MySQLConfig) *Config {
	return &Config{
		Host:            cfg.Host,
		Port:            cfg.Port,
		Username:        cfg.Username,
		Password:        cfg.Password,
		Database:        cfg.Database,
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: 3600,
	}
}

// DSN generates the data source name
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Username, c.Password, c.Host, c.Port, c.Database)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("MySQL host is required")
	}
	if c.Port == 0 {
		return fmt.Errorf("MySQL port is required")
	}
	if c.Username == "" {
		return fmt.Errorf("MySQL username is required")
	}
	if c.Database == "" {
		return fmt.Errorf("MySQL database name is required")
	}
	return nil
}

// ValidateConfig validates a MySQL config
func ValidateConfig(cfg *config.MySQLConfig) error {
	if cfg == nil {
		return fmt.Errorf("MySQL config is required")
	}
	if cfg.Host == "" {
		return fmt.Errorf("MySQL host is required")
	}
	if cfg.Port == 0 {
		return fmt.Errorf("MySQL port is required")
	}
	if cfg.Username == "" {
		return fmt.Errorf("MySQL username is required")
	}
	if cfg.Database == "" {
		return fmt.Errorf("MySQL database name is required")
	}
	return nil
}
