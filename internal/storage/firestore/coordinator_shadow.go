package firestore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mapval"
)

// collShadowComparisons holds one document per primary task
// (M-AILANG-LANE-EXECUTOR-SHADOW M4). The comparison is stored as one JSON
// field, the same document SQLite keeps, so the two stores cannot drift field
// by field as the grade layers add to it.
const collShadowComparisons = "shadow_comparisons"

// RecordShadowRun merges one side in a transaction: the primary's and the
// shadow's completions can land at the same moment on two instances.
func (s *CoordinatorStore) RecordShadowRun(ctx context.Context, parentID, side string, run coordinator.ShadowRunResult) (*coordinator.ShadowComparison, error) {
	if parentID == "" {
		return nil, fmt.Errorf("shadow comparison: parent id is required")
	}
	ref := s.client.Doc(collShadowComparisons, parentID)
	var out coordinator.ShadowComparison
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		cur := coordinator.ShadowComparison{ParentID: parentID}
		doc, err := tx.Get(ref)
		switch {
		case status.Code(err) == codes.NotFound:
		case err != nil:
			return err
		default:
			if err := json.Unmarshal([]byte(mapval.String(doc.Data(), "data")), &cur); err != nil {
				return fmt.Errorf("shadow comparison %s: %w", parentID, err)
			}
		}
		next, err := coordinator.MergeShadowRun(cur, side, run, time.Now().UTC())
		if err != nil {
			return err
		}
		raw, err := json.Marshal(next)
		if err != nil {
			return err
		}
		out = next
		return tx.Set(ref, map[string]interface{}{
			"parent_id":  parentID,
			"data":       string(raw),
			"complete":   next.Complete(),
			"updated_at": next.UpdatedAt,
		})
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetShadowComparison returns coordinator.ErrShadowComparisonNotFound for an unknown parent.
func (s *CoordinatorStore) GetShadowComparison(ctx context.Context, parentID string) (*coordinator.ShadowComparison, error) {
	doc, err := s.client.Doc(collShadowComparisons, parentID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("%w: %s", coordinator.ErrShadowComparisonNotFound, parentID)
	}
	if err != nil {
		return nil, err
	}
	var c coordinator.ShadowComparison
	if err := json.Unmarshal([]byte(mapval.String(doc.Data(), "data")), &c); err != nil {
		return nil, fmt.Errorf("shadow comparison %s: %w", parentID, err)
	}
	return &c, nil
}

var _ coordinator.ShadowComparisonStore = (*CoordinatorStore)(nil)
