package eval_analysis

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorCyan   = "\033[0;36m"
	colorBold   = "\033[1m"
)

// FormatComparison produces a human-readable comparison report
func FormatComparison(report *ComparisonReport, useColor bool) string {
	var sb strings.Builder

	// Header
	sb.WriteString(colorize("═══════════════════════════════════════════════\n", colorCyan, useColor))
	sb.WriteString(colorize(fmt.Sprintf("  Eval Diff: %s → %s\n", report.BaselineLabel, report.NewLabel), colorCyan, useColor))
	sb.WriteString(colorize("═══════════════════════════════════════════════\n", colorCyan, useColor))
	sb.WriteString("\n")

	// Summary
	sb.WriteString(colorize("Summary\n", colorBold, useColor))
	sb.WriteString("═══════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("%-30s %10s %10s\n", "", report.BaselineLabel, report.NewLabel))
	sb.WriteString(fmt.Sprintf("%-30s %10d %10d\n", "Total benchmarks", report.TotalBaselineBench, report.TotalNewBench))
	sb.WriteString("\n")

	// Fixed benchmarks
	if len(report.Fixed) > 0 {
		sb.WriteString(colorize(fmt.Sprintf("✓ Fixed (%d):\n", len(report.Fixed)), colorGreen, useColor))
		for _, change := range report.Fixed {
			sb.WriteString(fmt.Sprintf("  • %s (%s, %s)\n", change.ID, change.Lang, change.Model))
		}
		sb.WriteString("\n")
	}

	// Broken benchmarks
	if len(report.Broken) > 0 {
		sb.WriteString(colorize(fmt.Sprintf("✗ Broken (%d):\n", len(report.Broken)), colorRed, useColor))
		for _, change := range report.Broken {
			sb.WriteString(fmt.Sprintf("  • %s (%s, %s): %s\n",
				change.ID, change.Lang, change.Model, change.NewError))
		}
		sb.WriteString("\n")
	}

	// Still passing
	if len(report.StillPassing) > 0 {
		sb.WriteString(colorize(fmt.Sprintf("→ Still passing (%d)\n", len(report.StillPassing)), colorCyan, useColor))
		if len(report.StillPassing) <= 10 {
			for _, r := range report.StillPassing {
				sb.WriteString(fmt.Sprintf("  • %s\n", r.ID))
			}
		} else {
			sb.WriteString(fmt.Sprintf("  (%d benchmarks - too many to list)\n", len(report.StillPassing)))
		}
		sb.WriteString("\n")
	}

	// Still failing
	if len(report.StillFailing) > 0 {
		sb.WriteString(colorize(fmt.Sprintf("⚠ Still failing (%d)\n", len(report.StillFailing)), colorYellow, useColor))
		if len(report.StillFailing) <= 10 {
			for _, r := range report.StillFailing {
				sb.WriteString(fmt.Sprintf("  • %s: %s\n", r.ID, r.ErrorCategory))
			}
		} else {
			sb.WriteString(fmt.Sprintf("  (%d benchmarks - too many to list)\n", len(report.StillFailing)))
		}
		sb.WriteString("\n")
	}

	// New benchmarks
	if len(report.NewBenchmarks) > 0 {
		sb.WriteString(colorize(fmt.Sprintf("+ New benchmarks (%d):\n", len(report.NewBenchmarks)), colorCyan, useColor))
		for _, r := range report.NewBenchmarks {
			status := "passing"
			statusColor := colorGreen
			if !r.StdoutOk {
				status = "failing"
				statusColor = colorRed
			}
			sb.WriteString(fmt.Sprintf("  • %s %s\n", r.ID,
				colorize(fmt.Sprintf("(%s)", status), statusColor, useColor)))
		}
		sb.WriteString("\n")
	}

	// Removed benchmarks
	if len(report.Removed) > 0 {
		sb.WriteString(colorize(fmt.Sprintf("- Removed benchmarks (%d):\n", len(report.Removed)), colorYellow, useColor))
		for _, r := range report.Removed {
			sb.WriteString(fmt.Sprintf("  • %s\n", r.ID))
		}
		sb.WriteString("\n")
	}

	// Success rates
	sb.WriteString(colorize("Success Rates\n", colorBold, useColor))
	sb.WriteString("═══════════════════════════════════════════════\n")

	baselineSuccessCount := int(report.BaselineSuccessRate * float64(report.TotalBaselineBench))
	newSuccessCount := int(report.NewSuccessRate * float64(report.TotalNewBench))

	sb.WriteString(fmt.Sprintf("%-30s %10s %10s\n",
		report.BaselineLabel,
		fmt.Sprintf("%d/%d", baselineSuccessCount, report.TotalBaselineBench),
		fmt.Sprintf("(%.1f%%)", report.BaselineSuccessRate*100)))

	sb.WriteString(fmt.Sprintf("%-30s %10s %10s\n",
		report.NewLabel,
		fmt.Sprintf("%d/%d", newSuccessCount, report.TotalNewBench),
		fmt.Sprintf("(%.1f%%)", report.NewSuccessRate*100)))

	sb.WriteString("\n")

	// Delta
	delta := report.ImprovementPercent()
	if delta > 0 {
		sb.WriteString(colorize(fmt.Sprintf("Change: +%.1f%% improvement\n", delta), colorGreen, useColor))
	} else if delta < 0 {
		sb.WriteString(colorize(fmt.Sprintf("Change: %.1f%% regression\n", delta), colorRed, useColor))
	} else {
		sb.WriteString("Change: No change in success rate\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// FormatMatrix produces a human-readable matrix summary
func FormatMatrix(matrix *PerformanceMatrix, useColor bool) string {
	var sb strings.Builder

	sb.WriteString(colorize("═══════════════════════════════════════════════\n", colorCyan, useColor))
	sb.WriteString(colorize(fmt.Sprintf("  Performance Matrix: %s\n", matrix.Version), colorBold, useColor))
	sb.WriteString(colorize("═══════════════════════════════════════════════\n", colorCyan, useColor))
	sb.WriteString("\n")

	// Overall stats
	sb.WriteString(colorize("Overall Statistics\n", colorBold, useColor))
	sb.WriteString("───────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("Total runs:        %d\n", matrix.TotalRuns))
	sb.WriteString(fmt.Sprintf("0-shot success:    %.1f%%\n", matrix.Aggregates.ZeroShotSuccess*100))
	sb.WriteString(fmt.Sprintf("Final success:     %.1f%%\n", matrix.Aggregates.FinalSuccess*100))
	sb.WriteString(fmt.Sprintf("Repairs used:      %d\n", matrix.Aggregates.RepairUsed))
	sb.WriteString(fmt.Sprintf("Repair success:    %.1f%%\n", matrix.Aggregates.RepairSuccessRate*100))
	sb.WriteString(fmt.Sprintf("Total tokens:      %d\n", matrix.Aggregates.TotalTokens))
	sb.WriteString(fmt.Sprintf("Total cost:        $%.4f\n", matrix.Aggregates.TotalCostUSD))
	sb.WriteString(fmt.Sprintf("Avg duration:      %.0fms\n", matrix.Aggregates.AvgDurationMs))
	sb.WriteString("\n")

	// Model comparison
	if len(matrix.Models) > 0 {
		sb.WriteString(colorize("By Model\n", colorBold, useColor))
		sb.WriteString("───────────────────────────────────────────────\n")
		sb.WriteString(fmt.Sprintf("%-25s %10s %10s %8s\n", "Model", "0-shot", "Final", "Tokens"))
		for model, stats := range matrix.Models {
			sb.WriteString(fmt.Sprintf("%-25s %9.1f%% %9.1f%% %8d\n",
				truncate(model, 25),
				stats.Aggregates.ZeroShotSuccess*100,
				stats.Aggregates.FinalSuccess*100,
				stats.Aggregates.TotalTokens))
		}
		sb.WriteString("\n")
	}

	// Error codes
	if len(matrix.ErrorCodes) > 0 {
		sb.WriteString(colorize("Top Error Codes\n", colorBold, useColor))
		sb.WriteString("───────────────────────────────────────────────\n")
		sb.WriteString(fmt.Sprintf("%-15s %8s %12s\n", "Code", "Count", "Repair %"))
		for _, ec := range matrix.ErrorCodes {
			if ec.Count == 0 {
				continue
			}
			sb.WriteString(fmt.Sprintf("%-15s %8d %11.1f%%\n",
				ec.Code, ec.Count, ec.RepairSuccess*100))
		}
		sb.WriteString("\n")
	}

	// Language comparison
	if len(matrix.Languages) > 1 {
		sb.WriteString(colorize("By Language\n", colorBold, useColor))
		sb.WriteString("───────────────────────────────────────────────\n")
		for lang, stats := range matrix.Languages {
			sb.WriteString(fmt.Sprintf("%-15s: %.1f%% success, %.0f avg tokens\n",
				lang, stats.SuccessRate*100, stats.AvgTokens))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatJSONL converts results to JSONL format (one JSON object per line)
func FormatJSONL(results []*BenchmarkResult) (string, error) {
	var sb strings.Builder

	for _, r := range results {
		entry := r.ToSummaryEntry()
		data, err := json.Marshal(entry)
		if err != nil {
			return "", fmt.Errorf("failed to marshal result %s: %w", r.ID, err)
		}
		sb.Write(data)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// FormatJSON converts a matrix to pretty-printed JSON
func FormatJSON(matrix *PerformanceMatrix) (string, error) {
	data, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal matrix: %w", err)
	}
	return string(data), nil
}

// Helper functions

func colorize(text, color string, enabled bool) string {
	if !enabled {
		return text
	}
	return color + text + colorReset
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
