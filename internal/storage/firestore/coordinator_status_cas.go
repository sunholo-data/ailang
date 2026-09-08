package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// CompareAndSetTaskStatus sets a task's status only if it currently holds one of
// the expected values (M-COMPLETION-PATH-PARITY M1 / C1).
//
// Firestore has no conditional UPDATE, so the read and the write must be one
// transaction. Without it the check and the write race: two finalizers could both
// read "running", both decide the write is valid, and the slower one land last —
// which is the same regression by a different route.
func (s *CoordinatorStore) CompareAndSetTaskStatus(ctx context.Context, id string, expected []coordinator.TaskStatus, next coordinator.TaskStatus) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("task id is required")
	}
	if len(expected) == 0 {
		return false, fmt.Errorf("CompareAndSetTaskStatus requires at least one expected status: an unconditional write is what this exists to prevent")
	}

	allowed := make(map[string]bool, len(expected))
	for _, st := range expected {
		allowed[string(st)] = true
	}

	ref := s.client.Doc(collTasks, id)
	var applied bool

	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		applied = false
		doc, err := tx.Get(ref)
		if err != nil {
			return fmt.Errorf("reading task %s: %w", id, err)
		}
		current := getString(doc.Data(), "status")
		if !allowed[current] {
			// Not an error: the record advanced past what this finalisation was
			// told about, so the effect is superseded.
			return nil
		}
		applied = true
		return tx.Update(ref, []firestore.Update{{Path: "status", Value: string(next)}})
	})
	if err != nil {
		return false, fmt.Errorf("failed to compare-and-set status for %s: %w", id, err)
	}
	return applied, nil
}
