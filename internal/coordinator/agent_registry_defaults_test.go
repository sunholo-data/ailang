package coordinator

import (
	"os"
	"path/filepath"
	"testing"
)

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

	designAgent := registry.GetAgentByID("design-agent")
	if designAgent == nil {
		t.Fatal("expected to find design-agent")
	}
	if designAgent.Model != "opus" {
		t.Errorf("expected model 'opus', got %q", designAgent.Model)
	}

	sprintAgent := registry.GetAgentByID("sprint-agent")
	if sprintAgent == nil {
		t.Fatal("expected to find sprint-agent")
	}
	if sprintAgent.Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %q", sprintAgent.Model)
	}

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
