package observatory

import (
	"time"
)

// ChainStatus represents the status of an execution chain.
type ChainStatus string

const (
	ChainStatusActive          ChainStatus = "active"
	ChainStatusPendingApproval ChainStatus = "pending_approval"
	ChainStatusCompleted       ChainStatus = "completed"
	ChainStatusFailed          ChainStatus = "failed"
)

// ChainSourceType represents the origin of a chain.
type ChainSourceType string

const (
	ChainSourceGitHubIssue ChainSourceType = "github_issue"
	ChainSourceMessage     ChainSourceType = "message"
	ChainSourceManual      ChainSourceType = "manual"
	ChainSourceEvalSuite   ChainSourceType = "eval_suite"
)

// ExecutionChain represents a complete execution flow from source to completion.
// This is the top-level hierarchy: Issue -> Message -> Task -> Session -> Traces
type ExecutionChain struct {
	ID string `json:"id"` // UUID

	// Source (what triggered this chain)
	SourceType        ChainSourceType `json:"source_type"`
	SourceRef         string          `json:"source_ref,omitempty"`
	GitHubRepo        string          `json:"github_repo,omitempty"`
	GitHubIssueNumber int             `json:"github_issue_number,omitempty"`

	// Current state
	Status       ChainStatus `json:"status"`
	CurrentStage int         `json:"current_stage"`

	// Workspace context
	WorkspaceID   string `json:"workspace_id,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Summary metrics (denormalized for quick queries)
	TotalCost       float64 `json:"total_cost"`
	TotalTokens     int     `json:"total_tokens"`
	TotalTurns      int     `json:"total_turns"`
	StagesCompleted int     `json:"stages_completed"`

	// Stages (populated on read)
	Stages []*ChainStage `json:"stages,omitempty"`
}

// ChainStageStatus represents the status of a chain stage.
type ChainStageStatus string

const (
	StageStatusPending          ChainStageStatus = "pending"
	StageStatusRunning          ChainStageStatus = "running"
	StageStatusAwaitingApproval ChainStageStatus = "awaiting_approval"
	StageStatusCompleted        ChainStageStatus = "completed"
	StageStatusFailed           ChainStageStatus = "failed"
)

// ApprovalType represents the type of approval required.
type ApprovalType string

const (
	ApprovalTypeMerge        ApprovalType = "merge"
	ApprovalTypeHandoff      ApprovalType = "handoff"
	ApprovalTypeMergeHandoff ApprovalType = "merge_handoff"
)

// ChainStage represents a single agent execution within a chain.
type ChainStage struct {
	ID          string `json:"id"` // UUID
	ChainID     string `json:"chain_id"`
	StageNumber int    `json:"stage_number"`

	// Agent info
	AgentID  string   `json:"agent_id"`
	Provider Provider `json:"provider,omitempty"`

	// Links to other tables (foreign keys or cross-DB IDs)
	MessageID string `json:"message_id,omitempty"` // -> collaboration.db:inbox_messages.id
	TaskID    string `json:"task_id,omitempty"`    // -> coordinator.db:tasks.id
	SessionID string `json:"session_id,omitempty"` // -> sessions.session_id

	// State
	Status         ChainStageStatus `json:"status"`
	ApprovalStatus ApprovalStatus   `json:"approval_status,omitempty"`
	ApprovalType   ApprovalType     `json:"approval_type,omitempty"`

	// Handoff info
	HandoffTo     string `json:"handoff_to,omitempty"`
	Iteration     int    `json:"iteration"`
	HumanFeedback string `json:"human_feedback,omitempty"`

	// Timestamps
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Summary (denormalized from spans)
	Cost       float64 `json:"cost"`
	TokensIn   int     `json:"tokens_in"`
	TokensOut  int     `json:"tokens_out"`
	Turns      int     `json:"turns"`
	ToolCalls  int     `json:"tool_calls"`
	DurationMs int64   `json:"duration_ms"`

	// Error tracking
	ErrorMessage string `json:"error_message,omitempty"`
	ErrorCount   int    `json:"error_count"`

	// Eval assessment (M-EVAL-CHAINS: populated for eval_suite chains)
	EvalAssessment *EvalAssessment `json:"eval_assessment,omitempty"`

	// Session data (populated on read with full=true)
	Session *Session `json:"session,omitempty"`

	// Spans (populated on read with include_spans=true)
	Spans []*Span `json:"spans,omitempty"`
}

// EvalAssessment stores structured evaluation results for agent benchmark stages.
// Stored as JSON in chain_stages.eval_assessment column (M-EVAL-CHAINS).
type EvalAssessment struct {
	// Identity
	BenchmarkID string `json:"benchmark_id"`
	Model       string `json:"model"`
	Language    string `json:"language"`
	Condition   string `json:"condition,omitempty"`
	EvalMode    string `json:"eval_mode"`          // "agent" or "standard"
	Executor    string `json:"executor,omitempty"` // "claude", "gemini"
	Seed        int64  `json:"seed"`

	// Assessment results
	CompileOk     bool   `json:"compile_ok"`
	RuntimeOk     bool   `json:"runtime_ok"`
	StdoutOk      bool   `json:"stdout_ok"`
	ErrorCategory string `json:"error_category"`

	// Self-repair
	FirstAttemptOk bool   `json:"first_attempt_ok"`
	RepairUsed     bool   `json:"repair_used"`
	RepairOk       bool   `json:"repair_ok"`
	ErrCode        string `json:"err_code,omitempty"`

	// Contract verification
	VerifyOk        bool `json:"verify_ok"`
	VerifyVerified  int  `json:"verify_verified"`
	VerifyCounterex int  `json:"verify_counterexample"`
	VerifySkipped   int  `json:"verify_skipped"`
	VerifyErrors    int  `json:"verify_errors"`

	// Reproducibility
	PromptVersion string `json:"prompt_version,omitempty"`
	CodeHash      string `json:"code_hash,omitempty"`

	// Output (truncated for storage efficiency)
	Code           string `json:"code,omitempty"`
	Stdout         string `json:"stdout,omitempty"`
	ExpectedStdout string `json:"expected_stdout,omitempty"`
	Stderr         string `json:"stderr,omitempty"`
}

// EvalQueryOptions specifies filters for querying eval assessment stages.
type EvalQueryOptions struct {
	ChainID     string // Filter by chain
	Model       string // Filter by model
	Language    string // Filter by language
	BenchmarkID string // Filter by benchmark
	Condition   string // Filter by experimental condition
	EvalMode    string // "standard" or "agent"
	SuccessOnly bool   // Only passing benchmarks (stdout_ok = true)
	FailureOnly bool   // Only failing benchmarks (stdout_ok = false)
	Limit       int
}

// ChainListOptions specifies filters for listing chains.
type ChainListOptions struct {
	Status       ChainStatus `json:"status,omitempty"`
	SourceType   string      `json:"source_type,omitempty"`
	WorkspaceID  string      `json:"workspace_id,omitempty"`
	GitHubRepo   string      `json:"github_repo,omitempty"`
	AgentID      string      `json:"agent_id,omitempty"`
	CreatedAfter *time.Time  `json:"created_after,omitempty"`
	Limit        int         `json:"limit,omitempty"`
	Offset       int         `json:"offset,omitempty"`
}

// ChainWithStages returns the chain with all its stages populated.
// Options control what additional data is fetched.
type ChainReadOptions struct {
	IncludeStages   bool `json:"include_stages"`   // Fetch stages
	IncludeSpans    bool `json:"include_spans"`    // Fetch spans for each stage
	IncludeSessions bool `json:"include_sessions"` // Fetch session data for each stage
}

// DefaultChainReadOptions returns options for a full chain read.
func DefaultChainReadOptions() ChainReadOptions {
	return ChainReadOptions{
		IncludeStages:   true,
		IncludeSpans:    true,
		IncludeSessions: true,
	}
}

// ChainSummary is a lightweight view of a chain for list views.
type ChainSummary struct {
	ID                string      `json:"id"`
	SourceType        string      `json:"source_type"`
	SourceRef         string      `json:"source_ref"`
	GitHubRepo        string      `json:"github_repo,omitempty"`
	GitHubIssueNumber int         `json:"github_issue_number,omitempty"`
	Status            ChainStatus `json:"status"`
	CurrentStage      int         `json:"current_stage"`
	TotalCost         float64     `json:"total_cost"`
	TotalTokens       int         `json:"total_tokens"`
	TotalTurns        int         `json:"total_turns"`
	StagesCompleted   int         `json:"stages_completed"`
	CreatedAt         time.Time   `json:"created_at"`
	CompletedAt       *time.Time  `json:"completed_at,omitempty"`
	StageCount        int         `json:"stage_count"`
	MaxStage          int         `json:"max_stage"`
	AgentFlow         string      `json:"agent_flow"` // e.g., "design-doc-creator -> sprint-planner -> sprint-executor"
}

// PendingApprovalInfo contains approval info for dashboard views.
type PendingApprovalInfo struct {
	ChainID        string       `json:"chain_id"`
	SourceType     string       `json:"source_type"`
	SourceRef      string       `json:"source_ref"`
	StageID        string       `json:"stage_id"`
	StageNumber    int          `json:"stage_number"`
	AgentID        string       `json:"agent_id"`
	ApprovalStatus string       `json:"approval_status"`
	ApprovalType   ApprovalType `json:"approval_type"`
	TaskID         string       `json:"task_id"`
	SessionID      string       `json:"session_id"`
	Cost           float64      `json:"cost"`
	Turns          int          `json:"turns"`
	StageCreated   time.Time    `json:"stage_created"`
}

// NOTE: Session and SessionTool types are defined in store_sessions.go
// ChainStage.Session uses that existing type

// ChainStats contains aggregate statistics about chains.
type ChainStats struct {
	TotalChains        int     `json:"total_chains"`
	ActiveChains       int     `json:"active_chains"`
	PendingApprovals   int     `json:"pending_approvals"`
	CompletedChains    int     `json:"completed_chains"`
	FailedChains       int     `json:"failed_chains"`
	TotalCost          float64 `json:"total_cost"`
	TotalTokens        int64   `json:"total_tokens"`
	AverageStagesCount float64 `json:"average_stages_count"`
	AverageDurationMs  float64 `json:"average_duration_ms"`
}

// ChainCreateRequest contains the data needed to create a new chain.
type ChainCreateRequest struct {
	SourceType        ChainSourceType `json:"source_type"`
	SourceRef         string          `json:"source_ref,omitempty"`
	GitHubRepo        string          `json:"github_repo,omitempty"`
	GitHubIssueNumber int             `json:"github_issue_number,omitempty"`
	WorkspaceID       string          `json:"workspace_id,omitempty"`
	WorkspacePath     string          `json:"workspace_path,omitempty"`
}

// StageCreateRequest contains the data needed to create a new stage in a chain.
type StageCreateRequest struct {
	ChainID   string   `json:"chain_id"`
	AgentID   string   `json:"agent_id"`
	Provider  Provider `json:"provider,omitempty"`
	MessageID string   `json:"message_id,omitempty"`
	TaskID    string   `json:"task_id,omitempty"`
	HandoffTo string   `json:"handoff_to,omitempty"`
	Iteration int      `json:"iteration,omitempty"` // Defaults to 1
}

// StageUpdateRequest contains fields that can be updated on a stage.
type StageUpdateRequest struct {
	Status         *ChainStageStatus `json:"status,omitempty"`
	SessionID      *string           `json:"session_id,omitempty"`
	ApprovalStatus *ApprovalStatus   `json:"approval_status,omitempty"`
	ApprovalType   *ApprovalType     `json:"approval_type,omitempty"`
	HumanFeedback  *string           `json:"human_feedback,omitempty"`
	ErrorMessage   *string           `json:"error_message,omitempty"`
	// Metrics are typically updated by aggregation, not direct update
}

// ChainEnvironmentVars returns the environment variables to pass to executors
// for write-time linking of spans to this chain/stage.
func (c *ExecutionChain) ChainEnvironmentVars(stageID, taskID, messageID string) map[string]string {
	return map[string]string{
		"AILANG_CHAIN_ID":   c.ID,
		"AILANG_STAGE_ID":   stageID,
		"AILANG_TASK_ID":    taskID,
		"AILANG_MESSAGE_ID": messageID,
	}
}
