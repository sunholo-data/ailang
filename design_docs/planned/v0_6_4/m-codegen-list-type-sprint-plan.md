# Sprint Plan: M-CODEGEN-LIST-TYPE-DEFINITION

**Sprint ID**: M-CODEGEN-LIST-TYPE
**Target Version**: v0.6.4
**Duration**: 2-3 days (~6 hours)
**Risk Level**: Low
**Status**: Ready for Implementation

---

## 📋 Sprint Summary

**Goal**: Fix undefined List type errors in AILANG codegen by implementing comprehensive type mapping validation and unification.

**Key Deliverables**:
1. ✅ Root cause identified and documented in test case
2. ✅ Unified List type handling in TypeMapper
3. ✅ Pre-codegen validation to catch type mapping issues early
4. ✅ Test coverage for all List type scenarios
5. ✅ Working example showing list field ADTs
6. ✅ CHANGELOG updated

**Why This Matters**: Currently, ADTs with list fields fail to compile with `undefined: List` errors. This is blocking users from writing realistic data structures (e.g., game state with lists of entities, system models with arrays of components).

---

## 📊 Current Status Analysis

### Velocity Data (Last 14 days)
- Recent commits span telemetry improvements, coordinator features, and refactoring
- Typical feature implementation: 150-250 LOC per day
- Bug fixes with systemic analysis: 4-6 hours
- Test coverage ratio: 30-40% of implementation LOC

### What's Already Done
- TypeMapper has partial List handling (TList and TApp cases)
- Runtime has List helper functions (Cons, ListHead, ListTail, ListLen)
- ADT code generation framework exists

### What's Missing
- **Root cause identification**: Which codegen path creates `*List` references
- **Complete type mapping**: All paths don't use TList → []T conversion
- **Validation layer**: No pre-codegen checks for unmappable types
- **Test coverage**: No tests for ADTs with list fields

---

## 🎯 Proposed Milestones

### Milestone 1: Root Cause Investigation (1 hour, 0 LOC)

**Description**: Create minimal test case that reproduces the issue and trace the codegen path.

**Tasks**:
- [ ] Write minimal AILANG test case with list field ADT
  ```ailang
  type Item = { name: string, tags: [string] }
  ```
- [ ] Run codegen with DEBUG_CODEGEN=1 to trace path
- [ ] Identify exact location where `*List` reference is created
- [ ] Document findings in code comments

**Dependencies**: None

**Acceptance Criteria**:
- [ ] Test case compiles AILANG code
- [ ] Debug output shows codegen flow
- [ ] Comments explain which code path creates `*List`
- [ ] Evidence recorded in test file

**Estimated LOC**: 0 (investigation only)
**Risk**: Low (tracing, no changes)

---

### Milestone 2: Type Mapping Fix (1 hour, 200 LOC)

**Description**: Implement unified List type handling in TypeMapper and add reserved name checks.

**Tasks**:
- [ ] Add "List" reserved name check to `mapTCon()` in `internal/gen/golang/types.go`
  - When List is used as bare type (not applied), return clear error
  - Force users to use TList(T) or TApp("List", T)
- [ ] Add comprehensive TData case handling in `mapTypeWithVisited()`
  - Handle both "List" and "[]" as list type names
  - Extract element type and generate []T
  - Fallback to []interface{} for bare List
- [ ] Add edge case handling for List with no type arguments
- [ ] Verify TList and TApp("List", T) both work correctly

**Files Modified**:
- `internal/gen/golang/types.go` (~100 LOC modified)

**Dependencies**: Milestone 1

**Acceptance Criteria**:
- [ ] mapTCon("List") returns clear error message
- [ ] TList(stringType) maps to "[]string"
- [ ] TApp("List", intType) maps to "[]int"
- [ ] List with no args maps to "[]interface{}"
- [ ] Existing tests still pass
- [ ] No performance regression

**Estimated LOC**: 200
**Risk**: Low (isolated change, existing infrastructure)

---

### Milestone 3: Add Validation Layer (1 hour, 150 LOC)

**Description**: Implement pre-codegen validation to catch type mapping failures early.

**Tasks**:
- [ ] Create `internal/gen/golang/validate_types.go` (~120 LOC)
  - `ValidateTypeMapping(prog *core.Program)` function
  - Extract all types from program
  - Attempt to map each type
  - Collect errors for diagnostic output
- [ ] Integrate validation into compile pipeline
  - Call in `cmd/ailang/compile.go` after type checking
  - Fail with clear error message before codegen
  - Include type name and mapping failure reason
- [ ] Add tests for validation (~80 LOC in new file)

