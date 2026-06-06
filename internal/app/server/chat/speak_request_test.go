package chat

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"xiaozhi-esp32-server-golang/internal/app/server/auth"
	"xiaozhi-esp32-server-golang/internal/app/server/mqtt_udp"
	types_conn "xiaozhi-esp32-server-golang/internal/app/server/types"
	data_audio "xiaozhi-esp32-server-golang/internal/data/audio"
	data_client "xiaozhi-esp32-server-golang/internal/data/client"
	msgdata "xiaozhi-esp32-server-golang/internal/data/msg"
	"xiaozhi-esp32-server-golang/internal/util"
)

func TestShouldSendSpeakRequestSkipsActiveConversationSignals(t *testing.T) {
	now := time.Now()

	t.Run("listen phase", func(t *testing.T) {
		manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
		manager.helloInited = true
		manager.clientState.SetListenPhase(data_client.ListenPhaseListening)
		if manager.shouldSendSpeakRequest(now) {
			t.Fatal("expected active listen phase to skip speak_request")
		}
	})

	t.Run("client status", func(t *testing.T) {
		manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
		manager.helloInited = true
		manager.clientState.SetStatus(data_client.ClientStatusLLMStart)
		if manager.shouldSendSpeakRequest(now) {
			t.Fatal("expected llmStart status to skip speak_request")
		}
	})

	t.Run("tts active", func(t *testing.T) {
		manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
		manager.helloInited = true
		manager.session = &ChatSession{ttsManager: &TTSManager{}}
		manager.session.ttsManager.ttsActive.Store(true)
		if manager.shouldSendSpeakRequest(now) {
			t.Fatal("expected active TTS turn to skip speak_request")
		}
	})
}

func TestShouldSendSpeakRequestSkipsWarmPathWithinReuseWindow(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true
	manager.lastSpeakPathWarmAt.Store(time.Now().Add(-defaultSpeakRequestReuseWindow + time.Second).UnixMilli())

	if manager.shouldSendSpeakRequest(time.Now()) {
		t.Fatal("expected warm speak path within reuse window to skip speak_request")
	}
}

func TestShouldSendSpeakRequestUsesUDPBindingLastActive(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true
	conn.udpSession = &mqtt_udp.UdpSession{
		LastActive: time.Now().Add(-defaultSpeakRequestReuseWindow + time.Second),
	}

	if !manager.shouldSendSpeakRequest(time.Now()) {
		t.Fatal("expected missing UDP binding to require speak_request")
	}

	conn.udpSession.SetRemoteAddr(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9527})
	if manager.shouldSendSpeakRequest(time.Now()) {
		t.Fatal("expected active UDP binding within reuse window to skip speak_request")
	}
}

func TestShouldSendSpeakRequestRequiresWarmPathEvenWhenSessionExists(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.session = &ChatSession{}

	if !manager.shouldSendSpeakRequest(time.Now()) {
		t.Fatal("expected existing ChatSession without warm path to require speak_request")
	}
}

func TestShouldSendSpeakRequestWhenFreshHelloRequiredIgnoresWarmPath(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true
	manager.needFreshHello = true
	manager.lastSpeakPathWarmAt.Store(time.Now().UnixMilli())
	manager.clientState.SetListenPhase(data_client.ListenPhaseListening)
	manager.clientState.SetStatus(data_client.ClientStatusListening)

	if !manager.shouldSendSpeakRequest(time.Now()) {
		t.Fatal("expected fresh hello requirement to force speak_request")
	}
}

func TestShouldSendSpeakRequestWhenMqttRebootstrapPendingIgnoresActiveConversation(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true
	manager.mqttRebootstrapPending.Store(true)
	manager.lastSpeakPathWarmAt.Store(time.Now().UnixMilli())
	manager.clientState.SetListenPhase(data_client.ListenPhaseListening)
	manager.clientState.SetStatus(data_client.ClientStatusTTSStart)

	if !manager.shouldSendSpeakRequest(time.Now()) {
		t.Fatal("expected mqtt rebootstrap pending to force speak_request even when old conversation state looks active")
	}
}

func TestHandleHelloRejectsMissingAudioParamsBeforeSession(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)

	err := manager.HandleHelloMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeHello,
		DeviceID:  manager.DeviceID,
		Transport: types_conn.TransportTypeMqttUdp,
	})

	if err == nil || !strings.Contains(err.Error(), "audio_params") {
		t.Fatalf("expected missing audio_params error, got %v", err)
	}
	if manager.helloInited {
		t.Fatal("expected invalid hello not to initialize hello state")
	}
	if manager.GetSession() != nil {
		t.Fatal("expected invalid hello not to create ChatSession")
	}
	if conn.sentCmdCount() != 0 {
		t.Fatalf("expected invalid hello not to send protocol commands, got %d", conn.sentCmdCount())
	}
}

func TestHandleHelloRejectsUnsupportedTransportBeforeSession(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)

	err := manager.HandleHelloMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeHello,
		DeviceID:  manager.DeviceID,
		Transport: "invalid_transport",
		AudioParams: &data_audio.AudioFormat{
			SampleRate:    data_audio.SampleRate,
			Channels:      data_audio.Channels,
			FrameDuration: data_audio.FrameDuration,
			Format:        data_audio.Format,
		},
	})

	if err == nil || !strings.Contains(err.Error(), "Unsupported transfer type") {
		t.Fatalf("expected unsupported transport error, got %v", err)
	}
	if manager.helloInited {
		t.Fatal("expected invalid transport not to initialize hello state")
	}
	if manager.GetSession() != nil {
		t.Fatal("expected invalid transport not to create ChatSession")
	}
	if conn.sentCmdCount() != 0 {
		t.Fatalf("expected invalid transport not to send protocol commands, got %d", conn.sentCmdCount())
	}
}

