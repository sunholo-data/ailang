package coordinator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIntegration_StoreApprovalRequests tests approval request storage
func TestIntegration_StoreApprovalRequests(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// First create a task (approval requests need a task)
	task := &TaskRecord{
		ID:        "apr-store-task",
		Title:     "Task for approval",
		Content:   "Content",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusPendingApproval,
		CreatedAt: time.Now(),
	}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Create approval request
	request := &ApprovalRequestRecord{
		ID:          "apr-store-1",
		TaskID:      "apr-store-task",
		Type:        string(ApprovalTypeMerge),
		Description: "Please approve",
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := store.CreateApprovalRequest(ctx, request); err != nil {
		t.Fatalf("failed to create approval request: %v", err)
	}

	// List pending approvals
	pending, err := store.ListPendingApprovals(ctx)
	if err != nil {
		t.Fatalf("failed to list pending approvals: %v", err)
	}

	if len(pending) != 1 {
		t.Errorf("expected 1 pending approval, got %d", len(pending))
	}

	// Resolve approval
	if err := store.ResolveApprovalRequest(ctx, "apr-store-1", "approved", "test-user"); err != nil {
		t.Fatalf("failed to resolve approval: %v", err)
	}

	// Should be no pending approvals now
	pending, _ = store.ListPendingApprovals(ctx)
	if len(pending) != 0 {
		t.Errorf("expected 0 pending approvals after resolution, got %d", len(pending))
	}
}

// TestIntegration_TaskPriorityOrdering tests that tasks are returned in priority order
func TestIntegration_TaskPriorityOrdering(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create tasks with different priorities
	tasks := []*TaskRecord{
		{ID: "low-pri", Title: "Low Priority", Content: "c", Type: TaskTypeDocs, Status: TaskStatusPending, Priority: 1, CreatedAt: time.Now()},
		{ID: "high-pri", Title: "High Priority", Content: "c", Type: TaskTypeBugFix, Status: TaskStatusPending, Priority: 5, CreatedAt: time.Now()},
		{ID: "med-pri", Title: "Medium Priority", Content: "c", Type: TaskTypeFeature, Status: TaskStatusPending, Priority: 3, CreatedAt: time.Now()},
	}

	for _, task := range tasks {
		if err := store.CreateTask(ctx, task); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// List with priority ordering
	filter := &TaskFilter{
		Status:    []TaskStatus{TaskStatusPending},
		OrderBy:   "priority",
		OrderDesc: true,
	}

	result, err := store.ListTasks(ctx, filter)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result))
	}

	// Verify order: high, med, low
	if result[0].ID != "high-pri" {
		t.Errorf("expected high-pri first, got %s", result[0].ID)
	}
	if result[1].ID != "med-pri" {
		t.Errorf("expected med-pri second, got %s", result[1].ID)
	}
	if result[2].ID != "low-pri" {
		t.Errorf("expected low-pri third, got %s", result[2].ID)
	}
}

// TestIntegration_DuplicateTaskDetection tests fingerprint-based deduplication
func TestIntegration_DuplicateTaskDetection(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create first task with fingerprint
	task1 := &TaskRecord{
		ID:        "orig-task",
		Title:     "Original Task",
		Content:   "Fix the parser bug",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task1); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	fingerprint := uint64(0xABCDEF123456)
	if err := store.SetTaskFingerprint(ctx, task1.ID, fingerprint); err != nil {
		t.Fatalf("failed to set fingerprint: %v", err)
	}

	// Try to find duplicate with same fingerprint
	dup, err := store.FindDuplicateTask(ctx, fingerprint, 0.9)
	if err != nil {
		t.Fatalf("failed to find duplicate: %v", err)
	}

	if dup == nil {
		t.Fatal("expected to find duplicate task")
	}

	if dup.ID != task1.ID {
		t.Errorf("expected duplicate ID %s, got %s", task1.ID, dup.ID)
	}

	// Different fingerprint should not match
	dup2, _ := store.FindDuplicateTask(ctx, 0x111111111111, 0.9)
	if dup2 != nil {
		t.Error("expected no duplicate for different fingerprint")
	}
}

