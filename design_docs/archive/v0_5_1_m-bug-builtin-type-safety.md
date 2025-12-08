# M-BUG-BUILTIN-TYPE-SAFETY: Fix Unsafe Type Assertions in Builtins

**Status**: Planned
**Priority**: P0 (Safety)
**Estimated LOC**: ~200
**Target Version**: v0.5.0
**Originally**: m-bug-comparison-builtin-panic (v0.4.9)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No syntax changes |
| Preserve Semantic Clarity | + | +1 | Clear error messages vs panics |
| Increase Determinism | + | +2 | Errors instead of panics = predictable |
| Lower Token Cost | 0 | 0 | No change to token cost |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

Multiple builtin functions use **unsafe type assertions** that panic at runtime when passed values of unexpected types. This affects all builtins in math, string, list, JSON, clock, and I/O modules.

### Error Examples

```
panic: interface conversion: eval.Value is *eval.IntValue, not *eval.StringValue
  at internal/builtins/math.go:336

panic: interface conversion: eval.Value is *eval.StringValue, not *eval.IntValue
  at internal/builtins/math.go:264
```

### Scope Analysis (2025-12-01)

**63 unsafe type assertions across 7 files:**

| File | Count | Affected Builtins |
|------|-------|-------------------|
| math.go | 25 | Comparison, arithmetic, negation |
| string.go | 19 | String ops, parsing |
| json_encode.go | 9 | JSON encoding |
| list.go | 3 | List operations |
| io.go | 3 | I/O operations |
| clock.go | 2 | Time operations |
| env.go | 1 | Environment access |
| json_decode.go | 1 | JSON decoding |

### Impact

- Runtime panics instead of errors
- User code crashes despite passing type checking
- Poor debugging experience (panic traces vs error messages)
- Affects eval benchmarks (9+ failures in v0.4.8)

## Root Cause Analysis

All affected builtins use direct type assertions without checking:

```go
// UNSAFE - panics if type mismatch
a := args[0].(*eval.IntValue)
b := args[1].(*eval.IntValue)

// SAFE - returns error on mismatch
a, ok := args[0].(*eval.IntValue)
if !ok {
    return nil, fmt.Errorf("expected int, got %T", args[0])
}
```

The type checker may allow polymorphic code that gets specialized to mismatched types, or dictionary linking may wire up wrong implementations.

## Solution Design

### Helper Function Approach

Create type-safe assertion helpers to reduce boilerplate:

```go
// internal/builtins/helpers.go

func AsInt(v eval.Value) (*eval.IntValue, error) {
    if iv, ok := v.(*eval.IntValue); ok {
        return iv, nil
    }
    return nil, fmt.Errorf("expected int, got %T", v)
}

func AsFloat(v eval.Value) (*eval.FloatValue, error) {
    if fv, ok := v.(*eval.FloatValue); ok {
        return fv, nil
    }
    return nil, fmt.Errorf("expected float, got %T", v)
}

func AsString(v eval.Value) (*eval.StringValue, error) {
    if sv, ok := v.(*eval.StringValue); ok {
        return sv, nil
    }
    return nil, fmt.Errorf("expected string, got %T", v)
}

func AsBool(v eval.Value) (*eval.BoolValue, error) {
    if bv, ok := v.(*eval.BoolValue); ok {
        return bv, nil
    }
    return nil, fmt.Errorf("expected bool, got %T", v)
}

func AsList(v eval.Value) (*eval.ListValue, error) {
    if lv, ok := v.(*eval.ListValue); ok {
        return lv, nil
    }
    return nil, fmt.Errorf("expected list, got %T", v)
}
```

### Usage Pattern

```go
// Before (unsafe)
func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    a := args[0].(*eval.IntValue)  // PANIC!
    b := args[1].(*eval.IntValue)
    return &eval.BoolValue{Value: a.Value < b.Value}, nil
}

// After (safe)
func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    a, err := AsInt(args[0])
    if err != nil {
        return nil, err
    }
    b, err := AsInt(args[1])
    if err != nil {
        return nil, err
    }
    return &eval.BoolValue{Value: a.Value < b.Value}, nil
}
```

## Implementation Plan

### Phase 1: Create Helpers (~1 hour)
- [ ] Create `internal/builtins/helpers.go` with safe assertion helpers
- [ ] Add unit tests for helpers
- [ ] Helpers for: Int, Float, String, Bool, List, Record, Tuple

### Phase 2: Fix math.go (~2 hours)
- [ ] Replace 25 unsafe assertions with helper calls
- [ ] Test all comparison and arithmetic builtins
- [ ] Run full test suite

### Phase 3: Fix string.go (~2 hours)
- [ ] Replace 19 unsafe assertions
- [ ] Test string operations
- [ ] Run full test suite

### Phase 4: Fix remaining files (~2 hours)
- [ ] json_encode.go (9 assertions)
- [ ] list.go (3 assertions)
- [ ] io.go (3 assertions)
- [ ] clock.go (2 assertions)
- [ ] env.go (1 assertion)
- [ ] json_decode.go (1 assertion)

### Phase 5: Testing & Validation (~1 hour)
- [ ] Run eval baseline to verify reduced panics
- [ ] Add regression tests for type mismatch scenarios
- [ ] Update CHANGELOG.md

## Files to Modify

**New file:**
- `internal/builtins/helpers.go` (~50 LOC)
- `internal/builtins/helpers_test.go` (~30 LOC)

**Modified files:**
- `internal/builtins/math.go` (~50 LOC changes)
- `internal/builtins/string.go` (~40 LOC changes)
- `internal/builtins/json_encode.go` (~20 LOC changes)
- `internal/builtins/list.go` (~10 LOC changes)
- `internal/builtins/io.go` (~10 LOC changes)
- `internal/builtins/clock.go` (~5 LOC changes)
- `internal/builtins/env.go` (~5 LOC changes)
- `internal/builtins/json_decode.go` (~5 LOC changes)

**Total: ~200 LOC**

## Acceptance Criteria

- [ ] All 63 unsafe type assertions replaced with safe helpers
- [ ] No panics from type mismatches in builtins
- [ ] Clear error messages: "expected int, got *eval.StringValue"
- [ ] All existing tests pass
- [ ] New tests for type mismatch scenarios
- [ ] Eval baseline shows reduced panic failures

## Success Metrics

- 0 panic failures from type assertions in eval runs
- Clear, actionable error messages for type mismatches
- No performance regression (helper function overhead minimal)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Helper function overhead | Low | Inline critical paths if needed |
| Breaking tests | Medium | Run tests after each file change |
| Missing assertions | Medium | Grep audit before completion |

## References

- Original design doc: m-bug-comparison-builtin-panic (v0.4.9)
- v0.4.8 eval analysis showing panic failures
- Affected files: math.go, string.go, json_encode.go, list.go, io.go, clock.go, env.go, json_decode.go

---

**Document created**: 2025-11-29 (as m-bug-comparison-builtin-panic)
**Last updated**: 2025-12-01 (expanded scope, moved to v0.5.0)
