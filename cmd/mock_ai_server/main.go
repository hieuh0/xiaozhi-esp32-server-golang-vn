package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"xiaozhi-esp32-server-golang/internal/util"
)

type serverConfig struct {
	addr          string
	asrText       string
	asrDelayMs    int
	llmReply      string
	llmFirstDelay int
	llmChunkDelay int
	ttsMode       string
	ttsDurationMs int
	ttsSampleRate int
	ttsFirstDelay int
	ttsFrameDelay int
	asrReqID      uint64
	llmReqID      uint64
	ttsReqID      uint64
}

type openAIChatRequest struct {
	Model    string `json:"model"`
	Stream   bool   `json:"stream"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int `json:"index"`
		FinishReason any `json:"finish_reason"`
		Message      struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type openAITTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
}

type funasrRequest struct {
	IsSpeaking bool `json:"is_speaking"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	cfg := serverConfig{}
	flag.StringVar(&cfg.addr, "addr", ":18080", "server listen address")
	flag.StringVar(&cfg.asrText, "asr-text", "Hello, this is mock ASR result", "ASR final text")
	flag.IntVar(&cfg.asrDelayMs, "asr-delay-ms", 120, "ASR final response delay in milliseconds")
	flag.StringVar(&cfg.llmReply, "llm-reply", "Hello, I'm mock LLM, glad to serve you.", "LLM reply text")
	flag.IntVar(&cfg.llmFirstDelay, "llm-first-delay-ms", 80, "LLM first token delay in milliseconds")
	flag.IntVar(&cfg.llmChunkDelay, "llm-chunk-delay-ms", 40, "LLM chunk delay in milliseconds when stream=true")
	flag.StringVar(&cfg.ttsMode, "tts-mode", "silence", "TTS audio mode: silence|beep")
	flag.IntVar(&cfg.ttsDurationMs, "tts-duration-ms", 1200, "TTS audio duration in milliseconds")
	flag.IntVar(&cfg.ttsSampleRate, "tts-sample-rate", 16000, "TTS output sample rate")
	flag.IntVar(&cfg.ttsFirstDelay, "tts-first-delay-ms", 60, "TTS first frame delay in milliseconds")
	flag.IntVar(&cfg.ttsFrameDelay, "tts-frame-delay-ms", 0, "reserved for future frame delay")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", cfg.handleHealth)
	mux.HandleFunc("/asr/", cfg.handleFunASRWebSocket)
	mux.HandleFunc("/", cfg.handleRoot)
	mux.HandleFunc("/v1/chat/completions", cfg.handleChatCompletions)
	mux.HandleFunc("/v1/audio/speech", cfg.handleTTSSpeech)

	log.Printf("mock ai server start at %s", cfg.addr)
	log.Printf("endpoints: /asr/ (ws), /v1/chat/completions (http), /v1/audio/speech (http), /healthz")
	if err := http.ListenAndServe(cfg.addr, mux); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

func (c *serverConfig) handleRoot(w http.ResponseWriter, r *http.Request) {
	if websocket.IsWebSocketUpgrade(r) {
		c.handleFunASRWebSocket(w, r)
		return
	}
	http.NotFound(w, r)
}

func (c *serverConfig) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": time.Now().Unix()})
}

func (c *serverConfig) handleFunASRWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("asr ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	asrID := atomic.AddUint64(&c.asrReqID, 1)
	log.Printf("[ASR-%d] ws connected from %s", asrID, r.RemoteAddr)

	var audioPackets int
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[ASR-%d] read finished: %v", asrID, err)
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			audioPackets++
		case websocket.TextMessage:
			var req funasrRequest
			_ = json.Unmarshal(data, &req)
			if !req.IsSpeaking {
				if c.asrDelayMs > 0 {
					time.Sleep(time.Duration(c.asrDelayMs) * time.Millisecond)
				}
				resp := map[string]any{
					"text":       c.asrText,
					"is_final":   true,
					"wav_name":   "mock",
					"timestamp":  fmt.Sprintf("%d", time.Now().UnixMilli()),
					"mode":       "online",
					"confidence": 0.99,
				}
				if err := conn.WriteJSON(resp); err != nil {
					log.Printf("[ASR-%d] write final failed: %v", asrID, err)
					return
				}
				log.Printf("[ASR-%d] sent final text (audio_packets=%d)", asrID, audioPackets)
			}
		}
	}
}

