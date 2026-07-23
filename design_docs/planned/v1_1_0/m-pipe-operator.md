# M-PIPE-OPERATOR: Pipe operator and Option chaining

**Status:** **EVIDENCE-GATED ICEBOX (Mark 2026-07-23: "I've never seen pipe be an issue, lets icebox it")** — data: ZERO `|>` parse errors AND zero occurrences of any kind across the entire banked eval corpus incl. the v0.30.0 baseline. The design is technically sound (Rev-1 objection refuted at HEAD — preserved in Quorum History) but demonstrated demand is the bar. REOPEN: `|>` appearing in eval banks, or a ratified human-authors audience. Third sibling in the week's pattern (?-op, block-let-separator, |>).
**Target version:** v1.1.0
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
| A1: Determinism | 0 | Desugars to an explicit Let + App; evaluation order is pinned left-to-right (see "Elaboration") |
| A2: Replayability | 0 | Traces show the underlying Let/App |
| A3: Effect Legibility | 0 | LHS effects run before RHS effects, matching the visual left-to-right order |
| A7: Machines First | +1 | Reduces nesting depth, easier for AI to parse/generate |
| A8: Minimal Syntax | -1 | Adds one new token/operator |
| A9: Cost Visibility | 0 | No hidden costs, just reordered application |
| A10: Composability | +1 | Enables left-to-right composition of any arity-1 functions |

**Net score: +1.** Justification for A8 (-1): The syntax addition is warranted because it eliminates a class of deeply-nested code that is hard for both humans and AIs to read/generate. The pipe is a small syntactic transform (desugars to Let + function application), adds no new Core node and no type-system change. See "Frozen-core justification" below for the routing analysis.

## Frozen-core justification (routing per PROGRAM.md)

The north star ([design_docs/PROGRAM.md](../../PROGRAM.md)) freezes the **motoko core** and routes
every improvement into one of three lanes. The routing table (§4) explicitly assigns language-level
friction — "Bad/unfixable error, missing builtin, **dialect trap**, type-system gap" — to the
**AILANG fix** lane, which is exactly this doc: a design doc in `ailang/design_docs/` plus
regression tests. This proposal touches zero motoko-core surface. So the frozen core is not at
stake in the PROGRAM.md sense.

The *spirit* of the objection still deserves a direct answer: even within AILANG, prefer a library
over new syntax. Analysis:

1. **AILANG has no user-operator or macro mechanism.** All infix operators are hardcoded in
   `internal/lexer/token.go` (`Token.Precedence()`, line 426) and registered in
   `internal/parser/parser.go` (lines 110-132). There is no facility for a stdlib module or
   extension to introduce an infix operator. Any infix operator is therefore inherently a
   lexer/parser change — there is no extension-shaped alternative *for the infix form*.
2. **The library alternative exists but does not solve the problem.** A stdlib function
   `pipe(x, f) = f(x)` is expressible today with zero parser changes. But chaining it nests
   inside-out — `pipe(pipe(pipe(x, f), g), h)` — which is precisely the reported friction. The
   value of `|>` is irreducibly syntactic: infix position plus left-associativity is what produces
   top-to-bottom, left-to-right reading. No library can provide that.
3. **The cost is bounded and additive.** One token, one lexer branch, one precedence entry (in two
   files that must stay in sync), one elaboration special-case reusing existing ANF machinery, one
   formatter precedence entry. No new AST node (reuses `BinaryOp`), no new Core node (reuses
   `Let` + `App`), no type-checker or effect-system change (inference runs on the desugared Core).
4. **The alternative of waiting for a user-operator mechanism is worse.** A general user-defined
   operator / macro facility would be a far larger core-surface addition than this single operator,
   and none is planned.

**Verdict: core syntax change justified, in the sanctioned AILANG-fix lane.** The A8 (-1) cost is
paid once for the most-requested missing operator; the extension/library route was evaluated and
cannot deliver the DX (point 2).

## Proposed Solution: Pipe Operator `|>`

### Scope: `x |> f` only (simple pipe)

