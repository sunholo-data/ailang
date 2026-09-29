### Added — worktrees clean themselves up; SessionStart flags a stuck rebase or diverged `dev` (2026-09-29)

Nothing ever removed a worktree. On 2026-09-28 this clone held 30 worktrees and 33 branches, and every
non-infrastructure one removed that day was a leftover whose PR had already merged. The pile hid the two
commits that really were stranded.

- **`scripts/worktree_sweep.sh`** (`make worktree-sweep` dry run, `make worktree-sweep-apply`) removes a
  worktree only when it is landed (HEAD on origin/dev, or a PR merged at exactly HEAD, which catches squash
  merges `git cherry` misses), clean, idle for 2h or more, and has no live process inside. It never touches
  mission driver pins, the A/B and nightly checkouts, or coordinator task worktrees. If `lsof` gives no answer,
  it removes nothing. Landed local branches with no worktree go too; every removal is logged with its SHA to
  `~/.ailang/state/worktree-sweep.log`.
- **`scripts/hooks/git_health.sh`** (SessionStart, 34 ms, silent when healthy) reports a rebase, merge,
  cherry-pick or bisect in progress in the main checkout or the current worktree, and a `dev` that is both
  ahead and behind. At most every 6h it runs the sweep with `--apply` in the background. It is silent for
  mission stages and coordinator tasks; `AILANG_WORKTREE_SWEEP=0` turns the sweep off.
- `.claude/rules/dev-workflow.md`: attended work goes straight to `dev`, and worktrees are for unattended or
  long work.

Self-tests: `test_worktree_sweep.sh` (15 arms) and `test_git_health.sh` (9), both run in `test-launchd-drivers`
under bash 3.2, with the CI shell lane widened to the sweeper's paths. The busy, idle, dirty, protected and
`lsof`-failure guards are each mutation-checked.
