package cognition

import (
	"errors"
	"sync"
	"testing"
)

// ============================================================================
// Subscribe / Unsubscribe / Dispatch
// ============================================================================

func TestScheduler_Dispatch_DeliversToAllSubscribers(t *testing.T) {
	s := NewScheduler(nil)
	var count1, count2 int
	s.Subscribe("", func(e Event) { count1++ })
	s.Subscribe("", func(e Event) { count2++ })

	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)})
	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 2)})

	if count1 != 2 || count2 != 2 {
		t.Errorf("expected both subscribers to receive 2 events, got %d / %d", count1, count2)
	}
}

func TestScheduler_Dispatch_FilterByKind(t *testing.T) {
	s := NewScheduler(nil)
	var msgCount, patchCount, allCount int
	s.Subscribe("MessageSent", func(e Event) { msgCount++ })
	s.Subscribe("PatchApplied", func(e Event) { patchCount++ })
	s.Subscribe("", func(e Event) { allCount++ })

	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)})
	s.Dispatch(PatchAppliedEvent{EventBase: NewEventBase("PatchApplied", "a", 2)})
	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 3)})

	if msgCount != 2 {
		t.Errorf("MessageSent subscriber: got %d, want 2", msgCount)
	}
	if patchCount != 1 {
		t.Errorf("PatchApplied subscriber: got %d, want 1", patchCount)
	}
	if allCount != 3 {
		t.Errorf("all-kinds subscriber: got %d, want 3", allCount)
	}
}

func TestScheduler_Unsubscribe(t *testing.T) {
	s := NewScheduler(nil)
	var count int
	sub := s.Subscribe("", func(e Event) { count++ })

	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)})
	s.Unsubscribe(sub)
	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 2)})

	if count != 1 {
		t.Errorf("after unsubscribe, expected 1 delivery, got %d", count)
	}
}

func TestScheduler_Dispatch_IgnoresNil(t *testing.T) {
	s := NewScheduler(nil)
	var count int
	s.Subscribe("", func(e Event) { count++ })

	s.Dispatch(nil)
	if count != 0 {
		t.Errorf("nil dispatch should not invoke handlers, got %d invocations", count)
	}
}

func TestScheduler_Subscribe_IgnoresNilHandler(t *testing.T) {
	s := NewScheduler(nil)
	sub := s.Subscribe("", nil)
	if sub.id != -1 {
		t.Errorf("expected sentinel id -1 for nil handler, got %d", sub.id)
	}
	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)})
	// No panic, no crash — pass.
}

// ============================================================================
// RunFromLog — canonical ordering by (Clock, Sender)
// ============================================================================

func TestScheduler_RunFromLog_OrdersByLamportClock(t *testing.T) {
	s := NewScheduler(nil)
	got := []LamportValue{}
	s.Subscribe("", func(e Event) {
		got = append(got, e.Base().Clock)
	})

	// Events in shuffled order
	events := []Event{
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 5)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 3)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 2)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 4)},
	}
	if err := s.RunFromLog(events); err != nil {
		t.Fatal(err)
	}

	for i, want := range []LamportValue{1, 2, 3, 4, 5} {
		if got[i] != want {
			t.Errorf("dispatch[%d]: got clock %d, want %d (full: %v)", i, got[i], want, got)
		}
	}
}

func TestScheduler_RunFromLog_BreaksTiesBySender(t *testing.T) {
	s := NewScheduler(nil)
	got := []string{}
	s.Subscribe("", func(e Event) {
		got = append(got, e.Base().Sender)
	})

	// All clock=5; sender order determines dispatch
	events := []Event{
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "node_c", 5)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "node_a", 5)},
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "node_b", 5)},
	}
	if err := s.RunFromLog(events); err != nil {
		t.Fatal(err)
	}

	for i, want := range []string{"node_a", "node_b", "node_c"} {
		if got[i] != want {
			t.Errorf("dispatch[%d]: got sender %q, want %q (full: %v)", i, got[i], want, got)
		}
	}
}

// ============================================================================
// Determinism — replay byte-equivalence
// ============================================================================

