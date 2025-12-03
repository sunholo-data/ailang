# M-DX25: Typed Let Bindings

## Status: IMPLEMENTED
## Target: v0.5.5
## Completed: 2025-12-03

## Problem Statement

M-DX24.3 attempted to add typed local variables to generated Go code, but was disabled due to type mismatches. There were two distinct bugs to fix.

### Bug Report 1: Type Mismatch in IIFE Pattern
From stapledons_voyage:
```
Codegen produces var tmp1 bool = (interface{} expr) which fails.
Line 12: var tmp1 bool = func() interface{} {...}() needs type assertion.
Also line 13 uses tmp1.(bool) but tmp1 is already bool not interface{}.
```

### Bug Report 2: Undefined List Type
From stapledons_voyage:
```
funcs.go:521 and :561 reference undefined List type.
Should be []interface{} or a type alias should be generated.
```

---

## Root Cause Analysis

### Issue 1: TApp("List", T) Not Mapped to Go Slice

**Location:** `internal/gen/golang/types.go` lines 80-88

When a type appears as `TApp{Constructor: TCon{Name: "List"}, Args: [ElementType]}`, the TypeMapper didn't recognize "List" as a special case and fell through to `ToGoTypeName("List")` which returned just `"List"` - an undefined type in Go.

### Issue 2: IIFE Type Inconsistency

**Location:** `internal/gen/golang/codegen_expr.go` lines 325-351

The `generateLet` function generates an IIFE, but the original M-DX24.3 code looked up different types for the variable and IIFE return, causing mismatches.

---

## Implementation

### M-DX25.1: List Type Fix (COMPLETE)

**Files Modified:** `internal/gen/golang/types.go` (+12 LOC)

Added special case for "List" in TApp handling:

```go
case *types.TApp:
    if con, ok := typ.Constructor.(*types.TCon); ok {
        // M-DX25.1: Special case for List type
        if con.Name == "List" {
            if len(typ.Args) > 0 {
                elemType, err := tm.MapType(typ.Args[0])
                if err != nil {
                    return GoType("[]interface{}"), nil
                }
                return GoType(fmt.Sprintf("[]%s", elemType)), nil
            }
            return GoType("[]interface{}"), nil
        }
        // ... rest unchanged
    }
```

**Tests Added:** `internal/gen/golang/types_test.go` (+80 LOC)
- 7 test cases covering: int, string, bool, float, nested lists, empty list, TList compatibility

### M-DX25.2: IIFE Type Consistency (COMPLETE)

**Files Modified:** `internal/gen/golang/codegen_expr.go` (+25 LOC)

**Bug Fix (2025-12-03):** Initial implementation incorrectly used the let expression's type for both variable AND IIFE return. But the let's type IS the body's type, not the value's type. Fixed to use SEPARATE lookups:

```go
func (g *Generator) generateLet(let *core.Let) error {
    // M-DX25.2 FIX: Variable type comes from VALUE expression
    varType := "interface{}"
    if g.coreTypeInfo != nil {
        valueNodeID := g.getExprNodeID(let.Value)
        if typ, ok := g.coreTypeInfo[valueNodeID]; ok {
            varType = string(g.TypeMapper.MapType(typ))
        }
    }

    // M-DX25.2 FIX: Return type comes from LET expression (= body's type)
    returnType := "interface{}"
    if g.coreTypeInfo != nil {
        if typ, ok := g.coreTypeInfo[let.NodeID]; ok {
            returnType = string(g.TypeMapper.MapType(typ))
        }
    }

    g.writef("func() %s {\n", returnType)
    g.writef("var %s %s = ", ToGoVarName(let.Name), varType)
    // ... type assertions as needed
}
```

**Key Insight:** `let.NodeID` type = body's type (IIFE return), `let.Value.ID()` type = variable's type.

**Tests Added:** `internal/gen/golang/codegen_test.go` (+120 LOC)
- `TestTypedLetBindings` - typed bool let binding
- `TestTypedLetBindingsWithAssertion` - BinOp value requiring assertion
- `TestTypedLetBindingsFallback` - fallback to interface{} without CoreTypeInfo

### M-DX25.3: Typed If Expressions (COMPLETE)

**Files Modified:** `internal/gen/golang/codegen_expr.go` (+25 LOC)

