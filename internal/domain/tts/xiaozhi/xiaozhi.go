package xiaozhi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/gorilla/websocket"
)

var deviceIdList = []string{
	"ba:8f:17:de:94:94",
	"f2:85:44:27:7b:51",
	"4f:57:fb:d4:69:fa",
	"b3:1e:1c:80:cc:78",
	"32:a5:cc:b7:c0:e4",
	"2b:60:6a:5a:72:10",
	"ca:a6:8b:20:f1:6f",
	"26:1a:d7:27:9f:f8",
	"03:02:26:58:2b:06",
	"5f:f3:85:8b:5d:da",
}

// Record the deviceId of the latest error and its disabled expiration time
var (
	deviceIdBlocklist     = make(map[string]time.Time)
	deviceIdBlocklistLock sync.Mutex
	//Device ID disable time (how long it will not be used after the error)
	deviceIdBlockDuration = 5 * time.Second
)

// XiaozhiProvider XiaozhiTTS WebSocket Provider
// Supports streaming text-to-speech
type XiaozhiProvider struct {
	ServerAddr  string
	DeviceID    string
	AudioFormat map[string]interface{}
	Header      http.Header
}

// Regularly clean up the expired deviceId disabled list
func init() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			//Clean up the expired deviceId disabled list
			deviceIdBlocklistLock.Lock()
			now := time.Now()
			for id, expireTime := range deviceIdBlocklist {
				if now.After(expireTime) {
					delete(deviceIdBlocklist, id)
					log.Debugf("Device ID disable has expired, re-enable: %s", id)
				}
			}
			deviceIdBlocklistLock.Unlock()
		}
	}()
}

// Add deviceId to disabled list
func blockDeviceId(deviceId string) {
	deviceIdBlocklistLock.Lock()
	defer deviceIdBlocklistLock.Unlock()

	deviceIdBlocklist[deviceId] = time.Now().Add(deviceIdBlockDuration)
	log.Warnf("Device ID %s has been added to the disabled list and will be re-enabled after %v", deviceId, deviceIdBlockDuration)
}

// Check if deviceId is in disabled list
func isDeviceIdBlocked(deviceId string) bool {
	deviceIdBlocklistLock.Lock()
	defer deviceIdBlocklistLock.Unlock()

	expireTime, exists := deviceIdBlocklist[deviceId]
	if !exists {
		return false
	}

	//If expiration time has passed, remove from disabled list
	if time.Now().After(expireTime) {
		delete(deviceIdBlocklist, deviceId)
		log.Debugf("Device ID disable has expired, re-enable: %s", deviceId)
		return false
	}

	return true
}

// NewXiaozhiProvider creates a new XiaozhiTTS Provider
func NewXiaozhiProvider(config map[string]interface{}) *XiaozhiProvider {
	serverAddr, _ := config["server_addr"].(string)
	deviceID, _ := config["device_id"].(string)
	clientID, _ := config["client_id"].(string)
	token, _ := config["token"].(string)
	format := map[string]interface{}{
		"sample_rate":    16000,
		"channels":       1,
		"frame_duration": 20,
		"format":         "opus",
	}

	header := http.Header{}
	header.Set("Device-Id", deviceID)
	header.Set("Content-Type", "application/json")
	header.Set("Authorization", "Bearer "+token)
	header.Set("Protocol-Version", "1")
	header.Set("Client-Id", clientID)

	return &XiaozhiProvider{
		ServerAddr:  serverAddr,
		DeviceID:    deviceID,
		AudioFormat: format,
		Header:      header,
	}
}

