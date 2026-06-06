package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"

	"xiaozhi-esp32-server-golang/constants"
	"xiaozhi-esp32-server-golang/internal/domain/llm/coze_llm"
	"xiaozhi-esp32-server-golang/internal/domain/llm/dify_llm"
	"xiaozhi-esp32-server-golang/internal/domain/llm/eino_llm"
)

// LLMExtraErrorKey error transparent transmission convention: the key used in Message.Extra when ResponseWithContext fails
const LLMExtraErrorKey = "error"

// IsLLMErrorMessage determines whether it is an error message transmitted through LLM (Extra includes error)
func IsLLMErrorMessage(msg *schema.Message) bool {
	if msg == nil || msg.Extra == nil {
		return false
	}
	v, ok := msg.Extra[LLMExtraErrorKey]
	if !ok || v == nil {
		return false
	}
	_, ok = v.(string)
	return ok
}

// LLMErrorMessage parses the error text (if it is an error message) from Message.Extra
func LLMErrorMessage(msg *schema.Message) string {
	if msg == nil || msg.Extra == nil {
		return ""
	}
	v, ok := msg.Extra[LLMExtraErrorKey].(string)
	if !ok {
		return ""
	}
	return v
}

// LLMProvider large language model provider interface
// All LLM implementations must follow this interface and use Eino native types
type LLMProvider interface {
	//ResponseWithContext is a response with context control and supports cancellation operations.
	//ctx: context, which can be used to cancel long-running requests
	//sessionID: session identifier
	//dialogue: dialogue history, using Eino native message types
	ResponseWithContext(ctx context.Context, sessionID string, dialogue []*schema.Message, functions []*schema.ToolInfo) chan *schema.Message

	ResponseWithVllm(ctx context.Context, file []byte, text string, mimeType string) (string, error)

	//GetModelInfo gets model information
	//Returns the model name and other metadata
	GetModelInfo() map[string]interface{}
	//Close closes resources, releases connections, etc.
	Close() error
	//IsValid checks whether the resource is valid
	IsValid() bool
}

// LLMFactory large language model factory interface
// For creating different types of LLM providers
type LLMFactory interface {
	//CreateProvider creates an LLM provider based on configuration
	CreateProvider(config map[string]interface{}) (LLMProvider, error)
}

// GetLLMProvider creates an LLM provider
// Unified use of EinoLLMProvider to handle all types
func GetLLMProvider(providerName string, config map[string]interface{}) (LLMProvider, error) {
	cfg := cloneConfigMap(config)
	if providerName != "" {
		if _, ok := cfg["provider"]; !ok {
			cfg["provider"] = providerName
		}
	}

	llmType := resolveLLMType(providerName, cfg)
	cfg["type"] = llmType
	providerKey := resolveLLMProviderName(providerName, cfg, llmType)
	if defaultBaseURL := resolveDefaultBaseURL(providerKey); defaultBaseURL != "" {
		cfg["base_url"] = defaultBaseURL
	} else if baseURL, _ := cfg["base_url"].(string); strings.TrimSpace(baseURL) == "" {
		delete(cfg, "base_url")
	}

	switch llmType {
	case constants.LlmTypeOpenai, constants.LlmTypeOllama, constants.LlmTypeEinoLLM, constants.LlmTypeEino:
		//Use EinoLLMProvider uniformly to handle all types
		provider, err := eino_llm.NewEinoLLMProvider(cfg)
		if err != nil {
			return nil, fmt.Errorf("Failed to create Eino LLM provider: %v", err)
		}
		return provider, nil
	case constants.LlmTypeDify:
		provider, err := dify_llm.NewDifyLLMProvider(cfg)
		if err != nil {
			return nil, fmt.Errorf("Failed to create Dify LLM provider: %v", err)
		}
		return provider, nil
	case constants.LlmTypeCoze:
		provider, err := coze_llm.NewCozeLLMProvider(cfg)
		if err != nil {
			return nil, fmt.Errorf("Failed to create Coze LLM provider: %v", err)
		}
		return provider, nil
	}
	return nil, fmt.Errorf("Unsupported LLM provider: %s", llmType)
}

func resolveLLMProviderName(providerName string, config map[string]interface{}, llmType string) string {
	provider := strings.ToLower(strings.TrimSpace(providerName))
	if provider == "" {
		if rawProvider, ok := config["provider"].(string); ok {
			provider = strings.ToLower(strings.TrimSpace(rawProvider))
		}
	}
	if provider == "openai" {
		switch llmType {
		case constants.LlmTypeOllama:
			return "ollama"
		case constants.LlmTypeDify:
			return "dify"
		case constants.LlmTypeCoze:
			return "coze"
		}
	}
	return provider
}

func resolveDefaultBaseURL(provider string) string {
	switch provider {
	case "anthropic":
		return "https://api.anthropic.com/v1/"
	case "zhipu":
		return "https://open.bigmodel.cn/api/paas/v4"
	case "aliyun":
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	case "doubao":
		return "https://ark.cn-beijing.volces.com/api/v3"
	case "siliconflow":
		return "https://api.siliconflow.cn/v1"
	case "deepseek":
		return "https://api.deepseek.com/v1"
	default:
		return ""
	}
}

func resolveLLMType(providerName string, config map[string]interface{}) string {
	provider := strings.ToLower(strings.TrimSpace(providerName))
	if provider == "" {
		if rawProvider, ok := config["provider"].(string); ok {
			provider = strings.ToLower(strings.TrimSpace(rawProvider))
		}
	}

	llmType, _ := config["type"].(string)
	llmType = strings.ToLower(strings.TrimSpace(llmType))

	if provider == "openai" {
		switch llmType {
		case constants.LlmTypeOllama:
			return constants.LlmTypeOllama
		case constants.LlmTypeDify:
			return constants.LlmTypeDify
		case constants.LlmTypeCoze:
			return constants.LlmTypeCoze
		}
	}

	switch provider {
	case "ollama":
		return constants.LlmTypeOllama
	case "dify":
		return constants.LlmTypeDify
	case "coze":
		return constants.LlmTypeCoze
	case "openai", "azure", "anthropic", "zhipu", "aliyun", "doubao", "siliconflow", "deepseek":
		return constants.LlmTypeOpenai
	}

	switch llmType {
	case constants.LlmTypeOllama:
		return constants.LlmTypeOllama
	case constants.LlmTypeDify:
		return constants.LlmTypeDify
	case constants.LlmTypeCoze:
		return constants.LlmTypeCoze
	case constants.LlmTypeOpenai, constants.LlmTypeEinoLLM, constants.LlmTypeEino:
		return constants.LlmTypeOpenai
	default:
		return constants.LlmTypeOpenai
	}
}

// Config LLM configuration structure
type Config struct {
	ModelName  string                 `json:"model_name"`
	APIKey     string                 `json:"api_key"`
	BaseURL    string                 `json:"base_url"`
	MaxTokens  int                    `json:"max_tokens"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

func cloneConfigMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return make(map[string]interface{})
	}

	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
