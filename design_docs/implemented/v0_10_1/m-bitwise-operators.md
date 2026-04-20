# M-BITWISE-OPS: Bitwise Operators and Integer Builtins

**Status**: Planned
**Target**: v0.11.0
**Priority**: P1 — Blocks Track A of M-EVAL-XLANG (MiniHash requires FNV-1a with XOR)
**Estimated**: 1.5 days (~12 hours)
**Dependencies**: None (lexer/parser infrastructure already supports multi-char operators)
**Motivation**: [M-EVAL-XLANG](m-eval-cross-language-benchmark.md) — Cross-language benchmark integration

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Bitwise ops are pure, deterministic integer transforms |
| A2: Replayability | 0 | No impact on traces — same inputs always produce same outputs |
| A3: Effect Legibility | 0 | All new ops are pure (no effects) |
| A4: Explicit Authority | 0 | No capability changes — pure integer functions |
| A5: Bounded Verification | +1 | Closed over int type — type checker verifies statically |
| A6: Safe Concurrency | 0 | No concurrency changes — pure functions |
| A7: Machines First | +1 | Enables AILANG to express hash functions and binary protocols — critical for AI benchmark coverage |
| A8: Minimal Syntax | -1 | Adds 5 new operator tokens (`^`, `&`, `\|`, `<<`, `>>`) and 1 unary (`~`) |
| A9: Cost Visibility | 0 | O(1) operations, no hidden costs |
| A10: Composability | +1 | Composes with all existing int operations |
| A11: Structured Failure | 0 | No new error modes (shift amounts < 0 return error, same as div/0 pattern) |
| A12: System Boundary | 0 | No boundary crossings |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Pure integer arithmetic — fully deterministic
- [x] A3 (Effects): No side effects — all ops are `IsPure: true`
- [x] A4 (Authority): No ambient access — pure computation only
- [x] A7 (Machines First): Directly enables machine analysis of hash functions and binary protocols

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| **+3** | **✅ Proceed** |

## Problem Statement

AILANG lacks bitwise operators, preventing it from expressing hash functions, binary protocols, flag manipulation, and low-level algorithms. This is a critical gap for the M-EVAL-XLANG cross-language benchmark:

**Current State:**
- The `ai-coding-lang-bench` mini-git benchmark requires MiniHash — an FNV-1a variant using XOR and 64-bit unsigned multiply
- AILANG has no bitwise XOR (`^`), AND (`&`), OR (`|`), shift (`<<`, `>>`), or complement (`~`)
- The lexer treats `^` as `ILLEGAL`, single `&` as `ILLEGAL`, and single `|` as `PIPE` (pattern syntax only)
- Without these operators, AILANG cannot participate in Track A of the cross-language benchmark
- Every other language in the 15-language benchmark suite has bitwise operators

**Impact:**
- **Benchmark blockers**: Cannot implement FNV-1a hash (MiniHash), CRC checksums, or any hash-based data structure
- **Algorithm coverage gap**: LeetCode problems involving bit manipulation (~15% of algorithmic problems) are inexpressible
- **Credibility gap**: A language without bitwise operators appears incomplete to benchmark evaluators

## Goals

**Primary Goal:** Add bitwise operators as first-class infix operators with full pipeline support (lexer → parser → elaborator → evaluator), plus functional builtins in `std/math`.

**Success Metrics:**
- All 6 bitwise operators parse, type-check, and evaluate correctly on `int` operands
- FNV-1a hash function is expressible in AILANG (MiniHash reference implementation compiles and runs)
- `ailang doctor builtins` reports all new builtins as healthy
- Operator precedence matches C/Go convention (testable via precedence_test.go)
- `make test` and `make lint` clean

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Operator syntax (`^` `&` `\|` `<<` `>>` `~`) vs function-only (`xor(a,b)`) | Syntax affects every AILANG program using bits; function-only is safer but verbose | human | design | high |
| Precedence order (C-style vs simplified) | Affects expression parsing of ALL existing and future programs | human | design | high |
| Integer overflow behavior (wrap vs error) | Go wraps silently; explicit error aligns with A1 but breaks hash functions | human | design | high |
| Token reuse: `&` (currently ILLEGAL), `\|` (currently PIPE) | Changing `\|` semantics could break pattern matching syntax | compiler | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Operator vs function-only**: Use BOTH — infix operators for ergonomics + builtins for higher-order use
- [x] **Precedence**: C-standard dedicated bands — `~` (prefix) > `* /` > `+ -` > `<< >>` > `< >` > `== !=` > `&` > `^` > `&&` > `||`. Note: `&`/`^` bind LOOSER than `==` (C convention, requires parens for `(x & mask) == 0`).
- [x] **Overflow behavior**: Silent wrap (Go semantics) — required for hash functions, and AILANG's `int` is Go's `int` (64-bit signed). Hash algorithms may yield negative `int` values when printed because AILANG integers are signed 64-bit values. This does not affect deterministic bit-pattern correctness.
- [x] **Token disambiguation**: `&` becomes AMPERSAND when not followed by `&`. `|` stays as PIPE — infix bitwise OR is deferred; use `bitwiseOr(a, b)` function instead. This avoids context-sensitive lexing.

