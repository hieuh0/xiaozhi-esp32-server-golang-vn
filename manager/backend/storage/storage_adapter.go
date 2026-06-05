package storage

import (
	"context"
	"xiaozhi/manager/backend/models"
)

// StorageAdapter storage adapter, bridges interface differences
type StorageAdapter struct {
	*GormBaseStorage
	userStorage   *GormUserStorage
	deviceStorage *GormDeviceStorage
	agentStorage  *GormAgentStorage
	configAdapter *ConfigAdapter
}

// NewStorageAdapter creates a storage adapter
func NewStorageAdapter(base *GormBaseStorage) *StorageAdapter {
	configStorage := NewGormConfigStorage(base.DB)
	return &StorageAdapter{
		GormBaseStorage: base,
		userStorage:     NewGormUserStorage(base.DB),
		deviceStorage:   NewGormDeviceStorage(base.DB),
		agentStorage:    NewGormAgentStorage(base.DB),
		configAdapter:   NewConfigAdapter(configStorage),
	}
}

// Connect connects to the database (adapter method)
func (a *StorageAdapter) Connect(ctx context.Context) error {
	// base already connected, this is just interface adaptation
	return nil
}

// Ping checks the database connection (adapter method)
func (a *StorageAdapter) Ping(ctx context.Context) error {
	return a.GormBaseStorage.Ping()
}

// UserStorage returns the user storage interface
func (a *StorageAdapter) UserStorage() UserStorage {
	return a.userStorage
}

// CreateUser creates a user
func (a *StorageAdapter) CreateUser(ctx context.Context, user *models.User) error {
	return a.userStorage.CreateUser(ctx, user)
}

// GetUsers retrieves all users
func (a *StorageAdapter) GetUsers(ctx context.Context, offset, limit int) ([]*models.User, int64, error) {
	return a.userStorage.GetUsers(ctx, offset, limit)
}

// GetUserByID retrieves a user by ID
func (a *StorageAdapter) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	return a.userStorage.GetUserByID(ctx, id)
}

// GetUserByUsername retrieves a user by username
func (a *StorageAdapter) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return a.userStorage.GetUserByUsername(ctx, username)
}

// GetUserByEmail retrieves a user by email
func (a *StorageAdapter) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return a.userStorage.GetUserByEmail(ctx, email)
}

// UpdateUser updates a user
func (a *StorageAdapter) UpdateUser(ctx context.Context, id uint, updates map[string]interface{}) error {
	return a.userStorage.UpdateUser(ctx, id, updates)
}

// DeleteUser deletes a user
func (a *StorageAdapter) DeleteUser(ctx context.Context, id uint) error {
	return a.userStorage.DeleteUser(ctx, id)
}

// DeviceStorage returns the device storage interface
func (a *StorageAdapter) DeviceStorage() DeviceStorage {
	return a.deviceStorage
}

// CreateDevice creates a device
func (a *StorageAdapter) CreateDevice(ctx context.Context, device *models.Device) error {
	return a.deviceStorage.CreateDevice(ctx, device)
}

// GetDeviceByID retrieves a device by ID
func (a *StorageAdapter) GetDeviceByID(ctx context.Context, id uint) (*models.Device, error) {
	return a.deviceStorage.GetDeviceByID(ctx, id)
}

// GetDeviceByCode retrieves a device by device code
func (a *StorageAdapter) GetDeviceByCode(ctx context.Context, deviceCode string) (*models.Device, error) {
	return a.deviceStorage.GetDeviceByCode(ctx, deviceCode)
}

// GetDevicesByUserID retrieves a paginated list of devices by user ID
func (a *StorageAdapter) GetDevicesByUserID(ctx context.Context, userID uint, offset, limit int) ([]*models.Device, int64, error) {
	return a.deviceStorage.GetDevicesByUserID(ctx, userID, offset, limit)
}

// UpdateDevice updates a device
func (a *StorageAdapter) UpdateDevice(ctx context.Context, id uint, updates map[string]interface{}) error {
	return a.deviceStorage.UpdateDevice(ctx, id, updates)
}

// DeleteDevice deletes a device
func (a *StorageAdapter) DeleteDevice(ctx context.Context, id uint) error {
	return a.deviceStorage.DeleteDevice(ctx, id)
}

// AgentStorage returns the agent storage interface
func (a *StorageAdapter) AgentStorage() AgentStorage {
	return a.agentStorage
}

// CreateAgent creates an agent
func (a *StorageAdapter) CreateAgent(ctx context.Context, agent *models.Agent) error {
	return a.agentStorage.CreateAgent(ctx, agent)
}

// GetAgentByID retrieves an agent by ID
func (a *StorageAdapter) GetAgentByID(ctx context.Context, id uint) (*models.Agent, error) {
	return a.agentStorage.GetAgentByID(ctx, id)
}

// GetAgentsByUserID retrieves a paginated list of agents by user ID
func (a *StorageAdapter) GetAgentsByUserID(ctx context.Context, userID uint, offset, limit int) ([]*models.Agent, int64, error) {
	return a.agentStorage.GetAgentsByUserID(ctx, userID, offset, limit)
}

// UpdateAgent updates an agent
func (a *StorageAdapter) UpdateAgent(ctx context.Context, id uint, updates map[string]interface{}) error {
	return a.agentStorage.UpdateAgent(ctx, id, updates)
}

// DeleteAgent deletes an agent
func (a *StorageAdapter) DeleteAgent(ctx context.Context, id uint) error {
	return a.agentStorage.DeleteAgent(ctx, id)
}

// ConfigStorage returns the config storage interface
func (a *StorageAdapter) ConfigStorage() ConfigStorage {
	return a.configAdapter
}
