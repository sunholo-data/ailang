// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"database/sql"
	"fmt"
)

// ===== Control Plane Types =====

// ControlPlaneFilter defines filter parameters for Control Plane queries
type ControlPlaneFilter struct {
	SourceType string // eval, coordinator, direct_api, local, other
	Provider   string // claude, gemini, openai, etc.
	Model      string // claude-sonnet-4-5, gemini-2-5-pro, etc.
	Workspace  string // workspace ID
	StartDate  string // YYYY-MM-DD format for time range filter (inclusive)
	EndDate    string // YYYY-MM-DD format for time range filter (inclusive)
}

// IsEmpty returns true if no filters are set
func (f *ControlPlaneFilter) IsEmpty() bool {
	return f.SourceType == "" && f.Provider == "" && f.Model == "" && f.Workspace == "" && f.StartDate == "" && f.EndDate == ""
}

// HasTimeRange returns true if time range filter is set
func (f *ControlPlaneFilter) HasTimeRange() bool {
	return f.StartDate != "" || f.EndDate != ""
}

// BreakdownItem represents a single item in a breakdown aggregation
type BreakdownItem struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	SpanCount int     `json:"span_count"`
	TaskCount int     `json:"task_count,omitempty"`
	TokensIn  int64   `json:"tokens_in"`
	TokensOut int64   `json:"tokens_out"`
	CostUSD   float64 `json:"cost_usd"`
}

// HeatmapDataPoint represents activity data for a single day
type HeatmapDataPoint struct {
	Date        string  `json:"date"`         // YYYY-MM-DD
	SpanCount   int     `json:"span_count"`   // Number of spans
	TaskCount   int     `json:"task_count"`   // Number of distinct tasks
	Cost        float64 `json:"cost"`         // Total cost USD
	SuccessRate float64 `json:"success_rate"` // 0.0 to 1.0
	TokensIn    int64   `json:"tokens_in"`
	TokensOut   int64   `json:"tokens_out"`
}

// ===== Helper Functions =====

// buildSourceTypeCondition returns SQL condition for source type filter
func buildSourceTypeCondition(sourceType string) string {
	switch sourceType {
	case "eval":
		return "(name LIKE 'api_request%' OR name LIKE 'eval.%')"
	case "coordinator":
		return "(name LIKE 'coordinator.%' OR name LIKE 'claude.execute%')"
	case "direct_api":
		return "(name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%')"
	case "local":
		return "(name LIKE 'ailang %')"
	case "other":
		return "(name NOT LIKE 'api_request%' AND name NOT LIKE 'eval.%' AND name NOT LIKE 'coordinator.%' AND name NOT LIKE 'claude.execute%' AND name NOT LIKE 'anthropic.%' AND name NOT LIKE 'gemini.%' AND name NOT LIKE 'openai.%' AND name NOT LIKE 'ailang %')"
	default:
		return ""
	}
}

// buildFilterConditions builds WHERE clause conditions from a ControlPlaneFilter
// Returns conditions slice, args slice, ready for building WHERE clause
func buildFilterConditions(filter *ControlPlaneFilter) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter == nil {
		return conditions, args
	}

	if filter.SourceType != "" {
		if cond := buildSourceTypeCondition(filter.SourceType); cond != "" {
			conditions = append(conditions, cond)
		}
	}
	if filter.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, filter.Provider)
	}
	if filter.Model != "" {
		conditions = append(conditions, "model = ?")
		args = append(args, filter.Model)
	}
	if filter.StartDate != "" {
		// start_time is datetime, compare with date string (SQLite handles this)
		conditions = append(conditions, "date(start_time) >= ?")
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		conditions = append(conditions, "date(start_time) <= ?")
		args = append(args, filter.EndDate)
	}

	return conditions, args
}

// ===== Breakdown Queries =====

