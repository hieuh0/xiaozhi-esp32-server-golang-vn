package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "xiaozhi-esp32-server-golang/logger"
)

const stopSpeakingInterruptTimeout = 2 * time.Second

func (s *ChatSession) StopSpeaking(isSendTtsStop bool) {
	s.StopSpeakingWithReason(isSendTtsStop, "ChatSession.StopSpeaking")
}

func (s *ChatSession) StopSpeakingWithReason(isSendTtsStop bool, reason string) {
	s.stopSpeakingWithLock(true, isSendTtsStop, true, reason)
}

// StopAssistantOutputAfterAsrWithReason only stops the current assistant output and does not suspend media playback.
func (s *ChatSession) StopAssistantOutputAfterAsrWithReason(isSendTtsStop bool, reason string) {
	s.stopSpeakingWithLock(false, isSendTtsStop, false, reason)
}

// stopSpeakingWithLock adds mutex protection based on stopSpeaking
func (s *ChatSession) stopSpeakingWithLock(cancelSession bool, isSendTtsStop bool, suspendMedia bool, reason string) {
	s.stopSpeakingMu.Lock()
	defer s.stopSpeakingMu.Unlock()
	s.stopSpeaking(cancelSession, isSendTtsStop, suspendMedia, reason)
}

func (s *ChatSession) stopSpeaking(cancelSession bool, isSendTtsStop bool, suspendMedia bool, reason string) {
	reason = normalizeTTSReason(reason)
	state := "tts_manager=nil"
	if s != nil && s.ttsManager != nil {
		state = s.ttsManager.debugState()
	}
	log.Infof(
		"stop speaking requested: device=%s reason=%s cancelSession=%v sendTtsStop=%v suspendMedia=%v state={%s}",
		s.clientState.DeviceID,
		reason,
		cancelSession,
		isSendTtsStop,
		suspendMedia,
		state,
	)

	if cancelSession {
		s.clientState.SessionCtx.CancelWithReason(fmt.Sprintf("ChatSession.stopSpeaking(%s): session_ctx", reason))
		s.invalidateListenStart()
	}
	s.clientState.AfterAsrSessionCtx.CancelWithReason(fmt.Sprintf("ChatSession.stopSpeaking(%s): after_asr_ctx", reason))
	s.completeWelcomePlaybackWait(false)

	if suspendMedia && s.mediaPlayer != nil {
		if err := s.mediaPlayer.Suspend(); err != nil && !errors.Is(err, context.Canceled) {
			log.Warnf("stopSpeaking failed to suspend media playback: %v", err)
		}
	}

	s.clientState.IsWelcomePlaying = false

	s.ClearChatTextQueue()
	s.llmManager.ClearLLMResponseQueue()
	s.ttsManager.ClearTTSQueue()
	interruptCtx, cancel := context.WithTimeout(context.Background(), stopSpeakingInterruptTimeout)
	defer cancel()
	if err := s.ttsManager.InterruptAndStopSyncWithReason(interruptCtx, isSendTtsStop, context.Canceled, fmt.Sprintf("ChatSession.stopSpeaking(%s)", reason)); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Warnf("stopSpeaking sync interrupt timed out, cancel_session=%v", cancelSession)
		} else if !errors.Is(err, context.Canceled) {
			log.Warnf("stopSpeaking sync interrupt failed, cancel_session=%v, err=%v", cancelSession, err)
		}
	}
}

func (s *ChatSession) MqttClose() {
	s.serverTransport.SendMqttGoodbye()
}

func (s *ChatSession) ResetToSilentState() {
	if s == nil || s.IsClosing() {
		return
	}

	s.cancelPendingDetectLLM()
	s.finishOpenClawWarmup("", false)
	s.clearOpenClawStreams()
	s.clearPendingSpeakerResult()

	if s.clientState != nil {
		s.clientState.Abort = false
		s.clientState.IsWelcomeSpeaking = false
	}

	s.stopSpeakingWithLock(true, true, true, "ChatSession.ResetToSilentState")

	if s.clientState != nil {
		s.clientState.Destroy()
		s.clientState.Abort = false
		s.clientState.IsWelcomeSpeaking = false
		s.clientState.IsWelcomePlaying = false
	}
}