// TestIntegration_TaskEventStorage tests streaming event storage and retrieval
func TestIntegration_TaskEventStorage(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create a task first
	task := &TaskRecord{
		ID:        "event-task",
		Title:     "Event Test Task",
		Content:   "Test event storage",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusRunning,
		ThreadID:  "thread-event-1",
		CreatedAt: time.Now(),
	}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Store some events
	events := []*TaskEventRecord{
		{TaskID: task.ID, ThreadID: task.ThreadID, StreamType: "status", Status: "running"},
		{TaskID: task.ID, ThreadID: task.ThreadID, StreamType: "text", Text: "Processing..."},
		{TaskID: task.ID, ThreadID: task.ThreadID, StreamType: "text", Text: "Done!"},
		{TaskID: task.ID, ThreadID: task.ThreadID, StreamType: "status", Status: "completed"},
	}

	for _, event := range events {
		if err := store.StoreTaskEvent(ctx, event); err != nil {
			t.Fatalf("failed to store event: %v", err)
		}
	}

	// Retrieve events (limit 100)
	retrieved, err := store.GetTaskEvents(ctx, task.ID, 100)
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}

	if len(retrieved) != 4 {
		t.Errorf("expected 4 events, got %d", len(retrieved))
	}

	// Verify order (should be chronological)
	if retrieved[0].StreamType != "status" {
		t.Errorf("expected first event type 'status', got %s", retrieved[0].StreamType)
	}
}

// TestIntegration_StaleTasks tests recovery of stale tasks
func TestIntegration_StaleTasks(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create a "stale" running task (started long ago)
	oldTime := time.Now().Add(-10 * time.Minute)
	task := &TaskRecord{
		ID:        "stale-task",
		Title:     "Stale Task",
		Content:   "This task got stuck",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusRunning,
		CreatedAt: oldTime,
		StartedAt: &oldTime,
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Recover stale tasks (threshold: 5 minutes)
	recovered, err := store.RecoverStaleTasks(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to recover stale tasks: %v", err)
	}

	if recovered != 1 {
		t.Errorf("expected 1 recovered task, got %d", recovered)
	}

	// Task should now be cancelled
	retrieved, _ := store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusCancelled {
		t.Errorf("expected status %s, got %s", TaskStatusCancelled, retrieved.Status)
	}
}

// TestIntegration_WorktreeManager tests worktree creation and cleanup
func TestIntegration_WorktreeManager(t *testing.T) {
	// Skip if coordinator is running (has existing worktrees)
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".ailang", "state", "coordinator.pid")); err == nil {
		t.Skip("coordinator is running, skip worktree test to avoid conflicts")
	}

	// Skip if not in a git repo
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		// Find git root
		wd, _ := os.Getwd()
		for wd != "/" {
			if _, err := os.Stat(filepath.Join(wd, ".git")); err == nil {
				break
			}
			wd = filepath.Dir(wd)
		}
		if wd == "/" {
			t.Skip("not in a git repository")
		}
	}

	tmpDir := t.TempDir()
	worktreeBase := filepath.Join(tmpDir, "worktrees")

	// Get git root
	wd, _ := os.Getwd()
	gitRoot := wd
	for gitRoot != "/" {
		if _, err := os.Stat(filepath.Join(gitRoot, ".git")); err == nil {
			break
		}
		gitRoot = filepath.Dir(gitRoot)
	}

	// Get current branch name (CI may only have 'dev', not 'main')
	branchCmd := exec.Command("git", "-C", gitRoot, "rev-parse", "--abbrev-ref", "HEAD")
	branchOutput, err := branchCmd.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	currentBranch := strings.TrimSpace(string(branchOutput))

	mgr, err := NewWorktreeManager(gitRoot, worktreeBase, 3)
	if err != nil {
		t.Fatalf("failed to create worktree manager: %v", err)
	}

	// Record initial count (may have existing worktrees from other tests/runs)
	initialCount := mgr.Count()

	// Create a worktree from the current branch (not hardcoded 'main')
	wt, err := mgr.CreateWorktree("test-task-integ-1", currentBranch)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	if wt.TaskID != "test-task-integ-1" {
		t.Errorf("expected task ID test-task-integ-1, got %s", wt.TaskID)
	}

	// Verify directory exists
	if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
		t.Error("worktree path does not exist")
	}

	// Count should increase by 1
	if mgr.Count() != initialCount+1 {
		t.Errorf("expected %d worktrees after create, got %d", initialCount+1, mgr.Count())
	}

	// Get the worktree we created
	retrieved, found := mgr.GetWorktree("test-task-integ-1")
	if !found {
		t.Error("expected to find worktree test-task-integ-1")
	}
	if retrieved.TaskID != "test-task-integ-1" {
		t.Errorf("expected task ID test-task-integ-1, got %s", retrieved.TaskID)
	}

	// Remove worktree
	if err := mgr.RemoveWorktree("test-task-integ-1"); err != nil {
		t.Fatalf("failed to remove worktree: %v", err)
	}

	// Count should be back to initial
	if mgr.Count() != initialCount {
		t.Errorf("expected %d worktrees after remove, got %d", initialCount, mgr.Count())
	}
}

