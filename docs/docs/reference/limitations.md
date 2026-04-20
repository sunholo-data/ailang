---
sidebar_position: 5
title: Known Limitations
description: Known limitations, workarounds, and design constraints in AILANG
---

# AILANG Known Limitations

This document tracks known limitations, workarounds, and design constraints in AILANG.

For features that have been implemented, see [Design Documents](/docs/design-docs).

---

## Type System Limitations

### Y-Combinator and Recursive Lambdas (By Design)

**Status**: Design constraint, not a bug

The Y-combinator and similar recursive lambda expressions fail with "occurs check" errors:

```ailang
-- This fails:
let Y = \f. (\x. f(x(x)))(\x. f(x(x))) in
-- Error: occurs check failed: type variable α occurs in (α → β)
```

**Root Cause**: Hindley-Milner type inference prevents infinite types to ensure decidability. The Y-combinator requires a recursive type `α = α → β`, which would create an infinite type.

**Why This Exists**:
1. **Type Inference Decidability** — Allowing infinite types makes type inference undecidable
2. **AI-Friendly Design** — AILANG prioritizes deterministic, verifiable type checking
3. **Semantic Clarity** — Named recursion is more explicit than anonymous recursion

**Workaround**: Use named recursive functions:

```ailang
func factorial(n: int) -> int =
  if n <= 1 then 1 else n * factorial(n - 1)
```

---

### Polymorphic Arithmetic Operators in Lambdas

**Status**: Known bug — [Design Doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_7_0/m-poly-arithmetic-fix.md)
**Affects**: Arithmetic operators (`+`, `-`, `*`, `/`, `%`) inside polymorphic lambdas

**Problem**: Arithmetic operators in lambda bodies panic when called with floats:

```ailang
-- This panics at runtime:
let add = \x. \y. x + y in
add(3.14)(2.71)  -- panic: FloatValue, not IntValue
```

**Why**: Type inference defaults arithmetic operators to `int` (Num typeclass defaulting), but monomorphization runs after defaulting and sees the lambda already typed as `int -> int -> int`.

**What Works**:
```ailang
-- Comparison operators work:
let max = \x. \y. if x > y then x else y in
max(3.14)(2.71)  -- Returns: 3.14

-- Direct arithmetic (no lambdas) works:
let x = 3.14 + 2.71 in x * 2.0  -- Returns: 11.7

-- Named functions with explicit types work:
func addFloat(x: float, y: float) -> float = x + y
addFloat(3.14, 2.71)  -- Returns: 5.85
```

**Workaround**: Use named functions with concrete types, or use direct arithmetic without lambdas.

---

## Parser & Evaluation Limitations

### Pattern Guards (Parsed but Not Evaluated)

**Status**: Known limitation — [Design Doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_7_0/m-pattern-guards.md)

Pattern guards are syntactically supported but the guard conditions are not evaluated during matching:

```ailang
-- Guards are parsed but conditions are ignored:
match 5 {
  x if x > 100 => "over 100",  -- 5 > 100 is false, should skip
  x if x > 0 => "positive",    -- 5 > 0 is true, should match
  x => "other"
}
-- Expected: "positive"
-- Actual: "over 100" (first pattern matches, guard ignored)
```

**Workaround**: Use nested `if` expressions or separate match arms without guards.

---

## Language Feature Gaps

### String Interpolation

**Status**: Not implemented — planned for v0.13.0 as Phase 1 of [M-CONCAT-DISAMBIG](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_13_0/m-concat-disambiguation.md)

AILANG requires explicit concatenation:

```ailang
-- No string interpolation:
-- let msg = "Value: ${x}"  -- Not supported

-- Use concatenation:
let msg = "Value: " ++ show(x)
```

---

### Error Propagation Operator (`?`)

**Status**: Not implemented — [Design Doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_7_0/m-error-propagation.md)

The `?` operator for early return on errors is planned but not yet available.

---

### Typed Quasiquotes

**Status**: Planned — [Design Doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_7_0/m-quasi-typed-quasiquotes.md)

Typed quasiquotes for deterministic AST templates and secure string templating are designed but not yet implemented.

---

### CSP Concurrency

**Status**: Deferred

CSP-style concurrency with channels and session types is deferred. AILANG currently focuses on deterministic, single-threaded execution.

---

## Notes on Parser Error Messages

Parser errors now include:
- **Error codes** (e.g., `PAR_UNEXPECTED_TOKEN`)
- **Precise positions** (file, line, column)
- **Suggestions** for fixing the error

Example:
```
PAR_UNEXPECTED_TOKEN at file.ail:2:14: expected ), got INT
Suggestion: Add ')' to close grouped expression
```

---

## Reporting New Limitations

Found a limitation not listed here? Please file an issue at:
https://github.com/sunholo-data/ailang/issues

Include:
- AILANG version (`ailang --version`)
- Minimal reproduction code
- Expected vs actual behavior
- Whether it's a bug or design limitation
