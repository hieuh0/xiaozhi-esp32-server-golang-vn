package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/schema"

	"xiaozhi-esp32-server-golang/internal/domain/llm"
	llm_common "xiaozhi-esp32-server-golang/internal/domain/llm/common"
	"xiaozhi-esp32-server-golang/internal/pool"
	log "xiaozhi-esp32-server-golang/logger"
)

var openClawWarmupSchedule = []time.Duration{
	1 * time.Second,
	10 * time.Second,
	20 * time.Second,
	30 * time.Second,
	40 * time.Second,
	50 * time.Second,
	60 * time.Second,
	70 * time.Second,
	80 * time.Second,
	90 * time.Second,
	100 * time.Second,
}

const (
	openClawWarmupPlanTimeout = 8 * time.Second
	openClawWarmupPlanSize    = 11
)

const openClawWarmupSystemPrompt = `You are a warmup assistant in a real-time voice conversation, not the main responder.

Your task: Before the main response is returned, generate 11 very short English filler phrases to make the waiting period feel like someone is always responding.

Hard requirements:
1. Only responsible for warmup - do not directly answer questions, do not provide facts, conclusions, suggestions, steps, analysis, explanations or speculation.
2. Tone should be like a real person gently responding in a phone call: short, natural, conversational, patient.
3. Do not sound like customer service, system prompts, notifications, or marketing copy.
4. Do not repeat the user's original words, especially avoid echoing command phrases like “help me search”, “help me check”, “tell me”.
5. If you need to mention the topic, only distill it into a noun phrase from the assistant's perspective, e.g. “tomorrow's weather in Hanoi”, “this arrangement” - do not use imperative sentences.
6. The first 1-2 phrases should be lightest, not necessarily topic-specific, e.g. “Let me check” or “Just a moment” - do not start with heavy reassurance.
7. Later phrases should gradually express “still looking” and “still confirming” - but naturally, not mechanically repetitive.
8. Avoid stiff phrases like “processing your request”, “please wait”, “following up”, “retrieving data”, “connecting to service”.
9. Each phrase must be a single short English sentence suitable for voice broadcast, length 5-80 characters.
10. You will receive actual broadcast time points. The 11 phrases must be strictly designed in order for these time points:
   - 1st second: Just received the question, respond gently.
   - 10th second: Naturally add a phrase, still light in tone.
   - 20th, 30th seconds: Start expressing “still looking” but not mechanically.
   - 40th, 50th, 60th seconds: Continue reassuring, more explicitly saying “still confirming”.
   - 70th, 80th, 90th, 100th seconds: Acknowledge it's taking a while, but remain natural and calm, no complaining.
11. Output strictly a JSON array of length 11.
12. Each JSON item format must be: {“text”:”warmup phrase”}.
13. Do not output numbering, Markdown, explanations, code blocks, or anything other than JSON.`

type openClawWarmupTask struct {
	correlationID string
	sessionCtx    context.Context
	warmupCtx     context.Context
	cancelWarmup  context.CancelFunc

	linesMu sync.RWMutex
	lines   []string

	stateMu                  sync.Mutex
	speechStarted            bool
	speechEnded              bool
	nextWarmupSegmentIsStart bool
	planReadyAt              time.Time
	planReadySignaled        bool

	spokeAny    atomic.Bool
	planReadyCh chan struct{}
}

type openClawWarmupLine struct {
	Text string `json:"text"`
}

func (s *ChatSession) startOpenClawWarmup(correlationID string, userText string) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" || s == nil || s.clientState == nil {
		return
	}

	sessionCtx := s.clientState.SessionCtx.Get(s.clientState.Ctx)
	parentCtx := s.clientState.AfterAsrSessionCtx.Get(sessionCtx)
	warmupCtx, cancelWarmup := context.WithCancel(parentCtx)
	task := &openClawWarmupTask{
		correlationID:            correlationID,
		sessionCtx:               parentCtx,
		warmupCtx:                warmupCtx,
		cancelWarmup:             cancelWarmup,
		lines:                    make([]string, openClawWarmupPlanSize),
		nextWarmupSegmentIsStart: true,
		planReadyCh:              make(chan struct{}),
	}

	s.replaceOpenClawWarmup(task)
	log.Infof("OpenClaw warmup started: device=%s correlation_id=%s", s.clientState.DeviceID, correlationID)

	go s.runOpenClawWarmupTask(task, userText)
}

