# M-AILANG-FMT: `ailang fmt` — canonical AILANG formatter (deferred follow-up)

**Status**: Planned (deferred / stub)
**Target**: TBD (post-v0.29.x)
**Split from**: [m-syntax-ai-forgiving.md](../../implemented/v0_30_0/m-syntax-ai-forgiving.md) (sprint discrepancy D1)

> This is a **placeholder** doc capturing the formatter deferred out of the
> M-SYNTAX-AI-FORGIVING sprint, so the canonical-form erosion risk is tracked rather
> than dropped. It is intentionally minimal; flesh it out when the formatter is scheduled.

## Why this exists

The M-SYNTAX-AI-FORGIVING sprint plan assumed `ailang fmt` existed and needed only
"~0.5d of canonicalization." It **does not exist** — there is no `fmt` subcommand in
`cmd/ailang/main.go` and no formatter package. A from-scratch AST-reprinting
formatter (idempotence, round-trip, comment preservation, a CI gate) is a multi-day
item that would blow the parser sprint's budget and smuggle a formatter into a parser
change. The sprint therefore **split it here** (plan option b).

M-SYNTAX-AI-FORGIVING made the parser **accept** several statement-separator forms
that all produce the same AST:

- `=`-body `;`-sequence: `func f() = s1; s2; e`  (R1)
- braced `;`-sequence: `func f() { s1; s2; e }`
- braced newline-sequence: `func f() { s1\n s2\n e }` (R2)
- `let … in`: `func f() = let x = e in rest`

Acceptance is the v1.0 gate (bar v2 clause 3), and it is satisfied. But with multiple
accepted spellings and **no formatter**, source style can drift. In-sprint this is
mitigated by (1) the corpus AST-diff fuzz gate (same AST → same result — presentation
determinism / Axiom A1 is untouched, since A1 is about execution/trace determinism)
and (2) documented canonical-form guidance (`prompts/agent/dialect-traps.md`). A
formatter is a *convenience* on top of that, not a correctness requirement.

## Scope (when scheduled)

- `ailang fmt [--write] [--check] <files...>` — reprint AST to a canonical form.
- **Canonical separator choice** (deferred decision from the parent doc): pick ONE of
  `;`-on-line vs newline-per-statement as the emitted form. Recommendation: **newline
  per statement inside `{ }`** (matches what models write and reads cleanest), with
  the last expression bare (no trailing `;`); keep `let … in` only where explicit.
- Idempotence (`fmt(fmt(x)) == fmt(x)`), comment preservation, round-trip on the
  corpus, and a CI `--check` gate.
- Reuse the deep-AST infrastructure from `cmd/astdump` where useful.

## Non-goals

- Any change to what the parser *accepts* (that shipped in M-SYNTAX-AI-FORGIVING).
- Semantic rewrites — `fmt` is presentation-only.

## Open questions

- Comment attachment model (leading/trailing/floating) — the usual formatter hard part.
- Whether `fmt --check` becomes a `make verify-examples`-adjacent CI gate or opt-in.
