# Sprint Plan: M-LETREC-SCOPING Fix

## Summary
Fix the runtime letrec scoping regression where sequential letrec expressions with recursive lambdas fail with "undefined variable" errors.

**Duration:** 1 day (2-4 hours)
**Dependencies:** None
**Risk Level:** Low
**Design Doc:** [m-letrec-scoping-regression.md](m-letrec-scoping-regression.md)

## Diagnosis Summary (Updated)

**Original hypothesis:** Type checker doesn't add binding to environment before checking body.

**Actual finding:** Type checking passes. The bug is in the **evaluator**, not the type checker.

### Reproduction
```ailang
-- Works (single letrec)
let r1 = letrec a = \n. if n == 0 then 1 else n * a(n - 1) in a(3);
println("done");  -- OK

-- Fails (two sequential letrecs with recursion)
let r1 = letrec a = \n. if n == 0 then 1 else n * a(n - 1) in a(3);
let r2 = letrec b = \n. if n == 0 then 1 else n * b(n - 1) in b(3);
-- Error: undefined variable: a
```

### Root Cause Analysis
- `evalCoreLetRec` in `internal/eval/eval_expressions.go` correctly sets up recursive environment
- Bug is in environment restoration or closure capture when multiple letrecs execute sequentially
- The first letrec's binding `a` is incorrectly referenced during the second letrec's evaluation

## Proposed Milestones

### Milestone 1: Diagnose Evaluator Bug
**Goal:** Identify exact line where environment corruption occurs
**Estimated:** ~50 LOC debugging + instrumentation
**Duration:** 1 hour

**Tasks:**
1. Add debug logging to `evalCoreLetRec` showing environment before/after
2. Add debug logging to `buildClosure` showing captured environment
3. Trace execution of failing test case
4. Identify where `a` leaks into second letrec's scope

**Acceptance Criteria:**
- [ ] Can trace exact environment state at failure point
- [ ] Root cause identified with specific line number

### Milestone 2: Fix Evaluator
**Goal:** Correct environment handling in sequential letrecs
**Estimated:** ~30 LOC fix
**Duration:** 1 hour

**Suspected fixes (investigate in order):**
1. Environment not properly restored after first letrec (check `defer` behavior)
2. Closure capturing shared environment reference instead of copy
3. Child environment incorrectly inheriting parent bindings

**Tasks:**
1. Implement fix based on diagnosis
2. Verify single letrec still works
3. Verify sequential letrecs work
4. Verify nested letrecs work

**Acceptance Criteria:**
- [ ] Sequential letrecs with recursion work
- [ ] Existing tests pass

### Milestone 3: Testing & Verification
**Goal:** Comprehensive test coverage and example verification
**Estimated:** ~80 LOC tests
**Duration:** 1 hour

**Tasks:**
1. Add unit test for sequential letrec scoping
2. Add unit test for nested letrec scoping
3. Verify `examples/runnable/letrec_recursion.ail` passes
4. Run `make test` and `make verify-examples`

**Acceptance Criteria:**
- [ ] New unit tests passing
- [ ] `examples/runnable/letrec_recursion.ail` outputs expected results
- [ ] `make test` passes
- [ ] `make verify-examples` shows improved pass rate

## Files to Modify

**Primary (bug fix):**
- `internal/eval/eval_expressions.go` - Fix `evalCoreLetRec` environment handling (~30 LOC)

**Testing:**
- `internal/eval/recursion_test.go` - Add sequential letrec tests (~80 LOC)

**Debugging (temporary):**
- May add debug flags, remove before commit

## Success Metrics
- `letrec factorial = \n. ... factorial(n-1) ...` works in all contexts
- `examples/runnable/letrec_recursion.ail` passes with expected output
- All 6 test cases in example file pass (Fibonacci, Factorial, Sum, GCD, Power, Even)
- `make test` passes
- `make verify-examples` shows 55/56 or better

## Key Files to Review

```bash
# Evaluator (primary suspect)
internal/eval/eval_expressions.go:191  # evalCoreLetRec
internal/eval/eval_patterns.go:581     # buildClosure
internal/eval/env.go:17                # NewChildEnvironment

# For comparison (working recursion)
internal/eval/eval_evaluator.go:148    # Alternative letrec impl
```

## Notes

- Original design doc suspected type checker - my testing confirms type checking passes
- Bug manifests only at runtime with sequential recursive letrecs
- The `defer func() { e.env = oldEnv }()` pattern looks correct but may have subtle issue
- Closure environment capture may be the culprit (capturing by reference vs copy)

---

**Created:** 2025-12-18
**Sprint ID:** M-LETREC-SCOPING
