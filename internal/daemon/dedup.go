package daemon

import (
	"sync"
	"time"
)

// dedup is a fixed-window suppression map. Callers compute a key (e.g.
// "task:abc123:completed" or "msg:m_xyz") and call seen(key); the first call
// in a window returns false (caller should fire), subsequent calls within the
// window return true (caller should suppress).
//
// sweep() drops expired entries; intended to be called periodically by the
// daemon loop (the map would otherwise grow with traffic).
type dedup struct {
	mu     sync.Mutex
	window time.Duration
	hits   map[string]time.Time
	now    func() time.Time
}

func newDedup(window time.Duration) *dedup {
	return &dedup{
		window: window,
		hits:   make(map[string]time.Time),
		now:    time.Now,
	}
}

// seen reports whether key was hit within the dedup window. It records the
// hit before returning, so a fresh key returns false on the first call and
// true on subsequent calls until the window expires.
func (d *dedup) seen(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := d.now()
	if last, ok := d.hits[key]; ok && now.Sub(last) < d.window {
		return true
	}
	d.hits[key] = now
	return false
}

// sweep removes entries older than the dedup window. Call periodically.
func (d *dedup) sweep() {
	d.mu.Lock()
	defer d.mu.Unlock()
	cutoff := d.now().Add(-d.window)
	for k, t := range d.hits {
		if t.Before(cutoff) {
			delete(d.hits, k)
		}
	}
}

func (d *dedup) size() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.hits)
}
