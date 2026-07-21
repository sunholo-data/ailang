// agent_validation.go - Solution validation + grading for agent-mode benchmarks.
// Split from agent_runner_multi.go (check-file-sizes: >800 lines) during
// M-EVAL-MEM-GUARD. All model-code execution here goes through the guarded
// helpers in memlimit.go.

package eval_harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// validateSolution checks if the solution is correct
// seedInputFiles writes spec.InputFiles into the agent workspace root, mirroring
// the layout the standard runners use (runner.go:138/320). Without this, an agent
// working on a benchmark that reads a data file (e.g. cli_args reads numbers.txt)
// cannot test its own solution — the file does not exist in its workspace — so it
// submits blind and the task looks far harder in agent mode than in standard mode.
// Files land at the workspace root, which is the agent's cwd AND the cwd the
// AILANG/Python validators run from, so relative reads resolve identically during
// the agent's iteration and during grading.
func seedInputFiles(workspace string, spec *BenchmarkSpec) error {
	if spec == nil {
		return nil
	}
	for name, content := range spec.InputFiles {
		fpath := filepath.Join(workspace, name)
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return fmt.Errorf("failed to create dir for input file %s: %w", name, err)
		}
		if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write input file %s: %w", name, err)
		}
	}
	return nil
}

func validateSolution(result *executor.Result, spec *BenchmarkSpec, workspace, language, solutionPath string) ValidationResult {
	if !result.Success {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    result.Error,
		}
	}

	// M-EVAL-RELIABLE-GRADING: multi-file reimplement benchmarks grade a harness-owned probe
	// against the agent's PRESERVED workspace (with the modules it implemented), instead of
	// re-stubbing a fresh workspace and running solution.ail in isolation — which discards the
	// agent's implementation and only passes if it ignored the task and inlined everything.
	if language == "ailang" && spec.GradeEntrypoint != "" {
		return gradeInWorkspace(spec, workspace)
	}

	// Check if solution file exists
	solutionContent, err := os.ReadFile(solutionPath)
	if err != nil || len(solutionContent) == 0 {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    fmt.Sprintf("Solution file not found or empty: %v", err),
		}
	}

	// Run solution based on language via the runner registry.
	if language == "python" {
		return runPythonSolution(solutionPath, spec)
	}
	if language == "ailang" {
		return runAILANGSolution(string(solutionContent), spec)
	}

	// JS, Go, and future languages: use GetRunnerWithContext.
	r, runErr := GetRunnerWithContext(context.Background(), language, spec, "")
	if runErr != nil {
		return ValidationResult{Stderr: fmt.Sprintf("no runner for %q: %v", language, runErr)}
	}
	runResult, runRunErr := r.Run(string(solutionContent), 30*time.Second)
	if runRunErr != nil {
		return ValidationResult{Stderr: fmt.Sprintf("validation runner error: %v", runRunErr)}
	}
	stdoutOk := runResult.RuntimeOk && GradeStdout(spec, runResult.Stdout, string(solutionContent))
	return ValidationResult{
		CompileOk: runResult.CompileOk,
		RuntimeOk: runResult.RuntimeOk,
		StdoutOk:  stdoutOk,
		Stdout:    runResult.Stdout,
		Stderr:    runResult.Stderr,
	}
}

// runPythonSolution executes and validates a Python solution using the
// uv-managed pinned Python runtime.
//
// Wires spec.CliArgs into the subprocess invocation and spec.Stdin onto
// the subprocess's stdin. Without these, every benchmark whose Python
// solution reads sys.argv[1] or sys.stdin will fail at runtime even
// though the generated code is correct — see
// design_docs/planned/v0_21_0/m-eval-agent-python-stdio-wiring.md for the
// full context (v0.20.0 cli_args + pipeline benchmarks exposed this).
func runPythonSolution(solutionPath string, spec *BenchmarkSpec) ValidationResult {
	// Run from the solution's directory and seed input files there so relative
	// cli_args / input_files (e.g. cli_args reads numbers.txt) resolve the same
	// way they do for the agent and the standard runner. newPythonCommand sets no
	// cwd, so without cmd.Dir the args would resolve against the harness process
	// cwd (the repo root) and a correct solution would fail validation.
	workDir := filepath.Dir(solutionPath)
	if err := seedInputFiles(workDir, spec); err != nil {
		return ValidationResult{Stderr: err.Error()}
	}

	// newPythonCommand is variadic — pass solution path + spec's CLI args
	// in the same slice so they all land after `uv run --python 3.12 --`.
	args := append([]string{solutionPath}, spec.CliArgs...)
	cmd, uvErr := newPythonCommand(args...)
	if uvErr != nil {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    uvErr.Error(),
		}
	}
	cmd.Dir = workDir
	if spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(spec.Stdin)
	}
	// Run under the shared guards. This lane previously used CombinedOutput()
	// with NO timeout, NO process group, and NO memory cap — an unbounded
	// execution of model code (the class of hole behind the 2026-07-20 rig
	// kernel panic). 30s matches the JS/Go validation lane above.
	res := runGuarded(cmd, 30*time.Second, "Python validation timed out")
	if res.ExitCode != 0 {
		return ValidationResult{
			CompileOk: true,
			RuntimeOk: false,
			StdoutOk:  false,
			Stdout:    res.Stdout,
			Stderr:    res.Stderr,
		}
	}
	// Source not needed here: quine grading is AILANG standard-mode only; other
	// modes (default, prefix_line) ignore the submitted source.
	stdoutOk := GradeStdout(spec, res.Stdout, "")
	return ValidationResult{
		CompileOk: true,
		RuntimeOk: true,
		StdoutOk:  stdoutOk,
		Stdout:    res.Stdout,
	}
}

