# Surface Sugar Pack (S-CALL0, S-CONS, S-ARROWTYPE)

**Status**: Planned
**Target**: v0.4.2
**Priority**: P1 (Medium)
**Estimated**: 1-2 days
**Dependencies**: None (builds on existing parser/Surface→Core pipeline)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Familiar syntax: `f()`, `x::xs`, `T -> U` reduces verbosity |
| Preserve Semantic Clarity | 0 | 0 | Bijective mapping preserves deterministic Core semantics |
| Increase Determinism | + | +1 | Parser-only; canonical Core form enforced by formatter |
| Lower Token Cost | + | +1 | Reduces LLM prior-mismatch, familiar syntax lowers token footprint |
| **Net Score** | | **+3** | **Decision: Move forward** ✅ |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Current State:**
- AILANG's current syntax deviates from ML/Haskell conventions in ways that cause **LLM prior-mismatch**
- Zero-arg calls require explicit unit: `print ()` instead of `print()`
- List cons requires prefix notation: `::(x, xs)` instead of `x :: xs`
- Function types use verbose constructors: `funcType T U` instead of `T -> U`
- These deviations increase token cost and cognitive load for AI models trained on conventional ML syntax

**Impact:**
- **AI code generation**: Models frequently generate incorrect syntax (e.g., `f()` when AILANG requires `f ()`)
- **Teaching prompt size**: Extra tokens needed to explain AILANG-specific syntax constraints
- **Developer experience**: Human developers also find current syntax unnecessarily verbose
- **Prior-mismatch cost**: Estimated ~10-15% increase in prompt tokens due to syntax corrections

**AILANG's Philosophy:**
"Deterministic core, permeable surface" — we can accept familiar surface syntax if it desugars bijectively to a canonical Core representation.

## Goals

**Primary Goal:** Reduce syntactic friction for AI and human developers while preserving AILANG's deterministic core semantics.

**Success Metrics:**
- All three sugars desugar correctly to existing Core AST nodes (no new semantics)
- Formatter canonicalizes all sugared forms to standard Core representation
- Teaching prompt updated to teach constraints while permitting familiar spellings
- Zero regressions in existing tests (type checking, elaboration, monomorphization)
- Feature-gated with `--strict-syntax` flag for teams wanting canonical-only code

## Solution Design

### Overview

Implement three **parser-only** syntactic sugars that desugar immediately to canonical Core forms:

1. **S-CALL0**: `f()` → `f ()`  (zero-arg call sugar)
2. **S-CONS**: `x :: xs` → `::(x, xs)`  (infix cons sugar)
3. **S-ARROWTYPE**: `T -> U` → `funcType T U`  (function type arrow sugar)

All sugars are:
- **Bijective**: Each sugared form maps to exactly one canonical Core form
- **Phase-bounded**: Desugaring happens in Surface → Core lowering (before type inference)
- **Opt-out**: Enabled by default; `--strict-syntax` flag disables for teaching/testing
- **Canonical output**: Formatter always prints Core form (not sugar)

### Architecture

**Components:**

1. **Parser Extensions** (`internal/parser/parser.go`)
   - Add grammar rules for `()`, `::`, and `->`
   - Desugar to existing AST nodes immediately
   - No new AST node types needed

2. **Lowering Pass** (existing Surface → Core elaboration)
   - Ensure desugaring occurs before type inference
   - Validate that Core AST is unchanged (sugar fully erased)

3. **Formatter/Pretty-Printer** (`internal/format/`)
   - Canonical output: always print Core forms
   - Never print sugared syntax (enforces "one true representation")

4. **Feature Flags & Diagnostics**
   - `--surface-sugar=S-CALL0,S-CONS,S-ARROWTYPE` (default: all enabled in v0.4.2+)
   - `--strict-syntax` disables all sugar (for teaching/testing)
   - Error messages suggest canonical equivalents when sugar is disabled

