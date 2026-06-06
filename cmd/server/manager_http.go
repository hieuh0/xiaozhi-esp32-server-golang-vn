//go:build manager

package main

import (
	"context"
	"net/http"
	"time"

	log "xiaozhi-esp32-server-golang/logger"
	mbconfig "xiaozhi/manager/backend/config"
	"xiaozhi/manager/backend/database"
	"xiaozhi/manager/backend/router"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	defaultManagerHTTPPort   = "9000"
	defaultManagerConfigPath = "manager.json"
)

var (
	managerHTTPServer *http.Server // Embedded manager HTTP service handle for graceful shutdown.
	managerDB         *gorm.DB     // Manager database, closed during shutdown.
)

// StartManagerHTTP starts the embedded manager HTTP service on its two ports.
// configPath is the manager config path; an empty value uses the default path.
func StartManagerHTTP(configPath string) {
	if configPath == "" {
		configPath = defaultManagerConfigPath
	}
	log.Infof("starting embedded manager HTTP service with config: %s", configPath)

	cfg := mbconfig.LoadWithPath(configPath)
	port := cfg.Server.Port
	if port == "" {
		port = defaultManagerHTTPPort
	}
	cfg.Server.Port = port

	db := database.Init(cfg.Database)
	if db == nil {
		log.Warn("manager database initialization failed; skipping manager HTTP startup")
		return
	}
	managerDB = db

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := router.Setup(db, cfg)

	managerHTTPServer = &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Infof("manager HTTP service listening on port %s", port)
		if err := managerHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("manager HTTP service exited unexpectedly: %v", err)
		}
	}()
}

// StopManagerHTTP gracefully stops embedded manager HTTP services and closes the database.
func StopManagerHTTP() {
	if managerHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := managerHTTPServer.Shutdown(ctx); err != nil {
			log.Warnf("manager HTTP shutdown timed out or failed: %v", err)
		}
		managerHTTPServer = nil
		log.Info("manager HTTP service stopped")
	}
	if managerDB != nil {
		database.Close(managerDB)
		managerDB = nil
	}
}