// runAILANGSolution executes and validates an AILANG solution
func runAILANGSolution(solutionCode string, spec *BenchmarkSpec) ValidationResult {
	// Pass spec to runner so input_files, cli_args, and stdin are available
	// in the validation workspace (not just in the agent workspace).
	runner := NewAILANGRunnerWithTask(context.Background(), "", spec.Caps, "", spec)
	runResult, err := runner.Run(solutionCode, 10*time.Second)
	if err != nil {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    fmt.Sprintf("validation runner error: %v", err),
		}
	}

	stdoutOk := runResult.RuntimeOk && GradeStdout(spec, runResult.Stdout, solutionCode)
	return ValidationResult{
		CompileOk: runResult.CompileOk,
		RuntimeOk: runResult.RuntimeOk,
		StdoutOk:  stdoutOk,
		Stdout:    runResult.Stdout,
		Stderr:    runResult.Stderr,
	}
}

// gradeInWorkspace (M-EVAL-RELIABLE-GRADING) grades a multi-file benchmark by running the
// harness-owned probe (spec.GradeEntrypoint) against the modules the agent implemented in its
// PRESERVED workspace. Every input_file EXCEPT spec.SolutionFiles is re-seeded to its canonical
// form first, so the probe and fixed dependencies are tamper-proof while the agent's target
// implementation is the thing actually measured. This replaces the legacy path that re-stubbed a
// fresh workspace and ran solution.ail in isolation — discarding the agent's implementation.
func gradeInWorkspace(spec *BenchmarkSpec, workspace string) ValidationResult {
	keep := make(map[string]bool, len(spec.SolutionFiles))
	for _, f := range spec.SolutionFiles {
		keep[f] = true
	}
	// Restore canonical probe + fixed deps; preserve the agent's solution_files.
	for name, content := range spec.InputFiles {
		if keep[name] {
			continue
		}
		fpath := filepath.Join(workspace, name)
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return ValidationResult{Stderr: fmt.Sprintf("[harness_setup] grade: mkdir %s: %v", name, err)}
		}
		if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
			return ValidationResult{Stderr: fmt.Sprintf("[harness_setup] grade: seed %s: %v", name, err)}
		}
	}

	// Run the probe from the agent's workspace, mirroring the legacy runner's flags.
	ailangBin := os.Getenv("AILANG_BIN")
	if ailangBin == "" {
		ailangBin = "ailang" // PATH
	}
	cwd, _ := os.Getwd()
	args := []string{"run", "--entry", "main", "--quiet", "--relax-modules",
		"--stdlib-path", filepath.Join(cwd, "std")}
	if len(spec.Caps) > 0 {
		args = append(args, "--caps", strings.Join(spec.Caps, ","))
	}
	args = append(args, spec.GradeEntrypoint)

	// Run under the shared guards (process group, output limits, timeout,
	// memory watchdog) — the probe executes the MODEL's modules, so it is a
	// generated-code execution path like any other.
	cmd := exec.Command(ailangBin, args...)
	cmd.Dir = workspace
	res := runGuarded(cmd, 30*time.Second, "grade probe timed out")
	stdout, stderr := res.Stdout, res.Stderr

	// Classify. The probe is harness-owned and always valid, so a failure here is attributable to
	// the model (a compile error in its module, or wrong output) — never a missing-entrypoint
	// artifact, which was the legacy path's flaw.
	runtimeOk := res.ExitCode == 0
	combined := stdout + "\n" + stderr
	compileOk := !strings.Contains(combined, "type error") &&
		!strings.Contains(combined, "parse error") &&
		!strings.Contains(combined, "entrypoint 'main' not found")
	// Multi-file workspace grading: no single submitted source (quine N/A here).
	stdoutOk := runtimeOk && GradeStdout(spec, stdout, "")
	return ValidationResult{
		CompileOk: compileOk,
		RuntimeOk: runtimeOk,
		StdoutOk:  stdoutOk,
		Stdout:    stdout,
		Stderr:    stderr,
	}
}
