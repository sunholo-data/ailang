package observatory

import (
	"context"
	"database/sql"
	"fmt"
)

// ===== Filtered Queries =====

// GetFilteredHeatmapData returns daily activity data aggregated from spans
func (b *SQLiteBackend) GetFilteredHeatmapData(ctx context.Context, filter *ControlPlaneFilter, days int, wsConfig WorkspaceMapping) ([]HeatmapDataPoint, error) {
	conditions, args := buildFilterConditions(filter, wsConfig)

	// If no explicit date range in filter, use days parameter
	if filter == nil || (filter.StartDate == "" && filter.EndDate == "") {
		// Add days-based filter if not already set
		conditions = append(conditions, "date(start_time) >= date('now', ?)")
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			whereClause += " AND " + c
		}
	}

	query := fmt.Sprintf(`
		SELECT
			date(start_time) as date,
			COUNT(*) as span_count,
			COUNT(DISTINCT task_id) as task_count,
			COALESCE(SUM(cost_usd), 0) as cost,
			CAST(SUM(CASE WHEN status = 'OK' THEN 1 ELSE 0 END) AS REAL) /
				NULLIF(CAST(COUNT(*) AS REAL), 0) as success_rate,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out
		FROM spans
		%s
		GROUP BY date(start_time)
		ORDER BY date(start_time) ASC
	`, whereClause)

	rows, err := b.store.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []HeatmapDataPoint
	for rows.Next() {
		var point HeatmapDataPoint
		var successRate sql.NullFloat64
		if err := rows.Scan(&point.Date, &point.SpanCount, &point.TaskCount, &point.Cost,
			&successRate, &point.TokensIn, &point.TokensOut); err != nil {
			return nil, err
		}
		if successRate.Valid {
			point.SuccessRate = successRate.Float64
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

// GetFilteredMetricsSummary returns metrics filtered by Control Plane filter
func (b *SQLiteBackend) GetFilteredMetricsSummary(ctx context.Context, filter *ControlPlaneFilter, wsConfig WorkspaceMapping) (*MetricsSummary, error) {
	if filter == nil || filter.IsEmpty() {
		return b.store.GetMetricsSummary()
	}

	// Build WHERE clause using shared helper (includes time range filtering)
	conditions, args := buildFilterConditions(filter, wsConfig)

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			whereClause += " AND " + c
		}
	}

	// Build filtered query
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_spans,
			COUNT(DISTINCT task_id) as total_tasks,
			COUNT(DISTINCT provider) as total_agents,
			COALESCE(SUM(tokens_in), 0) as total_tokens_in,
			COALESCE(SUM(tokens_out), 0) as total_tokens_out,
			COALESCE(SUM(cost_usd), 0) as total_cost_usd,
			CAST(SUM(CASE WHEN status = 'OK' THEN 1 ELSE 0 END) AS REAL) /
				NULLIF(CAST(COUNT(*) AS REAL), 0) as success_rate,
			COALESCE(SUM(cache_read_tokens), 0) as total_cache_read_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as total_cache_creation_tokens,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as error_count
		FROM spans
		%s
	`, whereClause)

	row := b.store.DB().QueryRowContext(ctx, query, args...)
	var summary MetricsSummary
	var successRate sql.NullFloat64
	err := row.Scan(&summary.TotalSpans, &summary.TotalTasks, &summary.TotalAgents,
		&summary.TotalTokensIn, &summary.TotalTokensOut, &summary.TotalCostUSD, &successRate,
		&summary.TotalCacheReadTokens, &summary.TotalCacheCreationTokens, &summary.ErrorCount)
	if err != nil {
		return nil, err
	}
	if successRate.Valid {
		summary.SuccessRate = successRate.Float64
	}

	// Calculate cache savings
	if summary.TotalCacheReadTokens > 0 {
		summary.CacheSavingsUSD = CalculateCacheSavings("", summary.TotalCacheReadTokens)
	}

	// Workspace count filtered separately if workspace filter is set
	if filter.Workspace != "" {
		summary.TotalWorkspaces = 1
	} else {
		// Count distinct workspaces from filtered spans
		countQuery := fmt.Sprintf(`
			SELECT COUNT(DISTINCT t.workspace_id)
			FROM spans s
			LEFT JOIN tasks t ON s.task_id = t.id
			%s
		`, whereClause)
		var wsCount int
		if err := b.store.DB().QueryRowContext(ctx, countQuery, args...).Scan(&wsCount); err == nil {
			summary.TotalWorkspaces = wsCount
		}
	}

	return &summary, nil
}
