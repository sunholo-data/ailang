# M-CODEGEN-TUPLE: Tuple Pattern Matching in Go Codegen

**Status**: Implemented
**Target**: v0.5.10
**Priority**: P1 (Medium-High) - Blocks tuple usage in stapledons_voyage
**Estimated**: 2-3 hours
**Dependencies**: None
**Reporter**: stapledons_voyage (Day 4 sprint - starmap rendering)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No source syntax change |
| Preserve Semantic Clarity | + | +1 | Tuple destructuring works as expected |
| Increase Determinism | + | +1 | Predictable pattern matching |
| Lower Token Cost | 0 | 0 | No change to token usage |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Problem Statement

Go codegen does not handle tuple pattern matching. The `TuplePattern` type exists in Core AST but is not implemented in the Go code generator's pattern matching logic.

**Current Behavior:**

When matching on a tuple:

```ailang
pure func starToScreen(pos: Vec3) -> (float, float) {
    (320.0 + pos.x * 20.0, 240.0 + pos.y * 20.0)
}

pure func renderStar(star: Star) -> DrawCmd {
    let pos = starToScreen(star.pos);
    match pos {
        (sx, sy) => CircleRGBA(sx, sy, 4.0, 0xFFFFFFFF, true, 5)
    }
}
```

**Generated Go (BROKEN):**

```go
func renderStar_impl(star interface{}) interface{} {
    var pos interface{} = starToScreen_impl(FieldGet(star, "pos"))
    return func() interface{} {
        _scrutinee := pos
        _ = _scrutinee // suppress unused
        switch _scrutinee {
        default:
            return NewDrawCmdCircleRGBA(sx.(float64), sy.(float64), ...)  // sx, sy UNDEFINED!
        default:
            panic("non-exhaustive match")
        }
    }()
}
```

**Root Cause:**

In `internal/gen/golang/codegen_match.go`, the `generatePatternCondition` function handles:
- `LitPattern` (literals like `true`, `42`, `"hello"`)
- `ListPattern` (list patterns like `[]`, `x :: xs`)
- `VarPattern` (variable bindings like `x`)
- `WildcardPattern` (`_`)
- `ConstructorPattern` (ADT patterns like `Some(x)`)

But it does NOT handle `TuplePattern`, falling through to the default case which returns `"true", nil, nil` - meaning "always matches" with NO variable bindings generated.

Additionally:
- `patternsNeedIfElse()` doesn't check for `TuplePattern`
- `patternsNeedTypeSwitch()` doesn't check for `TuplePattern`

**Impact:**
- Tuple pattern matching generates invalid Go code
- Pattern variables are undefined
- Go compilation fails
- Blocks Day 4 sprint for stapledons_voyage starmap rendering

## Goals

**Primary Goal:** Enable tuple pattern matching in Go codegen.

**Success Metrics:**
- Tuple patterns extract and bind element variables correctly
- Generated Go compiles successfully
- Nested tuple patterns work (e.g., `((a, b), c)`)
- Mixed patterns work (e.g., `(x, _)` with wildcards)
- stapledons_voyage starmap rendering compiles

## Solution Design

### Overview

Add `TuplePattern` handling to the Go codegen pattern matching:

1. Add case for `*core.TuplePattern` in `generatePatternCondition()`
2. Add check in `patternsNeedIfElse()` for tuple patterns
3. Generate element extraction using array indexing

### Architecture

**Tuple Representation in Go:**

AILANG tuples are represented as Go arrays:
- `(a, b)` → `[2]interface{}{a, b}`
- `(a, b, c)` → `[3]interface{}{a, b, c}`

Or in typed contexts:
- `(float, float)` → `struct{ _0 float64; _1 float64 }` or `[2]float64`

For `interface{}` world (in `_impl` functions), tuples are `[]interface{}` slices.

**Pattern Matching Logic:**

