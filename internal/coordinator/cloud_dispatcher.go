package coordinator

import (
	"context"
	"errors"
)

// ErrDispatchPermanent marks a dispatch that will fail identically on every
// retry — an oversized directive, a request the backend rejects as malformed.
// The daemon fails the task on it instead of resetting it to pending: reset,
// task-68771ff3 was re-dispatched every five minutes for hours (2026-09-30)
// while its status said only "pending".
var ErrDispatchPermanent = errors.New("dispatch cannot succeed")

// MaxDirectiveBytes bounds a task directive. It is stored as one Firestore
// document (1 MiB limit) that the job reads by task id; anything larger is
// refused at dispatch rather than truncated, since truncation silently drops
// whatever the request put last.
const MaxDirectiveBytes = 900 * 1024

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
	TaskID    string // Coordinator task ID (e.g., "task-29404032")
	AgentID   string // Target agent (e.g., "sprint-executor")
	Workspace string // Workspace path
	Provider  string // AI provider ("claude" or "gemini")
	Directive string // Task prompt (optional — job can fetch from Firestore)
	// TaskTitle is the human description of the work, carried so the job does not
	// have to RE-DERIVE one from the prompt. The prompt is template-wrapped before
	// dispatch, so its first line is boilerplate ("You are an autonomous AILANG
	// …") and any heuristic that reads it produces a PR title naming the harness
	// instead of the change — measured on PR #62, 2026-09-14.
	TaskTitle  string
	RepoURL    string // Git repo URL
	Branch     string // Base branch (default: "dev")
	PushBranch string // If set, push directly to this branch (skip coordinator/ branch creation)
	PluginRepo string // Git URL for shared skills plugin (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	Model      string // AI model override (e.g., "sonnet", "opus") — from agent config
	Timeout    string // Executor timeout (e.g., "15m", "60m") — from agent config (M-CLOUD-OAUTH)
	// IdleTimeout is the max silence BETWEEN output events, distinct from
	// Timeout's hard ceiling. It must travel: the local daemon read it from the
	// agent config and the cloud job did not, so every cloud task ran on the
	// executor's hardcoded 3m no matter what the registry declared — 41 agents
	// declaring 5m/6m/10m, all silently 3m, while `coordinator agents <id>`
	// printed the declared value as EFFECTIVE. Measured 2026-09-22.
	IdleTimeout     string
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

	// AutoMerge asks the wrapper to enable GitHub's NATIVE auto-merge on the PR
	// it opens, so GitHub merges it once the required checks pass.
	//
	// Native rather than a merge call of our own: GitHub does the waiting, honours
	// branch protection, and simply never merges if the checks do not go green —
	// no polling loop of ours to get wrong, and no path where we merge something
	// protection would have refused. Before this, NOTHING in the codebase merged a
	// PR at all: `auto_merge` was read by the autonomy router and never reached a
	// merge API, so approving a cloud task marked it COMPLETED "(merge skipped)"
	// and left the branch open. Four design docs were open on 2026-09-10, the
	// oldest since 09-02, two of them MERGEABLE and simply waiting for a human.
	//
	// Trusted metadata from the agent registry, never from message content — the
	// same authority boundary as WorkTier.
	AutoMerge bool
	// Code auto-merge authority comes exclusively from trusted registry metadata.
	// Only a Secret Manager name travels here, never the approver token.
	AutoMergeCode             bool
	AutoMergeRequiredChecks   []string
	AutoMergeApproverSecret   string
	AutoMergeApproverIdentity string

	// ArtifactPatterns is what this agent is DECLARED to produce. The wrapper's
	// auto-merge guard requires every changed file to match one, so a run that
	// strays outside the declaration is not auto-merged even when the agent is
	// configured for it and the checks are green. Registry metadata, like
	// AutoMerge itself.
	ArtifactPatterns []string

	// PRLabels and MergeStarts make a PR say what it is and what merging it
	// does. Every coordinator PR used to carry only `agent-task`, so a triage
	// note, a design doc, a plan and 530 lines of code looked identical, and
	// nothing said that merging a plan starts sprint-executor (feedback
	// 2026-09-28). Both come from the registry (PRApprovalLabel,
	// approvalHandoffTargets) — the same source the approval card reads.
	PRLabels    []string
	MergeStarts []string

	// GitAuthorName/Email author this agent's commits. Empty inherits the
	// container's identity (the fleet bot).
	GitAuthorName  string
	GitAuthorEmail string

	// SSHKeySecret/SSHHostAlias select a per-repo deploy key. The secret NAME
	// only — never its value.
	SSHKeySecret string
	SSHHostAlias string

	// ToolPolicy / PolicyTOML: the agent's tool lane and the CONTENT of its
	// program policy (M-AGENT-AILANG-ONLY-EXECUTION). Content, not a path: a
	// Job cannot read the coordinator's disk. Registry metadata.
	ToolPolicy string
	PolicyTOML string

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
