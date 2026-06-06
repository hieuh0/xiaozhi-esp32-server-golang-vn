package doubao

import (
	"context"
	"fmt"
	"strings"
	"time"

	"xiaozhi-esp32-server-golang/internal/domain/asr/doubao/client"
	"xiaozhi-esp32-server-golang/internal/domain/asr/doubao/request"
	"xiaozhi-esp32-server-golang/internal/domain/asr/doubao/response"
	"xiaozhi-esp32-server-golang/internal/domain/asr/types"
	log "xiaozhi-esp32-server-golang/logger"
)

func shortDebugID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:6] + "..." + id[len(id)-6:]
}

func previewDoubaoText(text string, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 32
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "..."
}

func firstNonEmptyUtteranceText(payload *response.AsrResponsePayload) string {
	if payload == nil {
		return ""
	}
	for _, utterance := range payload.Result.Utterances {
		if utterance.Text != "" {
			return utterance.Text
		}
	}
	return ""
}

func classifyEmptyFinalReason(packetCount, nonEmptyPacketCount int, result *response.AsrResponse, audioDuration int) string {
	if result != nil && packetCount == 1 && nonEmptyPacketCount == 0 && result.PayloadSequence == 0 && result.Event == 0 && audioDuration == 0 {
		return types.EmptyReasonNoServerResponse
	}
	return types.EmptyReasonProviderEmptyFinal
}

func classifyDoubaoRetryReason(errMsg string, code int) string {
	errTextLower := strings.ToLower(errMsg)
	if strings.Contains(errTextLower, "waiting next packet timeout") &&
		strings.Contains(errTextLower, "session has ended") {
		return types.RetryReasonDoubaoWaitingNextPacketTimeout
	}
	if code == 45000081 || strings.Contains(errMsg, "45000081") {
		return types.RetryReasonDoubaoResponseCode45000081
	}
	return types.RetryReasonNone
}

// DoubaoV2ASR DoubaoASR implementation
type DoubaoV2ASR struct {
	config      DoubaoV2Config
	isStreaming bool
	reqID       string
	connectID   string

	//Streaming identification related fields
	result      string
	err         error
	sendDataCnt int
	c           *client.AsrWsClient
}

// NewDoubaoV2ASR Create a new DoubaoASR instance
func NewDoubaoV2ASR(config DoubaoV2Config) (*DoubaoV2ASR, error) {
	log.Info("Create DoubaoASR instance")
	log.Info(fmt.Sprintf("Configuration: %+v", config))

	if config.AppID == "" {
		log.Error("Missing appid configuration")
		return nil, fmt.Errorf("Missing appid configuration")
	}
	if config.AccessToken == "" {
		log.Error("Missing access_token configuration")
		return nil, fmt.Errorf("Missing access_token configuration")
	}

	//Fill in missing fields using default configuration
	if config.WsURL == "" {
		config.WsURL = DefaultConfig.WsURL
	}
	originalWsURL := config.WsURL
	config.WsURL = normalizeDoubaoWsURL(config.WsURL)
	if originalWsURL != config.WsURL {
		log.Warnf("DoubaoASR ws_url uses a non-streaming address and has automatically switched to a streaming address: %s -> %s", originalWsURL, config.WsURL)
	}
	if config.ResourceID == "" {
		config.ResourceID = DefaultConfig.ResourceID
	}
	if config.ModelName == "" {
		config.ModelName = DefaultConfig.ModelName
	}
	if config.EndWindowSize == 0 {
		config.EndWindowSize = DefaultConfig.EndWindowSize
	}
	if config.ResultType == "" {
		config.ResultType = DefaultConfig.ResultType
	}
	if config.ForceToSpeechTime == 0 {
		config.ForceToSpeechTime = DefaultConfig.ForceToSpeechTime
	}
	if config.ChunkDuration == 0 {
		config.ChunkDuration = DefaultConfig.ChunkDuration
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultConfig.Timeout
	}

	connectID := fmt.Sprintf("%d", time.Now().UnixNano())

	return &DoubaoV2ASR{
		config:    config,
		connectID: connectID,
	}, nil
}

