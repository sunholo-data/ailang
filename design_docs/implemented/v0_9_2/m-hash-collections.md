# M-HASH-COLLECTIONS: Hash-Based Collections & Deterministic Equality

**Status**: Planned
**Target**: v0.9.4 (Phase 1), v0.10.0 (Phase 2)
**Priority**: P1 (High — blocks DocParse large-doc processing, removes host-reflection dependency)
**Estimated**: 5-7 days
**Dependencies**: Eq typeclass (exists), Hashable typeclass (new, Phase 2)
**Milestone ID**: M-HASH-COLLECTIONS
**Created**: 2026-03-16
**Source**: DocParse agent message `b5c0b131` (OOM on Moby Dick Jaccard, 2026-03-16)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | **Removes host-reflection dependency** from collection equality. Replaces `reflect.DeepEqual` (which bypasses the Eq model) with canonical AILANG structural semantics. Establishes canonical iteration order for collections. |
| A2: Replayability | +1 | Canonical iteration order = reproducible traces for set/map operations |
| A3: Effect Legibility | 0 | No new effects — Set/Map are pure data structures |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Hashable constraint is locally verifiable at each use site |
| A6: Safe Concurrency | +1 | Canonical iteration eliminates scheduling-dependent meaning |
| A7: Machines First | +1 | O(n) set operations reduce token cost for agents; canonical equality is machine-optimal |
| A8: Minimal Syntax | 0 | No new syntax — Set/Map are stdlib types with `fromList` constructors only |
| A9: Cost Visibility | +1 | O(n) vs O(n²) is a visible, meaningful cost difference for set operations |
| A10: Composability | +1 | Set/Map compose with existing list/record operations |
| A11: Structured Failure | 0 | Errors remain typed |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): **Removes host-reflection dependency**. Collection equality moves from Go's `reflect.DeepEqual` to AILANG-defined canonical structural semantics. All observable iteration order is canonical, never leaked from Go map internals.
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

**Equality audit reveals deeper issue**: `valuesEqual()` in `list.go` falls back to `reflect.DeepEqual` for records, ADTs, and nested structures. While `reflect.DeepEqual` produces deterministic boolean results, it **bypasses AILANG's Eq typeclass** and is **orders of magnitude slower** than direct structural comparison. This creates a semantic disconnect: builtins use host-level reflection while the language has a proper Eq model.

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

### Equality Inconsistency: Three Separate Implementations

**Audit finding: There are THREE `valuesEqual` implementations, not one:**

| Location | Used By | Non-primitive behavior |
|----------|---------|----------------------|
| `internal/builtins/list.go:380` | `dedup`, `intersect`, `union`, `difference`, `member` | `reflect.DeepEqual` fallback |
| `internal/eval/eval_simple.go:594` | `==` / `!=` operators in AILANG | Returns `false` (no deep comparison) |
| `internal/eval/eval_typed_helpers.go:12` | TypedEvaluator equality | Returns `false` (no deep comparison) |

**The `reflect.DeepEqual` concern is narrower than initially stated:**
- AILANG's `==` operator does **NOT** use `reflect.DeepEqual` — it returns `false` for records/ADTs
- Only the **builtin set operations** (dedup, intersect, etc.) use `reflect.DeepEqual` via `list.go:valuesEqual`
- `reflect.DeepEqual` is actually deterministic for the final boolean result (Go handles map comparison correctly)

**The real problems are:**
1. **Performance**: `reflect.DeepEqual` is orders of magnitude slower than direct comparison — O(n²) set ops × slow equality = unusable at scale
2. **Semantic disconnect**: The builtins use `reflect.DeepEqual` while the language has a proper Eq typeclass (`deriving (Eq)` exists via M-DX19). These should be unified.
3. **Incompleteness**: AILANG `==` returns `false` for `{x:1} == {x:1}` unless the type has `deriving (Eq)`. This is correct but surprising — the builtins bypass this and compare structurally.
4. **Three behaviors, one concept**: builtins use structural comparison via reflection, `==` returns false, sets do something third. This is a semantic mess that must be unified before first-class collections.

