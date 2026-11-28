# Fleet iteration 145 — wake product missions on nightly rig failures

Baseline `9b35c44d0` · ticket `idle-fingerprint:blind-to-nightly-rig-red` · one executor change.

## Design

Read open `nightly-rig` issues from `$MISSION_REPO` and hash their issue numbers and IDs of comments starting with `The nightly `, because tested commits can already be behind the tip when the rig posts a status.
A new issue or failure comment changes the hash, and loop triage comments contribute zero bytes even when posted by the same account.
An open issue with zero matching reports across its opening body and all comments fails the read with a format-drift diagnostic.
Closing the issue removes its rows and changes the hash once, and an empty open-issue list contributes zero bytes to the legacy hash.
Use the fixed label convention with existing repo configuration and paginate both REST reads, failing on request errors, invalid data or incomplete comment totals.

Keep the existing summary byte-for-byte and the snapshot suffix `fire=… at=…` with `at=` last. The registry key would require a Go struct change, plist rendering and reinstall. This uses Bash 3.2, BSD tools and the existing jq dependency. An old matching report can mask later producer-format drift on the same issue because a changed prefix is indistinguishable from a triage note.

## Helper edits

In `tools/launchd/lib/idle-fingerprint.sh`, insert after the check-runs parse and before the count/hash lines:

```bash
# The rig can post red on an older SHA. The open incident follows the branch.
gh api --paginate "repos/$repo/issues?state=open&labels=nightly-rig&per_page=100" \
  >"$tmp/nightly-issues.json" 2>"$tmp/err" || die "nightly-rig issues read failed: $(tail -1 "$tmp/err")"
jq -rs '
  def whole: type=="number" and . == floor;
  if length==0 or any(.[]; type!="array") then error("expected issue pages") else add end
  | if any(.[]; type!="object") then error("expected issue objects") else .[] end
  | select(has("pull_request") | not)
  | if (.number | whole and .>0) and (.comments | whole and .>=0)
      and has("body") and (.body==null or (.body|type)=="string")
    then [.number, .comments, ((.body // "") | startswith("The nightly "))] | @tsv
    else error("invalid nightly issue fields") end' \
  "$tmp/nightly-issues.json" >"$tmp/nightly-issues" 2>"$tmp/err" \
  || die "nightly-rig issues unparseable: $(tail -1 "$tmp/err")"

while IFS=$'\t' read -r issue comment_count opening_report; do
  gh api --paginate "repos/$repo/issues/$issue/comments?per_page=100" \
    >"$tmp/nightly-comments.json" 2>"$tmp/err" || die "nightly-rig #$issue comments read failed: $(tail -1 "$tmp/err")"
  jq -rs --arg issue "$issue" --argjson expected "$comment_count" --argjson opened "$opening_report" '
    if length==0 or any(.[]; type!="array") then error("expected comment pages") else add end
    | if any(.[]; type!="object" or (.id|type)!="number" or (.id|.<=0 or .!=floor) or (.body|type)!="string")
      then error("invalid nightly comment fields") else . end
    | if length!=$expected or (map(.id)|unique|length)!=$expected
      then error("incomplete or changed comment list") else . end
    | map(select(.body | startswith("The nightly ")))
    | if $opened or length>0 then
        "nightly issue:\($issue)", (.[] | "nightly comment:\($issue):\(.id)")
      else error("no report starts with The nightly ") end' \
    "$tmp/nightly-comments.json" 2>"$tmp/err" \
    || die "nightly-rig #$issue comments invalid: $(tail -1 "$tmp/err")"
done <"$tmp/nightly-issues" >"$tmp/nightly" || die 'cannot stage nightly-rig rows'
```

Replace only the hash assignment:

```bash
hash=$(LC_ALL=C sort "$tmp/head" "$tmp/ledger" "$tmp/directives" "$tmp/inbox" "$tmp/red" "$tmp/nightly" | shasum -a 256 | awk '{print $1}') || die 'hash failed'
```

Both requests follow REST next-page links without a `--limit` cap. Enforce the issue's reported comment total before filtering, including unique IDs, so silent truncation and count changes during the read fail like inbox truncation. Count and pagination metadata never enter the hash. A legitimately full final page is valid.

