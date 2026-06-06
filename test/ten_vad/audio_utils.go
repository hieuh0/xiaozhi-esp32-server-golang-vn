package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"gopkg.in/hraban/opus.v2"
)

// WavToOpus converts WAV audio data to standard Opus format.
// Returns a slice of Opus frames, each element is one encoded Opus frame.
func WavToOpus(wavData []byte, sampleRate int, channels int, bitRate int) ([][]byte, error) {
	// Create WAV decoder
	wavReader := bytes.NewReader(wavData)
	wavDecoder := wav.NewDecoder(wavReader)
	if !wavDecoder.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file")
	}

	// Read WAV file info
	wavDecoder.ReadInfo()
	format := wavDecoder.Format()
	wavSampleRate := int(format.SampleRate)
	wavChannels := int(format.NumChannels)

	// If provided parameters differ from file parameters, use file parameters
	if sampleRate == 0 {
		sampleRate = wavSampleRate
	}
	if channels == 0 {
		channels = wavChannels
	}

	// Print wavDecoder info
	fmt.Println("WAV format:", format)

	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppAudio)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus encoder: %v", err)
	}

	// Set bit rate
	if bitRate > 0 {
		if err := enc.SetBitrate(bitRate); err != nil {
			return nil, fmt.Errorf("failed to set bit rate: %v", err)
		}
	}

	// Create output frame slice array
	opusFrames := make([][]byte, 0)

	perFrameDuration := 60
	// PCM buffer - Opus frame size (60ms)
	frameSize := sampleRate * perFrameDuration / 1000
	pcmBuffer := make([]int16, frameSize*channels)
	opusBuffer := make([]byte, 1000) // Large enough buffer to store encoded data

	// Read audio buffer
	audioBuf := &audio.IntBuffer{Data: make([]int, frameSize*channels), Format: format}

	fmt.Println("Starting conversion...")
	for {
		// Read WAV data
		n, err := wavDecoder.PCMBuffer(audioBuf)
		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read WAV data: %v", err)
		}

		// Convert int to int16
		for i := 0; i < len(audioBuf.Data); i++ {
			if i < len(pcmBuffer) {
				pcmBuffer[i] = int16(audioBuf.Data[i])
			}
		}

		// Encode to Opus format
		n, err = enc.Encode(pcmBuffer, opusBuffer)
		if err != nil {
			return nil, fmt.Errorf("encoding failed: %v", err)
		}

		// Copy current frame to new slice and append to frame array
		frameData := make([]byte, n)
		copy(frameData, opusBuffer[:n])
		opusFrames = append(opusFrames, frameData)
	}

	return opusFrames, nil
}

func OpusToWav(opusData [][]byte, sampleRate int, channels int, fileName string) ([][]int16, error) {
	opusDecoder, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus decoder: %v", err)
	}

	wavOut, err := os.Create(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create WAV file: %v", err)
	}

	pcmDataList := make([][]int16, 0)
	pcmBuffer := make([]int16, 8192)

	wavEncoder := wav.NewEncoder(wavOut, sampleRate, 16, channels, 1)
	wavBuffer := audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: channels, // Use the passed-in channel count
			SampleRate:  sampleRate,
		},
		SourceBitDepth: 16,
		Data:           make([]int, 8192),
	}

	for _, frame := range opusData {
		n, err := opusDecoder.Decode(frame, pcmBuffer)
		if err != nil {
			return nil, fmt.Errorf("decoding failed: %v", err)
		}
		copyData := make([]int16, len(pcmBuffer[:n]))
		copy(copyData, pcmBuffer[:n])
		//fmt.Println("decode pcmData len: ", len(copyData))
		pcmDataList = append(pcmDataList, copyData)

		//fmt.Println("pcmData len: ", len(copyData))

		// Convert PCM data to int format
		for i := 0; i < len(copyData); i++ {
			wavBuffer.Data = append(wavBuffer.Data, int(copyData[i]))
		}
	}

	// Write WAV file
	err = wavEncoder.Write(&wavBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to write WAV file: %v", err)
	}

	wavEncoder.Close()

	return pcmDataList, nil
}

func Wav2Pcm(wavData []byte, sampleRate int, channels int) ([][]float32, [][]byte, error) {
	// Create WAV decoder
	wavReader := bytes.NewReader(wavData)
	wavDecoder := wav.NewDecoder(wavReader)
	if !wavDecoder.IsValidFile() {
		return nil, nil, fmt.Errorf("invalid WAV file")
	}

	// Read WAV file info
	wavDecoder.ReadInfo()
	format := wavDecoder.Format()

	fmt.Println("WAV format:", format)

	perFrameDuration := 20
	// PCM buffer - 20ms frame size
	frameSize := sampleRate * perFrameDuration / 1000
	pcmBuffer := make([]int16, frameSize*channels)

	// Read audio buffer
	audioBuf := &audio.IntBuffer{Data: make([]int, frameSize*channels), Format: format}

	fmt.Println("Starting conversion...")
	resultFloat32 := make([][]float32, 0)
	result := make([][]byte, 0)
	for {
		// Read WAV data
		n, err := wavDecoder.PCMBuffer(audioBuf)
		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read WAV data: %v", err)
		}

		// Convert int to int16
		for i := 0; i < len(audioBuf.Data); i++ {
			if i < len(pcmBuffer) {
				pcmBuffer[i] = int16(audioBuf.Data[i])
			}
		}

		float32Data := audioBuf.AsFloat32Buffer()
		resultFloat32 = append(resultFloat32, float32Data.Data)

		// Convert int16 array to byte array
		frameBytes := PcmInt16ToByte(pcmBuffer)

		result = append(result, frameBytes)
	}

	return resultFloat32, result, nil
}

func PcmInt16ToByte(pcmData []int16) []byte {
	byteData := make([]byte, len(pcmData)*2)
	for i := 0; i < len(pcmData); i++ {
		byteData[i*2] = byte(pcmData[i] & 0xFF)
		byteData[i*2+1] = byte((pcmData[i] >> 8) & 0xFF)
	}
	return byteData
}
