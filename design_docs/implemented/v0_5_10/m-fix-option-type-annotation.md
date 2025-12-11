# Fix Type Application in Return Type Annotations (Option[T])

**Status**: ✅ Implemented
**Target**: v0.5.10
**Priority**: P1 (Medium-High) - Blocks std/string module
**Actual Time**: ~4 hours
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Enables natural Option[T] syntax |
| Preserve Semantic Clarity | + | +1 | Type annotations work as expected |
| Increase Determinism | + | +1 | Type checking is consistent |
| Lower Token Cost | 0 | 0 | Bug fix, no token change |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

Type applications (like `Option[int]`, `Result[T, E]`) in return type annotations fail with a type unification error.

**Error Message:**
```
Error: type error in std/string (decl 7): type unification failed at
[return type annotation at /Users/mark/dev/sunholo/ailang/std/string.ail:25:8]:
cannot unify type application with *types.TCon
```

## Root Cause Analysis

Two separate bugs were identified:

### Bug 1: Parser Discarding Type Arguments

The parser (`internal/parser/parser_type.go`) was parsing `Option[int]` as `SimpleType{Name: "Option"}`, completely discarding the type arguments `[int]`.

**Location:** `parser_type.go` lines 48-57

```go
// Before: Type args were lost
switch name {
case "Array":
    typ = &ast.ArrayType{...}
case "List":
    typ = &ast.ListType{...}
default:
    typ = &ast.SimpleType{Name: name} // Type args discarded!
}
```

### Bug 2: Factory Types Missing TApp

In `pipeline_module.go`, constructor factory types were created with `TCon("Option")` instead of `TApp(Option, [type vars])` for parameterized ADTs.

**Location:** `pipeline_module.go` line 308

```go
// Before: Missing type application
resultType := &types.TCon{Name: ctorInfo.TypeName}

// After: Correct TApp for parameterized ADTs
if ctorInfo.TypeParamCount > 0 {
    resultType = &types.TApp{
        Constructor: &types.TCon{Name: ctorInfo.TypeName},
        Args:        adtTypeVars,
    }
}
```

## Solution Implementation

### Phase 1: AST Changes

**Added `ast.TypeApp` struct** (`internal/ast/ast_type.go`):
```go
// TypeApp represents type application (generic types)
// Example: Option[int], Result[T, E], Map[string, int]
type TypeApp struct {
    Constructor string // Type constructor name (e.g., "Option", "Result")
    Args        []Type // Type arguments (e.g., [int], [T, E])
    Pos         Pos
}
```

### Phase 2: Parser Update

**Updated parser** (`internal/parser/parser_type.go`) to use TypeApp:
```go
default:
    // M-TAPP-FIX: Use TypeApp to preserve type arguments for generic types
    typ = &ast.TypeApp{Constructor: name, Args: typeArgs, Pos: startPos}
```

### Phase 3: Elaborator Update

**Updated elaborator** (`internal/elaborate/file.go`) to handle TypeApp:
```go
case *ast.TypeApp:
    args := make([]types.Type, len(typ.Args))
    for i, arg := range typ.Args {
        args[i] = e.astTypeToInternalType(arg)
    }
    return &types.TApp{
        Constructor: &types.TCon{Name: typ.Constructor},
        Args:        args,
    }
```

### Phase 4: Pipeline Updates

**Updated pipeline_module.go** to create correct TApp for factory types:
- Created type vars for ADT type params: `t0, t1, ...`
- Built result type as `TApp(TCon(TypeName), [type vars])` when TypeParamCount > 0
- Same fix applied to `runtime/runtime.go` for runtime interface building

### Phase 5: Pattern Matching Fix

