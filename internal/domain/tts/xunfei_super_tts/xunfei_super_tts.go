package xunfei_super_tts

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
	"unicode"

	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/domain/tts/streaming"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gopxl/beep"
	"github.com/gorilla/websocket"
)

const (
	defaultXunfeiSuperWSURL          = "wss://cbm01.cn-huabei-1.xf-yun.com/v1/private/mcd9m97e6"
	defaultXunfeiSuperVoice          = "x6_lingxiaoxue_pro"
	defaultXunfeiSuperAudioEncoding  = "raw"
	defaultXunfeiSuperSampleRate     = 24000
	defaultXunfeiSuperSpeed          = 50
	defaultXunfeiSuperVolume         = 50
	defaultXunfeiSuperPitch          = 50
	defaultXunfeiSuperFrameDuration  = audio.FrameDuration
	defaultXunfeiSuperConnectTimeout = 10
	defaultXunfeiSuperReadTimeout    = 30
	defaultXunfeiSuperOralLevel      = "mid"
	maxXunfeiSuperTextBytes          = 64 * 1024
)

var defaultXunfeiSuperDialer = websocket.Dialer{
	ReadBufferSize:   16 * 1024,
	WriteBufferSize:  16 * 1024,
	HandshakeTimeout: defaultXunfeiSuperConnectTimeout * time.Second,
}

type XunfeiSuperTTSProvider struct {
	AppID                  string
	APIKey                 string
	APISecret              string
	WSURL                  string
	Voice                  string
	AudioEncoding          string
	Encoding               string
	SampleRate             int
	Speed                  int
	Volume                 int
	Pitch                  int
	Bgs                    int
	Reg                    int
	Rdn                    int
	Rhy                    int
	OralLevel              string
	SparkAssist            int
	StopSplit              int
	Remain                 int
	FrameDuration          int
	ConnectTimeout         int
	ReadTimeout            int
	ExpectedOpusPayloadLen int

	connMu      sync.Mutex
	synthesisMu sync.Mutex
	conn        *websocket.Conn
}

type xunfeiSuperRequest struct {
	Header    xunfeiSuperHeader    `json:"header"`
	Parameter xunfeiSuperParameter `json:"parameter"`
	Payload   xunfeiSuperPayload   `json:"payload"`
}

type xunfeiSuperHeader struct {
	AppID   string `json:"app_id"`
	Status  int    `json:"status"` //Xunfei requires header.status to be included in the request, 0 means the first frame cannot be omitted
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	SID     string `json:"sid,omitempty"`
}

type xunfeiSuperParameter struct {
	Oral *xunfeiSuperOralParam `json:"oral,omitempty"`
	TTS  xunfeiSuperTTSParam   `json:"tts"`
}

type xunfeiSuperOralParam struct {
	OralLevel   string `json:"oral_level,omitempty"`
	SparkAssist int    `json:"spark_assist,omitempty"`
	StopSplit   int    `json:"stop_split,omitempty"`
	Remain      int    `json:"remain,omitempty"`
}

type xunfeiSuperTTSParam struct {
	VCN    string                `json:"vcn"`
	Speed  int                   `json:"speed,omitempty"`
	Volume int                   `json:"volume,omitempty"`
	Pitch  int                   `json:"pitch,omitempty"`
	Bgs    int                   `json:"bgs,omitempty"`
	Reg    int                   `json:"reg,omitempty"`
	Rdn    int                   `json:"rdn,omitempty"`
	Rhy    int                   `json:"rhy,omitempty"`
	Audio  xunfeiSuperAudioParam `json:"audio"`
}

type xunfeiSuperAudioParam struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels,omitempty"`
	BitDepth   int    `json:"bit_depth,omitempty"`
	FrameSize  int    `json:"frame_size,omitempty"`
}

type xunfeiSuperPayload struct {
	Text  xunfeiSuperTextPayload   `json:"text"`
	Audio *xunfeiSuperAudioResp    `json:"audio,omitempty"`
	Pybuf *xunfeiSuperPybufPayload `json:"pybuf,omitempty"` //Returned when rhy=1, base64 encoded phoneme/pinyin
}

// xunfeiSuperPybufPayload responds to payload.pybuf. For documentation, see https://www.xfyun.cn/doc/spark/super%20smart-tts.html
type xunfeiSuperPybufPayload struct {
	Encoding string `json:"encoding"`
	Compress string `json:"compress"`
	Format   string `json:"format"`
	Status   int    `json:"status"`
	Seq      int    `json:"seq"`
	Text     string `json:"text"` //Base64 encoding, decoded to phoneme information
}

type xunfeiSuperTextPayload struct {
	Encoding string `json:"encoding"`
	Compress string `json:"compress"`
	Format   string `json:"format"`
	Status   int    `json:"status"`
	Seq      int    `json:"seq"`
	Text     string `json:"text"`
}

type xunfeiSuperResponse struct {
	Header  xunfeiSuperHeader  `json:"header"`
	Payload xunfeiSuperPayload `json:"payload"`
}

type xunfeiSuperAudioResp struct {
	Audio      string `json:"audio"`
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	BitDepth   int    `json:"bit_depth"`
	FrameSize  int    `json:"frame_size"`
	Status     int    `json:"status"`
	Seq        int    `json:"seq"`
	Ced        string `json:"ced,omitempty"`
}

