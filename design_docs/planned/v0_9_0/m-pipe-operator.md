# M-PIPE-OPERATOR: Pipe operator and Option chaining

**Status:** Planned
**Version:** v0.7.3
**Priority:** P2 — reduces nested match boilerplate (6 instances of 3-4 levels deep in DocParse)
**Estimated effort:** 6-8 hours
**Origin:** DocParse DX Feedback (Feb 2026)
**Dependencies:** None (lexer/parser infrastructure already supports multi-char operators)

## Problem

Chaining Option operations creates deeply nested code. DocParse reported 6 instances of 3-4 level nesting for simple XML attribute extraction:

```ailang
match findFirst(p, "w:pPr") {
  Some(ppr) =>
    match findFirst(ppr, "w:numPr") {
      Some(numPr) =>
        match findFirst(numPr, "w:numId") {
          Some(numId) =>
            match getAttr(numId, "w:val") {
              Some(val) => val,
              None => ""
            },
          None => ""
        },
      None => ""
    },
  None => ""
}
```

`std/option` already has `flatMap` and `map`, but chaining them is inside-out:

```ailang
getOrElse(
  flatMap(\numId. getAttr(numId, "w:val"),
    flatMap(\numPr. findFirst(numPr, "w:numId"),
      flatMap(\ppr. findFirst(ppr, "w:numPr"),
        findFirst(p, "w:pPr")))),
  "")
```

This reads backwards. Users want left-to-right data flow.

## Axiom Compliance

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A1: Determinism | 0 | Pure desugaring, no semantic change |
| A2: Replayability | 0 | Traces still show the underlying application |
| A3: Effect Legibility | 0 | Effects propagate transparently through pipe |
| A7: Machines First | +1 | Reduces nesting depth, easier for AI to parse/generate |
| A8: Minimal Syntax | -1 | Adds one new token/operator |
| A9: Cost Visibility | 0 | No hidden costs, just reordered application |
| A10: Composability | +1 | Enables left-to-right composition of any functions |

**Net score: +1.** Justification for A8 (-1): The syntax addition is warranted because it eliminates a class of deeply-nested code that is hard for both humans and AIs to read/generate. The pipe is a pure syntactic transform (desugars to function application), adds no semantic complexity, and is the most common missing operator reported by users.

## Proposed Solution: Pipe Operator `|>`

### v0.7.3 Scope: `x |> f` only (simple pipe)

**`x |> f` desugars to `f(x)`.** That's it. No multi-arg form.

```ailang
findFirst(p, "w:pPr")
  |> bind(\ppr. findFirst(ppr, "w:numPr"))
  |> bind(\numPr. findFirst(numPr, "w:numId"))
  |> bind(\numId. getAttr(numId, "w:val"))
  |> getOrDefault("")
```

This reads top-to-bottom, left-to-right — the natural data flow.

### Why NOT `x |> f(a)` in v0.7.3

The previous draft proposed `x |> f(a)` desugaring to `f(a, x)` (insert as last argument). This is **incompatible with AILANG's runtime semantics**.

**AILANG has strict arity matching and NO partial application:**

```go
// internal/eval/eval_operations.go:46-47
if len(args) != len(fn.Params) {
    return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
}
```

- `\x y. x + y` creates a `FunctionValue{Params: ["x", "y"]}` — arity 2, strict.
- `f(a)` where `f` expects 2 args → **runtime error**, not partial application.
- Only `\x. \y. x + y` (explicit curried syntax) supports `f(a)` returning a closure.

**Core AST uses N-ary `App`** (`internal/core/core.go:140-150`):
```go
type App struct {
    CoreNode
    Func CoreExpr
    Args []CoreExpr  // Multiple arguments in single node
}
```

To support `x |> f(a)` → `f(a, x)`, the elaborator would need to detect a `Call` on the RHS and splice `x` into its argument list. This is doable (N-ary App supports it), but it creates a novel desugaring rule that doesn't exist anywhere else in the language and whose interaction with type inference is untested.

