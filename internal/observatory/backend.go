// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"database/sql"
	"fmt"
)

// Backend defines the interface for observatory storage backends.
// Implementations include SQLite (local), GCP Trace, and Jaeger.
type Backend interface {
	// Workspace operations
	CreateWorkspace(ctx context.Context, w *Workspace) error
	GetWorkspace(ctx context.Context, id string) (*Workspace, error)
	ListWorkspaces(ctx context.Context) ([]*Workspace, error)
	UpdateWorkspace(ctx context.Context, w *Workspace) error
	DeleteWorkspace(ctx context.Context, id string) error
	GetWorkspaceStats(ctx context.Context, id string) (*WorkspaceStats, error)

	// Task operations
	CreateTask(ctx context.Context, t *Task) error
	GetTask(ctx context.Context, id string) (*Task, error)
	ListTasks(ctx context.Context, opts TaskListOptions) ([]*Task, error)
	UpdateTask(ctx context.Context, t *Task) error
	DeleteTask(ctx context.Context, id string) error

	// Agent assignment operations
	CreateAgentAssignment(ctx context.Context, a *AgentAssignment) error
	GetAgentAssignment(ctx context.Context, id string) (*AgentAssignment, error)
	ListAgentAssignments(ctx context.Context, taskID string) ([]*AgentAssignment, error)
	UpdateAgentAssignment(ctx context.Context, a *AgentAssignment) error
	DeleteAgentAssignment(ctx context.Context, id string) error
	GetAgentStats(ctx context.Context, agentID string) (*AgentStats, error)

	// Span operations
	CreateSpan(ctx context.Context, span *Span) error
	GetSpan(ctx context.Context, id string) (*Span, error)
	ListSpans(ctx context.Context, opts SpanListOptions) ([]*Span, error)
	UpdateSpan(ctx context.Context, span *Span) error
	UpdateSpanLinks(ctx context.Context, spanID, taskID, assignmentID string) error
	RecalculateTaskAggregates(ctx context.Context, taskID string) error
	DeleteSpan(ctx context.Context, id string) error
	GetTrace(ctx context.Context, traceID string) (*Trace, error)
	ListTraces(ctx context.Context, opts TraceQuery) ([]*TraceSummary, error)
	// LookupTaskBySessionID finds task hierarchy for Claude Code session correlation
	LookupTaskBySessionID(ctx context.Context, sessionID string) (taskID, assignmentID, traceID string)
	// LinkOrphanedSpansBySession updates spans with matching session.id that lack task linkage
	// Called after storing claude.execute span to retroactively link orphaned Claude Code events
	LinkOrphanedSpansBySession(ctx context.Context, sessionID, taskID, assignmentID string) (int64, error)

	// Span event operations
	CreateSpanEvent(ctx context.Context, e *SpanEvent) error
	GetSpanEvents(ctx context.Context, spanID string) ([]SpanEvent, error)
	DeleteSpanEvent(ctx context.Context, id int64) error

	// Message operations
	CreateMessage(ctx context.Context, m *Message) error
	GetMessage(ctx context.Context, id string) (*Message, error)
	ListMessages(ctx context.Context, opts MessageListOptions) ([]*Message, error)
	UpdateMessage(ctx context.Context, m *Message) error
	DeleteMessage(ctx context.Context, id string) error
	MarkMessageRead(ctx context.Context, id string) error
	MarkMessageArchived(ctx context.Context, id string) error

	// Aggregate operations
	GetMetricsSummary(ctx context.Context) (*MetricsSummary, error)
	GetProviderComparison(ctx context.Context) ([]*ProviderComparison, error)
	GetTaskTimeline(ctx context.Context, taskID string) ([]*TaskTimeline, error)

	// Lifecycle
	Close() error
}

// SQLiteBackend implements Backend using SQLite storage.
type SQLiteBackend struct {
	store *Store
}

// NewSQLiteBackend creates a new SQLite backend.
func NewSQLiteBackend(db *sql.DB) (*SQLiteBackend, error) {
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	return &SQLiteBackend{store: NewStore(db)}, nil
}

// NewSQLiteBackendFromPath creates a SQLite backend from a file path.
func NewSQLiteBackendFromPath(path string) (*SQLiteBackend, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return NewSQLiteBackend(db)
}

// Close closes the database connection.
func (b *SQLiteBackend) Close() error {
	return b.store.DB().Close()
}

// ===== Workspace Operations =====

func (b *SQLiteBackend) CreateWorkspace(ctx context.Context, w *Workspace) error {
	return b.store.CreateWorkspace(w)
}

func (b *SQLiteBackend) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	return b.store.GetWorkspace(id)
}

func (b *SQLiteBackend) ListWorkspaces(ctx context.Context) ([]*Workspace, error) {
	return b.store.ListWorkspaces()
}

func (b *SQLiteBackend) UpdateWorkspace(ctx context.Context, w *Workspace) error {
	return b.store.UpdateWorkspace(w)
}

