package util

import (
	"os"

	"github.com/spf13/viper"
)

// GetBackendURL returns the backend URL, preferring the environment over configuration.
func GetBackendURL() string {
	// Prefer the environment variable.
	if backendURL := os.Getenv("BACKEND_URL"); backendURL != "" {
		return backendURL
	}
	// Fall back to configuration.
	return viper.GetString("manager.backend_url")
}
