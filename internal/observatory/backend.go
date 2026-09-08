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
	// LookupChainBySessionID resolves a chain and stage from a session id
	// (M-MISSION-LOOP-UNIFIED-TELEMETRY M1).
	//
	// OpenRouter Broadcast delivers our chain id as `session.id`, so this is how
	// a provider-side trace joins back to the run that caused it. Mirrors
	// LookupTaskBySessionID's contract deliberately: an unknown session returns
	// empties rather than an error, because not every session has a chain and
	// that is ordinary, not a fault.
	LookupChainBySessionID(ctx context.Context, sessionID string) (chainID, stageID string)

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
	// UpsertSessionWithCorrelation inserts/updates a session with correlation IDs (M-DETERMINISTIC-CHAT-LINKING)
	UpsertSessionWithCorrelation(ctx context.Context, sessionID, workspace, version, source string, corr *SessionCorrelation) error
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
	// GetExecTaskHierarchyWithMessages returns hierarchy grouped by triggering messages (4-level)
	GetExecTaskHierarchyWithMessages(ctx context.Context, limit int) (*ExecHierarchyWithMessages, error)
	// GetSpanHierarchy returns hierarchical tree of spans using parent_span_id relationships
	GetSpanHierarchy(ctx context.Context, limit int) (*SpanHierarchyResult, error)
	// GetToolsByTimestampRange returns session tools within a time range for enrichment
	GetToolsByTimestampRange(ctx context.Context, start, end time.Time, toolName string) ([]SessionTool, error)

	// Metric operations (Claude Code telemetry metrics)
	CreateMetric(ctx context.Context, m *Metric) error
	ListMetrics(ctx context.Context, opts MetricListOptions) ([]*Metric, error)
	GetSessionMetricsSummary(ctx context.Context, sessionID string) (*SessionMetricsSummary, error)

	// Session detail operations (M-CHAINS-SOURCE-OF-TRUTH)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	GetSessionTools(ctx context.Context, sessionID string) ([]SessionTool, error)

	// Chat message operations (M-CHAINS-SOURCE-OF-TRUTH)
	GetChatMessagesByTaskID(ctx context.Context, taskID string) ([]*ChatMessage, error)
	GetChatMessagesBySession(ctx context.Context, sessionID string, startTime, endTime time.Time) ([]*ChatMessage, error)
	CountChatMessages(ctx context.Context, q ChatMessageQuery) (total int, withTaskID int, err error)

	// Chain operations (M-CHAINS-SIMPLIFY unified hierarchy)
	CreateChain(ctx context.Context, req *ChainCreateRequest) (*ExecutionChain, error)
	GetChain(ctx context.Context, id string, opts ChainReadOptions) (*ExecutionChain, error)
	GetChainByMessageID(ctx context.Context, messageID string) (*ExecutionChain, error)
	GetChainByTaskID(ctx context.Context, taskID string) (*ExecutionChain, error)
	GetChainByGitHubIssue(ctx context.Context, repo string, issueNumber int) (*ExecutionChain, error)
	ListChains(ctx context.Context, opts ChainListOptions) ([]*ChainSummary, error)
	UpdateChainStatus(ctx context.Context, chainID string, status ChainStatus) error
	UpdateChainMetrics(ctx context.Context, id string, cost float64, tokens, turns int) error
	GetChainStats(ctx context.Context) (*ChainStats, error)
	// GetChainStatusCounts returns chain counts grouped by status (single query, M-PERF-OBSERVATORY).
	GetChainStatusCounts(ctx context.Context, createdAfter *time.Time) (*ChainStatusCounts, error)
	// GetChainStatsByAgent returns per-agent stats in a single SQL query (replaces N+1, M-PERF-OBSERVATORY).
	GetChainStatsByAgent(ctx context.Context, createdAfter *time.Time) ([]*AgentStatsResult, error)
	// GetCostRollup returns split reported/estimated/quota/unknown cost totals via a
	// Go per-stage classification pass (M-MISSION-COST-CHAINS M1). sourcePrefix filters
	// chains by source_ref prefix ("" = all).
	GetCostRollup(ctx context.Context, createdAfter *time.Time, sourcePrefix string) (CostRollup, error)
	// GetMissionRollups groups chains by mission (source_ref prefix) and returns
	// per-mission split totals, per-bucket quota counts, and top-N stages (M3).
	GetMissionRollups(ctx context.Context, createdAfter *time.Time, sourcePrefix string, topN int) ([]MissionRollup, error)

	// Chain stage operations
	CreateStage(ctx context.Context, req *StageCreateRequest) (*ChainStage, error)
	GetStage(ctx context.Context, id string) (*ChainStage, error)
	GetChainStages(ctx context.Context, chainID string, opts ChainReadOptions) ([]*ChainStage, error)
	UpdateStageStatus(ctx context.Context, stageID string, status ChainStageStatus) error

	// Idempotent finalisation writes (M-COMPLETION-PATH-PARITY M0b).
	//
	// Task finalisation is replayed — Pub/Sub push is at-least-once — so it needs
	// writes that are safe to apply twice. The Update* family above is not: the
	// metrics accumulate, UpdateStageStatus increments the chain's
	// stages_completed counter, and UpdateStageError increments error_count.
	// These absolute-write counterparts exist for that path; the accumulating
	// ones keep their semantics for importers and evaluators.
	SetStageStatus(ctx context.Context, stageID string, status ChainStageStatus) error
	SetStageMetrics(ctx context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64, costProvenance string) error
	SetStageError(ctx context.Context, stageID, errorMessage string) error
	// RecomputeChainAggregates derives a chain's totals and stages_completed from
	// its stage rows, so the value does not depend on who writes it or how often.
	RecomputeChainAggregates(ctx context.Context, chainID string) error

	// Chain reconciliation (M-COMPLETION-PATH-PARITY M4). A chain whose stages
	// have all finished but which is still "active" can never progress; it is
	// marked abandoned with a reason rather than given a verdict nobody observed.
	FindStrandedChains(ctx context.Context, minAge time.Duration) ([]StrandedChain, error)
	AbandonChain(ctx context.Context, chainID, reason string) error
	UpdateStageSession(ctx context.Context, stageID, sessionID string) error
	UpdateStageApproval(ctx context.Context, stageID string, status ApprovalStatus, approvalType ApprovalType, feedback string) error
	// UpdateStageMetrics accumulates a stage's denormalized metrics.
	// costProvenance labels whether `cost` was actually billed (see
	// executor.CostProvenance); pass "" when the caller cannot classify it —
	// that reads as unknown and never overwrites a label already recorded.
	UpdateStageMetrics(ctx context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64, costProvenance string) error
	UpdateStageError(ctx context.Context, stageID, errorMessage string) error
	// UpdateStageEvalAssessment records the structured assessment (model, benchmark,
	// eval mode) a stage ran under. On the interface because a mission iteration
	// posted to a REMOTE observatory must carry its per-stage model too — without it
	// the cost classifier cannot resolve a rate cloud-side (M-MISSION-LOOP-UNIFIED-
	// TELEMETRY M3).
	UpdateStageEvalAssessment(ctx context.Context, stageID string, assessment *EvalAssessment) error
	// UpdateStageQuotaTokens records SUBSCRIPTION token spend, which cannot ride on
	// UpdateStageMetrics: the cost estimator reads `tokens > 0` as "metered", so a
	// quota lane's real count in tokens_in/out would be priced as if it were billed
	// (M-QUOTA-RATIONING-ROUTING M2). On the interface for the same reason as
	// UpdateStageEvalAssessment — a remote-posted iteration must carry it too.
	UpdateStageQuotaTokens(ctx context.Context, stageID string, tokens int64) error
	GetSpansByStageID(ctx context.Context, stageID string) ([]*Span, error)
	// GetSpanLitesByStageID returns lightweight spans without attributes (M-PERF-OBSERVATORY).
	GetSpanLitesByStageID(ctx context.Context, stageID string, limit, offset int) (*SpanLitePage, error)
	LinkSpanToChain(ctx context.Context, spanID, chainID, stageID string) error
	ListPendingApprovals(ctx context.Context, limit int) ([]*PendingApprovalInfo, error)

	// Lifecycle
	Close() error
}
