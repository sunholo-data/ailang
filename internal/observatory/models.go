// Package observatory provides a unified observability platform for AILANG.
// It stores and queries traces, spans, metrics, and events from AILANG operations
// and external CLI tools (Claude Code, Gemini CLI).
package observatory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// DefaultDatabasePath returns the default path for the observatory database.
func DefaultDatabasePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "observatory.db"
	}
	return filepath.Join(homeDir, ".ailang", "state", "observatory.db")
}

// Workspace represents a git repository or project root.
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	GitRemote string    `json:"git_remote,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkspaceStats contains aggregated metrics for a workspace.
type WorkspaceStats struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	TaskCount    int       `json:"task_count"`
	TotalCost    float64   `json:"total_cost"`
	TotalTokens  int64     `json:"total_tokens"`
	SuccessRate  float64   `json:"success_rate"`
	UniqueAgents int       `json:"unique_agents"`
	LastActivity time.Time `json:"last_activity,omitempty"`
}

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// TaskSourceType represents the origin of a task.
type TaskSourceType string

const (
	TaskSourceGitHub  TaskSourceType = "github_issue"
	TaskSourceMessage TaskSourceType = "message"
	TaskSourceEmail   TaskSourceType = "email"
	TaskSourceManual  TaskSourceType = "manual"
)

// Task represents a unit of work (from GitHub, messages, email, etc.).
type Task struct {
	ID           string         `json:"id"`
	WorkspaceID  string         `json:"workspace_id"`
	ParentTaskID string         `json:"parent_task_id,omitempty"` // Links to parent task for handoff chains
	Title        string         `json:"title"`
	Description  string         `json:"description,omitempty"`
	SourceType   TaskSourceType `json:"source_type"`
	SourceRef    string         `json:"source_ref,omitempty"`
	Status       TaskStatus     `json:"status"`
	Priority     string         `json:"priority"`
	CreatedAt    time.Time      `json:"created_at"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`

	// Aggregated metrics
	TotalDurationMs int64   `json:"total_duration_ms"`
	TotalTokensIn   int64   `json:"total_tokens_in"`
	TotalTokensOut  int64   `json:"total_tokens_out"`
	TotalCostUSD    float64 `json:"total_cost_usd"`
	AgentCount      int     `json:"agent_count"`
	SpanCount       int     `json:"span_count"`
	ErrorCount      int     `json:"error_count"`
}

// AgentAssignmentStatus represents the status of an agent assignment.
type AgentAssignmentStatus string

const (
	AgentStatusPending   AgentAssignmentStatus = "pending"
	AgentStatusRunning   AgentAssignmentStatus = "running"
	AgentStatusCompleted AgentAssignmentStatus = "completed"
	AgentStatusFailed    AgentAssignmentStatus = "failed"
)

// Provider represents an AI provider.
type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderGemini Provider = "gemini"
	ProviderOllama Provider = "ollama"
)

// AgentAssignment represents a coordinator → agent delegation.
type AgentAssignment struct {
	ID                 string                `json:"id"`
	TaskID             string                `json:"task_id"`
	AgentID            string                `json:"agent_id"`
	Provider           Provider              `json:"provider"`
	Status             AgentAssignmentStatus `json:"status"`
	AssignedAt         time.Time             `json:"assigned_at"`
	StartedAt          *time.Time            `json:"started_at,omitempty"`
	CompletedAt        *time.Time            `json:"completed_at,omitempty"`
	ParentAssignmentID string                `json:"parent_assignment_id,omitempty"`

	// Agent-level aggregates
	DurationMs int64   `json:"duration_ms"`
	TokensIn   int64   `json:"tokens_in"`
	TokensOut  int64   `json:"tokens_out"`
	CostUSD    float64 `json:"cost_usd"`
	ToolCalls  int     `json:"tool_calls"`
	Turns      int     `json:"turns"`
}

// AgentStats contains aggregated metrics for an agent.
type AgentStats struct {
	AgentID         string   `json:"agent_id"`
	Provider        Provider `json:"provider"`
	ExecutionCount  int      `json:"execution_count"`
	TotalDurationMs int64    `json:"total_duration_ms"`
	AvgDurationMs   float64  `json:"avg_duration_ms"`
	TotalTokensIn   int64    `json:"total_tokens_in"`
	TotalTokensOut  int64    `json:"total_tokens_out"`
	TotalCost       float64  `json:"total_cost"`
	TotalToolCalls  int      `json:"total_tool_calls"`
	SuccessRate     float64  `json:"success_rate"`
}

// SpanKind represents the type of span.
type SpanKind string

const (
	SpanKindInternal SpanKind = "internal"
	SpanKindClient   SpanKind = "client"
	SpanKindServer   SpanKind = "server"
	SpanKindProducer SpanKind = "producer"
	SpanKindConsumer SpanKind = "consumer"
)

// SpanStatus represents the status of a span.
type SpanStatus string

const (
	SpanStatusOK    SpanStatus = "ok"
	SpanStatusError SpanStatus = "error"
	SpanStatusUnset SpanStatus = "unset"
)

// Span represents an OTEL span with normalized attributes.
type Span struct {
	ID                string     `json:"id"`
	TraceID           string     `json:"trace_id"`
	ParentSpanID      string     `json:"parent_span_id,omitempty"`
	TaskID            string     `json:"task_id,omitempty"`
	AgentAssignmentID string     `json:"agent_assignment_id,omitempty"`
	Name              string     `json:"name"`
	Kind              SpanKind   `json:"kind"`
	Status            SpanStatus `json:"status"`
	StatusMessage     string     `json:"status_message,omitempty"`
	StartTime         time.Time  `json:"start_time"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	DurationMs        int64      `json:"duration_ms,omitempty"`

	// Normalized attributes (common across providers)
	TokensIn  int64    `json:"tokens_in,omitempty"`
	TokensOut int64    `json:"tokens_out,omitempty"`
	CostUSD   float64  `json:"cost_usd,omitempty"`
	Model     string   `json:"model,omitempty"`
	Provider  Provider `json:"provider,omitempty"`

	// Full attributes as JSON (for provider-specific data)
	Attributes         map[string]any `json:"attributes,omitempty"`
	ResourceAttributes map[string]any `json:"resource_attributes,omitempty"`

	// Children (for tree building, not stored in DB)
	Children []*Span `json:"children,omitempty"`

	// Events (loaded separately)
	Events []SpanEvent `json:"events,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// AttributesJSON returns attributes as a JSON string for storage.
func (s *Span) AttributesJSON() string {
	if s.Attributes == nil {
		return "{}"
	}
	b, _ := json.Marshal(s.Attributes)
	return string(b)
}

// ResourceAttributesJSON returns resource attributes as a JSON string for storage.
func (s *Span) ResourceAttributesJSON() string {
	if s.ResourceAttributes == nil {
		return "{}"
	}
	b, _ := json.Marshal(s.ResourceAttributes)
	return string(b)
}

// ParseAttributes parses a JSON string into attributes.
func (s *Span) ParseAttributes(jsonStr string) error {
	if jsonStr == "" || jsonStr == "{}" {
		s.Attributes = nil
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), &s.Attributes)
}

// ParseResourceAttributes parses a JSON string into resource attributes.
func (s *Span) ParseResourceAttributes(jsonStr string) error {
	if jsonStr == "" || jsonStr == "{}" {
		s.ResourceAttributes = nil
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), &s.ResourceAttributes)
}

// EventType represents the type of span event.
type EventType string

const (
	EventTypeApproval EventType = "approval"
	EventTypeTool     EventType = "tool"
	EventTypeError    EventType = "error"
	EventTypeCustom   EventType = "custom"
)

// ApprovalStatus represents the status of an approval event.
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

// SpanEvent represents an event attached to a span (approvals, tool calls, errors).
type SpanEvent struct {
	ID        int64     `json:"id,omitempty"`
	SpanID    string    `json:"span_id"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	EventType EventType `json:"event_type,omitempty"`

	// Denormalized for common event types
	ApprovalStatus ApprovalStatus `json:"approval_status,omitempty"`
	ToolName       string         `json:"tool_name,omitempty"`
	ErrorMessage   string         `json:"error_message,omitempty"`

	Attributes map[string]any `json:"attributes,omitempty"`
}

