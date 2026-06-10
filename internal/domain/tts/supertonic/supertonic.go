//go:build supertonic

package supertonic

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	opus "gopkg.in/hraban/opus.v2"
	"xiaozhi-esp32-server-golang/internal/util"
)

type SupertonicTTSProvider struct {
	OnnxDir         string
	Voice           string
	Lang            string
	Steps           int
	Speed           float32
	SilenceDuration float32
	FrameDuration   int

	once    sync.Once
	tts     *TextToSpeech
	initErr error
	mu      sync.Mutex
	style   *Style
}

func NewSupertonicTTSProvider(config map[string]interface{}) *SupertonicTTSProvider {
	onnxDir, _ := config["onnx_dir"].(string)
	voice, _ := config["voice"].(string)
	lang, _ := config["lang"].(string)
	frameDuration, _ := config["frame_duration"].(float64)
	steps, _ := config["steps"].(float64)
	speed, _ := config["speed"].(float64)
	silence, _ := config["silence_duration"].(float64)
	voiceJsonPath, _ := config["voice_json_path"].(string)

	if voice == "" {
		voice = "M1"
	}
	if lang == "" {
		lang = "na"
	}
	if steps == 0 {
		steps = 8
	}
	if speed == 0 {
		speed = 1.0
	}
	if silence == 0 {
		silence = 0.3
	}
	if frameDuration == 0 {
		frameDuration = 60
	}
	// custom voice path overrides preset
	if voice == "custom" && voiceJsonPath != "" {
		voice = voiceJsonPath
	}

	return &SupertonicTTSProvider{
		OnnxDir:         onnxDir,
		Voice:           voice,
		Lang:            lang,
		Steps:           int(steps),
		Speed:           float32(speed),
		SilenceDuration: float32(silence),
		FrameDuration:   int(frameDuration),
	}
}

func (p *SupertonicTTSProvider) initOnce() {
	if err := InitializeONNXRuntime(); err != nil {
		p.initErr = fmt.Errorf("supertonic: init ONNX runtime: %w", err)
		return
	}
	cfg, err := LoadCfgs(p.OnnxDir)
	if err != nil {
		p.initErr = fmt.Errorf("supertonic: load configs from %q: %w", p.OnnxDir, err)
		return
	}
	p.tts, p.initErr = LoadTextToSpeech(p.OnnxDir, false, cfg)
	if p.initErr != nil {
		p.initErr = fmt.Errorf("supertonic: load TTS model: %w", p.initErr)
	}
}

func (p *SupertonicTTSProvider) resolveVoicePath(voice string) string {
	// absolute/relative path or JSON file → use as-is
	if strings.Contains(voice, "/") || strings.Contains(voice, "\\") || strings.HasSuffix(voice, ".json") {
		return voice
	}
	// preset name (M1, F1, …) → assets directory inside OnnxDir
	return filepath.Join(p.OnnxDir, "assets", "voices", voice+".json")
}

// loadStyleLocked reloads the voice style. Must be called with p.mu held.
func (p *SupertonicTTSProvider) loadStyleLocked(voicePath string) error {
	style, err := LoadVoiceStyle([]string{voicePath}, false)
	if err != nil {
		return fmt.Errorf("supertonic: load voice style %q: %w", voicePath, err)
	}
	p.style = style
	return nil
}

func (p *SupertonicTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	p.once.Do(p.initOnce)
	if p.initErr != nil {
		return nil, p.initErr
	}

	p.mu.Lock()
	if p.style == nil {
		if err := p.loadStyleLocked(p.resolveVoicePath(p.Voice)); err != nil {
			p.mu.Unlock()
			return nil, err
		}
	}
	style := p.style
	voiceName := p.Voice
	p.mu.Unlock()

	if style == nil {
		return nil, fmt.Errorf("supertonic: failed to load voice style for %q", voiceName)
	}

	pcmFloat32, _, err := p.tts.Call(text, p.Lang, style, p.Steps, p.Speed, p.SilenceDuration)
	if err != nil {
		return nil, fmt.Errorf("supertonic: synthesis failed: %w", err)
	}

	resampled := util.ResampleLinearFloat32(pcmFloat32, p.tts.SampleRate, sampleRate)
	int16Pcm := util.Float32SliceToInt16Slice(resampled)
	return encodeOpusFrames(int16Pcm, sampleRate, channels, frameDuration)
}

func (p *SupertonicTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	outputChan := make(chan []byte, 100)
	go func() {
		defer close(outputChan)
		frames, err := p.TextToSpeech(ctx, text, sampleRate, channels, frameDuration)
		if err != nil {
			return
		}
		for _, f := range frames {
			select {
			case <-ctx.Done():
				return
			case outputChan <- f:
			}
		}
	}()
	return outputChan, nil
}

func (p *SupertonicTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	voice, _ := voiceConfig["voice"].(string)
	if voice == "" {
		return fmt.Errorf("supertonic: SetVoice requires non-empty voice")
	}
	// mirror the same custom-voice resolution used in NewSupertonicTTSProvider
	if voice == "custom" {
		jsonPath, _ := voiceConfig["voice_json_path"].(string)
		if jsonPath == "" {
			return fmt.Errorf("supertonic: SetVoice custom voice requires voice_json_path")
		}
		voice = jsonPath
	}
	p.once.Do(p.initOnce)
	if p.initErr != nil {
		return p.initErr
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.loadStyleLocked(p.resolveVoicePath(voice)); err != nil {
		return err
	}
	p.Voice = voice // write inside lock — safe against concurrent TextToSpeech reads
	return nil
}

func (p *SupertonicTTSProvider) Close() error { return nil }

func (p *SupertonicTTSProvider) IsValid() bool { return p != nil && p.OnnxDir != "" }

func encodeOpusFrames(pcm []int16, sampleRate, channels, frameDurationMs int) ([][]byte, error) {
	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppAudio)
	if err != nil {
		return nil, fmt.Errorf("supertonic: create opus encoder: %w", err)
	}
	frameSize := sampleRate * frameDurationMs / 1000
	var frames [][]byte
	buf := make([]byte, 4000)
	for offset := 0; offset+frameSize <= len(pcm); offset += frameSize {
		n, err := enc.Encode(pcm[offset:offset+frameSize], buf)
		if err != nil {
			return nil, fmt.Errorf("supertonic: opus encode: %w", err)
		}
		frame := make([]byte, n)
		copy(frame, buf[:n])
		frames = append(frames, frame)
	}
	return frames, nil
}
