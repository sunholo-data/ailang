# Fleet iteration 67 — done-gate and root-cause evidence

Ticket `pin-root:worktree-removed-mid-fire` (stapledon iteration 55, filed 2026-10-28T15:27Z).
Branch `fleet/i67-sweep-account-home`, PR #1999. Base `4c3dc018261fd263cba18cce8f001a2bb61c6377`; head `d144c85bd2defb3bb1c90f84adab858bf3a6c776`.

## Root cause (first-party)

- Fleet iteration 66 ran its done-gate dry-run under `HOME=/tmp/fleet_i66_home.3S7K` at 2026-10-28T14:50:33Z. The command is in its transcript, and the charter's Guardrails prescribed it.
- That synthetic HOME's `.ailang/state/` holds `worktree-sweep.stamp` (15:50 CET) and `worktree-sweep.log`. The log is copied verbatim in [synthetic-home-worktree-sweep.log](synthetic-home-worktree-sweep.log). It records the removal of the real `~/.ailang-driver-pin/stapledon` (15:52:00 CET = 14:52Z), `triage` (15:52:08) and `world` (15:52:24), plus one landed local branch, `fix/agent-prompts-off-argv` (`cf73e7d82`).
- The real `~/.ailang/state/worktree-sweep.log` has no line for any of them, because the sweep's `LOG` is `$HOME`-relative.
- The mechanism has four links:
  1. The dry-run's claude lane probes run the SessionStart hook `scripts/hooks/git_health.sh`.
  2. `env -i` drops `AILANG_MISSION_STAGE` and `AILANG_TASK_ID`, so the hook does not exit early.
  3. The stamp is absent under the synthetic HOME, so the hook launches `worktree_sweep.sh --apply`.
  4. `PROTECT="$HOME/.ailang-driver-pin/:…"` points into the synthetic HOME, so the real pins were judged "clean, idle >2h, on origin/dev, no cwd inside" and read REMOVE.
- Stapledon's controller had its cwd in `stapledons-godot`, not in the pin, so the sweep's `lsof` check could not see the pin as busy.
- Ruled out by transcript scan (all claude, codex and pi sessions, 11:30–15:30Z): no controller or attended session removed a pin by name; the attended 15:06Z disk cleanup touched `~/dev/sunholo-data/stapledons-godot-*` only. Ruled out by code read: `pin-root.sh` never removes a worktree, and `test_pin_root.sh` isolates both `HOME` and `AILANG_DRIVER_PIN_DIR`.
- `perl -e 'print((getpwuid($<))[7])'` prints `/Users/voightkampff` even when `HOME=/tmp/x`.

## Gates (controller, outside the sandbox)

| Gate | Base `4c3dc0182` | Head `d144c85bd` |
|---|---|---|
| `/bin/bash scripts/hooks/test_git_health.sh` | 9 passed, rc 0 | 12 passed, rc 0 |
| `bash -n` + `shellcheck -S warning` (hook + test) | rc 0 | rc 0 |
| `make check-changelog` | rc 0 | rc 0 |
| `make test-launchd-drivers` | rc 0, 49 suites | rc 0, 49 suites |
| `rg -c 'AILANG_WORKTREE_SWEEP=0' design_docs/fleet-mission.md` | 0 | 2 |

Real-state observations (sha256 prefix of `~/.ailang/state/worktree-sweep.log`, the sweep stamp's mtime, and a hash of the `~/.ailang-driver-pin` listing) were unchanged across every gate and dry-run. Example: log `882378d5df7299e8`, 171 lines, stamp `1793221572`.

## Dry-runs (world, idle and armed; real HOME; `AILANG_WORKTREE_SWEEP=0 AILANG_DRIVER_PINNED=d144c85bd…`)

- Healthy: rc 0, `DRY RUN ok: mission=world`, `lanes=ok`.
- Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i67`): rc 0, `lanes=DEGRADED(1)`, and the executor was handed to `claude:claude-haiku-5-5` (probe rc 1).
- The world pidfile was absent before and after both arms.

## Mutation drills

- Executor (disposable copies): A `HOME_PREFIXES` reddened arms A and B (rc 1); B `DROP_CALLER_PROTECT` reddened arm B (rc 1); C `LOOKUP_FAIL_OPEN` reddened arm C (`0:3` vs `0:2`, rc 1).
- Judge: the same three drills reproduced independently. A sweep dry-run (no `--apply`) under a synthetic HOME printed REMOVE for real `.wt-fleet-iter66*` worktrees, and KEEP protected with the hook's protect string.
