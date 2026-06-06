package chat

import (
	"context"
	"encoding/json"

	. "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/domain/mcp"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/spf13/viper"
)

var (
	ensureDeviceMcpRuntime = func(deviceID string, mcpTransport *McpTransport) error {
		return mcp.EnsureDeviceIotOverMcp(deviceID, mcpTransport)
	}
	closeDeviceMcpRuntime = func(deviceID string, mcpTransport *McpTransport) {
		mcp.CloseDeviceIotOverMcp(deviceID, mcpTransport)
	}
	shouldScheduleDeviceMcpRuntimeInit = func(deviceID string, mcpTransport *McpTransport) bool {
		return mcp.ShouldScheduleDeviceIotOverMcp(deviceID, mcpTransport)
	}
)

type McpTransport struct {
	Client          *ClientState
	ServerTransport *ServerTransport
}

func (c *McpTransport) SendMcpMsg(payload []byte) error {
	// If it is an initialize request, inject vision
	var request transport.JSONRPCRequest
	err := json.Unmarshal(payload, &request)
	if err == nil {
		if request.Method == "initialize" {
			if origInitParams, ok := request.Params.(map[string]interface{}); ok {
				b, err := json.Marshal(origInitParams)
				if err != nil {
					return err
				}

				var initParams mcp.InitializeParams
				err = json.Unmarshal(b, &initParams)
				if err != nil {
					return err
				}
				initParams.Capabilities["vision"] = mcp.Vision{
					Url:   viper.GetString("vision.vision_url"),
					Token: "1234567890",
				}
				request.Params = initParams
			}
			payload, _ = json.Marshal(request)
		}
	}

	return c.ServerTransport.SendMcpMsg(payload)
}

func (c *McpTransport) RecvMcpMsg(ctx context.Context, timeOut int) ([]byte, error) {
	return c.ServerTransport.RecvMcpMsg(ctx, timeOut)
}

func (c *McpTransport) HandleMcpMessage(payload []byte) error {
	if c == nil || c.ServerTransport == nil {
		return nil
	}
	return c.ServerTransport.HandleMcpMessage(payload)
}

func (c *McpTransport) GetMcpTransportType() string {
	if c == nil || c.ServerTransport == nil {
		return ""
	}
	return c.ServerTransport.GetTransportType()
}

func initMcp(deviceID string, mcpTransport *McpTransport) error {
	if err := ensureDeviceMcpRuntime(deviceID, mcpTransport); err != nil {
		log.Errorf("Ensure IotOverMcp client failed: %v", err)
		return err
	}
	return nil
}
