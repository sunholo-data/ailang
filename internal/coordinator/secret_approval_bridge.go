package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/notify"
)

// secret_approval_bridge.go — the consume half of the secret-approval ntfy push
// (M-SECRET-REMOTE-APPROVAL-WIRING). The dashboard publishes a kind=approval
// event to the approvals topic; the coordinator's /pubsub/push handler routes it
// here, and this forwards the (value-free) notification to the ntfy service so it
// reaches the operator's phone with Approve/Deny action buttons.

// handlePushApproval forwards a kind=approval push event to ntfy. The push data
// is a marshalled notify.Notification built dashboard-side (it carries the
// reference, purpose, and signed action URLs — never a resolved value).
//
// Configured from env: AILANG_NTFY_SERVER_URL, AILANG_NTFY_TOPIC,
// AILANG_NTFY_AUTH_TOKEN. With ntfy unconfigured it logs and acks (no-op) rather
// than failing the push; a transient ntfy error is returned so Pub/Sub retries.
func (d *Daemon) handlePushApproval(ctx context.Context, data []byte, attrs map[string]string) error {
	approvalID := attrs["approval_id"]
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
		return fmt.Errorf("ntfy send for approval %s: %w", approvalID, err)
	}
	d.logger.Printf("Push approval %s: forwarded to ntfy topic %s", approvalID, topic)
	return nil
}
