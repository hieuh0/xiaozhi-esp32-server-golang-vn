package doubao

import (
	"context"
	"fmt"

	"xiaozhi-esp32-server-golang/internal/domain/asr/types"
	log "xiaozhi-esp32-server-golang/logger"
)

// DoubaoV2Adapter adapter, which implements the existing AsrProvider interface
type DoubaoV2Adapter struct {
	engine *DoubaoV2ASR
}

// NewDoubaoV2Adapter creates a new DoubaoASR adapter
func NewDoubaoV2Adapter(config map[string]interface{}) (*DoubaoV2Adapter, error) {

	//Create DoubaoASR configuration
	doubaoConfig := DefaultConfig

	//Get configuration items from map
	if appID, ok := config["appid"].(string); ok && appID != "" {
		doubaoConfig.AppID = appID
	}
	if accessToken, ok := config["access_token"].(string); ok && accessToken != "" {
		doubaoConfig.AccessToken = accessToken
	}
	if wsURL, ok := config["ws_url"].(string); ok && wsURL != "" {
		doubaoConfig.WsURL = wsURL
	}
	if resourceID, ok := config["resource_id"].(string); ok && resourceID != "" {
		doubaoConfig.ResourceID = resourceID
	}
	if modelName, ok := config["model_name"].(string); ok && modelName != "" {
		doubaoConfig.ModelName = modelName
	}
	if endWindowSize, ok := config["end_window_size"].(int); ok && endWindowSize > 0 {
		doubaoConfig.EndWindowSize = endWindowSize
	} else if endWindowSizeFloat, ok := config["end_window_size"].(float64); ok && endWindowSizeFloat > 0 {
		doubaoConfig.EndWindowSize = int(endWindowSizeFloat)
	}
	if enablePunc, ok := config["enable_punc"].(bool); ok {
		doubaoConfig.EnablePunc = enablePunc
	}
	if enableITN, ok := config["enable_itn"].(bool); ok {
		doubaoConfig.EnableITN = enableITN
	}
	if enableDDC, ok := config["enable_ddc"].(bool); ok {
		doubaoConfig.EnableDDC = enableDDC
	}
	if resultType, ok := config["result_type"].(string); ok && resultType != "" {
		doubaoConfig.ResultType = resultType
	}
	if showUtterances, ok := config["show_utterances"].(bool); ok {
		doubaoConfig.ShowUtterances = showUtterances
	}
	if forceToSpeechTime, ok := config["force_to_speech_time"].(int); ok && forceToSpeechTime > 0 {
		doubaoConfig.ForceToSpeechTime = forceToSpeechTime
	} else if forceToSpeechTimeFloat, ok := config["force_to_speech_time"].(float64); ok && forceToSpeechTimeFloat > 0 {
		doubaoConfig.ForceToSpeechTime = int(forceToSpeechTimeFloat)
	}
	if enableNonstream, ok := config["enable_nonstream"].(bool); ok {
		doubaoConfig.EnableNonstream = enableNonstream
	}
	if chunkDuration, ok := config["chunk_duration"].(int); ok && chunkDuration > 0 {
		doubaoConfig.ChunkDuration = chunkDuration
	} else if chunkDurationFloat, ok := config["chunk_duration"].(float64); ok && chunkDurationFloat > 0 {
		doubaoConfig.ChunkDuration = int(chunkDurationFloat)
	}
	if timeout, ok := config["timeout"].(int); ok && timeout > 0 {
		doubaoConfig.Timeout = timeout
	} else if timeoutFloat, ok := config["timeout"].(float64); ok && timeoutFloat > 0 {
		doubaoConfig.Timeout = int(timeoutFloat)
	}

	//Create DoubaoASR engine
	engine, err := NewDoubaoV2ASR(doubaoConfig)
	if err != nil {
		log.Errorf("Failed to create DoubaoASR engine: %v", err)
		return nil, fmt.Errorf("Failed to create DoubaoASR engine: %v", err)
	}
	log.Info("DoubaoASR engine created successfully")

	return &DoubaoV2Adapter{
		engine: engine,
	}, nil
}

// Process processes the entire audio segment at once and returns complete recognition results.
func (d *DoubaoV2Adapter) Process(pcmData []float32) (string, error) {
	return "", nil
}

// StreamingRecognize implements the streaming recognition interface
func (d *DoubaoV2Adapter) StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan types.StreamingResult, error) {
	return d.engine.StreamingRecognize(ctx, audioStream)
}

// Close closes resources, releases connections, etc.
func (d *DoubaoV2Adapter) Close() error {
	if d.engine != nil {
		return d.engine.Close()
	}
	return nil
}

// IsValid checks whether the resource is valid
func (d *DoubaoV2Adapter) IsValid() bool {
	return d != nil && d.engine != nil
}
