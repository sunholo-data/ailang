# Fleet i84 evaluation - PR #2116 (96e1691a9 vs base abd4dfd82)

Score: 91/100. Verdict: PASS. Blocking findings: none.

## Breakdown
- Correctness vs ticket (35): 34. CI logs read first-hand. Run 41803697385: jobs `test` (98 KB log) and `launchd drivers (bash 3.2)` (178 KB log) failed, both with "STALE: .agents/skills/mission-control differs from .claude/skills/mission-control". `test` failed on TestQuorumArtifactShellSuite "generated_docs_and_mirror: sync". The launchd job failed on test_quorum_artifacts.sh (rc=1) AND on test_agents_skills_sync.sh (rc=1, same STALE message); the ticket names only the first. No other FAIL lines in either log (all other launchd arms PASS). Ticket cause confirmed. Base .agents gate-3-triage.md existed but was 49 lines behind.
- Verification evidence (25): 24. All acceptance commands pass at head (below). Base failure is shown by the CI logs on the base-line state, not re-run locally.
- Scope discipline (15): 14. Diff = 3 files: .agents gate-3-triage.md (+49, exactly the D-TRIAGE-48 sections 0 and 7a from the canonical file), the changelog fragment, and a sprint-plan doc under design_docs/planned/. Only one mirrored file changed, so nothing was deleted and no .agents-only content was lost (the sync script rsyncs --delete only within its 6 mission-*/sprint-* skills; `diff -rq` of mission-control is clean).
- Regressions (10): 10. The full sync --check covers all 6 mirrored skills.
- Docs/changelog (15): 9. Format valid (`### Fixed - ...`, lowercase slug, make check-changelog rc=0). Inaccurate claim, see non-blocking 1.

## Commands (all in judge worktree, rc captured without pipe)
- `/bin/bash tools/launchd/sync-agents-skills.sh --check` rc=0
- `/bin/bash tools/launchd/test_agents_skills_sync.sh` rc=0 (5 passed, 0 failed)
- `/bin/bash tools/launchd/test_quorum_artifacts.sh` rc=0 (PASS generated_docs_and_mirror)
- `go test ./internal/mission/quorum/ -count=1` rc=0
- `make check-changelog` rc=0
- `diff -rq .claude/skills/mission-control .agents/skills/mission-control` empty
- Base failing state: taken from CI logs (rc=1 for both scripts), not re-run.
- PR check-runs on 96e1691a9 were still pending/unfinished when I looked (test, launchd, build blank). The final proof is CI on the PR.

## Mutations (restored with cp, verified with cmp; worktree clean)
1. Appended a line to .agents/skills/sprint-evaluator/SKILL.md: sync --check rc=1, test_quorum_artifacts rc=1, test_agents_skills_sync rc=1. Caught.
2. Deleted .agents/skills/mission-brief/SKILL.md: sync --check rc=1 "STALE: mission-brief". Caught.
3. Added extra file .agents/skills/sprint-planner/EXTRA.md: --check rc=1. Caught (diff -rq flags extras).

## Non-blocking findings
1. Changelog and commit message say `Build ubuntu-latest` was red. In run 41803697385 the job `build` was success and no job named "Build ubuntu-latest" is in that run's job list; I could not verify that claim. Reword to the two jobs confirmed (`test`, `launchd drivers (bash 3.2)`).
2. The changelog says "Resolves harness ticket ci:skill-mirror-not-synced-after-gate3-triage-edit" while the guard is not done; see item 6.
3. The sprint-plan doc was added under design_docs/planned/ (66 lines); harmless, but extra surface for a one-file fix.
4. The ticket's causal chain mentions only the quorum test; the launchd job had a second failing script. Same root cause, fixed.

## Item 6: guard out of scope?
Reasonable for a one-item iteration; the ticket marks it "consider". The same failure class has occurred before (sync script header cites 2026-09-29 staleness), and the check is already enforced in CI after the fact, so the gap is dev red for hours on a direct push. Recommendation: resolve the ticket for the red-CI fix, and file a separate follow-up (pre-push hook running `sync-agents-skills.sh --check` when .claude/skills/{mission-*,sprint-*} changes; the existing pre-push stdlib freeze gate is the model). Do not leave the red-CI ticket open for it.

## Strongest objection
Nothing prevents the next skill edit from re-breaking dev: the fix is a one-time resync and the root cause (direct dev push without mirror regeneration) is untouched, so the changelog's "Resolves" overstates. Also the PR's own CI had not finished when checked, and one claim in the changelog (Build ubuntu-latest) is unverified.
