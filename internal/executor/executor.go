// Package executor provides a unified interface for AI coding agent executors.
// This enables AILANG to use multiple AI backends (Claude Code, Gemini CLI, etc.)
// through a common interface for directive execution and benchmarking.
package executor

import (
	"context"
	"time"
)

// Executor is the common interface for all AI coding agent executors.
// Each provider (Claude, Gemini, Codex) implements this interface.
type Executor interface {
	// Name returns the executor identifier (e.g., "claude", "gemini", "codex")
	Name() string

	// Execute runs a task and returns the result
	Execute(ctx context.Context, task *Task) (*Result, error)

	// ExecuteStreaming runs a task with real-time event callbacks
	ExecuteStreaming(ctx context.Context, task *Task, handler EventHandler) (*Result, error)

	// Capabilities returns the list of features this executor supports
	Capabilities() []Capability

	// CostModel returns pricing information for cost calculations
	CostModel() *CostModel

	// HealthCheck verifies the executor is configured and accessible
	HealthCheck(ctx context.Context) error

	// Close releases any resources held by the executor
	Close() error
}

// Task represents a coding task to execute
type Task struct {
	ID           string // Unique task identifier
	ParentTaskID string // Parent task ID for hierarchy tracking (M-TASK-HIERARCHY)
	Directive    string // The instruction/prompt
	SystemPrompt string // Optional system-level context

	// PersistentSystemPrompt asks the executor to deliver SystemPrompt via a
	// persistent system-prompt channel (re-applied to the model every turn)
	// rather than concatenating it into the first user message. A large teaching
	// prompt concatenated into turn 1 ages out of attention over a long agent
	// loop (16-54 turns), causing the model to drift back to its training-corpus
	// defaults. opencode honors this by writing SystemPrompt to <Workspace>/AGENTS.md,
	// which opencode auto-loads into the system context on every turn. Executors
	// without a persistent channel ignore this flag. Set by the eval harness; the
	// coordinator leaves it false so opencode keeps auto-loading the repo's CLAUDE.md.
	PersistentSystemPrompt bool

	Workspace    string            // Working directory (local path)
	Timeout      time.Duration     // Hard ceiling execution timeout
	IdleTimeout  time.Duration     // Kill if no events for this long after first event (0 = use default 3m)
	TTFTTimeout  time.Duration     // Kill if no output before first event (prefill budget; 0 = use default 30s)
	AllowedTools []string          // Tools the agent can use
	Model        string            // Model to use (provider-specific)
	Metadata     map[string]string // Provider-specific options

	// Effort level (Claude Code 2.1.47+: "low", "medium", "high")
	Effort string

	// Plugin directories for Claude Code (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	// Each entry is a path to a plugin directory containing .claude-plugin/plugin.json.
	// Passed as --plugin-dir flags to Claude CLI.
	PluginDirs []string

	// Plugins specifies third-party plugins to install before execution (M-CLOUD-PLUGIN-SKILLS, v0.9.1).
	// Marketplaces are registered first, then plugins installed from those marketplaces.
	Plugins *PluginsConfig

	// Session continuity (M-TRANSCRIPT)
	Iteration       int    // Iteration number (1 = first run, 2+ = re-run with feedback)
	ResumeSessionID string // Previous session ID to resume (for Iteration > 1)

	// GCP override — when set, overrides GOOGLE_CLOUD_PROJECT/LOCATION env vars
	// for the subprocess. Populated from models.yml gcp_project / gcp_location fields.
	GCPProject  string
	GCPLocation string

	// Cost budget (M-EVAL-COST-AND-SPEED-BUDGETS, v0.15.1).
	// When set to a non-nil *CostBudget, executors call budget.Add(input, output)
	// at their natural token-tally event point and exit early with
	// Result.CostKilledAt > 0 if the budget is exceeded mid-stream.
	// Nil = legacy behaviour (wall-clock-only).
	Budget *CostBudget

	// MaxTokensPerBench (M-EVAL-OS-LONGITUDINAL Phase 1, v0.23.0): hard token
	// ceiling per benchmark for thrash detection on free (pricing=0) local
	// models. When cumulative input+output tokens exceed this value mid-stream,
	// executors must terminate the subprocess and set Result.ThrashKilledAt to
	// the observed token count. 0 = unlimited (legacy behaviour). Distinct
	// from Budget (cost-based) because local Ollama runs have $0 cost and
	// would never trigger cost-based abort regardless of how much they thrash.
	MaxTokensPerBench int

	// MaxOutputTokens (M-OLLAMA-PER-MODEL-MAX-TOKENS) is the PER-REQUEST output
	// budget from the model registry's max_output_tokens — the model's declared
	// strength. 0 = unset (use the harness/provider default). Executors that drive
	// a separate runtime (e.g. motoko's ollama path) forward this so a reasoning
	// model isn't truncated mid-<think> by a small default. Distinct from
	// MaxTokensPerBench (a cumulative thrash ceiling).
	MaxOutputTokens int
}

// PluginsConfig specifies third-party plugins to install before execution.
// This is a copy of coordinator.PluginsConfig to avoid circular imports.
type PluginsConfig struct {
	Marketplaces []string // Marketplaces to register (e.g., "anthropics/claude-code")
	Install      []string // Plugins to install (e.g., "frontend-design@anthropics-claude-code")
}

