package observatory

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestBackend(t *testing.T) Backend {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	backend, err := NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	return backend
}

// TestSQLiteBackend_Interface is unnecessary since setupTestBackend returns Backend.
// Interface compliance is enforced at compile time by the return type.

func TestSQLiteBackend_WorkspaceOperations(t *testing.T) {
	backend := setupTestBackend(t)
	defer backend.Close()
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	ws := &Workspace{
		ID:        "ws-1",
		Name:      "Test Workspace",
		Path:      "/test/path",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create
	if err := backend.CreateWorkspace(ctx, ws); err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	// Get
	got, err := backend.GetWorkspace(ctx, "ws-1")
	if err != nil {
		t.Fatalf("GetWorkspace failed: %v", err)
	}
	if got.Name != ws.Name {
		t.Errorf("Name mismatch")
	}

	// List
	list, err := backend.ListWorkspaces(ctx)
	if err != nil {
		t.Fatalf("ListWorkspaces failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 workspace, got %d", len(list))
	}

	// Update
	ws.Name = "Updated"
	if err := backend.UpdateWorkspace(ctx, ws); err != nil {
		t.Fatalf("UpdateWorkspace failed: %v", err)
	}

	// Delete
	if err := backend.DeleteWorkspace(ctx, "ws-1"); err != nil {
		t.Fatalf("DeleteWorkspace failed: %v", err)
	}

	_, err = backend.GetWorkspace(ctx, "ws-1")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows after delete")
	}
}

func TestSQLiteBackend_TaskOperations(t *testing.T) {
	backend := setupTestBackend(t)
	defer backend.Close()
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Create workspace first
	ws := &Workspace{ID: "ws-1", Name: "Test", Path: "/test", CreatedAt: now, UpdatedAt: now}
	backend.CreateWorkspace(ctx, ws)

	task := &Task{
		ID:          "task-1",
		WorkspaceID: "ws-1",
		Title:       "Test Task",
		SourceType:  TaskSourceManual,
		Status:      TaskStatusPending,
		Priority:    "P1",
		CreatedAt:   now,
	}

	// Create
	if err := backend.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Get
	got, err := backend.GetTask(ctx, "task-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if got.Title != task.Title {
		t.Errorf("Title mismatch")
	}

	// List
	tasks, err := backend.ListTasks(ctx, TaskListOptions{})
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// Update
	task.Status = TaskStatusRunning
	if err := backend.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Delete
	if err := backend.DeleteTask(ctx, "task-1"); err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}
}

func TestSQLiteBackend_SpanOperations(t *testing.T) {
	backend := setupTestBackend(t)
	defer backend.Close()
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	span := &Span{
		ID:        "span-1",
		TraceID:   "trace-1",
		Name:      "test.operation",
		Kind:      SpanKindClient,
		Status:    SpanStatusOK,
		StartTime: now,
		Provider:  ProviderClaude,
		CreatedAt: now,
	}

	// Create
	if err := backend.CreateSpan(ctx, span); err != nil {
		t.Fatalf("CreateSpan failed: %v", err)
	}

	// Get
	got, err := backend.GetSpan(ctx, "span-1")
	if err != nil {
		t.Fatalf("GetSpan failed: %v", err)
	}
	if got.Name != span.Name {
		t.Errorf("Name mismatch")
	}

	// List
	spans, err := backend.ListSpans(ctx, SpanListOptions{TraceID: "trace-1"})
	if err != nil {
		t.Fatalf("ListSpans failed: %v", err)
	}
	if len(spans) != 1 {
		t.Errorf("expected 1 span, got %d", len(spans))
	}

	// Get trace
	trace, err := backend.GetTrace(ctx, "trace-1")
	if err != nil {
		t.Fatalf("GetTrace failed: %v", err)
	}
	if trace.SpanCount != 1 {
		t.Errorf("expected 1 span in trace")
	}

	// Update
	endTime := now.Add(time.Second)
	span.EndTime = &endTime
	if err := backend.UpdateSpan(ctx, span); err != nil {
		t.Fatalf("UpdateSpan failed: %v", err)
	}

	// Delete
	if err := backend.DeleteSpan(ctx, "span-1"); err != nil {
		t.Fatalf("DeleteSpan failed: %v", err)
	}
}