func TestServerTransportSendMqttHelloIncludesUDPConfig(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	udpConfig := &msgdata.UdpConfig{
		Server: "127.0.0.1",
		Port:   12345,
		Key:    "test-key",
		Nonce:  "test-nonce",
	}

	if err := manager.serverTransport.SendHello(types_conn.TransportTypeMqttUdp, &manager.clientState.OutputAudioFormat, udpConfig); err != nil {
		t.Fatalf("SendHello returned error: %v", err)
	}

	serverMsg := waitForServerMessageType(t, conn, msgdata.ServerMessageTypeHello, time.Second)
	if serverMsg.Transport != types_conn.TransportTypeMqttUdp {
		t.Fatalf("expected mqtt udp transport, got %s", serverMsg.Transport)
	}
	if serverMsg.AudioFormat == nil {
		t.Fatal("expected hello response to include audio_params")
	}
	if serverMsg.Udp == nil {
		t.Fatal("expected hello response to include udp config")
	}
	if *serverMsg.Udp != *udpConfig {
		t.Fatalf("expected udp config %+v, got %+v", udpConfig, serverMsg.Udp)
	}
	if serverMsg.SessionID != manager.clientState.SessionID {
		t.Fatalf("expected session_id %q, got %q", manager.clientState.SessionID, serverMsg.SessionID)
	}
}

func TestPrepareSpeakPathForInjectedSpeechSendsSpeakRequestAndWaitsForReady(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true

	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.prepareSpeakPathForInjectedSpeech("主动播报内容", false)
	}()

	serverMsg := waitForServerMessage(t, conn, 0)
	if serverMsg.Type != msgdata.ServerMessageTypeSpeakRequest {
		t.Fatalf("expected speak_request, got %s", serverMsg.Type)
	}
	if serverMsg.Text != "主动播报内容" {
		t.Fatalf("expected speak_request text to be forwarded, got %q", serverMsg.Text)
	}
	if serverMsg.SessionID != manager.clientState.SessionID {
		t.Fatalf("expected speak_request session_id %q, got %q", manager.clientState.SessionID, serverMsg.SessionID)
	}
	if serverMsg.AutoListen == nil || *serverMsg.AutoListen {
		t.Fatal("expected speak_request auto_listen=false")
	}

	if err := manager.HandleSpeakReadyMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeSpeakReady,
		SessionID: manager.clientState.SessionID,
		State:     msgdata.MessageStateReady,
		SpeakUDPConfig: &data_client.SpeakReadyUDPConfig{
			Ready: true,
		},
	}); err != nil {
		t.Fatalf("HandleSpeakReadyMessage returned error: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("prepareSpeakPathForInjectedSpeech returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("prepareSpeakPathForInjectedSpeech did not unblock after speak_ready")
	}

	if manager.lastSpeakPathWarmAt.Load() == 0 {
		t.Fatal("expected speak_ready to mark speak path warm")
	}
	if manager.pendingSpeakRequest != nil {
		t.Fatal("expected pending speak_request to be cleared after speak_ready")
	}
}

func TestPrepareSpeakPathForInjectedSpeechForwardsAutoListenFlag(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true

	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.prepareSpeakPathForInjectedSpeech("主动播报自动续听", true)
	}()

	serverMsg := waitForServerMessage(t, conn, 0)
	if serverMsg.Type != msgdata.ServerMessageTypeSpeakRequest {
		t.Fatalf("expected speak_request, got %s", serverMsg.Type)
	}
	if serverMsg.AutoListen == nil || !*serverMsg.AutoListen {
		t.Fatal("expected speak_request auto_listen=true")
	}

	if err := manager.HandleSpeakReadyMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeSpeakReady,
		SessionID: manager.clientState.SessionID,
		State:     msgdata.MessageStateReady,
		SpeakUDPConfig: &data_client.SpeakReadyUDPConfig{
			Ready: true,
		},
	}); err != nil {
		t.Fatalf("HandleSpeakReadyMessage returned error: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("prepareSpeakPathForInjectedSpeech returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("prepareSpeakPathForInjectedSpeech did not unblock after speak_ready")
	}
}

func TestPrepareSpeakPathForInjectedSpeechReusesPendingSpeakRequest(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true

	errCh1 := make(chan error, 1)
	errCh2 := make(chan error, 1)
	go func() {
		errCh1 <- manager.prepareSpeakPathForInjectedSpeech("第一次播报", false)
	}()

	_ = waitForServerMessage(t, conn, 0)

	go func() {
		errCh2 <- manager.prepareSpeakPathForInjectedSpeech("第二次播报", false)
	}()

	time.Sleep(30 * time.Millisecond)
	if count := conn.sentCmdCount(); count != 1 {
		t.Fatalf("expected only one pending speak_request, got %d commands", count)
	}

	if err := manager.HandleSpeakReadyMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeSpeakReady,
		SessionID: manager.clientState.SessionID,
		State:     msgdata.MessageStateReady,
		SpeakUDPConfig: &data_client.SpeakReadyUDPConfig{
			Ready: true,
		},
	}); err != nil {
		t.Fatalf("HandleSpeakReadyMessage returned error: %v", err)
	}

	for idx, errCh := range []chan error{errCh1, errCh2} {
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("waiter %d returned error: %v", idx+1, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("waiter %d did not unblock after shared speak_ready", idx+1)
		}
	}
}

