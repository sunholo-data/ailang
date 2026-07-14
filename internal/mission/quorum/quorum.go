package quorum

import (
	"fmt"
	"sync"
)

// SynthesisVerdict is the quorum's composed decision.
type SynthesisVerdict string

const (
	// SynthProceed: unanimous pass among PRESENT reviewers → the doc proceeds.
	SynthProceed SynthesisVerdict = "proceed"
	// SynthBlocked: at least one present reviewer rejected → the objection
	// goes back to the doc author before planning.
	SynthBlocked SynthesisVerdict = "blocked"
)

// ControllerReview is the Claude controller's IN-SESSION judgement. It is NOT
// an API call (design doc gate 6) — the running mission-control skill fills it
// in from its own reasoning. Recorded as a distinct entry so the artifact
// shows the controller participated without pretending it was a provider call.
type ControllerReview struct {
	Verdict Verdict `json:"verdict"`
	Note    string  `json:"note"`
}

// QuorumResult is the full composed verdict + the per-reviewer records + the
// controller's in-session review. It is the artifact that seeds Phase E.
type QuorumResult struct {
	Doc                 string             `json:"doc"`
	ISOTimestamp        string             `json:"iso_ts"`
	Reviewers           []*ReviewerOutcome `json:"reviewers"`
	ControllerInSession *ControllerReview  `json:"controller_in_session,omitempty"`
	Synthesis           Synthesis          `json:"synthesis"`
}

// Synthesis is the composed decision plus the specific blocking objections and
// an explicit list of absent reviewers (never a silent pass).
type Synthesis struct {
	Verdict            SynthesisVerdict `json:"verdict"`
	BlockingObjections []string         `json:"blocking_objections"`
	// AbsentReviewers names every reviewer that did not produce a verdict,
	// with its reason. Empty when all reviewers were present.
	AbsentReviewers []AbsentNote `json:"absent_reviewers"`
	TotalCostUSD    float64      `json:"total_cost_usd"`
}

// AbsentNote records one missing reviewer for the synthesis.
type AbsentNote struct {
	Model  string `json:"model"`
	Reason string `json:"reason"`
}

// RunQuorum runs the given reviewer models in PARALLEL and composes their
// verdicts. The controller's in-session review (if provided) is folded into
// the synthesis as a distinct participant. Degradation is graceful: an absent
// reviewer is recorded (named, with reason) and the quorum proceeds with the
// remaining present reviewers — a missing reviewer is NEVER a silent pass.
//
// runner is the per-reviewer function (RunReviewer in production; a stub in
// tests). isoTS is injected so the artifact timestamp is deterministic in
// tests.
func RunQuorum(docPath, docBody, isoTS string, reviewerModels []string, maxCostUSD float64, controller *ControllerReview, runner func(model, docPath, docBody string, maxCostUSD float64) *ReviewerOutcome) *QuorumResult {
	outcomes := make([]*ReviewerOutcome, len(reviewerModels))
	var wg sync.WaitGroup
	for i, model := range reviewerModels {
		wg.Add(1)
		go func(i int, model string) {
			defer wg.Done()
			outcomes[i] = runner(model, docPath, docBody, maxCostUSD)
		}(i, model)
	}
	wg.Wait()

	return &QuorumResult{
		Doc:                 docPath,
		ISOTimestamp:        isoTS,
		Reviewers:           outcomes,
		ControllerInSession: controller,
		Synthesis:           synthesize(outcomes, controller),
	}
}

// synthesize composes present reviewers + the controller into a verdict.
//
// Rule (from the design doc):
//   - any present reviewer (or the controller) REJECTS → blocked; the objection
//     goes back to the author.
//   - all present participants PASS → proceed.
//
// Absent reviewers are recorded but do not vote; they can never turn a real
// reject into a proceed, and their absence is always visible (Principle 2).
func synthesize(outcomes []*ReviewerOutcome, controller *ControllerReview) Synthesis {
	s := Synthesis{Verdict: SynthProceed}
	presentCount := 0

	for _, o := range outcomes {
		s.TotalCostUSD += o.CostUSD
		if !o.Present {
			s.AbsentReviewers = append(s.AbsentReviewers, AbsentNote{Model: o.Model, Reason: o.AbsentReason})
			continue
		}
		presentCount++
		if o.Result.Verdict == VerdictReject {
			s.Verdict = SynthBlocked
			s.BlockingObjections = append(s.BlockingObjections,
				fmt.Sprintf("%s: %s", o.Model, o.Result.StrongestObjection))
		}
	}

	if controller != nil {
		presentCount++
		if controller.Verdict == VerdictReject {
			s.Verdict = SynthBlocked
			s.BlockingObjections = append(s.BlockingObjections,
				fmt.Sprintf("controller (in-session): %s", controller.Note))
		}
	}

	// Guard: if NO participant was present at all, that is not a pass — it is a
	// blocked verdict with a synthetic objection so the loop never proceeds on
	// zero signal (Principle 2 at the quorum level).
	if presentCount == 0 {
		s.Verdict = SynthBlocked
		s.BlockingObjections = append(s.BlockingObjections,
			"no reviewer produced a verdict (all absent) — refusing to proceed on zero signal")
	}

	return s
}