func (c *serverConfig) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	llmID := atomic.AddUint64(&c.llmReqID, 1)
	defer r.Body.Close()

	var req openAIChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	reply := c.llmReply
	if len(req.Messages) > 0 {
		last := strings.TrimSpace(req.Messages[len(req.Messages)-1].Content)
		if last != "" {
			reply = fmt.Sprintf("%s | echo: %s", c.llmReply, last)
		}
	}

	if req.Stream {
		c.handleChatStream(w, req.Model, reply)
		log.Printf("[LLM-%d] stream reply sent", llmID)
		return
	}

	if c.llmFirstDelay > 0 {
		time.Sleep(time.Duration(c.llmFirstDelay) * time.Millisecond)
	}

	resp := openAIChatResponse{
		ID:      fmt.Sprintf("chatcmpl-mock-%d", llmID),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
	}
	choice := struct {
		Index        int `json:"index"`
		FinishReason any `json:"finish_reason"`
		Message      struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}{
		Index:        0,
		FinishReason: "stop",
	}
	choice.Message.Role = "assistant"
	choice.Message.Content = reply
	resp.Choices = append(resp.Choices, choice)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	log.Printf("[LLM-%d] non-stream reply sent", llmID)
}

func (c *serverConfig) handleChatStream(w http.ResponseWriter, model, reply string) {
	if c.llmFirstDelay > 0 {
		time.Sleep(time.Duration(c.llmFirstDelay) * time.Millisecond)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}

	chunks := splitByRune(reply, 12)
	for _, chunk := range chunks {
		payload := map[string]any{
			"id":      "chatcmpl-mock-stream",
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{"content": chunk},
			}},
		}
		buf, _ := json.Marshal(payload)
		_, _ = w.Write([]byte("data: " + string(buf) + "\n\n"))
		flusher.Flush()
		if c.llmChunkDelay > 0 {
			time.Sleep(time.Duration(c.llmChunkDelay) * time.Millisecond)
		}
	}

	finalPayload := map[string]any{
		"id":      "chatcmpl-mock-stream",
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"}},
	}
	buf, _ := json.Marshal(finalPayload)
	_, _ = w.Write([]byte("data: " + string(buf) + "\n\n"))
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

func (c *serverConfig) handleTTSSpeech(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ttsID := atomic.AddUint64(&c.ttsReqID, 1)
	defer r.Body.Close()

	var req openAITTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if c.ttsFirstDelay > 0 {
		time.Sleep(time.Duration(c.ttsFirstDelay) * time.Millisecond)
	}
	_ = c.ttsFrameDelay // Reserved for future real frame-by-frame streaming responses.

	responseFormat := strings.ToLower(strings.TrimSpace(req.ResponseFormat))
	if responseFormat == "" {
		responseFormat = "wav"
	}

	switch responseFormat {
	case "opus":
		opusBytes, err := synthOggOpus(c.ttsMode, c.ttsSampleRate, c.ttsDurationMs)
		if err != nil {
			http.Error(w, fmt.Sprintf("synth opus failed: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "audio/ogg")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(opusBytes)))
		_, _ = w.Write(opusBytes)
		log.Printf("[TTS-%d] speech sent (format=%s text_len=%d bytes=%d)", ttsID, responseFormat, len(req.Input), len(opusBytes))
	default:
		wavBytes := synthWAV(c.ttsMode, c.ttsSampleRate, c.ttsDurationMs)
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(wavBytes)))
		_, _ = w.Write(wavBytes)
		log.Printf("[TTS-%d] speech sent (format=%s text_len=%d bytes=%d)", ttsID, responseFormat, len(req.Input), len(wavBytes))
	}
}

func splitByRune(s string, size int) []string {
	if size <= 0 {
		return []string{s}
	}
	runes := []rune(s)
	if len(runes) == 0 {
		return []string{""}
	}
	out := make([]string, 0, (len(runes)+size-1)/size)
	for i := 0; i < len(runes); i += size {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[i:end]))
	}
	return out
}

func synthWAV(mode string, sampleRate, durationMs int) []byte {
	pcm := synthPCM16(mode, sampleRate, durationMs)
	return encodeWAVPCM16(pcm, sampleRate)
}

func synthOggOpus(mode string, sampleRate, durationMs int) ([]byte, error) {
	sampleRate = util.NormalizeOpusSampleRate(sampleRate)
	pcm := synthPCM16(mode, sampleRate, durationMs)
	return util.PCM16ToOggOpus(pcm, sampleRate, 1, 20)
}

func synthPCM16(mode string, sampleRate, durationMs int) []int16 {
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	if durationMs <= 0 {
		durationMs = 1000
	}
	n := sampleRate * durationMs / 1000
	pcm := make([]int16, n)

	if mode == "beep" {
		freq := 440.0
		for i := 0; i < n; i++ {
			v := math.Sin(2 * math.Pi * freq * float64(i) / float64(sampleRate))
			pcm[i] = int16(v * 9000)
		}
	}
	return pcm
}

func encodeWAVPCM16(samples []int16, sampleRate int) []byte {
	const channels = 1
	const bitsPerSample = 16
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := len(samples) * 2
	riffSize := 36 + dataSize

	buf := &bytes.Buffer{}
	_, _ = buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(riffSize))
	_, _ = buf.WriteString("WAVE")
	_, _ = buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(channels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(byteRate))
	_ = binary.Write(buf, binary.LittleEndian, uint16(blockAlign))
	_ = binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))
	_, _ = buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataSize))
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}
	return buf.Bytes()
}
