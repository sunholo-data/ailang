# Sprint Plan: M-DX3 Lambda DX Fixes

## Summary
Fix two critical bugs blocking lambda examples: (1) comparison operators failing in lambda bodies due to type inference, and (2) `show` builtin not working for Bool values. These P0 blockers prevented ~20% of lambda patterns and added 50% overhead to v0.3.16 sprint.

**Duration:** 1 day (4-6 hours)
**Dependencies:** None (uses existing M-DX2 CoreTypeInfo infrastructure)
**Risk Level:** Low (surgical fixes to well-understood systems)

## Current Status Analysis

### Completed Recently (Last 14 Days)
- ✅ Lambda Examples Refactor (v0.3.16): 297 LOC in ~1.5 days
- ✅ M-DX2 Operator Lowering: ~150 LOC infrastructure in ~3 days
- ✅ JSON Decoding: ~860 LOC + 42 tests in ~2 days
- ✅ show() Builtin: ~350 LOC + 35 tests in ~1 day

### Velocity
- **Recent average:** ~200-300 LOC/day (implementation + tests)
- **Estimated capacity:** 400-500 LOC for 1-day sprint
- **Confidence:** High (fixes are well-scoped, no unknowns)

### Blocking Issues from v0.3.16
- 🔴 **P0:** Comparison operators in lambdas fail type checking
  - Example blocked: `let max = \x. \y. if x > y then x else y`
  - Error: "Operator '>' has no implementation for type Bool"
  - Root cause: Bool back-propagates to operands

- 🔴 **P0:** show(Bool) doesn't work
  - Example blocked: `show(true)`
  - Error: "cannot unify type constructors: string vs bool"
  - Root cause: Missing Bool case in type-directed lowering

## Proposed Milestones

### Milestone 1: Fix Comparison Operator Type Inference
**Goal:** Ensure comparison operators (`>`, `<`, `>=`, `<=`, `==`, `!=`) correctly constrain operands to Num instead of Bool when used in lambda bodies.

**Estimated:** 50 LOC implementation + 200 LOC tests = 250 LOC
**Duration:** 2-3 hours

**Tasks:**
- **Task 1.1 (30min):** Analyze current type inference flow
  - Trace `\x. x > 0` through `internal/types/infer.go`
  - Document constraint propagation (where Bool flows backward)
  - Identify fix location (likely in operator constraint generation)

- **Task 1.2 (60min):** Implement constraint ordering fix
  - Ensure comparison operators emit Num constraint for operands BEFORE Bool result
  - Add type constraint: `>` :: Num a => a -> a -> Bool
  - Test locally: `let max = \x. \y. if x > y then x else y`
  - Verify: `max(10)(20)` → `20`

- **Task 1.3 (60min):** Add comprehensive tests
  - File: `internal/types/typechecker_operators_test.go` (new)
  - Test all 6 comparison operators: `>`, `<`, `>=`, `<=`, `==`, `!=`
  - Test in lambda bodies: `\x. x > 0`, `\x. \y. x < y`
  - Test nested: `\f. \x. f(x) > 0`
  - Test guards: `\x. if x > 0 then "pos" else "neg"`
  - Test combined: `\x. x > 0 && x < 100`
  - Expected: 15-20 test cases

- **Task 1.4 (30min):** Regression tests
  - Ensure non-lambda comparisons unchanged: `if 10 > 5 then ...`
  - Test let-bound: `let b = x > y in if b then ...`
  - Run full test suite: `make test`

**Acceptance Criteria:**
- [ ] `let max = \x. \y. if x > y then x else y` type checks
- [ ] `max(10)(20)` evaluates to `20`
- [ ] All 6 comparison operators work in lambda bodies
- [ ] No regressions in existing comparison tests
- [ ] 15+ new test cases passing
- [ ] All tests passing: `make test`
- [ ] Linting clean: `make lint`

**Risks:**
- **Constraint ordering too complex:** Mitigation: Start with simplest fix (single operator), generalize if works
- **Breaks existing if-expressions:** Mitigation: Comprehensive regression tests before changing behavior

---

