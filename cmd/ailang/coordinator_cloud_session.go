package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/executor/nativesession"
)

// sessionCandidates keeps banked paths first and records roots even on lookup
// misses. Recompute each attempt so files appearing during mount flush are found.
func sessionCandidates(root, taskID, provider string, result *executor.Result) ([]string, []string) {
	var candidates, searched []string
	add := func(path string) {
		if path != "" {
			candidates = append(candidates, path)
			searched = append(searched, path)
		}
	}
	if p, ok := result.ProviderData[provider+"_session_jsonl"].(string); ok {
		add(p)
	}
	find := func(dir string, newest bool) {
		searched = append(searched, dir)
		if p := nativesession.Find(dir, provider, result.SessionID, newest); p != "" {
			add(p)
		}
	}
	home, homeErr := os.UserHomeDir()
	switch provider {
	case "claude":
		find(filepath.Join(taskClaudeArtifactDir(root, taskID), "projects"), false)
		if dir := config.ClaudeConfigDir(); dir != "" {
			find(filepath.Join(dir, "projects"), false)
		}
		if homeErr == nil {
			find(filepath.Join(home, ".claude", "projects"), false)
		} else {
			searched = append(searched, "home unavailable: "+homeErr.Error())
		}
	case "pi":
		find(nativesession.PiDir(taskID), true)
	case "codex":
		dir := config.CodexHome()
		isolated := dir != "" && config.CloudRunExecution() != ""
		if dir == "" {
			if homeErr != nil {
				searched = append(searched, "home unavailable: "+homeErr.Error())
				break
			}
			dir = filepath.Join(home, ".codex")
		}
		find(filepath.Join(dir, "sessions"), isolated)
	case "motoko":
		searched = append(searched, "ProviderData[motoko_session_jsonl]")
	default:
		searched = append(searched, "unsupported provider "+provider)
	}
	return candidates, searched
}

func copyNativeSession(root, taskID, provider string, result *executor.Result, sleep func(time.Duration)) int {
	if result == nil || result.SessionID == "deterministic-bump" {
		return 0
	}
	var searched []string
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		paths, roots := sessionCandidates(root, taskID, provider, result)
		searched = roots
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				lastErr = err
				continue
			}
			if len(data) == 0 {
				lastErr = fmt.Errorf("empty session file %s", path)
				continue
			}
			dst := filepath.Join(root, "tasks", taskID, "session.jsonl")
			if err := os.WriteFile(dst, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "execute-job: warning: session.jsonl write failed provider=%s session=%s searched=%v destination=%s: %v\n", provider, result.SessionID, searched, dst, err)
				return 0
			}
			fmt.Printf("execute-job: session.jsonl written (%d bytes)\n", len(data))
			return len(data)
		}
		if attempt < 2 {
			sleep(2 * time.Second)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no native session found")
	}
	fmt.Fprintf(os.Stderr, "execute-job: warning: session.jsonl unavailable provider=%s session=%s searched=%s: %v\n", provider, result.SessionID, strings.Join(searched, ", "), lastErr)
	return 0
}

// findSessionJSONL retains the config-root contract for existing callers.
func findSessionJSONL(claudeConfigDir, sessionID string) string {
	// The bucket config root contains projects, which may be a symlink elsewhere.
	return nativesession.Find(filepath.Join(claudeConfigDir, "projects"), "claude", sessionID, false)
}