func TestPrepareSpeakPathForInjectedSpeechAllocatesSessionIDWhenMissing(t *testing.T) {
	if err := auth.Init(); err != nil {
		t.Fatalf("auth.Init returned error: %v", err)
	}

	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.clientState.SessionID = ""

	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.prepareSpeakPathForInjectedSpeech("自动分配会话", false)
	}()

	serverMsg := waitForServerMessage(t, conn, 0)
	if strings.TrimSpace(serverMsg.SessionID) == "" {
		t.Fatal("expected speak_request to carry a non-empty session_id")
	}
	if manager.clientState.SessionID != serverMsg.SessionID {
		t.Fatalf("expected clientState session_id %q, got %q", manager.clientState.SessionID, serverMsg.SessionID)
	}
	if _, err := auth.A().GetSession(serverMsg.SessionID); err != nil {
		t.Fatalf("expected allocated session to be registered in auth manager, got %v", err)
	}

	manager.sessionMu.Lock()
	manager.session = &ChatSession{}
	manager.sessionMu.Unlock()

	if err := manager.HandleSpeakReadyMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeSpeakReady,
		SessionID: serverMsg.SessionID,
		State:     msgdata.MessageStateReady,
		SpeakUDPConfig: &data_client.SpeakReadyUDPConfig{
			Ready: true,
		},
	}); err != nil {
		t.Fatalf("HandleSpeakReadyMessage returned error: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("prepareSpeakPathForInjectedSpeech returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("prepareSpeakPathForInjectedSpeech did not return after session allocation and speak_ready")
	}
}

func TestPrepareSpeakPathForInjectedSpeechTimesOut(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.speakReadyTimeout = 30 * time.Millisecond

	err := manager.prepareSpeakPathForInjectedSpeech("超时播报", false)
	if err == nil {
		t.Fatal("expected prepareSpeakPathForInjectedSpeech to time out")
	}
	if !errors.Is(err, context.DeadlineExceeded) && err.Error() != "Waiting for speak_ready timeout" {
		t.Fatalf("expected speak_ready timeout error, got %v", err)
	}
	if conn.sentCmdCount() != 1 {
		t.Fatalf("expected one speak_request before timeout, got %d", conn.sentCmdCount())
	}
	if manager.pendingSpeakRequest != nil {
		t.Fatal("expected timeout to clear pending speak_request")
	}
}

func TestInjectMessageWithSkipLlmFalseStillSendsSpeakRequest(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.session = &ChatSession{
		clientState:   manager.clientState,
		chatTextQueue: util.NewQueue[AsrResponseChannelItem](1),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.InjectMessage("需要先过LLM", false, false)
	}()

	serverMsg := waitForServerMessage(t, conn, 0)
	if serverMsg.Type != msgdata.ServerMessageTypeSpeakRequest {
		t.Fatalf("expected speak_request, got %s", serverMsg.Type)
	}
	if serverMsg.AutoListen == nil || *serverMsg.AutoListen {
		t.Fatal("expected InjectMessage auto_listen=false to be forwarded")
	}

	if err := manager.HandleSpeakReadyMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeSpeakReady,
		SessionID: manager.clientState.SessionID,
		State:     msgdata.MessageStateReady,
		SpeakUDPConfig: &data_client.SpeakReadyUDPConfig{
			Ready: true,
		},
	}); err != nil {
		t.Fatalf("HandleSpeakReadyMessage returned error: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("InjectMessage returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("InjectMessage did not return after speak_ready")
	}

	item, err := manager.session.chatTextQueue.Pop(context.Background(), time.Duration(-1))
	if err != nil {
		t.Fatalf("expected injected message to enter chat queue, got %v", err)
	}
	if item.text != "需要先过LLM" {
		t.Fatalf("expected injected text to be queued, got %q", item.text)
	}
}

func TestInjectMessageWaitsForSessionBootstrapAfterSpeakReady(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true
	manager.needFreshHello = true

	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.InjectMessage("等待会话建立", false, false)
	}()

	serverMsg := waitForServerMessage(t, conn, 0)
	if serverMsg.Type != msgdata.ServerMessageTypeSpeakRequest {
		t.Fatalf("expected speak_request, got %s", serverMsg.Type)
	}

	if err := manager.HandleSpeakReadyMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeSpeakReady,
		SessionID: manager.clientState.SessionID,
		State:     msgdata.MessageStateReady,
		SpeakUDPConfig: &data_client.SpeakReadyUDPConfig{
			Ready: true,
		},
	}); err != nil {
		t.Fatalf("HandleSpeakReadyMessage returned error: %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("expected InjectMessage to keep waiting for ChatSession, got %v", err)
	case <-time.After(80 * time.Millisecond):
	}

	session := &ChatSession{
		clientState:   manager.clientState,
		chatTextQueue: util.NewQueue[AsrResponseChannelItem](1),
	}
	manager.sessionMu.Lock()
	manager.needFreshHello = false
	manager.session = session
	manager.sessionMu.Unlock()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("InjectMessage returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("InjectMessage did not return after ChatSession bootstrap")
	}

	item, err := session.chatTextQueue.Pop(context.Background(), time.Duration(-1))
	if err != nil {
		t.Fatalf("expected injected message to enter chat queue, got %v", err)
	}
	if item.text != "等待会话建立" {
		t.Fatalf("expected injected text to be queued, got %q", item.text)
	}
}