**Decision:** Ship the simple form `x |> f` only. This is safe, well-understood, and sufficient when combined with pipe-friendly stdlib functions. Defer `x |> f(a)` to v0.8.0+ when/if we decide on currying semantics.

## Design Decisions

### Semantics: `x |> f` desugars to `f(x)`

Simple, unambiguous, no edge cases:

```
x |> f           → App(Var(f), [x])
x |> (\y. ...)   → App(Lambda(...), [x])
```

The RHS of `|>` must be a **single-argument callable**: either a bare function name or a lambda. If the RHS is a literal, record, list, or other non-callable expression, it's a type error (detected by the type checker after desugaring, not the parser).

### Pipe-friendly stdlib convention: data-last, arity-1 wrappers

For `|>` to be useful with multi-arg stdlib functions, we provide arity-1 wrappers that close over the configuration and accept only the data argument:

```ailang
-- std/option already has flatMap(f, opt), but we can pipe:
someOption |> bind(\x. transform(x))

-- Because bind takes 1 arg (a lambda that returns Option):
-- bind(\x. transform(x)) returns a function Option[a] -> Option[b]
-- Wait — that requires currying! Which we don't have.
```

**Correction:** Since AILANG doesn't curry, `bind(f)` where bind expects 2 args would fail. So the approach is: **pipe-friendly functions must accept exactly 1 argument** (the data being piped). Configuration is captured in the lambda:

```ailang
-- This works with x |> f because \opt. flatMap(\x. transform(x), opt) is arity-1
someOption |> \opt. flatMap(\x. transform(x), opt)
```

That's verbose. So we add **arity-1 convenience wrappers** to `std/option`:

```ailang
-- bind: takes a function, returns a function that accepts an Option
-- bind(f) returns (\opt. flatMap(f, opt)) — but we can't do this without currying!
```

**The real solution:** since currying doesn't work, the pipe-friendly functions must themselves be arity-1, taking the data as their single argument. This means the lambda IS the pipe-friendly form:

```ailang
findFirst(p, "w:pPr")
  |> \opt. flatMap(\ppr. findFirst(ppr, "w:numPr"), opt)
  |> \opt. flatMap(\numPr. findFirst(numPr, "w:numId"), opt)
  |> \opt. flatMap(\numId. getAttr(numId, "w:val"), opt)
  |> \opt. getOrElse(opt, "")
```

This is still left-to-right but verbose. **Better approach:** add pipe-friendly arity-1 functions to `std/option` that take only the Option:

```ailang
-- These are arity-1 functions: each takes exactly one Option argument.
-- The "f" parameter is captured in the returned closure.

-- But wait — returning closures IS currying...
-- Actually no: we can define them as stdlib functions that happen to take 1 arg.
```

**Final resolution:** The key insight is that functions like `bind(\x. ...)` aren't partially applied stdlib functions — they're **fresh arity-1 lambdas** written inline. The pipe operator works perfectly with them:

```ailang
findFirst(p, "w:pPr")
  |> (\opt. flatMap(\ppr. findFirst(ppr, "w:numPr"), opt))
  |> (\opt. flatMap(\numPr. findFirst(numPr, "w:numId"), opt))
  |> (\opt. flatMap(\numId. getAttr(numId, "w:val"), opt))
  |> (\opt. getOrElse(opt, ""))
```

Each `(\opt. flatMap(f, opt))` is an arity-1 lambda — pipe applies the left side as the single argument. This is correct and works today.

**But it's ugly.** The DX win requires stdlib helpers. Since we can't curry, define them as **arity-2 AILANG functions** that the user wraps in a lambda only at the pipe boundary:

Actually — let's step back. The cleanest API that works without currying:

```ailang
-- std/option: pipe-ready functions (arity-1, take only the Option)
-- NOT possible as standalone functions without closures over f

-- What DOES work: named helpers that compose naturally
```

**The pragmatic answer:** Given no currying, `|>` composes best with arity-1 functions. For Option chaining, the natural arity-1 operation is a lambda that captures the transform:

