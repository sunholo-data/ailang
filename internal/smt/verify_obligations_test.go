package smt

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/verification"
)

// TestCountObligations counts contract-bearing functions without a solver —
// the number that turns "Z3 missing" into the `unavailable` lane
// (M-V1-VERIFICATION-HONESTY M1). requires-only counts: the verifier visits it
// (and skips it as "nothing to verify"), so a run without Z3 must not claim
// there was nothing to do.
func TestCountObligations(t *testing.T) {
	if got := CountObligations(nil); got != 0 {
		t.Fatalf("CountObligations(nil) = %d, want 0", got)
	}
	prog := &core.Program{Meta: map[string]*core.DeclMeta{
		"plain":    {Name: "plain"},
		"ensures":  {Name: "ensures", Contracts: []*core.Contract{{Kind: core.EnsuresKind}}},
		"requires": {Name: "requires", Contracts: []*core.Contract{{Kind: core.RequiresKind}}},
		"nilmeta":  nil,
	}}
	if got := CountObligations(prog); got != 2 {
		t.Fatalf("CountObligations = %d, want 2 (ensures + requires-only)", got)
	}
}

// TestVerifyReport_TallyProjectsEveryCounter: the report's tally is exactly its
// counters, with uncontracted rows outside the obligation count.
func TestVerifyReport_TallyProjectsEveryCounter(t *testing.T) {
	r := &VerifyReport{Verified: 1, Counterexample: 2, Inconclusive: 3, Skipped: 4, Errors: 5, Uncontracted: 6}
	got := r.Tally()
	want := verification.Tally{CheckPassed: true, SolverAvailable: true, Obligations: 15,
		Verified: 1, Counterexample: 2, Inconclusive: 3, Skipped: 4, Errors: 5}
	if got != want {
		t.Fatalf("Tally() = %+v, want %+v", got, want)
	}
}
