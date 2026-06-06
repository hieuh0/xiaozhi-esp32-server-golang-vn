package edge_offline

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gopxl/beep"
	"github.com/gorilla/websocket"
)

// EdgeOfflineTTSProvider WebSocket TTS provider
type EdgeOfflineTTSProvider struct {
	ServerURL        string
	Timeout          time.Duration
	HandshakeTimeout time.Duration

	//Connection management
	conn      *websocket.Conn
	connMutex sync.RWMutex
	//Send lock to ensure that only one request is using the connection at the same time
	sendMutex sync.Mutex
}

// NewEdgeOfflineTTSProvider Creates a new Edge Offline TTS provider
func NewEdgeOfflineTTSProvider(config map[string]interface{}) *EdgeOfflineTTSProvider {
	serverURL, _ := config["server_url"].(string)
	timeout, _ := config["timeout"].(float64)
	handshakeTimeout, _ := config["handshake_timeout"].(float64)

	//Set default value
	if serverURL == "" {
		serverURL = "ws://localhost:8080/tts"
	}
	if timeout == 0 {
		timeout = 30 //Default 30 seconds timeout
	}
	if handshakeTimeout == 0 {
		handshakeTimeout = 10 //Default 10 seconds handshake timeout
	}

	return &EdgeOfflineTTSProvider{
		ServerURL:        serverURL,
		Timeout:          time.Duration(timeout) * time.Second,
		HandshakeTimeout: time.Duration(handshakeTimeout) * time.Second,
	}
}

// getConnection Gets the connection and creates it if it does not exist
func (p *EdgeOfflineTTSProvider) getConnection(ctx context.Context) (*websocket.Conn, error) {
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

	//Create new connection
	dialer := &websocket.Dialer{
		HandshakeTimeout: p.HandshakeTimeout,
	}
	conn, _, err := dialer.DialContext(ctx, p.ServerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("WebSocket connection failed: %v", err)
	}

	p.conn = conn
	log.Infof("WebSocket connection established")
	return conn, nil
}

// clearConnection clears the connection (used for disconnection and reconnection)
func (p *EdgeOfflineTTSProvider) clearConnection() {
	p.connMutex.Lock()
	defer p.connMutex.Unlock()

	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
		log.Infof("WebSocket connection has been cleared, waiting for next reconnection")
	}
}

// writeMessage securely writes a message to a WebSocket connection
func (p *EdgeOfflineTTSProvider) writeMessage(conn *websocket.Conn, messageType int, data []byte) error {
	//Use read locks to protect connection write operations to prevent data confusion caused by concurrent writes
	p.connMutex.RLock()
	defer p.connMutex.RUnlock()

	//Check if the connection is valid
	if conn == nil {
		return fmt.Errorf("connection closed")
	}

	return conn.WriteMessage(messageType, data)
}

// TextToSpeech converts text to speech and returns audio frame data
func (p *EdgeOfflineTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	var frames [][]byte

	//Use send lock protection to ensure that only one request is using the connection at the same time
	p.sendMutex.Lock()
	//Note: The lock is not released when the function returns, but when the goroutine completes

	//Get a connection (reuse or create)
	conn, err := p.getConnection(ctx)
	if err != nil {
		p.sendMutex.Unlock() //Release the lock immediately when acquiring the connection fails
		return nil, err
	}

	//Send text (using protected write method)
	err = p.writeMessage(conn, websocket.TextMessage, []byte(text))
	if err != nil {
		//Failed to send, clear the connection, and automatically reconnect the next time you use it.
		log.Errorf("Failed to send text: %v, clear the connection", err)
		p.clearConnection()
		p.sendMutex.Unlock() //Release the lock immediately when sending fails
		return nil, fmt.Errorf("Failed to send text: %v", err)
	}

	//Create a pipe for audio data transfer
	pipeReader, pipeWriter := io.Pipe()
	outputChan := make(chan []byte, 1000)
	startTs := time.Now().UnixMilli()

	//Create audio decoder
	audioDecoder, err := util.CreateAudioDecoder(ctx, pipeReader, outputChan, frameDuration, "mp3")
	if err != nil {
		pipeReader.Close()
		p.sendMutex.Unlock() //Release the lock immediately when creating the decoder fails
		return nil, fmt.Errorf("Failed to create audio decoder: %v", err)
	}

	decoderDone := make(chan struct{})
	go func() {
		defer close(decoderDone)
		if err := audioDecoder.Run(startTs); err != nil {
			log.Errorf("Audio decoding failed: %v", err)
		}
	}()

	//Use WaitGroup to wait for the read goroutine to complete
	var wg sync.WaitGroup
	wg.Add(1)

	//Receive WebSocket data and write to the pipe; the lock is released by defer in this goroutine, ensuring that it will be released regardless of normal end, error or panic
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		defer p.sendMutex.Unlock()
		defer close(done)
		defer pipeWriter.Close()

		for {
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				log.Errorf("Failed to read WebSocket message: %v, clear the connection", err)
				//If the connection is disconnected, clear the connection and automatically reconnect the next time you use it.
				p.clearConnection()
				return
			}

			if messageType == websocket.BinaryMessage {
				if _, err := pipeWriter.Write(data); err != nil {
					log.Errorf("Failed to write audio data: %v", err)
					return
				}
			}
		}
	}()

	//Collect all Opus frames
	collectorDone := make(chan struct{})
	go func() {
		for frame := range outputChan {
			frames = append(frames, frame)
		}
		close(collectorDone)
	}()

	//Wait for completion or timeout
	select {
	case <-ctx.Done():
		_ = pipeWriter.CloseWithError(ctx.Err())
		p.clearConnection()
		<-decoderDone
		<-collectorDone
		return nil, fmt.Errorf("TTS synthesis timed out or was canceled")
	case <-done:
		<-decoderDone
		<-collectorDone
		return frames, nil
	}
}

