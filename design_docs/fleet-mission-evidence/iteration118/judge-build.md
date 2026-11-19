# i118 build evaluation (judge3) -- M-FLEET-RULINGS-LEDGER @ 1e114ec7c
Score: 91/100 -- PASS (70 to pass). No blocking findings.

## Evidence
- go test -run 'Mission|Doctor|Ruling' (mission + cmd/ailang): ok. go test ./internal/mission/ full: ok. gofmt -l: empty. go vet: clean. golangci-lint ./internal/mission/...: 0 issues. Build /tmp/judge3/ailang rc 0.
- Real repo (AILANG_MISSION_REGISTRY=$PWD/missions): new rc=1, installed `ailang` rc=1; the 6 DRIFT lines (env-source-drift x6) are byte-identical. Ruling notes print FIRST: 2 x ruling-link-missing (stapledon, world; charters live outside the repo), 0 ruling-contradiction. Matches design prediction. Default registry (dev checkout, no ledger yet) prints one `rulings rulings-ledger-absent` note first, same rc.
- Planted literals in a scratch tree copy /tmp/judge3/tree (no tracked file touched): `[note] v1 ruling-contradiction D-FR-1 design_docs/v1-mission.md:4952 contains retired text "WORLD IS A FORK"` and `[note] rulings ruling-contradiction D-FR-2 .claude/skills/mission-control/SKILL.md:630 ...`. Control: same registry unplanted = none.
- mission-lane-check cell (line 49-50 logic, tail -1 after the grep filter) for `doctor world|stapledon|docs`: last line is the DRIFT line, not a note (world and stapledon have a link-missing note above it). All-drift tree: last line is a DRIFT. The exit-0 case (notes only) ends in "N mission(s): no drift", the string test_mission_lane_check.sh:80 expects. No other script parses doctor output (grep tools/ scripts/).
- Scope: all 27 paths classify as harness (17) or fleet-support (10) via the pre-push functions extracted to a file (negative control `zzz/x` refused). Gate-0 sentence: .claude and .agents copies cmp-identical. scripts/mission_decisions.sh --check on fleet-rulings.md: valid, 2 rows.
- Transcription rows: D-FR-1 faithful to the D-FLEET-25 A ANSWER (Mark attended 2026-11-16; World fork claim corrected, shared driver). D-FR-2 faithful to v1 D-62 (vendor preference, model-level self-review refused). Design section 5 rule (transcription cites its source row in Evidence; review checks) followed. Nit: D-FR-2 also cites D-63, which retires NO-SINGLE-PROVIDER-ROLE, not generator!=judge; harmless over-citation. Note these rows are loop-authored RESOLVED; per CLAUDE.md principle 4 that is consistent with the design (artifact identical), and the design says so honestly.
- Commit-body deviations: M6 live run predicted two link-missing/0 contradiction was corrected to the ledger-absent case for the default registry; verified above. make test-changed rc2 attributed to two pre-existing failures outside mission code (not re-run by me; whole-repo run forbidden). During my cmd/ailang mutation run, TestDocsSearch_ConfinementActualCLI failed once under a mutant while the DoctorPrint tests were the intended killers; unrelated to the diff (it passed in my baseline run) but not investigated.

## Mutation table (scratchpad copies, all restored; git status clean)
| id | mutant | result |
|---|---|---|
| A | rd.note Severity Note->Drift | KILLED TestRulings_EveryFindingIsNote |
| B | cap slice 20->21 | KILLED TestRulings_HitCapTwentyThenSummary |
| C | sort disabled | KILLED TestRulings_OutputIsSorted |
| D | CLI summary condition back to len(Findings)==0 | KILLED TestMissionDoctorPrint_NotesThenSummary |
| E | link counted inside ledger block | KILLED |
| F | charterPath abs branch removed | KILLED TestRulingsCharterPath_Location |
| G | case-insensitive literal | KILLED ...ByteExactCaseSensitive |
| H | root disagreement >1 -> >2 | KILLED TestRulingsRoot_Derivation |
| N | skills walked once -> every detector | KILLED |
| P | CheckRulings given no missions | KILLED TestDoctor_RulingCheckRunsForLoadedMission |
| I | size cap > -> >= (exactly 4 MiB) | SURVIVED (boundary untested; harmless, non-blocking) |
| K | start-marker line counts as outside | SURVIVED (marker lines carry no literal in practice; nearly equivalent) |
| M | skill dir filter "mission-" -> "" | SURVIVED: no test proves non-mission skills are excluded; scope would silently widen |
| O | unreadable charter still link-checked | SURVIVED: would double-report (unreadable + link-missing); no test |

