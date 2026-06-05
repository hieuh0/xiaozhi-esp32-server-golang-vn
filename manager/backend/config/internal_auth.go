package config

import "strings"

const DefaultInternalAuthToken = "xiaozhi_admin_secret_key"
const DefaultEndpointAuthToken = "xiaozhi_mcp_openclaw_secret_key"

// ResolveInternalAuthToken resolves the internal service token for the console.
// Priority:
// 1. internal_auth_token from config file
// 2. default value (consistent with main program)
func ResolveInternalAuthToken(cfg *Config) string {
	if cfg != nil {
		if token := strings.TrimSpace(cfg.InternalAuthToken); token != "" {
			return token
		}
	}
	return DefaultInternalAuthToken
}

// ResolveEndpointAuthToken resolves the signing token for MCP/OpenClaw endpoint JWTs.
// Priority:
// 1. endpoint_auth_token from config file
// 2. default value (consistent with main program)
func ResolveEndpointAuthToken(cfg *Config) string {
	if cfg != nil {
		if token := strings.TrimSpace(cfg.EndpointAuthToken); token != "" {
			return token
		}
	}
	return DefaultEndpointAuthToken
}
