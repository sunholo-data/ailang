# M-PERF-GOROUTINE-ID: Eliminate runtime.Stack() Bottleneck in Builtin Dispatch

**Status**: Planned
**Target**: v0.11.0
**Priority**: P0 (Critical — affects ALL AILANG programs)
**Estimated**: 1 day
**Dependencies**: None
**Milestone ID**: M-PERF-GOROUTINE-ID
**Created**: 2026-04-09
**Source**: CPU profiling of docparse Alice EPUB benchmark — 42% of runtime in `runtime.Stack()`

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure performance optimization, no semantic change |
| A2: Replayability | 0 | No trace format changes |
| A3: Effect Legibility | 0 | Effect context resolution unchanged (same result, faster path) |
| A4: Explicit Authority | 0 | Capability model unchanged |
| A5: Bounded Verification | 0 | No type system changes |
| A6: Safe Concurrency | +1 | Same isolation guarantees, validated by existing concurrency tests |
| A7: Machines First | +2 | **42% speedup on all programs** — directly improves AI agent throughput |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Reduces hidden per-builtin tax that distorted profiling data |
| A10: Composability | 0 | No semantic changes |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No semantic changes — same results, same order, just faster
- [x] A3 (Effects): Effect context resolution produces identical results via cheaper lookup
- [x] A4 (Authority): Capability model unchanged
- [x] A7 (Machines First): Actively improves machine processing speed by ~42%

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

**Yes — there are 4 independent copies of the same `runtime.Stack()` parsing pattern:**

| Location | Function | Usage |
|----------|----------|-------|
| `internal/runtime/builtins.go:126` | `builtinGoroutineID()` | **HOT PATH** — every builtin call |
| `internal/runtime/entrypoint.go:17` | `goroutineID()` | Debug logging only (`DEBUG_CONCURRENCY=1`) |
| `internal/apiserver/routes.go:21` | `goroutineID()` | Appears unused |
| `internal/embed/embed.go:41` | `embedGoroutineID()` | Debug logging in `CallPreserveFloats()` |

Only `builtins.go` is on the critical path. The others are debug-only or dead code, but all should be cleaned up.

**Root cause**: Go intentionally doesn't expose goroutine IDs. The `runtime.Stack()` hack was the pragmatic choice when M-SERVE-API-CONCURRENCY was implemented (v0.9.4) to enable per-request evaluator lookup. The design was correct for concurrency safety, but the performance cost was never measured.

**Regression timeline**: Every AILANG program has been ~42% slower since v0.9.4 landed. All subsequent bytecode VM work (M-BYTECODE-PHASE2E, M-BYTECODE-XML-BUILTINS, etc.) showed minimal speedup because the bottleneck was in shared infrastructure, not in the evaluator vs VM dispatch.

---

## Problem Statement

### The Bottleneck

Every AILANG builtin call follows this path:

```
builtinFn(args) 
  → br.getEffContext()                    // builtins.go:91
    → builtinGoroutineID()               // builtins.go:147
      → runtime.Stack(buf[:], false)     // Full goroutine stack dump
      → strings.TrimPrefix(...)          // String parsing
      → fmt.Sscanf(...)                  // Integer extraction
    → br.goroutineEvals.Load(id)         // sync.Map lookup
  → spec.Impl(ctx, args)                 // Actual work
```

**CPU profile (docparse Alice EPUB, 12.5s total):**

| Component | Time | % of Total |
|-----------|------|------------|
| `runtime.Stack` / `traceback2` | 5.3s | **42%** |
| Evaluator tree-walking | 4.4s | 35% |
| GC pressure | 1.5s | 12% |
| Type checking / compilation | 1.5s | 12% |

The `runtime.Stack()` call is expensive because it:
1. Stops the goroutine
2. Walks the entire call stack frame-by-frame
3. Formats it into a human-readable string
4. Allocates and copies a byte buffer

This happens **thousands of times** in any non-trivial program. Docparse calls builtins for every XML node, list operation, string manipulation, etc.

### Impact

