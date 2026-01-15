# AILANG Dogfooding: Gaps Discovered

This document tracks gaps discovered while porting `internal/coordinator/event_formatter.go` to AILANG.

## Critical Issues

### GAP-1: Teaching Prompt Wrong About foldl Lambda Syntax

**Severity:** High (documentation vs implementation mismatch)

**Teaching prompt claims:**
```ailang
-- WRONG (from teaching prompt)
let sum = foldl(\(acc, x). acc + x, 0, [1,2,3,4,5])
```

**Actual working syntax:**
```ailang
-- CORRECT (from existing examples)
let sum = foldl(\acc x. acc + x, 0, [1,2,3,4,5])
```

**Status:** Teaching prompt needs update

---

### GAP-2: Unknown Bug - Same Lambda Syntax Works in examples/runnable but Not Elsewhere

**Severity:** Critical (non-deterministic behavior)

The `\acc x.` lambda syntax works in `examples/runnable/no_loops_fold.ail` but fails with "arity mismatch: 2 vs 1" error when used in any new file, even with identical code.

**Works:**
```bash
ailang check examples/runnable/no_loops_fold.ail  # ✓ passes
```

**Fails (same code in different location):**
```bash
ailang check internal/dashboard_transforms/test_fold.ail  # ✗ fails
```

**Root cause:** Unknown - may be path-dependent behavior in type checker or cached artifacts.

**Workaround:** Use inline `func` syntax instead (see GAP-3)

---

### GAP-3: Must Use Inline func Syntax for foldl (Workaround for GAP-2)

**Severity:** Medium (ergonomics issue)

Lambda syntax `\acc x.` doesn't work reliably with foldl. Must use verbose inline func syntax:

**Doesn't work reliably:**
```ailang
let max = foldl(\acc x. if x > acc then x else acc, 0, xs)
```

**Works:**
```ailang
let max = foldl(func(acc: int, x: int) -> int { if x > acc then x else acc }, 0, xs)
```

**Impact:** More verbose, requires explicit type annotations, less readable.

---

### GAP-4: No Record Width Subtyping

**Severity:** High (API design friction)

Records must match exactly - no width subtyping. Cannot pass `{a: int, b: string, c: bool}` to a function expecting `{a: int}`.

**Doesn't work:**
```ailang
pure func countTurns(events: [{turnNum: int}]) -> int = ...

let events = [{turnNum: 1, streamType: "text", text: "hello"}]
countTurns(events)  -- ✗ Error: field count mismatch: 1 vs 3
```

**Workaround:** Repeat the full record type in every function signature:
```ailang
pure func countTurns(events: [{turnNum: int, streamType: string, text: string}]) -> int = ...
```

**Impact:** Repetitive code, hard to maintain, no modularity for record-based APIs.

**Potential fix:** Implement row polymorphism properly (`{turnNum: int | r}`)

---

## Missing Stdlib Functions

### `repeat(s: string, n: int) -> string`

String repetition - needed for box-drawing UI.

**Workaround:** Implement locally:
```ailang
pure func repeat(s: string, n: int) -> string =
  if n <= 0 then "" else s ++ repeat(s, n - 1)
```

### `maximum(xs: [a]) -> Option[a]` (with Ord constraint)

Find maximum element in list.

**Workaround:** Use foldl with explicit comparison:
```ailang
pure func maxInt(xs: [int]) -> int =
  foldl(func(acc: int, x: int) -> int { if x > acc then x else acc }, 0, xs)
```

---

### GAP-5: Pipeline Cannot Evaluate Standalone Expressions

**Severity:** Medium (affects embed API)

The pipeline returns "empty program" when evaluating standalone expressions like `1 + 2` without module context.

**Doesn't work:**
```go
engine := embed.New(".")
result, err := engine.Eval("1 + 2")  // Error: "empty program"
```

**Root cause:** `internal/pipeline/pipeline_single.go:297-300` returns error when `coreProg.Decls` is empty.

**Workaround:** Use module context or wrap expressions in let bindings:
```go
result, err := engine.Eval("let x = 1 + 2 in x")  // Works
```

**Fix needed:** Pipeline should handle bare expressions by wrapping them in a synthetic let binding.

---

## Summary

| Gap | Severity | Workaround Available | Fix Needed |
|-----|----------|---------------------|------------|
| GAP-1: Teaching prompt wrong | High | N/A (docs) | Update teaching prompt |
| GAP-2: Path-dependent behavior | Critical | Use GAP-3 workaround | Debug type checker |
| GAP-3: Lambda syntax with foldl | Medium | Use inline func | Fix lambda unification |
| GAP-4: No record subtyping | High | Repeat full types | Row polymorphism |
| GAP-5: Standalone expression eval | Medium | Use module context | Fix pipeline |
| Missing `repeat` | Low | Local impl | Add to std/string |
| Missing `maximum` | Low | Use foldl | Add to std/list |

---

## Next Steps

1. [x] Update teaching prompt (GAP-1) - Fixed in prompts/v0.6.5.md
2. [ ] Investigate path-dependent behavior (GAP-2)
3. [ ] Consider adding type aliases to reduce GAP-4 friction
4. [ ] Add `repeat` to std/string
5. [ ] Add `maximum` to std/list
6. [ ] Fix pipeline standalone expression evaluation (GAP-5)
