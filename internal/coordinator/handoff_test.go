package coordinator

import (
	"testing"
	"time"
)

func TestHandoffApprovalRequest(t *testing.T) {
	// Test that handoff-specific fields are properly set in ApprovalRequest
	request := &ApprovalRequest{
		ID:            "handoff-test-1",
		TaskID:        "task-123",
		ThreadID:      "thread-456",
		Type:          ApprovalTypeHandoff,
		Title:         "Handoff: Agent A → Agent B",
		Description:   "Test handoff message",
		SourceAgentID: "agent-a",
		TargetAgentID: "agent-b",
		SessionID:     "session-xyz",
		Timeout:       24 * time.Hour,
		AutoReject:    true,
	}

	if request.Type != ApprovalTypeHandoff {
		t.Errorf("expected type %q, got %q", ApprovalTypeHandoff, request.Type)
	}
	if request.SourceAgentID != "agent-a" {
		t.Errorf("expected source agent 'agent-a', got %q", request.SourceAgentID)
	}
	if request.TargetAgentID != "agent-b" {
		t.Errorf("expected target agent 'agent-b', got %q", request.TargetAgentID)
	}
	if request.SessionID != "session-xyz" {
		t.Errorf("expected session ID 'session-xyz', got %q", request.SessionID)
	}
}

func TestAgentHandoffWithAutoApprove(t *testing.T) {
	registry := NewAgentRegistry()

	// Create source agent with auto_approve_handoffs=true
	sourceAgent := &AgentConfig{
		ID:                  "source-agent",
		Label:               "Source Agent",
		Inbox:               "source-inbox",
		Workspace:           "/tmp/source",
		Capabilities:        []string{"code", "test"},
		TriggerOnComplete:   []string{"target-agent"},
		AutoApproveHandoffs: true,
	}

	// Create target agent
	targetAgent := &AgentConfig{
		ID:           "target-agent",
		Label:        "Target Agent",
		Inbox:        "target-inbox",
		Workspace:    "/tmp/target",
		Capabilities: []string{"code"},
	}

	if err := registry.Register(sourceAgent); err != nil {
		t.Fatalf("failed to register source agent: %v", err)
	}
	if err := registry.Register(targetAgent); err != nil {
		t.Fatalf("failed to register target agent: %v", err)
	}

	// Verify configuration
	retrieved := registry.GetAgentByID("source-agent")
	if retrieved == nil {
		t.Fatal("expected to find source-agent")
	}
	if !retrieved.AutoApproveHandoffs {
		t.Error("expected AutoApproveHandoffs to be true")
	}
	if len(retrieved.TriggerOnComplete) != 1 || retrieved.TriggerOnComplete[0] != "target-agent" {
		t.Errorf("expected TriggerOnComplete to be [target-agent], got %v", retrieved.TriggerOnComplete)
	}

	// Validate should pass
	issues := registry.Validate()
	if len(issues) != 0 {
		t.Errorf("expected no validation issues, got: %v", issues)
	}
}

func TestAgentHandoffWithApprovalRequired(t *testing.T) {
	registry := NewAgentRegistry()

	// Create source agent with auto_approve_handoffs=false (default)
	sourceAgent := &AgentConfig{
		ID:                  "planner-agent",
		Label:               "Sprint Planner",
		Inbox:               "planner",
		Workspace:           "/tmp/planner",
		Capabilities:        []string{"research", "planning"},
		TriggerOnComplete:   []string{"executor-agent"},
		AutoApproveHandoffs: false, // Requires human approval
	}

	// Create target agent
	targetAgent := &AgentConfig{
		ID:           "executor-agent",
		Label:        "Sprint Executor",
		Inbox:        "executor",
		Workspace:    "/tmp/executor",
		Capabilities: []string{"code", "test"},
	}

	if err := registry.Register(sourceAgent); err != nil {
		t.Fatalf("failed to register source agent: %v", err)
	}
	if err := registry.Register(targetAgent); err != nil {
		t.Fatalf("failed to register target agent: %v", err)
	}

	// Verify auto_approve_handoffs is false
	retrieved := registry.GetAgentByID("planner-agent")
	if retrieved == nil {
		t.Fatal("expected to find planner-agent")
	}
	if retrieved.AutoApproveHandoffs {
		t.Error("expected AutoApproveHandoffs to be false")
	}
}

func TestExecuteResultSessionID(t *testing.T) {
	// Test that ExecuteResult includes SessionID
	result := &ExecuteResult{
		Success:       true,
		Output:        "Task completed",
		Provider:      "claude-code",
		Duration:      5 * time.Second,
		Cost:          0.05,
		TokensUsed:    1000,
		InputTokens:   500,
		OutputTokens:  500,
		FilesCreated:  []string{"new_file.go"},
		FilesModified: []string{"existing.go"},
		SessionID:     "claude-session-12345",
	}

	if result.SessionID != "claude-session-12345" {
		t.Errorf("expected SessionID 'claude-session-12345', got %q", result.SessionID)
	}
}
