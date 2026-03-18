# M-SERVE-API-CONCURRENCY: Per-Request Evaluator for Thread-Safe serve-api

**Status**: Planned
**Target**: v0.9.4
**Priority**: P0 (Critical — blocks any production deployment on Cloud Run or multi-user hosting)
**Estimated**: 0.5-1 day
**Dependencies**: None
**Milestone ID**: M-SERVE-API-CONCURRENCY
**Created**: 2026-03-18
**Source**: Production deployment analysis for Cloud Run — one VM must handle concurrent requests

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | **Fixes critical violation**: concurrent requests produce nondeterministic results due to shared mutable evaluator state. Per-request evaluator restores determinism. |
| A2: Replayability | +1 | Same inputs now produce same outputs regardless of concurrent load |
| A3: Effect Legibility | +1 | Effect context no longer bleeds between concurrent requests |
| A4: Explicit Authority | +1 | One request's capabilities can no longer leak to another's |
| A5: Bounded Verification | 0 | No change to verification |
| A6: Safe Concurrency | +2 | **Core fix**: eliminates three race conditions in the evaluator |
| A7: Machines First | +1 | Enables multi-request AI agent workflows against serve-api |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Cloud Run cost model depends on concurrency — 1 VM serving N requests vs N VMs |
| A10: Composability | 0 | No change |
| A11: Structured Failure | 0 | No change |
| A12: System Boundary | 0 | No change |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): **Fixes** nondeterminism from shared mutable state
- [x] A3 (Effects): **Fixes** effect context bleeding between requests
- [x] A4 (Authority): **Fixes** capability leakage between requests
- [x] A7 (Machines First): Enables concurrent AI agent workflows

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

No — this is a **single architectural fix**. The evaluator was designed for single-threaded CLI execution (`ailang run`) where there's only one active evaluation at a time. The serve-api use case (concurrent goroutines) was not in the original design.

The fix is clean: the evaluator is cheap to create, module instances are immutable after loading, and all mutable state is per-evaluation. This is a textbook case for per-request isolation.

---

## Problem Statement

### The Race Conditions

AILANG's evaluator (`CoreEvaluator`) is a **single shared instance** across all HTTP requests:

```
Engine (1 per server)
  └─ ModuleRuntime (1 per engine)
       └─ CoreEvaluator (1 per runtime) ← SHARED BY ALL REQUESTS
```

Three fields are mutated without synchronization during request handling:

| Field | Mutated By | Race |
|-------|-----------|------|
| `resolver` | `CallEntrypoint` → `SetGlobalResolver()` | Request A sets module A's resolver, Request B overwrites with module B's → A evaluates with wrong module context |
| `effContext` | `evaluateModule` → `SetEffContext()` | Request A's capabilities bleed into Request B |
| `env` | `CallFunction` → `e.env = newEnv` | Request A's environment swap interleaves with Request B's |

### Concrete Failure Scenario

```
Request A (module: docparse/api)        Request B (module: docparse/health)
   ↓                                      ↓
CallEntrypoint(rt, instA, "parse")     CallEntrypoint(rt, instB, "check")
   ↓                                      ↓
SetGlobalResolver(resolverA)           SetGlobalResolver(resolverB)  ← OVERWRITES A's
   ↓
CallFunction(parse, args)
   → resolver.ResolveValue("std/xml.parse")
   → uses resolverB (module health) ← WRONG MODULE
   → "std/xml not imported by docparse/health"
   → REQUEST A FAILS
```

### Impact

- **Cloud Run**: Cannot serve concurrent requests on a single instance. Must set `--max-instances=N` with `--concurrency=1`, meaning one VM per request — 10-100x the cost.
- **Any multi-user API**: Requests interfere with each other unpredictably.
- **Load testing**: Failures appear under concurrent load but not during development.

### Current State