// GetBreakdownByProvider returns cost/token breakdown by provider
func (b *SQLiteBackend) GetBreakdownByProvider(ctx context.Context) ([]BreakdownItem, error) {
	rows, err := b.store.DB().QueryContext(ctx, `
		SELECT
			COALESCE(provider, 'unknown') as id,
			COALESCE(provider, 'Unknown') as label,
			COUNT(*) as span_count,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out,
			COALESCE(SUM(cost_usd), 0) as cost_usd
		FROM spans
		WHERE provider IS NOT NULL AND provider != ''
		GROUP BY provider
		ORDER BY cost_usd DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetBreakdownBySourceType returns cost/token breakdown by source type (inferred from span names)
func (b *SQLiteBackend) GetBreakdownBySourceType(ctx context.Context) ([]BreakdownItem, error) {
	// Categorize spans by their name pattern
	// GROUP BY 1, 2 uses column position since SQLite doesn't support alias in GROUP BY
	rows, err := b.store.DB().QueryContext(ctx, `
		SELECT
			CASE
				WHEN name LIKE 'api_request%' THEN 'eval'
				WHEN name LIKE 'eval.%' THEN 'eval'
				WHEN name LIKE 'coordinator.%' OR name LIKE 'claude.execute%' THEN 'coordinator'
				WHEN name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%' THEN 'direct_api'
				WHEN name LIKE 'ailang.exec%' THEN 'exec'
				WHEN name LIKE 'ailang.%' OR name LIKE 'ailang %' THEN 'local'
				ELSE 'other'
			END as id,
			CASE
				WHEN name LIKE 'api_request%' THEN 'Eval Benchmarks'
				WHEN name LIKE 'eval.%' THEN 'Eval Benchmarks'
				WHEN name LIKE 'coordinator.%' OR name LIKE 'claude.execute%' THEN 'Coordinator Tasks'
				WHEN name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%' THEN 'Direct API Calls'
				WHEN name LIKE 'ailang.exec%' THEN 'Exec Tasks'
				WHEN name LIKE 'ailang.%' OR name LIKE 'ailang %' THEN 'Local Usage'
				ELSE 'Other'
			END as label,
			COUNT(*) as span_count,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out,
			COALESCE(SUM(cost_usd), 0) as cost_usd
		FROM spans
		GROUP BY 1, 2
		ORDER BY cost_usd DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetBreakdownByModel returns cost/token breakdown by model
func (b *SQLiteBackend) GetBreakdownByModel(ctx context.Context) ([]BreakdownItem, error) {
	rows, err := b.store.DB().QueryContext(ctx, `
		SELECT
			COALESCE(model, 'unknown') as id,
			COALESCE(model, 'Unknown') as label,
			COUNT(*) as span_count,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out,
			COALESCE(SUM(cost_usd), 0) as cost_usd
		FROM spans
		WHERE model IS NOT NULL AND model != ''
		GROUP BY model
		ORDER BY cost_usd DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetBreakdownByWorkspace returns cost/token breakdown by workspace
func (b *SQLiteBackend) GetBreakdownByWorkspace(ctx context.Context) ([]BreakdownItem, error) {
	rows, err := b.store.DB().QueryContext(ctx, `
		SELECT
			w.id,
			w.name as label,
			COUNT(DISTINCT s.id) as span_count,
			COUNT(DISTINCT t.id) as task_count,
			COALESCE(SUM(s.tokens_in), 0) as tokens_in,
			COALESCE(SUM(s.tokens_out), 0) as tokens_out,
			COALESCE(SUM(s.cost_usd), 0) as cost_usd
		FROM workspaces w
		LEFT JOIN tasks t ON t.workspace_id = w.id
		LEFT JOIN spans s ON s.task_id = t.id
		GROUP BY w.id
		ORDER BY cost_usd DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TaskCount, &item.TokensIn, &item.TokensOut, &item.CostUSD); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ===== Filtered Queries =====

// GetFilteredHeatmapData returns daily activity data aggregated from spans
func (b *SQLiteBackend) GetFilteredHeatmapData(ctx context.Context, filter *ControlPlaneFilter, days int) ([]HeatmapDataPoint, error) {
	conditions, args := buildFilterConditions(filter)

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
func (b *SQLiteBackend) GetFilteredMetricsSummary(ctx context.Context, filter *ControlPlaneFilter) (*MetricsSummary, error) {
	if filter == nil || filter.IsEmpty() {
		return b.store.GetMetricsSummary()
	}

	// Build WHERE clause using shared helper (includes time range filtering)
	conditions, args := buildFilterConditions(filter)

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
				NULLIF(CAST(COUNT(*) AS REAL), 0) as success_rate
		FROM spans
		%s
	`, whereClause)

	row := b.store.DB().QueryRowContext(ctx, query, args...)
	var summary MetricsSummary
	var successRate sql.NullFloat64
	err := row.Scan(&summary.TotalSpans, &summary.TotalTasks, &summary.TotalAgents,
		&summary.TotalTokensIn, &summary.TotalTokensOut, &summary.TotalCostUSD, &successRate)
	if err != nil {
		return nil, err
	}
	if successRate.Valid {
		summary.SuccessRate = successRate.Float64
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

// GetFilteredBreakdownByProvider returns provider breakdown with filters applied
func (b *SQLiteBackend) GetFilteredBreakdownByProvider(ctx context.Context, filter *ControlPlaneFilter) ([]BreakdownItem, error) {
	if filter == nil || filter.IsEmpty() {
		return b.GetBreakdownByProvider(ctx)
	}

	// Build base conditions (includes time range)
	conditions, args := buildFilterConditions(filter)
	// Add provider-specific condition
	conditions = append([]string{"provider IS NOT NULL AND provider != ''"}, conditions...)

	whereClause := "WHERE " + conditions[0]
	for _, c := range conditions[1:] {
		whereClause += " AND " + c
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(provider, 'unknown') as id,
			COALESCE(provider, 'Unknown') as label,
			COUNT(*) as span_count,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out,
			COALESCE(SUM(cost_usd), 0) as cost_usd
		FROM spans
		%s
		GROUP BY provider
		ORDER BY cost_usd DESC
	`, whereClause)

	rows, err := b.store.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetFilteredBreakdownBySourceType returns source type breakdown with filters applied
func (b *SQLiteBackend) GetFilteredBreakdownBySourceType(ctx context.Context, filter *ControlPlaneFilter) ([]BreakdownItem, error) {
	if filter == nil || filter.IsEmpty() {
		return b.GetBreakdownBySourceType(ctx)
	}

	// Build conditions using shared helper (includes time range)
	// Note: For source type breakdown, we exclude source_type from filter conditions
	// since the query groups BY source type
	tempFilter := &ControlPlaneFilter{
		Provider:  filter.Provider,
		Model:     filter.Model,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}
	conditions, args := buildFilterConditions(tempFilter)

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			whereClause += " AND " + c
		}
	}

	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN name LIKE 'api_request%%' THEN 'eval'
				WHEN name LIKE 'eval.%%' THEN 'eval'
				WHEN name LIKE 'coordinator.%%' OR name LIKE 'claude.execute%%' THEN 'coordinator'
				WHEN name LIKE 'anthropic.%%' OR name LIKE 'gemini.%%' OR name LIKE 'openai.%%' THEN 'direct_api'
				WHEN name LIKE 'ailang.exec%%' THEN 'exec'
				WHEN name LIKE 'ailang.%%' OR name LIKE 'ailang %%' THEN 'local'
				ELSE 'other'
			END as id,
			CASE
				WHEN name LIKE 'api_request%%' THEN 'Eval Benchmarks'
				WHEN name LIKE 'eval.%%' THEN 'Eval Benchmarks'
				WHEN name LIKE 'coordinator.%%' OR name LIKE 'claude.execute%%' THEN 'Coordinator Tasks'
				WHEN name LIKE 'anthropic.%%' OR name LIKE 'gemini.%%' OR name LIKE 'openai.%%' THEN 'Direct API Calls'
				WHEN name LIKE 'ailang.exec%%' THEN 'Exec Tasks'
				WHEN name LIKE 'ailang.%%' OR name LIKE 'ailang %%' THEN 'Local Usage'
				ELSE 'Other'
			END as label,
			COUNT(*) as span_count,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out,
			COALESCE(SUM(cost_usd), 0) as cost_usd
		FROM spans
		%s
		GROUP BY 1, 2
		ORDER BY cost_usd DESC
	`, whereClause)

	rows, err := b.store.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetFilteredBreakdownByModel returns model breakdown with filters applied
func (b *SQLiteBackend) GetFilteredBreakdownByModel(ctx context.Context, filter *ControlPlaneFilter) ([]BreakdownItem, error) {
	if filter == nil || filter.IsEmpty() {
		return b.GetBreakdownByModel(ctx)
	}

	// Build conditions using shared helper (includes time range)
	// Note: For model breakdown, we exclude model from filter conditions
	// since the query groups BY model
	tempFilter := &ControlPlaneFilter{
		SourceType: filter.SourceType,
		Provider:   filter.Provider,
		StartDate:  filter.StartDate,
		EndDate:    filter.EndDate,
	}
	conditions, args := buildFilterConditions(tempFilter)
	// Add model-specific condition
	conditions = append([]string{"model IS NOT NULL AND model != ''"}, conditions...)

	whereClause := "WHERE " + conditions[0]
	for _, c := range conditions[1:] {
		whereClause += " AND " + c
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(model, 'unknown') as id,
			COALESCE(model, 'Unknown') as label,
			COUNT(*) as span_count,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out,
			COALESCE(SUM(cost_usd), 0) as cost_usd
		FROM spans
		%s
		GROUP BY model
		ORDER BY cost_usd DESC
		LIMIT 20
	`, whereClause)

	rows, err := b.store.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
