# M-TRACE-BRIDGE: Bridge OTEL Spans to TraceRegistry

**Status**: Planned
**Target**: v0.7.1
**Priority**: P2 (Medium)
**Estimated**: 0.5 days (simpler than expected!)
**Dependencies**: M-TRACE-TEST (completed)

## Problem Statement

The trace testing framework (M-TRACE-TEST) provides `_trace_check` for verifying traces, but it uses a separate in-memory `TraceRegistry`. Meanwhile, **extensive OTEL instrumentation already exists** throughout the codebase - these spans are sent to external collectors but aren't visible to `_trace_check`.

**Current State:**
- ✅ OTEL spans exist: `compile.parse`, `compile.elaborate`, `compile.typecheck`, etc.
- ✅ OTEL spans sent to Jaeger/GCP Cloud Trace
- ❌ `effects.RecordTrace(name)` is never called
- ❌ `_trace_check("compile.parse")` always returns false (separate system)

**Desired State:**
- OTEL spans automatically recorded to TraceRegistry
- `_trace_check("compile.parse")` returns true after compilation
- Single instrumentation point, dual visibility (OTEL + TraceRegistry)

## Existing OTEL Instrumentation (Audit Results)

**Already instrumented - no new spans needed:**

| Component | Spans | File |
|-----------|-------|------|
| **Compiler Pipeline** | `compile.parse`, `compile.elaborate`, `compile.typecheck`, `compile.validate`, `compile.lower`, `compile.load`, `compile.topo_sort`, `compile.modules` | `internal/pipeline/pipeline_single.go`, `pipeline_module.go` |
| **REPL** | `repl.session`, `repl.input` | `internal/repl/repl.go` |
| **AI Providers** | `gemini.generate`, `anthropic.generate`, `ollama.generate`, `openai.generate` | `internal/ai/*/client.go` |
| **Executors** | `claude.execute`, `gemini.execute`, `exec.turn` | `internal/executor/*/` |
| **Messaging** | `messages.send`, `messages.list`, `messages.read`, `messages.ack`, `messages.search`, `messages.cleanup` | `internal/messaging/` |
| **Coordinator** | `approval.decision`, `human.feedback`, `human.approval`, `task.iteration_start` | `internal/coordinator/` |

## Solution Design

### Overview

Instead of adding duplicate instrumentation, **wrap the OTEL tracer** to also record to TraceRegistry. This is a one-line change at each tracer definition.

### Option A: Wrapper Tracer (Recommended)

Create a wrapper that records span names when spans are created:

```go
// internal/telemetry/traced_tracer.go
package telemetry

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "github.com/sunholo/ailang/internal/effects"
)

// TracedTracer wraps an OTEL tracer to also record to TraceRegistry
type TracedTracer struct {
    inner trace.Tracer
}

func NewTracedTracer(name string) trace.Tracer {
    return &TracedTracer{inner: otel.Tracer(name)}
}

func (t *TracedTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
    // Record to TraceRegistry for _trace_check
    effects.RecordTrace(spanName)

    // Delegate to real OTEL tracer
    return t.inner.Start(ctx, spanName, opts...)
}
```

**Usage change (one line per file):**
```go
// Before
var compilerTracer = otel.Tracer("ailang.compiler")

// After
var compilerTracer = telemetry.NewTracedTracer("ailang.compiler")
```

### Option B: Hook into OTEL SpanProcessor

Register a custom SpanProcessor that records to TraceRegistry:

```go
// internal/telemetry/trace_recorder.go
type TraceRecorderProcessor struct{}

func (p *TraceRecorderProcessor) OnStart(parent context.Context, s trace.ReadWriteSpan) {
    effects.RecordTrace(s.Name())
}

func (p *TraceRecorderProcessor) OnEnd(s trace.ReadOnlySpan) {}
func (p *TraceRecorderProcessor) Shutdown(ctx context.Context) error { return nil }
func (p *TraceRecorderProcessor) ForceFlush(ctx context.Context) error { return nil }
```