func TestInjectMessageTimesOutWhenSessionBootstrapDoesNotFinish(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true
	manager.needFreshHello = true
	manager.speakReadyTimeout = 40 * time.Millisecond

	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.InjectMessage("会话超时", false, false)
	}()

	serverMsg := waitForServerMessage(t, conn, 0)
	if serverMsg.Type != msgdata.ServerMessageTypeSpeakRequest {
		t.Fatalf("expected speak_request, got %s", serverMsg.Type)
	}

	if err := manager.HandleSpeakReadyMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeSpeakReady,
		SessionID: manager.clientState.SessionID,
		State:     msgdata.MessageStateReady,
		SpeakUDPConfig: &data_client.SpeakReadyUDPConfig{
			Ready: true,
		},
	}); err != nil {
		t.Fatalf("HandleSpeakReadyMessage returned error: %v", err)
	}

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected InjectMessage to time out while waiting for ChatSession")
		}
		if !strings.Contains(err.Error(), "Timeout waiting for ChatSession to be established") {
			t.Fatalf("expected ChatSession wait timeout error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("InjectMessage did not return after ChatSession wait timeout")
	}
}

func TestHandleIoTMessageRespondsSuccessWithPayload(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeWebsocket)

	if err := manager.HandleIoTMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeIot,
		DeviceID:  manager.DeviceID,
		SessionID: manager.clientState.SessionID,
		Text:      "turn_on_light",
	}); err != nil {
		t.Fatalf("HandleIoTMessage returned error: %v", err)
	}

	serverMsg := waitForServerMessageType(t, conn, msgdata.ServerMessageTypeIot, time.Second)
	if serverMsg.State != msgdata.MessageStateSuccess {
		t.Fatalf("expected iot success state, got %s", serverMsg.State)
	}
	if serverMsg.Text != "turn_on_light" {
		t.Fatalf("expected iot text to roundtrip, got %q", serverMsg.Text)
	}
	if serverMsg.SessionID != manager.clientState.SessionID {
		t.Fatalf("expected iot session_id %q, got %q", manager.clientState.SessionID, serverMsg.SessionID)
	}
}

func TestAddAsrResultToQueueWithOptionsCarriesPlaybackStartHook(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	session := &ChatSession{
		clientState:   manager.clientState,
		chatTextQueue: util.NewQueue[AsrResponseChannelItem](1),
	}

	startedCount := 0
	if err := session.AddAsrResultToQueueWithOptions("需要走LLM", nil, llmResponseChannelOptions{
		onTTSPlaybackStart: func() {
			startedCount++
		},
	}); err != nil {
		t.Fatalf("AddAsrResultToQueueWithOptions returned error: %v", err)
	}

	item, err := session.chatTextQueue.Pop(context.Background(), time.Duration(-1))
	if err != nil {
		t.Fatalf("expected queued asr item, got %v", err)
	}

	hook := ttsPlaybackStartHookFromContext(item.ctx)
	if hook == nil {
		t.Fatal("expected playback start hook to be carried in queued ctx")
	}
	if startedCount != 0 {
		t.Fatal("expected playback start hook to stay idle before TTS really starts")
	}

	hook()
	hook()
	if startedCount != 1 {
		t.Fatalf("expected playback start hook to fire once, got %d", startedCount)
	}
}

func TestAddAsrResultToQueueWithOptionsCarriesTurnEndPolicy(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	session := &ChatSession{
		clientState:   manager.clientState,
		chatTextQueue: util.NewQueue[AsrResponseChannelItem](1),
	}

	if err := session.AddAsrResultToQueueWithOptions("需要走LLM", nil, llmResponseChannelOptions{
		ttsTurnEndPolicy: ttsTurnEndPolicyGoodbyeAndIdle,
	}); err != nil {
		t.Fatalf("AddAsrResultToQueueWithOptions returned error: %v", err)
	}

	item, err := session.chatTextQueue.Pop(context.Background(), time.Duration(-1))
	if err != nil {
		t.Fatalf("expected queued asr item, got %v", err)
	}

	if got := ttsTurnEndPolicyFromContext(item.ctx); got != ttsTurnEndPolicyGoodbyeAndIdle {
		t.Fatalf("expected turn-end policy %d, got %d", ttsTurnEndPolicyGoodbyeAndIdle, got)
	}
}

func TestAddTextToTTSQueueWithOptionsKeepsPlaybackHookOutOfQueueStart(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	ttsManager := NewTTSManager(manager.clientState, NewServerTransport(conn, manager.clientState), nil)
	llmManager := NewLLMManager(manager.clientState, NewServerTransport(conn, manager.clientState), ttsManager, nil, nil)

	startedCount := 0
	if err := llmManager.AddTextToTTSQueueWithOptions("直接播报", llmResponseChannelOptions{
		onTTSPlaybackStart: func() {
			startedCount++
		},
	}); err != nil {
		t.Fatalf("AddTextToTTSQueueWithOptions returned error: %v", err)
	}

	item, err := llmManager.llmResponseQueue.Pop(context.Background(), time.Duration(-1))
	if err != nil {
		t.Fatalf("expected queued llm item, got %v", err)
	}

	if item.onStartFunc == nil {
		t.Fatal("expected queue onStartFunc to be present for tts_start enqueue")
	}
	item.onStartFunc()
	if startedCount != 0 {
		t.Fatal("expected playback start hook not to run from llm queue onStartFunc")
	}

	hook := ttsPlaybackStartHookFromContext(item.ctx)
	if hook == nil {
		t.Fatal("expected playback start hook to be carried in llm item ctx")
	}

	hook()
	hook()
	if startedCount != 1 {
		t.Fatalf("expected playback start hook to fire once, got %d", startedCount)
	}
}