// AttributesJSON returns attributes as a JSON string for storage.
func (e *SpanEvent) AttributesJSON() string {
	if e.Attributes == nil {
		return "{}"
	}
	b, _ := json.Marshal(e.Attributes)
	return string(b)
}

// ParseAttributes parses a JSON string into attributes.
func (e *SpanEvent) ParseAttributes(jsonStr string) error {
	if jsonStr == "" || jsonStr == "{}" {
		e.Attributes = nil
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), &e.Attributes)
}

// MessageStatus represents the status of a message.
type MessageStatus string

const (
	MessageStatusUnread   MessageStatus = "unread"
	MessageStatusRead     MessageStatus = "read"
	MessageStatusArchived MessageStatus = "archived"
)

// Message represents an agent-to-agent message.
type Message struct {
	ID          string        `json:"id"`
	TaskID      string        `json:"task_id,omitempty"`
	Inbox       string        `json:"inbox"`
	FromAgent   string        `json:"from_agent"`
	Title       string        `json:"title"`
	Content     string        `json:"content"`
	MessageType string        `json:"message_type"`
	Status      MessageStatus `json:"status"`
	Priority    string        `json:"priority"`

	// GitHub sync
	GitHubIssueNumber int    `json:"github_issue_number,omitempty"`
	GitHubRepo        string `json:"github_repo,omitempty"`

	// Correlation
	CorrelationID string `json:"correlation_id,omitempty"`
	ReplyToID     string `json:"reply_to_id,omitempty"`

	CreatedAt  time.Time  `json:"created_at"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`

	// Search
	ContentHash string `json:"content_hash,omitempty"`
}

