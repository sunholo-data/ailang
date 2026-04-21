package firestore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// --- Approval Workflow ---

func (s *MessagingStore) CreateApproval(threadID, instanceID string, effectDelta *messaging.EffectDelta, proposal, impact string, estimatedCost float64) (*messaging.Approval, error) {
	ctx := context.Background()
	now := time.Now()
	id := fmt.Sprintf("apr_%d_%s", now.UnixMilli(), generateShortID())

	deltaJSON := ""
	if effectDelta != nil {
		if b, err := json.Marshal(effectDelta); err == nil {
			deltaJSON = string(b)
		}
	}

	approval := &messaging.Approval{
		ID:              id,
		ThreadID:        threadID,
		InstanceID:      instanceID,
		CreatedAt:       now,
		EffectDeltaJSON: deltaJSON,
		Proposal:        proposal,
		Impact:          impact,
		EstimatedCost:   estimatedCost,
		Status:          "pending",
	}

	_, err := s.client.Doc(collMsgApproval, id).Set(ctx, approvalMsgToMap(approval))
	if err != nil {
		return nil, err
	}
	return approval, nil
}

func (s *MessagingStore) GetApproval(approvalID string) (*messaging.Approval, error) {
	doc, err := s.client.Doc(collMsgApproval, approvalID).Get(context.Background())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("approval not found: %s", approvalID)
		}
		return nil, err
	}
	return mapToApprovalMsg(doc.Data()), nil
}

func (s *MessagingStore) GetApprovalsByStatus(approvalStatus string, limit int) ([]messaging.Approval, error) {
	ctx := context.Background()
	q := s.client.Collection(collMsgApproval).
		Where("status", "==", approvalStatus).
		OrderBy("created_at", firestore.Desc)

	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var approvals []messaging.Approval
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, *mapToApprovalMsg(doc.Data()))
	}
	return approvals, nil
}

func (s *MessagingStore) ApproveApproval(approvalID, reviewedBy string, reviewNotes string, tokenDuration time.Duration) error {
	now := time.Now()
	token := fmt.Sprintf("cap_%d_%s", now.UnixMilli(), generateShortID())
	_, err := s.client.Doc(collMsgApproval, approvalID).Update(context.Background(), []firestore.Update{
		{Path: "status", Value: "approved"},
		{Path: "reviewed_by", Value: reviewedBy},
		{Path: "reviewed_at", Value: timeToFirestore(now)},
		{Path: "review_notes", Value: reviewNotes},
		{Path: "capability_token", Value: token},
		{Path: "token_expires_at", Value: timeToFirestore(now.Add(tokenDuration))},
	})
	return err
}

func (s *MessagingStore) RejectApproval(approvalID, reviewedBy string, reviewNotes string) error {
	now := time.Now()
	_, err := s.client.Doc(collMsgApproval, approvalID).Update(context.Background(), []firestore.Update{
		{Path: "status", Value: "rejected"},
		{Path: "reviewed_by", Value: reviewedBy},
		{Path: "reviewed_at", Value: timeToFirestore(now)},
		{Path: "review_notes", Value: reviewNotes},
	})
	return err
}

// --- History Tracking ---

func (s *MessagingStore) RecordApprovalHistory(approvalID, threadID, agentID, action, actor, proposal, impact string, estimatedCost *float64, capabilityToken string) error {
	ctx := context.Background()
	now := time.Now()
	id := fmt.Sprintf("ahist_%d_%s", now.UnixMilli(), generateShortID())

	entry := &messaging.ApprovalHistoryEntry{
		ID:              id,
		ApprovalID:      approvalID,
		ThreadID:        threadID,
		AgentID:         agentID,
		Action:          action,
		Actor:           actor,
		Proposal:        proposal,
		Impact:          impact,
		EstimatedCost:   estimatedCost,
		CapabilityToken: capabilityToken,
		CreatedAt:       now.UnixMilli(),
	}

	_, err := s.client.Doc(collHistory, id).Set(ctx, approvalHistoryToMap(entry))
	return err
}

func (s *MessagingStore) GetApprovalHistory(threadID string, limit int) ([]messaging.ApprovalHistoryEntry, error) {
	ctx := context.Background()
	q := s.client.Collection(collHistory).
		Where("thread_id", "==", threadID).
		OrderBy("created_at", firestore.Desc)

	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var entries []messaging.ApprovalHistoryEntry
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, mapToApprovalHistory(doc.Data()))
	}
	return entries, nil
}

func (s *MessagingStore) RecordInstanceStart(agentID, instanceID string) error {
	ctx := context.Background()
	id := fmt.Sprintf("inst_%d_%s", time.Now().UnixMilli(), generateShortID())

	entry := &messaging.InstanceHistoryEntry{
		ID:         id,
		AgentID:    agentID,
		InstanceID: instanceID,
		StartedAt:  time.Now().UnixMilli(),
	}

	_, err := s.client.Doc(collInstHistory, id).Set(ctx, instanceHistoryToMap(entry))
	return err
}

func (s *MessagingStore) RecordInstanceEnd(instanceID string, exitCode int, totalTokens, totalCostCents, threadCount int) error {
	ctx := context.Background()

	// Find the instance record by instanceID
	iter := s.client.Collection(collInstHistory).
		Where("instance_id", "==", instanceID).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("instance not found: %s", instanceID)
	}
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	_, err = doc.Ref.Update(ctx, []firestore.Update{
		{Path: "ended_at", Value: now},
		{Path: "exit_code", Value: exitCode},
		{Path: "total_tokens", Value: totalTokens},
		{Path: "total_cost_cents", Value: totalCostCents},
		{Path: "thread_count", Value: threadCount},
	})
	return err
}

func (s *MessagingStore) GetInstanceHistory(agentID string, limit int) ([]messaging.InstanceHistoryEntry, error) {
	ctx := context.Background()
	q := s.client.Collection(collInstHistory).
		Where("agent_id", "==", agentID).
		OrderBy("started_at", firestore.Desc)

	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var entries []messaging.InstanceHistoryEntry
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, mapToInstanceHistory(doc.Data()))
	}
	return entries, nil
}

func (s *MessagingStore) CleanupOldHistory(retentionDays int) (int64, int64, error) {
	ctx := context.Background()
	cutoff := time.Now().AddDate(0, 0, -retentionDays).UnixMilli()

	// Cleanup approval history
	var approvalCount int64
	iter := s.client.Collection(collHistory).
		Where("created_at", "<=", cutoff).
		Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iter.Stop()
			return approvalCount, 0, err
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			iter.Stop()
			return approvalCount, 0, err
		}
		approvalCount++
	}
	iter.Stop()

	// Cleanup instance history
	var instanceCount int64
	iter2 := s.client.Collection(collInstHistory).
		Where("started_at", "<=", cutoff).
		Documents(ctx)
	defer iter2.Stop()
	for {
		doc, err := iter2.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return approvalCount, instanceCount, err
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			return approvalCount, instanceCount, err
		}
		instanceCount++
	}

	return approvalCount, instanceCount, nil
}
