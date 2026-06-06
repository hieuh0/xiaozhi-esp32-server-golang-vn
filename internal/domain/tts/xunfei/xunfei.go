package xunfei

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gopxl/beep"
	"github.com/gorilla/websocket"
)

const (
	defaultXunfeiWSURL          = "wss://tts-api.xfyun.cn/v2/tts"
	defaultXunfeiVoice          = "xiaoyan"
	defaultXunfeiAudioEncoding  = "raw"
	defaultXunfeiSampleRate     = 16000
	defaultXunfeiSpeed          = 50
	defaultXunfeiVolume         = 50
	defaultXunfeiPitch          = 50
	defaultXunfeiTTE            = "UTF8"
	defaultXunfeiFrameDuration  = audio.FrameDuration
	defaultXunfeiConnectTimeout = 10
	defaultXunfeiReadTimeout    = 30
)

var defaultXunfeiDialer = websocket.Dialer{
	ReadBufferSize:   16 * 1024,
	WriteBufferSize:  16 * 1024,
	HandshakeTimeout: defaultXunfeiConnectTimeout * time.Second,
}

type XunfeiTTSProvider struct {
	AppID                  string
	APIKey                 string
	APISecret              string
	WSURL                  string
	Voice                  string
	AudioEncoding          string
	AUE                    string
	SampleRate             int
	Speed                  int
	Volume                 int
	Pitch                  int
	TTE                    string
	Reg                    int
	Rdn                    int
	FrameDuration          int
	ConnectTimeout         int
	ReadTimeout            int
	ExpectedOpusPayloadLen int
}

type xunfeiRequest struct {
	Common   xunfeiCommon   `json:"common"`
	Business xunfeiBusiness `json:"business"`
	Data     xunfeiDataReq  `json:"data"`
}

type xunfeiCommon struct {
	AppID string `json:"app_id"`
}

type xunfeiBusiness struct {
	AUE    string `json:"aue"`
	AUF    string `json:"auf"`
	VCN    string `json:"vcn"`
	Speed  int    `json:"speed,omitempty"`
	Volume int    `json:"volume,omitempty"`
	Pitch  int    `json:"pitch,omitempty"`
	TTE    string `json:"tte,omitempty"`
	Reg    int    `json:"reg,omitempty"`
	Rdn    int    `json:"rdn,omitempty"`
}

type xunfeiDataReq struct {
	Status int    `json:"status"`
	Text   string `json:"text"`
}

type xunfeiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	SID     string          `json:"sid"`
	Data    *xunfeiRespData `json:"data"`
}

type xunfeiRespData struct {
	Audio  string `json:"audio"`
	Ced    string `json:"ced"`
	Status int    `json:"status"`
}

func NewXunfeiTTSProvider(config map[string]interface{}) *XunfeiTTSProvider {
	provider := &XunfeiTTSProvider{
		AppID:          strings.TrimSpace(getString(config, "app_id", "")),
		APIKey:         strings.TrimSpace(getString(config, "api_key", "")),
		APISecret:      strings.TrimSpace(getString(config, "api_secret", "")),
		WSURL:          strings.TrimSpace(getString(config, "ws_url", defaultXunfeiWSURL)),
		Voice:          strings.TrimSpace(getString(config, "voice", defaultXunfeiVoice)),
		AudioEncoding:  strings.ToLower(strings.TrimSpace(getString(config, "audio_encoding", defaultXunfeiAudioEncoding))),
		SampleRate:     getInt(config, "sample_rate", defaultXunfeiSampleRate),
		Speed:          getInt(config, "speed", defaultXunfeiSpeed),
		Volume:         getInt(config, "volume", defaultXunfeiVolume),
		Pitch:          getInt(config, "pitch", defaultXunfeiPitch),
		TTE:            strings.TrimSpace(getString(config, "tte", defaultXunfeiTTE)),
		Reg:            getInt(config, "reg", 0),
		Rdn:            getInt(config, "rdn", 0),
		FrameDuration:  getInt(config, "frame_duration", defaultXunfeiFrameDuration),
		ConnectTimeout: getInt(config, "connect_timeout", defaultXunfeiConnectTimeout),
		ReadTimeout:    getInt(config, "read_timeout", defaultXunfeiReadTimeout),
	}

	if provider.WSURL == "" {
		provider.WSURL = defaultXunfeiWSURL
	}
	if provider.Voice == "" {
		provider.Voice = defaultXunfeiVoice
	}
	if provider.AudioEncoding == "" {
		provider.AudioEncoding = defaultXunfeiAudioEncoding
	}
	if provider.SampleRate != 8000 && provider.SampleRate != 16000 {
		provider.SampleRate = defaultXunfeiSampleRate
	}
	if provider.TTE == "" {
		provider.TTE = defaultXunfeiTTE
	}
	if provider.FrameDuration <= 0 {
		provider.FrameDuration = defaultXunfeiFrameDuration
	}
	if provider.ConnectTimeout <= 0 {
		provider.ConnectTimeout = defaultXunfeiConnectTimeout
	}
	if provider.ReadTimeout <= 0 {
		provider.ReadTimeout = defaultXunfeiReadTimeout
	}

	aue, expectedPayloadLen, err := mapXunfeiAudioEncoding(provider.AudioEncoding, provider.SampleRate)
	if err != nil {
		log.Warnf("Initialization of xunfei TTS configuration failed, falling back to raw/16k: %v", err)
		provider.AudioEncoding = defaultXunfeiAudioEncoding
		provider.SampleRate = defaultXunfeiSampleRate
		aue = "raw"
		expectedPayloadLen = 0
	}
	provider.AUE = aue
	provider.ExpectedOpusPayloadLen = expectedPayloadLen

	return provider
}

