# M-SYNTAX-ERGONOMICS-V1: Everyday Functional Syntax Ergonomics

**Status**: Implemented (2026-11-24)
**Target**: v0.61.0 (pre-v1.0.0 polish)
**Priority**: P1 (High — directly reduces first-attempt AI parse errors & human friction)
**Estimated**: 3–4 days across 3 independent milestones (E1, E2, E3)
**Dependencies**: None (all three are purely surface-syntax / elaboration / formatter; zero runtime or bytecode ISA changes)
**Author**: Antigravity (attended session with Mark)
**Created**: 2026-11-24

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | All three features desugar deterministically at parse/elaboration time; `Parse(fmt(x)) ≡ Parse(x)` preserved |
| A2: Replayability | 0 | No runtime trace or effect changes |
| A3: Effect Legibility | 0 | Effect rows and capability checks unchanged |
| A4: Explicit Authority | 0 | No ambient authority added; qualified types respect existing module-alias imports |
| A5: Bounded Verification | +1 | Desugaring `let (a, b) = rhs` to single-arm `match` reuses existing Z3 tuple-match encoding automatically |
| A6: Safe Concurrency | 0 | Purely sequential surface syntax |
| A7: Machines First | +1 | Directly eliminates top measured AI parse errors (`let (a, b) =`, block-ending tuple/record `;`, `S.Set[int]`) |
| A8: Minimal Syntax | 0 | Reuses existing tuple-pattern, dot-qualification, and newline-separator rules rather than inventing new keywords |
| A9: Cost Visibility | 0 | Zero runtime overhead; compiles to identical Core AST |
| A10: Composability | +1 | Makes multi-value tuple returns (`mapAccumL`, state transitions) and aliased modules (`import std/set as S`) compose without boilerplate |
| A11: Structured Failure | +1 | Replaces opaque `expected IDENT, got (` / `expected }, got (` errors with valid syntax (and keeps strict errors for ambiguous same-line forms) |
| A12: System Boundary | 0 | No host/FFI boundary changes |

**Net Score: +5** (≥ +2, no −1 on A1/A3/A4/A7).

---

## Problem Statement & Evidence

During the 2026-11-24 doc-sample burn-down (~275 AILANG blocks) and 54-benchmark `gemini-2-5-flash` eval (`core` + `stretch` + `frontier`), three everyday functional-programming syntax gaps repeatedly tripped both humans and LLMs:

1. **E1 — No tuple destructuring in `let` (`let (a, b) = expr`)**:
   - Returning a tuple (e.g. `mapAccumL`, `(nextState, out)`, `(head, tail)`) forces wrapping the entire remainder of the function inside `match expr { (a, b) => ... }`.
   - Caused multiple parse failures in our eval (`cp_60197`, `type_unify`) before we added `PAR_LET_TUPLE`.
2. **E2 — Module-qualified type names do not parse (`S.Set[int]`, `M.Map[string, int]`)**:
   - Documented as an open parser papercut in `docs/LIMITATIONS.md:46`.
   - `import std/set as S` works for value calls (`S.empty()`, `S.insert(...)`) but fails with `PAR_UNEXPECTED_TOKEN … got . instead` in type annotations (`func f(s: S.Set[int])`), forcing redundant double imports (`import std/set as S` + `import std/set (Set)`).
3. **E3 — Block-ending `(a, b)` or `{ r | x: 1 }` after a `let` statement requires a magic `;`**:
   - Inside a `{ ... }` block, writing:
     ```ailang
     let a = 1
     let b = 2
     (a, b)
     ```
     glues `(a, b)` onto `2` as a function call `2(a, b)` unless you remember `let b = 2;`. Same for `{ r | a: x }` on the line after `let x = 1`.
   - Hit ~15 times manually during the doc-sample burn-down, and explicitly called out as a trap in `prompts/v0.16.18.md:253` ("ONE place `;` is required").

---

## Prior Art & Relevant History

Checked `design_docs/implemented/` and `design_docs/planned/` before designing:

