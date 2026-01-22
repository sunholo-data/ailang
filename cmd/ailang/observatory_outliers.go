package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/observatory"
)

func observatoryOutliersCommand() {
	fs := flag.NewFlagSet("observatory outliers", flag.ExitOnError)
	taskID := fs.String("task", "", "Task ID to analyze (required)")
	metric := fs.String("metric", "", "Filter to specific metric: cost, duration, tokens (default: all)")
	threshold := fs.Float64("threshold", 2.0, "Z-score threshold for outlier detection")
	format := fs.String("format", "ascii", "Output format: json or ascii")
	showRate := fs.Bool("show-rate", false, "Include rate-of-change analysis")
	limit := fs.Int("limit", 10, "Maximum number of outliers to show")

	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *taskID == "" {
		fmt.Fprintf(os.Stderr, "Error: --task is required\n")
		fmt.Fprintf(os.Stderr, "Usage: ailang observatory outliers --task TASK_ID [options]\n")
		os.Exit(1)
	}

	ctx := context.Background()
	dbPath := observatory.DefaultDatabasePath()

	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	opts := observatory.OutlierOptions{
		Threshold: *threshold,
		Metric:    *metric,
		ShowRate:  *showRate,
		Limit:     *limit,
	}

	analysis, err := observatory.AnalyzeTaskOutliers(ctx, backend, *taskID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing task: %v\n", err)
		os.Exit(1)
	}

	// Build CLI command for display
	cliCmd := fmt.Sprintf("ailang observatory outliers --task %s --threshold %.1f", *taskID, *threshold)
	if *metric != "" {
		cliCmd += fmt.Sprintf(" --metric %s", *metric)
	}
	if *showRate {
		cliCmd += " --show-rate"
	}
	if *limit != 10 {
		cliCmd += fmt.Sprintf(" --limit %d", *limit)
	}

	if *format == "ascii" {
		printOutliersASCII(analysis, cliCmd)
	} else {
		output := map[string]interface{}{
			"analysis":    analysis,
			"cli_command": cliCmd + " --format json",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
	}
}

// printOutliersASCII formats outlier analysis for terminal display.
func printOutliersASCII(analysis *observatory.OutlierAnalysis, cliCmd string) {
	fmt.Println("=== Span Outlier Analysis ===")
	fmt.Println(strings.Repeat("═", 70))
	fmt.Printf("Task: %s (%s)\n", analysis.TaskID, analysis.TaskTitle)
	fmt.Printf("Spans: %d | Threshold: %.1fσ\n", analysis.SpanCount, analysis.Threshold)
	fmt.Println(strings.Repeat("─", 70))

	// Print metric statistics
	fmt.Println("\nMetric Statistics:")
	fmt.Printf("%-12s %8s %12s %12s %12s\n", "", "Count", "Sum", "Mean", "StdDev")
	for _, stat := range analysis.Stats {
		var sumStr, meanStr, stdStr string
		switch stat.Metric {
		case "cost_usd":
			sumStr = fmt.Sprintf("$%.4f", stat.Sum)
			meanStr = fmt.Sprintf("$%.4f", stat.Mean)
			stdStr = fmt.Sprintf("$%.4f", stat.StdDev)
		case "duration_ms":
			sumStr = formatDurationMs(stat.Sum)
			meanStr = formatDurationMs(stat.Mean)
			stdStr = formatDurationMs(stat.StdDev)
		case "tokens":
			sumStr = formatTokenCount(stat.Sum)
			meanStr = formatTokenCount(stat.Mean)
			stdStr = formatTokenCount(stat.StdDev)
		}
		fmt.Printf("  %-10s %8d %12s %12s %12s\n", stat.Metric, stat.Count, sumStr, meanStr, stdStr)
	}

	// Print outliers
	if len(analysis.Outliers) == 0 {
		fmt.Println("\nNo outliers detected (all spans within threshold).")
	} else {
		fmt.Printf("\nOutliers Detected (%d spans):\n", len(analysis.Outliers))
		fmt.Println(strings.Repeat("─", 70))
		for i, outlier := range analysis.Outliers {
			var valueStr string
			switch outlier.Metric {
			case "cost_usd":
				valueStr = fmt.Sprintf("$%.4f", outlier.Value)
			case "duration_ms":
				valueStr = formatDurationMs(outlier.Value)
			case "tokens":
				valueStr = formatTokenCount(outlier.Value)
			}
			fmt.Printf("  #%d %s\n", i+1, outlier.Span.Name)
			fmt.Printf("     %s: %s (z=%.2f, %.1f%% of total)\n",
				outlier.Metric, valueStr, outlier.ZScore, outlier.PercentOfTotal)
			if outlier.Span.Model != "" {
				fmt.Printf("     Model: %s | Provider: %s\n", outlier.Span.Model, outlier.Span.Provider)
			}
		}
	}

	// Print rate of change if available
	if analysis.RateOfChange != nil {
		fmt.Println("\nRate of Change (Top Contributors):")
		fmt.Println(strings.Repeat("─", 70))

		// Show top cost contributors
		if len(analysis.RateOfChange.CumulativeCost) > 0 {
			fmt.Println("  Cost:")
			printTopContributors(analysis.RateOfChange.CumulativeCost, "cost")
		}
		if len(analysis.RateOfChange.CumulativeTokens) > 0 {
			fmt.Println("  Tokens:")
			printTopContributors(analysis.RateOfChange.CumulativeTokens, "tokens")
		}
		if len(analysis.RateOfChange.CumulativeDuration) > 0 {
			fmt.Println("  Duration:")
			printTopContributors(analysis.RateOfChange.CumulativeDuration, "duration")
		}
	}

	fmt.Println()
	cliCmd += " --format ascii"
	fmt.Printf("CLI: %s\n", cliCmd)
}

// printTopContributors shows the top spans contributing to a metric.
func printTopContributors(points []observatory.CumulativePoint, metricType string) {
	// Sort by delta percent descending to find top contributors
	sorted := make([]observatory.CumulativePoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].DeltaPercent > sorted[j].DeltaPercent
	})

	// Show top 3
	limit := 3
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for i := 0; i < limit; i++ {
		p := sorted[i]
		var valueStr string
		switch metricType {
		case "cost":
			valueStr = fmt.Sprintf("$%.4f", p.Value)
		case "tokens":
			valueStr = formatTokenCount(p.Value)
		case "duration":
			valueStr = formatDurationMs(p.Value)
		}
		fmt.Printf("    • %s: %s (%.1f%%)\n", truncateSpanName(p.SpanName, 35), valueStr, p.DeltaPercent)
	}
}

// formatDurationMs formats milliseconds for display.
func formatDurationMs(ms float64) string {
	if ms >= 60000 {
		return fmt.Sprintf("%.1fm", ms/60000)
	} else if ms >= 1000 {
		return fmt.Sprintf("%.1fs", ms/1000)
	}
	return fmt.Sprintf("%.0fms", ms)
}

// formatTokenCount formats token counts for display.
func formatTokenCount(tokens float64) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", tokens/1000000)
	} else if tokens >= 1000 {
		return fmt.Sprintf("%.1fK", tokens/1000)
	}
	return fmt.Sprintf("%.0f", tokens)
}

// truncateSpanName truncates a string to max length with ellipsis.
func truncateSpanName(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// observatoryHierarchyCommand displays unified task/message/span hierarchy.
// Auto-detects ID type from the input argument.
