package smt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// A self-recursive helper without a contract is unrolled to a bounded depth
// (unrollRecursiveCallee). Before, IsSMTEncodableForCallee refused every
// recursive callee, the call fell through to the leak guard, and the caller was
// skipped with "could not resolve a required sort" (21 functions in the gap map).
func TestRecursiveCalleeIsUnrolled(t *testing.T) {
	path := filepath.Join("testdata", "recursion", "recursive_callee.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySourceWith(t, path, string(src), smt.VerifyOptions{Replay: nil})
	want := map[string]string{
		"allDigits": "verified", // through allDigitsFrom, which calls isDigit
		"useCount":  "verified",
		// countUp(2, 0) == 2: a model at the bound, so inconclusive here; the
		// CLI's witness replay confirms it (verify_recursive_callee_e2e_test).
		"useCountBad": "inconclusive",
		// needs facts deeper than the unrolling: never verified
		"useCountDeep": "inconclusive",
	}
	for _, r := range report.Results {
		w, ok := want[r.Function]
		if !ok {
			continue
		}
		if r.Status != w {
			t.Errorf("%s: %s (%s), want %s", r.Function, r.Status, r.Reason, w)
		}
		if w == "verified" && r.BoundedDepth == 0 {
			t.Errorf("%s: a proof through an unrolled callee must report its bounded depth", r.Function)
		}
		if r.Function == "useCountDeep" && !strings.Contains(r.Reason, "recursive helper") {
			t.Errorf("useCountDeep: reason should name the helper's unrolling, got %q", r.Reason)
		}
	}
}
