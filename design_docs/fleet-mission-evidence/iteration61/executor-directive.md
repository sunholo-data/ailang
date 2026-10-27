MISSION-ROLE: executor

You are the SPRINT-EXECUTOR role for fleet mission iteration 61 (unattended mission stage; do NOT run `ailang messages`; do NOT push; do NOT create PRs; do NOT touch any directory other than the worktree below and /tmp/fleet_i61/).

Worktree (yours): /Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter61-zsh-land — detached at origin/dev = c8e348d08736e6b273efb73175ae706ce82eef62, clean.
Planner addendum to follow: /tmp/fleet_i61/planner-addendum-i61.md (section B steps 1–3; read it first).

Do exactly these steps, in the foreground, from the worktree, and capture every command with its output into /tmp/fleet_i61/executor-report-i61.md:
1. `git checkout -b fleet/i61-zsh-libs-land` (inside the worktree; this is a new local branch, fine).
2. `git cherry-pick bd9a41cf3 dcab63a3c` — must be clean. If it conflicts, run `git cherry-pick --abort`, write `EXECUTOR VERDICT: BLOCKED — conflict` and stop.
3. Identity checks (run under /bin/bash -c '…'):
   - `git range-diff bf22f4943..dcab63a3c origin/dev..HEAD` (both lines must show `=`).
   - `git diff --name-only origin/dev...HEAD` (must be exactly the 7 paths listed in the addendum).
   - per-file blob compare for all 7 paths: `a=$(git rev-parse dcab63a3c:"$f"); b=$(git rev-parse HEAD:"$f")` and print IDENTICAL/DIFF per file.
4. Focused tests, outside any sandbox, under /bin/bash:
   `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_quorum_artifacts.sh; echo rc=$?`
   `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_pi_ext_args.sh; echo rc=$?`
   Record the pass/fail counts and confirm the zsh arms ran (no `SKIP (no /bin/zsh)` lines). If suite-env.sh does not take a script argument that way, read tools/launchd/lib/suite-env.sh and the plan for the exact invocation and record what you used.
5. `make check-changelog; echo rc=$?` and `git diff --check origin/dev...HEAD; echo rc=$?`.
6. Record `git rev-parse HEAD` (full 40 chars) and `git log --oneline origin/dev..HEAD`.

The report's first line must be `EXECUTOR VERDICT: PASS` (all of steps 2–5 green), or `EXECUTOR VERDICT: FAIL — <step and reason>`. Do not edit any source file. Do not background any command, and do not end your turn announcing a wait; nothing will re-invoke you.
