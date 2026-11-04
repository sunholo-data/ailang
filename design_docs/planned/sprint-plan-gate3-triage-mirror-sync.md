# Sprint plan: resync mission-control mirror after the gate-3-triage edit

Ticket: `ci:skill-mirror-not-synced-after-gate3-triage-edit` (fleet iteration 84).
Baseline: worktree `.wt-fleet-i84`, HEAD `abd4dfd82` (= origin/dev), clean tree, measured 2026-11-04.
Precedent: iteration 63 / PR #1972 (`sprint-plan-agents-mirror-sync-red.md`), same failure class.
Scope: mechanical repair, ~15 min. No new code, no hand edits to any mirrored file.

## Measured diff set (planner, first-party)

`sync-agents-skills.sh` mirrors six whole directories (mission-control, mission-brief,
mission-loop-change, sprint-planner, sprint-executor, sprint-evaluator) with
`rsync -a --delete`, so it also deletes `.agents`-only files in those six. `--check` is `diff -rq`.

| Check | Result |
|---|---|
| `diff -rq .claude/skills/<s> .agents/skills/<s>` for all six | only `mission-control/resources/gate-3-triage.md` differs; no `Only in` lines (no `.agents`-only file to delete) |
| `git show 88d654eca:.claude/.../gate-3-triage.md \| cmp - .agents/.../gate-3-triage.md` | rc 0: the mirror is exactly the canonical file at the last green commit |
| `diff .agents/... .claude/...` | 0 lines removed, 49 added: exactly f22bf1f4b's +49 to the canonical file |

So the write-mode sync only applies f22bf1f4b's canonical edit. No `.agents`-only content is clobbered.

## M1: sync, changelog, commit

1. `/bin/bash tools/launchd/sync-agents-skills.sh` (write mode, no args). Never hand-edit or `cp` the file.
2. `git status --short` must show exactly one modified file under `.agents/`:
   `.agents/skills/mission-control/resources/gate-3-triage.md`. Any other path: stop and report.
3. `cmp .claude/skills/mission-control/resources/gate-3-triage.md .agents/skills/mission-control/resources/gate-3-triage.md` → rc 0.
4. Changelog fragment `changelogs/unreleased/2026-11-04-gate3-triage-mirror-sync.md`, one section:
   `### Fixed — dev CI red: .agents mission-control mirror resynced after gate-3-triage edit (2026-11-04)`
   with a body naming f22bf1f4b as the cause, the 3 required checks it failed, and the fix (sync script).
5. Commit the mirror file, fragment and this plan together.

## Acceptance commands (run unpiped, record rc)

| Command | Base rc (abd4dfd82) | Expected head rc |
|---|---|---|
| `/bin/bash tools/launchd/sync-agents-skills.sh --check` | 1 (`STALE: .agents/skills/mission-control …`) | 0 (`agents skills in sync …`) |
| `/bin/bash tools/launchd/test_agents_skills_sync.sh` | 1 (4 passed, 1 failed) | 0 (5 passed, 0 failed) |
| `/bin/bash tools/launchd/test_quorum_artifacts.sh` (no env needed; it scrubs `MISSION_*` and sets a fixture HOME) | 1 (`FAIL generated_docs_and_mirror: sync`) | 0 |
| `go test ./internal/mission/quorum/ -run TestQuorumArtifactShellSuite -count=1` | 1 (same arm) | 0 |
| `make check-changelog` | 0 (2 fragments) | 0 (3 fragments) |
| CI required checks `test`, `launchd drivers (bash 3.2)`, `Build ubuntu-latest` | red (controller-measured; UNMEASURED by planner, CI-only) | green on the pushed head SHA (evaluator reads origin) |

## Mutation drill (executor must run, then restore)

Use a scratch copy. Do not run `git checkout <file>`, which would discard the uncommitted sync.
```
M=.agents/skills/mission-control/resources/gate-3-triage.md
cp "$M" /tmp/i84-synced.md
git show abd4dfd82:"$M" > "$M"                       # revert mirror to base
/bin/bash tools/launchd/sync-agents-skills.sh --check; echo rc=$?   # expect 1, STALE
go test ./internal/mission/quorum/ -run TestQuorumArtifactShellSuite -count=1; echo rc=$?  # expect 1
cp /tmp/i84-synced.md "$M"                           # restore
cmp "$M" .claude/skills/mission-control/resources/gate-3-triage.md; echo rc=$?  # expect 0
/bin/bash tools/launchd/sync-agents-skills.sh --check; echo rc=$?   # expect 0
```
Record all four rc values in the iteration log.

## Out of scope

- **Pre-push guard so a skill edit cannot reach dev without its mirror** (the ticket's optional item):
  not this sprint. It changes the pre-push hook (`scripts/hooks/pre-push` / `tools/launchd/githooks/pre-push`),
  which judges every push on the rig, so it needs its own design and drill rather than riding on a docs repair.
- Reconciling the non-fleet skills that carry `.agents`-only edits (model-manager, design-doc-creator):
  the script deliberately excludes them.
- Any change to `gate-3-triage.md` content, `sync-agents-skills.sh`, or the test suites.
