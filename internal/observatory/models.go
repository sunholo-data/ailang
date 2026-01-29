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
	ChainID           string     `json:"chain_id,omitempty"` // M-CHAINS-SIMPLIFY: links to execution_chains
	StageID           string     `json:"stage_id,omitempty"` // M-CHAINS-SIMPLIFY: links to chain_stages
	Name              string     `json:"name"`
	Kind              SpanKind   `json:"kind"`
	Status            SpanStatus `json:"status"`
	StatusMessage     string     `json:"status_message,omitempty"`
	StartTime         time.Time  `json:"start_time"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	DurationMs        int64      `json:"duration_ms,omitempty"`

	// Normalized attributes (common across providers)
	TokensIn            int64    `json:"tokens_in,omitempty"`
	TokensOut           int64    `json:"tokens_out,omitempty"`
	CacheReadTokens     int64    `json:"cache_read_tokens,omitempty"`
	CacheCreationTokens int64    `json:"cache_creation_tokens,omitempty"`
	CostUSD             float64  `json:"cost_usd,omitempty"`
	Model               string   `json:"model,omitempty"`
	Provider            Provider `json:"provider,omitempty"`

	// Full attributes as JSON (for provider-specific data)
	Attributes         map[string]any `json:"attributes,omitempty"`
	ResourceAttributes map[string]any `json:"resource_attributes,omitempty"`

	// Children (for tree building, not stored in DB)
	Children []*Span `json:"children,omitempty"`

	// Events (loaded separately)
	Events []SpanEvent `json:"events,omitempty"`

	// DisplayName is an enriched human-readable label (not stored in DB, computed from session_tools)
	DisplayName string `json:"display_name,omitempty"`

	// ChatContext contains embedded chat content (populated with include_chat=true API param)
	ChatContext *ChatContext `json:"chat_context,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// ChatContext contains conversation content for spans with chat history.
