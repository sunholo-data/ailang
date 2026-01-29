package coordinator

import (
	"context"
	"time"
)

// TaskRecord represents a task stored in the database
type TaskRecord struct {
	ID           string     `json:"id"`
	MessageID    string     `json:"message_id,omitempty"`
	ThreadID     string     `json:"thread_id,omitempty"`      // Thread in collaboration.db for dashboard visibility
	ParentTaskID string     `json:"parent_task_id,omitempty"` // Parent task for hierarchy tracking (handoffs)
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Type         TaskType   `json:"type"`
	Kind         string     `json:"kind,omitempty"` // "directive" or "question" - affects execution mode
	Priority     int        `json:"priority"`
	Status       TaskStatus `json:"status"`
	Provider     string     `json:"provider,omitempty"`
	AgentID      string     `json:"agent_id,omitempty"` // ID of agent that processed this task
	WorktreeID   string     `json:"worktree_id,omitempty"`
	WorktreePath string     `json:"worktree_path,omitempty"` // Path to git worktree (preserved until approval)
	BaseBranch   string     `json:"base_branch,omitempty"`   // Base branch worktree was created from (for diff comparison)
	BaseCommit   string     `json:"base_commit,omitempty"`   // Base commit hash at worktree creation (stable reference for diff)
	SessionID    string     `json:"session_id,omitempty"`    // Claude Code/Gemini CLI session for resumption
	Iteration    int        `json:"iteration,omitempty"`     // Iteration number (1 = first, 2+ = re-run with feedback)
	Workspace    string     `json:"workspace,omitempty"`     // Source workspace from thread (not worktree)
	// Execution chain tracking (M-CHAINS-SIMPLIFY)
	ChainID string `json:"chain_id,omitempty"` // ExecutionChain ID for unified hierarchy
	StageID string `json:"stage_id,omitempty"` // ChainStage ID for this agent's execution
	// GitHub integration (M-COORD-GITHUB-AUTO-ROUTING)
	GithubIssue    int       `json:"github_issue,omitempty"`     // Linked GitHub issue number
	GithubRepo     string    `json:"github_repo,omitempty"`      // GitHub repo (owner/repo) for issue operations
	Stage          TaskStage `json:"stage,omitempty"`            // Pipeline stage (design, sprint, implementation, merge)
	DesignDocPath  string    `json:"design_doc_path,omitempty"`  // Path to design doc (for merge comment)
	SprintPlanPath string    `json:"sprint_plan_path,omitempty"` // Path to sprint plan (for merge comment)
	// Timestamps
	CreatedAt   time.Time     `json:"created_at"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Error       string        `json:"error,omitempty"`
	Output      string        `json:"output,omitempty"`
	Cost        float64       `json:"cost,omitempty"`
	TokensUsed  int           `json:"tokens_used,omitempty"`
	// Detailed token breakdown
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	// Resource metrics
	PeakCPU    float64 `json:"peak_cpu,omitempty"`
	PeakMemory float64 `json:"peak_memory_mb,omitempty"`
	// Capability detection (M-DEPRECATE-AILANG-AGENT)
	Capabilities  []Capability `json:"capabilities,omitempty"`   // Detected capability requirements
	ImpactLevel   string       `json:"impact_level,omitempty"`   // "low", "medium", or "high"
	EstimatedCost float64      `json:"estimated_cost,omitempty"` // Pre-execution cost estimate in USD
}

// TaskStatus represents the lifecycle state of a task
type TaskStatus string

const (
	TaskStatusPending         TaskStatus = "pending"
	TaskStatusQueued          TaskStatus = "queued"
	TaskStatusRunning         TaskStatus = "running"
	TaskStatusPendingApproval TaskStatus = "pending_approval" // Work done, awaiting human review
	TaskStatusCompleted       TaskStatus = "completed"        // Approved and merged
	TaskStatusFailed          TaskStatus = "failed"
	TaskStatusRejected        TaskStatus = "rejected" // Human rejected the work
	TaskStatusCancelled       TaskStatus = "cancelled"
	TaskStatusDuplicate       TaskStatus = "duplicate"
)

// TaskStage represents the pipeline stage for GitHub-linked tasks.
//
// Deprecated: Use agent_id tracking with trigger_on_complete configuration instead.
// Stage is maintained for backwards compatibility with existing tasks.
// Will be removed in v0.9.0. See M-GENERIC-PIPELINE design doc for migration guidance.
//
// Migration guide:
//   - Replace TaskStageDesign with agent_id = "design-doc-creator"
//   - Replace TaskStageSprint with agent_id = "sprint-planner"
//   - Replace TaskStageImplementation with agent_id = "sprint-executor"
//   - Configure trigger_on_complete in ~/.ailang/config.yaml for automatic handoffs
type TaskStage string

const (
	// Deprecated: Use agent_id tracking instead
	TaskStageNone TaskStage = "" // Not part of a pipeline
	// Deprecated: Use agent_id = "design-doc-creator" with approval config instead
	TaskStageDesign TaskStage = "design" // Creating design document
	// Deprecated: Use agent_id = "sprint-planner" with approval config instead
	TaskStageSprint TaskStage = "sprint" // Creating sprint plan
	// Deprecated: Use agent_id = "sprint-executor" with approval config instead
	TaskStageImplementation TaskStage = "implementation" // Implementing the sprint
	// Deprecated: Use agent_id with approval config for merge workflow instead
	TaskStageMerge TaskStage = "merge" // Awaiting merge approval
)

// TaskFilter for querying tasks
type TaskFilter struct {
	Status    []TaskStatus
	Type      []TaskType
	Provider  string
	Workspace string // Filter by source workspace
	Since     *time.Time
	Until     *time.Time
	Limit     int
	Offset    int
	OrderBy   string // "created_at", "priority", "started_at"
	OrderDesc bool
}

// DetailedStats provides cost/token breakdown for a provider or workspace
type DetailedStats struct {
	Count        int     `json:"count"`
	CostUSD      float64 `json:"cost_usd"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
}

