package quorum

import (
	"strings"
	"testing"
)

// qr builds a QuorumResult from a set of reviewer outcomes for escalation tests.
func qr(reviewers []*ReviewerOutcome, controller *ControllerReview) *QuorumResult {
	return &QuorumResult{
		Doc:                 "doc.md",
		Reviewers:           reviewers,
		ControllerInSession: controller,
		Synthesis:           synthesize(reviewers, controller),
	}
}

func TestShouldEscalate_PremiseClassObjection(t *testing.T) {
	q := qr([]*ReviewerOutcome{
		present("m1", VerdictReject, "premise X is unverified: the doc claims internal/foo.go exports BarFunc"),
		present("m2", VerdictReject, "premise Y is unverified too"),
	}, nil)
	d := ShouldEscalate(q, false)
	if !d.Escalate {
		t.Fatalf("expected escalation on premise-class objection")
	}
	if d.Reason != "premise-class" {
		t.Errorf("reason = %q, want premise-class", d.Reason)
	}
	// Deterministic: the FIRST reviewer's objection is the contested premise.
	if !strings.Contains(d.ContestedPremise, "internal/foo.go exports BarFunc") {
		t.Errorf("contested premise = %q, want the m1 objection", d.ContestedPremise)
	}
}

func TestShouldEscalate_HighStakesDoc(t *testing.T) {
	// Unanimous pass, but the doc self-declares high-stakes => escalate.
	q := qr([]*ReviewerOutcome{
		present("m1", VerdictPass, "minor naming nit"),
		present("m2", VerdictPass, "no material concern"),
	}, nil)
	d := ShouldEscalate(q, true)
	if !d.Escalate {
		t.Fatalf("expected escalation on high-stakes doc")
	}
	if d.Reason != "high-stakes" {
		t.Errorf("reason = %q, want high-stakes", d.Reason)
	}
}

func TestShouldEscalate_SplitQuorum(t *testing.T) {
	// One pass, one reject with a NON-premise objection => split trigger.
	q := qr([]*ReviewerOutcome{
		present("m1", VerdictPass, "looks fine to me"),
		present("m2", VerdictReject, "the axiom bias toward extension is not respected"),
	}, nil)
	d := ShouldEscalate(q, false)
	if !d.Escalate {
		t.Fatalf("expected escalation on split quorum")
	}
	// A reject whose text isn't premise-class falls through to the split branch.
	if d.Reason != "split" {
		t.Errorf("reason = %q, want split", d.Reason)
	}
}

func TestShouldEscalate_CleanUnanimousPassDoesNotEscalate(t *testing.T) {
	// The cost-smart default: a clean unanimous non-high-stakes pass NEVER
	// triggers the dollar-billed Tier-2.
	q := qr([]*ReviewerOutcome{
		present("m1", VerdictPass, "minor naming nit"),
		present("m2", VerdictPass, "no material concern"),
	}, nil)
	d := ShouldEscalate(q, false)
	if d.Escalate {
		t.Fatalf("clean unanimous pass must NOT escalate (reason=%q)", d.Reason)
	}
}

func TestShouldEscalate_ControllerSplit(t *testing.T) {
	// The controller disagreeing with a present reviewer is also a split.
	q := qr([]*ReviewerOutcome{
		present("m1", VerdictPass, "fine"),
	}, &ControllerReview{Verdict: VerdictReject, Note: "an axiom conflict I see in session"})
	d := ShouldEscalate(q, false)
	if !d.Escalate || d.Reason != "split" {
		t.Fatalf("controller/reviewer disagreement should escalate as split, got %+v", d)
	}
}

func TestShouldEscalate_NilResult(t *testing.T) {
	if ShouldEscalate(nil, false).Escalate {
		t.Errorf("nil quorum result must not escalate")
	}
}

func TestDocSelfDeclaresHighStakes(t *testing.T) {
	if !DocSelfDeclaresHighStakes("## Conflict surface\nThis touches shared infra.") {
		t.Errorf("expected high-stakes for a doc with a Conflict surface section")
	}
	if DocSelfDeclaresHighStakes("A plain doc with no stakes markers at all.") {
		t.Errorf("plain doc should not be flagged high-stakes")
	}
}

// --- proposed_fix optionality (M2, option (a)) ---

// TestProposedFix_OptionalNotValidated proves proposed_fix is additive-optional:
// a verdict WITHOUT it and a verdict WITH it BOTH pass the UNCHANGED
// ValidateReviewResult, byte-identically to before.
func TestProposedFix_OptionalNotValidated(t *testing.T) {
	without := &ReviewResult{Verdict: VerdictReject, StrongestObjection: "premise unverified", Catch: "grepped foo.go"}
	if err := ValidateReviewResult(without); err != nil {
		t.Fatalf("a reject WITHOUT proposed_fix must validate (option (a)): %v", err)
	}
	with := &ReviewResult{Verdict: VerdictReject, StrongestObjection: "premise unverified", Catch: "grepped foo.go", ProposedFix: "replace paragraph 2 with the corrected claim"}
	if err := ValidateReviewResult(with); err != nil {
		t.Fatalf("a reject WITH proposed_fix must validate: %v", err)
	}
}

// TestProposedFix_ParsesFromJSON proves the field round-trips through the parse
// path AND that a reject with an empty proposed_fix is NOT a validation error
// (recorded as friction, not a hard error).
func TestProposedFix_ParsesFromJSON(t *testing.T) {
	raw := `{"verdict":"reject","strongest_objection":"premise X unverified","catch":"grepped foo.go; BarFunc absent","proposed_fix":"correct the claim to reference internal/bar.go instead"}`
	r, err := ParseReviewResult(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.ProposedFix != "correct the claim to reference internal/bar.go instead" {
		t.Errorf("proposed_fix not parsed: %q", r.ProposedFix)
	}

	// A fix-less reject still parses + validates fine (proves it is NOT required).
	fixless := `{"verdict":"reject","strongest_objection":"premise X unverified","catch":"grepped foo.go; BarFunc absent"}`
	if _, err := ParseReviewResult(fixless); err != nil {
		t.Fatalf("fix-less reject must parse+validate (friction, not error): %v", err)
	}
}

// TestReviewSchema_ProposedFixNotRequired asserts proposed_fix is in properties
// but ABSENT from required — the frozen-contract guard.
func TestReviewSchema_ProposedFixNotRequired(t *testing.T) {
	if !strings.Contains(reviewSchema, `"proposed_fix"`) {
		t.Errorf("proposed_fix missing from schema properties")
	}
	// required[] must NOT contain proposed_fix.
	if strings.Contains(reviewSchema, `"required": ["verdict", "strongest_objection", "catch", "proposed_fix"]`) {
		t.Errorf("proposed_fix must NOT be in reviewSchema.required (contract frozen)")
	}
	// The required list is exactly the three frozen fields.
	if !strings.Contains(reviewSchema, `"required": ["verdict", "strongest_objection", "catch"]`) {
		t.Errorf("frozen required[] changed; must be exactly [verdict, strongest_objection, catch]")
	}
}
