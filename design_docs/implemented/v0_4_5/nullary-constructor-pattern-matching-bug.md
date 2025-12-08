# Nullary Constructor Pattern Matching Bug

**Status**: Planned
**Target**: v0.4.5
**Priority**: P0 (High) - Blocks eval benchmarks
**Estimated**: 3-5 hours
**Dependencies**: None
**Bug ID**: M-BUG-NULLARY

## Problem Statement

Pattern matching fails when an ADT has **only nullary constructors** (constructors with zero arguments). All values match the first pattern regardless of which constructor was used, breaking the fundamental guarantee of exhaustive pattern matching.

**Current State:**
- `exhaustive_pattern_matching` eval benchmark: **96.1% success** (fails on nullary-only ADTs)
- All nullary enum values match first pattern, breaking exhaustiveness checking
- Simple enum types (Status, Color, Direction) are **unusable in production**
- AI code generation fails on 3.9% of common enum patterns

**Impact:**
- **Who affected**: All AILANG users + AI code generators
- **Severity**: High - Breaks type safety guarantees for entire class of types
- **Workaround cost**: Must use wrapper types or boolean flags (+30% code overhead)
- **Blocks**: Production use of domain modeling with simple enums

### What Works (Important Context)

Pattern matching works correctly for:
- ✅ **List patterns**: `x :: rest`, `1 :: 2 :: 3 :: []` (confirmed v0.4.4 eval analysis)
- ✅ **Mixed ADTs**: `type Option = Some(int) | None` - both variants match correctly
- ✅ **Non-nullary only**: ADTs where all constructors take arguments
- ✅ **Destructuring**: `match x { (a, b) => ... }` works for tuples, records

**Conclusion**: The pattern matching system core is **working correctly** - this is a specific bug in nullary-only ADT handling, not a systemic issue.

### Reproduction

```ailang
type SimpleEnum = A | B | C

func test(e: SimpleEnum) -> string {
  match e {
    A => "First",
    B => "Second",
    C => "Third"
  }
}

-- BUG: All three return "First"
test(A)  -- ✓ Returns "First" (correct)
test(B)  -- ❌ Returns "First" (should be "Second")
test(C)  -- ❌ Returns "First" (should be "Third")
```

**Comparison with working case:**
```ailang
-- This works correctly (mixed nullary/non-nullary)
type MyOption = MySome(int) | MyNone

func testOption(opt: MyOption) -> string {
  match opt {
    MySome(x) => "Some: " ++ show(x),
    MyNone => "None"
  }
}

testOption(MySome(42))  -- ✅ Returns "Some: 42"
testOption(MyNone)      -- ✅ Returns "None"
```

**Pattern**: Works when ADT has at least one constructor with arguments.

## Goals

**Primary Goal:** Fix nullary constructor pattern matching to enable simple enum types and restore type safety guarantees.

**Success Metrics:**
- Reproduction test cases: **0/3 passing → 3/3 passing**
- `exhaustive_pattern_matching` benchmark: **96.1% → 100% success**
- Zero regressions: All existing pattern matching tests still pass
- Performance: No measurable regression (<1% acceptable)
- AI codegen: Successfully generates nullary enum code

## Solution Design

### Overview

The bug exists in one of three places: decision tree compilation, pattern lowering, or runtime evaluation. Investigation will identify the exact location, then apply a minimal fix to ensure tag comparison happens for nullary constructors.

**Key insight**: The fact that `None` works in `Option` types tells us:
1. Runtime representation of nullary values is correct (they have tags)
2. Tag comparison logic exists somewhere (it works for mixed ADTs)
3. The bug is in a code path that **only triggers for nullary-only ADTs**

This narrows the search: likely an optimization or special case that incorrectly assumes "all nullary = identical".

### Root Cause Hypotheses

**Hypothesis 1: Tag Comparison Missing for Nullary Variants** (Most likely)

Nullary constructors are represented as simple tagged values with no payload. The compiler may be:
1. ✅ Generating correct tag values (confirmed: `show()` displays them correctly)
2. ❌ **Not emitting tag comparison code** in match expressions for nullary-only ADTs
3. ❌ Defaulting to first pattern when no tag comparison exists

**Expected behavior:**
```
match value {
  A => ...  // Should check: value.tag == A.tag (0)
  B => ...  // Should check: value.tag == B.tag (1)
  C => ...  // Should check: value.tag == C.tag (2)
}
```

