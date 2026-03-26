package coordinator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentRegistry_Basic(t *testing.T) {
	registry := NewAgentRegistry()

	// Register an agent
	agent := &AgentConfig{
		ID:           "test-agent",
		Label:        "Test Agent",
		Inbox:        "test-inbox",
		Workspace:    "/tmp/test",
		Capabilities: []string{"code", "test"},
		Provider:     "claude",
	}

	if err := registry.Register(agent); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// Verify count
	if count := registry.Count(); count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Lookup by ID
	found := registry.GetAgentByID("test-agent")
	if found == nil {
		t.Fatal("expected to find agent by ID")
	}
	if found.Label != "Test Agent" {
		t.Errorf("expected label 'Test Agent', got %q", found.Label)
	}

	// Lookup by inbox
	found = registry.GetAgentForInbox("test-inbox")
	if found == nil {
		t.Fatal("expected to find agent by inbox")
	}
	if found.ID != "test-agent" {
		t.Errorf("expected ID 'test-agent', got %q", found.ID)
	}

	// Not found
	if registry.GetAgentByID("nonexistent") != nil {
		t.Error("expected nil for nonexistent ID")
	}
	if registry.GetAgentForInbox("nonexistent") != nil {
		t.Error("expected nil for nonexistent inbox")
	}
}

func TestAgentRegistry_DuplicateID(t *testing.T) {
	registry := NewAgentRegistry()

	agent1 := &AgentConfig{
		ID:    "agent-1",
		Inbox: "inbox-1",
	}
	agent2 := &AgentConfig{
		ID:    "agent-1", // Same ID
		Inbox: "inbox-2",
	}

	if err := registry.Register(agent1); err != nil {
		t.Fatalf("failed to register first agent: %v", err)
	}

	err := registry.Register(agent2)
	if err == nil {
		t.Error("expected error for duplicate ID")
	}
}

func TestAgentRegistry_DuplicateInbox(t *testing.T) {
	registry := NewAgentRegistry()

	agent1 := &AgentConfig{
		ID:    "agent-1",
		Inbox: "inbox-1",
	}
	agent2 := &AgentConfig{
		ID:    "agent-2",
		Inbox: "inbox-1", // Same inbox
	}

	if err := registry.Register(agent1); err != nil {
		t.Fatalf("failed to register first agent: %v", err)
	}

	err := registry.Register(agent2)
	if err == nil {
		t.Error("expected error for duplicate inbox")
	}
}

func TestAgentRegistry_ListAgents(t *testing.T) {
	registry := NewAgentRegistry()

	_ = registry.Register(&AgentConfig{ID: "agent-1", Inbox: "inbox-1"})
	_ = registry.Register(&AgentConfig{ID: "agent-2", Inbox: "inbox-2"})
	_ = registry.Register(&AgentConfig{ID: "agent-3", Inbox: "inbox-3"})

	agents := registry.ListAgents()
	if len(agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(agents))
	}

	inboxes := registry.ListInboxes()
	if len(inboxes) != 3 {
		t.Errorf("expected 3 inboxes, got %d", len(inboxes))
	}
}

func TestAgentRegistry_Unregister(t *testing.T) {
	registry := NewAgentRegistry()

	_ = registry.Register(&AgentConfig{ID: "agent-1", Inbox: "inbox-1"})

	if !registry.HasAgent("agent-1") {
		t.Error("expected HasAgent to return true")
	}

	if err := registry.Unregister("agent-1"); err != nil {
		t.Fatalf("failed to unregister: %v", err)
	}

	if registry.HasAgent("agent-1") {
		t.Error("expected HasAgent to return false after unregister")
	}
	if registry.HasInbox("inbox-1") {
		t.Error("expected HasInbox to return false after unregister")
	}
}

func TestAgentRegistry_Validate(t *testing.T) {
	registry := NewAgentRegistry()

	// Agent with trigger to unknown agent
	_ = registry.Register(&AgentConfig{
		ID:                "agent-1",
		Inbox:             "inbox-1",
		Workspace:         "/tmp/test",
		Capabilities:      []string{"code"},
		TriggerOnComplete: []string{"unknown-agent"},
	})

	issues := registry.Validate()
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d: %v", len(issues), issues)
	}

	// Add the missing agent
	_ = registry.Register(&AgentConfig{
		ID:           "unknown-agent",
		Inbox:        "inbox-2",
		Workspace:    "/tmp/test2",
		Capabilities: []string{"test"},
	})

	issues = registry.Validate()
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d: %v", len(issues), issues)
	}
}

