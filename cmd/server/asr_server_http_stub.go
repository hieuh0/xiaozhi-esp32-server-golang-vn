//go:build !asr_server

package main

import log "xiaozhi-esp32-server-golang/logger"

// StartAsrServerHTTP is a no-op when asr_server is not enabled at build time.
func StartAsrServerHTTP(configPath string) {
	log.Warn("embedded asr_server is not included; rebuild with -tags asr_server to enable it")
}

// StopAsrServerHTTP is a no-op when asr_server is not enabled at build time.
func StopAsrServerHTTP() {}
