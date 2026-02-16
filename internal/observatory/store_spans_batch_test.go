package observatory

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ensureTestWorkspace creates a workspace for FK constraints.
// Uses a map keyed by store pointer to handle multiple test stores.
var testWorkspaceStores = make(map[*Store]bool)

func ensureTestWorkspace(t testing.TB, store *Store, now time.Time) {
	t.Helper()
	if testWorkspaceStores[store] {
		return
	}
	ws := &Workspace{
		ID:        "ws-test",
		Name:      "Test",
		Path:      "/tmp/test",
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.CreateWorkspace(ws)
	testWorkspaceStores[store] = true
}

// createTestTask creates a minimal task record to satisfy FK constraints on spans.task_id.
func createTestTask(t testing.TB, store *Store, taskID string, now time.Time) {
	t.Helper()
	ensureTestWorkspace(t, store, now)
	task := &Task{
		ID:          taskID,
		WorkspaceID: "ws-test",
		Title:       taskID,
		SourceType:  TaskSourceManual,
		Status:      TaskStatusPending,
		CreatedAt:   now,
	}
	if err := store.CreateTask(task); err != nil {
		t.Fatalf("CreateTask(%s) failed: %v", taskID, err)
	}
}

func TestStore_ListSpansByTaskIDs_Basic(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)

	// Create tasks first (FK constraint)
	for _, taskID := range []string{"task-aaa", "task-bbb", "task-ccc"} {
		createTestTask(t, store, taskID, now)
	}

	// Create spans across 3 different tasks
	for _, taskID := range []string{"task-aaa", "task-bbb", "task-ccc"} {
		for j := 0; j < 3; j++ {
			span := &Span{
				ID:        fmt.Sprintf("span-%s-%d", taskID, j),
				TraceID:   "trace-1",
				TaskID:    taskID,
				Name:      "operation",
				Kind:      SpanKindInternal,
				Status:    SpanStatusOK,
				StartTime: now.Add(time.Duration(j) * time.Minute),
				CreatedAt: now,
			}
			if err := store.CreateSpan(span); err != nil {
				t.Fatalf("CreateSpan failed: %v", err)
			}
		}
	}

	// Batch query for 2 of the 3 tasks
	result, err := store.ListSpansByTaskIDs([]string{"task-aaa", "task-ccc"}, 0)
	if err != nil {
		t.Fatalf("ListSpansByTaskIDs failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 task groups, got %d", len(result))
	}
	if len(result["task-aaa"]) != 3 {
		t.Errorf("expected 3 spans for task-aaa, got %d", len(result["task-aaa"]))
	}
	if len(result["task-ccc"]) != 3 {
		t.Errorf("expected 3 spans for task-ccc, got %d", len(result["task-ccc"]))
	}
	// task-bbb should NOT be in results
	if len(result["task-bbb"]) != 0 {
		t.Errorf("expected 0 spans for task-bbb (not requested), got %d", len(result["task-bbb"]))
	}
}

func TestStore_ListSpansByTaskIDs_LimitPerTask(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	createTestTask(t, store, "task-many", now)

	// Create 10 spans for a single task
	for j := 0; j < 10; j++ {
		span := &Span{
			ID:        fmt.Sprintf("span-limit-%d", j),
			TraceID:   "trace-1",
			TaskID:    "task-many",
			Name:      "operation",
			Kind:      SpanKindInternal,
			Status:    SpanStatusOK,
			StartTime: now.Add(time.Duration(j) * time.Minute),
			CreatedAt: now,
		}
		if err := store.CreateSpan(span); err != nil {
			t.Fatalf("CreateSpan failed: %v", err)
		}
	}

	// Limit to 3 per task
	result, err := store.ListSpansByTaskIDs([]string{"task-many"}, 3)
	if err != nil {
		t.Fatalf("ListSpansByTaskIDs failed: %v", err)
	}

	if len(result["task-many"]) != 3 {
		t.Errorf("expected 3 spans (limited), got %d", len(result["task-many"]))
	}
}

func TestStore_ListSpansByTaskIDs_EmptyInput(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	result, err := store.ListSpansByTaskIDs(nil, 0)
	if err != nil {
		t.Fatalf("ListSpansByTaskIDs with nil should not error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for nil input, got %v", result)
	}

	result, err = store.ListSpansByTaskIDs([]string{}, 0)
	if err != nil {
		t.Fatalf("ListSpansByTaskIDs with empty should not error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for empty input, got %v", result)
	}
}

