# M-STD-MAP: HashMap Type + Array Gaps

**Status**: Planned
**Target**: v0.11.0
**Priority**: P1 (High — blocks XLSX >5 MB, OOMs at 8.7 MB)
**Estimated**: 3-4 days
**Dependencies**: None
**Requested by**: ailang-parse agent (2026-03-31)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Map iteration order is sorted by key (deterministic) |
| A2: Replayability | 0 | Pure operations, traces unchanged |
| A3: Effect Legibility | 0 | All functions are pure |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | New type needs full type checker integration |
| A6: Safe Concurrency | 0 | Immutable data structures |
| A7: Machines First | +2 | Enables AI agents to process real XLSX documents |
| A8: Minimal Syntax | 0 | No new syntax — library-level addition |
| A9: Cost Visibility | +2 | O(1) lookups replace O(n) list scans (20 billion -> 100K ops) |
| A10: Composability | +1 | Maps compose with existing types (Option, tuples) |
| A11: Structured Failure | 0 | lookup returns Option[V] (safe by default) |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** -> **Decision: Move forward**

## Problem Statement

XLSX parsing has two critical O(n) bottlenecks caused by missing data structures:

### Bottleneck 1: Shared string resolution (needs Map OR Array)

XLSX files store all text in a shared string table (~200K entries). Cells reference
strings by integer index. Current code uses `nth(sharedStrings, idx)` which is O(n)
on a linked list.

- 5000 rows x 20 columns = 100K cells
- Each cell: O(200K) list traversal
- Total: ~20 billion list node visits

### Bottleneck 2: String index lookup (needs Map)

The XLSX generator uses `xlsxStringIndex()` which does O(n) linear search through
the entire string array for EVERY cell to find its index in the shared strings table.

- 100K cells x 50K unique strings = ~5 billion comparisons

### Impact

- DOCX: 50 MB in 5s (fine)
- PPTX: 50 MB in 9s (fine)
- XLSX: 8.7 MB -> OOM at 2 GB -> SIGKILLed

## Goals

**Primary Goal:** XLSX files up to 50 MB parseable without OOM

**Success Metrics:**
- `std/map` with O(1) lookup (Go map-backed, copy-on-write updates)
- `std/array` gap filled (`empty`; `append` as convenience only)
- XLSX 8.7 MB: completes without OOM, <10s, under 512 MB RSS
- Shared string lookup of 100K cell references completes without quadratic behavior
- Reverse string indexing during XLSX generation uses map lookup, not linear scan
- Full type checker integration (TMap unifies, substitutes, traverses)

## Current State

### std/array — ALREADY IMPLEMENTED

Builtins exist in `internal/builtins/array.go`, wrapper in `std/array.ail`.

**Implemented:** `make`, `fromList`, `toList`, `get`, `getOpt`, `unsafeGet`, `set`, `length`

**Missing (minor gaps):**

| Function | Signature | Notes |
|----------|-----------|-------|
| `empty` | `() -> Array[a]` | Convenience for `make(0, ...)` without needing a default |
| `append` | `(Array[a], a) -> Array[a]` | Grow array by one element |

These can be added as pure AILANG wrappers or thin builtins.

### std/map — NOT IMPLEMENTED

Needs everything: eval value, type system node, unification, builtins, wrapper.

## Solution Design

### Track 1: std/map (New — 2-3 days)

#### 1A. Eval Value (`internal/eval/value.go`)

For v0.11.0, `MapValue` is an immutable wrapper over a Go hashmap.
Lookups are O(1). Updates are copy-on-write and therefore O(n).
This is acceptable for XLSX parsing, which is lookup-heavy after initial construction.

```go
// MapValue represents an immutable hash map (copy-on-write)
type MapValue struct {
    Entries map[string]*MapEntry // canonical key -> entry for O(1) lookup
}

type MapEntry struct {
    Key   Value  // original key value (for export/display)
    Value Value
}

func (m *MapValue) Type() string { return "map" }
func (m *MapValue) String() string { /* deterministic sorted output */ }
func (m *MapValue) Lookup(key Value) (Value, bool) { /* O(1) */ }
func (m *MapValue) Insert(key, val Value) *MapValue { /* O(n) copy-on-write */ }
```