Register in `InitOTLP`:
```go
tracerProvider := sdktrace.NewTracerProvider(
    sdktrace.WithSpanProcessor(&TraceRecorderProcessor{}),
    // ... existing config
)
```

**Pros:** Zero changes to existing tracer definitions
**Cons:** Requires OTEL to be initialized (won't work when telemetry disabled)

### Recommended: Option A + Environment Toggle

Use Option A (wrapper tracer) with an environment toggle:

```go
func NewTracedTracer(name string) trace.Tracer {
    inner := otel.Tracer(name)
    if os.Getenv("AILANG_TRACE_RECORDING") != "1" {
        return inner  // No wrapper, zero overhead
    }
    return &TracedTracer{inner: inner}
}
```

## Implementation Plan

### Phase 1: Create Wrapper (~2 hours)

1. Create `internal/telemetry/traced_tracer.go` with wrapper implementation
2. Add `AILANG_TRACE_RECORDING` environment check
3. Write unit tests for wrapper

### Phase 2: Update Tracer Definitions (~1 hour)

Update these files to use `telemetry.NewTracedTracer`:
- `internal/pipeline/pipeline.go` (compilerTracer)
- `internal/repl/repl.go` (replTracer)
- `internal/ai/*/client.go` (4 files)
- `internal/executor/*/` (2 files)
- `internal/messaging/store.go` (messagingTracer)
- `internal/coordinator/*.go` (4 files)

### Phase 3: Integration Test (~1 hour)

1. Enable `AILANG_TRACE_RECORDING=1`
2. Compile a file
3. Verify `_trace_check("compile.parse")` returns true
4. Update example to demonstrate

## Success Criteria

- [ ] `AILANG_TRACE_RECORDING=1` enables recording to TraceRegistry
- [ ] Existing OTEL spans automatically visible to `_trace_check`
- [ ] Zero overhead when `AILANG_TRACE_RECORDING` not set
- [ ] No changes to span names or OTEL behavior
- [ ] Integration test passes

## Example After Implementation

```bash
# Enable trace recording
AILANG_TRACE_RECORDING=1 ailang run file.ail
```

```ailang
import stdlib/trace_test (assert_trace_exists)

-- These now work because OTEL spans are bridged!
export pure func verify_compilation() -> int {
  assert_trace_exists("compile.parse");      -- true!
  assert_trace_exists("compile.typecheck");  -- true!
  assert_trace_exists("compile.lower");      -- true!
  1
}

-- Can also verify AI provider calls
export pure func verify_ai_calls() -> int {
  assert_trace_exists("anthropic.generate"); -- true if Claude was called
  assert_trace_exists("gemini.generate");    -- true if Gemini was called
  1
}
```

## Non-Goals

- Adding new instrumentation points (already covered)
- Changing OTEL span names or attributes
- Trace persistence (TraceRegistry is in-memory)
- Performance profiling (traces are presence-only)

## Risks

**Risk**: Performance overhead from double-tracking
**Mitigation**: Disabled by default, wrapper is just a map insert

**Risk**: TraceRegistry memory growth
**Mitigation**: Clear between test runs, add LRU if needed

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact on determinism |
| A2: Replayability | +1 | Enables trace-based test replay |
| A3: Effect Legibility | +1 | Makes existing spans queryable |
| A7: Machines First | +1 | Improves automated testing |

**Net Score: +3** ✅ Proceed to implementation

## References

- **Prerequisite**: M-TRACE-TEST (trace testing framework)
- **Existing instrumentation**: `internal/pipeline/pipeline_single.go:81` and 40+ other locations
- **TraceRegistry**: `internal/effects/trace.go`

## Summary

This is much simpler than originally scoped. The instrumentation already exists - we just need to bridge OTEL spans to the TraceRegistry. Estimated effort reduced from 1-2 days to **0.5 days**.
