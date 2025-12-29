# M-ERROR-PROP: Error Propagation Operator (`?`)

**Status**: Planned
**Target**: v0.7.0+
**Priority**: P2 (Medium) - Ergonomic error handling, advances Axiom A11
**Dependencies**: Result type (std/result.ail) - already implemented

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Desugaring is pure syntactic transform, deterministic |
| A2: Replayability | +1 | Traceable via elaborated Core form (spans preserved) |
| A3: Effect Legibility | +1 | Error propagation preserves effect signatures unchanged |
| A4: Explicit Authority | 0 | No authority changes (Result is pure data) |
| A5: Bounded Verification | +2 | Compile-time check that enclosing function returns Result |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | **Key benefit** - 70% token reduction in error handling code |
| A8: Minimal Syntax | +1 | Single postfix character, low syntactic overhead |
| A9: Cost Visibility | 0 | No cost implications |
| A10: Composability | +2 | `a()?.b()?.c()` chains cleanly, desugars predictably |
| A11: Structured Failure | +2 | **Primary goal** - makes Result patterns ergonomic |
| A12: System Boundary | +1 | Errors remain typed across module boundaries |

**Net Score: +13** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Pure syntactic desugaring, no runtime nondeterminism
- [x] A3 (Effects): Does not hide or introduce effects
- [x] A4 (Authority): No ambient authority (Result is pure)
- [x] A7 (Machines First): Improves machine readability (less nesting)
- [x] A11 (Structured Failure): Enhances, does not weaken structured errors

### A2 Note (Traceability)

The `?` operator is visible in traces via its elaborated Core form. Elaboration preserves source spans through `CoreNode.OrigSpan`, so traces can map synthetic match expressions back to the original `?` position. Users see the surface `?` in error messages, not the internal match structure.

---

## Problem Statement

Error handling with `Result` types requires verbose nested pattern matching:

```ailang
-- Current (verbose) - 8 lines, deeply nested:
func processFile(path: string) -> Result[string, Error] ! {FS} =
  match readFile(path) {
    Ok(content) => match parseContent(content) {
      Ok(data) => match transform(data) {
        Ok(result) => Ok(result),
        Err(e) => Err(e)
      },
      Err(e) => Err(e)
    },
    Err(e) => Err(e)
  }

-- Desired (concise) - 4 lines, flat:
func processFile(path: string) -> Result[string, Error] ! {FS} =
  let content = readFile(path)? in
  let data = parseContent(content)? in
  let result = transform(data)? in
  Ok(result)
```

**Impact:**
- 70% token reduction in typical error-handling code
- Reduced nesting depth (AI models handle flat code better)
- Matches patterns from Rust, Swift, making AILANG more accessible

---

## Goals

**Primary Goal:** Add `?` postfix operator for early-return on `Err`.

**Success Metrics:**
- `expr?` syntax parses and type-checks
- Early return works correctly at runtime
- Compile error for `?` in non-Result functions
- Chained `?` works: `a()?.b()?.c()`
- Axiom A11 gap closed (score 1→2)

---

## Solution Design

### Syntax

The `?` operator is postfix and works on `Result[T, E]` expressions:

```ailang
expr?
```

**Semantics:**
- If `expr` evaluates to `Ok(v)`, the `?` expression evaluates to `v`
- If `expr` evaluates to `Err(e)`, early-return `Err(e)` from enclosing function

### Enclosing Return Boundary

**Definition:** The `?` operator targets the **nearest surrounding func or lambda** that introduces a return type. Blocks, `if`, `match` arms, and `let` bodies do **not** introduce a return boundary.

**Allowed contexts:**
- Function bodies returning `Result[_, E]`
- Lambda bodies returning `Result[_, E]`
- Blocks, `if`, `match` arms, `let` bodies within such functions

**Disallowed contexts:**
- Functions not returning `Result` (compile error)
- Lambdas not returning `Result` (compile error - `?` does NOT escape lambda)
- Top-level REPL expressions (v0.7.0: disallowed with clear error)

```ailang
-- OK: ? targets outer func
func outer() -> Result[int, string] =
  let x = {
    let y = foo()? in  -- targets outer, OK
    y + 1
  } in Ok(x)

-- ERROR: ? inside non-Result lambda
func outer() -> Result[int, string] =
  let f = \(). foo()? in  -- ERROR: lambda returns unit, not Result
  f()

-- OK: ? inside Result-returning lambda
func outer() -> Result[int, string] =
  let f = \(). let x = foo()? in Ok(x) in  -- OK: lambda returns Result
  f()
```

### Type Rules

