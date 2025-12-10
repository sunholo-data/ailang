# M-CODEGEN-STDLIB-MATH: Go Codegen Missing std/math Function Definitions

**Status**: ✅ IMPLEMENTED
**Target**: v0.5.9
**Priority**: P1 - Medium (blocks real-world codebase compilation to Go)
**Actual**: ~1.5 hours
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix, no syntax change |
| Preserve Semantic Clarity | + | +1 | AILANG math functions map correctly to Go math |
| Increase Determinism | + | +1 | Consistent codegen output |
| Lower Token Cost | 0 | 0 | No token impact |
| **Net Score** | | **+2** | **Decision: Move forward** |

## Problem Statement

**Error Message (Go build of generated code):**
```
sim_gen/bridge.go:75:27: undefined: PI
sim_gen/bridge.go:106:25: undefined: Sin
sim_gen/bridge.go:117:27: undefined: PI
sim_gen/bridge.go:149:26: undefined: Cos
```

**Root cause**: `mapEffectBuiltinToHandler` only handles effect builtins, not pure math builtins. When codegen encountered `PI`, `sin`, `cos`, etc., it fell through to `ToPascalCase()` generating undefined identifiers.

## Solution Implemented

### 1. Added `mapPureMathBuiltin` Function

**File:** `internal/gen/golang/codegen_expr_simple.go` (+65 LOC)

Maps all 19 math builtins to Go `math` package:
- Constants: `_math_PI` → `math.Pi`, `_math_E` → `math.E`
- Trig: `sin`, `cos`, `tan` → `math.Sin`, `math.Cos`, `math.Tan`
- Inverse trig: `asin`, `acos`, `atan`, `atan2`
- Exponential: `exp`, `log`, `log10`, `pow`, `sqrt`
- Rounding: `ceil`, `floor`, `round`
- Utility: `abs_Float` → `math.Abs`

Both underscore-prefixed builtins (`_math_sin`) and wrapper names (`sin`) are mapped.

### 2. Added `needsMathImport` Flag

**File:** `internal/gen/golang/codegen.go` (+15 LOC)

- Added `needsMathImport bool` field to Generator struct
- Set to true when `mapPureMathBuiltin` returns a match
- Conditionally adds `"math"` to import block

### 3. Two-Phase Generation

**File:** `internal/gen/golang/codegen.go` (modified Generate function)

Changed from single-pass to two-phase:
1. **Phase 1**: Generate declarations to temporary buffer (detects math usage)
2. **Phase 2**: Write header with correct imports, then append declarations

This ensures the `math` import is only added when actually needed.

### 4. Unit Tests

**File:** `internal/gen/golang/codegen_math_test.go` (+183 LOC)

- `TestMathBuiltinMapping` - Tests all 24 builtin mappings
- `TestMathImportGeneration` - Tests math import is added when needed
- `TestNoMathImportWhenNotNeeded` - Tests import is NOT added when not needed
- `TestMathFunctionCall` - Tests function calls generate correctly
- `TestMathNonBuiltin` - Tests non-math builtins don't trigger math import

## Files Modified

| File | Change | LOC |
|------|--------|-----|
| `internal/gen/golang/codegen_expr_simple.go` | Added `mapPureMathBuiltin` function | +65 |
| `internal/gen/golang/codegen.go` | Added `needsMathImport`, two-phase generation, import logic | +25 |
| `internal/gen/golang/codegen_math_test.go` | New test file | +183 |

**Total: ~273 LOC**

## Test Results

```
=== RUN   TestMathBuiltinMapping (24 subtests)
--- PASS: TestMathBuiltinMapping (0.00s)
=== RUN   TestMathImportGeneration
--- PASS: TestMathImportGeneration (0.00s)
=== RUN   TestMathFunctionCall
--- PASS: TestMathFunctionCall (0.00s)
=== RUN   TestMathNonBuiltin
--- PASS: TestMathNonBuiltin (0.00s)
```

Full test suite passes. Lint passes.

## Success Criteria

- [x] All 19 math builtins properly mapped (24 mappings including wrappers)
- [x] Constants (PI, E) work as values
- [x] Functions (sin, cos, etc.) generate as `math.Sin`, `math.Cos`
- [x] `"math"` import added only when needed
- [x] `"math"` import added even when `skipRuntimeHelpers=true` (multi-file mode)
- [x] All existing tests passing
- [x] No regression in other codegen

## Bug #26 Follow-up Fix

**Issue:** When `skipRuntimeHelpers=true` (multi-file compilation mode), the generated Go file had NO imports at all, even when it used `math.Pi`, `math.Sin`, etc.

**Fix:** Added `else if g.needsMathImport` branch in `writePackageHeader()`:
```go
} else if g.needsMathImport {
    // M-CODEGEN-STDLIB-MATH: Even when skipping runtime helpers,
    // we need to add math import if the code uses math functions
    g.writef("import \"math\"\n\n")
}
```

**Test:** `TestMathImportWithSkipRuntimeHelpers` verifies math import is present when `skipRuntimeHelpers=true`.

## Example Generated Code

**AILANG:**
```ailang
import std/math (PI, sin)
let area = PI * r * r
let y = sin(angle)
```

**Generated Go:**
```go
import (
    "fmt"
    "math"  // ← Added only when math builtins used
    "reflect"
    "strings"
)

// ...
area := math.Pi * r.(float64) * r.(float64)
y := math.Sin(angle)
```

---

**Document created**: 2025-12-10
**Implemented**: 2025-12-10
