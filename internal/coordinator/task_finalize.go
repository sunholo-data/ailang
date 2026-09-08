package coordinator

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// One finalisation path for both executors (M-COMPLETION-PATH-PARITY M1).
//
// The coordinator had two completion paths. The daemon path performed ten side
// effects; the cloud path — the only one production runs — performed two. The
// difference was entirely dead in production: agent handoffs, chain and stage
// progression, and the approval record itself. The configured pipeline
// design-doc-creator -> sprint-planner -> sprint-executor -> sprint-evaluator has
// therefore never once advanced past its first stage.
//
// This orchestrator is what both callers run, so the two cannot drift apart
// again. Its behaviour is specified by the outcome matrix below, which is
// characterised against the daemon path in completion_matrix_test.go — the path
// that works today, and whose behaviour this must preserve exactly.
//
// Two properties do the safety work, because cross-store atomicity is not
// available on either path:
//
//   - Every effect is individually idempotent (M0b), so replaying one is a no-op.
//   - Every status write is compare-and-set, so a stale replay cannot regress a
//     record another step has legitimately advanced.
//
// The ledger records progress on top of those; it does not promise atomicity, and
// nothing here depends on it doing so.

// StrategyKind distinguishes the executors. The orchestrator branches on it
// explicitly rather than inferring from an empty field: a cloud strategy whose
// cleanup method silently did nothing would be the same silent fallback this
// design exists to remove, dressed as an interface.
type StrategyKind int

const (
	StrategyKindDaemon StrategyKind = iota
	StrategyKindCloud
)

func (k StrategyKind) String() string {
	if k == StrategyKindCloud {
		return "cloud"
	}
	return "daemon"
}

// ExecutionStrategy supplies what genuinely differs between executors.
type ExecutionStrategy interface {
	Kind() StrategyKind
	// DiffSource returns the approval diff. The daemon reads its worktree; the
	// cloud path reads the two commit SHAs the executor publishes (M3). Returning
	// ErrNoDiffSource is explicit and is recorded on the approval as
	// diff_unavailable — never rendered as a confident "Files (0)", which is the
	// failure that had merges approved blind.
	DiffSource(ctx context.Context, task *TaskRecord) (DiffResult, error)
}

// ErrNoDiffSource says this executor cannot produce a diff for this task.
var ErrNoDiffSource = fmt.Errorf("no diff source available for this task")

// DiffResult is the approval card's evidence.
type DiffResult struct {
	Stat         string
	ChangedFiles []string
	Patch        string
}

// CompletionOutcome is what the run actually produced. It is not a synonym for
// the task's next status: no_changes exists precisely because "it ran and changed
// nothing" and "it succeeded" had been reported by the same value.
type CompletionOutcome string

const (
	OutcomeCompleted CompletionOutcome = "completed"
	OutcomeNoChanges CompletionOutcome = "no_changes"
	OutcomeFailed    CompletionOutcome = "failed"
)

// FinalizeDeps are the stores and collaborators finalisation writes through.
type FinalizeDeps struct {
	TaskStore     Store
	MsgStore      messaging.MessageStore
	ObsBackend    observatory.Backend
	AgentRegistry *AgentRegistry
	Logger        *log.Logger
	// Owner identifies this coordinator instance on ledger claims, so a stalled
	// effect can name who stalled rather than only that something did.
	Owner string
}

// FinalizeInput carries everything the effects need.
//
// The payload travels WITH the claim rather than being re-derived later: the
// reconciliation sweep reads only the database and has no completion payload, so
// without this it could never re-apply an effect it took over.
type FinalizeInput struct {
	Task       *TaskRecord
	Result     *ExecuteResult
	Outcome    CompletionOutcome
	BranchName string
	// SkipApproval mirrors the agent's configuration. It selects between the two
	// success columns of the matrix.
	SkipApproval bool
}

