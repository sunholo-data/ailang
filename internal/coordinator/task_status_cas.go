package coordinator

import (
	"context"
	"fmt"
	"strings"
)

// Compare-and-set on task status (M-COMPLETION-PATH-PARITY M1 / C1).
//
// The finalisation ledger stops an effect being applied twice by the SAME
// finalisation. It cannot stop a stale replay overwriting a record that some
// OTHER step has legitimately advanced in the meantime — and an absolute write is
// repeatable, not idempotent, once that has happened.
//
// The concrete failure: a finalizer writes pending_approval, crashes before
// marking its ledger row, a human approves the task, and then the redelivery (or
// the reconciliation sweep) re-applies pending_approval — silently reverting the
// approval. On an auto-approved edge that also re-releases a handoff, so the next
// agent starts twice on work that was already accepted.
//
// So a status write states what it expects to find. If the record has moved on,
// the write does not land and the caller records the effect as superseded, which
// is a normal outcome rather than an error.

// FinalizableFrom lists the statuses a finalisation may advance FROM.
//
// A task that has already reached a terminal state, or been resolved by a human,
// is not a valid source: reaching those means something with more authority than
// a replayed message has already spoken.
func FinalizableFrom() []TaskStatus {
	return []TaskStatus{
		TaskStatusPending,
		TaskStatusQueued,
		TaskStatusRunning,
	}
}

// CompareAndSetTaskStatus sets a task's status only if it currently holds one of
// the expected values, and reports whether the write landed.
//
// A false return is not an error: it means the record advanced past what this
// finalisation was told about, and the effect is superseded.
func (s *SQLiteStore) CompareAndSetTaskStatus(ctx context.Context, id string, expected []TaskStatus, next TaskStatus) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("task id is required")
	}
	if len(expected) == 0 {
		return false, fmt.Errorf("CompareAndSetTaskStatus requires at least one expected status: an unconditional write is what this exists to prevent")
	}

	placeholders := make([]string, len(expected))
	args := make([]interface{}, 0, len(expected)+2)
	args = append(args, string(next), id)
	for i, st := range expected {
		placeholders[i] = "?"
		args = append(args, string(st))
	}

	query := fmt.Sprintf(
		`UPDATE tasks SET status = ? WHERE id = ? AND status IN (%s)`,
		strings.Join(placeholders, ", "),
	)
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("failed to compare-and-set status for %s: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected for %s: %w", id, err)
	}
	return rows > 0, nil
}
