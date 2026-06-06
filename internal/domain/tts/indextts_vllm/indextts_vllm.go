package indextts_vllm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

const (
	defaultIndexTTSBaseURL     = "http://127.0.0.1:7860"
	indexttsTTSEndpoint        = "/audio/speech"
	defaultIndexTTSModel       = "indextts-vllm"
	defaultIndexResponseFormat = "wav"
)

var (
	indexHTTPClient     *http.Client
	indexHTTPClientOnce sync.Once
)

func getHTTPClient() *http.Client {
	indexHTTPClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		indexHTTPClient = &http.Client{Transport: transport, Timeout: 120 * time.Second}
	})
	return indexHTTPClient
}

type IndexTTSVLLMProvider struct {
	APIKey         string
	BaseURL        string
	Model          string
	Voice          string
	ResponseFormat string
	Stream         bool
	FrameDuration  int
}

type speechRequest struct {
	Model string `json:"model,omitempty"`
	Input string `json:"input"`
	Voice string `json:"voice"`
}

func NewIndexTTSVLLMProvider(config map[string]interface{}) *IndexTTSVLLMProvider {
	apiKey, _ := config["api_key"].(string)
	baseURL, _ := config["api_url"].(string)
	model, _ := config["model"].(string)
	voice, _ := config["voice"].(string)
	responseFormat, _ := config["response_format"].(string)
	stream, _ := config["stream"].(bool)
	frameDuration, _ := config["frame_duration"].(float64)

	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultIndexTTSBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if model == "" {
		model = defaultIndexTTSModel
	}
	if responseFormat == "" {
		responseFormat = defaultIndexResponseFormat
	}
	if frameDuration == 0 {
		frameDuration = audio.FrameDuration
	}

	return &IndexTTSVLLMProvider{
		APIKey:         strings.TrimSpace(apiKey),
		BaseURL:        baseURL,
		Model:          model,
		Voice:          voice,
		ResponseFormat: strings.ToLower(strings.TrimSpace(responseFormat)),
		Stream:         stream,
		FrameDuration:  int(frameDuration),
	}
}

func (p *IndexTTSVLLMProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	streamChan, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}
	frames := make([][]byte, 0, 32)
	for frame := range streamChan {
		frames = append(frames, frame)
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("IndexTTS returns empty audio")
	}
	return frames, nil
}

func (p *IndexTTSVLLMProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	if strings.TrimSpace(p.Voice) == "" {
		return nil, fmt.Errorf("indextts_vllm voice is not configured, please create a voice through /audio/clone first")
	}
	if frameDuration <= 0 {
		frameDuration = p.FrameDuration
	}
	if frameDuration <= 0 {
		frameDuration = audio.FrameDuration
	}

	payload := speechRequest{Model: p.Model, Input: text, Voice: p.Voice}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("Serialization request failed: %v", err)
	}

	url := p.BaseURL + indexttsTTSEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Create request failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/wav,application/octet-stream,*/*")
	if p.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))
	}

	outputChan = make(chan []byte, 100)
	go func() {
		defer close(outputChan)
		resp, reqErr := getHTTPClient().Do(req)
		if reqErr != nil {
			log.Errorf("IndexTTS request failed: %v", reqErr)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			log.Errorf("IndexTTS request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(msg)))
			return
		}

		decoder, decErr := util.CreateAudioDecoderWithSampleRate(ctx, resp.Body, outputChan, frameDuration, "wav", sampleRate)
		if decErr != nil {
			log.Errorf("Failed to create IndexTTS audio decoder: %v", decErr)
			return
		}
		if runErr := decoder.Run(time.Now().UnixMilli()); runErr != nil {
			log.Errorf("IndexTTS audio decoding failed: %v", runErr)
		}
	}()

	return outputChan, nil
}

func (p *IndexTTSVLLMProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && strings.TrimSpace(voice) != "" {
		p.Voice = strings.TrimSpace(voice)
		return nil
	}
	if character, ok := voiceConfig["character"].(string); ok && strings.TrimSpace(character) != "" {
		p.Voice = strings.TrimSpace(character)
		return nil
	}
	return fmt.Errorf("Invalid voice configuration: missing voice/character")
}

func (p *IndexTTSVLLMProvider) Close() error { return nil }

func (p *IndexTTSVLLMProvider) IsValid() bool { return p != nil }
