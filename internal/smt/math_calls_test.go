package smt_test

import (
	"os"
	"path/filepath"
	"testing"
)

// std/math in the verifier: floor/ceil/round/abs/PI defined exactly;
// sqrt/exp/pow/log/sin uninterpreted with per-call IEEE-true facts. Every
// caller used to be skipped (the _math_* builtins had no encoding).
func TestStdMathCallsVerify(t *testing.T) {
	path := filepath.Join("testdata", "stdlib", "math_calls.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{
		"dist": "verified", "growth": "verified", "powOne": "verified", "circ": "verified",
		"roundHalf": "verified", "roundNeg": "verified", "fl": "verified", "ce": "verified",
		"ab": "verified", "wave": "verified", "lg": "verified",
		// false (sqrt(0) is 0), but a model over an uninterpreted sqrt is not a
		// counterexample by itself: inconclusive here, confirmed by witness
		// replay in the CLI (TestVerify_WitnessReplay covers replay)
		"distBad": "inconclusive",
		// unprovable: sqrt is never given a VALUE (sqrt(2)^2 != 2 at runtime)
		"sqrtExactBad": "inconclusive",
	}
	got := map[string]string{}
	for _, r := range report.Results {
		got[r.Function] = r.Status
	}
	for fn, w := range want {
		if got[fn] != w {
			t.Errorf("%s: status %q, want %q", fn, got[fn], w)
		}
	}
}
