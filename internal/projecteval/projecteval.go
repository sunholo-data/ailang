// Package projecteval grades multi-file AILANG projects — the build/behaviour dimensions
// of the north-star falsification test (M-AILANG-NATIVE-HARNESS measurement frontier).
//
// A single-file benchmark is graded by stdout match; a PROJECT needs structural grading:
//
//	BuildOk    — `ailang check --package <dir>`: every module typechecks (parsed from the
//	             "X passed, Y failed" summary — robust to the --package exit code).
//	AcceptOk   — an acceptance command (the spec's test) exits 0.
//
// (iface-compare and contract verification are later dimensions.) This is where probe #3,
// semantic-edit, and the navigation tools get their real test on multi-file code.
package projecteval

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Result is a graded project outcome.
type Result struct {
	BuildOk  bool   // all modules typecheck (ailang check --package)
	AcceptOk bool   // acceptance command passed (only meaningful if an acceptance cmd was run)
	Passed   int    // modules passed (from check --package summary)
	Failed   int    // modules failed
	Detail   string // first error snippet
}

// CheckRunner runs the package check for a project dir and returns combined output + error.
// Injectable so grading logic is unit-testable without invoking the compiler.
type CheckRunner func(dir string) (string, error)

// Real `ailang check --package` summary formats (verified 2026-06-20 against a locked package):
//
//	failure: "✗ 2 files checked: 1 passed, 1 failed"
//	clean:   "✓ 2 files checked, all passed!"
//	single:  "✓ No errors found!"
var (
	passFailRe  = regexp.MustCompile(`(\d+)\s+passed,\s+(\d+)\s+failed`)
	allPassedRe = regexp.MustCompile(`(\d+)\s+files?\s+checked,\s+all passed`)
)

// parseCheck extracts (passed, failed) from a `ailang check --package` summary. ok=false when
// no recognised summary is present (caller falls back to clean/single-file detection).
func parseCheck(out string) (passed, failed int, ok bool) {
	if m := passFailRe.FindStringSubmatch(out); m != nil {
		passed, _ = strconv.Atoi(m[1])
		failed, _ = strconv.Atoi(m[2])
		return passed, failed, true
	}
	if m := allPassedRe.FindStringSubmatch(out); m != nil {
		passed, _ = strconv.Atoi(m[1])
		return passed, 0, true
	}
	return 0, 0, false
}

// GradeBuild grades the build dimension. Build is OK iff the summary reports zero failures
// (and at least one module passed), OR — when there's no package summary — the clean
// single-file "No errors found" signal is present. We parse output rather than trust the exit
// code (unreliable for --package).
func GradeBuild(dir string, run CheckRunner) Result {
	out, _ := run(dir)
	if passed, failed, ok := parseCheck(out); ok {
		return Result{BuildOk: failed == 0 && passed > 0, Passed: passed, Failed: failed, Detail: firstError(out)}
	}
	buildOk := strings.Contains(out, "No errors found") || strings.Contains(out, "✓ No errors")
	p := 0
	if buildOk {
		p = 1
	}
	return Result{BuildOk: buildOk, Passed: p, Detail: firstError(out)}
}

// GradeProject grades build, then (if it builds and an acceptance command is given) behaviour.
func GradeProject(dir string, run CheckRunner, acceptance func(dir string) bool) Result {
	r := GradeBuild(dir, run)
	if r.BuildOk && acceptance != nil {
		r.AcceptOk = acceptance(dir)
	}
	return r
}

// ProjectTask is one project-eval case: a baseline project, the task instruction, and the
// build + acceptance graders.
type ProjectTask struct {
	Dir        string                // baseline project dir (the fixture)
	Prompt     string                // task instruction (e.g. from TASK.md)
	Check      CheckRunner           // build grader (defaults to AilangCheckRunner)
	Acceptance func(dir string) bool // behaviour grader
}

// Harness runs an agent on a project WORKSPACE with a task prompt, editing files in place.
// Injectable: a stub for tests; the real impl shells the motoko/pi executor (the rig step).
type Harness func(workspace, prompt string) error

// RunProjectEval copies the baseline to a fresh temp workspace, runs the harness on it, then grades
// the workspace (build + acceptance). Copying isolates the run so the fixture is never mutated and
// every trial starts from the same baseline — the project analogue of the single-file eval flow.
func RunProjectEval(task ProjectTask, h Harness) (Result, error) {
	ws, err := copyProject(task.Dir)
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(ws)
	if err := h(ws, task.Prompt); err != nil {
		return Result{}, err
	}
	check := task.Check
	if check == nil {
		check = AilangCheckRunner("", 0)
	}
	return GradeProject(ws, check, task.Acceptance), nil
}

// copyProject recursively copies a project dir into a fresh temp dir (modules, ailang.toml, lock).
func copyProject(src string) (string, error) {
	dst, err := os.MkdirTemp("", "projecteval-*")
	if err != nil {
		return "", err
	}
	walkErr := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		info, _ := d.Info()
		mode := os.FileMode(0o644)
		if info != nil {
			mode = info.Mode()
		}
		return os.WriteFile(target, data, mode)
	})
	if walkErr != nil {
		_ = os.RemoveAll(dst)
		return "", walkErr
	}
	return dst, nil
}

// AcceptanceSpec configures running a project's entrypoint and comparing its stdout.
type AcceptanceSpec struct {
	Bin       string        // ailang binary (default "ailang")
	EntryFile string        // entrypoint file, relative to the project dir (e.g. "main.ail")
	Entry     string        // entrypoint function (default "main")
	Caps      string        // capabilities (default "IO")
	Expected  string        // expected stdout (trimmed compare)
	Timeout   time.Duration // run budget (default 30s)
}

// StdoutAcceptance returns an acceptance func that runs the project entrypoint (from the project
// dir, with --relax-modules so a package module path doesn't block single-file run) and reports
// whether trimmed stdout matches Expected. The behaviour dimension of project grading.
func StdoutAcceptance(spec AcceptanceSpec) func(dir string) bool {
	bin := spec.Bin
	if bin == "" {
		bin = "ailang"
	}
	entry := spec.Entry
	if entry == "" {
		entry = "main"
	}
	caps := spec.Caps
	if caps == "" {
		caps = "IO"
	}
	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return func(dir string) bool {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		// --quiet suppresses the "✓ Running …" status line so stdout is the program's output only.
		cmd := exec.CommandContext(ctx, bin, "run", "--entry", entry, "--caps", caps, "--relax-modules", "--quiet", spec.EntryFile)
		cmd.Dir = dir
		out, err := cmd.Output() // stdout only (status/warnings go to stderr)
		if err != nil {
			return false
		}
		return strings.TrimSpace(string(out)) == strings.TrimSpace(spec.Expected)
	}
}

// AilangCheckRunner returns a CheckRunner that shells `ailang check --package <dir>`.
func AilangCheckRunner(bin string, timeout time.Duration) CheckRunner {
	if bin == "" {
		bin = "ailang"
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return func(dir string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		// `check --package` is CWD-sensitive: intra-package import + lock resolution needs to run
		// FROM the project dir (verified 2026-06-20), so set cmd.Dir and check "." rather than
		// passing the dir as a path argument.
		cmd := exec.CommandContext(ctx, bin, "check", "--package", ".")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
}

func firstError(out string) string {
	for _, line := range strings.Split(out, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "•") || strings.Contains(s, "error") || strings.Contains(s, "Error") {
			if len(s) > 200 {
				return s[:200]
			}
			return s
		}
	}
	if len(out) > 200 {
		return out[:200]
	}
	return out
}
