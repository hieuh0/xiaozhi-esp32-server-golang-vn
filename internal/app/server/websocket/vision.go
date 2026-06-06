package websocket

import (
	"io"
	"net/http"
	"strings"
	"xiaozhi-esp32-server-golang/internal/app/server/chat"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/spf13/viper"
)

// handleVisionAPI handles the image recognition API
func (s *WebSocketServer) handleVisionAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Warnf("Image recognition request method error: %s", r.Method)
		http.Error(w, "Only POST requests are supported", http.StatusMethodNotAllowed)
		return
	}

	// Get Device-Id and Client-Id from headers
	deviceId := r.Header.Get("Device-Id")
	clientId := r.Header.Get("Client-Id")
	_ = clientId
	if deviceId == "" {
		log.Errorf("Image recognition request missing Device-Id")
		http.Error(w, "Missing Device-Id", http.StatusBadRequest)
		return
	}
	log.Infof("Image recognition request deviceId=%s", deviceId)

	if viper.GetBool("vision.enable_auth") {

		// Get Bearer token from Authorization header
		authToken := r.Header.Get("Authorization")
		if authToken == "" {
			log.Errorf("Image recognition request missing Authorization deviceId=%s", deviceId)
			http.Error(w, "Missing Authorization", http.StatusBadRequest)
			return
		}
		authToken = strings.TrimPrefix(authToken, "Bearer ")

		err := chat.VisvionAuth(authToken)
		if err != nil {
			log.Errorf("Image recognition authentication failed deviceId=%s err=%v", deviceId, err)
			http.Error(w, "Image recognition authentication failed", http.StatusUnauthorized)
			return
		}
		log.Infof("Image recognition authentication passed deviceId=%s", deviceId)
	}

	// Parse multipart form, max 10MB
	question := r.FormValue("question")
	if question == "" {
		log.Warnf("Image recognition request missing question deviceId=%s", deviceId)
		http.Error(w, "Missing question parameter", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Errorf("Image recognition request missing file or read failed deviceId=%s err=%v", deviceId, err)
		http.Error(w, "Missing file parameter or file read failed", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Errorf("Image recognition file read failed deviceId=%s err=%v", deviceId, err)
		http.Error(w, "File read failed", http.StatusInternalServerError)
		return
	}

	file.Close()
	log.Infof("Image recognition received file deviceId=%s filename=%s size=%d question=%s", deviceId, header.Filename, len(fileBytes), question)

	result, err := chat.HandleVllm(deviceId, fileBytes, question)
	if err != nil {
		log.Errorf("Image recognition failed deviceId=%s err=%v", deviceId, err)
		http.Error(w, "Image recognition failed", http.StatusInternalServerError)
		return
	}

	log.Infof("Image recognition succeeded deviceId=%s resultLen=%d", deviceId, len(result))
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}