```ailang
findFirst(p, "w:pPr")
  |> (\x. flatMap(\ppr. findFirst(ppr, "w:numPr"), x))
  |> (\x. flatMap(\numPr. findFirst(numPr, "w:numId"), x))
  |> (\x. flatMap(\numId. getAttr(numId, "w:val"), x))
  |> (\x. getOrElse(x, ""))
```

This IS left-to-right and avoids the nesting problem. It's more verbose than Elixir-style pipe but it's honest about AILANG's semantics. The lambda wrapper is the "partial application" — just explicit.

### Why this is still a DX win

Compare the three forms:

**Before (nested match — 12 lines, 4 levels deep):**
```ailang
match findFirst(p, "w:pPr") {
  Some(ppr) => match findFirst(ppr, "w:numPr") {
    Some(numPr) => match findFirst(numPr, "w:numId") {
      Some(numId) => match getAttr(numId, "w:val") {
        Some(val) => val, None => ""
      }, None => ""
    }, None => ""
  }, None => ""
}
```

**Before (inside-out flatMap — 5 lines, reads backwards):**
```ailang
getOrElse(flatMap(\numId. getAttr(numId, "w:val"),
  flatMap(\numPr. findFirst(numPr, "w:numId"),
    flatMap(\ppr. findFirst(ppr, "w:numPr"),
      findFirst(p, "w:pPr")))), "")
```

**After (pipe — 5 lines, reads forwards):**
```ailang
findFirst(p, "w:pPr")
  |> (\x. flatMap(\ppr. findFirst(ppr, "w:numPr"), x))
  |> (\x. flatMap(\numPr. findFirst(numPr, "w:numId"), x))
  |> (\x. flatMap(\numId. getAttr(numId, "w:val"), x))
  |> (\x. getOrElse(x, ""))
```

The pipe version reads top-to-bottom and each line does one thing. The nesting is gone. The lambda wrapper `(\x. ...)` is the cost of no currying — explicit but predictable.

### Parsing implementation (grounded in codebase)

**Current lexer state** (`internal/lexer/lexer.go:162-169`):
```go
case '|':
    if l.peekChar() == '|' {
        ch := l.ch
        l.readChar()
        tok = NewToken(OR, string(ch)+string(l.ch), line, column, l.file)
    } else {
        tok = NewToken(PIPE, string(l.ch), line, column, l.file)
    }
```

**Required change:** Add `|>` check before the `PIPE` fallback:
```go
case '|':
    if l.peekChar() == '|' {
        // existing OR handling
    } else if l.peekChar() == '>' {
        ch := l.ch
        l.readChar()
        tok = NewToken(PIPE_ARROW, string(ch)+string(l.ch), line, column, l.file)
    } else {
        tok = NewToken(PIPE, string(l.ch), line, column, l.file)
    }
```

**Precedence:** `|>` binds **very loosely** — lower than everything except sequencing (`;`). It's a top-level expression structuring tool.

**Current parser precedence** (`internal/parser/parser.go:37-52`):
```
LOWEST(0) → LAMBDA(1) → LogicalOr(2) → LogicalAnd(3) → EQUALS(4) →
LESSGREATER(5) → CONS(6) → APPEND(7) → SUM(8) → PRODUCT(9) →
PREFIX(10) → CALL(11) → DotAccess(12)
```

Add `PIPE` between `LOWEST` and `LAMBDA`:
```
LOWEST(0) → PIPE(1) → LAMBDA(2) → LogicalOr(3) → ...
```

This ensures:
- `a |> f |> g` left-associates: `(a |> f) |> g`
- `a |> \x. x + 1` parses as `a |> (\x. x + 1)` (lambda has higher precedence, so the lambda body extends to the right)
- `a + b |> f` parses as `(a + b) |> f` (arithmetic binds tighter)

**AST representation:** Use existing `BinaryOp` node (`internal/ast/ast_expr.go:94`) with `Op: "|>"`. No new AST type needed.