func TestAgentRegistry_Clear(t *testing.T) {
	registry := NewAgentRegistry()

	_ = registry.Register(&AgentConfig{ID: "agent-1", Inbox: "inbox-1"})
	_ = registry.Register(&AgentConfig{ID: "agent-2", Inbox: "inbox-2"})

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("expected count 0 after clear, got %d", registry.Count())
	}
}

func TestLoadAgentRegistryFrom(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
coordinator:
  default_provider: claude
  agents:
    - id: test-coordinator
      label: Test Coordinator
      inbox: coordinator
      workspace: /tmp/test
      capabilities: [code, test]
      provider: claude
    - id: test-planner
      label: Test Planner
      inbox: planner
      workspace: /tmp/test
      capabilities: [research]
      trigger_on_complete: [test-coordinator]
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	registry, err := LoadAgentRegistryFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	if registry.Count() != 2 {
		t.Errorf("expected 2 agents, got %d", registry.Count())
	}

	// Verify agent details
	coordinator := registry.GetAgentByID("test-coordinator")
	if coordinator == nil {
		t.Fatal("expected to find test-coordinator")
	}
	if coordinator.Provider != "claude" {
		t.Errorf("expected provider 'claude', got %q", coordinator.Provider)
	}

	planner := registry.GetAgentForInbox("planner")
	if planner == nil {
		t.Fatal("expected to find agent for planner inbox")
	}
	if len(planner.TriggerOnComplete) != 1 || planner.TriggerOnComplete[0] != "test-coordinator" {
		t.Errorf("expected trigger_on_complete [test-coordinator], got %v", planner.TriggerOnComplete)
	}

	// Validate should pass (trigger reference exists)
	issues := registry.Validate()
	if len(issues) != 0 {
		t.Errorf("expected no validation issues, got: %v", issues)
	}
}

func TestLoadAgentRegistryFrom_MissingFile(t *testing.T) {
	registry, err := LoadAgentRegistryFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}

	// Should return default configuration
	if registry.Count() < 1 {
		t.Error("expected at least 1 default agent")
	}

	coordinator := registry.GetAgentByID("coordinator")
	if coordinator == nil {
		t.Error("expected default coordinator agent")
	}
}

func TestLoadAgentRegistryFrom_NoCoordinatorSection(t *testing.T) {
	// Create a config file without coordinator section
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
github:
  expected_user: test-user
  default_repo: test/repo
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	registry, err := LoadAgentRegistryFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	// Should return default configuration
	if registry.Count() < 1 {
		t.Error("expected at least 1 default agent")
	}
}

func TestDefaultCoordinatorConfig(t *testing.T) {
	cfg := DefaultCoordinatorConfig()

	if cfg.DefaultProvider != "claude" {
		t.Errorf("expected default provider 'claude', got %q", cfg.DefaultProvider)
	}

	if len(cfg.Agents) < 1 {
		t.Fatal("expected at least 1 default agent")
	}

	coordinator := cfg.Agents[0]
	if coordinator.ID != "coordinator" {
		t.Errorf("expected first agent ID 'coordinator', got %q", coordinator.ID)
	}
}

func TestGitHubSyncConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
coordinator:
  agents:
    - id: coordinator
      inbox: coordinator
      workspace: /tmp
      capabilities: [code]
  github_sync:
    enabled: true
    interval_secs: 600
    watch_labels: [bug, feature]
    target_inbox: coordinator
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadCoordinatorConfigFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.GitHubSync == nil {
		t.Fatal("expected github_sync config")
	}
	if !cfg.GitHubSync.Enabled {
		t.Error("expected github_sync.enabled to be true")
	}
	if cfg.GitHubSync.IntervalSecs != 600 {
		t.Errorf("expected interval_secs 600, got %d", cfg.GitHubSync.IntervalSecs)
	}
	if len(cfg.GitHubSync.WatchLabels) != 2 {
		t.Errorf("expected 2 watch_labels, got %d", len(cfg.GitHubSync.WatchLabels))
	}
}

// Tests for InvokeConfig and ApprovalConfig (M-COORD-GENERIC-WORKFLOWS M1)