type xunfeiSuperSentenceSpan struct {
	Text      string
	StartByte int
	EndByte   int
	Started   bool
	Ended     bool
	MinFrames int
}

type xunfeiSuperSentenceTracker struct {
	mu              sync.Mutex
	spans           []*xunfeiSuperSentenceSpan
	totalBytes      int
	activeIdx       int
	activeFrames    int
	frameDurationMs int
}

func NewXunfeiSuperTTSProvider(config map[string]interface{}) *XunfeiSuperTTSProvider {
	provider := &XunfeiSuperTTSProvider{
		AppID:          strings.TrimSpace(getString(config, "app_id", "")),
		APIKey:         strings.TrimSpace(getString(config, "api_key", "")),
		APISecret:      strings.TrimSpace(getString(config, "api_secret", "")),
		WSURL:          strings.TrimSpace(getString(config, "ws_url", defaultXunfeiSuperWSURL)),
		Voice:          strings.TrimSpace(getString(config, "voice", defaultXunfeiSuperVoice)),
		AudioEncoding:  strings.ToLower(strings.TrimSpace(getString(config, "audio_encoding", defaultXunfeiSuperAudioEncoding))),
		SampleRate:     getInt(config, "sample_rate", defaultXunfeiSuperSampleRate),
		Speed:          getInt(config, "speed", defaultXunfeiSuperSpeed),
		Volume:         getInt(config, "volume", defaultXunfeiSuperVolume),
		Pitch:          getInt(config, "pitch", defaultXunfeiSuperPitch),
		Bgs:            getInt(config, "bgs", 0),
		Reg:            getInt(config, "reg", 0),
		Rdn:            getInt(config, "rdn", 0),
		Rhy:            getInt(config, "rhy", 0),
		OralLevel:      strings.TrimSpace(getString(config, "oral_level", defaultXunfeiSuperOralLevel)),
		SparkAssist:    getInt(config, "spark_assist", 1),
		StopSplit:      getInt(config, "stop_split", 0),
		Remain:         getInt(config, "remain", 0),
		FrameDuration:  getInt(config, "frame_duration", defaultXunfeiSuperFrameDuration),
		ConnectTimeout: getInt(config, "connect_timeout", defaultXunfeiSuperConnectTimeout),
		ReadTimeout:    getInt(config, "read_timeout", defaultXunfeiSuperReadTimeout),
	}

	if provider.WSURL == "" {
		provider.WSURL = defaultXunfeiSuperWSURL
	}
	if provider.Voice == "" {
		provider.Voice = defaultXunfeiSuperVoice
	}
	if provider.AudioEncoding == "" {
		provider.AudioEncoding = defaultXunfeiSuperAudioEncoding
	}
	if provider.SampleRate != 8000 && provider.SampleRate != 16000 && provider.SampleRate != 24000 {
		provider.SampleRate = defaultXunfeiSuperSampleRate
	}
	if provider.OralLevel == "" {
		provider.OralLevel = defaultXunfeiSuperOralLevel
	}
	if provider.FrameDuration <= 0 {
		provider.FrameDuration = defaultXunfeiSuperFrameDuration
	}
	if provider.ConnectTimeout <= 0 {
		provider.ConnectTimeout = defaultXunfeiSuperConnectTimeout
	}
	if provider.ReadTimeout <= 0 {
		provider.ReadTimeout = defaultXunfeiSuperReadTimeout
	}

	encoding, expectedPayloadLen, err := mapXunfeiSuperAudioEncoding(provider.AudioEncoding, provider.SampleRate)
	if err != nil {
		log.Warnf("Initialization of xunfei_super_tts configuration failed, falling back to raw/24k: %v", err)
		provider.AudioEncoding = defaultXunfeiSuperAudioEncoding
		provider.SampleRate = defaultXunfeiSuperSampleRate
		encoding = "raw"
		expectedPayloadLen = 0
	}
	provider.Encoding = encoding
	provider.ExpectedOpusPayloadLen = expectedPayloadLen

	return provider
}

func (p *XunfeiSuperTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	outputChan, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}

	audioFrames := make([][]byte, 0, 32)
	for frame := range outputChan {
		audioFrames = append(audioFrames, frame)
	}
	if len(audioFrames) == 0 {
		return nil, fmt.Errorf("xunfei_super_tts returns empty audio")
	}
	return audioFrames, nil
}

