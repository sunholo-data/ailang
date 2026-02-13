package observatory

import (
	"context"
	"fmt"
)

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
