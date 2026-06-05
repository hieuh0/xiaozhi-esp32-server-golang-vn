package controllers

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChatHistoryController struct {
	DB            *gorm.DB
	AudioBasePath string // base path for audio storage
	MaxFileSize   int64  // maximum file size (10 MB)
}

// SaveMessageRequest is the request body for saving a message.
type SaveMessageRequest struct {
	MessageID     string                 `json:"message_id" binding:"required"`
	DeviceID      string                 `json:"device_id" binding:"required"`
	AgentID       string                 `json:"agent_id" binding:"required"`
	SessionID     string                 `json:"session_id,omitempty"`
	Role          string                 `json:"role" binding:"required,oneof=user assistant system tool"`
	Content       string                 `json:"content" binding:"required"`
	ToolCallID    string                 `json:"tool_call_id,omitempty"`    // tool call ID (used by tool role)
	ToolCallsJSON *string                `json:"tool_calls_json,omitempty"` // tool call list JSON (used by assistant role), nil means NULL
	AudioData     string                 `json:"audio_data,omitempty"`      // base64 encoded
	AudioFormat   string                 `json:"audio_format,omitempty"`    // audio format from client; backend always stores wav
	AudioDuration int                    `json:"audio_duration,omitempty"`
	AudioSize     int                    `json:"audio_size,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SaveMessage saves a chat message.
func (c *ChatHistoryController) SaveMessage(ctx *gin.Context) {
	var req SaveMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify device exists (query by device_name field).
	var device models.Device
	if err := c.DB.Where("device_name = ?", req.DeviceID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query device: " + err.Error()})
		return
	}

	// If AgentID is not provided, use the device's associated AgentID.
	agentID := req.AgentID
	if agentID == "" && device.AgentID > 0 {
		agentID = fmt.Sprintf("%d", device.AgentID)
	}

	// If AgentID is still empty, skip saving.
	if agentID == "" {
		ctx.JSON(http.StatusOK, gin.H{"message": "skipped: no associated AgentID"})
		return
	}

	message := &models.ChatMessage{
		MessageID:     req.MessageID,
		DeviceID:      req.DeviceID,
		AgentID:       agentID,
		UserID:        device.UserID,
		SessionID:     req.SessionID,
		Role:          req.Role,
		Content:       req.Content,
		ToolCallID:    req.ToolCallID,
		ToolCallsJSON: req.ToolCallsJSON,
		Metadata:      req.Metadata,
	}

	// Check if the message already exists (avoid duplicate creation).
	var existingMessage models.ChatMessage
	err := c.DB.Where("message_id = ?", req.MessageID).First(&existingMessage).Error
	if err == nil {
		// Message exists — update audio data if provided.
		if req.AudioData != "" {
			audioPath, err := c.saveAudioFile(req.MessageID, req.AudioData)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save audio file: " + err.Error()})
				return
			}

			// Delete the previous audio file if present.
			if existingMessage.AudioPath != "" {
				c.deleteAudioFile(existingMessage.AudioPath)
			}

			// Update message.
			updates := map[string]interface{}{
				"audio_path":   audioPath,
				"audio_format": "wav",
			}
			if req.AudioSize > 0 {
				updates["audio_size"] = req.AudioSize
			}
			if req.AudioDuration > 0 {
				updates["audio_duration"] = req.AudioDuration
			}

			// Merge metadata.
			if existingMessage.Metadata == nil {
				existingMessage.Metadata = make(map[string]interface{})
			}
			if req.Metadata != nil {
				for k, v := range req.Metadata {
					existingMessage.Metadata[k] = v
				}
			}
			// Manually serialize metadata to MetadataJSON (Updates does not trigger BeforeSave hook).
			metadataJSONBytes, err := json.Marshal(existingMessage.Metadata)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize metadata: " + err.Error()})
				return
			}
			updates["metadata"] = string(metadataJSONBytes)

			if err := c.DB.Model(&existingMessage).Updates(updates).Error; err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update message"})
				return
			}
			ctx.JSON(http.StatusOK, existingMessage)
			return
		}
		// Message exists and no audio data — return as-is.
		ctx.JSON(http.StatusOK, existingMessage)
		return
	} else if err != gorm.ErrRecordNotFound {
		// Query error (not "record not found").
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query message: " + err.Error()})
		return
	}

	// Message does not exist — create it.
	// Handle audio data: save to filesystem (fixed wav format, two-level hash directory sharding).
	if req.AudioData != "" {
		audioPath, err := c.saveAudioFile(req.MessageID, req.AudioData)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save audio file: " + err.Error()})
			return
		}
		message.AudioPath = audioPath
		message.AudioFormat = "wav" // always wav
		if req.AudioSize > 0 {
			message.AudioSize = &req.AudioSize
		}
		if req.AudioDuration > 0 {
			message.AudioDuration = &req.AudioDuration
		}
	}

	if err := c.DB.Create(message).Error; err != nil {
		// If database save fails, delete the saved audio file.
		if message.AudioPath != "" {
			c.deleteAudioFile(message.AudioPath)
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save message: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, message)
}

// GetMessages returns a paginated list of messages (summarised by agentId).
func (c *ChatHistoryController) GetMessages(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	agentID := ctx.Query("agent_id")
	deviceID := ctx.Query("device_id")
	sessionID := ctx.Query("session_id")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "50"))
	role := ctx.Query("role") // user/assistant

	// Build query.
	query := c.DB.Model(&models.ChatMessage{}).
		Where("user_id = ? AND is_deleted = ?", userID, false)

	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	var total int64
	query.Count(&total)

	var messages []models.ChatMessage
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&messages).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      messages,
	})
}

// DeleteMessage soft-deletes a message and immediately removes its audio file.
func (c *ChatHistoryController) DeleteMessage(ctx *gin.Context) {
	id := ctx.Param("id")

	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Fetch message info.
	var message models.ChatMessage
	if err := c.DB.Where("id = ? AND user_id = ?", id, userID).First(&message).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	// Delete audio file first if present.
	if message.AudioPath != "" {
		if err := c.deleteAudioFile(message.AudioPath); err != nil {
			// Log but do not block the deletion.
			log.Printf("failed to delete audio file: %v", err)
		}
	}

	// Soft-delete the message.
	if err := c.DB.Model(&models.ChatMessage{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// GetMessagesByAgent returns messages for a given AgentID with optional filters.
func (c *ChatHistoryController) GetMessagesByAgent(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	agentID := ctx.Param("agent_id")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "50"))
	role := ctx.Query("role")            // user/assistant
	deviceID := ctx.Query("device_id")   // filter by device ID
	startDate := ctx.Query("start_date") // start date YYYY-MM-DD
	endDate := ctx.Query("end_date")     // end date YYYY-MM-DD

	// Build query.
	query := c.DB.Model(&models.ChatMessage{}).
		Where("user_id = ? AND agent_id = ? AND is_deleted = ?", userID, agentID, false)

	// Role filter.
	if role != "" {
		query = query.Where("role = ?", role)
	}

	// Device filter.
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}

	// Date range filter.
	if startDate != "" {
		if startTime, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if endDate != "" {
		if endTime, err := time.Parse("2006-01-02", endDate); err == nil {
			// End date is inclusive of the full day.
			endTime = endTime.Add(24 * time.Hour)
			query = query.Where("created_at < ?", endTime)
		}
	}

	// Total count.
	var total int64
	query.Count(&total)

	// Paginated query (newest first; frontend reverses the array to show newest at bottom).
	var messages []models.ChatMessage
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&messages).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      messages,
	})
}

// ExportMessages exports chat history in JSON format.
func (c *ChatHistoryController) ExportMessages(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	agentID := ctx.Query("agent_id")
	deviceID := ctx.Query("device_id")
	startDate := ctx.Query("start_date")
	endDate := ctx.Query("end_date")

	// Build query.
	query := c.DB.Model(&models.ChatMessage{}).
		Where("user_id = ? AND is_deleted = ?", userID, false)

	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if startDate != "" {
		if startTime, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if endDate != "" {
		if endTime, err := time.Parse("2006-01-02", endDate); err == nil {
			// End date is inclusive of the full day.
			endTime = endTime.Add(24 * time.Hour)
			query = query.Where("created_at < ?", endTime)
		}
	}

	var messages []models.ChatMessage
	if err := query.Order("created_at ASC").Find(&messages).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "export failed"})
		return
	}

	// Set response headers to trigger download.
	ctx.Header("Content-Type", "application/json")
	ctx.Header("Content-Disposition", "attachment; filename=chat_history_"+time.Now().Format("20060102_150405")+".json")
	ctx.JSON(http.StatusOK, gin.H{
		"export_time": time.Now().Format("2006-01-02 15:04:05"),
		"total":       len(messages),
		"messages":    messages,
	})
}

// saveAudioFile saves a base64-encoded audio file to the filesystem using two-level hash directory sharding.
func (c *ChatHistoryController) saveAudioFile(messageID, audioDataBase64 string) (string, error) {
	// Decode base64 audio data.
	audioData, err := base64.StdEncoding.DecodeString(audioDataBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode audio data: %v", err)
	}

	// Check file size.
	if int64(len(audioData)) > c.MaxFileSize {
		return "", fmt.Errorf("audio file size exceeds limit: %d > %d", len(audioData), c.MaxFileSize)
	}

	// Compute MD5 of message_id for the filename (without extension).
	fileNameHash := fmt.Sprintf("%x", md5.Sum([]byte(messageID)))

	// Two-level hash for directory sharding.
	hash1 := fileNameHash[0:2] // first 2 characters
	hash2 := fileNameHash[2:4] // characters 3-4

	// Build file path: {base_path}/{hash1}/{hash2}/{md5(message_id)}.wav
	relativePath := fmt.Sprintf("%s/%s/%s.wav", hash1, hash2, fileNameHash)
	fullPath := filepath.Join(c.AudioBasePath, relativePath)

	// Create directories.
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}

	// Write file.
	if err := os.WriteFile(fullPath, audioData, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	// Return relative path for database storage.
	return relativePath, nil
}

// deleteAudioFile deletes an audio file by its relative path.
func (c *ChatHistoryController) deleteAudioFile(relativePath string) error {
	fullPath := filepath.Join(c.AudioBasePath, relativePath)
	return os.Remove(fullPath)
}

// GetAudioFile serves an audio file proxied through Go.
func (c *ChatHistoryController) GetAudioFile(ctx *gin.Context) {
	id := ctx.Param("id")

	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Fetch message info.
	var message models.ChatMessage
	if err := c.DB.Where("id = ? AND user_id = ? AND is_deleted = ?", id, userID, false).First(&message).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	if message.AudioPath == "" {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "audio file not found"})
		return
	}

	// Read file.
	fullPath := filepath.Join(c.AudioBasePath, message.AudioPath)
	audioData, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "audio file not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read audio file"})
		}
		return
	}

	// Set response headers (wav format).
	ctx.Header("Content-Type", "audio/wav")
	ctx.Header("Content-Length", strconv.Itoa(len(audioData)))
	ctx.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", filepath.Base(message.AudioPath)))

	// Stream audio data.
	ctx.Data(http.StatusOK, "audio/wav", audioData)
}

// GetMessagesForInit returns messages for initialisation (internal service endpoint, no auth required).
func (c *ChatHistoryController) GetMessagesForInit(ctx *gin.Context) {
	deviceID := ctx.Query("device_id")
	agentID := ctx.Query("agent_id")
	sessionID := ctx.Query("session_id")
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))

	if deviceID == "" || agentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "device_id and agent_id cannot be empty"})
		return
	}

	// Build query (no user_id filter; this is an internal service endpoint).
	query := c.DB.Model(&models.ChatMessage{}).
		Where("device_id = ? AND agent_id = ? AND is_deleted = ?", deviceID, agentID, false)

	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}

	var messages []models.ChatMessage
	// Fetch the latest N records then reverse to chronological order (old → new) for LLM consumption.
	if err := query.Order("created_at DESC").
		Order("id DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	// Reverse to return old → new order.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	// Build response (text only, no audio).
	messageItems := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		item := map[string]interface{}{
			"message_id": msg.MessageID,
			"role":       msg.Role,
			"content":    msg.Content,
			"created_at": msg.CreatedAt.Format(time.RFC3339),
		}
		// Include tool_call_id if present.
		if msg.ToolCallID != "" {
			item["tool_call_id"] = msg.ToolCallID
		}
		// Include tool_calls if present.
		if msg.ToolCallsJSON != nil && *msg.ToolCallsJSON != "" {
			var toolCalls []interface{}
			if err := json.Unmarshal([]byte(*msg.ToolCallsJSON), &toolCalls); err == nil {
				item["tool_calls"] = toolCalls
			}
		}
		messageItems = append(messageItems, item)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"messages": messageItems,
	})
}

// UpdateMessageAudioRequest is the request body for updating message audio.
type UpdateMessageAudioRequest struct {
	AudioData   string                 `json:"audio_data" binding:"required"`
	AudioFormat string                 `json:"audio_format"`
	AudioSize   int                    `json:"audio_size"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateMessageAudio updates the audio of an existing message.
func (c *ChatHistoryController) UpdateMessageAudio(ctx *gin.Context) {
	messageID := ctx.Param("message_id")
	if messageID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "message_id cannot be empty"})
		return
	}

	var req UpdateMessageAudioRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find message.
	var message models.ChatMessage
	if err := c.DB.Where("message_id = ?", messageID).First(&message).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Message not found — skip update (may have been skipped during SaveMessage due to missing AgentID).
			ctx.JSON(http.StatusOK, gin.H{"message": "skipped: message not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query message"})
		return
	}

	// If the message has no associated AgentID, skip update.
	if message.AgentID == "" {
		ctx.JSON(http.StatusOK, gin.H{"message": "skipped: no associated AgentID"})
		return
	}

	// Save audio file.
	if req.AudioData != "" {
		audioPath, err := c.saveAudioFile(messageID, req.AudioData)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save audio file: " + err.Error()})
			return
		}

		// Delete previous audio file if present.
		if message.AudioPath != "" {
			c.deleteAudioFile(message.AudioPath)
		}

		// Update message.
		updates := map[string]interface{}{
			"audio_path":   audioPath,
			"audio_format": "wav",
		}
		if req.AudioSize > 0 {
			updates["audio_size"] = req.AudioSize
		}

		// Merge metadata.
		if message.Metadata == nil {
			message.Metadata = make(map[string]interface{})
		}
		for k, v := range req.Metadata {
			message.Metadata[k] = v
		}
		// Manually serialize metadata to MetadataJSON (Updates does not trigger BeforeSave hook).
		metadataJSONBytes, err := json.Marshal(message.Metadata)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize metadata: " + err.Error()})
			return
		}
		updates["metadata"] = string(metadataJSONBytes)

		if err := c.DB.Model(&message).Updates(updates).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update message"})
			return
		}
	}

	ctx.JSON(http.StatusOK, message)
}
