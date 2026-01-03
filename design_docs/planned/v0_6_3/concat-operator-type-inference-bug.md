# Concatenation Operator (`++`) Type Inference Bug

**Status**: Planned
**Target**: v0.6.3
**Priority**: P1 (High) - Blocks common list operations
**Estimated**: 4-8 hours
**Dependencies**: None
**Bug ID**: M-CONCAT-INFERENCE
**Master Doc**: [v0_6_3-bug-fixes.md](v0_6_3-bug-fixes.md)

> ⚠️ **UPDATE (v0.6.2)**: The original bug (defaulting to list when string intended)
> appears to have been over-corrected. The current bug is the **opposite**: `++` now
> defaults to **string concat** when **list concat** is intended. See examples below.

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes need for workarounds (let-binding intermediate results) |
| Preserve Semantic Clarity | 0 | 0 | No semantic changes - just fixes broken inference |
| Increase Determinism | + | +1 | Makes type inference deterministic in recursive contexts |
| Lower Token Cost | + | +1 | AI models can write natural recursive string code without workarounds |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

The `++` operator type inference incorrectly defaults to **list concatenation** when both operands are type variables in recursive contexts, causing type errors for perfectly valid string concatenation patterns.

**Current State:**
- `deterministic_list_transform` benchmark: **Fails with type error** (should pass)
- Error message: `cannot unify type constructor string with *types.TList`
- Affects recursive functions that build strings via concatenation
- AI models generate this pattern naturally (53.7% overall, but this specific pattern fails)
- No compile error - just wrong type inference leading to unification failure

**Root Cause:**
In `internal/types/typechecker_operators.go:160-163`, the type checker has this decision tree:
```go
// Decision tree for ++:
// 1. If at least one is a concrete list → list concat
// 2. If at least one is a concrete string → string concat
// 3. If both are type variables → list concat (more polymorphic) ← BUG!
// 4. Otherwise → string concat (fallback)
```

When type-checking recursive functions, the return type isn't yet resolved, so both operands appear as type variables → triggers rule #3 → wrong inference.

**Impact:**
- **Who affected**: AI code generators + developers writing recursive string functions
- **Severity**: Medium - Blocks common pattern but has workarounds
- **Workaround cost**: Requires let-binding intermediate results (+3-5 lines per function)
- **Blocks**: Natural recursive string building patterns that work in all other ML languages

### Reproduction

```ailang
module tmp/test_join

export func join(sep: string, xs: [int]) -> string {
  match xs {
    [] => "",
    x :: rest => show(x) ++ join(sep, rest)  -- Type error!
    --           ^^^^^^^^    ^^^^^^^^^^^^^^^^
    --           string      recursion → type var → defaults to list concat!
  }
}

export func main() -> () ! {IO} {
  print(join(", ", [1, 2, 3]))  -- Should work, but fails with:
  -- Error: type unification failed at [list concat left]:
  --        cannot unify type constructor string with *types.TList
}
```

**What works (workaround):**
```ailang
export func join(sep: string, xs: [int]) -> string {
  match xs {
    [] => "",
    x :: rest => {
      let h = show(x);        -- Force string type
      let t = join(sep, rest); -- Force string type
      h ++ sep ++ t            -- Now both are known strings
    }
  }
}
```

But even the workaround **still fails** because the type checker sees `t` (from recursive call) as a type variable!

**Comparison with working case:**
```ailang
-- This works (no recursion)
export func simpleConcat() -> string {
  "hello" ++ " " ++ "world"  -- All concrete strings
}

-- This works (recursion but type is known from signature)
export func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)  -- * operator works fine
}
```

**Pattern**: `++` fails when recursive call appears as operand before return type is fully resolved.

## Goals

**Primary Goal:** Fix `++` operator type inference to support recursive string concatenation patterns that work in all other functional languages.

**Success Metrics:**
- Reproduction test case: **Fails → Passes**
- `deterministic_list_transform` benchmark: **Compile error → Success**
- `string_manipulation` benchmark: **Still passes** (no regression)
- AI-generated recursive string code: **Type errors eliminated**
- Zero regressions: All existing `++` tests still pass

## Solution Design

### Overview

