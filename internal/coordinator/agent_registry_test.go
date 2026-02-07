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

// =============================================================================
// Default Config Tests (M-COORD-GENERIC-WORKFLOWS M5)
// =============================================================================

func TestDefaultInvokeConfig_KnownAgents(t *testing.T) {
	tests := []struct {
		agentID      string
		expectedType string
		expectedName string
	}{
		{"design-doc-creator", "skill", "design-doc-creator"},
		{"sprint-planner", "skill", "sprint-planner"},
		{"sprint-executor", "skill", "sprint-executor"},
	}

	for _, tc := range tests {
		t.Run(tc.agentID, func(t *testing.T) {
			config := DefaultInvokeConfig(tc.agentID)
			if config == nil {
				t.Fatalf("expected non-nil config for %s", tc.agentID)
			}
			if config.Type != tc.expectedType {
				t.Errorf("expected type %q, got %q", tc.expectedType, config.Type)
			}
			if config.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, config.Name)
			}
		})
	}
}

func TestDefaultInvokeConfig_UnknownAgent(t *testing.T) {
	config := DefaultInvokeConfig("unknown-agent")
	if config != nil {
		t.Error("expected nil config for unknown agent")
	}
}

func TestDefaultOutputMarkers_KnownAgents(t *testing.T) {
	tests := []struct {
		agentID  string
		expected []string
	}{
		{"design-doc-creator", []string{"DESIGN_DOC_PATH:"}},
		{"sprint-planner", []string{"SPRINT_PLAN_PATH:", "SPRINT_JSON_PATH:"}},
		{"sprint-executor", []string{"IMPLEMENTATION_COMPLETE:", "BRANCH_NAME:", "FILES_CREATED:", "FILES_MODIFIED:"}},
	}

	for _, tc := range tests {
		t.Run(tc.agentID, func(t *testing.T) {
			markers := DefaultOutputMarkers(tc.agentID)
			if len(markers) != len(tc.expected) {
				t.Fatalf("expected %d markers, got %d", len(tc.expected), len(markers))
			}
			for i, m := range markers {
				if m != tc.expected[i] {
					t.Errorf("marker %d: expected %q, got %q", i, tc.expected[i], m)
				}
			}
		})
	}
}

func TestDefaultOutputMarkers_UnknownAgent(t *testing.T) {
	markers := DefaultOutputMarkers("unknown-agent")
	if markers != nil {
		t.Error("expected nil markers for unknown agent")
	}
}

func TestDefaultApprovalConfig_KnownAgents(t *testing.T) {
	tests := []struct {
		agentID       string
		needsLabel    string
		approvedLabel string
	}{
		{"design-doc-creator", "needs-design-approval", "design-approved"},
		{"sprint-planner", "needs-sprint-approval", "sprint-approved"},
		{"sprint-executor", "needs-implementation-approval", "implementation-approved"},
	}

	for _, tc := range tests {
		t.Run(tc.agentID, func(t *testing.T) {
			config := DefaultApprovalConfig(tc.agentID)
			if config == nil {
				t.Fatalf("expected non-nil config for %s", tc.agentID)
			}
			if config.NeedsLabel != tc.needsLabel {
				t.Errorf("expected needs_label %q, got %q", tc.needsLabel, config.NeedsLabel)
			}
			if config.ApprovedLabel != tc.approvedLabel {
				t.Errorf("expected approved_label %q, got %q", tc.approvedLabel, config.ApprovedLabel)
			}
		})
	}
}

func TestDefaultApprovalConfig_UnknownAgent(t *testing.T) {
	config := DefaultApprovalConfig("unknown-agent")
	if config != nil {
		t.Error("expected nil config for unknown agent")
	}
}

func TestAgentConfig_GetEffectiveInvokeConfig_ExplicitConfig(t *testing.T) {
	agent := &AgentConfig{
		ID: "design-doc-creator",
		Invoke: &InvokeConfig{
			Type: "prompt",
			Name: "custom-prompt",
		},
	}

	config := agent.GetEffectiveInvokeConfig()
	if config.Type != "prompt" {
		t.Errorf("expected explicit config type 'prompt', got %q", config.Type)
	}
	if config.Name != "custom-prompt" {
		t.Errorf("expected explicit config name 'custom-prompt', got %q", config.Name)
	}
}

