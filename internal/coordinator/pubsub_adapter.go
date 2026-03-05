package coordinator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/pubsub"
)

// PubSubInboxAdapter implements MessageStore by pulling messages from a
// Pub/Sub subscription. Messages are buffered locally so ListUnread() returns
// immediately without blocking.
type PubSubInboxAdapter struct {
	subscriber *pubsub.Subscriber
	subName    string
	inbox      string
	logger     *log.Logger

	mu       sync.Mutex
	buffered []*Message
	running  bool
}

// NewPubSubInboxAdapter creates an adapter that pulls from a Pub/Sub subscription.
// Start() must be called to begin receiving messages.
func NewPubSubInboxAdapter(subscriber *pubsub.Subscriber, subName, inbox string, logger *log.Logger) *PubSubInboxAdapter {
	return &PubSubInboxAdapter{
		subscriber: subscriber,
		subName:    subName,
		inbox:      inbox,
		logger:     logger,
		buffered:   make([]*Message, 0),
	}
}

// Start begins pulling messages from the Pub/Sub subscription in the background.
// Messages are decoded and buffered for ListUnread().
func (a *PubSubInboxAdapter) Start(ctx context.Context) {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return
	}
	a.running = true
	a.mu.Unlock()

	go func() {
		err := a.subscriber.Subscribe(ctx, a.subName, func(ctx context.Context, data []byte, attrs map[string]string) error {
			notification, err := pubsub.DecodeMessageNotification(data)
			if err != nil {
				a.logger.Printf("PubSubInboxAdapter: failed to decode notification: %v", err)
				return nil // Ack to avoid retry loop on bad data.
			}

			msgAttrs := pubsub.AttributesFromMap(attrs)

			// Build coordinator Message from notification + attributes.
			msg := &Message{
				ID:        notification.MessageID,
				From:      msgAttrs.FromAgent,
				Title:     fmt.Sprintf("Pub/Sub notification from %s", msgAttrs.FromAgent),
				Content:   notification.MessageID, // Content lives in Firestore; this is just the ID.
				Type:      msgAttrs.Category,
				Kind:      msgAttrs.MessageType,
				CreatedAt: time.Now(),
			}

			a.mu.Lock()
			a.buffered = append(a.buffered, msg)
			a.mu.Unlock()

			a.logger.Printf("PubSubInboxAdapter: buffered message %s (inbox=%s, from=%s)",
				notification.MessageID, a.inbox, msgAttrs.FromAgent)
			return nil
		})
		if err != nil && ctx.Err() == nil {
			a.logger.Printf("PubSubInboxAdapter: subscription %s error: %v", a.subName, err)
		}

		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()
}

// ListUnread returns buffered messages and clears the buffer.
func (a *PubSubInboxAdapter) ListUnread() ([]*Message, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.buffered) == 0 {
		return nil, nil
	}

	msgs := a.buffered
	a.buffered = make([]*Message, 0)
	return msgs, nil
}

// MarkAsRead is a no-op for Pub/Sub — messages are acked on receipt.
func (a *PubSubInboxAdapter) MarkAsRead(_ string) error {
	return nil
}

// Compile-time check that PubSubInboxAdapter implements MessageStore.
var _ MessageStore = (*PubSubInboxAdapter)(nil)
