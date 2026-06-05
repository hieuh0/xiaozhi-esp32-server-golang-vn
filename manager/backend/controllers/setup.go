package controllers

import (
	"log"
	"net/http"
	"xiaozhi/manager/backend/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type SetupController struct {
	DB *gorm.DB
}

type SetupRequest struct {
	AdminUsername string `json:"admin_username" binding:"required,min=3,max=50"`
	AdminPassword string `json:"admin_password" binding:"required,min=6,max=100"`
	AdminEmail    string `json:"admin_email" binding:"required,email"`
}

// CheckSetupStatus checks whether the database needs initialization
func (sc *SetupController) CheckSetupStatus(c *gin.Context) {
	if sc.DB == nil {
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": true,
			"message":     "database connection unavailable",
		})
		return
	}

	// Check whether the user table exists
	if !sc.DB.Migrator().HasTable(&models.User{}) {
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": true,
			"message":     "database schema not initialized",
		})
		return
	}

	// Check whether an admin user exists
	var count int64
	sc.DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)

	if count == 0 {
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": true,
			"message":     "admin account needs to be created",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"needs_setup": false,
		"message":     "system already initialized",
	})
}

// InitializeDatabase initializes the database
func (sc *SetupController) InitializeDatabase(c *gin.Context) {
	var req SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if sc.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection unavailable"})
		return
	}

	// Begin transaction
	tx := sc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start database transaction"})
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Auto-migrate schema
	log.Println("starting database schema auto-migration...")
	err := tx.AutoMigrate(
		&models.User{},
		&models.Device{},
		&models.Agent{},
		&models.Config{},
		&models.MCPMarketService{},
		&models.GlobalRole{},
		&models.SpeakerGroup{},
		&models.SpeakerSample{},
		&models.ChatMessage{},
	)
	if err != nil {
		tx.Rollback()
		log.Printf("database schema migration failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database schema migration failed: " + err.Error()})
		return
	}
	log.Println("database schema migration succeeded")

	// 2. Check whether an admin user already exists
	var existingAdmin models.User
	if err := tx.Where("role = ?", "admin").First(&existingAdmin).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin user already exists, cannot re-initialize"})
		return
	}

	// 3. Check whether the username already exists
	var existingUser models.User
	if err := tx.Where("username = ?", req.AdminUsername).First(&existingUser).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
		return
	}

	// 4. Check whether the email already exists
	if err := tx.Where("email = ?", req.AdminEmail).First(&existingUser).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		return
	}

	// 5. Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// 6. Create the admin user
	admin := models.User{
		Username: req.AdminUsername,
		Password: string(hashedPassword),
		Email:    req.AdminEmail,
		Role:     "admin",
	}

	if err := tx.Create(&admin).Error; err != nil {
		tx.Rollback()
		log.Printf("failed to create admin user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin user: " + err.Error()})
		return
	}

	// 7. Create some default global roles
	defaultRoles := []models.GlobalRole{
		{
			Name:        "Assistant",
			Description: "A friendly AI assistant that helps users with various problems",
			Prompt:      "You are a friendly and professional AI assistant. Please answer user questions in clear and concise language and provide useful suggestions.",
			IsDefault:   true,
		},
		{
			Name:        "Teacher",
			Description: "A patient teacher who can explain complex concepts in detail",
			Prompt:      "You are an experienced teacher. Please explain complex concepts in plain language and give concrete examples to aid understanding.",
			IsDefault:   false,
		},
		{
			Name:        "Friend",
			Description: "A caring friend who listens and accompanies",
			Prompt:      "You are a caring friend. Please communicate with users in a warm and understanding manner, offering emotional support and encouragement.",
			IsDefault:   false,
		},
	}

	for _, role := range defaultRoles {
		if err := tx.Create(&role).Error; err != nil {
			log.Printf("failed to create default role: %v", err)
			// Do not interrupt initialization, continue
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit database transaction"})
		return
	}

	log.Printf("database initialization succeeded, admin user: %s", req.AdminUsername)
	c.JSON(http.StatusOK, gin.H{
		"message": "database initialization succeeded",
		"admin": gin.H{
			"username": admin.Username,
			"email":    admin.Email,
			"role":     admin.Role,
		},
	})
}
