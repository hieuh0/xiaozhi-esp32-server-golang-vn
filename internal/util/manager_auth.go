package util

import (
	"strings"

	"github.com/spf13/viper"
)

const DefaultManagerAuthToken = "xiaozhi_admin_secret_key"
const DefaultManagerEndpointAuthToken = "xiaozhi_mcp_openclaw_secret_key"

// GetManagerAuthToken returns the internal auth token shared between the main program and the console.
// Priority:
// 1. manager.auth_token
// 2. Default value (must be consistent on both ends)
func GetManagerAuthToken() string {
	if token := strings.TrimSpace(viper.GetString("manager.auth_token")); token != "" {
		return token
	}
	return DefaultManagerAuthToken
}

// GetManagerEndpointAuthToken returns the signing/verification token for MCP/OpenClaw endpoint JWTs.
// Priority:
// 1. manager.endpoint_auth_token
// 2. Default value (must be consistent with the console)
func GetManagerEndpointAuthToken() string {
	if token := strings.TrimSpace(viper.GetString("manager.endpoint_auth_token")); token != "" {
		return token
	}
	return DefaultManagerEndpointAuthToken
}
