package cognition

import (
	"bytes"
	"strings"
	"testing"
)

// ============================================================================
// Replayer.Run — JSONL → canonical dispatch
// ============================================================================

func TestReplayer_Run_DispatchesInCanonicalOrder(t *testing.T) {
	// Build a log in shuffled order
	src := NewEventLog(nil)
	_ = src.Append(MessageSentEvent{EventBase: NewEventBase("MessageSent", "node_b", 3), MsgID: "m3"})
	_ = src.Append(PatchAppliedEvent{EventBase: NewEventBase("PatchApplied", "node_a", 1), Region: "r1", PatchType: "AddPanel"})
	_ = src.Append(MessageSentEvent{EventBase: NewEventBase("MessageSent", "node_a", 2), MsgID: "m2"})

	var buf bytes.Buffer
	if err := src.ExportJSONL(&buf); err != nil {
		t.Fatal(err)
	}

	// Replay through a fresh scheduler
	s := NewScheduler(nil)
	seen := []LamportValue{}
	s.Subscribe("", func(e Event) { seen = append(seen, e.Base().Clock) })

	r := NewReplayer(s, &buf)
	n, err := r.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 events imported, got %d", n)
	}

	// Verify canonical order: clocks 1, 2, 3
	if len(seen) != 3 {
		t.Fatalf("expected 3 dispatches, got %d", len(seen))
	}
	for i, want := range []LamportValue{1, 2, 3} {
		if seen[i] != want {
			t.Errorf("dispatch[%d]: got clock %d, want %d", i, seen[i], want)
		}
	}
}

func TestReplayer_Run_NilScheduler_Errors(t *testing.T) {
	r := NewReplayer(nil, strings.NewReader(""))
	_, err := r.Run()
	if err == nil {
		t.Fatal("expected error for nil scheduler, got nil")
	}
}

func TestReplayer_Run_NilSource_Errors(t *testing.T) {
	r := NewReplayer(NewScheduler(nil), nil)
	_, err := r.Run()
	if err == nil {
		t.Fatal("expected error for nil source, got nil")
	}
}

func TestReplayer_Run_SkipsUnknownKinds(t *testing.T) {
	jsonl := `{"kind":"MessageSent","clock":1,"sender":"a","ts_ms":0,"to":"b","msg_id":"m1"}
{"kind":"FutureMCogMeshEvent","clock":2,"sender":"a","ts_ms":0}
{"kind":"PatchApplied","clock":3,"sender":"a","ts_ms":0,"region":"r1","patch_type":"AddPanel"}
`
	s := NewScheduler(nil)
	var count int
	s.Subscribe("", func(e Event) { count++ })

	r := NewReplayer(s, strings.NewReader(jsonl))
	n, err := r.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 known events imported, got %d", n)
	}
	if count != 2 {
		t.Errorf("expected 2 events dispatched (unknown skipped), got %d", count)
	}
}

// ============================================================================
// Determinism — repeated replays produce identical dispatch slices
// ============================================================================

func TestReplayer_RepeatedRuns_ProduceIdenticalDispatch(t *testing.T) {
	// Build a fixed log
	src := NewEventLog(nil)
	for i := 1; i <= 5; i++ {
		_ = src.Append(MessageSentEvent{
			EventBase: EventBase{
				EventKind:   "MessageSent",
				Clock:       LamportValue(i),
				Sender:      "node_a",
				TimestampMs: 1000, // fixed for byte-determinism
			},
			MsgID: "m1",
		})
	}
	var buf bytes.Buffer
	if err := src.ExportJSONL(&buf); err != nil {
		t.Fatal(err)
	}
	body := buf.Bytes()

	replay := func() []Event {
		s := NewScheduler(nil)
		s.Subscribe("", func(e Event) {})
		_, _ = NewReplayer(s, bytes.NewReader(body)).Run()
		return s.Dispatched()
	}

	d1 := replay()
	d2 := replay()
	if !AreReplaysEquivalent(d1, d2) {
		idx := ReplayDivergence(d1, d2)
		t.Errorf("replays diverge at index %d", idx)
	}
}

