// Package observatory provides outlier detection for span metrics within tasks.
package observatory

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// OutlierOptions configures outlier detection.
type OutlierOptions struct {
	Threshold float64 // Z-score threshold (default: 2.0)
	Metric    string  // Filter to specific metric: "cost", "duration", "tokens", or "" for all
	ShowRate  bool    // Include rate-of-change analysis
	Limit     int     // Max outliers to return (default: 10)
}

// DefaultOutlierOptions returns sensible defaults.
func DefaultOutlierOptions() OutlierOptions {
	return OutlierOptions{
		Threshold: 2.0,
		Metric:    "",
		ShowRate:  false,
		Limit:     10,
	}
}

// AnalyzeTaskOutliers performs statistical outlier detection on spans within a task.
func AnalyzeTaskOutliers(ctx context.Context, backend Backend, taskID string, opts OutlierOptions) (*OutlierAnalysis, error) {
	// Get all spans for the task, ordered by start_time
	spans, err := backend.ListSpans(ctx, SpanListOptions{
		TaskID: taskID,
		Limit:  10000, // High limit to get all spans
	})
	if err != nil {
		return nil, err
	}

	// If no spans found, return error
	if len(spans) == 0 {
		return nil, fmt.Errorf("no spans found for task %s", taskID)
	}

	// Try to get task info from tasks table, but don't fail if not found
	// (spans may have task_id that's actually a session_id, not in tasks table)
	var taskTitle string
	task, err := backend.GetTask(ctx, taskID)
	if err == nil && task != nil {
		taskTitle = task.Title
	} else {
		// Fall back to inferring title from spans
		taskTitle = inferTaskTitle(spans, taskID)
	}

	// Sort by start_time (ListSpans may not guarantee order)
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].StartTime.Before(spans[j].StartTime)
	})

	// Compute statistics for each metric
	stats := computeTaskStats(spans, opts.Metric)

	// Detect outliers
	outliers := detectOutliers(spans, stats, opts.Threshold, opts.Metric)

	// Sort outliers by |z-score| descending
	sort.Slice(outliers, func(i, j int) bool {
		return math.Abs(outliers[i].ZScore) > math.Abs(outliers[j].ZScore)
	})

	// Apply limit
	if opts.Limit > 0 && len(outliers) > opts.Limit {
		outliers = outliers[:opts.Limit]
	}

	analysis := &OutlierAnalysis{
		TaskID:     taskID,
		TaskTitle:  taskTitle,
		SpanCount:  len(spans),
		Stats:      stats,
		Outliers:   outliers,
		Threshold:  opts.Threshold,
		AnalyzedAt: time.Now(),
	}

	// Add rate-of-change analysis if requested
	if opts.ShowRate {
		analysis.RateOfChange = computeRateOfChange(spans)
	}

	return analysis, nil
}

// computeTaskStats calculates mean and standard deviation for each metric.
func computeTaskStats(spans []*Span, metricFilter string) []*TaskMetricStats {
	var stats []*TaskMetricStats

	// Collect values for each metric
	metrics := map[string][]float64{
		"cost_usd":    {},
		"duration_ms": {},
		"tokens":      {},
	}

	for _, span := range spans {
		metrics["cost_usd"] = append(metrics["cost_usd"], span.CostUSD)
		metrics["duration_ms"] = append(metrics["duration_ms"], float64(span.DurationMs))
		metrics["tokens"] = append(metrics["tokens"], float64(span.TokensIn+span.TokensOut))
	}

	// Calculate stats for each metric
	for metricName, values := range metrics {
		// Skip if filtering to a specific metric
		if metricFilter != "" && !matchesMetricFilter(metricName, metricFilter) {
			continue
		}

		stat := &TaskMetricStats{
			Metric: metricName,
		}

		// Count non-zero values
		var nonZeroValues []float64
		for _, v := range values {
			if v > 0 {
				nonZeroValues = append(nonZeroValues, v)
			}
		}

		stat.Count = len(nonZeroValues)
		if stat.Count == 0 {
			stats = append(stats, stat)
			continue
		}

		// Calculate sum, min, max
		stat.Sum = 0
		stat.Min = math.MaxFloat64
		stat.Max = -math.MaxFloat64
		for _, v := range nonZeroValues {
			stat.Sum += v
			if v < stat.Min {
				stat.Min = v
			}
			if v > stat.Max {
				stat.Max = v
			}
		}

		// Calculate mean
		stat.Mean = stat.Sum / float64(stat.Count)

		// Calculate standard deviation
		var sumSquaredDiff float64
		for _, v := range nonZeroValues {
			diff := v - stat.Mean
			sumSquaredDiff += diff * diff
		}
		if stat.Count > 1 {
			stat.StdDev = math.Sqrt(sumSquaredDiff / float64(stat.Count-1))
		}

		stats = append(stats, stat)
	}

	// Sort by metric name for consistent output
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Metric < stats[j].Metric
	})

	return stats
}

// matchesMetricFilter checks if a metric name matches the filter.
func matchesMetricFilter(metricName, filter string) bool {
	switch filter {
	case "cost":
		return metricName == "cost_usd"
	case "duration":
		return metricName == "duration_ms"
	case "tokens":
		return metricName == "tokens"
	default:
		return true
	}
}

