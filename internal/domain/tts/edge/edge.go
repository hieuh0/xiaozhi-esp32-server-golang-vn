package edge

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/difyz9/edge-tts-go/pkg/communicate"
)

// EdgeTTSProvider Edge TTS Provider
// Supports one-time and streaming TTS, output Opus frames
// Configuration parameters: voice, rate, volume, pitch, connectTimeout, receiveTimeout
type EdgeTTSProvider struct {
	Voice          string
	Rate           string
	Volume         string
	Pitch          string
	ConnectTimeout int
	ReceiveTimeout int
}

// NewEdgeTTSProvider Create EdgeTTSProvider
func NewEdgeTTSProvider(config map[string]interface{}) *EdgeTTSProvider {
	voice, _ := config["voice"].(string)
	rate, _ := config["rate"].(string)
	volume, _ := config["volume"].(string)
	pitch, _ := config["pitch"].(string)
	connectTimeout, _ := config["connect_timeout"].(int)
	receiveTimeout, _ := config["receive_timeout"].(int)
	if rate == "" {
		rate = "+0%"
	}
	if volume == "" {
		volume = "+0%"
	}
	if pitch == "" {
		pitch = "+0Hz"
	}
	if connectTimeout == 0 {
		connectTimeout = 10
	}
	if receiveTimeout == 0 {
		receiveTimeout = 60
	}
	return &EdgeTTSProvider{
		Voice:          voice,
		Rate:           rate,
		Volume:         volume,
		Pitch:          pitch,
		ConnectTimeout: connectTimeout,
		ReceiveTimeout: receiveTimeout,
	}
}

// TextToSpeech one-time synthesis, returns Opus frame
func (p *EdgeTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	startTs := time.Now().UnixMilli()
	//Temporary MP3 files
	tmpFile := fmt.Sprintf("/tmp/edge-tts-%d.mp3", time.Now().UnixNano())
	defer os.Remove(tmpFile)

	comm, err := communicate.NewCommunicate(
		text,
		p.Voice,
		p.Rate,
		p.Volume,
		p.Pitch,
		"", // proxy
		p.ConnectTimeout,
		p.ReceiveTimeout,
	)
	if err != nil {
		log.Errorf("EdgeTTS Communicate creation failed: %v", err)
		return nil, err
	}
	//Save MP3
	err = comm.Save(ctx, tmpFile, "")
	if err != nil {
		log.Errorf("EdgeTTS failed to save MP3: %v", err)
		return nil, err
	}
	//MP3 to Opus
	f, err := os.Open(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to open MP3: %v", err)
	}
	defer f.Close()
	pipeReader, pipeWriter := io.Pipe()
	outputChan := make(chan []byte, 1000)
	//Write MP3 data to pipe
	go func() {
		_, _ = io.Copy(pipeWriter, f)
		pipeWriter.Close()
	}()
	mp3Decoder, err := util.CreateAudioDecoder(ctx, pipeReader, outputChan, frameDuration, "mp3")
	if err != nil {
		return nil, fmt.Errorf("Failed to create MP3 decoder: %v", err)
	}
	var opusFrames [][]byte
	done := make(chan struct{})
	go func() {
		for frame := range outputChan {
			opusFrames = append(opusFrames, frame)
		}
		done <- struct{}{}
	}()
	if err := mp3Decoder.Run(startTs); err != nil {
		return nil, fmt.Errorf("MP3 decoding failed: %v", err)
	}
	<-done
	return opusFrames, nil
}

// TextToSpeechStream streaming synthesis, returns Opus frame chan
func (p *EdgeTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	startTs := time.Now().UnixMilli()
	comm, err := communicate.NewCommunicate(
		text,
		p.Voice,
		p.Rate,
		p.Volume,
		p.Pitch,
		"", // proxy
		p.ConnectTimeout,
		p.ReceiveTimeout,
	)
	if err != nil {
		log.Errorf("EdgeTTS Communicate creation failed: %v", err)
		return nil, err
	}

	chunkChan, errChan := comm.Stream(ctx)
	outputChan := make(chan []byte, 100)
	pipeReader, pipeWriter := io.Pipe()
	//MP3 to Opus decoder
	go func() {
		defer func() {
			pipeWriter.Close()
			log.Debugf("EdgeTTS streaming synthesis ends, time taken: %d ms", time.Now().UnixMilli()-startTs)
			if err := <-errChan; err != nil {
				log.Errorf("EdgeTTS streaming composition error: %v", err)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				log.Debugf("EdgeTTS Stream context done, exit")
				return
			default:
				select {
				case chunk, ok := <-chunkChan:
					if !ok {
						log.Debugf("EdgeTTS Stream channel closed, exit")
						return
					}
					if chunk.Type == "audio" {
						_, _ = pipeWriter.Write(chunk.Data)
					}
				}
			}
		}

	}()
	//Start MP3→Opus decoding
	go func() {
		mp3Decoder, err := util.CreateAudioDecoder(ctx, pipeReader, outputChan, frameDuration, "mp3")
		if err != nil {
			log.Errorf("EdgeTTS MP3 decoder creation failed: %v", err)
			return
		}
		if err := mp3Decoder.Run(startTs); err != nil {
			log.Errorf("EdgeTTS MP3 decoding failed: %v", err)
		}
		log.Debugf("EdgeTTS MP3 decoding completed, time taken: %d ms", time.Now().UnixMilli()-startTs)
	}()
	return outputChan, nil
}

// SetVoice sets voice parameters
func (p *EdgeTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && voice != "" {
		p.Voice = voice
		return nil
	}
	return fmt.Errorf("Invalid voice configuration: missing voice")
}

// Close closes the resource (stateless provider, no need to close)
func (p *EdgeTTSProvider) Close() error {
	return nil
}

// IsValid checks whether the resource is valid
func (p *EdgeTTSProvider) IsValid() bool {
	return p != nil
}
