package firestore

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
)

// First-write-wins primitives for task finalisation (M-COMPLETION-PATH-PARITY M0b).
//
// Finalisation is replayed — Pub/Sub push is at-least-once — so the writes it
// performs must be safe to apply twice. The two backends fail in OPPOSITE
// directions, which is why both needed a primitive rather than one inheriting the
// other's behaviour:
//
//   - SQLite uses a bare INSERT, so a replay raises a UNIQUE violation and turns
//     a routine redelivery into a crash loop.
//   - Firestore uses Doc(id).Set, which does not error at all — it OVERWRITES.
//     That is worse: a replayed completion would reset an approval a human has
//     already approved back to "pending", or mark a message the recipient has
//     read as unread again.
//
// Create() is the fix here: it fails with AlreadyExists rather than overwriting,
// and AlreadyExists is the expected outcome of a replay, so it is reported as
// "not created" rather than as an error.

// isAlreadyExists reports whether err is Firestore's AlreadyExists, which is the
// normal outcome when a replay re-attempts a write that already landed.
func isAlreadyExists(err error) bool {
	return status.Code(err) == codes.AlreadyExists
}

// CreateApprovalIfAbsent creates an approval unless one with the same id exists,
// and reports whether it created it.
//
// The id is already deterministic (apr-<task hash>), so a redelivered completion
// targets the same document. Using Set here would reset a resolved approval to
// pending — silently undoing a human decision — which is why this exists.
func (s *CoordinatorStore) CreateApprovalIfAbsent(ctx context.Context, req *coordinator.ApprovalRequestRecord) (bool, error) {
	if req == nil || req.ID == "" {
		return false, fmt.Errorf("CreateApprovalIfAbsent requires an approval with an explicit id")
	}
	_, err := s.client.Doc(collApprovals, req.ID).Create(ctx, approvalToMap(req))
	if err != nil {
		if isAlreadyExists(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to create approval %s: %w", req.ID, err)
	}
	return true, nil
}

// ReopenApprovalForNewWork puts a RESOLVED approval back to pending because a
// later execution of the same task produced a different change.
//
// Guarded on the CURRENT status inside a transaction, not read-then-write: two
// completions racing here would otherwise both see "rejected" and both reopen,
// and the second would overwrite the first's context with the same-but-older
// diff. The guard is also what stops this touching a pending row, which has
// nothing to reopen.
func (s *CoordinatorStore) ReopenApprovalForNewWork(ctx context.Context, taskID, description, contextJSON string) (bool, error) {
	if taskID == "" {
		return false, fmt.Errorf("ReopenApprovalForNewWork requires a task id")
	}
	docID := coordinator.ApprovalIDForTask(taskID)
	ref := s.client.Doc(collApprovals, docID)
	reopened := false
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				reopened = false
				return nil // nothing to reopen; the caller creates instead
			}
			return err
		}
		cur, _ := snap.Data()["status"].(string)
		if cur != "approved" && cur != "rejected" {
			reopened = false
			return nil
		}
		reopened = true
		return tx.Update(ref, []firestore.Update{
			{Path: "status", Value: "pending"},
			{Path: "resolved_by", Value: nil},
			{Path: "resolved_at", Value: nil},
			{Path: "description", Value: description},
			{Path: "context_json", Value: contextJSON},
		})
	})
	if err != nil {
		return false, fmt.Errorf("reopening approval for %s: %w", taskID, err)
	}
	return reopened, nil
}

// PutMessageIfAbsent inserts a message unless one with the same id exists, and
// reports whether it created it.
//
// Finalisation dispatches handoffs and completion notices under a deterministic
// id so that a replay collides instead of duplicating. Overwriting would discard
// whatever has happened to the message since — most importantly its read status.
func (s *MessagingStore) PutMessageIfAbsent(ctx context.Context, msg *messaging.InboxMessage) (bool, error) {
	if msg == nil || msg.ID == "" {
		return false, fmt.Errorf("PutMessageIfAbsent requires an explicit message id: the whole point is that a replay collides")
	}
	normalizeInboxDefaults(msg)

	_, err := s.client.Doc(collInbox, msg.ID).Create(ctx, inboxToMap(msg))
	if err != nil {
		if isAlreadyExists(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to create message %s: %w", msg.ID, err)
	}
	return true, nil
}