func TestInvokeConfig_SkillType(t *testing.T) {
	invoke := &InvokeConfig{
		Type: "skill",
		Name: "design-doc-creator",
	}

	if invoke.Type != "skill" {
		t.Errorf("expected Type 'skill', got %q", invoke.Type)
	}
	if invoke.Name != "design-doc-creator" {
		t.Errorf("expected Name 'design-doc-creator', got %q", invoke.Name)
	}
	if invoke.Template != "" {
		t.Errorf("expected empty Template for skill type, got %q", invoke.Template)
	}
}

func TestInvokeConfig_AgentType(t *testing.T) {
	invoke := &InvokeConfig{
		Type: "agent",
		Name: "sprint-planner",
	}

	if invoke.Type != "agent" {
		t.Errorf("expected Type 'agent', got %q", invoke.Type)
	}
	if invoke.Name != "sprint-planner" {
		t.Errorf("expected Name 'sprint-planner', got %q", invoke.Name)
	}
}

func TestInvokeConfig_PromptType(t *testing.T) {
	invoke := &InvokeConfig{
		Type:     "prompt",
		Template: "Please analyze the task: {{.TaskTitle}} and create a design document at {{.DesignDocPath}}",
	}

	if invoke.Type != "prompt" {
		t.Errorf("expected Type 'prompt', got %q", invoke.Type)
	}
	if invoke.Template == "" {
		t.Error("expected non-empty Template for prompt type")
	}
}

func TestApprovalConfig_Labels(t *testing.T) {
	approval := &ApprovalConfig{
		NeedsLabel:            "needs-design-approval",
		ApprovedLabel:         "design-approved",
		GithubCommentTemplate: "## Design Document Ready\n\nPlease review: {{.DesignDocPath}}",
	}

	if approval.NeedsLabel != "needs-design-approval" {
		t.Errorf("expected NeedsLabel 'needs-design-approval', got %q", approval.NeedsLabel)
	}
	if approval.ApprovedLabel != "design-approved" {
		t.Errorf("expected ApprovedLabel 'design-approved', got %q", approval.ApprovedLabel)
	}
	if approval.GithubCommentTemplate == "" {
		t.Error("expected non-empty GithubCommentTemplate")
	}
}

func TestAgentConfig_WithGenericWorkflow(t *testing.T) {
	agent := &AgentConfig{
		ID:           "design-agent",
		Label:        "Design Agent",
		Inbox:        "design",
		Workspace:    "/tmp/design",
		Capabilities: []string{"code", "research"},
		Provider:     "claude",
		Invoke: &InvokeConfig{
			Type: "skill",
			Name: "design-doc-creator",
		},
		OutputMarkers: []string{"DESIGN_DOC_PATH:", "SPRINT_PLAN_PATH:"},
		Approval: &ApprovalConfig{
			NeedsLabel:    "needs-design-approval",
			ApprovedLabel: "design-approved",
		},
	}

	// Verify Invoke config
	if agent.Invoke == nil {
		t.Fatal("expected non-nil Invoke config")
	}
	if agent.Invoke.Type != "skill" {
		t.Errorf("expected Invoke.Type 'skill', got %q", agent.Invoke.Type)
	}
	if agent.Invoke.Name != "design-doc-creator" {
		t.Errorf("expected Invoke.Name 'design-doc-creator', got %q", agent.Invoke.Name)
	}

	// Verify OutputMarkers
	if len(agent.OutputMarkers) != 2 {
		t.Errorf("expected 2 OutputMarkers, got %d", len(agent.OutputMarkers))
	}
	if agent.OutputMarkers[0] != "DESIGN_DOC_PATH:" {
		t.Errorf("expected first marker 'DESIGN_DOC_PATH:', got %q", agent.OutputMarkers[0])
	}

	// Verify Approval config
	if agent.Approval == nil {
		t.Fatal("expected non-nil Approval config")
	}
	if agent.Approval.NeedsLabel != "needs-design-approval" {
		t.Errorf("expected Approval.NeedsLabel 'needs-design-approval', got %q", agent.Approval.NeedsLabel)
	}
}

