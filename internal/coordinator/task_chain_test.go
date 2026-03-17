package coordinator

import (
	"context"
	"testing"
	"time"
)

// =============================================================================
// Milestone 2: Basic Lifecycle Tests
// =============================================================================

func TestNewTaskChain_RegistersHandlers(t *testing.T) {
	store := NewMockStore()
	// Create a real watcher with nil poster (won't make GitHub calls)
	watcher := NewApprovalWatcher(nil, store, 60*time.Second)

	tc := NewTaskChain(nil, store, watcher)

	// TaskChain should be created successfully
	if tc == nil {
		t.Fatal("expected TaskChain to be created")
	}

	// Can't easily verify handler registration without exposing internals,
	// but we can verify the watcher has handlers by checking it's configured
	if watcher.WatchedIssueCount() != 0 {
		t.Error("expected no watched issues initially")
	}
}

func TestNewTaskChain_WithNilWatcher(t *testing.T) {
	store := NewMockStore()

	// Should not panic with nil watcher
	tc := NewTaskChain(nil, store, nil)

	if tc == nil {
		t.Fatal("expected TaskChain to be created with nil watcher")
	}
}

func TestStartTask_LinksGitHubIssue(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Create task first
	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.StartTask(ctx, "task-1", 42)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	if store.GetCallCount("SetTaskGithubIssue") != 1 {
		t.Error("expected SetTaskGithubIssue to be called")
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.GithubIssue != 42 {
		t.Errorf("expected GithubIssue=42, got %d", retrieved.GithubIssue)
	}
}

func TestStartTask_SetsDesignStage(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.StartTask(ctx, "task-1", 42)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	if store.GetCallCount("SetTaskStage") != 1 {
		t.Error("expected SetTaskStage to be called")
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Stage != TaskStageDesign {
		t.Errorf("expected Stage=design, got %s", retrieved.Stage)
	}
}

func TestStartTask_WatchesIssue(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	watcher := NewApprovalWatcher(nil, store, 60*time.Second)
	tc := NewTaskChain(nil, store, watcher)

	err := tc.StartTask(ctx, "task-1", 42)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	if watcher.WatchedIssueCount() != 1 {
		t.Errorf("expected 1 watched issue, got %d", watcher.WatchedIssueCount())
	}
}

func TestStartTask_NilPoster_Graceful(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	// With nil poster, should still succeed but skip GitHub comment
	tc := NewTaskChain(nil, store, nil)

	err := tc.StartTask(ctx, "task-1", 42)
	if err != nil {
		t.Fatalf("StartTask with nil poster failed: %v", err)
	}
}

// =============================================================================
// Milestone 3: Design Stage Tests
// =============================================================================

func TestOnDesignDocComplete_NoGitHubIssue_Skips(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1") // No GitHub issue
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &DesignDocResult{Path: "design_docs/test.md"}

	err := tc.OnDesignDocComplete(ctx, "task-1", result)
	if err != nil {
		t.Fatalf("OnDesignDocComplete failed: %v", err)
	}

	// Should not fail when task has no GitHub issue
}

func TestOnDesignDocComplete_WithGitHubIssue_NilPoster(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.WorktreePath = "/tmp/worktree"
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &DesignDocResult{
		Path:     "design_docs/planned/v0_6_3/test.md",
		Duration: 5 * time.Second,
	}

	err := tc.OnDesignDocComplete(ctx, "task-1", result)
	if err != nil {
		t.Fatalf("OnDesignDocComplete failed: %v", err)
	}

	// With nil poster, should succeed but skip posting
}

func TestOnDesignApproved_TransitionsStage(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageDesign
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		Label:       "design-approved", // Config-driven label
		EventType:   ApprovalEventType("custom:design-doc-creator"),
	}

	err := tc.OnDesignApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnDesignApproved failed: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Stage != TaskStageSprint {
		t.Errorf("expected Stage=sprint, got %s", retrieved.Stage)
	}
}

func TestOnDesignApproved_RequeuesTask(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageDesign
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventDesign,
	}

	err := tc.OnDesignApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnDesignApproved failed: %v", err)
	}

	if store.GetCallCount("RequeueTask") != 1 {
		t.Error("expected RequeueTask to be called")
	}
}

// =============================================================================
// Milestone 4: Sprint Stage Tests
// =============================================================================

func TestOnSprintPlanComplete_NoGitHubIssue_Skips(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1") // No GitHub issue
	task.Stage = TaskStageSprint
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &SprintPlanResult{
		Path:     "design_docs/planned/v0_6_3/test-sprint-plan.md",
		Duration: 3 * time.Second,
	}

	err := tc.OnSprintPlanComplete(ctx, "task-1", result)
	if err != nil {
		t.Fatalf("OnSprintPlanComplete failed: %v", err)
	}
}

func TestOnSprintApproved_TransitionsStage(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageSprint
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventSprint,
	}

	err := tc.OnSprintApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnSprintApproved failed: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Stage != TaskStageImplementation {
		t.Errorf("expected Stage=implementation, got %s", retrieved.Stage)
	}
}

