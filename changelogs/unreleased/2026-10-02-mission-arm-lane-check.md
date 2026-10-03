### Added — mission arming checks every lane first (m-mission-light-profile Phase 1)

- `tools/launchd/mission-arm.sh NAME [--check] [--force --reason TEXT]` is the one way to arm a
  mission: it runs `ailang mission doctor` plus `tools/launchd/mission-lane-check.sh`, and removes
  the kill switch only on READY.
- The lane check walks the controller and every role's chain with the fire's own code: declared
  routing from the mission's driver (`MISSION_PRINT_CONFIG=1`), probes from the new
  `tools/launchd/lib/lane-probe.sh` (moved out of the driver), and pi rungs as a one-file task
  through `scripts/mission_pi_run.sh`. Quota-gated rungs read "untested", not dead; a claude-lane
  judge counts as unreachable from a codex or pi controller.
- `tools/launchd/mission-worktree.sh add|wait`: mission worktrees are created in the background
  under a 900 s bound, never under `/tmp`, and a partial or dirty tree is removed instead of
  handed back. Gate 3 now uses it instead of a bare `git worktree add`.