func (b *SQLiteBackend) DeleteWorkspace(ctx context.Context, id string) error {
	return b.store.DeleteWorkspace(id)
}

func (b *SQLiteBackend) GetWorkspaceStats(ctx context.Context, id string) (*WorkspaceStats, error) {
	return b.store.GetWorkspaceStats(id)
}

// ===== Task Operations =====

func (b *SQLiteBackend) CreateTask(ctx context.Context, t *Task) error {
	return b.store.CreateTask(t)
}

func (b *SQLiteBackend) GetTask(ctx context.Context, id string) (*Task, error) {
	return b.store.GetTask(id)
}

func (b *SQLiteBackend) ListTasks(ctx context.Context, opts TaskListOptions) ([]*Task, error) {
	return b.store.ListTasks(opts)
}

func (b *SQLiteBackend) UpdateTask(ctx context.Context, t *Task) error {
	return b.store.UpdateTask(t)
}

func (b *SQLiteBackend) DeleteTask(ctx context.Context, id string) error {
	return b.store.DeleteTask(id)
}

// ===== Agent Assignment Operations =====

func (b *SQLiteBackend) CreateAgentAssignment(ctx context.Context, a *AgentAssignment) error {
	return b.store.CreateAgentAssignment(a)
}

func (b *SQLiteBackend) GetAgentAssignment(ctx context.Context, id string) (*AgentAssignment, error) {
	return b.store.GetAgentAssignment(id)
}

func (b *SQLiteBackend) ListAgentAssignments(ctx context.Context, taskID string) ([]*AgentAssignment, error) {
	return b.store.ListAgentAssignments(taskID)
}

func (b *SQLiteBackend) UpdateAgentAssignment(ctx context.Context, a *AgentAssignment) error {
	return b.store.UpdateAgentAssignment(a)
}

func (b *SQLiteBackend) DeleteAgentAssignment(ctx context.Context, id string) error {
	return b.store.DeleteAgentAssignment(id)
}

func (b *SQLiteBackend) GetAgentStats(ctx context.Context, agentID string) (*AgentStats, error) {
	return b.store.GetAgentStats(agentID)
}

// ===== Span Operations =====

func (b *SQLiteBackend) CreateSpan(ctx context.Context, span *Span) error {
	// Use transaction for span creation with aggregation updates (M-TASK-HIERARCHY)
	if span.TaskID != "" || span.AgentAssignmentID != "" {
		return b.createSpanWithAggregation(ctx, span)
	}
	// No task/assignment linkage, use simple insert
	return b.store.CreateSpan(span)
}

// createSpanWithAggregation creates a span and updates related aggregates in a transaction.
func (b *SQLiteBackend) createSpanWithAggregation(ctx context.Context, span *Span) error {
	tx, err := b.store.DB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // No-op if committed

	// Insert span
	var parentSpanID, taskID, agentAssignmentID interface{}
	if span.ParentSpanID != "" {
		parentSpanID = span.ParentSpanID
	}
	if span.TaskID != "" {
		taskID = span.TaskID
	}
	if span.AgentAssignmentID != "" {
		agentAssignmentID = span.AgentAssignmentID
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO spans (id, trace_id, parent_span_id, task_id, agent_assignment_id,
		                   name, kind, status, status_message, start_time, end_time,
		                   duration_ms, tokens_in, tokens_out, cost_usd, model, provider,
		                   attributes, resource_attributes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, span.ID, span.TraceID, parentSpanID, taskID, agentAssignmentID,
		span.Name, span.Kind, span.Status, span.StatusMessage, span.StartTime, span.EndTime,
		span.DurationMs, span.TokensIn, span.TokensOut, span.CostUSD, span.Model, span.Provider,
		span.AttributesJSON(), span.ResourceAttributesJSON(), span.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert span: %w", err)
	}

	// Update task aggregates
	if err := UpdateTaskAggregates(ctx, tx, span); err != nil {
		return err
	}

	// Update agent assignment aggregates
	if err := UpdateAgentAssignmentAggregates(ctx, tx, span); err != nil {
		return err
	}

	return tx.Commit()
}

func (b *SQLiteBackend) GetSpan(ctx context.Context, id string) (*Span, error) {
	return b.store.GetSpan(id)
}

func (b *SQLiteBackend) ListSpans(ctx context.Context, opts SpanListOptions) ([]*Span, error) {
	return b.store.ListSpans(opts)
}

func (b *SQLiteBackend) UpdateSpan(ctx context.Context, span *Span) error {
	return b.store.UpdateSpan(span)
}

func (b *SQLiteBackend) UpdateSpanLinks(ctx context.Context, spanID, taskID, assignmentID string) error {
	var taskIDArg, assignmentIDArg interface{}
	if taskID != "" {
		taskIDArg = taskID
	}
	if assignmentID != "" {
		assignmentIDArg = assignmentID
	}
	_, err := b.store.DB().ExecContext(ctx, `
		UPDATE spans SET task_id = ?, agent_assignment_id = ?
		WHERE id = ?
	`, taskIDArg, assignmentIDArg, spanID)
	return err
}

func (b *SQLiteBackend) RecalculateTaskAggregates(ctx context.Context, taskID string) error {
	return RecalculateTaskAggregates(ctx, b.store.DB(), taskID)
}

func (b *SQLiteBackend) DeleteSpan(ctx context.Context, id string) error {
	return b.store.DeleteSpan(id)
}

func (b *SQLiteBackend) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	return b.store.GetTrace(traceID)
}

