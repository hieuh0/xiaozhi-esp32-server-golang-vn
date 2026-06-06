package aliyun_qwen3

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultWsURL          = "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"
	defaultModel          = "qwen3-asr-flash-realtime"
	defaultFormat         = "pcm"
	defaultSampleRate     = 16000
	defaultLanguage       = "zh"
	defaultAutoEnd        = false
	defaultVADThreshold   = 0.0
	defaultVADSilenceMs   = 400
	defaultTimeoutSeconds = 30
)

// Config Alibaba Cloud Qwen3 ASR configuration
type Config struct {
	APIKey       string
	WsURL        string
	Model        string
	Format       string
	SampleRate   int
	Language     string
	AutoEnd      bool
	VADThreshold float64
	VADSilenceMs int
	Timeout      time.Duration
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		WsURL:        defaultWsURL,
		Model:        defaultModel,
		Format:       defaultFormat,
		SampleRate:   defaultSampleRate,
		Language:     defaultLanguage,
		AutoEnd:      defaultAutoEnd,
		VADThreshold: defaultVADThreshold,
		VADSilenceMs: defaultVADSilenceMs,
		Timeout:      time.Duration(defaultTimeoutSeconds) * time.Second,
	}
}

// ConfigFromMap merges and generates configurations from configuration maps (supports configuration files + internal control systems)
func ConfigFromMap(cfg map[string]interface{}) Config {
	conf := DefaultConfig()

	//First merge the default values ​​in the configuration file
	applyViperDefaults(&conf)

	//Compatible with old formats: if { aliyun_qwen3: { ... } } is passed in, the internal map will be taken first
	if nested, ok := cfg["aliyun_qwen3"].(map[string]interface{}); ok {
		cfg = nested
	}

	applyMapOverrides(&conf, cfg)

	//api_key allows fallback environment variables when empty
	if conf.APIKey == "" {
		conf.APIKey = os.Getenv("DASHSCOPE_API_KEY")
	}

	return conf
}

func applyViperDefaults(conf *Config) {
	const prefix = "asr.aliyun_qwen3."
	if viper.IsSet(prefix + "api_key") {
		conf.APIKey = viper.GetString(prefix + "api_key")
	}
	if viper.IsSet(prefix + "ws_url") {
		conf.WsURL = viper.GetString(prefix + "ws_url")
	}
	if viper.IsSet(prefix + "model") {
		conf.Model = viper.GetString(prefix + "model")
	}
	if viper.IsSet(prefix + "format") {
		conf.Format = viper.GetString(prefix + "format")
	}
	if viper.IsSet(prefix + "sample_rate") {
		if sr := viper.GetInt(prefix + "sample_rate"); sr > 0 {
			conf.SampleRate = sr
		}
	}
	if viper.IsSet(prefix + "language") {
		conf.Language = viper.GetString(prefix + "language")
	}
	if viper.IsSet(prefix + "auto_end") {
		conf.AutoEnd = viper.GetBool(prefix + "auto_end")
	}
	if viper.IsSet(prefix + "vad_threshold") {
		conf.VADThreshold = viper.GetFloat64(prefix + "vad_threshold")
	}
	if viper.IsSet(prefix + "vad_silence_ms") {
		conf.VADSilenceMs = viper.GetInt(prefix + "vad_silence_ms")
	}
	if viper.IsSet(prefix + "timeout") {
		if t := viper.GetInt(prefix + "timeout"); t > 0 {
			conf.Timeout = time.Duration(t) * time.Second
		}
	}
}

func applyMapOverrides(conf *Config, cfg map[string]interface{}) {
	if v, ok := cfg["api_key"].(string); ok && v != "" {
		conf.APIKey = v
	}
	if v, ok := cfg["ws_url"].(string); ok && v != "" {
		conf.WsURL = v
	}
	if v, ok := cfg["model"].(string); ok && v != "" {
		conf.Model = v
	}
	if v, ok := cfg["format"].(string); ok && v != "" {
		conf.Format = v
	}
	if v, ok := cfg["sample_rate"].(int); ok && v > 0 {
		conf.SampleRate = v
	} else if v, ok := cfg["sample_rate"].(float64); ok && v > 0 {
		conf.SampleRate = int(v)
	}
	if v, ok := cfg["language"].(string); ok && v != "" {
		conf.Language = v
	}
	if v, ok := cfg["auto_end"].(bool); ok {
		conf.AutoEnd = v
	}
	if v, ok := cfg["vad_threshold"].(float64); ok {
		conf.VADThreshold = v
	}
	if v, ok := cfg["vad_silence_ms"].(int); ok && v >= 0 {
		conf.VADSilenceMs = v
	} else if v, ok := cfg["vad_silence_ms"].(float64); ok && v >= 0 {
		conf.VADSilenceMs = int(v)
	}
	if v, ok := cfg["timeout"].(int); ok && v > 0 {
		conf.Timeout = time.Duration(v) * time.Second
	} else if v, ok := cfg["timeout"].(float64); ok && v > 0 {
		conf.Timeout = time.Duration(int(v)) * time.Second
	}
}
