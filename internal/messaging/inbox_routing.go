package messaging

// UnroutedInbox receives messages that reached the store with no destination.
const UnroutedInbox = "unrouted"

// NormalizeInboxRouting redirects a message with an empty ToInbox to UnroutedInbox,
// reporting whether it had to. It returns false for messages that were already addressed.
//
// A message stored with ToInbox == "" is not merely mislabelled, it is UNREACHABLE:
// every read path filters by inbox, so the row is retained and billed but can never be
// listed, acked, or triaged. Measured 2026-08-25: 36 such messages in prod and 787 in
// dev — all of them task-failure notifications, i.e. exactly the traffic whose whole
// purpose is to be seen. They were produced by completion handlers that address the
// message to task.AgentID, which is empty whenever the originating inbox had no agent
// registered for it.
//
// This is deliberately a REROUTE rather than an error: these are failure notifications,
// so refusing the insert would trade an invisible record for no record at all. It is not
// a silent fallback — the destination is a real, listable inbox, and the span carries
// message.rerouted_from_empty_inbox so the redirect is greppable in telemetry. The
// upstream refusal in coordinator.resolveInboxAgent is what stops these being generated;
// this is the backstop for paths that do not go through it.
func NormalizeInboxRouting(msg *InboxMessage) bool {
	if msg == nil || msg.ToInbox != "" {
		return false
	}
	msg.ToInbox = UnroutedInbox
	return true
}
