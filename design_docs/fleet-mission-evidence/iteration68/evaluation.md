# Fleet i68 evaluation: M-PLANNER-PLACEMENT-STOP-GATES (249ee0d94)

VERDICT: PASS. Score 90/100 (pass mark 70).
Scores (my own split, not a line-by-line rubric re-score): correctness 38/40, tests and mutation 24/25, prose clarity 14/15, scope and bookkeeping 9/10, gates 10/10 minus 5 for the unrun launchd suite and unverified C6 = 5 deducted => 90 total is approximate; treat as 88-90.

## BLOCKING findings
None.

## Non-blocking findings
1. Case-variant bypass on macOS. `design_docs/planned/AILANG-core-triage/x.md` is accepted (rc=0). APFS folds the case, so the file lands in the triage dir while the guard compares case-sensitively. It rarely matters. A planner writing the triage name with wrong case is unlikely, and git on case-sensitive CI would see a different path, which the triage rubric test would not read. Cheap fix: lowercase both sides in the guard.
2. Python3 is a hard dependency of the guard. With python3 absent from PATH the script aborts rc=127 before any side effect (good, fail-loud). It is undocumented, and the image must have python3.
3. The committed `.ailang/state/sprints/sprint_M-PLANNER-PLACEMENT-STOP-GATES.json` shows M2 `"passes": false`. This is cosmetic bookkeeping. 279 files are already tracked in that directory, so committing it is not a defect.
4. "decision-ledger row ID" is cited in the rules but is not defined in SKILL.md. A planner with no ledger must cite a design section. This is acceptable because the rule allows "design section or ledger row".

## Gates run (judge worktree)
- `/bin/bash -n` and `shellcheck -S warning` on create_sprint_json.sh and test_create_sprint_json.sh, in both .claude and .agents: all clean.
- `test_create_sprint_json.sh`: .claude 49 passed 0 failed; .agents copy 49 passed 0 failed.
- `tools/launchd/sync-agents-skills.sh --check`: in sync. `tools/launchd/test_agents_skills_sync.sh`: 5 passed 0 failed.
- `bash scripts/check_context_docs.sh`: OK. `make check-skills`: OK. `make check-changelog`: OK (20 fragments).
- SKILL.md is 597 lines.
- `make test-launchd-drivers`: skipped, as allowed. make/test.mk wiring was read: the suite is listed in LAUNCHD_SUITES and both `bash -n` lines are present.

## My drills
Each ran against a throwaway repo copy with a stub `ailang` first on PATH and a throwaway HOME. The stub log line count shows whether `import-github` was reached. Results were identical for the .claude and .agents copies, so `parents[4]` resolves correctly in both.
| plan argument (cwd) | rc | stub calls |
|---|---|---|
| `design_docs/planned/./ailang-core-triage/x.md` | 2 | 0 |
| `design_docs/planned/v1/../ailang-core-triage/y.md` (nonexistent) | 2 | 0 |
| absolute path into triage | 2 | 0 |
| `../../design_docs/planned/ailang-core-triage/x.md` from `sub/deep` | 2 | 0 |
| `../lnk/x.md` from `sub`, where lnk is a symlink to the triage dir | 2 | 0 |
| `lnk/x.md` | 2 | 0 |
| `ailang-core-triage//x.md`, the bare triage dir, `triage/../triage/x.md` | 2 | 0 |
| `design_docs/planned/ailang-core-triage-extra/x.md` (sibling) | 0 (accepted, as required) | 1 |
| legal plan `v1/ok.md` with a triage design doc as 3rd argument | 0 (accepted, as required) | 2 |
| `design_docs/planned/AILANG-core-triage/x.md` | 0 (bypass, finding 1) | 1 |
| python3 absent from PATH | 127, aborts, no `.ailang` created, stub log never written | 0 |

Guard placement was confirmed by reading the script: `set -e` is at line 25, and the guard runs before the `PROGRESS_DIR` mkdir and before `import-github` at line 99.

## Mutation checks
- Guard call deleted: test fails 30 of 49 arms (red).
- Guard moved below the import-github call: 27 FAIL lines (red).
- Canonical-resolution branch removed from the guard: fails canonical-refusal, foreign-cwd-refusal and the before-mkdir, before-import and diagnostic arms (red).
- The test therefore pins ordering as well as presence.

## Prose review (asks 4 and 5)
The rule appears in three places and they agree: the SKILL.md checklist bullet, coordinator.md, and the template section "Placement and Human-Decision Rules". The template section is the anchor target of the SKILL.md link. A headless planner knows where to put a plan: a version dir or the planned root as `sprint-plan-<slug>.md`. It knows the gate wording: STOP, commit current work, report `BLOCKED: <decision>`, end the run. It knows the open-vs-ruled rule. The Open Questions template line was also rewritten. SKILL.md's existing examples (lines 47, 62, 158, 292, 405, 563) all use versioned directories and do not contradict. coordinator.md's example `design_doc_path` is a v0_6_3 path, also fine. One residual tension: the SKILL.md bullet is dense, but it links to the full rule.

## Claims
- C1 CONFIRMED. `dbe4b26a3` is an ancestor of HEAD. It edits the planner SKILL.md (+2 lines), adds the 2026-10-12 focused-gates changelog fragment, and changes make/test.mk. The bullet "Executor gates are focused, never the full suite" covers `make test-changed`, fragments, the 800-line companion-file rule, and the ~60-90 min milestone size.
- C2 CONFIRMED. `b2d21a070` is an ancestor, and `docker/Dockerfile.agent-base` installs `jq` (line 55) and checks `jq --version` (line 70).
- C3 CONFIRMED. The four commits are the plan (2d5fda267), M1 (29b945e55), M2 (0ea4dd233) and M3 (249ee0d94), and the diff matches the description. The test is wired into `LAUNCHD_SUITES` plus the two `bash -n` lines.
- C4 CONFIRMED. SKILL.md is 597 lines and check_context_docs passes.
- C5 CONFIRMED. See the drills: stub log empty and no `.ailang` directory on every refusal. `set -e` is present, and a missing python3 aborts with rc=127.
- C6 PARTIAL. I could not retrieve the executor's original tree to diff against. First-party observation: the new bullet at SKILL.md line 335 sits directly above the "Executor gates" bullet and inside the checklist list. The claim of a one-blank-line swap is unverifiable here but harmless.
- C7 CONFIRMED. At HEAD `internal/mission/quorum/quorum.go` counts only present external outcomes (`externalPresentCount`); the controller does not feed it. a5f9af982 (#1767, 2026-10-12) is an ancestor of HEAD. `TestSynthesize_ExternalSignalMatrix` pins the controller-pass cases. Mutation: changing the guard to `externalPresentCount == 0 && controller == nil` in the judge worktree turned four tests red (ExternalSignalMatrix, SalvageSynthesis, RunSeatedQuorum_AllAbsentControllerPassBlocks, SynthesizeWithTier2_PassCannotRescueZeroSignalTier1). I restored the file afterwards and the tree is clean. GitHub #651 is CLOSED. The ticket is a ghost and may be closed.

## Single strongest objection
The guard is a hard-coded, case-sensitive string match plus a Python subprocess. It misses the case-fold bypass on macOS, and it makes python3 a silent prerequisite of a script that previously needed only bash and jq-free tools. Neither is likely in practice. The better defence is the prose rule, and the guard is only a backstop.
