// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupControlPlaneBackend creates a test SQLiteBackend with fresh schema and cleanup.
func setupControlPlaneBackend(t *testing.T) (*SQLiteBackend, func()) {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	if err := Migrate(db); err != nil {
		db.Close()
		t.Fatalf("failed to migrate: %v", err)
	}

	backend, err := NewSQLiteBackend(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create backend: %v", err)
	}

	return backend, func() {
		backend.Close()
	}
}

// createTestSpanWithMetrics creates a span with token and cost data.
func createTestSpanWithMetrics(t *testing.T, backend *SQLiteBackend, name, provider, model, sourceType string, tokensIn, tokensOut int64, costUSD float64) *Span {
	t.Helper()
	ctx := context.Background()

	span := &Span{
		ID:        generateSpanID(),
		TraceID:   generateTraceID(),
		Name:      name,
		Kind:      SpanKindClient,
		Status:    SpanStatusOK,
		StartTime: time.Now().Add(-time.Hour),
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		CostUSD:   costUSD,
		Model:     model,
		Provider:  Provider(provider),
		Attributes: map[string]any{
			"source_type": sourceType,
		},
	}

	if err := backend.CreateSpan(ctx, span); err != nil {
		t.Fatalf("failed to create span: %v", err)
	}
	return span
}

func TestGetBreakdownByProvider(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create spans with different providers
	createTestSpanWithMetrics(t, backend, "claude-op", "claude", "claude-sonnet", "", 1000, 500, 0.05)
	createTestSpanWithMetrics(t, backend, "claude-op2", "claude", "claude-haiku", "", 2000, 1000, 0.03)
	createTestSpanWithMetrics(t, backend, "gemini-op", "gemini", "gemini-pro", "", 1500, 750, 0.02)

	// Test breakdown by provider
	breakdown, err := backend.GetBreakdownByProvider(ctx)
	if err != nil {
		t.Fatalf("GetBreakdownByProvider failed: %v", err)
	}

	if len(breakdown) < 2 {
		t.Errorf("expected at least 2 providers, got %d", len(breakdown))
	}

	// Find claude provider
	var claudeFound bool
	for _, b := range breakdown {
		if b.ID == "claude" {
			claudeFound = true
			if b.SpanCount != 2 {
				t.Errorf("expected 2 claude spans, got %d", b.SpanCount)
			}
			if b.TokensIn != 3000 {
				t.Errorf("expected 3000 tokens in, got %d", b.TokensIn)
			}
		}
	}
	if !claudeFound {
		t.Error("claude provider not found in breakdown")
	}
}

func TestGetBreakdownByModel(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create spans with different models
	createTestSpanWithMetrics(t, backend, "op1", "claude", "claude-sonnet-4-5", "", 1000, 500, 0.05)
	createTestSpanWithMetrics(t, backend, "op2", "claude", "claude-sonnet-4-5", "", 1000, 500, 0.05)
	createTestSpanWithMetrics(t, backend, "op3", "gemini", "gemini-2-5-pro", "", 1500, 750, 0.02)

	breakdown, err := backend.GetBreakdownByModel(ctx)
	if err != nil {
		t.Fatalf("GetBreakdownByModel failed: %v", err)
	}

	if len(breakdown) < 2 {
		t.Errorf("expected at least 2 models, got %d", len(breakdown))
	}

	// Find claude-sonnet-4-5
	var sonnetFound bool
	for _, b := range breakdown {
		if b.ID == "claude-sonnet-4-5" {
			sonnetFound = true
			if b.SpanCount != 2 {
				t.Errorf("expected 2 sonnet spans, got %d", b.SpanCount)
			}
		}
	}
	if !sonnetFound {
		t.Error("claude-sonnet-4-5 model not found in breakdown")
	}
}

func TestGetBreakdownBySourceType(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create spans with different source types
	createTestSpanWithMetrics(t, backend, "eval1", "claude", "claude-sonnet", "eval", 1000, 500, 0.05)
	createTestSpanWithMetrics(t, backend, "eval2", "gemini", "gemini-pro", "eval", 2000, 1000, 0.03)
	createTestSpanWithMetrics(t, backend, "coord1", "claude", "claude-haiku", "coordinator", 500, 250, 0.01)

	breakdown, err := backend.GetBreakdownBySourceType(ctx)
	if err != nil {
		t.Fatalf("GetBreakdownBySourceType failed: %v", err)
	}

	// Check that we got some breakdown items
	if len(breakdown) == 0 {
		t.Error("expected non-empty breakdown")
	}
}

