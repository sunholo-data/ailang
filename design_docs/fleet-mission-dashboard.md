# Fleet Mission Dashboard

Snapshot, overwritten every iteration (Gate 4). History: [fleet-mission.md](fleet-mission.md) STATUS + [fleet-mission-log.md](fleet-mission-log.md).

**As of iteration 4 — 2026-09-27**

- **Latest release**: v0.44.1. The fleet never releases.
- **Landed since iteration 3**: #1329 (`c912320fe`, the pi runner counts commits) and #1330 (`4d8dff929`, launchd suites run under an allowlisted env).
- **This iteration**: P0 #2 M1, a stall-watchdog CPU measurement tool (`tools/launchd/measure_stall_cpu.sh`). On the rig, macOS `ps -S` cannot see reaped-child CPU, so the tool reads `proc_pid_rusage`. Drill 62.5–97.7 CPU-s per window; wedges ≤1.72. Evaluator PASS 87.
- **Parked on Mark**:
  - D-FLEET-7: approve a descendants-only rusage arm at ≥10 CPU-s per 120 s (recommend yes). The live watchdog is unchanged until then.
- **Next pick**: P0 #4 `pi-runner:sandbox-extensions-not-wired` (D-FLEET-6 ruled, plan landed); then P0 #3's pre-dirty half; then P1 0a/0b.
- **Open tickets**: 17. Mark's attended triage order is in the charter Queue.
- **Dev CI note**: `launchd drivers (bash 3.2)` flaked once on dev (`test_driver_notify.sh` wall-clock limit of 7 s, read 8 s); the rerun was green. Ticket `ci:launchd-driver-suite-flakes` (P2 #11).
- **Loop**: `dev.ailang.mission-fleet`, every 6h, idles free with 0 tickets. Runs pinned `origin/dev`.
- **Routing**: controller claude-opus-5-5 · executor codex:gpt-6-sol (3 rounds, ≈172k tok) · evaluator sonnet (Agent tool, ≈177k tok) · designer and planner skipped.
- **Quota posture**: codex and Anthropic subscription buckets only. Metered $0.
- **Known caveats**: the codex sandbox cannot run `ps`-based suites, so the controller re-runs the done-gate outside it. The legacy `scripts/test_mission_pi_run.sh` TEST 3 is red at base (D-FLEET-5 now allows the fleet to fix it).
