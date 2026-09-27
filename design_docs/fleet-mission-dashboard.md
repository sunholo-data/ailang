# Fleet Mission Dashboard

Snapshot, overwritten every iteration (Gate 4). History: [fleet-mission.md](fleet-mission.md) STATUS + [fleet-mission-log.md](fleet-mission-log.md).

**As of iteration 3 — 2026-09-27**

- **Latest release**: v0.44.1. The fleet never releases.
- **Last landed**: `driver:slot-kill-leaves-orphan-descendants` (#1325, `e3dadcd07`, iteration 1).
- **Built, not merged, both PASS 97**:
  - #1329: the pi runner counts commits as work.
  - #1330: every `test-launchd-drivers` suite runs under an allowlisted env, so the driver's exports no longer redden the done-gate; this PR also carries the P0 #4 plan.
  - Both are blocked by the required `test` check, which is red on dev for `internal/iface/builder.go` at 801 lines (V1's to fix; handed over in iteration 2).
- **Parked on Mark**:
  - D-FLEET-4: a stall-watchdog CPU arm; recommend measuring first.
  - D-FLEET-5: widen the allowlist to `scripts/test_mission_*`; recommend yes.
  - D-FLEET-6: land the pi sandbox fix, including `tools/pi-extensions/sandbox/index.ts`, which fails open today; recommend yes.
- **Next pick**: merge #1329 and #1330 once dev is green; then P1 0a `resolver:planner-lane-field-missing-vs-spawn-pin` (RULED D-FLEET-1).
- **Open tickets**: 16 (none new). Order is Mark's attended triage in the charter Queue.
- **Loop**: `dev.ailang.mission-fleet`, every 6h, idles free with 0 tickets. Runs pinned `origin/dev`.
- **Routing**: controller claude-opus-5-5 · planner and executor codex:gpt-6-sol · evaluator sonnet (Agent tool) · designer skipped.
- **Quota posture**: iteration 3 used codex (≈153k tok over 2 runs) and the Anthropic subscription (sonnet ≈122k). Metered $0.
- **Known caveats**: until #1330 merges, run `make test-launchd-drivers` under `env -i`. The legacy `scripts/test_mission_pi_run.sh` TEST 3 is red at base (D-FLEET-5). The codex sandbox cannot run `test_mission_stall.sh`, so the controller always re-runs the done-gate outside it.
