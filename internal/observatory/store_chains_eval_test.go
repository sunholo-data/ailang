package observatory

import (
	"context"
	"testing"
)

// ===== Eval Assessment Tests (M-EVAL-CHAINS) =====

func createTestEvalChainWithStage(t *testing.T, store *Store) (*ExecutionChain, *ChainStage) {
	t.Helper()
	ctx := context.Background()

	chain, err := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType:    ChainSourceEvalSuite,
		SourceRef:     "v0.8.1/agent/baseline",
		WorkspacePath: "/tmp/eval",
	})
	if err != nil {
		t.Fatalf("failed to create eval chain: %v", err)
	}

	stage, err := store.CreateStage(ctx, &StageCreateRequest{
		ChainID: chain.ID,
		AgentID: "eval-agent",
	})
	if err != nil {
		t.Fatalf("failed to create stage: %v", err)
	}

	return chain, stage
}

func TestStore_UpdateStageEvalAssessment(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, stage := createTestEvalChainWithStage(t, store)

	assessment := &EvalAssessment{
		BenchmarkID:   "fizzbuzz",
		Model:         "claude-haiku-4-5",
		Language:      "ailang",
		EvalMode:      "agent",
		Executor:      "claude",
		CompileOk:     true,
		RuntimeOk:     true,
		StdoutOk:      true,
		ErrorCategory: "none",
		PromptVersion: "v0.3.24",
	}

	err := store.UpdateStageEvalAssessment(ctx, stage.ID, assessment)
	if err != nil {
		t.Fatalf("failed to update eval assessment: %v", err)
	}

	// Read back
	got, err := store.GetStageEvalAssessment(ctx, stage.ID)
	if err != nil {
		t.Fatalf("failed to get eval assessment: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil assessment")
	}
	if got.BenchmarkID != "fizzbuzz" {
		t.Errorf("expected benchmark_id fizzbuzz, got %s", got.BenchmarkID)
	}
	if !got.CompileOk {
		t.Error("expected compile_ok = true")
	}
	if !got.StdoutOk {
		t.Error("expected stdout_ok = true")
	}
	if got.EvalMode != "agent" {
		t.Errorf("expected eval_mode agent, got %s", got.EvalMode)
	}
}

func TestStore_UpdateStageEvalAssessment_Validation(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Empty stage ID
	err := store.UpdateStageEvalAssessment(ctx, "", &EvalAssessment{})
	if err == nil {
		t.Error("expected error for empty stage_id")
	}

	// Nil assessment
	err = store.UpdateStageEvalAssessment(ctx, "some-id", nil)
	if err == nil {
		t.Error("expected error for nil assessment")
	}
}

func TestStore_GetStageEvalAssessment_NoAssessment(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, stage := createTestEvalChainWithStage(t, store)

	// Stage exists but has no eval assessment
	got, err := store.GetStageEvalAssessment(ctx, stage.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil assessment for stage without eval data")
	}
}

