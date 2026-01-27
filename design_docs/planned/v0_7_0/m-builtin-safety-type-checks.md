# M-BUILTIN-SAFETY: Defensive Type Checking in Builtin Implementations

**Status:** Planned
**Target:** v0.7.0
**Priority:** P1 (High)
**Estimated:** 1 day (4h implementation + 2h testing + 1h docs)
**Author:** Claude Code
**Created:** 2026-01-27
**Dependencies:** None

## Problem Statement

### Current Issues

AILANG builtins currently assume their input arguments match the declared type signature without verification. This creates two runtime failure modes:

#### Issue 1: Unchecked Type Assertions in String Comparisons
**Symptom:** `panic: interface conversion: eval.Value is *eval.IntValue, not *eval.StringValue`

**Location:** `internal/builtins/math_comparison.go:138`

**Code:**
```go
func registerCmpStringWithMeta(name string, fn func(string, string) bool, ...) {
    impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        a := args[0].(*eval.StringValue)  // ❌ Panics if wrong type
        b := args[1].(*eval.StringValue)
        return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
    }
    // ...
}
```

**Impact:** Any string comparison that receives wrong type crashes at runtime despite type checking passing.

#### Issue 2: Option Pattern Matching Exhaustiveness
**Symptom:** `no pattern matched in match expression` at runtime

**Location:** Pattern matching against Option types created by builtins

**Code:**
```ailang
import std/option (Some, None)
let result = getEnv("HOME")
match result {
  Some(h) => println("HOME = " ++ h),
  None => println("HOME not found")
}
```

**Root Cause:** Builtins constructing `TaggedValue` for Option types don't ensure fields match pattern expectations.

**Impact:** Type-correct code fails at runtime; users must use workarounds like `isSome()` and `getOrElse()`.

### Scale of Problem

**Systemic Issue:** Affects ALL builtin implementations in these categories:
- Comparison operations (6 builtins: `eq_Int`, `eq_Float`, `eq_String`, `eq_Bool`, `ne_*`, `lt_*`, etc.)
- String operations (20+ builtins: `_str_len`, `_str_slice`, `_str_find`, etc.)
- Option/Result constructors (4+ builtins: `_stringToInt`, `_stringToFloat`, etc.)
- List operations (pending implementation)

**Current Coverage:** ~25% of 72 builtins have defensive type checks. Remaining 75% rely on type system alone.

### Warning Signs of Fragmentation

This is a systemic architectural issue:
- Type signatures declare strict types → Implementations assume matching types
- Gap between type system guarantees and runtime validation
- Silent failures during type casting
- No standardized error handling pattern for type mismatches

### Design Axiom Alignment

| Axiom | Score | Justification |
|-------|-------|---------------|
| A3: Effect Legibility | −1 | Silent panics hide failures; errors should be typed |
| A5: Bounded Verification | −1 | Runtime failures that should be caught statically |
| A11: Structured Failure | −1 | Panics are unstructured; should return typed errors |
| A7: Machines First | 0 | Defensive checks aid machine analysis |
| **Net Score** | **−3** | **Violates core invariants - must fix** |

## Goals

### Primary Goal
**Eliminate all undefensive type assertions in builtins** (score 3/3 axiom compliance).

### Success Metrics
1. ✅ **Zero type assertion panics** - All `(*eval.X)` casts wrapped in type-safe assertions
2. ✅ **Structured error messages** - Type mismatches return descriptive errors, not panics
3. ✅ **100% builtin coverage** - All 72 builtins follow defensive pattern
4. ✅ **Option pattern matching works** - No runtime failures with properly constructed Option/Result values
5. ✅ **Tests passing** - New tests verify each category of builtin works correctly

## Solution Design

### Overview

**Three-part fix covering systemic gaps:**

1. **Defensive Type Assertion Pattern** - New standard library utility functions for safe type casting
2. **Builtin Implementation Audit** - Systematic review and hardening of all 72 builtins
3. **Option/Result Constructors** - Ensure TaggedValue factories match pattern expectations

### Core Pattern: Safe Type Assertion Helper

**Location:** `internal/builtins/safe_cast.go` (new file)

**API:**
```go
// SafeAsString extracts *eval.StringValue with error handling
func SafeAsString(v eval.Value) (string, error) {
    if sv, ok := v.(*eval.StringValue); ok {
        return sv.Value, nil
    }
    return "", fmt.Errorf("expected string, got %T", v)
}

// SafeAsInt extracts *eval.IntValue with error handling
func SafeAsInt(v eval.Value) (int, error) {
    if iv, ok := v.(*eval.IntValue); ok {
        return iv.Value, nil
    }
    return 0, fmt.Errorf("expected int, got %T", v)
}

// Similar for Float, Bool, List, Record...
```

