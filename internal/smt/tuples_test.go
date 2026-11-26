package smt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// Tuples are the built-in datatypes TupleN (tuple_sorts.go). Every function
// here was skipped before ("tuple expressions cannot be encoded"); the false
// contracts are refuted, not proved.
func TestTupleSorts(t *testing.T) {
	path := filepath.Join("testdata", "tuples", "tuples.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{
		"swap": "verified", "swapBad": "counterexample", "third": "verified",
		"emptyFirst": "verified", "classify": "verified", "classifyBad": "counterexample",
		"someSum": "verified", "firstKey": "verified", "sumPair": "verified",
		"pairEq": "verified", "pairEqBad": "counterexample", "viaSplit": "verified",
		"sameLiteral": "verified", "edgeSum": "verified",
	}
	for _, r := range report.Results {
		if w, ok := want[r.Function]; ok {
			delete(want, r.Function)
			if r.Status != w {
				t.Errorf("%s: status %q (%s), want %q", r.Function, r.Status, r.Reason, w)
			}
		}
	}
	for fn := range want {
		t.Errorf("%s: no result", fn)
	}
}

// Unit is the 0-tuple Tuple0: a Result[(), E] helper, a unit-returning helper
// and a unit parameter encode (each was skipped before).
func TestUnitSort(t *testing.T) {
	path := filepath.Join("testdata", "tuples", "unit.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Depth 2, as the CLI: zeroRec is unrolled.
	report, _ := verifySourceWith(t, path, string(src), smt.VerifyOptions{RecursiveDepth: 2})
	want := map[string]string{"stoppedCase": "verified", "stoppedBad": "counterexample", "throughNoop": "verified", "unitParamBad": "counterexample", "zeroRec": "verified"}
	for _, r := range report.Results {
		if w, ok := want[r.Function]; ok {
			delete(want, r.Function)
			if r.Status != w {
				t.Errorf("%s: status %q (%s), want %q", r.Function, r.Status, r.Reason, w)
			}
		}
	}
	for fn := range want {
		t.Errorf("%s: no result", fn)
	}
}

// A declared type named like a tuple datatype keeps its name; a function that
// needs that sort is skipped, never encoded over the wrong datatype.
func TestTupleSortShadowed(t *testing.T) {
	path := filepath.Join("testdata", "tuples", "shadow.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	got := map[string]string{}
	for _, r := range report.Results {
		got[r.Function] = r.Status
		if r.Function == "fst" && !strings.Contains(r.Reason, "shadows the verifier's datatype for 2-tuples") {
			t.Errorf("fst: reason %q does not name the shadowed tuple sort", r.Reason)
		}
		if r.Function == "unitArg" && !strings.Contains(r.Reason, "shadows the verifier's datatype for unit, ()") {
			t.Errorf("unitArg: reason %q does not name the shadowed unit sort", r.Reason)
		}
	}
	if got["fst"] != "skipped" || got["unitArg"] != "skipped" || got["plain"] != "verified" {
		t.Errorf("statuses %v, want fst and unitArg skipped, plain verified", got)
	}
}
