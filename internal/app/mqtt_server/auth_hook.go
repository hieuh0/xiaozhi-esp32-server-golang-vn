package mqtt_server

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"encoding/json"

	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	mqttServer "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/spf13/viper"
)

// AuthHook implements custom authentication for regular users and administrators.
// Regular users: the username is a base64-encoded {"ip":"1.202.193.194"} object,
// and the password is an HMAC-SHA256 signature.
// Administrator: username admin, password shijingbo!@#
type AuthHook struct {
	mqttServer.HookBase
}

func (h *AuthHook) ID() string {
	return "custom-auth-hook"
}

func (h *AuthHook) Provides(b byte) bool {
	return b == mqttServer.OnConnectAuthenticate
}

func (h *AuthHook) OnConnectAuthenticate(cl *mqttServer.Client, pk packets.Packet) bool {
	// Check whether authentication is enabled.
	enableAuth := viper.GetBool("mqtt_server.enable_auth")
	if !enableAuth {
		//log.Infof("MQTT authentication is disabled; allowing all connections")
		return true
	}

	username := string(pk.Connect.Username)
	password := string(pk.Connect.Password)
	clientId := string(pk.Connect.ClientIdentifier)

	// Validate the administrator credentials.
	adminUsername := configuredAdminUsername()
	adminPassword := configuredAdminPassword()
	if username == adminUsername && password == adminPassword {
		log.Infof("Administrator login succeeded: %s", username)
		return true
	}
	if username == adminUsername {
		log.Warnf("MQTT administrator login failed: username=%s, clientId=%s, reason=incorrect password", username, clientId)
		return false
	}

	// Validate regular users with the signature-based flow.
	signatureKey := viper.GetString("mqtt_server.signature_key")
	if signatureKey != "" {
		credentialInfo, err := util.ValidateMqttCredentials(clientId, username, password, signatureKey)
		//log.Infof("Starting MQTT user validation: clientId=%s, username=%s, password=%s, signatureKey=%s",
		//	clientId, username, password, signatureKey)
		//log.Infof("Starting MQTT user validation: credentialInfo=%+v", credentialInfo)

		if err != nil {
			log.Warnf("MQTT credential validation failed: username=%s, clientId=%s, err=%v", username, clientId, err)
			return false
		}

		log.Infof("MQTT user validation succeeded: groupId=%s, macAddress=%s, uuid=%s",
			credentialInfo.GroupId, credentialInfo.MacAddress, credentialInfo.UUID)
		return true
	}

	// Fall back to the legacy AES validation flow when no signature key is configured.
	log.Warnf("OTA signature key is not configured; using AES validation")
	return h.validateWithAes(username, password)
}

// validateWithAes validates the password with AES for backward compatibility.
func (h *AuthHook) validateWithAes(username, password string) bool {
	// Validate the regular user payload.
	decoded, err := base64.StdEncoding.DecodeString(username)
	if err != nil {
		return false
	}
	var userInfo map[string]string
	if err := json.Unmarshal(decoded, &userInfo); err != nil {
		return false
	}
	if _, ok := userInfo["ip"]; !ok {
		return false
	}
	// Verify that password is the AES-encrypted username.
	if !checkAesPassword(username, password) {
		return false
	}
	return true
}

// checkAesPassword verifies password against AES-ECB-encrypted base64(username).
func checkAesPassword(username, password string) bool {
	key := []byte("xiaozhi_aes_key_1") // 16-byte key; configure this in production.
	ciphertext, err := aesEncryptECB([]byte(username), key)
	if err != nil {
		return false
	}
	cipherBase64 := base64.StdEncoding.EncodeToString(ciphertext)
	return cipherBase64 == password
}

// aesEncryptECB encrypts data with AES-ECB.
func aesEncryptECB(src, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	// Apply PKCS7 padding.
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	src = append(src, padtext...)
	encrypted := make([]byte, len(src))
	for bs, be := 0, blockSize; bs < len(src); bs, be = bs+blockSize, be+blockSize {
		block.Encrypt(encrypted[bs:be], src[bs:be])
	}
	return encrypted, nil
}
