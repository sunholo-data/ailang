package firestore

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
// by field as the grade layers add to it. `complete` and `grade_state` are
// copied out only so the grade sweep can query them.
const collShadowComparisons = "shadow_comparisons"

func decodeShadow(data map[string]interface{}, parentID string) (coordinator.ShadowComparison, error) {
	var c coordinator.ShadowComparison
	if err := json.Unmarshal([]byte(mapval.String(data, "data")), &c); err != nil {
		return c, fmt.Errorf("shadow comparison %s: %w", parentID, err)
	}
	return c, nil
}

func shadowDoc(c coordinator.ShadowComparison) (map[string]interface{}, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"parent_id":   c.ParentID,
		"data":        string(raw),
		"complete":    c.Complete(),
		"grade_state": c.GradeState,
		"updated_at":  c.UpdatedAt,
	}, nil
}

// shadowTx reads, transforms and writes one comparison in a transaction: the
// primary's and the shadow's completions, and the grade sweep, can all touch it
// at once on different instances.
func (s *CoordinatorStore) shadowTx(ctx context.Context, parentID string, fn func(coordinator.ShadowComparison, bool) (coordinator.ShadowComparison, bool, error)) (*coordinator.ShadowComparison, bool, error) {
	if parentID == "" {
		return nil, false, fmt.Errorf("shadow comparison: parent id is required")
	}
	ref := s.client.Doc(collShadowComparisons, parentID)
	var out coordinator.ShadowComparison
	var wrote bool
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		wrote = false
		cur := coordinator.ShadowComparison{ParentID: parentID}
		found := true
		doc, err := tx.Get(ref)
		switch {
		case status.Code(err) == codes.NotFound:
			found = false
		case err != nil:
			return err
		default:
			if cur, err = decodeShadow(doc.Data(), parentID); err != nil {
				return err
			}
		}
		next, write, err := fn(cur, found)
		out = cur
		if err != nil || !write {
			return err
		}
		m, err := shadowDoc(next)
		if err != nil {
			return err
		}
		out, wrote = next, true
		return tx.Set(ref, m)
	})
	if err != nil {
		return nil, false, err
	}
	return &out, wrote, nil
}

// RecordShadowRun merges one side into the parent's comparison.
func (s *CoordinatorStore) RecordShadowRun(ctx context.Context, parentID, side string, run coordinator.ShadowRunResult) (*coordinator.ShadowComparison, error) {
	c, _, err := s.shadowTx(ctx, parentID, func(cur coordinator.ShadowComparison, _ bool) (coordinator.ShadowComparison, bool, error) {
		next, err := coordinator.MergeShadowRun(cur, side, run, time.Now().UTC())
		return next, err == nil, err
	})
	return c, err
}

// UpdateShadowComparison applies fn to an existing comparison atomically.
func (s *CoordinatorStore) UpdateShadowComparison(ctx context.Context, parentID string, fn func(*coordinator.ShadowComparison) bool) (bool, error) {
	_, wrote, err := s.shadowTx(ctx, parentID, func(cur coordinator.ShadowComparison, found bool) (coordinator.ShadowComparison, bool, error) {
		if !found {
			return cur, false, fmt.Errorf("%w: %s", coordinator.ErrShadowComparisonNotFound, parentID)
		}
		if !fn(&cur) {
			return cur, false, nil
		}
		cur.UpdatedAt = time.Now().UTC()
		return cur, true, nil
	})
	return wrote, err
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
	c, err := decodeShadow(doc.Data(), parentID)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListShadowComparisonsByGradeState lists complete comparisons in one grade
// state. Equality filters only, so no composite index is needed; the order is
// applied here.
func (s *CoordinatorStore) ListShadowComparisonsByGradeState(ctx context.Context, state string, limit int) ([]*coordinator.ShadowComparison, error) {
	if limit <= 0 {
		limit = 20
	}
	docs, err := s.client.Collection(collShadowComparisons).
		Where("complete", "==", true).Where("grade_state", "==", state).
		Limit(limit).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]*coordinator.ShadowComparison, 0, len(docs))
	for _, d := range docs {
		c, err := decodeShadow(d.Data(), d.Ref.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.Before(out[j].UpdatedAt) })
	return out, nil
}

var _ coordinator.ShadowComparisonStore = (*CoordinatorStore)(nil)
