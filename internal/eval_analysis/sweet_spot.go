package eval_analysis

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
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

	// Failure-cause counts (from the typed ErrorCategory taxonomy).
	CostKilledCount    int `json:"cost_killed_count"`
	StepExhaustedCount int `json:"step_exhausted_count"`
	TimeoutCount       int `json:"timeout_count"`
	QuotaCount         int `json:"quota_count"`
	RateLimitCount     int `json:"rate_limit_count"`
	APIErrorCount      int `json:"api_error_count"`

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

	for _, r := range rs {
		if ShouldExcludeFromCapability(r.ErrorCategory) {
			excluded++
		}

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

	// Header
	sb.WriteString(fmt.Sprintf("%-38s %6s %8s %7s %10s %5s %5s %5s %5s\n",
		"Model", "Pass%", "MedTTS", "Tok/s", "p90$/win", "Fast", "Slow", "Bdgt", "Cap"))
	sb.WriteString(strings.Repeat("─", 95) + "\n")

	for _, row := range report.Rows {
		label := row.Model
		if row.Harness != "" && row.Harness != row.Model {
			label = row.Model + " · " + row.Harness
		}
		label = truncate(label, 38)
		sb.WriteString(fmt.Sprintf("%-38s %5.1f%% %7.1fs %7.0f %9.4f$ %5d %5d %5d %5d\n",
			label,
			row.PassRate*100,
			row.MedianTTSMs/1000,
			row.MedianTokensPerSec,
			row.P90CostPerSuccess,
			row.Buckets.FastPass,
			row.Buckets.SlowPass,
			row.Buckets.BudgetBlocked,
			row.Buckets.CapabilityBlocked,
		))
	}
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

// FormatSweetSpotJSON marshals the report to indented JSON.
func FormatSweetSpotJSON(report SweetSpotReport) (string, error) {
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func itoa(n int) string     { return fmt.Sprintf("%d", n) }
func ftoa(f float64) string { return fmt.Sprintf("%g", f) }