// FinalizeReport is what happened, for the caller to log. Every effect appears,
// including the ones deliberately not run for this outcome, so a reader can tell
// "not applicable" from "silently skipped".
type FinalizeReport struct {
	Ledger  FinalizationLedger
	Applied []string
	Skipped []string
}

func (d *FinalizeDeps) logf(format string, args ...interface{}) {
	if d.Logger != nil {
		d.Logger.Printf(format, args...)
	}
}

// FinalizeTaskCompletion performs every effect a finished task must have, exactly
// once, regardless of which executor ran it.
//
// Effects are outcome-conditional, per the matrix. They are NOT unconditional:
// creating an approval or dispatching a handoff for a failed task would start the
// next agent on work that errored, which is the specific hazard that makes
// unsupervised chaining dangerous.
func FinalizeTaskCompletion(ctx context.Context, deps *FinalizeDeps, in FinalizeInput, strategy ExecutionStrategy) (*FinalizeReport, error) {
	if deps == nil || deps.TaskStore == nil {
		return nil, fmt.Errorf("FinalizeTaskCompletion requires a task store")
	}
	if in.Task == nil || in.Task.ID == "" {
		return nil, fmt.Errorf("FinalizeTaskCompletion requires a task with an id")
	}

	ledger, err := deps.TaskStore.GetTaskFinalization(ctx, in.Task.ID)
	if err != nil {
		// Proceeding with an unknown ledger would re-run every effect while
		// believing none had run. The effects are idempotent, so that is survivable
		// — but it is not something to do silently.
		return nil, fmt.Errorf("cannot finalize %s without its ledger: %w", in.Task.ID, err)
	}

	f := &finalizer{deps: deps, in: in, strategy: strategy, ledger: ledger, report: &FinalizeReport{}}

	// Order matters only for legibility; each effect stands alone.
	f.run(EffectTaskStatus, true, f.applyTaskStatus)
	f.run(EffectStageStatus, in.Task.StageID != "", f.applyStageStatus)
	f.run(EffectStageSession, in.Task.StageID != "" && in.Result != nil && in.Result.SessionID != "", f.applyStageSession)
	f.run(EffectMetrics, in.Result != nil, f.applyMetrics)
	f.run(EffectStageError, in.Outcome == OutcomeFailed && in.Task.StageID != "", f.applyStageError)
	f.run(EffectChainStatus, in.Task.ChainID != "", f.applyChainStatus)
	f.run(EffectApproval, f.wantsApproval(), f.applyApproval)
	f.run(EffectHandoff, f.wantsHandoff(), f.applyHandoff)

	f.report.Ledger = f.ledger
	if err := deps.TaskStore.SetTaskFinalization(ctx, in.Task.ID, f.ledger); err != nil {
		// The effects landed; only the record of them failed. Say so plainly —
		// the consequence is a redelivery re-applying idempotent writes, not lost
		// work.
		deps.logf("finalize %s: effects applied but the ledger could not be written (a replay will re-apply them harmlessly): %v", in.Task.ID, err)
	}
	return f.report, nil
}

type finalizer struct {
	deps     *FinalizeDeps
	in       FinalizeInput
	strategy ExecutionStrategy
	ledger   FinalizationLedger
	report   *FinalizeReport
	ctx      context.Context
}