**Files Modified**:
- `cmd/ailang/compile.go` (~10 LOC)
- **New**: `internal/gen/golang/validate_types.go` (~120 LOC)
- **New**: `internal/gen/golang/validate_types_test.go` (~80 LOC)

**Dependencies**: Milestone 2

**Acceptance Criteria**:
- [ ] ValidateTypeMapping detects unmappable types
- [ ] Error messages clearly identify problem
- [ ] Validation runs before codegen
- [ ] Existing tests still pass
- [ ] No performance impact on execution

**Estimated LOC**: 230
**Risk**: Low (additive, early failure point)

---

### Milestone 4: Test Coverage (1.5 hours, 250 LOC)

**Description**: Add comprehensive test coverage for List type handling.

**Tasks**:
- [ ] Add test case: ADT with list field of primitives (~50 LOC)
  ```go
  type Config struct {
      Tags []string
  }
  ```
- [ ] Add test case: ADT with list field of other ADTs (~60 LOC)
  ```go
  type Container struct {
      Items []*Item
  }
  ```
- [ ] Add test case: Nested list types (~40 LOC)
  ```go
  type Matrix struct {
      Rows [][]float64
  }
  ```
- [ ] Integration test: Codegen ADT and verify generated Go code compiles (~100 LOC)
  - Write AILANG source
  - Run codegen
  - Attempt Go compilation
  - Assert success

**Files Modified**:
- `internal/gen/golang/types_test.go` (~150 LOC added)
- `internal/gen/golang/adt_test.go` (~100 LOC added)

**Dependencies**: Milestone 3

**Acceptance Criteria**:
- [ ] All 4 test cases pass (primitive lists, ADT lists, nested lists, integration)
- [ ] Generated Go code compiles without errors
- [ ] Test code is clear and well-commented
- [ ] Coverage for both TList and TApp forms

**Estimated LOC**: 250
**Risk**: Low (isolated tests)

---

### Milestone 5: Example & Documentation (0.5 hours, 70 LOC)

**Description**: Create working example and update documentation.

**Tasks**:
- [ ] Create `examples/adt_with_lists.ail` (~40 LOC)
  - Define ADTs with list fields
  - Show practical use case (game entities, data records)
  - Include comments explaining each type
  - Verify it runs without errors
