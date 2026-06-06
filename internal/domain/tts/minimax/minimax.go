package minimax

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"
)

// constant definition
const (
	wsURL = "wss://api.minimaxi.com/ws/v1/t2a_v2"
)

// GlobalWebSocket Dialer
var wsDialer = websocket.Dialer{
	ReadBufferSize:   16384, //16KB read buffer
	WriteBufferSize:  16384, //16KB write buffer
	HandshakeTimeout: 45 * time.Second,
}

// MinimaxTTSProvider Minimax TTS provider
type MinimaxTTSProvider struct {
	APIKey     string
	Model      string
	Voice      string
	Speed      float64
	Volume     float64
	Pitch      int
	SampleRate int
	Bitrate    int
	Format     string
	Channel    int

	//Connection management
	conn      *websocket.Conn
	connMutex sync.RWMutex
	//Send lock to ensure that only one request is using the connection at the same time
	sendMutex sync.Mutex
}

// WebSocket message structure
type minimaxMessage struct {
	Event           string        `json:"event,omitempty"`
	Model           string        `json:"model,omitempty"`
	VoiceSetting    *voiceSetting `json:"voice_setting,omitempty"`
	AudioSetting    *audioSetting `json:"audio_setting,omitempty"`
	ContinuousSound bool          `json:"continuous_sound,omitempty"`
	Text            string        `json:"text,omitempty"`
}

type minimaxResp struct {
	SessionId string            `json:"session_id,omitempty"`
	Event     string            `json:"event,omitempty"`
	TraceId   string            `json:"trace_id,omitempty"`
	Data      *minimaxData      `json:"data,omitempty"`
	IsFinal   bool              `json:"is_final,omitempty"`
	BaseResp  *minimaxBaseResp  `json:"base_resp,omitempty"`
	ExtraInfo *minimaxExtraInfo `json:"extra_info,omitempty"`
}

type minimaxExtraInfo struct {
	AudioLength     int    `json:"audio_length"`
	AudioSampleRate int    `json:"audio_sample_rate"`
	AudioDuration   int    `json:"audio_duration"`
	AudioSize       int    `json:"audio_size"`
	Bitrate         int    `json:"bitrate"`
	AudioFormat     string `json:"audio_format"`
	AudioChannel    int    `json:"audio_channel"`

	UsageCharacters int `json:"usage_characters"`
	WordCount       int `json:"word_count"`
}

type minimaxBaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type voiceSetting struct {
	VoiceID              string  `json:"voice_id"`
	Speed                float64 `json:"speed"`
	Vol                  float64 `json:"vol"`
	Pitch                int     `json:"pitch"`
	EnglishNormalization bool    `json:"english_normalization"`
}

type audioSetting struct {
	SampleRate int    `json:"sample_rate"`
	Bitrate    int    `json:"bitrate"`
	Format     string `json:"format"`
	Channel    int    `json:"channel"`
}

type minimaxData struct {
	Audio string `json:"audio"`
}

// NewMinimaxTTSProvider creates a new Minimax TTS provider
func NewMinimaxTTSProvider(config map[string]interface{}) *MinimaxTTSProvider {
	apiKey, _ := config["api_key"].(string)
	model, _ := config["model"].(string)
	voice, _ := config["voice"].(string)
	speed, _ := config["speed"].(float64)
	volume, _ := config["vol"].(float64)
	if volume == 0 {
		volume, _ = config["volume"].(float64)
	}
	pitch, _ := config["pitch"].(float64)
	sampleRate, _ := config["sample_rate"].(float64)
	bitrate, _ := config["bitrate"].(float64)
	format, _ := config["format"].(string)
	channel, _ := config["channel"].(float64)

	//Set default value
	if model == "" {
		model = "speech-2.8-hd"
	}
	if voice == "" {
		voice = "male-qn-qingse"
	}
	if speed == 0 {
		speed = 1.0
	}
	if volume == 0 {
		volume = 1.0
	}
	if sampleRate == 0 {
		sampleRate = 32000
	}
	if bitrate == 0 {
		bitrate = 128000
	}
	if format == "" {
		format = "mp3"
	}
	if channel == 0 {
		channel = 1
	}

	return &MinimaxTTSProvider{
		APIKey:     apiKey,
		Model:      model,
		Voice:      voice,
		Speed:      speed,
		Volume:     volume,
		Pitch:      int(pitch),
		SampleRate: int(sampleRate),
		Bitrate:    int(bitrate),
		Format:     format,
		Channel:    int(channel),
	}
}

