# M-DX3: Lambda DX Fixes (Comparison Operators + show(Bool))

**Status**: Planned
**Target**: v0.3.17
**Priority**: P0 - Critical (Blocks lambda examples)
**Estimated**: 1 day (4-6 hours)
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes need for workarounds (temporary bindings, avoiding comparisons) |
| Preserve Semantic Clarity | + | +1 | Comparisons work as expected in lambda bodies |
| Increase Determinism | 0 | 0 | No change (fixes regression, restores expected behavior) |
| Lower Token Cost | + | +1 | Enables ~20% more lambda patterns (predicates, guards, bool operations) |
| **Net Score** | | **+3** | **Decision: Move forward (P0 bug fixes)** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [Lambda Expressions Refactor DX Issues](../../implemented/v0_3_16/lambda-expressions-example-refactor.md#developer-experience-issues--improvement-opportunities)

## Problem Statement

During the lambda expressions example refactor (v0.3.16), we discovered two **critical bugs** that block fundamental lambda patterns:

**Current State:**
1. **Comparison operators in lambda bodies fail type checking:**
   ```ailang
   let max = \x. \y. if x > y then x else y in
   max(10)(20)
   -- Error: "Operator '>' has no implementation for type Bool"
   ```
   - Root cause: Type inference backward-propagates Bool from `if` condition to operands
   - Parameters `x`, `y` are incorrectly inferred as Bool instead of Int/Float
   - Blocks: max/min, abs, predicates, guards, sorting

2. **`show` builtin doesn't work for Bool:**
   ```ailang
   show(42)      -- ✅ Works: "42"
   show(true)    -- ❌ Fails: "cannot unify type constructors: string vs bool"
   ```
   - Root cause: Bool case missing from type-directed show lowering
   - Inconsistent: works for Int, Float, String, List, Record but not Bool
   - Blocks: displaying boolean results, debugging conditionals

**Impact:**
- **Who:** All developers writing lambda examples, tutorials, and real code
- **Severity:** P0 - Blocks basic functional programming patterns
- **Workarounds:** Painful (avoid comparisons, avoid showing booleans)
- **Evidence:** Lambda examples refactor had to exclude ~20% of planned content

**From the implementation report:**
> "These two issues blocked basic examples like max/abs and made it impossible to demonstrate boolean logic in lambdas. Combined they prevented ~20% of desired lambda example patterns and added 50% time overhead to the sprint."

## Goals

**Primary Goal:** Fix type inference for comparison operators in lambda bodies and enable `show(bool)` to unblock lambda examples.

**Success Metrics:**
- ✅ Comparison operators (`>`, `<`, `>=`, `<=`, `==`, `!=`) work correctly in lambda bodies
- ✅ Type inference constrains operands to Num (not Bool) for comparison operators
- ✅ `show(true)` and `show(false)` return `"true"` and `"false"`
- ✅ All 6 lambda example files from v0.3.16 can be extended with blocked patterns
- ✅ Zero regressions in existing tests

## Solution Design

### Overview

**Two independent bug fixes:**
1. **Fix comparison operator type inference** (internal/types/)
2. **Add Bool case to show builtin** (internal/builtins/show.go)

Both are surgical fixes to existing systems - no new architecture needed.

### Architecture

**Component 1: Comparison Operator Type Inference Fix**

**Current (broken) flow:**
```
1. Parse: \x. if x > 0 then -x else x
2. Infer: if requires Bool condition
3. Backward propagate: x > 0 must be Bool
4. ERROR: Infer x as Bool (wrong!)
5. ERROR: > has no implementation for Bool
```

**Fixed flow:**
```
1. Parse: \x. if x > 0 then -x else x
2. Infer comparison: x > 0 yields Bool, constrains x to Num
3. Infer if: condition is Bool ✓
4. Infer parameter: x is constrained to Num (Int or Float)
5. SUCCESS: Operator > works on Num types
```

**Where to fix:** `internal/types/typechecker_operators.go` or `internal/types/infer.go`
- Ensure comparison operators (`>`, `<`, etc.) constrain operands to Num **before** result type Bool flows backward
- Use constraint ordering: operand types → operator → result type

**Component 2: show(Bool) Support**

**Current (broken) flow:**
```
1. Call: show(true)
2. Type check: show :: a -> string, true :: bool
3. Type-directed lowering: Check type head
4. Cases: Int? No. Float? No. String? No. List? No. Record? No. Bool? MISSING
5. ERROR: Type mismatch
```

**Fixed flow:**
```
1. Call: show(true)
2. Type check: show :: a -> string, true :: bool ✓
3. Type-directed lowering: Check type head
4. Cases: ... Bool? YES → use show_Bool builtin
5. SUCCESS: Returns StringValue("true")
```

**Where to fix:** `internal/builtins/show.go` and `internal/pipeline/op_lowering.go`
- Add Bool case to type-head switch in show lowering
- Route to existing `show_Bool` implementation
- Reuse existing test infrastructure

### Implementation Plan

**Phase 1: Comparison Operators Fix** (~2-3 hours)

- [ ] **Task 1.1:** Analyze current comparison operator type inference (30min)
  - Trace through `\x. x > 0` in `internal/types/infer.go`
  - Identify where Bool back-propagates to operands
  - Document constraint flow in code comments

- [ ] **Task 1.2:** Implement constraint ordering fix (1h)
  - Ensure comparison operators constrain operands to Num first
  - Add constraint: `>` :: Num a => a -> a -> Bool
  - Prevent Bool from flowing backward to operands
  - Test locally with `let max = \x. \y. if x > y then x else y`

- [ ] **Task 1.3:** Add comprehensive tests (1h)
  - Test all comparison operators: `>`, `<`, `>=`, `<=`, `==`, `!=`
  - Test in lambda bodies: `\x. x > 0`, `\x. \y. x < y`
  - Test with nested lambdas: `\f. \x. f(x) > 0`
  - Test guards: `\x. if x > 0 then "pos" else "neg"`
  - Test combined: `\x. x > 0 && x < 100`
  - File: `internal/types/typechecker_operators_test.go`

- [ ] **Task 1.4:** Add regression tests (30min)
  - Ensure existing `if` expressions still work
  - Test non-lambda comparisons: `if 10 > 5 then ...`
  - Test let-bound comparisons: `let b = x > y in if b then ...`

**Phase 2: show(Bool) Fix** (~0.5-1 hour)

- [ ] **Task 2.1:** Add Bool case to show lowering (20min)
  - Modify `internal/pipeline/op_lowering.go`
  - Add case for `types.IsBool(argType)` in type-head switch
  - Route to `show_Bool` builtin (already exists)

- [ ] **Task 2.2:** Add tests (20min)
  - Test `show(true)` → `"true"`
  - Test `show(false)` → `"false"`
  - Test in expressions: `print("Result: " ++ show(x > 0))`
  - File: `internal/builtins/show_test.go`

- [ ] **Task 2.3:** Verify existing show_Bool still works (10min)
  - Ensure explicit `show_Bool(true)` calls unchanged
  - Check examples/snippets/typeclasses.ail

**Phase 3: Integration & Documentation** (~1 hour)

- [ ] **Task 3.1:** Test with lambda examples (20min)
  - Add max/min/abs examples to `lambdas_basic.ail`
  - Add predicate examples: `is_positive`, `is_negative`
  - Add guard examples with boolean display

- [ ] **Task 3.2:** Update LIMITATIONS.md (20min)
  - Remove any workarounds for these issues
  - Document that comparisons now work in lambdas
  - Add Y-combinator limitation (from v0.3.16 findings)

- [ ] **Task 3.3:** Update CHANGELOG (10min)
  - Add M-DX3 entry with fixes
  - Include before/after examples
  - Note which lambda patterns are now enabled

- [ ] **Task 3.4:** Verify no regressions (10min)
  - Run full test suite: `make test`
  - Run linting: `make lint`
  - Test all 6 lambda examples from v0.3.16

### Files to Modify/Create

**Modified files:**
- `internal/types/infer.go` or `internal/types/typechecker_operators.go` - Fix comparison constraint ordering (~50 LOC)
- `internal/pipeline/op_lowering.go` - Add Bool case to show lowering (~10 LOC)
- `docs/LIMITATIONS.md` - Update with Y-combinator note (~20 LOC)
- `CHANGELOG.md` - Add M-DX3 entry (~30 LOC)

**New test files:**
- `internal/types/typechecker_operators_test.go` - Comparison operator tests (~200 LOC)
- Tests in existing `internal/builtins/show_test.go` - Bool show tests (~50 LOC)

**Modified example files:**
- `examples/snippets/showcase/lambdas_basic.ail` - Add max/abs examples (~20 LOC)
- `examples/snippets/showcase/lambdas_advanced.ail` - Add predicate examples (~20 LOC)

**Total new/modified LOC:** ~400 LOC (mostly tests)

## Examples

### Example 1: Comparison Operators in Lambda Bodies

**Before (BROKEN):**
```ailang
-- Tried to write:
let max = \x. \y. if x > y then x else y in
print(show(max(10)(20)))

-- Error: "Operator '>' has no implementation for type Bool"
-- Workaround: Couldn't write max/min/abs at all
```

**After (FIXED):**
```ailang
-- Works perfectly:
let max = \x. \y. if x > y then x else y in
let min = \x. \y. if x < y then x else y in
let abs = \x. if x < 0 then -x else x in

print("max(10, 20) = " ++ show(max(10)(20)))  -- "max(10, 20) = 20"
print("min(10, 20) = " ++ show(min(10)(20)))  -- "min(10, 20) = 10"
print("abs(-5) = " ++ show(abs(-5)))           -- "abs(-5) = 5"
```

### Example 2: show(Bool) for Predicates

**Before (BROKEN):**
```ailang
-- Tried to write:
let is_positive = \x. x > 0 in
print("is_positive(5) = " ++ show(is_positive(5)))

-- Error: "cannot unify type constructors: string vs bool"
-- Workaround: Avoided boolean results entirely
```

**After (FIXED):**
```ailang
-- Works perfectly:
let is_positive = \x. x > 0 in
let is_negative = \x. x < 0 in
let is_zero = \x. x == 0 in

print("is_positive(5) = " ++ show(is_positive(5)))    -- "is_positive(5) = true"
print("is_negative(-3) = " ++ show(is_negative(-3)))  -- "is_negative(-3) = true"
print("is_zero(0) = " ++ show(is_zero(0)))            -- "is_zero(0) = true"
```

### Example 3: Guards and Combined Predicates

**Before (BROKEN):**
```ailang
-- Couldn't write range checks:
let in_range = \x. x > 0 && x < 100 in  -- ERROR
```

**After (FIXED):**
```ailang
-- Guards work:
let classify = \x.
  if x < 0 then "negative"
  else if x == 0 then "zero"
  else "positive"
in
print("classify(-5) = " ++ classify(-5))  -- "classify(-5) = negative"

-- Combined predicates work:
let in_range = \x. x > 0 && x < 100 in
print("in_range(50) = " ++ show(in_range(50)))    -- "in_range(50) = true"
print("in_range(150) = " ++ show(in_range(150)))  -- "in_range(150) = false"
```

## Success Criteria

**Functional:**
- [ ] `let max = \x. \y. if x > y then x else y` type checks successfully
- [ ] `max(10)(20)` evaluates to `20`
- [ ] `show(true)` returns `"true"`
- [ ] `show(false)` returns `"false"`
- [ ] `show(5 > 3)` returns `"true"`
- [ ] All comparison operators work in lambda bodies: `>`, `<`, `>=`, `<=`, `==`, `!=`
- [ ] Boolean results can be displayed with `show`

**Testing:**
- [ ] All new tests passing (>20 test cases)
- [ ] Zero regressions in existing test suite
- [ ] All 6 lambda examples from v0.3.16 still pass
- [ ] Extended lambda examples with new patterns pass

**Documentation:**
- [ ] CHANGELOG.md updated with M-DX3 entry
- [ ] LIMITATIONS.md updated (remove workarounds, add Y-combinator note)
- [ ] Lambda examples enhanced with max/min/abs/predicates

## Testing Strategy

**Unit tests:**
- Comparison operators in lambda bodies (all 6 operators)
- Nested lambdas with comparisons
- Combined predicates (&&, ||)
- show(bool) with literals and expressions

**Integration tests:**
- Run all lambda examples from v0.3.16
- Add new lambda patterns (max, min, abs, is_positive, in_range)
- Test REPL behavior matches module behavior

**Regression tests:**
- Existing if-expressions still work
- Existing show(int/float/string/list/record) unchanged
- Non-lambda comparisons still work

**Manual testing:**
- Run `ailang repl` and test interactively:
  ```
  λ> let max = \x. \y. if x > y then x else y
  λ> max(10)(20)
  20
  λ> show(true)
  "true"
  λ> show(5 > 3)
  "true"
  ```

## Non-Goals

**Not in this feature:**
- Type annotations for lambdas - Deferred to future work (would help but not required)
- Better parse error messages - Separate issue (M-DX3 task 3)
- Block expression improvements - Separate sprint (M-DX4)
- String interpolation - Separate sprint (M-DX4)
- REPL/module parity - Separate sprint (M-DX4)

**Rationale:** Focus on the two P0 blockers that prevent basic lambda patterns. Other DX improvements are important but not blocking.

## Timeline

**Day 1** (4-6 hours):
- Morning: Phase 1 (comparison operators fix + tests) - 2-3h
- Afternoon: Phase 2 (show bool fix + tests) - 0.5-1h
- End of day: Phase 3 (integration + docs) - 1h

**Total: ~4-6 hours in 1 day**

**Stretch:** If completed early, start documenting other DX issues for M-DX4

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Type inference fix breaks existing code | High | Comprehensive regression tests; test all existing examples |
| Constraint ordering too complex | Medium | Start with simplest fix; add complexity only if needed |
| show(Bool) conflicts with explicit show_Bool | Low | Keep show_Bool working; add tests for both paths |
| Takes longer than 1 day | Medium | Both fixes are independent; can ship incrementally if needed |

## References

- **Primary:** [Lambda Expressions Refactor DX Issues](../../implemented/v0_3_16/lambda-expressions-example-refactor.md#developer-experience-issues--improvement-opportunities)
- **Type system:** `internal/types/infer.go`, `internal/types/typechecker_operators.go`
- **Builtins:** `internal/builtins/show.go`, `internal/pipeline/op_lowering.go`
- **Related:** M-DX2 (type-guided operator lowering) - provides CoreTypeInfo infrastructure
- **Prior art:** M-DX1 (builtin development) - established testing patterns

## Future Work

**M-DX4: Effectful Expression Ergonomics** (next sprint):
- Block expressions as first-class values
- Do-notation or implicit unit binding
- String interpolation
- REPL/module parity (--script mode)

**Later:**
- Type annotations for lambdas (`\x: Int. x + 1`)
- Better error messages for type inference failures
- Refinement types for comparisons (`x > 0` → proof that `x` is positive)

---

**Document created**: 2025-10-21
**Last updated**: 2025-10-21
