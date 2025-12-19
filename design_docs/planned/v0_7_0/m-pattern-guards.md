# M-PATTERN-GUARDS: Evaluate Pattern Guard Conditions

**Status**: Planned
**Target**: v0.7.0
**Priority**: P2 (Medium) - Parsed but not functional
**Estimated Time**: 4-6 hours
**Dependencies**: None

## Problem Statement

Pattern guards are syntactically supported and parsed, but the guard conditions are not evaluated during pattern matching. The first structurally matching pattern always wins, ignoring the guard.

```ailang
match 5 {
  x if x > 100 => "over 100",  -- 5 > 100 is false, should skip
  x if x > 0 => "positive",    -- 5 > 0 is true, should match
  x => "other"
}
-- Expected: "positive"
-- Actual: "over 100" (first pattern matches structurally, guard ignored)
```

## Current Behavior

1. Parser correctly parses `pattern if condition => body` syntax
2. AST includes the guard expression
3. Pattern matching only checks structural match, ignores guard
4. First structurally matching arm is selected

## Proposed Solution

### Implementation Steps

1. **Decision Tree Compilation** (`internal/dtree/`):
   - When compiling pattern arms with guards, generate conditional branches
   - After structural match succeeds, evaluate guard expression
   - If guard is false, continue to next arm

2. **Core Evaluator** (`internal/eval/`):
   - Extend match evaluation to handle guard expressions
   - Guard expression has access to pattern bindings (e.g., `x` in `x if x > 0`)

3. **Type Checking**:
   - Guard expression must have type `bool`
   - Guard can reference pattern-bound variables

### Example Lowering

```ailang
match val {
  x if x > 0 => "positive",
  x => "other"
}
```

Conceptually becomes:
```ailang
let x = val in
if x > 0 then "positive" else "other"
```

For multiple guards:
```ailang
match val {
  x if x > 100 => "big",
  x if x > 0 => "positive",
  x => "other"
}
```

Becomes:
```ailang
let x = val in
if x > 100 then "big"
else if x > 0 then "positive"
else "other"
```

## Success Criteria

1. `examples/runnable/guards_basic.ail` produces correct output
2. Guards with pattern bindings work: `(x, y) if x > y => ...`
3. Guards with effectful expressions are rejected (must be pure)
4. Type errors for non-bool guards

## Test Cases

```ailang
-- Basic guard
match 5 {
  x if x > 0 => "positive",
  x => "non-positive"
}
-- Expected: "positive"

-- Multiple guards
match -3 {
  x if x > 0 => "positive",
  x if x == 0 => "zero",
  x if x < 0 => "negative"
}
-- Expected: "negative"

-- Guard with tuple binding
match (10, 5) {
  (a, b) if a > b => "first bigger",
  (a, b) if a < b => "second bigger",
  _ => "equal"
}
-- Expected: "first bigger"
```

## References

- [Limitations doc](/docs/reference/limitations#pattern-guards-parsed-but-not-evaluated)
- Example file: `examples/runnable/guards_basic.ail`
- Pattern matching compilation: [Maranget 2008](https://www.cs.tufts.edu/~nr/cs257/archive/luc-maranget/jun08.pdf)

---

## Website Links

**Update these when this feature is implemented:**
- [Limitations page](/docs/reference/limitations) — Remove from limitations list
- [Implementation Status](/docs/reference/implementation-status) — Update status
- Move this doc from `planned/` to `implemented/`
