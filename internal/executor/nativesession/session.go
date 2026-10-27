// Package nativesession locates opaque provider session files without parsing or
// reconstructing their contents. Newest-file fallback is only safe in isolated storage.
package nativesession

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// Find prefers a session identity match, then (when allowed) the newest native file.
// Callers pass the projects directory itself so a symlink at that boundary works.
func Find(root, provider, id string, newest bool) string {
	if root == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	var match, fallback string
	var latest time.Time
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			return nil
		}
		if provider == "codex" && !strings.HasPrefix(name, "rollout-") {
			return nil
		}
		exact := id != "" && (provider == "claude" && name == id+".jsonl" || provider == "pi" && strings.HasSuffix(name, "_"+id+".jsonl") || provider == "codex" && strings.HasSuffix(name, "-"+id+".jsonl"))
		if exact && match == "" {
			match = path
		}
		if newest && provider != "claude" {
			if info, err := d.Info(); err == nil && (fallback == "" || info.ModTime().After(latest)) {
				fallback, latest = path, info.ModTime()
			}
		}
		return nil
	})
	if match != "" {
		return match
	}
	return fallback
}

// PiDir is task scoped; the same path is used for argv and artifact discovery.
func PiDir(taskID string) string { return filepath.Join(os.TempDir(), "ailang-pi-sessions", taskID) }

// Bank preserves existing provider evidence and never invents a path on a miss.
func Bank(result *executor.Result, provider, path string) {
	if result == nil || path == "" {
		return
	}
	if result.ProviderData == nil {
		result.ProviderData = map[string]any{}
	}
	result.ProviderData[provider+"_session_jsonl"] = path
}
