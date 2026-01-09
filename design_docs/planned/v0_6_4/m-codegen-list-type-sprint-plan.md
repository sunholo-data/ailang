# M-CODEGEN-LIST-TYPE: Sprint Plan

**Status**: Ready for Execution
**Target**: v0.6.4
**Duration**: 2-3 days (4-6 hours estimated work)
**Priority**: P1 (Critical Compilation Issue)
**Created**: 2026-01-09
**Executor**: sprint-executor

---

## Sprint Summary

**Goal**: Fix the undefined List type in AILANG-generated Go code, enabling ADTs with list fields to compile cleanly.

**Key Outcomes**:
1. ✅ All AILANG ADTs with list fields generate valid, compilable Go code
2. ✅ Unified type mapping eliminates undefined reference fallbacks
3. ✅ Pre-codegen validation catches type mapping issues before code generation
4. ✅ Comprehensive test coverage for list field ADTs (3+ test cases)
5. ✅ Example programs demonstrating list field functionality

**Risk Level**: Low (isolated feature area, existing infrastructure in place)

---

## Current Status

### Completed Infrastructure
- **List handling in TypeMapper** (types.go:72-77, 125-138): TList and TApp("List", T) → []T conversion ✅
- **Type mapper architecture** (types.go): Comprehensive type mapping for all AILANG types ✅
- **Cycle detection** (types.go:56-66): M-DX11-TRAVERSE cycle-safe traversal ✅
- **Reserved name checks**: Some validation exists for user-defined types ✅

### Remaining Work
1. **Root Cause Investigation** (1h): Identify which code path creates `*List` references
2. **Type Mapping Fix** (1h): Ensure all paths use consistent List handling
3. **Validation Layer** (1h): Add pre-codegen validation
4. **Test Coverage** (1.5h): Verify all scenarios work
5. **Documentation** (0.5h): Update changelog and examples

---

## Milestone Breakdown

### Milestone 1: Root Cause Analysis (1 hour)
**Goal**: Identify the exact code path that generates `*List` references.

**Tasks**:
- [ ] Create minimal AILANG test file with list field ADT
- [ ] Run with DEBUG_CODEGEN=1 to trace type mapping
- [ ] Identify which AST node creates the problematic type reference
- [ ] Document findings in code comments

**Acceptance Criteria**:
- [ ] Root cause identified and documented
- [ ] Minimal test case created and reproduces issue
- [ ] Code path traced from AST to generated Go code

**Estimated LOC**: 30 (test case + comments)

---

### Milestone 2: Type Mapping Fix (1 hour)
**Goal**: Ensure all TypeMapper paths handle List types consistently.

**Files to Modify**:
- `internal/gen/golang/types.go` (mapTCon function)

**Tasks**:
- [ ] Add "List" reserved name check to mapTCon()
- [ ] Verify TList case works for all element types
- [ ] Verify TApp("List", T) case works for all element types
- [ ] Run existing tests to verify no regression

**Acceptance Criteria**:
- [ ] No code path can create bare `*List` references
- [ ] All List references go through TList or TApp handlers
- [ ] All existing type mapping tests pass

**Estimated LOC**: 35 (code + error handling)

---

### Milestone 3: Pre-Codegen Validation (1 hour)
**Goal**: Add validation layer to catch unmappable types before code generation.

**Files to Create**:
- `internal/gen/golang/validate_types.go` (NEW)

**Files to Modify**:
- `cmd/ailang/compile.go` (integrate validation)

**Tasks**:
- [ ] Create ValidateTypeMapping() function
- [ ] Integrate into compile pipeline (after type checking, before codegen)
- [ ] Design error messages that explain the issue and suggest fixes

**Acceptance Criteria**:
- [ ] All types in program can be validated
- [ ] Error messages guide users to fix the issue
- [ ] No performance impact on execution

**Estimated LOC**: 130 (validate_types.go + integration)

---

### Milestone 4: Test Coverage (1.5 hours)
**Goal**: Comprehensive test coverage for list field ADTs.

**Files to Modify**:
- `internal/gen/golang/types_test.go`
- `internal/gen/golang/adt_test.go`

**Test Cases**:
1. Primitive list fields
2. Record list fields
3. Nested lists
4. Type validation
5. Integration test with code compilation

**Acceptance Criteria**:
- [ ] All 5 test cases pass
- [ ] No regressions in existing tests
- [ ] Generated Go code actually compiles

**Estimated LOC**: 250 (test cases + test data)

---

### Milestone 5: Documentation & Examples (0.5 hours)
**Goal**: Document the fix and provide working examples.

**Files to Create**:
- `examples/adt_with_lists.ail` (NEW)

**Files to Modify**:
- `CHANGELOG.md`

**Acceptance Criteria**:
- [ ] Example file created and verified working
- [ ] CHANGELOG.md updated with clear description
- [ ] Example demonstrates list fields

**Estimated LOC**: 45 (example + changelog)

---

## Task Breakdown by Day

### Day 1 (2-2.5 hours)
**Focus**: Investigation and Foundation

1. **AM (1h)**: Root Cause Analysis (Milestone 1)
2. **AM (1h)**: Type Mapping Fix (Milestone 2)
3. **PM (0.5h)**: Setup validation infrastructure

### Day 2 (2-2.5 hours)
**Focus**: Implementation and Testing

1. **AM (1h)**: Validation Implementation (Milestone 3)
2. **PM (1.5h)**: Test Coverage (Milestone 4)

### Day 3 (0.5-1 hour)
**Focus**: Documentation and Completion

1. **AM (0.5h)**: Documentation (Milestone 5)
2. **Final check**: Run full test suite

---

## Success Metrics

**Implementation Metrics**:
- [ ] All AILANG ADTs with list fields compile to valid Go code
- [ ] No `undefined: List` errors in generated code
- [ ] All 5+ test cases passing
- [ ] Example programs execute correctly
- [ ] Zero regressions in existing tests

**Code Quality Metrics**:
- [ ] Code coverage > 90% for modified files
- [ ] No lint errors
- [ ] Clean error messages for validation failures

---

## Files Modified/Created

### New Files (170 LOC)
- `internal/gen/golang/validate_types.go` (120 LOC)
- `examples/adt_with_lists.ail` (30 LOC)
- Test files (20 LOC)

### Modified Files (300+ LOC)
- `internal/gen/golang/types.go` (35 LOC)
- `internal/gen/golang/types_test.go` (100 LOC)
- `internal/gen/golang/adt_test.go` (150 LOC)
- `cmd/ailang/compile.go` (10 LOC)
- `CHANGELOG.md` (10 LOC)

**Total**: ~500 LOC

---

## Dependencies & Blockers

**None** - All infrastructure exists:
- TypeMapper already supports List types
- Type checking already validates List types
- Compile pipeline ready for validation insertion

---

## Related Design Documents

- [M-CODEGEN-LIST-TYPE-DEFINITION.md](./m-codegen-list-type-definition.md) - Original design doc
- [M-DX25-LIST-TYPE-CODEGEN.md](../implemented/v0_6_3/m-dx25-list-type-codegen.md) - How List types work

---

**Ready for: sprint-executor**

This sprint plan provides a clear, achievable path to fixing the undefined List type issue while maintaining code quality and comprehensive test coverage.
