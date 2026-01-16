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
// Supports both single-repo (legacy) and multi-repo configurations.
type GitHubSyncConfig struct {
	// Legacy single-repo fields (for backwards compatibility)
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	IntervalSecs      int      `yaml:"interval_secs" json:"interval_secs"`               // Default: 300 (5 min)
	WatchLabels       []string `yaml:"watch_labels" json:"watch_labels"`                 // Filter by labels
	TargetInbox       string   `yaml:"target_inbox" json:"target_inbox"`                 // Where to send imported issues
	ResyncLabels      bool     `yaml:"resync_labels" json:"resync_labels"`               // Re-check labels on imported messages
	ResyncIntervalSec int      `yaml:"resync_interval_secs" json:"resync_interval_secs"` // Default: 3600 (1 hour)

	// Multi-repo configuration (v0.6.6+)
	// If Repos is non-empty, uses multi-repo mode (ignores legacy fields above except ResyncLabels)
	Repos []RepoSyncConfig `yaml:"repos" json:"repos"`
}

// RepoSyncConfig configures GitHub sync for a single repository.
type RepoSyncConfig struct {
	Repo         string             `yaml:"repo" json:"repo"`                   // GitHub repo (owner/repo)
	Enabled      bool               `yaml:"enabled" json:"enabled"`             // Enable sync for this repo
	IntervalSecs int                `yaml:"interval_secs" json:"interval_secs"` // Override default interval
	WatchLabels  []string           `yaml:"watch_labels" json:"watch_labels"`   // Filter by labels
	TargetInbox  string             `yaml:"target_inbox" json:"target_inbox"`   // Default inbox for this repo
	LabelRouting []LabelRouteConfig `yaml:"label_routing" json:"label_routing"` // Route by label prefix
}

// LabelRouteConfig maps a label prefix to a target inbox.
type LabelRouteConfig struct {
	LabelPrefix string `yaml:"label_prefix" json:"label_prefix"` // Match labels starting with this
	Target      string `yaml:"target" json:"target"`             // Route to this inbox
}

// GetRepos returns the list of repos to sync, handling backwards compatibility.
// If Repos is non-empty, returns it directly.
// Otherwise, constructs a single-repo config from legacy fields.
func (c *GitHubSyncConfig) GetRepos(defaultRepo string) []RepoSyncConfig {
	if len(c.Repos) > 0 {
		return c.Repos
	}
	// Legacy single-repo mode
	if !c.Enabled {
		return nil
	}
	return []RepoSyncConfig{
		{
			Repo:         defaultRepo,
			Enabled:      c.Enabled,
			IntervalSecs: c.IntervalSecs,
			WatchLabels:  c.WatchLabels,
			TargetInbox:  c.TargetInbox,
		},
	}
}

// ConfigFile represents the full ~/.ailang/config.yaml structure.
type ConfigFile struct {
	Coordinator *CoordinatorConfig `yaml:"coordinator"`
	Budgets     *BudgetsConfig     `yaml:"budgets"`
}

// BudgetsConfig represents budget limits from config.yaml
type BudgetsConfig struct {
	Global    *GlobalBudget             `yaml:"global"`
	Providers map[string]*ProviderLimit `yaml:"providers"`
}

// GlobalBudget defines default budget limits
type GlobalBudget struct {
	WorkspaceBudget  float64 `yaml:"workspace_budget"`
	DailyBudget      float64 `yaml:"daily_budget"`
	TaskMaxCost      float64 `yaml:"task_max_cost"`
	WarningThreshold float64 `yaml:"warning_threshold"`
}

// ProviderLimit defines per-provider budget overrides
type ProviderLimit struct {
	DailyBudget      float64 `yaml:"daily_budget"`
	TaskMaxCost      float64 `yaml:"task_max_cost"`
	HardLimit        bool    `yaml:"hard_limit"`
	WarningThreshold float64 `yaml:"warning_threshold"`
}

// DefaultBudgetsConfig returns sensible default budget limits
func DefaultBudgetsConfig() *BudgetsConfig {
	return &BudgetsConfig{
		Global: &GlobalBudget{
			WorkspaceBudget:  100.0, // $100 workspace budget
			DailyBudget:      50.0,  // $50 daily budget
			TaskMaxCost:      25.0,  // $25 max per task
			WarningThreshold: 0.8,   // Warn at 80% usage
		},
		Providers: map[string]*ProviderLimit{
			"claude": {
				DailyBudget: 30.0,
				TaskMaxCost: 15.0,
				HardLimit:   true,
			},
			"gemini": {
				DailyBudget: 20.0,
				TaskMaxCost: 10.0,
				HardLimit:   false,
			},
		},
	}
}

// LoadBudgetsConfig loads budget configuration from ~/.ailang/config.yaml
func LoadBudgetsConfig() (*BudgetsConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return DefaultBudgetsConfig(), nil
	}

	configPath := filepath.Join(homeDir, ".ailang", "config.yaml")
	return LoadBudgetsConfigFrom(configPath)
}

// LoadBudgetsConfigFrom loads budget configuration from a specific path
func LoadBudgetsConfigFrom(configPath string) (*BudgetsConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultBudgetsConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Budgets == nil {
		return DefaultBudgetsConfig(), nil
	}

	// Apply defaults
	budgets := config.Budgets
	if budgets.Global == nil {
		budgets.Global = DefaultBudgetsConfig().Global
	} else {
		// Apply individual defaults
		if budgets.Global.WorkspaceBudget == 0 {
			budgets.Global.WorkspaceBudget = 100.0
		}
		if budgets.Global.DailyBudget == 0 {
			budgets.Global.DailyBudget = 50.0
		}
		if budgets.Global.TaskMaxCost == 0 {
			budgets.Global.TaskMaxCost = 25.0
		}
		if budgets.Global.WarningThreshold == 0 {
			budgets.Global.WarningThreshold = 0.8
		}
	}

	return budgets, nil
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
		// Legacy single-repo defaults
		if cfg.GitHubSync.IntervalSecs == 0 {
			cfg.GitHubSync.IntervalSecs = 300
		}
		if cfg.GitHubSync.TargetInbox == "" {
			cfg.GitHubSync.TargetInbox = "coordinator"
		}
		// Multi-repo defaults
		for i := range cfg.GitHubSync.Repos {
			repo := &cfg.GitHubSync.Repos[i]
			if repo.IntervalSecs == 0 {
				repo.IntervalSecs = 300
			}
			if repo.TargetInbox == "" {
				repo.TargetInbox = "coordinator"
			}
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

  # GitHub sync - single repo (legacy)
  github_sync:
    enabled: true
    interval_secs: 300  # Check every 5 minutes
    watch_labels: [from:external, bug, feature]
    target_inbox: coordinator

  # GitHub sync - multi-repo (v0.6.6+)
  # github_sync:
  #   repos:
  #     - repo: sunholo-data/ailang
  #       enabled: true
  #       interval_secs: 300
  #       target_inbox: design-doc-creator
  #       label_routing:
  #         - label_prefix: "coordinator:bug"
  #           target: design-doc-creator
  #         - label_prefix: "coordinator:docs"
  #           target: coordinator
  #     - repo: sunholo-data/stapledons_voyage
  #       enabled: true
  #       interval_secs: 300
  #       target_inbox: stapledon-design-doc
  #       label_routing:
  #         - label_prefix: "feature"
  #           target: stapledon-design-doc
`
}
