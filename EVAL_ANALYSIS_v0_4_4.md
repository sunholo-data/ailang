# AILANG v0.4.4 Eval Analysis & Findings

**Date**: 2025-11-14
**Agent Eval Success**: 96.1% (73/76)
**Failures**: 3/76 (3.9%)

## Executive Summary

The v0.4.4 agent eval revealed **excellent AI-friendliness** (96.1% success) but uncovered **one critical compiler bug** and **one prompt gap** that prevent perfect scores.

## Critical Findings

### 1. ✅ List Cons Syntax WORKS PERFECTLY

**Tested**: `1 :: 2 :: 3 :: []`, `x :: rest` pattern matching, `(x * 2) :: doubleList(rest)`
**Result**: ✅ All syntax works correctly
**Conclusion**: The `deterministic_list_transform` failure was a **prompt issue**, not a language issue

**Evidence**:
```ailang
export func doubleList(xs: [int]) -> [int] {
  match xs {
    [] => [],
    x :: rest => (x * 2) :: doubleList(rest)  // ✅ Works perfectly
  }
}
// Output: [2, 4, 6] ✓
```

**Recommendation**: Update AI teaching prompt with list cons examples.

---

### 2. ❌ COMPILER BUG: Nullary Constructor Pattern Matching

**Severity**: High (blocks eval benchmarks)
**Impact**: `exhaustive_pattern_matching` benchmark fails
**Status**: Design doc created ([M-BUG-NULLARY](design_docs/planned/v0_4_5/nullary-constructor-pattern-matching-bug.md))

#### Problem

Pattern matching **fails** when an ADT has **only nullary constructors**. All values match the first pattern.

#### Reproduction

```ailang
type Status = Pending | InProgress | Completed

func describeStatus(s: Status) -> string {
  match s {
    Pending => "Waiting",
    InProgress => "Working",
    Completed => "Done"
  }
}

// ❌ BUG: All return "Waiting"
describeStatus(Pending)     // ✓ "Waiting" (correct)
describeStatus(InProgress)  // ❌ "Waiting" (should be "Working")
describeStatus(Completed)   // ❌ "Waiting" (should be "Done")
```

#### What Works

```ailang
type Status = Pending(int) | InProgress(int) | Completed(int)

func describeStatus(s: Status) -> string {
  match s {
    Pending(x) => "Waiting: " ++ show(x),
    InProgress(x) => "Working: " ++ show(x),
    Completed(x) => "Done: " ++ show(x)
  }
}

// ✅ WORKS: All return correct values
describeStatus(Pending(1))     // ✓ "Waiting: 1"
describeStatus(InProgress(2))  // ✓ "Working: 2"
describeStatus(Completed(3))   // ✓ "Done: 3"
```

**Pattern**: Works when at least one constructor has arguments.

#### Root Cause (Hypothesis)

The compiler likely:
1. Generates correct tag values (show() displays them correctly)
2. But **doesn't emit tag comparison code** for nullary constructors in match expressions
3. Defaults to first pattern when no comparison exists

#### Fix Locations

Likely in:
- `internal/dtree/` - Decision tree compilation
- `internal/elaborate/` - Pattern lowering
- `internal/eval/` - Runtime matching

See [design_docs/planned/v0_4_5/nullary-constructor-pattern-matching-bug.md](design_docs/planned/v0_4_5/nullary-constructor-pattern-matching-bug.md) for detailed analysis.

---

## Eval Results Breakdown

### Failures (3/76)

1. **deterministic_list_transform** (claude-sonnet-4-5)
   - **Issue**: Agent timed out trying different list cons syntax approaches
   - **Root Cause**: Prompt doesn't show list cons examples
   - **Fix**: Update AI teaching prompt with `::` syntax examples
   - **Language Issue**: None - syntax works perfectly

2. **exhaustive_pattern_matching** (claude-sonnet-4-5)
   - **Issue**: Compiler bug in nullary constructor matching
   - **Root Cause**: Pattern matching broken for ADTs with only nullary constructors
   - **Fix**: Fix compiler (see M-BUG-NULLARY design doc)
   - **Language Issue**: Yes - critical compiler bug

3. **exhaustive_pattern_matching** (claude-haiku-4-5)
   - **Issue**: Same as #2
   - **Root Cause**: Same compiler bug
   - **Fix**: Same as #2

### Successes (73/76 = 96.1%)

All other AILANG benchmarks passed with both Claude models:
- fizzbuzz, recursion_factorial, recursion_fibonacci
- simple_print, records_person, list_operations
- string_manipulation, nested_records, higher_order_functions
- pattern_matching_complex, record_update, effect_composition
- effect_tracking_io_fs, effect_pure_separation
- type_safe_record_access, explicit_state_threading
- referential_transparency

**This is excellent evidence of AILANG's AI-friendliness!**

---

## Recommendations

### Immediate (v0.4.5)

1. **Fix nullary constructor bug** (3-5 hours)
   - Follow design doc: [M-BUG-NULLARY](design_docs/planned/v0_4_5/nullary-constructor-pattern-matching-bug.md)
   - Add regression tests
   - Expected outcome: `exhaustive_pattern_matching` passes → 98.7% success (75/76)

2. **Update AI teaching prompt** (30 minutes)
   - Add list cons syntax examples:
     ```ailang
     -- Building lists
     let list = 1 :: 2 :: 3 :: []

     -- Pattern matching
     match list {
       [] => "empty",
       x :: rest => "head: " ++ show(x)
     }

     -- Constructing in recursion
     func double(xs: [int]) -> [int] {
       match xs {
         [] => [],
         x :: rest => (x * 2) :: double(rest)
       }
     }
     ```
   - Expected outcome: `deterministic_list_transform` passes → 100% success (76/76)

### Future Improvements

1. **Increase eval coverage** (optional)
   - Add more nullary constructor tests
   - Add more list manipulation benchmarks
   - Test edge cases discovered in manual testing

2. **Improve timeout handling** (optional)
   - Current 60s timeout is appropriate (tests AI-friendliness)
   - Don't increase - fix root causes instead

---

## CI/CD Status

**Current**: Build and Release workflow failing on Windows
**Issue**: Test validation order needs adjustment
**Fix**: In progress (reordering validation checks in `stdlib_resolver.go`)

---

## Conclusion

AILANG v0.4.4 demonstrates **96.1% AI code generation success**, proving the language design is highly effective for AI agents.

The two remaining issues are:
1. **Nullary constructor bug**: Fixable compiler bug (3-5 hours)
2. **List cons prompt gap**: Fixable documentation issue (30 minutes)

After these fixes, we expect **100% success on agent evals**.
