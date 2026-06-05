package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"xiaozhi/manager/backend/models"

	"gorm.io/gorm"
)

func (vcc *VoiceCloneController) startVoiceCloneWorkers() {
	if vcc == nil || vcc.DB == nil {
		return
	}
	for i := 0; i < voiceCloneTaskWorkerCount; i++ {
		go vcc.voiceCloneTaskWorkerLoop(i + 1)
	}
	go vcc.reloadPendingVoiceCloneTasks()
}

func (vcc *VoiceCloneController) reloadPendingVoiceCloneTasks() {
	var pendingTasks []models.VoiceCloneTask
	if err := vcc.DB.Where("status IN ?", []string{voiceCloneTaskStatusQueued, voiceCloneTaskStatusProcessing}).
		Order("created_at ASC").
		Find(&pendingTasks).Error; err != nil {
		log.Printf("[voice_clone][task] reload pending tasks failed: %v", err)
		return
	}
	for _, task := range pendingTasks {
		vcc.enqueueVoiceCloneTask(task.ID)
	}
	if len(pendingTasks) > 0 {
		log.Printf("[voice_clone][task] reloaded pending tasks: %d", len(pendingTasks))
	}
}

func (vcc *VoiceCloneController) enqueueVoiceCloneTask(taskPrimaryID uint) {
	if taskPrimaryID == 0 || vcc == nil || vcc.taskQueue == nil {
		return
	}
	select {
	case vcc.taskQueue <- taskPrimaryID:
	default:
		log.Printf("[voice_clone][task] queue is full, fallback to async enqueue: task_primary_id=%d", taskPrimaryID)
		go func(id uint) {
			vcc.taskQueue <- id
		}(taskPrimaryID)
	}
}

func (vcc *VoiceCloneController) voiceCloneTaskWorkerLoop(workerID int) {
	for taskPrimaryID := range vcc.taskQueue {
		log.Printf("[voice_clone][task] worker=%d picked task_primary_id=%d", workerID, taskPrimaryID)
		vcc.processVoiceCloneTask(taskPrimaryID)
	}
}

func (vcc *VoiceCloneController) processVoiceCloneTask(taskPrimaryID uint) {
	task, claimed, err := vcc.claimVoiceCloneTask(taskPrimaryID)
	if err != nil {
		log.Printf("[voice_clone][task] claim task failed: task_primary_id=%d err=%v", taskPrimaryID, err)
		return
	}
	if !claimed || task == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), voiceCloneTaskProcessLimit)
	defer cancel()

	var clone models.VoiceClone
	if err = vcc.DB.Where("id = ? AND user_id = ?", task.VoiceCloneID, task.UserID).First(&clone).Error; err != nil {
		vcc.finishVoiceCloneTaskFailed(task, nil, fmt.Errorf("task-linked clone record not found: %w", err))
		return
	}

	var audio models.VoiceCloneAudio
	if err = vcc.DB.Where("voice_clone_id = ? AND user_id = ?", clone.ID, task.UserID).Order("created_at DESC").First(&audio).Error; err != nil {
		vcc.finishVoiceCloneTaskFailed(task, &clone, fmt.Errorf("task-linked audio record not found: %w", err))
		return
	}
	if !vcc.AudioStorage.FileExists(audio.FilePath) {
		vcc.finishVoiceCloneTaskFailed(task, &clone, fmt.Errorf("task audio file not found: %s", audio.FilePath))
		return
	}

	var ttsCfg models.Config
	if err = vcc.DB.Where("type = ? AND config_id = ?", "tts", clone.TTSConfigID).First(&ttsCfg).Error; err != nil {
		vcc.finishVoiceCloneTaskFailed(task, &clone, fmt.Errorf("task-linked TTS config not found: %w", err))
		return
	}
	provider := normalizeCloneProvider(strings.TrimSpace(ttsCfg.Provider))
	var result *minimaxVoiceCloneResult
	switch provider {
	case "doubao":
		result, err = vcc.cloneWithDoubao(ctx, ttsCfg, clone.TTSConfigID, audio.FilePath, audio.FileName, audio.Transcript)
	case "minimax":
		result, err = vcc.cloneWithMinimax(ctx, ttsCfg, clone.TTSConfigID, audio.FilePath, audio.FileName, audio.Transcript)
	case "cosyvoice":
		result, err = vcc.cloneWithCosyVoice(ctx, audio.FilePath, audio.FileName, audio.Transcript)
	case "aliyun_qwen":
		result, err = vcc.cloneWithAliyunQwen(ctx, ttsCfg, clone.TTSConfigID, audio.FilePath, audio.FileName, audio.Transcript, audio.TranscriptLang)
	case "indextts_vllm":
		result, err = vcc.cloneWithIndexTTSVLLM(ctx, ttsCfg, audio.FilePath, audio.FileName)
	default:
		vcc.finishVoiceCloneTaskFailed(task, &clone, fmt.Errorf("provider not supported for this task: %s", provider))
		return
	}
	if err != nil {
		vcc.finishVoiceCloneTaskFailed(task, &clone, err)
		return
	}
	if err = vcc.finishVoiceCloneTaskSuccess(task, &clone, &audio, result); err != nil {
		log.Printf("[voice_clone][task] finish success failed: task_primary_id=%d err=%v", taskPrimaryID, err)
		return
	}
	log.Printf("[voice_clone][task] task completed: task_primary_id=%d task_id=%s voice_clone_id=%d", taskPrimaryID, task.TaskID, task.VoiceCloneID)
}

