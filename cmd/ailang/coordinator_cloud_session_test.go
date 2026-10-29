package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestNativeSessionArtifactLanding(t *testing.T) {
	for _, provider := range []string{"claude", "pi", "codex", "motoko"} {
		t.Run(provider, func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, "native.jsonl")
			payload := []byte(`{"tool_result":"` + strings.Repeat("full content", 1000) + `"}` + "\n")
			if err := os.WriteFile(source, payload, 0600); err != nil {
				t.Fatal(err)
			}
			result := &executor.Result{SessionID: "id", Transcript: "summary", InputTokens: 12, ProviderData: map[string]any{provider + "_session_jsonl": source}}
			output := captureStdout(t, func() { writeTaskArtifactsAt(root, "task", provider, result, func(time.Duration) {}) })
			if !strings.Contains(output, fmt.Sprintf("session.jsonl written (%d bytes)", len(payload))) {
				t.Fatal(output)
			}
			dest := filepath.Join(root, "tasks", "task")
			got, err := os.ReadFile(filepath.Join(dest, "session.jsonl"))
			if err != nil || string(got) != string(payload) {
				t.Fatalf("copy: %v, bytes %d", err, len(got))
			}
			raw, err := os.ReadFile(filepath.Join(dest, "metrics.json"))
			if err != nil {
				t.Fatal(err)
			}
			var metrics map[string]any
			if err := json.Unmarshal(raw, &metrics); err != nil {
				t.Fatal(err)
			}
			if metrics["session_provider"] != provider || metrics["session_jsonl_bytes"] != float64(len(payload)) || metrics["input_tokens"] != float64(12) {
				t.Fatal(metrics)
			}
			got, err = os.ReadFile(filepath.Join(dest, "transcript.txt"))
			if err != nil || string(got) != "summary" {
				t.Fatalf("transcript: %v", err)
			}
		})
	}
}

func TestNativeSessionRetriesAndFallback(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "late.jsonl")
	if err := os.WriteFile(source, nil, 0600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "tasks", "task")
	if err := os.MkdirAll(dest, 0700); err != nil {
		t.Fatal(err)
	}
	result := &executor.Result{SessionID: "id", ProviderData: map[string]any{"motoko_session_jsonl": source}}
	sleeps := 0
	n := copyNativeSession(root, "task", "motoko", result, func(d time.Duration) {
		if d != 2*time.Second {
			t.Fatal(d)
		}
		sleeps++
		if err := os.WriteFile(source, []byte("complete"), 0600); err != nil {
			t.Fatal(err)
		}
	})
	if n != 8 || sleeps != 1 {
		t.Fatalf("bytes=%d sleeps=%d", n, sleeps)
	}
	// An invalid banked path must not suppress a valid provider-specific fallback.
	cfg := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)
	path := filepath.Join(cfg, "projects", "work", "id.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fallback"), 0600); err != nil {
		t.Fatal(err)
	}
	result.ProviderData = map[string]any{"claude_session_jsonl": filepath.Join(root, "absent")}
	if n := copyNativeSession(root, "task", "claude", result, func(time.Duration) { t.Fatal("unexpected retry") }); n != 8 {
		t.Fatal(n)
	}
}

func TestNativeSessionFailures(t *testing.T) {
	for _, kind := range []string{"missing", "empty", "unreadable", "write"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, "source.jsonl")
			dest := filepath.Join(root, "tasks", "task", "session.jsonl")
			if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "empty":
				if err := os.WriteFile(source, nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "unreadable":
				if err := os.Mkdir(source, 0700); err != nil {
					t.Fatal(err)
				}
			case "write":
				if err := os.WriteFile(source, []byte("native"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(dest, 0700); err != nil {
					t.Fatal(err)
				}
			}
			old := os.Stderr
			capture, err := os.CreateTemp(t.TempDir(), "warnings")
			if err != nil {
				t.Fatal(err)
			}
			os.Stderr = capture
			defer func() { os.Stderr = old; capture.Close() }()
			result := &executor.Result{SessionID: "test-id", ProviderData: map[string]any{"motoko_session_jsonl": source}}
			sleeps := 0
			if n := copyNativeSession(root, "task", "motoko", result, func(time.Duration) { sleeps++ }); n != 0 {
				t.Fatal(n)
			}
			wantSleeps := 2
			if kind == "write" {
				wantSleeps = 0
			}
			if sleeps != wantSleeps {
				t.Fatal(sleeps)
			}
			warning, err := os.ReadFile(capture.Name())
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"warning:", "provider=motoko", "session=test-id", source} {
				if !strings.Contains(string(warning), want) {
					t.Fatalf("warning missing %q: %s", want, warning)
				}
			}
		})
	}
}

func TestNativeSessionDeterministicAndNil(t *testing.T) {
	root := t.TempDir()
	for _, result := range []*executor.Result{nil, {SessionID: "deterministic-bump"}} {
		warning := captureStderr(t, func() {
			if got := copyNativeSession(root, "task", "pi", result, func(time.Duration) { t.Fatal("retry on sessionless run") }); got != 0 {
				t.Fatal(got)
			}
		})
		if warning != "" {
			t.Fatal(warning)
		}
		writeTaskArtifactsAt(root, "task", "pi", result, func(time.Duration) { t.Fatal("retry") })
	}
	if _, err := os.Stat(filepath.Join(root, "tasks", "task", "session.jsonl")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestNativeSessionMissingMetrics(t *testing.T) {
	root := t.TempDir()
	result := &executor.Result{SessionID: "absent"}
	output := captureStderr(t, func() { writeTaskArtifactsAt(root, "task", "motoko", result, func(time.Duration) {}) })
	if !strings.Contains(output, "ProviderData[motoko_session_jsonl]") {
		t.Fatal(output)
	}
	raw, err := os.ReadFile(filepath.Join(root, "tasks", "task", "metrics.json"))
	if err != nil {
		t.Fatal(err)
	}
	var metrics map[string]any
	if err := json.Unmarshal(raw, &metrics); err != nil {
		t.Fatal(err)
	}
	if metrics["session_jsonl_bytes"] != float64(0) {
		t.Fatal(metrics)
	}
}

func TestNativeSessionProviderFallbacks(t *testing.T) {
	for _, provider := range []string{"claude", "pi", "codex"} {
		t.Run(provider, func(t *testing.T) {
			root := t.TempDir()
			home := t.TempDir()
			testutil.SetHomeDir(t, home)
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			t.Setenv("TMP", tmp)
			t.Setenv("TEMP", tmp)
			t.Setenv("CLOUD_RUN_EXECUTION", "execution")
			t.Setenv("CODEX_HOME", filepath.Join(home, "codex"))
			t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "claude"))
			var source string
			switch provider {
			case "claude":
				source = filepath.Join(root, "tasks", "task", "claude", "projects", "work", "id.jsonl")
			case "pi":
				source = filepath.Join(os.TempDir(), "ailang-pi-sessions", "task", "stamp_id.jsonl")
			case "codex":
				source = filepath.Join(home, "codex", "sessions", "2026", "10", "27", "rollout-stamp-id.jsonl")
			}
			if err := os.MkdirAll(filepath.Dir(source), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(source, []byte("native fallback"), 0600); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(root, "tasks", "task")
			if err := os.MkdirAll(dest, 0700); err != nil {
				t.Fatal(err)
			}
			result := &executor.Result{SessionID: "id"}
			if n := copyNativeSession(root, "task", provider, result, func(time.Duration) { t.Fatal("unexpected retry") }); n != 15 {
				t.Fatal(n)
			}
		})
	}
}
