package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/notify"
)

// secret_approval_bridge.go — the consume half of the secret-approval ntfy push
// (M-SECRET-REMOTE-APPROVAL-WIRING). The dashboard publishes a kind=approval
// event to the approvals topic; the coordinator's /pubsub/push handler routes it
// here, and this forwards the (value-free) notification to the ntfy service so it
// reaches the operator's phone with Approve/Deny action buttons.

// approvalDedupTTL bounds how long a forwarded push message ID is remembered for
// duplicate suppression. Pub/Sub redelivers within the ack deadline / retention
// window (minutes), so an hour is comfortably longer than any real redelivery.
const approvalDedupTTL = time.Hour

// approvalDedup drops duplicate Pub/Sub push deliveries by message ID. Pub/Sub is
// at-least-once: the same publish can be (re)delivered more than once (ack
// deadline exceeded, or genuine duplication), and push subscriptions can't use
// exactly-once. Without this, a redelivery forwards a second identical
// notification to the operator's phone. Entries are pruned lazily on access;
// approvals are infrequent so the map stays small.
type approvalDedup struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

func newApprovalDedup() *approvalDedup {
	return &approvalDedup{seen: make(map[string]time.Time)}
}

// markIfNew records id and returns true if it had not been seen within the TTL.
// It returns false for a duplicate. Marking is atomic with the check so two
// near-simultaneous redeliveries can't both pass; on a downstream failure the
// caller must unmark so the Pub/Sub retry can re-forward.
func (d *approvalDedup) markIfNew(id string) bool {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	if t, ok := d.seen[id]; ok && now.Sub(t) < approvalDedupTTL {
		return false
	}
	for k, t := range d.seen {
		if now.Sub(t) >= approvalDedupTTL {
			delete(d.seen, k)
		}
	}
	d.seen[id] = now
	return true
}

// unmark forgets id so a redelivery is forwarded again (used when the forward
// failed and we want Pub/Sub's retry to actually push).
func (d *approvalDedup) unmark(id string) {
	d.mu.Lock()
	delete(d.seen, id)
	d.mu.Unlock()
}

// handlePushApproval forwards a kind=approval push event to ntfy. The push data
// is a marshalled notify.Notification built dashboard-side (it carries the
// reference, purpose, and signed action URLs — never a resolved value).
//
// msgID is the Pub/Sub message ID, used to suppress duplicate deliveries.
//
// Configured from env: AILANG_NTFY_SERVER_URL, AILANG_NTFY_TOPIC,
// AILANG_NTFY_AUTH_TOKEN. With ntfy unconfigured it logs and acks (no-op) rather
// than failing the push; a transient ntfy error is returned so Pub/Sub retries.
func (d *Daemon) handlePushApproval(ctx context.Context, msgID string, data []byte, attrs map[string]string) error {
	approvalID := attrs["approval_id"]

	// Suppress at-least-once duplicate deliveries by message ID.
	d.approvalDedupOnce.Do(func() { d.approvalDedup = newApprovalDedup() })
	if msgID != "" && !d.approvalDedup.markIfNew(msgID) {
		d.logger.Printf("Push approval %s: duplicate delivery (msg %s) — skipping", approvalID, msgID)
		return nil
	}

	serverURL := os.Getenv("AILANG_NTFY_SERVER_URL")
	topic := os.Getenv("AILANG_NTFY_TOPIC")
	if serverURL == "" || topic == "" {
		d.logger.Printf("Push approval %s: ntfy not configured (AILANG_NTFY_SERVER_URL/TOPIC) — skipping push", approvalID)
		return nil
	}

	var n notify.Notification
	if err := json.Unmarshal(data, &n); err != nil {
		// Malformed payload: ack (don't retry forever), but report.
		d.logger.Printf("Push approval %s: bad notification payload: %v", approvalID, err)
		return nil
	}

	ch := notify.NewNtfyChannel(serverURL, topic, os.Getenv("AILANG_NTFY_AUTH_TOKEN"))
	if err := ch.Send(ctx, n); err != nil {
		// Forward failed: forget the message ID so Pub/Sub's retry re-pushes.
		if msgID != "" {
			d.approvalDedup.unmark(msgID)
		}
		return fmt.Errorf("ntfy send for approval %s: %w", approvalID, err)
	}
	d.logger.Printf("Push approval %s: forwarded to ntfy topic %s", approvalID, topic)
	return nil
}