func (p *XunfeiSuperTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	if strings.TrimSpace(text) == "" {
		outputChan := make(chan []byte)
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

	outputChan := make(chan []byte, 100)
	startTs := time.Now().UnixMilli()

	go func() {
		if err := p.streamSynthesis(ctx, text, targetSampleRate, targetFrameDuration, startTs, outputChan); err != nil && ctx.Err() == nil {
			log.Errorf("xunfei_super_tts streaming synthesis failed: %v", err)
		}
	}()

	return outputChan, nil
}

func (p *XunfeiSuperTTSProvider) streamSynthesis(ctx context.Context, text string, targetSampleRate int, frameDuration int, startTs int64, outputChan chan []byte) error {
	p.synthesisMu.Lock()
	defer p.synthesisMu.Unlock()

	//Xunfei super-anthropomorphic connection will end the input channel after a single synthesis. Multiplexing the connection across sentences will cause subsequent requests to fail directly.
	//Therefore, each synthesis uses an independent connection; multiple pieces of text within a single synthesis still reuse this connection.
	conn, err := p.reconnect(ctx)
	if err != nil {
		close(outputChan)
		return err
	}
	defer p.invalidateConnection(conn)

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			p.invalidateConnection(nil)
		case <-done:
		}
	}()

	pipeReader, pipeWriter := io.Pipe()
	var (
		decoderStarted    bool
		decoderAudioFmt   string
		decoderSampleRate int
		decoderDone       chan struct{}
	)

	ensureDecoder := func(sourceEncoding string, sourceSampleRate int) error {
		audioFormat, err := mapXunfeiSuperResponseAudioFormat(sourceEncoding)
		if err != nil {
			return err
		}
		if sourceSampleRate <= 0 {
			sourceSampleRate = p.SampleRate
		}

		if decoderStarted {
			if audioFormat != decoderAudioFmt {
				return fmt.Errorf("xunfei_super_tts returns audio encoding changes: %s -> %s", decoderAudioFmt, audioFormat)
			}
			if audioFormat != "mp3" && sourceSampleRate != decoderSampleRate {
				return fmt.Errorf("xunfei_super_tts returns a change in sampling rate: %d -> %d", decoderSampleRate, sourceSampleRate)
			}
			return nil
		}

		decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, audioFormat, targetSampleRate)
		if err != nil {
			return fmt.Errorf("Failed to create xunfei_super_tts audio decoder: %v", err)
		}
		if audioFormat != "mp3" {
			decoder.WithFormat(beep.Format{
				SampleRate:  beep.SampleRate(sourceSampleRate),
				NumChannels: 1,
			})
		}

		decoderStarted = true
		decoderAudioFmt = audioFormat
		decoderSampleRate = sourceSampleRate
		decoderDone = make(chan struct{})
		go func() {
			defer close(decoderDone)
			if err := decoder.Run(startTs); err != nil && ctx.Err() == nil {
				log.Errorf("xunfei_super_tts audio decoding failed: %v", err)
			}
		}()
		return nil
	}

	finishDecoder := func(closeErr error) {
		if closeErr != nil {
			_ = pipeWriter.CloseWithError(closeErr)
		} else {
			_ = pipeWriter.Close()
		}
		if decoderStarted {
			<-decoderDone
			return
		}
		_ = pipeReader.Close()
		close(outputChan)
	}

	if err := p.sendSynthesisRequestWithRetry(ctx, text); err != nil {
		finishDecoder(err)
		return err
	}

	streamErr := p.readSynthesisResponse(ctx, pipeWriter, ensureDecoder, nil)
	finishDecoder(streamErr)

	if streamErr == nil && ctx.Err() == nil {
		log.Infof("xunfei_super_tts Time consumption: Time consumption from input to acquisition of audio data: %d ms", time.Now().UnixMilli()-startTs)
	}

	return streamErr
}

func (p *XunfeiSuperTTSProvider) sendSynthesisRequestWithRetry(ctx context.Context, text string) error {
	reqBodies, err := p.buildSynthesisRequests(text)
	if err != nil {
		return err
	}

	payloads := make([][]byte, 0, len(reqBodies))
	for _, reqBody := range reqBodies {
		payload, marshalErr := json.Marshal(reqBody)
		if marshalErr != nil {
			return fmt.Errorf("Serialization xunfei_super_tts request failed: %v", marshalErr)
		}
		payloads = append(payloads, payload)
	}

	sendAll := func(conn *websocket.Conn) error {
		for _, payload := range payloads {
			if err := p.writeRequest(conn, payload); err != nil {
				return err
			}
		}
		return nil
	}

	conn, err := p.ensureConnection(ctx)
	if err != nil {
		return err
	}
	if err := sendAll(conn); err == nil {
		return nil
	}

	p.invalidateConnection(conn)

	conn, err = p.reconnect(ctx)
	if err != nil {
		return err
	}
	if err := sendAll(conn); err != nil {
		p.invalidateConnection(conn)
		return err
	}
	return nil
}

func (p *XunfeiSuperTTSProvider) buildSingleSynthesisRequest(text string, seq int, status int) xunfeiSuperRequest {
	req := xunfeiSuperRequest{
		Header: xunfeiSuperHeader{
			AppID:  p.AppID,
			Status: status,
		},
		Parameter: xunfeiSuperParameter{
			TTS: xunfeiSuperTTSParam{
				VCN:    p.Voice,
				Speed:  p.Speed,
				Volume: p.Volume,
				Pitch:  p.Pitch,
				Bgs:    p.Bgs,
				Reg:    p.Reg,
				Rdn:    p.Rdn,
				Rhy:    p.Rhy,
				Audio: xunfeiSuperAudioParam{
					Encoding:   p.Encoding,
					SampleRate: p.SampleRate,
					Channels:   1,
					BitDepth:   16,
				},
			},
		},
		Payload: xunfeiSuperPayload{
			Text: xunfeiSuperTextPayload{
				Encoding: "utf8",
				Compress: "raw",
				Format:   "plain",
				Status:   status,
				Seq:      seq,
				Text:     base64.StdEncoding.EncodeToString([]byte(text)),
			},
		},
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(p.Voice)), "x4_") {
		req.Parameter.Oral = &xunfeiSuperOralParam{
			OralLevel:   p.OralLevel,
			SparkAssist: p.SparkAssist,
			StopSplit:   p.StopSplit,
			Remain:      p.Remain,
		}
	}
	return req
}