func (p *XunfeiTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	outputChan, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}

	audioFrames := make([][]byte, 0, 32)
	for frame := range outputChan {
		audioFrames = append(audioFrames, frame)
	}
	if len(audioFrames) == 0 {
		return nil, fmt.Errorf("xunfei TTS returns empty audio")
	}
	return audioFrames, nil
}

func (p *XunfeiTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	if strings.TrimSpace(text) == "" {
		outputChan = make(chan []byte)
		close(outputChan)
		return outputChan, nil
	}
	if err := p.validate(); err != nil {
		return nil, err
	}

	targetSampleRate := sampleRate
	if targetSampleRate <= 0 {
		targetSampleRate = p.SampleRate
	}
	targetFrameDuration := frameDuration
	if targetFrameDuration <= 0 {
		targetFrameDuration = p.FrameDuration
	}

	outputChan = make(chan []byte, 100)
	startTs := time.Now().UnixMilli()

	go func() {
		if err := p.streamSynthesis(ctx, text, targetSampleRate, targetFrameDuration, startTs, outputChan); err != nil && ctx.Err() == nil {
			log.Errorf("xunfei TTS streaming synthesis failed: %v", err)
		}
	}()

	return outputChan, nil
}

func (p *XunfeiTTSProvider) streamSynthesis(ctx context.Context, text string, targetSampleRate int, frameDuration int, startTs int64, outputChan chan []byte) error {
	conn, err := p.dial(ctx)
	if err != nil {
		close(outputChan)
		return err
	}

	var closeOnce sync.Once
	closeConn := func() {
		closeOnce.Do(func() {
			_ = conn.Close()
		})
	}
	defer closeConn()

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			closeConn()
		case <-done:
		}
	}()

	pipeReader, pipeWriter := io.Pipe()

	audioFormat := "pcm"
	if p.AudioEncoding == "opus" {
		audioFormat = "opus"
	}

	decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, audioFormat, targetSampleRate)
	if err != nil {
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		close(outputChan)
		return fmt.Errorf("Failed to create xunfei audio decoder: %v", err)
	}
	decoder.WithFormat(beep.Format{
		SampleRate:  beep.SampleRate(p.SampleRate),
		NumChannels: 1,
	})

	decoderDone := make(chan struct{})
	go func() {
		defer close(decoderDone)
		if err := decoder.Run(startTs); err != nil && ctx.Err() == nil {
			log.Errorf("xunfei Audio decoding failed: %v", err)
		}
	}()

	if err := p.sendSynthesisRequest(conn, text); err != nil {
		_ = pipeWriter.CloseWithError(err)
		<-decoderDone
		return err
	}

	streamErr := p.readSynthesisResponse(ctx, conn, pipeWriter)
	if streamErr != nil {
		_ = pipeWriter.CloseWithError(streamErr)
	} else {
		_ = pipeWriter.Close()
	}
	<-decoderDone

	if streamErr == nil && ctx.Err() == nil {
		log.Infof("xunfei TTS time consuming: Time consuming from input to acquisition of audio data: %d ms", time.Now().UnixMilli()-startTs)
	}

	return streamErr
}

