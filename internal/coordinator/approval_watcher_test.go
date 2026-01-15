package coordinator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewApprovalWatcher tests initialization with default values
func TestNewApprovalWatcher(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}

	watcher := NewApprovalWatcher(poster, store, 0)

	if watcher == nil {
		t.Error("NewApprovalWatcher returned nil")
	}

	// Verify default poll interval is set when 0 is passed
	if watcher.pollInterval != 60*time.Second {
		t.Errorf("Default poll interval = %v, want %v", watcher.pollInterval, 60*time.Second)
	}

	if watcher.running {
		t.Error("Watcher should not be running on creation")
	}

	// Verify maps are initialized
	if watcher.handlers == nil {
		t.Error("handlers map not initialized")
	}
	if watcher.watchedIssues == nil {
		t.Error("watchedIssues map not initialized")
	}
	if watcher.agentByLabel == nil {
		t.Error("agentByLabel map not initialized")
	}
}

// TestRegisterHandlerNilHandler tests registering a nil handler
func TestRegisterHandlerNilHandler(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	// Should not panic when registering nil handler
	err := watcher.RegisterHandler(ApprovalEventDesign, nil)
	if err != nil {
		t.Errorf("RegisterHandler failed: %v", err)
	}

	// Verify nil handler is stored
	watcher.mu.Lock()
	handler := watcher.handlers[ApprovalEventDesign]
	watcher.mu.Unlock()

	if handler != nil {
		t.Error("Expected nil handler to be stored")
	}
}

// TestRegisterHandlerEmptyEventType tests RegisterHandler with empty event type
func TestRegisterHandlerEmptyEventType(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	err := watcher.RegisterHandler("", func(ctx context.Context, event *ApprovalEvent) error {
		return nil
	})

	if err == nil {
		t.Error("Expected error for empty event type, got nil")
	}
	if err.Error() != "event type cannot be empty" {
		t.Errorf("Error message = %q, want 'event type cannot be empty'", err.Error())
	}
}

// TestRegisterAgentApprovalNilAgent tests RegisterAgentApproval with nil agent
func TestRegisterAgentApprovalNilAgent(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	err := watcher.RegisterAgentApproval(nil, nil)
	if err == nil {
		t.Error("Expected error for nil agent, got nil")
	}
	if err.Error() != "agent config is nil" {
		t.Errorf("Error message = %q, want 'agent config is nil'", err.Error())
	}
}

// TestRegisterAgentApprovalNoApprovalConfig tests agent with no approval config
func TestRegisterAgentApprovalNoApprovalConfig(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	agent := &AgentConfig{
		ID: "test-agent",
		// No Approval field set
	}

	err := watcher.RegisterAgentApproval(agent, nil)
	if err == nil {
		t.Error("Expected error for missing approval config, got nil")
	}
	if err.Error() != "agent test-agent has no approval config" {
		t.Errorf("Error message = %q, want agent approval config error", err.Error())
	}
}

// TestRegisterAgentApprovalEmptyLabel tests agent with empty approved_label
func TestRegisterAgentApprovalEmptyLabel(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	agent := &AgentConfig{
		ID: "test-agent",
		Approval: &ApprovalConfig{
			ApprovedLabel: "", // Empty!
			NeedsLabel:    "needs-test",
		},
	}

	err := watcher.RegisterAgentApproval(agent, nil)
	if err == nil {
		t.Error("Expected error for empty approved_label, got nil")
	}
	if err.Error() != "agent test-agent has empty approved_label" {
		t.Errorf("Error message = %q, want empty label error", err.Error())
	}
}

// TestRegisterAgentApprovalSuccess tests successful agent registration
func TestRegisterAgentApprovalSuccess(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	agent := &AgentConfig{
		ID: "test-agent",
		Approval: &ApprovalConfig{
			ApprovedLabel: "ready-to-deploy",
			NeedsLabel:    "needs-deployment",
		},
	}

	err := watcher.RegisterAgentApproval(agent, nil)
	if err != nil {
		t.Errorf("RegisterAgentApproval failed: %v", err)
	}

	// Verify agent is registered
	if registered := watcher.GetAgentByLabel("ready-to-deploy"); registered == nil {
		t.Error("Agent not registered")
	} else if registered.ID != "test-agent" {
		t.Errorf("Registered agent ID = %q, want 'test-agent'", registered.ID)
	}
}

