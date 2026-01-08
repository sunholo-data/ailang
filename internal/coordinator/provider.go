// Package coordinator provides task execution and orchestration for the AILANG daemon.
package coordinator

import (
	"context"
	"time"

	"github.com/sunholo/ailang/internal/executor"
)

// TaskType represents the category of a task
type TaskType string

const (
	TaskTypeBugFix   TaskType = "bug-fix"
	TaskTypeFeature  TaskType = "feature"
	TaskTypeDocs     TaskType = "docs"
	TaskTypeResearch TaskType = "research"
	TaskTypeRefactor TaskType = "refactor"
	TaskTypeTest     TaskType = "test"
	TaskTypeUnknown  TaskType = "unknown"
)

// Task represents a task to be executed
type Task struct {
	ID           string
	Title        string
	Content      string
	Kind         string // "directive" or "question" - affects execution mode
	Priority     int
	MessageID    string
	ParentTaskID string // Parent task ID for hierarchy tracking (M-TASK-HIERARCHY)
	CreatedAt    time.Time
}

// AnalyzedTask is a task with analysis metadata
type AnalyzedTask struct {
	Task        *Task
	Type        TaskType
	Keywords    []string
	Fingerprint uint64
	DuplicateOf string // ID of duplicate task, if any

	// Capability detection (migrated from internal/agent)
	Capabilities  []Capability `json:"capabilities,omitempty"`   // Detected capability requirements
	ImpactLevel   string       `json:"impact_level,omitempty"`   // "low", "medium", or "high"
	EstimatedCost float64      `json:"estimated_cost,omitempty"` // Pre-execution cost estimate in USD
}

// ExecuteOptions configures task execution
type ExecuteOptions struct {
	Timeout      time.Duration
	DryRun       bool
	Workspace    string                // Working directory for the task
	Model        string                // Model to use (provider-specific)
	EventHandler executor.EventHandler // Optional handler for streaming events

	// Observatory context for trace linking (M-TASK-HIERARCHY)
	ObservatoryContext *ObservatoryContext

	// InvokeConfig for script execution (v0.6.4+)
	// Set when the agent has invoke.type: "script"
	InvokeConfig *InvokeConfig
}

// ObservatoryContext holds context for linking traces to coordinator entities.
// This enables the WORKSPACE → TASK → AGENT → SPANS hierarchy in Observatory.
type ObservatoryContext struct {
	TaskID       string // Coordinator task ID
	AgentID      string // Agent handling the task (e.g., "design-doc-creator")
	AssignmentID string // Observatory agent_assignment ID (aa_xxx)
	WorkspaceID  string // Observatory workspace ID (ws_xxx)
}

// DefaultExecuteOptions returns sensible defaults
func DefaultExecuteOptions() *ExecuteOptions {
	return &ExecuteOptions{
		Timeout: 5 * time.Minute,
		DryRun:  false,
	}
}

// ExecuteResult contains the result of task execution
type ExecuteResult struct {
	Success    bool
	Output     string
	Error      string
	Provider   string
	Duration   time.Duration
	Cost       float64
	TokensUsed int // Total tokens (InputTokens + OutputTokens)
	// Detailed token breakdown
	InputTokens  int
	OutputTokens int
	// Files affected
	FilesCreated  []string
	FilesModified []string
	// Session continuity (for agent-to-agent handoffs)
	SessionID string // Claude Code --resume ID or Gemini CLI --conversation-id
}

// Provider executes tasks using a specific AI backend.
// There are two types of providers:
//   - Executor-based (Claude Code, Gemini CLI): For agentic coding tasks
//   - API-based (Gemini API, Claude API): For simple text generation
type Provider interface {
	// Name returns the provider identifier (e.g., "claude", "gemini", "gemini-api")
	Name() string

	// CanHandle returns true if this provider can handle the given task type
	CanHandle(task *AnalyzedTask) bool

	// Execute runs a task and returns the result
	Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error)
}
