# Judge r1, commit 8e78d4ad8 (fleet iteration 120)
Verdict: PASS 93

Blocking findings: none.
Non-blocking: (a) design-doc-creator regression arms only cover nested script path + `-x` siblings (adequate). (b) Mutants cannot detect a pure comment edit (equivalent). (c) .claude/skills/use-ailang refuse row is duplicated in the test (harmless).

1. Fidelity: diff touches only _scope_is_fleet_support (hunk after line 28); _scope_is_harness untouched. Admits 2 design-doc-creator patterns + exactly 16 enumerated .agents/skills/<n>/* arms; no wildcard. Product loop (v1) allow-rows pass for all new paths (test arms). Attended (MISSION_NAME empty) returns early; unchanged.
2. Tests: `bash tools/launchd/test_mission_scope_guard.sh` rc=0. `make test-launchd-drivers` rc=0 (401s, 58 suites, bash3.2 syntax OK).
Mutants (rc=1 means killed), restored from backup after each; tree clean:
| M | mutation | rc | killed by |
| M1 | drop .claude design-doc-creator arm | 1 | fleet .claude ddc script refused |
| M2 | drop .agents ddc arm | 1 | same, .agents |
| M3 | ddc/* -> ddc* | 1 | ddc-x sibling |
| M4 | use-ailang/* -> use-ailang* | 1 | use-ailang-x |
| M5 | delete pr-monitor arm | 1 | pr-monitor allow |
| M6 | trace-debugger arm -> .agents/skills/* | 1 | over-widen row |
| M7 | gemini-cli-helper arm points at .claude | 1 | allow row |
| M8 | eval-analyzer typo | 1 | allow row |
| M9 | benchmark-runner arm also admits benchmark-manager | 1 | exactly-16 loop |
| M10 | .claude ddc arm -> .claude/skills/* | 1 | ddc-x |
| M11 | use-ailang moved into _scope_is_harness | 1 | v1 allow row |
| M12 | ddc moved into harness | 1 | v1 allow row |
Survivors: 0.
3. Charter: the Authority diff adds exactly the 2 ddc + 16 names (programmatic diff of guard arm names vs charter text: identical, 16 each); 9 added lines, nothing else changed.
4. Premise at HEAD: my own re-derivation (/tmp/fleet_i120/j_norm.sh, case-fold + claude/codex/.agents normalisation): all 16 exist in both trees, residual=0 for all 16, none has an only-one-side file. All 16 named skills exist; headless-runner (in the 11-content list) is absent but is not in the 16, irrelevant. Exactly-16 holds.
5. Pinned guard: sourcing origin/dev's pre-push, `_scope_verdict fleet` allows all 5 touched paths (2 changelogs, design_docs/fleet-mission.md, pre-push, test script via _scope_is_harness/support).
6. Changelogs: `scripts/changelog_fold.sh --check` rc=0; both open with `### Changed — ...`, no #/## headings, name format ok; content accurate (names, 2026-07-28 rename, collaboration-hub/trace-debugger identical).
Minor: the fragments' headings lack the optional date suffix shown in README example; not enforced.