// TestIntegration_EndToEnd_SimplePath tests a complete simple path through the system
func TestIntegration_EndToEnd_SimplePath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e.db")

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Step 1: Create task
	task := &TaskRecord{
		ID:        "e2e-task-1",
		MessageID: "e2e-msg-1",
		Title:     "End-to-End Test Task",
		Content:   "Fix the critical bug in authentication",
		Type:      TaskTypeBugFix,
		Priority:  5,
		Status:    TaskStatusPending,
		Workspace: tmpDir,
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("Step 1 failed - create task: %v", err)
	}

	// Verify task exists
	retrieved, err := store.GetTask(ctx, task.ID)
	if err != nil || retrieved == nil {
		t.Fatalf("Step 1 verification failed: %v", err)
	}

	// Step 2: Execute with mock provider
	mockProvider := NewIntegrationMockProvider("e2e-mock")
	mockProvider.SetExecuteFunc(func(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
		return &ExecuteResult{
			Success:       true,
			Output:        "Fixed the authentication bug by checking token expiry",
			Duration:      2 * time.Second,
			Cost:          0.02,
			TokensUsed:    500,
			InputTokens:   300,
			OutputTokens:  200,
			Provider:      "e2e-mock",
			FilesModified: []string{"auth.go", "auth_test.go"},
		}, nil
	})

	executor := NewTaskExecutor(mockProvider)

	// Mark running
	if err := store.MarkTaskRunning(ctx, task.ID, "e2e-mock", ""); err != nil {
		t.Fatalf("Step 2 failed - mark running: %v", err)
	}

	// Execute
	analyzedTask := &AnalyzedTask{
		Task: &Task{
			ID:      task.ID,
			Title:   task.Title,
			Content: task.Content,
		},
		Type: task.Type,
	}

	opts := &ExecuteOptions{
		Timeout:   30 * time.Second,
		Workspace: tmpDir,
	}

	result, err := executor.Execute(ctx, analyzedTask, opts)
	if err != nil {
		t.Fatalf("Step 2 failed - execute: %v", err)
	}

	if !result.Success {
		t.Fatalf("Step 2 failed - execution not successful: %s", result.Error)
	}

	// Step 3: Mark completed
	if err := store.MarkTaskCompleted(ctx, task.ID, result); err != nil {
		t.Fatalf("Step 3 failed - mark completed: %v", err)
	}

	// Final verification
	final, _ := store.GetTask(ctx, task.ID)
	if final.Status != TaskStatusCompleted {
		t.Errorf("final status expected %s, got %s", TaskStatusCompleted, final.Status)
	}
	if final.Cost != 0.02 {
		t.Errorf("final cost expected 0.02, got %f", final.Cost)
	}
	if final.TokensUsed != 500 {
		t.Errorf("final tokens expected 500, got %d", final.TokensUsed)
	}

	t.Logf("End-to-end test passed: task %s completed with cost $%.4f", task.ID, final.Cost)
}