// selectDeviceId Select an available device ID
func (p *XiaozhiProvider) selectDeviceId() string {
	//Find the deviceId that is not disabled from deviceIdList
	for _, deviceId := range deviceIdList {
		if !isDeviceIdBlocked(deviceId) {
			log.Debugf("Select a device ID that is not disabled: %s", deviceId)
			return deviceId
		}
	}

	//If all deviceIds are disabled, poll select from all deviceIds
	if len(deviceIdList) > 0 {
		//Use a simple polling strategy (time-based)
		selectedIndex := int(time.Now().Unix()) % len(deviceIdList)
		selectedDeviceId := deviceIdList[selectedIndex]
		log.Warnf("All deviceIds are disabled, polling selects device ID: %s (index: %d)", selectedDeviceId, selectedIndex)
		return selectedDeviceId
	}

	//If deviceIdList is empty, use the passed in deviceId
	if p.DeviceID != "" {
		log.Warnf("deviceIdList is empty, use the current device ID: %s", p.DeviceID)
		return p.DeviceID
	}

	//If neither exists, return the first device ID (if it exists)
	if len(deviceIdList) > 0 {
		return deviceIdList[0]
	}

	return ""
}

// createWSConnection creates a new WebSocket connection
func (p *XiaozhiProvider) createWSConnection(ctx context.Context) (*websocket.Conn, string, error) {
	//Choose an available device ID
	selectedDeviceId := p.selectDeviceId()
	if selectedDeviceId == "" {
		return nil, "", fmt.Errorf("Unable to select device ID")
	}

	//Update current p.DeviceID and Header
	p.DeviceID = selectedDeviceId
	p.Header.Set("Device-Id", selectedDeviceId)

	//Create new connection
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, p.ServerAddr, p.Header)
	if err != nil {
		log.Errorf("Failed to create WebSocket connection: %v, device ID: %s", err, selectedDeviceId)
		blockDeviceId(selectedDeviceId) //Add the failed deviceId to the disabled list
		return nil, "", err
	}

	//Set up stay connected
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
	})

	//Send hello message when creating a new connection
	helloMsg := map[string]interface{}{
		"type":         "hello",
		"device_id":    selectedDeviceId,
		"transport":    "websocket",
		"version":      1,
		"audio_params": p.AudioFormat,
	}
	log.Debugf("Create a new connection and send a hello message, device ID: %s", selectedDeviceId)
	if err := conn.WriteJSON(helloMsg); err != nil {
		conn.Close()
		return nil, "", fmt.Errorf("Failed to send hello message: %v", err)
	}

	return conn, selectedDeviceId, nil
}

type RecvMsg struct {
	Type    string `json:"type"`
	State   string `json:"state"`
	Text    string `json:"text"`
	Version int    `json:"version"`
}

// sendStopMessage sends a stop message and closes the connection
func sendStopMessage(conn *websocket.Conn, deviceId string) {
	stopMsg := map[string]interface{}{
		"type":      "listen",
		"device_id": deviceId,
		"state":     "stop",
	}
	if err := conn.WriteJSON(stopMsg); err != nil {
		log.Warnf("Failed to send stop message: %v, device ID: %s", err, deviceId)
	} else {
		log.Debugf("Sending stop message successfully, device ID: %s", deviceId)
	}
}

