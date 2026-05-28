package coordinator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// defaultConfigPath returns the AILANG config file path.
// Checks AILANG_CONFIG env var first (for Cloud Run), falls back to ~/.ailang/config.yaml.
func defaultConfigPath() string {
	if p := os.Getenv("AILANG_CONFIG"); p != "" {
		return p
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".ailang", "config.yaml")
}

// CoordinatorConfig is the coordinator section of the global config file.
type CoordinatorConfig struct {
	Agents          []*AgentConfig    `yaml:"agents" json:"agents"`
	DefaultProvider string            `yaml:"default_provider" json:"default_provider"`
	ClaudePath      string            `yaml:"claude_path" json:"claude_path,omitempty"` // Explicit path to Claude CLI binary (empty = auto-detect: native > PATH > NVM)
	MergeBranch     string            `yaml:"merge_branch" json:"merge_branch"`         // Target branch for approvals (default: "dev")
	GitHubSync      *GitHubSyncConfig `yaml:"github_sync" json:"github_sync"`

	// PluginRepo is a git URL for a shared skills plugin (M-CLOUD-PLUGIN-SKILLS, v0.9.1).
	// In cloud mode, this repo is cloned and passed as --plugin-dir to Claude CLI.
	// Example: "https://github.com/sunholo-data/ailang_bootstrap.git"
	PluginRepo string `yaml:"plugin_repo" json:"plugin_repo,omitempty"`

	// DevMode disables stale task detector and approval watcher to reduce
	// Firestore reads during local development. (M-COST1)
	DevMode bool `yaml:"dev_mode" json:"dev_mode,omitempty"`

	// Triage configures the auto-triage router that promotes inbound
	// bug/feature messages to the design-doc-creator inbox
	// (M-MSG-TRIAGE-ROUTER). Opt-in: nil or Enabled=false means off.
	Triage *TriageConfig `yaml:"triage" json:"triage,omitempty"`
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
	Firebase    *FirebaseConfig    `yaml:"firebase"`
	Workspaces  *WorkspacesConfig  `yaml:"workspaces"`
}

// WorkspacesConfig contains workspace-related configuration for access control.
type WorkspacesConfig struct {
	Mappings         []WorkspaceMapping `yaml:"mappings" json:"mappings"`
	DefaultWorkspace string             `yaml:"default_workspace" json:"default_workspace"`
	DeriveFromPath   bool               `yaml:"derive_from_path" json:"derive_from_path"` // If true, derive workspace from path when not matched
}

// WorkspaceMapping defines a path pattern to workspace ID mapping.
type WorkspaceMapping struct {
	Pattern   string `yaml:"pattern" json:"pattern"`     // Glob pattern (e.g., "*/dev/sunholo/ailang")
	Workspace string `yaml:"workspace" json:"workspace"` // Workspace ID (e.g., "sunholo-data/ailang")
}

// FirebaseConfig contains Firebase authentication settings.
type FirebaseConfig struct {
	ProjectID string `yaml:"project_id"` // GCP/Firebase project ID (e.g., "ailang-dev")
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

// LoadBudgetsConfig loads budget configuration from ~/.ailang/config.yaml.
// Respects AILANG_CONFIG env var for Cloud Run deployments.
func LoadBudgetsConfig() (*BudgetsConfig, error) {
	configPath := defaultConfigPath()
	if configPath == "" {
		return DefaultBudgetsConfig(), nil
	}
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

// LoadFirebaseConfig loads Firebase configuration from ~/.ailang/config.yaml.
// Returns nil if no Firebase config is set (Firebase auth will be disabled).
func LoadFirebaseConfig() *FirebaseConfig {
	configPath := defaultConfigPath()
	if configPath == "" {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil // No config file
	}

	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil // Invalid config
	}

	return config.Firebase
}

// LoadWorkspacesConfig loads workspace configuration from ~/.ailang/config.yaml.
// Returns a default configuration if no config is set.
func LoadWorkspacesConfig() *WorkspacesConfig {
	configPath := defaultConfigPath()
	if configPath == "" {
		return DefaultWorkspacesConfig()
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultWorkspacesConfig()
	}

	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return DefaultWorkspacesConfig()
	}

	if config.Workspaces == nil {
		return DefaultWorkspacesConfig()
	}

	// Apply defaults
	if config.Workspaces.DefaultWorkspace == "" {
		config.Workspaces.DefaultWorkspace = "public"
	}

	return config.Workspaces
}

