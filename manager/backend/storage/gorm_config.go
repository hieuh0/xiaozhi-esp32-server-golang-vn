package storage

import (
	"context"
	"gorm.io/gorm"
	"xiaozhi/manager/backend/models"
)

// GormConfigStorage generic GORM config storage implementation
type GormConfigStorage struct {
	db *gorm.DB
}

// NewGormConfigStorage creates a GORM config storage instance
func NewGormConfigStorage(db *gorm.DB) *GormConfigStorage {
	return &GormConfigStorage{
		db: db,
	}
}

// general config operation methods
func (s *GormConfigStorage) CreateConfig(ctx context.Context, config *models.Config) error {
	return s.db.WithContext(ctx).Create(config).Error
}

func (s *GormConfigStorage) GetConfigByID(ctx context.Context, id uint) (*models.Config, error) {
	var config models.Config
	err := s.db.WithContext(ctx).First(&config, id).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *GormConfigStorage) GetConfigs(ctx context.Context, configType string) ([]*models.Config, error) {
	var configs []*models.Config
	query := s.db.WithContext(ctx)
	if configType != "" {
		query = query.Where("type = ?", configType)
	}
	err := query.Find(&configs).Error
	return configs, err
}

func (s *GormConfigStorage) GetConfigByTypeAndName(ctx context.Context, configType, name string) (*models.Config, error) {
	var config models.Config
	err := s.db.WithContext(ctx).Where("type = ? AND name = ?", configType, name).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *GormConfigStorage) GetDefaultConfig(ctx context.Context, configType string) (*models.Config, error) {
	var config models.Config
	err := s.db.WithContext(ctx).Where("type = ? AND is_default = ?", configType, true).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *GormConfigStorage) UpdateConfig(ctx context.Context, id uint, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&models.Config{}).Where("id = ?", id).Updates(updates).Error
}

func (s *GormConfigStorage) DeleteConfig(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Config{}, id).Error
}

func (s *GormConfigStorage) SetDefaultConfig(ctx context.Context, configType string, id uint) error {
	// use transaction
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// clear default flag for all configs of this type
		if err := tx.Model(&models.Config{}).Where("type = ?", configType).Update("is_default", false).Error; err != nil {
			return err
		}
		// set the specified config as default
		return tx.Model(&models.Config{}).Where("id = ?", id).Update("is_default", true).Error
	})
}

// global role config methods
func (s *GormConfigStorage) CreateGlobalRole(ctx context.Context, role *models.GlobalRole) error {
	return s.db.WithContext(ctx).Create(role).Error
}

func (s *GormConfigStorage) GetGlobalRoleByID(ctx context.Context, id uint) (*models.GlobalRole, error) {
	var role models.GlobalRole
	err := s.db.WithContext(ctx).First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *GormConfigStorage) GetGlobalRoles(ctx context.Context) ([]*models.GlobalRole, error) {
	var roles []*models.GlobalRole
	err := s.db.WithContext(ctx).Find(&roles).Error
	return roles, err
}

func (s *GormConfigStorage) UpdateGlobalRole(ctx context.Context, id uint, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&models.GlobalRole{}).Where("id = ?", id).Updates(updates).Error
}

func (s *GormConfigStorage) DeleteGlobalRole(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.GlobalRole{}, id).Error
}
