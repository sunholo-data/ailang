# M-HASH-COLLECTIONS: Hash-Based Collections & Deterministic Equality

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (High — blocks DocParse large-doc processing, fixes Axiom 1 violation)
**Estimated**: 5-7 days
**Dependencies**: Eq typeclass (exists), Hashable typeclass (new)
**Milestone ID**: M-HASH-COLLECTIONS
**Created**: 2026-03-16
**Source**: DocParse agent message `b5c0b131` (OOM on Moby Dick Jaccard, 2026-03-16)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | **Fixes existing violation**: `reflect.DeepEqual` on RecordValues uses nondeterministic Go map iteration. Hash collections use sorted iteration order. |
| A2: Replayability | +1 | Deterministic iteration = reproducible traces for set/map operations |
| A3: Effect Legibility | 0 | No new effects — Set/Map are pure data structures |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Hashable constraint is locally verifiable at each use site |
| A6: Safe Concurrency | +1 | Deterministic iteration eliminates scheduling-dependent meaning |
| A7: Machines First | +1 | O(n) set operations reduce token cost for agents; hashed equality is machine-optimal |
| A8: Minimal Syntax | 0 | No new syntax — Set/Map are stdlib types with builtin constructors |
| A9: Cost Visibility | +1 | O(n) vs O(n²) is a visible, meaningful cost difference for set operations |
| A10: Composability | +1 | Set/Map compose with existing list/record operations |
| A11: Structured Failure | 0 | Errors remain typed |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): **Actively fixes** a current violation. Hash collections use deterministic iteration (sorted by insertion order or sorted keys).
- [x] A3 (Effects): No hidden side effects — Set/Map are pure
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Reduces computational cost for common operations

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — this is a **fundamental gap** in AILANG's collection infrastructure:

1. **v0.7.4** (M-STDLIB-GAPS): Added list set operations (dedup, intersect, union) — O(n²) with `reflect.DeepEqual`
2. **v0.9.3** (M-DOCPARSE-DX): DocParse adopted set operations, immediately hit O(n²) wall on real data
3. **v0.9.3** (M-PERF7): DocParse reports OOM on 200K-word Jaccard similarity — `flatMap` materializes full list, then `dedup` does 25M comparisons

**Pattern**: Every time AILANG is used for real data processing, O(n²) set operations become the bottleneck. Patching individual builtins won't fix this — the language needs hash-based collections as a first-class concept.

**Equality audit reveals deeper issue**: `valuesEqual()` in `list.go` falls back to `reflect.DeepEqual` for records, ADTs, and nested structures. Go's `reflect.DeepEqual` iterates maps nondeterministically, meaning **AILANG's set operations are technically nondeterministic today** — violating Axiom 1.

---

## Problem Statement

### Immediate Problem: DocParse OOM

DocParse's Jaccard similarity computation on Moby Dick (800KB EPUB, ~200K words):

```ailang
-- This OOMs: flatMap materializes 200K strings, dedup does 25M comparisons
let words1 = dedup(flatMap(\ch -> words(getText(ch)), chapters1))
let words2 = dedup(flatMap(\ch -> words(getText(ch)), chapters2))
let common = intersect(words1, words2)
let jaccard = length(common) / length(union(words1, words2))
```

**Why it fails:**
1. `flatMap` is eager — materializes 200K-element list before `dedup` can cap it
2. `dedup` on 5000 strings = 25M `valuesEqual` calls (O(n²))
3. `intersect` on two 5000-element lists = another 25M calls
4. Each `valuesEqual` for strings allocates via type switch + interface assertion

### Fundamental Problem: No Hash-Based Collections

AILANG has lists (O(n) lookup), records (fixed compile-time keys), and nothing in between. Real-world data processing needs:

| Operation | List (current) | Hash Set (needed) |
|-----------|---------------|-------------------|
| `member` | O(n) | O(1) |
| `dedup` | O(n²) | O(n) |
| `intersect` | O(n×m) | O(min(n,m)) |
| `union` | O(n×m) | O(n+m) |
| `lookup` by key | O(n) | O(1) |

### Axiom 1 Concern: Three Separate Equality Implementations

**Audit finding: There are THREE `valuesEqual` implementations, not one:**

