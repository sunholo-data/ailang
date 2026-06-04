# Let-Binding Pattern Destructuring (M-LET-PATTERN)

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 (Medium-High) — high-frequency LLM idiom, blocks ergonomic tuple/record use
**Estimated**: 1.5 days (≈10 hours)
**Dependencies**: None (reuses existing pattern infrastructure)
**Created**: 2026-06-04
**Reporter**: design-doc-creator (eval-gap audit)

## Problem Statement

`let` bindings only accept a **single identifier** as the binder. Any program that
tries to destructure a tuple, record, or constructor in a `let` fails to **parse** —
even though the exact same pattern works fine in a `match` arm. The pattern-matching
machinery already exists; it is simply not wired into `let`.

**Current State (all verified with `ailang check` on dev `b32dc9a`, 2026-06-04):**

| Construct | Result | Evidence |
|-----------|--------|----------|
| `let x = e in ...` | ✅ parses | baseline |
| `let _ = e in ...` | ✅ parses | wildcard goes through IDENT path |
| `let x: int = e in ...` | ✅ parses | typed binder |
| `let (a, b) = (1, 2) in ...` | ❌ `PAR_UNEXPECTED_TOKEN ...: expected next token to be IDENT, got ( instead` | tuple destructure |
| `let {name, age} = r in ...` | ❌ parse error | record destructure |
| `let Box(x) = Box(5) in ...` | ❌ parse error | constructor destructure |
| `match (1,2) { (a,b) => ... }` | ✅ parses | **same pattern works in `match`** |
| `match ((1,2),3) { ((a,b),c) => ... }` | ✅ parses | nested patterns already supported |

The failure point is exact and singular: `parseLetExpression` calls
`p.expectPeek(lexer.IDENT)` immediately after the `let` keyword
(`internal/parser/parser_expr.go:97`), so a binder starting with `(`, `{`, or a
constructor name can never be reached. `ast.Let` itself only carries a
`Name string` field (`internal/ast/ast_expr.go:240`).

**Impact:**

- **Who**: AI code-synthesis agents (the primary AILANG audience) and every human
  reader. Tuple/record destructuring in `let` is one of the most reflexive idioms a
  model trained on ML-family / Rust / Swift / JS reaches for. When it fails, the model
  must rewrite to a single-arm `match` — a non-obvious workaround that costs an
  iteration round-trip and tokens.
- **How significant**: Not a blocker (the `match` workaround is sound and runs), but a
  high-frequency papercut that degrades first-attempt eval pass-rates and makes
  idiomatic data-shuffling code (returning/unpacking pairs, splitting records) verbose.
- **Why now**: This is a *language* gap with a *trivial* fix — the pattern parser,
  AST pattern types, exhaustiveness checker, and binding logic all already exist for
  `match`. The cost/benefit is unusually favourable.

**The systemic framing (not just tuples):** the same `expectPeek(IDENT)` gate blocks
tuple **and** record **and** constructor **and** list (`::`) patterns. A fix that
special-cased only tuples would leave three identical papercuts and invite exactly the
incremental-special-casing anti-pattern the project warns against. The right fix routes
the binder through the *one* existing `parsePattern` entry point, covering all four at
once.

## Goals

**Primary Goal:** Allow any pattern already valid in a `match` arm to appear as the
binder of a `let` (both the `let … in …` expression form and the block
`{ let … = …; … }` form), by desugaring to a single-arm `match` — with zero new pattern
syntax and zero change to `match` semantics.

**Success Metrics:**

- `let (a, b) = e in …`, `let {x, y} = e in …`, `let Ctor(a) = e in …`, and
  `let h :: t = e in …` all `ailang check`-pass and evaluate to the same result as the
  equivalent single-arm `match`.
- Single-identifier `let x = e` and typed `let x: T = e` are **byte-for-byte unchanged**
  in parse output (same `*ast.Let` node, same elaboration path) — verified by snapshot.
- ≥ 3 new `examples/*.ail` demonstrate the feature and pass `make verify-examples`.
- No regression in the existing parser/elaborate/eval test suites.
- Exhaustiveness: a refutable pattern in `let` (e.g. `let Some(x) = …`) produces a
  clear, structured diagnostic (warning or error — see Design Freeze), not a silent
  partial match or a panic.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Desugar `let PAT = v in b` → `match v { PAT => b }` rather than teaching the binder logic about patterns | Reuses the entire match pipeline (type, exhaustiveness, binding, eval); avoids a second pattern-binding implementation that could drift | agent | design | low |
