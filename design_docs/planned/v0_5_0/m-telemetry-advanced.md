# M-TELEMETRY-ADVANCED: Advanced Pipeline Telemetry

**Status:** Planned
**Priority:** Medium
**Sprint:** v0.5.x
**Dependencies:** M-AGENT-MONITOR (completed)

## Overview

This design doc proposes additional telemetry features for the AILANG pipeline, building on the low-overhead metrics implemented in M-AGENT-MONITOR Phase 3. These features provide deeper visibility into compilation and runtime behavior but come with higher performance overhead.

## Current State (v0.5.0)

### Low-Overhead Metrics (Implemented)

Enabled via `AILANG_METRICS=1`:

| Metric | Overhead | Description |
|--------|----------|-------------|
| Phase timings | ~0% | Load, topo, compile, evaluate timing |
| Memory delta | ~0% | Memory allocated during pipeline |
| Allocation count | ~0% | Number of allocations |
| Specialization count | ~0% | Monomorphization statistics |

These metrics add negligible overhead (<1ms) and can be enabled in production.

## Proposed Medium-Overhead Metrics

Enable via `AILANG_METRICS=2` or `AILANG_METRICS_LEVEL=medium`:

### 1. Per-Module Breakdown (~5-10% overhead)

Track timing for each module in the dependency graph:

```go
type ModuleMetrics struct {
    ModuleID      string `json:"module_id"`
    ParseTime     int64  `json:"parse_ms"`
    ElaborateTime int64  `json:"elaborate_ms"`
    TypeCheckTime int64  `json:"typecheck_ms"`
    LowerTime     int64  `json:"lower_ms"`
    DeclCount     int    `json:"decl_count"`
    ImportCount   int    `json:"import_count"`
}

type PipelineMetrics struct {
    // ... existing fields ...
    Modules []ModuleMetrics `json:"modules,omitempty"`
}
```

**Use Cases:**
- Identify slow modules in large projects
- Detect import cycles causing recompilation
- Optimize module organization

**Implementation:**
- Add timing hooks in `pipeline_module.go` for each module compilation phase
- Store per-module metrics in a slice
- Report via telemetry/WebSocket to Monitor UI

### 2. Type Inference Statistics (~5% overhead)

Track type system behavior:

```go
type TypeInferenceMetrics struct {
    UnificationCalls   int   `json:"unification_calls"`
    SubstitutionApps   int   `json:"substitution_apps"`
    ConstraintsSolved  int   `json:"constraints_solved"`
    ConstraintsUnsolved int  `json:"constraints_unsolved"`
    MaxUnifyDepth      int   `json:"max_unify_depth"`
    TypeVarsCreated    int   `json:"type_vars_created"`
}
```

**Use Cases:**
- Debug type inference performance issues
- Identify code patterns that stress the type checker
- Optimize constraint solving

**Implementation:**
- Add counters in `internal/types/unify.go`
- Track during `InferWithConstraints` calls
- Aggregate across all modules

### 3. AST Node Counts (~2% overhead)

Count nodes by type in Surface and Core AST:

```go
type ASTMetrics struct {
    SurfaceNodes map[string]int `json:"surface_nodes"` // Lambda, Let, App, etc.
    CoreNodes    map[string]int `json:"core_nodes"`    // After elaboration
    TotalNodes   int            `json:"total_nodes"`
}
```

**Use Cases:**
- Measure code complexity
- Understand elaboration expansion ratio
- Identify AST patterns for optimization

**Implementation:**
- Walk AST in `internal/ast/walk.go`
- Count nodes by type during elaboration

## Proposed High-Overhead Metrics

Enable via `AILANG_METRICS=3` or `AILANG_METRICS_LEVEL=high`:

### 1. Execution Tracing (~50-100% overhead)

Capture detailed execution traces:

```go
type ExecutionTrace struct {
    Timestamp     int64  `json:"ts"`
    EventType     string `json:"event"`  // "call", "return", "effect", "error"
    FunctionName  string `json:"fn,omitempty"`
    Arguments     []any  `json:"args,omitempty"`
    ReturnValue   any    `json:"ret,omitempty"`
    EffectName    string `json:"effect,omitempty"`
    Duration      int64  `json:"dur_ns,omitempty"`
}

type RuntimeMetrics struct {
    Traces         []ExecutionTrace `json:"traces"`
    TotalCalls     int              `json:"total_calls"`
    EffectCount    map[string]int   `json:"effect_count"`
    RecursionDepth int              `json:"max_recursion"`
}
```

**Use Cases:**
- Debug complex control flow
- Generate training data for AI models
- Performance profiling at function level
- Verify effect usage

**Implementation:**
- Hook into `internal/eval/core_eval.go` at `App` evaluation
- Capture entry/exit with timing
- Buffer traces and flush periodically

### 2. Memory Profiling (~100% overhead)

Track allocation patterns:

