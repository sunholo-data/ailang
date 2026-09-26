package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// Reopening a rejected or cancelled task on the CLOUD plane.
//
// `ailang coordinator reopen` existed for a year and could only ever act on this
// machine's SQLite: ReopenTask was a method on *SQLiteStore and was not on the
// Store interface at all, so there was no cloud implementation to call even if
// the command had honoured --remote. Measured 2026-09-15: eleven cancelled
// ailang-core-triage tasks in prod, and the only way to put them back in the
// queue was to re-dispatch the work from scratch under new ids — losing the
// artifacts, the chain, and the very approval history the re-run fix needed to
// be tested against.
//
// The semantics here MIRROR the SQLite ones deliberately (refuse anything not
// rejected/cancelled; clear completed_at; reset the approval rather than create
// a second one). Two implementations of one concept that disagree is the fault
// class this whole sprint has been chasing; ReopenTaskContract in the
// coordinator package is the shared test.

// ReopenTask moves a rejected or cancelled task back to pending_approval.
//
// Transactional because the guard is a read: two operators reopening at once
// would otherwise both see "cancelled", and the second would reset an approval
// the first has already put back in front of a human.
func (s *CoordinatorStore) ReopenTask(ctx context.Context, taskID string) error {
	if taskID == "" {
		return fmt.Errorf("ReopenTask requires a task id")
	}
	taskRef := s.client.Doc(collTasks, taskID)

	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// All reads first: Firestore refuses a read that follows a write in the
		// same transaction.
		snap, err := tx.Get(taskRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return fmt.Errorf("task not found: %s", taskID)
			}
			return err
		}
		cur, _ := snap.Data()["status"].(string)
		if cur != string(coordinator.TaskStatusRejected) && cur != string(coordinator.TaskStatusCancelled) {
			return fmt.Errorf("cannot reopen task with status %q (only rejected or cancelled tasks can be reopened)", cur)
		}

		// Address the approval by task_id, not by the derived id: approvals
		// written before ApprovalIDForTask existed carry other ids, and creating
		// a second row for the same task would show the operator two decisions
		// for one piece of work.
		approvals := s.client.Collection(collApprovals).Where("task_id", "==", taskID)
		iter := tx.Documents(approvals)
		defer iter.Stop()
		var refs []*firestore.DocumentRef
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			refs = append(refs, doc.Ref)
		}

		if err := tx.Update(taskRef, []firestore.Update{
			{Path: "status", Value: string(coordinator.TaskStatusPendingApproval)},
			{Path: "completed_at", Value: nil},
		}); err != nil {
			return err
		}

		for _, ref := range refs {
			if err := tx.Update(ref, []firestore.Update{
				{Path: "status", Value: "pending"},
				{Path: "resolved_by", Value: nil},
				{Path: "resolved_at", Value: nil},
			}); err != nil {
				return err
			}
		}
		if len(refs) == 0 {
			rec := &coordinator.ApprovalRequestRecord{
				ID:          coordinator.ApprovalIDForTask(taskID),
				TaskID:      taskID,
				Type:        "merge",
				Description: "Task reopened for approval",
				Status:      "pending",
				CreatedAt:   time.Now(),
			}
			if err := tx.Set(s.client.Doc(collApprovals, rec.ID), approvalToMap(rec)); err != nil {
				return fmt.Errorf("failed to create approval request: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.invalidateStatsCache()
	return nil
}
