// Package observatory provides integration tests for M-TASK-HIERARCHY feature.
// Tests progress from simple entity CRUD to full end-to-end workflows.
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
// Level 1: Basic Entity CRUD and Linking
// ============================================================================

func TestIntegration_Level1_EntityCRUD(t *testing.T) {
	backend := setupIntegrationBackend(t)
	defer backend.Close()
	ctx := context.Background()

	t.Run("create_workspace", func(t *testing.T) {
		ws := &Workspace{
			ID:        "ws-test-001",
			Name:      "AILANG Test Project",
			Path:      "/Users/test/dev/ailang",
			GitRemote: "github.com/sunholo-data/ailang",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := backend.CreateWorkspace(ctx, ws); err != nil {
			t.Fatalf("CreateWorkspace failed: %v", err)
		}

		// Verify it exists
		got, err := backend.GetWorkspace(ctx, "ws-test-001")
		if err != nil {
			t.Fatalf("GetWorkspace failed: %v", err)
		}
		if got.Name != "AILANG Test Project" {
			t.Errorf("Name mismatch: got %q, want %q", got.Name, "AILANG Test Project")
		}
	})

	t.Run("create_task_with_workspace", func(t *testing.T) {
		task := &Task{
			ID:          "task-test-001",
			WorkspaceID: "ws-test-001",
			Title:       "Implement semantic caching",
			Description: "Add caching layer for LLM responses",
			SourceType:  TaskSourceGitHub,
			SourceRef:   "issue/123",
			Status:      TaskStatusPending,
			Priority:    "high",
			CreatedAt:   time.Now(),
		}
		if err := backend.CreateTask(ctx, task); err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}

		got, err := backend.GetTask(ctx, "task-test-001")
		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}
		if got.WorkspaceID != "ws-test-001" {
			t.Errorf("WorkspaceID mismatch: got %q, want %q", got.WorkspaceID, "ws-test-001")
		}
	})

	t.Run("create_agent_assignment_with_task", func(t *testing.T) {
		aa := &AgentAssignment{
			ID:         "assign-test-001",
			TaskID:     "task-test-001",
			AgentID:    "design-doc-creator",
			Provider:   ProviderClaude,
			Status:     AgentStatusRunning,
			AssignedAt: time.Now(),
		}
		if err := backend.CreateAgentAssignment(ctx, aa); err != nil {
			t.Fatalf("CreateAgentAssignment failed: %v", err)
		}

		assignments, err := backend.ListAgentAssignments(ctx, "task-test-001")
		if err != nil {
			t.Fatalf("ListAgentAssignments failed: %v", err)
		}
		if len(assignments) != 1 {
			t.Errorf("Expected 1 assignment, got %d", len(assignments))
		}
		if assignments[0].AgentID != "design-doc-creator" {
			t.Errorf("AgentID mismatch: got %q, want %q", assignments[0].AgentID, "design-doc-creator")
		}
	})

	t.Run("create_span_with_task_and_assignment", func(t *testing.T) {
		span := &Span{
			ID:                "span-test-001",
			TraceID:           "trace-test-001",
			TaskID:            "task-test-001",
			AgentAssignmentID: "assign-test-001",
			Name:              "anthropic.messages.create",
			Kind:              SpanKindClient,
			Status:            SpanStatusOK,
			StartTime:         time.Now().Add(-time.Second),
			DurationMs:        1500,
			TokensIn:          5000,
			TokensOut:         1200,
			CostUSD:           0.045,
			Model:             "claude-sonnet-4-5-20250514",
			Provider:          ProviderClaude,
			Attributes:        map[string]any{"gen_ai.system": "anthropic"},
			CreatedAt:         time.Now(),
		}
		if err := backend.CreateSpan(ctx, span); err != nil {
			t.Fatalf("CreateSpan failed: %v", err)
		}

		got, err := backend.GetSpan(ctx, "span-test-001")
		if err != nil {
			t.Fatalf("GetSpan failed: %v", err)
		}
		if got.TaskID != "task-test-001" {
			t.Errorf("TaskID mismatch: got %q, want %q", got.TaskID, "task-test-001")
		}
		if got.AgentAssignmentID != "assign-test-001" {
			t.Errorf("AgentAssignmentID mismatch: got %q, want %q", got.AgentAssignmentID, "assign-test-001")
		}
	})

	t.Run("query_spans_by_task_id", func(t *testing.T) {
		spans, err := backend.ListSpans(ctx, SpanListOptions{TaskID: "task-test-001"})
		if err != nil {
			t.Fatalf("ListSpans failed: %v", err)
		}
		if len(spans) != 1 {
			t.Errorf("Expected 1 span for task, got %d", len(spans))
		}
	})

	t.Run("query_spans_by_assignment_id", func(t *testing.T) {
		spans, err := backend.ListSpans(ctx, SpanListOptions{AgentAssignmentID: "assign-test-001"})
		if err != nil {
			t.Fatalf("ListSpans failed: %v", err)
		}
		if len(spans) != 1 {
			t.Errorf("Expected 1 span for assignment, got %d", len(spans))
		}
	})
}

