package codex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// TestCodexPromptStaysOffTheProcessTable is the codex half of the 2026-10-27/28 rig-kill
// regression: an eval agent's `pkill -f "ailang run"` matched every sibling agent CLI whose
// argv carried the teaching prompt. A killed codex row was then graded on whatever partial
// solution existed (banked as compile_error, not as a kill). The fake codex records what
// `ps` — and so `pkill -f` — sees for its own process, and what arrived on stdin.
func TestCodexPromptStaysOffTheProcessTable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mock binary tests use /bin/sh; skipping on windows")
	}
	t.Setenv("AILANG_CODEX_RUNTIME", "")
	dir := t.TempDir()
	body := `{"type":"session","session_id":"s1"}` + "\n" +
		`{"type":"message","turn_number":1,"text":"ok","tokens_used":{"input":5,"output":3}}` + "\n" +
		`{"type":"result","tokens_used":{"input":5,"output":3},"status":"success"}` + "\n"
	fixture := filepath.Join(dir, "fixture.jsonl")
	if err := os.WriteFile(fixture, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(dir, "fake-codex")
	script := fmt.Sprintf("#!/bin/sh\nps -ww -o command= -p $$ > %q\ncat > %q\ncat %q\n",
		filepath.Join(dir, "argv.txt"), filepath.Join(dir, "stdin.txt"), fixture)
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	const (
		system    = "TEACHING PROMPT. Run code: ailang run --entry main --caps IO solution.ail"
		directive = "Write solution.py, then check it with python3 solution.py < input.txt"
	)
	e, _ := New(&executor.Config{CodexPath: fake, CodexModel: "gpt-5-codex", TimeoutSeconds: 10})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := e.ExecuteStreaming(ctx, &executor.Task{ID: "t", Directive: directive, SystemPrompt: system}, &executor.NoOpEventHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if !result.Success {
		t.Fatalf("fake run failed: %s", result.Error)
	}

	argvBytes, err := os.ReadFile(filepath.Join(dir, "argv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	argv := string(argvBytes)
	for _, needle := range []string{"ailang run", "solution.py", "TEACHING PROMPT"} {
		if strings.Contains(argv, needle) {
			t.Errorf("codex's process-table command line contains %q — a sibling's `pkill -f %q` would kill it:\n%s",
				needle, needle, argv)
		}
	}
	stdin, err := os.ReadFile(filepath.Join(dir, "stdin.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if want := system + "\n\n" + directive; string(stdin) != want {
		t.Errorf("prompt on stdin = %q, want %q", stdin, want)
	}
}