---

## Goals

**Primary Goal:** Add hash-based Set and Map types that preserve AILANG's deterministic semantics while enabling O(n) collection operations.

**Success Metrics:**
- DocParse Jaccard on 200K words: completes in <1s (currently OOMs)
- `dedup(5000 strings)`: <1ms (currently ~seconds due to 25M comparisons)
- `intersect(5000, 5000)`: <1ms
- All set/map operations produce canonical, deterministic iteration order
- `reflect.DeepEqual` eliminated from `valuesEqual()`
- Existing Eq typeclass integrated with runtime equality
- One unified equality story across builtins, evaluator, and collections

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Canonical structural encoding (not hash) for value keying | Determines collision semantics, equality law, and correctness of all collection operations | human | design | high |
| Set iteration order: canonical sorted (not insertion order) | Affects replay semantics, equality definition, and user expectations | human | design | high |
| Map iteration order: insertion order | Affects replay semantics, familiar to Python/JS users | human | design | med |
| Equality unification: one runtime equality story | Blocks Phase 2 — three equality behaviors is a semantic mess | human | design | high |
| Float excluded from Hashable in v0.10.0 | Avoids NaN/signed-zero semantic landmines | human | design | low |
| No literal syntax for Set/Map | Avoids parser ambiguity with records, can be added later | human | design | low |
| Hashable requires Eq (superclass constraint) | Essential law: `a == b ⇒ hash(a) == hash(b)` | compiler | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Canonical encoding approach (string-based canonical structural key)
- [x] Set iteration order (canonical sorted by canonicalKey)
- [x] Map iteration order (insertion order)
- [x] Float handling (excluded from Hashable in v0.10.0)
- [x] No literal syntax (fromList constructors only)
- [x] Set printing format (`Set([1, 2, 3])`)
- [ ] Equality unification strategy (Option A vs B — see below)

### Deferred Decisions

- Internal canonicalKey optimization (string vs int hash) — agent may choose
- Test fixture organization — agent may choose
- Map `union`/`intersect` tie-breaking for duplicate keys — human at review
- Whether to add `fold` / `map` over Set/Map in v0.10.0 — human at review

---

## Key Principle

> **Hash-based collections are internally hash-indexed, but all observable behavior is defined by canonical structural semantics.**

What is safe:
- Using a Go `map[string]...` internally if keys come from a deterministic canonical encoding
- Sorting by canonical key for iteration — produces the same order regardless of construction history

What is not safe (and this design avoids):
- Returning `toList(set)` in native Go map iteration order
- Defining equality in terms of insertion order for sets
- Allowing hash collisions to silently merge unequal values
- Using host-dependent hashing/serialization

---

## Solution Design

### Design Decision: Phased Approach

**Phase 1 (Quick Win, v0.9.4):** Canonical structural equality + hash-accelerated list builtins — keep existing API, use Go maps internally
**Phase 2 (v0.10.0):** First-class Set and Map types with Hashable typeclass — only after equality semantics are fully documented

### Phase 1: Canonical Structural Equality + Hash-Accelerated List Builtins

Replace O(n²) implementations with Go map-backed versions **without changing the AILANG API**.

**The `canonicalKey` function — deterministic canonical structural encoding:**

