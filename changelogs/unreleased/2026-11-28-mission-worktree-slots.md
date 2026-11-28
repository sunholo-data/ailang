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
- Slot reset (M2): an existing slot is reset in place (`checkout --detach --force` + `clean -ffdx`)
  inside the bounded writer, then proven at the SHA with an empty status (else
  `failed:not_clean_after_reset` and the slot is removed). A dirty slot is first salvaged to
  `~/.ailang/state/slots/<slot>/salvage-<ts>/`: staged and unstaged diffs kept separately, a byte
  copy of the index taken before any git command, a tar of untracked files, and a
  `refs/salvage/slots/<slot>/<ts>` ref for a detached HEAD that no branch or remote contains.
  Restore: `git apply --index staged.patch && git apply unstaged.patch && tar -xf untracked.tar`
  in a tree at the INFO `head` (TST-SLOT-02, 03, 08).
