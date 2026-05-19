package cognition

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// ============================================================================
// Cognitive Event Log — M-COG-RUNTIME M2, v0.21.x
// ============================================================================
//
// The cognitive event log is the canonical, replayable record of agent
// cognition: every effect, message, DOM patch, capability event, and
// verification fact gets logged in clock-ordered append-only form.
//
// Design properties (locked in the umbrella design freeze):
//   - Append-only: events are never modified after emission
//   - Lamport-ordered: clock + sender NodeID gives a total order
//   - JSONL on disk: human-greppable, replayable via the M3 replay engine
//   - In-memory backing for M2; IndexedDB persistence is M-COG-MEMORY scope
//
// New event kinds must be added to:
//   1. The Event interface implementation (this file)
//   2. encodeEventToJSON / decodeEventFromJSON
//   3. The M3 scheduler's dispatch table

// ============================================================================
// Event ADT — closed sum type, discriminated by Kind field on the wire
// ============================================================================

// Event is the sum type for all cognitive events. Concrete variants embed
// EventBase to carry the common metadata (kind, clock, sender, timestamp).
//
// Pattern: marker method isEvent() prevents accidental implementations
// from outside this package — matches the DOMPatch / DOMEvent shape in
// internal/effects/dom.go.
type Event interface {
	isEvent()
	Kind() string
	Base() EventBase
}

// EventBase carries fields common to every event variant. Embedded into
// each concrete variant. Serialized at the top level of the JSONL line —
// the discriminator (kind) is part of EventBase, not inside a payload.
type EventBase struct {
	EventKind     string       `json:"kind"`
	Clock         LamportValue `json:"clock"`
	Sender        string       `json:"sender,omitempty"`
	TimestampMs   int64        `json:"ts_ms"`
	CorrelationID string       `json:"correlation,omitempty"`
}

// Kind returns the discriminator string for the variant.
func (b EventBase) Kind() string { return b.EventKind }

// Base returns the EventBase — used by the log to access common fields
// without type-switching on every variant.
func (b EventBase) Base() EventBase { return b }

// ============================================================================
// Concrete event variants — the M2 baseline set
// ============================================================================
//
// M-COG-MEMORY (M2 sibling) adds: MemoryWriteEvent, MemoryReadEvent, etc.
// M-COG-MESH adds: TransportErrorEvent, NodeJoinedEvent, NodeLeftEvent.
// Forward-compat: the EventKind discriminator is the extension point —
// old readers skip unknown kinds (handled in decodeEventFromJSON).

// MessageSentEvent records a Msg.send call.
type MessageSentEvent struct {
	EventBase
	To          string `json:"to"`
	MsgID       string `json:"msg_id"`
	PayloadHash string `json:"payload_hash,omitempty"`
}

func (MessageSentEvent) isEvent() {}

// MessageReceivedEvent records a Msg.recv call.
type MessageReceivedEvent struct {
	EventBase
	From  string `json:"from"`
	MsgID string `json:"msg_id"`
}

func (MessageReceivedEvent) isEvent() {}

// PatchAppliedEvent records a DOM.applyPatch / DOM.applyBatch call.
//
// PatchType is "AddPanel" | "UpdateNode" | "RemoveNode" | "AddTimeline"
// — matches the DOMPatch ADT in internal/effects/dom.go. NodeID is the
// handler-assigned identifier (content-hashed in browser host for replay
// determinism).
type PatchAppliedEvent struct {
	EventBase
	Region    string `json:"region"`
	PatchType string `json:"patch_type"`
	NodeID    string `json:"node_id,omitempty"`
}

func (PatchAppliedEvent) isEvent() {}

// CapabilityExceededEvent records a budget exhaustion. M2's enforcement
// hook on the budget pathway emits this when an op trips a budget.
type CapabilityExceededEvent struct {
	EventBase
	Effect string `json:"effect"`
	Op     string `json:"op"`
	Budget int    `json:"budget"`
}

