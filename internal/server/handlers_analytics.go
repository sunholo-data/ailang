package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// ============================================================================
// Task Evolution Handler
// ============================================================================

// GET /api/controlplane/task-evolution - Get task metrics evolution over time
// Returns cumulative metrics (cost, tokens, turns, spans) for each task over time.
// Each task starts at x=0 for easy comparison of growth rates.
//
// Query params:
//   - metric: Primary metric (cost, tokens, turns, spans) - affects CLI hint
//   - interval: Data point interval (turn, minute, 5min) - default: turn
//   - limit: Max tasks to return (default: 10, max: 50)
//   - task_ids: Comma-separated task IDs to fetch (overrides limit)
//   - Standard filters: provider, model, workspace, start_date, end_date
func (s *Server) handleTaskEvolution(w http.ResponseWriter, r *http.Request) {
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
		interval = "turn"
	}
	limit := 10
	if limitParam := q.Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	// Parse specific task IDs if provided
	var taskIDs []string
	if taskIDsParam := q.Get("task_ids"); taskIDsParam != "" {
		taskIDs = strings.Split(taskIDsParam, ",")
		for i := range taskIDs {
			taskIDs[i] = strings.TrimSpace(taskIDs[i])
		}
	}

	// Parse filters
	filter := parseControlPlaneFilter(r)

	// Build CLI command for response
	cliParts := []string{"ailang", "observatory", "evolution"}
	cliParts = append(cliParts, "--metric", metric)
	if len(taskIDs) > 0 {
		cliParts = append(cliParts, "--task-ids", strings.Join(taskIDs, ","))
	} else {
		cliParts = append(cliParts, "--limit", strconv.Itoa(limit))
	}
	if filter.Provider != "" {
		cliParts = append(cliParts, "--provider", filter.Provider)
	}
	if filter.Model != "" {
		cliParts = append(cliParts, "--model", filter.Model)
	}
	if filter.SourceType != "" {
		cliParts = append(cliParts, "--source", filter.SourceType)
	}
	if filter.Workspace != "" {
		cliParts = append(cliParts, "--workspace", filter.Workspace)
	}
	if filter.StartDate != "" {
		cliParts = append(cliParts, "--since", filter.StartDate)
	}
	if filter.EndDate != "" {
		cliParts = append(cliParts, "--until", filter.EndDate)
	}
	cliParts = append(cliParts, "--format", "json")
	cliCommand := strings.Join(cliParts, " ")

	response := TaskEvolutionResponse{
		Tasks:      []TaskEvolutionTask{},
		CliCommand: cliCommand,
	}

	// Get data from Observatory
	if s.obsBackend != nil {
		if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok {
			tasks, err := getTaskEvolutionData(ctx, sqliteBackend, filter, taskIDs, limit, interval)
			if err != nil {
				log.Printf("Failed to get task evolution data: %v", err)
			} else {
				response.Tasks = tasks
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode task evolution response: %v", err)
	}
}

// getTaskEvolutionData queries the observatory for task evolution data
func getTaskEvolutionData(ctx context.Context, backend *observatory.SQLiteBackend, filter *observatory.ControlPlaneFilter, taskIDs []string, limit int, interval string) ([]TaskEvolutionTask, error) {
	db := backend.DB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// First, get the tasks to analyze (either specific IDs or recent tasks)
	var tasks []TaskEvolutionTask

	if len(taskIDs) > 0 {
		// Fetch specific tasks
		for _, taskID := range taskIDs {
			task, err := getTaskEvolutionForTask(ctx, db, taskID, interval)
			if err != nil {
				log.Printf("Failed to get evolution for task %s: %v", taskID, err)
				continue
			}
			if task != nil {
				tasks = append(tasks, *task)
			}
		}
	} else {
		// Get recent tasks based on filter
		recentTasks, err := getRecentTasksForEvolution(ctx, db, filter, limit)
		if err != nil {
			return nil, err
		}
		for _, taskID := range recentTasks {
			task, err := getTaskEvolutionForTask(ctx, db, taskID, interval)
			if err != nil {
				log.Printf("Failed to get evolution for task %s: %v", taskID, err)
				continue
			}
			if task != nil {
				tasks = append(tasks, *task)
			}
		}
	}

	return tasks, nil
}

// getRecentTasksForEvolution returns recent task IDs matching the filter
func getRecentTasksForEvolution(ctx context.Context, db *sql.DB, filter *observatory.ControlPlaneFilter, limit int) ([]string, error) {
	// Build filter conditions
	conditions := []string{"task_id IS NOT NULL AND task_id != '' AND task_id != 'root'"}
	args := []interface{}{}

	if filter.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, filter.Provider)
	}
	if filter.Model != "" {
		conditions = append(conditions, "model = ?")
		args = append(args, filter.Model)
	}
	if filter.StartDate != "" {
		conditions = append(conditions, "date(start_time) >= ?")
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		conditions = append(conditions, "date(start_time) <= ?")
		args = append(args, filter.EndDate)
	}
	if filter.SourceType != "" {
		conditions = append(conditions, buildSourceTypeCondition(filter.SourceType))
	}
	if filter.Workspace != "" {
		conditions = append(conditions, `json_extract(resource_attributes, '$."process.cwd"') LIKE ?`)
		args = append(args, "%"+filter.Workspace+"%")
	}

	query := fmt.Sprintf(`
		SELECT task_id, MAX(start_time) as latest
		FROM spans
		WHERE %s
		GROUP BY task_id
		ORDER BY latest DESC
		LIMIT ?
	`, strings.Join(conditions, " AND "))
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var taskIDList []string
	for rows.Next() {
		var taskID string
		var latest string
		if err := rows.Scan(&taskID, &latest); err != nil {
			continue
		}
		taskIDList = append(taskIDList, taskID)
	}

	return taskIDList, rows.Err()
}

