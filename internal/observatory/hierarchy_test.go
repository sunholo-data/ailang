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
