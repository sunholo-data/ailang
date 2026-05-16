package eval_harness

import (
	"os"
	"strings"
	"testing"
)

// TestRunPythonSolution_WiresCliArgs covers the v0.20.0 regression where
// the agent-mode Python runner dropped spec.CliArgs from the subprocess
// invocation. Reproduces cli_args-style benchmarks: write a tiny Python
// program that reads sys.argv[1] and prints it; assert the runner passes
// the arg through and stdout matches.
//
// See design_docs/planned/v0_21_0/m-eval-agent-python-stdio-wiring.md for
// the full bug context.
func TestRunPythonSolution_WiresCliArgs(t *testing.T) {
	if _, err := resolveUv(); err != nil {
		t.Skipf("uv not available on this machine: %v", err)
	}

	dir := t.TempDir()
	solutionPath := dir + "/solution.py"
	if err := os.WriteFile(solutionPath, []byte("import sys\nprint(sys.argv[1])\n"), 0o644); err != nil {
		t.Fatalf("write solution: %v", err)
	}

	spec := &BenchmarkSpec{
		ID:          "test_cli_args",
		CliArgs:     []string{"hello-from-cli-args"},
		ExpectedOut: "hello-from-cli-args\n",
	}

	result := runPythonSolution(solutionPath, spec)

	if !result.CompileOk {
		t.Errorf("CompileOk = false, want true; stderr: %q", result.Stderr)
	}
	if !result.RuntimeOk {
		t.Errorf("RuntimeOk = false, want true; stderr: %q\nstdout: %q", result.Stderr, result.Stdout)
	}
	if !result.StdoutOk {
		t.Errorf("StdoutOk = false (cli arg was NOT delivered)\n  expected: %q\n  got:      %q",
			spec.ExpectedOut, result.Stdout)
	}
}

// TestRunPythonSolution_WiresStdin covers the v0.20.0 regression where
// the agent-mode Python runner failed to pipe spec.Stdin to the subprocess.
// Reproduces pipeline-style benchmarks: solution reads stdin, doubles each
// number; assert the runner pipes the spec.Stdin payload and stdout matches.
func TestRunPythonSolution_WiresStdin(t *testing.T) {
	if _, err := resolveUv(); err != nil {
		t.Skipf("uv not available on this machine: %v", err)
	}

	dir := t.TempDir()
	solutionPath := dir + "/solution.py"
	code := "import sys\nfor line in sys.stdin:\n    n = int(line.strip())\n    print(n * 2)\n"
	if err := os.WriteFile(solutionPath, []byte(code), 0o644); err != nil {
		t.Fatalf("write solution: %v", err)
	}

	spec := &BenchmarkSpec{
		ID:          "test_stdin",
		Stdin:       "1\n2\n3\n",
		ExpectedOut: "2\n4\n6\n",
	}

	result := runPythonSolution(solutionPath, spec)

	if !result.RuntimeOk {
		t.Errorf("RuntimeOk = false, want true; stderr: %q\nstdout: %q", result.Stderr, result.Stdout)
	}
	if !result.StdoutOk {
		t.Errorf("StdoutOk = false (stdin was NOT piped)\n  expected: %q\n  got:      %q",
			spec.ExpectedOut, result.Stdout)
	}
}

// TestRunPythonSolution_NoSpecInputsIsBenign verifies that the fix doesn't
// regress the common case where a spec has no CliArgs or Stdin — the
// subprocess should run cleanly with just the solution path.
func TestRunPythonSolution_NoSpecInputsIsBenign(t *testing.T) {
	if _, err := resolveUv(); err != nil {
		t.Skipf("uv not available on this machine: %v", err)
	}

	dir := t.TempDir()
	solutionPath := dir + "/solution.py"
	if err := os.WriteFile(solutionPath, []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("write solution: %v", err)
	}

	spec := &BenchmarkSpec{
		ID:          "test_no_inputs",
		ExpectedOut: "hello\n",
	}

	result := runPythonSolution(solutionPath, spec)

	if !result.StdoutOk {
		t.Errorf("StdoutOk = false on a no-input benchmark — fix introduced a regression\n  expected: %q\n  got:      %q\n  stderr:   %q",
			spec.ExpectedOut, result.Stdout, result.Stderr)
	}
}

// ensure strings stays imported in this file if it ever needs trimming.
var _ = strings.TrimSpace
