# Sprint Plan: M-BUG-RECORD-UPDATE-INFERENCE

## Summary
Fix record update type inference to work with lambda parameters by refactoring to use constraint-based type checking, following the existing `inferRecordAccess` pattern.

**Duration:** 2 days (~8 hours total)
**Dependencies:** None
**Risk Level:** Low-Medium (well-understood fix pattern, verified root cause)

## Current Status Analysis

### Completed Recently
- v0.4.8: Import aliasing, prompt updates (~200 LOC)
- v0.4.7: Inline testing infrastructure (~400 LOC)
- Cross-function dependency support (~150 LOC)

### Velocity
- Recent average: ~150-200 LOC/day (based on CHANGELOG analysis)
- Estimated capacity: ~100 LOC for this sprint (conservative, bug fix)

### Bug Analysis (Verified 2025-11-29)
- **Location:** `internal/types/typechecker_data.go:99-162`
- **Root Cause:** Eager type checking instead of constraint-based
- **Pattern to Follow:** `inferRecordAccess` at lines 59-95

## Proposed Milestones

### Milestone 1: Implement Constraint-Based Fix
**Goal:** Refactor `inferRecordUpdate` to defer type checking to constraint solver
**Estimated:** 40 LOC implementation + 20 LOC tests = 60 LOC
**Duration:** Day 1 (5 hours)

**Tasks:**
- Hour 1: Add regression test for let-bound records (currently works)
- Hour 2: Implement constraint-based approach following `inferRecordAccess`
- Hour 3: Handle row variable generation correctly
- Hour 4: Test lambda parameter case (main bug)
- Hour 5: Run full test suite, fix any regressions

**Acceptance Criteria:**
- [ ] Let-bound record updates still work (regression test passes)
- [ ] Lambda parameter with type annotation works
- [ ] Lambda parameter without type annotation works
- [ ] Multiple field updates work
- [ ] All existing tests passing
- [ ] Linting clean

**Risks:**
- Row polymorphism edge cases - Mitigation: Test with open records, defer complex cases to follow-up

### Milestone 2: Testing & Documentation
**Goal:** Comprehensive testing, example file, and external communication
**Estimated:** 40 LOC tests + example file = 60 LOC
**Duration:** Day 2 (3 hours)

**Tasks:**
- Hour 1: Create `examples/record_update.ail` with working examples
- Hour 2: Create edge case tests (error cases, nested records)
- Hour 3: Update CHANGELOG.md, respond to stapledons_voyage

**Acceptance Criteria:**
- [ ] Example file runs without errors: `ailang run examples/record_update.ail`
- [ ] Error case: updating non-record produces clear error
- [ ] CHANGELOG.md updated with fix description
- [ ] Response sent to stapledons_voyage via agent inbox

**Risks:**
- None significant - standard documentation work

## Day-by-Day Breakdown

### Day 1 (5 hours)
| Hour | Task | Deliverable |
|------|------|-------------|
| 1 | Write regression test for let-bound records | `tests/record_update_regression_test.go` |
| 2 | Refactor `inferRecordUpdate` to constraint-based | Modified `typechecker_data.go` |
| 3 | Handle row variable generation | Row polymorphism support |
| 4 | Test lambda parameter cases | Verified fix |
| 5 | Full test suite + lint | Green CI |

### Day 2 (3 hours)
| Hour | Task | Deliverable |
|------|------|-------------|
| 1 | Create example file | `examples/record_update.ail` |
| 2 | Edge case tests | Additional test coverage |
| 3 | Docs + agent response | CHANGELOG + stapledons_voyage response |

## Success Metrics
- Test coverage: Maintain current level
- Examples passing: `examples/record_update.ail` works
- Documentation: CHANGELOG.md updated
- All tests passing
- All linting passing
- stapledons_voyage bug acknowledged

## Files to Modify

**Core Fix:**
- `internal/types/typechecker_data.go:99-162` (~40 LOC)

**Tests:**
- `tests/record_update_inference_test.ail` (~40 LOC, new)
- `internal/types/typechecker_data_test.go` (if exists, add cases)

**Examples:**
- `examples/record_update.ail` (~20 LOC, new)

**Documentation:**
- `CHANGELOG.md` (add entry)

## Dependencies
- None - this is a self-contained bug fix

## Open Questions
- Should row polymorphic record updates be fully supported in this fix, or deferred?
  - **Recommendation:** Test basic case, defer complex row polymorphism to follow-up

## Notes
- The fix follows a well-established pattern in the codebase (`inferRecordAccess`)
- Root cause verified through code reading and reproduction
- Low risk due to isolated change in single function
- stapledons_voyage (external project) reported this bug - send response after fix

---

**Sprint created:** 2025-11-29
**Based on design doc:** [m-bug-record-update-inference.md](m-bug-record-update-inference.md)
