# M-CODEGEN-ADT-DOUBLE-PAREN: Fix Empty Double-Paren ADT Constructor Calls

**Status**: Implemented ✅
**Target**: v0.6.1
**Priority**: P0 (High) - Blocks stapledons_voyage compilation
**Estimated**: 2-4 hours
**Actual**: ~1.5 hours
**Dependencies**: None
**GitHub Issue**: #52

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix, no syntax change |
| Preserve Semantic Clarity | + | +1 | Ensures constructor args are passed correctly |
| Increase Determinism | + | +1 | Deterministic Go codegen for ADT constructors |
| Lower Token Cost | 0 | 0 | Bug fix, no token impact |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward

## Problem Statement

When a function returns an ADT variant with many parameters (like `Viewport` in `DrawCmd`), the generated Go code produces invalid double-paren calls:

```go
// BROKEN - empty double-paren call
NewDrawCmdViewport()()

// EXPECTED - args passed through
NewDrawCmdViewport(id, shapeType, x, y, w, h, ...)
```

**Current State:**
- stapledons_voyage `viewport.ail` generates invalid Go code
- Workaround: renamed to `viewport.ail.disabled`
- Pattern: `NewConstructor()()` instead of `NewConstructor(args...)`

**Impact:**
- Blocks any ADT constructor with fields from being used in function returns
- stapledons_voyage game engine relies heavily on ADT variants for protocol messages

## Root Cause Analysis

### Hypothesis 1: FieldCount Mismatch

In `codegen_expr_simple.go:88-92`:
```go
if ctorInfo, ok := g.adtConstructors[ctorName]; ok && ctorInfo.FieldCount == 0 {
    g.write(goFuncName + "()")  // First () - treating as nullary
} else {
    g.write(goFuncName)
}
```

If `ctorInfo` is NOT found for the constructor, the code falls through to line 91 and emits just `goFuncName`. Then if this VarGlobal is wrapped in an App node (which it would be when called with args), the App handler adds `()`.

**But wait** - the App handler at `codegen_expr_app.go:117-176` ALSO calls the constructor. This could result in:
1. VarGlobal emits `NewDrawCmdViewport()` (nullary branch taken incorrectly)
2. App wraps this and adds another `()` → `NewDrawCmdViewport()()`

### Hypothesis 2: Constructor Not Registered

The `Viewport` constructor may not be registered in `adtConstructors` map, causing:
1. `codegen_expr_simple.go` falls through to generic VarGlobal handling
2. Later App handling tries to apply it as a function

### Hypothesis 3: App → VarGlobal Double Generation

When an ADT constructor is called:
1. Parser creates `App(VarGlobal("$adt.make_DrawCmd_Viewport"), args...)`
2. `generateApp` at line 117 checks for ADT constructors
3. If it doesn't find `Viewport` in `adtConstructors`, it falls through to line 200+
4. Line 211 generates the func expression (VarGlobal) which outputs `NewDrawCmdViewport()`
5. Line 214 adds `(` and args and `)` → `NewDrawCmdViewport()(args...)`

**Most likely root cause:** The constructor lookup key mismatch between:
- `adtConstructors[ctorName]` uses just `"Viewport"`
- But the App function might reference `"make_DrawCmd_Viewport"` or full variant name

## Goals

**Primary Goal:** Fix codegen to emit correct ADT constructor calls with all arguments.

**Success Metrics:**
- stapledons_voyage `viewport.ail` compiles to valid Go code
- No `()()` double-paren patterns in generated code
- All existing codegen tests pass
- New regression test for multi-arg ADT constructors

## Solution Design

### Overview

Debug the exact code path that produces `()()` and fix the constructor lookup/generation logic.

### Investigation Plan

**Step 1: Reproduce with minimal example**

Create test case in AILANG:
```ailang
module test/viewport

type DrawCmd =
  | Clear
  | Viewport(id: int, x: float, y: float, w: float, h: float)

export let makeViewport: int -> float -> float -> float -> float -> DrawCmd =
  \id. \x. \y. \w. \h. Viewport(id, x, y, w, h)
```

**Step 2: Debug codegen path**