func TestScheduler_RunFromLog_DeterministicAcrossRuns(t *testing.T) {
	build := func() []Event {
		// Reproducible event list
		return []Event{
			MessageSentEvent{EventBase: NewEventBase("MessageSent", "node_b", 3), MsgID: "m3"},
			PatchAppliedEvent{EventBase: NewEventBase("PatchApplied", "node_a", 1), Region: "r1", PatchType: "AddPanel"},
			MessageSentEvent{EventBase: NewEventBase("MessageSent", "node_a", 2), MsgID: "m2"},
			TraceCapturedEvent{EventBase: NewEventBase("TraceCaptured", "node_a", 4), SpanName: "span"},
		}
	}

	run := func() []Event {
		s := NewScheduler(nil)
		s.Subscribe("", func(e Event) {})
		_ = s.RunFromLog(build())
		return s.Dispatched()
	}

	d1 := run()
	d2 := run()
	if len(d1) != len(d2) {
		t.Fatalf("dispatch length differs: %d vs %d", len(d1), len(d2))
	}
	for i := range d1 {
		b1, b2 := d1[i].Base(), d2[i].Base()
		if b1.Clock != b2.Clock || b1.Sender != b2.Sender || b1.EventKind != b2.EventKind {
			t.Errorf("dispatch[%d] differs: run1=%+v run2=%+v", i, b1, b2)
		}
	}
}

// ============================================================================
// Log integration
// ============================================================================

func TestScheduler_Dispatch_AppendsToLogWhenConfigured(t *testing.T) {
	log := NewEventLog(nil)
	s := NewScheduler(log)
	s.Subscribe("", func(e Event) {})

	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)})
	s.Dispatch(PatchAppliedEvent{EventBase: NewEventBase("PatchApplied", "a", 2)})

	if log.Len() != 2 {
		t.Errorf("scheduler should append every dispatch to log, got Len=%d", log.Len())
	}
}

func TestScheduler_NilLog_DoesNotPanic(t *testing.T) {
	s := NewScheduler(nil)
	s.Subscribe("", func(e Event) {})
	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)})
	// Pass = no panic
}

// ============================================================================
// Stop — graceful shutdown
// ============================================================================

func TestScheduler_Stop_BlocksDispatch(t *testing.T) {
	s := NewScheduler(nil)
	var count int
	s.Subscribe("", func(e Event) { count++ })

	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)})
	s.Stop()
	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 2)})

	if count != 1 {
		t.Errorf("after Stop, dispatch should be no-op; got count=%d", count)
	}
}

func TestScheduler_RunFromLog_AfterStop_ReturnsError(t *testing.T) {
	s := NewScheduler(nil)
	s.Stop()
	err := s.RunFromLog([]Event{
		MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)},
	})
	if !errors.Is(err, ErrSchedulerStopped) {
		t.Errorf("expected ErrSchedulerStopped, got %v", err)
	}
}

func TestScheduler_Stop_Idempotent(t *testing.T) {
	s := NewScheduler(nil)
	s.Stop()
	s.Stop() // no panic
}

// ============================================================================
// Re-entrant Subscribe from inside Handler (no deadlock)
// ============================================================================

func TestScheduler_HandlerCanReSubscribe(t *testing.T) {
	s := NewScheduler(nil)
	var added int

	// First handler subscribes a second one on its first invocation.
	s.Subscribe("MessageSent", func(e Event) {
		if added == 0 {
			added++
			s.Subscribe("MessageSent", func(e Event) {})
		}
	})

	// If Dispatch held the lock across handler invocation this would
	// deadlock; the test passes iff dispatch completes.
	s.Dispatch(MessageSentEvent{EventBase: NewEventBase("MessageSent", "a", 1)})
	if added != 1 {
		t.Errorf("expected handler to subscribe once, got %d", added)
	}
}

// ============================================================================
// Concurrent Dispatch — multiple goroutines must produce consistent
// observable order (sequential consistency at the scheduler boundary)
// ============================================================================

func TestScheduler_Dispatched_ConcurrentSafe(t *testing.T) {
	s := NewScheduler(nil)
	s.Subscribe("", func(e Event) {})

	const goroutines = 10
	const perG = 50
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		g := g
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				s.Dispatch(MessageSentEvent{
					EventBase: NewEventBase("MessageSent",
						"g"+string(rune('0'+g)), LamportValue(i)),
				})
			}
		}()
	}
	wg.Wait()

	dispatched := s.Dispatched()
	if len(dispatched) != goroutines*perG {
		t.Errorf("expected %d dispatches, got %d", goroutines*perG, len(dispatched))
	}
}