```
expr : Result[T, E]
enclosing return boundary returns Result[_, E]    -- exact match in v0.7.0
──────────────────────────────────────────────────
expr? : T
```

**v0.7.0 Constraint:** The operand error type `E` must **exactly match** the enclosing function's error type. No implicit conversion.

**Future (v0.8.0+):** Support error type conversion via `Into<Error>` trait or sum type construction.

### Evaluation Guarantees

**Single evaluation:** `expr` in `expr?` is evaluated exactly once. The desugaring binds the result to a temporary variable.

**Value preservation:** The `Err(e)` value is preserved without re-boxing. No new allocation occurs on the error path.

These invariants are important for:
- Side-effecting expressions (no duplicate FS/IO calls)
- Performance (no allocation overhead)
- Debugging (error values are identical, not copies)

### Desugaring

AILANG's Core IR is expression-only (A-Normal Form) with no explicit `Return` node. The `?` operator desugars purely to nested match expressions.

**General form:**

```ailang
-- Surface
expr?

-- Core (after elaboration)
match expr {
  Ok(__try_v) => __try_v,
  Err(__try_e) => Err(__try_e)
}
```

**In let context:**

```ailang
-- Surface
let x = foo()? in body

-- Core
match foo() {
  Ok(x) => body,
  Err(__try_e) => Err(__try_e)
}
```

**Key points:**
- No `Return` construct needed - match arms directly produce the function result
- Early return is structural: the `Err` arm produces `Err(e)` which becomes the function's return value
- Generated names use `__try_` prefix (reserved, not user-addressable)
- Source spans preserved: `CoreNode.OrigSpan` points to the `?` token

### Postfix Chain Grammar

The `?` operator participates in the **postfix chain** alongside calls and member access:

```
PostfixExpr := Primary { PostfixOp }*

PostfixOp :=
  | '(' Args ')'      -- function call
  | '.' IDENT         -- member access
  | '.' IDENT '(' Args ')'  -- method call
  | '?'               -- try operator
```

**Parse examples:**

| Expression | Parse tree |
|------------|------------|
| `foo()?` | Primary(`foo`) → Call(`()`) → Try(`?`) |
| `foo()?.bar()` | Primary(`foo`) → Call → Try → Member(`.bar`) → Call |
| `a()?.b()?.c()` | Primary(`a`) → Call → Try → Member → Call → Try → Member → Call |
| `foo()? + bar()?` | BinOp(`+`, TryExpr(`foo()`), TryExpr(`bar()`)) |

**Precedence:** `?` binds tighter than all infix operators (`+`, `|>`, etc.) but is part of the postfix chain with calls and member access.

**Note:** AILANG does not have optional chaining (`?.`). The sequence `?` followed by `.` is always Try followed by Member access, never a single token.

### Chained `?` Operators

```ailang
let x = foo()?.bar()? in ...
```

Desugars left-to-right (following postfix chain evaluation):

```ailang
match foo() {
  Ok(__try_1) => match __try_1.bar() {
    Ok(x) => ...,
    Err(__try_e) => Err(__try_e)
  },
  Err(__try_e) => Err(__try_e)
}
```

---

## Implementation Plan

### Phase 1: Lexer & Parser

**Lexer** (`internal/lexer/`):
- Add `QUESTION` token type for `?`
- Single-character token (no combined `?.` token)

**Parser** (`internal/parser/`):
- Extend postfix expression parsing to include `?`
- Create `TryExpr` AST node: `TryExpr{Operand: Expr, Pos: Position}`
- `?` is part of postfix chain (same loop as calls/member access)

### Phase 2: Type Checking

**Type Checker** (`internal/types/`):

Add to type checker context:
```go
type TCContext struct {
    // ... existing fields ...
    EnclosingResultErrorType *Type  // nil if not in Result-returning func/lambda
}
```

**Type checking `TryExpr`:**
1. Check operand is `Result[T, E]`
2. Check `ctx.EnclosingResultErrorType != nil` (error if nil)
3. Check `E == ctx.EnclosingResultErrorType` (exact match for v0.7.0)
4. Return type is `T`

**Context management:**
- Set `EnclosingResultErrorType` when entering func/lambda with Result return
- Clear when entering func/lambda with non-Result return
- Shadow correctly for nested lambdas

### Phase 3: Elaboration

**Elaboration** (`internal/elaborate/`):
- Desugar `TryExpr` to Core `Match` expression
- Generate fresh variables with `__try_` prefix (counter for uniqueness)
- Preserve source span: set `CoreNode.OrigSpan` to `TryExpr.Pos`

### Phase 4: Integration & Testing

- REPL: disallow `?` at top level with clear error
- Error message formatting (actionable messages)
- Documentation updates
- Example files