func (CapabilityExceededEvent) isEvent() {}

// TraceCapturedEvent is emitted by std/trace's Trace effect when a span
// completes. Bridges the existing trace.Collector spans into the
// cognitive log for unified replay.
type TraceCapturedEvent struct {
	EventBase
	SpanName   string `json:"span_name"`
	DurationNs int64  `json:"duration_ns,omitempty"`
}

func (TraceCapturedEvent) isEvent() {}

// ============================================================================
// EventLog — in-memory append-only store
// ============================================================================
//
// EventLog buffers events in memory and supports clock-ordered enumeration
// + JSONL export. Persistence to IndexedDB (browser) or file (native) is
// pluggable via Sink — the M2 default is an in-memory ring; M-COG-MEMORY
// adds the IndexedDB sink.
//
// Thread-safety: Append is safe under concurrent callers. Snapshot returns
// a defensive copy. Range (replay-style iteration) takes a read lock and
// must not call back into Append (would deadlock).

// EventLog is the in-memory event store.
type EventLog struct {
	mu     sync.RWMutex
	events []Event
	sink   Sink // optional flush target (file, IndexedDB, etc.)
}

// Sink is the optional persistence interface. Emit is called synchronously
// from EventLog.Append; impls should buffer or batch internally if they
// want non-blocking behaviour. A nil Sink means in-memory only.
type Sink interface {
	Emit(e Event) error
}

// NewEventLog creates an empty log. Pass nil for sink to skip persistence
// (useful in tests and for the M2 default).
func NewEventLog(sink Sink) *EventLog {
	return &EventLog{
		events: make([]Event, 0, 64),
		sink:   sink,
	}
}

// Append adds an event to the log. The event's EventBase fields (clock,
// timestamp) must be set by the caller — Append doesn't auto-stamp them
// because the M3 scheduler is the canonical source of clock values and
// scope-correct timestamps.
//
// If a Sink is configured and its Emit returns an error, Append returns
// that error WITHOUT appending — preserves the invariant that what's in
// the in-memory log is also in the sink. Callers handle sink failures
// (e.g. retry, fail-fast, etc.).
func (l *EventLog) Append(e Event) error {
	if e == nil {
		return fmt.Errorf("event_log: cannot append nil event")
	}
	if l.sink != nil {
		if err := l.sink.Emit(e); err != nil {
			return fmt.Errorf("event_log: sink rejected event: %w", err)
		}
	}
	l.mu.Lock()
	l.events = append(l.events, e)
	l.mu.Unlock()
	return nil
}

// Len returns the number of events currently in the log.
func (l *EventLog) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.events)
}

// Snapshot returns a defensive copy of all events in append order.
//
// "Append order" is not necessarily "Lamport order" — the M3 scheduler
// is responsible for inserting events in clock order. M2 only guarantees
// that Append calls are recorded in call order; replay code should sort
// by (Clock, Sender) before applying.
func (l *EventLog) Snapshot() []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Event, len(l.events))
	copy(out, l.events)
	return out
}

// Range iterates events in append order, calling fn for each. Iteration
// stops early if fn returns false.
//
// Locking: holds the read lock for the duration of iteration. fn must
// NOT call back into Append on the same log (would deadlock). For
// concurrent producers + a replay consumer, use Snapshot instead.
func (l *EventLog) Range(fn func(Event) bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, e := range l.events {
		if !fn(e) {
			return
		}
	}
}

// ============================================================================
// JSONL encoding — wire format for export, replay, and inter-process IPC
// ============================================================================

// ExportJSONL writes the log to w, one JSON object per line. Events
// are emitted in append order — caller is responsible for sorting if
// replay-determinism order is required.
//
// Error semantics: stops at first write error, returns it. Partial writes
// are possible — caller may need to truncate or retry.
func (l *EventLog) ExportJSONL(w io.Writer) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	l.mu.RLock()
	defer l.mu.RUnlock()

	for i, e := range l.events {
		line, err := encodeEventToJSON(e)
		if err != nil {
			return fmt.Errorf("event_log: encode event %d: %w", i, err)
		}
		if _, err := bw.Write(line); err != nil {
			return fmt.Errorf("event_log: write event %d: %w", i, err)
		}
		if err := bw.WriteByte('\n'); err != nil {
			return fmt.Errorf("event_log: write newline %d: %w", i, err)
		}
	}
	return nil
}

