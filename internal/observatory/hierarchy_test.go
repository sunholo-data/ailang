package observatory

import (
	"context"
	"testing"
	"time"
)

// timePtr converts a time.Time value to a pointer (for EndTime fields).
func timePtr(t time.Time) *time.Time {
	return &t
}

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

// TestTimestampCorrelation tests the virtual re-parenting of ailang.* spans under exec.tool_use.
// This is the core of the M-TASK-HIERARCHY feature for proper trace visualization.
func TestTimestampCorrelation(t *testing.T) {
	// Test the helper functions directly
	t.Run("isExecutorSpan", func(t *testing.T) {
		tests := []struct {
			name string
			want bool
		}{
			{"claude.execute", true},
			{"gemini.execute", true},
			{"exec.turn", false},
			{"exec.tool_use", false},
			{"ailang.run", false},
		}
		for _, tt := range tests {
			span := &Span{Name: tt.name}
			if got := isExecutorSpan(span); got != tt.want {
				t.Errorf("isExecutorSpan(%q) = %v, want %v", tt.name, got, tt.want)
			}
		}
	})

	t.Run("isToolUseSpan", func(t *testing.T) {
		tests := []struct {
			name string
			want bool
		}{
			{"exec.tool_use", true},
			{"exec.turn", false},
			{"claude.execute", false},
		}
		for _, tt := range tests {
			span := &Span{Name: tt.name}
			if got := isToolUseSpan(span); got != tt.want {
				t.Errorf("isToolUseSpan(%q) = %v, want %v", tt.name, got, tt.want)
			}
		}
	})

	t.Run("isAilangChildSpan", func(t *testing.T) {
		tests := []struct {
			name string
			want bool
		}{
			{"ailang.run", true},
			{"ailang.check", true},
			{"ailang.exec", true},
			{"compile.parse", true},
			{"compile.typecheck", true},
			{"eval.benchmark", true},
			{"exec.turn", false},
			{"exec.tool_use", false},
			{"claude.execute", false},
			{"anthropic.generate", false},
		}
		for _, tt := range tests {
			span := &Span{Name: tt.name}
			if got := isAilangChildSpan(span); got != tt.want {
				t.Errorf("isAilangChildSpan(%q) = %v, want %v", tt.name, got, tt.want)
			}
		}
	})
}