---

## Files to Modify/Create

**New files:**
- None (TryExpr added to existing ast.go)

**Modified files:**

| File | Changes | LOC |
|------|---------|-----|
| `internal/lexer/token.go` | Add QUESTION token | ~5 |
| `internal/lexer/lexer.go` | Recognize `?` | ~10 |
| `internal/parser/parser.go` | Parse `?` in postfix chain | ~50 |
| `internal/ast/ast.go` | TryExpr definition | ~20 |
| `internal/types/typecheck.go` | Type check TryExpr, add context field | ~80 |
| `internal/elaborate/elaborate.go` | Desugar TryExpr to Match | ~60 |
| `internal/types/traverse/` | Traverse TryExpr | ~10 |

**Estimated total:** ~235 LOC

---

## Edge Cases

### `?` in Non-Result Context

```ailang
func main() -> int =
  let x = foo()? in  -- Error: main returns int, not Result
  x
```

**Error message:**
```
TYPE003 at file.ail:2:12: cannot use ? operator here
  Expression type: Result[int, Error]
  Enclosing function returns: int
  The ? operator requires the enclosing function to return Result[_, E]
```

### `?` in Non-Result Lambda

```ailang
func outer() -> Result[int, string] =
  let f = \(). foo()? in  -- Error: lambda returns unit
  f()
```

**Error message:**
```
TYPE003 at file.ail:2:17: cannot use ? operator here
  Expression type: Result[int, string]
  Enclosing lambda returns: unit
  Hint: ? does not propagate out of lambdas. Make the lambda return Result.
```

### Multiple Error Types (v0.7.0: Error)

```ailang
func process() -> Result[int, Error] =
  let a = foo()? in   -- Returns Result[_, FooError]
  let b = bar()? in   -- Returns Result[_, BarError]
  Ok(a + b)
```

**Error message:**
```
TYPE004 at file.ail:2:12: error type mismatch in ? operator
  Expression error type: FooError
  Function error type: Error
  Hint: Use mapErr to convert error types, or use a common error type.
```

**Future (v0.8.0+):** Support error type unification via:
- `Into<Error>` trait for automatic conversion
- Sum type construction `FooError | BarError`

### Nested in Blocks

```ailang
let x = {
  let y = foo()? in  -- ? targets enclosing func, not block
  y + 1
} in x
```

Blocks don't introduce a return boundary. The `?` still targets the enclosing function.

---

## Testing Strategy

### Unit Tests

**Lexer tests** (`internal/lexer/lexer_test.go`):
- `?` tokenizes as QUESTION
- `foo()?.bar()` tokenizes as: IDENT LPAREN RPAREN QUESTION DOT IDENT LPAREN RPAREN (no combined token)

**Parser tests** (`internal/parser/parser_test.go`):
- `foo()?` parses as TryExpr wrapping CallExpr
- `foo()?.bar()` parses as CallExpr(MemberExpr(TryExpr(CallExpr)))
- `foo()? + bar()?` parses with correct precedence (both `?` bind before `+`)

**Type checker tests** (`internal/types/typecheck_test.go`):
- Ok case: Result function with `?` in body
- Error case: Non-Result function with `?`
- Error case: Non-Result lambda with `?`
- Ok case: Result-returning lambda with `?`
- Chained `?` type checking
- Error type mismatch detection

### Integration Tests

**New test file:** `tests/error_propagation_test.ail`

```ailang
-- Test: basic ? operator
func testBasic() -> Result[int, string] =
  let x = Ok(42)? in
  Ok(x + 1)
-- Expected: Ok(43)

-- Test: early return on Err
func testEarlyReturn() -> Result[int, string] =
  let x = Err("oops")? in
  Ok(x + 1)  -- Never reached
-- Expected: Err("oops")

-- Test: chained ?
func testChained() -> Result[int, string] =
  let a = Ok(1)? in
  let b = Ok(2)? in
  let c = Ok(3)? in
  Ok(a + b + c)
-- Expected: Ok(6)
```

### Critical "Gotcha" Tests

**1. No double evaluation:**
```ailang
-- Verify expr in expr? is evaluated exactly once
let counter = ref 0

func sideEffect() -> Result[int, string] =
  counter := !counter + 1;
  Ok(!counter)

func testSingleEval() -> Result[int, string] =
  let x = sideEffect()? in
  let y = sideEffect()? in
  Ok(x + y)
-- Expected: Ok(3) with counter == 2, not Ok(6) with counter == 4
```

**2. Precedence with infix operators:**
```ailang
func testPrecedence() -> Result[int, string] =
  let x = Ok(1)? + Ok(2)? in  -- Should parse as (Ok(1)?) + (Ok(2)?)
  Ok(x)
-- Expected: Ok(3)
```

