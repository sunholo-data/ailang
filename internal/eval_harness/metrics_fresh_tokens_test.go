package eval_harness

import "testing"

// TestFreshTokens pins the uncached-tokens KPI (Mark, 2026-08-11): the token
// metric must count work the provider actually did, so that caching — the
// behaviour we want — is not penalised.
func TestFreshTokens(t *testing.T) {
	tests := []struct {
		name       string
		m          RunMetrics
		wantTokens int
		wantOK     bool
	}{
		{
			// input_tokens is cache-INCLUSIVE, so fresh = 100000-75000-1000 = 24000,
			// plus 2000 output + 500 reasoning.
			name: "cache split subtracted",
			m: RunMetrics{
				InputTokens: 100000, CacheReadInputTokens: 75000, CacheCreationInputTokens: 1000,
				OutputTokens: 2000, ReasonTokens: 500, CacheAccounted: true,
			},
			wantTokens: 26500, wantOK: true,
		},
		{
			// A local model with no prompt cache is a REAL zero, and must count.
			name: "accounted zero cache counts fully",
			m: RunMetrics{
				InputTokens: 10000, OutputTokens: 500, CacheAccounted: true,
			},
			wantTokens: 10500, wantOK: true,
		},
		{
			// THE TRAP. Pre-2026-08-11 agent rows have no cache fields. Treating
			// absent-as-zero would score them 100% fresh, inflating every
			// historical row and manufacturing an improvement out of a schema
			// change. They must be refused, not counted.
			name: "unaccounted row is refused, NOT treated as all-fresh",
			m: RunMetrics{
				InputTokens: 100000, OutputTokens: 2000, // no CacheAccounted
			},
			wantTokens: 0, wantOK: false,
		},
		{
			// Contract violation: input_tokens must be cache-inclusive. If a
			// producer breaks that, refuse rather than report negative tokens.
			name: "impossible split refuses",
			m: RunMetrics{
				InputTokens: 100, CacheReadInputTokens: 5000, OutputTokens: 10, CacheAccounted: true,
			},
			wantTokens: 0, wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.m.FreshTokens()
			if ok != tt.wantOK || got != tt.wantTokens {
				t.Errorf("FreshTokens() = (%d, %v), want (%d, %v)", got, ok, tt.wantTokens, tt.wantOK)
			}
		})
	}
}

// TestFreshTokensRewardsCaching is the property the KPI change exists for: two
// runs doing identical real work must score the SAME, even when one served most
// of its prompt from cache. Under the old TotalTokens metric the cached run
// scored ~4x worse.
func TestFreshTokensRewardsCaching(t *testing.T) {
	uncached := RunMetrics{InputTokens: 30000, OutputTokens: 1000, CacheAccounted: true}
	cached := RunMetrics{InputTokens: 200000, CacheReadInputTokens: 170000, OutputTokens: 1000, CacheAccounted: true}

	u, okU := uncached.FreshTokens()
	c, okC := cached.FreshTokens()
	if !okU || !okC {
		t.Fatal("both rows are accounted; both should resolve")
	}
	if u != c {
		t.Errorf("equal real work should score equally: uncached=%d cached=%d", u, c)
	}
	// And the old metric would have punished the cached run — guard the regression.
	if cached.InputTokens+cached.OutputTokens <= uncached.InputTokens+uncached.OutputTokens {
		t.Fatal("test setup no longer reproduces the distortion it guards against")
	}
}
