package qwen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
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

const (
	defaultAPIURLBeijing    = "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	defaultAPIURLSingapore  = "https://dashscope-intl.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	defaultQwenModel        = "qwen3-tts-flash"
	defaultQwenVoice        = "Cherry"
	defaultQwenLanguageType = "Chinese"
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
			Timeout:   60 * time.Second,
		}
	})
	return httpClient
}

// QwenTTSProvider Alibaba Cloud Qianwen TTS provider
type QwenTTSProvider struct {
	APIKey        string
	APIURL        string
	Model         string
	Voice         string
	LanguageType  string
	Stream        bool
	FrameDuration int
}

// qwenRequest request structure
type qwenRequest struct {
	Model string           `json:"model"`
	Input qwenRequestInput `json:"input"`
}

type qwenRequestInput struct {
	Text         string `json:"text"`
	Voice        string `json:"voice"`
	LanguageType string `json:"language_type,omitempty"`
}

// qwenResponse non-streaming/streaming unified response structure
type qwenResponse struct {
	StatusCode int        `json:"status_code"`
	RequestID  string     `json:"request_id"`
	Code       string     `json:"code"`
	Message    string     `json:"message"`
	Output     qwenOutput `json:"output"`
	Usage      qwenUsage  `json:"usage"`
}

type qwenOutput struct {
	Text         interface{}   `json:"text"`
	FinishReason string        `json:"finish_reason"`
	Choices      interface{}   `json:"choices"`
	Audio        qwenAudioInfo `json:"audio"`
}

type qwenAudioInfo struct {
	Data      string `json:"data"`       //Base64 audio data (16bit PCM) when streaming output
	URL       string `json:"url"`        //WAV URL for non-streaming output
	ID        string `json:"id"`         //Audio ID
	ExpiresAt int64  `json:"expires_at"` //URL expiration timestamp
}

type qwenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	Characters   int `json:"characters"`
}

// NewQwenTTSProvider creates a new Alibaba Cloud Qianwen TTS provider
func NewQwenTTSProvider(config map[string]interface{}) *QwenTTSProvider {
	apiKey, _ := config["api_key"].(string)
	apiURL, _ := config["api_url"].(string)
	model, _ := config["model"].(string)
	voice, _ := config["voice"].(string)
	languageType, _ := config["language_type"].(string)
	stream, _ := config["stream"].(bool)
	frameDuration, _ := config["frame_duration"].(float64)
	region, _ := config["region"].(string)

	//Handle API URL/Region
	if apiURL == "" {
		if strings.EqualFold(region, "singapore") {
			apiURL = defaultAPIURLSingapore
		} else {
			apiURL = defaultAPIURLBeijing
		}
	}

	//Default value
	if model == "" {
		model = defaultQwenModel
	}
	if voice == "" {
		voice = defaultQwenVoice
	}
	if languageType == "" {
		languageType = defaultQwenLanguageType
	}
	if frameDuration == 0 {
		frameDuration = audio.FrameDuration
	}

	return &QwenTTSProvider{
		APIKey:        apiKey,
		APIURL:        apiURL,
		Model:         model,
		Voice:         voice,
		LanguageType:  languageType,
		Stream:        stream,
		FrameDuration: int(frameDuration),
	}
}

// TextToSpeech non-streaming text-to-speech: calls the HTTP interface, downloads WAV and decodes it into frames
func (p *QwenTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	startTs := time.Now().UnixMilli()

	//Construct request body
	reqBody := qwenRequest{
		Model: p.Model,
		Input: qwenRequestInput{
			Text:         text,
			Voice:        p.Voice,
			LanguageType: p.LanguageType,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Serialization request failed: %v", err)
	}

	//Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Create request failed: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))

	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response: %v", err)
	}

	var ttsResp qwenResponse
	if err := json.Unmarshal(body, &ttsResp); err != nil {
		return nil, fmt.Errorf("Failed to parse response: %v, response body: %s", err, string(body))
	}

	if ttsResp.StatusCode != 200 {
		return nil, fmt.Errorf("Qianwen TTS API error [%s]: %s", ttsResp.Code, ttsResp.Message)
	}

	if ttsResp.Output.Audio.URL == "" {
		return nil, fmt.Errorf("Audio URL not included in response")
	}

	log.Debugf("Qianwen TTS non-streaming, download audio URL: %s", ttsResp.Output.Audio.URL)

	//Download WAV and convert to frames with universal codec
	wavReq, err := http.NewRequestWithContext(ctx, http.MethodGet, ttsResp.Output.Audio.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create audio download request: %v", err)
	}

	wavResp, err := client.Do(wavReq)
	if err != nil {
		return nil, fmt.Errorf("Failed to download audio: %v", err)
	}
	defer wavResp.Body.Close()

	if wavResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(wavResp.Body)
		return nil, fmt.Errorf("Failed to download audio, status code: %d, response: %s", wavResp.StatusCode, string(body))
	}

	outputChan := make(chan []byte, 1000)

	decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, wavResp.Body, outputChan, frameDuration, "wav", sampleRate)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Qianwen audio decoder: %v", err)
	}

	//Start decoding
	go func() {
		if err := decoder.Run(startTs); err != nil {
			log.Errorf("Qianwen TTS non-streaming audio decoding failed: %v", err)
		}
	}()

	var frames [][]byte
	for frame := range outputChan {
		frames = append(frames, frame)
	}

	log.Debugf("Qianwen TTS non-streaming completion, the time taken from input to acquisition of audio data is: %d ms", time.Now().UnixMilli()-startTs)
	return frames, nil
}

