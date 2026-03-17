# Sprint Plan: M-PERF6 Compilation & Runtime Performance

## Summary
Implement remaining M-PERF3 quick wins (codegen memoization, type system pooling) and M-INCREMENTAL-TYPECHECK compilation caching to reduce both per-run overhead and repeat-invocation latency. Targets v0.9.3 pre-1.0.0 performance baseline.

**Duration:** 3 days (~18 hours)
**Dependencies:** None (all independent of current work)
**Risk Level:** Low (pure optimizations, no semantic changes)

## Current Status Analysis

### Completed Recently
- M-DOCPARSE-DX: 4 milestones in ~3 days (~700 LOC)
- M-BRAIN-VECTORS: 4 milestones in ~4 days (~1200 LOC)
- M-PERF5 (data-intensive): Bulk XML + string join = 34x DocParse improvement

### Velocity
- Recent average: ~200-300 LOC/day (implementation + tests)
- Estimated capacity: ~600 LOC for this 3-day sprint

### Performance Landscape

| Track | Status | Target |
|-------|--------|--------|
| M-PERF1 (effect checker) | Implemented v0.5.8 | N/A |
| M-PERF2 (operator lowering) | Implemented v0.5.8 | N/A |
| M-PERF3 Phase 1 (codegen memoization) | **Not implemented** | This sprint |
| M-PERF3 Phase 2 (evaluator) | **Not implemented** | Partial this sprint |
| M-PERF3 Phase 3 (codegen type specialization) | Not implemented | Deferred (complex) |
| M-PERF4 (bytecode interpreter) | Not implemented | Deferred to v1.0+ |
| M-PERF5 (data-intensive) | Implemented v0.9.2 | N/A |
| M-INCREMENTAL-TYPECHECK | **Not implemented** | This sprint |

### Key Findings from Profiling

- Go binary startup: ~370ms (fixed cost, not addressable)
- hello_world.ail compilation: 3ms (2 modules)
- invoice.ail compilation: 21ms (4 modules, 5.5MB allocs)
- `exprProducesInterface()`: 486-line file, called per-expression without memoization
- `GetAllBindings()`: Recursive but only called for introspection (NOT hot path) — **deprioritized**
- `visited` map allocations: 6+ locations in types/ allocating new maps per call
- No persistent compilation cache exists between runs

## Proposed Milestones

### Milestone 1: M1_CODEGEN_MEMOIZATION — Memoize Hot Codegen Paths
**Goal:** Add memoization cache for `exprProducesInterface()` and CoreTypeInfo lookups in codegen, eliminating redundant computation during Go code generation.
**Estimated:** ~80 LOC implementation + ~60 LOC tests = ~140 LOC
**Duration:** 0.5 day

**Tasks:**
- Add `interfaceCache map[core.CoreExpr]bool` field to `Generator` struct
- Check cache at top of `exprProducesInterface()`, write on miss
- Initialize cache in generator constructor, clear between modules
- Add benchmark test: codegen a 200+ line module, verify fewer map lookups
- Run `make test` and `make lint`

**Acceptance Criteria:**
- [ ] `exprProducesInterface()` results cached per codegen phase
- [ ] `make test` passes
- [ ] `make lint` clean
- [ ] Benchmark shows measurable reduction in codegen time for multi-module files

**Risks:**
- Cache key identity: CoreExpr pointer identity should work since expressions are not mutated during codegen. If not, fall back to node ID.

### Milestone 2: M2_TYPE_SYSTEM_POOLING — Pool Allocated Maps in Type System
**Goal:** Replace per-call `make(map[Type]bool)` and `make(map[Type]Type)` allocations with `sync.Pool` to reduce GC pressure during type checking.
**Estimated:** ~60 LOC implementation + ~40 LOC tests = ~100 LOC
**Duration:** 0.5 day

**Tasks:**
- Create `internal/types/pool.go` with `sync.Pool` for `map[Type]bool` and `map[Type]Type`
- Replace `visited := make(map[Type]bool)` in: `occursCheck`, `safeEquals`, `collectFreeVars`, `propagateTypeName`, `safeTypeString`
- Replace `visited := make(map[Type]Type)` in: `safeSubstitute`
- Clear maps on return to pool (range-delete pattern)
- Run `make test` with `-race` flag to verify no data races
- Run `make lint`

**Acceptance Criteria:**
- [ ] All `visited` map allocations in types/ use pool
- [ ] `make test -race` passes (no data races from pool reuse)
- [ ] `make lint` clean
- [ ] Allocation count reduced (verify with `AILANG_METRICS_VERBOSE=1`)

