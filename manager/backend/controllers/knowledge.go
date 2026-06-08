package controllers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const knowledgeDocumentUploadMaxBytes = 2 * 1024 * 1024

const knowledgeUploadContentPrefix = "__KB_FILE_UPLOAD_V1__:"

var allowedKnowledgeRagflowFileExt = map[string]struct{}{
	".txt":      {},
	".text":     {},
	".md":       {},
	".markdown": {},
	".pdf":      {},
	".doc":      {},
	".docx":     {},
	".ppt":      {},
	".pptx":     {},
	".xls":      {},
	".xlsx":     {},
	".wps":      {},
	".json":     {},
	".csv":      {},
	".log":      {},
	".xml":      {},
	".html":     {},
	".htm":      {},
	".yml":      {},
	".yaml":     {},
	".rtf":      {},
	".sql":      {},
	".ini":      {},
	".jpg":      {},
	".jpeg":     {},
	".png":      {},
	".gif":      {},
	".bmp":      {},
	".webp":     {},
	".tif":      {},
	".tiff":     {},
	".eml":      {},
	".msg":      {},
}

var allowedKnowledgeWeknoraFileExt = map[string]struct{}{
	".txt":      {},
	".text":     {},
	".md":       {},
	".markdown": {},
	".pdf":      {},
	".doc":      {},
	".docx":     {},
	".ppt":      {},
	".pptx":     {},
	".xls":      {},
	".xlsx":     {},
	".wps":      {},
	".json":     {},
	".csv":      {},
	".log":      {},
	".xml":      {},
	".html":     {},
	".htm":      {},
	".yml":      {},
	".yaml":     {},
	".rtf":      {},
	".sql":      {},
	".ini":      {},
	".jpg":      {},
	".jpeg":     {},
	".png":      {},
	".gif":      {},
	".bmp":      {},
	".webp":     {},
	".tif":      {},
	".tiff":     {},
	".eml":      {},
	".msg":      {},
}

var allowedKnowledgeDifyFileExt = map[string]struct{}{
	".txt":      {},
	".md":       {},
	".markdown": {},
	".pdf":      {},
	".html":     {},
	".htm":      {},
	".xlsx":     {},
	".xls":      {},
	".docx":     {},
	".csv":      {},
	".eml":      {},
	".msg":      {},
	".pptx":     {},
	".ppt":      {},
	".xml":      {},
	".epub":     {},
}

type knowledgeSearchTestHit struct {
	Title   string  `json:"title"`
	Score   float64 `json:"score"`
	Content string  `json:"content"`
}

func isKnowledgeFeatureEnabled(db *gorm.DB) (bool, error) {
	var count int64
	if err := db.Model(&models.Config{}).
		Where("type = ? AND enabled = ? AND is_default = ?", "knowledge_search", true, true).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func ensureKnowledgeFeatureEnabled(c *gin.Context, db *gorm.DB) bool {
	enabled, err := isKnowledgeFeatureEnabled(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check knowledge base status"})
		return false
	}
	if !enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "knowledge base feature disabled (no default provider configured)"})
		return false
	}
	return true
}

func (ac *AdminController) GetKnowledgeSearchConfigs(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	offset := (page - 1) * pageSize
	var total int64
	if err := ac.DB.Model(&models.Config{}).Where("type = ?", "knowledge_search").Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get knowledge base retrieval config"})
		return
	}
	var configs []models.Config
	if err := ac.DB.Where("type = ?", "knowledge_search").Order("id ASC").Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get knowledge base retrieval config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs, "total": total, "page": page, "page_size": pageSize})
}

func (ac *AdminController) CreateKnowledgeSearchConfig(c *gin.Context) {
	var config models.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Type = "knowledge_search"
	ac.createConfigWithType(c, &config)
}

func (ac *AdminController) UpdateKnowledgeSearchConfig(c *gin.Context) {
	ac.updateConfigWithType(c, "knowledge_search")
}

func (ac *AdminController) DeleteKnowledgeSearchConfig(c *gin.Context) {
	ac.deleteConfigWithType(c, "knowledge_search")
}

type knowledgeProviderModelOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Provider string `json:"provider,omitempty"`
}

