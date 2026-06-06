package websocket

import (
	"net/http"
	"strings"
	"xiaozhi-esp32-server-golang/internal/domain/mcp"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/golang-jwt/jwt/v4"
)

// MCPClaims JWT claims structure
type MCPClaims struct {
	UserID     uint   `json:"userId"`
	AgentID    string `json:"agentId"`
	EndpointID string `json:"endpointId"`
	Purpose    string `json:"purpose"`
	jwt.RegisteredClaims
}

// handleMCPWebSocket handles MCP WebSocket connections
func (s *WebSocketServer) handleMCPWebSocket(w http.ResponseWriter, r *http.Request) {
	var agentId string

	// First try to get token from URL parameters
	token := r.URL.Query().Get("token")
	if token != "" {
		// Parse agent ID from token
		claims, err := s.parseMCPToken(token)
		if err != nil {
			log.Warnf("Failed to parse token: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		log.Infof("Token parsed successfully: %v", claims)

		agentId = claims.AgentID
	} else {
		log.Errorf("Missing token")
		return
	}

	log.Infof("Received MCP server WebSocket connection request, Agent ID: %s", agentId)

	// Upgrade WebSocket connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	mcpClientSession := mcp.GetDeviceMcpClient(agentId)
	if mcpClientSession == nil {
		mcpClientSession = mcp.NewDeviceMCPSession(agentId)
		mcp.AddDeviceMcpClient(agentId, mcpClientSession)
	}

	// Create MCP client
	mcpClient := mcp.NewWsEndPointMcpClient(mcpClientSession.Ctx, agentId, conn)
	if mcpClient == nil {
		log.Errorf("Failed to create MCP client")
		conn.Close()
		return
	}
	mcpClientSession.AddWsEndPointMcp(mcpClient)

	// Clean up ws endpoint mcp client when mcp server disconnects
	go func() {
		<-mcpClient.Ctx.Done()
		log.Infof("MCP connection for server %s disconnected", mcpClient.GetServerName())
	}()

	log.Infof("MCP connection for server %s established", mcpClient.GetServerName())
}

// parseMCPToken parses MCP JWT token
func (s *WebSocketServer) parseMCPToken(tokenString string) (*MCPClaims, error) {
	// Remove "Bearer " prefix
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// Use the same key as when generating the token
	jwtSecret := []byte(util.GetManagerEndpointAuthToken())

	token, err := jwt.ParseWithClaims(tokenString, &MCPClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*MCPClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

// handleMCPAPI handles MCP REST API requests
func (s *WebSocketServer) handleMCPAPI(w http.ResponseWriter, r *http.Request) {
	// Extract deviceId from URL path
	// URL format: /xiaozhi/api/mcp/tools/{deviceId}
	path := strings.TrimPrefix(r.URL.Path, "/xiaozhi/api/mcp/tools/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "Missing device ID parameter", http.StatusBadRequest)
		return
	}

	deviceID := strings.TrimSuffix(path, "/")
	if deviceID == "" {
		http.Error(w, "Device ID cannot be empty", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetDeviceTools(w, r, deviceID)
	default:
		http.Error(w, "Unsupported HTTP method", http.StatusMethodNotAllowed)
	}
}
