package quorum

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// agenticStub builds an AgenticRunner returning a canned run and records that it
// was called + the workspace-equivalent it saw (via a distinct closure per
// reviewer). Used to assert Tier-2 orchestration behavior without a real
// executor.
func agenticStub(run *AgenticRun, err error, called *bool, mu *sync.Mutex) AgenticRunner {
	return func(_ context.Context, _, _ string) (*AgenticRun, error) {
		mu.Lock()
		*called = true
		mu.Unlock()
		return run, err
	}
}

func passRun(cost float64) *AgenticRun {
	return &AgenticRun{Success: true, CostUSD: cost, Output: `{"verdict":"pass","strongest_objection":"verified: ran ailang check, it matches the doc","catch":"ran ailang check on cited repro"}`}
}

func rejectRun(cost float64) *AgenticRun {
	return &AgenticRun{Success: true, CostUSD: cost, Output: `{"verdict":"reject","strongest_objection":"premise X is FALSE — ran ailang check, it passed, contradicting the doc","catch":"ran ailang check on the cited repro"}`}
}

// TestRunTier2_EscalatedRejectBlocks proves an escalated agentic reject blocks
// the synthesis exactly like a Tier-1 reject.
func TestRunTier2_EscalatedRejectBlocks(t *testing.T) {
	var mu sync.Mutex
	var c1, c2 bool
	reviewers := []AgenticReviewer{
		{Label: "gpt5-6-sol@codex", Runner: agenticStub(rejectRun(0.05), nil, &c1, &mu)},
		{Label: "claude@claude", Runner: agenticStub(passRun(0.04), nil, &c2, &mu)},
	}
	decision := EscalationDecision{Escalate: true, Reason: "premise-class", ContestedPremise: "premise X"}
	tier2 := RunTier2("doc.md", "body", decision, reviewers, DefaultMaxCostUSD, time.Minute)

	if tier2 == nil {
		t.Fatal("expected a Tier-2 result on escalation")
	}
	if !c1 || !c2 {
		t.Errorf("both agentic reviewers should have run: c1=%v c2=%v", c1, c2)
	}
	// A clean Tier-1 pass + Tier-2 reject => blocked.
	tier1 := []*ReviewerOutcome{present("m1", VerdictPass, "minor")}
	s := SynthesizeWithTier2(tier1, nil, tier2)
	if s.Verdict != SynthBlocked {
		t.Fatalf("escalated reject must block synthesis, got %q", s.Verdict)
	}
	found := false
	for _, obj := range s.BlockingObjections {
		if strings.Contains(obj, "[tier2]") && strings.Contains(obj, "premise X is FALSE") {
			found = true
		}
	}
	if !found {
		t.Errorf("blocking objections missing the labeled tier2 reject: %v", s.BlockingObjections)
	}
	// Every Tier-2 outcome is tagged with the tier label.
	for _, o := range tier2.Reviewers {
		if o.Tier != TierAgentic {
			t.Errorf("tier2 outcome %q not labeled %q (got %q)", o.Model, TierAgentic, o.Tier)
		}
	}
}

// TestRunTier2_NoEscalateSkipsAgentic proves no agentic call is made when
// escalation does not fire (cost-smart default).
func TestRunTier2_NoEscalateSkipsAgentic(t *testing.T) {
	var mu sync.Mutex
	var called bool
	reviewers := []AgenticReviewer{
		{Label: "gpt5-6-sol@codex", Runner: agenticStub(rejectRun(0.05), nil, &called, &mu)},
	}
	decision := EscalationDecision{Escalate: false}
	tier2 := RunTier2("doc.md", "body", decision, reviewers, DefaultMaxCostUSD, time.Minute)

	if tier2 != nil {
		t.Fatalf("no escalation must yield a nil Tier-2 result")
	}
	if called {
		t.Errorf("no agentic call must be made when escalation does not fire")
	}
}

// TestRunTier2_OverCapDegradesButComposes proves an over-cap Tier-2 reviewer
// becomes a named budget absence and the synthesis still composes (N-1).
func TestRunTier2_OverCapDegradesButComposes(t *testing.T) {
	var mu sync.Mutex
	var c1, c2 bool
	reviewers := []AgenticReviewer{
		{Label: "expensive@codex", Runner: agenticStub(passRun(0.99), nil, &c1, &mu)}, // over cap
		{Label: "claude@claude", Runner: agenticStub(rejectRun(0.03), nil, &c2, &mu)},
	}
	decision := EscalationDecision{Escalate: true, Reason: "split"}
	tier2 := RunTier2("doc.md", "body", decision, reviewers, DefaultMaxCostUSD, time.Minute)

	if tier2 == nil {
		t.Fatal("expected Tier-2 result")
	}
	var budgetAbsent, presentReject bool
	for _, o := range tier2.Reviewers {
		if o.Model == "expensive@codex" && !o.Present && o.AbsentReason == ReasonBudget {
			budgetAbsent = true
		}
		if o.Model == "claude@claude" && o.Present && o.Result.Verdict == VerdictReject {
			presentReject = true
		}
	}
	if !budgetAbsent {
		t.Errorf("over-cap Tier-2 reviewer should be a named budget absence")
	}
	if !presentReject {
		t.Errorf("the in-cap reviewer should still produce its verdict")
	}
	// Synthesis still composes: the in-cap reject blocks; the budget absence is named.
	s := SynthesizeWithTier2([]*ReviewerOutcome{present("m1", VerdictPass, "minor")}, nil, tier2)
	if s.Verdict != SynthBlocked {
		t.Errorf("in-cap reject should block despite the other reviewer's budget absence")
	}
}

