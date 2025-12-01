# M-BUG-MODULE-LET-SCOPE: Fix Module-Level Let Bindings Not Accessible in Functions

**Status**: IMPLEMENTED
**Target**: v0.5.0
**Priority**: P1 (Medium) - Affects module organization patterns
**Estimated**: 4-6 hours
**Actual**: ~2 hours
**Dependencies**: None
**Reported by**: stapledons_voyage (agent inbox, 2025-11-30)
**Implemented**: 2025-11-30

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Enables shared constants without duplication |
| Preserve Semantic Clarity | + | +1 | Standard lexical scoping behavior |
| Increase Determinism | 0 | 0 | No change to determinism |
| Lower Token Cost | + | +1 | Avoids repeating constants in every function |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

Module-level `let` bindings are not accessible inside `pure func` / `func` bodies in the same module. This violates standard lexical scoping rules found in all functional languages (Haskell, OCaml, F#, etc.).

**Current State:**
```ailang
module game/world

let tileSize: int = 8
let gridWidth: int = 64

pure func getTileSize() -> int {
    tileSize  -- ERROR: undefined variable: tileSize
}
```

**Root Cause** (verified in `internal/elaborate/file.go:140-225`):
1. Functions are elaborated first (lines 140-211) via `collectFuncSigs` and `funcToLambda`
2. Module-level let bindings are processed AFTER as "non-func statements" (lines 213-222)
3. Let bindings are **never added to the symbol scope** used when elaborating function bodies

**Impact:**
- Breaks standard functional programming patterns
- Forces code duplication (inline constants in every function)
- Confusing for users coming from any other functional language
- Reported by stapledons_voyage game project

## Goals

**Primary Goal:** Module-level let bindings should be in scope for all function bodies in the same module.

**Success Metrics:**
- `let x = 8; pure func f() -> int { x }` compiles and runs correctly
- Existing examples/tests continue to work (regression-free)
- Clear error messages if binding is used before definition (forward reference)

## Solution Design

### Overview

Refactor `ElaborateFile` to collect and process module-level let bindings BEFORE elaborating function bodies, adding them to the symbol environment.

### Architecture

**Current flow:**
```
1. Process type declarations
2. collectFuncSigs() → symbols map
3. Elaborate functions (symbols only contains funcs)
4. Process non-func statements (lets processed here, too late)
```

**New flow:**
```
1. Process type declarations
2. collectModuleLets() → collect let bindings
3. collectFuncSigs() → symbols map (includes let-bound names)
4. Build environment with let bindings
5. Elaborate functions WITH let bindings in scope
6. Process remaining statements
```

**Key insight:** Module-level let bindings are expressions that bind names. We need to:
1. Identify them early (before function elaboration)
2. Add their names to the symbol table
3. Wrap function elaboration in a scope that includes them

### Implementation Plan

**Phase 1: Collect Module-Level Lets** (~2 hours)
- [ ] Add `collectModuleLets(file *ast.File) []*ast.Let` function
- [ ] Identify let expressions at module level (in `file.Statements`)
- [ ] Extract name and type annotation for each
- [ ] Add to symbols map alongside function signatures

**Phase 2: Scope Threading** (~2 hours)
- [ ] Modify `ElaborateFile` to process lets before funcs
- [ ] Create Core let bindings that wrap all function definitions
- [ ] Ensure let-bound names are available in `funcToLambda` scope
- [ ] Handle mutual dependencies (let can reference func, func can reference let)

**Phase 3: Testing & Documentation** (~2 hours)
- [ ] Add test case: basic module-level let access
- [ ] Add test case: multiple lets with dependencies
- [ ] Add test case: let referencing imported symbol
- [ ] Add test case: error on forward reference (if applicable)
- [ ] Create `examples/module_constants.ail`
- [ ] Update CHANGELOG.md
- [ ] Respond to stapledons_voyage with fix notification

### Files to Modify/Create

**Modified files:**
- `internal/elaborate/file.go` - Add `collectModuleLets`, modify `ElaborateFile` (~60 LOC net change)

**New files:**
- `examples/module_constants.ail` - Example showing module-level constants (~20 LOC)
- `internal/elaborate/file_test.go` - Additional test cases (~40 LOC)

**Total estimated LOC:** ~120 LOC

## Examples

### Example 1: Basic Module Constants

**Before (FAILS):**
```ailang
module game/config

let tileSize: int = 8
let gridWidth: int = 64
let gridHeight: int = 48

pure func getTotalTiles() -> int {
    gridWidth * gridHeight  -- ERROR: undefined variable
}
```

**After (WORKS):**
```ailang
module game/config

let tileSize: int = 8
let gridWidth: int = 64
let gridHeight: int = 48

pure func getTotalTiles() -> int {
    gridWidth * gridHeight  -- Returns: 3072
}

pure func pixelWidth() -> int {
    gridWidth * tileSize  -- Returns: 512
}
```

### Example 2: Let Referencing Function

```ailang
module math/utils

pure func square(x: int) -> int { x * x }

let defaultRadius: int = 10
let defaultArea: int = square(defaultRadius)  -- Should work: 100

pure func getArea() -> int {
    defaultArea  -- Should work: 100
}
```

### Example 3: Current Workaround (For Reference)

```ailang
-- WORKAROUND: Use lambdas instead of pure func
let tileSize: int = 8
let getTileSize = \u. tileSize  -- Works because lambda captures scope

-- WORKAROUND: Inline constants
pure func pixelWidth(gridWidth: int) -> int {
    gridWidth * 8  -- Hardcoded instead of using tileSize
}
```

## Success Criteria

- [ ] `let x = 8; pure func f() -> int { x }` compiles and runs
- [ ] Module-level lets can reference each other in order
- [ ] Module-level lets can reference imported symbols
- [ ] Functions can reference module-level lets
- [ ] Existing test suite passes (no regressions)
- [ ] `examples/module_constants.ail` added and verified
- [ ] stapledons_voyage use case works

## Testing Strategy

**Unit tests** (`internal/elaborate/file_test.go`):
```go
func TestModuleLevelLetInFunctionScope(t *testing.T) {
    src := `
    module test/scope
    let x: int = 42
    pure func getX() -> int { x }
    `
    // Verify elaboration succeeds
    // Verify function body has x in scope
}
```

**Integration tests:**
- Run `examples/module_constants.ail` through full pipeline
- Verify REPL handles module-level lets correctly

**Manual testing:**
- Test stapledons_voyage actual use case
- Verify error messages for undefined variables are still clear

## Non-Goals

**Not in this fix:**
- Forward references (let x = y; let y = 5) - may require separate design
- Mutable module-level state - AILANG is immutable by design
- Lazy evaluation of let bindings - all lets are strict
- Module-level let exports - handled separately by export system

## Timeline

**Day 1** (~4 hours):
- Phase 1: Collect module-level lets
- Phase 2: Scope threading implementation
- Basic testing

**Day 2** (~2 hours):
- Phase 3: Full test coverage
- Documentation and examples
- Respond to stapledons_voyage

**Total: ~6 hours across 1-2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Circular dependencies between lets and funcs | Medium | Detect cycles, provide clear error message |
| Order-dependent scoping surprises | Low | Document that lets are processed in order |
| Breaking existing elaboration | High | Add regression tests first |
| Performance impact on large modules | Low | Collect phase is O(n), minimal overhead |

## References

- stapledons_voyage bug report (agent inbox, 2025-11-30)
- `internal/elaborate/file.go:140-225` - Current elaboration flow
- `internal/elaborate/file.go:106-111` - `collectFuncSigs` pattern to follow
- Haskell module system - Reference for standard behavior
- OCaml `let` bindings - Reference for lexical scoping

## Future Work

- Module-level `let rec` for recursive value bindings
- Export syntax for module-level lets (`export let x = 5`)
- Lazy module-level lets (if needed for performance)
- Better error messages for scope-related issues

---

## Implementation Notes (2025-11-30)

### Changes Made

**1. `internal/elaborate/file.go`:**
- Added `ModuleLet` struct to represent module-level let bindings
- Added `collectModuleLets(file *ast.File) []*ModuleLet` to extract module-level lets
- Added `wrapInLets(expr CoreExpr, lets [...]) CoreExpr` helper to wrap expressions in let bindings
- Modified `ElaborateFile` to:
  1. Collect module-level lets first
  2. Elaborate function declarations as before
  3. Wrap each function declaration in the module-level lets (creates nested Let structure)

**2. `internal/iface/builder.go`:**
- Added `extractExportsFromExpr(expr, meta, exports)` method for recursive export extraction
- Modified `extractExports` to use recursive extraction
- This handles the new nested Let structure where functions are wrapped inside module-level lets

**3. Files Modified:**
- `internal/elaborate/file.go` - ~50 LOC added
- `internal/iface/builder.go` - ~30 LOC added

### Core AST Structure

**Before fix:**
```
Program.Decls = [
  LetRec(main): ...
]
```

**After fix:**
```
Program.Decls = [
  Let(x):
    Value: 42
    Body: Let(main):  // Or LetRec for recursive functions
            Value: Lambda(...)
            Body: ...
]
```

### Test Results
- All existing tests pass
- Manual testing confirms module-level lets accessible in function bodies
- stapledons_voyage use case works

---

**Document created**: 2025-11-30
**Last updated**: 2025-11-30
**Implemented**: 2025-11-30