// ============================================================================
// Level 2: Entity Linking via UpdateSpanLinks
// ============================================================================

func TestIntegration_Level2_SpanLinking(t *testing.T) {
	backend := setupIntegrationBackend(t)
	defer backend.Close()
	ctx := context.Background()

	// Setup: Create unlinked entities
	setupUnlinkedEntities(t, backend, ctx)

	t.Run("link_span_to_task_and_assignment", func(t *testing.T) {
		// Span starts unlinked
		span, err := backend.GetSpan(ctx, "span-unlinked-001")
		if err != nil {
			t.Fatalf("GetSpan failed: %v", err)
		}
		if span.TaskID != "" {
			t.Errorf("Span should start unlinked, got TaskID=%q", span.TaskID)
		}

		// Link it
		err = backend.UpdateSpanLinks(ctx, "span-unlinked-001", "task-unlinked-001", "assign-unlinked-001")
		if err != nil {
			t.Fatalf("UpdateSpanLinks failed: %v", err)
		}

		// Verify linkage
		span, err = backend.GetSpan(ctx, "span-unlinked-001")
		if err != nil {
			t.Fatalf("GetSpan after link failed: %v", err)
		}
		if span.TaskID != "task-unlinked-001" {
			t.Errorf("TaskID mismatch after link: got %q, want %q", span.TaskID, "task-unlinked-001")
		}
		if span.AgentAssignmentID != "assign-unlinked-001" {
			t.Errorf("AgentAssignmentID mismatch after link: got %q, want %q", span.AgentAssignmentID, "assign-unlinked-001")
		}
	})

	t.Run("partial_link_task_only", func(t *testing.T) {
		// Create another unlinked span
		span := &Span{
			ID:        "span-unlinked-002",
			TraceID:   "trace-unlinked-002",
			Name:      "partial.link.test",
			Kind:      SpanKindInternal,
			Status:    SpanStatusOK,
			StartTime: time.Now(),
			CreatedAt: time.Now(),
		}
		backend.CreateSpan(ctx, span)

		// Link only to task (no assignment)
		err := backend.UpdateSpanLinks(ctx, "span-unlinked-002", "task-unlinked-001", "")
		if err != nil {
			t.Fatalf("UpdateSpanLinks (task only) failed: %v", err)
		}

		got, _ := backend.GetSpan(ctx, "span-unlinked-002")
		if got.TaskID != "task-unlinked-001" {
			t.Errorf("TaskID should be set, got %q", got.TaskID)
		}
		if got.AgentAssignmentID != "" {
			t.Errorf("AgentAssignmentID should be empty, got %q", got.AgentAssignmentID)
		}
	})
}

// ============================================================================
// Level 3: Full Hierarchy Query
// ============================================================================