**`x |> f` evaluates `x`, then evaluates `f`, then applies.** No multi-arg form, no arg-splicing.

```ailang
findFirst(p, "w:pPr")
  |> (\x. flatMap(\ppr. findFirst(ppr, "w:numPr"), x))
  |> (\x. flatMap(\numPr. findFirst(numPr, "w:numId"), x))
  |> (\x. flatMap(\numId. getAttr(numId, "w:val"), x))
  |> (\x. getOrElse(x, ""))
```

This reads top-to-bottom, left-to-right — the natural data flow. Each stage is an arity-1
callable: a bare function name or a (parenthesized) lambda.

### Why NOT `x |> f(a)` in this version

The previous draft proposed `x |> f(a)` desugaring to `f(a, x)` (insert as last argument). This is
incompatible with AILANG's runtime calling convention for under-application:

- Over-application auto-curries (M-DOCPARSE-DX M1, `internal/eval/eval_operations.go:62-109`):
  `f(a, b)` where `f = \x. \y. ...` applies in two batches.
- **Under-application is still a runtime error** (`internal/eval/eval_operations.go:111-113`):
  ```go
  if len(args) < len(fn.Params) {
      return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
  }
  ```
  So `flatMap(f)` (arity 2 called with 1 arg) does not return a closure — it errors. `x |> f(a)`
  would require either currying semantics or a novel elaborator arg-splicing rule whose
  interaction with type inference is untested.

**Decision:** Ship the simple form `x |> f` only. Defer `x |> f(a)` (see "Deferred").

## Design Decisions

### Semantics: `x |> f` desugars to LHS-first Let + App

A naive desugar to `App(f, [x])` has an **evaluation-order bug**: `evalCoreApp`
(`internal/eval/eval_operations.go:15-38`) evaluates `app.Func` *before* `app.Args`. If the RHS is
a side-effecting expression that evaluates to a function — e.g.
`logAndGet() |> (chooseHandler())` — the RHS would run before the piped LHS, violating the
left-to-right reading the operator exists to provide (Axioms A1/A3).

**The desugar therefore pins LHS-first order with Let bindings** (Core `Let` is
`internal/core/core.go:111-116`: `{Name string; Value CoreExpr; Body CoreExpr}`):

```
x |> f   →   Let(fresh_l, ⟦x⟧, Let(fresh_r, ⟦f⟧, App(Var(fresh_r), [Var(fresh_l)])))
```

with the standard ANF simplification that **atomic** operands (`core.IsAtomic`,
`internal/core/core.go:559-566`: `Var`, `Lit`, `Lambda`, `DictRef`, `VarGlobal`) need no binding.
In particular a lambda RHS — the common case — is atomic, so `x |> (\y. ...)` desugars to
`Let(fresh_l, ⟦x⟧, App(Lambda(...), [Var(fresh_l)]))` and a bare-variable pipe `a |> f` desugars
to plain `App(Var(f), [Var(a)])` with no overhead.

This falls out of existing machinery: the elaborator's `normalizeBinaryOp`
(`internal/elaborate/expr_simple.go:115-190`) already calls `normalizeToAtomic(binop.Left)`
**before** `normalizeToAtomic(binop.Right)` and wraps the result with the accumulated bindings in
that order (`wrapWithBindings(result, append(leftBinds, rightBinds...))`). The `|>` implementation
is a special case at the top of `normalizeBinaryOp` (alongside the existing `&&`/`||` special
case) that emits `App(right, []core.CoreExpr{left})` instead of an `Intrinsic` — the LHS-first
Let-wrapping comes for free from the shared binding logic.

The RHS of `|>` must be a **single-argument callable**. If the RHS is a literal, record, list, or
other non-callable expression, it's a type error (detected by the type checker on the desugared
Core, not the parser).

### Pipe-friendly usage without currying

Because under-application errors (no partial application), multi-arg stdlib functions like
`flatMap(f, opt)` cannot be piped directly. The pipe-friendly form is a **fresh arity-1 lambda**
written inline at the pipe boundary:

