package tts

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"xiaozhi-esp32-server-golang/constants"
	"xiaozhi-esp32-server-golang/internal/domain/tts/cosyvoice"
	"xiaozhi-esp32-server-golang/internal/domain/tts/supertonic"
	"xiaozhi-esp32-server-golang/internal/domain/tts/doubao"
	"xiaozhi-esp32-server-golang/internal/domain/tts/edge"
	"xiaozhi-esp32-server-golang/internal/domain/tts/edge_offline"
	"xiaozhi-esp32-server-golang/internal/domain/tts/minimax"
	"xiaozhi-esp32-server-golang/internal/domain/tts/openai"
	"xiaozhi-esp32-server-golang/internal/domain/tts/qwen"
	"xiaozhi-esp32-server-golang/internal/domain/tts/streaming"
	"xiaozhi-esp32-server-golang/internal/domain/tts/xiaozhi"
	"xiaozhi-esp32-server-golang/internal/domain/tts/xunfei"
	"xiaozhi-esp32-server-golang/internal/domain/tts/xunfei_super_tts"
	"xiaozhi-esp32-server-golang/internal/domain/tts/zhipu"
)

// Basic TTS provider interface (excluding Context method)
type BaseTTSProvider interface {
	TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error)
	TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error)
}

// DualStreamProvider TTS input and output are streaming optional interfaces: text is received and output is synthesized at the same time. Provider implements this interface if supported.
type DualStreamProvider interface {
	StreamingSynthesize(ctx context.Context, textChan <-chan string, sampleRate int, channels int, frameDuration int) (outputChan chan streaming.SynthesisEvent, err error)
}

// Complete TTS provider interface (including Context method)
type TTSProvider interface {
	BaseTTSProvider
	//SetVoice dynamically sets voice parameters
	//voiceConfig: map containing voice-related configurations, such as {"voice": "xxx"} or {"spk_id": "xxx"}
	SetVoice(voiceConfig map[string]interface{}) error
	//Close closes resources, releases connections, etc.
	Close() error
	//IsValid checks whether the resource is valid (whether the connection is alive, etc.)
	IsValid() bool
}

// GetTTSProvider Gets a complete TTS provider (supports Context)
// providerName: may be config_id/provider or resource pool key (such as "edge_tts:zh-CN-XiaoxiaoNeural")
// config: Configuration map parsed from the json_data field of the database configs table
// Priority is given to using the provider field in config, otherwise it is resolved from providerName (take the part before ":")
func GetTTSProvider(providerName string, config map[string]interface{}) (TTSProvider, error) {
	effectiveName := providerName
	if configProvider, ok := config["provider"].(string); ok && configProvider != "" {
		effectiveName = configProvider
	}
	//The resource pool key format is "provider:voiceID", take the first half as the provider type
	if idx := strings.Index(effectiveName, ":"); idx > 0 {
		effectiveName = effectiveName[:idx]
	}
	var baseProvider BaseTTSProvider

	switch effectiveName {
	case constants.TtsTypeDoubao:
		baseProvider = doubao.NewDoubaoTTSProvider(config)
	case constants.TtsTypeDoubaoWS:
		baseProvider = doubao.NewDoubaoWSProvider(config)
	case constants.TtsTypeCosyvoice:
		baseProvider = cosyvoice.NewCosyVoiceTTSProvider(config)
	case constants.TtsTypeEdge:
		baseProvider = edge.NewEdgeTTSProvider(config)
	case constants.TtsTypeEdgeOffline:
		baseProvider = edge_offline.NewEdgeOfflineTTSProvider(config)
	case constants.TtsTypeXiaozhi:
		baseProvider = xiaozhi.NewXiaozhiProvider(config)
	case constants.TtsTypeXunfei:
		baseProvider = xunfei.NewXunfeiTTSProvider(config)
	case constants.TtsTypeXunfeiSuper:
		baseProvider = xunfei_super_tts.NewXunfeiSuperTTSProvider(config)
	case constants.TtsTypeOpenAI:
		baseProvider = openai.NewOpenAITTSProvider(config)
	case constants.TtsTypeZhipu:
		baseProvider = zhipu.NewZhipuTTSProvider(config)
	case constants.TtsTypeMinimax:
		baseProvider = minimax.NewMinimaxTTSProvider(config)
	case constants.TtsTypeAliyunQwen:
		baseProvider = qwen.NewQwenTTSProvider(config)
	case constants.TtsTypeIndexTTSVLLM:
		baseProvider = openai.NewOpenAITTSProvider(buildIndexTTSOpenAIConfig(config))
	case constants.TtsTypeSupertonic:
		baseProvider = supertonic.NewSupertonicTTSProvider(config)
	default:
		return nil, fmt.Errorf("Unsupported TTS provider: %s", effectiveName)
	}

	if baseProvider == nil {
		return nil, fmt.Errorf("Unable to create TTS provider: %s", effectiveName)
	}

	//Use an adapter to wrap the base provider and convert it to a complete TTSProvider
	provider := &ContextTTSAdapter{baseProvider}

	return provider, nil
}

