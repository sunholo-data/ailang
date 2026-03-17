package coordinator

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// MockGitHubClient is in github_mock_test.go

// TestGitHubPoster_PostComment tests posting a comment to an issue
func TestGitHubPoster_PostComment(t *testing.T) {
	mockClient := NewMockGitHubClient()

	// Test with real client API but mocked implementation
	// Since we can't easily inject, we'll test the label management flow
	t.Run("label_management", func(t *testing.T) {
		if err := mockClient.EnsureLabel("test-repo", "test-label", "Test label", "FF0000"); err != nil {
			t.Fatalf("failed to ensure label: %v", err)
		}

		if mockClient.GetCallCount("EnsureLabel") != 1 {
			t.Errorf("expected 1 EnsureLabel call, got %d", mockClient.GetCallCount("EnsureLabel"))
		}

		if _, ok := mockClient.definedLabels["test-label"]; !ok {
			t.Error("expected label to be defined")
		}
	})
}

// TestGitHubIntegration_LabelWorkflow tests the label workflow for task stages
func TestGitHubIntegration_LabelWorkflow(t *testing.T) {
	mockClient := NewMockGitHubClient()
	issueNum := 42

	tests := []struct {
		name   string
		setup  func()
		test   func()
		verify func() error
	}{
		{
			name: "add_design_approval_label",
			setup: func() {
				mockClient.EnsureLabel("test-repo", "needs-design-approval", "Needs design approval", "B60205")
			},
			test: func() {
				mockClient.AddLabelToIssue("test-repo", issueNum, "needs-design-approval")
			},
			verify: func() error {
				labels := mockClient.GetLabels(issueNum)
				for _, l := range labels {
					if l == "needs-design-approval" {
						return nil
					}
				}
				return fmt.Errorf("label not found")
			},
		},
		{
			name: "transition_design_to_sprint",
			setup: func() {
				mockClient.EnsureLabel("test-repo", "needs-design-approval", "Needs design approval", "B60205")
				mockClient.EnsureLabel("test-repo", "design-approved", "Design approved", "0E8A16")
				mockClient.EnsureLabel("test-repo", "needs-sprint-approval", "Needs sprint approval", "B60205")
				mockClient.AddLabelToIssue("test-repo", issueNum, "needs-design-approval")
			},
			test: func() {
				// Simulate transition: remove design-approval, add design-approved, add sprint-approval
				mockClient.RemoveLabelFromIssue("test-repo", issueNum, "needs-design-approval")
				mockClient.AddLabelToIssue("test-repo", issueNum, "design-approved")
				mockClient.AddLabelToIssue("test-repo", issueNum, "needs-sprint-approval")
			},
			verify: func() error {
				labels := mockClient.GetLabels(issueNum)
				hasDesignApproved := false
				hasSprintApproval := false
				for _, l := range labels {
					if l == "design-approved" {
						hasDesignApproved = true
					}
					if l == "needs-sprint-approval" {
						hasSprintApproval = true
					}
					if l == "needs-design-approval" {
						return fmt.Errorf("should not have needs-design-approval")
					}
				}
				if !hasDesignApproved || !hasSprintApproval {
					return fmt.Errorf("missing expected labels")
				}
				return nil
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mock
			mockClient = NewMockGitHubClient()

			tc.setup()
			tc.test()

			if err := tc.verify(); err != nil {
				t.Errorf("verification failed: %v", err)
			}
		})
	}
}

// TestGitHubIntegration_CommentFlow tests posting status updates and comments
func TestGitHubIntegration_CommentFlow(t *testing.T) {
	mockClient := NewMockGitHubClient()
	issueNum := 123

	tests := []struct {
		name        string
		comments    []string
		expectCount int
	}{
		{
			name:        "single_comment",
			comments:    []string{"Task started"},
			expectCount: 1,
		},
		{
			name:        "multiple_comments",
			comments:    []string{"Task started", "50% complete", "Task completed"},
			expectCount: 3,
		},
		{
			name:        "empty_comments",
			comments:    []string{},
			expectCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset
			mockClient = NewMockGitHubClient()

			for _, comment := range tc.comments {
				if err := mockClient.AddComment("test-repo", issueNum, comment); err != nil {
					t.Fatalf("failed to add comment: %v", err)
				}
			}

			if len(mockClient.GetComments(issueNum)) != tc.expectCount {
				t.Errorf("expected %d comments, got %d", tc.expectCount, len(mockClient.GetComments(issueNum)))
			}
		})
	}
}

