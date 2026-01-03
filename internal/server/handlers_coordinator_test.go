package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
)

// MockTaskEventStore implements CoordinatorTaskEventStore for testing
type MockTaskEventStore struct {
	tasks  map[string]*coordinator.TaskRecord
	events map[string][]*coordinator.TaskEventRecord
}

func NewMockTaskEventStore() *MockTaskEventStore {
	return &MockTaskEventStore{
		tasks:  make(map[string]*coordinator.TaskRecord),
		events: make(map[string][]*coordinator.TaskEventRecord),
	}
}

func (m *MockTaskEventStore) GetTaskEvents(ctx context.Context, taskID string, limit int) ([]*coordinator.TaskEventRecord, error) {
	return m.events[taskID], nil
}

func (m *MockTaskEventStore) ListTasks(ctx context.Context, filter *coordinator.TaskFilter) ([]*coordinator.TaskRecord, error) {
	var result []*coordinator.TaskRecord
	for _, t := range m.tasks {
		result = append(result, t)
	}
	return result, nil
}

func (m *MockTaskEventStore) GetTask(ctx context.Context, id string) (*coordinator.TaskRecord, error) {
	return m.tasks[id], nil
}

// MockApprovalStore implements CoordinatorApprovalStore for testing
type MockApprovalStore struct {
	approvals map[string]*coordinator.ApprovalRequestRecord
}

func NewMockApprovalStore() *MockApprovalStore {
	return &MockApprovalStore{
		approvals: make(map[string]*coordinator.ApprovalRequestRecord),
	}
}

func (m *MockApprovalStore) GetApprovalRequest(ctx context.Context, id string) (*coordinator.ApprovalRequestRecord, error) {
	return m.approvals[id], nil
}

func (m *MockApprovalStore) ListPendingApprovals(ctx context.Context) ([]*coordinator.ApprovalRequestRecord, error) {
	var result []*coordinator.ApprovalRequestRecord
	for _, a := range m.approvals {
		if a.Status == "pending" {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *MockApprovalStore) ResolveApprovalRequest(ctx context.Context, id string, status string, resolvedBy string) error {
	if a, ok := m.approvals[id]; ok {
		a.Status = status
		a.ResolvedBy = resolvedBy
		now := time.Now()
		a.ResolvedAt = &now
	}
	return nil
}

func TestPendingApprovalsWithWorktreePath(t *testing.T) {
	// Create mock stores
	taskStore := NewMockTaskEventStore()
	approvalStore := NewMockApprovalStore()

	// Add a task with worktree path
	taskStore.tasks["task-123"] = &coordinator.TaskRecord{
		ID:           "task-123",
		Title:        "Fix bug",
		Content:      "Fix the parser bug",
		Type:         coordinator.TaskTypeBugFix,
		Status:       coordinator.TaskStatusPendingApproval,
		WorktreePath: "/tmp/worktrees/task-123",
		SessionID:    "session-abc",
		Provider:     "claude-code",
		CreatedAt:    time.Now(),
	}

	// Add a pending approval for that task
	approvalStore.approvals["approval-1"] = &coordinator.ApprovalRequestRecord{
		ID:          "approval-1",
		TaskID:      "task-123",
		Type:        "task_completion",
		Description: "Review changes",
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	// Create server with mock stores
	s := &Server{}
	s.SetTaskEventStore(taskStore)
	s.SetApprovalStore(approvalStore)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/coordinator/pending", nil)
	w := httptest.NewRecorder()

	// Call handler
	s.handleCoordinatorPendingApprovals(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Parse response
	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 approval, got %d", len(result))
	}

	// Verify enriched fields
	approval := result[0]
	if approval["worktree_path"] != "/tmp/worktrees/task-123" {
		t.Errorf("expected worktree_path '/tmp/worktrees/task-123', got %v", approval["worktree_path"])
	}
	if approval["session_id"] != "session-abc" {
		t.Errorf("expected session_id 'session-abc', got %v", approval["session_id"])
	}
	if approval["task_title"] != "Fix bug" {
		t.Errorf("expected task_title 'Fix bug', got %v", approval["task_title"])
	}
	if approval["provider"] != "claude-code" {
		t.Errorf("expected provider 'claude-code', got %v", approval["provider"])
	}
}

func TestTaskEventsEndpoint(t *testing.T) {
	taskStore := NewMockTaskEventStore()

	// Add a task
	taskStore.tasks["task-456"] = &coordinator.TaskRecord{
		ID:    "task-456",
		Title: "Test task",
	}

	// Add events
	taskStore.events["task-456"] = []*coordinator.TaskEventRecord{
		{
			TaskID:     "task-456",
			StreamType: "turn",
			Text:       "Hello",
			CreatedAt:  time.Now(),
		},
		{
			TaskID:     "task-456",
			StreamType: "tool",
			ToolName:   "Read",
			CreatedAt:  time.Now(),
		},
	}

	s := &Server{}
	s.SetTaskEventStore(taskStore)

	req := httptest.NewRequest(http.MethodGet, "/api/coordinator/tasks/task-456/events", nil)
	w := httptest.NewRecorder()

	s.handleCoordinatorTaskEvents_(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 events, got %d", len(result))
	}
}

func TestTaskDiffEndpoint(t *testing.T) {
	taskStore := NewMockTaskEventStore()

	// Add a task with worktree path
	taskStore.tasks["task-789"] = &coordinator.TaskRecord{
		ID:           "task-789",
		Title:        "Test task",
		WorktreePath: "/tmp/nonexistent-worktree",
	}

	s := &Server{}
	s.SetTaskEventStore(taskStore)

	req := httptest.NewRequest(http.MethodGet, "/api/coordinator/tasks/task-789/diff", nil)
	w := httptest.NewRecorder()

	s.handleCoordinatorTaskEvents_(w, req)

	// Should fail because worktree doesn't exist - but handler should not panic
	// Just verify we get a response (error is expected since path doesn't exist)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 200 or 500, got %d", w.Code)
	}
}

func TestTaskDiffEndpointNoWorktree(t *testing.T) {
	taskStore := NewMockTaskEventStore()

	// Add a task WITHOUT worktree path
	taskStore.tasks["task-no-wt"] = &coordinator.TaskRecord{
		ID:    "task-no-wt",
		Title: "No worktree task",
		// No WorktreePath set
	}

	s := &Server{}
	s.SetTaskEventStore(taskStore)

	req := httptest.NewRequest(http.MethodGet, "/api/coordinator/tasks/task-no-wt/diff", nil)
	w := httptest.NewRecorder()

	s.handleCoordinatorTaskEvents_(w, req)

	// Should return 404 because task has no worktree
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