## Solution Design

### Overview

Add 6 bitwise operators to the full AILANG pipeline, following the exact same pattern as existing arithmetic operators (`+`, `-`, `*`, `/`, `%`). Each operator gets:
1. A lexer token
2. A parser precedence level and infix registration
3. A `core.IntrinsicOp` constant
4. An elaboration case mapping AST operator string to intrinsic
5. A runtime evaluation case in `evalBinOp`
6. A registered builtin in `std/math` for functional use

### Operators

| Operator | Token | Arity | Type | Description |
|----------|-------|-------|------|-------------|
| `&` | AMPERSAND | binary | `int -> int -> int` | Bitwise AND |
| `\|` | _(deferred)_ | — | — | Bitwise OR deferred; use `bitwiseOr(a, b)` function |
| `^` | CARET | binary | `int -> int -> int` | Bitwise XOR |
| `<<` | SHL | binary | `int -> int -> int` | Left shift (discards high bits, fills low with zero) |
| `>>` | SHR | binary | `int -> int -> int` | Arithmetic right shift (sign-extends) |
| `~` | TILDE | unary | `int -> int` | Bitwise complement (NOT) |

**Shift semantics**: `<<` discards high bits on overflow; low bits fill with zero. `>>` is arithmetic right shift — sign-extends (preserves sign of negative numbers). Negative shift amounts return a runtime error. All operations act on the 64-bit signed integer representation.

### Precedence (C-standard dedicated bands)

C-standard precedence with dedicated bands for each bitwise category. This means `&` and `^` bind **looser** than `==` — the same as C, requiring explicit parens for `(x & mask) == 0`.

**Rationale**: Matching C/mainstream precedence minimizes surprise for both humans porting code and AI models generating expressions. The well-known `&` vs `==` pitfall is accepted as a known trade-off; AILANG should not invent a novel precedence order.

```
LOWEST       = 0
LAMBDA       = 1   // \x.
LogicalOr    = 2   // ||
LogicalAnd   = 3   // &&
(BitOR)      = 4   // | (reserved — not an operator; use bitwiseOr())
BitwiseXor   = 5   // ^
BitwiseAnd   = 6   // &
EQUALS       = 7   // ==, !=
LESSGREATER  = 8   // <, >, <=, >=
SHIFT        = 9   // <<, >>
CONS         = 10  // ::
APPEND       = 11  // ++
SUM          = 12  // +, -
PRODUCT      = 13  // *, /, %
PREFIX       = 14  // -, not, ~
CALL         = 15  // f(x)
DotAccess    = 16  // .
```

**Key implications:**
- `8 & 3 == 0` parses as `8 & (3 == 0)` — use `(8 & 3) == 0`
- `1 << 2 + 1` parses as `1 << (2 + 1)` — use `(1 << 2) + 1`
- `a & b ^ c` parses as `(a & b) ^ c` — `&` binds tighter than `^`

**Note**: This renumbers all existing precedence levels. Relative ordering of non-bitwise operators is preserved.

### Token Disambiguation

**`&` (AMPERSAND vs AND):**
```go
case '&':
    if l.peekChar() == '&' {
        tok = NewToken(AND, "&&", ...)      // logical AND (unchanged)
    } else {
        tok = NewToken(AMPERSAND, "&", ...)  // bitwise AND (NEW — was ILLEGAL)
    }
```

**`|` (PIPE — unchanged):**

The `|` token remains as PIPE. Bitwise OR is **not** an infix operator in v0.11.0. Use `bitwiseOr(a, b)` from `std/math` instead. This avoids the parser complexity of context-sensitive `|` disambiguation across expression, pattern, ADT, and record syntax.

Infix `|` may be revisited in a future version once the pipe/pattern/operator story is more settled.

**`^` (CARET):**
```go
case '^':
    tok = NewToken(CARET, "^", ...)  // bitwise XOR (NEW — was ILLEGAL)
```