func TestAddTextToTTSQueueWithOptionsCarriesTurnEndPolicy(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	ttsManager := NewTTSManager(manager.clientState, NewServerTransport(conn, manager.clientState), nil)
	llmManager := NewLLMManager(manager.clientState, NewServerTransport(conn, manager.clientState), ttsManager, nil, nil)

	if err := llmManager.AddTextToTTSQueueWithOptions("直接播报", llmResponseChannelOptions{
		ttsTurnEndPolicy: ttsTurnEndPolicyGoodbyeAndIdle,
	}); err != nil {
		t.Fatalf("AddTextToTTSQueueWithOptions returned error: %v", err)
	}

	item, err := llmManager.llmResponseQueue.Pop(context.Background(), time.Duration(-1))
	if err != nil {
		t.Fatalf("expected queued llm item, got %v", err)
	}

	if got := ttsTurnEndPolicyFromContext(item.ctx); got != ttsTurnEndPolicyGoodbyeAndIdle {
		t.Fatalf("expected turn-end policy %d, got %d", ttsTurnEndPolicyGoodbyeAndIdle, got)
	}
}

func TestTTSManagerFinishTtsStopDispatchesTurnEndPolicy(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	session := NewChatSession(manager.clientState, manager.serverTransport, nil, nil, WithChatSessionCloseHandler(manager.handleSessionClosed))
	manager.session = session

	manager.lastSpeakPathWarmAt.Store(time.Now().UnixMilli())
	manager.clientState.Abort = true
	manager.clientState.IsWelcomeSpeaking = true
	manager.clientState.IsWelcomePlaying = true
	manager.clientState.SetStatus(data_client.ClientStatusTTSStart)
	manager.clientState.SetListenPhase(data_client.ListenPhaseListening)
	manager.clientState.SessionCtx.Get(manager.clientState.Ctx)
	manager.clientState.AfterAsrSessionCtx.Get(manager.clientState.Ctx)

	ttsManager := session.ttsManager
	ctx := withTTSTurnEndPolicy(manager.clientState.Ctx, ttsTurnEndPolicyGoodbyeAndIdle)
	ttsManager.EnqueueTtsStart(ctx)
	ttsManager.ttsActive.Store(true)
	if !ttsManager.finishTtsStop(withTTSTurnPlaybackSettled(manager.clientState.Ctx), false, nil) {
		t.Fatal("expected finishTtsStop to report an active TTS turn")
	}

	waitForServerMessageType(t, conn, msgdata.ServerMessageTypeGoodBye, time.Second)
	if manager.GetSession() != nil {
		t.Fatal("expected turn-end policy to close the current session through the existing service-side goodbye flow")
	}
	if !manager.needFreshHello {
		t.Fatal("expected turn-end policy goodbye to require a fresh hello for the next session bootstrap")
	}
	if manager.lastSpeakPathWarmAt.Load() != 0 {
		t.Fatal("expected turn-end policy to reset warm speak path timestamp")
	}
	if manager.retainedSessionCleanupTimer != nil {
		t.Fatal("expected turn-end policy goodbye not to retain the closed session")
	}
	manager.clientState.SetListenPhase(data_client.ListenPhaseListening)
	manager.clientState.SetStatus(data_client.ClientStatusListening)
	if !manager.shouldSendSpeakRequest(time.Now()) {
		t.Fatal("expected fresh-hello requirement after service-side goodbye to force the next speak_request")
	}
}

func TestTTSManagerFinishTtsStopWaitsPlaybackGraceBeforeTurnEndGoodbye(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	session := NewChatSession(manager.clientState, manager.serverTransport, nil, nil, WithChatSessionCloseHandler(manager.handleSessionClosed))
	manager.session = session

	ttsManager := session.ttsManager
	ctx := withTTSTurnEndPolicy(manager.clientState.Ctx, ttsTurnEndPolicyGoodbyeAndIdle)
	ttsManager.EnqueueTtsStart(ctx)
	ttsManager.ttsActive.Store(true)

	startedAt := time.Now()
	if !ttsManager.finishTtsStop(manager.clientState.Ctx, false, nil) {
		t.Fatal("expected finishTtsStop to report an active TTS turn")
	}

	time.Sleep(ttsPlaybackCompletionGrace - 40*time.Millisecond)
	if hasServerMessageType(conn, msgdata.ServerMessageTypeGoodBye) {
		t.Fatal("expected goodbye to wait until playback completion grace elapses")
	}

	waitForServerMessageType(t, conn, msgdata.ServerMessageTypeGoodBye, time.Second)
	if elapsed := time.Since(startedAt); elapsed < ttsPlaybackCompletionGrace {
		t.Fatalf("expected goodbye after at least %v, got %v", ttsPlaybackCompletionGrace, elapsed)
	}
}