**3. Lambda boundary (error case):**
```ailang
-- This should NOT compile
func testLambdaBoundary() -> Result[int, string] =
  let f = \(). Err("x")? in  -- ERROR: lambda doesn't return Result
  Ok(1)
-- Expected: TYPE003 error
```

**4. Chaining shape:**
```ailang
-- Verify foo()?.bar() is (foo()?).bar() not foo()?.(bar())
func testChainingShape() -> Result[int, string] =
  let r = Ok({ value: 42 }) in
  let x = r?.value in  -- Should access .value on unwrapped record
  Ok(x)
-- Expected: Ok(42)
```

### REPL Testing

**v0.7.0 rule:** `?` is disallowed at REPL top level.

```
ailang> Ok(42)?
Error: ? operator cannot be used at top level
  The ? operator requires an enclosing function that returns Result.
  Hint: Define a function that returns Result, then call it.
```

### Manual Testing

- Error messages are clear and actionable
- Source positions point to `?`, not synthetic match
- Performance: no measurable overhead vs. explicit match

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Precedence conflicts with future syntax | Medium | Low | Document postfix chain grammar explicitly |
| Lambda boundary confusion | Medium | Medium | Clear error message with hint |
| Desugaring hygiene (name collision) | Low | Low | `__try_` prefix reserved, counter for uniqueness |
| User confusion (Rust-like but not identical) | Low | Medium | Document differences clearly |
| Source span mapping in errors | Medium | Low | Preserve OrigSpan through elaboration |

---

## Workaround (Current)

Until `?` is implemented, use explicit match or `flatMap`:

```ailang
-- Option 1: Explicit match
match readFile(path) {
  Ok(content) => process(content),
  Err(e) => Err(e)
}

-- Option 2: flatMap from std/result
import std/result (flatMap)

readFile(path) |> flatMap(\c. parseContent(c)) |> flatMap(\d. transform(d))
```

---

## Success Criteria

- [ ] `?` lexes as QUESTION token (no combined `?.`)
- [ ] `foo()?` parses as TryExpr in postfix chain
- [ ] Type checker enforces Result operand
- [ ] Type checker enforces exact error type match (v0.7.0)
- [ ] Type checker tracks enclosing return boundary through lambdas
- [ ] Elaboration produces correct match desugaring
- [ ] Source spans preserved (errors point to `?`)
- [ ] Chained `?` works: `a()?.b()?.c()`
- [ ] Lambda boundary enforced (error if lambda doesn't return Result)
- [ ] REPL disallows top-level `?`
- [ ] Single evaluation guaranteed
- [ ] All existing tests pass
- [ ] New test file with gotcha tests
- [ ] Documentation updated
- [ ] Limitations page updated (remove this limitation)
- [ ] Axiom scorecard A11 updated to 2/2

---

## Open Questions

1. **Do-block integration:** Should `?` work in future `do`-block-like constructs? (Defer to v0.8.0+)

2. **Error conversion:** Prefer `Into<Error>` trait or explicit `mapErr`? (v0.8.0+ decision)

3. **Trace visibility:** Current plan preserves surface spans. Should traces also show the desugared match for debugging? (Implementation detail)

---

## Related Documents

**Axiom References:**
- [Design Axioms](/docs/references/axioms) - A11: Failure Must Be Representable
- [Axiom Scorecard](docs/static/benchmarks/axiom_scorecard.json) - Current A11 gap

**Implementation References:**
- [std/result.ail](std/result.ail) - Result type definition
- [Limitations](/docs/reference/limitations#error-propagation-operator-) - Current limitation
- [internal/core/core.go](internal/core/core.go) - Core IR (expression-only, no Return)

**Language Inspirations:**
- [Rust ? operator](https://doc.rust-lang.org/book/ch09-02-recoverable-errors-with-result.html)
- [Swift try? operator](https://docs.swift.org/swift-book/LanguageGuide/ErrorHandling.html)

**Future Work:**
- [m-contracts-assert.md](../v0_8_0/m-contracts-assert.md) - Precondition assertions
- Error type conversion (`Into<Error>` or sum types)

---

## Website Links

**Update these when this feature is implemented:**
- [Limitations page](/docs/reference/limitations) — Remove from limitations list
- [Implementation Status](/docs/reference/implementation-status) — Update status
- Move this doc from `planned/` to `implemented/`
- Update axiom scorecard A11 to 2/2

---

**Document created**: 2024 (original)
**Last updated**: 2025-12-29 (axiom compliance, semantic precision, feedback integration)
