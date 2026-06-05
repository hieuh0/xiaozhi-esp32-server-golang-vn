package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"xiaozhi/manager/backend/models"
	mcpmarket "xiaozhi/manager/backend/services/mcp_market"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"gorm.io/gorm"
)

type upsertMCPMarketImportedServiceRequest struct {
	Name         string            `json:"name"`
	Enabled      *bool             `json:"enabled"`
	Transport    string            `json:"transport" binding:"required"`
	URL          string            `json:"url" binding:"required"`
	Headers      map[string]string `json:"headers"`
	AllowedTools []string          `json:"allowed_tools"`
	MarketID     *uint             `json:"market_id"`
	ProviderID   string            `json:"provider_id"`
	ServiceID    string            `json:"service_id"`
	ServiceName  string            `json:"service_name"`
}

type mcpMarketImportedServiceView struct {
	ID           uint              `json:"id"`
	Name         string            `json:"name"`
	Enabled      bool              `json:"enabled"`
	Transport    string            `json:"transport"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers,omitempty"`
	AllowedTools []string          `json:"allowed_tools,omitempty"`
	MarketID     *uint             `json:"market_id,omitempty"`
	ProviderID   string            `json:"provider_id,omitempty"`
	ServiceID    string            `json:"service_id,omitempty"`
	ServiceName  string            `json:"service_name,omitempty"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

type mcpMarketImportedToolView struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (ac *AdminController) GetMCPMarketImportedServices(c *gin.Context) {
	queryText := strings.TrimSpace(c.Query("q"))
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	db := ac.DB.Model(&models.MCPMarketService{})
	if queryText != "" {
		like := "%" + queryText + "%"
		db = db.Where("name LIKE ? OR service_id LIKE ? OR service_name LIKE ? OR url LIKE ?", like, like, like, like)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query imported service count"})
		return
	}

	var rows []models.MCPMarketService
	offset := (page - 1) * pageSize
	if err := db.Order("updated_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query imported service list"})
		return
	}

	items := make([]mcpMarketImportedServiceView, 0, len(rows))
	for _, row := range rows {
		items = append(items, toMCPMarketImportedServiceView(row))
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}})
}

func (ac *AdminController) CreateMCPMarketImportedService(c *gin.Context) {
	var req upsertMCPMarketImportedServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model, err := buildImportedServiceModelFromRequest(req, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.MCPMarketService
	if err := ac.DB.Where("url_hash = ?", model.URLHash).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "an imported service with the same URL already exists"})
		return
	} else if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query imported service"})
		return
	}

	if err := ac.DB.Create(&model).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create imported service"})
		return
	}

	ac.notifySystemConfigChanged()
	c.JSON(http.StatusCreated, gin.H{"data": toMCPMarketImportedServiceView(model)})
}

func (ac *AdminController) UpdateMCPMarketImportedService(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var existing models.MCPMarketService
	if err := ac.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "imported service not found"})
		return
	}

	var req upsertMCPMarketImportedServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := buildImportedServiceModelFromRequest(req, &existing)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var dup models.MCPMarketService
	if err := ac.DB.Where("id != ? AND url_hash = ?", existing.ID, updated.URLHash).First(&dup).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "an imported service with the same URL already exists"})
		return
	} else if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query imported service"})
		return
	}

	updateMap := map[string]interface{}{
		"name":               updated.Name,
		"enabled":            updated.Enabled,
		"transport":          updated.Transport,
		"url":                updated.URL,
		"url_hash":           updated.URLHash,
		"headers_json":       updated.HeadersJSON,
		"allowed_tools_json": updated.AllowedToolsJSON,
		"market_id":          updated.MarketID,
		"provider_id":        updated.ProviderID,
		"service_id":         updated.ServiceID,
		"service_name":       updated.ServiceName,
	}
	if err := ac.DB.Model(&existing).Updates(updateMap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update imported service"})
		return
	}

	if err := ac.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read updated imported service"})
		return
	}

	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"data": toMCPMarketImportedServiceView(existing)})
}

func (ac *AdminController) DeleteMCPMarketImportedService(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var existing models.MCPMarketService
	if err := ac.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "imported service not found"})
		return
	}

	if err := ac.DB.Delete(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete imported service"})
		return
	}

	ac.notifySystemConfigChanged()
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

func (ac *AdminController) GetMCPMarketImportedServiceTools(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var existing models.MCPMarketService
	if err := ac.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "imported service not found"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	tools, err := listImportedServiceTools(ctx, existing)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"service": toMCPMarketImportedServiceView(existing),
		"tools":   tools,
	}})
}

func (ac *AdminController) listMCPMarketServices(enabledOnly bool) ([]models.MCPMarketService, error) {
	var rows []models.MCPMarketService
	db := ac.DB.Model(&models.MCPMarketService{})
	if enabledOnly {
		db = db.Where("enabled = ?", true)
	}
	if err := db.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func mergeManualAndMarketServers(manualMCP map[string]interface{}, marketServices []models.MCPMarketService) (map[string]interface{}, []string, error) {
	merged := deepCopyMap(manualMCP)
	ensureMCPGlobalDefaults(merged)
	global := asMap(merged["global"])

	manualServers, err := decodeMCPServers(global["servers"])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse manual MCP servers: %w", err)
	}

	existingURLSet := make(map[string]struct{})
	for _, server := range manualServers {
		norm := normalizeServerURL(server)
		if norm == "" {
			continue
		}
		existingURLSet[norm] = struct{}{}
	}

	warnings := make([]string, 0)
	for _, service := range marketServices {
		if !service.Enabled {
			continue
		}

		normURL := mcpmarket.NormalizeURL(service.URL)
		if normURL == "" {
			continue
		}

		if _, exists := existingURLSet[normURL]; exists {
			warnings = append(warnings, fmt.Sprintf("market service %s skipped due to URL conflict with manual config", service.Name))
			continue
		}

		transport := normalizeImportedTransport(service.Transport)
		if transport != mcpmarket.TransportSSE && transport != mcpmarket.TransportStreamableHTTP {
			continue
		}

		server := mcpServerConfig{
			Name:         service.Name,
			Type:         transport,
			Url:          service.URL,
			Enabled:      service.Enabled,
			Provider:     "mcp-market",
			ServiceID:    service.ServiceID,
			Headers:      decodeHeadersJSON(service.HeadersJSON),
			AllowedTools: decodeAllowedToolsJSON(service.AllowedToolsJSON),
		}
		if transport == mcpmarket.TransportSSE {
			server.SSEUrl = service.URL
		}

		manualServers = append(manualServers, server)
		existingURLSet[normURL] = struct{}{}
	}

	global["servers"] = manualServers
	merged["global"] = global
	return merged, warnings, nil
}

func (ac *AdminController) mergeMCPWithEnabledMarketServices(manualMCP map[string]interface{}) (map[string]interface{}, []string, error) {
	services, err := ac.listMCPMarketServices(true)
	if err != nil {
		return nil, nil, err
	}
	return mergeManualAndMarketServers(manualMCP, services)
}

func filterEnabledMarketServices(rows []models.MCPMarketService) []models.MCPMarketService {
	ret := make([]models.MCPMarketService, 0, len(rows))
	for _, row := range rows {
		if row.Enabled {
			ret = append(ret, row)
		}
	}
	return ret
}

func collectManualURLSet(manualMCP map[string]interface{}) (map[string]struct{}, error) {
	ret := make(map[string]struct{})
	merged := deepCopyMap(manualMCP)
	ensureMCPGlobalDefaults(merged)
	global := asMap(merged["global"])
	servers, err := decodeMCPServers(global["servers"])
	if err != nil {
		return nil, err
	}
	for _, server := range servers {
		norm := normalizeServerURL(server)
		if norm == "" {
			continue
		}
		ret[norm] = struct{}{}
	}
	return ret, nil
}

func collectUsedServerNames(manualMCP map[string]interface{}, marketServices []models.MCPMarketService) (map[string]struct{}, error) {
	ret := make(map[string]struct{})
	merged := deepCopyMap(manualMCP)
	ensureMCPGlobalDefaults(merged)
	global := asMap(merged["global"])
	servers, err := decodeMCPServers(global["servers"])
	if err != nil {
		return nil, err
	}
	for _, server := range servers {
		name := strings.TrimSpace(server.Name)
		if name != "" {
			ret[name] = struct{}{}
		}
	}
	for _, item := range marketServices {
		name := strings.TrimSpace(item.Name)
		if name != "" {
			ret[name] = struct{}{}
		}
	}
	return ret, nil
}

func mergeServiceUpserts(base []models.MCPMarketService, upserts []models.MCPMarketService) []models.MCPMarketService {
	m := make(map[string]models.MCPMarketService)
	for _, item := range base {
		if item.URLHash == "" {
			continue
		}
		if item.Enabled {
			m[item.URLHash] = item
		}
	}
	for _, item := range upserts {
		if item.URLHash == "" {
			continue
		}
		if item.Enabled {
			m[item.URLHash] = item
		} else {
			delete(m, item.URLHash)
		}
	}

	ret := make([]models.MCPMarketService, 0, len(m))
	for _, item := range m {
		ret = append(ret, item)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].ID < ret[j].ID
	})
	return ret
}

func normalizeImportedTransport(transport string) string {
	transport = strings.ToLower(strings.TrimSpace(transport))
	switch transport {
	case "sse":
		return mcpmarket.TransportSSE
	case "streamablehttp", "streamable_http", "streamable-http", "http":
		return mcpmarket.TransportStreamableHTTP
	default:
		return transport
	}
}

func normalizeImportedServiceURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return mcpmarket.NormalizeURL(raw)
}

func normalizeImportedServiceName(name, serviceName, serviceID, fallbackURL string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	serviceName = strings.TrimSpace(serviceName)
	if serviceName != "" {
		return serviceName
	}
	serviceID = strings.TrimSpace(serviceID)
	if serviceID != "" {
		return serviceID
	}
	if fallbackURL != "" {
		return fallbackURL
	}
	return "mcp-service"
}

func normalizedURLHash(rawURL string) string {
	norm := normalizeImportedServiceURL(rawURL)
	if norm == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:])
}

func cleanHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]string)
	for k, v := range headers {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func encodeHeadersJSON(headers map[string]string) string {
	headers = cleanHeaders(headers)
	if len(headers) == 0 {
		return ""
	}
	b, err := json.Marshal(headers)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeHeadersJSON(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		return nil
	}
	return cleanHeaders(headers)
}

func cleanAllowedTools(tools []string) []string {
	if len(tools) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(tools))
	cleaned := make([]string, 0, len(tools))
	for _, toolName := range tools {
		toolName = strings.TrimSpace(toolName)
		if toolName == "" {
			continue
		}
		if _, exists := seen[toolName]; exists {
			continue
		}
		seen[toolName] = struct{}{}
		cleaned = append(cleaned, toolName)
	}
	if len(cleaned) == 0 {
		return nil
	}
	sort.Strings(cleaned)
	return cleaned
}

func encodeAllowedToolsJSON(tools []string) string {
	tools = cleanAllowedTools(tools)
	if len(tools) == 0 {
		return ""
	}
	b, err := json.Marshal(tools)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeAllowedToolsJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var tools []string
	if err := json.Unmarshal([]byte(raw), &tools); err != nil {
		return nil
	}
	return cleanAllowedTools(tools)
}

func buildImportedServiceModelFromRequest(req upsertMCPMarketImportedServiceRequest, existing *models.MCPMarketService) (models.MCPMarketService, error) {
	row := models.MCPMarketService{}
	if existing != nil {
		row = *existing
	}

	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	} else if existing == nil {
		row.Enabled = true
	}

	transport := strings.TrimSpace(req.Transport)
	if transport == "" && existing != nil {
		transport = existing.Transport
	}
	transport = normalizeImportedTransport(transport)
	if transport != mcpmarket.TransportSSE && transport != mcpmarket.TransportStreamableHTTP {
		return models.MCPMarketService{}, fmt.Errorf("transport only supports sse/streamablehttp")
	}

	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" && existing != nil {
		rawURL = existing.URL
	}
	if rawURL == "" {
		return models.MCPMarketService{}, fmt.Errorf("url cannot be empty")
	}
	if normalizeImportedServiceURL(rawURL) == "" {
		return models.MCPMarketService{}, fmt.Errorf("url format is invalid")
	}
	urlHash := normalizedURLHash(rawURL)
	if urlHash == "" {
		return models.MCPMarketService{}, fmt.Errorf("url cannot be empty")
	}

	row.Transport = transport
	row.URL = rawURL
	row.URLHash = urlHash
	row.Name = normalizeImportedServiceName(req.Name, req.ServiceName, req.ServiceID, rawURL)
	row.HeadersJSON = encodeHeadersJSON(req.Headers)
	if existing != nil && req.AllowedTools == nil {
		row.AllowedToolsJSON = existing.AllowedToolsJSON
	} else {
		row.AllowedToolsJSON = encodeAllowedToolsJSON(req.AllowedTools)
	}
	row.MarketID = req.MarketID
	row.ProviderID = mcpmarket.NormalizeProviderID(req.ProviderID)
	row.ServiceID = strings.TrimSpace(req.ServiceID)
	row.ServiceName = strings.TrimSpace(req.ServiceName)

	return row, nil
}

func toMCPMarketImportedServiceView(row models.MCPMarketService) mcpMarketImportedServiceView {
	return mcpMarketImportedServiceView{
		ID:           row.ID,
		Name:         row.Name,
		Enabled:      row.Enabled,
		Transport:    normalizeImportedTransport(row.Transport),
		URL:          row.URL,
		Headers:      decodeHeadersJSON(row.HeadersJSON),
		AllowedTools: decodeAllowedToolsJSON(row.AllowedToolsJSON),
		MarketID:     row.MarketID,
		ProviderID:   row.ProviderID,
		ServiceID:    row.ServiceID,
		ServiceName:  row.ServiceName,
		CreatedAt:    row.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    row.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func listImportedServiceTools(ctx context.Context, service models.MCPMarketService) ([]mcpMarketImportedToolView, error) {
	transportType := normalizeImportedTransport(service.Transport)
	if transportType != mcpmarket.TransportSSE && transportType != mcpmarket.TransportStreamableHTTP {
		return nil, fmt.Errorf("transport only supports sse/streamablehttp")
	}

	headers := decodeHeadersJSON(service.HeadersJSON)
	transportInstance, err := buildImportedServiceTransport(transportType, strings.TrimSpace(service.URL), headers)
	if err != nil {
		return nil, err
	}

	mcpClient := client.NewClient(transportInstance)
	defer mcpClient.Close()

	if err := mcpClient.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %v", err)
	}

	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "xiaozhi-manager-backend",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{
				Experimental: make(map[string]any),
			},
		},
	}
	if _, err := mcpClient.Initialize(ctx, initRequest); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP service: %v", err)
	}

	toolsResult, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get tool list: %v", err)
	}

	tools := make([]mcpMarketImportedToolView, 0, len(toolsResult.Tools))
	for _, item := range toolsResult.Tools {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		tools = append(tools, mcpMarketImportedToolView{
			Name:        name,
			Description: strings.TrimSpace(item.Description),
		})
	}
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})
	return tools, nil
}

func buildImportedServiceTransport(transportType, endpoint string, headers map[string]string) (transport.Interface, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("url cannot be empty")
	}

	switch transportType {
	case mcpmarket.TransportSSE:
		opts := make([]transport.ClientOption, 0)
		if len(headers) > 0 {
			opts = append(opts, transport.WithHeaders(headers))
		}
		sseTransport, err := transport.NewSSE(endpoint, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create SSE transport: %v", err)
		}
		return sseTransport, nil
	case mcpmarket.TransportStreamableHTTP:
		opts := make([]transport.StreamableHTTPCOption, 0)
		if len(headers) > 0 {
			opts = append(opts, transport.WithHTTPHeaders(headers))
		}
		httpTransport, err := transport.NewStreamableHTTP(endpoint, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create StreamableHTTP transport: %v", err)
		}
		return httpTransport, nil
	default:
		return nil, fmt.Errorf("unsupported transport: %s", transportType)
	}
}
