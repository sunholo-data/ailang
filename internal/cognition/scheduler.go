package cognition

import (
	"errors"
	"sort"
	"sync"
)

// ============================================================================
// Deterministic Scheduler — M-COG-RUNTIME M3, v0.21.x
// ============================================================================
//
// The scheduler is the canonical consumer of cognitive events. It:
//
//   1. Pulls events from the cognitive event log (or live transports)
//   2. Sorts them by (Lamport clock, sender NodeID) — total order
//   3. Dispatches to registered subscribers in canonical order
//   4. Records dispatch decisions (for replay determinism)
//
// Properties (locked by the umbrella design freeze):
//   - Single-threaded: one event loop, no concurrent dispatch
//   - Deterministic: identical event log → identical dispatch sequence
//   - Replayable: feed the same log to a fresh scheduler → same observers
//     see the same events in the same order
//
// This is the M3 substrate that the browser-side replay engine and the
// future M-COG-MESH distributed-replay machinery sit on top of.
//
// What this is NOT:
//   - Not an evaluator: it doesn't execute AILANG code; it just dispatches
//     events to registered subscribers (Go-side or, via M-COG-MESH wiring,
//     to AILANG closures through FnCaller — that wiring lands later)
//   - Not concurrent: parallel dispatch breaks total ordering. Use
//     multiple schedulers for multi-agent topologies (one per agent),
//     each consuming its own slice of the global event log.

// Handler is the callback type registered with the scheduler. Each
// dispatch invocation is synchronous — the scheduler waits for the
// Handler to return before dispatching the next event.
//
// Handlers should be fast and non-blocking; long-running work should
// be queued to a separate worker goroutine. Blocking the scheduler
// directly violates the single-threaded determinism contract.
type Handler func(e Event)

// Subscription identifies a registered handler. Used to unregister.
type Subscription struct {
	id   int
	kind string // event kind filter; "" matches all
}

// Scheduler is the single-threaded deterministic event-loop driver.
type Scheduler struct {
	mu         sync.Mutex
	subs       map[int]subscriberEntry
	nextSubID  int
	dispatched []Event // append-only record of dispatch order
	log        *EventLog
	stopped    bool
}

type subscriberEntry struct {
	kind    string // empty = match all kinds
	handler Handler
}

// NewScheduler constructs a fresh scheduler. The log argument is
// optional — non-nil means every dispatched event is also appended to
// the log (turn this on for replay capture, off for one-shot replay).
func NewScheduler(log *EventLog) *Scheduler {
	return &Scheduler{
		subs: make(map[int]subscriberEntry),
		log:  log,
	}
}

// Subscribe registers a Handler. If kind is "", the handler receives
// every event; otherwise only events with matching Kind() are dispatched.
//
// Returns a Subscription token used by Unsubscribe.
func (s *Scheduler) Subscribe(kind string, h Handler) Subscription {
	if h == nil {
		// Ignore nil — defensive; callers shouldn't do this but no
		// reason to panic the scheduler over it.
		return Subscription{id: -1}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextSubID++
	id := s.nextSubID
	s.subs[id] = subscriberEntry{kind: kind, handler: h}
	return Subscription{id: id, kind: kind}
}

// Unsubscribe removes a previously-registered Handler. Calling
// Unsubscribe on an unknown Subscription is a no-op.
func (s *Scheduler) Unsubscribe(sub Subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subs, sub.id)
}

// Dispatch delivers one event to all matching subscribers in
// registration order. Used for live single-event delivery (e.g. from
// transport callbacks bridged through the scheduler).
//
// Determinism note: registration order is stable within one scheduler;
// across schedulers, callers must register handlers in the same order
// to get the same dispatch sequence.
func (s *Scheduler) Dispatch(e Event) {
	if e == nil {
		return
	}
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	// Snapshot matching subscribers in stable id order.
	ids := make([]int, 0, len(s.subs))
	for id := range s.subs {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	matched := make([]Handler, 0, len(ids))
	for _, id := range ids {
		ent := s.subs[id]
		if ent.kind == "" || ent.kind == e.Kind() {
			matched = append(matched, ent.handler)
		}
	}
	s.dispatched = append(s.dispatched, e)
	if s.log != nil {
		_ = s.log.Append(e)
	}
	s.mu.Unlock()

	// Invoke outside the lock — handlers may call back into the
	// scheduler (e.g. Subscribe / Unsubscribe) without deadlocking.
	for _, h := range matched {
		h(e)
	}
}

// RunFromLog replays a slice of events through the scheduler in
// canonical Lamport+Sender order. This is the M3 replay-engine
// entry point.
//
// Returns ErrSchedulerStopped if Stop was called before completion.
// On success returns nil and the scheduler's dispatched slice
// contains every event in canonical order — callers can verify
// replay byte-equivalence by comparing dispatch slices across runs.
func (s *Scheduler) RunFromLog(events []Event) error {
	// Sort by canonical order (Lamport clock + sender NodeID).
	ordered := make([]Event, len(events))
	copy(ordered, events)
	sort.SliceStable(ordered, func(i, j int) bool {
		ei, ej := ordered[i].Base(), ordered[j].Base()
		return CompareEvents(ei.Clock, ei.Sender, ej.Clock, ej.Sender) < 0
	})

	for _, e := range ordered {
		s.mu.Lock()
		if s.stopped {
			s.mu.Unlock()
			return ErrSchedulerStopped
		}
		s.mu.Unlock()
		s.Dispatch(e)
	}
	return nil
}

// Dispatched returns a defensive copy of the dispatch record — the
// list of events the scheduler has handed to subscribers, in dispatch
// order. Used by replay tests to assert determinism (same input log →
// same dispatch slice).
func (s *Scheduler) Dispatched() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.dispatched))
	copy(out, s.dispatched)
	return out
}

// Stop halts the scheduler. Subsequent Dispatch / RunFromLog calls
// become no-ops returning ErrSchedulerStopped. Idempotent.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped = true
}

// ErrSchedulerStopped is returned by RunFromLog when Stop was called
// during a replay.
var ErrSchedulerStopped = errors.New("scheduler: stopped")
