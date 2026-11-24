package smt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// std/list's defaulted accessors (nthOr/nth_or, headOr/head_or, lastOr/last_or)
// are generic, so the callee-sort gate used to skip every caller. They are now
// encoded directly with the runtime's semantics over any element sort — full
// proofs, not bounded ones.
func TestListDefaultedAccessorsVerify(t *testing.T) {
	path := filepath.Join("testdata", "stdlib", "list_defaulted.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{
		"safeAt": "verified", "firstName": "verified", "lastWord": "verified", "outOfRange": "verified",
		"safeAtBad": "counterexample",
	}
	got := map[string]string{}
	for _, r := range report.Results {
		got[r.Function] = r.Status
		if r.BoundedDepth != 0 {
			t.Errorf("%s: a defaulted accessor needs no unrolling, got bounded depth %d", r.Function, r.BoundedDepth)
		}
	}
	for fn, w := range want {
		if got[fn] != w {
			t.Errorf("%s: status %q, want %q", fn, got[fn], w)
		}
	}
}

// std/list head/nth/last (Option-returning) and std/option getOrElse/isSome/
// isNone are encoded directly over the Option datatype, which is declared on
// demand even when no signature mentions Option.
func TestOptionAccessorsVerify(t *testing.T) {
	path := filepath.Join("testdata", "stdlib", "option_access.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{
		"firstOr": "verified", "hasThird": "verified", "noNeg": "verified", "lastOpt": "verified",
		"firstOrBad": "counterexample", "lastOptBad": "counterexample",
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

// take/drop were listed as encodable but had no encoding on any reachable path
// (only an unused unrolling generator), so every caller failed. They are now
// exact seq.extract terms with the runtime's edge cases (n < 0, n > length).
func TestTakeDropVerify(t *testing.T) {
	path := filepath.Join("testdata", "stdlib", "take_drop.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{"t1": "verified", "t2": "verified", "t3": "verified", "tBad": "counterexample"}
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

// std/list.concat is (seq.++ xs ys) over any element sort. With std/bytes also
// loaded (it defines its own concat) the call is still the list one.
func TestConcatVerify(t *testing.T) {
	path := filepath.Join("testdata", "stdlib", "concat.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{"c1": "verified", "c2": "verified", "c3": "verified", "cBad": "counterexample"}
	got := map[string]string{}
	for _, r := range report.Results {
		got[r.Function] = r.Status + " " + r.Reason
	}
	for fn, w := range want {
		if !strings.HasPrefix(got[fn], w+" ") {
			t.Errorf("%s: %q, want %q", fn, got[fn], w)
		}
	}
}
