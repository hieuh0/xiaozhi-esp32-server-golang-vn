package zhipu

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
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

	"github.com/gopxl/beep"
	sse "github.com/tmaxmax/go-sse"
)

// Global HTTP client, implementing connection pooling
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

const (
	zhipuDefaultSampleRate = 24000
	zhipuLeadingFadeInMs   = 5
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
			Timeout:   60 * time.Second,
		}
	})
	return httpClient
}

// ZhipuTTSProvider Zhipu TTS provider
type ZhipuTTSProvider struct {
	APIKey         string
	APIURL         string
	Model          string
	Voice          string
	ResponseFormat string
	Speed          float64
	Volume         float64
	Stream         bool
	EncodeFormat   string //Only used when streaming: base64 or hex
	FrameDuration  int
}

// Request structure (according to Zhipu API documentation)
type zhipuRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
	Volume         float64 `json:"volume,omitempty"`
	Stream         bool    `json:"stream,omitempty"`
	EncodeFormat   string  `json:"encode_format,omitempty"` //Only used when streaming: base64 or hex
}

// Event Stream response structure (similar to OpenAI format)
type zhipuEventStreamResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason,omitempty"`
		Delta        struct {
			Role             string `json:"role,omitempty"`
			Content          string `json:"content,omitempty"` //base64 encoded audio data
			ReturnSampleRate int    `json:"return_sample_rate,omitempty"`
			ReturnFormat     string `json:"return_format,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
}

// NewZhipuTTSProvider Create a new Zhipu TTS provider
func NewZhipuTTSProvider(config map[string]interface{}) *ZhipuTTSProvider {
	apiKey, _ := config["api_key"].(string)
	apiURL, _ := config["api_url"].(string)
	model, _ := config["model"].(string)
	voice, _ := config["voice"].(string)
	responseFormat, _ := config["response_format"].(string)
	speed, _ := config["speed"].(float64)
	volume, _ := config["volume"].(float64)
	stream, _ := config["stream"].(bool)
	encodeFormat, _ := config["encode_format"].(string)
	frameDuration, _ := config["frame_duration"].(float64)

	//Set default value
	if apiURL == "" {
		apiURL = "https://open.bigmodel.cn/api/paas/v4/audio/speech"
	}
	if model == "" {
		model = "glm-tts"
	}
	if voice == "" {
		voice = "tongtong" //Default sound
	}
	if responseFormat == "" {
		responseFormat = "pcm" //Zhipu defaults to pcm and also supports wav
	}
	if speed == 0 {
		speed = 1.0 //0.5 to 2.0
	}
	if volume == 0 {
		volume = 1.0 //0 to 10
	}
	if encodeFormat == "" {
		encodeFormat = "base64" //Default base64, also supports hex
	}
	if frameDuration == 0 {
		frameDuration = audio.FrameDuration
	}

	return &ZhipuTTSProvider{
		APIKey:         apiKey,
		APIURL:         apiURL,
		Model:          model,
		Voice:          voice,
		ResponseFormat: responseFormat,
		Stream:         stream,
		Speed:          speed,
		Volume:         volume,
		EncodeFormat:   encodeFormat,
		FrameDuration:  int(frameDuration),
	}
}

// TextToSpeech converts text to speech, returning audio frame data and errors
func (p *ZhipuTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	startTs := time.Now().UnixMilli()

	//Limit text length (Zhipu API maximum 1024 characters)
	if len(text) > 1024 {
		text = text[:1024]
		log.Warnf("Text length exceeds 1024 characters and has been truncated")
	}

	//Create request body
	reqBody := zhipuRequest{
		Model:          p.Model,
		Input:          text,
		Voice:          p.Voice,
		ResponseFormat: p.ResponseFormat,
		Speed:          p.Speed,
		Volume:         p.Volume,
		Stream:         false, //non-streaming
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Serialization request failed: %v", err)
	}

	//Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Create request failed: %v", err)
	}

	//Set request header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))

	//Use connection pool to send requests
	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	//Check response status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	//Check response content length
	contentLength := resp.ContentLength
	log.Debugf("Received Zhipu TTS response, Content-Length: %d", contentLength)

	//Determine whether Content-Length is reasonable
	if contentLength == 0 {
		log.Errorf("API returns empty response with Content-Length of 0")
		return nil, fmt.Errorf("API returns empty response with Content-Length of 0")
	}

	//Process the response according to the audio format (Zhipu only supports wav and pcm)
	if p.ResponseFormat == "wav" || p.ResponseFormat == "pcm" {
		audioReader := io.ReadCloser(resp.Body)
		if strings.EqualFold(p.ResponseFormat, "pcm") {
			pcmData, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("Failed to read Zhipu PCM data: %v", err)
			}
			audioReader = io.NopCloser(bytes.NewReader(
				applyPCM16MonoLeadingFadeIn(pcmData, leadingFadeInSampleCount(zhipuDefaultSampleRate, zhipuLeadingFadeInMs)),
			))
		}

		//Create a channel to collect audio frames
		outputChan := make(chan []byte, 1000)

		//Create audio decoder
		decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, audioReader, outputChan, frameDuration, p.ResponseFormat, sampleRate)
		if err != nil {
			return nil, fmt.Errorf("Failed to create audio decoder: %v", err)
		}
		if strings.EqualFold(p.ResponseFormat, "pcm") {
			decoder.WithFormat(beep.Format{
				SampleRate:  beep.SampleRate(zhipuDefaultSampleRate),
				NumChannels: 1,
			})
		}

		//Start decoding process
		go func() {
			if err := decoder.Run(startTs); err != nil {
				log.Errorf("Audio decoding failed: %v", err)
			}
		}()

		//Collect all audio frames
		var audioFrames [][]byte
		for frame := range outputChan {
			audioFrames = append(audioFrames, frame)
		}

		log.Debugf("Zhipu TTS is completed. The time taken from input to acquisition of audio data is: %d ms", time.Now().UnixMilli()-startTs)
		return audioFrames, nil
	}

	return nil, fmt.Errorf("Unsupported audio formats: %s, Zhipu only supports wav and pcm", p.ResponseFormat)
}

// TextToSpeechStream streaming speech synthesis implementation
func (p *ZhipuTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	startTs := time.Now().UnixMilli()

	//Limit text length (Zhipu API maximum 1024 characters)
	if len(text) > 1024 {
		text = text[:1024]
		log.Warnf("Text length exceeds 1024 characters and has been truncated")
	}

	//Only pcm and wav formats are supported during streaming
	responseFormat := p.ResponseFormat

	//Create request body
	reqBody := zhipuRequest{
		Model:          p.Model,
		Input:          text,
		Voice:          p.Voice,
		ResponseFormat: responseFormat,
		Speed:          p.Speed,
		Volume:         p.Volume,
		Stream:         true,           //streaming
		EncodeFormat:   p.EncodeFormat, //Use configured encoding format
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Serialization request failed: %v", err)
	}

	//Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Create request failed: %v", err)
	}

	//Set request header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))

	//Create a client using a connection pool
	client := getHTTPClient()

	//Create output channel
	outputChan = make(chan []byte, 100)

	//Start goroutine to process streaming response
	go func() {
		//Send request
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Failed to send Zhipu request: %v", err)
			close(outputChan)
			return
		}
		defer resp.Body.Close()

		//Check response status code
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("Zhipu API request failed, status code: %d, response: %s", resp.StatusCode, string(body))
			close(outputChan)
			return
		}

		//Check if Content-Type is Event Stream
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/event-stream") {
			log.Warnf("The Content-Type returned by Zhipu API is not text/event-stream: %s", contentType)
		}

		//Only pcm and wav formats are supported when streaming
		//log.Debugf("ZhZhipu TTS streaming responseFormat(request): %s(request): %s", responseFormat)
		if responseFormat == "pcm" || responseFormat == "wav" {
			//Create a pipeline to pass decoded binary data to the audio decoder
			pipeReader, pipeWriter := io.Pipe()

			//Start goroutine to parse Event Stream and decode
			go func() {
				defer func() {
					if err := pipeWriter.Close(); err != nil {
						log.Debugf("Failed to close pipe write end: %v", err)
					}
				}()

				//Call independent parsing method
				if err := p.parseEventStream(ctx, resp.Body, pipeWriter, text); err != nil {
					log.Errorf("Failed to parse Event Stream: %v", err)
				}
			}()

			//Create an audio decoder to read decoded binary data from the pipe
			decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, responseFormat, sampleRate)
			if err != nil {
				log.Errorf("Failed to create Zhipu audio decoder: %v", err)
				pipeReader.Close()
				close(outputChan)
				return
			}
			if strings.EqualFold(responseFormat, "pcm") {
				decoder.WithFormat(beep.Format{
					SampleRate:  beep.SampleRate(zhipuDefaultSampleRate),
					NumChannels: 1,
				})
			}

			//Start decoding process
			if err := decoder.Run(startTs); err != nil {
				log.Errorf("Zhipu audio decoding failed: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				log.Debugf("Zhipu TTS streaming synthesis cancelled, text: %s", text)
				return
			default:
				log.Debugf("Zhipu TTS time consumption: Time consumption from input to acquisition of audio data: %d ms", time.Now().UnixMilli()-startTs)
			}
		} else {
			log.Errorf("Zhipu streaming output only supports pcm format")
			close(outputChan)
		}
	}()

	return outputChan, nil
}

// parseEventStream uses go-sse to parse Zhipu's Event Stream response, decode the data and write it to the pipeline
// ctx: context, used to cancel the operation
// reader: response body reader
// writer: Pipe writing end, used to output decoded binary data
// text: original text, used for logging
func (p *ZhipuTTSProvider) parseEventStream(ctx context.Context, reader io.Reader, writer *io.PipeWriter, text string) error {
	//Configure go-sse's ReadConfig and set a larger MaxEventSize to handle long tokens
	//The base64-encoded audio data returned by Zhipu TTS may exceed the default 64KB limit
	readConfig := &sse.ReadConfig{
		MaxEventSize: 4 * 1024 * 1024, //4MB, enough to handle large base64 encoded audio data
	}
	fadeTotalSamples := 0
	fadeSamplesRemaining := -1

	for ev, evErr := range sse.Read(reader, readConfig) {
		if evErr != nil {
			return fmt.Errorf("Failed to read Zhipu SSE events: %w", evErr)
		}

		select {
		case <-ctx.Done():
			log.Debugf("Zhipu TTS streaming synthesis cancelled, text: %s", text)
			return ctx.Err()
		default:
		}

		//Event stream format:
		// data: {"id":"...","choices":[{"delta":{"content":"base64_data"}}]}
		// data: {"choices":[{"finish_reason":"stop"}]}

		dataValue := strings.TrimSpace(ev.Data)
		if dataValue == "" {
			continue
		}

		//Parse JSON
		var eventResp zhipuEventStreamResponse
		if err := json.Unmarshal([]byte(dataValue), &eventResp); err != nil {
			log.Warnf("Failed to parse Zhipu Event Stream JSON: %v, data: %s", err, previewString(dataValue, 200))
			continue
		}

		//Check if there is finish_reason, indicating the end of the stream
		for _, choice := range eventResp.Choices {
			if choice.FinishReason == "stop" {
				log.Debugf("Received finish_reason: stop, Event Stream ended")
				return nil
			}
		}

		//Extract the content field of each choice and process it independently
		for _, choice := range eventResp.Choices {
			if choice.Delta.Content != "" {
				decodedData, err := p.decodeAudioContent(choice.Delta.Content)
				if err != nil {
					return fmt.Errorf("Failed to process content: %v", err)
				}

				returnFormat := strings.TrimSpace(choice.Delta.ReturnFormat)
				if returnFormat == "" {
					returnFormat = p.ResponseFormat
				}
				if strings.EqualFold(returnFormat, "pcm") {
					if fadeSamplesRemaining < 0 {
						sampleRate := choice.Delta.ReturnSampleRate
						if sampleRate < 1 {
							sampleRate = zhipuDefaultSampleRate
						}
						fadeTotalSamples = leadingFadeInSampleCount(sampleRate, zhipuLeadingFadeInMs)
						fadeSamplesRemaining = fadeTotalSamples
					}
					applyPCM16MonoLeadingFadeInInPlace(decodedData, fadeTotalSamples, &fadeSamplesRemaining)
				}

				if len(decodedData) > 0 {
					if _, err := writer.Write(decodedData); err != nil {
						return fmt.Errorf("Failed to write to pipe: %v", err)
					}
				}
			}
		}
	}

	return nil
}

// previewString returns the first n characters of the string for logging
func previewString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// decodeAudioContent decodes a single content field
// content: base64 or hex encoded audio data string
func (p *ZhipuTTSProvider) decodeAudioContent(content string) ([]byte, error) {
	if content == "" {
		return nil, nil
	}

	//Decode according to encode_format
	var decodedData []byte
	var decodeErr error

	switch p.EncodeFormat {
	case "base64":
		decodedData, decodeErr = base64.StdEncoding.DecodeString(content)
	case "hex":
		decodedData, decodeErr = hex.DecodeString(content)
	default:
		log.Warnf("Unknown encoding format: %s, using base64", p.EncodeFormat)
		decodedData, decodeErr = base64.StdEncoding.DecodeString(content)
	}

	if decodeErr != nil {
		return nil, fmt.Errorf("Failed to decode audio data: %v, data length: %d", decodeErr, len(content))
	}

	return decodedData, nil
}

func leadingFadeInSampleCount(sampleRate int, fadeMs int) int {
	if sampleRate < 1 {
		sampleRate = zhipuDefaultSampleRate
	}
	if fadeMs < 1 {
		return 0
	}
	samples := sampleRate * fadeMs / 1000
	if samples < 1 {
		return 1
	}
	return samples
}

func applyPCM16MonoLeadingFadeIn(data []byte, remainingSamples int) []byte {
	if len(data) == 0 || remainingSamples <= 0 {
		return data
	}
	cloned := make([]byte, len(data))
	copy(cloned, data)
	applyPCM16MonoLeadingFadeInInPlace(cloned, remainingSamples, &remainingSamples)
	return cloned
}

func applyPCM16MonoLeadingFadeInInPlace(data []byte, totalSamples int, remainingSamples *int) {
	if len(data) < 2 || totalSamples <= 0 || remainingSamples == nil || *remainingSamples <= 0 {
		return
	}

	samplePairs := len(data) / 2
	for i := 0; i < samplePairs && *remainingSamples > 0; i++ {
		offset := i * 2
		sample := int16(uint16(data[offset]) | uint16(data[offset+1])<<8)
		appliedIndex := totalSamples - *remainingSamples
		scaled := int32(sample) * int32(appliedIndex) / int32(totalSamples)
		binarySample := uint16(int16(scaled))
		data[offset] = byte(binarySample)
		data[offset+1] = byte(binarySample >> 8)
		*remainingSamples = *remainingSamples - 1
	}
}

// SetVoice sets voice parameters
func (p *ZhipuTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && voice != "" {
		p.Voice = voice
		return nil
	}
	return fmt.Errorf("Invalid voice configuration: missing voice")
}

// Close closes the resource (stateless provider, no need to close)
func (p *ZhipuTTSProvider) Close() error {
	return nil
}

// IsValid checks whether the resource is valid
func (p *ZhipuTTSProvider) IsValid() bool {
	return p != nil
}
