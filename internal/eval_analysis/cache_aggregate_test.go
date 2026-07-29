package eval_analysis

import "testing"

// M-ANTHROPIC-CACHE-HIT-RATE M3 (deferred item): the hit rate has to be derived
// from the right denominator or it is worse than no number at all.
func TestCalculateAggregates_CacheHitRate(t *testing.T) {
	// A warm run: most input served from cache, a little uncached remainder.
	// Anthropic reports these as THREE separate buckets — r.InputTokens is the
	// uncached remainder only, NOT the total.
	results := []*BenchmarkResult{
		{InputTokens: 500, CacheReadInputTokens: 0, CacheCreationInputTokens: 16000, TotalTokens: 16800},
		{InputTokens: 500, CacheReadInputTokens: 16000, CacheCreationInputTokens: 0, TotalTokens: 16800},
		{InputTokens: 500, CacheReadInputTokens: 16000, CacheCreationInputTokens: 0, TotalTokens: 16800},
	}

	agg := calculateAggregates(results)

	if agg.CacheReadTokens != 32000 {
		t.Errorf("CacheReadTokens = %d, want 32000", agg.CacheReadTokens)
	}
	if agg.CacheCreationTokens != 16000 {
		t.Errorf("CacheCreationTokens = %d, want 16000", agg.CacheCreationTokens)
	}

	// Denominator = every input token seen: 1500 uncached + 32000 read + 16000 written.
	// Using InputTokens alone (1500) would report a nonsensical >2000%.
	wantRate := 32000.0 / 49500.0
	if diff := agg.CacheHitRate - wantRate; diff > 0.001 || diff < -0.001 {
		t.Errorf("CacheHitRate = %.4f, want %.4f (reads / all input tokens)", agg.CacheHitRate, wantRate)
	}
	if agg.CacheHitRate > 1.0 {
		t.Errorf("CacheHitRate = %.4f exceeds 100%% — wrong denominator", agg.CacheHitRate)
	}
}

// Pre-v0.31.0 baselines carry no cache fields; the rate must be 0, not NaN.
func TestCalculateAggregates_LegacyResultsNoNaN(t *testing.T) {
	results := []*BenchmarkResult{
		{InputTokens: 16000, TotalTokens: 16500},
		{InputTokens: 16000, TotalTokens: 16500},
	}
	agg := calculateAggregates(results)
	if agg.CacheHitRate != 0 {
		t.Errorf("CacheHitRate = %v, want 0 for pre-v0.31.0 results", agg.CacheHitRate)
	}
	if agg.CacheReadTokens != 0 || agg.CacheCreationTokens != 0 {
		t.Error("legacy results should aggregate to zero cache activity")
	}
}
