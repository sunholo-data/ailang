# M-TESTING-INLINE: Core Evaluation Fix - Sprint Plan

**Sprint ID**: M-TESTING-INLINE-CORE
**Duration**: 2-3 days (13-18 hours)
**Start**: 2025-11-27
**Target**: v0.4.7
**Risk Level**: Low (isolated changes, clear approach)

## Sprint Summary

**Goal**: Unblock M-TESTING-INLINE by implementing Core evaluation support for inline tests.

**Problem**: Inline tests fail at runtime because LetRec bindings don't scope across array elements. Tests call `factorial(0)` but `factorial` is undefined.

**Solution**: Build per-function test harness as synthetic Core expression with nested LetRecs.

**Key Deliverables**:
1. `internal/testing/harness.go` - Test harness builder (~80 LOC)
2. Updated `internal/testing/executor.go` - Use harness builder (~30 LOC)
3. Passing inline tests: `ailang test examples/factorial.ail` shows 4/4 ✅
4. M-TESTING-INLINE feature 100% complete (currently 95%)

## Current Status Analysis

### Completed (95% of M-TESTING-INLINE)
- ✅ Parser: Inline test syntax `tests [(input, expected)]` (100%)
- ✅ AST: FuncDecl.Tests, TestCase structures (100%)
- ✅ Collector: Extracts tests from AST (100%)
- ✅ Runner: Framework for test execution (100%)
- ✅ Generators: Type-directed value generation (100%)
- ✅ Shrinking: QuickCheck-style minimization (100%)
- ✅ CLI: `ailang test` command (100%)

**Total completed**: ~4,892 LOC across 8 files

### Remaining Work (5% - This Sprint)
- ❌ Core evaluation: Tests can't reference function being tested
- ❌ Integration: End-to-end inline test execution
- ❌ Examples: Verify factorial.ail and other examples work

**Blocker**: Core LetRec scoping prevents test execution

### Recent Velocity
From last 14 days of CHANGELOG:
- Pattern sugar: ~519 LOC (parser 49 + tests 470)
- Concat operator: ~260 LOC (impl 110 + tests 150)
- Nullary patterns: ~133 LOC (impl 13 + tests 120)

**Average velocity**: ~140 LOC/day (including tests)

**This sprint estimate**: ~280 LOC total (harness 80 + executor 30 + tests 170)
**Expected duration**: 2 days at current velocity, 3 days with buffer

## Milestone Breakdown

### Milestone 1: Test Harness Builder (Day 1, 6-8 hours)

**Goal**: Create `BuildInlineTestHarness()` function that transforms (function binding, tests) → Core expression.

**Tasks**:
- [ ] Create `internal/testing/harness.go` (80 LOC)
  - Implement `BuildInlineTestHarness(binding, tests)` function
  - Build nested LetRec with test calls in body
  - Handle single test case
  - Handle multiple test cases (nested Lets)
  - Handle multi-argument functions (curried application)
- [ ] Create `internal/testing/harness_test.go` (120 LOC)
  - Unit test: single test case
  - Unit test: multiple test cases
  - Unit test: multi-argument functions
  - Unit test: verify Core AST structure
- [ ] Manual verification
  - Print generated Core AST, verify structure
  - Compare to examples in design doc

**Estimated LOC**: 200 LOC (implementation 80 + tests 120)

**Acceptance Criteria**:
- [ ] `BuildInlineTestHarness()` produces valid Core expressions
- [ ] Generated AST matches design doc examples
- [ ] All unit tests pass (8-10 tests)
- [ ] Harness handles edge cases (0 tests, 10+ tests)

**Risk Factors**:
- Core AST construction details (node IDs, positions, types)
- Tuple construction for result values
- **Mitigation**: Reference existing Core builders in elaborator

### Milestone 2: Executor Integration (Day 2, 4-6 hours)

**Goal**: Update executor to use harness builder and evaluate inline tests.

**Tasks**:
- [ ] Update `internal/testing/executor.go` (30 LOC changes)
  - Import harness package
  - Extract function LetRec binding from elaborated module
  - Call `BuildInlineTestHarness()` for each tested function
  - Evaluate harness expression
  - Extract tuple results
  - Compare actuals to expected values
