package doubao

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/domain/doubaoapi"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

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
			Timeout:   90 * time.Second,
		}
	})
	return httpClient
}

type DoubaoTTSProvider struct {
	AppID       string
	AccessToken string
	Model       string
	ResourceID  string
	Voice       string
	APIURL      string
}

type doubaoTTSV3Request struct {
	User      doubaoTTSV3User      `json:"user"`
	ReqParams doubaoTTSV3ReqParams `json:"req_params"`
}

type doubaoTTSV3User struct {
	UID string `json:"uid"`
}

type doubaoTTSV3AudioParams struct {
	Format       string `json:"format"`
	SampleRate   int    `json:"sample_rate"`
	SpeechRate   int    `json:"speech_rate,omitempty"`
	LoudnessRate int    `json:"loudness_rate,omitempty"`
}

type doubaoTTSV3ReqParams struct {
	Text        string                 `json:"text"`
	Speaker     string                 `json:"speaker"`
	AudioParams doubaoTTSV3AudioParams `json:"audio_params"`
	Model       string                 `json:"model,omitempty"`
}

type doubaoTTSV3Event struct {
	Code     int     `json:"code"`
	Message  string  `json:"message"`
	Data     *string `json:"data"`
	Sequence int     `json:"sequence,omitempty"`
}

func NewDoubaoTTSProvider(config map[string]interface{}) *DoubaoTTSProvider {
	appID, _ := config["appid"].(string)
	accessToken, _ := config["access_token"].(string)
	model, _ := config["model"].(string)
	resourceID, _ := config["resource_id"].(string)
	voice, _ := config["voice"].(string)
	apiURL, _ := config["api_url"].(string)

	return &DoubaoTTSProvider{
		AppID:       appID,
		AccessToken: accessToken,
		Model:       normalizeDoubaoModel(model),
		ResourceID:  strings.TrimSpace(resourceID),
		Voice:       strings.TrimSpace(voice),
		APIURL:      normalizeDoubaoTTSURL(apiURL, defaultDoubaoHTTPURL),
	}
}

func newDoubaoTTSV3Request(text, speaker string, sampleRate int, requestModel string) doubaoTTSV3Request {
	return doubaoTTSV3Request{
		User: doubaoTTSV3User{UID: doubaoapi.NewRequestID()},
		ReqParams: doubaoTTSV3ReqParams{
			Text:    text,
			Speaker: speaker,
			AudioParams: doubaoTTSV3AudioParams{
				Format:     defaultDoubaoAudioFmt,
				SampleRate: sampleRate,
			},
			Model: requestModel,
		},
	}
}

func (p *DoubaoTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	outputChan, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}
	frames := make([][]byte, 0, 32)
	for frame := range outputChan {
		if len(frame) > 0 {
			frames = append(frames, frame)
		}
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("Doubao TTS returns empty audio")
	}
	return frames, nil
}

func (p *DoubaoTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	return p.streamHTTP(ctx, text, sampleRate, frameDuration)
}

func (p *DoubaoTTSProvider) streamHTTP(ctx context.Context, text string, sampleRate int, frameDuration int) (chan []byte, error) {
	voice := strings.TrimSpace(p.Voice)
	if voice == "" {
		return nil, fmt.Errorf("Doubao TTS missing voice")
	}
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	resolved, err := resolveDoubaoTTSModel(p.Model, voice)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.ResourceID) != "" {
		resolved.ResourceID = strings.TrimSpace(p.ResourceID)
	}
	if sampleRate <= 0 {
		sampleRate = defaultDoubaoSampleHz
	}
	reqBody := newDoubaoTTSV3Request(text, voice, sampleRate, resolved.RequestModel)
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Serialization Doubao TTS request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.APIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("Creation of Doubao TTS request failed: %w", err)
	}
	headers := doubaoapi.NewTTSHeaders(p.AppID, p.AccessToken, resolved.ResourceID)
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := getHTTPClient()
	outputChan := make(chan []byte, 1000)
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Doubao HTTP TTS request failed: %v", err)
			close(outputChan)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			log.Errorf("Doubao HTTP TTS returns error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
			close(outputChan)
			return
		}
		p.decodeStreamResponse(ctx, resp.Body, outputChan, sampleRate, frameDuration)
	}()
	return outputChan, nil
}

func (p *DoubaoTTSProvider) decodeStreamResponse(ctx context.Context, body io.Reader, outputChan chan []byte, sampleRate int, frameDuration int) {
	pipeReader, pipeWriter := io.Pipe()
	startTs := time.Now().UnixMilli()

	decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, defaultDoubaoAudioFmt, sampleRate)
	if err != nil {
		log.Errorf("Failed to create Doubao audio decoder: %v", err)
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		close(outputChan)
		return
	}
	go func() {
		if err := decoder.Run(startTs); err != nil {
			log.Errorf("Doubao audio decoding failed: %v", err)
		}
	}()

	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			_ = pipeWriter.CloseWithError(err)
			return
		}

		line = strings.TrimSpace(line)
		if event, parseErr := parseDoubaoTTSStreamEvent([]byte(line)); parseErr != nil {
			_ = pipeWriter.CloseWithError(parseErr)
			return
		} else if event != nil {
			if writeErr := writeDoubaoTTSAudioChunk(pipeWriter, event); writeErr != nil {
				_ = pipeWriter.CloseWithError(writeErr)
				return
			}
		}

		if err == io.EOF {
			_ = pipeWriter.Close()
			return
		}
	}
}

func parseDoubaoTTSStreamEvent(raw []byte) (*doubaoTTSV3Event, error) {
	line := strings.TrimSpace(string(raw))
	if line == "" {
		return nil, nil
	}
	if strings.HasPrefix(line, "event:") {
		return nil, nil
	}
	if strings.HasPrefix(line, "data:") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	}
	if line == "" || line == "[DONE]" {
		return nil, nil
	}

	var event doubaoTTSV3Event
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return nil, fmt.Errorf("Failed to parse Doubao TTS streaming events: %w", err)
	}
	if event.Code != 0 {
		msg := strings.TrimSpace(event.Message)
		if msg == "" {
			msg = fmt.Sprintf("Doubao TTS returned error code %d", event.Code)
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return &event, nil
}

func writeDoubaoTTSAudioChunk(writer *io.PipeWriter, event *doubaoTTSV3Event) error {
	chunk, err := decodeDoubaoTTSAudioChunk(event)
	if err != nil {
		return err
	}
	if len(chunk) == 0 {
		return nil
	}
	_, err = writer.Write(chunk)
	return err
}

func decodeDoubaoTTSAudioChunk(event *doubaoTTSV3Event) ([]byte, error) {
	if event == nil || event.Data == nil || strings.TrimSpace(*event.Data) == "" {
		return nil, nil
	}
	chunk, err := base64.StdEncoding.DecodeString(strings.TrimSpace(*event.Data))
	if err != nil {
		return nil, fmt.Errorf("Failed to decode Doubao audio data: %w", err)
	}
	return chunk, nil
}

func (p *DoubaoTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && strings.TrimSpace(voice) != "" {
		p.Voice = strings.TrimSpace(voice)
	}
	if model, ok := voiceConfig["model"].(string); ok && strings.TrimSpace(model) != "" {
		p.Model = normalizeDoubaoModel(model)
	}
	if resourceID, ok := voiceConfig["resource_id"].(string); ok {
		p.ResourceID = strings.TrimSpace(resourceID)
	}
	return nil
}

func (p *DoubaoTTSProvider) Close() error {
	return nil
}

func (p *DoubaoTTSProvider) IsValid() bool {
	return p != nil
}