// detectOutliers finds spans with z-scores exceeding the threshold.
func detectOutliers(spans []*Span, stats []*TaskMetricStats, threshold float64, metricFilter string) []*SpanOutlier {
	var outliers []*SpanOutlier

	// Build stats lookup
	statsMap := make(map[string]*TaskMetricStats)
	for _, s := range stats {
		statsMap[s.Metric] = s
	}

	for _, span := range spans {
		// Check each metric
		checkMetric := func(metricName string, value float64) {
			if metricFilter != "" && !matchesMetricFilter(metricName, metricFilter) {
				return
			}

			stat, ok := statsMap[metricName]
			if !ok || stat.StdDev == 0 || value == 0 {
				return
			}

			zScore := (value - stat.Mean) / stat.StdDev
			if math.Abs(zScore) >= threshold {
				outliers = append(outliers, &SpanOutlier{
					Span:           span,
					Metric:         metricName,
					Value:          value,
					Mean:           stat.Mean,
					StdDev:         stat.StdDev,
					ZScore:         zScore,
					PercentOfTotal: (value / stat.Sum) * 100,
				})
			}
		}

		checkMetric("cost_usd", span.CostUSD)
		checkMetric("duration_ms", float64(span.DurationMs))
		checkMetric("tokens", float64(span.TokensIn+span.TokensOut))
	}

	return outliers
}

// computeRateOfChange builds cumulative progression for each metric.
func computeRateOfChange(spans []*Span) *RateAnalysis {
	rate := &RateAnalysis{
		CumulativeCost:     make([]CumulativePoint, 0, len(spans)),
		CumulativeTokens:   make([]CumulativePoint, 0, len(spans)),
		CumulativeDuration: make([]CumulativePoint, 0, len(spans)),
	}

	// Calculate totals first for percent calculation
	var totalCost, totalTokens, totalDuration float64
	for _, span := range spans {
		totalCost += span.CostUSD
		totalTokens += float64(span.TokensIn + span.TokensOut)
		totalDuration += float64(span.DurationMs)
	}

	// Build cumulative points
	var cumCost, cumTokens, cumDuration float64
	for i, span := range spans {
		cost := span.CostUSD
		tokens := float64(span.TokensIn + span.TokensOut)
		duration := float64(span.DurationMs)

		cumCost += cost
		cumTokens += tokens
		cumDuration += duration

		// Only include points with non-zero values to reduce noise
		if cost > 0 {
			pct := 0.0
			if totalCost > 0 {
				pct = (cost / totalCost) * 100
			}
			rate.CumulativeCost = append(rate.CumulativeCost, CumulativePoint{
				SpanIndex:    i,
				SpanName:     span.Name,
				Timestamp:    span.StartTime,
				Value:        cost,
				Cumulative:   cumCost,
				DeltaPercent: pct,
			})
		}

		if tokens > 0 {
			pct := 0.0
			if totalTokens > 0 {
				pct = (tokens / totalTokens) * 100
			}
			rate.CumulativeTokens = append(rate.CumulativeTokens, CumulativePoint{
				SpanIndex:    i,
				SpanName:     span.Name,
				Timestamp:    span.StartTime,
				Value:        tokens,
				Cumulative:   cumTokens,
				DeltaPercent: pct,
			})
		}

		if duration > 0 {
			pct := 0.0
			if totalDuration > 0 {
				pct = (duration / totalDuration) * 100
			}
			rate.CumulativeDuration = append(rate.CumulativeDuration, CumulativePoint{
				SpanIndex:    i,
				SpanName:     span.Name,
				Timestamp:    span.StartTime,
				Value:        duration,
				Cumulative:   cumDuration,
				DeltaPercent: pct,
			})
		}
	}

	return rate
}

// inferTaskTitle attempts to derive a meaningful title from span data.
// Returns a descriptive title based on span attributes or a truncated task ID.
func inferTaskTitle(spans []*Span, taskID string) string {
	if len(spans) == 0 {
		return fmt.Sprintf("Session %s", truncateID(taskID))
	}

	// Try to find session info or model info from first span
	firstSpan := spans[0]
	if len(firstSpan.Attributes) > 0 {
		// Check for session title
		if title, ok := firstSpan.Attributes["session.title"].(string); ok && title != "" {
			return title
		}
		// Check for task title
		if title, ok := firstSpan.Attributes["task.title"].(string); ok && title != "" {
			return title
		}
	}

	// Calculate total cost for display
	var totalCost float64
	for _, span := range spans {
		totalCost += span.CostUSD
	}

	// Build descriptive title
	parts := []string{}
	providerStr := string(firstSpan.Provider)
	providerLower := strings.ToLower(providerStr)
	if providerLower == "claude" || strings.Contains(providerLower, "anthropic") {
		parts = append(parts, "Claude Code Session")
	} else if providerStr != "" {
		// Capitalize first letter of provider name
		capitalizedProvider := strings.ToUpper(providerStr[:1]) + providerStr[1:]
		parts = append(parts, fmt.Sprintf("%s Session", capitalizedProvider))
	} else {
		parts = append(parts, "Session")
	}

	parts = append(parts, fmt.Sprintf("(%d spans, $%.2f)", len(spans), totalCost))

	return strings.Join(parts, " ")
}

// truncateID returns a shortened version of a UUID for display.
func truncateID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