func (p *XunfeiSuperTTSProvider) buildSynthesisRequests(text string) ([]xunfeiSuperRequest, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, fmt.Errorf("xunfei_super_tts text cannot be empty")
	}
	if len([]byte(trimmed)) > maxXunfeiSuperTextBytes {
		return nil, fmt.Errorf("xunfei_super_tts text exceeds 64KB limit, currently: %d bytes", len([]byte(trimmed)))
	}

	return []xunfeiSuperRequest{
		p.buildSingleSynthesisRequest(trimmed, 0, 2),
	}, nil
}

func (p *XunfeiSuperTTSProvider) readSynthesisResponse(ctx context.Context, pipeWriter *io.PipeWriter, ensureDecoder func(string, int) error, beforeAudio func(*xunfeiSuperAudioResp, bool) error) error {
	for {
		select {
		case <-ctx.Done():
			p.invalidateConnection(nil)
			return ctx.Err()
		default:
		}

		conn := p.currentConn()
		if conn == nil {
			return fmt.Errorf("xunfei_super_tts connection has been disconnected")
		}

		if p.ReadTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(time.Duration(p.ReadTimeout) * time.Second))
		}

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				p.invalidateConnection(conn)
				return ctx.Err()
			}
			p.invalidateConnection(conn)
			return fmt.Errorf("Failed to read xunfei_super_tts WebSocket message: %v", err)
		}
		if messageType != websocket.TextMessage {
			continue
		}

		var resp xunfeiSuperResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			return fmt.Errorf("Failed to parse xunfei_super_tts response: %v, body=%s", err, previewString(string(message), 300))
		}
		if resp.Header.Code != 0 {
			return fmt.Errorf("xunfei_super_tts error [%d]: %s", resp.Header.Code, strings.TrimSpace(resp.Header.Message))
		}

		audioResp := resp.Payload.Audio
		if audioResp == nil {
			if resp.Header.Status == 2 {
				return nil
			}
			continue
		}

		//log.Infof("xunfei_super_tts payload.audio: status=%d seq=%d encoding=%s sample_rate=%d channels=%d bit_depth=%d frame_size=%d ced=%s audio_base64_len=%d",
		//	audioResp.Status, audioResp.Seq, audioResp.Encoding, audioResp.SampleRate, audioResp.Channels, audioResp.BitDepth, audioResp.FrameSize, strings.TrimSpace(audioResp.Ced), len(audioResp.Audio))

		audioData := cleanBase64(audioResp.Audio)
		if audioData != "" {
			chunk, err := base64.StdEncoding.DecodeString(audioData)
			if err != nil {
				return fmt.Errorf("Decoding xunfei_super_tts audio Base64 failed: %v", err)
			}

			encoding := strings.ToLower(strings.TrimSpace(audioResp.Encoding))
			if encoding == "" {
				encoding = p.Encoding
			}
			if ensureDecoder != nil {
				if err := ensureDecoder(encoding, audioResp.SampleRate); err != nil {
					return err
				}
			}
			if beforeAudio != nil {
				if err := beforeAudio(audioResp, len(chunk) > 0); err != nil {
					return err
				}
			}

			switch {
			case encoding == "raw":
				if _, err := pipeWriter.Write(chunk); err != nil {
					return fmt.Errorf("Failed to write xunfei_super_tts PCM data: %v", err)
				}
			case encoding == "lame" || encoding == "mp3":
				if _, err := pipeWriter.Write(chunk); err != nil {
					return fmt.Errorf("Failed to write xunfei_super_tts MP3 data: %v", err)
				}
			case strings.HasPrefix(encoding, "opus"):
				frames, err := decodeXunfeiSuperOpusFrames(chunk, p.ExpectedOpusPayloadLen)
				if err != nil {
					return fmt.Errorf("Failed to parse xunfei_super_tts Opus data: %v", err)
				}
				for _, frame := range frames {
					if err := util.WriteLengthPrefixedFrame(pipeWriter, frame); err != nil {
						return fmt.Errorf("Failed to write xunfei_super_tts Opus frame: %v", err)
					}
				}
			default:
				return fmt.Errorf("xunfei_super_tts returned unsupported audio encoding: %s", encoding)
			}
		}

		if audioResp.Status == 1 {
			if resp.Payload.Pybuf != nil && resp.Payload.Pybuf.Text != "" {
				decoded, err := base64.StdEncoding.DecodeString(cleanBase64(resp.Payload.Pybuf.Text))
				if err != nil {
					log.Infof("xunfei_super_tts status=1 pybuf.text(raw): %s", previewString(resp.Payload.Pybuf.Text, 200))
				} else {
					log.Infof("xunfei_super_tts status=1 pybuf.text: %s", previewString(string(decoded), 500))
				}
			}
		}

		if audioResp.Status == 2 {
			return nil
		}
	}
}

func (p *XunfeiSuperTTSProvider) writeRequest(conn *websocket.Conn, payload []byte) error {
	if conn == nil {
		return fmt.Errorf("xunfei_super_tts connection is empty")
	}
	if p.ReadTimeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(time.Duration(p.ReadTimeout) * time.Second))
	}
	if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		return fmt.Errorf("Failed to send xunfei_super_tts request: %v", err)
	}
	return nil
}

