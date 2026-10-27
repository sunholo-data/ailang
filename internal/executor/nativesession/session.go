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
	// Walk the resolved root (WalkDir does not follow a symlinked root) but hand
	// back paths under the caller's root: on macOS /var -> /private/var, and a
	// resolved prefix would not match the path the caller passed or expects.
	walkRoot := root
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		walkRoot = resolved
	}
	callerPath := func(p string) string {
		if p == "" {
			return ""
		}
		if rel, err := filepath.Rel(walkRoot, p); err == nil {
			return filepath.Join(root, rel)
		}
		return p
	}
	var match, fallback string
	var latest time.Time
	_ = filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
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
		if exact {
			// An identity match always wins; stop instead of walking a whole
			// local ~/.claude/projects tree.
			match = path
			return filepath.SkipAll
		}
		if newest && provider != "claude" {
			if info, err := d.Info(); err == nil && (fallback == "" || info.ModTime().After(latest)) {
				fallback, latest = path, info.ModTime()
			}
		}
		return nil
	})
	if match != "" {
		return callerPath(match)
	}
	return callerPath(fallback)
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
