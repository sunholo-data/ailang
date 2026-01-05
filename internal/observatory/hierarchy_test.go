package observatory

import (
	"context"
	"testing"
	"time"
)

// TestGetTaskHierarchy tests the full hierarchy query.
func TestGetTaskHierarchy(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	// Create workspace
	ws := &Workspace{
		ID:        "ws_hierarchy",
		Name:      "test",
		Path:      "/tmp/test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := backend.CreateWorkspace(ctx, ws); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create task
	task := &Task{
		ID:          "task-hier",
		WorkspaceID: "ws_hierarchy",
		Title:       "Hierarchy Test Task",
		Status:      TaskStatusRunning,
		Priority:    "medium",
		CreatedAt:   time.Now(),
	}
	if err := backend.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Create two agent assignments
	agent1 := &AgentAssignment{
		ID:         "aa_hier1",
		TaskID:     "task-hier",
		AgentID:    "sprint-planner",
		Provider:   ProviderClaude,
		Status:     AgentStatusCompleted,
		AssignedAt: time.Now(),
	}
	if err := backend.CreateAgentAssignment(ctx, agent1); err != nil {
		t.Fatalf("Failed to create agent1: %v", err)
	}

	agent2 := &AgentAssignment{
		ID:         "aa_hier2",
		TaskID:     "task-hier",
		AgentID:    "sprint-executor",
		Provider:   ProviderGemini,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	if err := backend.CreateAgentAssignment(ctx, agent2); err != nil {
		t.Fatalf("Failed to create agent2: %v", err)
	}

	// Create spans for agent1 (one trace)
	now := time.Now()
	rootSpan := &Span{
		ID:                "span-root",
		TraceID:           "trace-1",
		AgentAssignmentID: "aa_hier1",
		TaskID:            "task-hier",
		Name:              "anthropic.generate",
		Kind:              SpanKindClient,
		Status:            SpanStatusOK,
		StartTime:         now,
		DurationMs:        1000,
		TokensIn:          500,
		TokensOut:         200,
		CostUSD:           0.05,
		CreatedAt:         now,
	}
	if err := backend.CreateSpan(ctx, rootSpan); err != nil {
		t.Fatalf("Failed to create rootSpan: %v", err)
	}

	childSpan := &Span{
		ID:                "span-child",
		TraceID:           "trace-1",
		ParentSpanID:      "span-root",
		AgentAssignmentID: "aa_hier1",
		TaskID:            "task-hier",
		Name:              "claude_code.tool.write",
		Kind:              SpanKindClient,
		Status:            SpanStatusOK,
		StartTime:         now,
		DurationMs:        100,
		TokensIn:          50,
		TokensOut:         20,
		CostUSD:           0.01,
		CreatedAt:         now,
	}
	if err := backend.CreateSpan(ctx, childSpan); err != nil {
		t.Fatalf("Failed to create childSpan: %v", err)
	}

	// Create spans for agent2 (another trace)
	agent2Span := &Span{
		ID:                "span-agent2",
		TraceID:           "trace-2",
		AgentAssignmentID: "aa_hier2",
		TaskID:            "task-hier",
		Name:              "gemini.generate",
		Kind:              SpanKindClient,
		Status:            SpanStatusError,
		StatusMessage:     "Rate limit exceeded",
		StartTime:         now,
		DurationMs:        50,
		TokensIn:          100,
		TokensOut:         0,
		CostUSD:           0.005,
		CreatedAt:         now,
	}
	if err := backend.CreateSpan(ctx, agent2Span); err != nil {
		t.Fatalf("Failed to create agent2Span: %v", err)
	}

	// Get hierarchy with default options
	hierarchy, err := GetTaskHierarchy(ctx, backend, "task-hier", DefaultHierarchyOptions())
	if err != nil {
		t.Fatalf("GetTaskHierarchy failed: %v", err)
	}

	// Verify task
	if hierarchy.Task == nil {
		t.Fatal("Expected task in hierarchy")
	}
	if hierarchy.Task.ID != "task-hier" {
		t.Errorf("Expected task ID 'task-hier', got '%s'", hierarchy.Task.ID)
	}

	// Verify agents
	if len(hierarchy.Agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(hierarchy.Agents))
	}

	// Find agent1 and verify its traces
	var agent1Hier *AgentHierarchy
	for _, ah := range hierarchy.Agents {
		if ah.Agent.ID == "aa_hier1" {
			agent1Hier = ah
			break
		}
	}
	if agent1Hier == nil {
		t.Fatal("Expected to find agent1 in hierarchy")
	}
	if len(agent1Hier.Traces) != 1 {
		t.Errorf("Expected 1 trace for agent1, got %d", len(agent1Hier.Traces))
	}
	if agent1Hier.Traces[0].TraceID != "trace-1" {
		t.Errorf("Expected trace ID 'trace-1', got '%s'", agent1Hier.Traces[0].TraceID)
	}
	if len(agent1Hier.Traces[0].Spans) != 2 {
		t.Errorf("Expected 2 spans in trace, got %d", len(agent1Hier.Traces[0].Spans))
	}

	// Verify trace summary
	summary := agent1Hier.Traces[0].Summary
	if summary.SpanCount != 2 {
		t.Errorf("Expected SpanCount=2, got %d", summary.SpanCount)
	}
	if summary.TotalTokens != 770 { // 500+200+50+20
		t.Errorf("Expected TotalTokens=770, got %d", summary.TotalTokens)
	}
	if summary.ErrorCount != 0 {
		t.Errorf("Expected ErrorCount=0, got %d", summary.ErrorCount)
	}

	// Verify root span has child
	rootNode := agent1Hier.Traces[0].RootSpan
	if rootNode == nil {
		t.Fatal("Expected root span in trace hierarchy")
	}
	if len(rootNode.Children) != 1 {
		t.Errorf("Expected 1 child for root span, got %d", len(rootNode.Children))
	}

	// Find agent2 and verify error is tracked
	var agent2Hier *AgentHierarchy
	for _, ah := range hierarchy.Agents {
		if ah.Agent.ID == "aa_hier2" {
			agent2Hier = ah
			break
		}
	}
	if agent2Hier == nil {
		t.Fatal("Expected to find agent2 in hierarchy")
	}
	if len(agent2Hier.Traces) != 1 {
		t.Errorf("Expected 1 trace for agent2, got %d", len(agent2Hier.Traces))
	}
	if agent2Hier.Traces[0].Summary.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount=1 for agent2 trace, got %d", agent2Hier.Traces[0].Summary.ErrorCount)
	}
}