func (p *XunfeiTTSProvider) sendSynthesisRequest(conn *websocket.Conn, text string) error {
	reqBody := xunfeiRequest{
		Common: xunfeiCommon{
			AppID: p.AppID,
		},
		Business: xunfeiBusiness{
			AUE:    p.AUE,
			AUF:    fmt.Sprintf("audio/L16;rate=%d", p.SampleRate),
			VCN:    p.Voice,
			Speed:  p.Speed,
			Volume: p.Volume,
			Pitch:  p.Pitch,
			TTE:    p.TTE,
			Reg:    p.Reg,
			Rdn:    p.Rdn,
		},
		Data: xunfeiDataReq{
			Status: 2,
			Text:   base64.StdEncoding.EncodeToString([]byte(text)),
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("Serialization xunfei request failed: %v", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		return fmt.Errorf("Failed to send xunfei request: %v", err)
	}
	return nil
}

func (p *XunfeiTTSProvider) readSynthesisResponse(ctx context.Context, conn *websocket.Conn, pipeWriter *io.PipeWriter) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if p.ReadTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(time.Duration(p.ReadTimeout) * time.Second))
		}

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("Failed to read xunfei WebSocket message: %v", err)
		}
		if messageType != websocket.TextMessage {
			continue
		}

		var resp xunfeiResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			return fmt.Errorf("Failed to parse xunfei response: %v, body=%s", err, previewString(string(message), 300))
		}
		if resp.Code != 0 {
			return fmt.Errorf("xunfei TTS error [%d]: %s", resp.Code, strings.TrimSpace(resp.Message))
		}
		if resp.Data == nil {
			continue
		}

		audioData := cleanBase64(resp.Data.Audio)
		if audioData != "" {
			chunk, err := base64.StdEncoding.DecodeString(audioData)
			if err != nil {
				return fmt.Errorf("Decoding xunfei audio Base64 failed: %v", err)
			}

			if p.AudioEncoding == "raw" {
				if _, err := pipeWriter.Write(chunk); err != nil {
					return fmt.Errorf("Failed to write xunfei PCM data: %v", err)
				}
			} else {
				frames, err := p.decodeXunfeiOpusFrames(chunk)
				if err != nil {
					return fmt.Errorf("Failed to parse xunfei Opus data: %v", err)
				}
				for _, frame := range frames {
					if err := util.WriteLengthPrefixedFrame(pipeWriter, frame); err != nil {
						return fmt.Errorf("Writing Opus frame to audio chain failed: %v", err)
					}
				}
			}
		}

		if resp.Data.Status == 2 {
			return nil
		}
	}
}

func (p *XunfeiTTSProvider) decodeXunfeiOpusFrames(chunk []byte) ([][]byte, error) {
	if len(chunk) == 0 {
		return nil, nil
	}

	if p.ExpectedOpusPayloadLen > 0 && len(chunk) == p.ExpectedOpusPayloadLen {
		frame := make([]byte, len(chunk))
		copy(frame, chunk)
		return [][]byte{frame}, nil
	}

	frames := make([][]byte, 0, 4)
	offset := 0
	for offset < len(chunk) {
		if len(chunk)-offset < 2 {
			if len(frames) == 0 && p.ExpectedOpusPayloadLen > 0 && len(chunk[offset:]) == p.ExpectedOpusPayloadLen {
				frame := make([]byte, len(chunk[offset:]))
				copy(frame, chunk[offset:])
				return [][]byte{frame}, nil
			}
			return nil, fmt.Errorf("The remaining data is not enough to read the frame header: remain=%d", len(chunk)-offset)
		}

		payloadLen, ok := selectXunfeiPayloadLength(chunk[offset:offset+2], len(chunk)-offset-2, p.ExpectedOpusPayloadLen)
		if !ok {
			if len(frames) == 0 && p.ExpectedOpusPayloadLen > 0 && len(chunk[offset:]) == p.ExpectedOpusPayloadLen {
				frame := make([]byte, len(chunk[offset:]))
				copy(frame, chunk[offset:])
				return [][]byte{frame}, nil
			}
			headerEnd := offset + 2
			if headerEnd > len(chunk) {
				headerEnd = len(chunk)
			}
			return nil, fmt.Errorf("Unrecognized Opus frame length header: %v", chunk[offset:headerEnd])
		}

		start := offset + 2
		end := start + payloadLen
		if end > len(chunk) {
			return nil, fmt.Errorf("Opus frame length is out of bounds: offset=%d payload=%d total=%d", offset, payloadLen, len(chunk))
		}

		frame := make([]byte, payloadLen)
		copy(frame, chunk[start:end])
		frames = append(frames, frame)
		offset = end
	}

	if len(frames) == 0 {
		return nil, fmt.Errorf("No Opus frames were parsed out")
	}
	return frames, nil
}

