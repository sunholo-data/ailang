package coordinator

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestGitHubWorkflow_DesignDocStage tests the design document approval workflow
func TestGitHubWorkflow_DesignDocStage(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	mockClient := NewMockGitHubClient()
	ctx := context.Background()
	issueNum := 1001

	// Setup labels
	mockClient.EnsureLabel("test-repo", "coordinator:bug", "Bug fix", "D73A4A")
	mockClient.EnsureLabel("test-repo", "needs-design-approval", "Needs design approval", "B60205")
	mockClient.EnsureLabel("test-repo", "design-approved", "Design approved", "0E8A16")

	// Step 1: Issue created and task established
	task := &TaskRecord{
		ID:          "design-task-1",
		Title:       "Design new feature",
		Content:     "Need to create design doc for caching feature",
		Type:        TaskTypeFeature,
		Status:      TaskStatusPending,
		GithubIssue: issueNum,
		CreatedAt:   time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Step 2: Task assigned - post status and add label
	mockClient.AddComment("test-repo", issueNum, "🤖 Starting design document creation")
	mockClient.AddLabelToIssue("test-repo", issueNum, "coordinator:bug")
	mockClient.AddLabelToIssue("test-repo", issueNum, "needs-design-approval")

	// Step 3: Mark task as running
	if err := store.MarkTaskRunning(ctx, task.ID, "design-doc-creator", "wt-design-1"); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusRunning {
		t.Errorf("expected running status, got %s", retrieved.Status)
	}

	// Step 4: Design doc created - mark pending approval
	result := &ExecuteResult{
		Success:      true,
		Output:       "Design document created",
		Duration:     2 * time.Second,
		Cost:         0.02,
		TokensUsed:   500,
		InputTokens:  300,
		OutputTokens: 200,
	}

	if err := store.MarkTaskPendingApproval(ctx, task.ID, "/tmp/wt-design-1", result); err != nil {
		t.Fatalf("failed to mark pending approval: %v", err)
	}

	retrieved, _ = store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusPendingApproval {
		t.Errorf("expected pending approval status, got %s", retrieved.Status)
	}

	// Step 5: Human approves - update labels
	mockClient.RemoveLabelFromIssue("test-repo", issueNum, "needs-design-approval")
	mockClient.AddLabelToIssue("test-repo", issueNum, "design-approved")
	mockClient.AddComment("test-repo", issueNum, "✅ Design document approved")

	// Verify label state
	labels := mockClient.GetLabels(issueNum)
	hasDesignApproved := false
	for _, l := range labels {
		if l == "design-approved" {
			hasDesignApproved = true
		}
		if l == "needs-design-approval" {
			t.Errorf("should not have needs-design-approval after approval")
		}
	}

	if !hasDesignApproved {
		t.Error("expected design-approved label")
	}

	// Step 6: Task completes
	if err := store.MarkTaskCompleted(ctx, task.ID, result); err != nil {
		t.Fatalf("failed to mark completed: %v", err)
	}

	retrieved, _ = store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusCompleted {
		t.Errorf("expected completed status, got %s", retrieved.Status)
	}

	t.Logf("Design document workflow completed: issue %d approved", issueNum)
}

// TestGitHubWorkflow_SprintPlanning tests the sprint planning approval workflow
func TestGitHubWorkflow_SprintPlanning(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	mockClient := NewMockGitHubClient()
	ctx := context.Background()
	issueNum := 1002

	// Setup labels
	mockClient.EnsureLabel("test-repo", "design-approved", "Design approved", "0E8A16")
	mockClient.EnsureLabel("test-repo", "needs-sprint-approval", "Needs sprint approval", "B60205")
	mockClient.EnsureLabel("test-repo", "sprint-approved", "Sprint approved", "0E8A16")

	// Task transitioned from design to sprint
	task := &TaskRecord{
		ID:          "sprint-task-1",
		Title:       "Plan sprint for feature",
		Content:     "Create sprint plan from approved design doc",
		Type:        TaskTypeFeature,
		Status:      TaskStatusPending,
		GithubIssue: issueNum,
		CreatedAt:   time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Add labels from previous stage
	mockClient.AddLabelToIssue("test-repo", issueNum, "design-approved")
	mockClient.AddLabelToIssue("test-repo", issueNum, "needs-sprint-approval")
	mockClient.AddComment("test-repo", issueNum, "📋 Sprint planning in progress")

	// Sprint planner runs
	if err := store.MarkTaskRunning(ctx, task.ID, "sprint-planner", "wt-sprint-1"); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}

	// Sprint plan created
	result := &ExecuteResult{
		Success:      true,
		Output:       "Sprint plan created",
		Duration:     1 * time.Second,
		Cost:         0.01,
		TokensUsed:   200,
		InputTokens:  100,
		OutputTokens: 100,
	}

	if err := store.MarkTaskPendingApproval(ctx, task.ID, "/tmp/wt-sprint-1", result); err != nil {
		t.Fatalf("failed to mark pending approval: %v", err)
	}

	// Human approves sprint
	mockClient.RemoveLabelFromIssue("test-repo", issueNum, "needs-sprint-approval")
	mockClient.AddLabelToIssue("test-repo", issueNum, "sprint-approved")
	mockClient.AddComment("test-repo", issueNum, "✅ Sprint plan approved")

	// Verify transition
	labels := mockClient.GetLabels(issueNum)
	hasSprintApproved := false
	for _, l := range labels {
		if l == "sprint-approved" {
			hasSprintApproved = true
		}
	}

	if !hasSprintApproved {
		t.Error("expected sprint-approved label")
	}

	if err := store.MarkTaskCompleted(ctx, task.ID, result); err != nil {
		t.Fatalf("failed to mark completed: %v", err)
	}

	t.Logf("Sprint planning workflow completed: issue %d", issueNum)
}

// TestGitHubWorkflow_Implementation tests the implementation stage
func TestGitHubWorkflow_Implementation(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	mockClient := NewMockGitHubClient()
	ctx := context.Background()
	issueNum := 1003

	// Setup labels
	mockClient.EnsureLabel("test-repo", "sprint-approved", "Sprint approved", "0E8A16")
	mockClient.EnsureLabel("test-repo", "needs-merge-approval", "Needs merge approval", "B60205")
	mockClient.EnsureLabel("test-repo", "merge-approved", "Merge approved", "0E8A16")

	// Task for implementation
	task := &TaskRecord{
		ID:          "impl-task-1",
		Title:       "Implement feature from sprint plan",
		Content:     "Execute the sprint plan and implement feature",
		Type:        TaskTypeFeature,
		Status:      TaskStatusPending,
		GithubIssue: issueNum,
		CreatedAt:   time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Transition from sprint to implementation
	mockClient.AddLabelToIssue("test-repo", issueNum, "sprint-approved")
	mockClient.AddLabelToIssue("test-repo", issueNum, "needs-merge-approval")
	mockClient.AddComment("test-repo", issueNum, "⚙️ Implementation in progress")

	if err := store.MarkTaskRunning(ctx, task.ID, "sprint-executor", "wt-impl-1"); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}

	// Implementation complete
	result := &ExecuteResult{
		Success:       true,
		Output:        "Feature implemented and tested",
		Duration:      10 * time.Second,
		Cost:          0.1,
		TokensUsed:    2000,
		InputTokens:   1200,
		OutputTokens:  800,
		FilesModified: []string{"feature.go", "feature_test.go"},
	}

	if err := store.MarkTaskPendingApproval(ctx, task.ID, "/tmp/wt-impl-1", result); err != nil {
		t.Fatalf("failed to mark pending approval: %v", err)
	}

	// Simulate approval
	mockClient.RemoveLabelFromIssue("test-repo", issueNum, "needs-merge-approval")
	mockClient.AddLabelToIssue("test-repo", issueNum, "merge-approved")
	mockClient.AddComment("test-repo", issueNum, "✅ Implementation approved and merged")

	// Final state
	if err := store.MarkTaskCompleted(ctx, task.ID, result); err != nil {
		t.Fatalf("failed to mark completed: %v", err)
	}

	// Close issue
	mockClient.CloseIssue("test-repo", issueNum, "✨ Issue resolved in PR")

	if !mockClient.IsClosed(issueNum) {
		t.Error("expected issue to be closed")
	}

	t.Logf("Implementation workflow completed: issue %d closed", issueNum)
}

// TestGitHubWorkflow_RejectionAndRevision tests rejection and revision workflow
func TestGitHubWorkflow_RejectionAndRevision(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	mockClient := NewMockGitHubClient()
	ctx := context.Background()
	issueNum := 1004

	// Setup labels
	mockClient.EnsureLabel("test-repo", "needs-design-approval", "Needs design approval", "B60205")
	mockClient.EnsureLabel("test-repo", "needs-revision", "Needs revision", "FFA500")
	mockClient.EnsureLabel("test-repo", "design-approved", "Design approved", "0E8A16")

	task := &TaskRecord{
		ID:          "revision-task-1",
		Title:       "Design with revision needed",
		Content:     "Create design doc that needs revision",
		Type:        TaskTypeFeature,
		Status:      TaskStatusPending,
		GithubIssue: issueNum,
		CreatedAt:   time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Initial workflow
	mockClient.AddLabelToIssue("test-repo", issueNum, "needs-design-approval")
	mockClient.AddComment("test-repo", issueNum, "📝 Design document submitted")

	if err := store.MarkTaskRunning(ctx, task.ID, "design-doc-creator", "wt-rev-1"); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}

	result := &ExecuteResult{
		Success:      true,
		Output:       "Design created",
		Duration:     2 * time.Second,
		Cost:         0.02,
		TokensUsed:   500,
		InputTokens:  300,
		OutputTokens: 200,
	}

	if err := store.MarkTaskPendingApproval(ctx, task.ID, "/tmp/wt-rev-1", result); err != nil {
		t.Fatalf("failed to mark pending approval: %v", err)
	}

	// Human requests revision instead of approving
	mockClient.RemoveLabelFromIssue("test-repo", issueNum, "needs-design-approval")
	mockClient.AddLabelToIssue("test-repo", issueNum, "needs-revision")
	mockClient.AddComment("test-repo", issueNum, "⚠️ Design needs revision. Issues: 1) Missing error handling 2) No caching strategy")

	// Verify rejection state
	labels := mockClient.GetLabels(issueNum)
	hasRevisionNeeded := false
	for _, l := range labels {
		if l == "needs-revision" {
			hasRevisionNeeded = true
		}
	}

	if !hasRevisionNeeded {
		t.Error("expected needs-revision label")
	}

	// Task marked for revision (status should be pending for retry)
	// In real system, this would trigger a new run of design-doc-creator
	if err := store.MarkTaskFailed(ctx, task.ID, fmt.Errorf("revision requested by human reviewer")); err != nil {
		t.Fatalf("failed to mark as failed: %v", err)
	}

	// Simulate another run after revision
	newTask := &TaskRecord{
		ID:          "revision-task-1-v2",
		Title:       "Design with revision (revised)",
		Content:     "Design doc incorporating feedback",
		Type:        TaskTypeFeature,
		Status:      TaskStatusPending,
		GithubIssue: issueNum,
		CreatedAt:   time.Now(),
	}

	if err := store.CreateTask(ctx, newTask); err != nil {
		t.Fatalf("failed to create revised task: %v", err)
	}

	if err := store.MarkTaskRunning(ctx, newTask.ID, "design-doc-creator", "wt-rev-2"); err != nil {
		t.Fatalf("failed to mark revised running: %v", err)
	}

	if err := store.MarkTaskPendingApproval(ctx, newTask.ID, "/tmp/wt-rev-2", result); err != nil {
		t.Fatalf("failed to mark revised pending approval: %v", err)
	}

	// Now approve
	mockClient.RemoveLabelFromIssue("test-repo", issueNum, "needs-revision")
	mockClient.AddLabelToIssue("test-repo", issueNum, "design-approved")
	mockClient.AddComment("test-repo", issueNum, "✅ Revised design approved")

	if err := store.MarkTaskCompleted(ctx, newTask.ID, result); err != nil {
		t.Fatalf("failed to mark revised completed: %v", err)
	}

	labels = mockClient.GetLabels(issueNum)
	hasDesignApproved := false
	for _, l := range labels {
		if l == "design-approved" {
			hasDesignApproved = true
		}
	}

	if !hasDesignApproved {
		t.Error("expected design-approved after revision")
	}

	t.Logf("Revision workflow completed: issue %d approved after revision", issueNum)
}

// TestGitHubWorkflow_ConcurrentApprovals tests handling multiple concurrent approvals
func TestGitHubWorkflow_ConcurrentApprovals(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	mockClient := NewMockGitHubClient()
	ctx := context.Background()

	// Setup
	mockClient.EnsureLabel("test-repo", "needs-design-approval", "Needs design approval", "B60205")
	mockClient.EnsureLabel("test-repo", "design-approved", "Design approved", "0E8A16")

	// Create multiple tasks
	numTasks := 5
	tasks := make([]*TaskRecord, numTasks)
	for i := 0; i < numTasks; i++ {
		task := &TaskRecord{
			ID:          fmt.Sprintf("concurrent-task-%d", i),
			Title:       fmt.Sprintf("Task %d", i),
			Content:     fmt.Sprintf("Content %d", i),
			Type:        TaskTypeBugFix,
			Status:      TaskStatusPending,
			GithubIssue: 2000 + i,
			CreatedAt:   time.Now(),
		}
		tasks[i] = task

		if err := store.CreateTask(ctx, task); err != nil {
			t.Fatalf("failed to create task %d: %v", i, err)
		}

		mockClient.AddLabelToIssue("test-repo", task.GithubIssue, "needs-design-approval")
	}

	// Simulate concurrent approvals
	var wg sync.WaitGroup
	errors := make(chan error, numTasks)

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			task := tasks[idx]
			issueNum := task.GithubIssue

			// Run task
			if err := store.MarkTaskRunning(ctx, task.ID, "provider", "wt"); err != nil {
				errors <- err
				return
			}

			result := &ExecuteResult{
				Success:    true,
				Output:     "Done",
				Duration:   1 * time.Second,
				Cost:       0.01,
				TokensUsed: 100,
			}

			if err := store.MarkTaskPendingApproval(ctx, task.ID, "/tmp/wt", result); err != nil {
				errors <- err
				return
			}

			// Approve in parallel
			mockClient.RemoveLabelFromIssue("test-repo", issueNum, "needs-design-approval")
			mockClient.AddLabelToIssue("test-repo", issueNum, "design-approved")

			if err := store.MarkTaskCompleted(ctx, task.ID, result); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("concurrent operation failed: %v", err)
		}
	}

	// Verify all tasks completed
	for i := 0; i < numTasks; i++ {
		retrieved, _ := store.GetTask(ctx, fmt.Sprintf("concurrent-task-%d", i))
		if retrieved.Status != TaskStatusCompleted {
			t.Errorf("task %d expected completed, got %s", i, retrieved.Status)
		}
	}

	t.Logf("Successfully handled %d concurrent approvals", numTasks)
}