// TestGetTaskHierarchyDepthLimit tests the max depth option.
func TestGetTaskHierarchyDepthLimit(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	// Create workspace and task
	ws := &Workspace{ID: "ws_depth", Name: "test", Path: "/tmp/test", CreatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)
	task := &Task{ID: "task-depth", WorkspaceID: "ws_depth", Title: "Depth Test", Status: TaskStatusRunning, CreatedAt: time.Now()}
	backend.CreateTask(ctx, task)
	agent := &AgentAssignment{ID: "aa_depth", TaskID: "task-depth", AgentID: "test", Provider: ProviderClaude, Status: AgentStatusRunning, AssignedAt: time.Now()}
	backend.CreateAgentAssignment(ctx, agent)

	// Create 3-level deep span tree
	now := time.Now()
	backend.CreateSpan(ctx, &Span{ID: "level0", TraceID: "trace", AgentAssignmentID: "aa_depth", TaskID: "task-depth", Name: "root", StartTime: now, CreatedAt: now})
	backend.CreateSpan(ctx, &Span{ID: "level1", TraceID: "trace", ParentSpanID: "level0", AgentAssignmentID: "aa_depth", TaskID: "task-depth", Name: "child1", StartTime: now, CreatedAt: now})
	backend.CreateSpan(ctx, &Span{ID: "level2", TraceID: "trace", ParentSpanID: "level1", AgentAssignmentID: "aa_depth", TaskID: "task-depth", Name: "grandchild", StartTime: now, CreatedAt: now})

	// Get with depth limit of 1
	opts := HierarchyOptions{MaxDepth: 1, IncludeSpans: true}
	hierarchy, err := GetTaskHierarchy(ctx, backend, "task-depth", opts)
	if err != nil {
		t.Fatalf("GetTaskHierarchy failed: %v", err)
	}

	// Verify root has no children (depth=1 means only root level)
	rootNode := hierarchy.Agents[0].Traces[0].RootSpan
	if rootNode == nil {
		t.Fatal("Expected root span")
	}
	if len(rootNode.Children) != 0 {
		t.Errorf("Expected no children at depth=1, got %d", len(rootNode.Children))
	}
}

