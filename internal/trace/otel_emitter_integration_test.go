package trace

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// loadTraceFile reads a JSONL trace fixture from the repo's examples/traces dir.
func loadTraceFile(t *testing.T, rel string) []TraceEvent {
	t.Helper()
	path := "../../" + rel
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	events, err := ReadJSONL(f)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(events) == 0 {
		t.Fatalf("%s has no events", path)
	}
	return events
}

// countByPrefix counts exported span names starting with any of the prefixes.
func countByPrefix(names []string, prefix string) int {
	n := 0
	for _, name := range names {
		if strings.HasPrefix(name, prefix) {
			n++
		}
	}
	return n
}

// TestIntegration_FibTrace_StandardTier verifies the "standard" tier filters
// out every per-call eval.function.* span on a real recursive fib trace,
// which is the regression scenario for M-OBS-TRACE-TRIAGE.
func TestIntegration_FibTrace_StandardTier(t *testing.T) {
	events := loadTraceFile(t, "examples/traces/recursion_fibonacci.jsonl")

	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	err := EmitOTELSpansWithOptions(context.Background(), tracer, events, time.Now(),
		TracingOptions{Tier: TierStandard, MaxSpansPerTrace: 10000})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	names := make([]string, 0, len(exporter.GetSpans()))
	for _, s := range exporter.GetSpans() {
		names = append(names, s.Name)
	}

	if got := countByPrefix(names, "eval.function."); got != 0 {
		t.Errorf("TierStandard: want 0 eval.function.* spans, got %d", got)
	}
	if got := countByPrefix(names, "eval.module."); got < 1 {
		t.Errorf("TierStandard: want ≥1 module span, got %d", got)
	}
}

// TestIntegration_FibTrace_DeepTier verifies the "deep" tier emits all the
// per-call function spans that standard drops.
func TestIntegration_FibTrace_DeepTier(t *testing.T) {
	events := loadTraceFile(t, "examples/traces/recursion_fibonacci.jsonl")

	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	err := EmitOTELSpansWithOptions(context.Background(), tracer, events, time.Now(),
		TracingOptions{Tier: TierDeep, MaxSpansPerTrace: 10000})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	names := make([]string, 0, len(exporter.GetSpans()))
	for _, s := range exporter.GetSpans() {
		names = append(names, s.Name)
	}

	if got := countByPrefix(names, "eval.function."); got < 10 {
		t.Errorf("TierDeep: want many eval.function.* spans on recursive fib, got %d", got)
	}
}

// TestIntegration_FibTrace_OffTier verifies the "off" tier emits nothing.
func TestIntegration_FibTrace_OffTier(t *testing.T) {
	events := loadTraceFile(t, "examples/traces/recursion_fibonacci.jsonl")

	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	err := EmitOTELSpansWithOptions(context.Background(), tracer, events, time.Now(),
		TracingOptions{Tier: TierOff})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}
	if got := len(exporter.GetSpans()); got != 0 {
		t.Errorf("TierOff: want 0 spans, got %d", got)
	}
}

// TestIntegration_FibTrace_BudgetTruncates verifies that a tiny budget
// on a real recursive trace produces (budget + 1) spans — the kept spans
// plus exactly one trace.truncated rollup.
func TestIntegration_FibTrace_BudgetTruncates(t *testing.T) {
	events := loadTraceFile(t, "examples/traces/recursion_fibonacci.jsonl")

	exporter, tp := setupTestTracer()
	defer tp.Shutdown(context.Background())
	tracer := tp.Tracer("test")

	const budget = 10
	err := EmitOTELSpansWithOptions(context.Background(), tracer, events, time.Now(),
		TracingOptions{Tier: TierDeep, MaxSpansPerTrace: budget})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	spans := exporter.GetSpans()
	if got := len(spans); got != budget+1 {
		t.Fatalf("budget=%d on recursive fib: want %d spans, got %d", budget, budget+1, got)
	}

	rollups := 0
	for _, s := range spans {
		if s.Name == "trace.truncated" {
			rollups++
		}
	}
	if rollups != 1 {
		t.Errorf("want exactly 1 trace.truncated span, got %d", rollups)
	}
}
