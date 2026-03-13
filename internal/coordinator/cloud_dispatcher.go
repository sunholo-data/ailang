package coordinator

import "context"

// CloudDispatcher triggers remote task execution on a cloud backend.
// Implementations are backend-specific (Cloud Run Jobs, K8s Jobs, etc.)
// The coordinator calls Dispatch() without knowing the backend.
type CloudDispatcher interface {
	// Dispatch triggers execution of a task on the remote backend.
	// The task is already persisted in the task store — the dispatcher
	// only needs to trigger execution with the given parameters.
	Dispatch(ctx context.Context, params DispatchParams) error
}

// DispatchParams contains the parameters needed to trigger remote task execution.
type DispatchParams struct {
	TaskID     string  // Coordinator task ID (e.g., "task-29404032")
	AgentID    string  // Target agent (e.g., "sprint-executor")
	Workspace  string  // Workspace path
	Provider   string  // AI provider ("claude" or "gemini")
	Directive  string  // Task prompt (optional — job can fetch from Firestore)
	RepoURL    string  // Git repo URL
	Branch     string  // Base branch (default: "dev")
	PushBranch string  // If set, push directly to this branch (skip coordinator/ branch creation)
	PluginRepo string  // Git URL for shared skills plugin (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	Model      string  // AI model override (e.g., "sonnet", "opus") — from agent config
	Timeout    string  // Executor timeout (e.g., "15m", "60m") — from agent config (M-CLOUD-OAUTH)
	AuthMode   string  // "oauth" (default) or "apikey" — selects Cloud Run Job template (M-CLOUD-DUAL-AUTH)
	APIKey     string  // User-provided Anthropic API key, only when AuthMode == "apikey"
	MaxCostUSD float64 // Per-task cost budget (0 = unlimited) — M-CLOUD-PROGRESS-TRACKING
}
