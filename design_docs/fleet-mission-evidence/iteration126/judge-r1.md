# i126 judge r1 — PASS 92/100
Breakdown: correctness 37/40, tests/mutation 23/25, scope/safety 15/15, docs/changelog 9/10, evidence 8/10.

## Commands (rc)
- git diff ec3ad8b99..HEAD --stat/--name-only: 3 files, all under tools/launchd/ and changelogs/unreleased/ (rc 0)
- /bin/bash tools/launchd/test_mission_idle_fingerprint.sh: 3 PASS lines, rc=0
- IDLE_TEST_HELPER=<base helper copy> same test: FAIL "record-only design_docs/verification commit moved hash", rc=1 (fix-alone revert is caught)
- ... --mutations (unpiped): rc=0, all 14 mutants KILLED incl. 3 new (exclude-root-verification, exclude-dd-verification, exclude-dd-broad)
- make test-launchd-drivers: rc=0 (59 suites passed, bash 3.2 syntax OK)
- Real world clone (read-only, origin/dev): base pathspecs give head=a15551c2 (iter415 record commit, touches design_docs/verification/world-iter415 + world-mission*); with the new exclusion head=0e9e8d35. 4299cdb6 (iter414) likewise only touches world-iter414 + world-mission* paths. Defect reproduced on base, absent on head.
- Scratch repo glob probe: pathspec `world-iter*` crosses `/` (git wildmatch without WM_PATHNAME), so also matches world-iter10, world-iteration-foo, world-iter/sub/...
- grep for other consumers hardcoding the exclusion list (tools, .claude, docs): none; mission-control.sh only calls the helper.

## Blocking findings
none

## Non-blocking
- `world-iter*` is slightly broad (matches world-iteration-foo and nested paths) but identical to the pre-existing root `verification/${name}-iter*` rule; the new negative control only covers `world-notes`, not an `iter`-prefixed non-record. The changelog claim "stays narrow" is true for w-* and world-attended-* only.
- Real-world evidence is not in the repo's test (only a synthetic fixture); I reproduced it myself against the world clone.
- Changelog says "Two slots were lost" - unverifiable from here.

## Strongest objection
A glob `world-iter*` under design_docs/verification/ would silently hide a genuine non-record dir like `world-iteration-redesign/` from the wake fingerprint, and no test pins that boundary; a future feature dir named with that prefix would be ignored until another input moves. Risk is low (same already-accepted rule at repo root, naming convention is by-iteration-number), so it does not block.