// TaskStats provides aggregate statistics
type TaskStats struct {
	TotalTasks       int                       `json:"total_tasks"`
	PendingTasks     int                       `json:"pending_tasks"`
	RunningTasks     int                       `json:"running_tasks"`
	PendingApprovals int                       `json:"pending_approvals"` // Tasks awaiting human approval
	CompletedTasks   int                       `json:"completed_tasks"`
	FailedTasks      int                       `json:"failed_tasks"`
	ByType           map[string]int            `json:"by_type"`
	ByProvider       map[string]*DetailedStats `json:"by_provider"`
	ByWorkspace      map[string]*DetailedStats `json:"by_workspace"` // Per-workspace breakdown
	TotalCost        float64                   `json:"total_cost"`
	TotalTokens      int                       `json:"total_tokens"`
	AvgDuration      time.Duration             `json:"avg_duration"`
}

// TaskEventRecord represents a stored task streaming event
type TaskEventRecord struct {
	ID          int64     `json:"id"`
	TaskID      string    `json:"task_id"`
	ThreadID    string    `json:"thread_id,omitempty"`
	StreamType  string    `json:"stream_type"` // "text", "tool_use", "tool_result", "error", "status", "turn_start", "turn_end"
	TurnNum     int       `json:"turn_num,omitempty"`
	Text        string    `json:"text,omitempty"`
	ToolName    string    `json:"tool_name,omitempty"`
	ToolInput   string    `json:"tool_input,omitempty"`
	ToolOutput  string    `json:"tool_output,omitempty"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
	Status      string    `json:"status,omitempty"`
	TokensIn    int       `json:"tokens_in,omitempty"`
	TokensOut   int       `json:"tokens_out,omitempty"`
	Cost        float64   `json:"cost,omitempty"`
	DurationSec int       `json:"duration_sec,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Store is the neutral interface for task persistence.
// Implementations can be SQLite (local), PostgreSQL, or cloud services.
type Store interface {
	// Task CRUD operations
	CreateTask(ctx context.Context, task *TaskRecord) error
	GetTask(ctx context.Context, id string) (*TaskRecord, error)
	UpdateTask(ctx context.Context, task *TaskRecord) error
	DeleteTask(ctx context.Context, id string) error

	// Task queries
	ListTasks(ctx context.Context, filter *TaskFilter) ([]*TaskRecord, error)
	GetTaskStats(ctx context.Context) (*TaskStats, error)

	// Task state transitions
	MarkTaskQueued(ctx context.Context, id string) error
	MarkTaskRunning(ctx context.Context, id, provider, worktreeID string) error
	MarkTaskPendingApproval(ctx context.Context, id, worktreePath, worktreeBranch, baseBranch, baseCommit string, result *ExecuteResult) error // Work done, awaiting human review
	MarkTaskCompleted(ctx context.Context, id string, result *ExecuteResult) error
	MarkTaskFailed(ctx context.Context, id string, err error) error
	MarkTaskRejected(ctx context.Context, id string) error // Human rejected the work
	MarkTaskCancelled(ctx context.Context, id string) error
	RequeueTask(ctx context.Context, id string) error        // Reset status to pending for next stage execution
	ResetTaskToPending(ctx context.Context, id string) error // Reset running task back to pending (worktree limit recovery)

	// Duplicate detection
	FindDuplicateTask(ctx context.Context, fingerprint uint64, threshold float64) (*TaskRecord, error)
	SetTaskFingerprint(ctx context.Context, id string, fingerprint uint64) error

	// Thread linking (for dashboard visibility)
	SetTaskThreadID(ctx context.Context, id string, threadID string) error

	// Execution chain tracking (M-CHAINS-SIMPLIFY)
	UpdateTaskChainInfo(ctx context.Context, id, chainID, stageID string) error

	// Cross-database correlation (for Control Plane event classification)
	// Returns agent info for a task: agentID (FromAgent), inbox (ToInbox), title
	// Note: By convention, agent id == inbox in agent config
	GetTaskAgentInfo(ctx context.Context, taskID string) (agentID, inbox, title string, err error)

	// GitHub integration (M-COORD-GITHUB-AUTO-ROUTING)
	SetTaskGithubIssue(ctx context.Context, id string, issueNum int) error
	SetTaskStage(ctx context.Context, id string, stage TaskStage) error
	SetTaskDesignDocPath(ctx context.Context, id string, path string) error
	SetTaskSprintPlanPath(ctx context.Context, id string, path string) error
	GetTasksByGithubIssue(ctx context.Context, issueNum int) ([]*TaskRecord, error)
	GetTasksByStage(ctx context.Context, stage TaskStage) ([]*TaskRecord, error)

	// Resource metrics
	UpdateTaskMetrics(ctx context.Context, id string, peakCPU, peakMemory float64) error

	// Budget tracking (per-provider)
	GetCostByProvider() (map[string]float64, error)

	// Approval requests
	CreateApprovalRequest(ctx context.Context, req *ApprovalRequestRecord) error
	GetApprovalRequestByTask(ctx context.Context, taskID string) (*ApprovalRequestRecord, error)
	ListPendingApprovals(ctx context.Context) ([]*ApprovalRequestRecord, error)
	ResolveApprovalRequest(ctx context.Context, id, status, resolvedBy string) error
	ResolveApprovalRequestByTask(ctx context.Context, taskID, status, resolvedBy string) error
	MarkApprovalHandoffsTriggered(ctx context.Context, taskID string) error                        // Mark that handoffs were sent
	ListApprovedMergeHandoffsWithoutTrigger(ctx context.Context) ([]*ApprovalRequestRecord, error) // Find missed handoffs

	// Cleanup
	DeleteOldTasks(ctx context.Context, olderThan time.Duration) (int, error)
	RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration) (int, error) // Cancel stale running/queued tasks on startup
	RetryAllFailedTasks(ctx context.Context) (int, error)                             // Reset all failed tasks to pending

	// Event storage (for task logs)
	StoreTaskEvent(ctx context.Context, event *TaskEventRecord) error
	GetTaskEvents(ctx context.Context, taskID string, limit int) ([]*TaskEventRecord, error)

	// Lifecycle
	Close() error
}

// StoreConfig configures store creation
type StoreConfig struct {
	// For SQLite
	DBPath string

	// For cloud stores (future)
	Endpoint  string
	APIKey    string
	ProjectID string
	Region    string
}

// StoreType identifies the storage backend
type StoreType string

const (
	StoreTypeSQLite StoreType = "sqlite"
	StoreTypeCloud  StoreType = "cloud" // Future: could be Firestore, DynamoDB, etc.
)

// NewStore creates a store based on the type
func NewStore(storeType StoreType, cfg *StoreConfig) (Store, error) {
	switch storeType {
	case StoreTypeSQLite:
		return NewSQLiteStore(cfg.DBPath)
	case StoreTypeCloud:
		return NewCloudStore(cfg)
	default:
		return NewSQLiteStore(cfg.DBPath) // Default to SQLite
	}
}