**`~` (TILDE):**
```go
case '~':
    tok = NewToken(TILDE, "~", ...)  // bitwise NOT (NEW — was ILLEGAL)
```

**`<<` and `>>`:**
```go
case '<':
    if l.peekChar() == '<' {
        tok = NewToken(SHL, "<<", ...)   // left shift (NEW)
    } else if l.peekChar() == '=' {
        tok = NewToken(LTE, "<=", ...)   // (unchanged)
    } else if l.peekChar() == '-' {
        tok = NewToken(LARROW, "<-", ...) // (unchanged)
    } else {
        tok = NewToken(LT, "<", ...)     // (unchanged)
    }

case '>':
    if l.peekChar() == '>' {
        tok = NewToken(SHR, ">>", ...)   // right shift (NEW)
    } else if l.peekChar() == '=' {
        tok = NewToken(GTE, ">=", ...)   // (unchanged)
    } else {
        tok = NewToken(GT, ">", ...)     // (unchanged)
    }
```

### Core IR Extensions

Add to `internal/core/core.go` IntrinsicOp enum:

```go
OpBitwiseAnd  // &
OpBitwiseOr   // |
OpBitwiseXor  // ^
OpShiftLeft   // <<
OpShiftRight  // >>
OpBitwiseNot  // ~ (unary)
```

### Builtin Registration

Register functional equivalents in `internal/builtins/math_bitwise.go`:

| Builtin Name | Module | Args | Pure | Description |
|-------------|--------|------|------|-------------|
| `_math_bitwiseAnd` | std/math | 2 | yes | `a & b` |
| `_math_bitwiseOr` | std/math | 2 | yes | `a \| b` |
| `_math_bitwiseXor` | std/math | 2 | yes | `a ^ b` |
| `_math_shiftLeft` | std/math | 2 | yes | `a << b` |
| `_math_shiftRight` | std/math | 2 | yes | `a >> b` |
| `_math_bitwiseNot` | std/math | 1 | yes | `~a` |

Stdlib wrappers in `stdlib/std/math.ail`:

```ailang
export func bitwiseAnd(a: int, b: int) -> int = _math_bitwiseAnd(a, b)
export func bitwiseOr(a: int, b: int) -> int = _math_bitwiseOr(a, b)
export func bitwiseXor(a: int, b: int) -> int = _math_bitwiseXor(a, b)
export func shiftLeft(a: int, n: int) -> int = _math_shiftLeft(a, n)
export func shiftRight(a: int, n: int) -> int = _math_shiftRight(a, n)
export func bitwiseNot(a: int) -> int = _math_bitwiseNot(a)
```

### Implementation Plan

**Phase 1: Lexer + Parser** (~3 hours)
- [ ] Add 6 token types to `internal/lexer/token.go`: AMPERSAND, BITOR, CARET, TILDE, SHL, SHR
- [ ] Update lexer `NextToken()` in `internal/lexer/lexer.go` for `&`, `|`, `^`, `~`, `<<`, `>>`
- [ ] Rename PIPE → BITOR where it represents the `|` character (update all parser references)
- [ ] Add 4 new precedence levels: BitwiseOr, BitwiseXor, BitwiseAnd, Shift
- [ ] Renumber existing precedence constants (preserve relative order)
- [ ] Register 5 new infix operators + 1 prefix operator in parser
- [ ] Add precedence tests in `internal/parser/precedence_test.go`
- [ ] Add lexer tests for all new tokens

**Phase 2: Core IR + Elaborator + Evaluator** (~4 hours)
- [ ] Add 6 IntrinsicOp constants to `internal/core/core.go`
- [ ] Update `String()` method's opStr map
- [ ] Add 6 cases to `normalizeBinaryOp()` in `internal/elaborate/expr_simple.go`
- [ ] Add unary `~` case to `normalizeUnaryOp()` (or equivalent)
- [ ] Add 6 cases to `evalBinOp()` in `internal/eval/eval_simple.go`
- [ ] Add unary `~` evaluation
- [ ] Shift validation: negative shift amount → runtime error (like div/0)
- [ ] Type checking: all ops restricted to `int` (reject float/string/bool)

**Phase 3: Builtins + Stdlib + Tests** (~3 hours)
- [ ] Create `internal/builtins/math_bitwise.go` with 6 builtin registrations
- [ ] Write hermetic tests in `internal/builtins/math_bitwise_test.go`
- [ ] Add stdlib wrappers in `stdlib/std/math.ail`
- [ ] Write AILANG-level tests (`examples/runnable/bitwise.ail`)
- [ ] Run `ailang doctor builtins` to validate
- [ ] Update `examples/manifest.json`

