# M-CODEGEN-TYPED-SLICES: Runtime List Functions for Typed Slices

**Status**: Implemented
**Priority**: High
**Version**: v0.5.7
**Reported by**: stapledons_voyage project
**Implemented**: 2025-12-05

## Problem Statement

The generated runtime functions `ListLen`, `ListHead`, and `ListTail` only handle `[]interface{}` but codegen produces typed slices like `[]*Tile`. This causes list pattern matching to fail silently:

- `ListLen` returns `0` for typed slices (should return actual length)
- `ListHead` returns `nil` for typed slices (should return first element)
- `ListTail` returns `[]interface{}{}` for typed slices (should return tail)

### Current Implementation

```go
// ListLen - only handles []interface{}
func ListLen(list interface{}) int {
    if l, ok := list.([]interface{}); ok {
        return len(l)
    }
    return 0  // BUG: Returns 0 for []*Tile, []int, etc.
}

// ListHead - only handles []interface{}
func ListHead(list interface{}) interface{} {
    if l, ok := list.([]interface{}); ok && len(l) > 0 {
        return l[0]
    }
    return nil  // BUG: Returns nil for typed slices
}

// ListTail - only handles []interface{}
func ListTail(list interface{}) interface{} {
    if l, ok := list.([]interface{}); ok && len(l) > 0 {
        return l[1:]
    }
    return []interface{}{}  // BUG: Returns empty for typed slices
}
```

### Impact

Pattern matching on lists fails silently when lists contain typed values:

```ailang
-- This pattern match fails because ListLen returns 0 for typed slices
match tiles with
| [] -> "empty"
| [x] -> "single"
| [x, y, ...rest] -> "multiple"
```

## Solution

Use `reflect.ValueOf` to check `Kind() == reflect.Slice` and use reflection methods for typed slices, with fast path for `[]interface{}`.

### Proposed Implementation

```go
import "reflect"

func ListLen(list interface{}) int {
    // Fast path for []interface{} (most common case)
    if l, ok := list.([]interface{}); ok {
        return len(l)
    }
    // Reflection path for typed slices
    v := reflect.ValueOf(list)
    if v.Kind() == reflect.Slice {
        return v.Len()
    }
    return 0
}

func ListHead(list interface{}) interface{} {
    // Fast path for []interface{}
    if l, ok := list.([]interface{}); ok && len(l) > 0 {
        return l[0]
    }
    // Reflection path for typed slices
    v := reflect.ValueOf(list)
    if v.Kind() == reflect.Slice && v.Len() > 0 {
        return v.Index(0).Interface()
    }
    return nil
}

func ListTail(list interface{}) interface{} {
    // Fast path for []interface{}
    if l, ok := list.([]interface{}); ok && len(l) > 0 {
        return l[1:]
    }
    // Reflection path for typed slices
    v := reflect.ValueOf(list)
    if v.Kind() == reflect.Slice && v.Len() > 0 {
        return v.Slice(1, v.Len()).Interface()
    }
    return []interface{}{}
}
```

## Implementation Details

### File Changes

1. **`internal/gen/golang/codegen_runtime.go`** (~30 lines changed)
   - Add `import "reflect"` to runtime imports
   - Update `ListLen`, `ListHead`, `ListTail` to use reflection fallback

### Runtime Import Update

The generated code needs `reflect` imported. Check `emitImports()` to ensure `reflect` is included.

## Testing Strategy

1. **Unit tests**: Add tests in `internal/gen/golang/codegen_test.go` for:
   - `[]interface{}` (existing behavior - fast path)
   - `[]int` (typed slice - numeric)
   - `[]*SomeStruct` (typed slice - pointers)
   - Empty slices of each type
   - Single-element slices
   - Multi-element slices

2. **Integration test**: Create `examples/list_typed_pattern.ail`:
   ```ailang
   type Tile = { x: int, y: int }

   let tiles: [Tile] = [{ x: 1, y: 2 }, { x: 3, y: 4 }]

   match tiles with
   | [] -> "empty"
   | [t] -> show(t.x)
   | [t1, t2, ...rest] -> show(t1.x) ++ " and " ++ show(t2.x)
   ```

## Performance Considerations

- Fast path preserved for `[]interface{}` (no reflection overhead)
- Reflection only used when type assertion fails
- Benchmarking recommended to ensure <10% overhead for typed slices

## Acceptance Criteria

- [x] `ListLen` returns correct length for typed slices
- [x] `ListHead` returns first element for typed slices
- [x] `ListTail` returns tail slice for typed slices
- [x] Existing `[]interface{}` behavior unchanged (fast path)
- [x] All tests pass
- [ ] stapledons_voyage list pattern matching works (to be verified)

## Estimated Effort

- Implementation: 1 hour
- Testing: 30 minutes
- Total: ~1.5 hours
