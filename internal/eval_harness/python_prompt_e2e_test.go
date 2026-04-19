package eval_harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPythonPromptEndToEnd runs loadPythonPrompt from the repo root so the
// {{PYTHON_VERSION}} placeholder is actually substituted, and asserts the
// resulting text advertises the pinned version and uv invocation. This is the
// regression test for the "model saw 'Python 3' but grader ran 3.9" bug.
func TestPythonPromptEndToEnd(t *testing.T) {
	// Walk up until we find prompts/python.md — tests may run from
	// different working directories.
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	dir := origWD
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(dir, "prompts", "python.md")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("could not locate repo root with prompts/python.md")
		}
		dir = parent
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	got, err := loadPythonPrompt()
	if err != nil {
		t.Fatalf("loadPythonPrompt: %v", err)
	}
	if strings.Contains(got, "{{PYTHON_VERSION}}") {
		t.Error("loadPythonPrompt returned unsubstituted placeholder — substitution wiring is broken")
	}
	pin := PinnedPythonVersion()
	if !strings.Contains(got, pin) {
		t.Errorf("loaded prompt does not mention pinned version %q", pin)
	}
	if !strings.Contains(got, "uv") {
		t.Error("loaded prompt does not mention uv — models won't know the runtime is pinned")
	}
}
