# V1 Mission Dashboard — snapshot 2026-09-07, iteration 347
**Release**: v0.35.1 · **Goal**: N=12 design docs before v1.0.0 (unmoved this iteration — HARNESS item)

## In flight / just landed
- **LANDED (2 of 4 milestones)** `m-launchd-notify-subshell-observation` — PR #1090. Human ruling
  **D-60 option B**. The three driver notification paths are bounded by `_mc_bounded` at
  `NOTIFY_TIMEOUT` (30s); the seven inherited notify assertions are restored — `test_driver_notify.sh`
  **20/7 → 38 passed, 0 failed**. Judge `sonnet` **LAND 80/100**; both BLOCKING findings reproduced
  first-party and fixed in-iteration.
- **M3/M4 NOT delivered** — executor hit its 30-min cap; partial banked at
  `~/.ailang/state/mission-v1-iter347-m3-partial/`, queued as `m-launchd-drain-aggregate-budget`.

## Next three picks
1. `m-launchd-heartbeat-arms-red-behind-the-notify-red` — **a RED on `dev`.** With the notify red
   gone, `make test-launchd-drivers` reaches three more suites: two pass (84/0, 17/0, never run on
   `dev` before), `test_mission_heartbeat.sh` fails **9 arms**. Proven pre-existing by a paired
   base/head control — the job stays red for a different reason than before.
2. `m-launchd-drain-aggregate-budget` — the drain bounds each row, not the whole drain, in the
   preflight phase whose only backstop is the 6h `HARD_TIMEOUT`; the spool is uncapped.
3. `m-debugcacheforms-flaky-on-macos-ci` — iteration 346's own macOS flake.

## Loop / routing
2h interval + overlap guard. Designer = rotation (deepseek ran all 3 passes). Planner pin
`pi:ollama/kimi-k3:cloud` **probe-timed-out at 120s** → `pi:openrouter/moonshotai/kimi-k3`.
Executor `pi:ollama/deepseek-v4-flash:0731-cloud`. Evaluator `sonnet`. Generator != judge held.

## Parked on Mark
**Nothing.** Decision ledger: 60 rows, **ZERO open** (`mission_decisions.sh --check` valid).
D-55–D-60 were all resolved by attended ruling on 2026-09-07.

## Quota / cost posture
Metered **$0.80** of $5 — quorum $0.177 (3 rounds), OpenRouter planner $0.620. Designer (3 runs) and
executor flat-rate ollama-cloud at $0; controller and evaluator on Anthropic subscription.

## Watch
- Dropped `pull_request` synchronize: the 2nd push to #1090 created **1** check, not 20. Recovered
  with `gh workflow run CI --ref <branch>`.
- The narrow-refinement carve-out authorises a 2nd designer revision the Fable diet forbids. $0 here
  (flat-rate lane); evidence row 1 for a routing-policy fix.
