package observatory

import (
	"context"
	"fmt"
)

// ===== Breakdown Queries =====

// GetBreakdownByProvider returns cost/token breakdown by provider.
// Scoped to last 30 days for performance on large databases (M-PERF-OBSERVATORY).
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
			AND start_time > datetime('now', '-30 days')
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

// GetBreakdownBySourceType returns cost/token breakdown by source type.
// Uses two-phase aggregation: first aggregate per trace_id, then join with trace_summaries.
// M-PERF-OBSERVATORY: avoids 662K×213K JOIN by reducing to ~130K trace groups first.
func (b *SQLiteBackend) GetBreakdownBySourceType(ctx context.Context) ([]BreakdownItem, error) {
	rows, err := b.store.DB().QueryContext(ctx, `
		WITH trace_metrics AS (
			SELECT
				trace_id,
				COUNT(*) as span_count,
				COALESCE(SUM(tokens_in), 0) as tokens_in,
				COALESCE(SUM(tokens_out), 0) as tokens_out,
				COALESCE(SUM(cost_usd), 0) as cost_usd,
				COALESCE(SUM(duration_ms), 0) as duration_ms,
				COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
				COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens
			FROM spans
			WHERE start_time > datetime('now', '-30 days')
			GROUP BY trace_id
		)
		SELECT
			CASE
				WHEN t.root_span_name LIKE 'eval.%' THEN 'eval'
				WHEN t.root_span_name LIKE 'coordinator.%' OR t.root_span_name LIKE 'claude.execute%' OR t.root_span_name LIKE 'exec.%' OR t.root_span_name LIKE 'ailang.exec%' THEN 'coordinator'
				WHEN t.root_span_name LIKE 'messages.%' THEN 'messaging'
				WHEN t.root_span_name LIKE 'ailang-%' THEN 'server'
				WHEN t.root_span_name LIKE 'ailang.%' OR t.root_span_name LIKE 'ailang %' OR t.root_span_name LIKE 'compile%' OR t.root_span_name LIKE 'check.%' THEN 'cli'
				WHEN t.root_span_name LIKE 'claude_code.%' THEN 'user_session'
				WHEN t.root_span_name LIKE 'anthropic.%' OR t.root_span_name LIKE 'gemini.%' OR t.root_span_name LIKE 'openai.%' THEN 'direct_api'
				WHEN t.root_span_name IN ('api_request', 'api_error', 'call_llm', 'invocation') THEN 'direct_api'
				ELSE 'other'
			END as id,
			CASE
				WHEN t.root_span_name LIKE 'eval.%' THEN 'Eval Benchmarks'
				WHEN t.root_span_name LIKE 'coordinator.%' OR t.root_span_name LIKE 'claude.execute%' OR t.root_span_name LIKE 'exec.%' OR t.root_span_name LIKE 'ailang.exec%' THEN 'Coordinator Tasks'
				WHEN t.root_span_name LIKE 'messages.%' THEN 'Messaging'
				WHEN t.root_span_name LIKE 'ailang-%' THEN 'Server'
				WHEN t.root_span_name LIKE 'ailang.%' OR t.root_span_name LIKE 'ailang %' OR t.root_span_name LIKE 'compile%' OR t.root_span_name LIKE 'check.%' THEN 'CLI Usage'
				WHEN t.root_span_name LIKE 'claude_code.%' THEN 'User Sessions'
				WHEN t.root_span_name LIKE 'anthropic.%' OR t.root_span_name LIKE 'gemini.%' OR t.root_span_name LIKE 'openai.%' THEN 'Direct API'
				WHEN t.root_span_name IN ('api_request', 'api_error', 'call_llm', 'invocation') THEN 'Direct API'
				ELSE 'Other'
			END as label,
			SUM(tm.span_count) as span_count,
			SUM(tm.tokens_in) as tokens_in,
			SUM(tm.tokens_out) as tokens_out,
			SUM(tm.cost_usd) as cost_usd,
			SUM(tm.duration_ms) as duration_ms,
			SUM(tm.cache_read_tokens) as cache_read_tokens,
			SUM(tm.cache_creation_tokens) as cache_creation_tokens
		FROM trace_metrics tm
		INNER JOIN trace_summaries t ON tm.trace_id = t.trace_id
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

// GetBreakdownByModel returns cost/token breakdown by model.
// Scoped to last 30 days for performance on large databases (M-PERF-OBSERVATORY).
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
			AND start_time > datetime('now', '-30 days')
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

// GetFilteredBreakdownBySourceType returns source type breakdown with filters applied.
// Uses trace_summaries.root_span_name for fast categorization (M-PERF-OBSERVATORY).
func (b *SQLiteBackend) GetFilteredBreakdownBySourceType(ctx context.Context, filter *ControlPlaneFilter, wsConfig WorkspaceMapping) ([]BreakdownItem, error) {
	if filter == nil || filter.IsEmpty() {
		return b.GetBreakdownBySourceType(ctx)
	}

	// Build conditions using shared helper (includes time range)
	// Exclude source_type from filter since the query groups BY source type
	tempFilter := &ControlPlaneFilter{
		Provider:  filter.Provider,
		Model:     filter.Model,
		Workspace: filter.Workspace,
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	}
	conditions, args := buildFilterConditions(tempFilter, wsConfig)

	// Always add a 30-day floor if no date range specified
	if filter.StartDate == "" && filter.EndDate == "" {
		conditions = append(conditions, "s.start_time > datetime('now', '-30 days')")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "AND " + conditions[0]
		for _, c := range conditions[1:] {
			whereClause += " AND " + c
		}
	}

	// Uses trace_summaries for categorization — avoids json_extract on 3.9GB resource_attributes
	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN t.root_span_name LIKE 'eval.%%' THEN 'eval'
				WHEN t.root_span_name LIKE 'coordinator.%%' OR t.root_span_name LIKE 'claude.execute%%' OR t.root_span_name LIKE 'exec.%%' OR t.root_span_name LIKE 'ailang.exec%%' THEN 'coordinator'
				WHEN t.root_span_name LIKE 'messages.%%' THEN 'messaging'
				WHEN t.root_span_name LIKE 'ailang-%%' THEN 'server'
				WHEN t.root_span_name LIKE 'ailang.%%' OR t.root_span_name LIKE 'ailang %%' OR t.root_span_name LIKE 'compile%%' OR t.root_span_name LIKE 'check.%%' THEN 'cli'
				WHEN t.root_span_name LIKE 'claude_code.%%' THEN 'user_session'
				WHEN t.root_span_name LIKE 'anthropic.%%' OR t.root_span_name LIKE 'gemini.%%' OR t.root_span_name LIKE 'openai.%%' THEN 'direct_api'
				WHEN t.root_span_name IN ('api_request', 'api_error', 'call_llm', 'invocation') THEN 'direct_api'
				ELSE 'other'
			END as id,
			CASE
				WHEN t.root_span_name LIKE 'eval.%%' THEN 'Eval Benchmarks'
				WHEN t.root_span_name LIKE 'coordinator.%%' OR t.root_span_name LIKE 'claude.execute%%' OR t.root_span_name LIKE 'exec.%%' OR t.root_span_name LIKE 'ailang.exec%%' THEN 'Coordinator Tasks'
				WHEN t.root_span_name LIKE 'messages.%%' THEN 'Messaging'
				WHEN t.root_span_name LIKE 'ailang-%%' THEN 'Server'
				WHEN t.root_span_name LIKE 'ailang.%%' OR t.root_span_name LIKE 'ailang %%' OR t.root_span_name LIKE 'compile%%' OR t.root_span_name LIKE 'check.%%' THEN 'CLI Usage'
				WHEN t.root_span_name LIKE 'claude_code.%%' THEN 'User Sessions'
				WHEN t.root_span_name LIKE 'anthropic.%%' OR t.root_span_name LIKE 'gemini.%%' OR t.root_span_name LIKE 'openai.%%' THEN 'Direct API'
				WHEN t.root_span_name IN ('api_request', 'api_error', 'call_llm', 'invocation') THEN 'Direct API'
				ELSE 'Other'
			END as label,
			COUNT(*) as span_count,
			COALESCE(SUM(s.tokens_in), 0) as tokens_in,
			COALESCE(SUM(s.tokens_out), 0) as tokens_out,
			COALESCE(SUM(s.cost_usd), 0) as cost_usd,
			COALESCE(SUM(s.duration_ms), 0) as duration_ms,
			COALESCE(SUM(s.cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(s.cache_creation_tokens), 0) as cache_creation_tokens
		FROM spans s
		INNER JOIN trace_summaries t ON s.trace_id = t.trace_id
		WHERE 1=1 %s
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