func TestAgentConfig_GetEffectiveInvokeConfig_FallsBackToDefault(t *testing.T) {
	agent := &AgentConfig{
		ID:     "design-doc-creator",
		Invoke: nil, // No explicit config
	}

	config := agent.GetEffectiveInvokeConfig()
	if config == nil {
		t.Fatal("expected default config, got nil")
	}
	if config.Type != "skill" {
		t.Errorf("expected default type 'skill', got %q", config.Type)
	}
	if config.Name != "design-doc-creator" {
		t.Errorf("expected default name 'design-doc-creator', got %q", config.Name)
	}
}

func TestAgentConfig_GetEffectiveOutputMarkers_ExplicitMarkers(t *testing.T) {
	agent := &AgentConfig{
		ID:            "design-doc-creator",
		OutputMarkers: []string{"CUSTOM_MARKER:"},
	}

	markers := agent.GetEffectiveOutputMarkers()
	if len(markers) != 1 {
		t.Fatalf("expected 1 marker, got %d", len(markers))
	}
	if markers[0] != "CUSTOM_MARKER:" {
		t.Errorf("expected 'CUSTOM_MARKER:', got %q", markers[0])
	}
}

func TestAgentConfig_GetEffectiveOutputMarkers_FallsBackToDefault(t *testing.T) {
	agent := &AgentConfig{
		ID:            "design-doc-creator",
		OutputMarkers: nil, // No explicit markers
	}

	markers := agent.GetEffectiveOutputMarkers()
	if len(markers) != 1 {
		t.Fatalf("expected 1 default marker, got %d", len(markers))
	}
	if markers[0] != "DESIGN_DOC_PATH:" {
		t.Errorf("expected default 'DESIGN_DOC_PATH:', got %q", markers[0])
	}
}

func TestAgentConfig_GetEffectiveApprovalConfig_ExplicitConfig(t *testing.T) {
	agent := &AgentConfig{
		ID: "design-doc-creator",
		Approval: &ApprovalConfig{
			NeedsLabel:    "custom-needs",
			ApprovedLabel: "custom-approved",
		},
	}

	config := agent.GetEffectiveApprovalConfig()
	if config.NeedsLabel != "custom-needs" {
		t.Errorf("expected 'custom-needs', got %q", config.NeedsLabel)
	}
	if config.ApprovedLabel != "custom-approved" {
		t.Errorf("expected 'custom-approved', got %q", config.ApprovedLabel)
	}
}

func TestAgentConfig_GetEffectiveApprovalConfig_FallsBackToDefault(t *testing.T) {
	agent := &AgentConfig{
		ID:       "design-doc-creator",
		Approval: nil, // No explicit config
	}

	config := agent.GetEffectiveApprovalConfig()
	if config == nil {
		t.Fatal("expected default config, got nil")
	}
	if config.NeedsLabel != "needs-design-approval" {
		t.Errorf("expected 'needs-design-approval', got %q", config.NeedsLabel)
	}
	if config.ApprovedLabel != "design-approved" {
		t.Errorf("expected 'design-approved', got %q", config.ApprovedLabel)
	}
}

func TestAgentConfig_GetEffectiveApprovalConfig_UnknownAgent(t *testing.T) {
	agent := &AgentConfig{
		ID:       "unknown-agent",
		Approval: nil,
	}

	config := agent.GetEffectiveApprovalConfig()
	if config != nil {
		t.Error("expected nil config for unknown agent without explicit config")
	}
}

// =============================================================================
// Per-Agent Model Config Tests (v0.8.0+)
// =============================================================================

func TestAgentConfig_ModelField(t *testing.T) {
	agent := &AgentConfig{
		ID:    "test-agent",
		Inbox: "test",
		Model: "opus",
	}

	if agent.Model != "opus" {
		t.Errorf("expected model 'opus', got %q", agent.Model)
	}
}

