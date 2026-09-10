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
	TaskID          string  // Coordinator task ID (e.g., "task-29404032")
	AgentID         string  // Target agent (e.g., "sprint-executor")
	Workspace       string  // Workspace path
	Provider        string  // AI provider ("claude" or "gemini")
	Directive       string  // Task prompt (optional — job can fetch from Firestore)
	RepoURL         string  // Git repo URL
	Branch          string  // Base branch (default: "dev")
	PushBranch      string  // If set, push directly to this branch (skip coordinator/ branch creation)
	PluginRepo      string  // Git URL for shared skills plugin (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	Model           string  // AI model override (e.g., "sonnet", "opus") — from agent config
	Timeout         string  // Executor timeout (e.g., "15m", "60m") — from agent config (M-CLOUD-OAUTH)
	AuthMode        string  // "oauth" (default) or "apikey" — selects Cloud Run Job template (M-CLOUD-DUAL-AUTH)
	APIKey          string  // User-provided Anthropic API key, only when AuthMode == "apikey"
	MaxCostUSD      float64 // Per-task cost budget (0 = unlimited) — M-CLOUD-PROGRESS-TRACKING
	GitMode         string  // "guardrails", "strict", "permissive" — M-GIT-GUARDRAILS
	SiteSlug        string  // Website site slug for commit message — M-HARNESS-COMMIT-CONTRACT
	BriefID         string  // Brief ID for commit message — M-HARNESS-COMMIT-CONTRACT
	Subdirectory    string  // Monorepo subdirectory for package agents — M-PKG-AUTONOMOUS-UPDATES
	ExecutorVariant string  // Docker image variant — M-EXECUTOR-VARIANTS ("", "go", "codex", etc.)

	// WorkTier is the permission tier the session-protocol gate runs under
	// ("tier1"/"tier2") — M-COORDINATOR-EXECUTION-TRUST M1a. Always set via
	// ResolveWorkTier, never from message content (design doc V18) and never
	// from a sender-chosen inbox (V25). Empty is read as tier 2 by the gate.
	WorkTier string

	// AcknowledgeOnly marks a dispatch that is NOT expected to change files.
	// Trusted metadata (from the agent registry), the same authority boundary as
	// WorkTier, and deliberately NOT the content-derived task type (V18).
	//
	// Stated as "acknowledge-only" rather than "expect changes" so the Go zero
	// value is the LOUD direction: a dispatch that forgets to set this reports a
	// no-diff run as no_changes, instead of silently inheriting the lenient
	// behaviour this milestone exists to remove.
	AcknowledgeOnly bool

	// M-PKG-CASCADE-DETERMINISTIC-FIRST: cascade envelope fields, propagated
	// from TaskRecord so the Cloud Run Job wrapper can decide deterministic-
	// bump vs AI-escalation without re-fetching the task. Empty/false for
	// non-cascade tasks.
	RootPackage       string
	RootChangeClass   string
	FromVersion       string
	ToVersion         string
	FromInterfaceHash string
	ToInterfaceHash   string
	EffectsWidened    bool
}
