# M-FIX-FLOAT-OP: Float Operator Dispatch in Pure Functions

**Status**: ✅ Implemented (fixed by M-TAPP-FIX)
**Target**: v0.5.10
**Priority**: P0 - High (blocks std/math usage in user code)
**Actual Time**: Fixed as side-effect of M-TAPP-FIX
**Dependencies**: None (bug fix)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No change to syntax |
| Preserve Semantic Clarity | + | +1 | Float operations execute correctly as expected |
| Increase Determinism | + | +1 | Operators dispatch deterministically based on type |
| Lower Token Cost | + | +1 | Users don't need workarounds (e.g., `pow(x,2.0)` instead of `x*x`) |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

Float arithmetic operators (`+`, `-`, `*`, `/`) in user-defined pure functions dispatch to integer implementations (`mul_Int`, `add_Int`, etc.) instead of float implementations (`mul_Float`, `add_Float`, etc.), causing runtime panics.

**Error Message:**
```
panic: interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue
```

**Failing Code (before fix):**
```ailang
-- This panicked at runtime
pure func distance(x1: float, y1: float, x2: float, y2: float) -> float =
    sqrt((x2 - x1) * (x2 - x1) + (y2 - y1) * (y2 - y1))
```

## Root Cause

The bug was fixed as a **side effect** of M-TAPP-FIX changes. The improvements to type propagation through the pipeline (specifically the correct handling of type applications and ADT type parameters) also fixed the float operator dispatch issue.

Key changes that contributed to the fix:
1. Better type variable tracking in constructor factory types
2. Correct TApp creation for parameterized ADTs
3. Improved type parameter propagation through the pipeline

## Solution

Fixed automatically by M-TAPP-FIX. No additional changes needed.

See [m-fix-option-type-annotation.md](m-fix-option-type-annotation.md) for the implementation details.

## Verification

**Test Results (2025-12-11):**

```ailang
pure func square(x: float) -> float = x * x
pure func distance(x1: float, y1: float, x2: float, y2: float) -> float =
    let dx = x2 - x1 in
    let dy = y2 - y1 in
    sqrt(dx * dx + dy * dy)
pure func quadratic(a: float, b: float, c: float, x: float) -> float =
    a * x * x + b * x + c
pure func circleArea(r: float) -> float =
    PI() * r * r
```

**Output:**
```
=== Float Operators in Pure Functions ===
distance(0,0,3,4) = 5.0       ✅
quadratic(1,2,1,3) = 16.0     ✅
circleArea(2) = 12.566370...  ✅
```

## Success Criteria ✅

- [x] `pure func f(x: float) -> float = x * x` compiles and runs correctly
- [x] `distance(0.0, 0.0, 3.0, 4.0)` returns `5.0`
- [x] Float operators work with let-bound variables in pure functions
- [x] No regression in integer arithmetic
- [x] All tests passing

## Impact

- `examples/math_trig.ail` no longer needs `pow(x, 2.0)` workaround
- Users can write natural float arithmetic in pure functions
- std/math functions can be used freely in user-defined pure functions

---

**Document created**: 2025-12-10
**Implemented**: 2025-12-11 (as side effect of M-TAPP-FIX)
