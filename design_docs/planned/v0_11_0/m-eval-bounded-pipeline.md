# M-EVAL-BOUNDED-PIPELINE: Fused Bounded Combinators + Memory Ceiling

**Status**: Planned
**Target**: v0.9.4
**Priority**: P1 (blocks DocParse eval on large documents)
**Estimated**: 1 day
**Dependencies**: None
**Milestone ID**: M-EVAL-BOUNDED-PIPELINE
**Created**: 2026-03-18
**Revised**: 2026-03-19
**Source**: DocParse agent — AILANG eval OOMs on Moby Dick (1.4MB JSON, 216K words)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Fused builtins produce the same result as the unfused pipeline (same elements, same order), just without OOM |
| A2: Replayability | 0 | No change — fused builtins are still pure functions |
| A3: Effect Legibility | +1 | For effectful `f`, `takeFlatMap(n, f, xs)` makes the short-circuit boundary explicit in the function name |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | Memory ceiling makes resource usage verifiable |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | AI agents can process larger documents without hitting resource limits |
| A8: Minimal Syntax | 0 | No syntax changes — new builtins only |
| A9: Cost Visibility | +1 | `--max-memory` makes resource limits explicit; fused builtins avoid hidden O(n) allocation |
| A10: Composability | 0 | No change to composability model |
| A11: Structured Failure | +1 | Memory limit produces a clean typed error instead of OS OOM kill |
| A12: System Boundary | 0 | No change |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No violation — fused builtins are deterministic
- [x] A3 (Effects): No violation — short-circuit semantics are explicit in function name
- [x] A4 (Authority): No violation
- [x] A7 (Machines First): Positive — enables larger workloads

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

The OOM is caused by two distinct problems:

**Category A: Pipeline expansion before capping.** Patterns like `take(N, flatMap(f, xs))` where flatMap materializes everything before take can cap. This is the pipeline fusion problem — solved by bounded fused combinators.

**Category B: Oversized source materialization.** `json.decode` on 1.4MB builds a full recursive AST. Holding two (golden + actual) simultaneously doubles memory. This is a streaming parse / chunked decode problem — **not addressed here**, scoped to separate milestone [M-JSON-STREAMING](#future-work).

This milestone addresses Category A only. Category B is explicitly out of scope.

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
| `json.decode` | `internal/builtins/json_decode.go:235` | Full recursive AST in memory (out of scope) |
| `map`, `filter` | `internal/builtins/list_iterative.go` | Also eager, but less problematic alone |

### What Works Fine

- Set operations (hash-based, O(n)) — 34ms for 216K dedup
- Sequential performance — fast once data fits in memory
- Small-to-medium documents (<500KB JSON output)

---

## Goals

**Primary Goal:** Eliminate OOM on large-document eval pipelines by adding bounded fused combinators and a memory ceiling.

**Success Metrics:**
- DocParse Moby Dick eval (flatten → tokenize → take 10000) no longer OOMs
- `takeFlatMap` and `takeMap` avoid materializing unnecessary intermediate results
- Memory limit failures produce a clean, typed error instead of OS OOM kill
- No changes to ordinary list semantics — existing code unchanged
- All existing tests pass

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Status |
|----------|-----------------|-----------|--------|
| Fused builtins vs general laziness | Laziness changes effect semantics; fused builtins are explicit | human | **Decided: fused builtins** |
| Which fused combinators to add | Combinatorial explosion risk | human | **Decided: takeMap, takeFlatMap only** |
| `take` stays strict | Making take lazy creates semantic ambiguity | human | **Decided: strict** |

### Why Not General Lazy Lists (Yet)

Full lazy lists (Option B from the brainstorming doc) are deferred because:

1. **Laziness changes when effects happen.** If `flatMap(f, xs)` is lazy and `f` has effects, then `take(10, flatMap(f, xs))` means "run `f` until 10 results" — not "run `f` for all, take 10." That is a real semantic shift, not just an optimization. For AILANG's explicit/replayable effect model, this is dangerous without precise specification.

2. **Debuggability degrades.** With eager lists, pipeline stages are concrete and evaluation order is obvious. With lazy lists, "why did this effect run now?" becomes harder — exactly the hidden execution model AILANG tries to avoid.

3. **Codegen surface area.** Lazy values need interpreter support, codegen support, builtins understanding two collection modes, and cross-boundary materialization rules. Too much surface for the immediate problem.

4. **The pathological pattern is narrow.** The actual problem is "expansion before capping" — `flatMap` expands, `take` caps too late. Fused combinators solve this precisely without architectural risk.

---

## Solution Design

### Overview

Two complementary changes:
1. **Fused bounded combinators** — `takeMap` and `takeFlatMap` that short-circuit after N results
2. **Memory ceiling** — `--max-memory` flag with clean error on exceeding limit

### 1. Fused Bounded Combinators

#### `takeFlatMap(n, f, xs)` — Early-Exit FlatMap

```go
// internal/builtins/list_bounded.go

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

#### `takeMap(n, f, xs)` — Early-Exit Map

```go
func takeMapImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    n := args[0].(*eval.IntValue).Value
    f := args[1]
    xs := args[2].(*eval.ListValue).Elements

    limit := n
    if limit > len(xs) { limit = len(xs) }
    result := make([]eval.Value, 0, limit)
    for i := 0; i < len(xs) && len(result) < n; i++ {
        result = append(result, callFunc(f, xs[i]))
    }
    return &eval.ListValue{Elements: result}, nil
}
```

#### Effect Semantics (Important)

For effectful `f`, `takeFlatMap(n, f, xs)` evaluates `f` **only for as many input elements as needed to produce the first `n` output elements**. This is an intentional, visible short-circuiting semantic — the bounded behavior is explicit in the function name.

This is semantically distinct from `take(n, flatMap(f, xs))` where `f` runs on ALL elements. Users choosing `takeFlatMap` are opting into the short-circuit.

#### AILANG Usage

```ail
-- BEFORE (OOMs on large input):
let words = take(10000, flatMap(tokenize, elements))

