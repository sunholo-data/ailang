package coordinator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// InvokeConfig specifies how an agent should be invoked.
// Supports four types:
//   - "skill": Invoke a Claude Code skill (e.g., "/design-doc-creator")
//   - "agent": Send message to another agent (e.g., "sprint-planner")
//   - "prompt": Use custom prompt template with variable substitution
//   - "script": Execute a shell script with JSON payload as environment variables (v0.6.4+)
type InvokeConfig struct {
	Type         string `yaml:"type" json:"type"`                   // "skill", "agent", "prompt", or "script"
	Name         string `yaml:"name" json:"name"`                   // Skill/agent name (for skill/agent types)
	Template     string `yaml:"template" json:"template"`           // Custom template (for prompt type) - inline
	TemplateFile string `yaml:"template_file" json:"template_file"` // Path to template file (for prompt type) - v0.6.7+

	// Script-specific fields (v0.6.4+)
	// Used when Type == "script" for deterministic workflow execution
	Command        string `yaml:"command" json:"command,omitempty"`                   // Script path or inline command
	Shell          string `yaml:"shell" json:"shell,omitempty"`                       // Shell to use (default: /bin/sh)
	EnvFromPayload bool   `yaml:"env_from_payload" json:"env_from_payload,omitempty"` // Parse JSON payload as env vars
	Timeout        string `yaml:"timeout" json:"timeout,omitempty"`                   // Execution timeout (e.g., "30m", "2h")
	WorkingDir     string `yaml:"working_dir" json:"working_dir,omitempty"`           // Working directory (supports {{.Workspace}})
}

// ResolveTemplate returns the template content, loading from file if template_file is set.
// Priority: template_file > template (inline)
// Supports:
//   - Absolute paths: /path/to/template.md
//   - Home directory: ~/.ailang/templates/design-doc.md
//   - Relative to workspace: templates/design-doc.md (requires workspace param)
func (ic *InvokeConfig) ResolveTemplate(workspace string) (string, error) {
	// If template_file is set, load from file
	if ic.TemplateFile != "" {
		path := ic.TemplateFile

		// Expand ~ to home directory
		if strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get home directory: %w", err)
			}
			path = filepath.Join(home, path[2:])
		}

		// If not absolute, resolve relative to workspace
		if !filepath.IsAbs(path) && workspace != "" {
			path = filepath.Join(workspace, path)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read template file %q: %w", ic.TemplateFile, err)
		}
		return string(content), nil
	}

	// Fall back to inline template
	return ic.Template, nil
}

// ApprovalConfig specifies the approval workflow for an agent.
// Controls GitHub labels and comment templates for human-in-the-loop approval.
type ApprovalConfig struct {
	NeedsLabel            string `yaml:"needs_label" json:"needs_label"`                         // Label to add when awaiting approval (e.g., "needs-design-approval")
	ApprovedLabel         string `yaml:"approved_label" json:"approved_label"`                   // Label that triggers next stage (e.g., "design-approved")
	GithubCommentTemplate string `yaml:"github_comment_template" json:"github_comment_template"` // Template for GitHub comment on completion
}