| Location | Used By | Non-primitive behavior |
|----------|---------|----------------------|
| `internal/builtins/list.go:380` | `dedup`, `intersect`, `union`, `difference`, `member` | `reflect.DeepEqual` fallback |
| `internal/eval/eval_simple.go:594` | `==` / `!=` operators in AILANG | Returns `false` (no deep comparison) |
| `internal/eval/eval_typed_helpers.go:12` | TypedEvaluator equality | Returns `false` (no deep comparison) |

**The `reflect.DeepEqual` concern is narrower than initially stated:**
- AILANG's `==` operator does **NOT** use `reflect.DeepEqual` — it returns `false` for records/ADTs
- Only the **builtin set operations** (dedup, intersect, etc.) use `reflect.DeepEqual` via `list.go:valuesEqual`
- `reflect.DeepEqual` is actually deterministic for the final boolean result (Go handles map comparison correctly), but it's **slow** and **bypasses the Eq typeclass**

**The real problems are:**
1. **Performance**: `reflect.DeepEqual` is orders of magnitude slower than direct comparison — O(n²) set ops × slow equality = unusable at scale
2. **Semantic disconnect**: The builtins use `reflect.DeepEqual` while the language has a proper Eq typeclass (`deriving (Eq)` exists via M-DX19). These should be unified.
3. **Incompleteness**: AILANG `==` returns `false` for `{x:1} == {x:1}` unless the type has `deriving (Eq)`. This is correct but surprising — the builtins bypass this and compare structurally.

---

## Goals

**Primary Goal:** Add hash-based Set and Map types that preserve AILANG's deterministic semantics while enabling O(n) collection operations.

**Success Metrics:**
- DocParse Jaccard on 200K words: completes in <1s (currently OOMs)
- `dedup(5000 strings)`: <1ms (currently ~seconds due to 25M comparisons)
- `intersect(5000, 5000)`: <1ms
- All set/map operations produce deterministic iteration order
- `reflect.DeepEqual` eliminated from `valuesEqual()`
- Existing Eq typeclass integrated with runtime equality

---

## Solution Design

### Design Decision: Phased Approach

**Phase 1 (Quick Win, v0.9.4):** Hash-accelerated list builtins — keep existing API, use Go maps internally
**Phase 2 (v0.10.0):** First-class Set and Map types with Hashable typeclass

### Phase 1: Hash-Accelerated List Builtins

Replace O(n²) implementations with Go map-backed versions **without changing the AILANG API**:

```go
// _list_dedup: O(n) using Go map for seen-tracking
func listDedupImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    list := args[0].(*eval.ListValue)
    seen := make(map[string]bool, len(list.Elements))
    result := make([]eval.Value, 0, len(list.Elements))

    for _, elem := range list.Elements {
        key := hashKey(elem)  // deterministic string key
        if !seen[key] {
            seen[key] = true
            result = append(result, elem)
        }
    }
    return &eval.ListValue{Elements: result}, nil
}
```

**The `hashKey` function — deterministic value hashing:**

```go
// hashKey produces a deterministic string representation for use as map key.
// This is NOT a cryptographic hash — it's a canonical string encoding.
// Records are serialized with sorted keys to ensure determinism.
func hashKey(v eval.Value) string {
    switch val := v.(type) {
    case *eval.IntValue:
        return fmt.Sprintf("i:%d", val.Value)
    case *eval.FloatValue:
        return fmt.Sprintf("f:%g", val.Value)
    case *eval.StringValue:
        return "s:" + val.Value
    case *eval.BoolValue:
        if val.Value { return "b:1" }
        return "b:0"
    case *eval.UnitValue:
        return "u"
    case *eval.ListValue:
        var b strings.Builder
        b.WriteString("l:[")
        for i, elem := range val.Elements {
            if i > 0 { b.WriteByte(',') }
            b.WriteString(hashKey(elem))
        }
        b.WriteByte(']')
        return b.String()
    case *eval.RecordValue:
        // Sort keys for determinism
        keys := make([]string, 0, len(val.Fields))
        for k := range val.Fields { keys = append(keys, k) }
        sort.Strings(keys)
        var b strings.Builder
        b.WriteString("r:{")
        for i, k := range keys {
            if i > 0 { b.WriteByte(',') }
            b.WriteString(k)
            b.WriteByte(':')
            b.WriteString(hashKey(val.Fields[k]))
        }
        b.WriteByte('}')
        return b.String()
    case *eval.TaggedValue:
        return "t:" + val.Tag + "(" + hashKey(val.Value) + ")"
    default:
        return "x:" + v.String()
    }
}
```

