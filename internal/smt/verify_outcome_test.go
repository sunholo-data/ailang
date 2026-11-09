package smt

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/verification"
)

// M-VERIFY-BOUNDED-UNROLLING-FALSE-COUNTEREXAMPLE M2 — the ONE classification
// seam. Both `ailang verify` and `ailang ai-check` run smt.Verify, so the
// status a solver outcome becomes is decided here and nowhere else.

// allSolverStatuses enumerates every SolverStatus member, bounded by the
// String() sentinel ("status(N)") so a member added to solver.go is picked up
// without editing this test (MUT-ADD-STATUS).
func allSolverStatuses(t *testing.T) []SolverStatus {
	t.Helper()
	var out []SolverStatus
	for s := SolverStatus(0); ; s++ {
		if s.String() == fmt.Sprintf("status(%d)", int(s)) {
			break
		}
		out = append(out, s)
		if s > 64 {
			t.Fatal("instrument failure: SolverStatus.String() has no sentinel default")
		}
	}
	if len(out) < 4 {
		t.Fatalf("instrument failure: enumerated only %d statuses", len(out))
	}
	return out
}

// TestSolveOutcome_BoundedSatIsInconclusive: a sat on a bounded-unrolled
// obligation is NOT a violation. Its model may rest on an invented value for
// the uninterpreted frontier stub f_0, so it is reported inconclusive — with
// the depth it is relative to and a reason — and the model is withheld unless
// the caller asked for verbose output. Kills MUT-REMOVE-BOUNDED-CHECK and
// MUT-DROP-BOUNDED-DEPTH.
func TestSolveOutcome_BoundedSatIsInconclusive(t *testing.T) {
	model := []ModelBinding{{Name: "$p_xs", Sort: "SList", Value: "(SCons 3 (SCons 2 SNil))"}}
	sat := &SolverResult{Status: StatusCounterexample, Model: model}

	vr := solverVerifyResult("sLength", sat, 2, false)
	if vr.Status != "inconclusive" {
		t.Fatalf("bounded sat status = %q, want inconclusive", vr.Status)
	}
	if vr.BoundedDepth != 2 {
		t.Fatalf("bounded_depth = %d, want 2 (the verdict is relative to it)", vr.BoundedDepth)
	}
	if vr.ReasonCode != verification.BoundedInconclusive {
		t.Fatalf("reason_code = %q, want %q", vr.ReasonCode, verification.BoundedInconclusive)
	}
	for _, want := range []string{"depth 2", "sLength_0", "requires clause bounds", "neither a proof nor a confirmed violation"} {
		if !strings.Contains(vr.Reason, want) {
			t.Fatalf("reason %q does not mention %q", vr.Reason, want)
		}
	}
	if len(vr.Model) != 0 {
		t.Fatalf("frontier-dependent model surfaced by default: %+v", vr.Model)
	}
	if v := solverVerifyResult("sLength", sat, 2, true); len(v.Model) != 1 {
		t.Fatalf("verbose must keep the model for debugging, got %+v", v.Model)
	}
}

// TestSolveOutcome_UnboundedSatIsCounterexample: with no unrolling (a
// non-recursive function, or depth 0) the obligation is fully concrete and a
// sat model is a real witness. Kills MUT-OVERSUPPRESS.
func TestSolveOutcome_UnboundedSatIsCounterexample(t *testing.T) {
	model := []ModelBinding{{Name: "$p_x", Sort: "Int", Value: "(- 1)"}}
	vr := solverVerifyResult("bBad", &SolverResult{Status: StatusCounterexample, Model: model}, 0, false)
	if vr.Status != "counterexample" || len(vr.Model) != 1 || vr.ReasonCode != verification.Counterexample {
		t.Fatalf("genuine counterexample lost: %+v", vr)
	}
}

// TestSolveOutcome_UnsatStaysVerified: unsat under havoc is sound (the real
// function is one interpretation of the stub) — the verified surface keeps its
// bounded caveat and is otherwise unchanged.
func TestSolveOutcome_UnsatStaysVerified(t *testing.T) {
	vr := solverVerifyResult("f", &SolverResult{Status: StatusVerified}, 3, false)
	if vr.Status != "verified" || vr.BoundedDepth != 3 {
		t.Fatalf("bounded unsat changed: %+v", vr)
	}
}

// TestSolveOutcomeExhaustive: every SolverStatus member, bounded or not, maps
// to a status in the allowed set and moves EXACTLY ONE report counter (the
// INV-1 partition: counters sum to len(results)). An added SolverStatus or an
// emitted status string with no counter fails here (MUT-ADD-STATUS,
// MUT-ADD-JSON-STATUS).
func TestSolveOutcomeExhaustive(t *testing.T) {
	allowed := map[string]bool{"verified": true, "counterexample": true, "inconclusive": true, "unknown": true, "error": true}
	for _, st := range allSolverStatuses(t) {
		for _, depth := range []int{0, 2} {
			vr := solverVerifyResult("f", &SolverResult{Status: st, Error: "detail"}, depth, false)
			if !allowed[vr.Status] {
				t.Fatalf("SolverStatus %v (depth %d) -> unmapped status %q", st, depth, vr.Status)
			}
			r := &VerifyReport{}
			r.tally(vr)
			sum := r.Verified + r.Counterexample + r.Inconclusive + r.Skipped + r.Errors + r.Uncontracted
			if sum != 1 || len(r.Results) != 1 {
				t.Fatalf("SolverStatus %v (depth %d) status %q moved %d counters / %d results, want 1/1", st, depth, vr.Status, sum, len(r.Results))
			}
		}
	}
}

// TestTally_EveryStatusHasOneCounter covers the non-solver statuses Verify
// emits directly, so the partition holds for every result row.
func TestTally_EveryStatusHasOneCounter(t *testing.T) {
	for _, status := range []string{"verified", "counterexample", "inconclusive", "skipped", "error", "unknown", "uncontracted"} {
		r := &VerifyReport{}
		r.tally(VerifyResult{Function: "f", Status: status})
		sum := r.Verified + r.Counterexample + r.Inconclusive + r.Skipped + r.Errors + r.Uncontracted
		if sum != 1 {
			t.Fatalf("status %q moved %d counters, want 1", status, sum)
		}
	}
}