func (ac *AdminController) ListWeknoraModels(c *gin.Context) {
	var req struct {
		BaseURL string `json:"base_url" binding:"required"`
		APIKey  string `json:"api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}

	baseURL := strings.TrimSpace(req.BaseURL)
	apiKey := strings.TrimSpace(req.APIKey)
	if baseURL == "" || apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "base_url and api_key cannot be empty"})
		return
	}

	client := &http.Client{Timeout: 20 * time.Second}
	endpoints := buildWeknoraModelListCandidateEndpoints(baseURL)
	var (
		statusCode int
		bodyBytes  []byte
		err        error
		lastErr    error
		lastStatus int
		lastURL    string
		tryLogs    []string
	)
	for _, endpoint := range endpoints {
		lastURL = endpoint
		statusCode, bodyBytes, err = doWeknoraJSONRequest(client, http.MethodGet, endpoint, apiKey, nil, nil)
		if err == nil {
			lastErr = nil
			lastStatus = statusCode
			break
		}
		lastErr = err
		lastStatus = statusCode
		tryLogs = append(tryLogs, fmt.Sprintf("%s => status=%d err=%v", endpoint, statusCode, err))
		if statusCode != http.StatusNotFound {
			break
		}
	}
	if lastErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":       fmt.Sprintf("failed to fetch WeKnora model list: %v; tried paths: %s", lastErr, strings.Join(tryLogs, " | ")),
			"status_code": lastStatus,
			"endpoint":    lastURL,
		})
		return
	}

	var parsed map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to parse WeKnora model list: " + err.Error()})
			return
		}
	}

	allModels := extractWeknoraModelOptions(parsed)
	embeddingModels := make([]knowledgeProviderModelOption, 0, len(allModels))
	llmModels := make([]knowledgeProviderModelOption, 0, len(allModels))
	rerankModels := make([]knowledgeProviderModelOption, 0, len(allModels))
	for _, item := range allModels {
		if isWeknoraEmbeddingModel(item) {
			embeddingModels = append(embeddingModels, item)
			continue
		}
		if isWeknoraRerankModel(item) {
			rerankModels = append(rerankModels, item)
			continue
		}
		if isWeknoraLLMModel(item) {
			llmModels = append(llmModels, item)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"embedding_models": embeddingModels,
			"llm_models":       llmModels,
			"rerank_models":    rerankModels,
			"all_models":       allModels,
		},
	})
}

func extractWeknoraModelOptions(parsed map[string]interface{}) []knowledgeProviderModelOption {
	if len(parsed) == 0 {
		return []knowledgeProviderModelOption{}
	}
	modelMaps := make([]map[string]interface{}, 0, 32)
	collectWeknoraModelMaps(parsed, 0, &modelMaps)
	if len(modelMaps) == 0 {
		return []knowledgeProviderModelOption{}
	}

	seen := make(map[string]struct{}, len(modelMaps))
	options := make([]knowledgeProviderModelOption, 0, len(modelMaps))
	for _, item := range modelMaps {
		id := firstNonEmptyMapString(item, "model_id", "id", "uid")
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		name := strings.TrimSpace(firstNonEmptyMapString(item, "display_name", "name", "model_name", "model_id", "id"))
		if name == "" {
			name = id
		}
		modelType := strings.TrimSpace(firstNonEmptyMapString(item, "model_type", "type", "category", "task_type", "capability"))
		provider := strings.TrimSpace(firstNonEmptyMapString(item, "provider", "vendor"))

		options = append(options, knowledgeProviderModelOption{
			ID:       id,
			Name:     name,
			Type:     modelType,
			Provider: provider,
		})
	}

	sort.SliceStable(options, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(options[i].Name))
		right := strings.ToLower(strings.TrimSpace(options[j].Name))
		if left == right {
			return options[i].ID < options[j].ID
		}
		return left < right
	})
	return options
}

func collectWeknoraModelMaps(raw interface{}, depth int, out *[]map[string]interface{}) {
	if raw == nil || depth > 6 {
		return
	}

	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			collectWeknoraModelMaps(item, depth+1, out)
		}
	case map[string]interface{}:
		if isLikelyWeknoraModelRecord(v) {
			*out = append(*out, v)
		}
		for _, key := range []string{"data", "list", "items", "models", "rows", "records", "results", "model"} {
			if next, ok := v[key]; ok {
				collectWeknoraModelMaps(next, depth+1, out)
			}
		}
	}
}

func isLikelyWeknoraModelRecord(v map[string]interface{}) bool {
	if v == nil {
		return false
	}
	if strings.TrimSpace(firstNonEmptyMapString(v, "model_id")) != "" {
		return true
	}
	if strings.TrimSpace(firstNonEmptyMapString(v, "id")) == "" {
		return false
	}
	if strings.TrimSpace(firstNonEmptyMapString(v, "name", "model_name", "display_name")) != "" {
		return true
	}
	if strings.TrimSpace(firstNonEmptyMapString(v, "model_type", "type", "category", "task_type")) != "" {
		return true
	}
	return false
}

func firstNonEmptyMapString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		raw, ok := m[key]
		if !ok || raw == nil {
			continue
		}
		value := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func isWeknoraEmbeddingModel(model knowledgeProviderModelOption) bool {
	corpus := strings.ToLower(strings.Join([]string{
		model.ID,
		model.Name,
		model.Type,
	}, " "))
	return strings.Contains(corpus, "embedding") || strings.Contains(corpus, "embed")
}

func isWeknoraLLMModel(model knowledgeProviderModelOption) bool {
	if isWeknoraEmbeddingModel(model) {
		return false
	}
	if isWeknoraRerankModel(model) {
		return false
	}

	corpus := strings.ToLower(strings.Join([]string{
		model.ID,
		model.Name,
		model.Type,
	}, " "))
	if strings.Contains(corpus, "tts") || strings.Contains(corpus, "asr") || strings.Contains(corpus, "speech") {
		return false
	}
	if strings.Contains(corpus, "llm") || strings.Contains(corpus, "chat") || strings.Contains(corpus, "reason") {
		return true
	}
	if strings.Contains(corpus, "knowledgeqa") || strings.Contains(corpus, "completion") || strings.Contains(corpus, "generation") {
		return true
	}
	if strings.Contains(corpus, "gpt") || strings.Contains(corpus, "qwen") || strings.Contains(corpus, "deepseek") || strings.Contains(corpus, "glm") || strings.Contains(corpus, "claude") || strings.Contains(corpus, "gemini") {
		return true
	}
	// Use permissive fallback when type is empty to preserve selectability.
	return strings.TrimSpace(model.Type) == ""
}

func isWeknoraRerankModel(model knowledgeProviderModelOption) bool {
	corpus := strings.ToLower(strings.Join([]string{
		model.ID,
		model.Name,
		model.Type,
	}, " "))
	return strings.Contains(corpus, "rerank") || strings.Contains(corpus, "re-rank")
}

func buildWeknoraModelListCandidateEndpoints(baseURL string) []string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return []string{}
	}

	candidates := []string{
		buildWeknoraURL(trimmed, "/models"),
		buildWeknoraURL(trimmed, "/model"),
		trimmed + "/models",
		trimmed + "/model",
	}
	if !strings.HasSuffix(strings.ToLower(trimmed), "/v1") {
		candidates = append(candidates, trimmed+"/v1/models", trimmed+"/v1/model")
	}
	if !strings.HasSuffix(strings.ToLower(trimmed), "/api/v1") {
		candidates = append(candidates, trimmed+"/api/v1/models", trimmed+"/api/v1/model")
	}

	seen := make(map[string]struct{}, len(candidates))
	ret := make([]string, 0, len(candidates))
	for _, item := range candidates {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		ret = append(ret, item)
	}
	return ret
}

func buildKnowledgeGlobalConfigData(db *gorm.DB) gin.H {
	fallback := gin.H{
		"default_provider": "dify",
		"providers":        gin.H{},
	}

	if db == nil {
		return fallback
	}

	var configs []models.Config
	if err := db.Where("type = ? AND enabled = ?", "knowledge_search", true).Find(&configs).Error; err != nil {
		log.Printf("[Knowledge] failed to get knowledge base global config: %v", err)
		return fallback
	}

	selectedByProvider := make(map[string]models.Config)
	for _, cfg := range configs {
		provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
		if provider == "" {
			continue
		}
		prev, exists := selectedByProvider[provider]
		if !exists || (!prev.IsDefault && cfg.IsDefault) {
			selectedByProvider[provider] = cfg
		}
	}

	if len(selectedByProvider) == 0 {
		return fallback
	}

	providerNames := make([]string, 0, len(selectedByProvider))
	for provider := range selectedByProvider {
		providerNames = append(providerNames, provider)
	}
	sort.Strings(providerNames)

	providers := make(gin.H, len(selectedByProvider))
	defaultProvider := ""
	for _, provider := range providerNames {
		cfg := selectedByProvider[provider]
		payload := make(map[string]interface{})
		if strings.TrimSpace(cfg.JsonData) != "" {
			_ = json.Unmarshal([]byte(cfg.JsonData), &payload)
		}
		providers[provider] = payload
		if cfg.IsDefault {
			defaultProvider = provider
		}
	}

	if defaultProvider == "" {
		defaultProvider = providerNames[0]
	}

	return gin.H{
		"default_provider": defaultProvider,
		"providers":        providers,
	}
}

func (uc *UserController) GetKnowledgeBases(c *gin.Context) {
	userID, _ := c.Get("user_id")
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	offset := (page - 1) * pageSize

	var total int64
	if err := uc.DB.Model(&models.KnowledgeBase{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get knowledge base list"})
		return
	}

	var items []models.KnowledgeBase
	if err := uc.DB.Where("user_id = ?", userID).Order("id DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get knowledge base list"})
		return
	}

	type knowledgeBaseDocCountRow struct {
		KnowledgeBaseID uint  `gorm:"column:knowledge_base_id"`
		DocCount        int64 `gorm:"column:doc_count"`
	}
	docCountMap := make(map[uint]int64, len(items))
	if len(items) > 0 {
		kbIDs := make([]uint, 0, len(items))
		for _, item := range items {
			kbIDs = append(kbIDs, item.ID)
		}
		var rows []knowledgeBaseDocCountRow
		if err := uc.DB.Model(&models.KnowledgeBaseDocument{}).
			Select("knowledge_base_id, COUNT(*) AS doc_count").
			Where("knowledge_base_id IN ?", kbIDs).
			Group("knowledge_base_id").
			Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count knowledge base documents"})
			return
		}
		for _, row := range rows {
			docCountMap[row.KnowledgeBaseID] = row.DocCount
		}
	}

	type knowledgeBaseListItem struct {
		models.KnowledgeBase
		DocCount int64 `json:"doc_count"`
	}
	resp := make([]knowledgeBaseListItem, 0, len(items))
	for _, item := range items {
		resp = append(resp, knowledgeBaseListItem{
			KnowledgeBase: item,
			DocCount:      docCountMap[item.ID],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      resp,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"knowledge": buildKnowledgeGlobalConfigData(uc.DB),
	})
}

func (uc *UserController) CreateKnowledgeBase(c *gin.Context) {
	if !ensureKnowledgeFeatureEnabled(c, uc.DB) {
		return
	}
	userID, _ := c.Get("user_id")
	var req struct {
		Name                   string   `json:"name" binding:"required,min=1,max=100"`
		Description            string   `json:"description"`
		Content                string   `json:"content"`
		Status                 string   `json:"status"`
		RetrievalThreshold     *float64 `json:"retrieval_threshold"`
		InheritGlobalThreshold *bool    `json:"inherit_global_threshold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	retrievalThreshold, err := buildKnowledgeRetrievalThreshold(req.InheritGlobalThreshold, req.RetrievalThreshold)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := models.KnowledgeBase{
		UserID:             userID.(uint),
		Name:               req.Name,
		Description:        req.Description,
		Content:            req.Content,
		RetrievalThreshold: retrievalThreshold,
		Status:             req.Status,
		SyncStatus:         knowledgeSyncStatusPending,
		SyncProvider:       resolveDefaultKnowledgeProviderName(uc.DB),
	}
	if err := uc.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create knowledge base"})
		return
	}
	if err := enqueueKnowledgeSyncUpsert(uc.DB, item.ID); err != nil {
		_ = uc.DB.Model(&models.KnowledgeBase{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
			"sync_status": knowledgeSyncStatusFailed,
			"sync_error":  truncateSyncError(err.Error()),
		}).Error
		_ = uc.DB.Where("id = ?", item.ID).First(&item).Error
		c.JSON(http.StatusCreated, gin.H{
			"data":       item,
			"warning":    "knowledge base saved but sync task enqueue failed",
			"sync_error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item, "message": "knowledge base saved, syncing in background"})
}

func (uc *UserController) GetKnowledgeBase(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")
	var item models.KnowledgeBase
	if err := uc.DB.Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (uc *UserController) UpdateKnowledgeBase(c *gin.Context) {
	if !ensureKnowledgeFeatureEnabled(c, uc.DB) {
		return
	}
	userID, _ := c.Get("user_id")
	id := c.Param("id")
	var item models.KnowledgeBase
	if err := uc.DB.Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	var req struct {
		Name                   string   `json:"name" binding:"required,min=1,max=100"`
		Description            string   `json:"description"`
		Content                string   `json:"content"`
		Status                 string   `json:"status"`
		RetrievalThreshold     *float64 `json:"retrieval_threshold"`
		InheritGlobalThreshold *bool    `json:"inherit_global_threshold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}
	item.Name = req.Name
	item.Description = req.Description
	item.Content = req.Content
	if req.Status != "" {
		item.Status = req.Status
	}
	if req.InheritGlobalThreshold != nil || req.RetrievalThreshold != nil {
		retrievalThreshold, err := buildKnowledgeRetrievalThreshold(req.InheritGlobalThreshold, req.RetrievalThreshold)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		item.RetrievalThreshold = retrievalThreshold
	}
	item.SyncStatus = knowledgeSyncStatusPending
	item.SyncError = ""
	if err := uc.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update knowledge base"})
		return
	}
	if err := enqueueKnowledgeSyncUpsert(uc.DB, item.ID); err != nil {
		_ = uc.DB.Model(&models.KnowledgeBase{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
			"sync_status": knowledgeSyncStatusFailed,
			"sync_error":  truncateSyncError(err.Error()),
		}).Error
		_ = uc.DB.Where("id = ?", item.ID).First(&item).Error
		c.JSON(http.StatusOK, gin.H{
			"data":       item,
			"warning":    "knowledge base updated but sync task enqueue failed",
			"sync_error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item, "message": "knowledge base updated, syncing in background"})
}

func (uc *UserController) DeleteKnowledgeBase(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	var item models.KnowledgeBase
	if err := uc.DB.Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	var docs []models.KnowledgeBaseDocument
	if err := uc.DB.Where("knowledge_base_id = ?", item.ID).Find(&docs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query knowledge base documents"})
		return
	}

	err := uc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", id, userID).Delete(&models.KnowledgeBase{}).Error; err != nil {
			return err
		}
		if err := tx.Where("knowledge_base_id = ?", id).Delete(&models.KnowledgeBaseDocument{}).Error; err != nil {
			return err
		}
		return tx.Where("knowledge_base_id = ?", id).Delete(&models.AgentKnowledgeBase{}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete knowledge base"})
		return
	}
	for _, doc := range docs {
		if err := enqueueKnowledgeDocumentSyncDelete(uc.DB, item, doc); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message":    "deleted successfully",
				"warning":    "local deletion successful but some document cleanup tasks failed to enqueue",
				"sync_error": err.Error(),
			})
			return
		}
	}

	if len(docs) == 0 {
		if err := enqueueKnowledgeSyncDelete(uc.DB, item); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message":    "deleted successfully",
				"warning":    "local deletion successful but cleanup task enqueue failed",
				"sync_error": err.Error(),
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully, cleaning up knowledge base data in background"})
}

func (uc *UserController) SyncKnowledgeBase(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	var item models.KnowledgeBase
	if err := uc.DB.Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}

	item.SyncStatus = knowledgeSyncStatusPending
	item.SyncError = ""
	if err := uc.DB.Model(&models.KnowledgeBase{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
		"sync_status": knowledgeSyncStatusPending,
		"sync_error":  "",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update sync status: " + err.Error()})
		return
	}

	if err := enqueueKnowledgeSyncUpsert(uc.DB, item.ID); err != nil {
		_ = uc.DB.Model(&models.KnowledgeBase{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
			"sync_status": knowledgeSyncStatusFailed,
			"sync_error":  truncateSyncError(err.Error()),
		}).Error
		_ = uc.DB.Where("id = ?", item.ID).First(&item).Error
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "sync task enqueue failed: " + err.Error(),
			"data":       item,
			"sync_error": err.Error(),
		})
		return
	}
	_ = uc.DB.Where("id = ?", item.ID).First(&item).Error
	c.JSON(http.StatusAccepted, gin.H{"message": "sync task submitted", "data": item})
}

func (uc *UserController) TestKnowledgeBaseSearch(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDUint := userID.(uint)
	startAt := time.Now()
	kbID, _ := strconv.Atoi(c.Param("id"))
	if kbID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}
	kb, err := uc.getOwnedKnowledgeBase(userIDUint, uint(kbID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Query     string   `json:"query" binding:"required"`
		TopK      int      `json:"top_k"`
		Threshold *float64 `json:"threshold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}

	query := strings.TrimSpace(req.Query)
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query cannot be empty"})
		return
	}
	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}
	if req.Threshold != nil {
		if *req.Threshold < 0 || *req.Threshold > 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "threshold must be between 0 and 1"})
			return
		}
	}

	datasetID := strings.TrimSpace(kb.ExternalKBID)
	if datasetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "knowledge base not yet synced to external provider (external_kb_id is empty)"})
		return
	}

	var docsTotal int64
	var docsSynced int64
	var docsPending int64
	var docsFailed int64
	pendingStatuses := []string{
		knowledgeSyncStatusPending,
		knowledgeSyncStatusUploading,
		knowledgeSyncStatusUploaded,
		knowledgeSyncStatusParsing,
	}
	failedStatuses := []string{
		knowledgeSyncStatusFailed,
		knowledgeSyncStatusUploadFailed,
		knowledgeSyncStatusParseFailed,
	}
	_ = uc.DB.Model(&models.KnowledgeBaseDocument{}).Where("knowledge_base_id = ?", kb.ID).Count(&docsTotal).Error
	_ = uc.DB.Model(&models.KnowledgeBaseDocument{}).Where("knowledge_base_id = ? AND sync_status = ?", kb.ID, knowledgeSyncStatusSynced).Count(&docsSynced).Error
	_ = uc.DB.Model(&models.KnowledgeBaseDocument{}).Where("knowledge_base_id = ? AND sync_status IN ?", kb.ID, pendingStatuses).Count(&docsPending).Error
	_ = uc.DB.Model(&models.KnowledgeBaseDocument{}).Where("knowledge_base_id = ? AND sync_status IN ?", kb.ID, failedStatuses).Count(&docsFailed).Error
	log.Printf(
		"[KnowledgeTest] Start user_id=%d kb_id=%d kb_name=%q sync_provider=%s sync_status=%s dataset_id=%s retrieval_threshold=%s request_threshold=%s docs(total=%d synced=%d pending=%d failed=%d) query=%q top_k=%d",
		userIDUint,
		kb.ID,
		strings.TrimSpace(kb.Name),
		strings.TrimSpace(kb.SyncProvider),
		strings.TrimSpace(kb.SyncStatus),
		datasetID,
		formatKnowledgeThresholdForLog(kb.RetrievalThreshold),
		formatKnowledgeThresholdForLog(req.Threshold),
		docsTotal,
		docsSynced,
		docsPending,
		docsFailed,
		query,
		topK,
	)

	provider, _, providerData, err := resolveKnowledgeProviderForKB(uc.DB, kb)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider = strings.ToLower(strings.TrimSpace(provider))
	log.Printf(
		"[KnowledgeTest] ProviderResolved user_id=%d kb_id=%d resolved_provider=%s kb_sync_provider=%s",
		userIDUint,
		kb.ID,
		provider,
		strings.TrimSpace(kb.SyncProvider),
	)
	client := &http.Client{Timeout: 12 * time.Second}

	var hits []knowledgeSearchTestHit
	switch provider {
	case "dify":
		cfg, err := parseDifyKnowledgeSyncConfig(providerData)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		hits, err = queryKnowledgeTestByDify(client, cfg, req.Threshold, kb.RetrievalThreshold, providerData, datasetID, strings.TrimSpace(kb.Name), query, topK)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "ragflow":
		cfg, err := parseRagflowKnowledgeSyncConfig(providerData)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		hits, err = queryKnowledgeTestByRagflow(client, cfg, req.Threshold, kb.RetrievalThreshold, providerData, datasetID, strings.TrimSpace(kb.Name), query, topK)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "weknora":
		cfg, err := parseWeknoraKnowledgeSyncConfig(providerData)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		hits, err = queryKnowledgeTestByWeknora(client, cfg, req.Threshold, kb.RetrievalThreshold, providerData, datasetID, strings.TrimSpace(kb.Name), query, topK)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("test search not supported for provider %s", provider)})
		return
	}

	log.Printf(
		"[KnowledgeTest] Finish user_id=%d kb_id=%d provider=%s dataset_id=%s retrieval_threshold=%s request_threshold=%s query=%q top_k=%d hits=%d docs(total=%d synced=%d pending=%d failed=%d)",
		userIDUint,
		kb.ID,
		provider,
		datasetID,
		formatKnowledgeThresholdForLog(kb.RetrievalThreshold),
		formatKnowledgeThresholdForLog(req.Threshold),
		query,
		topK,
		len(hits),
		docsTotal,
		docsSynced,
		docsPending,
		docsFailed,
	)
	if len(hits) == 0 {
		log.Printf(
			"[KnowledgeTest] EmptyResultHint kb_id=%d dataset_id=%s provider=%s hint=check that documents are synced and indexed on the external platform before adjusting threshold and query keywords",
			kb.ID,
			datasetID,
			provider,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"knowledge_base_id": kb.ID,
			"provider":          provider,
			"dataset_id":        datasetID,
			"query":             query,
			"top_k":             topK,
			"threshold":         req.Threshold,
			"elapsed_ms":        time.Since(startAt).Milliseconds(),
			"count":             len(hits),
			"hits":              hits,
		},
	})
}

