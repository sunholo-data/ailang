# Mission Dashboard — V1

> Snapshot only (overwritten each iteration); history lives in `v1-mission.md` (STATUS + queue)
> and `v1-mission-log.md`.

**Iteration 281 · 2026-08-26 · bookkeeping issue [#852](https://github.com/sunholo-data/ailang/issues/852)**

## Latest
- **Release**: v0.33.2. `dev` @ `2fc0c8b77`.
- **This iteration**: PARKED by design — implemented the queued formatter fix, measured that it
  **silently deletes user comments**, and withdrew it. No source shipped; record + corrected rows only.
- **Key find**: attachment and emission are **coupled**. Registering a comment-attachment list whose
  owner the printer renders on one line turns a fail-closed refusal (rc=2, nothing written) into
  silent comment loss (rc=0, comments gone). Measured: `std/dom.ail` 54 → 50 comment lines,
  `std/ai/streaming.ail` 135 → 132.

## In flight / next
1. `m-format-comment-brackets-break-wall-scan` — class 5, purely lexical, needs no printer change.
2. Confirm class 2 (top-level tail) is owned by the already-registered file top-level list.
3. `m-fmt-typedecl-printer-needs-multiline-emit` — **BLOCKED on `D-38`** (a multi-line type-decl form
   changes canonical AILANG).

## Loop health
- Cadence: launchd, unattended. Controller `opus`.
- **Routing deviation, 4th consecutive iteration**: Agent tool forbidden by this session's operating
  instructions, so designer/planner/executor/**evaluator** were not spawned — **no independent judge**.
  The repo's own `TestCorpusCommentGate`, not a judge, caught half of this iteration's defect.
- **Stale-PATH messaging trap fired 4th consecutive iteration**: `~/go/bin/ailang` (v0.33.2) accepts an
  invalid `AILANG_MESSAGES_STORE` at rc=0. Always build fresh before touching messages.
- Cost: metered **$0.00** of $5. No Fable spend; rotation pointer untouched.
- Standing non-required red: SonarCloud `new_coverage` 52.8% < 80 (filed, not the pick).

## Parked on Mark — 6 OPEN decisions
`D-30` ai-check version coupling · `D-31` designer rotation has one usable authoring lane ·
`D-32` `inconclusive` in the cost KPI · `D-36` round-3 evaluator FAIL: land or park ·
`D-37` `mode=routeable` → `std/ai.call` (sole reason `make ci` is red) ·
`D-38` reformat 341 files, or is the formatter's canonical form wrong? — **now also blocks the
formatter work above.**
