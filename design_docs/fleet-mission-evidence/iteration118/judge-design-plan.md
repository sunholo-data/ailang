# Judge: M-FLEET-RULINGS-LEDGER design (R4.1) + sprint plan, fleet i118 (judge2)
Subject 96c837841, origin/dev now ad1fdac57 (#2396 merged). Worktree /Users/voightkampff/.ailang-driver-pin/.fleet-i118-judge2.

## Verdict: PASS, 82/100 (70 = pass). No blocking defect in the design. One must-fix-before-build item in the plan (F1, one line).

## Q1 Ruling coverage
- One ledger: design_docs/fleet-rulings.md, in the existing decision-ledger format.
- Charter link: section 6 gives verbatim text, "nothing in this charter overrides one". It covers the five in-repo charters, plus the template and Gate 0 in the plan. World and stapledon are in other repos; the doc names them as an attended follow-up, and the doctor reports the gap as a note.
- Doctor check that REPORTS: `CheckRulings`, all findings `Note`.
- Extras beyond the ruling: six finding kinds, a 64-line cap, supersession, and the summary-line change in `mission_cmd.go`. The summary line is a needed consequence of notes (see Q2 item 7). The rest is bounded machinery, not enforcement. R1/R2's provenance and `require` machinery was removed. The template edit is small and sensible.
- Report-only is consistent with "cannot override". The ruling asks for reporting. "Cannot override" is a documentary rule (the link text) backed by the report. Making notes gate would create the arming-block risk (lane-check.sh:49-50 -> mission-arm.sh:30-36), which the design rightly avoids.

## Q2 Spot-checks (commands run; 12 claims, 0 false in the design body)
1. doctor.go: `:136` `const sharedDriverRepo`, `:151` `Doctor`, `:154` `DoctorWith(` , `:294` `}`, `:295` sort, `:296` return. All TRUE (sed output).
2. registry.go: `:90` `Root()`, `:251` `m.root = abs`, `:271` `LoadFile` called from `Load`. TRUE. The comment at `:88` still cites ":245-246", stale but harmless (the plan notes it).
3. HasDrift counts only Drift (`doctor.go:118-120`). TRUE.
4. mission-lane-check.sh:49-50: exit-code keyed, `tail -1` of the filtered output. TRUE. I re-ran stub binaries through the exact expression:
   - append order (DRIFT then note, rc 1) -> `dead: [note] rulings ruling-contradiction y`
   - prepend order (note then DRIFT, rc 1) -> `dead: [DRIFT] w env-source-drift X`
   - two notes then summary, rc 0 -> `ok: 1 mission(s): no drift`
   The plan's P8b finding, that the design's append hides the drift reason on exit 1, is REAL. The prepend correction is right and is a genuine catch the design missed. Arming is unaffected either way.
5. mission_cmd.go: `:249` one-mission registry, `:255` `len(rep.Findings)==0`, summary printed only then. TRUE. So the summary change is required for exit 0 with notes.
6. mission-arm.sh:30-36 refuses on rc != 0. TRUE (sed). test_mission_lane_check.sh:70,80 stub `ailang` and assert the summary string. TRUE.
7. scripts/mission_decisions.sh on a ledger built from the design's section 1 seed (header, separator, 2 rows, detector block outside the markers): `decision ledger valid: 2 rows`, rc 0. Control with status PARKED: `invalid status for D-FR-1: PARKED`, rc 1. TRUE. mission_answer.sh keeps the `n != 6` row check and "supersede it with a new ID instead" refusal (lines found by grep). TRUE.
8. #2396 test hunks (git diff 950a0cc54 ad1fdac57): doctor_test.go and driver_agreement_test.go changed comments only; no assertion on Findings. TRUE (answers quorum R4 glm). golden_live_test.go uses by-name `findingsOfKind` lookups only (lines 337/361/380 region read). TRUE. No test asserts `len(rep.Findings)` (grep returned nothing, and the same grep pattern is used with positive results elsewhere in P7).
9. Write scope: pre-push `_scope_is_harness` covers `internal/mission/*` (incl. testdata), `cmd/ailang/mission*` (incl. the new `mission_doctor_print_test.go`), `.claude/skills/mission-*`, `.agents/skills/mission-*`, `design_docs/fleet-mission*`. `_scope_is_fleet_support` covers `design_docs/*`, `changelogs/*`, `.ailang/state/sprints/*`. Every file in the plan is inside. TRUE. `missions/*` is also harness scope but untouched.
10. Charter/registry facts: 7 `doc` values relative; 5 `repo = sunholo-data/ailang`; world and stapledon workdir charters exist at ~/dev/sunholo-data/{ailang-world,stapledons-godot}/design_docs. TRUE.
11. Gate 0 file line 4 is "Attended setup:" as the plan says; the mirror exists. TRUE. `tools/launchd/sync-agents-skills.sh` exists. TRUE. `make test-launchd-drivers` (make/test.mk:136) and `test-changed` (:450) exist. TRUE. `make check-changelog` exists but covers only the root CHANGELOG.md index (code-health.mk:219).
12. Sprint JSON parses; 7 features M1..M7, `design_doc`/`sprint_plan` paths right, status not_started. TRUE.
- Doc-quality note: the Verification Log is mostly reproducible. I did not re-derive V1..V33 individually.

## Q3 Seed detectors
- Safe on current dev: origin/dev `missions/world.toml` has 0 hits (ad1fdac57 is an ancestor of origin/dev, confirmed by git merge-base). A scan of origin/dev missions/*.toml, all mission-* skill .md (both mirrors) and the five in-repo charters finds `WORLD IS A FORK` only in design_docs/fleet-mission.md (1, inside the D-FLEET-25 ledger row, excluded by the charter ledger-block rule). `generator≠judge guard (HARD` is 0 everywhere. The world and stapledon workdir charters on this machine: 0 for the fork literal, 0 for `fleet-rulings.md` (so `ruling-link-missing` x2 is expected).
- Non-vacuous: 663429dba and 950a0cc54 `missions/world.toml` -> 1 hit each; `git show 349e311b9^:...gate-3-route.md` -> 1 (line 242 within the snapshot range 240-245); current gate-3-route.md lines 255-260 contain `generator≠judge` (1). The matcher sees the bytes on both sides. Known-positive controls present.
- Net: the live doctor will report 0 `ruling-contradiction` and 2 `ruling-link-missing` until the world/stapledon charters are linked.

## Q4 Sprint plan executability
Good: exact types and signatures, rules numbered with a test each, a named mutant per test, acceptance commands that exist, run unpiped, mutate via scratchpad copy, and a P8b-driven prepend with a test (`TestDoctor_RulingFindingsPrecedeMissionFindings`) that kills the append mutant. M7 carries the full done-gate: go tests with the `Mission|Doctor|Ruling` regex, `make test-launchd-drivers`, doctor-layer healthy and degraded, driver-layer healthy and degraded dry-runs, plus the live-run gate. Executor is haiku, one milestone per run. M1 (~380 LOC) and M3 (~430 LOC) are the heaviest; M1 packs 8 test cases with 5 mutants and may need a retry, but is sized.

## Findings
Blocking (fix before building):
- None in the design. Plan F1 is a must-fix-but-trivial edit (below).

F1 (must fix, one line; plan M6 step 4). The snapshot command `cp missions/world.toml .../world.pre-2396.toml` captures the PRE-fix file only if the executor's tree is pre-#2396. #2396 is now merged (ad1fdac57) and worktrees branch from origin/dev, so an executor on current dev gets a 0-hit file and `TestRulingsSeedQuietOnActivation` (expects 1) fails, inviting a weakened test. Use `git show 950a0cc54:missions/world.toml > ...world.pre-2396.toml`; `c34c409c3` is present locally, but prefer `git show ad1fdac57:missions/world.toml` for the post file.

Non-blocking:
- F2. Stale #2396 state in the design (Dependencies, section 7, Success criteria "With #2396 unmerged ... one ruling-contradiction") and plan M7 ("ruling-contradiction ... world.toml:3 while #2396 is unmerged"). #2396 is merged, so the correct live expectation is 0 `ruling-contradiction`, 2 `ruling-link-missing`. The prediction is hedged with "else 0", and the live-run gate corrects mismatches, but the M7 doctor-layer expectation line would mislead a small model. Reword it before M7.
- F3. The design text (append, "Call site, exactly") and the plan (prepend) disagree until M4 edits the doc. Disclosed, but a reader of R4.1 alone gets the wrong order. Fix the doc now, not at M4.
- F4. `CheckRulings` iterates all `missions` for the registry/charters scope but the root is derived only from missions that have one. A mixed slice (hand-built plus loaded) would emit `ruling-file-unreadable` for hand-built missions (m.Path "fixture:x", doc "d.md"). Production (`Load`) never mixes, and M4's test appends a loaded mission to `newFleet`'s hand-built ones but only asserts the ledger-absent early return, so it does not exercise this. Specify skipping missions with empty `Root()` in scopes, or accept and note it.
- F5. M5 acceptance runs `TestCLIReferenceMatchesTable` expecting the help-line change not to need doc regeneration. I grepped: the help string appears only in `mission_cmd.go`, so likely fine, but the claim is untested by me.
- F6. Whole-ledger-fail on any malformed line means one bad line silences all rulings. Chosen deliberately and mitigated by T7a (CI parses the live ledger); the failure is loud (one named note). Acceptable.
- F7. Notes appear on every `mission doctor <name>` lane-check run for the whole skills corpus; only visible in full output. Noted in the plan; fine.

## Quorum disposition honesty
Honest. Round-4 objections were both about existing-test tolerance and write scope, plus one parser edge. The controller measured the premises first (V31/V32/V33). I independently confirmed V31 (comment-only hunks) and V32 (by-name assertions) and the write scope (V33). The applied edits are: a merge-order sentence, an A1 wording change, a quoted-lines citation, a malformed-row rule plus T4 case, and a note to use WalkDir. None changes direction. The malformed-row rule restates the already-stated "whole ledger fails" principle. The one weak point is procedural: R4.1's additions (the malformed-row rule and T4 case) were never re-reviewed; they are small and verbatim from the rejecting reviewers, so the carve-out is defensible. The later P8b prepend is a post-quorum design change, but it is disclosed, evidence-backed and strictly safer.

## Strongest objection to my own verdict
I verified the plan's mechanics and the claims, but did not execute any milestone, so compile-level problems (the `rulingsFixture` TOML needing every required field, `golangci-lint` `//nolint:gosec` rules, M3's ~430 LOC for a haiku-class executor) are untested. The design's ruling-detection power is also thin: literal forbids over three scopes seeded with two rulings, one of which has now been fixed, so the shipped check will likely report only `ruling-link-missing` x2 on day one. That is faithful to the ruling's "reporting" ask, but if the reviewer's bar is "demonstrably catches the class", the value is mostly structural, and the seed D-FR-2 detector guards only one retired phrase.
