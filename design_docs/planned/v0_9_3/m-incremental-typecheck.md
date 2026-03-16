# M-INCREMENTAL-TYPECHECK: Cached Compilation for Repeat Invocations

**Status**: Planned
**Target**: v0.9.3
**Priority**: P1 (Medium)
**Estimated**: 3-4 days (~24 hours implementation + testing)
**Dependencies**: None (builds on existing pipeline infrastructure)

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
| A7: Machines First | +1 | Reduces AI agent iteration latency from seconds to milliseconds |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Compilation cost reduced — visible via `--debug-compile` timing |
| A10: Composability | 0 | Cache is transparent to module composition |
| A11: Structured Failure | 0 | Cache miss falls through to normal compilation with same errors |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Cache keyed on SHA-256 of source content — deterministic by construction
- [x] A3 (Effects): No hidden side effects — cache is pure memoization
- [x] A4 (Authority): No ambient access — cache stored in project-local `.ailang/cache/`
- [x] A7 (Machines First): Directly improves AI agent iteration speed

## Problem Statement

Every invocation of `ailang run` re-parses, re-elaborates, re-type-checks, and re-monomorphizes **all** source files and their transitive dependencies from scratch. For small programs this is negligible, but for larger projects the overhead becomes the dominant cost.

**Current State (measured v0.9.1):**

| Program | Lines | Imports | Compile Time | Total Wall Time |
|---------|-------|---------|-------------|-----------------|
| hello_world.ail | 3 | std/io | 3ms | ~400ms |
| inline_tests.ail | 248 | none | 3ms | ~400ms |
| invoice.ail | 197 | std/option, std/string, std/list | 10ms | ~410ms |
| docparse eval.ail | 492 | many modules + stdlib | ~2-3s (reported) | ~2-3s |

Key observations:
- **Go binary startup: ~370ms** — fixed cost on every invocation regardless of program size
- **Stdlib re-compilation**: std/list (25 functions), std/string, std/option all re-compiled per invocation
- **Pipeline phases are fast individually** but the sum across many modules adds up

**Impact:**
- **docparse agent**: Reports 2-3s startup overhead per invocation, making rapid iteration painful
- **eval suite**: Each benchmark invocation pays full compilation cost
- **AI agents**: Latency-sensitive — extra seconds per invocation compound across multi-turn sessions

## Goals

**Primary Goal:** Reduce repeat-invocation compilation time to <100ms for unchanged programs by caching compilation artifacts.

**Success Metrics:**
- Unchanged program re-run: <100ms total (vs ~2-3s baseline for docparse)
- Single-file edit re-run: only changed file + dependents re-compiled
- Cache hit rate >95% for typical edit-run cycles
- Zero correctness regressions — cache invalidation must be sound

## Solution Design

### Overview

Introduce a **content-addressed compilation cache** that stores per-module compilation artifacts (elaborated Core, type environment, monomorphized output) keyed by source content hash + dependency hashes. On subsequent runs, unchanged modules skip all compilation phases and load cached artifacts directly.

### Architecture

**Components:**

1. **Content Hasher** (`internal/pipeline/cache_key.go`): Compute SHA-256 of source content + compiler version + dependency interface hashes to produce a stable cache key per module.

2. **Artifact Store** (`internal/pipeline/cache_store.go`): Read/write compiled module artifacts (serialized Core AST, TypeEnv, CoreTypeInfo, interface) to `.ailang/cache/compile/`. Uses gob encoding for Go struct serialization.

3. **Cache-Aware Module Pipeline** (modify `pipeline_module.go`): Before compiling each module in topo order, check cache. On hit, load artifacts and skip elaborate→typecheck→monomorph→lower. On miss, compile normally and write artifacts.

### Cache Key Design

```
CacheKey = SHA256(
    compiler_version,          // Invalidate on ailang upgrade
    source_content_hash,       // Invalidate on source edit
    [dep_interface_hash...],   // Invalidate if any dependency changes
    pipeline_config_hash       // Invalidate if compilation flags change
)
```

This is **sound by construction**: any change to source, dependencies, or compiler produces a different key, forcing recompilation.

### Artifact Format

Per-module cache entry stored as `.ailang/cache/compile/<hex-key>.gob`:

