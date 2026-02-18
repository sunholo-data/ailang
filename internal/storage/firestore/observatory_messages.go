package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	obs "github.com/sunholo/ailang/internal/observatory"
)

// --- Observatory Message operations ---

func (s *ObservatoryStore) CreateMessage(ctx context.Context, m *obs.Message) error {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	_, err := s.client.Doc(collObsMessages, m.ID).Set(ctx, obsMessageToMap(m))
	return err
}

func (s *ObservatoryStore) GetMessage(ctx context.Context, id string) (*obs.Message, error) {
	doc, err := s.client.Doc(collObsMessages, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("message not found: %s", id)
		}
		return nil, err
	}
	return mapToObsMessage(doc.Data()), nil
}

func (s *ObservatoryStore) ListMessages(ctx context.Context, opts obs.MessageListOptions) ([]*obs.Message, error) {
	q := s.client.Collection(collObsMessages).Query
	if opts.Inbox != "" {
		q = q.Where("inbox", "==", opts.Inbox)
	}
	if opts.Status != "" {
		q = q.Where("status", "==", string(opts.Status))
	}
	if opts.TaskID != "" {
		q = q.Where("task_id", "==", opts.TaskID)
	}
	if opts.FromAgent != "" {
		q = q.Where("from_agent", "==", opts.FromAgent)
	}
	q = q.OrderBy("created_at", firestore.Desc)
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var result []*obs.Message
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, mapToObsMessage(doc.Data()))
	}
	return result, nil
}

func (s *ObservatoryStore) UpdateMessage(ctx context.Context, m *obs.Message) error {
	_, err := s.client.Doc(collObsMessages, m.ID).Set(ctx, obsMessageToMap(m))
	return err
}

func (s *ObservatoryStore) DeleteMessage(ctx context.Context, id string) error {
	_, err := s.client.Doc(collObsMessages, id).Delete(ctx)
	return err
}

func (s *ObservatoryStore) MarkMessageRead(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.client.Doc(collObsMessages, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: "read"},
		{Path: "read_at", Value: timeToFirestore(now)},
	})
	return err
}

func (s *ObservatoryStore) MarkMessageArchived(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.client.Doc(collObsMessages, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: "archived"},
		{Path: "archived_at", Value: timeToFirestore(now)},
	})
	return err
}

// --- Conversion helpers ---

func obsMessageToMap(m *obs.Message) map[string]interface{} {
	return map[string]interface{}{
		"id":                  m.ID,
		"task_id":             m.TaskID,
		"inbox":               m.Inbox,
		"from_agent":          m.FromAgent,
		"title":               m.Title,
		"content":             m.Content,
		"message_type":        m.MessageType,
		"status":              string(m.Status),
		"priority":            m.Priority,
		"github_issue_number": m.GitHubIssueNumber,
		"github_repo":         m.GitHubRepo,
		"correlation_id":      m.CorrelationID,
		"reply_to_id":         m.ReplyToID,
		"content_hash":        m.ContentHash,
		"created_at":          timeToFirestore(m.CreatedAt),
		"read_at":             timePtrToFirestore(m.ReadAt),
		"archived_at":         timePtrToFirestore(m.ArchivedAt),
	}
}

func mapToObsMessage(data map[string]interface{}) *obs.Message {
	return &obs.Message{
		ID:                getString(data, "id"),
		TaskID:            getString(data, "task_id"),
		Inbox:             getString(data, "inbox"),
		FromAgent:         getString(data, "from_agent"),
		Title:             getString(data, "title"),
		Content:           getString(data, "content"),
		MessageType:       getString(data, "message_type"),
		Status:            obs.MessageStatus(getString(data, "status")),
		Priority:          getString(data, "priority"),
		GitHubIssueNumber: getInt(data, "github_issue_number"),
		GitHubRepo:        getString(data, "github_repo"),
		CorrelationID:     getString(data, "correlation_id"),
		ReplyToID:         getString(data, "reply_to_id"),
		ContentHash:       getString(data, "content_hash"),
		CreatedAt:         snapshotToTime(data, "created_at"),
		ReadAt:            snapshotToTimePtr(data, "read_at"),
		ArchivedAt:        snapshotToTimePtr(data, "archived_at"),
	}
}