func (vcc *VoiceCloneController) claimVoiceCloneTask(taskPrimaryID uint) (*models.VoiceCloneTask, bool, error) {
	var task models.VoiceCloneTask
	claimed := false
	err := vcc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&task, taskPrimaryID).Error; err != nil {
			return err
		}
		if task.Status == voiceCloneTaskStatusSucceeded || task.Status == voiceCloneTaskStatusFailed {
			return nil
		}
		if task.Status != voiceCloneTaskStatusQueued && task.Status != voiceCloneTaskStatusProcessing {
			return nil
		}

		now := time.Now()
		updateResult := tx.Model(&models.VoiceCloneTask{}).
			Where("id = ? AND status IN ?", task.ID, []string{voiceCloneTaskStatusQueued, voiceCloneTaskStatusProcessing}).
			Updates(map[string]any{
				"status":      voiceCloneTaskStatusProcessing,
				"attempts":    gorm.Expr("attempts + ?", 1),
				"last_error":  "",
				"started_at":  now,
				"finished_at": nil,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return nil
		}
		if err := tx.Model(&models.VoiceClone{}).Where("id = ? AND user_id = ?", task.VoiceCloneID, task.UserID).Updates(map[string]any{
			"status": voiceCloneStatusProcessing,
		}).Error; err != nil {
			return err
		}
		if err := tx.First(&task, task.ID).Error; err != nil {
			return err
		}
		claimed = true
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !claimed {
		return &task, false, nil
	}
	return &task, true, nil
}

func (vcc *VoiceCloneController) finishVoiceCloneTaskSuccess(task *models.VoiceCloneTask, clone *models.VoiceClone, audio *models.VoiceCloneAudio, result *minimaxVoiceCloneResult) error {
	if task == nil || clone == nil || audio == nil || result == nil {
		return errors.New("task/clone/audio/result parameters must not be nil")
	}
	now := time.Now()

	cloneMetaJSON := mergeJSONMeta(clone.MetaJSON, map[string]any{
		"source_type": audio.SourceType,
		"request_id":  result.RequestID,
		"http_code":   result.ResponseCode,
		"response":    result.RawResponse,
		"task_id":     task.TaskID,
		"task_status": voiceCloneTaskStatusSucceeded,
		"finished_at": now,
	})
	taskMeta := map[string]any{
		"request_id":  result.RequestID,
		"http_code":   result.ResponseCode,
		"response":    result.RawResponse,
		"voice_id":    result.VoiceID,
		"finished_at": now,
	}
	if strings.TrimSpace(result.TargetModel) != "" {
		targetModel := strings.TrimSpace(result.TargetModel)
		cloneMetaJSON = mergeJSONMeta(cloneMetaJSON, map[string]any{
			"target_model": targetModel,
		})
		taskMeta["target_model"] = targetModel
	}
	taskMetaJSON, _ := json.Marshal(taskMeta)

	return vcc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.VoiceClone{}).Where("id = ? AND user_id = ? AND status != ?", clone.ID, clone.UserID, "deleted").Updates(map[string]any{
			"provider_voice_id": result.VoiceID,
			"status":            voiceCloneStatusActive,
			"meta_json":         cloneMetaJSON,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.VoiceCloneTask{}).Where("id = ?", task.ID).Updates(map[string]any{
			"status":      voiceCloneTaskStatusSucceeded,
			"last_error":  "",
			"finished_at": now,
			"meta_json":   string(taskMetaJSON),
		}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (vcc *VoiceCloneController) finishVoiceCloneTaskFailed(task *models.VoiceCloneTask, clone *models.VoiceClone, failure error) {
	if task == nil {
		return
	}
	now := time.Now()
	lastError := "unknown error"
	if failure != nil {
		lastError = truncateForLog(strings.TrimSpace(failure.Error()), 2000)
	}

	cloneMetaJSON := ""
	if clone != nil {
		cloneMetaJSON = mergeJSONMeta(clone.MetaJSON, map[string]any{
			"task_id":     task.TaskID,
			"task_status": voiceCloneTaskStatusFailed,
			"last_error":  lastError,
			"finished_at": now,
		})
	}
	taskMetaJSON, _ := json.Marshal(map[string]any{
		"last_error":  lastError,
		"finished_at": now,
	})

	err := vcc.DB.Transaction(func(tx *gorm.DB) error {
		cloneUpdates := map[string]any{
			"status": voiceCloneStatusFailed,
		}
		if cloneMetaJSON != "" {
			cloneUpdates["meta_json"] = cloneMetaJSON
		}
		if err := tx.Model(&models.VoiceClone{}).
			Where("id = ? AND user_id = ? AND status != ?", task.VoiceCloneID, task.UserID, "deleted").
			Updates(cloneUpdates).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.VoiceCloneTask{}).Where("id = ?", task.ID).Updates(map[string]any{
			"status":      voiceCloneTaskStatusFailed,
			"last_error":  lastError,
			"finished_at": now,
			"meta_json":   string(taskMetaJSON),
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Printf("[voice_clone][task] mark failed status failed: task_primary_id=%d err=%v origin=%s", task.ID, err, lastError)
		return
	}
	log.Printf("[voice_clone][task] task failed: task_primary_id=%d task_id=%s reason=%s", task.ID, task.TaskID, lastError)
}

func mergeJSONMeta(raw string, updates map[string]any) string {
	meta := make(map[string]any)
	if raw = strings.TrimSpace(raw); raw != "" {
		_ = json.Unmarshal([]byte(raw), &meta)
	}
	for key, value := range updates {
		meta[key] = value
	}
	merged, err := json.Marshal(meta)
	if err != nil {
		return raw
	}
	return string(merged)
}