The bug is in the operator overload resolution heuristic. When both operands are type variables, the type checker defaults to list concatenation because "it's more polymorphic". This is wrong for recursive functions where the return type is still being inferred.

**Key insight:** We need to look at **type annotations** and **expected types** from context, not just the immediate operand types.

### Architecture

**Three potential fixes (ordered by complexity):**

#### Fix 1: Context-Aware Inference (Recommended)

Use the **expected return type** from the enclosing function to guide operator overload resolution.

```go
// In typechecker_operators.go, around line 160
case "++":
    // NEW: Check if we have an expected type from context
    expectedType := ctx.getExpectedType()  // From function signature

    if expectedType == TString {
        // We know from context this should be string concat
        resultType = TString
        ctx.addConstraint(TypeEq{Left: leftType, Right: TString, ...})
        ctx.addConstraint(TypeEq{Left: rightType, Right: TString, ...})
    } else if leftIsList || rightIsList {
        // At least one is concrete list → list concat
        resultType = &TList{Element: ctx.freshTypeVar()}
        // ... existing list logic
    } else if leftIsString || rightIsString {
        // At least one is concrete string → string concat
        resultType = TString
        // ... existing string logic
    } else {
        // CHANGED: Default to string concat (less surprising)
        resultType = TString
        ctx.addConstraint(TypeEq{Left: leftType, Right: TString, ...})
        ctx.addConstraint(TypeEq{Right: rightType, Right: TString, ...})
    }
```

**Benefits:**
- Fixes the immediate bug
- Leverages existing type annotation information
- Minimal changes to existing code
- Aligns with how other operators work

**Drawbacks:**
- Requires threading expected type through type checker
- May need refactoring of type checking context

#### Fix 2: Change Default (Simple but Risky)

Just change the default from list concat to string concat when both are type variables.

```go
// Around line 160-163
} else if leftIsVar && rightIsVar {
    // CHANGED: Default to string concat (more common)
    resultType = TString
    ctx.addConstraint(TypeEq{Left: leftType, Right: TString, ...})
    ctx.addConstraint(TypeEq{Right: rightType, Right: TString, ...})
}
```

**Benefits:**
- Minimal code change (1 line)
- Fixes the bug for common case

**Drawbacks:**
- **May break existing polymorphic list code** if it relies on this default
- Doesn't address root cause (lack of context awareness)
- Could cause new failures in other benchmarks

#### Fix 3: Bidirectional Type Checking (Complete but Complex)

Implement full bidirectional type checking where expected types flow through the entire expression tree.

**Benefits:**
- Most correct solution
- Fixes many type inference edge cases

**Drawbacks:**
- Major refactoring of type checker
- 2-4 weeks of work
- High risk of regressions
- Out of scope for this bug fix

### Recommended Approach: Fix 1 (Context-Aware Inference)

**Implementation steps:**

**Phase 1: Add Expected Type Threading** (~2-3 hours)
- Add `expectedType` field to type checking context
- Thread expected type through function bodies
- Extract expected type from function signature return type

**Phase 2: Update `++` Operator Logic** (~1-2 hours)
- Check expected type before defaulting
- Prefer string concat when expected type is string
- Fall back to existing logic if no expected type

**Phase 3: Testing & Validation** (~2-3 hours)
- Write unit tests for recursive string concat
- Run existing `++` tests (ensure no regressions)
- Test `deterministic_list_transform` benchmark
- Test edge cases (polymorphic functions, nested recursion)

### Implementation Plan

**Phase 1: Add Expected Type Context** (~2-3 hours)
- [ ] Add `expectedType *Type` field to `TypeCheckContext`
- [ ] Add `pushExpectedType()` and `popExpectedType()` methods
- [ ] Thread expected type in `checkFuncDecl()` from return type annotation
- [ ] Thread expected type through `checkExpr()` recursion

**Phase 2: Fix `++` Operator** (~1-2 hours)
- [ ] Modify `case "++"` in `typechecker_operators.go`
- [ ] Check `ctx.expectedType` before defaulting
- [ ] Prefer string concat when expected type is string
- [ ] Add debug logging to verify fix

**Phase 3: Testing** (~2-3 hours)
- [ ] Write unit test for recursive string concat
- [ ] Write unit test for list concat (ensure not broken)
- [ ] Run existing operator tests
- [ ] Test `deterministic_list_transform` benchmark
- [ ] Test `string_manipulation` benchmark (regression check)

