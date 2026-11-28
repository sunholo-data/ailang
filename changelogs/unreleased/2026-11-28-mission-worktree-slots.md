### Added — mission worktree slots (M-MISSION-WORKTREE-SLOTS)

- `tools/launchd/mission-worktree.sh acquire MISSION ROLE SHA [BRANCH]` / `held FIRE_ID ROLE` /
  `release-fire FIRE_ID`: reusable, lock-held worktree slots at fixed paths
  (`~/.ailang-slots/<repo>/<mission>/<role>-<n>`), so mission checkouts can be incremental and Go
  build-cache keys stay stable. A slot is owned by the fire (the driver PID in
  `MISSION_FIRE_ID`, checked against its start time), not by the short-lived call that acquired
  it; taking or stealing a lock happens inside a per-mission mutex. Slot roots under `/tmp`, any
  sweep-protected prefix or `~/dev/sunholo-data` (the weekly checkout-audit's delete scope) are
  refused. New logic lives in `tools/launchd/lib/mission-slot.sh`; tests in
  `tools/launchd/test_mission_slots.sh` (TST-SLOT-01, 04, 05, 07, 09, 12, 13, 14).
  `add` / `wait` are unchanged (the existing 194-case suite passes unmodified).
