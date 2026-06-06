package coze_llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	llm_common "xiaozhi-esp32-server-golang/internal/domain/llm/common"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	sse "github.com/tmaxmax/go-sse"
)

const (
	defaultCozeBaseURL  = "https://api.coze.com"
	defaultConnectorID  = "1024"
	defaultUserPrefix   = "xiaozhi"
	llmExtraErrorKey    = "error"
	streamCreatePath    = "/v3/chat"
	maxIdleConns        = 200
	maxIdleConnsPerHost = 50
	idleConnTimeout     = 90 * time.Second
	dialTimeout         = 30 * time.Second
	keepAliveTimeout    = 30 * time.Second
)

var (
	httpClientOnce sync.Once
	httpClientInst *http.Client
)

type CozeLLMProvider struct {
	apiKey      string
	baseURL     string
	botID       string
	userPrefix  string
	connectorID string
	httpClient  *http.Client

	conversationMu  sync.RWMutex
	conversationIDs map[string]string
}

type cozeMessage struct {
	Role        string `json:"role"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

type cozeCreateChatRequest struct {
	BotID              string         `json:"bot_id"`
	UserID             string         `json:"user_id"`
	Stream             bool           `json:"stream"`
	ConversationID     string         `json:"conversation_id,omitempty"`
	ConnectorID        string         `json:"connector_id,omitempty"`
	AdditionalMessages []*cozeMessage `json:"additional_messages,omitempty"`
}

type cozeStreamEvent struct {
	Event          string            `json:"event"`
	Message        *cozeEventMessage `json:"message,omitempty"`
	Chat           *cozeEventChat    `json:"chat,omitempty"`
	Code           int               `json:"code,omitempty"`
	Msg            string            `json:"msg,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
}

type cozeEventMessage struct {
	Content string `json:"content"`
}

type cozeEventChat struct {
	LastError      *cozeLastError `json:"last_error,omitempty"`
	ConversationID string         `json:"conversation_id,omitempty"`
}

type cozeLastError struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: keepAliveTimeout,
			}).DialContext,
			MaxIdleConns:        maxIdleConns,
			MaxIdleConnsPerHost: maxIdleConnsPerHost,
			IdleConnTimeout:     idleConnTimeout,
			DisableKeepAlives:   false,
		}
		httpClientInst = &http.Client{
			Transport: transport,
			Timeout:   0,
		}
	})
	return httpClientInst
}

func NewCozeLLMProvider(config map[string]interface{}) (*CozeLLMProvider, error) {
	apiKey, _ := config["api_key"].(string)
	apiKey = normalizeAPIToken(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("coze api_key cannot be empty")
	}

	botID, _ := config["bot_id"].(string)
	botID = strings.TrimSpace(botID)
	if botID == "" {
		return nil, fmt.Errorf("coze bot_id cannot be empty")
	}

	baseURL, _ := config["base_url"].(string)
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultCozeBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	userPrefix, _ := config["user_prefix"].(string)
	userPrefix = strings.TrimSpace(userPrefix)
	if userPrefix == "" {
		userPrefix = defaultUserPrefix
	}

	connectorID, _ := config["connector_id"].(string)
	connectorID = strings.TrimSpace(connectorID)
	if connectorID == "" {
		connectorID = defaultConnectorID
	}

	return &CozeLLMProvider{
		apiKey:          apiKey,
		baseURL:         baseURL,
		botID:           botID,
		userPrefix:      userPrefix,
		connectorID:     connectorID,
		httpClient:      getHTTPClient(),
		conversationIDs: make(map[string]string),
	}, nil
}

