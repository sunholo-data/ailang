# M-ITERATIVE-LIST: Iterative List Builtins for Large Collections

**Status**: Planned
**Target**: v0.9.3
**Priority**: P0 — blocks DocParse on 5K+ row XLSX processing (performance + recursion limits)
**Estimated**: 1 day (~4 hours implementation + 2 hours testing + 1 hour docs)
**Dependencies**: M-PERF6 (parseElements, completed)
**Source**: DocParse 5K row XLSX takes 60s+ with recursive list ops; 10K+ hits recursion limits

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Iterative Go loops produce identical results to recursive AILANG functions. Same evaluation order (left-to-right). No new nondeterminism. |
| A2: Replayability | 0 | No impact — pure functions produce no trace events. Same inputs → same outputs. |
| A3: Effect Legibility | +1 | All 4 builtins are `pure` — no effects required. Signatures unchanged from AILANG originals. |
| A4: Explicit Authority | 0 | No capabilities needed — pure computation. |
| A5: Bounded Verification | +1 | Type signatures identical to AILANG versions. No new type complexity. Builtin types registered via existing `types.NewBuilder()` pattern. |
| A6: Safe Concurrency | 0 | Single-threaded iteration. No goroutines, no shared state. |
| A7: Machines First | +1 | Go builtins are directly callable, inspectable via `ailang builtins list`. Structured error messages on type mismatch. |
| A8: Minimal Syntax | +1 | Zero new syntax. Existing `map`, `filter`, `foldl` function names unchanged. Transparent delegation from `std/list.ail`. |
| A9: Cost Visibility | +1 | Removes hidden cost: recursive map over 10K elements consumed 10K stack frames. Iterative version uses O(1) stack. |
| A10: Composability | +1 | Functions compose identically — `map(f, filter(p, xs))` works the same. Callback invocation via `FnCallerN` is general-purpose and reusable by future builtins. |
| A11: Structured Failure | +1 | Type errors return structured `fmt.Errorf` messages. `FnCallerN` not wired → clear error string. Callback errors propagate immediately. |
| A12: System Boundary | 0 | No system boundary crossing — pure in-process computation. |

**Net Score: +8** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Iterative loop is deterministic, same left-to-right evaluation order as recursive version
- [x] A3 (Effects): All builtins are `pure`, no effect declarations needed
- [x] A4 (Authority): No capabilities required
- [x] A7 (Machines First): Builtins registered in builtin registry, inspectable via CLI

## Problem Statement

`std/list` functions (`map`, `filter`, `foldl`, etc.) are implemented as AILANG-level recursive functions in `std/list.ail`. Each element consumes one recursion depth level **and** one full evaluator call frame (environment creation, pattern match on `[x, ...rest]`, list destructuring).

**Current State:**
- DocParse now successfully stream-parses 10K+ rows via `parseElements` (M-PERF6)
- Processing 5K rows x 3 cells through recursive `map`/`flatMap` takes **60+ seconds** — the bottleneck is per-element evaluator overhead, not just recursion depth
- At 10K+ rows, hits `RT_REC_003: max recursion depth 10000 exceeded`
- No tail-call optimization exists in the evaluator

**Two bottlenecks:**
1. **Performance**: Each recursive call creates a new environment, pattern-matches `[x, ...rest]`, destructures the list, and copies the tail. For 5K rows x 3 cells = 15K recursive calls, this dominates runtime. DocParse 2000 rows = 28s, 5K rows = 60s+ (should be <5s).
2. **Recursion depth**: At 10K+ elements, hits the hard limit regardless of time.

**Impact:**
- DocParse is unusably slow on real-world XLSX sheets (5K+ rows common)
- Even with `--max-recursion-depth 50000`, performance is still O(n) evaluator frames
- Go-level iteration eliminates both problems: O(1) stack, tight loop with no environment/pattern-match overhead

**Systemic Analysis:**
This is part of a broader pattern: AILANG stdlib functions implemented in AILANG itself are slow at scale due to per-element evaluator overhead. The fix here (Go-level iterative builtins) establishes the pattern for migrating hot-path recursive stdlib functions to Go.

