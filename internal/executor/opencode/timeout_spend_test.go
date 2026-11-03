package opencode

// Split from the timeout-shape fix of 2026-11-03: the timeout return paths
// dropped DurationMS, FinishReason and the turn/tool tally, so a wall-clock
// timeout banked as api_error — or, with zero turns reported, as non_agentic
// (the 2026-10-20..11-03 os-rolling rotation: opencode cp_60166 rows read
// "non_agentic 2 + api_error 2" for what were timeouts). These tests pin the
// full spend shape on the killed paths.

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// TestExecuteStreaming_HardTimeoutCarriesSpend pins the fix: a run killed at
// its wall-clock bound must bank the elapsed time, the step/tool tally, the
// tokens seen so far, and FinishReason=timeout.
func TestExecuteStreaming_HardTimeoutCarriesSpend(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()

	// An opencode that emits one full step (token-carrying step_finish, one
	// tool call) and then hangs until the harness kills it at the hard
	// timeout.
	script := filepath.Join(dir, "opencode")
	events := []string{
		`{"type":"step_start","timestamp":1,"sessionID":"ses_timeout","part":{"id":"p1","messageID":"m1","sessionID":"ses_timeout","type":"step-start"}}`,
		`{"type":"tool_use","timestamp":2,"sessionID":"ses_timeout","part":{"type":"tool","tool":"write","callID":"toolu_1","state":{"status":"completed","input":{"filePath":"/tmp/x.txt","content":"hi"},"output":"Wrote file successfully.","title":"x.txt","time":{"start":1,"end":2}},"id":"p2","sessionID":"ses_timeout","messageID":"m1"}}`,
		`{"type":"step_finish","timestamp":3,"sessionID":"ses_timeout","part":{"id":"p3","reason":"tool-calls","messageID":"m1","sessionID":"ses_timeout","type":"step-finish","tokens":{"total":900,"input":100,"output":50,"reasoning":750,"cache":{"write":0,"read":0}},"cost":0.001}}`,
	}
	var body strings.Builder
	body.WriteString("#!/bin/sh\n")
	body.WriteString(`
case "$1" in
  run)
    true ;;  # fall through to event emission
  --version)
    echo "1.14.20"
    exit 0 ;;
  *)
    exit 0 ;;
esac
`)
	for _, ev := range events {
		body.WriteString("printf '%s\\n' '" + strings.ReplaceAll(ev, "'", "'\\''") + "'\n")
	}
	body.WriteString("sleep 30\n")
	if err := os.WriteFile(script, []byte(body.String()), 0755); err != nil {
		t.Fatalf("write fake opencode: %v", err)
	}

	e, err := New(&executor.Config{
		OpenCodePath:   script,
		OpenCodeModel:  "anthropic/claude-haiku-4-5",
		TimeoutSeconds: 10,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	result, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID:        "test-timeout-spend",
		Directive: "solve cp_60166",
		Workspace: dir,
		Timeout:   1 * time.Second,
	}, &collectingHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}

	if result.Success {
		t.Error("a run killed at its wall-clock bound must not report success")
	}
	if result.FinishReason != executor.FinishTimeout {
		t.Errorf("FinishReason = %q, want %q (was: empty, so the row banked as api_error/non_agentic)", result.FinishReason, executor.FinishTimeout)
	}
	// THE defect: the elapsed time must reach the banked row.
	if result.DurationMS < 900 {
		t.Errorf("DurationMS = %d, want >= 900 (the run was killed at a 1s bound)", result.DurationMS)
	}
	if result.DurationMS > 30*1000 {
		t.Errorf("DurationMS = %d, want <= 30s (the run was killed, not waited out)", result.DurationMS)
	}
	if result.NumTurns != 1 {
		t.Errorf("NumTurns = %d, want 1", result.NumTurns)
	}
	if result.ToolCallCount != 1 {
		t.Errorf("ToolCallCount = %d, want 1", result.ToolCallCount)
	}
	if result.InputTokens != 100 || result.OutputTokens != 50 || result.ReasonTokens != 750 {
		t.Errorf("InputTokens/OutputTokens/ReasonTokens = %d/%d/%d, want 100/50/750 (tokens before the kill)", result.InputTokens, result.OutputTokens, result.ReasonTokens)
	}
	if result.SessionID != "ses_timeout" {
		t.Errorf("SessionID = %q, want ses_timeout", result.SessionID)
	}
}
