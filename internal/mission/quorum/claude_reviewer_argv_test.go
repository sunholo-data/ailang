package quorum

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The real call path, against a fake `claude` on PATH: the system prompt must
// reach the CLI byte-exact through --system-prompt-file, the user prompt on
// stdin, and neither may appear on the process table, where a sibling's
// `pkill -f` matches (2026-10-27/28). The prompt file is gone afterwards.
func TestCallClaudeReview_PromptsStayOffTheProcessTable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake-binary tests need a POSIX shell")
	}
	dir := t.TempDir()
	body := fmt.Sprintf(`#!/bin/sh
ps -ww -o command= -p $$ > %q
while [ $# -gt 0 ]; do
  if [ "$1" = "--system-prompt-file" ]; then cat "$2" > %q; echo "$2" > %q; fi
  shift
done
cat > %q
echo '{"result":"{}","is_error":false}'
`, filepath.Join(dir, "argv.txt"), filepath.Join(dir, "sys.txt"), filepath.Join(dir, "syspath.txt"), filepath.Join(dir, "stdin.txt"))
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	const sys, user = "SYSTEM PROMPT: review this, ailang run", "USER PROMPT: the doc"
	if _, err := callClaudeReview(context.Background(), ClaudeReviewerModel, sys, user); err != nil {
		t.Fatalf("callClaudeReview: %v", err)
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return string(b)
	}
	if argv := read("argv.txt"); strings.Contains(argv, "SYSTEM PROMPT") || strings.Contains(argv, "USER PROMPT") {
		t.Errorf("prompt text on the process table: %s", argv)
	}
	if got := read("sys.txt"); got != sys {
		t.Errorf("system prompt file = %q, want %q", got, sys)
	}
	if got := read("stdin.txt"); got != user {
		t.Errorf("stdin = %q, want %q", got, user)
	}
	if p := strings.TrimSpace(read("syspath.txt")); p == "" {
		t.Error("no --system-prompt-file passed")
	} else if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("system prompt file %s survived the call: %v", p, err)
	}
}
