# M-VERIFY-FLOAT-CMP: Fix Panic on Float Comparisons in Contract Verification

**Status**: Implemented
**Version**: v0.8.1
**Priority**: P0 (High) - Runtime panic crashed both CLI and serve-api
**Actual Effort**: ~30 minutes (two commits)
**GitHub Issue**: #139

## Summary

Fixed a Go-level panic in `--verify-contracts` when contract expressions used float comparisons (e.g., `requires { price >= 0.0 }`). The root cause was bare type assertions (`args[0].(*eval.IntValue)`) in builtin comparison functions that panicked when receiving `*eval.FloatValue` arguments due to incorrect operator specialization during contract elaboration.

The fix replaced all bare type assertions across 8 builtin files with safe `ok`-checked assertions that return descriptive errors instead of panicking. This completed the work that M-BUILTIN-SAFETY (v0.7.0) left unfinished.

## Problem Statement

When `--verify-contracts` was enabled, contract predicate expressions containing float comparisons caused Go-level panics:

```
panic: interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue
  goroutine 1 [running]:
  ...internal/builtins/math_comparison.go:66
```

**Root cause**: `registerCmpWithMeta` (Int comparisons) at line 66 used unsafe bare type assertions:

```go
// BEFORE: Panics if args[0] is *eval.FloatValue
a := args[0].(*eval.IntValue)
b := args[1].(*eval.IntValue)
```

When a contract expression like `price >= 0.0` was elaborated, the operator could resolve to an Int-suffixed builtin (`ge_Int` instead of `ge_Float`) due to missing or incorrect type specialization. The Int builtin then received Float values and panicked.

**Impact:**
- All users of `--verify-contracts` with float-typed parameters were affected
- `serve-api` crashed on any API function with float preconditions
- Panic was NOT caught by AILANG's error handling (complete process crash)
- Int-only contracts (e.g., `requires { qty > 0 }`) worked fine

## Implementation

### Fix Commits

| Commit | Date | Description | Files |
|--------|------|-------------|-------|
| `f9c0ccb7` | 2026-02-20 | Fix comparison panic + serve-api float coercion + A2A text parts | 5 files, +109/-11 |
| `24c7d929` | 2026-02-20 | Fix bare type assertions in math/logic/conversion builtins (systemic) | 6 files, +93/-23 |

**Total: 11 files changed, +202/-34 lines**

### Phase 1: Immediate Fix (commit f9c0ccb7)

Fixed the two comparison functions that caused the reported panic:

**`internal/builtins/math_comparison.go`** - `registerCmpWithMeta` (Int) and `registerCmpBoolWithMeta` (Bool):

```go
// AFTER: Safe ok-checked assertions return errors instead of panicking
a, ok := args[0].(*eval.IntValue)
if !ok {
    return nil, fmt.Errorf("%s: expected IntValue for arg 0, got %T", name, args[0])
}
b, ok := args[1].(*eval.IntValue)
if !ok {
    return nil, fmt.Errorf("%s: expected IntValue for arg 1, got %T", name, args[1])
}
```

This commit also fixed two related serve-api bugs:
1. **Float coercion**: REST/MCP/A2A now use `CallPreserveFloats` so JSON `100.0` stays as `FloatValue`
2. **A2A text parts**: `tasks/send` now extracts args from `type:text` parts by parsing as JSON

### Phase 2: Systemic Hardening (commit 24c7d929)

Following CLAUDE.md Principle 6 (Systemic Fixes - Audit Before Patching), ALL remaining bare type assertions were fixed across 6 additional files:

| File | Functions Fixed |
|------|----------------|
| `internal/builtins/math.go` | `intToInt`, `intIntToInt`, `intIntToIntErr`, `floatFloatToFloat` |
| `internal/builtins/math_arithmetic.go` | `floatDivFloat`, `floatModFloat`, `floatNegFloat` |
| `internal/builtins/math_trig.go` | `registerTrigFunc`, `registerTrigFunc2`, `abs_Int` |
| `internal/builtins/math_logic.go` | `registerLogicOpWithMeta`, `registerLogicUnaryWithMeta` |
| `internal/builtins/math_conversion.go` | `intToFloat`, `floatToInt` |
| `internal/builtins/string_convert.go` | `floatToStr`, `intToStr` |

### Pattern Applied

All bare assertions were replaced with the same safe pattern:

