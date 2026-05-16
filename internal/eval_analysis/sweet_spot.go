package eval_analysis

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
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

// formatOverhead renders a CostOverheadVsBest / TokenOverheadVsBest ratio
// as a compact human-readable string. "—" for zero (no qualifying passes),
// "1.0×" for matched-the-cheapest, "X.X×" for typical, "XXX×" with no
// decimal when ratio ≥ 100 to keep column width sane.
func formatOverhead(ratio float64) string {
	if ratio <= 0 {
		return "—"
	}
	if ratio >= 100 {
		return fmt.Sprintf("%.0f×", ratio)
	}
	return fmt.Sprintf("%.1f×", ratio)
}

// FormatSweetSpotText renders the report as a human-readable ANSI table.
func FormatSweetSpotText(report SweetSpotReport, useColor bool) string {
	var sb strings.Builder
	sb.WriteString(colorize("═══════════════════════════════════════════════════════════════════════════════════════════════\n", colorCyan, useColor))
	sb.WriteString(colorize(fmt.Sprintf("  Sweet-Spot Report (slow threshold = %.1fs, %d total runs)\n",
		float64(report.SlowMs)/1000, report.TotalRuns), colorBold, useColor))
	sb.WriteString(colorize("═══════════════════════════════════════════════════════════════════════════════════════════════\n", colorCyan, useColor))
	sb.WriteString("\n")

	if len(report.Rows) == 0 {
		sb.WriteString("(no results)\n")
		return sb.String()
	}

	// Header — $-Ovhd and Tok-Ovhd added per the "actual value found" metric:
	// median ratio of this-model-cost / cheapest-passer-cost per benchmark.
	// 1.0× = matched the cheapest passer on every benchmark this model passed.
	sb.WriteString(fmt.Sprintf("%-38s %6s %8s %7s %10s %8s %9s %5s %5s %5s %5s\n",
		"Model", "Pass%", "MedTTS", "Tok/s", "p90$/win", "$-Ovhd", "Tok-Ovhd", "Fast", "Slow", "Bdgt", "Cap"))
	sb.WriteString(strings.Repeat("─", 115) + "\n")

	for _, row := range report.Rows {
		label := row.Model
		if row.Harness != "" && row.Harness != row.Model {
			label = row.Model + " · " + row.Harness
		}
		label = truncate(label, 38)
		sb.WriteString(fmt.Sprintf("%-38s %5.1f%% %7.1fs %7.0f %9.4f$ %7s %8s %5d %5d %5d %5d\n",
			label,
			row.PassRate*100,
			row.MedianTTSMs/1000,
			row.MedianTokensPerSec,
			row.P90CostPerSuccess,
			formatOverhead(row.CostOverheadVsBest),
			formatOverhead(row.TokenOverheadVsBest),
			row.Buckets.FastPass,
			row.Buckets.SlowPass,
			row.Buckets.BudgetBlocked,
			row.Buckets.CapabilityBlocked,
		))
	}
	sb.WriteString("\n")
	sb.WriteString(colorize("  $-Ovhd / Tok-Ovhd: median ratio of this-model-cost (or tokens) vs cheapest passer per benchmark.\n", colorYellow, useColor))
	sb.WriteString(colorize("  1.0× = matched the cheapest passer on every benchmark; — = no qualifying passes.\n", colorYellow, useColor))
	sb.WriteString("\n")

	// Failure-cause breakdown (only when at least one model has typed
	// failures — keeps the report concise for clean datasets).
	anyTyped := false
	for _, r := range report.Rows {
		if r.CostKilledCount+r.StepExhaustedCount+r.TimeoutCount+r.QuotaCount+r.RateLimitCount > 0 {
			anyTyped = true
			break
		}
	}
	if anyTyped {
		sb.WriteString(colorize("Failure Causes (typed categories)\n", colorBold, useColor))
		sb.WriteString(strings.Repeat("─", 95) + "\n")
		sb.WriteString(fmt.Sprintf("%-38s %6s %6s %6s %6s %6s %6s\n",
			"Model", "cost", "step", "timeout", "quota", "ratelim", "api"))
		for _, row := range report.Rows {
			total := row.CostKilledCount + row.StepExhaustedCount + row.TimeoutCount +
				row.QuotaCount + row.RateLimitCount + row.APIErrorCount
			if total == 0 {
				continue
			}
			sb.WriteString(fmt.Sprintf("%-38s %6d %6d %6d %6d %6d %6d\n",
				truncate(row.Model, 38),
				row.CostKilledCount, row.StepExhaustedCount, row.TimeoutCount,
				row.QuotaCount, row.RateLimitCount, row.APIErrorCount))
		}
		sb.WriteString("\n")
	}

	// Cost-vs-Time frontier (NEW v0.19.0)
	if frontier := FormatCostSpeedFrontier(report, useColor); frontier != "" {
		sb.WriteString(frontier)
		sb.WriteString("\n")
	}

	// Champions
	if len(report.Champions) > 0 {
		sb.WriteString(colorize("Cheapest / Fastest Pass per Benchmark\n", colorBold, useColor))
		sb.WriteString(strings.Repeat("─", 95) + "\n")
		sb.WriteString(fmt.Sprintf("%-28s %-30s %9s  %-30s %7s\n",
			"Benchmark", "Cheapest", "$/win", "Fastest", "TTS"))
		for _, c := range report.Champions {
			sb.WriteString(fmt.Sprintf("%-28s %-30s %8.4f$ %-30s %6.1fs\n",
				truncate(c.BenchmarkID, 28),
				truncate(c.CheapestModel, 30),
				c.CheapestCost,
				truncate(c.FastestModel, 30),
				c.FastestTTSMs/1000))
		}
	}

	return sb.String()
}