```go
type CachedModule struct {
    Version     string           // Compiler version that produced this
    Key         [32]byte         // Cache key for verification
    CoreAST     core.CoreExpr    // Elaborated + lowered Core
    TypeEnv     *types.TypeEnv   // Accumulated type environment
    CoreTypeInfo *types.CoreTypeInfo // Per-node type annotations
    Iface       *iface.Iface    // Module interface for dependents
    Timestamp   time.Time        // For cache eviction
}
```

### Implementation Plan

**Phase 1: Content Hashing + Cache Store** (~8 hours)
- [ ] Create `internal/pipeline/cache_key.go` — SHA-256 key computation
- [ ] Create `internal/pipeline/cache_store.go` — gob serialize/deserialize, file I/O
- [ ] Add `--no-cache` flag to disable caching
- [ ] Add cache directory management (`.ailang/cache/compile/`)
- [ ] Unit tests for key computation and round-trip serialization

**Phase 2: Cache-Aware Pipeline** (~8 hours)
- [ ] Modify `pipeline_module.go` `compileModule()` to check cache before compilation
- [ ] On cache hit: load `CachedModule`, populate pipeline state, skip compilation phases
- [ ] On cache miss: compile normally, write `CachedModule` to cache after success
- [ ] Add cache hit/miss reporting to `--debug-compile` output
- [ ] Integration tests: verify cache produces identical results to uncached

**Phase 3: Correctness Validation + Stdlib Pre-cache** (~8 hours)
- [ ] Run full test suite with caching enabled — verify zero regressions
- [ ] Run eval suite to verify cached vs uncached produce identical outputs
- [ ] Add `ailang cache clear` CLI command
- [ ] Add `ailang cache stats` CLI command (hit rate, size, entries)
- [ ] Benchmark: measure docparse eval.ail with warm cache
- [ ] Documentation and CHANGELOG

### Files to Modify/Create

**New files:**
- `internal/pipeline/cache_key.go` — Cache key computation (~100 LOC)
- `internal/pipeline/cache_store.go` — Artifact serialization + file store (~200 LOC)
- `internal/pipeline/cache_key_test.go` — Key computation tests (~80 LOC)
- `internal/pipeline/cache_store_test.go` — Round-trip serialization tests (~120 LOC)
- `cmd/ailang/cache.go` — `ailang cache clear|stats` commands (~80 LOC)

**Modified files:**
- `internal/pipeline/pipeline_module.go` — Cache check/write around compilation (~50 LOC delta)
- `internal/pipeline/pipeline.go` — Wire cache store into pipeline config (~20 LOC delta)
- `cmd/ailang/run.go` — Add `--no-cache` flag (~10 LOC delta)
- `cmd/ailang/root.go` — Register cache subcommand (~5 LOC delta)

## Examples

### Example 1: Repeat Invocation (No Changes)

**Before:**
```
$ time ailang run --caps IO,FS docparse/eval.ail
...output...
real 0m2.4s   # Full compilation every time
```

**After:**
```
$ time ailang run --caps IO,FS docparse/eval.ail
...output...
real 0m2.4s   # First run: full compilation + cache write

$ time ailang run --caps IO,FS docparse/eval.ail
...output...
real 0m0.09s  # Second run: cache hit, skip compilation
```

### Example 2: Debug Compile Output with Cache

```
$ ailang run --debug-compile --caps IO file.ail
[CACHE] std/option: HIT (compiled 2m ago)
[CACHE] std/string: HIT (compiled 2m ago)
[CACHE] std/list: HIT (compiled 2m ago)
[CACHE] mymodule: MISS (source changed)
[DEBUG] Dictionary elaboration complete for module mymodule
...
[METRICS] Phase Timings:
[METRICS]   Load:           1ms
[METRICS]   CacheLoad:      5ms  (3 modules from cache)
[METRICS]   Compile:        3ms  (1 module recompiled)
[METRICS]   Evaluate:      <1ms
[METRICS]   Total:          9ms
```

### Example 3: Cache Management

```
$ ailang cache stats
Compilation cache: .ailang/cache/compile/
  Entries: 47
  Size: 2.3 MB
  Hit rate (last session): 94.2%
  Oldest entry: 3 days ago

$ ailang cache clear
Cleared 47 cached entries (2.3 MB)
```

## Success Criteria

