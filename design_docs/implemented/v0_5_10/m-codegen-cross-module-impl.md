# M-CODEGEN-CROSS-MODULE: Cross-Module Function Calls Use _impl

**Status**: Implemented
**Version**: v0.5.10
**Priority**: P0 (Critical) - Blocks stapledons_voyage compilation
**Dependencies**: None
**Reporter**: stapledons_voyage (Agent Inbox)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No source syntax change |
| Preserve Semantic Clarity | 0 | 0 | No semantic change |
| Increase Determinism | + | +1 | More predictable codegen output |
| Lower Token Cost | 0 | 0 | Similar generated code size |
| **Net Score** | | **+1** | **Decision: Move forward** |

## Problem Statement

Go codegen produces incorrect function calls for cross-module references in `_impl` functions, causing Go compilation errors.

**Root Cause:**

When module A imports function `f` from module B:
1. In `_impl` functions, all variables are `interface{}`
2. Cross-module function references generate typed wrappers (`SpectralFromRoll`)
3. Typed wrappers expect concrete types (`float64`), not `interface{}`
4. Result: type mismatch error

**Current Behavior (WRONG):**

```go
// In generateStar_impl (interface{} everywhere)
var spectralRoll interface{} = hashToFloat_impl(tmp32)  // OK
var spectral interface{} = SpectralFromRoll(spectralRoll)  // ERROR!
// SpectralFromRoll expects float64, got interface{}
```

**Go Compiler Error:**
```
cannot use spectralRoll (variable of type interface{}) as float64 value
in argument to SpectralFromRoll: need type assertion
```

**Expected Behavior:**
```go
// Should use _impl version for consistency
var spectral interface{} = spectralFromRoll_impl(spectralRoll)  // OK
```

## Technical Analysis

**Code Location:** `internal/gen/golang/codegen_expr_simple.go:124-133`

```go
// M-CODEGEN-TYPE-ASSERTIONS: In _impl functions, call other _impl functions
if g.expectedReturnType == "interface{}" {
    // Check if this is a known top-level function (has _impl version)
    if _, isTopLevel := g.topLevelFuncs[e.Ref.Name]; isTopLevel {  // BUG!
        g.write(ToGoVarName(e.Ref.Name) + "_impl")
        return nil
    }
}
g.write(ToPascalCase(e.Ref.Name))  // Falls through for imported functions
```

**Bug:** `g.topLevelFuncs` only contains functions defined in the current module. Imported functions from other modules are not tracked, causing them to fall through to the typed wrapper pattern.

**VarGlobal Structure:**
```go
type VarGlobal struct {
    Ref struct {
        Module string  // e.g., "sim/starmap"
        Name   string  // e.g., "spectralFromRoll"
    }
}
```

The `Ref.Module` field indicates cross-module references but is not being used.

## Solution Design

### Option A: Assume All Cross-Module Functions Have _impl (Recommended)

When in `_impl` context and `Ref.Module` is a real module path (not `$adt`, `$builtin`, etc.), generate `_impl` version.

**Rationale:**
- Simple change (3-5 lines)
- Works for all cross-module references
- All exported functions have both typed wrapper and `_impl` version
- Safe assumption: if it's a function reference, `_impl` exists

### Option B: Track Imported Functions

Expand `topLevelFuncs` or create `importedFuncs` map to track imported functions.

**Cons:**
- More complex (requires tracking during import processing)
- Overkill when Option A works

### Chosen Solution: Option A

**Implementation:**

```go
// In generateVarGlobal (codegen_expr_simple.go)
func (g *Generator) generateVarGlobal(e *core.VarGlobal) error {
    // ... existing ADT, effect, math checks ...

    // M-CODEGEN-CROSS-MODULE: In _impl functions, use _impl for cross-module calls
    if g.expectedReturnType == "interface{}" {
        // Check if this is from another USER-DEFINED module (not pseudo-module, not stdlib)
        // Stdlib modules (std/*) use typed wrappers that call runtime helpers
        if e.Ref.Module != "" && !strings.HasPrefix(e.Ref.Module, "$") && !strings.HasPrefix(e.Ref.Module, "std/") {
            // Cross-module function reference - use _impl version
            g.write(ToGoVarName(e.Ref.Name) + "_impl")
            return nil
        }
        // Also check local top-level functions
        if _, isTopLevel := g.topLevelFuncs[e.Ref.Name]; isTopLevel {
            g.write(ToGoVarName(e.Ref.Name) + "_impl")
            return nil
        }
    }
    g.write(ToPascalCase(e.Ref.Name))
    return nil
}
```

**Key Changes:**
1. Check if `e.Ref.Module` is a real USER module (not empty, not `$` prefix, not `std/` prefix)
2. Exclude stdlib modules - they use typed wrappers that map to runtime helpers
3. If user module and in `_impl` context, generate `funcName_impl`
4. Keep existing local function check as fallback

**Important Discovery:** Initial fix caused `undefined: concat_impl` errors because stdlib
functions don't generate `_impl` versions - they map to runtime helpers like `ConcatString`.

## Success Criteria

- [x] Cross-module function calls use `_impl` version in `_impl` functions
- [x] `spectralFromRoll`, `makeStar`, `makeVec3` calls work correctly
- [x] stapledons_voyage compiles: `go build ./...`
- [x] All existing AILANG tests pass

## Files to Modify

- `internal/gen/golang/codegen_expr_simple.go` (5-10 LOC change)

## Test Plan

1. **Unit Test:** Add test case for cross-module function reference in _impl context
2. **Integration:** Compile stapledons_voyage with `go build ./...`
3. **Regression:** Run `make test` to ensure no breakage

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Some VarGlobal refs aren't functions | Medium | ADT/effect/math checks already exclude non-functions |
| Module path edge cases | Low | Check for non-empty and non-`$` prefix is sufficient |

## References

- GitHub Issue: stapledons_voyage agent inbox message
- Related: M-CODEGEN-LIST (same file, different issue)
- Related: M-CODEGEN-TUPLE-PATTERN (pattern matching fix)

---

**Document created**: 2025-12-11
**Implemented**: 2025-12-11
