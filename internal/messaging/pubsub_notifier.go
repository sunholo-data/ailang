package messaging

import (
	"context"
	"fmt"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// PubSubNotifier wraps a Pub/Sub publisher for sending message notifications.
// It is created from config and provides a simple Notify() method for dual-write.
type PubSubNotifier struct {
	client    *pubsub.Client
	publisher *pubsub.Publisher
}

// NewPubSubNotifier creates a notifier from the messaging config.
// Returns nil (not an error) if pubsub is not enabled.
func NewPubSubNotifier(cfg *PubSubConfig) (*PubSubNotifier, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}

	ctx := context.Background()
	projectID := cfg.ProjectID
	if projectID == "" {
		p, err := config.CloudProject(ctx)
		if err != nil {
			return nil, fmt.Errorf("pubsub enabled but no project_id set: %w", err)
		}
		projectID = p
	}

	prefix := cfg.TopicPrefix
	if prefix == "" {
		prefix = pubsub.DefaultTopicPrefix
	}

	client, err := pubsub.NewClient(ctx, projectID, prefix)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client: %w", err)
	}

	return &PubSubNotifier{
		client:    client,
		publisher: pubsub.NewPublisher(client),
	}, nil
}

// Notify publishes a message notification to Pub/Sub.
// This should be called AFTER the message is durably stored in SQLite/Firestore.
func (n *PubSubNotifier) Notify(ctx context.Context, msg *InboxMessage) error {
	if n == nil {
		return nil
	}

	attrs := pubsub.MessageAttributes{
		Inbox:       msg.ToInbox,
		FromAgent:   msg.FromAgent,
		Category:    msg.Category,
		MessageType: msg.MessageType,
	}

	return n.publisher.PublishMessage(ctx, NotificationIDFor(msg), attrs)
}

// NotificationIDFor is the identifier a notification must carry so the RECIPIENT
// can fetch the message back.
//
// The two stores resolve differently and only one identifier works on both:
//
//	Firestore  GetInboxMessage(id) -> client.Doc(collInbox, id)   — ID ONLY
//	SQLite     GetInboxMessage(id) -> WHERE id = ? OR message_id = ?  — either
//
// So ID is the answer, and MessageID is the answer only by coincidence — on
// Firestore normalizeInboxDefaults sets MessageID = ID, which makes publishing
// MessageID look correct for as long as nothing sets it first. A message that
// picked up a SQLite-style "msg_<ts>_<prefix>" MessageID before reaching
// Firestore publishes an id whose document does not exist, the recipient's
// fetch fails, and the notification is unresolvable.
//
// That is the shape behind the 2026-09-14 content-free tasks: ids like
// "msg_20260914_153340_b787f96a" — the "msg_" form, whose suffix is the first
// eight characters of a UUID, i.e. a message id minted by the SQLite path —
// arriving at a coordinator reading Firestore.
//
// Three publishers existed with two answers (daemon_evaluator.go and
// daemon_http.go send msg.ID; this one sent msg.MessageID). One helper, so
// there is one answer.
func NotificationIDFor(msg *InboxMessage) string {
	if msg == nil {
		return ""
	}
	if msg.ID != "" {
		return msg.ID
	}
	// No ID: pre-insert, or a caller that only set the business id. MessageID
	// still resolves on SQLite, and an empty publish resolves nowhere.
	return msg.MessageID
}

// Close releases Pub/Sub resources.
func (n *PubSubNotifier) Close() {
	if n == nil {
		return
	}
	n.publisher.Stop()
	n.client.Close()
}
