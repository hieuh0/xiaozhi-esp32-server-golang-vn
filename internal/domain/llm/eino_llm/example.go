package eino_llm

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"

	log "xiaozhi-esp32-server-golang/logger"
)

// ExampleConfig example configuration
var ExampleConfig = map[string]interface{}{
	"type":       "eino_llm",
	"model_name": "gpt-3.5-turbo",
	"api_key":    "your-api-key-here",
	"base_url":   "https://api.openai.com/v1",
	"max_tokens": 500,
	"streamable": true,
}

// ExampleUsage shows how to use EinoLLMProvider
func ExampleUsage() {
	//1. OpenAI configuration example
	openaiConfig := map[string]interface{}{
		"type":       "openai",
		"model_name": "gpt-3.5-turbo",
		"api_key":    "your-openai-api-key",
		"base_url":   "https://api.openai.com/v1",
		"max_tokens": 500,
		"streamable": true,
	}

	//2. Ollama configuration example
	ollamaConfig := map[string]interface{}{
		"type":       "ollama",
		"model_name": "llama2",
		"base_url":   "http://localhost:11434",
		"max_tokens": 500,
		"streamable": true,
	}

	//3. Create a provider
	openaiProvider, err := NewEinoLLMProvider(openaiConfig)
	if err != nil {
		log.Errorf("Failed to create OpenAI provider: %v", err)
		return
	}

	ollamaProvider, err := NewEinoLLMProvider(ollamaConfig)
	if err != nil {
		log.Errorf("Failed to create Ollama provider: %v", err)
		return
	}

	//4. Use Eino native message types
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "You are a helpful assistant",
		},
		{
			Role:    schema.User,
			Content: "Please introduce the Eino framework",
		},
	}

	//5. Basic conversation
	fmt.Println("=== OpenAI Basic Dialogue ===")
	responseChan := openaiProvider.ResponseWithContext(context.Background(), "example_session", messages, nil)
	for resp := range responseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()

	fmt.Println("=== Ollama Basic Dialogue ===")
	responseChan = ollamaProvider.ResponseWithContext(context.Background(), "example_session", messages, nil)
	for resp := range responseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()

	//6. Tool call example
	tools := []*schema.ToolInfo{
		{
			Name:        "get_weather",
			ParamsOneOf: &schema.ParamsOneOf{
				//Tool parameter definition
			},
		},
	}

	fmt.Println("=== Dialogue with tool call ===")
	toolResponseChan := openaiProvider.ResponseWithContext(context.Background(), "example_session", messages, tools)
	for resp := range toolResponseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()

	//7. Chain call example
	fmt.Println("=== Chain call example ===")
	enhancedProvider := openaiProvider.
		WithMaxTokens(1000).
		WithStreamable(false)

	fmt.Printf("Provider type: %s\n", enhancedProvider.GetProviderType())
	fmt.Printf("Model information: %+v\n", enhancedProvider.GetModelInfo())
}

// ExampleAdvancedUsage Advanced usage example
func ExampleAdvancedUsage() {
	config := map[string]interface{}{
		"type":       "openai",
		"model_name": "gpt-4",
		"api_key":    "your-api-key",
		"max_tokens": 1000,
		"streamable": true,
	}

	provider, err := NewEinoLLMProvider(config)
	if err != nil {
		log.Errorf("Failed to create provider: %v", err)
		return
	}

	//Use contextual controls
	ctx := context.Background()
	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: "Please write a long article about AI",
		},
	}

	fmt.Println("=== Dialogue with context control ===")
	responseChan := provider.ResponseWithContext(ctx, "advanced_session", messages, nil)
	for resp := range responseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()

	//Use Eino ChatModel directly
	chatModel := provider.GetChatModel()
	result, err := chatModel.Generate(ctx, messages)
	if err != nil {
		log.Errorf("Direct call to ChatModel failed: %v", err)
		return
	}

	fmt.Printf("Direct call result: %s\n", result.Content)
}

