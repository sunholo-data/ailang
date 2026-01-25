// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
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
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	SpanCount  int     `json:"span_count"`
	TaskCount  int     `json:"task_count,omitempty"`
	TokensIn   int64   `json:"tokens_in"`
	TokensOut  int64   `json:"tokens_out"`
	CostUSD    float64 `json:"cost_usd"`
	DurationMs int64   `json:"duration_ms"` // Total execution time in ms

	// Cache metrics
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	CacheSavingsUSD     float64 `json:"cache_savings_usd"`
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
//
// Canonical source types (must match InferInboxSourceType in handlers_inbox.go):
// - user_session: Claude Code user sessions
// - eval: Eval suite runs
// - coordinator: Coordinator-managed agent tasks
// - github: GitHub sync messages (inbox only, no spans)
// - messaging: Agent-to-agent messages
// - cli: CLI tool usage
// - direct_api: Direct API calls
// - other: Catch-all
func buildSourceTypeCondition(sourceType string) string {
	switch sourceType {
	case "user_session", "claude_code":
		// Match Claude Code spans from direct user sessions, but EXCLUDE coordinator-spawned sessions
		// This ensures coordinator tasks get proper cost attribution
		return `(
			(json_extract(resource_attributes, '$."service.name"') = 'claude-code' OR
			 json_extract(resource_attributes, '$."ailang.source"') = 'user' OR
			 name LIKE 'claude_code.%')
			AND COALESCE(json_extract(resource_attributes, '$."ailang.source"'), 'user') != 'coordinator'
		)`
	case "eval":
		return `(json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' OR json_extract(resource_attributes, '$."ailang.source"') = 'eval' OR name LIKE 'eval.%')`
	case "coordinator":
		return `(json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' OR name LIKE 'coordinator.%' OR name LIKE 'claude.execute%')`
	case "github":
		// GitHub has no spans - this condition matches nothing intentionally
		// GitHub messages are filtered in handlers_inbox.go via InferInboxSourceType
		return "1=0"
	case "messaging":
		return "(name LIKE 'messages.%')"
	case "cli":
		return "(name LIKE 'ailang.%' OR name LIKE 'ailang %' OR name LIKE 'compile%' OR name LIKE 'check.%')"
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
// The wsConfig parameter is used to reverse-map workspace IDs to path patterns for filtering.
func buildFilterConditions(filter *ControlPlaneFilter, wsConfig WorkspaceMapping) ([]string, []interface{}) {
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
	if filter.Workspace != "" {
		// Workspace filter: use reverse mapping to find path patterns that match
		// Spans store raw paths in process.cwd, but filters use org/repo workspace IDs
		var patterns []string
		if wsConfig != nil {
			patterns = wsConfig.GetPathPatternsForWorkspace(filter.Workspace)
		}
		if len(patterns) > 0 {
			// Build OR clause for all matching patterns
			var orClauses []string
			for _, p := range patterns {
				orClauses = append(orClauses, `json_extract(resource_attributes, '$."process.cwd"') LIKE ?`)
				args = append(args, p)
			}
			conditions = append(conditions, "("+strings.Join(orClauses, " OR ")+")")
		} else {
			// Fallback: direct match if no mapping found (supports raw path filtering)
			conditions = append(conditions, `json_extract(resource_attributes, '$."process.cwd"') LIKE ?`)
			args = append(args, "%"+filter.Workspace+"%")
		}
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
			COALESCE(SUM(cost_usd), 0) as cost_usd,
			COALESCE(SUM(duration_ms), 0) as duration_ms,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens
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
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD, &item.DurationMs, &item.CacheReadTokens, &item.CacheCreationTokens); err != nil {
			return nil, err
		}
		// Calculate cache savings (90% discount on cache reads)
		if item.CacheReadTokens > 0 {
			item.CacheSavingsUSD = CalculateCacheSavings("", item.CacheReadTokens)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetBreakdownBySourceType returns cost/token breakdown by source type (inferred from root span of each trace)
func (b *SQLiteBackend) GetBreakdownBySourceType(ctx context.Context) ([]BreakdownItem, error) {
	// Costs are attributed to the INITIATING SOURCE (root span of each trace).
	// This ensures that API calls made within a user session are attributed to "User Sessions",
	// not "Direct API". The root span determines the source category for all spans in its trace.
	rows, err := b.store.DB().QueryContext(ctx, `
		WITH root_categories AS (
			-- Find root span of each trace and categorize by INITIATING SOURCE
			-- Priority: ailang.source attribute (explicit) > service.name > span name patterns
			-- This ensures coordinator-spawned Claude Code sessions are attributed to "Coordinator Tasks"
			SELECT
				trace_id,
				CASE
					-- 1. Check ailang.source FIRST (explicit source from AILANG executor)
					-- Critical for proper cost attribution: GitHub → Coordinator → Claude Code
					WHEN json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' THEN 'coordinator'
					WHEN json_extract(resource_attributes, '$."ailang.source"') = 'eval' THEN 'eval'
					-- 2. Then check service.name (generic tool identity)
					WHEN json_extract(resource_attributes, '$."service.name"') = 'claude-code' THEN 'user_session'
					WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' THEN 'eval'
					WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-coordinator' THEN 'coordinator'
					-- 3. Check specific span name patterns
					WHEN name LIKE 'eval.%' THEN 'eval'
					WHEN name LIKE 'coordinator.%' OR name LIKE 'claude.execute%' OR name LIKE 'exec.%' THEN 'coordinator'
					WHEN name LIKE 'messages.%' THEN 'messaging'
					WHEN name LIKE 'ailang-%' THEN 'server'
					WHEN name LIKE 'ailang.exec%' THEN 'coordinator'
					WHEN name LIKE 'ailang.%' OR name LIKE 'ailang %' OR name LIKE 'compile%' OR name LIKE 'check.%' THEN 'cli'
					WHEN name LIKE 'claude_code.%' THEN 'user_session'
					-- 4. API calls without service.name are truly direct API usage
					WHEN name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%' THEN 'direct_api'
					WHEN name IN ('api_request', 'api_error', 'call_llm', 'invocation') THEN 'direct_api'
					ELSE 'other'
				END as source_id,
				CASE
					-- Same priority: ailang.source > service.name > span name
					WHEN json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' THEN 'Coordinator Tasks'
					WHEN json_extract(resource_attributes, '$."ailang.source"') = 'eval' THEN 'Eval Benchmarks'
					WHEN json_extract(resource_attributes, '$."service.name"') = 'claude-code' THEN 'User Sessions'
					WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' THEN 'Eval Benchmarks'
					WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-coordinator' THEN 'Coordinator Tasks'
					WHEN name LIKE 'eval.%' THEN 'Eval Benchmarks'
					WHEN name LIKE 'coordinator.%' OR name LIKE 'claude.execute%' OR name LIKE 'exec.%' THEN 'Coordinator Tasks'
					WHEN name LIKE 'messages.%' THEN 'Messaging'
					WHEN name LIKE 'ailang-%' THEN 'Server'
					WHEN name LIKE 'ailang.exec%' THEN 'Coordinator Tasks'
					WHEN name LIKE 'ailang.%' OR name LIKE 'ailang %' OR name LIKE 'compile%' OR name LIKE 'check.%' THEN 'CLI Usage'
					WHEN name LIKE 'claude_code.%' THEN 'User Sessions'
					WHEN name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%' THEN 'Direct API'
					WHEN name IN ('api_request', 'api_error', 'call_llm', 'invocation') THEN 'Direct API'
					ELSE 'Other'
				END as source_label
			FROM spans
			WHERE parent_span_id IS NULL OR parent_span_id = ''
		)
		SELECT
			COALESCE(r.source_id, 'other') as id,
			COALESCE(r.source_label, 'Other') as label,
			COUNT(*) as span_count,
			COALESCE(SUM(s.tokens_in), 0) as tokens_in,
			COALESCE(SUM(s.tokens_out), 0) as tokens_out,
			COALESCE(SUM(s.cost_usd), 0) as cost_usd,
			COALESCE(SUM(s.duration_ms), 0) as duration_ms,
			COALESCE(SUM(s.cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(s.cache_creation_tokens), 0) as cache_creation_tokens
		FROM spans s
		LEFT JOIN root_categories r ON s.trace_id = r.trace_id
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
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD, &item.DurationMs, &item.CacheReadTokens, &item.CacheCreationTokens); err != nil {
			return nil, err
		}
		// Calculate cache savings
		if item.CacheReadTokens > 0 {
			item.CacheSavingsUSD = CalculateCacheSavings("", item.CacheReadTokens)
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
			COALESCE(SUM(cost_usd), 0) as cost_usd,
			COALESCE(SUM(duration_ms), 0) as duration_ms,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens
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
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD, &item.DurationMs, &item.CacheReadTokens, &item.CacheCreationTokens); err != nil {
			return nil, err
		}
		// Calculate cache savings
		if item.CacheReadTokens > 0 {
			item.CacheSavingsUSD = CalculateCacheSavings("", item.CacheReadTokens)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetBreakdownByWorkspace returns cost/token breakdown by workspace.
// Extracts workspace directly from spans' process.cwd resource attribute.
// WorkspaceMapping provides workspace path-to-ID and ID-to-label mappings.
// This interface allows the observatory package to use workspace config without importing coordinator.
type WorkspaceMapping interface {
	// BuildWorkspaceMappingSQL returns a SQL CASE statement mapping cwdColumn values to workspace IDs.
	BuildWorkspaceMappingSQL(cwdColumn string) string
	// GetWorkspaceLabel returns a human-friendly label for a workspace ID.
	GetWorkspaceLabel(workspaceID string) string
	// GetPathPatternsForWorkspace returns SQL LIKE patterns that match a workspace ID.
	// Used for reverse mapping (workspace ID → path patterns) when filtering spans.
	GetPathPatternsForWorkspace(workspaceID string) []string
}

// GetBreakdownByWorkspace groups spans by workspace using config-driven path mapping.
// Note: Claude Code spans don't include process.cwd, so they show as "Unknown Workspace".
func (b *SQLiteBackend) GetBreakdownByWorkspace(ctx context.Context) ([]BreakdownItem, error) {
	// Use default mapping (returns workspace IDs without config-driven mapping)
	return b.GetFilteredBreakdownByWorkspace(ctx, nil, nil)
}

// GetBreakdownByWorkspaceWithMapping uses the provided mapping to convert file paths to workspace IDs.
func (b *SQLiteBackend) GetBreakdownByWorkspaceWithMapping(ctx context.Context, mapping WorkspaceMapping, wsConfig WorkspaceMapping) ([]BreakdownItem, error) {
	return b.GetFilteredBreakdownByWorkspaceWithMapping(ctx, nil, mapping, wsConfig)
}

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

// GetFilteredBreakdownByProvider returns provider breakdown with filters applied
func (b *SQLiteBackend) GetFilteredBreakdownByProvider(ctx context.Context, filter *ControlPlaneFilter, wsConfig WorkspaceMapping) ([]BreakdownItem, error) {
	if filter == nil || filter.IsEmpty() {
		return b.GetBreakdownByProvider(ctx)
	}

	// Build base conditions (includes time range)
	conditions, args := buildFilterConditions(filter, wsConfig)
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
			COALESCE(SUM(cost_usd), 0) as cost_usd,
			COALESCE(SUM(duration_ms), 0) as duration_ms,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens
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
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD, &item.DurationMs, &item.CacheReadTokens, &item.CacheCreationTokens); err != nil {
			return nil, err
		}
		// Calculate cache savings
		if item.CacheReadTokens > 0 {
			item.CacheSavingsUSD = CalculateCacheSavings("", item.CacheReadTokens)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetFilteredBreakdownBySourceType returns source type breakdown with filters applied
func (b *SQLiteBackend) GetFilteredBreakdownBySourceType(ctx context.Context, filter *ControlPlaneFilter, wsConfig WorkspaceMapping) ([]BreakdownItem, error) {
	if filter == nil || filter.IsEmpty() {
		return b.GetBreakdownBySourceType(ctx)
	}

	// Build conditions using shared helper (includes time range)
	// Note: For source type breakdown, we exclude source_type from filter conditions
	// since the query groups BY source type
	tempFilter := &ControlPlaneFilter{
		Provider:  filter.Provider,
		Model:     filter.Model,
		Workspace: filter.Workspace,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}
	conditions, args := buildFilterConditions(tempFilter, wsConfig)

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			whereClause += " AND " + c
		}
	}

	// Costs are attributed to the INITIATING SOURCE (root span of each trace).
	// Filters apply to the spans being aggregated, but categorization comes from root spans.
	query := fmt.Sprintf(`
		WITH root_categories AS (
			-- Find root span of each trace and categorize by INITIATING SOURCE
			-- Priority: ailang.source attribute (explicit) > service.name > span name patterns
			-- This ensures coordinator-spawned Claude Code sessions are attributed to "Coordinator Tasks"
			SELECT
				trace_id,
				CASE
					-- 1. Check ailang.source FIRST (explicit source from AILANG executor)
					-- Critical for proper cost attribution: GitHub → Coordinator → Claude Code
					WHEN json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' THEN 'coordinator'
					WHEN json_extract(resource_attributes, '$."ailang.source"') = 'eval' THEN 'eval'
					-- 2. Then check service.name (generic tool identity)
					WHEN json_extract(resource_attributes, '$."service.name"') = 'claude-code' THEN 'user_session'
					WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' THEN 'eval'
					WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-coordinator' THEN 'coordinator'
					-- 3. Check specific span name patterns
					WHEN name LIKE 'eval.%%' THEN 'eval'
					WHEN name LIKE 'coordinator.%%' OR name LIKE 'claude.execute%%' OR name LIKE 'exec.%%' THEN 'coordinator'
					WHEN name LIKE 'messages.%%' THEN 'messaging'
					WHEN name LIKE 'ailang-%%' THEN 'server'
					WHEN name LIKE 'ailang.exec%%' THEN 'coordinator'
					WHEN name LIKE 'ailang.%%' OR name LIKE 'ailang %%' OR name LIKE 'compile%%' OR name LIKE 'check.%%' THEN 'cli'
					WHEN name LIKE 'claude_code.%%' THEN 'user_session'
					-- 4. API calls without service.name are truly direct API usage
					WHEN name LIKE 'anthropic.%%' OR name LIKE 'gemini.%%' OR name LIKE 'openai.%%' THEN 'direct_api'
					WHEN name IN ('api_request', 'api_error', 'call_llm', 'invocation') THEN 'direct_api'
					ELSE 'other'
				END as source_id,
				CASE
					-- Same priority: ailang.source > service.name > span name
					WHEN json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' THEN 'Coordinator Tasks'
					WHEN json_extract(resource_attributes, '$."ailang.source"') = 'eval' THEN 'Eval Benchmarks'
					WHEN json_extract(resource_attributes, '$."service.name"') = 'claude-code' THEN 'User Sessions'
					WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' THEN 'Eval Benchmarks'
					WHEN json_extract(resource_attributes, '$."service.name"') = 'ailang-coordinator' THEN 'Coordinator Tasks'
					WHEN name LIKE 'eval.%%' THEN 'Eval Benchmarks'
					WHEN name LIKE 'coordinator.%%' OR name LIKE 'claude.execute%%' OR name LIKE 'exec.%%' THEN 'Coordinator Tasks'
					WHEN name LIKE 'messages.%%' THEN 'Messaging'
					WHEN name LIKE 'ailang-%%' THEN 'Server'
					WHEN name LIKE 'ailang.exec%%' THEN 'Coordinator Tasks'
					WHEN name LIKE 'ailang.%%' OR name LIKE 'ailang %%' OR name LIKE 'compile%%' OR name LIKE 'check.%%' THEN 'CLI Usage'
					WHEN name LIKE 'claude_code.%%' THEN 'User Sessions'
					WHEN name LIKE 'anthropic.%%' OR name LIKE 'gemini.%%' OR name LIKE 'openai.%%' THEN 'Direct API'
					WHEN name IN ('api_request', 'api_error', 'call_llm', 'invocation') THEN 'Direct API'
					ELSE 'Other'
				END as source_label
			FROM spans
			WHERE parent_span_id IS NULL OR parent_span_id = ''
		)
		SELECT
			COALESCE(r.source_id, 'other') as id,
			COALESCE(r.source_label, 'Other') as label,
			COUNT(*) as span_count,
			COALESCE(SUM(s.tokens_in), 0) as tokens_in,
			COALESCE(SUM(s.tokens_out), 0) as tokens_out,
			COALESCE(SUM(s.cost_usd), 0) as cost_usd,
			COALESCE(SUM(s.duration_ms), 0) as duration_ms,
			COALESCE(SUM(s.cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(s.cache_creation_tokens), 0) as cache_creation_tokens
		FROM spans s
		LEFT JOIN root_categories r ON s.trace_id = r.trace_id
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
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD, &item.DurationMs, &item.CacheReadTokens, &item.CacheCreationTokens); err != nil {
			return nil, err
		}
		// Calculate cache savings
		if item.CacheReadTokens > 0 {
			item.CacheSavingsUSD = CalculateCacheSavings("", item.CacheReadTokens)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetFilteredBreakdownByModel returns model breakdown with filters applied
func (b *SQLiteBackend) GetFilteredBreakdownByModel(ctx context.Context, filter *ControlPlaneFilter, wsConfig WorkspaceMapping) ([]BreakdownItem, error) {
	if filter == nil || filter.IsEmpty() {
		return b.GetBreakdownByModel(ctx)
	}

	// Build conditions using shared helper (includes time range)
	// Note: For model breakdown, we exclude model from filter conditions
	// since the query groups BY model
	tempFilter := &ControlPlaneFilter{
		SourceType: filter.SourceType,
		Provider:   filter.Provider,
		Workspace:  filter.Workspace,
		StartDate:  filter.StartDate,
		EndDate:    filter.EndDate,
	}
	conditions, args := buildFilterConditions(tempFilter, wsConfig)
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
			COALESCE(SUM(cost_usd), 0) as cost_usd,
			COALESCE(SUM(duration_ms), 0) as duration_ms,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens
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
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TokensIn, &item.TokensOut, &item.CostUSD, &item.DurationMs, &item.CacheReadTokens, &item.CacheCreationTokens); err != nil {
			return nil, err
		}
		// Calculate cache savings
		if item.CacheReadTokens > 0 {
			item.CacheSavingsUSD = CalculateCacheSavings("", item.CacheReadTokens)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetFilteredBreakdownByWorkspace returns workspace breakdown with filters applied.
// Uses a basic fallback mapping - for full config support use GetFilteredBreakdownByWorkspaceWithMapping.
func (b *SQLiteBackend) GetFilteredBreakdownByWorkspace(ctx context.Context, filter *ControlPlaneFilter, wsConfig WorkspaceMapping) ([]BreakdownItem, error) {
	// Use fallback mapping when no config provided
	return b.GetFilteredBreakdownByWorkspaceWithMapping(ctx, filter, nil, wsConfig)
}

// GetFilteredBreakdownByWorkspaceWithMapping returns workspace breakdown with config-driven path mapping.
// The mapping parameter converts file paths to Firestore workspace IDs.
// Note: Workspace filter is deliberately excluded from conditions since this query groups BY workspace.
func (b *SQLiteBackend) GetFilteredBreakdownByWorkspaceWithMapping(ctx context.Context, filter *ControlPlaneFilter, mapping WorkspaceMapping, wsConfig WorkspaceMapping) ([]BreakdownItem, error) {
	// Build conditions using shared helper
	// Note: For workspace breakdown, we exclude workspace from filter conditions
	// since the query groups BY workspace
	var conditions []string
	var args []interface{}
	if filter != nil && !filter.IsEmpty() {
		tempFilter := &ControlPlaneFilter{
			SourceType: filter.SourceType,
			Provider:   filter.Provider,
			Model:      filter.Model,
			StartDate:  filter.StartDate,
			EndDate:    filter.EndDate,
			// NOTE: Workspace intentionally excluded - this query groups BY workspace
		}
		conditions, args = buildFilterConditions(tempFilter, wsConfig)
	}

	// Build WHERE clause (empty if no conditions)
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			whereClause += " AND " + c
		}
	}

	// Build workspace mapping SQL - use config if provided, else use cwd directly
	var workspaceIDMapping string
	if mapping != nil {
		workspaceIDMapping = mapping.BuildWorkspaceMappingSQL("cwd")
	} else {
		// Fallback: use default mapping patterns (hardcoded for backwards compatibility)
		workspaceIDMapping = `CASE
			WHEN cwd = 'unknown' THEN 'unknown'
			WHEN cwd LIKE '%/.eval_workspace/%' THEN 'eval_workspace'
			WHEN cwd LIKE '%/worktrees/%' THEN 'coordinator_worktrees'
			WHEN cwd LIKE '%/sunholo/ailang/ui' THEN 'sunholo-data/ailang'
			WHEN cwd LIKE '%/sunholo/ailang' THEN 'sunholo-data/ailang'
			WHEN cwd LIKE '%/stapledon%' THEN 'sunholo-data/stapledons_voyage'
			WHEN cwd LIKE '%/twilight%' THEN 'MarkEdmondson1234/TwilightGame'
			ELSE cwd
		END`
	}

	// Use config-driven workspace mapping
	query := fmt.Sprintf(`
		WITH workspace_data AS (
			SELECT
				COALESCE(json_extract(resource_attributes, '$."process.cwd"'), 'unknown') as cwd,
				tokens_in,
				tokens_out,
				cost_usd,
				duration_ms,
				cache_read_tokens,
				cache_creation_tokens,
				id
			FROM spans
			%s
		),
		-- Map file paths to Firestore workspace IDs using config-driven patterns
		mapped AS (
			SELECT
				%s as workspace_id,
				tokens_in,
				tokens_out,
				cost_usd,
				duration_ms,
				cache_read_tokens,
				cache_creation_tokens
			FROM workspace_data
		)
		SELECT
			workspace_id as id,
			workspace_id as label,
			COUNT(*) as span_count,
			0 as task_count,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out,
			COALESCE(SUM(cost_usd), 0) as cost_usd,
			COALESCE(SUM(duration_ms), 0) as duration_ms,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens
		FROM mapped
		GROUP BY workspace_id
		ORDER BY cost_usd DESC
	`, whereClause, workspaceIDMapping)

	rows, err := b.store.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BreakdownItem
	for rows.Next() {
		var item BreakdownItem
		if err := rows.Scan(&item.ID, &item.Label, &item.SpanCount, &item.TaskCount, &item.TokensIn, &item.TokensOut, &item.CostUSD, &item.DurationMs, &item.CacheReadTokens, &item.CacheCreationTokens); err != nil {
			return nil, err
		}
		// Apply workspace labels from mapping if provided
		if mapping != nil {
			item.Label = mapping.GetWorkspaceLabel(item.ID)
		} else {
			// Fallback labels
			item.Label = defaultWorkspaceLabel(item.ID)
		}
		// Calculate cache savings
		if item.CacheReadTokens > 0 {
			item.CacheSavingsUSD = CalculateCacheSavings("", item.CacheReadTokens)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// defaultWorkspaceLabel returns a human-friendly label for a workspace ID.
// For internal workspaces, returns predefined labels.
// For user workspaces (org/repo format), returns a formatted label.
// For raw paths, derives the label from path segments.
func defaultWorkspaceLabel(workspaceID string) string {
	// Internal workspace labels
	internalLabels := map[string]string{
		"eval_workspace":        "Eval Benchmarks",
		"coordinator_worktrees": "Coordinator Tasks",
		"unknown":               "No Workspace",
	}
	if label, ok := internalLabels[workspaceID]; ok {
		return label
	}

	// Check if this looks like a raw file path (starts with / or has more than 2 slashes)
	if strings.HasPrefix(workspaceID, "/") || strings.Count(workspaceID, "/") > 1 {
		// Derive workspace from path by taking last two meaningful segments
		derived := deriveWorkspaceFromPath(workspaceID)
		if parts := strings.Split(derived, "/"); len(parts) == 2 {
			return formatLabel(parts[1])
		}
		return formatLabel(derived)
	}

	// For org/repo format, make the repo name the label
	if parts := strings.Split(workspaceID, "/"); len(parts) == 2 {
		return formatLabel(parts[1])
	}

	// For single-segment workspace, format it nicely
	return formatLabel(workspaceID)
}

// deriveWorkspaceFromPath extracts a workspace ID from a file path.
// Uses the last two meaningful path segments (parent/basename).
func deriveWorkspaceFromPath(path string) string {
	if path == "" || path == "unknown" {
		return "unknown"
	}

	// Split path into components
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))

	// Collect meaningful path segments (skip empty, hidden, and common dirs)
	var meaningful []string
	for _, p := range parts {
		if p == "" || p == "tmp" || p == "var" || p == "folders" || strings.HasPrefix(p, ".") {
			continue
		}
		// Also skip common home/user path segments
		if p == "Users" || p == "home" {
			continue
		}
		meaningful = append(meaningful, p)
	}

	if len(meaningful) == 0 {
		return "unknown"
	}

	// Use last two segments as org/repo (or just last one if only one exists)
	if len(meaningful) >= 2 {
		return meaningful[len(meaningful)-2] + "/" + meaningful[len(meaningful)-1]
	}
	return meaningful[len(meaningful)-1]
}

// formatLabel converts a workspace name to a human-readable label.
// Handles camel case: "TwilightGame" -> "Twilight Game"
func formatLabel(name string) string {
	if name == "" {
		return "Unknown"
	}

	// If all uppercase, keep it (like ROCKGAP, ROCKGPT)
	if strings.ToUpper(name) == name && len(name) > 1 {
		return name
	}

	// Insert spaces before uppercase letters (camel case -> spaces)
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if previous char was lowercase (camel case boundary)
			prev := rune(name[i-1])
			if prev >= 'a' && prev <= 'z' {
				result.WriteRune(' ')
			}
		}
		result.WriteRune(r)
	}
	name = result.String()

	// Replace underscores/dashes with spaces
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	// Title case each word
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

// ===== Claude Code Event Types =====

// ClaudeCodeEvent represents a Claude Code API request span formatted as an event
// for the Event Queue. Each api_request span becomes an event that can be clicked
// to show its hierarchy (tool calls correlated by timestamp).
type ClaudeCodeEvent struct {
	ID             string  `json:"id"`           // span_id (also used as task_id for hierarchy lookup)
	CreatedAt      string  `json:"created_at"`   // ISO8601 timestamp (matches inbox message format)
	Type           string  `json:"message_type"` // "claude_code_turn"
	FromAgent      string  `json:"from_agent"`   // Agent ID (e.g., "design-doc-creator") or "claude-code" for user sessions
	ToInbox        string  `json:"to_inbox"`     // Agent inbox (e.g., "design-doc-creator") or "user" for user sessions
	Title          string  `json:"title"`        // "Claude Code Turn ($X.XX)"
	TaskID         string  `json:"task_id"`      // Same as ID for hierarchy lookup
	Status         string  `json:"status"`       // "read" (not actionable like inbox messages)
	CostUSD        float64 `json:"cost_usd"`
	TokensIn       int64   `json:"tokens_in"`
	TokensOut      int64   `json:"tokens_out"`
	DurationMs     int     `json:"duration_ms"`
	Workspace      string  `json:"workspace,omitempty"`       // Working directory (from resource attributes)
	Model          string  `json:"model,omitempty"`           // Model used for this event (e.g., "claude-sonnet-4-5")
	Provider       string  `json:"provider,omitempty"`        // AI provider (e.g., "claude", "gemini")
	Directive      string  `json:"directive,omitempty"`       // Initial user prompt (truncated preview)
	DirectiveFull  string  `json:"directive_full,omitempty"`  // Full directive (for detail views)
	TurnCount      int     `json:"turn_count"`                // Number of turns in session
	MetricsSummary string  `json:"metrics_summary,omitempty"` // "3 turns • $0.42 • 12.5s"
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
			SUM(COALESCE(tokens_out, 0)) as total_tokens_out,
			MAX(model) as model,
			MAX(provider) as provider,
			MAX(json_extract(resource_attributes, '$."process.cwd"')) as workspace,
			COALESCE(
				MAX(json_extract(attributes, '$."task.directive"')),
				MAX(json_extract(attributes, '$.prompt')),
				MAX(json_extract(attributes, '$."user.prompt"')),
				MAX(json_extract(attributes, '$."benchmark.name"')),
				MAX(json_extract(attributes, '$."eval.benchmark"'))
			) as directive
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
		var model, provider, workspace, directive sql.NullString

		if err := rows.Scan(&sessionID, &firstSpanID, &latestTimeRaw, &turnCount, &totalDurationMs, &totalCostUSD, &totalTokensIn, &totalTokensOut, &model, &provider, &workspace, &directive); err != nil {
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

		// Build metrics summary for display as badges
		metricsSummary := fmt.Sprintf("%d turns • %s • %s", turnCount, costStr, durationStr)

		// Set default provider if not in DB
		providerVal := "claude"
		if provider.Valid && provider.String != "" {
			providerVal = provider.String
		}

		// Truncate directive for preview (200 chars), keep full for detail view
		directivePreview := directive.String
		directiveFull := directive.String
		if len(directivePreview) > 200 {
			directivePreview = directivePreview[:200] + "..."
		}

		// Title = directive preview (for scanability), fallback to generic description
		title := directivePreview
		if title == "" {
			title = "Claude Code Session"
		}

		events = append(events, ClaudeCodeEvent{
			ID:             sessionID,  // Use session_id as the event ID
			CreatedAt:      latestTime, // Most recent activity
			Type:           "claude_code_session",
			FromAgent:      "claude-code",
			ToInbox:        "user",
			Title:          title,
			TaskID:         sessionID, // Use session_id for hierarchy lookup (aggregates all turns)
			Status:         "read",
			CostUSD:        totalCostUSD,
			TokensIn:       totalTokensIn,
			TokensOut:      totalTokensOut,
			DurationMs:     totalDurationMs,
			Model:          model.String,
			Provider:       providerVal,
			Workspace:      workspace.String,
			Directive:      directivePreview,
			DirectiveFull:  directiveFull,
			TurnCount:      turnCount,
			MetricsSummary: metricsSummary,
		})
	}

	return events, rows.Err()
}

// GetClaudeCodeEventsWithLookup returns Claude Code sessions with agent info resolved.
// For sessions spawned by coordinator tasks, FromAgent and ToInbox are set from agent_assignments.
// For user-initiated sessions (no coordinator task), defaults to "claude-code" / "user".
// Note: The lookup parameter is kept for API compatibility but is no longer used - agent info
// comes directly from observatory.db via JOIN with agent_assignments.
// The sourceType parameter filters by source:
//   - "coordinator": Only sessions with agent_assignment (coordinator-spawned)
//   - "user_session": Only sessions without agent_assignment (user-initiated)
//   - "": No filter (all sessions)
//
// The workspaceFilter parameter filters by workspace path:
//   - Full path: "/Users/mark/dev/TwilightGame" (exact or contains match)
//   - "unknown" or "No Workspace": Sessions with empty workspace
//   - "coordinator_worktrees": Sessions in worktree directories
//   - "": No filter (all workspaces)
func (b *SQLiteBackend) GetClaudeCodeEventsWithLookup(ctx context.Context, limit int, lookup TaskAgentLookup, sourceType, workspaceFilter string, wsConfig WorkspaceMapping) ([]ClaudeCodeEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	// Build source type filter condition
	var sourceFilter string
	switch sourceType {
	case "coordinator":
		// Only sessions with agent_assignment (coordinator-spawned)
		sourceFilter = " AND aa.agent_id IS NOT NULL"
	case "user_session":
		// Only sessions without agent_assignment (user-initiated)
		sourceFilter = " AND (aa.agent_id IS NULL OR aa.agent_id = '')"
	default:
		sourceFilter = "" // No filter
	}

	// Build workspace filter condition using reverse mapping
	// This converts workspace IDs (org/repo) to path patterns that match process.cwd
	var workspaceFilterSQL string
	var workspaceArgs []interface{}
	switch workspaceFilter {
	case "":
		workspaceFilterSQL = ""
	case "unknown", "No Workspace":
		// Match sessions with empty or null workspace
		workspaceFilterSQL = " AND (json_extract(s.resource_attributes, '$.\"process.cwd\"') IS NULL OR json_extract(s.resource_attributes, '$.\"process.cwd\"') = '')"
	case "coordinator_worktrees", "Coordinator Tasks":
		// Match worktree directories
		workspaceFilterSQL = " AND json_extract(s.resource_attributes, '$.\"process.cwd\"') LIKE '%/worktrees/%'"
	default:
		// Use reverse mapping to find path patterns that match the workspace ID
		patterns := wsConfig.GetPathPatternsForWorkspace(workspaceFilter)
		if len(patterns) > 0 {
			// Build OR clause for all matching patterns
			var orClauses []string
			for _, p := range patterns {
				orClauses = append(orClauses, `json_extract(s.resource_attributes, '$."process.cwd"') LIKE ?`)
				workspaceArgs = append(workspaceArgs, p)
			}
			workspaceFilterSQL = " AND (" + strings.Join(orClauses, " OR ") + ")"
		} else {
			// Fallback to direct substring match if no mapping found
			workspaceFilterSQL = " AND json_extract(s.resource_attributes, '$.\"process.cwd\"') LIKE ?"
			workspaceArgs = append(workspaceArgs, "%"+workspaceFilter+"%")
		}
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
			MAX(aa.agent_id) as agent_id,
			MAX(s.model) as model,
			MAX(s.provider) as provider,
			MAX(json_extract(s.resource_attributes, '$."process.cwd"')) as workspace,
			COALESCE(
				MAX(json_extract(s.attributes, '$."task.directive"')),
				MAX(json_extract(s.attributes, '$.prompt')),
				MAX(json_extract(s.attributes, '$."user.prompt"')),
				MAX(json_extract(s.attributes, '$."benchmark.name"')),
				MAX(json_extract(s.attributes, '$."eval.benchmark"'))
			) as directive
		FROM spans s
		LEFT JOIN agent_assignments aa ON s.task_id = aa.task_id
		WHERE s.name = 'api_request'
		  AND json_extract(s.resource_attributes, '$."service.name"') = 'claude-code'
		  AND json_extract(s.attributes, '$."session.id"') IS NOT NULL` + sourceFilter + workspaceFilterSQL + `
		GROUP BY json_extract(s.attributes, '$."session.id"')
		ORDER BY latest_time DESC
		LIMIT ?
	`

	// Build query arguments
	var args []interface{}
	args = append(args, workspaceArgs...)
	args = append(args, limit)

	rows, err := b.store.DB().QueryContext(ctx, query, args...)
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
		var coordTaskID, agentID, model, provider, workspace, directive sql.NullString

		if err := rows.Scan(&sessionID, &firstSpanID, &latestTimeRaw, &turnCount, &totalDurationMs, &totalCostUSD, &totalTokensIn, &totalTokensOut, &coordTaskID, &agentID, &model, &provider, &workspace, &directive); err != nil {
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

		// Build metrics summary for display as badges
		metricsSummary := fmt.Sprintf("%d turns • %s • %s", turnCount, costStr, durationStr)

		// Default values for user-initiated sessions
		fromAgent := "claude-code"
		toInbox := "user"

		// If we have agent_id from agent_assignments, use it
		// (By convention, agent id == inbox in agent config)
		if agentID.Valid && agentID.String != "" {
			fromAgent = agentID.String
			toInbox = agentID.String
		}

		// Set default provider if not in DB
		providerVal := "claude"
		if provider.Valid && provider.String != "" {
			providerVal = provider.String
		}

		// Truncate directive for preview (200 chars), keep full for detail view
		directivePreview := directive.String
		directiveFull := directive.String
		if len(directivePreview) > 200 {
			directivePreview = directivePreview[:200] + "..."
		}

		// Title = directive preview (for scanability), fallback to generic description
		title := directivePreview
		if title == "" {
			title = "Claude Code Session"
		}

		events = append(events, ClaudeCodeEvent{
			ID:             sessionID,  // Use session_id as the event ID
			CreatedAt:      latestTime, // Most recent activity
			Type:           "claude_code_session",
			FromAgent:      fromAgent,
			ToInbox:        toInbox,
			Title:          title,
			TaskID:         sessionID, // Use session_id for hierarchy lookup (aggregates all turns)
			Status:         "read",
			CostUSD:        totalCostUSD,
			TokensIn:       totalTokensIn,
			TokensOut:      totalTokensOut,
			DurationMs:     totalDurationMs,
			Model:          model.String,
			Provider:       providerVal,
			Workspace:      workspace.String,
			Directive:      directivePreview,
			DirectiveFull:  directiveFull,
			TurnCount:      turnCount,
			MetricsSummary: metricsSummary,
		})
	}

	return events, rows.Err()
}

// splitPath splits a file path into components (works for both Unix and Windows paths)
//
//nolint:unused // Utility for path-based span filtering
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
	// NOTE: Only sum api_request durations to avoid double-counting.
	// Tool calls happen INSIDE api_request turns - their duration is already
	// part of the turn's duration. Summing all spans would double-count.
	var totalTokensIn, totalTokensOut int64
	var totalCost float64
	var totalDuration int64
	for _, span := range apiRequests {
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
//
//nolint:unused // Scaffolded for session-level analytics
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
//
//nolint:unused // Scaffolded for span detail view
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
//
//nolint:unused // Scaffolded for timestamp correlation feature
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