// ============================================================================
// AreReplaysEquivalent + ReplayDivergence
// ============================================================================

func TestAreReplaysEquivalent_Identical(t *testing.T) {
	events := []Event{
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)},
		PatchAppliedEvent{EventBase: NewEventBase("PatchApplied", "a", 2)},
	}
	if !AreReplaysEquivalent(events, events) {
		t.Error("identical slices should be equivalent")
	}
}

func TestAreReplaysEquivalent_DifferentLength(t *testing.T) {
	a := []Event{MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)}}
	b := []Event{}
	if AreReplaysEquivalent(a, b) {
		t.Error("different-length slices should not be equivalent")
	}
}

func TestAreReplaysEquivalent_DifferentKind(t *testing.T) {
	a := []Event{MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)}}
	b := []Event{PatchAppliedEvent{EventBase: NewEventBase("PatchApplied", "a", 1)}}
	if AreReplaysEquivalent(a, b) {
		t.Error("different kinds should not be equivalent")
	}
}

func TestAreReplaysEquivalent_DifferentClock(t *testing.T) {
	a := []Event{MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)}}
	b := []Event{MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 2)}}
	if AreReplaysEquivalent(a, b) {
		t.Error("different clocks should not be equivalent")
	}
}

func TestReplayDivergence_Identical_ReturnsNegOne(t *testing.T) {
	events := []Event{
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 2)},
	}
	if got := ReplayDivergence(events, events); got != -1 {
		t.Errorf("identical: got %d, want -1", got)
	}
}

func TestReplayDivergence_DivergeAtIndex(t *testing.T) {
	a := []Event{
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 2)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 3)},
	}
	b := []Event{
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 99)}, // diverges here
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 3)},
	}
	if got := ReplayDivergence(a, b); got != 1 {
		t.Errorf("divergence index: got %d, want 1", got)
	}
}

func TestReplayDivergence_LengthMismatch_ReturnsMinLen(t *testing.T) {
	a := []Event{
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 2)},
	}
	b := []Event{
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)},
	}
	if got := ReplayDivergence(a, b); got != 1 {
		t.Errorf("length-mismatch divergence: got %d, want 1", got)
	}
}

// ============================================================================
// End-to-end: capture in one scheduler, replay in another, verify
// equivalence. This is the canonical "deterministic distributed replay"
// property in miniature.
// ============================================================================

func TestReplay_RoundTrip_CaptureAndReplay(t *testing.T) {
	// Phase 1: capture
	captureLog := NewEventLog(nil)
	captureSched := NewScheduler(captureLog)
	captureSched.Subscribe("", func(e Event) {})

	// Simulate live activity
	captureSched.Dispatch(MessageSentEvent{
		EventBase: EventBase{EventKind: "MessageSent", Clock: 1, Sender: "a", TimestampMs: 1000},
		MsgID:     "m1", To: "b",
	})
	captureSched.Dispatch(PatchAppliedEvent{
		EventBase: EventBase{EventKind: "PatchApplied", Clock: 2, Sender: "a", TimestampMs: 1001},
		Region:    "r1", PatchType: "AddPanel",
	})
	captureSched.Dispatch(MessageSentEvent{
		EventBase: EventBase{EventKind: "MessageSent", Clock: 3, Sender: "b", TimestampMs: 1002},
		MsgID:     "m2", To: "a",
	})

	captured := captureSched.Dispatched()

	// Phase 2: serialize the capture log to JSONL
	var buf bytes.Buffer
	if err := captureLog.ExportJSONL(&buf); err != nil {
		t.Fatal(err)
	}

	// Phase 3: replay through a fresh scheduler
	replaySched := NewScheduler(nil)
	replaySched.Subscribe("", func(e Event) {})
	_, err := NewReplayer(replaySched, &buf).Run()
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	replayed := replaySched.Dispatched()

	if !AreReplaysEquivalent(captured, replayed) {
		idx := ReplayDivergence(captured, replayed)
		t.Errorf("capture/replay diverge at index %d", idx)
	}
}
