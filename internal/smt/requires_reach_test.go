package smt_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// M0 (M-VERIFY-SOUNDNESS-COVERAGE): requires_callees measures the population
// decision D1 could change — results whose function reaches a callee with a
// requires the verifier assumes away (S1). It must see direct calls, calls
// through an inlined helper, and calls inside contracts, and nothing else.
func TestRequiresCalleesIsTheD1Population(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "requires_reach"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	src, err := os.ReadFile("rr.ail")
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, "rr.ail", string(src))
	got := map[string][]string{}
	for _, r := range report.Results {
		got[r.Function] = r.RequiresCallees
	}
	want := map[string][]string{
		"half":       nil,
		"double":     nil,
		"direct":     {"half"},
		"indirect":   {"half"},
		"plain":      nil,
		"inContract": {"half"},
	}
	for fn, w := range want {
		g, ok := got[fn]
		if !ok {
			t.Errorf("%s: no result", fn)
			continue
		}
		if !slices.Equal(g, w) {
			t.Errorf("%s: requires_callees = %v, want %v", fn, g, w)
		}
	}
}
