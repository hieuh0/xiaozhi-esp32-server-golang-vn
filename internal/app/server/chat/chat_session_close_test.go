package chat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	types_conn "xiaozhi-esp32-server-golang/internal/app/server/types"
	. "xiaozhi-esp32-server-golang/internal/data/client"
)

type sessionCloseTestConn struct {
	deviceID        string
	transportType   string
	sentCmds        [][]byte
	closeAudioCalls int
}

func (c *sessionCloseTestConn) SendCmd(msg []byte) error {
	copyMsg := append([]byte(nil), msg...)
	c.sentCmds = append(c.sentCmds, copyMsg)
	return nil
}

func (c *sessionCloseTestConn) RecvCmd(ctx context.Context, timeout int) ([]byte, error) {
	return nil, nil
}

func (c *sessionCloseTestConn) SendAudio(audio []byte) error {
	return nil
}

func (c *sessionCloseTestConn) RecvAudio(ctx context.Context, timeout int) ([]byte, error) {
	return nil, nil
}

func (c *sessionCloseTestConn) GetDeviceID() string {
	return c.deviceID
}

func (c *sessionCloseTestConn) Close() error {
	return nil
}

func (c *sessionCloseTestConn) OnClose(func(deviceId string)) {}

func (c *sessionCloseTestConn) CloseAudioChannel() error {
	c.closeAudioCalls++
	return nil
}

func (c *sessionCloseTestConn) GetTransportType() string {
	return c.transportType
}

func (c *sessionCloseTestConn) GetData(key string) (interface{}, error) {
	return nil, nil
}

func assertSingleGoodbyeCommand(t *testing.T, fakeConn *sessionCloseTestConn, sessionID string) {
	t.Helper()

	if len(fakeConn.sentCmds) != 1 {
		t.Fatalf("expected one mqtt goodbye command, got %d", len(fakeConn.sentCmds))
	}

	var msg map[string]any
	if err := json.Unmarshal(fakeConn.sentCmds[0], &msg); err != nil {
		t.Fatalf("failed to unmarshal goodbye command: %v", err)
	}
	if got := msg["type"]; got != "goodbye" {
		t.Fatalf("expected goodbye type, got %v", got)
	}
	if got := msg["state"]; got != "stop" {
		t.Fatalf("expected goodbye state stop, got %v", got)
	}
	if got := msg["session_id"]; got != sessionID {
		t.Fatalf("expected session_id %s, got %v", sessionID, got)
	}
}

func TestHandleSessionClosedSendsMqttGoodbyeOnExplicitExit(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	session := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         session,
		helloInited:     true,
	}

	manager.handleSessionClosed(session, chatSessionCloseReasonExplicitExit)

	if manager.GetSession() != nil {
		t.Fatalf("expected session to be cleared after explicit exit")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("expected CloseAudioChannel to be skipped on explicit exit, got %d", fakeConn.closeAudioCalls)
	}
	assertSingleGoodbyeCommand(t, fakeConn, "session-1")
	if !manager.helloInited {
		t.Fatalf("expected helloInited to stay true after mqtt explicit exit")
	}
	if !manager.needFreshHello {
		t.Fatalf("expected mqtt explicit exit to require a fresh hello before the next session bootstrap")
	}
}

func TestHandleSessionClosedSendsMqttGoodbyeOnFatalError(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	session := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         session,
		helloInited:     true,
	}

	manager.handleSessionClosed(session, chatSessionCloseReasonFatalError)

	if manager.GetSession() != nil {
		t.Fatalf("expected session to be cleared after fatal_error")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("expected CloseAudioChannel to be skipped on fatal_error, got %d", fakeConn.closeAudioCalls)
	}
	assertSingleGoodbyeCommand(t, fakeConn, "session-1")
	if !manager.helloInited {
		t.Fatalf("expected helloInited to stay true after fatal_error")
	}
	if !manager.needFreshHello {
		t.Fatalf("expected fatal_error to require a fresh hello before the next session bootstrap")
	}
}

func TestHandleSessionClosedSkipsMqttGoodbyeOnRetainedIdleTimeout(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	session := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         session,
		helloInited:     true,
	}

	manager.handleSessionClosed(session, chatSessionCloseReasonRetainedIdleTimeout)

	if manager.GetSession() != nil {
		t.Fatalf("expected session to be cleared after retained_idle_timeout")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("expected CloseAudioChannel to be skipped on retained_idle_timeout, got %d", fakeConn.closeAudioCalls)
	}
	if len(fakeConn.sentCmds) != 0 {
		t.Fatalf("expected retained_idle_timeout to avoid sending mqtt goodbye, got %d commands", len(fakeConn.sentCmds))
	}
	if !manager.helloInited {
		t.Fatalf("expected helloInited to stay true after retained_idle_timeout")
	}
	if !manager.needFreshHello {
		t.Fatalf("expected retained_idle_timeout to require a fresh hello before the next session bootstrap")
	}
}

func TestHandleSessionClosedSendsMqttGoodbyeOnAudioIdleTimeout(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	session := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         session,
		helloInited:     true,
	}

	manager.handleSessionClosed(session, chatSessionCloseReasonAudioIdleTimeout)

	if manager.GetSession() != nil {
		t.Fatalf("expected session to be cleared after audio_idle_timeout")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("expected CloseAudioChannel to be skipped on audio_idle_timeout, got %d", fakeConn.closeAudioCalls)
	}
	assertSingleGoodbyeCommand(t, fakeConn, "session-1")
	if !manager.helloInited {
		t.Fatalf("expected helloInited to stay true after audio_idle_timeout")
	}
	if !manager.needFreshHello {
		t.Fatalf("expected audio_idle_timeout to require a fresh hello before the next session bootstrap")
	}
}