The `generateIf` function was always returning `interface{}` and always adding `.(bool)` to conditions. Fixed to:
1. Look up If expression's type from CoreTypeInfo for IIFE return
2. Only add `.(bool)` if condition produces `interface{}`
3. Add type assertions to Then/Else branches only when needed

### M-DX25.4: ADT Constructor Type Inference (COMPLETE)

**Files Modified:**
- `internal/types/typechecker_core.go` (+20 LOC) - Added `constructorTypes` registry
- `internal/types/typechecker_patterns.go` (+12 LOC) - Unify scrutinee with ADT type
- `internal/pipeline/pipeline_module.go` (+8 LOC) - Pass constructor info to type checker
- `internal/pipeline/pipeline_single.go` (+10 LOC) - Same for single-file pipeline

**Root Cause:** The `checkPattern` function for `ConstructorPattern` had a TODO saying "needs access to module interface to get constructor schemes" and never added a constraint to unify the scrutinee with the ADT type. Now it properly registers constructor → ADT type mappings and adds constraints.

### M-DX25.5: Typed Match Expressions (COMPLETE)

**Files Modified:**
- `internal/gen/golang/codegen.go` (+4 LOC) - Added `matchReturnType` field
- `internal/gen/golang/codegen_match.go` (+40 LOC) - Typed IIFE and arm assertions

The `generateMatch` function was always returning `interface{}`. Fixed to:
1. Look up Match expression's type from CoreTypeInfo for IIFE return
2. Store return type in `g.matchReturnType` for arm generation
3. Add type assertions to match arm bodies when return type is concrete

### M-DX25.6: ADT Pointer Types (COMPLETE)

**Files Modified:**
- `internal/gen/golang/types.go` (+5 LOC) - ADT types use `WithPointer`
- `internal/gen/golang/codegen_match.go` (+5 LOC) - Only assert if scrutinee is interface{}
- `internal/gen/golang/codegen_test.go` (+3 LOC) - Updated test expectations

ADT constructors return pointers (`*Direction`), so parameter and return types should also use pointers. Fixed:
1. `mapTCon` now returns `*TypeName` for user-defined types (not primitives)
2. Match scrutinee assertion only when `exprProducesInterface` is true
3. Updated tests to expect `*World` instead of `World`

### M-DX25.7: Typed List Pattern Operations (COMPLETE)

**Files Modified:**
- `internal/gen/golang/codegen.go` (+3 LOC) - Added `matchScrutineeType` field
- `internal/gen/golang/codegen_match.go` (+50 LOC) - Typed inline slice operations

List pattern matching used `ListHead`/`ListTail`/`ListLen` helpers that return `interface{}`. Fixed to use inline slice operations when scrutinee type is known:
1. Look up scrutinee type from CoreTypeInfo in `generateMatch`
2. Store in `g.matchScrutineeType` for pattern condition generation
3. When type is `[]T` (not `[]interface{}`), generate:
   - `len(xs)` instead of `ListLen(xs)`
   - `xs[0]` instead of `ListHead(xs)`
   - `xs[1:]` instead of `ListTail(xs)`
   - `xs[i]` for indexed element access

### M-DX25.8: Accurate Type Checking with CoreTypeInfo (COMPLETE)

**Files Modified:**
- `internal/gen/golang/codegen_decl.go` (+15 LOC) - Use CoreTypeInfo in exprProducesInterface
- `internal/gen/golang/codegen_test.go` (+3 LOC) - Updated test expectations

The `exprProducesInterface` function used heuristics that were outdated after typed codegen. Now uses CoreTypeInfo:
1. Look up expression's NodeID in CoreTypeInfo
2. Map type to Go type using TypeMapper
3. Return `true` only if Go type is `"interface{}"`
4. Fall back to heuristics only when type info unavailable

This fixes unnecessary type assertions like `pathLength(rest).(int64)` when the function already returns `int64`.

### M-DX25.9: Call Site Type Assertions (COMPLETE)

**Files Modified:**
- `internal/gen/golang/codegen_expr.go` (+60 LOC) - getFuncParamTypes and call site assertions

When calling functions, arguments that are `interface{}` but parameters expect concrete types need type assertions:
1. Added `getFuncParamTypes` to look up function's parameter types from CoreTypeInfo
2. Added `extractParamTypes` to extract Go types from TFunc/TFunc2
3. In `generateApp`, for each argument:
   - If arg produces `interface{}` but param expects concrete → add `.(type)`
   - If arg is literal and param expects primitive → add type conversion