func TestIntegration_Level3_Hierarchy(t *testing.T) {
	backend := setupIntegrationBackend(t)
	defer backend.Close()
	ctx := context.Background()

	// Setup: Create a rich hierarchy
	setupRichHierarchy(t, backend, ctx)

	t.Run("get_task_hierarchy_with_agents_and_spans", func(t *testing.T) {
		hierarchy, err := GetTaskHierarchy(ctx, backend, "task-hier-001", DefaultHierarchyOptions())
		if err != nil {
			t.Fatalf("GetTaskHierarchy failed: %v", err)
		}

		// Verify task
		if hierarchy.Task.ID != "task-hier-001" {
			t.Errorf("Task ID mismatch: got %q", hierarchy.Task.ID)
		}
		if hierarchy.Task.Title != "Full Hierarchy Test Task" {
			t.Errorf("Task title mismatch: got %q", hierarchy.Task.Title)
		}

		// Verify agents
		if len(hierarchy.Agents) != 2 {
			t.Errorf("Expected 2 agents, got %d", len(hierarchy.Agents))
		}

		// Find claude agent and verify traces
		var claudeAgent *AgentHierarchy
		for _, a := range hierarchy.Agents {
			if a.Agent.Provider == ProviderClaude {
				claudeAgent = a
				break
			}
		}
		if claudeAgent == nil {
			t.Fatal("Could not find Claude agent in hierarchy")
		}

		if len(claudeAgent.Traces) == 0 {
			t.Error("Claude agent should have traces")
		}

		// Verify span count across traces
		totalSpans := 0
		for _, trace := range claudeAgent.Traces {
			totalSpans += trace.Summary.SpanCount
		}
		if totalSpans < 2 {
			t.Errorf("Expected at least 2 spans for Claude agent, got %d", totalSpans)
		}
	})

	t.Run("hierarchy_depth_limit", func(t *testing.T) {
		opts := HierarchyOptions{
			MaxDepth:     1,
			IncludeSpans: true,
		}
		hierarchy, err := GetTaskHierarchy(ctx, backend, "task-hier-001", opts)
		if err != nil {
			t.Fatalf("GetTaskHierarchy with depth limit failed: %v", err)
		}

		// Should have agents but spans should have no children
		for _, agent := range hierarchy.Agents {
			for _, trace := range agent.Traces {
				for _, spanNode := range trace.Spans {
					if len(spanNode.Children) > 0 {
						t.Error("Span children should be truncated at depth 1")
					}
				}
			}
		}
	})

	t.Run("hierarchy_without_spans", func(t *testing.T) {
		opts := HierarchyOptions{
			IncludeSpans: false,
		}
		hierarchy, err := GetTaskHierarchy(ctx, backend, "task-hier-001", opts)
		if err != nil {
			t.Fatalf("GetTaskHierarchy without spans failed: %v", err)
		}

		// Should have agents but no traces
		for _, agent := range hierarchy.Agents {
			if len(agent.Traces) > 0 {
				t.Error("Traces should be empty when IncludeSpans=false")
			}
		}
	})
}

// ============================================================================
// Level 4: Aggregation Updates
// ============================================================================