```ailang
someOption |> (\o. flatMap(\x. transform(x), o))
```

Each `(\o. flatMap(f, o))` is an arity-1 lambda; the pipe applies the left value as its single
argument. This works with today's semantics, needs no stdlib changes, and is the idiom the
teaching prompt will show. It is more verbose than Elixir-style pipes, but honest about AILANG's
calling convention — the lambda wrapper *is* the partial application, made explicit.

### Why this is still a DX win

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

The pipe version reads top-to-bottom and each line does one thing. The nesting is gone.

### Parsing

**Lexer** (`internal/lexer/lexer.go:182-189` at commit 76c9e70f1):
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
Add a `peekChar() == '>'` branch between the `||` check and the `PIPE` fallback, emitting a new
`PIPE_ARROW` token — same two-char pattern as `||`.

**Precedence.** `|>` binds looser than every other operator. The current precedence bands
(`internal/parser/parser.go:53-70`) are C-standard **dedicated** bands with a hard invariant —
"These MUST match the values returned by token.Precedence()" (`internal/lexer/token.go:426-461`):

```
LOWEST(0) → LAMBDA(1) → LogicalOr(2) → LogicalAnd(3) → [4 reserved: bitwise |] →
BitwiseXor(5) → BitwiseAnd(6) → EQUALS(7) → LESSGREATER(8) → SHIFT(9) →
CONS(10) → APPEND(11) → SUM(12) → PRODUCT(13) → PREFIX(14) → CALL(15) →
DotAccess(16) → HIGHEST(17)
```

Rev-0 proposed inserting a new `PIPE(1)` band and shifting everything up. That would renumber
**every** constant in both `parser.go` and `token.Precedence()` — high-churn and error-prone.
Instead, **`PIPE_ARROW` shares band 1 with `LAMBDA`**. This is safe because `BACKSLASH`'s
precedence value is never consulted in the infix loop — `BACKSLASH` has no registered infix parse
function, so `parseExpression`'s loop (`internal/parser/parser_expr.go:44`) breaks on it
regardless. Concretely:
- `parser.go`: add `PIPE = 1` (documented as sharing the lambda band) and
  `p.registerInfix(lexer.PIPE_ARROW, p.parseInfixExpression)`.
- `token.go` `Precedence()`: `case PIPE_ARROW: return 1`.

This yields:
- `a |> f |> g` → `(a |> f) |> g`. Left-associativity is the Pratt default here:
  `parseInfixExpression` (`internal/parser/parser_expr.go:587-599`) parses the RHS at the *same*
  precedence, and the loop continues only on *strictly greater* peek precedence.
- `a + b |> f` → `(a + b) |> f`; `a |> b || c` → `a |> (b || c)`. Everything binds tighter than
  pipe.
- `a |> \x. x + 1` → `a |> (\x. x + 1)` — but see the lambda caveat below.

**Lambda RHS caveat (corrects a Rev-0 claim).** `parseBackslashLambda`
(`internal/parser/parser_expr.go:523-585`) parses the lambda body with `parseExpression(LOWEST)`
— the body extends *maximally* to the right. So in a chain, an unparenthesized lambda swallows
the rest of the chain:

```
a |> \x. x + 1 |> g     parses as     a |> (\x. ((x + 1) |> g))     — NOT ((a |> \x. x+1) |> g)
```

For a *trailing* lambda stage this is exactly what the user wants; for a *middle* stage it is
not. The rule, enforced by documentation, teaching prompt, and an acceptance test: **parenthesize
lambda stages in chains** — `a |> (\x. x + 1) |> g`. (This is the same behavior lambdas already
exhibit with every other trailing construct; the pipe adds no new parser rule.)

**AST representation:** reuse the existing `BinaryOp` node (`internal/ast/ast_expr.go:100`) with
`Op: "|>"`. No new AST type.

### Elaboration (Surface → Core)