**Suspected actual behavior:**
```
match value {
  A => ...  // No tag check - always matches
  B => ...  // Never reached
  C => ...  // Never reached
}
```

**Hypothesis 2: Arity=0 Optimization Bug** (Less likely)

The compiler might be applying an incorrect optimization:
- Assumes all nullary constructors are identical (no data to distinguish)
- Skips tag checking entirely for arity=0 variants
- Always matches first pattern

**Evidence for H2:**
- Mixed ADTs work (non-nullary constructors force tag checking)
- Pure nullary ADTs fail (optimizer sees arity=0, skips checks?)

### Investigation Plan

**Phase 1: Verify Core AST representation** (~30 min)

```bash
# Create minimal test case
cat > /tmp/test_nullary.ail <<EOF
type E = A | B | C
func f(e: E) -> string { match e { A => "a", B => "b", C => "c" } }
EOF

# Check elaboration to Core
ailang debug ast /tmp/test_nullary.ail --show-types
```

**Expected**: Each variant pattern should have distinct `Tag` values (0, 1, 2)
**If broken**: Tags are missing or all zero → Bug in elaboration

**Phase 2: Check decision tree compilation** (~30 min)

```bash
# Search for nullary handling in decision tree code
grep -rn "Arity.*0\|nullary" internal/dtree/
grep -rn "TagTest\|MatchVariant" internal/dtree/
```

**Look for**: Code that handles `variant.Arity == 0` - may be missing tag test generation
**Red flag**: Early return when arity=0 without emitting TagTest

**Phase 3: Check runtime evaluation** (~30 min)

```bash
# Search for pattern matching in evaluator
grep -rn "MatchVariant\|matchPattern" internal/eval/
grep -A 10 "case.*VariantPattern" internal/eval/eval.go
```

**Look for**: Tag comparison logic - may be skipped when `arity == 0`
**Red flag**: Checking arity before checking tag

**Phase 4: Add debug logging** (~30 min)

Instrument suspected code paths:
```go
// In internal/dtree/compile.go or internal/elaborate/patterns.go
if variant.Arity == 0 {
    fmt.Fprintf(os.Stderr, "DEBUG: Nullary variant %s tag=%d\n", variant.Name, variant.Tag)
}
```

Run test case and observe: Do different nullary constructors have different tags?

### Implementation Plan

**Phase 1: Locate Bug** (~1-2 hours)
- [ ] Run investigation steps 1-4
- [ ] Identify exact code location of bug
- [ ] Understand why mixed ADTs work but nullary-only fail
- [ ] Document findings in commit message

**Phase 2: Apply Fix** (~1-2 hours)
- [ ] Implement minimal fix (see "Likely Fix Locations" below)
- [ ] Verify fix with manual test case
- [ ] Run existing pattern matching tests (ensure no regressions)
- [ ] Write unit tests for nullary matching

**Phase 3: Testing & Documentation** (~1 hour)
- [ ] Run full test suite
- [ ] Run `exhaustive_pattern_matching` benchmark
- [ ] Update CHANGELOG.md
- [ ] Update `docs/LIMITATIONS.md` if applicable (remove limitation)
- [ ] Commit with detailed message

### Likely Fix Locations

**Fix Location 1: Decision Tree (`internal/dtree/compile.go`)**

Ensure tag comparison is generated even for nullary patterns:
```go
func compilePattern(pat Pattern) DecisionTree {
    switch p := pat.(type) {
    case VariantPattern:
        // FIX: Always emit TagTest, even when arity=0
        return TagTest{
            Tag:      p.Tag,
            ThenTree: compileSubPatterns(p.Subpatterns),  // Will be empty for arity=0
            ElseTree: Fail,
        }
    }
}
```

**Fix Location 2: Runtime Evaluation (`internal/eval/eval.go`)**

Ensure tag comparison happens before arity check:
```go
func matchPattern(value Value, pattern Pattern) (bool, Bindings) {
    switch v := value.(type) {
    case VariantValue:
        if pat, ok := pattern.(VariantPattern); ok {
            // FIX: Check tag FIRST, before arity check
            if v.Tag != pat.Tag {
                return false, nil
            }
            // Now handle arguments (if arity > 0)
            if pat.Arity == 0 {
                return true, emptyBindings()
            }
            // ... handle non-nullary arguments
        }
    }
}
```

**Fix Location 3: Pattern Lowering (`internal/elaborate/patterns.go`)**

