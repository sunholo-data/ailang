package observatory

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

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

	// Build workspace mapping SQL — references t.workspace from trace_summaries
	var workspaceIDMapping string
	if mapping != nil {
		workspaceIDMapping = mapping.BuildWorkspaceMappingSQL("t.workspace")
	} else {
		// Fallback: use default mapping patterns (hardcoded for backwards compatibility)
		workspaceIDMapping = `CASE
			WHEN t.workspace = '' OR t.workspace IS NULL THEN 'unknown'
			WHEN t.workspace LIKE '%/.eval_workspace/%' THEN 'eval_workspace'
			WHEN t.workspace LIKE '%/worktrees/%' THEN 'coordinator_worktrees'
			WHEN t.workspace LIKE '%/sunholo/ailang/ui' THEN 'sunholo-data/ailang'
			WHEN t.workspace LIKE '%/sunholo/ailang' THEN 'sunholo-data/ailang'
			WHEN t.workspace LIKE '%/stapledon%' THEN 'sunholo-data/stapledons_voyage'
			WHEN t.workspace LIKE '%/twilight%' THEN 'MarkEdmondson1234/TwilightGame'
			ELSE t.workspace
		END`
	}

	// Build additional filter conditions as AND clauses for the main query
	var filterClause string
	if len(conditions) > 0 {
		for _, c := range conditions {
			filterClause += " AND " + c
		}
	}

	// Uses trace_summaries.workspace (pre-extracted, no json_extract at query time).
	// M-PERF-OBSERVATORY: Two-phase aggregation — first aggregate metrics per trace_id
	// (reduces 662K spans → ~130K trace groups), then join the smaller result with
	// trace_summaries for workspace mapping. This avoids a 662K×213K JOIN.
	query := fmt.Sprintf(`
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
			%s
			GROUP BY trace_id
		)
		SELECT
			%s as id,
			%s as label,
			SUM(tm.span_count) as span_count,
			0 as task_count,
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
		LIMIT 30
	`, filterClause, workspaceIDMapping, workspaceIDMapping)

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