func TestSQLiteBackend_MessageOperations(t *testing.T) {
	backend := setupTestBackend(t)
	defer backend.Close()
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	msg := &Message{
		ID:          "msg-1",
		Inbox:       "test-inbox",
		FromAgent:   "test-agent",
		Title:       "Test Message",
		Content:     "Test content",
		MessageType: "info",
		Status:      MessageStatusUnread,
		Priority:    "P2",
		CreatedAt:   now,
	}

	// Create
	if err := backend.CreateMessage(ctx, msg); err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	// Get
	got, err := backend.GetMessage(ctx, "msg-1")
	if err != nil {
		t.Fatalf("GetMessage failed: %v", err)
	}
	if got.Title != msg.Title {
		t.Errorf("Title mismatch")
	}

	// List
	messages, err := backend.ListMessages(ctx, MessageListOptions{Inbox: "test-inbox"})
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	// Mark read
	if err := backend.MarkMessageRead(ctx, "msg-1"); err != nil {
		t.Fatalf("MarkMessageRead failed: %v", err)
	}

	got, _ = backend.GetMessage(ctx, "msg-1")
	if got.Status != MessageStatusRead {
		t.Errorf("expected read status")
	}

	// Mark archived
	if err := backend.MarkMessageArchived(ctx, "msg-1"); err != nil {
		t.Fatalf("MarkMessageArchived failed: %v", err)
	}

	got, _ = backend.GetMessage(ctx, "msg-1")
	if got.Status != MessageStatusArchived {
		t.Errorf("expected archived status")
	}

	// Delete
	if err := backend.DeleteMessage(ctx, "msg-1"); err != nil {
		t.Fatalf("DeleteMessage failed: %v", err)
	}
}

// NOTE: span_events table removed in v4 migration (M-DB-CLEANUP) - test skipped
func TestSQLiteBackend_SpanEventOperations(t *testing.T) {
	t.Skip("span_events table removed in v4 migration (M-DB-CLEANUP)")
}

func TestSQLiteBackend_AggregateOperations(t *testing.T) {
	backend := setupTestBackend(t)
	defer backend.Close()
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Create test data
	ws := &Workspace{ID: "ws-1", Name: "Test", Path: "/test", CreatedAt: now, UpdatedAt: now}
	backend.CreateWorkspace(ctx, ws)

	task := &Task{ID: "task-1", WorkspaceID: "ws-1", Title: "Test", SourceType: TaskSourceManual, Status: TaskStatusCompleted, Priority: "P1", CreatedAt: now}
	backend.CreateTask(ctx, task)

	span := &Span{
		ID:        "span-1",
		TraceID:   "trace-1",
		Name:      "test",
		Kind:      SpanKindClient,
		Status:    SpanStatusOK,
		StartTime: now,
		TokensIn:  1000,
		TokensOut: 500,
		CostUSD:   0.05,
		Provider:  ProviderClaude,
		CreatedAt: now,
	}
	backend.CreateSpan(ctx, span)

	// Get metrics summary
	summary, err := backend.GetMetricsSummary(ctx)
	if err != nil {
		t.Fatalf("GetMetricsSummary failed: %v", err)
	}
	if summary.TotalWorkspaces != 1 {
		t.Errorf("TotalWorkspaces mismatch: got %d", summary.TotalWorkspaces)
	}
	if summary.TotalTasks != 1 {
		t.Errorf("TotalTasks mismatch: got %d", summary.TotalTasks)
	}
	if summary.TotalSpans != 1 {
		t.Errorf("TotalSpans mismatch: got %d", summary.TotalSpans)
	}

	// Get provider comparison
	comparisons, err := backend.GetProviderComparison(ctx)
	if err != nil {
		t.Fatalf("GetProviderComparison failed: %v", err)
	}
	// May be empty if view has no data aggregated yet
	_ = comparisons
}

func TestNewSQLiteBackendFromPath(t *testing.T) {
	// Use temp file
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteBackendFromPath failed: %v", err)
	}
	defer backend.Close()

	// Verify it works
	ctx := context.Background()
	now := time.Now()
	ws := &Workspace{ID: "ws-1", Name: "Test", Path: "/test", CreatedAt: now, UpdatedAt: now}
	if err := backend.CreateWorkspace(ctx, ws); err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}
}
