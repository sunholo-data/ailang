# Unattended runs (coordinator tasks, mission fires, `claude -p`)

When no human is in the session to answer, every "pause", "ask the user" or
"wait for approval" step in this skill becomes **commit and continue**:

- **Commit after each milestone** on the work branch. Uncommitted work is lost work:
  include the updated sprint JSON/checkpoint in each milestone commit. Never wait for
  permission to commit. For cloud task publication, follow the sequence below.
- **A blocker on one milestone is not a stop.** Record it (sprint JSON `notes`, the plan's
  checkbox left open with a one-line reason), then carry on with every milestone that
  does not depend on it. A failure that already exists at the starting commit (a broken
  docs build, a pre-existing failing test) is recorded and worked around, not escalated.
- **Stop only when no remaining milestone can make progress**, and say exactly what
  blocks each one in the final report.
- Decisions the plan leaves open: take the option the design doc's decisions point to,
  write down why in the sprint JSON, and continue.

Measured 2026-10-08: Claude Haiku 5.5 replaying four merged sprint-executor tasks stopped
twice at these pause points — once holding a finished sprint uncommitted for permission,
once halting after M1 over a docs failure that predated the task — where the original
executor completed both. Haiku now sits on the executor fallback chain.

## Cloud tasks: push every milestone

Run `/bin/bash .claude/skills/sprint-executor/scripts/cloud_branch_sync.sh start`
**once at cloud session start, before selecting or executing a milestone**. Inspect
its verdict, then read the sprint JSON restored by a resume and continue from the
first unfinished milestone. Do not rerun `start` after creating local milestone commits.

Keep committing after each milestone, including its updated sprint JSON/checkpoint.
Immediately after **each** milestone commit, run
`/bin/bash .claude/skills/sprint-executor/scripts/cloud_branch_sync.sh push`.
The 2026-10-08 ticket `crash:sprint-executor:sigbus-ram-tmp-full-make-test`
showed that SIGBUS/timeout before the wrapper's final push loses unpublished work.
Shallow single-branch clones lack the task tracking ref, so the wrapper can miss
later commits unless the exact fetch refspec is installed before the first agent push.

The helper acts only when a nonempty `AILANG_TASK_ID` matches the current branch
exactly as `coordinator/<taskID>`; attended/mission, live/direct-push, wrong-task,
missing-ID and detached sessions skip. Both commands install the exact task fetch
refspec once, preserving the base mapping. `start` reports `fresh` for an absent
remote task branch or fetches and uses `reset --keep` to resume existing history,
retaining untracked wrapper files. `push` never forces and verifies the remote SHA.
Network Git is noninteractive and bounded to 30 seconds per call plus at most one
second termination grace (`CLOUD_BRANCH_SYNC_TIMEOUT_S` overrides the deadline).

Read the verdict, not just the exit code: rc 0 includes `failed:*`, as well as
`pushed`, `resumed`, `fresh` and `skipped:*`; only usage errors return 2. Each verdict
names its branch/SHA. Credentials may be absent or refused, particularly for shadow
children. Record every `failed:*` verdict with branch, SHA and reason in sprint JSON
notes, then continue independent work. For start failures, preserve the tree and
record the failure; do not improvise destructive recovery. Never force, install
credentials or expand the branch gate. The wrapper still attempts final publication;
a refused push is not durable merely because the helper returns rc 0.

## Verification resources

- **Never install toolchains.** Never install or build compilers, sysroots, C/CGO
  toolchains or system packages: no apt, brew, sudo, building cc under `/tmp`, or
  downloading Go or z3. If focused verification needs missing cc for cgo/sqlite,
  z3, make or another tool, run the focused gates that work, record the exact skipped
  test and missing tool in sprint JSON notes, and leave that verification to CI.
  Use `make test-changed`; never run full `make test` locally. Do not disable cgo
  or otherwise change test semantics to claim a pass.
