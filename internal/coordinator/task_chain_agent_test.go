package coordinator

import (
	"context"
	"testing"
	"time"
)

// =============================================================================
// Milestone 5: Implementation and Merge Tests
// =============================================================================

func TestOnImplementationComplete_NoGitHubIssue_Skips(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1") // No GitHub issue
	task.Stage = TaskStageImplementation
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &ImplementResult{
		BranchName:    "fix-bug-123",
		WorktreePath:  "/tmp/worktree",
		Duration:      10 * time.Second,
		FilesCreated:  []string{"internal/foo.go"},
		FilesModified: []string{"internal/bar.go"},
	}

	err := tc.OnImplementationComplete(ctx, "task-1", result)
	if err != nil {
		t.Fatalf("OnImplementationComplete failed: %v", err)
	}
}

func TestOnImplementationComplete_SetsStageToMerge(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageImplementation
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &ImplementResult{BranchName: "fix-bug-123"}

	err := tc.OnImplementationComplete(ctx, "task-1", result)
	if err != nil {
		t.Fatalf("OnImplementationComplete failed: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Stage != TaskStageMerge {
		t.Errorf("expected Stage=merge, got %s", retrieved.Stage)
	}
}

func TestOnMergeApproved_UnwatchesIssue(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageMerge
	store.CreateTask(ctx, task)

	watcher := NewApprovalWatcher(nil, store, 60*time.Second)
	watcher.WatchIssue(42, "task-1")

	tc := NewTaskChain(nil, store, watcher)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventMerge,
	}

	err := tc.OnMergeApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnMergeApproved failed: %v", err)
	}

	if watcher.WatchedIssueCount() != 0 {
		t.Error("expected issue to be unwatched after merge")
	}
}

func TestOnMergeApproved_ClearsStage(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageMerge
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventMerge,
	}

	err := tc.OnMergeApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnMergeApproved failed: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Stage != TaskStageNone {
		t.Errorf("expected Stage=none, got %s", retrieved.Stage)
	}
}

// =============================================================================
// Milestone 6: Error and Edge Case Tests
// =============================================================================

func TestOnNeedsRevision_NoError(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageDesign
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventRevision,
	}

	err := tc.OnNeedsRevision(ctx, event)
	if err != nil {
		t.Fatalf("OnNeedsRevision failed: %v", err)
	}
}

func TestOnError_NoError(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageDesign
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.OnError(ctx, "task-1", "Something went wrong")
	if err != nil {
		t.Fatalf("OnError failed: %v", err)
	}
}

func TestOnError_NoGitHubIssue(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1") // No GitHub issue
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.OnError(ctx, "task-1", "Something went wrong")
	if err != nil {
		t.Fatalf("OnError with no GitHub issue failed: %v", err)
	}
}

func TestTaskChain_NilPoster_AllMethodsGraceful(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	// All methods should handle nil poster gracefully

	// StartTask
	err := tc.StartTask(ctx, "task-1", 42)
	if err != nil {
		t.Fatalf("StartTask with nil poster failed: %v", err)
	}

	// OnDesignDocComplete
	err = tc.OnDesignDocComplete(ctx, "task-1", &DesignDocResult{Path: "test.md"})
	if err != nil {
		t.Fatalf("OnDesignDocComplete with nil poster failed: %v", err)
	}

	// OnSprintPlanComplete
	err = tc.OnSprintPlanComplete(ctx, "task-1", &SprintPlanResult{Path: "test.md"})
	if err != nil {
		t.Fatalf("OnSprintPlanComplete with nil poster failed: %v", err)
	}

	// OnImplementationComplete
	err = tc.OnImplementationComplete(ctx, "task-1", &ImplementResult{BranchName: "test"})
	if err != nil {
		t.Fatalf("OnImplementationComplete with nil poster failed: %v", err)
	}
}

// =============================================================================
// M-GENERIC-PIPELINE: Unified OnAgentComplete Tests
// =============================================================================

func TestOnAgentComplete_NoApprovalConfig_Skips(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	// Agent without approval config (not a known agent, no explicit config)
	result := &AgentResult{
		ArtifactPath: "output.txt",
		Duration:     5 * time.Second,
	}

	// Should succeed but skip GitHub workflow
	err := tc.OnAgentComplete(ctx, "task-1", "unknown-agent", result, nil)
	if err != nil {
		t.Fatalf("OnAgentComplete failed: %v", err)
	}
}