**Example from DocParse:**
```ailang
match parseElements(sheetXml, "row", 5000) {
  Ok(rows) => foldl(\acc. \row. acc ++ [processRow(row)], [], rows)
  -- ^^^ 5K rows: 60s+ due to recursive overhead
  -- ^^^ 10K rows: RT_REC_003 recursion limit hit
}
```

## Goals

**Primary Goal:** Replace recursive `map`, `filter`, `foldl` in `std/list` with Go-level iterative builtins that eliminate both the performance bottleneck (evaluator frame overhead per element) and the recursion depth limit.

**Success Metrics:**
- `map` over 50K elements completes in <1s (vs minutes with recursive version)
- `foldl` over 50K elements completes without recursion limit errors
- DocParse 5K row XLSX processing time drops from 60s+ to <10s
- Existing `std/list` tests pass unchanged (behavioral compatibility)
- `make test` and `make verify-examples` pass
- DocParse XLSX processing works without `--max-recursion-depth` workaround

## Solution Design

### Overview

Add Go builtins that loop iteratively, calling AILANG callbacks via a new `FnCallerN` on `EffContext`. The `std/list.ail` functions delegate to these builtins (same pattern as `member`, `dedup`, etc. at lines 336-348).

### FnCallerN Design

`EffContext.FnCaller` currently supports single-arg calls only:
```go
FnCaller func(fn eval.Value, arg eval.Value) (eval.Value, error)
```

For `foldl` callback `f(acc, x)` which takes 2 args, we need multi-arg support.

**Decision: Add `FnCallerN`** — a multi-arg variant:
```go
FnCallerN func(fn eval.Value, args []eval.Value) (eval.Value, error)
```

This is a small addition to `context.go`. The evaluator already has `CallFunction(fn *FunctionValue, args []Value)` which handles multi-arg application — `FnCallerN` just exposes this through the same callback pattern as `FnCaller`. Future builtins (e.g., `sortBy`, `groupBy`) will also need multi-arg callbacks.

### Builtin Implementations

| Builtin | Signature | Replaces | Callback Args |
|---------|-----------|----------|---------------|
| `_list_map` | `(a -> b, [a]) -> [b]` | `map` in std/list | 1 (element) |
| `_list_filter` | `(a -> bool, [a]) -> [a]` | `filter` in std/list | 1 (element) |
| `_list_foldl` | `((b, a) -> b, b, [a]) -> b` | `foldl` in std/list | 2 (acc, element) |

**Note:** The design doc originally included `_list_forEach`, but `std/list.ail` has no pure `forEach` — only effectful `forEachE`. A pure `forEach` returning `()` is pointless. `forEachE` remains recursive for now (effectful iterative builtins are a separate concern).

### Implementation Sketch

```go
// _list_map: iterative map using FnCaller (single-arg callback)
func listMapImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    fn := args[0]
    list := args[1].(*eval.ListValue)
    result := make([]eval.Value, len(list.Elements))
    for i, elem := range list.Elements {
        val, err := ctx.FnCaller(fn, elem)
        if err != nil {
            return nil, err
        }
        result[i] = val
    }
    return &eval.ListValue{Elements: result}, nil
}

// _list_foldl: iterative fold using FnCallerN (multi-arg callback)
func listFoldlImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    fn := args[0]
    acc := args[1]
    list := args[2].(*eval.ListValue)
    for _, elem := range list.Elements {
        var err error
        acc, err = ctx.FnCallerN(fn, []eval.Value{acc, elem})
        if err != nil {
            return nil, err
        }
    }
    return acc, nil
}
```

### Migration Strategy

**Option 2 (Replace):** Change `std/list.ail` functions to delegate to Go builtins, following the established pattern used by `member`, `dedup`, `intersect`, `union`, `difference` (lines 336-348).

