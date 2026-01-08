package coordinator

import (
	"fmt"
	"sync"
)

// InvokeConfig specifies how an agent should be invoked.
// Supports four types:
//   - "skill": Invoke a Claude Code skill (e.g., "/design-doc-creator")
//   - "agent": Send message to another agent (e.g., "sprint-planner")
//   - "prompt": Use custom prompt template with variable substitution
//   - "script": Execute a shell script with JSON payload as environment variables (v0.6.4+)
type InvokeConfig struct {
	Type     string `yaml:"type" json:"type"`         // "skill", "agent", "prompt", or "script"
	Name     string `yaml:"name" json:"name"`         // Skill/agent name (for skill/agent types)
	Template string `yaml:"template" json:"template"` // Custom template (for prompt type)

	// Script-specific fields (v0.6.4+)
	// Used when Type == "script" for deterministic workflow execution
	Command        string `yaml:"command" json:"command,omitempty"`                   // Script path or inline command
	Shell          string `yaml:"shell" json:"shell,omitempty"`                       // Shell to use (default: /bin/sh)
	EnvFromPayload bool   `yaml:"env_from_payload" json:"env_from_payload,omitempty"` // Parse JSON payload as env vars
	Timeout        string `yaml:"timeout" json:"timeout,omitempty"`                   // Execution timeout (e.g., "30m", "2h")
	WorkingDir     string `yaml:"working_dir" json:"working_dir,omitempty"`           // Working directory (supports {{.Workspace}})
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
	MaxConcurrentTasks  int      `yaml:"max_concurrent_tasks" json:"max_concurrent_tasks"`   // 0 = unlimited
	SessionContinuity   bool     `yaml:"session_continuity" json:"session_continuity"`       // Use --resume for Claude Code / --conversation-id for Gemini

	// Generic workflow configuration (v0.6.3+)
	Invoke           *InvokeConfig   `yaml:"invoke" json:"invoke,omitempty"`                       // How to invoke this agent
	OutputMarkers    []string        `yaml:"output_markers" json:"output_markers,omitempty"`       // Markers to extract from output (e.g., "DESIGN_DOC_PATH:")
	ArtifactPatterns []string        `yaml:"artifact_patterns" json:"artifact_patterns,omitempty"` // File patterns for artifacts (e.g., "*.md", "design_docs/**")
	Approval         *ApprovalConfig `yaml:"approval" json:"approval,omitempty"`                   // Approval workflow configuration
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
// Deprecated: New agents should use explicit Invoke, OutputMarkers, and Approval config.
// These defaults will be removed in v0.7.0.

// DefaultInvokeConfig returns the legacy invoke config for known AILANG agent IDs.
// Returns nil for unknown agents (no default behavior).
//
// Deprecated: Agents should have explicit InvokeConfig in YAML.
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

// DefaultOutputMarkers returns the legacy output markers for known AILANG agent IDs.
// Returns nil for unknown agents (no markers expected).
//
// Deprecated: Agents should have explicit OutputMarkers in YAML.
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

// DefaultArtifactPatterns returns the default file patterns for known AILANG agent IDs.
// These patterns are used with git diff to deterministically discover created/modified artifacts.
// Returns nil for unknown agents (no patterns = no artifact collection).
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
		return nil
	}
}

// DefaultApprovalConfig returns the legacy approval config for known AILANG agent IDs.
// Returns nil for unknown agents (no approval workflow).
//
// Deprecated: Agents should have explicit ApprovalConfig in YAML.
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

// GetEffectiveApprovalConfig returns the agent's approval config, or defaults for known agents.
func (a *AgentConfig) GetEffectiveApprovalConfig() *ApprovalConfig {
	if a.Approval != nil {
		return a.Approval
	}
	return DefaultApprovalConfig(a.ID)
}
