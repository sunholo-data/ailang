// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestLookupTaskBySessionID_Found(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	store := NewStore(db)

	// Create a workspace
	ws := &Workspace{
		ID:        "ws-1",
		Name:      "test-workspace",
		Path:      "/test/path",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.CreateWorkspace(ws); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Create a task
	task := &Task{
		ID:          "task-12345678",
		WorkspaceID: ws.ID,
		Title:       "Test Task",
		Status:      TaskStatusRunning,
		CreatedAt:   time.Now(),
	}
	if err := store.CreateTask(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Create an agent assignment
	assignment := &AgentAssignment{
		ID:         "assign-1",
		TaskID:     task.ID,
		AgentID:    "claude-code",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	if err := store.CreateAgentAssignment(assignment); err != nil {
		t.Fatalf("failed to create assignment: %v", err)
	}

	// Create a claude.execute span with session.id that links task and assignment
	sessionID := "session-abc123"
	traceID := generateTraceID()
	span := &Span{
		ID:                generateSpanID(),
		TraceID:           traceID,
		TaskID:            task.ID,
		AgentAssignmentID: assignment.ID,
		Name:              "claude.execute",
		Kind:              SpanKindClient,
		Status:            SpanStatusOK,
		StartTime:         time.Now(),
		Attributes: map[string]any{
			"session.id": sessionID,
		},
	}
	if err := store.CreateSpan(span); err != nil {
		t.Fatalf("failed to create span: %v", err)
	}

	// Now lookup by session ID
	foundTaskID, foundAssignmentID, foundTraceID := store.LookupTaskBySessionID(sessionID)

	if foundTaskID != task.ID {
		t.Errorf("expected task_id %s, got %s", task.ID, foundTaskID)
	}
	if foundAssignmentID != assignment.ID {
		t.Errorf("expected assignment_id %s, got %s", assignment.ID, foundAssignmentID)
	}
	if foundTraceID != traceID {
		t.Errorf("expected trace_id %s, got %s", traceID, foundTraceID)
	}
}

func TestLookupTaskBySessionID_NotFound(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	store := NewStore(db)

	// Lookup non-existent session
	foundTaskID, foundAssignmentID, foundTraceID := store.LookupTaskBySessionID("non-existent-session")

	if foundTaskID != "" {
		t.Errorf("expected empty task_id, got %s", foundTaskID)
	}
	if foundAssignmentID != "" {
		t.Errorf("expected empty assignment_id, got %s", foundAssignmentID)
	}
	if foundTraceID != "" {
		t.Errorf("expected empty trace_id, got %s", foundTraceID)
	}
}

func TestLinkOrphanedSpansBySession(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	store := NewStore(db)

	// Create workspace and task
	ws := &Workspace{
		ID:        "ws-1",
		Name:      "test-workspace",
		Path:      "/test/path",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.CreateWorkspace(ws); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	task := &Task{
		ID:          "task-orphan-test",
		WorkspaceID: ws.ID,
		Title:       "Orphan Test Task",
		Status:      TaskStatusRunning,
		CreatedAt:   time.Now(),
	}
	if err := store.CreateTask(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	assignment := &AgentAssignment{
		ID:         "assign-orphan",
		TaskID:     task.ID,
		AgentID:    "claude-code",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	if err := store.CreateAgentAssignment(assignment); err != nil {
		t.Fatalf("failed to create assignment: %v", err)
	}

	sessionID := "session-orphan-test"

	// Create orphaned spans (have session.id but no task_id)
	for i := 0; i < 3; i++ {
		span := &Span{
			ID:        generateSpanID(),
			TraceID:   generateTraceID(),
			Name:      "claude_code.api_request",
			Kind:      SpanKindClient,
			Status:    SpanStatusOK,
			StartTime: time.Now(),
			TokensIn:  100,
			TokensOut: 50,
			Attributes: map[string]any{
				"session.id": sessionID,
			},
		}
		if err := store.CreateSpan(span); err != nil {
			t.Fatalf("failed to create orphaned span: %v", err)
		}
	}

	// Create one span that already has task_id (should NOT be updated)
	linkedSpan := &Span{
		ID:                generateSpanID(),
		TraceID:           generateTraceID(),
		TaskID:            task.ID,
		AgentAssignmentID: assignment.ID,
		Name:              "already-linked",
		Kind:              SpanKindClient,
		Status:            SpanStatusOK,
		StartTime:         time.Now(),
		Attributes: map[string]any{
			"session.id": sessionID,
		},
	}
	if err := store.CreateSpan(linkedSpan); err != nil {
		t.Fatalf("failed to create linked span: %v", err)
	}

	// Link orphaned spans
	linked, err := store.LinkOrphanedSpansBySession(sessionID, task.ID, assignment.ID)
	if err != nil {
		t.Fatalf("LinkOrphanedSpansBySession failed: %v", err)
	}

	if linked != 3 {
		t.Errorf("expected 3 spans linked, got %d", linked)
	}

	// Verify spans are now linked
	spans, err := store.ListSpans(SpanListOptions{TaskID: task.ID})
	if err != nil {
		t.Fatalf("failed to list spans: %v", err)
	}

	// Should have 4 spans now (3 orphaned + 1 already linked)
	if len(spans) != 4 {
		t.Errorf("expected 4 spans linked to task, got %d", len(spans))
	}

	// Verify all have correct assignment_id
	for _, span := range spans {
		if span.AgentAssignmentID != assignment.ID {
			t.Errorf("span %s has wrong assignment_id: %s", span.ID, span.AgentAssignmentID)
		}
	}
}

func TestLinkOrphanedSpansBySession_NoOrphans(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	store := NewStore(db)

	// Try to link with no orphaned spans
	linked, err := store.LinkOrphanedSpansBySession("non-existent-session", "task-1", "assign-1")
	if err != nil {
		t.Fatalf("LinkOrphanedSpansBySession failed: %v", err)
	}

	if linked != 0 {
		t.Errorf("expected 0 spans linked, got %d", linked)
	}
}

func TestSessionCorrelation_EndToEnd(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Setup: Create workspace, task, assignment
	ws := &Workspace{
		ID:        "ws-e2e",
		Name:      "E2E Workspace",
		Path:      "/e2e/path",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := backend.CreateWorkspace(ctx, ws); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	task := &Task{
		ID:          "task-e2e",
		WorkspaceID: ws.ID,
		Title:       "E2E Task",
		Status:      TaskStatusRunning,
		CreatedAt:   time.Now(),
	}
	if err := backend.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	assignment := &AgentAssignment{
		ID:         "assign-e2e",
		TaskID:     task.ID,
		AgentID:    "claude-code",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	if err := backend.CreateAgentAssignment(ctx, assignment); err != nil {
		t.Fatalf("failed to create assignment: %v", err)
	}

	sessionID := "session-e2e-test"
	traceID := generateTraceID()

	// Step 1: Orphaned Claude Code events arrive first (via OTLP logs)
	// These have session.id but no task_id
	orphanSpan := &Span{
		ID:        generateSpanID(),
		TraceID:   generateTraceID(),
		Name:      "claude_code.api_request",
		Kind:      SpanKindClient,
		Status:    SpanStatusOK,
		StartTime: time.Now().Add(-time.Minute),
		TokensIn:  500,
		TokensOut: 200,
		CostUSD:   0.05,
		Attributes: map[string]any{
			"session.id": sessionID,
		},
	}
	if err := backend.CreateSpan(ctx, orphanSpan); err != nil {
		t.Fatalf("failed to create orphan span: %v", err)
	}

	// Step 2: claude.execute span arrives later (via OTLP traces)
	// This has BOTH session.id AND task_id
	executeSpan := &Span{
		ID:                generateSpanID(),
		TraceID:           traceID,
		TaskID:            task.ID,
		AgentAssignmentID: assignment.ID,
		Name:              "claude.execute",
		Kind:              SpanKindClient,
		Status:            SpanStatusOK,
		StartTime:         time.Now(),
		Attributes: map[string]any{
			"session.id": sessionID,
		},
	}
	if err := backend.CreateSpan(ctx, executeSpan); err != nil {
		t.Fatalf("failed to create execute span: %v", err)
	}

	// Step 3: Lookup by session ID should find the task hierarchy
	foundTaskID, foundAssignmentID, foundTraceID := backend.LookupTaskBySessionID(ctx, sessionID)
	if foundTaskID != task.ID {
		t.Errorf("lookup: expected task_id %s, got %s", task.ID, foundTaskID)
	}
	if foundAssignmentID != assignment.ID {
		t.Errorf("lookup: expected assignment_id %s, got %s", assignment.ID, foundAssignmentID)
	}
	if foundTraceID != traceID {
		t.Errorf("lookup: expected trace_id %s, got %s", traceID, foundTraceID)
	}

	// Step 4: Link orphaned spans
	linked, err := backend.LinkOrphanedSpansBySession(ctx, sessionID, task.ID, assignment.ID)
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}
	if linked != 1 {
		t.Errorf("expected 1 span linked, got %d", linked)
	}

	// Step 5: Verify orphan is now linked
	spans, err := backend.ListSpans(ctx, SpanListOptions{TaskID: task.ID})
	if err != nil {
		t.Fatalf("list spans failed: %v", err)
	}

	// Should have 2 spans: execute + orphan
	if len(spans) != 2 {
		t.Errorf("expected 2 spans, got %d", len(spans))
	}

	// Find the orphan span and verify it's linked
	for _, s := range spans {
		if s.Name == "claude_code.api_request" {
			if s.TaskID != task.ID {
				t.Errorf("orphan task_id not updated: got %s", s.TaskID)
			}
			if s.AgentAssignmentID != assignment.ID {
				t.Errorf("orphan assignment_id not updated: got %s", s.AgentAssignmentID)
			}
		}
	}
}

func TestSQLiteBackend_SessionCorrelation(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Test that SQLiteBackend properly delegates session correlation methods

	// LookupTaskBySessionID - should return empty for non-existent session
	taskID, assignmentID, traceID := backend.LookupTaskBySessionID(ctx, "non-existent")
	if taskID != "" || assignmentID != "" || traceID != "" {
		t.Errorf("expected empty results for non-existent session")
	}

	// LinkOrphanedSpansBySession - should return 0 for non-existent session
	linked, err := backend.LinkOrphanedSpansBySession(ctx, "non-existent", "task", "assign")
	if err != nil {
		t.Fatalf("LinkOrphanedSpansBySession should not error: %v", err)
	}
	if linked != 0 {
		t.Errorf("expected 0 linked for non-existent session, got %d", linked)
	}
}
