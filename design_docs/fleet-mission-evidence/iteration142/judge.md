# Fleet i142 judge report (commit b79be251f)

Score: 88/100 (correctness 37/40, tests 20/25, scope/safety 18/20, docs 13/15). Verdict PASS. LAND.

## Blocking findings
None.

## Non-blocking findings
1. Test negative control is thin. The only ordinary-verdict control is eval_M-FOO_round_1.json, so the exclusion's `_record.json` suffix is not pinned. Mutants that widen it SURVIVE the plain test (table below), and the committed 17-mutant list has no mutant for them. No known real file is affected: stapledon's only `eval_MISSION-ITER-*` files are the `_record.json` ones.
2. Evidence files under design_docs/fleet-mission-evidence/iteration142/ were not verified (executor claims untrusted); they do not affect behaviour.
3. Changelog fragment is accurate and brief; it does not state that other record-file layouts (other missions) are not covered. That is fine, since the ticket only asked for stapledon's.

## Strongest objection
The exclusion is a hardcoded stapledon-specific filename convention, not keyed to ${name}. Any mission that writes a differently named record verdict still never idles. The glob also leans on the `*_record.json` suffix, which the tests do not pin: `eval_*_record.json` or `*_record.json` pass all tests. Mitigation: the test's mutants do pin the `.ailang/state/evaluations/` prefix and the ordinary-round negative control. Real-clone repro confirms behaviour is correct today.

## Measurements
- `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` rc=0, 3 PASS lines.
- `... --mutations` rc=0: 17 MUTATION lines, 17 KILLED (list has 17 names: 15 old + exclude-record-eval + exclude-evaluations-broad). New ones: exclude-record-eval killed by "FAIL record-only evaluation JSON commit moved hash"; exclude-evaluations-broad killed by "FAIL ordinary evaluation JSON commit did not move hash". Both have an expected_fail check on the exact message.
- `make test-launchd-drivers` rc=0, "61 passed, 0 failed", idle-fingerprint test PASS in it, bash 3.2 syntax OK.
- shellcheck on idle-fingerprint.sh and test_mission_idle_fingerprint.sh: rc=0, no output (old helper also rc=0). `bash -n` rc=0. No new bashisms (pathspec is a plain single-quoted arg).
- Other consumers: grep of tools/launchd/mission-control.sh, .claude/.agents skills mission-control: driver only invokes the helper (lines 1787, 2451); gate-5-retro.md mentions idle-fingerprint but holds no copy of the exclusion list. No duplicate list found. MISSION-ITER appears only in the helper and test.

## Real-clone repro (stapledon, `git fetch -q origin main` rc=0, remote-tracking only)
- OLD pathspecs: head = a510c43b "mission(stapledon): iteration 119 record..." and `git show --stat` includes .ailang/state/evaluations/eval_MISSION-ITER-119_record.json, i.e. the bug reproduced.
- NEW (+exclusion): head = beb17c4a "docs(r1): 7aw landed ... judge round 2" touching eval_R1-7AW-PROFILE-SECRETS_round_2_sonnet.json and sprint docs: the real ordinary verdict, so it still moves head.
- origin/main has 7 eval_MISSION-ITER-<N>_record.json files (106,112,115-119), all matched by the pattern; ~230 other eval files (round_N etc.) are untouched by it.

## Mutation table (scratch copies via IDLE_TEST_HELPER; plain test)
| mutant | result |
|---|---|
| (a) helper reverted to parent 162f2dcd1 | KILLED: "FAIL record-only evaluation JSON commit moved hash" (intended assertion) |
| c: exact name eval_MISSION-ITER-1_record.json | KILLED: same assertion |
| d: drop `.ailang/state/evaluations/` prefix (root-anchored) | KILLED: same assertion |
| b: drop `_record.json` suffix (`eval_MISSION-ITER-*`) | SURVIVED |
| e: `eval_*_record.json` | SURVIVED |
| f: `eval_MISSION-ITER-*_*.json` | SURVIVED |
| g: `:(exclude)*_record.json` (drops dir and prefix) | SURVIVED |
| committed: exclude-record-eval, exclude-evaluations-broad | KILLED (see above) |

Survivors b/e/f/g all widen the exclusion only for files no fixture or real stapledon file has (e.g. eval_MISSION-ITER-9_round_1.json); cheap fix would be a second negative control of that shape. Non-blocking.
