package smt_test

import (
	"os"
	"path/filepath"
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