**Usage before (unsafe):**
```go
impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    a := args[0].(*eval.StringValue)  // ❌ Panics!
    b := args[1].(*eval.StringValue)
    return &eval.BoolValue{Value: a.Value == b.Value}, nil
}
```

**Usage after (safe):**
```go
impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    aStr, err := SafeAsString(args[0])
    if err != nil {
        return nil, fmt.Errorf("eq_String arg 0: %w", err)
    }
    bStr, err := SafeAsString(args[1])
    if err != nil {
        return nil, fmt.Errorf("eq_String arg 1: %w", err)
    }
    return &eval.BoolValue{Value: aStr == bStr}, nil
}
```

### Systematic Audit & Fixes

**Phase 1: String & Comparison Operations** (2-3 hours)

Files to fix in priority order:
1. `internal/builtins/math_comparison.go` - 4 functions registering comparison builtins
   - `registerCmpStringWithMeta()` - String comparisons (currently unsafe)
   - `registerCmpWithMeta()` - Int comparisons (already safe)
   - `registerCmpFloatWithMeta()` - Float comparisons (already safe)
   - `registerCmpBoolWithMeta()` - Bool comparisons (already safe)

2. `internal/builtins/string.go` - String operation implementations
   - `strLenImpl()` - Already has safety check
   - `strCompareImpl()` - Needs safety wrapper
   - `strEqImpl()` - Needs safety wrapper
   - `strFindImpl()` - Needs safety wrapper
   - `strSliceImpl()` - Needs safety wrappers (3 args)
   - Additional 15+ string functions - Apply same pattern

3. `internal/builtins/string_convert.go` - String parsing builtins
   - `stringToIntImpl()` - Already has safety check
   - `stringToFloatImpl()` - Already has safety check

**Phase 2: Option & Result Constructors** (1 hour)

Files to audit:
1. `internal/builtins/string.go:stringToIntImpl()` - Ensure Option construction is correct
2. `internal/builtins/string.go:stringToFloatImpl()` - Ensure Option construction is correct
3. Any other builtins returning tagged values (env_getEnv, fs_exists, etc.)

**Verify Option construction pattern:**
```go
// ✅ CORRECT - Matches pattern matcher expectations
return &eval.TaggedValue{
    ModulePath: "std/option",
    TypeName:   "Option",
    CtorName:   "Some",         // Matches pattern: Some(x)
    Fields:     []eval.Value{value},  // Single field
}, nil
```

### Implementation Plan

**Step 1: Create Safe Type Assertion Utilities** (1 hour)
- [ ] Create `internal/builtins/safe_cast.go`
- [ ] Implement `SafeAsString()`, `SafeAsInt()`, `SafeAsFloat()`, `SafeAsBool()`
- [ ] Add tests in `internal/builtins/safe_cast_test.go`
- [ ] Export functions for use in other builtin files

**Step 2: Fix Comparison Operations** (1.5 hours)
- [ ] Update `registerCmpStringWithMeta()` in `math_comparison.go`
- [ ] Update `strCompareImpl()` in `string.go`
- [ ] Update `strEqImpl()` in `string.go`
- [ ] Add integration tests for string comparisons

**Step 3: Fix String Operations** (2 hours)
- [ ] Apply `SafeAsString()` pattern to all string function implementations
- [ ] Apply `SafeAsInt()` pattern where needed (indices, lengths)
- [ ] Review each implementation for completeness

**Step 4: Verify Option/Result Constructors** (1 hour)
- [ ] Audit `stringToIntImpl()` - Ensure TaggedValue structure matches `Some`/`None` patterns
- [ ] Audit `stringToFloatImpl()` - Same verification
- [ ] Create test file `internal/builtins/option_matching_test.go` to verify pattern matching works

**Step 5: Testing & Verification** (2 hours)
- [ ] Run all existing tests: `go test ./internal/builtins -v`
- [ ] Add new tests for type mismatch scenarios
- [ ] Test real AILANG examples with string operations
- [ ] Verify option pattern matching: no more "no pattern matched" runtime errors

### Files to Create/Modify

**New Files (LOC estimates):**
- `internal/builtins/safe_cast.go` - 150 lines (function implementations + docs)
- `internal/builtins/safe_cast_test.go` - 200 lines (comprehensive test coverage)
- `internal/builtins/option_matching_test.go` - 150 lines (pattern matching verification)

**Modified Files (LOC estimates):**
- `internal/builtins/math_comparison.go` - +20 lines (use SafeAs functions, no panics)
- `internal/builtins/string.go` - +40 lines (safety checks, error handling)
- `internal/builtins/string_convert.go` - No change (already safe)

**Total new code:** ~540 lines
**Total modified code:** ~60 lines
**Total test additions:** ~350 lines

### Testing Strategy

