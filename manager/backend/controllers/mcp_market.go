package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"xiaozhi/manager/backend/models"
	mcpmarket "xiaozhi/manager/backend/services/mcp_market"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultMCPConfigName = "MCP Global Config"
	defaultMCPConfigID   = "mcp_global_config"
)

type upsertMCPMarketRequest struct {
	Name              string              `json:"name" binding:"required"`
	ProviderID        string              `json:"provider_id"`
	CatalogURL        string              `json:"catalog_url" binding:"required"`
	DetailURLTemplate string              `json:"detail_url_template"`
	Auth              upsertMCPMarketAuth `json:"auth"`
	Enabled           *bool               `json:"enabled"`
}

type upsertMCPMarketAuth struct {
	Type         string            `json:"type"`
	Token        string            `json:"token"`
	HeaderName   string            `json:"header_name"`
	ExtraHeaders map[string]string `json:"extra_headers"`
}

type importMCPMarketServiceRequest struct {
	MarketID     uint   `json:"market_id" binding:"required"`
	ServiceID    string `json:"service_id" binding:"required"`
	NameOverride string `json:"name_override"`
}

func (ac *AdminController) GetMCPMarketProviders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": mcpmarket.ListProviderPresets()})
}

func (ac *AdminController) GetMCPMarkets(c *gin.Context) {
	configs, err := ac.loadMCPMarketConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get MCP market connections"})
		return
	}

	views := make([]mcpmarket.MarketConnectionView, 0, len(configs))
	for _, cfg := range configs {
		marketCfg, err := parseStoredMarketConfig(cfg.JsonData)
		if err != nil {
			continue
		}
		views = append(views, toMarketConnectionView(cfg, marketCfg))
	}

	c.JSON(http.StatusOK, gin.H{"data": views})
}

func (ac *AdminController) CreateMCPMarket(c *gin.Context) {
	var req upsertMCPMarketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	marketCfg, err := buildStoredMarketConfig(req, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jsonData, err := json.Marshal(marketCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize market config"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	config := models.Config{
		Type:      mcpmarket.MCPMarketConfigType,
		Name:      strings.TrimSpace(req.Name),
		ConfigID:  "mcp_market_" + uuid.NewString(),
		JsonData:  string(jsonData),
		Enabled:   enabled,
		IsDefault: false,
	}

	if err := ac.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create MCP market connection"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": toMarketConnectionView(config, marketCfg)})
}

func (ac *AdminController) UpdateMCPMarket(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var existing models.Config
	if err := ac.DB.Where("id = ? AND type = ?", id, mcpmarket.MCPMarketConfigType).First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP market connection not found"})
		return
	}

	existingCfg, err := parseStoredMarketConfig(existing.JsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "existing market config is corrupted"})
		return
	}

	var req upsertMCPMarketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	marketCfg, err := buildStoredMarketConfig(req, &existingCfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	jsonData, err := json.Marshal(marketCfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize market config"})
		return
	}

	existing.Name = strings.TrimSpace(req.Name)
	existing.JsonData = string(jsonData)
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	if err := ac.DB.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update MCP market connection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toMarketConnectionView(existing, marketCfg)})
}

func (ac *AdminController) DeleteMCPMarket(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := ac.DB.Where("id = ? AND type = ?", id, mcpmarket.MCPMarketConfigType).Delete(&models.Config{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete MCP market connection"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

func (ac *AdminController) TestMCPMarket(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	marketModel, marketCfg, authCfg, err := ac.getMarketConfigByID(uint(id), false)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	raw, err := fetchMarketCatalog(ctx, marketCfg, authCfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("connection test failed: %v", err)})
		return
	}
	items := mcpmarket.ExtractServiceList(raw)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"market_id":       marketModel.ID,
		"market_name":     marketModel.Name,
		"reachable":       true,
		"service_count":   len(items),
		"checked_at_unix": time.Now().Unix(),
	}})
}

