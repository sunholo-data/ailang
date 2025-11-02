# AILANG Known Limitations

This document tracks known limitations, workarounds, and design constraints in AILANG.

## Recent Improvements

### Syntactic Sugar (v0.4.1 - NEW!)

**Status**: ✅ Implemented
**Since**: v0.4.1 (November 2, 2025)
**Impact**: Makes AILANG more ergonomic while preserving canonical forms

**What's Now Available**:

1. **Infix Cons Operator** (`x :: xs`):
   ```ailang
   -- ✅ Sugar syntax (v0.4.1+):
   let list = 1 :: 2 :: 3 :: []

   -- Desugars to canonical: ::(1, ::(2, ::(3, [])))

   -- Works in patterns too:
   match list {
     [] => 0,
     h :: t => h + sum(t)
   }
   ```

2. **Function Type Arrows** (`int -> bool`):
   ```ailang
   -- ✅ Sugar syntax (v0.4.1+):
   let f: int -> bool = \x. x > 0

   -- Desugars to canonical: funcType int bool

   -- Multi-argument types:
   let add: int -> int -> int = \x. \y. x + y
   ```

3. **Zero-Argument Calls** (`f()` in expressions):
   ```ailang
   -- ✅ Works in expression contexts:
   let result = if ready() then compute() else 0

   -- ⚠️ Top-level still requires space: main ()
   ```

**Disable Sugar**: Use `--strict-syntax` flag or `:strict` REPL command to reject all sugar and require canonical forms.

**See Also**: `prompts/v0.4.1.md` for complete syntax guide, `CHANGELOG.md` for technical details.

---

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

### Polymorphic Operators in Lambda Bodies

**Status**: Partially fixed in v0.4.0 (M-POLY-B Phase 1)
**Since**: v0.1.0 (fundamental to current compilation pipeline)
**Affects**: Operators with polymorphic types inside lambda expressions
**Last Updated**: v0.4.0

**What Works Now** (v0.4.0 - M-POLY-B Phase 1):
```ailang
-- ✅ Comparison operators work with polymorphic lambdas:
let max = \x. \y. if x > y then x else y in
max(3.14)(2.71)  -- Returns: 3.14 ✓

let equals = \x. \y. x == y in
equals(42.0)(42.0)  -- Returns: 42.0 ✓

-- ✅ All comparison operators work: >, <, >=, <=, ==, !=
-- ✅ Works with: floats, ints, strings (any Ord type)
-- ✅ Both inline and var-bound lambdas work
```

**What's Still Broken** (v0.4.0):
```ailang
-- ❌ Arithmetic operators panic with polymorphic lambdas:
let add = \x. \y. x + y in
add(3.14)(2.71)  -- panic: FloatValue, not IntValue

-- ❌ ALL arithmetic operators broken: +, -, *, /, %
-- ❌ Affects BOTH inline AND var-bound lambdas
-- ❌ Even this panics: (\x. \y. x + y)(3.14)(2.71)
```

**What Works Without Lambdas** (v0.3.18+):
```ailang
-- ✅ Direct arithmetic (no lambdas):
let x = 3.14 + 2.71 in  -- Correctly uses add_Float
x * 2.0                  -- Correctly uses mul_Float
```

**Root Cause**:
Monomorphization (v0.4.0) partially solves this, but type inference still defaults arithmetic operators to `int`:

1. **Type Inference**: Defaults arithmetic to `int` (Num typeclass defaulting)
2. **Monomorphization**: Runs AFTER defaulting, sees lambda already as `int -> int -> int`
3. **Runtime**: Receives Float values but code expects Int → panic

**Why Comparison Works But Arithmetic Doesn't**:
- **Comparison operators** (`>`, `<`, etc.): Type inference keeps them polymorphic (`Ord a => a -> a -> Bool`)
- **Arithmetic operators** (`+`, `-`, etc.): Type inference defaults them to `int` (Num typeclass defaulting)

This is a **type inference issue**, not a monomorphization issue. M-POLY-B Phase 1 fixed comparison operators by preserving their polymorphism. Phase 2 will fix arithmetic operators the same way.

**Workarounds for Comparison Operators** (No longer needed in v0.4.0!):
```ailang
-- ✅ Polymorphic comparison lambdas now work:
let max = \x. \y. if x > y then x else y in
max(3.14)(2.71)  -- Works! No workaround needed
```

**Workarounds for Arithmetic Operators** (Still needed in v0.4.0):
```ailang
-- ❌ Option 1: Type annotations (doesn't work - type inference ignores them)
-- let add: float -> float -> float = \x. \y. x + y in
-- add(3.14)(2.71)  -- Still panics!

-- ✅ Option 2: Named functions with concrete types
func addFloat(x: float, y: float) -> float = x + y
addFloat(3.14, 2.71)  -- Works!

func addInt(x: int, y: int) -> int = x + y
addInt(42, 17)  -- Works!

-- ✅ Option 3: Direct arithmetic (no lambdas)
let result = 3.14 + 2.71 in  -- Works!
result * 2.0
```

**Technical Details**:
- **M-POLY-B Phase 1** (v0.4.0): Fixed comparison operators by preserving polymorphism through monomorphization
- **Phase 2** (v0.4.2): Will fix arithmetic operators by changing type inference defaulting
- See: [M-POLY-B-PHASE1-COMPLETION-REPORT.md](../M-POLY-B-PHASE1-COMPLETION-REPORT.md)

**Related Issues**:
- Monomorphization implementation: `internal/pipeline/specialize.go`
- Phase 1 completion report: `M-POLY-B-PHASE1-COMPLETION-REPORT.md`
- Phase 2 tracking: Deferred to v0.4.2 (~4-8 hours estimated)

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