**Phase 4: Documentation** (~30 min)
- [ ] Update CHANGELOG.md
- [ ] Remove limitation from prompt if documented
- [ ] Update this design doc with implementation notes

### Files to Modify/Create

**Modified files:**
- `internal/types/typechecker_operators.go` (~15 LOC) - Fix `++` operator logic
- `internal/types/typechecker.go` (~30 LOC) - Add expected type threading
- `internal/types/context.go` (~20 LOC) - Add expected type fields/methods

**New test files:**
- `internal/types/typechecker_concat_test.go` (~80 LOC) - Test recursive concat
- `tests/recursive_string_concat.ail` (~30 LOC) - Integration test

**Documentation:**
- `CHANGELOG.md` (+10 LOC) - Document fix
- `prompts/v0.4.5.md` (if limitation exists) - Remove limitation

**Total estimated changes:** ~185 LOC

## Examples

### Example 1: Recursive String Join (Currently Broken)

**Before (broken):**
```ailang
module benchmark/solution

export func join(sep: string, xs: [int]) -> string {
  match xs {
    [] => "",
    x :: rest => show(x) ++ sep ++ join(sep, rest)
    --           ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    --           Type error: defaults to list concat!
  }
}

-- Error: type unification failed at [list concat left]:
--        cannot unify type constructor string with *types.TList
```

**After (fixed):**
```ailang
module benchmark/solution

export func join(sep: string, xs: [int]) -> string {
  match xs {
    [] => "",
    x :: rest => show(x) ++ sep ++ join(sep, rest)  -- ✓ Works!
    --           ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    --           Knows return type is string from signature
  }
}

-- Output: "1, 2, 3" ✓
```

### Example 2: Deterministic List Transform (Benchmark)

**Before (broken):**
```ailang
module benchmark/solution

import std/io (println)

export func map[a, b](f: a -> b, xs: [a]) -> [b] {
  match xs {
    [] => [],
    x :: rest => f(x) :: map(f, rest)  -- ✓ Works (no ++)
  }
}

export func join(sep: string, xs: [int]) -> string {
  match xs {
    [] => "",
    [x] => show(x),
    x :: rest => show(x) ++ sep ++ join(sep, rest)  -- ❌ Type error!
  }
}

export func main() -> () ! {IO} {
  let numbers = [1, 2, 3, 4, 5];
  let double = func(x: int) -> int { x * 2 };
  let doubled = map(double, numbers);
  let result = join(", ", doubled);  -- Compile error here
  println(result)
}
```

**After (fixed):**
```ailang
module benchmark/solution

import std/io (println)

export func map[a, b](f: a -> b, xs: [a]) -> [b] {
  match xs {
    [] => [],
    x :: rest => f(x) :: map(f, rest)  -- ✓ Still works
  }
}

export func join(sep: string, xs: [int]) -> string {
  match xs {
    [] => "",
    [x] => show(x),
    x :: rest => show(x) ++ sep ++ join(sep, rest)  -- ✓ Fixed!
  }
}

export func main() -> () ! {IO} {
  let numbers = [1, 2, 3, 4, 5];
  let double = func(x: int) -> int { x * 2 };
  let doubled = map(double, numbers);
  let result = join(", ", doubled);
  println(result)  -- Output: "2, 4, 6, 8, 10"
}
```

### Example 3: List Concat Still Works (Regression Test)

**After fix (should still work):**
```ailang
-- List concatenation with type variables should still work
export func appendLists[a](xs: [a], ys: [a]) -> [a] {
  match xs {
    [] => ys,
    x :: rest => x :: appendLists(rest, ys)  -- Uses ::, not ++
  }
}

-- Explicit list concat with ++
export func concatLists[a](xs: [a], ys: [a]) -> [a] {
  xs ++ ys  -- Should infer as list concat (both have list type from signature)
}
```

## Success Criteria

