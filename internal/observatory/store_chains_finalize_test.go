package observatory

import (
	"context"
	"testing"
)

// M-COMPLETION-PATH-PARITY M0b — every finalisation write must survive a replay.
//
// These are the arms that would have caught the bug the design doc spent three
// quorum rounds asserting away: the existing Update* family accumulates, so a
// redelivered completion double-counts cost. Each test below applies a primitive
// twice and asserts the second application changed nothing — and each is paired
// with a control proving the OLD function really does drift, so "idempotent" is
// demonstrated against a live counterexample rather than asserted.
//
// The metrics assertions check the VALUE, not that a write happened. A
// double-count is invisible to a write-count assertion.

func seedChainAndStage(t *testing.T, store *Store) (chainID, stageID string) {
	t.Helper()
	ctx := context.Background()

	chain, err := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceMessage,
		SourceRef:  "inbox_test_finalize",
	})
	if err != nil {
		t.Fatalf("create chain: %v", err)
	}
	stage, err := store.CreateStage(ctx, &StageCreateRequest{
		ChainID:   chain.ID,
		AgentID:   "design-doc-creator",
		MessageID: "inbox_test_finalize",
		TaskID:    "task-finalize",
	})
	if err != nil {
		t.Fatalf("create stage: %v", err)
	}
	return chain.ID, stage.ID
}

func TestSetStageMetrics_IsIdempotent(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	_, stageID := seedChainAndStage(t, store)

	for i := 0; i < 3; i++ {
		if err := store.SetStageMetrics(ctx, stageID, 0.25, 100, 50, 4, 7, 1200, "reported"); err != nil {
			t.Fatalf("apply %d: %v", i+1, err)
		}
	}

	stage, err := store.GetStage(ctx, stageID)
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if stage.Cost != 0.25 {
		t.Errorf("cost = %v after 3 applications, want 0.25 — the write is accumulating, not absolute", stage.Cost)
	}
	if stage.TokensIn != 100 || stage.TokensOut != 50 {
		t.Errorf("tokens = %d/%d after 3 applications, want 100/50", stage.TokensIn, stage.TokensOut)
	}
	if stage.Turns != 4 || stage.ToolCalls != 7 {
		t.Errorf("turns/tools = %d/%d after 3 applications, want 4/7", stage.Turns, stage.ToolCalls)
	}
	if stage.DurationMs != 1200 {
		t.Errorf("duration_ms = %d after 3 applications, want 1200", stage.DurationMs)
	}
}

// TestUpdateStageMetrics_StillAccumulates is the control for the test above. If
// this ever stops drifting, the accumulating path changed under its existing
// callers and SetStageMetrics is no longer proving anything distinctive.
func TestUpdateStageMetrics_StillAccumulates(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	_, stageID := seedChainAndStage(t, store)

	for i := 0; i < 3; i++ {
		if err := store.UpdateStageMetrics(ctx, stageID, 0.25, 100, 50, 4, 7, 1200, "reported"); err != nil {
			t.Fatalf("apply %d: %v", i+1, err)
		}
	}

	stage, err := store.GetStage(ctx, stageID)
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if stage.Cost == 0.25 {
		t.Fatal("control failed: UpdateStageMetrics no longer accumulates, so the idempotency arm above proves nothing")
	}
	if stage.Cost != 0.75 {
		t.Errorf("cost = %v after 3 accumulating applications, want 0.75", stage.Cost)
	}
}

// TestSetStageMetrics_PreservesFirstProvenanceLabel keeps the "first non-empty
// label wins" rule the accumulating version documents: a caller that cannot
// classify the cost must not erase what an earlier one established.
func TestSetStageMetrics_PreservesFirstProvenanceLabel(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	_, stageID := seedChainAndStage(t, store)

	if err := store.SetStageMetrics(ctx, stageID, 0.25, 100, 50, 4, 7, 1200, "reported"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := store.SetStageMetrics(ctx, stageID, 0.25, 100, 50, 4, 7, 1200, ""); err != nil {
		t.Fatalf("second: %v", err)
	}

	stage, err := store.GetStage(ctx, stageID)
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if stage.CostProvenance != "reported" {
		t.Errorf("cost_provenance = %q, want \"reported\" — an unclassified replay erased the label", stage.CostProvenance)
	}
}

// TestSetStageStatus_DoesNotTouchTheChainCounter pins the distinction that took a
// quorum round to find: UpdateStageStatus writes the status absolutely but ALSO
// increments the chain's stages_completed counter, which is what makes it
// non-idempotent. SetStageStatus must leave the counter entirely alone.
func TestSetStageStatus_DoesNotTouchTheChainCounter(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	chainID, stageID := seedChainAndStage(t, store)

	for i := 0; i < 3; i++ {
		if err := store.SetStageStatus(ctx, stageID, StageStatusCompleted); err != nil {
			t.Fatalf("apply %d: %v", i+1, err)
		}
	}

	stage, err := store.GetStage(ctx, stageID)
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if stage.Status != StageStatusCompleted {
		t.Errorf("status = %s, want %s", stage.Status, StageStatusCompleted)
	}

	chain, err := store.GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("get chain: %v", err)
	}
	if chain.StagesCompleted != 0 {
		t.Errorf("stages_completed = %d, want 0 — SetStageStatus must not carry the counter side effect; RecomputeChainAggregates owns it", chain.StagesCompleted)
	}
}

