package doubao

import (
	"fmt"
	"strings"
)

const (
	defaultDoubaoTTSModel  = "seed-tts-1.1"
	defaultDoubaoHTTPURL   = "https://openspeech.bytedance.com/api/v3/tts/unidirectional"
	defaultDoubaoWSURL     = "wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream"
	defaultDoubaoAudioFmt  = "mp3"
	defaultDoubaoSampleHz  = 24000
	resourceSeedTTS10      = "seed-tts-1.0"
	resourceSeedTTS20      = "seed-tts-2.0"
	resourceSeedICL10      = "seed-icl-1.0"
	resourceSeedICL20      = "seed-icl-2.0"
	modelSeedTTS11         = "seed-tts-1.1"
	modelSeedTTS20Standard = "seed-tts-2.0-standard"
	modelSeedTTS20Expr     = "seed-tts-2.0-expressive"
	modelSeedICL10         = "seed-icl-1.0"
	modelSeedICL20Standard = "seed-icl-2.0-standard"
	modelSeedICL20Expr     = "seed-icl-2.0-expressive"
)

type resolvedTTSModel struct {
	ConfigModel    string
	RequestModel   string
	ResourceID     string
	IsCloneVoice   bool
	VoiceFamily    string
	DerivedByVoice bool
}

func normalizeDoubaoTTSURL(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	return raw
}

func normalizeDoubaoModel(model string) string {
	model = strings.TrimSpace(strings.ToLower(model))
	switch model {
	case "", "default":
		return ""
	case strings.ToLower(modelSeedTTS11):
		return modelSeedTTS11
	case strings.ToLower(modelSeedTTS20Standard), "seed-tts-2.0":
		return modelSeedTTS20Standard
	case strings.ToLower(modelSeedTTS20Expr):
		return modelSeedTTS20Expr
	case strings.ToLower(modelSeedICL10):
		return modelSeedICL10
	case strings.ToLower(modelSeedICL20Standard):
		return modelSeedICL20Standard
	case strings.ToLower(modelSeedICL20Expr):
		return modelSeedICL20Expr
	default:
		return strings.TrimSpace(model)
	}
}

func inferDoubaoVoiceFamily(voice string) string {
	voice = strings.TrimSpace(strings.ToLower(voice))
	switch {
	case voice == "":
		return "unknown"
	case strings.HasPrefix(voice, "saturn_"):
		return "tts2"
	case strings.HasPrefix(voice, "s_"):
		return "icl1"
	case strings.HasPrefix(voice, "icl_"):
		return "icl1"
	case strings.Contains(voice, "_bigtts"):
		return "tts2"
	default:
		return "tts1"
	}
}

func isDoubaoCloneVoice(voice string) bool {
	switch inferDoubaoVoiceFamily(voice) {
	case "icl1", "icl2":
		return true
	default:
		return false
	}
}

func resolveDoubaoTTSModel(model, voice string) (resolvedTTSModel, error) {
	voiceFamily := inferDoubaoVoiceFamily(voice)
	isClone := voiceFamily == "icl1" || voiceFamily == "icl2"
	normalized := normalizeDoubaoModel(model)
	if voiceFamily == "tts2" && normalized == modelSeedTTS11 {
		normalized = modelSeedTTS20Standard
	}
	if normalized == "" {
		switch voiceFamily {
		case "icl2":
			return resolvedTTSModel{
				ConfigModel:    modelSeedICL20Expr,
				RequestModel:   modelSeedTTS20Expr,
				ResourceID:     resourceSeedICL20,
				IsCloneVoice:   true,
				VoiceFamily:    voiceFamily,
				DerivedByVoice: true,
			}, nil
		case "icl1":
			return resolvedTTSModel{
				ConfigModel:    modelSeedICL10,
				RequestModel:   "",
				ResourceID:     resourceSeedICL10,
				IsCloneVoice:   true,
				VoiceFamily:    voiceFamily,
				DerivedByVoice: true,
			}, nil
		case "tts2":
			return resolvedTTSModel{
				ConfigModel:    modelSeedTTS20Standard,
				RequestModel:   "",
				ResourceID:     resourceSeedTTS20,
				IsCloneVoice:   false,
				VoiceFamily:    voiceFamily,
				DerivedByVoice: true,
			}, nil
		default:
			return resolvedTTSModel{
				ConfigModel:    defaultDoubaoTTSModel,
				RequestModel:   modelSeedTTS11,
				ResourceID:     resourceSeedTTS10,
				IsCloneVoice:   false,
				VoiceFamily:    voiceFamily,
				DerivedByVoice: true,
			}, nil
		}
	}

	resolved := resolvedTTSModel{
		ConfigModel:  normalized,
		IsCloneVoice: isClone,
		VoiceFamily:  voiceFamily,
	}
	switch normalized {
	case modelSeedTTS11:
		if voiceFamily == "tts2" {
			resolved.RequestModel = ""
			resolved.ResourceID = resourceSeedTTS20
			break
		}
		resolved.RequestModel = modelSeedTTS11
		resolved.ResourceID = resourceSeedTTS10
	case modelSeedTTS20Standard:
		if voiceFamily == "tts2" {
			resolved.RequestModel = ""
		} else {
			resolved.RequestModel = modelSeedTTS20Standard
		}
		resolved.ResourceID = resourceSeedTTS20
	case modelSeedTTS20Expr:
		if voiceFamily == "tts2" {
			resolved.RequestModel = ""
		} else {
			resolved.RequestModel = modelSeedTTS20Expr
		}
		resolved.ResourceID = resourceSeedTTS20
	case modelSeedICL10:
		resolved.RequestModel = ""
		resolved.ResourceID = resourceSeedICL10
	case modelSeedICL20Standard:
		resolved.RequestModel = modelSeedTTS20Standard
		resolved.ResourceID = resourceSeedICL20
	case modelSeedICL20Expr:
		resolved.RequestModel = modelSeedTTS20Expr
		resolved.ResourceID = resourceSeedICL20
	default:
		return resolvedTTSModel{}, fmt.Errorf("Unsupported Doubao TTS model: %s", model)
	}

	if voiceFamily == "icl1" && resolved.ResourceID != resourceSeedICL10 {
		return resolvedTTSModel{}, fmt.Errorf("Doubao's remastered 1.0 tone needs to match the seed-icl-1.0 model family")
	}
	if voiceFamily == "icl2" && resolved.ResourceID != resourceSeedICL20 {
		return resolvedTTSModel{}, fmt.Errorf("Doubao's remastered 2.0 timbre needs to match the seed-icl-2.0 model family")
	}
	if voiceFamily == "tts1" && strings.HasPrefix(resolved.ResourceID, "seed-icl-") {
		return resolvedTTSModel{}, fmt.Errorf("Doubao public version sounds cannot use ICL replica model family")
	}
	if voiceFamily == "tts1" && resolved.ResourceID == resourceSeedTTS20 {
		return resolvedTTSModel{}, fmt.Errorf("Doubao 1.0 public version sounds need to match the seed-tts-1.0 model family")
	}

	return resolved, nil
}