| Desugar happens in the **parser** (emit `*ast.Match`) vs. a new `*ast.LetPattern` AST node lowered in elaboration | Parser desugar = smallest surface, no new AST/elaborate/eval node, no codegen changes. New node = better error spans but touches every AST visitor | human | design | med |
| Refutable pattern in `let` → **warning + runtime error on no-match** (not a hard compile error) | Determines strictness; affects A11 (structured failure). A hard error is safest but rejects `let Some(x)=knownSome` ergonomics | human | design | med |
| Single-ident / wildcard / typed-ident keep the **existing** `*ast.Let` fast path | Preserves `letrec` recursion semantics and the type-annotation path; guarantees zero churn for the 99% case | agent | design | low |
| `letrec` stays identifier-only (no pattern binders) | Recursion needs a name in scope in the value; a pattern has no single name to bind recursively | compiler | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Desugar to single-arm `match` (reuse existing pipeline)
- [ ] **AST strategy**: parser-emits-`Match` vs. new `*ast.LetPattern` node — *recommend parser desugar* (see Solution Design); needs human sign-off because it trades error-message quality for implementation size
- [ ] **Refutability policy**: warn-and-runtime-error vs. hard-compile-error for non-exhaustive `let` patterns
- [ ] Whether the `let h :: t = …` cons-sugar form is in-scope for M1 or deferred (it falls out for free from `parsePattern`, but adds a regression fixture)

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact wording of the refutable-pattern diagnostic — agent may choose (must be structured, name the missing constructor)
- Whether to emit the desugared `match` with a synthesized wildcard arm (`_ => <error>`) or rely on the eval-time non-exhaustive-match path — agent may choose, must produce a typed error not a panic
- Test fixture file names and organization — agent may choose
- Internal naming of the new parser helper (`parseLetBinder`, `letBinderIsPattern`, …) — agent may choose

## Solution Design

### Overview

When the parser is positioned just after the `let` keyword, it currently demands an
`IDENT`. We extend it to **peek** at the binder shape:

- A binder that is a plain lowercase identifier (optionally `: Type`), `_`, or any
  existing form → **unchanged**: build `*ast.Let` exactly as today.
- A binder that begins a pattern (`(`, `{`, or `IDENT(` constructor application, or an
  identifier followed by `::`) → parse it with the **existing** `parsePattern`
  (`internal/parser/parser_pattern.go:10`) and emit
  `match <value> { <pattern> => <body> }` as an `*ast.Match` node.

Because `parsePattern` already handles tuple, record, constructor, nested, literal, and
`::`-cons patterns, the single re-route covers every destructuring form in one change.
The desugared `*ast.Match` then flows through the unmodified type-checker, exhaustiveness
checker, binder, and evaluator — so there is no new semantics to verify, only a new way
to *spell* an existing one.

### Architecture

**Components:**

1. **Binder disambiguator** (`parser_expr.go`): a small `letBinderIsPattern()` lookahead
   that decides "variable binder" vs "pattern binder" using ≤ 2 tokens (table below).
2. **Pattern-let desugar** (`parser_expr.go`): when the binder is a pattern, parse
   `parsePattern()`, consume `=`, parse the value, consume `in`/`;`, parse the body, and
   construct `&ast.Match{Expr: value, Cases: []*ast.Case{{Pattern: pat, Body: body}}}`.
   Reuses the same value/body parsing the existing `let` path uses.
3. **Block-let parity** (`parseBlockOrExpression`, `parser_expr.go:327`): the sequenced
   `{ let PAT = e; rest }` form routes through the same disambiguator so both surfaces
   behave identically.
4. **(Only if "new AST node" is chosen)** `*ast.LetPattern` + an elaboration case in
   `internal/elaborate/expressions.go` that lowers to `core` match. *Not recommended for
   M1.*

### Disambiguation (the ≤2-token lookahead)

Positioned at `let`, peeking the next token (`peek`) and the one after (`peek2`):

| `peek` | `peek2` | Interpretation | Path |
|--------|---------|----------------|------|
| `IDENT` (lowercase) | `=` or `:` | variable binder | **existing** `*ast.Let` |
| `_` (wildcard) | `=` | wildcard binder | **existing** `*ast.Let` |
| `IDENT` | `(` | constructor pattern (`Some(x)`) | desugar → `match` |
| `IDENT` | `::` | cons pattern (`h :: t`) | desugar → `match` |
| `(` | — | tuple/unit pattern | desugar → `match` |
| `{` | — | record pattern | desugar → `match` |

This is sound because variable binders and the four pattern openers are disjoint at this
depth — see **Conflict Surface** for the full enumeration and the one deliberate
behavior change (`let f(x) = …`).

