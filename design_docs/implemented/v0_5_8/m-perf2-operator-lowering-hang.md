# M-PERF2: Operator Lowering Hang on Complex Modules

**Status**: Investigation
**Target**: v0.5.8
**Priority**: P0 (High - DX blocker for game development)
**Estimated**: 4-8 hours
**Dependencies**: None (M-PERF1 effect checker fix is done)
**Reporter**: stapledons_voyage project

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No syntax changes |
| Preserve Semantic Clarity | 0 | 0 | Semantics unchanged |
| Increase Determinism | 0 | 0 | Behavior unchanged |
| Lower Token Cost | + | +1 | Faster feedback loop = less iteration tokens |
| **Net Score** | | **+1** | **Decision: Move forward** |

## Problem Statement

When running `ailang check sim/test_combined.ail`, the compiler hangs during the **operator lowering phase** (NOT effect checking - that was M-PERF1 and is fixed).

**Reproducer:**
The stapledons_voyage project's `sim/test_combined.ail` file triggers the hang:

```ailang
module sim/test_combined

import sim/protocol (Coord)
import sim/npc_ai (Direction, North)
import std/option (Option, Some, None)

-- ADT types and complex record types...
```

**Current State:**
- Type checking completes successfully
- Effect checking completes successfully (M-PERF1 fix worked!)
- **Operator lowering hangs indefinitely** on `sim/npc_ai` module
- Debug output shows ~305 `lowerExpr` calls before processing stops
- Last call is for a `*core.Lit` (should be trivial, instant return)

**Impact:**
- **Who**: Game developers using AILANG for tile-based games
- **Severity**: DX blocker - cannot compile modules with recursive functions and pattern matching
- **Workarounds**: None currently identified

## Root Cause Analysis (Investigation Status)

### What We Know

**1. Hang Location Identified:**
- Hang is in `internal/pipeline/op_lowering.go` in the `lowerExpr` function
- Specifically when processing `sim/npc_ai` module (decl 12, a LetRec)
- ~305 `lowerExpr` calls complete, then processing stops

**2. Module Compilation Order:**
```
std/prelude → std/list → std/option → sim/protocol → sim/npc_ai → sim/test_combined
```
Earlier modules compile fine; hang starts during `sim/npc_ai`.

**3. Debug Output Pattern:**
```
[DEBUG] Lower() processing decl 12 (type: *core.LetRec)...
[DEBUG] lowerExpr #280 (type: *core.LetRec)
[DEBUG] lowerExpr #281 (type: *core.Lambda)
...
[DEBUG] lowerExpr #305 (type: *core.Lit)
<--- Processing stops here - no more output, no panic
```

**4. Code Inspection Findings:**

**IMPORTANT:** After inspecting `op_lowering.go`, there are **NO explicit fixed-point loops** (`for {` or `for changed`). The code is purely recursive descent:
- `lowerExpr` recursively calls itself for sub-expressions
- `lowerIntrinsic` calls `lowerExprs` for arguments
- All `for` loops iterate over fixed collections (bindings, arms, fields)

**5. sim/npc_ai Has Complex Pattern:**
- Recursive function `updateAllNPCs` with nested pattern matching
- Uses ADT types (Direction, MoveState)
- Multiple LetRec bindings
- List patterns like `[npc, ...rest]`

### Symptom Analysis

The symptoms are unusual:
- **305 lowerExpr calls then silence** - not infinite (counter would hit 50K)
- **Last call is `*core.Lit`** - should return instantly (just `return expr`)
- **No panic** - so not a nil dereference or explicit panic
- **Process appears "stuck"** - either blocking operation or very tight loop

If the problem were:
| Issue | Expected Behavior | Actual Behavior |
|-------|-------------------|-----------------|
| Deep recursion | Counter keeps climbing to 50K, panic | Counter stops at 305 |
| Type lookup deadlock | Goroutine dump shows blocking syscall | TBD - need profile |
| Cycle in AST | Counter keeps going, revisiting nodes | Counter stops at 305 |
| Local tight loop | Counter stops, one frame spinning | **Matches!** |

### Root Cause Identified: Cyclic Type Traversal

