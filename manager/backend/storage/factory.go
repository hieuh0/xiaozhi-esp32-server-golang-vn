package storage

import (
	"fmt"

	"xiaozhi/manager/backend/config"
	"xiaozhi/manager/backend/storage/mysql"
	"xiaozhi/manager/backend/storage/sqlite"
)

// StorageType storage type
type StorageType string

const (
	StorageTypeMySQL  StorageType = "mysql"
	StorageTypeSQLite StorageType = "sqlite"
)

// Factory storage factory
type Factory struct{}

// NewFactory creates a storage factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateStorage creates a storage instance
func CreateStorage(dbConfig config.DatabaseConfig) (*StorageAdapter, error) {
	// determine storage type from config
	storageType := dbConfig.GetStorageType()

	switch StorageType(storageType) {
	case StorageTypeSQLite:
		if dbConfig.SQLite == nil {
			return nil, fmt.Errorf("SQLite config is required")
		}
		// validate SQLite config
		if err := sqlite.ValidateConfig(dbConfig.SQLite); err != nil {
			return nil, fmt.Errorf("invalid SQLite config: %w", err)
		}
		// create SQLite config
		sqliteConfig := sqlite.NewConfigFromDatabase(dbConfig.SQLite)
		// create SQLite storage
		sqliteStorage, err := sqlite.NewStorage(sqliteConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite storage: %w", err)
		}
		// create base storage
		baseStorage := NewGormBaseStorage(sqliteStorage.DB)
		// return adapter
		return NewStorageAdapter(baseStorage), nil

	case StorageTypeMySQL:
		if dbConfig.MySQL == nil {
			return nil, fmt.Errorf("MySQL config is required")
		}
		// validate MySQL config
		if err := mysql.ValidateConfig(dbConfig.MySQL); err != nil {
			return nil, fmt.Errorf("invalid MySQL config: %w", err)
		}
		// create MySQL config
		mysqlConfig := mysql.NewConfigFromDatabase(dbConfig.MySQL)
		// create MySQL storage
		mysqlStorage, err := mysql.NewStorage(mysqlConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create MySQL storage: %w", err)
		}
		// create base storage
		baseStorage := NewGormBaseStorage(mysqlStorage.DB)
		// return adapter
		return NewStorageAdapter(baseStorage), nil

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}

// GetSupportedTypes returns supported storage types
func (f *Factory) GetSupportedTypes() []StorageType {
	return []StorageType{
		StorageTypeMySQL,
		StorageTypeSQLite,
	}
}
