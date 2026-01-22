// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"time"
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

	// Session operations (M-SESSION-WORKSPACE-HOOKS)
	// GetSessionWorkspace returns workspace for a session (for span enrichment)
	GetSessionWorkspace(sessionID string) (string, error)
	// UpsertSession inserts or updates a session record from hook data
	UpsertSession(ctx context.Context, sessionID, workspace, version, source string) error
	// UpdateSessionEnded marks a session as ended
	UpdateSessionEnded(ctx context.Context, sessionID string) error
	// InsertToolStart records the start of a tool call
	InsertToolStart(ctx context.Context, sessionID, toolUseID, toolName, toolInput string) error
	// FindLatestUnfinishedTool finds the most recent tool call that hasn't completed yet
	// Used to correlate PostToolUse with PreToolUse when tool_use_id is not provided
	FindLatestUnfinishedTool(ctx context.Context, sessionID, toolName string) (string, error)
	// UpdateToolEnd records the completion of a tool call
	UpdateToolEnd(ctx context.Context, toolUseID, toolResponse string, success bool) error
	// GetToolForSpan finds the session_tool that best matches a span by timestamp + tool name
	GetToolForSpan(ctx context.Context, sessionID, toolName string, spanTime time.Time) (*SessionTool, error)
	// BackfillSpansWorkspace updates existing spans that have session.id but missing workspace
	BackfillSpansWorkspace(ctx context.Context, sessionID, workspace string) (int64, error)

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
	// GetExecTaskHierarchy returns hierarchy of ailang command spans (exec, run, check)
	GetExecTaskHierarchy(ctx context.Context, limit int) ([]*ExecTaskNode, error)

	// Metric operations (Claude Code telemetry metrics)
	CreateMetric(ctx context.Context, m *Metric) error
	ListMetrics(ctx context.Context, opts MetricListOptions) ([]*Metric, error)
	GetSessionMetricsSummary(ctx context.Context, sessionID string) (*SessionMetricsSummary, error)

	// Lifecycle
	Close() error
}
