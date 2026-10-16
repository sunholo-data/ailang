### Fixed — mission-worktree.sh reports a dead writer

- Detach background worktree writers with capability-probed Perl `POSIX::setsid`,
  falling back to bash job control when Perl is unavailable.
- Publish writer identity before returning a path. `wait` now returns
  `failed:writer_died` (exit 1) for an abandoned checkout, quiesces its owned checkout
  group, and removes the partial worktree and branch. Live writers remain pending.
- Re-read terminal status before recovery so a concurrent completion is preserved;
  keep legacy pending state and inconclusive process inspection conservative.