Dispatch: `internal/elaborate/expressions.go:71-72` routes `*ast.BinaryOp` to `normalizeBinaryOp`
(`internal/elaborate/expr_simple.go:115`). Add a `binop.Op == "|>"` special case at the top
(before the intrinsic-mapping switch, alongside the `&&`/`||` short-circuit case) that:
1. `normalizeToAtomic(binop.Left)` → `left, leftBinds`
2. `normalizeToAtomic(binop.Right)` → `right, rightBinds`
3. returns `wrapWithBindings(&core.App{Func: right, Args: []core.CoreExpr{left}}, append(leftBinds, rightBinds...))`

This must **not** fall through to the `default:` branch of the operator switch, which emits a
`core.BinOp` with the raw op string — that would leak `"|>"` into the evaluator.

**Order relative to type/effect inference:** elaboration runs before type checking; inference and
effect checking operate on the desugared Core (`Let`/`App`), so no type-checker or effect-checker
change is needed. Effects propagate exactly as for any Let/App: a chain containing `! {FS}` and
`! {AI}` stages infers `! {FS, AI}`, and the Let sequencing pins the *order* the effects fire:
strictly left-to-right.

### Interaction with budgets

No budget impact. Pipe introduces at most two Let bindings per stage (zero for the common
atomic-operand case), no scopes, no effect consumption.

## Proposed std/option Improvements

Verified current API (`std/option.ail` at 76c9e70f1): `map(f, opt)` (line 13),
`flatMap(f, opt)` (line 20), `getOrElse(opt, default)` (line 27), `isSome` (34), `isNone` (38),
`filter(pred, opt)` (42). None of the additions below exist yet.

Rev-0 also proposed `unwrapOr(opt, default)` on the premise that `getOrElse` had a confusing
argument order — but `getOrElse` is *already* `(opt, default)`, so `unwrapOr` would be an exact
duplicate. **Dropped.** Remaining additions (useful independently of the pipe):

```ailang
-- flatten: collapse nested Option
export pure func flatten[a](opt: Option[Option[a]]) -> Option[a] {
  match opt { Some(inner) => inner, None => None }
}

-- isSomeAnd: test contained value
export pure func isSomeAnd[a](opt: Option[a], pred: (a) -> bool) -> bool {
  match opt { Some(x) => pred(x), None => false }
}
```

Note `flatten` is arity-1 and therefore directly pipeable: `nested |> flatten`.

## Verification and Conflict Surface

All commands run in this worktree at commit **76c9e70f1**. Stale Rev-0 line numbers corrected
throughout.

