package eval_harness

import (
	"encoding/json"
	"testing"
)

// M-VERIFY-BOUNDED-UNROLLING-FALSE-COUNTEREXAMPLE M1 — the harness readers.
//
// `ailang ai-check` now reports a sat found under bounded recursion unrolling
// as `inconclusive` (the solver may have invented a value for the frontier
// stub f_0, so the model is not a real input). These tests pin how the harness
// reads that counter: carried into its own field, never a violation, never a
// repair trigger.

// newEmitterJSON is what the fixed ai-check prints for a file with one proved
// obligation and one bounded-unrolling sat (cmd/ailang/testdata/verify_bounded/adt.ail).
const newEmitterJSON = `{"file":"solution.ail","check":{"passed":true,"error_count":0,"errors":[]},` +
	`"verify":{"available":true,"verified":1,"counterexample":0,"inconclusive":1,"skipped":0,"errors":0,` +
	`"results":[{"function":"sLength","status":"inconclusive","reason":"bounded unrolling exhausted at depth 2","bounded_depth":2},` +
	`{"function":"headOr0","status":"verified"}]}}`

func parseAICheck(t *testing.T, raw string) *AICheckResult {
	t.Helper()
	var r AICheckResult
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("fixture JSON does not parse: %v", err)
	}
	return &r
}

// TestPopulateVerifyMetrics_CarriesInconclusive: the reader copies the new
// counter, and an inconclusive-only report claims no violation (VerifyOk).
// Whether that run is a VERIFIED success is the observatory predicate's call
// (it requires verify_inconclusive == 0), not VerifyOk's.
func TestPopulateVerifyMetrics_CarriesInconclusive(t *testing.T) {
	m := &RunMetrics{}
	PopulateVerifyMetrics(m, parseAICheck(t, newEmitterJSON), newEmitterJSON)
	if m.VerifyInconclusive != 1 {
		t.Fatalf("VerifyInconclusive = %d, want 1", m.VerifyInconclusive)
	}
	if m.VerifyCounterex != 0 || m.VerifyVerified != 1 {
		t.Fatalf("counters mis-read: %+v", m)
	}
	if !m.VerifyOk {
		t.Fatalf("VerifyOk = false: an undecided obligation is not a violation")
	}
}

// TestApplyAgentVerification_CarriesInconclusive: the agent-mode funnel reads
// the same field, so both eval modes bank it.
func TestApplyAgentVerification_CarriesInconclusive(t *testing.T) {
	f := &fakeVerifier{result: parseAICheck(t, newEmitterJSON), raw: newEmitterJSON}
	withFakeVerifier(t, f)
	out := &AgentBenchmarkResult{}
	applyAgentVerification(out, AgentBenchmarkConfig{Verify: true}, contractSpec(), "ailang", "/tmp/s.ail", true)
	if out.VerifyInconclusive != 1 {
		t.Fatalf("VerifyInconclusive = %d, want 1", out.VerifyInconclusive)
	}
}

// TestVerifyRepairSignal_InconclusiveIsNotARepairTrigger (AC-9, kills
// MUT-REPAIR-WIDEN): an inconclusive-only report must NOT feed the model a
// VERIFY_COUNTEREXAMPLE repair turn — those "counterexample inputs" are not
// violating inputs, and the old behaviour coached models to break correct code.
func TestVerifyRepairSignal_InconclusiveIsNotARepairTrigger(t *testing.T) {
	code, hint, stderr := verifyRepairSignal(parseAICheck(t, newEmitterJSON), newEmitterJSON)
	if code != "" || hint != nil || stderr != "" {
		t.Fatalf("inconclusive-only report triggered repair: code=%q hint=%v stderr=%q", code, hint, stderr)
	}
}

// TestVerifyRepairSignal_CounterexampleStillTriggers is the positive control:
// a genuine counterexample keeps its repair turn.
func TestVerifyRepairSignal_CounterexampleStillTriggers(t *testing.T) {
	raw := `{"check":{"passed":true},"verify":{"available":true,"verified":0,"counterexample":1,` +
		`"results":[{"function":"bBad","status":"counterexample","model":[{"name":"$p_x","sort":"Int","value":"(- 1)"}]}]}}`
	code, hint, stderr := verifyRepairSignal(parseAICheck(t, raw), raw)
	if code != VERIFY_COUNTEREXAMPLE || hint == nil || stderr != raw {
		t.Fatalf("genuine counterexample lost its repair turn: code=%q hint=%v", code, hint)
	}
}