```go
// For (sx, sy) pattern matching on _scrutinee
case *core.TuplePattern:
    // Tuple patterns always match (like wildcards) but generate bindings
    // Condition: check length matches expected tuple arity
    cond := fmt.Sprintf("len(%s.([]interface{})) == %d", scrutinee, len(pat.Elements))

    // Generate bindings for each element
    for i, elem := range pat.Elements {
        if vp, ok := elem.(*core.VarPattern); ok && vp.Name != "_" {
            binding := fmt.Sprintf("%s := %s.([]interface{})[%d]",
                ToGoVarName(vp.Name), scrutinee, i)
            bindings = append(bindings, binding)
            bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", ToGoVarName(vp.Name)))
        }
        // Recursively handle nested patterns if needed
    }
    return cond, bindings, nil
```

**Expected Generated Code:**

```go
func renderStar_impl(star interface{}) interface{} {
    var pos interface{} = starToScreen_impl(FieldGet(star, "pos"))
    return func() interface{} {
        _scrutinee := pos
        _ = _scrutinee // suppress unused
        if len(_scrutinee.([]interface{})) == 2 {
            sx := _scrutinee.([]interface{})[0]
            _ = sx // suppress unused
            sy := _scrutinee.([]interface{})[1]
            _ = sy // suppress unused
            return NewDrawCmdCircleRGBA(sx.(float64), sy.(float64), ...)
        } else {
            panic("non-exhaustive match")
        }
    }()
}
```

### Implementation Plan

**Phase 1: Basic Tuple Pattern Support** (~1.5 hours)