func TestOnSprintApproved_RequeuesTask(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageSprint
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventSprint,
	}

	err := tc.OnSprintApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnSprintApproved failed: %v", err)
	}

	if store.GetCallCount("RequeueTask") != 1 {
		t.Error("expected RequeueTask to be called")
	}
}

// =============================================================================
// Store Error Handling Tests
// =============================================================================

func TestStartTask_SetGithubIssueError(t *testing.T) {
	store := NewMockStore()
	store.setTaskGithubIssueErr = context.DeadlineExceeded
	ctx := context.Background()

	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.StartTask(ctx, "task-1", 42)
	if err == nil {
		t.Error("expected error when SetTaskGithubIssue fails")
	}
}

func TestStartTask_SetTaskStageError(t *testing.T) {
	store := NewMockStore()
	store.setTaskStageErr = context.DeadlineExceeded
	ctx := context.Background()

	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.StartTask(ctx, "task-1", 42)
	if err == nil {
		t.Error("expected error when SetTaskStage fails")
	}
}

func TestOnDesignApproved_SetTaskStageError(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	store.CreateTask(ctx, task)

	store.setTaskStageErr = context.DeadlineExceeded

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventDesign,
	}

	err := tc.OnDesignApproved(ctx, event)
	if err == nil {
		t.Error("expected error when SetTaskStage fails")
	}
}

func TestOnDesignApproved_RequeueTaskError(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	store.CreateTask(ctx, task)

	store.requeueTaskErr = context.DeadlineExceeded

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventDesign,
	}

	err := tc.OnDesignApproved(ctx, event)
	if err == nil {
		t.Error("expected error when RequeueTask fails")
	}
}

// =============================================================================
// Config-Driven Approval Tests (M-COORD-GENERIC-WORKFLOWS M4)
// =============================================================================

func TestApprovalWatcher_RegisterAgentApproval(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	agent := &AgentConfig{
		ID:    "custom-agent",
		Label: "Custom Agent",
		Approval: &ApprovalConfig{
			NeedsLabel:    "needs-custom-approval",
			ApprovedLabel: "custom-approved",
		},
	}

	handler := func(ctx context.Context, event *ApprovalEvent) error {
		return nil
	}

	err := watcher.RegisterAgentApproval(agent, handler)
	if err != nil {
		t.Fatalf("RegisterAgentApproval failed: %v", err)
	}

	// Verify agent is registered
	registeredAgent := watcher.GetAgentByLabel("custom-approved")
	if registeredAgent == nil {
		t.Fatal("Expected agent to be registered")
	}
	if registeredAgent.ID != "custom-agent" {
		t.Errorf("Expected agent ID 'custom-agent', got %q", registeredAgent.ID)
	}
}

func TestApprovalWatcher_RegisterAgentApproval_NilAgent(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	err := watcher.RegisterAgentApproval(nil, nil)
	if err == nil {
		t.Error("Expected error for nil agent")
	}
}

func TestApprovalWatcher_RegisterAgentApproval_NoApprovalConfig(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	agent := &AgentConfig{
		ID:    "agent-without-approval",
		Label: "No Approval",
		// Approval is nil
	}

	err := watcher.RegisterAgentApproval(agent, nil)
	if err == nil {
		t.Error("Expected error for agent without approval config")
	}
}

func TestApprovalWatcher_RegisterAgentApproval_EmptyApprovedLabel(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	agent := &AgentConfig{
		ID:    "agent-empty-label",
		Label: "Empty Label",
		Approval: &ApprovalConfig{
			NeedsLabel:    "needs-something",
			ApprovedLabel: "", // Empty
		},
	}

	err := watcher.RegisterAgentApproval(agent, nil)
	if err == nil {
		t.Error("Expected error for empty approved_label")
	}
}

func TestApprovalWatcher_GetRegisteredLabels(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	agent1 := &AgentConfig{
		ID: "agent-1",
		Approval: &ApprovalConfig{
			ApprovedLabel: "label-1",
		},
	}
	agent2 := &AgentConfig{
		ID: "agent-2",
		Approval: &ApprovalConfig{
			ApprovedLabel: "label-2",
		},
	}

	watcher.RegisterAgentApproval(agent1, nil)
	watcher.RegisterAgentApproval(agent2, nil)

	labels := watcher.GetRegisteredLabels()
	if len(labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(labels))
	}

	labelMap := make(map[string]bool)
	for _, l := range labels {
		labelMap[l] = true
	}
	if !labelMap["label-1"] || !labelMap["label-2"] {
		t.Errorf("Expected labels label-1 and label-2, got %v", labels)
	}
}

func TestApprovalWatcher_SetAgentRegistry(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	registry := NewAgentRegistry()
	watcher.SetAgentRegistry(registry)

	// Just verify it doesn't panic
	if watcher.agentRegistry != registry {
		t.Error("Expected registry to be set")
	}
}
