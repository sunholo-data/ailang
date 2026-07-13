package eval_analysis

import (
	"sort"
)

// SweetSpotOpts controls the sweet-spot bucketing thresholds.
type SweetSpotOpts struct {
	// SlowMs is the median-time-to-success threshold (in ms) above which a
	// benchmark is classified as "slow_pass" rather than "fast_pass" for a
	// given model. Defaults to 60000 (60s) when zero.
	SlowMs int64
}

// defaultOpts fills in zero-valued fields with documented defaults.
func (o SweetSpotOpts) defaultOpts() SweetSpotOpts {
	if o.SlowMs <= 0 {
		o.SlowMs = 60_000
	}
	return o
}

// SweetSpotBucket counts benchmarks per outcome category for one (model, harness).
type SweetSpotBucket struct {
	// FastPass: model passes the benchmark with median TTS ≤ SlowMs.
	FastPass int `json:"fast_pass"`
	// SlowPass: model passes but median TTS > SlowMs — "give it more time" wins.
	SlowPass int `json:"slow_pass"`
	// BudgetBlocked: failed with cost_killed or step_exhausted — more budget
	// might make it pass. These are real capability-adjacent signals.
	BudgetBlocked int `json:"budget_blocked"`
	// CapabilityBlocked: compile/runtime/logic/timeout failures — model
	// genuinely couldn't solve it within reasonable scope.
	CapabilityBlocked int `json:"capability_blocked"`
	// Refused: the model's safety layer declined the prompt (ErrorCategory
	// "refused", e.g. Fable 5's deterministic refusals). Model behavior, NOT a
	// coding failure — its own bucket, and excluded from capability scoring
	// (see capability.go) just like provider noise.
	Refused int `json:"refused"`
	// ProviderBlocked: quota_exhausted / rate_limit / api_error — not the
	// model's fault. Excluded from capability scoring (see capability.go).
	ProviderBlocked int `json:"provider_blocked"`
}

// SweetSpotRow is one model's full sweet-spot profile.
type SweetSpotRow struct {
	Model     string `json:"model"`
	Harness   string `json:"harness,omitempty"` // executor name (claude/motoko/api/etc.)
	TotalRuns int    `json:"total_runs"`

	// Capability-aware pass rate: passes / (total - provider_blocked).
	// 0 when every run was provider-blocked.
	PassRate float64 `json:"pass_rate"`

	// Efficiency (mirrors EfficiencyAggregates, but flattened for table output).
	MedianTTSMs        float64 `json:"median_tts_ms"`
	MedianTokensPerSec float64 `json:"median_tokens_per_sec"`
	P90CostPerSuccess  float64 `json:"p90_cost_per_success"`
	SpeedEfficiency    float64 `json:"speed_efficiency"`

	// DollarsPerPass is total $ across all runs / number of passes. The
	// headline economic metric for the dashboard. 0 when no passes.
	// M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION (v0.19.0).
	DollarsPerPass float64 `json:"dollars_per_pass"`

	// CostOverheadVsBest is the median ratio (this_model_cost / best_passer_cost)
	// computed per benchmark this model passed. 1.0 = matched the cheapest passer
	// on every benchmark; 5.0 = on average spent 5x more $ than the cheapest
	// passer. 0 when the model passed no benchmarks. The "best" denominator is
	// per-benchmark min(CostUSD) across all models that passed THAT benchmark.
	// Captures "if a perfect router picked the cheapest model per benchmark, how
	// much more would this model cost than that router?"
	CostOverheadVsBest float64 `json:"cost_overhead_vs_best"`

	// TokenOverheadVsBest is the same shape but for TotalTokens. Distinguishes
	// "expensive because of per-token pricing" (high $-overhead, low token-overhead)
	// from "expensive because of inefficient iteration" (high on both).
	TokenOverheadVsBest float64 `json:"token_overhead_vs_best"`

	// ParetoFrontier is true when no other model has BOTH lower $/win AND
	// lower median TTS. Computed across all rows in the same SweetSpotReport.
	// M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION (v0.19.0).
	ParetoFrontier bool `json:"pareto_frontier"`

	// Failure-cause counts (from the typed ErrorCategory taxonomy).
	CostKilledCount    int `json:"cost_killed_count"`
	StepExhaustedCount int `json:"step_exhausted_count"`
	TimeoutCount       int `json:"timeout_count"`
	QuotaCount         int `json:"quota_count"`
	RateLimitCount     int `json:"rate_limit_count"`
	APIErrorCount      int `json:"api_error_count"`

	// FinishReasons counts the executor finish_reason values seen across
	// this model's runs. Empty/missing finish_reason counts as "".
	// M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION (v0.19.0).
	FinishReasons map[string]int `json:"finish_reasons,omitempty"`

	Buckets SweetSpotBucket `json:"buckets"`
}