func TestGetFilteredHeatmapData(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create spans across multiple days
	now := time.Now()
	for i := 0; i < 5; i++ {
		span := &Span{
			ID:        generateSpanID(),
			TraceID:   generateTraceID(),
			Name:      "test-op",
			Kind:      SpanKindClient,
			Status:    SpanStatusOK,
			StartTime: now.Add(-time.Duration(i*24) * time.Hour),
			TokensIn:  100,
			TokensOut: 50,
			CostUSD:   0.01,
			Model:     "claude-sonnet",
			Provider:  ProviderClaude,
		}
		if err := backend.CreateSpan(ctx, span); err != nil {
			t.Fatalf("failed to create span: %v", err)
		}
	}

	// Test with no filter (nil wsConfig for tests)
	heatmap, err := backend.GetFilteredHeatmapData(ctx, nil, 30, nil)
	if err != nil {
		t.Fatalf("GetFilteredHeatmapData failed: %v", err)
	}

	if len(heatmap) == 0 {
		t.Error("expected non-empty heatmap data")
	}

	// Verify structure
	for _, point := range heatmap {
		if point.Date == "" {
			t.Error("expected non-empty date")
		}
	}
}

func TestGetFilteredHeatmapData_WithProviderFilter(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create spans with different providers
	createTestSpanWithMetrics(t, backend, "claude-op", "claude", "claude-sonnet", "", 1000, 500, 0.05)
	createTestSpanWithMetrics(t, backend, "gemini-op", "gemini", "gemini-pro", "", 1500, 750, 0.02)

	// Filter by claude
	filter := &ControlPlaneFilter{Provider: "claude"}
	heatmap, err := backend.GetFilteredHeatmapData(ctx, filter, 30, nil)
	if err != nil {
		t.Fatalf("GetFilteredHeatmapData with filter failed: %v", err)
	}

	// Should have data only for claude spans
	var totalSpans int
	for _, point := range heatmap {
		totalSpans += point.SpanCount
	}
	if totalSpans != 1 {
		t.Errorf("expected 1 span with claude filter, got %d", totalSpans)
	}
}

func TestGetFilteredMetricsSummary(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create test data
	createTestSpanWithMetrics(t, backend, "op1", "claude", "claude-sonnet", "", 1000, 500, 0.05)
	createTestSpanWithMetrics(t, backend, "op2", "gemini", "gemini-pro", "", 1500, 750, 0.02)

	// Get metrics without filter (nil wsConfig for tests)
	metrics, err := backend.GetFilteredMetricsSummary(ctx, nil, nil)
	if err != nil {
		t.Fatalf("GetFilteredMetricsSummary failed: %v", err)
	}

	if metrics.TotalSpans != 2 {
		t.Errorf("expected 2 total spans, got %d", metrics.TotalSpans)
	}

	if metrics.TotalTokensIn != 2500 {
		t.Errorf("expected 2500 tokens in, got %d", metrics.TotalTokensIn)
	}
}

func TestBuildFilterConditions(t *testing.T) {
	tests := []struct {
		name           string
		filter         *ControlPlaneFilter
		wantConditions int
		wantArgs       int // May differ from conditions because SourceType is inline SQL
	}{
		{
			name:           "nil filter",
			filter:         nil,
			wantConditions: 0,
			wantArgs:       0,
		},
		{
			name:           "empty filter",
			filter:         &ControlPlaneFilter{},
			wantConditions: 0,
			wantArgs:       0,
		},
		{
			name:           "provider only",
			filter:         &ControlPlaneFilter{Provider: "claude"},
			wantConditions: 1,
			wantArgs:       1,
		},
		{
			name: "multiple filters with source_type",
			filter: &ControlPlaneFilter{
				Provider:   "claude",
				Model:      "claude-sonnet",
				SourceType: "eval", // SourceType is inline SQL, no placeholder
			},
			wantConditions: 3,
			wantArgs:       2, // Only provider and model have placeholders
		},
		{
			name: "date range",
			filter: &ControlPlaneFilter{
				StartDate: "2024-01-01",
				EndDate:   "2024-12-31",
			},
			wantConditions: 2,
			wantArgs:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pass nil wsConfig - tests don't need workspace reverse mapping
			conditions, args := buildFilterConditions(tt.filter, nil)

			if len(conditions) != tt.wantConditions {
				t.Errorf("expected %d conditions, got %d", tt.wantConditions, len(conditions))
			}

			if len(args) != tt.wantArgs {
				t.Errorf("expected %d args, got %d", tt.wantArgs, len(args))
			}
		})
	}
}

