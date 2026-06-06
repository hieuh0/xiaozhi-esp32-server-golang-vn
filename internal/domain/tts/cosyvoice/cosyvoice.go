package cosyvoice

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/data/audio"
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
			Timeout:   30 * time.Second,
		}
	})
	return httpClient
}

// CosyVoiceTTSProvider CosyVoice TTS Provider
type CosyVoiceTTSProvider struct {
	APIURL        string
	SpeakerID     string
	FrameDuration int
	TargetSR      int
	AudioFormat   string
	InstructText  string
}

// response structure
type cosyVoiceResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []byte `json:"data"`
}

// NewCosyVoiceTTSProvider Create a new CosyVoice TTS provider
func NewCosyVoiceTTSProvider(config map[string]interface{}) *CosyVoiceTTSProvider {
	apiURL, _ := config["api_url"].(string)
	speakerID, _ := config["spk_id"].(string)
	frameDuration, _ := config["frame_duration"].(float64)
	targetSR, _ := config["target_sr"].(float64)
	audioFormat, _ := config["audio_format"].(string)
	instructText, _ := config["instruct_text"].(string)

	//Set default value
	if apiURL == "" {
		apiURL = "https://tts.linkerai.cn/tts"
	}
	if speakerID == "" {
		speakerID = "OUeAo1mhq6IBExi"
	}
	if frameDuration == 0 {
		frameDuration = audio.FrameDuration
	}
	if targetSR == 0 {
		targetSR = audio.SampleRate
	}
	if audioFormat == "" {
		audioFormat = "mp3"
	}

	return &CosyVoiceTTSProvider{
		APIURL:        apiURL,
		SpeakerID:     speakerID,
		FrameDuration: int(frameDuration),
		TargetSR:      int(targetSR),
		AudioFormat:   audioFormat,
		InstructText:  instructText,
	}
}

// TextToSpeech converts text to speech, returning audio frame data and errors
func (p *CosyVoiceTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	//Build query parameters
	params := url.Values{}
	params.Add("tts_text", text)
	params.Add("spk_id", p.SpeakerID)
	params.Add("frame_durition", fmt.Sprintf("%d", p.FrameDuration))
	params.Add("stream", "true") //Streaming requests
	params.Add("target_sr", fmt.Sprintf("%d", p.TargetSR))
	params.Add("audio_format", p.AudioFormat)

	startTs := time.Now().UnixMilli()

	//Build full URL
	requestURL := fmt.Sprintf("%s?%s", p.APIURL, params.Encode())

	//Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Create request failed: %v", err)
	}

	req.Header.Set("Accept", "application/json")

	//Use connection pool to send requests
	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	//Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response: %v", err)
	}

	//Check response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	//Check response content type and content length
	// contentType := resp.Header.Get("Content-Type")
	contentLength := resp.ContentLength

	//Record response length to log
	log.Debugf("Received TTS response, Content-Length: %d", contentLength)

	//Determine whether Content-Length is reasonable
	if contentLength == 0 {
		log.Errorf("API returns empty response with Content-Length of 0")
		return nil, fmt.Errorf("API returns empty response with Content-Length of 0")
	}

	//MP3 file header requires at least 100 bytes for normal parsing
	//-1 means unknown length (e.g. chunked transfer)
	if contentLength > 0 && contentLength < 100 {
		log.Errorf("The response returned by the API is too small to parse as MP3: %d bytes", contentLength)
		return nil, fmt.Errorf("The response returned by the API is too small to parse as MP3: %d bytes", contentLength)
	}

	//Convert to Opus frame
	if p.AudioFormat == "mp3" {
		//Create a pipeline
		doneChan := make(chan struct{})
		outputChan := make(chan []byte, 1000)

		//Create MP3 decoder
		mp3Decoder, err := util.CreateAudioDecoder(ctx, io.NopCloser(bytes.NewReader(body)), outputChan, frameDuration, p.AudioFormat)
		if err != nil {
			close(doneChan)
			return nil, fmt.Errorf("Failed to create MP3 decoder: %v", err)
		}
		//Start decoding process
		go func() {
			if err := mp3Decoder.Run(startTs); err != nil {
				log.Errorf("MP3 decoding failed: %v", err)
			}
		}()

		//Collect all Opus frames
		var opusFrames [][]byte
		for frame := range outputChan {
			opusFrames = append(opusFrames, frame)
		}

		return opusFrames, nil
	}

	return nil, fmt.Errorf("Unsupported audio format: %s", p.AudioFormat)
}

