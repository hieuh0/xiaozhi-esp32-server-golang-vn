package funasr

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"xiaozhi-esp32-server-golang/constants"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"

	"xiaozhi-esp32-server-golang/internal/data/audio"
	"xiaozhi-esp32-server-golang/internal/domain/asr/types"
)

// FunasrConfig configuration structure
type FunasrConfig struct {
	Host          string //FunASR service host address
	Port          string //FunASR service port
	Mode          string //recognition mode, such as "online"
	SampleRate    int    //Sampling rate
	ChunkSize     []int  //Chunk size
	ChunkInterval int    //chunking interval
	Timeout       int    //Connection timeout (seconds)
	AutoEnd       bool   //Whether the timeout ends automatically in xx ms, does not depend on isSpeaking being false
}

// DefaultConfig default configuration
var DefaultConfig = FunasrConfig{
	Host:          "localhost",
	Port:          "10095",
	Mode:          "online",
	SampleRate:    audio.SampleRate,
	ChunkInterval: 10,
	ChunkSize:     []int{5, 10, 5},
	Timeout:       30,
}

// Funasr implements the ASR interface
type Funasr struct {
	config FunasrConfig

	//Connection management
	conn      *websocket.Conn
	connMutex sync.RWMutex
	//Send lock to ensure that only one request is using the connection at the same time
	sendMutex sync.Mutex
}

var funasrStreamSeq atomic.Uint64
var funasrStreamPrefix = uuid.NewString()

type streamDebugState struct {
	audioChunkCount  atomic.Uint64
	audioSampleCount atomic.Uint64
}

// FunasrRequest FunASR WebSocket request structure
type FunasrRequest struct {
	Mode          string `json:"mode,omitempty"`           //recognition mode, such as "online"
	ChunkSize     []int  `json:"chunk_size,omitempty"`     //Chunk size
	ChunkInterval int    `json:"chunk_interval,omitempty"` //chunking interval
	AudioFs       int    `json:"audio_fs,omitempty"`       //Sampling rate
	WavName       string `json:"wav_name,omitempty"`       //audio name
	WavFormat     string `json:"wav_format,omitempty"`     //audio format
	IsSpeaking    bool   `json:"is_speaking"`              //Are you talking?
	Hotwords      string `json:"hotwords,omitempty"`       //Hot words
	Itn           bool   `json:"itn,omitempty"`            //Whether to perform text regularization
}

// FunasrResponse FunASR WebSocket response structure
type FunasrResponse struct {
	Text       string  `json:"text"`       //recognized text
	IsFinal    bool    `json:"is_final"`   //Is it the final result?
	WavName    string  `json:"wav_name"`   //audio name
	TimeStamp  string  `json:"timestamp"`  //Timestamp
	Mode       string  `json:"mode"`       //mode
	Confidence float64 `json:"confidence"` //Confidence
}

// NewFunasr creates a new Funasr instance
func NewFunasr(config FunasrConfig) (*Funasr, error) {
	if config.Host == "" {
		config = DefaultConfig
	}

	return &Funasr{
		config: config,
	}, nil
}

// getConnection Gets the connection and creates it if it does not exist
func (f *Funasr) getConnection(ctx context.Context) (*websocket.Conn, error) {
	//Try reading an existing connection first
	f.connMutex.RLock()
	conn := f.conn
	f.connMutex.RUnlock()

	if conn != nil {
		log.Debugf("FunASR WebSocket multiplexed connection: conn=%p", conn)
		return conn, nil
	}

	//New connection needs to be created
	f.connMutex.Lock()
	defer f.connMutex.Unlock()

	//Double check, maybe other goroutine has already created the connection
	if f.conn != nil {
		return f.conn, nil
	}

	//Create new connection
	url := fmt.Sprintf("ws://%s:%s/", f.config.Host, f.config.Port)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		if isServiceUnavailableError(err) {
			return nil, fmt.Errorf("FunASR service not ready at %s (still starting up or not running): %v", url, err)
		}
		return nil, fmt.Errorf("Failed to connect to FunASR service: %v", err)
	}

	f.conn = conn
	log.Infof("FunASR WebSocket connection established: conn=%p", conn)
	return conn, nil
}

// clearConnection clears the connection (used for disconnection and reconnection)
func (f *Funasr) clearConnection() {
	f.connMutex.Lock()
	defer f.connMutex.Unlock()

	if f.conn != nil {
		log.Infof("FunASR WebSocket connection cleared: conn=%p", f.conn)
		f.conn.Close()
		f.conn = nil
	}
}