### Implementation Plan

**Phase 1: Expression-form `let … in …`** (~4 hours)
- [ ] Add `letBinderIsPattern()` lookahead helper in `parser_expr.go`
- [ ] In `parseLetExpression`, branch: pattern → `parsePattern()` + build single-arm `*ast.Match`; else existing path verbatim
- [ ] Unit tests: tuple/record/constructor/cons/nested patterns parse to the expected `*ast.Match` AST; single-ident & typed-ident produce the **unchanged** `*ast.Let` (snapshot)

**Phase 2: Block-form `{ let PAT = e; … }`** (~2 hours)
- [ ] Route the sequenced-let binder in `parseBlockOrExpression` through the same disambiguator/desugar
- [ ] Unit tests for block form parity

**Phase 3: Semantics, diagnostics, examples** (~4 hours)
- [ ] Refutable-pattern handling per Design Freeze decision (synth wildcard arm → typed error, or compile warning) — verify no panic on `let Some(x) = None`
- [ ] Confirm type inference flows (the binder pattern's types unify through the existing match path) with a polymorphic case: `let (a, b) = p in …` infers `a`, `b` independently
- [ ] 3+ `examples/runnable/*.ail` (tuple swap, record unpack, constructor unpack) + goldens; `make verify-examples`
- [ ] CHANGELOG + teaching-prompt note (deferred to next deliberate prompt-version bump — active prompt is hash-locked for eval reproducibility)

### Pipeline Pass Coverage

This feature adds **no new pipeline pass**; it desugars at parse time into an existing
node. The checklist below confirms the desugared node reaches every consumer the
hand-written `match` already reaches (no bypassed path):

- [ ] `prog.Decls` — top-level `let` in a decl body lowers identically
- [ ] Function-body `let` (both `… in …` and block `;` forms)
- [ ] `let` inside `match`-arm bodies (nested) still parses
- [ ] No contract-expression interaction (this change is purely binder-position; `requires`/`ensures` exprs unaffected)

### Files to Modify/Create

**Modified files:**
- `internal/parser/parser_expr.go` (+~60/−2 LOC) — `letBinderIsPattern()` + pattern branch in `parseLetExpression`; same in the block-let path of `parseBlockOrExpression`
- `internal/parser/parser_expr_test.go` (+~120 LOC) — disambiguation + desugar AST tests
- `CHANGELOG.md` / `changelogs/v0.10-current.md` (+~15 LOC)

**New files:**
- `examples/runnable/let_pattern_tuple.ail` (~15 LOC) + golden
- `examples/runnable/let_pattern_record.ail` (~15 LOC) + golden
- `examples/runnable/let_pattern_constructor.ail` (~18 LOC) + golden

**Untouched (the point of the parser-desugar approach):** `internal/ast/`,
`internal/elaborate/`, `internal/types/`, `internal/eval/`, `internal/vm/`,
`internal/codegen/`. *(If the "new AST node" strategy is chosen instead, add
`internal/ast/ast_expr.go` +1 node and `internal/elaborate/expressions.go` +~40 LOC; the
Conflict Surface below is unchanged.)*

## Examples

### Example 1: Tuple destructuring (the headline case)

**Before (only the `match` workaround parses):**
```ailang
export func swap(p: (int, int)) -> (int, int) =
  match p { (a, b) => (b, a) }
```

**After (both forms valid; identical runtime behavior):**
```ailang
export func swap(p: (int, int)) -> (int, int) =
  let (a, b) = p in (b, a)
```

### Example 2: Record unpack

**Before:**
```ailang
let name = match user { {name: n, age: _} => n }
```

**After:**
```ailang
let {name, age} = user in
greet(name, age)
```

### Example 3: Constructor unpack (irrefutable single-ctor types)

```ailang
type Vec2 = Vec2(float, float)

export func len2(v: Vec2) -> float =
  let Vec2(x, y) = v in
  x *. x +. y *. y
```

### Example 4: Falls out for free — cons sugar (already a pattern via S-CONS)

```ailang
let h :: t = xs in   -- desugars to match xs { ::(h, t) => ... }
process(h, t)
```

## Conflict Surface

> Required because this change touches `internal/parser/` (and, under the alternative
> strategy, `internal/ast/` + `internal/elaborate/`).

### Syntactic positions touched

- **The binder position immediately after the `let` keyword**, parsed by
  `parseLetExpression` at `internal/parser/parser_expr.go:97` (`expectPeek(lexer.IDENT)`).
  We replace the unconditional `IDENT` demand with a disambiguating branch.
- **The same binder position in the sequenced/block form**, parsed inside
  `parseBlockOrExpression` (`parser_expr.go:327`) when it encounters a `let` keyword in a
  `{ …; … }` block.
- `parseLetRecExpression` (`parser_expr.go:136`) is **explicitly not touched** —
  `letrec` stays identifier-only.

### What else lives here

| Position | Existing valid form | Shape |
|----------|--------------------|-------|
| After `let` | variable binder | `IDENT` (lowercase) then `=` or `:` |
| After `let` | typed variable binder | `IDENT : <type> =` |
| After `let` | wildcard binder | `_` then `=` (`_` lexes as `IDENT`) |
| After `let` | **tuple pattern (this change)** | `( … )` then `=` |
| After `let` | **record pattern (this change)** | `{ … }` then `=` |
| After `let` | **constructor pattern (this change)** | `IDENT ( … )` then `=` |
| After `let` | **cons pattern (this change)** | `IDENT :: …` then `=` |
| After `let` | *(does NOT exist today)* function-definition sugar `let f(x) = …` | — verified: `ailang check` on `let f(x)=x+1 in …` → `PAR_UNEXPECTED_TOKEN`. There is **no** `let f(x)=` form, so the `IDENT(` shape is unclaimed and free for constructor patterns |

The lambda form `let f = \x. …` (the actual way to bind a local function) is a plain
**variable** binder (`IDENT` then `=`) and is therefore on the **unchanged** path.

### Disambiguation strategy

≤ 2-token lookahead from `let` (see table in Solution Design). The four pattern openers
(`(`, `{`, `IDENT(`, `IDENT::`) are disjoint from the variable-binder shapes
(`IDENT =`, `IDENT :`, `_ =`) at this depth:

- `(` and `{` cannot begin a variable name → unambiguously patterns.
- `IDENT` is a variable binder **unless** immediately followed by `(` or `::`. Since
  `let f(x) = …` does not exist as a valid program today, `IDENT(` never legitimately
  means "function definition"; routing it to a constructor pattern claims previously-
  invalid syntax (a strict superset — see *What deliberately changes*).
- `parsePattern`/`parseBasePattern` already resolves constructor-vs-variable by the
  trailing `(` (`parser_pattern.go:55–63`); constructor-ness (is `Foo` a real
  constructor?) is settled later in elaboration, exactly as for `match` today. We add no
  new casing rule.

### Programs that MUST still work (→ regression fixtures in M1)

1. `examples/` files using single-ident `let … in …` (e.g. any `let x = … in …`) — must
   still produce an `*ast.Let`, not a `*ast.Match`. **Snapshot the AST.**
2. Typed binder `let x: int = 5 in x` — verified-passing today; must remain `*ast.Let`.
3. Local-function lambda `let f = \x. x + 1 in f(2)` — verified-passing today; variable
   binder path.
4. Block form `export func main() -> () ! {} { let x = 5; () }` — verified-passing today.
5. An existing `match` with tuple/record/nested arms (e.g. the
   `m-codegen-tuple-pattern` fixtures) — must be byte-for-byte unaffected (we touch
   `let`, not `match`/`parsePattern`).

Each becomes a pinned parse/AST snapshot test. A diff here is a regression, not churn.

### What deliberately changes

- `let f(x) = …` (lowercase ident followed by `(`) previously produced a **parse error**
  (`PAR_UNEXPECTED_TOKEN`). After this change it parses as a **constructor pattern** and,
  if `f` is not a known constructor, fails in **elaboration** with a constructor-not-found
  error instead. This is a strictly-better diagnostic for a program that was already
  invalid; no previously-*valid* program changes meaning. Documented as the single
  intentional incompatibility.

## Testing Strategy

**Unit tests (parser):**
- Each pattern form (`tuple`, `record`, `constructor`, `cons`, `nested`) after `let`
  desugars to the expected single-arm `*ast.Match` (AST assertion).
- Single-ident, typed-ident, wildcard, and lambda binders produce the **unchanged**
  `*ast.Let` (snapshot — guards the fast path).
- Block-form parity (`{ let (a,b) = …; … }`).
- Negative: malformed binders give a sensible `PAR_*` error (e.g. `let (a, = …`).

**Integration tests (elaborate + eval):**
- `let (a,b) = (1,2) in a + b` evaluates to `3`; record and constructor analogues.
- Polymorphic inference: `let (a, b) = p in …` types `a`/`b` independently for a generic `p`.
- Refutable pattern: `let Some(x) = None in x` yields the chosen structured error (no panic).

**Regression-surface tests (REQUIRED — from Conflict Surface):**
- One pinned snapshot per "Programs that MUST still work" entry (5 total).

**Manual testing:**
- `ailang run` each of the 3 new examples; confirm output matches the `match`-rewritten equivalent.

## Non-Goals

- **`letrec` pattern binders** — recursion needs a single name; out of scope.
- **New pattern *syntax*** — we add no pattern forms; only `let` reuse of existing ones.
- **`@`/as-patterns** (`let p @ (a, b) = …`) — not currently in `match` either; deferred.
- **Refactoring `match` or `parsePattern`** — deliberately untouched to keep the conflict surface minimal.
- **Teaching-prompt rollout** — the prompt note lands in the next deliberate prompt-version bump (active prompt is hash-locked for eval reproducibility), not in this PR.

## Timeline

**Day 1** (~6 hours):
- Phase 1: expression-form desugar + disambiguator + tests (4h)
- Phase 2: block-form parity + tests (2h)

**Day 2** (~4 hours):
- Phase 3: refutability handling, inference checks, 3 examples + goldens (3h)
- CHANGELOG, docs, final `make ci` (1h)

**Total: ~10 hours (1.5 days)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fast-path regression — single-ident `let` accidentally routed to `match` | High | Snapshot tests assert `*ast.Let` for all non-pattern binders; disambiguator defaults to the existing path |
| Refutable `let` panics at runtime instead of erroring | Medium | Synthesize a wildcard arm producing a typed non-exhaustive error; explicit test `let Some(x)=None` |
| Worse error spans than a dedicated AST node | Low | Carry the pattern's `Pos` into the synthesized `*ast.Match`; revisit the `*ast.LetPattern` strategy only if spans prove poor |
| Hidden third claimant of the `IDENT(` position | Low | Verified `let f(x)=…` does not parse today; enumerated in Conflict Surface |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure parse-time desugar; no nondeterminism |
| A2: Replayability | 0 | No runtime/state impact |
| A3: Effect Legibility | 0 | Effects unchanged; desugars to existing match |
| A4: Explicit Authority | 0 | No capability/effect-row change |
| A5: Bounded Verification | +1 | Reuses match exhaustiveness checking for `let`; refutability becomes locally checkable |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Removes a high-frequency first-attempt parse failure for AI synthesis; fewer iteration round-trips/tokens |
| A8: Minimal Syntax | +1 | Reuses existing pattern syntax in a new position; no new tokens or forms — strictly less syntactic noise for the same intent |
| A9: Cost Visibility | 0 | No resource-cost change |
| A10: Composability | +1 | `let` now composes with the full pattern language already used by `match` |
| A11: Structured Failure | +1 | Refutable binders yield a structured non-exhaustive error instead of forcing manual `match` boilerplate (contingent on the warn-vs-error decision being structured) |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +5** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects — desugars to existing `match`
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Optimizes for machine analysis (fewer parse failures), not human-only convenience

## Related Documents

- [M-CODEGEN-TUPLE: Tuple Pattern Matching](../../implemented/v0_5_10/m-codegen-tuple-pattern.md) — the tuple-pattern machinery this feature reuses
- [M-VALIDATOR-PATTERN-BINDINGS](../../implemented/v0_10_0/m-validator-pattern-bindings.md) — pattern-binding validation prior art
- [Surface Sugar Pack (S-CONS, S-CALL0)](../../archive/v0_4_2_surface-sugar-pack.md) — the `::` cons-pattern sugar that falls out for free here
- [M-MATCH-ADT-XCHECK](../../implemented/v0_18_10/m-match-adt-xcheck.md) — match exhaustiveness checking the desugar inherits

## References

- **Motivation**: eval-gap audit, 2026-06-04 — all claims verified live with `ailang check` on dev `b32dc9a`
- **Verification transcript**: `let (a,b)=(1,2)` → `PAR_UNEXPECTED_TOKEN` (fail); `match (1,2){(a,b)=>…}` → no errors (pass); `let f(x)=x+1` → `PAR_UNEXPECTED_TOKEN` (confirms no function-sugar conflict)
- **Conflict-surface case study**: [M-PARSER-REFINEMENT-LOOKAHEAD](../../../changelogs/v0.10-current.md) — why the lookahead/conflict enumeration is mandatory
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- `@`/as-patterns in both `let` and `match` (`let p @ (a, b) = …`).
- Pattern binders in `for`/comprehension headers if/when those land.
- A teaching-prompt entry once the next deliberate prompt-version bump occurs.