// run applies one effect under the ledger: claim, apply, resolve.
//
// applicable is the matrix cell. An effect that does not apply to this outcome is
// recorded as skipped rather than passed over quietly, so the report can
// distinguish "not applicable here" from "should have run and did not".
func (f *finalizer) run(effect string, applicable bool, apply func(context.Context) (FinalizationState, error)) {
	if !applicable {
		f.report.Skipped = append(f.report.Skipped, effect)
		return
	}
	if f.ledger.IsDone(effect) {
		f.report.Skipped = append(f.report.Skipped, effect+" (already done)")
		return
	}
	if f.ledger.IsExhausted(effect) {
		f.deps.logf("finalize %s: effect %s is exhausted after %d attempts and will not be retried", f.in.Task.ID, effect, MaxFinalizationAttempts)
		f.report.Skipped = append(f.report.Skipped, effect+" (exhausted)")
		return
	}

	now := time.Now()
	f.ledger = f.ledger.Claim(effect, f.deps.Owner, now)

	ctx := f.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	state, err := apply(ctx)
	if err != nil {
		attempt := f.ledger[effect].Attempt
		if attempt >= MaxFinalizationAttempts {
			// Terminal and visible. An effect that cannot be applied must never
			// be silently dropped.
			f.ledger = f.ledger.Resolve(effect, FinalizationFailed, time.Now(), err.Error())
			f.deps.logf("finalize %s: effect %s FAILED terminally after %d attempts: %v", f.in.Task.ID, effect, attempt, err)
		} else {
			f.ledger = f.ledger.Resolve(effect, FinalizationPending, time.Now(), err.Error())
			f.deps.logf("finalize %s: effect %s failed (attempt %d/%d), will retry: %v", f.in.Task.ID, effect, attempt, MaxFinalizationAttempts, err)
		}
		return
	}

	f.ledger = f.ledger.Resolve(effect, state, time.Now(), "")
	if state == FinalizationSuperseded {
		f.report.Skipped = append(f.report.Skipped, effect+" (superseded)")
		f.deps.logf("finalize %s: effect %s superseded — the record advanced past this completion", f.in.Task.ID, effect)
		return
	}
	f.report.Applied = append(f.report.Applied, effect)
}

// nextTaskStatus is matrix row 1.
func (f *finalizer) nextTaskStatus() TaskStatus {
	switch f.in.Outcome {
	case OutcomeFailed:
		return TaskStatusFailed
	case OutcomeNoChanges:
		// D5 (Mark, attended 2026-09-03): terminal, and nothing follows.
		return TaskStatusNoChanges
	default:
		if f.in.SkipApproval {
			return TaskStatusCompleted
		}
		return TaskStatusPendingApproval
	}
}

func (f *finalizer) applyTaskStatus(ctx context.Context) (FinalizationState, error) {
	applied, err := f.deps.TaskStore.CompareAndSetTaskStatus(ctx, f.in.Task.ID, FinalizableFrom(), f.nextTaskStatus())
	if err != nil {
		return FinalizationPending, err
	}
	if !applied {
		return FinalizationSuperseded, nil
	}
	return FinalizationDone, nil
}

func (f *finalizer) applyStageStatus(ctx context.Context) (FinalizationState, error) {
	if f.deps.ObsBackend == nil {
		return FinalizationPending, fmt.Errorf("no observatory backend")
	}
	var status observatory.ChainStageStatus
	switch f.in.Outcome {
	case OutcomeFailed:
		status = observatory.StageStatusFailed
	case OutcomeNoChanges:
		status = observatory.StageStatusCompleted
	default:
		if f.in.SkipApproval {
			status = observatory.StageStatusCompleted
		} else {
			status = observatory.StageStatusAwaitingApproval
		}
	}
	if err := f.deps.ObsBackend.SetStageStatus(ctx, f.in.Task.StageID, status); err != nil {
		return FinalizationPending, err
	}
	return FinalizationDone, nil
}

func (f *finalizer) applyStageSession(ctx context.Context) (FinalizationState, error) {
	if f.deps.ObsBackend == nil {
		return FinalizationPending, fmt.Errorf("no observatory backend")
	}
	if err := f.deps.ObsBackend.UpdateStageSession(ctx, f.in.Task.StageID, f.in.Result.SessionID); err != nil {
		return FinalizationPending, err
	}
	return FinalizationDone, nil
}

