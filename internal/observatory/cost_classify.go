package observatory

// Cost-attribution classifier (M-MISSION-COST-CHAINS, M1).
//
// The chains rollup historically summed the self-reported `cost` column with a SQL
// `SUM()`. That silently shows a misleading `$0.0000` for any token-bearing stage
// whose sender never self-reported a cost — a false "free" signal that undermines
// the mission's cost tracker.
//
// This file adds a READ-SIDE, per-stage classifier under Mark's SCOPED-INFERENCE
// rule (2026-07-24, `4e1348adb`). It NEVER mutates stored data and NEVER guesses a
// model or a rate. The four statuses are mutually exclusive:
//
//   - reported : cost > 0 in storage → the sender attributed cost. Left untouched.
//   - estimated: tokens > 0, cost == 0, and the stage's model resolves to a
//     NON-ZERO metered rate → cost is computed tokens×rate and flagged.
//   - unknown  : tokens > 0, cost == 0, and the model is unresolvable → surfaced as
//     `unknown` (NEVER a fabricated metered $0). A model that resolves to a $0 rate
//     is `estimated` at $0 (free/local), not `unknown`.
//   - quota    : tokens == 0 (and cost == 0) → subscription/quota-lane spend that
//     surfaces NO reportable token count. $0-by-design. NEVER estimated.
//
// The model is not a stage column; it is recovered (in order) from the stage's
// EvalAssessment.Model, then any child span's Model. If none is found, a
// token-bearing stage is `unknown`.

// CostStatus is the provenance of a stage's cost in the rollup.
type CostStatus string

const (
	// CostStatusReported: the stage self-reported a non-zero cost.
	CostStatusReported CostStatus = "reported"
	// CostStatusEstimated: cost was inferred from tokens×rate (flagged, never stored).
	CostStatusEstimated CostStatus = "estimated"
	// CostStatusUnknown: token-bearing but no resolvable model — surfaced, never faked as $0.
	CostStatusUnknown CostStatus = "unknown"
	// CostStatusQuota: quota/subscription lane (no tokens) — $0-by-design, never estimated.
	CostStatusQuota CostStatus = "quota"
)

// StageCost is the classified cost of a single stage.
type StageCost struct {
	Status CostStatus `json:"status"`
	// CostUSD is the effective dollar cost for the rollup:
	//   - reported : the stored cost.
	//   - estimated: the computed tokens×rate (may be $0 for a free model).
	//   - unknown  : 0 (but MUST be surfaced as unknown, not as $0 metered spend).
	//   - quota    : 0.
	CostUSD float64 `json:"cost_usd"`
	// Estimated is true only for CostStatusEstimated (the `cost_estimated=true` flag).
	Estimated bool `json:"cost_estimated"`
	// Model is the model the classifier resolved for this stage (for diagnostics); "" if none.
	Model string `json:"model,omitempty"`
}

// resolveStageModel recovers the model for a stage without guessing.
// Order: EvalAssessment.Model, then the first child span with a non-empty Model.
// Returns "" if no model can be recovered.
func resolveStageModel(stage *ChainStage) string {
	if stage == nil {
		return ""
	}
	if stage.EvalAssessment != nil && stage.EvalAssessment.Model != "" {
		return stage.EvalAssessment.Model
	}
	for _, sp := range stage.Spans {
		if sp != nil && sp.Model != "" {
			return sp.Model
		}
	}
	return ""
}