// ExampleMultiProvider Multiple provider example
func ExampleMultiProvider() {
	providers := make(map[string]*EinoLLMProvider)

	//Create multiple providers
	configs := map[string]map[string]interface{}{
		"openai": {
			"type":       "openai",
			"model_name": "gpt-3.5-turbo",
			"api_key":    "your-openai-key",
		},
		"ollama": {
			"type":       "ollama",
			"model_name": "llama2",
			"base_url":   "http://localhost:11434",
		},
	}

	for name, config := range configs {
		provider, err := NewEinoLLMProvider(config)
		if err != nil {
			log.Errorf("Failed to create %s provider: %v", name, err)
			continue
		}
		providers[name] = provider
	}

	//Use different providers to handle the same request
	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: "Hello, please introduce yourself",
		},
	}

	for name, provider := range providers {
		fmt.Printf("=== %s Provider response ===\n", name)
		responseChan := provider.ResponseWithContext(context.Background(), "multi_session", messages, nil)
		for resp := range responseChan {
			if resp.Content != "" {
				fmt.Print(resp.Content)
			}
			if len(resp.ToolCalls) > 0 {
				fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
			}
		}
		fmt.Println()
	}
}

// ExampleWithTools tool call example
func ExampleWithTools() {
	provider, err := NewEinoLLMProvider(ExampleConfig)
	if err != nil {
		log.Errorf("Failed to create provider: %v", err)
		return
	}

	//Use Eino native message types
	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: "What's the weather like in Beijing today? Please help me check.",
		},
	}

	//Use Eino native tool types
	tools := []*schema.ToolInfo{
		{
			Name:        "get_weather",
			ParamsOneOf: &schema.ParamsOneOf{
				//Simplified tool parameter definition
				//In actual use, the parameter structure needs to be correctly defined here
			},
		},
	}

	fmt.Println("=== Tool call example ===")

	//Use Eino native tools to call interfaces
	fmt.Println("--- Eino native tool call ---")
	responseChan := provider.ResponseWithContext(context.Background(), "tool_session", messages, tools)
	for resp := range responseChan {
		fmt.Printf("Response: %+v\n", resp)
	}
}