func (ac *AdminController) GetMCPMarketServices(c *gin.Context) {
	queryText := strings.TrimSpace(c.Query("q"))
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	markets, err := ac.loadEnabledMarketsWithAuth()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(markets) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": []mcpmarket.MarketServiceSummary{}, "total": 0, "page": page, "page_size": pageSize}})
		return
	}

	items, warnings := fetchAllMarketServices(c.Request.Context(), markets)
	if queryText != "" {
		items = filterMarketServices(items, queryText)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].MarketName == items[j].MarketName {
			return items[i].Name < items[j].Name
		}
		return items[i].MarketName < items[j].MarketName
	})

	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	paged := items[start:end]

	data := gin.H{
		"items":     paged,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	if len(warnings) > 0 {
		data["warnings"] = warnings
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (ac *AdminController) GetMCPMarketServiceDetail(c *gin.Context) {
	marketID, _ := strconv.Atoi(c.Param("market_id"))
	serviceID := strings.TrimSpace(c.Param("service_id"))
	serviceID = strings.TrimPrefix(serviceID, "/")
	if serviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service_id cannot be empty"})
		return
	}

	marketModel, marketCfg, authCfg, err := ac.getMarketConfigByID(uint(marketID), true)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	detail, err := fetchServiceDetail(c.Request.Context(), marketModel, marketCfg, authCfg, serviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": detail})
}

func (ac *AdminController) ImportMCPMarketService(c *gin.Context) {
	var req importMCPMarketServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	marketModel, marketCfg, authCfg, err := ac.getMarketConfigByID(req.MarketID, true)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	detail, err := fetchServiceDetail(c.Request.Context(), marketModel, marketCfg, authCfg, req.ServiceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(detail.Endpoints) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this service has no available SSE/StreamableHTTP endpoints; please activate or deploy it upstream first"})
		return
	}

	current, _, err := ac.loadCurrentMCPConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read existing MCP config: %v", err)})
		return
	}

	existingAll, err := ac.listMCPMarketServices(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read imported services: %v", err)})
		return
	}
	existingEnabled := filterEnabledMarketServices(existingAll)

	manualURLSet, err := collectManualURLSet(current.MCP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to parse manual MCP config: %v", err)})
		return
	}
	usedNames, err := collectUsedServerNames(current.MCP, existingAll)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to parse MCP service names: %v", err)})
		return
	}

	urlHashToExisting := make(map[string]models.MCPMarketService)
	for _, item := range existingAll {
		if strings.TrimSpace(item.URLHash) != "" {
			urlHashToExisting[item.URLHash] = item
		}
	}

	baseName := strings.TrimSpace(req.NameOverride)
	if baseName == "" {
		baseName = strings.TrimSpace(detail.Name)
	}
	if baseName == "" {
		baseName = "mcp-service"
	}

	upserts := make([]models.MCPMarketService, 0, len(detail.Endpoints))
	importedNames := make([]string, 0, len(detail.Endpoints))
	skippedURLs := make([]string, 0)

	for idx, endpoint := range detail.Endpoints {
		normURL := mcpmarket.NormalizeURL(endpoint.URL)
		if normURL == "" {
			continue
		}
		if _, conflict := manualURLSet[normURL]; conflict {
			skippedURLs = append(skippedURLs, endpoint.URL)
			continue
		}

		nameCandidate := baseName
		if len(detail.Endpoints) > 1 {
			nameCandidate = fmt.Sprintf("%s-%d", baseName, idx+1)
		}
		nameCandidate = resolveUniqueName(usedNames, nameCandidate)
		usedNames[nameCandidate] = struct{}{}

		row := models.MCPMarketService{
			Name:             nameCandidate,
			Enabled:          true,
			Transport:        normalizeImportedTransport(endpoint.Transport),
			URL:              endpoint.URL,
			URLHash:          normalizedURLHash(endpoint.URL),
			HeadersJSON:      encodeHeadersJSON(endpoint.Headers),
			AllowedToolsJSON: "",
			MarketID:         &marketModel.ID,
			ProviderID:       mcpmarket.NormalizeProviderID(marketCfg.ProviderID),
			ServiceID:        detail.ServiceID,
			ServiceName:      detail.Name,
		}
		if row.Transport != mcpmarket.TransportSSE && row.Transport != mcpmarket.TransportStreamableHTTP {
			continue
		}
		if row.URLHash == "" {
			continue
		}
		if existing, ok := urlHashToExisting[row.URLHash]; ok {
			row.ID = existing.ID
			row.AllowedToolsJSON = existing.AllowedToolsJSON
			if strings.TrimSpace(req.NameOverride) == "" && strings.TrimSpace(existing.Name) != "" {
				row.Name = strings.TrimSpace(existing.Name)
			}
		}

		upserts = append(upserts, row)
		importedNames = append(importedNames, row.Name)
	}

	if len(upserts) == 0 {
		if len(skippedURLs) > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "all importable endpoints conflict with manual config URLs and were skipped (manual takes priority)", "skipped_urls": skippedURLs})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid importable endpoints found"})
		return
	}

	candidateEnabled := mergeServiceUpserts(existingEnabled, upserts)
	_, mergeWarnings, err := mergeManualAndMarketServers(current.MCP, candidateEnabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to merge MCP config: %v", err)})
		return
	}

	tx := ac.DB.Begin()
	for _, row := range upserts {
		if row.ID == 0 {
			if err := tx.Create(&row).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to write imported service: %v", err)})
				return
			}
			continue
		}
		updateMap := map[string]interface{}{
			"name":               row.Name,
			"enabled":            row.Enabled,
			"transport":          row.Transport,
			"url":                row.URL,
			"url_hash":           row.URLHash,
			"headers_json":       row.HeadersJSON,
			"allowed_tools_json": row.AllowedToolsJSON,
			"market_id":          row.MarketID,
			"provider_id":        row.ProviderID,
			"service_id":         row.ServiceID,
			"service_name":       row.ServiceName,
		}
		if err := tx.Model(&models.MCPMarketService{}).Where("id = ?", row.ID).Updates(updateMap).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update imported service: %v", err)})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to persist imported service: %v", err)})
		return
	}

	ac.notifySystemConfigChanged()

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"service_id":     detail.ServiceID,
		"service_name":   detail.Name,
		"market_id":      detail.MarketID,
		"market_name":    detail.MarketName,
		"imported_names": importedNames,
		"imported_count": len(importedNames),
		"skipped_urls":   skippedURLs,
		"warnings":       mergeWarnings,
	}})
}

