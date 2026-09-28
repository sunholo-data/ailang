# Fleet Mission Dashboard

Snapshot, overwritten every iteration (Gate 4). History: [fleet-mission.md](fleet-mission.md) STATUS + [fleet-mission-log.md](fleet-mission-log.md).

**As of iteration 6 — 2026-09-28**

- **Latest release**: v0.47.1. The fleet never releases.
- **Landed since iteration 4**: #1336 (`d9e1211d0`, the stall-watchdog CPU measurement tool).
- **Iteration 5** (07:56 fire) built the cron-kicker session-lost fix and the pi sandbox wiring, then crashed at Gate 3b on an API DNS error with no record. The 23:09 fire before it died in the Aqua-session loss.
- **This iteration**: re-judged both fixes with fresh sonnet evaluators (kicker **PASS 92**, sandbox **PASS 96**, 0 blocking) and combined them in **#1377**, because every single-fix PR conflicted on the changelog within minutes.
- **Blocked on**: required `lint` is red on dev itself since `1fcc479f1` (a direct push; gofmt on `internal/executor/motoko/healthcheck.go`, one line). That is outside fleet scope and was sent to `mission-v1`. When dev lint is green: push to #1377 → green → merge → resolve both tickets.
- **Parked on Mark**:
  - D-FLEET-7: approve a descendants-only rusage arm at ≥10 CPU-s per 120 s (recommend yes).
- **Next pick**: land #1377; then P0 #3's pre-dirty half; then P1 0a/0b (ruled D-FLEET-1/2).
- **Open tickets**: 21, including the `blocking=all` aqua ticket (fix in #1377). Mark's triage order is in the charter Queue.
- **Dev CI note**: `launchd drivers (bash 3.2)` red on `0c41e3185` and `666c9d4e7` in `tools/eval/test_motoko_connection_probe.sh` (flake class, P2 #11; `tools/eval` is outside fleet scope).
- **Loop**: `dev.ailang.mission-fleet`, every 6h, idles free with 0 tickets. Runs pinned `origin/dev`.
- **Routing**: controller claude-opus-5-5 · evaluator sonnet ×2 (Agent tool, ≈265k tok) · designer, planner and executor not needed (iteration 5's work). Codex was over its daily ration this fire (probe rc 75).
- **Quota posture**: Anthropic subscription buckets only this fire. Metered $0.
- **Known caveats**: the kicker runs from the MAIN checkout via crontab, so its fix reaches the rig only when that checkout's `dev` is updated. It is currently detached with a merge in progress, which needs a human.
