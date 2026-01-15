package coordinator

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CoordinatorConfig is the coordinator section of the global config file.
type CoordinatorConfig struct {
	Agents          []*AgentConfig    `yaml:"agents" json:"agents"`
	DefaultProvider string            `yaml:"default_provider" json:"default_provider"`
	MergeBranch     string            `yaml:"merge_branch" json:"merge_branch"` // Target branch for approvals (default: "dev")
	GitHubSync      *GitHubSyncConfig `yaml:"github_sync" json:"github_sync"`
}

// GitHubSyncConfig configures automatic GitHub issue import.
type GitHubSyncConfig struct {
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	IntervalSecs      int      `yaml:"interval_secs" json:"interval_secs"`               // Default: 300 (5 min)
	WatchLabels       []string `yaml:"watch_labels" json:"watch_labels"`                 // Filter by labels
	TargetInbox       string   `yaml:"target_inbox" json:"target_inbox"`                 // Where to send imported issues
	ResyncLabels      bool     `yaml:"resync_labels" json:"resync_labels"`               // Re-check labels on imported messages
	ResyncIntervalSec int      `yaml:"resync_interval_secs" json:"resync_interval_secs"` // Default: 3600 (1 hour)
}

// ConfigFile represents the full ~/.ailang/config.yaml structure.
// Only the coordinator section is defined here; other sections are ignored.
type ConfigFile struct {
	Coordinator *CoordinatorConfig `yaml:"coordinator"`
}

// DefaultCoordinatorConfig returns a minimal default configuration.
func DefaultCoordinatorConfig() *CoordinatorConfig {
	return &CoordinatorConfig{
		DefaultProvider: "claude",
		Agents: []*AgentConfig{
			{
				ID:                "coordinator",
				Label:             "Coordinator Agent",
				Inbox:             "coordinator",
				Workspace:         ".",
				Capabilities:      []string{"code", "test", "docs"},
				Provider:          "claude",
				SessionContinuity: true,
			},
		},
		GitHubSync: &GitHubSyncConfig{
			Enabled:      false,
			IntervalSecs: 300,
			TargetInbox:  "coordinator",
		},
	}
}

// LoadCoordinatorConfig loads the coordinator configuration from ~/.ailang/config.yaml.
// If the file doesn't exist, returns a default configuration.
// If the file exists but has no coordinator section, returns a default configuration.
func LoadCoordinatorConfig() (*CoordinatorConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return DefaultCoordinatorConfig(), nil
	}

	configPath := filepath.Join(homeDir, ".ailang", "config.yaml")
	return LoadCoordinatorConfigFrom(configPath)
}

// LoadCoordinatorConfigFrom loads the coordinator configuration from a specific path.
func LoadCoordinatorConfigFrom(configPath string) (*CoordinatorConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, use defaults
			return DefaultCoordinatorConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Coordinator == nil {
		// No coordinator section, use defaults
		return DefaultCoordinatorConfig(), nil
	}

	// Validate and apply defaults
	cfg := config.Coordinator
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = "claude"
	}

	// Apply global merge branch default first (so agents can inherit it)
	if cfg.MergeBranch == "" {
		cfg.MergeBranch = "dev"
	}

	// Apply defaults to agents
	for _, agent := range cfg.Agents {
		if agent.Provider == "" {
			agent.Provider = cfg.DefaultProvider
		}
		if agent.MaxConcurrentTasks == 0 {
			agent.MaxConcurrentTasks = 1 // Default: 1 task at a time
		}
		// Per-agent merge branch - inherits from global if not set
		if agent.MergeBranch == "" {
			agent.MergeBranch = cfg.MergeBranch
		}
	}

	// Apply defaults to GitHub sync
	if cfg.GitHubSync != nil {
		if cfg.GitHubSync.IntervalSecs == 0 {
			cfg.GitHubSync.IntervalSecs = 300
		}
		if cfg.GitHubSync.TargetInbox == "" {
			cfg.GitHubSync.TargetInbox = "coordinator"
		}
	}

	return cfg, nil
}

// LoadAgentRegistry loads agents from config and returns a populated registry.
func LoadAgentRegistry() (*AgentRegistry, error) {
	cfg, err := LoadCoordinatorConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load coordinator config: %w", err)
	}

	registry := NewAgentRegistry()
	for _, agent := range cfg.Agents {
		if err := registry.Register(agent); err != nil {
			return nil, fmt.Errorf("failed to register agent %q: %w", agent.ID, err)
		}
	}

	return registry, nil
}

// LoadAgentRegistryFrom loads agents from a specific config path.
func LoadAgentRegistryFrom(configPath string) (*AgentRegistry, error) {
	cfg, err := LoadCoordinatorConfigFrom(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load coordinator config from %q: %w", configPath, err)
	}

	registry := NewAgentRegistry()
	for _, agent := range cfg.Agents {
		if err := registry.Register(agent); err != nil {
			return nil, fmt.Errorf("failed to register agent %q: %w", agent.ID, err)
		}
	}

	return registry, nil
}

// SampleAgentConfig returns a sample configuration string for documentation.
func SampleAgentConfig() string {
	return `# Coordinator agent configuration
# Place in ~/.ailang/config.yaml

coordinator:
  default_provider: claude  # Default AI provider for agents

  agents:
    - id: coordinator
      label: "Main Coordinator"
      inbox: coordinator
      workspace: /path/to/main/project
      capabilities: [code, test, docs, research]
      provider: claude
      session_continuity: true
      max_concurrent_tasks: 1

    - id: sprint-planner
      label: "Sprint Planner"
      inbox: sprint-planner
      workspace: /path/to/main/project
      capabilities: [research, docs]
      provider: claude
      trigger_on_complete: [sprint-executor]
      auto_approve_handoffs: false  # Require approval for handoffs
      session_continuity: true

    - id: sprint-executor
      label: "Sprint Executor"
      inbox: sprint-executor
      workspace: /path/to/main/project
      capabilities: [code, test]
      provider: claude
      auto_merge: false  # Require approval before merge
      session_continuity: true

    # Script agent for deterministic workflows (v0.6.4+)
    # Runs shell scripts instead of AI - useful for evals, deploys, syncs
    - id: eval-runner
      label: "Eval Runner"
      inbox: eval-runner
      workspace: /path/to/main/project
      invoke:
        type: script                    # Run script, not AI
        command: ./scripts/run-eval.sh  # Script to execute
        env_from_payload: true          # JSON payload becomes env vars
        timeout: 2h                     # Long timeout for evals
      output_markers: ["EVAL_RESULT:", "PASS_RATE:"]
      trigger_on_complete: []           # End of pipeline

  github_sync:
    enabled: true
    interval_secs: 300  # Check every 5 minutes
    watch_labels: [from:external, bug, feature]
    target_inbox: coordinator
`
}