**Phase 4: Validation + MiniHash** (~2 hours)
- [ ] Write FNV-1a hash implementation in AILANG to prove operators work
- [ ] Run full `make test` and `make lint`
- [ ] Update `ailang prompt` output to include bitwise operators
- [ ] Update CHANGELOG.md

### Files to Modify/Create

**New files:**
- `internal/builtins/math_bitwise.go` — Bitwise builtin registrations (~120 LOC)
- `internal/builtins/math_bitwise_test.go` — Hermetic tests (~150 LOC)
- `examples/runnable/bitwise.ail` — Example AILANG program (~30 LOC)

**Modified files:**
- `internal/lexer/token.go` — Add 6 token types, update Precedence() (~30 LOC)
- `internal/lexer/lexer.go` — Update NextToken() for new character handling (~40 LOC)
- `internal/lexer/lexer_test.go` — Token recognition tests (~40 LOC)
- `internal/parser/parser.go` — Renumber precedence, register infix/prefix (~20 LOC)
- `internal/parser/precedence_test.go` — Precedence verification (~30 LOC)
- `internal/core/core.go` — Add IntrinsicOp constants (~15 LOC)
- `internal/elaborate/expr_simple.go` — Add normalization cases (~15 LOC)
- `internal/eval/eval_simple.go` — Add evaluation cases (~50 LOC)
- `stdlib/std/math.ail` — Add stdlib wrappers (~10 LOC)
- `prompts/ailang_prompt.md` — Document bitwise operators (~20 LOC)
- `examples/manifest.json` — Add bitwise example entry (~5 LOC)

**Estimated total: ~575 LOC** (implementation + tests)

## Examples

### Example 1: Basic Bitwise Operations

```ailang
module bitwise_demo
import std/math (bitwiseAnd, bitwiseOr, bitwiseXor, shiftLeft, shiftRight, bitwiseNot)

export func main() -> () ! {IO} {
  println("AND: " ++ show(12 & 10));        -- 8  (1100 & 1010 = 1000)
  println("OR:  " ++ show(12 | 10));        -- 14 (1100 | 1010 = 1110)
  println("XOR: " ++ show(12 ^ 10));        -- 6  (1100 ^ 1010 = 0110)
  println("SHL: " ++ show(1 << 4));          -- 16
  println("SHR: " ++ show(16 >> 2));         -- 4
  println("NOT: " ++ show(~0));              -- -1

  -- Functional style (for higher-order use)
  let ops = [bitwiseAnd, bitwiseOr, bitwiseXor];
  println("functional: " ++ show(map(\f. f(12, 10), ops)))
}
```

### Example 2: FNV-1a Hash (MiniHash — benchmark requirement)

```ailang
module minihash
import std/string (chars)

-- FNV-1a hash for strings (64-bit)
-- FNV offset basis: 14695981039346656037 (overflows to negative in signed 64-bit)
-- FNV prime: 1099511628211

func fnv1a(s: string) -> int =
  let basis = 14695981039346656037 in
  let prime = 1099511628211 in
  foldl(\hash c. (hash ^ charCode(c)) * prime, basis, chars(s))

export func main() -> () ! {IO} {
  println("hash of 'hello': " ++ show(fnv1a("hello")));
  println("hash of 'world': " ++ show(fnv1a("world")))
}
```

### Example 3: Bit Flags

```ailang
module flags

let READ    = 1 << 0   -- 1
let WRITE   = 1 << 1   -- 2
let EXECUTE = 1 << 2   -- 4

func hasFlag(flags: int, flag: int) -> bool = (flags & flag) != 0
func setFlag(flags: int, flag: int) -> int = flags | flag
func clearFlag(flags: int, flag: int) -> int = flags & ~flag

export func main() -> () ! {IO} {
  let perms = READ | WRITE;
  println("has READ: " ++ show(hasFlag(perms, READ)));      -- true
  println("has EXECUTE: " ++ show(hasFlag(perms, EXECUTE))); -- false
  let perms2 = setFlag(perms, EXECUTE);
  println("after setFlag: " ++ show(perms2));                 -- 7
  let perms3 = clearFlag(perms2, WRITE);
  println("after clearFlag: " ++ show(perms3))                -- 5
}
```

## Success Criteria