```go
// internal/eval/eval_evaluator.go
type CoreEvaluator struct {
    env                   *Environment           // MUTABLE — swapped per function call
    registry              *types.DictionaryRegistry  // READ-ONLY after init
    resolver              GlobalResolver         // MUTABLE — set per entrypoint call
    effContext            interface{}            // MUTABLE — set during module eval
    recursionDepth        int                    // MUTABLE — tracks call depth
    maxRecursionDepth     int                    // READ-ONLY after init
    coreTypeInfo          types.CoreTypeInfo     // MUTABLE — set during module eval
}
```

Only `registry` and `maxRecursionDepth` are safe to share. Everything else is per-evaluation state.

---

## Goals

**Primary Goal:** Make serve-api safe for concurrent requests by creating a fresh evaluator per request, enabling multi-request Cloud Run deployment.

**Success Metrics:**
- 100 concurrent requests to different modules produce correct results (verified with `-race`)
- No `go test -race` failures in apiserver, runtime, or eval packages
- Cloud Run deploys with `--concurrency=80` (default) instead of `--concurrency=1`
- No performance regression for sequential requests (evaluator creation is ~1µs)
- Existing `ailang run` behavior unchanged

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Per-request evaluator vs mutex | Determines concurrency model for all future serve-api work | human | design | high |
| What state is shared vs cloned | Incorrect sharing = races; over-cloning = performance | agent | compile | med |
| Where to create the evaluator | Engine.Call vs CallEntrypoint vs handler | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Per-request evaluator** chosen over mutex (mutex serializes all requests, defeating the purpose)
- [x] **Shared read-only state**: registry (DictionaryRegistry), module instances (immutable after loading), EffContext config
- [x] **Per-request state**: env, resolver, recursionDepth, coreTypeInfo

---

## Solution Design

### Overview

Replace the shared `CoreEvaluator` with per-request evaluator instances. The `ModuleRuntime` keeps its evaluator for module loading (single-threaded startup), but `CallEntrypoint` creates a fresh evaluator for each function call.

### Architecture

```
BEFORE (shared evaluator):
  Engine → ModuleRuntime → CoreEvaluator (shared)
                              ↓
  Request A ──────────────→ SetGlobalResolver(A) → CallFunction
  Request B ──────────────→ SetGlobalResolver(B) → CallFunction  ← RACE

AFTER (per-request evaluator):
  Engine → ModuleRuntime → CoreEvaluator (for module loading only)

  Request A → CoreEvaluator(A) → SetGlobalResolver(A) → CallFunction
  Request B → CoreEvaluator(B) → SetGlobalResolver(B) → CallFunction  ← SAFE
```

### Implementation

**Step 1: Add `Fork()` method to CoreEvaluator**

Creates a lightweight copy with shared read-only state and fresh per-request state:

```go
// internal/eval/eval_evaluator.go

// Fork creates a new evaluator that shares read-only state (registry)
// but has independent per-request state (env, resolver, effContext).
// This is the concurrency primitive: each goroutine gets its own Fork.
func (e *CoreEvaluator) Fork() *CoreEvaluator {
    env := NewEnvironment()
    registerBuiltins(env)

    return &CoreEvaluator{
        env:                   env,
        registry:              e.registry,          // shared, read-only after init
        experimentalBinopShim: e.experimentalBinopShim,
        effContext:            e.effContext,         // copied reference (EffContext is goroutine-safe for reads)
        maxRecursionDepth:     e.maxRecursionDepth,
        // resolver: nil       — set by CallEntrypoint
        // recursionDepth: 0   — fresh per request
        // coreTypeInfo: zero  — set during evaluation
    }
}
```

**Step 2: Use forked evaluator in `CallEntrypoint`**

```go
// internal/runtime/entrypoint.go

func CallEntrypoint(rt *ModuleRuntime, inst *ModuleInstance, name string, args []eval.Value) (eval.Value, error) {
    entrypoint, err := inst.GetExport(name)
    if err != nil {
        return nil, err
    }

    fn, ok := entrypoint.(*eval.FunctionValue)
    if !ok {
        return nil, fmt.Errorf("entrypoint '%s' is not a function (got %T)", name, entrypoint)
    }

    // Fork the evaluator for this request — each goroutine gets its own copy
    reqEval := rt.evaluator.Fork()

    // Set up resolver for this request's module context
    resolver := newModuleGlobalResolver(inst, rt)
    reqEval.SetGlobalResolver(resolver)

    // Call the function on the forked evaluator
    return reqEval.CallFunction(fn, args)
}
```

