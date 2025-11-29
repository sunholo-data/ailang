# M-BUG-COMPARISON-PANIC: Fix Type-Unsafe Comparison Builtins

**Status**: Planned
**Priority**: P0 (Safety)
**Estimated LOC**: ~80
**Target Version**: v0.4.9

## Problem Statement

The comparison builtins (`registerCmpWithMeta`, `registerCmpStringWithMeta`) in `internal/builtins/math.go` panic at runtime when passed values of unexpected types. This is a type safety bug - the type checker should prevent this, but valid-looking code crashes at runtime.

### Error Examples

```
panic: interface conversion: eval.Value is *eval.IntValue, not *eval.StringValue
  at internal/builtins/math.go:336

panic: interface conversion: eval.Value is *eval.StringValue, not *eval.IntValue
  at internal/builtins/math.go:264
```

### Impact

- 9 runtime failures in v0.4.8 eval baseline
- 100% failure on `symbolic_diff` benchmark due to this bug
- User code that passes type checking crashes at runtime

## Root Cause Analysis

The comparison builtins use unsafe type assertions without checking:

```go
// math.go:264 - registerCmpWithMeta
func(args []Value) (Value, error) {
    v1 := args[0].(*IntValue)  // PANIC if not IntValue!
    v2 := args[1].(*IntValue)
    // ...
}
```

The type checker may allow polymorphic code that eventually gets specialized to mismatched types, or the dictionary linking may wire up the wrong implementation.

## Proposed Solution

### Option A: Safe Type Assertions (Recommended)

Add runtime type checking with clear error messages:

```go
func(args []Value) (Value, error) {
    v1, ok := args[0].(*IntValue)
    if !ok {
        return nil, fmt.Errorf("comparison error: expected int, got %T", args[0])
    }
    v2, ok := args[1].(*IntValue)
    if !ok {
        return nil, fmt.Errorf("comparison error: expected int, got %T", args[1])
    }
    // ...
}
```

### Option B: Fix Type Checker

Investigate why the type checker allows these mismatched comparisons and fix at the source.

## Implementation Plan

1. **Audit all builtins** for unsafe type assertions (grep for `args[0].(*`)
2. **Add safe assertions** with descriptive error messages
3. **Add regression test** using `symbolic_diff` benchmark code
4. **Verify type checker** is correctly constraining comparison types

## Acceptance Criteria

- [ ] No panics in comparison builtins (safe assertions)
- [ ] Clear error messages when types mismatch
- [ ] `symbolic_diff` benchmark no longer crashes
- [ ] Regression test added
- [ ] All existing tests pass

## Files to Modify

- `internal/builtins/math.go` (~40 LOC changes)
- `tests/builtins/comparison_test.go` (~40 LOC new test)

## References

- v0.4.8 eval analysis showing 9 panics
- `internal/builtins/math.go:264,336`
