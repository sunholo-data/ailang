# M-LIST-COMPREHENSIONS: List Comprehension Syntax

**Status**: Planned
**Target**: v0.25.0
**Priority**: P1 (Medium) — high-frequency eval gap, not release-blocking
**Estimated**: 3 days (2x of 1.5-day initial guess)
**Dependencies**: None (pure desugaring onto existing `std/list` primitives)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

List comprehensions are **pure syntactic sugar**: every comprehension desugars
deterministically into existing, already-shipped `std/list` calls (`map`,
`filter`, `flatMap`) during elaboration. No new runtime semantics, no new
effect behaviour, no new type rules — the elaborated AST is indistinguishable
from hand-written `map`/`filter`/`flatMap`.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Desugars to deterministic stdlib calls; no nondeterminism introduced |
| A2: Replayability | 0 | Traces are produced by the desugared `map`/`filter` calls, unchanged |
| A3: Effect Legibility | 0 | Body effects flow through `map`/`flatMap` exactly as if hand-written; nothing hidden |
| A4: Explicit Authority | 0 | No capabilities granted; sugar carries no authority of its own |
| A5: Bounded Verification | +1 | Desugars to well-typed primitives; type/effect checking stays local and unchanged |
| A6: Safe Concurrency | 0 | No concurrency surface touched |
| A7: Machines First | +1 | Matches the syntax distribution AI agents are trained on (Python/Haskell), reducing synthesis retries and token cost — AI code synthesis is the primary user |
| A8: Minimal Syntax | −1 | Adds surface syntax. Mitigated: zero new semantics, one localized grammar production, desugars away before type checking |
| A9: Cost Visibility | 0 | Allocation cost identical to the desugared form |
| A10: Composability | +1 | Composes with all existing list operations; generators and guards chain naturally |
| A11: Structured Failure | 0 | Errors remain the typed errors of the desugared primitives |
| A12: System Boundary | 0 | No boundary crossings |

**Net Score: +2** → **Decision: Proceed to implementation**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced — desugars to deterministic primitives
- [x] A3 (Effects): No hidden side effects — body effects propagate through `map`/`flatMap`
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Scores +1 — optimizes for machine (AI) synthesis success, not human convenience

The single −1 (A8) is on a non-hard-violation axis and is bounded: the feature
adds exactly one grammar production and disappears during elaboration.

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

**Result: +2 → Proceed.**

## Problem Statement

AILANG has no list comprehension syntax. Comprehensions are one of the most
common collection-transformation idioms in the languages AI agents are trained
on (Python, Haskell, Erlang, Scala, Elixir). When an agent reaches for the
idiom it is met with a hard parse error, forcing a retry and burning tokens.

**Current State (verified with `ailang check` against v0.24.0, commit d4ba009):**

```ail
-- [x*2 | x <- xs]
let xs = [x * 2 | x <- [1,2,3]];
```
```
PAR_UNEXPECTED_TOKEN: expected next token to be ,, got | instead
  Suggestion: Add or correct the , token
```

```ail
-- nested: [(x,y) | x <- [1,2], y <- [3,4]]
let pairs = [(x,y) | x <- [1,2], y <- [3,4]];
```
```
PAR_UNEXPECTED_TOKEN: expected next token to be ,, got | instead
```