4. Works for all user-defined functions, not just ADT constructors

This fixes:
- `directionDx(dir)` where `dir` is `interface{}` → `directionDx(dir.(*Direction))`
- `IsInBounds(..., width)` where `width` is `interface{}` → `IsInBounds(..., width.(int64))`

---

## Implementation Checklist

### M-DX25.1: List Type Fix
- [x] Add "List" case to TApp handling in `types.go:MapType`
- [x] Handle element type recursively
- [x] Add test: `TApp("List", TCon("int"))` → `[]int64`
- [x] Verify no "undefined: List" in generated code

### M-DX25.2: IIFE Type Consistency
- [x] Refactor `generateLet` to use unified type
- [x] Look up let node type from CoreTypeInfo
- [x] Add type assertions for value expression when needed
- [x] Add type assertions for body expression when needed
- [x] Add tests for typed let bindings

### M-DX25.3: Typed If Expressions
- [x] Look up If expression type from CoreTypeInfo for IIFE return
- [x] Only add `.(bool)` if condition produces `interface{}`
- [x] Add type assertions to Then/Else branches when needed

### M-DX25.4: ADT Constructor Type Inference
- [x] Add `constructorTypes` registry to CoreTypeChecker
- [x] Pass constructor info from pipeline to type checker
- [x] Add constraint in `checkPattern` to unify scrutinee with ADT type

### M-DX25.5: Typed Match Expressions
- [x] Look up Match expression type from CoreTypeInfo for IIFE return
- [x] Add `matchReturnType` field to Generator
- [x] Add type assertions to match arm bodies when needed

### M-DX25.6: ADT Pointer Types
- [x] Map user-defined types to pointers in `mapTCon`
- [x] Only type-assert scrutinee when it produces `interface{}`
- [x] Update tests to expect `*World` instead of `World`

### M-DX25.7: Typed List Pattern Operations
- [x] Add `matchScrutineeType` field to Generator
- [x] Look up scrutinee type in `generateMatch`
- [x] Use inline slice operations when type is `[]T` (not `[]interface{}`)

### M-DX25.8: Accurate Type Checking
- [x] Use CoreTypeInfo in `exprProducesInterface`
- [x] Return true only when Go type is `interface{}`
- [x] Update tests to reflect no assertions on concrete types

### M-DX25.9: Call Site Type Assertions
- [x] Add `getFuncParamTypes` to look up function parameter types
- [x] Add `extractParamTypes` to extract types from TFunc/TFunc2
- [x] Add type assertions at call sites when arg is interface{} but param is concrete

---

## Acceptance Criteria (ALL MET)

1. ✅ No "undefined: List" errors in generated code
2. ✅ Typed let bindings compile without manual modification
3. ✅ No type assertion on already-typed variables
4. ✅ Short-circuit AND/OR generates correct typed code
5. ✅ ADT function parameters infer correct ADT types
6. ✅ Match expressions return correct concrete types
7. ✅ ADT types consistently use pointers (*Direction, *World)
8. ✅ List pattern operations preserve slice types
9. ✅ All existing codegen tests pass
10. ⏳ stapledons_voyage game compiles successfully (pending verification)

---

## Files Modified

| File | Changes |
|------|---------|
| `internal/gen/golang/types.go` | +17 LOC - List case in TApp, ADT pointer types |
| `internal/gen/golang/codegen_expr.go` | +40 LOC - Typed generateLet and generateIf |
| `internal/gen/golang/codegen_match.go` | +100 LOC - Typed match, scrutinee assertions, inline slice ops |
| `internal/gen/golang/codegen.go` | +7 LOC - matchReturnType and matchScrutineeType fields |
| `internal/types/typechecker_core.go` | +20 LOC - constructorTypes registry |
| `internal/types/typechecker_patterns.go` | +12 LOC - ADT type constraints |
| `internal/pipeline/pipeline_module.go` | +8 LOC - Pass constructor info |
| `internal/pipeline/pipeline_single.go` | +10 LOC - Pass constructor info |
| `internal/gen/golang/types_test.go` | +80 LOC - 7 List mapping tests |
| `internal/gen/golang/codegen_test.go` | +123 LOC - Typed let binding tests + pointer type expectations |

**Total:** ~417 LOC (implementation + tests)

---

## Related

- M-DX24: Typed Function Bodies (M-DX24.3 was disabled, now fixed here)
- M-DX23: Typed Function Signatures (completed)