**CONFIRMED:** The hang is in `types.Head()` or `t.String()` on cyclic type graphs.

**Call chain at hang:**
```
Lower() on sim/npc_ai
  → lowerExpr(LetRec)
    → lowerExpr(Lambda/Match/...)
      → lowerExpr(App)
        → lowerExpr(Func)
        → lowerExpr(Args[0..N])
          → lowerExpr(ArgN = *core.Lit)  ← prints "lowerExpr #305 (*core.Lit)"
          ← returns instantly (just `return expr`)
        → lowerIntrinsic(...)           ← hang is HERE
          → inferredType := l.CoreTI.Get(typeNode)
          → head := types.Head(inferredType)  ← INFINITE LOOP on cyclic type
          → OR: typeStr := t.String()          ← INFINITE LOOP on cyclic type
```

**Why cyclic types exist:**

`sim/npc_ai` has recursive list types:
```ailang
func updateAllNPCs(npcs: [NPCState], dt: float) -> [NPCState]
```

Where `NPCState` contains fields that reference `[NPCState]`, creating cycles:
```
List[NPCState] → NPCState → ... → List[NPCState] (cycle)
```

Monomorphization and type inference can tie the knot in the type graph (actual pointer cycles) when interning types.

**The buggy code in `getTypeSuffixFromType`:**
```go
func getTypeSuffixFromType(t types.Type) string {
    switch t {
    case types.TInt, types.TFloat, types.TBool, types.TString:
        // ... fast paths ...
    default:
        // DANGER: This traverses cyclic types forever!
        typeStr := t.String()  // ← HANGS HERE
        // ... string comparisons ...
    }
}
```

**Symptoms match perfectly:**
| Observation | Explanation |
|-------------|-------------|
| ~305 lowerExpr calls then silence | Lit returned, lowerIntrinsic entered, type traversal loops |
| Last call is `*core.Lit` | Lit case is O(1), returns immediately |
| No panic | Not stack overflow - tight loop inside `String()` or `Head()` |
| Process appears "stuck" | CPU spinning in recursive type traversal |

### Concrete Investigation Plan

#### Step 1: Instrument the RETURN paths, not just entry
```go
func (l *OpLowerer) lowerExpr(expr core.CoreExpr) core.CoreExpr {
    debugLowerCounter++
    fmt.Fprintf(os.Stderr, "[ENTER] lowerExpr #%d (type: %T)\n", debugLowerCounter, expr)

    result := l.lowerExprImpl(expr)  // Move logic to impl

    fmt.Fprintf(os.Stderr, "[EXIT] lowerExpr #%d\n", debugLowerCounter)
    return result
}
```

This will tell us if we EXIT the Lit case successfully or hang inside.

#### Step 2: Check `types.Head()` for infinite loops
The `lowerIntrinsic` function calls:
```go
head := types.Head(inferredType)
```

If `inferredType` is a cyclic or malformed type, `types.Head()` might loop. Add:
```go
// In lowerIntrinsic, before Head call:
fmt.Fprintf(os.Stderr, "[DEBUG] About to call types.Head on: %v\n", inferredType)
head := types.Head(inferredType)
fmt.Fprintf(os.Stderr, "[DEBUG] types.Head returned: %v\n", head)
```

#### Step 3: Check `getTypeSuffixFromType()`
Line 486 calls `t.String()`. If `t` is cyclic:
```go
typeStr := t.String()  // Could loop forever on cyclic types!
```

Add iteration guard or check for cycles before calling String().

#### Step 4: Goroutine Stack Dump
```bash
AILANG_DEBUG=1 ailang check sim/test_combined.ail &
sleep 2
kill -QUIT $!   # SIGQUIT triggers goroutine dump
```

Look at the main goroutine stack - if it shows:
- Single `lowerExpr` frame at bottom → local loop inside that call
- Many `lowerExpr` frames → recursive issue (but counter should still climb)
- Blocking syscall → deadlock in type lookup

#### Step 5: CPU Profile
```bash
go tool pprof -http=:8080 cpu.prof
```
Run with CPU profiling to see where time is spent.