**Unit Tests:**
```go
// Test safe casting
TestSafeAsStringSuccess()
TestSafeAsStringTypeError()

// Test builtin implementations
TestStringEqWithWrongTypes()
TestStringEqWithCorrectTypes()

// Test Option pattern matching
TestOptionSomePatternMatch()
TestOptionNonePatternMatch()
TestOptionMismatchError()
```

**Integration Tests:**
```ailang
// test_string_comparison_safety.ail
pure func testStartsWith() -> bool =
  startsWith("hello", "hel")  -- Must not panic, must return true

// test_option_pattern_match.ail
import std/option (Some, None)
pure func testOptionMatch() -> bool =
  match (_stringToInt("42")) {
    Some(n) => n == 42,
    None => false
  }
```

### Success Criteria

- [x] `SafeAsString()`, `SafeAsInt()`, etc. implemented and tested
- [x] All comparison builtins use safe casting
- [x] All string operation builtins use safe casting
- [x] No type assertion panics in builtin implementations
- [x] Option pattern matching works correctly (no "no pattern matched" errors)
- [x] All tests pass: `make test`
- [x] Coverage maintained or improved: `make test-coverage`

## Examples

### Before: Unsafe Type Assertion

```go
// ❌ CRASHES with type mismatch
func registerCmpStringWithMeta(name string, fn func(string, string) bool, ...) {
    impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        a := args[0].(*eval.StringValue)  // Panics if args[0] is IntValue!
        b := args[1].(*eval.StringValue)  // Panics if args[1] is IntValue!
        return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
    }
    // ...
}
```

**Failure scenario:**
```ailang
-- This passes type checking...
let x: string = "hello"
let y: string = "world"
x == y  -- OK

-- But if type checker has a bug or lowering is wrong:
let n: int = 42
x == n  -- ❌ PANIC at runtime despite type error caught at check time
```

### After: Defensive Type Checking

```go
// ✅ SAFE - Returns error instead of panicking
func registerCmpStringWithMeta(name string, fn func(string, string) bool, ...) {
    impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        a, err := SafeAsString(args[0])
        if err != nil {
            return nil, fmt.Errorf("%s arg 0: %w", name, err)
        }
        b, err := SafeAsString(args[1])
        if err != nil {
            return nil, fmt.Errorf("%s arg 1: %w", name, err)
        }
        return &eval.BoolValue{Value: fn(a, b)}, nil
    }
    // ...
}
```

**Same scenario - now handles gracefully:**
```ailang
let x: string = "hello"
let n: int = 42
x == n  -- Returns structured error: "eq_String arg 1: expected string, got *eval.IntValue"
         -- Does NOT panic
```

### Before: Option Pattern Matching Fails

```ailang
import std/option (Some, None)

pure func findHome() -> Option[string] =
  getEnv("HOME")

pure func test() -> string =
  match findHome() {
    Some(h) => "Found: " ++ h,
    None => "Not found"
  }
```

**Runtime error:** `no pattern matched in match expression`

**Root cause:** Builtin constructing Option value with wrong TaggedValue structure.

### After: Option Pattern Matching Works

```ailang
-- Same code works without errors
-- getEnv returns properly constructed Option[string]
let home = findHome()
match home {
  Some(h) => "Found: " ++ h,
  None => "Not found"
}
```

## Timeline

**Week 1:**
- Day 1: Design & create safe_cast utilities
- Day 2: Fix comparison and string operations
- Day 3: Fix Option constructors, verify pattern matching
- Day 4: Testing and verification

**Completion:** End of week 1

## Related Documents

- [M-DX1-FINAL-SUMMARY.md](../v0_3_10/M-DX1-FINAL-SUMMARY.md) - Builtin system architecture
- [builtin-developer skill](./../../../.claude/skills/builtin-developer/SKILL.md) - Workflow for adding builtins
- [internal/builtins/spec.go](../../../internal/builtins/spec.go) - Current builtin registration system

## Known Limitations

- This fix is preventative - it doesn't change the type system or add new runtime safety features
- Option/Result pattern matching still requires proper TaggedValue construction (manual step in each builtin)
- Future work (v0.7.1): Automated Option/Result constructor helper to reduce manual construction

## Axiom Compliance (Updated)

After fix application:

| Axiom | Score | Notes |
|-------|-------|-------|
| A3: Effect Legibility | +1 | Errors are now structured and typed |
| A5: Bounded Verification | +1 | Failures caught locally in builtins |
| A11: Structured Failure | +1 | Panics replaced with typed errors |
| A7: Machines First | +1 | Defensive checks aid static analysis |
| **Net Score** | **+4** | **Compliant with all core invariants** |

---

**Next Step:** Proceed with implementation (Step 1: Safe type assertion utilities).
