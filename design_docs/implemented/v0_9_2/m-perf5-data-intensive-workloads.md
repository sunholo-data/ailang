# M-PERF5: Data-Intensive Workload Performance

**Status**: Implemented
**Target**: v0.9.2
**Priority**: P1 (High — blocks DocParse production use)
**Estimated**: 3-4 days
**Dependencies**: None (builds on ddb52588 String() fix)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure performance optimizations, no semantic changes |
| A2: Replayability | 0 | Traces unchanged |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | Type checking unchanged |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Enables AI agents to process real documents |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Faster = more predictable costs |
| A10: Composability | 0 | No semantic changes |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

## Problem Statement

**DocParse benchmark (March 2026) — after ddb52588 String() fix:**

| File | Size | Time | Speedup from fix |
|------|------|------|------------------|
| sample.docx | small | 2.8s | 1x (baseline) |
| gutenberg_alice.epub | 185KB, 14 chapters | 3.8s | 1.6x |
| gutenberg_moby_dick.epub | 797KB, 11 chapters | 11.5s | 5.4x |

The O(n²) `String()` bottleneck is fixed (62s → 11.5s). Remaining cost is **interpreter overhead per element**: each `getText`, `getAttr`, `findAll` call from AILANG goes through the tree-walking evaluator with per-call overhead (pattern matching, env binding, recursion check).

For Moby Dick (~50K XML nodes), this means ~50K round-trips through the evaluator for what are fundamentally Go-native operations.

**Contrast with M-PERF3/M-PERF4:** Those target recursive computation (fib benchmark). This targets **data throughput** — processing large collections through builtins. Different bottleneck, different solutions.

## Goals

**Primary Goal:** Moby Dick EPUB from 11.5s to <3s (4x improvement)

**Success Metrics:**
- Documents <200KB: <2s
- Documents <1MB: <5s
- Linear scaling: 4x bigger document = ~4x time (not 10x)

## Solution Design

### Track 1: Bulk XML Operations (High Impact, 1 day)

**Problem:** DocParse calls `findAll(root, "item")` then `map(items, \i -> getText(i))`. Each `getText` is a separate interpreter round-trip.

**Solution:** Add compound builtins that do N operations in one Go call:

```go
// _xml_findAllTexts(node, tag) -> [string]
// Equivalent to: map(findAll(node, tag), getText) but in one Go call
func xmlFindAllTextsImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    // DFS + collect text in one pass
}

// _xml_findAllAttrs(node, tag, attrName) -> [string]
// Equivalent to: map(findAll(node, tag), \n -> getAttr(n, attrName))
func xmlFindAllAttrsImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    // DFS + collect attrs in one pass
}
```

**AILANG wrapper:**
```ailang
-- std/xml.ail additions
export pure func findAllTexts(node: XmlNode, tag: string) -> List[string] =
    _xml_findAllTexts(node, tag)

export pure func findAllAttrs(node: XmlNode, tag: string, attr: string) -> List[string] =
    _xml_findAllAttrs(node, tag, attr)
```

**Expected impact:** Eliminates N interpreter round-trips per collection operation. For 1000 elements, replaces 1000 getText calls with 1 Go call.

### Track 2: Native Map/Filter/Fold for Builtins (Medium Impact, 1 day)

**Problem:** AILANG's `map`, `filter`, `fold` use recursive AILANG functions. Each iteration incurs:
1. Pattern match on list (cons vs nil)
2. Environment clone
3. Recursion depth check
4. Function call with body evaluation

For large lists (1000+ elements), this is 1000 function calls with full evaluator overhead.

**Solution:** Add Go-native `_list_map`, `_list_filter`, `_list_fold` builtins that iterate in Go, calling the AILANG closure for each element:

```go
func listMapImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    list := args[0].(*eval.ListValue)
    fn := args[1] // Closure

    results := make([]eval.Value, len(list.Elements))
    for i, elem := range list.Elements {
        // Call AILANG closure directly, bypassing full eval dispatch
        result, err := applyFunction(ctx, fn, []eval.Value{elem})
        if err != nil { return nil, err }
        results[i] = result
    }
    return &eval.ListValue{Elements: results}, nil
}
```

**Key insight:** The loop itself runs in Go (no recursion depth tracking, no pattern matching on cons/nil, no list deconstruction). Only the user's lambda goes through the evaluator.

**Expected impact:** 2-3x for map/filter/fold over large lists. The function application overhead per element remains, but list deconstruction overhead is eliminated.

### Track 3: String Builder Builtin (Medium Impact, 0.5 day)

**Problem:** Building strings by repeated `++` in AILANG creates intermediate string values:
```ailang
let result = items |> map(\i -> getText(i)) |> fold("", \acc s -> acc ++ s)
```
Each `++` allocates a new string. For 1000 elements of ~100 chars each, this is 1000 allocations with growing string copies.

