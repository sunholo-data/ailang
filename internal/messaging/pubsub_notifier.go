package messaging

import (
	"context"
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/pubsub"
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

	projectID := cfg.ProjectID
	if projectID == "" {
		projectID = os.Getenv("AILANG_CLOUD_PROJECT")
	}
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if projectID == "" {
		return nil, fmt.Errorf("pubsub enabled but no project_id set (config or AILANG_CLOUD_PROJECT env)")
	}

	prefix := cfg.TopicPrefix
	if prefix == "" {
		prefix = pubsub.DefaultTopicPrefix
	}

	ctx := context.Background()
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

	return n.publisher.PublishMessage(ctx, msg.MessageID, attrs)
}

// Close releases Pub/Sub resources.
func (n *PubSubNotifier) Close() {
	if n == nil {
		return
	}
	n.publisher.Stop()
	n.client.Close()
}
