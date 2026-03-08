package effects

import (
	"strings"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/eval"
)

// TestIntegration_StdinWithSelectEvents verifies the full pipeline:
// stdinSource → selectEventsLoop → handler receives SourceText events.
func TestIntegration_StdinWithSelectEvents(t *testing.T) {
	input := "line1\nline2\nline3\n"
	reader := strings.NewReader(input)

	stdinSrc := NewStdinSource(reader, "stdin", 10)
	defer stdinSrc.Close()

	handler := &testHandler{stopAfter: 0} // Don't stop; let EOF close
	fnCaller := mockFnCaller(handler)

	err := selectEventsLoop(
		[]EventSource{stdinSrc},
		&eval.UnitValue{},
		fnCaller,
		5*time.Second,
		10*time.Second,
	)
	if err != nil {
		t.Fatalf("selectEventsLoop error: %v", err)
	}

	expected := []string{
		"SourceText:stdin:line1",
		"SourceText:stdin:line2",
		"SourceText:stdin:line3",
	}
	if len(handler.events) != len(expected) {
		t.Fatalf("received %d events, want %d: %v", len(handler.events), len(expected), handler.events)
	}
	for i, exp := range expected {
		if handler.events[i] != exp {
			t.Errorf("event[%d] = %q, want %q", i, handler.events[i], exp)
		}
	}
}

// TestIntegration_StdinPriorityOverMock verifies that a higher-priority stdin
// source is dispatched before a lower-priority mock source.
func TestIntegration_StdinPriorityOverMock(t *testing.T) {
	stdinReader := strings.NewReader("stdin-line\n")
	stdinSrc := NewStdinSource(stdinReader, "stdin", 10) // high priority
	mockSrc := newMockSource("mock", 1, 10)              // low priority
	defer stdinSrc.Close()

	// Pre-populate both sources
	mockSrc.Send(streamEvent{kind: "message", text: "mock-msg"})
	time.Sleep(50 * time.Millisecond) // Let stdin goroutine populate

	handler := &testHandler{stopAfter: 2}
	fnCaller := mockFnCaller(handler)

	err := selectEventsLoop(
		[]EventSource{mockSrc, stdinSrc},
		&eval.UnitValue{},
		fnCaller,
		5*time.Second,
		10*time.Second,
	)
	if err != nil {
		t.Fatalf("selectEventsLoop error: %v", err)
	}

	if len(handler.events) != 2 {
		t.Fatalf("received %d events, want 2: %v", len(handler.events), handler.events)
	}

	// High-priority stdin should come first
	if handler.events[0] != "SourceText:stdin:stdin-line" {
		t.Errorf("first event = %q, want stdin (high priority)", handler.events[0])
	}
	if handler.events[1] != "Message:mock-msg" {
		t.Errorf("second event = %q, want mock (low priority)", handler.events[1])
	}
}

// TestIntegration_HandlerStopsOnFalse verifies handler returning false stops the loop.
func TestIntegration_HandlerStopsOnFalse(t *testing.T) {
	input := "stop\nignored\n"
	reader := strings.NewReader(input)
	src := NewStdinSource(reader, "stdin", 0)
	defer src.Close()

	handler := &testHandler{stopAfter: 1} // Stop after first event
	fnCaller := mockFnCaller(handler)

	err := selectEventsLoop(
		[]EventSource{src},
		&eval.UnitValue{},
		fnCaller,
		5*time.Second,
		10*time.Second,
	)
	if err != nil {
		t.Fatalf("selectEventsLoop error: %v", err)
	}

	if len(handler.events) != 1 {
		t.Errorf("handler received %d events, want 1 (should stop after returning false): %v",
			len(handler.events), handler.events)
	}
}
