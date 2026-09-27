# Fleet Mission Dashboard

Snapshot, overwritten every iteration (Gate 4). History: [fleet-mission.md](fleet-mission.md) STATUS + [fleet-mission-log.md](fleet-mission-log.md).

**As of iteration 2 — 2026-09-27**

- **Latest release**: v0.44.1. The fleet never releases.
- **Last landed**: `driver:slot-kill-leaves-orphan-descendants` (#1325, `e3dadcd07`, iteration 1).
- **Built, not merged**: #1329, where the pi runner counts executor commits as work (commit-blind half of P0 #3). Evaluator PASS 97. Blocked by the required `test` check, which is red on dev for `internal/iface/builder.go` at 801 lines (V1's to fix; handed over). Merge once dev CI is green; the ticket stays open for its pre-dirty half (D-58 design).
- **Parked on Mark**: D-FLEET-4, a stall-watchdog CPU arm that needs a new threshold (recommend measure first). D-FLEET-5, widening the allowlist to `scripts/test_mission_*` (recommend yes).
- **Next pick**: merge #1329 if dev is green; then P0 #4 `pi-runner:sandbox-extensions-not-wired` (design; D-FLEET-3 means Mark approves before it lands); then P1 #0 `tests:driver-env-leaks-into-launchd-suite`.
- **Open tickets**: 16 (new: `pi-runner:quota-429-reported-as-empty-worktree`). Order is Mark's attended triage in the charter Queue.
- **Loop**: `dev.ailang.mission-fleet`, every 6h, idles free with 0 tickets. Runs pinned `origin/dev`.
- **Routing**: controller claude-opus-5-5 · planner and executor codex:gpt-6-sol · evaluator sonnet (Agent tool) · designer skipped.
- **Quota posture**: iteration 2 used codex (≈173k tok over 3 runs) and the Anthropic subscription (sonnet ≈284k). Metered $0.
- **Known caveats**: run `make test-launchd-drivers` under `env -i` (P1 #0 fixes this). Legacy `scripts/test_mission_pi_run.sh` TEST 3 is red at base (D-FLEET-5).
