package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleTTSSpeechReturnsOpus(t *testing.T) {
	cfg := &serverConfig{
		ttsMode:       "beep",
		ttsDurationMs: 400,
		ttsSampleRate: 16000,
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(`{
		"model":"tts-1",
		"input":"hello",
		"voice":"alloy",
		"response_format":"opus"
	}`))
	rec := httptest.NewRecorder()

	cfg.handleTTSSpeech(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status code 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	if got := rec.Header().Get("Content-Type"); got != "audio/ogg" {
		t.Fatalf("expected Content-Type=audio/ogg, got %s", got)
	}

	body := rec.Body.Bytes()
	if len(body) == 0 {
		t.Fatal("no audio data returned")
	}
	if !bytes.HasPrefix(body, []byte("OggS")) {
		t.Fatalf("expected Ogg Opus data, got first 4 bytes %q", body[:minInt(len(body), 4)])
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
