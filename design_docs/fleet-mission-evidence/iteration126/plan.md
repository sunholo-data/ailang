# Iteration 126 plan: idle fingerprint ignores design_docs/verification records

Ticket `driver:idle-fingerprint-head-ignores-design-docs-verification` (from world, 2 slots lost).
Scope: `tools/launchd/**` + `changelogs/unreleased/**` ONLY. Bash 3.2 / BSD tools. Do not commit
until every check in step 4 is green. The planner tested this whole diff in a scratch copy of
tools/launchd (dev ec3ad8b99): plain rc=0, --mutations rc=0 (16/16 killed), shellcheck rc=0.

## 1. Helper: `tools/launchd/lib/idle-fingerprint.sh` (lines 72-77)
Replace lines 72-73 (comment) and add one exclusion line after line 76. Result:
```bash
# Keep the exclusions on their own lines: the charter/log and this mission's per-iteration
# verification records (root verification/ and design_docs/verification/, both in use) are
# records. Reading ledger statuses separately lets an attended ruling still wake us.
head_sha=$(git -C "$work" log -1 --format=%H "origin/$branch" -- . \
  ":(exclude)design_docs/${name}-mission*" \
  ":(exclude)verification/${name}-iter*" \
  ":(exclude)design_docs/verification/${name}-iter*" \
  2>"$tmp/err") || die "head read failed: $(tail -1 "$tmp/err")"
```
KEEP the root `verification/${name}-iter*` line. World has 22 such dirs in its history, and other
repos may still write there. Change nothing else in the helper.