// Populated when include_chat=true on API requests.
// This enables embedding chat prompts/responses directly in span data.
type ChatContext struct {
	UserPrompt        string `json:"user_prompt,omitempty"`        // First 500 chars of user prompt
	AssistantResponse string `json:"assistant_response,omitempty"` // First 500 chars of response
	HasThinking       bool   `json:"has_thinking"`
	TurnNumber        int    `json:"turn_number,omitempty"`
	FullChatURL       string `json:"full_chat_url,omitempty"` // Link to full conversation
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

	// Cache metrics (from spans)
	TotalCacheReadTokens     int64   `json:"total_cache_read_tokens,omitempty"`
	TotalCacheCreationTokens int64   `json:"total_cache_creation_tokens,omitempty"`
	CacheSavingsUSD          float64 `json:"cache_savings_usd,omitempty"`

	// Lines of Code metrics (from metrics table)
	LinesAdded   int64 `json:"lines_added,omitempty"`
	LinesRemoved int64 `json:"lines_removed,omitempty"`

	// Activity metrics (from metrics table)
	CommitCount      int64 `json:"commit_count,omitempty"`
	PullRequestCount int64 `json:"pull_request_count,omitempty"`
	ActiveTimeMs     int64 `json:"active_time_ms,omitempty"`

	// Session metrics
	TurnCount  int `json:"turn_count,omitempty"`
	ToolCalls  int `json:"tool_calls,omitempty"`
	ErrorCount int `json:"error_count,omitempty"`
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

// SpanOutlier represents a span that is statistically anomalous within its task.
type SpanOutlier struct {
	Span           *Span   `json:"span"`
	Metric         string  `json:"metric"`           // "cost_usd", "duration_ms", "tokens"
	Value          float64 `json:"value"`            // Actual metric value
	Mean           float64 `json:"mean"`             // Task mean for this metric
	StdDev         float64 `json:"std_dev"`          // Task standard deviation
	ZScore         float64 `json:"z_score"`          // (value - mean) / stddev
	PercentOfTotal float64 `json:"percent_of_total"` // What % of task total this span represents
}

// TaskMetricStats holds statistical summary for a single metric within a task.
type TaskMetricStats struct {
	Metric string  `json:"metric"`
	Count  int     `json:"count"`   // Number of spans with non-zero values
	Sum    float64 `json:"sum"`     // Total value across all spans
	Mean   float64 `json:"mean"`    // Average value
	StdDev float64 `json:"std_dev"` // Standard deviation
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

// OutlierAnalysis contains the full outlier analysis for a task.
type OutlierAnalysis struct {
	TaskID       string             `json:"task_id"`
	TaskTitle    string             `json:"task_title"`
	SpanCount    int                `json:"span_count"`
	Stats        []*TaskMetricStats `json:"stats"`                    // Per-metric statistics
	Outliers     []*SpanOutlier     `json:"outliers"`                 // Detected outliers (sorted by |z-score| desc)
	RateOfChange *RateAnalysis      `json:"rate_of_change,omitempty"` // Optional rate analysis
	Threshold    float64            `json:"threshold"`                // Z-score threshold used
	AnalyzedAt   time.Time          `json:"analyzed_at"`
}

// RateAnalysis shows how metrics accumulated during task execution.
type RateAnalysis struct {
	CumulativeCost     []CumulativePoint `json:"cumulative_cost"`
	CumulativeTokens   []CumulativePoint `json:"cumulative_tokens"`
	CumulativeDuration []CumulativePoint `json:"cumulative_duration"`
}

// CumulativePoint represents a point in the cumulative progression.
type CumulativePoint struct {
	SpanIndex    int       `json:"span_index"`
	SpanName     string    `json:"span_name"`
	Timestamp    time.Time `json:"timestamp"`
	Value        float64   `json:"value"`         // Value of this span
	Cumulative   float64   `json:"cumulative"`    // Running total up to this span
	DeltaPercent float64   `json:"delta_percent"` // This span's contribution as % of final total
}

// SpanHierarchyNodeType categorizes spans for visualization.
type SpanHierarchyNodeType string

const (
	NodeTypeCoordinator SpanHierarchyNodeType = "coordinator"
	NodeTypeExecutor    SpanHierarchyNodeType = "executor"
	NodeTypeTurn        SpanHierarchyNodeType = "turn"
	NodeTypeTool        SpanHierarchyNodeType = "tool"
	NodeTypeOther       SpanHierarchyNodeType = "other"
)

// SpanHierarchyNode represents a span with its children for hierarchy visualization.
// Unlike Span, this is optimized for graph/tree rendering with parent_span_id-based relationships.
type SpanHierarchyNode struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	ParentID            string                 `json:"parent_id,omitempty"`
	Depth               int                    `json:"depth"`
	StartTime           time.Time              `json:"start_time"`
	DurationMs          int64                  `json:"duration_ms"`
	TokensIn            int64                  `json:"tokens_in,omitempty"`
	TokensOut           int64                  `json:"tokens_out,omitempty"`
	CacheReadTokens     int64                  `json:"cache_read_tokens,omitempty"`
	CacheCreationTokens int64                  `json:"cache_creation_tokens,omitempty"`
	CostUSD             float64                `json:"cost_usd,omitempty"`
	TurnNumber          int                    `json:"turn_number,omitempty"`
	ToolName            string                 `json:"tool_name,omitempty"`
	NodeType            SpanHierarchyNodeType  `json:"node_type"`
	SessionID           string                 `json:"session_id,omitempty"`
	Status              SpanStatus             `json:"status"`
	Provider            Provider               `json:"provider,omitempty"`
	Children            []*SpanHierarchyNode   `json:"children,omitempty"`
	Attributes          map[string]interface{} `json:"attributes,omitempty"` // Selected useful attributes
}

// SpanHierarchyStats contains aggregated stats for a hierarchy result.
type SpanHierarchyStats struct {
	TotalSpans  int     `json:"total_spans"`
	TotalCost   float64 `json:"total_cost"`
	TotalTokens struct {
		In            int64 `json:"in"`
		Out           int64 `json:"out"`
		CacheRead     int64 `json:"cache_read"`
		CacheCreation int64 `json:"cache_creation"`
	} `json:"total_tokens"`
	CacheSavingsUSD float64 `json:"cache_savings_usd,omitempty"`
	TimeRange       struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"time_range"`
	MaxDepth int `json:"max_depth"`
}

// SpanHierarchyResult contains the complete hierarchy result with roots and sessions.
type SpanHierarchyResult struct {
	Roots    []*SpanHierarchyNode `json:"roots"`
	Sessions map[string]int       `json:"sessions"` // session_id -> turn_count
	Stats    SpanHierarchyStats   `json:"stats"`
}

// Metric represents an OTLP counter or gauge metric from Claude Code telemetry.
// Captures: lines_of_code.count, commit.count, pull_request.count, active_time.total, etc.
type Metric struct {
	ID        int64  `json:"id,omitempty"`
	Name      string `json:"name"`
	Type      string `json:"metric_type"` // "counter", "gauge"
	SessionID string `json:"session_id,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Provider  string `json:"provider,omitempty"`

	// Denormalized labels for efficient queries
	LabelType     string `json:"label_type,omitempty"`     // "added"/"removed" for LOC
	LabelTool     string `json:"label_tool,omitempty"`     // tool name for tool metrics
	LabelDecision string `json:"label_decision,omitempty"` // "approved"/"rejected" for code_edit_tool
	LabelLanguage string `json:"label_language,omitempty"` // language for LOC
	LabelModel    string `json:"label_model,omitempty"`    // model name for model-specific metrics

	// Values (only one will be set based on metric type)
	ValueInt   int64   `json:"value_int,omitempty"`
	ValueFloat float64 `json:"value_float,omitempty"`

	// Full labels as JSON (for provider-specific data)
	Labels             map[string]any `json:"labels,omitempty"`
	ResourceAttributes map[string]any `json:"resource_attributes,omitempty"`

	Timestamp time.Time `json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`
}

// LabelsJSON returns labels as a JSON string for storage.
func (m *Metric) LabelsJSON() string {
	if m.Labels == nil {
		return "{}"
	}
	b, _ := json.Marshal(m.Labels)
	return string(b)
}

// ResourceAttributesJSON returns resource attributes as a JSON string for storage.
func (m *Metric) ResourceAttributesJSON() string {
	if m.ResourceAttributes == nil {
		return "{}"
	}
	b, _ := json.Marshal(m.ResourceAttributes)
	return string(b)
}

// MetricListOptions specifies filters for listing metrics.
type MetricListOptions struct {
	SessionID string     `json:"session_id,omitempty"`
	Workspace string     `json:"workspace,omitempty"`
	Name      string     `json:"name,omitempty"` // e.g., "claude_code.lines_of_code.count"
	TimeRange *TimeRange `json:"time_range,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// SessionMetricsSummary aggregates all metrics for a session.
type SessionMetricsSummary struct {
	SessionID string `json:"session_id"`

	// Token metrics (from spans)
	TokensIn            int64 `json:"tokens_in"`
	TokensOut           int64 `json:"tokens_out"`
	CacheReadTokens     int64 `json:"cache_read_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`

	// Cost metrics (from spans + calculated)
	TotalCostUSD    float64 `json:"total_cost_usd"`
	CacheSavingsUSD float64 `json:"cache_savings_usd"` // Savings from cache reads

	// Lines of code (from metrics)
	LinesAdded   int64 `json:"lines_added"`
	LinesRemoved int64 `json:"lines_removed"`

	// Activity metrics (from metrics)
	CommitCount      int64   `json:"commit_count"`
	PullRequestCount int64   `json:"pull_request_count"`
	ActiveTimeMs     int64   `json:"active_time_ms"`
	TurnCount        int     `json:"turn_count"`
	ToolCalls        int     `json:"tool_calls"`
	DurationMs       int64   `json:"duration_ms"`
	SpanCount        int     `json:"span_count"`
	ErrorCount       int     `json:"error_count"`
	SuccessRate      float64 `json:"success_rate"`
}
