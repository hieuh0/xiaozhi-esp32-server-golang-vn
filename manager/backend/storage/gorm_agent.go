package storage

import (
	"context"
	"gorm.io/gorm"
	"xiaozhi/manager/backend/models"
)

// GormAgentStorage generic GORM agent storage implementation
type GormAgentStorage struct {
	db *gorm.DB
}

// NewGormAgentStorage creates a GORM agent storage instance
func NewGormAgentStorage(db *gorm.DB) *GormAgentStorage {
	return &GormAgentStorage{
		db: db,
	}
}

// CreateAgent creates an agent
func (s *GormAgentStorage) CreateAgent(ctx context.Context, agent *models.Agent) error {
	return s.db.WithContext(ctx).Create(agent).Error
}

// GetAgentByID retrieves an agent by ID
func (s *GormAgentStorage) GetAgentByID(ctx context.Context, id uint) (*models.Agent, error) {
	var agent models.Agent
	err := s.db.WithContext(ctx).First(&agent, id).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// GetAgentsByUserID retrieves a paginated list of agents by user ID
func (s *GormAgentStorage) GetAgentsByUserID(ctx context.Context, userID uint, offset, limit int) ([]*models.Agent, int64, error) {
	var agents []*models.Agent
	var total int64

	// get total count
	err := s.db.WithContext(ctx).Model(&models.Agent{}).Where("user_id = ?", userID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// get paginated data
	err = s.db.WithContext(ctx).Where("user_id = ?", userID).Offset(offset).Limit(limit).Find(&agents).Error
	return agents, total, err
}

// UpdateAgent updates an agent
func (s *GormAgentStorage) UpdateAgent(ctx context.Context, id uint, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&models.Agent{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteAgent deletes an agent
func (s *GormAgentStorage) DeleteAgent(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Agent{}, id).Error
}
