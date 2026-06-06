package client

import "time"

// Statistic struct is deprecated; use statistic_plugin to get statistics at MetricTtsStop
type Statistic struct {
	TurnStartTs        int64
	VoiceSilenceTs     int64
	AsrFirstTextTs     int64
	AsrFinalTextTs     int64
	LlmStartTs         int64
	LlmFirstTokenTs    int64
	LlmFirstSentenceTs int64
	LlmEndTs           int64
	TtsStartTs         int64
	TtsFirstFrameTs    int64
	TtsStopTs          int64
}

// MarkTurnStart records the turn start timestamp
func (state *ClientState) MarkTurnStart() {
	state.Statistic.TurnStartTs = time.Now().UnixMilli()
	state.Statistic.VoiceSilenceTs = 0
	state.Statistic.AsrFirstTextTs = 0
	state.Statistic.AsrFinalTextTs = 0
}

// MarkVoiceSilenceAt records the voice silence start timestamp; returns whether this is the first record in this turn
func (state *ClientState) MarkVoiceSilenceAt(ts int64) bool {
	if state.Statistic.VoiceSilenceTs != 0 {
		return false
	}
	state.Statistic.VoiceSilenceTs = ts
	return true
}

// MarkVoiceSilence records the voice silence start timestamp; returns whether this is the first record in this turn
func (state *ClientState) MarkVoiceSilence() bool {
	return state.MarkVoiceSilenceAt(time.Now().UnixMilli())
}

// MarkAsrFirstText records the timestamp of the first ASR text return
func (state *ClientState) MarkAsrFirstText() {
	if state.Statistic.AsrFirstTextTs == 0 {
		state.Statistic.AsrFirstTextTs = time.Now().UnixMilli()
	}
}

// MarkAsrFinalText records the ASR final text timestamp
func (state *ClientState) MarkAsrFinalText() {
	state.MarkAsrFinalTextAt(time.Now().UnixMilli())
}

// MarkAsrFinalTextAt records the ASR final text timestamp; returns whether this is the first record in this turn
func (state *ClientState) MarkAsrFinalTextAt(ts int64) bool {
	if state.Statistic.AsrFinalTextTs != 0 {
		return false
	}
	state.Statistic.AsrFinalTextTs = ts
	return true
}

// MarkLlmStart records the LLM start timestamp
func (state *ClientState) MarkLlmStart() {
	state.Statistic.LlmStartTs = time.Now().UnixMilli()
	state.Statistic.LlmFirstTokenTs = 0
	state.Statistic.LlmFirstSentenceTs = 0
	state.Statistic.LlmEndTs = 0
}

// MarkLlmFirstToken records the timestamp of the first LLM token return
func (state *ClientState) MarkLlmFirstToken() {
	state.Statistic.LlmFirstTokenTs = time.Now().UnixMilli()
}

// MarkLlmFirstSentenceAt records the LLM first sentence output timestamp; returns whether this is the first record in this turn
func (state *ClientState) MarkLlmFirstSentenceAt(ts int64) bool {
	if state.Statistic.LlmFirstSentenceTs != 0 {
		return false
	}
	state.Statistic.LlmFirstSentenceTs = ts
	return true
}

// MarkLlmFirstSentence records the LLM first sentence output timestamp; returns whether this is the first record in this turn
func (state *ClientState) MarkLlmFirstSentence() bool {
	return state.MarkLlmFirstSentenceAt(time.Now().UnixMilli())
}

// MarkLlmEnd records the LLM end timestamp
func (state *ClientState) MarkLlmEnd() {
	state.Statistic.LlmEndTs = time.Now().UnixMilli()
}

// MarkTtsStart records the TTS start timestamp
func (state *ClientState) MarkTtsStart() {
	state.Statistic.TtsStartTs = time.Now().UnixMilli()
	state.Statistic.TtsFirstFrameTs = 0
	state.Statistic.TtsStopTs = 0
}

// MarkTtsFirstFrame records the TTS first frame timestamp
func (state *ClientState) MarkTtsFirstFrame() {
	if state.Statistic.TtsFirstFrameTs == 0 {
		state.Statistic.TtsFirstFrameTs = time.Now().UnixMilli()
	}
}

// MarkTtsStop records the TTS stop timestamp
func (state *ClientState) MarkTtsStop() {
	state.Statistic.TtsStopTs = time.Now().UnixMilli()
}

// SetStartAsrTs sets ASR start timestamp (alias for compatibility)
func (state *ClientState) SetStartAsrTs() { state.MarkVoiceSilence() }

// SetStartLlmTs sets LLM start timestamp (alias for compatibility)
func (state *ClientState) SetStartLlmTs() { state.MarkLlmStart() }

// SetStartTtsTs sets TTS start timestamp (alias for compatibility)
func (state *ClientState) SetStartTtsTs() { state.MarkTtsStart() }

// GetAsrDuration returns ASR processing duration (deprecated, method signature kept for compatibility)
func (state *ClientState) GetAsrDuration() int64 {
	return calcStatisticDuration(state.Statistic.VoiceSilenceTs, state.Statistic.AsrFinalTextTs)
}

// GetAsrLlmTtsDuration returns overall pipeline duration (deprecated, method signature kept for compatibility)
func (state *ClientState) GetAsrLlmTtsDuration() int64 {
	return calcStatisticDuration(state.Statistic.VoiceSilenceTs, state.Statistic.TtsFirstFrameTs)
}

// GetLlmDuration returns LLM duration (deprecated, method signature kept for compatibility)
func (state *ClientState) GetLlmDuration() int64 {
	return calcStatisticDuration(state.Statistic.LlmStartTs, state.Statistic.LlmEndTs)
}

// GetTtsDuration returns TTS duration (deprecated, method signature kept for compatibility)
func (state *ClientState) GetTtsDuration() int64 {
	return calcStatisticDuration(state.Statistic.TtsStartTs, state.Statistic.TtsStopTs)
}

func calcStatisticDuration(start, end int64) int64 {
	if start <= 0 || end <= 0 || end < start {
		return 0
	}
	return end - start
}

func (s *Statistic) Reset() {
	if s == nil {
		return
	}
	*s = Statistic{}
}