func (p *CozeLLMProvider) ResponseWithContext(ctx context.Context, sessionID string, dialogue []*schema.Message, _ []*schema.ToolInfo) chan *schema.Message {
	out := make(chan *schema.Message, 200)

	go func() {
		defer close(out)

		query := buildCozeQuery(dialogue)
		if strings.TrimSpace(query) == "" {
			sendLLMError(out, fmt.Errorf("coze query cannot be empty"))
			return
		}

		conversationID := p.getConversationID(sessionID)
		reqBody := cozeCreateChatRequest{
			BotID:          p.botID,
			UserID:         llm_common.BuildStableUserID(p.userPrefix, sessionID),
			Stream:         true,
			ConversationID: conversationID,
			ConnectorID:    p.connectorID,
			AdditionalMessages: []*cozeMessage{
				{
					Role:        "user",
					Type:        "question",
					Content:     query,
					ContentType: "text",
				},
			},
		}
		reqBody.Stream = true

		requestBodies, err := buildCozeRequestBodies(reqBody)
		if err != nil {
			sendLLMError(out, err)
			return
		}

		var resp *http.Response
		var openErr error
		for i, currentBody := range requestBodies {
			resp, openErr = p.openStreamRequest(ctx, currentBody)
			if openErr == nil {
				break
			}
			if i == 0 && len(requestBodies) > 1 {
				log.Warnf("Coze's first request failed, try to roll back and try again: %v", openErr)
			}
		}
		if openErr != nil {
			sendLLMError(out, openErr)
			return
		}
		defer resp.Body.Close()

		seenDelta := false
		for event, eventErr := range sse.Read(resp.Body, nil) {
			if eventErr != nil {
				if ctx.Err() != nil {
					return
				}
				if seenDelta && strings.Contains(strings.ToLower(eventErr.Error()), "unexpected end of input") {
					//Some Coze instances will disconnect directly after the last event, tolerating this scenario.
					return
				}
				sendLLMError(out, fmt.Errorf("Failed to read coze stream: %w", eventErr))
				return
			}

			eventType := strings.TrimSpace(event.Type)
			if strings.EqualFold(eventType, "done") {
				return
			}

			data := normalizeCozeStreamData(event.Data)
			if data == "" {
				continue
			}
			if isCozeDoneMarker(data) {
				return
			}

			var streamEvent cozeStreamEvent
			_ = json.Unmarshal([]byte(data), &streamEvent)
			if cid := extractCozeConversationID(streamEvent, data); cid != "" {
				p.setConversationID(sessionID, cid)
			}
			if eventType == "" {
				eventType = strings.TrimSpace(streamEvent.Event)
			}

			switch eventType {
			case "conversation.message.delta":
				content := extractCozeMessageContent(data, streamEvent)
				if content != "" {
					seenDelta = true
					out <- &schema.Message{
						Role:    schema.Assistant,
						Content: content,
					}
				}
			case "conversation.message.completed":
				content := extractCozeMessageContent(data, streamEvent)
				if content != "" && !seenDelta {
					out <- &schema.Message{
						Role:    schema.Assistant,
						Content: content,
					}
				}
				seenDelta = seenDelta || content != ""
			case "conversation.chat.completed", "done":
				return
			case "conversation.chat.failed", "error":
				sendLLMError(out, errors.New(extractCozeError(streamEvent, data)))
				return
			default:
				// Fallback for payloads where event name is not stable but content exists.
				content := extractCozeMessageContent(data, streamEvent)
				if content != "" {
					seenDelta = true
					out <- &schema.Message{
						Role:    schema.Assistant,
						Content: content,
					}
				}
			}
		}
	}()

	return out
}