func (s *ChatSession) replaceOpenClawWarmup(task *openClawWarmupTask) {
	s.openClawWarmupMu.Lock()
	oldTask := s.openClawWarmup
	s.openClawWarmup = task
	s.openClawWarmupMu.Unlock()

	if oldTask != nil {
		oldTask.cancelWarmupOnly()
	}
}

func (task *openClawWarmupTask) cancelWarmupOnly() {
	if task == nil || task.cancelWarmup == nil {
		return
	}
	task.cancelWarmup()
}

func (task *openClawWarmupTask) markSpeechStarted() bool {
	if task == nil {
		return false
	}
	task.stateMu.Lock()
	defer task.stateMu.Unlock()
	if task.speechStarted || task.speechEnded {
		return false
	}
	task.speechStarted = true
	return true
}

func (task *openClawWarmupTask) markSpeechEnded() bool {
	if task == nil {
		return false
	}
	task.stateMu.Lock()
	defer task.stateMu.Unlock()
	if !task.speechStarted || task.speechEnded {
		return false
	}
	task.speechEnded = true
	return true
}

func (task *openClawWarmupTask) takeWarmupSegmentStartFlag() bool {
	if task == nil {
		return true
	}
	task.stateMu.Lock()
	defer task.stateMu.Unlock()
	isStart := task.nextWarmupSegmentIsStart
	task.nextWarmupSegmentIsStart = false
	return isStart
}

func (task *openClawWarmupTask) markPlanReady(readyAt time.Time) {
	if task == nil {
		return
	}
	task.stateMu.Lock()
	if task.planReadySignaled {
		task.stateMu.Unlock()
		return
	}
	task.planReadyAt = readyAt
	task.planReadySignaled = true
	close(task.planReadyCh)
	task.stateMu.Unlock()
}

func (task *openClawWarmupTask) waitPlanReady(ctx context.Context) (time.Time, bool) {
	if task == nil {
		return time.Time{}, false
	}

	select {
	case <-ctx.Done():
		return time.Time{}, false
	case <-task.planReadyCh:
	}

	task.stateMu.Lock()
	defer task.stateMu.Unlock()
	if task.planReadyAt.IsZero() {
		return time.Time{}, false
	}
	return task.planReadyAt, true
}

func (task *openClawWarmupTask) hasSpokenAny() bool {
	if task == nil {
		return false
	}
	return task.spokeAny.Load()
}

func (s *ChatSession) getOpenClawWarmupTask(correlationID string) *openClawWarmupTask {
	if s == nil {
		return nil
	}
	correlationID = strings.TrimSpace(correlationID)
	s.openClawWarmupMu.Lock()
	defer s.openClawWarmupMu.Unlock()
	task := s.openClawWarmup
	if task == nil {
		return nil
	}
	if correlationID != "" && task.correlationID != correlationID {
		return nil
	}
	return task
}

func (s *ChatSession) takeOpenClawWarmupTask(correlationID string) *openClawWarmupTask {
	if s == nil {
		return nil
	}
	correlationID = strings.TrimSpace(correlationID)
	s.openClawWarmupMu.Lock()
	defer s.openClawWarmupMu.Unlock()
	task := s.openClawWarmup
	if task == nil {
		return nil
	}
	if correlationID != "" && task.correlationID != correlationID {
		return nil
	}
	s.openClawWarmup = nil
	return task
}

func (s *ChatSession) cancelOpenClawWarmup(correlationID string, interrupt bool) bool {
	if s == nil {
		return false
	}

	task := s.getOpenClawWarmupTask(correlationID)
	if task == nil {
		return false
	}
	if task.warmupCtx.Err() != nil {
		return false
	}

	task.cancelWarmupOnly()
	if interrupt && task.hasSpokenAny() {
		s.InterruptAndClearTTSQueueWithReason(fmt.Sprintf("OpenClaw warmup canceled correlation_id=%s", correlationID))
	}

	log.Infof(
		"OpenClaw warmup canceled: device=%s correlation_id=%s interrupt=%v spoke_any=%v",
		s.clientState.DeviceID,
		task.correlationID,
		interrupt,
		task.hasSpokenAny(),
	)
	return true
}

func (s *ChatSession) finishOpenClawWarmup(correlationID string, interrupt bool) bool {
	task := s.takeOpenClawWarmupTask(correlationID)
	if task == nil {
		return false
	}

	task.cancelWarmupOnly()
	if interrupt {
		s.InterruptAndClearTTSQueueWithReason(fmt.Sprintf("OpenClaw warmup finished correlation_id=%s interrupt", correlationID))
	}
	s.endOpenClawSpeech(task)

	log.Infof(
		"OpenClaw warmup finished: device=%s correlation_id=%s interrupt=%v spoke_any=%v",
		s.clientState.DeviceID,
		task.correlationID,
		interrupt,
		task.hasSpokenAny(),
	)
	return true
}

