package eval_harness

import "testing"

// TestCalculateCostForModelWithCache pins the cache-aware pricing added 2026-08-11.
//
// The bug this replaces: cache-read tokens were billed at $0 because no rate
// existed, a documented ~25-30% undercount that banked data could not verify.
func TestCalculateCostForModelWithCache(t *testing.T) {
	cfg := &ModelsConfig{Models: map[string]ModelConfig{
		"priced":  {Pricing: Pricing{InputPer1K: 0.00008, OutputPer1K: 0.00018, CacheReadPer1K: 0.000016}},
		"nocache": {Pricing: Pricing{InputPer1K: 0.00008, OutputPer1K: 0.00018}},
		"free":    {Pricing: Pricing{}},
	}}

	tests := []struct {
		name               string
		model              string
		in, out, cacheRead int
		want               float64
	}{
		// 1000 fresh in @0.08/M + 1000 out @0.18/M + 10000 cached @0.016/M
		{"cache priced separately", "priced", 1000, 1000, 10000, 0.00008 + 0.00018 + 0.00016},
		// Zero cache reads must equal the non-cache helper exactly, or every
		// pre-existing baseline silently reprices.
		{"no cache reads matches old math", "priced", 1000, 1000, 0, 0.00008 + 0.00018},
		// NO declared cache rate => bill at FULL input rate. Overstating is
		// visible in a budget; $0 hides both the spend and a broken cache.
		{"undeclared rate falls back to input rate", "nocache", 1000, 1000, 10000, 0.00008 + 0.00018 + 0.0008},
		{"free model stays free", "free", 1000, 1000, 10000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cfg.CalculateCostForModelWithCache(tt.model, tt.in, tt.out, tt.cacheRead)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := got - tt.want; diff > 1e-12 || diff < -1e-12 {
				t.Errorf("cost = %.10f, want %.10f", got, tt.want)
			}
		})
	}

	// No silent fallback on an unknown model, matching CalculateCostForModel.
	if _, err := cfg.CalculateCostForModelWithCache("nope", 1, 1, 1); err == nil {
		t.Error("expected an error for an unknown model, got nil")
	}
}

// TestCacheReadIsCheaperThanFreshInput is the property that motivated the work:
// the same token volume must cost strictly less when served from cache.
func TestCacheReadIsCheaperThanFreshInput(t *testing.T) {
	cfg := &ModelsConfig{Models: map[string]ModelConfig{
		"m": {Pricing: Pricing{InputPer1K: 0.00008, OutputPer1K: 0.00018, CacheReadPer1K: 0.000016}},
	}}
	fresh, _ := cfg.CalculateCostForModelWithCache("m", 27673, 100, 0)
	cached, _ := cfg.CalculateCostForModelWithCache("m", 281, 100, 27392)
	if !(cached < fresh) {
		t.Fatalf("cached run (%.8f) should cost less than fresh (%.8f)", cached, fresh)
	}
	if ratio := fresh / cached; ratio < 3.0 {
		t.Errorf("expected a large saving on a ~27.7k cached prompt, got %.2fx", ratio)
	}
}
