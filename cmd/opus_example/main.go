package main

import (
	"fmt"
	"os"

	"github.com/hraban/opus"
)

func main() {
	// Audio parameters.
	channels := 1
	sampleRate := 16000 // 16kHz
	fmt.Printf("channels: %d, sample rate: %d Hz\n", channels, sampleRate)

	// Create an encoder for low-latency VoIP audio.
	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
	if err != nil {
		fmt.Printf("failed to create encoder: %v\n", err)
		os.Exit(1)
	}

	// Set the bit rate to 16 kbps.
	if err = enc.SetBitrate(16000); err != nil {
		fmt.Printf("failed to set bit rate: %v\n", err)
		os.Exit(1)
	}

	// Set complexity from 0 to 10; higher values improve quality but use more CPU.
	if err = enc.SetComplexity(5); err != nil {
		fmt.Printf("failed to set complexity: %v\n", err)
		os.Exit(1)
	}

	// Generate 20 ms of test PCM data (320 samples at 16 kHz).
	frameSize := 320
	pcm := make([]int16, frameSize*channels)

	// Generate a simple sine wave for testing.
	for i := 0; i < frameSize; i++ {
		// Use a frequency of approximately 440 Hz.
		value := int16(10000.0 * float64(i%36) / 36.0)
		pcm[i] = value
	}

	// Buffer for encoded data.
	data := make([]byte, 1000)

	// Encode PCM data as Opus.
	n, err := enc.Encode(pcm, data)
	if err != nil {
		fmt.Printf("encoding failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("encoded %d samples into %d bytes of Opus data, compression ratio: %.2f%%\n",
		frameSize*channels, n, float64(n)/float64(frameSize*channels*2)*100)

	// Create a decoder for the decode test.
	dec, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		fmt.Printf("failed to create decoder: %v\n", err)
		os.Exit(1)
	}

	// Buffer for decoded PCM data.
	decodedPCM := make([]int16, frameSize*channels)

	// Decode Opus data to PCM.
	samplesDecoded, err := dec.Decode(data[:n], decodedPCM)
	if err != nil {
		fmt.Printf("decoding failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("decoded %d bytes of Opus data into %d samples\n", n, samplesDecoded)

	// Calculate the difference between original and decoded PCM.
	var sumDiff int64
	for i := 0; i < frameSize; i++ {
		diff := int64(pcm[i]) - int64(decodedPCM[i])
		if diff < 0 {
			diff = -diff
		}
		sumDiff += diff
	}
	avgDiff := float64(sumDiff) / float64(frameSize)

	fmt.Printf("average difference between original and decoded PCM: %.2f\n", avgDiff)
	fmt.Println("Opus encode/decode example completed")
}
