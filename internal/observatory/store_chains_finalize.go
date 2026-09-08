package observatory

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrChainWritesUnsupported is returned by backends that do not hold the chain
// hierarchy (Cloud Trace holds spans; Jaeger holds traces).
//
// The existing Update* methods on those backends return nil and do nothing — a
// silent fallback that would erase the chain model with no signal if such a
// backend were ever selected (design doc V20). That is pre-existing and out of
// scope to change here, but the finalisation writes must not join it: a
// finalisation that silently does nothing is exactly the class of defect this
// milestone exists to remove. Callers log this rather than aborting, so the
// failure is visible without taking the daemon down.
var ErrChainWritesUnsupported = errors.New("observatory backend does not support chain/stage writes")

// Idempotent chain/stage writes for task finalisation (M-COMPLETION-PATH-PARITY M0b).
//
// Finalisation is replayed: Pub/Sub push is at-least-once with a 60s ack deadline,
// and the coordinator's terminal-state guard does not cover pending_approval. So
// every write finalisation performs must be safe to apply twice.
//
// The existing Update* family is NOT safe that way, and it took a quorum round to
// establish which parts were which:
//
//   - UpdateStageMetrics  accumulates (`cost = cost + ?`), so a replay double-counts.
//   - UpdateChainMetrics  accumulates likewise.
//   - UpdateStageStatus   writes the status absolutely, but ALSO increments the
//     chain's stages_completed counter as a side effect — so it is not idempotent
//     either, which the design doc's matrix initially got wrong.
//   - UpdateStageError    increments error_count.
//
// Those functions keep their accumulating semantics for their existing callers
// (importers, evaluators) — this file adds absolute-write counterparts for the
// finalisation path rather than changing behaviour under anyone's feet.
//
// The chain-level aggregates are DERIVED, never set from a caller-supplied delta.
// That is deliberate: an application-level read-modify-write ("read the total, add
// my share, write it back") reproduces the double-count across a crash, and a
// read-then-write in two statements lets two concurrent finalizers compute
// different snapshots with the stale one landing last. Deriving the totals inside
// a single UPDATE removes both failure modes — the value is a pure function of the
// stage rows, so it does not matter who writes it, how often, or in what order.

// SetStageMetrics writes a stage's metrics ABSOLUTELY, replacing whatever is there.
//
// Safe to replay: the values belong to one finished run and do not change, so a
// second application is a no-op. Compare UpdateStageMetrics, which accumulates.
//
// costProvenance keeps the same "first non-empty label wins" rule as the
// accumulating version: a caller that cannot classify the cost passes "" and must
// not erase a label an earlier caller established.
func (s *Store) SetStageMetrics(ctx context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64, costProvenance string) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE chain_stages
		SET cost = ?,
		    tokens_in = ?,
		    tokens_out = ?,
		    turns = ?,
		    tool_calls = ?,
		    duration_ms = ?,
		    cost_provenance = COALESCE(NULLIF(cost_provenance, ''), NULLIF(?, ''))
		WHERE id = ?
	`, cost, tokensIn, tokensOut, turns, toolCalls, durationMs, costProvenance, stageID)
	if err != nil {
		return fmt.Errorf("failed to set stage metrics: %w", err)
	}
	return nil
}

// SetStageStatus writes a stage's status with NO chain-counter side effect.
//
// UpdateStageStatus increments execution_chains.stages_completed when a stage
// completes. That increment is why the status write is not replay-safe. Here the
// counter is left alone and recomputed by RecomputeChainAggregates, which derives
// it from the stage rows instead.
func (s *Store) SetStageStatus(ctx context.Context, stageID string, status ChainStageStatus) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}

	now := time.Now()
	var completedAt interface{}
	if status == StageStatusCompleted || status == StageStatusFailed {
		completedAt = now
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE chain_stages
		SET status = ?,
		    completed_at = COALESCE(?, completed_at)
		WHERE id = ?
	`, status, completedAt, stageID)
	if err != nil {
		return fmt.Errorf("failed to set stage status: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("stage not found: %s", stageID)
	}
	return nil
}

// SetStageError records a stage's error message and marks error_count as 1 rather
// than incrementing it, so a replayed failure does not inflate the count.
func (s *Store) SetStageError(ctx context.Context, stageID, errorMessage string) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE chain_stages
		SET error_message = ?, error_count = 1
		WHERE id = ?
	`, errorMessage, stageID)
	if err != nil {
		return fmt.Errorf("failed to set stage error: %w", err)
	}
	return nil
}

// RecomputeChainAggregates derives a chain's totals from its stage rows in a
// single UPDATE — atomic by construction, and a pure function of the stages, so
// concurrent or repeated calls converge on the same value.
//
// It also recomputes stages_completed, which is otherwise maintained by an
// increment inside UpdateStageStatus. That counter is what the chain list view
// reports, so an increment lost or doubled shows up directly as a wrong stage
// count on the dashboard.
func (s *Store) RecomputeChainAggregates(ctx context.Context, chainID string) error {
	if chainID == "" {
		return fmt.Errorf("chain id is required")
	}

	// One statement: scalar subqueries rather than row-value assignment, which
	// needs SQLite >= 3.15 and buys nothing here.
	_, err := s.db.ExecContext(ctx, `
		UPDATE execution_chains
		SET total_cost        = (SELECT COALESCE(SUM(cost), 0) FROM chain_stages WHERE chain_id = ?),
		    total_tokens      = (SELECT COALESCE(SUM(tokens_in) + SUM(tokens_out), 0) FROM chain_stages WHERE chain_id = ?),
		    total_turns       = (SELECT COALESCE(SUM(turns), 0) FROM chain_stages WHERE chain_id = ?),
		    stages_completed  = (SELECT COUNT(*) FROM chain_stages WHERE chain_id = ? AND status = 'completed'),
		    updated_at        = ?
		WHERE id = ?
	`, chainID, chainID, chainID, chainID, time.Now(), chainID)
	if err != nil {
		return fmt.Errorf("failed to recompute chain aggregates: %w", err)
	}
	return nil
}