// BenchmarkChampion records, per benchmark, the cheapest and fastest model
// that passed it. Drives the "cheapest pass" / "fastest pass" footer.
type BenchmarkChampion struct {
	BenchmarkID   string  `json:"benchmark_id"`
	CheapestModel string  `json:"cheapest_model"`
	CheapestCost  float64 `json:"cheapest_cost_usd"`
	CheapestTTSMs float64 `json:"cheapest_tts_ms"`
	FastestModel  string  `json:"fastest_model"`
	FastestTTSMs  float64 `json:"fastest_tts_ms"`
	FastestCost   float64 `json:"fastest_cost_usd"`
}

// SweetSpotReport is the full sweet-spot output.
type SweetSpotReport struct {
	Rows      []SweetSpotRow      `json:"rows"`
	Champions []BenchmarkChampion `json:"champions"`
	SlowMs    int64               `json:"slow_threshold_ms"`
	TotalRuns int                 `json:"total_runs"`
}

// BuildSweetSpot groups results by (Model, Executor) and produces a
// ranked SweetSpotReport. Use the executor field as the harness label when
// present; otherwise the row's Harness is empty and the model name carries
// the full identity (matches existing FormatMatrix conventions).
//
// Pure function — no I/O. Safe for concurrent use; does not modify results.
func BuildSweetSpot(results []*BenchmarkResult, opts SweetSpotOpts) SweetSpotReport {
	opts = opts.defaultOpts()

	type key struct{ model, harness string }
	byKey := map[key][]*BenchmarkResult{}
	for _, r := range results {
		byKey[key{r.Model, r.Executor}] = append(byKey[key{r.Model, r.Executor}], r)
	}

	rows := make([]SweetSpotRow, 0, len(byKey))
	for k, rs := range byKey {
		rows = append(rows, buildRow(k.model, k.harness, rs, opts))
	}

	// Per-benchmark "best passer" cost and tokens — used to compute each
	// model's overhead vs the optimal allocator (price-of-anarchy framing).
	// Only PASSING runs contribute to the denominator: failing runs don't
	// have a meaningful "cost" and would skew the floor downward unfairly.
	bestCostByBench := map[string]float64{}
	bestTokensByBench := map[string]int{}
	for _, r := range results {
		if !r.StdoutOk {
			continue
		}
		// Only count runs with positive cost/tokens — zero values mean the
		// executor never charged us (motoko local, api_error early-out, etc.)
		// and would create a degenerate 0 floor.
		if r.CostUSD > 0 {
			if cur, ok := bestCostByBench[r.ID]; !ok || r.CostUSD < cur {
				bestCostByBench[r.ID] = r.CostUSD
			}
		}
		if r.TotalTokens > 0 {
			if cur, ok := bestTokensByBench[r.ID]; !ok || r.TotalTokens < cur {
				bestTokensByBench[r.ID] = r.TotalTokens
			}
		}
	}
	// Group passing runs per (model, harness) and compute the overhead median.
	for i := range rows {
		ri := &rows[i]
		rs := byKey[key{ri.Model, ri.Harness}]
		var costRatios, tokenRatios []float64
		for _, r := range rs {
			if !r.StdoutOk {
				continue
			}
			if r.CostUSD > 0 {
				if best, ok := bestCostByBench[r.ID]; ok && best > 0 {
					costRatios = append(costRatios, r.CostUSD/best)
				}
			}
			if r.TotalTokens > 0 {
				if best, ok := bestTokensByBench[r.ID]; ok && best > 0 {
					tokenRatios = append(tokenRatios, float64(r.TotalTokens)/float64(best))
				}
			}
		}
		ri.CostOverheadVsBest = median(costRatios)
		ri.TokenOverheadVsBest = median(tokenRatios)
	}

	// M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION (v0.19.0): Pareto-frontier
	// classification across all rows. A row is on the frontier if no
	// other row has BOTH lower P90CostPerSuccess AND lower MedianTTSMs.
	// Rows with zero values (no successful runs) are NEVER on the
	// frontier — they can't be the "best" at anything.
	for i := range rows {
		ri := &rows[i]
		if ri.P90CostPerSuccess <= 0 || ri.MedianTTSMs <= 0 {
			ri.ParetoFrontier = false
			continue
		}
		dominated := false
		for j := range rows {
			if i == j {
				continue
			}
			rj := rows[j]
			if rj.P90CostPerSuccess <= 0 || rj.MedianTTSMs <= 0 {
				continue
			}
			if rj.P90CostPerSuccess <= ri.P90CostPerSuccess &&
				rj.MedianTTSMs <= ri.MedianTTSMs &&
				(rj.P90CostPerSuccess < ri.P90CostPerSuccess || rj.MedianTTSMs < ri.MedianTTSMs) {
				dominated = true
				break
			}
		}
		ri.ParetoFrontier = !dominated
	}

	sort.SliceStable(rows, func(i, j int) bool {
		// Primary: SpeedEfficiency desc. Secondary: PassRate desc. Tertiary: Model asc.
		if rows[i].SpeedEfficiency != rows[j].SpeedEfficiency {
			return rows[i].SpeedEfficiency > rows[j].SpeedEfficiency
		}
		if rows[i].PassRate != rows[j].PassRate {
			return rows[i].PassRate > rows[j].PassRate
		}
		return rows[i].Model < rows[j].Model
	})

	champions := buildChampions(results)

	return SweetSpotReport{
		Rows:      rows,
		Champions: champions,
		SlowMs:    opts.SlowMs,
		TotalRuns: len(results),
	}
}