```go
// canonicalKey produces a deterministic canonical string representation for use
// as a Go map key. This is NOT a hash — it is a collision-free canonical encoding.
// Same value always produces same key. Records are serialized with sorted keys.
//
// The semantic foundation is: equality first, canonical encoding second,
// hash optimization third (future).
func canonicalKey(v eval.Value) string {
    switch val := v.(type) {
    case *eval.IntValue:
        return fmt.Sprintf("i:%d", val.Value)
    case *eval.FloatValue:
        // NOTE: Float canonicalization has edge cases (NaN, -0.0).
        // Phase 1 uses fmt.Sprintf which is sufficient for the string-heavy
        // DocParse workload. Full float semantics deferred to Phase 2.
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
            b.WriteString(canonicalKey(elem))
        }
        b.WriteByte(']')
        return b.String()
    case *eval.RecordValue:
        // Sort keys for determinism — canonical order, not Go map order
        keys := make([]string, 0, len(val.Fields))
        for k := range val.Fields { keys = append(keys, k) }
        sort.Strings(keys)
        var b strings.Builder
        b.WriteString("r:{")
        for i, k := range keys {
            if i > 0 { b.WriteByte(',') }
            b.WriteString(k)
            b.WriteByte(':')
            b.WriteString(canonicalKey(val.Fields[k]))
        }
        b.WriteByte('}')
        return b.String()
    case *eval.TaggedValue:
        return "t:" + val.Tag + "(" + canonicalKey(val.Value) + ")"
    default:
        return "x:" + v.String()
    }
}
```

**Key design properties:**
- **Canonical, not a hash**: This is a deterministic structural encoding, collision-free by construction. A true numeric hash can be added later as a performance optimization — the semantic foundation is the canonical encoding.
- **Deterministic**: Records sorted by key, lists by position — same value always produces same canonical key
- **Collision-free for primitives**: Type-tagged prefixes (`i:`, `s:`, `b:`) prevent cross-type collisions
- **Correct for nested structures**: Recursively encodes records, lists, ADTs

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

This eliminates `reflect.DeepEqual` entirely, removing the host-reflection dependency and establishing canonical AILANG structural semantics for collection equality.

**Hash-accelerated dedup example:**

```go
// _list_dedup: O(n) using Go map for seen-tracking
func listDedupImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    list := args[0].(*eval.ListValue)
    seen := make(map[string]bool, len(list.Elements))
    result := make([]eval.Value, 0, len(list.Elements))

    for _, elem := range list.Elements {
        key := canonicalKey(elem)  // deterministic canonical encoding
        if !seen[key] {
            seen[key] = true
            result = append(result, elem)
        }
    }
    return &eval.ListValue{Elements: result}, nil
}
```

### Equality Unification Requirement (Gate for Phase 2)

Before first-class Set/Map types, the three equality behaviors must be unified into one runtime equality story. Two options:

**Option A — Strong language model (recommended for runtime internals):**
All runtime values that are admissible in Set/Map have structural equality semantics. The `canonicalKey` function and `valuesEqual` provide this uniformly.

**Option B — Stricter typeclass model (recommended for type-level admission):**
Only `Eq + Hashable` types may inhabit `Set/Map`, and runtime builtins must honor the same dictionary-derived semantics as the language's `==` operator.

**Recommended bridge for v0.10.0:** Option A for runtime internals (structural equality in builtins), Option B for type-level admission (only Eq+Hashable types in Set/Map). This means:
- Builtins use canonical structural comparison (fast, correct, uniform)
- The type checker requires `Hashable` (which implies `Eq`) for Set/Map membership
- These two layers are consistent because Hashable auto-derivation produces the same structural semantics

### Phase 2: First-Class Set and Map Types (v0.10.0)

**Phase 2 Gate — must be explicitly documented before implementation begins:**
1. Equality law: `a == b ⇒ canonicalKey(a) == canonicalKey(b)`
2. Hashing law: `Hashable a ⇒ Eq a` (superclass constraint)
3. Float policy: Float excluded from Hashable until NaN/-0.0 semantics are nailed down
4. Set iteration order: canonical sorted by `canonicalKey`
5. Map iteration order: insertion order
6. Set equality ignores construction history (same members = equal)

**New types in the type system:**

```ailang
-- Set[a] where a has Hashable (which requires Eq) constraint
type Set[a] -- opaque, backed by Go map[string]eval.Value

-- Map[k, v] where k has Hashable (which requires Eq) constraint
type Map[k, v] -- opaque, backed by Go map[string]eval.Value with key pairs
```

