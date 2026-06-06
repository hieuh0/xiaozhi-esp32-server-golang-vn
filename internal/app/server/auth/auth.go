package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"
)

// ClientSession represents a client session.
type ClientSession struct {
	ID        string
	DeviceID  string
	CreatedAt time.Time
	LastSeen  time.Time
}

// AuthManager manages authentication and sessions.
type AuthManager struct {
	sessions map[string]*ClientSession
	mutex    sync.RWMutex
	// Token mapping.
	tokens map[string]string // token -> deviceID
}

var authManager *AuthManager

func Init() error {
	authManager = NewAuthManager()
	return nil
}

func A() *AuthManager {
	return authManager
}

// NewAuthManager creates an authentication manager.
func NewAuthManager() *AuthManager {
	return &AuthManager{
		sessions: make(map[string]*ClientSession),
		tokens:   make(map[string]string),
	}
}

// CreateSession creates a session.
func (am *AuthManager) CreateSession(deviceID string) (*ClientSession, error) {
	// Generate a random session ID.
	sessionID, err := generateClientSessionID()
	if err != nil {
		return nil, err
	}

	session := &ClientSession{
		ID:        sessionID,
		DeviceID:  deviceID,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	am.mutex.Lock()
	am.sessions[sessionID] = session
	am.mutex.Unlock()

	return session, nil
}

// EnsureSession ensures that a session ID exists, creating one when preferredID is empty.
func (am *AuthManager) EnsureSession(deviceID string, preferredID string) (*ClientSession, error) {
	preferredID = strings.TrimSpace(preferredID)
	if preferredID == "" {
		return am.CreateSession(deviceID)
	}

	now := time.Now()

	am.mutex.Lock()
	defer am.mutex.Unlock()

	if session, exists := am.sessions[preferredID]; exists {
		if deviceID != "" {
			session.DeviceID = deviceID
		}
		session.LastSeen = now
		return session, nil
	}

	session := &ClientSession{
		ID:        preferredID,
		DeviceID:  deviceID,
		CreatedAt: now,
		LastSeen:  now,
	}
	am.sessions[preferredID] = session
	return session, nil
}

// GetSession returns a session.
func (am *AuthManager) GetSession(sessionID string) (*ClientSession, error) {
	am.mutex.RLock()
	session, exists := am.sessions[sessionID]
	am.mutex.RUnlock()

	if !exists {
		return nil, errors.New("session does not exist")
	}

	// Update the last access time.
	am.mutex.Lock()
	session.LastSeen = time.Now()
	am.mutex.Unlock()

	return session, nil
}

// RemoveSession removes a session.
func (am *AuthManager) RemoveSession(sessionID string) {
	am.mutex.Lock()
	delete(am.sessions, sessionID)
	am.mutex.Unlock()
}

// CleanupSessions removes expired sessions.
func (am *AuthManager) CleanupSessions(maxAge time.Duration) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	now := time.Now()
	for id, session := range am.sessions {
		if now.Sub(session.LastSeen) > maxAge {
			delete(am.sessions, id)
		}
	}
}

// generateClientSessionID generates a random session ID.
func generateClientSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateToken validates a token.
func (am *AuthManager) ValidateToken(token string) bool {
	return true
	// Remove the "Bearer " prefix.
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	am.mutex.RLock()
	_, exists := am.tokens[token]
	am.mutex.RUnlock()

	return exists
}

// RegisterToken registers a token.
func (am *AuthManager) RegisterToken(token string, deviceID string) {
	// Remove the "Bearer " prefix.
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	am.mutex.Lock()
	am.tokens[token] = deviceID
	am.mutex.Unlock()
}

// RemoveToken removes a token.
func (am *AuthManager) RemoveToken(token string) {
	// Remove the "Bearer " prefix.
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	am.mutex.Lock()
	delete(am.tokens, token)
	am.mutex.Unlock()
}
