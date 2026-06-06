package openai

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

	"github.com/gopxl/beep"

	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

// Global HTTP client, implementing connection pooling
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

// Get an HTTP client configured with a connection pool
func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		httpClient = &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second, //OpenAI TTS may take longer
		}
	})
	return httpClient
}

// OpenAITTSProvider OpenAI TTS provider
type OpenAITTSProvider struct {
	APIKey         string
	APIURL         string
	Model          string
	Voice          string
	ResponseFormat string
	Speed          float64
	Stream         bool
	FrameDuration  int
}

// Request structure
type openAIRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
	Stream         bool    `json:"stream,omitempty"`
}

// NewOpenAITTSProvider creates a new OpenAI TTS provider
func NewOpenAITTSProvider(config map[string]interface{}) *OpenAITTSProvider {
	apiKey, _ := config["api_key"].(string)
	apiURL, _ := config["api_url"].(string)
	model, _ := config["model"].(string)
	voice, _ := config["voice"].(string)
	responseFormat, _ := config["response_format"].(string)
	speed, _ := config["speed"].(float64)
	stream, _ := config["stream"].(bool)
	frameDuration, _ := config["frame_duration"].(float64)

	//Set default value
	if apiURL == "" {
		apiURL = "https://api.openai.com/v1/audio/speech"
	}
	if model == "" {
		model = "tts-1" //tts-1 or tts-1-hd
	}
	if voice == "" {
		voice = "alloy" // alloy, echo, fable, onyx, nova, shimmer
	}
	if responseFormat == "" {
		responseFormat = "mp3" // mp3, opus, aac, flac, wav, pcm
	}
	if speed == 0 {
		speed = 1.0 //0.25 to 4.0
	}
	if frameDuration == 0 {
		frameDuration = audio.FrameDuration
	}

	return &OpenAITTSProvider{
		APIKey:         apiKey,
		APIURL:         apiURL,
		Model:          model,
		Voice:          voice,
		ResponseFormat: responseFormat,
		Stream:         stream,
		Speed:          speed,
		FrameDuration:  int(frameDuration),
	}
}

// TextToSpeech converts text to speech, returning audio frame data and errors
func (p *OpenAITTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	streamChan, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}

	audioFrames := make([][]byte, 0, 32)
	for frame := range streamChan {
		audioFrames = append(audioFrames, frame)
	}
	if len(audioFrames) == 0 {
		return nil, fmt.Errorf("OpenAI TTS returns empty audio")
	}
	return audioFrames, nil
}

// TextToSpeechStream streaming speech synthesis implementation
func (p *OpenAITTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	startTs := time.Now().UnixMilli()

	//Create request body
	reqBody := openAIRequest{
		Model:          p.Model,
		Input:          text,
		Voice:          p.Voice,
		ResponseFormat: p.ResponseFormat,
		Speed:          p.Speed,
		Stream:         p.Stream,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Serialization request failed: %v", err)
	}

	//log.Debugf("OpOpenAI TTS request: %s: %s", string(jsonData))

	//Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Create request failed: %v", err)
	}

	//Set request header
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))
	}

	//Create a client using a connection pool
	client := getHTTPClient()

	//Create output channel
	outputChan = make(chan []byte, 100)

	//Start goroutine to process streaming response
	go func() {
		//Send request
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Failed to send OpenAI request: %v", err)
			close(outputChan)
			return
		}
		defer resp.Body.Close()

		//Check response status code
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("OpenAI API request failed, status code: %d, response: %s", resp.StatusCode, string(body))
			close(outputChan)
			return
		}

		//Check response content length
		contentLength := resp.ContentLength
		log.Debugf("Received OpenAI TTS response, Content-Length: %d", contentLength)

		//Determine whether Content-Length is reasonable
		if contentLength == 0 {
			log.Errorf("OpenAI API returns empty response with Content-Length of 0")
			close(outputChan)
			return
		}

		responseFormat := strings.ToLower(strings.TrimSpace(p.ResponseFormat))
		decoderFormat := responseFormat
		if responseFormat == "opus" {
			decoderFormat = "ogg_opus"
			contentTypeFormat := util.GetAudioFormatByMimeType(strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type"))))
			if contentTypeFormat == "ogg_opus" || contentTypeFormat == "opus" {
				decoderFormat = contentTypeFormat
			}
		}

		if decoderFormat != "mp3" && decoderFormat != "wav" && decoderFormat != "pcm" && decoderFormat != "opus" && decoderFormat != "ogg_opus" {
			log.Errorf("Currently only supports streaming synthesis in mp3/wav/pcm/opus/ogg_opus format")
			close(outputChan)
			return
		}

		decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, resp.Body, outputChan, frameDuration, decoderFormat, sampleRate)
		if err != nil {
			log.Errorf("Failed to create OpenAI audio decoder: %v", err)
			close(outputChan)
			return
		}
		if decoderFormat == "opus" {
			sourceChannels := channels
			if sourceChannels < 1 {
				sourceChannels = 1
			}
			decoder.WithFormat(beep.Format{
				SampleRate:  beep.SampleRate(util.NormalizeOpusSampleRate(sampleRate)),
				NumChannels: sourceChannels,
			})
		}

		if err := decoder.Run(startTs); err != nil {
			log.Errorf("OpenAI audio decoding failed: %v", err)
			return
		}

		select {
		case <-ctx.Done():
			log.Debugf("OpenAI TTS streaming synthesis canceled, text: %s", text)
			return
		default:
			log.Infof("OpenAI TTS time consuming: from input to audio data acquisition: %d ms", time.Now().UnixMilli()-startTs)
		}
	}()

	return outputChan, nil
}

// SetVoice sets voice parameters
func (p *OpenAITTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && voice != "" {
		p.Voice = voice
		return nil
	}
	return fmt.Errorf("Invalid voice configuration: missing voice")
}

// Close closes the resource (stateless provider, no need to close)
func (p *OpenAITTSProvider) Close() error {
	return nil
}

// IsValid checks whether the resource is valid
func (p *OpenAITTSProvider) IsValid() bool {
	return p != nil
}
