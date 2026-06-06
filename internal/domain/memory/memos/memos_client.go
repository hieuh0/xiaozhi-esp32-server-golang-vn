package memos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	log "xiaozhi-esp32-server-golang/logger"
)

const (
	defaultBaseURL   = "https://memos.memtensor.cn/api/openmem/v1"
	defaultTimeoutMS = 10000
)

// Client is a MemOS independent provider client implementation.
type Client struct {
	baseURL         string
	apiKey          string
	httpClient      *http.Client
	enableSearch    bool
	searchTopK      int
	searchThreshold float64
}

// GetWithConfig Initializes the MemOS client with configuration.
// Actual request URL = base_url + fixed path
func GetWithConfig(config map[string]interface{}) (*Client, error) {
	if config == nil {
		config = map[string]interface{}{}
	}

	baseURL := getString(config, "base_url", defaultBaseURL)
	apiKey := getString(config, "api_key", "")
	timeoutMS := getInt(config, "timeout_ms", defaultTimeoutMS)
	enableSearch := getBool(config, "enable_search", true)
	searchTopK := getInt(config, "search_top_k", 3)
	searchThreshold := getFloat(config, "search_threshold", 0.5)

	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("memos.base_url configuration is missing or empty")
	}
	if searchTopK <= 0 {
		searchTopK = 3
	}
	if timeoutMS <= 0 {
		timeoutMS = defaultTimeoutMS
	}

	client := &Client{
		baseURL:         strings.TrimRight(baseURL, "/"),
		apiKey:          strings.TrimSpace(apiKey),
		httpClient:      &http.Client{Timeout: time.Duration(timeoutMS) * time.Millisecond},
		enableSearch:    enableSearch,
		searchTopK:      searchTopK,
		searchThreshold: searchThreshold,
	}

	log.Log().Infof("MemOS client initialization successful, base_url: %s", client.baseURL)
	return client, nil
}

func (c *Client) AddMessage(ctx context.Context, agentID string, msg schema.Message) error {
	payload, err := c.newIdentityPayload(agentID)
	if err != nil {
		return err
	}
	payload["messages"] = []map[string]string{{
		"role":    string(msg.Role),
		"content": msg.Content,
	}}
	_, err = c.requestJSON(ctx, http.MethodPost, "/add/message", payload)
	if err != nil {
		return fmt.Errorf("memos add_message failed: %w", err)
	}
	return nil
}

func (c *Client) GetMessages(ctx context.Context, agentID string, count int) ([]*schema.Message, error) {
	if count <= 0 {
		count = 20
	}
	payload, err := c.newIdentityPayload(agentID)
	if err != nil {
		return nil, err
	}
	payload["limit"] = count
	data, err := c.requestJSON(ctx, http.MethodPost, "/get/messages", payload)
	if err != nil {
		return nil, fmt.Errorf("memos get_messages failed: %w", err)
	}

	msgsRaw := getArrayField(data, "messages", "message_list", "items")
	if len(msgsRaw) == 0 {
		return []*schema.Message{}, nil
	}

	messages := make([]*schema.Message, 0, len(msgsRaw))
	for _, item := range msgsRaw {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		role := schema.Assistant
		if r, ok := obj["role"].(string); ok {
			switch strings.ToLower(r) {
			case "user":
				role = schema.User
			case "assistant":
				role = schema.Assistant
			case "system":
				role = schema.System
			}
		}
		content := getStringFromMap(obj, "content", "memory", "text")
		messages = append(messages, &schema.Message{Role: role, Content: content})
	}
	return messages, nil
}

func (c *Client) GetContext(ctx context.Context, agentID string, maxToken int) (string, error) {
	if !c.enableSearch {
		return "", nil
	}
	return c.Search(ctx, agentID, "", c.searchTopK, 0)
}

