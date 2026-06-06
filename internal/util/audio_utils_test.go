package util

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"

	"gopkg.in/hraban/opus.v2"
)

func TestAudioDecoderRunOggOpusPassThrough(t *testing.T) {
	expectedPackets := [][]byte{
		{0x08, 0xAA, 0xBB},
		{0x08, 0xCC, 0xDD, 0xEE},
	}
	oggData := WrapOggOpusPackets(expectedPackets, 16000, 1, 320)

	outputChan := make(chan []byte, len(expectedPackets))
	decoder, err := CreateAudioDecoderWithSampleRate(context.Background(), newReadCloserWrapper(oggData), outputChan, 20, "ogg_opus", 16000)
	if err != nil {
		t.Fatalf("CreateAudioDecoderWithSampleRate failed: %v", err)
	}

	if err := decoder.Run(time.Now().UnixMilli()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var actualPackets [][]byte
	for packet := range outputChan {
		actualPackets = append(actualPackets, packet)
	}

	if !reflect.DeepEqual(actualPackets, expectedPackets) {
		t.Fatalf("passthrough packets differ, actual=%x expected=%x", actualPackets, expectedPackets)
	}
}

func TestGetAudioFormatByMimeTypeSupportsOggOpus(t *testing.T) {
	if got := GetAudioFormatByMimeType("audio/ogg"); got != "ogg_opus" {
		t.Fatalf("audio/ogg should map to ogg_opus, got %s", got)
	}
	if got := GetAudioFormatByMimeType("application/ogg"); got != "ogg_opus" {
		t.Fatalf("application/ogg should map to ogg_opus, got %s", got)
	}
	if got := GetAudioFormatByMimeType("audio/opus"); got != "opus" {
		t.Fatalf("audio/opus should map to opus, got %s", got)
	}
}

func TestAudioDecoderRunOggOpusRepacketizeTo60ms(t *testing.T) {
	sampleRate := 16000
	packets := makeTestOpusPackets(t, sampleRate, 1, 20, 120)
	oggData := WrapOggOpusPackets(packets, sampleRate, 1, sampleRate*20/1000)

	outputChan := make(chan []byte, 16)
	decoder, err := CreateAudioDecoderWithSampleRate(context.Background(), newReadCloserWrapper(oggData), outputChan, 60, "ogg_opus", sampleRate)
	if err != nil {
		t.Fatalf("CreateAudioDecoderWithSampleRate failed: %v", err)
	}

	if err := decoder.Run(time.Now().UnixMilli()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var durations []int
	for packet := range outputChan {
		dur, err := opusPacketDurationMs(packet, sampleRate)
		if err != nil {
			t.Fatalf("failed to parse output packet duration: %v", err)
		}
		durations = append(durations, dur)
	}

	expected := []int{60, 60}
	if !reflect.DeepEqual(durations, expected) {
		t.Fatalf("repacketized frame durations differ, actual=%v expected=%v", durations, expected)
	}
}

func makeTestOpusPackets(t *testing.T, sampleRate int, channels int, frameDurationMs int, totalDurationMs int) [][]byte {
	t.Helper()

	frameSize := sampleRate * frameDurationMs / 1000
	totalSamples := sampleRate * totalDurationMs / 1000
	if frameSize <= 0 || totalSamples <= 0 {
		t.Fatalf("invalid test parameters: frameSize=%d totalSamples=%d", frameSize, totalSamples)
	}

	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppAudio)
	if err != nil {
		t.Fatalf("failed to create test Opus encoder: %v", err)
	}

	pcm := make([]int16, totalSamples*channels)
	for i := 0; i < totalSamples; i++ {
		sample := int16(math.Sin(2*math.Pi*440*float64(i)/float64(sampleRate)) * 12000)
		for ch := 0; ch < channels; ch++ {
			pcm[i*channels+ch] = sample
		}
	}

	opusBuf := make([]byte, 1000)
	packets := make([][]byte, 0, totalSamples/frameSize)
	for offset := 0; offset < len(pcm); offset += frameSize * channels {
		end := offset + frameSize*channels
		if end > len(pcm) {
			end = len(pcm)
		}
		frame := make([]int16, frameSize*channels)
		copy(frame, pcm[offset:end])
		n, err := enc.Encode(frame, opusBuf)
		if err != nil {
			t.Fatalf("failed to encode test Opus packet: %v", err)
		}
		packet := make([]byte, n)
		copy(packet, opusBuf[:n])
		packets = append(packets, packet)
	}
	return packets
}
