package vad

import (
	"errors"
	"xiaozhi-esp32-server-golang/constants"
	"xiaozhi-esp32-server-golang/internal/domain/vad/inter"
	"xiaozhi-esp32-server-golang/internal/domain/vad/silero_vad"
	"xiaozhi-esp32-server-golang/internal/domain/vad/ten_vad"
	// "xiaozhi-esp32-server-golang/internal/domain/vad/webrtc_vad"
)

func AcquireVAD(provider string, config map[string]interface{}) (inter.VAD, error) {
	// Prefer provider from config; fall back to the parameter provider
	if configProvider, ok := config["provider"].(string); ok && configProvider != "" {
		provider = configProvider
	}

	// If provider is empty, return a clear error message
	if provider == "" {
		return nil, errors.New("vad provider is empty, please set provider in config (supported: silero_vad, ten_vad)")
	}

	switch provider {
	case constants.VadTypeSileroVad:
		return silero_vad.AcquireVAD(config)
	// case constants.VadTypeWebRTCVad:
	// 	return webrtc_vad.AcquireVAD(config)
	case constants.VadTypeTenVad:
		return ten_vad.AcquireVAD(config)
	default:
		return nil, errors.New("invalid vad provider: " + provider + " (supported: silero_vad, ten_vad)")
	}
}

func ReleaseVAD(vad inter.VAD) error {
	// Call the corresponding ReleaseVAD method based on the VAD type
	switch vad.(type) {
	// case *webrtc_vad.WebRTCVAD:
	// 	return webrtc_vad.ReleaseVAD(vad)
	case *silero_vad.SileroVAD:
		return silero_vad.ReleaseVAD(vad)
	case *ten_vad.TenVAD:
		return ten_vad.ReleaseVAD(vad)
	default:
		return errors.New("invalid vad type")
	}
}
