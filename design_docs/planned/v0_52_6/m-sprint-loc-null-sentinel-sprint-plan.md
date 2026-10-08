# Sprint Plan: M-SPRINT-LOC-NULL-SENTINEL

**Refs #563** — sunholo-data/ailang; no new issue.
**Design:** [Approved sentinel migration](../../implemented/v0_52_6/m-sprint-loc-null-sentinel.md)
**Date:** 2026-10-08
**Status:** Implemented; ready for evaluator. Core regression requires a compiler-equipped environment.
**Duration:** 1 working day, approximately 5 hours including 1 hour contingency.
**Risk:** Low; pre-existing zero placeholders require an explicit audit.

## Goal and current status

Make JSON `null` the unfilled milestone LOC sentinel. Accept an honest numeric zero, reject null or a missing key, and preserve the same contract in `.claude` and `.agents` skills.

Read issue #563 and its comments through the GitHub API on 2026-10-08: open, P3, no comments. The approved design records the live defect on origin/dev 658ff76a3. This checkout still writes zero on both creator paths and rejects zero in both validators. The session display interpolates null literally; the schema still says number. No implementation has been completed.

The seven-day velocity script found no usable LOC metrics. Git is shallow and exposes one recent documentation commit, so a measured LOC/day rate cannot be inferred. The current v0.52.5 changelog confirms recent activity but supplies no comparable script-migration duration. Use the design's five-hour estimate, with four hours scheduled work plus one hour contingency. Estimated changed production/documentation lines: 40 across both trees and the completion report; fixture commands are temporary and have no committed test LOC. Target capacity: 40 LOC/day, a planning allocation rather than measured velocity.

## Registry reuse audit

M1 and M2 both use action `none`, package null. This is a repository-local Bash/Python/jq placeholder protocol and its verification, with no package-like capability, AILANG module or dependency to source from the registry. Registry searches and candidate inspection are therefore not applicable. Reuse the existing creator, validator, session-start scripts and design-doc move helper; add no tool or dependency to the repository.

## Milestones

### M1: Migrate sentinel producers and consumers (~24 LOC)

**Estimated:** 24 changed implementation/documentation LOC; 0 committed test LOC.
**Duration:** 2 hours. **Dependencies:** None.

Update these relative paths under **both** `.claude/skills/` and `.agents/skills/`:

- `sprint-planner/scripts/create_sprint_json.sh`: Python `None` on the no-LOC parsed heading and no-milestone fallback; keep parsed integers, including zero.
- `sprint-executor/scripts/validate_sprint_json.sh`: `.estimated_loc == null`, with the comment explaining null/absent versus valid zero.
- `sprint-executor/scripts/session_start.sh`: display `(.estimated_loc // "unset")`.
- `sprint-executor/resources/json_progress_schema.md`: number-or-null field contract; null/absent is unfilled and zero is a real estimate.

**Acceptance criteria:**

- [x] Both creator placeholder paths emit JSON null; explicit heading estimates including zero retain their numeric value.
- [x] Both validators reject null and an absent key and accept zero without changing other placeholder checks.
- [x] Session displays preserve `0 LOC` and render unfilled estimates as `unset LOC`; both schemas document this contract.
- [x] All four pairs of modified files are byte-identical; `bash -n` passes for all six modified shell files.

**Risk:** Editing only one skill tree or one creator branch. Mitigate with pairwise `cmp` and two creator probes.

### M2: Verify fixtures and record migration audit (~16 LOC)

**Estimated:** 16 completion/changelog/report LOC; 0 committed test LOC.
**Duration:** 2 hours plus 1 hour contingency. **Dependencies:** M1.

Use throwaway working directories with `.ailang/state/sprints/` fixtures, invoking each tree's actual scripts by absolute path. Each fixture must have at least two real milestones, valid full dependency IDs, real descriptions/criteria, and populated registry reuse. Vary only one milestone's estimate. Exercise creator fallback/no-LOC paths and explicit `(~0 LOC)`/`(~245 LOC)` headings. Record commands and exit codes in the implementation report; no new `.ail` examples are warranted for shell tooling.

**Acceptance criteria:**