func (uc *UserController) GetKnowledgeBaseDocuments(c *gin.Context) {
	userID, _ := c.Get("user_id")
	kbID, _ := strconv.Atoi(c.Param("id"))
	if kbID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}
	kb, err := uc.getOwnedKnowledgeBase(userID.(uint), uint(kbID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var docs []models.KnowledgeBaseDocument
	if err := uc.DB.Where("knowledge_base_id = ?", kb.ID).Order("id DESC").Find(&docs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get knowledge base documents"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": docs})
}

func (uc *UserController) CreateKnowledgeBaseDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	kbID, _ := strconv.Atoi(c.Param("id"))
	if kbID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}
	kb, err := uc.getOwnedKnowledgeBase(userID.(uint), uint(kbID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Name    string `json:"name" binding:"required,min=1,max=200"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document content cannot be empty"})
		return
	}

	doc, enqueueErr, err := uc.createKnowledgeBaseDocumentRecord(kb.ID, req.Name, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document"})
		return
	}
	if enqueueErr != nil {
		c.JSON(http.StatusCreated, gin.H{
			"data":       doc,
			"warning":    "document saved but sync task enqueue failed",
			"sync_error": enqueueErr.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": doc, "message": "document saved, syncing in background"})
}

func (uc *UserController) CreateKnowledgeBaseDocumentByUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	kbID, _ := strconv.Atoi(c.Param("id"))
	if kbID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}
	kb, err := uc.getOwnedKnowledgeBase(userID.(uint), uint(kbID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	provider, _, _, providerErr := resolveKnowledgeProviderForKB(uc.DB, kb)
	if providerErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": providerErr.Error()})
		return
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != "dify" && provider != "ragflow" && provider != "weknora" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file upload document creation not supported for provider %s", provider)})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "please upload a file (file)"})
		return
	}
	uploadFileName, fileData, err := readKnowledgeUploadFileData(provider, fileHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	content, err := encodeKnowledgeUploadContent(uploadFileName, fileData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to encode uploaded file"})
		return
	}

	docName := buildKnowledgeUploadDocumentName(c.PostForm("name"), fileHeader.Filename)
	doc, enqueueErr, err := uc.createKnowledgeBaseDocumentRecord(kb.ID, docName, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document from uploaded file"})
		return
	}
	if enqueueErr != nil {
		c.JSON(http.StatusCreated, gin.H{
			"data":       doc,
			"warning":    "file uploaded and document created but sync task enqueue failed",
			"sync_error": enqueueErr.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": doc, "message": "file uploaded successfully, document created and async sync submitted"})
}

func (uc *UserController) createKnowledgeBaseDocumentRecord(kbID uint, name, content string) (models.KnowledgeBaseDocument, error, error) {
	doc := models.KnowledgeBaseDocument{
		KnowledgeBaseID: kbID,
		Name:            truncateRunes(strings.TrimSpace(name), 200),
		Content:         content,
		SyncStatus:      knowledgeSyncStatusPending,
	}
	if doc.Name == "" {
		doc.Name = "uploaded document"
	}
	if err := uc.DB.Create(&doc).Error; err != nil {
		return doc, nil, err
	}

	if err := enqueueKnowledgeDocumentSyncUpsert(uc.DB, kbID, doc.ID); err != nil {
		_ = uc.DB.Model(&models.KnowledgeBaseDocument{}).Where("id = ?", doc.ID).Updates(map[string]interface{}{
			"sync_status": knowledgeSyncStatusFailed,
			"sync_error":  truncateSyncError(err.Error()),
		}).Error
		_ = uc.DB.Where("id = ?", doc.ID).First(&doc).Error
		return doc, err, nil
	}
	return doc, nil, nil
}

func (uc *UserController) UpdateKnowledgeBaseDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	kbID, _ := strconv.Atoi(c.Param("id"))
	docID, _ := strconv.Atoi(c.Param("doc_id"))
	if kbID <= 0 || docID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parameters"})
		return
	}
	kb, err := uc.getOwnedKnowledgeBase(userID.(uint), uint(kbID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var doc models.KnowledgeBaseDocument
	if err := uc.DB.Where("id = ? AND knowledge_base_id = ?", docID, kb.ID).First(&doc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	var req struct {
		Name    string `json:"name" binding:"required,min=1,max=200"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document content cannot be empty"})
		return
	}

	doc.Name = strings.TrimSpace(req.Name)
	doc.Content = req.Content
	doc.SyncStatus = knowledgeSyncStatusPending
	doc.SyncError = ""
	if err := uc.DB.Save(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update document"})
		return
	}

	if err := enqueueKnowledgeDocumentSyncUpsert(uc.DB, kb.ID, doc.ID); err != nil {
		_ = uc.DB.Model(&models.KnowledgeBaseDocument{}).Where("id = ?", doc.ID).Updates(map[string]interface{}{
			"sync_status": knowledgeSyncStatusFailed,
			"sync_error":  truncateSyncError(err.Error()),
		}).Error
		_ = uc.DB.Where("id = ?", doc.ID).First(&doc).Error
		c.JSON(http.StatusOK, gin.H{
			"data":       doc,
			"warning":    "document updated but sync task enqueue failed",
			"sync_error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": doc, "message": "document updated, syncing in background"})
}

func (uc *UserController) DeleteKnowledgeBaseDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	kbID, _ := strconv.Atoi(c.Param("id"))
	docID, _ := strconv.Atoi(c.Param("doc_id"))
	if kbID <= 0 || docID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parameters"})
		return
	}
	kb, err := uc.getOwnedKnowledgeBase(userID.(uint), uint(kbID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var doc models.KnowledgeBaseDocument
	if err := uc.DB.Where("id = ? AND knowledge_base_id = ?", docID, kb.ID).First(&doc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	if err := uc.DB.Delete(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete document"})
		return
	}
	if err := enqueueKnowledgeDocumentSyncDelete(uc.DB, *kb, doc); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message":    "deleted successfully",
			"warning":    "local deletion successful but knowledge base document cleanup task enqueue failed",
			"sync_error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully, cleaning up knowledge base document in background"})
}