// TestGetTaskHierarchyNotFound tests error handling for missing task.
func TestGetTaskHierarchyNotFound(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	_, err = GetTaskHierarchy(ctx, backend, "nonexistent", DefaultHierarchyOptions())
	if err == nil {
		t.Error("Expected error for nonexistent task")
	}
}

// TestGetTaskHierarchy_SpansWithTaskIDOnly tests that spans with task_id but no agent_assignment_id
// are included in the hierarchy. This is critical for M-TASK-HIERARCHY because OTLP spans
// from Claude Code subprocess (ailang commands) have task_id extracted from cwd but
// no explicit agent_assignment_id.
func TestGetTaskHierarchy_SpansWithTaskIDOnly(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	// Create workspace and task
	ws := &Workspace{ID: "ws_taskonly", Name: "test", Path: "/tmp/test", CreatedAt: time.Now()}
	if err := backend.CreateWorkspace(ctx, ws); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	task := &Task{
		ID:          "task-12345678",
		WorkspaceID: "ws_taskonly",
		Title:       "Task ID Only Test",
		Status:      TaskStatusRunning,
		CreatedAt:   time.Now(),
	}
	if err := backend.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Create agent assignment
	agent := &AgentAssignment{
		ID:         "aa_taskonly",
		TaskID:     "task-12345678",
		AgentID:    "sprint-executor",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	if err := backend.CreateAgentAssignment(ctx, agent); err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	now := time.Now()

	// Create span WITH both task_id AND agent_assignment_id (normal case)
	spanWithBoth := &Span{
		ID:                "span-with-both",
		TraceID:           "trace-taskonly",
		AgentAssignmentID: "aa_taskonly",
		TaskID:            "task-12345678",
		Name:              "claude_code.api_request",
		Kind:              SpanKindClient,
		Status:            SpanStatusOK,
		StartTime:         now,
		DurationMs:        500,
		TokensIn:          1000,
		TokensOut:         500,
		CostUSD:           0.10,
	}
	if err := backend.CreateSpan(ctx, spanWithBoth); err != nil {
		t.Fatalf("Failed to create spanWithBoth: %v", err)
	}

	// Create span with ONLY task_id (NO agent_assignment_id)
	// This simulates OTLP spans from ailang subprocess calls
	spanTaskOnly := &Span{
		ID:                "span-task-only",
		TraceID:           "trace-taskonly",
		AgentAssignmentID: "", // NO assignment ID - simulates OTLP from subprocess
		TaskID:            "task-12345678",
		Name:              "messages.send",
		Kind:              SpanKindClient,
		Status:            SpanStatusOK,
		StartTime:         now.Add(100 * time.Millisecond),
		DurationMs:        50,
		TokensIn:          20,
		TokensOut:         10,
		CostUSD:           0.001,
	}
	if err := backend.CreateSpan(ctx, spanTaskOnly); err != nil {
		t.Fatalf("Failed to create spanTaskOnly: %v", err)
	}

	// Create another span with only task_id (different trace)
	spanTaskOnly2 := &Span{
		ID:                "span-task-only-2",
		TraceID:           "trace-taskonly-2",
		AgentAssignmentID: "", // NO assignment ID
		TaskID:            "task-12345678",
		Name:              "compile.typecheck",
		Kind:              SpanKindInternal,
		Status:            SpanStatusOK,
		StartTime:         now.Add(200 * time.Millisecond),
		DurationMs:        100,
		TokensIn:          0,
		TokensOut:         0,
		CostUSD:           0,
	}
	if err := backend.CreateSpan(ctx, spanTaskOnly2); err != nil {
		t.Fatalf("Failed to create spanTaskOnly2: %v", err)
	}

	// Get hierarchy
	hierarchy, err := GetTaskHierarchy(ctx, backend, "task-12345678", DefaultHierarchyOptions())
	if err != nil {
		t.Fatalf("GetTaskHierarchy failed: %v", err)
	}

	// Verify task
	if hierarchy.Task == nil || hierarchy.Task.ID != "task-12345678" {
		t.Fatal("Expected task in hierarchy")
	}

	// Find the agent hierarchy
	if len(hierarchy.Agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(hierarchy.Agents))
	}
	agentHier := hierarchy.Agents[0]

	// CRITICAL: Should have 2 traces (one with both spans, one with span-task-only-2)
	if len(agentHier.Traces) != 2 {
		t.Errorf("Expected 2 traces (merged spans with task_id), got %d", len(agentHier.Traces))
		for i, trace := range agentHier.Traces {
			t.Logf("  Trace[%d]: %s with %d spans", i, trace.TraceID, len(trace.Spans))
		}
	}

	// Count total spans across all traces
	totalSpans := 0
	for _, trace := range agentHier.Traces {
		totalSpans += len(trace.Spans)
	}

	// CRITICAL: Should include ALL 3 spans (2 with task_id only + 1 with both)
	if totalSpans != 3 {
		t.Errorf("Expected 3 total spans (including task_id-only spans), got %d", totalSpans)
	}

	// Verify the spans include the task_id-only ones
	foundTaskOnly := false
	foundTaskOnly2 := false
	for _, trace := range agentHier.Traces {
		for _, node := range trace.Spans {
			if node.Span.ID == "span-task-only" {
				foundTaskOnly = true
			}
			if node.Span.ID == "span-task-only-2" {
				foundTaskOnly2 = true
			}
		}
	}

	if !foundTaskOnly {
		t.Error("Missing span-task-only (span with task_id but no agent_assignment_id)")
	}
	if !foundTaskOnly2 {
		t.Error("Missing span-task-only-2 (span with task_id but no agent_assignment_id)")
	}
}