| Prior Doc | Relevance | Takeaway |
|-----------|-----------|----------|
| `design_docs/implemented/v0_30_0/m-syntax-ai-forgiving.md` (R2) | Introduced newline statement separation in `{ }` blocks via `peekStartsNewlineBlockStatement()` (`parser_func.go:401`) | Deliberately excluded `(` and `{` from `peekStartsBlockStatement()` so multi-line expressions like `f(x)\n(y)` or `x +\n(y)` don't split. **Crucial insight:** after a `let` statement with `Body == nil` (no `in`), the previous statement is already a complete binding, and a *new-line* `(a, b)` or `{ ... }` is a tuple/record/block value expression — NOT a curried call on the RHS literal/binding! |
| `design_docs/implemented/v0_5_5/m-dx24-typed-function-bodies.md:289` | Listed `let (a, b) = pair in a + b` as "Test 9: Let binding with tuple destructuring" | Mentioned in an early test spec, never implemented in `internal/parser/`. |
| `design_docs/implemented/v0_5_10/m-codegen-tuple-pattern.md` | Tuple pattern matching in `match` arms + Go codegen | Confirms `core.Match` with `core.TuplePattern` already works across evaluator, bytecode VM, SMT/Z3, and Go codegen! |
| `design_docs/planned/v1_0_0/m-v1-nominal-type-identity.md` & `internal/pipeline/type_name_shadow.go` | Nominal type identity (`types.NominalKey(modPath, name)`) and module-alias resolution | Module aliases (`imp.ModuleAlias → imp.Path`) are already tracked cleanly in `ast.ImportDecl` and resolved in pipeline/elaborator. |
| `docs/LIMITATIONS.md:46` | Explicitly lists `Module-qualified type names do not parse (S.Set[int])` | Confirms this is an acknowledged papercut, not a deliberate restriction. |
| `internal/format/format.go:460` (`startsWithStatementStarter`) | Formatter emits `;` before non-statement-starters to guarantee `Parse(fmt(x)) ≡ Parse(x)` | Any change to E1/E2/E3 MUST keep `internal/format` round-trip (`TestFormatterOutputAlwaysReParses`) and `TestFmtDoesNotDriftFromTeachingPrompt` green! |

---

## Proposed Design

### Milestone E1: Tuple Destructuring in `let` (`let (a, b) = expr`)

#### Surface Syntax
Supported in both block-statement form and `let ... in` expression form:
```ailang
-- Block form
func demo(pair: (int, string)) -> string {
  let (n, s) = pair
  "${s}: ${show(n)}"
}

-- Nested tuples & wildcards
let (a, (b, _)) = nested

-- Expression form
let (x, y) = divMod(10, 3) in x + y
```

**Out of scope for E1 (Non-Goal):** General refutable patterns in `let` (e.g., `let Some(x) = opt`, `let h :: t = xs`) — those are partial/refutable and belong in `match` so exhaustiveness (`TC_MATCH_002`) stays sound. E1 allows **irrefutable tuple patterns only** (identifiers, `_`, and nested tuple patterns of those).

#### AST & Desugaring Design
- **AST (`internal/ast/ast_expr.go`):** Add optional `Pattern ast.Pattern` field to `ast.Let` (or a dedicated `ast.LetPattern` node — **Decision D1** below).
  - *Recommended (Option A — `ast.LetPattern` or `Pattern` on `ast.Let`)*: A dedicated `ast.LetTuple` / `ast.LetPattern` keeps `ast.Let.Name` invariant everywhere in existing code, OR adding `Pattern ast.Pattern` on `ast.Let` (only non-nil when destructuring) avoids duplicating `let`-chain handling in `internal/format`. Let's weigh both cleanly in High-Impact Decisions.
- **Parser (`internal/parser/parser_expr.go`):**
  - In `parseLetExpression()`: when `p.peekTokenIs(lexer.LPAREN)`, parse an irrefutable tuple pattern `(p1, p2, ...)` (rejecting 0-element `()` or non-tuple/refutable constructor patterns with a clear error pointing to `match`), expect `=`, parse `Value`, and optionally `in Body`.
  - Keep `PAR_LET_TUPLE` for `letrec (a, b) = ...` (recursive tuple destructuring is not supported).
- **Elaboration (`internal/elaborate/expressions.go` & `expr_control.go`):**
  - Desugar `let pat = val in body` → `&core.Match{Scrutinee: val, Arms: []core.MatchArm{{Pattern: elabPat, Body: body}}}`!
  - In `ast.Block` (`elaborateBlock` & `blockScopeSets`), collect all bound variable names from `pat` (e.g. `a`, `b`) into scope for subsequent statements, and fold the remaining block expressions into `Arm.Body`.
  - **Zero changes** needed in `internal/types`, `internal/eval`, `internal/vm`, `internal/smt`, or `internal/gen` — they all already handle single-arm `core.Match` on `core.TuplePattern`!
- **Formatter (`internal/format/`):**
  - Print `let (a, b) = val` (and `in body` if expression form) directly from the AST so `ailang fmt` preserves the written destructuring rather than expanding it to `match`.

---

### Milestone E2: Module-Qualified Type Names (`Alias.Type`, `Alias.Type[Args]`)

