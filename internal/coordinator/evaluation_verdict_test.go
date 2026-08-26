package coordinator

import (
	"strings"
	"testing"
)

// M-PIPELINE-RECONCILIATION M1 (decision D1(b), ratified 2026-08-26).
//
// The verdict is a CLOSED type: PASS(score) | FAIL(score, reasons) |
// UNAVAILABLE(reason). Absence is unrepresentable by construction — evaluator
// error, timeout, or unparsable output all map to UNAVAILABLE, never to an
// empty field. This came out of quorum round 1: the first draft said "a FAIL
// does not block — it informs" while scoring itself +1 on Structured Failure,
// and left evaluator death silent.
func TestParseEvaluationVerdict_Pass(t *testing.T) {
	v := ParseEvaluationVerdict("PASS score=84")
	if v.Kind != VerdictPass {
		t.Fatalf("expected PASS, got %q", v.Kind)
	}
	if v.Score != 84 {
		t.Errorf("expected score 84, got %d", v.Score)
	}
	if v.BlocksAutomation() {
		t.Error("PASS must not block automatic progression")
	}
}

func TestParseEvaluationVerdict_Fail(t *testing.T) {
	v := ParseEvaluationVerdict("FAIL score=45 reasons=missing tests; AC3 unmet")
	if v.Kind != VerdictFail {
		t.Fatalf("expected FAIL, got %q", v.Kind)
	}
	if v.Score != 45 {
		t.Errorf("expected score 45, got %d", v.Score)
	}
	if !strings.Contains(v.Reason, "AC3 unmet") {
		t.Errorf("reasons must survive parsing, got %q", v.Reason)
	}
	if !v.BlocksAutomation() {
		t.Error("FAIL must block automatic progression")
	}
}

// The load-bearing case: garbage in yields UNAVAILABLE, not an error and not
// an empty verdict. An evaluator that emits nonsense must still produce a
// typed, visible outcome on the approval.
func TestParseEvaluationVerdict_UnparsableIsUnavailable(t *testing.T) {
	for _, raw := range []string{"", "banana", "PASS score=notanumber", "MAYBE score=50"} {
		v := ParseEvaluationVerdict(raw)
		if v.Kind != VerdictUnavailable {
			t.Errorf("ParseEvaluationVerdict(%q): expected UNAVAILABLE, got %q", raw, v.Kind)
		}
		if v.Reason == "" {
			t.Errorf("ParseEvaluationVerdict(%q): UNAVAILABLE must carry a reason", raw)
		}
		if !v.BlocksAutomation() {
			t.Errorf("ParseEvaluationVerdict(%q): UNAVAILABLE must block automatic progression", raw)
		}
	}
}

func TestEvaluationVerdict_FormatRoundTrip(t *testing.T) {
	for _, raw := range []string{"PASS score=84", "FAIL score=45 reasons=x; y", "UNAVAILABLE reason=evaluator task failed: timeout"} {
		v := ParseEvaluationVerdict(raw)
		v2 := ParseEvaluationVerdict(v.String())
		if v2.Kind != v.Kind || v2.Score != v.Score {
			t.Errorf("round-trip changed the verdict: %q -> %q -> %q", raw, v.String(), v2.String())
		}
	}
}

func TestUnavailableVerdict_Constructor(t *testing.T) {
	v := UnavailableVerdict("evaluator task failed: exit 1")
	if v.Kind != VerdictUnavailable || !strings.Contains(v.Reason, "exit 1") {
		t.Errorf("constructor must produce a reasoned UNAVAILABLE, got %+v", v)
	}
}