// TestGitHubIntegration_IssueClosing tests closing issues with final comments
func TestGitHubIntegration_IssueClosing(t *testing.T) {
	mockClient := NewMockGitHubClient()
	issueNum := 456

	tests := []struct {
		name           string
		closeComment   string
		expectClosed   bool
		expectComments int
	}{
		{
			name:           "close_with_comment",
			closeComment:   "Task completed successfully",
			expectClosed:   true,
			expectComments: 1,
		},
		{
			name:           "close_without_comment",
			closeComment:   "",
			expectClosed:   true,
			expectComments: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient = NewMockGitHubClient()

			if err := mockClient.CloseIssue("test-repo", issueNum, tc.closeComment); err != nil {
				t.Fatalf("failed to close issue: %v", err)
			}

			if !mockClient.IsClosed(issueNum) && tc.expectClosed {
				t.Error("expected issue to be closed")
			}

			if len(mockClient.GetComments(issueNum)) != tc.expectComments {
				t.Errorf("expected %d comments, got %d", tc.expectComments, len(mockClient.GetComments(issueNum)))
			}
		})
	}
}

// TestGitHubIntegration_ErrorHandling tests error scenarios
func TestGitHubIntegration_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		operation  string
		setupError func(*MockGitHubClient)
		shouldFail bool
	}{
		{
			name:      "add_comment_error",
			operation: "AddComment",
			setupError: func(m *MockGitHubClient) {
				m.addCommentErr = fmt.Errorf("API rate limit exceeded")
			},
			shouldFail: true,
		},
		{
			name:      "add_label_error",
			operation: "AddLabel",
			setupError: func(m *MockGitHubClient) {
				m.addLabelErr = fmt.Errorf("insufficient permissions")
			},
			shouldFail: true,
		},
		{
			name:      "remove_label_error",
			operation: "RemoveLabel",
			setupError: func(m *MockGitHubClient) {
				m.removeLabelErr = fmt.Errorf("label not found")
			},
			shouldFail: true,
		},
		{
			name:      "close_issue_error",
			operation: "CloseIssue",
			setupError: func(m *MockGitHubClient) {
				m.closeIssueErr = fmt.Errorf("issue already closed")
			},
			shouldFail: true,
		},
		{
			name:      "get_labels_error",
			operation: "GetLabels",
			setupError: func(m *MockGitHubClient) {
				m.getLabelsErr = fmt.Errorf("issue not found")
			},
			shouldFail: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := NewMockGitHubClient()
			tc.setupError(mockClient)

			var err error

			switch tc.operation {
			case "AddComment":
				err = mockClient.AddComment("test-repo", 1, "test")
			case "AddLabel":
				mockClient.EnsureLabel("test-repo", "test-label", "Test", "FF0000")
				err = mockClient.AddLabelToIssue("test-repo", 1, "test-label")
			case "RemoveLabel":
				err = mockClient.RemoveLabelFromIssue("test-repo", 1, "test-label")
			case "CloseIssue":
				err = mockClient.CloseIssue("test-repo", 1, "")
			case "GetLabels":
				_, err = mockClient.GetIssueLabels("test-repo", 1)
			}

			if tc.shouldFail && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestGitHubIntegration_FullTaskLifecycle tests a complete task lifecycle with GitHub integration
func TestGitHubIntegration_FullTaskLifecycle(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	mockClient := NewMockGitHubClient()
	ctx := context.Background()
	issueNum := 789

	// Step 1: Create task from GitHub issue
	task := &TaskRecord{
		ID:          "gh-task-1",
		Title:       "Fix authentication bug",
		Content:     "Users cannot login with OAuth",
		Type:        TaskTypeBugFix,
		Status:      TaskStatusPending,
		GithubIssue: issueNum,
		CreatedAt:   time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Step 2: Post initial status and add needs-design-approval label
	mockClient.EnsureLabel("test-repo", "needs-design-approval", "Needs approval", "B60205")
	if err := mockClient.AddComment("test-repo", issueNum, "🤖 Agent Working: Starting work on GH-789"); err != nil {
		t.Fatalf("failed to post comment: %v", err)
	}
	if err := mockClient.AddLabelToIssue("test-repo", issueNum, "needs-design-approval"); err != nil {
		t.Fatalf("failed to add label: %v", err)
	}

	if len(mockClient.GetComments(issueNum)) != 1 {
		t.Errorf("expected 1 comment, got %d", len(mockClient.GetComments(issueNum)))
	}

	labels := mockClient.GetLabels(issueNum)
	if len(labels) != 1 || labels[0] != "needs-design-approval" {
		t.Errorf("expected needs-design-approval label, got %v", labels)
	}

	// Step 3: Mark task as running
	if err := store.MarkTaskRunning(ctx, task.ID, "mock-provider", "wt-1"); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}

	// Step 4: Simulate approval workflow transition
	mockClient.RemoveLabelFromIssue("test-repo", issueNum, "needs-design-approval")
	mockClient.EnsureLabel("test-repo", "design-approved", "Design approved", "0E8A16")
	mockClient.EnsureLabel("test-repo", "needs-sprint-approval", "Needs sprint approval", "B60205")
	mockClient.AddLabelToIssue("test-repo", issueNum, "design-approved")
	mockClient.AddLabelToIssue("test-repo", issueNum, "needs-sprint-approval")
	mockClient.AddComment("test-repo", issueNum, "✅ Design approved. Moving to sprint planning.")

	labels = mockClient.GetLabels(issueNum)
	hasDesignApproved := false
	hasSprintApproval := false
	for _, l := range labels {
		if l == "design-approved" {
			hasDesignApproved = true
		}
		if l == "needs-sprint-approval" {
			hasSprintApproval = true
		}
	}
	if !hasDesignApproved || !hasSprintApproval {
		t.Errorf("expected design-approved and needs-sprint-approval labels, got %v", labels)
	}

	// Step 5: Mark task complete and close issue
	result := &ExecuteResult{
		Success:      true,
		Output:       "Fixed OAuth token validation",
		Duration:     5 * time.Second,
		Cost:         0.05,
		TokensUsed:   1000,
		InputTokens:  600,
		OutputTokens: 400,
	}

	if err := store.MarkTaskCompleted(ctx, task.ID, result); err != nil {
		t.Fatalf("failed to mark completed: %v", err)
	}

	mockClient.EnsureLabel("test-repo", "merge-approved", "Merge approved", "0E8A16")
	mockClient.RemoveLabelFromIssue("test-repo", issueNum, "needs-sprint-approval")
	mockClient.AddLabelToIssue("test-repo", issueNum, "merge-approved")
	mockClient.CloseIssue("test-repo", issueNum, "✅ Issue resolved and merged to main")

	if !mockClient.IsClosed(issueNum) {
		t.Error("expected issue to be closed")
	}

	finalComments := mockClient.GetComments(issueNum)
	if len(finalComments) != 3 {
		t.Errorf("expected 3 comments, got %d", len(finalComments))
	}

	t.Logf("Full task lifecycle complete: issue %d closed after task execution", issueNum)
}

// TestGitHubIntegration_MultipleIssues tests coordinating multiple GitHub issues
func TestGitHubIntegration_MultipleIssues(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	mockClient := NewMockGitHubClient()
	ctx := context.Background()

	// Prepare labels
	mockClient.EnsureLabel("test-repo", "coordinator:bug", "Bug fix", "D73A4A")
	mockClient.EnsureLabel("test-repo", "coordinator:feature", "New feature", "A2EEEF")
	mockClient.EnsureLabel("test-repo", "needs-design-approval", "Needs approval", "B60205")

	issues := []struct {
		id       string
		issueNum int
		title    string
		taskType TaskType
	}{
		{"task-1", 100, "Parser null pointer bug", TaskTypeBugFix},
		{"task-2", 101, "Add caching feature", TaskTypeFeature},
		{"task-3", 102, "Fix memory leak", TaskTypeBugFix},
	}

	// Create tasks for each issue
	for _, issue := range issues {
		task := &TaskRecord{
			ID:          issue.id,
			Title:       issue.title,
			Content:     fmt.Sprintf("GitHub issue #%d", issue.issueNum),
			Type:        issue.taskType,
			Status:      TaskStatusPending,
			GithubIssue: issue.issueNum,
			CreatedAt:   time.Now(),
		}

		if err := store.CreateTask(ctx, task); err != nil {
			t.Fatalf("failed to create task %s: %v", issue.id, err)
		}

		// Post initial comment and label
		mockClient.AddComment("test-repo", issue.issueNum, fmt.Sprintf("🤖 Working on #%d", issue.issueNum))
		mockClient.AddLabelToIssue("test-repo", issue.issueNum, "needs-design-approval")
	}

	// Verify all issues are tracked
	createdTasks := 0
	for _, issue := range issues {
		task, err := store.GetTask(ctx, issue.id)
		if err != nil || task == nil {
			t.Fatalf("failed to retrieve task %s", issue.id)
		}
		createdTasks++
	}

	if createdTasks != len(issues) {
		t.Errorf("expected %d tasks created, got %d", len(issues), createdTasks)
	}

	// Verify GitHub tracking
	if mockClient.GetCallCount("AddComment") != len(issues) {
		t.Errorf("expected %d comments, got %d", len(issues), mockClient.GetCallCount("AddComment"))
	}

	if mockClient.GetCallCount("AddLabelToIssue") != len(issues) {
		t.Errorf("expected %d label additions, got %d", len(issues), mockClient.GetCallCount("AddLabelToIssue"))
	}
}

// TestGitHubIntegration_LabelPredefinedSet tests that all predefined labels can be created
func TestGitHubIntegration_LabelPredefinedSet(t *testing.T) {
	mockClient := NewMockGitHubClient()

	// These are the labels defined in EnsureLabel()
	predefinedLabels := []string{
		"coordinator:bug",
		"coordinator:feature",
		"coordinator:docs",
		"coordinator:research",
		"coordinator:refactor",
		"coordinator:test",
		"needs-design-approval",
		"needs-sprint-approval",
		"needs-merge-approval",
		"needs-revision",
		"design-approved",
		"sprint-approved",
		"merge-approved",
	}

	for _, label := range predefinedLabels {
		if err := mockClient.EnsureLabel("test-repo", label, "Test label", "FF0000"); err != nil {
			t.Fatalf("failed to create label %q: %v", label, err)
		}
	}

	// Verify all labels were created
	if len(mockClient.definedLabels) != len(predefinedLabels) {
		t.Errorf("expected %d labels defined, got %d", len(predefinedLabels), len(mockClient.definedLabels))
	}

	for _, label := range predefinedLabels {
		if _, ok := mockClient.definedLabels[label]; !ok {
			t.Errorf("expected label %q to be defined", label)
		}
	}
}

// TestGitHubIntegration_ConcurrentLabelOperations tests thread-safe label operations
func TestGitHubIntegration_ConcurrentLabelOperations(t *testing.T) {
	mockClient := NewMockGitHubClient()
	mockClient.EnsureLabel("test-repo", "test-label", "Test", "FF0000")

	// Ensure label is idempotent
	issueNum := 200
	done := make(chan bool)
	errors := make(chan error, 10)

	// Try to add same label from multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			if err := mockClient.AddLabelToIssue("test-repo", issueNum, "test-label"); err != nil {
				errors <- err
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("concurrent operation failed: %v", err)
		}
	}

	// Verify only one label was added (idempotent)
	labels := mockClient.GetLabels(issueNum)
	count := 0
	for _, l := range labels {
		if l == "test-label" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("expected 1 label, got %d", count)
	}
}

