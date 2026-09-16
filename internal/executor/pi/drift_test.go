package pi

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// M-PI-HARNESS-UPGRADE M2: drift that matters is LOUD. Before this milestone
// parsePiEvent skipped what it could not parse and the event switch ignored
// unknown types, so a schema change showed up as quietly missing metrics.

func runFake(t *testing.T, events []string) *executor.Result {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	_ = writeFakePi(t, dir, events)
	e, _ := New(&executor.Config{PiPath: filepath.Join(dir, "pi"), PiModel: "anthropic/claude-haiku-4-5", TimeoutSeconds: 10})
	res, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID: "d", Directive: "x", Workspace: dir, Timeout: 10 * time.Second,
	}, &executor.NoOpEventHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	return res
}

func TestHealthCheck_VersionMismatchNamesBoth(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	_ = writeFakePiVersion(t, dir, "0.70.2", nil)
	e, _ := New(&executor.Config{PiPath: filepath.Join(dir, "pi"), PiModel: "anthropic/claude-haiku-4-5", TimeoutSeconds: 5})
	err := e.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected a version-mismatch error, got nil")
	}
	for _, want := range []string{ExpectedVersion, "0.70.2", ExpectedPackage} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("mismatch error must name %q; got: %v", want, err)
		}
	}
}

func TestExecuteStreaming_UnknownEventsBanked(t *testing.T) {
	events := loadFixtureLines(t, "v0_85_1/fizzbuzz.ndjson")
	events = append(events[:1], append([]string{
		`{"type":"totally_new_event","x":1}`,
		`{"type":"totally_new_event","x":2}`,
		`{"type":"another_new_event"}`,
	}, events[1:]...)...)
	res := runFake(t, events)
	if !res.Success {
		t.Fatalf("an unknown event type is RECORDED, never fatal (D4); got Success=false Error=%q", res.Error)
	}
	unk, _ := res.ProviderData["pi_unknown_events"].(map[string]int)
	if unk["totally_new_event"] != 2 || unk["another_new_event"] != 1 {
		t.Fatalf("pi_unknown_events = %v, want {totally_new_event:2, another_new_event:1}", res.ProviderData["pi_unknown_events"])
	}
}

func TestExecuteStreaming_NoUnknownEventsOnCleanStream(t *testing.T) {
	res := runFake(t, loadFixtureLines(t, "v0_85_1/tool_use.ndjson"))
	if _, present := res.ProviderData["pi_unknown_events"]; present {
		t.Fatalf("a clean 0.85.1 stream must bank NO unknown events, got %v", res.ProviderData["pi_unknown_events"])
	}
}

func TestExecuteStreaming_MissingUsageIsFatal(t *testing.T) {
	var events []string
	for _, ln := range loadFixtureLines(t, "v0_85_1/fizzbuzz.ndjson") {
		if strings.Contains(ln, `"type":"message_end"`) && strings.Contains(ln, `"role":"assistant"`) {
			ln = strings.Replace(ln, `"usage":{`, `"usageX":{`, 1)
		}
		events = append(events, ln)
	}
	res := runFake(t, events)
	if res.Success {
		t.Fatal("an assistant message_end without usage is our cost record being WRONG (D4); the run must fail, not bank zero cost")
	}
	if res.FinishReason != executor.FinishWireDrift {
		t.Errorf("FinishReason = %q, want %q", res.FinishReason, executor.FinishWireDrift)
	}
	if !strings.Contains(res.Error, "message_end without usage") {
		t.Errorf("Error must name the missing field, got %q", res.Error)
	}
}

func TestExecuteStreaming_RetriesBanked(t *testing.T) {
	events := loadFixtureLines(t, "v0_85_1/fizzbuzz.ndjson")
	events = append(events[:2], append([]string{
		`{"type":"auto_retry_start","attempt":1,"maxAttempts":3,"delayMs":1000,"errorMessage":"overloaded"}`,
		`{"type":"auto_retry_end","success":true,"attempt":1}`,
	}, events[2:]...)...)
	res := runFake(t, events)
	if !res.Success {
		t.Fatalf("a bounded retry is a slow success, not a failure; got Error=%q", res.Error)
	}
	r, _ := res.ProviderData["pi_retries"].(map[string]any)
	if r["count"] != 1 || r["max_attempts"] != 3 || r["exhausted"] != false {
		t.Fatalf("pi_retries = %v, want {count:1 max_attempts:3 exhausted:false}", res.ProviderData["pi_retries"])
	}
}

func TestExecuteStreaming_RetriesExhaustedIsNamed(t *testing.T) {
	events := loadFixtureLines(t, "v0_85_1/fizzbuzz.ndjson")
	events = append(events[:2], append([]string{
		`{"type":"auto_retry_start","attempt":3,"maxAttempts":3,"delayMs":4000,"errorMessage":"overloaded"}`,
		`{"type":"auto_retry_end","success":false,"attempt":3,"finalError":"overloaded"}`,
	}, events[2:]...)...)
	res := runFake(t, events)
	r, _ := res.ProviderData["pi_retries"].(map[string]any)
	if r["exhausted"] != true {
		t.Fatalf("attempt == maxAttempts with success=false must bank exhausted:true, got %v", res.ProviderData["pi_retries"])
	}
}

func TestExecuteStreaming_UnparsedLinesCounted(t *testing.T) {
	events := loadFixtureLines(t, "v0_85_1/fizzbuzz.ndjson")
	events = append([]string{"(node) DeprecationWarning: something", "not json either"}, events...)
	res := runFake(t, events)
	if !res.Success {
		t.Fatalf("preamble noise is tolerated, got Error=%q", res.Error)
	}
	if res.ProviderData["pi_unparsed_lines"] != 2 {
		t.Fatalf("pi_unparsed_lines = %v, want 2", res.ProviderData["pi_unparsed_lines"])
	}
}

func TestBuildPiArgs_MaxThinkingLevel(t *testing.T) {
	args, err := buildPiArgs("m", &executor.Task{ReasoningEffort: "max"}, "d")
	if err != nil {
		t.Fatalf("`max` is a real pi --thinking level since 0.84; got %v", err)
	}
	if !strings.Contains(strings.Join(args, " "), "--thinking max") {
		t.Fatalf("args = %v", args)
	}
}

func TestExecuteStreaming_FirstAttemptOnLowercaseWrite(t *testing.T) {
	// pi's builtin tools are lowercase in every version; the old "Write"/"Edit"
	// comparison never matched, so first_attempt_ms always fell through to
	// first-text. The tool_use fixture's first write PRECEDES any text delta.
	var events []string
	for _, ln := range loadFixtureLines(t, "v0_85_1/tool_use.ndjson") {
		if strings.Contains(ln, `"text_delta"`) {
			continue // no text at all: first_attempt can ONLY come from the write tool
		}
		events = append(events, ln)
	}
	res := runFake(t, events)
	if res.FirstAttemptMs < 0 {
		t.Fatalf("FirstAttemptMs = %d; the lowercase `write` tool call must count as the first attempt", res.FirstAttemptMs)
	}
}
