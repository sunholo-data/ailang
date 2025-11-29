# Sprint Plan: M-BUG-RECURSION-DEPTH

## Summary
Fix infinite recursion bug when using `match` expressions in recursive functions. `if-then-else` works correctly for 5000+ recursive calls, but `match` triggers stack overflow even for `count(1)`.

**Duration:** 2 days
**Dependencies:** None
**Risk Level:** Medium (root cause narrowed but requires evaluator debugging)

## Verified Bug Characteristics

### Reproduction (verified 2025-11-29)
```ailang
-- FAILS even for count(1) - infinite recursion!
pure func count(n: int) -> int {
  match n {
    0 => 0,
    _ => 1 + count(n-1)
  }
}

-- WORKS up to count(5000)
pure func count(n: int) -> int {
  if n <= 0 then 0 else 1 + count(n-1)
}
```

### Key Finding
This is **infinite recursion**, not expensive recursion:
- `count(1)` with `match` exceeds 100,000 depth limit
- Same logic with `if-then-else` works at depth 5000+
- The `match` version never terminates regardless of input

### Root Cause Hypothesis
The pattern matching on integers in a recursive context likely has one of:
1. **Wildcard `_` consuming all values** including 0 (pattern order issue)
2. **Recursive function lookup failing** inside match arms
3. **Integer literal patterns not matching correctly**

## Current Status Analysis

### Velocity (last 14 days)
- Recent average: ~200 LOC/day
- Milestones completed: v0.4.7 inline tests, v0.4.8 import aliasing
- Estimated capacity: 400 LOC for 2-day sprint

### Related Code
- `internal/eval/eval_patterns.go` - evalCoreMatch, matchPattern (262 lines)
- `internal/eval/eval_operations.go` - recursion depth tracking (line 39)
- `internal/elaborate/patterns.go` - normalizeMatch (198 lines)
- `internal/eval/recursion_test.go` - existing recursion tests

## Proposed Milestones

### Milestone 1: Debug and Root Cause (Day 1)
**Goal:** Identify exactly why `match` causes infinite recursion
**Estimated:** 50 LOC (debug tracing) + 30 LOC tests = 80 LOC
**Duration:** 4-6 hours

**Tasks:**
1. Add debug tracing to `evalCoreMatch` and `matchPattern`
   - Log pattern type being matched
   - Log scrutinee value
   - Log which arm matches (if any)
2. Run minimal reproduction (`count(1)`)
3. Analyze trace to find infinite loop point
4. Document root cause

**Acceptance Criteria:**
- [ ] Root cause identified and documented
- [ ] Debug trace shows exactly where loop occurs
- [ ] Hypothesis confirmed or revised

**Files to Modify:**
- `internal/eval/eval_patterns.go` (add debug logging)

### Milestone 2: Implement Fix (Day 2)
**Goal:** Fix the infinite recursion and add regression tests
**Estimated:** 80 LOC (fix) + 150 LOC (tests) = 230 LOC
**Duration:** 4-6 hours

**Tasks:**
1. Implement fix based on Day 1 findings
   - If pattern order: fix pattern matching logic
   - If function lookup: fix environment handling in match arms
   - If literal matching: fix integer pattern comparison
2. Add regression tests
   - `match` with 0 base case
   - `match` with wildcard fallthrough
   - Various recursion depths (10, 100, 1000)
3. Verify existing recursion tests still pass
4. Clean up debug code

**Acceptance Criteria:**
- [ ] `count(10)` with `match` works
- [ ] `count(1000)` with `match` works
- [ ] `count(10001)` correctly triggers depth limit
- [ ] All existing tests pass
- [ ] Linting clean

**Files to Modify:**
- `internal/eval/eval_patterns.go` or `internal/elaborate/patterns.go`
- `internal/eval/recursion_test.go` (new tests)
- `tests/recursion_depth_test.ail` (example file)

### Milestone 3: Documentation and Cleanup
**Goal:** Document fix and add examples
**Estimated:** 60 LOC (docs + example)
**Duration:** 1-2 hours

**Tasks:**
1. Move test file to `examples/runnable/recursion_match.ail`
2. Update CHANGELOG.md
3. Remove debug tracing if any remains

**Acceptance Criteria:**
- [ ] Working example in examples/runnable/
- [ ] CHANGELOG updated
- [ ] No debug code in production

## Success Metrics
- Test coverage: Maintained (add 5+ new recursion tests)
- Examples passing: +1 (recursion_match.ail)
- All tests passing: ✅
- All linting passing: ✅

## Total Estimates
- **Implementation:** ~160 LOC
- **Tests:** ~180 LOC
- **Documentation:** ~30 LOC
- **Total:** ~370 LOC over 2 days

## Risks
| Risk | Impact | Mitigation |
|------|--------|------------|
| Root cause harder than expected | High | Day 1 dedicated to investigation |
| Fix breaks other pattern matching | Medium | Run full test suite continuously |
| Multiple overlapping issues | Low | Address one issue at a time |

## Notes
- The bug was reported by stapledons_voyage via agent inbox
- Workaround: Use `if-then-else` instead of `match` for recursive base cases
- Consider responding to stapledons_voyage after fix is verified

---
**Created:** 2025-11-29
**Status:** Ready for execution
