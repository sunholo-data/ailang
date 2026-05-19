package cognition

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// ============================================================================
// EventBase + variants — basic shape
// ============================================================================

func TestEventBase_Kind(t *testing.T) {
	e := MessageSentEvent{
		EventBase: NewEventBase("MessageSent", "node_a", 42),
		To:        "inbox_b",
		MsgID:     "m1",
	}
	if e.Kind() != "MessageSent" {
		t.Errorf("Kind: got %q, want MessageSent", e.Kind())
	}
	if e.Base().Clock != 42 {
		t.Errorf("Base().Clock: got %d, want 42", e.Base().Clock)
	}
	if e.Base().Sender != "node_a" {
		t.Errorf("Base().Sender: got %q, want node_a", e.Base().Sender)
	}
	if e.Base().TimestampMs == 0 {
		t.Errorf("TimestampMs should be auto-stamped, got 0")
	}
}

// ============================================================================
// EventLog — Append / Len / Snapshot / Range
// ============================================================================

func TestEventLog_AppendAndLen(t *testing.T) {
	l := NewEventLog(nil)
	if l.Len() != 0 {
		t.Errorf("fresh log Len: got %d, want 0", l.Len())
	}

	if err := l.Append(MessageSentEvent{
		EventBase: NewEventBase("MessageSent", "a", 1),
		To:        "b", MsgID: "m1",
	}); err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if l.Len() != 1 {
		t.Errorf("after 1 append: got Len=%d, want 1", l.Len())
	}
}

func TestEventLog_Append_RejectsNil(t *testing.T) {
	l := NewEventLog(nil)
	err := l.Append(nil)
	if err == nil {
		t.Fatal("Append(nil) should error, got nil")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error message should mention nil, got: %v", err)
	}
}

func TestEventLog_Snapshot_DefensiveCopy(t *testing.T) {
	l := NewEventLog(nil)
	_ = l.Append(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1), MsgID: "m1"})

	snap := l.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("snap len: got %d, want 1", len(snap))
	}

	// Mutating the snapshot slice shouldn't affect log
	snap[0] = nil
	again := l.Snapshot()
	if again[0] == nil {
		t.Error("Snapshot should be defensive — mutation leaked back into log")
	}
}

func TestEventLog_Range_AppendOrder(t *testing.T) {
	l := NewEventLog(nil)
	for i := 1; i <= 5; i++ {
		_ = l.Append(MessageSentEvent{
			EventBase: NewEventBase("MessageSent", "a", LamportValue(i)),
			MsgID:     fmt.Sprintf("m%d", i),
		})
	}

	seen := []LamportValue{}
	l.Range(func(e Event) bool {
		seen = append(seen, e.Base().Clock)
		return true
	})
	for i, c := range seen {
		want := LamportValue(i + 1)
		if c != want {
			t.Errorf("Range[%d].Clock: got %d, want %d", i, c, want)
		}
	}
}

