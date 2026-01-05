package observatory

import (
	"context"
	"testing"
	"time"
)

// TestAggregationUpdates tests that task and assignment aggregates update when spans are created.
func TestAggregationUpdates(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	// Create workspace
	ws := &Workspace{
		ID:        "ws_test",
		Name:      "test",
		Path:      "/tmp/test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := backend.CreateWorkspace(ctx, ws); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create task with zero aggregates
	task := &Task{
		ID:          "task-agg",
		WorkspaceID: "ws_test",
		Title:       "Aggregation Test",
		Status:      TaskStatusRunning,
		Priority:    "medium",
		CreatedAt:   time.Now(),
	}
	if err := backend.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Create agent assignment with zero aggregates
	assignment := &AgentAssignment{
		ID:         "aa_agg",
		TaskID:     "task-agg",
		AgentID:    "test-agent",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	if err := backend.CreateAgentAssignment(ctx, assignment); err != nil {
		t.Fatalf("Failed to create assignment: %v", err)
	}

	// Create first span linked to task and assignment
	now := time.Now()
	span1 := &Span{
		ID:                "span-1",
		TraceID:           "trace-1",
		TaskID:            "task-agg",
		AgentAssignmentID: "aa_agg",
		Name:              "anthropic.generate",
		Kind:              SpanKindClient,
		Status:            SpanStatusOK,
		StartTime:         now,
		EndTime:           &now,
		DurationMs:        100,
		TokensIn:          1000,
		TokensOut:         500,
		CostUSD:           0.05,
		Provider:          ProviderClaude,
		CreatedAt:         now,
	}
	if err := backend.CreateSpan(ctx, span1); err != nil {
		t.Fatalf("Failed to create span1: %v", err)
	}

	// Verify task aggregates updated
	updatedTask, err := backend.GetTask(ctx, "task-agg")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	if updatedTask.SpanCount != 1 {
		t.Errorf("Expected SpanCount=1, got %d", updatedTask.SpanCount)
	}
	if updatedTask.TotalTokensIn != 1000 {
		t.Errorf("Expected TotalTokensIn=1000, got %d", updatedTask.TotalTokensIn)
	}
	if updatedTask.TotalTokensOut != 500 {
		t.Errorf("Expected TotalTokensOut=500, got %d", updatedTask.TotalTokensOut)
	}
	if updatedTask.TotalDurationMs != 100 {
		t.Errorf("Expected TotalDurationMs=100, got %d", updatedTask.TotalDurationMs)
	}
	if updatedTask.TotalCostUSD != 0.05 {
		t.Errorf("Expected TotalCostUSD=0.05, got %f", updatedTask.TotalCostUSD)
	}

	// Verify assignment aggregates updated
	updatedAssignment, err := backend.GetAgentAssignment(ctx, "aa_agg")
	if err != nil {
		t.Fatalf("Failed to get assignment: %v", err)
	}
	if updatedAssignment.TokensIn != 1000 {
		t.Errorf("Expected assignment TokensIn=1000, got %d", updatedAssignment.TokensIn)
	}
	if updatedAssignment.TokensOut != 500 {
		t.Errorf("Expected assignment TokensOut=500, got %d", updatedAssignment.TokensOut)
	}
	if updatedAssignment.DurationMs != 100 {
		t.Errorf("Expected assignment DurationMs=100, got %d", updatedAssignment.DurationMs)
	}
	if updatedAssignment.CostUSD != 0.05 {
		t.Errorf("Expected assignment CostUSD=0.05, got %f", updatedAssignment.CostUSD)
	}

	// Create second span with error status
	span2 := &Span{
		ID:                "span-2",
		TraceID:           "trace-1",
		TaskID:            "task-agg",
		AgentAssignmentID: "aa_agg",
		Name:              "claude_code.tool.write",
		Kind:              SpanKindClient,
		Status:            SpanStatusError,
		StatusMessage:     "Permission denied",
		StartTime:         now,
		EndTime:           &now,
		DurationMs:        50,
		TokensIn:          200,
		TokensOut:         100,
		CostUSD:           0.01,
		Provider:          ProviderClaude,
		CreatedAt:         now,
	}
	if err := backend.CreateSpan(ctx, span2); err != nil {
		t.Fatalf("Failed to create span2: %v", err)
	}

	// Verify aggregates accumulated
	updatedTask, _ = backend.GetTask(ctx, "task-agg")
	if updatedTask.SpanCount != 2 {
		t.Errorf("Expected SpanCount=2, got %d", updatedTask.SpanCount)
	}
	if updatedTask.TotalTokensIn != 1200 {
		t.Errorf("Expected TotalTokensIn=1200, got %d", updatedTask.TotalTokensIn)
	}
	if updatedTask.TotalDurationMs != 150 {
		t.Errorf("Expected TotalDurationMs=150, got %d", updatedTask.TotalDurationMs)
	}
	if updatedTask.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount=1, got %d", updatedTask.ErrorCount)
	}

	// Verify tool call increment for assignment
	updatedAssignment, _ = backend.GetAgentAssignment(ctx, "aa_agg")
	if updatedAssignment.ToolCalls != 1 {
		t.Errorf("Expected ToolCalls=1, got %d", updatedAssignment.ToolCalls)
	}
}

// TestIsToolCallSpan tests the tool call detection logic.
func TestIsToolCallSpan(t *testing.T) {
	tests := []struct {
		name     string
		spanName string
		expected bool
	}{
		{"Claude tool call", "claude_code.tool.write", true},
		{"Gemini tool call", "gemini.tool.read", true},
		{"Generic tool call", "tool.execute", true},
		{"API call", "anthropic.generate", false},
		{"Generic span", "compile.typecheck", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isToolCallSpan(tt.spanName)
			if got != tt.expected {
				t.Errorf("isToolCallSpan(%q) = %v, want %v", tt.spanName, got, tt.expected)
			}
		})
	}
}

