package types

const (
	EmptyReasonNone               = ""
	EmptyReasonNoServerResponse   = "no_server_response"
	EmptyReasonProviderEmptyFinal = "provider_empty_final"

	RetryReasonNone                           = ""
	RetryReasonDoubaoResponseCode45000081     = "doubao_response_code_45000081"
	RetryReasonDoubaoWaitingNextPacketTimeout = "doubao_waiting_next_packet_timeout"
	RetryReasonXunfeiServiceInstanceInvalid   = "xunfei_service_instance_invalid"
	RetryReasonAliyunQwen3ConnectionClosed    = "aliyun_qwen3_connection_closed"
)

// StreamingResult Streaming recognition result
type StreamingResult struct {
	Text        string //recognized text
	IsFinal     bool   //Is it the final result?
	Error       error  //error message
	AsrType     string //asr type
	Mode        string //mode
	EmptyReason string //Reason for empty result, only used to distinguish upstream empty result/idle when Text is empty
	RetryReason string //Recoverable error reasons, only used when you need to release the current resources and try again
}