### Milestone 2: Add Bool Support to show Builtin
**Goal:** Enable `show(true)` and `show(false)` to return `"true"` and `"false"` by adding Bool case to type-directed show lowering.

**Estimated:** 10 LOC implementation + 50 LOC tests = 60 LOC
**Duration:** 0.5-1 hour

**Tasks:**
- **Task 2.1 (20min):** Add Bool case to show lowering
  - File: `internal/pipeline/op_lowering.go`
  - Find type-head switch in show lowering
  - Add case: `types.IsBool(argType)` → route to `show_Bool`
  - Use existing `show_Bool` builtin (already implemented)

- **Task 2.2 (20min):** Add tests
  - File: `internal/builtins/show_test.go` (existing)
  - Test: `show(true)` → `"true"`
  - Test: `show(false)` → `"false"`
  - Test in expression: `"Result: " ++ show(x > 0)`
  - Test with predicates: `show(is_positive(5))`
  - Expected: 5-8 test cases

- **Task 2.3 (10min):** Verify show_Bool compatibility
  - Ensure explicit `show_Bool(true)` still works
  - Test examples/snippets/typeclasses.ail
  - No breaking changes to existing code

**Acceptance Criteria:**
- [ ] `show(true)` returns `"true"`
- [ ] `show(false)` returns `"false"`
- [ ] `show(5 > 3)` returns `"true"`
- [ ] Explicit `show_Bool(true)` unchanged
- [ ] 5+ new test cases passing
- [ ] All tests passing: `make test`

**Risks:**
- **Conflicts with show_Bool:** Mitigation: Keep both paths working; test both
- **Type-head detection wrong:** Mitigation: Use existing M-DX2 CoreTypeInfo infrastructure (proven)

---

### Milestone 3: Integration, Examples, Documentation
**Goal:** Update lambda examples with previously-blocked patterns, document fixes in CHANGELOG, update LIMITATIONS.md.

**Estimated:** 90 LOC (examples + docs)
**Duration:** 1 hour

**Tasks:**
- **Task 3.1 (20min):** Enhance lambda examples
  - File: `examples/snippets/showcase/lambdas_basic.ail`
    - Add: `max`, `min`, `abs` examples
    - Add: predicate examples with `show(bool)` output
  - File: `examples/snippets/showcase/lambdas_advanced.ail`
    - Add: `in_range`, `classify` guard examples
    - Add: combined predicates with boolean display

- **Task 3.2 (20min):** Update LIMITATIONS.md
  - Add Y-combinator limitation (discovered in v0.3.16):
    ```markdown
    ## Y-Combinator Not Supported

    AILANG's occurs check prevents Y-combinator. Use `let rec` instead:

    ❌ Don't use Y-combinator:
    let fix = \f. let self = \x. f(\v. (x(x))(v)) in self(self)

    ✅ Use let rec instead:
    let rec factorial = \n. if n == 0 then 1 else n * factorial(n - 1)
    ```
  - Remove workarounds for comparison operators and show(bool)

- **Task 3.3 (10min):** Update CHANGELOG.md
  - Add M-DX3 section under v0.3.17
  - Include before/after examples
  - Note enabled lambda patterns
  - Include LOC metrics

- **Task 3.4 (10min):** Final verification
  - Run: `make test` (all tests pass)
  - Run: `make lint` (clean)
  - Test all 6 lambda examples from v0.3.16
  - Test new examples with max/min/abs/predicates

**Acceptance Criteria:**
- [ ] Lambda examples enhanced with 4+ new patterns
- [ ] LIMITATIONS.md updated (Y-combinator documented)
- [ ] CHANGELOG.md includes M-DX3 entry
- [ ] All 6 lambda examples pass
- [ ] New lambda patterns work correctly
- [ ] Documentation is accurate and complete

**Risks:**
- **Examples too complex:** Mitigation: Keep examples simple and focused (one concept each)
- **Documentation unclear:** Mitigation: Include concrete before/after code examples

---

## Day-by-Day Breakdown