Ensure tag is preserved during elaboration:
```go
func (e *Elaborator) elaboratePattern(pat *ast.Pattern) *core.Pattern {
    switch p := pat.Kind.(type) {
    case *ast.VariantPattern:
        variant := e.lookupVariant(p.Name)
        // FIX: Ensure tag is set correctly for nullary variants
        return &core.PatternVariant{
            Tag:   variant.Tag,
            Arity: variant.Arity,  // Will be 0 for nullary
            Args:  nil,            // Empty, not omitted
        }
    }
}
```

### Files to Modify/Create

**Modified files** (exact location TBD based on investigation):
- `internal/dtree/compile.go` (~5-10 LOC) - Add tag test for nullary patterns
- `internal/eval/eval.go` (~5-10 LOC) - Tag comparison before arity check
- `internal/elaborate/patterns.go` (~5 LOC) - Preserve tag for nullary variants

**New test files:**
- `internal/eval/eval_test.go` (+40 LOC) - `TestNullaryConstructorMatching`
- `tests/nullary_pattern_matching_test.ail` (+20 LOC) - Integration test

**Documentation:**
- `CHANGELOG.md` (+10 LOC) - Document fix
- `docs/LIMITATIONS.md` (remove limitation if listed)

**Total estimated changes:** ~80-100 LOC

## Examples

### Example 1: Simple Enum (Currently Broken)

**Before (broken):**
```ailang
type Color = Red | Green | Blue

func describe(c: Color) -> string {
  match c {
    Red => "Red color",
    Green => "Green color",
    Blue => "Blue color"
  }
}

-- BUG: All return "Red color"
describe(Red)    -- Returns "Red color" ✓
describe(Green)  -- Returns "Red color" ❌
describe(Blue)   -- Returns "Red color" ❌
```

**After (fixed):**
```ailang
type Color = Red | Green | Blue

func describe(c: Color) -> string {
  match c {
    Red => "Red color",
    Green => "Green color",
    Blue => "Blue color"
  }
}

-- Fixed: Each returns correct value
describe(Red)    -- Returns "Red color" ✓
describe(Green)  -- Returns "Green color" ✓
describe(Blue)   -- Returns "Blue color" ✓
```

### Example 2: Status Enum (Production Use Case)

**Before (broken - must use workaround):**
```ailang
-- Can't use simple enum, need wrapper type
type Status = Pending(()) | InProgress(()) | Completed(())
--           ^^^^^^^^ ^^   ^^^^^^^^^^^^ ^^   ^^^^^^^^^^^ ^^
--           Ugly workarounds to force non-nullary
```

**After (fixed - clean enum):**
```ailang
-- Natural, idiomatic code
type Status = Pending | InProgress | Completed

func formatStatus(s: Status) -> string {
  match s {
    Pending => "⏳ Waiting",
    InProgress => "⚙️ Working",
    Completed => "✅ Done"
  }
}
```

## Testing Strategy

### 1. Unit Tests

**File**: `internal/eval/eval_test.go`

```go
func TestNullaryConstructorMatching(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        expected string
    }{
        {
            name: "nullary enum - first variant",
            code: `
                type Color = Red | Green | Blue
                func test() -> string {
                    match Red { Red => "R", Green => "G", Blue => "B" }
                }
                test()
            `,
            expected: "R",
        },
        {
            name: "nullary enum - second variant",
            code: `
                type Color = Red | Green | Blue
                func test() -> string {
                    match Green { Red => "R", Green => "G", Blue => "B" }
                }
                test()
            `,
            expected: "G",
        },
        {
            name: "nullary enum - third variant",
            code: `
                type Color = Red | Green | Blue
                func test() -> string {
                    match Blue { Red => "R", Green => "G", Blue => "B" }
                }
                test()
            `,
            expected: "B",
        },
        {
            name: "nullary enum - via parameter",
            code: `
                type Status = Pending | InProgress | Completed
                func describe(s: Status) -> string {
                    match s {
                        Pending => "Waiting",
                        InProgress => "Working",
                        Completed => "Done"
                    }
                }
                describe(InProgress)
            `,
            expected: "Working",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := evalCode(t, tt.code)
            if result != tt.expected {
                t.Errorf("expected %q, got %q", tt.expected, result)
            }
        })
    }
}
```

### 2. Integration Test

**File**: `tests/nullary_pattern_matching_test.ail`