func buildRow(model, harness string, rs []*BenchmarkResult, opts SweetSpotOpts) SweetSpotRow {
	row := SweetSpotRow{
		Model:     model,
		Harness:   harness,
		TotalRuns: len(rs),
	}
	if len(rs) == 0 {
		return row
	}

	// Median TTS per-benchmark grouping for fast/slow bucket classification.
	// A model passes a benchmark if ANY run of that benchmark succeeded;
	// the bucket comes from the median TTS across the passing runs.
	type benchAgg struct {
		passed   bool
		ttsTotal float64
		ttsCount int
	}
	byBench := map[string]*benchAgg{}

	passes := 0
	excluded := 0
	totalCost := 0.0
	finishReasons := map[string]int{}

	for _, r := range rs {
		if ShouldExcludeFromCapability(r.ErrorCategory) {
			excluded++
		}
		totalCost += r.CostUSD
		finishReasons[r.FinishReason]++

		// Failure-cause tally
		switch r.ErrorCategory {
		case "cost_killed":
			row.CostKilledCount++
		case "step_exhausted":
			row.StepExhaustedCount++
		case "timeout":
			row.TimeoutCount++
		case "quota_exhausted":
			row.QuotaCount++
		case "rate_limit":
			row.RateLimitCount++
		case "api_error":
			row.APIErrorCount++
		}

		agg := byBench[r.ID]
		if agg == nil {
			agg = &benchAgg{}
			byBench[r.ID] = agg
		}

		if r.StdoutOk {
			passes++
			agg.passed = true
			switch {
			case r.SuccessAtMs > 0:
				agg.ttsTotal += float64(r.SuccessAtMs)
				agg.ttsCount++
			case r.DurationMs > 0:
				agg.ttsTotal += float64(r.DurationMs)
				agg.ttsCount++
			}
		}
	}

	// Capability-aware pass rate: passes / (total - excluded). Drops to 0
	// when every run was provider-blocked.
	denominator := len(rs) - excluded
	if denominator > 0 {
		row.PassRate = float64(passes) / float64(denominator)
	}

	// Efficiency aggregates use the same per-(model) results.
	eff := ComputeEfficiency(rs)
	row.MedianTTSMs = eff.MedianTimeToSuccessMs
	row.MedianTokensPerSec = eff.MedianTokensPerSec
	row.P90CostPerSuccess = eff.P90CostPerSuccess
	row.SpeedEfficiency = eff.SpeedEfficiencyScore

	// M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION (v0.19.0): per-pass economics
	// and finish_reason breakdown. DollarsPerPass is the headline website
	// number; FinishReasons is the executor-stop diagnostic.
	if passes > 0 {
		row.DollarsPerPass = totalCost / float64(passes)
	}
	row.FinishReasons = finishReasons

	// Bucket each benchmark for this model.
	for _, r := range rs {
		agg := byBench[r.ID]
		if agg == nil {
			continue
		}

		// Each (model, benchmark) gets bucketed once. Defer to a per-bench
		// label so we don't double-count across multiple runs of the same
		// benchmark/model pair.
	}
	// Iterate over the per-benchmark aggregates to assign exactly-one bucket.
	for benchID, agg := range byBench {
		_ = benchID
		switch {
		case agg.passed:
			if agg.ttsCount > 0 && agg.ttsTotal/float64(agg.ttsCount) > float64(opts.SlowMs) {
				row.Buckets.SlowPass++
			} else {
				row.Buckets.FastPass++
			}
		default:
			// Look at the failure modes for this benchmark across runs for
			// this model. Priority: budget > capability > provider.
			budget := false
			capability := false
			refused := false
			provider := false
			for _, r := range rs {
				if r.ID != benchID || r.StdoutOk {
					continue
				}
				switch r.ErrorCategory {
				case "cost_killed", "step_exhausted":
					budget = true
				case "timeout", "compile_error", "runtime_error", "logic_error", "verify_error":
					capability = true
				case "refused":
					refused = true
				case "quota_exhausted", "rate_limit", "api_error":
					provider = true
				default:
					capability = true
				}
			}
			switch {
			case budget:
				row.Buckets.BudgetBlocked++
			case capability:
				row.Buckets.CapabilityBlocked++
			case refused:
				row.Buckets.Refused++
			case provider:
				row.Buckets.ProviderBlocked++
			}
		}
	}

	return row
}

