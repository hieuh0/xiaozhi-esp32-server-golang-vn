package mcp_market

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

const marketSecretEnv = "MCP_MARKET_SECRET_KEY"

func loadSecretKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(marketSecretEnv))
	if raw == "" {
		return nil, fmt.Errorf("environment variable %s is not set", marketSecretEnv)
	}

	if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
		rawKey := decoded
		if len(rawKey) == 32 {
			return rawKey, nil
		}
	}

	rawKey := []byte(raw)
	if len(rawKey) != 32 {
		return nil, fmt.Errorf("%s must be a 32-byte plaintext or Base64(32-byte) key", marketSecretEnv)
	}

	return rawKey, nil
}

func EncryptText(plain string) (ciphertextB64, nonceB64 string, err error) {
	if plain == "" {
		return "", "", nil
	}

	key, err := loadSecretKey()
	if err != nil {
		return "", "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", fmt.Errorf("failed to generate random nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), base64.StdEncoding.EncodeToString(nonce), nil
}

func DecryptText(ciphertextB64, nonceB64 string) (string, error) {
	if strings.TrimSpace(ciphertextB64) == "" {
		return "", nil
	}

	key, err := loadSecretKey()
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plain), nil
}

func MaskToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
