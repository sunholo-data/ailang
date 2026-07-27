package observatory

import (
	"context"
	"fmt"
	"time"
)

// Cost-Per-Verified-Success KPI rollup (M-COST-PER-SUCCESS-KPI, M1).
//
// This is the SINGLE authoritative computation of the v1.0 headline KPI:
//
//	cost_per_verified_success_usd = total_known_metered_cost_usd(C) / verified_success_count(C)
//
// for one frozen benchmark cohort C. It is intentionally narrow and reuses the
// already-shipped, authoritative cost primitives — it does NOT reimplement cost
// math. Every consumer (CLI, HTTP handler, latest.json publisher, React) MUST
// call this and serialize its fields rather than recompute anything:
//
//   - Numerator: ClassifyStageCost + CostRollup.TotalKnownCost() (reported +
//     estimated), summed over EVERY stage in the cohort — including the cost of
//     FAILED runs. Quota lanes contribute $0-by-design but are counted; a single
//     UNKNOWN-cost stage makes the whole KPI unavailable (never a silent $0).
//   - Denominator: the count of VERIFIED successes (see isVerifiedSuccess).
//
// The predicate for a VERIFIED success is defined EXACTLY ONCE here (the doc's
// High-Impact-Decision row). Nothing else in the codebase may re-derive it.

// CostPerVerifiedSuccessOptions selects the frozen cohort. It is parameterized
// only — no v1.0 cohort is materialized in this sprint (that is M4, Mark-gated).
type CostPerVerifiedSuccessOptions struct {
	// BaselineID is the caller-supplied identity of the frozen cohort (e.g.
	// "v1.0"). It is echoed into the result for provenance; it does NOT filter.
	BaselineID string
	// SourceRef is the chains.source_ref prefix that scopes the cohort (e.g.
	// "v1.0/"). This is how mission-development spend ("mission:...") is excluded.
	SourceRef string
	// Optional cohort window on chains.created_at.
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	// Language / EvalMode scope the cohort. Defaults ("ailang" / "agent") are
	// applied when empty because the headline is an AILANG agent-mode number.
	Language string
	EvalMode string
}

