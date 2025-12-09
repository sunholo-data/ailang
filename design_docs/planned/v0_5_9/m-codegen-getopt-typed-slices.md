# M-CODEGEN: Array Runtime Typed Slice Support

**Status**: Implemented
**Target**: v0.5.9
**Priority**: P0 (High) - Blocks Array operations in generated Go code
**Estimated**: 1 hour
**Dependencies**: None
**Completed**: 2025-12-09

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No syntax change - runtime fix |
| Preserve Semantic Clarity | + | +1 | Array indexing semantics now work correctly |
| Increase Determinism | + | +1 | Consistent behavior across slice types |
| Lower Token Cost | 0 | 0 | No change to generated code size |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 -> Move forward

## Problem Statement

Multiple Array runtime functions in generated Go code only handle `[]interface{}` slices. However, Go codegen creates typed slices like `[]int64` for `Array[int]` types. When these functions receive typed slices, the type assertions fail silently.

**Affected Functions:**
| Function | Broken Behavior | Impact |
|----------|-----------------|--------|
| `GetOpt` | Returns `None` for all indices | Cannot access Array elements |
| `Get` | Panics "not an array" | Crashes on valid arrays |
| `Length` | Returns `0` | Wrong array length |
| `FromList` | Returns `[]interface{}{}` | Loses all data |
| `ToList` | Returns `[]interface{}{}` | Loses all data |

**Current State:**
- Array functions use `arr.([]interface{})` type assertion - fails for typed slices
- `ListLen`, `ListHead`, `ListTail` already use reflection for typed slices
- Inconsistent behavior: List operations work but Array operations fail

**Impact:**
- **stapledons_voyage project**: Bridge floor rendering generates 0 tiles instead of 192 because `A.getOpt` always returns `None`
- **All AILANG projects**: Cannot use Array module with typed slices
- **Severity**: Complete breakage of Array operations - high priority fix

**Bug Report Source:** stapledons_voyage via ailang messages (2025-12-09)

## Goals

**Primary Goal:** Make all Array runtime functions work with typed slices using reflection fallback.

**Success Metrics:**
- `GetOpt(arr, 0)` returns `Some(value)` for typed slices with valid indices
- `Get(arr, 0)` returns element for typed slices (instead of panicking)
- `Length(arr)` returns correct length for typed slices
- `FromList(xs)` and `ToList(arr)` preserve typed slice data
- stapledons_voyage bridge rendering produces 192 tiles (not 0)

## Solution Design

### Overview

Add reflection-based fallback to `GetOpt` following the same pattern used by `ListLen`, `ListHead`, and `ListTail`. The fix is a simple code generation change in `codegen_runtime_collections.go`.

### Architecture

**Current GetOpt (broken):**
```go
func GetOpt(arr interface{}, idx interface{}) interface{} {
    i := toInt64(idx)
    if i < 0 {
        return makeOptionNone()
    }
    if slice, ok := arr.([]interface{}); ok {  // <-- FAILS for []int64
        if i >= int64(len(slice)) {
            return makeOptionNone()
        }
        return makeOptionSome(slice[i])
    }
    return makeOptionNone()  // <-- Always returns None for typed slices!
}
```

**Fixed GetOpt (with reflection):**
```go
func GetOpt(arr interface{}, idx interface{}) interface{} {
    i := toInt64(idx)
    if i < 0 {
        return makeOptionNone()
    }
    // Fast path for []interface{}
    if slice, ok := arr.([]interface{}); ok {
        if i >= int64(len(slice)) {
            return makeOptionNone()
        }
        return makeOptionSome(slice[i])
    }
    // Reflection path for typed slices (e.g., []int64, []*Tile)
    v := reflect.ValueOf(arr)
    if v.Kind() == reflect.Slice {
        if i >= int64(v.Len()) {
            return makeOptionNone()
        }
        return makeOptionSome(v.Index(int(i)).Interface())
    }
    return makeOptionNone()
}
```

### Implementation Plan

