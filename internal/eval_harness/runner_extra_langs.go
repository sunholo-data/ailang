package eval_harness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// JSRunner executes JavaScript (Node.js) code
type JSRunner struct {
	spec *BenchmarkSpec
}

// NewJSRunner creates a new JavaScript runner
func NewJSRunner() *JSRunner { return &JSRunner{} }

// NewJSRunnerWithSpec creates a new JavaScript runner with benchmark spec
func NewJSRunnerWithSpec(spec *BenchmarkSpec) *JSRunner { return &JSRunner{spec: spec} }

// Language returns "javascript"
func (r *JSRunner) Language() string { return "javascript" }

// Run executes JavaScript code via node
func (r *JSRunner) Run(code string, timeout time.Duration) (*RunResult, error) {
	tmpDir, err := os.MkdirTemp("", "eval_js_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if r.spec != nil {
		for name, content := range r.spec.InputFiles {
			fpath := filepath.Join(tmpDir, name)
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return nil, fmt.Errorf("failed to create dir for input file %s: %w", name, err)
			}
			if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("failed to write input file %s: %w", name, err)
			}
		}
	}

	tmpFile := filepath.Join(tmpDir, "solution.js")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code: %w", err)
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		return &RunResult{
			Stderr:    "node not found in PATH: " + err.Error(),
			ExitCode:  -1,
			CompileOk: false,
			RuntimeOk: false,
		}, nil
	}

	cmdArgs := []string{tmpFile}
	if r.spec != nil {
		cmdArgs = append(cmdArgs, r.spec.CliArgs...)
	}

	return runSubprocess(nodePath, cmdArgs, tmpDir, r.spec, timeout, "JavaScript")
}

// GoRunner executes Go code via go run
type GoRunner struct {
	spec *BenchmarkSpec
}

// NewGoRunner creates a new Go runner
func NewGoRunner() *GoRunner { return &GoRunner{} }

// NewGoRunnerWithSpec creates a new Go runner with benchmark spec
func NewGoRunnerWithSpec(spec *BenchmarkSpec) *GoRunner { return &GoRunner{spec: spec} }

// Language returns "go"
func (r *GoRunner) Language() string { return "go" }

// Run executes Go code via go run
func (r *GoRunner) Run(code string, timeout time.Duration) (*RunResult, error) {
	tmpDir, err := os.MkdirTemp("", "eval_go_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if r.spec != nil {
		for name, content := range r.spec.InputFiles {
			fpath := filepath.Join(tmpDir, name)
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return nil, fmt.Errorf("failed to create dir for input file %s: %w", name, err)
			}
			if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("failed to write input file %s: %w", name, err)
			}
		}
	}

	// Ensure code starts with package main
	src := code
	if len(src) > 0 && src[:7] != "package" {
		src = "package main\n\n" + src
	}

	tmpFile := filepath.Join(tmpDir, "solution.go")
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code: %w", err)
	}

	goPath, err := exec.LookPath("go")
	if err != nil {
		return &RunResult{
			Stderr:    "go not found in PATH: " + err.Error(),
			ExitCode:  -1,
			CompileOk: false,
			RuntimeOk: false,
		}, nil
	}

	cmdArgs := []string{"run", tmpFile}
	if r.spec != nil {
		cmdArgs = append(cmdArgs, r.spec.CliArgs...)
	}

	return runSubprocess(goPath, cmdArgs, tmpDir, r.spec, timeout, "Go")
}

// MoonbitRunner executes MoonBit code via `moon run <file>.mbt`.
// MoonBit's `moon` CLI accepts a single .mbt file directly without requiring
// a full project scaffold (moon.mod.json / moon.pkg) — this is the path the
// eval harness uses for single-file solutions.
type MoonbitRunner struct {
	spec *BenchmarkSpec
}

// NewMoonbitRunner creates a new MoonBit runner.
func NewMoonbitRunner() *MoonbitRunner { return &MoonbitRunner{} }

// NewMoonbitRunnerWithSpec creates a new MoonBit runner with benchmark spec.
func NewMoonbitRunnerWithSpec(spec *BenchmarkSpec) *MoonbitRunner { return &MoonbitRunner{spec: spec} }

// Language returns "moonbit".
func (r *MoonbitRunner) Language() string { return "moonbit" }

// Run executes MoonBit code via `moon run`.
func (r *MoonbitRunner) Run(code string, timeout time.Duration) (*RunResult, error) {
	tmpDir, err := os.MkdirTemp("", "eval_mbt_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if r.spec != nil {
		for name, content := range r.spec.InputFiles {
			fpath := filepath.Join(tmpDir, name)
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return nil, fmt.Errorf("failed to create dir for input file %s: %w", name, err)
			}
			if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("failed to write input file %s: %w", name, err)
			}
		}
	}

	tmpFile := filepath.Join(tmpDir, "solution.mbt")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code: %w", err)
	}

	cmdArgs := []string{tmpFile}
	if r.spec != nil {
		cmdArgs = append(cmdArgs, r.spec.CliArgs...)
	}

	start := time.Now()
	cmd, err := newMoonbitCommand(cmdArgs...)
	if err != nil {
		return &RunResult{
			Stderr:    err.Error(),
			ExitCode:  -1,
			Duration:  time.Since(start),
			CompileOk: false,
			RuntimeOk: false,
		}, nil
	}
	cmd.Dir = tmpDir

	if r.spec != nil && r.spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(r.spec.Stdin)
	}

	res := runGuarded(cmd, timeout, "execution timed out")
	// `moon run` handles compile+execute in one step. We can't trivially
	// distinguish compile errors from runtime errors without parsing stderr,
	// so we mark CompileOk=true when the process exits with a recognised
	// status; downstream error categorisation in errors.go classifies the
	// failure mode from stderr patterns. (Timeout/memkill stderr is
	// harness-written, so the "error:" scan only applies to real runs; the
	// res.CompileOk gate keeps a start-failure's CompileOk=false intact.)
	if !res.TimedOut && !res.MemExceeded && res.ExitCode != 0 && res.CompileOk {
		res.CompileOk = !strings.Contains(res.Stderr, "error:")
	}
	return res, nil
}

