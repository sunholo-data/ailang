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

// Ensure SQLiteBackend implements Backend
var _ Backend = (*SQLiteBackend)(nil)
