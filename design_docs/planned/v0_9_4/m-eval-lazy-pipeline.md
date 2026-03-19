# M-EVAL-LAZY-PIPELINE: Lazy List Pipelines to Fix OOM on Large Documents

> **Superseded by:** [M-EVAL-BOUNDED-PIPELINE](./m-eval-bounded-pipeline.md) (Phase 1 scope)
> This document is retained as the original brainstorming analysis. The implementation plan is in the superseding doc.

**Status**: Superseded (brainstorming retained for reference)
**Target**: v0.9.4
**Priority**: P1 (blocks DocParse eval on large documents)
**Estimated**: TBD — depends on approach chosen
**Dependencies**: None
**Milestone ID**: M-EVAL-LAZY-PIPELINE
**Created**: 2026-03-18
**Source**: DocParse agent — AILANG eval OOMs on Moby Dick (1.4MB JSON, 216K words)

---

## Problem Statement

AILANG lists are **strict (eager)**. Every list operation materializes the full result in memory before the next operation can consume it. This means `take(N, flatMap(f, xs))` must compute the ENTIRE `flatMap` result before taking N elements.

### Concrete OOM Scenario (DocParse Moby Dick eval)

```
json.decode(1.4MB JSON)           → full recursive Json AST in memory
  × 2 (golden + actual)           → 2× ASTs simultaneously
flatMap(evalFlattenOneBlock, 2861 blocks) → materializes ALL [EvalElement]
flatMap(evalTokenize, elements)   → materializes ALL word lists (~215K strings)
take(10000, words)                → too late, 215K already allocated
```

Each stage materializes the full intermediate list before the next stage can start. With large documents, the intermediate lists exceed available memory.

### Current Implementation

| Component | File | Problem |
|-----------|------|---------|
| `ListValue` | `internal/eval/value.go:83` | `Elements []Value` — eager Go slice |
| `flatMap` | `internal/builtins/registry_codegen.go:488` | Appends ALL inner results to one slice |
| `take` | `internal/builtins/list.go:625` | Input already fully materialized |
| `json.decode` | `internal/builtins/json_decode.go:235` | Full recursive AST in memory |
| `map`, `filter` | `internal/builtins/list_iterative.go` | Also eager, but less problematic alone |

### What Works Fine

- Set operations (hash-based, O(n)) — 34ms for 216K dedup
- Sequential performance — fast once data fits in memory
- Small-to-medium documents (<500KB JSON output)

---

## Design Options

### Option A: Fused Pipeline Builtins (Pragmatic, Minimal)

Add specialized builtins that fuse common patterns into single-pass operations:

```ail
-- Instead of: take(N, flatMap(f, xs))
-- Use: takeFlatMap(N, f, xs)  -- stops after N results
```

**Implementation:**
```go
func takeFlatMapImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    n := args[0].(*eval.IntValue).Value
    f := args[1]
    xs := args[2].(*eval.ListValue).Elements

    var result []eval.Value
    for _, x := range xs {
        if len(result) >= n { break }  // EARLY EXIT
        inner := callFunc(f, x)
        for _, elem := range toSlice(inner) {
            result = append(result, elem)
            if len(result) >= n { break }
        }
    }
    return &eval.ListValue{Elements: result}, nil
}
```

**Pros:**
- Trivial to implement (~50 LOC)
- No new value types, no type system changes
- Fixes the exact OOM pattern DocParse hits

**Cons:**
- Combinatorial explosion: `takeFlatMap`, `takeMap`, `takeFilter`, `takeFlatMapFilter`...
- Doesn't generalize — each new pattern needs a new builtin
- Not composable — users must know which fused variant to use
- Doesn't fix `json.decode` memory usage

**Effort:** ~0.5 day

---

### Option B: Iterator/Generator Protocol (Proper, Medium)

Introduce a `LazyList` value type with pull-based iteration:

```go
// New value type
type LazyListValue struct {
    Next func() (Value, bool)  // pull next element, false = done
}
```

Make `flatMap`, `map`, `filter`, `take` return `LazyListValue` when given lazy input. Add `collect()` to materialize when needed.

```ail
-- These return lazy iterators, no allocation:
let words = flatMap(tokenize, elements)  -- lazy
let capped = take(10000, words)          -- lazy, stops pulling after 10000
let result = collect(capped)             -- materializes only 10000 elements
```

