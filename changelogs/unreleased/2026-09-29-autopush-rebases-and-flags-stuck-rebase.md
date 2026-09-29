### Changed — Stop-hook auto-push now rebases a diverged `dev`, and a stuck rebase is loud (2026-09-29)

`scripts/hooks/push_dev_on_stop.sh` used to refuse when local `dev` was both ahead of and behind
origin, which stranded work: on 2026-09-28 two commits sat unpushed, and a hand-run `pull --rebase`
then stopped on the changelog and left the shared checkout mid-rebase overnight. The hook now:

- **rebases and pushes** when it is safe: no uncommitted edit touches a file the rebase would
  change (a sibling session's unsaved work is never stashed or rewritten), and there is no conflict;
- **aborts on any conflict or failure** and verifies the abort, so it never leaves a half-done rebase.
  A textual conflict still goes to a human, naming the file;
- **reports a git operation in progress** (rebase, merge, cherry-pick, bisect) on stdout with the
  command to finish or abort it. Mid-rebase HEAD is detached, and the old check order exited as
  "not on dev" before it ever looked, so the overnight stuck rebase was never reported;
- takes a per-checkout lock (stale after 10 min), so two sessions stopping together cannot interleave.

`AILANG_AUTOREBASE=0` restores the old refuse-only behaviour. `test_push_dev_on_stop.sh` gains the
C2–C5 and E2 arms (40 in all). Mutation-checked: removing the overlap guard, the abort, or the
rebase-dir detection each fails its arm.