- [ ] Unchanged program re-run completes in <100ms (excluding Go binary startup)
- [ ] Single-file edit only recompiles changed module + dependents
- [ ] `make test` passes with caching enabled (default)
- [ ] `make test` passes with `--no-cache` (bypass)
- [ ] Cached and uncached compilation produce byte-identical evaluation results
- [ ] docparse eval.ail warm-cache benchmark under 500ms total wall time
- [ ] Cache invalidation is sound — no stale artifact bugs
- [ ] All tests passing
- [ ] Documentation updated
- [ ] CHANGELOG updated

## Testing Strategy

**Unit tests:**
- Cache key computation: same input → same key, different input → different key
- Artifact serialization: round-trip encode/decode for all Core AST node types
- Cache store: write, read, evict, clear operations

**Integration tests:**
- Compile program uncached, compile again cached — verify identical `Result`
- Edit one module, recompile — verify only that module + dependents recompile
- Change compiler version string — verify full cache invalidation
- Corrupt cache file — verify graceful fallback to full compilation

**Regression tests:**
- Full eval suite with caching enabled
- All example files with caching enabled
- `make verify-examples` with caching enabled

## Non-Goals

**Not in this feature:**
- **Persistent REPL caching** — REPL has different lifecycle; defer to future work
- **Parallel module compilation** — Topo sort is inherently sequential for dependencies; parallelism within independent modules is future work
- **Go binary startup reduction** — 370ms fixed cost is Go runtime overhead, not addressable here
- **Incremental within-module type-checking** — Would require tracking expression-level dependencies; too complex for v1
- **Shared/global cache** — Cache is project-local (`.ailang/cache/`); shared caches raise invalidation concerns

## Timeline

**Day 1-2** (~16 hours):
- Phase 1: Content hashing + cache store
- Phase 2: Cache-aware pipeline integration

**Day 3-4** (~8 hours):
- Phase 3: Correctness validation
- Benchmarking with docparse eval.ail
- CLI commands, documentation, CHANGELOG

**Total: ~24 hours across 3-4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Gob serialization doesn't support all Core AST types | High | Phase 1 tests catch this early; add custom encoders as needed |
| Cache invalidation unsound (stale artifacts) | High | Content-addressed keys make this structurally impossible; integration tests verify |
| Cache size grows unbounded | Low | Add LRU eviction (max 100MB default); `ailang cache clear` as escape hatch |
| Core AST types not gob-serializable (interfaces) | Med | Register concrete types with `gob.Register()`; test round-trip for every node type |
| Performance gain smaller than expected | Med | Measure early in Phase 2; if <2x improvement, investigate alternative serialization (protobuf) |

## Related Documents

<!-- Auto-populated by Ollama neural search on "incremental typecheck" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_3_18/M-DX4-SPRINT-PLAN-REFINED.md](design_docs/implemented/v0_3_18/M-DX4-SPRINT-PLAN-REFINED.md) — CoreTypeInfo completeness (type info we need to cache)
- [design_docs/implemented/v0_7_0/concat-operator-type-inference-bug.md](design_docs/implemented/v0_7_0/concat-operator-type-inference-bug.md) — Type inference edge cases to test with cache

**Planned (check for overlap):**
- [design_docs/planned/m-smt-cross-module-types.md](design_docs/planned/m-smt-cross-module-types.md) — Cross-module type resolution (cache must preserve cross-module interfaces)

## References

- [Design Axioms](/docs/references/axioms)
- `internal/pipeline/pipeline_module.go` — Current module compilation pipeline
- `internal/pipeline/specialize.go` — Existing monomorphization cache (per-run only)
- `internal/loader/loader.go` — Existing module loader cache (per-run only)
- docparse agent message (2026-03-16): "Cached/incremental type-checking (startup overhead still ~2-3s per invocation)"

## Future Work

- **Daemon mode**: Keep compiler resident in memory, eliminating Go binary startup (~370ms) and cache deserialization overhead
- **Parallel module compilation**: Compile independent modules concurrently within topo sort levels
- **Incremental within-module compilation**: Track per-declaration dependencies for finer-grained invalidation
- **Shared cache**: Cross-project cache for stdlib modules (all projects share same stdlib artifacts)
- **REPL incremental mode**: Cache REPL session state between lines

---

**Document created**: 2026-03-16
**Last updated**: 2026-03-16