func (p *CozeLLMProvider) openStreamRequest(ctx context.Context, bodyBytes []byte) (*http.Response, error) {
	url := p.baseURL + streamCreatePath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("coze request failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()
		return nil, fmt.Errorf("coze request failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "text/event-stream") {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()
		return nil, fmt.Errorf(
			"coze response is not an SSE stream path=%s content_type=%s body=%s",
			streamCreatePath,
			contentType,
			strings.TrimSpace(string(errBody)),
		)
	}

	return resp, nil
}

func buildCozeRequestBodies(req cozeCreateChatRequest) ([][]byte, error) {
	candidates := make([]cozeCreateChatRequest, 0, 4)
	candidates = append(candidates, req)

	if strings.TrimSpace(req.ConnectorID) != "" {
		reqNoConnector := req
		reqNoConnector.ConnectorID = ""
		candidates = append(candidates, reqNoConnector)
	}

	if strings.TrimSpace(req.ConversationID) != "" {
		reqNoConversation := req
		reqNoConversation.ConversationID = ""
		candidates = append(candidates, reqNoConversation)

		if strings.TrimSpace(reqNoConversation.ConnectorID) != "" {
			reqNoConversationNoConnector := reqNoConversation
			reqNoConversationNoConnector.ConnectorID = ""
			candidates = append(candidates, reqNoConversationNoConnector)
		}
	}

	uniqueBodies := make([][]byte, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate.Stream = true
		bodyBytes, err := json.Marshal(candidate)
		if err != nil {
			return nil, err
		}
		key := string(bodyBytes)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		uniqueBodies = append(uniqueBodies, bodyBytes)
	}
	return uniqueBodies, nil
}

func (p *CozeLLMProvider) getConversationID(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ""
	}
	p.conversationMu.RLock()
	defer p.conversationMu.RUnlock()
	return p.conversationIDs[sessionID]
}

func (p *CozeLLMProvider) setConversationID(sessionID, conversationID string) {
	sessionID = strings.TrimSpace(sessionID)
	conversationID = strings.TrimSpace(conversationID)
	if sessionID == "" || conversationID == "" {
		return
	}
	p.conversationMu.Lock()
	defer p.conversationMu.Unlock()
	p.conversationIDs[sessionID] = conversationID
}

func buildCozeQuery(dialogue []*schema.Message) string {
	if len(dialogue) == 0 {
		return ""
	}

	//Coze session mode only sends the current round of user input and does not splice local history.
	for i := len(dialogue) - 1; i >= 0; i-- {
		msg := dialogue[i]
		if msg == nil || msg.Role != schema.User {
			continue
		}
		if text := extractCozeQueryText(msg); text != "" {
			return text
		}
	}

	//Bottom line: If there is no user message, fall back to the last message from which text can be extracted.
	for i := len(dialogue) - 1; i >= 0; i-- {
		if text := extractCozeQueryText(dialogue[i]); text != "" {
			return text
		}
	}

	return ""
}