// CostPerVerifiedSuccessResult is the canonical, serialized-everywhere KPI.
//
// The JSON field names are the wire contract shared by the CLI (--json), the
// HTTP handler, and the latest.json headline object. Do NOT rename them without
// updating all three consumers and the React card.
type CostPerVerifiedSuccessResult struct {
	// Identity / provenance.
	BaselineID  string    `json:"baseline_id"`
	GeneratedAt time.Time `json:"generated_at"`
	Language    string    `json:"language"`
	EvalMode    string    `json:"eval_mode"`
	SourceRef   string    `json:"source_ref"`

	// Denominator breakdown.
	TotalRuns            int `json:"total_runs"`
	PassedRuns           int `json:"passed_runs"` // compile+runtime+stdout ok (verified or not)
	VerifiedSuccesses    int `json:"verified_successes"`
	UnverifiedPasses     int `json:"unverified_passes"`
	VerificationFailures int `json:"verification_failures"`

	// Numerator / cost provenance (reuses CostRollup semantics).
	ReportedCostUSD  float64 `json:"reported_cost_usd"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	KnownCostUSD     float64 `json:"known_cost_usd"` // reported + estimated
	QuotaStages      int     `json:"quota_stages"`
	UnknownStages    int     `json:"unknown_stages"`

	// Completeness / result.
	IncompleteData            bool    `json:"incomplete_data"`
	CostPerVerifiedSuccessUSD float64 `json:"cost_per_verified_success_usd"`
	Available                 bool    `json:"available"`
	// Reason is a short machine tag for why Available is false; "" when available.
	Reason string `json:"reason,omitempty"`
}

// Reason tags for an unavailable KPI (stable machine strings).
const (
	CVSReasonEmptyCohort     = "empty_cohort"
	CVSReasonUnknownCost     = "unknown_cost"
	CVSReasonZeroDenominator = "zero_denominator"
)

// isVerifiedSuccess is the ONE definition of a VERIFIED success. A run counts
// only when it passes compile/runtime/stdout grading AND carries affirmative
// verification evidence: verify_ok, at least one proved obligation
// (verify_verified > 0), and zero counterexamples, skipped obligations, and
// verifier errors. An absent/all-zero verify block is verification_missing —
// never a success (backward-compat with historical banked rows).
func isVerifiedSuccess(a *EvalAssessment) bool {
	if a == nil {
		return false
	}
	return a.CompileOk && a.RuntimeOk && a.StdoutOk &&
		a.VerifyOk &&
		a.VerifyVerified > 0 &&
		a.VerifyCounterex == 0 &&
		a.VerifySkipped == 0 &&
		a.VerifyErrors == 0
}

// isPass reports execution-grading success (compile+runtime+stdout), independent
// of verification. This matches the existing pass-rate / per-model $/success view.
func isPass(a *EvalAssessment) bool {
	if a == nil {
		return false
	}
	return a.CompileOk && a.RuntimeOk && a.StdoutOk
}

// isVerificationFailure reports an ATTEMPTED-but-failed verification on a passing
// run: verification produced a counterexample, a skipped obligation, or a
// verifier error. (A pass with NO verification evidence is verification_missing,
// i.e. an unverified pass — not a failure.)
func isVerificationFailure(a *EvalAssessment) bool {
	if a == nil || !isPass(a) {
		return false
	}
	return a.VerifyCounterex > 0 || a.VerifySkipped > 0 || a.VerifyErrors > 0
}

// computeCostPerVerifiedSuccess is the observatory-local rollup. It queries the
// observatory's OWN chains store via QueryEvalResults (no eval-package import,
// no import cycle), applies the single verified-success predicate, and reuses
// ClassifyStageCost / CostRollup for the numerator.
func computeCostPerVerifiedSuccess(ctx context.Context, store *Store, opts CostPerVerifiedSuccessOptions) (*CostPerVerifiedSuccessResult, error) {
	if store == nil {
		return nil, fmt.Errorf("observatory store is required")
	}

	lang := opts.Language
	if lang == "" {
		lang = "ailang"
	}
	mode := opts.EvalMode
	if mode == "" {
		mode = "agent"
	}

	res := &CostPerVerifiedSuccessResult{
		BaselineID:  opts.BaselineID,
		GeneratedAt: time.Now().UTC(),
		Language:    lang,
		EvalMode:    mode,
		SourceRef:   opts.SourceRef,
	}

	stages, err := store.QueryEvalResults(ctx, EvalQueryOptions{
		SourceRefPrefix: opts.SourceRef,
		CreatedAfter:    opts.CreatedAfter,
		CreatedBefore:   opts.CreatedBefore,
		Language:        lang,
		EvalMode:        mode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query cohort eval results: %w", err)
	}

	var rollup CostRollup
	for _, stage := range stages {
		if stage == nil {
			continue
		}
		res.TotalRuns++
		a := stage.EvalAssessment

		switch {
		case isVerifiedSuccess(a):
			res.VerifiedSuccesses++
			res.PassedRuns++
		case isVerificationFailure(a):
			res.VerificationFailures++
			res.PassedRuns++
		case isPass(a):
			res.UnverifiedPasses++
			res.PassedRuns++
		}

		// Numerator: classify EVERY stage's cost (passing or failing) and fold
		// it in. This is where failed-run cost enters the numerator.
		rollup.AddStage(stage)
	}

	res.ReportedCostUSD = rollup.ReportedCost
	res.EstimatedCostUSD = rollup.EstimatedCost
	res.KnownCostUSD = rollup.TotalKnownCost()
	res.QuotaStages = rollup.QuotaStages
	res.UnknownStages = rollup.UnknownStages
	res.IncompleteData = rollup.HasIncompleteData()

	// Availability (fail loud — never emit $0/infinity/stale on any of these):
	//  1. empty cohort               → unavailable
	//  2. any unknown-cost stage     → unavailable (numerator would understate)
	//  3. zero verified successes    → unavailable (zero denominator)
	switch {
	case res.TotalRuns == 0:
		res.Available = false
		res.Reason = CVSReasonEmptyCohort
	case res.UnknownStages > 0:
		res.Available = false
		res.Reason = CVSReasonUnknownCost
	case res.VerifiedSuccesses == 0:
		res.Available = false
		res.Reason = CVSReasonZeroDenominator
	default:
		res.CostPerVerifiedSuccessUSD = res.KnownCostUSD / float64(res.VerifiedSuccesses)
		res.Available = true
	}

	return res, nil
}

// CostPerVerifiedSuccess is the exported entry point for consumers (CLI, HTTP,
// publisher). It is a thin wrapper over the observatory-local rollup so callers
// never recompute cost or re-derive the verified-success predicate.
func (s *Store) CostPerVerifiedSuccess(ctx context.Context, opts CostPerVerifiedSuccessOptions) (*CostPerVerifiedSuccessResult, error) {
	return computeCostPerVerifiedSuccess(ctx, s, opts)
}
