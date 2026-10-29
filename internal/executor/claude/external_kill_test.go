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

// A claude that a sibling's `pkill -f` SIGTERMs traps the signal and exits 143
// with empty stderr. That was banked as api_error on 2026-10-27/28; the
// executor must report it as external_kill so the harness banks it invalid.
// The control is an ordinary non-zero exit, which stays FinishError.
func TestClaudeSIGTERMExitIsExternalKill(t *testing.T) {
	for _, tc := range []struct {
		name, exit string
		want       string
	}{
		{"trapped SIGTERM (exit 143)", "exit 143", executor.FinishExternalKill},
		{"untrapped SIGTERM", "kill -TERM $$", executor.FinishExternalKill},
		{"ordinary failure", "exit 1", executor.FinishError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				t.Skip("fake-binary tests need a POSIX shell")
			}
			t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
			t.Setenv("AILANG_AUTH_MODE", "")
			dir := t.TempDir()
			fake := filepath.Join(dir, "claude")
			body := fmt.Sprintf("#!/bin/sh\ncase \"$1\" in --version) echo '2.0.0 (fake)'; exit 0 ;; esac\ncat >/dev/null\nhead -n 3 %q\n%s\n",
				claudeFixturePath(t, claudeStreamFixture), tc.exit)
			if err := os.WriteFile(fake, []byte(body), 0o755); err != nil {
				t.Fatal(err)
			}
			e, err := New(&executor.Config{ClaudePath: fake, ClaudeModel: "haiku"})
			if err != nil {
				t.Fatal(err)
			}
			res, err := e.ExecuteStreaming(context.Background(), &executor.Task{
				ID: "sigterm", Directive: "x", Workspace: dir, Timeout: 30 * time.Second,
			}, &MockEventHandler{})
			if err != nil {
				t.Fatalf("ExecuteStreaming: %v", err)
			}
			if res.Success {
				t.Fatal("a killed run reported success")
			}
			if res.FinishReason != tc.want {
				t.Errorf("FinishReason = %q, want %q (Error %q)", res.FinishReason, tc.want, res.Error)
			}
			if tc.want == executor.FinishExternalKill && !strings.Contains(res.Error, "external kill") {
				t.Errorf("Error = %q, want it to name the external kill", res.Error)
			}
		})
	}
}
