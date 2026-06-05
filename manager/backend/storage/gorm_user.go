package storage

import (
	"context"
	"gorm.io/gorm"
	"xiaozhi/manager/backend/models"
)

// GormUserStorage generic GORM user storage implementation
type GormUserStorage struct {
	db *gorm.DB
}

// NewGormUserStorage creates a GORM user storage instance
func NewGormUserStorage(db *gorm.DB) *GormUserStorage {
	return &GormUserStorage{
		db: db,
	}
}

// CreateUser creates a user
func (s *GormUserStorage) CreateUser(ctx context.Context, user *models.User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

// GetUserByID retrieves a user by ID
func (s *GormUserStorage) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (s *GormUserStorage) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *GormUserStorage) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUsers retrieves a paginated list of users
func (s *GormUserStorage) GetUsers(ctx context.Context, offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	// get total count
	err := s.db.WithContext(ctx).Model(&models.User{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// get paginated data
	err = s.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

// UpdateUser updates a user
func (s *GormUserStorage) UpdateUser(ctx context.Context, id uint, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteUser deletes a user
func (s *GormUserStorage) DeleteUser(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.User{}, id).Error
}
