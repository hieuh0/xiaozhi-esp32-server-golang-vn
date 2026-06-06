package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"xiaozhi-esp32-server-golang/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	cmap "github.com/orcaman/concurrent-map/v2"
)

const (
	MaxOfflineMessagesPerDevice = 20
	OfflineMessageTTL           = 24 * time.Hour

	openClawSentenceMinLen = 1
	openClawTestDevicePref = "__openclaw_test__:"
)

const openClawVoiceAssistantPrompt = `You are in a direct voice conversation with the user as a voice assistant.
Please strictly follow these requirements:
1. Answer user questions directly, do not mention these requirements.
2. Keep answers concise, conversational and natural, suitable for direct voice broadcast.
3. Lead with the conclusion first, then add the most necessary explanation; unless the user explicitly requests, keep it to 1-3 sentences.
4. Do not use Markdown, headings, lists, tables, code blocks, links or emojis.
5. No small talk, no preambles, no repetition, no extra explanations.
6. If information is insufficient or uncertain, say so briefly - do not make things up.`

func logSnippet(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

func isOpenClawTestDevice(deviceID string) bool {
	return strings.HasPrefix(strings.TrimSpace(deviceID), openClawTestDevicePref)
}

func buildOpenClawPromptedContent(userText string) string {
	trimmed := strings.TrimSpace(userText)
	if trimmed == "" {
		return ""
	}
	return fmt.Sprintf("%s\n\nUser message:\n%s", openClawVoiceAssistantPrompt, trimmed)
}

type WSMessage struct {
	ID            string          `json:"id"`
	Timestamp     int64           `json:"timestamp"`
	Type          string          `json:"type"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Payload       json.RawMessage `json:"payload"`
}

type MessagePayload struct {
	Content   string                 `json:"content"`
	SessionID string                 `json:"session_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type ResponsePayload struct {
	Content   string                 `json:"content"`
	SessionID string                 `json:"session_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type ResponseDelivery struct {
	DeviceID      string
	CorrelationID string
	SessionID     string
	Text          string
	IsStart       bool
	IsEnd         bool
	Metadata      map[string]interface{}
}

type OfflineMessage struct {
	Text          string
	CorrelationID string
	IsEnd         bool
	CreatedAt     time.Time
}

type pendingRoute struct {
	DeviceID  string
	CreatedAt time.Time
}

type responseStreamState struct {
	DeviceID    string
	Buffer      string
	EmittedText string
	PendingText string
	HasDelta    bool
	IsFirst     bool
	LastSeq     int64
	CreatedAt   time.Time
}

type AgentSession struct {
	agentID string
	conn    *websocket.Conn

	ctx    context.Context
	cancel context.CancelFunc

	writeMu sync.Mutex
	pending sync.Map // correlation_id -> pendingRoute
	modes   sync.Map // device_id -> bool
	streams sync.Map // correlation_id -> *responseStreamState
}

func newAgentSession(agentID string, conn *websocket.Conn) *AgentSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &AgentSession{
		agentID: agentID,
		conn:    conn,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (s *AgentSession) Send(msg WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Warnf("OpenClaw ws marshal failed: agent=%s type=%s id=%s corr=%s err=%v", s.agentID, msg.Type, msg.ID, msg.CorrelationID, err)
		return err
	}

	logger.Debugf(
		"OpenClaw ws send start: agent=%s type=%s id=%s corr=%s payload_bytes=%d frame_bytes=%d",
		s.agentID,
		msg.Type,
		msg.ID,
		msg.CorrelationID,
		len(msg.Payload),
		len(data),
	)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.Warnf("OpenClaw ws send failed: agent=%s type=%s id=%s corr=%s err=%v", s.agentID, msg.Type, msg.ID, msg.CorrelationID, err)
		return err
	}
	logger.Debugf("OpenClaw ws send ok: agent=%s type=%s id=%s corr=%s", s.agentID, msg.Type, msg.ID, msg.CorrelationID)
	return nil
}

func (s *AgentSession) TrackPending(correlationID string, deviceID string) {
	correlationID = strings.TrimSpace(correlationID)
	deviceID = strings.TrimSpace(deviceID)
	if correlationID == "" || deviceID == "" {
		return
	}
	s.pending.Store(correlationID, pendingRoute{
		DeviceID:  deviceID,
		CreatedAt: time.Now(),
	})
	logger.Debugf("OpenClaw pending tracked: agent=%s correlation_id=%s device=%s", s.agentID, correlationID, deviceID)
}

func (s *AgentSession) RemovePending(correlationID string) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return
	}
	s.pending.Delete(correlationID)
	logger.Debugf("OpenClaw pending removed: agent=%s correlation_id=%s", s.agentID, correlationID)
}

func (s *AgentSession) ResolvePending(correlationID string) (string, bool) {
	if strings.TrimSpace(correlationID) == "" {
		return "", false
	}

	value, ok := s.pending.Load(correlationID)
	if !ok {
		return "", false
	}
	s.pending.Delete(correlationID)

	route, ok := value.(pendingRoute)
	if !ok {
		return "", false
	}
	logger.Debugf("OpenClaw pending resolved: agent=%s correlation_id=%s device=%s", s.agentID, correlationID, route.DeviceID)
	return route.DeviceID, route.DeviceID != ""
}

func (s *AgentSession) PeekPending(correlationID string) (string, bool) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return "", false
	}
	value, ok := s.pending.Load(correlationID)
	if !ok {
		return "", false
	}
	route, ok := value.(pendingRoute)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(route.DeviceID), strings.TrimSpace(route.DeviceID) != ""
}

func (s *AgentSession) LoadOrCreateStream(correlationID string) *responseStreamState {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return nil
	}
	if existing, ok := s.streams.Load(correlationID); ok {
		if state, ok := existing.(*responseStreamState); ok && state != nil {
			return state
		}
	}
	newState := &responseStreamState{
		IsFirst:   true,
		CreatedAt: time.Now(),
	}
	actual, _ := s.streams.LoadOrStore(correlationID, newState)
	state, _ := actual.(*responseStreamState)
	return state
}

func (s *AgentSession) GetStream(correlationID string) (*responseStreamState, bool) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return nil, false
	}
	value, ok := s.streams.Load(correlationID)
	if !ok {
		return nil, false
	}
	state, ok := value.(*responseStreamState)
	return state, ok && state != nil
}

func (s *AgentSession) RemoveStream(correlationID string) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return
	}
	s.streams.Delete(correlationID)
}

func (s *AgentSession) IsSameConn(conn *websocket.Conn) bool {
	return s.conn == conn
}

func (s *AgentSession) EnterMode(deviceID string) bool {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false
	}
	s.modes.Store(deviceID, true)
	return true
}

func (s *AgentSession) ExitMode(deviceID string) bool {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false
	}
	s.modes.Delete(deviceID)
	return true
}

func (s *AgentSession) IsModeEnabled(deviceID string) bool {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false
	}
	value, ok := s.modes.Load(deviceID)
	if !ok {
		return false
	}
	enabled, ok := value.(bool)
	return ok && enabled
}

func (s *AgentSession) copyModesFrom(other *AgentSession) {
	if other == nil {
		return
	}
	other.modes.Range(func(key, value interface{}) bool {
		deviceID, ok := key.(string)
		if !ok {
			return true
		}
		enabled, ok := value.(bool)
		if !ok || !enabled {
			return true
		}
		s.modes.Store(deviceID, true)
		return true
	})
}

func (s *AgentSession) Close() {
	s.cancel()
	_ = s.conn.Close()
}

type Manager struct {
	sessions cmap.ConcurrentMap[string, *AgentSession]

	offlineMu sync.Mutex
	offline   map[string][]OfflineMessage
}

var (
	defaultManager *Manager
	managerOnce    sync.Once
)

func GetManager() *Manager {
	managerOnce.Do(func() {
		defaultManager = &Manager{
			sessions: cmap.New[*AgentSession](),
			offline:  make(map[string][]OfflineMessage),
		}
	})
	return defaultManager
}

func (m *Manager) RegisterAgentConnection(agentID string, conn *websocket.Conn) *AgentSession {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil
	}

	newSession := newAgentSession(agentID, conn)
	if oldSession, ok := m.sessions.Get(agentID); ok && oldSession != nil {
		newSession.copyModesFrom(oldSession)
		logger.Infof("OpenClaw session replaced: agent=%s", agentID)
		oldSession.Close()
	}
	m.sessions.Set(agentID, newSession)
	logger.Infof("OpenClaw session registered: agent=%s", agentID)
	return newSession
}

func (m *Manager) UnregisterAgentConnection(agentID string, session *AgentSession) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return
	}

	current, ok := m.sessions.Get(agentID)
	if !ok || current == nil {
		return
	}

	if session == nil || current == session {
		m.sessions.Remove(agentID)
		logger.Infof("OpenClaw session unregistered: agent=%s", agentID)
	}
}

func (m *Manager) GetAgentSession(agentID string) *AgentSession {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil
	}
	session, ok := m.sessions.Get(agentID)
	if !ok {
		return nil
	}
	return session
}

func (m *Manager) SendMessage(agentID string, deviceID string, content string, sessionID string) (string, error) {
	rawContent := content
	agentID = strings.TrimSpace(agentID)
	deviceID = strings.TrimSpace(deviceID)
	content = strings.TrimSpace(content)
	sessionID = strings.TrimSpace(sessionID)

	logger.Debugf(
		"OpenClaw SendMessage requested: agent=%s device=%s session=%s content_len=%d content_trim_len=%d content_snippet=%q",
		agentID,
		deviceID,
		sessionID,
		len(rawContent),
		len(content),
		logSnippet(content, 64),
	)

	if agentID == "" {
		err := fmt.Errorf("agentID is required")
		logger.Warnf("OpenClaw SendMessage rejected: %v", err)
		return "", err
	}
	if deviceID == "" {
		err := fmt.Errorf("deviceID is required")
		logger.Warnf("OpenClaw SendMessage rejected: agent=%s err=%v", agentID, err)
		return "", err
	}
	if content == "" {
		err := fmt.Errorf("content is required")
		logger.Warnf("OpenClaw SendMessage rejected: agent=%s device=%s err=%v", agentID, deviceID, err)
		return "", err
	}
	promptedContent := buildOpenClawPromptedContent(content)
	if promptedContent == "" {
		err := fmt.Errorf("prompted content is required")
		logger.Warnf("OpenClaw SendMessage rejected after prompt wrap: agent=%s device=%s err=%v", agentID, deviceID, err)
		return "", err
	}

	session := m.GetAgentSession(agentID)
	if session == nil {
		err := fmt.Errorf("openclaw session not found for agent %s", agentID)
		logger.Warnf("OpenClaw SendMessage rejected: agent=%s device=%s session=%s err=%v", agentID, deviceID, sessionID, err)
		return "", err
	}

	messageID := uuid.NewString()
	payloadBytes, err := json.Marshal(MessagePayload{
		Content:   promptedContent,
		SessionID: sessionID,
		Metadata: map[string]interface{}{
			"device_id": deviceID,
			"agent_id":  agentID,
			"stream":    true,
		},
	})
	if err != nil {
		logger.Warnf("OpenClaw SendMessage payload marshal failed: agent=%s device=%s message_id=%s err=%v", agentID, deviceID, messageID, err)
		return "", err
	}

	logger.Debugf(
		"OpenClaw outbound prompt applied: agent=%s device=%s session=%s prompted_len=%d user_snippet=%q",
		agentID,
		deviceID,
		sessionID,
		len(promptedContent),
		logSnippet(content, 64),
	)
	session.TrackPending(messageID, deviceID)
	logger.Debugf("OpenClaw SendMessage dispatching: agent=%s device=%s session=%s message_id=%s payload_bytes=%d", agentID, deviceID, sessionID, messageID, len(payloadBytes))
	err = session.Send(WSMessage{
		ID:        messageID,
		Timestamp: time.Now().UnixMilli(),
		Type:      "message",
		Payload:   payloadBytes,
	})
	if err != nil {
		session.RemovePending(messageID)
		logger.Warnf("OpenClaw SendMessage send failed: agent=%s device=%s session=%s message_id=%s err=%v", agentID, deviceID, sessionID, messageID, err)
		return "", err
	}

	logger.Debugf("OpenClaw SendMessage dispatched: agent=%s device=%s session=%s message_id=%s", agentID, deviceID, sessionID, messageID)
	return messageID, nil
}

func (m *Manager) EnterMode(agentID string, deviceID string) bool {
	agentID = strings.TrimSpace(agentID)
	deviceID = strings.TrimSpace(deviceID)
	session := m.GetAgentSession(agentID)
	if session == nil {
		logger.Warnf("OpenClaw EnterMode failed: agent=%s device=%s reason=no_agent_session", agentID, deviceID)
		return false
	}
	ok := session.EnterMode(deviceID)
	logger.Infof("OpenClaw mode enabled: agent=%s device=%s ok=%v", agentID, deviceID, ok)
	return ok
}

func (m *Manager) ExitMode(agentID string, deviceID string) bool {
	agentID = strings.TrimSpace(agentID)
	deviceID = strings.TrimSpace(deviceID)
	session := m.GetAgentSession(agentID)
	if session == nil {
		logger.Debugf("OpenClaw ExitMode ignored: agent=%s device=%s reason=no_agent_session", agentID, deviceID)
		return false
	}
	ok := session.ExitMode(deviceID)
	logger.Infof("OpenClaw mode disabled: agent=%s device=%s ok=%v", agentID, deviceID, ok)
	return ok
}

func (m *Manager) IsModeEnabled(agentID string, deviceID string) bool {
	agentID = strings.TrimSpace(agentID)
	deviceID = strings.TrimSpace(deviceID)
	session := m.GetAgentSession(agentID)
	if session == nil {
		logger.Debugf("OpenClaw mode check: agent=%s device=%s enabled=false reason=no_agent_session", agentID, deviceID)
		return false
	}
	enabled := session.IsModeEnabled(deviceID)
	logger.Debugf("OpenClaw mode check: agent=%s device=%s enabled=%v", agentID, deviceID, enabled)
	return enabled
}

func (m *Manager) HandleResponse(
	agentID string,
	session *AgentSession,
	correlationID string,
	payload ResponsePayload,
	deliver func(event ResponseDelivery) bool,
) {
	agentID = strings.TrimSpace(agentID)
	correlationID = strings.TrimSpace(correlationID)
	sessionID := strings.TrimSpace(payload.SessionID)
	content := strings.TrimSpace(payload.Content)
	streamDone := parseMetadataBool(payload.Metadata, "done")
	streamSeq := parseMetadataInt64(payload.Metadata, "seq")
	streamID := readMetadataString(payload.Metadata, "stream_id")
	streamPhase := strings.ToLower(readMetadataString(payload.Metadata, "phase"))
	streamContentType := strings.ToLower(readMetadataString(payload.Metadata, "content_type"))
	isSnapshotFrame := isOpenClawSnapshotFrame(streamPhase, streamContentType)
	isStreaming := streamDone || streamSeq > 0 || streamID != "" || streamPhase != "" || streamContentType != ""

	//Non-streaming is considered one-time completion by default; streaming responses missing correlation_id are also downgraded to one-time processing.
	if !isStreaming || correlationID == "" {
		streamDone = true
	}

	deviceID := ""
	routeSource := ""
	if payload.Metadata != nil {
		deviceID = readMetadataString(payload.Metadata, "device_id")
		if deviceID != "" {
			routeSource = "metadata.device_id"
		}
	}
	if deviceID == "" && session != nil && correlationID != "" {
		if state, ok := session.GetStream(correlationID); ok && state != nil {
			if cached := strings.TrimSpace(state.DeviceID); cached != "" {
				deviceID = cached
				routeSource = "stream.correlation_id"
			}
		}
	}
	if deviceID == "" && session != nil && correlationID != "" {
		if resolvedDeviceID, ok := session.PeekPending(correlationID); ok {
			deviceID = strings.TrimSpace(resolvedDeviceID)
			if deviceID != "" {
				routeSource = "pending.correlation_id"
			}
		}
	}

	if deviceID == "" {
		logger.Warnf(
			"OpenClaw response missing device route, agent=%s correlation_id=%s session=%s done=%v seq=%d stream_id=%s phase=%s content_type=%s",
			agentID,
			correlationID,
			sessionID,
			streamDone,
			streamSeq,
			streamID,
			streamPhase,
			streamContentType,
		)
		return
	}

	var state *responseStreamState
	if session != nil && correlationID != "" {
		state = session.LoadOrCreateStream(correlationID)
		if state != nil && strings.TrimSpace(state.DeviceID) == "" {
			state.DeviceID = deviceID
		}
	}

	if state != nil && streamSeq > 0 {
		if state.LastSeq > 0 && streamSeq <= state.LastSeq {
			logger.Warnf(
				"OpenClaw response seq ignored: agent=%s correlation_id=%s seq=%d last_seq=%d stream_id=%s phase=%s content_type=%s",
				agentID,
				correlationID,
				streamSeq,
				state.LastSeq,
				streamID,
				streamPhase,
				streamContentType,
			)
			return
		}
		state.LastSeq = streamSeq
	}

	incrementalContent := normalizeOpenClawSpeechText(content)
	workingText := incrementalContent
	bufferedSnapshot := false
	if state != nil {
		if isSnapshotFrame {
			snapshotBuffer := state.applySnapshotContent(content)
			incrementalContent = ""
			workingText = ""
			bufferedSnapshot = true
			if content != "" {
				logger.Debugf(
					"OpenClaw snapshot buffered: agent=%s device=%s correlation_id=%s seq=%d stream_id=%s phase=%s content_type=%s snapshot_buffer_len=%d",
					agentID,
					deviceID,
					correlationID,
					streamSeq,
					streamID,
					streamPhase,
					streamContentType,
					len(snapshotBuffer),
				)
			}
		} else {
			incrementalContent = state.toIncrementalContent(content, streamDone)
			if content != "" && incrementalContent == "" {
				action := "deferred"
				if state.HasDelta {
					action = "ignored"
				}
				logger.Debugf(
					"OpenClaw snapshot %s: agent=%s device=%s correlation_id=%s seq=%d stream_id=%s phase=%s content_type=%s",
					action,
					agentID,
					deviceID,
					correlationID,
					streamSeq,
					streamID,
					streamPhase,
					streamContentType,
				)
			}
			if incrementalContent != "" {
				state.Buffer = normalizeOpenClawSpeechText(state.Buffer + incrementalContent)
			}
			workingText = strings.TrimSpace(state.Buffer)
		}
	}

	logger.Infof(
		"OpenClaw response routed: agent=%s device=%s session=%s correlation_id=%s route=%s done=%v seq=%d stream_id=%s phase=%s content_type=%s content_len=%d content_snippet=%q",
		agentID,
		deviceID,
		sessionID,
		correlationID,
		routeSource,
		streamDone,
		streamSeq,
		streamID,
		streamPhase,
		streamContentType,
		len(content),
		logSnippet(content, 64),
	)

	isFirst := true
	if state != nil {
		isFirst = state.IsFirst
	}

	sentences := make([]string, 0)
	remaining := strings.TrimSpace(workingText)
	if remaining != "" {
		sentences, remaining = extractOpenClawSentences(workingText, openClawSentenceMinLen, isFirst)
	}
	if state != nil {
		if bufferedSnapshot {
			remaining = strings.TrimSpace(state.Buffer)
		} else {
			state.Buffer = remaining
		}
	}

	emit := func(text string, isStart bool, isEnd bool) {
		text = strings.TrimSpace(text)
		if text == "" && !isEnd {
			return
		}
		event := ResponseDelivery{
			DeviceID:      deviceID,
			CorrelationID: correlationID,
			SessionID:     sessionID,
			Text:          text,
			IsStart:       isStart,
			IsEnd:         isEnd,
			Metadata:      payload.Metadata,
		}
		if deliver != nil && deliver(event) {
			logger.Debugf(
				"OpenClaw response delivered online: agent=%s device=%s correlation_id=%s start=%v end=%v text_len=%d",
				agentID,
				deviceID,
				correlationID,
				isStart,
				isEnd,
				len(text),
			)
			return
		}
		logger.Warnf("OpenClaw response queued offline: agent=%s device=%s correlation_id=%s", agentID, deviceID, correlationID)
		m.AddOfflineMessage(deviceID, text, correlationID, isEnd)
	}

	//The conversation test device (__openclaw_test__) directly transparently transmits fragments to avoid splitting sentences, which will cause the offline queue entries to skyrocket and trigger the truncation of the upper limit of 20.
	if isOpenClawTestDevice(deviceID) {
		if incrementalContent != "" {
			emit(incrementalContent, isFirst, streamDone)
			if state != nil {
				state.markEmitted(incrementalContent)
				state.IsFirst = false
				state.Buffer = ""
			}
		} else if streamDone {
			finalText := ""
			finalIsStart := isFirst
			if state != nil {
				finalText = strings.TrimSpace(state.Buffer)
				finalIsStart = state.IsFirst
			}
			emit(finalText, finalIsStart, true)
			if state != nil {
				if finalText != "" {
					state.markEmitted(finalText)
					state.IsFirst = false
					state.Buffer = ""
				}
			}
		}

		if streamDone && session != nil && correlationID != "" {
			session.RemovePending(correlationID)
			session.RemoveStream(correlationID)
		}
		return
	}

	for i, sentence := range sentences {
		emit(sentence, isFirst && i == 0, false)
		if state != nil {
			state.markEmitted(sentence)
		}
	}
	if state != nil && len(sentences) > 0 {
		state.IsFirst = false
	}

	if !streamDone {
		return
	}

	finalText := remaining
	finalIsStart := isFirst
	if state != nil {
		finalText = strings.TrimSpace(state.Buffer)
		finalIsStart = state.IsFirst
	}

	if finalText != "" {
		emit(finalText, finalIsStart, true)
		if state != nil {
			state.markEmitted(finalText)
			state.IsFirst = false
			state.Buffer = ""
		}
	} else {
		//The end frame allows empty content, which is used to drive the receiving end to finish.
		emit("", finalIsStart, true)
	}

	if session != nil && correlationID != "" {
		session.RemovePending(correlationID)
		session.RemoveStream(correlationID)
	}
}

func (s *responseStreamState) toIncrementalContent(content string, streamDone bool) string {
	if s == nil {
		return normalizeOpenClawSpeechText(content)
	}

	normalizedContent := normalizeOpenClawSpeechText(content)
	if normalizedContent == "" {
		if streamDone && !s.HasDelta && s.PendingText != "" {
			snapshot := s.PendingText
			s.PendingText = ""
			return snapshot
		}
		return ""
	}

	if s.HasDelta {
		accountedText := s.accountedText()
		if accountedText != "" {
			if delta, ok := trimOpenClawCanonicalPrefix(normalizedContent, accountedText); ok {
				return delta
			}
		}
		return normalizedContent
	}

	accountedText := s.accountedText()
	if accountedText != "" {
		s.HasDelta = true
		if delta, ok := trimOpenClawCanonicalPrefix(normalizedContent, accountedText); ok {
			return delta
		}
		return normalizedContent
	}

	if s.PendingText == "" {
		s.PendingText = normalizedContent
		if streamDone {
			snapshot := s.PendingText
			s.PendingText = ""
			return snapshot
		}
		return ""
	}

	if isOpenClawCanonicalGrowth(s.PendingText, normalizedContent) {
		if len(openClawCanonicalKey(normalizedContent)) >= len(openClawCanonicalKey(s.PendingText)) {
			s.PendingText = normalizedContent
		}
		if streamDone {
			snapshot := s.PendingText
			s.PendingText = ""
			return snapshot
		}
		return ""
	}

	if isOpenClawPunctuationOnly(normalizedContent) {
		s.PendingText = normalizeOpenClawSpeechText(s.PendingText + normalizedContent)
		if streamDone {
			snapshot := s.PendingText
			s.PendingText = ""
			return snapshot
		}
		return ""
	}

	s.HasDelta = true
	combined := normalizeOpenClawSpeechText(s.PendingText + normalizedContent)
	s.PendingText = ""
	return combined
}

func (s *responseStreamState) applySnapshotContent(content string) string {
	if s == nil {
		return normalizeOpenClawSpeechText(content)
	}

	normalizedContent := normalizeOpenClawSpeechText(content)
	if normalizedContent == "" {
		return ""
	}

	s.PendingText = ""

	snapshotBuffer := normalizedContent
	emittedText := normalizeOpenClawSpeechText(s.EmittedText)
	if emittedText != "" {
		if suffix, ok := trimOpenClawCanonicalPrefix(normalizedContent, emittedText); ok {
			snapshotBuffer = suffix
		}
	}

	s.Buffer = normalizeOpenClawSpeechText(snapshotBuffer)
	return s.Buffer
}

func (s *responseStreamState) markEmitted(text string) {
	if s == nil {
		return
	}
	normalized := normalizeOpenClawSpeechText(text)
	if normalized == "" {
		return
	}
	s.EmittedText = normalizeOpenClawSpeechText(s.EmittedText + normalized)
}

func (s *responseStreamState) accountedText() string {
	if s == nil {
		return ""
	}
	return normalizeOpenClawSpeechText(strings.TrimSpace(s.EmittedText) + strings.TrimSpace(s.Buffer))
}

func readMetadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	value, exists := metadata[key]
	if !exists || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func parseMetadataBool(metadata map[string]interface{}, key string) bool {
	if metadata == nil {
		return false
	}
	value, exists := metadata[key]
	if !exists || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		return err == nil && b
	case json.Number:
		n, err := v.Int64()
		return err == nil && n != 0
	case float64:
		return v != 0
	case float32:
		return v != 0
	case int:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	default:
		return false
	}
}

func parseMetadataInt64(metadata map[string]interface{}, key string) int64 {
	if metadata == nil {
		return 0
	}
	value, exists := metadata[key]
	if !exists || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		if v > uint64((1<<63)-1) {
			return 0
		}
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0
		}
		return n
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func isOpenClawSentenceSeparator(r rune, _ bool) bool {
	switch r {
	case '。', '？', '！', ';', '；', '.', '?', '!':
		return true
	default:
		return false
	}
}

func extractOpenClawSentences(text string, minLen int, isFirst bool) ([]string, string) {
	trimmed := normalizeOpenClawSpeechText(text)
	if trimmed == "" {
		return nil, ""
	}
	runes := []rune(trimmed)
	start := 0
	sentences := make([]string, 0, 4)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if !isOpenClawSentenceSeparator(r, isFirst) {
			continue
		}

		segment := trimOpenClawSegment(string(runes[start : i+1]))
		if segment == "" {
			start = skipOpenClawDelimiters(runes, i+1)
			continue
		}
		if len([]rune(segment)) < minLen {
			continue
		}
		sentences = append(sentences, segment)
		start = skipOpenClawDelimiters(runes, i+1)
	}

	remaining := trimOpenClawSegment(string(runes[start:]))
	return sentences, remaining
}

func trimOpenClawCanonicalPrefix(text string, prefix string) (string, bool) {
	normalizedText := normalizeOpenClawSpeechText(text)
	normalizedPrefix := normalizeOpenClawSpeechText(prefix)
	if normalizedPrefix == "" {
		return strings.TrimSpace(normalizedText), true
	}

	textKey := openClawComparableKey(normalizedText)
	prefixKey := openClawComparableKey(normalizedPrefix)
	if prefixKey == "" {
		return strings.TrimSpace(normalizedText), true
	}
	if !strings.HasPrefix(textKey, prefixKey) {
		return "", false
	}
	if textKey == prefixKey {
		return "", true
	}

	textRunes := []rune(normalizedText)
	prefixRunes := []rune(normalizedPrefix)
	matched := 0
	advancePrefix := func() {
		for matched < len(prefixRunes) && isOpenClawComparableIgnorableRune(prefixRunes[matched]) {
			matched++
		}
	}
	advancePrefix()
	for idx, r := range textRunes {
		if isOpenClawComparableIgnorableRune(r) {
			continue
		}
		if matched >= len(prefixRunes) || r != prefixRunes[matched] {
			return "", false
		}
		matched++
		advancePrefix()
		if matched == len(prefixRunes) {
			suffixStart := idx + 1
			for suffixStart < len(textRunes) && isOpenClawComparableIgnorableRune(textRunes[suffixStart]) {
				suffixStart++
			}
			return strings.TrimSpace(string(textRunes[suffixStart:])), true
		}
	}
	return "", false
}

func isOpenClawCanonicalGrowth(base string, candidate string) bool {
	baseKey := openClawComparableKey(base)
	candidateKey := openClawComparableKey(candidate)
	if baseKey == "" || candidateKey == "" {
		return false
	}
	return strings.HasPrefix(candidateKey, baseKey) || strings.HasPrefix(baseKey, candidateKey)
}

func isOpenClawSnapshotFrame(phase string, contentType string) bool {
	phase = strings.TrimSpace(strings.ToLower(phase))
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	return phase == "snapshot" || contentType == "snapshot"
}

func openClawCanonicalKey(text string) string {
	normalized := normalizeOpenClawSpeechText(text)
	if normalized == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, r := range normalized {
		if unicode.IsSpace(r) {
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func openClawComparableKey(text string) string {
	normalized := normalizeOpenClawSpeechText(text)
	if normalized == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, r := range normalized {
		if isOpenClawComparableIgnorableRune(r) {
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func isOpenClawComparableIgnorableRune(r rune) bool {
	return unicode.IsSpace(r) || isOpenClawPauseRune(r)
}

func isOpenClawPunctuationOnly(text string) bool {
	normalized := normalizeOpenClawSpeechText(text)
	if normalized == "" {
		return false
	}
	for _, r := range normalized {
		if unicode.IsSpace(r) {
			continue
		}
		if !isOpenClawPauseRune(r) {
			return false
		}
	}
	return true
}

func isOpenClawSoftSeparator(r rune) bool {
	switch r {
	case '，', ',', '、':
		return true
	default:
		return false
	}
}

func normalizeOpenClawSpeechText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"\r", "",
		"\t", " ",
		"```", "",
		"`", "",
		"**", "",
		"__", "",
		"###", "",
		"##", "",
		"#", "",
		"\n- ", "，",
		"\n* ", "，",
		"\n• ", "，",
		"\n", "，",
		"|", "，",
	)
	text = replacer.Replace(text)

	out := make([]rune, 0, len(text))
	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			if len(out) == 0 || out[len(out)-1] == ' ' || isOpenClawPauseRune(out[len(out)-1]) {
				continue
			}
			out = append(out, ' ')
		case r == '*' || r == '_' || r == '`' || r == '#':
			continue
		case isOpenClawSoftSeparator(r):
			trimOpenClawTrailingSpace(&out)
			if len(out) == 0 || isOpenClawPauseRune(out[len(out)-1]) {
				continue
			}
			out = append(out, '，')
		case isOpenClawSentenceSeparator(r, false):
			trimOpenClawTrailingSpace(&out)
			if len(out) == 0 {
				continue
			}
			out = append(out, r)
		case r == '：' || r == ':':
			trimOpenClawTrailingSpace(&out)
			if len(out) == 0 {
				continue
			}
			out = append(out, '：')
		case r == '-' || r == '•':
			if len(out) == 0 || isOpenClawPauseRune(out[len(out)-1]) {
				continue
			}
			out = append(out, r)
		default:
			out = append(out, r)
		}
	}

	return trimOpenClawSegment(string(out))
}

