package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo/ailang/internal/messaging"
)

// --- Inbox Management ---

func (s *MessagingStore) InsertInboxMessage(msg *messaging.InboxMessage) error {
	return s.InsertInboxMessageWithContext(context.Background(), msg)
}

func (s *MessagingStore) InsertInboxMessageWithContext(ctx context.Context, msg *messaging.InboxMessage) error {
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("inbox_%d_%s", time.Now().UnixMilli(), generateShortID())
	}
	if msg.Status == "" {
		msg.Status = "unread"
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	_, err := s.client.Doc(collInbox, msg.ID).Set(ctx, inboxToMap(msg))
	return err
}

func (s *MessagingStore) ListInboxMessages(opts messaging.InboxListOptions) ([]messaging.InboxMessage, error) {
	ctx := context.Background()
	q := s.client.Collection(collInbox).Query

	if opts.Inbox != "" {
		q = q.Where("to_inbox", "==", opts.Inbox)
	}
	if opts.UnreadOnly {
		q = q.Where("status", "==", "unread")
	} else if opts.Status != "" {
		q = q.Where("status", "==", opts.Status)
	}
	if opts.FromAgent != "" {
		q = q.Where("from_agent", "==", opts.FromAgent)
	}
	if opts.Collapsed {
		// Firestore doesn't support IS NULL natively in compound queries.
		// We filter dup_of=="" to get non-duplicate messages.
		q = q.Where("dup_of", "==", "")
	}
	if opts.DupOf != "" {
		q = q.Where("dup_of", "==", opts.DupOf)
	}

	q = q.OrderBy("created_at", firestore.Desc)
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var msgs []messaging.InboxMessage
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		m := mapToInbox(doc.Data())

		// Client-side date filtering (Firestore compound query limits)
		if opts.StartDate != "" {
			if start, parseErr := time.Parse("2006-01-02", opts.StartDate); parseErr == nil {
				if m.CreatedAt.Before(start) {
					continue
				}
			}
		}
		if opts.EndDate != "" {
			if end, parseErr := time.Parse("2006-01-02", opts.EndDate); parseErr == nil {
				if m.CreatedAt.After(end.Add(24 * time.Hour)) {
					continue
				}
			}
		}
		msgs = append(msgs, *m)
	}
	return msgs, nil
}

func (s *MessagingStore) GetInboxMessage(id string) (*messaging.InboxMessage, error) {
	doc, err := s.client.Doc(collInbox, id).Get(context.Background())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("inbox message not found: %s", id)
		}
		return nil, err
	}
	return mapToInbox(doc.Data()), nil
}

// FindMessageByPrefix resolves a short ID prefix to a full message ID.
// Firestore doesn't support LIKE queries, so we use range queries on the ID field.
func (s *MessagingStore) FindMessageByPrefix(prefix string) (string, error) {
	// Firestore range query: id >= prefix AND id < prefix + high unicode char
	endPrefix := prefix + "\uf8ff"
	iter := s.client.Collection(collInbox).
		Where("id", ">=", prefix).
		Where("id", "<", endPrefix).
		Limit(2).
		Documents(context.Background())
	defer iter.Stop()

	var matches []string
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", err
		}
		matches = append(matches, doc.Ref.ID)
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no message found with prefix '%s'", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous prefix '%s' matches multiple messages, use a longer prefix", prefix)
	}
}

func (s *MessagingStore) MarkInboxMessageRead(id string) error {
	now := time.Now()
	_, err := s.client.Doc(collInbox, id).Update(context.Background(), []firestore.Update{
		{Path: "status", Value: "read"},
		{Path: "read_at", Value: timeToFirestore(now)},
	})
	return err
}

func (s *MessagingStore) MarkInboxMessageUnread(id string) error {
	_, err := s.client.Doc(collInbox, id).Update(context.Background(), []firestore.Update{
		{Path: "status", Value: "unread"},
		{Path: "read_at", Value: nil},
	})
	return err
}

