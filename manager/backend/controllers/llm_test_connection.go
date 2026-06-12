package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// TestLLMConnection tests connectivity to an LLM provider by making a minimal API call directly from the manager.
// No WebSocket relay needed — manager calls the LLM API directly.
func (ac *AdminController) TestLLMConnection(c *gin.Context) {
	var body struct {
		Type    string `json:"type"`
		APIKey  string `json:"api_key"`
		BaseURL string `json:"base_url"`
		Model   string `json:"model_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid request: type is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	var ok bool
	var errMsg string

	switch body.Type {
	case "ollama":
		ok, errMsg = testOllamaReachable(ctx, body.BaseURL)
	case "dify":
		ok, errMsg = testDifyReachable(ctx, body.BaseURL, body.APIKey)
	case "coze":
		ok, errMsg = testCozeReachable(ctx, body.BaseURL, body.APIKey)
	default: // openai and all compatible providers
		ok, errMsg = testOpenAIReachable(ctx, body.BaseURL, body.APIKey, body.Model)
	}

	c.JSON(http.StatusOK, gin.H{"ok": ok, "error": errMsg})
}

func testOpenAIReachable(ctx context.Context, baseURL, apiKey, model string) (bool, string) {
	if model == "" {
		return false, "model_name is required"
	}
	baseURL = normalizeBaseURL(baseURL, "https://api.openai.com/v1")

	payload, _ := json.Marshal(map[string]interface{}{
		"model":      model,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
		"max_tokens": 1,
		"stream":     false,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewBuffer(payload))
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, ""
	}
	return false, parseOpenAIError(resp.StatusCode, body)
}

func parseOpenAIError(statusCode int, body []byte) string {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", statusCode, errResp.Error.Message)
	}
	return fmt.Sprintf("HTTP %d: %s", statusCode, truncateForLog(string(body), 200))
}

func testOllamaReachable(ctx context.Context, baseURL string) (bool, string) {
	baseURL = normalizeBaseURL(baseURL, "http://localhost:11434")

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/tags", nil)
	if err != nil {
		return false, err.Error()
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, ""
	}
	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

func testDifyReachable(ctx context.Context, baseURL, apiKey string) (bool, string) {
	baseURL = normalizeBaseURL(baseURL, "https://api.dify.ai/v1")

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/info", nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, ""
	}
	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncateForLog(string(body), 200))
}

func testCozeReachable(ctx context.Context, baseURL, apiKey string) (bool, string) {
	baseURL = normalizeBaseURL(baseURL, "https://api.coze.com")

	// GET /v1/space/list is a lightweight auth check for Coze
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/v1/space/list?page_num=1&page_size=1", nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	// Accept any non-5xx as "reachable" — 400 still means auth worked
	if resp.StatusCode < 500 {
		if resp.StatusCode == 401 {
			return false, "unauthorized: invalid API key"
		}
		return true, ""
	}
	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncateForLog(string(body), 200))
}

// FetchLLMModels fetches available model IDs from the provider's models endpoint.
func (ac *AdminController) FetchLLMModels(c *gin.Context) {
	var body struct {
		Type    string `json:"type"`
		APIKey  string `json:"api_key"`
		BaseURL string `json:"base_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	var models []string
	var errMsg string

	switch body.Type {
	case "ollama":
		models, errMsg = fetchOllamaModels(ctx, body.BaseURL)
	default: // openai-compatible
		models, errMsg = fetchOpenAIModels(ctx, body.BaseURL, body.APIKey)
	}

	if errMsg != "" {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": errMsg, "models": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "models": models})
}

func fetchOpenAIModels(ctx context.Context, baseURL, apiKey string) ([]string, string) {
	baseURL = normalizeBaseURL(baseURL, "https://api.openai.com/v1")
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
	if err != nil {
		return nil, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err.Error()
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, parseOpenAIError(resp.StatusCode, respBody)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, "failed to parse models response"
	}
	ids := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		ids = append(ids, m.ID)
	}
	sort.Strings(ids)
	return ids, ""
}

func fetchOllamaModels(ctx context.Context, baseURL string) ([]string, string) {
	baseURL = normalizeBaseURL(baseURL, "http://localhost:11434")
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err.Error()
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err.Error()
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, "failed to parse ollama models"
	}
	names := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		names = append(names, m.Name)
	}
	sort.Strings(names)
	return names, ""
}

func normalizeBaseURL(url, defaultURL string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return defaultURL
	}
	return strings.TrimRight(url, "/")
}
