package effects

import (
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// captureSink records every cognitive trace event for assertion.
type captureSink struct {
	mu     sync.Mutex
	events []captureSinkEvent
}

type captureSinkEvent struct {
	SpanName   string
	DurationNs int64
}

func (s *captureSink) EmitCognitiveTrace(spanName string, durationNs int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, captureSinkEvent{SpanName: spanName, DurationNs: durationNs})
}

func (s *captureSink) snapshot() []captureSinkEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]captureSinkEvent, len(s.events))
	copy(out, s.events)
	return out
}

func TestTraceEmit_RoutesToCognitiveSink(t *testing.T) {
	sink := &captureSink{}
	SetCognitiveTraceSink(sink)
	defer SetCognitiveTraceSink(nil)

	ctx := &EffContext{}
	_, err := traceEmit(ctx, []eval.Value{
		&eval.StringValue{Value: "demo_span"},
		&eval.IntValue{Value: 1234},
	})
	if err != nil {
		t.Fatalf("traceEmit: %v", err)
	}
	events := sink.snapshot()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].SpanName != "demo_span" || events[0].DurationNs != 1234 {
		t.Errorf("unexpected event: %+v", events[0])
	}
}

func TestTraceEmit_NoSink_IsNoOp(t *testing.T) {
	// Ensure no sink is installed
	SetCognitiveTraceSink(nil)
	ctx := &EffContext{}
	_, err := traceEmit(ctx, []eval.Value{
		&eval.StringValue{Value: "no_sink_span"},
		&eval.IntValue{Value: 0},
	})
	if err != nil {
		t.Errorf("traceEmit without sink should be no-op, got error: %v", err)
	}
}

func TestTraceEmit_WrongArgCount_Errors(t *testing.T) {
	SetCognitiveTraceSink(nil)
	ctx := &EffContext{}
	_, err := traceEmit(ctx, []eval.Value{&eval.StringValue{Value: "only"}})
	if err == nil {
		t.Fatal("expected error for missing duration arg")
	}
}

func TestTraceEmit_WrongArgTypes_Errors(t *testing.T) {
	SetCognitiveTraceSink(nil)
	ctx := &EffContext{}
	_, err := traceEmit(ctx, []eval.Value{
		&eval.IntValue{Value: 0}, // span_name should be string
		&eval.IntValue{Value: 0},
	})
	if err == nil {
		t.Fatal("expected error for wrong span_name type")
	}
}

func TestSetCognitiveTraceSink_ConcurrentSafe(t *testing.T) {
	// Concurrent SetSink + emit must not race
	const goroutines = 10
	const perG = 100
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				sink := &captureSink{}
				SetCognitiveTraceSink(sink)
			}
		}()
		go func() {
			defer wg.Done()
			ctx := &EffContext{}
			for i := 0; i < perG; i++ {
				_, _ = traceEmit(ctx, []eval.Value{
					&eval.StringValue{Value: "stress"},
					&eval.IntValue{Value: 1},
				})
			}
		}()
	}
	wg.Wait()
	// Pass if no race condition (run with -race for full coverage)
}