func (s *ChatSession) beginOpenClawSpeech(task *openClawWarmupTask) {
	if task == nil {
		return
	}
	if !task.markSpeechStarted() {
		return
	}
	s.ttsManager.ClearAudioHistory()
	s.ttsManager.EnqueueTtsStartWithReason(task.sessionCtx, fmt.Sprintf("OpenClaw warmup start correlation_id=%s", task.correlationID))
}

func (s *ChatSession) endOpenClawSpeech(task *openClawWarmupTask) {
	if task == nil {
		return
	}
	if !task.markSpeechEnded() {
		return
	}
	s.ttsManager.GetAndClearAudioHistory()
}

func (s *ChatSession) runOpenClawWarmupTask(task *openClawWarmupTask, userText string) {
	planCtx, cancel := context.WithTimeout(task.warmupCtx, openClawWarmupPlanTimeout)
	defer cancel()
	defer log.Infof(
		"OpenClaw warmup task stopped: device=%s correlation_id=%s warmup_err=%v session_err=%v spoke_any=%v",
		s.clientState.DeviceID,
		task.correlationID,
		task.warmupCtx.Err(),
		task.sessionCtx.Err(),
		task.hasSpokenAny(),
	)

	go func() {
		lines, err := s.generateOpenClawWarmupPlan(planCtx, task.correlationID, userText)
		if err != nil {
			if planCtx.Err() == nil {
				log.Warnf("OpenClaw warmup plan generation failed: device=%s correlation_id=%s err=%v", s.clientState.DeviceID, task.correlationID, err)
			}
			task.markPlanReady(time.Time{})
			return
		}
		task.setLines(lines)
		task.markPlanReady(time.Now())
		log.Infof("OpenClaw warmup plan ready: device=%s correlation_id=%s line_count=%d", s.clientState.DeviceID, task.correlationID, len(lines))
	}()

	baseAt, ok := task.waitPlanReady(task.warmupCtx)
	if !ok {
		return
	}

	for idx, delay := range openClawWarmupSchedule {
		if !waitOpenClawWarmupUntil(task.warmupCtx, baseAt.Add(delay)) {
			return
		}
		if task.warmupCtx.Err() != nil {
			return
		}

		text := task.lineAt(idx)
		if text == "" {
			continue
		}

		log.Infof(
			"OpenClaw warmup speaking: device=%s correlation_id=%s slot=%d text=%q",
			s.clientState.DeviceID,
			task.correlationID,
			idx,
			text,
		)
		if err := s.speakOpenClawWarmupLine(task, text); err != nil && task.sessionCtx.Err() == nil {
			log.Warnf("OpenClaw warmup speak failed: device=%s correlation_id=%s slot=%d err=%v", s.clientState.DeviceID, task.correlationID, idx, err)
			return
		}
		task.spokeAny.Store(true)
	}

	// Do not clean up the active task here: the last warm-up audio may still be sent/played,
	// It is necessary to continue to allow OpenClaw to perform preemption interrupts when the first sentence arrives.
}

