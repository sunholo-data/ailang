package notify

import "context"

// Channel is an outbound notification transport. Implementations deliver a
// Notification (the same struct used by the macOS Notify path) to their medium.
// The notification daemon (internal/daemon) fans out to one or more registered
// channels; the macOS notifier and the Discord webhook are both Channels.
//
// This is the Go port of the design behind Aitana's v6 Channels Framework
// (aitana-labs/aitana-skills/backend/channels): a tiny interface plus a Registry
// that acts as the output router. It is outbound-first — the inbound half
// (webhook verify/parse → reply-to-approve) is deferred and can be added as an
// optional sub-interface without breaking outbound adapters.
type Channel interface {
	// Name is the channel's stable identifier (and its config key), e.g. "macos",
	// "discord". Used by the registry and per-channel routing filters.
	Name() string
	// Send delivers n via the channel's medium. The destination is part of the
	// channel's own configuration (a webhook URL, the local desktop), so callers
	// do not pass one. Implementations handle their own length limits (chunking).
	// A non-nil error means delivery failed and the caller should treat the event
	// as undelivered (nack / dead-letter).
	Send(ctx context.Context, n Notification) error
}

// LocalChannel is implemented by channels whose delivery is best-effort and
// local (desktop notifications). The fan-out fires them but does NOT count their
// success toward an ack and does NOT retry on their failure — because a local
// notifier can report success merely by queueing a banner the (absent) user
// never sees. Remote/push channels (Discord, etc.) do not implement this and are
// treated as authoritative: an ack requires every remote channel to succeed.
type LocalChannel interface {
	IsLocal() bool
}

// isLocal reports whether ch declared itself best-effort/local.
func isLocal(ch Channel) bool {
	lc, ok := ch.(LocalChannel)
	return ok && lc.IsLocal()
}

// EventFilter is implemented by channels that want to opt out of certain event
// types — e.g. Discord declines `completed` task events to keep phone pings to
// "needs my attention" only, while macOS continues to receive everything as a
// passive desktop feed. A channel without this method (or whose Accepts returns
// true) is treated as "accepts every event type".
//
// The fan-out treats a filtered-out send as a SKIP, not a failure: it does not
// count toward the remote-authoritative ack quota and does not trigger a retry.
type EventFilter interface {
	Accepts(eventType string) bool
}

// channelAccepts returns true if ch wants n delivered. Channels without
// EventFilter implicitly accept everything.
func channelAccepts(ch Channel, n Notification) bool {
	f, ok := ch.(EventFilter)
	if !ok {
		return true
	}
	return f.Accepts(n.EventType)
}

// MacOSChannel adapts the existing macOS Notify path to the Channel interface,
// so the long-standing local-desktop notifier is just another registered
// channel alongside Discord et al.
type MacOSChannel struct{}

// Name implements Channel.
func (MacOSChannel) Name() string { return "macos" }

// Send fires a native macOS notification. Returns ErrNotifierUnavailable on
// non-Darwin hosts or when no notifier binary is present — callers may treat
// that as a soft degradation rather than a hard failure.
func (MacOSChannel) Send(_ context.Context, n Notification) error { return Notify(n) }

// IsLocal marks macOS as a best-effort local channel: it reports success as soon
// as a banner is queued (even if the user is away), so it must not gate acks or
// trigger retries. Its per-task/per-inbox Group coalesces any duplicate banners.
func (MacOSChannel) IsLocal() bool { return true }
