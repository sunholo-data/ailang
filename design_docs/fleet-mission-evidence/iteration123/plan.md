# Iteration 123 plan — `gate0:self-notices-rc2-no-watermark-file`

Worktree: `/Users/voightkampff/.ailang-driver-pin/.fleet-i123-self-notices` (base `d2f2ec238`, branch
`fleet/i123-self-notices-absent-watermark`). Every command below: `cd` that path first, run scripts under
`/bin/bash`.

## 1. Verified root cause (each claim + the command that backs it)

| # | Claim | Command | Output |
|---|-------|---------|--------|
| R1 | The floor fires on the live fleet inputs | `/bin/bash scripts/mission_gate0_self_notices.sh --issue 2341 --prev-issue 2172 --repo sunholo-data/ailang --self sunholo-voight-kampff --watermark-file ~/.ailang/state/mission-2341-last-seen --watermark-file ~/.ailang/state/mission-2172-last-seen --control auto --driver-src tools/launchd/mission-control.sh` | two `watermark: … ABSENT — DEGRADED: single-watermark read` lines, then `✗ watermark channel unavailable: no readable watermark among 2 file(s): …`, `rc=2` |
| R2 | Neither fleet nor stapledon has a watermark file | `for n in 2341 2172 450 410; do test -e ~/.ailang/state/mission-$n-last-seen && echo P || echo A; done` (control: `mission-1945-last-seen` → PRESENT, so the probe sees positives) | all four ABSENT |
| R3 | No `*-last-seen` file has been written since the day after #981 landed | `ls -t ~/.ailang/state/*-last-seen \| head -5 \| xargs ls -l` | newest = `Oct 27 21:27` (mission-1892/1974); #981 commits 351c415de / 24c77c15b are dated 2026-10-26 |
| R4 | Post-#981 `mission_directives.sh` writes the file only on `--mark`; absent ⇒ epoch, full read | `sed -n 172,186p scripts/mission_directives.sh` + `:246` | `EPOCH="1970-01-01T00:00:00Z"; WM_STATE="absent"; … SINCE="$EPOCH"`; warn `watermark file … is absent — reading every allowlisted comment on #N; the first --mark creates it`; the only writer is the `--mark` branch (`mv -f "$tmp" "$WMFILE"`) |
| R5 | Pre-#981 the skill wrote the watermark every fire at read time | `git show 351c415de^:.claude/skills/mission-control/resources/gate-0-preflight.md \| sed -n 258,268p` | "write the newest processed `createdAt` to `"$WATERMARK"` … before routing" |
| R6 | The self-notices design assumed ≥1 file exists | `sed -n 36,44p design_docs/implemented/v0_53_1/sprint-plan-gate0-self-notices.md` | "First fire after rotation: current file ABSENT → DEGRADED single read off the prev watermark"; also declares the re-report residual "the rule makes the log credit idempotent by `url=`" |
| R7 | The floor is hiding a REAL died fire | same script with `--since 1970-01-01T00:00:00Z` on stapledon `--issue 450 --prev-issue 410 --repo sunholo-data/stapledons-godot --control auto` | `crash: issue=410 at=2026-11-13T14:15:39Z rc=1 in-window=yes url=…/410#issuecomment-6855501436`, rc 1 — invisible to Gate 0 since then. Fleet (#2341/#2172): 0 crash notices, rc 0 |
| R8 | An epoch since works with both the parser and the comparisons | the `--since 1970-01-01T00:00:00Z` runs above passed `validate_ts` (BSD arm) and printed `since: 1970-01-01T00:00:00Z`; in-window is a string compare `[ "$at" \> "$SINCE" ]` (lines 343, 353), and every GitHub `createdAt` sorts after `1970-…` | — |
| R9 | No code consumer parses the rc-2 text | `grep -rln "gate0_self_notices\|watermark channel unavailable" cmd internal tools .github \| wc -l` → `0`. Positive control: the same pattern over `scripts/` hits `scripts/mission_gate0_self_notices.sh:270` and `scripts/test_mission_gate0_self_notices.sh:526` | Consumers are: the script, its suite (arm 21), `scripts/test_mission_directives.sh:211` (file-PRESENT case, unaffected), and both `gate-0-preflight.md` copies (prose) |

**Decision on the controller's direction: adopt it.** All-absent ⇒ DEGRADED whole-issue read with since =
`1970-01-01T00:00:00Z`, exit by hits. Present-but-unreadable/empty/invalid stays F10 (rc 2), checked in the
same loop *before* the all-absent branch, so an absent file can never outvote a bad one. The epoch is a
constant, so it is **not** routed through `validate_ts` (that function validates input; R8 shows it would
accept the value anyway). The per-file `watermark: <path> ABSENT — …` line is unchanged (arm 20 greps it
verbatim); one new summary line `watermark: DEGRADED — …` is added only when every file is absent.

Rejected alternatives: (i) have the script write its own watermark — gives the no-authority read a write
side, contradicting "never moves the watermark"; (ii) use the file mtime or a heartbeat — new coupling and
a new failure mode; (iii) restore read-time writes in Gate 0 — that is exactly the #981 bug. Note that the
whole-issue window matches what a *present* stale watermark already produces (e.g. any file last marked in
October); only current+prev issues are read, so the window is bounded by issue rotation.

## 2. Code change — apply this patch verbatim

A pre-built, pre-verified patch exists: `/tmp/mc-fleet/i123-code.patch`
(sha256 `d5422b3579ea067a26190a6505581597a580bef2c25e27d5cd82c7c1f3b1314f`; `git apply --check` passes on
the pristine worktree). Apply it with:

```bash
cd /Users/voightkampff/.ailang-driver-pin/.fleet-i123-self-notices
shasum -a 256 /tmp/mc-fleet/i123-code.patch   # must match the sha above; if not, apply the inline copy below by hand
git apply /tmp/mc-fleet/i123-code.patch
```

What it changes in `scripts/mission_gate0_self_notices.sh`:
- header comment: new 5-line "Watermarks:" paragraph;
- new constant `WM_EPOCH="1970-01-01T00:00:00Z"` beside `TS_ARM`;
- `usage()`: new 3-line paragraph on all-absent, and the exit-2 list says `watermark invalid` (not `watermark unavailable/invalid`). The `^  [012] ` lines stay at exactly 3;
- watermark loop: `ABSENT` counter; after the loop `READABLE -eq 0` now (a) asserts `ABSENT == GIVEN` (else `✗ internal: …`, rc 2), (b) prints
  `watermark: DEGRADED — no watermark file exists among $GIVEN given ($WMPATHS): reading the WHOLE of each issue since $WM_EPOCH; absent is not corrupt (mission_directives.sh creates the file only on --mark), so the window widens and no notice is dropped`,
  (c) `SINCE="$WM_EPOCH"`, `WSMARK="watermarks 0/$GIVEN, all absent: whole-issue read"` (this flows into the verdict's `[...]` suffix);
- since line: new middle branch `since: $SINCE (no watermark among $GIVEN file(s): whole of each issue)`.
The string `watermark channel unavailable` disappears from the script.

Live output with the patched script (scratch copy), stapledon inputs exactly as Gate 0 passes them:
`watermark: DEGRADED — …`, `since: 1970-01-01T00:00:00Z (no watermark among 2 file(s): whole of each issue)`,
`crash: issue=410 at=2026-11-13T14:15:39Z rc=1 in-window=yes url=…`, verdict `… [mission_gate0_self_notices.sh, control 4:5, watermarks 0/2, all absent: whole-issue read] …`, rc 1.
Fleet inputs: same DEGRADED lines, `verdict: 0 crash notice(s) since 1970-01-01T00:00:00Z on #2341,#2172`, rc 0.

## 3. Tests (in the same patch) — `scripts/test_mission_gate0_self_notices.sh`

The old arm 21 (`watermark-unavailable`, F9 rc 2 + empty argv) is **rewritten** as `arm_watermark_all_absent`
(dispatcher key `watermark-all-absent`, run-list updated, header comment no longer lists 21 as a floor arm).
Its fixture style is unchanged: heredoc `make_json`, `make_stub_gh`, `run_instr`.

| Case | Where | Asserts | Mutation it kills (all measured on a scratch copy via `G0_SCRIPT_UT`) |
|------|-------|---------|-----------------------------------------------|
| (a) both absent + 1 crash notice (issue 5, control 5:1) | arm 21 | rc 1; `crash: … in-window=yes url=https://x/5#1`; exact DEGRADED line; `since: 1970-… (no watermark among 2 file(s): whole of each issue)`; verdict carries `watermarks 0/2, all absent: whole-issue read]`; stderr has no `channel unavailable` | M1 restore old `exit 2` → 8 red; M2 since=now instead of epoch → 5 red; M3 drop the DEGRADED echo → 2 red; M7 drop the since branch → 1 red |
| (b) both absent + no notices (issue 6, control 6:0) | arm 21 | rc 0 + DEGRADED line; `verdict: 0 crash notice(s) since 1970-01-01T00:00:00Z on #6 ` | M1, M3 |
| (c) one absent + one valid | arm 20 (existing, plus 1 assertion) | existing ABSENT line + `since: … (older of 1 readable watermark(s) of 2)` unchanged; NEW: no `watermark: DEGRADED — no watermark` line | M5 print the whole-issue line whenever any file is absent → 1 red |
| (d) absent FIRST, then malformed | arm 22 new case (b2) | rc 2, `✗ watermark invalid: …/wm_bad (not-ISO8601 'not-a-date')` | M4 `break` instead of `continue` on an absent file → red (plus 4 arm-21 reds) |
| (d') present-but-empty | arm 22 (a), existing | rc 2 `(empty)` | M6 treat empty as absent → red |

Counts: 168 − 2 (old arm 21) + 8 (new arm 21) + 1 (arm 20) + 1 (arm 22) = **176 passed, 0 failed** (measured).
File length goes 786 → 810 lines. No gate covers `.sh` length (`make check-file-sizes` scans only
`internal`/`cmd` `*.go`, `make/code-health.mk:164-170`); iteration 87's "<800" was a planner preference.
Do NOT split the file in this iteration.

Mutation drill (optional, ~2 min per mutant; run M1 and M4 at least, in the foreground, one at a time):
```bash
cd /Users/voightkampff/.ailang-driver-pin/.fleet-i123-self-notices
for m in M1 M4; do python3 /tmp/mc-fleet/i123-mut.py scripts/mission_gate0_self_notices.sh /tmp/mc-fleet/i123-mut-$m.sh $m &&
  G0_SCRIPT_UT=/tmp/mc-fleet/i123-mut-$m.sh /bin/bash scripts/test_mission_gate0_self_notices.sh | tail -1; done
# expected: M1 "168 passed, 8 failed"; M4 "171 passed, 5 failed"
```
(`/tmp/mc-fleet/i123-mut.py` holds M1–M7 as exact string replacements; it asserts each anchor is unique.)

## 4. Doc / skill edits

### 4a. `.claude/skills/mission-control/resources/gate-0-preflight.md`, step 6a (line ~270)

Old:
```
   in the log ONCE per `url=` (grep first). **rc 0** = no death signal (traces still run). **rc 2 =
   instrument failure** (named floor) — report it; NOT a verdict either way. A hit grants nothing:
   unparks nothing, picks nothing, never moves the watermark.
```
New:
```
   in the log ONCE per `url=` (grep first). **rc 0** = no death signal (traces still run). **rc 2 =
   instrument failure** (named floor) — report it; NOT a verdict either way. A hit grants nothing:
   unparks nothing, picks nothing, never moves the watermark.
   **No watermark file is NOT rc 2.** Since #981 `mission_directives.sh` creates
   `mission-<issue>-last-seen` only on `--mark`, so a mission with no routed directive has none. When
   both files are absent the script prints `watermark: DEGRADED` and reads the WHOLE of both issues:
   an old crash notice then recurs every fire until its issue rotates out of `-prev`. That is the
   once-per-`url=` rule's job, not a new death. A file that exists but is unreadable, empty or
   invalid is still rc 2.
```

### 4b. `.agents/skills/mission-control/resources/gate-0-preflight.md`: YES, same edit
The two trees are byte-identical today (`diff -r .claude/skills/mission-control .agents/skills/mission-control` → no output),
and `tools/launchd/test_agents_skills_sync.sh` (in `make test-launchd-drivers`, `make/test.mk:105`) runs
`tools/launchd/sync-agents-skills.sh --check`, which fails on any drift. Do not hand-edit the mirror. Run:
```bash
/bin/bash tools/launchd/sync-agents-skills.sh && /bin/bash tools/launchd/sync-agents-skills.sh --check
```
Then confirm `git status --short` shows only the gate-0-preflight.md pair under `.agents/` (rsync --delete over
in-sync trees should touch nothing else; if it touches anything else, STOP and report).

### 4c. `design_docs/implemented/v0_53_1/sprint-plan-gate0-self-notices.md` §2 item 1 (lines 41-44)
Append, after "…the rule makes the log credit idempotent by `url=`.", on its own line, indented 3 spaces to stay inside item 1:
```
   *Amended 2026-11-19 (fleet iteration 123):* #981 made `mission_directives.sh` create the file only on
   `--mark`, so both files can be absent. Doing that used to hit the F9 floor (rc 2) on every fire. It is
   now a DEGRADED whole-issue read since the epoch. F10 (present but invalid) is unchanged.
```

No other prose names F9/"watermark unavailable" as live behaviour: `design_docs/fleet-mission-evidence/**` and
`.ailang/state/sprints/*.json` are historical records. Do not edit them.

## 5. Changelog fragment — create `changelogs/unreleased/2026-11-19-gate0-self-notices-absent-watermark.md`

```markdown
### Fixed — Gate 0 self-notices read is no longer blind when no watermark file exists (2026-11-19)

`scripts/mission_gate0_self_notices.sh` exited 2 ("watermark channel unavailable") whenever every
`--watermark-file` was absent. Since #981, `mission_directives.sh` creates
`mission-<issue>-last-seen` only on `--mark`, so a mission with no routed human directive never has
one. As a result the Gate 0 step 6a read was an instrument floor on every fire for the fleet (#2341/#2172) and
stapledon (#450/#410), where it hid a real died fire (#410, 2026-11-13, rc=1). All-absent is now
a DEGRADED read of the whole of each issue since `1970-01-01T00:00:00Z`, which matches how
`mission_directives.sh` reads an absent file. A `watermark: DEGRADED` line and a
`watermarks 0/N, all absent: whole-issue read` verdict suffix make it visible, and it exits 0/1 by
hits. A file that exists but is unreadable, empty or invalid is still a floor (rc 2). The
mission-control skill's step 6a (both copies) explains it. Ticket
`gate0:self-notices-rc2-no-watermark-file`.
```
(Format verified against `changelogs/unreleased/README.md`: one `### ` heading, no `#`/`##`.)

## 6. Acceptance commands (all from the worktree)

```bash
cd /Users/voightkampff/.ailang-driver-pin/.fleet-i123-self-notices
/bin/bash -n scripts/mission_gate0_self_notices.sh && /bin/bash -n scripts/test_mission_gate0_self_notices.sh
/bin/bash scripts/mission_gate0_self_notices.sh --help | grep -c '^  [012] '          # → 3
grep -c 'watermark channel unavailable' scripts/mission_gate0_self_notices.sh           # → 0
grep -nE 'declare -A|\$\{[a-zA-Z_]+,,\}|mapfile|readarray' scripts/mission_gate0_self_notices.sh  # → only the line-18 disclaimer comment
/bin/bash scripts/test_mission_gate0_self_notices.sh | tail -1                          # → 176 passed, 0 failed (rc 0)
/bin/bash scripts/test_mission_gate0_self_notices.sh --only watermark-all-absent | tail -1  # → 8 passed, 0 failed
/bin/bash scripts/test_mission_directives.sh | tail -1                                   # consumer of the line-1 format: still all pass
shellcheck -S warning scripts/mission_gate0_self_notices.sh scripts/test_mission_gate0_self_notices.sh 2>&1 | grep -c 'SC[0-9]'  # → 9 (= baseline; all pre-existing SC2034 in the test)
/bin/bash tools/launchd/sync-agents-skills.sh --check                                   # → agents skills in sync (...)
/bin/bash tools/launchd/test_agents_skills_sync.sh; echo rc=$?                          # → rc=0
make check-changelog                                                                     # → ✓ 5 changelog fragment(s) valid …
git diff --check                                                                         # → no output
git diff --name-only   # must list ONLY: scripts/mission_gate0_self_notices.sh, scripts/test_mission_gate0_self_notices.sh,
                       # .claude/skills/mission-control/resources/gate-0-preflight.md, .agents/skills/mission-control/resources/gate-0-preflight.md,
                       # design_docs/implemented/v0_53_1/sprint-plan-gate0-self-notices.md   (+ the new changelog fragment, untracked)
# Live, read-only (gh reads only; exactly what Gate 0 runs):
/bin/bash scripts/mission_gate0_self_notices.sh --issue 450 --prev-issue 410 --repo sunholo-data/stapledons-godot \
  --self sunholo-voight-kampff --watermark-file ~/.ailang/state/mission-450-last-seen --watermark-file ~/.ailang/state/mission-410-last-seen \
  --control auto --driver-src tools/launchd/mission-control.sh; echo rc=$?               # → rc=1, crash on #410 2026-11-13T14:15:39Z
/bin/bash scripts/mission_gate0_self_notices.sh --issue 2341 --prev-issue 2172 --repo sunholo-data/ailang \
  --self sunholo-voight-kampff --watermark-file ~/.ailang/state/mission-2341-last-seen --watermark-file ~/.ailang/state/mission-2172-last-seen \
  --control auto --driver-src tools/launchd/mission-control.sh; echo rc=$?               # → rc=0, DEGRADED line present
# Slow (~6 min). Run in the foreground with a long timeout, or background it and poll the log in a bounded date +%s loop:
make test-launchd-drivers > /tmp/mc-fleet/i123-launchd.log 2>&1; echo rc=$?             # → rc=0; the log shows "ok    scripts/test_mission_gate0_self_notices.sh"
```

## 7. Risks / what could break

1. **A recurring rc 1 (intended, and already declared in the v0_53_1 design §2.1).** With no watermark, any crash
   notice on the current or `-prev` issue is in-window on every fire until the issue rotates out of `-prev`.
   Stapledon will get rc 1 for the #410 2026-11-13 notice for the rest of #410's `-prev` week. Gate 0 credits the orphan
   once per `url=`, and the new skill sentence says so. The window can only widen, so it never misses a notice.
2. **A typo'd watermark path now gives a whole-issue read instead of a floor.** That is loud, not silent: a DEGRADED
   line, `watermarks 0/N, all absent` in the verdict, and the window only over-reports. The skill derives both paths from
   `$HOME/.ailang/state/mission-${ISSUE|PREV}-last-seen`, the same derivation `mission_directives.sh` uses.
3. **Consumers of the rc or the string:** none in code (R9). The only text consumer was arm 21, which is rewritten.
   `scripts/test_mission_directives.sh:211` runs the script with a file present, so it is unaffected (acceptance runs it).
4. **The per-file ABSENT line still says "single-watermark read"** even when both are absent. It is kept verbatim because
   arm 20 pins it and the ticket asked for the format to stay. The new summary line follows it directly and states the real window.
5. **World's fork** has its own `gate0_self_notices.sh` in a separate repo and is out of reach and out of scope here.
   It probably has the same latent bug if its files are also --mark-only. Mention this in the report as a follow-up ticket, but do not file or fix it here.
6. **`.agents` sync:** `rsync --delete` runs over mission-*/sprint-* and the D-FLEET-38 list. It is safe while the trees
   are in sync (verified at base). If `git status` shows more than the one mirrored file, stop.
7. **Scope guard:** every touched path is in the allowed set (`scripts/mission_*`, `scripts/test_mission_*`,
   `.claude/skills/mission-*/**`, `.agents/skills/mission-*/**`, `design_docs/**`, `changelogs/**`). No `make/test.mk` change is needed:
   the suite is already wired at `make/test.mk:95`.

## Appendix — inline copy of `/tmp/mc-fleet/i123-code.patch`

```diff
--- a/scripts/mission_gate0_self_notices.sh
+++ b/scripts/mission_gate0_self_notices.sh
@@ -9,6 +9,12 @@
 # is ever printed. Transcript notices require an anchored old/current prefix, a whitespace-free
 # fire ID, ASCII hit count and literal terminal period. Only extracted metadata is printed.
 #
+# Watermarks: the window is the OLDER of the readable --watermark-file values. A file that is
+# ABSENT is not corrupt: since #981 mission_directives.sh creates it only on --mark, so a mission
+# with no routed human directive has none. When EVERY given file is absent the read is a DEGRADED
+# whole-issue read (since the epoch, as mission_directives.sh reads an absent file) — it can only
+# over-report, never miss a notice. A file that exists but is unreadable, empty or invalid is F10.
+#
 # This is a bash-3.2 script (the rig's /bin/bash is 3.2.57). No associative arrays, no ${x,,},
 # no {n} intervals. The gh binary is the ONLY read seam (the stub in the suite shadows --gh-bin).
 set -uo pipefail
@@ -18,6 +24,7 @@
 SKEW=300
 TS_CTL="2026-09-08T06:19:00Z"
 TS_ARM=""
+WM_EPOCH="1970-01-01T00:00:00Z"   # the whole-issue window, as mission_directives.sh reads an absent file
 ISSUE=""; PREV_ISSUE=""; PREV_PASSED=0; REPO=""; SELF=""; SINCE_ARG=""; CONTROL=""; DRIVER_SRC=""
 WMFILES=()
 
@@ -33,6 +40,10 @@
 reporting is informational (notice counts, hit counts, fire IDs); it does not change exit codes.
 A hit grants nothing. No comment body is printed.
 
+Every --watermark-file ABSENT is not a floor: it is a DEGRADED read of the whole of each issue
+(since 1970-01-01T00:00:00Z), reported on a 'watermark: DEGRADED' line. A file that exists but
+is unreadable, empty or invalid is still a floor.
+
 --control auto resolves the control from a per-repo table keyed by --repo; an unknown repo
 is a floor (no fallback).
 
@@ -40,7 +51,7 @@
   0  no in-window crash notice, controls ok
   1  one or more in-window crash notices (a controller must look; run Gate 2's traces (a)-(c))
   2  any floor F0-F10 (usage, binary, timeout, gh failure, malformed, truncated, control,
-     signature-source, watermark unavailable/invalid, timestamp parser unusable)
+     signature-source, watermark invalid, timestamp parser unusable)
 EOF
 }
 
@@ -237,10 +248,10 @@
 # ── timestamp-parser control (pre-network) ───────────────────────────────────
 ts_resolve_arm
 
-# ── watermark disposition + F9/F10 + since ───────────────────────────────────
+# ── watermark disposition + all-absent DEGRADED + F10 + since ───────────────────────────────────
 echo "gate0_self_notices: repo=$REPO self=$SELF issues=$ISSUE${PREV_ISSUE:+,$PREV_ISSUE}"
 SELF_LC="$(printf '%s' "$SELF" | tr '[:upper:]' '[:lower:]')"
-READABLE=0; GIVEN=0; WMIN_LIST=""; WMPATHS=""
+READABLE=0; GIVEN=0; ABSENT=0; WMIN_LIST=""; WMPATHS=""
 if [ -n "$SINCE_ARG" ]; then
   validate_ts "$SINCE_ARG" "--since"
   SINCE="$SINCE_ARG"
@@ -251,6 +262,7 @@
     if [ -n "$WMPATHS" ]; then WMPATHS="$WMPATHS $wf"; else WMPATHS="$wf"; fi
     if [ ! -e "$wf" ]; then
       echo "watermark: $wf ABSENT — DEGRADED: single-watermark read (iter-52 rule wants two)"
+      ABSENT=$((ABSENT+1))
       continue
     fi
     if [ ! -r "$wf" ]; then
@@ -267,14 +279,23 @@
     WMIN_LIST="${WMIN_LIST:+$WMIN_LIST$'\n'}$v"
   done
   if [ "$READABLE" -eq 0 ]; then
-    echo "✗ watermark channel unavailable: no readable watermark among $GIVEN file(s): $WMPATHS" >&2; exit 2
+    # Every present file either counted as readable or exited F10 above, so 0 readable = all absent.
+    if [ "$ABSENT" -ne "$GIVEN" ]; then
+      echo "✗ internal: $GIVEN watermark file(s), $ABSENT absent, 0 readable" >&2; exit 2
+    fi
+    echo "watermark: DEGRADED — no watermark file exists among $GIVEN given ($WMPATHS): reading the WHOLE of each issue since $WM_EPOCH; absent is not corrupt (mission_directives.sh creates the file only on --mark), so the window widens and no notice is dropped"
+    SINCE="$WM_EPOCH"
+    WSMARK="watermarks 0/$GIVEN, all absent: whole-issue read"
+  else
+    SINCE="$(printf '%s\n' "$WMIN_LIST" | /usr/bin/sort | head -1)"
+    WSMARK="watermarks $READABLE/$GIVEN"
   fi
-  SINCE="$(printf '%s\n' "$WMIN_LIST" | /usr/bin/sort | head -1)"
-  WSMARK="watermarks $READABLE/$GIVEN"
 fi
 
 if [ -n "$SINCE_ARG" ]; then
   echo "since: $SINCE (--since literal)"
+elif [ "$READABLE" -eq 0 ]; then
+  echo "since: $SINCE (no watermark among $GIVEN file(s): whole of each issue)"
 else
   echo "since: $SINCE (older of $READABLE readable watermark(s) of $GIVEN)"
 fi
--- a/scripts/test_mission_gate0_self_notices.sh
+++ b/scripts/test_mission_gate0_self_notices.sh
@@ -9,9 +9,9 @@
 # synthetic arm can accidentally satisfy a snapshot assertion.
 #
 # Row-90 property: each arm greps only the fields its behaviour owns. Classification arms
-# (1-5) grep `crash:` lines by `at=` + one token; floor arms (7-14, 21, 22, 23, 24) their own
+# (1-5) grep `crash:` lines by `at=` + one token; floor arms (7-14, 22, 23, 24) their own
 # message, rc, argv-log emptiness where stated; report/transcript arms check verdicts; snapshots
-# a full `issue:` line and `control:`; watermark arms (19-22) `watermark:`/`since:` only.
+# a full `issue:` line and `control:`; watermark arms (19-22) `watermark:`/`since:` (21 also its verdict).
 set -uo pipefail
 cd "$(dirname "$0")/.." || exit 1
 SCRIPT_UT="${G0_SCRIPT_UT:-scripts/mission_gate0_self_notices.sh}"
@@ -514,17 +514,37 @@
   [ "$rc" -eq 0 ] || { notok "watermark-degraded rc=$rc"; return; }
   /usr/bin/grep -q "watermark: $d/wm_absent ABSENT — DEGRADED: single-watermark read (iter-52 rule wants two)" "$d/out" && ok "watermark-degraded ABSENT" || notok "watermark-degraded ABSENT line"
   /usr/bin/grep -q '^since: 2026-09-08T06:19:00Z (older of 1 readable watermark(s) of 2)' "$d/out" && ok "watermark-degraded since" || notok "watermark-degraded since"
+  /usr/bin/grep -q '^watermark: DEGRADED — no watermark' "$d/out" && notok "watermark-degraded whole-issue line" || ok "watermark-degraded no whole-issue line"
 }
 
-# ── arm 21: watermark-unavailable (AC-M1-15) ──────────────────────────────────
-arm_watermark_unavailable() {
-  local d out err rc
+# ── arm 21: watermark-all-absent — every file ABSENT is a DEGRADED whole-issue read, not F9 ──
+arm_watermark_all_absent() {
+  local d rc deg
   d="$SCRATCH/a21"; mkdir -p "$d"
+  deg="watermark: DEGRADED — no watermark file exists among 2 given ($d/p1 $d/p2): reading the WHOLE of each issue since 1970-01-01T00:00:00Z"
+  make_json "$d/5.json" <<EOF
+{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/5#1","body":"${SIG}143 — died"}]}
+EOF
+  printf '{"comments": 1}\n' > "$d/5.meta.json"
+  make_json "$d/6.json" <<EOF
+{"comments":[{"author":{"login":"sunholo-voight-kampff"},"createdAt":"2026-09-01T00:00:00Z","url":"https://x/6#1","body":"plain"}]}
+EOF
+  printf '{"comments": 1}\n' > "$d/6.meta.json"
   make_stub_gh "$d/stub" "$d" "$d/argv.log"
-  run_instr "$d/out" "$d/err" --issue 5 --repo test/test --self sunholo-voight-kampff \
-    --watermark-file "$d/p1" --watermark-file "$d/p2" --control 5:0 --gh-bin "$d/stub"; rc=$?
-  /usr/bin/grep -q "✗ watermark channel unavailable: no readable watermark among 2 file(s): $d/p1 $d/p2" "$d/err" && [ "$rc" -eq 2 ] && ok "watermark-unavailable F9 rc2" || notok "watermark-unavailable rc=$rc"
-  if [ ! -s "$d/argv.log" ]; then ok "watermark-unavailable empty argv"; else notok "watermark-unavailable argv non-empty"; fi
+  # (a) both absent + one crash notice -> rc 1, the notice is in-window
+  run_instr "$d/oa" "$d/ea" --issue 5 --repo test/test --self sunholo-voight-kampff \
+    --watermark-file "$d/p1" --watermark-file "$d/p2" --control 5:1 --gh-bin "$d/stub"; rc=$?
+  [ "$rc" -eq 1 ] && ok "watermark-all-absent crash rc1" || notok "watermark-all-absent crash rc=$rc"
+  /usr/bin/grep -q '^crash: issue=5 at=2026-09-01T00:00:00Z rc=143 in-window=yes url=https://x/5#1$' "$d/oa" && ok "watermark-all-absent crash in-window" || notok "watermark-all-absent crash line"
+  /usr/bin/grep -qF "$deg" "$d/oa" && ok "watermark-all-absent DEGRADED line" || notok "watermark-all-absent DEGRADED line"
+  /usr/bin/grep -q '^since: 1970-01-01T00:00:00Z (no watermark among 2 file(s): whole of each issue)$' "$d/oa" && ok "watermark-all-absent since" || notok "watermark-all-absent since"
+  /usr/bin/grep -q '^verdict: 1 crash notice(s) since 1970-01-01T00:00:00Z on #5 — A FIRE DIED .*watermarks 0/2, all absent: whole-issue read\]' "$d/oa" && ok "watermark-all-absent verdict" || notok "watermark-all-absent verdict"
+  /usr/bin/grep -q 'channel unavailable' "$d/ea" && notok "watermark-all-absent old F9 text" || ok "watermark-all-absent no F9 text"
+  # (b) both absent + no notices -> rc 0, still DEGRADED
+  run_instr "$d/ob" "$d/eb" --issue 6 --repo test/test --self sunholo-voight-kampff \
+    --watermark-file "$d/p1" --watermark-file "$d/p2" --control 6:0 --gh-bin "$d/stub"; rc=$?
+  [ "$rc" -eq 0 ] && /usr/bin/grep -qF "$deg" "$d/ob" && ok "watermark-all-absent quiet rc0 DEGRADED" || notok "watermark-all-absent quiet rc=$rc"
+  /usr/bin/grep -q '^verdict: 0 crash notice(s) since 1970-01-01T00:00:00Z on #6 ' "$d/ob" && ok "watermark-all-absent quiet verdict" || notok "watermark-all-absent quiet verdict"
 }
 
 # ── arm 22: watermark-invalid (AC-M1-15) ──────────────────────────────────────
@@ -542,6 +562,10 @@
   run_instr "$d/ob" "$d/eb" --issue 5 --repo test/test --self sunholo-voight-kampff \
     --watermark-file "$d/wm_a" --watermark-file "$d/wm_bad" --control 5:0 --gh-bin "$d/stub"; rc=$?
   /usr/bin/grep -q "✗ watermark invalid: $d/wm_bad (not-ISO8601 'not-a-date')" "$d/eb" && [ "$rc" -eq 2 ] && ok "watermark-invalid not-ISO8601" || notok "watermark-invalid not-ISO8601 rc=$rc"
+  # (b2) absent beside malformed — absent never excuses an invalid file
+  run_instr "$d/ob2" "$d/eb2" --issue 5 --repo test/test --self sunholo-voight-kampff \
+    --watermark-file "$d/wm_absent" --watermark-file "$d/wm_bad" --control 5:0 --gh-bin "$d/stub"; rc=$?
+  /usr/bin/grep -q "✗ watermark invalid: $d/wm_bad (not-ISO8601 'not-a-date')" "$d/eb2" && [ "$rc" -eq 2 ] && ok "watermark-invalid absent+bad F10" || notok "watermark-invalid absent+bad rc=$rc"
   # (c) unreadable
   printf '2026-09-08T06:19:00Z' > "$d/wm_unreadable"
   chmod 000 "$d/wm_unreadable"
@@ -746,7 +770,7 @@
     snapshot-129) arm_snapshot_129 ;;
     watermark-older) arm_watermark_older ;;
     watermark-degraded) arm_watermark_degraded ;;
-    watermark-unavailable) arm_watermark_unavailable ;;
+    watermark-all-absent) arm_watermark_all_absent ;;
     watermark-invalid) arm_watermark_invalid ;;
     watermark-strict) arm_watermark_strict ;;
     watermark-parser-control) arm_watermark_parser_control ;;
@@ -774,7 +798,7 @@
   for arm in anchored-signature rc-extract since-window self-filter prev-issue no-bodies \
              usage gh-missing gh-timeout gh-failed malformed truncated control-required \
              control-mismatch signature-source report snapshot-107 snapshot-129 \
-             watermark-older watermark-degraded watermark-unavailable watermark-invalid \
+             watermark-older watermark-degraded watermark-all-absent watermark-invalid \
              watermark-strict watermark-parser-control \
              driver-literal single-literal control-auto snapshot-1380 $(transcript_arms); do
     run_arm "$arm"
```