func (p *XunfeiSuperTTSProvider) ensureConnection(ctx context.Context) (*websocket.Conn, error) {
	p.connMu.Lock()
	defer p.connMu.Unlock()

	if p.conn != nil {
		return p.conn, nil
	}

	conn, err := p.dial(ctx)
	if err != nil {
		return nil, err
	}
	p.conn = conn
	return conn, nil
}

func (p *XunfeiSuperTTSProvider) reconnect(ctx context.Context) (*websocket.Conn, error) {
	p.connMu.Lock()
	defer p.connMu.Unlock()

	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}

	conn, err := p.dial(ctx)
	if err != nil {
		return nil, err
	}
	p.conn = conn
	return conn, nil
}

func (p *XunfeiSuperTTSProvider) currentConn() *websocket.Conn {
	p.connMu.Lock()
	defer p.connMu.Unlock()
	return p.conn
}

func (p *XunfeiSuperTTSProvider) invalidateConnection(conn *websocket.Conn) {
	p.connMu.Lock()
	defer p.connMu.Unlock()

	if p.conn == nil {
		return
	}
	if conn != nil && p.conn != conn {
		return
	}
	_ = p.conn.Close()
	p.conn = nil
}

func (p *XunfeiSuperTTSProvider) dial(ctx context.Context) (*websocket.Conn, error) {
	signedURL, err := p.buildSignedURL()
	if err != nil {
		return nil, err
	}

	dialer := defaultXunfeiSuperDialer
	if p.ConnectTimeout > 0 {
		dialer.HandshakeTimeout = time.Duration(p.ConnectTimeout) * time.Second
	}

	conn, resp, err := dialer.DialContext(ctx, signedURL, nil)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("Failed to connect to xunfei_super_tts WebSocket, status code: %d, response: %s, err: %v", resp.StatusCode, string(body), err)
		}
		return nil, fmt.Errorf("Failed to connect to xunfei_super_tts WebSocket: %v", err)
	}
	return conn, nil
}

