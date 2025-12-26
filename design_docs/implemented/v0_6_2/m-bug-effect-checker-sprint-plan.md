# Sprint Plan: M-BUG-EFFECT-CHECKER-CONFLATION

**Sprint ID:** M-BUG-EFFECT-CHECKER
**Target Version:** v0.6.2
**Duration:** 1 day (4 hours)
**Risk Level:** Low
**Created:** 2025-12-24

## Sprint Summary

**Goal:** Fix effect checker bug that incorrectly requires IO effects for pure functions when called inside println statements that follow other println statements.

**Key Deliverable:** `examples/runnable/pattern_sugar.ail` passes verification without modification.

**Confidence:** High - Clear reproduction, well-defined scope, existing test infrastructure.

## Current Status Analysis

### Completed Work
- Bug confirmed: `pattern_sugar.ail` fails with spurious "Missing effects: IO" on pure `sum` function
- Design doc created with clear reproduction steps
- Root cause hypothesis: Effect propagation in nested block expressions

### Recent Velocity
From last 7 days of commits:
- M-CAPABILITY-BUDGETS: ~125 LOC (parser + evaluator + tests + examples)
- M-DX16 (record match arms): ~320 LOC (parser + tests + example)
- M-DX18 (namespace fix): ~50 LOC
- **Average:** ~150-200 LOC per feature, ~1-2 days per milestone

### Remaining Work
- Phase 1: Diagnosis (find exact bug location)
- Phase 2: Fix effect propagation logic
- Phase 3: Add regression test and verify

## Proposed Milestones

### M1: Diagnosis and Root Cause Identification (~1.5 hours)

**Tasks:**
1. Add debug logging to effect checker (`internal/types/effects.go`)
2. Create minimal reproduction test case
3. Compare effect flow in passing vs failing cases
4. Identify exact code location where IO incorrectly propagates to pure functions

**Estimated LOC:**
- Debug logging: ~30 LOC
- Minimal test case: ~40 LOC
- Total: ~70 LOC

**Acceptance Criteria:**
- [ ] Can reproduce bug in isolated test case
- [ ] Debug trace shows exact point where IO effect incorrectly attaches to `sum`
- [ ] Root cause identified in specific function/line

**Files to Modify:**
- `internal/types/effects.go` - Add debug logging
- `internal/types/effects_test.go` - Add minimal reproduction test

**Example to Create:**
- Minimal test case demonstrating the bug pattern

**Dependencies:** None

**Risk Factors:**
- Low: Effect checker code is well-isolated
- Existing test suite provides safety net

### M2: Implement Fix (~1.5 hours)

**Tasks:**
1. Fix effect propagation logic (likely scoping/isolation issue)
2. Add regression test for the specific bug pattern
3. Verify `pattern_sugar.ail` passes without modification
4. Verify all existing effect-checking tests still pass

**Estimated LOC:**
- Fix: ~20 LOC
- Regression test: ~40 LOC
- Total: ~60 LOC

**Acceptance Criteria:**
- [ ] Fix applied to effect checker
- [ ] `pattern_sugar.ail` passes: `ailang check examples/runnable/pattern_sugar.ail`
- [ ] Regression test added and passing
- [ ] All existing tests pass: `make test`

**Files to Modify:**
- `internal/types/effects.go` - Fix effect propagation
- `internal/types/effects_test.go` - Add regression test
- Possibly `internal/types/infer.go` or `internal/types/unify.go` (TBD from diagnosis)

**Example to Update:**
- None (pattern_sugar.ail should work as-is)

**Dependencies:** M1 (need root cause before fixing)

**Risk Factors:**
- Low: Isolated change to effect checker
- Medium: Could affect legitimate effect inference if fix is too broad

### M3: Verification and Cleanup (~1 hour)

**Tasks:**
1. Run full test suite: `make test`
2. Run example verification: `make verify-examples`
3. Remove debug logging added in M1 (or gate behind DEBUG_EFFECTS flag)
4. Update CHANGELOG.md with bug fix entry
5. Commit changes with proper message