**Key encoding:** Uses a dedicated internal canonical key encoder, separate from
user-facing `String()` rendering. `String()` is presentation-oriented, not
identity-oriented — coupling runtime semantics to debug printing is fragile.

```go
// mapKey returns a canonical, collision-free string key for Go map lookup.
// Distinct from String() which is for display only.
func mapKey(v eval.Value) (string, error) {
    switch k := v.(type) {
    case *eval.IntValue:    return fmt.Sprintf("i:%d", k.Value), nil
    case *eval.StringValue: return "s:" + k.Value, nil
    case *eval.BoolValue:   return fmt.Sprintf("b:%v", k.Value), nil
    default:
        return "", fmt.Errorf("unsupported map key type: %T", v)
    }
}
```

Supported key types for v0.11.0: `int`, `string`, `bool`.
Compound keys (tuples, records) deferred to v1.0 — requires extending `mapKey`.

#### 1B. Type System (`internal/types/types.go`)

```go
// TMap represents a map type Map[K, V]
type TMap struct {
    Key   Type
    Value Type
}

func (t *TMap) String() string { return fmt.Sprintf("Map[%s, %s]", t.Key, t.Value) }
func (t *TMap) Equals(other Type) bool { /* check both Key and Value */ }
func (t *TMap) Substitute(subs map[string]Type) Type { /* substitute both */ }
```

**Type switch locations to update** (follow TArray pattern):

| File | Function | What to add |
|------|----------|-------------|
| `types/inference_helpers.go` | `collectFreeTypeVars` | `case *TMap:` both Key+Value |
| `types/typechecker_defaulting.go` | `collectFreeVarsWithVisited` | `case *TMap:` both |
| `types/unification_equality.go` | `safeEqualsWithVisited` | `case *TMap:` |
| `types/unification_core.go` | `Unify` + `propagateTypeName` | `case *TMap:` |
| `types/unification_substitution.go` | `safeSubstitute` | `case *TMap:` |
| `types/unification_occurs.go` | `occursWithVisited` | `case *TMap:` both |
| `types/safe_string.go` | `safeTypeStringWithDepth` | `case *TMap:` |
| `types/normalize.go` | `NormalizeTypeName` + `IsGroundType` | `case *TMap:` |
| `types/type_head.go` | `TypeHead` | `case *TMap:` + new `HeadMap` |
| `types/types_v2.go` | `Kind` | `case *TMap:` -> Star |
| `types/traverse/traverse.go` | `Children` | `case *TMap:` -> [Key, Value] |
| `types/typechecker_substitution.go` | `propagateTypeNameRecursively` | `case *TMap:` |
| `elaborate/file.go` | `astTypeToType2` | `case *ast.MapType:` (if AST node added) |
| `iface/builder.go` | `convertAstType` | `case *ast.MapType:` |

#### 1C. Builtins (`internal/builtins/map.go`)

| Builtin | AILANG name | Signature | Complexity |
|---------|-------------|-----------|------------|
| `_map_empty` | `empty` | `() -> Map[k, v]` | O(1) |
| `_map_insert` | `insert` | `(Map[k, v], k, v) -> Map[k, v]` | O(n) copy-on-write |
| `_map_lookup` | `lookup` | `(Map[k, v], k) -> Option[v]` | O(1) |
| `_map_member` | `member` | `(Map[k, v], k) -> bool` | O(1) |
| `_map_remove` | `remove` | `(Map[k, v], k) -> Map[k, v]` | O(n) copy-on-write |
| `_map_size` | `size` | `(Map[k, v]) -> int` | O(1) |
| `_map_keys` | `keys` | `(Map[k, v]) -> [k]` | O(n log n) sorted |
| `_map_values` | `values` | `(Map[k, v]) -> [v]` | O(n log n) sorted by key |
| `_map_from_list` | `fromList` | `([(k, v)]) -> Map[k, v]` | O(n) avg (mutable internally) |
| `_map_to_list` | `toList` | `(Map[k, v]) -> [(k, v)]` | O(n log n) sorted |

