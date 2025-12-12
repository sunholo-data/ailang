# Sprint Plan: M-TYPENAME-NESTED-PROPAGATION

## Summary
Fix TypeName propagation for nested record literals in the type checker, enabling proper nominal type identity for records nested within struct fields, and removing the codegen workaround.

**Duration:** 1 day (4-6 hours)
**Dependencies:** v0.5.10 TypeName fixes already implemented
**Risk Level:** Medium (touches type unification, needs careful testing)

## Current Status Analysis

### Completed Recently (v0.5.10)
- Fix 1: TypeName preservation in substitution (`unification_substitution.go`) - 1 LOC
- Fix 2: TypeName propagation during TRecord unification (`unification_records.go`) - 8 LOC
- Codegen workaround for nested records (`codegen_ops.go:174-190`) - ~20 LOC
- Tests for TypeName preservation (`unification_substitution_test.go`) - ~120 LOC

### Velocity
- Recent average: ~2800 LOC/day (high productivity day)
- Estimated capacity: ~150 LOC for this small sprint
- This sprint estimated: ~100 LOC total

### Problem Statement
Nested record literals don't receive TypeName in CoreTypeInfo because:
1. Elaboration creates fresh TRecord for each nested record
2. Unification propagates TypeName between operands
3. CoreTypeInfo stores a **different** TRecord instance (not updated)

### Current Workaround
Codegen guesses expected type from field definition - works but architecturally wrong.

## Proposed Milestones

### Milestone 1: Investigation & Test Case
**Goal:** Trace type checking flow, confirm hypothesis, create failing test
**Estimated:** 30 LOC (test) + investigation
**Duration:** 1 hour

**Tasks:**
- Trace type checking of nested record using debug mode
- Identify exact code location where record literal types are unified
- Confirm CoreTypeInfo entries for nested records lack TypeName
- Create unit test that reproduces the issue at type-checker level

**Acceptance Criteria:**
- [ ] Test in `typechecker_typename_test.go` shows TypeName missing for nested record
- [ ] Exact code location identified for fix (file:line)
- [ ] Data flow understood and documented

**Risks:**
- Unification more complex than expected - Mitigation: May need Option B (pass expected type through elaboration)

### Milestone 2: Core Implementation
**Goal:** Update CoreTypeInfo after unification with the unified result
**Estimated:** 30 LOC (implementation) + 50 LOC (tests)
**Duration:** 2 hours

**Tasks:**
- Implement Option A: Update CoreTypeInfo[nodeID] after unification
- Handle pointer types (*SystemPos vs SystemPos)
- Handle multiple nesting levels
- Add DEBUG_TYPECHECKER flag for verbose logging
- Write regression tests

**Acceptance Criteria:**
- [ ] Nested `{x: 0.0, y: 0.0, z: 0.0}` in `*SystemPos` field has `TypeName: "SystemPos"` in CoreTypeInfo
- [ ] Multi-level nesting works correctly
- [ ] All existing tests pass
- [ ] New regression tests pass

**Risks:**
- Breaking existing type inference - Mitigation: Run full test suite after each change

### Milestone 3: Cleanup & Verification
**Goal:** Remove codegen workaround, verify with stapledons_voyage
**Estimated:** -20 LOC (remove workaround) + docs
**Duration:** 1-2 hours

**Tasks:**
- Remove codegen workaround in `codegen_ops.go:174-190`
- Verify stapledons_voyage sim/ compiles with original type definitions
- Update design docs (move to implemented/)
- Send confirmation message to stapledons_voyage

**Acceptance Criteria:**
- [ ] Codegen workaround removed
- [ ] stapledons_voyage compiles without type rename workarounds
- [ ] Design doc moved to `design_docs/implemented/v0_5_11/`
- [ ] All tests passing
- [ ] Linting clean

## Files to Modify

**Investigation targets:**
- `internal/types/typechecker_expr.go` - Record literal type checking
- `internal/types/typechecker_data.go` - CoreTypeInfo population

**Implementation:**
- `internal/types/typechecker_expr.go` - Update CoreTypeInfo after unification (~+15 LOC)
- `internal/types/typechecker_typename_test.go` - Regression tests (NEW, ~80 LOC)

**Cleanup:**
- `internal/gen/golang/codegen_ops.go` - Remove workaround (~-20 LOC)

## Success Metrics
- Test coverage: No regression
- All tests passing: make test
- Linting clean: make lint
- stapledons_voyage sim/ compiles
- Documentation updated

## Dependencies
- v0.5.10 TypeName fixes (DONE)
- Understanding of type checking flow (M1 deliverable)

## Open Questions
- None - approach is clear from design doc

## Notes
- If Option A doesn't work, fall back to Option B (pass expected type through elaboration) or Option C (post-pass to fix CoreTypeInfo)
- This completes the type metadata preservation work started in v0.5.10
- GitHub Issue #38 will be fully resolved

---

**Created:** 2025-12-12
**Target:** v0.5.11