// getTaskEvolutionForTask gets evolution points for a single task
func getTaskEvolutionForTask(ctx context.Context, db *sql.DB, taskID string, interval string) (*TaskEvolutionTask, error) {
	// First, try to get task title from tasks table
	var taskTitle string
	var taskStatus string
	titleQuery := `SELECT COALESCE(title, ''), COALESCE(status, '') FROM tasks WHERE id = ?`
	_ = db.QueryRowContext(ctx, titleQuery, taskID).Scan(&taskTitle, &taskStatus)

	// If no title found, try to derive from span's workspace path
	if taskTitle == "" {
		// For eval tasks, use a clean format
		if strings.HasPrefix(taskID, "eval-") {
			shortID := taskID
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}
			taskTitle = shortID
		} else {
			workspaceQuery := `
				SELECT json_extract(resource_attributes, '$."process.cwd"')
				FROM spans
				WHERE task_id = ? AND resource_attributes IS NOT NULL
				LIMIT 1
			`
			var workspace sql.NullString
			if err := db.QueryRowContext(ctx, workspaceQuery, taskID).Scan(&workspace); err == nil && workspace.Valid && workspace.String != "" {
				// Extract project name from workspace path
				parts := strings.Split(workspace.String, "/")
				projectName := ""
				for i := len(parts) - 1; i >= 0; i-- {
					part := parts[i]
					// Skip temp workspace components
					if part == "" || part == ".eval_workspace" || strings.HasPrefix(part, ".") {
						continue
					}
					// Skip numeric-looking temp dirs
					if len(part) > 15 && part[0] >= '0' && part[0] <= '9' {
						continue
					}
					projectName = part
					break
				}
				if projectName == "" {
					projectName = "session"
				}
				// Add short task ID suffix to distinguish sessions in same project
				shortID := taskID
				if len(shortID) > 8 {
					shortID = shortID[:8]
				}
				taskTitle = fmt.Sprintf("%s/%s", projectName, shortID)
			} else {
				// Fallback to just short task ID
				shortID := taskID
				if len(shortID) > 8 {
					shortID = shortID[:8]
				}
				taskTitle = fmt.Sprintf("session/%s", shortID)
			}
		}
	}

	// Query spans for this task ordered by time
	query := `
		SELECT
			start_time,
			COALESCE(cost_usd, 0) as cost,
			COALESCE(tokens_in, 0) as tokens_in,
			COALESCE(tokens_out, 0) as tokens_out,
			CASE WHEN name = 'api_request' THEN 1 ELSE 0 END as is_turn,
			COALESCE(provider, '') as provider,
			COALESCE(model, '') as model,
			COALESCE(duration_ms, 0) as duration_ms
		FROM spans
		WHERE task_id = ?
		ORDER BY start_time ASC
	`

	rows, err := db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	task := &TaskEvolutionTask{
		TaskID: taskID,
		Title:  taskTitle,
		Status: taskStatus,
		Points: []TaskEvolutionPoint{},
	}

	var cumulCost float64
	var cumulTokensIn, cumulTokensOut int64
	var cumulTurns, cumulSpans int
	var cumulDurationMs int64
	var prevCost float64
	var prevSpans int
	var prevDurationMs int64
	var firstSpanTime time.Time

	for rows.Next() {
		var startTime string
		var cost float64
		var tokensIn, tokensOut int64
		var isTurn int
		var provider, model string
		var durationMs int64

		if err := rows.Scan(&startTime, &cost, &tokensIn, &tokensOut, &isTurn, &provider, &model, &durationMs); err != nil {
			continue
		}

		// Update cumulative values
		cumulCost += cost
		cumulTokensIn += tokensIn
		cumulTokensOut += tokensOut
		cumulTurns += isTurn
		cumulSpans++
		cumulDurationMs += durationMs

		// Parse current span time for elapsed calculation
		currentTime, _ := time.Parse(time.RFC3339, startTime)
		if currentTime.IsZero() {
			// Try alternate format
			currentTime, _ = time.Parse("2006-01-02 15:04:05", startTime)
		}

		// Set task metadata from first span
		if task.StartTime == "" {
			task.StartTime = startTime
			task.Provider = provider
			task.Model = model
			firstSpanTime = currentTime
		}
		task.EndTime = startTime // Will be last span's time

		// Calculate elapsed time since first span
		var elapsedMs int64
		if !firstSpanTime.IsZero() && !currentTime.IsZero() {
			elapsedMs = currentTime.Sub(firstSpanTime).Milliseconds()
		}

		// Create point
		point := TaskEvolutionPoint{
			X:               len(task.Points),
			Timestamp:       startTime,
			Cost:            cumulCost,
			Tokens:          cumulTokensIn + cumulTokensOut,
			TokensIn:        cumulTokensIn,
			TokensOut:       cumulTokensOut,
			Turns:           cumulTurns,
			Spans:           cumulSpans,
			DeltaCost:       cumulCost - prevCost,
			DeltaSpans:      cumulSpans - prevSpans,
			DurationMs:      cumulDurationMs,
			DeltaDurationMs: cumulDurationMs - prevDurationMs,
			ElapsedMs:       elapsedMs,
		}
		task.Points = append(task.Points, point)

		prevCost = cumulCost
		prevSpans = cumulSpans
		prevDurationMs = cumulDurationMs
	}

	if len(task.Points) == 0 {
		return nil, nil // No data for this task
	}

	return task, rows.Err()
}