// StreamingResult Streaming recognition result
type StreamingResult struct {
	Text    string //recognized text
	IsFinal bool   //Is it the final result?
}

// isTimeoutError determines whether it is a timeout error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	//Check if it is a network timeout error
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	//Check whether the error message contains the timeout keyword
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "i/o timeout")
}

// isConnectionClosedError determines whether the connection is closed error
func isConnectionClosedError(err error) bool {
	if err == nil {
		return false
	}

	//Check if closing error for WebSocket
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
		return true
	}

	//Check if the error message contains the connection close keyword
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "connection closed") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "use of closed network connection")
}

// writeMessage securely writes a message to a WebSocket connection
func (f *Funasr) writeMessage(conn *websocket.Conn, messageType int, data []byte) error {
	//Use read locks to protect connection write operations to prevent data confusion caused by concurrent writes
	f.connMutex.RLock()
	defer f.connMutex.RUnlock()

	//Check if the connection is valid
	if conn == nil {
		return fmt.Errorf("connection closed")
	}

	return conn.WriteMessage(messageType, data)
}

// StreamingRecognize implements streaming recognition
// Receive audio data from audioStream and return the result through resultChan
// Cancellation and timeout of the recognition process can be controlled through ctx
func (f *Funasr) StreamingRecognize(ctx context.Context, audioStream <-chan []float32) (chan types.StreamingResult, error) {
	//Use send lock protection to ensure that only one request is using the connection at the same time
	f.sendMutex.Lock()
	//Note: The lock is not released when the function returns, but when the goroutine completes

	//Get a connection (reuse or create)
	conn, err := f.getConnection(ctx)
	if err != nil {
		f.sendMutex.Unlock() //Release the lock immediately when acquiring the connection fails
		return nil, err
	}

	subCtx, cancelFunc := context.WithCancel(ctx)
	streamID := fmt.Sprintf("funasr-stream-%s-%d", funasrStreamPrefix, funasrStreamSeq.Add(1))
	wavName := streamID
	debugState := &streamDebugState{}

	//Send initial message
	firstMessage := FunasrRequest{
		Mode:          f.config.Mode,
		ChunkSize:     []int{5, 10, 5},
		ChunkInterval: f.config.ChunkInterval,
		AudioFs:       f.config.SampleRate,
		WavName:       wavName,
		WavFormat:     "pcm",
		IsSpeaking:    true,
		Hotwords:      "{\"hello world\":40}",
		Itn:           true,
	}

	log.Debugf(
		"funasr StreamingRecognize starts: stream_id=%s, conn=%p, mode=%s, chunk_interval=%d, chunk_size=%v, wav_name=%s",
		streamID,
		conn,
		f.config.Mode,
		f.config.ChunkInterval,
		firstMessage.ChunkSize,
		firstMessage.WavName,
	)

	messageBytes, err := json.Marshal(firstMessage)
	if err != nil {
		cancelFunc()
		f.sendMutex.Unlock() //Release the lock immediately when serialization fails
		return nil, fmt.Errorf("Failed to serialize initial message: %v", err)
	}

	err = f.writeMessage(conn, websocket.TextMessage, messageBytes)
	if err != nil {
		//Failed to send, clear the connection, and automatically reconnect the next time you use it.
		log.Errorf("Failed to send initial message: %v, clear connection", err)
		f.clearConnection()
		cancelFunc()
		f.sendMutex.Unlock() //Release the lock immediately when sending fails
		return nil, fmt.Errorf("Failed to send initial message: %v", err)
	}

	//Create a result channel with buffering to avoid blocking
	resultChan := make(chan types.StreamingResult, 20)

	//Use WaitGroup to wait for two goroutines to complete
	var wg sync.WaitGroup
	wg.Add(2)

	//Start goroutine to receive and send data
	//Release the lock when the goroutine completes
	go func() {
		defer wg.Done()
		f.recvResult(subCtx, conn, streamID, wavName, debugState, resultChan)
	}()

	go func() {
		defer wg.Done()
		f.forwardStreamAudio(subCtx, cancelFunc, conn, streamID, wavName, debugState, audioStream)
	}()

	//Wait in the background for the goroutine to complete and release the lock
	go func() {
		wg.Wait()
		f.clearConnection()
		f.sendMutex.Unlock()
		log.Debugf(
			"funasr StreamingRecognize goroutine completed, released sendMutex: stream_id=%s, wav_name=%s, chunks=%d, samples=%d",
			streamID,
			wavName,
			debugState.audioChunkCount.Load(),
			debugState.audioSampleCount.Load(),
		)
	}()

	return resultChan, nil
}

