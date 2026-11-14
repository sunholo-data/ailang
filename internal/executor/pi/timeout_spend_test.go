package pi

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

// TestExecuteStreaming_HardTimeoutCarriesSpend pins the 2026-11-03 fix: a run
// killed at its wall-clock bound must bank the elapsed time, the turn/tool
// tally and the tokens seen so far. Before the fix every timeout return path
// built its Result WITHOUT DurationMS, NumTurns or ToolCallCount, so every
// timeout row banked duration_ms=0, 0 turns and $0 — the rig's most expensive
// runs were invisible to wall-time and cost accounting (the
// 2026-10-20..11-03 os-rolling rotation: cp_60166/cp_60159 banked
// mostly-zero timeout rows across pi/motoko/opencode).
func TestExecuteStreaming_HardTimeoutCarriesSpend(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()

	// A pi that streams one full turn (usage-carrying assistant message_end,
	// one tool call) and then hangs until the harness kills it at the hard
	// timeout.
	script := filepath.Join(dir, "pi")
	events := []string{
		`{"type":"session","id":"sess-timeout"}`,
		`{"type":"turn_start"}`,
		`{"type":"tool_execution_start","toolCallId":"toolu_1","toolName":"write","args":{"path":"main.ail","content":"..."}}`,
		`{"type":"message_end","message":{"role":"assistant","usage":{"input":800,"output":90,"cacheRead":0,"cacheWrite":0,"totalTokens":890,"cost":{"input":0,"output":0,"cacheRead":0,"cacheWrite":0,"total":0}},"stopReason":"toolUse"}}`,
	}
	var body strings.Builder
	body.WriteString("#!/bin/sh\n")
	body.WriteString(`
case "$1" in
  --version)
    echo "` + ExpectedVersion + `"
    exit 0 ;;
esac
`)
	for _, ev := range events {
		body.WriteString("printf '%s\\n' '" + strings.ReplaceAll(ev, "'", "'\\''") + "'\n")
	}
	body.WriteString("sleep 30\n")
	if err := os.WriteFile(script, []byte(body.String()), 0755); err != nil {
		t.Fatalf("write fake pi: %v", err)
	}

	e, err := New(&executor.Config{
		PiPath:         script,
		PiModel:        "anthropic/claude-haiku-4-5",
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
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, executor.FinishTimeout)
	}
	// THE defect: the elapsed time must reach the banked row.
	if result.DurationMS < 900 {
		t.Errorf("DurationMS = %d, want >= 900 (the run was killed at a 1s bound)", result.DurationMS)
	}
	if result.DurationMS > 30*1000 {
		t.Errorf("DurationMS = %d, want <= 30s (the run was killed, not waited out)", result.DurationMS)
	}
	// The spend seen before the kill must be carried too, so CarryFailedRunSpend
	// has something to bank.
	if result.NumTurns != 1 {
		t.Errorf("NumTurns = %d, want 1", result.NumTurns)
	}
	if result.ToolCallCount != 1 {
		t.Errorf("ToolCallCount = %d, want 1", result.ToolCallCount)
	}
	if result.InputTokens != 800 || result.OutputTokens != 90 {
		t.Errorf("InputTokens/OutputTokens = %d/%d, want 800/90 (tokens before the kill)", result.InputTokens, result.OutputTokens)
	}
	if result.SessionID != "sess-timeout" {
		t.Errorf("SessionID = %q, want sess-timeout", result.SessionID)
	}
}