func TestInjectedSpeechTTSTurnEndPolicy(t *testing.T) {
	if got := injectedSpeechTTSTurnEndPolicy(true); got != ttsTurnEndPolicyNone {
		t.Fatalf("expected autoListen=true to keep default turn-end policy, got %d", got)
	}
	if got := injectedSpeechTTSTurnEndPolicy(false); got != ttsTurnEndPolicyGoodbyeAndIdle {
		t.Fatalf("expected autoListen=false to switch to goodbye-and-idle policy, got %d", got)
	}
}

func TestHandleMqttTransportReadyResetsConversationState(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	session := NewChatSession(manager.clientState, manager.serverTransport, nil, nil, WithChatSessionCloseHandler(manager.handleSessionClosed))
	manager.session = session

	manager.lastSpeakPathWarmAt.Store(time.Now().UnixMilli())
	manager.clientState.Abort = true
	manager.clientState.IsWelcomeSpeaking = true
	manager.clientState.IsWelcomePlaying = true
	manager.clientState.SetStatus(data_client.ClientStatusTTSStart)
	manager.clientState.SetListenPhase(data_client.ListenPhaseListening)
	manager.clientState.SessionCtx.Get(manager.clientState.Ctx)
	manager.clientState.AfterAsrSessionCtx.Get(manager.clientState.Ctx)

	manager.HandleMqttTransportReady()

	if !manager.needsMqttRebootstrap() {
		t.Fatal("expected mqtt transport ready to mark conversation state stale")
	}
	if manager.GetSession() != session {
		t.Fatal("expected mqtt transport ready to retain current ChatSession")
	}
	if manager.lastSpeakPathWarmAt.Load() != 0 {
		t.Fatal("expected mqtt transport ready to clear warm speak path timestamp")
	}
	if manager.retainedSessionCleanupTimer == nil {
		t.Fatal("expected mqtt transport ready to schedule retained-session cleanup")
	}
	if manager.clientState.GetStatus() != data_client.ClientStatusInit {
		t.Fatalf("expected client status to reset to init, got %s", manager.clientState.GetStatus())
	}
	if manager.clientState.GetListenPhase() != data_client.ListenPhaseIdle {
		t.Fatalf("expected listen phase to reset to idle, got %s", manager.clientState.GetListenPhase())
	}
	if manager.clientState.Abort {
		t.Fatal("expected mqtt transport ready to clear abort flag")
	}
	if manager.clientState.IsWelcomeSpeaking || manager.clientState.IsWelcomePlaying {
		t.Fatal("expected mqtt transport ready to clear welcome flags")
	}
	if conn.sentCmdCount() != 0 {
		t.Fatalf("expected mqtt transport ready normalization to avoid protocol commands, got %d", conn.sentCmdCount())
	}
}

func TestHandleMqttTransportReadyResetsMcpRuntimeBeforeWarmup(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.mcpInitState = chatMcpInitStateReady

	oldCloseRuntime := closeDeviceMcpRuntime
	oldEnsureRuntime := ensureDeviceMcpRuntime
	oldShouldSchedule := shouldScheduleDeviceMcpRuntimeInit
	defer func() {
		closeDeviceMcpRuntime = oldCloseRuntime
		ensureDeviceMcpRuntime = oldEnsureRuntime
		shouldScheduleDeviceMcpRuntimeInit = oldShouldSchedule
	}()

	closeCalled := make(chan struct{}, 1)
	initCalled := make(chan struct{}, 1)
	closeDeviceMcpRuntime = func(deviceID string, mcpTransport *McpTransport) {
		if deviceID != manager.DeviceID {
			t.Fatalf("unexpected device id passed to closeDeviceMcpRuntime: %s", deviceID)
		}
		if mcpTransport != manager.mcpTransport {
			t.Fatal("expected closeDeviceMcpRuntime to receive current MCP transport")
		}
		closeCalled <- struct{}{}
	}
	shouldScheduleDeviceMcpRuntimeInit = func(deviceID string, mcpTransport *McpTransport) bool {
		return true
	}
	ensureDeviceMcpRuntime = func(deviceID string, mcpTransport *McpTransport) error {
		if deviceID != manager.DeviceID {
			t.Fatalf("unexpected device id passed to ensureDeviceMcpRuntime: %s", deviceID)
		}
		if mcpTransport != manager.mcpTransport {
			t.Fatal("expected ensureDeviceMcpRuntime to receive current MCP transport")
		}
		initCalled <- struct{}{}
		return nil
	}

	manager.HandleMqttTransportReady()

	select {
	case <-closeCalled:
	case <-time.After(time.Second):
		t.Fatal("expected mqtt transport ready to close existing IoT MCP runtime")
	}
	if manager.mcpInitState != chatMcpInitStateIdle {
		t.Fatalf("expected MCP init state to reset to idle, got %v", manager.mcpInitState)
	}

	manager.WarmupMcp()

	select {
	case <-initCalled:
	case <-time.After(time.Second):
		t.Fatal("expected WarmupMcp to reinitialize IoT MCP runtime after transport ready")
	}

	waitForMcpInitState(t, manager, chatMcpInitStateReady)
}

