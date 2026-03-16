# Sprint Plan: M-ITERATIVE-LIST — Iterative List Builtins

## Summary

Replace recursive `map`, `filter`, `foldl` in `std/list` with Go-level iterative builtins, fixing both the **performance bottleneck** (5K rows = 60s+ due to evaluator frame overhead) and the **recursion depth limit** (10K+ rows = RT_REC_003). Small, well-scoped change — all architectural patterns proven. ~400 LOC, 1 day, low risk.

- **Design doc**: `design_docs/planned/v0_9_3/m-iterative-list-builtins.md`
- **Duration**: 1 day (~7 hours)
- **Dependencies**: M-PERF6 (completed), FnCaller pattern (established)
- **Risk**: Low — follows existing builtin + FnCaller patterns exactly

## Current Status Analysis

### Completed Recently
- M-PERF6: Compilation performance (4 milestones, content-addressed cache)
- M-DOCPARSE-DX: Set operations, XML serialization, list builtins (member, dedup, etc.)
- FnCaller pattern established for Stream event handlers (M-STREAM-BIDI)

### Velocity
- M-PERF6: ~2000 LOC across 4 milestones in 2 days (~1000 LOC/day)
- M-DOCPARSE-DX: ~1500 LOC across 4 milestones in 2 days (~750 LOC/day)
- Conservative estimate for this sprint: 450 LOC/day capacity
- This sprint: ~380 LOC total — comfortably within 1 day

### Existing Infrastructure (reuse targets)
- `internal/builtins/list.go` — 10 list builtins registered, pattern proven (~640 LOC)
- `internal/effects/context.go:52` — `FnCaller` field, wired in main.go:700
- `internal/eval/eval_evaluator.go:254` — `CallValue` single-arg, `CallFunction` multi-arg
- `std/list.ail:336-348` — Delegation pattern (`member = _list_member`, etc.)
- `cmd/ailang/main.go:700` — `effCtx.FnCaller = evaluator.CallValue` wiring

## Proposed Milestones

### Milestone 1: M1_FNCALLERN — Add FnCallerN to EffContext + Evaluator

**Goal**: Enable Go builtins to call AILANG functions with multiple arguments.
**Estimated**: ~25 LOC implementation

**Tasks**:
1. Add `FnCallerN func(fn eval.Value, args []eval.Value) (eval.Value, error)` field to `EffContext` in `internal/effects/context.go`
2. Propagate `FnCallerN` in `WithBudget()` method (same as `FnCaller`)
3. Add `CallValueN(fn Value, args []Value) (Value, error)` method to `CoreEvaluator` in `internal/eval/eval_evaluator.go`
4. Wire `effCtx.FnCallerN = evaluator.CallValueN` in `cmd/ailang/main.go` and `cmd/ailang/serve_api.go`

**Acceptance Criteria**:
- [ ] `FnCallerN` field exists on `EffContext`
- [ ] `FnCallerN` propagated in `WithBudget()`
- [ ] `CallValueN` handles `FunctionValue`, `BuiltinFunction`, and error cases
- [ ] Wired in both `main.go` and `serve_api.go`
- [ ] `make test` passes (no regressions)
- [ ] `make lint` clean

**Risks**: None — additive change, no existing code modified.

### Milestone 2: M2_ITERATIVE_BUILTINS — Implement Go-level map, filter, foldl

**Goal**: Create iterative Go builtins that replace recursive AILANG implementations.
**Estimated**: ~150 LOC implementation + ~200 LOC tests

**Tasks**:
1. Create `internal/builtins/list_iterative.go` with:
   - `_list_map`: iterative map using `FnCaller` (1-arg callback)
   - `_list_filter`: iterative filter using `FnCaller` (1-arg callback)
   - `_list_foldl`: iterative foldl using `FnCallerN` (2-arg callback)
   - Registration via `RegisterEffectBuiltin` with proper type signatures
2. Create `internal/builtins/list_iterative_test.go` with:
   - Functional correctness tests (identity, transform, empty list, single element)
   - Type error tests (non-function, non-list)
   - Callback error propagation tests
   - 50K element stress tests for each builtin
   - `FnCallerN` nil check test for foldl

