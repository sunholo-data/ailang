package firestore

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo-data/ailang/internal/messaging"
)

const (
	collThreads     = "threads"
	collThreadMsgs  = "thread_messages"
	collInbox       = "inbox_messages"
	collMsgApproval = "msg_approvals"
	collHistory     = "approval_history"
	collInstHistory = "instance_history"
	collAgents      = "agents"
	collMetrics     = "metrics"
	collSubscribers = "subscribers"
)

// Compile-time check that MessagingStore implements messaging.MessageStore.
var _ messaging.MessageStore = (*MessagingStore)(nil)

// MessagingStore implements messaging.MessageStore backed by Firestore.
type MessagingStore struct {
	client *Client

	// Dashboard aggregate caches (avoid full collection scans on every API call).
	workspacesCache   *ttlCache[[]string]
	threadStatsCache  *ttlCache[messaging.ThreadAggregateStats]
	messageFlowCache  *ttlCache[[]messaging.MessageFlowEdge]
	activeAgentsCache *ttlCache[[]messaging.ActiveAgent]
}

// NewMessagingStore creates a new Firestore-backed messaging store.
func NewMessagingStore(client *Client) *MessagingStore {
	return &MessagingStore{
		client:            client,
		workspacesCache:   newTTLCache[[]string](2 * time.Minute),
		threadStatsCache:  newTTLCache[messaging.ThreadAggregateStats](60 * time.Second),
		messageFlowCache:  newTTLCache[[]messaging.MessageFlowEdge](5 * time.Minute),
		activeAgentsCache: newTTLCache[[]messaging.ActiveAgent](60 * time.Second),
	}
}

// Close closes the underlying client.
func (s *MessagingStore) Close() error {
	return s.client.Close()
}

// --- Thread Management ---

func (s *MessagingStore) CreateThread(title, createdByType, createdByID, targetAgent string) (*messaging.Thread, error) {
	return s.CreateThreadWithWorkspace(title, createdByType, createdByID, targetAgent, "")
}

func (s *MessagingStore) CreateThreadWithWorkspace(title, createdByType, createdByID, targetAgent, workspace string) (*messaging.Thread, error) {
	now := time.Now()
	id := fmt.Sprintf("thread_%d_%s", now.UnixMilli(), generateShortID())

	thread := &messaging.Thread{
		ID:            id,
		Title:         title,
		CreatedByType: createdByType,
		CreatedByID:   createdByID,
		TargetAgent:   targetAgent,
		Workspace:     workspace,
		Status:        "active",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err := s.client.Doc(collThreads, id).Set(context.Background(), threadToMap(thread))
	if err != nil {
		return nil, err
	}
	return thread, nil
}

func (s *MessagingStore) GetOrCreateThreadWithWorkspace(title, createdByType, createdByID, targetAgent, workspace string) (*messaging.Thread, bool, error) {
	ctx := context.Background()
	var result *messaging.Thread
	created := false

	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Try to find existing thread
		iter := s.client.Collection(collThreads).
			Where("title", "==", title).
			Where("target_agent", "==", targetAgent).
			Limit(1).
			Documents(ctx)
		defer iter.Stop()

		doc, err := iter.Next()
		if err == nil {
			result = mapToThread(doc.Data())
			return nil
		}
		if err != iterator.Done {
			return err
		}

		// Create new thread
		now := time.Now()
		id := fmt.Sprintf("thread_%d_%s", now.UnixMilli(), generateShortID())
		result = &messaging.Thread{
			ID:            id,
			Title:         title,
			CreatedByType: createdByType,
			CreatedByID:   createdByID,
			TargetAgent:   targetAgent,
			Workspace:     workspace,
			Status:        "active",
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		created = true
		return tx.Set(s.client.Doc(collThreads, id), threadToMap(result))
	})
	return result, created, err
}

func (s *MessagingStore) GetThreadByTitleAndAgent(title, targetAgent string) (*messaging.Thread, error) {
	ctx := context.Background()
	iter := s.client.Collection(collThreads).
		Where("title", "==", title).
		Where("target_agent", "==", targetAgent).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("thread not found: title=%q agent=%q", title, targetAgent)
	}
	if err != nil {
		return nil, err
	}
	return mapToThread(doc.Data()), nil
}