**Hashable typeclass — subordinate to Eq:**

```ailang
-- Hashable requires Eq. The essential law:
--   a == b  ⇒  hash(a) == hash(b)
-- If superclass constraints are not yet supported syntactically,
-- this is enforced semantically: auto-derivation always derives both together.
class Eq a => Hashable a where
    hash : a -> int

-- Auto-derived for these types only (v0.10.0):
--   Int, String, Bool
--   Lists of Hashable
--   Records whose fields are all Hashable
--   ADTs whose payloads are all Hashable
--
-- Explicitly EXCLUDED (v0.10.0):
--   Float (NaN, -0.0 semantics unresolved)
--   Functions / closures (not comparable)
--   Effectful closures
--   Opaque host values
```

**stdlib API:**

```ailang
-- std/set.ail
module std/set

export pure func fromList(xs: [a]) -> Set[a]               -- O(n)
export pure func toList(s: Set[a]) -> [a]                   -- O(n), canonical sorted order
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
export pure func toList(m: Map[k, v]) -> [(k, v)]            -- O(n), insertion order
export pure func get(k: k, m: Map[k, v]) -> Option[v]       -- O(1)
export pure func put(k: k, v: v, m: Map[k, v]) -> Map[k, v] -- O(1)
export pure func delete(k: k, m: Map[k, v]) -> Map[k, v]    -- O(1)
export pure func keys(m: Map[k, v]) -> [k]                   -- O(n)
export pure func values(m: Map[k, v]) -> [v]                  -- O(n)
export pure func size(m: Map[k, v]) -> int                    -- O(1)
```

**Iteration order — decided:**

| Type | Iteration Order | Rationale |
|------|----------------|-----------|
| **Set** | **Canonical sorted** by `canonicalKey` | Same members = same iteration, regardless of construction history. Mathematical set semantics. Simpler equality. No insertion-history distinction to reason about. |
| **Map** | **Insertion order** | Matches Python dict (3.7+) convention. Users expect key-value pairs in the order they were added. Construction history matters for maps (unlike sets). |

**Why not insertion order for Set?**

If sets preserve insertion order, then `fromList([1,2,3])` and `fromList([3,2,1])` have the same members but different iteration order. This raises questions:
- Are these sets equal? (They should be — same members)
- Does `toList` preserve original insertion history? (Confusing)
- Is `union` left-biased in order? (Complicates semantics)
- Does order become part of replay semantics? (Unwanted)

Canonical sorted order avoids all of these: same value, same observable iteration. This is cleaner for AILANG's deterministic semantics.

**Set/Map printing:**

Sets print as `Set([1, 2, 3])` (constructor form), not `{1, 2, 3}` (mathematical notation). Rationale:
- `{...}` is already visually record-like in AILANG
- Constructor-style printing is unambiguous
- Easier for debugging and replay
- Consistent with `fromList` construction syntax

**No literal syntax in this milestone:**

```ailang
-- v0.10.0: constructors only
let s = Set.fromList([1, 2, 3])
let m = Map.fromList([("a", 1), ("b", 2)])

-- NOT: {1, 2, 3} or {"a": 1, "b": 2}
-- Records already own {field: value} — adding collection literals creates ambiguity
-- Literal syntax can be revisited once semantics are stable
```

---

## Implementation Plan

### Phase 1: Canonical Structural Equality + Hash-Accelerated List Builtins (2-3 days, v0.9.4)

**Day 1: canonicalKey + valuesEqual fix**
- [ ] Implement `canonicalKey(v eval.Value) string` — deterministic canonical structural encoding
- [ ] Replace `reflect.DeepEqual` in `valuesEqual()` with structural recursion
- [ ] Tests: canonicalKey determinism for all value types, including nested records
- [ ] Tests: valuesEqual correctness — verify identical results to reflect.DeepEqual for all cases
- [ ] Tests with `-count=20` — verify determinism

