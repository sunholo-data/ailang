# String Conversion Functions (floatToStr, intToStr)

**Status**: ✅ Implemented
**Target**: v0.5.10
**Priority**: P1 (Medium)
**Estimated**: 3 hours
**Actual**: ~2.5 hours
**Dependencies**: None (builds on existing std/math builtins infrastructure)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | No workarounds needed for basic type conversion |
| Preserve Semantic Clarity | 0 | 0 | Explicit function calls, no magic |
| Increase Determinism | + | +1 | Pure functions with predictable output |
| Lower Token Cost | + | +1 | Single function call vs complex workarounds |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Context:** Feature request from Stapledon's Voyage project (GitHub Issue #29)

AILANG lacked basic type-to-string conversion functions. Users could not convert numeric values to strings for display purposes (HUD text, logging, debugging output).

**Previous State:**
- No `floatToStr` function existed
- No `intToStr` function existed
- Users had no workaround for rendering numeric values as text
- Stapledon's Voyage HUD sprint blocked (8/9 criteria met, missing this)

## Implementation Summary

### Components Implemented

1. **Builtin Registration** (`internal/builtins/string_convert.go` - ~99 LOC)
   - `_string_floatToStr`: float → string (uses `strconv.FormatFloat` with 'g' format)
   - `_string_intToStr`: int → string (uses `strconv.Itoa`)
   - Full metadata documentation
   - Unit tests in `string_convert_test.go`

2. **std/string Module** (existing `std/string.ail` extended)
   - Export wrappers: `floatToStr(f: float) -> string`
   - Export wrappers: `intToStr(n: int) -> string`
   - Both marked as `pure func`

3. **Go Codegen Support** (`internal/gen/golang/`)
   - Added `needsStrconvImport` flag to Generator
   - Added `StringConvKind` enum and `getStringConvFunction()` mapping
   - Special handling in `generateApp()` for:
     - `floatToStr(f)` → `strconv.FormatFloat(f.(float64), 'g', -1, 64)`
     - `intToStr(n)` → `strconv.Itoa(int(n.(int64)))`
   - Automatic `strconv` import when needed
   - Tests in `codegen_string_test.go`

### Files Created/Modified

**New files:**
- `internal/builtins/string_convert.go` - Builtin implementations (~99 LOC)
- `internal/builtins/string_convert_test.go` - Unit tests (~130 LOC)
- `internal/gen/golang/codegen_string_test.go` - Codegen tests (~260 LOC)

**Modified files:**
- `std/string.ail` - Added floatToStr and intToStr exports
- `internal/gen/golang/codegen.go` - Added needsStrconvImport field
- `internal/gen/golang/codegen_expr_simple.go` - Added string conversion mappings
- `internal/gen/golang/codegen_expr_app.go` - Added string conversion handling
- `internal/builtins/spec_test.go` - Fixed test isolation with `withFreshRegistry(t)`
- `internal/pipeline/testdata/builtin_types.golden` - Updated golden file

## Examples

### Example 1: HUD Text Rendering (Stapledon's Voyage Use Case)

```ailang
import std/string (floatToStr, intToStr)

let velocity = 0.15
let gamma = 1.0075   -- Lorentz factor

-- Build HUD text
let velocityText = floatToStr(velocity * 100.0) ++ "% c"
let gammaText = "γ = " ++ floatToStr(gamma)
```

### Example 2: Debug Output

```ailang
import std/string (intToStr, floatToStr)
import std/io (println)

let count = 42
let ratio = 3.14159

-- Debug output
println("Count: " ++ intToStr(count))
println("Ratio: " ++ floatToStr(ratio))
```

## Success Criteria

- [x] `floatToStr(3.14)` returns `"3.14"`
- [x] `intToStr(42)` returns `"42"`
- [x] `floatToStr(-0.5)` returns `"-0.5"` (negative numbers)
- [x] `intToStr(-100)` returns `"-100"` (negative numbers)
- [x] Functions are pure (no effect annotation required)
- [x] All tests passing
- [x] Go codegen support (strconv import auto-added)
- [ ] Example file works end-to-end (blocked by Option[T] type annotation bug)

## Known Issues

**std/string import blocked**: The std/string module currently cannot be imported due to a pre-existing bug with Option[T] return type annotations in the type checker. See [m-fix-option-type-annotation.md](./m-fix-option-type-annotation.md) for details.

**Workaround**: Call builtins directly:
```ailang
-- Until Option[T] bug is fixed, use builtins directly
let s = _string_floatToStr(3.14)
```

## Testing Strategy

**Unit tests (implemented):**
- Test positive/negative numbers ✅
- Test zero values ✅
- Test large numbers ✅
- Test float precision ✅

**Go codegen tests (implemented):**
- Test builtin mapping ✅
- Test strconv import generation ✅
- Test type assertions ✅
- Test skipRuntimeHelpers mode ✅

## Non-Goals

**Not in this feature:**
- `strToFloat` / `strToInt` (parsing) - Already exists in std/string
- Formatting options (precision, padding) - Future enhancement
- `boolToStr` - Can be added later if needed
- Locale-aware formatting - Out of scope

## Bug Fixes During Implementation

While implementing this feature, discovered and fixed a pre-existing test isolation bug in `internal/builtins/spec_test.go`:
- Tests were clearing the global registry without restoring it
- Added `withFreshRegistry(t)` helper that uses `t.Cleanup()` to restore state
- This prevented test order-dependent failures

## References

- [GitHub Issue #29](https://github.com/sunholo-data/ailang/issues/29) - Original feature request
- Message from stapledons_voyage - Context for use case
- `internal/builtins/math_conversion.go` - Similar pattern for intToFloat/floatToInt
- `internal/gen/golang/codegen_math_test.go` - Pattern for stdlib codegen tests

## Future Work

- Fix Option[T] type annotation bug to enable full std/string imports
- String parsing functions (`strToFloat`, `strToInt`) with error handling
- Formatting functions with precision control
- Additional conversions (`boolToStr`, `charToStr`)
- String manipulation functions (substring, replace, split)

---

**Document created**: 2025-12-10
**Implementation completed**: 2025-12-10
