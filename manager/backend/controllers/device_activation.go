package controllers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeviceActivationController struct {
	DB *gorm.DB
}

// generateCode generates a 6-digit random numeric code
func generateCode() string {
	randomBytes := make([]byte, 3)
	rand.Read(randomBytes)
	code := 0
	for i, b := range randomBytes {
		code += int(b) << (8 * i)
	}
	return fmt.Sprintf("%06d", code%1000000)
}

// generateChallenge generates a UUID-format challenge code
func generateChallenge() string {
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)

	// set version (4) and variant bits
	randomBytes[6] = (randomBytes[6] & 0x0f) | 0x40
	randomBytes[8] = (randomBytes[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		randomBytes[0:4],
		randomBytes[4:6],
		randomBytes[6:8],
		randomBytes[8:10],
		randomBytes[10:16])
}

// CheckDeviceActivation checks whether a device is activated
// GET /api/internal/device/check-activation?device_id=xxx&client_id=xxx
func (dac *DeviceActivationController) CheckDeviceActivation(c *gin.Context) {
	deviceId := c.Query("device_id")

	if deviceId == "" {
		c.JSON(http.StatusOK, gin.H{
			"activated": false,
			"error":     "device_id parameter is required",
		})
		return
	}

	var device models.Device
	if err := dac.DB.Where("device_name = ?", deviceId).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"activated": false,
				"message":   "device not found",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"activated": false,
			"error":     "query failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activated": device.Activated,
		"message": func() string {
			if device.Activated {
				return "device is activated"
			}
			return "device is not activated"
		}(),
	})
}

// GetActivationInfo returns activation info for a device
// GET /api/internal/device/activation-info?device_id=xxx&client_id=xxx
func (dac *DeviceActivationController) GetActivationInfo(c *gin.Context) {
	deviceId := c.Query("device_id")

	if deviceId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id and client_id parameters are required"})
		return
	}

	var device models.Device
	var isNewDevice bool

	if err := dac.DB.Where("device_name = ?", deviceId).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			device = models.Device{
				DeviceName: deviceId,
				UserID:     0,
				DeviceCode: generateCode(),
				Challenge:  generateChallenge(),
				Activated:  false,
			}

			if err := dac.DB.Create(&device).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create device record"})
				return
			}
			isNewDevice = true
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
	}

	if device.Activated {
		c.JSON(http.StatusOK, gin.H{
			"activated": true,
			"message":   "device is activated",
		})
		return
	}

	needUpdate := false

	if device.DeviceCode == "" {
		device.DeviceCode = generateCode()
		needUpdate = true
	}

	if device.Challenge == "" {
		device.Challenge = generateChallenge()
		needUpdate = true
	}

	if !isNewDevice && device.UserID != 0 {
		device.UserID = 0
		needUpdate = true
	}

	if needUpdate {
		updates := map[string]interface{}{}
		if device.DeviceCode != "" {
			updates["device_code"] = device.DeviceCode
		}
		if device.Challenge != "" {
			updates["challenge"] = device.Challenge
		}
		if !isNewDevice && device.UserID == 0 {
			updates["user_id"] = device.UserID
		}
		if err := updateDeviceColumns(dac.DB, device.ID, updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update device info"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"activated": false,
		"code":      device.DeviceCode,
		"challenge": device.Challenge,
		"message":   "activate device in the admin panel, code: " + device.DeviceCode,
	})
}

// verifyHMAC validates HMAC-SHA256
func verifyHMAC(challenge, secretKey, providedHmac string) bool {
	if secretKey == "" {
		return true // pass if pre_secret_key is empty
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(challenge))
	expectedHmac := hex.EncodeToString(mac.Sum(nil))

	return expectedHmac == providedHmac
}

// ActivateDevice handles device activation
// POST /api/internal/device/activate
func (dac *DeviceActivationController) ActivateDevice(c *gin.Context) {
	var req struct {
		DeviceId     string `json:"device_id" binding:"required"`
		ClientId     string `json:"client_id" binding:"required"`
		Challenge    string `json:"challenge" binding:"required"`
		Algorithm    string `json:"algorithm" binding:"required"`
		SerialNumber string `json:"serial_number" binding:"required"`
		Hmac         string `json:"hmac" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parameters: " + err.Error()})
		return
	}

	var device models.Device
	if err := dac.DB.Where("device_name = ?", req.DeviceId).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "device not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "query failed",
		})
		return
	}

	if device.Activated {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "device is activated",
		})
		return
	}

	if device.UserID == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "device not bound to any user",
		})
		return
	}

	if device.Challenge != req.Challenge {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "invalid challenge",
		})
		return
	}

	if !verifyHMAC(req.Challenge, device.PreSecretKey, req.Hmac) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "HMAC verification failed",
		})
		return
	}

	device.Activated = true
	if err := updateDeviceColumns(dac.DB, device.ID, map[string]interface{}{
		"activated": device.Activated,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to activate device",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "device activated successfully",
		"data": gin.H{
			"device_id": device.DeviceName,
			"activated": device.Activated,
		},
	})
}
