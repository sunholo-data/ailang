package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// GetTaskFinalization returns a task's finalisation ledger
// (M-COMPLETION-PATH-PARITY C1).
func (s *CoordinatorStore) GetTaskFinalization(ctx context.Context, taskID string) (coordinator.FinalizationLedger, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}
	doc, err := s.client.Doc(collTasks, taskID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read finalization ledger for %s: %w", taskID, err)
	}
	return ledgerFromMap(doc.Data()["finalization"]), nil
}

// SetTaskFinalization writes a task's finalisation ledger.
func (s *CoordinatorStore) SetTaskFinalization(ctx context.Context, taskID string, ledger coordinator.FinalizationLedger) error {
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}
	_, err := s.client.Doc(collTasks, taskID).Update(ctx, []firestore.Update{
		{Path: "finalization", Value: ledgerToMap(ledger)},
	})
	if err != nil {
		return fmt.Errorf("failed to write finalization ledger for %s: %w", taskID, err)
	}
	return nil
}