func selectXunfeiPayloadLength(header []byte, remaining int, expected int) (int, bool) {
	if len(header) < 2 || remaining <= 0 {
		return 0, false
	}

	candidates := []int{
		int(binary.LittleEndian.Uint16(header)),
		int(binary.BigEndian.Uint16(header)),
	}

	seen := make(map[int]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate <= 0 || candidate > remaining {
			continue
		}
		if expected > 0 && candidate != expected {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		return candidate, true
	}

	return 0, false
}

func (p *XunfeiTTSProvider) dial(ctx context.Context) (*websocket.Conn, error) {
	signedURL, err := p.buildSignedURL()
	if err != nil {
		return nil, err
	}

	dialer := defaultXunfeiDialer
	if p.ConnectTimeout > 0 {
		dialer.HandshakeTimeout = time.Duration(p.ConnectTimeout) * time.Second
	}

	conn, resp, err := dialer.DialContext(ctx, signedURL, nil)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("Failed to connect to xunfei WebSocket, status code: %d, response: %s, err: %v", resp.StatusCode, string(body), err)
		}
		return nil, fmt.Errorf("Failed to connect to xunfei WebSocket: %v", err)
	}
	return conn, nil
}

func (p *XunfeiTTSProvider) buildSignedURL() (string, error) {
	parsed, err := url.Parse(p.WSURL)
	if err != nil {
		return "", fmt.Errorf("Invalid xunfei ws_url: %v", err)
	}

	host := parsed.Host
	if host == "" {
		return "", fmt.Errorf("xunfei ws_url is missing host")
	}

	requestURI := parsed.EscapedPath()
	if requestURI == "" {
		requestURI = "/"
	}
	if parsed.RawQuery != "" {
		requestURI += "?" + parsed.RawQuery
	}

	date := time.Now().UTC().Format(http.TimeFormat)
	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\nGET %s HTTP/1.1", host, date, requestURI)
	mac := hmac.New(sha256.New, []byte(p.APISecret))
	_, _ = mac.Write([]byte(signatureOrigin))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	authorizationOrigin := fmt.Sprintf(
		`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`,
		p.APIKey,
		signature,
	)

	query := parsed.Query()
	query.Set("authorization", base64.StdEncoding.EncodeToString([]byte(authorizationOrigin)))
	query.Set("date", date)
	query.Set("host", host)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func (p *XunfeiTTSProvider) validate() error {
	if p == nil {
		return fmt.Errorf("xunfei provider cannot be empty")
	}
	if p.AppID == "" {
		return fmt.Errorf("xunfei app_id cannot be empty")
	}
	if p.APIKey == "" {
		return fmt.Errorf("xunfei api_key cannot be empty")
	}
	if p.APISecret == "" {
		return fmt.Errorf("xunfei api_secret cannot be empty")
	}
	if _, _, err := mapXunfeiAudioEncoding(p.AudioEncoding, p.SampleRate); err != nil {
		return err
	}
	return nil
}

func (p *XunfeiTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && strings.TrimSpace(voice) != "" {
		p.Voice = strings.TrimSpace(voice)
		return nil
	}
	return fmt.Errorf("Invalid voice configuration: missing voice")
}

func (p *XunfeiTTSProvider) Close() error {
	return nil
}

func (p *XunfeiTTSProvider) IsValid() bool {
	return p != nil
}

func mapXunfeiAudioEncoding(audioEncoding string, sampleRate int) (string, int, error) {
	switch strings.ToLower(strings.TrimSpace(audioEncoding)) {
	case "", "raw":
		if sampleRate != 8000 && sampleRate != 16000 {
			return "", 0, fmt.Errorf("xunfei raw only supports 8000/16000 sampling rate, currently: %d", sampleRate)
		}
		return "raw", 0, nil
	case "opus":
		switch sampleRate {
		case 8000:
			return "opus", 20, nil
		case 16000:
			return "opus-wb", 40, nil
		default:
			return "", 0, fmt.Errorf("xunfei opus only supports 8000/16000 sampling rate, currently: %d", sampleRate)
		}
	default:
		return "", 0, fmt.Errorf("Unsupported xunfei audio_encoding: %s", audioEncoding)
	}
}

func getString(config map[string]interface{}, key string, defaultValue string) string {
	if config == nil {
		return defaultValue
	}

	value, ok := config[key]
	if !ok || value == nil {
		return defaultValue
	}

	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func getInt(config map[string]interface{}, key string, defaultValue int) int {
	if config == nil {
		return defaultValue
	}

	value, ok := config[key]
	if !ok || value == nil {
		return defaultValue
	}

	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return int(i)
		}
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return i
		}
	}

	return defaultValue
}

func cleanBase64(s string) string {
	if s == "" {
		return s
	}
	var builder strings.Builder
	builder.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t' {
			continue
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func previewString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
