# M-PARSER-BLOCK-LET-SEPARATOR: Consistent statement-separator handling after block-RHS `let`

**Status**: PARKED (evidence-gated — MEASURED NEGLIGIBLE 2026-07-23, mission iteration 91; stays parked, do not route to a sprint)
**Target**: TBD (v1.0 clause-3 accessibility, footgun burn-down)
**Priority**: P3 (minor DX footgun; NOT blocking)
**Estimated**: 1–2 days (parser change — Conflict Surface mandatory)
**Dependencies**: None
**Created**: 2026-07-17 (mission iteration 40, split out of m-dx-expected-fail-fixes)
**Source**: Gate-2 live-repro of Bug 3 in `m-dx-expected-fail-fixes` (serve_api_webhook.ail)

---

## Problem Statement

AILANG block bodies inconsistently require a statement separator (`;`) between a `let` binding
and a following expression, depending on the SHAPE of the `let`'s right-hand side.

**A simple-RHS `let` tolerates omitting the separator** (parses cleanly at HEAD `1ee919386`):

```ailang
export func f(x: int) -> int {
  let y = x + 1      -- no ';' and no 'in'
  y * 2              -- final expression
}
-- ✓ No errors found
```

**A block-RHS `let` (RHS ends in `}` — e.g. `match {...}`, `if {...}`) does NOT** — same shape,
but fails to parse:

```ailang
export func f(o: Option[string]) -> string {
  let sig = match o {
    Some(s) => s,
    None => "none"
  }                  -- no ';' and no 'in'
  "result: " ++ sig  -- final expression
}
-- ✗ PAR_UNEXPECTED_TOKEN: expected next token to be }, got STRING instead
```

Adding a `;` after the `match`'s closing `}` (or using `let ... in`) fixes it. The canonical
forms (`let x = e; ...` / `let x = e in body`) both work; only the *separator-elision* path is
inconsistent between simple-RHS and block-RHS bindings.

## Why it matters (thesis-relevant)

AILANG targets AI code synthesis. Models frequently omit trailing separators, and an
inconsistency where "the same omission works for `let y = x+1` but not for `let y = match{...}`"
is exactly the kind of non-obvious footgun the clause-3 accessibility work burns down. It cost a
real example (`serve_api_webhook.ail`) a parse failure that read as a language bug.

## Open design question (why this is NOT auto-fixed)

Two defensible resolutions — **decide with data, not vibes**, before any parser change:

1. **Make block-RHS `let` consistent with simple-RHS** — accept the separator-elided form after a
   `}`-terminated RHS too. Risk: the block parser's `}`-lookahead currently disambiguates "end of
   enclosing block" from "continue with another statement"; relaxing it could mask genuine
   missing-`}` errors. Conflict Surface must map the lookahead/precedence interaction.
2. **Remove the simple-RHS leniency instead** — require an explicit `;`/`in` after EVERY `let`,
   and ship a teaching diagnostic ("add `;` or use `let ... in`") for the elided form. More
   uniform, but changes accepted syntax for existing code — needs a `verify-examples` sweep.

Per **PROGRAM.md default-bias-not-core**, option (2) (diagnostic + doc, no grammar relaxation) is
the lower-risk default unless eval data shows the elided block-RHS form is a high-frequency model
failure that option (1) would eliminate.

## Evidence gate (before routing to a sprint)

Route only with a measured failure rate: count `PAR_UNEXPECTED_TOKEN ... got X, expected }`
occurrences attributable to block-RHS-`let` separator elision across recent eval rotations
(`tools/analyze_run_steps.py` / eval logs). If negligible, this stays parked; if material, it
routes as a NEW-DOC parser sprint with a mandatory Conflict Surface.

### MEASUREMENT — mission iteration 91 (2026-07-23): NEGLIGIBLE → stays parked

The bug is **REAL at HEAD** (`82084c1a9`) — live-reproduced: a simple-RHS `let y = x+1` tolerates
separator elision (only fails the unrelated `MOD014`), a block-RHS `let sig = match o {...}` fails
`PAR_UNEXPECTED_TOKEN: expected next token to be }, got STRING`. Confirmed.

But the **routing evidence is negligible**. Measured across the entire local eval corpus —
**27,359** result files (all `eval_results/baselines/*` + `eval_results/rotation/*`, all-time):

| Metric | Count |
|---|---|
| Files with any `expected next token to be }` error | 540 |
| Files where the code contains a block-RHS-`let` separator-elision pattern **and** the parser's error line lands on that block's closing brace (±2) | **10 all-time** / **8 in the current era** (v0.29.2 + v0.30.0) |
| Files where block-RHS-`let` elision is the **decisive** error (first parse error, ≤3 total parse errors — i.e. it uniquely cost an otherwise-parseable run) | **0** |

Every one of the ~10 attributable occurrences sits inside a **catastrophically-broken generation**
(51 – 1,542 parse errors per file; `compile_ok=false`; the *first* error is always elsewhere on an
earlier line — `ILLEGAL`/`PAR_FIELD_NAME_EXPECTED`/`PAR_TYPE_UNEXPECTED`/etc.). The `expected }` at
the footgun line is parser-recovery cascade noise, not the cause of the failure. Fixing this footgun
would flip **zero** of these runs from fail → pass. The original trigger (`serve_api_webhook.ail`)
was a hand-authored **example**, not a model eval failure.

**Verdict:** per this gate, negligible → **stays parked**. Routing a *core parser* change
(default-bias-not-core, PROGRAM.md) against zero decisive eval evidence is not justified. Re-open
only if a future rotation produces a **decisive** case (block-RHS-`let` elision as the sole/first
error in an otherwise-parseable file). Method preserved in the mission log (iteration 91); the
classifier is a ~40-line code-pattern + error-line cross-check (regenerable from the log entry).

## Key files (for the eventual sprint)

- `internal/parser/parser_expr.go` — block / `let` statement parsing, `}`-lookahead
- `internal/parser/parser_decl.go` — function body block parsing
- Regression: `examples/runnable/serve_api_webhook.ail` already guards the canonical form; a fix
  would add a guard for the separator-elided block-RHS form.