func TestIntegration_Level4_Aggregations(t *testing.T) {
	backend := setupIntegrationBackend(t)
	defer backend.Close()
	ctx := context.Background()

	// Setup: Create task and assignment with zero aggregates
	ws := &Workspace{ID: "ws-agg", Name: "Agg Test", Path: "/tmp/agg", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	backend.CreateWorkspace(ctx, ws)

	task := &Task{
		ID:          "task-agg-001",
		WorkspaceID: "ws-agg",
		Title:       "Aggregation Test Task",
		Status:      TaskStatusRunning,
		CreatedAt:   time.Now(),
	}
	backend.CreateTask(ctx, task)

	aa := &AgentAssignment{
		ID:         "assign-agg-001",
		TaskID:     "task-agg-001",
		AgentID:    "test-agent",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	backend.CreateAgentAssignment(ctx, aa)

	t.Run("span_with_task_updates_aggregates", func(t *testing.T) {
		// Create span linked to task
		span := &Span{
			ID:                "span-agg-001",
			TraceID:           "trace-agg-001",
			TaskID:            "task-agg-001",
			AgentAssignmentID: "assign-agg-001",
			Name:              "api.request.1",
			Kind:              SpanKindClient,
			Status:            SpanStatusOK,
			StartTime:         time.Now().Add(-2 * time.Second),
			DurationMs:        2000,
			TokensIn:          10000,
			TokensOut:         2500,
			CostUSD:           0.10,
			Provider:          ProviderClaude,
			CreatedAt:         time.Now(),
		}
		if err := backend.CreateSpan(ctx, span); err != nil {
			t.Fatalf("CreateSpan failed: %v", err)
		}

		// Verify task aggregates updated
		task, err := backend.GetTask(ctx, "task-agg-001")
		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}

		if task.SpanCount < 1 {
			t.Errorf("Task span_count should be >= 1, got %d", task.SpanCount)
		}
		if task.TotalTokensIn < 10000 {
			t.Errorf("Task total_tokens_in should be >= 10000, got %d", task.TotalTokensIn)
		}
		if task.TotalTokensOut < 2500 {
			t.Errorf("Task total_tokens_out should be >= 2500, got %d", task.TotalTokensOut)
		}
		if task.TotalCostUSD < 0.10 {
			t.Errorf("Task total_cost_usd should be >= 0.10, got %.4f", task.TotalCostUSD)
		}
	})

	t.Run("multiple_spans_accumulate", func(t *testing.T) {
		// Add more spans
		for i := 2; i <= 5; i++ {
			span := &Span{
				ID:                fmt.Sprintf("span-agg-%03d", i),
				TraceID:           "trace-agg-001",
				TaskID:            "task-agg-001",
				AgentAssignmentID: "assign-agg-001",
				Name:              fmt.Sprintf("api.request.%d", i),
				Kind:              SpanKindClient,
				Status:            SpanStatusOK,
				StartTime:         time.Now(),
				DurationMs:        1000,
				TokensIn:          5000,
				TokensOut:         1000,
				CostUSD:           0.05,
				Provider:          ProviderClaude,
				CreatedAt:         time.Now(),
			}
			backend.CreateSpan(ctx, span)
		}

		task, _ := backend.GetTask(ctx, "task-agg-001")

		// Should have 5 spans total
		if task.SpanCount < 5 {
			t.Errorf("Task span_count should be >= 5, got %d", task.SpanCount)
		}

		// Should have accumulated tokens: 10000 + 4*5000 = 30000
		expectedTokensIn := int64(10000 + 4*5000)
		if task.TotalTokensIn < expectedTokensIn {
			t.Errorf("Task total_tokens_in should be >= %d, got %d", expectedTokensIn, task.TotalTokensIn)
		}
	})

	t.Run("error_spans_increment_error_count", func(t *testing.T) {
		// Create error span
		errorSpan := &Span{
			ID:                "span-agg-error",
			TraceID:           "trace-agg-001",
			TaskID:            "task-agg-001",
			AgentAssignmentID: "assign-agg-001",
			Name:              "api.request.error",
			Kind:              SpanKindClient,
			Status:            SpanStatusError,
			StatusMessage:     "Rate limit exceeded",
			StartTime:         time.Now(),
			DurationMs:        500,
			TokensIn:          0,
			TokensOut:         0,
			CostUSD:           0,
			Provider:          ProviderClaude,
			CreatedAt:         time.Now(),
		}
		backend.CreateSpan(ctx, errorSpan)

		task, _ := backend.GetTask(ctx, "task-agg-001")
		if task.ErrorCount < 1 {
			t.Errorf("Task error_count should be >= 1, got %d", task.ErrorCount)
		}
	})
}

// ============================================================================
// Level 5: Time-Based Span Filtering (for Backfill)
// ============================================================================

func TestIntegration_Level5_TimeBasedFiltering(t *testing.T) {
	backend := setupIntegrationBackend(t)
	defer backend.Close()
	ctx := context.Background()

	// Setup: Create spans at different times
	baseTime := time.Now().Add(-time.Hour)

	for i := 0; i < 10; i++ {
		span := &Span{
			ID:         fmt.Sprintf("span-time-%03d", i),
			TraceID:    "trace-time-001",
			Name:       fmt.Sprintf("timed.span.%d", i),
			Kind:       SpanKindClient,
			Status:     SpanStatusOK,
			StartTime:  baseTime.Add(time.Duration(i*5) * time.Minute),
			DurationMs: 1000,
			CreatedAt:  time.Now(),
		}
		backend.CreateSpan(ctx, span)
	}

	t.Run("filter_spans_by_time_window", func(t *testing.T) {
		// Query spans in the first 15 minutes (inclusive on both ends)
		// Spans at 0, 5, 10, 15 minutes match start_time >= baseTime AND start_time <= baseTime+15min
		spans, err := backend.ListSpans(ctx, SpanListOptions{
			StartAfter:  baseTime,
			StartBefore: baseTime.Add(15 * time.Minute),
		})
		if err != nil {
			t.Fatalf("ListSpans with time filter failed: %v", err)
		}

		// Should get spans 0, 1, 2, 3 (at 0, 5, 10, 15 minutes) - inclusive bounds
		if len(spans) != 4 {
			t.Errorf("Expected 4 spans in first 15 minutes (inclusive), got %d", len(spans))
		}
	})

	t.Run("filter_spans_middle_window", func(t *testing.T) {
		// Query spans between 20 and 40 minutes (inclusive on both ends)
		// Spans at 20, 25, 30, 35, 40 minutes match
		spans, err := backend.ListSpans(ctx, SpanListOptions{
			StartAfter:  baseTime.Add(20 * time.Minute),
			StartBefore: baseTime.Add(40 * time.Minute),
		})
		if err != nil {
			t.Fatalf("ListSpans with middle window failed: %v", err)
		}

		// Should get spans 4, 5, 6, 7, 8 (at 20, 25, 30, 35, 40 minutes) - inclusive bounds
		if len(spans) != 5 {
			t.Errorf("Expected 5 spans in middle window (inclusive), got %d", len(spans))
		}
	})
}

// ============================================================================
// Level 6: Provider Matching (for Backfill)
// ============================================================================

func TestIntegration_Level6_ProviderMatching(t *testing.T) {
	backend := setupIntegrationBackend(t)
	defer backend.Close()
	ctx := context.Background()

	// Create spans with different providers
	providers := []struct {
		id       string
		name     string
		provider Provider
		attrs    map[string]any
	}{
		{"span-claude-001", "anthropic.messages.create", ProviderClaude, map[string]any{"gen_ai.system": "anthropic"}},
		{"span-gemini-001", "gemini.generate_content", ProviderGemini, map[string]any{"gen_ai.system": "google"}},
		{"span-ollama-001", "ollama.generate", ProviderOllama, map[string]any{"gen_ai.system": "ollama"}},
		{"span-claude-002", "claude.tool.call", ProviderClaude, nil},
		{"span-internal-001", "internal.processing", "", nil},
	}

	for _, p := range providers {
		span := &Span{
			ID:                 p.id,
			TraceID:            "trace-provider-001",
			Name:               p.name,
			Kind:               SpanKindClient,
			Status:             SpanStatusOK,
			StartTime:          time.Now(),
			DurationMs:         1000,
			Provider:           p.provider,
			Attributes:         p.attrs,
			ResourceAttributes: map[string]any{"service.name": "test-service"},
			CreatedAt:          time.Now(),
		}
		backend.CreateSpan(ctx, span)
	}

	t.Run("count_spans_by_provider", func(t *testing.T) {
		allSpans, _ := backend.ListSpans(ctx, SpanListOptions{})

		claudeCount := 0
		geminiCount := 0
		for _, s := range allSpans {
			switch s.Provider {
			case ProviderClaude:
				claudeCount++
			case ProviderGemini:
				geminiCount++
			}
		}

		if claudeCount != 2 {
			t.Errorf("Expected 2 Claude spans, got %d", claudeCount)
		}
		if geminiCount != 1 {
			t.Errorf("Expected 1 Gemini span, got %d", geminiCount)
		}
	})
}

// ============================================================================
// Level 7: Multi-Trace Hierarchy
// ============================================================================

func TestIntegration_Level7_MultiTraceHierarchy(t *testing.T) {
	backend := setupIntegrationBackend(t)
	defer backend.Close()
	ctx := context.Background()

	// Setup: Task with multiple agents, each generating multiple traces
	setupMultiTraceScenario(t, backend, ctx)

	t.Run("hierarchy_groups_spans_by_trace", func(t *testing.T) {
		hierarchy, err := GetTaskHierarchy(ctx, backend, "task-multi-001", DefaultHierarchyOptions())
		if err != nil {
			t.Fatalf("GetTaskHierarchy failed: %v", err)
		}

		// Should have 2 agents
		if len(hierarchy.Agents) != 2 {
			t.Errorf("Expected 2 agents, got %d", len(hierarchy.Agents))
		}

		// Find agent with multiple traces
		for _, agent := range hierarchy.Agents {
			if agent.Agent.AgentID == "multi-trace-agent" {
				if len(agent.Traces) < 2 {
					t.Errorf("Expected at least 2 traces for multi-trace-agent, got %d", len(agent.Traces))
				}
				return
			}
		}
		t.Error("Could not find multi-trace-agent")
	})

	t.Run("trace_summaries_accurate", func(t *testing.T) {
		hierarchy, _ := GetTaskHierarchy(ctx, backend, "task-multi-001", DefaultHierarchyOptions())

		for _, agent := range hierarchy.Agents {
			for _, trace := range agent.Traces {
				if trace.Summary.SpanCount == 0 {
					t.Errorf("Trace %s has 0 span count", trace.TraceID)
				}
				if trace.Summary.TotalTokens == 0 && agent.Agent.Provider == ProviderClaude {
					t.Errorf("Trace %s should have tokens for Claude provider", trace.TraceID)
				}
			}
		}
	})
}

// ============================================================================
// Level 8: End-to-End Workflow Simulation
// ============================================================================

func TestIntegration_Level8_EndToEndWorkflow(t *testing.T) {
	backend := setupIntegrationBackend(t)
	defer backend.Close()
	ctx := context.Background()

	// Simulate: Complete workflow from task creation through execution
	t.Run("full_coordinator_to_observatory_flow", func(t *testing.T) {
		// Step 1: Create workspace (like coordinator would)
		ws := &Workspace{
			ID:        "ws-e2e-001",
			Name:      "ailang",
			Path:      "/Users/mark/dev/sunholo/ailang",
			GitRemote: "github.com/sunholo-data/ailang",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := backend.CreateWorkspace(ctx, ws); err != nil {
			t.Fatalf("Step 1 (CreateWorkspace) failed: %v", err)
		}

		// Step 2: Create task (coordinator syncs to observatory)
		now := time.Now()
		task := &Task{
			ID:          "task-e2e-001",
			WorkspaceID: "ws-e2e-001",
			Title:       "Implement M-TASK-HIERARCHY feature",
			Description: "Connect coordinator tasks with Observatory traces",
			SourceType:  TaskSourceGitHub,
			SourceRef:   "issue/456",
			Status:      TaskStatusRunning,
			Priority:    "high",
			CreatedAt:   now,
			StartedAt:   &now,
		}
		if err := backend.CreateTask(ctx, task); err != nil {
			t.Fatalf("Step 2 (CreateTask) failed: %v", err)
		}

		// Step 3: Create agent assignment (coordinator spawns executor)
		aa := &AgentAssignment{
			ID:         "assign-e2e-001",
			TaskID:     "task-e2e-001",
			AgentID:    "sprint-executor",
			Provider:   ProviderClaude,
			Status:     AgentStatusRunning,
			AssignedAt: now,
			StartedAt:  &now,
		}
		if err := backend.CreateAgentAssignment(ctx, aa); err != nil {
			t.Fatalf("Step 3 (CreateAgentAssignment) failed: %v", err)
		}

		// Step 4: Simulate spans arriving (OTLP receiver processes them)
		// These would have task_id from OTEL_RESOURCE_ATTRIBUTES
		spans := []struct {
			id       string
			name     string
			parent   string
			tokens   int64
			duration int64
			status   SpanStatus
		}{
			{"span-e2e-001", "executor.claude.execute", "", 0, 60000, SpanStatusOK},
			{"span-e2e-002", "anthropic.messages.create", "span-e2e-001", 15000, 5000, SpanStatusOK},
			{"span-e2e-003", "tool.Edit", "span-e2e-001", 0, 500, SpanStatusOK},
			{"span-e2e-004", "anthropic.messages.create", "span-e2e-001", 12000, 4000, SpanStatusOK},
			{"span-e2e-005", "tool.Bash", "span-e2e-001", 0, 2000, SpanStatusOK},
			{"span-e2e-006", "anthropic.messages.create", "span-e2e-001", 8000, 3000, SpanStatusError},
		}

		for _, s := range spans {
			span := &Span{
				ID:                s.id,
				TraceID:           "trace-e2e-001",
				ParentSpanID:      s.parent,
				TaskID:            "task-e2e-001",
				AgentAssignmentID: "assign-e2e-001",
				Name:              s.name,
				Kind:              SpanKindClient,
				Status:            s.status,
				StartTime:         now.Add(-time.Duration(60-s.duration/1000) * time.Second),
				DurationMs:        s.duration,
				TokensIn:          s.tokens,
				TokensOut:         s.tokens / 4, // Rough estimate
				CostUSD:           float64(s.tokens) * 0.000003,
				Provider:          ProviderClaude,
				ResourceAttributes: map[string]any{
					"service.name":         "claude-code",
					"ailang.task_id":       "task-e2e-001",
					"ailang.agent_id":      "sprint-executor",
					"ailang.workspace":     "/Users/mark/dev/sunholo/ailang",
					"ailang.assignment_id": "assign-e2e-001",
				},
				CreatedAt: time.Now(),
			}
			if err := backend.CreateSpan(ctx, span); err != nil {
				t.Fatalf("Step 4 (CreateSpan %s) failed: %v", s.id, err)
			}
		}

		// Step 5: Query hierarchy and verify everything is linked
		hierarchy, err := GetTaskHierarchy(ctx, backend, "task-e2e-001", DefaultHierarchyOptions())
		if err != nil {
			t.Fatalf("Step 5 (GetTaskHierarchy) failed: %v", err)
		}

		// Verify task
		if hierarchy.Task.Status != TaskStatusRunning {
			t.Errorf("Task status should be running, got %s", hierarchy.Task.Status)
		}

		// Verify task aggregates
		taskAfter, _ := backend.GetTask(ctx, "task-e2e-001")
		if taskAfter.SpanCount < 6 {
			t.Errorf("Task should have >= 6 spans, got %d", taskAfter.SpanCount)
		}
		if taskAfter.ErrorCount < 1 {
			t.Errorf("Task should have >= 1 error, got %d", taskAfter.ErrorCount)
		}
		expectedTokens := int64(15000 + 12000 + 8000)
		if taskAfter.TotalTokensIn < expectedTokens {
			t.Errorf("Task tokens_in should be >= %d, got %d", expectedTokens, taskAfter.TotalTokensIn)
		}

		// Verify agent in hierarchy
		if len(hierarchy.Agents) != 1 {
			t.Errorf("Should have 1 agent, got %d", len(hierarchy.Agents))
		}
		if hierarchy.Agents[0].Agent.AgentID != "sprint-executor" {
			t.Errorf("Agent ID mismatch: got %q", hierarchy.Agents[0].Agent.AgentID)
		}

		// Verify traces in hierarchy
		if len(hierarchy.Agents[0].Traces) != 1 {
			t.Errorf("Should have 1 trace, got %d", len(hierarchy.Agents[0].Traces))
		}
		trace := hierarchy.Agents[0].Traces[0]
		if trace.Summary.SpanCount != 6 {
			t.Errorf("Trace should have 6 spans, got %d", trace.Summary.SpanCount)
		}
		if trace.Summary.ErrorCount != 1 {
			t.Errorf("Trace should have 1 error, got %d", trace.Summary.ErrorCount)
		}

		// Verify span tree structure
		if trace.RootSpan == nil {
			t.Error("Trace should have root span")
		} else {
			if trace.RootSpan.Span.Name != "executor.claude.execute" {
				t.Errorf("Root span name mismatch: got %q", trace.RootSpan.Span.Name)
			}
			if len(trace.RootSpan.Children) != 5 {
				t.Errorf("Root span should have 5 children, got %d", len(trace.RootSpan.Children))
			}
		}

		t.Logf("E2E Test Complete:")
		t.Logf("  Task: %s (%s)", taskAfter.Title, taskAfter.Status)
		t.Logf("  Spans: %d, Errors: %d", taskAfter.SpanCount, taskAfter.ErrorCount)
		t.Logf("  Tokens: in=%d, out=%d", taskAfter.TotalTokensIn, taskAfter.TotalTokensOut)
		t.Logf("  Cost: $%.4f", taskAfter.TotalCostUSD)
	})
}

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
