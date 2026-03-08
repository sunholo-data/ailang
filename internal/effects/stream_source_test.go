package effects

import (
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/eval"
)

// mockSource implements EventSource for testing
type mockSource struct {
	name     string
	priority int
	ch       chan streamEvent
	closed   bool
}

func newMockSource(name string, priority int, bufSize int) *mockSource {
	return &mockSource{
		name:     name,
		priority: priority,
		ch:       make(chan streamEvent, bufSize),
	}
}

func (ms *mockSource) Name() string               { return ms.name }
func (ms *mockSource) Priority() int              { return ms.priority }
func (ms *mockSource) Events() <-chan streamEvent { return ms.ch }
func (ms *mockSource) Close() {
	if !ms.closed {
		ms.closed = true
		close(ms.ch)
	}
}

func (ms *mockSource) Send(evt streamEvent) {
	ms.ch <- evt
}

// --- connSource tests ---

func TestConnSourceName(t *testing.T) {
	conn := &StreamConnection{
		eventBuffer: make(chan streamEvent, 10),
	}
	src := NewConnSource(conn, "ws:echo.example.com", 10)
	if src.Name() != "ws:echo.example.com" {
		t.Errorf("expected name 'ws:echo.example.com', got %q", src.Name())
	}
}

func TestConnSourcePriority(t *testing.T) {
	conn := &StreamConnection{
		eventBuffer: make(chan streamEvent, 10),
	}
	src := NewConnSource(conn, "ws:test", 42)
	if src.Priority() != 42 {
		t.Errorf("expected priority 42, got %d", src.Priority())
	}
}

func TestConnSourceEventsChannel(t *testing.T) {
	conn := &StreamConnection{
		eventBuffer: make(chan streamEvent, 10),
	}
	src := NewConnSource(conn, "ws:test", 0)

	// Send event through connection's buffer
	conn.eventBuffer <- streamEvent{kind: "message", text: "hello"}

	// Should be readable from source's Events channel
	select {
	case evt := <-src.Events():
		if evt.kind != "message" || evt.text != "hello" {
			t.Errorf("unexpected event: %+v", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestConnSourceCloseIdempotent(t *testing.T) {
	conn := &StreamConnection{
		eventBuffer: make(chan streamEvent, 10),
	}
	src := NewConnSource(conn, "ws:test", 0)

	// Close twice should not panic
	src.Close()
	src.Close()
}

// --- StreamContext source management tests ---

func TestStreamContextAcquireSource(t *testing.T) {
	ctx := NewStreamContext()
	src := newMockSource("test", 0, 10)

	id := ctx.AcquireSource(src)
	if id < 0 {
		t.Errorf("expected non-negative source ID, got %d", id)
	}

	got, ok := ctx.GetSource(id)
	if !ok {
		t.Fatal("source not found after acquire")
	}
	if got.Name() != "test" {
		t.Errorf("expected name 'test', got %q", got.Name())
	}
}

func TestStreamContextReleaseSource(t *testing.T) {
	ctx := NewStreamContext()
	src := newMockSource("test", 0, 10)

	id := ctx.AcquireSource(src)
	ctx.ReleaseSource(id)

	_, ok := ctx.GetSource(id)
	if ok {
		t.Fatal("source should not be found after release")
	}
}

func TestStreamContextMultipleSources(t *testing.T) {
	ctx := NewStreamContext()
	s1 := newMockSource("a", 0, 10)
	s2 := newMockSource("b", 0, 10)

	id1 := ctx.AcquireSource(s1)
	id2 := ctx.AcquireSource(s2)

	if id1 == id2 {
		t.Error("source IDs should be unique")
	}

	got1, _ := ctx.GetSource(id1)
	got2, _ := ctx.GetSource(id2)
	if got1.Name() != "a" || got2.Name() != "b" {
		t.Errorf("source names mismatch: got %q and %q", got1.Name(), got2.Name())
	}
}

// --- eventToADT tests for new event kinds ---

func TestEventToADT_SourceText(t *testing.T) {
	evt := streamEvent{kind: "source_text", sourceName: "stdin", text: "hello world"}
	val := eventToADT(evt)

	tagged, ok := val.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", val)
	}
	if tagged.CtorName != "SourceText" {
		t.Errorf("expected ctor SourceText, got %q", tagged.CtorName)
	}
	if len(tagged.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(tagged.Fields))
	}
	src := tagged.Fields[0].(*eval.StringValue).Value
	txt := tagged.Fields[1].(*eval.StringValue).Value
	if src != "stdin" || txt != "hello world" {
		t.Errorf("unexpected fields: source=%q text=%q", src, txt)
	}
}

func TestEventToADT_SourceBytes(t *testing.T) {
	evt := streamEvent{kind: "source_bytes", sourceName: "pipe:audio", data: []byte{0x01, 0x02, 0x03}}
	val := eventToADT(evt)

	tagged, ok := val.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", val)
	}
	if tagged.CtorName != "SourceBytes" {
		t.Errorf("expected ctor SourceBytes, got %q", tagged.CtorName)
	}
	if len(tagged.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(tagged.Fields))
	}
	src := tagged.Fields[0].(*eval.StringValue).Value
	data := tagged.Fields[1].(*eval.BytesValue).Value
	if src != "pipe:audio" {
		t.Errorf("unexpected source: %q", src)
	}
	if len(data) != 3 || data[0] != 0x01 {
		t.Errorf("unexpected bytes: %v", data)
	}
}