// ClassifyStageCost classifies a single stage's cost under Mark's scoped rule.
// It is pure: it reads the stage (and any already-populated child spans) and never
// mutates storage. Model resolution uses resolveStageModel; unresolvable →
// `unknown`, never a fabricated rate.
func ClassifyStageCost(stage *ChainStage) StageCost {
	if stage == nil {
		return StageCost{Status: CostStatusQuota}
	}

	// 1. Self-reported cost is authoritative and never re-estimated.
	if stage.Cost > 0 {
		return StageCost{Status: CostStatusReported, CostUSD: stage.Cost}
	}

	// Data-integrity guard: negative token counts are corrupt upstream data (seen in
	// real banks: tokens_out as low as -13k). We MUST NOT turn corrupt tokens into a
	// negative dollar estimate — that is a fabricated/misleading figure. Surface it
	// as `unknown` so it is counted and warned about, never silently rolled into a
	// dollar total (CLAUDE.md: fail loudly on data-integrity issues).
	if stage.TokensIn < 0 || stage.TokensOut < 0 {
		return StageCost{Status: CostStatusUnknown}
	}

	totalTokens := int64(stage.TokensIn) + int64(stage.TokensOut)

	// 2. No tokens → quota/subscription lane. $0-by-design, never estimated.
	if totalTokens == 0 {
		return StageCost{Status: CostStatusQuota}
	}

	// 3. Token-bearing but no reported cost → estimate from tokens×rate IF the
	//    model resolves; otherwise `unknown` (never a fabricated $0).
	model := resolveStageModel(stage)
	if model == "" {
		return StageCost{Status: CostStatusUnknown}
	}

	cost, resolved := ResolveCostFromTokens(model, int64(stage.TokensIn), int64(stage.TokensOut))
	if !resolved {
		// Model recovered from the stage but not present in the pricing registry.
		return StageCost{Status: CostStatusUnknown, Model: model}
	}

	// A negative computed cost means the underlying token/rate data is inconsistent.
	// Never emit negative dollars — surface as unknown instead.
	if cost < 0 {
		return StageCost{Status: CostStatusUnknown, Model: model}
	}

	// Resolved: estimated (cost may be $0 for a free/local model — that rolls up to
	// $0 naturally, still flagged estimated so it is not conflated with reported).
	return StageCost{Status: CostStatusEstimated, CostUSD: cost, Estimated: true, Model: model}
}

// CostRollup aggregates classified per-stage costs into split totals so the CLI can
// present reported / estimated / quota / unknown separately (never conflated).
type CostRollup struct {
	ReportedCost  float64 `json:"reported_cost"`
	EstimatedCost float64 `json:"estimated_cost"`
	// UnknownCost is always 0 by construction (unknown is never given a dollar figure);
	// UnknownStages is the count that MUST trigger an incomplete-data warning.
	ReportedStages  int `json:"reported_stages"`
	EstimatedStages int `json:"estimated_stages"`
	QuotaStages     int `json:"quota_stages"`
	UnknownStages   int `json:"unknown_stages"`
	// Token counts split the same way (quota lanes carry no tokens by definition).
	ReportedTokens  int64 `json:"reported_tokens"`
	EstimatedTokens int64 `json:"estimated_tokens"`
	UnknownTokens   int64 `json:"unknown_tokens"`
}

// TotalKnownCost returns reported + estimated dollars (the credible total).
// Unknown stages contribute NO dollars — callers must surface UnknownStages
// separately as an incomplete-data warning rather than silently reading $0.
func (r CostRollup) TotalKnownCost() float64 {
	return r.ReportedCost + r.EstimatedCost
}

// HasIncompleteData reports whether any stage could not be attributed (unknown).
// The CLI emits a visible warning (or fails in strict mode) when true.
func (r CostRollup) HasIncompleteData() bool {
	return r.UnknownStages > 0
}

// AddStage classifies a stage and folds it into the rollup.
func (r *CostRollup) AddStage(stage *ChainStage) {
	sc := ClassifyStageCost(stage)
	tokens := int64(0)
	if stage != nil {
		tokens = int64(stage.TokensIn) + int64(stage.TokensOut)
	}
	switch sc.Status {
	case CostStatusReported:
		r.ReportedCost += sc.CostUSD
		r.ReportedStages++
		r.ReportedTokens += tokens
	case CostStatusEstimated:
		r.EstimatedCost += sc.CostUSD
		r.EstimatedStages++
		r.EstimatedTokens += tokens
	case CostStatusUnknown:
		r.UnknownStages++
		r.UnknownTokens += tokens
	case CostStatusQuota:
		r.QuotaStages++
	}
}

// RollupStages classifies and aggregates a slice of stages.
func RollupStages(stages []*ChainStage) CostRollup {
	var r CostRollup
	for _, s := range stages {
		r.AddStage(s)
	}
	return r
}
