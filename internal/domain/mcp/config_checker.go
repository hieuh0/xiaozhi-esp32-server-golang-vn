package mcp

import (
	"fmt"
	"net/url"
	"strings"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/spf13/viper"
)

// CheckMCPConfig checks MCP configuration and reports potential problems
func CheckMCPConfig() {
	log.Info("=== MCP configuration check ===")

	//Check global enablement status
	globalEnabled := viper.GetBool("mcp.global.enabled")
	log.Infof("Global MCP enabled status: %v", globalEnabled)

	if !globalEnabled {
		log.Info("Global MCP disabled, configuration check completed")
		return
	}

	//Check reconnection configuration
	reconnectInterval := viper.GetInt("mcp.global.reconnect_interval")
	maxAttempts := viper.GetInt("mcp.global.max_reconnect_attempts")
	log.Infof("Reconnection configuration: interval=%d seconds, maximum number of attempts=%d", reconnectInterval, maxAttempts)

	//Check server configuration
	var serverConfigs []MCPServerConfig
	if err := viper.UnmarshalKey("mcp.global.servers", &serverConfigs); err != nil {
		log.Errorf("❌ Failed to parse MCP server configuration: %v", err)
		return
	}

	if len(serverConfigs) == 0 {
		log.Warn("⚠️ No MCP server configured")
		return
	}

	log.Infof("A total of %d MCP servers are configured:", len(serverConfigs))

	enabledCount := 0
	problemCount := 0

	for i, config := range serverConfigs {
		status := "✅"
		issues := []string{}

		//Check name
		if config.Name == "" {
			status = "❌"
			issues = append(issues, "name is empty")
			problemCount++
		}

		transportType, endpoint, err := endpointForConfig(config)
		if err != nil {
			status = "❌"
			issues = append(issues, err.Error())
			problemCount++
		} else {
			if _, parseErr := url.ParseRequestURI(endpoint); parseErr != nil {
				status = "❌"
				issues = append(issues, "invalid URL format")
				problemCount++
			}
			if transportType == "sse" && !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
				status = "⚠️"
				issues = append(issues, "SSE URL format may be invalid")
			}
		}

		//Check enabled status
		if config.Enabled {
			enabledCount++
		}

		//Output inspection results
		issueStr := ""
		if len(issues) > 0 {
			issueStr = fmt.Sprintf(" - issues: %s", strings.Join(issues, ", "))
		}

		log.Infof("[%d] %s %s (URL: %s, Enable: %v)%s",
			i+1, status, config.Name, endpointForLog(config), config.Enabled, issueStr)
	}

	//Summary
	log.Infof("Configuration check completed: %d servers are enabled, %d have problems", enabledCount, problemCount)

	if problemCount > 0 {
		log.Warn("⚠️ Found a configuration problem, please check the above errors and fix them")
	}

	log.Info("=== MCP configuration check completed ===")
}

func endpointForLog(config MCPServerConfig) string {
	_, endpoint, err := endpointForConfig(config)
	if err != nil {
		if strings.TrimSpace(config.Url) != "" {
			return config.Url
		}
		return config.SSEUrl
	}
	return endpoint
}
