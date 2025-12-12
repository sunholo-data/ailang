# M-CODEGEN-ADT-TYPE-ASSERT: Fix Invalid Type Assertions on ADT Constructor Results

**Status**: Implemented
**Target**: v0.5.10
**Priority**: P0 (High) - Blocked stapledons_voyage compilation
**Actual Time**: ~20 minutes
**Dependencies**: None
**Created**: 2025-12-12
**Implemented**: 2025-12-12

## Problem Statement

When passing an ADT constructor result as an argument to another ADT constructor, the codegen was incorrectly adding type assertions:

```go
// BROKEN - invalid type assertion on non-interface value
var tmp13 interface{} = NewStarTypeMainSequence(NewSpectralClassG().(*SpectralClass))
// Error: invalid type assertion: NewSpectralClassG() (value of type *SpectralClass) is not an interface
```

**Root Cause:**
- In `_impl` functions, most expressions produce `interface{}` and need type assertions
- BUT: ADT constructor calls (like `NewSpectralClassG()`) return typed values (`*SpectralClass`), not `interface{}`
- The `exprProducesInterface()` function only checked for `*core.App` ADT constructors
- Nullary ADT constructors (with no fields) are represented as `*core.VarGlobal`, not `*core.App`

## Solution Implemented

### Changes to `internal/gen/golang/codegen_decl.go`

**1. Added VarGlobal check in `exprProducesInterface`** (lines 262-268):

```go
// M-CODEGEN-ADT-TYPE-ASSERT: Nullary ADT constructors are VarGlobal, not App.
// Example: `G` in `type SpectralClass = O | B | A | F | G | K | M` becomes NewSpectralClassG()
if vg, isVG := expr.(*core.VarGlobal); isVG {
    if g.isNullaryADTConstructor(vg) {
        return false // Nullary ADT constructors return *ADT, not interface{}
    }
}
```

**2. Added `isNullaryADTConstructor` helper** (lines 411-424):

```go
// isNullaryADTConstructor checks if a VarGlobal is a nullary ADT constructor.
// M-CODEGEN-ADT-TYPE-ASSERT: Nullary constructors (with no fields) are VarGlobal, not App.
// Example: `G` in `type SpectralClass = O | B | A | F | G | K | M` is a nullary constructor.
func (g *Generator) isNullaryADTConstructor(vg *core.VarGlobal) bool {
    // Check for $adt.make_TypeName_CtorName pattern
    if vg.Ref.Module == "$adt" && strings.HasPrefix(vg.Ref.Name, "make_") {
        return true
    }
    // Check if it's in the adtConstructors map (nullary constructors have 0 field types)
    if info, ok := g.adtConstructors[vg.Ref.Name]; ok {
        return len(info.FieldTypes) == 0
    }
    return false
}
```

### Changes to `internal/gen/golang/codegen_expr_app.go`

**Added check before adding type assertion** (lines 154-161):

```go
} else {
    // M-CODEGEN-ADT-TYPE-ASSERT: Only add type assertion if arg produces interface{}
    // ADT constructor calls (like NewSpectralClassG()) return typed values, not interface{}
    if err := g.generateExpr(arg); err != nil {
        return err
    }
    if g.exprProducesInterface(arg) {
        g.writef(".(%s)", goType)
    }
}
```

## Result

**Before (broken):**
```go
var tmp13 interface{} = NewStarTypeMainSequence(NewSpectralClassG().(*SpectralClass))
```

**After (fixed):**
```go
var tmp13 interface{} = NewStarTypeMainSequence(NewSpectralClassG())
```

## Files Modified

| File | Changes | LOC |
|------|---------|-----|
| `internal/gen/golang/codegen_decl.go` | VarGlobal check + helper | +20 |
| `internal/gen/golang/codegen_expr_app.go` | Check before type assertion | +5 |

**Total:** ~25 LOC

## Test Results

- [x] stapledons_voyage `sim/celestial.ail` compiles to valid Go
- [x] Generated code no longer has invalid type assertions
- [x] All codegen tests pass (`go test ./internal/gen/golang/...`)
- [x] Full test suite passes (`make test`)

## ADT Constructor Representation Summary

| Constructor Type | Core AST Node | Example | Go Output |
|------------------|---------------|---------|-----------|
| Nullary (no fields) | `*core.VarGlobal` | `G` | `NewSpectralClassG()` |
| With fields | `*core.App` | `MainSequence(G)` | `NewStarTypeMainSequence(...)` |

Both return typed values (`*ADT`), not `interface{}`, even in `_impl` functions.

## References

- GitHub #36 (stapledons_voyage bug report)
- Related: M-DX26 - dual function generation (`_impl` + typed wrapper)
- Also fixed in same session: M-CROSS-MODULE-RECORD-UNIFICATION