func TestLoadAgentRegistryFrom_WithGenericWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
coordinator:
  default_provider: claude
  agents:
    - id: design-agent
      label: Design Agent
      inbox: design
      workspace: /tmp/design
      capabilities: [code, research]
      provider: claude
      invoke:
        type: skill
        name: design-doc-creator
      output_markers:
        - "DESIGN_DOC_PATH:"
        - "STATUS:"
      approval:
        needs_label: needs-design-approval
        approved_label: design-approved
        github_comment_template: "## Design Ready\n\nPath: {{.DesignDocPath}}"
    - id: sprint-agent
      label: Sprint Agent
      inbox: sprint
      workspace: /tmp/sprint
      capabilities: [code]
      invoke:
        type: agent
        name: sprint-planner
      output_markers:
        - "SPRINT_PLAN_PATH:"
      approval:
        needs_label: needs-sprint-approval
        approved_label: sprint-approved
    - id: custom-agent
      label: Custom Prompt Agent
      inbox: custom
      workspace: /tmp/custom
      capabilities: [research]
      invoke:
        type: prompt
        template: "Analyze task: {{.TaskTitle}}"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	registry, err := LoadAgentRegistryFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	if registry.Count() != 3 {
		t.Errorf("expected 3 agents, got %d", registry.Count())
	}

	// Verify design-agent with skill invoke
	designAgent := registry.GetAgentByID("design-agent")
	if designAgent == nil {
		t.Fatal("expected to find design-agent")
	}
	if designAgent.Invoke == nil {
		t.Fatal("expected design-agent to have Invoke config")
	}
	if designAgent.Invoke.Type != "skill" {
		t.Errorf("expected Invoke.Type 'skill', got %q", designAgent.Invoke.Type)
	}
	if designAgent.Invoke.Name != "design-doc-creator" {
		t.Errorf("expected Invoke.Name 'design-doc-creator', got %q", designAgent.Invoke.Name)
	}
	if len(designAgent.OutputMarkers) != 2 {
		t.Errorf("expected 2 OutputMarkers, got %d", len(designAgent.OutputMarkers))
	}
	if designAgent.Approval == nil {
		t.Fatal("expected design-agent to have Approval config")
	}
	if designAgent.Approval.NeedsLabel != "needs-design-approval" {
		t.Errorf("expected NeedsLabel 'needs-design-approval', got %q", designAgent.Approval.NeedsLabel)
	}
	if designAgent.Approval.GithubCommentTemplate == "" {
		t.Error("expected non-empty GithubCommentTemplate")
	}

	// Verify sprint-agent with agent invoke
	sprintAgent := registry.GetAgentByID("sprint-agent")
	if sprintAgent == nil {
		t.Fatal("expected to find sprint-agent")
	}
	if sprintAgent.Invoke == nil {
		t.Fatal("expected sprint-agent to have Invoke config")
	}
	if sprintAgent.Invoke.Type != "agent" {
		t.Errorf("expected Invoke.Type 'agent', got %q", sprintAgent.Invoke.Type)
	}
	if len(sprintAgent.OutputMarkers) != 1 {
		t.Errorf("expected 1 OutputMarker, got %d", len(sprintAgent.OutputMarkers))
	}

	// Verify custom-agent with prompt invoke
	customAgent := registry.GetAgentByID("custom-agent")
	if customAgent == nil {
		t.Fatal("expected to find custom-agent")
	}
	if customAgent.Invoke == nil {
		t.Fatal("expected custom-agent to have Invoke config")
	}
	if customAgent.Invoke.Type != "prompt" {
		t.Errorf("expected Invoke.Type 'prompt', got %q", customAgent.Invoke.Type)
	}
	if customAgent.Invoke.Template == "" {
		t.Error("expected non-empty Template for prompt type")
	}
	// Custom agent has no approval config (optional)
	if customAgent.Approval != nil {
		t.Error("expected custom-agent to have nil Approval config")
	}
}

func TestAgentConfig_NilOptionalFields(t *testing.T) {
	// Agents without generic workflow config should work (backwards compatibility)
	agent := &AgentConfig{
		ID:           "legacy-agent",
		Label:        "Legacy Agent",
		Inbox:        "legacy",
		Workspace:    "/tmp/legacy",
		Capabilities: []string{"code"},
		Provider:     "claude",
		// Invoke, OutputMarkers, and Approval are all nil/empty
	}

	if agent.Invoke != nil {
		t.Error("expected nil Invoke for legacy agent")
	}
	if len(agent.OutputMarkers) != 0 {
		t.Errorf("expected empty OutputMarkers, got %d", len(agent.OutputMarkers))
	}
	if agent.Approval != nil {
		t.Error("expected nil Approval for legacy agent")
	}
}