func (uc *UserController) SyncKnowledgeBaseDocument(c *gin.Context) {
	userID, _ := c.Get("user_id")
	kbID, _ := strconv.Atoi(c.Param("id"))
	docID, _ := strconv.Atoi(c.Param("doc_id"))
	if kbID <= 0 || docID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parameters"})
		return
	}
	kb, err := uc.getOwnedKnowledgeBase(userID.(uint), uint(kbID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	var doc models.KnowledgeBaseDocument
	if err := uc.DB.Where("id = ? AND knowledge_base_id = ?", docID, kb.ID).First(&doc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}
	if strings.TrimSpace(doc.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document content is empty, cannot sync"})
		return
	}

	if err := uc.DB.Model(&models.KnowledgeBaseDocument{}).Where("id = ?", doc.ID).Updates(map[string]interface{}{
		"sync_status": knowledgeSyncStatusPending,
		"sync_error":  "",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update sync status: " + err.Error()})
		return
	}
	if err := enqueueKnowledgeDocumentSyncUpsert(uc.DB, kb.ID, doc.ID); err != nil {
		_ = uc.DB.Model(&models.KnowledgeBaseDocument{}).Where("id = ?", doc.ID).Updates(map[string]interface{}{
			"sync_status": knowledgeSyncStatusFailed,
			"sync_error":  truncateSyncError(err.Error()),
		}).Error
		_ = uc.DB.Where("id = ?", doc.ID).First(&doc).Error
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "sync task enqueue failed: " + err.Error(),
			"data":       doc,
			"sync_error": err.Error(),
		})
		return
	}
	_ = uc.DB.Where("id = ?", doc.ID).First(&doc).Error
	c.JSON(http.StatusAccepted, gin.H{"message": "sync task submitted", "data": doc})
}