## 2. Test: `tools/launchd/test_mission_idle_fingerprint.sh`
### 2a. Positive assertion and negative control: insert right after line 260
(`[ "$(hash)" = "$h0" ] || fail 'record-only verification commit moved hash'`):
```bash
mkdir -p "$MISSION_WORKDIR/design_docs/verification/world-iter9"
echo record >"$MISSION_WORKDIR/design_docs/verification/world-iter9/x.txt"; commit
[ "$(hash)" = "$h0" ] || fail 'record-only design_docs/verification commit moved hash'
# Negative control: only THIS mission's -iter dirs are records; a sibling dir is not.
mkdir -p "$MISSION_WORKDIR/design_docs/verification/world-notes"
echo note >"$MISSION_WORKDIR/design_docs/verification/world-notes/x.txt"; commit
hn=$(hash); [ -n "$hn" ] && [ "$hn" != "$h0" ] || fail 'non-iter design_docs/verification commit did not move hash'
h0=$hn
```
`h0=$hn` re-bases the baseline, so the existing check at old line 265 ("non-record head did not
move hash") still measures the product.txt commit alone and does not pass for free.

**Negative-control decision: a non-iter path under design_docs/verification MUST move the hash.**
Only `<name>-iter*` dirs are per-iteration loop records. World's real tree also holds
`design_docs/verification/w-*` (feature-sprint verification, e.g. `w-race-gate-blindspot`) and
`world-attended-*` (attended evidence). Those are new work or a human's work, and either should wake the loop. Waking
by mistake costs one fire. Staying idle by mistake hides work, which is worse. So the exclusion is
deliberately narrow, and this control pins it.

### 2b. Mutants: line 28 loop list and the `case` at line 30
Line 28 becomes:
```bash
    for mutant in excludes exclude-root-verification exclude-dd-verification exclude-dd-broad wrong-record keep-changed fail-closed age limit pending store-pin dry-run fire-check record-as-snapshot snapshot-keeps-stale; do
```
Insert these three arms right after line 31 (`excludes) sed '/:(exclude)/d' ...`). Keep `excludes`.
```bash
        exclude-root-verification) sed '/(exclude)verification\//d' "$HELPER" >"$TMP/helper" ;;
        exclude-dd-verification) sed '/(exclude)design_docs\/verification\//d' "$HELPER" >"$TMP/helper" ;;
        exclude-dd-broad) sed 's|(exclude)design_docs/verification/${name}-iter\*|(exclude)design_docs/verification/*|' "$HELPER" >"$TMP/helper" ;;
```
Each mutant changes exactly one helper line (checked with `sed ... | diff`). `(exclude)verification/`
does not match `(exclude)design_docs/verification/`, so the root mutant deletes only line 76. Each
mutant has exactly one killer:
| mutant | first FAIL (observed in scratch) |
|---|---|
| excludes | `record-only design doc commit moved hash` |
| exclude-root-verification | `record-only verification commit moved hash` |
| exclude-dd-verification | `record-only design_docs/verification commit moved hash` (new) |
| exclude-dd-broad | `non-iter design_docs/verification commit did not move hash` (new negative control) |
The executor must check that the `--mutations` output shows these exact FAIL texts. If a mutant is
killed by a different message, the assertion is not the sole killer: stop and report.

## 3. Changelog: `changelogs/unreleased/2026-11-19-idle-fingerprint-design-docs-verification.md`
```markdown
### Fixed — mission idle pre-check woke on design_docs/verification record-only commits (2026-11-19)

`tools/launchd/lib/idle-fingerprint.sh` excluded a mission's per-iteration records from `head=` only
under the repo-root `verification/<name>-iter*`. World now writes them under
`design_docs/verification/<name>-iter*`, so every record-only commit (e.g. world iter 414,
`4299cdb6`) moved `head=` and the driver ran a full fire on unchanged state. Two slots were lost.
The helper now excludes both locations. The exclusion stays narrow: other `design_docs/verification/`
dirs (feature `w-*`, `world-attended-*`) still wake the loop. The idle-fingerprint test adds the
new record arm, a non-iter negative control and per-exclusion mutants. Ticket
`driver:idle-fingerprint-head-ignores-design-docs-verification`.
```

## 4. Verification: run each and report its rc (all must be 0)
Run from the worktree root `/Users/voightkampff/.ailang-driver-pin/fleet-i126-wt`:
1. `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh` prints 3 PASS lines.
2. `/bin/bash tools/launchd/test_mission_idle_fingerprint.sh --mutations`: 16 `MUTATION ... KILLED`
   lines, matching the table in 2b. It takes a few minutes. Run it unpiped, or capture
   `${PIPESTATUS[0]}`.
3. `make test-launchd-drivers` (runs bash-3.2 `-n` on `tools/launchd/lib/*.sh` and the suite).
   Run it unpiped.
4. ShellCheck: no Makefile target covers these files (`shellcheck-autopush` covers only the
   auto-push hooks), so run it directly:
   `shellcheck tools/launchd/lib/idle-fingerprint.sh tools/launchd/test_mission_idle_fingerprint.sh`.
   Baseline rc=0 on dev, and it must stay 0.
5. `make check-changelog` (fragment shape gate).
6. Real-world reproduction, read-only with no fetch. Use the same pathspecs as the helper, with name=world:
   ```bash
   W=~/dev/sunholo-data/ailang-world
   git -C $W log -1 --format='%h %s' origin/dev -- . ':(exclude)design_docs/world-mission*' ':(exclude)verification/world-iter*'
   git -C $W log -1 --format='%h %s' origin/dev -- . ':(exclude)design_docs/world-mission*' ':(exclude)verification/world-iter*' ':(exclude)design_docs/verification/world-iter*'
   ```
   Expected at plan time: first `a15551c2 record: iteration 415 …` (old behaviour, a record
   commit), second `0e9e8d35 test(row 243 V4 S2b M3): …` (a product commit). If origin/dev has
   moved, report what you see. The second must not be a `record:` commit.
7. Scope: `git diff --name-only origin/dev` must list only `tools/launchd/lib/idle-fingerprint.sh`,
   `tools/launchd/test_mission_idle_fingerprint.sh` and the changelog fragment. The plan file under
   design_docs/ is fleet evidence and is not part of the product diff. Follow the iteration's own
   rule for evidence files.

## Out of scope / non-goals
- No change to `mission-control.sh` or the pre-check block. The bug is only in the pathspec set.
- No generalising to `**/verification/<name>-iter*`. That would hide unknown future layouts, and
  we want those to stay loud.