-- AFTER (bounded, no OOM):
let words = takeFlatMap(10000, tokenize, elements)
```

### 2. Memory Ceiling

```go
// cmd/ailang/main.go (flag parsing)
var maxMemory = flag.String("max-memory", "", "Memory limit (e.g., 512MB)")

// internal/eval/memory_limit.go
func SetMemoryLimit(limit string) error {
    bytes, err := parseMemorySize(limit)
    if err != nil { return err }
    debug.SetMemoryLimit(int64(bytes))
    return nil
}
```

Go 1.19+ `debug.SetMemoryLimit` triggers aggressive GC near the limit. Combined with a periodic check, we can produce a clean error:

```
Error: evaluation memory limit exceeded (512MB)
Hint: pipeline materializes large intermediate lists — consider takeFlatMap or takeMap
```

### 3. Compiler Guidance

When the compiler sees `take(N, flatMap(f, xs))`, emit a diagnostic note:

```
note: take(N, flatMap(f, xs)) materializes the full flatMap result before capping.
      Consider takeFlatMap(N, f, xs) for bounded evaluation.
```

This is a note, not a warning — the unfused form is still correct for small inputs.

### Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/builtins/list_bounded.go` | **New** — `takeFlatMap`, `takeMap` implementations | ~80 |
| `internal/builtins/registry.go` | Register new builtins | ~10 |
| `internal/builtins/list_bounded_test.go` | **New** — tests | ~120 |
| `cmd/ailang/main.go` | `--max-memory` flag | ~10 |
| `internal/eval/memory_limit.go` | **New** — memory limit setup + error | ~30 |
| `internal/pipeline/` or relevant compiler phase | Diagnostic note for `take(N, flatMap(...))` | ~20 |

**Total: ~270 LOC** (implementation + tests)

---

## Examples

### Before (OOMs)

```ail
-- DocParse eval: Moby Dick (2861 blocks, ~215K words)
let blocks = flatMap(evalFlattenOneBlock, allBlocks)   -- 215K elements materialized
let words = flatMap(evalTokenize, blocks)               -- 215K strings materialized
let sample = take(10000, words)                         -- too late, already OOM
```

### After (bounded)

```ail
-- Same pipeline, bounded execution:
let words = takeFlatMap(10000, \block.
    flatMap(evalTokenize, evalFlattenOneBlock(block)),
    allBlocks
)
-- Only processes blocks until 10000 words collected
```

### Memory ceiling

```bash
# Clean failure instead of OOM kill:
$ ailang run --max-memory 512MB eval.ail
Error: evaluation memory limit exceeded (512MB)
Hint: pipeline materializes large intermediate lists — consider takeFlatMap or takeMap
```

---

## Success Criteria

- [ ] `takeFlatMap(10000, tokenize, mobyDickBlocks)` completes without OOM
- [ ] `takeFlatMap` processes only as many input elements as needed (verified with counter)
- [ ] `takeMap(100, f, xs)` where `len(xs) == 100000` only calls `f` 100 times
- [ ] `--max-memory 128MB` produces clean error on large eval, not OS OOM
- [ ] Compiler note emitted for `take(N, flatMap(f, xs))` pattern
- [ ] All existing tests pass — no change to ordinary list semantics
- [ ] Lint clean
- [ ] `make verify-examples` passes