Note: `fromList` builds the map mutably inside the single builtin call, then returns
an immutable result. This is the intended bulk construction path for XLSX parsing.

#### 1D. Stdlib Wrapper (`std/map.ail`)

```ailang
-- std/map - Hash maps with O(1) lookup
--
-- Maps provide O(1) key-value lookup backed by Go hashmaps.
-- Keys must be comparable types (int, string, bool).
-- All operations are pure and return new maps (immutable).
--
-- COST MODEL:
--   empty()          O(1)      - creates empty map
--   insert(m, k, v)  O(n)      - copy-on-write (immutable semantics)
--   lookup(m, k)     O(1)      - hash lookup
--   member(m, k)     O(1)      - key existence check
--   remove(m, k)     O(n)      - copy-on-write (immutable semantics)
--   size(m)          O(1)      - stored metadata
--   keys(m)          O(n log n) - sorted for determinism (A1)
--   values(m)        O(n log n) - sorted by key for determinism
--   fromList(pairs)  O(n)      - bulk construction (mutable internally)
--   toList(m)        O(n log n) - sorted pairs for determinism
--
-- USAGE PATTERN: Build once with fromList, then many lookups.
-- Avoid repeated insert() in loops — use fromList for bulk construction.

module std/map
import std/option (Option, Some, None)

export pure func empty() -> Map[k, v] { _map_empty() }
export pure func insert[k, v](m: Map[k, v], key: k, val: v) -> Map[k, v] { _map_insert(m, key, val) }
export pure func lookup[k, v](m: Map[k, v], key: k) -> Option[v] { _map_lookup(m, key) }
export pure func member[k, v](m: Map[k, v], key: k) -> bool { _map_member(m, key) }
export pure func remove[k, v](m: Map[k, v], key: k) -> Map[k, v] { _map_remove(m, key) }
export pure func size[k, v](m: Map[k, v]) -> int { _map_size(m) }
export pure func keys[k, v](m: Map[k, v]) -> [k] { _map_keys(m) }
export pure func values[k, v](m: Map[k, v]) -> [v] { _map_values(m) }
export pure func fromList[k, v](pairs: [(k, v)]) -> Map[k, v] { _map_from_list(pairs) }
export pure func toList[k, v](m: Map[k, v]) -> [(k, v)] { _map_to_list(m) }
```

### Track 2: std/array Gaps (Minor — 0.5 day)

The XLSX-critical addition is `empty`. `append` is a convenience gap — not part of
the main XLSX rescue, since the workload is "build once from list, then index".

#### 2A. New Builtins

```go
// _array_empty() -> Array[a]   (zero-length array, no default needed)
// _array_append(arr, val) -> Array[a]  (copy + append, O(n) — convenience only)
```

#### 2B. Stdlib Wrapper Updates (`std/array.ail`)

```ailang
-- Create empty array. O(1).
export pure func empty[a]() -> Array[a] { _array_empty() }

-- Append element to end of array. O(n) due to copy.
-- For bulk building, prefer fromList over repeated append.
export pure func append[a](arr: Array[a], val: a) -> Array[a] { _array_append(arr, val) }
```

## Implementation Plan

### Phase 1: XLSX Unblocker (3-4 days)

#### Milestone 1: eval.MapValue + types.TMap (Day 1)
- Add `MapValue` to `internal/eval/value.go` with `mapKey()` canonical encoder
- Add `TMap` to `internal/types/types.go`
- Update all type switch locations (14 files — grep `case *TArray:` for completeness)
- Add `TMap` traversal test
- `make test` passes

#### Milestone 2: Map Builtins + Wrapper (Day 2)
- Create `internal/builtins/map.go` with all 10 builtins
- Create `std/map.ail` wrapper
- Create `examples/runnable/map_basic.ail`
- Add `_array_empty` builtin + `std/array.ail` update
- `make verify-examples` passes

