package coordinator

import (
	"fmt"
	"sync"
)

// AgentConfig represents a configured agent in the coordinator system.
// Each agent has an inbox, workspace, and capabilities for task execution.
type AgentConfig struct {
	ID                   string   `yaml:"id" json:"id"`
	Label                string   `yaml:"label" json:"label"`
	Inbox                string   `yaml:"inbox" json:"inbox"`                                   // Message inbox to watch
	Workspace            string   `yaml:"workspace" json:"workspace"`                           // Base directory for worktrees
	Capabilities         []string `yaml:"capabilities" json:"capabilities"`                     // e.g., ["code", "research", "docs"]
	TriggerOnComplete    []string `yaml:"trigger_on_complete" json:"trigger_on_complete"`       // Agent IDs to trigger when this agent completes
	AutoApproveHandoffs  bool     `yaml:"auto_approve_handoffs" json:"auto_approve_handoffs"`   // Skip approval for agent-to-agent handoffs
	AutoMerge            bool     `yaml:"auto_merge" json:"auto_merge"`                         // Automatically merge approved work
	Provider             string   `yaml:"provider" json:"provider"`                             // "claude" or "gemini"
	MaxConcurrentTasks   int      `yaml:"max_concurrent_tasks" json:"max_concurrent_tasks"`     // 0 = unlimited
	SessionContinuity    bool     `yaml:"session_continuity" json:"session_continuity"`         // Use --resume for Claude Code / --conversation-id for Gemini
}

// AgentRegistry manages the set of configured agents.
// It provides thread-safe lookup by ID or inbox.
type AgentRegistry struct {
	mu     sync.RWMutex
	agents map[string]*AgentConfig // key: agent ID
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
