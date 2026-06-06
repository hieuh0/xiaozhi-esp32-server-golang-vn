package doubao

import "strings"

const (
	legacyDoubaoNonstreamPath = "bigmodel_nostream"
	doubaoStreamingPath       = "bigmodel_async"
)

// DoubaoV2Config DoubaoASR configuration structure
type DoubaoV2Config struct {
	AppID             string //Application ID
	AccessToken       string //access token
	WsURL             string // WebSocket URL
	ResourceID        string //Resource ID
	ModelName         string //Model name
	EndWindowSize     int    //end window size
	EnablePunc        bool   //Whether to enable punctuation
	EnableITN         bool   //Whether to enable ITN
	EnableDDC         bool   //Whether to enable DDC
	ResultType        string //Result return mode
	ShowUtterances    bool   //Whether to return clause information
	ForceToSpeechTime int    //Minimum time before forced conversion to voice
	EnableNonstream   bool   //Whether to enable bidirectional streaming optimized version
	ChunkDuration     int    //Chunking duration (milliseconds)
	Timeout           int    //Timeout (seconds)
}

// DefaultConfig default configuration
var DefaultConfig = DoubaoV2Config{
	WsURL:             "wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async",
	ResourceID:        "volc.bigasr.sauc.duration",
	ModelName:         "bigmodel",
	EndWindowSize:     800,
	EnablePunc:        true,
	EnableITN:         true,
	EnableDDC:         false,
	ResultType:        "full",
	ShowUtterances:    true,
	ForceToSpeechTime: 1000,
	EnableNonstream:   false,
	ChunkDuration:     200,
	Timeout:           30,
}

func normalizeDoubaoWsURL(wsURL string) string {
	if wsURL == "" || !strings.Contains(wsURL, legacyDoubaoNonstreamPath) {
		return wsURL
	}
	return strings.ReplaceAll(wsURL, legacyDoubaoNonstreamPath, doubaoStreamingPath)
}

// DoubaoV2Request DoubaoASR request structure
type DoubaoV2Request struct {
	User struct {
		UID string `json:"uid"`
	} `json:"user"`
	Audio struct {
		Format   string `json:"format"`
		Rate     int    `json:"rate"`
		Bits     int    `json:"bits"`
		Channel  int    `json:"channel"`
		Language string `json:"language"`
	} `json:"audio"`
	Request struct {
		ModelName         string `json:"model_name"`
		EndWindowSize     int    `json:"end_window_size"`
		EnablePunc        bool   `json:"enable_punc"`
		EnableITN         bool   `json:"enable_itn"`
		EnableDDC         bool   `json:"enable_ddc"`
		ResultType        string `json:"result_type"`
		ShowUtterances    bool   `json:"show_utterances"`
		ForceToSpeechTime int    `json:"force_to_speech_time"`
		EnableNonstream   bool   `json:"enable_nonstream"`
	} `json:"request"`
}

// DoubaoV2Response DoubaoASR response structure
type DoubaoV2Response struct {
	Code   int `json:"code"`
	Result struct {
		Text string `json:"text"`
	} `json:"result,omitempty"`
	Error string `json:"error,omitempty"`
}
