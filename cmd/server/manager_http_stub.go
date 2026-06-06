//go:build !manager

package main

import log "xiaozhi-esp32-server-golang/logger"

// StartManagerHTTP is a no-op when manager is not enabled at build time.
func StartManagerHTTP(configPath string) {
	log.Warn("embedded manager is not included; rebuild with -tags manager to enable it")
}

// StopManagerHTTP is a no-op when manager is not enabled at build time.
func StopManagerHTTP() {}
