package coordinator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/pubsub"
)

// PubSubInboxAdapter buffers incoming message notifications for the coordinator.
// Messages arrive either via pull subscription (Start) or push HTTP endpoint
// (HandleNotification). ListUnread() drains the buffer.
type PubSubInboxAdapter struct {
	subscriber *pubsub.Subscriber
	msgStore   messaging.MessageStore // For fetching full message content from Firestore
	subName    string
	inbox      string
	logger     *log.Logger

	mu       sync.Mutex
	buffered []*Message
	running  bool
}

// NewPubSubInboxAdapter creates an adapter for receiving message notifications.
// msgStore is used to fetch full message content from Firestore when a notification arrives.
// For pull mode, call Start(). For push mode, the HTTP handler calls HandleNotification() directly.
func NewPubSubInboxAdapter(subscriber *pubsub.Subscriber, subName, inbox string, msgStore messaging.MessageStore, logger *log.Logger) *PubSubInboxAdapter {
	return &PubSubInboxAdapter{
		subscriber: subscriber,
		subName:    subName,
		inbox:      inbox,
		msgStore:   msgStore,
		logger:     logger,
		buffered:   make([]*Message, 0),
	}
}

// HandleNotification processes a message notification from either pull subscription
// or push HTTP endpoint. It decodes the notification, fetches full content from
// Firestore, and buffers the message for ListUnread().
func (a *PubSubInboxAdapter) HandleNotification(data []byte, attrs map[string]string) error {
	notification, err := pubsub.DecodeMessageNotification(data)
	if err != nil {
		a.logger.Printf("PubSubInboxAdapter: failed to decode notification: %v", err)
		return nil // Return nil to ack — avoid retry loop on bad data.
	}

	msgAttrs := pubsub.AttributesFromMap(attrs)

	// Build coordinator Message from notification + attributes.
	msg := &Message{
		ID:        notification.MessageID,
		From:      msgAttrs.FromAgent,
		Inbox:     msgAttrs.Inbox,
		Title:     fmt.Sprintf("Pub/Sub notification from %s", msgAttrs.FromAgent),
		Content:   notification.MessageID, // Fallback: just the ID
		Type:      msgAttrs.Category,
		Kind:      msgAttrs.MessageType,
		CreatedAt: time.Now(),
	}

	// Fetch full message content from Firestore.
	// The Pub/Sub notification is intentionally minimal (just message_id);
	// the actual title, content, and metadata live in Firestore.
	if a.msgStore != nil {
		fullMsg, fetchErr := a.msgStore.GetInboxMessage(notification.MessageID)
		if fetchErr != nil {
			a.logger.Printf("PubSubInboxAdapter: failed to fetch message %s from store: %v (using notification-only data)",
				notification.MessageID, fetchErr)
		} else if fullMsg != nil {
			msg.Title = fullMsg.Title
			msg.Content = fullMsg.Payload
			msg.From = fullMsg.FromAgent
			msg.Type = fullMsg.Category
			msg.Inbox = fullMsg.ToInbox
		}
	}

	a.mu.Lock()
	a.buffered = append(a.buffered, msg)
	a.mu.Unlock()

	a.logger.Printf("PubSubInboxAdapter: buffered message %s (inbox=%s, from=%s)",
		notification.MessageID, msg.Inbox, msg.From)
	return nil
}

// Start begins pulling messages from the Pub/Sub subscription in the background.
// Not used in push mode — the HTTP handler calls HandleNotification() directly.
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
			return a.HandleNotification(data, attrs)
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