// Trace represents a complete trace with its span tree.
type Trace struct {
	TraceID    string    `json:"trace_id"`
	RootSpan   *Span     `json:"root_span,omitempty"`
	Spans      []*Span   `json:"spans"`
	SpanCount  int       `json:"span_count"`
	DurationMs int64     `json:"duration_ms"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
}

// ProviderComparison contains comparison metrics between providers.
type ProviderComparison struct {
	Provider        Provider `json:"provider"`
	TotalExecutions int      `json:"total_executions"`
	TotalTokensIn   int64    `json:"total_tokens_in"`
	TotalTokensOut  int64    `json:"total_tokens_out"`
	TotalCost       float64  `json:"total_cost"`
	AvgDurationMs   float64  `json:"avg_duration_ms"`
	SuccessRate     float64  `json:"success_rate"`
}

// TaskTimeline represents a task with its span timeline.
type TaskTimeline struct {
	TaskID     string     `json:"task_id"`
	Title      string     `json:"title"`
	Status     TaskStatus `json:"status"`
	SpanID     string     `json:"span_id,omitempty"`
	SpanName   string     `json:"span_name,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	DurationMs int64      `json:"duration_ms,omitempty"`
	SpanStatus SpanStatus `json:"span_status,omitempty"`
	TokensIn   int64      `json:"tokens_in,omitempty"`
	TokensOut  int64      `json:"tokens_out,omitempty"`
	CostUSD    float64    `json:"cost_usd,omitempty"`
	Provider   Provider   `json:"provider,omitempty"`
}

// MetricsSummary contains global metrics summary.
type MetricsSummary struct {
	TotalWorkspaces int     `json:"total_workspaces"`
	TotalTasks      int     `json:"total_tasks"`
	TotalSpans      int     `json:"total_spans"`
	TotalAgents     int     `json:"total_agents"`
	TotalTokensIn   int64   `json:"total_tokens_in"`
	TotalTokensOut  int64   `json:"total_tokens_out"`
	TotalCostUSD    float64 `json:"total_cost_usd"`
	SuccessRate     float64 `json:"success_rate"`
}

// TimeRange represents a time range for queries.
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// TraceQuery represents query parameters for listing traces.
type TraceQuery struct {
	TaskID    string     `json:"task_id,omitempty"`
	TraceID   string     `json:"trace_id,omitempty"`
	TimeRange *TimeRange `json:"time_range,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// TraceSource indicates where the trace data originated from.
type TraceSource string

const (
	TraceSourceLocal  TraceSource = "local"  // Local OTLP receiver
	TraceSourceGCP    TraceSource = "gcp"    // Google Cloud Trace
	TraceSourceJaeger TraceSource = "jaeger" // Jaeger backend
)

// TraceSummary represents a summary of a trace for list views.
type TraceSummary struct {
	TraceID     string      `json:"trace_id"`
	RootSpan    string      `json:"root_span"`
	SpanCount   int         `json:"span_count"`
	DurationMs  int64       `json:"duration_ms"`
	StartTime   time.Time   `json:"start_time"`
	Status      SpanStatus  `json:"status"`
	TaskID      string      `json:"task_id,omitempty"`
	ServiceName string      `json:"service_name,omitempty"` // e.g., "ailang-run", "ailang-eval", "claude-code"
	Source      TraceSource `json:"source,omitempty"`       // Where trace came from: local, gcp, jaeger
}
