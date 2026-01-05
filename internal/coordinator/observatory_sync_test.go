package coordinator

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

func TestObservatorySyncTask(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "obs-sync-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create observatory backend
	dbPath := filepath.Join(tmpDir, "observatory.db")
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create observatory backend: %v", err)
	}
	defer backend.Close()

	sync := NewObservatorySync(backend, log.Default())

	ctx := context.Background()

	// Create a coordinator task
	task := &TaskRecord{
		ID:        "task-001",
		Title:     "Test Task",
		Content:   "Test task content",
		Type:      TaskTypeBugFix,
		Priority:  50,
		Status:    TaskStatusPending,
		Workspace: tmpDir,
		CreatedAt: time.Now(),
	}

	// Sync the task
	if err := sync.SyncTask(ctx, task); err != nil {
		t.Fatalf("SyncTask failed: %v", err)
	}

	// Verify task exists in observatory
	obsTask, err := backend.GetTask(ctx, "task-001")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if obsTask == nil {
		t.Fatal("Task not found in observatory")
	}
	if obsTask.Title != "Test Task" {
		t.Errorf("Expected title 'Test Task', got '%s'", obsTask.Title)
	}
	if obsTask.Status != observatory.TaskStatusPending {
		t.Errorf("Expected status pending, got '%s'", obsTask.Status)
	}

	// Update the task status
	task.Status = TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now

	if err := sync.SyncTask(ctx, task); err != nil {
		t.Fatalf("SyncTask (update) failed: %v", err)
	}

	// Verify update
	obsTask, _ = backend.GetTask(ctx, "task-001")
	if obsTask.Status != observatory.TaskStatusRunning {
		t.Errorf("Expected status running after update, got '%s'", obsTask.Status)
	}
}

func TestObservatorySyncAgentAssignment(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "obs-sync-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create observatory backend
	dbPath := filepath.Join(tmpDir, "observatory.db")
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create observatory backend: %v", err)
	}
	defer backend.Close()

	sync := NewObservatorySync(backend, log.Default())

	ctx := context.Background()

	// First create a workspace and task
	task := &TaskRecord{
		ID:        "task-002",
		Title:     "Agent Assignment Test",
		Content:   "Test content",
		Priority:  50,
		Status:    TaskStatusPending,
		Workspace: tmpDir,
		CreatedAt: time.Now(),
	}
	if err := sync.SyncTask(ctx, task); err != nil {
		t.Fatalf("SyncTask failed: %v", err)
	}

	// Create agent assignment
	assignmentID, err := sync.SyncAgentAssignment(ctx, "task-002", "design-doc-creator", "claude")
	if err != nil {
		t.Fatalf("SyncAgentAssignment failed: %v", err)
	}
	if assignmentID == "" {
		t.Fatal("Expected non-empty assignment ID")
	}
	if !hasPrefix(assignmentID, "aa_") {
		t.Errorf("Expected assignment ID to start with 'aa_', got '%s'", assignmentID)
	}

	// Verify assignment exists
	assignment, err := backend.GetAgentAssignment(ctx, assignmentID)
	if err != nil {
		t.Fatalf("GetAgentAssignment failed: %v", err)
	}
	if assignment == nil {
		t.Fatal("Assignment not found in observatory")
	}
	if assignment.AgentID != "design-doc-creator" {
		t.Errorf("Expected agent 'design-doc-creator', got '%s'", assignment.AgentID)
	}
	if assignment.Provider != observatory.ProviderClaude {
		t.Errorf("Expected provider 'claude', got '%s'", assignment.Provider)
	}
	if assignment.Status != observatory.AgentStatusRunning {
		t.Errorf("Expected status running, got '%s'", assignment.Status)
	}

	// Complete the assignment
	if err := sync.CompleteAgentAssignment(ctx, assignmentID, true); err != nil {
		t.Fatalf("CompleteAgentAssignment failed: %v", err)
	}

	// Verify completion
	assignment, _ = backend.GetAgentAssignment(ctx, assignmentID)
	if assignment.Status != observatory.AgentStatusCompleted {
		t.Errorf("Expected status completed, got '%s'", assignment.Status)
	}
	if assignment.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestObservatorySyncWorkspaceCreation(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "obs-sync-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create observatory backend
	dbPath := filepath.Join(tmpDir, "observatory.db")
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		t.Fatalf("Failed to create observatory backend: %v", err)
	}
	defer backend.Close()

	sync := NewObservatorySync(backend, log.Default())

	ctx := context.Background()

	// Create two tasks with the same workspace
	task1 := &TaskRecord{
		ID:        "task-ws-1",
		Title:     "Task 1",
		Content:   "Content 1",
		Workspace: tmpDir,
		CreatedAt: time.Now(),
	}
	task2 := &TaskRecord{
		ID:        "task-ws-2",
		Title:     "Task 2",
		Content:   "Content 2",
		Workspace: tmpDir,
		CreatedAt: time.Now(),
	}

	if err := sync.SyncTask(ctx, task1); err != nil {
		t.Fatalf("SyncTask 1 failed: %v", err)
	}
	if err := sync.SyncTask(ctx, task2); err != nil {
		t.Fatalf("SyncTask 2 failed: %v", err)
	}

	// Verify both tasks have the same workspace ID
	obsTask1, _ := backend.GetTask(ctx, "task-ws-1")
	obsTask2, _ := backend.GetTask(ctx, "task-ws-2")

	if obsTask1.WorkspaceID != obsTask2.WorkspaceID {
		t.Errorf("Expected same workspace ID, got '%s' and '%s'", obsTask1.WorkspaceID, obsTask2.WorkspaceID)
	}
	if obsTask1.WorkspaceID == "" {
		t.Error("Expected non-empty workspace ID")
	}

	// Verify workspace exists
	ws, err := backend.GetWorkspace(ctx, obsTask1.WorkspaceID)
	if err != nil {
		t.Fatalf("GetWorkspace failed: %v", err)
	}
	if ws == nil {
		t.Fatal("Workspace not found")
	}
	if ws.Path != tmpDir {
		t.Errorf("Expected path '%s', got '%s'", tmpDir, ws.Path)
	}
}