func TestStore_ListSpansByTaskIDs_NonExistentTasks(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	result, err := store.ListSpansByTaskIDs([]string{"task-nonexistent"}, 0)
	if err != nil {
		t.Fatalf("ListSpansByTaskIDs failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 groups for nonexistent task, got %d", len(result))
	}
}

func TestStore_ListSpansByTaskIDs_PreservesFields(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	endTime := now.Add(5 * time.Minute)

	createTestTask(t, store, "task-fields", now)

	span := &Span{
		ID:            "span-fields",
		TraceID:       "trace-fields",
		ParentSpanID:  "parent-1",
		TaskID:        "task-fields",
		Name:          "test.operation",
		Kind:          SpanKindClient,
		Status:        SpanStatusError,
		StatusMessage: "something broke",
		StartTime:     now,
		EndTime:       &endTime,
		DurationMs:    300000,
		TokensIn:      100,
		TokensOut:     200,
		CostUSD:       0.05,
		Model:         "claude-sonnet-4-5",
		Provider:      ProviderClaude,
		CreatedAt:     now,
	}
	if err := store.CreateSpan(span); err != nil {
		t.Fatalf("CreateSpan failed: %v", err)
	}

	result, err := store.ListSpansByTaskIDs([]string{"task-fields"}, 0)
	if err != nil {
		t.Fatalf("ListSpansByTaskIDs failed: %v", err)
	}

	if len(result["task-fields"]) != 1 {
		t.Fatalf("expected 1 span, got %d", len(result["task-fields"]))
	}

	got := result["task-fields"][0]
	if got.ID != "span-fields" {
		t.Errorf("ID: got %s, want span-fields", got.ID)
	}
	if got.TraceID != "trace-fields" {
		t.Errorf("TraceID: got %s, want trace-fields", got.TraceID)
	}
	if got.ParentSpanID != "parent-1" {
		t.Errorf("ParentSpanID: got %s, want parent-1", got.ParentSpanID)
	}
	if got.Name != "test.operation" {
		t.Errorf("Name: got %s, want test.operation", got.Name)
	}
	if got.Status != SpanStatusError {
		t.Errorf("Status: got %s, want error", got.Status)
	}
	if got.TokensIn != 100 {
		t.Errorf("TokensIn: got %d, want 100", got.TokensIn)
	}
	if got.TokensOut != 200 {
		t.Errorf("TokensOut: got %d, want 200", got.TokensOut)
	}
	if got.Model != "claude-sonnet-4-5" {
		t.Errorf("Model: got %s, want claude-sonnet-4-5", got.Model)
	}
	if got.Provider != ProviderClaude {
		t.Errorf("Provider: got %s, want claude", got.Provider)
	}
}

// BenchmarkListSpansByTaskIDs_vs_Individual compares batch vs per-task queries.
// This benchmark documents the performance regression that caused 400% CPU.
func BenchmarkListSpansByTaskIDs_vs_Individual(b *testing.B) {
	store := setupBenchStore(b)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)

	// Create workspace for FK
	store.CreateWorkspace(&Workspace{ID: "ws-bench", Name: "Bench", Path: "/tmp/bench", CreatedAt: now, UpdatedAt: now})

	// Create 50 tasks with 50 spans each (realistic dashboard scenario)
	taskIDs := make([]string, 50)
	for i := 0; i < 50; i++ {
		taskIDs[i] = fmt.Sprintf("task-%03d", i)
		task := &Task{ID: taskIDs[i], WorkspaceID: "ws-bench", Title: taskIDs[i], SourceType: TaskSourceManual, Status: TaskStatusPending, CreatedAt: now}
		store.CreateTask(task)

		for j := 0; j < 50; j++ {
			span := &Span{
				ID:        fmt.Sprintf("span-%03d-%03d", i, j),
				TraceID:   fmt.Sprintf("trace-%03d", i),
				TaskID:    taskIDs[i],
				Name:      "operation",
				Kind:      SpanKindInternal,
				Status:    SpanStatusOK,
				StartTime: now.Add(time.Duration(j) * time.Minute),
				CreatedAt: now,
			}
			store.CreateSpan(span)
		}
	}

	b.Run("Individual_N_Queries", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, tid := range taskIDs {
				store.ListSpans(SpanListOptions{TaskID: tid, Limit: 500})
			}
		}
	})

	b.Run("Batch_Single_Query", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			store.ListSpansByTaskIDs(taskIDs, 500)
		}
	})
}

func setupBenchStore(b *testing.B) *Store {
	b.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		b.Fatalf("failed to enable foreign keys: %v", err)
	}
	if err := Migrate(db); err != nil {
		b.Fatalf("failed to migrate: %v", err)
	}
	return NewStore(db)
}
