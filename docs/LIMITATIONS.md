# AILANG Known Limitations

This document tracks known limitations, workarounds, and design constraints in AILANG.

## Type System Limitations

### Y-Combinator and Recursive Lambdas (By Design)

**Status**: Design constraint, not a bug
**Since**: v0.1.0 (Hindley-Milner type system)
**Affects**: Recursive anonymous functions, fixed-point combinators

**Problem**:
The Y-combinator and similar recursive lambda expressions fail with "occurs check" errors:

```ailang
-- ❌ This fails:
let Y = \f. (\x. f(x(x)))(\x. f(x(x))) in
-- Error: occurs check failed: type variable α occurs in (α → β)
```

**Root Cause**:
Hindley-Milner type inference prevents infinite types to ensure decidability. The Y-combinator requires a recursive type `α = α → β`, which would create an infinite type. This is an intentional limitation of the type system, not a bug.

**Why This Limitation Exists**:
1. **Type Inference Decidability**: Allowing infinite types would make type inference undecidable
2. **AI-Friendly Design**: AILANG prioritizes deterministic, verifiable type checking over expressive power
3. **Semantic Clarity**: Named recursion (`func factorial(n) = ...`) is more explicit than anonymous recursion

**Workaround**:
Use named recursive functions instead:

```ailang
-- ✅ Use named recursion:
func factorial(n: int) -> int =
  if n <= 1 then
    1
  else
    n * factorial(n - 1)
```

**Future**:
We may explore:
- Explicit type annotations for recursive lambdas (requires programmer-provided types)
- Iso-recursive types (requires manual fold/unfold, less ergonomic)
- Restricted fixed-point operators with bounded recursion

For now, use named functions for recursion. This aligns with AILANG's goal of semantic clarity for AI code synthesis.

---

### Float Comparisons in Lambda Bodies (Pre-existing Bug)

**Status**: Known bug (pre-dates M-DX3)
**Since**: Unknown (likely v0.1.0)
**Affects**: Comparison operators (`>`, `<`, `>=`, `<=`, `==`, `!=`) on Float values inside lambda expressions
**Fixed for Int in**: v0.3.17 (M-DX3)

**Problem**:
Float comparisons panic at runtime when used in lambda bodies:

```ailang
-- ❌ This panics:
let max = \x. \y. if x > y then x else y in
let f1 = 3.14 in
let f2 = 2.71 in
max(f1)(f2)  -- panic: interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue
```

**Root Cause**:
The operator lowering phase (`internal/pipeline/op_lowering.go`) correctly identifies comparison operators should use operand types (not result type Bool), but CoreTypeInfo doesn't have type information for Float variables. This causes the lowering to default to "Int" suffix, calling `gt_Int` instead of `gt_Float`.

**Why Int Works but Float Doesn't**:
M-DX3 (v0.3.17) fixed comparison operators for Int by using operand types instead of result type. However, the fix relies on CoreTypeInfo having the operand's type. For some reason, Float literal types aren't being stored in CoreTypeInfo, causing fallback to the default (Int).

**Workaround**:
None currently. Float comparisons don't work in lambda bodies. Use Float comparisons outside lambdas:

```ailang
-- ✅ This works:
let f1 = 3.14 in
let f2 = 2.71 in
if f1 > f2 then f1 else f2  -- Works outside lambda
```

**Status**:
This is out of scope for M-DX3, which focused on Int comparisons (the originally reported bug). Float comparisons need investigation into why CoreTypeInfo isn't populated for Float literals/variables.

**Related Issues**:
- See `design_docs/implemented/v0_3_17/m-dx3-lambda-dx-fixes.md` for Int comparison fix
- See `internal/pipeline/op_lowering_comparison_test.go` for Int comparison tests
- Float comparison tests currently don't exist (would fail)

---

## Parser Limitations

### Parse Error Messages

**Status**: Known limitation
**Since**: v0.1.0
**Affects**: Error messages for syntax errors

**Problem**:
Parse errors can be cryptic, especially for complex expressions:

```ailang
-- Example: Missing closing parenthesis
let x = (1 + 2
-- Error: "unexpected token: EOF"
```

**Workaround**:
- Use clear formatting and indentation
- Test small pieces in the REPL
- Check matching pairs: `()`, `{}`, `[]`

**Future**:
Better error recovery and suggestions planned for v0.4.0+

---

## Language Feature Gaps

### String Interpolation

**Status**: Not implemented
**Since**: v0.1.0
**Affects**: String construction

**Problem**:
AILANG requires explicit concatenation:

```ailang
-- ❌ No string interpolation:
-- let msg = "Value: ${x}"  -- Not supported

-- ✅ Use concatenation:
let msg = "Value: " ++ show(x)
```

**Workaround**:
Use `++` (string concatenation) and `show()` (value-to-string conversion).

**Future**:
String interpolation is planned for v0.4.0+

---

### Block Expressions

**Status**: Partially implemented
**Since**: v0.3.0
**Affects**: Sequencing multiple expressions

**Problem**:
Block expressions `{e1; e2; e3}` exist but have limitations around types and effects.

**Workaround**:
Use `let _ = expr in` chains for now:

```ailang
-- Current workaround:
let _ = print("step 1") in
let _ = print("step 2") in
print("step 3")
```

**Future**:
Block expressions will be refined in v0.4.0+

---

## Testing & Development

### REPL/File Parity

**Status**: Minor inconsistencies
**Since**: v0.1.0
**Affects**: Code that works in REPL but not in files (or vice versa)

**Problem**:
Some expressions work in the REPL but fail when in module files, usually due to:
- Module path validation
- Import/export requirements
- Effect capability checking

**Workaround**:
- Always test final code as a module file with `ailang run`
- Use `ailang repl` for quick experiments only

**Future**:
Improve parity and document differences clearly.

---

## Reporting New Limitations

Found a limitation not listed here? Please file an issue at:
https://github.com/sunholo-data/ailang/issues

Include:
- AILANG version (`ailang --version`)
- Minimal reproduction code
- Expected vs actual behavior
- Whether it's a bug or design limitation
