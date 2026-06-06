package asr

import (
	"context"
	"fmt"

	"xiaozhi-esp32-server-golang/constants"
	"xiaozhi-esp32-server-golang/internal/domain/asr/doubao"
	"xiaozhi-esp32-server-golang/internal/domain/asr/types"
	log "xiaozhi-esp32-server-golang/logger"
)

// Asr speech recognition interface
type AsrProvider interface {
	//Process processes the entire audio segment at once and returns complete recognition results.
	Process(pcmData []float32) (string, error)

	//StreamingRecognize streaming recognition interface
	//The input audio data passes through the audioStream channel, and the recognition result is obtained through the returned channel.
	//When audioStream is closed, it indicates the end of input, the final result will be sent through the returned channel, and then the channel is closed
	//Cancellation and timeout of the recognition process can be controlled through ctx
	StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan types.StreamingResult, error)
	//Close closes resources, releases connections, etc.
	Close() error
	//IsValid checks whether the resource is valid
	IsValid() bool
}

// NewAsrProvider creates a new ASR instance
// asrType: ASR engine type, currently supports "funasr"
// config: ASR engine configuration, of type map[string]interface{}
func NewAsrProvider(asrType string, config map[string]interface{}) (AsrProvider, error) {
	//Prioritize using the provider in config, otherwise use the provider in the parameters
	if configProvider, ok := config["provider"].(string); ok && configProvider != "" {
		asrType = configProvider
	}
	switch asrType {
	case constants.AsrTypeFunAsr:
		return NewFunasrAdapter(config)
	case constants.AsrTypeAliyunFunASR:
		return NewAliyunFunASRAdapter(config)
	case constants.AsrTypeDoubao:
		log.Info("Using the DoubaoASR provider")
		provider, err := doubao.NewDoubaoV2Adapter(config)
		if err != nil {
			log.Errorf("DoubaoASR adapter creation failed: %v", err)
		} else {
			log.Info("DoubaoASR adapter created successfully")
		}
		return provider, err
	case constants.AsrTypeAliyunQwen3:
		log.Info("Using Alibaba Cloud Qwen3 ASR provider")
		provider, err := NewAliyunQwen3Adapter(config)
		if err != nil {
			log.Errorf("Alibaba Cloud Qwen3 ASR adapter creation failed: %v", err)
		} else {
			log.Info("Alibaba Cloud Qwen3 ASR adapter created successfully")
		}
		return provider, err
	case constants.AsrTypeXunfei:
		log.Info("Using Xunfei ASR provider")
		provider, err := NewXunfeiAdapter(config)
		if err != nil {
			log.Errorf("Xunfei ASR adapter creation failed: %v", err)
		} else {
			log.Info("Xunfei ASR adapter created successfully")
		}
		return provider, err
	default:
		return nil, fmt.Errorf("Unsupported ASR engine type: %s, currently only supports 'funasr', 'aliyun_funasr', 'doubao', 'aliyun_qwen3', 'xunfei'", asrType)
	}
}
