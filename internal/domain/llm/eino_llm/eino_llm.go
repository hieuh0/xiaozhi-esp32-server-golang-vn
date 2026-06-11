package eino_llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	log "xiaozhi-esp32-server-golang/logger"
)

// EinoLLMProvider LLM provider based on Eino framework
// Directly use Eino's ChatModel interface and type, supporting openai and ollama
type EinoLLMProvider struct {
	chatModel        model.ToolCallingChatModel
	modelName        string
	maxTokens        int
	streamable       bool
	config           map[string]interface{}
	providerType     string //"openai" or "ollama"
	reasoningTracker *reasoningContentTracker
}

// EinoConfig Eino LLM configuration
type EinoConfig struct {
	Type       string                 `json:"type"` //"openai" or "ollama"
	ModelName  string                 `json:"model_name"`
	APIKey     string                 `json:"api_key"`
	BaseURL    string                 `json:"base_url"`
	MaxTokens  int                    `json:"max_tokens"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Streamable bool                   `json:"streamable,omitempty"`
}

// Connection pool configuration
const (
	maxIdleConns          = 200
	maxIdleConnsPerHost   = 50
	idleConnTimeout       = 90 * time.Second
	dialTimeout           = 30 * time.Second
	keepAliveTimeout      = 30 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 60 * time.Second
)

// Global HTTP client, used for all OpenAI requests
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

// getHTTPClient returns the HTTP client configured with a connection pool
func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: keepAliveTimeout,
			}).DialContext,
			MaxIdleConns:          maxIdleConns,
			MaxIdleConnsPerHost:   maxIdleConnsPerHost,
			IdleConnTimeout:       idleConnTimeout,
			TLSHandshakeTimeout:   tlsHandshakeTimeout,
			ResponseHeaderTimeout: responseHeaderTimeout,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     false,
		}

		httpClient = &http.Client{
			Transport: transport,
			//In streaming output scenarios, do not use http.Client.Timeout to truncate the entire connection. Instead, use ctx to control the request life cycle.
			Timeout: 0,
		}
	})

	return httpClient
}

// NewEinoLLMProvider creates a new Eino LLM provider, supporting openai and ollama according to type
func NewEinoLLMProvider(config map[string]interface{}) (*EinoLLMProvider, error) {
	//log.Debugf("NewEinoLLMProvider config: %+v", config)
	var tracker *reasoningContentTracker
	if enabled, _ := config[reasoningDetectConfigKey].(bool); enabled {
		tracker = &reasoningContentTracker{}
		config[reasoningTrackerConfigKey] = tracker
	}
	parsedConfig, err := decodeOpenAICompatibleConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse LLM configuration: %v", err)
	}

	providerType := parsedConfig.Type
	if providerType == "" {
		return nil, fmt.Errorf("type cannot be empty and must be 'openai' or 'ollama'")
	}

	modelName := parsedConfig.ModelName
	if modelName == "" {
		return nil, fmt.Errorf("model_name cannot be empty")
	}

	maxTokens := 500
	if parsedConfig.MaxTokens != nil {
		maxTokens = *parsedConfig.MaxTokens
	}

	streamable := true
	if parsedConfig.Streamable != nil {
		streamable = *parsedConfig.Streamable
	}

	var chatModel model.ToolCallingChatModel

	//Create different ChatModel implementations based on type
	switch providerType {
	case "openai":
		chatModel, err = createOpenAIChatModel(config)
		if err != nil {
			return nil, fmt.Errorf("Failed to create OpenAI ChatModel: %v", err)
		}
	case "ollama":
		chatModel, err = createOllamaChatModel(config)
		if err != nil {
			return nil, fmt.Errorf("Failed to create Ollama ChatModel: %v", err)
		}
	default:
		return nil, fmt.Errorf("Unsupported model type: %s", providerType)
	}

	provider := &EinoLLMProvider{
		chatModel:        chatModel,
		modelName:        modelName,
		maxTokens:        maxTokens,
		streamable:       streamable,
		config:           config,
		providerType:     providerType,
		reasoningTracker: tracker,
	}

	return provider, nil
}

func (p *EinoLLMProvider) HasReasoningContent() bool {
	return p != nil && p.reasoningTracker != nil && p.reasoningTracker.HasReturned()
}

// createOpenAIChatModel creates OpenAI's ChatModel implementation
func createOpenAIChatModel(config map[string]interface{}) (model.ToolCallingChatModel, error) {
	ctx := context.Background()

	parsedConfig, err := decodeOpenAICompatibleConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse OpenAI compatible configuration: %v", err)
	}

	modelName := parsedConfig.ModelName
	if modelName == "" {
		modelName = "gpt-3.5-turbo"
	}

	apiKey := parsedConfig.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	httpClient := buildThinkingHTTPClient(config, getHTTPClient())
	useMaxCompletionTokens := shouldUseMaxCompletionTokens(parsedConfig.Provider, modelName)

	//Create OpenAI ChatModel configuration
	openaiConfig := &openai.ChatModelConfig{
		Model:      modelName,
		APIKey:     apiKey,
		HTTPClient: httpClient,
	}

	if parsedConfig.BaseURL != "" {
		openaiConfig.BaseURL = parsedConfig.BaseURL
	}
	if parsedConfig.APIVersion != "" {
		openaiConfig.APIVersion = parsedConfig.APIVersion
	}
	if !useMaxCompletionTokens && parsedConfig.MaxTokens != nil && *parsedConfig.MaxTokens > 0 {
		openaiConfig.MaxTokens = parsedConfig.MaxTokens
	}
	if parsedConfig.Temperature != nil {
		openaiConfig.Temperature = parsedConfig.Temperature
	}
	if parsedConfig.TopP != nil {
		openaiConfig.TopP = parsedConfig.TopP
	}

	log.Debugf("openaiConfig: %+v", openaiConfig)

	//Implemented using eino-ext official OpenAI
	chatModel, err := openai.NewChatModel(ctx, openaiConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to create OpenAI ChatModel: %v", err)
	}

	log.Infof("OpenAI ChatModel was successfully created, model: %s", modelName)
	return chatModel, nil
}

// createOllamaChatModel creates Ollama's ChatModel implementation
func createOllamaChatModel(config map[string]interface{}) (model.ToolCallingChatModel, error) {
	ctx := context.Background()

	modelName, _ := config["model_name"].(string)
	baseURL, _ := config["base_url"].(string)

	if modelName == "" || baseURL == "" {
		log.Warnf("model_name and base_url cannot be empty, use the default model: %s", modelName)
		return nil, fmt.Errorf("model_name and base_url cannot be empty")
	}

	//Create Ollama ChatModel configuration
	ollamaConfig := &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
	}

	//Implemented using eino-ext official Ollama
	chatModel, err := ollama.NewChatModel(ctx, ollamaConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Ollama ChatModel: %v", err)
	}

	log.Infof("Ollama ChatModel was successfully created, model: %s", modelName)
	return chatModel, nil
}

// GetModelInfo gets model information
func (p *EinoLLMProvider) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"model_name":      p.modelName,
		"max_tokens":      p.maxTokens,
		"streamable":      p.streamable,
		"type":            "eino",
		"provider_type":   p.providerType,
		"framework":       "eino",
		"adapter_version": "3.0.0",
		"base_url":        p.config["base_url"],
	}
}

// ResponseWithFunctions Response with function call, using Eino native tool type, directly calling EinoResponseWithTools
func (p *EinoLLMProvider) ResponseWithContext(ctx context.Context, sessionID string, dialogue []*schema.Message, functions []*schema.ToolInfo) chan *schema.Message {

	log.Infof("[Eino-LLM] Start processing requests with tools - SessionID: %s, Type: %s", sessionID, p.providerType)

	logMessages(dialogue)
	//Directly call EinoResponseWithTools to obtain Eino native response
	einoResponseChan := p.EinoResponseWithTools(ctx, sessionID, dialogue, functions)

	log.Infof("[Eino-LLM] Tool call request processing completed - SessionID: %s", sessionID)

	return einoResponseChan
}

func logMessages(messages []*schema.Message) {
	for _, msg := range messages {
		if msg == nil {
			log.Debugf("history llm msg: <nil>")
			continue
		}
		log.Debugf("history llm msg: %s\n", msg.String())
	}
}

// llmExtraErrorKey is consistent with domain/llm.LLMExtraErrorKey, and is used to transparently transmit errors in case of failure (to avoid circular dependencies)
const llmExtraErrorKey = "error"

// sendLLMError sends an error message with Extra.error to the channel
func sendLLMError(ch chan *schema.Message, err error) {
	ch <- &schema.Message{
		Role:  schema.System,
		Extra: map[string]any{llmExtraErrorKey: err.Error()},
	}
}

// EinoResponseWithTools directly uses the Eino type response with tools
func (p *EinoLLMProvider) EinoResponseWithTools(ctx context.Context, sessionID string, messages []*schema.Message, tools []*schema.ToolInfo) chan *schema.Message {
	responseChan := make(chan *schema.Message, 200)

	var err error
	go func() {
		defer close(responseChan)
		if p.reasoningTracker != nil {
			p.reasoningTracker.Reset()
		}

		log.Infof("[Eino-LLM] Start processing Eino tool requests - SessionID: %s, tools: %+v", sessionID, tools)

		//If there is a tool, you need to bind the tool to the ChatModel
		if len(tools) > 0 {
			p.chatModel, err = p.chatModel.WithTools(tools)
			if err != nil {
				log.Errorf("Binding tool failed: %v", err)
				sendLLMError(responseChan, err)
				return
			}
		}

		if p.streamable {
			log.Debugf("EinoLLMProvider.EinoResponseWithTools() streamable: %t", p.streamable)
			//Directly use Eino's Stream method
			streamReader, err := p.chatModel.Stream(ctx, messages, p.buildModelCallOptions()...)
			if err != nil {
				log.Errorf("Eino tool streaming call failed: %v", err)
				//For mock implementation, if Stream fails, fall back to Generate
				message, genErr := p.chatModel.Generate(ctx, messages, p.buildModelCallOptions()...)
				if genErr != nil {
					log.Errorf("Eino tool failed to generate response: %v", genErr)
					sendLLMError(responseChan, genErr)
					return
				}
				if message != nil {
					responseChan <- message
				}
				return
			}

			if streamReader != nil {
				defer streamReader.Close()

				var currentToolCall *schema.ToolCall
				var toolCallBuffer string
				var isToolCallComplete bool
				var streamChunkCount int

				//Handling streaming responses
				for {
					message, err := streamReader.Recv()
					//log.Debugf("streamReader.Recv() message: %+v", message)
					if err == io.EOF {
						if streamChunkCount == 0 {
							sendLLMError(responseChan, errors.New("Streaming response is empty"))
							break
						}
						//If there are outstanding tool calls, send the last
						if currentToolCall != nil {
							completeMessage := &schema.Message{
								Role:      schema.Assistant,
								ToolCalls: []schema.ToolCall{*currentToolCall},
							}
							responseChan <- completeMessage
						}
						break
					}
					if err != nil {
						if ctxErr := ctx.Err(); ctxErr != nil {
							if errors.Is(ctxErr, context.Canceled) {
								log.Debugf("Streaming response canceled: %v", ctxErr)
							} else {
								log.Warnf("Streaming response ended: %v", ctxErr)
							}
							break
						}
						log.Errorf("Failed to receive streaming response: %v", err)
						sendLLMError(responseChan, err)
						break
					}

					if message != nil {
						streamChunkCount++
						//Check if this is the beginning of a tool call
						if len(message.ToolCalls) > 0 {
							toolCall := message.ToolCalls[0]

							if toolCall.Function.Name != "" {
								//New tool call starts
								currentToolCall = &toolCall
								toolCallBuffer = toolCall.Function.Arguments
								isToolCallComplete = false
							} else if currentToolCall != nil {
								//Cumulative tool call parameters
								toolCallBuffer += toolCall.Function.Arguments
								currentToolCall.Function.Arguments = toolCallBuffer

								//Check if the parameter is a complete JSON
								if isValidJSON(toolCallBuffer) {
									isToolCallComplete = true
								}
							}

							//If the tool call is complete, send a message
							if isToolCallComplete {
								completeMessage := &schema.Message{
									Role:      schema.Assistant,
									ToolCalls: []schema.ToolCall{*currentToolCall},
								}
								responseChan <- completeMessage

								//reset state
								currentToolCall = nil
								toolCallBuffer = ""
								isToolCallComplete = false
							}
						} else if message.Content != "" {
							//Send ordinary messages that are not tool calls
							message.ToolCalls = nil
							responseChan <- message
						}
					}
				}
			} else {
				sendLLMError(responseChan, errors.New("Streaming response is empty"))
			}
		} else {
			//Directly use Eino's Generate method
			message, err := p.chatModel.Generate(ctx, messages, p.buildModelCallOptions()...)
			if err != nil {
				log.Errorf("Eino tool failed to generate response: %v", err)
				sendLLMError(responseChan, err)
				return
			}

			if message != nil {
				responseChan <- message
			}
		}

		log.Infof("[Eino-LLM] Eino tool request processing completed - SessionID: %s", sessionID)
	}()

	return responseChan
}

func (p *EinoLLMProvider) buildModelCallOptions() []model.Option {
	if p == nil || p.maxTokens <= 0 {
		return nil
	}

	provider := ""
	if p.config != nil {
		if rawProvider, ok := p.config["provider"].(string); ok {
			provider = rawProvider
		}
	}

	if shouldUseMaxCompletionTokens(provider, p.modelName) {
		return nil
	}

	return []model.Option{model.WithMaxTokens(p.maxTokens)}
}

// isValidJSON checks if a string is valid JSON
func isValidJSON(str string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(str), &js) == nil
}

// GetChatModel Gets the underlying Eino ChatModel
func (p *EinoLLMProvider) GetChatModel() model.ToolCallingChatModel {
	return p.chatModel
}

// GetProviderType Gets the provider type
func (p *EinoLLMProvider) GetProviderType() string {
	return p.providerType
}

// WithMaxTokens sets the maximum number of tokens
func (p *EinoLLMProvider) WithMaxTokens(maxTokens int) *EinoLLMProvider {
	newProvider := *p
	newProvider.maxTokens = maxTokens
	return &newProvider
}

// WithStreamable sets whether to support streaming
func (p *EinoLLMProvider) WithStreamable(streamable bool) *EinoLLMProvider {
	newProvider := *p
	newProvider.streamable = streamable
	return &newProvider
}

// Close closes the resource (stateless provider, no need to close)
func (p *EinoLLMProvider) Close() error {
	return nil
}

// IsValid checks whether the resource is valid
func (p *EinoLLMProvider) IsValid() bool {
	return p != nil && p.chatModel != nil
}