func (ac *AdminController) loadMCPMarketConfigs() ([]models.Config, error) {
	var configs []models.Config
	if err := ac.DB.Where("type = ?", mcpmarket.MCPMarketConfigType).Order("id ASC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (ac *AdminController) loadEnabledMarketsWithAuth() ([]marketWithAuth, error) {
	configs, err := ac.loadMCPMarketConfigs()
	if err != nil {
		return nil, err
	}

	ret := make([]marketWithAuth, 0)
	for _, cfg := range configs {
		marketCfg, err := parseStoredMarketConfig(cfg.JsonData)
		if err != nil {
			continue
		}
		if !marketCfg.Enabled || !cfg.Enabled {
			continue
		}
		authCfg, err := decodeMarketAuthConfig(marketCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt auth for market %s: %w", cfg.Name, err)
		}
		ret = append(ret, marketWithAuth{ConfigModel: cfg, MarketConfig: marketCfg, AuthConfig: authCfg})
	}
	return ret, nil
}

func (ac *AdminController) getMarketConfigByID(id uint, requireEnabled bool) (models.Config, mcpmarket.MarketConnection, mcpmarket.AuthConfig, error) {
	var cfg models.Config
	if err := ac.DB.Where("id = ? AND type = ?", id, mcpmarket.MCPMarketConfigType).First(&cfg).Error; err != nil {
		return models.Config{}, mcpmarket.MarketConnection{}, mcpmarket.AuthConfig{}, err
	}
	marketCfg, err := parseStoredMarketConfig(cfg.JsonData)
	if err != nil {
		return models.Config{}, mcpmarket.MarketConnection{}, mcpmarket.AuthConfig{}, err
	}
	if requireEnabled && (!cfg.Enabled || !marketCfg.Enabled) {
		return models.Config{}, mcpmarket.MarketConnection{}, mcpmarket.AuthConfig{}, fmt.Errorf("market connection is disabled")
	}
	authCfg, err := decodeMarketAuthConfig(marketCfg)
	if err != nil {
		return models.Config{}, mcpmarket.MarketConnection{}, mcpmarket.AuthConfig{}, err
	}
	return cfg, marketCfg, authCfg, nil
}

func fetchAllMarketServices(ctx context.Context, markets []marketWithAuth) ([]mcpmarket.MarketServiceSummary, []string) {
	items := make([]mcpmarket.MarketServiceSummary, 0)
	warnings := make([]string, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, market := range markets {
		market := market
		wg.Add(1)
		go func() {
			defer wg.Done()
			raw, err := fetchMarketCatalog(ctx, market.MarketConfig, market.AuthConfig)
			if err != nil {
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf("%s: %v", market.ConfigModel.Name, err))
				mu.Unlock()
				return
			}
			list := mcpmarket.ExtractServiceList(raw)
			local := make([]mcpmarket.MarketServiceSummary, 0, len(list))
			for _, item := range list {
				serviceID, name, desc := mcpmarket.ParseServiceSummary(item)
				if serviceID == "" || name == "" {
					continue
				}
				local = append(local, mcpmarket.MarketServiceSummary{
					MarketID:    market.ConfigModel.ID,
					MarketName:  market.ConfigModel.Name,
					ServiceID:   serviceID,
					Name:        name,
					Description: desc,
				})
			}
			mu.Lock()
			items = append(items, local...)
			mu.Unlock()
		}()
	}

	wg.Wait()
	return dedupeServiceSummaries(items), warnings
}

func fetchServiceDetail(ctx context.Context, marketModel models.Config, marketCfg mcpmarket.MarketConnection, authCfg mcpmarket.AuthConfig, serviceID string) (*mcpmarket.MarketServiceDetail, error) {
	if mcpmarket.NormalizeProviderID(marketCfg.ProviderID) == mcpmarket.ProviderModelScope && strings.TrimSpace(authCfg.Token) == "" {
		return nil, fmt.Errorf("ModelScope service detail requires a Token; please set the Token in this market connection")
	}

	detailURL, err := mcpmarket.BuildDetailURL(marketCfg.CatalogURL, marketCfg.DetailURLTemplate, serviceID)
	if err != nil {
		return nil, err
	}

	raw, err := fetchMarketDetail(ctx, marketCfg, authCfg, detailURL)
	if err != nil {
		if mcpmarket.NormalizeProviderID(marketCfg.ProviderID) == mcpmarket.ProviderModelScope {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "invalidauthentication") || strings.Contains(errMsg, "authentication failed") || strings.Contains(errMsg, "401") {
				return nil, fmt.Errorf("ModelScope service detail authentication failed; please verify the Token in this market connection")
			}
		}
		return nil, fmt.Errorf("failed to fetch service detail: %w", err)
	}
	detail, err := mcpmarket.ParseServiceDetail(raw, marketModel.ID, marketModel.Name, serviceID, mcpmarket.BuildHeaders(authCfg))
	if err != nil {
		return nil, err
	}
	if mcpmarket.NormalizeProviderID(marketCfg.ProviderID) == mcpmarket.ProviderModelScope && len(detail.Endpoints) == 0 {
		return nil, fmt.Errorf("this service has no callable endpoints (it may not be fully activated on ModelScope); please activate it there first")
	}
	return detail, nil
}