// ============================================================================
// Token Distribution Handler
// ============================================================================

// GET /api/controlplane/token-distribution - Get token usage histogram
// Returns distribution of token usage across tasks.
//
// Query params:
//   - Standard filters: provider, model, workspace, start_date, end_date
func (s *Server) handleTokenDistribution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	filter := parseControlPlaneFilter(r)

	// Build CLI command
	cliParts := []string{"ailang", "observatory", "tokens"}
	if filter.Provider != "" {
		cliParts = append(cliParts, "--provider", filter.Provider)
	}
	cliParts = append(cliParts, "--format", "json")
	cliCommand := strings.Join(cliParts, " ")

	// Define buckets
	buckets := []TokenDistributionBucket{
		{Label: "0-1K", Min: 0, Max: 1000},
		{Label: "1K-5K", Min: 1000, Max: 5000},
		{Label: "5K-20K", Min: 5000, Max: 20000},
		{Label: "20K-50K", Min: 20000, Max: 50000},
		{Label: "50K-100K", Min: 50000, Max: 100000},
		{Label: "100K+", Min: 100000, Max: -1}, // -1 = no limit
	}

	response := TokenDistributionResponse{
		Buckets:    buckets,
		CliCommand: cliCommand,
	}

	// Get data from Observatory
	if s.obsBackend != nil {
		if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok {
			populatedBuckets, totalTasks, totalCost, err := getTokenDistributionData(ctx, sqliteBackend, filter, buckets)
			if err != nil {
				log.Printf("Failed to get token distribution data: %v", err)
			} else {
				response.Buckets = populatedBuckets
				response.TotalTasks = totalTasks
				response.TotalCost = totalCost
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode token distribution response: %v", err)
	}
}

// getTokenDistributionData queries the observatory for token distribution data
func getTokenDistributionData(ctx context.Context, backend *observatory.SQLiteBackend, filter *observatory.ControlPlaneFilter, buckets []TokenDistributionBucket) ([]TokenDistributionBucket, int, float64, error) {
	db := backend.DB()
	if db == nil {
		return buckets, 0, 0, fmt.Errorf("database not available")
	}

	// Build filter conditions
	conditions := []string{"task_id IS NOT NULL AND task_id != '' AND task_id != 'root'"}
	args := []interface{}{}

	if filter.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, filter.Provider)
	}
	if filter.Model != "" {
		conditions = append(conditions, "model = ?")
		args = append(args, filter.Model)
	}
	if filter.StartDate != "" {
		conditions = append(conditions, "date(start_time) >= ?")
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		conditions = append(conditions, "date(start_time) <= ?")
		args = append(args, filter.EndDate)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Query to get per-task token totals
	query := fmt.Sprintf(`
		SELECT
			task_id,
			SUM(COALESCE(tokens_in, 0) + COALESCE(tokens_out, 0)) as total_tokens,
			SUM(COALESCE(cost_usd, 0)) as total_cost,
			COUNT(*) as span_count
		FROM spans
		WHERE %s
		GROUP BY task_id
	`, whereClause)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return buckets, 0, 0, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var totalTasks int
	var totalCost float64

	// Process each task and assign to bucket
	for rows.Next() {
		var taskID string
		var totalTokens int64
		var cost float64
		var spanCount int

		if err := rows.Scan(&taskID, &totalTokens, &cost, &spanCount); err != nil {
			continue
		}

		totalTasks++
		totalCost += cost

		// Find appropriate bucket
		for i := range buckets {
			if totalTokens >= buckets[i].Min && (buckets[i].Max == -1 || totalTokens < buckets[i].Max) {
				buckets[i].TaskCount++
				buckets[i].SpanCount += spanCount
				buckets[i].TotalCost += cost
				break
			}
		}
	}

	return buckets, totalTasks, totalCost, nil
}

