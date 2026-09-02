package coordinator

import "log"

// M-COORDINATOR-EXECUTION-TRUST M4 — a dispatched message must be marked read.
//
// `ailang messages health` reports "routable, but never dispatched", and says
// that number should always be zero because push delivers everything. It never
// could be. In cloud mode the daemon called MarkAsRead on the Pub/Sub adapter,
// whose implementation is `return nil` — correct for the WIRE (the Pub/Sub
// message really is acked on receipt) but it never touches the Firestore row, so
// the message stays unread forever after its task has been created and
// dispatched.
//
// Measured 2026-09-02: health flagged 7 messages as routable-and-never-
// dispatched; every one of them had in fact been dispatched. A number that
// cannot reach zero stops being read, and this one is the plane's only
// self-report.

// inboxReadMarker is the one method this needs. Narrow on purpose: the full
// MessageStore is ~40 methods, and depending on all of them to set one flag
// would make the behaviour untestable without a large fake.
type inboxReadMarker interface {
	MarkInboxMessageRead(id string) error
}

// markDispatchedMessageRead records that a message has become a task.
//
// Best-effort by design: the task is already created and dispatched by this
// point, so failing the dispatch because the bookkeeping write failed would
// trade real work for a status flag. It is logged loudly instead — a persistent
// failure here shows up as health never reaching zero, which is exactly the
// symptom this fixes and therefore the right place to notice it.
func markDispatchedMessageRead(store inboxReadMarker, messageID string, logger *log.Logger) bool {
	if store == nil || messageID == "" {
		return false
	}
	if err := store.MarkInboxMessageRead(messageID); err != nil {
		if logger != nil {
			logger.Printf("Warning: task dispatched but message %s could not be marked read: %v "+
				"(messages health will keep counting it as undispatched)", messageID, err)
		}
		return false
	}
	return true
}
