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
	"github.com/sunholo-data/ailang/internal/testutil"
)

// fakeSetupToken has the shape `claude setup-token` prints; it is not a credential.
const fakeSetupToken = "sk-ant-oat01-test-not-a-real-token"

// A setup-token never becomes a credentials file: Claude Code reads it from the
// environment, and a file would put it where the JSON blob's refresh logic
// would try to use it.
func TestWriteCredentialsFile_SetupTokenWritesNoFile(t *testing.T) {
	home, artifacts := credAuthEnv(t, func(home, _ string) string {
		return filepath.Join(home, ".claude-tasks", "task-1")
	})
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", fakeSetupToken)

	if err := writeCredentialsFile(); err != nil {
		t.Fatalf("writeCredentialsFile with a setup-token: %v", err)
	}
	if got := credentialFilesUnder(t, home); len(got) != 0 {
		t.Errorf("setup-token was written to credential file(s) %v; it belongs in the child env", got)
	}
	if got := credentialFilesUnder(t, artifacts); len(got) != 0 {
		t.Errorf("credential file(s) under the artifact root: %v", got)
	}
}

// The child receives a setup-token in CLAUDE_CODE_OAUTH_TOKEN, and never
// receives the JSON blob, which crashes Claude Code when passed that way.
//
// Measured 2026-10-09: the cloud secrets held JSON blobs whose refresh token was
// spent ("OAuth session expired and could not be refreshed"), so every cloud
// claude run failed in its first turn. A setup-token has no refresh to spend.
func TestClaudeChildEnv_SetupTokenPassedJSONBlobWithheld(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake-binary tests need a POSIX shell")
	}
	for _, c := range []struct {
		name, token string
		wantInChild bool
	}{
		{"setup-token reaches the child", fakeSetupToken, true},
		{"JSON blob stays out of the child", fakeOAuthToken, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			testutil.SetHomeDir(t, home)
			t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude-tasks", "t"))
			t.Setenv("AILANG_AUTH_MODE", "")
			t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", c.token)
			prev := credentialArtifactRoot
			credentialArtifactRoot = filepath.Join(home, "artifacts")
			t.Cleanup(func() { credentialArtifactRoot = prev })

			dir := t.TempDir()
			envDump := filepath.Join(dir, "child.env")
			fake := writeEnvDumpClaude(t, dir, envDump)
			e, err := New(&executor.Config{ClaudePath: fake, ClaudeModel: "haiku"})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if _, err := e.ExecuteStreaming(context.Background(), &executor.Task{
				ID: "setup-token-env", Directive: "noop", Workspace: dir, Timeout: 30 * time.Second,
			}, &MockEventHandler{}); err != nil {
				t.Fatalf("ExecuteStreaming: %v", err)
			}

			data, err := os.ReadFile(envDump)
			if err != nil {
				t.Fatalf("fake claude never ran: %v", err)
			}
			var got string
			for _, line := range strings.Split(string(data), "\n") {
				if v, ok := strings.CutPrefix(line, "CLAUDE_CODE_OAUTH_TOKEN="); ok {
					got = v
				}
			}
			switch {
			case c.wantInChild && got != c.token:
				t.Errorf("child CLAUDE_CODE_OAUTH_TOKEN = %q, want the setup-token", got)
			case !c.wantInChild && got != "":
				t.Errorf("child received CLAUDE_CODE_OAUTH_TOKEN = %q; the JSON blob must not reach it", got)
			}
		})
	}
}

// writeEnvDumpClaude writes the child's environment to envDump, then replays the
// recorded stream so the executor sees a normal run.
func writeEnvDumpClaude(t *testing.T, dir, envDump string) string {
	t.Helper()
	fixture := claudeFixturePath(t, claudeStreamFixture)
	script := filepath.Join(dir, "claude")
	body := fmt.Sprintf(`#!/bin/sh
case "$1" in
  --version) echo "2.0.0 (fake)" ; exit 0 ;;
esac
env > %q
cat %q
exit 0
`, envDump, fixture)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("writeEnvDumpClaude: %v", err)
	}
	return script
}