### Detailed Specifications

---

#### S-CALL0: Zero-Arg Call Sugar

**Surface:** `f()`
**Desugars to:** `f ()`  (application with unit value)

**Contract:**
- Pure sugar; no new semantics
- Always desugars to an application with the unit literal `()`
- Not a "nullary definition" — AILANG has no true nullary functions
- Pretty-printer never outputs `f()`; canonicalizes to `f ()`

**Grammar:**
```ebnf
PostfixCall0 := IDENT "(" ")"
```

**Precedence:** Same as function application (postfix)

**Desugaring:**
```
f()  →  App(f, Unit)
```

**Ambiguities / Edge Cases:**
- **Unit vs. no-arg**: AILANG has no true nullary functions; `()` is a value
- **Pipelines / partials**: `map(f(), xs)` becomes `map(f (), xs)` — remains a value, not a function
- If callee's type is not `Unit -> τ`, type checker emits existing error

**Diagnostics:**
```
Function application with () implies a Unit parameter; found <σ>.
Did you mean `f` (no call) or `f x` (with argument)?
```

**Type checking:** No changes needed — desugared form uses existing application type rules

**Examples:**
```ailang
-- Before (current AILANG)
let main: Unit -> Unit ! {IO} = \u.
  let _ = print () in
  ()

-- After (with S-CALL0)
let main: Unit -> Unit ! {IO} = \u.
  let _ = print() in
  ()

-- Both desugar to identical Core AST
```

---

#### S-CONS: Infix Cons Sugar

**Surface (patterns & expressions):** `x :: xs`
**Desugars to:** `::(x, xs)`  (prefix constructor application)

**Contract:**
- Works in both pattern and term positions
- Right-associative: `a :: b :: c` → `::(a, ::(b, c))`
- Higher precedence than `|` (record updates), lower than unary operators
- Pretty-printer canonicalizes to prefix: `::(x, xs)`

**Grammar:**
```ebnf
ConsExpr  := Term ("::" Term)*    -- right-associative
ConsPat   := Pat  ("::" Pat)*     -- right-associative
```

**Desugaring:**
```
a :: b :: c  →  ::(a, ::(b, c))
```

**Ambiguities / Edge Cases:**
- **Singleton lists**: `[x]` unchanged (not affected by cons sugar)
- **Mixed with tuples**: `x :: (y, z)` → `::(x, (y, z))` (tuple is a value)
- **Constructor overloading**: Only the list `::` constructor receives infix treatment (behind feature flag)

**Diagnostics:**
```
Right side of :: must be a list; got <τ>.
Did you mean ::(x, [y]) to create a list?
```

**Pattern matching:**
```ailang
-- Before (current AILANG)
match xs {
  | ::(x, rest) => x
  | [] => 0
}

-- After (with S-CONS)
match xs {
  | x :: rest => x
  | [] => 0
}

-- Both desugar to identical decision tree
```

**Expressions:**
```ailang
-- Before
let xs = ::(1, ::(2, ::(3, [])))

-- After
let xs = 1 :: 2 :: 3 :: []

-- Both produce same Core AST
```

---

#### S-ARROWTYPE: Function Type Arrow Sugar

**Surface (types only):** `T -> U`  (right-associative)
**Desugars to:** `funcType T U`  (or internal `TFun(T, U)`)

**Contract:**
- Parser-only; no change to value-level lambdas `\x. ...`
- Right-associative: `A -> B -> C` ≡ `A -> (B -> C)`
- Lower precedence than type application, higher than union types (if any)
- Pretty-printer canonicalizes to `funcType T U`

**Grammar:**
```ebnf
TypeArrow := TypeApp ("->" TypeArrow)?   -- right-associative
TypeApp   := TypeAtom (TypeAtom)*        -- existing
```

**Desugaring:**
```
A -> B -> C  →  TFun(A, TFun(B, C))
```