func TestMarkMqttConversationStateStaleSkipsDuplicateHelloWhenSpeakRequestPending(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.lastSpeakPathWarmAt.Store(123)
	pending, created := manager.getOrCreatePendingSpeakRequest()
	if !created || pending == nil {
		t.Fatal("expected pending speak_request to be created")
	}

	manager.markMqttConversationStateStale("duplicate_hello")

	if manager.needsMqttRebootstrap() {
		t.Fatal("expected duplicate hello during pending speak_request not to mark mqtt rebootstrap")
	}
	if manager.pendingSpeakRequest != pending {
		t.Fatal("expected pending speak_request to survive duplicate hello handshake")
	}
	if manager.lastSpeakPathWarmAt.Load() != 123 {
		t.Fatal("expected duplicate hello handshake not to reset warm speak path state")
	}
	select {
	case <-pending.done:
		t.Fatal("expected pending speak_request not to be resolved by duplicate hello handshake")
	default:
	}

	manager.finishPendingSpeakRequest(pending, nil)
}

func TestHandleSpeakReadyMessageClearsMqttRebootstrapPending(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.mqttRebootstrapPending.Store(true)
	pending, created := manager.getOrCreatePendingSpeakRequest()
	if !created || pending == nil {
		t.Fatal("expected pending speak_request to be created")
	}

	if err := manager.HandleSpeakReadyMessage(&data_client.ClientMessage{
		Type:      msgdata.MessageTypeSpeakReady,
		SessionID: manager.clientState.SessionID,
		State:     msgdata.MessageStateReady,
		SpeakUDPConfig: &data_client.SpeakReadyUDPConfig{
			Ready: true,
		},
	}); err != nil {
		t.Fatalf("HandleSpeakReadyMessage returned error: %v", err)
	}

	if manager.needsMqttRebootstrap() {
		t.Fatal("expected speak_ready to clear mqtt rebootstrap pending")
	}
}

func TestRefreshSpeakPathWarmFromTransportSkipsPendingSpeakRequest(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	conn.udpSession = &mqtt_udp.UdpSession{
		LastActive: time.Now(),
	}
	pending, created := manager.getOrCreatePendingSpeakRequest()
	if !created || pending == nil {
		t.Fatal("expected pending speak_request to be created")
	}

	manager.refreshSpeakPathWarmFromTransport()

	if manager.lastSpeakPathWarmAt.Load() != 0 {
		t.Fatal("expected pending speak_request handshake to defer warm path refresh until speak_ready")
	}

	manager.finishPendingSpeakRequest(pending, nil)
}

func TestHandleListenMessageIgnoredWhenFreshHelloRequired(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.helloInited = true
	manager.needFreshHello = true
	manager.clientState.SetStatus(data_client.ClientStatusInit)
	manager.clientState.SetListenPhase(data_client.ListenPhaseIdle)

	if err := manager.HandleListenMessage(&data_client.ClientMessage{
		Type:     msgdata.MessageTypeListen,
		DeviceID: manager.DeviceID,
		State:    msgdata.MessageStateStart,
		Mode:     "auto",
	}); err != nil {
		t.Fatalf("HandleListenMessage returned error: %v", err)
	}

	if manager.clientState.GetStatus() != data_client.ClientStatusInit {
		t.Fatalf("expected ignored listen message to keep status init, got %s", manager.clientState.GetStatus())
	}
	if manager.clientState.GetListenPhase() != data_client.ListenPhaseIdle {
		t.Fatalf("expected ignored listen message to keep phase idle, got %s", manager.clientState.GetListenPhase())
	}
	if !manager.needFreshHello {
		t.Fatal("expected ignored listen message to preserve the fresh-hello requirement")
	}
}

func TestHandleListenMessageIgnoredBeforeHello(t *testing.T) {
	manager, _ := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	manager.clientState.SetStatus(data_client.ClientStatusInit)
	manager.clientState.SetListenPhase(data_client.ListenPhaseIdle)

	if err := manager.HandleListenMessage(&data_client.ClientMessage{
		Type:     msgdata.MessageTypeListen,
		DeviceID: manager.DeviceID,
		State:    msgdata.MessageStateStart,
		Mode:     "manual",
	}); err != nil {
		t.Fatalf("HandleListenMessage returned error: %v", err)
	}

	if manager.GetSession() != nil {
		t.Fatal("expected listen before hello not to create ChatSession")
	}
	if manager.clientState.GetStatus() != data_client.ClientStatusInit {
		t.Fatalf("expected ignored listen before hello to keep status init, got %s", manager.clientState.GetStatus())
	}
	if manager.clientState.GetListenPhase() != data_client.ListenPhaseIdle {
		t.Fatalf("expected ignored listen before hello to keep phase idle, got %s", manager.clientState.GetListenPhase())
	}
}

func TestTTSManagerFinishTtsStopSkipsTurnEndPolicyWhenErr(t *testing.T) {
	manager, conn := newSpeakRequestTestManager(types_conn.TransportTypeMqttUdp)
	session := NewChatSession(manager.clientState, manager.serverTransport, nil, nil, WithChatSessionCloseHandler(manager.handleSessionClosed))
	manager.session = session

	ttsManager := session.ttsManager
	ctx := withTTSTurnEndPolicy(manager.clientState.Ctx, ttsTurnEndPolicyGoodbyeAndIdle)
	ttsManager.EnqueueTtsStart(ctx)
	ttsManager.ttsActive.Store(true)
	if !ttsManager.finishTtsStop(manager.clientState.Ctx, false, context.Canceled) {
		t.Fatal("expected finishTtsStop to report an active TTS turn")
	}

	time.Sleep(50 * time.Millisecond)
	if conn.sentCmdCount() != 0 {
		t.Fatalf("expected no goodbye when turn-end policy finishes with error, got %d commands", conn.sentCmdCount())
	}
}

