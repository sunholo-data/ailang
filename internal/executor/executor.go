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
	ID           string            // Unique task identifier
	ParentTaskID string            // Parent task ID for hierarchy tracking (M-TASK-HIERARCHY)
	Directive    string            // The instruction/prompt
	SystemPrompt string            // Optional system-level context
	Workspace    string            // Working directory (local path)
	Timeout      time.Duration     // Hard ceiling execution timeout
	IdleTimeout  time.Duration     // Kill if no events for this long (0 = use default 3m)
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
