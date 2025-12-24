# Sprint Plan: DX-17 Phase 2 - Normalize TList to TApp

## Summary
Eliminate the duplicate `TList` type representation by normalizing `[T]` syntax to `TApp("list", T)` at parse time, creating a uniform internal representation for all container types.

**Duration:** 1 day (~4-6 hours)
**Dependencies:** DX-17 Phase 1 (v0.5.11, completed)
**Risk Level:** Low (incremental refactoring with comprehensive tests)

## Current Status Analysis

### Completed Recently
- ✅ Effect checker bug fix: ~90 LOC in 1 hour
- ✅ Cloud-setup skill: ~850 LOC in 2 hours
- ✅ Test skip conditions: ~22 LOC in 30 min

### Velocity
- Recent average: ~150-200 LOC/day (conservative estimate)
- Estimated capacity: 100-150 LOC for this sprint

### Remaining from Design Doc
- ⏳ M1: Update parser to emit TApp (~30 LOC)
- ⏳ M2: Audit and update TList switch statements (~40 LOC)
- ⏳ M3: Update type printer (~15 LOC)
- ⏳ M4: Deprecate TList struct (~20 LOC)
- 📋 M5: Add migration warning (optional, ~30 LOC)
- 📋 M6: Remove TList (future, v0.6.0)

## Proposed Milestones

### Milestone 1: Update Parser to Emit TApp
**Goal:** Change parser to emit `TApp("list", T)` instead of `ListType` for `[T]` syntax
**Estimated:** 30 LOC implementation + 30 LOC tests = 60 LOC
**Duration:** 1-2 hours

**Tasks:**
- Locate `LBRACKET` handling in `internal/parser/parser_type.go`
- Change `ast.ListType` to `ast.TypeApp{Constructor: "List", Args: [elemType]}`
- Add parser test for list type normalization
- Verify existing tests still pass

**Acceptance Criteria:**
- [ ] `[int]` parses to `TypeApp("List", [int])` not `ListType{Element: int}`
- [ ] `[[string]]` parses to nested `TypeApp`
- [ ] All existing parser tests pass
- [ ] New parser tests added for list normalization

**Risks:**
- AST types may differ from Core types - need to check elaboration
- Mitigation: Check how `ast.ListType` is elaborated to `types.TList`

### Milestone 2: Update Elaboration to Emit TApp
**Goal:** Ensure elaboration emits `types.TApp("list", T)` for list types
**Estimated:** 40 LOC implementation + 20 LOC tests = 60 LOC
**Duration:** 1-2 hours

**Tasks:**
- Find where `ast.ListType` → `types.TList` conversion happens
- Change to `ast.TypeApp` → `types.TApp("list", T)`
- Audit all `case *types.TList:` statements (12 locations per design doc)
- Run tests after each file change

**Acceptance Criteria:**
- [ ] Elaboration produces `TApp("list", T)` for list types
- [ ] All type checker tests pass
- [ ] No runtime panics in list operations

**Risks:**
- Many switch statements to update - could miss one
- Mitigation: Use grep to find all `*TList` references, update systematically

### Milestone 3: Update Type Printer
**Goal:** Ensure `TApp("list", T)` prints consistently as `list[T]`
**Estimated:** 15 LOC implementation + 15 LOC tests = 30 LOC
**Duration:** 30 min

**Tasks:**
- Update `internal/types/safe_string.go` to handle list TApp specially
- Ensure error messages show `list[int]` not `TApp(list, int)`

**Acceptance Criteria:**
- [ ] `TApp("list", int).String()` returns `"list[int]"`
- [ ] Error messages display list types correctly

**Risks:**
- Low - cosmetic change only

### Milestone 4: Deprecate TList Struct
**Goal:** Add deprecation comment and keep backward compatibility
**Estimated:** 20 LOC = 20 LOC
**Duration:** 15 min

**Tasks:**
- Add `// Deprecated:` comment to `TList` struct
- Keep `AsList` helper for convenience

**Acceptance Criteria:**
- [ ] `TList` struct has deprecation comment
- [ ] `AsList` helper still works
- [ ] All tests pass

**Risks:**
- None - documentation only

## Success Metrics
- Test coverage: No regression from current
- Examples passing: All 48+ examples still work
- Documentation: Update design doc status
- All tests passing: ✅
- All linting passing: ✅

## Dependencies
- DX-17 Phase 1 (v0.5.11) - COMPLETE
- `AsList` helper exists and works

## Open Questions
- Should parser emit `ast.TypeApp` or keep `ast.ListType` and only change elaboration?
- Answer: Check if `ast.ListType` is used elsewhere in parser

## Notes
- This is a cleanup/refactoring task - no new features
- Keep changes incremental and test after each milestone
- `TList` removal deferred to v0.6.0 for backward compatibility