func TestAgentConfig_ModelFieldEmpty(t *testing.T) {
	agent := &AgentConfig{
		ID:    "test-agent",
		Inbox: "test",
		// Model not set - should be empty string
	}

	if agent.Model != "" {
		t.Errorf("expected empty model, got %q", agent.Model)
	}
}

func TestLoadAgentRegistryFrom_WithModel(t *testing.T) {
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
      capabilities: [code]
      model: opus
    - id: sprint-agent
      label: Sprint Agent
      inbox: sprint
      workspace: /tmp/sprint
      capabilities: [code]
      model: sonnet
    - id: cheap-agent
      label: Cheap Agent
      inbox: cheap
      workspace: /tmp/cheap
      capabilities: [docs]
      # model not set - should use executor default
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	registry, err := LoadAgentRegistryFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	// Verify design-agent has opus model
	designAgent := registry.GetAgentByID("design-agent")
	if designAgent == nil {
		t.Fatal("expected to find design-agent")
	}
	if designAgent.Model != "opus" {
		t.Errorf("expected model 'opus', got %q", designAgent.Model)
	}

	// Verify sprint-agent has sonnet model
	sprintAgent := registry.GetAgentByID("sprint-agent")
	if sprintAgent == nil {
		t.Fatal("expected to find sprint-agent")
	}
	if sprintAgent.Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %q", sprintAgent.Model)
	}

	// Verify cheap-agent has no model (empty = executor default)
	cheapAgent := registry.GetAgentByID("cheap-agent")
	if cheapAgent == nil {
		t.Fatal("expected to find cheap-agent")
	}
	if cheapAgent.Model != "" {
		t.Errorf("expected empty model for cheap-agent, got %q", cheapAgent.Model)
	}
}

func TestAgentConfig_GetEffectiveTimeout(t *testing.T) {
	tests := []struct {
		name    string
		agent   *AgentConfig
		wantMin float64 // minutes
	}{
		{"nil agent returns 60m", nil, 60},
		{"empty timeout returns 60m", &AgentConfig{Timeout: ""}, 60},
		{"15m timeout", &AgentConfig{Timeout: "15m"}, 15},
		{"30m timeout", &AgentConfig{Timeout: "30m"}, 30},
		{"1h timeout", &AgentConfig{Timeout: "1h"}, 60},
		{"invalid timeout returns 60m", &AgentConfig{Timeout: "not-a-duration"}, 60},
		{"zero timeout returns 60m", &AgentConfig{Timeout: "0s"}, 60},
		{"negative timeout returns 60m", &AgentConfig{Timeout: "-5m"}, 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.agent.GetEffectiveTimeout()
			gotMin := got.Minutes()
			if gotMin != tt.wantMin {
				t.Errorf("GetEffectiveTimeout() = %v (%.0fm), want %.0fm", got, gotMin, tt.wantMin)
			}
		})
	}
}

func TestAgentConfig_GetEffectiveIdleTimeout(t *testing.T) {
	tests := []struct {
		name    string
		agent   *AgentConfig
		wantMin float64 // minutes
	}{
		{"nil agent returns 3m", nil, 3},
		{"empty idle_timeout returns 3m", &AgentConfig{IdleTimeout: ""}, 3},
		{"2m idle_timeout", &AgentConfig{IdleTimeout: "2m"}, 2},
		{"5m idle_timeout", &AgentConfig{IdleTimeout: "5m"}, 5},
		{"30s idle_timeout", &AgentConfig{IdleTimeout: "30s"}, 0.5},
		{"invalid idle_timeout returns 3m", &AgentConfig{IdleTimeout: "not-a-duration"}, 3},
		{"zero idle_timeout returns 3m", &AgentConfig{IdleTimeout: "0s"}, 3},
		{"negative idle_timeout returns 3m", &AgentConfig{IdleTimeout: "-1m"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.agent.GetEffectiveIdleTimeout()
			gotMin := got.Minutes()
			if gotMin != tt.wantMin {
				t.Errorf("GetEffectiveIdleTimeout() = %v (%.1fm), want %.1fm", got, gotMin, tt.wantMin)
			}
		})
	}
}