// ImportJSONL reads JSONL from r and appends each parsed event to the
// log. Unknown event kinds are skipped (forward-compat with future
// M-COG-MEMORY and M-COG-MESH event types) rather than producing errors.
//
// Returns the number of events successfully appended and any parse/IO
// error encountered. Partial imports are normal — caller may inspect
// the returned count.
func (l *EventLog) ImportJSONL(r io.Reader) (int, error) {
	scanner := bufio.NewScanner(r)
	// Large buffer for events that may carry trace payloads or hashes
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	count := 0
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		e, err := decodeEventFromJSON(line)
		if err != nil {
			return count, fmt.Errorf("event_log: line %d: %w", lineNum, err)
		}
		if e == nil {
			// Unknown kind — skip silently (forward-compat).
			continue
		}
		if err := l.Append(e); err != nil {
			return count, fmt.Errorf("event_log: line %d: append: %w", lineNum, err)
		}
		count++
	}
	return count, scanner.Err()
}

// encodeEventToJSON serializes an Event to JSON bytes.
//
// Implementation: each variant marshals itself via json.Marshal — the
// EventBase fields are embedded so the discriminator naturally appears
// at the top level alongside variant-specific fields.
func encodeEventToJSON(e Event) ([]byte, error) {
	return json.Marshal(e)
}

// decodeEventFromJSON parses one JSONL line into a concrete event variant.
//
// Two-pass decode:
//  1. Parse only the "kind" field to find the discriminator.
//  2. Re-parse the full line into the matching variant struct.
//
// Returns (nil, nil) for unknown kinds — forward-compatibility. Returns
// (nil, err) on actual parse failures (malformed JSON, missing kind).
func decodeEventFromJSON(line []byte) (Event, error) {
	// First pass: extract kind
	var hdr struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(line, &hdr); err != nil {
		return nil, fmt.Errorf("malformed JSON: %w", err)
	}
	if hdr.Kind == "" {
		return nil, fmt.Errorf("missing 'kind' field")
	}

	// Second pass: dispatch on kind
	switch hdr.Kind {
	case "MessageSent":
		var e MessageSentEvent
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("decode MessageSent: %w", err)
		}
		return e, nil
	case "MessageReceived":
		var e MessageReceivedEvent
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("decode MessageReceived: %w", err)
		}
		return e, nil
	case "PatchApplied":
		var e PatchAppliedEvent
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("decode PatchApplied: %w", err)
		}
		return e, nil
	case "CapabilityExceeded":
		var e CapabilityExceededEvent
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("decode CapabilityExceeded: %w", err)
		}
		return e, nil
	case "TraceCaptured":
		var e TraceCapturedEvent
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("decode TraceCaptured: %w", err)
		}
		return e, nil
	default:
		// Forward-compat: silently skip unknown kinds. M-COG-MEMORY and
		// M-COG-MESH extend this dispatch table.
		return nil, nil
	}
}

// ============================================================================
// Helpers — sender-stamp + clock advance for callers
// ============================================================================

// NewEventBase constructs an EventBase with current timestamp and the
// provided clock + sender. Centralizes the timestamp policy (Unix
// millis, matching the existing trace and messaging conventions).
//
// The kind argument is the variant discriminator; callers building
// concrete events embed the returned EventBase.
func NewEventBase(kind, sender string, clock LamportValue) EventBase {
	return EventBase{
		EventKind:   kind,
		Clock:       clock,
		Sender:      sender,
		TimestampMs: time.Now().UnixMilli(),
	}
}
