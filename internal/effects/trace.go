package effects

import (
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/eval"
)

// TraceRegistry provides a simple in-memory registry for recording span names
// during program execution. This enables trace verification in tests.
//
// The registry is designed for testing scenarios where you want to verify
// that certain trace spans were created during execution.
//
// Thread-safe for concurrent access.
type TraceRegistry struct {
	mu    sync.RWMutex
	spans map[string]int // span name -> count
}

// globalTraceRegistry is the default registry used by _trace_check
var globalTraceRegistry = NewTraceRegistry()

// NewTraceRegistry creates a new empty trace registry.
func NewTraceRegistry() *TraceRegistry {
	return &TraceRegistry{
		spans: make(map[string]int),
	}
}

// Record records that a span with the given name was created.
// Can be called multiple times for the same name to track count.
func (r *TraceRegistry) Record(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.spans[name]++
}

// Exists returns true if a span with the given name exists in the registry.
// Supports prefix matching: "compile" matches "compile.parse", "compile.typecheck", etc.
func (r *TraceRegistry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Exact match first
	if _, ok := r.spans[name]; ok {
		return true
	}

	// Prefix match (for hierarchical span names like "compile.parse")
	for spanName := range r.spans {
		if strings.HasPrefix(spanName, name+".") || spanName == name {
			return true
		}
	}

	return false
}

// Count returns the number of times a span with the given name was recorded.
// Returns 0 if the span doesn't exist.
func (r *TraceRegistry) Count(name string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.spans[name]
}

// Clear removes all recorded spans from the registry.
// Useful for resetting state between tests.
func (r *TraceRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.spans = make(map[string]int)
}

// All returns a copy of all recorded span names and their counts.
func (r *TraceRegistry) All() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]int, len(r.spans))
	for k, v := range r.spans {
		result[k] = v
	}
	return result
}

// GlobalTraceRegistry returns the global trace registry.
// This is used by the _trace_check builtin.
func GlobalTraceRegistry() *TraceRegistry {
	return globalTraceRegistry
}

// RecordTrace records a span name in the global registry.
// Call this from instrumented code points (compiler, evaluator, etc.)
func RecordTrace(name string) {
	globalTraceRegistry.Record(name)
}

// ClearGlobalTraces clears all traces from the global registry.
// Useful for resetting state between tests.
func ClearGlobalTraces() {
	globalTraceRegistry.Clear()
}

// TraceCheckImpl is the implementation for the _trace_check builtin.
// It checks if a trace with the given name exists in the global registry.
//
// Type: string -> bool
func TraceCheckImpl(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	nameVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return &eval.BoolValue{Value: false}, nil
	}

	exists := globalTraceRegistry.Exists(nameVal.Value)
	return &eval.BoolValue{Value: exists}, nil
}
