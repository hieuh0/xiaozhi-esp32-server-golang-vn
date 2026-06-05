package controllers

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPickVoiceID(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]any
		want string
	}{
		{name: "top-level voice id", in: map[string]any{"voice_id": "vc_1"}, want: "vc_1"},
		{name: "nested voice id", in: map[string]any{"data": map[string]any{"voiceId": "vc_2"}}, want: "vc_2"},
		{name: "speaker id fallback", in: map[string]any{"speaker_id": "spk_1"}, want: "spk_1"},
		{name: "missing", in: map[string]any{"data": map[string]any{"x": "y"}}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pickVoiceID(tt.in); got != tt.want {
				t.Fatalf("pickVoiceID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildMinimaxCustomVoiceID(t *testing.T) {
	tests := []struct {
		name        string
		ttsConfigID string
		prefix      string
	}{
		{
			name:        "normal config id",
			ttsConfigID: "minimax_default",
			prefix:      "minimax_default_",
		},
		{
			name:        "sanitize non identifier chars",
			ttsConfigID: "minimax-default.v1",
			prefix:      "minimax_default_v1_",
		},
		{
			name:        "empty fallback",
			ttsConfigID: "   ",
			prefix:      "voice_",
		},
	}

	digitsPattern := regexp.MustCompile(`^\d{8}$`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildMinimaxCustomVoiceID(tt.ttsConfigID)
			if !strings.HasPrefix(got, tt.prefix) {
				t.Fatalf("buildMinimaxCustomVoiceID() prefix = %q, want prefix %q", got, tt.prefix)
			}
			suffix := strings.TrimPrefix(got, tt.prefix)
			if !digitsPattern.MatchString(suffix) {
				t.Fatalf("buildMinimaxCustomVoiceID() suffix = %q, want 8 digits", suffix)
			}
		})
	}
}

func TestParseMinimaxStatus(t *testing.T) {
	tests := []struct {
		name       string
		in         map[string]any
		wantCode   int
		wantMsg    string
		wantParsed bool
	}{
		{
			name:       "parsed",
			in:         map[string]any{"base_resp": map[string]any{"status_code": float64(2013), "status_msg": "invalid params"}},
			wantCode:   2013,
			wantMsg:    "invalid params",
			wantParsed: true,
		},
		{
			name:       "missing base_resp",
			in:         map[string]any{"data": map[string]any{}},
			wantParsed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg, ok := parseMinimaxStatus(tt.in)
			if ok != tt.wantParsed {
				t.Fatalf("parseMinimaxStatus() parsed = %v, want %v", ok, tt.wantParsed)
			}
			if !ok {
				return
			}
			if code != tt.wantCode || msg != tt.wantMsg {
				t.Fatalf("parseMinimaxStatus() = (%d, %q), want (%d, %q)", code, msg, tt.wantCode, tt.wantMsg)
			}
		})
	}
}

