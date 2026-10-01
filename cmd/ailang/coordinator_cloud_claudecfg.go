package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cloudArtifactRoot is where the cloud executor job mounts the shared artifacts
// bucket (gcsfuse, read-write, on every lane — including the external apikey
// lanes). Anything written under it is readable by every lane's job SA, and
// gcsfuse ignores file modes, so a 0600 file there is not private.
const cloudArtifactRoot = "/artifacts"

// taskClaudeArtifactDir is the bucket path that has always held a task's Claude
// session logs: <root>/tasks/<taskID>/claude, with the JSONL at
// projects/<encoded-cwd>/<sessionID>.jsonl beneath it.
func taskClaudeArtifactDir(artifactRoot, taskID string) string {
	return filepath.Join(artifactRoot, "tasks", taskID, "claude")
}

// setupClaudeConfigDir prepares CLAUDE_CONFIG_DIR for one cloud task (F-H6-1).
//
// CLAUDE_CONFIG_DIR used to BE the bucket path, so session JSONL streamed to GCS
// with no upload step — and so did everything else Claude Code keeps there:
// .credentials.json (the Claude Max OAuth access + refresh token, written by
// writeCredentialsFile and rewritten by the CLI on refresh), .claude.json
// (account identity; in apikey mode the approved key suffix), shell snapshots,
// todos, file history. Every lane mounts that bucket, so every lane could read
// every past task's credential.
//
// Now the config dir is local disk — <localBase>/<taskID>, mode 0700, outside
// the cloned workspace (/workspace/<taskID>) so the agent cannot commit it — and
// only its projects/ entry is a symlink to <bucket>/tasks/<taskID>/claude/projects.
// Claude still appends the session JSONL straight to the gcsfuse path while the
// task runs, at the same object names as before; nothing else reaches the bucket.
//
// Returns the local config dir.
func setupClaudeConfigDir(localBase, artifactRoot, taskID string) (string, error) {
	if taskID == "" || filepath.Base(taskID) != taskID || taskID == "." || taskID == ".." {
		return "", fmt.Errorf("claude config dir: invalid task id %q", taskID)
	}
	if rel, err := filepath.Rel(filepath.Clean(artifactRoot), filepath.Clean(localBase)); err == nil &&
		(rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))) {
		return "", fmt.Errorf("claude config dir: local base %s is inside the artifact root %s", localBase, artifactRoot)
	}
	if err := os.MkdirAll(localBase, 0o700); err != nil {
		return "", fmt.Errorf("claude config dir: create %s: %w", localBase, err)
	}
	configDir := filepath.Join(localBase, taskID)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return "", fmt.Errorf("claude config dir: create %s: %w", configDir, err)
	}
	// MkdirAll leaves an existing dir's mode alone, and the umask can narrow a
	// new one; set it explicitly — this dir holds the OAuth credential.
	if err := os.Chmod(configDir, 0o700); err != nil {
		return "", fmt.Errorf("claude config dir: chmod %s: %w", configDir, err)
	}

	projectsTarget := filepath.Join(taskClaudeArtifactDir(artifactRoot, taskID), "projects")
	if err := os.MkdirAll(projectsTarget, 0o755); err != nil {
		return "", fmt.Errorf("claude config dir: create %s: %w", projectsTarget, err)
	}
	link := filepath.Join(configDir, "projects")
	if cur, err := os.Readlink(link); err == nil && cur == projectsTarget {
		return configDir, nil
	}
	if err := os.RemoveAll(link); err != nil {
		return "", fmt.Errorf("claude config dir: clear %s: %w", link, err)
	}
	if err := os.Symlink(projectsTarget, link); err != nil {
		return "", fmt.Errorf("claude config dir: link %s -> %s: %w", link, projectsTarget, err)
	}
	return configDir, nil
}

// cloudClaudeConfigBase is the local parent of per-task Claude config dirs:
// $HOME/.claude-tasks, falling back to the OS temp dir when HOME is unusable.
func cloudClaudeConfigBase() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".claude-tasks")
	}
	return filepath.Join(os.TempDir(), "ailang-claude-tasks")
}