| # | Claim | Command | Evidence / Result |
|---|-------|---------|-------------------|
| 1 | Gap is real: no pipe operator at HEAD | `ailang check /tmp/pipe_repro.ail` on a module containing `42 \|> show` | `PAR_UNEXPECTED_TOKEN at 4:6: expected next token to be }, got \| instead`. Mission reality-check additionally confirmed `PIPE_ARROW`/`parsePipeExpression` do not exist in `internal/{lexer,parser,elaborate}`. |
| 2 | Lexer `case '\|'` location + shape | `grep -n "case '\|'" -A 10 internal/lexer/lexer.go` | Lines **182-189** (Rev-0 said 162-169 — stale). Handles `\|\|` then falls back to `PIPE`. Two-char pattern to copy is right there. |
| 3 | Precedence constants | `sed -n '50,80p' internal/parser/parser.go` | Lines **53-70** (Rev-0 said 37-52 — stale). Table gained `BitwiseXor(5)`, `BitwiseAnd(6)`, `SHIFT(9)`, `HIGHEST(17)`; **band 4 is reserved for a future bitwise `\|`**. Comment: "These MUST match the values returned by token.Precedence()". Rev-0's insert-and-renumber plan replaced with shared band 1 (see Parsing). |
| 4 | Precedence lookup table | `sed -n '420,480p' internal/lexer/token.go` | `Token.Precedence()` at **token.go:426-461** — a parallel hardcoded switch. Any new operator must be added in BOTH files. |
| 5 | Infix registration + left-associativity | `grep -n registerInfix internal/parser/parser.go`; `sed -n '587,599p' internal/parser/parser_expr.go` | Registrations at parser.go:110-132. `parseInfixExpression` builds `ast.BinaryOp` and parses RHS at the *same* precedence; `parseExpression` loop (parser_expr.go:44) continues only on strictly-greater peek precedence → same-precedence operators left-associate. `a \|> f \|> g` = `(a \|> f) \|> g` confirmed by construction. |
| 6 | Lambda RHS behavior | `sed -n '523,585p' internal/parser/parser_expr.go` | `parseBackslashLambda` parses the body with `parseExpression(LOWEST)` — body extends maximally right. `a \|> \x. x + 1` parses as claimed, but an unparenthesized lambda mid-chain swallows the rest of the chain. Documented as the parenthesization rule + acceptance test. Rev-0's blanket "left-associates" claim corrected. |
| 7 | `BinaryOp` consumers (exhaustive) | `grep -rn "BinaryOp" --include="*.go" internal/ \| grep -v _test` | 19 files. **Op-string-sensitive (need `\|>` handling):** `elaborate/expr_simple.go` (`normalizeBinaryOp` — the implementation site; its `default:` leaks unknown ops as `core.BinOp`, so `\|>` must be special-cased before it), `format/precedence.go` (`binaryPrecedence` defaults unknown ops to `precLowest` and `rightAssociative` returns false — add an explicit `"\|>"` case so the formatter parenthesizes chains correctly), `types/inference.go:283` (surface-AST inference; `default:` → `unknown operator: %s` error) and `eval/eval_simple.go:151` (`SimpleEvaluator.evalBinOp`) — both are on the legacy surface-AST path, NOT the live path (REPL and pipeline use `CoreEvaluator` + Core-side inference; verified `grep -rn CoreEvaluator internal/repl/` → repl.go:66,90 etc.); they see only already-elaborated code or none, but should be given explicit cases or left to their loud-error defaults — either way no silent misbehavior. **Op-agnostic (visit Left/Right only, verified safe):** `types/inference_helpers.go:484` (free vars), `types/ifc_check.go:163` (label join), `elaborate/scc.go:197`, `lsp/index.go:124`, `ast/print.go`, `ast/ast_expr.go` (definition), `parser/testutil.go:160`, `testing/harness.go:200`, `testing/executor_helpers.go:375`, `format/{attach,anchors,envelope,format}.go` (structural walks). No exhaustive switch without a default was found — nothing panics on an unknown op. |
| 8 | Elaboration order vs type/effect inference | `sed -n '115,190p' internal/elaborate/expr_simple.go`; dispatch at `elaborate/expressions.go:71-72` | `normalizeBinaryOp` runs during Surface→Core elaboration, i.e. before Core type/effect inference. It already normalizes Left before Right and wraps bindings left-first: `wrapWithBindings(result, append(leftBinds, rightBinds...))` — the LHS-first Let order needed for objection-2's fix is the existing convention. `core.Let` shape confirmed at core.go:111-116; `core.IsAtomic` at core.go:559-566 includes `Lambda` → lambda RHS needs no binding. |
| 9 | Evaluator order (the objection-2 crux) | `sed -n '1,45p' internal/eval/eval_operations.go` | `evalCoreApp` evaluates `app.Func` (line 17) **before** `app.Args` (lines 31-38). Confirms naive `App(f, [x])` runs RHS-first; hence the Let-binding desugar. |
| 10 | Arity semantics drift | `grep -n 'fn.Params' internal/eval/eval_operations.go` | Rev-0's cited check (`len(args) != len(fn.Params)` at "46-47") no longer exists. Current: **over**-application auto-curries (M-DOCPARSE-DX M1, lines 62-109); **under**-application still errors (lines 111-113). "No partial application" still holds for the under-applied `f(a)` case the doc relies on; wording and citations updated. |
| 11 | `\|`/`\|\|` no ambiguity | `grep -rn 'lexer.PIPE\b' internal/parser/*.go` | Single `\|` (PIPE) is consumed in: effect rows (`parser_effect.go:198`), record update `{r \| f: v}` (`parser_literals.go:337-343,440`), ADT alternatives (`parser_expr.go:414`), error recovery (`parser_error.go:220`). In every such position the token after `\|` is an identifier/constructor — `\|` immediately followed by `>` is a parse error in all current grammar positions, so lexing `\|>` as one token changes the meaning of no valid program. `\|\|` is matched first in the lexer (line 183), and PIPE has no infix registration in expression context. |
| 12 | Current `std/option` API | `grep -n 'export\|func ' std/option.ail` | `map(f,opt)`:13, `flatMap(f,opt)`:20, `getOrElse(opt,default)`:27, `isSome`:34, `isNone`:38, `filter(pred,opt)`:42. `unwrapOr`/`flatten`/`isSomeAnd` do **not** exist. `getOrElse` is already `(opt, default)` → Rev-0's `unwrapOr` was a duplicate; dropped. |
| 13 | AST/Core citations | `grep -n 'type BinaryOp' internal/ast/ast_expr.go`; `grep -n 'type App struct' -A 5 internal/core/core.go` | `ast.BinaryOp` at **ast_expr.go:100** (Rev-0 said 94). `core.App` at core.go:141-145 (N-ary `Args []CoreExpr`). |
| 14 | M-CONCAT-DISAMBIG scope conflict | `grep -rln 'm-concat' design_docs/` | `m-concat-disambiguation` was **implemented in v0.13.0** (`design_docs/implemented/v0_13_0/`). Rev-0's "absorbed into Phase 1" reference was stale; removed from scope and Related Documents. |

