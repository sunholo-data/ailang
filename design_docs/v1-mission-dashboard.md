# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.*

**Last iteration:** 285 · 2026-08-26 · controller `opus`

## Release
- **v0.34.0** shipped 2026-08-26. Next planned bucket: `v0_35_0`.

## In flight / next
1. **`m-fmt-printer-line-width-limit` M0+M1 — PR [#918](https://github.com/sunholo-data/ailang/pull/918), NOT LANDED.**
   Evaluator **FAIL 78/100**, overriding a numeric pass on one blocking defect: a `let..in` at
   beginning-of-line reads `writer.col == 0` before deferred indentation flushes, so the formatter
   emitted a **122-rune line under `MaxWidth=120`**. `TestLetInWidthBoundary` misses it because it
   fakes the column with `p.w.write(spaces)`, which flushes `atBOL`. **Round 2 owed** — fix + a
   regression test driven through a real `hardline()`, then M2/M3.
2. **Dev red fix — PR [#919](https://github.com/sunholo-data/ailang/pull/919), awaiting CI.**
3. `m-motoko-lane-enumerator-field-order-blind` (iter-284) · `m-verify-stdlib-wrapper-exit-propagation-unpinned` ·
   `m-skills-parity-no-ci-gate` · `m-eval-suite-agent-tempdir-unguarded` (iter-283).
4. **NEW `m-weekly-sweep-orphans-2026-08-26`** — 18 orphans of 92 open issues, mostly the GitHub
   mirrors of the unrouted public-feedback queue, including `#900` which tracks that unrouting.

## Loop health
- **`dev` WAS RED on the REQUIRED `test` context** — `cmd/ailang/eval_suite.go` went 800 → **826**
  at `2c8498886`, tripping `make check-file-sizes`. Blocked every PR in the repo, including #918's
  own `test` job. V1 owns this repo, so it outranked the queue. Fix in #919 (826 → 778).
- **Cadence:** launchd; driver ran **UNPINNED** again (`MISSION_WORKDIR`/`AILANG_DRIVER_SRC` both unset).
- **Routing (all four roles spawned):** designer **not needed** (doc ready from iter-284, Fable diet
  unspent) · planner `opus` (`fail-closed:planner-lane-field-missing`) · executor `codex:gpt-5.6-sol`
  (probe rc=0) · evaluator `sonnet`. generator≠judge held.
- **Running skill == origin** (`cmp` rc=0 through the resolved `readlink` target; same inode).
- **Two wall-clock-bound tests fail under concurrent-agent load** — `TestIdleReader_IdleWindow`
  (`internal/ai/ollama`) and `TestSolve_HardTimeout_FakeSolverIgnoringT` (`internal/smt`). Both rc=0
  in both arms when isolated; neither package depends on `internal/format`. Rule-3m class.

## Parked on Mark
- **`D-41` (the ONLY open row; 41 total, 40 resolved)** — may an ACTIVE prompt version be edited in
  place, or must a content change bump the version? Bears on eval-baseline reproducibility.
- **SonarCloud has no token on this rig**; standing `new_coverage` red, 8 consecutive commits.

## Quota / spend
- Iteration 285 metered **$0.00** (no quorum round; doc was already refined). Quota buckets: opus
  (controller + planner), codex (executor), sonnet (evaluator).
