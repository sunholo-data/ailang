package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	obs "github.com/sunholo-data/ailang/internal/observatory"
)

// Idempotent chain/stage writes for task finalisation (M-COMPLETION-PATH-PARITY M0b).
//
// This is the production path: the cloud coordinator's observatory, coordinator
// and messaging stores all share one Firestore client, and finalisation here is
// replayed because Pub/Sub push is at-least-once.
//
// The backends do NOT behave alike, which is why each of these is written out
// rather than assumed to mirror SQLite:
//
//   - UpdateStageMetrics is already an absolute set here, while its SQLite
//     counterpart accumulates. The two paths have been recording different stage
//     metrics semantics all along.
//   - UpdateChainMetrics uses firestore.Increment, so it DOES double-count on
//     replay, exactly like SQLite.
//   - UpdateStageStatus increments the chain's stages_completed counter, same as
//     SQLite — and this backend's ListChains reports that denormalized counter
//     rather than counting stage documents, so a lost or doubled increment shows
//     up directly as a wrong stage count on the dashboard.

// SetStageStatus writes a stage's status with no chain-counter side effect.
// RecomputeChainAggregates owns stages_completed instead, deriving it from the
// stage documents.
func (s *ObservatoryStore) SetStageStatus(ctx context.Context, stageID string, stageStatus obs.ChainStageStatus) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}

	updates := []firestore.Update{
		{Path: "status", Value: string(stageStatus)},
	}
	if stageStatus == obs.StageStatusCompleted || stageStatus == obs.StageStatusFailed {
		updates = append(updates, firestore.Update{Path: "completed_at", Value: time.Now()})
	}

	if _, err := s.client.Doc(collObsChainStages, stageID).Update(ctx, updates); err != nil {
		return fmt.Errorf("failed to set stage status: %w", err)
	}
	return nil
}

// SetStageMetrics writes a stage's metrics absolutely.
//
// The provenance rule matches SQLite's COALESCE(NULLIF(...)): a caller that
// cannot classify the cost passes "" and must not erase a label an earlier
// caller established.
func (s *ObservatoryStore) SetStageMetrics(ctx context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64, costProvenance string) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}

	updates := []firestore.Update{
		{Path: "cost", Value: cost},
		{Path: "tokens_in", Value: tokensIn},
		{Path: "tokens_out", Value: tokensOut},
		{Path: "turns", Value: turns},
		{Path: "tool_calls", Value: toolCalls},
		{Path: "duration_ms", Value: durationMs},
	}
	if costProvenance != "" {
		updates = append(updates, firestore.Update{Path: "cost_provenance", Value: costProvenance})
	}

	if _, err := s.client.Doc(collObsChainStages, stageID).Update(ctx, updates); err != nil {
		return fmt.Errorf("failed to set stage metrics: %w", err)
	}
	return nil
}

// SetStageError records a stage's error with error_count set to 1 rather than
// incremented, so a replayed failure does not inflate it.
func (s *ObservatoryStore) SetStageError(ctx context.Context, stageID, errorMessage string) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}

	_, err := s.client.Doc(collObsChainStages, stageID).Update(ctx, []firestore.Update{
		{Path: "error_message", Value: errorMessage},
		{Path: "error_count", Value: 1},
	})
	if err != nil {
		return fmt.Errorf("failed to set stage error: %w", err)
	}
	return nil
}

// RecomputeChainAggregates derives a chain's totals and stages_completed from its
// stage documents, inside a transaction.
//
// Firestore has no equivalent of SQLite's single UPDATE … (SELECT SUM …), so the
// read of the stages and the write of the chain must be one atomic unit — this is
// the only transaction in the finalisation design. Without it, two concurrent
// finalizers could compute different snapshots and the stale one land last, which
// is the same double-count by a slower route.
//
// Because the totals are a pure function of the stage documents, the result does
// not depend on who runs this, how often, or in what order — and it repairs
// chains the incrementing path has already inflated.
func (s *ObservatoryStore) RecomputeChainAggregates(ctx context.Context, chainID string) error {
	if chainID == "" {
		return fmt.Errorf("chain id is required")
	}

	chainRef := s.client.Doc(collObsChains, chainID)
	stageQuery := s.client.Collection(collObsChainStages).Where("chain_id", "==", chainID)

	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// All reads before any write — a Firestore transaction requirement.
		iter := tx.Documents(stageQuery)
		defer iter.Stop()

		var totalCost float64
		var totalTokens, totalTurns, stagesCompleted int

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("reading stages for chain %s: %w", chainID, err)
			}
			data := doc.Data()
			totalCost += getFloat64(data, "cost")
			totalTokens += getInt(data, "tokens_in") + getInt(data, "tokens_out")
			totalTurns += getInt(data, "turns")
			if getString(data, "status") == string(obs.StageStatusCompleted) {
				stagesCompleted++
			}
		}

		return tx.Update(chainRef, []firestore.Update{
			{Path: "total_cost", Value: totalCost},
			{Path: "total_tokens", Value: totalTokens},
			{Path: "total_turns", Value: totalTurns},
			{Path: "stages_completed", Value: stagesCompleted},
			{Path: "updated_at", Value: time.Now()},
		})
	})
	if err != nil {
		return fmt.Errorf("failed to recompute chain aggregates: %w", err)
	}
	return nil
}
