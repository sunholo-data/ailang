# Mission Dashboard — V1

_Snapshot, overwritten every iteration. History: `v1-mission.md` (STATUS) + `v1-mission-log.md`._

**Last iteration:** 354 · 2026-09-14 · HARNESS · LANDED · **Goal distance:** N=12 docs before v1.0.0 (±0)
**Latest release:** v0.38.5 (2026-09-13). Second fire since the six-day kill switch.

## Just landed
`m-pin-drift-blind-under-sha-pin` — PR #1166 → `266cf23a2`, 22 checks zero not-green. `PIN_DRIFT` read 0 under
any ancestor SHA pin (the 09-07 pin was 266 behind `origin/dev`); the driver now also reports **`PIN_AGE`**
(target vs an origin/dev SHA captured once), own state file + threshold 25 + dedupe-until-doubling + `pin-age`
notice, old-helper handshake. Pin suite 81→117, notify 50→82, 22-mutant audit. Judge `sonnet` r1 PASS 96 / r2 PASS 100.
Also: dev's REQUIRED `test` context went red mid-iteration from a concurrent commit (`7423434b4`,
`pi-or-minimax-m3` without a `ceilingLimited` entry) — fixed forward first as #1165 → `fd44be9ee`.
**Watch next fire:** the driver log should print `driver pin age: 0 below warning threshold 25`.

## Next picks
1. `m-weekly-sweep-orphans-2026-09-14` — 7 orphans of 85 (triage-lite; 4 daneel issues already have docs).
2. `m-approval-poll-production-defaults-unexercised` · `m-ratelimit-window-default-unpinned` — small, judge-measured.
3. `m-gate0-self-crash-notice-read` **[world-DEMAND]** — Gate 0 is blind to the driver's own rc=143 notices (real on V1: #1072).
4. `m-sonar-dev-branch-security-rating-c-on-new-code` — standing SonarCloud branch red.
5. `m-codex-quota-admission-test-reads-live-ledger` — rig-only red; the suite reads the live quota ledger.

## Loop health
- **pi executor 30-min cap < plan-sized milestone, 3rd instance** (347 M3, 354 M1, 354 M2): both fallback links
  spent, end-of-chain opus spawn is hook-denied → controller finished the arms. Gate-5 skill edit: cap is now a
  per-run `--max-seconds`, 3600 for milestones the plan sizes >150 LOC. deepseek promotion count reset (rc=13).
- `scripts/mission_pi_run.sh` does NOT load the two `-e` sandbox extensions the recipe calls mandatory (unfixed, 1 reading).
- All three pi lanes: first probe silent past 110 s, immediate retry rc=0 (cold start) — three readings today.
- Skill copies: `gate-1-observe.md` differs — origin has iter-353's rule, the main checkout does not (worktree-committed edit).
- Ledger 63 rows, **ZERO open**. No directives on #1163 / #1072.

## Routing / cost
designer `codex:gpt-6-astra` (1 doc + 1 revision; sol substituted into the quorum) · quorum r1/r2 blocked → carve-out r3 ·
planner `pi:kimi-k3` ok (4th consecutive) · executor `pi:deepseek` wall_timeout → `pi:openrouter/deepseek` wall_timeout →
controller · evaluator `sonnet` ×2. Metered **$0.29** of $5.

## Parked on Mark
none.
