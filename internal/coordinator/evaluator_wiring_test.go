package coordinator

import (
	"testing"
)

// M-PIPELINE-RECONCILIATION M2 (D1(b)).
//
// AutoApproveHandoffs is all-or-nothing per source agent. D1(b) needs exactly
// one edge auto-approved — executor→evaluator, justified because the evaluator
// is read-only — while every other handoff stays behind the human gate.
func TestAutoApprovesHandoffTo(t *testing.T) {
	exec := &AgentConfig{
		ID:                   "sprint-executor",
		TriggerOnComplete:    []string{"sprint-evaluator", "some-other-agent"},
		AutoApproveHandoffs:  false,
		AutoApproveHandoffTo: []string{"sprint-evaluator"},
	}
	if !exec.AutoApprovesHandoffTo("sprint-evaluator") {
		t.Error("the listed edge must auto-approve")
	}
	if exec.AutoApprovesHandoffTo("some-other-agent") {
		t.Error("an unlisted edge must NOT auto-approve while the bool is false")
	}

	// The bool still means "all edges" when set.
	all := &AgentConfig{ID: "x", AutoApproveHandoffs: true}
	if !all.AutoApprovesHandoffTo("anything") {
		t.Error("AutoApproveHandoffs=true must cover every edge")
	}

	var nilAgent *AgentConfig
	if nilAgent.AutoApprovesHandoffTo("y") {
		t.Error("nil agent must not auto-approve")
	}
}

// The verdict extraction seam: an EvaluatesParent agent's completion output
// carries an EVALUATION_VERDICT: line; extraction must find it anywhere in the
// output, and its ABSENCE must yield UNAVAILABLE — a completed evaluator that
// forgot to emit a verdict is not a pass.
func TestExtractEvaluationVerdict(t *testing.T) {
	out := "some preamble\nEVALUATION_VERDICT: PASS score=84\ntrailing notes"
	v := ExtractEvaluationVerdict(out)
	if v.Kind != VerdictPass || v.Score != 84 {
		t.Errorf("expected PASS 84, got %+v", v)
	}

	v = ExtractEvaluationVerdict("EVALUATION_VERDICT: FAIL score=45 reasons=AC2 unmet")
	if v.Kind != VerdictFail || v.Score != 45 {
		t.Errorf("expected FAIL 45, got %+v", v)
	}

	v = ExtractEvaluationVerdict("task completed fine, no verdict line at all")
	if v.Kind != VerdictUnavailable {
		t.Errorf("a missing verdict line must be UNAVAILABLE, got %+v", v)
	}

	// Last occurrence wins: an evaluator that revises its verdict mid-output
	// means the final one.
	v = ExtractEvaluationVerdict("EVALUATION_VERDICT: FAIL score=30\n...rechecked...\nEVALUATION_VERDICT: PASS score=75")
	if v.Kind != VerdictPass || v.Score != 75 {
		t.Errorf("last verdict must win, got %+v", v)
	}
}

// Blocking semantics at the automation seam: an approval whose Evaluation is
// FAIL or UNAVAILABLE must not be auto-progressable; PASS and (for now) an
// EMPTY evaluation must be — empty means "no evaluator stage configured", and
// making empty block would instantly freeze every non-evaluated agent's flow.
func TestApprovalAllowsAutomation(t *testing.T) {
	cases := map[string]bool{
		"":                        true, // no evaluator stage configured
		"PASS score=84":           true,
		"FAIL score=45 reasons=x": false,
		"UNAVAILABLE reason=died": false,
		"complete garbage":        false, // unparsable stored value = not a pass
	}
	for eval, want := range cases {
		rec := &ApprovalRequestRecord{Evaluation: eval}
		if got := rec.AllowsAutomation(); got != want {
			t.Errorf("AllowsAutomation(%q) = %v, want %v", eval, got, want)
		}
	}
}
