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

// --- Budget Tracking ---

// GetCostByProvider returns total cost per provider from the in-memory cache.
// Zero Firestore reads on the hot path — counters are synced in the background.
func (s *CoordinatorStore) GetCostByProvider() (map[string]float64, error) {
	s.costMu.RLock()
	defer s.costMu.RUnlock()

	// If counters haven't been loaded yet (StartCostSync not called), fall back to full scan.
	if !s.costLoaded {
		s.costMu.RUnlock()
		costs, err := s.fullScanCostByProvider(context.Background())
		s.costMu.RLock()
		return costs, err
	}

	result := make(map[string]float64, len(s.costCounters))
	for k, v := range s.costCounters {
		result[k] = v
	}
	return result, nil
}

// --- Approval Requests ---

func (s *CoordinatorStore) CreateApprovalRequest(ctx context.Context, req *coordinator.ApprovalRequestRecord) error {
	data := approvalToMap(req)
	_, err := s.client.Doc(collApprovals, req.ID).Set(ctx, data)
	return err
}

func (s *CoordinatorStore) GetApprovalRequest(ctx context.Context, id string) (*coordinator.ApprovalRequestRecord, error) {
	doc, err := s.client.Doc(collApprovals, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("approval not found: %s", id)
		}
		return nil, err
	}
	return mapToApproval(doc.Data()), nil
}

func (s *CoordinatorStore) GetApprovalRequestByTask(ctx context.Context, taskID string) (*coordinator.ApprovalRequestRecord, error) {
	iter := s.client.Collection(collApprovals).
		Where("task_id", "==", taskID).
		Where("status", "==", "pending").
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no pending approval for task: %s", taskID)
	}
	if err != nil {
		return nil, err
	}
	return mapToApproval(doc.Data()), nil
}

func (s *CoordinatorStore) GetApprovalRequestByTaskAnyStatus(ctx context.Context, taskID string) (*coordinator.ApprovalRequestRecord, error) {
	iter := s.client.Collection(collApprovals).
		Where("task_id", "==", taskID).
		OrderBy("created_at", firestore.Desc).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no approval for task: %s", taskID)
	}
	if err != nil {
		return nil, err
	}
	return mapToApproval(doc.Data()), nil
}

func (s *CoordinatorStore) ListPendingApprovals(ctx context.Context) ([]*coordinator.ApprovalRequestRecord, error) {
	iter := s.client.Collection(collApprovals).
		Where("status", "==", "pending").
		OrderBy("created_at", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var approvals []*coordinator.ApprovalRequestRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, mapToApproval(doc.Data()))
	}
	return approvals, nil
}

func (s *CoordinatorStore) ListResolvedApprovals(ctx context.Context, limit int) ([]*coordinator.ApprovalRequestRecord, error) {
	q := s.client.Collection(collApprovals).
		Where("status", "in", []string{"approved", "rejected"}).
		OrderBy("resolved_at", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}
	iter := q.Documents(ctx)
	defer iter.Stop()

	var approvals []*coordinator.ApprovalRequestRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, mapToApproval(doc.Data()))
	}
	return approvals, nil
}

func (s *CoordinatorStore) ResolveApprovalRequest(ctx context.Context, id, resolveStatus, resolvedBy string) error {
	now := time.Now()
	_, err := s.client.Doc(collApprovals, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: resolveStatus},
		{Path: "resolved_by", Value: resolvedBy},
		{Path: "resolved_at", Value: now},
	})
	return err
}

func (s *CoordinatorStore) ResolveApprovalRequestByTask(ctx context.Context, taskID, resolveStatus, resolvedBy string) error {
	// Find the pending approval for this task, then resolve it
	iter := s.client.Collection(collApprovals).
		Where("task_id", "==", taskID).
		Where("status", "==", "pending").
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("no pending approval for task: %s", taskID)
	}
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = doc.Ref.Update(ctx, []firestore.Update{
		{Path: "status", Value: resolveStatus},
		{Path: "resolved_by", Value: resolvedBy},
		{Path: "resolved_at", Value: now},
	})
	return err
}

func (s *CoordinatorStore) MarkApprovalHandoffsTriggered(ctx context.Context, taskID string) error {
	// Find approval for task and mark handoffs as triggered
	iter := s.client.Collection(collApprovals).
		Where("task_id", "==", taskID).
		Where("status", "==", "approved").
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil // No approval to update
	}
	if err != nil {
		return err
	}

	_, err = doc.Ref.Update(ctx, []firestore.Update{
		{Path: "handoffs_triggered", Value: true},
	})
	return err
}

func (s *CoordinatorStore) ListApprovedMergeHandoffsWithoutTrigger(ctx context.Context) ([]*coordinator.ApprovalRequestRecord, error) {
	// Find approved merge_handoff approvals where handoffs haven't been triggered
	iter := s.client.Collection(collApprovals).
		Where("status", "==", "approved").
		Where("type", "==", "merge_handoff").
		Where("handoffs_triggered", "==", false).
		Documents(ctx)
	defer iter.Stop()

	var approvals []*coordinator.ApprovalRequestRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, mapToApproval(doc.Data()))
	}
	return approvals, nil
}

// --- Cleanup ---

func (s *CoordinatorStore) DeleteOldTasks(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	iter := s.client.Collection(collTasks).
		Where("created_at", "<", cutoff).
		Documents(ctx)
	defer iter.Stop()

	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return count, err
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *CoordinatorStore) RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration) (int, error) {
	cutoff := time.Now().Add(-staleThreshold)

	// Find running/queued tasks older than threshold
	count := 0
	for _, taskStatus := range []string{"running", "queued"} {
		iter := s.client.Collection(collTasks).
			Where("status", "==", taskStatus).
			Where("created_at", "<", cutoff).
			Documents(ctx)

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return count, err
			}
			_, err = doc.Ref.Update(ctx, []firestore.Update{
				{Path: "status", Value: string(coordinator.TaskStatusCancelled)},
				{Path: "error", Value: "recovered: stale task from previous daemon run"},
				{Path: "completed_at", Value: time.Now()},
			})
			if err != nil {
				iter.Stop()
				return count, err
			}
			count++
		}
		iter.Stop()
	}
	return count, nil
}

func (s *CoordinatorStore) RetryAllFailedTasks(ctx context.Context) (int, error) {
	iter := s.client.Collection(collTasks).
		Where("status", "==", string(coordinator.TaskStatusFailed)).
		Documents(ctx)
	defer iter.Stop()

	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return count, err
		}
		_, err = doc.Ref.Update(ctx, []firestore.Update{
			{Path: "status", Value: string(coordinator.TaskStatusPending)},
			{Path: "error", Value: ""},
			{Path: "started_at", Value: nil},
			{Path: "completed_at", Value: nil},
		})
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// --- Event Storage ---

func (s *CoordinatorStore) StoreTaskEvent(ctx context.Context, event *coordinator.TaskEventRecord) error {
	data := eventToMap(event)
	// Use subcollection: /tasks/{task_id}/events/{auto_id}
	_, _, err := s.client.Collection(collTasks).Doc(event.TaskID).Collection("events").Add(ctx, data)
	return err
}

func (s *CoordinatorStore) GetTaskEvents(ctx context.Context, taskID string, limit int) ([]*coordinator.TaskEventRecord, error) {
	if limit <= 0 {
		limit = 1000
	}
	iter := s.client.Collection(collTasks).Doc(taskID).Collection("events").
		OrderBy("created_at", firestore.Asc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	var events []*coordinator.TaskEventRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		events = append(events, mapToEvent(doc.Data(), taskID))
	}
	return events, nil
}