## Fixture and mutations

In `tools/launchd/test_mission_idle_fingerprint.sh`, add these arms to the existing exact-argv `gh` stub before its unknown-argv arm:

```bash
  'api --paginate repos/example/world/issues?state=open&labels=nightly-rig&per_page=100')
    [ "${STUB_NIGHTLY_FAIL:-}" != issues ] || { echo 'HTTP 502' >&2; exit 1; }
    cat "$STUB_DIR/nightly-issues";;
  'api --paginate repos/example/world/issues/392/comments?per_page=100')
    [ "${STUB_NIGHTLY_FAIL:-}" != comments ] || { echo 'HTTP 502' >&2; exit 1; }
    cat "$STUB_DIR/nightly-comments";;
```

Set both fixtures to `[]` in `reset_stub`. Keep the unknown-argv rejection, bookkeeping #99, exact summary assertion at line 289 and existing snapshot/record format assertions.

**(c) Legacy bytes.** Immediately after `h0=${base%% *}`, compute the original checksum from the initial fixture's two populated inputs:

```bash
legacy_head=$("$REAL_GIT" -C "$MISSION_WORKDIR" rev-parse HEAD) || fail 'legacy head read'
legacy_hash=$(printf 'head %s\nledger D-WORLD-1:OPEN\n' "$legacy_head" | LC_ALL=C sort | shasum -a 256 | awk '{print $1}') || fail 'legacy hash read'
[ "$h0" = "$legacy_hash" ] || fail 'empty nightly issues changed the legacy hash'
```

**(a) Red and (b) loop triage.** After the initial bare-run assertion and before the record-exclusion commits, add:

```bash
nightly_issue() {
  printf '[{"number":392,"comments":%s,"body":"%s"}]\n' "$1" "$2" >"$S/nightly-issues"
}
opening='The nightly make test (rig/full) failed on main at 78f390c0.'
nightly_issue 0 "$opening"
nightly_open=$(hash) || fail "new nightly issue read: $(cat "$TMP/herr")"
jq -nc '[range(100) | {id:(.+1),body:"Earlier triage",user:{login:"sunholo-voight-kampff"}}]' >"$S/nightly-page1" || fail 'nightly first page fixture'
report='{"id":7210499888,"body":"The nightly make test (rig/full) failed on main at 468ceb97.","user":{"login":"sunholo-voight-kampff"}}'
triage='{"id":7211245483,"body":"Triage from mission iteration 123.","user":{"login":"sunholo-voight-kampff"}}'
nightly_issue 101 "$opening"
{ cat "$S/nightly-page1"; printf '[%s]\n' "$report"; } >"$S/nightly-comments"
nightly_red=$(hash) || fail "nightly failure read: $(cat "$TMP/herr")"
[ "$nightly_red" != "$nightly_open" ] || fail 'nightly failure comment did not move hash'
nightly_issue 102 "$opening"
{ cat "$S/nightly-page1"; printf '[%s,%s]\n' "$report" "$triage"; } >"$S/nightly-comments"
nightly_triage=$(hash) || fail "nightly triage read: $(cat "$TMP/herr")"
[ "$nightly_triage" = "$nightly_red" ] || fail 'nightly triage comment moved hash'
[ "$nightly_open" != "$h0" ] || fail 'new nightly issue did not move hash'
reset_stub
nightly_closed=$(hash) || fail 'closed nightly issue read'
[ "$nightly_closed" = "$h0" ] && [ "$nightly_closed" != "$nightly_red" ] || fail 'closing nightly issue did not restore legacy hash'
```

**(d) Failed reads.** After the above, use `bad` to assert nonzero rc, empty stdout and a diagnostic. With a fresh reset before each case, test `STUB_NIGHTLY_FAIL=issues bad`, malformed issue pages (`not json`, `{}`, empty stream), and a malformed issue row with missing `number`. For comments, seed `nightly_issue 1 "$opening"` after each reset, then test `STUB_NIGHTLY_FAIL=comments bad`, the same malformed page cases, and a comment with no body. Seed `nightly_issue 101 "$opening"` with only `nightly-page1` as the comments response and require `bad` with `incomplete or changed comment list` on stderr. Seed `nightly_issue 0 'Changed producer format'` with `[]` comments and require `bad` with `no report starts with The nightly` on stderr. Reset again before running the existing head assertions.