// TestSynthesizeWithTier2_NilIsTier1Identical proves a nil Tier-2 yields the
// exact Tier-1 synthesis (shipped shape preserved).
func TestSynthesizeWithTier2_NilIsTier1Identical(t *testing.T) {
	tier1 := []*ReviewerOutcome{
		present("m1", VerdictPass, "minor"),
		present("m2", VerdictPass, "minor"),
	}
	base := synthesize(tier1, nil)
	with := SynthesizeWithTier2(tier1, nil, nil)
	if with.Verdict != base.Verdict || len(with.BlockingObjections) != len(base.BlockingObjections) {
		t.Errorf("nil Tier-2 must be identical to Tier-1 synthesis: base=%+v with=%+v", base, with)
	}
}

// TestRunQuorumWithEscalation_EscalatesOnPremiseObjection is the end-to-end
// orchestration test: a premise-class Tier-1 reject triggers Tier-2, and the
// agentic reviewers run and fold into the final synthesis.
func TestRunQuorumWithEscalation_EscalatesOnPremiseObjection(t *testing.T) {
	tier1Table := map[string]*ReviewerOutcome{
		"m1": present("m1", VerdictReject, "premise X is unverified — the doc claims foo.go exports Bar"),
		"m2": present("m2", VerdictPass, "no other concern"),
	}
	var mu sync.Mutex
	var agenticCalled bool
	agentic := []AgenticReviewer{
		{Label: "gpt5-6-sol@codex", Runner: agenticStub(rejectRun(0.05), nil, &agenticCalled, &mu)},
	}
	q := RunQuorumWithEscalation(
		"doc.md", "body", "2026-07-16T00:00:00Z",
		[]string{"m1", "m2"}, DefaultMaxCostUSD, nil, fakeRunner(tier1Table),
		agentic, time.Minute,
	)
	if q.Tier2 == nil {
		t.Fatal("expected Tier-2 to run on a premise-class objection")
	}
	if !agenticCalled {
		t.Errorf("the agentic reviewer should have been called")
	}
	if q.Tier2.Decision.Reason != "premise-class" {
		t.Errorf("escalation reason = %q, want premise-class", q.Tier2.Decision.Reason)
	}
	if q.Synthesis.Verdict != SynthBlocked {
		t.Errorf("premise-class Tier-1 reject + Tier-2 reject => blocked, got %q", q.Synthesis.Verdict)
	}
}

// TestRunQuorumWithEscalation_NoEscalateOnCleanPass proves the cost-smart
// default: a clean unanimous non-high-stakes pass never runs Tier-2.
func TestRunQuorumWithEscalation_NoEscalateOnCleanPass(t *testing.T) {
	tier1Table := map[string]*ReviewerOutcome{
		"m1": present("m1", VerdictPass, "minor"),
		"m2": present("m2", VerdictPass, "minor"),
	}
	var mu sync.Mutex
	var agenticCalled bool
	agentic := []AgenticReviewer{
		{Label: "gpt5-6-sol@codex", Runner: agenticStub(rejectRun(0.05), nil, &agenticCalled, &mu)},
	}
	// docBody has no high-stakes marker.
	q := RunQuorumWithEscalation(
		"doc.md", "a plain doc body with no stakes markers", "t",
		[]string{"m1", "m2"}, DefaultMaxCostUSD, nil, fakeRunner(tier1Table),
		agentic, time.Minute,
	)
	if q.Tier2 != nil {
		t.Fatalf("clean unanimous pass must not run Tier-2")
	}
	if agenticCalled {
		t.Errorf("no agentic call on a clean pass (cost-smart default)")
	}
	if q.Synthesis.Verdict != SynthProceed {
		t.Errorf("clean pass should proceed, got %q", q.Synthesis.Verdict)
	}
}

// TestTier2Artifact_RendersDistinctly proves Tier-2 outcomes are recorded
// distinctly (tier label) in the markdown while the Tier-1 shape is preserved.
func TestTier2Artifact_RendersDistinctly(t *testing.T) {
	tier1Table := map[string]*ReviewerOutcome{
		"m1": present("m1", VerdictReject, "premise unverified: doc claims X"),
	}
	var mu sync.Mutex
	var called bool
	agentic := []AgenticReviewer{
		{Label: "gpt5-6-sol@codex", Runner: agenticStub(rejectRun(0.05), nil, &called, &mu)},
	}
	q := RunQuorumWithEscalation(
		"design_docs/foo.md", "body", "2026-07-16T00:00:00Z",
		[]string{"m1"}, DefaultMaxCostUSD, nil, fakeRunner(tier1Table),
		agentic, time.Minute,
	)
	md := MarkdownBlock(q)
	for _, want := range []string{"Tier-2 escalation", "premise-class", "tier2", "gpt5-6-sol@codex"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
}
