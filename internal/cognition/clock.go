package cognition

import (
	"sync"
	"sync/atomic"
)

// ============================================================================
// Lamport clock — M-COG-RUNTIME M2, v0.21.x
// ============================================================================
//
// LamportClock is a non-decreasing logical timestamp following Lamport's
// "Time, Clocks, and the Ordering of Events" (1978). Used to order events
// in the cognitive event log so that:
//
//   - Local events: Tick() returns a strictly-increasing value
//   - Cross-node events: Update(remoteClock) advances local to max(local, remote)+1
//
// This is the M-COG-RUNTIME M2 ordering primitive. M-COG-MESH upgrades to
// vector clocks for true partial ordering across distributed nodes; Lamport
// is sufficient for single-tab and cross-tab BroadcastChannel scenarios
// where the scheduler serializes through one event loop.
//
// Tiebreaking: when two events share a Lamport value, ordering falls back
// to sender NodeID lex order. This happens in the M3 scheduler, not here.

// LamportValue is the underlying integer timestamp. int64 chosen for
// JSONL wire compatibility (JavaScript numbers are 53-bit safe integers,
// so the upper bound is 2^53; we won't approach that in any realistic
// agent lifetime).
type LamportValue int64

// Clock is a thread-safe Lamport clock.
//
// Tick is called on local event emission; Update is called when a message
// is received from another node. Both operations are atomic — multiple
// goroutines may share one Clock safely (the M3 scheduler is single-
// threaded but tests may exercise concurrency).
type Clock struct {
	value int64 // atomic access only
}

// NewClock creates a clock starting at 0. The first Tick returns 1.
func NewClock() *Clock {
	return &Clock{}
}

// NewClockAt creates a clock initialized to the given value. Useful for
// replay scenarios where the starting point is loaded from a JSONL log.
func NewClockAt(v LamportValue) *Clock {
	return &Clock{value: int64(v)}
}

// Tick increments the clock and returns the new value.
//
// Invariant: every Tick on a single Clock returns a strictly larger
// value than the previous Tick (or Update) on that Clock.
func (c *Clock) Tick() LamportValue {
	return LamportValue(atomic.AddInt64(&c.value, 1))
}

// Update advances the local clock to max(local, remote)+1.
//
// Called when a message is received with a remote clock value. Establishes
// the happens-before relation: any local event after Update has a clock
// strictly greater than the remote sender's clock at send-time.
//
// Returns the new local clock value (for symmetry with Tick).
func (c *Clock) Update(remote LamportValue) LamportValue {
	// Atomic compare-and-swap loop to ensure correctness under concurrent
	// Tick/Update calls. The "if remote > local then set to remote, then
	// always +1" semantics require us to either bump by 1 (if remote <= local)
	// or bump to remote+1 (otherwise) — done in one CAS.
	for {
		current := atomic.LoadInt64(&c.value)
		var next int64
		if int64(remote) > current {
			next = int64(remote) + 1
		} else {
			next = current + 1
		}
		if atomic.CompareAndSwapInt64(&c.value, current, next) {
			return LamportValue(next)
		}
		// CAS failed — another goroutine bumped concurrently; retry.
	}
}

// Read returns the current clock value without advancing it.
//
// Used by tests and replay machinery to peek at the clock without
// affecting causal ordering. Production code should call Tick or
// Update — calling Read followed by emitting an event with that value
// breaks the monotonicity invariant.
func (c *Clock) Read() LamportValue {
	return LamportValue(atomic.LoadInt64(&c.value))
}

// Reset sets the clock back to 0. Used in tests and replay.
//
// Calling Reset on a clock that's actively being used by other
// goroutines is undefined behavior — there's no race-safe way to
// "rewind" Lamport time, and any attempt would violate monotonicity
// from the perspective of in-flight events.
func (c *Clock) Reset() {
	atomic.StoreInt64(&c.value, 0)
}

// ============================================================================
// SenderTieBreaker — secondary ordering when Lamport values collide
// ============================================================================
//
// When two events share a Lamport value (legal: a Tick and an Update with
// remote=current both produce current+1), ordering falls back to sender
// NodeID. This is a stable lex comparison used by the M3 scheduler when
// it serializes events into the cognitive event log.
//
// Kept here next to the Clock primitive so the ordering rule is in one
// place — easier to audit determinism guarantees.

// CompareEvents returns negative / zero / positive following standard
// strcmp semantics for (Lamport, senderID) tuples.
//
// Determinism rule: identical (Lamport, senderID) means semantically
// equivalent in the partial order; the scheduler chooses based on
// event-content hash as a third tiebreaker (handled by the scheduler,
// not here).
func CompareEvents(aClock LamportValue, aSender string, bClock LamportValue, bSender string) int {
	if aClock < bClock {
		return -1
	}
	if aClock > bClock {
		return 1
	}
	if aSender < bSender {
		return -1
	}
	if aSender > bSender {
		return 1
	}
	return 0
}

// ============================================================================
// ClockRegistry — managed clock-per-node for multi-agent topologies
// ============================================================================
//
// In single-agent scenarios one Clock is enough; with multiple agents in
// the same runtime (the M-COG-MESH 4-agent demo) each agent needs its
// own clock to track its local view. ClockRegistry indexes clocks by
// NodeID for that case.
//
// Future M-COG-MESH extension: ClockRegistry becomes the snapshot
// surface for vector clocks (each entry tracks the most-recent Lamport
// value from each peer, not just local).

// ClockRegistry maps NodeID strings to Clock instances. Thread-safe.
type ClockRegistry struct {
	mu     sync.RWMutex
	clocks map[string]*Clock
}

// NewClockRegistry creates an empty registry.
func NewClockRegistry() *ClockRegistry {
	return &ClockRegistry{clocks: make(map[string]*Clock)}
}

// Get returns the clock for nodeID, creating it lazily if absent.
//
// Lazy creation is deliberate: in topologies where an agent learns of a
// peer only when the first message arrives, the receiving node may need
// to track the peer's clock without pre-registering it.
func (r *ClockRegistry) Get(nodeID string) *Clock {
	r.mu.RLock()
	c, ok := r.clocks[nodeID]
	r.mu.RUnlock()
	if ok {
		return c
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// Re-check after acquiring write lock (another goroutine may have created).
	if c, ok := r.clocks[nodeID]; ok {
		return c
	}
	c = NewClock()
	r.clocks[nodeID] = c
	return c
}

// Snapshot returns a copy of the current (nodeID → Lamport value) state.
//
// Useful for replay setup and for cognitive event log entries that need
// to record the full causal context. The map is a defensive copy — caller
// can mutate without affecting the registry.
func (r *ClockRegistry) Snapshot() map[string]LamportValue {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]LamportValue, len(r.clocks))
	for id, c := range r.clocks {
		out[id] = c.Read()
	}
	return out
}
