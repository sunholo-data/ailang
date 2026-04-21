package server

import (
	"context"
	"encoding/json"
	"log"

	"github.com/sunholo-data/ailang/internal/pubsub"
	"github.com/sunholo-data/ailang/internal/websocket"
)

// PubSubEventSubscriber pulls real-time task stream events from the
// ailang-events-dashboard Pub/Sub subscription and broadcasts them
// to connected WebSocket clients. Only active in cloud mode.
type PubSubEventSubscriber struct {
	subscriber *pubsub.Subscriber
	wsServer   *websocket.Server
	subName    string
	logger     *log.Logger
	cancel     context.CancelFunc
}

// NewPubSubEventSubscriber creates a subscriber that bridges Pub/Sub events
// to WebSocket broadcasts. The subName should include the topic prefix
// (e.g., "ailang-dev-events-dashboard").
func NewPubSubEventSubscriber(subscriber *pubsub.Subscriber, wsServer *websocket.Server, subName string, logger *log.Logger) *PubSubEventSubscriber {
	return &PubSubEventSubscriber{
		subscriber: subscriber,
		wsServer:   wsServer,
		subName:    subName,
		logger:     logger,
	}
}

// Start begins pulling events from Pub/Sub. Blocks until ctx is cancelled
// or an unrecoverable error occurs. Should be called in a goroutine.
func (s *PubSubEventSubscriber) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)

	s.logger.Printf("PubSubEventSubscriber: starting on subscription %s", s.subName)

	if err := s.subscriber.Subscribe(ctx, s.subName, s.handleEvent); err != nil {
		// context.Canceled is expected on shutdown.
		if ctx.Err() == nil {
			s.logger.Printf("PubSubEventSubscriber: subscription error: %v", err)
		}
	}

	s.logger.Printf("PubSubEventSubscriber: stopped")
}

// Stop cancels the subscription pull loop.
func (s *PubSubEventSubscriber) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// handleEvent deserializes a TaskStreamEvent from Pub/Sub and broadcasts it.
func (s *PubSubEventSubscriber) handleEvent(ctx context.Context, data []byte, attrs map[string]string) error {
	var event websocket.TaskStreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		// Ack malformed messages to prevent infinite retry.
		s.logger.Printf("PubSubEventSubscriber: bad event JSON (acking to prevent retry): %v", err)
		return nil
	}

	s.wsServer.BroadcastTaskEvent(&event)
	return nil // Ack
}
