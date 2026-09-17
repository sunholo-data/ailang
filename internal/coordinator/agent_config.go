package coordinator

import (
	"errors"
	"fmt"

	"github.com/sunholo-data/ailang/internal/config"
)

// The coordinator's sections of ~/.ailang/config.yaml are read through the
// ONE loader in internal/config (M-V1-SIMPLIFY-S3 M3): one parse per process,
// AILANG_CONFIG honoured everywhere, and a broken file is an error from
// every reader rather than a default from some of them.

// loadSection decodes one top-level section of the config file into out.
// It returns (false, nil) when there is no config file or no such section,
// so the caller applies its defaults; a file that exists but does not parse,
// or a section that does not fit, is an error.
func loadSection(key string, out any) (bool, error) {
	f, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrConfigNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read config file: %w", err)
	}
	present, err := f.Section(key, out)
	if err != nil {
		return present, fmt.Errorf("failed to parse config file: %w", err)
	}
	return present, nil
}

// loadSectionFrom is loadSection for an explicit path (tests and
// `coordinator agent check --repo-config`).
func loadSectionFrom(path, key string, out any) (bool, error) {
	f, err := config.LoadFrom(path)
	if err != nil {
		if errors.Is(err, config.ErrConfigNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read config file: %w", err)
	}
	present, err := f.Section(key, out)
	if err != nil {
		return present, fmt.Errorf("failed to parse config file: %w", err)
	}
	return present, nil
}

// CoordinatorConfig is the coordinator section of the global config file.
type CoordinatorConfig struct {
	Agents []*AgentConfig `yaml:"agents" json:"agents"`

	// TriageOnlyInboxes are inboxes deliberately served by no agent because a
	// human triages them (M-MESSAGE-PLANE-FAIL-LOUD M2, decision D2).
	//
	// public-feedback is the canonical case: submit_feedback documents
	// auto_dispatch as "default false — files for human triage", so anonymous
	// input is never handed to something that acts on it, and Discord is the
	// routing (humanTriageInbox in internal/daemon/handlers.go). Declaring it
	// here is what makes "unrouted" mean INTENDED rather than FORGOTTEN.
	TriageOnlyInboxes []string `yaml:"triage_only_inboxes" json:"triage_only_inboxes,omitempty"`

	// Pipelines declare a stage chain once and bind it per project
	// (M-PIPELINE-RECONCILIATION M4, D2). ExpandPipelines materializes bindings
	// into AgentConfigs at load time; expanded agents behave identically to
	// hand-written entries.
	Pipelines []PipelineConfig `yaml:"pipelines" json:"pipelines,omitempty"`

	// model_routing was DELETED by M-MODEL-REGISTRY-SINGLE-SOURCE M7. The
	// registry (internal/modelreg, `roles:` in models.yml) answers "which model
	// runs this role?" now, so the table no longer has a second home here that
	// needs its own deploy. Proven inert before removal: zero of the 34 cloud
	// agents change resolution without it.

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

	// FeedbackGate configures the cost & abuse gate on the cloud dispatch path
	// (M-FEEDBACK-TRIAGE-GATE). Opt-in: nil or Enabled=false means off (full
	// pass-through, zero behavior change). DISTINCT from Triage above — see the
	// naming-disambiguation note in internal/feedbackgate.
	FeedbackGate *FeedbackGateConfig `yaml:"feedback_gate" json:"feedback_gate,omitempty"`
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

// The coordinator's sections of the config file are `coordinator:`,
// `budgets:`, `firebase:` and `workspaces:`; each loader below decodes its
// own through config.Load, so there is no whole-file struct here any more.

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

// LoadBudgetsConfig loads the budgets section of the config file
// (AILANG_CONFIG, else ~/.ailang/config.yaml). No file or no section means
// the defaults.
func LoadBudgetsConfig() (*BudgetsConfig, error) {
	budgets := &BudgetsConfig{}
	present, err := loadSection("budgets", budgets)
	if err != nil {
		return nil, err
	}
	if !present {
		return DefaultBudgetsConfig(), nil
	}
	return applyBudgetDefaults(budgets), nil
}

// LoadBudgetsConfigFrom is LoadBudgetsConfig for an explicit path.
func LoadBudgetsConfigFrom(configPath string) (*BudgetsConfig, error) {
	budgets := &BudgetsConfig{}
	present, err := loadSectionFrom(configPath, "budgets", budgets)
	if err != nil {
		return nil, err
	}
	if !present {
		return DefaultBudgetsConfig(), nil
	}
	return applyBudgetDefaults(budgets), nil
}

func applyBudgetDefaults(budgets *BudgetsConfig) *BudgetsConfig {
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
	return budgets
}

// LoadFirebaseConfig loads the firebase section of the config file. Returns
// nil when there is no file or no section (Firebase auth disabled). A file
// that does not parse is also nil here — the callers are UI defaults — but
// the coordinator daemon has already refused to start on it.
func LoadFirebaseConfig() *FirebaseConfig {
	fb := &FirebaseConfig{}
	present, err := loadSection("firebase", fb)
	if err != nil || !present {
		return nil
	}
	return fb
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

// LoadCoordinatorConfig loads the coordinator section of the config file
// (AILANG_CONFIG, else ~/.ailang/config.yaml). No file or no coordinator
// section returns the default configuration; a file that does not parse is
// an error.
func LoadCoordinatorConfig() (*CoordinatorConfig, error) {
	cfg := &CoordinatorConfig{}
	present, err := loadSection("coordinator", cfg)
	if err != nil {
		return nil, err
	}
	if !present {
		return DefaultCoordinatorConfig(), nil
	}
	return applyCoordinatorDefaults(cfg), nil
}

// LoadCoordinatorConfigFrom is LoadCoordinatorConfig for an explicit path.
func LoadCoordinatorConfigFrom(configPath string) (*CoordinatorConfig, error) {
	cfg, _, err := loadCoordinatorConfigDeclared(configPath)
	return cfg, err
}

// loadCoordinatorConfigDeclared also reports whether the FILE declared a
// `coordinator:` section, which is the difference between "these are the
// deployment's agents" and "these are AILANG's built-in defaults".
//
// The distinction is invisible in the returned config: a file with no
// coordinator section yields DefaultCoordinatorConfig(), which builds a registry
// holding exactly one agent (`coordinator`, measured 2026-09-17). A caller that
// shows that as "the registry" is presenting a built-in stub as a deployment.
// See LoadAgentRegistryFromDeclared.
func loadCoordinatorConfigDeclared(configPath string) (*CoordinatorConfig, bool, error) {
	cfg := &CoordinatorConfig{}
	present, err := loadSectionFrom(configPath, "coordinator", cfg)
	if err != nil {
		return nil, false, err
	}
	if !present {
		return DefaultCoordinatorConfig(), false, nil
	}
	return applyCoordinatorDefaults(cfg), true, nil
}

// applyCoordinatorDefaults validates and fills the defaults of a loaded
// coordinator section.
func applyCoordinatorDefaults(cfg *CoordinatorConfig) *CoordinatorConfig {
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

	return cfg
}

// LoadAgentRegistry loads agents from config and returns a populated registry.
func LoadAgentRegistry() (*AgentRegistry, error) {
	cfg, err := LoadCoordinatorConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load coordinator config: %w", err)
	}

	return buildRegistryFromConfig(cfg)
}

// buildRegistryFromConfig registers literal agents AND pipeline-expanded ones
// (M-PIPELINE-RECONCILIATION M4). Expansion errors are fatal: a config whose
// pipelines cannot expand is a config we do not understand.
func buildRegistryFromConfig(cfg *CoordinatorConfig) (*AgentRegistry, error) {
	registry := NewAgentRegistry()
	for _, agent := range cfg.Agents {
		if err := registry.Register(agent); err != nil {
			return nil, fmt.Errorf("failed to register agent %q: %w", agent.ID, err)
		}
	}
	expanded, err := cfg.ExpandPipelines()
	if err != nil {
		return nil, fmt.Errorf("pipeline expansion failed: %w", err)
	}
	for _, agent := range expanded {
		if err := registry.Register(agent); err != nil {
			return nil, fmt.Errorf("failed to register pipeline agent %q: %w", agent.ID, err)
		}
	}
	registry.SetTriageOnlyInboxes(cfg.TriageOnlyInboxes)
	// M-PKG-QUALITY-LADDER M6: the CLI readouts (`messages inboxes`, health,
	// the send guard) see the same derived package agents the daemon serves.
	registry.MaterializePackageAgentsFromRegistry(nil)
	return registry, nil
}

// LoadAgentRegistryFrom loads agents from a specific config path.
func LoadAgentRegistryFrom(configPath string) (*AgentRegistry, error) {
	reg, _, err := LoadAgentRegistryFromDeclared(configPath)
	return reg, err
}

// LoadAgentRegistryFromDeclared loads agents from a config path AND reports
// whether that file actually declared any.
//
// A file with no `coordinator:` section does not produce an empty registry — it
// produces AILANG's built-in default, a registry of one agent (`coordinator`).
// Handing that back unlabelled turns a config-shape mistake into a confident
// wrong answer about which agents exist: every other inbox reads as unserved.
//
// Measured 2026-09-17 (daneel v0.2.5 → v0.2.11): Daneel's send path sets
// $AILANG_CONFIG to a pubsub-only file, because a send publishes its
// notification only when the sender's config has a pubsub section. Every
// command that resolves a registry from that variable — inboxes, health, prs,
// approvals, the send guard, pipeline, lint, agents — then answered from the
// default fleet while labelling it as the plane's registry. Daneel's own guard
// ("refuse unless the registry is the shared plane's") caught it and deferred
// two of Mark's design requests for seven hours; nothing else would have.
func LoadAgentRegistryFromDeclared(configPath string) (*AgentRegistry, bool, error) {
	cfg, declared, err := loadCoordinatorConfigDeclared(configPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load coordinator config from %q: %w", configPath, err)
	}
	reg, err := buildRegistryFromConfig(cfg)
	return reg, declared, err
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

// UnknownConfigKeys returns config keys that no struct field reads.
//
// YAML silently drops keys it cannot map, so a plausible-looking setting can sit
// in the config doing nothing. Measured 2026-09-11: `push_branch: dev` was added
// to an agent entry to make it push directly. AgentConfig has no PushBranch
// field — the real control is skip_approval + merge_branch — so the key was
// inert, and the entry read as if it were configured.
//
// Returns the offending key paths rather than an error, because the caller
// wants to report all of them, not stop at the first. Only AGENT keys are
// reported: the decode target models `coordinator:` alone, so every sibling
// top-level block (github:, pubsub:) would report as unknown and be a false
// positive — and a checker that cries wolf gets ignored.
func UnknownConfigKeys(data []byte) ([]string, error) {
	f, err := config.Parse(data)
	if err != nil {
		return nil, err
	}
	var cfg struct {
		Coordinator CoordinatorConfig `yaml:"coordinator"`
	}
	return f.UnknownKeys(&cfg, "coordinator.AgentConfig")
}