// TextToSpeechStream streaming speech synthesis implementation
func (p *CosyVoiceTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	//Build query parameters
	params := url.Values{}
	params.Add("tts_text", text)
	params.Add("spk_id", p.SpeakerID)
	params.Add("frame_durition", fmt.Sprintf("%d", frameDuration))
	params.Add("stream", "true") //Streaming requests
	params.Add("target_sr", fmt.Sprintf("%d", sampleRate))
	params.Add("audio_format", p.AudioFormat)

	startTs := time.Now().UnixMilli()

	//Build full URL
	requestURL := fmt.Sprintf("%s?%s", p.APIURL, params.Encode())

	//Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Create request failed: %v", err)
	}

	req.Header.Set("Accept", "application/json")

	//Create a client using a connection pool
	client := getHTTPClient()

	//Create output channel
	outputChan = make(chan []byte, 100)
	//Start goroutine to process streaming response
	go func() {
		decoderStarted := false
		defer func() {
			if !decoderStarted {
				close(outputChan)
			}
		}()

		//Send request
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Failed to send request: %v", err)
			return
		}
		defer func() {
			resp.Body.Close()
		}()

		//Check response status code
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("API request failed, status code: %d, response: %s", resp.StatusCode, string(body))
			return
		}

		//Check response content type and content length
		// contentType := resp.Header.Get("Content-Type")
		contentLength := resp.ContentLength

		//Record response length to log
		log.Debugf("Received TTS response, Content-Length: %d", contentLength)

		//Determine whether Content-Length is reasonable
		if contentLength == 0 {
			log.Errorf("API returns empty response with Content-Length of 0")
			return
		}

		//MP3 file header requires at least 100 bytes for normal parsing
		//-1 means unknown length (e.g. chunked transfer)
		if contentLength > 0 && contentLength < 100 {
			log.Errorf("The response returned by the API is too small to parse as MP3: %d bytes", contentLength)
			return
		}

		//Handle streaming responses based on audio format
		if p.AudioFormat == "mp3" {
			//Create MP3 decoder, passing context instead of done channel
			mp3Decoder, err := util.CreateAudioDecoder(ctx, resp.Body, outputChan, frameDuration, p.AudioFormat)
			if err != nil {
				log.Errorf("Failed to create MP3 decoder: %v", err)
				return
			}

			//Start decoding process
			decoderStarted = true
			if err := mp3Decoder.Run(startTs); err != nil {
				log.Errorf("MP3 decoding failed: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				log.Debugf("TTS streaming synthesis cancelled, text: %s", text)
				return
			default:
				log.Infof("tts time consuming: from input to getting MP3 data: %d ms", time.Now().UnixMilli()-startTs)

			}
		} else {
			log.Errorf("Currently only streaming composition in MP3 format is supported")
		}
	}()

	return outputChan, nil
}

// SetVoice sets voice parameters
func (p *CosyVoiceTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if spkID, ok := voiceConfig["spk_id"].(string); ok && spkID != "" {
		p.SpeakerID = spkID
		return nil
	}
	return fmt.Errorf("Invalid sound configuration: missing spk_id")
}

// Close closes the resource (stateless provider, no need to close)
func (p *CosyVoiceTTSProvider) Close() error {
	return nil
}

// IsValid checks whether the resource is valid
func (p *CosyVoiceTTSProvider) IsValid() bool {
	return p != nil
}