func (p *XunfeiSuperTTSProvider) buildSignedURL() (string, error) {
	parsed, err := url.Parse(p.WSURL)
	if err != nil {
		return "", fmt.Errorf("Invalid xunfei_super_tts ws_url: %v", err)
	}

	host := parsed.Host
	if host == "" {
		return "", fmt.Errorf("xunfei_super_tts ws_url missing host")
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

func (p *XunfeiSuperTTSProvider) validate() error {
	if p == nil {
		return fmt.Errorf("xunfei_super_tts provider cannot be empty")
	}
	if p.AppID == "" {
		return fmt.Errorf("xunfei_super_tts app_id cannot be empty")
	}
	if p.APIKey == "" {
		return fmt.Errorf("xunfei_super_tts api_key cannot be empty")
	}
	if p.APISecret == "" {
		return fmt.Errorf("xunfei_super_tts api_secret cannot be empty")
	}
	if _, _, err := mapXunfeiSuperAudioEncoding(p.AudioEncoding, p.SampleRate); err != nil {
		return err
	}
	return nil
}

// StreamingSynthesize dual-stream synthesis: continuously receiving text from textChan and outputting events while compositing. textChan off indicates the end of text.
func (p *XunfeiSuperTTSProvider) StreamingSynthesize(ctx context.Context, textChan <-chan string, sampleRate int, channels int, frameDuration int) (chan streaming.SynthesisEvent, error) {
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
	outputChan := make(chan streaming.SynthesisEvent, 100)
	startTs := time.Now().UnixMilli()
	go func() {
		if err := p.streamingSynthesisLoop(ctx, textChan, targetSampleRate, targetFrameDuration, startTs, outputChan); err != nil && ctx.Err() == nil {
			log.Errorf("xunfei_super_tts dual-stream synthesis failed: %v", err)
		}
	}()
	return outputChan, nil
}

func (p *XunfeiSuperTTSProvider) streamingSynthesisLoop(ctx context.Context, textChan <-chan string, targetSampleRate int, frameDuration int, startTs int64, outputChan chan streaming.SynthesisEvent) error {
	p.synthesisMu.Lock()
	defer p.synthesisMu.Unlock()

	conn, err := p.reconnect(ctx)
	if err != nil {
		close(outputChan)
		return err
	}
	defer p.invalidateConnection(conn)

	ctxDone := make(chan struct{})
	defer close(ctxDone)
	go func() {
		select {
		case <-ctx.Done():
			p.invalidateConnection(nil)
		case <-ctxDone:
		}
	}()

	pipeReader, pipeWriter := io.Pipe()
	audioFrameChan := make(chan []byte, 100)
	tracker := &xunfeiSuperSentenceTracker{
		activeIdx:       -1,
		frameDurationMs: frameDuration,
	}
	var pendingEventsMu sync.Mutex
	pendingSignals := make([]streaming.SentenceSignal, 0, 8)
	var fallbackModeMu sync.RWMutex
	fallbackMode := false
	queuePendingSignals := func(signals []streaming.SentenceSignal) {
		if len(signals) == 0 {
			return
		}
		pendingEventsMu.Lock()
		pendingSignals = append(pendingSignals, signals...)
		pendingEventsMu.Unlock()
	}
	drainPendingSignals := func() []streaming.SentenceSignal {
		pendingEventsMu.Lock()
		defer pendingEventsMu.Unlock()

		if len(pendingSignals) == 0 {
			return nil
		}

		signals := make([]streaming.SentenceSignal, len(pendingSignals))
		copy(signals, pendingSignals)
		pendingSignals = pendingSignals[:0]
		return signals
	}
	enableFallbackMode := func() {
		fallbackModeMu.Lock()
		fallbackMode = true
		fallbackModeMu.Unlock()
	}
	isFallbackMode := func() bool {
		fallbackModeMu.RLock()
		defer fallbackModeMu.RUnlock()
		return fallbackMode
	}
	emitEvent := func(event streaming.SynthesisEvent) bool {
		select {
		case <-ctx.Done():
			return false
		case outputChan <- event:
			return true
		}
	}
	mergeDone := make(chan struct{})
	go func() {
		defer close(mergeDone)
		defer close(outputChan)

		var buffered *streaming.SynthesisEvent
		flushBuffered := func() bool {
			if buffered == nil {
				return true
			}
			if !emitEvent(*buffered) {
				return false
			}
			buffered = nil
			return true
		}

		for frame := range audioFrameChan {
			if buffered != nil && !flushBuffered() {
				return
			}

			signals := drainPendingSignals()
			if isFallbackMode() {
				signals = append(signals, tracker.SignalsForFallbackFrame()...)
			}

			frameCopy := make([]byte, len(frame))
			copy(frameCopy, frame)
			buffered = &streaming.SynthesisEvent{
				Audio:           frameCopy,
				SentenceSignals: signals,
			}
		}

		tailSignals := drainPendingSignals()
		if len(tailSignals) > 0 {
			if buffered == nil {
				buffered = &streaming.SynthesisEvent{}
			}
			buffered.SentenceSignals = append(buffered.SentenceSignals, tailSignals...)
		}
		if !flushBuffered() {
			return
		}
	}()
	var (
		decoderStarted    bool
		decoderAudioFmt   string
		decoderSampleRate int
		decoderDone       chan struct{}
	)

	ensureDecoder := func(sourceEncoding string, sourceSampleRate int) error {
		audioFormat, err := mapXunfeiSuperResponseAudioFormat(sourceEncoding)
		if err != nil {
			return err
		}
		if sourceSampleRate <= 0 {
			sourceSampleRate = p.SampleRate
		}
		if decoderStarted {
			if audioFormat != decoderAudioFmt {
				return fmt.Errorf("xunfei_super_tts returns audio encoding changes: %s -> %s", decoderAudioFmt, audioFormat)
			}
			if audioFormat != "mp3" && sourceSampleRate != decoderSampleRate {
				return fmt.Errorf("xunfei_super_tts returns a change in sampling rate: %d -> %d", decoderSampleRate, sourceSampleRate)
			}
			return nil
		}
		decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, audioFrameChan, frameDuration, audioFormat, targetSampleRate)
		if err != nil {
			return fmt.Errorf("Failed to create xunfei_super_tts audio decoder: %v", err)
		}
		if audioFormat != "mp3" {
			decoder.WithFormat(beep.Format{
				SampleRate:  beep.SampleRate(sourceSampleRate),
				NumChannels: 1,
			})
		}
		decoderStarted = true
		decoderAudioFmt = audioFormat
		decoderSampleRate = sourceSampleRate
		decoderDone = make(chan struct{})
		go func() {
			defer close(decoderDone)
			if err := decoder.Run(startTs); err != nil && ctx.Err() == nil {
				log.Errorf("xunfei_super_tts Dual-stream audio decoding failed: %v", err)
			}
		}()
		return nil
	}

	finishDecoder := func(closeErr error) {
		if closeErr != nil {
			_ = pipeWriter.CloseWithError(closeErr)
		} else {
			_ = pipeWriter.Close()
		}
		if decoderStarted {
			<-decoderDone
			return
		}
		_ = pipeReader.Close()
		close(audioFrameChan)
	}

	//Wait for the first non-empty text
	var firstText string
	for {
		select {
		case <-ctx.Done():
			finishDecoder(ctx.Err())
			return ctx.Err()
		case text, ok := <-textChan:
			if !ok {
				finishDecoder(nil)
				return nil
			}
			if t := strings.TrimSpace(text); t != "" {
				firstText = t
				goto gotFirstText
			}
		}
	}
gotFirstText:

	//Dual-stream request status is advanced independently according to the protocol:
	//The first non-empty text must use status=0, subsequent text must use status=1, and end with status=2 when input is closed.
	sendErrCh := make(chan error, 1)
	go func() {
		seq := 0
		fail := func(err error) {
			p.invalidateConnection(conn)
			sendErrCh <- err
		}
		writeRequest := func(text string, status int) error {
			payload, err := json.Marshal(p.buildSingleSynthesisRequest(text, seq, status))
			if err != nil {
				return err
			}
			if err := p.writeRequest(conn, payload); err != nil {
				return err
			}
			seq++
			return nil
		}

		if err := tracker.Append(firstText); err != nil {
			fail(err)
			return
		}
		if err := writeRequest(firstText, 0); err != nil {
			fail(err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				sendErrCh <- ctx.Err()
				return
			case text, ok := <-textChan:
				if !ok {
					if err := writeRequest("", 2); err != nil {
						fail(err)
						return
					}
					sendErrCh <- nil
					return
				}
				text = strings.TrimSpace(text)
				if text == "" {
					continue
				}
				if err := tracker.Append(text); err != nil {
					fail(err)
					return
				}
				if err := writeRequest(text, 1); err != nil {
					fail(err)
					return
				}
			}
		}
	}()

	fallbackLogged := false
	streamErr := p.readSynthesisResponse(ctx, pipeWriter, ensureDecoder, func(audioResp *xunfeiSuperAudioResp, hasAudio bool) error {
		if !hasAudio {
			return nil
		}

		progress, ok := parseXunfeiSuperCed(audioResp.Ced)
		if ok {
			queuePendingSignals(tracker.SignalsForProgress(progress))
			return nil
		}

		if !fallbackLogged {
			log.Warnf("xunfei_super_tts dual-stream response does not return ced, falls back to audio block-level sentence boundary estimation")
			fallbackLogged = true
		}
		enableFallbackMode()
		return nil
	})
	queuePendingSignals(tracker.FinalizeSignals())
	finishDecoder(streamErr)
	<-mergeDone

	sendErr := <-sendErrCh
	if streamErr == nil {
		streamErr = sendErr
	}
	if streamErr == nil && ctx.Err() == nil {
		log.Infof("xunfei_super_tts Dual stream time consumption: %d ms", time.Now().UnixMilli()-startTs)
	}
	return streamErr
}

func (p *XunfeiSuperTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && strings.TrimSpace(voice) != "" {
		p.Voice = strings.TrimSpace(voice)
		return nil
	}
	return fmt.Errorf("Invalid voice configuration: missing voice")
}

func (p *XunfeiSuperTTSProvider) Close() error {
	p.invalidateConnection(nil)
	return nil
}

func (p *XunfeiSuperTTSProvider) IsValid() bool {
	return p != nil
}

func mapXunfeiSuperResponseAudioFormat(encoding string) (string, error) {
	switch normalized := strings.ToLower(strings.TrimSpace(encoding)); {
	case normalized == "raw":
		return "pcm", nil
	case normalized == "lame" || normalized == "mp3":
		return "mp3", nil
	case strings.HasPrefix(normalized, "opus"):
		return "opus", nil
	default:
		return "", fmt.Errorf("Unsupported xunfei_super_tts response audio encoding: %s", encoding)
	}
}

func mapXunfeiSuperAudioEncoding(audioEncoding string, sampleRate int) (string, int, error) {
	switch strings.ToLower(strings.TrimSpace(audioEncoding)) {
	case "", "raw":
		if sampleRate != 8000 && sampleRate != 16000 && sampleRate != 24000 {
			return "", 0, fmt.Errorf("xunfei_super_tts raw only supports 8000/16000/24000 sampling rate, currently: %d", sampleRate)
		}
		return "raw", 0, nil
	case "opus":
		switch sampleRate {
		case 8000:
			return "opus", 20, nil
		case 16000:
			return "opus-wb", 40, nil
		case 24000:
			//The documentation lists the opus-swb encoding name but does not explicitly respond to the payload size, and the length is not enforced here.
			return "opus-swb", 0, nil
		default:
			return "", 0, fmt.Errorf("xunfei_super_tts opus only supports 8000/16000/24000 sampling rate, currently: %d", sampleRate)
		}
	default:
		return "", 0, fmt.Errorf("Unsupported xunfei_super_tts audio_encoding: %s", audioEncoding)
	}
}

func decodeXunfeiSuperOpusFrames(chunk []byte, expected int) ([][]byte, error) {
	if len(chunk) == 0 {
		return nil, nil
	}

	if expected > 0 && len(chunk) == expected {
		frame := make([]byte, len(chunk))
		copy(frame, chunk)
		return [][]byte{frame}, nil
	}

	if frames, ok := tryParseLengthPrefixedFrames(chunk, expected); ok {
		return frames, nil
	}

	frame := make([]byte, len(chunk))
	copy(frame, chunk)
	return [][]byte{frame}, nil
}

func tryParseLengthPrefixedFrames(chunk []byte, expected int) ([][]byte, bool) {
	if len(chunk) < 4 {
		return nil, false
	}

	knownLengths := map[int]struct{}{20: {}, 40: {}, 60: {}}
	frames := make([][]byte, 0, 4)
	offset := 0
	for offset < len(chunk) {
		if len(chunk)-offset < 2 {
			return nil, false
		}

		header := chunk[offset : offset+2]
		remaining := len(chunk) - offset - 2
		candidate, ok := selectPayloadLength(header, remaining, expected)
		if !ok {
			if expected == 0 {
				if _, known := knownLengths[int(binary.LittleEndian.Uint16(header))]; !known {
					if _, known = knownLengths[int(binary.BigEndian.Uint16(header))]; !known {
						return nil, false
					}
				}
				candidate, ok = selectPayloadLength(header, remaining, 0)
				if !ok {
					return nil, false
				}
			} else {
				return nil, false
			}
		}

		start := offset + 2
		end := start + candidate
		if candidate <= 0 || end > len(chunk) {
			return nil, false
		}

		frame := make([]byte, candidate)
		copy(frame, chunk[start:end])
		frames = append(frames, frame)
		offset = end
	}

	if len(frames) == 0 || offset != len(chunk) {
		return nil, false
	}
	return frames, true
}

func selectPayloadLength(header []byte, remaining int, expected int) (int, bool) {
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

func estimateXunfeiSuperSentenceMinFrames(text string, frameDurationMs int) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 1
	}

	if frameDurationMs <= 0 {
		frameDurationMs = defaultXunfeiSuperFrameDuration
	}

	durationMs := 0
	for _, r := range trimmed {
		switch {
		case unicode.IsSpace(r):
			continue
		case isXunfeiSuperPausePunctuation(r):
			durationMs += 180
		case r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			durationMs += 70
		default:
			durationMs += 110
		}
	}

	if durationMs < 320 {
		durationMs = 320
	}
	if durationMs > 2600 {
		durationMs = 2600
	}

	frames := (durationMs + frameDurationMs - 1) / frameDurationMs
	if frames < 1 {
		return 1
	}
	return frames
}

