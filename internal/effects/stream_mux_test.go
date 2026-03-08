package effects

import (
	"sync"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/eval"
)

// testHandler collects events and optionally stops after N events.
type testHandler struct {
	mu        sync.Mutex
	events    []string // Collected event descriptions
	stopAfter int      // Stop after this many events (0 = never stop)
	count     int
}

func (th *testHandler) handle(event eval.Value) bool {
	th.mu.Lock()
	defer th.mu.Unlock()

	th.count++
	desc := describeEvent(event)
	th.events = append(th.events, desc)

	if th.stopAfter > 0 && th.count >= th.stopAfter {
		return false
	}
	return true
}

func describeEvent(v eval.Value) string {
	tv, ok := v.(*eval.TaggedValue)
	if !ok {
		return "unknown"
	}
	switch tv.CtorName {
	case "Message":
		return "Message:" + tv.Fields[0].(*eval.StringValue).Value
	case "SourceText":
		src := tv.Fields[0].(*eval.StringValue).Value
		txt := tv.Fields[1].(*eval.StringValue).Value
		return "SourceText:" + src + ":" + txt
	case "StreamError":
		return "StreamError"
	default:
		return tv.CtorName
	}
}

// mockFnCaller creates a fnCaller that calls a Go function as if it were an AILANG handler.
func mockFnCaller(handler *testHandler) func(eval.Value, eval.Value) (eval.Value, error) {
	return func(_ eval.Value, arg eval.Value) (eval.Value, error) {
		cont := handler.handle(arg)
		return &eval.BoolValue{Value: cont}, nil
	}
}

func TestSelectEvents_SingleSource(t *testing.T) {
	src := newMockSource("ws:test", 0, 10)
	handler := &testHandler{stopAfter: 2}
	fnCaller := mockFnCaller(handler)

	go func() {
		src.Send(streamEvent{kind: "message", text: "a"})
		src.Send(streamEvent{kind: "message", text: "b"})
	}()

	err := selectEventsLoop(
		[]EventSource{src},
		&eval.UnitValue{}, // dummy handler value (fnCaller ignores it)
		fnCaller,
		60*time.Second,
		5*time.Minute,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.events) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(handler.events), handler.events)
	}
	if handler.events[0] != "Message:a" || handler.events[1] != "Message:b" {
		t.Errorf("unexpected events: %v", handler.events)
	}
}

func TestSelectEvents_PriorityOrdering(t *testing.T) {
	high := newMockSource("high", 10, 10)
	low := newMockSource("low", 1, 10)
	handler := &testHandler{stopAfter: 3}
	fnCaller := mockFnCaller(handler)

	// Pre-load both sources with events — high priority should be consumed first
	high.Send(streamEvent{kind: "source_text", sourceName: "high", text: "h1"})
	high.Send(streamEvent{kind: "source_text", sourceName: "high", text: "h2"})
	low.Send(streamEvent{kind: "source_text", sourceName: "low", text: "l1"})

	// Small delay to ensure events are buffered
	time.Sleep(10 * time.Millisecond)

	err := selectEventsLoop(
		[]EventSource{low, high}, // Passed in wrong order — mux should sort by priority
		&eval.UnitValue{},
		fnCaller,
		60*time.Second,
		5*time.Minute,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.events) != 3 {
		t.Fatalf("expected 3 events, got %d: %v", len(handler.events), handler.events)
	}
	// First two should be from high-priority source
	if handler.events[0] != "SourceText:high:h1" {
		t.Errorf("event 0: expected high-priority, got %s", handler.events[0])
	}
	if handler.events[1] != "SourceText:high:h2" {
		t.Errorf("event 1: expected high-priority, got %s", handler.events[1])
	}
	if handler.events[2] != "SourceText:low:l1" {
		t.Errorf("event 2: expected low-priority, got %s", handler.events[2])
	}
}

