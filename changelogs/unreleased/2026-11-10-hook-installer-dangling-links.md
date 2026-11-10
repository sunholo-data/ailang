### Fixed — `make install-hooks` repairs a dangling `pre-push` link and never links into a worktree (2026-11-10)

The main checkout's `pre-push` hook had pointed into a removed worktree
(`.claude/worktrees/fleet-rulings`) since 2026-09-27. Git skips a dangling hook without a word, and
`scripts/hooks/install_hooks.sh` refused to replace it because it was "not ours", so the stdlib and
ci-quick push gates were off for six weeks while every SessionStart re-ran the installer.

- The installer now replaces a hook link that is dangling or points into a worktree
  (`.claude/worktrees/`, `.git/worktrees/`), printing old → new. Regular files and live links
  elsewhere are still refused.
- It always links the **main** checkout's copy. When the main checkout has no
  `scripts/hooks/pre-push` it refuses, instead of falling back to the worktree's copy — the fallback
  that made the 09-27 link.
- New `install_hooks.sh --check` (read-only). `git_health.sh` runs it at SessionStart and prints one
  line when a managed hook is missing, dangling, linked into a worktree or not ours.
- Self-test `scripts/hooks/test_install_hooks.sh` (27 arms, bash 3.2) joins
  `make test-launchd-drivers`; `test_git_health.sh` gains a planted-dangling-hook arm.

Design: `design_docs/planned/v1_0_0/m-v1-release-gates.md` §5 (M1). The main checkout's hook is
repaired by running `make install-hooks` there after this lands.
