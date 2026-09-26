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
		"written": {Pricing: Pricing{InputPer1K: 0.00008, OutputPer1K: 0.00018, CacheWritePer1K: 0.0001}},
	}}

	tests := []struct {
		name                           string
		model                          string
		in, out, cacheRead, cacheWrite int
		want                           float64
	}{
		// 1000 fresh in @0.08/M + 1000 out @0.18/M + 10000 cached @0.016/M
		{"cache priced separately", "priced", 1000, 1000, 10000, 0, 0.00008 + 0.00018 + 0.00016},
		// Zero cache reads must equal the non-cache helper exactly, or every
		// pre-existing baseline silently reprices.
		{"no cache reads matches old math", "priced", 1000, 1000, 0, 0, 0.00008 + 0.00018},
		// NO declared cache rate => bill at FULL input rate. Overstating is
		// visible in a budget; $0 hides both the spend and a broken cache.
		{"undeclared rate falls back to input rate", "nocache", 1000, 1000, 10000, 0, 0.00008 + 0.00018 + 0.0008},
		{"free model stays free", "free", 1000, 1000, 10000, 10000, 0},
		// CACHE WRITES, added 2026-09-14. They had no parameter at all, so a cached
		// prompt was created for free. 10000 written @0.10/M declared.
		{"write priced separately", "written", 1000, 1000, 0, 10000, 0.00008 + 0.00018 + 0.001},
		// Undeclared write rate bills at the FULL input rate, same stance as reads.
		{"undeclared write rate falls back to input rate", "priced", 1000, 1000, 0, 10000, 0.00008 + 0.00018 + 0.0008},
		// Zero writes must reproduce the pre-change number exactly, or every banked
		// baseline silently reprices.
		{"zero writes matches old math", "priced", 1000, 1000, 10000, 0, 0.00008 + 0.00018 + 0.00016},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cfg.CalculateCostForModelWithCache(tt.model, tt.in, tt.out, tt.cacheRead, tt.cacheWrite)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := got - tt.want; diff > 1e-12 || diff < -1e-12 {
				t.Errorf("cost = %.10f, want %.10f", got, tt.want)
			}
		})
	}

	// No silent fallback on an unknown model, matching CalculateCostForModel.
	if _, err := cfg.CalculateCostForModelWithCache("nope", 1, 1, 1, 1); err == nil {
		t.Error("expected an error for an unknown model, got nil")
	}
}

// TestCacheReadIsCheaperThanFreshInput is the property that motivated the work:
// the same token volume must cost strictly less when served from cache.
func TestCacheReadIsCheaperThanFreshInput(t *testing.T) {
	cfg := &ModelsConfig{Models: map[string]ModelConfig{
		"m": {Pricing: Pricing{InputPer1K: 0.00008, OutputPer1K: 0.00018, CacheReadPer1K: 0.000016}},
	}}
	fresh, _ := cfg.CalculateCostForModelWithCache("m", 27673, 100, 0, 0)
	cached, _ := cfg.CalculateCostForModelWithCache("m", 281, 100, 27392, 0)
	if !(cached < fresh) {
		t.Fatalf("cached run (%.8f) should cost less than fresh (%.8f)", cached, fresh)
	}
	if ratio := fresh / cached; ratio < 3.0 {
		t.Errorf("expected a large saving on a ~27.7k cached prompt, got %.2fx", ratio)
	}
}

// The measured defect: a caching harness reports almost all of its paid-for prompt as
// cache CREATION, and that bucket used to cost nothing.
//
// Numbers are a real role-run, 2026-09-14: claude-haiku-4-5 reading five files reported
// InputTokens=50, CacheCreationInputTokens=44,841, CacheReadInputTokens=172,565.
func TestCacheWritesWereBilledAtZero(t *testing.T) {
	cfg := &ModelsConfig{Models: map[string]ModelConfig{
		"m": {Pricing: Pricing{InputPer1K: 0.003, OutputPer1K: 0.015, CacheReadPer1K: 0.0003}},
	}}
	const in, out, cacheRead, cacheWrite = 50, 648, 172565, 44841

	withWrites, err := cfg.CalculateCostForModelWithCache("m", in, out, cacheRead, cacheWrite)
	if err != nil {
		t.Fatal(err)
	}
	asBefore, err := cfg.CalculateCostForModelWithCache("m", in, out, cacheRead, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !(withWrites > asBefore) {
		t.Fatal("cache writes still cost nothing")
	}
	// 44,841 tokens at the undeclared-write fallback (the input rate, $3/M) is $0.1345 —
	// more than the entire rest of the run, which is why $0 was not a rounding error.
	if delta := withWrites - asBefore; delta < 0.13 || delta > 0.14 {
		t.Fatalf("write cost = $%.4f, want ~$0.1345", delta)
	}
	t.Logf("same run: $%.4f before, $%.4f with writes priced", asBefore, withWrites)
}
