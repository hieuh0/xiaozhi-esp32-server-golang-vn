package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"

	"xiaozhi-esp32-server-golang/internal/domain/asr/funasr"
)

// readWavFile reads WAV files and converts them to PCM []float32 data
func readWavFile(filePath string) ([]float32, error) {
	//Open WAV file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to open WAV file: %v", err)
	}
	defer file.Close()

	//Create WAV decoder
	wavDecoder := wav.NewDecoder(file)
	if !wavDecoder.IsValidFile() {
		return nil, fmt.Errorf("Invalid WAV file")
	}

	//Read WAV file information
	wavDecoder.ReadInfo()
	format := wavDecoder.Format()

	fmt.Printf("WAV format: sampling rate=%dHz, number of channels=%d\n",
		int(format.SampleRate), format.NumChannels)

	//Read all PCM data
	var allPcmData []float32

	//Use 20ms frame size as buffer
	perFrameDuration := 20
	frameSize := int(format.SampleRate) * perFrameDuration / 1000
	audioBuf := &audio.IntBuffer{
		Format:         format,
		SourceBitDepth: 16,
		Data:           make([]int, frameSize*format.NumChannels),
	}

	fmt.Printf("Use frame size: %d sampling points (%.1fms)\n", frameSize, float64(perFrameDuration))
	fmt.Println("Start reading WAV data...")

	for {
		//Read WAV data
		n, err := wavDecoder.PCMBuffer(audioBuf)
		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to read WAV data: %v", err)
		}

		//Convert int data to float32 (range -1.0 to 1.0)
		for i := 0; i < n; i++ {
			//Convert int to float32 in the range [-32768, 32767] to [-1.0, 1.0]
			floatSample := float32(audioBuf.Data[i]) / 32767.0
			allPcmData = append(allPcmData, floatSample)
		}
	}

	fmt.Printf("Successfully read WAV file, total sampling points: %d, duration: %.2f seconds \n",
		len(allPcmData), float64(len(allPcmData))/float64(format.SampleRate))

	return allPcmData, nil
}

func main() {
	//Define command line parameters
	var (
		host = flag.String("host", "192.168.208.214", "FunASR server IP address")
		port = flag.String("port", "10096", "FunASR server port")
		mode = flag.String("mode", "offline", "Recognition mode (online/offline)")
		file = flag.String("file", "test.wav", "Path to the WAV file to recognize")
	)

	//Parse command line arguments
	flag.Parse()

	//Show instructions
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./streaming_example [options]")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("\n example:")
		fmt.Println("  ./streaming_example -host=192.168.1.100 -port=10095 -file=audio.wav")
		fmt.Println("  ./streaming_example -mode=online -file=test.wav")
		return
	}

	config := funasr.FunasrConfig{
		Host:          *host,
		Port:          *port,
		Mode:          *mode,
		SampleRate:    16000,
		ChunkSize:     []int{5, 10, 5},
		ChunkInterval: 10,
		Timeout:       30,
		AutoEnd:       false,
	}

	//Create an ASR instance using configuration
	asr, err := funasr.NewFunasr(config)
	if err != nil {
		fmt.Printf("Failed to create ASR instance: %v\n", err)
		return
	}

	fmt.Printf("Target server: %s:%s, mode: %s\n", config.Host, config.Port, config.Mode)

	//Audio file path specified using command line arguments
	audioFilePath := *file

	//Check if the audio file exists
	if _, err := os.Stat(audioFilePath); os.IsNotExist(err) {
		fmt.Printf("Audio file %s does not exist \n", audioFilePath)
		fmt.Println("Please provide a valid audio file path")
		return
	}

	//Read WAV files and convert to PCM data
	pcmData, err := readWavFile(audioFilePath)
	if err != nil {
		fmt.Printf("Failed to read WAV file: %v\n", err)
		return
	}

	//Perform identification
	result, err := asr.Process(pcmData)
	if err != nil {
		fmt.Printf("Recognition failed: %v\n", err)
		return
	}

	//Format and print the results
	fmt.Println("Recognition results:")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println(result)
	fmt.Println(strings.Repeat("-", 40))
}
