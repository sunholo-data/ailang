package observatory

import (
	"context"
	"testing"
)

// seedStage creates a stage in a chain and sets its cost/tokens/model. If model is
// non-empty it is stored via the eval_assessment JSON (exercising json_extract in
// the rollup SQL).
func seedStage(t *testing.T, store *Store, chainID, agentID string, cost float64, tokensIn, tokensOut int, model string) {
	t.Helper()
	ctx := context.Background()
	stage, err := store.CreateStage(ctx, &StageCreateRequest{ChainID: chainID, AgentID: agentID})
	if err != nil {
		t.Fatalf("CreateStage: %v", err)
	}
	if err := store.UpdateStageMetrics(ctx, stage.ID, cost, tokensIn, tokensOut, 0, 0, 0, ""); err != nil {
		t.Fatalf("UpdateStageMetrics: %v", err)
	}
	if model != "" {
		if err := store.UpdateStageEvalAssessment(ctx, stage.ID, &EvalAssessment{Model: model, BenchmarkID: "b", Language: "ailang", EvalMode: "agent"}); err != nil {
			t.Fatalf("UpdateStageEvalAssessment: %v", err)
		}
	}
}

func TestStore_GetCostRollup_ClassifiesViaSQL(t *testing.T) {
	ensurePricingLoaded(t)
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	chain, err := store.CreateChain(ctx, &ChainCreateRequest{SourceType: ChainSourceEvalSuite, SourceRef: "eval_suite:x"})
	if err != nil {
		t.Fatalf("CreateChain: %v", err)
	}

	// reported: cost>0
	seedStage(t, store, chain.ID, "reporter", 3.50, 1000, 500, meteredModel)
	// estimated: tokens>0, cost==0, metered model via eval_assessment JSON
	seedStage(t, store, chain.ID, "estimator", 0, 1_000_000, 500_000, meteredModel)
	// unknown: tokens>0, cost==0, no model
	seedStage(t, store, chain.ID, "unknowner", 0, 8_500_000, 0, "")
	// quota: no tokens
	seedStage(t, store, chain.ID, "quota (quota:opus)", 0, 0, 0, "")

	rollup, err := store.GetCostRollup(ctx, nil, "")
	if err != nil {
		t.Fatalf("GetCostRollup: %v", err)
	}

	if rollup.ReportedStages != 1 || rollup.ReportedCost != 3.50 {
		t.Errorf("reported: got %d stages $%f", rollup.ReportedStages, rollup.ReportedCost)
	}
	if rollup.EstimatedStages != 1 || rollup.EstimatedCost <= 0 {
		t.Errorf("estimated: got %d stages $%f (want 1 stage, >0)", rollup.EstimatedStages, rollup.EstimatedCost)
	}
	if rollup.UnknownStages != 1 {
		t.Errorf("unknown: got %d (want 1)", rollup.UnknownStages)
	}
	if rollup.QuotaStages != 1 {
		t.Errorf("quota: got %d (want 1)", rollup.QuotaStages)
	}
	if !rollup.HasIncompleteData() {
		t.Errorf("expected incomplete-data flag")
	}
}

func TestStore_GetMissionRollups_GroupsAndBudgets(t *testing.T) {
	ensurePricingLoaded(t)
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Two iterations of the same mission → one grouped rollup "mission:v1".
	c1, _ := store.CreateChain(ctx, &ChainCreateRequest{SourceType: ChainSourceManual, SourceRef: "mission:v1/iter-1"})
	c2, _ := store.CreateChain(ctx, &ChainCreateRequest{SourceType: ChainSourceManual, SourceRef: "mission:v1/iter-2"})
	// A non-mission chain must be excluded by the prefix filter.
	cx, _ := store.CreateChain(ctx, &ChainCreateRequest{SourceType: ChainSourceEvalSuite, SourceRef: "eval_suite:y"})

	seedStage(t, store, c1.ID, "codex-executor", 2.00, 100, 100, meteredModel) // reported
	seedStage(t, store, c2.ID, "controller (quota:opus)", 0, 0, 0, "")         // quota:opus
	seedStage(t, store, c2.ID, "reviewer (quota:sonnet)", 0, 0, 0, "")         // quota:sonnet
	seedStage(t, store, cx.ID, "eval-agent", 9.99, 100, 100, meteredModel)     // excluded

	rollups, err := store.GetMissionRollups(ctx, nil, "mission:", 5)
	if err != nil {
		t.Fatalf("GetMissionRollups: %v", err)
	}
	if len(rollups) != 1 {
		t.Fatalf("expected 1 mission group, got %d", len(rollups))
	}
	mr := rollups[0]
	if mr.Mission != "mission:v1" {
		t.Errorf("mission key: got %q, want mission:v1", mr.Mission)
	}
	if mr.Rollup.ReportedCost != 2.00 {
		t.Errorf("metered total: got $%f, want $2.00 (excluded chain must not leak)", mr.Rollup.ReportedCost)
	}
	if mr.QuotaByBucket["opus"] != 1 || mr.QuotaByBucket["sonnet"] != 1 {
		t.Errorf("quota buckets: got %v, want opus=1 sonnet=1", mr.QuotaByBucket)
	}
	if len(mr.TopStages) == 0 || mr.TopStages[0].CostUSD != 2.00 {
		t.Errorf("top stage: got %+v, want most-expensive $2.00 first", mr.TopStages)
	}
}