func fetchMarketCatalog(ctx context.Context, marketCfg mcpmarket.MarketConnection, authCfg mcpmarket.AuthConfig) (interface{}, error) {
	catalogURL := strings.TrimSpace(marketCfg.CatalogURL)
	if mcpmarket.NormalizeProviderID(marketCfg.ProviderID) == mcpmarket.ProviderModelScope {
		if strings.TrimSpace(authCfg.Token) == "" {
			return nil, fmt.Errorf("ModelScope market only fetches activated services; please set the Token in this market connection first")
		}
		catalogURL = strings.TrimRight(catalogURL, "/") + "/operational"
	}
	return mcpmarket.FetchJSON(ctx, catalogURL, mcpmarket.BuildHeaders(authCfg), buildCatalogFetchOptions(marketCfg, authCfg))
}

func fetchMarketDetail(ctx context.Context, marketCfg mcpmarket.MarketConnection, authCfg mcpmarket.AuthConfig, detailURL string) (interface{}, error) {
	return mcpmarket.FetchJSON(ctx, detailURL, mcpmarket.BuildHeaders(authCfg), buildDetailFetchOptions(marketCfg, authCfg))
}

func buildCatalogFetchOptions(marketCfg mcpmarket.MarketConnection, authCfg mcpmarket.AuthConfig) mcpmarket.HTTPOptions {
	opts := mcpmarket.HTTPOptions{
		Timeout: 15 * time.Second,
		Cookies: mcpmarket.BuildCookies(authCfg, marketCfg.ProviderID),
	}
	return opts
}

func buildDetailFetchOptions(marketCfg mcpmarket.MarketConnection, authCfg mcpmarket.AuthConfig) mcpmarket.HTTPOptions {
	opts := mcpmarket.HTTPOptions{
		Timeout: 20 * time.Second,
		Cookies: mcpmarket.BuildCookies(authCfg, marketCfg.ProviderID),
	}

	if mcpmarket.NormalizeProviderID(marketCfg.ProviderID) == mcpmarket.ProviderModelScope {
		opts.Query = map[string]string{"get_operational_url": "true"}
	}
	return opts
}