- [ ] Update `internal/testing/executor_test.go` (50 LOC)
  - Integration test: factorial with inline tests
  - Integration test: multi-argument function (add)
  - Integration test: multiple tests per function
- [ ] **Verify cross-definition scoping** (CRITICAL)
  - Create test with helper function reference
  - If fails, document for Approach 2 fallback

**Estimated LOC**: 80 LOC (implementation 30 + tests 50)

**Acceptance Criteria**:
- [ ] `ailang test examples/factorial.ail` shows 4/4 tests passing
- [ ] Executor properly extracts and compares results
- [ ] Test output format unchanged (backward compatible)
- [ ] **Cross-definition test passes OR documented for fallback**

**Risk Factors**:
- **HIGH**: Functions that reference helpers/constants may fail
- Extracting LetRec bindings from elaborated module
- **Mitigation**: Phase 2 includes explicit verification; Approach 2 ready as fallback

### Milestone 3: Testing & Validation (Day 2-3, 4-6 hours)

**Goal**: Comprehensive testing and verification of inline test feature.

**Tasks**:
- [ ] End-to-end testing
  - Run `ailang test examples/factorial.ail` → 4/4 passing
  - Run `ailang test examples/testing_basic.ail` → verify standalone tests still work
  - Test recursive functions (factorial, fibonacci)
  - Test multi-argument functions
  - Test edge cases (0 tests, 1 test, 10+ tests)
- [ ] Cross-definition scoping tests
  - Create example with helper function
  - Create example with module constant
  - **If fails**: Implement Approach 2 (env accumulation) - estimated +6-8 hours
- [ ] Regression testing
  - Run full test suite: `make test` → All 110+ tests passing
  - Run example verification: `make verify-examples`
  - Ensure no regression in normal module execution
- [ ] Documentation updates
  - Add implementation notes to design doc
  - Update M-TESTING-INLINE status to "Implemented"
  - Document any limitations discovered

**Estimated LOC**: 0 LOC (testing only, no new code unless Approach 2 needed)

**Acceptance Criteria**:
- [ ] All M-TESTING-INLINE integration tests pass
- [ ] `ailang test examples/factorial.ail` → 4/4 passing
- [ ] Recursive functions work in inline tests
- [ ] All 110+ existing tests still pass (no regression)
- [ ] M-TESTING-INLINE feature is 100% complete
- [ ] Documentation updated with "Implemented" status

**Risk Factors**:
- Cross-definition scoping may require Approach 2 fallback (+6-8 hours)
- Unknown edge cases discovered during testing
- **Mitigation**: Approach 2 design is ready; can add incrementally

## Day-by-Day Breakdown

### Day 1 (6-8 hours)
**Focus**: Milestone 1 - Test Harness Builder

**Morning** (3-4 hours):
- Create `internal/testing/harness.go`
- Implement `BuildInlineTestHarness()` core logic
- Handle single test case first

**Afternoon** (3-4 hours):
- Extend to multiple test cases (nested Lets)
- Handle multi-argument functions
- Create unit tests in `harness_test.go`
- Verify generated Core AST structure

**End-of-day checkpoint**:
- ✅ Harness builder complete and tested
- ✅ 8-10 unit tests passing
- ✅ Generated AST matches design doc

### Day 2 (4-6 hours)
**Focus**: Milestone 2 - Executor Integration

**Morning** (2-3 hours):
- Update `executor.go` to use harness builder
- Extract LetRec bindings from module
- Evaluate harness and compare results

**Afternoon** (2-3 hours):
- Create integration tests
- **CRITICAL**: Test cross-definition scoping
- Run `ailang test examples/factorial.ail`
- Debug any failures

**End-of-day checkpoint**:
- ✅ `ailang test examples/factorial.ail` → 4/4 passing
- ✅ Integration tests pass
- ⚠️ Cross-definition scoping verified OR documented for fallback

### Day 3 (3-4 hours)
**Focus**: Milestone 3 - Testing & Validation

**Morning** (2-3 hours):
- Comprehensive end-to-end testing
- Test edge cases and recursion
- Run full test suite for regression

**Afternoon** (1-2 hours):
- Documentation updates
- Move design doc to implemented/
- Update M-TESTING-INLINE status