// TestGetTaskHierarchy_ChronologicalSorting tests that traces are sorted by earliest span start time.
func TestGetTaskHierarchy_ChronologicalSorting(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	// Create workspace and task
	ws := &Workspace{ID: "ws_chrono", Name: "test", Path: "/tmp/test", CreatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)

	task := &Task{
		ID:          "task-chrono",
		WorkspaceID: "ws_chrono",
		Title:       "Chronological Test",
		Status:      TaskStatusRunning,
		CreatedAt:   time.Now(),
	}
	backend.CreateTask(ctx, task)

	agent := &AgentAssignment{
		ID:         "aa_chrono",
		TaskID:     "task-chrono",
		AgentID:    "test",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	backend.CreateAgentAssignment(ctx, agent)

	// Create spans in REVERSE chronological order (newest first in database)
	baseTime := time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC)

	// Trace C - started at baseTime + 2min (LATEST)
	backend.CreateSpan(ctx, &Span{
		ID:                "span-c",
		TraceID:           "trace-c",
		AgentAssignmentID: "aa_chrono",
		TaskID:            "task-chrono",
		Name:              "third-operation",
		StartTime:         baseTime.Add(2 * time.Minute),
		DurationMs:        100,
	})

	// Trace A - started at baseTime (EARLIEST)
	backend.CreateSpan(ctx, &Span{
		ID:                "span-a",
		TraceID:           "trace-a",
		AgentAssignmentID: "aa_chrono",
		TaskID:            "task-chrono",
		Name:              "first-operation",
		StartTime:         baseTime,
		DurationMs:        100,
	})

	// Trace B - started at baseTime + 1min (MIDDLE)
	backend.CreateSpan(ctx, &Span{
		ID:                "span-b",
		TraceID:           "trace-b",
		AgentAssignmentID: "aa_chrono",
		TaskID:            "task-chrono",
		Name:              "second-operation",
		StartTime:         baseTime.Add(1 * time.Minute),
		DurationMs:        100,
	})

	// Get hierarchy
	hierarchy, err := GetTaskHierarchy(ctx, backend, "task-chrono", DefaultHierarchyOptions())
	if err != nil {
		t.Fatalf("GetTaskHierarchy failed: %v", err)
	}

	// Verify traces are sorted chronologically
	agentHier := hierarchy.Agents[0]
	if len(agentHier.Traces) != 3 {
		t.Fatalf("Expected 3 traces, got %d", len(agentHier.Traces))
	}

	// Should be in order: trace-a, trace-b, trace-c
	expectedOrder := []string{"trace-a", "trace-b", "trace-c"}
	for i, expected := range expectedOrder {
		if agentHier.Traces[i].TraceID != expected {
			t.Errorf("Trace[%d] = %s, want %s (traces should be sorted chronologically)",
				i, agentHier.Traces[i].TraceID, expected)
		}
	}
}

