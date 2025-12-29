package coordinator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSQLiteStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestSQLiteStoreCreateAndGetTask(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	task := &TaskRecord{
		ID:        "task-1",
		MessageID: "msg-1",
		Title:     "Test Task",
		Content:   "Fix the bug in parser",
		Type:      TaskTypeBugFix,
		Priority:  2,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	retrieved, err := store.GetTask(ctx, "task-1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}

	if retrieved.ID != task.ID {
		t.Errorf("got ID %q, want %q", retrieved.ID, task.ID)
	}
	if retrieved.Title != task.Title {
		t.Errorf("got title %q, want %q", retrieved.Title, task.Title)
	}
	if retrieved.Type != task.Type {
		t.Errorf("got type %q, want %q", retrieved.Type, task.Type)
	}
	if retrieved.Priority != task.Priority {
		t.Errorf("got priority %d, want %d", retrieved.Priority, task.Priority)
	}
	if retrieved.Status != task.Status {
		t.Errorf("got status %q, want %q", retrieved.Status, task.Status)
	}
}

func TestSQLiteStoreUpdateTask(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	task := &TaskRecord{
		ID:        "task-1",
		Title:     "Original Title",
		Content:   "Original content",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Update task
	task.Title = "Updated Title"
	task.Status = TaskStatusRunning
	task.Provider = "claude-code"
	now := time.Now()
	task.StartedAt = &now

	if err := store.UpdateTask(ctx, task); err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	retrieved, err := store.GetTask(ctx, "task-1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("got title %q, want %q", retrieved.Title, "Updated Title")
	}
	if retrieved.Status != TaskStatusRunning {
		t.Errorf("got status %q, want %q", retrieved.Status, TaskStatusRunning)
	}
	if retrieved.Provider != "claude-code" {
		t.Errorf("got provider %q, want %q", retrieved.Provider, "claude-code")
	}
}

func TestSQLiteStoreDeleteTask(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	task := &TaskRecord{
		ID:        "task-1",
		Title:     "Task to delete",
		Content:   "Content",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := store.DeleteTask(ctx, "task-1"); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	_, err := store.GetTask(ctx, "task-1")
	if err == nil {
		t.Error("expected error getting deleted task")
	}
}

func TestSQLiteStoreListTasks(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create multiple tasks
	tasks := []*TaskRecord{
		{ID: "task-1", Title: "Bug 1", Content: "c", Type: TaskTypeBugFix, Status: TaskStatusPending, Priority: 1, CreatedAt: time.Now()},
		{ID: "task-2", Title: "Feature 1", Content: "c", Type: TaskTypeFeature, Status: TaskStatusRunning, Priority: 5, CreatedAt: time.Now()},
		{ID: "task-3", Title: "Bug 2", Content: "c", Type: TaskTypeBugFix, Status: TaskStatusCompleted, Priority: 2, CreatedAt: time.Now()},
	}

	for _, task := range tasks {
		if err := store.CreateTask(ctx, task); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// List all
	all, err := store.ListTasks(ctx, &TaskFilter{})
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("got %d tasks, want 3", len(all))
	}

	// Filter by status
	pending, err := store.ListTasks(ctx, &TaskFilter{Status: []TaskStatus{TaskStatusPending}})
	if err != nil {
		t.Fatalf("failed to list pending tasks: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("got %d pending tasks, want 1", len(pending))
	}

	// Filter by type
	bugs, err := store.ListTasks(ctx, &TaskFilter{Type: []TaskType{TaskTypeBugFix}})
	if err != nil {
		t.Fatalf("failed to list bug tasks: %v", err)
	}
	if len(bugs) != 2 {
		t.Errorf("got %d bug tasks, want 2", len(bugs))
	}

	// With limit
	limited, err := store.ListTasks(ctx, &TaskFilter{Limit: 2})
	if err != nil {
		t.Fatalf("failed to list limited tasks: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("got %d tasks with limit, want 2", len(limited))
	}
}

func TestSQLiteStoreTaskStateTransitions(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	task := &TaskRecord{
		ID:        "task-1",
		Title:     "Test Task",
		Content:   "Content",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Mark queued
	if err := store.MarkTaskQueued(ctx, "task-1"); err != nil {
		t.Fatalf("failed to mark queued: %v", err)
	}
	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Status != TaskStatusQueued {
		t.Errorf("got status %q, want %q", retrieved.Status, TaskStatusQueued)
	}

	// Mark running
	if err := store.MarkTaskRunning(ctx, "task-1", "claude-code", "wt-123"); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}
	retrieved, _ = store.GetTask(ctx, "task-1")
	if retrieved.Status != TaskStatusRunning {
		t.Errorf("got status %q, want %q", retrieved.Status, TaskStatusRunning)
	}
	if retrieved.Provider != "claude-code" {
		t.Errorf("got provider %q, want %q", retrieved.Provider, "claude-code")
	}
	if retrieved.WorktreeID != "wt-123" {
		t.Errorf("got worktree %q, want %q", retrieved.WorktreeID, "wt-123")
	}
	if retrieved.StartedAt == nil {
		t.Error("expected started_at to be set")
	}

	// Mark completed
	result := &ExecuteResult{
		Success:    true,
		Output:     "Task completed successfully",
		Duration:   5 * time.Second,
		Cost:       0.05,
		TokensUsed: 1000,
	}
	if err := store.MarkTaskCompleted(ctx, "task-1", result); err != nil {
		t.Fatalf("failed to mark completed: %v", err)
	}
	retrieved, _ = store.GetTask(ctx, "task-1")
	if retrieved.Status != TaskStatusCompleted {
		t.Errorf("got status %q, want %q", retrieved.Status, TaskStatusCompleted)
	}
	if retrieved.Output != "Task completed successfully" {
		t.Errorf("got output %q, want %q", retrieved.Output, "Task completed successfully")
	}
	if retrieved.Cost != 0.05 {
		t.Errorf("got cost %f, want %f", retrieved.Cost, 0.05)
	}
	if retrieved.TokensUsed != 1000 {
		t.Errorf("got tokens %d, want %d", retrieved.TokensUsed, 1000)
	}
}

func TestSQLiteStoreGetTaskStats(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create tasks with various statuses
	tasks := []*TaskRecord{
		{ID: "task-1", Title: "t1", Content: "c", Type: TaskTypeBugFix, Status: TaskStatusPending, CreatedAt: time.Now()},
		{ID: "task-2", Title: "t2", Content: "c", Type: TaskTypeBugFix, Status: TaskStatusRunning, CreatedAt: time.Now()},
		{ID: "task-3", Title: "t3", Content: "c", Type: TaskTypeFeature, Status: TaskStatusCompleted, CreatedAt: time.Now()},
		{ID: "task-4", Title: "t4", Content: "c", Type: TaskTypeFeature, Status: TaskStatusFailed, CreatedAt: time.Now()},
	}

	for _, task := range tasks {
		if err := store.CreateTask(ctx, task); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	stats, err := store.GetTaskStats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.TotalTasks != 4 {
		t.Errorf("got total %d, want 4", stats.TotalTasks)
	}
	if stats.PendingTasks != 1 {
		t.Errorf("got pending %d, want 1", stats.PendingTasks)
	}
	if stats.RunningTasks != 1 {
		t.Errorf("got running %d, want 1", stats.RunningTasks)
	}
	if stats.CompletedTasks != 1 {
		t.Errorf("got completed %d, want 1", stats.CompletedTasks)
	}
	if stats.FailedTasks != 1 {
		t.Errorf("got failed %d, want 1", stats.FailedTasks)
	}
	if stats.ByType[string(TaskTypeBugFix)] != 2 {
		t.Errorf("got bug-fix %d, want 2", stats.ByType[string(TaskTypeBugFix)])
	}
	if stats.ByType[string(TaskTypeFeature)] != 2 {
		t.Errorf("got feature %d, want 2", stats.ByType[string(TaskTypeFeature)])
	}
}

func TestSQLiteStoreDeleteOldTasks(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create old and new tasks
	oldTime := time.Now().Add(-48 * time.Hour)
	newTime := time.Now()

	tasks := []*TaskRecord{
		{ID: "old-1", Title: "old", Content: "c", Type: TaskTypeBugFix, Status: TaskStatusCompleted, CreatedAt: oldTime},
		{ID: "old-2", Title: "old", Content: "c", Type: TaskTypeBugFix, Status: TaskStatusFailed, CreatedAt: oldTime},
		{ID: "new-1", Title: "new", Content: "c", Type: TaskTypeBugFix, Status: TaskStatusCompleted, CreatedAt: newTime},
		{ID: "pending", Title: "pending", Content: "c", Type: TaskTypeBugFix, Status: TaskStatusPending, CreatedAt: oldTime}, // Should NOT be deleted
	}

	for _, task := range tasks {
		if err := store.CreateTask(ctx, task); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// Delete tasks older than 24 hours
	deleted, err := store.DeleteOldTasks(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to delete old tasks: %v", err)
	}

	if deleted != 2 {
		t.Errorf("deleted %d tasks, want 2 (old completed/failed only)", deleted)
	}

	// Verify remaining tasks
	remaining, _ := store.ListTasks(ctx, &TaskFilter{})
	if len(remaining) != 2 {
		t.Errorf("got %d remaining tasks, want 2", len(remaining))
	}
}

func TestSQLiteStoreFingerprint(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	task := &TaskRecord{
		ID:        "task-1",
		Title:     "Test Task",
		Content:   "Fix the bug in parser",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	fingerprint := uint64(12345678901234)
	if err := store.SetTaskFingerprint(ctx, "task-1", fingerprint); err != nil {
		t.Fatalf("failed to set fingerprint: %v", err)
	}

	// Find by exact fingerprint
	dup, err := store.FindDuplicateTask(ctx, fingerprint, 0.9)
	if err != nil {
		t.Fatalf("failed to find duplicate: %v", err)
	}
	if dup == nil {
		t.Error("expected to find duplicate")
	}
	if dup != nil && dup.ID != "task-1" {
		t.Errorf("got ID %q, want task-1", dup.ID)
	}

	// Non-matching fingerprint
	dup2, err := store.FindDuplicateTask(ctx, 99999999, 0.9)
	if err != nil {
		t.Fatalf("failed to find duplicate: %v", err)
	}
	if dup2 != nil {
		t.Error("expected no duplicate for different fingerprint")
	}
}

// Helper to create a test store
func createTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return store
}