The parser sees `[`, parses the first expression, then expects `,` or `]`
(list-literal continuation) and fails on `|`. The error suggestion ("Add or
correct the `,` token") actively misleads the agent toward a wrong fix.

The working idiom today is nested `map`/`filter`/`flatMap`, which all exist and
type-check cleanly (verified — see Conflict Surface fixtures):

```ail
import std/list (map, filter, flatMap)
let r = map(\x. x * 2, filter(\x. x % 2 == 0, xs));
```

**Impact:**
- **Who:** Every AI agent generating AILANG list transformations; this is the
  primary product audience (autonomous code synthesis).
- **Significance:** Comprehensions are a top-tier Python idiom. The misleading
  parse error converts a one-line idiom into a multi-turn debugging loop. This
  is precisely the class of eval gap that the v0.24.0 `m-prompt-*` and
  `m-eval-*` docs are chartered to close — but a prompt can only *teach around*
  the gap, whereas this closes it.

> **Note on this doc's provenance:** created via the design-doc-creator
> coordinator pipeline. The dispatching task carried no specific feature spec,
> so the topic was selected by probing the live compiler for a genuine,
> verifiable gap (every language claim below is backed by an `ailang check`/`run`
> transcript). List comprehensions surfaced as a real, unowned, well-scoped gap
> aligned with the current eval/LLM-iteration milestone theme.

## Goals

**Primary Goal:** Let AILANG agents and humans write list comprehensions
(`[expr | generators, guards]`) that desugar to existing `std/list` primitives,
eliminating a high-frequency parse-error class.

**Success Metrics:**
- `[x*2 | x <- xs]` and `[(x,y) | x <- xs, y <- ys, x < y]` parse, type-check,
  and run, producing results identical to the equivalent `map`/`filter`/`flatMap`.
- Zero regressions in `make verify-examples` and the full parser test suite.
- The eval benchmark(s) that currently fail on comprehension syntax flip to pass
  (measure on the recent-date eval segment, not the all-time aggregate).
- The misleading "Add or correct the `,` token" suggestion no longer fires for
  comprehension-shaped input.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Desugar-only (no new Core node) | Keeps type system, effects, traces, codegen untouched | compiler | design | high |
| Generator binder is a **pattern**, not just an identifier | Determines whether `[x | (x,_) <- pairs]` works; affects elaboration shape | human | design | med |
| Guards desugar to `filter`, multi-generator to `flatMap` | Fixes the semantic ordering (cartesian product, left-to-right) | compiler | design | med |
| Scope of v0.25.0: generators + guards only; **no** range sugar `[1..n]` | Bounds the parser surface; `[1..n]` is a separate unowned gap | human | design | high |
| `let` bindings inside comprehensions (`[y | x <- xs, let y = f(x)]`) in or out | Affects grammar and desugar; common in Haskell | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Generator binder grammar**: identifier-only (M1, minimal) vs. full pattern
      destructuring (`(a,b) <- pairs`, constructor patterns). Recommendation:
      reuse the existing `let`-pattern parser (see sibling doc
      `m-let-pattern-destructuring`) so binders and `let` share one pattern grammar.
- [ ] **Range sugar `[1..n]` is OUT of scope** — confirm. It is a *separate*
      verified gap (`[1..10]` → `PAR_UNEXPECTED_TOKEN ... got .. instead`) and
      `range` is not exported by `std/list` (IMP010). Bundling it doubles the
      parser surface and the conflict analysis. Recommend a follow-up doc.
- [ ] **Comprehension `let` bindings** in or out for v0.25.0.

## Solution Design

### Overview

Add a single grammar production inside the list-literal parser: after parsing
the head expression, if the next token is `|`, parse a comprehension instead of
a list literal. Lower the comprehension to `map`/`filter`/`flatMap` calls during
elaboration. Nothing downstream of elaboration changes.

Grammar (informal):

```
listLiteral   := "[" [ expr ( "," expr )* ] "]"
comprehension := "[" expr "|" qualifier ( "," qualifier )* "]"
qualifier     := generator | guard
generator     := pattern "<-" expr
guard         := expr            -- of type bool
```

### Architecture

**Components:**
1. **Lexer**: no change. `|` (`PIPE`) and `<-` already tokenize. (`<-` is used
   today by effect/do-notation; confirm it lexes inside `[` — it does, lexing is
   context-free.)
2. **Parser** (`internal/parser/parser_literals.go`): extend `parseListLiteral`
   (registered at `parser.go:81`) — after the head expression, branch on `PIPE`.
   Add `parseComprehension` producing a new surface AST node `*ast.ListComp`
   (`internal/ast/`) holding `Head ast.Expr` and `Quals []ast.Qualifier`.
3. **Elaboration** (`internal/elaborate/`): a `desugarListComp` pass that lowers
   `*ast.ListComp` to nested `map`/`filter`/`flatMap` Core application **before**
   type checking. After this pass no `*ast.ListComp` survives, so the type
   checker, effect checker, and codegen are completely unaware of comprehensions.

### Desugar rules (verified equivalences)

Right-to-left fold over qualifiers, with the head as the innermost body:

| Form | Desugars to |
|------|-------------|
| `[e \| x <- xs]` | `map(\x. e, xs)` |
| `[e \| x <- xs, g]` (g is a guard) | `map(\x. e, filter(\x. g, xs))` |
| `[e \| x <- xs, y <- ys]` | `flatMap(\x. map(\y. e, ys), xs)` |
| `[e \| x <- xs, y <- ys, g]` | `flatMap(\x. map(\y. e, filter(\y. g, ys)), xs)` |

A guard attaches to the **nearest enclosing generator to its left** (standard
comprehension scoping), so it can see all binders introduced before it.

**Verified end-to-end** (`ailang run`, v0.24.0):

```ail
import std/list (map, filter, flatMap)
let xs = [1,2,3,4,5,6];
let r1 = map(\x. x * 2, filter(\x. x % 2 == 0, xs));   -- => [4, 8, 12]
let r2 = flatMap(\x. map(\y. (x, y), [10, 20]), [1, 2]); -- => 4 tuples
```
Output: `[4, 8, 12]` and four tuple values — confirming the desugar targets
exist and produce the intended results.

### Implementation Plan

**Phase 1: Parser + AST** (~6 hours)
- [ ] Add `ast.ListComp` and `ast.Qualifier` (`Generator{Pat, Src}` | `Guard{Expr}`).
- [ ] Branch `parseListLiteral` on `PIPE` after the head expr → `parseComprehension`.
- [ ] Generator binder: reuse the `let`-pattern parser for the LHS of `<-`.
- [ ] Suppress the misleading "Add or correct the `,` token" suggestion for this path.
- [ ] Parser unit tests: single generator, multi-generator, guards, nested,
      pattern binders, and the **regression fixtures** below.

**Phase 2: Elaboration desugar** (~6 hours)
- [ ] `desugarListComp` lowering pass (runs before type checking).
- [ ] Resolve `map`/`filter`/`flatMap` against `std/list` the same way an explicit
      import would (confirm resolution when the user has not imported them — either
      auto-resolve the prelude names or require import; decide in Design Freeze).
- [ ] Golden tests asserting elaborated Core matches the hand-written equivalent.

**Phase 3: Type/effect verification, examples, docs** (~4 hours)
- [ ] Confirm body effects propagate (e.g. `[println(x) | x <- xs]` carries `! {IO}`).
- [ ] `examples/list_comprehension.ail` (per coding-standards: every feature needs an example).
- [ ] Update `ailang prompt` / `prompts/` teaching material and CHANGELOG.
- [ ] `make verify-examples`, full parser suite, `make test`.

### Files to Modify/Create

**New files:**
- `examples/list_comprehension.ail` — feature example (~30 LOC)
- `internal/elaborate/listcomp.go` — desugar pass (~120 LOC)

**Modified files:**
- `internal/ast/ast.go` (or relevant ast file) — `ListComp` + `Qualifier` nodes (~40 LOC)
- `internal/parser/parser_literals.go` — `parseComprehension`, branch in `parseListLiteral` (~120 LOC)
- `internal/parser/parser_error.go` — drop misleading `,` suggestion on the `|`-in-`[` path (~10 LOC)
- `internal/elaborate/elaborate.go` — wire in `desugarListComp` (~15 LOC)
- `CHANGELOG.md` / `changelogs/v0.10-current.md` — changelog entry
- `prompts/` + `ailang prompt` content — teach the new idiom (~30 LOC)

## Conflict Surface

This change touches `internal/parser/` and `internal/ast/` and `internal/elaborate/`,
so a conflict surface analysis is mandatory.

### Syntactic positions touched

- Extends the list-literal production: after the head expression inside `[`,
  parsed by `parseListLiteral` (registered `internal/parser/parser.go:81`), add
  a branch on `PIPE`. Today that position accepts only `,` (more elements) or `]`.
- Adds the binder arrow `<-` inside `[...]`. `<-` exists in the lexer already.
- Adds `*ast.ListComp` to the expression AST.

### What else lives here

`|` (PIPE) is a heavily reused token. Enumerating every existing claimer
(verified by grep over `internal/parser/`):

| Position | Existing valid form | Shape | File |
|----------|---------------------|-------|------|
| Inside effect-row braces | row variable | `! { IO \| r }` | parser_effect.go:197 |
| After `{ ident` | record update | `{ r \| field: v }` | parser_literals.go:317, parser_expr.go:353 |
| In a type declaration | sum-type variant separator | `type T = A \| B` | parser_type_decl.go:87 |
| **Inside `[ ... ]`** | **(none today)** | **— this PR adds `[ e \| quals ]`** | new |

Critical finding: **no existing construct uses `|` inside square brackets.**
The three existing `|` claimers all live in *different* delimiter contexts
(`{...}` for rows and record-update, type-decl RHS), none of which can appear
where the list-literal parser is active. The conflict surface inside `[...]` is
empty, which is why the change is a clean extension rather than a disambiguation
problem.

Match guards are *not* a conflict: AILANG guards use `if`, not `|` (verified:
`match n { x if x > 0 => "pos", _ => "other" }` compiles cleanly).

### Disambiguation strategy

Single-token lookahead, fully sufficient:

```
parseListLiteral:
  consume "["
  if peek == "]" : empty list, done
  head = parseExpr()
  switch peek:
    case "|"  -> parseComprehension(head)   // NEW
    case ","  -> parseListLiteralRest(head)  // existing
    case "]"  -> singleton list              // existing
    default   -> error
```

The decision is made by one token (`|` vs `,` vs `]`) immediately after the head
expression. Because no other construct places `|` after an expression inside
`[`, there is no ambiguity at depth 1. The binder arrow `<-` is likewise
unambiguous inside a comprehension qualifier.

### Programs that MUST still work (regression fixtures)

These existing syntactic neighbours must be re-verified post-change:

1. **List literals**: `let xs = [1, 2, 3]` and `let ys = []` and nested
   `[[1,2],[3,4]]` (the exact production being extended).
2. **Effect-row variables**: any function with `! {IO | r}` in its signature
   (search `std/` and `examples/` for polymorphic effect rows) — confirms `|`
   in `{...}` is untouched.
3. **Record update**: `{ rec | field: value }` (parser_literals.go path) — any
   `examples/` file exercising record update.
4. **Sum-type declarations**: `type Color = Red | Green | Blue`
   (parser_type_decl.go) — `examples/` with ADT declarations.
5. **Match with guards**: `match x { n if n > 0 => ... }` — confirms guards
   (which use `if`, not `|`) are unaffected.

All five must be added as explicit regression-surface tests; the parser suite
already exercises 1–5 but the comprehension PR must assert they still pass with
the new branch present.

### What deliberately changes

- The parse error for `[ expr | ... ]` changes from
  `PAR_UNEXPECTED_TOKEN: expected ,` (with the misleading "Add or correct the `,`
  token" suggestion) to **successful parse**. This is the intended behaviour
  change; no previously-*valid* program produced that error, so it is not a
  regression.
- Nothing else changes. Any other previously-valid program that breaks is a bug,
  not an intended change.

## Examples

### Example 1: Map + filter

**Before (works today, verbose):**
```ail
import std/list (map, filter)
let evens_doubled = map(\x. x * 2, filter(\x. x % 2 == 0, xs));
```

**After (this feature):**
```ail
let evens_doubled = [x * 2 | x <- xs, x % 2 == 0];
```

### Example 2: Nested generators (cartesian product)

**Before:**
```ail
import std/list (map, flatMap)
let pairs = flatMap(\x. map(\y. (x, y), ys), xs);
```

**After:**
```ail
let pairs = [(x, y) | x <- xs, y <- ys];
```

### Example 3: Pattern binder + guard (depends on Design Freeze)

**After:**
```ail
-- keep first components of pairs whose second is positive
let firsts = [a | (a, b) <- pairs, b > 0];
```

## Success Criteria

- [ ] `[x*2 | x <- [1,2,3]]` parses, type-checks, runs → `[2,4,6]` (acceptance test)
- [ ] `[(x,y) | x <- [1,2], y <- [3,4]]` runs → 4 tuples in cartesian order (acceptance test)
- [ ] `[x | x <- xs, x > 0]` desugars to `filter` and matches hand-written output (golden test)
- [ ] Effectful body `[println(x) | x <- xs]` carries `! {IO}` in its inferred type
- [ ] Misleading "Add or correct the `,` token" suggestion no longer fires for `[e | ...]`
- [ ] All 5 conflict-surface regression fixtures pass
- [ ] All tests passing (`make test`, parser suite)
- [ ] Documentation updated (CHANGELOG, `ailang prompt`, prompts/)
- [ ] Example added (`examples/list_comprehension.ail`)

## Testing Strategy

**Unit tests (parser):**
- Single generator, multi-generator, guard-only, generator+guard, nested.
- Identifier binder and (if in scope) pattern binder.
- Negative: `[x |]` (no qualifiers), `[x | x]` (generator without `<-`), trailing comma.

**Integration tests (elaborate + run):**
- Each desugar-rule row produces output identical to its hand-written equivalent.
- Effect propagation through comprehension bodies.

**Regression-surface tests (REQUIRED — Conflict Surface was filled in):**
- The 5 fixtures above (list literals, effect rows, record update, sum types, match guards).

## Deferred Decisions

- **Whether `map`/`filter`/`flatMap` must be imported by the user or are
  auto-resolved by the desugar** — implementer may choose, but must be consistent
  with how AILANG resolves other implicitly-used prelude functions (check
  `m-prelude-option-result` direction).
- **Tuple `show`** — `show` over tuples currently renders `<*eval.TupleValue>`
  (observed in verification); orthogonal to this feature, agent may leave as-is.

## Non-Goals

- **Range sugar `[1..n]`** — separate verified gap (`..` does not parse, `range`
  not exported by `std/list`). Deserves its own doc; bundling doubles parser surface.
- **`concatMap`** — not exported by `std/list` (IMP010); `flatMap` is the desugar
  target, so no stdlib change is required by this feature.
- **Set/dict comprehensions** — AILANG has no set/dict literal comprehension
  precedent; out of scope.
- **Parallel/zip comprehensions** (`[x+y | x <- xs | y <- ys]`) — niche; defer.

## Timeline

**Week 1** (~16 hours):
- Phase 1: parser + AST (~6h)
- Phase 2: elaboration desugar (~6h)
- Phase 3: type/effect verification, examples, docs (~4h)

**Total: ~16 hours (≈3 working days with review/iteration buffer)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Pattern binders interact badly with `let`-pattern parser | Med | Reuse the existing pattern parser rather than forking; coordinate with `m-let-pattern-destructuring` |
| Guard scoping bug (guard can't see earlier binders) | Med | Right-to-left fold places guards inside the enclosing generator's lambda; covered by golden tests |
| `|` lookahead collides with a position not enumerated | High | Conflict Surface confirms no `|` claimer inside `[...]`; 5 regression fixtures lock it in |
| Effects silently dropped by desugar | High | Desugar to real `map`/`flatMap` (which already thread effects); assert with effect-propagation test |

## Related Documents

**Planned (check for overlap):**
- `design_docs/planned/v0_24_0/m-let-pattern-destructuring.md` — share the pattern
  parser for generator binders.
- `design_docs/planned/v0_24_0/m-prelude-option-result.md` — informs prelude/import
  resolution for the desugar targets.
- `design_docs/planned/v0_24_0/m-prompt-split-list-operations.md` and
  `m-prompt-string-concat-plusplus.md` — sibling eval-gap work; this doc *closes*
  a gap rather than teaching around it.

**Implemented (may inform design):**
- `design_docs/implemented/v0_6_2/dx-17-phase2-sprint-plan.md` — `TList`→`TApp`
  normalization; confirms list type representation the desugar produces.

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- [M-PARSER-REFINEMENT-LOOKAHEAD](../../../changelogs/v0.10-current.md) — case study motivating the Conflict Surface analysis
- All language claims in this doc verified against `ailang` v0.24.0, commit d4ba009

## Future Work

- Range sugar `[1..n]` / `[1..n by k]` (its own doc) + `range`/`enumFromTo` in `std/list`.
- Comprehension `let` bindings (`[y | x <- xs, let y = f(x)]`).
- `concatMap` export in `std/list` if other call sites want it.

---

**Document created**: 2026-06-05
**Last updated**: 2026-06-05
