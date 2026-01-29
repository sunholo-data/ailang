# M-BUG-LETREC-SINGLE-CALL: Letrec with Single Recursive Call Fails

**Status**: Planned
**Target**: v0.7.2
**Priority**: P1 - Medium
**Estimated**: 4 hours
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Fix ensures deterministic letrec evaluation |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effects involved |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Type checking already passes; runtime should match |
| A6: Safe Concurrency | 0 | No concurrency |
| A7: Machines First | +1 | Enables AI to use standard recursive patterns |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Letrec should compose with any recursive pattern |
| A11: Structured Failure | 0 | Error is already structured |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: ✅ Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Fix improves machine usability

## Problem Statement

**Bug Report:** Letrec expressions with exactly ONE recursive call in the body fail at runtime with "missing dictionary method: prelude::Ord::Bool::lt", while letrec with TWO recursive calls works correctly.

**Current State:**
- Fibonacci (double recursive) works: `letrec fib = \n. if n < 2 then n else fib(n-1) + fib(n-2)`
- Factorial (single recursive) fails: `letrec fact = \n. if n < 2 then 1 else n * fact(n-1)`
- Type checking passes in both cases - bug is in evaluator/runtime
- Error message references `Bool::lt` despite no boolean comparison in source

**Reproduction:**

```ailang
-- WORKS (two recursive calls)
let fib_result = letrec fib = \n. if n < 2 then n else fib(n - 1) + fib(n - 2) in fib(10);
-- Output: 55

-- FAILS (one recursive call)
let fact_result = letrec factorial = \n. if n < 2 then 1 else n * factorial(n - 1) in factorial(5);
-- Error: missing dictionary method: prelude::Ord::Bool::lt
```

**Impact:**
- Common recursive patterns (factorial, sum, power, countdown) are unusable
- Users must use awkward double-call workarounds
- Surprising behavior - pattern that should work doesn't

## Goals

**Primary Goal:** Fix letrec evaluation so single-recursive-call patterns work identically to double-recursive-call patterns.

**Success Metrics:**
- `letrec factorial = \n. if n < 2 then 1 else n * factorial(n-1) in factorial(5)` returns `120`
- `letrec sum = \n. if n < 1 then 0 else n + sum(n-1) in sum(100)` returns `5050`
- All `examples/runnable/letrec_recursion.ail` patterns work
- Existing fib (double-call) continues working
- `make test` passes

## Solution Design

### Overview

The bug is likely in dictionary passing or type specialization during letrec evaluation. The bogus `Bool::lt` reference suggests type information is being corrupted or misrouted.

### Diagnosis Approach

1. Compare Core/lowered AST for single vs double recursive calls
2. Trace dictionary passing during evaluation
3. Check if monomorphization handles these cases differently
4. Look for conditional paths that behave differently based on recursive call count

### Suspected Locations

**Primary suspects:**
- `internal/eval/eval_expressions.go:evalCoreLetRec` - Letrec evaluation
- `internal/link/linker.go` - Dictionary linking
- `internal/elaborate/elaborate.go` - Surface→Core elaboration for letrec

**Secondary (monomorphization):**
- `internal/types/mono.go` - Specialization logic
- `internal/types/unify.go` - Type unification

### Implementation Plan

**Phase 1: Diagnosis** (~2 hours)
- [ ] Add DEBUG_LETREC flag for detailed evaluation tracing
- [ ] Compare Core AST for working fib vs broken factorial
- [ ] Trace dictionary arguments during evaluation
- [ ] Identify where Bool::lt is incorrectly selected

**Phase 2: Fix** (~1.5 hours)
- [ ] Implement fix based on diagnosis
- [ ] Verify single-call patterns work
- [ ] Verify double-call patterns still work
- [ ] Remove debug instrumentation

**Phase 3: Testing** (~0.5 hours)
- [ ] Add unit tests for single-call letrec patterns
- [ ] Update `examples/runnable/letrec_recursion.ail` with full examples
- [ ] Run `make test` and `make verify-examples`
- [ ] Update CHANGELOG.md