// AgentConfig represents a configured agent in the coordinator system.
// Each agent has an inbox, workspace, and capabilities for task execution.
type AgentConfig struct {
	ID                  string   `yaml:"id" json:"id"`
	Label               string   `yaml:"label" json:"label"`
	Inbox               string   `yaml:"inbox" json:"inbox"`                                 // Message inbox to watch
	Workspace           string   `yaml:"workspace" json:"workspace"`                         // Base directory for worktrees
	Capabilities        []string `yaml:"capabilities" json:"capabilities"`                   // e.g., ["code", "research", "docs"]
	TriggerOnComplete   []string `yaml:"trigger_on_complete" json:"trigger_on_complete"`     // Agent IDs to trigger when this agent completes
	AutoApproveHandoffs bool     `yaml:"auto_approve_handoffs" json:"auto_approve_handoffs"` // Skip approval for agent-to-agent handoffs
	AutoMerge           bool     `yaml:"auto_merge" json:"auto_merge"`                       // Automatically merge approved work
	SkipApproval        bool     `yaml:"skip_approval" json:"skip_approval"`                 // Skip approval workflow entirely (for script agents)
	Provider            string   `yaml:"provider" json:"provider"`                           // "claude" or "gemini"
	MergeBranch         string   `yaml:"merge_branch" json:"merge_branch"`                   // Target branch for merges (e.g., "dev", "main")
	MaxConcurrentTasks  int      `yaml:"max_concurrent_tasks" json:"max_concurrent_tasks"`   // 0 = unlimited
	SessionContinuity   bool     `yaml:"session_continuity" json:"session_continuity"`       // Use --resume for Claude Code / --conversation-id for Gemini

	// Generic workflow configuration (v0.6.3+)
	Invoke           *InvokeConfig   `yaml:"invoke" json:"invoke,omitempty"`                       // How to invoke this agent
	OutputMarkers    []string        `yaml:"output_markers" json:"output_markers,omitempty"`       // Markers to extract from output (e.g., "DESIGN_DOC_PATH:")
	ArtifactPatterns []string        `yaml:"artifact_patterns" json:"artifact_patterns,omitempty"` // File patterns for artifacts (e.g., "*.md", "design_docs/**")
	Approval         *ApprovalConfig `yaml:"approval" json:"approval,omitempty"`                   // Approval workflow configuration

	// Per-agent system prompt (v0.8.0+)
	// Appended to the global meta-prompt for agent-specific instructions.
	SystemPrompt string `yaml:"system_prompt" json:"system_prompt,omitempty"`

	// Per-agent model override (v0.8.0+)
	// Controls which Claude model is used for this agent's tasks.
	// Examples: "opus", "sonnet", "haiku", "claude-opus-4-5-20251101"
	// If empty, falls back to the executor's default (currently "haiku").
	Model string `yaml:"model" json:"model,omitempty"`

	// Per-agent execution timeout (v0.8.1+)
	// Go duration string (e.g., "15m", "30m", "1h"). Default: 60m (hard ceiling).
	Timeout string `yaml:"timeout" json:"timeout,omitempty"`

	// Per-agent idle timeout (v0.8.1+)
	// Kill if no streaming events for this long. Default: 3m.
	// Distinguishes "agent is stuck" from "agent is working but slow".
	IdleTimeout string `yaml:"idle_timeout" json:"idle_timeout,omitempty"`
}

// AgentRegistry manages the set of configured agents.
// It provides thread-safe lookup by ID or inbox.
type AgentRegistry struct {
	mu      sync.RWMutex
	agents  map[string]*AgentConfig // key: agent ID
	byInbox map[string]*AgentConfig // key: inbox name
}

// NewAgentRegistry creates an empty agent registry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents:  make(map[string]*AgentConfig),
		byInbox: make(map[string]*AgentConfig),
	}
}

// Register adds an agent to the registry.
// Returns an error if an agent with the same ID or inbox already exists.
func (r *AgentRegistry) Register(agent *AgentConfig) error {
	if agent == nil {
		return fmt.Errorf("agent config is nil")
	}
	if agent.ID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if agent.Inbox == "" {
		return fmt.Errorf("agent inbox is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agent.ID]; exists {
		return fmt.Errorf("agent with ID %q already registered", agent.ID)
	}
	if _, exists := r.byInbox[agent.Inbox]; exists {
		return fmt.Errorf("agent with inbox %q already registered", agent.Inbox)
	}

	r.agents[agent.ID] = agent
	r.byInbox[agent.Inbox] = agent
	return nil
}

// GetAgentByID returns an agent by its ID.
// Returns nil if not found.
func (r *AgentRegistry) GetAgentByID(id string) *AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[id]
}

// GetAgentForInbox returns the agent configured to watch the given inbox.
// Returns nil if no agent watches that inbox.
func (r *AgentRegistry) GetAgentForInbox(inbox string) *AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byInbox[inbox]
}

// ListAgents returns all registered agents.
func (r *AgentRegistry) ListAgents() []*AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]*AgentConfig, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}