func mergeEndpointsIntoMCPConfig(current mcpmarket.MergedMCPConfig, detail *mcpmarket.MarketServiceDetail, nameOverride string) (mcpmarket.MergedMCPConfig, []string, string, error) {
	mcpMap := deepCopyMap(current.MCP)
	localMap := deepCopyMap(current.LocalMCP)
	ensureMCPGlobalDefaults(mcpMap)
	global := asMap(mcpMap["global"])

	servers, err := decodeMCPServers(global["servers"])
	if err != nil {
		return mcpmarket.MergedMCPConfig{}, nil, "", fmt.Errorf("failed to parse existing MCP servers: %w", err)
	}

	existingURLSet := make(map[string]struct{})
	existingNames := make(map[string]struct{})
	for _, server := range servers {
		norm := normalizeServerURL(server)
		if norm != "" {
			existingURLSet[norm] = struct{}{}
		}
		existingNames[server.Name] = struct{}{}
	}

	baseName := strings.TrimSpace(nameOverride)
	if baseName == "" {
		baseName = strings.TrimSpace(detail.Name)
	}
	if baseName == "" {
		baseName = "mcp-service"
	}

	importedNames := make([]string, 0)
	for idx, endpoint := range detail.Endpoints {
		normURL := mcpmarket.NormalizeURL(endpoint.URL)
		if _, exists := existingURLSet[normURL]; exists {
			return mcpmarket.MergedMCPConfig{}, nil, endpoint.URL, fmt.Errorf("URL conflict: an MCP service with the same address already exists")
		}

		candidateName := baseName
		if len(detail.Endpoints) > 1 {
			candidateName = fmt.Sprintf("%s-%d", baseName, idx+1)
		}
		candidateName = resolveUniqueName(existingNames, candidateName)

		newServer := mcpServerConfig{
			Name:         candidateName,
			Type:         endpoint.Transport,
			Url:          endpoint.URL,
			Enabled:      true,
			Provider:     "mcp-market",
			ServiceID:    detail.ServiceID,
			Headers:      endpoint.Headers,
			AllowedTools: nil,
		}
		if endpoint.Transport == mcpmarket.TransportSSE {
			newServer.SSEUrl = endpoint.URL
		}

		servers = append(servers, newServer)
		existingNames[candidateName] = struct{}{}
		existingURLSet[normURL] = struct{}{}
		importedNames = append(importedNames, candidateName)
	}

	global["enabled"] = true
	global["servers"] = servers
	mcpMap["global"] = global

	return mcpmarket.MergedMCPConfig{MCP: mcpMap, LocalMCP: localMap}, importedNames, "", nil
}

func (ac *AdminController) loadCurrentMCPConfig() (mcpmarket.MergedMCPConfig, *models.Config, error) {
	var configs []models.Config
	if err := ac.DB.Where("type = ?", mcpmarket.MCPConfigType).Order("is_default DESC, id ASC").Find(&configs).Error; err != nil {
		return mcpmarket.MergedMCPConfig{}, nil, err
	}

	mcpMap := defaultMCPMap()
	localMap := defaultLocalMCPMap()
	var selected *models.Config

	if len(configs) > 0 {
		selected = &configs[0]
		if payload, err := parseJSONMap(selected.JsonData); err == nil {
			if v, ok := payload["mcp"]; ok {
				if mv := asMap(v); mv != nil {
					mcpMap = mv
				}
			} else if v, ok := payload["global"]; ok {
				mcpMap = map[string]interface{}{"global": v}
			} else {
				mcpMap = payload
			}

			if v, ok := payload["local_mcp"]; ok {
				if lv := asMap(v); lv != nil {
					localMap = lv
				}
			}
		}
	}

	ensureMCPGlobalDefaults(mcpMap)
	ensureLocalMCPDefaults(localMap)
	return mcpmarket.MergedMCPConfig{MCP: mcpMap, LocalMCP: localMap}, selected, nil
}