// TestGetTaskHierarchy_ChildSpansSortedByTime tests that child spans are sorted by start time.
func TestGetTaskHierarchy_ChildSpansSortedByTime(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	// Create workspace, task, agent
	ws := &Workspace{ID: "ws_childorder", Name: "test", Path: "/tmp/test", CreatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)
	task := &Task{ID: "task-childorder", WorkspaceID: "ws_childorder", Title: "Child Order Test", Status: TaskStatusRunning, CreatedAt: time.Now()}
	backend.CreateTask(ctx, task)
	agent := &AgentAssignment{ID: "aa_childorder", TaskID: "task-childorder", AgentID: "test", Provider: ProviderClaude, Status: AgentStatusRunning, AssignedAt: time.Now()}
	backend.CreateAgentAssignment(ctx, agent)

	baseTime := time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC)

	// Create root span
	backend.CreateSpan(ctx, &Span{
		ID:                "root",
		TraceID:           "trace-children",
		AgentAssignmentID: "aa_childorder",
		TaskID:            "task-childorder",
		Name:              "parent",
		StartTime:         baseTime,
		DurationMs:        1000,
	})

	// Create children in REVERSE order (to test sorting)
	backend.CreateSpan(ctx, &Span{
		ID:                "child-c",
		TraceID:           "trace-children",
		ParentSpanID:      "root",
		AgentAssignmentID: "aa_childorder",
		TaskID:            "task-childorder",
		Name:              "child-third",
		StartTime:         baseTime.Add(300 * time.Millisecond),
		DurationMs:        100,
	})

	backend.CreateSpan(ctx, &Span{
		ID:                "child-a",
		TraceID:           "trace-children",
		ParentSpanID:      "root",
		AgentAssignmentID: "aa_childorder",
		TaskID:            "task-childorder",
		Name:              "child-first",
		StartTime:         baseTime.Add(100 * time.Millisecond),
		DurationMs:        100,
	})

	backend.CreateSpan(ctx, &Span{
		ID:                "child-b",
		TraceID:           "trace-children",
		ParentSpanID:      "root",
		AgentAssignmentID: "aa_childorder",
		TaskID:            "task-childorder",
		Name:              "child-second",
		StartTime:         baseTime.Add(200 * time.Millisecond),
		DurationMs:        100,
	})

	// Get hierarchy
	hierarchy, err := GetTaskHierarchy(ctx, backend, "task-childorder", DefaultHierarchyOptions())
	if err != nil {
		t.Fatalf("GetTaskHierarchy failed: %v", err)
	}

	// Get root span
	rootSpan := hierarchy.Agents[0].Traces[0].RootSpan
	if rootSpan == nil {
		t.Fatal("Expected root span")
	}

	// Verify children are sorted by start time
	if len(rootSpan.Children) != 3 {
		t.Fatalf("Expected 3 children, got %d", len(rootSpan.Children))
	}

	expectedChildOrder := []string{"child-a", "child-b", "child-c"}
	for i, expected := range expectedChildOrder {
		if rootSpan.Children[i].Span.ID != expected {
			t.Errorf("Child[%d] = %s, want %s (children should be sorted by start time)",
				i, rootSpan.Children[i].Span.ID, expected)
		}
	}
}

// TestGetTaskHierarchyNoSpans tests hierarchy without spans option.
func TestGetTaskHierarchyNoSpans(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	// Create minimal structure
	ws := &Workspace{ID: "ws_nospans", Name: "test", Path: "/tmp/test", CreatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)
	task := &Task{ID: "task-nospans", WorkspaceID: "ws_nospans", Title: "No Spans Test", Status: TaskStatusRunning, CreatedAt: time.Now()}
	backend.CreateTask(ctx, task)
	agent := &AgentAssignment{ID: "aa_nospans", TaskID: "task-nospans", AgentID: "test", Provider: ProviderClaude, Status: AgentStatusRunning, AssignedAt: time.Now()}
	backend.CreateAgentAssignment(ctx, agent)

	// Create a span
	now := time.Now()
	backend.CreateSpan(ctx, &Span{ID: "span-nospans", TraceID: "trace", AgentAssignmentID: "aa_nospans", TaskID: "task-nospans", Name: "test", StartTime: now, CreatedAt: now})

	// Get with include_spans=false
	opts := HierarchyOptions{IncludeSpans: false}
	hierarchy, err := GetTaskHierarchy(ctx, backend, "task-nospans", opts)
	if err != nil {
		t.Fatalf("GetTaskHierarchy failed: %v", err)
	}

	// Verify no traces returned
	if len(hierarchy.Agents[0].Traces) != 0 {
		t.Errorf("Expected no traces when IncludeSpans=false, got %d", len(hierarchy.Agents[0].Traces))
	}
}