// applyMetrics writes the stage's own values absolutely, then DERIVES the chain
// totals from the stage rows.
//
// A failed run still burned tokens, so this applies to every outcome — the daemon
// path has always done so, and dropping it would make failures free in the cost
// rollup.
func (f *finalizer) applyMetrics(ctx context.Context) (FinalizationState, error) {
	if f.deps.ObsBackend == nil {
		return FinalizationPending, fmt.Errorf("no observatory backend")
	}
	r := f.in.Result
	if f.in.Task.StageID != "" {
		err := f.deps.ObsBackend.SetStageMetrics(ctx, f.in.Task.StageID,
			r.Cost, r.InputTokens, r.OutputTokens, r.NumTurns, r.ToolCallCount,
			r.Duration.Milliseconds(), r.CostProvenance)
		if err != nil {
			return FinalizationPending, err
		}
	}
	if f.in.Task.ChainID != "" {
		if err := f.deps.ObsBackend.RecomputeChainAggregates(ctx, f.in.Task.ChainID); err != nil {
			return FinalizationPending, err
		}
	}
	return FinalizationDone, nil
}

func (f *finalizer) applyStageError(ctx context.Context) (FinalizationState, error) {
	if f.deps.ObsBackend == nil {
		return FinalizationPending, fmt.Errorf("no observatory backend")
	}
	msg := "task failed"
	if f.in.Result != nil && f.in.Result.Error != "" {
		msg = f.in.Result.Error
	}
	if err := f.deps.ObsBackend.SetStageError(ctx, f.in.Task.StageID, msg); err != nil {
		return FinalizationPending, err
	}
	return FinalizationDone, nil
}

func (f *finalizer) applyChainStatus(ctx context.Context) (FinalizationState, error) {
	if f.deps.ObsBackend == nil {
		return FinalizationPending, fmt.Errorf("no observatory backend")
	}
	var status observatory.ChainStatus
	switch f.in.Outcome {
	case OutcomeFailed:
		status = observatory.ChainStatusFailed
	case OutcomeNoChanges:
		// D5: terminal. A chain left active is the leak this work closes — 315 of
		// them, the oldest four months old.
		status = observatory.ChainStatusCompleted
	default:
		if f.in.SkipApproval {
			status = observatory.ChainStatusCompleted
		} else {
			status = observatory.ChainStatusPendingApproval
		}
	}
	if err := f.deps.ObsBackend.UpdateChainStatus(ctx, f.in.Task.ChainID, status); err != nil {
		return FinalizationPending, err
	}
	return FinalizationDone, nil
}

// wantsApproval is matrix row 3: an approval exists only for a normal successful
// completion. Never for a failure, never for no_changes (there is nothing to
// approve), and never when the agent skips approval.
func (f *finalizer) wantsApproval() bool {
	return f.in.Outcome == OutcomeCompleted && !f.in.SkipApproval
}

// wantsHandoff is matrix row 5.
//
// handleAgentHandoffs runs after the approval branch and inside the success arm
// of the daemon path, so auto-approved edges dispatch at COMPLETION for both
// success sub-branches — not on approval. Only non-auto targets are embedded in
// the merge approval and wait for it. An earlier revision of the design read this
// as "on approval" and would have changed live behaviour while claiming to
// preserve it.
//
// Never for a failure: the next agent would build on an error.
func (f *finalizer) wantsHandoff() bool {
	if f.in.Outcome != OutcomeCompleted {
		return false
	}
	return len(f.autoHandoffTargets()) > 0
}

// autoHandoffTargets returns the edges configured to dispatch without approval.
//
// The topology comes from the agent registry — never from model output and never
// from the sending message. That is the property unsupervised chaining depends
// on, and it already holds.
func (f *finalizer) autoHandoffTargets() []string {
	if f.deps.AgentRegistry == nil || f.in.Task.AgentID == "" {
		return nil
	}
	agent := f.deps.AgentRegistry.GetAgentByID(f.in.Task.AgentID)
	if agent == nil {
		return nil
	}
	var targets []string
	for _, tgt := range agent.TriggerOnComplete {
		if agent.AutoApproveHandoffs || agent.AutoApprovesHandoffTo(tgt) {
			targets = append(targets, tgt)
		}
	}
	return targets
}