func (b *SQLiteBackend) ListTraces(ctx context.Context, opts TraceQuery) ([]*TraceSummary, error) {
	return b.store.ListTraces(opts)
}

// LookupTaskBySessionID finds task hierarchy for Claude Code session correlation.
// Used by OTLP receiver to link Claude Code internal events to their parent executor span.
func (b *SQLiteBackend) LookupTaskBySessionID(ctx context.Context, sessionID string) (taskID, assignmentID, traceID string) {
	return b.store.LookupTaskBySessionID(sessionID)
}

// LinkOrphanedSpansBySession updates orphaned spans when claude.execute span arrives.
// Fixes race condition where Claude Code events arrive before parent span is batch-flushed.
func (b *SQLiteBackend) LinkOrphanedSpansBySession(ctx context.Context, sessionID, taskID, assignmentID string) (int64, error) {
	return b.store.LinkOrphanedSpansBySession(sessionID, taskID, assignmentID)
}

// ===== Span Event Operations =====

func (b *SQLiteBackend) CreateSpanEvent(ctx context.Context, e *SpanEvent) error {
	return b.store.CreateSpanEvent(e)
}

func (b *SQLiteBackend) GetSpanEvents(ctx context.Context, spanID string) ([]SpanEvent, error) {
	return b.store.GetSpanEvents(spanID)
}

func (b *SQLiteBackend) DeleteSpanEvent(ctx context.Context, id int64) error {
	return b.store.DeleteSpanEvent(id)
}

// ===== Message Operations =====

func (b *SQLiteBackend) CreateMessage(ctx context.Context, m *Message) error {
	return b.store.CreateMessage(m)
}

func (b *SQLiteBackend) GetMessage(ctx context.Context, id string) (*Message, error) {
	return b.store.GetMessage(id)
}

func (b *SQLiteBackend) ListMessages(ctx context.Context, opts MessageListOptions) ([]*Message, error) {
	return b.store.ListMessages(opts)
}

func (b *SQLiteBackend) UpdateMessage(ctx context.Context, m *Message) error {
	return b.store.UpdateMessage(m)
}

func (b *SQLiteBackend) DeleteMessage(ctx context.Context, id string) error {
	return b.store.DeleteMessage(id)
}

func (b *SQLiteBackend) MarkMessageRead(ctx context.Context, id string) error {
	return b.store.MarkMessageRead(id)
}

func (b *SQLiteBackend) MarkMessageArchived(ctx context.Context, id string) error {
	return b.store.MarkMessageArchived(id)
}

// ===== Aggregate Operations =====

func (b *SQLiteBackend) GetMetricsSummary(ctx context.Context) (*MetricsSummary, error) {
	return b.store.GetMetricsSummary()
}

func (b *SQLiteBackend) GetProviderComparison(ctx context.Context) ([]*ProviderComparison, error) {
	return b.store.GetProviderComparison()
}

func (b *SQLiteBackend) GetTaskTimeline(ctx context.Context, taskID string) ([]*TaskTimeline, error) {
	return b.store.GetTaskTimeline(taskID)
}

// ===== Breakdown Queries for Control Plane =====

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
				WHEN name LIKE 'ailang %' THEN 'local'
				ELSE 'other'
			END as id,
			CASE
				WHEN name LIKE 'api_request%' THEN 'Eval Benchmarks'
				WHEN name LIKE 'eval.%' THEN 'Eval Benchmarks'
				WHEN name LIKE 'coordinator.%' OR name LIKE 'claude.execute%' THEN 'Coordinator Tasks'
				WHEN name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%' THEN 'Direct API Calls'
				WHEN name LIKE 'ailang %' THEN 'Local Usage'
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

// ===== Filtered Queries for Control Plane Interactive Filtering =====

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
				WHEN name LIKE 'ailang %%' THEN 'local'
				ELSE 'other'
			END as id,
			CASE
				WHEN name LIKE 'api_request%%' THEN 'Eval Benchmarks'
				WHEN name LIKE 'eval.%%' THEN 'Eval Benchmarks'
				WHEN name LIKE 'coordinator.%%' OR name LIKE 'claude.execute%%' THEN 'Coordinator Tasks'
				WHEN name LIKE 'anthropic.%%' OR name LIKE 'gemini.%%' OR name LIKE 'openai.%%' THEN 'Direct API Calls'
				WHEN name LIKE 'ailang %%' THEN 'Local Usage'
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

// Ensure SQLiteBackend implements Backend
var _ Backend = (*SQLiteBackend)(nil)