**Updated typechecker_patterns.go** to use TApp for ADTs in pattern matching:
```go
if paramCount, hasParams := tc.adtTypeParams[adtTypeName]; hasParams && paramCount > 0 {
    typeArgs := make([]Type, paramCount)
    for i := 0; i < paramCount; i++ {
        typeArgs[i] = ctx.freshTypeVar()
    }
    adtType = &TApp{
        Constructor: &TCon{Name: adtTypeName},
        Args:        typeArgs,
    }
}
```

### Phase 6: Supporting Changes

- **AST Printer** (`internal/ast/print.go`): Added TypeApp case for golden file tests
- **Go Codegen** (`internal/gen/golang/adt.go`): Added TypeApp case in mapASTType
- **Unification** (`internal/types/unification_types.go`): Added better error message for TApp vs TCon

### Phase 7: Imported Constructor Fix (v0.5.10.1)

**Problem:** Imported constructors (e.g., `Some`, `None` from `std/option`) weren't getting their type parameter count propagated, causing `Option[a]` return types to fail in modules that import Option.

**Error:** `type Option expects 1 type argument(s), but got 0`

**Root Causes:**
1. **Interface builder** (`internal/iface/builder.go`): Created `TCon("Option")` instead of `TApp(Option, [a])` for result types
2. **iface.ConstructorInfo**: Missing `TypeParamCount` field
3. **Import handling** (`internal/pipeline/pipeline_module.go`): Didn't register imported constructors in `ctorTypes` and `adtTypeParams` maps

**Solution:**
- Added `TypeParamCount` to `iface.ConstructorInfo`
- Updated interface builder to create TApp for parameterized ADTs
- Added imported constructor tracking maps: `importedCtorTypes`, `importedADTTypeParams`
- Merge imported constructors with local ones before setting on type checker

## Files Modified

| File | Change |
|------|--------|
| `internal/ast/ast_type.go` | Added TypeApp struct |
| `internal/parser/parser_type.go` | Use TypeApp for generic types |
| `internal/elaborate/file.go` | Handle TypeApp in astTypeToInternalType |
| `internal/elaborate/core.go` | Added TypeParamCount to ConstructorInfo |
| `internal/types/typechecker_core.go` | Added adtTypeParams map |
| `internal/types/typechecker_patterns.go` | Use TApp for parameterized ADTs |
| `internal/types/unification_core.go` | Added TCon vs TApp handling |
| `internal/types/unification_types.go` | Better TApp vs TCon error message |
| `internal/pipeline/compile_unit.go` | Added TypeParamCount field |
| `internal/pipeline/pipeline_module.go` | Create TApp for factory result types; track imported constructors |
| `internal/pipeline/pipeline_converters.go` | Propagate TypeParamCount in conversion |
| `internal/iface/builder.go` | Added TypeParamCount; create TApp for ADT result types |
| `internal/runtime/runtime.go` | Create TApp for constructor schemes |
| `internal/ast/print.go` | Added TypeApp case for printer |
| `internal/gen/golang/adt.go` | Added TypeApp case in mapASTType |
| `internal/parser/type_test.go` | Updated tests for TypeApp |
| `testdata/parser/type/*.golden` | Updated golden files |

## Success Criteria ✅

- [x] `std/string.ail` imports and type checks correctly
- [x] `stringToInt("42")` returns `Option[int]`
- [x] Functions can declare `Option[T]` return types
- [x] Functions can declare `Result[T, E]` return types
- [x] All tests passing
- [x] No regression in existing type inference

## Testing

**Manual test (passes):**
```ailang
module test/tapp

type Option[a] = Some(a) | None

pure func map[a, b](f: (a) -> b, opt: Option[a]) -> Option[b] {
  match opt {
    Some(x) => Some(f(x)),
    None => None
  }
}

pure func main() -> int {
  let double = \x. x * 2 in
  let opt: Option[int] = Some(21) in
  let result = map(double, opt) in
  getOrElse(result, 0)
}
```

**Result:** ✓ No errors found!

---

**Document created**: 2025-12-10
**Implemented**: 2025-12-11
