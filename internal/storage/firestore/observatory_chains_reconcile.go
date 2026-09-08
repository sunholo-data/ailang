package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	obs "github.com/sunholo-data/ailang/internal/observatory"
)

// Chain reconciliation on the production backend
// (M-COMPLETION-PATH-PARITY M4, Firestore half added 2026-09-04).
//
// The SQLite implementation shipped first and could not touch production, which
// runs Firestore — the same mistake a quorum reviewer caught earlier in this
// work: verifying the backend nobody runs and asserting the one everybody does.
//
// Measured in prod 2026-09-03: 400 chains, 311 of them "active", the oldest
// since 2026-04-27, none progressing. The cloud completion path advanced no
// chain, so once created nothing ever moved one. "Active chains" has therefore
// never been a usable health signal, only a count that grows.

// FindStrandedChains returns chains left active whose stages have all finished.
//
// Firestore has no NOT EXISTS, so this reads each candidate's stages rather than
// expressing the condition in one query. That is acceptable for a reconciliation
// pass over a few hundred rows and deliberately preferred over a denormalized
// flag, which would be one more thing that can silently disagree with reality.
//
// Conservative by design: a chain with ANY stage still pending or running is not
// stranded, however old. Under the 2h task ceiling a healthy run can look idle
// for a long time, and a cleanup that sweeps up live work is worse than the leak
// it fixes.
func (s *ObservatoryStore) FindStrandedChains(ctx context.Context, minAge time.Duration) ([]obs.StrandedChain, error) {
	cutoff := time.Now().Add(-minAge)

	// Filter on status only, then apply the age cutoff in Go.
	//
	// status + created_at together need a composite index, and this is a
	// maintenance pass over hundreds of rows — not a hot path worth an index and
	// a terraform change. If the active set ever reaches a scale where reading it
	// is expensive, that is itself the alarm: the whole point of this work is
	// that "active" stops being an unbounded pile.
	iter := s.client.Collection(collObsChains).
		Where("status", "==", string(obs.ChainStatusActive)).
		Documents(ctx)
	defer iter.Stop()

	var out []obs.StrandedChain
	now := time.Now()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing active chains: %w", err)
		}

		created := snapshotToTime(doc.Data(), "created_at")
		if !created.Before(cutoff) {
			continue
		}

		chainID := doc.Ref.ID
		live, err := s.hasLiveStage(ctx, chainID)
		if err != nil {
			return nil, err
		}
		if live {
			continue
		}

		out = append(out, obs.StrandedChain{
			ChainID:   chainID,
			CreatedAt: created,
			Age:       now.Sub(created),
		})
	}
	return out, nil
}

// hasLiveStage reports whether any stage could still be running — the condition
// that makes a chain NOT stranded.
func (s *ObservatoryStore) hasLiveStage(ctx context.Context, chainID string) (bool, error) {
	// Read the chain's stages and judge each one. A pending stage only counts as
	// life if it started within LiveStageWindow; older than that it is the frozen
	// record the broken completion path left behind, and treating it as life is
	// what made the first version of this reconciler report nothing to do against
	// 311 known-dead chains.
	cutoff := time.Now().Add(-obs.LiveStageWindow)

	it := s.client.Collection(collObsChainStages).
		Where("chain_id", "==", chainID).
		Documents(ctx)
	defer it.Stop()

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("checking live stages for chain %s: %w", chainID, err)
		}
		data := doc.Data()
		status := getString(data, "status")
		if status != string(obs.StageStatusPending) && status != string(obs.StageStatusRunning) {
			continue
		}
		if snapshotToTime(data, "started_at").After(cutoff) {
			return true, nil
		}
	}
}

// AbandonChain marks one chain terminal with an explicit reason, creating no
// stage transitions.
//
// Abandoned, never completed or failed: those assert an outcome, and 311 chains
// leaked without anyone recording one. Backfilling a verdict would invent history
// and poison the first dataset used to check whether the fix worked.
//
// The write is conditional on the chain still being active, inside a transaction:
// if something legitimately finished it between the scan and this write, that
// verdict wins over ours.
func (s *ObservatoryStore) AbandonChain(ctx context.Context, chainID, reason string) error {
	if chainID == "" {
		return fmt.Errorf("chain id is required")
	}
	if reason == "" {
		// A chain abandoned without a recorded reason is indistinguishable, later,
		// from one that genuinely ended — the ambiguity this exists to remove.
		return fmt.Errorf("abandoning a chain requires a reason")
	}

	ref := s.client.Doc(collObsChains, chainID)
	now := time.Now()

	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			return fmt.Errorf("reading chain %s: %w", chainID, err)
		}
		if getString(doc.Data(), "status") != string(obs.ChainStatusActive) {
			// Someone reached a real verdict first. Leave it.
			return nil
		}
		updates := []firestore.Update{
			{Path: "status", Value: string(obs.ChainStatusAbandoned)},
			{Path: "reason", Value: reason},
			{Path: "updated_at", Value: now},
		}
		if snapshotToTime(doc.Data(), "completed_at").IsZero() {
			updates = append(updates, firestore.Update{Path: "completed_at", Value: now})
		}
		return tx.Update(ref, updates)
	})
	if err != nil {
		return fmt.Errorf("failed to abandon chain %s: %w", chainID, err)
	}
	return nil
}