**Elaboration (Surface → Core):** In `internal/elaborate/`, desugar `BinaryOp{Op: "|>", Left: x, Right: f}` to `App(f, [x])`. Simple, single rule.

### Interaction with effects

Pipe is transparent to effects. It's pure desugaring — no effect rows introduced:

```ailang
readFile("doc.xml")                        -- string ! {FS}
  |> parseXml                              -- XmlNode (pure)
  |> (\node. describeWithAI(node))         -- string ! {AI}
```

The type checker sees the desugared form and infers effects from the actual function calls. Effect rows compose: if the chain contains `! {FS}` and `! {AI}` steps, the overall expression has `! {FS, AI}`.

### Interaction with budgets

No budget impact. Pipe doesn't create scopes, doesn't consume effects, doesn't introduce calls. It's purely syntactic.

## Proposed std/option Improvements

Even without `x |> f(a)`, we can make common patterns more concise. These are useful independently of the pipe operator:

```ailang
-- unwrapOr: extract value with default (currently getOrElse but with confusing arg order)
export pure func unwrapOr[a](opt: Option[a], default: a) -> a {
  match opt { Some(x) => x, None => default }
}

-- flatten: collapse nested Option
export pure func flatten[a](opt: Option[Option[a]]) -> Option[a] {
  match opt { Some(inner) => inner, None => None }
}

-- isSomeAnd: test contained value
export pure func isSomeAnd[a](opt: Option[a], pred: (a) -> bool) -> bool {
  match opt { Some(x) => pred(x), None => false }
}
```

### Future: `x |> f(a)` form (v0.8.0+)

If/when AILANG decides to support either:
- Implicit currying (all multi-arg functions are auto-curried), OR
- Arg-splicing pipe (`x |> f(a)` → `App(f, [a, x])` via elaborator rewrite)

...then the pipe operator becomes strictly more powerful. The v0.7.3 `x |> f` form is forwards-compatible with both approaches. No breaking change needed.

The arg-splicing approach (Option C from feedback) is the most likely path since it doesn't require language-wide currying — just a local elaboration rule for `|>`. This would use the existing N-ary `App{Args: []CoreExpr}` in Core, appending the piped value to the Args slice.

## Implementation Plan

### Phase 1: Lexer + Parser (2-3 hours)
1. Add `PIPE_ARROW` token to `internal/lexer/token.go` (after PIPE at line ~79)
2. Add string repr: `PIPE_ARROW: "|>"`
3. Update lexer `case '|':` at `internal/lexer/lexer.go:162-169` to check `peekChar() == '>'`
4. Add `PIPE` precedence constant in `internal/parser/parser.go:37` (between LOWEST and LAMBDA)
5. Add precedence case in `token.go:Precedence()` for PIPE_ARROW
6. Register infix: `p.registerInfix(lexer.PIPE_ARROW, p.parsePipeExpression)` in parser.go
7. Implement `parsePipeExpression` — parse RHS as expression, wrap in `BinaryOp{Op: "|>"}`

### Phase 2: Elaboration (1-2 hours)
8. Handle `BinaryOp{Op: "|>"}` in `internal/elaborate/` — desugar to `App(Right, [Left])`
9. Verify type inference works on desugared form (no type checker changes needed)
10. Test effect propagation through pipe chains

### Phase 3: Option improvements (1 hour)
11. Add `unwrapOr`, `flatten`, `isSomeAnd` to `std/option.ail`
12. Register exports

### Phase 4: Documentation (1-2 hours)
13. Add pipe operator to teaching prompt with examples
14. Create `examples/runnable/pipe_operator.ail`
15. Update CHANGELOG.md

## Files to Modify

