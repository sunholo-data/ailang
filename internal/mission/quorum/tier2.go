package quorum

import (
	"fmt"
	"sync"
	"time"
)

// AgenticReviewer names one Tier-2 escalated reviewer: a label for the artifact
// (e.g. "gpt5-6-sol@codex") and the bounded AgenticRunner that runs it over its
// OWN read-only worktree. The mission-control wiring builds these from
// NewCoordinatorAgenticRunner (codex=OpenAI, claude — each with a distinct
// read-only worktree; gemini/managed_agents is the M0-gated M4 sub-path, out of
// scope here).
type AgenticReviewer struct {
	// Label is the artifact model label for this reviewer.
	Label string
	// Runner is the bounded, read-only agentic run for this reviewer.
	Runner AgenticRunner
}

// Tier2Result records the escalated verification pass: the decision that
// triggered it (for audit) plus the per-reviewer outcomes. It is ADDITIVE to
// QuorumResult — a run with no escalation leaves it nil, preserving the shipped
// Tier-1 artifact shape for existing Phase E consumers.
type Tier2Result struct {
	Decision  EscalationDecision `json:"decision"`
	Reviewers []*ReviewerOutcome `json:"reviewers"`
}

// RunTier2 runs the escalated agentic verification IF ShouldEscalate fired. It
// returns nil when no escalation is warranted (cost-smart default: no agentic
// call is made). Each reviewer runs in PARALLEL over its OWN read-only worktree
// with the same bounded + N-1 discipline as Tier-1: an over-cap or hung
// reviewer degrades to a named absence, never blocks the loop.
//
// docPath/docBody are the reviewed doc; decision carries the contested premise
// (focusing the agentic reviewer). maxCostUSD is the per-review cap; timeout
// bounds each reviewer's wall clock.
//
// It NEVER makes an agentic call when decision.Escalate is false — the caller
// must have already computed the decision via ShouldEscalate.
func RunTier2(docPath, docBody string, decision EscalationDecision, reviewers []AgenticReviewer, maxCostUSD float64, timeout time.Duration) *Tier2Result {
	if !decision.Escalate || len(reviewers) == 0 {
		return nil
	}

	outcomes := make([]*ReviewerOutcome, len(reviewers))
	var wg sync.WaitGroup
	for i, r := range reviewers {
		wg.Add(1)
		go func(i int, r AgenticReviewer) {
			defer wg.Done()
			out := RunAgenticReviewer(r.Label, docPath, docBody, decision.ContestedPremise, maxCostUSD, timeout, r.Runner)
			out.Tier = TierAgentic
			outcomes[i] = out
		}(i, r)
	}
	wg.Wait()

	return &Tier2Result{Decision: decision, Reviewers: outcomes}
}

// SynthesizeWithTier2 composes the FULL verdict over Tier-1 + Tier-2 outcomes.
// An escalated (Tier-2) reject blocks EXACTLY like a Tier-1 reject — the
// controller still synthesizes; the reviewer has NO independent merge authority.
// When tier2 is nil (no escalation), it returns the plain Tier-1 synthesis
// byte-identically (existing shape preserved).
//
// The blocking-objection text for a Tier-2 reject is prefixed "[tier2] " so the
// synthesis makes the escalated origin visible to the author.
func SynthesizeWithTier2(tier1 []*ReviewerOutcome, controller *ControllerReview, tier2 *Tier2Result) Synthesis {
	s := synthesize(tier1, controller)
	if tier2 == nil {
		return s
	}
	for _, o := range tier2.Reviewers {
		if o == nil {
			continue
		}
		s.TotalCostUSD += o.CostUSD
		if !o.Present {
			s.AbsentReviewers = append(s.AbsentReviewers, AbsentNote{Model: o.Model, Reason: o.AbsentReason})
			continue
		}
		if o.Result.Verdict == VerdictReject {
			s.Verdict = SynthBlocked
			s.BlockingObjections = append(s.BlockingObjections,
				fmt.Sprintf("[tier2] %s: %s", o.Model, o.Result.StrongestObjection))
		}
	}
	return s
}