// StreamingRecognize implements the streaming recognition interface
// Note: The connection will be delayed when the first audio packet is received to avoid server timeout due to VAD delay.
func (d *DoubaoV2ASR) StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan types.StreamingResult, error) {
	connectID := fmt.Sprintf("%s-%d", d.connectID, time.Now().UnixMilli())
	streamID := shortDebugID(connectID)
	requestOptions := request.FullClientRequestOptions{
		Uid:               connectID,
		ModelName:         d.config.ModelName,
		EnableITN:         d.config.EnableITN,
		EnablePUNC:        d.config.EnablePunc,
		EnableDDC:         d.config.EnableDDC,
		EndWindowSize:     d.config.EndWindowSize,
		ResultType:        d.config.ResultType,
		ShowUtterances:    d.config.ShowUtterances,
		ForceToSpeechTime: d.config.ForceToSpeechTime,
		EnableNonstream:   d.config.EnableNonstream,
	}
	//Create client instance (no connection established immediately)
	d.c = client.NewAsrWsClient(d.config.WsURL, d.config.AppID, d.config.AccessToken, d.config.ResourceID, connectID, streamID, requestOptions)
	log.Debugf(
		"[doubao-asr:%s] StreamingRecognize start: ws=%s, resource_id=%s, result_type=%s, show_utterances=%v, force_to_speech_time=%d, chunk_duration=%d, timeout=%d",
		streamID,
		d.config.WsURL,
		d.config.ResourceID,
		d.config.ResultType,
		d.config.ShowUtterances,
		d.config.ForceToSpeechTime,
		d.config.ChunkDuration,
		d.config.Timeout,
	)

	//Recognition results returned by Doubao
	doubaoResultChan := make(chan *response.AsrResponse, 10)
	//Result channel inside program
	resultChan := make(chan types.StreamingResult, 10)

	//Start audio streaming (the connection will be established when the first audio packet arrives)
	go func() {
		defer close(doubaoResultChan)
		if err := d.c.StartAudioStream(ctx, audioStream, doubaoResultChan); err != nil {
			log.Warnf("[doubao-asr:%s] StartAudioStream returned error: %v", streamID, err)
			payload := &response.AsrResponsePayload{}
			payload.Error = err.Error()
			select {
			case <-ctx.Done():
			case doubaoResultChan <- &response.AsrResponse{
				Code:          -1,
				IsLastPackage: true,
				PayloadMsg:    payload,
			}:
			}
		}
	}()

	//Start result receiving goroutine
	go d.receiveStreamResults(ctx, streamID, resultChan, doubaoResultChan)

	return resultChan, nil
}

