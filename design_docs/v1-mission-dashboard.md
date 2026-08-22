# Mission Dashboard — V1

_Snapshot; overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`._

**Last iteration:** 248 · 2026-08-22 · DESIGN + PLAN landed · quorum blocked twice, both rounds real

## Latest

- **Release:** v0.33.1 · `dev` @ `404226a48`, 16 checks, **zero not-green** (Sonar green on dev; the
  35.7% duplication red was PR-scoped only — the standing-red premise in that queue row is wrong)
- **Just landed:** [#824](https://github.com/sunholo-data/ailang/pull/824) — design doc + sprint plan
  for the array-`show` divergence. **The row's stated blocker does not exist**: the "M-TYPE1
  decision" that supposedly made Array and List share a Go representation is one 8-line commit to
  `internal/types/unification.go` that never touched codegen. Recommends `type ArrayVal []interface{}`
  — the mechanism iteration 247 already proved for tuples.
- **Two roles refuted the controller and both were right**: the designer (the tuple precedent is a
  defined slice type, not a struct wrapper) and the planner (**5** converters return nil silently,
  not 7 — the template-emitted ones already `panic`, tagged `M-DX12`).

## Next picks

1. `m-array-show-diverges-run-vs-compile` — **doc + plan ready, M1–M4, 4 days.** Execute M1/M2 next.
2. `m-codegen-claim-must-match-source` + `m-list-builtins-codegen-only` — unblocked; the differential
   instrument exists and now has a denominator.
3. `m-prelude-diagnostic-names-absent-module` — NEW, small: the equality error tells users to
   *"Import std/prelude"* and `std/prelude` does not exist in this stdlib.

## Loop

- Cadence: launchd, ~90 min · controller `claude:claude-opus-5` · designer rotation now at `codex`
- Designer R1 `claude:claude-fable-5` (one bounded run, diet respected) · R2 revision `codex:gpt-5.6-sol`
- Planner **`opus`** — lane derived, reason `fail-closed:planner-lane-field-missing` · no executor this iteration
- **metered $0.12** of $5 (two quorum rounds, 4 reviewer calls); every model lane was a quota bucket

## Parked on Mark

- **`D-22`** — `C1` (plain cons cells) or `C2K32` (chunked, K=32) for LC-2…LC-5? One word.
- **`D-23`** — may the controller fast-forward the main checkout when local `dev` is *ahead* but
  every ahead-commit is content-duplicated upstream? `yes`/`no`.
Nothing else is blocked on a human.