```ailang
-- Before (recursive):
export pure func map[a, b](f: (a) -> b, xs: [a]) -> [b] {
  match xs { [] => [], [x, ...rest] => [f(x)] ++ map(f, rest) }
}

-- After (delegate to Go builtin):
export pure func map[a, b](f: (a) -> b, xs: [a]) -> [b] = _list_map(f, xs)
```

## Files to Modify/Create

**New files:**
- `internal/builtins/list_iterative.go` — Go-level `_list_map`, `_list_filter`, `_list_foldl` (~150 LOC)
- `internal/builtins/list_iterative_test.go` — Unit tests including 50K element stress tests (~200 LOC)

**Modified files:**
- `internal/effects/context.go` — Add `FnCallerN` field + propagate in `WithBudget` (~5 LOC)
- `internal/eval/eval_evaluator.go` — Add `CallValueN` method, wire to `FnCallerN` (~15 LOC)
- `cmd/ailang/main.go` — Wire `FnCallerN` alongside `FnCaller` (~1 LOC)
- `cmd/ailang/serve_api.go` — Wire `FnCallerN` for serve-api mode (~1 LOC)
- `std/list.ail` — Change `map`/`filter`/`foldl` to delegate to Go builtins (~6 LOC)

**Estimated total:** ~380 LOC (implementation + tests)

## Examples

```ailang
import std/list (map, filter, foldl)

-- These all work on 50K+ element lists now:
let big = range(0, 50000)
let doubled = map(\x. x * 2, big)
let evens = filter(\x. x % 2 == 0, big)
let sum = foldl(\acc. \x. acc + x, 0, big)
```

## Success Criteria

- [ ] `_list_map`, `_list_filter`, `_list_foldl` builtins registered and working
- [ ] `FnCallerN` wired in evaluator and serve-api
- [ ] `std/list.ail` delegates `map`, `filter`, `foldl` to Go builtins
- [ ] Stress test: `foldl` over 50K elements completes without recursion limit
- [ ] All existing tests pass (`make test`)
- [ ] Examples verify (`make verify-examples`)
- [ ] Lint clean (`make lint`)
- [ ] CHANGELOG.md updated
- [ ] Golden snapshot updated (builtin count)

## Testing Strategy

**Unit tests** (`list_iterative_test.go`):
- `_list_map` with identity, transform, empty list
- `_list_filter` with always-true, always-false, mixed
- `_list_foldl` with sum, string concat, empty list
- Type error cases (non-function, non-list args)
- 50K element stress test for each builtin
- Callback error propagation

**Integration tests:**
- `std/list.ail` functions work end-to-end via AILANG programs
- Existing `make verify-examples` passes

## Non-Goals

- **Effectful variants** (`mapE`, `filterE`, `foldlE`, `forEachE`): These require effect context propagation through callbacks. Separate design needed.
- **Tail-call optimization**: General TCO in the evaluator is a much larger change. These builtins solve the immediate problem.
- **`foldr`, `flatMap`, `sortBy`**: Less commonly used at scale. Can add later following the same pattern.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Callback error in middle of iteration | Medium | Errors propagate immediately, partial results discarded (consistent with recursive behavior) |
| Type mismatch at runtime (callback returns wrong type) | Low | Downstream code catches type errors. Go builtins don't add new failure modes. |
| `FnCallerN` not wired in some code path | Medium | Nil check with clear error message. Same pattern as existing `FnCaller` nil checks. |

## Related Documents

**Implemented (informs design):**
- `design_docs/implemented/v0_9_0/m-async-io-process.md` — Established `FnCaller` pattern for Go→AILANG callbacks
- M-PERF6 — `parseElements` that produces the 10K+ row lists this feature processes

**Planned (check for overlap):**
- None — this is the only list performance design in v0.9.3

## Future Work

- **Effectful iterative builtins** (`_list_mapE`, `_list_foldlE`): Need effect context threading through callbacks
- **`_list_sortBy`**: Iterative merge sort with 2-arg comparator callback (uses `FnCallerN`)
- **General TCO**: Would make all recursive AILANG functions stack-safe, making these builtins unnecessary long-term

---

**Document created**: 2026-03-16
**Last updated**: 2026-03-16