// TestRecalculateAggregates tests the backfill/recalculation functions.
func TestRecalculateAggregates(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()
	db := backend.store.DB()

	// Create workspace
	ws := &Workspace{
		ID:        "ws_recalc",
		Name:      "test",
		Path:      "/tmp/test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	backend.CreateWorkspace(ctx, ws)

	// Create task with manual wrong values (simulating drift)
	task := &Task{
		ID:              "task-recalc",
		WorkspaceID:     "ws_recalc",
		Title:           "Recalculation Test",
		Status:          TaskStatusRunning,
		Priority:        "medium",
		CreatedAt:       time.Now(),
		SpanCount:       999, // Wrong value
		TotalTokensIn:   999, // Wrong value
		TotalTokensOut:  999, // Wrong value
		TotalDurationMs: 999, // Wrong value
		TotalCostUSD:    999, // Wrong value
		ErrorCount:      999, // Wrong value
	}
	backend.CreateTask(ctx, task)

	// Create spans without aggregation (bypass the transactional path)
	now := time.Now()
	backend.store.CreateSpan(&Span{
		ID:        "span-r1",
		TraceID:   "trace-r1",
		TaskID:    "task-recalc",
		Name:      "test.span1",
		Kind:      SpanKindClient,
		Status:    SpanStatusOK,
		StartTime: now,
		TokensIn:  100,
		TokensOut: 50,
		CostUSD:   0.01,
		CreatedAt: now,
	})
	backend.store.CreateSpan(&Span{
		ID:        "span-r2",
		TraceID:   "trace-r1",
		TaskID:    "task-recalc",
		Name:      "test.span2",
		Kind:      SpanKindClient,
		Status:    SpanStatusError,
		StartTime: now,
		TokensIn:  200,
		TokensOut: 100,
		CostUSD:   0.02,
		CreatedAt: now,
	})

	// Recalculate aggregates
	err = RecalculateTaskAggregates(ctx, db, "task-recalc")
	if err != nil {
		t.Fatalf("RecalculateTaskAggregates failed: %v", err)
	}

	// Verify correct values
	updatedTask, _ := backend.GetTask(ctx, "task-recalc")
	if updatedTask.SpanCount != 2 {
		t.Errorf("Expected SpanCount=2, got %d", updatedTask.SpanCount)
	}
	if updatedTask.TotalTokensIn != 300 {
		t.Errorf("Expected TotalTokensIn=300, got %d", updatedTask.TotalTokensIn)
	}
	if updatedTask.TotalTokensOut != 150 {
		t.Errorf("Expected TotalTokensOut=150, got %d", updatedTask.TotalTokensOut)
	}
	if updatedTask.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount=1, got %d", updatedTask.ErrorCount)
	}
}

// TestNoAggregationWithoutTaskID tests that spans without task_id don't trigger aggregation.
func TestNoAggregationWithoutTaskID(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	// Create span without task_id
	now := time.Now()
	span := &Span{
		ID:        "span-no-task",
		TraceID:   "trace-no-task",
		Name:      "standalone.operation",
		Kind:      SpanKindClient,
		Status:    SpanStatusOK,
		StartTime: now,
		TokensIn:  500,
		TokensOut: 250,
		CostUSD:   0.10,
		CreatedAt: now,
	}
	if err := backend.CreateSpan(ctx, span); err != nil {
		t.Fatalf("Failed to create span: %v", err)
	}

	// Verify span was created
	retrieved, err := backend.GetSpan(ctx, "span-no-task")
	if err != nil {
		t.Fatalf("Failed to retrieve span: %v", err)
	}
	if retrieved.TokensIn != 500 {
		t.Errorf("Expected TokensIn=500, got %d", retrieved.TokensIn)
	}
}