**Key design properties:**
- **Deterministic**: Records sorted by key, lists by position — same value always produces same hash key
- **Collision-free for primitives**: Type-tagged prefixes (`i:`, `s:`, `b:`) prevent cross-type collisions
- **Correct for nested structures**: Recursively hashes records, lists, ADTs
- **Not a cryptographic hash**: Just a canonical string encoding for use as Go map key

**Also fix `valuesEqual`** to use structural comparison instead of `reflect.DeepEqual`:

```go
func valuesEqual(left, right eval.Value) bool {
    if left == right { return true }

    switch l := left.(type) {
    case *eval.IntValue:
        r, ok := right.(*eval.IntValue); return ok && l.Value == r.Value
    case *eval.StringValue:
        r, ok := right.(*eval.StringValue); return ok && l.Value == r.Value
    // ... other primitives ...
    case *eval.RecordValue:
        r, ok := right.(*eval.RecordValue)
        if !ok || len(l.Fields) != len(r.Fields) { return false }
        for k, lv := range l.Fields {
            rv, exists := r.Fields[k]
            if !exists || !valuesEqual(lv, rv) { return false }
        }
        return true
    case *eval.ListValue:
        r, ok := right.(*eval.ListValue)
        if !ok || len(l.Elements) != len(r.Elements) { return false }
        for i := range l.Elements {
            if !valuesEqual(l.Elements[i], r.Elements[i]) { return false }
        }
        return true
    case *eval.TaggedValue:
        r, ok := right.(*eval.TaggedValue)
        return ok && l.Tag == r.Tag && valuesEqual(l.Value, r.Value)
    }
    return false
}
```

This eliminates `reflect.DeepEqual` entirely, fixing the Axiom 1 violation.

### Phase 2: First-Class Set and Map Types (v0.10.0)

**New types in the type system:**

```ailang
-- Set[a] where a has Hashable constraint
type Set[a] -- opaque, backed by Go map[string]eval.Value

-- Map[k, v] where k has Hashable constraint
type Map[k, v] -- opaque, backed by Go map[string]eval.Value with key pairs
```

**Hashable typeclass:**

```ailang
class Hashable a where
    hash : a -> int

-- Auto-derived for all types with Eq
-- Primitive instances built-in: Int, Float, String, Bool
-- Records: hash of sorted field hashes
-- ADTs: hash of tag + payload hash
-- Lists: hash of element hashes
```

**stdlib API:**

```ailang
-- std/set.ail
module std/set

export pure func fromList(xs: [a]) -> Set[a]               -- O(n)
export pure func toList(s: Set[a]) -> [a]                   -- O(n), deterministic order
export pure func member(x: a, s: Set[a]) -> bool            -- O(1)
export pure func insert(x: a, s: Set[a]) -> Set[a]          -- O(1)
export pure func delete(x: a, s: Set[a]) -> Set[a]          -- O(1)
export pure func union(s1: Set[a], s2: Set[a]) -> Set[a]    -- O(n+m)
export pure func intersect(s1: Set[a], s2: Set[a]) -> Set[a] -- O(min(n,m))
export pure func difference(s1: Set[a], s2: Set[a]) -> Set[a] -- O(n)
export pure func size(s: Set[a]) -> int                      -- O(1)

-- std/map.ail
module std/map

export pure func empty() -> Map[k, v]
export pure func singleton(k: k, v: v) -> Map[k, v]
export pure func fromList(pairs: [(k, v)]) -> Map[k, v]     -- O(n)
export pure func toList(m: Map[k, v]) -> [(k, v)]            -- O(n), deterministic order
export pure func get(k: k, m: Map[k, v]) -> Option[v]       -- O(1)
export pure func put(k: k, v: v, m: Map[k, v]) -> Map[k, v] -- O(1)
export pure func delete(k: k, m: Map[k, v]) -> Map[k, v]    -- O(1)
export pure func keys(m: Map[k, v]) -> [k]                   -- O(n)
export pure func values(m: Map[k, v]) -> [v]                  -- O(n)
export pure func size(m: Map[k, v]) -> int                    -- O(1)
```

