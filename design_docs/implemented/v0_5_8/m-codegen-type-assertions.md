# M-CODEGEN-TYPE-ASSERTIONS: Fix _impl Functions Calling Typed Exports

**Status**: IMPLEMENTED
**Target**: v0.5.8
**Priority**: P0 - High (breaks Go compilation)
**Estimated**: 2-3 hours
**Actual**: ~30 minutes
**Dependencies**: M-CODEGEN-TYPED-PARAMS (completed)
**Reported by**: stapledons_voyage project

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix, no syntax change |
| Preserve Semantic Clarity | + | +1 | Generated Go compiles correctly |
| Increase Determinism | + | +1 | Predictable typed output |
| Lower Token Cost | 0 | 0 | No change to source tokens |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Goals

**Primary Goal:** Generated Go code compiles when `_impl` functions call typed exports

**Success Metrics:**
- `_impl` functions add type assertions when calling typed exports
- stapledons_voyage compiles without type mismatch errors
- Generated Go passes `go build` without manual fixes

## Problem Statement

After M-CODEGEN-TYPED-PARAMS, exported functions now have typed signatures. However, `_impl` functions still use `interface{}` variables and call exports without type assertions.

**Generated Go (current - broken):**
```go
// Exported function with typed signature (CORRECT)
func UpdateAllNPCs(npcs []*NPC, width int64, height int64) []*NPC {
    return updateAllNPCs_impl(npcs, width, height).([]*NPC)
}

// Internal _impl function (INCORRECT - type mismatch)
func someOther_impl(args ...interface{}) interface{} {
    // ...
    var tmp66 interface{} = /* some computation */
    var tmp68 interface{} = /* some computation */
    var tmp70 interface{} = /* some computation */

    // ❌ ERROR: cannot use tmp66 (type interface{}) as []*NPC
    return UpdateAllNPCs(tmp66, tmp68, tmp70)
}
```

**Expected Go:**
```go
func someOther_impl(args ...interface{}) interface{} {
    // ...
    var tmp66 interface{} = /* some computation */
    var tmp68 interface{} = /* some computation */
    var tmp70 interface{} = /* some computation */

    // ✅ CORRECT: Type assertions before calling typed export
    return UpdateAllNPCs(tmp66.([]*NPC), tmp68.(int64), tmp70.(int64))
}
```

**Error from stapledons_voyage:**
```
step.go:457: cannot use tmp66 (variable of type interface{}) as []*NPC value in argument to UpdateAllNPCs
```

## Root Cause Analysis

### Chain of Events

1. **M-CODEGEN-TYPED-PARAMS** (completed): Exports now have typed signatures
2. **_impl functions unchanged**: Still generate `interface{}` temporaries
3. **Call mismatch**: `interface{}` args passed to typed params → compile error

### Location

**File:** `internal/gen/golang/codegen_ops.go` - `generateApp` function

When generating a function application:
- If callee is an exported function with typed signature
- And arguments are `interface{}` temporaries
- Need to wrap arguments with type assertions

### Why This Wasn't Caught Before

- Pre-typed-params: Both exports and calls used `interface{}` → compatible
- Post-typed-params: Exports typed, calls still `interface{}` → mismatch
- The fix for typed params created this new incompatibility

## Proposed Solution

### Option A: Type Assertions at Call Sites (Recommended)

When generating `App` (function application), check if callee is a typed export and wrap arguments:

