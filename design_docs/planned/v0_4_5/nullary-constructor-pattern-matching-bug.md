# Nullary Constructor Pattern Matching Bug (M-BUG-NULLARY)

**Status**: Planned  
**Version**: v0.4.5  
**Priority**: High (blocks eval benchmarks)  
**Complexity**: Medium  
**Est. Time**: 3-5 hours

## Problem Statement

Pattern matching fails when an ADT has **only nullary constructors** (constructors with zero arguments). All values match the first pattern regardless of which constructor was used.

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

### What Works

```ailang
-- Mixed constructors work correctly
type MyOption = MySome(int) | MyNone

func testOption(opt: MyOption) -> string {
  match opt {
    MySome(x) => "Some: " ++ show(x),
    MyNone => "None"
  }
}

testOption(MySome(42))  -- ✓ Returns "Some: 42"
testOption(MyNone)      -- ✓ Returns "None"
```

**Pattern**: Works when ADT has at least one constructor with arguments.

## Root Cause Analysis

The bug likely exists in the decision tree compilation or pattern matching lowering phase:

### Hypothesis 1: Tag Comparison Missing

Nullary constructors are represented as simple tagged values with no payload. The compiler may be:
1. Generating correct tag values (show() displays them correctly)
2. But **not emitting tag comparison code** in the match expression
3. Defaulting to first pattern when no tag comparison exists

### Hypothesis 2: Optimization Bug

The compiler might be applying an incorrect optimization:
- Assumes all nullary constructors are identical (no data to distinguish)
- Skips tag checking entirely
- Always matches first pattern

## Investigation Steps

1. **Check Core AST**: Examine how nullary constructors are elaborated to Core
   ```bash
   ailang debug ast /tmp/test_enum.ail --show-types
   ```

2. **Check Decision Tree**: Verify decision tree compilation handles nullary patterns
   ```bash
   grep -r "nullary\|arity.*0" internal/dtree/
   ```

3. **Check Evaluator**: Verify runtime value representation and matching
   ```bash
   grep -r "MatchVariant\|Tag" internal/eval/
   ```

4. **Add Debug Logging**: Instrument pattern matching codegen
   ```go
   // In internal/dtree/ or internal/elaborate/
   if variant.Arity == 0 {
       log.Printf("DEBUG: Nullary variant %s tag=%d", variant.Name, variant.Tag)
   }
   ```

## Likely Fix Locations

### 1. Decision Tree Compilation (`internal/dtree/`)

Check if nullary patterns are handled correctly:

```go
// File: internal/dtree/compile.go (or similar)
func compilePattern(pat Pattern) DecisionTree {
    switch p := pat.(type) {
    case VariantPattern:
        if p.Arity == 0 {
            // BUG: May be missing tag comparison here
            // FIX: Emit tag comparison even when arity=0
            return TagTest{
                Tag:      p.Tag,
                ThenTree: compileSubPatterns(p.Subpatterns),
            }
        }
        // ...
    }
}
```

### 2. Pattern Lowering (`internal/elaborate/`)

Check elaboration from Surface AST to Core:

```go
// File: internal/elaborate/patterns.go (or similar)
func (e *Elaborator) elaboratePattern(pat *ast.Pattern) *core.Pattern {
    switch p := pat.Kind.(type) {
    case *ast.VariantPattern:
        variant := e.lookupVariant(p.Name)
        if len(p.Args) == 0 && variant.Arity == 0 {
            // BUG: May be creating wrong Core pattern
            // FIX: Ensure tag is preserved
            return &core.PatternVariant{
                Tag:   variant.Tag,
                Arity: 0,
                Args:  nil,  // Important: empty, not omitted
            }
        }
    }
}
```

### 3. Runtime Evaluation (`internal/eval/`)

Check if runtime matching handles nullary variants:

```go
// File: internal/eval/eval.go
func matchPattern(value Value, pattern Pattern) (bool, Bindings) {
    switch v := value.(type) {
    case VariantValue:
        if pat, ok := pattern.(VariantPattern); ok {
            // BUG: May not be comparing tags for arity=0
            if v.Tag != pat.Tag {
                return false, nil  // FIX: Add this check!
            }
            if pat.Arity == 0 {
                return true, emptyBindings()
            }
            // ... handle arguments
        }
    }
}
```

## Testing Strategy

### 1. Add Unit Tests

```go
// File: internal/eval/eval_test.go
func TestNullaryConstructorMatching(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        expected string
    }{
        {
            name: "simple enum - all nullary",
            code: `
                type Color = Red | Green | Blue
                func test() -> string {
                    match Red { Red => "R", Green => "G", Blue => "B" }
                }
            `,
            expected: "R",
        },
        {
            name: "enum second variant",
            code: `
                type Color = Red | Green | Blue
                func test() -> string {
                    match Green { Red => "R", Green => "G", Blue => "B" }
                }
            `,
            expected: "G",
        },
        {
            name: "enum third variant",
            code: `
                type Color = Red | Green | Blue
                func test() -> string {
                    match Blue { Red => "R", Green => "G", Blue => "B" }
                }
            `,
            expected: "B",
        },
    }
    // ...
}
```

### 2. Add Integration Test

```ailang
// File: tests/nullary_pattern_matching_test.ail
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
  assert(describeStatus(Pending) == "Waiting");
  assert(describeStatus(InProgress) == "Working");
  assert(describeStatus(Completed) == "Done");
  print("All tests passed")
}
```

### 3. Update Benchmarks

Once fixed, `exhaustive_pattern_matching` benchmark should pass:

```bash
ailang eval-suite --benchmarks exhaustive_pattern_matching --langs ailang
```

## Success Criteria

- [ ] All three test cases in `TestNullaryConstructorMatching` pass
- [ ] Integration test `tests/nullary_pattern_matching_test.ail` passes
- [ ] `exhaustive_pattern_matching` benchmark passes with both Claude models
- [ ] Existing tests still pass (no regressions)
- [ ] `None` matching still works correctly in `Option` types

## Implementation Notes

**Timeline**:
- Investigation: 1-2 hours
- Fix: 1-2 hours  
- Testing: 1 hour
- Total: 3-5 hours

**Dependencies**: None (standalone bug fix)

**Blockers**: None

**Related Issues**:
- Blocks: `exhaustive_pattern_matching` eval benchmark (96.1% → 100% success)
- Related: Decision tree compilation (M-DX7)

## References

- Decision tree docs: `internal/dtree/README.md`
- Pattern matching elaboration: `internal/elaborate/patterns.go`
- Evaluator: `internal/eval/eval.go`
- Test cases: `/tmp/test_none.ail`, `/tmp/test_debug.ail`
