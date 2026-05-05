package eval_analysis

import (
	"sort"
)

// EfficiencyAggregates captures per-model speed-and-cost efficiency
// observables (M-EVAL-COST-AND-SPEED-BUDGETS, v0.15.1).
//
// All fields are summary statistics over the runs that succeeded
// (StdoutOk == true). Fields default to 0 when:
//   - No runs succeeded
//   - The underlying speed metric was never measured (executor pre-v0.15.1)
type EfficiencyAggregates struct {
	// Time-to-first-attempt: ms from task start to first solution submission.
	// Median across successful runs.
	MedianTimeToFirstAttemptMs float64 `json:"median_time_to_first_attempt_ms"`

	// Time-to-success: ms from task start to passing solution. Only counted
	// when SuccessAtMs > 0 (some executors leave it at -1, in which case
	// DurationMs is the fallback).
	MedianTimeToSuccessMs float64 `json:"median_time_to_success_ms"`

	// Median agent-mode turns to success (0 if not agent mode).
	MedianTurnsToSuccess float64 `json:"median_turns_to_success"`

	// Median tokens-per-second generation rate.
	MedianTokensPerSec float64 `json:"median_tokens_per_sec"`

	// 90th-percentile cost per success — flags expensive outliers.
	P90CostPerSuccess float64 `json:"p90_cost_per_success"`

	// Speed-efficiency score: dimensionless [0..1]. Higher = better
	// success-rate-per-second. Used as the y-axis on the Pareto frontier
	// chart. Computed as success_rate / (1 + median_TTS_seconds / 60).
	SpeedEfficiencyScore float64 `json:"speed_efficiency_score"`

	// CostKilledCount is the number of runs aborted because the cost
	// budget was exceeded. Distinct from api_error/timeout/logic_error.
	CostKilledCount int `json:"cost_killed_count"`
}

// ComputeEfficiency builds the per-model EfficiencyAggregates from a slice
// of result rows. The slice should already be filtered to one model — the
// caller is responsible for grouping.
//
// Returns the aggregate even when len(results) == 0; downstream callers
// can decide whether to omit it (we keep it present so dashboard JSONs
// remain shape-stable across versions).
func ComputeEfficiency(results []*BenchmarkResult) EfficiencyAggregates {
	if len(results) == 0 {
		return EfficiencyAggregates{}
	}

	// Collect per-success timing samples
	var ttfa, tts, turns, tps []float64
	var costPerSuccess []float64
	successCount := 0
	costKilled := 0

	for _, r := range results {
		if r.CostKilledAt > 0 {
			costKilled++
		}

		if !r.StdoutOk {
			continue
		}
		successCount++

		if r.FirstAttemptMs > 0 {
			ttfa = append(ttfa, float64(r.FirstAttemptMs))
		}
		// Prefer SuccessAtMs when measured; fall back to DurationMs
		// (post-hoc total) for executors that don't track it.
		switch {
		case r.SuccessAtMs > 0:
			tts = append(tts, float64(r.SuccessAtMs))
		case r.DurationMs > 0:
			tts = append(tts, float64(r.DurationMs))
		}
		if r.AgentTurns > 0 {
			turns = append(turns, float64(r.AgentTurns))
		}
		if r.TokensPerSec > 0 {
			tps = append(tps, r.TokensPerSec)
		}
		if r.CostUSD > 0 {
			costPerSuccess = append(costPerSuccess, r.CostUSD)
		}
	}

	successRate := float64(successCount) / float64(len(results))
	medianTTS := median(tts)
	speedScore := 0.0
	if successRate > 0 {
		// success_rate / (1 + median_TTS_seconds / 60)
		// 60s baseline so 1-minute solutions get score = success_rate / 2
		ttsSeconds := medianTTS / 1000.0
		speedScore = successRate / (1.0 + ttsSeconds/60.0)
	}

	return EfficiencyAggregates{
		MedianTimeToFirstAttemptMs: median(ttfa),
		MedianTimeToSuccessMs:      medianTTS,
		MedianTurnsToSuccess:       median(turns),
		MedianTokensPerSec:         median(tps),
		P90CostPerSuccess:          percentile(costPerSuccess, 0.9),
		SpeedEfficiencyScore:       speedScore,
		CostKilledCount:            costKilled,
	}
}

// median returns the median of a float64 slice. Returns 0 for an empty
// slice. Modifies the slice ordering — caller should pass a copy if
// preserving order matters.
func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sort.Float64s(xs)
	n := len(xs)
	if n%2 == 1 {
		return xs[n/2]
	}
	return (xs[n/2-1] + xs[n/2]) / 2.0
}

// percentile returns the p-th percentile of a float64 slice (p in [0..1]).
// Uses nearest-rank method. Returns 0 for an empty slice.
func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sort.Float64s(xs)
	n := len(xs)
	idx := int(float64(n) * p)
	if idx >= n {
		idx = n - 1
	}
	return xs[idx]
}

// CountCostKilled returns the count of cost-killed runs in the slice.
// Used to populate the "Top Error Codes" table (cost_killed becomes a
// distinct category).
func CountCostKilled(results []*BenchmarkResult) int {
	n := 0
	for _, r := range results {
		if r.CostKilledAt > 0 {
			n++
		}
	}
	return n
}