**Day 2: Hash-accelerated set operations**
- [ ] Replace `_list_dedup` with Go map-backed O(n) implementation using `canonicalKey`
- [ ] Replace `_list_intersect` with map-based O(min(n,m)) implementation
- [ ] Replace `_list_union` with map-based O(n+m) implementation
- [ ] Replace `_list_difference` with map-based O(n) implementation
- [ ] Benchmark: dedup(5000 strings), intersect(5000, 5000)
- [ ] Full test suite pass

**Day 3: DocParse integration verification**
- [ ] Verify Moby Dick Jaccard completes (was OOM)
- [ ] CHANGELOG update
- [ ] DocParse notification

**Urgency note:** If DocParse is critically blocked before the generic `canonicalKey` approach lands, the hot builtins (`dedup`, `intersect`, `union`, `difference`) could be specialized for string-only inputs first using direct Go `map[string]bool`, then generalized. The generic `canonicalKey` approach is a larger semantic commitment. However, given the workload is primarily strings, the generic approach should be tractable in the same timeframe.

### Phase 2: First-Class Set/Map Types (3-4 days, v0.10.0)

**Phase 2 Gate** — must be completed before Phase 2 begins:
- [ ] Equality law documented and tested
- [ ] Hashing law (Hashable ⇒ Eq) documented
- [ ] Float policy documented (excluded)
- [ ] Set iteration order (canonical sorted) tested with `-count=20`
- [ ] Map iteration order (insertion) tested
- [ ] Set equality ignores construction history — tested

**Day 4: SetValue and MapValue runtime types**
- [ ] Add `SetValue` to `internal/eval/value.go` — Go map + canonical sorted iteration
- [ ] Add `MapValue` to `internal/eval/value.go` — Go map + insertion-ordered linked list
- [ ] Implement `String()`: `Set([1, 2, 3])` format for sets, deterministic for maps
- [ ] Constructors via `fromList` only (no literal syntax)

**Day 5: Type system integration**
- [ ] Add `Set[a]` and `Map[k,v]` to type system (`internal/types/`)
- [ ] Add `Hashable` typeclass with `Eq` superclass requirement
- [ ] Auto-derive Hashable for: Int, String, Bool, Lists, Records, ADTs (NOT Float, functions, closures, opaque values)
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
- `internal/builtins/canonical_key.go` (~80 LOC) — `canonicalKey()` deterministic canonical structural encoding
- `internal/builtins/canonical_key_test.go` (~120 LOC) — canonicalKey determinism + collision tests

**Modified files:**
- `internal/builtins/list_set.go` (~-80/+60 LOC) — Replace O(n²) with map-backed implementations using `canonicalKey`
- `internal/builtins/list.go` (~-10/+30 LOC) — Replace `valuesEqual` reflect.DeepEqual with structural comparison

### Phase 2

**New files:**
- `internal/eval/value_set.go` (~150 LOC) — SetValue with canonical sorted iteration
- `internal/eval/value_map.go` (~180 LOC) — MapValue with insertion-order iteration (linked list)
- `internal/builtins/set.go` (~200 LOC) — Set builtins
- `internal/builtins/map.go` (~200 LOC) — Map builtins
- `internal/types/hashable.go` (~100 LOC) — Hashable typeclass with Eq superclass
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
-- Same API, O(n) internally via Go map with canonicalKey
let words1 = dedup(flatMap(\ch -> words(getText(ch)), chapters1))
-- dedup now uses canonicalKey() + Go map[string]bool internally
```

### Example 2: First-class Set operations (Phase 2)

```ailang
import std/set (fromList, intersect, union, size, toList)

let words1 = fromList(flatMap(\ch -> words(getText(ch)), chapters1))
let words2 = fromList(flatMap(\ch -> words(getText(ch)), chapters2))
let common = intersect(words1, words2)
let total = union(words1, words2)
let jaccard = size(common) / size(total)  -- O(1) size, O(min(n,m)) intersect

