package observatory

import (
	"context"
	"fmt"
	"time"
)

// Chain reconciliation (M-COMPLETION-PATH-PARITY M4).
//
// Measured in production 2026-09-03: 404 chains, 315 of them "active", the oldest
// since 2026-04-27. None was progressing. The cloud completion path advanced no
// chain and no stage, so once a chain was created nothing ever moved it — which
// also means "active chains" has never been a usable health signal, only a
// monotonically growing count.
//
// Two things happen here, and the distinction matters:
//
//   - The one-shot pass closes the backlog. It marks chains ABANDONED, never
//     completed or failed, and creates NO stage transitions. Backfilling a
//     verdict would invent history that nobody observed and would poison the
//     first dataset used to check whether the fix worked (D3, Mark, attended
//     2026-09-03).
//   - The recurring check stops the class recurring. A chain still active while
//     its task is terminal is now a detectable condition rather than something
//     that has to be noticed by hand four months later.

// AbandonReasonPreFix is recorded on chains stranded by the pre-fix cloud path.
const AbandonReasonPreFix = "stranded before M-COMPLETION-PATH-PARITY: the cloud completion path performed no chain progression"

// StrandedChain is a chain that cannot progress.
type StrandedChain struct {
	ChainID   string
	CreatedAt time.Time
	Age       time.Duration
}

// FindStrandedChains returns chains left active whose stages have all finished,
// or which have no stages at all and are older than minAge.
//
// It is deliberately conservative: a chain with a stage still running is NOT
// stranded, however old, because a legitimately long task must never be swept up
// by a cleanup. The 2h task ceiling means a genuine run can look idle for a long
// time.
func (s *Store) FindStrandedChains(ctx context.Context, minAge time.Duration) ([]StrandedChain, error) {
	cutoff := time.Now().Add(-minAge)

	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.created_at
		FROM execution_chains c
		WHERE c.status = ?
		  AND c.created_at < ?
		  AND NOT EXISTS (
			SELECT 1 FROM chain_stages s
			WHERE s.chain_id = c.id
			  AND s.status IN ('pending', 'running')
		  )
		ORDER BY c.created_at ASC
	`, string(ChainStatusActive), cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to find stranded chains: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []StrandedChain
	now := time.Now()
	for rows.Next() {
		var c StrandedChain
		if err := rows.Scan(&c.ChainID, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan stranded chain: %w", err)
		}
		c.Age = now.Sub(c.CreatedAt)
		out = append(out, c)
	}
	return out, rows.Err()
}

// AbandonChain marks one chain terminal with an explicit reason, and creates no
// stage transitions.
func (s *Store) AbandonChain(ctx context.Context, chainID, reason string) error {
	if chainID == "" {
		return fmt.Errorf("chain id is required")
	}
	if reason == "" {
		// A chain abandoned without a recorded reason is indistinguishable, later,
		// from one that genuinely ended — which is the ambiguity this exists to
		// remove.
		return fmt.Errorf("abandoning a chain requires a reason")
	}

	now := time.Now()
	result, err := s.db.ExecContext(ctx, `
		UPDATE execution_chains
		SET status = ?, reason = ?, completed_at = COALESCE(completed_at, ?), updated_at = ?
		WHERE id = ? AND status = ?
	`, string(ChainStatusAbandoned), reason, now, now, chainID, string(ChainStatusActive))
	if err != nil {
		return fmt.Errorf("failed to abandon chain %s: %w", chainID, err)
	}
	// Conditional on still being active: if something legitimately finished the
	// chain between the scan and this write, that verdict wins over ours.
	if rows, _ := result.RowsAffected(); rows == 0 {
		return nil
	}
	return nil
}

// ReconcileStrandedChains abandons every chain that can no longer progress and
// returns how many it closed.
func (s *Store) ReconcileStrandedChains(ctx context.Context, minAge time.Duration, reason string) (int, error) {
	stranded, err := s.FindStrandedChains(ctx, minAge)
	if err != nil {
		return 0, err
	}
	for _, c := range stranded {
		if err := s.AbandonChain(ctx, c.ChainID, reason); err != nil {
			// Report what was closed before the failure rather than claiming none
			// of it happened: the writes already landed.
			return 0, fmt.Errorf("reconciliation stopped at chain %s: %w", c.ChainID, err)
		}
	}
	return len(stranded), nil
}