// TextToSpeech one-time synthesis (not supported yet, using streaming implementation)
func (p *MinimaxTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	//Minimax mainly supports streaming, where streaming data can be collected and returned
	outputChan, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}

	var frames [][]byte
	for frame := range outputChan {
		frames = append(frames, frame)
	}

	return frames, nil
}

// TextToSpeechStream streaming speech synthesis implementation
func (p *MinimaxTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	startTs := time.Now().UnixMilli()

	//Use send lock protection to ensure that only one request is using the connection at the same time
	p.sendMutex.Lock()
	//Note: The lock is not released when the function returns, but when the goroutine completes

	//Get a connection (reuse or create)
	conn, err := p.getConnection(ctx)
	if err != nil {
		p.sendMutex.Unlock()
		return nil, fmt.Errorf("Failed to obtain WebSocket connection: %v", err)
	}

	//Create output channel
	outputChan = make(chan []byte, 100)

	//Create a pipeline for audio decoding
	pipeReader, pipeWriter := io.Pipe()

	//Start the audio decoder goroutine
	go func() {
		decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, p.Format, sampleRate)
		if err != nil {
			log.Errorf("Failed to create audio decoder: %v", err)
			pipeReader.Close()
			close(outputChan)
			return
		}

		if err := decoder.Run(startTs); err != nil {
			log.Errorf("Audio decoding failed: %v", err)
		}
	}()

	//Use WaitGroup to wait for the read goroutine to complete
	var wg sync.WaitGroup
	wg.Add(1)

	//Start reading and processing goroutine; the lock is uniformly released by defer in this goroutine, ensuring that it will be released regardless of normal end, error or panic
	go func() {
		defer wg.Done()
		defer p.sendMutex.Unlock()
		defer func() {
			pipeWriter.Close()
			pipeReader.Close()
		}()

		p.processStreamTTS(ctx, conn, text, pipeWriter)
	}()

	//Wait in the background for the goroutine to complete and release the lock
	go func() {
		wg.Wait()
		log.Debugf("Minimax TTS streaming synthesis is completed, taking: %d ms", time.Now().UnixMilli()-startTs)
	}()

	return outputChan, nil
}

