package database

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"xiaozhi/manager/backend/config"
	"xiaozhi/manager/backend/models"
	"xiaozhi/manager/backend/services/configprovider"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Init(cfg config.DatabaseConfig) *gorm.DB {
	var db *gorm.DB
	var err error

	storageType := cfg.GetStorageType()

	if storageType == "sqlite" {
		if cfg.SQLite == nil {
			log.Println("SQLite config is empty, running in fallback mode (hardcoded user auth)")
			return nil
		}
		// ensure the directory containing the database file exists to avoid SQLite "unable to open database file"
		dir := filepath.Dir(cfg.SQLite.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("failed to create database directory %s: %v", dir, err)
			return nil
		}
		log.Println("using SQLite database:", cfg.SQLite.FilePath)
		db, err = gorm.Open(sqlite.Open(cfg.SQLite.FilePath), &gorm.Config{})
	} else {
		if cfg.MySQL == nil {
			log.Println("MySQL config is empty, running in fallback mode (hardcoded user auth)")
			return nil
		}
		// MySQL database connection
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.MySQL.Username, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}

	if err != nil {
		log.Println("database connection failed:", err)
		log.Println("running in fallback mode (hardcoded user auth)")
		return nil
	}

	log.Println("database connected")

	// auto-migrate database schema
	log.Println("starting auto-migration...")
	err = db.AutoMigrate(
		&models.User{},
		&models.APIToken{},
		&models.Device{},
		&models.Agent{},
		&models.KnowledgeBase{},
		&models.KnowledgeBaseDocument{},
		&models.AgentKnowledgeBase{},
		&models.Config{},
		&models.MCPMarketService{},
		&models.GlobalRole{},
		&models.Role{}, // unified role table
		&models.ChatMessage{},
		&models.SpeakerGroup{},
		&models.SpeakerSample{},
		&models.VoiceClone{},
		&models.VoiceCloneAudio{},
		&models.VoiceCloneTask{},
		&models.UserVoiceCloneQuota{},
	)
	if err != nil {
		log.Printf("auto-migration failed: %v", err)
		log.Println("running in fallback mode (hardcoded user auth)")
		return nil
	}
	log.Println("auto-migration completed")

	if err := dropDeprecatedAgentStatusColumn(db); err != nil {
		log.Printf("failed to drop deprecated agent status column: %v", err)
	}

	// migrate existing global role data to the new roles table
	log.Println("checking whether global role data migration is needed...")
	if err := migrateGlobalRolesToRoles(db); err != nil {
		log.Printf("global role data migration failed: %v", err)
		// migration failure does not block startup
	}
	if err := repairConfigProviders(db); err != nil {
		log.Printf("failed to repair config providers: %v", err)
	}
	return db
}

func dropDeprecatedAgentStatusColumn(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Agent{}) {
		return nil
	}
	hasColumn, err := hasDatabaseColumn(db, "agents", "status")
	if err != nil {
		return err
	}
	if !hasColumn {
		return nil
	}
	err = db.Exec("ALTER TABLE agents DROP COLUMN status").Error
	if err != nil {
		return err
	}
	log.Println("deprecated agents.status column dropped")
	return nil
}

func hasDatabaseColumn(db *gorm.DB, tableName, columnName string) (bool, error) {
	switch db.Dialector.Name() {
	case "sqlite":
		var columns []struct {
			Name string `gorm:"column:name"`
		}
		if err := db.Raw(fmt.Sprintf("PRAGMA table_info(%s)", tableName)).Scan(&columns).Error; err != nil {
			return false, err
		}
		for _, column := range columns {
			if column.Name == columnName {
				return true, nil
			}
		}
		return false, nil
	case "mysql":
		var count int64
		if err := db.Raw(
			"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?",
			tableName,
			columnName,
		).Scan(&count).Error; err != nil {
			return false, err
		}
		return count > 0, nil
	default:
		return db.Migrator().HasColumn(tableName, columnName), nil
	}
}

func Close(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Println("failed to get database connection:", err)
		return
	}
	sqlDB.Close()
}

// migrateGlobalRolesToRoles migrates existing global role data to the new roles table
func migrateGlobalRolesToRoles(db *gorm.DB) error {
	// check if roles table already has data
	var count int64
	if err := db.Table("roles").Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check roles table: %w", err)
	}

	// skip migration if roles table already has data
	if count > 0 {
		log.Println("roles table already has data, skipping migration")
		return nil
	}

	// check if global_roles table has data
	var globalRoleCount int64
	if err := db.Table("global_roles").Count(&globalRoleCount).Error; err != nil {
		// global_roles table may not exist, not an error
		log.Println("global_roles table does not exist, skipping migration")
		return nil
	}

	if globalRoleCount == 0 {
		log.Println("global_roles table has no data, skipping migration")
		return nil
	}

	log.Printf("migrating %d global role(s) to roles table...", globalRoleCount)

	// query all global roles
	var globalRoles []models.GlobalRole
	if err := db.Table("global_roles").Find(&globalRoles).Error; err != nil {
		return fmt.Errorf("failed to query global_roles: %w", err)
	}

	// convert and insert into roles table
	for _, gr := range globalRoles {
		role := models.Role{
			UserID:      nil, // global role user_id is NULL
			Name:        gr.Name,
			Description: gr.Description,
			Prompt:      gr.Prompt,
			RoleType:    "global",
			Status:      "active",
			SortOrder:   0,
			IsDefault:   gr.IsDefault,
			CreatedAt:   gr.CreatedAt,
			UpdatedAt:   gr.UpdatedAt,
		}
		if err := db.Create(&role).Error; err != nil {
			log.Printf("failed to insert role %s: %v", gr.Name, err)
			continue
		}
		log.Printf("migrated global role: %s", gr.Name)
	}

	log.Println("global role data migration completed")
	return nil
}

func repairConfigProviders(db *gorm.DB) error {
	var configs []models.Config
	if err := db.Where("type IN ?", []string{"vad", "asr", "llm", "tts", "memory", "vision"}).Find(&configs).Error; err != nil {
		return err
	}

	repaired := 0
	for _, cfg := range configs {
		var data map[string]interface{}
		if cfg.JsonData != "" {
			if err := json.Unmarshal([]byte(cfg.JsonData), &data); err != nil {
				log.Printf("skipping provider repair, failed to parse json_data type=%s config_id=%s: %v", cfg.Type, cfg.ConfigID, err)
				continue
			}
		}
		if data == nil {
			data = map[string]interface{}{}
		}

		provider := configprovider.NormalizeExistingProvider(cfg.Type, cfg.Provider, cfg.ConfigID, data)
		if provider == "" || provider == cfg.Provider {
			if jsonProvider, _ := data["provider"].(string); strings.TrimSpace(jsonProvider) == "" || strings.EqualFold(strings.TrimSpace(jsonProvider), provider) {
				continue
			}
		}

		updates := map[string]interface{}{}
		if provider != "" && provider != cfg.Provider {
			updates["provider"] = provider
		}
		if provider != "" {
			if jsonProvider, _ := data["provider"].(string); !strings.EqualFold(strings.TrimSpace(jsonProvider), provider) {
				data["provider"] = provider
				bytes, err := json.Marshal(data)
				if err != nil {
					return err
				}
				updates["json_data"] = string(bytes)
			}
		}
		if len(updates) == 0 {
			continue
		}
		if err := db.Model(&models.Config{}).Where("id = ?", cfg.ID).Updates(updates).Error; err != nil {
			return err
		}
		repaired++
	}

	if repaired > 0 {
		log.Printf("repaired %d config provider(s)", repaired)
	}
	return nil
}