func (c *Client) Search(ctx context.Context, agentID string, query string, topK int, timeRangeDays int64) (string, error) {
	if !c.enableSearch {
		return "", nil
	}
	if topK <= 0 {
		topK = c.searchTopK
	}
	payload, err := c.newIdentityPayload(agentID)
	if err != nil {
		return "", err
	}
	payload["query"] = query
	payload["memory_limit_number"] = topK
	payload["relativity"] = c.searchThreshold
	_ = timeRangeDays //The document currently does not have a time_range_days field, retaining signature compatibility
	data, err := c.requestJSON(ctx, http.MethodPost, "/search/memory", payload)
	if err != nil {
		return "", fmt.Errorf("memos search failed: %w", err)
	}
	items := getSearchItems(data)
	if len(items) == 0 {
		return "", nil
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		text := getStringFromMap(obj, "memory_value", "content", "memory", "text")
		if text != "" {
			lines = append(lines, "- "+text)
		}
	}
	return strings.Join(lines, "\n"), nil
}

func (c *Client) Flush(ctx context.Context, agentID string) error {
	payload, err := c.newIdentityPayload(agentID)
	if err != nil {
		return err
	}
	_, err = c.requestJSON(ctx, http.MethodPost, "/flush", payload)
	if err != nil {
		return fmt.Errorf("memos flush failed: %w", err)
	}
	return nil
}

func (c *Client) ResetMemory(ctx context.Context, agentID string) error {
	payload, err := c.newIdentityPayload(agentID)
	if err != nil {
		return err
	}
	_, err = c.requestJSON(ctx, http.MethodPost, "/reset/memory", payload)
	if err != nil {
		return fmt.Errorf("memos reset_memory failed: %w", err)
	}
	return nil
}

func (c *Client) newIdentityPayload(agentID string) (map[string]interface{}, error) {
	identity := strings.TrimSpace(agentID)
	if identity == "" {
		return nil, fmt.Errorf("agentID is empty: user_id and conversation_id are required by MemOS")
	}

	payload := map[string]interface{}{
		"user_id":         identity,
		"conversation_id": identity,
	}

	//agent_id is an optional field in the MemOS document and is only passed if there is a value
	if identity != "" {
		payload["agent_id"] = identity
	}
	return payload, nil
}

func (c *Client) requestJSON(ctx context.Context, method, path string, payload map[string]interface{}) (map[string]interface{}, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d, body=%s", resp.StatusCode, string(respBytes))
	}
	if len(respBytes) == 0 {
		return map[string]interface{}{}, nil
	}

	var out map[string]interface{}
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("invalid json response: %w", err)
	}

	if data, ok := out["data"].(map[string]interface{}); ok {
		return data, nil
	}
	if result, ok := out["result"].(map[string]interface{}); ok {
		return result, nil
	}
	return out, nil
}

func getString(config map[string]interface{}, key, defaultValue string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

func getInt(config map[string]interface{}, key string, defaultValue int) int {
	if v, ok := config[key]; ok {
		switch value := v.(type) {
		case int:
			return value
		case int32:
			return int(value)
		case int64:
			return int(value)
		case float64:
			return int(value)
		}
	}
	return defaultValue
}

func getFloat(config map[string]interface{}, key string, defaultValue float64) float64 {
	if v, ok := config[key]; ok {
		switch value := v.(type) {
		case float64:
			return value
		case float32:
			return float64(value)
		case int:
			return float64(value)
		case int64:
			return float64(value)
		}
	}
	return defaultValue
}

func getBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := config[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func getSearchItems(data map[string]interface{}) []interface{} {
	keys := []string{
		"memory_detail_list",
		"preference_detail_list",
		"tool_memory_detail_list",
		"skill_detail_list",
		"results",
		"memories",
		"items",
	}
	merged := make([]interface{}, 0)
	for _, key := range keys {
		if arr, ok := data[key].([]interface{}); ok && len(arr) > 0 {
			merged = append(merged, arr...)
		}
	}
	return merged
}

func getArrayField(data map[string]interface{}, keys ...string) []interface{} {
	for _, key := range keys {
		if arr, ok := data[key].([]interface{}); ok {
			return arr
		}
	}
	return nil
}

func getStringFromMap(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key].(string); ok && val != "" {
			return val
		}
	}
	return ""
}