// TestIntegration_GitHubPipelineStages tests the GitHub-driven pipeline stages
func TestIntegration_GitHubPipelineStages(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create a GitHub-linked task
	task := &TaskRecord{
		ID:          "gh-pipeline-1",
		MessageID:   "msg-gh-1",
		Title:       "Add new feature",
		Content:     "Implement the frobnitz widget",
		Type:        TaskTypeFeature,
		Status:      TaskStatusPending,
		GithubIssue: 42,
		Stage:       TaskStageDesign,
		CreatedAt:   time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Verify stage-aware directive for design stage
	directive := BuildStageDirective(task)
	if directive == task.Content {
		t.Error("expected stage-aware directive, got original content")
	}
	if !containsString(directive, "design-doc-creator") {
		t.Error("design stage directive should mention design-doc-creator skill")
	}
	if !containsString(directive, "DESIGN_DOC_PATH:") {
		t.Error("design stage directive should include output format")
	}

	// Test output parsing for design stage
	designOutput := `Created design document for frobnitz widget.
DESIGN_DOC_PATH: design_docs/planned/v0_6_3/frobnitz-widget.md
Ready for review.`

	designResult := ParseStageOutput(designOutput, TaskStageDesign)
	if designResult.DesignDocPath != "design_docs/planned/v0_6_3/frobnitz-widget.md" {
		t.Errorf("expected design doc path, got: %s", designResult.DesignDocPath)
	}

	// Transition to sprint stage
	if err := store.SetTaskStage(ctx, task.ID, TaskStageSprint); err != nil {
		t.Fatalf("failed to set stage: %v", err)
	}

	// Verify sprint directive
	task.Stage = TaskStageSprint
	sprintDirective := BuildStageDirective(task)
	if !containsString(sprintDirective, "sprint-planner") {
		t.Error("sprint stage directive should mention sprint-planner skill")
	}

	// Test output parsing for sprint stage
	sprintOutput := `Created sprint plan.
SPRINT_PLAN_PATH: design_docs/planned/v0_6_3/frobnitz-sprint-plan.md
Ready for execution.`

	sprintResult := ParseStageOutput(sprintOutput, TaskStageSprint)
	if sprintResult.SprintPlanPath != "design_docs/planned/v0_6_3/frobnitz-sprint-plan.md" {
		t.Errorf("expected sprint plan path, got: %s", sprintResult.SprintPlanPath)
	}

	// Test RequeueTask
	if err := store.MarkTaskRunning(ctx, task.ID, "test-provider", ""); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusRunning {
		t.Errorf("expected running, got %s", retrieved.Status)
	}

	if err := store.RequeueTask(ctx, task.ID); err != nil {
		t.Fatalf("failed to requeue task: %v", err)
	}

	retrieved, _ = store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusPending {
		t.Errorf("expected pending after requeue, got %s", retrieved.Status)
	}

	// Verify implementation stage parsing
	implOutput := `Implementation complete.
IMPLEMENTATION_COMPLETE: true
BRANCH_NAME: feature/frobnitz-widget
FILES_CREATED: internal/frobnitz/widget.go, internal/frobnitz/widget_test.go
FILES_MODIFIED: internal/registry/registry.go`

	implResult := ParseStageOutput(implOutput, TaskStageImplementation)
	if implResult.BranchName != "feature/frobnitz-widget" {
		t.Errorf("expected branch name, got: %s", implResult.BranchName)
	}
	if len(implResult.FilesCreated) != 2 {
		t.Errorf("expected 2 files created, got: %v", implResult.FilesCreated)
	}
	if len(implResult.FilesModified) != 1 {
		t.Errorf("expected 1 file modified, got: %v", implResult.FilesModified)
	}

	t.Log("GitHub pipeline stages test passed")
}
