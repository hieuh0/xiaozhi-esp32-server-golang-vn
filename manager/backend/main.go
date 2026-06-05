package main

import (
	"flag"
	"log"
	"xiaozhi/manager/backend/config"
	"xiaozhi/manager/backend/database"
	"xiaozhi/manager/backend/router"

	"github.com/gin-gonic/gin"
)

func main() {
	// Define command-line flags
	var configFile string
	flag.StringVar(&configFile, "config", "config/config.json", "path to config file")
	flag.StringVar(&configFile, "c", "config/config.json", "path to config file (shorthand)")
	flag.Parse()

	// Load configuration
	cfg := config.LoadWithPath(configFile)

	// Initialize database
	db := database.Init(cfg.Database)
	if db == nil {
		log.Fatal("database initialization failed, exiting")
	}
	defer database.Close(db)

	// Set Gin mode
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	r := router.Setup(db, cfg)

	// Start server
	log.Printf("using config file: %s", configFile)
	log.Printf("server listening on port: %s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("server failed to start:", err)
	}
}
