# Sprint Plan: M-EVAL-BOUNDED-PIPELINE

## Summary

Add fused bounded combinators (`takeFlatMap`, `takeMap`) and a `--max-memory` flag to fix OOM on large-document eval pipelines. Includes compiler diagnostic note for unfused `take(N, flatMap(...))` patterns.

**Duration:** 1 day (estimated 3-4 hours active implementation)
**Dependencies:** None
**Risk Level:** Low — new builtins only, no changes to existing list semantics
**Design Doc:** `design_docs/planned/v0_9_4/m-eval-bounded-pipeline.md`

## Current Status Analysis

### Completed Recently
- M-CONTRACT-PIPELINE-DX: Quick wins, ~200 LOC in <1 day
- M-SERVE-API-CONCURRENCY: Per-request evaluator, ~200 LOC in <1 day
- M-CONTRACT-OPLOWERING-FIX: Complete OpLowering, ~400 LOC in <1 day
- M-HASH-COLLECTIONS Phase 1: O(n) set ops, ~350 LOC in 1 day

### Velocity
- Recent average: ~300-400 LOC/day (implementation + tests)
- This sprint estimated: ~270 LOC total — well within single-day capacity

### Codebase State
- `list_iterative.go` — established pattern for iterative builtins with `ctx.FnCaller` callbacks
- `RegisterEffectBuiltin` in `spec.go` — standard registration API
- `types.NewBuilder()` — type builder DSL for builtin signatures
- `cmd/ailang/main.go` — flag parsing infrastructure (follow `--timeout` pattern)

## Proposed Milestones

### Milestone 1: Fused Bounded Combinators
**Goal:** Add `takeFlatMap(n, f, xs)` and `takeMap(n, f, xs)` builtins that short-circuit after N results
**Estimated:** ~80 LOC implementation + ~120 LOC tests = ~200 LOC
**Duration:** ~2 hours

**Tasks:**
1. Create `internal/builtins/list_bounded.go`:
   - `registerTakeFlatMap()` with `RegisterEffectBuiltin` — follow `list_iterative.go` pattern
   - `makeTakeFlatMapType()` using `types.NewBuilder()`: `(Int, (a -> [b]), [a]) -> [b]`
   - `takeFlatMapImpl()` — iterate `xs`, call `ctx.FnCaller(f, x)`, append to result, break when `len(result) >= n`
   - `registerTakeMap()` — same pattern, simpler (1:1 mapping, no inner list)
   - `makeMapTakeType()`: `(Int, (a -> b), [a]) -> [b]`
   - `takeMapImpl()` — iterate `xs`, call `ctx.FnCaller(f, x)`, break when `len(result) >= n`
   - Both with `BuiltinMetadata` docs
2. Create `internal/builtins/list_bounded_test.go`:
   - `takeFlatMap` with pure function — correct results and count
   - `takeFlatMap` where N > total — returns all elements
   - `takeFlatMap` with N=0 — returns empty list
   - `takeFlatMap` verifying early exit (count function calls)
   - Same coverage for `takeMap`
   - Edge cases: empty input list, N=1

**Acceptance Criteria:**
- [ ] `takeFlatMap(3, \x. [x, x], [1,2,3,4,5])` returns `[1,1,2]`
- [ ] `takeMap(2, \x. x*2, [1,2,3,4,5])` returns `[2,4]`
- [ ] Early exit verified: `takeFlatMap(3, f, longList)` calls `f` only ~2 times, not all
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- `ctx.FnCaller` nil check — Mitigation: follow `list_iterative.go` nil guard pattern

### Milestone 2: Memory Ceiling
**Goal:** Add `--max-memory` CLI flag that sets Go's `debug.SetMemoryLimit` for clean OOM errors
**Estimated:** ~30 LOC implementation + ~20 LOC tests = ~50 LOC
**Duration:** ~30 min

**Tasks:**
1. Add `--max-memory` flag to `cmd/ailang/main.go` (follow `--timeout` pattern)
2. Create `internal/eval/memory_limit.go`:
   - `ParseMemorySize(s string) (int64, error)` — parse "512MB", "1GB", etc.
   - `SetMemoryLimit(limit string) error` — calls `runtime/debug.SetMemoryLimit`
3. Wire flag into `run` and `serve-api` commands
4. Test: `ParseMemorySize` unit tests for valid/invalid input

**Acceptance Criteria:**
- [ ] `ailang run --max-memory 512MB file.ail` sets memory limit
- [ ] `ailang run --max-memory invalid` prints clean error with examples
- [ ] ParseMemorySize handles MB, GB, and plain bytes
- [ ] `make test` passes

**Risks:**
- `debug.SetMemoryLimit` requires Go 1.19+ — Mitigation: already on Go 1.22+

### Milestone 3: Compiler Diagnostic Note
**Goal:** Emit a note when compiler sees `take(N, flatMap(f, xs))` suggesting `takeFlatMap`
**Estimated:** ~20 LOC
**Duration:** ~30 min

**Tasks:**
1. Find the pipeline/elaboration phase where `take` applied to `flatMap` result is visible
2. Add pattern match: if application is `take(N, flatMap(f, xs))`, emit diagnostic note
3. Use existing `errors/report.go` infrastructure or simple stderr note

**Acceptance Criteria:**
- [ ] `take(5, flatMap(f, xs))` emits note suggesting `takeFlatMap`
- [ ] Note is informational (does not prevent compilation)
- [ ] `take(5, someList)` does NOT emit note (only when wrapping flatMap)

**Risks:**
- AST structure may make pattern detection non-trivial — Mitigation: if too complex, defer to M2 and document as known limitation. The builtins themselves are the priority.

## Success Metrics
- DocParse Moby Dick eval pipeline pattern no longer requires full materialization
- `takeFlatMap` and `takeMap` registered and callable from AILANG
- `--max-memory` provides clean failure instead of OS OOM
- All existing tests pass — zero regressions
- `make lint` clean
- `make verify-examples` passes

## Implementation Order

```
M1: list_bounded.go + tests  →  M2: --max-memory flag  →  M3: compiler note
         (~2h)                        (~30min)                  (~30min)
```

M1 is the critical path. M2 and M3 are independent of each other but both depend on M1 being done first (for verification flow).