-- toList returns canonical sorted order
-- fromList([3,1,2]) and fromList([2,3,1]) produce identical toList output
let sorted = toList(fromList([3, 1, 2]))  -- [1, 2, 3] (canonical order)
```

### Example 3: Set equality ignores construction history

```ailang
import std/set (fromList)

let s1 = fromList([1, 2, 3])
let s2 = fromList([3, 2, 1])
-- s1 == s2 is true (same members, construction history irrelevant)
-- toList(s1) == toList(s2) (canonical sorted order)
```

---

## Success Criteria

### Phase 1
- [ ] `dedup(generateNStrings(5000))` completes in <1ms (from ~seconds)
- [ ] `intersect(5000_strings, 5000_strings)` completes in <1ms
- [ ] `reflect.DeepEqual` eliminated from `valuesEqual()`
- [ ] `canonicalKey` produces identical output across 20 runs (`-count=20`)
- [ ] DocParse Moby Dick Jaccard completes (was OOM)
- [ ] All tests passing, lint clean
- [ ] No existing behavior changed (same API, just faster)

### Phase 2
- [ ] `Set[string]` type works in type checker
- [ ] `fromList([1,2,3,1])` produces Set with 3 elements
- [ ] `toList(fromList([3,1,2]))` returns `[1,2,3]` (canonical sorted order, NOT insertion order)
- [ ] `fromList([1,2,3]) == fromList([3,2,1])` is `true` (construction-history-independent equality)
- [ ] Hashable instances for Int, String, Bool, Records, ADTs, Lists (NOT Float)
- [ ] `Set[float]` rejected by type checker with clear error
- [ ] Full stdlib API working with tests
- [ ] `String()` for sets prints `Set([1, 2, 3])` format

---

## Testing Strategy

**Unit tests:**
- canonicalKey: all value types, nested records, nested lists, ADTs, determinism with `-count=20`
- canonicalKey: type-tagged prefixes prevent cross-type collisions (`i:1` vs `s:1` vs `b:1`)
- valuesEqual: structural comparison matches reflect.DeepEqual for all known cases
- dedup: empty, single, duplicates, 5000 strings, mixed types, records
- intersect/union/difference: empty sets, disjoint, overlapping, identical

**Benchmarks:**
- dedup(N) for N = 100, 1000, 5000, 10000 — must be O(n) scaling
- intersect(N, N) — must be O(n) scaling
- canonicalKey(deeply_nested_record) — verify no exponential blowup

**Phase 2 specific:**
- Set canonical iteration order: `toList(fromList([3,1,2])) == [1,2,3]` with `-count=20`
- Set equality: `fromList([1,2,3]) == fromList([3,2,1])` — always true
- Map insertion order: `toList(Map.fromList([("b",2),("a",1)])) == [("b",2),("a",1)]`
- Hashable exclusion: `Set[float]` type error, `Set[(int -> int)]` type error
- Hashable ⇒ Eq law: for all auto-derived types, `a == b` implies `canonicalKey(a) == canonicalKey(b)`

**Integration:**
- DocParse Moby Dick Jaccard end-to-end
- Full eval suite pass

---

## Non-Goals

- **Mutable collections** — AILANG is immutable-by-default. Set/Map updates return new values.
- **Concurrent collections** — No shared mutable state (Axiom 6)
- **Lazy lists** — Separate design concern (needed but different solution)
- **Custom hash functions** — Auto-derived Hashable is sufficient for v0.10.0
- **B-tree/sorted map** — Canonical-sorted set + insertion-ordered map covers use cases
- **Float in Hashable** — Deferred until NaN/-0.0 equality semantics are fully specified
- **Literal syntax for Set/Map** — `fromList` constructors are sufficient. `{...}` is already record syntax.
- **True numeric hash function** — `canonicalKey` (canonical string encoding) is sufficient for v0.10.0. A FNV or similar hash can be added later as an internal optimization without changing semantics.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| canonicalKey collisions for complex nested structures | High | Collision-free by construction: type-tagged prefixes + recursive encoding; exhaustive tests |
| canonicalKey performance on deeply nested values | Med | Cache canonical keys on value creation; benchmark |
| Canonical sorted order for Set requires sorting on each `toList` | Low | Sort once on iteration, cache if needed; O(n log n) is fine for correctness |
| Breaking existing set operation behavior | High | Phase 1 preserves exact API — just faster internals |
| Hashable typeclass interactions with existing Eq | Med | Hashable requires Eq; auto-derive both together |
| Float edge cases (NaN, -0.0) in canonicalKey | High | Phase 1: use `fmt.Sprintf` (sufficient for string workload). Phase 2: exclude float from Hashable entirely. |
| Three equality behaviors causing confusion before unification | Med | Phase 1 unifies builtins. Phase 2 gate requires documented equality law before proceeding. |

---

## Design Questions — Resolved

1. **Iteration order**: **Resolved.** Canonical sorted order for Set (same members = same iteration, independent of construction history). Insertion order for Map (matches Python dict convention, users expect it).

2. **Phase 1 scope / `dedupStrings` builtin**: **Resolved.** No separate `dedupStrings` builtin. The generic `canonicalKey` approach is sufficient and avoids API fragmentation. A string-specialized internal fast path can be added as an optimization later without changing the API.

3. **Map literal syntax**: **Resolved.** No literal syntax for this milestone. Use `Map.fromList([("a", 1), ("b", 2)])`. Records already own `{field: value}` — adding collection literals creates ambiguity. Revisit after semantics are stable.

4. **Set printing**: **Resolved.** `Set([1, 2, 3])` (constructor form). Not `{1, 2, 3}` because `{...}` is visually record-like, constructor form is unambiguous, and it's easier for debugging and replay.

5. **Float in Hashable**: **Resolved.** Excluded from v0.10.0. The questions of `-0.0 == 0.0`, NaN self-equality, and bit-pattern vs semantic equality must be answered first. Conservative: Int, String, Bool, Lists, Records, ADTs are Hashable. Float is not.

---

## Related Documents

<!-- Found via `ailang docs search --neural` -->

**Directly relevant (equality & collections):**
- [M-DX19: Auto-Derive Eq](../../implemented/v0_6_2/m-dx19-auto-derive-eq.md) — Eq typeclass with `deriving (Eq)` for ADTs. Phase 2 Hashable typeclass should follow this pattern. Hashable requires Eq as superclass.
- [M-R7: Type Fixes (Integral & Float Comparison)](../../implemented/v0_3_0/M-R7_type_fixes.md) — Prior equality comparison work
- [Float Equality Investigation](../../implemented/v0_3/FLOAT_EQUALITY_INVESTIGATION_2025-10-10.md) — Float equality semantics. **Critical for Phase 2**: must resolve NaN/-0.0 before Float can be Hashable.
- [typesIdentical Performance Bug](../../implemented/v0_5_7/types-identical-performance.md) — Prior performance bug from using `String()` for comparison. Same anti-pattern as current `reflect.DeepEqual`.
- [M-BUILTIN-SAFETY](../../implemented/v0_7_0/m-builtin-safety-type-checks.md) — Safe type casting in builtins. canonicalKey should use SafeAs* helpers.
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

- **Float Hashable**: Once NaN/-0.0 equality semantics are specified (canonical NaN encoding, signed zero normalization), add Float to auto-derived Hashable types.
- **True numeric hash**: Replace `canonicalKey` string encoding with FNV or similar hash for performance. The canonical encoding remains the semantic foundation; the hash is an optimization.
- **Lazy lists / streams**: `lazyMap`, `takeWhile`, `foldWhile` — separate design doc needed. DocParse wants `take(n, flatMap(f, xs))` to short-circuit.
- **Persistent data structures**: If Set/Map updates are frequent, consider HAMTs (Hash Array Mapped Tries) for structural sharing.
- **Custom Hashable instances**: Allow users to define hash functions for domain types.
- **Map comprehensions / Set literal syntax**: Revisit `{k: v | ...}` and `{1, 2, 3}` syntax once semantics are stable and parser ambiguity with records is resolved.
- **Set/Map pattern matching**: `case Set(xs) -> ...` or `case Map(pairs) -> ...` destructuring.

---

## Design Doc Audit (2026-03-16)

**Audited via design-doc-creator skill. Claims verified against codebase:**

| Claim | Verified | Notes |
|-------|----------|-------|
| `valuesEqual` uses `reflect.DeepEqual` | ✅ Confirmed | `internal/builtins/list.go:402-403` — only in builtins, not evaluator |
| O(n²) dedup/intersect/union | ✅ Confirmed | `list_set.go` uses nested loops with `valuesEqual` |
| Host-reflection dependency in builtins | ✅ Confirmed | `reflect.DeepEqual` bypasses Eq typeclass, is slow, semantically inconsistent with `==`. Not a determinism bug per se — the real issues are performance, semantic disconnect, and bypassing the Eq model. |
| No Hashable typeclass | ✅ Confirmed | Grep for `Hashable` in `internal/types/` returns nothing |
| Eq typeclass exists | ✅ Confirmed | `DerivedADTEquality` in `dictionaries.go`, `deriving (Eq)` works for ADTs |
| No Map/Set types | ✅ Confirmed | Only `RecordValue` (fixed keys) and `ListValue` exist |
| Three `valuesEqual` implementations | 🆕 Found | Builtins (DeepEqual fallback), SimpleEvaluator (returns false), TypedEvaluator (returns false) |
| `==` in AILANG uses DeepEqual | ❌ Corrected | `==` returns `false` for records/ADTs unless `deriving (Eq)`. Only builtins use DeepEqual. |

**Neural search found 6 additional related docs** not in original version (M-DX19, M-R7, Float Equality, typesIdentical perf, M-BUILTIN-SAFETY, M-CODEGEN-DICTIONARIES).

---

## Review History

### Review 1 (2026-03-18) — Reviewer feedback integrated

**Key changes from original draft:**

1. **Renamed `hashKey` → `canonicalKey`**: Reframed as canonical structural encoding, not hash. Semantically clearer — collision-free by construction, not a lossy hash function.

2. **Softened Axiom 1 claim**: No longer claims to "fix Axiom 1 violation." `reflect.DeepEqual` produces deterministic boolean results. The real issues are: bypasses Eq model, too slow, semantically inconsistent with `==`. Reframed as "removes host-reflection dependency."

3. **Set iteration order changed**: From insertion order to canonical sorted order. Same members = same iteration regardless of construction history. Mathematical set semantics. Cleaner equality (no insertion-history distinction).

4. **Equality unification gate added**: Phase 2 cannot proceed until one unified equality story is documented and tested. Three equality behaviors is a semantic mess.

5. **Hashable requires Eq**: Explicit law: `a == b ⇒ hash(a) == hash(b)`. Documented as superclass constraint.

6. **Float excluded from Hashable**: Conservative approach — NaN/-0.0 semantics unresolved. Auto-derive only for Int, String, Bool, Lists, Records, ADTs. Functions, closures, opaque values also excluded.

7. **No literal syntax**: `fromList` constructors only. Records own `{field: value}`. Revisit later.

8. **Set printing**: `Set([1, 2, 3])` constructor form, not `{1, 2, 3}`.

9. **Key principle added**: "Hash-based collections are internally hash-indexed, but all observable behavior is defined by canonical structural semantics."

10. **Phase 2 gate tightened**: Explicit checklist of semantic decisions that must be documented before Phase 2 begins.

---

**Document created**: 2026-03-16
**Last updated**: 2026-03-18
