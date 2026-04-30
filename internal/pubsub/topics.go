package pubsub

// DefaultTopicPrefix is the default prefix for all AILANG Pub/Sub topics.
const DefaultTopicPrefix = "ailang"

// Topic base names. The full topic name is "{prefix}-{base}".
const (
	TopicMessages    = "messages"    // New message notifications (attribute-routed by inbox)
	TopicTasks       = "tasks"       // Task dispatch to Cloud Run Jobs
	TopicCompletions = "completions" // Task completion notifications
	TopicEvents      = "events"      // Real-time dashboard/laptop event streaming
	TopicDeadLetter  = "dead-letter" // Failed message sink
	TopicCascade     = "cascade"     // Authoritative package cascade triggers (M-PKG-AUTONOMOUS-CASCADE-SAFE M2)
)

// SourceCascade is the value for the "source" message attribute on
// cascade-topic publishes. Agents check this against the message they
// receive to decide whether to act on a bump request.
const SourceCascade = "cascade"

// Subscription base names. The full name is "{prefix}-{base}".
const (
	SubMessagesCoordinator    = "messages-coordinator"    // Cloud Run coordinator
	SubMessagesLaptop         = "messages-laptop"         // Developer laptop (pull)
	SubTasksExecutor          = "tasks-executor"          // Eventarc → Cloud Run Job
	SubCompletionsCoordinator = "completions-coordinator" // Coordinator receives results
	SubEventsDashboard        = "events-dashboard"        // Dashboard server
	SubEventsLaptop           = "events-laptop"           // Laptop real-time updates
)

// MessageNotification is published to the messages topic.
// Intentionally minimal — full message content lives in Firestore.
type MessageNotification struct {
	MessageID string `json:"message_id"`
}

// TaskDispatch is published to the tasks topic to trigger a Cloud Run Job.
type TaskDispatch struct {
	TaskID  string `json:"task_id"`
	AgentID string `json:"agent_id"`
}

// TaskCompletion is published to the completions topic when a job finishes.
type TaskCompletion struct {
	TaskID     string `json:"task_id"`
	AgentID    string `json:"agent_id"`
	Status     string `json:"status"`                // "completed" or "failed"
	BranchName string `json:"branch_name,omitempty"` // Git branch with changes
	ErrorMsg   string `json:"error_msg,omitempty"`

	// Changed files discovered via git diff after execution.
	// Used by external clients (portal, sidecar) to know which files were created/modified.
	ChangedFiles []string `json:"changed_files,omitempty"`

	// Executor metrics (populated when using full executor infrastructure)
	SessionID     string  `json:"session_id,omitempty"`
	NumTurns      int     `json:"num_turns,omitempty"`
	ToolCallCount int     `json:"tool_call_count,omitempty"`
	InputTokens   int     `json:"input_tokens,omitempty"`
	OutputTokens  int     `json:"output_tokens,omitempty"`
	CostUSD       float64 `json:"cost_usd,omitempty"`
	DurationMS    int     `json:"duration_ms,omitempty"`

	// Cache token breakdown (omitted from InputTokens total — additive cost context)
	CacheReadTokens     int `json:"cache_read_tokens,omitempty"`
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`

	// GCS path prefix for raw artifacts: transcript.txt, session.jsonl, metrics.json
	// Format: "tasks/{taskID}" (relative to the per-environment artifact bucket)
	ArtifactGCSPath string `json:"artifact_gcs_path,omitempty"`
}

// MessageAttributes carries routing metadata as Pub/Sub message attributes.
// Used for subscription filtering (e.g., filter by inbox or workspace).
type MessageAttributes struct {
	Inbox       string // Target agent inbox (e.g., "design-doc-creator")
	Workspace   string // Project workspace (e.g., "sunholo-data/ailang")
	FromAgent   string // Sender agent ID (e.g., "user", "sprint-planner")
	Category    string // Message category (e.g., "bug", "feature", "general")
	MessageType string // Message type (e.g., "request", "notification")

	// Source identifies which Pub/Sub topic the message arrived on.
	// (M-PKG-AUTONOMOUS-CASCADE-SAFE M1) Used by the agent to distinguish
	// authoritative cascade-driven bumps ("cascade") from public-routed
	// feedback ("messages" or empty). Set by the publisher; the receiving
	// adapter copies it through to Message.Source for downstream guards.
	Source string
}

// ToMap converts attributes to map[string]string for Pub/Sub message publishing.
// Only non-empty values are included.
func (a MessageAttributes) ToMap() map[string]string {
	m := make(map[string]string, 6)
	if a.Inbox != "" {
		m["inbox"] = a.Inbox
	}
	if a.Workspace != "" {
		m["workspace"] = a.Workspace
	}
	if a.FromAgent != "" {
		m["from_agent"] = a.FromAgent
	}
	if a.Category != "" {
		m["category"] = a.Category
	}
	if a.MessageType != "" {
		m["message_type"] = a.MessageType
	}
	if a.Source != "" {
		m["source"] = a.Source
	}
	return m
}

// AttributesFromMap creates MessageAttributes from a Pub/Sub message attributes map.
func AttributesFromMap(m map[string]string) MessageAttributes {
	return MessageAttributes{
		Inbox:       m["inbox"],
		Workspace:   m["workspace"],
		FromAgent:   m["from_agent"],
		Category:    m["category"],
		MessageType: m["message_type"],
		Source:      m["source"],
	}
}
