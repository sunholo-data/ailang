// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
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
// Uses resource_attributes (service.name, ailang.source) first, then falls back to span name patterns
// Note: JSON keys with dots must be quoted in json_extract (e.g., '$."service.name"')
func buildSourceTypeCondition(sourceType string) string {
	switch sourceType {
	case "claude_code":
		return `(json_extract(resource_attributes, '$."service.name"') = 'claude-code' OR json_extract(resource_attributes, '$."ailang.source"') = 'user' OR name LIKE 'claude_code.%')`
	case "eval":
		return `(json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' OR json_extract(resource_attributes, '$."ailang.source"') = 'eval' OR name LIKE 'eval.%')`
	case "coordinator":
		return `(json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' OR name LIKE 'coordinator.%' OR name LIKE 'claude.execute%')`
	case "direct_api":
		return "(name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%')"
	case "exec":
		return "(name LIKE 'ailang.exec%')"
	case "local":
		return "(name LIKE 'ailang.%' OR name LIKE 'ailang %')"
	case "other":
		// Exclude all known categories
		return `(
			json_extract(resource_attributes, '$."service.name"') IS NULL OR
			json_extract(resource_attributes, '$."service.name"') NOT IN ('claude-code', 'ailang-eval')
		) AND (
			json_extract(resource_attributes, '$."ailang.source"') IS NULL OR
			json_extract(resource_attributes, '$."ailang.source"') NOT IN ('user', 'coordinator', 'eval')
		) AND name NOT LIKE 'eval.%'
		  AND name NOT LIKE 'coordinator.%'
		  AND name NOT LIKE 'claude.execute%'
		  AND name NOT LIKE 'anthropic.%'
		  AND name NOT LIKE 'gemini.%'
		  AND name NOT LIKE 'openai.%'
		  AND name NOT LIKE 'ailang.%'
		  AND name NOT LIKE 'ailang %'
		  AND name NOT LIKE 'claude_code.%'`
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

// GetBreakdownBySourceType returns cost/token breakdown by source type (inferred from attributes + span names)
func (b *SQLiteBackend) GetBreakdownBySourceType(ctx context.Context) ([]BreakdownItem, error) {
	// Categorize spans by resource attributes first (service.name, ailang.source), then span name patterns
	// Priority: service.name/ailang.source > span name patterns
	// GROUP BY 1, 2 uses column position since SQLite doesn't support alias in GROUP BY
	rows, err := b.store.DB().QueryContext(ctx, `
		SELECT
			CASE
				-- Check service.name from resource_attributes first (keys with dots need quoting)
				WHEN json_extract(resource_attributes, '$."service.name"') = 'claude-code' THEN 'claude_code'
				WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' THEN 'eval'
				-- Check ailang.source attribute
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'user' THEN 'claude_code'
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' THEN 'coordinator'
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'eval' THEN 'eval'
				-- Fall back to span name patterns
				WHEN name LIKE 'eval.%' THEN 'eval'
				WHEN name LIKE 'coordinator.%' OR name LIKE 'claude.execute%' THEN 'coordinator'
				WHEN name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%' THEN 'direct_api'
				WHEN name LIKE 'ailang.exec%' THEN 'exec'
				WHEN name LIKE 'ailang.%' OR name LIKE 'ailang %' THEN 'local'
				WHEN name LIKE 'claude_code.%' THEN 'claude_code'
				ELSE 'other'
			END as id,
			CASE
				-- Check service.name from resource_attributes first (keys with dots need quoting)
				WHEN json_extract(resource_attributes, '$."service.name"') = 'claude-code' THEN 'Claude Code'
				WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' THEN 'Eval Benchmarks'
				-- Check ailang.source attribute
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'user' THEN 'Claude Code'
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' THEN 'Coordinator Tasks'
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'eval' THEN 'Eval Benchmarks'
				-- Fall back to span name patterns
				WHEN name LIKE 'eval.%' THEN 'Eval Benchmarks'
				WHEN name LIKE 'coordinator.%' OR name LIKE 'claude.execute%' THEN 'Coordinator Tasks'
				WHEN name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%' THEN 'Direct API Calls'
				WHEN name LIKE 'ailang.exec%' THEN 'Exec Tasks'
				WHEN name LIKE 'ailang.%' OR name LIKE 'ailang %' THEN 'Local Usage'
				WHEN name LIKE 'claude_code.%' THEN 'Claude Code'
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
				-- Check service.name from resource_attributes first (keys with dots need quoting)
				WHEN json_extract(resource_attributes, '$."service.name"') = 'claude-code' THEN 'claude_code'
				WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' THEN 'eval'
				-- Check ailang.source attribute
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'user' THEN 'claude_code'
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' THEN 'coordinator'
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'eval' THEN 'eval'
				-- Fall back to span name patterns
				WHEN name LIKE 'eval.%%' THEN 'eval'
				WHEN name LIKE 'coordinator.%%' OR name LIKE 'claude.execute%%' THEN 'coordinator'
				WHEN name LIKE 'anthropic.%%' OR name LIKE 'gemini.%%' OR name LIKE 'openai.%%' THEN 'direct_api'
				WHEN name LIKE 'ailang.exec%%' THEN 'exec'
				WHEN name LIKE 'ailang.%%' OR name LIKE 'ailang %%' THEN 'local'
				WHEN name LIKE 'claude_code.%%' THEN 'claude_code'
				ELSE 'other'
			END as id,
			CASE
				-- Check service.name from resource_attributes first (keys with dots need quoting)
				WHEN json_extract(resource_attributes, '$."service.name"') = 'claude-code' THEN 'Claude Code'
				WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' THEN 'Eval Benchmarks'
				-- Check ailang.source attribute
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'user' THEN 'Claude Code'
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' THEN 'Coordinator Tasks'
				WHEN json_extract(resource_attributes, '$."ailang.source"') = 'eval' THEN 'Eval Benchmarks'
				-- Fall back to span name patterns
				WHEN name LIKE 'eval.%%' THEN 'Eval Benchmarks'
				WHEN name LIKE 'coordinator.%%' OR name LIKE 'claude.execute%%' THEN 'Coordinator Tasks'
				WHEN name LIKE 'anthropic.%%' OR name LIKE 'gemini.%%' OR name LIKE 'openai.%%' THEN 'Direct API Calls'
				WHEN name LIKE 'ailang.exec%%' THEN 'Exec Tasks'
				WHEN name LIKE 'ailang.%%' OR name LIKE 'ailang %%' THEN 'Local Usage'
				WHEN name LIKE 'claude_code.%%' THEN 'Claude Code'
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

// ===== Claude Code Event Types =====

// ClaudeCodeEvent represents a Claude Code API request span formatted as an event
// for the Event Queue. Each api_request span becomes an event that can be clicked
// to show its hierarchy (tool calls correlated by timestamp).
type ClaudeCodeEvent struct {
	ID         string  `json:"id"`           // span_id (also used as task_id for hierarchy lookup)
	CreatedAt  string  `json:"created_at"`   // ISO8601 timestamp (matches inbox message format)
	Type       string  `json:"message_type"` // "claude_code_turn"
	FromAgent  string  `json:"from_agent"`   // Agent ID (e.g., "design-doc-creator") or "claude-code" for user sessions
	ToInbox    string  `json:"to_inbox"`     // Agent inbox (e.g., "design-doc-creator") or "user" for user sessions
	Title      string  `json:"title"`        // "Claude Code Turn ($X.XX)"
	TaskID     string  `json:"task_id"`      // Same as ID for hierarchy lookup
	Status     string  `json:"status"`       // "read" (not actionable like inbox messages)
	CostUSD    float64 `json:"cost_usd"`
	TokensIn   int64   `json:"tokens_in"`
	TokensOut  int64   `json:"tokens_out"`
	DurationMs int     `json:"duration_ms"`
	Workspace  string  `json:"workspace,omitempty"` // Working directory (from resource attributes)
}

// TaskAgentLookup is a callback to resolve coordinator task_id to agent info.
// Returns: agentID (used for FromAgent), inbox (used for ToInbox), title
// If the task_id is not found, returns empty strings (not an error).
type TaskAgentLookup func(ctx context.Context, taskID string) (agentID, inbox, title string, err error)

// GetClaudeCodeEvents returns Claude Code sessions aggregated by session.id.
// Multiple api_request spans (turns) in one session are aggregated into a single event.
// This provides a cleaner Event Queue with one entry per Claude Code conversation.
func (b *SQLiteBackend) GetClaudeCodeEvents(ctx context.Context, limit int) ([]ClaudeCodeEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	// Aggregate by session.id from attributes - sum cost/tokens, get latest timestamp
	// Use MIN(id) as the representative span_id for hierarchy lookup
	query := `
		SELECT
			json_extract(attributes, '$."session.id"') as session_id,
			MIN(id) as first_span_id,
			MAX(start_time) as latest_time,
			COUNT(*) as turn_count,
			SUM(COALESCE(duration_ms, 0)) as total_duration_ms,
			SUM(COALESCE(cost_usd, 0)) as total_cost_usd,
			SUM(COALESCE(tokens_in, 0)) as total_tokens_in,
			SUM(COALESCE(tokens_out, 0)) as total_tokens_out
		FROM spans
		WHERE name = 'api_request'
		  AND json_extract(resource_attributes, '$."service.name"') = 'claude-code'
		  AND json_extract(attributes, '$."session.id"') IS NOT NULL
		GROUP BY json_extract(attributes, '$."session.id"')
		ORDER BY latest_time DESC
		LIMIT ?
	`

	rows, err := b.store.DB().QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ClaudeCodeEvent
	for rows.Next() {
		var sessionID, firstSpanID, latestTimeRaw string
		var turnCount int
		var totalDurationMs int
		var totalCostUSD float64
		var totalTokensIn, totalTokensOut int64

		if err := rows.Scan(&sessionID, &firstSpanID, &latestTimeRaw, &turnCount, &totalDurationMs, &totalCostUSD, &totalTokensIn, &totalTokensOut); err != nil {
			return nil, err
		}

		// Convert SQLite datetime format to RFC3339 (replace space with T)
		latestTime := strings.Replace(latestTimeRaw, " ", "T", 1)

		// Format cost for title
		costStr := fmt.Sprintf("$%.2f", totalCostUSD)
		if totalCostUSD < 0.01 && totalCostUSD > 0 {
			costStr = fmt.Sprintf("$%.4f", totalCostUSD)
		}

		// Format duration for title
		durationStr := fmt.Sprintf("%.1fs", float64(totalDurationMs)/1000)
		if totalDurationMs >= 60000 {
			durationStr = fmt.Sprintf("%.1fm", float64(totalDurationMs)/60000)
		}

		// Build title showing turns count and totals
		title := fmt.Sprintf("Claude Code Session (%d turns, %s, %s)", turnCount, costStr, durationStr)

		events = append(events, ClaudeCodeEvent{
			ID:         sessionID,  // Use session_id as the event ID
			CreatedAt:  latestTime, // Most recent activity
			Type:       "claude_code_session",
			FromAgent:  "claude-code",
			ToInbox:    "user",
			Title:      title,
			TaskID:     sessionID, // Use session_id for hierarchy lookup (aggregates all turns)
			Status:     "read",
			CostUSD:    totalCostUSD,
			TokensIn:   totalTokensIn,
			TokensOut:  totalTokensOut,
			DurationMs: totalDurationMs,
		})
	}

	return events, rows.Err()
}

// GetClaudeCodeEventsWithLookup returns Claude Code sessions with agent info resolved.
// For sessions spawned by coordinator tasks, FromAgent and ToInbox are set from agent_assignments.
// For user-initiated sessions (no coordinator task), defaults to "claude-code" / "user".
// Note: The lookup parameter is kept for API compatibility but is no longer used - agent info
// comes directly from observatory.db via JOIN with agent_assignments.
func (b *SQLiteBackend) GetClaudeCodeEventsWithLookup(ctx context.Context, limit int, lookup TaskAgentLookup) ([]ClaudeCodeEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	// Aggregate by session.id, join with agent_assignments to get agent info
	// Uses spans.task_id COLUMN (not JSON attribute) to link to agent_assignments
	query := `
		SELECT
			json_extract(s.attributes, '$."session.id"') as session_id,
			MIN(s.id) as first_span_id,
			MAX(s.start_time) as latest_time,
			COUNT(*) as turn_count,
			SUM(COALESCE(s.duration_ms, 0)) as total_duration_ms,
			SUM(COALESCE(s.cost_usd, 0)) as total_cost_usd,
			SUM(COALESCE(s.tokens_in, 0)) as total_tokens_in,
			SUM(COALESCE(s.tokens_out, 0)) as total_tokens_out,
			MAX(s.task_id) as coord_task_id,
			MAX(aa.agent_id) as agent_id
		FROM spans s
		LEFT JOIN agent_assignments aa ON s.task_id = aa.task_id
		WHERE s.name = 'api_request'
		  AND json_extract(s.resource_attributes, '$."service.name"') = 'claude-code'
		  AND json_extract(s.attributes, '$."session.id"') IS NOT NULL
		GROUP BY json_extract(s.attributes, '$."session.id"')
		ORDER BY latest_time DESC
		LIMIT ?
	`

	rows, err := b.store.DB().QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ClaudeCodeEvent
	for rows.Next() {
		var sessionID, firstSpanID, latestTimeRaw string
		var turnCount int
		var totalDurationMs int
		var totalCostUSD float64
		var totalTokensIn, totalTokensOut int64
		var coordTaskID, agentID sql.NullString

		if err := rows.Scan(&sessionID, &firstSpanID, &latestTimeRaw, &turnCount, &totalDurationMs, &totalCostUSD, &totalTokensIn, &totalTokensOut, &coordTaskID, &agentID); err != nil {
			return nil, err
		}

		// Convert SQLite datetime format to RFC3339 (replace space with T)
		latestTime := strings.Replace(latestTimeRaw, " ", "T", 1)

		// Format cost for title
		costStr := fmt.Sprintf("$%.2f", totalCostUSD)
		if totalCostUSD < 0.01 && totalCostUSD > 0 {
			costStr = fmt.Sprintf("$%.4f", totalCostUSD)
		}

		// Format duration for title
		durationStr := fmt.Sprintf("%.1fs", float64(totalDurationMs)/1000)
		if totalDurationMs >= 60000 {
			durationStr = fmt.Sprintf("%.1fm", float64(totalDurationMs)/60000)
		}

		// Build title showing turns count and totals
		title := fmt.Sprintf("Claude Code Session (%d turns, %s, %s)", turnCount, costStr, durationStr)

		// Default values for user-initiated sessions
		fromAgent := "claude-code"
		toInbox := "user"

		// If we have agent_id from agent_assignments, use it
		// (By convention, agent id == inbox in agent config)
		if agentID.Valid && agentID.String != "" {
			fromAgent = agentID.String
			toInbox = agentID.String
		}

		events = append(events, ClaudeCodeEvent{
			ID:         sessionID,  // Use session_id as the event ID
			CreatedAt:  latestTime, // Most recent activity
			Type:       "claude_code_session",
			FromAgent:  fromAgent,
			ToInbox:    toInbox,
			Title:      title,
			TaskID:     sessionID, // Use session_id for hierarchy lookup (aggregates all turns)
			Status:     "read",
			CostUSD:    totalCostUSD,
			TokensIn:   totalTokensIn,
			TokensOut:  totalTokensOut,
			DurationMs: totalDurationMs,
		})
	}

	return events, rows.Err()
}

// splitPath splits a file path into components (works for both Unix and Windows paths)
func splitPath(path string) []string {
	// Remove trailing slashes
	path = strings.TrimRight(path, "/\\")
	if path == "" {
		return nil
	}
	// Split on both / and \
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' || c == '\\' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// GetClaudeCodeHierarchy returns the hierarchy for a Claude Code session.
// The sessionID is the session.id from Claude Code telemetry (UUID format).
//
// Claude Code telemetry creates a SEPARATE trace for each span (api_request and tool calls).
// This means we can't use standard trace hierarchy building (which relies on parent_span_id).
// Instead, we build a turn-based hierarchy using timestamp correlation:
//   - api_request spans = "turns" (top level)
//   - tool calls that occur before the next api_request = children of that turn
func (b *SQLiteBackend) GetClaudeCodeHierarchy(ctx context.Context, sessionID string) (*TaskHierarchy, error) {
	// Fetch all spans for this session by session.id attribute
	spans, err := b.getSpansBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list spans for session: %w", err)
	}
	if len(spans) == 0 {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Sort spans by start_time (already sorted by query, but ensure)
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].StartTime.Before(spans[j].StartTime)
	})

	// Separate api_requests (turns) from tool calls
	var apiRequests []*Span
	var toolCalls []*Span
	for _, span := range spans {
		if span.Name == "api_request" {
			apiRequests = append(apiRequests, span)
		} else if strings.HasPrefix(span.Name, "claude_code.tool.") {
			toolCalls = append(toolCalls, span)
		}
	}

	// Build turn-based hierarchy: each api_request is a turn with tool calls as children
	var turnNodes []*SpanNode
	for i, apiReq := range apiRequests {
		// Find the end of this turn's time window
		var turnEnd time.Time
		if i+1 < len(apiRequests) {
			turnEnd = apiRequests[i+1].StartTime
		} else {
			// Last turn: use a far future time
			turnEnd = time.Now().Add(24 * time.Hour)
		}

		// Find tool calls that belong to this turn (started after this api_request, before next)
		var turnChildren []*SpanNode
		for _, tool := range toolCalls {
			if tool.StartTime.After(apiReq.StartTime) && tool.StartTime.Before(turnEnd) {
				turnChildren = append(turnChildren, &SpanNode{Span: tool})
			}
		}

		turnNodes = append(turnNodes, &SpanNode{
			Span:     apiReq,
			Children: turnChildren,
		})
	}

	// Calculate totals for the session
	var totalTokensIn, totalTokensOut int64
	var totalCost float64
	var totalDuration int64
	for _, span := range spans {
		totalTokensIn += span.TokensIn
		totalTokensOut += span.TokensOut
		totalCost += span.CostUSD
		totalDuration += span.DurationMs
	}

	// Create a virtual "session" span as the root containing all turns
	sessionSpan := &SpanNode{
		Span: &Span{
			ID:         sessionID,
			Name:       "claude_code.session",
			StartTime:  spans[0].StartTime,
			DurationMs: totalDuration,
			Provider:   "claude",
			TokensIn:   totalTokensIn,
			TokensOut:  totalTokensOut,
			CostUSD:    totalCost,
		},
		Children: turnNodes,
	}

	// Create a single "session" trace containing all turns
	sessionTrace := &TraceHierarchy{
		TraceID:  sessionID, // Use session ID as trace ID for the virtual trace
		RootSpan: sessionSpan,
		Spans:    []*SpanNode{sessionSpan}, // Include session root so frontend sees hierarchy
		Summary: &HierarchyTraceSummary{
			SpanCount:    len(spans),
			TotalTokens:  totalTokensIn + totalTokensOut,
			TotalCostUSD: totalCost,
			DurationMs:   totalDuration,
		},
	}

	// Build short ID for display
	shortID := sessionID
	if len(sessionID) >= 8 {
		shortID = sessionID[:8]
	}

	return &TaskHierarchy{
		Agents: []*AgentHierarchy{{
			Agent: &AgentAssignment{
				ID:       "claude-code-session-" + shortID,
				AgentID:  "claude-code",
				Provider: "claude",
			},
			Traces: []*TraceHierarchy{sessionTrace},
		}},
	}, nil
}

// getSpansBySessionID retrieves ALL spans for a Claude Code session.
// This queries by session.id attribute to support existing spans that don't have task_id set.
func (b *SQLiteBackend) getSpansBySessionID(ctx context.Context, sessionID string) ([]*Span, error) {
	query := `
		SELECT id, trace_id, parent_span_id, name, start_time, end_time, duration_ms,
		       status, attributes, resource_attributes, provider, model,
		       tokens_in, tokens_out, cost_usd, task_id, agent_assignment_id
		FROM spans
		WHERE json_extract(resource_attributes, '$."service.name"') = 'claude-code'
		  AND json_extract(attributes, '$."session.id"') = ?
		ORDER BY start_time ASC
		LIMIT 1000
	`

	rows, err := b.store.DB().QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []*Span
	for rows.Next() {
		var span Span
		var endTime sql.NullTime
		var parentSpanID, status, provider, model, taskID, agentAssignmentID sql.NullString
		var attrs, resourceAttrs sql.NullString

		if err := rows.Scan(
			&span.ID, &span.TraceID, &parentSpanID, &span.Name,
			&span.StartTime, &endTime, &span.DurationMs,
			&status, &attrs, &resourceAttrs, &provider, &model,
			&span.TokensIn, &span.TokensOut, &span.CostUSD,
			&taskID, &agentAssignmentID,
		); err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if endTime.Valid {
			span.EndTime = &endTime.Time
		}
		if status.Valid {
			span.Status = SpanStatus(status.String)
		}
		if provider.Valid {
			span.Provider = Provider(provider.String)
		}
		if model.Valid {
			span.Model = model.String
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if attrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(attrs.String), &m) == nil {
				span.Attributes = m
			}
		}
		if resourceAttrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(resourceAttrs.String), &m) == nil {
				span.ResourceAttributes = m
			}
		}

		spans = append(spans, &span)
	}

	return spans, rows.Err()
}

// getSessionSpans retrieves all api_request spans for a Claude Code session.
func (b *SQLiteBackend) getSessionSpans(ctx context.Context, sessionID string) ([]*Span, error) {
	query := `
		SELECT id, trace_id, parent_span_id, name, start_time, end_time, duration_ms,
		       status, attributes, resource_attributes, provider, model,
		       tokens_in, tokens_out, cost_usd, task_id, agent_assignment_id
		FROM spans
		WHERE name = 'api_request'
		  AND json_extract(resource_attributes, '$."service.name"') = 'claude-code'
		  AND json_extract(attributes, '$."session.id"') = ?
		ORDER BY start_time ASC
	`

	rows, err := b.store.DB().QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []*Span
	for rows.Next() {
		var span Span
		var endTime sql.NullTime
		var parentSpanID, status, provider, model, taskID, agentAssignmentID sql.NullString
		var attrs, resourceAttrs sql.NullString

		if err := rows.Scan(
			&span.ID, &span.TraceID, &parentSpanID, &span.Name,
			&span.StartTime, &endTime, &span.DurationMs,
			&status, &attrs, &resourceAttrs, &provider, &model,
			&span.TokensIn, &span.TokensOut, &span.CostUSD,
			&taskID, &agentAssignmentID,
		); err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if endTime.Valid {
			span.EndTime = &endTime.Time
		}
		if status.Valid {
			span.Status = SpanStatus(status.String)
		}
		if provider.Valid {
			span.Provider = Provider(provider.String)
		}
		if model.Valid {
			span.Model = model.String
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if attrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(attrs.String), &m) == nil {
				span.Attributes = m
			}
		}
		if resourceAttrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(resourceAttrs.String), &m) == nil {
				span.ResourceAttributes = m
			}
		}

		spans = append(spans, &span)
	}

	return spans, rows.Err()
}

// getSpanByID retrieves a single span by its ID.
func (b *SQLiteBackend) getSpanByID(ctx context.Context, spanID string) (*Span, error) {
	query := `
		SELECT id, trace_id, parent_span_id, name, start_time, end_time, duration_ms,
		       status, attributes, resource_attributes, provider, model,
		       tokens_in, tokens_out, cost_usd, task_id, agent_assignment_id
		FROM spans WHERE id = ?
	`
	row := b.store.DB().QueryRowContext(ctx, query, spanID)

	var span Span
	var endTime sql.NullTime
	var parentSpanID, status, provider, model, taskID, agentAssignmentID sql.NullString
	var attrs, resourceAttrs sql.NullString

	if err := row.Scan(
		&span.ID, &span.TraceID, &parentSpanID, &span.Name,
		&span.StartTime, &endTime, &span.DurationMs,
		&status, &attrs, &resourceAttrs, &provider, &model,
		&span.TokensIn, &span.TokensOut, &span.CostUSD,
		&taskID, &agentAssignmentID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if parentSpanID.Valid {
		span.ParentSpanID = parentSpanID.String
	}
	if endTime.Valid {
		span.EndTime = &endTime.Time
	}
	if status.Valid {
		span.Status = SpanStatus(status.String)
	}
	if provider.Valid {
		span.Provider = Provider(provider.String)
	}
	if model.Valid {
		span.Model = model.String
	}
	if taskID.Valid {
		span.TaskID = taskID.String
	}
	if agentAssignmentID.Valid {
		span.AgentAssignmentID = agentAssignmentID.String
	}
	if attrs.Valid {
		var m map[string]interface{}
		if json.Unmarshal([]byte(attrs.String), &m) == nil {
			span.Attributes = m
		}
	}
	if resourceAttrs.Valid {
		var m map[string]interface{}
		if json.Unmarshal([]byte(resourceAttrs.String), &m) == nil {
			span.ResourceAttributes = m
		}
	}

	return &span, nil
}

// getToolCallsInWindow finds tool call spans within a time window.
// These are spans from claude_code.tool.* that started within the given time range.
func (b *SQLiteBackend) getToolCallsInWindow(ctx context.Context, start time.Time, end *time.Time) ([]*Span, error) {
	if end == nil {
		return nil, nil // Can't correlate without end time
	}

	query := `
		SELECT id, trace_id, parent_span_id, name, start_time, end_time, duration_ms,
		       status, attributes, resource_attributes, provider, model,
		       tokens_in, tokens_out, cost_usd, task_id, agent_assignment_id
		FROM spans
		WHERE (
			name LIKE 'claude_code.tool.%'
			OR (json_extract(resource_attributes, '$."service.name"') = 'claude-code' AND name NOT IN ('api_request', 'user_prompt'))
		)
		AND start_time >= ? AND start_time <= ?
		ORDER BY start_time ASC
		LIMIT 100
	`

	rows, err := b.store.DB().QueryContext(ctx, query, start, *end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []*Span
	for rows.Next() {
		var span Span
		var endTime sql.NullTime
		var parentSpanID, status, provider, model, taskID, agentAssignmentID sql.NullString
		var attrs, resourceAttrs sql.NullString

		if err := rows.Scan(
			&span.ID, &span.TraceID, &parentSpanID, &span.Name,
			&span.StartTime, &endTime, &span.DurationMs,
			&status, &attrs, &resourceAttrs, &provider, &model,
			&span.TokensIn, &span.TokensOut, &span.CostUSD,
			&taskID, &agentAssignmentID,
		); err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if endTime.Valid {
			span.EndTime = &endTime.Time
		}
		if status.Valid {
			span.Status = SpanStatus(status.String)
		}
		if provider.Valid {
			span.Provider = Provider(provider.String)
		}
		if model.Valid {
			span.Model = model.String
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if attrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(attrs.String), &m) == nil {
				span.Attributes = m
			}
		}
		if resourceAttrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(resourceAttrs.String), &m) == nil {
				span.ResourceAttributes = m
			}
		}

		spans = append(spans, &span)
	}

	return spans, rows.Err()
}
