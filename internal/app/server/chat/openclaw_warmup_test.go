package chat

import (
	"strings"
	"testing"
)

func TestParseOpenClawWarmupPlanObjects(t *testing.T) {
	got := parseOpenClawWarmupPlan(`[{"text":"我先看一下天气。"},{"text":"天气情况我继续跟进。"},{"text":"这个问题我还在处理。"},{"text":"天气结果还在路上。"},{"text":"我继续盯着天气。"},{"text":"这边继续核对天气。"},{"text":"天气数据还在更新。"},{"text":"我继续盯着最新预报。"},{"text":"这边还在做最后确认。"},{"text":"结果一到就告诉你。"},{"text":"我再看一眼天气。"}]`)

	if len(got) != openClawWarmupPlanSize {
		t.Fatalf("unexpected plan size: got %d want %d", len(got), openClawWarmupPlanSize)
	}
	if got[0] != "我先看一下天气。" {
		t.Fatalf("unexpected first line: %q", got[0])
	}
	if got[4] != "我继续盯着天气。" {
		t.Fatalf("unexpected last line: %q", got[4])
	}
	if got[9] != "结果一到就告诉你。" {
		t.Fatalf("unexpected tenth line: %q", got[9])
	}
	if got[10] != "我再看一眼天气。" {
		t.Fatalf("unexpected eleventh line: %q", got[10])
	}
}

func TestParseOpenClawWarmupPlanReturnsEmptyOnInvalidJSON(t *testing.T) {
	got := parseOpenClawWarmupPlan("not-json")

	for idx, line := range got {
		if line != "" {
			t.Fatalf("expected empty line at %d, got %q", idx, line)
		}
	}
}

func TestBuildOpenClawWarmupHint(t *testing.T) {
	got := buildOpenClawWarmupHint("帮我查一下上海今天天气怎么样？")
	if got == "" {
		t.Fatal("expected non-empty hint")
	}
	if strings.Contains(got, "帮我") {
		t.Fatalf("hint should not contain user command: %q", got)
	}
	if len([]rune(got)) > 10 {
		t.Fatalf("hint too long: %q", got)
	}
}

func TestBuildOpenClawWarmupHintWeatherTopic(t *testing.T) {
	got := buildOpenClawWarmupHint("天津后天的天气怎么样？")
	if got != "天津后天的天气" {
		t.Fatalf("unexpected weather hint: %q", got)
	}
}

func TestBuildOpenClawWarmupUserPromptIncludesTimeline(t *testing.T) {
	got := buildOpenClawWarmupUserPrompt("天津后天的天气怎么样？")
	if !strings.Contains(got, "用户本轮任务：") {
		t.Fatalf("task label missing from prompt: %q", got)
	}
	if !strings.Contains(got, "只能提炼成名词短语"天津后天的天气"") {
		t.Fatalf("topic hint missing from prompt: %q", got)
	}
	if !strings.Contains(got, "第1秒、第10秒、第20秒、第30秒、第40秒、第50秒、第60秒、第70秒、第80秒、第90秒、第100秒") {
		t.Fatalf("timeline missing from prompt: %q", got)
	}
}

func TestFormatOpenClawWarmupTopicWeather(t *testing.T) {
	got := formatOpenClawWarmupTopic("天津后天的天气")
	if got != "天津后天的天气" {
		t.Fatalf("unexpected formatted topic: %q", got)
	}
}

func TestSanitizeOpenClawWarmupTextRejectsUserCommandEcho(t *testing.T) {
	got := sanitizeOpenClawWarmupText("我先看看帮我查询一下。")
	if got != "" {
		t.Fatalf("expected invalid warmup text to be rejected, got %q", got)
	}
}

func TestTakeWarmupSegmentStartFlagOnlyMarksFirstWarmupSentence(t *testing.T) {
	task := &openClawWarmupTask{nextWarmupSegmentIsStart: true}

	if !task.takeWarmupSegmentStartFlag() {
		t.Fatal("expected first warmup sentence to carry start flag")
	}
	if task.takeWarmupSegmentStartFlag() {
		t.Fatal("expected subsequent warmup sentence to clear start flag")
	}
}