Append these cases and names to the `--mutations` loop. The red-comment assertion comes before the issue-open assertion so the missing-input mutant dies at the specified check.

```bash
        nightly-omit)
          sed '/^hash=/s/ "\$tmp\/nightly"//' "$HELPER" >"$TMP/helper"
          expected_fail='FAIL nightly failure comment did not move hash'
          ;;
        nightly-all-comments)
          sed '/map(select(\.body | startswith("The nightly ")))/d' "$HELPER" >"$TMP/helper"
          expected_fail='FAIL nightly triage comment moved hash'
          ;;
```

## Consumer audit

- `tools/launchd/mission-control.sh:1771–1792,2450–2456` — keep. Pre-check reads the first token and the anchored terminal `at=`, and snapshot checks rc/file existence and logs the line. `grep -n 'red='` hits only `deferred=` at 227 and 242.
- `.claude/skills/mission-control/resources/gate-5-retro.md:21,27–29` and its identical `.agents/skills/mission-control/` counterpart — keep the invocation and policy. Update both input lists by changing `red checks on the branch tip}` to `red checks on the branch tip, open nightly-rig issues and their failure reports}`.
- `docs/` — no helper or summary-format hits. Other skill hits on `check-runs` are independent CI probes and stay as written.
- `design_docs/fleet-mission-log.md:701,756,812,866,936,997,1050` and `fleet-mission-status-archive.md:775,779` — keep captured summary lines. Keep historical references in the fleet charter, dashboard, indexes, log/archive and `design_docs/fleet-mission-evidence/` (including snapshot examples and the iteration 91 five-input report). Keep prior proposals `m-idle-record-degraded-bar.md` and `sprint-plan-codex-capacity-transient.md`.
- Exact summary copies under `design_docs/fleet-mission-evidence/` — keep each:
  - `iteration126/executor-report.md:37`, `iteration127/idle-judge.md:49`, `iteration128/idle-judge.md:47–48`, `iteration129/idle-judge.md:45`, `iteration133/idle-judge.md:11`.
  - `world-idle-snapshots.txt` in iterations 127–132, `iteration133/world-424-426.txt:4,9,14`, `iteration134/world-427.txt:18`, `iteration135/world-idle-exits.txt:1–2`.
  - `tickets.json` in iterations 127, 128, 133–135 and 137–141.
- `changelogs/v0.32-current.md:974–981,5084–5089` and the 2026-11-26 record-eval fragment — keep historical entries. The executor adds the fragment below.

## Changelog fragment

`changelogs/unreleased/2026-11-27-idle-fingerprint-nightly-rig.md`

```markdown
### Fixed — Nightly rig failures wake idle product missions (2026-11-27)

A failed nightly run can report against a commit behind the branch tip. The
idle fingerprint now includes open nightly rig issues and their failure reports,
so a new red night wakes the mission while follow-up triage comments leave the
fingerprint unchanged. [Stapledon nightly rig issue](https://github.com/sunholo-data/stapledons-godot/issues/392)

The fleet ticket signature is idle-fingerprint:blind-to-nightly-rig-red.
```

## Executor gates

- `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` — rc 0, existing PASS sections and all nightly observations above.
- `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations` — rc 0, all 23 KILLED (21 existing plus 2), including the exact first FAIL messages above.
- `shellcheck -s bash tools/launchd/lib/idle-fingerprint.sh tools/launchd/test_mission_idle_fingerprint.sh` — rc 0, zero findings.
- `make check-changelog` — rc 0, new fragment passes fold validation.
- Controller outside the sandbox runs `make test-launchd-drivers` — expected rc 0.

All acceptance checks observe stdout, stderr, rc, hashes or files inside the test's disposable fixture. Fixture HOME, TMPDIR and state overrides keep the real `$HOME/.ailang/state` untouched. Label any sandbox permission or socket failure **UNINFORMATIVE UNDER SANDBOX**. If a gate is backgrounded, poll its output file in a bounded `date +%s` loop and collect its exit status.
