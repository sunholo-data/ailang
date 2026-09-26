# Fleet Mission Dashboard

Snapshot, overwritten every iteration (Gate 4). History: [fleet-mission.md](fleet-mission.md) STATUS + [fleet-mission-log.md](fleet-mission-log.md).

**As of iteration 1 — 2026-09-26**

- **Latest release**: v0.44.1 (1f24fe297). The fleet never releases.
- **Last landed**: `driver:slot-kill-leaves-orphan-descendants` (#1325, `e3dadcd07`). Slot kills now reap the controller's whole process tree. Evaluator 87.
- **Next pick**: P0 #2 `stall-watchdog:kills-controller-on-long-drill`, mechanical half only. The 600s threshold and sample counts are policy and park.
- **Then**: P0 #3 `pi-runner:verdict-blind-to-commits-and-predirty`, P0 #4 `pi-runner:sandbox-extensions-not-wired`.
- **Open tickets**: 15 (was 16). Order is Mark's attended triage in the charter Queue.
- **Loop**: `dev.ailang.mission-fleet`, every 6h, idles free when there are 0 tickets. Runs pinned `origin/dev`.
- **Routing**: controller claude-opus-5-5 · planner and executor codex:gpt-6-sol · evaluator sonnet (Agent tool) · designer rotation (skipped for mechanical fixes).
- **Parked on Mark**: 2 policy tickets (HD-2a): `resolver:planner-lane-field-missing-vs-spawn-pin` and `spawn-pin-hook:no-fallback-mode`. Both carry recommendations in the charter Queue.
- **Quota posture**: iteration 1 used the codex (≈203k tok) and Anthropic subscription buckets. Metered $0.
- **Known caveat**: when a controller runs `make test-launchd-drivers` it inherits the driver env (MISSION_*, GIT_CONFIG_*), and the suite reads red at base. Run it under `env -i`.
