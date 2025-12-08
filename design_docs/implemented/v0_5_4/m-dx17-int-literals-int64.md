# M-DX17: Integer Literals Generated as int Instead of int64

**Version**: v0.5.4
**Priority**: High (causes runtime panics)
**Estimated Effort**: 30 minutes
**Status**: Planned

## Problem Statement

Integer literals in let bindings are generated as Go's default `int` type instead of `int64`, causing runtime panics when type-asserting to `int64`.

**Source**: Bug report from `stapledons_voyage` project (msg_20251203_175826_71676c9d0cb2)

## Current Behavior

AILANG code:
```ailang
let w: int = 8
```

Generated Go (WRONG):
```go
var w interface{} = 8  // Go infers 'int' (not int64)
// ... later ...
return w.(int64)       // PANIC: interface {} is int, not int64
```

## Expected Behavior

Generated Go (CORRECT):
```go
var w interface{} = int64(8)  // Explicit int64
// ... later ...
return w.(int64)              // Works!
```

## Root Cause

In M-DX13.3, we changed let bindings to use `var x interface{} = ...` to allow type assertions. However, integer literals like `8` are typed as `int` by Go, not `int64`.

The `generateLit` function in `codegen_expr.go` generates raw integer literals without explicit type conversion.

## Proposed Solution

Modify `generateLit` to wrap integer literals in `int64()` conversion:

```go
case core.LitInt:
    // M-DX17: Wrap in int64() for consistent interface{} assertions
    g.writef("int64(%d)", e.Value.(int64))
```

Similarly for float literals:
```go
case core.LitFloat:
    g.writef("float64(%g)", e.Value.(float64))
```

## Files to Modify

1. **internal/gen/golang/codegen_expr.go**
   - Update `generateLit()` to wrap numeric literals in type conversions

## Test Cases

1. Let binding with integer literal compiles and runs
2. Type assertion to int64 succeeds
3. Arithmetic operations work correctly with converted literals

## Related Issues

- M-DX13.3: Changed let bindings to `var x interface{}`
- M-DX18: RecordAccess on typed structs (separate but related)
