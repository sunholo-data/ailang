# Fleet iteration 142 — Keep record-only verdicts out of the idle fingerprint

Ticket [idle-fingerprint:record-eval-json-moves-head](design_docs/fleet-mission-evidence/iteration141/tickets.json#L515), filed by Stapledon iteration 118. Plan against AILANG dev **162f2dcd1**. The patch spans the shared helper, its existing test, and one changelog fragment.

Controller evidence on Stapledon's `origin/main` at `891c3bc0` shows `head=` selecting `a510c43b` (iteration 119 record). Under the existing pathspecs, that commit and `6fd5f76c` (iteration 118) each show only their record evaluation JSON. The proposed pathspec selects product commit `beb17c4a` on the same history. Stapledon has seven files in this convention. AILANG and World have zero.

## 1. Update the shared helper

In `tools/launchd/lib/idle-fingerprint.sh` at the `head_sha` lookup, replace the comment and add the fourth exclusion immediately before the stderr redirect. This is the resulting block.

```bash
# Keep the exclusions on their own lines. Skip the charter/log, this mission's
# per-iteration verification records in either verification directory, and
# the record-only MISSION-ITER judge verdicts. Ledger status changes still wake
# the mission through the separate ledger read.
head_sha=$(git -C "$work" log -1 --format=%H "origin/$branch" -- . \
  ":(exclude)design_docs/${name}-mission*" \
  ":(exclude)verification/${name}-iter*" \
  ":(exclude)design_docs/verification/${name}-iter*" \
  ':(exclude).ailang/state/evaluations/eval_MISSION-ITER-*_record.json' \
  2>"$tmp/err") || die "head read failed: $(tail -1 "$tmp/err")"
```

Match only `eval_MISSION-ITER-*_record.json` in the measured evaluations location because the writer uses that literal prefix and suffix across iteration numbers. A directory-wide exclusion or a prefix without the record suffix would hide ordinary sprint evidence or future verdict formats, so `eval_*_round_*` and other verdicts outside the measured convention continue to move `head=`. This follows the narrow pathspec precedent in `ad2d4196e`.

## 2. Extend the disposable Git fixture

In `tools/launchd/test_mission_idle_fingerprint.sh`, replace the stale “Both exclusion patterns” comment at line 258 with `# Exercise the record exclusions using commits in a real origin.` Keep the existing arms. After the non-iter verification control sets `h0=$hn` (line 271), add:

```bash
mkdir -p "$MISSION_WORKDIR/.ailang/state/evaluations"
echo '{}' >"$MISSION_WORKDIR/.ailang/state/evaluations/eval_MISSION-ITER-9_record.json"; commit
hr=$(hash) || fail "record evaluation fingerprint read: $(cat "$TMP/herr")"
[ "$hr" = "$h0" ] || fail 'record-only evaluation JSON commit moved hash'
```

The existing check-runs assertion now observes the record commit as the branch tip. Immediately after that assertion and before the `product.txt` change, add:

```bash
# An ordinary sprint verdict is a wake input.
echo '{}' >"$MISSION_WORKDIR/.ailang/state/evaluations/eval_M-FOO_round_1.json"; commit
hn=$(hash) || fail "ordinary evaluation fingerprint read: $(cat "$TMP/herr")"
[ -n "$hn" ] && [ "$hn" != "$h0" ] || fail 'ordinary evaluation JSON commit did not move hash'
h0=$hn
```

Each `commit` stages and pushes a single new file to the fixture origin. The record arm observes the same hash as before its commit. The ordinary arm observes a different nonempty hash and advances the baseline so the existing product change still needs to move it independently. The preceding arms and stub inputs stay in place.

### Mutations

Insert `exclude-record-eval` and `exclude-evaluations-broad` after `exclude-dd-broad` in the line 28 list and add these matching cases. Each `sed` changes one line of the helper copy.

```bash
        exclude-record-eval)
          sed '\|:(exclude)\.ailang/state/evaluations/eval_MISSION-ITER-\*_record\.json|d' "$HELPER" >"$TMP/helper"
          expected_fail='FAIL record-only evaluation JSON commit moved hash'
          ;;
        exclude-evaluations-broad)
          sed 's|:(exclude)\.ailang/state/evaluations/eval_MISSION-ITER-\*_record\.json|:(exclude).ailang/state/evaluations/*|' "$HELPER" >"$TMP/helper"
          expected_fail='FAIL ordinary evaluation JSON commit did not move hash'
          ;;
```

Initialize `expected_fail=''` before the `case` in each loop iteration. After the existing nonzero-rc and `FAIL` checks, capture the first `FAIL` line using the existing `sed -n '/^FAIL /{p;q;}'`. For either new mutant, fail the mutation gate if that line differs from `expected_fail`. Print the captured line in the existing `KILLED` message. A helper read error has a distinct diagnostic and cannot count as either expected kill. The deletion reaches the new record arm first. The broadened pathspec passes the record arm and reaches the ordinary verdict control.

There is no structural assertion counting pathspecs or matching the exclusion block. The existing `excludes` mutant removes every exclusion line and still stops at `FAIL record-only design doc commit moved hash`. Keep the existing driver placement, argv, format, and snapshot assertions.

## 3. Audit of other references

Repository searches covered exclusion pathspecs, the literal record filename convention, and references to the helper, including hidden skill directories.

| Hit | Decision |
| --- | --- |
| `tools/launchd/lib/idle-fingerprint.sh:72` | Update. This is the sole executable idle-fingerprint pathspec. |
| `tools/launchd/test_mission_idle_fingerprint.sh:28,258` | Update the fixture and mutation cases. Its existing patterns are test mutations. |
| `tools/launchd/mission-control.sh:1787,2451` | Keep. Bare pre-check and `--snapshot` call the shared helper. |
| `.claude/skills/mission-control/resources/gate-5-retro.md:21` | Keep. `--record` promotes the snapshot and the prose refers to record paths generically. |
| `.agents/skills/mission-control/resources/gate-5-retro.md:21` | Keep for the same reason. |
| `make/test.mk:125` | Keep. The suite is already registered in the launchd target. |
| `design_docs/fleet-mission-evidence/iteration91/executor-directive.txt`, `iteration126/plan.md`, `iteration126/pr-2431-body.md`, and `iteration141/tickets.json` under the same evidence directory | Keep each historical pathspec example and the original ticket report. |
| `design_docs/fleet-mission-log.md`, its archive, charter, dashboard, index/status files, and other iteration evidence | Keep the historical measurements and ticket references. |
| `design_docs/planned/m-idle-record-degraded-bar.md` and `changelogs/v0.32-current.md` | Keep the separate record-gating design and the prior release notes. |

The mission-control skill trees contain no second pathspec. The `docs/` search found no idle-fingerprint pathspec or helper consumer.

## 4. Add the changelog fragment

Create `changelogs/unreleased/2026-11-26-idle-fingerprint-record-eval-json.md` with this content.

```markdown
### Fixed — Mission idle pre-check woke on record-only evaluation commits (2026-11-26)

Stapledon's per-iteration judge verdicts advanced the mission idle fingerprint's
head and woke otherwise unchanged fires. The shared helper now skips
`.ailang/state/evaluations/eval_MISSION-ITER-*_record.json` when selecting the
fingerprint head. Ordinary sprint verdicts still wake the mission.
[Fleet ticket evidence](https://github.com/sunholo-data/ailang/blob/162f2dcd117f81d66385ef54b056e55b40e891d3/design_docs/fleet-mission-evidence/iteration141/tickets.json#L515).
```

## 5. Executor gates and expected observations

1. `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` — rc 0 and three `PASS` lines. The fixture shows the hash staying stable on the record verdict, the check-runs lookup following the branch tip, and the hash advancing on the ordinary verdict.
2. `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations` — rc 0 and **17/17 KILLED** after the pristine control succeeds. Verify both new `KILLED` lines carry their exact `FAIL` text above. Preserve the command's exit status when capturing output.
3. `shellcheck tools/launchd/lib/idle-fingerprint.sh tools/launchd/test_mission_idle_fingerprint.sh` — rc 0, no diagnostics.
4. `make check-changelog` — rc 0 with the fragment accepted.

The controller runs `make test-launchd-drivers` outside the sandbox, expecting rc 0. For any gate stopped by a sandbox permission or socket denial, record the command and diagnostic as **UNINFORMATIVE UNDER SANDBOX**.

Use Bash 3.2 and BSD-compatible tools with no new environment variables. Use the existing fixture's private Git origin, temporary HOME and state paths for every test write. Leave `$HOME/.ailang/state` untouched, including the fixture default state location that the existing test observes. Acceptance comes from the test output and fixture behavior. The controller owns live rollout and git operations on the worktree.

The controller supplied green baseline results for the test and its 15 mutants. The planner checked baseline ShellCheck and `make check-changelog` (both rc 0). The executor gates above apply to the proposed patch.
