package controllers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"xiaozhi/manager/backend/models"
)

const (
	defaultDoubaoCloneUploadEndpoint = "https://openspeech.bytedance.com/api/v1/mega_tts/audio/upload"
	defaultDoubaoCloneStatusEndpoint = "https://openspeech.bytedance.com/api/v1/mega_tts/status"
	defaultDoubaoPreviewEndpoint     = "https://openspeech.bytedance.com/api/v3/tts/unidirectional"
)

type doubaoCloneBaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type doubaoCloneUploadResponse struct {
	BaseResp     doubaoCloneBaseResp `json:"base_resp"`
	SpeakerID    string              `json:"speaker_id"`
	ICLSpeakerID string              `json:"icl_speaker_id"`
}

type doubaoCloneStatusResponse struct {
	BaseResp     doubaoCloneBaseResp `json:"base_resp"`
	SpeakerID    string              `json:"speaker_id"`
	ICLSpeakerID string              `json:"icl_speaker_id"`
	TrainStatus  string              `json:"train_status"`
	Status       string              `json:"status"`
	DemoAudio    string              `json:"demo_audio"`
}

type doubaoPreviewRequest struct {
	User struct {
		UID string `json:"uid"`
	} `json:"user"`
	ReqParams struct {
		Text        string `json:"text"`
		Speaker     string `json:"speaker"`
		AudioParams struct {
			Format     string `json:"format"`
			SampleRate int    `json:"sample_rate"`
		} `json:"audio_params"`
		Model string `json:"model,omitempty"`
	} `json:"req_params,omitempty"`
}

type doubaoPreviewEvent struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Data    *string `json:"data"`
}

func (vcc *VoiceCloneController) cloneWithDoubao(ctx context.Context, ttsCfg models.Config, ttsConfigID, filePath, fileName, transcript string) (*minimaxVoiceCloneResult, error) {
	cfgMap := make(map[string]any)
	if strings.TrimSpace(ttsCfg.JsonData) != "" {
		if err := json.Unmarshal([]byte(ttsCfg.JsonData), &cfgMap); err != nil {
			return nil, fmt.Errorf("failed to parse Doubao TTS config: %w", err)
		}
	}

	appID := strings.TrimSpace(getStringAny(cfgMap, "appid"))
	accessToken := strings.TrimSpace(getStringAny(cfgMap, "access_token"))
	if appID == "" || accessToken == "" {
		return nil, fmt.Errorf("Doubao clone missing appid or access_token")
	}
	modelType, targetModel := resolveDoubaoCloneTargetModel(getStringAny(cfgMap, "model"))
	resourceID := resolveDoubaoModelSelection(targetModel, "").ResourceID
	uploadURL := strings.TrimSpace(getStringAny(cfgMap, "clone_upload_url"))
	if uploadURL == "" {
		uploadURL = defaultDoubaoCloneUploadEndpoint
	}
	statusURL := strings.TrimSpace(getStringAny(cfgMap, "clone_status_url"))
	if statusURL == "" {
		statusURL = defaultDoubaoCloneStatusEndpoint
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Doubao clone audio file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("appid", appID)
	_ = writer.WriteField("language", "zh")
	_ = writer.WriteField("model_type", strconv.Itoa(modelType))
	if strings.TrimSpace(transcript) != "" {
		_ = writer.WriteField("demo_text", strings.TrimSpace(transcript))
	}
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Doubao clone upload form: %w", err)
	}
	if _, err = io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to write Doubao clone audio: %w", err)
	}
	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to build Doubao clone request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create Doubao clone request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer;%s", accessToken))
	req.Header.Set("X-Api-App-Id", appID)
	req.Header.Set("X-Api-Access-Key", accessToken)
	req.Header.Set("X-Api-Resource-Id", resourceID)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := vcc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Doubao clone upload endpoint: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read Doubao clone upload response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("Doubao clone upload HTTP %d: %s", resp.StatusCode, truncateForLog(strings.TrimSpace(string(respBody)), 1024))
	}

	uploadResp := doubaoCloneUploadResponse{}
	_ = json.Unmarshal(respBody, &uploadResp)
	uploadMap, err := unmarshalJSONMap(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Doubao clone upload response: %w", err)
	}
	if uploadResp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("Doubao clone upload failed (code=%d,msg=%s)", uploadResp.BaseResp.StatusCode, uploadResp.BaseResp.StatusMsg)
	}
	speakerID := firstNonEmptyDoubaoVoiceID(uploadResp.ICLSpeakerID, uploadResp.SpeakerID, getStringAny(uploadMap, "icl_speaker_id"), getStringAny(uploadMap, "speaker_id"))
	if speakerID == "" {
		return nil, fmt.Errorf("Doubao clone upload succeeded but no speaker_id returned")
	}

	statusResult, statusRaw, statusHTTPCode, err := vcc.pollDoubaoCloneStatus(ctx, statusURL, appID, accessToken, resourceID, speakerID)
	if err != nil {
		return nil, err
	}
	finalVoiceID := firstNonEmptyDoubaoVoiceID(
		statusResult.ICLSpeakerID,
		getStringAny(statusRaw, "icl_speaker_id"),
		getStringAny(statusRaw, "speaker"),
		statusResult.SpeakerID,
		speakerID,
	)
	return &minimaxVoiceCloneResult{
		VoiceID:      finalVoiceID,
		TargetModel:  targetModel,
		RawResponse:  statusRaw,
		ResponseCode: statusHTTPCode,
	}, nil
}