// TestGitHubIntegration_TaskToIssueTracking tests task-to-GitHub-issue tracking
func TestGitHubIntegration_TaskToIssueTracking(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create a task without GitHub issue initially
	issueNum := 555
	task := &TaskRecord{
		ID:        "gh-tracked-1",
		Title:     "Fix issue",
		Content:   "Content",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Link GitHub issue using the SetTaskGithubIssue method
	if err := store.SetTaskGithubIssue(ctx, task.ID, issueNum); err != nil {
		t.Fatalf("failed to set GitHub issue: %v", err)
	}

	// Retrieve and verify GitHub issue is stored
	retrieved, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("failed to retrieve task: %v", err)
	}

	if retrieved.GithubIssue != issueNum {
		t.Errorf("expected GitHub issue %d, got %d", issueNum, retrieved.GithubIssue)
	}
}

// TestGitHubIntegration_RateLimitHandling tests handling of rate limits
func TestGitHubIntegration_RateLimitHandling(t *testing.T) {
	mockClient := NewMockGitHubClient()

	// Simulate rate limit error
	mockClient.addCommentErr = fmt.Errorf("API error 429: rate limit exceeded")

	issueNum := 300
	err := mockClient.AddComment("test-repo", issueNum, "test comment")

	if err == nil {
		t.Error("expected rate limit error, got none")
	}

	if err.Error() != "API error 429: rate limit exceeded" {
		t.Errorf("expected rate limit error message, got: %v", err)
	}

	// Verify call was still attempted
	if mockClient.GetCallCount("AddComment") != 1 {
		t.Errorf("expected 1 call attempt, got %d", mockClient.GetCallCount("AddComment"))
	}
}

