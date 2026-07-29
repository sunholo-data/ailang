package eval_analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

// FormatMatrix renders the human-readable summary. The cache line is
// conditional: it appears only when some provider actually reported activity,
// because a flat "0.0%" for a fleet that never reports would read as "caching is
// broken" rather than "nothing measured here".
func TestFormatMatrix_CacheLineIsConditional(t *testing.T) {
	withCache := &PerformanceMatrix{
		Version: "v0.31.0",
		Aggregates: Aggregates{
			TotalTokens:         50000,
			CacheReadTokens:     32000,
			CacheCreationTokens: 16000,
			CacheHitRate:        0.646,
		},
	}
	out := FormatMatrix(withCache, false)
	if !strings.Contains(out, "Cache hit rate") {
		t.Error("expected a cache hit rate line when cache activity was reported")
	}
	if !strings.Contains(out, "64.6%") {
		t.Errorf("expected the rate rendered as a percentage, got:\n%s", out)
	}

	noCache := &PerformanceMatrix{
		Version:    "v0.30.0",
		Aggregates: Aggregates{TotalTokens: 50000},
	}
	if got := FormatMatrix(noCache, false); strings.Contains(got, "Cache hit rate") {
		t.Error("cache line must be omitted when nothing reported cache activity — a bare 0.0% reads as a broken cache")
	}
}

// The JSON export is the dashboard feed — it is how the cache hit rate actually
// reaches a human. Pin that the three cache keys are emitted, so a rename or a
// dropped line shows up here rather than as a silently missing dashboard panel.
func TestExportBenchmarkJSON_EmitsCacheKeys(t *testing.T) {
	matrix := &PerformanceMatrix{
		Version: "v0.31.0",
		Aggregates: Aggregates{
			TotalTokens:         50000,
			CacheReadTokens:     32000,
			CacheCreationTokens: 16000,
			CacheHitRate:        0.646,
		},
	}
	out := filepath.Join(t.TempDir(), "latest.json")
	if _, err := ExportBenchmarkJSON(matrix, nil, nil, out); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	blob, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(blob, &got); err != nil {
		t.Fatalf("export is not valid JSON: %v", err)
	}

	body := string(blob)
	for _, key := range []string{"cacheReadTokens", "cacheCreationTokens", "cacheHitRate"} {
		if !strings.Contains(body, key) {
			t.Errorf("dashboard export is missing %q — the hit rate would never reach the dashboard", key)
		}
	}
}