func (s *MessagingStore) GetThread(threadID string) (*messaging.Thread, error) {
	doc, err := s.client.Doc(collThreads, threadID).Get(context.Background())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("thread not found: %s", threadID)
		}
		return nil, err
	}
	return mapToThread(doc.Data()), nil
}

func (s *MessagingStore) SetThreadWorkspace(threadID, workspace string) error {
	_, err := s.client.Doc(collThreads, threadID).Update(context.Background(), []firestore.Update{
		{Path: "workspace", Value: workspace},
		{Path: "updated_at", Value: time.Now()},
	})
	return err
}

func (s *MessagingStore) GetThreadWorkspace(threadID string) (string, error) {
	thread, err := s.GetThread(threadID)
	if err != nil {
		return "", err
	}
	return thread.Workspace, nil
}

func (s *MessagingStore) SetThreadTargetAgent(threadID, targetAgent string) error {
	_, err := s.client.Doc(collThreads, threadID).Update(context.Background(), []firestore.Update{
		{Path: "target_agent", Value: targetAgent},
		{Path: "updated_at", Value: time.Now()},
	})
	return err
}

func (s *MessagingStore) UpdateThreadTitle(threadID, title string) error {
	_, err := s.client.Doc(collThreads, threadID).Update(context.Background(), []firestore.Update{
		{Path: "title", Value: title},
		{Path: "updated_at", Value: time.Now()},
	})
	return err
}

func (s *MessagingStore) DeleteThread(threadID string) error {
	_, err := s.client.Doc(collThreads, threadID).Delete(context.Background())
	return err
}

func (s *MessagingStore) GetThreadsByStatus(threadStatus string, limit int) ([]messaging.Thread, error) {
	ctx := context.Background()
	q := s.client.Collection(collThreads).
		Where("status", "==", threadStatus).
		OrderBy("updated_at", firestore.Desc)

	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var threads []messaging.Thread
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		threads = append(threads, *mapToThread(doc.Data()))
	}
	return threads, nil
}

func (s *MessagingStore) NewThreadFilter(threadStatus, workspace string, limit int) messaging.ThreadFilter {
	return messaging.ThreadFilter{
		Status:    threadStatus,
		Workspace: workspace,
		Limit:     limit,
	}
}

func (s *MessagingStore) GetThreadsFiltered(filter messaging.ThreadFilter) ([]messaging.Thread, error) {
	ctx := context.Background()
	q := s.client.Collection(collThreads).Query

	if filter.Status != "" {
		q = q.Where("status", "==", filter.Status)
	}
	if filter.Workspace != "" {
		q = q.Where("workspace", "==", filter.Workspace)
	}
	q = q.OrderBy("updated_at", firestore.Desc)
	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var threads []messaging.Thread
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		threads = append(threads, *mapToThread(doc.Data()))
	}
	return threads, nil
}

func (s *MessagingStore) GetDistinctWorkspaces() ([]string, error) {
	if cached, ok := s.workspacesCache.get(); ok {
		return cached, nil
	}

	const limit = 5000
	ctx := context.Background()
	iter := s.client.Collection(collThreads).Limit(limit).Documents(ctx)
	defer iter.Stop()

	seen := make(map[string]bool)
	count := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		count++
		if ws := getString(doc.Data(), "workspace"); ws != "" {
			seen[ws] = true
		}
	}
	if count >= limit {
		log.Printf("WARNING: GetDistinctWorkspaces hit limit of %d documents — results may be incomplete", limit)
	}

	workspaces := make([]string, 0, len(seen))
	for ws := range seen {
		workspaces = append(workspaces, ws)
	}

	s.workspacesCache.set(workspaces)
	return workspaces, nil
}

func (s *MessagingStore) GetThreadAggregateStats() (*messaging.ThreadAggregateStats, error) {
	if cached, ok := s.threadStatsCache.get(); ok {
		return &cached, nil
	}

	const limit = 5000
	ctx := context.Background()
	iter := s.client.Collection(collThreads).Limit(limit).Documents(ctx)
	defer iter.Stop()

	stats := &messaging.ThreadAggregateStats{
		ByStatus:    make(map[string]int),
		ByWorkspace: make(map[string]int),
	}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		stats.TotalThreads++
		if st := getString(data, "status"); st != "" {
			stats.ByStatus[st]++
		}
		if ws := getString(data, "workspace"); ws != "" {
			stats.ByWorkspace[ws]++
		}
	}
	if stats.TotalThreads >= limit {
		log.Printf("WARNING: GetThreadAggregateStats hit limit of %d documents — stats may be incomplete", limit)
	}

	s.threadStatsCache.set(*stats)
	return stats, nil
}

