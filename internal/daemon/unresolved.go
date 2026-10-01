package daemon

import (
	"sync"
	"time"
)

// absentGrace is how long a notification whose message cannot be resolved keeps
// being retried before the daemon acks it and says so.
//
// The window exists for one real case: Pub/Sub can deliver the notification
// before Firestore has replicated the document it names, so a fresh absence is
// worth a retry. That lag is seconds. Anything still absent minutes later was
// never written — the publisher's store was a temp dir, a different project, or
// the doc was deleted — and no number of retries will conjure it.
const absentGrace = 5 * time.Minute

// unresolved records when each message id was FIRST seen unresolvable, so the
// daemon can retry a replication lag but give up on a message that does not
// exist. Without it every absence is retried forever: a permanently-missing doc
// is redelivered as fast as Pub/Sub can cycle it, and because a subscription
// delivers its backlog oldest-first, the poison sits in front of the real
// traffic and starves it.
//
// The clock resets on daemon restart, which is deliberate: a restart is also how
// a genuinely transient outage clears, and the subscription's dead-letter policy
// is the backstop for anything that outlives the process.
type unresolved struct {
	mu    sync.Mutex
	first map[string]time.Time
	// dropped is the verdict, kept separately from first so that a message this
	// process has ALREADY given up on is acked on sight instead of earning a
	// fresh grace window.
	//
	// It has to be sticky because an ack does not reliably retire the message.
	// The subscription has a 30s ack deadline and no exactly-once delivery, so a
	// message nacked on every cycle has several deliveries in flight at once;
	// acking one of them while another is still outstanding puts it straight
	// back. Without the verdict the daemon re-times the same 5 minutes on every
	// resurrection, which is a 5-minute storm per restart.
	dropped map[string]struct{}
	now     func() time.Time
}

func newUnresolved() *unresolved {
	return &unresolved{
		first:   make(map[string]time.Time),
		dropped: make(map[string]struct{}),
		now:     time.Now,
	}
}

// age records the first sighting of id and returns how long it has been
// unresolvable, plus whether this call WAS that first sighting.
//
// The two are returned together on purpose. Asking separately meant comparing a
// stored timestamp against a second time.Now(), which differs by nanoseconds, so
// "is this the first sighting?" was always false against a real clock and only
// held against a frozen test clock. The caller uses the flag to log the retry
// once instead of on every redelivery — a poison message cycles several times a
// second and would bury the traffic that matters.
func (u *unresolved) age(id string) (time.Duration, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	now := u.now()
	if t, ok := u.first[id]; ok {
		return now.Sub(t), false
	}
	u.first[id] = now
	return 0, true
}

// giveUp records the verdict that id will never resolve.
func (u *unresolved) giveUp(id string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.first, id)
	u.dropped[id] = struct{}{}
}

// gaveUp reports whether this process has already decided id is unresolvable.
func (u *unresolved) gaveUp(id string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	_, ok := u.dropped[id]
	return ok
}

// clear drops id, so a message that resolves after a lag does not leak an entry
// — and forgets any verdict, because a message that resolved is not poison.
func (u *unresolved) clear(id string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.first, id)
	delete(u.dropped, id)
}

// sweep drops ids not seen for well past the grace window, so the map cannot
// grow without bound on a long-lived daemon.
func (u *unresolved) sweep() {
	u.mu.Lock()
	defer u.mu.Unlock()
	cutoff := u.now().Add(-10 * absentGrace)
	for id, t := range u.first {
		if t.Before(cutoff) {
			delete(u.first, id)
		}
	}
}

func (u *unresolved) size() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.first)
}