#### Step 6: Add Visited Set Guard
```go
type OpLowerer struct {
    // ... existing fields ...
    visiting map[core.CoreExpr]bool  // Cycle detection
}

func (l *OpLowerer) lowerExpr(expr core.CoreExpr) core.CoreExpr {
    if l.visiting[expr] {
        panic(fmt.Sprintf("cycle detected in lowerExpr on %T: %#v", expr, expr))
    }
    l.visiting[expr] = true
    defer delete(l.visiting, expr)
    // ... rest of function
}
```

#### Step 7: Minimal Reproducer Test
Create unit test with suspected patterns:
```go
func TestLowerExpr_MatchWithListPattern(t *testing.T) {
    // Construct minimal Core AST that mirrors updateAllNPCs pattern:
    // - LetRec with recursive function
    // - Match over list with [x, ...rest] pattern
    // - Recursive call in arm body

    lowerer := NewOpLowerer(typeEnv, coreTypeInfo)
    result := lowerer.lowerExpr(minimalAST)
    // Should not hang
}
```

### Likely Offending Patterns in sim/npc_ai

```ailang
export pure func updateAllNPCs(npcs: [NPCState], dt: float) -> [NPCState] {
    match npcs {
        [] -> [],
        [npc, ...rest] -> {
            let updated = updateNPC(npc, dt)
            [updated, ...updateAllNPCs(rest, dt)]
        }
    }
}
```

Possible issues:
1. **List spread `[npc, ...rest]`** - pattern matching sugar that desugars to something complex
2. **List construction `[updated, ...updateAllNPCs(...)]`** - spread in expression context
3. **Type inference on recursive call** - polymorphic types might create cycles

## Goals

**Primary Goal:** Operator lowering completes in O(n) time for all modules.

**Success Metrics:**
- `ailang check` on sim/test_combined.ail completes in <5 seconds
- `ailang check` on sim/npc_ai.ail completes in <3 seconds
- No observable slowdown for small files (<100ms overhead)
- All existing lowering tests still pass

## Solution Design

### Overview

**Primary fix:** Make `getTypeSuffixFromType()` purely shallow - never call `t.String()` on arbitrary types. Lowering only needs top-level type tags, not full pretty-printed strings.

**Secondary fix:** Add cycle-safe guards to `types.Head()` if needed.

### The Fix: Shallow Type Tagging

**Current buggy code:**
```go
func getTypeSuffixFromType(t types.Type) string {
    switch t {
    case types.TInt, types.TFloat, types.TBool, types.TString:
        // ... fast paths ...
    default:
        // DANGER: Deep traversal on cyclic types!
        typeStr := t.String()  // ← REMOVE THIS
        // ... string comparisons ...
    }
}
```

**Fixed code (purely shallow):**
```go
func getTypeSuffixFromType(t types.Type) string {
    // Direct primitives - O(1)
    switch t {
    case types.TInt:
        return "Int"
    case types.TFloat:
        return "Float"
    case types.TBool:
        return "Bool"
    case types.TString:
        return "String"
    }

    // Shallow application: List[a] - only check top-level constructor
    if app, ok := t.(*types.TApp); ok {
        if con, ok := app.Constructor.(*types.TCon); ok {
            if con.Name == "List" {
                return "List"
            }
        }
    }

    // Last resort: default to Int (backward compatibility)
    // NO t.String() - no risk of cycles
    return "Int"
}
```

**Why this works:**
- Lowering only needs to know: Int, Float, Bool, String, or List
- Never traverses type arguments or nested structures
- O(1) constant time regardless of type complexity
- Cannot hang on cyclic types

### Optional: Cycle-Safe `types.Head()` Guard

If `types.Head()` also hangs (less likely but possible):

```go
func safeHead(t types.Type) types.HeadTag {
    const maxDepth = 64
    visited := make(map[types.Type]bool)

    var goHead func(types.Type, int) types.HeadTag
    goHead = func(tp types.Type, depth int) types.HeadTag {
        if depth > maxDepth {
            panic(fmt.Sprintf("types.Head: exceeded max depth on %T", tp))
        }
        if visited[tp] {
            panic(fmt.Sprintf("types.Head: cycle detected on %T", tp))
        }
        visited[tp] = true
        return types.Head(tp)
    }

    return goHead(t, 0)
}
```

