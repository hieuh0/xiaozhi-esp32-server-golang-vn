package manager

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	opus "gopkg.in/hraban/opus.v2"

	"xiaozhi-esp32-server-golang/internal/domain/tts"
	"xiaozhi-esp32-server-golang/internal/pool"
	log "xiaozhi-esp32-server-golang/logger"
)

const (
	previewSampleRate    = 24000
	previewChannels      = 1
	previewFrameDuration = 60
	previewTimeout       = 15 * time.Second
)

// RunTTSAudioPreview synthesizes speech using the given TTS config, decodes the
// Opus frames returned by the provider to PCM-16, and wraps the result in a
// standard WAV file suitable for browser playback.
func RunTTSAudioPreview(cfg map[string]interface{}, text string) (wavBytes []byte, firstPacketMs int64, err error) {
	if text == "" {
		text = DefaultTestText
	}

	wrapper, err := pool.Acquire[tts.TTSProvider]("tts", "_preview", cfg)
	if err != nil {
		return nil, 0, fmt.Errorf("TTS init failed: %w", err)
	}
	defer pool.Release(wrapper)

	provider := wrapper.GetProvider()
	ctx, cancel := context.WithTimeout(context.Background(), previewTimeout)
	defer cancel()

	outputChan, err := provider.TextToSpeechStream(ctx, text, previewSampleRate, previewChannels, previewFrameDuration)
	if err != nil {
		return nil, 0, fmt.Errorf("TTS stream failed: %w", err)
	}

	dec, err := opus.NewDecoder(previewSampleRate, previewChannels)
	if err != nil {
		return nil, 0, fmt.Errorf("opus decoder: %w", err)
	}

	frameSize := previewSampleRate * previewFrameDuration / 1000 // samples per frame
	pcmBuf := make([]int16, frameSize*previewChannels)
	var allPCM []int16
	t0 := time.Now()
	firstPacketMs = -1

	for chunk := range outputChan {
		if len(chunk) == 0 {
			continue
		}
		if firstPacketMs < 0 {
			firstPacketMs = time.Since(t0).Milliseconds()
		}
		n, decErr := dec.Decode(chunk, pcmBuf)
		if decErr != nil {
			log.Debugf("[tts_preview] opus decode error: %v", decErr)
			continue
		}
		allPCM = append(allPCM, pcmBuf[:n*previewChannels]...)
	}

	if firstPacketMs < 0 {
		firstPacketMs = time.Since(t0).Milliseconds()
	}
	if len(allPCM) == 0 {
		return nil, firstPacketMs, fmt.Errorf("no audio generated")
	}

	return buildWAV(allPCM, previewSampleRate, previewChannels), firstPacketMs, nil
}

// buildWAV wraps PCM-16 samples in a standard RIFF/WAV header.
func buildWAV(pcm []int16, sampleRate, channels int) []byte {
	dataSize := len(pcm) * 2 // 16-bit = 2 bytes per sample
	buf := make([]byte, 44+dataSize)

	copy(buf[0:], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:], uint32(36+dataSize))
	copy(buf[8:], "WAVE")

	copy(buf[12:], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:], 16)
	binary.LittleEndian.PutUint16(buf[20:], 1) // PCM
	binary.LittleEndian.PutUint16(buf[22:], uint16(channels))
	binary.LittleEndian.PutUint32(buf[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(buf[28:], uint32(sampleRate*channels*2))
	binary.LittleEndian.PutUint16(buf[32:], uint16(channels*2))
	binary.LittleEndian.PutUint16(buf[34:], 16)

	copy(buf[36:], "data")
	binary.LittleEndian.PutUint32(buf[40:], uint32(dataSize))

	for i, s := range pcm {
		binary.LittleEndian.PutUint16(buf[44+i*2:], uint16(s))
	}
	return buf
}