func waitOpenClawWarmupUntil(ctx context.Context, deadline time.Time) bool {
	wait := time.Until(deadline)
	if wait <= 0 {
		return ctx.Err() == nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (task *openClawWarmupTask) setLines(lines []string) {
	if task == nil || len(lines) == 0 {
		return
	}

	task.linesMu.Lock()
	defer task.linesMu.Unlock()

	if task.lines == nil {
		task.lines = make([]string, openClawWarmupPlanSize)
	}
	for idx := 0; idx < openClawWarmupPlanSize && idx < len(lines); idx++ {
		if text := sanitizeOpenClawWarmupText(lines[idx]); text != "" {
			task.lines[idx] = text
		}
	}
}

func (task *openClawWarmupTask) lineAt(index int) string {
	if task == nil || index < 0 {
		return ""
	}

	task.linesMu.RLock()
	defer task.linesMu.RUnlock()

	if index >= len(task.lines) {
		return ""
	}
	return strings.TrimSpace(task.lines[index])
}

func (s *ChatSession) speakOpenClawWarmupLine(task *openClawWarmupTask, text string) error {
	text = sanitizeOpenClawWarmupText(text)
	if text == "" {
		return nil
	}
	if task == nil {
		return nil
	}
	if task.sessionCtx.Err() != nil {
		return task.sessionCtx.Err()
	}

	s.beginOpenClawSpeech(task)
	if task.sessionCtx.Err() != nil {
		return task.sessionCtx.Err()
	}

	resp := llm_common.LLMResponseStruct{
		Text:    text,
		IsStart: task.takeWarmupSegmentStartFlag(),
		IsEnd:   true,
	}
	// The warm-up sentence needs to ensure that it has entered the sending link to avoid being followed by a formal reply saying "it doesn't look like it took effect".
	return s.ttsManager.handleTextResponse(task.sessionCtx, resp, true)
}

func (s *ChatSession) generateOpenClawWarmupPlan(ctx context.Context, correlationID string, userText string) ([]string, error) {
	llmWrapper, err := pool.Acquire[llm.LLMProvider](
		"llm",
		s.clientState.DeviceConfig.Llm.Provider,
		s.clientState.DeviceConfig.Llm.Config,
	)
	if err != nil {
		return nil, fmt.Errorf("acquire llm provider: %w", err)
	}
	defer pool.Release(llmWrapper)

	dialogue := []*schema.Message{
		schema.SystemMessage(openClawWarmupSystemPrompt),
		schema.UserMessage(buildOpenClawWarmupUserPrompt(userText)),
	}

	msgChan := llmWrapper.GetProvider().ResponseWithContext(
		ctx,
		buildOpenClawWarmupSessionID(s.clientState.SessionID, correlationID),
		dialogue,
		nil,
	)

	raw, err := collectOpenClawWarmupResponse(ctx, msgChan)
	if err != nil {
		return nil, err
	}
	lines := parseOpenClawWarmupPlan(raw)
	if countOpenClawWarmupLines(lines) == 0 {
		return nil, fmt.Errorf("empty warmup plan")
	}
	return lines, nil
}

func buildOpenClawWarmupUserPrompt(userText string) string {
	trimmed := strings.TrimSpace(userText)
	topic := formatOpenClawWarmupTopic(buildOpenClawWarmupHint(userText))
	topicLine := “Do not repeat user command phrases like \”help me search\”.”
	if topic != “” {
		topicLine = fmt.Sprintf(“If you need to mention the topic, distill it into the noun phrase \”%s\” only; do not repeat user command phrases like \”help me search\”.”, topic)
	}
	return fmt.Sprintf(
		“User's current task:\n%s\n\n%s\n\nActual broadcast time points in order: 1st second, 10th second, 20th second, 30th second, 40th second, 50th second, 60th second, 70th second, 80th second, 90th second, 100th second.\nPlease output 11 warmup phrases, one for each of the above 11 time points.”,
		trimmed,
		topicLine,
	)
}

func buildOpenClawWarmupSessionID(sessionID string, correlationID string) string {
	base := strings.TrimSpace(sessionID)
	if base == "" {
		base = "openclaw"
	}
	correlationID = strings.TrimSpace(correlationID)
	if len(correlationID) > 12 {
		correlationID = correlationID[:12]
	}
	if correlationID == "" {
		return base + ":warmup"
	}
	return base + ":warmup:" + correlationID
}

func collectOpenClawWarmupResponse(ctx context.Context, msgChan chan *schema.Message) (string, error) {
	var builder strings.Builder

	for {
		select {
		case <-ctx.Done():
			return builder.String(), ctx.Err()
		case msg, ok := <-msgChan:
			if !ok {
				return builder.String(), nil
			}
			if msg == nil {
				continue
			}
			if llm.IsLLMErrorMessage(msg) {
				errMsg := strings.TrimSpace(llm.LLMErrorMessage(msg))
				if errMsg == "" {
					errMsg = "unknown llm error"
				}
				return builder.String(), fmt.Errorf("llm returned error: %s", errMsg)
			}
			if msg.Content != "" {
				builder.WriteString(msg.Content)
			}
		}
	}
}

func parseOpenClawWarmupPlan(raw string) []string {
	lines := make([]string, openClawWarmupPlanSize)

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return lines
	}

	candidate := raw
	start := strings.Index(candidate, "[")
	end := strings.LastIndex(candidate, "]")
	if start >= 0 && end > start {
		candidate = candidate[start : end+1]
	}

	var objectItems []openClawWarmupLine
	if err := json.Unmarshal([]byte(candidate), &objectItems); err == nil {
		return buildOpenClawWarmupPlanLines(objectItemsToStrings(objectItems))
	}

	var stringItems []string
	if err := json.Unmarshal([]byte(candidate), &stringItems); err == nil {
		return buildOpenClawWarmupPlanLines(stringItems)
	}

	log.Warnf("OpenClaw warmup plan parse failed, ignored: raw=%q", raw)
	return lines
}

func objectItemsToStrings(items []openClawWarmupLine) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, item.Text)
	}
	return lines
}

