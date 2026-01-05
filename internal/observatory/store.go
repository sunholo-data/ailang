// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Store provides CRUD operations for the observatory platform.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store with the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// ===== Workspace Operations =====

// CreateWorkspace inserts a new workspace.
func (s *Store) CreateWorkspace(w *Workspace) error {
	_, err := s.db.Exec(`
		INSERT INTO workspaces (id, name, path, git_remote, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, w.ID, w.Name, w.Path, w.GitRemote, w.CreatedAt, w.UpdatedAt)
	return err
}

// GetWorkspace retrieves a workspace by ID.
func (s *Store) GetWorkspace(id string) (*Workspace, error) {
	w := &Workspace{}
	var gitRemote sql.NullString
	err := s.db.QueryRow(`
		SELECT id, name, path, git_remote, created_at, updated_at
		FROM workspaces WHERE id = ?
	`, id).Scan(&w.ID, &w.Name, &w.Path, &gitRemote, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if gitRemote.Valid {
		w.GitRemote = gitRemote.String
	}
	return w, nil
}

// ListWorkspaces returns all workspaces.
func (s *Store) ListWorkspaces() ([]*Workspace, error) {
	rows, err := s.db.Query(`
		SELECT id, name, path, git_remote, created_at, updated_at
		FROM workspaces ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []*Workspace
	for rows.Next() {
		w := &Workspace{}
		var gitRemote sql.NullString
		if err := rows.Scan(&w.ID, &w.Name, &w.Path, &gitRemote, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		if gitRemote.Valid {
			w.GitRemote = gitRemote.String
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

// UpdateWorkspace updates an existing workspace.
func (s *Store) UpdateWorkspace(w *Workspace) error {
	w.UpdatedAt = time.Now()
	_, err := s.db.Exec(`
		UPDATE workspaces SET name = ?, path = ?, git_remote = ?, updated_at = ?
		WHERE id = ?
	`, w.Name, w.Path, w.GitRemote, w.UpdatedAt, w.ID)
	return err
}

// DeleteWorkspace removes a workspace by ID.
func (s *Store) DeleteWorkspace(id string) error {
	_, err := s.db.Exec("DELETE FROM workspaces WHERE id = ?", id)
	return err
}

// GetWorkspaceStats returns aggregated stats for a workspace.
func (s *Store) GetWorkspaceStats(workspaceID string) (*WorkspaceStats, error) {
	stats := &WorkspaceStats{}
	var lastActivity sql.NullTime
	err := s.db.QueryRow(`
		SELECT id, name, path, task_count, total_cost, total_tokens,
		       success_rate, unique_agents, last_activity
		FROM workspace_stats WHERE id = ?
	`, workspaceID).Scan(
		&stats.ID, &stats.Name, &stats.Path, &stats.TaskCount, &stats.TotalCost,
		&stats.TotalTokens, &stats.SuccessRate, &stats.UniqueAgents, &lastActivity,
	)
	if err != nil {
		return nil, err
	}
	if lastActivity.Valid {
		stats.LastActivity = lastActivity.Time
	}
	return stats, nil
}

// ===== Task Operations =====

// CreateTask inserts a new task.
func (s *Store) CreateTask(t *Task) error {
	_, err := s.db.Exec(`
		INSERT INTO tasks (id, workspace_id, title, description, source_type, source_ref,
		                   status, priority, created_at, started_at, completed_at,
		                   total_duration_ms, total_tokens_in, total_tokens_out,
		                   total_cost_usd, agent_count, span_count, error_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.WorkspaceID, t.Title, t.Description, t.SourceType, t.SourceRef,
		t.Status, t.Priority, t.CreatedAt, t.StartedAt, t.CompletedAt,
		t.TotalDurationMs, t.TotalTokensIn, t.TotalTokensOut,
		t.TotalCostUSD, t.AgentCount, t.SpanCount, t.ErrorCount)
	return err
}

// GetTask retrieves a task by ID.
func (s *Store) GetTask(id string) (*Task, error) {
	t := &Task{}
	var desc, sourceRef sql.NullString
	var startedAt, completedAt sql.NullTime
	err := s.db.QueryRow(`
		SELECT id, workspace_id, title, description, source_type, source_ref,
		       status, priority, created_at, started_at, completed_at,
		       total_duration_ms, total_tokens_in, total_tokens_out,
		       total_cost_usd, agent_count, span_count, error_count
		FROM tasks WHERE id = ?
	`, id).Scan(&t.ID, &t.WorkspaceID, &t.Title, &desc, &t.SourceType, &sourceRef,
		&t.Status, &t.Priority, &t.CreatedAt, &startedAt, &completedAt,
		&t.TotalDurationMs, &t.TotalTokensIn, &t.TotalTokensOut,
		&t.TotalCostUSD, &t.AgentCount, &t.SpanCount, &t.ErrorCount)
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		t.Description = desc.String
	}
	if sourceRef.Valid {
		t.SourceRef = sourceRef.String
	}
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return t, nil
}

// ListTasks returns tasks with optional filtering.
func (s *Store) ListTasks(opts TaskListOptions) ([]*Task, error) {
	query := `
		SELECT id, workspace_id, title, description, source_type, source_ref,
		       status, priority, created_at, started_at, completed_at,
		       total_duration_ms, total_tokens_in, total_tokens_out,
		       total_cost_usd, agent_count, span_count, error_count
		FROM tasks WHERE 1=1
	`
	var args []interface{}

	if opts.WorkspaceID != "" {
		query += " AND workspace_id = ?"
		args = append(args, opts.WorkspaceID)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.SourceType != "" {
		query += " AND source_type = ?"
		args = append(args, opts.SourceType)
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t := &Task{}
		var desc, sourceRef sql.NullString
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.Title, &desc, &t.SourceType, &sourceRef,
			&t.Status, &t.Priority, &t.CreatedAt, &startedAt, &completedAt,
			&t.TotalDurationMs, &t.TotalTokensIn, &t.TotalTokensOut,
			&t.TotalCostUSD, &t.AgentCount, &t.SpanCount, &t.ErrorCount); err != nil {
			return nil, err
		}
		if desc.Valid {
			t.Description = desc.String
		}
		if sourceRef.Valid {
			t.SourceRef = sourceRef.String
		}
		if startedAt.Valid {
			t.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// TaskListOptions configures task listing.
type TaskListOptions struct {
	WorkspaceID string
	Status      TaskStatus
	SourceType  TaskSourceType
	Limit       int
	Offset      int
}

// UpdateTask updates an existing task.
func (s *Store) UpdateTask(t *Task) error {
	_, err := s.db.Exec(`
		UPDATE tasks SET title = ?, description = ?, source_type = ?, source_ref = ?,
		                 status = ?, priority = ?, started_at = ?, completed_at = ?,
		                 total_duration_ms = ?, total_tokens_in = ?, total_tokens_out = ?,
		                 total_cost_usd = ?, agent_count = ?, span_count = ?, error_count = ?
		WHERE id = ?
	`, t.Title, t.Description, t.SourceType, t.SourceRef,
		t.Status, t.Priority, t.StartedAt, t.CompletedAt,
		t.TotalDurationMs, t.TotalTokensIn, t.TotalTokensOut,
		t.TotalCostUSD, t.AgentCount, t.SpanCount, t.ErrorCount, t.ID)
	return err
}

// DeleteTask removes a task by ID.
func (s *Store) DeleteTask(id string) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

// ===== Agent Assignment Operations =====

// CreateAgentAssignment inserts a new agent assignment.
func (s *Store) CreateAgentAssignment(a *AgentAssignment) error {
	// Convert empty strings to NULL for foreign key columns
	var parentID interface{}
	if a.ParentAssignmentID != "" {
		parentID = a.ParentAssignmentID
	}

	_, err := s.db.Exec(`
		INSERT INTO agent_assignments (id, task_id, agent_id, provider, status,
		                               assigned_at, started_at, completed_at,
		                               parent_assignment_id, duration_ms,
		                               tokens_in, tokens_out, cost_usd, tool_calls, turns)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.TaskID, a.AgentID, a.Provider, a.Status,
		a.AssignedAt, a.StartedAt, a.CompletedAt,
		parentID, a.DurationMs,
		a.TokensIn, a.TokensOut, a.CostUSD, a.ToolCalls, a.Turns)
	return err
}

// GetAgentAssignment retrieves an agent assignment by ID.
func (s *Store) GetAgentAssignment(id string) (*AgentAssignment, error) {
	a := &AgentAssignment{}
	var parentID sql.NullString
	var startedAt, completedAt sql.NullTime
	err := s.db.QueryRow(`
		SELECT id, task_id, agent_id, provider, status,
		       assigned_at, started_at, completed_at,
		       parent_assignment_id, duration_ms,
		       tokens_in, tokens_out, cost_usd, tool_calls, turns
		FROM agent_assignments WHERE id = ?
	`, id).Scan(&a.ID, &a.TaskID, &a.AgentID, &a.Provider, &a.Status,
		&a.AssignedAt, &startedAt, &completedAt,
		&parentID, &a.DurationMs,
		&a.TokensIn, &a.TokensOut, &a.CostUSD, &a.ToolCalls, &a.Turns)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		a.ParentAssignmentID = parentID.String
	}
	if startedAt.Valid {
		a.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		a.CompletedAt = &completedAt.Time
	}
	return a, nil
}

// ListAgentAssignments returns assignments for a task.
func (s *Store) ListAgentAssignments(taskID string) ([]*AgentAssignment, error) {
	rows, err := s.db.Query(`
		SELECT id, task_id, agent_id, provider, status,
		       assigned_at, started_at, completed_at,
		       parent_assignment_id, duration_ms,
		       tokens_in, tokens_out, cost_usd, tool_calls, turns
		FROM agent_assignments WHERE task_id = ?
		ORDER BY assigned_at ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []*AgentAssignment
	for rows.Next() {
		a := &AgentAssignment{}
		var parentID sql.NullString
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&a.ID, &a.TaskID, &a.AgentID, &a.Provider, &a.Status,
			&a.AssignedAt, &startedAt, &completedAt,
			&parentID, &a.DurationMs,
			&a.TokensIn, &a.TokensOut, &a.CostUSD, &a.ToolCalls, &a.Turns); err != nil {
			return nil, err
		}
		if parentID.Valid {
			a.ParentAssignmentID = parentID.String
		}
		if startedAt.Valid {
			a.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			a.CompletedAt = &completedAt.Time
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

// UpdateAgentAssignment updates an existing assignment.
func (s *Store) UpdateAgentAssignment(a *AgentAssignment) error {
	_, err := s.db.Exec(`
		UPDATE agent_assignments SET status = ?, started_at = ?, completed_at = ?,
		                             duration_ms = ?, tokens_in = ?, tokens_out = ?,
		                             cost_usd = ?, tool_calls = ?, turns = ?
		WHERE id = ?
	`, a.Status, a.StartedAt, a.CompletedAt,
		a.DurationMs, a.TokensIn, a.TokensOut,
		a.CostUSD, a.ToolCalls, a.Turns, a.ID)
	return err
}

// DeleteAgentAssignment removes an assignment by ID.
func (s *Store) DeleteAgentAssignment(id string) error {
	_, err := s.db.Exec("DELETE FROM agent_assignments WHERE id = ?", id)
	return err
}

// GetAgentStats returns aggregated stats for an agent.
func (s *Store) GetAgentStats(agentID string) (*AgentStats, error) {
	stats := &AgentStats{}
	err := s.db.QueryRow(`
		SELECT agent_id, provider, execution_count, total_duration_ms,
		       avg_duration_ms, total_tokens_in, total_tokens_out,
		       total_cost, total_tool_calls, success_rate
		FROM agent_stats WHERE agent_id = ?
	`, agentID).Scan(
		&stats.AgentID, &stats.Provider, &stats.ExecutionCount, &stats.TotalDurationMs,
		&stats.AvgDurationMs, &stats.TotalTokensIn, &stats.TotalTokensOut,
		&stats.TotalCost, &stats.TotalToolCalls, &stats.SuccessRate,
	)
	return stats, err
}

// ===== Span Operations =====

// CreateSpan inserts a new span.
func (s *Store) CreateSpan(span *Span) error {
	// Convert empty strings to NULL for foreign key columns
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

	_, err := s.db.Exec(`
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
		return err
	}

	// Update task aggregates if span is linked to a task (M-TASK-HIERARCHY M4)
	if span.TaskID != "" {
		if err := s.updateTaskAggregatesFromSpan(span); err != nil {
			// Log but don't fail span creation
			fmt.Printf("observatory: warning: failed to update task aggregates for task %s: %v\n", span.TaskID, err)
		}
	}

	return nil
}

// updateTaskAggregatesFromSpan updates task totals when a span is added.
// Called from CreateSpan to maintain real-time aggregates.
func (s *Store) updateTaskAggregatesFromSpan(span *Span) error {
	// Update task with span metrics
	// Using atomic UPDATE to handle concurrent span insertions
	isError := 0
	if span.Status == SpanStatusError {
		isError = 1
	}

	_, err := s.db.Exec(`
		UPDATE tasks SET
			span_count = span_count + 1,
			total_tokens_in = total_tokens_in + ?,
			total_tokens_out = total_tokens_out + ?,
			total_cost_usd = total_cost_usd + ?,
			total_duration_ms = total_duration_ms + ?,
			error_count = error_count + ?,
			updated_at = datetime('now')
		WHERE id = ?
	`, span.TokensIn, span.TokensOut, span.CostUSD, span.DurationMs, isError, span.TaskID)
	return err
}

// GetSpan retrieves a span by ID.
func (s *Store) GetSpan(id string) (*Span, error) {
	span := &Span{}
	var parentSpanID, taskID, agentAssignmentID, statusMessage, model sql.NullString
	var provider sql.NullString
	var endTime sql.NullTime
	var attributesJSON, resourceAttributesJSON string

	err := s.db.QueryRow(`
		SELECT id, trace_id, parent_span_id, task_id, agent_assignment_id,
		       name, kind, status, status_message, start_time, end_time,
		       duration_ms, tokens_in, tokens_out, cost_usd, model, provider,
		       attributes, resource_attributes, created_at
		FROM spans WHERE id = ?
	`, id).Scan(&span.ID, &span.TraceID, &parentSpanID, &taskID, &agentAssignmentID,
		&span.Name, &span.Kind, &span.Status, &statusMessage, &span.StartTime, &endTime,
		&span.DurationMs, &span.TokensIn, &span.TokensOut, &span.CostUSD, &model, &provider,
		&attributesJSON, &resourceAttributesJSON, &span.CreatedAt)
	if err != nil {
		return nil, err
	}

	if parentSpanID.Valid {
		span.ParentSpanID = parentSpanID.String
	}
	if taskID.Valid {
		span.TaskID = taskID.String
	}
	if agentAssignmentID.Valid {
		span.AgentAssignmentID = agentAssignmentID.String
	}
	if statusMessage.Valid {
		span.StatusMessage = statusMessage.String
	}
	if endTime.Valid {
		span.EndTime = &endTime.Time
	}
	if model.Valid {
		span.Model = model.String
	}
	if provider.Valid {
		span.Provider = Provider(provider.String)
	}

	span.ParseAttributes(attributesJSON)
	span.ParseResourceAttributes(resourceAttributesJSON)

	return span, nil
}

// ListSpans returns spans with optional filtering.
func (s *Store) ListSpans(opts SpanListOptions) ([]*Span, error) {
	query := `
		SELECT id, trace_id, parent_span_id, task_id, agent_assignment_id,
		       name, kind, status, status_message, start_time, end_time,
		       duration_ms, tokens_in, tokens_out, cost_usd, model, provider,
		       attributes, resource_attributes, created_at
		FROM spans WHERE 1=1
	`
	var args []interface{}

	if opts.TraceID != "" {
		query += " AND trace_id = ?"
		args = append(args, opts.TraceID)
	}
	if opts.TaskID != "" {
		query += " AND task_id = ?"
		args = append(args, opts.TaskID)
	}
	if opts.AgentAssignmentID != "" {
		query += " AND agent_assignment_id = ?"
		args = append(args, opts.AgentAssignmentID)
	}
	if !opts.StartAfter.IsZero() {
		query += " AND start_time >= ?"
		args = append(args, opts.StartAfter)
	}
	if !opts.StartBefore.IsZero() {
		query += " AND start_time <= ?"
		args = append(args, opts.StartBefore)
	}

	query += " ORDER BY start_time ASC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []*Span
	for rows.Next() {
		span := &Span{}
		var parentSpanID, taskID, agentAssignmentID, statusMessage, model sql.NullString
		var provider sql.NullString
		var endTime sql.NullTime
		var attributesJSON, resourceAttributesJSON string

		if err := rows.Scan(&span.ID, &span.TraceID, &parentSpanID, &taskID, &agentAssignmentID,
			&span.Name, &span.Kind, &span.Status, &statusMessage, &span.StartTime, &endTime,
			&span.DurationMs, &span.TokensIn, &span.TokensOut, &span.CostUSD, &model, &provider,
			&attributesJSON, &resourceAttributesJSON, &span.CreatedAt); err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if statusMessage.Valid {
			span.StatusMessage = statusMessage.String
		}
		if endTime.Valid {
			span.EndTime = &endTime.Time
		}
		if model.Valid {
			span.Model = model.String
		}
		if provider.Valid {
			span.Provider = Provider(provider.String)
		}

		span.ParseAttributes(attributesJSON)
		span.ParseResourceAttributes(resourceAttributesJSON)

		spans = append(spans, span)
	}
	return spans, rows.Err()
}

// SpanListOptions configures span listing.
type SpanListOptions struct {
	TraceID           string
	TaskID            string
	AgentAssignmentID string
	StartAfter        time.Time
	StartBefore       time.Time
	Limit             int
	Offset            int
}

// UpdateSpan updates an existing span.
func (s *Store) UpdateSpan(span *Span) error {
	_, err := s.db.Exec(`
		UPDATE spans SET status = ?, status_message = ?, end_time = ?,
		                 duration_ms = ?, tokens_in = ?, tokens_out = ?,
		                 cost_usd = ?, attributes = ?, resource_attributes = ?
		WHERE id = ?
	`, span.Status, span.StatusMessage, span.EndTime,
		span.DurationMs, span.TokensIn, span.TokensOut,
		span.CostUSD, span.AttributesJSON(), span.ResourceAttributesJSON(), span.ID)
	return err
}

// DeleteSpan removes a span by ID.
func (s *Store) DeleteSpan(id string) error {
	_, err := s.db.Exec("DELETE FROM spans WHERE id = ?", id)
	return err
}

// GetTrace retrieves a complete trace with all spans.
func (s *Store) GetTrace(traceID string) (*Trace, error) {
	spans, err := s.ListSpans(SpanListOptions{TraceID: traceID})
	if err != nil {
		return nil, err
	}
	if len(spans) == 0 {
		return nil, sql.ErrNoRows
	}

	trace := &Trace{
		TraceID:   traceID,
		Spans:     spans,
		SpanCount: len(spans),
	}

	// Find root span and calculate duration
	for _, span := range spans {
		if span.ParentSpanID == "" {
			trace.RootSpan = span
		}
		if trace.StartTime.IsZero() || span.StartTime.Before(trace.StartTime) {
			trace.StartTime = span.StartTime
		}
		if span.EndTime != nil && span.EndTime.After(trace.EndTime) {
			trace.EndTime = *span.EndTime
		}
	}

	if !trace.EndTime.IsZero() {
		trace.DurationMs = trace.EndTime.Sub(trace.StartTime).Milliseconds()
	}

	return trace, nil
}

// ListTraces returns trace summaries.
func (s *Store) ListTraces(opts TraceQuery) ([]*TraceSummary, error) {
	query := `
		SELECT trace_id,
		       (SELECT name FROM spans s2 WHERE s2.trace_id = s.trace_id AND s2.parent_span_id IS NULL LIMIT 1) as root_span,
		       COUNT(*) as span_count,
		       COALESCE(SUM(duration_ms), 0) as duration_ms,
		       MIN(start_time) as start_time,
		       (SELECT status FROM spans s3 WHERE s3.trace_id = s.trace_id AND s3.parent_span_id IS NULL LIMIT 1) as status,
		       task_id,
		       (SELECT resource_attributes FROM spans s4 WHERE s4.trace_id = s.trace_id AND s4.parent_span_id IS NULL LIMIT 1) as resource_attrs
		FROM spans s
		WHERE 1=1
	`
	var args []interface{}

	if opts.TaskID != "" {
		query += " AND task_id = ?"
		args = append(args, opts.TaskID)
	}
	if opts.TraceID != "" {
		query += " AND trace_id = ?"
		args = append(args, opts.TraceID)
	}
	if opts.TimeRange != nil {
		query += " AND start_time >= ? AND start_time <= ?"
		args = append(args, opts.TimeRange.Start, opts.TimeRange.End)
	}

	query += " GROUP BY trace_id ORDER BY start_time DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*TraceSummary
	for rows.Next() {
		ts := &TraceSummary{
			Source: TraceSourceLocal, // Mark as coming from local OTLP
		}
		var rootSpan, status, taskID, resourceAttrs sql.NullString
		var startTimeStr string
		if err := rows.Scan(&ts.TraceID, &rootSpan, &ts.SpanCount, &ts.DurationMs,
			&startTimeStr, &status, &taskID, &resourceAttrs); err != nil {
			return nil, err
		}
		if rootSpan.Valid {
			ts.RootSpan = rootSpan.String
		}
		if status.Valid {
			ts.Status = SpanStatus(status.String)
		}
		if taskID.Valid {
			ts.TaskID = taskID.String
		}
		// Extract service.name from resource_attributes JSON
		if resourceAttrs.Valid && resourceAttrs.String != "" {
			ts.ServiceName = extractServiceName(resourceAttrs.String)
		}
		// Parse start_time from string (SQLite MIN() returns string)
		// SQLite stores as "2006-01-02 15:04:05.999999999+07:00" (space, not T)
		if parsedTime, err := time.Parse(time.RFC3339Nano, startTimeStr); err == nil {
			ts.StartTime = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", startTimeStr); err == nil {
			ts.StartTime = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05-07:00", startTimeStr); err == nil {
			ts.StartTime = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", startTimeStr); err == nil {
			ts.StartTime = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05", startTimeStr); err == nil {
			ts.StartTime = parsedTime
		}
		summaries = append(summaries, ts)
	}
	return summaries, rows.Err()
}

// extractServiceName extracts service.name from resource_attributes JSON.
func extractServiceName(jsonStr string) string {
	if jsonStr == "" || jsonStr == "{}" {
		return ""
	}
	var attrs map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &attrs); err != nil {
		return ""
	}
	if sn, ok := attrs["service.name"].(string); ok {
		return sn
	}
	return ""
}

// ===== Span Event Operations =====

// CreateSpanEvent inserts a new span event.
func (s *Store) CreateSpanEvent(e *SpanEvent) error {
	result, err := s.db.Exec(`
		INSERT INTO span_events (span_id, name, timestamp, event_type,
		                         approval_status, tool_name, error_message, attributes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, e.SpanID, e.Name, e.Timestamp, e.EventType,
		e.ApprovalStatus, e.ToolName, e.ErrorMessage, e.AttributesJSON())
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	e.ID = id
	return nil
}

// GetSpanEvents retrieves all events for a span.
func (s *Store) GetSpanEvents(spanID string) ([]SpanEvent, error) {
	rows, err := s.db.Query(`
		SELECT id, span_id, name, timestamp, event_type,
		       approval_status, tool_name, error_message, attributes
		FROM span_events WHERE span_id = ?
		ORDER BY timestamp ASC
	`, spanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SpanEvent
	for rows.Next() {
		e := SpanEvent{}
		var eventType, approvalStatus, toolName, errorMessage sql.NullString
		var attributesJSON string
		if err := rows.Scan(&e.ID, &e.SpanID, &e.Name, &e.Timestamp, &eventType,
			&approvalStatus, &toolName, &errorMessage, &attributesJSON); err != nil {
			return nil, err
		}
		if eventType.Valid {
			e.EventType = EventType(eventType.String)
		}
		if approvalStatus.Valid {
			e.ApprovalStatus = ApprovalStatus(approvalStatus.String)
		}
		if toolName.Valid {
			e.ToolName = toolName.String
		}
		if errorMessage.Valid {
			e.ErrorMessage = errorMessage.String
		}
		e.ParseAttributes(attributesJSON)
		events = append(events, e)
	}
	return events, rows.Err()
}

// DeleteSpanEvent removes a span event by ID.
func (s *Store) DeleteSpanEvent(id int64) error {
	_, err := s.db.Exec("DELETE FROM span_events WHERE id = ?", id)
	return err
}

// ===== Message Operations =====

// CreateMessage inserts a new message.
func (s *Store) CreateMessage(m *Message) error {
	// Convert empty strings to NULL for foreign key columns
	var taskID, replyToID interface{}
	if m.TaskID != "" {
		taskID = m.TaskID
	}
	if m.ReplyToID != "" {
		replyToID = m.ReplyToID
	}

	// Convert zero to NULL for github_issue_number
	var ghIssue interface{}
	if m.GitHubIssueNumber != 0 {
		ghIssue = m.GitHubIssueNumber
	}

	_, err := s.db.Exec(`
		INSERT INTO messages (id, task_id, inbox, from_agent, title, content,
		                      message_type, status, priority, github_issue_number,
		                      github_repo, correlation_id, reply_to_id, created_at,
		                      read_at, archived_at, content_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, taskID, m.Inbox, m.FromAgent, m.Title, m.Content,
		m.MessageType, m.Status, m.Priority, ghIssue,
		m.GitHubRepo, m.CorrelationID, replyToID, m.CreatedAt,
		m.ReadAt, m.ArchivedAt, m.ContentHash)
	return err
}

// GetMessage retrieves a message by ID.
func (s *Store) GetMessage(id string) (*Message, error) {
	m := &Message{}
	var taskID, correlationID, replyToID, contentHash, ghRepo sql.NullString
	var ghIssue sql.NullInt64
	var readAt, archivedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, task_id, inbox, from_agent, title, content,
		       message_type, status, priority, github_issue_number,
		       github_repo, correlation_id, reply_to_id, created_at,
		       read_at, archived_at, content_hash
		FROM messages WHERE id = ?
	`, id).Scan(&m.ID, &taskID, &m.Inbox, &m.FromAgent, &m.Title, &m.Content,
		&m.MessageType, &m.Status, &m.Priority, &ghIssue,
		&ghRepo, &correlationID, &replyToID, &m.CreatedAt,
		&readAt, &archivedAt, &contentHash)
	if err != nil {
		return nil, err
	}

	if taskID.Valid {
		m.TaskID = taskID.String
	}
	if correlationID.Valid {
		m.CorrelationID = correlationID.String
	}
	if replyToID.Valid {
		m.ReplyToID = replyToID.String
	}
	if contentHash.Valid {
		m.ContentHash = contentHash.String
	}
	if ghRepo.Valid {
		m.GitHubRepo = ghRepo.String
	}
	if ghIssue.Valid {
		m.GitHubIssueNumber = int(ghIssue.Int64)
	}
	if readAt.Valid {
		m.ReadAt = &readAt.Time
	}
	if archivedAt.Valid {
		m.ArchivedAt = &archivedAt.Time
	}

	return m, nil
}

// ListMessages returns messages with optional filtering.
func (s *Store) ListMessages(opts MessageListOptions) ([]*Message, error) {
	query := `
		SELECT id, task_id, inbox, from_agent, title, content,
		       message_type, status, priority, github_issue_number,
		       github_repo, correlation_id, reply_to_id, created_at,
		       read_at, archived_at, content_hash
		FROM messages WHERE 1=1
	`
	var args []interface{}

	if opts.Inbox != "" {
		query += " AND inbox = ?"
		args = append(args, opts.Inbox)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.TaskID != "" {
		query += " AND task_id = ?"
		args = append(args, opts.TaskID)
	}
	if opts.FromAgent != "" {
		query += " AND from_agent = ?"
		args = append(args, opts.FromAgent)
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		m := &Message{}
		var taskID, correlationID, replyToID, contentHash, ghRepo sql.NullString
		var ghIssue sql.NullInt64
		var readAt, archivedAt sql.NullTime

		if err := rows.Scan(&m.ID, &taskID, &m.Inbox, &m.FromAgent, &m.Title, &m.Content,
			&m.MessageType, &m.Status, &m.Priority, &ghIssue,
			&ghRepo, &correlationID, &replyToID, &m.CreatedAt,
			&readAt, &archivedAt, &contentHash); err != nil {
			return nil, err
		}

		if taskID.Valid {
			m.TaskID = taskID.String
		}
		if correlationID.Valid {
			m.CorrelationID = correlationID.String
		}
		if replyToID.Valid {
			m.ReplyToID = replyToID.String
		}
		if contentHash.Valid {
			m.ContentHash = contentHash.String
		}
		if ghRepo.Valid {
			m.GitHubRepo = ghRepo.String
		}
		if ghIssue.Valid {
			m.GitHubIssueNumber = int(ghIssue.Int64)
		}
		if readAt.Valid {
			m.ReadAt = &readAt.Time
		}
		if archivedAt.Valid {
			m.ArchivedAt = &archivedAt.Time
		}

		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// MessageListOptions configures message listing.
type MessageListOptions struct {
	Inbox     string
	Status    MessageStatus
	TaskID    string
	FromAgent string
	Limit     int
	Offset    int
}

// UpdateMessage updates an existing message.
func (s *Store) UpdateMessage(m *Message) error {
	_, err := s.db.Exec(`
		UPDATE messages SET status = ?, read_at = ?, archived_at = ?
		WHERE id = ?
	`, m.Status, m.ReadAt, m.ArchivedAt, m.ID)
	return err
}

// DeleteMessage removes a message by ID.
func (s *Store) DeleteMessage(id string) error {
	_, err := s.db.Exec("DELETE FROM messages WHERE id = ?", id)
	return err
}

// MarkMessageRead marks a message as read.
func (s *Store) MarkMessageRead(id string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE messages SET status = ?, read_at = ? WHERE id = ?
	`, MessageStatusRead, now, id)
	return err
}

// MarkMessageArchived marks a message as archived.
func (s *Store) MarkMessageArchived(id string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE messages SET status = ?, archived_at = ? WHERE id = ?
	`, MessageStatusArchived, now, id)
	return err
}

// ===== Aggregate Queries =====

// GetMetricsSummary returns global metrics.
func (s *Store) GetMetricsSummary() (*MetricsSummary, error) {
	summary := &MetricsSummary{}

	// Count workspaces
	s.db.QueryRow("SELECT COUNT(*) FROM workspaces").Scan(&summary.TotalWorkspaces)

	// Count tasks
	s.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&summary.TotalTasks)

	// Count spans
	s.db.QueryRow("SELECT COUNT(*) FROM spans").Scan(&summary.TotalSpans)

	// Count unique agents
	s.db.QueryRow("SELECT COUNT(DISTINCT agent_id) FROM agent_assignments").Scan(&summary.TotalAgents)

	// Sum tokens and cost
	s.db.QueryRow(`
		SELECT COALESCE(SUM(tokens_in), 0), COALESCE(SUM(tokens_out), 0), COALESCE(SUM(cost_usd), 0)
		FROM spans
	`).Scan(&summary.TotalTokensIn, &summary.TotalTokensOut, &summary.TotalCostUSD)

	// Calculate success rate
	var completed, failed int
	s.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'completed'").Scan(&completed)
	s.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'failed'").Scan(&failed)
	if completed+failed > 0 {
		summary.SuccessRate = float64(completed) / float64(completed+failed) * 100
	}

	return summary, nil
}

// GetProviderComparison returns comparison metrics across providers.
func (s *Store) GetProviderComparison() ([]*ProviderComparison, error) {
	rows, err := s.db.Query(`
		SELECT provider, total_executions, total_tokens_in, total_tokens_out,
		       total_cost, avg_duration_ms, success_rate
		FROM provider_comparison
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comparisons []*ProviderComparison
	for rows.Next() {
		pc := &ProviderComparison{}
		if err := rows.Scan(&pc.Provider, &pc.TotalExecutions, &pc.TotalTokensIn,
			&pc.TotalTokensOut, &pc.TotalCost, &pc.AvgDurationMs, &pc.SuccessRate); err != nil {
			return nil, err
		}
		comparisons = append(comparisons, pc)
	}
	return comparisons, rows.Err()
}

// GetTaskTimeline returns timeline data for a task.
func (s *Store) GetTaskTimeline(taskID string) ([]*TaskTimeline, error) {
	rows, err := s.db.Query(`
		SELECT task_id, title, status, span_id, span_name, start_time, end_time,
		       duration_ms, span_status, tokens_in, tokens_out, cost_usd, provider
		FROM task_timeline WHERE task_id = ?
		ORDER BY start_time ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var timeline []*TaskTimeline
	for rows.Next() {
		tl := &TaskTimeline{}
		var spanID, spanName sql.NullString
		var startTime, endTime sql.NullTime
		var spanStatus, provider sql.NullString
		if err := rows.Scan(&tl.TaskID, &tl.Title, &tl.Status, &spanID, &spanName,
			&startTime, &endTime, &tl.DurationMs, &spanStatus,
			&tl.TokensIn, &tl.TokensOut, &tl.CostUSD, &provider); err != nil {
			return nil, err
		}
		if spanID.Valid {
			tl.SpanID = spanID.String
		}
		if spanName.Valid {
			tl.SpanName = spanName.String
		}
		if startTime.Valid {
			tl.StartTime = &startTime.Time
		}
		if endTime.Valid {
			tl.EndTime = &endTime.Time
		}
		if spanStatus.Valid {
			tl.SpanStatus = SpanStatus(spanStatus.String)
		}
		if provider.Valid {
			tl.Provider = Provider(provider.String)
		}
		timeline = append(timeline, tl)
	}
	return timeline, rows.Err()
}
