package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

// ============================================================================
// Usage Time Series Handler
// ============================================================================

// GET /api/controlplane/usage-timeseries - Get aggregated usage metrics over time
// Returns time-bucketed metrics for bar/column chart visualization.
//
// Query params:
//   - metric: Primary metric (cost, tokens, turns, spans) - default: cost
//   - interval: Time bucket (hour, day, week) - default: day
//   - split_by: Dimension to split by (provider, model, workspace, agent) - optional
//   - since: ISO8601 start time (default: 7 days ago)
//   - until: ISO8601 end time (default: now)
//   - Standard filters: provider, model, workspace, source_type
func (s *Server) handleUsageTimeSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	// Parse parameters
	metric := q.Get("metric")
	if metric == "" {
		metric = "cost"
	}
	interval := q.Get("interval")
	if interval == "" {
		interval = "day"
	}
	splitBy := q.Get("split_by")

	// Parse filters (includes start_date, end_date, workspace, provider)
	filter := parseControlPlaneFilter(r)

	// Determine time range from filter
	now := time.Now()
	until := now.AddDate(0, 0, 1)  // Include today
	since := now.AddDate(0, 0, -7) // Default 7 days

	if filter.StartDate != "" {
		if t, err := time.Parse("2006-01-02", filter.StartDate); err == nil {
			since = t
		}
	}
	if filter.EndDate != "" {
		if t, err := time.Parse("2006-01-02", filter.EndDate); err == nil {
			// Add 1 day to include the full end date
			until = t.AddDate(0, 0, 1)
		}
	}

	// Build CLI command
	cliParts := []string{"ailang", "observatory", "usage"}
	cliParts = append(cliParts, "--metric", metric)
	cliParts = append(cliParts, "--interval", interval)
	if filter.StartDate != "" {
		cliParts = append(cliParts, "--since", filter.StartDate)
	}
	if filter.EndDate != "" {
		cliParts = append(cliParts, "--until", filter.EndDate)
	}
	if filter.Workspace != "" {
		cliParts = append(cliParts, "--workspace", filter.Workspace)
	}
	if filter.Provider != "" {
		cliParts = append(cliParts, "--provider", filter.Provider)
	}
	if splitBy != "" {
		cliParts = append(cliParts, "--split-by", splitBy)
	}
	cliParts = append(cliParts, "--format", "json")
	cliCommand := strings.Join(cliParts, " ")

	response := UsageTimeSeriesResponse{
		Points:     []UsageTimeSeriesPoint{},
		Interval:   interval,
		Metric:     metric,
		SplitBy:    splitBy,
		CliCommand: cliCommand,
	}

	// Get data from Observatory
	if s.obsBackend != nil {
		if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok {
			points, totalCost, err := getUsageTimeSeriesData(ctx, sqliteBackend, filter, since, until, interval, splitBy)
			if err != nil {
				log.Printf("Failed to get usage time series data: %v", err)
			} else {
				response.Points = points
				response.TotalCost = totalCost
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode usage time series response: %v", err)
	}
}

// getUsageTimeSeriesData queries the observatory for time-bucketed usage data
func getUsageTimeSeriesData(ctx context.Context, backend *observatory.SQLiteBackend, filter *observatory.ControlPlaneFilter, since, until time.Time, interval, splitBy string) ([]UsageTimeSeriesPoint, float64, error) {
	db := backend.DB()
	if db == nil {
		return nil, 0, fmt.Errorf("database not available")
	}

	// Determine SQL date format based on interval
	var dateFmt string
	switch interval {
	case "hour":
		dateFmt = "%Y-%m-%dT%H:00:00Z"
	case "week":
		dateFmt = "%Y-%W" // Year-Week format
	default: // day
		dateFmt = "%Y-%m-%d"
	}

	// Build filter conditions
	conditions := []string{
		"start_time >= ?",
		"start_time < ?",
	}
	args := []interface{}{
		since.Format("2006-01-02 15:04:05"),
		until.Format("2006-01-02 15:04:05"),
	}

	if filter.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, filter.Provider)
	}
	if filter.Model != "" {
		conditions = append(conditions, "model = ?")
		args = append(args, filter.Model)
	}
	if filter.SourceType != "" {
		conditions = append(conditions, buildSourceTypeCondition(filter.SourceType))
	}
	if filter.Workspace != "" {
		conditions = append(conditions, "json_extract(resource_attributes, '$.\"process.cwd\"') = ?")
		args = append(args, filter.Workspace)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Build query with optional split
	var query string
	if splitBy != "" {
		// With dimension split
		var splitCol string
		switch splitBy {
		case "provider":
			splitCol = "COALESCE(provider, 'unknown')"
		case "model":
			splitCol = "COALESCE(model, 'unknown')"
		case "workspace":
			splitCol = "COALESCE(json_extract(resource_attributes, '$.\"process.cwd\"'), 'unknown')"
		case "agent":
			splitCol = "COALESCE(json_extract(attributes, '$.\"agent.id\"'), 'unknown')"
		default:
			splitCol = "COALESCE(provider, 'unknown')"
		}

		query = fmt.Sprintf(`
			SELECT
				strftime('%s', start_time) as bucket,
				%s as dimension,
				SUM(COALESCE(cost_usd, 0)) as cost,
				SUM(COALESCE(tokens_in, 0)) as tokens_in,
				SUM(COALESCE(tokens_out, 0)) as tokens_out,
				COUNT(CASE WHEN name = 'api_request' THEN 1 END) as turns,
				COUNT(*) as spans,
				COUNT(DISTINCT task_id) as task_count,
				SUM(COALESCE(duration_ms, 0)) as duration_ms
			FROM spans
			WHERE %s
			GROUP BY bucket, dimension
			ORDER BY bucket ASC
		`, dateFmt, splitCol, whereClause)
	} else {
		// Without split
		query = fmt.Sprintf(`
			SELECT
				strftime('%s', start_time) as bucket,
				SUM(COALESCE(cost_usd, 0)) as cost,
				SUM(COALESCE(tokens_in, 0)) as tokens_in,
				SUM(COALESCE(tokens_out, 0)) as tokens_out,
				COUNT(CASE WHEN name = 'api_request' THEN 1 END) as turns,
				COUNT(*) as spans,
				COUNT(DISTINCT task_id) as task_count,
				SUM(COALESCE(duration_ms, 0)) as duration_ms
			FROM spans
			WHERE %s
			GROUP BY bucket
			ORDER BY bucket ASC
		`, dateFmt, whereClause)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Process results
	bucketMap := make(map[string]*UsageTimeSeriesPoint)
	var totalCost float64

	for rows.Next() {
		var bucket string
		var cost float64
		var tokensIn, tokensOut int64
		var turns, spans, taskCount int
		var durationMs int64

		if splitBy != "" {
			var dimension string
			if err := rows.Scan(&bucket, &dimension, &cost, &tokensIn, &tokensOut, &turns, &spans, &taskCount, &durationMs); err != nil {
				return nil, 0, fmt.Errorf("scan failed: %w", err)
			}

			// Normalize workspace paths to aggregate evals and tasks
			if splitBy == "workspace" {
				dimension = normalizeWorkspacePath(dimension)
			}

			// Initialize bucket if needed
			if bucketMap[bucket] == nil {
				bucketMap[bucket] = &UsageTimeSeriesPoint{
					Bucket:      bucket,
					ByDimension: make(map[string]float64),
				}
			}
			point := bucketMap[bucket]
			point.Cost += cost
			point.TokensIn += tokensIn
			point.TokensOut += tokensOut
			point.Tokens = point.TokensIn + point.TokensOut
			point.Turns += turns
			point.Spans += spans
			point.TaskCount += taskCount
			point.DurationMs += durationMs
			// Aggregate by normalized dimension (sum if same dimension appears multiple times)
			point.ByDimension[dimension] += cost
		} else {
			if err := rows.Scan(&bucket, &cost, &tokensIn, &tokensOut, &turns, &spans, &taskCount, &durationMs); err != nil {
				return nil, 0, fmt.Errorf("scan failed: %w", err)
			}

			bucketMap[bucket] = &UsageTimeSeriesPoint{
				Bucket:     bucket,
				Cost:       cost,
				TokensIn:   tokensIn,
				TokensOut:  tokensOut,
				Tokens:     tokensIn + tokensOut,
				Turns:      turns,
				Spans:      spans,
				TaskCount:  taskCount,
				DurationMs: durationMs,
			}
		}
		totalCost += cost
	}

	// Convert map to sorted slice
	var points []UsageTimeSeriesPoint
	for _, point := range bucketMap {
		points = append(points, *point)
	}
	sort.Slice(points, func(i, j int) bool {
		return points[i].Bucket < points[j].Bucket
	})

	// Calculate bucket end times
	for i := range points {
		if i < len(points)-1 {
			points[i].BucketEnd = points[i+1].Bucket
		} else {
			// Last bucket ends at 'until'
			points[i].BucketEnd = until.Format(time.RFC3339)
		}
	}

	return points, totalCost, nil
}

// buildSourceTypeCondition wraps the observatory version for use in this package
func buildSourceTypeCondition(sourceType string) string {
	switch sourceType {
	case "user_session", "claude_code":
		return `(json_extract(resource_attributes, '$."service.name"') = 'claude-code')`
	case "eval":
		return `(json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' OR name LIKE 'eval.%')`
	case "coordinator":
		return `(json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' OR name LIKE 'coordinator.%')`
	case "cli":
		return `(name LIKE 'ailang.%' OR name LIKE 'compile%')`
	case "direct_api":
		return `(name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%')`
	default:
		return "1=1"
	}
}

// normalizeWorkspacePath converts raw workspace paths to clean, aggregated names.
// - Eval workspaces (.Eval_workspace) -> "Eval"
// - Task worktrees (Worktrees/) -> "Tasks"
// - Regular workspaces -> project name (last meaningful directory)
func normalizeWorkspacePath(path string) string {
	if path == "" || path == "unknown" {
		return "unknown"
	}

	// Check for eval workspace
	if strings.Contains(path, ".Eval_workspace") || strings.Contains(path, ".eval_workspace") {
		return "Eval"
	}

	// Check for coordinator task worktree (case-insensitive)
	lowerPath := strings.ToLower(path)
	if strings.Contains(lowerPath, "/worktrees/") || strings.Contains(lowerPath, "/.ailang/state/worktrees/") {
		return "Tasks"
	}

	// Extract project name from regular workspace path
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if part == "" {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(part, ".") {
			continue
		}
		// Skip common non-project directories
		switch part {
		case "Users", "home", "var", "tmp", "temp", "Worktrees":
			continue
		}
		// Skip numeric-looking temp dirs (timestamps)
		if len(part) > 10 && part[0] >= '0' && part[0] <= '9' {
			continue
		}
		// Found a good project name
		return part
	}

	// Fallback to last segment
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}