Add debug logging to trace:
1. What's in `adtConstructors` after type generation
2. What VarGlobal ref name is being looked up
3. Whether App or VarGlobal handler is emitting the `()`

**Step 3: Fix the mismatch**

Based on findings, likely fixes:
1. Ensure constructor is registered with correct key
2. Or fix lookup in `getADTConstructorForApp` to handle all naming patterns
3. Or prevent VarGlobal from emitting `()` when it's going to be wrapped in App

### Implementation Plan

**Phase 1: Reproduce & Debug** (~1 hour)
- [ ] Create minimal repro test case
- [ ] Add debug logging to codegen_expr_simple.go and codegen_expr_app.go
- [ ] Trace exact code path producing `()()`
- [ ] Identify root cause

**Phase 2: Fix** (~1-2 hours)
- [ ] Implement fix based on root cause
- [ ] Add unit test for multi-arg ADT constructor
- [ ] Verify stapledons_voyage viewport.ail compiles

**Phase 3: Cleanup** (~30 min)
- [ ] Remove debug logging
- [ ] Update this design doc with implementation report
- [ ] Verify all tests pass

### Files to Modify/Create

**Modified files:**
- `internal/gen/golang/codegen_expr_simple.go` - Fix VarGlobal ADT handling (~10 LOC)
- `internal/gen/golang/codegen_expr_app.go` - Fix App ADT constructor handling (~10 LOC)

**New files:**
- `internal/gen/golang/codegen_adt_multiarg_test.go` - Regression test (~50 LOC)

## Examples

### Example 1: DrawCmd Viewport Constructor

**AILANG Source:**
```ailang
type DrawCmd =
  | Clear
  | Viewport(id: int, x: float, y: float, w: float, h: float)

let v = Viewport(1, 0.0, 0.0, 800.0, 600.0)
```

**Current (BROKEN):**
```go
var v interface{} = NewDrawCmdViewport()()  // ← empty double-paren
```

**Expected (FIXED):**
```go
var v interface{} = NewDrawCmdViewport(int64(1), float64(0.0), float64(0.0), float64(800.0), float64(600.0))
```

### Example 2: Function Returning ADT

**AILANG Source:**
```ailang
export let makeViewport: int -> float -> float -> float -> float -> DrawCmd =
  \id. \x. \y. \w. \h. Viewport(id, x, y, w, h)
```

**Current (BROKEN):**
```go
func MakeViewport(id int64, x, y, w, h float64) *DrawCmd {
    return makeViewport_impl(id, x, y, w, h).(*DrawCmd)
}

func makeViewport_impl(id, x, y, w, h interface{}) interface{} {
    return NewDrawCmdViewport()()  // ← bug here
}
```

**Expected (FIXED):**
```go
func makeViewport_impl(id, x, y, w, h interface{}) interface{} {
    return NewDrawCmdViewport(id.(int64), x.(float64), y.(float64), w.(float64), h.(float64))
}
```

## Success Criteria

- [ ] stapledons_voyage `viewport.ail` compiles without errors
- [ ] Generated Go code has no `()()` double-paren patterns
- [ ] New regression test passes
- [ ] All existing codegen tests pass
- [ ] `make test` passes
- [ ] Documentation updated (this design doc)

## Testing Strategy

**Unit tests:**
- Add test in `codegen_adt_test.go` for multi-arg ADT constructor
- Test both nullary (0 args) and n-ary (multiple args) constructors
- Test ADT constructor in function return position

**Integration tests:**
- Compile and run stapledons_voyage `viewport.ail`
- Verify generated Go code executes correctly

**Manual testing:**
- Enable `viewport.ail.disabled` → `viewport.ail`
- Run stapledons_voyage full compilation

## Non-Goals

**Not in this feature:**
- Refactoring ADT codegen architecture - just fix the bug
- Adding new ADT features - bug fix only
- Performance optimization - correctness first

## Timeline

**Single session** (~2-4 hours):
- 1h: Reproduce and debug
- 1-2h: Fix implementation
- 30min: Tests and cleanup

**Total: ~2-4 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fix breaks other ADT codegen | High | Run full test suite, test stapledons_voyage |
| Root cause is deeper than expected | Medium | Time-box investigation to 1h, escalate if needed |
| Multiple code paths affected | Medium | Audit all VarGlobal/App handlers for ADT |