// processStreamTTS handles the streaming TTS synthesis process
func (p *MinimaxTTSProvider) processStreamTTS(ctx context.Context, conn *websocket.Conn, text string, pipeWriter *io.PipeWriter) {
	//Send task start message
	startMsg := minimaxMessage{
		Event: "task_start",
		Model: p.Model,
		VoiceSetting: &voiceSetting{
			VoiceID:              p.Voice,
			Speed:                p.Speed,
			Vol:                  p.Volume,
			Pitch:                p.Pitch,
			EnglishNormalization: false,
		},
		AudioSetting: &audioSetting{
			SampleRate: p.SampleRate,
			Bitrate:    p.Bitrate,
			Format:     p.Format,
			Channel:    p.Channel,
		},
		ContinuousSound: false,
	}

	log.Debugf("minimax sends task start message: model=%s, voice=%s, format=%s", p.Model, p.Voice, p.Format)
	if err := p.sendMessage(conn, startMsg); err != nil {
		log.Errorf("Failed to send task start message: %v", err)
		p.clearConnection()
		return
	}

	//Waiting for task start confirmation
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msg, err := p.readMessage(conn)
	if err != nil {
		// Check whether this is a timeout error.
		if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
			log.Errorf("Read task start confirmation timeout (no response received within 10 seconds)")
		} else {
			log.Errorf("Reading task start confirmation failed: %v", err)
		}
		p.clearConnection()
		return
	}

	log.Debugf("Received task start confirmation message: %+v", msg)

	if msg.Event != "task_started" {
		log.Errorf("Task start failed, expected 'task_started', received: event=%s, complete message=%+v", msg.Event, msg)
		if msg.BaseResp != nil && msg.BaseResp.StatusCode != 0 {
			log.Errorf("Error details: status_code=%d, status_msg=%s", msg.BaseResp.StatusCode, msg.BaseResp.StatusMsg)
		}
		p.clearConnection()
		return
	}
	//Reset read timeout
	conn.SetReadDeadline(time.Time{})

	log.Debugf("Task start confirmed successful")

	//Send text message
	continueMsg := minimaxMessage{
		Event: "task_continue",
		Text:  text,
	}

	if err := p.sendMessage(conn, continueMsg); err != nil {
		log.Errorf("Failed to send text message: %v", err)
		p.clearConnection()
		return
	}

	//Read audio data
	chunkCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Debugf("Minimax TTS streaming synthesis cancelled, text: %s", text)
			//Send task end message
			finishMsg := minimaxMessage{Event: "task_finish"}
			p.sendMessage(conn, finishMsg)

			//According to the documentation, the server closes the WebSocket connection after receiving task_finish
			//Try to read the task_finished response (if sent by the server)
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			if finishResp, err := p.readMessage(conn); err == nil {
				log.Debugf("Received task end confirmation: event=%s, complete message=%+v", finishResp.Event, finishResp)
			} else {
				//The connection may have been closed, this is normal behavior
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Debugf("The server has closed the connection (normal behavior)")
					if closeErr, ok := err.(*websocket.CloseError); ok {
						log.Debugf("Close frame details: code=%d, text=%s", closeErr.Code, closeErr.Text)
					}
				} else {
					log.Debugf("Failed to read task completion confirmation: %v", err)
					if closeErr, ok := err.(*websocket.CloseError); ok {
						log.Debugf("Close frame details: code=%d, text=%s", closeErr.Code, closeErr.Text)
					}
				}
			}

			//Clear the connection status because the server has closed the connection
			p.clearConnection()
			return
		default:
		}

		//Set read timeout
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		msg, err := p.readMessage(conn)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("Failed to read WebSocket message: %v", err)
				//Try to get closing frame information
				if closeErr, ok := err.(*websocket.CloseError); ok {
					log.Errorf("WebSocket close frame details: code=%d, text=%s", closeErr.Code, closeErr.Text)
				}
				p.clearConnection()
				return
			}
			//Normal shutdown or read error
			log.Debugf("WebSocket connection closed or read error: %v", err)
			if closeErr, ok := err.(*websocket.CloseError); ok {
				log.Debugf("WebSocket close frame details: code=%d, text=%s", closeErr.Code, closeErr.Text)
			}
			return
		}

		if msg.BaseResp != nil && msg.BaseResp.StatusCode != 0 {
			log.Errorf("BaseResp: status_code=%d, status_msg=%s", msg.BaseResp.StatusCode, msg.BaseResp.StatusMsg)
		}

		//Check for error messages
		if msg.Event == "error" || msg.Event == "task_error" {
			log.Errorf("Received error message: %+v", msg)
			if msg.BaseResp != nil && msg.BaseResp.StatusCode != 0 {
				log.Errorf("Error details: status_code=%d, status_msg=%s", msg.BaseResp.StatusCode, msg.BaseResp.StatusMsg)
			}
			p.clearConnection()
			return
		}

		//Process audio data
		if msg.Data != nil && msg.Data.Audio != "" {
			chunkCount++

			//Convert hex encoded audio data to binary
			audioBytes, err := hex.DecodeString(msg.Data.Audio)
			if err != nil {
				log.Errorf("Failed to decode audio data: %v", err)
				continue
			}

			//Write to the pipeline for processing by the decoder
			if _, err := pipeWriter.Write(audioBytes); err != nil {
				log.Errorf("Failed to write audio data to pipe: %v", err)
				p.clearConnection()
				return
			}
		}

		//Check if completed
		if msg.IsFinal {
			log.Debugf("Received the last audio clip, total %d clips", chunkCount)
			//Send task end message
			finishMsg := minimaxMessage{Event: "task_finish"}
			p.sendMessage(conn, finishMsg)

			//Clear the connection status because the server has closed the connection
			//You need to create a new connection the next time you use it
			p.clearConnection()
			return
		}
	}
}

