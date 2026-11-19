package eval_harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdirRepoRoot moves the test to the repo root so the real prompt files load.
// Fails (not skips): a test that silently skips proves nothing.
func chdirRepoRoot(t *testing.T) {
	t.Helper()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	dir := origWD
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root (go.mod) from " + origWD)
		}
		dir = parent
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
}

// TestPythonPromptEndToEnd drives the Python prompts a model is ACTUALLY shown —
// the agent system prompt and the standard-mode prompt — and asserts the
// runtime placeholder is substituted with the pinned version. Regression test
// for two bugs: "model saw 'Python 3' but grader ran 3.9", and (2026-11-16)
// both live paths sending a literal {{PYTHON_VERSION}} while this test guarded
// a loader nothing called.
func TestPythonPromptEndToEnd(t *testing.T) {
	chdirRepoRoot(t)
	pin := PinnedPythonVersion()

	sys, _, err := LoadSystemPromptForLanguage("python", "")
	if err != nil {
		t.Fatalf("LoadSystemPromptForLanguage(python): %v", err)
	}
	spec := &BenchmarkSpec{ID: "py_e2e", Prompt: "Print hello."}
	base, task := spec.PromptPartsForLanguage("python")

	for name, got := range map[string]string{
		"agent system prompt":  sys,
		"standard-mode prompt": base + task,
	} {
		if strings.Contains(got, "{{PYTHON_VERSION}}") {
			t.Errorf("%s: unsubstituted {{PYTHON_VERSION}} reaches the model", name)
		}
		if !strings.Contains(got, "--python "+pin) {
			t.Errorf("%s: does not tell the model `uv run --python %s`", name, pin)
		}
	}
}
