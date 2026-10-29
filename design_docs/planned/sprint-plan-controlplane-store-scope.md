# Sprint plan: scope mission-control's prescribed message commands to the prod store

**Ticket:** `gate5:controlplane-send-refused-local-store` (mission-world iter 232, 2026-10-05)
**Iteration:** fleet 66 · **Branch:** `fleet/i66-controlplane-store-scope` (base `a4cdb274d`)
**Lane:** mission-skill fix (mechanical). **Size:** 2 milestones, about 1 hour.

## Problem

The mission-control skill prescribes `ailang messages send controlplane …` (Gate 5) and
`ailang messages send approvals …` (Gate 3) with no store scoping. On the LOCAL messaging store
the inbox guard refuses `controlplane` ("FILED, NOT DISPATCHED"; registry `~/.ailang/config.yaml`).
Sends work today only because this rig's `~/.zshenv` exports the gcp pair, which is a user
dotfile rather than the harness. The Gate-5 rule 3(c) assert, `ailang messages read --peek --json <id>`,
has the same gap: on the local store it cannot find a message that was sent to gcp.

## Direction (agree with controller)

Scope each prescribed command inline with
`env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac`, the same form the
driver uses for its own sends (`tools/launchd/mission-control.sh` ~248/307/361/2641) and
gate-3-triage uses via `$MSG`. Rejected alternatives:
- **Global driver export**: this would also move every controller read (`messages list`), and
  the standing operator rule is "cloud prefix on sends only, never a global export".
- **Declare controlplane as triage in the local registry**: `~/.ailang/config.yaml` is outside
  the repo and outside fleet write scope. It would also fix only one rig.

**Use the literal env pair for the new edits, not `$MSG`.** The Bash tool does not keep shell
variables between calls. If an agent defines `MSG=` in one call, `$MSG` is empty in the next,
and the command silently runs unscoped. That is the same defect this ticket fixes. The guard
still accepts `$MSG` in gate-3-triage.md because that file defines it, but new sites should not
copy it. Converting gate-3-triage is a possible follow-up and is outside this ticket.

Project is the literal `ailang-multivac`, not `${AILANG_MESSAGES_PROJECT:-…}`. `controlplane` and
`approvals` are prod-registry inboxes, and a stray dev-project export should not silently
re-route them. This matches gate-3-triage's `$MSG`.

## Audit (re-run by planner at a4cdb274d)

`grep -rnE "ailang messages (send|read|ack|unack) " .claude/skills/mission-control | grep -vE "… (--help|-h)"`
finds exactly 7 command-shaped lines:

| File:line | Command | State |
|---|---|---|
| resources/gate-5-retro.md:47 | `ailang messages send controlplane "<digest>" …` | **unscoped. Fix** |
| resources/gate-5-retro.md:72 | `ailang messages send controlplane "$(cat /tmp/body.txt)" …` | **unscoped. Fix** |
| resources/gate-5-retro.md:74 | `ailang messages read --peek --json <id>` (rule 3(c)) | **unscoped. Fix** |
| resources/gate-3-route.md:829 | `ailang messages send approvals \` | **unscoped. Fix** |
| resources/gate-3-triage.md:197,203,204 | `$MSG ailang messages send …` | scoped (MSG defined at :185) |

Out of scope, and why:
- SKILL.md:622 `ailang messages list --unread` is a read of the controller's own inbox, and the
  operator rule keeps reads unscoped.
- gate-3-route.md:474 and gate-5-retro.md:49/63/70 are prose mentions (backtick-terminated or `--help`).
- gate-2-pick.md:62 and gate-0-preflight.md:38 are prohibitions ("never `messages read`").
- mission-brief and mission-loop-change have no prescribed sends. sprint-* example sends are
  excluded by the ticket.

## Milestone 1: scope the four commands, sync the mirror, changelog

Edit `.claude/skills/mission-control/resources/` only:

1. **gate-5-retro.md step 3**: add one sentence before the first bullet: "Every
   `ailang messages` command in this step runs with
   `env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac` inline (as the
   driver's own sends do); on the local store the inbox guard refuses `controlplane` and the
   3(c) read cannot see the message." Then prefix lines 47, 72 and 74 with that pair.
   **Keep the env pair and `ailang messages` on the same physical line**, and re-wrap before
   `env` if needed, because the guard is per-line. The 3(c) span becomes
   `env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages read --peek --json <id>`.
2. **gate-3-route.md:829**: change it to
   `  env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages send approvals \`.
   Add no new prose; the one-sentence reason lives in gate-5 step 3.
3. Run `/bin/bash tools/launchd/sync-agents-skills.sh` to regenerate `.agents/skills/mission-control`.
   Do not hand-edit the mirror.
4. Add a changelog fragment, `changelogs/unreleased/2026-10-28-controlplane-store-scope.md`, under
   `### Fixed`, in one or two sentences, naming the ticket.

**Acceptance (M1):**
- The audit grep, minus lines containing `AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages`
  or `$MSG ailang messages`, prints nothing.
- `/bin/bash tools/launchd/sync-agents-skills.sh --check` prints `agents skills in sync` with rc 0.
- `git diff --stat` touches only the two .claude resources, their two .agents mirrors, and the fragment.

## Milestone 2: guard arm, existing 3(c) arm update, mutation drills