type speakRequestTestConn struct {
	mu            sync.Mutex
	transportType string
	deviceID      string
	sentCmds      [][]byte
	sendCmdErr    error
	udpSession    *mqtt_udp.UdpSession
	onClose       func(deviceId string)
}

func newSpeakRequestTestManager(transportType string) (*ChatManager, *speakRequestTestConn) {
	conn := &speakRequestTestConn{
		transportType: transportType,
		deviceID:      "test-device",
	}
	manager := &ChatManager{
		DeviceID: conn.deviceID,
	}
	baseCtx := context.WithValue(context.Background(), "chat_session_operator", ChatSessionOperator(manager))
	baseCtx = withTTSTurnEndPolicyHandler(baseCtx, manager)
	ctx, cancel := context.WithCancel(baseCtx)
	clientState := &data_client.ClientState{
		Ctx:         ctx,
		Cancel:      cancel,
		Dialogue:    &data_client.Dialogue{},
		DeviceID:    conn.deviceID,
		SessionID:   "test-session",
		ListenPhase: data_client.ListenPhaseIdle,
		Status:      data_client.ClientStatusInit,
		OutputAudioFormat: data_audio.AudioFormat{
			SampleRate:    data_audio.SampleRate,
			Channels:      data_audio.Channels,
			FrameDuration: data_audio.FrameDuration,
			Format:        data_audio.Format,
		},
		OpusAudioBuffer: make(chan []byte, 8),
		AsrAudioBuffer: &data_client.AsrAudioBuffer{
			PcmData:          make([]float32, 0),
			AudioBufferMutex: sync.RWMutex{},
		},
		VoiceStatus: data_client.VoiceStatus{
			SilenceThresholdTime: 400,
		},
		SessionCtx: data_client.Ctx{},
	}
	manager.transport = conn
	manager.clientState = clientState
	manager.serverTransport = NewServerTransport(conn, clientState)
	manager.mcpTransport = &McpTransport{
		Client:          clientState,
		ServerTransport: manager.serverTransport,
	}
	manager.ctx = ctx
	manager.cancel = cancel
	manager.speakReadyTimeout = 200 * time.Millisecond
	return manager, conn
}

func (c *speakRequestTestConn) SendCmd(msg []byte) error {
	if c.sendCmdErr != nil {
		return c.sendCmdErr
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	msgCopy := make([]byte, len(msg))
	copy(msgCopy, msg)
	c.sentCmds = append(c.sentCmds, msgCopy)
	return nil
}

func (c *speakRequestTestConn) RecvCmd(ctx context.Context, timeout int) ([]byte, error) {
	return nil, context.DeadlineExceeded
}

func (c *speakRequestTestConn) SendAudio(audio []byte) error {
	return nil
}

func (c *speakRequestTestConn) RecvAudio(ctx context.Context, timeout int) ([]byte, error) {
	return nil, context.DeadlineExceeded
}

func (c *speakRequestTestConn) GetDeviceID() string {
	return c.deviceID
}

func (c *speakRequestTestConn) Close() error {
	if c.onClose != nil {
		c.onClose(c.deviceID)
	}
	return nil
}

func (c *speakRequestTestConn) OnClose(handler func(deviceId string)) {
	c.onClose = handler
}

func (c *speakRequestTestConn) CloseAudioChannel() error {
	return nil
}

func (c *speakRequestTestConn) GetTransportType() string {
	return c.transportType
}

func (c *speakRequestTestConn) GetData(key string) (interface{}, error) {
	return nil, nil
}

func (c *speakRequestTestConn) GetUdpSession() *mqtt_udp.UdpSession {
	return c.udpSession
}

func (c *speakRequestTestConn) sentCmdCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.sentCmds)
}

func waitForServerMessage(t *testing.T, conn *speakRequestTestConn, index int) msgdata.ServerMessage {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		conn.mu.Lock()
		if len(conn.sentCmds) > index {
			payload := make([]byte, len(conn.sentCmds[index]))
			copy(payload, conn.sentCmds[index])
			conn.mu.Unlock()

			var serverMsg msgdata.ServerMessage
			if err := json.Unmarshal(payload, &serverMsg); err != nil {
				t.Fatalf("failed to decode server message: %v", err)
			}
			return serverMsg
		}
		conn.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for server message %d", index)
	return msgdata.ServerMessage{}
}

func hasServerMessageType(conn *speakRequestTestConn, messageType string) bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	for _, raw := range conn.sentCmds {
		var serverMsg msgdata.ServerMessage
		if err := json.Unmarshal(raw, &serverMsg); err != nil {
			continue
		}
		if serverMsg.Type == messageType {
			return true
		}
	}
	return false
}

func waitForServerMessageType(t *testing.T, conn *speakRequestTestConn, messageType string, timeout time.Duration) msgdata.ServerMessage {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn.mu.Lock()
		payloads := make([][]byte, len(conn.sentCmds))
		for i, raw := range conn.sentCmds {
			payload := make([]byte, len(raw))
			copy(payload, raw)
			payloads[i] = payload
		}
		conn.mu.Unlock()

		for _, payload := range payloads {
			var serverMsg msgdata.ServerMessage
			if err := json.Unmarshal(payload, &serverMsg); err != nil {
				t.Fatalf("failed to decode server message: %v", err)
			}
			if serverMsg.Type == messageType {
				return serverMsg
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for server message type %s", messageType)
	return msgdata.ServerMessage{}
}
