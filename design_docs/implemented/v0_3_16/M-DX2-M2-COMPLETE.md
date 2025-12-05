# M-DX2 Milestone 2: Core IR Helpers - COMPLETE ✅

**Date**: 2025-10-21
**Sprint**: M-DX2 (Operator Development Experience Improvements)
**Status**: ✅ COMPLETE
**Estimated Time**: 1 hour
**Actual Time**: ~35 minutes

## Summary

Successfully implemented Core IR helper functions for ANF traversal with cycle detection and 100% test coverage. These helpers provide a clean API for future operator implementations and debug utilities that need to inspect ANF values.

## Deliverables

### Files Created

**`internal/core/helpers.go`** (~110 LOC)
- `ResolveValue(expr, bindings)` - Follow variable bindings to ultimate value
- `resolveValueWithVisited(expr, bindings, visited)` - Internal with cycle detection
- `IsListValue(expr, bindings)` - Check if expr resolves to List
- `IsStringValue(expr, bindings)` - Check if expr resolves to String
- `IsIntValue(expr, bindings)` - Check if expr resolves to Int
- `IsFloatValue(expr, bindings)` - Check if expr resolves to Float
- `IsBoolValue(expr, bindings)` - Check if expr resolves to Bool

**Key Features**:
- **Cycle Detection**: Uses visited set (`map[string]struct{}`) - fail-closed on cycles
- **Pure Functions**: No global state, no logging, ANF-local scope only
- **Zero Allocations** (except visited set): Minimal overhead
- **Clear Docstrings**: When to use / not use (prefer CoreTypeInfo)

**`internal/core/helpers_test.go`** (~310 LOC)
- Direct literal resolution
- Single variable binding
- Chained variables (3-deep)
- Cycle detection (a → b → c → a)
- Unbound variables
- Deep chain (20 links) - mini-fuzz test
- Type helper tests for all 5 types
- Cross-cutting test (all helpers + all types in chains)

## Test Results

**All tests passing**:
```bash
$ go test ./internal/core -v
PASS
ok  	github.com/sunholo/ailang/internal/core	0.196s
```

**Coverage**:
```bash
$ go test ./internal/core -cover
coverage: 71.6% of statements (100% on helpers.go)

$ go tool cover -func=/tmp/core_coverage.out | grep helpers.go
ResolveValue              100.0%
resolveValueWithVisited   100.0%
IsListValue               100.0%
IsStringValue             100.0%
IsIntValue                100.0%
IsFloatValue              100.0%
IsBoolValue               100.0%
```

**Full test suite**: ✅ All packages pass

## Implementation Details

### ResolveValue Design

**Signature**:
```go
func ResolveValue(expr CoreExpr, bindings map[string]CoreExpr) CoreExpr
```

**Algorithm**:
1. Create empty visited set
2. Recursively follow Var → binding
3. Check visited before following (cycle detection)
4. Return non-Var or last resolvable Var (fail-closed)

**Example**:
```go
bindings := map[string]CoreExpr{
    "x": &List{...},
    "y": &Var{Name: "x"},
    "z": &Var{Name: "y"},
}
ResolveValue(&Var{Name: "z"}, bindings)  // → &List{...}
```

**Cycle Handling**:
```go
bindings := map[string]CoreExpr{
    "a": &Var{Name: "b"},
    "b": &Var{Name: "c"},
    "c": &Var{Name: "a"},  // Cycle!
}
ResolveValue(&Var{Name: "a"}, bindings)  // → &Var{Name: "a"} (fail-closed)
```

### Type Helper Design

All type helpers follow the same pattern:
```go
func IsXValue(expr CoreExpr, bindings map[string]CoreExpr) bool {
    resolved := ResolveValue(expr, bindings)
    // Check if resolved is X type
    return /* type check */
}
```

This provides a consistent API and ensures all type checks benefit from cycle detection.

## Metrics

| Metric | Value |
|--------|-------|
| Implementation LOC | ~110 |
| Test LOC | ~310 |
| Test Coverage | 100% (helpers.go) |
| Number of Functions | 7 (1 public + 1 internal + 5 type helpers) |
| Number of Tests | 18 test functions |
| Time Spent | ~35 minutes |
| Test Execution Time | 0.196s |

## Usage Notes

**From the docstrings**:

> Note: Prefer using CoreTypeInfo from type inference for type-guided operations.
> ResolveValue is a fallback for non-typed passes and debug utilities.

These helpers are NOT used in the main operator lowering path (which uses CoreTypeInfo). They are available for:
- **Debug utilities** (M3: `ailang debug ast`)
- **Future passes** that need ANF inspection without types
- **Edge cases** where CoreTypeInfo is unavailable

## Design Decisions

1. **Simplified Signature**: Removed `maxDepth` parameter in favor of cycle detection
   - **Before**: `ResolveValue(expr, bindings, maxDepth int)`
   - **After**: `ResolveValue(expr, bindings)`
   - **Rationale**: Cycle detection is more robust and doesn't require tuning maxDepth

2. **Fail-Closed on Cycles**: Returns last resolvable var instead of error
   - **Rationale**: Pure function (no error return), defensive behavior

3. **ANF-Local Scope**: Only operates on provided bindings map
   - **Rationale**: No cross-module resolution, keeps helpers simple

4. **No Logging**: Pure functions with no side effects
   - **Rationale**: Clean, testable, predictable

## Integration

**No changes to existing code required**. These are pure additions that don't affect the compilation pipeline. The helpers are available for future use in:
- M3 (Debug CLI)
- Future optimizations
- Custom analysis passes

## Next Steps

**Milestone 2 is complete!** Ready to proceed to:
- **M3**: Debug CLI (~2.5-3h) - Implement `ailang debug ast --show-types`
- **M4**: Better Runtime Errors (~1h) - Structured error messages
- **M5**: Documentation (~1.5-2h) - ANF guide and operator checklist

Total progress: **M1 ✅ + M2 ✅** (2/5 milestones, ~3.5h of ~8h total)
