package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

func TestEstimateRunCostUSDWithMeans_HandComputed(t *testing.T) {
	// Fixture: one benchmark with real history, one with none; one model
	// with known pricing, one with no pricing entry at all.
	means := &benchmarkTokenMeans{
		input:   map[string]float64{"has_history": 1000},
		output:  map[string]float64{"has_history": 500},
		covered: map[string]bool{"has_history": true},
	}
	cfg := &eval_harness.ModelsConfig{
		Models: map[string]eval_harness.ModelConfig{
			"priced-model": {
				Pricing: eval_harness.Pricing{InputPer1K: 0.01, OutputPer1K: 0.02},
			},
			// "unpriced-model" deliberately absent from cfg.Models
		},
	}

	est := estimateRunCostUSDWithMeans(
		[]string{"priced-model", "unpriced-model"},
		[]string{"has_history", "no_history"},
		2, // langs
		3, // trials
		means, cfg,
	)

	// priced-model x has_history: (1000*0.01 + 500*0.02)/1000 = (10+10)/1000 = 0.02 per run
	//   x 2 langs x 3 trials = 0.12
	// priced-model x no_history: defaults (3000*0.01 + 1500*0.02)/1000 = (30+30)/1000 = 0.06 per run
	//   x 2 langs x 3 trials = 0.36
	// unpriced-model x *: no pricing entry, contributes $0, counted separately
	wantTotal := 0.12 + 0.36
	if diff := est.TotalUSD - wantTotal; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("TotalUSD = %v, want %v", est.TotalUSD, wantTotal)
	}
	if est.Pairs != 4 {
		t.Errorf("Pairs = %d, want 4 (2 models x 2 benchmarks)", est.Pairs)
	}
	if est.NoHistoryPairs != 1 {
		t.Errorf("NoHistoryPairs = %d, want 1 (only priced-model x no_history counts — unpriced-model pairs short-circuit before the history check)", est.NoHistoryPairs)
	}
	if est.UnknownPricing != 2 {
		t.Errorf("UnknownPricing = %d, want 2 (unpriced-model x has_history, unpriced-model x no_history)", est.UnknownPricing)
	}
}

func TestEstimateRunCostUSDWithMeans_EmptyMeans(t *testing.T) {
	// nil means must not panic — every pair falls back to the flat default.
	cfg := &eval_harness.ModelsConfig{
		Models: map[string]eval_harness.ModelConfig{
			"m": {Pricing: eval_harness.Pricing{InputPer1K: 0.01, OutputPer1K: 0.01}},
		},
	}
	est := estimateRunCostUSDWithMeans([]string{"m"}, []string{"b"}, 1, 1, nil, cfg)
	want := (defaultMeanInputTokens*0.01 + defaultMeanOutputTokens*0.01) / 1000.0
	if diff := est.TotalUSD - want; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("TotalUSD = %v, want %v", est.TotalUSD, want)
	}
	if est.NoHistoryPairs != 1 {
		t.Errorf("NoHistoryPairs = %d, want 1", est.NoHistoryPairs)
	}
}

func TestEstimateRunCostUSDWithMeans_NilConfig(t *testing.T) {
	// nil cfg must not panic — every pair is UnknownPricing, $0 total.
	est := estimateRunCostUSDWithMeans([]string{"m"}, []string{"b1", "b2"}, 1, 1, nil, nil)
	if est.TotalUSD != 0 {
		t.Errorf("TotalUSD = %v, want 0", est.TotalUSD)
	}
	if est.UnknownPricing != 2 {
		t.Errorf("UnknownPricing = %d, want 2", est.UnknownPricing)
	}
}
