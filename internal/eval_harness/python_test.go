package eval_harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPinnedPythonVersion verifies the pin is a sensible non-empty "X.Y" value.
func TestPinnedPythonVersion(t *testing.T) {
	v := PinnedPythonVersion()
	if v == "" {
		t.Fatal("PinnedPythonVersion returned empty string")
	}
	if !strings.HasPrefix(v, "3.") {
		t.Errorf("expected PinnedPythonVersion to start with '3.', got %q", v)
	}
	if PinnedPythonMajor != 3 {
		t.Errorf("expected PinnedPythonMajor=3, got %d", PinnedPythonMajor)
	}
	if PinnedPythonMinor < 10 {
		t.Errorf("expected PinnedPythonMinor>=10 (needed for match/case), got %d", PinnedPythonMinor)
	}
}

// TestDetectedPythonVersionMatchesPin confirms that with uv managing the
// runtime, DetectedPythonVersion is simply the pin. Guards against anyone
// reintroducing a "detect and fall back" path without thinking about it.
func TestDetectedPythonVersionMatchesPin(t *testing.T) {
	if got, want := DetectedPythonVersion(), PinnedPythonVersion(); got != want {
		t.Errorf("DetectedPythonVersion()=%q, want %q (with uv they must be equal)", got, want)
	}
}

// TestNewPythonCommandShape verifies that newPythonCommand assembles the
// expected `uv run --python <pin> -- <args...>` invocation. This is the
// contract that keeps prompt strings and actual execution aligned.
//
// If uv is not installed on this machine, the test should report an
// ErrUvMissing with a helpful install hint rather than a generic exec error.
func TestNewPythonCommandShape(t *testing.T) {
	cmd, err := newPythonCommand("solution.py", "--flag")
	if err != nil {
		// uv missing is acceptable on developer machines — just check the
		// error is the one we designed (not a silent fallback).
		if _, ok := err.(*ErrUvMissing); !ok {
			t.Fatalf("expected ErrUvMissing when uv is absent, got %T: %v", err, err)
		}
		if !strings.Contains(err.Error(), "uv") {
			t.Errorf("ErrUvMissing message should mention uv, got: %q", err.Error())
		}
		return
	}
	// uv present — verify the argv shape.
	args := cmd.Args
	if len(args) < 5 {
		t.Fatalf("expected >=5 args, got %d: %v", len(args), args)
	}
	if !strings.HasSuffix(args[0], "uv") {
		t.Errorf("expected binary to be uv, got %q", args[0])
	}
	wantPrefix := []string{"run", "--python", PinnedPythonVersion(), "--"}
	for i, w := range wantPrefix {
		if args[1+i] != w {
			t.Errorf("args[%d]=%q, want %q (full argv: %v)", 1+i, args[1+i], w, args)
		}
	}
	wantTail := []string{"solution.py", "--flag"}
	tail := args[len(args)-len(wantTail):]
	for i, w := range wantTail {
		if tail[i] != w {
			t.Errorf("tail arg[%d]=%q, want %q (full argv: %v)", i, tail[i], w, args)
		}
	}
}

// TestPythonPromptHasVersionPlaceholder catches the case where the prompt file
// loses its {{PYTHON_VERSION}} placeholder — that would silently revert us to
// vague "Python 3" wording that caused the state_machine_elevator fairness
// regression.
func TestPythonPromptHasVersionPlaceholder(t *testing.T) {
	data, err := findAndRead([]string{
		"../../prompts/python.md",
		"prompts/python.md",
	})
	if err != nil {
		t.Fatalf("could not locate prompts/python.md: %v", err)
	}
	if !strings.Contains(string(data), "{{PYTHON_VERSION}}") {
		t.Error("prompts/python.md is missing the {{PYTHON_VERSION}} placeholder — " +
			"the Python runtime version must be substituted into the teaching prompt")
	}
	if !strings.Contains(string(data), "uv") {
		t.Error("prompts/python.md should mention uv so the model knows the runtime is pinned, not best-effort")
	}
}

// TestPythonTaskTemplateHasVersionPlaceholder mirrors the above for the task
// template the agent runner assembles at eval time.
func TestPythonTaskTemplateHasVersionPlaceholder(t *testing.T) {
	data, err := findAndRead([]string{
		"templates/agent_task_python.txt",
		"../../internal/eval_harness/templates/agent_task_python.txt",
		"internal/eval_harness/templates/agent_task_python.txt",
	})
	if err != nil {
		t.Fatalf("could not locate agent_task_python.txt: %v", err)
	}
	if !strings.Contains(string(data), "{{PYTHON_VERSION}}") {
		t.Error("agent_task_python.txt is missing the {{PYTHON_VERSION}} placeholder")
	}
}

func findAndRead(candidates []string) ([]byte, error) {
	var lastErr error
	for _, p := range candidates {
		abs, err := filepath.Abs(p)
		if err != nil {
			lastErr = err
			continue
		}
		data, err := os.ReadFile(abs)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