// MultiProviderExample Multi-provider example
func MultiProviderExample() {
	//OpenAI provider example
	fmt.Println("=== OpenAI provider example ===")
	openaiConfig := map[string]interface{}{
		"type":       "openai",
		"model_name": "gpt-3.5-turbo",
		"api_key":    "your-openai-api-key",
		"base_url":   "https://api.openai.com/v1",
		"max_tokens": 500,
	}

	openaiProvider, err := NewEinoLLMProvider(openaiConfig)
	if err != nil {
		log.Errorf("Failed to create OpenAI provider: %v", err)
		return
	}

	fmt.Printf("Provider type: %s\n", openaiProvider.GetProviderType())

	//Ollama provider example
	fmt.Println("\n=== Ollama Provider Example ===")
	ollamaConfig := map[string]interface{}{
		"type":       "ollama",
		"model_name": "llama2",
		"base_url":   "http://localhost:11434",
		"max_tokens": 500,
	}

	ollamaProvider, err := NewEinoLLMProvider(ollamaConfig)
	if err != nil {
		log.Errorf("Failed to create Ollama provider: %v", err)
		return
	}

	fmt.Printf("Provider type: %s\n", ollamaProvider.GetProviderType())

	//Use Eino native message types
	messages := []*schema.Message{
		{
			Role:    schema.User,
			Content: "Please introduce yourself.",
		},
	}

	//Test both providers separately
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("\n--- OpenAI response ---")
	openaiResponse := openaiProvider.ResponseWithContext(ctx, "openai_session", messages, nil)
	for resp := range openaiResponse {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}

	fmt.Println("\n--- Ollama response ---")
	ollamaResponse := ollamaProvider.ResponseWithContext(ctx, "ollama_session", messages, nil)
	for resp := range ollamaResponse {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()
}

// EinoFrameworkAdvantages Advantages of Eino framework
func EinoFrameworkAdvantages() string {
	return `
Main advantages of the Eino framework:

1. **Component-based design**
   - Rich component abstractions (ChatModel, Tool, ChatTemplate, Retriever, etc.)
   - Each component has a unified input/output interface
   - Supports component nesting and complex business logic encapsulation

2. **Powerful orchestration**
   - Graph-based data flow orchestration
   - Automatic type checking, stream processing, concurrency management
   - Supports branch execution, state management, field mapping

3. **Complete stream processing**
   - Automatic stream chunk chaining
   - Auto-boxing non-stream data into streams
   - Automatic stream merging
   - Automatic stream copying to multiple downstream nodes

4. **High extensibility**
   - Custom callback handler support
   - Five aspect hooks (OnStart, OnEnd, OnError, etc.)
   - Injectable logging, tracing, monitoring cross-cutting concerns

5. **Production ready**
   - Complete error handling mechanism
   - Timeout and cancellation support
   - Connection pooling and performance optimization
   - Detailed logging and monitoring

Implementation highlights:

**Multi-provider support**:
- Unified Eino interface supports OpenAI and Ollama
- Flexible provider switching via type config
- Each provider uses the same Eino ChatModel interface

**Eino native implementation**:
- Uses *schema.Message types directly for conversation
- Uses *schema.ToolInfo types directly for tool calls
- Fully built on Eino framework, no type conversion needed

**Enhancements**:
- Chain call support (WithMaxTokens, WithStreamable)
- Unified error handling and logging
- Streaming and non-streaming call mode support
- Fully compatible with the original LLMProvider interface

**Best Practices**:
- Context cancellation and timeout control
- Structured logging and monitoring integration
- Type-safe configuration management
- Automatic resource management and cleanup

This implementation truly leverages the core capabilities of the Eino framework while supporting multiple LLM providers.
`
}

// BasicUsageExample basic usage example
func BasicUsageExample() {
	provider, err := NewEinoLLMProvider(ExampleConfig)
	if err != nil {
		log.Errorf("Failed to create provider: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	//Demo chain configuration
	enhancedProvider := provider.
		WithMaxTokens(2000).
		WithStreamable(true)

	//Get the underlying Eino ChatModel
	chatModel := enhancedProvider.GetChatModel()
	fmt.Printf("Underlying ChatModel: %+v\n", chatModel)

	//Get provider type
	providerType := enhancedProvider.GetProviderType()
	fmt.Printf("Provider type: %s\n", providerType)

	//Get enhanced model information
	modelInfo := enhancedProvider.GetModelInfo()
	fmt.Printf("Enhanced model information: %+v\n", modelInfo)

	//Complex dialogue example - using Eino native message types
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "You are a professional software architect, proficient in Go language and AI application development.",
		},
		{
			Role:    schema.User,
			Content: "Please design a chatbot system architecture based on the Eino framework.",
		},
	}

	//Call using enhanced configuration
	responseChan := enhancedProvider.ResponseWithContext(ctx, "basic_example", messages, nil)
	fmt.Printf("Architectural design response:\n")
	for resp := range responseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()
}

// EinoNativeExample Eino native API example
func EinoNativeExample() {
	provider, err := NewEinoLLMProvider(ExampleConfig)
	if err != nil {
		log.Errorf("Failed to create provider: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//Use Eino native message types
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "You are a helpful AI assistant.",
		},
		{
			Role:    schema.User,
			Content: "Please briefly introduce the Eino framework.",
		},
	}

	fmt.Println("=== Eino native API example ===")

	//1. Using EinoResponse
	fmt.Println("--- EinoResponse ---")
	responseChan := provider.ResponseWithContext(ctx, "eino_session", messages, nil)
	for resp := range responseChan {
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
	fmt.Println()

	//2. Use EinoResponseWithTools
	fmt.Println("\n--- EinoResponseWithTools ---")
	tools := []*schema.ToolInfo{
		{
			Name:        "search_docs",
			ParamsOneOf: &schema.ParamsOneOf{
				//Tool parameter definition
			},
		},
	}

	toolResponseChan := provider.ResponseWithContext(ctx, "eino_tools_session", messages, tools)
	for resp := range toolResponseChan {
		if resp.Content != "" {
			fmt.Printf("Content: %s\n", resp.Content)
		}
		if len(resp.ToolCalls) > 0 {
			fmt.Printf("Tool call: %+v\n", resp.ToolCalls)
		}
	}
}

func main() {
	BasicUsageExample()
}
