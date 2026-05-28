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