// Result is the normalized execution result
type Result struct {
	Success bool   // Overall success
	Output  string // Final text output from agent
	Error   string // Error message if failed

	// Metrics
	DurationMS    int     // Total execution time in milliseconds
	NumTurns      int     // Conversation turns
	ToolCallCount int     // Number of tool invocations (file edits, bash, etc.)
	CostUSD       float64 // Total cost in USD

	// Token usage
	InputTokens              int
	OutputTokens             int
	CacheReadInputTokens     int
	CacheCreationInputTokens int

	// Session info
	SessionID  string // Provider's session identifier
	Transcript string // Full conversation log

	// Artifacts
	FilesCreated  []string // Files created in workspace
	FilesModified []string // Files modified

	// Provider-specific data
	ProviderData map[string]any // Raw provider response data

	// Cost-and-speed budget metrics (M-EVAL-COST-AND-SPEED-BUDGETS, v0.15.1).
	// Populated by executors that wire Task.Budget. Zero values mean "not measured".
	CostKilledAt   float64 // > 0 if execution stopped because cost budget exceeded; 0 otherwise
	FirstAttemptMs int64   // ms from task start to first solution submission (-1 = never)
	SuccessAtMs    int64   // ms from task start to first passing solution (-1 = never)
	TokensPerSec   float64 // OutputTokens / generation_seconds (0 if duration not measured)

	// Thrash abort metric (M-EVAL-OS-LONGITUDINAL Phase 1, v0.23.0).
	// > 0 if execution stopped because cumulative tokens exceeded Task.MaxTokensPerBench.
	// 0 means either the limit wasn't reached or no limit was set.
	ThrashKilledAt int

	// FinishReason (M-EVAL-SWEET-SPOT, v0.19.0) is the structured executor
	// stop signal: "stop" (normal completion), "cost_exhausted" (motoko cost
	// cap), "step_exhausted" (agent ran out of turns), "timeout", "error",
	// or "". The eval harness consumes this via CategorizeAgentError to
	// promote ambiguous api_error rows into typed buckets.
	FinishReason string
}

// TokenUsage captures token metrics
type TokenUsage struct {
	InputTokens              int
	OutputTokens             int
	CacheReadInputTokens     int
	CacheCreationInputTokens int
}

// Capability flags for feature detection
type Capability string

const (
	CapStreaming         Capability = "streaming"
	CapToolControl       Capability = "tool_control"
	CapSessionResume     Capability = "session_resume"
	CapApprovalFlow      Capability = "approval_flow"
	CapGitHubIntegration Capability = "github_integration"
	CapLocalWorkspace    Capability = "local_workspace"
	CapStructuredOutput  Capability = "structured_output"

	// CapRemoteSandbox means the executor runs the agent in an isolated
	// environment that does NOT share the caller's filesystem (e.g.
	// Vertex Managed Agents API runs the agent in a Google-hosted Linux
	// sandbox). Callers that need to receive file artifacts from the agent
	// must either:
	//   (a) Provide an upload/download channel out-of-band (e.g. GCS), or
	//   (b) Instruct the agent to emit final artifacts in its text response
	//       and parse them from Result.Output.
	// The eval harness handles this by appending a "dump code as fenced
	// block" instruction to the system prompt and extracting from Output
	// after the run. Other backend callers can ignore this capability if
	// they don't need file bridging.
	CapRemoteSandbox Capability = "remote_sandbox"
)

// EventHandler receives streaming events during execution
type EventHandler interface {
	OnTurnStart(turnNum int)
	OnText(text string)
	OnToolUse(toolName string, input string)
	OnToolResult(toolName string, output string)
	OnTurnEnd(turnNum int)
	OnError(err error)
}

// ContextAwareHandler is an optional interface for handlers that need
// the executor's trace context for proper span hierarchy.
// When implemented, the executor calls SetContext after creating its span,
// so child spans created by the handler are properly nested.
type ContextAwareHandler interface {
	EventHandler
	SetContext(ctx context.Context)
}

// MetricsHandler is an optional interface for handlers that want execution metrics.
// When implemented, the executor calls OnMetrics after parsing the final result,
// allowing the handler to broadcast cost/token data before the executor returns.
// This is used by the cloud event handler to publish cost data to Pub/Sub.
type MetricsHandler interface {
	EventHandler
	OnMetrics(metrics ExecutionMetrics)
}

// ExecutionMetrics contains cost and token data from the executor.
type ExecutionMetrics struct {
	NumTurns     int
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	DurationMS   int
	SessionID    string
	Success      bool
}

// CostModel contains pricing information
type CostModel struct {
	ProviderName    string
	InputTokenCost  float64 // Cost per 1K input tokens
	OutputTokenCost float64 // Cost per 1K output tokens
	CacheReadCost   float64 // Cost per 1K cache read tokens
	CacheWriteCost  float64 // Cost per 1K cache write tokens
	MinimumCharge   float64 // Minimum per-request charge
}

// CalculateCost computes total cost from token usage
func (c *CostModel) CalculateCost(usage TokenUsage) float64 {
	inputCost := float64(usage.InputTokens) / 1000.0 * c.InputTokenCost
	outputCost := float64(usage.OutputTokens) / 1000.0 * c.OutputTokenCost
	cacheReadCost := float64(usage.CacheReadInputTokens) / 1000.0 * c.CacheReadCost

	total := inputCost + outputCost + cacheReadCost
	if total < c.MinimumCharge {
		return c.MinimumCharge
	}
	return total
}

// NoOpEventHandler is a no-op implementation of EventHandler
type NoOpEventHandler struct{}

func (h *NoOpEventHandler) OnTurnStart(turnNum int)                     {}
func (h *NoOpEventHandler) OnText(text string)                          {}
func (h *NoOpEventHandler) OnToolUse(toolName string, input string)     {}
func (h *NoOpEventHandler) OnToolResult(toolName string, output string) {}
func (h *NoOpEventHandler) OnTurnEnd(turnNum int)                       {}
func (h *NoOpEventHandler) OnError(err error)                           {}
