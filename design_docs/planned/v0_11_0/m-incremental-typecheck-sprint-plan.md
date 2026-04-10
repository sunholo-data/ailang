# Sprint Plan: M-INCREMENTAL-TYPECHECK

## Summary
Skip compilation (elaborate→typecheck→monomorph→lower) for unchanged modules by serializing full compiled artifacts to disk. Cache infrastructure already exists (M-PERF6) with correct hit/miss detection — this sprint adds artifact serialization and pipeline skip logic.

**Duration:** 3 days
**Dependencies:** M-PERF6 Phase 1 (done), M-PERF-DOCPARSE (done)
**Risk Level:** Medium (JSON round-trip correctness is critical)

## Current Status Analysis

### Completed Recently
- ✅ M-PERF-DOCPARSE: 95 LOC in 1 day — deferred CoreTI substitution, GOGC tuning, leaf-type fast path
- ✅ M-PERF-GOROUTINE-ID: 3 milestones in 1 day — 2.5-3.6x speedup on goroutine ID extraction
- ✅ M-PERF6 Phase 1: Cache infrastructure — content-addressed keys, manifest, hit/miss detection

### Velocity
- Recent average: ~100-150 LOC/day (actual, post-estimation)
- Design doc estimate: 720 LOC / 2-3 days
- Sprint capacity: ~450 LOC over 3 days (conservative)

### What Exists vs What's Missing
- ✅ Cache key computation: `SHA-256(compiler_version + source_hash + dep_iface_digests)`
- ✅ Hit/miss detection: `cacheStore.Lookup(modID, cacheKey)` in pipeline_module.go
- ✅ Manifest persistence: `.ailang/cache/compile/manifest.json`
- ❌ **types.Type JSON marshal/unmarshal** — 14 concrete types, no serialization
- ❌ **Kind JSON marshal/unmarshal** — 4 concrete kinds (KStar, KRow, KEffect, KRecord)
- ❌ **core.CoreExpr JSON marshal/unmarshal** — 22 concrete node types
- ❌ **CachedModule struct** — stores Core AST + CoreTypeInfo + Iface + constructors
- ❌ **Pipeline skip logic** — load cached artifacts instead of compiling

## Proposed Milestones

### Milestone 1: types.Type + Kind JSON Serialization
**Goal:** Add tagged-union JSON marshal/unmarshal for all Type and Kind implementations, enabling disk persistence of type information.

**Estimated:** 300 LOC implementation + 150 LOC tests = 450 LOC

**Types to serialize (14):**
| Type | Fields | Complexity |
|------|--------|-----------|
| `TCon` | `Name string` | Simple |
| `TVar` | `Name string` | Simple |
| `TVar2` | `Name string, Kind Kind` | Medium (Kind is interface) |
| `RowVar` | `Name string, Kind Kind` | Medium |
| `TList` | `Element Type` | Recursive |
| `TArray` | `Element Type` | Recursive |
| `TMap` | `Key Type, Value Type` | Recursive |
| `TTuple` | `Elements []Type` | Recursive |
| `TFunc2` | `Params []Type, Return Type, EffectRow *Row` | Complex |
| `TRecord` | `Fields map[string]Type, Row Type, TypeName string` | Complex |
| `TRecordOpen` | `Fields map[string]Type, Row Type` | Complex |
| `TRecord2` | `Row *Row` | Medium |
| `TApp` | `Constructor Type, Args []Type` | Recursive |
| `Row` | `Kind Kind, Labels map[string]Type, Tail *RowVar, Budgets, MinBudgets` | Most complex |

**Kinds to serialize (4):** `KStar`, `KRow` (recursive), `KEffect`, `KRecord`

**Approach:** Tagged-union pattern with `{"kind": "tcon", "data": {...}}` envelope.

**Files:**
- **New:** `internal/types/json.go` (~300 LOC) — MarshalJSON/UnmarshalJSON + `MarshalType`/`UnmarshalType` dispatcher
- **New:** `internal/types/json_test.go` (~150 LOC) — Round-trip test for every type, including nested/recursive structures

**Acceptance Criteria:**
- All 14 Type implementations have MarshalJSON/UnmarshalJSON
- All 4 Kind implementations serialize correctly
- Round-trip test: `marshal(t) → unmarshal → marshal(t2)` produces identical JSON for all types
- Nested types (e.g., `TFunc2` with `TList[TRecord]` params) round-trip correctly
- Row with Tail (RowVar chain) round-trips correctly
- `make test` passes with no regressions

**Risks:**
- Row.Budgets/MinBudgets (`map[string]*int`) — nullable pointer values need careful handling. Mitigation: explicit nil checks in marshal, test with nil and non-nil values.
- Kind is an interface — needs its own tagged-union dispatch. Mitigation: only 4 implementations, small.

---

### Milestone 2: CompileUnit + CoreExpr Serialization
**Goal:** Serialize full compiled module state (Core AST, CoreTypeInfo, Iface) to disk. With Phase 1 providing Type serialization, this extends to CoreExpr nodes and the CachedModule struct.