// ============================================================================
// Outliers Analysis Handler
// ============================================================================

// GET /api/controlplane/outliers - Get statistical outlier analysis for a task
// Detects spans with metrics (cost, duration, tokens) that deviate significantly from the mean.
//
// Query params:
//   - task_id: Task ID to analyze (required)
//   - threshold: Z-score threshold (default: 2.0)
//   - metric: Filter to specific metric: cost, duration, tokens (default: all)
//   - rate: Include rate-of-change analysis (default: false)
//   - limit: Max outliers to return (default: 10)
func (s *Server) handleOutliersAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	// Parse task_id (required)
	taskID := q.Get("task_id")
	if taskID == "" {
		http.Error(w, "task_id parameter is required", http.StatusBadRequest)
		return
	}

	// Parse optional parameters
	threshold := 2.0
	if thresholdParam := q.Get("threshold"); thresholdParam != "" {
		if !strings.ContainsRune(thresholdParam, '_') {
			if t, err := strconv.ParseFloat(thresholdParam, 64); err == nil && t > 0 {
				threshold = t
			}
		}
	}

	metric := q.Get("metric")
	showRate := q.Get("rate") == "true"

	limit := 10
	if limitParam := q.Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Build CLI command
	cliParts := []string{"ailang", "observatory", "outliers"}
	cliParts = append(cliParts, "--task", taskID)
	cliParts = append(cliParts, "--threshold", fmt.Sprintf("%.1f", threshold))
	if metric != "" {
		cliParts = append(cliParts, "--metric", metric)
	}
	if showRate {
		cliParts = append(cliParts, "--show-rate")
	}
	if limit != 10 {
		cliParts = append(cliParts, "--limit", strconv.Itoa(limit))
	}
	cliParts = append(cliParts, "--format", "json")
	cliCommand := strings.Join(cliParts, " ")

	// Perform analysis
	if s.obsBackend == nil {
		http.Error(w, "Observatory backend not available", http.StatusServiceUnavailable)
		return
	}

	opts := observatory.OutlierOptions{
		Threshold: threshold,
		Metric:    metric,
		ShowRate:  showRate,
		Limit:     limit,
	}

	analysis, err := observatory.AnalyzeTaskOutliers(ctx, s.obsBackend, taskID, opts)
	if err != nil {
		// Check if task was not found (sql.ErrNoRows)
		if err == sql.ErrNoRows || strings.Contains(err.Error(), "no rows") {
			log.Printf("Task not found for outliers analysis: %s", taskID)
			http.Error(w, fmt.Sprintf("Task not found: %s", taskID), http.StatusNotFound)
			return
		}
		log.Printf("Failed to analyze outliers for task %s: %v", taskID, err)
		http.Error(w, fmt.Sprintf("Failed to analyze task: %v", err), http.StatusInternalServerError)
		return
	}

	response := OutliersAnalysisResponse{
		TaskID:       analysis.TaskID,
		TaskTitle:    analysis.TaskTitle,
		SpanCount:    analysis.SpanCount,
		Threshold:    analysis.Threshold,
		Stats:        analysis.Stats,
		Outliers:     analysis.Outliers,
		RateOfChange: analysis.RateOfChange,
		CliCommand:   cliCommand,
		AnalyzedAt:   analysis.AnalyzedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode outliers response: %v", err)
	}
}