// TextToSpeechStream streaming text-to-speech implementation
func (p *QwenTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {

	startTs := time.Now().UnixMilli()

	//Construct request body
	reqBody := qwenRequest{
		Model: p.Model,
		Input: qwenRequestInput{
			Text:         text,
			Voice:        p.Voice,
			LanguageType: p.LanguageType,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Serialization request failed: %v", err)
	}

	//Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Create request failed: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))
	req.Header.Set("X-DashScope-SSE", "enable") //Enable streaming output

	client := getHTTPClient()

	outputChan = make(chan []byte, 100)

	go func() {

		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Failed to send Qianwen streaming request: %v", err)
			close(outputChan)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("Qianwen Streaming API request failed, status code: %d, response: %s", resp.StatusCode, string(body))
			close(outputChan)
			return
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/event-stream") {
			log.Warnf("The Content-Type returned by Qianwen Streaming API is not text/event-stream: %s", contentType)
			close(outputChan)
			return
		}

		//Pipeline: Parse SSE -> PCM -> Decode to frames
		pipeReader, pipeWriter := io.Pipe()

		//Parse SSE and write raw PCM data.
		//The audio.data returned by Qwen streaming may carry a WAV header once in actual measurement, which needs to be stripped first and then processed according to PCM.
		go func() {
			defer func() {
				if err := pipeWriter.Close(); err != nil {
					log.Debugf("Failed to close Qianwen pipeline write end: %v", err)
				}
			}()

			if err := p.parseEventStream(ctx, resp.Body, pipeWriter, text); err != nil {
				log.Errorf("Failed to parse Qianwen Event Stream: %v", err)
			}
		}()

		//Create an audio decoder, read PCM from the pipe, and output opus frames
		decoder, err := util.CreateAudioDecoderWithSampleRate(
			ctx,
			pipeReader,
			outputChan,
			frameDuration,
			"pcm", //parseEventStream will strip the WAV header when needed and output pure 16bit PCM
			sampleRate,
		)
		if err != nil {
			log.Errorf("Failed to create Qianwen streaming audio decoder: %v", err)
			close(outputChan)
			pipeReader.Close()
			return
		}

		//Tells the decoder the sample rate/channel information of the PCM
		decoder.WithFormat(beep.Format{
			SampleRate:  beep.SampleRate(24000),
			NumChannels: 1,
		})

		//decoder.Run() internally closes outputChan
		//Use sync.Once to ensure that even if decoder.Run() closes the channel, defer will not close it repeatedly
		if err := decoder.Run(startTs); err != nil {
			log.Errorf("Qianwen streaming audio decoding failed: %v", err)
			return
		}

		//If decoder.Run() completes successfully, it closes the channel
		//So here you need to cancel the closing operation of defer (already processed through sync.Once)

		select {
		case <-ctx.Done():
			log.Debugf("Qianwen TTS streaming synthesis cancelled, text: %s", text)
			return
		default:
			log.Debugf("Qianwen TTS streaming time: from input to the end of obtaining audio data: %d ms", time.Now().UnixMilli()-startTs)
		}
	}()

	return outputChan, nil
}

// parseEventStream uses go-sse to parse Alibaba Cloud Qianwen's SSE, decode Base64 PCM and write to the pipeline
func (p *QwenTTSProvider) parseEventStream(ctx context.Context, reader io.Reader, writer *io.PipeWriter, text string) error {
	var leadingAudio bytes.Buffer
	wroteLeadingAudio := false

	for ev, evErr := range sse.Read(reader, nil) {
		if evErr != nil {
			return fmt.Errorf("Failed to read Qianwen SSE events: %w", evErr)
		}

		select {
		case <-ctx.Done():
			log.Debugf("Qianwen TTS streaming synthesis cancelled, text: %s", text)
			return ctx.Err()
		default:
		}

		dataValue := strings.TrimSpace(ev.Data)
		if dataValue == "" {
			continue
		}

		var eventResp qwenResponse
		if err := json.Unmarshal([]byte(dataValue), &eventResp); err != nil {
			log.Warnf("Failed to parse Qianwen Event Stream JSON: %v, data: %s", err, previewString(dataValue, 200))
			continue
		}

		//Check the business status code (streaming data may not contain status_code, if not, it will be 0, which is considered successful)
		if eventResp.StatusCode != 0 && eventResp.StatusCode != 200 {
			return fmt.Errorf("Qianwen Streaming API Error [%s]: %s", eventResp.Code, eventResp.Message)
		}

		//Decode Base64 PCM data
		if eventResp.Output.Audio.Data != "" {
			encoded := cleanBase64(eventResp.Output.Audio.Data)
			audioBytes, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				log.Errorf("Decoding Qianwen Base64 PCM failed: %v", err)
				continue
			}

			if len(audioBytes) > 0 {
				if !wroteLeadingAudio {
					leadingAudio.Write(audioBytes)
					normalized, needMore, detectedWAV, err := normalizeLeadingQwenAudio(leadingAudio.Bytes())
					if err != nil {
						return fmt.Errorf("Failed to parse Qianwen streaming audio header: %w", err)
					}
					if needMore {
						continue
					}
					wroteLeadingAudio = true
					if detectedWAV {
						log.Infof("Qianwen streaming audio detects WAV header, which has been stripped and processed according to PCM")
					}
					if len(normalized) == 0 {
						continue
					}
					if _, err := writer.Write(normalized); err != nil {
						return fmt.Errorf("Writing PCM to pipe failed: %v", err)
					}
					continue
				}

				if _, err := writer.Write(audioBytes); err != nil {
					return fmt.Errorf("Writing PCM to pipe failed: %v", err)
				}
			}
		}

		//Check if completed
		if eventResp.Output.FinishReason == "stop" {
			log.Debugf("Qianwen Streaming received finish_reason=stop, request ID: %s", eventResp.RequestID)
			return nil
		}
	}

	return nil
}

