package effects

import (
	"sync"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

func TestTraceRegistry_Basic(t *testing.T) {
	r := NewTraceRegistry()

	// Initially empty
	if r.Exists("compile.parse") {
		t.Error("expected empty registry to return false")
	}

	// Record a span
	r.Record("compile.parse")
	if !r.Exists("compile.parse") {
		t.Error("expected recorded span to exist")
	}

	// Count should be 1
	if count := r.Count("compile.parse"); count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Record again
	r.Record("compile.parse")
	if count := r.Count("compile.parse"); count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestTraceRegistry_PrefixMatch(t *testing.T) {
	r := NewTraceRegistry()

	r.Record("compile.parse")
	r.Record("compile.typecheck")
	r.Record("compile.elaborate")

	// Prefix match should work
	if !r.Exists("compile") {
		t.Error("expected prefix 'compile' to match compile.* spans")
	}

	// Exact match
	if !r.Exists("compile.parse") {
		t.Error("expected exact match to work")
	}

	// Non-matching prefix
	if r.Exists("eval") {
		t.Error("expected non-existent prefix to return false")
	}

	// Partial prefix that doesn't match
	if r.Exists("comp") {
		t.Error("expected partial prefix to NOT match (must be followed by '.')")
	}
}

func TestTraceRegistry_Clear(t *testing.T) {
	r := NewTraceRegistry()

	r.Record("span1")
	r.Record("span2")

	if !r.Exists("span1") || !r.Exists("span2") {
		t.Error("expected spans to exist after recording")
	}

	r.Clear()

	if r.Exists("span1") || r.Exists("span2") {
		t.Error("expected spans to be cleared")
	}
}

func TestTraceRegistry_All(t *testing.T) {
	r := NewTraceRegistry()

	r.Record("span1")
	r.Record("span2")
	r.Record("span1") // Record again

	all := r.All()

	if len(all) != 2 {
		t.Errorf("expected 2 unique spans, got %d", len(all))
	}

	if all["span1"] != 2 {
		t.Errorf("expected span1 count 2, got %d", all["span1"])
	}

	if all["span2"] != 1 {
		t.Errorf("expected span2 count 1, got %d", all["span2"])
	}
}

func TestTraceRegistry_Concurrent(t *testing.T) {
	r := NewTraceRegistry()

	var wg sync.WaitGroup
	numGoroutines := 100
	numRecords := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numRecords; j++ {
				r.Record("concurrent.span")
			}
		}()
	}

	wg.Wait()

	expectedCount := numGoroutines * numRecords
	if count := r.Count("concurrent.span"); count != expectedCount {
		t.Errorf("expected count %d, got %d", expectedCount, count)
	}
}

func TestGlobalTraceRegistry(t *testing.T) {
	// Clear any existing state
	ClearGlobalTraces()

	// Record using global function
	RecordTrace("global.test")

	// Check using global registry
	if !GlobalTraceRegistry().Exists("global.test") {
		t.Error("expected global trace to be recorded")
	}

	// Clear
	ClearGlobalTraces()
	if GlobalTraceRegistry().Exists("global.test") {
		t.Error("expected global traces to be cleared")
	}
}

func TestTraceCheckImpl(t *testing.T) {
	// Clear any existing state
	ClearGlobalTraces()

	// Record a trace
	RecordTrace("test.span")

	tests := []struct {
		name     string
		input    eval.Value
		expected bool
	}{
		{
			name:     "existing span",
			input:    &eval.StringValue{Value: "test.span"},
			expected: true,
		},
		{
			name:     "prefix match",
			input:    &eval.StringValue{Value: "test"},
			expected: true,
		},
		{
			name:     "non-existent span",
			input:    &eval.StringValue{Value: "other.span"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TraceCheckImpl(nil, []eval.Value{tt.input})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			boolResult, ok := result.(*eval.BoolValue)
			if !ok {
				t.Fatalf("expected *BoolValue, got %T", result)
			}

			if boolResult.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolResult.Value)
			}
		})
	}

	// Clean up
	ClearGlobalTraces()
}