This is **AILANG-wide**, not specific to docparse. Every program pays the same per-builtin tax:
- `string.length("hello")` → 1 `runtime.Stack()` call
- `list.map(xs, f)` → 1 call for `list.map` + 1 per callback invocation
- `xml.parse(s)` → 1 call
- A program with 10,000 builtin calls wastes ~4 seconds on goroutine ID lookup alone

### Why It Wasn't Caught

Sprint evaluation metrics focused on:
- EvalOnly counts (does it compile to bytecode?)
- Wall-clock time (is it faster overall?)
- Test pass rates

But never asked **where** CPU time is spent. Wall-clock comparisons showed marginal bytecode VM improvement but couldn't explain why — because the bottleneck was in shared infrastructure hit by both evaluator and VM paths.

**Recommendation**: Add CPU profiling to sprint executor's benchmark step (see Risks & Mitigations).

---

## Goals

**Primary Goal:** Eliminate the `runtime.Stack()` bottleneck so that goroutine-to-evaluator lookup costs O(1) without string parsing.

**Success Metrics:**
- `builtinGoroutineID` / `runtime.Stack` drops from 42% to <1% of CPU profile
- Docparse Alice EPUB: from 12s → ~7s (evaluator), target proportional improvement with bytecode
- Docparse Moby Dick EPUB: proportional improvement (~35s → ~20s)
- No regression in `serve-api` concurrent request isolation (existing tests pass)
- No regression in `make test`

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Use `sync.Map` with `goid` assembly vs context threading | Assembly is faster but non-portable; context threading is clean but requires API changes | human | design | high |
| Whether to also fix evaluator-side EffContext resolution | The VM path bypasses this entirely; evaluator is the main beneficiary | agent | compile | low |
| Whether to remove `goroutineEvals` entirely for single-goroutine case | Fast-path optimization: skip sync.Map when no Fork has been registered | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Approach**: Use `goid` package for cheap goroutine ID extraction (see Solution Design)
- [ ] **Portability**: Accept assembly dependency or use pure-Go fallback?

---

## Solution Design

### Overview

Replace the `runtime.Stack()` + string-parsing approach with Go's internal `runtime.goid` field, accessed either via a small assembly stub or the `github.com/petermattis/goid` package. This gives O(1) goroutine ID extraction without stack walks.

Additionally, add a fast-path for the common single-goroutine case (CLI, REPL) that skips the `sync.Map` lookup entirely.

### Architecture

**Option A: `goid` assembly (recommended)**

```go
// internal/runtime/goid.go — one file per GOARCH
// Uses go:linkname to access runtime.getg() or reads TLS directly

//go:nosplit
func goid() int64 // implemented in goid_amd64.s / goid_arm64.s
```

Pros: Zero allocation, ~2ns per call (vs ~500ns for `runtime.Stack`)
Cons: Assembly per architecture, relies on Go runtime internals

**Option B: `github.com/petermattis/goid` dependency**

```go
import "github.com/petermattis/goid"

func builtinGoroutineID() int64 {
    return goid.Get()
}
```

Pros: Well-maintained, supports all platforms, ~3ns per call
Cons: External dependency

**Option C: Pure-Go `runtime.Stack` with caching (fallback)**

```go
var goroutineIDCache sync.Map // per-goroutine, set once

func builtinGoroutineID() int64 {
    // Can't cache — goroutine ID IS the key we're looking for
    // This option doesn't work. Listed for completeness.
}
```

This doesn't work because we can't cache a goroutine ID without already knowing which goroutine we're on.

**Option D: Thread EffContext through call chain (long-term)**

Change builtin signatures from:
```go
func(args []eval.Value) (eval.Value, error)
```
to:
```go
func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error)
```

Pros: No goroutine ID needed at all, cleanest solution
Cons: Breaks every builtin signature (~100+ functions), major API change, affects VM interop layer

**Recommended approach**: Option B (external `goid` package) for immediate fix, with a fast-path check.

### Fast-Path Optimization

For the common case (CLI `ailang run`, REPL — single goroutine, no Fork):

