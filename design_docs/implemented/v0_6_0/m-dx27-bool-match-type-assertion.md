# M-DX27: Fix Invalid Bool Type Assertion in Nested Match

**Status**: Implemented
**Target**: v0.5.11
**Priority**: P1 (Medium-High - blocks external projects)
**Estimated**: 2 hours
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No change to AILANG syntax |
| Preserve Semantic Clarity | + | +1 | Fixes incorrect Go codegen - generated code now matches intent |
| Increase Determinism | + | +1 | Same AILANG always produces same valid Go code |
| Lower Token Cost | 0 | 0 | No change to token footprint |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

## Problem Statement

The Go code generator emits invalid type assertions for bool values in nested match expressions. When a bool variable is extracted from an ADT field and used as a match scrutinee in a nested context, the codegen incorrectly adds `.(bool)` type assertion even though the variable is already a concrete `bool`, not an `interface{}`.

**Current State:**
- Nested bool match generates invalid Go code
- `go build` fails with: `invalid operation: s (variable of type bool) is not an interface`
- External projects (stapledons_voyage) blocked by this bug

**Reproduction:**
```ailang
ContentStarfield(d, s) => [d, match s { true => 1.0, false => 0.0 }, 0.0]
```

**Generated Go (invalid):**
```go
s := _adt.ContentStarfield.Scroll  // s is bool
if s.(bool) {  // ERROR: s is already bool, not interface{}
    return 1.0
}
```

**Impact:**
- External projects using ADTs with bool fields cannot compile
- Blocks stapledons_voyage project development
- Bug reported via ailang messages (2 duplicate reports)

## Goals

**Primary Goal:** Generate valid Go code for nested match expressions with bool scrutinees.

**Success Metrics:**
- Generated Go code compiles without errors
- Existing tests continue to pass
- New test case for nested bool match on ADT field

## Solution Design

### Overview

The bug is in `generateFlatBoolMatchChain()` in `internal/gen/golang/codegen_match.go`. This function unconditionally adds `.(bool)` type assertion on lines 740 and 754, without checking if the expression is already a concrete bool type.

The fix is to use the existing `exprProducesInterface()` helper to conditionally add the type assertion only when needed.

### Architecture

**Root Cause Location:**
- File: `internal/gen/golang/codegen_match.go`
- Function: `generateFlatBoolMatchChain()` (lines 704-768)
- Lines 740 and 754 unconditionally emit `.(bool)`

**The pattern in other match codegen:**
The codebase already has the correct pattern elsewhere. For example, line 88:
```go
// M-DX25.6: Only type-assert if scrutinee produces interface{}
if g.exprProducesInterface(match.Scrutinee) {
    g.writef("_adt := _scrutinee.(*%s)\n", goADTName)
} else {
    // Scrutinee is already typed - no assertion needed
    g.writef("_adt := _scrutinee\n")
}
```

### Implementation Plan

**Phase 1: Fix** (~1 hour)
- [ ] Modify `generateFlatBoolMatchChain()` to check `exprProducesInterface(entry.Condition)` before adding `.(bool)`
- [ ] Apply fix to both line 740 (first condition) and line 754 (else-if conditions)

**Phase 2: Testing** (~1 hour)
- [ ] Add test case for nested bool match on ADT field
- [ ] Add test case for nested bool match on interface{} (should still add assertion)
- [ ] Verify existing codegen tests pass
- [ ] Run full test suite

### Files to Modify/Create

**Modified files:**
- `internal/gen/golang/codegen_match.go` - Fix `generateFlatBoolMatchChain()`, ~10 LOC change
- `internal/gen/golang/codegen_match_test.go` - Add test cases, ~50 LOC

## Examples

### Example 1: Nested bool match on ADT field (the bug)

**AILANG Input:**
```ailang
type Content =
  | ContentStarfield(density: float, scroll: bool)
  | ContentSolid(color: int)

let render = \content. match content {
  ContentStarfield(d, s) => [d, match s { true => 1.0, false => 0.0 }, 0.0],
  ContentSolid(c) => [float(c), 0.0, 0.0]
}
```

**Before (invalid):**
```go
s := _adt.ContentStarfield.Scroll  // bool
if s.(bool) {  // ERROR: bool is not an interface
    return float64(1.0)
} else {
    return float64(0.0)
}
```

**After (valid):**
```go
s := _adt.ContentStarfield.Scroll  // bool
if s {  // Correct: no type assertion needed
    return float64(1.0)
} else {
    return float64(0.0)
}
```

### Example 2: Bool match on interface{} (should still add assertion)

When the scrutinee IS an interface{}, the type assertion should still be added:

```go
var x interface{} = true
if x.(bool) {  // Correct: x is interface{}, needs assertion
    return 1.0
}
```

## Success Criteria

- [ ] `go build` succeeds on generated code with nested bool match
- [ ] Test case for ADT field bool match passes
- [ ] Test case for interface{} bool match still adds assertion
- [ ] All existing codegen tests pass
- [ ] Full test suite passes (`make test`)
- [ ] stapledons_voyage project compiles successfully

## Testing Strategy

**Unit tests:**
- Test `generateFlatBoolMatchChain()` with typed bool scrutinee (no assertion)
- Test `generateFlatBoolMatchChain()` with interface{} scrutinee (with assertion)
- Test nested match inside ADT arm

**Integration tests:**
- Compile generated Go code with `go build`
- Run generated code to verify correct runtime behavior

**Manual testing:**
- Compile stapledons_voyage example that triggered the bug
- Verify the generated viewport.go compiles

## Non-Goals

**Not in this feature:**
- Refactoring the entire bool match chain detection - Only fixing the type assertion issue
- Optimizing bool match code generation - Separate enhancement

## Timeline

**Implementation**: ~2 hours total
- 1 hour: Fix `generateFlatBoolMatchChain()`
- 1 hour: Add tests, verify all passing

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fix breaks interface{} case | Medium | Add explicit test for interface{} scrutinee |
| Other bool match paths affected | Low | Search for other `.(bool)` emissions in codegen |

## References

- Bug reports: Messages from stapledons_voyage (msg_20251212_201122 and msg_20251212_200703)
- Similar fix pattern: `codegen_match.go:88` (M-DX25.6 type-assert check)
- Related: M-DX25.5, M-DX25.6, M-DX25.7 type assertion improvements

## Future Work

- Consider adding a lint check for unconditional type assertions in codegen
- Audit other type assertion sites for similar issues

---

**Document created**: 2025-12-12
**Last updated**: 2025-12-12
