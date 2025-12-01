# M-BUG-ADT-TEST-HARNESS-SCOPE: ADT Constructor Resolution in Test Harness

## Status: Planned (v0.5.0)

## Problem Statement

The test harness cannot resolve ADT constructors defined in the same module. When using ADT constructors in `tests` declarations, the harness reports "undefined variable" for constructor names.

**Reported by:** stapledons_voyage (via agent inbox, 2025-12-01)
**Note:** This is a follow-up to a previous panic bug that has been fixed. The constructors no longer cause panics, but scope resolution fails.

## Error Message

```
undefined variable: North
```

## Reproduction

### Minimal Failing Case
```ailang
module game/direction

export type Direction = North | South | East | West

pure func directionDx(dir: Direction) -> int
tests [(North, 0), (South, 0), (East, 1), (West, -1)]
{
    match dir {
        North => 0,
        South => 0,
        East => 1,
        West => -1
    }
}

export pure func directionDy(dir: Direction) -> int
tests [(North, -1), (South, 1), (East, 0), (West, 0)]
{
    match dir {
        North => -1,
        South => 1,
        East => 0,
        West => 0
    }
}
```

Running tests on this module:
```bash
ailang test game/direction.ail
# Error: undefined variable: North
```

### What Works
- ADT constructors work in match expressions within the function body
- ADT constructors work when called from other modules (after import)
- Function execution works correctly

### What Fails
- ADT constructors in `tests [(Constructor, expected), ...]` declarations
- Only affects test harness evaluation, not regular execution

## Analysis

### Root Cause Hypothesis

The test harness evaluates test case expressions in a scope that doesn't include:
1. ADT constructors from the same module
2. Possibly other module-level bindings

When the test harness parses `tests [(North, 0), ...]`, it tries to evaluate `North` as an expression but:
- The constructor is not in scope for the test expression context
- The harness may be using a minimal/isolated evaluation environment

### Related Previous Fix

A previous bug caused the test harness to **panic** when encountering ADT constructors:
> "Tests no longer panic (fix confirmed!) but fail with `undefined variable: North`"

The panic was likely due to missing case handling in `astExprToCore` (line 201) for ADT constructor identifiers. That fix made it not crash, but scope resolution was not addressed.

## Investigation Areas

1. **`internal/eval_harness/harness.go`** - Test execution context setup
2. **`internal/eval_harness/runner.go`** - Test case evaluation
3. **`internal/elaborate/file.go`** - How ADT constructors enter scope
4. **`internal/loader/module.go`** - Module loading and scope construction

### Key Questions

1. How does the test harness construct its evaluation scope?
2. Are ADT constructors added to the scope before test expression evaluation?
3. Is there a separate elaboration pass for test expressions?
4. Does the test harness use the module's full scope or a subset?

## Proposed Fix

### Phase 1: Diagnosis
- Add debug logging to test harness scope construction
- Compare scope available during function body vs test expression evaluation
- Trace ADT constructor registration in module scope

### Phase 2: Fix
Likely solutions:
1. **Expand test harness scope**: Include ADT constructors from the same module in test expression evaluation context
2. **Pre-elaborate test expressions**: Run test expressions through full elaboration with module scope before evaluation
3. **Inject constructors**: Explicitly add ADT constructor bindings to test harness environment

### Implementation Notes

The test harness likely needs to:
```go
// Pseudocode for fix
func (h *Harness) evaluateTestCase(module *Module, testCase TestCase) {
    scope := module.GetFullScope()  // Include ADT constructors
    // OR
    scope := h.baseScope.ExtendWith(module.ADTConstructors)

    inputValue := h.evaluate(testCase.Input, scope)
    expectedValue := h.evaluate(testCase.Expected, scope)
    // ...
}
```

## Impact

**Medium** - Blocks test-driven development for any code using ADTs, which is common in game engines and domain modeling.

## Test Cases

```ailang
-- test_adt_harness.ail
module test/adt_harness

-- Case 1: Simple ADT in tests
type Color = Red | Green | Blue

pure func colorCode(c: Color) -> int
tests [(Red, 0), (Green, 1), (Blue, 2)]
{
    match c {
        Red => 0,
        Green => 1,
        Blue => 2
    }
}

-- Case 2: ADT with data in tests
type Maybe[a] = Just(a) | Nothing

pure func fromMaybe(def: int, m: Maybe[int]) -> int
tests [(5, Nothing, 5), (5, Just(10), 10)]
{
    match m {
        Just(x) => x,
        Nothing => def
    }
}

-- Case 3: Nested ADT constructors
type Tree = Leaf(int) | Node(Tree, Tree)

pure func treeSum(t: Tree) -> int
tests [(Leaf(5), 5), (Node(Leaf(1), Leaf(2)), 3)]
{
    match t {
        Leaf(n) => n,
        Node(l, r) => treeSum(l) + treeSum(r)
    }
}
```

## Workarounds

### Current Workaround: Separate Test File
```ailang
-- game/direction.ail (no tests declaration)
module game/direction
export type Direction = North | South | East | West
export pure func directionDx(dir: Direction) -> int { ... }

-- tests/direction_test.ail
module tests/direction_test
import game/direction (Direction, North, South, East, West, directionDx)

-- Manual test calls
let test1 = directionDx(North) == 0
let test2 = directionDx(East) == 1
-- ...
```

This is verbose and loses the benefit of inline test declarations.

## Related Issues

- Previous panic fix for ADT constructors in test harness
- M-BUG-MODULE-LET-SCOPE (v0.4.9) - Similar scope resolution issue for let bindings
- Test harness architecture

## References

- Stapledons message: msg_20251201_145005_2827a1f4db2b
- Previous context: "Tests no longer panic (fix confirmed!)"
