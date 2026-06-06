package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"hash/fnv"
	"runtime"
	"sync"
	"time"

	data_client "xiaozhi-esp32-server-golang/internal/data/client"
	"xiaozhi-esp32-server-golang/internal/data/history"
	"xiaozhi-esp32-server-golang/internal/domain/eventbus"
	"xiaozhi-esp32-server-golang/internal/domain/memory/llm_memory"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/cloudwego/eino/schema"
	"github.com/spf13/viper"
)

var (
	// MessageWorkerNum is the CPU-based worker count shared by Redis and History processing.
	// It must be a power of two for hash distribution.
	MessageWorkerNum = getMessageWorkerNum()
)

// getMessageWorkerNum rounds the CPU count up to the nearest power of two.
// The minimum is 4 and the maximum is 64.
func getMessageWorkerNum() int {
	cpuNum := runtime.NumCPU()

	// Clamp the worker count between 4 and 64.
	if cpuNum < 4 {
		return 4
	}
	if cpuNum > 64 {
		return 64
	}

	// Round up to the nearest power of two.
	power := 1
	for power < cpuNum {
		power <<= 1
	}
	return power
}

// MessageWorker processes messages with a fixed goroutine pool.
// SessionID hashing preserves message order within each session.
// It handles Redis, MemoryProvider, and History messages.
type MessageWorker struct {
	client  *history.HistoryClient
	workers []chan *eventbus.AddMessageEvent // One channel per worker.
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewMessageWorker creates a message worker.
func NewMessageWorker(cfg history.HistoryClientConfig) *MessageWorker {
	client := history.NewHistoryClient(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	worker := &MessageWorker{
		client:  client,
		workers: make([]chan *eventbus.AddMessageEvent, MessageWorkerNum),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize each worker channel and start its goroutine.
	for i := 0; i < MessageWorkerNum; i++ {
		worker.workers[i] = make(chan *eventbus.AddMessageEvent, 100) // Buffer up to 100 messages.
		worker.wg.Add(1)
		go worker.workerLoop(i)
	}

	worker.subscribeEvents()
	log.Infof("MessageWorker initialized with %d worker goroutines for Redis, MemoryProvider, and History", MessageWorkerNum)
	return worker
}

// workerLoop processes messages sequentially for one worker.
func (w *MessageWorker) workerLoop(index int) {
	defer w.wg.Done()
	defer log.Infof("MessageWorker worker %d exited", index)

	ch := w.workers[index]
	for {
		select {
		case <-w.ctx.Done():
			// Process messages remaining in the channel.
			for {
				select {
				case event := <-ch:
					if event != nil {
						w.processMessage(event)
					}
				default:
					return
				}
			}
		case event, ok := <-ch:
			if !ok {
				// The channel is closed.
				return
			}
			if event != nil {
				w.processMessage(event)
			}
		}
	}
}

// processMessage handles messages sequentially in a worker goroutine.
// It preserves message order per device or session across Redis, MemoryProvider, and History.
func (w *MessageWorker) processMessage(event *eventbus.AddMessageEvent) {
	// 1. Process History for every message.
	// Use an independent context so conversation cancellation does not interrupt history persistence.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Determine whether this is an insert or update.
	if event.IsUpdate {
		// Second stage: update audio.
		w.updateMessageAudio(ctx, event)
	} else {
		// First stage: save the text message, including Redis handling.
		w.saveMessageText(ctx, event)
	}

	// 2. Process MemoryProvider only for new messages, independently of Redis and manager.
	// Long-term memory providers such as memobase and mem0 are required in either configuration.
	if !event.IsUpdate {
		w.processMemoryProvider(event)
	}
}

// processMemoryProvider handles long-term memory providers such as memobase and mem0.
// It runs independently of Redis and manager configuration.
func (w *MessageWorker) processMemoryProvider(event *eventbus.AddMessageEvent) {
	clientState := event.ClientState
	if clientState.MemoryProvider == nil {
		return
	}
	if clientState.GetMemoryMode() != data_client.MemoryModeLong {
		return
	}

	err := clientState.MemoryProvider.AddMessage(
		clientState.Ctx,
		clientState.GetDeviceIDOrAgentID(),
		event.Msg)
	if err != nil {
		log.Errorf("add message to memory provider failed: %v", err)
	}
}

// hashSessionID hashes a SessionID and returns a worker index.
func (w *MessageWorker) hashSessionID(sessionID string) int {
	if sessionID == "" {
		return 0 // Use the first worker when SessionID is empty.
	}

	// Use the FNV-1a hash function.
	h := fnv.New32a()
	h.Write([]byte(sessionID))
	hash := h.Sum32()
	return int(hash) % MessageWorkerNum
}

// subscribeEvents subscribes to EventBus events.
func (w *MessageWorker) subscribeEvents() {
	bus := eventbus.Get()
	// Subscribe to the unified message-add event on the same topic as EventHandle.
	bus.Subscribe(eventbus.TopicAddMessage, w.handleAddMessage)
}

// handleAddMessage routes message-add events to the appropriate worker.
func (w *MessageWorker) handleAddMessage(event *eventbus.AddMessageEvent) {
	if event == nil || event.ClientState == nil {
		return
	}

	// Prefer SessionID as the routing key and fall back to DeviceID.
	key := event.ClientState.SessionID
	if key == "" {
		key = event.ClientState.DeviceID
	}
	if key == "" {
		log.Warnf("SessionID and DeviceID are both empty; cannot route message")
		return
	}

	// Calculate the hash and select a worker.
	workerIndex := w.hashSessionID(key)

	// Send to the worker channel without blocking.
	select {
	case w.workers[workerIndex] <- event:
		// Message queued successfully.
	default:
		// The buffered channel should rarely be full.
		log.Warnf("worker %d channel is full; dropping message, session_id: %s, device_id: %s",
			workerIndex, event.ClientState.SessionID, event.ClientState.DeviceID)
	}
}

// saveMessageText saves a first-stage text message, optionally with audio.
// It includes Redis handling when config_provider.type is redis.
func (w *MessageWorker) saveMessageText(ctx context.Context, event *eventbus.AddMessageEvent) {
	// Add the message to Redis for LLM context when Redis is the config provider.
	providerType := viper.GetString("config_provider.type")
	if providerType == "redis" {
		clientState := event.ClientState
		llm_memory.Get().AddMessage(
			clientState.Ctx,
			clientState.DeviceID,
			clientState.AgentID,
			event.Msg)
		return
	}

	// Determine the message role.
	var role history.MessageType
	switch event.Msg.Role {
	case schema.User:
		role = history.MessageTypeUser
	case schema.Assistant:
		role = history.MessageTypeAssistant
	case schema.Tool:
		role = history.MessageTypeTool
	case schema.System:
		role = history.MessageTypeSystem
	default:
		log.Warnf("unsupported message role: %s", event.Msg.Role)
		return
	}

	// Convert audio when present.
	var audioBase64 string
	var audioFormat string
	var audioSize int

	if len(event.AudioData) > 0 {
		// Save ASR text and audio together.
		var wavData []byte
		var err error

		// Select the audio conversion method by message role.
		if event.Msg.Role == schema.User {
			// User ASR messages use PCM float32.
			if len(event.AudioData) > 0 {
				wavData, err = util.PCMFloat32BytesToWav(
					event.AudioData[0], // User messages contain one audio element.
					event.SampleRate,
					event.Channels)
			}
		} else {
			// Assistant TTS messages use Opus; normally they are saved in two stages.
			wavData, err = util.OpusFramesToWav(
				event.AudioData,
				event.SampleRate,
				event.Channels)
		}

		if err != nil {
			log.Errorf("audio conversion failed, device_id: %s, message_id: %s, role: %s, error: %v",
				event.ClientState.DeviceID, event.MessageID, event.Msg.Role, err)
			// Fall back to concatenating the raw frames.
			var fallbackData []byte
			for _, frame := range event.AudioData {
				fallbackData = append(fallbackData, frame...)
			}
			audioBase64 = base64.StdEncoding.EncodeToString(fallbackData)
			audioSize = event.AudioSize
			audioFormat = "raw" // The fallback preserves the raw format.
		} else {
			audioBase64 = base64.StdEncoding.EncodeToString(wavData)
			audioSize = len(wavData)
			audioFormat = "wav"
		}
	}

	// Build metadata containing only the timestamp.
	metadata := map[string]interface{}{
		"timestamp": event.Timestamp.Format(time.RFC3339),
	}

	// Prepare tool-call fields.
	var toolCallID string
	var toolCallsJSON *string

	// Store tool_call_id for Tool messages.
	if event.Msg.Role == schema.Tool && event.Msg.ToolCallID != "" {
		toolCallID = event.Msg.ToolCallID
	}

	// Store ToolCalls for Assistant messages when present.
	if event.Msg.Role == schema.Assistant && len(event.Msg.ToolCalls) > 0 {
		// Serialize ToolCalls as a JSON string.
		toolCallsBytes, err := json.Marshal(event.Msg.ToolCalls)
		if err != nil {
			log.Warnf("failed to serialize ToolCalls, device_id: %s, message_id: %s, error: %v",
				event.ClientState.DeviceID, event.MessageID, err)
		} else {
			jsonStr := string(toolCallsBytes)
			toolCallsJSON = &jsonStr
		}
	}

	req := &history.SaveMessageRequest{
		MessageID:     event.MessageID,
		DeviceID:      event.ClientState.DeviceID,
		AgentID:       event.ClientState.AgentID,
		SessionID:     event.ClientState.SessionID,
		Role:          role,
		Content:       event.Msg.Content,
		ToolCallID:    toolCallID,
		ToolCallsJSON: toolCallsJSON,
		AudioData:     audioBase64,
		AudioFormat:   audioFormat,
		AudioSize:     audioSize,
		Metadata:      metadata,
	}

	if err := w.client.SaveMessage(ctx, req); err != nil {
		log.Errorf("failed to save message, device_id: %s, message_id: %s, error: %v",
			event.ClientState.DeviceID, event.MessageID, err)
	}
}

// updateMessageAudio updates message audio in the second stage.
func (w *MessageWorker) updateMessageAudio(ctx context.Context, event *eventbus.AddMessageEvent) {
	// Convert the audio format.
	var audioBase64 string
	var audioSize int

	if len(event.AudioData) > 0 {
		var wavData []byte
		var err error

		// Select the audio conversion method by message role.
		// User ASR messages use PCMFloat32BytesToWav.
		// Assistant TTS messages use OpusFramesToWav.
		if event.Msg.Role == schema.User {
			// User messages use PCM float32.
			// event.AudioData is [][]byte, but User messages contain one complete PCM float32 byte array.
			if len(event.AudioData) > 0 {
				wavData, err = util.PCMFloat32BytesToWav(
					event.AudioData[0], // User messages contain one audio element.
					event.SampleRate,
					event.Channels)
			}
		} else {
			// Assistant messages use Opus.
			wavData, err = util.OpusFramesToWav(
				event.AudioData,
				event.SampleRate,
				event.Channels)
		}

		if err != nil {
			log.Errorf("audio conversion failed, device_id: %s, message_id: %s, role: %s, error: %v",
				event.ClientState.DeviceID, event.MessageID, event.Msg.Role, err)
			// Fall back to concatenating the raw frames.
			var fallbackData []byte
			for _, frame := range event.AudioData {
				fallbackData = append(fallbackData, frame...)
			}
			audioBase64 = base64.StdEncoding.EncodeToString(fallbackData)
			audioSize = event.AudioSize
		} else {
			audioBase64 = base64.StdEncoding.EncodeToString(wavData)
			audioSize = len(wavData)
		}
	}

	// Build the update request.
	req := &history.UpdateMessageAudioRequest{
		MessageID:   event.MessageID,
		AudioData:   audioBase64,
		AudioFormat: "wav",
		AudioSize:   audioSize,
		Metadata: map[string]interface{}{
			"tts_duration": event.TTSDuration,
		},
	}

	// Call the update endpoint.
	if err := w.client.UpdateMessageAudio(ctx, req); err != nil {
		log.Errorf("failed to update message audio, device_id: %s, message_id: %s, error: %v",
			event.ClientState.DeviceID, event.MessageID, err)
	}
}