func TestSelectEvents_RoundRobinSamePriority(t *testing.T) {
	a := newMockSource("a", 5, 10)
	b := newMockSource("b", 5, 10)
	handler := &testHandler{stopAfter: 4}
	fnCaller := mockFnCaller(handler)

	// Pre-load both with 2 events each
	a.Send(streamEvent{kind: "source_text", sourceName: "a", text: "a1"})
	b.Send(streamEvent{kind: "source_text", sourceName: "b", text: "b1"})
	a.Send(streamEvent{kind: "source_text", sourceName: "a", text: "a2"})
	b.Send(streamEvent{kind: "source_text", sourceName: "b", text: "b2"})

	time.Sleep(10 * time.Millisecond)

	err := selectEventsLoop(
		[]EventSource{a, b},
		&eval.UnitValue{},
		fnCaller,
		60*time.Second,
		5*time.Minute,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.events) != 4 {
		t.Fatalf("expected 4 events, got %d: %v", len(handler.events), handler.events)
	}
	// Round-robin: should alternate a, b, a, b
	if handler.events[0] != "SourceText:a:a1" {
		t.Errorf("event 0: expected a, got %s", handler.events[0])
	}
	if handler.events[1] != "SourceText:b:b1" {
		t.Errorf("event 1: expected b, got %s", handler.events[1])
	}
	if handler.events[2] != "SourceText:a:a2" {
		t.Errorf("event 2: expected a, got %s", handler.events[2])
	}
	if handler.events[3] != "SourceText:b:b2" {
		t.Errorf("event 3: expected b, got %s", handler.events[3])
	}
}

func TestSelectEvents_HandlerReturnsFalse(t *testing.T) {
	src := newMockSource("test", 0, 10)
	handler := &testHandler{stopAfter: 1} // Stop after first event
	fnCaller := mockFnCaller(handler)

	src.Send(streamEvent{kind: "message", text: "only"})
	src.Send(streamEvent{kind: "message", text: "ignored"})

	err := selectEventsLoop(
		[]EventSource{src},
		&eval.UnitValue{},
		fnCaller,
		60*time.Second,
		5*time.Minute,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.events) != 1 {
		t.Fatalf("expected 1 event (handler said stop), got %d", len(handler.events))
	}
}

func TestSelectEvents_SourceCloseOthersContine(t *testing.T) {
	closeable := newMockSource("closeable", 0, 10)
	persistent := newMockSource("persistent", 0, 10)
	handler := &testHandler{stopAfter: 2}
	fnCaller := mockFnCaller(handler)

	closeable.Send(streamEvent{kind: "source_text", sourceName: "closeable", text: "c1"})

	go func() {
		time.Sleep(20 * time.Millisecond)
		closeable.Close() // Close one source
		time.Sleep(10 * time.Millisecond)
		persistent.Send(streamEvent{kind: "source_text", sourceName: "persistent", text: "p1"})
	}()

	err := selectEventsLoop(
		[]EventSource{closeable, persistent},
		&eval.UnitValue{},
		fnCaller,
		2*time.Second,
		5*time.Minute,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.events) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(handler.events), handler.events)
	}
}

func TestSelectEvents_IdleTimeout(t *testing.T) {
	src := newMockSource("test", 0, 10)
	handler := &testHandler{stopAfter: 0} // Never stops voluntarily
	fnCaller := mockFnCaller(handler)

	// Don't send any events — idle timeout should fire
	err := selectEventsLoop(
		[]EventSource{src},
		&eval.UnitValue{},
		fnCaller,
		100*time.Millisecond, // Short idle timeout
		5*time.Minute,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have received a StreamError (timeout)
	if len(handler.events) != 1 {
		t.Fatalf("expected 1 timeout event, got %d: %v", len(handler.events), handler.events)
	}
	if handler.events[0] != "StreamError" {
		t.Errorf("expected StreamError, got %s", handler.events[0])
	}
}

func TestSelectEvents_NoSources(t *testing.T) {
	handler := &testHandler{}
	fnCaller := mockFnCaller(handler)

	err := selectEventsLoop(
		[]EventSource{},
		&eval.UnitValue{},
		fnCaller,
		60*time.Second,
		5*time.Minute,
	)
	if err == nil {
		t.Fatal("expected error for empty sources")
	}
}