func isXunfeiSuperPausePunctuation(r rune) bool {
	switch r {
	case ',', '.', '!', '?', ';', ':', '，', '。', '！', '？', '；', '：', '、', '…':
		return true
	default:
		return false
	}
}

func (t *xunfeiSuperSentenceTracker) Append(text string) error {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return nil
	}

	textBytes := len([]byte(normalized))
	if textBytes == 0 {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.totalBytes+textBytes > maxXunfeiSuperTextBytes {
		return fmt.Errorf("xunfei_super_tts dual stream text exceeds 64KB limit, currently: %d bytes", t.totalBytes+textBytes)
	}

	startByte := t.totalBytes
	t.totalBytes += textBytes
	t.spans = append(t.spans, &xunfeiSuperSentenceSpan{
		Text:      normalized,
		StartByte: startByte,
		EndByte:   t.totalBytes,
		MinFrames: estimateXunfeiSuperSentenceMinFrames(normalized, t.frameDurationMs),
	})
	return nil
}

func (t *xunfeiSuperSentenceTracker) SignalsForProgress(progress int) []streaming.SentenceSignal {
	if progress <= 0 {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	signals := make([]streaming.SentenceSignal, 0, 4)
	for idx, span := range t.spans {
		if span.Started || progress <= span.StartByte {
			continue
		}

		for prevIdx := 0; prevIdx < idx; prevIdx++ {
			prev := t.spans[prevIdx]
			if !prev.Started || prev.Ended {
				continue
			}
			prev.Ended = true
			signals = append(signals, streaming.SentenceSignal{
				Type: streaming.SentenceSignalEnd,
				Text: prev.Text,
			})
		}

		span.Started = true
		t.activeIdx = idx
		t.activeFrames = 0
		signals = append(signals, streaming.SentenceSignal{
			Type: streaming.SentenceSignalStart,
			Text: span.Text,
		})
	}

	return signals
}

func (t *xunfeiSuperSentenceTracker) SignalsForFallbackFrame() []streaming.SentenceSignal {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.spans) == 0 {
		return nil
	}

	if t.activeIdx < 0 {
		first := t.spans[0]
		first.Started = true
		t.activeIdx = 0
		t.activeFrames = 1
		return []streaming.SentenceSignal{{
			Type: streaming.SentenceSignalStart,
			Text: first.Text,
		}}
	}

	current := t.spans[t.activeIdx]
	nextIdx := t.activeIdx + 1
	if nextIdx >= len(t.spans) {
		t.activeFrames++
		return nil
	}

	if t.activeFrames < current.MinFrames {
		t.activeFrames++
		return nil
	}

	next := t.spans[nextIdx]
	if current.Ended || next.Started {
		t.activeFrames++
		return nil
	}

	current.Ended = true
	next.Started = true
	t.activeIdx = nextIdx
	t.activeFrames = 1
	return []streaming.SentenceSignal{
		{
			Type: streaming.SentenceSignalEnd,
			Text: current.Text,
		},
		{
			Type: streaming.SentenceSignalStart,
			Text: next.Text,
		},
	}
}