// TestRegisterAgentApprovalWithHandler tests registration with custom handler
func TestRegisterAgentApprovalWithHandler(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	agent := &AgentConfig{
		ID: "test-agent",
		Approval: &ApprovalConfig{
			ApprovedLabel: "test-label",
			NeedsLabel:    "needs-test",
		},
	}

	handlerCalled := false
	handler := func(ctx context.Context, event *ApprovalEvent) error {
		handlerCalled = true
		return nil
	}

	err := watcher.RegisterAgentApproval(agent, handler)
	if err != nil {
		t.Errorf("RegisterAgentApproval failed: %v", err)
	}

	// Verify custom handler is registered by checking internal state
	watcher.mu.Lock()
	customHandler, exists := watcher.customHandlers["test-label"]
	watcher.mu.Unlock()

	if !exists {
		t.Error("Custom handler not registered")
	}

	if customHandler != nil {
		ctx := context.Background()
		event := &ApprovalEvent{TaskID: "test"}
		customHandler(ctx, event)
		if !handlerCalled {
			t.Error("Custom handler was not called")
		}
	}
}

// TestGetRegisteredLabels returns empty slice when no labels registered
func TestGetRegisteredLabelsEmpty(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	labels := watcher.GetRegisteredLabels()
	if len(labels) != 0 {
		t.Errorf("GetRegisteredLabels() = %v, want empty", labels)
	}
}

// TestGetRegisteredLabels returns registered labels
func TestGetRegisteredLabels(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	agent1 := &AgentConfig{
		ID: "agent1",
		Approval: &ApprovalConfig{
			ApprovedLabel: "label1",
			NeedsLabel:    "needs1",
		},
	}

	agent2 := &AgentConfig{
		ID: "agent2",
		Approval: &ApprovalConfig{
			ApprovedLabel: "label2",
			NeedsLabel:    "needs2",
		},
	}

	watcher.RegisterAgentApproval(agent1, nil)
	watcher.RegisterAgentApproval(agent2, nil)

	labels := watcher.GetRegisteredLabels()
	if len(labels) != 2 {
		t.Errorf("GetRegisteredLabels() returned %d labels, want 2", len(labels))
	}

	labelSet := make(map[string]bool)
	for _, l := range labels {
		labelSet[l] = true
	}

	if !labelSet["label1"] || !labelSet["label2"] {
		t.Errorf("Labels missing. Got: %v", labels)
	}
}

// TestWatchIssueAndUnwatch tests watching and unwatching issues
func TestWatchIssueAndUnwatch(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	// Watch an issue
	err := watcher.WatchIssue(123, "task-id-1")
	if err != nil {
		t.Fatalf("WatchIssue failed: %v", err)
	}

	if watcher.WatchedIssueCount() != 1 {
		t.Errorf("WatchedIssueCount() = %d, want 1", watcher.WatchedIssueCount())
	}

	// Unwatch
	watcher.UnwatchIssue(123)

	if watcher.WatchedIssueCount() != 0 {
		t.Errorf("WatchedIssueCount() = %d after unwatch, want 0", watcher.WatchedIssueCount())
	}
}

// TestWatchIssueNegativeIssueNumber tests WatchIssue with invalid issue number
func TestWatchIssueNegativeIssueNumber(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	err := watcher.WatchIssue(-1, "task-id-1")
	if err == nil {
		t.Error("Expected error for negative issue number, got nil")
	}
	if !strings.Contains(err.Error(), "must be positive") {
		t.Errorf("Error message = %q, want 'must be positive'", err.Error())
	}
}

// TestWatchIssueZeroIssueNumber tests WatchIssue with zero issue number
func TestWatchIssueZeroIssueNumber(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	err := watcher.WatchIssue(0, "task-id-1")
	if err == nil {
		t.Error("Expected error for zero issue number, got nil")
	}
}

// TestWatchIssueEmptyTaskID tests WatchIssue with empty task ID
func TestWatchIssueEmptyTaskID(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	err := watcher.WatchIssue(123, "")
	if err == nil {
		t.Error("Expected error for empty task ID, got nil")
	}
	if err.Error() != "task ID cannot be empty" {
		t.Errorf("Error message = %q, want 'task ID cannot be empty'", err.Error())
	}
}

// TestStartAlreadyRunning tests starting watcher when already running
func TestStartAlreadyRunning(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := watcher.Start(ctx)
	if err != nil {
		t.Fatalf("First Start failed: %v", err)
	}
	defer watcher.Stop()

	// Try to start again
	err = watcher.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already-running watcher")
	}
	if err.Error() != "approval watcher already running" {
		t.Errorf("Error message = %q, want 'approval watcher already running'", err.Error())
	}
}

// TestStopNotRunning tests stopping watcher that isn't running
func TestStopNotRunning(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	// Should not panic when stopping not-running watcher
	watcher.Stop()

	if watcher.IsRunning() {
		t.Error("Watcher should not be running")
	}
}

