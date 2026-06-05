package storage

import (
	"context"
	"xiaozhi/manager/backend/models"
)

// ConfigAdapter config adapter
type ConfigAdapter struct {
	GormConfigStorage ConfigStorage
}

// NewConfigAdapter creates a config adapter
func NewConfigAdapter(configStorage ConfigStorage) *ConfigAdapter {
	return &ConfigAdapter{
		GormConfigStorage: configStorage,
	}
}

// general config adapter methods
func (a *ConfigAdapter) CreateConfig(ctx context.Context, config *models.Config) error {
	return a.GormConfigStorage.CreateConfig(ctx, config)
}

func (a *ConfigAdapter) GetConfigs(ctx context.Context, configType string) ([]*models.Config, error) {
	return a.GormConfigStorage.GetConfigs(ctx, configType)
}

func (a *ConfigAdapter) GetConfigByID(ctx context.Context, id uint) (*models.Config, error) {
	return a.GormConfigStorage.GetConfigByID(ctx, id)
}

func (a *ConfigAdapter) GetConfigByTypeAndName(ctx context.Context, configType, name string) (*models.Config, error) {
	return a.GormConfigStorage.GetConfigByTypeAndName(ctx, configType, name)
}

func (a *ConfigAdapter) GetDefaultConfig(ctx context.Context, configType string) (*models.Config, error) {
	return a.GormConfigStorage.GetDefaultConfig(ctx, configType)
}

func (a *ConfigAdapter) UpdateConfig(ctx context.Context, id uint, updates map[string]interface{}) error {
	return a.GormConfigStorage.UpdateConfig(ctx, id, updates)
}

func (a *ConfigAdapter) DeleteConfig(ctx context.Context, id uint) error {
	return a.GormConfigStorage.DeleteConfig(ctx, id)
}

func (a *ConfigAdapter) SetDefaultConfig(ctx context.Context, configType string, id uint) error {
	return a.GormConfigStorage.SetDefaultConfig(ctx, configType, id)
}

// GlobalRole config adapter methods
func (a *ConfigAdapter) CreateGlobalRole(ctx context.Context, role *models.GlobalRole) error {
	return a.GormConfigStorage.CreateGlobalRole(ctx, role)
}

func (a *ConfigAdapter) GetGlobalRoles(ctx context.Context) ([]*models.GlobalRole, error) {
	return a.GormConfigStorage.GetGlobalRoles(ctx)
}

func (a *ConfigAdapter) GetGlobalRoleByID(ctx context.Context, id uint) (*models.GlobalRole, error) {
	return a.GormConfigStorage.GetGlobalRoleByID(ctx, id)
}

func (a *ConfigAdapter) UpdateGlobalRole(ctx context.Context, id uint, updates map[string]interface{}) error {
	return a.GormConfigStorage.UpdateGlobalRole(ctx, id, updates)
}

func (a *ConfigAdapter) DeleteGlobalRole(ctx context.Context, id uint) error {
	return a.GormConfigStorage.DeleteGlobalRole(ctx, id)
}
