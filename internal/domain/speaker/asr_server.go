package speaker

import (
	"context"
	"fmt"
	"sync"

	log "xiaozhi-esp32-server-golang/logger"
)

// AsrServerProvider asr_server Voiceprint recognition provider
type AsrServerProvider struct {
	streamingClient *StreamingClient
	threshold       float32 //Voiceprint recognition threshold
	isActive        bool
	mutex           sync.Mutex
}

// NewAsrServerProvider creates asr_server voiceprint recognition provider
func NewAsrServerProvider(config map[string]interface{}) (*AsrServerProvider, error) {
	baseURL, ok := config["base_url"].(string)
	if !ok || baseURL == "" {
		return nil, fmt.Errorf("The service.base_url field is missing from the configuration")
	}

	//Read threshold configuration, default value is 0.4
	threshold := float32(0.4)
	if thresholdVal, ok := config["threshold"]; ok {
		switch v := thresholdVal.(type) {
		case float64:
			threshold = float32(v)
		case float32:
			threshold = v
		case int:
			threshold = float32(v)
		case int64:
			threshold = float32(v)
		}
		//Validation threshold range
		if threshold < 0 || threshold > 1 {
			log.Warnf("Threshold %.4f is outside valid range [0.0, 1.0], use default value 0.4", threshold)
			threshold = 0.4
		}
	}

	streamingClient := NewStreamingClient(baseURL)
	return &AsrServerProvider{
		streamingClient: streamingClient,
		threshold:       threshold,
		isActive:        false,
	}, nil
}

// StartStreaming starts streaming recognition
func (p *AsrServerProvider) StartStreaming(ctx context.Context, sampleRate int, agentId string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isActive {
		return nil //Already activated, return directly
	}

	err := p.streamingClient.Connect(sampleRate, agentId, p.threshold)
	if err != nil {
		log.Warnf("Failed to start voiceprint recognition stream: %v", err)
		return err
	}

	p.isActive = true
	log.Debugf("The voiceprint recognition stream has been started, sampling rate: %d Hz, agent_id: %s, threshold: %.4f", sampleRate, agentId, p.threshold)
	return nil
}

// SendAudioChunk sends an audio chunk
func (p *AsrServerProvider) SendAudioChunk(ctx context.Context, pcmData []float32) error {
	p.mutex.Lock()
	isActive := p.isActive
	streamingClient := p.streamingClient
	p.mutex.Unlock()

	if !isActive {
		return nil //Not activated, silently ignored
	}

	err := streamingClient.SendAudioChunk(pcmData)
	if err != nil {
		log.Warnf("Failed to send audio chunks to voiceprint recognition service: %v", err)
		//When sending fails, it is marked as inactive.
		p.mutex.Lock()
		p.isActive = false
		p.mutex.Unlock()
		return err
	}

	return nil
}

// FinishAndIdentify completes identification and obtains results
func (p *AsrServerProvider) FinishAndIdentify(ctx context.Context) (*IdentifyResult, error) {
	p.mutex.Lock()
	if !p.isActive {
		p.mutex.Unlock()
		return nil, nil //Not activated, returns nil
	}
	p.isActive = false
	streamingClient := p.streamingClient
	p.mutex.Unlock()

	result, err := streamingClient.FinishAndIdentify(ctx)

	if err != nil {
		log.Warnf("Failed to obtain voiceprint recognition result: %v", err)
		return nil, err
	}

	return result, nil
}

// PeekAndIdentify gets the intermediate identification results (without ending the current round)
// Returns: recognition result, whether it is stabilized by the server, error
func (p *AsrServerProvider) PeekAndIdentify(ctx context.Context, requestID string) (*IdentifyResult, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}

	p.mutex.Lock()
	isActive := p.isActive
	streamingClient := p.streamingClient
	p.mutex.Unlock()

	if !isActive {
		return nil, false, nil
	}

	result, throttled, err := streamingClient.PeekAndIdentify(ctx, requestID)
	if err != nil {
		if !streamingClient.IsConnected() {
			p.mutex.Lock()
			p.isActive = false
			p.mutex.Unlock()
		}
		log.Warnf("Failed to obtain intermediate voiceprint recognition result: %v", err)
		return nil, throttled, err
	}

	return result, throttled, nil
}

// Close Close voiceprint provider
func (p *AsrServerProvider) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.isActive = false
	if p.streamingClient != nil {
		return p.streamingClient.Close()
	}
	return nil
}

// IsActive checks whether it is active
func (p *AsrServerProvider) IsActive() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.isActive
}
