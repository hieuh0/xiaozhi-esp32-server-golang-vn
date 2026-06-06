package play_music

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"bytes"

	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"
)

// Global HTTP client, implementing connection pooling
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

// Get an HTTP client configured with a connection pool
func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		httpClient = &http.Client{
			Transport: transport,
			//Timeout:   30 * time.Second,
		}
	})
	return httpClient
}

// PlayMusicStream plays music from URL and returns audio stream channel
// frameDuration: duration of each frame (milliseconds), default 20ms
// audioFormat: audio format, supports "mp3"
func PlayMusicStream(ctx context.Context, url string, sampleRate int, frameDuration int, audioFormat string) (outputChan chan []byte, err error) {
	//Parameter checksum default value settings
	if frameDuration <= 0 {
		frameDuration = 20 //Default 20ms frame duration
	}
	if audioFormat == "" {
		audioFormat = "mp3" //Default MP3 format
	}

	startTs := time.Now().UnixMilli()

	//Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Create request failed: %v", err)
	}

	req.Header.Set("Accept", "audio/*")
	req.Header.Set("User-Agent", "MusicPlayer/1.0")

	//Create a client using a connection pool
	client := getHTTPClient()

	//Create output channel
	outputChan = make(chan []byte, 100)

	//Start goroutine to process streaming response
	go func() {
		//Send request
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Failed to send request: %v", err)
			close(outputChan)
			return
		}
		defer func() {
			resp.Body.Close()
		}()

		//Check response status code
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("API request failed, status code: %d, response: %s", resp.StatusCode, string(body))
			close(outputChan)
			return
		}

		//Check response content type and content length
		contentLength := resp.ContentLength

		//Record response length to log
		log.Debugf("Received music streaming response, Content-Length: %d", contentLength)

		//Determine whether Content-Length is reasonable
		if contentLength == 0 {
			log.Errorf("Music stream returns empty response, Content-Length is 0")
			close(outputChan)
			return
		}

		//MP3 file header requires at least 100 bytes for normal parsing
		//-1 means unknown length (e.g. chunked transfer)
		if contentLength > 0 && contentLength < 100 {
			log.Errorf("Music stream response too small to parse to MP3: %d bytes", contentLength)
			close(outputChan)
			return
		}

		log.Infof("Start playing music: %s", url)

		//Handle streaming responses based on audio format
		if audioFormat == "mp3" {
			//Create MP3 decoder, passing context instead of done channel
			mp3Decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, resp.Body, outputChan, frameDuration, audioFormat, sampleRate)
			if err != nil {
				log.Errorf("Failed to create MP3 decoder: %v", err)
				close(outputChan)
				return
			}

			//Start decoding process
			if err := mp3Decoder.Run(startTs); err != nil {
				log.Errorf("MP3 decoding failed: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				log.Debugf("Music playback canceled, URL: %s", url)
				return
			default:
				log.Infof("Time taken to complete music playback: %d ms", time.Now().UnixMilli()-startTs)
			}
		} else {
			log.Errorf("Currently only supports streaming playback in MP3 format, incoming format: %s", audioFormat)
			close(outputChan)
		}
	}()

	return outputChan, nil
}

func PlayMusicFromAudioData(ctx context.Context, audioData []byte, sampleRate int, frameDuration int, audioFormat string) (outputChan chan []byte, err error) {
	//Parameter checksum default value settings
	if frameDuration <= 0 {
		frameDuration = 20 //Default 20ms frame duration
	}
	if audioFormat == "" {
		audioFormat = "mp3" //Default MP3 format
	}

	//Add debugging information
	log.Debugf("PlayMusicFromAudioData: Audio data length=%d bytes, sampling rate=%d, frame duration=%dms, format=%s",
		len(audioData), sampleRate, frameDuration, audioFormat)

	//Check if the audio data is empty
	if len(audioData) == 0 {
		log.Errorf("The audio data is empty and cannot be played")
		return nil, fmt.Errorf("Audio data is empty")
	}

	startTs := time.Now().UnixMilli()

	//Create output channel
	outputChan = make(chan []byte, 100)

	//Start goroutine to process streaming response
	go func() {
		//Create an io.ReadCloser from audioData
		audioReader := io.NopCloser(bytes.NewReader(audioData))

		//Handle streaming responses based on audio format
		if audioFormat == "mp3" {
			//Create MP3 decoder, passing context instead of done channel
			mp3Decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, audioReader, outputChan, frameDuration, audioFormat, sampleRate)
			if err != nil {
				log.Errorf("Failed to create MP3 decoder: %v", err)
				return
			}

			//Start decoding process
			if err := mp3Decoder.Run(startTs); err != nil {
				log.Errorf("MP3 decoding failed: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				log.Debugf("Music playback canceled")
				return
			default:
				log.Infof("Time taken to complete music playback: %d ms", time.Now().UnixMilli()-startTs)
			}
		} else {
			log.Errorf("Currently only supports streaming playback in MP3 format, incoming format: %s", audioFormat)
		}
	}()

	return outputChan, nil
}

func PlayMusicFromPipe(ctx context.Context, pipeReader *io.PipeReader, sampleRate int, frameDuration int, audioFormat string) (outputChan chan []byte, err error) {
	//Parameter checksum default value settings
	if frameDuration <= 0 {
		frameDuration = 20 //Default 20ms frame duration
	}
	if audioFormat == "" {
		audioFormat = "mp3" //Default MP3 format
	}

	//Add debugging information
	log.Debugf("PlayMusicFromPipe: Sampling rate=%d, Frame duration=%dms, Format=%s",
		sampleRate, frameDuration, audioFormat)

	startTs := time.Now().UnixMilli()

	//Create output channel
	outputChan = make(chan []byte, 100)

	//Start goroutine to process streaming response
	go func() {
		//Handle streaming responses based on audio format
		if audioFormat == "mp3" {
			//Create MP3 decoder, passing context instead of done channel
			mp3Decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, audioFormat, sampleRate)
			if err != nil {
				log.Errorf("Failed to create MP3 decoder: %v", err)
				return
			}

			//Start decoding process
			if err := mp3Decoder.Run(startTs); err != nil {
				log.Errorf("MP3 decoding failed: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				log.Debugf("Music playback canceled")
				return
			default:
				log.Infof("Time taken to complete music playback: %d ms", time.Now().UnixMilli()-startTs)
			}
		} else {
			log.Errorf("Currently only supports streaming playback in MP3 format, incoming format: %s", audioFormat)
		}
	}()

	return outputChan, nil
}