```go
func (br *BuiltinRegistry) getEffContext() *effects.EffContext {
    // Fast path: no forked evaluators registered → use shared evaluator directly
    if br.goroutineEvalCount.Load() == 0 {
        return br.getEffContextFromEvaluator(br.evaluator)
    }
    
    // Slow path: concurrent serve-api → look up per-goroutine evaluator
    evaluator := br.evaluator
    if goroutineEval, ok := br.goroutineEvals.Load(goid.Get()); ok {
        evaluator = goroutineEval.(*eval.CoreEvaluator)
    }
    return br.getEffContextFromEvaluator(evaluator)
}
```

This means CLI programs pay **zero** lookup cost — no goroutine ID extraction, no `sync.Map` access.

### Implementation Plan

**Phase 1: Add goid + fast-path** (~2 hours)
- [ ] Add `github.com/petermattis/goid` dependency (or inline assembly)
- [ ] Replace `builtinGoroutineID()` with `goid.Get()` in `internal/runtime/builtins.go`
- [ ] Add `goroutineEvalCount atomic.Int64` to `BuiltinRegistry`
- [ ] Increment/decrement in `SetGoroutineEvaluator` / `ClearGoroutineEvaluator`
- [ ] Add fast-path check in `getEffContext()` — skip `sync.Map` when count is 0
- [ ] Run `make test` to verify no regressions

**Phase 2: Benchmark + verify** (~1 hour)
- [ ] Re-run docparse Alice EPUB benchmark (evaluator + bytecode)
- [ ] Re-run docparse Moby Dick EPUB benchmark
- [ ] CPU profile to confirm `runtime.Stack` is gone from hot path
- [ ] Run `serve-api` concurrency tests to verify isolation

**Phase 3: Cleanup** (~30 min)
- [ ] Remove duplicate `goroutineID()` from `internal/runtime/entrypoint.go` (replace with `goid.Get()`)
- [ ] Remove unused `goroutineID()` from `internal/apiserver/routes.go`
- [ ] Update `internal/embed/embed.go` to use shared implementation
- [ ] Update changelog

### Files to Modify/Create

**Modified files:**
- `internal/runtime/builtins.go` — Replace `builtinGoroutineID()`, add fast-path (~30 LOC changed)
- `internal/runtime/entrypoint.go` — Remove duplicate `goroutineID()` (~10 LOC removed)
- `internal/apiserver/routes.go` — Remove dead `goroutineID()` (~10 LOC removed)
- `internal/embed/embed.go` — Use shared goid (~5 LOC changed)
- `go.mod` / `go.sum` — Add `goid` dependency (if external package)
- `changelogs/v0.11-current.md` — Changelog entry

---

## Examples

### Example 1: CLI execution (fast path)

**Before:**
```
ailang run program.ail
  → Every builtin call:
    → runtime.Stack() → parse string → sync.Map.Load()
    → ~500ns overhead per call
    → 10,000 calls = 5ms wasted (small program)
    → 100,000 calls = 50ms wasted (medium program)
    → 1,000,000 calls = 500ms wasted (docparse)
```

**After:**
```
ailang run program.ail
  → Every builtin call:
    → atomic.Load() == 0 → use shared evaluator directly
    → ~1ns overhead per call
    → Effectively zero overhead at any scale
```

### Example 2: serve-api (slow path, still faster)

**Before:**
```
serve-api with concurrent requests:
  → runtime.Stack() per builtin call → ~500ns
```

**After:**
```
serve-api with concurrent requests:
  → atomic.Load() > 0 → goid.Get() + sync.Map.Load()
  → ~5ns (250x faster than runtime.Stack)
```

---

## Success Criteria

- [ ] `runtime.Stack` / `traceback` absent from top-20 CPU profile entries
- [ ] Docparse Alice EPUB ≤ 8s (evaluator mode) — down from 12s
- [ ] Docparse Moby Dick EPUB shows proportional improvement
- [ ] `make test` passes (all existing tests)
- [ ] `serve-api` concurrency tests pass (per-request isolation preserved)
- [ ] All 4 `goroutineID` implementations unified or removed
- [ ] Documentation updated (changelog)

---

## Testing Strategy

**Unit tests:**
- Verify `getEffContext()` returns correct evaluator in single-goroutine mode
- Verify `getEffContext()` returns forked evaluator after `SetGoroutineEvaluator`
- Verify fast-path activates when `goroutineEvalCount == 0`