**Ambiguities / Edge Cases:**
- **Higher-order**: `(A -> B) -> C` parses correctly; outer parens optional when unambiguous
- **Quantification**: `[T] T -> U` keeps existing param syntax; no new `[A -> B]` forms
- **Effects**: Effects apply to the full function type to the right:
  ```
  T -> U ! {IO}    -- Effect on (T -> U)
  (T -> U) ! {IO}  -- Explicit parens (same meaning)
  ```

**Diagnostics:**
```
`func(T) -> U` is a type form. Value-level lambdas use `\x. ...`.
Did you mean `\x. ...` (value) or `T -> U` (type annotation)?
```

**Examples:**
```ailang
-- Before (current AILANG)
let id: [T] funcType T T = \x. x

-- After (with S-ARROWTYPE)
let id: [T] T -> T = \x. x

-- Both produce identical type AST
```

**Higher-order:**
```ailang
-- Before
let map: [A, B] funcType (funcType A B) (funcType (List A) (List B))

-- After
let map: [A, B] (A -> B) -> List A -> List B

-- Desugars to:
funcType (funcType A B) (funcType (List A) (List B))
```

---

### Implementation Plan

**Phase 1: Parser Extensions** (~4 hours)
- [ ] Add `()` postfix rule to parser; emit `App(f, Unit)`
- [ ] Add infix `::` with right-associativity; desugar to prefix constructor
- [ ] Add type arrow `->` with right-associativity; build `TFun` nodes
- [ ] Add unit tests for each sugar (parsing + desugaring)
- [ ] Test precedence interactions with existing operators

**Phase 2: Lowering Validation** (~2 hours)
- [ ] Ensure Surface → Core elaboration sees fully desugared AST
- [ ] Verify type inference, dictionary linking, monomorphization unchanged
- [ ] Add integration tests: sugared code → Core → type checking → evaluation
- [ ] Test that Core AST matches non-sugared equivalent (bijective property)

**Phase 3: Formatter & Canonicalization** (~3 hours)
- [ ] Update formatter to always output canonical Core forms
- [ ] Never print `f()`, `x :: xs`, or `T -> U` — use `f ()`, `::(x, xs)`, `funcType T U`
- [ ] Add tests: sugared input → formatter → canonical output
- [ ] Verify round-trip: parse → format → parse produces identical AST

**Phase 4: Feature Flags & Diagnostics** (~3 hours)
- [ ] Add `--surface-sugar=S-CALL0,S-CONS,S-ARROWTYPE` flag (default: all enabled)
- [ ] Add `--strict-syntax` flag to disable all sugar
- [ ] Update error messages to suggest canonical equivalents when sugar disabled
- [ ] Add REPL `:strict` command to toggle sugar mode
- [ ] Update teaching prompt with sugar/canonical side-by-side examples

**Phase 5: Testing & Documentation** (~4 hours)
- [ ] Golden tests: sugared vs. canonical produce identical Core/eval results
- [ ] Negative tests: type errors show helpful diagnostics
- [ ] Update [prompts/v0.4.2.md](../../prompts/v0.4.2.md) with sugar documentation
- [ ] Update [docs/LIMITATIONS.md](../../docs/LIMITATIONS.md) to remove current restrictions
- [ ] Add examples in [examples/](../../examples/) directory

### Files to Modify/Create

**Modified files:**
- `internal/parser/parser.go` - Add grammar rules for `()`, `::`, `->` (~150 LOC)
- `internal/lexer/token.go` - Add `COLONCOLON`, `ARROW` tokens if not already present (~10 LOC)
- `internal/format/formatter.go` - Canonical output for all sugared forms (~50 LOC)
- `cmd/ailang/main.go` - Add `--surface-sugar` and `--strict-syntax` flags (~20 LOC)
- `internal/repl/repl.go` - Add `:strict` toggle command (~15 LOC)
- `prompts/v0.4.2.md` - Update teaching prompt with sugar examples (~100 LOC)
- `docs/LIMITATIONS.md` - Remove "NO x::rest / NO (T -> U)" entries (~20 LOC deletions)

