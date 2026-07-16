package quorum

import "strings"

// EscalationDecision is the result of ShouldEscalate: whether to run a Tier-2
// agentic verification and, if so, the specific contested premise the escalated
// reviewer should verify (empty when escalation is triggered by high-stakes or
// a split rather than a specific premise objection).
type EscalationDecision struct {
	Escalate bool
	// ContestedPremise is the Tier-1 objection text that motivated escalation
	// (the premise the agentic reviewer should focus on). Empty for a
	// high-stakes-only or split-only trigger with no premise-class objection.
	ContestedPremise string
	// Reason names WHY escalation fired (audit): "premise-class", "high-stakes",
	// or "split". Empty when Escalate is false.
	Reason string
}

// premiseSignals are lower-cased substrings that mark a Tier-1 objection as
// PREMISE-class: a disputed factual claim about the codebase (a file, API, or
// behavior) that an agentic reviewer could VERIFY against the code. This is a
// heuristic on the objection text; it errs toward escalation (a false positive
// only adds a verification pass, never removes signal).
var premiseSignals = []string{
	"premise",
	"unverified",
	"not verified",
	"verify",
	"verification",
	"claim",
	"claims",
	"asserted",
	"assertion",
	"does not exist",
	"doesn't exist",
	"no such",
	"contradict",
	"factual",
	"cite",
	"citation",
	"grep",
	"ailang check",
}

// isPremiseClass reports whether an objection text disputes a factual codebase
// claim (a premise an agentic reviewer could verify).
func isPremiseClass(objection string) bool {
	lc := strings.ToLower(objection)
	for _, sig := range premiseSignals {
		if strings.Contains(lc, sig) {
			return true
		}
	}
	return false
}

// ShouldEscalate is a PURE function (no I/O) deciding whether the Tier-1 quorum
// result warrants a Tier-2 agentic verification. It escalates when ANY of:
//
//	(a) any present Tier-1 reviewer's objection is PREMISE-class (a disputed
//	    factual codebase claim) — the exact class an agentic reviewer can verify
//	    and the text tier structurally cannot;
//	(b) the doc self-declares high-stakes (touches shared infra / a Conflict
//	    Surface) — docSelfDeclaredHighStakes;
//	(c) Tier-1 is SPLIT — present reviewers disagree (at least one pass AND at
//	    least one reject among present reviewers, or the controller disagrees
//	    with a present reviewer).
//
// Otherwise Tier-1 alone stands (cost-smart default): a clean unanimous
// non-high-stakes pass does NOT escalate.
//
// The contested premise returned is the FIRST premise-class objection found
// (deterministic: reviewers are iterated in their fixed slice order), so the
// Tier-2 reviewer focuses on a specific claim rather than re-reviewing the doc.
func ShouldEscalate(q *QuorumResult, docSelfDeclaredHighStakes bool) EscalationDecision {
	if q == nil {
		return EscalationDecision{}
	}

	// (a) premise-class objection among present reviewers (deterministic order).
	for _, o := range q.Reviewers {
		if o == nil || !o.Present || o.Result == nil {
			continue
		}
		if o.Result.Verdict == VerdictReject && isPremiseClass(o.Result.StrongestObjection) {
			return EscalationDecision{
				Escalate:         true,
				ContestedPremise: o.Result.StrongestObjection,
				Reason:           "premise-class",
			}
		}
	}

	// (b) doc self-declares high-stakes.
	if docSelfDeclaredHighStakes {
		return EscalationDecision{Escalate: true, Reason: "high-stakes"}
	}

	// (c) Tier-1 split: present reviewers (and the controller) disagree.
	if tier1Split(q) {
		return EscalationDecision{Escalate: true, Reason: "split"}
	}

	return EscalationDecision{}
}

// tier1Split reports whether the present Tier-1 participants disagree — at least
// one pass AND at least one reject among present reviewers plus the controller.
func tier1Split(q *QuorumResult) bool {
	sawPass, sawReject := false, false
	for _, o := range q.Reviewers {
		if o == nil || !o.Present || o.Result == nil {
			continue
		}
		switch o.Result.Verdict {
		case VerdictPass:
			sawPass = true
		case VerdictReject:
			sawReject = true
		}
	}
	if q.ControllerInSession != nil {
		switch q.ControllerInSession.Verdict {
		case VerdictPass:
			sawPass = true
		case VerdictReject:
			sawReject = true
		}
	}
	return sawPass && sawReject
}

// DocSelfDeclaresHighStakes is a small helper that inspects a doc body for a
// self-declared high-stakes marker: the presence of a "Conflict surface"
// section (the design-doc-creator convention) or an explicit stakes marker.
// It is a heuristic input to ShouldEscalate, parsed from the doc body already
// available to BuildPrompt — no I/O.
func DocSelfDeclaresHighStakes(docBody string) bool {
	lc := strings.ToLower(docBody)
	markers := []string{
		"conflict surface",
		"conflict-surface",
		"high-stakes",
		"high stakes",
		"shared infra",
		"touches shared",
	}
	for _, m := range markers {
		if strings.Contains(lc, m) {
			return true
		}
	}
	return false
}
