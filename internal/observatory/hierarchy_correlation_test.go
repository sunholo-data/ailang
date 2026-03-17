package observatory

import (
	"context"
	"testing"
	"time"
)

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
		ID: "executor", TraceID: "trace-corr", AgentAssignmentID: "aa_corr", TaskID: "task-corr",
		Name: "claude.execute", StartTime: baseTime, EndTime: timePtr(baseTime.Add(10 * time.Second)), DurationMs: 10000,
	}
	backend.CreateSpan(ctx, executor)

	// Create turn span (child of executor)
	turn := &Span{
		ID: "turn-1", TraceID: "trace-corr", ParentSpanID: "executor", AgentAssignmentID: "aa_corr", TaskID: "task-corr",
		Name: "exec.turn", StartTime: baseTime.Add(100 * time.Millisecond), EndTime: timePtr(baseTime.Add(5 * time.Second)), DurationMs: 4900,
	}
	backend.CreateSpan(ctx, turn)

	// Create tool_use span (child of executor)
	tool := &Span{
		ID: "tool-1", TraceID: "trace-corr", ParentSpanID: "executor", AgentAssignmentID: "aa_corr", TaskID: "task-corr",
		Name: "exec.tool_use", StartTime: baseTime.Add(200 * time.Millisecond), EndTime: timePtr(baseTime.Add(4 * time.Second)), DurationMs: 3800,
	}
	backend.CreateSpan(ctx, tool)

	// Create ailang.run span (child of executor in DB, but should correlate to tool)
	ailangRun := &Span{
		ID: "ailang-run", TraceID: "trace-corr", ParentSpanID: "executor", AgentAssignmentID: "aa_corr", TaskID: "task-corr",
		Name: "ailang.run", StartTime: baseTime.Add(300 * time.Millisecond), EndTime: timePtr(baseTime.Add(3 * time.Second)), DurationMs: 2700,
	}
	backend.CreateSpan(ctx, ailangRun)

	// Create compile.parse span (also should correlate to tool)
	compileSpan := &Span{
		ID: "compile-parse", TraceID: "trace-corr", ParentSpanID: "executor", AgentAssignmentID: "aa_corr", TaskID: "task-corr",
		Name: "compile.parse", StartTime: baseTime.Add(400 * time.Millisecond), EndTime: timePtr(baseTime.Add(500 * time.Millisecond)), DurationMs: 100,
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
}

// TestTimestampCorrelation_NoMatchingTool tests that spans outside tool windows stay under executor.
func TestTimestampCorrelation_NoMatchingTool(t *testing.T) {
	baseTime := time.Date(2025, 1, 8, 10, 0, 0, 0, time.UTC)

	executor := &SpanNode{Span: &Span{
		ID: "executor", Name: "claude.execute",
		StartTime: baseTime, EndTime: timePtr(baseTime.Add(10 * time.Second)),
	}}

	tool := &SpanNode{Span: &Span{
		ID: "tool-1", Name: "exec.tool_use", ParentSpanID: "executor",
		StartTime: baseTime.Add(1 * time.Second), EndTime: timePtr(baseTime.Add(2 * time.Second)),
	}}

	ailangRun := &SpanNode{Span: &Span{
		ID: "ailang-run", Name: "ailang.run", ParentSpanID: "executor",
		StartTime: baseTime.Add(5 * time.Second), EndTime: timePtr(baseTime.Add(6 * time.Second)),
	}}

	executor.Children = []*SpanNode{tool, ailangRun}

	spanIndex := map[string]*SpanNode{
		"executor":   executor,
		"tool-1":     tool,
		"ailang-run": ailangRun,
	}

	applyTimestampCorrelation(spanIndex)

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

	if len(tool.Children) != 0 {
		t.Errorf("Tool should have no children, got %d", len(tool.Children))
	}
}

// TestCrossTraceMerging tests that spans with ParentSpanID in different traces get merged.
func TestCrossTraceMerging(t *testing.T) {
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	defer backend.Close()
	ctx := context.Background()

	ws := &Workspace{ID: "ws_merge", Name: "test", Path: "/tmp/test", CreatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)
	task := &Task{ID: "task-merge", WorkspaceID: "ws_merge", Title: "Merge Test", Status: TaskStatusRunning, CreatedAt: time.Now()}
	backend.CreateTask(ctx, task)
	agent := &AgentAssignment{ID: "aa_merge", TaskID: "task-merge", AgentID: "test", Provider: ProviderClaude, Status: AgentStatusRunning, AssignedAt: time.Now()}
	backend.CreateAgentAssignment(ctx, agent)

	baseTime := time.Date(2025, 1, 9, 10, 0, 0, 0, time.UTC)

	coordinatorSpan := &Span{
		ID: "coordinator-span", TraceID: "trace-A", AgentAssignmentID: "aa_merge", TaskID: "task-merge",
		Name: "coordinator.task.execute", StartTime: baseTime, EndTime: timePtr(baseTime.Add(10 * time.Second)), DurationMs: 10000,
	}
	backend.CreateSpan(ctx, coordinatorSpan)

	claudeSpan := &Span{
		ID: "claude-span", TraceID: "trace-B", ParentSpanID: "coordinator-span",
		AgentAssignmentID: "aa_merge", TaskID: "task-merge",
		Name: "claude.execute", StartTime: baseTime.Add(100 * time.Millisecond),
		EndTime: timePtr(baseTime.Add(9 * time.Second)), DurationMs: 8900,
	}
	backend.CreateSpan(ctx, claudeSpan)

	turnSpan := &Span{
		ID: "turn-span", TraceID: "trace-B", ParentSpanID: "claude-span",
		AgentAssignmentID: "aa_merge", TaskID: "task-merge",
		Name: "exec.turn", StartTime: baseTime.Add(200 * time.Millisecond),
		EndTime: timePtr(baseTime.Add(8 * time.Second)), DurationMs: 7800,
	}
	backend.CreateSpan(ctx, turnSpan)

	hierarchy, err := GetTaskHierarchy(ctx, backend, "task-merge", DefaultHierarchyOptions())
	if err != nil {
		t.Fatalf("GetTaskHierarchy failed: %v", err)
	}

	if len(hierarchy.Agents[0].Traces) != 1 {
		t.Errorf("Expected 1 merged trace, got %d", len(hierarchy.Agents[0].Traces))
	}

	mergedTrace := hierarchy.Agents[0].Traces[0]
	if mergedTrace.TraceID != "trace-A" {
		t.Errorf("Expected merged trace ID to be trace-A, got %s", mergedTrace.TraceID)
	}

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

	var foundClaude bool
	for _, child := range coordinatorNode.Children {
		if child.Span.Name == "claude.execute" {
			foundClaude = true
			break
		}
	}

	if !foundClaude {
		t.Error("claude.execute should be merged as child of coordinator.task.execute (cross-trace merge failed)")
	}

	if mergedTrace.Summary.SpanCount < 3 {
		t.Errorf("Expected at least 3 spans in merged summary, got %d", mergedTrace.Summary.SpanCount)
	}
}
