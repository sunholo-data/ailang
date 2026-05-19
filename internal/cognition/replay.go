package cognition

import (
	"fmt"
	"io"
)

// ============================================================================
// Replay engine — M-COG-RUNTIME M3, v0.21.x
// ============================================================================
//
// The replay engine reconstructs an agent's observable history from a
// JSONL cognitive event log. Given the same log, the same set of
// registered handlers, and the same registration order, replay produces
// the same observer-side observations on every machine — that is the
// "deterministic distributed replay" property the umbrella design doc
// names as the strategic claim of the Cognitive OS.
//
// Replay flow:
//
//   1. Load JSONL log from an io.Reader (file, in-memory buffer, IPC)
//   2. Create a fresh Scheduler with optional capture log
//   3. Register handlers in canonical order (caller's responsibility)
//   4. Call Replay → events feed scheduler in (Clock, Sender) order
//   5. Compare scheduler.Dispatched() across runs to verify equivalence
//
// What this is NOT:
//   - Not a DOM reconstructor: that's browser-side (M3 browser scope,
//     deferred to a follow-up sprint). This file is the Go-side
//     orchestration of event dispatch in canonical order.
//   - Not concurrent: replay is sequential by design. Multi-agent
//     topologies use multiple Replayers, each consuming their slice.

// Replayer wraps the load → sort → dispatch flow into a single value-
// type that's easy to test and compose. One Replayer per replay run;
// reuse is not supported (call Run once).
type Replayer struct {
	scheduler *Scheduler
	source    io.Reader
}

// NewReplayer constructs a Replayer over the given event source and
// scheduler. The scheduler must already have its handlers registered
// before Run is called (registration order affects dispatch order).
func NewReplayer(scheduler *Scheduler, source io.Reader) *Replayer {
	return &Replayer{
		scheduler: scheduler,
		source:    source,
	}
}

// Run parses the JSONL source into an event slice, then dispatches in
// canonical order via the scheduler.
//
// Returns the number of events replayed and any error encountered.
// Forward-compat with future M-COG-MEMORY and M-COG-MESH event types:
// unknown kinds are silently skipped during load (ImportJSONL semantics).
func (r *Replayer) Run() (int, error) {
	if r.scheduler == nil {
		return 0, fmt.Errorf("replay: scheduler is nil")
	}
	if r.source == nil {
		return 0, fmt.Errorf("replay: source is nil")
	}

	// Load into a holding log; we don't append to the scheduler's
	// capture log here because RunFromLog dispatches each event, which
	// triggers the scheduler's own append. Keeping the loader separate
	// avoids double-counting.
	holding := NewEventLog(nil)
	count, err := holding.ImportJSONL(r.source)
	if err != nil {
		return count, fmt.Errorf("replay: load: %w", err)
	}

	if err := r.scheduler.RunFromLog(holding.Snapshot()); err != nil {
		return count, fmt.Errorf("replay: dispatch: %w", err)
	}
	return count, nil
}

// ============================================================================
// Equivalence check — verify two replay runs produce identical dispatch
// ============================================================================

// AreReplaysEquivalent checks that two dispatch slices represent the same
// observable history. Used by replay-determinism tests and (eventually)
// by M-COG-MESH cross-machine replay verification.
//
// Equivalence: same length AND for each index, same (Kind, Clock, Sender).
// Note: this does NOT check payload-level equality — replay is about
// canonical ordering, not byte-identical bodies. Body equivalence is a
// stronger property tested separately via JSONL round-trip.
func AreReplaysEquivalent(a, b []Event) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ba, bb := a[i].Base(), b[i].Base()
		if ba.EventKind != bb.EventKind {
			return false
		}
		if ba.Clock != bb.Clock {
			return false
		}
		if ba.Sender != bb.Sender {
			return false
		}
	}
	return true
}

// ReplayDivergence reports the index at which two dispatch slices first
// diverge, or -1 if equivalent. Useful for debugging non-deterministic
// replay scenarios.
func ReplayDivergence(a, b []Event) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		ba, bb := a[i].Base(), b[i].Base()
		if ba.EventKind != bb.EventKind || ba.Clock != bb.Clock || ba.Sender != bb.Sender {
			return i
		}
	}
	if len(a) != len(b) {
		return minLen
	}
	return -1
}