func trimOpenClawTrailingSpace(out *[]rune) {
	for len(*out) > 0 && (*out)[len(*out)-1] == ' ' {
		*out = (*out)[:len(*out)-1]
	}
}

func isOpenClawPauseRune(r rune) bool {
	switch r {
	case ' ', '，', ',', '、', '。', '！', '？', '!', '?', '；', ';', '：', ':':
		return true
	default:
		return false
	}
}

func skipOpenClawDelimiters(runes []rune, start int) int {
	for start < len(runes) {
		r := runes[start]
		if unicode.IsSpace(r) || isOpenClawSoftSeparator(r) {
			start++
			continue
		}
		break
	}
	return start
}

func trimOpenClawSegment(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimLeft(text, "-•*，,、;；:： ")
	replacer := strings.NewReplacer(
		" ，", "，",
		" 。", "。",
		" ！", "！",
		" ？", "？",
		" ；", "；",
		" ：", "：",
		"( ", "(",
		"（ ", "（",
		" )", ")",
		" ）", "）",
	)
	text = replacer.Replace(text)
	return strings.TrimSpace(text)
}

func (m *Manager) AddOfflineMessage(deviceID string, text string, correlationID string, isEnd bool) {
	deviceID = strings.TrimSpace(deviceID)
	text = strings.TrimSpace(text)
	correlationID = strings.TrimSpace(correlationID)
	if deviceID == "" {
		return
	}
	if text == "" && !isEnd {
		return
	}

	m.offlineMu.Lock()
	defer m.offlineMu.Unlock()

	m.pruneOfflineLocked(deviceID)
	msgList := m.offline[deviceID]
	if text == "" && isEnd {
		//The end frame allows empty content: the last one marked with the same correlation is the end first; if it does not exist, an empty end mark is written.
		for i := len(msgList) - 1; i >= 0; i-- {
			if correlationID == "" || strings.TrimSpace(msgList[i].CorrelationID) == correlationID {
				msgList[i].IsEnd = true
				m.offline[deviceID] = msgList
				logger.Infof("OpenClaw offline message marked end: device=%s correlation_id=%s total=%d", deviceID, correlationID, len(msgList))
				return
			}
		}
	}

	msgList = append(msgList, OfflineMessage{
		Text:          text,
		CorrelationID: correlationID,
		IsEnd:         isEnd,
		CreatedAt:     time.Now(),
	})
	if len(msgList) > MaxOfflineMessagesPerDevice {
		msgList = msgList[len(msgList)-MaxOfflineMessagesPerDevice:]
	}
	m.offline[deviceID] = msgList
	logger.Infof("OpenClaw offline message appended: device=%s correlation_id=%s end=%v total=%d", deviceID, correlationID, isEnd, len(msgList))
}

