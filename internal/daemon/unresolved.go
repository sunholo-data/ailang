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
	now   func() time.Time
}

func newUnresolved() *unresolved {
	return &unresolved{first: make(map[string]time.Time), now: time.Now}
}

// age records the first sighting of id and returns how long it has been
// unresolvable. A first sighting returns 0.
func (u *unresolved) age(id string) time.Duration {
	u.mu.Lock()
	defer u.mu.Unlock()
	now := u.now()
	if t, ok := u.first[id]; ok {
		return now.Sub(t)
	}
	u.first[id] = now
	return 0
}

// clear drops id, so a message that resolves after a lag (or is given up on)
// does not leak an entry.
func (u *unresolved) clear(id string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.first, id)
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