#### Milestone 3: XLSX Integration Test (Day 3-4)
- Test XLSX parsing with map-backed shared string lookup
- Verify 8.7 MB XLSX completes without OOM, under 512 MB RSS
- Benchmark before/after
- Acceptance: shared string resolution is O(1), reverse index is O(1)

### Phase 2: Refinement (deferred, not blocking XLSX)

- `array.append` convenience builtin (O(n) copy-based)
- Richer key type support (tuples, records)
- Better update performance (persistent HAMT/CHAMP if needed)
- Map syntax literals (`{k: v}`) if warranted by usage

### Parser Note

`Map[k, v]` works via existing `TApp` infrastructure — the parser already handles
multi-parameter type constructors (e.g., `Result[a, e]`). No new `*ast.MapType` node
is needed for v0.11.0. `TMap` is a dedicated type-system node that unifies with
`TApp("Map", [k, v])`, following the same pattern as `TArray`/`TApp("Array", a)`.

## Alternatives Considered

### 1. Array-only (no Map)
Could use `Array[string]` for shared string resolution (index-based access), but
the generator's reverse lookup (string -> index) still needs O(n) scan without Map.
**Rejected:** Only solves half the problem.

### 2. Builtin-only (no type system integration)
Could implement maps as opaque `OpaqueValue` without `TMap`. Simpler but loses
type safety — `lookup(m, 42)` wouldn't catch key type mismatches at compile time.
**Rejected:** Violates A5 (Bounded Verification).

### 3. Sorted tree map (persistent)
Could use a balanced BST for O(log n) everything with structural sharing. More
FP-pure but slower than Go's built-in hashmap for this use case.
**Deferred to v1.0:** If immutable performance matters, revisit with persistent
data structures. For now, copy-on-write over Go map is sufficient.

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Missing type switch locations | Medium | High (silent bugs) | Grep for ALL `case *TArray:` and add parallel `case *TMap:` |
| Key encoding collisions | Low | Low | `mapKey()` uses type-prefixed encoding (`i:`, `s:`, `b:`) — collision-free for supported types |
| Determinism violation (A1) | Medium | High | Sort keys on every export operation (keys/values/toList/String) |
| Compound key types | Low | Low | Defer to v1.0, `mapKey()` returns error on unsupported types |
| Copy-on-write cost for insert-heavy workloads | Medium | Medium | Document clearly; XLSX workload is build-once-read-many so acceptable for v0.11.0 |

## Non-Goals

- Mutable maps (all operations return new maps)
- Compound key types (tuples, records as keys) — deferred
- Map syntax literals (`{k: v}` notation) — deferred
- Lazy map operations — deferred
- Persistent/structural-sharing implementation — deferred to v1.0

## Related Documents

- Agent messages: `9d018578` (std/array request), `a0ce90a5` (std/map request)
- `design_docs/implemented/v0_9_2/m-perf5-data-intensive-workloads.md` — Prior perf work
- `design_docs/archive/v0_4_10_m-array-type.md` — Historical array type design
- `internal/builtins/array.go` — Pattern to follow for map builtins
- `std/array.ail` — Pattern to follow for map wrapper

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-03-31 | Use Go map backing, not BST | Performance > persistence for XLSX use case |
| 2026-03-31 | Deterministic key ordering via sort | Axiom A1 compliance; O(n log n) cost acceptable |
| 2026-03-31 | Canonical key encoding, not String() | String() is display-oriented, not identity-oriented; type-prefixed encoding is collision-free |
| 2026-03-31 | Copy-on-write for insert/remove (O(n)) | Honest about cost; XLSX is build-once-read-many; persistent HAMT deferred to v1.0 |
| 2026-03-31 | Two-phase plan: XLSX unblocker then refinement | Narrow first landing, ship what matters |
| 2026-03-31 | Map[k,v] via TApp, no ast.MapType needed | Parser already supports multi-param type constructors |
| 2026-03-31 | Target v0.11.0 | Substantial type system work, not a patch |