// ListInboxes returns all registered inbox names.
func (r *AgentRegistry) ListInboxes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inboxes := make([]string, 0, len(r.byInbox))
	for inbox := range r.byInbox {
		inboxes = append(inboxes, inbox)
	}
	return inboxes
}

// Count returns the number of registered agents.
func (r *AgentRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// Clear removes all registered agents.
func (r *AgentRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents = make(map[string]*AgentConfig)
	r.byInbox = make(map[string]*AgentConfig)
}

// Unregister removes an agent by ID.
// Returns an error if the agent is not found.
func (r *AgentRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[id]
	if !exists {
		return fmt.Errorf("agent %q not found", id)
	}

	delete(r.agents, id)
	delete(r.byInbox, agent.Inbox)
	return nil
}

// HasAgent returns true if an agent with the given ID is registered.
func (r *AgentRegistry) HasAgent(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.agents[id]
	return exists
}

// HasInbox returns true if an agent is registered for the given inbox.
func (r *AgentRegistry) HasInbox(inbox string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.byInbox[inbox]
	return exists
}

// Validate checks that the registry is internally consistent.
// Returns a list of issues found.
func (r *AgentRegistry) Validate() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var issues []string

	for id, agent := range r.agents {
		if agent.ID != id {
			issues = append(issues, fmt.Sprintf("agent ID mismatch: key=%q, agent.ID=%q", id, agent.ID))
		}
		if agent.Workspace == "" {
			issues = append(issues, fmt.Sprintf("agent %q has no workspace configured", id))
		}
		if len(agent.Capabilities) == 0 {
			issues = append(issues, fmt.Sprintf("agent %q has no capabilities configured", id))
		}
		// Validate trigger_on_complete references
		for _, targetID := range agent.TriggerOnComplete {
			if _, exists := r.agents[targetID]; !exists {
				issues = append(issues, fmt.Sprintf("agent %q triggers unknown agent %q", id, targetID))
			}
		}
	}

	return issues
}

// =============================================================================
// Default Configs (Backwards Compatibility for AILANG Workflow)
// =============================================================================
// These functions provide legacy defaults for agents without explicit config.
// New agents should use explicit Invoke, OutputMarkers, and Approval config in YAML.
// These defaults are used by GetEffective*() methods when YAML config is not set.

// DefaultInvokeConfig returns the default invoke config for known AILANG agent IDs.
// Returns nil for unknown agents (no default behavior).
// Used by GetEffectiveInvokeConfig() when agent has no explicit YAML config.
func DefaultInvokeConfig(agentID string) *InvokeConfig {
	switch agentID {
	case "design-doc-creator":
		return &InvokeConfig{
			Type: "skill",
			Name: "design-doc-creator",
		}
	case "sprint-planner":
		return &InvokeConfig{
			Type: "skill",
			Name: "sprint-planner",
		}
	case "sprint-executor":
		return &InvokeConfig{
			Type: "skill",
			Name: "sprint-executor",
		}
	default:
		return nil
	}
}

// DefaultOutputMarkers returns the default output markers for known AILANG agent IDs.
// Returns nil for unknown agents (no markers expected).
// Used by GetEffectiveOutputMarkers() when agent has no explicit YAML config.
// Consider using ArtifactPatterns + git diff instead for deterministic artifact discovery.
func DefaultOutputMarkers(agentID string) []string {
	switch agentID {
	case "design-doc-creator":
		return []string{"DESIGN_DOC_PATH:"}
	case "sprint-planner":
		return []string{"SPRINT_PLAN_PATH:", "SPRINT_JSON_PATH:"}
	case "sprint-executor":
		return []string{"IMPLEMENTATION_COMPLETE:", "BRANCH_NAME:", "FILES_CREATED:", "FILES_MODIFIED:"}
	default:
		return nil
	}
}

