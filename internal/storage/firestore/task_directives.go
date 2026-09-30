package firestore

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// A task's directive — the fully templated prompt the executor runs — travels
// to the Cloud Run Job through this collection, not through an env var.
//
// It used to be AILANG_DIRECTIVE, and Cloud Run caps one env value at 32,768
// bytes. Daneel's requests carry fetched source pages and the target repo's
// open PRs, and on 2026-09-30 a 43 KB request (task-68771ff3) was refused
// with InvalidArgument on every dispatch — 18 retries over 2.5 hours while the
// task sat "pending" and nobody was told. A Firestore document holds ~1 MiB,
// the executor SA already reads Firestore (datastore.user), and the job is
// already handed AILANG_TASK_ID, so the key needs no new plumbing.
const (
	collTaskDirectives = "task_directives"

	// directiveTTL sets expires_at for Firestore TTL housekeeping. A directive
	// is read once, at job start; a week covers retries and re-dispatches.
	directiveTTL = 7 * 24 * time.Hour
)

// TaskDirectiveStore writes and reads task directives.
type TaskDirectiveStore struct {
	c *Client
}

// NewTaskDirectiveStore returns a store backed by c.
func NewTaskDirectiveStore(c *Client) *TaskDirectiveStore {
	return &TaskDirectiveStore{c: c}
}

// PutDirective stores the directive for taskID, replacing any earlier one (a
// re-dispatch after a rejection carries new feedback).
func (s *TaskDirectiveStore) PutDirective(ctx context.Context, taskID, directive string) error {
	if taskID == "" {
		return fmt.Errorf("put directive: empty task id")
	}
	if len(directive) > coordinator.MaxDirectiveBytes {
		return fmt.Errorf("%w: directive for %s is %d bytes, over the %d-byte limit",
			coordinator.ErrDispatchPermanent, taskID, len(directive), coordinator.MaxDirectiveBytes)
	}
	now := time.Now().UTC()
	_, err := s.c.Doc(collTaskDirectives, taskID).Set(ctx, map[string]any{
		"task_id":    taskID,
		"directive":  directive,
		"bytes":      len(directive),
		"written_at": now,
		"expires_at": now.Add(directiveTTL),
	})
	if err != nil {
		return fmt.Errorf("put directive for %s: %w", taskID, err)
	}
	return nil
}

// GetDirective returns the directive stored for taskID. A missing document is
// an error, never "": an executor that ran an empty prompt would still open a
// PR, and nothing downstream could tell it had never been told anything.
func (s *TaskDirectiveStore) GetDirective(ctx context.Context, taskID string) (string, error) {
	snap, err := s.c.Doc(collTaskDirectives, taskID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return "", fmt.Errorf("no directive stored for %s in %s", taskID, collTaskDirectives)
	}
	if err != nil {
		return "", fmt.Errorf("read directive for %s: %w", taskID, err)
	}
	d, _ := snap.Data()["directive"].(string)
	if d == "" {
		return "", fmt.Errorf("directive for %s is empty in %s", taskID, collTaskDirectives)
	}
	return d, nil
}