// DefaultWorkspacesConfig returns minimal default workspace mappings.
// Only internal patterns are hardcoded - user projects should be in config.
// For unmapped paths, DeriveWorkspaceFromPath() extracts workspace ID from path.
func DefaultWorkspacesConfig() *WorkspacesConfig {
	return &WorkspacesConfig{
		Mappings: []WorkspaceMapping{
			// Internal patterns only - user project mappings go in config
			{Pattern: "*/.eval_workspace/*", Workspace: "eval_workspace"},
			{Pattern: "*/worktrees/*", Workspace: "coordinator_worktrees"},
		},
		DefaultWorkspace: "", // Empty means use derived workspace
		DeriveFromPath:   true,
	}
}

// DeriveWorkspaceFromPath extracts a workspace ID from a file path.
// Uses the last two meaningful path segments (parent/basename) as the workspace ID.
// This is portable across different directory structures.
//
// Examples:
//   - /Users/mark/dev/sunholo/ailang -> sunholo/ailang
//   - /home/user/projects/rockwool/ROCKGAP -> rockwool/ROCKGAP
//   - /path/to/TwilightGame -> to/TwilightGame (or just TwilightGame if only 1 meaningful segment)
//   - /tmp/foo -> foo
func DeriveWorkspaceFromPath(path string) string {
	if path == "" || path == "unknown" {
		return "unknown"
	}

	// Clean path and split into components
	path = filepath.Clean(path)
	parts := strings.Split(path, string(filepath.Separator))

	// Collect meaningful path segments (skip empty, hidden, and common temp dirs)
	var meaningful []string
	for _, p := range parts {
		if p == "" || p == "tmp" || p == "var" || p == "folders" || strings.HasPrefix(p, ".") {
			continue
		}
		// Also skip common home/user path segments
		if p == "Users" || p == "home" || p == "mark" {
			continue
		}
		meaningful = append(meaningful, p)
	}

	if len(meaningful) == 0 {
		return "unknown"
	}

	// Use last two segments as org/repo (or just last one if only one exists)
	if len(meaningful) >= 2 {
		return meaningful[len(meaningful)-2] + "/" + meaningful[len(meaningful)-1]
	}
	return meaningful[len(meaningful)-1]
}

// BuildWorkspaceMappingSQL generates a SQL CASE statement for mapping file paths to workspace IDs.
// Used by the analytics backend to map process.cwd values to Firestore workspace IDs.
//
// When DeriveFromPath is true, unmapped paths are derived using SQL expressions:
// - Paths with /dev/{org}/{repo} return "org/repo"
// - Other paths return "unknown"
func (c *WorkspacesConfig) BuildWorkspaceMappingSQL(cwdColumn string) string {
	if c == nil {
		return "'unknown'"
	}

	var cases []string
	for _, m := range c.Mappings {
		// Convert glob pattern to SQL LIKE pattern:
		// - "*" at start/end becomes "%"
		// - Internal "*" becomes "%"
		pattern := m.Pattern
		pattern = strings.ReplaceAll(pattern, "*", "%")

		cases = append(cases, fmt.Sprintf(
			"WHEN %s LIKE '%s' THEN '%s'",
			cwdColumn, pattern, m.Workspace))
	}

	// ELSE clause: either use default or derive from path
	if c.DeriveFromPath {
		// For unmapped paths, use the raw cwd value as workspace.
		// The config mappings handle the main workspaces.
		// Label formatting will make raw paths human-readable.
		deriveSql := fmt.Sprintf(`ELSE %s`, cwdColumn)
		cases = append(cases, deriveSql)
	} else {
		defaultWs := c.DefaultWorkspace
		if defaultWs == "" {
			defaultWs = "unknown"
		}
		cases = append(cases, fmt.Sprintf("ELSE '%s'", defaultWs))
	}

	return "CASE\n" + strings.Join(cases, "\n") + "\nEND"
}

