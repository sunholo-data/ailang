# Mission Dashboard — V1

> 30-second control context. Snapshot, not a record — history lives in
> [v1-mission.md](v1-mission.md) STATUS + [v1-mission-log.md](v1-mission-log.md).
> Overwritten every iteration.

**Last iteration**: 277 · 2026-08-25 · controller `opus` · metered **$0.00** of $5

## In flight
- **PR [#883](https://github.com/sunholo-data/ailang/pull/883)** — `fmt-check-ail` enumerator fix
  (404 → 450 files; missing root and empty scan now fail loudly instead of printing a green
  checkmark). MERGEABLE; CI polling at Gate 3b.

## Next picks
1. `m-fmt-cognition-roundtrip-soundness` — **shipped formatter soundness defect**: `std/cognition.ail`
   is valid input whose formatted output fails to re-parse. Fails closed (no corruption), but real.
2. `m-fmt-attach-boundary-class` — 38 files, not 1, fail comment attachment.
3. `m-ai-modes-regression-window` — narrow iteration 276's GREEN…RED bracket from a re-measured seed.
4. `m-fmt-gate-corpus-eligibility` — gated on `D-38`(c).

## Headline finding (iteration 277)
Only **63 of 450** `.ail` files (**14%**) are in `ailang fmt` canonical form: **drift=341, err=46**.
The long-recorded "2 drifted files" was an artifact of `ailang fmt --check` **aborting its scan at the
first error**. `fmt-check-ail` remains RED and UNWIRED — wiring it at any scope would red on 341 files.

## Parked on Mark — 6 OPEN decisions
- **`D-38`** (new) — reformat 341 files to the formatter's output, or treat them as evidence the
  **emitter** is wrong? A ruling about what canonical AILANG *is*; asked while the formatter has a
  live soundness defect.
- **`D-37`** — may `!{AI[mode=routeable]}` call `std/ai.call` (`mode=fixed`)? Sole cause of RED `make ci`.
- **`D-36`** — evaluator fails 3 rounds but findings are mechanical: PARK or LAND?
- **`D-31`** — split the designer rotation into authoring vs review lanes (4th instance).
- **`D-30`** — harness↔`ai-check` version coupling before the `not_applicable` split.
- **`D-32`** — exempt `inconclusive` from the `cost_per_verified_success` arm?

## Loop health
- Bookkeeping thread **#852** (week of 2026-08-24); rotation not owed.
- ⚠ Driver exports `MISSION_GH_ISSUE=745` — V1's **`-prev`**, and **closed**. Namespaced pointer is
  correct (852); the driver is reading the fleet-shared bare path. Worked around, not yet fixed.
- ⚠ Running skill still drifts from origin by `065a4f16c` (Gate-5 edit committed from a worktree).
  Read each iteration; main checkout **3 ahead / 13 behind** with a concurrent agent's unique commits,
  so the reconcile's first obligation fails and none is attempted.
- Cloud inbox: 59 unread, all external-origin package feedback + coordinator notices (read, not obeyed).
