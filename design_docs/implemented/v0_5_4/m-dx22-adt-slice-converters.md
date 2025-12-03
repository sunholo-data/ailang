# M-DX22: Auto-Generate ADT Slice Converters

**Status**: Planned
**Target**: v0.5.5
**Priority**: P1 (Medium)
**Estimated**: 2 hours
**Dependencies**: None (builds on existing M-DX12 infrastructure)
**Source**: DX feedback from `stapledons_voyage` agent

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes hand-written boilerplate converters |
| Preserve Semantic Clarity | 0 | 0 | Same semantics, just auto-generated |
| Increase Determinism | + | +1 | Generated code is reproducible |
| Lower Token Cost | + | +1 | Less code for AI to maintain (~50 LOC saved) |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

Users currently write manual slice conversion functions in their `runtime.go`:

```go
// Manual boilerplate in stapledons_voyage/runtime.go (~50 LOC)
func convertToDirectionSlice(v interface{}) []*Direction { ... }
func convertToDrawCmdSlice(v interface{}) []*DrawCmd { ... }
func convertToNPCSlice(v interface{}) []*NPC { ... }
func convertToTileSlice(v interface{}) []*Tile { ... }
func convertToKeyEventSlice(v interface{}) []*KeyEvent { ... }
```

**Current State:**
- Existing `writeADTSliceConverters()` only generates for types in `g.adtSliceTypes`
- Types must be explicitly registered - no automatic discovery
- Users write ~10 LOC per ADT type manually

**Impact:**
- Every project using ADT slices duplicates this boilerplate
- Manual code = opportunity for bugs and inconsistency
- AI must maintain code that could be auto-generated

## Goals

**Primary Goal:** Auto-generate `convertToXxxSlice` functions for all ADT types discovered during compilation.

**Success Metrics:**
- [ ] All ADT types get converters generated automatically
- [ ] Zero manual runtime.go converters needed
- [ ] stapledons_voyage compiles without manual converters

## Solution Design

### Overview

Generate slice converters for ALL ADT types defined in the module, not just explicitly registered ones.

### Architecture

**Current flow:**
```
generateSumType() → writes type definition only
writeADTSliceConverters() → only types in g.adtSliceTypes
```

**Proposed flow:**
```
generateSumType() → writes type + registers in g.allADTTypes
writeADTSliceConverters() → iterates ALL g.allADTTypes
```

### Implementation Plan

**Phase 1: Track ADT Types** (~30 min)
- [ ] Add `allADTTypes map[string]bool` to Generator
- [ ] Register types in `generateSumType()`

**Phase 2: Generate All Converters** (~1 hour)
- [ ] Modify `writeADTSliceConverters()` to use `allADTTypes`
- [ ] Skip duplicates (if already in adtSliceTypes)

**Phase 3: Test** (~30 min)
- [ ] Add test case for auto-discovery
- [ ] Verify stapledons_voyage works

### Files to Modify/Create

**Modified files:**
- `internal/gen/golang/codegen.go` - Add allADTTypes field (~3 LOC)
- `internal/gen/golang/codegen_types.go` - Register in generateSumType (~2 LOC)
- `internal/gen/golang/codegen_runtime.go` - Use allADTTypes (~5 LOC)
- `internal/gen/golang/codegen_test.go` - Add test (~20 LOC)

**Total:** ~30 LOC

## Examples

### Example 1: Auto-generated converters

**Before (manual runtime.go):**
```go
// User must write this for each ADT type
func convertToNPCSlice(v interface{}) []*NPC {
    if v == nil { return nil }
    src, ok := v.([]interface{})
    if !ok { panic(...) }
    out := make([]*NPC, len(src))
    for i, e := range src {
        out[i] = e.(*NPC)
    }
    return out
}
```

**After (auto-generated in funcs.go):**
```go
// Generated automatically for every ADT type
func convertToNPCSlice(v interface{}) []*NPC { ... }
func convertToTileSlice(v interface{}) []*Tile { ... }
func convertToDrawCmdSlice(v interface{}) []*DrawCmd { ... }
// etc. for all ADT types in the module
```

## Success Criteria

- [ ] All ADT types get `convertToXxxSlice` generated
- [ ] stapledons_voyage compiles without manual converters
- [ ] All existing tests pass
- [ ] No performance regression (< 50ms build time increase)

## Non-Goals

**Not in this feature:**
- Struct-to-struct converters - Only ADT slice converters
- Generic list helpers - Use existing ListHead/ListTail

## References

- M-DX12: Original ADT slice converter implementation
- [codegen_runtime.go](../../../internal/gen/golang/codegen_runtime.go) - Current implementation
- stapledons_voyage feedback message (2025-12-03)

---

**Document created**: 2025-12-03
**Last updated**: 2025-12-03
