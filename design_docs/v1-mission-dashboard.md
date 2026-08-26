# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.*

**Last iteration:** 284 · 2026-08-26 · controller `opus`

## Release
- **v0.34.0** shipped 2026-08-26. Next planned bucket: `v0_35_0`.

## In flight / next
1. **`m-fmt-printer-line-width-limit`** — design doc written (iter-284, `design_docs/planned/v0_35_0/`),
   quorum **blocked twice**, both rounds' objections applied. **Ready for sprint-planner.** Per `D-39`
   this is the queue head; it also gates `m-fmt-typedecl-printer-needs-multiline-emit` and forbids
   wiring/freezing the `fmt` gate until it lands + the 2nd `fmt --write` pass runs.
2. `m-motoko-lane-enumerator-field-order-blind` — NEW (iter-284, judge-found, controller-reproduced).
3. `m-verify-stdlib-wrapper-exit-propagation-unpinned` · `m-skills-parity-no-ci-gate` ·
   `m-eval-suite-agent-tempdir-unguarded` (all iter-283).

## Loop health
- **Cadence:** launchd, driver ran **UNPINNED** this fire (`MISSION_WORKDIR` unset) — code provenance
  is the main checkout's working tree, not a pin. Reported to controlplane; see iter-284 STATUS.
- **Routing:** designer `fable` (only authoring-capable lane, `D-31`(a)) · planner lane derives
  `opus fail-closed:planner-lane-field-missing` · executor `codex:gpt-5.6-sol` · evaluator `sonnet`.
  Designer + evaluator both spawned this iteration; planner/executor **not reached** (see STATUS).
- **Running skill == origin** (`cmp` rc=0 through the resolved `readlink` target).

## Parked on Mark
- **`D-41` (the ONLY open row; 41 total, 40 resolved)** — may an ACTIVE prompt version be edited in
  place, or must a content change bump the version? Bears on eval-baseline reproducibility.
- **SonarCloud has no token on this rig** (all three env names UNSET), so the standing
  `new_coverage` + D-security-rating reds cannot be durably triaged by the loop. iter-283 triaged
  all 13 vulns first-party: 12 not defects, 1 real and filed.

## Quota / spend
- Iteration 284 metered **$0.17** of $5 (two quorum rounds). Quota buckets: opus, fable, sonnet.