// GetWorkspaceLabel returns a human-friendly label for a workspace ID.
// For internal workspaces, returns predefined labels.
// For user workspaces (org/repo format), returns a formatted label.
// For raw paths (from DeriveFromPath fallback), derives and formats the label.
func (c *WorkspacesConfig) GetWorkspaceLabel(workspaceID string) string {
	// Internal workspace labels
	internalLabels := map[string]string{
		"eval_workspace":        "Eval Benchmarks",
		"coordinator_worktrees": "Coordinator Tasks",
		"unknown":               "No Workspace",
	}
	if label, ok := internalLabels[workspaceID]; ok {
		return label
	}

	// Check if this looks like a raw file path (starts with / or has more than 2 slashes)
	if strings.HasPrefix(workspaceID, "/") || strings.Count(workspaceID, "/") > 1 {
		// Derive workspace from path and use that
		derived := DeriveWorkspaceFromPath(workspaceID)
		if parts := strings.Split(derived, "/"); len(parts) == 2 {
			return formatWorkspaceLabel(parts[1])
		}
		return formatWorkspaceLabel(derived)
	}

	// For org/repo format, make the repo name the label
	if parts := strings.Split(workspaceID, "/"); len(parts) == 2 {
		return formatWorkspaceLabel(parts[1])
	}

	// For single-segment workspace, format it nicely
	return formatWorkspaceLabel(workspaceID)
}

// GetPathPatternsForWorkspace returns SQL LIKE patterns that match a workspace ID.
// Used for filtering spans by workspace when the filter is an org/repo ID.
// Returns nil if no matching mappings found (caller should use fallback).
//
// Example: For workspace "MarkEdmondson1234/TwilightGame" with mapping
// {Pattern: "*/TwilightGame*", Workspace: "MarkEdmondson1234/TwilightGame"},
// returns ["%/TwilightGame%"] as the pattern to match process.cwd values.
func (c *WorkspacesConfig) GetPathPatternsForWorkspace(workspaceID string) []string {
	if c == nil || workspaceID == "" {
		return nil
	}

	var patterns []string
	for _, m := range c.Mappings {
		if m.Workspace == workspaceID {
			// Convert glob pattern to SQL LIKE pattern
			pattern := strings.ReplaceAll(m.Pattern, "*", "%")
			patterns = append(patterns, pattern)
		}
	}

	// If no explicit mapping, try to derive patterns from the workspace ID
	// For org/repo format, generate patterns that would match the repo name
	if len(patterns) == 0 && c.DeriveFromPath {
		// Extract the repo name from org/repo format
		parts := strings.Split(workspaceID, "/")
		if len(parts) >= 1 {
			repoName := parts[len(parts)-1]
			// Generate pattern that matches paths ending with this repo name
			patterns = append(patterns, "%/"+repoName)
			patterns = append(patterns, "%/"+repoName+"/%")
		}
	}

	return patterns
}

// formatWorkspaceLabel converts a workspace name to a human-readable label.
// Examples: "ailang" -> "Ailang", "ROCKGAP" -> "ROCKGAP", "stapledons_voyage" -> "Stapledons Voyage"
//
//	"TwilightGame" -> "Twilight Game" (splits camel case)
func formatWorkspaceLabel(name string) string {
	if name == "" {
		return "Unknown"
	}

	// If all uppercase, keep it (like ROCKGAP, ROCKGPT)
	if strings.ToUpper(name) == name && len(name) > 1 {
		return name
	}

	// Insert spaces before uppercase letters (camel case -> spaces)
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if previous char was lowercase (camel case boundary)
			prev := rune(name[i-1])
			if prev >= 'a' && prev <= 'z' {
				result.WriteRune(' ')
			}
		}
		result.WriteRune(r)
	}
	name = result.String()

	// Replace underscores/dashes with spaces
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	// Title case each word
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
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
	configPath := defaultConfigPath()
	if configPath == "" {
		return DefaultCoordinatorConfig(), nil
	}
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
      trigger_on_complete: [sprint-evaluator]  # Evaluator judges implementation quality
      auto_merge: false  # Require approval before merge
      session_continuity: true

    - id: sprint-evaluator
      label: "Sprint Evaluator"
      inbox: sprint-evaluator
      workspace: /path/to/main/project
      capabilities: [review, test, docs]
      provider: claude
      trigger_on_complete: []  # End of chain on pass
      auto_merge: false
      session_continuity: false  # Stateless per evaluation round

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