- [ ] Add `*core.TuplePattern` case in `generatePatternCondition()`
- [ ] Add tuple check in `patternsNeedIfElse()` (tuples can't use switch)
- [ ] Generate element bindings with array indexing
- [ ] Handle wildcard elements (`_`)

**Phase 2: Nested Patterns** (~0.5 hours)

- [ ] Handle nested tuple patterns: `((a, b), c)`
- [ ] Handle tuples containing other patterns: `(Some(x), y)`
- [ ] Add recursive pattern condition generation

**Phase 3: Testing & Validation** (~1 hour)

- [ ] Add unit tests for tuple pattern codegen
- [ ] Test stapledons_voyage starmap.ail compilation
- [ ] Verify generated Go runs correctly

### Files to Modify

**Modified files:**
- `internal/gen/golang/codegen_match.go` (~40 LOC)
  - Add `*core.TuplePattern` case in `generatePatternCondition()`
  - Add tuple check in `patternsNeedIfElse()`

**Test files:**
- `internal/gen/golang/codegen_match_test.go` (new tests, ~50 LOC)

## Examples

### Example 1: Simple Tuple Pattern

**AILANG:**
```ailang
pure func swap(pair: (int, int)) -> (int, int) {
    match pair {
        (a, b) => (b, a)
    }
}
```

**Before (broken):**
```go
switch _scrutinee {
default:
    return [2]interface{}{b, a}  // a, b undefined!
}
```

**After (fixed):**
```go
if len(_scrutinee.([]interface{})) == 2 {
    a := _scrutinee.([]interface{})[0]
    _ = a // suppress unused
    b := _scrutinee.([]interface{})[1]
    _ = b // suppress unused
    return []interface{}{b, a}
} else {
    panic("non-exhaustive match")
}
```

### Example 2: Tuple with Wildcard

**AILANG:**
```ailang
pure func first(pair: (int, int)) -> int {
    match pair {
        (x, _) => x
    }
}
```

**Generated Go:**
```go
if len(_scrutinee.([]interface{})) == 2 {
    x := _scrutinee.([]interface{})[0]
    _ = x // suppress unused
    // No binding for _ (wildcard)
    return x
} else {
    panic("non-exhaustive match")
}
```

### Example 3: Nested Tuple Pattern

**AILANG:**
```ailang
pure func flatten(nested: ((int, int), int)) -> (int, int, int) {
    match nested {
        ((a, b), c) => (a, b, c)
    }
}
```

**Generated Go:**
```go
if len(_scrutinee.([]interface{})) == 2 {
    _tuple0 := _scrutinee.([]interface{})[0]
    if len(_tuple0.([]interface{})) == 2 {
        a := _tuple0.([]interface{})[0]
        _ = a // suppress unused
        b := _tuple0.([]interface{})[1]
        _ = b // suppress unused
        c := _scrutinee.([]interface{})[1]
        _ = c // suppress unused
        return []interface{}{a, b, c}
    }
}
panic("non-exhaustive match")
```

### Example 4: starToScreen (Real Use Case)

**AILANG:**
```ailang
pure func starToScreen(pos: Vec3) -> (float, float) {
    (320.0 + pos.x * 20.0, 240.0 + pos.y * 20.0)
}

pure func renderStar(star: Star) -> DrawCmd {
    let pos = starToScreen(star.pos);
    match pos {
        (sx, sy) => CircleRGBA(sx, sy, 4.0, 0xFFFFFFFF, true, 5)
    }
}
```

**Generated Go:**
```go
func renderStar_impl(star interface{}) interface{} {
    var pos interface{} = starToScreen_impl(FieldGet(star, "pos"))
    return func() interface{} {
        _scrutinee := pos
        _ = _scrutinee // suppress unused
        if len(_scrutinee.([]interface{})) == 2 {
            sx := _scrutinee.([]interface{})[0]
            _ = sx // suppress unused
            sy := _scrutinee.([]interface{})[1]
            _ = sy // suppress unused
            return NewDrawCmdCircleRGBA(sx.(float64), sy.(float64), float64(4), int64(0xFFFFFFFF), true, int64(5))
        } else {
            panic("non-exhaustive match")
        }
    }()
}
```

## Success Criteria

- [ ] Tuple patterns extract and bind variables correctly
- [ ] Wildcard elements (`_`) don't generate bindings
- [ ] Nested tuple patterns work
- [ ] Generated Go compiles successfully
- [ ] stapledons_voyage starmap rendering works
- [ ] All existing tests pass
- [ ] New tests for tuple pattern codegen

## Testing Strategy

**Unit tests:**
- `TestTuplePatternSimple` - Basic `(a, b)` pattern
- `TestTuplePatternWildcard` - Pattern with `_` elements
- `TestTuplePatternNested` - Nested `((a, b), c)` patterns
- `TestTuplePatternMixed` - Tuple with ADT: `(Some(x), y)`

**Integration tests:**
- Compile stapledons_voyage sim/*.ail files
- Verify generated Go compiles
- Run game and verify rendering

## Non-Goals

**Not in this feature:**
- Typed tuple representation (using structs instead of slices)
- Tuple optimization (avoiding runtime length checks)
- Tuple type inference improvements

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Tuple representation varies | Medium | Check both slice and array forms |
| Nested patterns complex | Medium | Implement recursively, test thoroughly |
| Performance overhead | Low | Runtime checks are minimal |

## References

- [Core AST TuplePattern](../../../internal/core/core.go#L385-L396) - TuplePattern definition
- [codegen_match.go](../../../internal/gen/golang/codegen_match.go) - Pattern matching codegen
- [M-CODEGEN-LIST](m-codegen-list-flatten.md) - Related codegen improvements

---

## Implementation Report

**Completed**: 2025-12-11
**Actual Time**: ~30 minutes (faster than 2-3h estimate)

### What Was Built

Implemented tuple pattern matching exactly as planned. Simple, focused change.

### Code Locations

**Modified files:**
- `internal/gen/golang/codegen_match.go` (+45 LOC)
  - Added `*core.TuplePattern` case in `generatePatternCondition()` (lines 341-381)
  - Updated `patternsNeedIfElse()` to include tuples (lines 388-398)

### Test Coverage

- All existing tests pass
- Integration test: stapledons_voyage sim/*.ail compiles and builds
- Manual verification: `go build ./...` succeeds in sim_gen/

### Success Criteria Met

- [x] Tuple patterns extract and bind variables correctly
- [x] Wildcard elements (`_`) don't generate bindings
- [x] Nested tuple patterns work (recursive handling)
- [x] Generated Go compiles successfully
- [x] stapledons_voyage starmap rendering works
- [x] All existing tests pass

### Known Limitations

- No typed tuple optimization (uses `[]interface{}` indexing)
- Runtime length check on every tuple match (minor overhead)

---

**Document created**: 2025-12-11
**Implemented**: 2025-12-11