func TestStore_QueryEvalResults(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	chain, _ := createTestEvalChainWithStage(t, store)

	// Create multiple stages with different assessments
	benchmarks := []struct {
		id       string
		model    string
		lang     string
		stdoutOk bool
	}{
		{"fizzbuzz", "claude-haiku-4-5", "ailang", true},
		{"fizzbuzz", "gemini-2-5-flash", "ailang", false},
		{"json_parse", "claude-haiku-4-5", "ailang", true},
		{"json_parse", "claude-haiku-4-5", "python", true},
	}

	for i, b := range benchmarks {
		stage, err := store.CreateStage(ctx, &StageCreateRequest{
			ChainID: chain.ID,
			AgentID: "eval-agent",
		})
		if err != nil {
			t.Fatalf("failed to create stage %d: %v", i, err)
		}

		assessment := &EvalAssessment{
			BenchmarkID:   b.id,
			Model:         b.model,
			Language:      b.lang,
			EvalMode:      "agent",
			CompileOk:     true,
			RuntimeOk:     true,
			StdoutOk:      b.stdoutOk,
			ErrorCategory: "none",
		}
		if !b.stdoutOk {
			assessment.ErrorCategory = "logic_error"
		}

		err = store.UpdateStageEvalAssessment(ctx, stage.ID, assessment)
		if err != nil {
			t.Fatalf("failed to update assessment %d: %v", i, err)
		}
	}

	// Query all eval results for chain
	results, err := store.QueryEvalResults(ctx, EvalQueryOptions{ChainID: chain.ID})
	if err != nil {
		t.Fatalf("failed to query eval results: %v", err)
	}
	// 4 stages + 1 original stage without assessment = 4 with assessment
	if len(results) != 4 {
		t.Errorf("expected 4 results, got %d", len(results))
	}

	// Filter by model
	results, err = store.QueryEvalResults(ctx, EvalQueryOptions{
		ChainID: chain.ID,
		Model:   "claude-haiku-4-5",
	})
	if err != nil {
		t.Fatalf("failed to query by model: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 claude results, got %d", len(results))
	}

	// Filter by benchmark
	results, err = store.QueryEvalResults(ctx, EvalQueryOptions{
		ChainID:     chain.ID,
		BenchmarkID: "fizzbuzz",
	})
	if err != nil {
		t.Fatalf("failed to query by benchmark: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 fizzbuzz results, got %d", len(results))
	}

	// Filter failures only
	results, err = store.QueryEvalResults(ctx, EvalQueryOptions{
		ChainID:     chain.ID,
		FailureOnly: true,
	})
	if err != nil {
		t.Fatalf("failed to query failures: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 failure, got %d", len(results))
	}
	if results[0].EvalAssessment == nil {
		t.Fatal("expected eval assessment to be populated")
	}
	if results[0].EvalAssessment.Model != "gemini-2-5-flash" {
		t.Errorf("expected gemini model in failure, got %s", results[0].EvalAssessment.Model)
	}

	// Filter successes only
	results, err = store.QueryEvalResults(ctx, EvalQueryOptions{
		ChainID:     chain.ID,
		SuccessOnly: true,
	})
	if err != nil {
		t.Fatalf("failed to query successes: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 successes, got %d", len(results))
	}

	// Filter by language
	results, err = store.QueryEvalResults(ctx, EvalQueryOptions{
		ChainID:  chain.ID,
		Language: "python",
	})
	if err != nil {
		t.Fatalf("failed to query by language: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 python result, got %d", len(results))
	}
}

func TestStore_ListEvalChains(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create an eval chain
	_, err := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceEvalSuite,
		SourceRef:  "v0.8.1/agent/baseline",
	})
	if err != nil {
		t.Fatalf("failed to create eval chain: %v", err)
	}

	// Create a non-eval chain
	_, err = store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceManual,
		SourceRef:  "test-task",
	})
	if err != nil {
		t.Fatalf("failed to create manual chain: %v", err)
	}

	// List eval chains only
	evalChains, err := store.ListEvalChains(ctx, 10)
	if err != nil {
		t.Fatalf("failed to list eval chains: %v", err)
	}
	if len(evalChains) != 1 {
		t.Errorf("expected 1 eval chain, got %d", len(evalChains))
	}
	if evalChains[0].SourceType != string(ChainSourceEvalSuite) {
		t.Errorf("expected source_type eval_suite, got %s", evalChains[0].SourceType)
	}
}

func TestStore_EvalAssessment_ContractVerification(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, stage := createTestEvalChainWithStage(t, store)

	assessment := &EvalAssessment{
		BenchmarkID:     "safe_sub",
		Model:           "claude-sonnet-4-5",
		Language:        "ailang",
		EvalMode:        "agent",
		CompileOk:       true,
		RuntimeOk:       true,
		StdoutOk:        true,
		VerifyOk:        true,
		VerifyVerified:  3,
		VerifyCounterex: 0,
		VerifySkipped:   1,
		VerifyErrors:    0,
		Condition:       "z3_guided",
		PromptVersion:   "v0.3.24",
		CodeHash:        "a1b2c3d4",
	}

	err := store.UpdateStageEvalAssessment(ctx, stage.ID, assessment)
	if err != nil {
		t.Fatalf("failed to update assessment: %v", err)
	}

	got, err := store.GetStageEvalAssessment(ctx, stage.ID)
	if err != nil {
		t.Fatalf("failed to get assessment: %v", err)
	}
	if !got.VerifyOk {
		t.Error("expected verify_ok = true")
	}
	if got.VerifyVerified != 3 {
		t.Errorf("expected 3 verified, got %d", got.VerifyVerified)
	}
	if got.Condition != "z3_guided" {
		t.Errorf("expected condition z3_guided, got %s", got.Condition)
	}
}
