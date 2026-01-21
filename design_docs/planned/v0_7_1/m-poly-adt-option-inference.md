# M-POLY-ADT: Polymorphic ADT Type Inference with Option

## Status
- **Priority**: High (affects error handling patterns)
- **Discovered**: 2026-01-06 via eval gap analysis
- **Affects**: Result types, Either types, any polymorphic ADT used with Option

## Problem Statement

Polymorphic ADT types like `Result[a]` fail to unify correctly when pattern matching against `Option[int]` from stdlib functions like `stringToInt`.

### Minimal Reproduction

```ailang
module test
import std/string (stringToInt)

type Result[a] = Ok(a) | Err(string)

-- This FAILS with: "cannot unify type constructors: string vs int"
pure func parseIntResult(s: string) -> Result[int] =
  match stringToInt(s) {
    Some(n) => Ok(n),
    None => Err("Invalid integer")
  }
```

### Error Message

```
Error: type error in test (decl 0): type unification failed at
  [return type annotation at test.ail:6:6]:
  failed to unify type argument 0: cannot unify type constructors: string vs int
```

### Working Workaround

Use monomorphic types instead:

```ailang
-- This WORKS
type IntResult = IntOk(int) | IntErr(string)

pure func parseIntResult(s: string) -> IntResult =
  match stringToInt(s) {
    Some(n) => IntOk(n),
    None => IntErr("Invalid integer")
  }
```

## Impact

- **Error handling patterns**: Can't use standard `Result[a]` pattern
- **API consistency**: Forces verbose type definitions per concrete type
- **Agent benchmarks**: `error_handling` benchmark fails due to this issue

## Root Cause Analysis

### Hypothesis 1: Polymorphic ADT Instantiation

The type checker may not be correctly instantiating type variables when:
1. Matching against `Option[int]` (from stringToInt)
2. Constructing `Result[int]` (user-defined polymorphic ADT)

The unification error "string vs int" suggests the type checker is confusing:
- The `string` from `Err(string)` variant
- The `int` from `Result[int]` type argument

### Hypothesis 2: Type Variable Scoping

The type variable `a` in `Result[a]` may be conflicting with internal type variables used during Option pattern matching.

### Files to Investigate

1. `internal/types/unify.go` - Unification algorithm
2. `internal/types/infer.go` - Type inference for match expressions
3. `internal/elaborate/elaborate.go` - ADT instantiation during elaboration

## Proposed Fix

### Option A: Fix Unification (Preferred)

Ensure type variables in user-defined polymorphic ADTs are properly scoped and instantiated during match expression type checking.

**Steps:**
1. Add test case reproducing the bug in `internal/types/*_test.go`
2. Debug unification trace to identify where "string vs int" error originates
3. Fix scoping issue in type variable instantiation
4. Verify with `error_handling` benchmark

### Option B: Document Limitation

If fix is complex, document as known limitation and recommend monomorphic types.

## Test Cases

```go
// In internal/types/infer_test.go
func TestPolymorphicADTWithOption(t *testing.T) {
    src := `
    import std/string (stringToInt)
    type Result[a] = Ok(a) | Err(string)
    pure func parse(s: string) -> Result[int] =
      match stringToInt(s) {
        Some(n) => Ok(n),
        None => Err("bad")
      }
    `
    // Should type check successfully
}
```

## Success Criteria

1. Minimal reproduction compiles without error
2. `error_handling` benchmark passes with standard `Result[a]` pattern
3. No regression in existing polymorphic ADT tests

## References

- Prompt workaround: `prompts/v0.6.5.md` line 498 (Note about monomorphic types)
- Benchmark: `benchmarks/error_handling.yml`
- Eval results: `eval_results/v0.6.5-g3-haiku/` (error_handling failures)