// AverRunner executes Aver code via `aver run <file>.av`.
// Aver's `aver` CLI accepts a single .av file directly; entry point is `main`.
type AverRunner struct {
	spec *BenchmarkSpec
}

// NewAverRunner creates a new Aver runner.
func NewAverRunner() *AverRunner { return &AverRunner{} }

// NewAverRunnerWithSpec creates a new Aver runner with benchmark spec.
func NewAverRunnerWithSpec(spec *BenchmarkSpec) *AverRunner { return &AverRunner{spec: spec} }

// Language returns "aver".
func (r *AverRunner) Language() string { return "aver" }

// Run executes Aver code via `aver run`.
func (r *AverRunner) Run(code string, timeout time.Duration) (*RunResult, error) {
	tmpDir, err := os.MkdirTemp("", "eval_av_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if r.spec != nil {
		for name, content := range r.spec.InputFiles {
			fpath := filepath.Join(tmpDir, name)
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return nil, fmt.Errorf("failed to create dir for input file %s: %w", name, err)
			}
			if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("failed to write input file %s: %w", name, err)
			}
		}
	}

	tmpFile := filepath.Join(tmpDir, "solution.av")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code: %w", err)
	}

	cmdArgs := []string{tmpFile}
	if r.spec != nil && len(r.spec.CliArgs) > 0 {
		cmdArgs = append(cmdArgs, "--")
		cmdArgs = append(cmdArgs, r.spec.CliArgs...)
	}

	start := time.Now()
	cmd, err := newAverCommand(cmdArgs...)
	if err != nil {
		return &RunResult{
			Stderr:    err.Error(),
			ExitCode:  -1,
			Duration:  time.Since(start),
			CompileOk: false,
			RuntimeOk: false,
		}, nil
	}
	cmd.Dir = tmpDir

	if r.spec != nil && r.spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(r.spec.Stdin)
	}

	res := runGuarded(cmd, timeout, "execution timed out")
	// Timeout/memkill stderr is harness-written, so the compile-error scan and
	// diagnostics enrichment below only apply to real runs; the res.CompileOk
	// gate keeps a start-failure's CompileOk=false intact.
	if !res.TimedOut && !res.MemExceeded && res.ExitCode != 0 && res.CompileOk {
		// `aver run` reports compile-class failures (parse/type errors, undeclared
		// effects, non-exhaustive matches) as a terse one-line `error[...]` on stderr.
		compileErr := strings.Contains(res.Stderr, "error[")
		// On a compile-class failure, re-run via `aver check` and surface its
		// structured diagnostics (named categories, `repair:` hints, source
		// excerpts) instead — far better retry feedback for the LLM. `aver run`
		// stays authoritative for pass/fail: `aver check` additionally requires a
		// `module` declaration that `run` does not, so gating on it would fail
		// runnable solutions. We only enrich the message, never change the verdict.
		// See sunholo-data/ailang#241.
		if compileErr {
			res.CompileOk = false
			if diag := averCheckDiagnostics(tmpFile, tmpDir, timeout); diag != "" {
				res.Stderr = diag
			}
		}
	}
	return res, nil
}

// averCheckDiagnostics runs `aver check <file>` and returns its rich diagnostic
// text (named error categories, `repair:` hints, source-line excerpts), which
// Aver writes to stdout. It is best-effort feedback enrichment for the LLM retry
// loop — any harness-level failure (binary missing, timeout, start error) yields
// "" so the caller falls back to the original `aver run` output. It never
// influences the pass/fail verdict. See sunholo-data/ailang#241.
func averCheckDiagnostics(file, workDir string, timeout time.Duration) string {
	cmd, err := newAverCheckCommand(file)
	if err != nil {
		return ""
	}
	cmd.Dir = workDir
	SetProcessGroup(cmd)

	stdout := NewLimitedWriter(MaxOutputSize)
	stderr := NewLimitedWriter(MaxOutputSize)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return ""
	}

	// Best-effort enrichment: guard with the same timeout + memory watchdog as
	// the main run (the type checker also processes adversarial model source),
	// and fall back to the original `aver run` output on any guard kill.
	maxRSS, rssErr := evalMaxRSS()
	if rssErr != nil {
		maxRSS = 0 // main run already failed loudly on the bad env value
	}
	g := waitWithGuards(cmd, timeout, maxRSS)
	if g.timedOut || g.memKilled {
		return ""
	}
	// `aver check` writes diagnostics to stdout; stderr is usually empty but
	// we append it for safety.
	diag := strings.TrimSpace(stdout.String())
	if e := strings.TrimSpace(stderr.String()); e != "" {
		if diag != "" {
			diag += "\n"
		}
		diag += e
	}
	return diag
}

// runSubprocess is a shared helper for JSRunner and GoRunner.
// It runs binary with the given args under the shared guards (process group,
// output limits, wall-clock timeout, memory watchdog).
func runSubprocess(binary string, args []string, workDir string, spec *BenchmarkSpec, timeout time.Duration, langLabel string) (*RunResult, error) {
	cmd := exec.Command(binary, args...)
	cmd.Dir = workDir

	if spec != nil && spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(spec.Stdin)
	}

	return runGuarded(cmd, timeout, langLabel+" execution timed out"), nil
}