func (m *Manager) ReplayOfflineMessages(deviceID string, deliver func(msg OfflineMessage) error) (int, int) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || deliver == nil {
		return 0, 0
	}

	m.offlineMu.Lock()
	m.pruneOfflineLocked(deviceID)
	snapshot := append([]OfflineMessage(nil), m.offline[deviceID]...)
	m.offlineMu.Unlock()

	delivered := 0
	for _, msg := range snapshot {
		if err := deliver(msg); err != nil {
			break
		}
		delivered++
	}

	m.offlineMu.Lock()
	defer m.offlineMu.Unlock()

	m.pruneOfflineLocked(deviceID)
	current := m.offline[deviceID]
	if delivered > 0 {
		if delivered >= len(current) {
			delete(m.offline, deviceID)
			return delivered, 0
		}
		m.offline[deviceID] = current[delivered:]
		current = m.offline[deviceID]
	}
	return delivered, len(current)
}

func (m *Manager) pruneOfflineLocked(deviceID string) {
	msgList, exists := m.offline[deviceID]
	if !exists || len(msgList) == 0 {
		delete(m.offline, deviceID)
		return
	}

	now := time.Now()
	filtered := make([]OfflineMessage, 0, len(msgList))
	for _, msg := range msgList {
		if msg.CreatedAt.IsZero() {
			continue
		}
		if now.Sub(msg.CreatedAt) > OfflineMessageTTL {
			continue
		}
		filtered = append(filtered, msg)
	}

	if len(filtered) == 0 {
		delete(m.offline, deviceID)
		return
	}
	m.offline[deviceID] = filtered
}