**Deterministic iteration order:**

| Option | Iteration Order | Pros | Cons |
|--------|----------------|------|------|
| **Insertion order** | Elements returned in order they were inserted | Predictable, matches user intent | Requires linked list + map (more memory) |
| **Sorted order** | Elements returned sorted by hash key | Canonical, no extra state | May surprise users, sorting cost on each iteration |

**Recommendation:** Insertion order (like Python dicts since 3.7). Implementation: doubly-linked list threading through map entries. This matches the "determinism = reproducibility" axiom — same insertions in same order = same iteration.

---

## Implementation Plan

### Phase 1: Hash-Accelerated List Builtins (2-3 days)

**Day 1: hashKey + valuesEqual fix**
- [ ] Implement `hashKey(v eval.Value) string` — deterministic canonical encoding
- [ ] Replace `reflect.DeepEqual` in `valuesEqual()` with structural recursion
- [ ] Tests: hashKey determinism for all value types, including nested records
- [ ] Tests: valuesEqual correctness — verify identical results to reflect.DeepEqual for all cases
- [ ] Tests with `-count=20` — verify determinism

**Day 2: Hash-accelerated set operations**
- [ ] Replace `_list_dedup` with Go map-backed O(n) implementation
- [ ] Replace `_list_intersect` with map-based O(min(n,m)) implementation
- [ ] Replace `_list_union` with map-based O(n+m) implementation
- [ ] Replace `_list_difference` with map-based O(n) implementation
- [ ] Benchmark: dedup(5000 strings), intersect(5000, 5000)
- [ ] Full test suite pass

**Day 3: DocParse integration verification**
- [ ] Verify Moby Dick Jaccard completes (was OOM)
- [ ] CHANGELOG update
- [ ] DocParse notification

### Phase 2: First-Class Set/Map Types (3-4 days, v0.10.0)

**Day 4: SetValue and MapValue runtime types**
- [ ] Add `SetValue` to `internal/eval/value.go` — insertion-ordered Go map + linked list
- [ ] Add `MapValue` to `internal/eval/value.go`
- [ ] Implement deterministic `String()` for both
- [ ] Set/Map literal syntax (or just `fromList` constructors)

**Day 5: Type system integration**
- [ ] Add `Set[a]` and `Map[k,v]` to type system (`internal/types/`)
- [ ] Add `Hashable` typeclass with auto-deriving for all Eq types
- [ ] Wire Hashable constraint into Set/Map operations

**Day 6: stdlib and builtins**
- [ ] Create `std/set.ail` and `std/map.ail` with full API
- [ ] Register Go builtins for each operation
- [ ] Tests for all operations with `-count=20`
- [ ] Examples: `examples/runnable/sets.ail`, `examples/runnable/maps.ail`

**Day 7: Documentation and verification**
- [ ] Update language reference
- [ ] Full eval suite pass
- [ ] CHANGELOG, design doc update

---

## Files to Modify/Create

### Phase 1

**New files:**
- `internal/builtins/hash.go` (~80 LOC) — `hashKey()` deterministic value hashing
- `internal/builtins/hash_test.go` (~120 LOC) — hashKey determinism + collision tests

**Modified files:**
- `internal/builtins/list_set.go` (~-80/+60 LOC) — Replace O(n²) with map-backed implementations
- `internal/builtins/list.go` (~-10/+30 LOC) — Replace `valuesEqual` reflect.DeepEqual with structural comparison

### Phase 2

**New files:**
- `internal/eval/value_set.go` (~150 LOC) — SetValue with insertion-order iteration
- `internal/eval/value_map.go` (~150 LOC) — MapValue with insertion-order iteration
- `internal/builtins/set.go` (~200 LOC) — Set builtins
- `internal/builtins/map.go` (~200 LOC) — Map builtins
- `internal/types/hashable.go` (~100 LOC) — Hashable typeclass
- `std/set.ail` (~30 LOC) — Set stdlib
- `std/map.ail` (~30 LOC) — Map stdlib
- `examples/runnable/sets.ail` (~30 LOC)
- `examples/runnable/maps.ail` (~30 LOC)

