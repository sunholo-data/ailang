package observatory

import (
	"context"
	"testing"
	"time"
)

// M-COMPLETION-PATH-PARITY M4 — closing the leak without inventing history.
//
// 315 production chains sat "active", the oldest for four months, because the
// cloud completion path advanced no chain. D3 (Mark, attended 2026-09-03): mark
// them abandoned, create no synthetic stage transitions.
//
// The arms that matter are the ones asserting what reconciliation must NOT touch.
// A cleanup that sweeps up a live chain is worse than the leak it fixes: a task
// can legitimately run for two hours under the current ceiling.

func seedReconcileChain(t *testing.T, store *Store, age time.Duration) string {
	t.Helper()
	ctx := context.Background()
	chain, err := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceMessage,
		SourceRef:  "inbox_reconcile_test",
	})
	if err != nil {
		t.Fatalf("create chain: %v", err)
	}
	if age > 0 {
		_, err := store.db.Exec(`UPDATE execution_chains SET created_at = ? WHERE id = ?`,
			time.Now().Add(-age), chain.ID)
		if err != nil {
			t.Fatalf("age chain: %v", err)
		}
	}
	return chain.ID
}

func seedReconcileStage(t *testing.T, store *Store, chainID string, status ChainStageStatus) string {
	t.Helper()
	ctx := context.Background()
	stage, err := store.CreateStage(ctx, &StageCreateRequest{
		ChainID: chainID, AgentID: "design-doc-creator", TaskID: "task-r", MessageID: "m",
	})
	if err != nil {
		t.Fatalf("create stage: %v", err)
	}
	if err := store.SetStageStatus(ctx, stage.ID, status); err != nil {
		t.Fatalf("set stage status: %v", err)
	}
	return stage.ID
}

func TestReconcile_AbandonsAStrandedChain(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	chainID := seedReconcileChain(t, store, 48*time.Hour)
	seedReconcileStage(t, store, chainID, StageStatusCompleted)

	n, err := store.ReconcileStrandedChains(ctx, time.Hour, AbandonReasonPreFix)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 1 {
		t.Fatalf("reconciled %d chains, want 1", n)
	}

	chain, err := store.GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("get chain: %v", err)
	}
	if chain.Status != ChainStatusAbandoned {
		t.Errorf("status = %q, want abandoned", chain.Status)
	}
	if chain.CompletedAt == nil {
		t.Error("an abandoned chain must be terminal, with a completed_at, or it keeps appearing in every future sweep")
	}
}

// TestReconcile_NeverAbandonsAChainWithLiveWork is the arm that protects real
// runs. Under the 2h task ceiling a chain can look idle for a long time and still
// be perfectly healthy.
func TestReconcile_NeverAbandonsAChainWithLiveWork(t *testing.T) {
	for _, live := range []ChainStageStatus{StageStatusPending, StageStatusRunning} {
		t.Run(string(live), func(t *testing.T) {
			db, store := setupTestDB(t)
			defer db.Close()
			ctx := context.Background()

			// A month-old chain, but with a stage that started just now: this is
			// live work on a long-lived chain and must never be swept.
			chainID := seedReconcileChain(t, store, 30*24*time.Hour)
			seedReconcileStage(t, store, chainID, live)
			touchStageStart(t, store, chainID, time.Now())

			n, err := store.ReconcileStrandedChains(ctx, time.Hour, AbandonReasonPreFix)
			if err != nil {
				t.Fatalf("reconcile: %v", err)
			}
			if n != 0 {
				t.Fatalf("reconciled %d chains with a %s stage; a live run was swept up", n, live)
			}

			chain, err := store.GetChain(ctx, chainID, ChainReadOptions{})
			if err != nil {
				t.Fatalf("get chain: %v", err)
			}
			if chain.Status != ChainStatusActive {
				t.Errorf("status = %q, want active — a chain with live work must be left alone", chain.Status)
			}
		})
	}
}

func TestReconcile_RespectsMinimumAge(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	chainID := seedReconcileChain(t, store, time.Minute)
	seedReconcileStage(t, store, chainID, StageStatusCompleted)

	n, err := store.ReconcileStrandedChains(ctx, time.Hour, AbandonReasonPreFix)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 0 {
		t.Errorf("reconciled a chain %v old against a 1h floor; a completion still in flight would be closed under it", time.Minute)
	}
	_ = chainID
}