- [ ] Update `CHANGELOG.md` (~10 LOC)
  - Document the fix
  - Note which GitHub issue it closes (#116)
  - Include code snippet showing before/after
- [ ] Verify example works with `make verify-examples` (~20 LOC)

**Files Modified**:
- **New**: `examples/adt_with_lists.ail` (~40 LOC)
- `CHANGELOG.md` (~10 LOC)

**Dependencies**: Milestone 4

**Acceptance Criteria**:
- [ ] Example file exists and runs without errors
- [ ] CHANGELOG entry clearly describes the fix
- [ ] Example is included in `examples/` manifest
- [ ] `make verify-examples` shows example as passing

**Estimated LOC**: 70
**Risk**: Low (documentation)

---

## 📅 Day-by-Day Implementation Plan

### Day 1 (2 hours)
- **Morning (1h)**: Milestone 1 - Root Cause Investigation
  - Create test case with list field ADT
  - Debug and trace codegen path
  - Document findings

- **Afternoon (1h)**: Milestone 2 - Type Mapping Fix
  - Modify `mapTCon()` to reject bare List
  - Add TData case handling
  - Run existing tests

### Day 2 (2.5 hours)
- **Morning (1h)**: Milestone 3 - Validation Layer
  - Create validate_types.go
  - Integrate into compile pipeline
  - Test validation against broken code

- **Afternoon (1.5h)**: Milestone 4 - Test Coverage
  - Write 4 test cases
  - Run integration test
  - Verify generated Go code compiles

### Day 3 (1.5 hours)
- **Morning (0.5h)**: Milestone 5 - Example & Documentation
  - Create example file
  - Update CHANGELOG
  - Run verification

- **Afternoon (1h)**: Final Verification
  - Run full test suite (`make test`)
  - Run linter (`make lint`)
  - Verify all examples pass (`make verify-examples`)
  - Commit and prepare for review

---

## 📊 Success Metrics

### Completion Criteria
- [x] Root cause identified and documented
- [x] All paths through TypeMapper use consistent List → []T mapping
- [x] New validation catches type mapping failures early
- [x] All existing tests pass
- [x] New test cases for ADTs with list fields all pass
- [x] Example programs compile and run correctly
- [x] CHANGELOG.md updated with fix description
- [x] No performance regression in codegen
- [x] All linting checks pass

### Test Coverage Target
- Existing coverage: ~30%
- Target: Maintain or improve (no regression)
- New tests: 4 cases covering all List scenarios

### Code Quality
- No performance regression in type mapping (~O(n) with n = number of types)
- Clear error messages for unmappable types
- Validation runs in <100ms for typical programs

---

## 🔗 Dependencies & Blockers

### External Dependencies
- None - this is an isolated codegen fix

### Internal Dependencies
- Depends on existing TypeMapper infrastructure ✅ (ready)
- Depends on validation framework ✅ (ready)
- No breaking changes to existing APIs

### Known Unknowns
- ❓ Exact codegen path creating `*List` (investigation will clarify)
- ❓ Whether similar issues exist with other generic types (deferred to M-CODEGEN-UNIFIED-GENERIC-TYPES)

---

## ⚠️ Risk Assessment

### Low Risk Areas
1. **Type Mapping Fix** - Isolated to one module (types.go)
2. **Validation Layer** - Additive (only catches new errors)
3. **Test Coverage** - Pure tests, no impact on production code

### Moderate Risk Areas
1. **Root Cause Investigation** - Depends on understanding codegen flow
   - **Mitigation**: Start with minimal test case, use debug flags
   - **Fallback**: If path unclear, add validation after all known paths

### Risk Mitigation Strategies
- Start with minimal reproducible test case
- Use debug flags to trace execution
- Run full test suite after each milestone
- Don't commit until all tests pass
- Keep changes small and focused per milestone

---

## 📝 Implementation Notes

### Testing Strategy
- **Unit tests**: TypeMapper.MapType() with various inputs
- **Integration tests**: ADT → Go codegen → go build
- **Example tests**: Run examples and verify output

### Validation Placement
- After type checking passes (types are correct)
- Before codegen starts (fail fast)
- Not in runtime (zero performance impact)

### Error Messages
Should clearly indicate:
- Which type couldn't be mapped
- Why (what was expected vs actual)
- Hint: "Use TList(T) or [T] in AILANG for list fields"

### Files Overview

| File | Changes | Estimate |
|------|---------|----------|
| `internal/gen/golang/types.go` | Add List handling + reserved name check | 100 LOC |
| `internal/gen/golang/validate_types.go` | NEW - Type validation | 120 LOC |
| `cmd/ailang/compile.go` | Call ValidateTypeMapping() | 10 LOC |
| `internal/gen/golang/types_test.go` | Test List type mapping | 150 LOC |
| `internal/gen/golang/adt_test.go` | Test ADT with list fields | 100 LOC |
| `internal/gen/golang/validate_types_test.go` | Validation tests | 80 LOC |
| `examples/adt_with_lists.ail` | Example showing list fields | 40 LOC |
| `CHANGELOG.md` | Document the fix | 10 LOC |

**Total**: ~610 LOC (includes test code)

---

## 🚀 Acceptance & Handoff

### Pre-Handoff Checklist
- [ ] All milestones completed
- [ ] Full test suite passes (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Examples verify (`make verify-examples`)
- [ ] CHANGELOG updated
- [ ] Code reviewed for clarity

### Definition of Done
1. ✅ Code compiles without errors
2. ✅ All tests pass (existing + new)
3. ✅ No linting warnings
4. ✅ Examples work and are documented
5. ✅ CHANGELOG entry present
6. ✅ Documentation/comments clear

### Next Steps (After Sprint)
- Merge to main branch
- Tag as part of v0.6.4 release
- Monitor for any related issues
- Consider M-CODEGEN-UNIFIED-GENERIC-TYPES for Option/Either types

---

## 📚 Related Documents

- **Design Doc**: `design_docs/planned/v0_6_4/m-codegen-list-type-definition.md`
- **Type Mapping**: `internal/gen/golang/types.go`
- **ADT Generation**: `internal/gen/golang/adt.go`
- **Previous Work**:
  - M-DX25-LIST-TYPE-CODEGEN.md
  - M-CODEGEN-UNIFIED-SLICE-CONVERTERS.md

---

## 🎯 Success Indicators

✅ **This sprint succeeds when:**
1. ADTs with list fields compile to valid Go code
2. All existing tests pass with no regression
3. New test cases for list fields all pass
4. Generated Go code compiles successfully
5. Example file demonstrates practical usage
6. CHANGELOG clearly documents the fix
7. No performance impact on codegen

✅ **We'll know we're done when:**
- `make test` returns 0 failures
- `make lint` returns 0 issues
- `make verify-examples` shows adt_with_lists.ail as passing
- GitHub issue #116 can be closed
