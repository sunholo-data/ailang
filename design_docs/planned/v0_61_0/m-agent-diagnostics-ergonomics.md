# M-AGENT-DIAGNOSTICS-ERGONOMICS: High-Signal Agent Diagnostics & Contextual Record Fields

**Status**: Implemented (2026-11-24)
**Target**: v0.61.0 (pre-v1.0.0 polish)
**Priority**: P1 (High — directly targets 9 of 15 remaining failures in the `v0.16.19` + V2 `stretch + frontier` eval)
**Estimated**: 1 day across 4 independent milestones (M1–M4)
**Dependencies**: `M-SYNTAX-ERGONOMICS-V2` (implemented `39cfcae29`), `M-AGENT-STUCK-FIXES` (`internal/importhint`)
**Author**: Antigravity (attended session with Mark)
**Created**: 2026-11-24

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Deterministic diagnostics and unambiguous record-field parsing |
| A2: Replayability | 0 | No runtime or trace changes |
| A3: Effect Legibility | 0 | Effect checking unchanged |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | 0 | Contract clauses (`requires`/`ensures`/`invariant`) unaffected outside record fields |
| A6: Safe Concurrency | 0 | Purely frontend/diagnostic changes |
| A7: Machines First | +1 | Replaces 4 actively misleading diagnostics with actionable fixes and lifts `IMP010` self-repair |
| A8: Minimal Syntax | +1 | M1–M3 add zero syntax; M4 only relaxes contract keywords in unambiguous record-field positions |
| A9: Cost Visibility | 0 | Zero runtime cost |
| A10: Composability | 0 | No semantic changes |
| A11: Structured Failure | +1 | Dedicated coded diagnostics (`PAR_IF_BRANCH_BLOCK`, `PAR_LOCAL_FUNC`, `PAR_HASH_COMMENT`, `PAR_PIPE_OPERATOR`) + better `IMP010` suggestions |
| A12: System Boundary | 0 | No host/FFI changes |

**Net Score: +4** (≥ +2, no −1 on A1/A3/A4/A7).

---

## Problem Statement & Empirical Evidence

From the `stretch + frontier` (`n = 31`, `gemini-2-5-flash`, `39cfcae29`) failure audit:

1. **M1 — `PAR_IF_BRANCH_BLOCK` (`log_file_analyzer`, `ssa_constant_fold`, `binary_strings_1e18` — 3/15 failures)**:
   - Unbraced multi-statement `then` branches (`if c then let x = ...; expr else ...`) currently emit `expected next token to be else, got ;` + misleading `PAR017: ';' is only valid inside '{ ... }' block bodies, not in expression-body functions` even when already inside a `{ }` function body.
2. **M2 — Stronger `IMP010` suggestions (`glob_match_spec`)**:
   - `IMP010: symbol 'nthChar' not exported by 'std/string'` returned no suggestion because `nthChar` is neither a prefix nor edit-distance ≤ 2 of `charAt`/`chars`, leaving agents guessing blind in self-repair.
3. **M3 — 3 misleading parser diagnostics (`cp_60166`, `run_length_encode_v2`, `cp_60149` — 3/15 failures)**:
   - `func helper(...) { ... }` inside a block emits `expected (, got IDENT` + `PAR020` instead of `PAR_LOCAL_FUNC`.
   - `# comment` (Python-style) emits `expected [, got IDENT` (from array literal `#[...]`) instead of `PAR_HASH_COMMENT`.
   - `x |> f` (pipe operator) emits `PAR016` telling the agent *"use De Morgan's law `~(~a & ~b)` for bitwise OR"* instead of `PAR_PIPE_OPERATOR`.
4. **M4 — Contextual contract keywords as record field names (`dep_resolver_backtrack`)**:
   - `type PackageDefinition = { requires: [Requirement] }` fails with `PAR_FIELD_NAME_EXPECTED` because `requires` is a contract keyword, even though `requires:` in record types/literals/updates and `.requires` in field access are 100% unambiguous.

---

## Proposed Design

### Milestone M1: `PAR_IF_BRANCH_BLOCK` (`internal/parser/parser_expr.go`)
- In `parseIfExpression()`, after parsing `expr.Then`, if `p.peekTokenIs(lexer.SEMICOLON)`, emit `PAR_IF_BRANCH_BLOCK` with clear suggestions (`if cond then { let x = ...; ... } else ...` or `let ... in ...`) instead of falling through to `expected else, got ;` + `PAR017`.

### Milestone M2: Word-Stem & Common-Miss `IMP010` Suggestions (`internal/importhint/importhint.go`)
- Extend `closestExport(want, exports)` with a camelCase/snake_case word-token overlap fallback (e.g. `nthChar` shares `"char"` with `charAt`/`chars`, preferring the shortest/highest-overlap export) while preserving `""` for unrelated symbols (`"nope"`, `"zzqqxx"`).

### Milestone M3: `PAR_LOCAL_FUNC`, `PAR_HASH_COMMENT`, `PAR_PIPE_OPERATOR`
- **`PAR_LOCAL_FUNC`** (`internal/parser/parser_lambda.go`, `parseLambda`): when `p.peekTokenIs(lexer.IDENT)` after `func` in expression position, emit `PAR_LOCAL_FUNC` directing to top-level `func` or `letrec <name> = \args. ...`.
- **`PAR_HASH_COMMENT`** (`internal/parser/parser_literals.go`, `parseArrayLiteral`): when `#` is not followed by `[`, emit `PAR_HASH_COMMENT` explaining `#` starts array literals `#[...]` and comments use `--` (or `//`).
- **`PAR_PIPE_OPERATOR`** (`internal/parser/parser_error.go` + infix check): when `|` is immediately followed by `>` (`|>`), emit `PAR_PIPE_OPERATOR` instead of `PAR016` (bitwise OR De Morgan hint).

### Milestone M4: Contextual Contract Keywords in Record Field Positions
- Add `isContextualFieldKeyword(t lexer.TokenType) bool` for `REQUIRES`, `ENSURES`, `INVARIANT`, `PROPERTY`, `TEST` (leaving core keywords like `with`, `recv`, `if`, `let`, `match`, `in`, `func` reserved).
- Accept contextual field keywords in:
  1. Record type field defs (`internal/parser/parser_type.go`: `parseRecordFieldDef`)
  2. Record literals & record updates (`internal/parser/parser_literals.go`: `parseRecordLiteral`)
  3. Record field access `r.requires` (`internal/parser/parser_expr.go`: `parseFieldAccess`)
  4. Record pattern field labels `{ requires: rs }` (`internal/parser/parser_pattern.go`: `parseRecordPattern`, explicit `field: pat` form)

---

## Success Criteria
- All 4 milestones have dedicated unit tests in `internal/parser/` and `internal/importhint/`.
- `dep_resolver_backtrack`-style record types/literals/updates/access with `requires`/`ensures`/`invariant` pass `ailang check`, `ailang test`, `ailang test --strict-bytecode` (0 VM fallbacks), and `ailang fmt --check`.
- Zero regressions across `internal/parser`, `internal/importhint`, `internal/format`, `internal/testing`, and `docsamples`.