**Implementation sketch:**
```go
func lazyFlatMap(f Value, xs Value) *LazyListValue {
    // If xs is lazy, pull from it; if eager, iterate slice
    outer := toIterator(xs)
    var inner []Value
    var innerIdx int

    return &LazyListValue{
        Next: func() (Value, bool) {
            for {
                if innerIdx < len(inner) {
                    v := inner[innerIdx]
                    innerIdx++
                    return v, true
                }
                x, ok := outer.Next()
                if !ok { return nil, false }
                inner = toSlice(callFunc(f, x))
                innerIdx = 0
            }
        },
    }
}
```

**Type system consideration:** `LazyList(a)` could be a subtype of `List(a)` (auto-collected when a strict list is needed), or a separate type requiring explicit `collect()`.

**Pros:**
- Composable — any chain of map/filter/flatMap/take stays lazy
- Fixes the general problem, not just one pattern
- Natural fit for streaming JSON decode later
- Users write normal code, laziness is automatic

**Cons:**
- New value type that must be handled in every list builtin
- Type system implications (is `LazyList(a)` the same as `List(a)`?)
- Debugging laziness is harder (evaluation order changes)
- Closures capture evaluator state — need to ensure safety
- Medium implementation effort

**Effort:** ~2-3 days

---

### Option C: Compiler-Level flatMap Fusion (Optimal, Hard)

The compiler detects `take(N, flatMap(f, xs))` patterns and rewrites them to early-exit loops. No runtime changes.

```
-- Source:
take(N, flatMap(f, xs))

-- Compiled to equivalent of:
let result = []
for x in xs:
    if len(result) >= N: break
    for y in f(x):
        result = append(result, y)
        if len(result) >= N: break
```

**Pros:**
- Zero user-facing changes
- Optimal performance
- Works with existing code

**Cons:**
- Requires compiler pattern matching infrastructure
- Only handles patterns the compiler knows about
- Complex to implement correctly with nested pipelines
- Harder to debug (compiled code differs from source)

**Effort:** ~5+ days, high complexity

---

### Option D: Memory-Bounded Lists (Safety Net)

Don't fix laziness. Instead, add a hard memory limit that fails cleanly:

```bash
ailang run --max-memory 512MB eval.ail
```

Plus a `chunkedFlatMap(chunkSize, f, xs)` that processes in batches.

**Pros:**
- Clean error instead of laptop OOM
- Easy to implement
- No architectural changes

**Cons:**
- Doesn't fix the problem, just makes failure nicer
- Users must manually restructure code
- Large documents still can't be processed

**Effort:** ~0.5 day for memory limit, ~1 day for chunked operations

---

## Recommendation: Option A Now + Option B Next

**Phase 1 (this sprint):** Option A — fused `takeFlatMap` builtin. Fixes DocParse OOM immediately with minimal risk. Also add `--max-memory` flag (Option D) as a safety net.

**Phase 2 (v0.10 or v1.0):** Option B — proper lazy iterator protocol. This is the right long-term architecture but needs careful type system design.

### Phase 1 Scope

1. **`takeFlatMap(n, f, xs)`** — early-exit flatMap, ~50 LOC
2. **`takeMap(n, f, xs)`** — early-exit map, ~30 LOC
3. **`--max-memory` flag** — `runtime.MemoryLimit()` + clean error, ~20 LOC
4. **Compiler warning** for `take(N, flatMap(...))` suggesting `takeFlatMap`

### Open Questions for Brainstorming

1. **Naming:** `takeFlatMap` vs `flatMapTake` vs `limitFlatMap` vs something else?
2. **Should `take` itself become lazy?** If `take(N, xs)` returned a lazy view that only pulled N elements from an iterator, we'd get fusion for free with any lazy-producing operation.
3. **JSON decode:** Should `json.decodeArray(str, path)` stream elements? This is orthogonal to list laziness but addresses the same OOM.
4. **Is DocParse the only user hitting this?** If so, maybe the fix belongs in DocParse's eval code (process in chunks) rather than in the language.
5. **Codegen implications:** The Go codegen already compiles to eager Go slices. Lazy lists would need a codegen strategy too.
6. **Effect safety:** Lazy evaluation defers side effects. If `flatMap(f, xs)` where `f` has effects, laziness changes WHEN effects execute. This may violate AILANG's explicit effects principle.

---

## DocParse Workaround (Immediate)

While we decide on approach, DocParse can work around the OOM by:
- Keeping the Python eval for large documents (already works, 1.7s for Moby Dick)
- Using `take(N, blocks)` BEFORE `flatMap` (cap input, not output)
- Splitting large JSON into chunks before decode

---

**Document created**: 2026-03-18
**Last updated**: 2026-03-18
