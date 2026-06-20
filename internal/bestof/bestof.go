// Package bestof implements best-of-N candidate selection with an EXACT verifier —
// the AILANG-native pass-rate lever (M-AILANG-NATIVE-HARNESS probe #3).
//
// qwen at temp>0 produces a DISTRIBUTION of candidate solutions. A general harness must
// submit a guess; AILANG can pick the verified-correct one. The verifier here is the
// realistic, reference-free selector: a candidate that TYPECHECKS (`ailang check`) and
// RUNS cleanly outranks one that only typechecks, which outranks one that does neither.
// (When the task ships contracts/tests, the verifier can also run those — strictly
// stronger; that lifts the few logic_error cases the typecheck+run selector can't catch.)
//
// Validated free from the 2026-06-20 h2h: realistic best-of-3 takes motoko 90.6%→97.4%
// (vs pi 88.9%→94.9%), motoko 0 hard-fails — see analysis-log + eval_best_of_n.py.
package bestof

import (
	"context"
	"os/exec"
	"time"
)

// Verdict records how a candidate fared under the verifier.
type Verdict struct {
	TypeChecks bool   // passed `ailang check`
	Runs       bool   // executed without error (implies TypeChecks)
	Detail     string // first error snippet, if any
}

// score ranks a verdict: runs(2) > typechecks-only(1) > neither(0).
func (v Verdict) score() int {
	switch {
	case v.Runs:
		return 2
	case v.TypeChecks:
		return 1
	default:
		return 0
	}
}

// Verifier checks one candidate solution file. Injectable so SelectBest is unit-testable
// without invoking the real compiler.
type Verifier interface {
	Verify(path string) Verdict
}

// SelectBest verifies every candidate and returns the index of the best one (highest
// score; ties → earliest, preserving the model's own ordering) plus all verdicts.
// Returns -1 for an empty candidate list. Never returns -1 for a non-empty list — even
// if nothing verifies, it returns the first (a guess is better than nothing), so the
// caller can fall back gracefully.
func SelectBest(paths []string, v Verifier) (int, []Verdict) {
	if len(paths) == 0 {
		return -1, nil
	}
	verdicts := make([]Verdict, len(paths))
	best, bestScore := 0, -1
	for i, p := range paths {
		verdicts[i] = v.Verify(p)
		if s := verdicts[i].score(); s > bestScore {
			best, bestScore = i, s
		}
	}
	return best, verdicts
}

// AilangVerifier is the real, reference-free selector: `ailang check` then `ailang run`.
type AilangVerifier struct {
	Bin     string        // ailang binary (default "ailang")
	Entry   string        // entrypoint (default "main")
	Caps    string        // capabilities, e.g. "IO,FS" (default "IO")
	Timeout time.Duration // per-candidate wall budget (default 30s)
}

func (a AilangVerifier) bin() string {
	if a.Bin != "" {
		return a.Bin
	}
	return "ailang"
}

func (a AilangVerifier) entry() string {
	if a.Entry != "" {
		return a.Entry
	}
	return "main"
}

func (a AilangVerifier) caps() string {
	if a.Caps != "" {
		return a.Caps
	}
	return "IO"
}

func (a AilangVerifier) timeout() time.Duration {
	if a.Timeout > 0 {
		return a.Timeout
	}
	return 30 * time.Second
}

// Verify runs `ailang check` then (if it typechecks) `ailang run` on the candidate.
func (a AilangVerifier) Verify(path string) Verdict {
	out, err := a.runCmd(a.bin(), "check", path)
	if err != nil {
		return Verdict{TypeChecks: false, Detail: snippet(out)}
	}
	runOut, runErr := a.runCmd(a.bin(), "run", "--entry", a.entry(), "--caps", a.caps(), path)
	if runErr != nil {
		return Verdict{TypeChecks: true, Runs: false, Detail: snippet(runOut)}
	}
	return Verdict{TypeChecks: true, Runs: true}
}

func (a AilangVerifier) runCmd(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout())
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

func snippet(s string) string {
	const max = 200
	if len(s) > max {
		return s[:max]
	}
	return s
}
