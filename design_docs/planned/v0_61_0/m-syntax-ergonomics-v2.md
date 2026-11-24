# M-SYNTAX-ERGONOMICS-V2: Lambda Tuple Destructuring, Record `let` Destructuring, and Top-Level `let` Semicolons

**Status**: Implemented (2026-11-24)
**Target**: v0.61.0 (pre-v1.0.0 polish)
**Priority**: P1 (High — directly targets the top remaining AI parse errors in `v0.16.19`)
**Estimated**: 1.5–2 days across 3 independent milestones (E4, E5, E6)
**Dependencies**: `M-SYNTAX-ERGONOMICS-V1` (implemented `de314a613` — `ast.LetPattern` and single-arm `core.Match` desugaring)
**Author**: Antigravity (attended session with Mark)
**Created**: 2026-11-24

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Purely syntactic desugaring at parse/elaboration time; `Parse(fmt(x)) ≡ Parse(x)` preserved |
| A2: Replayability | 0 | No runtime or trace changes |
| A3: Effect Legibility | 0 | Effect checking unchanged |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Desugars to `core.Match` on tuple/record patterns, which Z3 already encodes |
| A6: Safe Concurrency | 0 | Purely sequential syntax |
| A7: Machines First | +1 | Directly eliminates 4 remaining `v0.16.19` parse failures (`type_unify`, `dep_resolver_backtrack`, `binary_strings_1e18`, `cp_60166`) |
| A8: Minimal Syntax | 0 | Reuses existing `TuplePattern`, `RecordPattern`, and `ast.LetPattern` — zero new keywords |
| A9: Cost Visibility | 0 | Zero runtime overhead; identical Core AST |
| A10: Composability | +1 | Makes `map`/`filter`/`foldl` over `M.toList(m)` and `zip(xs, ys)` compose cleanly (`\(k, v). ...`) |
| A11: Structured Failure | +1 | Refutable patterns in `let`/`\` still fail loudly with actionable `PAR_LET_PATTERN` / `match` hints |
| A12: System Boundary | 0 | No host/FFI changes |

**Net Score: +5** (≥ +2, no −1 on A1/A3/A4/A7).

---

## Problem Statement & Evidence

From the `v0.16.19` `gemini-2-5-flash` 54-benchmark eval (`core` + `stretch` + `frontier`):

1. **E4 — Lambda parameter tuple destructuring (`\(a, b). body`)**:
   - Directly caused parse failures in **`type_unify`** (`map(\(varName, typ). (varName, apply(s2, typ)), s1)`) and **`dep_resolver_backtrack`**.
   - Every time you iterate over `M.toList(m)` or `zip(xs, ys)` with `map`/`filter`/`foldl`, you receive a `(k, v)` tuple; requiring `\pair. match pair { (k, v) => ... }` is pure ceremony.
2. **E5 — Record destructuring in `let` (`let {name, age} = person`, `let {name, ...} = person`)**:
   - Natural irrefutable sibling of `let (a, b) = pair` (explicitly deferred from E1 as D2).
   - Uses the exact same `ast.LetPattern` → single-arm `core.Match` on `core.RecordPattern` pipeline shipped in `M-SYNTAX-ERGONOMICS-V1`.
3. **E6 — Optional trailing `;` on top-level `let` bindings (`let MOD = 1000000007;`)**:
   - Directly caused `PAR017` failures in **`binary_strings_1e18`** and **`cp_60166`** — both wrote a single harmless trailing `;` on a module-level constant (`let MOD = 1000000007;`).

---

## Proposed Design

### Milestone E4: Lambda Tuple Destructuring (`\(a, b). body`, `\acc (k, v). body`)

- **Surface syntax:**
  ```ailang
  map(\(k, v). "${k}=${show(v)}", M.toList(m))
  foldl(\acc (k, v). acc + v, 0, M.toList(m))
  map(\(a, (b, _)). a + b, nested)
  ```
- **Parser (`internal/parser/parser_expr.go`, `parseLambdaExpression`):**
  - When a lambda parameter starts with `lexer.LPAREN`, parse an irrefutable tuple pattern (`isIrrefutableTuplePattern(pat)`, reusing E1's validator!).
  - Desugar in the parser (or preserve in `ast.Param`? — **Decision D1** below):
    - *Recommended (Option A — desugar in parser to synthetic param + `ast.LetPattern` body)*:
      `\(a, b). body` → `\$tup0. let (a, b) = $tup0 in body` (or store `Pattern` on `ast.Param` so `ailang fmt` preserves `\(a, b). body` verbatim — Option B!).
    - Wait: **Option B (`Pattern ast.Pattern` on `ast.Param`)** is only ~15 lines in `elaborate` + `format`, AND keeps `ailang fmt` idempotent (`\(a, b). body` formats back to `\(a, b). body` instead of `\$tup0. let (a, b) = $tup0 in body`)! Let's recommend **Option B**.
  - Reject refutable patterns in lambda params (e.g. `\(Some(x)). ...`) with `PAR_LAMBDA_PATTERN` pointing to `match`.

### Milestone E5: Record Destructuring in `let` (`let {name, age} = r`, `let {name, ...} = r`)

- **Surface syntax:**
  ```ailang
  let {name, age} = person
  let {host, port: p, ...} = config
  let {name, ...} = person in name
  ```
- **Parser (`internal/parser/parser_expr.go`):**
  - In `parseLetExpression()`: also accept `p.peekTokenIs(lexer.LBRACE)` and route into `parseLetPattern(let.Pos)`.
  - Extend `isIrrefutableLetPattern(pat)` to accept `*ast.RecordPattern` whose field values are all irrefutable (`*ast.Identifier`, `*ast.WildcardPattern`, nested irrefutable `*ast.TuplePattern` / `*ast.RecordPattern`). Reject literal/constructor field patterns (e.g. `let {status: 200} = r`) with `PAR_LET_PATTERN`.
- **Elaboration & Formatter:**
  - `ast.LetPattern` ALREADY handles elaboration (`e.elaboratePattern` → single-arm `core.Match`), `blockScopeSets` (just add `*ast.RecordPattern` case to `astPatternBinders` in `internal/elaborate/scc.go` if not already there!), `internal/format`, and `internal/testing`!

### Milestone E6: Optional Trailing `;` on Top-Level `let` Bindings

- **Surface syntax:**
  ```ailang
  module mymod
  let MOD = 1000000007;
  export pure func norm(x: int) -> int = x % MOD
  ```
- **Parser (`internal/parser/parser_file.go`):**
  - After parsing a top-level `let` declaration (without `in`), if `p.peekTokenIs(lexer.SEMICOLON)`, consume the optional trailing `;` instead of rejecting it with `PAR017`.
  - Keep `PAR017` for `;` inside `func f() = a; b` expression-body functions (where `;` is genuinely a mixed-style bug).

---

## High-Impact Decisions

| # | Decision | Options | Recommendation | Owner | Change Cost |
|---|----------|---------|----------------|-------|-------------|
| **D1** | AST representation for E4 (`\(a, b). body`) | **(a)** Desugar in parser to `\$tup. let (a, b) = $tup in body` vs **(b)** Optional `Pattern ast.Pattern` on `ast.Param` (desugared in elaborator, printed cleanly by `ailang fmt`) | **(b)** Optional `Pattern` on `ast.Param` — preserves `\(a, b). body` in `ailang fmt` and avoids synthetic `$tup` names leaking into formatted code | Mark | Low |
| **D2** | Scope of E5 (`let { ... } = r`) | **(a)** Closed & rest record patterns (`{a, b}`, `{a: x, ...}`) with irrefutable sub-patterns vs **(b)** Closed only | **(a)** Both closed and `...` rest patterns (reuses existing `RecordPattern` directly) | Mark | Low |

### Design Freeze
- [x] **D1** ratified — **(b)** Optional `Pattern ast.Pattern` on `ast.Param` (Mark, 2026-11-24)
- [x] **D2** ratified — **(a)** Closed & rest record patterns in `let` (Mark, 2026-11-24)

---

## Success Criteria

- `\(a, b). a + b`, `\acc (k, v). acc + v`, `let {name, ...} = r`, and top-level `let MOD = 1000000007;` all pass `ailang check`, `ailang test`, `ailang test --strict-bytecode` (0 VM fallbacks), and `ailang fmt --check` idempotently.
- Zero regressions across `internal/parser`, `internal/elaborate`, `internal/format`, `internal/testing`, and `make check-doc-samples`.