// TestReconcile_CreatesNoStageTransitions is D3 made executable: the reconciler
// records that a chain stopped, never a verdict about what it did.
func TestReconcile_CreatesNoStageTransitions(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	chainID := seedReconcileChain(t, store, 48*time.Hour)
	stageID := seedReconcileStage(t, store, chainID, StageStatusFailed)

	if _, err := store.ReconcileStrandedChains(ctx, time.Hour, AbandonReasonPreFix); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	stage, err := store.GetStage(ctx, stageID)
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if stage.Status != StageStatusFailed {
		t.Errorf("stage status = %q, want failed — reconciliation rewrote history it did not observe", stage.Status)
	}

	chain, err := store.GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("get chain: %v", err)
	}
	if chain.Status == ChainStatusCompleted || chain.Status == ChainStatusFailed {
		t.Errorf("chain status = %q — abandoning must not assert an outcome nobody observed", chain.Status)
	}
	if chain.StagesCompleted != 0 {
		t.Errorf("stages_completed = %d; reconciliation backfilled a counter", chain.StagesCompleted)
	}
}

// TestAbandonChain_RequiresAReason: without one, an abandoned chain is later
// indistinguishable from one that genuinely ended.
func TestAbandonChain_RequiresAReason(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	chainID := seedReconcileChain(t, store, time.Hour)
	if err := store.AbandonChain(ctx, chainID, ""); err == nil {
		t.Error("a chain was abandoned with no reason recorded")
	}
	if err := store.AbandonChain(ctx, "", AbandonReasonPreFix); err == nil {
		t.Error("AbandonChain accepted an empty chain id")
	}
}

// TestAbandonChain_DoesNotOverrideARealVerdict: if something legitimately
// finished the chain between the scan and the write, that verdict wins.
func TestAbandonChain_DoesNotOverrideARealVerdict(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	chainID := seedReconcileChain(t, store, 48*time.Hour)
	if err := store.UpdateChainStatus(ctx, chainID, ChainStatusCompleted); err != nil {
		t.Fatalf("complete chain: %v", err)
	}

	if err := store.AbandonChain(ctx, chainID, AbandonReasonPreFix); err != nil {
		t.Fatalf("abandon: %v", err)
	}

	chain, err := store.GetChain(ctx, chainID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("get chain: %v", err)
	}
	if chain.Status != ChainStatusCompleted {
		t.Errorf("status = %q, want completed — reconciliation overwrote a real verdict", chain.Status)
	}
}

// TestReconcile_IsIdempotent: the recurring check runs forever, so a second pass
// must find nothing left to do.
func TestReconcile_IsIdempotent(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	chainID := seedReconcileChain(t, store, 48*time.Hour)
	seedReconcileStage(t, store, chainID, StageStatusCompleted)

	first, err := store.ReconcileStrandedChains(ctx, time.Hour, AbandonReasonPreFix)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := store.ReconcileStrandedChains(ctx, time.Hour, AbandonReasonPreFix)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first != 1 || second != 0 {
		t.Errorf("passes reconciled %d then %d, want 1 then 0", first, second)
	}
}

// touchStageStart sets a stage's started_at, so an arm can distinguish work in
// flight from the frozen pending records the broken completion path left behind.
func touchStageStart(t *testing.T, store *Store, chainID string, at time.Time) {
	t.Helper()
	if _, err := store.db.Exec(`UPDATE chain_stages SET started_at = ? WHERE chain_id = ?`, at, chainID); err != nil {
		t.Fatalf("touch stage start: %v", err)
	}
}

// TestReconcile_AFrozenPendingStageIsNotLiveWork is the arm that would have
// caught the first version of this reconciler.
//
// It reported "nothing to reconcile" against 311 known-dead prod chains, because
// it treated ANY pending stage as proof of life. Those chains were stranded
// precisely BECAUSE their stages were frozen at pending — the rule excluded the
// exact population it existed to find.
func TestReconcile_AFrozenPendingStageIsNotLiveWork(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	chainID := seedReconcileChain(t, store, 30*24*time.Hour)
	seedReconcileStage(t, store, chainID, StageStatusPending)
	// Frozen: started long before the live window.
	touchStageStart(t, store, chainID, time.Now().Add(-30*24*time.Hour))

	n, err := store.ReconcileStrandedChains(ctx, time.Hour, AbandonReasonPreFix)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 1 {
		t.Fatalf("reconciled %d chains; a stage pending for a month is a frozen record, not work in flight", n)
	}
}