**New files:**
- `internal/parser/sugar_test.go` - Unit tests for each sugar (~200 LOC)
- `tests/golden/surface_sugar/` - Golden tests for integration (~10 test files)

**Total estimated LOC:** ~400 new, ~50 modified, ~20 deleted

## Examples

### Example 1: S-CALL0 (Zero-Arg Calls)

**Before:**
```ailang
let main: Unit -> Unit ! {IO} = \u.
  let _ = print () in  -- Space required!
  let _ = flush () in
  ()
```

**After:**
```ailang
let main: Unit -> Unit ! {IO} = \u.
  let _ = print() in   -- Familiar syntax
  let _ = flush() in
  ()
```

**Formatted output (canonical):**
```ailang
let main: Unit -> Unit ! {IO} = \u.
  let _ = print () in  -- Formatter always uses canonical form
  let _ = flush () in
  ()
```

---

### Example 2: S-CONS (Infix Cons)

**Before:**
```ailang
let rec sum: List int -> int = \xs.
  match xs {
    | ::(x, rest) => x + sum rest
    | [] => 0
  }

let numbers = ::(1, ::(2, ::(3, [])))
```

**After:**
```ailang
let rec sum: List int -> int = \xs.
  match xs {
    | x :: rest => x + sum rest  -- Familiar pattern matching
    | [] => 0
  }

let numbers = 1 :: 2 :: 3 :: []  -- Right-associative
```

**Formatted output (canonical):**
```ailang
let rec sum: List int -> int = \xs.
  match xs {
    | ::(x, rest) => x + sum rest  -- Canonical prefix form
    | [] => 0
  }

let numbers = ::(1, ::(2, ::(3, [])))
```

---

### Example 3: S-ARROWTYPE (Function Type Arrows)

**Before:**
```ailang
let map: [A, B] funcType (funcType A B) (funcType (List A) (List B)) =
  \f. \xs.
    match xs {
      | ::(x, rest) => ::(f x, map f rest)
      | [] => []
    }
```

**After:**
```ailang
let map: [A, B] (A -> B) -> List A -> List B =  -- Readable!
  \f. \xs.
    match xs {
      | x :: rest => f x :: map f rest
      | [] => []
    }
```

**Formatted output (canonical):**
```ailang
let map: [A, B] funcType (funcType A B) (funcType (List A) (List B)) =
  \f. \xs.
    match xs {
      | ::(x, rest) => ::(f x, map f rest)
      | [] => []
    }
```

---

### Example 4: All Sugars Combined

**Before:**
```ailang
let main: Unit -> Unit ! {IO} = \u.
  let numbers = ::(1, ::(2, ::(3, []))) in
  let printer: funcType int (Unit ! {IO}) = \n.
    print () in
  ()
```

**After:**
```ailang
let main: Unit -> Unit ! {IO} = \u.
  let numbers = 1 :: 2 :: 3 :: [] in
  let printer: int -> Unit ! {IO} = \n.
    print() in
  ()
```

**Both produce identical Core AST and evaluation results.**

---

## Success Criteria

- [ ] All three sugars parse correctly and desugar to canonical Core forms
- [ ] Formatter never outputs sugared syntax (always canonical)
- [ ] Type checking, elaboration, monomorphization unchanged (bijective mapping verified)
- [ ] `--strict-syntax` flag disables all sugar; error messages suggest canonical forms
- [ ] REPL `:strict` command toggles sugar mode
- [ ] Teaching prompt ([prompts/v0.4.2.md](../../prompts/v0.4.2.md)) updated with side-by-side examples
- [ ] [docs/LIMITATIONS.md](../../docs/LIMITATIONS.md) updated (remove "NO x::rest / NO (T -> U)")
- [ ] Golden tests confirm sugared and canonical code produce identical results
- [ ] All existing tests passing (zero regressions)
- [ ] Documentation updated (CHANGELOG.md, README.md)
- [ ] Examples added ([examples/surface_sugar.ail](../../examples/surface_sugar.ail))

