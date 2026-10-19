package coordinator

import (
	"context"
	"fmt"
	"time"
)

// Shadow finalization (M-AILANG-LANE-EXECUTOR-SHADOW M4). Both sides of a
// comparison record their completion under the primary's ID: a shadow child
// because it has ShadowOf, and a primary because a `<id>-shadow` child exists.
// The effect is ledgered like any other, so a redelivered completion cannot lose
// a side and an idempotent rewrite is harmless.

// shadowRecordSide returns the comparison this task's completion belongs to,
// or ok=false when it has none.
func (f *finalizer) shadowRecordSide(ctx context.Context) (parentID, side string, ok bool) {
	if f.in.Task.ShadowOf != "" {
		return f.in.Task.ShadowOf, ShadowSideShadow, true
	}
	child, err := f.deps.TaskStore.GetTask(ctx, shadowTaskID(f.in.Task.ID))
	if err != nil || child == nil || child.ShadowOf != f.in.Task.ID {
		return "", "", false
	}
	return f.in.Task.ID, ShadowSidePrimary, true
}

// wantsShadowRecord: only the cloud path carries the completion payload the
// record is built from, and shadows are cloud-only in v1.
func (f *finalizer) wantsShadowRecord(ctx context.Context) bool {
	if _, cloud := f.strategy.(*CloudStrategy); !cloud {
		return false
	}
	_, _, ok := f.shadowRecordSide(ctx)
	return ok
}

func (f *finalizer) applyShadowRecord(ctx context.Context) (FinalizationState, error) {
	store, ok := f.deps.TaskStore.(ShadowComparisonStore)
	if !ok {
		return FinalizationPending, fmt.Errorf("task store %T cannot hold shadow comparisons", f.deps.TaskStore)
	}
	parentID, side, ok := f.shadowRecordSide(ctx)
	if !ok {
		return FinalizationSuperseded, nil
	}
	c := f.strategy.(*CloudStrategy).Completion
	run := ShadowRunResult{
		TaskID: f.in.Task.ID, AgentID: c.AgentID, Provider: f.in.Task.Provider, Model: c.ModelUsed,
		Status: c.Status, BaseCommit: c.BaseCommit, HeadCommit: c.HeadCommit,
		ChangedFiles: c.ChangedFiles, DiffStat: c.DiffStat, Diff: c.Diff,
		CostUSD: c.CostUSD, InputTokens: c.InputTokens, OutputTokens: c.OutputTokens,
		NumTurns: c.NumTurns, ToolCallCount: c.ToolCallCount, DurationMS: int64(c.DurationMS),
		SessionID: c.SessionID, ArtifactGCSPath: c.ArtifactGCSPath, ErrorMsg: c.ErrorMsg,
		CompletedAt: time.Now().UTC(),
	}
	if _, err := store.RecordShadowRun(ctx, parentID, side, run); err != nil {
		return FinalizationPending, err
	}
	return FinalizationDone, nil
}
