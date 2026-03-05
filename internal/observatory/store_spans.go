// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SpanListOptions configures span listing.
type SpanListOptions struct {
	TraceID           string
	TaskID            string
	AgentAssignmentID string
	StartAfter        time.Time
	StartBefore       time.Time
	Limit             int
	Offset            int
	// Filter by provider (claude, gemini, openai, etc.)
	Provider string
	// Filter by model name (claude-sonnet-4-5, gemini-2-5-flash, etc.)
	Model string
	// Filter by span status (ok, error)
	Status string
	// Filter by workspace path (filters via task→workspace relationship)
	Workspace string
	// Filter by workspace ID directly (more efficient when workspace_id is known)
	WorkspaceID string
}

// CreateSpan inserts a new span.
// Note: This is the low-level insert WITHOUT aggregation updates.
// For transactional span creation with aggregation updates, use SQLiteBackend.CreateSpan
// which wraps this in a transaction with UpdateTaskAggregates and UpdateAgentAssignmentAggregates.
func (s *Store) CreateSpan(span *Span) error {
	// Write-time chain linking: if span has task_id but no chain_id/stage_id,
	// look up the chain stage that owns this task (M-AUDIT-OBSERVATORY Phase 2B).
	if span.TaskID != "" && span.ChainID == "" {
		s.resolveChainLink(span)
	}

	// Convert empty strings to NULL for foreign key columns
	var parentSpanID, taskID, agentAssignmentID, chainID, stageID interface{}
	if span.ParentSpanID != "" {
		parentSpanID = span.ParentSpanID
	}
	if span.TaskID != "" {
		taskID = span.TaskID
	}
	if span.AgentAssignmentID != "" {
		agentAssignmentID = span.AgentAssignmentID
	}
	if span.ChainID != "" {
		chainID = span.ChainID
	}
	if span.StageID != "" {
		stageID = span.StageID
	}

	_, err := s.db.Exec(`
		INSERT INTO spans (id, trace_id, parent_span_id, task_id, agent_assignment_id,
		                   chain_id, stage_id,
		                   name, kind, status, status_message, start_time, end_time,
		                   duration_ms, tokens_in, tokens_out, cache_read_tokens, cache_creation_tokens,
		                   cost_usd, model, provider, attributes, resource_attributes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, span.ID, span.TraceID, parentSpanID, taskID, agentAssignmentID,
		chainID, stageID,
		span.Name, span.Kind, span.Status, span.StatusMessage, span.StartTime, span.EndTime,
		span.DurationMs, span.TokensIn, span.TokensOut, span.CacheReadTokens, span.CacheCreationTokens,
		span.CostUSD, span.Model, span.Provider,
		span.AttributesJSON(), span.ResourceAttributesJSON(), span.CreatedAt)
	if err != nil {
		return err
	}

	// Maintain trace_summaries incrementally (M-PERF-OBSERVATORY)
	s.upsertTraceSummary(span)

	return nil
}

// upsertTraceSummary maintains the trace_summaries materialized table incrementally.
// Called from CreateSpan to keep summaries up to date without expensive aggregation queries.
// Best-effort: errors are silently ignored to not block span creation.
func (s *Store) upsertTraceSummary(span *Span) {
	if span.TraceID == "" {
		return
	}

	// Determine if this is a root span
	var rootName, rootStatus interface{}
	if span.ParentSpanID == "" {
		rootName = span.Name
		rootStatus = string(span.Status)
	}

	// Extract service name and workspace from resource attributes
	var serviceName, workspace interface{}
	if span.ResourceAttributes != nil {
		if sn, ok := span.ResourceAttributes["service.name"]; ok {
			if s, ok := sn.(string); ok {
				serviceName = s
			}
		}
		if cwd, ok := span.ResourceAttributes["process.cwd"]; ok {
			if s, ok := cwd.(string); ok {
				workspace = s
			}
		}
	}

	_, _ = s.db.Exec(`
		INSERT INTO trace_summaries (trace_id, root_span_name, root_span_status, span_count, total_duration_ms, start_time, task_id, service_name, workspace, updated_at)
		VALUES (?, ?, ?, 1, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(trace_id) DO UPDATE SET
			span_count = trace_summaries.span_count + 1,
			total_duration_ms = trace_summaries.total_duration_ms + excluded.total_duration_ms,
			root_span_name = COALESCE(excluded.root_span_name, trace_summaries.root_span_name),
			root_span_status = COALESCE(excluded.root_span_status, trace_summaries.root_span_status),
			start_time = MIN(trace_summaries.start_time, excluded.start_time),
			task_id = COALESCE(excluded.task_id, trace_summaries.task_id),
			service_name = COALESCE(excluded.service_name, trace_summaries.service_name),
			workspace = COALESCE(excluded.workspace, trace_summaries.workspace),
			updated_at = CURRENT_TIMESTAMP
	`, span.TraceID, rootName, rootStatus, span.DurationMs, span.StartTime, span.TaskID, serviceName, workspace)
}

// resolveChainLink looks up chain_id and stage_id for a span based on its task_id.
// Called at write time to link spans to their execution chain (M-AUDIT-OBSERVATORY).
// Uses a simple in-memory cache to avoid repeated DB lookups for spans in the same task.
// Best-effort: failures don't block span creation.
func (s *Store) resolveChainLink(span *Span) {
	// Check cache first
	s.chainLinkMu.RLock()
	if link, ok := s.chainLinkCache[span.TaskID]; ok {
		s.chainLinkMu.RUnlock()
		span.ChainID = link.chainID
		span.StageID = link.stageID
		return
	}
	s.chainLinkMu.RUnlock()

	// Query chain_stages for this task_id
	var chainID, stageID string
	err := s.db.QueryRow(`
		SELECT chain_id, id FROM chain_stages WHERE task_id = ? LIMIT 1
	`, span.TaskID).Scan(&chainID, &stageID)
	if err != nil {
		// No matching stage — cache the miss to avoid repeated lookups
		s.chainLinkMu.Lock()
		s.chainLinkCache[span.TaskID] = chainLink{}
		s.chainLinkMu.Unlock()
		return
	}

	// Cache and apply
	s.chainLinkMu.Lock()
	s.chainLinkCache[span.TaskID] = chainLink{chainID: chainID, stageID: stageID}
	s.chainLinkMu.Unlock()

	span.ChainID = chainID
	span.StageID = stageID
}

// GetSpan retrieves a span by ID.
func (s *Store) GetSpan(id string) (*Span, error) {
	span := &Span{}
	var parentSpanID, taskID, agentAssignmentID, chainID, stageID, statusMessage, model sql.NullString
	var provider sql.NullString
	var endTime sql.NullTime
	var cacheReadTokens, cacheCreationTokens sql.NullInt64
	var attributesJSON, resourceAttributesJSON string

	err := s.db.QueryRow(`
		SELECT id, trace_id, parent_span_id, task_id, agent_assignment_id,
		       COALESCE(chain_id, ''), COALESCE(stage_id, ''),
		       name, kind, status, status_message, start_time, end_time,
		       duration_ms, tokens_in, tokens_out,
		       COALESCE(cache_read_tokens, 0), COALESCE(cache_creation_tokens, 0),
		       cost_usd, model, provider,
		       attributes, resource_attributes, created_at
		FROM spans WHERE id = ?
	`, id).Scan(&span.ID, &span.TraceID, &parentSpanID, &taskID, &agentAssignmentID,
		&chainID, &stageID,
		&span.Name, &span.Kind, &span.Status, &statusMessage, &span.StartTime, &endTime,
		&span.DurationMs, &span.TokensIn, &span.TokensOut,
		&cacheReadTokens, &cacheCreationTokens,
		&span.CostUSD, &model, &provider,
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
	if chainID.Valid {
		span.ChainID = chainID.String
	}
	if stageID.Valid {
		span.StageID = stageID.String
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
	if cacheReadTokens.Valid {
		span.CacheReadTokens = cacheReadTokens.Int64
	}
	if cacheCreationTokens.Valid {
		span.CacheCreationTokens = cacheCreationTokens.Int64
	}

	span.ParseAttributes(attributesJSON)
	span.ParseResourceAttributes(resourceAttributesJSON)

	return span, nil
}

// ListSpans returns spans with optional filtering.
func (s *Store) ListSpans(opts SpanListOptions) ([]*Span, error) {
	query := `
		SELECT id, trace_id, parent_span_id, task_id, agent_assignment_id,
		       COALESCE(chain_id, ''), COALESCE(stage_id, ''),
		       name, kind, status, status_message, start_time, end_time,
		       duration_ms, tokens_in, tokens_out,
		       COALESCE(cache_read_tokens, 0), COALESCE(cache_creation_tokens, 0),
		       cost_usd, model, provider,
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
	if opts.Provider != "" {
		query += " AND provider = ?"
		args = append(args, opts.Provider)
	}
	if opts.Model != "" {
		query += " AND model = ?"
		args = append(args, opts.Model)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.Workspace != "" {
		// Filter spans by workspace via task→workspace relationship
		query += ` AND task_id IN (
			SELECT id FROM tasks WHERE workspace_id IN (
				SELECT id FROM workspaces WHERE path = ?
			)
		)`
		args = append(args, opts.Workspace)
	}
	if opts.WorkspaceID != "" {
		// Filter spans by workspace_id directly (more efficient)
		query += ` AND task_id IN (
			SELECT id FROM tasks WHERE workspace_id = ?
		)`
		args = append(args, opts.WorkspaceID)
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
		var parentSpanID, taskID, agentAssignmentID, chainID, stageID, statusMessage, model sql.NullString
		var provider sql.NullString
		var endTime sql.NullTime
		var cacheReadTokens, cacheCreationTokens sql.NullInt64
		var attributesJSON, resourceAttributesJSON string

		if err := rows.Scan(&span.ID, &span.TraceID, &parentSpanID, &taskID, &agentAssignmentID,
			&chainID, &stageID,
			&span.Name, &span.Kind, &span.Status, &statusMessage, &span.StartTime, &endTime,
			&span.DurationMs, &span.TokensIn, &span.TokensOut,
			&cacheReadTokens, &cacheCreationTokens,
			&span.CostUSD, &model, &provider,
			&attributesJSON, &resourceAttributesJSON, &span.CreatedAt); err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if cacheReadTokens.Valid {
			span.CacheReadTokens = cacheReadTokens.Int64
		}
		if cacheCreationTokens.Valid {
			span.CacheCreationTokens = cacheCreationTokens.Int64
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if chainID.Valid {
			span.ChainID = chainID.String
		}
		if stageID.Valid {
			span.StageID = stageID.String
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

// ListSpansLightweight returns spans without the heavy attributes/resource_attributes columns.
// Instead of reading the full JSON blobs (3.9GB total), it extracts only the specific
// attribute fields needed for enrichment: tool_name and session.id.
// This is 10-50x faster than ListSpans for large datasets. (M-AUDIT-OBSERVATORY)
func (s *Store) ListSpansLightweight(opts SpanListOptions) ([]*Span, error) {
	query := `
		SELECT id, trace_id, parent_span_id, task_id, agent_assignment_id,
		       COALESCE(chain_id, ''), COALESCE(stage_id, ''),
		       name, kind, status, status_message, start_time, end_time,
		       duration_ms, tokens_in, tokens_out,
		       COALESCE(cache_read_tokens, 0), COALESCE(cache_creation_tokens, 0),
		       cost_usd, model, provider,
		       json_extract(attributes, '$.tool_name'),
		       json_extract(attributes, '$."session.id"'),
		       created_at
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
	if opts.Provider != "" {
		query += " AND provider = ?"
		args = append(args, opts.Provider)
	}
	if opts.Model != "" {
		query += " AND model = ?"
		args = append(args, opts.Model)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.Workspace != "" {
		query += ` AND task_id IN (
			SELECT id FROM tasks WHERE workspace_id IN (
				SELECT id FROM workspaces WHERE path = ?
			)
		)`
		args = append(args, opts.Workspace)
	}
	if opts.WorkspaceID != "" {
		query += ` AND task_id IN (
			SELECT id FROM tasks WHERE workspace_id = ?
		)`
		args = append(args, opts.WorkspaceID)
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
		var parentSpanID, taskID, agentAssignmentID, chainID, stageID, statusMessage, model sql.NullString
		var provider sql.NullString
		var endTime sql.NullTime
		var cacheReadTokens, cacheCreationTokens sql.NullInt64
		var toolName, sessionID sql.NullString

		if err := rows.Scan(&span.ID, &span.TraceID, &parentSpanID, &taskID, &agentAssignmentID,
			&chainID, &stageID,
			&span.Name, &span.Kind, &span.Status, &statusMessage, &span.StartTime, &endTime,
			&span.DurationMs, &span.TokensIn, &span.TokensOut,
			&cacheReadTokens, &cacheCreationTokens,
			&span.CostUSD, &model, &provider,
			&toolName, &sessionID,
			&span.CreatedAt); err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if cacheReadTokens.Valid {
			span.CacheReadTokens = cacheReadTokens.Int64
		}
		if cacheCreationTokens.Valid {
			span.CacheCreationTokens = cacheCreationTokens.Int64
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if chainID.Valid {
			span.ChainID = chainID.String
		}
		if stageID.Valid {
			span.StageID = stageID.String
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

		// Populate only the attributes needed for enrichment (tool_name, session.id)
		if toolName.Valid || sessionID.Valid {
			span.Attributes = make(map[string]any)
			if toolName.Valid {
				span.Attributes["tool_name"] = toolName.String
			}
			if sessionID.Valid {
				span.Attributes["session.id"] = sessionID.String
			}
		}

		spans = append(spans, span)
	}
	return spans, rows.Err()
}

// ListSpansByTaskIDs fetches spans for multiple task IDs in a single query.
// Returns a map of task_id -> []*Span. Much more efficient than calling
// ListSpans once per task (single query vs N queries).
func (s *Store) ListSpansByTaskIDs(taskIDs []string, limitPerTask int) (map[string][]*Span, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}

	// Build IN clause with placeholders
	placeholders := make([]string, len(taskIDs))
	args := make([]interface{}, len(taskIDs))
	for i, id := range taskIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, trace_id, parent_span_id, task_id, agent_assignment_id,
		       COALESCE(chain_id, ''), COALESCE(stage_id, ''),
		       name, kind, status, status_message, start_time, end_time,
		       duration_ms, tokens_in, tokens_out,
		       COALESCE(cache_read_tokens, 0), COALESCE(cache_creation_tokens, 0),
		       cost_usd, model, provider,
		       attributes, resource_attributes, created_at
		FROM spans
		WHERE task_id IN (%s)
		ORDER BY task_id, start_time ASC
	`, strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]*Span)
	for rows.Next() {
		span := &Span{}
		var parentSpanID, taskID, agentAssignmentID, chainID, stageID, statusMessage, model sql.NullString
		var provider sql.NullString
		var endTime sql.NullTime
		var cacheReadTokens, cacheCreationTokens sql.NullInt64
		var attributesJSON, resourceAttributesJSON string

		if err := rows.Scan(&span.ID, &span.TraceID, &parentSpanID, &taskID, &agentAssignmentID,
			&chainID, &stageID,
			&span.Name, &span.Kind, &span.Status, &statusMessage, &span.StartTime, &endTime,
			&span.DurationMs, &span.TokensIn, &span.TokensOut,
			&cacheReadTokens, &cacheCreationTokens,
			&span.CostUSD, &model, &provider,
			&attributesJSON, &resourceAttributesJSON, &span.CreatedAt); err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if cacheReadTokens.Valid {
			span.CacheReadTokens = cacheReadTokens.Int64
		}
		if cacheCreationTokens.Valid {
			span.CacheCreationTokens = cacheCreationTokens.Int64
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if chainID.Valid {
			span.ChainID = chainID.String
		}
		if stageID.Valid {
			span.StageID = stageID.String
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

		tid := span.TaskID
		if limitPerTask > 0 && len(result[tid]) >= limitPerTask {
			continue // Skip if we already have enough spans for this task
		}
		result[tid] = append(result[tid], span)
	}
	return result, rows.Err()
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

// UpdateSpanLinks updates a span's task and agent assignment links.
// Used for session correlation to link orphaned spans to their parent task.
func (s *Store) UpdateSpanLinks(spanID, taskID, assignmentID string) error {
	var taskIDArg, assignmentIDArg interface{}
	if taskID != "" {
		taskIDArg = taskID
	}
	if assignmentID != "" {
		assignmentIDArg = assignmentID
	}
	_, err := s.db.Exec(`
		UPDATE spans SET task_id = ?, agent_assignment_id = ?
		WHERE id = ?
	`, taskIDArg, assignmentIDArg, spanID)
	return err
}

// DeleteSpan removes a span by ID.
func (s *Store) DeleteSpan(id string) error {
	_, err := s.db.Exec("DELETE FROM spans WHERE id = ?", id)
	return err
}

// LookupTaskBySessionID finds task hierarchy info for a given session ID.
// This enables correlating Claude Code internal events to their parent executor span.
// Returns taskID, agentAssignmentID, traceID (for trace linking), or empty strings if not found.
func (s *Store) LookupTaskBySessionID(sessionID string) (taskID, assignmentID, traceID string) {
	if sessionID == "" {
		return "", "", ""
	}

	// Query for claude.execute span with matching session.id in attributes
	// The session.id key contains a dot, so it must be quoted in the JSON path: $."session.id"
	row := s.db.QueryRow(`
		SELECT task_id, agent_assignment_id, trace_id
		FROM spans
		WHERE name = 'claude.execute'
		AND json_extract(attributes, '$."session.id"') = ?
		ORDER BY start_time DESC
		LIMIT 1
	`, sessionID)

	var taskIDNull, assignmentIDNull, traceIDNull sql.NullString
	if err := row.Scan(&taskIDNull, &assignmentIDNull, &traceIDNull); err != nil {
		return "", "", ""
	}

	return taskIDNull.String, assignmentIDNull.String, traceIDNull.String
}

// LinkOrphanedSpansBySession updates spans that have a matching session.id but no task_id.
// This fixes the race condition where Claude Code internal events arrive via OTLP before
// the claude.execute span is batch-flushed by the OTEL SDK.
// Called after storing a claude.execute span to retroactively link orphaned child spans.
func (s *Store) LinkOrphanedSpansBySession(sessionID, taskID, assignmentID string) (int64, error) {
	if sessionID == "" || taskID == "" {
		return 0, nil
	}

	// Update all spans that:
	// 1. Have matching session.id in attributes
	// 2. Don't already have a task_id (orphaned)
	// 3. Are NOT the claude.execute span itself
	var assignmentArg interface{}
	if assignmentID != "" {
		assignmentArg = assignmentID
	}

	result, err := s.db.Exec(`
		UPDATE spans SET
			task_id = ?,
			agent_assignment_id = ?
		WHERE json_extract(attributes, '$."session.id"') = ?
		AND (task_id IS NULL OR task_id = '')
		AND name != 'claude.execute'
	`, taskID, assignmentArg, sessionID)
	if err != nil {
		return 0, fmt.Errorf("link orphaned spans: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
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
// Uses the trace_summaries materialized table (M-PERF-OBSERVATORY v12 migration)
// instead of correlated subqueries on the spans table.
func (s *Store) ListTraces(opts TraceQuery) ([]*TraceSummary, error) {
	// Try fast path: trace_summaries table (no correlated subqueries)
	summaries, err := s.listTracesFromSummaries(opts)
	if err == nil && len(summaries) > 0 {
		return summaries, nil
	}
	// Fallback: trace_summaries table might be empty (pre-v12 migration)
	// Use the original query with correlated subqueries
	return s.listTracesLegacy(opts)
}

// listTracesFromSummaries reads from the pre-computed trace_summaries table.
// O(1) per row — no correlated subqueries.
func (s *Store) listTracesFromSummaries(opts TraceQuery) ([]*TraceSummary, error) {
	query := `
		SELECT trace_id, root_span_name, root_span_status, span_count,
		       total_duration_ms, start_time, task_id, service_name
		FROM trace_summaries
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

	query += " ORDER BY start_time DESC"

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
			Source: TraceSourceLocal,
		}
		var rootSpan, rootStatus, taskID, serviceName sql.NullString
		var startTimeStr string
		if err := rows.Scan(&ts.TraceID, &rootSpan, &rootStatus, &ts.SpanCount,
			&ts.DurationMs, &startTimeStr, &taskID, &serviceName); err != nil {
			return nil, err
		}
		if rootSpan.Valid {
			ts.RootSpan = rootSpan.String
		}
		if rootStatus.Valid {
			ts.Status = SpanStatus(rootStatus.String)
		}
		if taskID.Valid {
			ts.TaskID = taskID.String
		}
		if serviceName.Valid {
			ts.ServiceName = serviceName.String
		}
		ts.StartTime = parseTimeString(startTimeStr)
		summaries = append(summaries, ts)
	}
	return summaries, rows.Err()
}

// listTracesLegacy is the original query with correlated subqueries.
// Kept as fallback for databases that haven't run the v12 migration yet.
func (s *Store) listTracesLegacy(opts TraceQuery) ([]*TraceSummary, error) {
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
			Source: TraceSourceLocal,
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
		if resourceAttrs.Valid && resourceAttrs.String != "" {
			ts.ServiceName = extractServiceName(resourceAttrs.String)
		}
		ts.StartTime = parseTimeString(startTimeStr)
		summaries = append(summaries, ts)
	}
	return summaries, rows.Err()
}

// parseTimeString handles the various time formats SQLite may return.
func parseTimeString(s string) time.Time {
	formats := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// extractServiceName extracts service.name from resource_attributes JSON.
// TaskSpanSummary holds aggregated span statistics for a task_id.
// Used by "ailang chains find --task-id" when no execution_chain exists
// (e.g., Claude Code user sessions that aren't coordinator-managed).
type TaskSpanSummary struct {
	TaskID     string    `json:"task_id"`
	SpanCount  int       `json:"span_count"`
	TraceCount int       `json:"trace_count"`
	TokensIn   int64     `json:"tokens_in"`
	TokensOut  int64     `json:"tokens_out"`
	CostUSD    float64   `json:"cost_usd"`
	DurationMs int64     `json:"duration_ms"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	// Top span names by count (e.g., "api_request: 487, claude_code.tool.Read: 338")
	TopSpanNames []SpanNameCount `json:"top_span_names"`
}

// SpanNameCount holds a span name and its count.
type SpanNameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// GetTaskSpanSummary returns aggregated span statistics for a task_id.
// Works for ALL task_id formats (coordinator, eval, UUID sessions).
func (s *Store) GetTaskSpanSummary(ctx context.Context, taskID string) (*TaskSpanSummary, error) {
	if taskID == "" {
		return nil, nil
	}

	summary := &TaskSpanSummary{TaskID: taskID}

	// Aggregate metrics (scan times as strings since SQLite returns text)
	var startStr, endStr string
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as span_count,
			COUNT(DISTINCT trace_id) as trace_count,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out,
			COALESCE(SUM(cost_usd), 0) as cost_usd,
			COALESCE(SUM(duration_ms), 0) as duration_ms,
			COALESCE(MIN(start_time), '') as start_time,
			COALESCE(MAX(start_time), '') as end_time
		FROM spans WHERE task_id = ?
	`, taskID).Scan(
		&summary.SpanCount, &summary.TraceCount,
		&summary.TokensIn, &summary.TokensOut,
		&summary.CostUSD, &summary.DurationMs,
		&startStr, &endStr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get task span summary: %w", err)
	}
	// Parse time strings
	if startStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", startStr); err == nil {
			summary.StartTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05.999999999Z07:00", startStr); err == nil {
			summary.StartTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05Z07:00", startStr); err == nil {
			summary.StartTime = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", endStr); err == nil {
			summary.EndTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05.999999999Z07:00", endStr); err == nil {
			summary.EndTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05Z07:00", endStr); err == nil {
			summary.EndTime = t
		}
	}
	if summary.SpanCount == 0 {
		return nil, nil
	}

	// Top span names
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, COUNT(*) as cnt
		FROM spans WHERE task_id = ?
		GROUP BY name ORDER BY cnt DESC LIMIT 10
	`, taskID)
	if err != nil {
		return summary, nil // Non-fatal
	}
	defer rows.Close()
	for rows.Next() {
		var nc SpanNameCount
		if err := rows.Scan(&nc.Name, &nc.Count); err == nil {
			summary.TopSpanNames = append(summary.TopSpanNames, nc)
		}
	}

	return summary, nil
}

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