---

## Examples

### Example 1: DocParse Jaccard with hash-accelerated dedup (Phase 1)

**Before (OOM):**
```ailang
-- O(n²) dedup on 200K words -> OOM
let words1 = dedup(flatMap(\ch -> words(getText(ch)), chapters1))
```

**After (completes in <1s):**
```ailang
-- Same API, O(n) internally via Go map
let words1 = dedup(flatMap(\ch -> words(getText(ch)), chapters1))
-- dedup now uses hashKey() + Go map[string]bool internally
```

### Example 2: First-class Set operations (Phase 2)

```ailang
import std/set (fromList, intersect, union, size)

let words1 = fromList(flatMap(\ch -> words(getText(ch)), chapters1))
let words2 = fromList(flatMap(\ch -> words(getText(ch)), chapters2))
let common = intersect(words1, words2)
let total = union(words1, words2)
let jaccard = size(common) / size(total)  -- O(1) size, O(min(n,m)) intersect
```

---

## Success Criteria

### Phase 1
- [ ] `dedup(generateNStrings(5000))` completes in <1ms (from ~seconds)
- [ ] `intersect(5000_strings, 5000_strings)` completes in <1ms
- [ ] `reflect.DeepEqual` eliminated from `valuesEqual()`
- [ ] hashKey produces identical output across 20 runs (`-count=20`)
- [ ] DocParse Moby Dick Jaccard completes (was OOM)
- [ ] All tests passing, lint clean
- [ ] No existing behavior changed (same API, just faster)

### Phase 2
- [ ] `Set[string]` type works in type checker
- [ ] `fromList([1,2,3,1])` produces Set with 3 elements
- [ ] `toList(fromList([3,1,2]))` returns `[3,1,2]` (insertion order preserved)
- [ ] Hashable instances for Int, Float, String, Bool, Records, ADTs, Lists
- [ ] Full stdlib API working with tests

---

## Testing Strategy

**Unit tests:**
- hashKey: all value types, nested records, nested lists, ADTs, determinism with `-count=20`
- valuesEqual: structural comparison matches reflect.DeepEqual for all known cases
- dedup: empty, single, duplicates, 5000 strings, mixed types, records
- intersect/union/difference: empty sets, disjoint, overlapping, identical

**Benchmarks:**
- dedup(N) for N = 100, 1000, 5000, 10000 — must be O(n) scaling
- intersect(N, N) — must be O(n) scaling
- hashKey(deeply_nested_record) — verify no exponential blowup

**Integration:**
- DocParse Moby Dick Jaccard end-to-end
- Full eval suite pass

---

## Non-Goals

- **Mutable collections** — AILANG is immutable-by-default. Set/Map updates return new values.
- **Concurrent collections** — No shared mutable state (Axiom 6)
- **Lazy lists** — Separate design concern (needed but different solution)
- **Custom hash functions** — Auto-derived Hashable is sufficient for v0.10.0
- **B-tree/sorted map** — Insertion-ordered map is simpler and covers use cases

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| hashKey collisions for complex nested structures | High | Type-tagged prefixes + recursive encoding; exhaustive tests |
| hashKey performance on deeply nested values | Med | Cache hash keys on value creation; benchmark |
| Insertion-order map adds memory overhead (~2 pointers per entry) | Low | Acceptable for correctness; document in cost model |
| Breaking existing set operation behavior | High | Phase 1 preserves exact API — just faster internals |
| Hashable typeclass interactions with existing Eq | Med | Hashable implies Eq; auto-derive both together |

---

## Design Questions (Need Input)

1. **Iteration order**: Insertion order (Python-style) vs sorted order? Recommendation: insertion order for predictability.

2. **Phase 1 scope**: Should we also add a quick `dedupStrings` builtin that's optimized for the common string-only case? Or is the generic `hashKey` approach sufficient?

3. **Map literal syntax**: Should maps have literal syntax (`{key: value}`) or only constructors (`fromList([(k1,v1), (k2,v2)])`)? Records already use `{field: value}` — potential ambiguity.

4. **Set printing**: How should `Set[int]` display? `{1, 2, 3}` (mathematical notation) or `Set([1, 2, 3])` (constructor form)?

---

## Related Documents