// FormatSweetSpotCSV writes the per-model rows as CSV. Champions are emitted
// in a separate "## Champions" section below the rows.
func FormatSweetSpotCSV(report SweetSpotReport) (string, error) {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	header := []string{
		"model", "harness", "total_runs", "pass_rate",
		"median_tts_ms", "median_tokens_per_sec", "p90_cost_per_success", "speed_efficiency",
		"fast_pass", "slow_pass", "budget_blocked", "capability_blocked", "provider_blocked",
		"cost_killed", "step_exhausted", "timeout", "quota_exhausted", "rate_limit", "api_error",
	}
	if err := w.Write(header); err != nil {
		return "", err
	}
	for _, r := range report.Rows {
		if err := w.Write([]string{
			r.Model, r.Harness, itoa(r.TotalRuns), ftoa(r.PassRate),
			ftoa(r.MedianTTSMs), ftoa(r.MedianTokensPerSec), ftoa(r.P90CostPerSuccess), ftoa(r.SpeedEfficiency),
			itoa(r.Buckets.FastPass), itoa(r.Buckets.SlowPass), itoa(r.Buckets.BudgetBlocked),
			itoa(r.Buckets.CapabilityBlocked), itoa(r.Buckets.ProviderBlocked),
			itoa(r.CostKilledCount), itoa(r.StepExhaustedCount), itoa(r.TimeoutCount),
			itoa(r.QuotaCount), itoa(r.RateLimitCount), itoa(r.APIErrorCount),
		}); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}

	if len(report.Champions) > 0 {
		sb.WriteString("\n## Champions (cheapest / fastest pass per benchmark)\n")
		w2 := csv.NewWriter(&sb)
		_ = w2.Write([]string{"benchmark_id", "cheapest_model", "cheapest_cost_usd", "cheapest_tts_ms",
			"fastest_model", "fastest_tts_ms", "fastest_cost_usd"})
		for _, c := range report.Champions {
			_ = w2.Write([]string{
				c.BenchmarkID, c.CheapestModel, ftoa(c.CheapestCost), ftoa(c.CheapestTTSMs),
				c.FastestModel, ftoa(c.FastestTTSMs), ftoa(c.FastestCost),
			})
		}
		w2.Flush()
	}

	return sb.String(), nil
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
		"model":                 r.Model,
		"harness":               r.Harness,
		"total_runs":            r.TotalRuns,
		"pass_rate":             r.PassRate,
		"median_tts_ms":         r.MedianTTSMs,
		"median_tokens_per_sec": r.MedianTokensPerSec,
		"p90_cost_per_success":  r.P90CostPerSuccess,
		"speed_efficiency":      r.SpeedEfficiency,
		"dollars_per_pass":      r.DollarsPerPass,
		"pareto_frontier":       r.ParetoFrontier,
		"buckets": map[string]int{
			"fast_pass":          r.Buckets.FastPass,
			"slow_pass":          r.Buckets.SlowPass,
			"budget_blocked":     r.Buckets.BudgetBlocked,
			"capability_blocked": r.Buckets.CapabilityBlocked,
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

// FormatSweetSpotJSON marshals the report to indented JSON.
func FormatSweetSpotJSON(report SweetSpotReport) (string, error) {
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FormatCostSpeedFrontier produces an ASCII cost-vs-time scatter on log-log
// axes and a Pareto-frontier table. Each model with at least one successful
// run gets a point at (median p90 cost-per-success, median time-to-success).
//
// Pareto-optimal models — those for which no other model has BOTH lower cost
// AND lower time — are flagged with `*` in the scatter and listed first in
// the frontier table.
//
// Returns "" if no model has both metrics populated.
func FormatCostSpeedFrontier(report SweetSpotReport, useColor bool) string {
	var pts []frontierPoint
	for _, r := range report.Rows {
		if r.P90CostPerSuccess <= 0 || r.MedianTTSMs <= 0 {
			continue
		}
		label := r.Model
		if r.Harness != "" && r.Harness != r.Model {
			label = r.Model + "·" + r.Harness
		}
		pts = append(pts, frontierPoint{
			label:   label,
			costUSD: r.P90CostPerSuccess,
			ttsSec:  r.MedianTTSMs / 1000.0,
		})
	}
	if len(pts) < 2 {
		return ""
	}

	// Pareto frontier: point P is dominated if ∃ Q with Q.cost ≤ P.cost
	// AND Q.tts ≤ P.tts AND (Q.cost < P.cost OR Q.tts < P.tts).
	for i := range pts {
		dominated := false
		for j := range pts {
			if i == j {
				continue
			}
			if pts[j].costUSD <= pts[i].costUSD && pts[j].ttsSec <= pts[i].ttsSec &&
				(pts[j].costUSD < pts[i].costUSD || pts[j].ttsSec < pts[i].ttsSec) {
				dominated = true
				break
			}
		}
		pts[i].pareto = !dominated
	}

	// Compute log-axis bounds.
	minLogCost, maxLogCost := pts[0].logCost(), pts[0].logCost()
	minLogTime, maxLogTime := pts[0].logTime(), pts[0].logTime()
	for _, p := range pts[1:] {
		if v := p.logCost(); v < minLogCost {
			minLogCost = v
		} else if v > maxLogCost {
			maxLogCost = v
		}
		if v := p.logTime(); v < minLogTime {
			minLogTime = v
		} else if v > maxLogTime {
			maxLogTime = v
		}
	}
	// Pad bounds slightly.
	const pad = 0.1
	minLogCost -= pad
	maxLogCost += pad
	minLogTime -= pad
	maxLogTime += pad

	// Grid dimensions
	const W, H = 50, 12
	grid := make([][]byte, H)
	for i := range grid {
		grid[i] = []byte(strings.Repeat(" ", W))
	}

	// Plot points (top-left = high cost, fast; bottom-right = low cost, slow — we flip Y so high-time is at the bottom).
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if len(pts) > len(letters) {
		pts = pts[:len(letters)] // safety
	}
	for i, p := range pts {
		x := int(((p.logCost() - minLogCost) / (maxLogCost - minLogCost)) * float64(W-1))
		// Y axis: small logTime (fast) → small y (top); large logTime (slow) → large y (bottom).
		// Matches the "fast … slow" labels rendered below.
		y := int(((p.logTime() - minLogTime) / (maxLogTime - minLogTime)) * float64(H-1))
		if x < 0 {
			x = 0
		} else if x >= W {
			x = W - 1
		}
		if y < 0 {
			y = 0
		} else if y >= H {
			y = H - 1
		}
		ch := letters[i]
		if p.pareto {
			ch = letters[i] - 'A' + 'a' // lowercase = on frontier
		}
		grid[y][x] = ch
	}

	var sb strings.Builder
	sb.WriteString(colorize("Cost vs Time Frontier (log-log; lowercase = Pareto-optimal)\n", colorBold, useColor))
	sb.WriteString(strings.Repeat("─", 60) + "\n")
	sb.WriteString(fmt.Sprintf("  fast %6.1fs ┤", pts[0].invLog(minLogTime)))
	sb.WriteString(string(grid[0]))
	sb.WriteString("\n")
	for i := 1; i < H-1; i++ {
		sb.WriteString("              │")
		sb.WriteString(string(grid[i]))
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("  slow %6.1fs ┤", pts[0].invLog(maxLogTime)))
	sb.WriteString(string(grid[H-1]))
	sb.WriteString("\n")
	sb.WriteString("              └" + strings.Repeat("─", W) + "\n")
	sb.WriteString(fmt.Sprintf("              %8.4f$%s%8.4f$ cost / success\n",
		expLog(minLogCost),
		strings.Repeat(" ", W-18),
		expLog(maxLogCost)))
	sb.WriteString("\n")

	// Legend with Pareto frontier first
	sb.WriteString(fmt.Sprintf("%-3s %-38s %10s %10s %s\n",
		"Sym", "Model", "$/win", "Med TTS", "Pareto?"))
	for i, p := range pts {
		ch := letters[i]
		paretoFlag := ""
		if p.pareto {
			ch = letters[i] - 'A' + 'a'
			paretoFlag = colorize("✓ frontier", colorGreen, useColor)
		} else {
			paretoFlag = "dominated"
		}
		sb.WriteString(fmt.Sprintf(" %c  %-38s %9.4f$ %8.1fs  %s\n",
			ch, truncate(p.label, 38), p.costUSD, p.ttsSec, paretoFlag))
	}
	return sb.String()
}

// frontierPoint is used by FormatCostSpeedFrontier to position models on
// the cost-vs-time scatter and to classify Pareto-frontier membership.
type frontierPoint struct {
	label   string
	costUSD float64
	ttsSec  float64
	pareto  bool
}

// logCost returns log10(cost). Cost is guaranteed > 0 by caller filter.
func (p frontierPoint) logCost() float64         { return math.Log10(p.costUSD) }
func (p frontierPoint) logTime() float64         { return math.Log10(p.ttsSec) }
func (p frontierPoint) invLog(v float64) float64 { return math.Pow(10, v) }
func expLog(v float64) float64                   { return math.Pow(10, v) }

// FormatSweetSpotMDX renders the report as a Docusaurus-ready markdown
// section (plain tables, no custom components). Suitable for inlining into
// the auto-generated dashboard MDX.
func FormatSweetSpotMDX(report SweetSpotReport) string {
	var sb strings.Builder
	sb.WriteString("## Sweet Spot\n\n")
	sb.WriteString(fmt.Sprintf(
		"Per-model cost-vs-time-vs-success ranking (slow threshold = %.0fs, %d total runs).\n\n",
		float64(report.SlowMs)/1000, report.TotalRuns,
	))
	sb.WriteString("Buckets per (model × benchmark):\n\n")
	sb.WriteString("- **fast_pass** — model passes within the slow threshold\n")
	sb.WriteString("- **slow_pass** — passes but takes longer than the threshold\n")
	sb.WriteString("- **budget_blocked** — failed due to `cost_killed` or `step_exhausted` (more budget could help)\n")
	sb.WriteString("- **capability_blocked** — `compile_error` / `runtime_error` / `logic_error` / `timeout`\n")
	sb.WriteString("- **provider_blocked** — `quota_exhausted` / `rate_limit` / `api_error` (excluded from pass rate)\n\n")

	if len(report.Rows) == 0 {
		sb.WriteString("_No data._\n")
		return sb.String()
	}

	sb.WriteString("| Model | Pass% | Median TTS | Tokens/s | p90 $/win | Fast | Slow | Budget | Capability |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range report.Rows {
		label := r.Model
		if r.Harness != "" && r.Harness != r.Model {
			label = r.Model + " · " + r.Harness
		}
		sb.WriteString(fmt.Sprintf("| %s | %.1f%% | %.1fs | %.0f | $%.4f | %d | %d | %d | %d |\n",
			label,
			r.PassRate*100,
			r.MedianTTSMs/1000,
			r.MedianTokensPerSec,
			r.P90CostPerSuccess,
			r.Buckets.FastPass,
			r.Buckets.SlowPass,
			r.Buckets.BudgetBlocked,
			r.Buckets.CapabilityBlocked,
		))
	}

	if len(report.Champions) > 0 {
		sb.WriteString("\n### Cheapest / Fastest Pass per Benchmark\n\n")
		sb.WriteString("| Benchmark | Cheapest model | $/win | Fastest model | TTS |\n")
		sb.WriteString("|---|---|---:|---|---:|\n")
		for _, c := range report.Champions {
			sb.WriteString(fmt.Sprintf("| %s | %s | $%.4f | %s | %.1fs |\n",
				c.BenchmarkID, c.CheapestModel, c.CheapestCost,
				c.FastestModel, c.FastestTTSMs/1000))
		}
	}

	return sb.String()
}

func itoa(n int) string     { return fmt.Sprintf("%d", n) }
func ftoa(f float64) string { return fmt.Sprintf("%g", f) }