- [ ] All 6 operators tokenize correctly (lexer tests)
- [ ] Operator precedence matches C/Go convention (precedence tests)
- [ ] All 6 operators type-check as `int -> int -> int` (or `int -> int` for `~`)
- [ ] `12 & 10 == 8`, `12 | 10 == 14`, `12 ^ 10 == 6`, `1 << 4 == 16`, `16 >> 2 == 4`, `~0 == -1`
- [ ] Negative shift amount returns runtime error
- [ ] FNV-1a hash function compiles and produces correct output
- [ ] `ailang doctor builtins` reports all new builtins healthy
- [ ] Existing `|` in pattern matching and record syntax is unaffected
- [ ] `make test` and `make lint` clean
- [ ] Example file `examples/runnable/bitwise.ail` runs correctly
- [ ] `ailang prompt` output includes bitwise operators

## Testing Strategy

**Unit tests (Go):**
- Lexer: token recognition for all 6 operators, including edge cases (`&&` vs `&`, `||` vs `|`, `<<` vs `<`)
- Parser: precedence tests (`a & b | c` parses as `(a & b) | c`)
- Evaluator: correctness for each operator with positive, negative, and zero operands
- Builtins: hermetic tests via `testctx.NewMockEffContext()` with `-count=20`
- Shift edge cases: shift by 0, shift by 63, negative shift amount

**Integration tests (AILANG):**
- `examples/runnable/bitwise.ail` — exercises all operators with expected output
- FNV-1a hash produces consistent results across runs (determinism check)

**Manual testing:**
- REPL: `:type bitwiseXor` shows `int -> int -> int`
- REPL: `12 ^ 10` evaluates to `6`

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Exact error message for negative shift** — agent may choose wording (e.g., "negative shift amount: -3" or "shift count must be non-negative")
- **Builtin naming convention** — `bitwiseAnd` vs `bitAnd` vs `band` — agent may choose, consistency with existing `abs_Int` style preferred
- **PIPE token rename strategy** — whether to rename PIPE→BITOR globally or introduce BITOR as a new token alongside PIPE — agent may choose based on parser impact analysis

## Non-Goals

**Not attempted in this feature:**
- **Unsigned integer type** — AILANG's `int` is Go's signed `int64`. Unsigned arithmetic (needed for correct FNV-1a overflow) would require a new type. Signed overflow in Go wraps silently, which produces the same bit pattern — sufficient for hash functions.
- **Bitwise operators on floats** — No language supports this; integers only.
- **Operator overloading** — Bitwise ops work on `int` only, not user-defined types.
- **Bit manipulation stdlib** (popcount, leading zeros, trailing zeros, rotate) — Useful but out of scope. Can be added incrementally later.

## Timeline

**Day 1** (~8 hours):
- Phase 1: Lexer + Parser (tokens, precedence, tests)
- Phase 2: Core IR + Elaborator + Evaluator

**Day 2** (~4 hours):
- Phase 3: Builtins + Stdlib
- Phase 4: Validation + MiniHash example + CHANGELOG

**Total: ~12 hours across 1.5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `\|` token change breaks pattern matching | High | Careful parser context analysis; `\|` in match arms vs expressions already disambiguated by parser state |
| `<<`/`>>` conflict with type annotations or generics | Medium | AILANG has no generics syntax using `<>`; `<<` and `>>` are unambiguous in expression context |
| Precedence renumbering breaks existing tests | Low | All existing precedence tests use relative comparisons; renumbering preserves ordering |
| Signed overflow semantics differ from unsigned | Medium | Document that AILANG uses signed 64-bit; bit patterns match for XOR/AND/OR/shift; only multiplication overflow differs (same low bits) |

## Related Documents

- [M-EVAL-XLANG](m-eval-cross-language-benchmark.md) — Cross-language benchmark (primary motivation)
- [M-PIPE-OPERATOR](m-pipe-operator.md) — Pipe operator design (similar lexer/parser changes, coordinate `|>` token)
- [M-EVAL-XLANG Sprint Plan](m-eval-cross-language-benchmark-sprint-plan.md) — Sprint plan M1_BITWISE_FORK depends on this

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Go spec: Arithmetic operators](https://go.dev/ref/spec#Arithmetic_operators) — Go's bitwise operator semantics (our target)
- [FNV hash](https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function) — FNV-1a algorithm requiring XOR
- [ai-coding-lang-bench MiniHash](https://github.com/mame/ai-coding-lang-bench) — Benchmark requiring bitwise ops

## Future Work

- **Unsigned integer type** (`uint`) — Full unsigned arithmetic with explicit overflow semantics
- **Bit manipulation stdlib** — `popcount`, `leadingZeros`, `trailingZeros`, `rotateLeft`, `rotateRight`
- **Binary literal syntax** — `0b1010` for readability in bit manipulation code
- **Hex literal syntax** — `0xFF` (may already exist — verify)

---

**Document created**: 2026-03-30
**Last updated**: 2026-03-30
