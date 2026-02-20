# Sprint Plan: Fix Inline Tests with Complex Imports

**Sprint ID**: M-DX25
**Duration**: 2-3 days (10-12 hours)
**Target Version**: v0.7.2
**Priority**: P1 - Critical bug blocking practical inline testing
**Risk Level**: Medium

## Goal

Fix the critical bug where inline tests fail with "cannot apply non-function value: <nil>" when testing functions that call imported stdlib functions (std/fs, std/net, std/json).

## Current State

**What's Working**:
- Inline tests work for pure functions with no imports ✓
- Inline tests work with builtin functions ✓
- Inline tests parse and extract correctly ✓
- Test harness construction is correct ✓

**What's Broken**:
- ❌ Inline tests fail when function calls imported functions
- ❌ Error: "harness evaluation failed: cannot apply non-function value: <nil>"
- ❌ Affects modules importing std/fs, std/net, std/json
- ❌ Affects transitive function calls (user function → helper → stdlib)

**Root Cause**:
Environment variable capture in FunctionValue prevents resolution of imported functions during test harness evaluation. The GlobalResolver isn't being used correctly for nested function calls.

## Implementation Plan

### Milestone 1: Understand and Diagnose (0.5 days)

**Objective**: Fully understand the evaluation flow and confirm the root cause

**Tasks**:
1. Create minimal reproduction case
   - Module with std/fs import
   - Pure function that calls imported function
   - Inline test that calls that function
2. Add DEBUG logging to trace evaluation flow
3. Confirm that GlobalResolver IS initialized correctly
4. Confirm that FunctionValue Env capture IS the issue
5. Document the exact point of failure

**Acceptance Criteria**:
- [ ] Reproduction case created and fails as expected
- [ ] Can trace execution and see where nil is returned
- [ ] Understand why GlobalResolver isn't consulted

**Estimated LOC**: 20 LOC (test case + debug output)
**Estimated Time**: 0.5 days

---

### Milestone 2: Implement Core Fix (1 day)

**Objective**: Fix the FunctionValue environment capture issue

**Tasks**:
1. Modify `injectModuleBindings()` in executor.go
   - Change from capturing `env` to passing `nil` or a special marker
   - Add comment explaining why we don't capture environment
2. Verify CombinedResolver handles nil Env case correctly
3. Test the fix with reproduction case
4. Run full test suite to catch breakage
5. Fix any failing tests caused by the change

**Key Files to Modify**:
- `internal/testing/executor.go` (lines 726-779)
- `internal/eval/eval_evaluator.go` (if needed, for nil Env handling)

**Implementation Pattern**:

```go
// In injectModuleBindings, for each imported function:
funcVal := &eval.FunctionValue{
    Params: extractLambdaParams(lambda),
    Body:   lambda.Body,
    Env:    nil,              // ← Changed: nil means "use global resolver"
    Typed:  true,
}
env.Set(d.Name, funcVal)
```

**Acceptance Criteria**:
- [ ] Reproduction case now passes
- [ ] All existing inline test tests still pass
- [ ] No new test failures
- [ ] Code change is minimal and focused (~50 LOC)

**Estimated LOC**: ~80 LOC (fix + defensive checks)
**Estimated Time**: 1 day (includes test suite run + debugging)

---

### Milestone 3: Add Regression Tests (1 day)

**Objective**: Prevent this bug from regressing

**Tasks**:
1. Un-skip `TestIntegration_InlineTestsWithImports` if it exists
2. Add test for std/fs imports
3. Add test for std/net imports
4. Add test for std/json imports
5. Add test for helper functions (user-defined, not imported)
6. Add test for deep call chains (3+ levels)
7. Create example file `examples/testing_with_imports.ail`
8. Verify example file works with `ailang test`

**Test Cases**:
```go
func TestIntegration_InlineTestsWithStdFsImport(t *testing.T) {
    // Module imports std/fs
    // Function calls fs function
    // Inline tests pass
}

func TestIntegration_InlineTestsWithHelperFunctions(t *testing.T) {
    // Module defines multiple functions
    // Main function calls helper
    // Inline tests on main pass
}

func TestIntegration_InlineTestsWithDeepCallChains(t *testing.T) {
    // main -> helper1 -> helper2 -> stdlib.func
    // All should resolve correctly
}
```

**Acceptance Criteria**:
- [ ] 3+ new integration tests added
- [ ] All tests pass
- [ ] Example file created and verified
- [ ] Example file included in repo
- [ ] Test coverage for transitive calls

**Estimated LOC**: ~150 LOC (tests + examples)
**Estimated Time**: 1 day (includes writing + debugging)

---

## Task Breakdown by Day

### Day 1 (0.5 days): Diagnosis + Initial Fix
**Time**: 3-4 hours

1. **Morning (1 hour)**: Create reproduction case
   - Write test module with std/fs import
   - Verify it fails with expected error
   - Add debug logging

2. **Morning (1.5 hours)**: Understand the flow
   - Trace through executor.go EvaluateInlineTestsWithHarness
   - Understand how CombinedResolver is constructed
   - Confirm FunctionValue Env capture is the issue

3. **Afternoon (1 hour)**: Implement the fix
   - Modify injectModuleBindings() to pass Env: nil
   - Run reproduction case
   - Verify it passes

