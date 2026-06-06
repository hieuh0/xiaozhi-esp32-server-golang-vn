package doubao

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestNewDoubaoTTSProvider(t *testing.T) {
	config := map[string]interface{}{
		"appid":        "test_app_id",
		"access_token": "test_token",
		"model":        "seed-tts-1.1",
		"voice":        "BV001_streaming",
		"api_url":      "https://api.test.com/v3/tts/unidirectional",
	}

	provider := NewDoubaoTTSProvider(config)
	if provider.AppID != "test_app_id" {
		t.Fatalf("AppID = %q", provider.AppID)
	}
	if provider.AccessToken != "test_token" {
		t.Fatalf("AccessToken = %q", provider.AccessToken)
	}
	if provider.Model != "seed-tts-1.1" {
		t.Fatalf("Model = %q", provider.Model)
	}
	if provider.Voice != "BV001_streaming" {
		t.Fatalf("Voice = %q", provider.Voice)
	}
	if provider.APIURL != "https://api.test.com/v3/tts/unidirectional" {
		t.Fatalf("APIURL = %q", provider.APIURL)
	}
}

func TestResolveDoubaoTTSModelDefaults(t *testing.T) {
	got, err := resolveDoubaoTTSModel("", "BV001_streaming")
	if err != nil {
		t.Fatalf("resolveDoubaoTTSModel default error = %v", err)
	}
	if got.ConfigModel != defaultDoubaoTTSModel {
		t.Fatalf("ConfigModel = %q", got.ConfigModel)
	}
	if got.ResourceID != resourceSeedTTS10 {
		t.Fatalf("ResourceID = %q", got.ResourceID)
	}
	if got.RequestModel != modelSeedTTS11 {
		t.Fatalf("RequestModel = %q", got.RequestModel)
	}
}

func TestResolveDoubaoTTSModelCloneVoice(t *testing.T) {
	got, err := resolveDoubaoTTSModel("", "ICL_demo_voice")
	if err != nil {
		t.Fatalf("resolveDoubaoTTSModel clone error = %v", err)
	}
	if got.ResourceID != resourceSeedICL10 {
		t.Fatalf("ResourceID = %q", got.ResourceID)
	}
	if got.ConfigModel != modelSeedICL10 {
		t.Fatalf("ConfigModel = %q", got.ConfigModel)
	}
}

func TestResolveDoubaoTTSModelSaturnVoiceDefaultsToTTS20(t *testing.T) {
	got, err := resolveDoubaoTTSModel("", "saturn_zh_female_cancan_tob")
	if err != nil {
		t.Fatalf("resolveDoubaoTTSModel saturn error = %v", err)
	}
	if got.ResourceID != resourceSeedTTS20 {
		t.Fatalf("ResourceID = %q", got.ResourceID)
	}
	if got.ConfigModel != modelSeedTTS20Standard {
		t.Fatalf("ConfigModel = %q", got.ConfigModel)
	}
}

func TestResolveDoubaoTTSModelBigTTSVoiceDefaultsToTTS20(t *testing.T) {
	got, err := resolveDoubaoTTSModel("", "zh_female_qinqienvsheng_moon_bigtts")
	if err != nil {
		t.Fatalf("resolveDoubaoTTSModel bigtts error = %v", err)
	}
	if got.ResourceID != resourceSeedTTS20 {
		t.Fatalf("ResourceID = %q", got.ResourceID)
	}
	if got.ConfigModel != modelSeedTTS20Standard {
		t.Fatalf("ConfigModel = %q", got.ConfigModel)
	}
}

func TestResolveDoubaoTTSModelUpgradesLegacyTTS10ForBigTTSVoice(t *testing.T) {
	got, err := resolveDoubaoTTSModel(modelSeedTTS11, "zh_female_qinqienvsheng_moon_bigtts")
	if err != nil {
		t.Fatalf("resolveDoubaoTTSModel legacy tts1.0 error = %v", err)
	}
	if got.ResourceID != resourceSeedTTS20 {
		t.Fatalf("ResourceID = %q", got.ResourceID)
	}
	if got.ConfigModel != modelSeedTTS20Standard {
		t.Fatalf("ConfigModel = %q", got.ConfigModel)
	}
}