func buildOpenClawWarmupPlanLines(items []string) []string {
	lines := make([]string, openClawWarmupPlanSize)
	for idx := 0; idx < openClawWarmupPlanSize && idx < len(items); idx++ {
		if text := sanitizeOpenClawWarmupText(items[idx]); text != "" {
			lines[idx] = text
		}
	}
	return lines
}

func countOpenClawWarmupLines(lines []string) int {
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func sanitizeOpenClawWarmupText(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "\"'`[]{}")
	text = strings.TrimLeft(text, "0123456789.、- ")
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return ""
	}
	if isInvalidOpenClawWarmupText(text) {
		return ""
	}

	if len(text) > 80 {
		return ""
	}
	return text
}

func isInvalidOpenClawWarmupText(text string) bool {
	lower := strings.ToLower(text)
	for _, bad := range []string{
		"help me",
		"tell me",
		"please help",
		"can you",
		"could you",
		"search for",
		"look up",
		"find out",
	} {
		if strings.Contains(lower, bad) {
			return true
		}
	}
	return false
}

func buildOpenClawWarmupHint(userText string) string {
	trimmed := strings.TrimSpace(userText)
	if trimmed == "" {
		return ""
	}

	normalized := removePunctuation(trimmed)
	if normalized == "" {
		return ""
	}
	normalized = trimOpenClawWarmupCommandPrefix(normalized)
	normalized = trimOpenClawWarmupQuestionSuffix(normalized)
	if normalized == "" {
		return ""
	}

	for _, keyword := range []string{"weather", "temperature", "forecast"} {
		if idx := strings.Index(strings.ToLower(normalized), keyword); idx >= 0 {
			limit := idx + len(keyword)
			if limit > len(normalized) {
				limit = len(normalized)
			}
			normalized = normalized[:limit]
			break
		}
	}

	words := strings.Fields(normalized)
	if len(words) > 6 {
		words = words[:6]
	}
	return strings.Join(words, " ")
}

func trimOpenClawWarmupCommandPrefix(text string) string {
	trimmed := strings.TrimSpace(text)
	lower := strings.ToLower(trimmed)
	for {
		changed := false
		for _, prefix := range []string{
			"please help me search for",
			"please help me look up",
			"please help me find",
			"could you please search for",
			"could you please look up",
			"could you please find",
			"can you please search for",
			"can you please look up",
			"can you help me find",
			"can you find",
			"can you search",
			"could you search",
			"could you find",
			"help me search for",
			"help me look up",
			"help me find",
			"help me check",
			"tell me about",
			"tell me",
			"i want to know about",
			"i want to know",
			"i'd like to know",
			"search for",
			"look up",
			"find out",
			"find",
			"check",
			"search",
		} {
			if strings.HasPrefix(lower, prefix) {
				trimmed = strings.TrimSpace(trimmed[len(prefix):])
				lower = strings.ToLower(trimmed)
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}
	return trimmed
}

func trimOpenClawWarmupQuestionSuffix(text string) string {
	trimmed := strings.TrimSpace(text)
	lower := strings.ToLower(trimmed)
	for _, suffix := range []string{
		"how is it",
		"how is that",
		"what is it",
		"what's that",
		"right",
		"ok",
	} {
		if strings.HasSuffix(lower, suffix) {
			trimmed = strings.TrimSpace(trimmed[:len(trimmed)-len(suffix)])
			lower = strings.ToLower(trimmed)
		}
	}
	return trimmed
}

func formatOpenClawWarmupTopic(hint string) string {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return ""
	}
	lower := strings.ToLower(hint)
	for _, keyword := range []string{"weather", "temperature", "forecast"} {
		if idx := strings.Index(lower, keyword); idx > 0 {
			prefix := strings.TrimSpace(hint[:idx])
			if prefix == "" || strings.HasSuffix(strings.ToLower(prefix), "in") || strings.HasSuffix(strings.ToLower(prefix), "for") {
				return hint
			}
			return prefix + " " + hint[idx:]
		}
	}
	return hint
}