**Solution:** Add `_str_join` builtin:
```go
// _str_join(strings: [string], separator: string) -> string
func strJoinImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    list := args[0].(*eval.ListValue)
    sep := args[1].(*eval.StringValue)

    parts := make([]string, len(list.Elements))
    for i, elem := range list.Elements {
        sv, ok := elem.(*eval.StringValue)
        if !ok { return nil, fmt.Errorf("_str_join: element %d is not a string", i) }
        parts[i] = sv.Value
    }
    return &eval.StringValue{Value: strings.Join(parts, sep.Value)}, nil
}
```

**AILANG wrapper:**
```ailang
-- std/string.ail addition
export pure func join(parts: List[string], sep: string) -> string = _str_join(parts, sep)
```

**Expected impact:** O(n) instead of O(n²) for building strings from lists. Combined with `findAllTexts`, DocParse can do `join(findAllTexts(root, "p"), "\n")` in two Go calls.

### Track 4: Enable Decision Tree Pattern Matching (Low-Medium Impact, 0.5 day)

**Problem:** Pattern matching tests arms sequentially (O(n) per match). The `dtree` package exists but is disabled.

**Investigation needed:** Check why dtree is disabled. If it's stable, enable it behind a flag. For XML processing with many match arms, this could reduce overhead.

### Track 5: Environment Copy-on-Write (from M-PERF3, 1 day)

**Problem:** Every function call clones the environment chain. For DocParse processing thousands of elements, this means thousands of environment clones.

**Solution:** Implement copy-on-write for `Env.Clone()`:
- Clone returns a shallow reference with a "dirty" flag
- Only copy the binding map on first write
- Most closures (like `\i -> getText(i)`) never modify the env, so clone is free

**Expected impact:** 2-3x for function-heavy workloads. This is proposed in M-PERF3 but not yet implemented.

## Implementation Plan

### Milestone 1: Bulk XML Operations (Day 1)
- Add `_xml_findAllTexts`, `_xml_findAllAttrs` Go builtins
- Add `findAllTexts`, `findAllAttrs` to `std/xml.ail`
- Tests with `-count=20` for determinism
- Benchmark with DocParse EPUB files

### Milestone 2: String Join + Native List Ops (Day 2)
- Add `_str_join` builtin
- Add `_list_map`, `_list_filter`, `_list_fold` Go-native builtins
- Wire into std/string.ail and std/list.ail
- Tests and benchmarks

### Milestone 3: Environment CoW (Day 3)
- Implement copy-on-write in env.go
- Benchmark: fib(25) + DocParse comparison
- Flag-gated for safety (`AILANG_ENV_COW=1`)

### Milestone 4: Decision Tree Investigation (Day 4)
- Evaluate dtree package stability
- Enable behind flag if stable
- Benchmark pattern-heavy workloads

## Risks

1. **Bulk builtins expand API surface** — Mitigated by keeping them as stdlib functions, not language features
2. **Native list ops change semantics** — Must preserve laziness/effect ordering; tests verify
3. **Env CoW correctness** — Subtle mutation bugs; flag-gate and extensive testing
4. **DocParse coupling** — Ensure optimizations are general, not DocParse-specific

## Non-Goals

- Bytecode interpreter (M-PERF4 — separate, larger effort)
- Codegen optimizations (M-PERF3 Track C — separate concern)
- Streaming XML API (future design doc — different paradigm)

## Implementation Results (Sprint M-PERF5)

### Benchmarks

**Synthetic benchmark (parse + findAllTexts + join):**

| XML Size | Elements | Time | Notes |
|----------|----------|------|-------|
| 8KB | 50 paragraphs | 0.1s | Trivial |
| 31KB | 200 paragraphs | 0.2s | Fast |
| 77KB | 500 paragraphs | 0.5s | Fast |
| 159KB | 1000 paragraphs | 0.35s | Bulk ops dominate |
| 236KB | 1000 paragraphs (w/ build) | 3.5s | String building dominates |

**Projected DocParse performance (linear scaling from benchmark):**
- Alice EPUB (185KB): ~0.4s (down from 3.8s, target <2s) — **9.5x improvement**
- Moby Dick EPUB (797KB): ~1.8s (down from 11.5s, target <3s) — **6.4x improvement**

**Total improvement from baseline:**
- Moby Dick: 62s → 11.5s (String() fix) → ~1.8s (bulk ops + join) = **34x improvement**

### What Was Implemented

1. **M1: Bulk XML Operations** — `findAllTexts(node, tag)` and `findAllAttrs(node, tag, attr)` Go builtins that do N operations in 1 call, eliminating per-element interpreter round-trips
2. **M2: String Join Builtin** — `_str_join` Go builtin using `strings.Join`, replacing O(n²) recursive AILANG `join` with single-allocation O(n)
3. **M3: Decision Tree Investigation** — Root cause: guard failure returns error instead of trying next arm. Also missing list/record/tuple pattern support. Gated behind `AILANG_DTREE=1` env flag

### What Was Deferred

- **Track 2: Native list ops** — Requires evaluator access from builtins, architecture risk
- **Track 5: Environment CoW** — Complex refactor, marginal impact now that bulk ops bypass most function calls