// TestStartAndStop tests starting and stopping the watcher
func TestStartAndStop(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	err := watcher.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !watcher.IsRunning() {
		t.Error("Watcher should be running after Start()")
	}

	// Stop immediately
	watcher.Stop()

	if watcher.IsRunning() {
		t.Error("Watcher should not be running after Stop()")
	}

	cancel()
}

// TestGetStatus tests status reporting
func TestGetStatus(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 30*time.Second)

	watcher.WatchIssue(123, "task-1")
	watcher.WatchIssue(456, "task-2")

	status := watcher.GetStatus()

	if status.Running {
		t.Error("Status.Running should be false before Start()")
	}

	if status.PollInterval != 30*time.Second {
		t.Errorf("Status.PollInterval = %v, want 30s", status.PollInterval)
	}

	if len(status.WatchedIssues) != 2 {
		t.Errorf("Status.WatchedIssues count = %d, want 2", len(status.WatchedIssues))
	}
}

// TestLoadWatchedIssuesFromStoreError tests error handling when loading from store
func TestLoadWatchedIssuesFromStoreError(t *testing.T) {
	storeWithError := &mockStore{
		err: errors.New("database error"),
	}

	poster := &GitHubPoster{}
	watcher := NewApprovalWatcher(poster, storeWithError, 10*time.Second)

	err := watcher.LoadWatchedIssuesFromStore(context.Background())
	if err == nil {
		t.Error("Expected error when store fails, got nil")
	}
}

// TestLoadWatchedIssuesFromStoreEmptyTasks tests loading when no tasks exist
func TestLoadWatchedIssuesFromStoreEmptyTasks(t *testing.T) {
	storeEmpty := &mockStore{
		tasks: []*TaskRecord{}, // Empty
	}

	poster := &GitHubPoster{}
	watcher := NewApprovalWatcher(poster, storeEmpty, 10*time.Second)

	err := watcher.LoadWatchedIssuesFromStore(context.Background())
	if err != nil {
		t.Errorf("LoadWatchedIssuesFromStore failed: %v", err)
	}

	if watcher.WatchedIssueCount() != 0 {
		t.Errorf("WatchedIssueCount = %d, want 0", watcher.WatchedIssueCount())
	}
}

// TestConcurrentWatchIssue tests concurrent issue watching for race conditions
func TestConcurrentWatchIssue(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	var wg sync.WaitGroup
	for i := 1; i <= 10; i++ { // Start from 1 to have positive issue numbers
		wg.Add(1)
		go func(issueNum int) {
			defer wg.Done()
			_ = watcher.WatchIssue(issueNum, fmt.Sprintf("task-%d", issueNum))
		}(i)
	}

	wg.Wait()

	if watcher.WatchedIssueCount() != 10 {
		t.Errorf("WatchedIssueCount = %d, want 10", watcher.WatchedIssueCount())
	}
}

// TestConcurrentRegisterAgent tests concurrent agent registration for race conditions
func TestConcurrentRegisterAgent(t *testing.T) {
	poster := &GitHubPoster{}
	store := &mockStore{}
	watcher := NewApprovalWatcher(poster, store, 10*time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			agent := &AgentConfig{
				ID: fmt.Sprintf("agent-%d", idx),
				Approval: &ApprovalConfig{
					ApprovedLabel: fmt.Sprintf("label-%d", idx),
					NeedsLabel:    fmt.Sprintf("needs-%d", idx),
				},
			}
			watcher.RegisterAgentApproval(agent, nil)
		}(i)
	}

	wg.Wait()

	if len(watcher.GetRegisteredLabels()) != 5 {
		t.Errorf("Registered labels = %d, want 5", len(watcher.GetRegisteredLabels()))
	}
}

// Mock implementations for testing

type mockGitHubPoster struct {
	getLabels              func(issueNum int) ([]string, error)
	removeLabel            func(issueNum int, label string) error
	getRecentHumanComments func(issueNum int, since time.Time) ([]IssueComment, error)
}

func (m *mockGitHubPoster) GetLabels(issueNum int) ([]string, error) {
	if m.getLabels != nil {
		return m.getLabels(issueNum)
	}
	return []string{}, nil
}

func (m *mockGitHubPoster) RemoveLabel(issueNum int, label string) error {
	if m.removeLabel != nil {
		return m.removeLabel(issueNum, label)
	}
	return nil
}

func (m *mockGitHubPoster) GetRecentHumanComments(issueNum int, since time.Time) ([]IssueComment, error) {
	if m.getRecentHumanComments != nil {
		return m.getRecentHumanComments(issueNum, since)
	}
	return []IssueComment{}, nil
}