```ailang
module tests/nullary_pattern_matching

type Status = Pending | InProgress | Completed

export func describeStatus(s: Status) -> string {
  match s {
    Pending => "Waiting",
    InProgress => "Working",
    Completed => "Done"
  }
}

export func main() -> () ! {IO} {
  -- Test all three variants
  assert(describeStatus(Pending) == "Waiting");
  assert(describeStatus(InProgress) == "Working");
  assert(describeStatus(Completed) == "Done");

  print("✓ All nullary pattern matching tests passed")
}
```

**Run with:**
```bash
ailang run --caps IO --entry main tests/nullary_pattern_matching_test.ail
```

### 3. Regression Tests

Ensure existing pattern matching still works:
```bash
# Option type (mixed nullary/non-nullary) - CRITICAL regression check
ailang run examples/option.ail

# List patterns (confirmed working in v0.4.4)
ailang run examples/list_operations.ail

# Complex pattern matching
ailang eval-suite --benchmarks pattern_matching_complex --langs ailang
```

### 4. Eval Benchmark

```bash
# Full benchmark suite
ailang eval-suite --benchmarks exhaustive_pattern_matching --langs ailang --models gpt5,claude-sonnet-4-5
```

**Expected improvement:**
- Current: 96.1% success (25/26 test cases)
- After fix: 100% success (26/26 test cases)

## Success Criteria

- [ ] All four unit tests in `TestNullaryConstructorMatching` pass
- [ ] Integration test `tests/nullary_pattern_matching_test.ail` passes
- [ ] `exhaustive_pattern_matching` benchmark: **96.1% → 100% success**
- [ ] Zero regressions: Existing pattern matching tests still pass
- [ ] `Option` type `None` variant still works correctly (CRITICAL)
- [ ] List pattern matching still works (`x :: rest`)
- [ ] Performance: No measurable regression (<1% on pattern match benchmarks)
- [ ] Documentation updated: CHANGELOG.md, docs/LIMITATIONS.md
- [ ] All tests passing: `make test`
- [ ] Code reviewed and approved

## Non-Goals

**Not in this fix:**
- Optimizing decision tree compilation performance
- Adding exhaustiveness checking for other ADT edge cases
- Improving pattern match error messages
- Supporting guard clauses in patterns (`x when x > 0 => ...`)
- Adding pattern match compilation metrics/debugging
- Implementing pattern match optimizations (fall-through, etc.)

## Timeline

**Week 1** (~3-5 hours total):

**Day 1** (1-2 hours):
- Investigation Phase 1-4 (2 hours)
- Identify bug location (confirmed)

**Day 2** (1-2 hours):
- Implement fix (1 hour)
- Write unit tests (30 min)
- Write integration test (15 min)
- Manual testing (15 min)

**Day 3** (1 hour):
- Run full test suite (20 min)
- Run eval benchmarks (20 min)
- Update documentation (10 min)
- Final code review (10 min)

**Total: 3-5 hours across 3 days (or 1 focused session)**

## Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|-----------|
| Fix breaks `Option` type `None` matching | **High** | Low | Test `None` explicitly in regression suite before merge |
| Fix breaks list pattern matching (`::`) | **High** | Low | Run list_operations.ail before and after fix |
| Decision tree optimization conflicts | Medium | Medium | Preserve existing optimizations, only fix tag test generation |
| Performance regression | Medium | Low | Benchmark pattern matching performance before/after |
| Fix incomplete (edge cases missed) | Medium | Medium | Test multiple nullary ADTs with different arity counts |
| Bug exists in multiple locations | Low | Low | Fix all locations found during investigation |

## References

- **Eval analysis**: `EVAL_ANALYSIS_v0_4_4.md` - Confirms list patterns work correctly
- **Pattern matching design**: `internal/dtree/README.md` - Decision tree compilation
- **Evaluator**: `internal/eval/eval.go` - Runtime matching logic
- **Elaboration**: `internal/elaborate/patterns.go` - Surface to Core lowering
- **Test cases**: `/tmp/test_none.ail`, `/tmp/test_debug.ail`
- **Related benchmarks**: `benchmarks/exhaustive_pattern_matching/`

## Future Work

After this fix ships:
- Consider adding exhaustiveness checking for other edge cases (wildcards, nested patterns)
- Improve pattern match error messages (show which patterns matched/didn't match)
- Add guard clauses support (`x when x > 0 => ...`)
- Pattern match compilation metrics for debugging
- Optimization: Detect simple enums and compile to switch statement