// DefaultArtifactPatterns returns the default file patterns for artifact discovery.
// These patterns are used with git diff to deterministically discover created/modified artifacts.
//
// For known AILANG agents, returns specific patterns for their typical outputs.
// For unknown agents, returns universal pattern "**/*" to discover ALL changed files.
// This ensures artifact discovery works for any agent without hardcoded configuration.
func DefaultArtifactPatterns(agentID string) []string {
	switch agentID {
	case "design-doc-creator":
		// Design docs only create .md files in design_docs/
		return []string{"design_docs/**/*.md"}
	case "sprint-planner":
		// Sprint planner creates design docs and JSON progress files
		return []string{"design_docs/**/*.md", ".ailang/state/sprints/*.json"}
	case "sprint-executor":
		// Sprint executor modifies many files - collect .go, .md, .json
		return []string{"**/*.go", "**/*.md", "**/*.json", "**/*.ail"}
	default:
		// Universal default: discover ALL changed files
		// No hardcoded agent IDs - works for any agent
		// GitHub comments will filter to .md files for human-readable summary
		return []string{"**/*"}
	}
}

// DefaultApprovalConfig returns the default approval config for known AILANG agent IDs.
// Returns nil for unknown agents (no approval workflow).
// Used by GetEffectiveApprovalConfig() when agent has no explicit YAML config.
func DefaultApprovalConfig(agentID string) *ApprovalConfig {
	switch agentID {
	case "design-doc-creator":
		return &ApprovalConfig{
			NeedsLabel:            "needs-design-approval",
			ApprovedLabel:         "design-approved",
			GithubCommentTemplate: "design_doc",
		}
	case "sprint-planner":
		return &ApprovalConfig{
			NeedsLabel:            "needs-sprint-approval",
			ApprovedLabel:         "sprint-approved",
			GithubCommentTemplate: "sprint_plan",
		}
	case "sprint-executor":
		return &ApprovalConfig{
			NeedsLabel:            "needs-implementation-approval",
			ApprovedLabel:         "implementation-approved",
			GithubCommentTemplate: "implementation",
		}
	default:
		return nil
	}
}

// GetEffectiveInvokeConfig returns the agent's invoke config, or defaults for known agents.
// Returns nil for unknown agents without explicit config.
//
// Note: Deprecation warnings for using defaults should be logged by the caller,
// as they have logger access. Check if result differs from explicit config.
func (a *AgentConfig) GetEffectiveInvokeConfig() *InvokeConfig {
	if a.Invoke != nil {
		return a.Invoke
	}
	return DefaultInvokeConfig(a.ID)
}

// GetEffectiveOutputMarkers returns the agent's output markers, or defaults for known agents.
func (a *AgentConfig) GetEffectiveOutputMarkers() []string {
	if len(a.OutputMarkers) > 0 {
		return a.OutputMarkers
	}
	return DefaultOutputMarkers(a.ID)
}

// GetEffectiveArtifactPatterns returns the agent's artifact patterns, or defaults for known agents.
// These patterns are used with git diff to discover created/modified files.
func (a *AgentConfig) GetEffectiveArtifactPatterns() []string {
	if len(a.ArtifactPatterns) > 0 {
		return a.ArtifactPatterns
	}
	return DefaultArtifactPatterns(a.ID)
}

// GetEffectiveTimeout returns the agent's configured hard ceiling timeout, or the default (60m).
// This is the maximum wall-clock time regardless of activity. Safe to call on nil receiver.
func (a *AgentConfig) GetEffectiveTimeout() time.Duration {
	if a != nil && a.Timeout != "" {
		if d, err := time.ParseDuration(a.Timeout); err == nil && d > 0 {
			return d
		}
	}
	return 60 * time.Minute
}

// GetEffectiveIdleTimeout returns the agent's configured idle timeout, or the default (3m).
// The agent is killed if no streaming events are produced for this duration.
// Safe to call on nil receiver.
func (a *AgentConfig) GetEffectiveIdleTimeout() time.Duration {
	if a != nil && a.IdleTimeout != "" {
		if d, err := time.ParseDuration(a.IdleTimeout); err == nil && d > 0 {
			return d
		}
	}
	return 3 * time.Minute
}

// GetEffectiveApprovalConfig returns the agent's approval config, or defaults for known agents.
func (a *AgentConfig) GetEffectiveApprovalConfig() *ApprovalConfig {
	if a.Approval != nil {
		return a.Approval
	}
	return DefaultApprovalConfig(a.ID)
}