func TestEventLog_Range_EarlyStop(t *testing.T) {
	l := NewEventLog(nil)
	for i := 1; i <= 5; i++ {
		_ = l.Append(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", LamportValue(i))})
	}
	visited := 0
	l.Range(func(e Event) bool {
		visited++
		return visited < 3 // stop after 3
	})
	if visited != 3 {
		t.Errorf("Range should stop early at fn=false, visited %d (want 3)", visited)
	}
}

// ============================================================================
// JSONL export + import — round-trip
// ============================================================================

func TestEventLog_JSONL_RoundTrip_AllVariants(t *testing.T) {
	l1 := NewEventLog(nil)
	original := []Event{
		MessageSentEvent{
			EventBase: NewEventBase("MessageSent", "node_a", 1),
			To:        "inbox_b", MsgID: "m1", PayloadHash: "sha256:abc",
		},
		MessageReceivedEvent{
			EventBase: NewEventBase("MessageReceived", "node_b", 2),
			From:      "node_a", MsgID: "m1",
		},
		PatchAppliedEvent{
			EventBase: NewEventBase("PatchApplied", "node_a", 3),
			Region:    "agent_a_dom", PatchType: "AddPanel", NodeID: "n42",
		},
		CapabilityExceededEvent{
			EventBase: NewEventBase("CapabilityExceeded", "node_a", 4),
			Effect:    "DOM", Op: "applyPatch", Budget: 20,
		},
		TraceCapturedEvent{
			EventBase: NewEventBase("TraceCaptured", "node_a", 5),
			SpanName:  "renderHeatmap", DurationNs: 1234,
		},
	}
	for _, e := range original {
		if err := l1.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	// Export
	var buf bytes.Buffer
	if err := l1.ExportJSONL(&buf); err != nil {
		t.Fatalf("ExportJSONL: %v", err)
	}

	// Validate: each line is a JSON object with a 'kind' field
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != len(original) {
		t.Fatalf("exported lines: got %d, want %d", len(lines), len(original))
	}
	for i, line := range lines {
		if !strings.Contains(line, `"kind"`) {
			t.Errorf("line %d missing 'kind' field: %s", i, line)
		}
	}

	// Import into a fresh log
	l2 := NewEventLog(nil)
	n, err := l2.ImportJSONL(&buf)
	if err != nil {
		t.Fatalf("ImportJSONL: %v", err)
	}
	if n != len(original) {
		t.Errorf("ImportJSONL count: got %d, want %d", n, len(original))
	}
	if l2.Len() != len(original) {
		t.Errorf("imported log Len: got %d, want %d", l2.Len(), len(original))
	}

	// Verify variants survived round-trip
	imported := l2.Snapshot()
	if _, ok := imported[0].(MessageSentEvent); !ok {
		t.Errorf("imported[0]: got %T, want MessageSentEvent", imported[0])
	}
	if _, ok := imported[2].(PatchAppliedEvent); !ok {
		t.Errorf("imported[2]: got %T, want PatchAppliedEvent", imported[2])
	}
	if cap := imported[3].(CapabilityExceededEvent); cap.Effect != "DOM" {
		t.Errorf("CapabilityExceeded.Effect lost: got %q", cap.Effect)
	}
}

func TestEventLog_ImportJSONL_SkipsUnknownKinds(t *testing.T) {
	jsonl := `{"kind":"MessageSent","clock":1,"sender":"a","ts_ms":0,"to":"b","msg_id":"m1"}
{"kind":"FutureEventFromMCogMesh","clock":2,"sender":"a","ts_ms":0,"weird_field":"value"}
{"kind":"MessageReceived","clock":3,"sender":"b","ts_ms":0,"from":"a","msg_id":"m1"}
`
	l := NewEventLog(nil)
	n, err := l.ImportJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("ImportJSONL: %v", err)
	}
	// Should have imported 2 (MessageSent + MessageReceived), skipped 1 unknown
	if n != 2 {
		t.Errorf("expected 2 known events imported, got %d", n)
	}
}

func TestEventLog_ImportJSONL_RejectsMalformed(t *testing.T) {
	jsonl := `{"kind":"MessageSent","clock":1,"sender":"a","ts_ms":0,"to":"b","msg_id":"m1"}
not valid json
`
	l := NewEventLog(nil)
	n, err := l.ImportJSONL(strings.NewReader(jsonl))
	if err == nil {
		t.Fatal("expected error on malformed line, got nil")
	}
	// Partial import is OK: first event should have been appended
	if n != 1 {
		t.Errorf("expected partial import count=1, got %d", n)
	}
}

func TestEventLog_ImportJSONL_RejectsMissingKind(t *testing.T) {
	jsonl := `{"clock":1,"sender":"a","to":"b"}
`
	l := NewEventLog(nil)
	_, err := l.ImportJSONL(strings.NewReader(jsonl))
	if err == nil {
		t.Fatal("expected error on missing 'kind' field, got nil")
	}
}

func TestEventLog_ImportJSONL_HandlesBlankLines(t *testing.T) {
	jsonl := `{"kind":"MessageSent","clock":1,"sender":"a","ts_ms":0,"to":"b","msg_id":"m1"}

{"kind":"MessageReceived","clock":2,"sender":"b","ts_ms":0,"from":"a","msg_id":"m1"}
`
	l := NewEventLog(nil)
	n, err := l.ImportJSONL(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("ImportJSONL: %v", err)
	}
	if n != 2 {
		t.Errorf("blank-line skip: got %d events, want 2", n)
	}
}

// ============================================================================
// Sink — pluggable persistence
// ============================================================================

type captureSink struct {
	mu       sync.Mutex
	captured []Event
	failNext bool
}

func (s *captureSink) Emit(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failNext {
		s.failNext = false
		return fmt.Errorf("sink: simulated failure")
	}
	s.captured = append(s.captured, e)
	return nil
}

func TestEventLog_Sink_EmitsOnAppend(t *testing.T) {
	sink := &captureSink{}
	l := NewEventLog(sink)
	_ = l.Append(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)})
	_ = l.Append(MessageReceivedEvent{EventBase: NewEventBase("MessageReceived", "a", 2)})
	if len(sink.captured) != 2 {
		t.Errorf("sink should have received 2 events, got %d", len(sink.captured))
	}
}

func TestEventLog_Sink_FailureBlocksAppend(t *testing.T) {
	sink := &captureSink{failNext: true}
	l := NewEventLog(sink)
	err := l.Append(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)})
	if err == nil {
		t.Fatal("expected error from sink failure, got nil")
	}
	if l.Len() != 0 {
		t.Errorf("log should not have stored event when sink failed, Len=%d", l.Len())
	}
	if len(sink.captured) != 0 {
		t.Errorf("sink should not have stored event on failure, len=%d", len(sink.captured))
	}
}

// ============================================================================
// Thread safety
// ============================================================================

func TestEventLog_Append_ConcurrentSafe(t *testing.T) {
	l := NewEventLog(nil)
	const goroutines = 20
	const perG = 50

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		g := g
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				_ = l.Append(MessageSentEvent{
					EventBase: NewEventBase("MessageSent", fmt.Sprintf("g%d", g), LamportValue(i)),
					MsgID:     fmt.Sprintf("g%d_m%d", g, i),
				})
			}
		}()
	}
	wg.Wait()

	if l.Len() != goroutines*perG {
		t.Errorf("expected %d events, got %d", goroutines*perG, l.Len())
	}
}

// ============================================================================
// Determinism — same input always produces same JSONL
// ============================================================================

func TestEventLog_ExportJSONL_Deterministic(t *testing.T) {
	build := func() *EventLog {
		l := NewEventLog(nil)
		_ = l.Append(MessageSentEvent{
			EventBase: EventBase{EventKind: "MessageSent", Clock: 1, Sender: "a", TimestampMs: 1000},
			To:        "b", MsgID: "m1",
		})
		_ = l.Append(PatchAppliedEvent{
			EventBase: EventBase{EventKind: "PatchApplied", Clock: 2, Sender: "a", TimestampMs: 1001},
			Region:    "r1", PatchType: "AddPanel", NodeID: "n1",
		})
		return l
	}

	var b1, b2 bytes.Buffer
	if err := build().ExportJSONL(&b1); err != nil {
		t.Fatal(err)
	}
	if err := build().ExportJSONL(&b2); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1.Bytes(), b2.Bytes()) {
		t.Errorf("ExportJSONL must be byte-deterministic for fixed inputs.\nfirst:  %s\nsecond: %s", b1.String(), b2.String())
	}
}