// buildChampions finds the cheapest-to-pass and fastest-to-pass model per
// benchmark, considering only successful (StdoutOk=true) runs.
func buildChampions(results []*BenchmarkResult) []BenchmarkChampion {
	type candidate struct {
		model string
		cost  float64
		ttsMs float64
	}
	byBench := map[string][]candidate{}
	for _, r := range results {
		if !r.StdoutOk {
			continue
		}
		tts := float64(r.SuccessAtMs)
		if tts <= 0 {
			tts = float64(r.DurationMs)
		}
		if tts <= 0 {
			continue
		}
		byBench[r.ID] = append(byBench[r.ID], candidate{
			model: modelLabel(r),
			cost:  r.CostUSD,
			ttsMs: tts,
		})
	}

	champions := make([]BenchmarkChampion, 0, len(byBench))
	for bench, cands := range byBench {
		if len(cands) == 0 {
			continue
		}
		ch := BenchmarkChampion{BenchmarkID: bench}

		// Cheapest = min cost; tie-break on tts.
		cheapest := cands[0]
		fastest := cands[0]
		for _, c := range cands[1:] {
			if c.cost < cheapest.cost || (c.cost == cheapest.cost && c.ttsMs < cheapest.ttsMs) {
				cheapest = c
			}
			if c.ttsMs < fastest.ttsMs || (c.ttsMs == fastest.ttsMs && c.cost < fastest.cost) {
				fastest = c
			}
		}
		ch.CheapestModel = cheapest.model
		ch.CheapestCost = cheapest.cost
		ch.CheapestTTSMs = cheapest.ttsMs
		ch.FastestModel = fastest.model
		ch.FastestTTSMs = fastest.ttsMs
		ch.FastestCost = fastest.cost
		champions = append(champions, ch)
	}
	sort.SliceStable(champions, func(i, j int) bool {
		return champions[i].BenchmarkID < champions[j].BenchmarkID
	})
	return champions
}

func modelLabel(r *BenchmarkResult) string {
	if r.Executor != "" && r.Executor != r.Model {
		return r.Model + "/" + r.Executor
	}
	return r.Model
}

// renderSweetSpotRow converts a SweetSpotRow into the camelCase-keyed map
// shape used in the dashboard JSON output (M-EVAL-SWEET-SPOT-WEBSITE-
// INTEGRATION v0.19.0). Mirrors the JSX field names that the new
// DollarsPerPassTable / BenchmarkChampionsTable / FailureCategoryBars
// components read.
//
// All numeric fields are emitted unconditionally so consumers can rely on
// shape stability — zero values mean "not measured / not applicable".
func renderSweetSpotRow(r SweetSpotRow) map[string]interface{} {
	return map[string]interface{}{
		"model":                  r.Model,
		"harness":                r.Harness,
		"total_runs":             r.TotalRuns,
		"pass_rate":              r.PassRate,
		"median_tts_ms":          r.MedianTTSMs,
		"median_tokens_per_sec":  r.MedianTokensPerSec,
		"p90_cost_per_success":   r.P90CostPerSuccess,
		"speed_efficiency":       r.SpeedEfficiency,
		"dollars_per_pass":       r.DollarsPerPass,
		"cost_overhead_vs_best":  r.CostOverheadVsBest,
		"token_overhead_vs_best": r.TokenOverheadVsBest,
		"pareto_frontier":        r.ParetoFrontier,
		"buckets": map[string]int{
			"fast_pass":          r.Buckets.FastPass,
			"slow_pass":          r.Buckets.SlowPass,
			"budget_blocked":     r.Buckets.BudgetBlocked,
			"capability_blocked": r.Buckets.CapabilityBlocked,
			"refused":            r.Buckets.Refused,
			"provider_blocked":   r.Buckets.ProviderBlocked,
		},
		"error_categories": map[string]int{
			"cost_killed":     r.CostKilledCount,
			"step_exhausted":  r.StepExhaustedCount,
			"timeout":         r.TimeoutCount,
			"quota_exhausted": r.QuotaCount,
			"rate_limit":      r.RateLimitCount,
			"api_error":       r.APIErrorCount,
		},
		"finish_reasons": r.FinishReasons,
	}
}
