package controllers

import (
	"errors"

	"xiaozhi/manager/backend/models"

	"gorm.io/gorm"
)

// updateDeviceColumns writes only explicitly declared columns to avoid overwriting
// historical zero-value fields (e.g. created_at) with a full-row update.
func updateDeviceColumns(db *gorm.DB, deviceID uint, updates map[string]interface{}) error {
	if deviceID == 0 {
		return errors.New("device id is required")
	}
	if len(updates) == 0 {
		return nil
	}

	return db.Model(&models.Device{}).Where("id = ?", deviceID).Updates(updates).Error
}

func countDevicesByAgentID(db *gorm.DB, agentID uint) (int64, error) {
	if agentID == 0 {
		return 0, nil
	}

	var count int64
	err := db.Model(&models.Device{}).Where("agent_id = ?", agentID).Count(&count).Error
	return count, err
}
