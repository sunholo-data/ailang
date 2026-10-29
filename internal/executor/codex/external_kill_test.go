package codex

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

// A SIGTERMed codex exits 0 (measured 2026-10-28: the node wrapper and the
// native binary both shut down "cleanly"), leaving only a turn that started and
// never completed. That run was graded on its half-written solution and banked
// as compile_error; it must come back as a failed external_kill with an Error,
// which makes the eval harness bank it instead of grading it.
func TestCodexMidTurnCleanExitIsExternalKill(t *testing.T) {
	interrupted := `{"type":"thread.started","thread_id":"t1"}` + "\n" +
		`{"type":"turn.started"}` + "\n" +
		`{"type":"item.completed","item":{"type":"agent_message","text":"working"}}` + "\n"
	res := runFakeCodexBody(t, interrupted, "")
	if res.Success {
		t.Fatal("an interrupted codex run reported success")
	}
	if res.FinishReason != executor.FinishExternalKill {
		t.Errorf("FinishReason = %q, want %q", res.FinishReason, executor.FinishExternalKill)
	}
	if !strings.Contains(res.Error, "external kill") {
		t.Errorf("Error = %q: without an Error the harness grades the partial solution", res.Error)
	}
}

// Controls: a turn that FAILED (codex reports its own error) and a completed
// turn are not external kills.
func TestCodexMidTurnControls(t *testing.T) {
	failed := `{"type":"thread.started","thread_id":"t1"}` + "\n" +
		`{"type":"turn.started"}` + "\n" +
		`{"type":"turn.failed","error":{"message":"usage limit"}}` + "\n"
	if res := runFakeCodexBody(t, failed, ""); res.FinishReason == executor.FinishExternalKill {
		t.Errorf("a turn.failed run was classified external_kill (Error %q)", res.Error)
	}
	done := `{"type":"thread.started","thread_id":"t1"}` + "\n" +
		`{"type":"turn.started"}` + "\n" +
		`{"type":"turn.completed","usage":{"input_tokens":5,"cached_input_tokens":0,"output_tokens":3}}` + "\n"
	res := runFakeCodexBody(t, done, "")
	if !res.Success || res.FinishReason != executor.FinishStop || res.Error != "" {
		t.Errorf("completed run: success=%v finish=%q error=%q, want true/stop/empty", res.Success, res.FinishReason, res.Error)
	}
}

// A codex that does die of the signal (exit 143 / signal: terminated) is an
// external kill too.
func TestCodexSIGTERMDeathIsExternalKill(t *testing.T) {
	body := `{"type":"thread.started","thread_id":"t1"}` + "\n" + `{"type":"turn.started"}` + "\n"
	res := runFakeCodexBody(t, body, "kill -TERM $$\n")
	if res.FinishReason != executor.FinishExternalKill {
		t.Errorf("FinishReason = %q, want %q (Error %q)", res.FinishReason, executor.FinishExternalKill, res.Error)
	}
}

func runFakeCodexBody(t *testing.T, body, after string) *executor.Result {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock binary tests use /bin/sh")
	}
	t.Setenv("AILANG_CODEX_RUNTIME", "")
	dir := t.TempDir()
	fixture := filepath.Join(dir, "fixture.jsonl")
	if err := os.WriteFile(fixture, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(dir, "fake-codex")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\ncat >/dev/null\ncat "+fixture+"\n"+after), 0o755); err != nil {
		t.Fatal(err)
	}
	e, _ := New(&executor.Config{CodexPath: fake, CodexModel: "gpt-5-codex", TimeoutSeconds: 10})
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	res, err := e.ExecuteStreaming(ctx, &executor.Task{ID: "t", Directive: "x"}, &executor.NoOpEventHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	return res
}