### Day 2 (1 day): Testing and Validation
**Time**: 6-8 hours

1. **Morning (2 hours)**: Run test suite
   - Run `make test` to catch any breakage
   - Fix failing tests (if any)
   - Document findings

2. **Mid-morning (1.5 hours)**: Add regression tests
   - Create test functions for std/fs, std/net, std/json
   - Create test for helper functions
   - Create test for deep call chains

3. **Afternoon (2 hours)**: Create example file
   - Write `examples/testing_with_imports.ail`
   - Test with `ailang test examples/testing_with_imports.ail`
   - Verify output is correct

4. **Late afternoon (1 hour)**: Final verification
   - Run full test suite again
   - Check code coverage for new tests
   - Review changes for quality

### Day 3 (0.5 days): Documentation and Cleanup
**Time**: 2-3 hours

1. **Morning (1 hour)**: Update documentation
   - Add note to LIMITATIONS.md about the fix
   - Update CHANGELOG.md with the fix
   - Add comment to code explaining the change

2. **Morning (1 hour)**: Final testing
   - Test with real modules (gcp_auth.ail, ga4_queries.ail)
   - Verify no regressions
   - Clean up debug code

3. **Afternoon (0.5 hours)**: Commit and summary
   - Commit all changes with descriptive message
   - Create summary of what was fixed
   - Document lessons learned

## Success Metrics

### Testing Coverage
- [ ] Inline tests now pass with std/fs imports
- [ ] Inline tests now pass with std/net imports
- [ ] Inline tests now pass with std/json imports
- [ ] Helper function calls work in inline tests
- [ ] Transitive function calls (3+ levels) work
- [ ] All existing tests still pass (no regressions)

### Code Quality
- [ ] Code change is minimal and focused (~80 LOC total)
- [ ] New tests have clear assertions and documentation
- [ ] Example file is well-commented and educational
- [ ] No DEBUG code left in production

### Documentation
- [ ] CHANGELOG.md updated with the fix
- [ ] LIMITATIONS.md updated if needed
- [ ] Example file created and verified working
- [ ] Code comments explain the change

## Risk Mitigation

### Risk 1: FunctionValue Changes Break Other Code
**Likelihood**: Medium
**Impact**: Test failures, silent bugs
**Mitigation**:
- Run full test suite after change
- Use DEBUG_STRICT=1 to catch panics
- Review all callers of FunctionValue creation

### Risk 2: GlobalResolver Doesn't Handle All Cases
**Likelihood**: Low
**Impact**: New failures in test execution
**Mitigation**:
- Add defensive error handling
- Test with multiple import scenarios
- Verify ADT constructors still work

### Risk 3: Performance Impact
**Likelihood**: Very Low
**Impact**: Slightly slower test execution
**Mitigation**:
- Profile if needed (likely negligible)
- Consider caching if bottleneck found

## Dependencies

**None** - This is a standalone bug fix. All dependencies (inline test infrastructure, GlobalResolver, etc.) already exist.

## Prerequisites

- Knowledge of inline test harness (internal/testing/)
- Understanding of eval.FunctionValue and environment handling
- Familiarity with GlobalResolver pattern

## Related Bugs/Design Docs

- M-BUG-ADT-TEST-HARNESS-SCOPE (v0.5.0) - Similar scoping issue
- M-TESTING-INLINE-CORE-EVALUATION (v0.4.7) - Inline test implementation
- M-DX23-INLINE-TESTS-DOCUMENTATION (v0.7.1) - Inline test documentation

## Velocity Data

Based on recent fixes to inline test infrastructure:
- Similar scoping bug (ADT constructors): ~1.5 days
- Inline test parser additions: ~2 days
- Test infrastructure improvements: ~1-2 days

**Estimate**: 2-3 days is realistic and conservative

## Notes

- This is a critical bug that blocks practical use of inline tests
- The fix is relatively simple (environment capture issue) but requires careful testing
- Example files are essential to verify the fix works in practice
- Consider combining with M-DX23 documentation update for next release

## Files to Modify

**Core Fix**:
- `internal/testing/executor.go` - injectModuleBindings()

**New Tests**:
- `internal/testing/integration_test.go` - Add 3+ tests

**New Examples**:
- `examples/testing_with_imports.ail` - Example with stdlib imports

**Documentation**:
- `CHANGELOG.md` - Note the fix
- `LIMITATIONS.md` - Update if needed

## Rollout Plan

1. **Day 3 End**: Commit fix to feature branch
2. **Code Review**: Ensure no regressions
3. **Merge to Dev**: Integrate with main development
4. **v0.7.2 Release**: Include in next patch release
5. **Announce**: Include in release notes with example

## Total Estimated Effort

- **Implementation**: 0.5 days (80 LOC)
- **Testing**: 1 day (150 LOC)
- **Documentation**: 0.5 days (example + changelog)
- **Buffer**: 0.5 days (debugging, unexpected issues)
- **TOTAL**: 2.5 days (medium estimate: 2-3 days)

**This is a realistic estimate based on:**
- Minimal code change required
- Clear understanding of the root cause
- Existing test infrastructure
- Known solution pattern (use global resolver)
