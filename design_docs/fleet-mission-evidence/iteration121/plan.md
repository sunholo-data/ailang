# Fleet iteration 121 — sprint plan: D-FLEET-36 A + D-FLEET-38 A follow-up

**Verdict: READY, but push-blocked until PR #2413 is on origin/dev.** The patch is built, gated and
mutation-drilled against `41f55ffdc` (#2413's head). Every changed path is ALLOWED by the #2413 guard
for `MISSION_NAME=fleet` (19/19) and REFUSED by the current origin/dev guard (16/19), as expected.

- Patch: `/tmp/fleet_i121/final-41f55ffdc.patch`
  sha256 `32f4cb10af94536e71d16371ccfc6907d9f4fc60696bb65bdd70ffd8f2bff19b`
- 19 files, +181 / -124: 2 `create_planned_doc.sh` (8 lines), 14 regenerated `.agents/skills/*/SKILL.md`,
  `tools/launchd/sync-agents-skills.sh`, `tools/launchd/test_agents_skills_sync.sh`, 1 new changelog fragment
  `changelogs/unreleased/2026-11-18-fleet-dfleet36-38-grep-and-agents-sync.md`.
- Rulings (charter `design_docs/fleet-mission.md`, Mark attended 2026-11-16): D-FLEET-36 A "the fleet
  replaces the 8 grep sites with [0-9] and resolves the ticket"; D-FLEET-38 A "the fleet extends
  sync-agents-skills.sh (SKILLS list and test_agents_skills_sync.sh) to the 16 rename-only skills".

## Re-measurement (first-party, at 41f55ffdc)

| Claim | Command | Result |
|---|---|---|
| Worktree at #2413 head, clean | `git log -1; git status --porcelain` | `41f55ffdc`, empty — CONFIRMED |
| #2413 guard has the ddc arm + 16 `.agents` arms | `sed -n 29,55p tools/launchd/githooks/pre-push` | ddc arm line 31; 16 enumerated arms, no wildcard — CONFIRMED |
| 8 `\d` sites at the stated lines | `grep -rnF '\d' .claude/skills/design-doc-creator .agents/skills/design-doc-creator` | `.claude` 128/137/154/163, `.agents` 125/134/153/162, all `grep -E "^\d+\." \|\| echo ""` — CONFIRMED. Positive control: `grep -rn 'grep -E'` sees the same 8 plus the pre-existing `[0-9]` site at `.agents:116` |
| The two copies are content-class | `diff -rq .claude/... .agents/.../design-doc-creator` | 4 files differ, `test_project_root.sh` only in `.claude` — CONFIRMED (edited in place, not synced) |
| 16 skills: none has an `.agents`-only file | `diff -rq` per skill, count `Only in .agents` | 0 for all 16 (also 0 `Only in .claude`) — CONFIRMED. Positive control: same instrument reports `Only in .claude/.../design-doc-creator/scripts: test_project_root.sh` |
| 14 rename-only + 2 identical | normalise `.claude` by `Claude Code/claude-code/.claude/Claude/claude -> Codex/.Codex`, diff vs `.agents` | 14 differ in exactly 1 file (`SKILL.md`), residual 0 after normalisation; collaboration-hub, trace-debugger identical — CONFIRMED. Positive control: model-manager residual 68 lines, design-doc-creator 91 |
| No file-mode differences in the 16 | `stat -f %Lp` per file, both trees | none |
| test suite actually runs in the done-gate | `grep -n test_agents_skills_sync make/test.mk` | in `LAUNCHD_SUITES` (line 105) — CONFIRMED |
| `\d` elsewhere | `grep -rlF '\d' .claude/skills .agents/skills` | also `sprint-planner/scripts/create_sprint_json.sh` (both copies): lines 139-168 are Python `re` (correct); **line 204 `grep -oE '...(\d+)'` is the same GNU-grep bug class — NOT in the ruling, NOT fixed here (follow-up ticket)** |
| Post-patch: skills still differing | `diff -rq` all paired skills | 10: design-doc-creator + agent-inbox, ailang-packages, benchmark-manager, cli-doc-maintainer, docs-sync, microrag, model-manager, motoko-analyzer, post-release (the charter's 11 had headless-runner, now gone, and release-manager, synced 11-12; microrag is new) |

BSD grep behaviour (`/tmp/fleet_i121/grep_behaviour.sh`, log `grep_behaviour.log`; grep = BSD grep
2.6.0-FreeBSD): fixture `1. foo.md (0.91)`, `12. bar.md`, `d. literal-d line`, `dd. two-d line`,
`  3. indented`, `x. other`. Before `^\d+\.` and after `^[0-9]+\.` both select exactly `1. ...` and
`12. ...` — byte-identical. Control `^d+\.` selects only the two literal-d lines, so the instrument
can tell a literal-d reading apart. GNU grep is not on this machine (documented semantics, unmeasured).

## Executor steps

Preconditions: #2413 merged to origin/dev (else the push is refused by the pinned guard — see Risks).

```bash
WT=<fleet executor worktree on a branch from origin/dev containing #2413>
cd "$WT"
git status --porcelain                      # must be empty
git apply --check /tmp/fleet_i121/final-41f55ffdc.patch && git apply /tmp/fleet_i121/final-41f55ffdc.patch
bash tools/launchd/sync-agents-skills.sh --check     # rc 0 (if origin/dev moved a synced .claude skill, re-run
                                                     # the sync without --check, then re-check; diff must stay within the 23 skills)
# gates (unpiped)
/bin/bash tools/launchd/test_agents_skills_sync.sh
bash tools/launchd/sync-agents-skills.sh --check
/bin/bash tools/launchd/test_mission_scope_guard.sh
bash scripts/changelog_fold.sh --check
make check-changelog
git add -A && git diff --cached --check
make test-launchd-drivers                   # ~5.5 min
git commit -m "fix(fleet): design-doc-creator grep [0-9] + sync the 16 .agents skill mirrors (D-FLEET-36 A, D-FLEET-38 A, ruled 2026-11-16)" \
           -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

If the patch does not apply because #2413 landed via a different merge, apply with `git apply -3` or
redo the four mechanical steps: (1) `perl -pi -e 's/grep -E "\^\\d\+\\\."/grep -E "^[0-9]+\\."/g'` on both
`create_planned_doc.sh`; (2) the SKILLS / header edits; (3) the test edits; (4) `bash tools/launchd/sync-agents-skills.sh`.

## Acceptance commands (patched tree at 41f55ffdc, outside any sandbox, unpiped)

| Gate | Baseline (pristine) | Patched |
|---|---|---|
| `/bin/bash tools/launchd/test_agents_skills_sync.sh` | rc 0 (5 passed) | rc 0 (7 passed, 0 failed) |
| `bash tools/launchd/sync-agents-skills.sh --check` | rc 0 (7 skills) | rc 0 (23 skills) |
| `/bin/bash tools/launchd/test_mission_scope_guard.sh` | rc 0 | rc 0 |
| `bash scripts/changelog_fold.sh --check` | rc 0 (3 fragments) | rc 0 (4 fragments) |
| `make check-changelog` | rc 0 | rc 0 |
| `git diff --check` (and `--cached --check` incl. the new fragment) | rc 0 | rc 0 |
| `make test-launchd-drivers` | not baselined (long; tree held for the patched run) | **rc 0, "all 58 suites passed", 332 s** |
| `/bin/bash -n` on both edited scripts (bash 3.2.57) | — | rc 0 |
| Round trip: patch applied to a `git archive` export, test re-run | — | rc 0 |
| `git apply --check` on the restored clean worktree | — | rc 0 |

Pinned-guard check (`/tmp/fleet_i121/guard_check.sh`, sources the hook with `MISSION_SCOPE_LIB=1`,
calls `_scope_verdict fleet <path>` for every path from `git status --porcelain`, new file included):
- worktree (post-#2413) guard: **allow 19, refuse 0**. Controls refused as expected:
  `internal/parser/x.go` (core), `.claude/skills/model-manager/SKILL.md`, `.agents/skills/model-manager/SKILL.md`.
- origin/dev (`7ec6edd25`) guard: **refuses 16** — both `create_planned_doc.sh` and the 14 `.agents/skills/*/SKILL.md`;
  allows only the two `tools/launchd` scripts and the fragment. This is why the commit cannot push before #2413 lands.

## Mutation drill (`/tmp/fleet_i121/mutate.sh`, log `mutation.log`; scratch-copy restore, sha256-verified)

| Mutant | Killing row | Result |
|---|---|---|
| (i) `.claude` copy line 128 back to `\d` | `design-doc-creator searches use [0-9], not \d` (names `:128`, sites:3) | KILLED (test rc 1) |
| (i-b) `.agents` copy line 162 back to `\d` | same row (names `:162`) | KILLED |
| (ii) drop `trace-debugger` from SKILLS | `sync list is exactly the named skills` (only expected: trace-debugger) | KILLED — note `--check` alone stays rc 0, so this row is the only guard |
| (iii) add `model-manager` as a 17th | `sync list is exactly ...` (only in script: model-manager) + `synced .agents copies equal .claude` | KILLED |
| (iv) stale `.agents/skills/eval-analyzer/SKILL.md` | `synced .agents copies equal .claude` (STALE eval-analyzer); `--check` rc 1 | KILLED |

5/5 killed; after restore all four touched files matched their pre-drill sha256, and the suite and
`--check` were re-run green.

## Risks

1. **Push ordering.** The commit is refused by the origin/dev guard until #2413 merges (16 paths). Do not
   push before; do not widen the guard in this commit.
2. **Drift of the synced .claude sources.** If origin/dev changes any of the 23 `.claude` skills between
   now and execution, `--check` goes red after `git apply`; fix by re-running the sync (verbatim copy),
   never by hand-editing the mirror.
3. **The verbatim sync changes what codex/pi read** in 14 `SKILL.md` files (`.claude` wording, `claude -p`,
   `Claude Code`) — this is the ruled behaviour (D-FLEET-38 A, "the existing precedent"), and it repairs the
   `.Codex/skills/...` paths that did not exist.
4. **`sprint-planner/scripts/create_sprint_json.sh:204`** carries the same `\d`-in-`grep -E` bug (both copies);
   it is outside this ruling and the regression guard is scoped to design-doc-creator only. File a fleet ticket.
5. **GNU-side behaviour is unmeasured** (no GNU grep here); `[0-9]` is POSIX ERE and portable by definition.
6. **Ticket resolution** (`skill:design-doc-creator-grep-portability`, `skills:agents-copies-stale-non-mission`)
   and the charter rows' status text are bookkeeping for the evaluator/controller after merge — not in this patch.