// TestGitHubWorkflow_StateTransitionOrder tests that label transitions occur in correct order
func TestGitHubWorkflow_StateTransitionOrder(t *testing.T) {
	mockClient := NewMockGitHubClient()
	issueNum := 2100

	// Define workflow stages
	stages := []struct {
		name             string
		labelsToAdd      []string
		labelsToRemove   []string
		expectedLabels   []string
	}{
		{
			name:           "stage1_needs_design",
			labelsToAdd:    []string{"needs-design-approval"},
			labelsToRemove: []string{},
			expectedLabels: []string{"needs-design-approval"},
		},
		{
			name:           "stage2_design_approved",
			labelsToAdd:    []string{"design-approved", "needs-sprint-approval"},
			labelsToRemove: []string{"needs-design-approval"},
			expectedLabels: []string{"design-approved", "needs-sprint-approval"},
		},
		{
			name:           "stage3_sprint_approved",
			labelsToAdd:    []string{"sprint-approved", "needs-merge-approval"},
			labelsToRemove: []string{"needs-sprint-approval"},
			expectedLabels: []string{"design-approved", "sprint-approved", "needs-merge-approval"},
		},
		{
			name:           "stage4_merge_approved",
			labelsToAdd:    []string{"merge-approved"},
			labelsToRemove: []string{"needs-merge-approval"},
			expectedLabels: []string{"design-approved", "sprint-approved", "merge-approved"},
		},
	}

	// Ensure all labels exist
	mockClient.EnsureLabel("test-repo", "needs-design-approval", "Needs design", "B60205")
	mockClient.EnsureLabel("test-repo", "design-approved", "Design approved", "0E8A16")
	mockClient.EnsureLabel("test-repo", "needs-sprint-approval", "Needs sprint", "B60205")
	mockClient.EnsureLabel("test-repo", "sprint-approved", "Sprint approved", "0E8A16")
	mockClient.EnsureLabel("test-repo", "needs-merge-approval", "Needs merge", "B60205")
	mockClient.EnsureLabel("test-repo", "merge-approved", "Merge approved", "0E8A16")

	for _, stage := range stages {
		// Remove old labels
		for _, label := range stage.labelsToRemove {
			if err := mockClient.RemoveLabelFromIssue("test-repo", issueNum, label); err != nil {
				t.Fatalf("stage %s: failed to remove label %s: %v", stage.name, label, err)
			}
		}

		// Add new labels
		for _, label := range stage.labelsToAdd {
			if err := mockClient.AddLabelToIssue("test-repo", issueNum, label); err != nil {
				t.Fatalf("stage %s: failed to add label %s: %v", stage.name, label, err)
			}
		}

		// Verify expected labels
		currentLabels := mockClient.GetLabels(issueNum)
		if len(currentLabels) != len(stage.expectedLabels) {
			t.Errorf("stage %s: expected %d labels, got %d: %v", stage.name, len(stage.expectedLabels), len(currentLabels), currentLabels)
		}

		for _, expected := range stage.expectedLabels {
			found := false
			for _, current := range currentLabels {
				if current == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("stage %s: expected label %s not found in %v", stage.name, expected, currentLabels)
			}
		}

		t.Logf("✓ Stage %s: labels %v", stage.name, currentLabels)
	}
}