func TestResolveDoubaoTTSModelRejectsPublicVoiceWithICL(t *testing.T) {
	if _, err := resolveDoubaoTTSModel(modelSeedICL10, "BV001_streaming"); err == nil {
		t.Fatal("expected public voice with ICL model to fail")
	}
}

func TestBuildDoubaoWSAttemptModelsWithExplicitResourceAddsFallback(t *testing.T) {
	derived, err := resolveDoubaoTTSModel(modelSeedTTS20Standard, "zh_female_vv_uranus_bigtts")
	if err != nil {
		t.Fatalf("resolveDoubaoTTSModel error = %v", err)
	}

	got := buildDoubaoWSAttemptModels(derived, "TTS-SeedTTS2.02000000628041826146", "zh_female_vv_uranus_bigtts")
	if len(got) != 3 {
		t.Fatalf("attempt model len = %d", len(got))
	}
	if got[0].ResourceID != "TTS-SeedTTS2.02000000628041826146" {
		t.Fatalf("first resource = %q", got[0].ResourceID)
	}
	if got[1].ResourceID != resourceSeedTTS20 {
		t.Fatalf("second resource = %q", got[1].ResourceID)
	}
	if got[2].ResourceID != resourceSeedTTS10 {
		t.Fatalf("third resource = %q", got[2].ResourceID)
	}
}

func TestSummarizeDoubaoWSAttemptErrorResourceNotGranted(t *testing.T) {
	err := summarizeDoubaoWSAttemptError(
		"zh_female_vv_uranus_bigtts",
		resolvedTTSModel{ConfigModel: modelSeedTTS20Standard, ResourceID: "TTS-SeedTTS2.02000000628041826146"},
		[]string{"TTS-SeedTTS2.02000000628041826146"},
		[]error{errors.New(`Failed to establish Doubao WebSocket TTS connection: websocket handshake status=403 body={"error":"[resource_id=TTS-SeedTTS2.02000000628041826146] requested resource not granted"}`)},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "resource_id Unauthorized") {
		t.Fatalf("unexpected error = %v", err)
	}
}

func TestSummarizeDoubaoWSAttemptErrorExplicitUnauthorizedThenMismatch(t *testing.T) {
	err := summarizeDoubaoWSAttemptError(
		"zh_female_vv_uranus_bigtts",
		resolvedTTSModel{ConfigModel: modelSeedTTS20Standard, ResourceID: "TTS-SeedTTS2.02000000628041826146"},
		[]string{"TTS-SeedTTS2.02000000628041826146", resourceSeedTTS20},
		[]error{
			errors.New(`Failed to establish Doubao WebSocket TTS connection: websocket handshake status=403 body={"error":"[resource_id=TTS-SeedTTS2.02000000628041826146] requested resource not granted"}`),
			errors.New(`resource ID is mismatched with speaker related resource`),
		},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "explicit resource_id is not authorized") {
		t.Fatalf("unexpected error = %v", err)
	}
}

func TestNewDoubaoTTSV3RequestUsesNestedReqParams(t *testing.T) {
	req := newDoubaoTTSV3Request("你好", "voice-demo", 24000, modelSeedTTS11)

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal request error = %v", err)
	}

	if _, exists := payload["text"]; exists {
		t.Fatal("unexpected top-level text field")
	}
	reqParams, ok := payload["req_params"].(map[string]any)
	if !ok {
		t.Fatalf("req_params missing: %#v", payload)
	}
	if reqParams["text"] != "你好" {
		t.Fatalf("req_params.text = %#v", reqParams["text"])
	}
	if reqParams["speaker"] != "voice-demo" {
		t.Fatalf("req_params.speaker = %#v", reqParams["speaker"])
	}
	audioParams, ok := reqParams["audio_params"].(map[string]any)
	if !ok {
		t.Fatalf("audio_params missing: %#v", reqParams)
	}
	if audioParams["format"] != defaultDoubaoAudioFmt {
		t.Fatalf("audio_params.format = %#v", audioParams["format"])
	}
}