**Integration tests:**
- Existing `make test` suite (exercises all builtin paths)
- `serve-api` concurrency test (multiple concurrent requests with different capabilities)

**Benchmark tests:**
- CPU profile before/after showing `runtime.Stack` elimination
- Docparse Alice EPUB wall-clock comparison
- Docparse Moby Dick EPUB wall-clock comparison

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Whether to use `goid` package or inline assembly** — agent may choose based on build complexity. Prefer the package for portability unless it introduces problems.
- **Whether to extract `getEffContextFromEvaluator` helper** — agent may refactor for readability
- **Exact atomic counter type** — `atomic.Int64` vs `atomic.Int32` — agent may choose

---

## Non-Goals

**Not attempted in this feature:**
- Changing builtin function signatures to thread `EffContext` (Option D) — too large an API change for a perf fix
- Optimizing the evaluator tree-walking itself (35% of profile) — separate effort (M-PERF4 bytecode interpreter)
- Reducing GC pressure (12% of profile) — separate effort
- Wiring remaining 57 effectful builtins to VM — separate sprints

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `goid` package relies on Go runtime internals that may change | Med | Package is well-maintained (used by CockroachDB, etc.); pin version; CI catches breakage on Go upgrades |
| Fast-path incorrectly activates during concurrent serve-api | High | Atomic counter is incremented in `SetGoroutineEvaluator` BEFORE any builtin call; decremented in `ClearGoroutineEvaluator` AFTER all calls complete; race-free by construction |
| Profile improvement doesn't translate to wall-clock improvement | Low | Unlikely — 42% of CPU is a massive fraction; even 50% elimination = measurable |

---

## Process Improvement: Sprint Evaluator Profiling

**This bottleneck should have been caught earlier.** The sprint evaluator scores quality after implementation, but currently only checks:
1. EvalOnly count reductions (structural metric)
2. Wall-clock time (coarse metric)
3. Test pass rates

None of these ask **where** CPU time is spent. A sprint can score well by reducing EvalOnly counts while delivering zero actual speedup.

**Add a CPU profiling check to the sprint-evaluator skill for perf-related sprints:**
```bash
# Before scoring a perf sprint, run:
ailang run -cpuprofile /tmp/bench.prof -caps IO,FS,Env <benchmark_file>
go tool pprof -top -cum /tmp/bench.prof | head -20
# Score down if: targeted function unchanged in profile, or new hotspot introduced
```

This would have caught that XML builtin wiring (17 builtins, 0% speedup) was addressing the wrong bottleneck.

---

## Related Documents

**Implemented (informs design):**
- [m-serve-api-concurrency.md](design_docs/implemented/v0_9_4/m-serve-api-concurrency.md) — Introduced the goroutineEvals sync.Map pattern
- [m-perf3-performance-quick-wins.md](design_docs/implemented/v0_8_1/m-perf3-performance-quick-wins.md) — Prior performance optimization pass
- [m-bytecode-vm.md](design_docs/implemented/v0_11_0/m-bytecode-vm.md) — VM architecture (bypasses this bottleneck for pure builtins)

**Planned (related):**
- [m-perf4-bytecode-interpreter.md](design_docs/planned/v1_0_0/m-perf4-bytecode-interpreter.md) — Full bytecode interpreter (addresses the 35% evaluator overhead)
- [m-std-string-perf.md](design_docs/planned/v0_11_0/m-std-string-perf.md) — String performance improvements

---

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [github.com/petermattis/goid](https://github.com/petermattis/goid) — Go goroutine ID package (used by CockroachDB)
- CPU profile data: `/tmp/alice.prof` (generated 2026-04-09)
- [Go issue #6580](https://github.com/golang/go/issues/6580) — Why Go doesn't expose goroutine IDs

---

## Future Work

- **Option D (context threading)**: Long-term, threading `EffContext` through the call chain eliminates the need for goroutine ID entirely. This is the cleanest solution but requires a major API change affecting ~100+ builtin functions. Should be considered for v1.0.0.
- **Sprint executor profiling**: Add CPU profiling as a standard step in performance-related sprints to catch this class of issue earlier.

---

**Document created**: 2026-04-09
**Last updated**: 2026-04-09