---

## Testing Strategy

**Unit tests:**
- `takeFlatMap` with pure functions — correct results, correct element count
- `takeFlatMap` where N > total elements — returns all (no error)
- `takeFlatMap` with N=0 — returns empty list
- `takeMap` — same coverage
- Memory limit parsing — valid sizes, invalid input, edge cases

**Integration tests:**
- Pipeline that would OOM with unfused `take(N, flatMap(...))` succeeds with `takeFlatMap`
- `--max-memory` flag produces structured error

**Manual verification:**
- DocParse Moby Dick eval with `takeFlatMap` — no OOM, correct results

---

## DocParse Workaround (Immediate)

While this milestone is implemented, DocParse can work around the OOM by:
- Keeping the Python eval for large documents (already works, 1.7s for Moby Dick)
- Using `take(N, blocks)` BEFORE `flatMap` (cap input, not output)
- Splitting large JSON into chunks before decode

---

## Non-Goals

- **General lazy lists / iterators** — Deferred until effect semantics, iterator typing, and codegen strategy are designed. See [Future Work](#future-work).
- **Streaming JSON decode** — Separate milestone (M-JSON-STREAMING). This milestone fixes pipeline expansion only.
- **Compiler-level fusion** — Requires optimization infrastructure not yet in place. The compiler note guides users to fused builtins instead.
- **`takeFilter`** — Not adding yet. Only `takeMap` and `takeFlatMap` for now. Add more fused forms only when proven needed.
- **Making `take` itself lazy** — Creates semantic ambiguity unless input is already iterator-typed. Keep `take` strict.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `takeFlatMap` called with effectful `f` — users surprised by short-circuit | Med | Document explicitly; effect boundary is visible in function name |
| Combinatorial growth of fused builtins | Low | Strict scope: only `takeMap` and `takeFlatMap`. Review before adding more |
| `--max-memory` false positives on legitimate large workloads | Low | Default is no limit; opt-in only |
| DocParse still OOMs due to JSON decode (Category B) | Med | Acknowledged as separate milestone; Python eval is immediate workaround |

---

## Future Work

### M-JSON-STREAMING (Next Priority)

Streamed / bounded JSON ingestion — addresses Category B (oversized source materialization):

- `json.decodeArrayStream` — yields elements one at a time
- `json.foldArray(jsonStr, path, acc, f)` — fold over array without materializing
- `json.takeArray(n, jsonStr, path)` — decode only first N elements

This will likely buy more real-world headroom than general lazy lists, because the Moby Dick scenario has two full recursive ASTs in memory simultaneously.

### Bounded Reducers (Future)

Short-circuit fold operations for "scan until enough evidence" patterns:

- `foldWhile(pred, acc, f, xs)` — fold until predicate is false
- `any(pred, xs)` / `all(pred, xs)` — short-circuit boolean reducers

These are more aligned with AILANG's explicit evaluation model than full laziness.

### Iterator Protocol (Deferred — Requires Design)

A constrained iterator protocol, only if effect semantics can be preserved cleanly:

- Explicit `Iter(a)` or `Seq(a)` type (NOT a subtype of `List(a)`)
- Explicit conversions: `List.toIter(xs)`, `Iter.collect(it)`
- Lazy operations: `Iter.take(n, it)`, `Iter.flatMap(f, it)`
- Separate from list operations — no hidden laziness under `List` API

**Prerequisites before designing:**
1. What happens with effectful functions in lazy pipelines?
2. Is iteration replayable?
3. How does Go codegen represent iterators?
4. Where do materialization boundaries become explicit?

---

## Related Documents

- [M-EVAL-LAZY-PIPELINE](./m-eval-lazy-pipeline.md) — Original brainstorming doc (this milestone supersedes Phase 1)
- [M-HASH-COLLECTIONS Phase 1](./m-hash-collections-phase1-sprint-plan.md) — Hash set ops that fixed the O(n^2) dedup OOM
- [M-SERVE-API-CONCURRENCY](./m-serve-api-concurrency.md) — Per-request evaluator (related perf work)
- [M-PERF4: Bytecode Interpreter](../v1_0_0/m-perf4-bytecode-interpreter.md) — Future perf architecture

---

**Document created**: 2026-03-18
**Last updated**: 2026-03-19
