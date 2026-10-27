# Sprint plan: repair sprint-evaluator mirror drift

Goal: restore required CI checks and give Codex/pi evaluators the coordinator-readable
final `EVALUATION_VERDICT:` instruction already present in the canonical skill.

Planning baseline: 2026-10-27, HEAD `5f56181a1`, initially clean working tree.
Scope: mechanical repair, approximately 15–30 minutes; no implementation during planning.
Registry reuse: none — existing sync tooling, no package-like capability or new code.

## Verified sync scope

Each command below was run before any repository edits; rc 0 means identical.
Only `.agents/skills/sprint-evaluator/SKILL.md` would change; no other drift found.

| Command | rc | Result |
|---|---|---|
| `diff -rq .claude/skills/mission-control .agents/skills/mission-control` | 0 | Identical; no output. |
| `diff -rq .claude/skills/mission-brief .agents/skills/mission-brief` | 0 | Identical; no output. |
| `diff -rq .claude/skills/mission-loop-change .agents/skills/mission-loop-change` | 0 | Identical; no output. |
| `diff -rq .claude/skills/sprint-planner .agents/skills/sprint-planner` | 0 | Identical; no output. |
| `diff -rq .claude/skills/sprint-executor .agents/skills/sprint-executor` | 0 | Identical; no output. |
| `diff -rq .claude/skills/sprint-evaluator .agents/skills/sprint-evaluator` | 1 | Only SKILL.md differs. |

## M1: synchronize and document

Run exactly `bash tools/launchd/sync-agents-skills.sh` with no arguments.
Confirm the sole changed mirror file is `.agents/skills/sprint-evaluator/SKILL.md`.
If a fresh preflight finds additional drift, STOP and report loudly: sync could clobber
an `.agents`-only edit; reconcile scope before running the overwrite.
Create `changelogs/unreleased/2026-10-27-agents-mirror-sprint-evaluator.md` with:

```markdown
### Fixed — sprint-evaluator skill mirror drift (2026-10-27)

The evaluator instruction update left the Codex/pi skill mirror stale, failing required
mirror checks and retaining the obsolete `EVALUATION_RESULT: pass|fail` marker.
Re-synced the mirror to emit the final coordinator-readable `EVALUATION_VERDICT:` line.
See the [repair plan](../../design_docs/planned/sprint-plan-agents-mirror-sync-red.md).
```

## M2: acceptance and mutation drill

Untouched-tree baselines, run sequentially to completion:

| Acceptance command | rc | Base reading |
|---|---|---|
| `bash tools/launchd/sync-agents-skills.sh --check` | 1 | RED: STALE sprint-evaluator mirror only. |
| `bash tools/launchd/test_agents_skills_sync.sh` | 1 | RED: same STALE line; 2 passed, 1 failed. |
| `bash tools/launchd/test_quorum_artifacts.sh` | 1 | RED: same STALE line at generated_docs_and_mirror: sync. |
| `go test ./internal/mission/quorum/ -run TestQuorumArtifactShellSuite -count=1` | 1 | RED: shell suite reaches the same STALE mirror arm. |

No sandbox denial or unrelated failure was observed. After M1, all four commands must
return 0; all six directory comparisons must return 0. Run `make check-changelog` too.
If any run instead hits a sandbox denial (e.g. bind denied or permission outside the
tree), label it **UNINFORMATIVE UNDER SANDBOX**, never pass/fail; report other reds separately.

One mutation, performed only after the fix is green (not executed by the planner):
1. Save first: `target=.agents/skills/sprint-evaluator/SKILL.md`, then
   `scratch=$(mktemp /tmp/agents-evaluator.XXXXXX)` and `cp "$target" "$scratch"`.
2. After successful backup, append one byte with `printf '\n' >> "$target"`.
3. `bash tools/launchd/sync-agents-skills.sh --check` must redden: rc 1, STALE evaluator.
4. Regardless of the result, restore with `cp "$scratch" "$target"`; verify with
   `cmp "$scratch" "$target"` (rc 0). Keep the scratch copy until restoration is verified.
5. Re-run all four acceptance commands; require rc 0 after restoration.

## Non-goals

No sync skill-list changes, hooks, prevention mechanisms, language-core changes,
new tests, or git writes. The controller handles git and subsequent execution.