// TestTimestampCorrelation_FullHierarchy tests the complete timestamp correlation flow.
func TestTimestampCorrelation_FullHierarchy(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	// Create workspace and task
	ws := &Workspace{ID: "ws_corr", Name: "test", Path: "/tmp/test", CreatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)
	task := &Task{ID: "task-corr", WorkspaceID: "ws_corr", Title: "Correlation Test", Status: TaskStatusRunning, CreatedAt: time.Now()}
	backend.CreateTask(ctx, task)
	agent := &AgentAssignment{ID: "aa_corr", TaskID: "task-corr", AgentID: "test", Provider: ProviderClaude, Status: AgentStatusRunning, AssignedAt: time.Now()}
	backend.CreateAgentAssignment(ctx, agent)

	baseTime := time.Date(2025, 1, 8, 10, 0, 0, 0, time.UTC)

	// Create executor span (claude.execute)
	executor := &Span{
		ID:                "executor",
		TraceID:           "trace-corr",
		AgentAssignmentID: "aa_corr",
		TaskID:            "task-corr",
		Name:              "claude.execute",
		StartTime:         baseTime,
		EndTime:           timePtr(baseTime.Add(10 * time.Second)),
		DurationMs:        10000,
	}
	backend.CreateSpan(ctx, executor)

	// Create turn span (child of executor)
	turn := &Span{
		ID:                "turn-1",
		TraceID:           "trace-corr",
		ParentSpanID:      "executor",
		AgentAssignmentID: "aa_corr",
		TaskID:            "task-corr",
		Name:              "exec.turn",
		StartTime:         baseTime.Add(100 * time.Millisecond),
		EndTime:           timePtr(baseTime.Add(5 * time.Second)),
		DurationMs:        4900,
	}
	backend.CreateSpan(ctx, turn)

	// Create tool_use span (child of executor)
	// Tool starts at T+200ms and ends at T+4s
	tool := &Span{
		ID:                "tool-1",
		TraceID:           "trace-corr",
		ParentSpanID:      "executor",
		AgentAssignmentID: "aa_corr",
		TaskID:            "task-corr",
		Name:              "exec.tool_use",
		StartTime:         baseTime.Add(200 * time.Millisecond),
		EndTime:           timePtr(baseTime.Add(4 * time.Second)),
		DurationMs:        3800,
	}
	backend.CreateSpan(ctx, tool)

	// Create ailang.run span (child of executor in DB, but should correlate to tool)
	// This span starts at T+300ms which is WITHIN tool's time window
	ailangRun := &Span{
		ID:                "ailang-run",
		TraceID:           "trace-corr",
		ParentSpanID:      "executor", // In DB, parent is executor
		AgentAssignmentID: "aa_corr",
		TaskID:            "task-corr",
		Name:              "ailang.run",
		StartTime:         baseTime.Add(300 * time.Millisecond),
		EndTime:           timePtr(baseTime.Add(3 * time.Second)),
		DurationMs:        2700,
	}
	backend.CreateSpan(ctx, ailangRun)

	// Create compile.parse span (also should correlate to tool)
	compileSpan := &Span{
		ID:                "compile-parse",
		TraceID:           "trace-corr",
		ParentSpanID:      "executor", // In DB, parent is executor
		AgentAssignmentID: "aa_corr",
		TaskID:            "task-corr",
		Name:              "compile.parse",
		StartTime:         baseTime.Add(400 * time.Millisecond),
		EndTime:           timePtr(baseTime.Add(500 * time.Millisecond)),
		DurationMs:        100,
	}
	backend.CreateSpan(ctx, compileSpan)

	// Get hierarchy - this should apply timestamp correlation
	hierarchy, err := GetTaskHierarchy(ctx, backend, "task-corr", DefaultHierarchyOptions())
	if err != nil {
		t.Fatalf("GetTaskHierarchy failed: %v", err)
	}

	// Find the executor node
	trace := hierarchy.Agents[0].Traces[0]
	var executorNode *SpanNode
	for _, node := range trace.Spans {
		if node.Span.Name == "claude.execute" {
			executorNode = node
			break
		}
	}
	if executorNode == nil {
		t.Fatal("Expected to find executor node")
	}

	// After correlation, executor should have: turn, tool (but NOT ailang.run, compile.parse)
	executorChildNames := make([]string, 0, len(executorNode.Children))
	for _, child := range executorNode.Children {
		executorChildNames = append(executorChildNames, child.Span.Name)
	}
	t.Logf("Executor children after correlation: %v", executorChildNames)

	// Find tool node
	var toolNode *SpanNode
	for _, child := range executorNode.Children {
		if child.Span.Name == "exec.tool_use" {
			toolNode = child
			break
		}
	}
	if toolNode == nil {
		t.Fatal("Expected to find tool node under executor")
	}

	// Tool should now have ailang.run and compile.parse as children (re-parented!)
	toolChildNames := make([]string, 0, len(toolNode.Children))
	for _, child := range toolNode.Children {
		toolChildNames = append(toolChildNames, child.Span.Name)
	}
	t.Logf("Tool children after correlation: %v", toolChildNames)

	// Verify ailang.run is under tool
	foundAilangRun := false
	foundCompile := false
	for _, child := range toolNode.Children {
		if child.Span.Name == "ailang.run" {
			foundAilangRun = true
		}
		if child.Span.Name == "compile.parse" {
			foundCompile = true
		}
	}

	if !foundAilangRun {
		t.Error("ailang.run should be re-parented under exec.tool_use (timestamp correlation failed)")
	}
	if !foundCompile {
		t.Error("compile.parse should be re-parented under exec.tool_use (timestamp correlation failed)")
	}

	// Verify ailang.run is NOT a direct child of executor anymore
	for _, child := range executorNode.Children {
		if child.Span.Name == "ailang.run" {
			t.Error("ailang.run should NOT be a direct child of executor after correlation")
		}
	}
}

