# Sprint Plan: M-BUGFIX v0.5.9 Bug Fix Sprint

## Summary

Fix critical Go codegen bugs and type alias unification issue reported by stapledons_voyage. This sprint unblocks their bridge-interior-v1 sprint Session 4-8.

**Duration:** 1 day (~6 hours)
**Dependencies:** None (all required code is accessible)
**Risk Level:** Low (targeted bug fixes with clear scope)
**Sprint ID:** M-BUGFIX

## Current Status Analysis

### Completed Recently
- v0.5.8: Array codegen support (~195 LOC)
- v0.5.7: Typed slice runtime functions (~150 LOC)
- v0.5.7: Effect traversal fixes (~210 LOC)

### Velocity
- Recent average: ~150-200 LOC/day for targeted fixes
- Estimated capacity: ~400 LOC for this sprint (targeted fixes are efficient)

### Remaining from Design Docs
- M-CODEGEN-BLANK: Blank identifier + return types (~200 LOC)
- M-TYPE-ALIAS: Type alias expansion (~100 LOC)

## Proposed Milestones

### Milestone 1: M1-BLANK-IDENTIFIER
**Goal:** Fix blank identifier (`_`) generating invalid Go code when used as function call argument
**Estimated:** 50 LOC implementation + 30 LOC tests = 80 LOC
**Duration:** 1 hour

**Tasks:**
1. Modify `generateTypedWrapper()` in `codegen_decl.go` to detect `_` parameters
2. Generate `_unused0`, `_unused1`, etc. for blank identifier params
3. Apply same fix to `generateImplFunc()` if needed
4. Add test case with `_` parameter in `codegen_test.go`

**Files to Modify:**
- `internal/gen/golang/codegen_decl.go` (~30 LOC)
- `internal/gen/golang/codegen_test.go` (~50 LOC)

**Acceptance Criteria:**
- [ ] `\_ . expr` functions compile to valid Go
- [ ] Parameters named `_` become `_unused0`, `_unused1`, etc.
- [ ] All existing codegen tests pass
- [ ] `make test` passes

**Risks:**
- None - surgical fix to specific location

### Milestone 2: M2-RETURN-TYPES
**Goal:** Fix exported functions returning wrong types (struct{} instead of actual struct type)
**Estimated:** 80 LOC implementation + 40 LOC tests = 120 LOC
**Duration:** 1.5 hours

**Tasks:**
1. Improve `getTypedSignature()` type resolution in `codegen_decl.go`
2. Add struct type tracking during declaration pass
3. Add ADT type tracking for pointer returns
4. Add test case for struct-returning exported functions

**Files to Modify:**
- `internal/gen/golang/codegen_decl.go` (~50 LOC)
- `internal/gen/golang/types.go` (~30 LOC) - if type mapping needs enhancement
- `internal/gen/golang/codegen_test.go` (~40 LOC)

**Acceptance Criteria:**
- [ ] `InitBridge` returns `*BridgeState` not `struct{}`
- [ ] Struct-returning functions have correct typed signatures
- [ ] ADT-returning functions return pointer types
- [ ] All existing codegen tests pass

**Risks:**
- Medium - need to trace through type representations
- Mitigation: Use existing CoreTypeInfo infrastructure

### Milestone 3: M3-EXPORT-CONVERTERS
**Goal:** Export slice converter functions so external packages can use them
**Estimated:** 20 LOC implementation + 20 LOC tests = 40 LOC
**Duration:** 0.5 hours

**Tasks:**
1. Change `convertTo*Slice` naming to `ConvertTo*Slice` in codegen
2. Update all internal references to use new name
3. Add test for exported converter generation

**Files to Modify:**
- `internal/gen/golang/codegen_decl.go` (~10 LOC)
- `internal/gen/golang/codegen_test.go` (~20 LOC)

**Acceptance Criteria:**
- [ ] `convertToDrawCmdSlice` becomes `ConvertToDrawCmdSlice`
- [ ] Generated code compiles correctly
- [ ] Converters are accessible from external packages

**Risks:**
- None - simple naming change

### Milestone 4: M4-TYPE-ALIAS
**Goal:** Fix type alias expansion during unification so ADT variants with alias parameters work
**Estimated:** 80 LOC implementation + 50 LOC tests = 130 LOC
**Duration:** 2 hours

**Tasks:**
1. Add `aliasEnv map[string]Type` field to `Unifier` struct
2. Create `NewUnifierWithAliases()` constructor
3. Add `expandAlias()` method to expand TCon → underlying type
4. Call `expandAlias()` at start of `Unify()`
5. Wire alias environment from TypeChecker to Unifier
6. Add tests for alias in ADT variant parameters

**Files to Modify:**
- `internal/types/unification.go` (~50 LOC)
- `internal/types/typechecker.go` (~30 LOC)
- `internal/types/unification_test.go` (~50 LOC)

**Acceptance Criteria:**
- [ ] `type Coord = {x: int, y: int}` with `IsoTile(tile: Coord)` works
- [ ] Type alias in function parameters works
- [ ] Error messages still show alias names
- [ ] No regression in existing unification tests

**Risks:**
- Medium - need to thread alias environment through type checker
- Mitigation: Existing type environment patterns can be followed

### Milestone 5: M5-VERIFICATION
**Goal:** Verify all fixes work with stapledons_voyage codebase
**Estimated:** 0 LOC (verification only)
**Duration:** 1 hour

**Tasks:**
1. Run full test suite: `make test`
2. Run linting: `make lint`
3. Regenerate stapledons_voyage code (if accessible)
4. Verify bridge code compiles
5. Send confirmation to stapledons_voyage

**Acceptance Criteria:**
- [ ] All tests pass
- [ ] Linting clean
- [ ] stapledons_voyage code compiles (verified or confirmed)
- [ ] Response sent to stapledons_voyage

**Risks:**
- None - verification only

## Success Metrics
- Test coverage: Maintained (add ~190 LOC tests)
- Implementation: ~230 LOC
- Total: ~370 LOC
- All tests passing: ✅
- All linting passing: ✅
- stapledons_voyage unblocked: ✅

## Session Breakdown

### Session 1 (~3 hours): Codegen Fixes
- M1-BLANK-IDENTIFIER (1 hour)
- M2-RETURN-TYPES (1.5 hours)
- M3-EXPORT-CONVERTERS (0.5 hours)

### Session 2 (~3 hours): Type System Fix + Verification
- M4-TYPE-ALIAS (2 hours)
- M5-VERIFICATION (1 hour)

## Dependencies
- None - all required infrastructure exists

## Open Questions
- None - clear implementation path for all fixes

## Notes
- P0 priority: stapledons_voyage is blocked waiting for these fixes
- All fixes are targeted with clear scope
- Low risk due to existing test coverage
- Should be completable in single focused day
