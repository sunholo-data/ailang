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
		t.Error("expected to find agent by ID")
	}
	if found.Label != "Test Agent" {
		t.Errorf("expected label 'Test Agent', got %q", found.Label)
	}

	// Lookup by inbox
	found = registry.GetAgentForInbox("test-inbox")
	if found == nil {
		t.Error("expected to find agent by inbox")
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