func (t *xunfeiSuperSentenceTracker) FinalizeSignals() []streaming.SentenceSignal {
	t.mu.Lock()
	defer t.mu.Unlock()

	signals := make([]streaming.SentenceSignal, 0, len(t.spans)*2)
	for idx, span := range t.spans {
		if !span.Started {
			span.Started = true
			signals = append(signals, streaming.SentenceSignal{
				Type: streaming.SentenceSignalStart,
				Text: span.Text,
			})
		}
		if span.Ended {
			continue
		}
		span.Ended = true
		signals = append(signals, streaming.SentenceSignal{
			Type: streaming.SentenceSignalEnd,
			Text: span.Text,
		})
		if t.activeIdx < idx {
			t.activeIdx = idx
		}
	}
	return signals
}

func parseXunfeiSuperCed(raw string) (int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}

	if value, err := strconv.Atoi(trimmed); err == nil {
		return value, true
	}

	start := -1
	for idx, r := range trimmed {
		if r >= '0' && r <= '9' {
			start = idx
			break
		}
	}
	if start < 0 {
		return 0, false
	}

	end := start
	for end < len(trimmed) {
		ch := trimmed[end]
		if ch < '0' || ch > '9' {
			break
		}
		end++
	}
	if end <= start {
		return 0, false
	}

	value, err := strconv.Atoi(trimmed[start:end])
	if err != nil {
		return 0, false
	}
	return value, true
}
