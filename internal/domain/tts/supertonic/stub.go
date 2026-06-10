//go:build !supertonic

package supertonic

import (
	"context"
	"fmt"
)

// SupertonicTTSProvider stub — requires CGO + local Supertonic SDK.
// Build with -tags supertonic to enable; see docs for setup instructions.
type SupertonicTTSProvider struct{}

func NewSupertonicTTSProvider(_ map[string]interface{}) *SupertonicTTSProvider {
	return &SupertonicTTSProvider{}
}

func (p *SupertonicTTSProvider) TextToSpeech(_ context.Context, _ string, _ int, _ int, _ int) ([][]byte, error) {
	return nil, fmt.Errorf("supertonic TTS not available: build with -tags supertonic after setting up the local SDK")
}

func (p *SupertonicTTSProvider) TextToSpeechStream(_ context.Context, _ string, _ int, _ int, _ int) (chan []byte, error) {
	return nil, fmt.Errorf("supertonic TTS not available: build with -tags supertonic after setting up the local SDK")
}
