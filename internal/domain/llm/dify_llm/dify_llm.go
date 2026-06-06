package dify_llm

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
	defaultDifyBaseURL = "https://api.dify.ai/v1"
	defaultUserPrefix  = "xiaozhi"
	llmExtraErrorKey   = "error"

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

type DifyLLMProvider struct {
	apiKey     string
	baseURL    string
	userPrefix string
	httpClient *http.Client

	conversationMu  sync.RWMutex
	conversationIDs map[string]string
}

type difyChatRequest struct {
	Inputs         map[string]interface{} `json:"inputs"`
	Query          string                 `json:"query"`
	ResponseMode   string                 `json:"response_mode"`
	User           string                 `json:"user"`
	ConversationID string                 `json:"conversation_id,omitempty"`
}

type difyStopRequest struct {
	User string `json:"user"`
}

type difyStreamEvent struct {
	Event          string `json:"event"`
	TaskID         string `json:"task_id"`
	ConversationID string `json:"conversation_id"`
	Answer         string `json:"answer"`
	Message        string `json:"message"`
	Code           string `json:"code"`
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

func NewDifyLLMProvider(config map[string]interface{}) (*DifyLLMProvider, error) {
	apiKey, _ := config["api_key"].(string)
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("dify api_key cannot be empty")
	}

	baseURL, _ := config["base_url"].(string)
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultDifyBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	// The Dify API uses /v1 by default.
	if !strings.HasSuffix(strings.ToLower(baseURL), "/v1") {
		baseURL += "/v1"
	}

	userPrefix, _ := config["user_prefix"].(string)
	userPrefix = strings.TrimSpace(userPrefix)
	if userPrefix == "" {
		userPrefix = defaultUserPrefix
	}

	return &DifyLLMProvider{
		apiKey:          apiKey,
		baseURL:         baseURL,
		userPrefix:      userPrefix,
		httpClient:      getHTTPClient(),
		conversationIDs: make(map[string]string),
	}, nil
}

func (p *DifyLLMProvider) ResponseWithContext(ctx context.Context, sessionID string, dialogue []*schema.Message, _ []*schema.ToolInfo) chan *schema.Message {
	out := make(chan *schema.Message, 200)

	go func() {
		defer close(out)

		query := buildDifyQuery(dialogue)
		if strings.TrimSpace(query) == "" {
			sendLLMError(out, fmt.Errorf("dify query cannot be empty"))
			return
		}

		userID := llm_common.BuildStableUserID(p.userPrefix, sessionID)
		conversationID := p.getConversationID(sessionID)
		reqBody := difyChatRequest{
			Inputs:       map[string]interface{}{},
			Query:        query,
			ResponseMode: "streaming",
			User:         userID,
		}
		if conversationID != "" {
			reqBody.ConversationID = conversationID
		}
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			sendLLMError(out, err)
			return
		}

		url := p.baseURL + "/chat-messages"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			sendLLMError(out, err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			sendLLMError(out, fmt.Errorf("dify request failed: %w", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			sendLLMError(out, fmt.Errorf("dify request failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(errBody))))
			return
		}

		var taskID string
		for event, eventErr := range sse.Read(resp.Body, nil) {
			if eventErr != nil {
				if ctx.Err() != nil {
					break
				}
				sendLLMError(out, fmt.Errorf("Failed to read dify stream: %w", eventErr))
				return
			}

			data := strings.TrimSpace(event.Data)
			if data == "" {
				continue
			}
			if data == "[DONE]" {
				return
			}

			var streamEvent difyStreamEvent
			if err := json.Unmarshal([]byte(data), &streamEvent); err != nil {
				log.Warnf("Failed to parse the differentiate stream event: %v, data=%s", err, previewString(data, 256))
				continue
			}

			if streamEvent.TaskID != "" {
				taskID = streamEvent.TaskID
			}
			if streamEvent.ConversationID != "" {
				p.setConversationID(sessionID, streamEvent.ConversationID)
			}

			switch streamEvent.Event {
			case "error":
				msg := streamEvent.Message
				if msg == "" {
					msg = "Dify returned an error"
				}
				sendLLMError(out, errors.New(msg))
				return
			case "message", "agent_message":
				if streamEvent.Answer != "" {
					out <- &schema.Message{
						Role:    schema.Assistant,
						Content: streamEvent.Answer,
					}
				}
			case "message_end":
				return
			default:
				// Some providers only carry textual chunks and no stable event name.
				if streamEvent.Answer != "" {
					out <- &schema.Message{
						Role:    schema.Assistant,
						Content: streamEvent.Answer,
					}
				}
			}
		}

		if ctx.Err() != nil && taskID != "" {
			p.stopTask(taskID, userID)
		}
	}()

	return out
}

func (p *DifyLLMProvider) stopTask(taskID, userID string) {
	stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	bodyBytes, err := json.Marshal(difyStopRequest{User: userID})
	if err != nil {
		return
	}

	url := fmt.Sprintf("%s/chat-messages/%s/stop", p.baseURL, taskID)
	req, err := http.NewRequestWithContext(stopCtx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Debugf("dify stop task request failed: %v", err)
		return
	}
	defer resp.Body.Close()
}

func (p *DifyLLMProvider) getConversationID(sessionID string) string {
	if strings.TrimSpace(sessionID) == "" {
		return ""
	}
	p.conversationMu.RLock()
	defer p.conversationMu.RUnlock()
	return p.conversationIDs[sessionID]
}

func (p *DifyLLMProvider) setConversationID(sessionID, conversationID string) {
	sessionID = strings.TrimSpace(sessionID)
	conversationID = strings.TrimSpace(conversationID)
	if sessionID == "" || conversationID == "" {
		return
	}
	p.conversationMu.Lock()
	defer p.conversationMu.Unlock()
	p.conversationIDs[sessionID] = conversationID
}

func (p *DifyLLMProvider) ResponseWithVllm(_ context.Context, _ []byte, _ string, _ string) (string, error) {
	return "", fmt.Errorf("dify provider does not support vllm capability")
}

func buildDifyQuery(dialogue []*schema.Message) string {
	if len(dialogue) == 0 {
		return ""
	}

	//In Dify session mode, only the current round of input is sent, and the history is not spliced into the query.
	for i := len(dialogue) - 1; i >= 0; i-- {
		msg := dialogue[i]
		if msg == nil || msg.Role != schema.User {
			continue
		}
		if text := extractDifyMessageText(msg); text != "" {
			return text
		}
	}

	//Bottom line: If there is no user message, use the last message from which text can be extracted.
	for i := len(dialogue) - 1; i >= 0; i-- {
		if text := extractDifyMessageText(dialogue[i]); text != "" {
			return text
		}
	}

	return ""
}

func extractDifyMessageText(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	if text := strings.TrimSpace(msg.Content); text != "" {
		return text
	}
	if len(msg.MultiContent) > 0 {
		parts := make([]string, 0, len(msg.MultiContent))
		for _, part := range msg.MultiContent {
			if text := strings.TrimSpace(part.Text); text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

func (p *DifyLLMProvider) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"type":        "dify",
		"provider":    "dify",
		"base_url":    p.baseURL,
		"user_prefix": p.userPrefix,
	}
}

func (p *DifyLLMProvider) Close() error {
	return nil
}

func (p *DifyLLMProvider) IsValid() bool {
	return p != nil && p.apiKey != "" && p.baseURL != ""
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