// M-PKG-AUTONOMOUS-UPDATES: Tests for subdirectory field and pkg: inbox format.

func TestAgentConfig_SubdirectoryField(t *testing.T) {
	agent := &AgentConfig{
		ID:           "pkg-sunholo-auth",
		Label:        "Package: sunholo/auth",
		Inbox:        "pkg:sunholo/auth",
		Workspace:    "/tmp/ailang-packages",
		Subdirectory: "packages/auth",
		Capabilities: []string{"code", "test"},
		Provider:     "claude",
	}

	if agent.Subdirectory != "packages/auth" {
		t.Errorf("expected Subdirectory 'packages/auth', got %q", agent.Subdirectory)
	}

	// Verify subdirectory produces correct workspace path
	expectedPath := filepath.Join(agent.Workspace, agent.Subdirectory)
	wantPath := filepath.Join("/tmp", "ailang-packages", "packages", "auth")
	if expectedPath != wantPath {
		t.Errorf("expected path %q, got %q", wantPath, expectedPath)
	}
}

func TestAgentRegistry_PackageInboxFormat(t *testing.T) {
	// Verify that pkg:vendor/name inbox format works for registration and lookup
	registry := NewAgentRegistry()

	agent := &AgentConfig{
		ID:           "pkg-sunholo-auth",
		Label:        "Package: sunholo/auth",
		Inbox:        "pkg:sunholo/auth",
		Workspace:    "/tmp/test",
		Subdirectory: "packages/auth",
	}

	if err := registry.Register(agent); err != nil {
		t.Fatalf("failed to register agent with pkg: inbox: %v", err)
	}

	// Lookup by inbox with colon
	found := registry.GetAgentForInbox("pkg:sunholo/auth")
	if found == nil {
		t.Fatal("expected to find agent by pkg:sunholo/auth inbox")
	}
	if found.ID != "pkg-sunholo-auth" {
		t.Errorf("expected ID 'pkg-sunholo-auth', got %q", found.ID)
	}
	if found.Subdirectory != "packages/auth" {
		t.Errorf("expected Subdirectory 'packages/auth', got %q", found.Subdirectory)
	}

	// Register multiple package agents
	if err := registry.Register(&AgentConfig{
		ID:           "pkg-sunholo-gcp-auth",
		Inbox:        "pkg:sunholo/gcp-auth",
		Workspace:    "/tmp/test",
		Subdirectory: "packages/gcp-auth",
	}); err != nil {
		t.Fatalf("failed to register second pkg agent: %v", err)
	}

	// Verify both are findable
	if registry.GetAgentForInbox("pkg:sunholo/gcp-auth") == nil {
		t.Error("expected to find pkg:sunholo/gcp-auth agent")
	}
	if registry.Count() != 2 {
		t.Errorf("expected 2 agents, got %d", registry.Count())
	}
}

func TestLoadAgentRegistryFrom_WithSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
coordinator:
  default_provider: claude
  agents:
    - id: pkg-sunholo-auth
      label: "Package: sunholo/auth"
      inbox: "pkg:sunholo/auth"
      workspace: /tmp/ailang-packages
      subdirectory: packages/auth
      capabilities: [code, test]
      provider: claude
      model: sonnet
      timeout: "30m"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	registry, err := LoadAgentRegistryFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	agent := registry.GetAgentByID("pkg-sunholo-auth")
	if agent == nil {
		t.Fatal("expected to find pkg-sunholo-auth agent")
	}
	if agent.Inbox != "pkg:sunholo/auth" {
		t.Errorf("expected inbox 'pkg:sunholo/auth', got %q", agent.Inbox)
	}
	if agent.Subdirectory != "packages/auth" {
		t.Errorf("expected subdirectory 'packages/auth', got %q", agent.Subdirectory)
	}
	if agent.Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %q", agent.Model)
	}

	// Verify inbox lookup with colon works after YAML load
	found := registry.GetAgentForInbox("pkg:sunholo/auth")
	if found == nil {
		t.Fatal("expected to find agent by inbox after YAML load")
	}
}
