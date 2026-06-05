package sqlite

import (
	"fmt"
	"path/filepath"
	"xiaozhi/manager/backend/config"
)

// Config SQLite configuration
type Config struct {
	// FilePath database file path (e.g. ./data/xiaozhi.db or /path/to/xiaozhi.db)
	FilePath string `json:"file_path"`

	// connection pool config (SQLite typically works fine with a single connection)
	MaxIdleConns    int `json:"max_idle_conns"`
	MaxOpenConns    int `json:"max_open_conns"`
	ConnMaxLifetime int `json:"conn_max_lifetime"`
}

// NewConfigFromDatabase creates a SQLite config from a database config
func NewConfigFromDatabase(cfg *config.SQLiteConfig) *Config {
	filePath := cfg.FilePath
	if filePath == "" {
		filePath = "./data/xiaozhi.db"
	}

	return &Config{
		FilePath:        filePath,
		MaxIdleConns:    1,
		MaxOpenConns:    1,
		ConnMaxLifetime: 3600,
	}
}

// DSN generates the data source name (GORM SQLite format)
func (c *Config) DSN() string {
	// use file: prefix to support extended options
	return "file:" + c.FilePath + "?_foreign_keys=on&_journal_mode=WAL"
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.FilePath == "" {
		return fmt.Errorf("SQLite file path is required")
	}

	// check file extension
	ext := filepath.Ext(c.FilePath)
	if ext != ".db" && ext != ".sqlite" && ext != ".sqlite3" {
		return fmt.Errorf("SQLite file must have .db, .sqlite or .sqlite3 extension")
	}

	return nil
}

// ValidateConfig validates a SQLite config
func ValidateConfig(cfg *config.SQLiteConfig) error {
	if cfg == nil {
		return fmt.Errorf("SQLite config is required")
	}
	if cfg.FilePath == "" {
		return fmt.Errorf("SQLite file path is required")
	}

	// check file extension
	ext := filepath.Ext(cfg.FilePath)
	if ext != ".db" && ext != ".sqlite" && ext != ".sqlite3" {
		return fmt.Errorf("SQLite file must have .db, .sqlite or .sqlite3 extension")
	}

	return nil
}