## Implementation Plan

### Phase 1: Lexer + Parser (2-3 hours)
1. Add `PIPE_ARROW` token to `internal/lexer/token.go` (near PIPE) + string repr `"|>"`
2. Add `case PIPE_ARROW: return 1` to `Token.Precedence()` (token.go:426)
3. Update lexer `case '|':` (`internal/lexer/lexer.go:182-189`) with a `peekChar() == '>'` branch
4. Add `PIPE = 1` precedence constant in `internal/parser/parser.go:53-70` block, documented as
   sharing the LAMBDA band (no renumbering — see Parsing rationale)
5. Register infix: `p.registerInfix(lexer.PIPE_ARROW, p.parseInfixExpression)` (parser.go:110-132
   block). The generic `parseInfixExpression` suffices — no bespoke `parsePipeExpression` needed;
   it produces `BinaryOp{Op: "|>"}` directly.

### Phase 2: Elaboration (1-2 hours)
6. Add `binop.Op == "|>"` special case at the top of `normalizeBinaryOp`
   (`internal/elaborate/expr_simple.go:115`): normalize Left then Right to atomic, emit
   `core.App{Func: right, Args: [left]}`, wrap with left-then-right bindings (LHS-first Let form)
7. Add `"|>"` cases to `internal/format/precedence.go` (`binaryPrecedence`, left-associative)
8. Test: type inference and effect propagation on desugared form (no checker changes expected)
9. Test: **evaluation order** — side-effecting LHS and side-effecting RHS-that-returns-a-function;
   assert LHS effect fires first

### Phase 3: Option improvements (1 hour)
10. Add `flatten`, `isSomeAnd` to `std/option.ail` (`unwrapOr` dropped — duplicates `getOrElse`)

### Phase 4: Documentation (1-2 hours)
11. Add pipe operator to teaching prompt (incl. the parenthesize-lambda-stages rule)
12. Create `examples/runnable/pipe_operator.ail`
13. Update CHANGELOG.md

## Files to Modify

| File | Change | LOC Est |
|------|--------|---------|
| `internal/lexer/token.go` | Add PIPE_ARROW token + `Precedence()` case | +5 |
| `internal/lexer/lexer.go` | Add `\|>` branch in pipe case (182-189) | +5 |
| `internal/parser/parser.go` | Add PIPE precedence const + register infix | +3 |
| `internal/elaborate/expr_simple.go` | `\|>` special case in `normalizeBinaryOp` | +12 |
| `internal/format/precedence.go` | `\|>` precedence for formatter | +3 |
| `std/option.ail` | Add `flatten`, `isSomeAnd` | +10 |
| `prompts/` teaching prompt | Pipe operator section | +20 |
| Tests (parser, elaboration, eval-order, integration) | | +70 |
| **Total** | | **~128** |

