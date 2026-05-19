package effects

import (
	"fmt"
	"sync"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// Trace cognitive-event extension — M-COG-RUNTIME-BROWSER M5
// ============================================================================
//
// Bridges the existing trace.Collector spans into the cognitive event log.
// When AILANG code calls `traceEmit(span_name, duration_ns)`, the runtime:
//   1. Records the span via the existing trace.Collector (untouched)
//   2. Calls a registered CognitiveTraceSink (if configured) so the event
//      gets persisted to IndexedDB / the JSONL replay log
//
// Crucial invariant: the EXISTING `--emit-trace jsonl` output is BYTE-
// IDENTICAL pre/post this extension. The trace.Collector path is
// untouched; this file only adds a SIDE-CHANNEL hook for cognitive
// persistence. The cmd/wasm bridge installs a CognitiveTraceSink at
// init time; native CLI runs leave the sink nil (no behavior change).

// CognitiveTraceSink is the pluggable hook for cognitive-trace events.
// Implementations route to the cognitive event log (IndexedDB sink in
// the browser, JSONL/file sink for native).
//
// Sink methods are called from the cognitive Trace ops only — NOT from
// the existing trace.Collector path. So enabling/disabling the sink
// affects M-COG-RUNTIME-BROWSER replay only; trace export semantics
// remain unchanged.
type CognitiveTraceSink interface {
	EmitCognitiveTrace(spanName string, durationNs int64)
}

// cognitiveSinkRegistry holds the process-wide sink (nil = disabled).
// Mutex-guarded for goroutine safety across set/get from the WASM
// init path + the trace-op call path.
var (
	cognitiveSinkMu sync.Mutex
	cognitiveSink   CognitiveTraceSink
)

// SetCognitiveTraceSink installs (or clears) the cognitive sink.
// Called by cmd/wasm/effects_cognition.go init() to route Trace events
// into the browser's IndexedDB log. Pass nil to disable.
func SetCognitiveTraceSink(sink CognitiveTraceSink) {
	cognitiveSinkMu.Lock()
	defer cognitiveSinkMu.Unlock()
	cognitiveSink = sink
}

// emitCognitiveTrace is the dispatch hook called by the Trace op below.
// Goroutine-safe; nil-safe (no sink = no-op).
func emitCognitiveTrace(spanName string, durationNs int64) {
	cognitiveSinkMu.Lock()
	s := cognitiveSink
	cognitiveSinkMu.Unlock()
	if s != nil {
		s.EmitCognitiveTrace(spanName, durationNs)
	}
}

// ============================================================================
// Op registration
// ============================================================================
//
// The "emit" op is a NEW op added under the existing Trace effect. It
// does NOT replace the existing trace.Collector hooks — it's an extra
// op that AILANG code can call to record a named cognitive event.

func init() {
	RegisterOp("Trace", "emit", traceEmit)
}

// traceEmit implements Trace.emit(span_name: string, duration_ns: int) -> () ! Trace.
//
// Records a TraceCaptured event into the cognitive sink (if configured).
// The existing trace.Collector is untouched — if the AILANG caller also
// wants this event in the standard --emit-trace output, they call
// Collector.RecordEvent separately.
func traceEmit(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("E_TRACE_TYPE_ERROR: emit: expected 2 args (span_name, duration_ns), got %d", len(args))
	}
	name, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_TRACE_TYPE_ERROR: emit: span_name must be string, got %T", args[0])
	}
	dur, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("E_TRACE_TYPE_ERROR: emit: duration_ns must be int, got %T", args[1])
	}
	emitCognitiveTrace(name.Value, int64(dur.Value))
	return &eval.UnitValue{}, nil
}