**Acceptance Criteria**:
- [ ] `_list_map`, `_list_filter`, `_list_foldl` registered as builtins
- [ ] All unit tests pass including 50K stress tests
- [ ] Callback errors propagate correctly
- [ ] Type mismatches produce clear error messages
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks**: Type signature construction — mitigate by copying patterns from existing list builtins.

### Milestone 3: M3_STDLIB_MIGRATION — Wire std/list.ail + verify

**Goal**: Migrate `std/list.ail` to delegate to Go builtins and verify everything works.
**Estimated**: ~10 LOC changes + verification

**Tasks**:
1. Update `std/list.ail`:
   - `map` → `export pure func map[a, b](f: (a) -> b, xs: [a]) -> [b] = _list_map(f, xs)`
   - `filter` → `export pure func filter[a](p: (a) -> bool, xs: [a]) -> [a] = _list_filter(p, xs)`
   - `foldl` → `export pure func foldl[a, b](f: (b, a) -> b, acc: b, xs: [a]) -> b = _list_foldl(f, acc, xs)`
2. Update golden snapshot (builtin count)
3. Run `make test` — verify all existing list tests pass
4. Run `make verify-examples` — verify all examples still work

**Acceptance Criteria**:
- [ ] `std/list.ail` delegates `map`, `filter`, `foldl` to Go builtins
- [ ] Golden snapshot updated with new builtin count
- [ ] All existing tests pass (`make test`)
- [ ] All examples verify (`make verify-examples`)
- [ ] `make lint` clean

**Risks**: Type unification between Go builtin signatures and AILANG type inference — mitigate by matching existing delegation pattern exactly.

### Milestone 4: M4_BENCHMARK_VERIFY — Performance benchmarks + docs

**Goal**: Verify performance improvement with benchmarks, update docs.
**Estimated**: ~20 LOC benchmark + docs

**Tasks**:
1. Add Go benchmark in `list_iterative_test.go`:
   - `BenchmarkListMap50K` — map over 50K elements
   - `BenchmarkListFoldl50K` — foldl over 50K elements
2. Run benchmarks, record results
3. If possible, compare with recursive version timing (5K row DocParse scenario)
4. Update CHANGELOG.md with performance results
5. Update design doc status

**Acceptance Criteria**:
- [ ] Benchmarks show <1s for 50K element map/foldl
- [ ] CHANGELOG.md updated with benchmark results
- [ ] Design doc moved to `implemented/v0_9_3/` or status updated
- [ ] `make test` passes
- [ ] `make lint` clean

## Day-by-Day Breakdown

### Day 1 (~7 hours)

- **Morning** (~1.5h): M1_FNCALLERN — Add FnCallerN field, CallValueN method, wire in main/serve-api
- **Midday** (~3h): M2_ITERATIVE_BUILTINS — Implement 3 builtins + comprehensive tests
- **Afternoon** (~1.5h): M3_STDLIB_MIGRATION — Wire std/list.ail, verify all tests/examples
- **Late afternoon** (~1h): M4_BENCHMARK_VERIFY — Run benchmarks, update CHANGELOG + design doc

## Success Metrics

- `map` over 50K elements: completes in <1s (vs evaluator-bound recursive version)
- `foldl` over 50K elements: completes without RT_REC_003 error
- DocParse-scale workloads (5K rows x 3 cells) dramatically faster
- Zero test regressions (`make test` green)
- Zero example regressions (`make verify-examples` green)
- Lint clean (`make lint`)
- Builtin count increased by 3

## Dependencies

- M-PERF6 `parseElements` (completed) — produces the large lists
- `FnCaller` pattern (established) — provides the callback architecture
- `CallFunction` multi-arg (exists) — `CallValueN` wraps it

## Open Questions

None — all architectural decisions resolved during feasibility audit.

## Notes

- `forEach` dropped from scope: no pure `forEach` exists in std/list.ail; only effectful `forEachE`
- Effectful variants (`mapE`, `foldlE`, `forEachE`) deferred — need effect context threading
- `FnCallerN` will be reused by future builtins (sortBy, groupBy, etc.)
