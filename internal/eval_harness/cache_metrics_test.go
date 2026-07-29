package eval_harness

import (
	"encoding/json"
	"testing"
)

// M-ANTHROPIC-CACHE-HIT-RATE M3.
//
// Before v0.31.0 cache tokens were modelled on ai.Response but never persisted:
// zero of 28,139 banked result files carried them, so our own cache hit rate was
// invisible in our own data. These tests pin the two properties that keep it
// visible AND keep old baselines readable.

// TestRunMetrics_CacheTokensRoundTrip: the fields survive a JSON round-trip, so
// a warm run's hit rate is actually recoverable from a banked file.
func TestRunMetrics_CacheTokensRoundTrip(t *testing.T) {
	m := NewRunMetrics("bench1", "ailang", "claude-sonnet-5", 42)
	m.InputTokens = 500
	m.CacheReadInputTokens = 16000
	m.CacheCreationInputTokens = 0

	blob, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got RunMetrics
	if err := json.Unmarshal(blob, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.CacheReadInputTokens != 16000 {
		t.Errorf("cache_read_input_tokens = %d, want 16000", got.CacheReadInputTokens)
	}
	if got.CacheCreationInputTokens != 0 {
		t.Errorf("cache_creation_input_tokens = %d, want 0", got.CacheCreationInputTokens)
	}
}

// TestRunMetrics_PreV031BaselinesStillParse: the fields are additive and
// omitempty, so a baseline banked before v0.31.0 (which has neither key) must
// still load, reading them as 0 rather than erroring. Without this, adding the
// metric would break every historical comparison.
func TestRunMetrics_PreV031BaselinesStillParse(t *testing.T) {
	// A pre-v0.31.0 record: no cache keys at all.
	legacy := `{
		"id": "bench1", "lang": "ailang", "model": "claude-sonnet-5", "seed": 42,
		"input_tokens": 12000, "output_tokens": 300, "total_tokens": 12300,
		"cost_usd": 0.05, "compile_ok": true, "runtime_ok": true, "stdout_ok": true
	}`

	var m RunMetrics
	if err := json.Unmarshal([]byte(legacy), &m); err != nil {
		t.Fatalf("pre-v0.31.0 baseline failed to parse: %v", err)
	}
	if m.InputTokens != 12000 {
		t.Errorf("InputTokens = %d, want 12000", m.InputTokens)
	}
	if m.CacheReadInputTokens != 0 || m.CacheCreationInputTokens != 0 {
		t.Errorf("absent cache fields should read as 0, got read=%d create=%d",
			m.CacheReadInputTokens, m.CacheCreationInputTokens)
	}
}

// TestRunMetrics_CacheFieldsOmittedWhenZero keeps rows for providers that do not
// report cache activity free of misleading zero-valued keys.
func TestRunMetrics_CacheFieldsOmittedWhenZero(t *testing.T) {
	m := NewRunMetrics("bench1", "ailang", "gemma4:26b", 1)
	blob, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(blob, &generic); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, present := generic["cache_read_input_tokens"]; present {
		t.Error("cache_read_input_tokens should be omitted when zero")
	}
}
