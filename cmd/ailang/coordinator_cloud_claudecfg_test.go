package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// claudeLocalOnlyFiles are what Claude Code keeps in CLAUDE_CONFIG_DIR besides
// projects/. Each holds a secret or account data and must stay on local disk:
// .credentials.json is the Max OAuth access + refresh token; .claude.json carries
// the account identity (and, in apikey mode, the approved key suffix).
var claudeLocalOnlyFiles = []string{
	".credentials.json",
	".claude.json",
	"todos/t.json",
	"shell-snapshots/snapshot.sh",
	"statsig/cache",
	"file-history/x/1",
}

// F-H6-1 regression: after setup, a simulated Claude run that writes its config
// files AND a session JSONL through CLAUDE_CONFIG_DIR leaves no secret file under
// the artifact root, while the JSONL lands at the bucket path readers expect.
func TestSetupClaudeConfigDir_OnlySessionLogsReachArtifacts(t *testing.T) {
	root := t.TempDir()
	artifacts := filepath.Join(root, "artifacts")
	localBase := filepath.Join(root, "home", ".claude-tasks")
	const taskID = "task-abc123"
	const sessionID = "0b7c-session"

	configDir, err := setupClaudeConfigDir(localBase, artifacts, taskID)
	if err != nil {
		t.Fatalf("setupClaudeConfigDir: %v", err)
	}
	if got := filepath.Join(localBase, taskID); configDir != got {
		t.Fatalf("configDir = %s, want %s", configDir, got)
	}
	if info, err := os.Stat(configDir); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("config dir mode: info=%v err=%v, want 0700", info, err)
	}
	if strings.HasPrefix(configDir, artifacts) || strings.HasPrefix(configDir, "/workspace/") {
		t.Fatalf("config dir %s must be off the artifact mount and out of the workspace", configDir)
	}

	// Simulate Claude Code: config files at the top level, the session under projects/.
	for _, rel := range claudeLocalOnlyFiles {
		p := filepath.Join(configDir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	sessDir := filepath.Join(configDir, "projects", "-workspace-"+taskID)
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		t.Fatalf("mkdir through projects symlink: %v", err)
	}
	f, err := os.OpenFile(filepath.Join(sessDir, sessionID+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"type":"user"}` + "\n")
	_, _ = f.WriteString(`{"type":"assistant"}` + "\n")
	_ = f.Close()

	// Nothing but session logs under the artifact root.
	_ = filepath.WalkDir(artifacts, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".jsonl") {
			t.Errorf("non-session file reached the artifact root: %s", p)
		}
		return nil
	})

	// The JSONL is at the same object path as before the fix.
	want := filepath.Join(artifacts, "tasks", taskID, "claude", "projects", "-workspace-"+taskID, sessionID+".jsonl")
	if data, err := os.ReadFile(want); err != nil || strings.Count(string(data), "\n") != 2 {
		t.Fatalf("session JSONL at %s: data=%q err=%v", want, data, err)
	}

	// writeTaskArtifacts' lookup: it searches the bucket path, because WalkDir
	// does not descend through the projects symlink in the local config dir.
	if got := findSessionJSONL(taskClaudeArtifactDir(artifacts, taskID), sessionID); got != want {
		t.Errorf("findSessionJSONL(bucket path) = %q, want %q", got, want)
	}
	if got := findSessionJSONL(configDir, sessionID); got != "" {
		t.Logf("note: WalkDir followed the symlink on this platform (%s)", got)
	}

	// Idempotent: a second setup keeps the link and the logs.
	if again, err := setupClaudeConfigDir(localBase, artifacts, taskID); err != nil || again != configDir {
		t.Fatalf("second setup: dir=%s err=%v", again, err)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("second setup lost the session log: %v", err)
	}
}

func TestSetupClaudeConfigDir_Refusals(t *testing.T) {
	root := t.TempDir()
	artifacts := filepath.Join(root, "artifacts")

	if _, err := setupClaudeConfigDir(filepath.Join(artifacts, "home"), artifacts, "task-1"); err == nil {
		t.Error("a local base inside the artifact root must be refused")
	}
	for _, bad := range []string{"", ".", "..", "a/b", "../escape"} {
		if _, err := setupClaudeConfigDir(filepath.Join(root, "local"), artifacts, bad); err == nil {
			t.Errorf("task id %q must be refused", bad)
		}
	}
}
