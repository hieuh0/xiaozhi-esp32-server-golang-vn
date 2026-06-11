//go:build supertonic

package supertonic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// ONNX Runtime must only be initialized once per process.
var onnxInitOnce sync.Once
var onnxInitErr error

// InitializeONNXRuntime locates and initializes the ONNX Runtime shared library.
// Safe to call multiple times — initialization only happens once.
func InitializeONNXRuntime() error {
	onnxInitOnce.Do(func() {
		libPath := os.Getenv("ONNXRUNTIME_LIB_PATH")
		if libPath == "" {
			candidates := []string{
				"/opt/homebrew/opt/onnxruntime/lib/libonnxruntime.dylib",
				"/usr/local/opt/onnxruntime/lib/libonnxruntime.dylib",
				"/opt/homebrew/lib/libonnxruntime.dylib",
				"/usr/local/lib/libonnxruntime.dylib",
				"/usr/local/lib/libonnxruntime.so",
				"/usr/lib/libonnxruntime.so",
			}
			for _, candidate := range candidates {
				if _, err := os.Stat(candidate); err == nil {
					libPath = candidate
					break
				}
			}
			if libPath == "" {
				libPath = "/usr/local/lib/libonnxruntime.so"
			}
		}
		ort.SetSharedLibraryPath(libPath)
		if err := ort.InitializeEnvironment(); err != nil {
			onnxInitErr = fmt.Errorf("failed to initialize ONNX Runtime: %w\nHint: install ONNX Runtime (macOS: brew install onnxruntime) or set ONNXRUNTIME_LIB_PATH", err)
		}
	})
	return onnxInitErr
}

func LoadCfgs(onnxDir string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(onnxDir, "tts.json"))
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func LoadVoiceStyle(voiceStylePaths []string, verbose bool) (*Style, error) {
	bsz := len(voiceStylePaths)

	firstData, err := os.ReadFile(voiceStylePaths[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read voice style file: %w", err)
	}
	var firstStyle VoiceStyleData
	if err := json.Unmarshal(firstData, &firstStyle); err != nil {
		return nil, fmt.Errorf("failed to parse voice style JSON: %w", err)
	}

	ttlDims := firstStyle.StyleTTL.Dims
	dpDims := firstStyle.StyleDP.Dims
	ttlDim1, ttlDim2 := ttlDims[1], ttlDims[2]
	dpDim1, dpDim2 := dpDims[1], dpDims[2]

	ttlFlat := make([]float32, int(int64(bsz)*ttlDim1*ttlDim2))
	dpFlat := make([]float32, int(int64(bsz)*dpDim1*dpDim2))

	for i := 0; i < bsz; i++ {
		data, err := os.ReadFile(voiceStylePaths[i])
		if err != nil {
			return nil, fmt.Errorf("failed to read voice style file: %w", err)
		}
		var voiceStyle VoiceStyleData
		if err := json.Unmarshal(data, &voiceStyle); err != nil {
			return nil, fmt.Errorf("failed to parse voice style JSON: %w", err)
		}

		ttlOffset := int(int64(i) * ttlDim1 * ttlDim2)
		idx := 0
		for _, batch := range voiceStyle.StyleTTL.Data {
			for _, row := range batch {
				for _, val := range row {
					ttlFlat[ttlOffset+idx] = float32(val)
					idx++
				}
			}
		}

		dpOffset := int(int64(i) * dpDim1 * dpDim2)
		idx = 0
		for _, batch := range voiceStyle.StyleDP.Data {
			for _, row := range batch {
				for _, val := range row {
					dpFlat[dpOffset+idx] = float32(val)
					idx++
				}
			}
		}
	}

	ttlTensor, err := ort.NewTensor([]int64{int64(bsz), ttlDim1, ttlDim2}, ttlFlat)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTL tensor: %w", err)
	}
	dpTensor, err := ort.NewTensor([]int64{int64(bsz), dpDim1, dpDim2}, dpFlat)
	if err != nil {
		ttlTensor.Destroy()
		return nil, fmt.Errorf("failed to create DP tensor: %w", err)
	}

	if verbose {
		fmt.Printf("Loaded %d voice styles\n", bsz)
	}
	return &Style{TtlTensor: ttlTensor, DpTensor: dpTensor}, nil
}

func LoadTextToSpeech(onnxDir string, useGPU bool, cfg Config) (*TextToSpeech, error) {
	if useGPU {
		return nil, fmt.Errorf("GPU mode is not supported")
	}

	dpOrt, err := ort.NewDynamicAdvancedSession(
		filepath.Join(onnxDir, "duration_predictor.onnx"),
		[]string{"text_ids", "style_dp", "text_mask"}, []string{"duration"}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load duration predictor: %w", err)
	}

	textEncOrt, err := ort.NewDynamicAdvancedSession(
		filepath.Join(onnxDir, "text_encoder.onnx"),
		[]string{"text_ids", "style_ttl", "text_mask"}, []string{"text_emb"}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load text encoder: %w", err)
	}

	vectorEstOrt, err := ort.NewDynamicAdvancedSession(
		filepath.Join(onnxDir, "vector_estimator.onnx"),
		[]string{"noisy_latent", "text_emb", "style_ttl", "latent_mask", "text_mask", "current_step", "total_step"},
		[]string{"denoised_latent"}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load vector estimator: %w", err)
	}

	vocoderOrt, err := ort.NewDynamicAdvancedSession(
		filepath.Join(onnxDir, "vocoder.onnx"),
		[]string{"latent"}, []string{"wav_tts"}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load vocoder: %w", err)
	}

	textProcessor, err := NewUnicodeProcessor(filepath.Join(onnxDir, "unicode_indexer.json"))
	if err != nil {
		return nil, err
	}

	return &TextToSpeech{
		cfg:           cfg,
		textProcessor: textProcessor,
		dpOrt:         dpOrt,
		textEncOrt:    textEncOrt,
		vectorEstOrt:  vectorEstOrt,
		vocoderOrt:    vocoderOrt,
		SampleRate:    cfg.AE.SampleRate,
		baseChunkSize: cfg.AE.BaseChunkSize,
		chunkCompress: cfg.TTL.ChunkCompressFactor,
		ldim:          cfg.TTL.LatentDim,
	}, nil
}

func ArrayToTensor(array [][][]float64, shape []int64) *ort.Tensor[float32] {
	totalSize := int64(1)
	for _, dim := range shape {
		totalSize *= dim
	}
	flat := make([]float32, totalSize)
	idx := 0
	for b := range array {
		for d := range array[b] {
			for _, val := range array[b][d] {
				flat[idx] = float32(val)
				idx++
			}
		}
	}
	tensor, err := ort.NewTensor(shape, flat)
	if err != nil {
		panic(err)
	}
	return tensor
}

func IntArrayToTensor(array [][]int64, shape []int64) *ort.Tensor[int64] {
	totalSize := int64(1)
	for _, dim := range shape {
		totalSize *= dim
	}
	flat := make([]int64, totalSize)
	idx := 0
	for b := range array {
		for _, val := range array[b] {
			flat[idx] = val
			idx++
		}
	}
	tensor, err := ort.NewTensor(shape, flat)
	if err != nil {
		panic(err)
	}
	return tensor
}