**Estimated:** 200 LOC implementation + 100 LOC tests = 300 LOC

**Note:** core.CoreExpr has 22 implementations, but most share the same `CoreNode` base struct with `ID uint64` and `Span ast.Pos`. We can use Go's `encoding/gob` with registered types for CoreExpr (all in same package) rather than hand-writing JSON for 22 types. Only types.Type needs custom JSON because it crosses package boundaries.

**Approach:**
- `CachedModule` struct with gob-encoded Core program + JSON-encoded CoreTypeInfo
- CoreTypeInfo is `map[uint64]Type` — trivially serializable once Type has JSON support
- Store as `<modID>.cache` binary file alongside manifest

**Files:**
- **Modified:** `internal/pipeline/cache_store.go` (~120 LOC) — Add `StoreArtifacts(modID, CachedModule)` and `LoadArtifacts(modID) (CachedModule, error)`
- **New:** `internal/pipeline/cache_store_test.go` (~100 LOC) — Round-trip test for CachedModule with realistic program
- **Modified:** `internal/core/core.go` (~20 LOC) — Register all CoreExpr types for gob

**Acceptance Criteria:**
- CachedModule stores: core.Program, CoreTypeInfo, Iface, constructor schemes
- Round-trip test: compile a module → store → load → compare types match
- Gob registration covers all 22 CoreExpr types
- CoreTypeInfo entries survive serialization (uint64 keys, Type values)
- `make test` passes

**Risks:**
- core.CoreExpr may have unexported fields blocking gob. Mitigation: test early, fall back to tagged-union JSON if needed.
- Pointer identity lost after deserialization. Mitigation: types are compared structurally, not by pointer.

---

### Milestone 3: Pipeline Skip + Validation + Benchmarks
**Goal:** Wire cache hit → skip compilation in the module pipeline, validate correctness across all examples, and benchmark the speedup.

**Estimated:** 80 LOC implementation + benchmarks = 120 LOC

**Pipeline change (pipeline_module.go):**
```go
if entry, ok := cacheStore.Lookup(modID, cacheKey); ok {
    cached, err := cacheStore.LoadArtifacts(modID)
    if err == nil {
        compiledUnits[modID] = cached.ToCompileUnit()
        continue // Skip elaborate→typecheck→monomorph→lower
    }
    // Fall through on error — recompile
}
```

**Files:**
- **Modified:** `internal/pipeline/pipeline_module.go` (~30 LOC) — Cache hit skip logic
- **Modified:** `internal/pipeline/pipeline_module_compile.go` (~30 LOC) — Store artifacts after compilation
- **Modified:** `changelogs/v0.10-current.md` (~20 LOC) — Performance results

**Acceptance Criteria:**
- `make test` passes with and without cache (`AILANG_NO_CACHE=1`)
- `make verify-examples` passes with warm cache (all 107 runnable examples)
- Cached and uncached runs produce identical eval results for Alice EPUB
- `--debug-compile` shows "SKIP" instead of "HIT" for cached modules
- Warm-cache compilation: <1ms per cached module
- Alice EPUB batch: ≤1.0s (from 1.67s) — 27/58 modules skip type checking
- Moby Dick EPUB batch: ≤1.8s (from 2.35s)
- CHANGELOG updated with benchmark results
- Design doc status updated to "Implemented"

**Risks:**
- Stale cache producing wrong results. Mitigation: cache key includes compiler version + source hash + dep digests — deterministic invalidation. Run full example suite both ways.
- Performance target not met. Mitigation: even partial skip (e.g., 20/27 modules) is a win. Adjust targets based on measured data.

## Success Metrics
- All 14+4 Type/Kind implementations have lossless JSON round-trips
- Cache hit → compilation skip works for unchanged modules
- Alice EPUB batch ≤1.0s (from 1.67s, -40%)
- Zero correctness regressions across test suite + examples
- `make test` and `make verify-examples` pass

## Dependencies
- M-PERF6 Phase 1 cache infrastructure (done)
- M-PERF-DOCPARSE deferred substitution (done)

## Open Questions
- Row.Budgets/MinBudgets: Are these needed at runtime or only during type checking? If only during type checking, we can skip serializing them.
- TypeEnv: The design doc mentions caching TypeEnv but only exported bindings are needed. May simplify to just exported Schemes from Iface.

## Notes
- Design doc estimated 720 LOC / 2-3 days. This plan estimates 870 LOC / 3 days, primarily because Phase 2 is larger than originally scoped (22 CoreExpr types need gob registration).
- The gob approach for CoreExpr avoids writing 22 JSON marshalers by hand. If gob fails due to unexported fields, fall back to JSON tagged-union (adds ~200 LOC to M2).
- Recent velocity (M-PERF-DOCPARSE: 95 LOC actual vs 430 estimated) suggests actual LOC will be lower than estimates.
