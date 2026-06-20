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
	"os/exec"
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

var passFailRe = regexp.MustCompile(`(\d+)\s+passed,\s+(\d+)\s+failed`)

// parseCheck extracts (passed, failed) from `ailang check --package` output. ok=false when
// no summary line is present (single-file "✓ No errors found!" path).
func parseCheck(out string) (passed, failed int, ok bool) {
	m := passFailRe.FindStringSubmatch(out)
	if m == nil {
		return 0, 0, false
	}
	passed, _ = strconv.Atoi(m[1])
	failed, _ = strconv.Atoi(m[2])
	return passed, failed, true
}

// GradeBuild grades the build dimension. Build is OK iff the summary reports zero failures
// (and at least one module passed), OR — when there's no summary — the clean "No errors"
// signal is present. We parse output rather than trust the exit code (unreliable for --package).
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
		out, err := exec.CommandContext(ctx, bin, "check", "--package", dir).CombinedOutput()
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