func TestBuildSourceTypeCondition(t *testing.T) {
	tests := []struct {
		sourceType string
		wantEmpty  bool
	}{
		{"eval", false},
		{"coordinator", false},
		{"direct_api", false},
		{"local", false},
		{"other", false},
		{"", true},
		{"unknown", true}, // Unknown source types return empty (no condition)
	}

	for _, tt := range tests {
		t.Run(tt.sourceType, func(t *testing.T) {
			condition := buildSourceTypeCondition(tt.sourceType)
			if tt.wantEmpty && condition != "" {
				t.Errorf("expected empty condition, got %q", condition)
			}
			if !tt.wantEmpty && condition == "" {
				t.Errorf("expected non-empty condition for %q", tt.sourceType)
			}
		})
	}
}

func TestGetFilteredBreakdownByProvider(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create spans with names that match source_type patterns:
	// "eval" matches: api_request* or eval.*
	// "coordinator" matches: coordinator.* or claude.execute*
	createTestSpanWithMetrics(t, backend, "eval.benchmark", "claude", "claude-sonnet", "eval", 1000, 500, 0.05)
	createTestSpanWithMetrics(t, backend, "api_request_generate", "claude", "claude-haiku", "eval", 500, 250, 0.02)
	createTestSpanWithMetrics(t, backend, "coordinator.task", "gemini", "gemini-pro", "coordinator", 1500, 750, 0.03)

	// Filter by source_type=eval (matches eval.* and api_request*)
	filter := &ControlPlaneFilter{SourceType: "eval"}
	breakdown, err := backend.GetFilteredBreakdownByProvider(ctx, filter, nil)
	if err != nil {
		t.Fatalf("GetFilteredBreakdownByProvider failed: %v", err)
	}

	// Should only have claude (the only provider with eval spans)
	if len(breakdown) != 1 {
		t.Errorf("expected 1 provider with eval filter, got %d", len(breakdown))
	}

	if len(breakdown) > 0 && breakdown[0].ID != "claude" {
		t.Errorf("expected claude provider, got %s", breakdown[0].ID)
	}
}

func TestGetFilteredBreakdownByModel(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create spans
	createTestSpanWithMetrics(t, backend, "op1", "claude", "claude-sonnet", "", 1000, 500, 0.05)
	createTestSpanWithMetrics(t, backend, "op2", "claude", "claude-sonnet", "", 1000, 500, 0.05)
	createTestSpanWithMetrics(t, backend, "op3", "gemini", "gemini-pro", "", 1500, 750, 0.03)

	// Filter by provider=claude
	filter := &ControlPlaneFilter{Provider: "claude"}
	breakdown, err := backend.GetFilteredBreakdownByModel(ctx, filter, nil)
	if err != nil {
		t.Fatalf("GetFilteredBreakdownByModel failed: %v", err)
	}

	// Should only have claude-sonnet
	if len(breakdown) != 1 {
		t.Errorf("expected 1 model with claude filter, got %d", len(breakdown))
	}

	if len(breakdown) > 0 && breakdown[0].ID != "claude-sonnet" {
		t.Errorf("expected claude-sonnet model, got %s", breakdown[0].ID)
	}
}

// createTestSpanWithWorkspace creates a span with workspace in resource attributes.
func createTestSpanWithWorkspace(t *testing.T, backend *SQLiteBackend, name, provider, model, sourceType, workspace string, tokensIn, tokensOut int64, costUSD float64) *Span {
	t.Helper()
	ctx := context.Background()

	span := &Span{
		ID:        generateSpanID(),
		TraceID:   generateTraceID(),
		Name:      name,
		Kind:      SpanKindClient,
		Status:    SpanStatusOK,
		StartTime: time.Now().Add(-time.Hour),
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		CostUSD:   costUSD,
		Model:     model,
		Provider:  Provider(provider),
		Attributes: map[string]any{
			"source_type": sourceType,
		},
		ResourceAttributes: map[string]any{
			"process.cwd": workspace,
		},
	}

	if err := backend.CreateSpan(ctx, span); err != nil {
		t.Fatalf("failed to create span: %v", err)
	}
	return span
}

