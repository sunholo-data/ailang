package coordinator

import (
	"context"
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// countingObs records the two observatory writes that would put a shadow's
// cost into the fleet's chain totals.
type countingObs struct {
	observatory.Backend
	recompute, stageMetrics int
}

func (c *countingObs) RecomputeChainAggregates(ctx context.Context, chainID string) error {
	c.recompute++
	return c.Backend.RecomputeChainAggregates(ctx, chainID)
}

func (c *countingObs) SetStageMetrics(ctx context.Context, stageID string, cost float64, in, out, turns, tools int, ms int64, prov string) error {
	c.stageMetrics++
	return c.Backend.SetStageMetrics(ctx, stageID, cost, in, out, turns, tools, ms, prov)
}

func shadowCompletion(taskID, base string) *CloudStrategy {
	return &CloudStrategy{Completion: pubsub.TaskCompletion{
		TaskID: taskID, AgentID: "design-doc-creator", Status: "completed", ModelUsed: "glm",
		BaseCommit: base, HeadCommit: "head-" + taskID, ChangedFiles: []string{"fix.ail"},
		Diff: "diff --git a/fix.ail", CostUSD: 0.31, InputTokens: 900, OutputTokens: 80, NumTurns: 7,
		SessionID: "sess-" + taskID,
	}}
}

// addShadowChild creates the harness primary's shadow child, running, with no
// stage or chain — exactly as dispatchShadow makes it.
func (h *finalizeHarness) addShadowChild(t *testing.T) *TaskRecord {
	t.Helper()
	ctx := context.Background()
	child := &TaskRecord{ID: shadowTaskID(h.task.ID), ShadowOf: h.task.ID, Title: "x", Content: "x",
		AgentID: h.task.AgentID, Provider: "motoko", Status: TaskStatusPending}
	if err := h.store.CreateTask(ctx, child); err != nil {
		t.Fatal(err)
	}
	if _, err := h.store.CompareAndSetTaskStatus(ctx, child.ID, AllTaskStatuses(), TaskStatusRunning); err != nil {
		t.Fatal(err)
	}
	return child
}

func (h *finalizeHarness) finalizeTask(t *testing.T, task *TaskRecord, outcome CompletionOutcome, s ExecutionStrategy) *FinalizeReport {
	t.Helper()
	report, err := FinalizeTaskCompletion(context.Background(), h.deps, FinalizeInput{
		Task: task, Result: &ExecuteResult{Success: outcome == OutcomeCompleted, Cost: 0.31, InputTokens: 900},
		Outcome: outcome, BranchName: "coordinator/" + task.ID,
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

// A successful shadow completion ends Completed (not PendingApproval), creates
// no approval card and no handoff even on an auto-handoff agent, and never
// touches stage metrics or chain aggregates.
func TestFinalizeShadowHasNoFleetEffects(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(true))
	obs := &countingObs{Backend: h.obs}
	h.deps.ObsBackend = obs
	child := h.addShadowChild(t)

	h.finalizeTask(t, child, OutcomeCompleted, shadowCompletion(child.ID, "base-1"))

	got, err := h.store.GetTask(context.Background(), child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != TaskStatusCompleted {
		t.Errorf("shadow status = %s, want completed (pending_approval would queue it for approval)", got.Status)
	}
	if _, err := h.store.GetApprovalRequest(context.Background(), ApprovalIDForTask(child.ID)); err == nil {
		t.Error("a shadow completion created an approval card")
	}
	if n := len(h.handoffs(t)); n != 0 {
		t.Errorf("a shadow completion dispatched %d handoffs", n)
	}
	if obs.recompute != 0 || obs.stageMetrics != 0 {
		t.Errorf("shadow touched fleet metrics: recompute=%d stage_metrics=%d", obs.recompute, obs.stageMetrics)
	}
}

// Regression: the primary of a shadowed task finalizes exactly as before (card,
// handoff, PendingApproval), and also records its side of the comparison.
func TestFinalizeShadowedPrimaryUnchanged(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(true))
	h.addShadowChild(t)

	h.finalizeTask(t, h.task, OutcomeCompleted, shadowCompletion(h.task.ID, "base-1"))

	if got := h.taskStatus(t); got != TaskStatusPendingApproval {
		t.Errorf("primary status = %s, want pending_approval", got)
	}
	if h.approval(t) == nil {
		t.Error("primary lost its approval card")
	}
	if n := len(h.handoffs(t)); n == 0 {
		t.Error("primary lost its auto handoff")
	}
	c, err := h.store.GetShadowComparison(context.Background(), h.task.ID)
	if err != nil || c.Primary == nil || c.Primary.Diff == "" || c.Shadow != nil {
		t.Fatalf("primary side not recorded: %+v, %v", c, err)
	}
}

// Both sides land under the primary's ID, with every metric the report needs,
// and differing base commits set base_drift.
func TestFinalizeShadowComparisonBothSides(t *testing.T) {
	for _, tc := range []struct {
		name             string
		pBase, sBase     string
		wantDrift        bool
		shadowLandsFirst bool
	}{
		{"same base", "base-1", "base-1", false, false},
		{"drift", "base-1", "base-2", true, false},
		{"shadow first", "base-1", "base-1", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newFinalizeHarness(t, handoffAgent(false))
			child := h.addShadowChild(t)
			primary := func() { h.finalizeTask(t, h.task, OutcomeCompleted, shadowCompletion(h.task.ID, tc.pBase)) }
			shadow := func() { h.finalizeTask(t, child, OutcomeFailed, shadowCompletion(child.ID, tc.sBase)) }
			if tc.shadowLandsFirst {
				shadow()
				primary()
			} else {
				primary()
				shadow()
			}
			c, err := h.store.GetShadowComparison(context.Background(), h.task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !c.Complete() || c.BaseDrift != tc.wantDrift {
				t.Fatalf("complete=%v drift=%v, want true/%v", c.Complete(), c.BaseDrift, tc.wantDrift)
			}
			s := c.Shadow
			if s.TaskID != child.ID || s.Provider != "motoko" || s.CostUSD != 0.31 || s.InputTokens != 900 ||
				s.NumTurns != 7 || s.SessionID != "sess-"+child.ID || s.Diff == "" || s.BaseCommit != tc.sBase {
				t.Errorf("shadow side incomplete: %+v", s)
			}
		})
	}
}

// A task with no shadow child records nothing, and the lookup for an unknown
// parent says not-found rather than returning an empty comparison.
func TestFinalizeUnshadowedTaskRecordsNothing(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))
	report := h.finalizeTask(t, h.task, OutcomeCompleted, shadowCompletion(h.task.ID, "b"))
	for _, e := range report.Applied {
		if e == EffectShadowRecord {
			t.Fatal("shadow_record applied to a task with no shadow")
		}
	}
	if _, err := h.store.GetShadowComparison(context.Background(), h.task.ID); !errors.Is(err, ErrShadowComparisonNotFound) {
		t.Fatalf("err = %v, want ErrShadowComparisonNotFound", err)
	}
}

// A redelivered shadow completion rewrites its side; it does not duplicate or
// lose the other side.
func TestShadowRunRecordIsIdempotent(t *testing.T) {
	h := newFinalizeHarness(t, nil)
	ctx := context.Background()
	run := ShadowRunResult{TaskID: "p-shadow", Status: "completed", BaseCommit: "b"}
	if _, err := h.store.RecordShadowRun(ctx, "p", ShadowSidePrimary, ShadowRunResult{TaskID: "p", BaseCommit: "b"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := h.store.RecordShadowRun(ctx, "p", ShadowSideShadow, run); err != nil {
			t.Fatal(err)
		}
	}
	c, err := h.store.GetShadowComparison(ctx, "p")
	if err != nil || c.Primary == nil || c.Shadow == nil || c.Shadow.TaskID != "p-shadow" {
		t.Fatalf("after redelivery: %+v, %v", c, err)
	}
	if _, err := h.store.RecordShadowRun(ctx, "p", "sideways", run); err == nil {
		t.Fatal("unknown side accepted")
	}
}

// A task that merely carries the `-shadow` suffix, without ShadowOf pointing
// back, is not a shadow: the primary records nothing.
func TestFinalizeLookalikeChildIsNotAShadow(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))
	if err := h.store.CreateTask(context.Background(), &TaskRecord{ID: shadowTaskID(h.task.ID), Title: "x", Content: "x"}); err != nil {
		t.Fatal(err)
	}
	h.finalizeTask(t, h.task, OutcomeCompleted, shadowCompletion(h.task.ID, "b"))
	if _, err := h.store.GetShadowComparison(context.Background(), h.task.ID); !errors.Is(err, ErrShadowComparisonNotFound) {
		t.Fatalf("lookalike recorded a comparison: %v", err)
	}
}