**Step 3: Verify EffContext is safe for concurrent reads**

`EffContext` is set once at startup and read during evaluation. The mutable fields (`FnCaller`, `FnCallerN`) are set during `wireFnCallers` which happens at startup, not per-request. We need to wire FnCallers on the forked evaluator:

```go
func (e *CoreEvaluator) Fork() *CoreEvaluator {
    forked := &CoreEvaluator{
        env:                   NewEnvironment(),
        registry:              e.registry,
        experimentalBinopShim: e.experimentalBinopShim,
        effContext:            e.effContext,
        maxRecursionDepth:     e.maxRecursionDepth,
    }
    registerBuiltins(forked.env)

    // Wire FnCallers so the forked evaluator's CallValue/CallValueN
    // are used for iterative builtins (map, filter, foldl)
    if forked.effContext != nil {
        forked.wireFnCallers()
    }
    return forked
}
```

Wait — `wireFnCallers` sets `ctx.FnCaller = e.CallValue` on the **shared** EffContext. This means a forked evaluator would overwrite the parent's FnCaller binding. We need per-request EffContext too:

```go
func (e *CoreEvaluator) Fork() *CoreEvaluator {
    forked := &CoreEvaluator{
        env:                   NewEnvironment(),
        registry:              e.registry,
        experimentalBinopShim: e.experimentalBinopShim,
        maxRecursionDepth:     e.maxRecursionDepth,
    }
    registerBuiltins(forked.env)

    // Clone EffContext so each request has its own FnCaller bindings
    if e.effContext != nil {
        forked.effContext = cloneEffContext(e.effContext)
        forked.wireFnCallers()
    }
    return forked
}
```

Where `cloneEffContext` shallow-copies the EffContext struct (all fields are either immutable configs or per-request function pointers):

```go
// internal/eval/eval_evaluator.go
func cloneEffContext(ctx interface{}) interface{} {
    type cloneable interface {
        Clone() interface{}
    }
    if c, ok := ctx.(cloneable); ok {
        return c.Clone()
    }
    return ctx // fallback: share (not ideal but safe for read-only)
}
```

And add `Clone()` to EffContext:

```go
// internal/effects/context.go
func (ctx *EffContext) Clone() interface{} {
    clone := *ctx  // shallow copy — all config fields are value types or shared pointers
    return &clone
}
```

### What's Shared vs Cloned

| State | Strategy | Rationale |
|-------|----------|-----------|
| `registry` (DictionaryRegistry) | **Shared** | Populated at startup, read-only during evaluation |
| `maxRecursionDepth` | **Shared** | Config constant |
| `experimentalBinopShim` | **Shared** | Config flag |
| Module instances | **Shared** | Immutable after loading (exports, bindings frozen by sync.Once) |
| `env` | **Fresh** | New builtin env per request (function calls push/pop scopes) |
| `resolver` | **Per-request** | Set to calling module's resolver |
| `recursionDepth` | **Fresh** | Starts at 0 per request |
| `coreTypeInfo` | **Per-request** | Set during evaluation |
| `effContext` | **Cloned** | Shallow copy so FnCaller bindings are per-request |

### Files to Modify

**Modified files:**
- `internal/eval/eval_evaluator.go` (~+25 LOC) — Add `Fork()` method, `cloneEffContext` helper
- `internal/effects/context.go` (~+5 LOC) — Add `Clone()` method
- `internal/runtime/entrypoint.go` (~+3/-2 LOC) — Use `rt.evaluator.Fork()` instead of shared evaluator

**No new files needed.**

---

## Examples

### Before (races under concurrent load)

```
$ wrk -c 10 -t 4 http://localhost:8080/api/math/add
# Intermittent errors:
#   "module std/string not imported by api/health"
#   "undefined binding 'x' in module api/greet"
# Results vary between runs
```