### Quick Validation Test

**Before implementing full fix, confirm culprit with this 1-line change:**

```go
// In getTypeSuffixFromType, replace default case:
default:
    // TEMP: skip t.String() to confirm this is the hang
    return "Int"
```

If `ailang check sim/test_combined.ail` suddenly works → confirmed.

### Implementation Plan

**Phase 1: Quick Fix** (~30 minutes)
- [ ] Replace `t.String()` call in `getTypeSuffixFromType` with shallow logic
- [ ] Rebuild and test with sim/test_combined.ail
- [ ] Verify hang is fixed

**Phase 2: Add Guards** (~1 hour)
- [ ] Add `safeHead()` wrapper with cycle detection
- [ ] Add debug logging for type cycles (gated by `--debug-compile`)
- [ ] Run full test suite

**Phase 3: Testing** (~2 hours)
- [ ] Add unit test with synthetic cyclic type
- [ ] Add integration test with sim/npc_ai pattern
- [ ] Add benchmark for type-heavy modules

### Files to Modify

**Modified files:**
- `internal/pipeline/op_lowering.go` - Fix `getTypeSuffixFromType()` (~10 LOC change)
- `internal/pipeline/op_lowering_test.go` - Add regression tests (~50 LOC)

**Optional (if `types.Head` also hangs):**
- `internal/types/head.go` - Add cycle detection

## Success Criteria

- [ ] `ailang check sim/test_combined.ail` completes in <5 seconds
- [ ] `ailang check sim/npc_ai.ail` completes in <3 seconds
- [ ] All existing pipeline tests pass
- [ ] Regression test for this pattern added
- [ ] No performance regression on simple files

## Testing Strategy

**Unit tests:**
- Test lowerExpr with deeply nested Match expressions
- Test lowerExpr with recursive LetRec bindings
- Test lowerExpr with list spread patterns

**Integration tests:**
- Test full pipeline on sim/npc_ai.ail
- Test full pipeline on sim/test_combined.ail

**Manual testing:**
- Verify stapledons_voyage compilation works end-to-end

## Timeline

**Now that root cause is identified:**

**Phase 1** (~30 minutes):
- Apply quick fix: remove `t.String()` from `getTypeSuffixFromType`
- Rebuild and verify hang is fixed

**Phase 2** (~1 hour):
- Add cycle-safe guards
- Run full test suite

**Phase 3** (~1-2 hours):
- Add regression tests
- Documentation

**Total: ~2-4 hours (down from original 6-8 estimate)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Root cause is in type system, not lowering | Medium | Profile + stack dump to confirm location |
| Cyclic types from monomorphization | High | Check mono output for cycles |
| Fix breaks other lowering patterns | High | Comprehensive test suite |

## References

- Related: [M-PERF1 Effect Checker Performance](m-perf1-effect-checker-large-arrays.md) - Completed
- Operator lowering: `internal/pipeline/op_lowering.go`
- Type operations: `internal/types/head.go`, `internal/types/type.go`
- Debug session findings: Session 2025-12-08

## Appendix: Key Code Paths

**lowerExpr switch cases that call recursively:**
```
Let     → lowerExpr(Value), lowerExpr(Body)
LetRec  → for bindings: lowerExpr(Value), then lowerExpr(Body)
Lambda  → lowerExpr(Body)
App     → lowerExpr(Func), lowerExprs(Args)
If      → lowerExpr(Cond), lowerExpr(Then), lowerExpr(Else)
Match   → lowerExpr(Scrutinee), for arms: lowerExpr(Guard), lowerExpr(Body)
BinOp   → creates Intrinsic, calls lowerIntrinsic
UnOp    → creates Intrinsic, calls lowerIntrinsic
Record  → for fields: lowerExpr(v)
List    → lowerExprs(Elements)
Lit     → return expr (INSTANT - no recursion)
```

**lowerIntrinsic operations:**
```
OpAnd/OpOr → lowerExpr(Args[0]), lowerExpr(Args[1]), return If node
Other ops  → CoreTI.Get(), types.Head(), lowerExprs(Args), create App
```

---

**Document created**: 2025-12-08
**Last updated**: 2025-12-08
