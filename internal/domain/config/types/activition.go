package types

//ActivationPayload/ActivationRequest structure definition

type ActivationPayload struct {
	Algorithm    string `json:"algorithm"`
	SerialNumber string `json:"serial_number"`
	Challenge    string `json:"challenge"`
	HMAC         string `json:"hmac"`
}
