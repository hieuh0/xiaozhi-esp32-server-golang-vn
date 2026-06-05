package storage

import (
	"context"
	"gorm.io/gorm"
	"xiaozhi/manager/backend/models"
)

// GormDeviceStorage generic GORM device storage implementation
type GormDeviceStorage struct {
	db *gorm.DB
}

// NewGormDeviceStorage creates a GORM device storage instance
func NewGormDeviceStorage(db *gorm.DB) *GormDeviceStorage {
	return &GormDeviceStorage{
		db: db,
	}
}

// CreateDevice creates a device
func (s *GormDeviceStorage) CreateDevice(ctx context.Context, device *models.Device) error {
	return s.db.WithContext(ctx).Create(device).Error
}

// GetDeviceByID retrieves a device by ID
func (s *GormDeviceStorage) GetDeviceByID(ctx context.Context, id uint) (*models.Device, error) {
	var device models.Device
	err := s.db.WithContext(ctx).First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDeviceByCode retrieves a device by device code
func (s *GormDeviceStorage) GetDeviceByCode(ctx context.Context, deviceCode string) (*models.Device, error) {
	var device models.Device
	err := s.db.WithContext(ctx).Where("device_code = ?", deviceCode).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDevicesByUserID retrieves a paginated list of devices by user ID
func (s *GormDeviceStorage) GetDevicesByUserID(ctx context.Context, userID uint, offset, limit int) ([]*models.Device, int64, error) {
	var devices []*models.Device
	var total int64

	// get total count
	err := s.db.WithContext(ctx).Model(&models.Device{}).Where("user_id = ?", userID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// get paginated data
	err = s.db.WithContext(ctx).Where("user_id = ?", userID).Offset(offset).Limit(limit).Find(&devices).Error
	return devices, total, err
}

// UpdateDevice updates a device
func (s *GormDeviceStorage) UpdateDevice(ctx context.Context, id uint, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&models.Device{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteDevice deletes a device
func (s *GormDeviceStorage) DeleteDevice(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Device{}, id).Error
}