func TestOnAgentComplete_KnownAgent_UsesDefaults(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.WorktreePath = "/tmp/worktree"
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil) // nil poster - won't make GitHub calls

	result := &AgentResult{
		ArtifactPath: "design_docs/test.md",
		Duration:     5 * time.Second,
	}

	// design-doc-creator is a known agent with default approval config
	err := tc.OnAgentComplete(ctx, "task-1", "design-doc-creator", result, nil)
	if err != nil {
		t.Fatalf("OnAgentComplete failed: %v", err)
	}

	// Verify design doc path was stored
	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.DesignDocPath != "design_docs/test.md" {
		t.Errorf("expected DesignDocPath='design_docs/test.md', got %s", retrieved.DesignDocPath)
	}
}

func TestOnAgentComplete_CustomAgent_WithRegistry(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	// Create registry with custom agent
	registry := NewAgentRegistry()
	registry.Register(&AgentConfig{
		ID:    "custom-agent",
		Inbox: "custom-inbox",
		Approval: &ApprovalConfig{
			NeedsLabel:    "needs-custom-approval",
			ApprovedLabel: "custom-approved",
		},
	})

	result := &AgentResult{
		ArtifactPath: "output/custom.md",
		AllArtifacts: []string{"output/custom.md", "output/data.json"},
		Duration:     10 * time.Second,
	}

	err := tc.OnAgentComplete(ctx, "task-1", "custom-agent", result, registry)
	if err != nil {
		t.Fatalf("OnAgentComplete failed: %v", err)
	}
}

func TestOnAgentComplete_NoArtifacts_ReturnsError(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	// Empty result - no artifact, no files
	result := &AgentResult{
		Duration: 5 * time.Second,
		// ArtifactPath is empty
		// AllArtifacts is empty
	}

	// design-doc-creator has approval config, so this should fail
	err := tc.OnAgentComplete(ctx, "task-1", "design-doc-creator", result, nil)
	if err == nil {
		t.Error("expected error when no artifacts found")
	}
}

func TestOnAgentComplete_NoGitHubIssue_Skips(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1") // No GitHub issue
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &AgentResult{
		ArtifactPath: "design_docs/test.md",
		Duration:     5 * time.Second,
	}

	// Should succeed but skip GitHub notification
	err := tc.OnAgentComplete(ctx, "task-1", "design-doc-creator", result, nil)
	if err != nil {
		t.Fatalf("OnAgentComplete failed: %v", err)
	}
}

func TestRenderAgentCompleteComment_DesignDoc(t *testing.T) {
	tc := &TaskChain{}

	result := &AgentResult{
		ArtifactPath: "design_docs/test.md",
		Duration:     5 * time.Second,
		Cost:         0.05,
		TokensUsed:   1000,
	}

	approval := &ApprovalConfig{
		GithubCommentTemplate: "design_doc",
		ApprovedLabel:         "design-approved",
	}

	comment, err := tc.renderAgentCompleteComment("design-doc-creator", "task-1", result, "# Test Content", approval)
	if err != nil {
		t.Fatalf("renderAgentCompleteComment failed: %v", err)
	}

	if comment == "" {
		t.Error("expected non-empty comment")
	}
}

func TestRenderGenericCompleteComment(t *testing.T) {
	tc := &TaskChain{}

	result := &AgentResult{
		ArtifactPath: "output/custom.md",
		AllArtifacts: []string{"output/custom.md"},
		Duration:     10 * time.Second,
		Cost:         0.10,
		TokensUsed:   2000,
		InputTokens:  1500,
		OutputTokens: 500,
	}

	approval := &ApprovalConfig{
		ApprovedLabel: "custom-approved",
	}

	comment, err := tc.renderGenericCompleteComment("custom-agent", "task-1", result, "# Custom Content", approval)
	if err != nil {
		t.Fatalf("renderGenericCompleteComment failed: %v", err)
	}

	if comment == "" {
		t.Error("expected non-empty comment")
	}

	// Verify it contains key elements
	if !containsString(comment, "custom-agent") {
		t.Error("expected comment to contain agent ID")
	}
	if !containsString(comment, "task-1") {
		t.Error("expected comment to contain task ID")
	}
	if !containsString(comment, "custom-approved") {
		t.Error("expected comment to contain approved label")
	}
}

// containsString helper already defined in integration_test.go
