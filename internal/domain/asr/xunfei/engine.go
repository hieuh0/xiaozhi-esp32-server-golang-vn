package xunfei

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	asrtypes "xiaozhi-esp32-server-golang/internal/domain/asr/types"
	log "xiaozhi-esp32-server-golang/logger"
)

// ASR Xunfei streaming dictation implementation
// Create an independent connection for each identification to avoid concurrency contention.
type ASR struct {
	config Config
}

func classifyXunfeiRetryReason(code int, message string) string {
	if code == 10008 || strings.Contains(strings.ToLower(message), "service instance invalid") {
		return asrtypes.RetryReasonXunfeiServiceInstanceInvalid
	}
	return asrtypes.RetryReasonNone
}

func New(cfg Config) (*ASR, error) {
	finalCfg := defaultConfig()
	if cfg.AppID != "" {
		finalCfg.AppID = cfg.AppID
	}
	if cfg.APIKey != "" {
		finalCfg.APIKey = cfg.APIKey
	}
	if cfg.APISecret != "" {
		finalCfg.APISecret = cfg.APISecret
	}
	if cfg.Host != "" {
		finalCfg.Host = cfg.Host
	}
	if cfg.Path != "" {
		finalCfg.Path = cfg.Path
	}
	if cfg.Language != "" {
		finalCfg.Language = cfg.Language
	}
	if cfg.Accent != "" {
		finalCfg.Accent = cfg.Accent
	}
	if cfg.Domain != "" {
		finalCfg.Domain = cfg.Domain
	}
	if cfg.SampleRate > 0 {
		finalCfg.SampleRate = cfg.SampleRate
	}
	if cfg.Timeout > 0 {
		finalCfg.Timeout = cfg.Timeout
	}

	if finalCfg.AppID == "" || finalCfg.APIKey == "" || finalCfg.APISecret == "" {
		return nil, fmt.Errorf("xunfei asr missing required credentials: appid/api_key/api_secret")
	}

	return &ASR{config: finalCfg}, nil
}

func (a *ASR) StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan asrtypes.StreamingResult, error) {
	resultChan := make(chan asrtypes.StreamingResult, 20)

	wsURL, err := a.buildWebSocketURL()
	if err != nil {
		return nil, err
	}

	dialer := websocket.DefaultDialer
	h := http.Header{}
	conn, _, err := dialer.DialContext(ctx, wsURL, h)
	if err != nil {
		return nil, fmt.Errorf("xunfei dial failed: %w", err)
	}

	go a.handleStreaming(ctx, conn, audioStream, resultChan)
	return resultChan, nil
}