File: `tools/launchd/test_agents_skills_sync.sh` (already in `LAUNCHD_SUITES`, so no make change).

1. **Update the existing 3(c) arm (required, or it goes red after M1).** It currently matches
   spans with `^ailang messages read` and requires an exact equality. Inside the awk loop,
   before the regex test, strip the scope prefix from a copy of the span:
   `s = spans[i]; sub(/^env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac /, "", s)`.
   Then test and compare `s`. BSD awk `sub()` with a regex literal is fine. Leave the PASS/FAIL
   names and messages as they are.
2. **New arm: "mission-control prescribed message commands are store-scoped".** Use bash 3.2,
   BSD grep `-E` only (no `\|`, no `-P`), and `[[:space:]]` classes where needed:
   - `cmds=$(grep -rnE 'ailang messages (send|read|ack|unack) ' "$ROOT/.claude/skills/mission-control" | grep -vE 'ailang messages (send|read|ack|unack) (--help|-h)')`
   - **Positive control (an empty search is a claim):** fail unless `cmds` has at least one
     `gate-5-retro.md` line containing `send controlplane` and at least one `gate-3-route.md` line
     containing `send approvals`. This proves the instrument sees the sites it guards.
   - `unscoped=$(printf '%s\n' "$cmds" | grep -vE 'AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages|\$MSG ailang messages')`.
     If `unscoped` is non-empty, fail and print the offending `file:line` entries with
     `${…#$ROOT/}`.
   - `$MSG` legitimacy: for every file with a `$MSG ailang messages` line, require
     `grep -qF 'MSG="env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac"'` in
     that same file. Otherwise fail, naming the file.
   - Add a one-line comment above the arm saying why: the local store refuses `controlplane`, and
     only a user dotfile made the sends work.
3. Keep the file under about 120 lines. It is 68 now, and the arm adds about 25.

**Mutation drills (executor must run all three and record red→green in the PR/iteration log).**
Back up each file to a scratch copy and restore with `cp`. **Never `git checkout <file>`**: it
would also revert the uncommitted M1 edits.

```bash
WT=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter66; T=$(mktemp -d)
R5=$WT/.claude/skills/mission-control/resources/gate-5-retro.md
R3=$WT/.claude/skills/mission-control/resources/gate-3-route.md
cp "$R5" "$T/g5"; cp "$R3" "$T/g3"
# D1: unscope the Gate-5 controlplane send (line ~47) -> new arm FAILS naming gate-5-retro.md
L=$(grep -n 'send controlplane "<digest>"' "$R5" | head -1 | cut -d: -f1)
sed -i '' "${L}s/env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac //" "$R5"
/bin/bash $WT/tools/launchd/test_agents_skills_sync.sh; cp "$T/g5" "$R5"
# D2: unscope gate-3-route approvals line -> new arm FAILS naming gate-3-route.md
#     (edit line ~829 to drop the env pair), run, then: cp "$T/g3" "$R3"
# D3: unscope the 3(c) read (line ~74) -> new arm FAILS; the 3(c) arm still PASSES
#     (proves the two arms are independent), run, then: cp "$T/g5" "$R5"
cmp "$T/g5" "$R5" && cmp "$T/g3" "$R3" && rm -rf "$T"
```

Make every mutation by line number, found with `grep -n` as D1 does. BSD `sed` has no
`0,/re/` address. Optional D4: delete the `MSG=` definition from
gate-3-triage.md. The `$MSG` legitimacy check should go red, and the file is then restored with `cp`.
The drills run against the .claude source only. The sync check will also be red during a
drill, which is expected. After restoring, confirm that `sync-agents-skills.sh --check` is green again.

**Acceptance (M2):**
- `/bin/bash tools/launchd/test_agents_skills_sync.sh` has rc 0, and its output includes the new
  arm's PASS line and the unchanged 3(c) PASS line.
- D1, D2 and D3 each turn the new arm red with the right file named. After the restore, `cmp`
  confirms the files are byte-identical to the backups.

## Done gate (executor runs, unpiped)

```bash
cd /Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter66
/bin/bash tools/launchd/sync-agents-skills.sh --check   # rc 0
make test-launchd-drivers                               # rc 0 (do not pipe: a pipe hides the rc)
git status --short   # only: 2 .claude resources, 2 .agents mirrors, test_agents_skills_sync.sh, fragment, this plan
```

## Files to change

- `.claude/skills/mission-control/resources/gate-5-retro.md`: one sentence plus three prefixes
- `.claude/skills/mission-control/resources/gate-3-route.md`: one prefix
- `.agents/skills/mission-control/resources/gate-5-retro.md`, `…/gate-3-route.md`: generated by sync
- `tools/launchd/test_agents_skills_sync.sh`: 3(c) arm prefix-strip plus the new scope arm
- `changelogs/unreleased/2026-10-28-controlplane-store-scope.md`: `### Fixed`

All are within fleet write scope. The change does not touch the driver, the registry or `make/test.mk`.

## Residual risk

- The guard is per-line. A future command wrapped so that `env …` and `ailang messages` fall on
  different lines reads as unscoped and fails the arm. That is a false red, and it errs in the
  safe direction.
- A line that holds one scoped and one unscoped command would pass. No such line exists today.
- gate-3-triage's `$MSG` still depends on shell-variable persistence, as noted above.