- [x] Both validators return 0, 1, 1, 0 for zero, null, absent and 245 respectively; rejected milestone IDs appear in diagnostics.
- [x] Both creators emit null for no-LOC headings and fallback templates, and retain numeric zero/245 from explicit headings.
- [x] Session-start round trip demonstrates zero and unset output, using valid temporary fixtures so validation gates do not obscure display verification.
- [x] Two or three completed sprint regression controls pass before and after; record any baseline warnings separately rather than altering records to hide them.
- [x] Audit every current non-completed zero milestone and record its disposition in this sprint's notes/report; no pre-existing sprint JSON is rewritten.
- [x] Pairwise comparisons and shell syntax checks pass; run `make test-core` as the approved design's regression check and record its result.
- [x] Update `changelogs/v0.32-current.md` under Unreleased; use the existing move helper to move the design to implemented/v0_52_6 and update this sprint's design_doc link at completion.

**Risk:** Old unfilled numeric zeros become indistinguishable after migration. Explicitly flag uncertain estimates for their owning planners; do not fabricate replacement estimates or mutate historical state.

## Day 1 schedule

1. Hours 0–2: M1 edits, parity and shell syntax checks.
2. Hours 2–3.5: M2 fixture matrix, creator/display probes and baseline-qualified completed-sprint controls.
3. Hours 3.5–4: Audit report, changelog and design completion bookkeeping.
4. Hour 4–5: Contingency for fixture isolation, baseline failures or documentation helper behavior.

## Read-only legacy audit and design clarification

Current audit of all 190 sprint files confirms these non-completed zero milestones:

| Sprint / milestone | Disposition |
|---|---|
| M-CHAINS-EXECUTOR-TRANSCRIPTS / M0_PREFLIGHT | Valid zero: execution-baseline checks with no implementation deliverable. |
| M-LIST-ACCESSOR-API / M6_CENSUS_REPORT_AND_HANDOFF | Valid zero under the approved design's docs-only convention; census/report and handoff work. |
| M-MISSION-AGENTIC-ROUTING / M1b_CODEX_CROSS_PROVIDER_EXECUTOR | Unresolved legacy zero: explicitly requires on-disk skill changes; owning planner must confirm or re-estimate before its next execution. |
| M-MISSION-AGENTIC-ROUTING / M2_RIGHTSIZING_TABLE_AND_EVIDENCE_SCHEMA | Valid zero under docs-only convention: charter/table and evidence schema edits. |
| M-MISSION-AGENTIC-ROUTING / M3_PLANNER_DOWNTIER_AB_PARKED | Valid zero under docs-only convention: parked protocol deliverable, execution deferred. |

M-DX27-DOCS-SEARCH-GITHUB-FALLBACK has empty features: no zero milestone to classify. It is not execution-ready but is unrelated to this sentinel migration.

The design simultaneously calls for a read-only audit and recording notes in old sprint files, including setting old placeholders to null. This plan follows its explicit non-goal and success criterion: leave **all pre-existing state files untouched**, and record rulings in this new sprint's notes/report. Any unresolved legacy estimate is routed to its owner as subsequent work. Completed zero-bearing history remains unchanged. Refresh the audit at execution time.

## Success metrics and handoff

40 estimated changed LOC, two dependency-ordered milestones, complete sentinel fixture coverage on both trees, and no newly introduced dependency. Keep `velocity.estimated_total_loc`'s warning-only default and historical references unchanged. No language-level examples, percentage coverage target or compiler implementation changes are required.

Tool requirements: Bash, Python 3, jq, Go/make for `make test-core`. Planning environment initially lacks jq and gh; issue inspection used curl and jq 1.7.1 was staged outside the repository for artifact validation. Executor must verify its own prerequisites.

The coordinator owns review/approval and dispatch of the plan. Do not self-merge or initiate implementation in this planning stage. The coordinator-created PR body must include **Refs #563** and link both artifacts; do not create a new issue or link #544 as an implementation target. No unresolved design decision blocks this plan.

## Execution result

Both milestones implemented; see the design implementation report for fixture exits, controls, and the refreshed audit. `make test-core` could not start because make is absent; direct Go execution passed all listed core packages except CGO-dependent effects SQLite tests. Refs #563.