## Findings (non-blocking)
1. M survivor: add a rulings_corpus_test case with .claude/skills/other-skill/x.md containing a literal, assert excluded.
2. O survivor: add a test that an absent charter yields ruling-file-unreadable and no ruling-link-missing (the code is right; the guard is unpinned).
3. I survivor: add a 4 MiB exactly / +1 file test (or accept).
4. D-FR-2 cites D-63 needlessly.
5. Per-mission `doctor <name>` still walks skills and reports skill contradictions under mission "rulings" (by design, mildly noisy).
6. Literal matching is byte-exact: a reworded or differently-cased stale claim goes unreported (documented limitation; check is advisory, so not vacuous, but narrow: 3 detectors).

## Cannot change exit code or block arming
Every finding is Severity Note; ExitCode() keys on Drift only (mutant A proves a Drift would be caught); note output is prepended so the last line is unchanged for lane-check.

## Strongest objection to my verdict
Four of fourteen mutants survived, and M and O are real unpinned behaviours that a small executor model could regress without a test failing; also the live run exercised only the absent/present-ledger paths with two seed detectors, so noisiness at scale (many detectors, big charters) is predicted, not measured. I judge these polish, not merge blockers, because the check is report-only and the seeds produce zero false positives on the real repo.

EVALUATION_SCORE: 91/100
EVALUATION_ROUND: 1
EVALUATION_REPORT_PATH: /tmp/fleet_i118/judge-build.md
FEEDBACK_SUMMARY: none blocking; add tests for non-mission skill exclusion and unreadable-charter link skip
EVALUATION_VERDICT: PASS score=91

## Delta review (9bf781b30, a3f6a5088 on top of 1e114ec7c) -- PASS 91 stands (no change)
1. Windows fix: correct and complete as far as can be reasoned without a Windows host. `GOOS=windows go vet ./internal/mission/` rc 0 and `GOOS=windows go test -c` compiles. Per test: CharterPath_Location used "/reg" and Join(Separator,"abs") roots, not absolute on Windows (no volume), so IsAbs/Join assertions diverged; now t.TempDir roots. DisplayPath used Rel output with backslashes against a "/" expectation; now ToSlash on repo-relative and relative paths, with an added native-Join guard. TestDoctor_RulingCheckRunsForLoadedMission wrote a backslash temp dir into a TOML basic string (invalid escape); now ToSlash. Residual: the 5-test attribution is the coordinator's; I could not run Windows. I grepped other ruling tests for native-separator expectations: only rulings_test.go:374 (already "/"-based). Out-of-root absolute paths stay native by design (documented in the comment).
2. Non-Windows behaviour: ToSlash is the identity on Unix, so output is byte-identical; only the Windows rendering changes (to the slash form the design/tests use). Output contract unchanged. Production diff is 8 lines in displayPath only.
3. `go test ./internal/mission/ -count=1` ok; `go test ./internal/mission/ ./cmd/ailang/ -run 'Mission|Doctor|Ruling' -count=1` ok (both packages).
4. Mutant M: CORRECTION to my earlier table. My original M mutant was a bad mutant: the perl substitution without /g replaced the first "mission-" occurrence, which is in a doc comment (rulings_corpus.go:57), so nothing executable changed and "SURVIVED" was my harness error, not a test gap or a -run pattern gap. Re-done on the real condition (`HasPrefix(e.Name(), "")` at :71): KILLED by TestRulingsSkillFiles_WalkDirDepthAndFilter under both -run 'Ruling' and -run 'Rulings' (the `other/SKILL.md` fixture proves exclusion). So finding 1 from the first report is withdrawn. The rename is still a harmless improvement: it makes the test match -run 'Rulings' like its siblings.
Mutant O re-checked on the delta head (real condition, git diff shows 1 line changed): still SURVIVED; stays a non-blocking finding. Mutants I and K unchanged.
Updated verdict: PASS score=91 (no change; removing finding M is offset by confirming O).
