package eval_analysis

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

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

func itoa(n int) string     { return fmt.Sprintf("%d", n) }
func ftoa(f float64) string { return fmt.Sprintf("%g", f) }

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