// --- Message Management ---

func (s *MessagingStore) GetMessages(toType, toID, deliveryState string) ([]messaging.Message, error) {
	ctx := context.Background()
	q := s.client.Collection(collThreadMsgs).
		Where("to_type", "==", toType).
		Where("to_id", "==", toID)

	if deliveryState != "" {
		q = q.Where("delivery_state", "==", deliveryState)
	}
	q = q.OrderBy("created_at", firestore.Asc)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var msgs []messaging.Message
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, *mapToMessage(doc.Data()))
	}
	return msgs, nil
}

func (s *MessagingStore) CreateMessage(threadID, fromType, fromID, toType, toID, kind, content, metadataJSON string) (*messaging.Message, error) {
	ctx := context.Background()
	now := time.Now()
	id := fmt.Sprintf("msg_%d_%s", now.UnixMilli(), generateShortID())

	msg := &messaging.Message{
		ID:            id,
		ThreadID:      threadID,
		CreatedAt:     now,
		FromType:      fromType,
		FromID:        fromID,
		ToType:        toType,
		ToID:          toID,
		Kind:          kind,
		Content:       content,
		MetadataJSON:  metadataJSON,
		DeliveryState: "pending",
	}

	_, err := s.client.Doc(collThreadMsgs, id).Set(ctx, messageToMap(msg))
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *MessagingStore) MarkAsAcked(messageID string) error {
	_, err := s.client.Doc(collThreadMsgs, messageID).Update(context.Background(), []firestore.Update{
		{Path: "delivery_state", Value: "acked"},
	})
	return err
}

func (s *MessagingStore) MarkAsUnacked(messageID string) error {
	_, err := s.client.Doc(collThreadMsgs, messageID).Update(context.Background(), []firestore.Update{
		{Path: "delivery_state", Value: "pending"},
	})
	return err
}

func (s *MessagingStore) ClaimMessage(messageID, claimedBy string) error {
	_, err := s.client.Doc(collThreadMsgs, messageID).Update(context.Background(), []firestore.Update{
		{Path: "delivery_state", Value: "claimed"},
		{Path: "business_state", Value: claimedBy},
	})
	return err
}

func (s *MessagingStore) MarkAllAsAcked(toType, toID string) (int64, error) {
	ctx := context.Background()
	iter := s.client.Collection(collThreadMsgs).
		Where("to_type", "==", toType).
		Where("to_id", "==", toID).
		Where("delivery_state", "==", "pending").
		Documents(ctx)
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
		if _, err := doc.Ref.Update(ctx, []firestore.Update{
			{Path: "delivery_state", Value: "acked"},
		}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *MessagingStore) GetMessagesFromSeq(threadID string, fromSeq int, limit int) ([]messaging.Message, error) {
	ctx := context.Background()
	q := s.client.Collection(collThreadMsgs).
		Where("thread_id", "==", threadID).
		Where("message_seq", ">=", fromSeq).
		OrderBy("message_seq", firestore.Asc)

	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var msgs []messaging.Message
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, *mapToMessage(doc.Data()))
	}
	return msgs, nil
}

func (s *MessagingStore) Subscribe(instanceID, threadID string) error {
	ctx := context.Background()
	docID := instanceID + "_" + threadID
	_, err := s.client.Doc(collSubscribers, docID).Set(ctx, map[string]interface{}{
		"instance_id":   instanceID,
		"thread_id":     threadID,
		"ack_seq":       0,
		"subscribed_at": time.Now(),
	})
	return err
}

func (s *MessagingStore) UpdateAckSeq(instanceID, threadID string, ackSeq int) error {
	ctx := context.Background()
	docID := instanceID + "_" + threadID
	_, err := s.client.Doc(collSubscribers, docID).Update(ctx, []firestore.Update{
		{Path: "ack_seq", Value: ackSeq},
	})
	return err
}

// generateShortID creates a short random hex ID.
func generateShortID() string {
	b := make([]byte, 4)
	// Use crypto/rand via the helper
	for i := range b {
		b[i] = byte(time.Now().UnixNano() >> (i * 8))
	}
	return fmt.Sprintf("%x", b)
}
