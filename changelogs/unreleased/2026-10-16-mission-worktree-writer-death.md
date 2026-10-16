### Fixed — mission-worktree.sh reports a dead writer

- Detach background worktree writers with capability-probed Perl `POSIX::setsid`,
  falling back to bash job control when Perl is unavailable.
- Publish writer identity before returning a path. `wait` now returns
  `failed:writer_died` (exit 1) for an abandoned checkout, quiesces its owned checkout
  group, and removes the partial worktree and branch. Live writers remain pending.
- Re-read terminal status before recovery so a concurrent completion is preserved;
  keep inconclusive process inspection conservative.
- Terminalize registration failures as `failed:owner_never_published` (exit 1).
  Without an owner record, a dead `.started` writer returns `failed:writer_died`;
  a live registered writer remains pending. Bare pending state expires after 60
  seconds (override with `MISSION_WORKTREE_OWNER_GRACE` for tests). Ownerless
  recovery consumes state only, preserving unrelated paths and branches.
- Cover caller death before owner publication and writer registration, legacy
  pending expiry, registration failures, and a changed checkout process group.
  Retain the PGID re-check: PID/birth/token identity alone cannot prove group
  membership, so cleanup must refuse a mismatched group before signaling it.