type mockStore struct {
	tasks []*TaskRecord
	err   error
}

func (m *mockStore) GetTasksByStage(ctx context.Context, stage TaskStage) ([]*TaskRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tasks, nil
}

func (m *mockStore) Close() error {
	return nil
}

// Implement other required Store interface methods as no-ops
func (m *mockStore) CreateTask(ctx context.Context, task *TaskRecord) error      { return nil }
func (m *mockStore) GetTask(ctx context.Context, id string) (*TaskRecord, error) { return nil, nil }
func (m *mockStore) UpdateTask(ctx context.Context, task *TaskRecord) error      { return nil }
func (m *mockStore) DeleteTask(ctx context.Context, id string) error             { return nil }
func (m *mockStore) ListTasks(ctx context.Context, filter *TaskFilter) ([]*TaskRecord, error) {
	return nil, nil
}
func (m *mockStore) GetTaskStats(ctx context.Context) (*TaskStats, error) { return nil, nil }
func (m *mockStore) MarkTaskQueued(ctx context.Context, id string) error  { return nil }
func (m *mockStore) MarkTaskRunning(ctx context.Context, id, provider, worktreeID string) error {
	return nil
}
func (m *mockStore) MarkTaskPendingApproval(ctx context.Context, id, worktreePath, worktreeBranch string, result *ExecuteResult) error {
	return nil
}
func (m *mockStore) MarkTaskCompleted(ctx context.Context, id string, result *ExecuteResult) error {
	return nil
}
func (m *mockStore) MarkTaskFailed(ctx context.Context, id string, err error) error { return nil }
func (m *mockStore) MarkTaskRejected(ctx context.Context, id string) error          { return nil }
func (m *mockStore) MarkTaskCancelled(ctx context.Context, id string) error         { return nil }
func (m *mockStore) RequeueTask(ctx context.Context, id string) error               { return nil }
func (m *mockStore) FindDuplicateTask(ctx context.Context, fingerprint uint64, threshold float64) (*TaskRecord, error) {
	return nil, nil
}
func (m *mockStore) SetTaskFingerprint(ctx context.Context, id string, fingerprint uint64) error {
	return nil
}
func (m *mockStore) SetTaskThreadID(ctx context.Context, id string, threadID string) error {
	return nil
}
func (m *mockStore) GetTaskAgentInfo(ctx context.Context, taskID string) (agentID, inbox, title string, err error) {
	return "", "", "", nil
}
func (m *mockStore) SetTaskGithubIssue(ctx context.Context, id string, issueNum int) error {
	return nil
}
func (m *mockStore) SetTaskStage(ctx context.Context, id string, stage TaskStage) error {
	return nil
}
func (m *mockStore) SetTaskDesignDocPath(ctx context.Context, id string, path string) error {
	return nil
}
func (m *mockStore) SetTaskSprintPlanPath(ctx context.Context, id string, path string) error {
	return nil
}
func (m *mockStore) GetTasksByGithubIssue(ctx context.Context, issueNum int) ([]*TaskRecord, error) {
	return nil, nil
}
func (m *mockStore) UpdateTaskMetrics(ctx context.Context, id string, peakCPU, peakMemory float64) error {
	return nil
}
func (m *mockStore) CreateApprovalRequest(ctx context.Context, req *ApprovalRequestRecord) error {
	return nil
}
func (m *mockStore) GetApprovalRequestByTask(ctx context.Context, taskID string) (*ApprovalRequestRecord, error) {
	return nil, nil
}
func (m *mockStore) ListPendingApprovals(ctx context.Context) ([]*ApprovalRequestRecord, error) {
	return nil, nil
}
func (m *mockStore) ResolveApprovalRequest(ctx context.Context, id, status, resolvedBy string) error {
	return nil
}
func (m *mockStore) ResolveApprovalRequestByTask(ctx context.Context, taskID, status, resolvedBy string) error {
	return nil
}
func (m *mockStore) DeleteOldTasks(ctx context.Context, olderThan time.Duration) (int, error) {
	return 0, nil
}
func (m *mockStore) RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration) (int, error) {
	return 0, nil
}
func (m *mockStore) RetryAllFailedTasks(ctx context.Context) (int, error) {
	return 0, nil
}
func (m *mockStore) StoreTaskEvent(ctx context.Context, event *TaskEventRecord) error {
	return nil
}
func (m *mockStore) GetTaskEvents(ctx context.Context, taskID string, limit int) ([]*TaskEventRecord, error) {
	return nil, nil
}
func (m *mockStore) ResetTaskToPending(ctx context.Context, id string) error {
	return nil
}