func extractCozeQueryText(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	if text := strings.TrimSpace(msg.Content); text != "" {
		return text
	}
	if len(msg.MultiContent) == 0 {
		return ""
	}

	parts := make([]string, 0, len(msg.MultiContent))
	for _, part := range msg.MultiContent {
		if text := strings.TrimSpace(part.Text); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

func extractCozeConversationID(event cozeStreamEvent, data string) string {
	if cid := strings.TrimSpace(event.ConversationID); cid != "" {
		return cid
	}
	if event.Chat != nil {
		if cid := strings.TrimSpace(event.Chat.ConversationID); cid != "" {
			return cid
		}
	}
	payload, ok := parseJSONMap(data)
	if !ok {
		return ""
	}
	if cid := extractString(payload["conversation_id"]); cid != "" {
		return cid
	}
	if chat, ok := payload["chat"].(map[string]any); ok {
		if cid := extractString(chat["conversation_id"]); cid != "" {
			return cid
		}
	}
	if nestedData, ok := payload["data"]; ok {
		switch v := nestedData.(type) {
		case string:
			normalized := normalizeCozeStreamData(v)
			if normalized != "" && normalized != data {
				if cid := extractCozeConversationID(cozeStreamEvent{}, normalized); cid != "" {
					return cid
				}
			}
		case map[string]any:
			if cid := extractString(v["conversation_id"]); cid != "" {
				return cid
			}
			if chat, ok := v["chat"].(map[string]any); ok {
				if cid := extractString(chat["conversation_id"]); cid != "" {
					return cid
				}
			}
		}
	}
	return ""
}

func extractCozeError(event cozeStreamEvent, data string) string {
	if event.Chat != nil && event.Chat.LastError != nil && event.Chat.LastError.Msg != "" {
		return event.Chat.LastError.Msg
	}
	if event.Msg != "" {
		return event.Msg
	}
	if payload, ok := parseJSONMap(data); ok {
		if msg := extractString(payload["msg"]); msg != "" {
			return msg
		}
		if msg := extractString(payload["message"]); msg != "" {
			return msg
		}
		if chat, ok := payload["chat"].(map[string]any); ok {
			if lastError, ok := chat["last_error"].(map[string]any); ok {
				if msg := extractString(lastError["msg"]); msg != "" {
					return msg
				}
			}
		}
		if lastError, ok := payload["last_error"].(map[string]any); ok {
			if msg := extractString(lastError["msg"]); msg != "" {
				return msg
			}
		}
		if nestedData, ok := payload["data"]; ok {
			switch v := nestedData.(type) {
			case string:
				normalized := normalizeCozeStreamData(v)
				if normalized != "" && normalized != data {
					if nestedMsg := extractCozeError(cozeStreamEvent{}, normalized); nestedMsg != "Coze returned an error" {
						return nestedMsg
					}
				}
			case map[string]any:
				if msg := extractString(v["msg"]); msg != "" {
					return msg
				}
				if msg := extractString(v["message"]); msg != "" {
					return msg
				}
			}
		}
	}
	return "Coze returned an error"
}

func extractCozeMessageContent(data string, event cozeStreamEvent) string {
	if event.Message != nil && strings.TrimSpace(event.Message.Content) != "" {
		return strings.TrimSpace(event.Message.Content)
	}

	payload, ok := parseJSONMap(data)
	if !ok {
		return ""
	}

	if content := extractString(payload["content"]); content != "" {
		return content
	}
	if msg, ok := payload["message"].(map[string]any); ok {
		if content := extractString(msg["content"]); content != "" {
			return content
		}
	}
	if delta, ok := payload["delta"].(map[string]any); ok {
		if content := extractString(delta["content"]); content != "" {
			return content
		}
	}
	if nestedData, ok := payload["data"]; ok {
		switch v := nestedData.(type) {
		case string:
			normalized := normalizeCozeStreamData(v)
			if normalized != "" && normalized != data {
				if content := extractCozeMessageContent(normalized, cozeStreamEvent{}); content != "" {
					return content
				}
			}
		case map[string]any:
			if content := extractString(v["content"]); content != "" {
				return content
			}
			if msg, ok := v["message"].(map[string]any); ok {
				if content := extractString(msg["content"]); content != "" {
					return content
				}
			}
			if delta, ok := v["delta"].(map[string]any); ok {
				if content := extractString(delta["content"]); content != "" {
					return content
				}
			}
		}
	}

	return ""
}

func normalizeCozeStreamData(data string) string {
	data = strings.TrimSpace(data)
	if data == "" {
		return ""
	}

	for i := 0; i < 2; i++ {
		var decoded string
		if err := json.Unmarshal([]byte(data), &decoded); err != nil {
			break
		}
		data = strings.TrimSpace(decoded)
	}
	return data
}

func isCozeDoneMarker(data string) bool {
	d := strings.TrimSpace(data)
	return strings.EqualFold(d, "[DONE]") || strings.EqualFold(d, "done")
}

func parseJSONMap(data string) (map[string]any, bool) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func extractString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func (p *CozeLLMProvider) ResponseWithVllm(_ context.Context, _ []byte, _ string, _ string) (string, error) {
	return "", fmt.Errorf("coze provider does not support vllm capability")
}

func (p *CozeLLMProvider) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"type":         "coze",
		"provider":     "coze",
		"base_url":     p.baseURL,
		"bot_id":       p.botID,
		"user_prefix":  p.userPrefix,
		"connector_id": p.connectorID,
	}
}

func (p *CozeLLMProvider) Close() error {
	return nil
}

func (p *CozeLLMProvider) IsValid() bool {
	return p != nil && p.apiKey != "" && p.botID != ""
}

func sendLLMError(ch chan *schema.Message, err error) {
	ch <- &schema.Message{
		Role:  schema.System,
		Extra: map[string]any{llmExtraErrorKey: err.Error()},
	}
}

func previewString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func normalizeAPIToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) >= 7 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	return token
}
