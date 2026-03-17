# M-PERF6: Compilation & Runtime Performance (Phase 1)

**Status**: Implemented
**Target**: v0.9.3
**Priority**: P1 (Medium)
**Estimated**: 3 days (~18 hours)
**Actual**: ~1 session (~2 hours)
**Actual LOC**: ~570 LOC (implementation + tests)
**Dependencies**: None
**Sprint ID**: M-PERF6

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Cache keyed on content hash — identical input always produces identical output |
| A2: Replayability | 0 | No impact on traces; compilation is pre-execution |
| A3: Effect Legibility | 0 | No change to effect system |
| A4: Explicit Authority | 0 | Cache is internal to compiler, no new capabilities |
| A5: Bounded Verification | +1 | Faster compilation enables more frequent verification cycles |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Reduces AI agent iteration latency |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Compilation cost visible via `--debug-compile` cache reporting |
| A10: Composability | 0 | Cache is transparent to module composition |
| A11: Structured Failure | 0 | Cache miss falls through to normal compilation |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

## Problem Statement

Cross-language benchmarking revealed AILANG interpreter is ~5x slower than Python for recursive workloads (fib(25): 260ms vs 48ms). The M-PERF3 design doc (v0.8.1) proposed quick wins but most were never implemented. Additionally, docparse agent reported 2-3s startup overhead per invocation.

**Profiling results (v0.9.1):**
- Go binary startup: ~370ms (fixed, not addressable)
- Pipeline compilation: 3-16ms depending on module count
- `exprProducesInterface()`: 486-line function called 37+ times per module without memoization
- `visited` map allocations: 6+ sites in types/ allocating new maps per call
- No persistent compilation cache between runs

## What Was Implemented

### M1: Codegen Memoization (~20 LOC)
- Added `interfaceCache map[core.CoreExpr]bool` to `Generator` struct
- Split `exprProducesInterface()` into cached wrapper + `exprProducesInterfaceUncached()`
- Cache cleared per function declaration (when `expectedReturnType`/`typedLocalVars` change)
- Commit: `cccdfc32`

### M2: Type System Pooling (~70 LOC)
- Created `internal/types/pool.go` with `sync.Pool` for 3 map types:
  - `map[Type]bool` — occurs check, free vars, type name propagation, safe string (5 sites)
  - `map[Type]Type` — substitution (1 site)
  - `map[typePair]bool` — equality (1 site)
- Maps cleared via range-delete on return to pool
- Verified with `go test -race` — no data races
- Commit: `b8b5436d`

### M3: Compilation Cache Infrastructure (~400 LOC)
- `internal/pipeline/cache_key.go` — `ModuleCacheKey()` computes SHA-256 from compiler version + source content hash + sorted dependency interface digests
- `internal/pipeline/cache_store.go` — `CacheStore` with JSON manifest at `.ailang/cache/compile/manifest.json`. Lookup, Store, Save, Clear, Stats operations
- `internal/pipeline/pipeline_module.go` — Cache integration in module compilation loop: compute key before compile, store entry after interface build, report hits/misses
- `internal/pipeline/pipeline.go` — Added `NoCache` config flag
- `cmd/ailang/cache_compile.go` — `ailang cache compile-clear` and `compile-stats` CLI commands
- `AILANG_NO_CACHE=1` environment variable to bypass cache
- Commit: `0515adda`

### M4: Benchmark + Documentation (~40 LOC)
- CHANGELOG updated in `changelogs/v0.9-current.md`
- Commit: `f6c587d1`

## Benchmark Results

**invoice.ail (197 lines, 4 modules: std/option, std/string, std/list, invoice):**

| Run | Cache | Compile Time | Modules |
|-----|-------|-------------|---------|
| Cold (first) | 4 MISS | 15ms | All compiled |
| Warm (second) | 4 HIT | 17ms | All compiled (skip not yet implemented) |

**Key insight:** Pipeline compilation is already fast (15ms for 4 modules). The cache correctly detects unchanged modules but does **not yet skip compilation** — that requires serializing `types.Type` and `core.Program` which contain Go interface types that can't be gob-encoded without extensive registration.

## Files Changed

**New files:**
- `internal/pipeline/cache_key.go` (~40 LOC)
- `internal/pipeline/cache_key_test.go` (~60 LOC)
- `internal/pipeline/cache_store.go` (~110 LOC)
- `internal/pipeline/cache_store_test.go` (~110 LOC)
- `internal/types/pool.go` (~65 LOC)
- `cmd/ailang/cache_compile.go` (~40 LOC)

**Modified files:**
- `internal/gen/golang/codegen.go` — Added `interfaceCache` field (+5 LOC)
- `internal/gen/golang/codegen_type_analysis.go` — Cache wrapper (+15 LOC)
- `internal/gen/golang/codegen_decl.go` — Cache clear on function boundary (+3 LOC)
- `internal/types/unification_substitution.go` — Pool usage (+2 LOC)
- `internal/types/unification_occurs.go` — Pool usage (+2 LOC)
- `internal/types/unification_equality.go` — Pool usage (+2 LOC)
- `internal/types/typechecker_defaulting.go` — Pool usage (+2 LOC)
- `internal/types/unification_core.go` — Pool usage (+3 LOC)
- `internal/types/safe_string.go` — Pool usage (+2 LOC)
- `internal/pipeline/pipeline.go` — `NoCache` config flag (+1 LOC)
- `internal/pipeline/pipeline_module.go` — Cache integration (~40 LOC)
- `cmd/ailang/main.go` — `NoCache` wiring (+1 LOC)
- `cmd/ailang/cache.go` — compile-clear/compile-stats routing (+4 LOC)

## What Was NOT Implemented (Deferred)

These items remain in the planned design doc for a future sprint:

1. **Compilation skip on cache hit** — Requires JSON serialization for `types.Type` (interface with ~15 concrete implementations) and `core.Program` (deep interface trees). This is the key feature that would reduce repeat-run compilation from 15ms to <1ms.
2. **M-PERF3 Phase 2: Evaluator improvements** — `GetAllBindings()` iterative refactor deprioritized (only called for introspection, not hot path). Copy-on-write environment deferred.
3. **M-PERF3 Phase 3: Codegen type specialization** — Complex, separate sprint.
4. **M-PERF4: Bytecode interpreter** — Major rewrite, deferred to v1.0+.

## Related Documents

- [M-PERF3: Performance Quick Wins](../v0_8_1/m-perf3-performance-quick-wins.md) — Original design doc (Phase 1 items now implemented)
- [M-PERF5: Data-Intensive Workloads](../../planned/v0_9_2/m-perf5-data-intensive-workloads.md) — Bulk XML/string ops (implemented)
- [M-INCREMENTAL-TYPECHECK](../../planned/v0_9_3/m-incremental-typecheck.md) — Remaining: compilation skip via artifact serialization

---

**Document created**: 2026-03-16
**Last updated**: 2026-03-16
