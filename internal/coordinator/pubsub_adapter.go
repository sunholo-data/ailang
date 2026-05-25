package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// PubSubInboxAdapter buffers incoming message notifications for the coordinator.
// Messages arrive either via pull subscription (Start) or push HTTP endpoint
// (HandleNotification). ListUnread() drains the buffer.
//
// M-COORD-MULTI-HOST-WORKERS (v0.24.0): adapters can advertise worker tags so
// the same Pub/Sub topic can carry tag-routed messages claimed only by hosts
// whose advertised tags ⊇ the message's required tag set. Call SetWorkerTags
// after construction to opt in; adapters without tags advertised reject
// any message that has a non-empty `requires` attribute.
type PubSubInboxAdapter struct {
	subscriber *pubsub.Subscriber
	msgStore   messaging.MessageStore // For fetching full message content from Firestore
	subName    string
	inbox      string
	logger     *log.Logger

	mu       sync.Mutex
	buffered []*Message
	running  bool

	// M-COORD-MULTI-HOST-WORKERS v0.24.0: worker identity + advertised tags
	// used for tag-subset filtering of incoming messages. Read protected by mu.
	hostID         string
	advertisedTags []string
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

// SetWorkerTags advertises this adapter's host identity and tag set for
// tag-routed message filtering (M-COORD-MULTI-HOST-WORKERS, v0.24.0).
// Safe to call after construction and before Start(). Empty tags = match-all
// for legacy single-host setups; non-empty tags reject messages whose
// `requires` attribute names tags this adapter does not advertise.
func (a *PubSubInboxAdapter) SetWorkerTags(hostID string, tags []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.hostID = hostID
	// Copy to avoid sharing the caller's slice.
	a.advertisedTags = append([]string{}, tags...)
}

// parseRequiresAttr parses the comma-separated `requires` Pub/Sub attribute
// into a clean tag list (trimmed, empty entries dropped). An empty / missing
// attribute returns nil — interpreted by the caller as "no constraint".
func parseRequiresAttr(attrs map[string]string) []string {
	raw, ok := attrs["requires"]
	if !ok {
		return nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// shouldClaim returns true iff this adapter should process and ack the
// notification, given the message's required tag set. An empty `required`
// list ALWAYS returns true (no constraint). Otherwise, this adapter's
// advertised tags must satisfy the requirement (set inclusion + glob).
func (a *PubSubInboxAdapter) shouldClaim(required []string) bool {
	if len(required) == 0 {
		return true
	}
	a.mu.Lock()
	advertised := a.advertisedTags
	a.mu.Unlock()
	return TagMatches(required, advertised)
}

// HandleNotification processes a message notification from either pull subscription
// or push HTTP endpoint. It decodes the notification, fetches full content from
// Firestore, and buffers the message for ListUnread().
//
// M-COORD-MULTI-HOST-WORKERS (v0.24.0): if the message's `requires` attribute
// names tags this adapter does not advertise, the function returns a non-nil
// error WITHOUT fetching from the message store. Pub/Sub's redelivery
// semantics treat that error as a NACK, leaving the message available for
// another worker. Error string contains "tag filter" for observability.
func (a *PubSubInboxAdapter) HandleNotification(data []byte, attrs map[string]string) error {
	// Tag-routing pre-flight: reject (nack) BEFORE any Firestore fetch if the
	// message's `requires` set is not satisfied by our advertised tags.
	if required := parseRequiresAttr(attrs); len(required) > 0 {
		if !a.shouldClaim(required) {
			a.mu.Lock()
			hostID := a.hostID
			adv := a.advertisedTags
			a.mu.Unlock()
			a.logger.Printf("PubSubInboxAdapter: tag filter rejected message: required=%v, advertised=%v, host=%s",
				required, adv, hostID)
			return fmt.Errorf("tag filter: requires=%v not satisfied by advertised=%v", required, adv)
		}
	}

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