// TextToSpeechStream streaming speech synthesis
func (p *EdgeOfflineTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	outputChan := make(chan []byte, 100)

	go func() {
		//Use send lock protection to ensure that only one request is using the connection at the same time
		p.sendMutex.Lock()

		//Get a connection (reuse or create)
		conn, err := p.getConnection(ctx)
		if err != nil {
			p.sendMutex.Unlock()
			close(outputChan)
			log.Errorf("Failed to obtain WebSocket connection: %v", err)
			return
		}

		//Send text (using protected write method)
		err = p.writeMessage(conn, websocket.TextMessage, []byte(text))
		if err != nil {
			p.sendMutex.Unlock()
			close(outputChan)
			log.Errorf("Failed to send text: %v, clear the connection", err)
			//Failed to send, clear the connection, and automatically reconnect the next time you use it.
			p.clearConnection()
			return
		}

		//Create a pipe for audio data transfer
		pipeReader, pipeWriter := io.Pipe()
		startTs := time.Now().UnixMilli()
		audioDecoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, "pcm", sampleRate)
		if err != nil {
			p.sendMutex.Unlock()
			_ = pipeReader.Close()
			_ = pipeWriter.Close()
			close(outputChan)
			log.Errorf("Failed to create audio decoder: %v", err)
			return
		}
		audioDecoder.WithFormat(beep.Format{
			SampleRate:  beep.SampleRate(24000),
			NumChannels: channels,
			Precision:   2,
		})

		decoderDone := make(chan struct{})
		go func() {
			defer close(decoderDone)
			if err := audioDecoder.Run(startTs); err != nil {
				log.Errorf("Audio decoding failed: %v", err)
			}
		}()

		defer func() {
			_ = pipeWriter.Close()
			<-decoderDone
			//Release the lock after reading is complete
			log.Debugf("TextToSpeechStream read completed, release sendMutex")
			p.sendMutex.Unlock()
		}()

		//Receive WebSocket data and write to the pipe (lock is held during reading to ensure serialization)
		for {
			select {
			case <-ctx.Done():
				log.Debugf("TextToSpeechStream context done, exit")
				//Close the pipeWriter, let the decoder finish naturally and close the channel
				return
			default:
				messageType, data, err := conn.ReadMessage()
				if err != nil {
					//Close the pipeWriter, let the decoder finish naturally and close the channel
					pipeWriter.Close()
					if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
						return
					}
					log.Errorf("Failed to read WebSocket message: %v, clear the connection", err)
					//If the connection is disconnected, clear the connection and automatically reconnect the next time you use it.
					p.clearConnection()
					return
				}

				if messageType == websocket.BinaryMessage {
					if _, err := pipeWriter.Write(data); err != nil {
						log.Errorf("Failed to write audio data: %v", err)
						return
					}
					return
				}
			}
		}
	}()

	return outputChan, nil
}

// SetVoice sets the tone parameters (EdgeOffline does not support dynamically setting the tone, but does not report an error)
func (p *EdgeOfflineTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	//EdgeOffline is connected through WebSocket. The timbre is controlled by the server and does not support dynamic setting by the client.
	//Return nil to indicate that the operation was successful (although no operation was actually performed)
	return nil
}

// Close closes the resource and releases the connection
func (p *EdgeOfflineTTSProvider) Close() error {
	p.clearConnection()
	return nil
}

// IsValid checks whether the resource is valid
func (p *EdgeOfflineTTSProvider) IsValid() bool {
	p.connMutex.RLock()
	conn := p.conn
	p.connMutex.RUnlock()

	//Check if the connection exists
	return conn != nil
}
