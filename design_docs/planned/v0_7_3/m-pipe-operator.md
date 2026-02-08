# M-PIPE-OPERATOR: Pipe operator and Option chaining

**Status:** Planned
**Version:** v0.7.3
**Priority:** P2 — reduces nested match boilerplate (6 instances of 3-4 levels deep in DocParse)
**Estimated effort:** 6-8 hours
**Origin:** DocParse DX Feedback (Feb 2026)

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

## Proposed Solution: Pipe Operator `|>`

Add a pipe operator that passes the left-hand result as the **last** argument to the right-hand function:

```ailang
-- x |> f  desugars to  f(x)
-- x |> f(a)  desugars to  f(a, x)

findFirst(p, "w:pPr")
  |> flatMap(\ppr. findFirst(ppr, "w:numPr"))
  |> flatMap(\numPr. findFirst(numPr, "w:numId"))
  |> flatMap(\numId. getAttr(numId, "w:val"))
  |> getOrElse("")
```

This reads top-to-bottom, left-to-right — the natural data flow.

## Design Decisions

### Semantics: Last argument vs first argument

| Option | `x \|> f(a)` becomes | Precedent | Compatibility |
|--------|---------------------|-----------|---------------|
| Last arg | `f(a, x)` | Elixir, F#, OCaml | Works with `flatMap(fn, opt)` |
| First arg | `f(x, a)` | — | Works with `map(val, fn)` |

**Recommendation:** Last argument. This matches `std/option.flatMap(f, opt)` where the Option is the last parameter. Most AILANG stdlib follows the convention of "data last" for composability.

**However**, `std/option` currently has `flatMap(f, opt)` — the function first, data last. This is already pipe-compatible with last-arg semantics:

```ailang
someOption |> flatMap(\x. transform(x))  -- flatMap(\x. transform(x), someOption)
```

### Parsing

`|>` is a new binary operator. Implementation:

1. **Lexer:** Add `PIPE_ARROW` token for `|>`
2. **Parser:** Parse as left-associative binary operator with low precedence (below `++`, above `$`)
3. **Elaboration:** Desugar `a |> f` to `f(a)` and `a |> f(b)` to `f(b, a)` in Surface → Core

### Precedence

```
(lowest)
  $           -- application operator
  |>          -- pipe (NEW)
  or
  and
  == != < > <= >=
  ++ ::
  + -
  * / %
  not - (unary)
  f(x)        -- function application
(highest)
```

### Interaction with effects

Pipe should be transparent to effects:

```ailang
-- This should type-check with ! {FS, AI}
readFile("doc.xml")                           -- ! {FS}
  |> parseXml                                 -- pure
  |> flatMapE(\node. aiDescribe(node))        -- ! {AI}
```

The pipe doesn't introduce effects — it just reorders application.

### Alternative considered: `andThen` method on Option

Instead of a general pipe operator, add `andThen` to `std/option`:

```ailang
-- std/option
export pure func andThen[a, b](opt: Option[a], f: (a) -> Option[b]) -> Option[b] {
  match opt { Some(x) => f(x), None => None }
}
```

Usage with pipe: `findFirst(p, "w:pPr") |> andThen(\ppr. findFirst(ppr, "w:numPr"))`

**Decision:** Do both. `andThen` is a useful alias for `flatMap` with "data first" argument order (better for piping). The pipe operator is generally useful beyond Option.

## Implementation Plan

### Phase 1: Pipe operator (4-5 hours)
1. Add `PIPE_ARROW` (`|>`) token to `internal/lexer/token.go`
2. Add lexer rule in `internal/lexer/lexer.go` (already handles `|`, just add `|>`)
3. Add `PipeExpr` AST node or desugar in parser directly
4. Set precedence in parser (left-assoc, low precedence)
5. Elaborate: `a |> f` → `App(f, a)`, `a |> f(b)` → `App(App(f, b), a)`
6. Type checking: no special rules needed (it's just application)

### Phase 2: Option improvements (2-3 hours)
7. Add `andThen` to `std/option` (data-first flatMap alias)
8. Add `mapOr` to `std/option`: `mapOr(opt, default, f)` — map with fallback
9. Add `flatten` to `std/option`: `Option[Option[a]] -> Option[a]`
10. Update `std/option` exports

### Phase 3: Documentation (1-2 hours)
11. Add pipe operator to teaching prompt
12. Create `examples/runnable/pipe_operator.ail`
13. Update CHANGELOG.md

## Risks

1. **Parser complexity** — `|>` contains `>` which is also a comparison operator. Lexer must handle `|>` as a single token. Risk: low, lexer already handles multi-char tokens.
2. **Partial application** — `x |> f(a)` requires partial application or currying. If AILANG doesn't auto-curry, this might need `x |> (\y. f(a, y))` which is verbose. Mitigation: document the lambda wrapper pattern.
3. **Precedence conflicts** — `|>` near `|` (ADT pipe) could confuse users. Mitigation: different context (expressions vs type definitions).

## Acceptance Criteria

- [ ] `42 |> show` produces `"42"`
- [ ] `[1,2,3] |> map(\x. x * 2) |> filter(\x. x > 2)` works
- [ ] Option chaining with `|> flatMap(\x. ...)` works
- [ ] `|> andThen(\x. ...)` works for Option
- [ ] Effects propagate through pipe chains
- [ ] Teaching prompt updated
- [ ] Example file created and verified
- [ ] CHANGELOG.md updated

## References

- Elixir `|>` operator — same semantics (last argument)
- F# `|>` operator — same semantics (last argument)
- OCaml `|>` operator — same semantics
- Haskell `&` operator — flip of `$`, equivalent to `|>`
- DocParse DX Feedback — 6 instances of 3-4 level nested match
