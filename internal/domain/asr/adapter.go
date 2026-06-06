package asr

import (
	"context"
	"strconv"
	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/domain/asr/funasr"
	"xiaozhi-esp32-server-golang/internal/domain/asr/types"
	log "xiaozhi-esp32-server-golang/logger"
)

// FunasrAdapter adapts the funasr package to the asr interface
type FunasrAdapter struct {
	engine *funasr.Funasr
}

// NewFunasrAdapter creates a new FunASR adapter
func NewFunasrAdapter(config map[string]interface{}) (AsrProvider, error) {
	//Create FunasrConfig configuration
	funasrConfig := funasr.FunasrConfig{
		Host:          "localhost",
		Port:          "10095",
		Mode:          "online",
		SampleRate:    audio.SampleRate,
		ChunkInterval: audio.FrameDuration,
		Timeout:       30,
		AutoEnd:       false,
	}

	log.Log().Infof("funasr config: %+v", config)

	//Get configuration items from map
	if host, ok := config["host"].(string); ok && host != "" {
		funasrConfig.Host = host
	}
	if port, ok := config["port"].(string); ok && port != "" {
		funasrConfig.Port = port
	} else if portInt, ok := config["port"].(int); ok && portInt > 0 {
		funasrConfig.Port = strconv.Itoa(portInt)
	} else if portFloat, ok := config["port"].(float64); ok && portFloat > 0 {
		funasrConfig.Port = strconv.Itoa(int(portFloat))
	}

	if mode, ok := config["mode"].(string); ok && mode != "" {
		funasrConfig.Mode = mode
	}
	if sampleRate, ok := config["sample_rate"].(int); ok && sampleRate > 0 {
		funasrConfig.SampleRate = sampleRate
	} else if sampleRateFloat, ok := config["sample_rate"].(float64); ok && sampleRateFloat > 0 {
		funasrConfig.SampleRate = int(sampleRateFloat)
	}
	if chunkInterval, ok := config["chunk_interval"].(int); ok && chunkInterval > 0 {
		funasrConfig.ChunkInterval = chunkInterval
	} else if chunkIntervalFloat, ok := config["chunk_interval"].(float64); ok && chunkIntervalFloat > 0 {
		funasrConfig.ChunkInterval = int(chunkIntervalFloat)
	}
	if timeout, ok := config["timeout"].(int); ok && timeout > 0 {
		funasrConfig.Timeout = timeout
	} else if timeoutFloat, ok := config["timeout"].(float64); ok && timeoutFloat > 0 {
		funasrConfig.Timeout = int(timeoutFloat)
	}
	if chunkSize, ok := config["chunk_size"].([]int); ok && len(chunkSize) > 0 {
		funasrConfig.ChunkSize = chunkSize
	}

	if autoEnd, ok := config["auto_end"].(bool); ok {
		funasrConfig.AutoEnd = autoEnd
	}

	//Create FunASR engine
	engine, err := funasr.NewFunasr(funasrConfig)
	if err != nil {
		return nil, err
	}
	return &FunasrAdapter{engine: engine}, nil
}

// Process implements the Asr interface
func (a *FunasrAdapter) Process(pcmData []float32) (string, error) {
	return a.engine.Process(pcmData)
}

// StreamingRecognize implements the streaming recognition interface
func (a *FunasrAdapter) StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan types.StreamingResult, error) {
	//Call the StreamingRecognize method of the funasr package
	resultChan, err := a.engine.StreamingRecognize(ctx, audioStream)
	if err != nil {
		return nil, err
	}

	return resultChan, nil
}

// Close closes the resource (stateless provider, no need to close)
func (a *FunasrAdapter) Close() error {
	return nil
}

// IsValid checks whether the resource is valid
func (a *FunasrAdapter) IsValid() bool {
	return a != nil && a.engine != nil
}