// getConnection Gets the connection and creates it if it does not exist
func (p *MinimaxTTSProvider) getConnection(ctx context.Context) (*websocket.Conn, error) {
	//Try reading an existing connection first
	p.connMutex.RLock()
	conn := p.conn
	p.connMutex.RUnlock()

	if conn != nil {
		return conn, nil
	}

	//New connection needs to be created
	p.connMutex.Lock()
	defer p.connMutex.Unlock()

	//Double check, maybe other goroutine has already created the connection
	if p.conn != nil {
		return p.conn, nil
	}

	//Create HTTP header
	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))

	//Create new connection
	conn, resp, err := wsDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		if resp != nil {
			log.Errorf("WebSocket connection failed, status code: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("WebSocket connection failed: %v", err)
	}

	//Set message reading limit
	conn.SetReadLimit(1024 * 1024) //1MB maximum message size

	//Set up stay connected
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(1*time.Second))
	})

	//Wait for connection success message
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("Failed to read connection confirmation message: %v", err)
	}

	log.Debugf("Connection confirmation message received (original): %s", string(message))

	var connectMsg minimaxResp
	if err := json.Unmarshal(message, &connectMsg); err != nil {
		conn.Close()
		log.Errorf("Failed to parse connection confirmation message, original message: %s, error: %v", string(message), err)
		return nil, fmt.Errorf("Failed to parse connection confirmation message: %v", err)
	}

	log.Debugf("Connection confirmation message received (after parsing): %+v", connectMsg)

	if connectMsg.Event != "connected_success" {
		conn.Close()
		log.Errorf("Connection failed, expecting 'connected_success', received: %+v", connectMsg)
		return nil, fmt.Errorf("Connection failed, received: %+v", connectMsg)
	}

	p.conn = conn
	log.Infof("Minimax WebSocket connection established")
	return conn, nil
}

// clearConnection clears the connection (used for disconnection and reconnection)
func (p *MinimaxTTSProvider) clearConnection() {
	p.connMutex.Lock()
	defer p.connMutex.Unlock()

	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
		log.Infof("Minimax WebSocket connection has been cleared, waiting for next reconnection")
	}
}

// sendMessage sends JSON message
func (p *MinimaxTTSProvider) sendMessage(conn *websocket.Conn, msg minimaxMessage) error {
	p.connMutex.RLock()
	defer p.connMutex.RUnlock()

	if conn == nil {
		return fmt.Errorf("connection closed")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Failed to serialize message: %v", err)
	}

	log.Debugf("minimax send message: %s", string(data))

	return conn.WriteMessage(websocket.TextMessage, data)
}

// readMessage reads JSON message
func (p *MinimaxTTSProvider) readMessage(conn *websocket.Conn) (*minimaxResp, error) {
	messageType, message, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	_ = messageType
	//log.Debugf("miminimax reads the WebSocket message: type=%d, original content length=%d, content=%sontent length=%d, content=%s", messageType, len(message), string(message))

	var msg minimaxResp
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Errorf("Failed to parse message, original message: %s, error: %v", string(message), err)
		return nil, fmt.Errorf("Failed to parse message: %v", err)
	}

	return &msg, nil
}

// SetVoice sets voice parameters
func (p *MinimaxTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	return nil
}

// Close closes the resource and releases the connection
func (p *MinimaxTTSProvider) Close() error {
	p.clearConnection()
	return nil
}

// IsValid checks whether the resource is valid
func (p *MinimaxTTSProvider) IsValid() bool {
	p.connMutex.RLock()
	conn := p.conn
	p.connMutex.RUnlock()

	return conn != nil
}