func (uc *UserController) GetAgentKnowledgeBases(c *gin.Context) {
	userID, _ := c.Get("user_id")
	agentID, _ := strconv.Atoi(c.Param("id"))
	if agentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}
	if err := uc.assertAgentOwnership(userID.(uint), uint(agentID)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ids, err := uc.listAgentKnowledgeBaseIDs(uint(agentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent knowledge base associations"})
		return
	}
	var items []models.KnowledgeBase
	if len(ids) > 0 {
		if err := uc.DB.Where("id IN ? AND user_id = ?", ids, userID).Find(&items).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get knowledge base details"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"knowledge_base_ids": ids, "knowledge_bases": items}})
}

func (uc *UserController) UpdateAgentKnowledgeBases(c *gin.Context) {
	userID, _ := c.Get("user_id")
	agentID, _ := strconv.Atoi(c.Param("id"))
	if agentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}
	if err := uc.assertAgentOwnership(userID.(uint), uint(agentID)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	var req struct {
		KnowledgeBaseIDs []uint `json:"knowledge_base_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}
	if err := uc.validateKnowledgeBaseOwnership(userID.(uint), req.KnowledgeBaseIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := uc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentKnowledgeBase{}).Error; err != nil {
			return err
		}
		for _, kbID := range uniqueUintSlice(req.KnowledgeBaseIDs) {
			item := models.AgentKnowledgeBase{AgentID: uint(agentID), KnowledgeBaseID: kbID}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update agent knowledge base associations"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated successfully", "data": gin.H{"knowledge_base_ids": uniqueUintSlice(req.KnowledgeBaseIDs)}})
}

func (uc *UserController) getOwnedKnowledgeBase(userID uint, kbID uint) (*models.KnowledgeBase, error) {
	var kb models.KnowledgeBase
	if err := uc.DB.Where("id = ? AND user_id = ?", kbID, userID).First(&kb).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("knowledge base not found")
		}
		return nil, err
	}
	return &kb, nil
}

func (uc *UserController) assertAgentOwnership(userID uint, agentID uint) error {
	var count int64
	if err := uc.DB.Model(&models.Agent{}).Where("id = ? AND user_id = ?", agentID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("agent not found or does not belong to current user")
	}
	return nil
}

func (uc *UserController) validateKnowledgeBaseOwnership(userID uint, knowledgeBaseIDs []uint) error {
	if len(knowledgeBaseIDs) == 0 {
		return nil
	}
	uniqueIDs := uniqueUintSlice(knowledgeBaseIDs)
	var count int64
	if err := uc.DB.Model(&models.KnowledgeBase{}).Where("user_id = ? AND id IN ?", userID, uniqueIDs).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(uniqueIDs)) {
		return fmt.Errorf("contains invalid or unauthorized knowledge base IDs")
	}
	return nil
}

func (uc *UserController) listAgentKnowledgeBaseIDs(agentID uint) ([]uint, error) {
	var links []models.AgentKnowledgeBase
	if err := uc.DB.Where("agent_id = ?", agentID).Order("id ASC").Find(&links).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.KnowledgeBaseID)
	}
	return ids, nil
}

func uniqueUintSlice(values []uint) []uint {
	if len(values) == 0 {
		return []uint{}
	}
	m := make(map[uint]struct{}, len(values))
	ret := make([]uint, 0, len(values))
	for _, v := range values {
		if v == 0 {
			continue
		}
		if _, ok := m[v]; ok {
			continue
		}
		m[v] = struct{}{}
		ret = append(ret, v)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return ret
}

func buildKnowledgeRetrievalThreshold(inherit *bool, value *float64) (*float64, error) {
	useGlobal := value == nil
	if inherit != nil {
		useGlobal = *inherit
	}
	if useGlobal {
		return nil, nil
	}
	if value == nil {
		return nil, fmt.Errorf("please enter a custom retrieval threshold (0~1)")
	}
	v := *value
	if v < 0 || v > 1 {
		return nil, fmt.Errorf("retrieval threshold must be between 0 and 1")
	}
	ret := v
	return &ret, nil
}

func clampKnowledgeThreshold(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func resolveKnowledgeThreshold(requestThreshold *float64, kbThreshold *float64, globalThreshold float64) (float64, string) {
	if requestThreshold != nil {
		return clampKnowledgeThreshold(*requestThreshold), "request"
	}
	if kbThreshold != nil {
		return clampKnowledgeThreshold(*kbThreshold), "kb"
	}
	return clampKnowledgeThreshold(globalThreshold), "global"
}

func formatKnowledgeThresholdForLog(v *float64) string {
	if v == nil {
		return "inherit_global"
	}
	return strconv.FormatFloat(clampKnowledgeThreshold(*v), 'f', 4, 64)
}

func queryKnowledgeTestByDify(
	client *http.Client,
	cfg *difyKnowledgeSyncConfig,
	requestThreshold *float64,
	kbThreshold *float64,
	providerData map[string]interface{},
	datasetID, datasetName, query string,
	topK int,
) ([]knowledgeSearchTestHit, error) {
	scoreThreshold, thresholdSource := resolveKnowledgeThreshold(
		requestThreshold,
		kbThreshold,
		parseKnowledgeSearchFloat(providerData["score_threshold"], 0.2),
	)
	payload := map[string]interface{}{
		"query": strings.TrimSpace(query),
		"retrieval_model": map[string]interface{}{
			"top_k":                   topK,
			"score_threshold":         scoreThreshold,
			"score_threshold_enabled": scoreThreshold > 0,
			"search_method":           "semantic_search",
			"reranking_enable":        false,
		},
	}
	path := fmt.Sprintf("/datasets/%s/retrieve", url.PathEscape(datasetID))
	log.Printf(
		"[KnowledgeTest][Dify] RetrieveRequest dataset_id=%s query=%q top_k=%d score_threshold=%.4f threshold_enabled=%t threshold_source=%s",
		datasetID,
		strings.TrimSpace(query),
		topK,
		scoreThreshold,
		scoreThreshold > 0,
		thresholdSource,
	)
	var resp struct {
		Records []struct {
			Score   float64 `json:"score"`
			Segment struct {
				Content string `json:"content"`
			} `json:"segment"`
		} `json:"records"`
		Data struct {
			Records []struct {
				Score   float64 `json:"score"`
				Segment struct {
					Content string `json:"content"`
				} `json:"segment"`
			} `json:"records"`
		} `json:"data"`
	}
	statusCode, bodyBytes, err := doDifyJSONRequest(client, http.MethodPost, buildDifyURL(cfg.BaseURL, path), cfg.APIKey, payload, &resp)
	if err != nil {
		return nil, fmt.Errorf("Dify retrieval failed (dataset_id=%s): %w", datasetID, err)
	}

	title := strings.TrimSpace(datasetName)
	if title == "" {
		title = datasetID
	}
	hits := make([]knowledgeSearchTestHit, 0, len(resp.Records)+len(resp.Data.Records))
	appendRecord := func(score float64, content string) {
		content = strings.TrimSpace(content)
		if content == "" {
			return
		}
		hits = append(hits, knowledgeSearchTestHit{
			Title:   title,
			Score:   score,
			Content: content,
		})
	}
	for _, record := range resp.Records {
		appendRecord(record.Score, record.Segment.Content)
	}
	if len(hits) == 0 {
		for _, record := range resp.Data.Records {
			appendRecord(record.Score, record.Segment.Content)
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > topK {
		hits = hits[:topK]
	}
	log.Printf(
		"[KnowledgeTest][Dify] RetrieveParsed dataset_id=%s status=%d records=%d data_records=%d hits=%d",
		datasetID,
		statusCode,
		len(resp.Records),
		len(resp.Data.Records),
		len(hits),
	)
	if len(hits) == 0 {
		log.Printf(
			"[KnowledgeTest][Dify] EmptyBody dataset_id=%s status=%d body=%s",
			datasetID,
			statusCode,
			truncateForLog(string(bodyBytes), 1500),
		)
	}
	return hits, nil
}

func queryKnowledgeTestByRagflow(
	client *http.Client,
	cfg *ragflowKnowledgeSyncConfig,
	requestThreshold *float64,
	kbThreshold *float64,
	providerData map[string]interface{},
	datasetID, datasetName, query string,
	topK int,
) ([]knowledgeSearchTestHit, error) {
	similarityThreshold, thresholdSource := resolveKnowledgeThreshold(
		requestThreshold,
		kbThreshold,
		parseKnowledgeSearchFloat(providerData["similarity_threshold"], 0.2),
	)
	vectorSimilarityWeight := parseKnowledgeSearchFloat(providerData["vector_similarity_weight"], 0.3)
	if vectorSimilarityWeight <= 0 {
		vectorSimilarityWeight = 0.3
	}
	payload := map[string]interface{}{
		"question":                 strings.TrimSpace(query),
		"dataset_ids":              []string{datasetID},
		"top_k":                    topK,
		"page":                     1,
		"page_size":                topK,
		"similarity_threshold":     similarityThreshold,
		"vector_similarity_weight": vectorSimilarityWeight,
		"keyword":                  parseKnowledgeSearchBool(providerData["keyword"], false),
		"highlight":                parseKnowledgeSearchBool(providerData["highlight"], false),
	}
	log.Printf(
		"[KnowledgeTest][Ragflow] RetrieveRequest dataset_id=%s query=%q top_k=%d similarity_threshold=%.4f vector_similarity_weight=%.4f keyword=%t highlight=%t threshold_source=%s",
		datasetID,
		strings.TrimSpace(query),
		topK,
		similarityThreshold,
		vectorSimilarityWeight,
		parseKnowledgeSearchBool(providerData["keyword"], false),
		parseKnowledgeSearchBool(providerData["highlight"], false),
		thresholdSource,
	)

	var resp struct {
		Data struct {
			Chunks []struct {
				Content          string  `json:"content"`
				Highlight        string  `json:"highlight"`
				Similarity       float64 `json:"similarity"`
				VectorSimilarity float64 `json:"vector_similarity"`
				DocumentName     string  `json:"document_name"`
			} `json:"chunks"`
		} `json:"data"`
	}
	statusCode, bodyBytes, err := doRagflowJSONRequest(client, http.MethodPost, buildRagflowURL(cfg.BaseURL, "/retrieval"), cfg.APIKey, payload, &resp)
	if err != nil {
		return nil, fmt.Errorf("RAGFlow retrieval failed (dataset_id=%s): %w", datasetID, err)
	}

	title := strings.TrimSpace(datasetName)
	if title == "" {
		title = datasetID
	}
	hits := make([]knowledgeSearchTestHit, 0, len(resp.Data.Chunks))
	for _, chunk := range resp.Data.Chunks {
		content := strings.TrimSpace(chunk.Content)
		if highlight := strings.TrimSpace(chunk.Highlight); highlight != "" {
			content = highlight
		}
		if content == "" {
			continue
		}
		score := chunk.Similarity
		if score <= 0 {
			score = chunk.VectorSimilarity
		}
		chunkTitle := title
		if chunkTitle == "" {
			chunkTitle = strings.TrimSpace(chunk.DocumentName)
		}
		if chunkTitle == "" {
			chunkTitle = datasetID
		}
		hits = append(hits, knowledgeSearchTestHit{
			Title:   chunkTitle,
			Score:   score,
			Content: content,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > topK {
		hits = hits[:topK]
	}
	log.Printf(
		"[KnowledgeTest][Ragflow] RetrieveParsed dataset_id=%s status=%d chunks=%d hits=%d",
		datasetID,
		statusCode,
		len(resp.Data.Chunks),
		len(hits),
	)
	if len(hits) == 0 {
		log.Printf(
			"[KnowledgeTest][Ragflow] EmptyBody dataset_id=%s status=%d body=%s",
			datasetID,
			statusCode,
			truncateForLog(string(bodyBytes), 1500),
		)
	}
	return hits, nil
}

func queryKnowledgeTestByWeknora(
	client *http.Client,
	cfg *weknoraKnowledgeSyncConfig,
	requestThreshold *float64,
	kbThreshold *float64,
	providerData map[string]interface{},
	datasetID string,
	datasetName, query string,
	topK int,
) ([]knowledgeSearchTestHit, error) {
	scoreThreshold, thresholdSource := resolveKnowledgeThreshold(
		requestThreshold,
		kbThreshold,
		parseKnowledgeSearchFloat(providerData["score_threshold"], 0.2),
	)
	payload := map[string]interface{}{
		"query":              strings.TrimSpace(query),
		"knowledge_base_ids": []string{strings.TrimSpace(datasetID)},
	}
	log.Printf(
		"[KnowledgeTest][Weknora] RetrieveRequest dataset_id=%s knowledge_base_ids=1 query=%q top_k=%d score_threshold=%.4f threshold_source=%s threshold_filter=disabled",
		datasetID,
		strings.TrimSpace(query),
		topK,
		scoreThreshold,
		thresholdSource,
	)

	var resp struct {
		Data []struct {
			Content           string                 `json:"content"`
			KnowledgeTitle    string                 `json:"knowledge_title"`
			Score             float64                `json:"score"`
			Similarity        float64                `json:"similarity"`
			Metadata          map[string]interface{} `json:"metadata"`
			KnowledgeMetadata map[string]interface{} `json:"knowledge_metadata"`
		} `json:"data"`
	}
	statusCode, bodyBytes, err := doWeknoraJSONRequest(client, http.MethodPost, buildWeknoraURL(cfg.BaseURL, "/knowledge-search"), cfg.APIKey, payload, &resp)
	if err != nil {
		return nil, fmt.Errorf("Weknora retrieval failed (dataset_id=%s): %w", datasetID, err)
	}

	title := strings.TrimSpace(datasetName)
	if title == "" {
		title = datasetID
	}
	hits := make([]knowledgeSearchTestHit, 0, len(resp.Data))
	for _, item := range resp.Data {
		content := strings.TrimSpace(item.Content)
		if content == "" && item.KnowledgeMetadata != nil {
			if chunkText, ok := item.KnowledgeMetadata["chunk_text"]; ok {
				content = strings.TrimSpace(fmt.Sprintf("%v", chunkText))
			}
		}
		if content == "" || content == "<nil>" {
			continue
		}

		score := item.Score
		if score <= 0 {
			score = item.Similarity
		}
		if score <= 0 && item.Metadata != nil {
			score = parseKnowledgeSearchFloat(item.Metadata["score"], 0)
		}

		chunkTitle := strings.TrimSpace(item.KnowledgeTitle)
		if chunkTitle == "" {
			chunkTitle = title
		}
		hits = append(hits, knowledgeSearchTestHit{
			Title:   chunkTitle,
			Score:   score,
			Content: content,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > topK {
		hits = hits[:topK]
	}
	log.Printf(
		"[KnowledgeTest][Weknora] RetrieveParsed dataset_id=%s status=%d records=%d hits=%d",
		datasetID,
		statusCode,
		len(resp.Data),
		len(hits),
	)
	if len(hits) == 0 {
		log.Printf(
			"[KnowledgeTest][Weknora] EmptyBody dataset_id=%s status=%d body=%s",
			datasetID,
			statusCode,
			truncateForLog(string(bodyBytes), 1500),
		)
	}
	return hits, nil
}

func parseKnowledgeSearchFloat(input interface{}, defaultValue float64) float64 {
	switch v := input.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func normalizeKnowledgeIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	ret := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ret = append(ret, id)
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}

func parseKnowledgeSearchBool(input interface{}, defaultValue bool) bool {
	switch v := input.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return defaultValue
}

func readKnowledgeUploadFileData(provider string, fileHeader *multipart.FileHeader) (string, []byte, error) {
	if fileHeader == nil {
		return "", nil, fmt.Errorf("uploaded file cannot be empty")
	}
	if fileHeader.Size > knowledgeDocumentUploadMaxBytes {
		return "", nil, fmt.Errorf("file too large, maximum supported size is %dMB", knowledgeDocumentUploadMaxBytes/(1024*1024))
	}

	fileName := sanitizeKnowledgeUploadFileName(fileHeader.Filename)
	ext := strings.ToLower(filepath.Ext(fileName))
	allowedExtMap, supportedText := getAllowedKnowledgeUploadExtByProvider(provider)
	if ext == "" {
		return "", nil, fmt.Errorf("unsupported file type: missing extension; %s supported formats: %s", strings.ToUpper(provider), supportedText)
	}
	if _, ok := allowedExtMap[ext]; !ok {
		return "", nil, fmt.Errorf("unsupported file type; %s supported formats: %s", strings.ToUpper(provider), supportedText)
	}

	f, err := fileHeader.Open()
	if err != nil {
		return "", nil, fmt.Errorf("failed to read uploaded file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, knowledgeDocumentUploadMaxBytes+1))
	if err != nil {
		return "", nil, fmt.Errorf("failed to read uploaded file: %w", err)
	}
	if int64(len(data)) > knowledgeDocumentUploadMaxBytes {
		return "", nil, fmt.Errorf("file too large, maximum supported size is %dMB", knowledgeDocumentUploadMaxBytes/(1024*1024))
	}
	if len(data) == 0 {
		return "", nil, fmt.Errorf("uploaded file is empty")
	}
	return fileName, data, nil
}

func getAllowedKnowledgeUploadExtByProvider(provider string) (map[string]struct{}, string) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "dify":
		return allowedKnowledgeDifyFileExt, "txt, md, markdown, pdf, html, htm, xlsx, xls, docx, csv, eml, msg, pptx, ppt, xml, epub"
	case "ragflow":
		return allowedKnowledgeRagflowFileExt, "txt, text, md, markdown, pdf, doc, docx, ppt, pptx, xls, xlsx, wps, json, csv, log, xml, html, htm, yml, yaml, rtf, sql, ini, jpg, jpeg, png, gif, bmp, webp, tif, tiff, eml, msg"
	case "weknora":
		return allowedKnowledgeWeknoraFileExt, "txt, text, md, markdown, pdf, doc, docx, ppt, pptx, xls, xlsx, wps, json, csv, log, xml, html, htm, yml, yaml, rtf, sql, ini, jpg, jpeg, png, gif, bmp, webp, tif, tiff, eml, msg"
	default:
		return allowedKnowledgeRagflowFileExt, "txt, md, pdf, docx etc."
	}
}

func buildKnowledgeUploadDocumentName(inputName, fileName string) string {
	name := strings.TrimSpace(inputName)
	if name == "" {
		name = strings.TrimSpace(fileName)
		if ext := filepath.Ext(name); ext != "" {
			name = strings.TrimSpace(strings.TrimSuffix(name, ext))
		}
	}
	if name == "" {
		name = "uploaded document"
	}
	return truncateRunes(name, 200)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func sanitizeKnowledgeUploadFileName(fileName string) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		return "document.txt"
	}
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "\n", "_")
	name = strings.ReplaceAll(name, "\r", "_")
	if name == "" {
		return "document.txt"
	}
	return name
}

func encodeKnowledgeUploadContent(fileName string, fileData []byte) (string, error) {
	payload := map[string]string{
		"file_name":      sanitizeKnowledgeUploadFileName(fileName),
		"content_base64": base64.StdEncoding.EncodeToString(fileData),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return knowledgeUploadContentPrefix + string(b), nil
}

func decodeKnowledgeUploadContent(content string) (string, []byte, bool, error) {
	raw := strings.TrimSpace(content)
	if !strings.HasPrefix(raw, knowledgeUploadContentPrefix) {
		return "", nil, false, nil
	}

	jsonPart := strings.TrimSpace(strings.TrimPrefix(raw, knowledgeUploadContentPrefix))
	if jsonPart == "" {
		return "", nil, true, fmt.Errorf("uploaded file metadata is empty")
	}

	var payload struct {
		FileName      string `json:"file_name"`
		ContentBase64 string `json:"content_base64"`
	}
	if err := json.Unmarshal([]byte(jsonPart), &payload); err != nil {
		return "", nil, true, fmt.Errorf("failed to parse uploaded file metadata: %w", err)
	}
	payload.FileName = sanitizeKnowledgeUploadFileName(payload.FileName)
	if strings.TrimSpace(payload.ContentBase64) == "" {
		return "", nil, true, fmt.Errorf("uploaded file content is empty")
	}
	fileData, err := base64.StdEncoding.DecodeString(payload.ContentBase64)
	if err != nil {
		return "", nil, true, fmt.Errorf("failed to decode uploaded file content: %w", err)
	}
	if len(fileData) == 0 {
		return "", nil, true, fmt.Errorf("uploaded file content is empty")
	}
	return payload.FileName, fileData, true, nil
}

func (ac *AdminController) GetUserKnowledgeBasesAdmin(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))
	if userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	var items []models.KnowledgeBase
	if err := ac.DB.Where("user_id = ?", userID).Order("id DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get knowledge base list"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (ac *AdminController) CreateUserKnowledgeBaseAdmin(c *gin.Context) {
	if !ensureKnowledgeFeatureEnabled(c, ac.DB) {
		return
	}
	userID, _ := strconv.Atoi(c.Param("id"))
	if userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	var req struct {
		Name                   string   `json:"name" binding:"required,min=1,max=100"`
		Description            string   `json:"description"`
		Content                string   `json:"content"`
		Status                 string   `json:"status"`
		RetrievalThreshold     *float64 `json:"retrieval_threshold"`
		InheritGlobalThreshold *bool    `json:"inherit_global_threshold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	retrievalThreshold, err := buildKnowledgeRetrievalThreshold(req.InheritGlobalThreshold, req.RetrievalThreshold)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := models.KnowledgeBase{
		UserID:             uint(userID),
		Name:               req.Name,
		Description:        req.Description,
		Content:            req.Content,
		RetrievalThreshold: retrievalThreshold,
		Status:             req.Status,
		SyncStatus:         knowledgeSyncStatusPending,
		SyncProvider:       resolveDefaultKnowledgeProviderName(ac.DB),
	}
	if err := ac.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create knowledge base"})
		return
	}
	if err := enqueueKnowledgeSyncUpsert(ac.DB, item.ID); err != nil {
		_ = ac.DB.Model(&models.KnowledgeBase{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
			"sync_status": knowledgeSyncStatusFailed,
			"sync_error":  truncateSyncError(err.Error()),
		}).Error
		_ = ac.DB.Where("id = ?", item.ID).First(&item).Error
		c.JSON(http.StatusCreated, gin.H{
			"data":       item,
			"warning":    "knowledge base saved but sync task enqueue failed",
			"sync_error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item, "message": "knowledge base saved, syncing in background"})
}

func (ac *AdminController) UpdateUserKnowledgeBaseAdmin(c *gin.Context) {
	if !ensureKnowledgeFeatureEnabled(c, ac.DB) {
		return
	}
	userID, _ := strconv.Atoi(c.Param("id"))
	kbID, _ := strconv.Atoi(c.Param("kb_id"))
	if userID <= 0 || kbID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parameters"})
		return
	}
	var item models.KnowledgeBase
	if err := ac.DB.Where("id = ? AND user_id = ?", kbID, userID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	var req struct {
		Name                   string   `json:"name" binding:"required,min=1,max=100"`
		Description            string   `json:"description"`
		Content                string   `json:"content"`
		Status                 string   `json:"status"`
		RetrievalThreshold     *float64 `json:"retrieval_threshold"`
		InheritGlobalThreshold *bool    `json:"inherit_global_threshold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}
	item.Name = req.Name
	item.Description = req.Description
	item.Content = req.Content
	if req.Status != "" {
		item.Status = req.Status
	}
	if req.InheritGlobalThreshold != nil || req.RetrievalThreshold != nil {
		retrievalThreshold, err := buildKnowledgeRetrievalThreshold(req.InheritGlobalThreshold, req.RetrievalThreshold)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		item.RetrievalThreshold = retrievalThreshold
	}
	item.SyncStatus = knowledgeSyncStatusPending
	item.SyncError = ""
	if err := ac.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update knowledge base"})
		return
	}
	if err := enqueueKnowledgeSyncUpsert(ac.DB, item.ID); err != nil {
		_ = ac.DB.Model(&models.KnowledgeBase{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
			"sync_status": knowledgeSyncStatusFailed,
			"sync_error":  truncateSyncError(err.Error()),
		}).Error
		_ = ac.DB.Where("id = ?", item.ID).First(&item).Error
		c.JSON(http.StatusOK, gin.H{
			"data":       item,
			"warning":    "knowledge base updated but sync task enqueue failed",
			"sync_error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item, "message": "knowledge base updated, syncing in background"})
}

func (ac *AdminController) DeleteUserKnowledgeBaseAdmin(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))
	kbID, _ := strconv.Atoi(c.Param("kb_id"))
	if userID <= 0 || kbID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parameters"})
		return
	}

	var item models.KnowledgeBase
	if err := ac.DB.Where("id = ? AND user_id = ?", kbID, userID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	var docs []models.KnowledgeBaseDocument
	if err := ac.DB.Where("knowledge_base_id = ?", item.ID).Find(&docs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query knowledge base documents"})
		return
	}

	err := ac.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", kbID, userID).Delete(&models.KnowledgeBase{}).Error; err != nil {
			return err
		}
		if err := tx.Where("knowledge_base_id = ?", kbID).Delete(&models.KnowledgeBaseDocument{}).Error; err != nil {
			return err
		}
		return tx.Where("knowledge_base_id = ?", kbID).Delete(&models.AgentKnowledgeBase{}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete knowledge base"})
		return
	}
	for _, doc := range docs {
		if err := enqueueKnowledgeDocumentSyncDelete(ac.DB, item, doc); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message":    "deleted successfully",
				"warning":    "local deletion successful but some document cleanup tasks failed to enqueue",
				"sync_error": err.Error(),
			})
			return
		}
	}

	if len(docs) == 0 {
		if err := enqueueKnowledgeSyncDelete(ac.DB, item); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message":    "deleted successfully",
				"warning":    "local deletion successful but cleanup task enqueue failed",
				"sync_error": err.Error(),
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully, cleaning up knowledge base data in background"})
}
