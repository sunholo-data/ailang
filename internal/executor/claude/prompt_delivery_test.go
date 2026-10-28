package claude

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

// TestClaudePromptsStayOffTheProcessTable is the regression for the 2026-10-27/28 rig
// kills: an eval agent's `pkill -f "ailang run"` SIGTERMed every sibling claude whose argv
// carried the teaching prompt (exit status 143, banked as api_error). The fake claude
// records what `ps` — and therefore `pkill -f` — sees for its own process, plus what it
// received on stdin and in --system-prompt-file. Both prompts must reach the child
// byte-exact, and neither may appear on its command line.
func TestClaudePromptsStayOffTheProcessTable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake-binary tests need a POSIX shell")
	}
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("AILANG_AUTH_MODE", "")

	dir := t.TempDir()
	fake := filepath.Join(dir, "claude")
	body := fmt.Sprintf(`#!/bin/sh
case "$1" in --version) echo "2.0.0 (fake)"; exit 0 ;; esac
ps -ww -o command= -p $$ > %q
while [ $# -gt 0 ]; do
  if [ "$1" = "--system-prompt-file" ]; then cat "$2" > %q; fi
  shift
done
cat > %q
cat %q
`, filepath.Join(dir, "argv.txt"), filepath.Join(dir, "sys.txt"),
		filepath.Join(dir, "stdin.txt"), claudeFixturePath(t, claudeStreamFixture))
	if err := os.WriteFile(fake, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	const (
		system    = "TEACHING PROMPT. Run code: ailang run --entry main --caps IO solution.ail"
		directive = "Write solution.py, then check it with python3 solution.py < input.txt"
	)
	e, err := New(&executor.Config{ClaudePath: fake, ClaudeModel: "haiku"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID: "prompt-off-argv", Directive: directive, SystemPrompt: system,
		Workspace: dir, Timeout: 60 * time.Second,
	}, &MockEventHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if !result.Success {
		t.Fatalf("fake run failed: %q", result.Error)
	}

	argv := readFile(t, filepath.Join(dir, "argv.txt"))
	for _, needle := range []string{"ailang run", "solution.py", "TEACHING PROMPT"} {
		if strings.Contains(argv, needle) {
			t.Errorf("claude's process-table command line contains %q — a sibling's `pkill -f %q` would kill it:\n%s",
				needle, needle, argv)
		}
	}
	if got := readFile(t, filepath.Join(dir, "stdin.txt")); got != directive {
		t.Errorf("directive on stdin = %q, want %q", got, directive)
	}
	if got := readFile(t, filepath.Join(dir, "sys.txt")); got != system {
		t.Errorf("system prompt file = %q, want %q", got, system)
	}
}

// TestDeliverPromptCleansUpSystemPromptFile: the temp file holds the full teaching prompt;
// it must not outlive the run, and no file is made when there is no system prompt.
func TestDeliverPromptCleansUpSystemPromptFile(t *testing.T) {
	d, err := deliverPrompt(&executor.Task{Directive: "d", SystemPrompt: "s"})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.args) != 3 || d.args[0] != "-p" || d.args[1] != "--system-prompt-file" {
		t.Fatalf("args = %q, want [-p --system-prompt-file <path>]", d.args)
	}
	path := d.args[2]
	if info, err := os.Stat(path); err != nil {
		t.Fatalf("system prompt file missing before cleanup: %v", err)
	} else if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm&0o077 != 0 {
		// Windows reports every writable file as 0666; its ACLs are not mode bits.
		t.Errorf("system prompt file mode = %v, want owner-only", perm)
	}
	d.cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("system prompt file survived cleanup: %v", err)
	}

	bare, err := deliverPrompt(&executor.Task{Directive: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if len(bare.args) != 1 || bare.args[0] != "-p" {
		t.Errorf("args without a system prompt = %q, want [-p]", bare.args)
	}
}

// An unwritable temp dir must fail the run loudly, before any child starts — not
// fall back to putting the system prompt back on argv.
func TestDeliverPromptFailsLoudlyWithoutATempDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	t.Setenv("TMPDIR", missing) // unix
	t.Setenv("TMP", missing)    // windows
	t.Setenv("TEMP", missing)
	if _, err := deliverPrompt(&executor.Task{Directive: "d", SystemPrompt: "s"}); err == nil {
		t.Fatal("deliverPrompt succeeded with no usable temp dir")
	}
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("AILANG_AUTH_MODE", "")
	e, err := New(&executor.Config{ClaudePath: filepath.Join(missing, "claude"), ClaudeModel: "haiku"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID: "no-tmp", Directive: "d", SystemPrompt: "s", Workspace: t.TempDir(), Timeout: 10 * time.Second,
	}, &MockEventHandler{})
	if err == nil || !strings.Contains(err.Error(), "system prompt file") {
		t.Fatalf("ExecuteStreaming = (%v, %v), want the system-prompt-file error", res, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