func (a *ASR) Process(pcmData []float32) (string, error) {
	ctx := context.Background()
	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	audioStream := make(chan []float32, 1)
	go func() {
		audioStream <- pcmData
		close(audioStream)
	}()

	resultChan, err := a.StreamingRecognize(ctx, audioStream)
	if err != nil {
		return "", err
	}

	var finalText string
	for result := range resultChan {
		if result.Error != nil {
			return "", result.Error
		}
		if result.Text != "" {
			finalText = result.Text
		}
		if result.IsFinal {
			return finalText, nil
		}
	}

	if finalText != "" {
		return finalText, nil
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return "", fmt.Errorf("no xunfei asr result")
}

func (a *ASR) Close() error  { return nil }
func (a *ASR) IsValid() bool { return a != nil }

func (a *ASR) handleStreaming(ctx context.Context, conn *websocket.Conn, audioStream <-chan []float32, resultChan chan asrtypes.StreamingResult) {
	defer close(resultChan)
	defer conn.Close()

	go a.sendAudio(ctx, conn, audioStream)

	var resultBuilder strings.Builder
	recvCount := 0
	for {
		select {
		case <-ctx.Done():
			resultChan <- asrtypes.StreamingResult{Error: ctx.Err()}
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				finalText := resultBuilder.String()
				emptyReason := asrtypes.EmptyReasonNone
				if finalText == "" {
					if recvCount == 0 {
						emptyReason = asrtypes.EmptyReasonNoServerResponse
					} else {
						emptyReason = asrtypes.EmptyReasonProviderEmptyFinal
					}
				}
				resultChan <- asrtypes.StreamingResult{Text: finalText, IsFinal: true, EmptyReason: emptyReason}
				return
			}
			resultChan <- asrtypes.StreamingResult{Error: fmt.Errorf("xunfei read message failed: %w", err), IsFinal: true}
			return
		}
		recvCount++

		var rsp response
		if err := json.Unmarshal(msg, &rsp); err != nil {
			resultChan <- asrtypes.StreamingResult{Error: fmt.Errorf("xunfei decode response failed: %w", err), IsFinal: true}
			return
		}
		if rsp.Code != 0 {
			resultChan <- asrtypes.StreamingResult{
				Error:       fmt.Errorf("xunfei asr error code=%d message=%s sid=%s", rsp.Code, rsp.Message, rsp.SID),
				IsFinal:     true,
				RetryReason: classifyXunfeiRetryReason(rsp.Code, rsp.Message),
			}
			return
		}

		text := rsp.extractText()
		if text == "" {
			log.Debugf(
				"xunfei asr recv[%d]: sid=%s status=%d sn=%d ls=%v pgs=%q rg=%v ws=%d cw=%d text_len=%d raw=%s",
				recvCount,
				rsp.SID,
				rsp.Data.Status,
				rsp.Data.Result.Sn,
				rsp.Data.Result.Ls,
				rsp.Data.Result.Pgs,
				rsp.Data.Result.Rg,
				len(rsp.Data.Result.Ws),
				rsp.candidateCount(),
				len(text),
				truncateForLog(string(msg), 512),
			)
		}
		if text != "" {
			resultBuilder.WriteString(text)
			resultChan <- asrtypes.StreamingResult{Text: resultBuilder.String(), IsFinal: false}
		}

		if rsp.Data.Status == 2 {
			finalText := resultBuilder.String()
			emptyReason := asrtypes.EmptyReasonNone
			if finalText == "" {
				emptyReason = asrtypes.EmptyReasonProviderEmptyFinal
			}
			if finalText == "" {
				log.Debugf(
					"xunfei asr final result empty: sid=%s recv_count=%d ws=%d cw=%d sn=%d ls=%v pgs=%q rg=%v raw=%s",
					rsp.SID,
					recvCount,
					len(rsp.Data.Result.Ws),
					rsp.candidateCount(),
					rsp.Data.Result.Sn,
					rsp.Data.Result.Ls,
					rsp.Data.Result.Pgs,
					rsp.Data.Result.Rg,
					truncateForLog(string(msg), 512),
				)
			}
			resultChan <- asrtypes.StreamingResult{Text: finalText, IsFinal: true, EmptyReason: emptyReason}
			return
		}
	}
}

func (a *ASR) sendAudio(ctx context.Context, conn *websocket.Conn, audioStream <-chan []float32) {
	status := 0
	for {
		select {
		case <-ctx.Done():
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
			return
		case pcm, ok := <-audioStream:
			if !ok {
				if err := a.sendFrame(conn, nil, 2); err != nil {
					return
				}
				return
			}
			audioBytes := make([]byte, len(pcm)*2)
			float32ToPCMBytes(pcm, audioBytes)
			if err := a.sendFrame(conn, audioBytes, status); err != nil {
				return
			}
			status = 1
		}
	}
}

func (a *ASR) sendFrame(conn *websocket.Conn, audioBytes []byte, status int) error {
	req := request{Data: data{Status: status, Format: fmt.Sprintf("audio/L16;rate=%d", a.config.SampleRate), Encoding: "raw", Audio: base64.StdEncoding.EncodeToString(audioBytes)}}
	if status == 0 {
		req.Common.AppID = a.config.AppID
		req.Business = business{Language: a.config.Language, Domain: a.config.Domain, Accent: a.config.Accent}
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}

func (a *ASR) buildWebSocketURL() (string, error) {
	host := a.config.Host
	date := time.Now().UTC().Format(time.RFC1123)
	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\nGET %s HTTP/1.1", host, date, a.config.Path)

	h := hmac.New(sha256.New, []byte(a.config.APISecret))
	if _, err := h.Write([]byte(signatureOrigin)); err != nil {
		return "", err
	}
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	authorizationOrigin := fmt.Sprintf(`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`, a.config.APIKey, signature)
	authorization := base64.StdEncoding.EncodeToString([]byte(authorizationOrigin))

	v := url.Values{}
	v.Set("authorization", authorization)
	v.Set("date", date)
	v.Set("host", host)

	return fmt.Sprintf("wss://%s%s?%s", host, a.config.Path, v.Encode()), nil
}

func float32ToPCMBytes(samples []float32, pcmBytes []byte) {
	for i, sample := range samples {
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		v := int16(sample * 32767)
		pcmBytes[i*2] = byte(v)
		pcmBytes[i*2+1] = byte(v >> 8)
	}
}

func truncateForLog(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}