func TestGetFilteredBreakdownByWorkspace(t *testing.T) {
	backend, cleanup := setupControlPlaneBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create spans with different workspaces
	createTestSpanWithWorkspace(t, backend, "eval.benchmark", "claude", "claude-sonnet", "eval", "/Users/mark/dev/sunholo/ailang", 1000, 500, 0.05)
	createTestSpanWithWorkspace(t, backend, "eval.benchmark2", "claude", "claude-haiku", "eval", "/Users/mark/dev/sunholo/ailang", 500, 250, 0.02)
	createTestSpanWithWorkspace(t, backend, "coordinator.task", "gemini", "gemini-pro", "coordinator", "/Users/mark/dev/sunholo/twilight", 1500, 750, 0.03)
	createTestSpanWithWorkspace(t, backend, "coordinator.task2", "claude", "claude-sonnet", "coordinator", "/Users/mark/dev/sunholo/stapledon", 2000, 1000, 0.04)

	// Test 1: Filter by source_type=eval (should show workspaces with eval spans only)
	filter := &ControlPlaneFilter{SourceType: "eval"}
	breakdown, err := backend.GetFilteredBreakdownByWorkspace(ctx, filter, nil)
	if err != nil {
		t.Fatalf("GetFilteredBreakdownByWorkspace failed: %v", err)
	}

	// Should only have ailang workspace (the only one with eval spans)
	if len(breakdown) != 1 {
		t.Errorf("expected 1 workspace with eval filter, got %d", len(breakdown))
		for _, b := range breakdown {
			t.Logf("  workspace: %s, label: %s", b.ID, b.Label)
		}
	}

	// Label should be derived from path when no mapping config provided
	// formatLabel("ailang") returns "Ailang" (title case)
	if len(breakdown) > 0 && breakdown[0].Label != "Ailang" {
		t.Errorf("expected Ailang workspace label, got %s", breakdown[0].Label)
	}

	// Test 2: Filter by provider=gemini (should show workspaces with gemini spans only)
	filter2 := &ControlPlaneFilter{Provider: "gemini"}
	breakdown2, err := backend.GetFilteredBreakdownByWorkspace(ctx, filter2, nil)
	if err != nil {
		t.Fatalf("GetFilteredBreakdownByWorkspace (provider filter) failed: %v", err)
	}

	// Should only have twilight workspace
	if len(breakdown2) != 1 {
		t.Errorf("expected 1 workspace with gemini filter, got %d", len(breakdown2))
		for _, b := range breakdown2 {
			t.Logf("  workspace: %s, label: %s", b.ID, b.Label)
		}
	}

	if len(breakdown2) > 0 && breakdown2[0].Label != "Twilight Game" {
		t.Errorf("expected Twilight Game workspace, got %s", breakdown2[0].Label)
	}

	// Test 3: No filter - should return all workspaces
	breakdown3, err := backend.GetFilteredBreakdownByWorkspace(ctx, nil, nil)
	if err != nil {
		t.Fatalf("GetFilteredBreakdownByWorkspace (no filter) failed: %v", err)
	}

	if len(breakdown3) != 3 {
		t.Errorf("expected 3 workspaces with no filter, got %d", len(breakdown3))
		for _, b := range breakdown3 {
			t.Logf("  workspace: %s, label: %s", b.ID, b.Label)
		}
	}
}

func TestWorkspaceFilterInBuildFilterConditions(t *testing.T) {
	// Test that workspace filter generates correct SQL condition
	// With nil wsConfig, falls back to direct LIKE match
	filter := &ControlPlaneFilter{
		Workspace: "ailang",
	}

	conditions, args := buildFilterConditions(filter, nil)

	if len(conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(conditions))
	}

	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}

	// Verify the arg contains the workspace with wildcards
	if len(args) > 0 {
		expectedArg := "%ailang%"
		if args[0] != expectedArg {
			t.Errorf("expected arg %q, got %q", expectedArg, args[0])
		}
	}
}
