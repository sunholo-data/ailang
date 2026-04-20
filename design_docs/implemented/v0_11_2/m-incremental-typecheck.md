# M-INCREMENTAL-TYPECHECK: Compilation Skip via Artifact Serialization

**Status**: Implemented (2026-04-10)
**Target**: v0.10.x
**Priority**: P2 (Medium)
**Estimated**: 2-3 days (completed in 1 day)
**Dependencies**: M-PERF6 Phase 1 (implemented), types.Type JSON marshaling (implemented)
**Commits**: 7c4ff925 (M1), d96de92f (M2), 4f91d27e (M3)

## What Already Exists (from M-PERF6)

The cache **infrastructure** is fully implemented:
- Content-addressed cache keys: `SHA-256(compiler_version + source_hash + dep_iface_digests)`
- Disk-persisted manifest at `.ailang/cache/compile/manifest.json`
- Per-module hit/miss detection and reporting via `--debug-compile`
- `AILANG_NO_CACHE=1` to bypass, `ailang cache compile-clear/compile-stats` CLI
- Iface JSON stored per cache entry (via existing `ToNormalizedJSON()`)

**What's missing:** On cache hit, the pipeline still recompiles because it can't deserialize the compiled artifacts (`core.Program`, `types.CoreTypeInfo`, `types.TypeEnv`) from disk.

## Problem Statement

The cache correctly identifies unchanged modules (4/4 HIT on repeat run of invoice.ail) but cannot skip compilation because `types.Type` is a Go interface with ~15 concrete implementations (`TCon`, `TVar2`, `TFunc2`, `TList`, `TRecord`, `TRecord2`, `TApp`, `Row`, `RowVar`, etc.) that have no JSON marshal/unmarshal support and can't be gob-encoded without registering every concrete type.

**Current state (measured v0.9.3, simple programs):**

| Run | Cache | Compile Time | Notes |
|-----|-------|-------------|-------|
| Cold | 4 MISS | 15ms | Full compilation + cache write |
| Warm | 4 HIT | 17ms | Full compilation (skip not implemented) |
| Target | 4 HIT | <1ms | Load cached artifacts, skip compilation |

**Docparse benchmark (measured v0.10.15, after M-PERF-DOCPARSE):**

| Metric | Value | Notes |
|--------|-------|-------|
| Total modules | 58 | std/*, pkg/*, docparse/* |
| Cache HITs | 27 | All still recompile — skip not implemented |
| Cache MISSes | 30 | Source/dep changes |
| Type checking CPU | 30% (~500ms) | After deferred CoreTI substitution optimization |
| Alice EPUB batch | 1,670ms | Target: ≤1,000ms with this feature |
| Moby Dick batch | 2,350ms | Target: ≤1,800ms |

Skipping compilation for the 27 cache-hit modules would save ~230ms (27/58 × 500ms),
bringing Alice EPUB to ~1.4s. Combined with skipping elaboration (~100ms saved),
target of ≤1.0s may be achievable.

## Goals

**Primary Goal:** On cache hit, skip elaborate→typecheck→monomorph→lower and load cached `CompileUnit` from disk.

**Success Metrics:**
- Warm-cache compilation: <1ms per cached module (vs 15ms currently)
- Docparse eval.ail repeat invocation: <500ms total wall time (Go startup + cache load)
- Zero correctness regressions

## Solution Design

### Phase 1: types.Type JSON Serialization (~1 day)

Add `MarshalJSON`/`UnmarshalJSON` to all `types.Type` implementations using a tagged-union pattern:

```go
type TypeJSON struct {
    Kind   string          `json:"kind"`   // "tcon", "tvar2", "tfunc2", "tlist", etc.
    Data   json.RawMessage `json:"data"`
}

// Example for TCon:
func (t *TCon) MarshalJSON() ([]byte, error) {
    return json.Marshal(TypeJSON{
        Kind: "tcon",
        Data: mustMarshal(struct{ Name string }{t.Name}),
    })
}
```

**Types to implement (15):**
`TCon`, `TVar`, `TVar2`, `RowVar`, `TFunc`, `TFunc2`, `TList`, `TArray`, `TTuple`, `TRecord`, `TRecordOpen`, `TRecord2`, `Row`, `TApp`, `Qualifier`

### Phase 2: CompileUnit Serialization (~0.5 day)

With types.Type JSON support, serialize the full `CachedModule`:

```go
type CachedModule struct {
    Core     *core.Program       `json:"core"`
    CoreTI   types.CoreTypeInfo  `json:"core_type_info"`
    Iface    *iface.Iface        `json:"iface"`
    TypeEnv  *types.TypeEnv      `json:"type_env"`  // only exported bindings
}
```

`core.Program` also contains `types.Type` in its AST nodes, so Phase 1 unblocks this.

### Phase 3: Pipeline Skip Integration (~0.5 day)

Modify `pipeline_module.go` compilation loop:

```go
if entry, ok := cacheStore.Lookup(modID, cacheKey); ok {
    // Cache hit — load cached artifacts instead of compiling
    cachedUnit, err := cacheStore.LoadArtifacts(modID)
    if err == nil {
        compiledUnits[modID] = cachedUnit
        continue // Skip elaborate→typecheck→monomorph→lower
    }
    // Fall through to normal compilation on load error
}
```

### Phase 4: Validation (~0.5 day)

- Run `make test` with and without cache
- Run `make verify-examples` with warm cache
- Verify cached and uncached produce identical eval results
- Benchmark docparse eval.ail

## Files to Modify/Create

**New files:**
- `internal/types/json.go` — MarshalJSON/UnmarshalJSON for all Type impls (~400 LOC)
- `internal/types/json_test.go` — Round-trip tests for all 15 types (~200 LOC)

**Modified files:**
- `internal/pipeline/cache_store.go` — Add `LoadArtifacts()`, `StoreArtifacts()` (~100 LOC)
- `internal/pipeline/pipeline_module.go` — Add skip logic on cache hit (~20 LOC)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| types.Type JSON round-trip lossy | High | Comprehensive test for all 15 types, structural equality check |
| core.Program has internal pointers | Med | Serialize by value, rebuild references on load |
| Cache grows unbounded | Low | LRU eviction (max 50MB), `ailang cache compile-clear` |
| Stale cache on compiler bug | Low | Cache key includes compiler version, auto-invalidates |

## Non-Goals

- **Persistent REPL caching** — Different lifecycle
- **Parallel module compilation** — Separate concern
- **Go binary startup reduction** — 370ms is Go runtime, not addressable
- **Incremental within-module compilation** — Expression-level deps too complex

## Related Documents

- [M-PERF6 Implementation](../../implemented/v0_9_2/m-perf6-compilation-performance.md) — Cache infrastructure (Phase 1)
- [M-PERF-DOCPARSE](m-perf-docparse.md) — Deferred CoreTI substitution + GOGC tuning (prerequisite, done)
- [M-PERF-GOROUTINE-ID](m-perf-goroutine-id.md) — Eliminated runtime.Stack() bottleneck (prerequisite, done)
- [M-PERF3: Performance Quick Wins](../../implemented/v0_8_1/m-perf3-performance-quick-wins.md) — Original perf design

## Future Work

- **Daemon mode**: Keep compiler resident, eliminating Go startup + cache deserialization
- **Shared stdlib cache**: All projects share cached stdlib compilation
- **Parallel module compilation**: Independent modules compile concurrently within topo sort levels

---

**Document created**: 2026-03-16
**Last updated**: 2026-04-10 (updated with M-PERF-DOCPARSE profiling data)