func (f *Funasr) recvResult(ctx context.Context, conn *websocket.Conn, streamID string, wavName string, debugState *streamDebugState, resultChan chan types.StreamingResult) {
	defer func() {
		close(resultChan)
	}()

	for {
		select {
		case <-ctx.Done():
			//Context cancellation, exit goroutine
			log.Debugf("funasr recvResult Canceled: %v", ctx.Err())
			return
		default:
			//Continue processing normally
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Debugf("funasr recvResult failed to read the recognition result: stream_id=%s, conn=%p, err=%v, clear the connection", streamID, conn, err)
			//If the read fails, the connection will be cleared and the connection will be automatically reconnected next time.
			f.clearConnection()
			return
		}
		log.Debugf(
			"funasr recvResult reads the recognition results: stream_id=%s, conn=%p, chunks=%d, samples=%d, payload=%v",
			streamID,
			conn,
			debugState.audioChunkCount.Load(),
			debugState.audioSampleCount.Load(),
			string(message),
		)

		var response FunasrResponse
		err = json.Unmarshal(message, &response)
		if err != nil {
			log.Debugf("funasr recvResult failed to parse and identify the result: %v", err)
			continue
		}

		if response.WavName != "" && response.WavName != wavName {
			log.Warnf(
				"funasr recvResult ignores non-current stream results: stream_id=%s, expected_wav=%s, actual_wav=%s, conn=%p, chunks=%d, samples=%d",
				streamID,
				wavName,
				response.WavName,
				conn,
				debugState.audioChunkCount.Load(),
				debugState.audioSampleCount.Load(),
			)
			continue
		}

		//Only send results if there is text
		/*if response.Text == "" {
			continue
		}*/

		streamingResult := f.toStreamingResult(response)

		//Send recognition results
		select {
		case <-ctx.Done():
			//Context cancellation, exit goroutine
			log.Debugf("funasr recvResult Canceled: %v", ctx.Err())
			return
		case resultChan <- streamingResult:
		}
		/*if f.config.AutoEnd {
			log.Debugf("funasr recvResult autoend")
			return
		}*/
		//Result sent successfully
		//If this is the final result and the input has ended, exit the loop
		if streamingResult.IsFinal {
			log.Debugf(
				"funasr recvResult isfinal: stream_id=%s, conn=%p, response_mode=%s, raw_is_final=%v, text_len=%d, wav_name=%s, chunks=%d, samples=%d",
				streamID,
				conn,
				response.Mode,
				response.IsFinal,
				len([]rune(response.Text)),
				response.WavName,
				debugState.audioChunkCount.Load(),
				debugState.audioSampleCount.Load(),
			)
			return
		}
	}
}

func (f *Funasr) toStreamingResult(response FunasrResponse) types.StreamingResult {
	result := types.StreamingResult{
		Text:    response.Text,
		IsFinal: response.IsFinal,
		AsrType: constants.AsrTypeFunAsr,
		Mode:    response.Mode,
	}

	if strings.EqualFold(strings.TrimSpace(f.config.Mode), "2pass") {
		switch strings.ToLower(strings.TrimSpace(response.Mode)) {
		case "2pass-online":
			result.IsFinal = false
		case "2pass-offline":
			result.IsFinal = true
		}
	}

	if result.IsFinal && strings.TrimSpace(result.Text) == "" {
		result.EmptyReason = types.EmptyReasonProviderEmptyFinal
	}

	return result
}

