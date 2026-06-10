//go:build supertonic

package supertonic

import ort "github.com/yalue/onnxruntime_go"

type SpecProcessorConfig struct {
	NFFT      int     `json:"n_fft"`
	WinLength int     `json:"win_length"`
	HopLength int     `json:"hop_length"`
	NMels     int     `json:"n_mels"`
	Eps       float64 `json:"eps"`
	NormMean  float64 `json:"norm_mean"`
	NormStd   float64 `json:"norm_std"`
}

type EncoderConfig struct {
	SpecProcessor SpecProcessorConfig `json:"spec_processor"`
}

type AEConfig struct {
	SampleRate    int           `json:"sample_rate"`
	BaseChunkSize int           `json:"base_chunk_size"`
	Encoder       EncoderConfig `json:"encoder"`
}

type StyleTokenLayerConfig struct {
	NStyle        int `json:"n_style"`
	StyleValueDim int `json:"style_value_dim"`
}

type StyleEncoderConfig struct {
	StyleTokenLayer StyleTokenLayerConfig `json:"style_token_layer"`
}

type ProjOutConfig struct {
	Idim int `json:"idim"`
	Odim int `json:"odim"`
}

type TextEncoderConfig struct {
	ProjOut ProjOutConfig `json:"proj_out"`
}

type TTLConfig struct {
	ChunkCompressFactor int                `json:"chunk_compress_factor"`
	LatentDim           int                `json:"latent_dim"`
	StyleEncoder        StyleEncoderConfig `json:"style_encoder"`
	TextEncoder         TextEncoderConfig  `json:"text_encoder"`
}

type DPStyleEncoderConfig struct {
	StyleTokenLayer StyleTokenLayerConfig `json:"style_token_layer"`
}

type DPConfig struct {
	LatentDim           int                  `json:"latent_dim"`
	ChunkCompressFactor int                  `json:"chunk_compress_factor"`
	StyleEncoder        DPStyleEncoderConfig `json:"style_encoder"`
}

type Config struct {
	AE  AEConfig  `json:"ae"`
	TTL TTLConfig `json:"ttl"`
	DP  DPConfig  `json:"dp"`
}

type VoiceStyleData struct {
	StyleTTL struct {
		Data [][][]float64 `json:"data"`
		Dims []int64       `json:"dims"`
		Type string        `json:"type"`
	} `json:"style_ttl"`
	StyleDP struct {
		Data [][][]float64 `json:"data"`
		Dims []int64       `json:"dims"`
		Type string        `json:"type"`
	} `json:"style_dp"`
}

type UnicodeProcessor struct {
	indexer []int64
}

type Style struct {
	TtlTensor *ort.Tensor[float32]
	DpTensor  *ort.Tensor[float32]
}

func (s *Style) Destroy() {
	if s.TtlTensor != nil {
		s.TtlTensor.Destroy()
	}
	if s.DpTensor != nil {
		s.DpTensor.Destroy()
	}
}

type TextToSpeech struct {
	cfg           Config
	textProcessor *UnicodeProcessor
	dpOrt         *ort.DynamicAdvancedSession
	textEncOrt    *ort.DynamicAdvancedSession
	vectorEstOrt  *ort.DynamicAdvancedSession
	vocoderOrt    *ort.DynamicAdvancedSession
	SampleRate    int
	baseChunkSize int
	chunkCompress int
	ldim          int
}