## Related Documents

**Most Relevant:**
- [M-CODEGEN-ADT-TYPE-ASSERT](../../implemented/v0_5_10/m-codegen-adt-type-assert.md) - Recent ADT codegen fix (nullary vs n-ary distinction)
- [M-DX26](../../implemented/) - Dual function generation (_impl + typed wrapper)

**Code Locations:**
- `internal/gen/golang/codegen_expr_simple.go` - VarGlobal generation
- `internal/gen/golang/codegen_expr_app.go` - App generation for ADT constructors
- `internal/gen/golang/codegen.go` - ADTConstructorInfo struct

## References

- GitHub Issue #52 - stapledons_voyage bug report
- Agent message from stapledons_voyage (2025-12-19)

## Future Work

- Consider unifying ADT constructor code paths to prevent similar bugs
- Add CI check for `()()` pattern in generated code (lint rule)

---

## Implementation Report

### Root Cause Confirmed

**Hypothesis 1 was correct, with a twist.** The issue was in how nullary vs non-nullary constructors were distinguished:

1. **`codegen_expr_app.go:117`**: The condition `len(ctorInfo.FieldTypes) > 0` meant nullary constructors (with 0 fields) fell through to generic App handling, which could cause double-paren issues.

2. **`codegen_decl.go:488-508`**: The `isNullaryADTConstructor` function returned `true` for ALL `$adt.make_*` patterns without checking the actual field count from the `adtConstructors` map.

### Changes Made

**File 1: `internal/gen/golang/codegen_expr_app.go`**

Line 117 - Changed condition from:
```go
if ctorInfo := g.getADTConstructorForApp(app); ctorInfo != nil && len(ctorInfo.FieldTypes) > 0 {
```
To:
```go
if ctorInfo := g.getADTConstructorForApp(app); ctorInfo != nil {
```

This ensures ALL ADT constructors (nullary and non-nullary) are handled by the ADT-specific code path, preventing fallthrough to generic App handling.

**File 2: `internal/gen/golang/codegen_decl.go`**

Updated `isNullaryADTConstructor` function (lines 488-508) to properly parse the `$adt.make_TypeName_CtorName` pattern and look up actual field count:

```go
func (g *Generator) isNullaryADTConstructor(vg *core.VarGlobal) bool {
    if vg.Ref.Module == "$adt" && strings.HasPrefix(vg.Ref.Name, "make_") {
        parts := strings.SplitN(vg.Ref.Name[5:], "_", 2)
        if len(parts) == 2 {
            ctorName := parts[1]
            if info, ok := g.adtConstructors[ctorName]; ok {
                return len(info.FieldTypes) == 0
            }
        }
        return false  // Default to false (safer)
    }
    if info, ok := g.adtConstructors[vg.Ref.Name]; ok {
        return len(info.FieldTypes) == 0
    }
    return false
}
```

**File 3: `internal/gen/golang/codegen_adt_multiarg_test.go` (NEW)**

Added regression test with 3 test cases:
- `TestADTConstructorMultiArg`: Registered multi-arg constructor
- `TestADTConstructorUnregistered`: Unregistered constructor falls back safely
- `TestADTConstructorNullary`: Registered nullary constructor generates `()`

### Test Results

```
✅ make test - All tests pass
✅ make lint - No errors
✅ Generated code verified - no ()() patterns
```

### Success Criteria Status

- [x] Generated Go code has no `()()` double-paren patterns
- [x] New regression test passes
- [x] All existing codegen tests pass
- [x] `make test` passes
- [x] Documentation updated (this design doc)
- [ ] stapledons_voyage `viewport.ail` compiles without errors (to be tested by downstream project)

### LOC Summary

| File | Lines Changed |
|------|---------------|
| codegen_expr_app.go | 5 (added comment, removed condition) |
| codegen_decl.go | 20 (rewrote isNullaryADTConstructor) |
| codegen_adt_multiarg_test.go | ~120 (new test file) |
| **Total** | ~145 LOC |

---

**Document created**: 2025-12-19
**Last updated**: 2025-12-19