### Hour 0-3: Morning (Comparison Operators)
- **00:00-00:30:** Analyze type inference (Task 1.1)
- **00:30-01:30:** Implement constraint fix (Task 1.2)
- **01:30-02:30:** Write tests (Task 1.3)
- **02:30-03:00:** Regression tests (Task 1.4)
- **Checkpoint:** `make test && make lint` → all green

### Hour 3-4: Midday (show Bool)
- **03:00-03:20:** Add Bool case to lowering (Task 2.1)
- **03:20-03:40:** Write tests (Task 2.2)
- **03:40-03:50:** Verify show_Bool compatibility (Task 2.3)
- **Checkpoint:** `make test` → all green

### Hour 4-5: Afternoon (Integration & Docs)
- **04:00-04:20:** Enhance lambda examples (Task 3.1)
- **04:20-04:40:** Update LIMITATIONS.md (Task 3.2)
- **04:40-04:50:** Update CHANGELOG.md (Task 3.3)
- **04:50-05:00:** Final verification (Task 3.4)
- **Checkpoint:** All tests, examples, docs complete

### Hour 5-6: Buffer & Review
- **05:00-05:30:** Manual testing in REPL
- **05:30-06:00:** Code review, cleanup, commit
- **Final:** Ready to merge

---

## Success Metrics

### Quantitative
- **Test coverage:** > 90% for new code (M1: 50 LOC impl + 200 LOC tests; M2: 10 LOC impl + 50 LOC tests)
- **Examples passing:** All 6 from v0.3.16 + 4 new patterns = 10 passing examples
- **Total LOC:** ~400 LOC (60 impl + 340 tests/docs)
- **Zero regressions:** All existing tests still pass

### Qualitative
- **Unblocked patterns:** max, min, abs, is_positive, in_range, classify, guards
- **Developer experience:** Can now write natural lambda code without workarounds
- **Documentation:** Clear LIMITATIONS.md entry prevents Y-combinator confusion

### Files Updated
- **Implementation:** 2 files (~60 LOC)
  - `internal/types/infer.go` or `internal/types/typechecker_operators.go`
  - `internal/pipeline/op_lowering.go`
- **Tests:** 2 files (~250 LOC)
  - `internal/types/typechecker_operators_test.go` (new)
  - `internal/builtins/show_test.go` (extended)
- **Examples:** 2 files (~40 LOC)
  - `examples/snippets/showcase/lambdas_basic.ail`
  - `examples/snippets/showcase/lambdas_advanced.ail`
- **Documentation:** 2 files (~50 LOC)
  - `docs/LIMITATIONS.md`
  - `CHANGELOG.md`

---

## Dependencies

**External:** None

**Internal:**
- ✅ M-DX2 (Operator Lowering) - Provides CoreTypeInfo for type-directed decisions
- ✅ show_Bool builtin - Already implemented, just needs routing
- ✅ Comparison operators - Already exist, just need constraint fix

**Blocked by this:** None (but unblocks lambda tutorial expansion)

---

## Open Questions

None - sprint is fully scoped based on v0.3.16 findings.

---

## Assumptions

1. **Type inference fix location:** Assumes fix is in `internal/types/infer.go` or `internal/types/typechecker_operators.go`. If constraint generation is elsewhere, may need to search.

2. **show_Bool exists:** Assumes `show_Bool` builtin is already implemented (confirmed in v0.3.16 analysis). Just needs routing in type-directed lowering.

3. **No breaking changes:** Assumes constraint ordering fix won't break existing non-lambda comparisons. Mitigated by comprehensive regression tests.

4. **One-day timeline:** Based on recent velocity (~200-300 LOC/day) and well-scoped work. Buffer included for unknowns.

---

## Notes

- **Risk mitigation:** Both fixes are independent - can ship incrementally if one takes longer
- **Testing philosophy:** 85% of LOC is tests (340/400) - ensures correctness and prevents regressions
- **User impact:** Removes major friction from v0.3.16 sprint, enables 20% more lambda patterns
- **Follow-up:** M-DX4 (Effectful Expression Ergonomics) builds on these fixes for better DX

---

**Created:** 2025-10-21
**Last Updated:** 2025-10-21
**Status:** Ready for execution