func TestObservatorySyncNilBackend(t *testing.T) {
	// Test that sync operations are no-ops when backend is nil
	sync := NewObservatorySync(nil, log.Default())

	ctx := context.Background()
	task := &TaskRecord{
		ID:        "task-nil",
		Title:     "Nil Backend Test",
		CreatedAt: time.Now(),
	}

	// Should not error with nil backend
	if err := sync.SyncTask(ctx, task); err != nil {
		t.Errorf("SyncTask with nil backend should not error: %v", err)
	}

	assignmentID, err := sync.SyncAgentAssignment(ctx, "task-nil", "agent", "claude")
	if err != nil {
		t.Errorf("SyncAgentAssignment with nil backend should not error: %v", err)
	}
	if assignmentID != "" {
		t.Errorf("Expected empty assignment ID with nil backend, got '%s'", assignmentID)
	}

	if err := sync.CompleteAgentAssignment(ctx, "", true); err != nil {
		t.Errorf("CompleteAgentAssignment with nil backend should not error: %v", err)
	}
}

func TestConvertTaskStatus(t *testing.T) {
	sync := &ObservatorySync{}

	tests := []struct {
		input    TaskStatus
		expected observatory.TaskStatus
	}{
		{TaskStatusPending, observatory.TaskStatusPending},
		{TaskStatusQueued, observatory.TaskStatusPending},
		{TaskStatusRunning, observatory.TaskStatusRunning},
		{TaskStatusCompleted, observatory.TaskStatusCompleted},
		{TaskStatusPendingApproval, observatory.TaskStatusCompleted},
		{TaskStatusFailed, observatory.TaskStatusFailed},
		{TaskStatusRejected, observatory.TaskStatusFailed},
		{TaskStatusCancelled, observatory.TaskStatusFailed},
	}

	for _, tt := range tests {
		result := sync.convertTaskStatus(tt.input)
		if result != tt.expected {
			t.Errorf("convertTaskStatus(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestConvertProvider(t *testing.T) {
	sync := &ObservatorySync{}

	tests := []struct {
		input    string
		expected observatory.Provider
	}{
		{"claude", observatory.ProviderClaude},
		{"claude-code", observatory.ProviderClaude},
		{"Claude", observatory.ProviderClaude},
		{"gemini", observatory.ProviderGemini},
		{"gemini-cli", observatory.ProviderGemini},
		{"ollama", observatory.ProviderOllama},
		{"unknown", observatory.ProviderClaude}, // Default
	}

	for _, tt := range tests {
		result := sync.convertProvider(tt.input)
		if result != tt.expected {
			t.Errorf("convertProvider(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestConvertPriority(t *testing.T) {
	sync := &ObservatorySync{}

	tests := []struct {
		input    int
		expected string
	}{
		{100, "critical"},
		{80, "critical"},
		{79, "high"},
		{60, "high"},
		{59, "medium"},
		{40, "medium"},
		{39, "low"},
		{0, "low"},
	}

	for _, tt := range tests {
		result := sync.convertPriority(tt.input)
		if result != tt.expected {
			t.Errorf("convertPriority(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

// hasPrefix checks if a string starts with a prefix
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
