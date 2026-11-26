# Fleet iteration 143 — Pin the idle fingerprint’s evaluation exclusion

Carry forward PR #2473 from 04f307466 on fleet/i142-idle-fp-record-eval-json.
For idle-fingerprint:record-eval-json-moves-head, edit only the test and the existing
changelog fragment. Keep the production helper byte-for-byte identical. The controller
owns branch and PR operations and re-judging. The required dev test is red outside this scope.

## Plain fixture

In tools/launchd/test_mission_idle_fingerprint.sh, insert both arms immediately after
the ordinary eval_M-FOO_round_1.json assertion and its h0 assignment, before the
product.txt commit. Use the existing fixture directory, stub responses and commit helper.
Each commit adds exactly the one JSON file shown.

~~~bash
echo '{}' >"$MISSION_WORKDIR/.ailang/state/evaluations/eval_MISSION-ITER-9_round_1.json"; commit
hn=$(hash) || fail "MISSION-ITER round evaluation fingerprint read: $(cat "$TMP/herr")"
[ -n "$hn" ] && [ "$hn" != "$h0" ] || fail 'MISSION-ITER round evaluation JSON commit did not move hash'
h0=$hn

echo '{}' >"$MISSION_WORKDIR/.ailang/state/evaluations/eval_M-FOO_record.json"; commit
hn=$(hash) || fail "non-MISSION-ITER record evaluation fingerprint read: $(cat "$TMP/herr")"
[ -n "$hn" ] && [ "$hn" != "$h0" ] || fail 'non-MISSION-ITER record evaluation JSON commit did not move hash'
h0=$hn
~~~

The round control kills b and f. The ordinary record control kills e and g. The
original record-only arm, its branch-tip assertion and the ordinary round control
stay ahead of both, preserving the first failures for the existing evaluation mutants.
The next assertion compares the product commit’s h1 to the latest h0. The working
tree charter check uses h1, then the ledger flip recomputes h0 from fp before
the remaining h0 comparisons. No later check needs a specific earlier h0.

## Mutation loop

Insert the four names below into the --mutations list after exclude-evaluations-broad,
in the order shown. Add these case arms and keep the existing exact first-FAIL check.

~~~bash
exclude-record-eval-no-suffix)
  sed 's|:(exclude)\.ailang/state/evaluations/eval_MISSION-ITER-\*_record\.json|:(exclude).ailang/state/evaluations/eval_MISSION-ITER-*|' "$HELPER" >"$TMP/helper" || fail "mutant $mutant sed failed"
  expected_fail='FAIL MISSION-ITER round evaluation JSON commit did not move hash'
  ;;
exclude-record-eval-any-mission)
  sed 's|:(exclude)\.ailang/state/evaluations/eval_MISSION-ITER-\*_record\.json|:(exclude).ailang/state/evaluations/eval_*_record.json|' "$HELPER" >"$TMP/helper" || fail "mutant $mutant sed failed"
  expected_fail='FAIL non-MISSION-ITER record evaluation JSON commit did not move hash'
  ;;
exclude-record-eval-any-verdict)
  sed 's|:(exclude)\.ailang/state/evaluations/eval_MISSION-ITER-\*_record\.json|:(exclude).ailang/state/evaluations/eval_MISSION-ITER-*_*.json|' "$HELPER" >"$TMP/helper" || fail "mutant $mutant sed failed"
  expected_fail='FAIL MISSION-ITER round evaluation JSON commit did not move hash'
  ;;
exclude-record-any-dir)
  sed 's|:(exclude)\.ailang/state/evaluations/eval_MISSION-ITER-\*_record\.json|:(exclude)*_record.json|' "$HELPER" >"$TMP/helper" || fail "mutant $mutant sed failed"
  expected_fail='FAIL non-MISSION-ITER record evaluation JSON commit did not move hash'
  ;;
~~~

Replace the combined unchanged check with a check of the edited file. Initialize
the shell variable mutation_target=helper for each mutant. Set it to driver in
the six driver-only arms: wrong-record, keep-changed, fail-closed, age, dry-run
and snapshot-keeps-stale. After the case and before running the fixture, use:

~~~bash
if [ "$mutation_target" = helper ]; then
  if cmp -s "$HELPER" "$TMP/helper"; then fail "mutant $mutant sed did not apply"; fi
elif cmp -s "$DRIVER" "$TMP/driver"; then
  fail "mutant $mutant unchanged"
fi
~~~

Read-only verification at 04f307466 confirmed that BSD sed changes the helper
for all 11 existing helper mutants and changes exactly the exclusion line for
each of the four new ones. Expect 21 KILLED lines in total, including these four.

~~~text
MUTATION exclude-record-eval-no-suffix KILLED: FAIL MISSION-ITER round evaluation JSON commit did not move hash
MUTATION exclude-record-eval-any-mission KILLED: FAIL non-MISSION-ITER record evaluation JSON commit did not move hash
MUTATION exclude-record-eval-any-verdict KILLED: FAIL MISSION-ITER round evaluation JSON commit did not move hash
MUTATION exclude-record-any-dir KILLED: FAIL non-MISSION-ITER record evaluation JSON commit did not move hash
~~~

## Changelog

Append one sentence to changelogs/unreleased/2026-11-26-idle-fingerprint-record-eval-json.md.

> The idle fingerprint tests now verify that MISSION-ITER round verdicts and ordinary sprint record verdicts still wake the mission.

## Executor gates

1. Run /bin/bash tools/launchd/test_mission_idle_fingerprint.sh. Expect rc 0
   and exactly three PASS lines covering the driver arms, fire-start snapshot and helper.
2. Run /bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations.
   Expect rc 0 and 21 MUTATION … KILLED lines, including the exact lines above.
3. Run shellcheck tools/launchd/lib/idle-fingerprint.sh tools/launchd/test_mission_idle_fingerprint.sh.
   Expect rc 0.
4. Run make check-changelog. Expect rc 0.
5. For each of b, e, f and g, apply its listed sed to the pristine helper into
   a worktree-local scratch copy. Check sed succeeded and cmp shows a difference.
   Run the plain test as IDLE_TEST_HELPER="$scratch/helper" /bin/bash
   tools/launchd/test_mission_idle_fingerprint.sh with stdout and stderr captured
   in scratch. Expect rc 1 and exactly one line beginning with `FAIL `, byte-for-byte equal to
   that mutant’s expected_fail. This exercises both controls in isolation from
   the mutation runner. Delete scratch after verification.
6. The controller runs make test-launchd-drivers outside the sandbox. Expect rc 0.

Use Bash 3.2 and BSD sed/grep. Reuse IDLE_TEST_HELPER and add no environment
variables. Keep test writes and
Git mutations inside the disposable fixture and validation scratch inside the worktree.
Preserve the fixture’s private HOME and its read-only observation of the default
state path. Leave the caller’s $HOME/.ailang/state untouched. Treat a gate
blocked by a sandbox permission or socket denial as UNINFORMATIVE UNDER SANDBOX.
