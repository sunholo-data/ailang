# M-CODEGEN-TYPED-PARAMS: Fix Typed Parameters Becoming interface{}

**Status**: IMPLEMENTED
**Target**: v0.5.8
**Priority**: P0 - High (breaks type safety)
**Estimated**: 2-3 hours
**Actual**: ~2 hours
**Dependencies**: None
**Reported by**: stapledons_voyage project

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix, no syntax change |
| Preserve Semantic Clarity | + | +1 | Go types match AILANG annotations |
| Increase Determinism | + | +1 | Predictable typed output |
| Lower Token Cost | 0 | 0 | No change to source tokens |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Goals

**Primary Goal:** Generated Go functions preserve AILANG type annotations

**Success Metrics:**
- `stepArrival(state: ArrivalState, input: ArrivalInput) -> ArrivalState` generates typed Go params
- Type assertions removed from calling code
- stapledons_voyage compiles without manual type fixes

## Problem Statement

Functions with typed parameters generate `interface{}` instead of proper types.

**AILANG:**
```ailang
export pure func stepArrival(state: ArrivalState, input: ArrivalInput) -> ArrivalState
```

**Generated Go (current - broken):**
```go
func StepArrival(state interface{}, input interface{}) interface{}
```

**Expected Go:**
```go
func StepArrival(state *ArrivalState, input *ArrivalInput) *ArrivalState
```

**Impact:**
- Loss of type safety in generated Go code
- Requires manual type assertions at every call site
- Defeats the purpose of AILANG's type system

## Root Cause Analysis

### ROOT CAUSE: Type Annotations Discarded in Elaboration

**Location:** `internal/elaborate/expressions.go:141-159`

In `normalizeLambda`, type information is completely discarded:

```go
func (e *Elaborator) normalizeLambda(lam *ast.Lambda) (core.CoreExpr, error) {
    params := make([]string, len(lam.Params))
    for i, p := range lam.Params {
        params[i] = p.Name  // <-- ONLY NAME! TYPE IS DISCARDED!
    }
    // ...
    coreLam := &core.Lambda{
        Params: params,  // Just strings, no types
        Body:   body,
    }
}
```

The `ast.Param` has both `Name` and `Type`, but only `Name` is preserved in Core Lambda.

### Chain of Failure

1. **Elaboration** (`normalizeLambda`): Discards `state: ArrivalState` type annotation
2. **Type Inference** (`inferLambda`): Creates fresh `TVar` for `state` (no annotation to constrain it)
3. **Unification**: Without constraints, TVar stays unresolved
4. **Codegen**: Maps unresolved TVar → `interface{}`

### Supporting Evidence

**Type inference always creates fresh vars** (`internal/types/typechecker_functions.go:16-20`):
```go
for i, param := range lam.Params {
    paramType := ctx.freshTypeVar()  // Always fresh, no annotation lookup!
    paramTypes[i] = paramType
}
```

**Unresolved vars become interface{}** (`internal/gen/golang/types.go:119-123`):
```go
case *types.TVar:
    return GoType("interface{}"), nil
```

## Proposed Solution

### Option A: Store Type Annotations in Elaborator Side Table (Recommended)

Store param type annotations in a map during elaboration, lookup during type checking:

```go
// In Elaborator
type Elaborator struct {
    // ...
    paramTypeAnnotations map[uint64][]types.Type  // LambdaNodeID → param types
}

// In normalizeLambda
func (e *Elaborator) normalizeLambda(lam *ast.Lambda) (core.CoreExpr, error) {
    params := make([]string, len(lam.Params))
    paramTypes := make([]types.Type, len(lam.Params))
    for i, p := range lam.Params {
        params[i] = p.Name
        if p.Type != nil {
            paramTypes[i] = e.astTypeToInternalType(p.Type)
        }
    }
    coreLam := &core.Lambda{...}
    e.paramTypeAnnotations[coreLam.NodeID] = paramTypes  // Store!
    return coreLam, nil
}

// Pass to type checker, use in inferLambda to constrain fresh vars
```

**Pros:**
- No Core AST changes needed
- Type checker can use annotations to constrain vars
- Backward compatible

**Cons:**
- Another side table to pass through pipeline

### Option B: Add ParamTypes to Core Lambda

Extend Core Lambda to include type annotations:

```go
type Lambda struct {
    CoreNode
    Params     []string
    ParamTypes []types.Type  // NEW: Optional type annotations
    Body       CoreExpr
}
```

**Pros:**
- Self-contained - no side tables
- Clear ownership

**Cons:**
- Core AST change affects many files
- May break serialization/cloning

### Option C: Register Function Types in Compile Pipeline

Extract types from AST in compile.go, register with codegen:

```go
// In compile.go after processing FuncDecl
for _, fn := range file.Funcs {
    paramTypes := extractParamTypes(fn.Params)
    returnType := extractReturnType(fn.ReturnType)
    codeGen.RegisterFunctionType(fn.Name, paramTypes, returnType)
}
```

**Pros:**
- Minimal change to existing code
- Uses existing AST info

