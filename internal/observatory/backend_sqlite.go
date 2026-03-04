// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SQLiteBackend implements Backend using SQLite storage.
type SQLiteBackend struct {
	store *Store
}

// Store returns the underlying store for direct queries.
func (b *SQLiteBackend) Store() *Store {
	return b.store
}

// DB returns the underlying database for direct SQL queries.
func (b *SQLiteBackend) DB() *sql.DB {
	if b.store == nil {
		return nil
	}
	return b.store.DB()
}

// NewSQLiteBackend creates a new SQLite backend.
func NewSQLiteBackend(db *sql.DB) (*SQLiteBackend, error) {
	// Use versioned migration to support incremental schema updates
	if _, err := MigrateWithVersion(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	return &SQLiteBackend{store: NewStore(db)}, nil
}

// NewSQLiteBackendFromPath creates a SQLite backend from a file path.
func NewSQLiteBackendFromPath(path string) (*SQLiteBackend, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
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

func (b *SQLiteBackend) ListSpansByTaskIDs(ctx context.Context, taskIDs []string, limitPerTask int) (map[string][]*Span, error) {
	return b.store.ListSpansByTaskIDs(taskIDs, limitPerTask)
}

func (b *SQLiteBackend) UpdateSpan(ctx context.Context, span *Span) error {
	return b.store.UpdateSpan(span)
}

func (b *SQLiteBackend) UpdateSpanLinks(ctx context.Context, spanID, taskID, assignmentID string) error {
	return b.store.UpdateSpanLinks(spanID, taskID, assignmentID)
}

func (b *SQLiteBackend) RecalculateTaskAggregates(ctx context.Context, taskID string) error {
	return b.store.RecalculateTaskAggregates(taskID)
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

func (b *SQLiteBackend) GetExecTaskHierarchy(ctx context.Context, limit int) ([]*ExecTaskNode, error) {
	return b.store.GetExecTaskHierarchy(limit)
}

// ===== Session Operations (M-SESSION-WORKSPACE-HOOKS) =====

func (b *SQLiteBackend) GetSessionWorkspace(sessionID string) (string, error) {
	return b.store.GetSessionWorkspace(sessionID)
}

func (b *SQLiteBackend) UpsertSession(ctx context.Context, sessionID, workspace, version, source string) error {
	return b.store.UpsertSession(ctx, sessionID, workspace, version, source)
}

func (b *SQLiteBackend) UpsertSessionWithCorrelation(ctx context.Context, sessionID, workspace, version, source string, corr *SessionCorrelation) error {
	return b.store.UpsertSessionWithCorrelation(ctx, sessionID, workspace, version, source, corr)
}

func (b *SQLiteBackend) UpdateSessionEnded(ctx context.Context, sessionID string) error {
	return b.store.UpdateSessionEnded(ctx, sessionID)
}

func (b *SQLiteBackend) InsertToolStart(ctx context.Context, sessionID, toolUseID, toolName, toolInput string) error {
	return b.store.InsertToolStart(ctx, sessionID, toolUseID, toolName, toolInput)
}

func (b *SQLiteBackend) UpdateToolEnd(ctx context.Context, toolUseID, toolResponse string, success bool) error {
	return b.store.UpdateToolEnd(ctx, toolUseID, toolResponse, success)
}

func (b *SQLiteBackend) FindLatestUnfinishedTool(ctx context.Context, sessionID, toolName string) (string, error) {
	return b.store.FindLatestUnfinishedTool(ctx, sessionID, toolName)
}

func (b *SQLiteBackend) BackfillSpansWorkspace(ctx context.Context, sessionID, workspace string) (int64, error) {
	return b.store.BackfillSpansWorkspace(ctx, sessionID, workspace)
}

func (b *SQLiteBackend) GetToolForSpan(ctx context.Context, sessionID, toolName string, spanTime time.Time) (*SessionTool, error) {
	return b.store.GetToolForSpan(ctx, sessionID, toolName, spanTime)
}

// ===== Metric Operations (Claude Code telemetry) =====

func (b *SQLiteBackend) CreateMetric(ctx context.Context, m *Metric) error {
	return b.store.CreateMetric(m)
}

func (b *SQLiteBackend) ListMetrics(ctx context.Context, opts MetricListOptions) ([]*Metric, error) {
	return b.store.ListMetrics(opts)
}

func (b *SQLiteBackend) GetSessionMetricsSummary(ctx context.Context, sessionID string) (*SessionMetricsSummary, error) {
	return b.store.GetSessionMetricsSummary(sessionID)
}

// Chain operations (M-CHAINS-SIMPLIFY)

func (b *SQLiteBackend) CreateChain(ctx context.Context, req *ChainCreateRequest) (*ExecutionChain, error) {
	return b.store.CreateChain(ctx, req)
}

func (b *SQLiteBackend) GetChain(ctx context.Context, id string, opts ChainReadOptions) (*ExecutionChain, error) {
	return b.store.GetChain(ctx, id, opts)
}

func (b *SQLiteBackend) GetChainByMessageID(ctx context.Context, messageID string) (*ExecutionChain, error) {
	return b.store.GetChainByMessageID(ctx, messageID)
}

func (b *SQLiteBackend) GetChainByTaskID(ctx context.Context, taskID string) (*ExecutionChain, error) {
	return b.store.GetChainByTaskID(ctx, taskID)
}

// GetTaskSpanSummary returns aggregated span statistics for a task_id.
// Works for ALL task_id formats (coordinator, eval, UUID sessions).
func (b *SQLiteBackend) GetTaskSpanSummary(ctx context.Context, taskID string) (*TaskSpanSummary, error) {
	return b.store.GetTaskSpanSummary(ctx, taskID)
}

func (b *SQLiteBackend) ListChains(ctx context.Context, opts ChainListOptions) ([]*ChainSummary, error) {
	return b.store.ListChains(ctx, opts)
}

func (b *SQLiteBackend) UpdateChainStatus(ctx context.Context, chainID string, status ChainStatus) error {
	return b.store.UpdateChainStatus(ctx, chainID, status)
}

func (b *SQLiteBackend) UpdateChainMetrics(ctx context.Context, id string, cost float64, tokens, turns int) error {
	return b.store.UpdateChainMetrics(ctx, id, cost, tokens, turns)
}

func (b *SQLiteBackend) GetChainByGitHubIssue(ctx context.Context, repo string, issueNumber int) (*ExecutionChain, error) {
	return b.store.GetChainByGitHubIssue(ctx, repo, issueNumber)
}

func (b *SQLiteBackend) GetChainStats(ctx context.Context) (*ChainStats, error) {
	return b.store.GetChainStats(ctx)
}

func (b *SQLiteBackend) GetChainStatusCounts(ctx context.Context, createdAfter *time.Time) (*ChainStatusCounts, error) {
	return b.store.GetChainStatusCounts(ctx, createdAfter)
}

func (b *SQLiteBackend) GetChainStatsByAgent(ctx context.Context, createdAfter *time.Time) ([]*AgentStatsResult, error) {
	return b.store.GetChainStatsByAgent(ctx, createdAfter)
}

func (b *SQLiteBackend) CreateStage(ctx context.Context, req *StageCreateRequest) (*ChainStage, error) {
	return b.store.CreateStage(ctx, req)
}

func (b *SQLiteBackend) GetStage(ctx context.Context, id string) (*ChainStage, error) {
	return b.store.GetStage(ctx, id)
}

func (b *SQLiteBackend) GetChainStages(ctx context.Context, chainID string, opts ChainReadOptions) ([]*ChainStage, error) {
	return b.store.GetChainStages(ctx, chainID, opts)
}

func (b *SQLiteBackend) UpdateStageStatus(ctx context.Context, stageID string, status ChainStageStatus) error {
	return b.store.UpdateStageStatus(ctx, stageID, status)
}

func (b *SQLiteBackend) UpdateStageSession(ctx context.Context, stageID, sessionID string) error {
	return b.store.UpdateStageSession(ctx, stageID, sessionID)
}

func (b *SQLiteBackend) UpdateStageMetrics(ctx context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64) error {
	return b.store.UpdateStageMetrics(ctx, stageID, cost, tokensIn, tokensOut, turns, toolCalls, durationMs)
}

func (b *SQLiteBackend) UpdateStageApproval(ctx context.Context, stageID string, status ApprovalStatus, approvalType ApprovalType, feedback string) error {
	return b.store.UpdateStageApproval(ctx, stageID, status, approvalType, feedback)
}

func (b *SQLiteBackend) UpdateStageError(ctx context.Context, stageID, errorMessage string) error {
	return b.store.UpdateStageError(ctx, stageID, errorMessage)
}

func (b *SQLiteBackend) GetSpanLitesByStageID(ctx context.Context, stageID string, limit, offset int) (*SpanLitePage, error) {
	return b.store.GetSpanLitesByStageID(ctx, stageID, limit, offset)
}

func (b *SQLiteBackend) GetSpansByStageID(ctx context.Context, stageID string) ([]*Span, error) {
	return b.store.GetSpansByStageID(ctx, stageID)
}

func (b *SQLiteBackend) LinkSpanToChain(ctx context.Context, spanID, chainID, stageID string) error {
	return b.store.LinkSpanToChain(ctx, spanID, chainID, stageID)
}

func (b *SQLiteBackend) ListPendingApprovals(ctx context.Context, limit int) ([]*PendingApprovalInfo, error) {
	return b.store.ListPendingApprovals(ctx, limit)
}

// Session detail operations (M-CHAINS-SOURCE-OF-TRUTH)

func (b *SQLiteBackend) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	return b.store.GetSession(ctx, sessionID)
}

func (b *SQLiteBackend) GetSessionTools(ctx context.Context, sessionID string) ([]SessionTool, error) {
	return b.store.GetSessionTools(ctx, sessionID)
}

// Chat message operations (M-CHAINS-SOURCE-OF-TRUTH)

func (b *SQLiteBackend) GetChatMessagesByTaskID(ctx context.Context, taskID string) ([]*ChatMessage, error) {
	return b.store.GetChatMessagesByTaskID(ctx, taskID)
}

func (b *SQLiteBackend) GetChatMessagesBySession(ctx context.Context, sessionID string, startTime, endTime time.Time) ([]*ChatMessage, error) {
	return b.store.GetChatMessagesBySession(ctx, sessionID, startTime, endTime)
}

func (b *SQLiteBackend) CountChatMessages(ctx context.Context, q ChatMessageQuery) (int, int, error) {
	return b.store.CountChatMessages(ctx, q)
}

// Ensure SQLiteBackend implements Backend
var _ Backend = (*SQLiteBackend)(nil)
