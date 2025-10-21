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

### Polymorphic Operators in Lambda Bodies (Architectural Limitation)

**Status**: Architectural limitation - requires monomorphization (v0.4.0+)
**Since**: v0.1.0 (fundamental to current compilation pipeline)
**Affects**: ALL operators (`>`, `<`, `+`, `-`, etc.) with polymorphic types inside lambda expressions
**Partially Fixed in**: v0.3.18 (M-DX4) - simple cases now work

**Problem**:
Operators with polymorphic operands (lambda parameters) default to Int at compile time, causing runtime panics when called with other types:

```ailang
-- ❌ This panics with Float arguments:
let maxFloat = \x. \y. if x > y then x else y in
maxFloat(3.14)(2.71)  -- panic: interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue

-- ❌ This panics with String arguments:
let concat3 = \a. \b. \c. a ++ b ++ c in
concat3("hello")(" ")("world")  -- panic: expects String, got IntValue
```

**What Works Now** (v0.3.18+):
```ailang
-- ✅ Simple comparisons work:
let f1 = 3.14 in
let f2 = 2.71 in
if f1 > f2 then f1 else f2  -- Correctly uses gt_Float

-- ✅ Direct arithmetic works:
let x = 3.14 + 2.71 in  -- Correctly uses add_Float
x * 2.0                  -- Correctly uses mul_Float
```

**Root Cause**:
AILANG lacks **monomorphization** - the compiler pass that specializes polymorphic functions when called with concrete types. The current pipeline:

1. **Type Inference**: Lambda parameters get polymorphic types (`α`)
2. **Operator Lowering**: Happens on lambda BODY before knowing call-site argument types
3. **Defaulting**: Ambiguous types default to Int
4. **Runtime**: Receives Float values but code expects Int → panic

**Why This Happens**:
The lambda `\x. \y. x > y` correctly has polymorphic type `Ord a => a -> a -> Bool`. At compile time, `x` and `y` are type variables, not concrete types. Operator lowering must choose a specific builtin (`gt_Int`, `gt_Float`, etc.) but doesn't know which type `a` will be at runtime.

**Correct Behavior for AILANG's Design**:
This is NOT a bug in type inference or CoreTypeInfo population - it's a **missing compiler pass**. Two solutions exist:

1. **Monomorphization** (Rust, MLton): Clone function body for each concrete type it's called with
2. **Dictionary Passing** (Haskell): Pass type class dictionaries at runtime

AILANG currently does neither, so polymorphic operators default to Int.

**Workaround**:
Use top-level named functions or explicit type annotations:

```ailang
-- ✅ Option 1: Named functions (get specialized at call site)
func maxFloat(x: float, y: float) -> float =
  if x > y then x else y

maxFloat(3.14, 2.71)  -- Works!

-- ✅ Option 2: Avoid polymorphic operators in lambdas
let max = \x. \y. {
  let cmp = x > y in  -- Move comparison out of conditional
  if cmp then x else y
} in
-- Still fails - comparison still polymorphic!

-- ✅ Option 3: Use simple cases only
let result = 3.14 > 2.71 in  -- Works (M-DX4 fixed this)
if result then 3.14 else 2.71
```

**Technical Details**:
M-DX4 (v0.3.18) fixed CoreTypeInfo population:
- CoreTI now has concrete types after defaulting (was: type variables)
- Simple float comparisons work (3.14 > 2.71)
- Lambda parameters remain polymorphic (correct behavior!)
- Substitution chains fully resolved (α37 → α38 → Float)

See `M-DX4-IMPLEMENTATION-REPORT.md` for complete analysis.

**Future Plan**:
v0.4.0 will implement monomorphization:
- Detect polymorphic function calls with concrete arguments
- Clone function body for each concrete type
- Re-run operator lowering on specialized version
- Estimated effort: 2-3 weeks

**Related Issues**:
- See `M-DX4-IMPLEMENTATION-REPORT.md` for root cause analysis
- See `design_docs/planned/v0_3_18/m-dx4-coreti-population-gaps.md` for design
- See `M-DX4-AUDIT-FINDINGS.md` for investigation details

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