```go
func (g *Generator) generateApp(app *core.App) error {
    // ... existing code to get function name and args ...

    // M-CODEGEN-TYPE-ASSERTIONS: Check if callee is typed export
    if override, ok := g.funcTypeOverrides[funcName]; ok {
        // Generate type-asserted call
        return g.generateTypedCall(funcName, args, override.ParamTypes, override.ReturnType)
    }

    // ... existing untyped call generation ...
}

func (g *Generator) generateTypedCall(name string, args []string, paramTypes []GoType, returnType GoType) error {
    // Wrap each arg with type assertion
    typedArgs := make([]string, len(args))
    for i, arg := range args {
        if i < len(paramTypes) && paramTypes[i] != "interface{}" {
            // arg.(ParamType)
            typedArgs[i] = fmt.Sprintf("%s.(%s)", arg, paramTypes[i])
        } else {
            typedArgs[i] = arg
        }
    }

    // Generate call: FuncName(typedArg1, typedArg2, ...)
    g.emit("%s(%s)", name, strings.Join(typedArgs, ", "))
    return nil
}
```

**Pros:**
- Minimal change - only affects call sites
- Works with existing `funcTypeOverrides` infrastructure
- No changes to variable generation

**Cons:**
- Adds runtime type assertions (panic on type mismatch)

### Option B: Typed Variables in _impl

Generate typed variables instead of `interface{}` when type is known:

```go
var tmp66 []*NPC = /* computation */
```

**Pros:**
- No runtime assertions needed
- Catches type errors at generation time

**Cons:**
- Major refactor of variable generation
- Requires tracking types through all computations
- Much more complex

### Option C: Keep _impl Untyped, Assert at Export Boundary

Keep `_impl` using `interface{}` throughout, but the wrapper already handles conversion.

**Problem:** This doesn't help when `_impl` calls OTHER typed exports.

## Actual Solution Implemented: Option D (Better than A)

**Instead of adding type assertions, redirect `_impl` calls to `_impl` versions.**

The code already had logic in `core.Var` handling to call `_impl` versions from `_impl` functions.
The bug was that `core.VarGlobal` (cross-module function references) was missing this logic.

**The fix (8 lines in `codegen_expr.go`):**
```go
// In VarGlobal case, before the default PascalCase handling:

// M-CODEGEN-TYPE-ASSERTIONS: In _impl functions, call other _impl functions
// to avoid type mismatches (typed exports expect concrete types, not interface{})
if g.expectedReturnType == "interface{}" {
    // Check if this is a known top-level function (has _impl version)
    if _, isTopLevel := g.topLevelFuncs[e.Ref.Name]; isTopLevel {
        g.write(ToGoVarName(e.Ref.Name) + "_impl")
        return nil
    }
}
```

**Why this is better than Option A:**
1. No runtime type assertions needed (no panic risk)
2. Simpler - no new helper functions
3. Consistent with existing `core.Var` behavior
4. Generated code is cleaner (no `.()` assertions everywhere)

**Before fix:**
```go
func gameLoop_impl(g interface{}, p interface{}) interface{} {
    // ❌ Calls typed wrapper - type mismatch!
    var newPlayer interface{} = MovePlayer(p, 1, 0)
}
```

**After fix:**
```go
func gameLoop_impl(g interface{}, p interface{}) interface{} {
    // ✅ Calls _impl version - compatible interface{} types
    var newPlayer interface{} = movePlayer_impl(p, int64(1), int64(0))
}
```

## Implementation Summary

**File modified:** `internal/gen/golang/codegen_expr.go`

Added 8 lines in the `VarGlobal` case (around line 74-82) to redirect calls to `_impl` versions
when generating code inside an `_impl` function.

**Total LOC:** 8 lines added

## Files Modified

| File | Changes | LOC |
|------|---------|-----|
| `internal/gen/golang/codegen_expr.go` | Add _impl redirection for VarGlobal | ~8 |

## Acceptance Criteria

- [x] `_impl` functions call `_impl` versions instead of typed exports
- [x] stapledons_voyage compiles without manual fixes (verified with test case)
- [x] All existing codegen tests pass (`make test` passes)
- [x] Generated Go code compiles (`go build` passes)

## Related Issues

- M-CODEGEN-TYPED-PARAMS (COMPLETED): This bug is a consequence of that fix
- M-CROSS-MODULE (COMPLETED): Cross-module type contamination fix

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09
