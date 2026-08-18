# Mission profile env files — SOURCE COPIES, not the live config

`mission-control.sh` sources `~/.config/ailang/mission-<name>.env`, never this
directory. These are versioned copies so the routing that governs three unattended
loops is reviewable in git. **Editing a file here changes nothing on the rig** — copy
it to `~/.config/ailang/` to deploy. Same shape as the launchd plists next door.

## Why these exist (2026-08-18)

The live files were unversioned entirely, so every value below — model routing, the
`PATH` fix, `MISSION_WORKDIR` — was invisible to review and to CI. That is the same
class as the motoko plist, which also lived only in `~/Library/LaunchAgents`.

It cost real diagnosis time. `MISSION_WORKDIR` was a **bare assignment** here, which
violated these files' own stated `${VAR:-value}` convention: `pin-root.sh` exports
`MISSION_WORKDIR=<pinned worktree>` and re-execs, the driver sets `REPO` from it
correctly, and then this file was re-sourced and clobbered the variable back to the
source clone. Behaviour stayed right, but the driver's `driver pin:` log line then
named the **stale clone** as the workdir while the loop actually ran the pinned one —
reporting the opposite of the truth, and reading as 60+ commits of drift that was not
happening. Fixed by restoring the `${VAR:-...}` form.

## Convention (load-bearing, not style)

Every entry uses `${VAR:-value}` so a command-line pin and an exported value from
`pin-root.sh` both still win. A bare assignment silently overrides its own caller.

## Verifying the live files match these

```bash
for m in v1 motoko world; do diff -q ~/.config/ailang/mission-$m.env tools/launchd/mission-env/mission-$m.env; done
```
