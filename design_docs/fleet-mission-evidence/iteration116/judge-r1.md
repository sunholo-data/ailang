# Judge r1 — iteration 116, D-FLEET-18 scope arm (48ba7ed54)
Score 94/100 — PASS (>=70)
- Correctness 29/30: pre-push adds ONE case line to _scope_is_fleet_support with the two exact paths. Old-vs-new verdict diff over 4 missions (fleet,v1,motoko,docs) x 6 paths: only change is fleet allow on the 2 paths; no product verdict changed. Core paths stay refused (core check runs first).
- Test strength 21/25: suite rc 0 (run unpiped). 13 new arms. 8 own mutants, 6 killed, 2 survived (below).
- Scope 20/20: exactly 4 files (hook, its test, charter Authority line, changelog); nothing under tools/eval/. Pinned origin/dev guard admits all 4 paths for fleet (verdict empty).
- Landing 14/15: only pre-push exists in core.hooksPath (tools/launchd/githooks), so hooksPath=/dev/null skipped nothing beyond the guard itself, which I ran manually on the paths. Docked 1 for the bypass flag.
- Docs 10/10: fragment says ruled A 2026-11-16, not PROPOSED; Authority line matches hook; "13 regression arms" is correct (13 added). Claim that probe fixes follow separately matches the ledger.
Blocking: none.
Non-blocking: (1) no arm for case/near-name variants (Motoko_..., probe_sh) so [mM] and ?-for-dot mutants survive; both widen by at most one non-existent filename, not material. (2) Changelog calls them "support paths, not harness"; accurate to the code.
Mutants (own): m1 glob tools/eval/*motoko_... KILLED (xmotoko arm); m2 path2->tools/eval/* KILLED (qwen38 arm); m3 [mM] first letter SURVIVED; m4 move both to _scope_is_harness KILLED (motoko/v1 allow arms); m5 drop both KILLED; m6 suffix glob on path2 KILLED; m7 arm placed in _scope_is_core KILLED; m8 '.'->'?' in path1 SURVIVED. All restored, cmp silent. Drill ran in a throwaway worktree (removed).
Strongest objection: the test only pins a fixed list of negatives, so a pattern widening by one character class/wildcard on a single char is not caught; a table-driven exactness check (path with each char mutated) would close it. Immaterial for landing.