// TestTimestampCorrelation_NoMatchingTool tests that spans outside tool windows stay under executor.
func TestTimestampCorrelation_NoMatchingTool(t *testing.T) {
	baseTime := time.Date(2025, 1, 8, 10, 0, 0, 0, time.UTC)

	// Create span index manually
	executor := &SpanNode{Span: &Span{
		ID:        "executor",
		Name:      "claude.execute",
		StartTime: baseTime,
		EndTime:   timePtr(baseTime.Add(10 * time.Second)),
	}}

	// Tool from T+1s to T+2s
	tool := &SpanNode{Span: &Span{
		ID:           "tool-1",
		Name:         "exec.tool_use",
		ParentSpanID: "executor",
		StartTime:    baseTime.Add(1 * time.Second),
		EndTime:      timePtr(baseTime.Add(2 * time.Second)),
	}}

	// ailang.run at T+5s (OUTSIDE tool window)
	ailangRun := &SpanNode{Span: &Span{
		ID:           "ailang-run",
		Name:         "ailang.run",
		ParentSpanID: "executor",
		StartTime:    baseTime.Add(5 * time.Second),
		EndTime:      timePtr(baseTime.Add(6 * time.Second)),
	}}

	// Build tree: executor -> [tool, ailangRun]
	executor.Children = []*SpanNode{tool, ailangRun}

	spanIndex := map[string]*SpanNode{
		"executor":   executor,
		"tool-1":     tool,
		"ailang-run": ailangRun,
	}

	// Apply correlation
	applyTimestampCorrelation(spanIndex)

	// ailang.run should STILL be under executor (no matching tool window)
	foundUnderExecutor := false
	for _, child := range executor.Children {
		if child.Span.Name == "ailang.run" {
			foundUnderExecutor = true
			break
		}
	}

	if !foundUnderExecutor {
		t.Error("ailang.run should remain under executor when no matching tool window exists")
	}

	// Tool should have no children
	if len(tool.Children) != 0 {
		t.Errorf("Tool should have no children, got %d", len(tool.Children))
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

// TestCrossTraceMerging tests that spans with ParentSpanID in different traces get merged.
// This tests the mergeRelatedTraces function which handles cross-trace parent-child links.
func TestCrossTraceMerging(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	// Create minimal structure
	ws := &Workspace{ID: "ws_merge", Name: "test", Path: "/tmp/test", CreatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)
	task := &Task{ID: "task-merge", WorkspaceID: "ws_merge", Title: "Merge Test", Status: TaskStatusRunning, CreatedAt: time.Now()}
	backend.CreateTask(ctx, task)
	agent := &AgentAssignment{ID: "aa_merge", TaskID: "task-merge", AgentID: "test", Provider: ProviderClaude, Status: AgentStatusRunning, AssignedAt: time.Now()}
	backend.CreateAgentAssignment(ctx, agent)

	baseTime := time.Date(2025, 1, 9, 10, 0, 0, 0, time.UTC)

	// Trace A: Coordinator creates coordinator.task.execute
	coordinatorSpan := &Span{
		ID:                "coordinator-span",
		TraceID:           "trace-A",
		ParentSpanID:      "", // Root of trace A
		AgentAssignmentID: "aa_merge",
		TaskID:            "task-merge",
		Name:              "coordinator.task.execute",
		StartTime:         baseTime,
		EndTime:           timePtr(baseTime.Add(10 * time.Second)),
		DurationMs:        10000,
	}
	backend.CreateSpan(ctx, coordinatorSpan)

	// Trace B: Claude Code creates claude.execute with ParentSpanID pointing to Trace A
	// This simulates TRACEPARENT propagation: child trace refers to parent trace's span
	claudeSpan := &Span{
		ID:                "claude-span",
		TraceID:           "trace-B",          // Different trace!
		ParentSpanID:      "coordinator-span", // Points to Trace A!
		AgentAssignmentID: "aa_merge",
		TaskID:            "task-merge",
		Name:              "claude.execute",
		StartTime:         baseTime.Add(100 * time.Millisecond),
		EndTime:           timePtr(baseTime.Add(9 * time.Second)),
		DurationMs:        8900,
	}
	backend.CreateSpan(ctx, claudeSpan)

	// Child of claude.execute (same trace B)
	turnSpan := &Span{
		ID:                "turn-span",
		TraceID:           "trace-B",
		ParentSpanID:      "claude-span",
		AgentAssignmentID: "aa_merge",
		TaskID:            "task-merge",
		Name:              "exec.turn",
		StartTime:         baseTime.Add(200 * time.Millisecond),
		EndTime:           timePtr(baseTime.Add(8 * time.Second)),
		DurationMs:        7800,
	}
	backend.CreateSpan(ctx, turnSpan)

	// Get hierarchy - this should merge traces
	hierarchy, err := GetTaskHierarchy(ctx, backend, "task-merge", DefaultHierarchyOptions())
	if err != nil {
		t.Fatalf("GetTaskHierarchy failed: %v", err)
	}

	// Should have ONE merged trace (trace B merged into trace A)
	if len(hierarchy.Agents[0].Traces) != 1 {
		t.Errorf("Expected 1 merged trace, got %d", len(hierarchy.Agents[0].Traces))
		for i, tr := range hierarchy.Agents[0].Traces {
			t.Logf("Trace %d: %s with %d spans", i, tr.TraceID, len(tr.Spans))
		}
	}

	// The merged trace should be trace-A (the parent trace)
	mergedTrace := hierarchy.Agents[0].Traces[0]
	if mergedTrace.TraceID != "trace-A" {
		t.Errorf("Expected merged trace ID to be trace-A, got %s", mergedTrace.TraceID)
	}

	// Find the coordinator node
	var coordinatorNode *SpanNode
	for _, node := range mergedTrace.Spans {
		if node.Span.Name == "coordinator.task.execute" {
			coordinatorNode = node
			break
		}
	}
	if coordinatorNode == nil {
		t.Fatal("Expected to find coordinator node")
	}

	// Coordinator should now have claude.execute as a child (cross-trace merge!)
	var foundClaude bool
	for _, child := range coordinatorNode.Children {
		if child.Span.Name == "claude.execute" {
			foundClaude = true
			// Verify exec.turn is still under claude
			var foundTurn bool
			for _, grandchild := range child.Children {
				if grandchild.Span.Name == "exec.turn" {
					foundTurn = true
					break
				}
			}
			if !foundTurn {
				t.Error("exec.turn should be under claude.execute")
			}
			break
		}
	}

	if !foundClaude {
		t.Error("claude.execute should be merged as child of coordinator.task.execute (cross-trace merge failed)")
		t.Logf("Coordinator children: %d", len(coordinatorNode.Children))
		for _, child := range coordinatorNode.Children {
			t.Logf("  - %s", child.Span.Name)
		}
	}

	// Summary should include spans from both traces
	if mergedTrace.Summary.SpanCount < 3 {
		t.Errorf("Expected at least 3 spans in merged summary, got %d", mergedTrace.Summary.SpanCount)
	}

	t.Logf("Cross-trace merge successful: coordinator.task.execute → claude.execute → exec.turn")
}