func (s *MessagingStore) MarkAllInboxMessagesRead(inbox string) (int64, error) {
	ctx := context.Background()
	q := s.client.Collection(collInbox).
		Where("status", "==", "unread")
	if inbox != "" {
		q = q.Where("to_inbox", "==", inbox)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	now := timeToFirestore(time.Now())
	var count int64
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return count, err
		}
		if _, err := doc.Ref.Update(ctx, []firestore.Update{
			{Path: "status", Value: "read"},
			{Path: "read_at", Value: now},
		}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *MessagingStore) ForwardInboxMessage(id string, toInbox string) error {
	_, err := s.client.Doc(collInbox, id).Update(context.Background(), []firestore.Update{
		{Path: "to_inbox", Value: toInbox},
	})
	return err
}

func (s *MessagingStore) InboxMessageExistsByGitHub(repo string, issueNumber int) (bool, error) {
	ctx := context.Background()
	iter := s.client.Collection(collInbox).
		Where("github_repo", "==", repo).
		Where("github_issue", "==", issueNumber).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *MessagingStore) InboxMessageExistsByTitle(inbox string, title string) (string, error) {
	ctx := context.Background()
	iter := s.client.Collection(collInbox).
		Where("to_inbox", "==", inbox).
		Where("title", "==", title).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return getString(doc.Data(), "id"), nil
}

func (s *MessagingStore) UpdateInboxMessageGitHub(messageID string, issueNumber int, repo string) error {
	_, err := s.client.Doc(collInbox, messageID).Update(context.Background(), []firestore.Update{
		{Path: "github_issue", Value: issueNumber},
		{Path: "github_repo", Value: repo},
	})
	return err
}

func (s *MessagingStore) CleanupInboxMessages(olderThan time.Duration, expiredOnly bool) (int64, error) {
	ctx := context.Background()
	cutoff := time.Now().Add(-olderThan)

	var q firestore.Query
	if expiredOnly {
		q = s.client.Collection(collInbox).
			Where("expires_at", "<=", timeToFirestore(time.Now()))
	} else {
		q = s.client.Collection(collInbox).
			Where("created_at", "<=", timeToFirestore(cutoff))
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var count int64
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

func (s *MessagingStore) CountInboxMessagesByStatus(inbox string) (map[string]int64, error) {
	ctx := context.Background()
	q := s.client.Collection(collInbox).Query
	if inbox != "" {
		q = q.Where("to_inbox", "==", inbox)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	counts := make(map[string]int64)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		st := getString(doc.Data(), "status")
		if st != "" {
			counts[st]++
		}
	}
	return counts, nil
}

func (s *MessagingStore) GetMessageFlowEdges() ([]messaging.MessageFlowEdge, error) {
	ctx := context.Background()
	iter := s.client.Collection(collInbox).Documents(ctx)
	defer iter.Stop()

	type edgeKey struct{ from, to string }
	edges := make(map[edgeKey]*messaging.MessageFlowEdge)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		from := getString(data, "from_agent")
		to := getString(data, "to_inbox")
		if from == "" || to == "" {
			continue
		}
		key := edgeKey{from, to}
		if e, ok := edges[key]; ok {
			e.MessageCount++
		} else {
			edges[key] = &messaging.MessageFlowEdge{
				FromAgent:    from,
				ToInbox:      to,
				MessageCount: 1,
			}
		}
	}

	result := make([]messaging.MessageFlowEdge, 0, len(edges))
	for _, e := range edges {
		result = append(result, *e)
	}
	return result, nil
}

func (s *MessagingStore) GetActiveAgents() ([]messaging.ActiveAgent, error) {
	ctx := context.Background()
	iter := s.client.Collection(collInbox).Documents(ctx)
	defer iter.Stop()

	agents := make(map[string]*messaging.ActiveAgent)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		from := getString(data, "from_agent")
		to := getString(data, "to_inbox")

		if from != "" {
			if a, ok := agents[from]; ok {
				a.MessagesSent++
			} else {
				agents[from] = &messaging.ActiveAgent{ID: from, Label: from, MessagesSent: 1}
			}
		}
		if to != "" {
			if a, ok := agents[to]; ok {
				a.MessagesRecv++
			} else {
				agents[to] = &messaging.ActiveAgent{ID: to, Label: to, MessagesRecv: 1}
			}
		}
	}

	result := make([]messaging.ActiveAgent, 0, len(agents))
	for _, a := range agents {
		result = append(result, *a)
	}
	return result, nil
}
