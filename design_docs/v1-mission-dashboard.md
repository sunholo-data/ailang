# Mission Dashboard — V1

> Snapshot only, overwritten every iteration. History lives in `v1-mission.md` (queue + STATUS)
> and `v1-mission-log.md` (full records). The bare `mission-dashboard.md` here is **Motoko's**.

**Updated**: 2026-09-08 ~00:00 UTC (iteration 349) · **Release**: v0.35.2 (attended, 2026-09-07)

## ⚠ Read this first
**The whole fleet has been running frozen driver code since 2026-09-07 15:19.** All four missions
carry `AILANG_DRIVER_REF=48c4a6e49…` (attended deployment pin); `origin/dev` is **43 commits**
ahead, **3 touching `tools/launchd/`** — including `e5a325a20`, M1/M2 of the very sprint whose M3
just merged. **This landing is inert on the rig until the pin moves.** `PIN_DRIFT` cannot report it
— it measures the clone against *the ref*, so a SHA pin reads `0` forever, and this fire logged
`driver pin drift: 0` with 43 commits of drift. → **D-61**.

## Just landed
- **iter-349** `m-launchd-drain-aggregate-budget` (M3) — PR #1107 → [`b5513ccdf`](https://github.com/sunholo-data/ailang/commit/b5513ccdfb7b9e014c197e9021d27be5272ebbe1), judge
  **PASS 88/100, zero blocking**. `MISSION_DRAIN_BUDGET` (default 90 s) caps the WHOLE notice drain in the one
  phase of a fire with no deadline behind it but the 6-hour `HARD_TIMEOUT`; deferred rows are
  re-spooled unchanged, so an outage costs a delay, not the record.
- **Attempt 1 of this slot died at Gate 3b** holding that PR green and unmerged, with no record at
  all. Attempt 2 verified it independently rather than adopting it, then landed it.

## Next three
1. `m-fleet-sha-pin-freezes-every-driver-fix` — the env-pin half is blocked on D-61; making
   `pin-root.sh` report drift against `origin/dev` under a SHA pin is not blocked.
2. `m-sonar-dev-branch-security-rating-c-on-new-code` — `dev` quality gate red on C Security Rating
   since `8e3927950`, deferred three iterations. A GitHub App, so no workflow name surfaces it.
3. `m-debugcacheforms-flaky-on-macos-ci` — structural assertions, not a third `t.Skip`.

## Loop health
- 345–349 all landed, but **349 needed two fires** (attempt 1 stall-killed at Gate 3b, `rc=143`).
- Routing: controller `claude:claude-opus-5` · evaluator `agent-tool sonnet` in its own worktree.
  Designer/planner/executor **not spawned** — verify-and-land of an already-quorumed sprint.
- `codex:gpt-5.6-sol` **ration-blocked before the probe for six consecutive fires**; planner and
  executor run on pi fallbacks every time.
- V1's lane-degradation notices have **never delivered**: 3 rows stuck since 2026-09-07T00:37Z,
  re-failing on six fires, while `mission-docs` delivers the identical notice.
- Metered **$0.00** of the $5 ceiling.

## Waiting on Mark
**D-61** — may this loop return `AILANG_DRIVER_REF` to the `origin/dev` default itself once a
SHA-pinned deployment's fix has merged (never pin forward, never pick a SHA)? Ledger 61 rows, 1 open.