## Testing Strategy

**Unit tests:**
- Parser tests for each sugar (correct AST generation)
- Precedence tests (interaction with existing operators)
- Associativity tests (right-assoc for `::` and `->`)
- Formatter tests (canonical output verified)

**Integration tests:**
- Golden tests: sugared code → parse → elaborate → type check → eval
- Equivalence tests: sugared vs. canonical produce identical Core AST
- Error tests: type mismatches show helpful diagnostics
- REPL tests: `:strict` mode toggles sugar on/off

**Manual testing:**
- REPL interaction with sugar enabled/disabled
- Verify formatter output on real-world code
- Test teaching prompt examples work correctly

## Non-Goals

**Not in this feature:**
- **User-defined infix operators** - Future work (requires operator precedence table)
- **Custom sugar syntax** - Only the three specified sugars for v0.4.2
- **Sugared output in formatter** - Formatter always prints canonical Core
- **Sugar in Core AST** - Sugar is fully erased before type inference
- **Backwards compatibility mode** - Sugar is always enabled by default (opt-out with `--strict-syntax`)

## Timeline

**Day 1** (8 hours):
- Phase 1: Parser extensions (4h)
- Phase 2: Lowering validation (2h)
- Phase 3: Formatter (2h)

**Day 2** (8 hours):
- Phase 4: Feature flags & diagnostics (3h)
- Phase 5: Testing & documentation (4h)
- Buffer for unexpected issues (1h)

**Total: ~16 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Precedence conflicts** with existing operators | Medium | Careful precedence table design; extensive parser tests |
| **Formatter breaks** on edge cases | Medium | Golden tests for all sugar forms; round-trip validation |
| **Type error messages** less clear with sugar | Low | Add suggestions in diagnostics to show canonical equivalents |
| **Teaching prompt confusion** (sugar vs. canonical) | Low | Side-by-side examples; clear "Formatted output" sections |
| **REPL vs. file mode** inconsistency | Low | Same parser used; feature flags apply to both |

## References

- [AILANG Vision: AI-First DX](../../benchmarks/VISION_BENCHMARKS.md) - Core philosophy
- [Example Parity & Vision Alignment](../v0_3_15/example-parity-vision-alignment.md) - Syntactic entropy reduction
- [Hindley-Milner Type System](../../docs/guides/type-system.md) - Type inference unchanged
- [Surface-to-Core Pipeline](../../internal/elaborate/README.md) - Desugaring phase
- [Parser Architecture](../../../CLAUDE.md#parser-development) - Parser conventions
- [Teaching Prompt v0.4.1](../../prompts/v0.4.1.md) - Current syntax documentation

## Future Work

**Post-v0.4.2 enhancements:**
- **User-defined infix operators** - Allow custom infix syntax with precedence declarations
- **Additional sugars** - Based on LLM prior-mismatch analysis (e.g., `do` notation for effects)
- **Lint rule**: `no-sugar-in-repo` for teams wanting canonical-only code checked in
- **IDE support**: Syntax highlighting for sugar (if IDE integration added later)
- **Sugar expansion in error messages** - Show both sugared and canonical forms in type errors

**Compatibility with future features:**
- **Reflection (v0.5.0+)**: Sugar fully erased before reflection; reflected AST is canonical Core
- **Quasiquotes (v0.5.0+)**: Sugar works in quoted code; quotes contain canonical Core
- **Polymorphic operators (v0.4.3)**: Arrow syntax compatible with operator overloading

---

**Document created**: 2025-11-01
**Last updated**: 2025-11-01
