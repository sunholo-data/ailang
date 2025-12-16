# M-DX30: Add Missing EqString Runtime Helper

**Status**: Implemented
**Target**: v0.5.11
**Priority**: P1 (blocks external projects)
**Estimated**: 30 minutes
**Dependencies**: None

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No change to AILANG syntax |
| Preserve Semantic Clarity | + | +1 | String equality now works correctly |
| Increase Determinism | + | +1 | Consistent codegen for all comparison operators |
| Lower Token Cost | 0 | 0 | No change to token footprint |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Problem Statement

Go codegen generates calls to `EqString(a, b)` for string equality comparisons, but the runtime helper function is not emitted in `runtime.go`. This causes Go build failures.

**Current State:**
- `EqInt` exists in runtime.go for integer equality
- `EqString` is generated in code but function is missing
- Go build fails: `undefined: EqString`

**Error:**
```
sim_gen/bridge.go:2146:30: undefined: EqString
```

**Generated code:**
```go
var tmp382 interface{} = EqString(tmp381, "pressed")  // ERROR: EqString undefined
```

## Goals

**Primary Goal:** Add `EqString` function to runtime helpers

**Success Metrics:**
- String equality comparisons compile successfully
- stapledons_voyage builds without errors
- Consistent with existing `EqInt` pattern

## Solution Design

### Overview

Add `EqString` function alongside existing `EqInt` in the runtime helpers generator.

### Implementation

Add to `codegen_runtime.go` in the arithmetic/comparison helpers section:

```go
// EqString - string equality with interface{} handling
func EqString(a, b interface{}) bool {
    return a.(string) == b.(string)
}
```

### Files to Modify

**Modified files:**
- `internal/gen/golang/codegen_runtime.go` - Add EqString function (~5 LOC)

## Success Criteria

- [ ] `EqString` function emitted in runtime.go
- [ ] String equality comparisons work in generated code
- [ ] stapledons_voyage builds successfully
- [ ] All existing tests pass

---

**Document created**: 2025-12-12
**Last updated**: 2025-12-12