func (ac *AdminController) persistMCPConfig(existing *models.Config, merged mcpmarket.MergedMCPConfig) error {
	payload := map[string]interface{}{
		"mcp":       merged.MCP,
		"local_mcp": merged.LocalMCP,
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if existing != nil {
		existing.Name = defaultMCPConfigName
		existing.Type = mcpmarket.MCPConfigType
		existing.ConfigID = defaultMCPConfigID
		existing.IsDefault = true
		existing.Enabled = true
		existing.JsonData = string(jsonBytes)
		return ac.DB.Save(existing).Error
	}

	model := models.Config{
		Type:      mcpmarket.MCPConfigType,
		Name:      defaultMCPConfigName,
		ConfigID:  defaultMCPConfigID,
		Enabled:   true,
		IsDefault: true,
		JsonData:  string(jsonBytes),
	}
	return ac.DB.Create(&model).Error
}

type marketWithAuth struct {
	ConfigModel  models.Config
	MarketConfig mcpmarket.MarketConnection
	AuthConfig   mcpmarket.AuthConfig
}

type mcpServerConfig struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Url          string            `json:"url"`
	SSEUrl       string            `json:"sse_url,omitempty"`
	Enabled      bool              `json:"enabled"`
	Provider     string            `json:"provider,omitempty"`
	ServiceID    string            `json:"service_id,omitempty"`
	AuthRef      string            `json:"auth_ref,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	AllowedTools []string          `json:"allowed_tools,omitempty"`
}

func buildStoredMarketConfig(req upsertMCPMarketRequest, existing *mcpmarket.MarketConnection) (mcpmarket.MarketConnection, error) {
	name := strings.TrimSpace(req.Name)
	providerID := strings.TrimSpace(req.ProviderID)
	if providerID == "" && existing != nil {
		providerID = existing.ProviderID
	}
	providerID = mcpmarket.NormalizeProviderID(providerID)

	catalogURL := strings.TrimSpace(req.CatalogURL)
	detailTemplate := strings.TrimSpace(req.DetailURLTemplate)
	if name == "" || catalogURL == "" {
		return mcpmarket.MarketConnection{}, fmt.Errorf("name and catalog_url cannot be empty")
	}
	if _, err := url.ParseRequestURI(catalogURL); err != nil {
		return mcpmarket.MarketConnection{}, fmt.Errorf("catalog_url format is invalid")
	}
	if detailTemplate != "" {
		validateURL := strings.ReplaceAll(detailTemplate, "{id}", "placeholder")
		validateURL = strings.ReplaceAll(validateURL, "{raw_id}", "placeholder/raw")
		if _, err := url.ParseRequestURI(validateURL); err != nil {
			return mcpmarket.MarketConnection{}, fmt.Errorf("detail_url_template format is invalid")
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	} else if existing != nil {
		enabled = existing.Enabled
	}

	authType := strings.ToLower(strings.TrimSpace(req.Auth.Type))
	if authType == "" {
		if existing != nil && existing.AuthType != "" {
			authType = existing.AuthType
		} else {
			authType = mcpmarket.AuthTypeNone
		}
	}
	if providerID == mcpmarket.ProviderModelScope {
		authType = mcpmarket.AuthTypeBearer
	}
	if authType != mcpmarket.AuthTypeNone && authType != mcpmarket.AuthTypeBearer && authType != mcpmarket.AuthTypeHeader {
		return mcpmarket.MarketConnection{}, fmt.Errorf("auth.type only supports none/bearer/header")
	}

	headerName := strings.TrimSpace(req.Auth.HeaderName)
	if headerName == "" && existing != nil {
		headerName = existing.HeaderName
	}
	if authType == mcpmarket.AuthTypeHeader && headerName == "" {
		headerName = "Authorization"
	}
	if providerID == mcpmarket.ProviderModelScope {
		headerName = "Authorization"
	}

	tokenCipher := ""
	tokenNonce := ""
	tokenMask := ""
	if existing != nil {
		tokenCipher = existing.TokenCiphertext
		tokenNonce = existing.TokenNonce
		tokenMask = existing.TokenMask
	}

	newToken := strings.TrimSpace(req.Auth.Token)
	switch {
	case authType == mcpmarket.AuthTypeNone:
		tokenCipher, tokenNonce, tokenMask = "", "", ""
	case newToken != "":
		tokenCipher = newToken
		tokenNonce = ""
		tokenMask = mcpmarket.MaskToken(newToken)
	case existing == nil && authType != mcpmarket.AuthTypeNone:
		return mcpmarket.MarketConnection{}, fmt.Errorf("auth is enabled but no token was provided")
	}

	extraHeadersCipher := ""
	extraHeadersNonce := ""
	if existing != nil {
		extraHeadersCipher = existing.ExtraHeadersCiphertext
		extraHeadersNonce = existing.ExtraHeadersNonce
	}
	if req.Auth.ExtraHeaders != nil {
		headersJSON, err := json.Marshal(req.Auth.ExtraHeaders)
		if err != nil {
			return mcpmarket.MarketConnection{}, fmt.Errorf("extra_headers format is invalid")
		}
		if string(headersJSON) == "null" || string(headersJSON) == "{}" {
			extraHeadersCipher = ""
			extraHeadersNonce = ""
		} else {
			extraHeadersCipher = string(headersJSON)
			extraHeadersNonce = ""
		}
	}
	if providerID == mcpmarket.ProviderModelScope {
		extraHeadersCipher = ""
		extraHeadersNonce = ""
	}

	return mcpmarket.MarketConnection{
		Name:                   name,
		ProviderID:             providerID,
		CatalogURL:             catalogURL,
		DetailURLTemplate:      detailTemplate,
		Enabled:                enabled,
		AuthType:               authType,
		HeaderName:             headerName,
		TokenCiphertext:        tokenCipher,
		TokenNonce:             tokenNonce,
		TokenMask:              tokenMask,
		ExtraHeadersCiphertext: extraHeadersCipher,
		ExtraHeadersNonce:      extraHeadersNonce,
		UpdatedAtUnix:          time.Now().Unix(),
	}, nil
}

func parseStoredMarketConfig(jsonData string) (mcpmarket.MarketConnection, error) {
	if strings.TrimSpace(jsonData) == "" {
		return mcpmarket.MarketConnection{}, fmt.Errorf("json_data is empty")
	}
	var cfg mcpmarket.MarketConnection
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		return mcpmarket.MarketConnection{}, err
	}
	cfg.ProviderID = mcpmarket.NormalizeProviderID(cfg.ProviderID)
	if cfg.Name == "" || cfg.CatalogURL == "" {
		return mcpmarket.MarketConnection{}, fmt.Errorf("market config is missing name/catalog_url")
	}
	return cfg, nil
}

func decodeMarketAuthConfig(cfg mcpmarket.MarketConnection) (mcpmarket.AuthConfig, error) {
	auth := mcpmarket.AuthConfig{
		Type:       cfg.AuthType,
		HeaderName: cfg.HeaderName,
	}
	if mcpmarket.NormalizeProviderID(cfg.ProviderID) == mcpmarket.ProviderModelScope {
		auth.Type = mcpmarket.AuthTypeBearer
		auth.HeaderName = "Authorization"
	}
	if auth.Type == "" {
		auth.Type = mcpmarket.AuthTypeNone
	}

	if cfg.TokenCiphertext != "" {
		auth.Token = cfg.TokenCiphertext
		// backward-compat with legacy encrypted storage: decrypt with old key if nonce is present
		if strings.TrimSpace(cfg.TokenNonce) != "" {
			if token, err := mcpmarket.DecryptText(cfg.TokenCiphertext, cfg.TokenNonce); err == nil {
				auth.Token = token
			}
		}
	}

	if cfg.ExtraHeadersCiphertext != "" {
		headersJSON := cfg.ExtraHeadersCiphertext
		var headers map[string]string
		if err := json.Unmarshal([]byte(headersJSON), &headers); err != nil {
			// backward-compat with legacy encrypted storage: try decrypting with old key
			nonce := cfg.ExtraHeadersNonce
			if strings.TrimSpace(nonce) == "" {
				nonce = cfg.TokenNonce
			}
			if decrypted, derr := mcpmarket.DecryptText(cfg.ExtraHeadersCiphertext, nonce); derr == nil {
				if uerr := json.Unmarshal([]byte(decrypted), &headers); uerr != nil {
					headers = nil
				}
			}
		}
		if len(headers) > 0 {
			auth.ExtraHeaders = headers
		}
	}

	return auth, nil
}

func toMarketConnectionView(cfg models.Config, marketCfg mcpmarket.MarketConnection) mcpmarket.MarketConnectionView {
	authType := marketCfg.AuthType
	headerName := marketCfg.HeaderName
	if mcpmarket.NormalizeProviderID(marketCfg.ProviderID) == mcpmarket.ProviderModelScope {
		authType = mcpmarket.AuthTypeBearer
		headerName = "Authorization"
	}
	return mcpmarket.MarketConnectionView{
		ID:                cfg.ID,
		ConfigID:          cfg.ConfigID,
		Name:              marketCfg.Name,
		ProviderID:        mcpmarket.NormalizeProviderID(marketCfg.ProviderID),
		CatalogURL:        marketCfg.CatalogURL,
		DetailURLTemplate: marketCfg.DetailURLTemplate,
		Enabled:           cfg.Enabled && marketCfg.Enabled,
		AuthType:          authType,
		HeaderName:        headerName,
		TokenMask:         marketCfg.TokenMask,
		HasToken:          marketCfg.TokenCiphertext != "",
		CreatedAt:         cfg.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         cfg.UpdatedAt.Format(time.RFC3339),
	}
}

func dedupeServiceSummaries(items []mcpmarket.MarketServiceSummary) []mcpmarket.MarketServiceSummary {
	ret := make([]mcpmarket.MarketServiceSummary, 0, len(items))
	seen := make(map[string]struct{})
	for _, item := range items {
		key := fmt.Sprintf("%d|%s", item.MarketID, item.ServiceID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ret = append(ret, item)
	}
	return ret
}

func filterMarketServices(items []mcpmarket.MarketServiceSummary, q string) []mcpmarket.MarketServiceSummary {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return items
	}
	ret := make([]mcpmarket.MarketServiceSummary, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), q) || strings.Contains(strings.ToLower(item.Description), q) || strings.Contains(strings.ToLower(item.ServiceID), q) {
			ret = append(ret, item)
		}
	}
	return ret
}

func parsePositiveInt(s string, fallback int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && n > 0 {
		if n > 200 {
			return 200
		}
		return n
	}
	return fallback
}

func parseJSONMap(raw string) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	if strings.TrimSpace(raw) == "" {
		return ret, nil
	}
	if err := json.Unmarshal([]byte(raw), &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func defaultMCPMap() map[string]interface{} {
	return map[string]interface{}{
		"global": map[string]interface{}{
			"enabled":                true,
			"servers":                []interface{}{},
			"reconnect_interval":     300,
			"max_reconnect_attempts": 10,
		},
	}
}

func defaultLocalMCPMap() map[string]interface{} {
	return map[string]interface{}{
		"exit_conversation":          true,
		"clear_conversation_history": true,
		"play_music":                 false,
	}
}

func ensureMCPGlobalDefaults(m map[string]interface{}) {
	if m == nil {
		return
	}
	global := asMap(m["global"])
	if global == nil {
		global = map[string]interface{}{}
	}
	if _, ok := global["enabled"]; !ok {
		global["enabled"] = true
	}
	if _, ok := global["servers"]; !ok {
		global["servers"] = []interface{}{}
	}
	if _, ok := global["reconnect_interval"]; !ok {
		global["reconnect_interval"] = 300
	}
	if _, ok := global["max_reconnect_attempts"]; !ok {
		global["max_reconnect_attempts"] = 10
	}
	m["global"] = global
}

func ensureLocalMCPDefaults(m map[string]interface{}) {
	if m == nil {
		return
	}
	if _, ok := m["exit_conversation"]; !ok {
		m["exit_conversation"] = true
	}
	if _, ok := m["clear_conversation_history"]; !ok {
		m["clear_conversation_history"] = true
	}
	if _, ok := m["play_music"]; !ok {
		m["play_music"] = false
	}
}

func decodeMCPServers(v interface{}) ([]mcpServerConfig, error) {
	if v == nil {
		return []mcpServerConfig{}, nil
	}
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var servers []mcpServerConfig
	if err := json.Unmarshal(jsonBytes, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}

func normalizeServerURL(server mcpServerConfig) string {
	endpoint := strings.TrimSpace(server.Url)
	if strings.EqualFold(strings.TrimSpace(server.Type), mcpmarket.TransportSSE) {
		if strings.TrimSpace(server.SSEUrl) != "" {
			endpoint = strings.TrimSpace(server.SSEUrl)
		}
	}
	return mcpmarket.NormalizeURL(endpoint)
}

func resolveUniqueName(existing map[string]struct{}, base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "mcp-service"
	}
	if _, ok := existing[base]; !ok {
		return base
	}
	for i := 2; i < 10000; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := existing[candidate]; !ok {
			return candidate
		}
	}
	return base + "-dup"
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	b, _ := json.Marshal(m)
	ret := make(map[string]interface{})
	_ = json.Unmarshal(b, &ret)
	return ret
}

func asMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}
