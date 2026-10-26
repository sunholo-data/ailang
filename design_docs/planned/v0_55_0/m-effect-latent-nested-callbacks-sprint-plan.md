# Sprint Plan — M-EFFECT-LATENT-NESTED-CALLBACKS

**Design doc**: [m-effect-latent-nested-callbacks.md](m-effect-latent-nested-callbacks.md) (freeze rulings recorded 2026-10-26, Mark, attended)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-EFFECT-LATENT-NESTED-CALLBACKS.json`
**Tracks**: #1718, #1826
**Target**: v0.55.0 (minor; Breaking — soundness)
**Branch**: `p0/effect-latent-nested-callbacks` (base `origin/dev` `8e634480f`)
**Risk**: Medium — touches the type checker's per-App effect publication; prototype measured 0 corpus flips.
**Approval**: pre-approved by Mark for this run (no human pause).

## Current status (measured on base `8e634480f`)

The scratch repro matrix (design §Problem Statement, copied from the design session's `dd1/`) was
re-run against a binary built from `8e634480f`: every row matches the design's base column —
R1–R9 `PASS` (leak), R10/R11 and S2–S4 `TAIL`, S1 `INVARIANT`, controls as tabulated. No upstream
change since the design's base (`772703831`) moved any of them.

## Velocity

Recent internal/types soundness sprints (M-EFFECT-LATENT-FUNCTION-VALUES, M-EFFECT-ROW-VAR-UNIFICATION)
landed ~150–250 LOC/day including tests. This sprint is ~650 LOC (≈300 impl, ≈350 tests/docs),
with a working prototype to port, so 2 days is realistic; one executor run per milestone.

## Registry reuse gate

| Milestone | Package | Action | Reason |
|---|---|---|---|
| M1–M4 | none | none | Compiler-internal type-checker change (`internal/types`); no package-shaped capability. `ailang pkg search effect callback` is irrelevant to checker internals. |

## AILANG syntax gate

AILANG prompt version loaded: v0.16.7 (`ailang prompt --version-active` on the base build).

Showcase module `examples/runnable/effect_latent_nested_callbacks.ail`:

| Row | Decision |
|---|---|
| contracts | skip: the example demonstrates effect rows on higher-order calls; contracts would add an unrelated Z3 dependency |
| effects | include: `! {IO}` on every caller that hands an effectful callback in a tuple/list/Option, plus a `mapE` S2 form |
| inline tests | skip: the functions under demonstration are `! {IO}`; inline tests run pure functions only |

## Milestones

### M1 — Data-position walker + concrete-tail normalisation (#1826) — ~200 LOC

- New `internal/types/typechecker_latent_data.go`: `dataArrows(t, expand)` (list/array/map/tuple/TApp
  args/record fields/TRecord2 labels; never arrow params or returns; depth-capped, alias heads
  expanded per level when an expander is supplied), `latentRow(fn)` (closes a
  `ConcreteEffectContract` tail, the `applicationCallRow` rule), `nestedLatentRow`.
- Site (d) `inferApp`: a `ConcreteEffectContract` callee contributes `callRow`.
- Site (b) top-level branch publishes `latentRow(fn)` (publish + re-zonk).
- Tests: walker unit tests (positions, alias per level, recursive alias terminates, no arrow
  descent); S1, S2, S3, S4 (+ pure S4 variant rejected without a tail line); mcp_oauth
  `revokeFamily` reduced fixture.
- **Acceptance**: S1–S4 check clean; pure S4 → `Missing effects: IO` and no `Unresolved effect tail`.

### M2 — Nested hand-off (#1718) — ~180 LOC

- Site (a) `resolvedCallbackMask`: mask true when `dataArrows(param)` holds an open non-concrete arrow.
- Site (b) nested branch: publish `nestedLatentRow(argType)` and recompute it after substitution.
  Tails owned implicitly by the caller's binders are dropped per-arrow before the union; two
  remaining distinct tails are a #616 conflict (`CheckedUnionEffectRows`) and publish no row, so
  the validator's fail-loud missing-callback-row invariant fires (never first-wins).
- Site (c) `implicitRowsAbove`: structural, via the same walker.
- Tests: R1–R11 (R5 cross-module), R6/R7/R8/R9 shapes, AC2 `Some(logIt)`/`Hold(logIt)` charged,
  AC4 accept controls, AC5 reject controls incl. `mixed_rows` naming FS, two-engine
  run-rejects-before-execution for R1/R2/R5-style programs.
- **Acceptance**: AC1, AC2, AC4, AC5 green.

### M3 — erase ⊆ charge invariant (#1718 RC1) — ~60 LOC

- Test-only (agent's choice per Deferred Decisions): for every application inside a callee body
  whose call row / callback row tail is erased as implicit, the tail must belong to an arrow
  reachable through `dataArrows` of one of the enclosing function's parameter types, and that
  parameter must be charged by `resolvedCallbackMask`. Exercised over the R/C fixtures.
- **Acceptance**: AC7 — reverting (a) while keeping (c) turns it red.

### M4 — Corpus, docs, examples, changelog — ~200 LOC (docs/fixtures)

- Base-vs-fixed sweep over `examples/**` + `std/**` (`sweep.sh`), registry-cache mcp_oauth
  0.1.0/0.1.1 copies; record flip counts.
- Remove the #1718 rows from `docs/LIMITATIONS.md` and `docs/docs/reference/limitations.md`;
  add the `map` → `mapE` migration note.
- `examples/runnable/effect_latent_nested_callbacks.ail` + manifest entry.
- Teaching-prompt sentence extended and verified with `ailang check`.
- Changelog fragment `changelogs/unreleased/2026-10-26-effect-latent-nested-callbacks.md`
  (`### Fixed — Breaking (soundness) …`), listing the three intended flips and migrations.
- **Acceptance**: AC8, AC10; gates `make test-changed`, `make lint`, `make verify-examples`,
  `make test-imports`, `make check-file-sizes`, `make check-changelog` (all unpiped). Never `make test`.

## File-size guard

`typechecker_core.go` is exactly 800 lines — not touched. `typechecker_functions.go` 756 → ≤765.
`typechecker_app_effects.go` 374 → ~400. New code goes in `typechecker_latent_data.go`.

## Mutation plan (recorded in the PR)

Per design table: revert (a) → AC1 red; (b-nested) → AC1 red with invariant; (c) → AC2 red;
(d) → AC3 red; (b top-level `latentRow`) → S4 tail line. Done in a scratch copy, never `git checkout <file>`.