// TestUpdateStageStatus_StillIncrementsTheCounter is the control: it proves the
// side effect the test above avoids is real and still present for existing
// callers. Three applications, three increments — the double-count in the flesh.
func TestUpdateStageStatus_StillIncrementsTheCounter(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	chainID, stageID := seedChainAndStage(t, store)

	for i := 0; i < 3; i++ {
		if err := store.UpdateStageStatus(ctx, stageID, StageStatusCompleted); err != nil {
			t.Fatalf("apply %d: %v", i+1, err)
		}
	}

	chain, err := store.GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("get chain: %v", err)
	}
	if chain.StagesCompleted == 0 {
		t.Fatal("control failed: UpdateStageStatus no longer increments stages_completed, so the arm above proves nothing")
	}
	if chain.StagesCompleted != 3 {
		t.Errorf("stages_completed = %d after 3 identical completions, want 3 (one increment each) — the counter drift this design replaces", chain.StagesCompleted)
	}
}

// TestRecomputeChainAggregates_IsIdempotentAndDerived is the heart of M0b.
//
// The totals are a pure function of the stage rows, so calling it repeatedly —
// or concurrently, from two finalizers with different snapshots — converges on
// the same value. It also repairs a counter that increments left wrong, which is
// what makes it safe to run after the old path has already drifted.
func TestRecomputeChainAggregates_IsIdempotentAndDerived(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	chainID, stageID := seedChainAndStage(t, store)

	if err := store.SetStageMetrics(ctx, stageID, 0.40, 200, 100, 5, 9, 3000, "reported"); err != nil {
		t.Fatalf("set stage metrics: %v", err)
	}
	if err := store.SetStageStatus(ctx, stageID, StageStatusCompleted); err != nil {
		t.Fatalf("set stage status: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := store.RecomputeChainAggregates(ctx, chainID); err != nil {
			t.Fatalf("apply %d: %v", i+1, err)
		}
	}

	chain, err := store.GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("get chain: %v", err)
	}
	if chain.TotalCost != 0.40 {
		t.Errorf("total_cost = %v after 3 recomputes, want 0.40", chain.TotalCost)
	}
	if chain.TotalTokens != 300 {
		t.Errorf("total_tokens = %d, want 300 (tokens_in + tokens_out)", chain.TotalTokens)
	}
	if chain.TotalTurns != 5 {
		t.Errorf("total_turns = %d, want 5", chain.TotalTurns)
	}
	if chain.StagesCompleted != 1 {
		t.Errorf("stages_completed = %d, want 1 — derived from the stage rows, not incremented", chain.StagesCompleted)
	}
}

// TestRecomputeChainAggregates_RepairsDriftedCounters proves the recompute is not
// merely idempotent going forward but corrective: it fixes chains the
// incrementing path has already inflated. That is what lets M4 reconcile history
// without inventing it.
func TestRecomputeChainAggregates_RepairsDriftedCounters(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	chainID, stageID := seedChainAndStage(t, store)

	// Drift it the way a redelivered completion would, using the old path.
	for i := 0; i < 3; i++ {
		if err := store.UpdateStageMetrics(ctx, stageID, 0.40, 200, 100, 5, 9, 3000, "reported"); err != nil {
			t.Fatalf("drift %d: %v", i+1, err)
		}
		if err := store.UpdateStageStatus(ctx, stageID, StageStatusCompleted); err != nil {
			t.Fatalf("drift status %d: %v", i+1, err)
		}
		if err := store.UpdateChainMetrics(ctx, chainID, 0.40, 300, 5); err != nil {
			t.Fatalf("drift chain %d: %v", i+1, err)
		}
	}

	drifted, err := store.GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("get drifted chain: %v", err)
	}
	if drifted.TotalCost == 0.40 {
		t.Fatal("control failed: the old path did not drift, so this repair test proves nothing")
	}

	// Now repair: set the stage to its true values, then derive the chain.
	if err := store.SetStageMetrics(ctx, stageID, 0.40, 200, 100, 5, 9, 3000, "reported"); err != nil {
		t.Fatalf("repair stage: %v", err)
	}
	if err := store.RecomputeChainAggregates(ctx, chainID); err != nil {
		t.Fatalf("recompute: %v", err)
	}

	chain, err := store.GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("get chain: %v", err)
	}
	if chain.TotalCost != 0.40 {
		t.Errorf("total_cost = %v after repair, want 0.40 (drifted value was %v)", chain.TotalCost, drifted.TotalCost)
	}
	if chain.StagesCompleted != 1 {
		t.Errorf("stages_completed = %d after repair, want 1 (drifted value was %d)", chain.StagesCompleted, drifted.StagesCompleted)
	}
}

func TestSetStageError_DoesNotInflateErrorCount(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	_, stageID := seedChainAndStage(t, store)

	for i := 0; i < 3; i++ {
		if err := store.SetStageError(ctx, stageID, "pi idle for 3m0s mid-generation"); err != nil {
			t.Fatalf("apply %d: %v", i+1, err)
		}
	}

	stage, err := store.GetStage(ctx, stageID)
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if stage.ErrorCount != 1 {
		t.Errorf("error_count = %d after 3 identical applications, want 1", stage.ErrorCount)
	}
	if stage.ErrorMessage != "pi idle for 3m0s mid-generation" {
		t.Errorf("error_message = %q, want the recorded error", stage.ErrorMessage)
	}
}

func TestFinalizePrimitives_RejectEmptyIDs(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if err := store.SetStageMetrics(ctx, "", 0, 0, 0, 0, 0, 0, ""); err == nil {
		t.Error("SetStageMetrics accepted an empty stage id")
	}
	if err := store.SetStageStatus(ctx, "", StageStatusCompleted); err == nil {
		t.Error("SetStageStatus accepted an empty stage id")
	}
	if err := store.SetStageError(ctx, "", "boom"); err == nil {
		t.Error("SetStageError accepted an empty stage id")
	}
	if err := store.RecomputeChainAggregates(ctx, ""); err == nil {
		t.Error("RecomputeChainAggregates accepted an empty chain id")
	}
}
