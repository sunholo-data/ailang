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
	// Normalize CRLF→LF: Python's print() emits \r\n on Windows. The bug we're
	// regression-testing is "stdin reaches the subprocess and is processed" —
	// line-ending discipline is orthogonal and would only mask the wiring bug.
	gotNormalized := strings.ReplaceAll(result.Stdout, "\r\n", "\n")
	if gotNormalized != spec.ExpectedOut {
		t.Errorf("stdout mismatch (stdin may not have been piped)\n  expected: %q\n  got:      %q",
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

// TestBuildChainMetadata_PopulatesBothWhenSet covers the M-EVAL-LOCAL-OBSERVABILITY
// FOLLOWUP wiring: when MultiExecutorConfig.ChainID and StageID are both set,
// the resulting Task.Metadata must include both keys with the expected values
// so executor.BuildEnvironment can convert them into OTEL_RESOURCE_ATTRIBUTES.
func TestBuildChainMetadata_PopulatesBothWhenSet(t *testing.T) {
	m := buildChainMetadata("chain-test-abc", "stage-test-xyz")
	if got, want := m["chain_id"], "chain-test-abc"; got != want {
		t.Errorf("chain_id = %q, want %q", got, want)
	}
	if got, want := m["stage_id"], "stage-test-xyz"; got != want {
		t.Errorf("stage_id = %q, want %q", got, want)
	}
	if len(m) != 2 {
		t.Errorf("expected 2 keys, got %d: %v", len(m), m)
	}
}

// TestBuildChainMetadata_OmitsEmpty verifies that empty IDs are NOT inserted
// as empty-string values (would otherwise pollute resource attrs with
// "ailang.chain_id=" which downstream code would have to filter).
func TestBuildChainMetadata_OmitsEmpty(t *testing.T) {
	cases := []struct {
		name             string
		chainID, stageID string
		wantLen          int
		wantKey          string
	}{
		{"both empty (coordinator path)", "", "", 0, ""},
		{"only chain", "chain-only", "", 1, "chain_id"},
		{"only stage", "", "stage-only", 1, "stage_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := buildChainMetadata(tc.chainID, tc.stageID)
			if len(m) != tc.wantLen {
				t.Errorf("len = %d, want %d (map: %v)", len(m), tc.wantLen, m)
			}
			if tc.wantKey != "" {
				if _, ok := m[tc.wantKey]; !ok {
					t.Errorf("expected key %q to be set", tc.wantKey)
				}
			}
		})
	}
}