- [ ] Reproduction test case (`join` function) compiles and runs correctly
- [ ] `deterministic_list_transform` benchmark passes (currently fails)
- [ ] `string_manipulation` benchmark still passes (regression check)
- [ ] List concatenation tests still pass (no regressions)
- [ ] All existing operator tests passing
- [ ] Unit tests for recursive string concat added
- [ ] Integration test added
- [ ] Documentation updated (CHANGELOG, prompt if needed)
- [ ] All tests passing: `make test`

## Testing Strategy

**Unit tests** (`internal/types/typechecker_concat_test.go`):
```go
func TestConcatOperatorInference(t *testing.T) {
    tests := []struct {
        name        string
        code        string
        shouldError bool
    }{
        {
            name: "recursive string concat",
            code: `
                func join(sep: string, xs: [int]) -> string {
                    match xs {
                        [] => "",
                        x :: rest => show(x) ++ join(sep, rest)
                    }
                }
            `,
            shouldError: false,
        },
        {
            name: "list concat with type vars",
            code: `
                func appendLists[a](xs: [a], ys: [a]) -> [a] {
                    xs ++ ys
                }
            `,
            shouldError: false,
        },
        {
            name: "mixed string and list",
            code: `
                func broken(s: string, xs: [int]) -> string {
                    s ++ xs  -- Should error: can't concat string + list
                }
            `,
            shouldError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := typeCheckCode(tt.code)
            if tt.shouldError && err == nil {
                t.Error("expected error but got none")
            }
            if !tt.shouldError && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
```

**Integration tests:**
```bash
# Run specific benchmark that was failing
ailang eval-suite --benchmarks deterministic_list_transform --langs ailang

# Run all string-related benchmarks
ailang eval-suite --benchmarks string_manipulation,list_operations --langs ailang

# Full test suite
make test
```

**Manual testing:**
```bash
# Test the reproduction case
cat > /tmp/test_concat_fix.ail <<EOF
module tmp/test_concat_fix

export func join(sep: string, xs: [int]) -> string {
  match xs {
    [] => "",
    x :: rest => show(x) ++ join(sep, rest)
  }
}

export func main() -> () ! {IO} {
  print(join(", ", [1, 2, 3, 4, 5]))
}
EOF

ailang run --caps IO --entry main /tmp/test_concat_fix.ail
# Expected output: 1, 2, 3, 4, 5
```

## Non-Goals

**Not in this fix:**
- Full bidirectional type checking - Too large, deferred to future work
- Type class-based operator overloading - Different design approach
- Better error messages for `++` type errors - Separate issue
- Performance optimization of type inference - Not affected by this bug
- Supporting `++` for other types (custom types) - Out of scope

## Timeline

**Day 1** (3-4 hours):
- Phase 1: Add expected type threading
- Phase 2: Fix `++` operator logic
- Manual testing

**Day 2** (2-3 hours):
- Phase 3: Write unit and integration tests
- Run full test suite
- Run benchmarks
- Fix any regressions

**Day 3** (1-2 hours):
- Phase 4: Documentation
- Code review
- Final testing
- Commit and PR

**Total: ~6-9 hours across 2-3 days (or 1 focused session)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking list concat with type variables | High | Comprehensive test suite for list `++` before merging |
| Expected type threading causes regressions | Medium | Add tests for all operator types, not just `++` |
| Performance regression from context threading | Low | Benchmark type checking performance before/after |
| Edge cases in nested recursion | Medium | Test deeply nested recursive functions |
| Fix incomplete (other operators affected) | Low | Focus only on `++`, defer other operators |

## References

- **Bug discovery**: Analysis during prompt v0.4.4 testing
- **Root cause**: `internal/types/typechecker_operators.go:160-163`
- **Affected benchmark**: `benchmarks/deterministic_list_transform/`
- **Prompt update**: `prompts/v0.4.4.md` (list cons syntax works, but `++` fails)
- **Related issues**: M-BUG-NULLARY (separate pattern matching bug)
- **Type system design**: `design_docs/implemented/v0_3/type_system.md`

## Future Work

After this fix ships:
- Consider full bidirectional type checking (better inference overall)
- Improve error messages for operator type mismatches
- Add type class-based operator overloading (extensible `++`)
- Investigate other operators that might have similar issues (`*`, `+`, etc.)
- Add operator overload resolution debugging (show why a particular overload was chosen)

---

**Document created**: 2025-11-16
**Last updated**: 2025-11-16
