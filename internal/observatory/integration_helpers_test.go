// Package observatory provides integration test helper functions.
package observatory

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ============================================================================
// Helper Functions
// ============================================================================

func setupIntegrationBackend(t *testing.T) *SQLiteBackend {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory db: %v", err)
	}

	backend, err := NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	return backend
}

func setupUnlinkedEntities(t *testing.T, backend *SQLiteBackend, ctx context.Context) {
	t.Helper()

	// Workspace
	ws := &Workspace{ID: "ws-unlinked", Name: "Unlinked Test", Path: "/tmp/unlinked", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)

	// Task (no spans yet)
	task := &Task{
		ID:          "task-unlinked-001",
		WorkspaceID: "ws-unlinked",
		Title:       "Unlinked Task",
		Status:      TaskStatusRunning,
		CreatedAt:   time.Now(),
	}
	backend.CreateTask(ctx, task)

	// Agent assignment
	aa := &AgentAssignment{
		ID:         "assign-unlinked-001",
		TaskID:     "task-unlinked-001",
		AgentID:    "unlinked-agent",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	backend.CreateAgentAssignment(ctx, aa)

	// Unlinked span (no task_id, no agent_assignment_id)
	span := &Span{
		ID:         "span-unlinked-001",
		TraceID:    "trace-unlinked-001",
		Name:       "unlinked.span",
		Kind:       SpanKindClient,
		Status:     SpanStatusOK,
		StartTime:  time.Now(),
		DurationMs: 1000,
		CreatedAt:  time.Now(),
	}
	backend.CreateSpan(ctx, span)
}

func setupRichHierarchy(t *testing.T, backend *SQLiteBackend, ctx context.Context) {
	t.Helper()

	// Workspace
	ws := &Workspace{ID: "ws-hier", Name: "Hierarchy Test", Path: "/tmp/hier", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)

	// Task
	task := &Task{
		ID:          "task-hier-001",
		WorkspaceID: "ws-hier",
		Title:       "Full Hierarchy Test Task",
		Status:      TaskStatusRunning,
		CreatedAt:   time.Now(),
	}
	backend.CreateTask(ctx, task)

	// Two agents
	for _, agent := range []struct {
		id       string
		provider Provider
	}{
		{"design-doc-creator", ProviderClaude},
		{"sprint-planner", ProviderGemini},
	} {
		aa := &AgentAssignment{
			ID:         fmt.Sprintf("assign-hier-%s", agent.id),
			TaskID:     "task-hier-001",
			AgentID:    agent.id,
			Provider:   agent.provider,
			Status:     AgentStatusRunning,
			AssignedAt: time.Now(),
		}
		backend.CreateAgentAssignment(ctx, aa)
	}

	// Spans for Claude agent
	claudeSpans := []struct {
		id     string
		name   string
		parent string
		tokens int64
	}{
		{"span-hier-c1", "executor.claude.execute", "", 0},
		{"span-hier-c2", "anthropic.messages.create", "span-hier-c1", 10000},
		{"span-hier-c3", "tool.Edit", "span-hier-c1", 0},
	}
	for _, s := range claudeSpans {
		span := &Span{
			ID:                s.id,
			TraceID:           "trace-hier-claude",
			ParentSpanID:      s.parent,
			TaskID:            "task-hier-001",
			AgentAssignmentID: "assign-hier-design-doc-creator",
			Name:              s.name,
			Kind:              SpanKindClient,
			Status:            SpanStatusOK,
			StartTime:         time.Now(),
			DurationMs:        1000,
			TokensIn:          s.tokens,
			TokensOut:         s.tokens / 4,
			Provider:          ProviderClaude,
			CreatedAt:         time.Now(),
		}
		backend.CreateSpan(ctx, span)
	}

	// Spans for Gemini agent
	geminiSpans := []struct {
		id     string
		name   string
		tokens int64
	}{
		{"span-hier-g1", "gemini.generate_content", 8000},
	}
	for _, s := range geminiSpans {
		span := &Span{
			ID:                s.id,
			TraceID:           "trace-hier-gemini",
			TaskID:            "task-hier-001",
			AgentAssignmentID: "assign-hier-sprint-planner",
			Name:              s.name,
			Kind:              SpanKindClient,
			Status:            SpanStatusOK,
			StartTime:         time.Now(),
			DurationMs:        2000,
			TokensIn:          s.tokens,
			TokensOut:         s.tokens / 4,
			Provider:          ProviderGemini,
			CreatedAt:         time.Now(),
		}
		backend.CreateSpan(ctx, span)
	}
}

func setupMultiTraceScenario(t *testing.T, backend *SQLiteBackend, ctx context.Context) {
	t.Helper()

	// Workspace
	ws := &Workspace{ID: "ws-multi", Name: "Multi-Trace Test", Path: "/tmp/multi", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)

	// Task
	task := &Task{
		ID:          "task-multi-001",
		WorkspaceID: "ws-multi",
		Title:       "Multi-Trace Test Task",
		Status:      TaskStatusRunning,
		CreatedAt:   time.Now(),
	}
	backend.CreateTask(ctx, task)

	// Agent 1: Multiple traces
	aa1 := &AgentAssignment{
		ID:         "assign-multi-001",
		TaskID:     "task-multi-001",
		AgentID:    "multi-trace-agent",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	backend.CreateAgentAssignment(ctx, aa1)

	// Agent 2: Single trace
	aa2 := &AgentAssignment{
		ID:         "assign-multi-002",
		TaskID:     "task-multi-001",
		AgentID:    "single-trace-agent",
		Provider:   ProviderGemini,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	backend.CreateAgentAssignment(ctx, aa2)

	// Trace 1 for agent 1
	for i := 1; i <= 3; i++ {
		span := &Span{
			ID:                fmt.Sprintf("span-multi-t1-%d", i),
			TraceID:           "trace-multi-001",
			TaskID:            "task-multi-001",
			AgentAssignmentID: "assign-multi-001",
			Name:              fmt.Sprintf("span.trace1.%d", i),
			Kind:              SpanKindClient,
			Status:            SpanStatusOK,
			StartTime:         time.Now(),
			DurationMs:        1000,
			TokensIn:          5000,
			TokensOut:         1000,
			Provider:          ProviderClaude,
			CreatedAt:         time.Now(),
		}
		backend.CreateSpan(ctx, span)
	}

	// Trace 2 for agent 1
	for i := 1; i <= 2; i++ {
		span := &Span{
			ID:                fmt.Sprintf("span-multi-t2-%d", i),
			TraceID:           "trace-multi-002",
			TaskID:            "task-multi-001",
			AgentAssignmentID: "assign-multi-001",
			Name:              fmt.Sprintf("span.trace2.%d", i),
			Kind:              SpanKindClient,
			Status:            SpanStatusOK,
			StartTime:         time.Now(),
			DurationMs:        500,
			TokensIn:          3000,
			TokensOut:         800,
			Provider:          ProviderClaude,
			CreatedAt:         time.Now(),
		}
		backend.CreateSpan(ctx, span)
	}

	// Single trace for agent 2
	span := &Span{
		ID:                "span-multi-a2",
		TraceID:           "trace-multi-003",
		TaskID:            "task-multi-001",
		AgentAssignmentID: "assign-multi-002",
		Name:              "gemini.generate",
		Kind:              SpanKindClient,
		Status:            SpanStatusOK,
		StartTime:         time.Now(),
		DurationMs:        2000,
		TokensIn:          8000,
		TokensOut:         2000,
		Provider:          ProviderGemini,
		CreatedAt:         time.Now(),
	}
	backend.CreateSpan(ctx, span)
}
