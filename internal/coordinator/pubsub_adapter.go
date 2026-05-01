package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
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
	// M-PKG-CASCADE-DETERMINISTIC-FIRST: cascade messages now embed the full
	// envelope in the data field. Try that decode first; fall back to the
	// legacy notification-only decode if the data isn't an envelope. This
	// keeps backward compatibility with older publishers.
	var (
		notification pubsub.MessageNotification
		envelope     *pubsub.CascadeEnvelopeFields
	)
	var cascadeData pubsub.CascadeMessageData
	if cdErr := json.Unmarshal(data, &cascadeData); cdErr == nil && cascadeData.MessageID != "" {
		notification.MessageID = cascadeData.MessageID
		envelope = cascadeData.Envelope
	} else {
		decoded, err := pubsub.DecodeMessageNotification(data)
		if err != nil {
			a.logger.Printf("PubSubInboxAdapter: failed to decode notification: %v", err)
			return nil // Return nil to ack — avoid retry loop on bad data.
		}
		notification = decoded
	}

	msgAttrs := pubsub.AttributesFromMap(attrs)

	// Build coordinator Message from notification + attributes.
	msg := &Message{
		ID:      notification.MessageID,
		From:    msgAttrs.FromAgent,
		Inbox:   msgAttrs.Inbox,
		Title:   fmt.Sprintf("Pub/Sub notification from %s", msgAttrs.FromAgent),
		Content: notification.MessageID, // Fallback: just the ID
		Type:    msgAttrs.Category,
		Kind:    msgAttrs.MessageType,
		// M-PKG-AUTONOMOUS-CASCADE-SAFE M1: surface the source topic so
		// downstream agents can distinguish authoritative cascade-driven
		// bumps from public-routed feedback.
		Source:    msgAttrs.Source,
		CreatedAt: time.Now(),
	}

	// M-PKG-CASCADE-DETERMINISTIC-FIRST: populate cascade envelope fields
	// from the embedded Pub/Sub data. These let the cloud coordinator make
	// deterministic dispatch decisions (toml bump vs AI escalation) without
	// fetching from a separate store.
	if envelope != nil {
		msg.RootPackage = envelope.RootPackage
		msg.RootChangeClass = envelope.ChangeClass
		msg.FromVersion = envelope.FromVersion
		msg.ToVersion = envelope.ToVersion
		msg.FromInterfaceHash = envelope.FromInterfaceHash
		msg.ToInterfaceHash = envelope.ToInterfaceHash
		msg.FromContentHash = envelope.FromContentHash
		msg.ToContentHash = envelope.ToContentHash
		msg.EffectsWidened = envelope.EffectsWidened
		msg.PrevEffectCeiling = envelope.PrevEffectCeiling
		msg.NewEffectCeiling = envelope.NewEffectCeiling
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
