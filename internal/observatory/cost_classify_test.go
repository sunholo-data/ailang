package observatory

import (
	"os"
	"path/filepath"
	"testing"
)

// meteredModel resolves to a non-zero rate in models.yml.
const meteredModel = "claude-sonnet-4-5"

// zeroRateModel is a local (ollama) model with input_per_1k: 0.0 / output_per_1k: 0.0.
const zeroRateModel = "motoko-local-qwen3-5-35b-a3b-mxfp8"

// ensurePricingLoaded points the pricing loader at the repo models.yml. The
// observatory pricing loader searches relative paths; from the package test cwd
// (internal/observatory) the models.yml lives at ../eval_harness/models.yml, which
// is NOT one of the default search paths, so we chdir to the repo root for the test.
func ensurePricingLoaded(t *testing.T) {
	t.Helper()
	ResetPricingConfig()

	// Walk up from cwd to find the repo root (dir containing internal/eval_harness/models.yml).
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 6; i++ {
		if _, statErr := os.Stat(filepath.Join(dir, "internal", "eval_harness", "models.yml")); statErr == nil {
			cfg := GetPricingConfig()
			if cfg == nil {
				// Not loaded from default paths; load explicitly by chdir.
				old, _ := os.Getwd()
				if chErr := os.Chdir(dir); chErr != nil {
					t.Fatalf("chdir repo root: %v", chErr)
				}
				t.Cleanup(func() { _ = os.Chdir(old) })
				ResetPricingConfig()
				cfg = GetPricingConfig()
			}
			if cfg == nil {
				t.Skip("pricing config could not be loaded; skipping cost classifier test")
			}
			return
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("could not locate internal/eval_harness/models.yml; skipping")
}

func TestResolveCostFromTokens_DistinguishesUnresolvableFromZeroRate(t *testing.T) {
	ensurePricingLoaded(t)

	// Metered model: resolved=true, cost > 0.
	cost, resolved := ResolveCostFromTokens(meteredModel, 1_000_000, 1_000_000)
	if !resolved {
		t.Fatalf("expected metered model %q to resolve", meteredModel)
	}
	if cost <= 0 {
		t.Fatalf("expected non-zero cost for metered model, got %f", cost)
	}

	// Zero-rate local model: resolved=true, cost == 0 (free, NOT unknown).
	cost0, resolved0 := ResolveCostFromTokens(zeroRateModel, 1_000_000, 1_000_000)
	if !resolved0 {
		t.Fatalf("expected zero-rate model %q to resolve (found in registry)", zeroRateModel)
	}
	if cost0 != 0 {
		t.Fatalf("expected $0 cost for zero-rate model, got %f", cost0)
	}

	// Unresolvable model: resolved=false — MUST NOT be conflated with $0-rate.
	_, resolvedX := ResolveCostFromTokens("this-model-does-not-exist-xyz", 1_000_000, 0)
	if resolvedX {
		t.Fatalf("expected unresolvable model to report resolved=false")
	}

	// Empty model: resolved=false.
	if _, r := ResolveCostFromTokens("", 1000, 1000); r {
		t.Fatalf("empty model must be unresolvable")
	}
}

// Case (a): token-bearing, no reported cost, metered rate → estimated (NOT $0), flagged.
func TestClassifyStageCost_EstimatesMeteredNoCostStage(t *testing.T) {
	ensurePricingLoaded(t)

	stage := &ChainStage{
		Cost:      0,
		TokensIn:  1_000_000,
		TokensOut: 500_000,
		EvalAssessment: &EvalAssessment{
			Model: meteredModel,
		},
	}
	sc := ClassifyStageCost(stage)
	if sc.Status != CostStatusEstimated {
		t.Fatalf("expected estimated, got %s", sc.Status)
	}
	if !sc.Estimated {
		t.Fatalf("expected cost_estimated=true flag")
	}
	if sc.CostUSD <= 0 {
		t.Fatalf("expected a NON-ZERO estimated cost (never $0), got %f", sc.CostUSD)
	}
}

// Model recovered from a child span (not eval_assessment) also estimates.
func TestClassifyStageCost_EstimatesFromSpanModel(t *testing.T) {
	ensurePricingLoaded(t)

	stage := &ChainStage{
		Cost:      0,
		TokensIn:  200_000,
		TokensOut: 100_000,
		Spans: []*Span{
			{Model: meteredModel},
		},
	}
	sc := ClassifyStageCost(stage)
	if sc.Status != CostStatusEstimated || sc.CostUSD <= 0 {
		t.Fatalf("expected estimated non-zero from span model, got %s $%f", sc.Status, sc.CostUSD)
	}
	if sc.Model != meteredModel {
		t.Fatalf("expected model recovered from span, got %q", sc.Model)
	}
}

// Case (b): a stage WITH self-reported cost is reported and unchanged (no double-count / re-estimate).
func TestClassifyStageCost_ReportedUnchanged(t *testing.T) {
	ensurePricingLoaded(t)

	stage := &ChainStage{
		Cost:      1.2345,
		TokensIn:  9_000_000, // large tokens must NOT trigger re-estimation
		TokensOut: 1_000_000,
		EvalAssessment: &EvalAssessment{
			Model: meteredModel,
		},
	}
	sc := ClassifyStageCost(stage)
	if sc.Status != CostStatusReported {
		t.Fatalf("expected reported, got %s", sc.Status)
	}
	if sc.CostUSD != 1.2345 {
		t.Fatalf("reported cost must be untouched, got %f", sc.CostUSD)
	}
	if sc.Estimated {
		t.Fatalf("reported cost must NOT be flagged estimated")
	}
}

// Case (c1): tokens but no resolvable model → unknown (never a fabricated metered $0).
func TestClassifyStageCost_UnknownWhenModelUnresolvable(t *testing.T) {
	ensurePricingLoaded(t)

	// No eval_assessment, no spans → model unrecoverable.
	stage := &ChainStage{
		Cost:      0,
		TokensIn:  8_500_000,
		TokensOut: 0,
	}
	sc := ClassifyStageCost(stage)
	if sc.Status != CostStatusUnknown {
		t.Fatalf("expected unknown, got %s", sc.Status)
	}
	if sc.CostUSD != 0 {
		t.Fatalf("unknown must carry $0 (surfaced separately), got %f", sc.CostUSD)
	}
	if sc.Estimated {
		t.Fatalf("unknown must NOT be flagged estimated")
	}

	// Model recovered but not in the registry → still unknown, not $0 metered.
	stage2 := &ChainStage{
		Cost:           0,
		TokensIn:       1_000,
		TokensOut:      1_000,
		EvalAssessment: &EvalAssessment{Model: "no-such-model-abc"},
	}
	if sc2 := ClassifyStageCost(stage2); sc2.Status != CostStatusUnknown {
		t.Fatalf("expected unknown for unresolvable recovered model, got %s", sc2.Status)
	}
}

// Data-integrity: negative token counts (seen in real banks) must NEVER produce a
// negative dollar estimate — they surface as unknown, not fabricated metered spend.
func TestClassifyStageCost_NegativeTokensAreUnknownNotNegativeDollars(t *testing.T) {
	ensurePricingLoaded(t)

	stage := &ChainStage{
		Cost:           0,
		TokensIn:       1000,
		TokensOut:      -13479, // corrupt upstream data
		EvalAssessment: &EvalAssessment{Model: meteredModel},
	}
	sc := ClassifyStageCost(stage)
	if sc.Status != CostStatusUnknown {
		t.Fatalf("expected unknown for negative tokens, got %s", sc.Status)
	}
	if sc.CostUSD < 0 {
		t.Fatalf("must never emit negative dollars, got %f", sc.CostUSD)
	}
}

// Case (c2): a $0-rate (free/local) model rolls up to $0 as estimated, NOT unknown.
func TestClassifyStageCost_ZeroRateModelIsEstimatedZeroNotUnknown(t *testing.T) {
	ensurePricingLoaded(t)

	stage := &ChainStage{
		Cost:           0,
		TokensIn:       5_000_000,
		TokensOut:      2_000_000,
		EvalAssessment: &EvalAssessment{Model: zeroRateModel},
	}
	sc := ClassifyStageCost(stage)
	if sc.Status != CostStatusEstimated {
		t.Fatalf("expected estimated (free model), got %s", sc.Status)
	}
	if sc.CostUSD != 0 {
		t.Fatalf("expected $0 for free model, got %f", sc.CostUSD)
	}
}

// Case (d): a quota stage (tokens == 0) is NEVER estimated.
func TestClassifyStageCost_QuotaNeverEstimated(t *testing.T) {
	ensurePricingLoaded(t)

	stage := &ChainStage{
		Cost:      0,
		TokensIn:  0,
		TokensOut: 0,
		AgentID:   "sprint-executor (quota:opus)",
		// Even if a model is present, zero tokens must classify as quota, not estimated.
		EvalAssessment: &EvalAssessment{Model: meteredModel},
	}
	sc := ClassifyStageCost(stage)
	if sc.Status != CostStatusQuota {
		t.Fatalf("expected quota, got %s", sc.Status)
	}
	if sc.CostUSD != 0 || sc.Estimated {
		t.Fatalf("quota must be $0 and never estimated, got $%f estimated=%v", sc.CostUSD, sc.Estimated)
	}
}

// Rollup splits totals across the four statuses without conflation.
func TestRollupStages_SplitsTotals(t *testing.T) {
	ensurePricingLoaded(t)

	stages := []*ChainStage{
		{Cost: 2.0}, // reported
		{Cost: 0, TokensIn: 1_000_000, EvalAssessment: &EvalAssessment{Model: meteredModel}}, // estimated (>0)
		{Cost: 0, TokensIn: 1_000, TokensOut: 1_000},                                         // unknown (no model)
		{Cost: 0, TokensIn: 0, AgentID: "quota:opus"},                                        // quota
	}
	r := RollupStages(stages)

	if r.ReportedStages != 1 || r.ReportedCost != 2.0 {
		t.Fatalf("reported split wrong: stages=%d cost=%f", r.ReportedStages, r.ReportedCost)
	}
	if r.EstimatedStages != 1 || r.EstimatedCost <= 0 {
		t.Fatalf("estimated split wrong: stages=%d cost=%f", r.EstimatedStages, r.EstimatedCost)
	}
	if r.UnknownStages != 1 {
		t.Fatalf("expected 1 unknown stage, got %d", r.UnknownStages)
	}
	if r.QuotaStages != 1 {
		t.Fatalf("expected 1 quota stage, got %d", r.QuotaStages)
	}
	if !r.HasIncompleteData() {
		t.Fatalf("expected incomplete-data flag due to unknown stage")
	}
	if r.TotalKnownCost() != r.ReportedCost+r.EstimatedCost {
		t.Fatalf("TotalKnownCost must be reported+estimated only")
	}
}

// TestClassifyStageCost_Subscription pins the 2026-07-30 fix: a non-zero cost
// from a subscription lane is real arithmetic over real tokens that NOBODY was
// billed for, and must not land in a metered-dollars total.
func TestClassifyStageCost_Subscription(t *testing.T) {
	tests := []struct {
		name       string
		provenance string
		wantStatus CostStatus
	}{
		{"subscription lane splits out", "list-price-equivalent", CostStatusSubscription},
		{"metered stays reported", "metered", CostStatusReported},
		// Every row banked before the column existed. Unlabelled is NOT proof of
		// metering — but downgrading all history to unknown would destroy the
		// rollup, so it stays `reported` and the doc says what that means.
		{"unlabelled legacy row stays reported", "", CostStatusReported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyStageCost(&ChainStage{
				Cost: 0.34259375, TokensIn: 256987, TokensOut: 2136,
				CostProvenance: tt.provenance,
			})
			if got.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", got.Status, tt.wantStatus)
			}
			if got.CostUSD != 0.34259375 {
				t.Errorf("cost = %v, want the stored figure preserved", got.CostUSD)
			}
		})
	}
}

// TestCostRollup_SubscriptionExcludedFromKnownCost is the KPI guard: a cohort
// mixing a subscription stage with a metered one must not report their sum as
// spend. TotalKnownCost is the metered-dollars numerator.
func TestCostRollup_SubscriptionExcludedFromKnownCost(t *testing.T) {
	r := RollupStages([]*ChainStage{
		{Cost: 2.00, TokensIn: 1000, TokensOut: 100, CostProvenance: "metered"},
		{Cost: 8.36, TokensIn: 6241312, TokensOut: 55679, CostProvenance: "list-price-equivalent"},
	})
	if r.TotalKnownCost() != 2.00 {
		t.Errorf("TotalKnownCost = %v, want 2.00 (subscription must be excluded)", r.TotalKnownCost())
	}
	if r.SubscriptionCost != 8.36 {
		t.Errorf("SubscriptionCost = %v, want 8.36 (surfaced, not discarded)", r.SubscriptionCost)
	}
	if r.SubscriptionStages != 1 || r.ReportedStages != 1 {
		t.Errorf("stage split = %d subscription / %d reported, want 1/1", r.SubscriptionStages, r.ReportedStages)
	}
}
