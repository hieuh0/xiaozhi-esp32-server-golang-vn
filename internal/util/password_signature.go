package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// GeneratePasswordSignature generates a password signature.
// Uses HMAC-SHA256 based on clientId + '|' + username and a signature key.
func GeneratePasswordSignature(data, key string) string {
	// Generate signature using HMAC-SHA256
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	signature := h.Sum(nil)

	// Return the base64-encoded signature
	return base64.StdEncoding.EncodeToString(signature)
}

// ValidateMqttCredentials validates MQTT credentials.
// Implemented according to the provided JavaScript validation logic.
func ValidateMqttCredentials(clientId, username, password, signatureKey string) (*MqttCredentialInfo, error) {
	// Validate the signature key
	if signatureKey == "" {
		return nil, fmt.Errorf("missing signature key configuration")
	}

	// Validate clientId
	if clientId == "" {
		return nil, fmt.Errorf("clientId must be a non-empty string")
	}

	// Validate clientId format (must contain @@@ separator)
	clientIdParts := strings.Split(clientId, "@@@")
	if len(clientIdParts) != 3 {
		return nil, fmt.Errorf("clientId format error: must contain @@@ separator")
	}

	// Validate username
	if username == "" {
		return nil, fmt.Errorf("username must be a non-empty string")
	}

	// Attempt to decode username (should be base64-encoded JSON)
	var userData map[string]interface{}
	decodedUsername, err := base64.StdEncoding.DecodeString(username)
	if err != nil {
		return nil, fmt.Errorf("username is not valid base64: %v", err)
	}

	if err := json.Unmarshal(decodedUsername, &userData); err != nil {
		return nil, fmt.Errorf("username is not valid base64-encoded JSON: %v", err)
	}

	// Validate password signature
	signatureData := clientId + "|" + username
	expectedSignature := GeneratePasswordSignature(signatureData, signatureKey)
	if password != expectedSignature {
		return nil, fmt.Errorf("password signature validation failed")
	}

	// Parse information from clientId
	groupId := clientIdParts[0]
	macAddress := strings.ReplaceAll(clientIdParts[1], "_", ":")
	uuid := clientIdParts[2]

	// Return parsed information on successful validation
	return &MqttCredentialInfo{
		GroupId:    groupId,
		MacAddress: macAddress,
		UUID:       uuid,
		UserData:   userData,
	}, nil
}

// MqttCredentialInfo holds parsed MQTT credential information
type MqttCredentialInfo struct {
	GroupId    string                 `json:"groupId"`
	MacAddress string                 `json:"macAddress"`
	UUID       string                 `json:"uuid"`
	UserData   map[string]interface{} `json:"userData"`
}

// GenerateMqttCredentials generates MQTT credentials.
// Used by the OTA interface to generate MQTT connection information.
func GenerateMqttCredentials(deviceId, clientId, ip, signatureKey string) (*MqttCredentials, error) {
	// Process deviceId (replace colons with underscores)
	deviceId = strings.ReplaceAll(deviceId, ":", "_")

	// Build username data (includes IP information)
	userName := struct {
		Ip string `json:"ip"`
	}{
		Ip: ip,
	}
	userNameJson, err := json.Marshal(userName)
	if err != nil {
		return nil, fmt.Errorf("username serialization failed: %v", err)
	}
	base64UserName := base64.StdEncoding.EncodeToString(userNameJson)

	// Build clientId in format: GID_test@@@deviceId@@@clientId
	mqttClientId := fmt.Sprintf("GID_test@@@%s@@@%s", deviceId, clientId)

	// Generate password signature
	var pwd string
	if signatureKey != "" {
		// Use signature key to generate password
		signatureData := mqttClientId + "|" + base64UserName
		pwd = GeneratePasswordSignature(signatureData, signatureKey)
	} else {
		// Fallback to old logic if no signature key is configured
		pwd = Sha256Digest([]byte(mqttClientId))
	}

	return &MqttCredentials{
		ClientId: mqttClientId,
		Username: base64UserName,
		Password: pwd,
	}, nil
}

// MqttCredentials holds MQTT connection credentials
type MqttCredentials struct {
	ClientId string `json:"client_id"`
	Username string `json:"username"`
	Password string `json:"password"`
}