func normalizeLeadingQwenAudio(data []byte) (normalized []byte, needMore bool, detectedWAV bool, err error) {
	if len(data) < 12 {
		return nil, true, false, nil
	}

	if !bytes.HasPrefix(data, []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return data, false, false, nil
	}

	offset, needMore, err := qwenWAVDataOffset(data)
	if err != nil {
		return nil, false, true, err
	}
	if needMore {
		return nil, true, true, nil
	}
	if offset > len(data) {
		return nil, false, true, fmt.Errorf("WAV data offset out of bounds: %d > %d", offset, len(data))
	}
	return data[offset:], false, true, nil
}

func qwenWAVDataOffset(data []byte) (offset int, needMore bool, err error) {
	if len(data) < 12 {
		return 0, true, nil
	}
	if !bytes.HasPrefix(data, []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return 0, false, fmt.Errorf("Not a valid WAV header")
	}

	offset = 12
	for {
		if len(data) < offset+8 {
			return 0, true, nil
		}

		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		if chunkSize < 0 {
			return 0, false, fmt.Errorf("Illegal WAV chunk size: %d", chunkSize)
		}
		offset += 8

		if chunkID == "data" {
			return offset, false, nil
		}

		nextOffset := offset + chunkSize
		if chunkSize%2 == 1 {
			nextOffset++
		}
		if len(data) < nextOffset {
			return 0, true, nil
		}
		offset = nextOffset
	}
}

// SetVoice Set the voice
func (p *QwenTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && voice != "" {
		p.Voice = voice
		return nil
	}
	return fmt.Errorf("Invalid voice configuration: missing voice")
}

// Close closes the resource (stateless provider, no need to close)
func (p *QwenTTSProvider) Close() error {
	return nil
}

// IsValid checks whether the resource is valid
func (p *QwenTTSProvider) IsValid() bool {
	return p != nil
}

// cleanBase64 removes all whitespace characters from a Base64 string
func cleanBase64(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t' {
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

// previewString returns the first n characters of the string for logging
func previewString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