func (f *Funasr) forwardStreamAudio(ctx context.Context, cancelFunc context.CancelFunc, conn *websocket.Conn, streamID string, wavName string, debugState *streamDebugState, audioStream <-chan []float32) {
	sendEndMsg := func() {
		//Send termination message
		endMessage := FunasrRequest{
			Mode:          f.config.Mode,
			ChunkInterval: f.config.ChunkInterval,
			ChunkSize:     []int{5, 10, 5},
			WavName:       wavName,
			IsSpeaking:    false,
		}
		endMessageBytes, _ := json.Marshal(endMessage)
		log.Debugf(
			"funasr forwardStreamAudio sends end message: stream_id=%s, conn=%p, chunks=%d, samples=%d, payload=%v",
			streamID,
			conn,
			debugState.audioChunkCount.Load(),
			debugState.audioSampleCount.Load(),
			string(endMessageBytes),
		)
		err := f.writeMessage(conn, websocket.TextMessage, endMessageBytes)
		if err != nil {
			log.Debugf("funasr forwardStreamAudio failed to send the end message: stream_id=%s, conn=%p, err=%v, clear the connection", streamID, conn, err)
			f.clearConnection()
		}
	}
	//Process the input audio stream
	for {
		select {
		case <-ctx.Done():
			//Context cancels, sends end message and exits
			log.Debugf(
				"funasr forwardStreamAudio context canceled: stream_id=%s, conn=%p, chunks=%d, samples=%d, err=%v",
				streamID,
				conn,
				debugState.audioChunkCount.Load(),
				debugState.audioSampleCount.Load(),
				ctx.Err(),
			)
			//Note: There is no need to call cancelFunc() here, because ctx.Done() has been triggered indicating that the context has been cancelled.
			sendEndMsg()
			return
		case pcmChunk, ok := <-audioStream:
			if !ok {
				//The channel has been closed, the input has ended, and the receiving goroutine needs to be notified to stop.
				log.Debugf(
					"funasr forwardStreamAudio audio channel closed: stream_id=%s, conn=%p, chunks=%d, samples=%d",
					streamID,
					conn,
					debugState.audioChunkCount.Load(),
					debugState.audioSampleCount.Load(),
				)
				sendEndMsg()
				return
			}

			//Convert PCM data to bytes
			audioBytes := Float32SliceToBytes(pcmChunk)

			//log.Debugf("fufunasr forwardStreamAudio sends audio data, pcmChunk len: %v, audioBytes len: %v len: %v", len(pcmChunk), len(audioBytes))

			//Send audio data
			err := f.writeMessage(conn, websocket.BinaryMessage, audioBytes)
			if err != nil {
				log.Debugf("funasr forwardStreamAudio failed to send audio data: stream_id=%s, conn=%p, err=%v, clear the connection", streamID, conn, err)
				f.clearConnection()
				cancelFunc() //Cancel the context when sending fails and notify recvResult goroutine to stop
				return
			}
			chunkCount := debugState.audioChunkCount.Add(1)
			sampleCount := debugState.audioSampleCount.Add(uint64(len(pcmChunk)))
			if chunkCount <= 3 || chunkCount%10 == 0 {
				log.Debugf(
					"funasr forwardStreamAudio Audio chunks sent: stream_id=%s, conn=%p, chunk=%d, chunk_samples=%d, total_samples=%d, bytes=%d",
					streamID,
					conn,
					chunkCount,
					len(pcmChunk),
					sampleCount,
					len(audioBytes),
				)
			}
		}
	}
}

