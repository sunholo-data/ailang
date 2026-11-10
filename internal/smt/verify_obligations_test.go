package smt

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/verification"
)

// TestCountObligations counts proof obligations without a solver — the number
// that turns "Z3 missing" into the `unavailable` lane (M-V1-VERIFICATION-
// HONESTY M1). A requires-only function is NOT one (review fix 2): the
// verifier assumes a precondition and never checks it at call sites, so it has
// nothing to prove there, and the verdict must not depend on whether Z3 is
// installed. Red mutation: count any contract — "requires" makes it 2.
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
	if got := CountObligations(prog); got != 1 {
		t.Fatalf("CountObligations = %d, want 1 (only the function with an ensures)", got)
	}
}

// TestTallyResults_RowsToTally: the exit tally is computed from the result
// rows; uncontracted and requires-only rows are evidence, not obligations.
// Red mutation: make IsObligation return true for every row — the
// requires-only skip counts and Skipped becomes 2.
func TestTallyResults_RowsToTally(t *testing.T) {
	rows := []VerifyResult{
		{Function: "a", Status: "verified"},
		{Function: "b", Status: "counterexample"},
		{Function: "c", Status: "inconclusive"},
		{Function: "d", Status: "skipped", ReasonCode: verification.UnsupportedEncoding},
		{Function: "e", Status: "error"},
		{Function: "f", Status: "unknown"},
		{Function: "req", Status: "skipped", ReasonCode: verification.MissingPostcondition},
		{Function: "main", Status: "uncontracted", ReasonCode: verification.Uncontracted},
	}
	got := TallyResults(rows)
	want := verification.Tally{CheckPassed: true, SolverAvailable: true, Obligations: 6,
		Verified: 1, Counterexample: 1, Inconclusive: 1, Skipped: 1, Errors: 2}
	if got != want {
		t.Fatalf("TallyResults = %+v, want %+v", got, want)
	}
	r := &VerifyReport{Results: rows}
	if r.Tally() != want || r.Obligations() != 6 {
		t.Fatalf("report Tally/Obligations disagree with TallyResults: %+v / %d", r.Tally(), r.Obligations())
	}
}