// TestGitHubWorkflow_CommentSequence tests that comments are posted in correct order
func TestGitHubWorkflow_CommentSequence(t *testing.T) {
	mockClient := NewMockGitHubClient()
	issueNum := 2200

	expectedSequence := []string{
		"🤖 Starting work on this issue",
		"📝 Analyzing requirements",
		"✅ Design document created",
		"⏳ Waiting for approval",
		"✅ Approved! Starting implementation",
		"⚙️ Implementing feature",
		"✅ Implementation complete",
		"✅ Issue resolved and merged",
	}

	for _, comment := range expectedSequence {
		if err := mockClient.AddComment("test-repo", issueNum, comment); err != nil {
			t.Fatalf("failed to add comment: %v", err)
		}
	}

	comments := mockClient.GetComments(issueNum)

	if len(comments) != len(expectedSequence) {
		t.Errorf("expected %d comments, got %d", len(expectedSequence), len(comments))
	}

	for i, expected := range expectedSequence {
		if i < len(comments) && comments[i] != expected {
			t.Errorf("comment %d: expected %q, got %q", i, expected, comments[i])
		}
	}

	t.Logf("✓ Comment sequence verified: %d comments in correct order", len(comments))
}

// TestGitHubWorkflow_EdgeCase_NoIssueNumber tests handling tasks without GitHub issue numbers
func TestGitHubWorkflow_EdgeCase_NoIssueNumber(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Task without GitHub issue (local task)
	task := &TaskRecord{
		ID:      "local-task-1",
		Title:   "Local task without GitHub issue",
		Content: "This is a local task",
		Type:    TaskTypeBugFix,
		Status:  TaskStatusPending,
		// Note: GithubIssue is 0 (not set)
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create local task: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, task.ID)
	if retrieved.GithubIssue != 0 {
		t.Errorf("expected no GitHub issue, got %d", retrieved.GithubIssue)
	}

	// Complete the task without GitHub integration
	result := &ExecuteResult{
		Success:    true,
		Output:     "Completed",
		Duration:   1 * time.Second,
		Cost:       0.01,
		TokensUsed: 100,
	}

	if err := store.MarkTaskRunning(ctx, task.ID, "provider", "wt"); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}

	if err := store.MarkTaskCompleted(ctx, task.ID, result); err != nil {
		t.Fatalf("failed to mark completed: %v", err)
	}

	final, _ := store.GetTask(ctx, task.ID)
	if final.Status != TaskStatusCompleted {
		t.Errorf("expected completed, got %s", final.Status)
	}

	t.Logf("✓ Local task completed without GitHub integration")
}
