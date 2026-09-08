package coordinator

import (
	"context"
	"fmt"
)

// Ledger persistence (M-COMPLETION-PATH-PARITY C1).
//
// Dedicated accessors rather than extra columns threaded through the wide task
// INSERT/SELECT: the ledger is read once at the start of finalisation and written
// after each effect, so it has a different access pattern from the rest of the
// record — and editing a twenty-column scan to carry it would be a much larger
// blast radius for no benefit.

// GetTaskFinalization returns a task's finalisation ledger.
//
// A task with no ledger yet returns an empty one, not an error: every task
// predating this feature has none, and so does every task on its first delivery.
func (s *SQLiteStore) GetTaskFinalization(ctx context.Context, taskID string) (FinalizationLedger, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}
	var raw *string
	err := s.db.QueryRowContext(ctx, `SELECT finalization FROM tasks WHERE id = ?`, taskID).Scan(&raw)
	if err != nil {
		return nil, fmt.Errorf("failed to read finalization ledger for %s: %w", taskID, err)
	}
	if raw == nil {
		return FinalizationLedger{}, nil
	}
	return UnmarshalLedger(*raw)
}

// SetTaskFinalization writes a task's finalisation ledger.
func (s *SQLiteStore) SetTaskFinalization(ctx context.Context, taskID string, ledger FinalizationLedger) error {
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}
	encoded, err := MarshalLedger(ledger)
	if err != nil {
		return fmt.Errorf("failed to encode finalization ledger for %s: %w", taskID, err)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE tasks SET finalization = ? WHERE id = ?`, encoded, taskID)
	if err != nil {
		return fmt.Errorf("failed to write finalization ledger for %s: %w", taskID, err)
	}
	// A ledger written to a task that does not exist would look like progress
	// while recording nothing — fail loudly instead.
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("failed to write finalization ledger: task not found: %s", taskID)
	}
	return nil
}