#### Surface Syntax
```ailang
import std/map as M
import std/set as S

func keys(m: M.Map[string, int]) -> S.Set[string] = ...
type State = { visited: S.Set[(int, int)], scores: M.Map[string, int] }
```
Works anywhere `parseType()` is called: function params, return types, `let x: M.Map[...]` annotations, `type` aliases, ADT fields, record type fields, and nested type arguments (`Option[S.Set[int]]`).

#### AST, Parser, Resolution & Formatter Design
- **Parser (`internal/parser/parser_type.go`):**
  - In `case lexer.IDENT:` (line 22): check `if p.peekTokenIs(lexer.DOT)`.
  - Consume `.`, expect `IDENT` (`typeName`). Keep `moduleAlias = name`.
  - If followed by `[`, parse type args into `&ast.TypeApp{Module: moduleAlias, Constructor: typeName, Args: typeArgs, Pos: startPos}` (or encode as `QualifiedType` / `Module` field on `SimpleType` & `TypeApp`).
  - If not followed by `[`, emit `&ast.SimpleType{Module: moduleAlias, Name: typeName, Pos: startPos}` (never a type variable — a qualified name is always a concrete type constructor/alias).
- **Resolution (`internal/elaborate/` & `internal/pipeline/`):**
  - Use the file's `import <path> as <alias>` map (`aliases[imp.ModuleAlias] = imp.Path`, same as `importedNames` in `warn_deprecated_ast.go:35`).
  - If `Module != ""` is not a declared module alias in the file, emit a clear error (`TYP_UNKNOWN_MODULE_ALIAS` / `IMP010`).
  - Verify the referenced type is actually exported by that module's `iface.Iface` (reject unexported types cleanly, respecting `m-v1-nominal-type-identity.md`).
  - Resolve to the exact same `types.TCon` / `types.TApp` / expanded alias as an explicit `import std/set (Set)` — so `S.Set[int]` and `Set[int]` unify identically!
- **Formatter (`internal/format/types.go`):**
  - When `SimpleType.Module != ""` or `TypeApp.Module != ""`, emit `Module + "." + Name` (and `[Args]`), preserving `S.Set[int]` verbatim on round-trip.
- **Docs:** Remove the `Module-qualified type names do not parse` row from `docs/LIMITATIONS.md` once shipped!

---

### Milestone E3: Newline-Safe Block Termination After `let` Statements (`(a, b)` and `{ r | x: 1 }`)

#### Surface Syntax
Inside `{ ... }` blocks (function bodies, match-arm blocks, `if`-branch blocks), allow:
```ailang
pure func bump(r: {a: int}) -> {a: int} {
  let x = 1
  {r | a: x}
}

pure func pair(n: int) -> (int, int) {
  let a = n + 1
  let b = n * 2
  (a, b)
}
```
without requiring a `;` on the `let` line before `{` or `(`.

#### Why This Is Safe (No Regression on Multi-Line Expressions)
Let's check the exact ambiguity that R2 (`m-syntax-ai-forgiving.md`) guarded against:
- R2 avoided treating `\n (` as a statement separator in general because a multi-line curried call `f(a)\n(b)` or parenthesized infix continuation should not split.
- **HOWEVER**, look at `parseInfixExpression` / `parseCallExpression`:
  - A `let` statement's RHS (`let b = n * 2`) parses with `parseExpression(LOWEST)`. Today, because the lexer skips newlines, `parseExpression(LOWEST)` sees `peekToken = LPAREN` on the **next line** and greedily calls `parseCallExpression` on `2` → `2(a, b)`!
  - And for `let x = 1\n{r | a: x}`, `parseExpression(LOWEST)` stops at `1`, returns to the block loop, and `peekStartsNewlineBlockStatement()` returns `false` because `LBRACE` is not in the starter list!
