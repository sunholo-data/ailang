package trace

import (
	"context"
	"os"
	"testing"
	"time"
)

// makeFunctionBurst builds an event stream with N top-level function calls.
// Each call is a single enter/exit pair at depth 1 — emits one span each.
func makeFunctionBurst(n int) []TraceEvent {
	events := make([]TraceEvent, 0, 2+2*n)
	events = append(events, TraceEvent{
		Version: "1.0", Event: EventModuleStart, TimestampNS: 0, Depth: 0,
		Module: &ModuleEvent{Name: "burst"},
	})
	ts := int64(100)
	for i := 0; i < n; i++ {
		events = append(events,
			TraceEvent{Version: "1.0", Event: EventFunctionEnter, TimestampNS: ts, Depth: 1,
				Function: &FunctionEvent{Name: "f"}},
			TraceEvent{Version: "1.0", Event: EventFunctionExit, TimestampNS: ts + 1, Depth: 1,
				Function: &FunctionEvent{Name: "f", Result: "()"}},
		)
		ts += 2
	}
	events = append(events, TraceEvent{
		Version: "1.0", Event: EventModuleEnd, TimestampNS: ts, Depth: 0,
		Module: &ModuleEvent{Name: "burst"},
	})
	return events
}

func TestBudget_Enforced(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	// 50 function calls → 50 function spans + 1 module = 51 spans.
	// Budget=10 → keep 10, drop 41, emit 1 rollup → total 11 spans.
	const budget = 10
	events := makeFunctionBurst(50)
	err := EmitOTELSpansWithOptions(context.Background(), tracer, events, time.Now(),
		TracingOptions{Tier: TierDeep, MaxSpansPerTrace: budget})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	if got := len(spans); got != budget+1 {
		t.Fatalf("budget=%d: want %d total spans (budget + 1 rollup), got %d", budget, budget+1, got)
	}

	// Exactly one rollup span.
	rollupCount := 0
	var rollup *struct {
		dropped int64
		first   string
	}
	_ = rollup
	for _, s := range spans {
		if s.Name == "trace.truncated" {
			rollupCount++
			var dropped int64
			var first string
			for _, a := range s.Attributes {
				switch string(a.Key) {
				case "ailang.trace.dropped_count":
					dropped = a.Value.AsInt64()
				case "ailang.trace.first_dropped_name":
					first = a.Value.AsString()
				}
			}
			if dropped <= 0 {
				t.Errorf("rollup dropped_count=%d, want >0", dropped)
			}
			if first == "" {
				t.Error("rollup first_dropped_name is empty")
			}
		}
	}
	if rollupCount != 1 {
		t.Errorf("want exactly 1 trace.truncated span, got %d", rollupCount)
	}
}

func TestBudget_NoRollupWhenUnder(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	events := makeFunctionBurst(3) // 3 funcs + 1 module = 4 spans
	err := EmitOTELSpansWithOptions(context.Background(), tracer, events, time.Now(),
		TracingOptions{Tier: TierDeep, MaxSpansPerTrace: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range exporter.GetSpans() {
		if s.Name == "trace.truncated" {
			t.Error("should not emit trace.truncated when under budget")
		}
	}
}

func TestBudget_Zero_Unlimited(t *testing.T) {
	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	events := makeFunctionBurst(20)
	err := EmitOTELSpansWithOptions(context.Background(), tracer, events, time.Now(),
		TracingOptions{Tier: TierDeep, MaxSpansPerTrace: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 20 func + 1 module = 21 spans, no rollup.
	if got := len(exporter.GetSpans()); got != 21 {
		t.Errorf("budget=0 (unlimited): want 21 spans, got %d", got)
	}
}

func TestDefaultTracingOptions_EnvOverride(t *testing.T) {
	old := os.Getenv("AILANG_TRACE_MAX_SPANS")
	defer os.Setenv("AILANG_TRACE_MAX_SPANS", old)

	os.Setenv("AILANG_TRACE_MAX_SPANS", "42")
	if got := DefaultTracingOptions().MaxSpansPerTrace; got != 42 {
		t.Errorf("env override: got %d, want 42", got)
	}

	os.Setenv("AILANG_TRACE_MAX_SPANS", "not-a-number")
	if got := DefaultTracingOptions().MaxSpansPerTrace; got != 500 {
		t.Errorf("invalid env: got %d, want default 500", got)
	}

	os.Unsetenv("AILANG_TRACE_MAX_SPANS")
	if got := DefaultTracingOptions().MaxSpansPerTrace; got != 500 {
		t.Errorf("no env: got %d, want default 500", got)
	}
}