func TestHandleSessionClosedDoesNotRequireHelloAfterMqttExplicitExit(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	session := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         session,
		helloInited:     true,
	}

	manager.handleSessionClosed(session, chatSessionCloseReasonExplicitExit)

	if !manager.needFreshHello {
		t.Fatalf("expected explicit exit to require a fresh hello")
	}
}

func TestHandleSessionClosedIgnoresStaleSessionCallback(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	staleSession := &ChatSession{}
	currentSession := &ChatSession{}
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         currentSession,
		helloInited:     true,
	}

	manager.handleSessionClosed(staleSession, chatSessionCloseReasonExplicitExit)

	if manager.GetSession() != currentSession {
		t.Fatalf("expected current session to stay active after stale callback")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("expected stale callback to avoid closing audio, got %d", fakeConn.closeAudioCalls)
	}
	if len(fakeConn.sentCmds) != 0 {
		t.Fatalf("expected stale callback to avoid sending mqtt goodbye, got %d commands", len(fakeConn.sentCmds))
	}
}

func TestEnsureSessionRejectsClosingSession(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	closingSession := &ChatSession{}
	closingSession.closing.Store(true)
	manager := &ChatManager{
		transport:       fakeConn,
		serverTransport: NewServerTransport(fakeConn, clientState),
		session:         closingSession,
		helloInited:     true,
	}

	session, err := manager.ensureSession()
	if err == nil {
		t.Fatalf("expected ensureSession to reject a closing session")
	}
	if session != nil {
		t.Fatalf("expected no replacement session while current session is closing")
	}
	if manager.GetSession() != closingSession {
		t.Fatalf("expected closing session to remain registered until close callback completes")
	}
	if !strings.Contains(err.Error(), "closing") {
		t.Fatalf("expected ensureSession error to mention closing state, got %v", err)
	}
}

func TestEnsureSessionWaitsForStartingSession(t *testing.T) {
	manager := &ChatManager{
		helloInited:         true,
		startingSession:     &ChatSession{},
		startingSessionDone: make(chan struct{}),
	}
	expectedSession := &ChatSession{}

	type result struct {
		session *ChatSession
		err     error
	}
	resultCh := make(chan result, 1)
	go func() {
		session, err := manager.ensureSession()
		resultCh <- result{session: session, err: err}
	}()

	select {
	case <-resultCh:
		t.Fatalf("expected ensureSession to wait while session startup is in progress")
	case <-time.After(50 * time.Millisecond):
	}

	manager.sessionMu.Lock()
	waitCh := manager.startingSessionDone
	manager.startingSession = nil
	manager.startingSessionDone = nil
	manager.session = expectedSession
	manager.sessionMu.Unlock()
	close(waitCh)

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("expected waiting ensureSession to succeed, got %v", result.err)
		}
		if result.session != expectedSession {
			t.Fatalf("expected waiting ensureSession to reuse published session")
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for ensureSession to resume")
	}
}

func TestHandleSessionClosedClearsStartingSessionWithoutTransportCleanup(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{SessionID: "session-1"}
	startingSession := &ChatSession{}
	waitCh := make(chan struct{})
	manager := &ChatManager{
		transport:           fakeConn,
		serverTransport:     NewServerTransport(fakeConn, clientState),
		startingSession:     startingSession,
		startingSessionDone: waitCh,
		helloInited:         true,
	}

	manager.handleSessionClosed(startingSession, chatSessionCloseReasonFatalError)

	select {
	case <-waitCh:
	default:
		t.Fatalf("expected startingSession wait channel to be closed")
	}
	if manager.startingSession != nil {
		t.Fatalf("expected startingSession to be cleared after close callback")
	}
	if fakeConn.closeAudioCalls != 0 {
		t.Fatalf("expected startup close callback to avoid closing audio, got %d", fakeConn.closeAudioCalls)
	}
	if len(fakeConn.sentCmds) != 0 {
		t.Fatalf("expected startup close callback to avoid sending mqtt goodbye, got %d commands", len(fakeConn.sentCmds))
	}
}

func TestShutdownClosesStartingSession(t *testing.T) {
	fakeConn := &sessionCloseTestConn{
		deviceID:      "device-1",
		transportType: types_conn.TransportTypeMqttUdp,
	}
	clientState := &ClientState{
		DeviceID: "device-1",
		Ctx:      context.Background(),
	}
	serverTransport := NewServerTransport(fakeConn, clientState)
	waitCh := make(chan struct{})
	manager := &ChatManager{
		transport:           fakeConn,
		serverTransport:     serverTransport,
		startingSessionDone: waitCh,
	}
	startingSession := NewChatSession(clientState, serverTransport, nil, nil, WithChatSessionCloseHandler(manager.handleSessionClosed))
	manager.startingSession = startingSession

	if err := manager.shutdown(false); err != nil {
		t.Fatalf("expected shutdown without transport to succeed, got %v", err)
	}
	if manager.startingSession != nil {
		t.Fatalf("expected shutdown to clear startingSession")
	}
	if !startingSession.IsClosing() {
		t.Fatalf("expected shutdown to close startingSession")
	}

	select {
	case <-waitCh:
	default:
		t.Fatalf("expected shutdown to release startingSession waiters")
	}
}
