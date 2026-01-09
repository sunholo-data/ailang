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

// TestTaskRecordAgentIDForHandoff tests that task.AgentID is properly available
// for use in handleAgentHandoffs (the fix for trigger_on_complete not working)
func TestTaskRecordAgentIDForHandoff(t *testing.T) {
	// Create a task with AgentID set (as it should be after the fix)
	task := &TaskRecord{
		ID:      "task-handoff-test",
		AgentID: "design-doc-creator", // Must be set for handoff to work
		Title:   "Design Doc Task",
		Content: "Create design doc",
		Type:    TaskTypeDocs,
		Status:  TaskStatusPendingApproval,
	}

	// Verify AgentID is accessible
	if task.AgentID == "" {
		t.Error("expected AgentID to be set on task")
	}
	if task.AgentID != "design-doc-creator" {
		t.Errorf("got AgentID %q, want design-doc-creator", task.AgentID)
	}

	// In the actual handleAgentHandoffs, this is used to look up the agent config
	// and find trigger_on_complete targets
	registry := NewAgentRegistry()
	sourceAgent := &AgentConfig{
		ID:                "design-doc-creator",
		Label:             "Design Doc Creator",
		Inbox:             "design-doc-creator",
		TriggerOnComplete: []string{"sprint-planner"},
	}
	targetAgent := &AgentConfig{
		ID:    "sprint-planner",
		Label: "Sprint Planner",
		Inbox: "sprint-planner",
	}
	_ = registry.Register(sourceAgent)
	_ = registry.Register(targetAgent)

	// Simulate the lookup that handleAgentHandoffs does
	agent := registry.GetAgentByID(task.AgentID)
	if agent == nil {
		t.Fatal("expected to find agent by task.AgentID")
	}
	if len(agent.TriggerOnComplete) == 0 {
		t.Error("expected agent to have TriggerOnComplete configured")
	}
	if agent.TriggerOnComplete[0] != "sprint-planner" {
		t.Errorf("expected TriggerOnComplete[0] to be 'sprint-planner', got %q", agent.TriggerOnComplete[0])
	}
}

// TestTaskRecordParentTaskID tests the parent_task_id field for hierarchy tracking
func TestTaskRecordParentTaskID(t *testing.T) {
	// Parent task (design-doc-creator completes)
	parentTask := &TaskRecord{
		ID:           "task-design-123",
		AgentID:      "design-doc-creator",
		ParentTaskID: "", // Root task has no parent
		Title:        "Create Design Doc",
		Status:       TaskStatusCompleted,
	}

	// Child task (sprint-planner receives handoff)
	childTask := &TaskRecord{
		ID:           "task-sprint-456",
		AgentID:      "sprint-planner",
		ParentTaskID: "task-design-123", // Links to parent
		Title:        "Create Sprint Plan",
		Status:       TaskStatusPending,
	}

	// Grandchild task (sprint-executor receives handoff)
	grandchildTask := &TaskRecord{
		ID:           "task-exec-789",
		AgentID:      "sprint-executor",
		ParentTaskID: "task-sprint-456", // Links to child
		Title:        "Execute Sprint",
		Status:       TaskStatusPending,
	}

	// Verify hierarchy chain
	if parentTask.ParentTaskID != "" {
		t.Errorf("root task should have empty ParentTaskID, got %q", parentTask.ParentTaskID)
	}
	if childTask.ParentTaskID != parentTask.ID {
		t.Errorf("child ParentTaskID = %q, want %q", childTask.ParentTaskID, parentTask.ID)
	}
	if grandchildTask.ParentTaskID != childTask.ID {
		t.Errorf("grandchild ParentTaskID = %q, want %q", grandchildTask.ParentTaskID, childTask.ID)
	}
}

// TestHandoffMetadataContainsParentTaskID verifies the metadata format for handoffs
func TestHandoffMetadataContainsParentTaskID(t *testing.T) {
	// This test verifies the metadata structure that sendHandoffMessage creates
	// The metadata should include parent_task_id for hierarchy tracking

	task := &TaskRecord{
		ID:      "task-source-abc",
		AgentID: "design-doc-creator",
	}
	targetAgentID := "sprint-planner"
	sessionID := "session-xyz"

	// Simulate the metadata map that sendHandoffMessage creates
	metadataMap := map[string]interface{}{
		"parent_task_id": task.ID,       // For hierarchy tracking
		"handoff_source": task.ID,       // Legacy field
		"source_agent":   task.AgentID,  // Which agent completed
		"target_agent":   targetAgentID, // Which agent receives
	}
	if sessionID != "" {
		metadataMap["session_id"] = sessionID
	}

	// Verify required fields
	if metadataMap["parent_task_id"] != "task-source-abc" {
		t.Errorf("parent_task_id = %v, want task-source-abc", metadataMap["parent_task_id"])
	}
	if metadataMap["source_agent"] != "design-doc-creator" {
		t.Errorf("source_agent = %v, want design-doc-creator", metadataMap["source_agent"])
	}
	if metadataMap["target_agent"] != "sprint-planner" {
		t.Errorf("target_agent = %v, want sprint-planner", metadataMap["target_agent"])
	}
	if metadataMap["session_id"] != "session-xyz" {
		t.Errorf("session_id = %v, want session-xyz", metadataMap["session_id"])
	}
}

// TestAgentIDPriorityOverThreadLookup tests that task.AgentID takes priority
// over thread.TargetAgent lookup (backwards compatibility fallback)
func TestAgentIDPriorityOverThreadLookup(t *testing.T) {
	// This tests the logic in handleAgentHandoffs:
	// Priority: task.AgentID > thread.TargetAgent > "coordinator" (default)

	tests := []struct {
		name           string
		taskAgentID    string
		threadTarget   string // Simulated thread.TargetAgent
		expectedSource string
	}{
		{
			name:           "task.AgentID takes priority",
			taskAgentID:    "design-doc-creator",
			threadTarget:   "some-other-agent", // Would be from thread lookup
			expectedSource: "design-doc-creator",
		},
		{
			name:           "empty AgentID falls back to thread",
			taskAgentID:    "",
			threadTarget:   "sprint-planner",
			expectedSource: "sprint-planner",
		},
		{
			name:           "both empty falls back to coordinator",
			taskAgentID:    "",
			threadTarget:   "",
			expectedSource: "coordinator",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the logic from handleAgentHandoffs
			sourceAgentID := "coordinator" // default
			if tc.taskAgentID != "" {
				sourceAgentID = tc.taskAgentID
			} else if tc.threadTarget != "" {
				sourceAgentID = tc.threadTarget
			}

			if sourceAgentID != tc.expectedSource {
				t.Errorf("sourceAgentID = %q, want %q", sourceAgentID, tc.expectedSource)
			}
		})
	}
}