func buildIndexTTSOpenAIConfig(config map[string]interface{}) map[string]interface{} {
	const (
		defaultIndexTTSURL   = "http://127.0.0.1:7860/audio/speech"
		defaultIndexTTSModel = "indextts-vllm"
	)

	normalized := make(map[string]interface{}, len(config)+4)
	for k, v := range config {
		normalized[k] = v
	}

	apiURL, _ := normalized["api_url"].(string)
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		apiURL = defaultIndexTTSURL
	} else {
		parsed, err := url.Parse(apiURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			trimmed := strings.TrimRight(apiURL, "/")
			if !strings.HasSuffix(strings.ToLower(trimmed), "/audio/speech") {
				trimmed += "/audio/speech"
			}
			apiURL = trimmed
		} else {
			if strings.TrimSpace(parsed.Path) == "" || parsed.Path == "/" {
				parsed.Path = "/audio/speech"
				parsed.RawPath = ""
				apiURL = parsed.String()
			}
		}
	}
	normalized["api_url"] = strings.TrimRight(apiURL, "/")

	if model, _ := normalized["model"].(string); strings.TrimSpace(model) == "" {
		normalized["model"] = defaultIndexTTSModel
	}
	if responseFormat, _ := normalized["response_format"].(string); strings.TrimSpace(responseFormat) == "" {
		normalized["response_format"] = "wav"
	}
	if _, exists := normalized["stream"]; !exists {
		normalized["stream"] = false
	}
	if _, exists := normalized["speed"]; !exists {
		normalized["speed"] = float64(1.0)
	}

	return normalized
}

// ContextTTSAdapter is an adapter that adds Context support to the base TTS provider
type ContextTTSAdapter struct {
	Provider BaseTTSProvider
}

// StreamingSynthesize Dual-streaming synthesis interface that proxies to the original provider
func (a *ContextTTSAdapter) StreamingSynthesize(ctx context.Context, textChan <-chan string, sampleRate int, channels int, frameDuration int) (outputChan chan streaming.SynthesisEvent, err error) {
	//Check whether the underlying Provider supports dual streams
	if dsProvider, ok := a.Provider.(DualStreamProvider); ok {
		return dsProvider.StreamingSynthesize(ctx, textChan, sampleRate, channels, frameDuration)
	}
	return nil, fmt.Errorf("The underlying Provider does not support dual-stream composition")
}

// TextToSpeech proxy to original provider
func (a *ContextTTSAdapter) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	return a.Provider.TextToSpeech(ctx, text, sampleRate, channels, frameDuration)
}

// TextToSpeechStream proxies to the original provider
func (a *ContextTTSAdapter) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	return a.Provider.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
}

// SetVoice proxies to the underlying Provider's SetVoice method
func (a *ContextTTSAdapter) SetVoice(voiceConfig map[string]interface{}) error {
	//If the underlying Provider implements the SetVoice method, call it directly
	if setter, ok := a.Provider.(interface {
		SetVoice(map[string]interface{}) error
	}); ok {
		return setter.SetVoice(voiceConfig)
	}
	//Otherwise an unsupported error is returned
	return fmt.Errorf("The underlying Provider does not support the SetVoice method")
}

// TextToSpeechWithContext uses the Context version of text-to-speech
func (a *ContextTTSAdapter) TextToSpeechWithContext(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	//Check if the provider directly supports the Context version
	if provider, ok := a.Provider.(interface {
		TextToSpeechWithContext(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error)
	}); ok {
		//Providers directly support Context versions
		return provider.TextToSpeechWithContext(ctx, text, sampleRate, channels, frameDuration)
	}

	//Otherwise use the standard version and implement context control through goroutines and channels
	resultChan := make(chan struct {
		frames [][]byte
		err    error
	})

	go func() {
		frames, err := a.Provider.TextToSpeech(ctx, text, sampleRate, channels, frameDuration)
		select {
		case <-ctx.Done():
			//Context canceled, no results sent
			return
		case resultChan <- struct {
			frames [][]byte
			err    error
		}{frames, err}:
			//Result sent
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultChan:
		return result.frames, result.err
	}
}

// TextToSpeechStreamWithContext uses the Context version of streaming text-to-speech
func (a *ContextTTSAdapter) TextToSpeechStreamWithContext(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, cancelFunc func(), err error) {
	//Check if the provider directly supports the Context version
	if provider, ok := a.Provider.(interface {
		TextToSpeechStreamWithContext(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, func(), error)
	}); ok {
		//Providers directly support Context versions
		return provider.TextToSpeechStreamWithContext(ctx, text, sampleRate, channels, frameDuration)
	}

	//Otherwise use the standard version, but create a wrapper to handle context cancellation
	streamCtx, cancel := context.WithCancel(ctx)
	streamChan, err := a.Provider.TextToSpeechStream(streamCtx, text, sampleRate, channels, frameDuration)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	cancelFunc = cancel

	//Create a new output channel for forwarding and handling cancellations
	outputChan = make(chan []byte, 10)

	//Create a goroutine to forward data and listen for context cancellation
	go func() {
		defer close(outputChan)

		for {
			select {
			case <-streamCtx.Done():
				//The context has been canceled, call the original cancellation function and exit
				cancelFunc()
				return
			case frame, ok := <-streamChan:
				if !ok {
					//The original channel is closed
					return
				}
				//forward data
				select {
				case <-streamCtx.Done():
					//context canceled
					cancelFunc()
					return
				case outputChan <- frame:
					//Data forwarded successfully
				}
			}
		}
	}()

	return outputChan, cancelFunc, nil
}

// Close closes the resource
func (a *ContextTTSAdapter) Close() error {
	//If the underlying Provider implements the Close method, call it directly
	if closer, ok := a.Provider.(interface {
		Close() error
	}); ok {
		return closer.Close()
	}
	return nil
}

// IsValid checks whether the resource is valid
func (a *ContextTTSAdapter) IsValid() bool {
	//If the underlying Provider implements the IsValid method, call it directly
	if validator, ok := a.Provider.(interface {
		IsValid() bool
	}); ok {
		return validator.IsValid()
	}
	//Otherwise check if Provider is nil
	return a.Provider != nil
}
