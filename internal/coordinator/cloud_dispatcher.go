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
	TaskID     string // Coordinator task ID (e.g., "task-29404032")
	AgentID    string // Target agent (e.g., "sprint-executor")
	Workspace  string // Workspace path
	Provider   string // AI provider ("claude" or "gemini")
	Directive  string // Task prompt (optional — job can fetch from Firestore)
	RepoURL    string // Git repo URL
	Branch     string // Base branch (default: "dev")
	PushBranch string // If set, push directly to this branch (skip coordinator/ branch creation)
}