### Files to Modify

**Diagnostic (temporary):**
- `internal/eval/eval_expressions.go` - Add debug tracing (~20 LOC)

**Fix (TBD based on diagnosis):**
- Likely one of: `eval_expressions.go`, `linker.go`, or `elaborate.go`
- Estimated: ~30-50 LOC

**Testing:**
- `internal/eval/recursion_test.go` - Add single-call test cases (~50 LOC)
- `examples/runnable/letrec_recursion.ail` - Restore full examples

## Examples

### Example 1: Factorial (currently broken)

**Before (fails):**
```ailang
let fact_result = letrec factorial = \n.
  if n < 2 then 1 else n * factorial(n - 1)
in factorial(5);
-- Error: missing dictionary method: prelude::Ord::Bool::lt
```

**After (should work):**
```ailang
let fact_result = letrec factorial = \n.
  if n < 2 then 1 else n * factorial(n - 1)
in factorial(5);
-- Output: 120
```

### Example 2: Sum (currently broken)

**Before:**
```ailang
let sum_result = letrec sum = \n.
  if n < 1 then 0 else n + sum(n - 1)
in sum(100);
-- Error: missing dictionary method: prelude::Ord::Bool::lt
```

**After:**
```ailang
let sum_result = letrec sum = \n.
  if n < 1 then 0 else n + sum(n - 1)
in sum(100);
-- Output: 5050
```

## Success Criteria

- [ ] Factorial `letrec factorial = \n. if n < 2 then 1 else n * factorial(n-1) in factorial(5)` returns 120
- [ ] Sum `letrec sum = \n. if n < 1 then 0 else n + sum(n-1) in sum(100)` returns 5050
- [ ] Power function works
- [ ] GCD function works
- [ ] isEven/isOdd mutual recursion works
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples restored

## Testing Strategy

**Unit tests:**
- Test single-recursive-call letrec with various operators (+, *, -)
- Test base case with literal (1) vs variable (n)
- Test deeply nested single-call recursion

**Integration tests:**
- Full letrec_recursion.ail example file
- REPL evaluation of letrec patterns

**Regression tests:**
- Ensure Fibonacci (double-call) still works
- Ensure sequential letrecs still work (prior fix in v0.6.1)

## Non-Goals

**Not in this feature:**
- Mutual recursion (`letrec f = ... g ... and g = ... f ...`) - separate syntax
- Letrec performance optimization - focus on correctness

## Timeline

**Day 1** (~4 hours):
- Phase 1: Diagnosis (2 hours)
- Phase 2: Fix (1.5 hours)
- Phase 3: Testing (0.5 hours)

**Total: ~4 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Bug deeper than suspected (elaboration vs eval) | Medium | Systematic tracing from source to error |
| Fix breaks double-call patterns | Medium | Test both patterns before/after |
| Multiple interacting bugs | Low | Prior v0.6.1 fix addressed sequential letrecs |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_6_1/m-letrec-scoping-sprint-plan.md](design_docs/implemented/v0_6_1/m-letrec-scoping-sprint-plan.md) - Prior letrec fix (sequential letrecs)

**Planned (check for overlap):**
- None directly related

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Prior letrec fix: v0.6.1 M-LETREC-SCOPING addressed sequential letrec scoping
- Evaluator: `internal/eval/eval_expressions.go:evalCoreLetRec`

## Diagnostic Notes

Key observation: The error `missing dictionary method: prelude::Ord::Bool::lt` is bizarre because:
1. The code uses `n < 2` which should be `Int::lt`, not `Bool::lt`
2. Type checking passes - so types are correct at compile time
3. Only manifests at runtime with single recursive call

This suggests the dictionary passing or specialization is confusing the recursive call count with something type-related.

---

**Document created**: 2026-01-29
**Last updated**: 2026-01-29
