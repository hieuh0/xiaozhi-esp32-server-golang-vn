package controllers

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// PreviewTTSAudio synthesizes TTS audio via the main server and returns WAV bytes
// for direct browser playback. The request body mirrors the TTS config saved in DB.
func (ac *AdminController) PreviewTTSAudio(c *gin.Context) {
	var body struct {
		Provider string                 `json:"provider"`
		Config   map[string]interface{} `json:"config"`
		Text     string                 `json:"text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if ac.WebSocketController == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "main server not connected"})
		return
	}
	clientUUID := ac.WebSocketController.GetFirstConnectedClientUUID()
	if clientUUID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "main server not connected"})
		return
	}

	ttsConfig := body.Config
	if body.Provider != "" {
		ttsConfig["provider"] = body.Provider
	}

	text := body.Text
	if text == "" {
		text = "Config test"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	resp, err := ac.WebSocketController.SendRequestToClient(ctx, clientUUID, "POST", "/api/config/tts-audio", map[string]interface{}{
		"tts":  ttsConfig,
		"text": text,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if resp.Status != 200 {
		msg := resp.Error
		if msg == "" {
			if m, _ := resp.Body["message"].(string); m != "" {
				msg = m
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return
	}

	audioBase64, _ := resp.Body["audio_base64"].(string)
	if audioBase64 == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no audio returned from main server"})
		return
	}
	firstPacketMs, _ := resp.Body["first_packet_ms"].(float64)

	audioBytes, err := base64.StdEncoding.DecodeString(audioBase64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "audio decode failed"})
		return
	}

	c.Header("X-First-Packet-Ms", itoa(int64(firstPacketMs)))
	c.Data(http.StatusOK, "audio/wav", audioBytes)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