// receiveStreamResults receives streaming recognition results
func (d *DoubaoV2ASR) receiveStreamResults(ctx context.Context, streamID string, resultChan chan types.StreamingResult, asrResponseChan chan *response.AsrResponse) {
	packetCount := 0
	nonEmptyPacketCount := 0
	lastNonEmptyText := ""
	lastNonEmptyUtterance := ""
	lastPartialText := ""
	defer func() {
		close(resultChan)
		if d.c != nil {
			d.c.Close()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			log.Debugf("[doubao-asr:%s] receiveStreamResults context canceled", streamID)
			return
		case result, ok := <-asrResponseChan:
			if !ok {
				log.Debugf(
					"[doubao-asr:%s] receiveStreamResults asrResponseChan closed: packets=%d, non_empty_packets=%d, last_non_empty=%q, last_non_empty_utterance=%q",
					streamID,
					packetCount,
					nonEmptyPacketCount,
					previewDoubaoText(lastNonEmptyText, 24),
					previewDoubaoText(lastNonEmptyUtterance, 24),
				)
				return
			}
			packetCount++

			text := ""
			textLen := 0
			utteranceCount := 0
			firstUtterance := ""
			audioDuration := 0
			candidateText := ""
			if result.PayloadMsg != nil {
				text = result.PayloadMsg.Result.Text
				textLen = len([]rune(text))
				utteranceCount = len(result.PayloadMsg.Result.Utterances)
				firstUtterance = firstNonEmptyUtteranceText(result.PayloadMsg)
				audioDuration = result.PayloadMsg.AudioInfo.Duration
			}
			candidateText = text
			if candidateText == "" {
				candidateText = firstUtterance
			}
			if candidateText != "" {
				nonEmptyPacketCount++
				lastNonEmptyText = candidateText
			}
			if firstUtterance != "" {
				lastNonEmptyUtterance = firstUtterance
			}
			log.Debugf(
				"[doubao-asr:%s] Summary of upstream results: idx=%d, payload_seq=%d, event=%d, last=%v, code=%d, text_len=%d, text=%q, utterances=%d, first_utterance=%q, audio_duration=%d",
				streamID,
				packetCount,
				result.PayloadSequence,
				result.Event,
				result.IsLastPackage,
				result.Code,
				textLen,
				previewDoubaoText(text, 24),
				utteranceCount,
				previewDoubaoText(firstUtterance, 24),
				audioDuration,
			)
			if result.Code != 0 {
				errMsg := fmt.Sprintf("asr response code: %d", result.Code)
				if result.PayloadMsg != nil && result.PayloadMsg.Error != "" {
					errMsg = result.PayloadMsg.Error
				}
				retryReason := classifyDoubaoRetryReason(errMsg, result.Code)
				log.Warnf(
					"[doubao-asr:%s] Received error results: packets=%d, non_empty_packets=%d, last_non_empty=%q, last_non_empty_utterance=%q, err=%s, retry_reason=%s",
					streamID,
					packetCount,
					nonEmptyPacketCount,
					previewDoubaoText(lastNonEmptyText, 24),
					previewDoubaoText(lastNonEmptyUtterance, 24),
					errMsg,
					retryReason,
				)
				//Use select to avoid sending to a closed channel (if ctx has been cancelled, prefer ctx.Done())
				select {
				case <-ctx.Done():
					log.Debugf("[doubao-asr:%s] The context has been canceled when sending the wrong result, skipping the sending", streamID)
					return
				case resultChan <- types.StreamingResult{
					Text:        "",
					IsFinal:     true,
					Error:       fmt.Errorf("%s", errMsg),
					RetryReason: retryReason,
				}:
				}
				return
			}
			if !result.IsLastPackage && candidateText != "" {
				if candidateText != lastPartialText {
					lastPartialText = candidateText
					log.Debugf(
						"[doubao-asr:%s] Transparent transmission intermediate results: packets=%d, partial_text=%q, payload_seq=%d, event=%d",
						streamID,
						packetCount,
						previewDoubaoText(candidateText, 24),
						result.PayloadSequence,
						result.Event,
					)
					select {
					case <-ctx.Done():
						log.Debugf("[doubao-asr:%s] The context has been canceled when sending intermediate results, skip sending", streamID)
						return
					case resultChan <- types.StreamingResult{
						Text:    candidateText,
						IsFinal: false,
					}:
					}
				}
			}
			if result.IsLastPackage {
				finalText := text
				if finalText == "" {
					if lastNonEmptyText != "" {
						finalText = lastNonEmptyText
					} else if firstUtterance != "" {
						finalText = firstUtterance
					} else if lastNonEmptyUtterance != "" {
						finalText = lastNonEmptyUtterance
					}
				}
				if finalText == "" {
					emptyReason := classifyEmptyFinalReason(packetCount, nonEmptyPacketCount, result, audioDuration)
					log.Warnf(
						"[doubao-asr:%s] The final packet text is empty: packets=%d, non_empty_packets=%d, last_non_empty=%q, last_non_empty_utterance=%q, payload_seq=%d, event=%d, utterances=%d, audio_duration=%d, empty_reason=%s",
						streamID,
						packetCount,
						nonEmptyPacketCount,
						previewDoubaoText(lastNonEmptyText, 24),
						previewDoubaoText(lastNonEmptyUtterance, 24),
						result.PayloadSequence,
						result.Event,
						utteranceCount,
						audioDuration,
						emptyReason,
					)
					select {
					case <-ctx.Done():
						log.Debugf("[doubao-asr:%s] The context has been canceled when sending the final empty result, skipping sending", streamID)
						return
					case resultChan <- types.StreamingResult{
						Text:        "",
						IsFinal:     true,
						EmptyReason: emptyReason,
					}:
					}
					return
				}
				if text == "" {
					log.Warnf(
						"[doubao-asr:%s] The final packet text is empty, fallback to the latest non-empty result: packets=%d, non_empty_packets=%d, final_text=%q, last_non_empty_utterance=%q, payload_seq=%d, event=%d, utterances=%d, audio_duration=%d",
						streamID,
						packetCount,
						nonEmptyPacketCount,
						previewDoubaoText(finalText, 24),
						previewDoubaoText(lastNonEmptyUtterance, 24),
						result.PayloadSequence,
						result.Event,
						utteranceCount,
						audioDuration,
					)
				} else {
					log.Debugf(
						"[doubao-asr:%s] Final packet text: packets=%d, non_empty_packets=%d, final_text=%q",
						streamID,
						packetCount,
						nonEmptyPacketCount,
						previewDoubaoText(text, 24),
					)
				}
				//Handle final results (including empty results in silent cases), use select to avoid sending to closed channels
				select {
				case <-ctx.Done():
					log.Debugf("[doubao-asr:%s] The context was canceled when sending the final result, skipping sending", streamID)
					return
				case resultChan <- types.StreamingResult{
					Text:    finalText,
					IsFinal: true,
				}:
				}
				return
			}
		}
	}
}

// Reset reset ASR status
func (d *DoubaoV2ASR) Reset() error {

	log.Info("ASR status has been reset")
	return nil
}

// Close closes resources, releases connections, etc.
func (d *DoubaoV2ASR) Close() error {
	if d.c != nil {
		return d.c.Close()
	}
	return nil
}

// IsValid checks whether the resource is valid
func (d *DoubaoV2ASR) IsValid() bool {
	return d != nil
}