- **Two-part surgical fix:**
  1. **No cross-line function call (`(` on a new line):** In `parseExpression` / infix `LPAREN` dispatch, do NOT treat `(` on a *strictly later line* (`p.peekToken.Line > p.curToken.Line`) as a function-call application on the previous expression! (Precedent: `parser_expr.go:53` ALREADY has a `Line` check for GAP2 `LPAREN` — we'll audit every call site in the corpus to confirm zero legitimate AILANG programs put `(` of a function call on a new line after the callee.)
  2. **Allow `LPAREN`, `LBRACE`, `LBRACKET` as newline block-statement starters (`peekStartsNewlineBlockStatement`):** Once cross-line `(` is no longer a call, a new-line `(a, b)`, `{ r | a: x }`, or `[a, b]` inside a `{ }` block unambiguously starts the next statement (typically the block's return value!).
  3. **Formatter (`internal/format/format.go:460`):** Update `startsWithStatementStarter` to match, so `ailang fmt` no longer emits ugly `;` before block-ending tuples/records/lists — verified against the full 36s `internal/format` round-trip & corpus test suite!

---

## High-Impact Decisions

| # | Decision | Options | Recommendation | Owner | Change Cost |
|---|----------|---------|----------------|-------|-------------|
| **D1** | AST representation for E1 (`let (a, b) = e`) | **(a)** Dedicated `ast.LetPattern{Pattern, Value, Body}` node vs **(b)** Optional `Pattern ast.Pattern` field on `ast.Let` | **(a) Dedicated `ast.LetPattern`** (or `ast.LetTuple`) — keeps `ast.Let.Name` non-empty everywhere and forces exhaustive switches (including `internal/format` and `internal/elaborate`) to handle it explicitly without silent bugs | Mark | Medium (touches AST + format + elaborate) |
| **D2** | Scope of E1 patterns | **(a)** Irrefutable tuple patterns only (`(a, b)`, nested tuples, `_`) vs **(b)** Also record patterns (`let {x, y} = r`) | **(a) Tuple patterns only** for E1 (record destructuring can be a clean additive follow-on if desired, keeping E1 small and crystal-clear) | Mark | Low (purely additive later) |
| **D3** | Cross-line `(` rule for E3 | **(a)** Disallow `callee\n(args)` across a newline everywhere (like Go/Ruby/JS ASI) vs **(b)** Only suppress cross-line `(` inside `{ }` block statements | **(a) Everywhere** (first verify 0 hits across all `examples/`, `std/`, and `internal/` test fixtures) — simpler, uniform rule with zero context-dependent surprises | Mark | Low (verified by corpus grep before coding) |

### Design Freeze
- [x] **D1** ratified: **(a)** Dedicated `ast.LetPattern` node (Mark, 2026-11-24)
- [x] **D2** ratified: **(a)** Irrefutable tuple patterns only in E1 (Mark, 2026-11-24)
- [x] **D3** ratified: **(a)** Disallow cross-line `callee\n(args)` uniformly after corpus verification (Mark, 2026-11-24)

### Deferred Decisions (Agent Latitude)
- Exact error code names for invalid patterns in `let (...)` (e.g. `PAR_LET_PATTERN_REFUTABLE`) and unknown module aliases in qualified types (`IMP_UNKNOWN_ALIAS` or reusing `IMP010`).
- Internal helper decomposition and test file organization.

---

## Implementation Plan (3 Independent Milestones)

1. **Milestone 1 — E3 (Newline before `(`, `{`, `[` in blocks):** ~0.5–1 day
   - Audit corpus for any cross-line `callee\n(args)` calls.
   - Update `internal/parser/` (`LPAREN` line check + `peekStartsNewlineBlockStatement`) and `internal/format/` (`startsWithStatementStarter`).
   - Verify full `go test ./internal/parser/... ./internal/format/...` (including 36s corpus round-trip) + `make check-doc-samples`.
2. **Milestone 2 — E2 (Module-qualified type names `Alias.Type[Args]`):** ~1 day
   - Add `Module string` field to `ast.SimpleType` and `ast.TypeApp`.
   - Update `parseType()` (`internal/parser/parser_type.go`), type resolution (`internal/elaborate/` + `internal/types/`), and `internal/format/types.go`.
   - Add unit + multi-module integration tests (`S.Set[int]`, `M.Map[string, int]`, unexported type rejection, unknown alias error, `ailang fmt` round-trip, `--strict-bytecode`).
   - Remove the stale row from `docs/LIMITATIONS.md`.
3. **Milestone 3 — E1 (`let (a, b) = expr` tuple destructuring):** ~1–1.5 days
   - Add `ast.LetPattern` (or `ast.Let.Pattern` per D1), parser support in `parseLetExpression()`, elaboration to single-arm `core.Match` in `internal/elaborate/`, and formatting in `internal/format/`.
   - Add tests across parser, elaborator, evaluator, `--strict-bytecode`, Z3 `ailang verify`, and `ailang fmt` idempotency.

---

## Success Criteria

- `let (a, b) = expr` (nested tuples, `_`, block and `in` forms) passes `ailang check`, `ailang run`, `ailang run --strict-bytecode`, `ailang verify`, and `ailang fmt` idempotently.
- `import std/set as S` + `func f(s: S.Set[int]) -> S.Set[int]` passes `ailang check`, `ailang run`, `--strict-bytecode`, and `ailang fmt` without a redundant `import std/set (Set)`.
- `{ let a = 1\n let b = 2\n (a, b) }` and `{ let x = 1\n {r | a: x} }` parse and format cleanly without `;`.
- Zero regressions across `make test`, `internal/format` corpus round-trip, `TestFmtDoesNotDriftFromTeachingPrompt`, and `make check-doc-samples`.