```go
// BEFORE (panics):
a := args[0].(*eval.FloatValue)

// AFTER (returns error):
a, ok := args[0].(*eval.FloatValue)
if !ok {
    return nil, fmt.Errorf("funcName: expected FloatValue for arg 0, got %T", args[0])
}
```

## Systemic Audit Results

### Bare Casts Fixed (15+ functions across 8 files)

| File | Function | Type | Risk Level |
|------|----------|------|-----------|
| `math_comparison.go` | `registerCmpWithMeta` | IntValue | **HIGH** (reported panic) |
| `math_comparison.go` | `registerCmpBoolWithMeta` | BoolValue | Medium |
| `math.go` | `intToInt` | IntValue | Medium |
| `math.go` | `intIntToInt` | IntValue | Medium |
| `math.go` | `intIntToIntErr` | IntValue | Medium |
| `math.go` | `floatFloatToFloat` | FloatValue | Medium |
| `math_arithmetic.go` | `floatDivFloat` | FloatValue | Medium |
| `math_arithmetic.go` | `floatModFloat` | FloatValue | Medium |
| `math_arithmetic.go` | `floatNegFloat` | FloatValue | Low |
| `math_trig.go` | `registerTrigFunc` | FloatValue | Low |
| `math_trig.go` | `registerTrigFunc2` | FloatValue | Low |
| `math_trig.go` | `abs_Int` | IntValue | Low |
| `math_logic.go` | `registerLogicOpWithMeta` | BoolValue | Low |
| `math_logic.go` | `registerLogicUnaryWithMeta` | BoolValue | Low |
| `math_conversion.go` | `intToFloat` | IntValue | Low |
| `math_conversion.go` | `floatToInt` | FloatValue | Low |
| `string_convert.go` | `floatToStr` | FloatValue | Low |
| `string_convert.go` | `intToStr` | IntValue | Low |

### Already Safe (no changes needed)

| File | Function | Pattern |
|------|----------|---------|
| `math_comparison.go` | `registerCmpFloatWithMeta` | `ok` check (fixed in v0.7.0) |
| `math_comparison.go` | `registerCmpStringWithMeta` | `SafeAsString()` |
| `safe_cast.go` | All helpers | `ok` check pattern |
| `sharedindex.go` | Multiple | `ok` check pattern |
| `array.go` | Multiple | `ok` check pattern |
| `numeric.go` | Multiple | `ok` check pattern |

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A3: Effect Legibility | +1 | Panics (hidden failures) replaced by proper errors |
| A5: Bounded Verification | +1 | Contract verification now works for all numeric types |
| A7: Machines First | +1 | Structured errors machines can catch + retry |
| A10: Composability | +1 | Float contracts compose symmetrically with int contracts |
| A11: Structured Failure | +1 | Panics replaced by typed `error` returns with context |

**Net Score: +5**

## Verification

- Float contract `requires { price >= 0.0 }` no longer panics
- Int contracts continue to work correctly
- Type mismatches produce descriptive error messages (e.g., `ge_Int: expected IntValue for arg 0, got *eval.FloatValue`)
- All existing tests pass
- serve-api float coercion fixed (JSON `100.0` preserved as `FloatValue`)

## Related Documents

- [M-BUILTIN-SAFETY](../v0_7_0/m-builtin-safety-type-checks.md) - Partial fix (v0.7.0) that left Int/Bool/logic bare
- [M-VERIFY-CONTRACTS](../v0_7_1/m-verify-contracts.md) - Runtime contract enforcement where the panic manifested
- [M-VERIFY: Runtime Contracts](../v0_6_1/m-verify-runtime-contracts.md) - Original runtime contracts design

## Lessons Learned

1. **M-BUILTIN-SAFETY (v0.7.0) was incomplete**: It claimed "0% panic-prone builtins" but only fixed String and Float comparisons, leaving Int, Bool, logic, arithmetic, and conversion builtins with bare casts.

2. **Systemic audit is essential**: Following CLAUDE.md Principle 6, the fix went beyond the single reported panic to harden ALL 15+ bare type assertion sites across the codebase.

3. **Bare type assertions in dynamic dispatch are always wrong**: When builtins can receive values through dictionary dispatch or contract evaluation, type mismatches are possible even when the type system would normally prevent them. Defensive checking is required.

---

**Implemented**: 2026-02-20
**Commits**: `f9c0ccb7`, `24c7d929`