<!-- Found via `ailang docs search --neural` -->

**Directly relevant (equality & collections):**
- [M-DX19: Auto-Derive Eq](../../implemented/v0_6_2/m-dx19-auto-derive-eq.md) — Eq typeclass with `deriving (Eq)` for ADTs. Phase 2 Hashable typeclass should follow this pattern.
- [M-R7: Type Fixes (Integral & Float Comparison)](../../implemented/v0_3_0/M-R7_type_fixes.md) — Prior equality comparison work
- [Float Equality Investigation](../../implemented/v0_3/FLOAT_EQUALITY_INVESTIGATION_2025-10-10.md) — Float equality semantics (relevant for hashKey of floats)
- [typesIdentical Performance Bug](../../implemented/v0_5_7/types-identical-performance.md) — Prior performance bug from using `String()` for comparison. Same anti-pattern as current `reflect.DeepEqual`.
- [M-BUILTIN-SAFETY](../../implemented/v0_7_0/m-builtin-safety-type-checks.md) — Safe type casting in builtins. hashKey should use SafeAs* helpers.
- [M-CODEGEN-DICTIONARIES](../../implemented/v0_6_2/m-codegen-dictionaries.md) — Typeclass dictionary system. Hashable instances would be generated here.

**Performance precedent:**
- [M-DOCPARSE-DX](../v0_9_3/m-docparse-dx.md) — Added list set operations (M2), now hitting O(n²) wall
- [M-PERF7](../v0_9_3/m-perf7-docparse-production-pipeline.md) — DocParse production pipeline optimization
- [M-PERF5](../v0_9_2/m-perf5-data-intensive-workloads.md) — Data-intensive workload performance

**Pattern matching (affected by Set/Map types):**
- [M-R3: Pattern Matching](../../implemented/v0_2_0/m_r3_pattern_matching.md) — Pattern matching system. Set/Map may need match support in Phase 2+.
- [DEDUP_IMPLEMENTATION](../../implemented/v0_3_6/DEDUP_IMPLEMENTATION.md) — Original dedup design (analysis-level, not collection-level)

---

## Future Work

- **Lazy lists / streams**: `lazyMap`, `takeWhile`, `foldWhile` — separate design doc needed. DocParse wants `take(n, flatMap(f, xs))` to short-circuit.
- **Persistent data structures**: If Set/Map updates are frequent, consider HAMTs (Hash Array Mapped Tries) for structural sharing.
- **Custom Hashable instances**: Allow users to define hash functions for domain types.
- **Map comprehensions**: `{k: v | (k,v) <- pairs, predicate(k)}` syntax.

---

## Design Doc Audit (2026-03-16)

**Audited via design-doc-creator skill. Claims verified against codebase:**

| Claim | Verified | Notes |
|-------|----------|-------|
| `valuesEqual` uses `reflect.DeepEqual` | ✅ Confirmed | `internal/builtins/list.go:402-403` — only in builtins, not evaluator |
| O(n²) dedup/intersect/union | ✅ Confirmed | `list_set.go` uses nested loops with `valuesEqual` |
| Axiom 1 violation via `reflect.DeepEqual` | ⚠️ Narrowed | `reflect.DeepEqual` result is deterministic; real issue is performance + bypassing Eq typeclass |
| No Hashable typeclass | ✅ Confirmed | Grep for `Hashable` in `internal/types/` returns nothing |
| Eq typeclass exists | ✅ Confirmed | `DerivedADTEquality` in `dictionaries.go`, `deriving (Eq)` works for ADTs |
| No Map/Set types | ✅ Confirmed | Only `RecordValue` (fixed keys) and `ListValue` exist |
| Three `valuesEqual` implementations | 🆕 Found | Builtins (DeepEqual fallback), SimpleEvaluator (returns false), TypedEvaluator (returns false) |
| `==` in AILANG uses DeepEqual | ❌ Corrected | `==` returns `false` for records/ADTs unless `deriving (Eq)`. Only builtins use DeepEqual. |

**Neural search found 6 additional related docs** not in original version (M-DX19, M-R7, Float Equality, typesIdentical perf, M-BUILTIN-SAFETY, M-CODEGEN-DICTIONARIES).

---

**Document created**: 2026-03-16
**Last updated**: 2026-03-16
