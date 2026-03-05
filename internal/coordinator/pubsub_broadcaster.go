package coordinator

import (
	"context"
	"encoding/json"
	"log"

	"github.com/sunholo/ailang/internal/pubsub"
	"github.com/sunholo/ailang/internal/websocket"
)

// PubSubBroadcaster sends task stream events to the Pub/Sub events topic.
// This replaces HTTPBroadcaster when running in cloud mode, enabling the
// dashboard and laptop to receive events via pull subscriptions.
type PubSubBroadcaster struct {
	publisher *pubsub.Publisher
	workspace string
	logger    *log.Logger
}

// NewPubSubBroadcaster creates a broadcaster that publishes events to Pub/Sub.
func NewPubSubBroadcaster(publisher *pubsub.Publisher, workspace string, logger *log.Logger) *PubSubBroadcaster {
	return &PubSubBroadcaster{
		publisher: publisher,
		workspace: workspace,
		logger:    logger,
	}
}

// Broadcast sends a task event to the Pub/Sub events topic.
func (b *PubSubBroadcaster) Broadcast(event *websocket.TaskStreamEvent) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		b.logger.Printf("PubSubBroadcaster: failed to marshal event: %v", err)
		return
	}

	eventType := string(event.StreamType)
	ctx := context.Background()

	if err := b.publisher.PublishEvent(ctx, eventJSON, eventType, event.TaskID, b.workspace); err != nil {
		b.logger.Printf("PubSubBroadcaster: failed to publish event (type=%s, task=%s): %v",
			eventType, event.TaskID, err)
	}
}

// BroadcastFunc returns the EventBroadcaster function.
func (b *PubSubBroadcaster) BroadcastFunc() EventBroadcaster {
	return b.Broadcast
}