## Risks

1. **Lexer ambiguity** — none: `|>` lexes from the `|` case after the `||` check; no current
   grammar position allows `|` directly followed by `>` (Conflict Surface #11).
2. **Precedence-table sync** — parser.go constants and token.go `Precedence()` must stay in sync
   (both files change; the shared-band-1 choice avoids renumbering 17 constants).
3. **Lambda-in-chain capture** — an unparenthesized `\x.` stage swallows the rest of the chain
   (Conflict Surface #6). Mitigation: documented rule + acceptance test; consider a
   formatter/lint hint later.
4. **Formatter round-trip** — `format/precedence.go` defaults unknown ops to `precLowest`; without
   an explicit `"|>"` case, formatting could drop needed parens. In scope (Phase 2, step 7).
5. **Limited without currying** — `x |> f` requires `f` arity-1; multi-arg functions need lambda
   wrappers. Verbose but correct; document clearly, defer arg-splicing.

## Acceptance Criteria

- [ ] `42 |> show` produces `"42"` (bare function pipe)
- [ ] `"hello" |> (\s. s ++ " world")` works (lambda pipe)
- [ ] `Some(42) |> (\o. flatMap(\x. Some(x + 1), o))` works (Option with lambda wrapper)
- [ ] **Evaluation order pinned:** with `lhs()` printing `"L"` then returning a value, and `rhs()`
      printing `"R"` then returning an arity-1 function, `lhs() |> (rhs())` prints `"L"` then
      `"R"` (LHS before RHS — the Let-binding desugar, not naive App order)
- [ ] Effects propagate: pipe chain with `! {FS}` step type-checks correctly
- [ ] Left-associativity: `a |> f |> g` parses as `(a |> f) |> g` (non-lambda stages)
- [ ] Lambda-stage rule: `a |> (\x. x + 1) |> g` chains; test documents that the unparenthesized
      form binds the tail into the lambda body
- [ ] Pipe precedence lower than all arithmetic/comparison/logical operators
- [ ] `x |> 42` produces a type error (not a crash, not a leaked `core.BinOp`)
- [ ] Formatter round-trips `(a |> f) |> g` and `a |> (b |> g)` without changing parse
- [ ] `flatten`, `isSomeAnd` added to std/option; `nested |> flatten` works
- [ ] Teaching prompt updated with pipe examples
- [ ] Example file created and verified
- [ ] CHANGELOG.md updated

## Deferred

- **`x |> f(a)` arg-splicing form** — requires either currying or an elaborator arg-append rule.
  The `x |> f` form is forwards-compatible with both. Arg-splicing (appending the piped value to
  the existing N-ary `App.Args`) is the likely path since it needs no language-wide currying.
- **`option { ... }` sugar** — Kotlin-style scoped Option unwrapping. Separate design doc.
- **Global data-last convention** — standardizing stdlib arg order for pipe composability.
  Separate discussion, touches every module.

## Related Documents

- [M-EFFECTFUL-LIST-COMBINATORS](../../implemented/v0_7_3/m-effectful-list-combinators.md) (implemented v0.7.3) — pipe composes with `mapE` etc. via lambda wrappers
- [M-CALL-SUGAR-OPTIONAL](m-call-sugar-optional.md) — related syntax change for function calls
- [PROGRAM.md](../../PROGRAM.md) — routing lanes; this doc is an AILANG-fix-lane item

## References

- F# / OCaml `|>` — `x |> f` = `f x` (unary form, same as this proposal); Haskell `&`; Elixir `|>`
  (arg-splicing form, requires their calling convention — deferred here)
- DocParse DX Feedback (Feb 2026) — 6 instances of 3-4 level nested match
- Under-application error: `internal/eval/eval_operations.go:111-113`; auto-curry (over-application):
  `eval_operations.go:62-109` (M-DOCPARSE-DX M1)
- `evalCoreApp` Func-before-Args order: `internal/eval/eval_operations.go:15-38`
- Core `App`: `internal/core/core.go:141-145`; Core `Let`: `core.go:111-116`; `IsAtomic`: `core.go:559-566`
- Lexer pipe case: `internal/lexer/lexer.go:182-189`; precedence: `internal/parser/parser.go:53-70`
  + `internal/lexer/token.go:426-461`
- Elaborator: `internal/elaborate/expressions.go:71-72` → `expr_simple.go:115-190`
- Current `std/option.ail`: map:13, flatMap:20, getOrElse:27, isSome:34, isNone:38, filter:42

## Quorum History

- **Rev-0** (this doc's first quorum, iter-89): **rejected 0-2**.
  - `gpt5-6-sol`: missing Verification & Conflict Surface analysis; no frozen-core/extension-lane
    justification for a syntax addition; stale line-number citations.
  - `gemini-3-1-pro`: eval-order bug — `App(f, [x])` desugar evaluates the RHS before the piped
    LHS (`evalCoreApp` evaluates `Func` before `Args`), violating A1/A3 for effectful RHS.
- **Rev-1** (this revision): added the Verification and Conflict Surface section (14 rows, all
  command-backed at 76c9e70f1) and the frozen-core justification (verdict: justified,
  AILANG-fix lane); changed the desugar to the LHS-first Let-binding form and pinned it with an
  evaluation-order acceptance test; corrected stale citations (lexer 182-189, precedence 53-70,
  BinaryOp ast_expr.go:100, arity check now auto-curry + under-application error); replaced the
  insert-and-renumber precedence plan with shared band 1; corrected the lambda-RHS associativity
  claim; dropped `unwrapOr` (duplicate of existing `getOrElse`) and the stale M-CONCAT-DISAMBIG
  scope reference (implemented in v0.13.0); trimmed the currying stream-of-consciousness to the
  final resolution. Goes back to a fresh quorum.
- **Rev-1 re-quorum** (iter-89, the one allowed re-quorum): **1 pass / 1 reject → BLOCKED**.
  - `gemini-3-1-pro`: **PASS** (eval-order objection resolved by the Let-binding desugar).
  - `gpt5-6-sol`: **reject**, but a much narrower objection than Rev-0 — it asserted that the
    only *verified* arity behavior is a runtime under-application error, so the claims that
    `x |> 42` is a type error and that no type-checker change is needed are unsupported and
    "may leave invalid pipes to fail only during evaluation."
- **Controller verification of the remaining objection (2026-07-23, HEAD 76c9e70f1):** the
  objection's premise is **REFUTED by direct test**. The desugar emits `App(right, [left])`, and
  plain application already rejects both failure modes at **type-check time**, not runtime:
  - Non-callable RHS (`x |> 42` → `App(42,[x])`): `ailang check` fails at check-time —
    `No instance for Num[int -> int]` (the checker unifies the callee with a function type and
    finds `42` is not one). So `x |> 42` IS a check-time type error, exactly as the doc claims.
  - Under-application / arity>1 RHS (`x |> f` where `f` is arity-2): `ailang check` fails at
    check-time with `TC_ARITY_001: function expects 2 argument(s), but 1 provided` +
    "AILANG has no partial application" hint.
  Therefore the doc's acceptance criterion ("`x |> 42` produces a type error, not a crash") and
  the "no type-checker change needed" claim are **correct at HEAD**. The reviewer's concern does
  not hold. **Per the QUORUM-AT-PICK gate (one re-quorum is the bound; still-rejected → park), the
  item is parked `needs-human-review` rather than force-passed by the controller** (the controller
  passed this doc both rounds and is not a neutral adjudicator of a residual reviewer reject).
  **Recommended unpark:** a human confirms the T1/T2 evidence above and routes the doc straight to
  sprint-planner — no further design change is required. Optionally the acceptance list can add a
  golden test pinning check-time rejection of both cases to make the claim regression-proof.
