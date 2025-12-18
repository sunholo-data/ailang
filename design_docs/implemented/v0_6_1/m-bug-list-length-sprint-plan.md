# Sprint Plan: M-BUG-LIST-LENGTH - Resolution Documentation

**Sprint ID**: M-BUG-LIST-LENGTH-DOC
**Status**: Documentation Only (Bug Already Fixed)
**Duration**: 30 minutes
**Risk Level**: Low

## Summary

The bug described in `m-bug-list-length-returns-wrong-value.md` has already been fixed by commit `752cf226` (M-LETREC-SCOPING) on Dec 18, 2025. This sprint handles documentation cleanup only.

## Discovery

**Original bug report:**
- `length(numbers)` returned 15 instead of 5 for a 5-element list
- Suspected wildcard pattern `_` was not working correctly
- Affected recursive ADT operations

**Actual root cause (found in M-LETREC-SCOPING):**
- Monomorphization cache key collision for lambdas
- Used generic `"(lambda)"` key for ALL anonymous lambdas
- Two lambdas with same type signature would share cache key
- Second lambda would get first lambda's specialized body

**The fix:**
```go
// Before (broken):
DefSym: "(lambda)"  // Same key for all lambdas with same type!

// After (fixed):
DefSym: fmt.Sprintf("(lambda@%d)", lambda.ID())  // Unique per lambda
```

**Impact:**
- More comprehensive than just letrec - fixed ALL lambda specialization
- Non-recursive cases would produce wrong results silently
- Fix in `internal/pipeline/specialize_lambda.go`

## Verification Results

**Test 1: Original reproduction case**
```bash
$ ./bin/ailang run examples/runnable/list_sum.ail
(15, 5)  # ✅ PASS: sum=15, length=5
```

**Test 2: Isolated length function**
```bash
$ ./bin/ailang run /tmp/test_length.ail
5  # ✅ PASS: Returns 5, not 15
```

**Test 3: Wildcard patterns**
```ailang
ICons(_, tail) => 1 + length(tail)  # ✅ Wildcard works correctly
```

## Milestones

### M1: Documentation Cleanup (15 min, ~50 LOC)

**Tasks:**
- [x] Move `m-bug-list-length-returns-wrong-value.md` to `implemented/v0_6_1/`
- [x] Add "Resolution" section documenting the fix
- [x] Link to M-LETREC-SCOPING commit and design doc
- [x] Update status to "Resolved by M-LETREC-SCOPING"

**Acceptance Criteria:**
- Design doc moved to implemented with resolution notes
- Clear explanation of root cause and fix
- Links to related commits and design docs

### M2: Verify Test Coverage (15 min)

**Tasks:**
- [x] Verify `internal/eval/recursion_test.go` has regression test
- [x] Verify `internal/pipeline/specialize_integration_test.go` covers lambda collision
- [x] Confirm `examples/runnable/list_sum.ail` works correctly

**Acceptance Criteria:**
- All regression tests passing
- Test coverage for lambda cache key collision
- Example file verified working

## Files Modified

- `design_docs/planned/v0_6_1/m-bug-list-length-returns-wrong-value.md` → `design_docs/implemented/v0_6_1/`
- No code changes required (already fixed)

## Success Metrics

- [x] Bug verified as fixed
- [x] Design doc moved to implemented/
- [x] Resolution documented
- [x] Test coverage verified
- [x] No new code required

## Related Work

- **Fix commit**: `752cf226` - M-LETREC-SCOPING
- **Design doc**: `design_docs/implemented/v0_6_1/m-letrec-scoping-regression.md`
- **Tests added**:
  - `internal/eval/recursion_test.go` - Sequential letrec test
  - `internal/pipeline/specialize_integration_test.go` - Lambda collision test

## Timeline

**Total Duration**: 30 minutes

- M1: Documentation cleanup (15 min)
- M2: Test coverage verification (15 min)

**Estimated LOC**: ~50 (documentation updates only)

---

**Created**: 2025-12-18
**Status**: Ready for execution (documentation only)