// Process processes audio data and returns recognition results
func (f *Funasr) Process(pcmData []float32) (string, error) {
	ctx := context.Background()

	//Use send lock protection to ensure that only one request is using the connection at the same time
	f.sendMutex.Lock()
	defer f.sendMutex.Unlock()

	//Get a connection (reuse or create)
	conn, err := f.getConnection(ctx)
	if err != nil {
		return "", err
	}

	audioBytes := Float32SliceToBytes(pcmData)

	//Send initial message
	firstMessage := FunasrRequest{
		Mode:          f.config.Mode,
		ChunkSize:     []int{5, 10, 5},
		ChunkInterval: f.config.ChunkInterval,
		AudioFs:       f.config.SampleRate,
		WavName:       "stream",
		WavFormat:     "pcm",
		IsSpeaking:    true,
		Hotwords:      "",
		Itn:           true,
	}

	messageBytes, err := json.Marshal(firstMessage)
	if err != nil {
		return "", fmt.Errorf("Failed to serialize initial message: %v", err)
	}

	err = f.writeMessage(conn, websocket.TextMessage, messageBytes)
	if err != nil {
		//Failed to send, clear the connection, and automatically reconnect the next time you use it.
		log.Errorf("Failed to send initial message: %v, clear connection", err)
		f.clearConnection()
		return "", fmt.Errorf("Failed to send initial message: %v", err)
	}

	//Send audio data in chunks
	chunkSize := int(audio.SampleRate * 0.1) //Each block size is about 100ms of audio (16000 * 0.1)
	for i := 0; i < len(audioBytes); i += chunkSize {
		end := i + chunkSize
		if end > len(audioBytes) {
			end = len(audioBytes)
		}
		chunk := audioBytes[i:end]

		err = f.writeMessage(conn, websocket.BinaryMessage, chunk)
		if err != nil {
			//Failed to send, clear the connection, and automatically reconnect the next time you use it.
			log.Errorf("Failed to send audio data: %v, clear the connection", err)
			f.clearConnection()
			return "", fmt.Errorf("Failed to send audio data: %v", err)
		}
	}

	//Send termination message
	endMessage := FunasrRequest{
		IsSpeaking: false,
	}
	endMessageBytes, _ := json.Marshal(endMessage)
	err = f.writeMessage(conn, websocket.TextMessage, endMessageBytes)
	if err != nil {
		//Failed to send, clear the connection, and automatically reconnect the next time you use it.
		log.Errorf("Failed to send termination message: %v, clear the connection", err)
		f.clearConnection()
		return "", fmt.Errorf("Failed to send termination message: %v", err)
	}

	//Set read timeout
	conn.SetReadDeadline(time.Now().Add(time.Duration(f.config.Timeout) * time.Second))

	//Read results
	var result string
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if isTimeoutError(err) {
				log.Debugf("funasr Process reading result timeout: %v", err)
				f.clearConnection() //Read timeout, clear connection
				return "", fmt.Errorf("Reading result timeout: %v", err)
			}
			if isConnectionClosedError(err) {
				log.Debugf("funasr Process read result connection is closed: %v", err)
				f.clearConnection() //The connection has been closed, clear the connection
				return "", fmt.Errorf("Connection closed: %v", err)
			}
			//If the read fails, the connection will be cleared and the connection will be automatically reconnected next time.
			log.Errorf("Funasr Process failed to read the result: %v, clear the connection", err)
			f.clearConnection()
			return "", fmt.Errorf("Failed to read result: %v", err)
		}

		var response FunasrResponse
		err = json.Unmarshal(message, &response)
		if err != nil {
			continue
		}

		//Check if the result is final
		if response.IsFinal {
			result = response.Text
			break
		}
	}

	return result, nil
}

func Float32ToInt16(sample float32) int16 {
	//Limit to [-1, 1] to avoid overflow
	if sample > 1.0 {
		sample = 1.0
	} else if sample < -1.0 {
		sample = -1.0
	}
	return int16(sample * 32767)
}

func Float32SliceToBytes(samples []float32) []byte {
	data := make([]byte, len(samples)*2)
	for i, s := range samples {
		i16 := Float32ToInt16(s)
		data[2*i] = byte(i16)
		data[2*i+1] = byte(i16 >> 8)
	}
	return data
}

// isServiceUnavailableError returns true when the error indicates the remote
// service is not running yet (connection refused, dial failure, unknown host).
func isServiceUnavailableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "no such host")
}

// Close closes the resource and releases the connection
func (f *Funasr) Close() error {
	f.clearConnection()
	return nil
}

// IsValid checks whether the resource is valid
func (f *Funasr) IsValid() bool {
	f.connMutex.RLock()
	conn := f.conn
	f.connMutex.RUnlock()
	return conn != nil
}

/*
Error classification examples:

1. Timeout error:
   if isTimeoutError(err) {
       //Handle timeout situations, you may need to retry or adjust the timeout period
       log.Warnf("Operation timeout: %v", err)
   }

2. Connection closed error:
   if isConnectionClosedError(err) {
       //Handle connection closed situations, which may require re-establishing the connection
       log.Warnf("Connection closed: %v", err)
   }

3. Combined error handling:
   _, message, err := conn.ReadMessage()
   if err != nil {
       if isTimeoutError(err) {
           //Timeout: Possibly network delay or slow server response
           //Suggestion: Adjust the timeout or try again
       } else if isConnectionClosedError(err) {
           //Connection closed: The server may be actively disconnected or the network may be interrupted.
           //Suggestion: Re-establish the connection
       } else {
           //Other errors: possibly protocol error or data format error
           //Recommendation: Check data format or protocol implementation
       }
   }

Common error types:
- Timeout: i/o timeout, context deadline exceeded
- Connection closed: connection closed, broken pipe, connection reset
- WebSocket closed: close 1000 (normal), close 1001 (going away)
*/