// handleTTSConnection encapsulates the logic of obtaining connections, sending messages, and receiving messages
func (p *XiaozhiProvider) handleTTSConnection(ctx context.Context, text string, outputChan chan []byte) error {
	//Create new connection
	conn, deviceId, err := p.createWSConnection(ctx)
	if err != nil {
		return fmt.Errorf("Failed to create XiaozhiTTS connection: %v", err)
	}
	defer func() {
		//Send a stop message and close the connection
		sendStopMessage(conn, deviceId)
		conn.Close()
	}()

	//Send listen detect message
	sendText := fmt.Sprintf("`%s`", text)
	listenMsg := map[string]interface{}{
		"type":      "listen",
		"device_id": deviceId,
		"state":     "detect",
		"text":      sendText,
	}
	log.Debugf("Send xiaozhi server message: %v", listenMsg)

	if err := conn.WriteJSON(listenMsg); err != nil {
		log.Errorf("Failed to send listen message: %v, device ID: %s", err, deviceId)
		blockDeviceId(deviceId) //Add the wrong deviceId to the disabled list
		return fmt.Errorf("Failed to send message: %v", err)
	}

	//Read and process messages
	startTs := time.Now().UnixMilli()
	var firstFrameTs bool
	i := 0
	receivedFrames := false

	for {
		select {
		case <-ctx.Done():
			log.Debugf("xiaozhi server message ctx.Done(), device ID: %s", deviceId)
			return nil
		default:
		}
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			//Connection error
			log.Errorf("Error reading message: %v, device ID: %s", err, deviceId)

			//If no audio frames have been received, there may be a problem with the connection. Add deviceId to the disabled list.
			if !receivedFrames {
				blockDeviceId(deviceId)
			}

			return fmt.Errorf("Error reading message: %v", err)
		}
		if msgType == websocket.TextMessage {
			log.Debugf("Received xiaozhi server message: %s", string(msg))
			var recvMsg RecvMsg
			err := json.Unmarshal(msg, &recvMsg)
			if err != nil {
				continue
			}
			if recvMsg.Type == "tts" {
				if recvMsg.State == "stop" {
					log.Debugf("xiaozhi server message tts stop message")
					return nil
				}
			}
		} else if msgType == websocket.BinaryMessage {
			receivedFrames = true
			if !firstFrameTs {
				firstFrameTs = true
				log.Debugf("Tts time consumption statistics: xiaozhi service tts first audio frame time: %d", time.Now().UnixMilli()-startTs)
			}
			outputChan <- msg
			if i%20 == 0 {
				log.Debugf("xiaozhi server audio message, %d audio frames have been received", i)
			}
			i++
		}
	}
}

// TextToSpeechStream implements streaming TTS and returns opus audio frame chan
func (p *XiaozhiProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	outputChan := make(chan []byte, 1000)

	//Try to handle TTS connection, support retry
	go func() {
		defer close(outputChan)

		retryCount := 0
		maxRetries := 2
		var lastError error

		//Try maxRetries times at most
		for retryCount <= maxRetries {
			if retryCount > 0 {
				log.Infof("Try to reacquire the connection, %d/%d retry", retryCount, maxRetries)

				//Check if context has been canceled before retrying
				select {
				case <-ctx.Done():
					log.Debugf("Context canceled, stop retrying")
					return
				default:
					//Keep trying again
				}
			}

			//Handling TTS connections
			err := p.handleTTSConnection(ctx, text, outputChan)

			if err == nil {
				//Connection processing successful, no need to retry
				return
			}

			lastError = err
			log.Errorf("TTS connection processing failed: %v (Retry: %d/%d)", err, retryCount, maxRetries)

			retryCount++
		}

		if retryCount > maxRetries {
			log.Warnf("Reached the maximum number of retries %d and gave up retrying. The final error was: %v", maxRetries, lastError)
		}
	}()

	return outputChan, nil
}

// GetVoiceInfo Get TTS configuration information
func (p *XiaozhiProvider) GetVoiceInfo() map[string]interface{} {
	return map[string]interface{}{
		"type":         "xiaozhi_ws",
		"server_addr":  p.ServerAddr,
		"device_id":    p.DeviceID,
		"audio_format": p.AudioFormat,
	}
}

// SetVoice sets voice parameters (Xiaozhi Provider does not support dynamic setting of voice)
func (p *XiaozhiProvider) SetVoice(voiceConfig map[string]interface{}) error {
	return fmt.Errorf("Xiaozhi TTS Provider does not support dynamically setting timbres")
}

// Close closes the resource (stateless provider, no need to close)
func (p *XiaozhiProvider) Close() error {
	return nil
}

// IsValid checks whether the resource is valid
func (p *XiaozhiProvider) IsValid() bool {
	return p != nil
}

// TextToSpeech implements the BaseTTSProvider interface and directly aggregates streaming frames
func (p *XiaozhiProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	ch, err := p.TextToSpeechStream(ctx, text, sampleRate, channels, frameDuration)
	if err != nil {
		return nil, err
	}
	var frames [][]byte
	for frame := range ch {
		frames = append(frames, frame)
	}
	return frames, nil
}
