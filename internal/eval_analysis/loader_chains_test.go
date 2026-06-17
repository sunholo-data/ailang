package eval_analysis

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sunholo-data/ailang/internal/observatory"
)

func setupTestObservatory(t *testing.T) *observatory.Store {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if _, err := observatory.MigrateWithVersion(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	return observatory.NewStore(db)
}

func TestStageToResult(t *testing.T) {
	stage := &observatory.ChainStage{
		ID:         "stage-1",
		ChainID:    "chain-1",
		Cost:       0.005,
		TokensIn:   1000,
		TokensOut:  500,
		Turns:      4,
		ToolCalls:  7,
		DurationMs: 12000,
		EvalAssessment: &observatory.EvalAssessment{
			BenchmarkID:    "fizzbuzz",
			Model:          "claude-haiku-4-5",
			Language:       "python",
			EvalMode:       "agent",
			Executor:       "claude",
			Seed:           42,
			CompileOk:      true,
			RuntimeOk:      true,
			StdoutOk:       true,
			ErrorCategory:  "",
			FirstAttemptOk: true,
			PromptVersion:  "v0.3.24",
		},
	}

	result := stageToResult(stage)

	if result.ID != "fizzbuzz" {
		t.Errorf("expected ID fizzbuzz, got %s", result.ID)
	}
	if result.Model != "claude-haiku-4-5" {
		t.Errorf("expected model claude-haiku-4-5, got %s", result.Model)
	}
	if result.Lang != "python" {
		t.Errorf("expected lang python, got %s", result.Lang)
	}
	if !result.StdoutOk {
		t.Error("expected StdoutOk true")
	}
	if result.CostUSD != 0.005 {
		t.Errorf("expected cost 0.005, got %f", result.CostUSD)
	}
	if result.InputTokens != 1000 {
		t.Errorf("expected 1000 input tokens, got %d", result.InputTokens)
	}
	if result.TotalTokens != 1500 {
		t.Errorf("expected 1500 total tokens, got %d", result.TotalTokens)
	}
	if result.AgentTurns != 4 {
		t.Errorf("expected 4 agent turns, got %d", result.AgentTurns)
	}
	if result.AgentToolCalls != 7 {
		t.Errorf("expected 7 agent tool calls, got %d", result.AgentToolCalls)
	}
	if result.EvalMode != "agent" {
		t.Errorf("expected eval mode agent, got %s", result.EvalMode)
	}
}

func TestLoadResultsFromChain(t *testing.T) {
	store := setupTestObservatory(t)
	ctx := context.Background()

	// Create a chain
	chain, err := store.CreateChain(ctx, &observatory.ChainCreateRequest{
		SourceType:    observatory.ChainSourceEvalSuite,
		SourceRef:     "test/agent/baseline",
		WorkspacePath: "/tmp/test",
	})
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}

	// Create stages with assessments
	benchmarks := []struct {
		id   string
		lang string
		pass bool
	}{
		{"fizzbuzz", "python", true},
		{"fizzbuzz", "ailang", false},
		{"json_parse", "python", true},
	}

	for _, b := range benchmarks {
		stage, err := store.CreateStage(ctx, &observatory.StageCreateRequest{
			ChainID: chain.ID,
			AgentID: "eval-agent",
		})
		if err != nil {
			t.Fatalf("failed to create stage: %v", err)
		}

		assessment := &observatory.EvalAssessment{
			BenchmarkID: b.id,
			Model:       "claude-haiku-4-5",
			Language:    b.lang,
			EvalMode:    "agent",
			Executor:    "claude",
			Seed:        42,
			CompileOk:   true,
			RuntimeOk:   b.pass,
			StdoutOk:    b.pass,
		}
		if err := store.UpdateStageEvalAssessment(ctx, stage.ID, assessment); err != nil {
			t.Fatalf("failed to update assessment: %v", err)
		}
		if err := store.UpdateStageMetrics(ctx, stage.ID, 0.003, 500, 200, 3, 5, 8000); err != nil {
			t.Fatalf("failed to update metrics: %v", err)
		}
	}

	// Query results using the chain ID
	stages, err := store.QueryEvalResults(ctx, observatory.EvalQueryOptions{
		ChainID: chain.ID,
	})
	if err != nil {
		t.Fatalf("failed to query eval results: %v", err)
	}

	if len(stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(stages))
	}

	// Convert to BenchmarkResults
	var results []*BenchmarkResult
	for _, stage := range stages {
		if stage.EvalAssessment != nil {
			results = append(results, stageToResult(stage))
		}
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Check that passing/failing is correctly mapped
	passCount := 0
	for _, r := range results {
		if r.StdoutOk {
			passCount++
		}
	}
	if passCount != 2 {
		t.Errorf("expected 2 passing results, got %d", passCount)
	}
}

func TestLoadResultsFromChain_EmptyChain(t *testing.T) {
	store := setupTestObservatory(t)
	ctx := context.Background()

	// Create a chain with no stages
	chain, err := store.CreateChain(ctx, &observatory.ChainCreateRequest{
		SourceType:    observatory.ChainSourceEvalSuite,
		SourceRef:     "test/empty",
		WorkspacePath: "/tmp/test",
	})
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}

	// Query should return no results
	stages, err := store.QueryEvalResults(ctx, observatory.EvalQueryOptions{
		ChainID: chain.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stages) != 0 {
		t.Errorf("expected 0 stages for empty chain, got %d", len(stages))
	}
}