### After (correct under concurrent load)

```
$ wrk -c 10 -t 4 http://localhost:8080/api/math/add
# All requests succeed
# Results are deterministic
# go test -race passes
```

### Cloud Run deployment

```yaml
# Before: forced to serialize
spec:
  containerConcurrency: 1  # One request per VM = expensive

# After: full concurrency
spec:
  containerConcurrency: 80  # Default, 80 concurrent requests per VM
```

---

## Success Criteria

- [ ] `go test -race ./internal/eval/... ./internal/runtime/... ./internal/apiserver/...` passes
- [ ] 100 concurrent requests to different modules all return correct results
- [ ] 100 concurrent requests to the same module all return correct results
- [ ] `Fork()` allocates <1µs (benchmark)
- [ ] Sequential request latency unchanged (no regression)
- [ ] `ailang run` behavior unchanged (doesn't use Fork)
- [ ] All existing tests pass
- [ ] Lint clean

---

## Testing Strategy

**Race detection:**
- `go test -race` on eval, runtime, and apiserver packages
- Concurrent integration test: spawn 10 goroutines calling different modules simultaneously

**Correctness:**
- Test that concurrent calls to module A and module B return correct results for each
- Test that effect context doesn't leak between requests (one with IO caps, one pure)

**Performance:**
- Benchmark `Fork()` — target <1µs
- Benchmark sequential vs concurrent request throughput

**Manual:**
- `wrk` or `hey` load test against serve-api with multiple modules

---

## Deferred Decisions

- **EffContext pooling** — If Fork() allocation becomes a bottleneck under extreme load, we could pool evaluators with `sync.Pool`. Agent may choose to add this if benchmarks show need.
- **Per-request effect budgets** — Currently effect budgets are per-evaluator. With Fork, each request gets fresh budgets. This is correct for serve-api (each request has its own budget) but may need review for other use cases.

## Non-Goals

- **Evaluator caching/pooling** — Not needed. Go's garbage collector handles short-lived structs efficiently.
- **Concurrent module loading** — Module loading happens at startup (single-threaded). Only request handling needs concurrency.
- **Shared mutable state between requests** — AILANG is pure/explicit-effects. There's no use case for sharing mutable state between requests.
- **Making the existing evaluator thread-safe with locks** — This would serialize requests, defeating the purpose of Go's concurrency model.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fork() misses a shared mutable field | High — subtle races | Audit every field in CoreEvaluator; run `go test -race` |
| EffContext.Clone() misses mutable sub-fields | Med — effect leaks | EffContext fields are mostly config; FnCaller/FnCallerN are the only per-request fields |
| registerBuiltins() per fork is expensive | Low — measured | Benchmark; if slow, share builtin env (it's read-only) |
| Module evaluation (startup) uses shared evaluator | Low — single-threaded | Module loading completes before requests start |

---

## Related Documents

- [M-SERVE-API-DX](./m-serve-api-dx.md) — Parent feature for serve-api production readiness
- [M-SERVE-API-TRANSITIVE-IMPORTS](./m-serve-api-transitive-imports.md) — Transitive import fix (also serve-api)
- [M-R2: Effect System](../../implemented/v0_2_0/m_r2_effect_system.md) — EffContext design
- [M-EVAL-GUARD: Eval Process Guardrails](../../implemented/v0_5_6/m-eval-process-guardrails.md) — Process-level isolation (different scope but related concept)
- [M-BUILTIN-SAFETY](../../implemented/v0_7_0/m-builtin-safety-type-checks.md) — Evaluator safety patterns

---

## Future Work

- **Evaluator pooling** — `sync.Pool` for evaluator reuse if allocation proves costly at scale
- **Per-request tracing** — Attach OTEL span to forked evaluator for request-level telemetry
- **Concurrent module loading** — Parallelize startup loading for large projects (19+ modules)
- **Request-scoped effect budgets** — Per-request AI/IO budget tracking for rate limiting

---

**Document created**: 2026-03-18
**Last updated**: 2026-03-18