| File | Change | LOC Est |
|------|--------|---------|
| `internal/lexer/token.go` | Add PIPE_ARROW token + precedence | +5 |
| `internal/lexer/lexer.go` | Add `\|>` case in pipe handler | +5 |
| `internal/parser/parser.go` | Add PIPE precedence + register infix | +3 |
| `internal/parser/parser_expr.go` | Add `parsePipeExpression` | +15 |
| `internal/elaborate/elaborate.go` | Desugar pipe to App | +10 |
| `std/option.ail` | Add unwrapOr, flatten, isSomeAnd | +15 |
| `prompts/v0.7.3.md` | Add pipe operator section | +20 |
| Tests (various) | Parser, elaboration, integration | +60 |
| **Total** | | **~133** |

## Risks

1. **Lexer ambiguity** — `|>` contains `>` (comparison operator). Risk: low. The lexer processes `|` first, then peeks. `|>` is always lexed from the `|` case. Same pattern as `||` (OR).
2. **Precedence interaction** — `a |> b |> c` must left-associate: `(a |> b) |> c`. This is the default for the Pratt parser with left-associative precedence. `a |> \x. x + 1` parses as `a |> (\x. x + 1)` because lambda has higher precedence than pipe.
3. **PIPE vs PIPE_ARROW confusion** — `|` is used in ADT definitions and match arms. `|>` is only in expressions. The parser context disambiguates (type definitions vs expression parsing). No ambiguity.
4. **Limited without currying** — `x |> f` requires `f` to be arity-1. Multi-arg functions need lambda wrappers: `x |> (\y. f(a, y))`. This is verbose but correct. Users may expect `x |> f(a)` to work. Mitigation: document clearly; plan arg-splicing for v0.8.0+.

## Acceptance Criteria

- [ ] `42 |> show` produces `"42"` (bare function pipe)
- [ ] `"hello" |> (\s. s ++ " world")` works (lambda pipe)
- [ ] `Some(42) |> (\o. flatMap(\x. Some(x + 1), o))` works (Option with lambda wrapper)
- [ ] Effects propagate: pipe chain with `! {FS}` step type-checks correctly
- [ ] Left-associativity: `a |> f |> g` parses as `(a |> f) |> g`
- [ ] Pipe precedence lower than all arithmetic/comparison/logical operators
- [ ] `x |> 42` produces type error (not crash)
- [ ] `unwrapOr`, `flatten`, `isSomeAnd` added to std/option
- [ ] Teaching prompt updated with pipe examples
- [ ] Example file created and verified
- [ ] CHANGELOG.md updated

## Deferred to v0.8.0+

- **`x |> f(a)` arg-splicing form** — requires either currying or elaborator arg-append. Decision deferred until currying/calling conventions are settled.
- **`option { ... }` sugar** — Kotlin-style scoped Option unwrapping. Larger feature, separate design doc.
- **Global data-last convention** — standardizing all stdlib to `(config..., data)` for pipe composability. Separate discussion, touches every module.

## Related Documents

- [M-EFFECTFUL-LIST-COMBINATORS](m-effectful-list-combinators.md) — pipe composes with `mapE` etc. via lambda wrappers
- [M-CALL-SUGAR-OPTIONAL](../v0_8_0/m-call-sugar-optional.md) — related syntax change for function calls
- [M-STRING-INTERPOLATION](../v0_8_0/m-string-interpolation.md) — another DX syntax improvement

## References

- Elixir `|>` operator — pipes as last argument (requires Elixir's calling convention)
- F# `|>` operator — `x |> f` = `f x` (unary only, same as this proposal)
- OCaml `|>` operator — `x |> f` = `f x` (unary only)
- Haskell `&` operator — flip of `$`, `x & f` = `f x` (unary only)
- DocParse DX Feedback (Feb 2026) — 6 instances of 3-4 level nested match
- AILANG arity semantics: `internal/eval/eval_operations.go:46-47` (strict matching, no partial application)
- Core App representation: `internal/core/core.go:140-150` (N-ary `App{Args: []CoreExpr}`)
- Current lexer: `internal/lexer/lexer.go:162-169` (pipe case)
- Current parser precedence: `internal/parser/parser.go:37-52`
- Current std/option: `std/option.ail` (flatMap at line 20, map at line 13)