**Risks:**
- sync.Pool items may retain stale entries if not properly cleared. Mitigation: clear-on-return with explicit range-delete.
- Race conditions if type checking ever becomes concurrent. Mitigation: `-race` testing.

### Milestone 3: M3_COMPILATION_CACHE — Content-Addressed Module Cache
**Goal:** Cache compiled module artifacts (Core AST, TypeEnv, CoreTypeInfo, Iface) keyed on content hash so repeat invocations of unchanged programs skip recompilation.
**Estimated:** ~280 LOC implementation + ~120 LOC tests = ~400 LOC
**Duration:** 1.5 days

**Tasks:**
- Day 1 AM: Create `internal/pipeline/cache_key.go` — SHA-256 key from (compiler version + source hash + dep interface hashes + pipeline config)
- Day 1 AM: Create `internal/pipeline/cache_store.go` — gob serialize/deserialize `CachedModule` to `.ailang/cache/compile/`
- Day 1 PM: Unit tests for key computation + round-trip serialization of all Core AST node types
- Day 2 AM: Modify `pipeline_module.go` `compileModule()` — check cache before compile, write cache after successful compile
- Day 2 AM: Add `--no-cache` flag to run command
- Day 2 PM: Add `ailang cache clear` and `ailang cache stats` CLI commands
- Day 2 PM: Integration tests — verify cached == uncached for all example files

**Acceptance Criteria:**
- [ ] Unchanged program re-run skips compilation (cache hit logged with `--debug-compile`)
- [ ] Single-file edit only recompiles changed module + dependents
- [ ] `make test` passes with caching enabled
- [ ] `make verify-examples` passes with caching enabled
- [ ] `--no-cache` flag bypasses cache
- [ ] `ailang cache clear` and `ailang cache stats` work
- [ ] Cached and uncached produce identical evaluation results

**Risks:**
- Gob serialization may not handle all Core AST interface types — test round-trip early, add `gob.Register()` calls as needed
- Cache key must include ALL inputs that affect output — missing an input = stale cache bugs. Mitigation: content-addressed design makes this structurally sound.

### Milestone 4: M4_BENCHMARK_VERIFY — Benchmark and Validate
**Goal:** Run benchmarks before/after all optimizations, verify improvements, update CHANGELOG and design docs.
**Estimated:** ~50 LOC (benchmark script updates) + docs
**Duration:** 0.5 day

**Tasks:**
- Benchmark invoice.ail (cold + warm cache) with `AILANG_METRICS_VERBOSE=1`
- Benchmark largest available multi-module example (cold + warm)
- Run `make verify-examples` to confirm no regressions
- Run eval suite subset to confirm no behavioral changes
- Update CHANGELOG.md with results
- Update M-PERF3 design doc status (mark Phase 1 items as done)
- Update M-INCREMENTAL-TYPECHECK design doc with benchmark results

**Acceptance Criteria:**
- [ ] Warm-cache re-run <100ms compile time (vs 20ms+ cold)
- [ ] No test regressions
- [ ] CHANGELOG.md updated
- [ ] Design docs updated with actual results

## Success Metrics
- Codegen memoization: measurable improvement on multi-module compilation
- Type system pooling: reduced allocation count visible in metrics
- Compilation cache: <100ms compile time on warm cache
- All tests passing
- All linting clean
- CHANGELOG and design docs updated

## Dependencies
- None — all milestones are independent of external work
- M1 and M2 are independent of each other (can be done in any order)
- M3 is independent of M1/M2 (different part of pipeline)
- M4 depends on M1+M2+M3 completion

## Open Questions
- Should compilation cache be enabled by default or opt-in? Recommendation: **enabled by default** with `--no-cache` to disable.
- Cache eviction policy: LRU with 100MB cap? Or simple age-based (clear entries >7 days)?

## Notes
- `GetAllBindings()` iterative refactor **deprioritized** — only called for introspection, not on evaluation hot path
- M-PERF3 Phase 2 (evaluator env copy-on-write) **deferred** — M-PERF5 bulk ops already bypass most function call overhead for data workloads
- M-PERF3 Phase 3 (codegen type specialization) **deferred** — complex, better suited for dedicated sprint
- Go binary startup (~370ms) is a fixed Go runtime cost, not addressable in this sprint

---

**Sprint plan created**: 2026-03-16