**Phase 1: Fix GetOpt** (~30 minutes)
- [ ] Update `genGetOpt()` in `internal/gen/golang/codegen_runtime_collections.go`
- [ ] Add reflection fallback after `[]interface{}` fast path
- [ ] Follow exact pattern from `ListHead`/`ListLen`

**Phase 2: Testing** (~30 minutes)
- [ ] Add unit test for `GetOpt` with `[]int64`
- [ ] Add unit test for `GetOpt` with `[]*struct{}`
- [ ] Verify existing tests still pass
- [ ] Test with stapledons_voyage bridge rendering

### Files to Modify/Create

**Modified files:**
- `internal/gen/golang/codegen_runtime_collections.go` - Add reflection fallback (~10 LOC)

**Test files:**
- `internal/gen/golang/codegen_runtime_collections_test.go` - Add typed slice tests (~30 LOC)

## Examples

### Example 1: Array of Integers

**Before (broken):**
```ailang
let arr: Array[int] = [1, 2, 3]
let first = A.getOpt(arr, 0)  -- Returns None (wrong!)
```

Generated Go:
```go
arr := []int64{1, 2, 3}
first := GetOpt(arr, 0)  // arr.([]interface{}) fails, returns None
```

**After (fixed):**
```ailang
let arr: Array[int] = [1, 2, 3]
let first = A.getOpt(arr, 0)  -- Returns Some(1) (correct!)
```

Generated Go:
```go
arr := []int64{1, 2, 3}
first := GetOpt(arr, 0)  // reflection fallback returns Some(1)
```

### Example 2: Array of ADT Values

**Before (broken):**
```ailang
type Tile = Tile(int, int)
let tiles: Array[Tile] = [Tile(0, 0), Tile(1, 1)]
let t = A.getOpt(tiles, 0)  -- Returns None (wrong!)
```

**After (fixed):**
```ailang
type Tile = Tile(int, int)
let tiles: Array[Tile] = [Tile(0, 0), Tile(1, 1)]
let t = A.getOpt(tiles, 0)  -- Returns Some(Tile(0, 0)) (correct!)
```

## Success Criteria

- [ ] `GetOpt([]int64{1,2,3}, 0)` returns `Some(1)`
- [ ] `GetOpt([]int64{1,2,3}, 5)` returns `None` (out of bounds)
- [ ] `GetOpt([]int64{1,2,3}, -1)` returns `None` (negative)
- [ ] `GetOpt([]*Struct{...}, 0)` returns `Some(*Struct{...})`
- [ ] All existing Go codegen tests pass
- [ ] stapledons_voyage bridge rendering fixed (192 tiles)

## Testing Strategy

**Unit tests:**
- Test `GetOpt` with `[]int64` (common case)
- Test `GetOpt` with `[]*struct{}` (pointer slices)
- Test `GetOpt` with `[]interface{}` (regression: fast path still works)
- Test boundary conditions (negative, out-of-bounds, empty slice)

**Integration tests:**
- Compile and run AILANG program using `A.getOpt` on typed arrays
- Verify correct values are returned

**Manual testing:**
- Recompile stapledons_voyage bridge.ail
- Verify 192 tiles generated (not 0)

## Non-Goals

**Not in this feature:**
- Optimizing reflection path performance - Keep it simple for now
- Adding new Array builtins - Focus on fixing existing broken functionality
- Changing type inference for Arrays - This is a runtime-only fix

## Timeline

**Day 1** (1 hour):
- Phase 1: Fix GetOpt implementation
- Phase 2: Add tests and verify
- Send completion notification to stapledons_voyage

**Total: ~1 hour**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Reflection slower than direct access | Low | Keep []interface{} fast path; reflection only for typed slices |
| Break existing code | Low | Fast path unchanged; reflection is fallback only |

## References

- Bug report: `ailang messages read` (msg from stapledons_voyage, 2025-12-09)
- Pattern to follow: `ListLen`, `ListHead`, `ListTail` in same file
- File: `internal/gen/golang/codegen_runtime_collections.go:331`

## Future Work

- Consider type-specific fast paths for common types (`[]int64`, `[]float64`, `[]string`)
- Profile reflection overhead in generated code if performance becomes an issue

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09