**Cons:**
- Bypasses type inference entirely
- Won't catch type errors
- Duplicates type extraction logic

## Recommended Solution: Option C (Pragmatic)

Option C is the fastest fix for stapledons_voyage while we design the proper solution.

**Rationale:**
- Option A (side table) requires pipeline changes across 4+ files
- Option B (Core AST) is invasive and risky
- Option C is localized to compile.go, gets stapledons_voyage unblocked NOW

**Proper fix (Option A) can follow in v0.5.10.**

## Implementation Plan

### Phase 1: Quick Fix via Compile Pipeline (~1 hour)

#### M1: Add RegisterFunctionType to Generator

**File:** `internal/gen/golang/codegen.go`

```go
// funcTypeOverrides stores explicit type signatures from AST
// M-TYPED-PARAMS: Used when CoreTypeInfo doesn't have resolved types
type Generator struct {
    // ... existing fields ...
    funcTypeOverrides map[string]*FuncTypeOverride
}

type FuncTypeOverride struct {
    ParamTypes []GoType
    ReturnType GoType
}

func (g *Generator) RegisterFunctionType(name string, paramTypes []GoType, returnType GoType) {
    g.funcTypeOverrides[name] = &FuncTypeOverride{
        ParamTypes: paramTypes,
        ReturnType: returnType,
    }
}
```

#### M2: Use Override in getTypedSignature

**File:** `internal/gen/golang/codegen_decl.go`

```go
func (g *Generator) getTypedSignature(lam *core.Lambda) ([]GoType, GoType) {
    // M-TYPED-PARAMS: Check for explicit override first
    // This handles cases where CoreTypeInfo has unresolved type vars
    if override, ok := g.funcTypeOverrides[g.currentFuncName]; ok {
        return override.ParamTypes, override.ReturnType
    }

    // ... existing CoreTypeInfo lookup ...
}
```

#### M3: Register Types in compile.go

**File:** `cmd/ailang/compile.go`

```go
// After creating codeGen, before generating functions
for _, file := range filenames {
    if astFile := result.Artifacts.AST; astFile != nil {
        for _, fn := range astFile.Funcs {
            if fn.IsExtern {
                continue
            }
            paramTypes := make([]gen.GoType, 0)
            for _, p := range fn.Params {
                if p.Name == "_" && p.Type != nil && p.Type.String() == "()" {
                    continue // Skip unit params
                }
                paramTypes = append(paramTypes, gen.GoType(ailangTypeToGoType(p.Type)))
            }
            var returnType gen.GoType = "interface{}"
            if fn.ReturnType != nil {
                returnType = gen.GoType(ailangTypeToGoType(fn.ReturnType))
            }
            codeGen.RegisterFunctionType(fn.Name, paramTypes, returnType)
        }
    }
}
```

### Phase 2: Testing (~30 min)

#### M4: Add Test for Typed Params

**File:** `internal/gen/golang/codegen_test.go`

```go
func TestTypedParamsOverride(t *testing.T) {
    // Test that RegisterFunctionType overrides interface{} fallback
}
```

#### M5: Verify with stapledons_voyage

- Recompile stapledons_voyage
- Check StepArrival has typed params
- Verify Go code compiles

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/gen/golang/codegen.go` | Add funcTypeOverrides map, RegisterFunctionType | ~20 |
| `internal/gen/golang/codegen_decl.go` | Check override in getTypedSignature | ~10 |
| `cmd/ailang/compile.go` | Register function types from AST | ~30 |
| `internal/gen/golang/codegen_test.go` | Add test | ~30 |

**Estimated Total: ~90 LOC**

## Acceptance Criteria

- [x] `stepArrival(state: ArrivalState, input: ArrivalInput) -> ArrivalState` generates typed params
- [x] All exported functions get proper typed signatures
- [x] Fallback to interface{} only for truly polymorphic functions
- [x] stapledons_voyage generates type-safe Go code
- [x] Existing tests still pass

## Implementation Notes

**Solution implemented: Option C with extensions for cross-module type preservation**

Files modified:
- `internal/types/types.go` - Added `TypeName` field to TRecord for nominal type identity
- `internal/types/unification.go` - Modified `expandAlias` to set TypeName when expanding TCon to TRecord
- `internal/gen/golang/types.go` - Check TypeName first in MapType
- `internal/gen/golang/codegen.go` - Added `FuncTypeOverride`, `RegisterFunctionType`, `currentFuncDeclaredReturn`
- `internal/gen/golang/codegen_decl.go` - Set currentFuncDeclaredReturn, updated getTypedSignature
- `internal/gen/golang/codegen_ops.go` - Updated generateRecord to use declared return type
- `cmd/ailang/compile.go` - Register function types from AST

Total: ~100 LOC across 7 files

## Related Issues

- M-CODEGEN-ZERO-ARG (FIXED): Zero-arg functions get wrong params
- M-CODEGEN-TYPE-ASSERTIONS (FIXED): _impl functions calling typed exports
- M-CROSS-MODULE (FIXED): Cross-module type contamination

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09
