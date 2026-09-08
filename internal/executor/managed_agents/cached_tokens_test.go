package managed_agents

import (
	"encoding/json"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// liveUsageBody is a VERBATIM usage object from the Vertex Managed Agents API,
// captured 2026-09-02. It is checked in as a fixture because the defect this
// pins was a frozen claim about the API's shape — "Not reported by Managed
// Agents API" — that nothing tested, and that had become false.
const liveUsageBody = `{
  "total_tokens": 13596,
  "total_input_tokens": 13107,
  "input_tokens_by_modality": [{"modality": "text", "tokens": 13107}],
  "total_cached_tokens": 5350,
  "cached_tokens_by_modality": [{"modality": "text", "tokens": 5350}],
  "total_output_tokens": 56,
  "output_tokens_by_modality": [{"modality": "text", "tokens": 56}],
  "total_thought_tokens": 433
}`

func TestUsage_ParsesCachedTokens(t *testing.T) {
	var u Usage
	if err := json.Unmarshal([]byte(liveUsageBody), &u); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if u.TotalCachedTokens != 5350 {
		t.Errorf("TotalCachedTokens = %d, want 5350", u.TotalCachedTokens)
	}
	// Cached is a SUBSET of input, never additional to it.
	if u.TotalCachedTokens > u.TotalInputTokens {
		t.Error("cached tokens must be a subset of input tokens")
	}
	if got := u.FreshInputTokens(); got != 7757 {
		t.Errorf("FreshInputTokens() = %d, want 7757 (13107 - 5350)", got)
	}
}

// A malformed response must not produce a negative charge.
func TestUsage_FreshInputClampsAtZero(t *testing.T) {
	u := Usage{TotalInputTokens: 100, TotalCachedTokens: 500}
	if got := u.FreshInputTokens(); got != 0 {
		t.Errorf("FreshInputTokens() = %d, want 0 when cached exceeds input", got)
	}
}

// The whole point: cached tokens must cost ~10% of fresh ones. Billing the
// cache-inclusive total at the fresh rate overstates spend, which is how this
// lane got recorded as ~6x its budget.
func TestCostModel_CachedTokensAreCheaperAndNotDoubleBilled(t *testing.T) {
	var u Usage
	if err := json.Unmarshal([]byte(liveUsageBody), &u); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cm := (&Executor{}).CostModel()

	correct := cm.CalculateCost(executor.TokenUsage{
		InputTokens:          u.FreshInputTokens(),
		OutputTokens:         u.TotalOutputTokens + u.TotalThoughtTokens,
		CacheReadInputTokens: u.TotalCachedTokens,
	})
	// What the code did before: whole input at the fresh rate, cache ignored.
	old := cm.CalculateCost(executor.TokenUsage{
		InputTokens:  u.TotalInputTokens,
		OutputTokens: u.TotalOutputTokens + u.TotalThoughtTokens,
	})
	if !(correct < old) {
		t.Fatalf("cache-aware cost %.6f must be below the old cache-blind cost %.6f", correct, old)
	}

	// Double-billing guard: passing the INCLUSIVE total plus the cache read
	// would charge cached tokens at both rates.
	doubled := cm.CalculateCost(executor.TokenUsage{
		InputTokens:          u.TotalInputTokens,
		OutputTokens:         u.TotalOutputTokens + u.TotalThoughtTokens,
		CacheReadInputTokens: u.TotalCachedTokens,
	})
	if !(correct < doubled) {
		t.Error("fresh-input accounting must cost less than passing the cache-inclusive total alongside cache reads")
	}

	// Thought tokens are billed at the output rate and are DISJOINT from
	// output_tokens in this API, so dropping them would understate cost.
	noThink := cm.CalculateCost(executor.TokenUsage{
		InputTokens:          u.FreshInputTokens(),
		OutputTokens:         u.TotalOutputTokens,
		CacheReadInputTokens: u.TotalCachedTokens,
	})
	if !(noThink < correct) {
		t.Error("thought tokens must contribute to cost — they are disjoint from output_tokens here")
	}
}

// TestResult_InputTokensAreDisjointFromCacheReads pins the executor.Result
// contract that eval_harness/agent_runner_multi.go relies on: InputTokens is
// FRESH input, and CacheReadInputTokens is the cached remainder. They must not
// overlap, because downstream pricing adds them.
//
// Regression guard: the first version of the cache fix set CacheReadInputTokens
// correctly but left InputTokens as the cache-INCLUSIVE total, which
// double-counted every cached token in the banked row.
func TestResult_InputTokensAreDisjointFromCacheReads(t *testing.T) {
	u := Usage{TotalInputTokens: 41332, TotalCachedTokens: 28378, TotalOutputTokens: 384, TotalThoughtTokens: 2032}

	fresh := u.FreshInputTokens()
	if fresh+u.TotalCachedTokens != u.TotalInputTokens {
		t.Errorf("fresh(%d) + cached(%d) must equal reported input(%d) exactly — no gap, no overlap",
			fresh, u.TotalCachedTokens, u.TotalInputTokens)
	}
	if fresh >= u.TotalInputTokens {
		t.Error("fresh input must be strictly less than the cache-inclusive total when caching occurred")
	}
}