**End-of-day checkpoint**:
- ✅ All tests passing
- ✅ No regressions
- ✅ M-TESTING-INLINE 100% complete

## Files to Create/Modify

### New Files
| File | Purpose | LOC | Milestone |
|------|---------|-----|-----------|
| `internal/testing/harness.go` | Test harness builder | 80 | M1 |
| `internal/testing/harness_test.go` | Unit tests | 120 | M1 |

### Modified Files
| File | Changes | LOC | Milestone |
|------|---------|-----|-----------|
| `internal/testing/executor.go` | Use harness builder | 30 | M2 |
| `internal/testing/executor_test.go` | Integration tests | 50 | M2 |

### Contingency Files (If Approach 2 Needed)
| File | Purpose | LOC | Time |
|------|---------|-----|------|
| `internal/testing/envaccum.go` | Env accumulator | 50 | +3-4h |
| `internal/testing/envaccum_test.go` | Tests | 60 | +2-3h |

## Success Metrics

### Quantitative
- [ ] 280 LOC added (base) or 390 LOC (with Approach 2)
- [ ] 4/4 inline tests passing in `examples/factorial.ail`
- [ ] All 110+ existing tests still pass
- [ ] Test coverage maintained or improved
- [ ] 0 regressions introduced

### Qualitative
- [ ] Inline tests execute correctly
- [ ] Error messages are clear and actionable
- [ ] Code is maintainable (isolated to testing package)
- [ ] Design doc accurately reflects implementation
- [ ] Examples demonstrate feature usage

### Feature Completeness
- [ ] Parser ✅ (100%)
- [ ] AST ✅ (100%)
- [ ] Collector ✅ (100%)
- [ ] Runner ✅ (100%)
- [ ] Generators ✅ (100%)
- [ ] Shrinking ✅ (100%)
- [ ] **Executor ✅ (100%)** ← THIS SPRINT
- [ ] CLI ✅ (100%)
- [ ] Examples ✅ (100%)

**M-TESTING-INLINE: 100% COMPLETE** 🎉

## Dependencies

**Blockers** (must complete first):
- None (all dependencies complete)

**Blocked by this sprint**:
- M-TESTING-INLINE feature release
- Example file creation for inline tests
- Documentation of inline test feature

## Open Questions

1. **Cross-definition scoping**: Will simple harness work for all cases?
   - **Answer by**: End of Day 2 (Milestone 2)
   - **If no**: Implement Approach 2 fallback

2. **Performance**: Is harness evaluation fast enough?
   - **Measure**: Run benchmarks in Milestone 3
   - **Threshold**: < 1ms per test case

3. **Error messages**: Are test failures clear?
   - **Verify**: Check error output in Milestone 2
   - **Improve**: If needed, enhance error reporting

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Cross-definition scoping fails | Medium | High | Approach 2 ready as fallback (+6-8h) |
| Core AST construction complexity | Low | Medium | Reference existing elaborator code |
| Integration test failures | Low | Medium | Comprehensive unit testing first |
| Performance issues | Low | Low | Harness evaluation is minimal overhead |
| Timeline slippage | Low | Medium | 3-day buffer built into estimate |

**Overall Risk Level**: **Low**
- Clear technical approach
- Isolated changes (test package only)
- Fallback plan ready (Approach 2)
- No changes to production code paths

## Progress Tracking

**JSON Progress File**: `.ailang/state/sprint_M-TESTING-INLINE-CORE.json`

**Status Updates**:
- Milestone 1: Update `passes` field when harness builder complete
- Milestone 2: Update when executor integration complete
- Milestone 3: Update when all tests passing

**Resumption**: Use `ailang agent inbox sprint-executor` to check status between sessions.

## References

- **Design Doc**: [design_docs/planned/v0_4_7/m-testing-inline-core-evaluation.md](m-testing-inline-core-evaluation.md)
- **M-TESTING-INLINE Sprint**: `.ailang/state/sprints/sprint_M-TESTING-INLINE.json`
- **Parser Implementation**: `internal/parser/parser_test.go`
- **Current Executor**: `internal/testing/executor.go` (needs fixing)
- **Core AST**: `internal/core/core.go` (LetRec semantics)

---

**Created**: 2025-11-27
**Last Updated**: 2025-11-27
**Status**: Ready for execution
