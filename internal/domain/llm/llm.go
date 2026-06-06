package llm

import (
	"context"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
)

// ConvertMCPToolsToEinoTools Convert MCP tools to Eino ToolInfo format
func ConvertMCPToolsToEinoTools(ctx context.Context, mcpTools map[string]interface{}) ([]*schema.ToolInfo, error) {
	var einoTools []*schema.ToolInfo

	for toolName, mcpTool := range mcpTools {
		//Try to get tool information
		if invokableTool, ok := mcpTool.(interface {
			Info(context.Context) (*schema.ToolInfo, error)
		}); ok {
			toolInfo, err := invokableTool.Info(ctx)
			if err != nil {
				log.Errorf("Failed to obtain tool %s information: %v", toolName, err)
				continue
			}
			einoTools = append(einoTools, toolInfo)
		} else {
			log.Warnf("Tool %s does not support Info interface, skip conversion", toolName)
		}
	}

	log.Infof("Successfully converted %d MCP tools to Eino tools", len(einoTools))
	return einoTools, nil
}