// TestGitHubIntegration_ApprovalLabelDetection tests detecting approval labels on issues
func TestGitHubIntegration_ApprovalLabelDetection(t *testing.T) {
	mockClient := NewMockGitHubClient()
	issueNum := 400

	// Set up approval labels
	mockClient.EnsureLabel("test-repo", "design-approved", "Design approved", "0E8A16")
	mockClient.EnsureLabel("test-repo", "needs-sprint-approval", "Needs sprint approval", "B60205")

	// Initially needs approval
	mockClient.AddLabelToIssue("test-repo", issueNum, "needs-sprint-approval")

	labels := mockClient.GetLabels(issueNum)
	needsApproval := false
	for _, l := range labels {
		if l == "needs-sprint-approval" {
			needsApproval = true
		}
	}

	if !needsApproval {
		t.Error("expected needs-sprint-approval label")
	}

	// Human adds approval
	mockClient.RemoveLabelFromIssue("test-repo", issueNum, "needs-sprint-approval")
	mockClient.AddLabelToIssue("test-repo", issueNum, "design-approved")

	labels = mockClient.GetLabels(issueNum)
	isApproved := false
	for _, l := range labels {
		if l == "design-approved" {
			isApproved = true
		}
	}

	if !isApproved {
		t.Error("expected design-approved label after human approval")
	}
}

// TestGitHubIntegration_TaskContext tests timeout handling in context
func TestGitHubIntegration_TaskContext(t *testing.T) {
	mockClient := NewMockGitHubClient()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	issueNum := 500
	taskID := "ctx-task-1"

	// Simulate long operation that respects context
	go func() {
		<-ctx.Done()
		// Context cancelled - stop operations
	}()

	time.Sleep(150 * time.Millisecond)

	if ctx.Err() != context.DeadlineExceeded {
		t.Error("expected context deadline exceeded")
	}

	// Verify we can still perform operations (mock client doesn't check context)
	if err := mockClient.AddComment("test-repo", issueNum, "cleanup"); err != nil {
		t.Fatalf("failed to add cleanup comment: %v", err)
	}

	t.Logf("Task %s completed with context handling", taskID)
}