func TestNormalizeMinimaxAPIKey(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "sk-abc", want: "sk-abc"},
		{name: "with bearer", in: "Bearer sk-abc", want: "sk-abc"},
		{name: "with lowercase bearer and spaces", in: "  bearer   sk-abc  ", want: "sk-abc"},
		{name: "quoted", in: "\"sk-abc\"", want: "sk-abc"},
		{name: "empty", in: "  ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeMinimaxAPIKey(tt.in); got != tt.want {
				t.Fatalf("normalizeMinimaxAPIKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetStringOrNumberAny(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]any
		keys []string
		want string
	}{
		{
			name: "string",
			in:   map[string]any{"file_id": "abc"},
			keys: []string{"file_id"},
			want: "abc",
		},
		{
			name: "json number",
			in:   map[string]any{"file_id": json.Number("367885918900670")},
			keys: []string{"file_id"},
			want: "367885918900670",
		},
		{
			name: "float64 number from unmarshal",
			in:   map[string]any{"file_id": float64(367885918900670)},
			keys: []string{"file_id"},
			want: "367885918900670",
		},
		{
			name: "missing",
			in:   map[string]any{"x": "y"},
			keys: []string{"file_id"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStringOrNumberAny(tt.in, tt.keys...); got != tt.want {
				t.Fatalf("getStringOrNumberAny() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnmarshalJSONMapUseNumber(t *testing.T) {
	in := []byte(`{"file":{"file_id":367885918900670},"base_resp":{"status_code":0}}`)
	parsed, err := unmarshalJSONMap(in)
	if err != nil {
		t.Fatalf("unmarshalJSONMap() error = %v", err)
	}
	fileMap, ok := parsed["file"].(map[string]any)
	if !ok {
		t.Fatalf("file map missing")
	}
	if got := getStringOrNumberAny(fileMap, "file_id"); got != "367885918900670" {
		t.Fatalf("file_id = %q, want %q", got, "367885918900670")
	}
}

func TestMakeMinimaxFileIDPayload(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantType string
		wantVal  string
	}{
		{name: "numeric", in: "367885918900670", wantType: "number", wantVal: "367885918900670"},
		{name: "string", in: "file_abc_1", wantType: "string", wantVal: "file_abc_1"},
		{name: "trim numeric", in: " 123 ", wantType: "number", wantVal: "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeMinimaxFileIDPayload(tt.in)
			switch tt.wantType {
			case "number":
				v, ok := got.(json.Number)
				if !ok {
					t.Fatalf("type = %T, want json.Number", got)
				}
				if v.String() != tt.wantVal {
					t.Fatalf("value = %q, want %q", v.String(), tt.wantVal)
				}
			case "string":
				v, ok := got.(string)
				if !ok {
					t.Fatalf("type = %T, want string", got)
				}
				if v != tt.wantVal {
					t.Fatalf("value = %q, want %q", v, tt.wantVal)
				}
			}
		})
	}
}

func TestGetWAVDurationSeconds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wav")
	const sampleRate = 16000
	const channels = 1
	const bitsPerSample = 16
	const durationSec = 12
	if err := os.WriteFile(path, buildPCM16WAV(sampleRate, channels, bitsPerSample, durationSec), 0644); err != nil {
		t.Fatalf("write wav failed: %v", err)
	}
	got, err := getWAVDurationSeconds(path)
	if err != nil {
		t.Fatalf("getWAVDurationSeconds() error = %v", err)
	}
	if got < 11.99 || got > 12.01 {
		t.Fatalf("getWAVDurationSeconds() = %.6f, want ~12", got)
	}
}

func TestGetMinimaxCloneAudioDurationSeconds(t *testing.T) {
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "ok.wav")
	if err := os.WriteFile(wavPath, buildPCM16WAV(16000, 1, 16, 10), 0644); err != nil {
		t.Fatalf("write wav failed: %v", err)
	}
	got, err := getMinimaxCloneAudioDurationSeconds(wavPath)
	if err != nil {
		t.Fatalf("getMinimaxCloneAudioDurationSeconds(wav) error = %v", err)
	}
	if got < 9.99 || got > 10.01 {
		t.Fatalf("duration = %.6f, want ~10", got)
	}

	webmPath := filepath.Join(dir, "bad.webm")
	if err := os.WriteFile(webmPath, []byte("webm"), 0644); err != nil {
		t.Fatalf("write webm failed: %v", err)
	}
	if _, err := getMinimaxCloneAudioDurationSeconds(webmPath); err == nil {
		t.Fatalf("expected error for non-wav file")
	}
}

func TestValidateCloneAudioForProvider(t *testing.T) {
	dir := t.TempDir()

	shortPath := filepath.Join(dir, "short.wav")
	if err := os.WriteFile(shortPath, buildPCM16WAV(16000, 1, 16, 2), 0644); err != nil {
		t.Fatalf("write short wav failed: %v", err)
	}
	if err := validateCloneAudioForProvider("cosyvoice", shortPath); err != nil {
		t.Fatalf("validateCloneAudioForProvider(cosyvoice) unexpected error: %v", err)
	}
	if err := validateCloneAudioForProvider("minimax", shortPath); err == nil {
		t.Fatalf("validateCloneAudioForProvider(minimax) expected duration error")
	}

	nonWavPath := filepath.Join(dir, "bad.webm")
	if err := os.WriteFile(nonWavPath, []byte("webm"), 0644); err != nil {
		t.Fatalf("write webm failed: %v", err)
	}
	if err := validateCloneAudioForProvider("cosyvoice", nonWavPath); err == nil {
		t.Fatalf("validateCloneAudioForProvider(cosyvoice/non-wav) expected error")
	}
	if err := validateCloneAudioForProvider("unknown_provider", shortPath); err == nil {
		t.Fatalf("validateCloneAudioForProvider(unknown_provider) expected error")
	}

	mp3Path := filepath.Join(dir, "ok.mp3")
	if err := os.WriteFile(mp3Path, []byte{0x49, 0x44, 0x33, 0x03, 0x00}, 0644); err != nil {
		t.Fatalf("write mp3 failed: %v", err)
	}
	if err := validateCloneAudioForProvider("aliyun_qwen", mp3Path); err != nil {
		t.Fatalf("validateCloneAudioForProvider(aliyun_qwen/mp3) unexpected error: %v", err)
	}
	if err := validateCloneAudioForProvider("minimax", mp3Path); err == nil {
		t.Fatalf("validateCloneAudioForProvider(minimax/mp3) expected error")
	}
}

func TestResolveAliyunQwenCloneEndpoint(t *testing.T) {
	if got := resolveAliyunQwenCloneEndpoint(map[string]any{"api_url": "https://dashscope-intl.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"}); got != defaultAliyunQwenCloneEndpointIntl {
		t.Fatalf("resolveAliyunQwenCloneEndpoint(intl) = %q, want %q", got, defaultAliyunQwenCloneEndpointIntl)
	}
	if got := resolveAliyunQwenCloneEndpoint(map[string]any{"clone_endpoint": "https://example.com/custom"}); got != "https://example.com/custom" {
		t.Fatalf("resolveAliyunQwenCloneEndpoint(custom) = %q", got)
	}
	if got := resolveAliyunQwenCloneEndpoint(map[string]any{}); got != defaultAliyunQwenCloneEndpoint {
		t.Fatalf("resolveAliyunQwenCloneEndpoint(default) = %q, want %q", got, defaultAliyunQwenCloneEndpoint)
	}
}

func TestResolveAliyunQwenTargetModel(t *testing.T) {
	if model := resolveAliyunQwenTargetModel(); model != defaultAliyunQwenCloneTargetModel {
		t.Fatalf("resolveAliyunQwenTargetModel() = %q, want %q", model, defaultAliyunQwenCloneTargetModel)
	}
}

func TestMapAliyunQwenCloneLanguage(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "zh-CN", want: "zh"},
		{in: "en-US", want: "en"},
		{in: "ja-JP", want: "ja"},
		{in: "ru-RU", want: "ru"},
		{in: "unknown", want: ""},
	}
	for _, tt := range tests {
		if got := mapAliyunQwenCloneLanguage(tt.in); got != tt.want {
			t.Fatalf("mapAliyunQwenCloneLanguage(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeCloneProvider(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "qwen_tts", want: "qwen_tts"},
		{in: " aliyun_qwen ", want: "aliyun_qwen"},
		{in: "minimax", want: "minimax"},
	}
	for _, tt := range tests {
		if got := normalizeCloneProvider(tt.in); got != tt.want {
			t.Fatalf("normalizeCloneProvider(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGetCloneProviderCapabilityAlias(t *testing.T) {
	if got := GetCloneProviderCapability("qwen_tts"); got.Enabled {
		t.Fatalf("GetCloneProviderCapability(qwen_tts).Enabled expected false")
	}
}

func buildPCM16WAV(sampleRate, channels, bitsPerSample, durationSec int) []byte {
	bytesPerSample := bitsPerSample / 8
	dataSize := sampleRate * channels * bytesPerSample * durationSec
	riffSize := 36 + dataSize
	buf := make([]byte, 44+dataSize)

	copy(buf[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(riffSize))
	copy(buf[8:12], []byte("WAVE"))

	copy(buf[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(buf[16:20], 16) // PCM fmt chunk size
	binary.LittleEndian.PutUint16(buf[20:22], 1)  // PCM
	binary.LittleEndian.PutUint16(buf[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(sampleRate))
	byteRate := sampleRate * channels * bytesPerSample
	binary.LittleEndian.PutUint32(buf[28:32], uint32(byteRate))
	blockAlign := channels * bytesPerSample
	binary.LittleEndian.PutUint16(buf[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(buf[34:36], uint16(bitsPerSample))

	copy(buf[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))
	// silence data (all zeros by default)
	return buf
}
