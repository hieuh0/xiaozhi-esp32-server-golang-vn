package manager

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"xiaozhi-esp32-server-golang/internal/components/http"
	"xiaozhi-esp32-server-golang/internal/domain/config/types"
	log "xiaozhi-esp32-server-golang/logger"
)

//HTTP interface response structure

// CheckActivationResponse Check activation status response
type CheckActivationResponse struct {
	Activated bool   `json:"activated"`
	Message   string `json:"message"`
}

// GetActivationInfoResponse Get activation information response
type GetActivationInfoResponse struct {
	Activated bool   `json:"activated"`
	Code      string `json:"code,omitempty"` //Modified to string type to match backend API
	Challenge string `json:"challenge,omitempty"`
	Message   string `json:"message,omitempty"`
}

// ActivateDeviceRequest device activation request
type ActivateDeviceRequest struct {
	DeviceId     string `json:"device_id"`
	ClientId     string `json:"client_id"`
	Code         string `json:"code"`
	Challenge    string `json:"challenge"`
	Algorithm    string `json:"algorithm"`
	SerialNumber string `json:"serial_number"`
	Hmac         string `json:"hmac"`
}

// ActivateDeviceResponse device activation response
type ActivateDeviceResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// IsDeviceActivated checks whether the device is activated
func (am *ConfigManager) IsDeviceActivated(ctx context.Context, deviceId string, clientId string) (bool, error) {
	//Directly call the HTTP interface of the back-end management system
	activated, err := am.callCheckActivationAPI(ctx, deviceId, clientId)
	if err != nil {
		log.Log().Errorf("Check device %s activation status failed: %v", deviceId, err)
		return false, err
	}

	log.Log().Debugf("Device %s Activation status: %v", deviceId, activated)
	return activated, nil
}

// GetActivationInfo Gets device activation information
func (am *ConfigManager) GetActivationInfo(ctx context.Context, deviceId string, clientId string) (string, string, string, int) {
	//Directly call the HTTP interface of the back-end management system
	activated, codeStr, challenge, message, err := am.callGetActivationInfoAPI(ctx, deviceId, clientId)
	if err != nil {
		log.Log().Errorf("Failed to obtain device %s activation information: %v", deviceId, err)
		return "", "", "", 0
	}

	//If the device is activated, return directly
	if activated {
		log.Log().Debugf("Device %s is activated", deviceId)
		return "", "", message, 0
	}

	//Check if Challenge is empty
	if challenge == "" {
		log.Log().Errorf("The Challenge field of device %s is empty", deviceId)
		return "", "", "Challenge field is empty, please contact administrator", 0
	}

	//The device is not activated, return activation information
	timeoutMs := 300000 // 5 minutes in milliseconds
	log.Log().Debugf("Get device %s activation information: code=%s, challenge=%s", deviceId, codeStr, challenge)
	if codeStr == "" {
		log.Log().Warnf("Device %s activation code is empty", deviceId)
	}

	return codeStr, challenge, message, timeoutMs
}

// VerifyChallenge Verify challenge code and HMAC
func (am *ConfigManager) VerifyChallenge(ctx context.Context, deviceId string, clientId string, activationPayload types.ActivationPayload) (bool, error) {
	//Verify HMAC (if HMAC is provided)
	if activationPayload.HMAC != "" {
		if !am.verifyHMAC(activationPayload.Challenge, activationPayload.HMAC) {
			log.Log().Warnf("Device %s HMAC verification failed", deviceId)
			return false, fmt.Errorf("HMAC verification failed")
		}
	}

	//Directly call the activation interface of the back-end management system
	verified, err := am.callActivateDeviceAPI(ctx, deviceId, clientId, activationPayload)
	if err != nil {
		log.Log().Errorf("Device activation failed: %v", err)
		return false, err
	}

	if verified {
		log.Log().Infof("Device %s activation verification successful", deviceId)
	}

	return verified, nil
}

// verifyHMAC Verify HMAC signature
func (am *ConfigManager) verifyHMAC(challenge, providedHmac string) bool {
	//Here you can configure the key according to actual needs
	//Temporarily use an empty key. In actual applications, it should be obtained from the configuration.
	secretKey := ""

	if secretKey == "" {
		//If there is no configured key, pass the verification directly
		return true
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(challenge))
	expectedHmac := hex.EncodeToString(mac.Sum(nil))

	return expectedHmac == providedHmac
}

//HTTP API calling method

// callCheckActivationAPI calls the check activation status interface
func (am *ConfigManager) callCheckActivationAPI(ctx context.Context, deviceId, clientId string) (bool, error) {
	var response CheckActivationResponse

	//Send HTTP request
	err := am.client.DoRequest(ctx, http.RequestOptions{
		Method: "GET",
		Path:   "/api/internal/device/check-activation",
		QueryParams: map[string]string{
			"device_id": deviceId,
			"client_id": clientId,
		},
		Response: &response,
	})
	if err != nil {
		return false, fmt.Errorf("Request failed: %w", err)
	}

	log.Log().Debugf("Check activation status response: %+v", response)
	return response.Activated, nil
}

// callGetActivationInfoAPI calls to get activation information interface
func (am *ConfigManager) callGetActivationInfoAPI(ctx context.Context, deviceId, clientId string) (bool, string, string, string, error) {
	var response GetActivationInfoResponse

	//Send HTTP request
	err := am.client.DoRequest(ctx, http.RequestOptions{
		Method: "GET",
		Path:   "/api/internal/device/activation-info",
		QueryParams: map[string]string{
			"device_id": deviceId,
			"client_id": clientId,
		},
		Response: &response,
	})
	if err != nil {
		return false, "", "", "", fmt.Errorf("Request failed: %w", err)
	}

	log.Log().Debugf("Get activation information response: %+v", response)

	if response.Activated {
		return true, "", "", response.Message, nil
	}

	return false, response.Code, response.Challenge, response.Message, nil
}

// callActivateDeviceAPI calls the device activation interface
func (am *ConfigManager) callActivateDeviceAPI(ctx context.Context, deviceId, clientId string, activationPayload types.ActivationPayload) (bool, error) {
	//Build request body
	request := ActivateDeviceRequest{
		DeviceId:     deviceId,
		ClientId:     clientId,
		Challenge:    activationPayload.Challenge,
		Algorithm:    activationPayload.Algorithm,
		SerialNumber: activationPayload.SerialNumber,
		Hmac:         activationPayload.HMAC,
	}

	var response ActivateDeviceResponse

	//Send HTTP request
	err := am.client.DoRequest(ctx, http.RequestOptions{
		Method:   "POST",
		Path:     "/api/internal/device/activate",
		Body:     request,
		Response: &response,
	})
	if err != nil {
		return false, fmt.Errorf("Request failed: %w", err)
	}

	log.Log().Debugf("Device activation response: %+v", response)

	if !response.Success {
		return false, nil
	}

	return response.Success, nil
}