**Estimated LOC:**
- CHANGELOG update: ~15 LOC
- Debug flag gating (if keeping logging): ~10 LOC
- Total: ~25 LOC

**Acceptance Criteria:**
- [ ] All tests pass: `make test`
- [ ] All examples pass: `make verify-examples`
- [ ] No regressions in effect checking
- [ ] CHANGELOG.md updated
- [ ] Changes committed with descriptive message

**Files to Modify:**
- `CHANGELOG.md` - Document bug fix
- `internal/types/effects.go` - Clean up or gate debug logging

**Example to Verify:**
- `examples/runnable/pattern_sugar.ail` runs successfully
- All other effect-using examples still work

**Dependencies:** M2 (need fix before verification)

**Risk Factors:**
- Very low: Verification only

## Implementation Timeline

**Total Duration:** 4 hours (1 day)

**Day 1 (4 hours):**
- Morning (2 hours): M1 - Diagnosis
  - 0:00-0:30: Add debug logging to effect checker
  - 0:30-1:00: Create minimal test case, run traces
  - 1:00-1:30: Compare passing vs failing effect flow
  - 1:30-2:00: Identify root cause, document findings

- Afternoon (2 hours): M2 + M3 - Fix and Verify
  - 2:00-2:30: Implement fix based on diagnosis
  - 2:30-3:00: Add regression test, verify pattern_sugar.ail
  - 3:00-3:30: Run full test suite, verify examples
  - 3:30-4:00: Clean up, update CHANGELOG, commit

## Success Metrics

### Code Quality
- [ ] All existing tests pass
- [ ] Regression test added for this specific bug
- [ ] No new effect checking warnings

### Functionality
- [ ] `examples/runnable/pattern_sugar.ail` passes without modification
- [ ] Pure functions remain pure (no false IO requirements)
- [ ] Legitimate IO effects still properly enforced

### Documentation
- [ ] CHANGELOG.md updated with bug fix entry
- [ ] Regression test documents the bug pattern
- [ ] Optional: DEBUG_EFFECTS flag for future debugging

## Total LOC Estimate

| Category | LOC |
|----------|-----|
| Implementation | 20 |
| Tests | 80 |
| Debug/Tooling | 30 |
| Documentation | 15 |
| **Total** | **~145 LOC** |

## Dependencies and Blockers

**None** - This is a self-contained bug fix with no external dependencies.

## Open Questions

**Q:** Should we keep debug logging behind DEBUG_EFFECTS flag for future debugging?
**A:** Yes, if minimal overhead. Otherwise remove after fix is verified.

**Q:** Could this bug affect other effect types (FS, Net, Clock)?
**A:** Likely yes - the fix should be general, not IO-specific.

**Q:** Do we need to update documentation about effect checking?
**A:** No - this is a bug fix, not a feature change. Users expect pure functions to remain pure.

## Risk Assessment

**Overall Risk: Low**

**Technical Risks:**
- Low: Effect checker code is well-isolated with existing tests
- Medium: Fix could be too broad and affect legitimate effect inference

**Mitigation:**
- Thorough testing with existing test suite
- Regression test for specific bug pattern
- Careful review of effect propagation logic

**Schedule Risks:**
- Very Low: 4-hour estimate with clear scope
- Diagnosis may take longer if root cause is subtle

**Mitigation:**
- Break at end of M1 if diagnosis incomplete
- Detailed debug logging to trace effect flow

## Related Work

- **M-BUG-RECURSION-DEPTH** - Similar compiler bug fix pattern
- **M-PARSER-NESTED-MATCH** - Related nested context bug
- **Effect System** - Core type system feature (stable)

## Notes

- This bug was discovered during example verification
- Binary search of file contents isolated the trigger pattern
- Bug only manifests with specific nesting: println before println(show(pureFunc))
- Neither condition alone triggers the bug
