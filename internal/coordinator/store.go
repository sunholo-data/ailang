package coordinator

import (
	"context"
	"time"
)

// TaskRecord represents a task stored in the database
type TaskRecord struct {
	ID          string        `json:"id"`
	MessageID   string        `json:"message_id,omitempty"`
	ThreadID    string        `json:"thread_id,omitempty"` // Thread in collaboration.db for dashboard visibility
	Title       string        `json:"title"`
	Content     string        `json:"content"`
	Type        TaskType      `json:"type"`
	Kind        string        `json:"kind,omitempty"` // "directive" or "question" - affects execution mode
	Priority    int           `json:"priority"`
	Status      TaskStatus    `json:"status"`
	Provider    string        `json:"provider,omitempty"`
	WorktreeID  string        `json:"worktree_id,omitempty"`
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
}

// TaskStatus represents the lifecycle state of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusQueued     TaskStatus = "queued"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
	TaskStatusDuplicate  TaskStatus = "duplicate"
)

// TaskFilter for querying tasks
type TaskFilter struct {
	Status    []TaskStatus
	Type      []TaskType
	Provider  string
	Since     *time.Time
	Until     *time.Time
	Limit     int
	Offset    int
	OrderBy   string // "created_at", "priority", "started_at"
	OrderDesc bool
}

// TaskStats provides aggregate statistics
type TaskStats struct {
	TotalTasks     int            `json:"total_tasks"`
	PendingTasks   int            `json:"pending_tasks"`
	RunningTasks   int            `json:"running_tasks"`
	CompletedTasks int            `json:"completed_tasks"`
	FailedTasks    int            `json:"failed_tasks"`
	ByType         map[string]int `json:"by_type"`
	ByProvider     map[string]int `json:"by_provider"`
	TotalCost      float64        `json:"total_cost"`
	TotalTokens    int            `json:"total_tokens"`
	AvgDuration    time.Duration  `json:"avg_duration"`
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
	MarkTaskCompleted(ctx context.Context, id string, result *ExecuteResult) error
	MarkTaskFailed(ctx context.Context, id string, err error) error
	MarkTaskCancelled(ctx context.Context, id string) error

	// Duplicate detection
	FindDuplicateTask(ctx context.Context, fingerprint uint64, threshold float64) (*TaskRecord, error)
	SetTaskFingerprint(ctx context.Context, id string, fingerprint uint64) error

	// Thread linking (for dashboard visibility)
	SetTaskThreadID(ctx context.Context, id string, threadID string) error

	// Resource metrics
	UpdateTaskMetrics(ctx context.Context, id string, peakCPU, peakMemory float64) error

	// Cleanup
	DeleteOldTasks(ctx context.Context, olderThan time.Duration) (int, error)

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
