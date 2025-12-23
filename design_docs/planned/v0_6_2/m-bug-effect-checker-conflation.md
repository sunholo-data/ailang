# M-BUG-EFFECT-CHECKER-CONFLATION: Effect Checker Incorrectly Requires IO for Pure Functions

**Status:** Planned
**Target:** v0.6.2
**Priority:** P1 (Medium - breaks valid examples)
**Estimated:** 4 hours
**Dependencies:** None
**Created:** 2025-12-23
**Last Updated:** 2025-12-23

## Problem Statement

The effect checker incorrectly requires IO effects for pure functions when they are called inside a `println` that appears AFTER another `println` in the same block.

### Reproduction

This passes:
```ailang
export func main() -> () ! {IO} {
  let nums = [1, 2, 3, 4, 5];
  println("sum = " ++ show(sum(nums)));  -- Works fine
  ()
}
```

This FAILS with spurious error:
```ailang
export func main() -> () ! {IO} {
  println("Header");                      -- Adding this line...
  let nums = [1, 2, 3, 4, 5];
  println("sum = " ++ show(sum(nums)));  -- ...causes this to fail!
  ()
}
```

**Error message:**
```
Error: effect checking failed: Effect checking failed for function 'sum'
  Function uses effects not declared in signature

  Missing effects: IO

  Current signature: func sum(...) -> T
  Suggested fix:     func sum(...) -> T ! {IO}
```

### Root Cause Analysis

The effect checker appears to be conflating the IO effect from the outer `println` context with the inner pure function call when:
1. There is a `println` statement earlier in the block
2. A subsequent `println` contains a call to a pure function via `show(pureFunc(args))`

The effect analysis is incorrectly propagating IO requirements to the pure function, even though:
- The pure function is correctly declared as `pure func`
- The IO effect comes from `println`, not from `sum`
- The `show` builtin is also pure

### Impact

- **Affected example:** `examples/runnable/pattern_sugar.ail` fails verification
- **User impact:** Valid code rejected by compiler
- **Workaround:** Reorder statements (remove leading `println`) - not ideal

## Goals

**Primary Goal:** Fix effect checker to correctly handle pure function calls inside effectful contexts.

**Success Metrics:**
- [ ] `examples/runnable/pattern_sugar.ail` passes without modification
- [ ] All existing effect-checking tests continue to pass
- [ ] No regression in effect inference for legitimate cases
- [ ] Clear test case added for this specific bug pattern

## Solution Design

### Hypothesis

The bug is likely in how effect analysis handles nested expressions within effectful blocks. Possible locations:

1. **Effect propagation in block expressions** (`internal/types/effects.go` or similar)
   - When analyzing `{ println("foo"); println(show(f(x))) }`, the IO effect from the first statement may be incorrectly attached to the function `f` in the second statement

2. **Effect inference for function calls** (`internal/types/infer.go`)
   - The inference may not properly isolate the effect context of nested calls

3. **Effect unification** (`internal/types/unify.go`)
   - Effect constraints from different call sites may be incorrectly merged

### Investigation Steps

1. Add `DEBUG_EFFECTS=1` environment variable support for effect checking traces
2. Compare traces between passing case (no leading println) and failing case
3. Identify where IO effect incorrectly flows into pure function's effect set
4. Fix the specific propagation/unification bug

### Implementation Plan

**Phase 1: Diagnosis** (~1.5 hours)
- [ ] Add debug logging to effect checker
- [ ] Create minimal reproduction test case
- [ ] Trace effect flow in passing vs failing cases
- [ ] Identify exact code location of bug

**Phase 2: Fix** (~1.5 hours)
- [ ] Implement fix (likely scoping/isolation issue)
- [ ] Add regression test
- [ ] Verify pattern_sugar.ail passes

**Phase 3: Verification** (~1 hour)
- [ ] Run full test suite
- [ ] Run example verification
- [ ] Update pattern_sugar.ail if any workarounds were added

## Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/types/effects.go` | Fix effect propagation | ~20 |
| `internal/types/effects_test.go` | Add regression test | ~40 |
| TBD based on diagnosis | | |

## Test Cases

### Regression Test
```go
func TestEffectChecker_PureFunctionInNestedPrintln(t *testing.T) {
    src := `
module test/effect_nested

import std/io (println)

pure func sum(xs: List[int]) -> int {
  match xs {
    x :: rest => x + sum(rest),
    [] => 0
  }
}

export func main() -> () ! {IO} {
  println("Header");
  let nums = [1, 2, 3];
  println("sum = " ++ show(sum(nums)));
  ()
}
`
    // Should pass without effect errors
    result := compileAndCheck(src)
    assert.NoError(t, result.Err)
}
```

## Success Criteria

- [ ] Minimal test case passes
- [ ] `pattern_sugar.ail` example passes
- [ ] All existing tests pass
- [ ] No new warnings in effect checker
- [ ] Debug flag added for future effect debugging

## Related Documents

- [M-BUG-RECURSION-DEPTH](../../implemented/v0_4_8/m-bug-recursion-depth.md) - Similar compiler bug fix pattern
- [M-PARSER-NESTED-MATCH](../../archive/v0_4_1_m-parser-nested-match-delimiter-fix.md) - Related nested context bug

## Discovery Context

This bug was discovered while investigating why `examples/runnable/pattern_sugar.ail` was failing verification. Through binary search of the file contents, the trigger was isolated to the presence of a `println` statement before another `println` containing a pure function call.

**Key observation:** The bug only manifests when both conditions are true:
1. A `println` appears earlier in the block
2. A later `println` contains `show(pureFn(args))`

Neither condition alone triggers the bug.
