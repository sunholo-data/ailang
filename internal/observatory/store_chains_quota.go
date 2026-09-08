// Package observatory provides quota-lane spend recording for execution chains.
package observatory

import (
	"context"
	"fmt"
)

// ===== Quota-lane spend (M-QUOTA-RATIONING-ROUTING M2) =====

// UpdateStageQuotaTokens records a subscription stage's token consumption.
//
// It writes chain_stages.quota_tokens and NEVER tokens_in/tokens_out. The cost
// estimator has no schema flag distinguishing a metered stage from a subscription
// one; it uses `tokens > 0` as that marker. Writing here instead of there is what
// keeps the two accounting systems from contaminating each other — the ration can
// see subscription spend, and the metered KPI never invoices a run nobody paid for.
//
// A ZERO count is rejected rather than written: every quota stage already reads
// zero, so "wrote 0" and "never reported" would be indistinguishable, and the
// ration would silently measure low exactly as it does today.
func (s *Store) UpdateStageQuotaTokens(ctx context.Context, stageID string, tokens int64) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}
	if tokens <= 0 {
		return fmt.Errorf("quota_tokens must be positive (got %d); a zero write is indistinguishable from never reporting", tokens)
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE chain_stages SET quota_tokens = ? WHERE id = ?
	`, tokens, stageID)
	if err != nil {
		return fmt.Errorf("failed to update quota tokens: %w", err)
	}
	// A silent no-op here would leave the ledger short by a whole stage with no
	// error anywhere, which is the failure mode this milestone exists to end.
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return fmt.Errorf("no chain stage %q to record quota tokens against", stageID)
	}
	return nil
}
