# Numeric Coercion for Mixed-Type Arithmetic

**Status**: 📋 PLANNED (Design Phase)
**Priority**: P3 (Medium - Convenience feature)
**Estimated**: ~500-800 LOC
**Duration**: 2-3 days
**Target**: v0.4.0+

## Problem Statement

**User Expectation**: `1 + 2.5` should "just work" by coercing `1` to `1.0`.

```ailang
-- ❌ CURRENT: Runtime error
λ> 1 + 2.5
Runtime error: builtin add_Int expects Int arguments

-- ✅ WORKAROUND: Manual conversion
λ> 1.0 + 2.5
3.5 :: Float
```

**Root Cause**:
- Type classes (`Num[Int]`, `Num[Float]`) allow `+` to be polymorphic
- BUT they don't provide automatic coercion between types
- Type inference resolves `+` to EITHER `add_Int` OR `add_Float`, not both
- Mixed literals like `1 + 2.5` cause type unification failure

**Current Behavior**:
- `1 + 2` → Works (both `Int`, uses `add_Int`)
- `1.0 + 2.0` → Works (both `Float`, uses `add_Float`)
- `1 + 2.5` → **FAILS** (can't unify `Int` with `Float`)

**What Users Expect** (from Python, JavaScript, etc.):
- Automatic "widening" conversion: `Int → Float`
- `1 + 2.5` → `1.0 + 2.5` → `3.5`

## Design Challenges

### Challenge 1: Purity vs Convenience

**Haskell-style (current AILANG):**
- No implicit coercion - types must match exactly
- Explicit conversion: `fromIntegral(1) + 2.5` or `intToFloat(1) + 2.5`
- PRO: Type safety, predictability, no surprises
- CON: Verbose for numeric code

**Python/JavaScript-style:**
- Automatic coercion follows "widening" rules
- PRO: Convenient, matches mathematical intuition
- CON: Can hide bugs, performance surprises

**Middle Ground Options:**
1. **Type-directed coercion** - Insert coercions during type checking
2. **Explicit conversion operators** - Add `1 as Float + 2.5` syntax
3. **Coercion type class** - `Coercible Int Float` with automatic insertion

### Challenge 2: When to Coerce?

**Coercion Rules Need to Be Defined:**
- Int → Float (safe widening)
- Float → Int? (lossy - should this be automatic?)
- String → Int? (parsing - definitely not automatic)

**Haskell approach:** No automatic coercion, use explicit functions
**Rust approach:** No implicit coercion, use `as` keyword
**Python approach:** Automatic widening for numeric operations

### Challenge 3: Type Inference Complexity

**Current type inference:**
```
1 + 2.5
  ↓ Infer
1 :: α, 2.5 :: β
  ↓ Constraint
Num[α], Num[β], α ~ β
  ↓ Defaulting
α := Int, β := Float
  ↓ Unification
Int ~ Float → ERROR!
```

**With coercion:**
```
1 + 2.5
  ↓ Infer
1 :: α, 2.5 :: β
  ↓ Constraint
Num[α], Fractional[β], α ~ β OR Coercible[α, β]
  ↓ Resolve
Coercible[Int, Float] ✓
  ↓ Insert coercion
intToFloat(1) + 2.5 :: Float
```

This requires:
1. New constraint type: `Coercible[From, To]`
2. Constraint solver extension
3. Core AST coercion nodes: `core.Coerce{From, To, Expr}`
4. Runtime coercion functions

## Proposed Approaches

### Option 1: Explicit Conversion Functions (Easy - RECOMMENDED)

**Add builtin conversion functions, NO implicit coercion:**

```ailang
-- Add to stdlib/std/prelude
export func intToFloat(x: int) -> float { ... }
export func floatToInt(x: float) -> int { ... }  -- Truncates

-- Users write:
λ> intToFloat(1) + 2.5
3.5 :: Float
```

**Pros:**
- Simple to implement (~100 LOC)
- No type system changes needed
- Explicit, predictable, no surprises
- Matches Haskell philosophy

**Cons:**
- Verbose for numeric-heavy code
- Doesn't match user expectations from Python/JS

**Estimated:** ~100 LOC, 2 hours

### Option 2: Type-Directed Implicit Coercion (Medium)

**Insert coercions automatically during type checking:**

1. Add `Coercible[From, To]` constraint type
2. Define safe coercions: `Coercible[Int, Float]`
3. During type checking, if `α ~ β` fails, try `Coercible[α, β]`
4. Insert `core.Coerce` nodes in elaboration
5. Runtime evaluates coercions

**Pros:**
- Convenient: `1 + 2.5` just works
- Still type-safe: only pre-defined coercions allowed
- Matches mathematical intuition

**Cons:**
- Complex type system changes (~500 LOC)
- Potential for confusion: "Why did this convert?"
- Performance implications (extra runtime checks)

**Estimated:** ~500 LOC, 2-3 days

### Option 3: Explicit `as` Keyword (Medium-Hard)

**Add `as` syntax for explicit coercion:**

```ailang
λ> 1 as Float + 2.5
3.5 :: Float

λ> (1 + 2) as Float / 3.0
1.0 :: Float
```

**Pros:**
- Explicit but less verbose than function calls
- Clear where coercion happens
- Matches Rust, TypeScript conventions

**Cons:**
- New syntax to learn
- Parser changes needed
- Still verbose compared to implicit coercion

**Estimated:** ~300 LOC, 1-2 days

## Recommendation

**For v0.4.0+: Start with Option 1 (Explicit Functions)**

**Reasoning:**
1. **Quick win** - Can ship in hours, not days
2. **Low risk** - No type system changes, no new syntax
3. **Aligns with FP philosophy** - Explicit over implicit
4. **Unblocks users** - Provides immediate solution
5. **Reversible** - Can add implicit coercion later if needed

**Implementation Plan (Option 1):**

```ailang
-- stdlib/std/prelude.ail additions

-- Safe widening conversion (always succeeds)
export func intToFloat(x: int) -> float {
  $builtin.intToFloat(x)
}

-- Lossy conversion (truncates)
export func floatToInt(x: float) -> int {
  $builtin.floatToInt(x)
}

-- Rounding variants (future)
export func round(x: float) -> int { ... }
export func floor(x: float) -> int { ... }
export func ceiling(x: float) -> int { ... }
```

**Builtin implementations** (`internal/builtins/`):
- Add `intToFloat` and `floatToInt` to builtin registry
- Runtime conversion in evaluator

**Documentation updates:**
- Teach prompt: "For mixed numeric types, use intToFloat()"
- Playground: Show conversion example
- Examples: Add `examples/numeric_conversion.ail`

## Future Considerations

**If we add implicit coercion later (v0.5.0+):**
- Implement Option 2 (Type-Directed Coercion)
- Keep explicit functions for performance-critical code
- Add flag: `AILANG_IMPLICIT_COERCION=1` for opt-in behavior
- Study Haskell's approach (no coercion) vs Python's (automatic)

## Non-Goals (Out of Scope)

- ❌ String to number conversion (use `parseInt`, `parseFloat`)
- ❌ Bool to int conversion (use pattern matching)
- ❌ Array/List coercions
- ❌ Custom user-defined coercions

## Related Design Docs

- Type class system (implemented in v0.3.0)
- Numeric defaulting (implemented in v0.3.0)
- Future: Generic numeric tower (v0.5.0+)

## References

**Languages with NO implicit coercion:**
- Haskell - Use `fromIntegral`, `realToFrac`
- Rust - Use `as` keyword
- OCaml - Use explicit conversion functions

**Languages WITH implicit coercion:**
- Python - Int → Float automatic
- JavaScript - Aggressive coercion (often problematic)
- C - Widening conversions automatic

**AILANG Philosophy:** Follow Haskell/OCaml (explicit), not Python/JS (implicit)

---

**Status:** Design phase - need user feedback on approach
**Decision needed:** Explicit functions only, or invest in type-directed coercion?