func (vcc *VoiceCloneController) pollDoubaoCloneStatus(ctx context.Context, statusURL, appID, accessToken, resourceID, speakerID string) (*doubaoCloneStatusResponse, map[string]any, int, error) {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		statusResp, raw, httpCode, err := vcc.fetchDoubaoCloneStatus(ctx, statusURL, appID, accessToken, resourceID, speakerID)
		if err != nil {
			return nil, nil, httpCode, err
		}
		if isDoubaoCloneSuccess(statusResp) {
			return statusResp, raw, httpCode, nil
		}
		if isDoubaoCloneFailed(statusResp) {
			msg := statusResp.BaseResp.StatusMsg
			if strings.TrimSpace(msg) == "" {
				msg = firstNonEmptyDoubaoVoiceID(getStringAny(raw, "message"), getStringAny(raw, "error"), "Doubao clone training failed")
			}
			return nil, nil, httpCode, fmt.Errorf("%s", msg)
		}

		select {
		case <-ctx.Done():
			return nil, nil, httpCode, fmt.Errorf("timed out waiting for Doubao clone result: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (vcc *VoiceCloneController) fetchDoubaoCloneStatus(ctx context.Context, statusURL, appID, accessToken, resourceID, speakerID string) (*doubaoCloneStatusResponse, map[string]any, int, error) {
	payload := map[string]any{
		"appid":      appID,
		"speaker_id": speakerID,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to serialize Doubao clone status request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, statusURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to create Doubao clone status request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer;%s", accessToken))
	req.Header.Set("X-Api-App-Id", appID)
	req.Header.Set("X-Api-Access-Key", accessToken)
	req.Header.Set("X-Api-Resource-Id", resourceID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := vcc.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to call Doubao clone status endpoint: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, nil, resp.StatusCode, fmt.Errorf("failed to read Doubao clone status response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, nil, resp.StatusCode, fmt.Errorf("Doubao clone status HTTP %d: %s", resp.StatusCode, truncateForLog(strings.TrimSpace(string(respBody)), 1024))
	}

	statusResp := &doubaoCloneStatusResponse{}
	_ = json.Unmarshal(respBody, statusResp)
	raw, err := unmarshalJSONMap(respBody)
	if err != nil {
		return nil, nil, resp.StatusCode, fmt.Errorf("failed to parse Doubao clone status response: %w", err)
	}
	if statusResp.BaseResp.StatusCode != 0 {
		return nil, nil, resp.StatusCode, fmt.Errorf("Doubao clone status query failed (code=%d,msg=%s)", statusResp.BaseResp.StatusCode, statusResp.BaseResp.StatusMsg)
	}
	return statusResp, raw, resp.StatusCode, nil
}

func (vcc *VoiceCloneController) previewDoubaoClonedVoice(ctx context.Context, cfgMap map[string]any, voiceID, text string) ([]byte, string, error) {
	appID := strings.TrimSpace(getStringAny(cfgMap, "appid"))
	accessToken := strings.TrimSpace(getStringAny(cfgMap, "access_token"))
	if appID == "" || accessToken == "" {
		return nil, "", fmt.Errorf("Doubao TTS missing appid or access_token")
	}
	selection := resolveDoubaoModelSelection(getStringAny(cfgMap, "model"), voiceID)
	endpoint := strings.TrimSpace(getStringAny(cfgMap, "api_url"))
	if endpoint == "" {
		endpoint = defaultDoubaoPreviewEndpoint
	}

	reqBody := doubaoPreviewRequest{}
	reqBody.User.UID = randomDigits(12)
	reqBody.ReqParams.Text = text
	reqBody.ReqParams.Speaker = strings.TrimSpace(voiceID)
	reqBody.ReqParams.AudioParams.Format = "mp3"
	reqBody.ReqParams.AudioParams.SampleRate = 24000
	reqBody.ReqParams.Model = selection.RequestModel

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to serialize Doubao preview request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create Doubao preview request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer;%s", accessToken))
	req.Header.Set("X-Api-App-Id", appID)
	req.Header.Set("X-Api-Access-Key", accessToken)
	req.Header.Set("X-Api-Resource-Id", selection.ResourceID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := vcc.HTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to call Doubao preview: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, "", fmt.Errorf("Doubao preview HTTP %d: %s", resp.StatusCode, truncateForLog(strings.TrimSpace(string(body)), 512))
	}

	reader := bufio.NewReader(resp.Body)
	var merged []byte
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, "", fmt.Errorf("failed to read Doubao preview stream: %w", err)
		}
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "event:") {
			if strings.HasPrefix(line, "data:") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
			if line != "" && line != "[DONE]" {
				var event doubaoPreviewEvent
				if unmarshalErr := json.Unmarshal([]byte(line), &event); unmarshalErr != nil {
					return nil, "", fmt.Errorf("failed to parse Doubao preview event: %w", unmarshalErr)
				}
				if event.Code != 0 {
					return nil, "", fmt.Errorf("Doubao preview failed (code=%d,msg=%s)", event.Code, event.Message)
				}
				if event.Data != nil && strings.TrimSpace(*event.Data) != "" {
					chunk, decodeErr := base64.StdEncoding.DecodeString(strings.TrimSpace(*event.Data))
					if decodeErr != nil {
						return nil, "", fmt.Errorf("failed to decode Doubao preview audio: %w", decodeErr)
					}
					merged = append(merged, chunk...)
				}
			}
		}
		if err == io.EOF {
			break
		}
	}
	if len(merged) == 0 {
		return nil, "", fmt.Errorf("Doubao preview returned empty audio")
	}
	return merged, "audio/mpeg", nil
}

func firstNonEmptyDoubaoVoiceID(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func isDoubaoCloneSuccess(resp *doubaoCloneStatusResponse) bool {
	if resp == nil {
		return false
	}
	statuses := []string{
		strings.ToLower(strings.TrimSpace(resp.TrainStatus)),
		strings.ToLower(strings.TrimSpace(resp.Status)),
	}
	for _, status := range statuses {
		switch status {
		case "9", "success", "succeeded", "done", "completed", "finish", "finished":
			return true
		}
	}
	return false
}

func isDoubaoCloneFailed(resp *doubaoCloneStatusResponse) bool {
	if resp == nil {
		return true
	}
	statuses := []string{
		strings.ToLower(strings.TrimSpace(resp.TrainStatus)),
		strings.ToLower(strings.TrimSpace(resp.Status)),
	}
	for _, status := range statuses {
		switch status {
		case "-1", "0", "failed", "error", "rejected":
			return true
		}
	}
	return false
}
