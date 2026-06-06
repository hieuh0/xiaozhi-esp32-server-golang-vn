//go:build asr_server

package main

import (
	"context"
	"net/http"
	"time"

	"voice_server/server"
	log "xiaozhi-esp32-server-golang/logger"
)

const (
	defaultAsrServerConfigPath = "asr_server.json"
)

var (
	asrHTTPServer *http.Server // Embedded asr_server HTTP service handle for graceful shutdown.
)

// StartAsrServerHTTP starts the embedded asr_server HTTP service on a separate port.
// configPath is the asr_server config path; an empty value uses asr_server/config.json.
func StartAsrServerHTTP(configPath string) {
	if configPath == "" {
		configPath = defaultAsrServerConfigPath
	}
	log.Infof("starting embedded asr_server HTTP service with config: %s", configPath)

	handler, addr, readTimeout, err := server.Setup(configPath)
	if err != nil {
		log.Warnf("asr_server initialization failed; skipping startup: %v", err)
		return
	}

	asrHTTPServer = &http.Server{
		Addr:        addr,
		Handler:     handler,
		ReadTimeout: readTimeout,
	}

	go func() {
		log.Infof("asr_server HTTP service listening on %s", addr)
		if err := asrHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("asr_server HTTP service exited unexpectedly: %v", err)
		}
	}()
}

// StopAsrServerHTTP gracefully stops the embedded asr_server HTTP service.
func StopAsrServerHTTP() {
	if asrHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := asrHTTPServer.Shutdown(ctx); err != nil {
			log.Warnf("asr_server HTTP shutdown timed out or failed: %v", err)
		}
		asrHTTPServer = nil
		log.Info("asr_server HTTP service stopped")
	}
}