```go
type MemoryProfile struct {
    AllocsByPhase   map[string]int64 `json:"allocs_by_phase"`
    LargestAllocs   []AllocInfo      `json:"largest_allocs"`
    GCPauses        []int64          `json:"gc_pauses_ns"`
    HeapInUse       int64            `json:"heap_in_use"`
    HeapObjects     int64            `json:"heap_objects"`
}

type AllocInfo struct {
    Size     int64  `json:"size"`
    Type     string `json:"type"`
    Location string `json:"loc"`
}
```

**Use Cases:**
- Identify memory leaks
- Optimize allocation patterns
- Debug OOM issues

**Implementation:**
- Use `runtime.MemProfile()` at phase boundaries
- Track heap statistics via `runtime.ReadMemStats()`
- Sample allocation stacks (expensive)

### 3. Hot Path Analysis (~200% overhead)

Profile which code paths are executed most:

```go
type HotPathProfile struct {
    FunctionCalls  map[string]int   `json:"fn_calls"`      // fn_name -> count
    BranchCoverage map[string]int   `json:"branch_hit"`    // location -> count
    LoopIterations map[string]int64 `json:"loop_iters"`    // location -> total iters
    Hotspots       []Hotspot        `json:"hotspots"`
}

type Hotspot struct {
    Location   string  `json:"loc"`
    Calls      int     `json:"calls"`
    TotalTime  int64   `json:"total_ns"`
    AvgTime    int64   `json:"avg_ns"`
    Percentage float64 `json:"pct"`
}
```

**Use Cases:**
- Optimize critical paths
- Guide monomorphization decisions
- Identify loop-heavy code for potential optimization

**Implementation:**
- Instrument function entry/exit in evaluator
- Track call counts and timing
- Aggregate and compute hotspots

## UI Integration

### Monitor Tab Enhancements

The Collaboration Hub's Monitor tab should display:

1. **Phase Timeline**: Visual timeline showing phase durations
2. **Module Graph**: Dependency graph with timing color-coding
3. **Type Stats**: Real-time constraint solving progress
4. **Execution Flame Graph**: For high-level tracing (when enabled)

### WebSocket Events

New telemetry events for real-time streaming:

```typescript
interface MetricsEvent {
    type: 'metrics_update';
    instanceId: string;
    level: 'low' | 'medium' | 'high';
    metrics: PipelineMetrics;
    modules?: ModuleMetrics[];
    typeInference?: TypeInferenceMetrics;
    execution?: RuntimeMetrics;
}
```

## Environment Variables

| Variable | Values | Description |
|----------|--------|-------------|
| `AILANG_METRICS` | `0`, `1`, `2`, `3` | Enable metrics at level (0=off, 1=low, 2=medium, 3=high) |
| `AILANG_METRICS_LEVEL` | `off`, `low`, `medium`, `high` | Alternative to numeric levels |
| `AILANG_METRICS_VERBOSE` | `0`, `1` | Print metrics summary to stderr |
| `AILANG_METRICS_DEBUG` | `0`, `1` | Debug metrics collection itself |
| `AILANG_HUB_URL` | URL | Send metrics to collaboration hub |
| `AILANG_TRACE_FILE` | path | Write execution traces to file (JSON Lines) |

## Implementation Plan

### Phase 1: Medium-Level Metrics (v0.5.1)

1. Add per-module timing breakdown
2. Add type inference statistics counters
3. Update Monitor UI with module timeline

**Estimated effort:** 2-3 days
**Risk:** Low - additive changes, behind feature flag

### Phase 2: High-Level Tracing (v0.5.2)

1. Add execution trace infrastructure
2. Implement trace buffer with overflow handling
3. Add trace file export

**Estimated effort:** 3-4 days
**Risk:** Medium - requires careful performance tuning

### Phase 3: Memory & Hot Path Profiling (v0.5.3+)

1. Memory profiling integration
2. Hot path analysis
3. Flame graph generation

**Estimated effort:** 5+ days
**Risk:** High - significant performance impact, complex implementation

## Testing Strategy

1. **Overhead Benchmarks**: Measure compile time with/without each level
2. **Accuracy Tests**: Verify metrics match expected values for known inputs
3. **Integration Tests**: Test WebSocket streaming, file export
4. **Load Tests**: Ensure metrics don't cause memory leaks

## Security Considerations

- Execution traces may contain sensitive data (function arguments)
- Trace files should not be auto-created in production
- WebSocket metrics should respect authentication
- Consider redaction options for sensitive values

## Alternatives Considered

1. **pprof integration**: Go's built-in profiler
   - Pro: Mature, well-understood
   - Con: Not AILANG-aware, can't track semantic phases

2. **OpenTelemetry**: Standard observability framework
   - Pro: Industry standard, rich ecosystem
   - Con: Heavy dependency, overkill for single-binary CLI

3. **Custom sampling profiler**: Statistical sampling
   - Pro: Lower overhead than instrumentation
   - Con: Less accurate for short runs, complex to implement

## References

- [M-AGENT-MONITOR](./m-agent-monitor.md) - Monitor sprint design doc
- [pipeline/metrics.go](../../../internal/pipeline/metrics.go) - Low-level metrics implementation
- [Go runtime package](https://pkg.go.dev/runtime) - Memory statistics API
