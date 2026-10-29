# Sprint plan: exit-path notices bounded (fleet iteration 33)

**Ticket:** `driver:exit-path-notices-unbounded` (P1 #9, `design_docs/fleet-mission.md`)
**Base:** origin/dev `06d524c59` · **Branch:** `fleet/i33-exit-notices-bounded`
**Lane:** motoko-free driver fix (tools/launchd only). No routing, ration, lane-order or billing change.
**Size:** ~0.5 day. 3 milestones, TDD order (markers + red arms first, then the fix).

## 1. Problem

The driver's three exit-path notices call `ailang messages send` and `gh issue comment` bare.
Both are network calls to third parties and can hang. A hang at these sites wedges the process.
launchd does not start a second instance of a job that is still running, so a hung exit notice
blocks that mission's next fire until something kills it. On the refusal path that means a
blocked mission cannot even get back to its own retry. The slot-verdict notices were bounded in
`10448bad5`. These three sites were missed.

There is a second defect at the same sites. The bare sends do **not** scope the message store.
Neither the driver, nor any `~/.config/ailang/mission-*.env`, nor `secrets.env`, nor any
installed `dev.ailang.mission-*` plist sets `AILANG_STORAGE_MESSAGING` or
`AILANG_MESSAGES_PROJECT` (checked on the rig, 2026-10-12: zero matches). Under launchd these
sends therefore go to the machine-private SQLite inbox and exit 0. That is the silent fallback
`_mc_notify`'s comment measured on 2026-09-07 (`msg_...` local vs `inbox_...` prod). The same
gap exists in the already-bounded SLOT NOTIFY block (`_mc_bounded 30 ailang messages send`, no
`env`).

## 2. Exact sites (HEAD `06d524c59`, `tools/launchd/mission-control.sh`)

| # | Lines | Path | Title (messages) | GH body starts |
|---|-------|------|------------------|----------------|
| A | 2038 / 2041 | `if ! select_model` refusal, episode-gated by `$BLOCKED_FILE`, then `exit 1` | `Mission iteration blocked: no usable model` | `⚠️ Mission iteration did not start: **no usable controller**` |
| B | 2529 / 2532 | `RC≠0` but the mission log gained a `## ` record ("late kill, work landed") | `Mission iteration killed post-record (rc=$RC) — work landed` | `ℹ️ Mission iteration exited **rc=$RC after landing its record**` |
| C | 2550 / 2553 | `RC≠0`, episode-gated by `*-rcfail.episode` | `Mission iteration FAILED (rc=$RC)` | `⚠️ Mission iteration **FAILED to complete** (rc=$RC` ← **F8 literal** |
| S | 2511 / 2514 | SLOT NOTIFY (already bounded, `30`) | unchanged | unchanged (store scoping only) |

`NOTIFY_TIMEOUT` is defined at :882. `_mc_bounded` is sourced from `lib/lane-probe.sh` at :885.
Both come before every site above. The refusal path has not written `$PIDFILE` at that point.
Sites B and C run after `rm -f "$PIDFILE"`, but the launchd job is still live.

## 3. Fix shape (decided)

### 3.1 One helper, `_mc_exit_notice`, defined right after `_mc_notify` (~:340)

```bash
# _mc_exit_notice LABEL TITLE MSG_BODY GH_BODY — one-shot, BOUNDED, NON-FATAL exit-path notice.
_mc_exit_notice() {
  local label="$1" title="$2" mbody="$3" gbody="$4" _rc _out
  _mc_bounded "$NOTIFY_TIMEOUT" env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT="${AILANG_MESSAGES_PROJECT:-ailang-multivac}" \
    ailang messages send controlplane "$mbody" --title "$title" --from "$MSG_FROM"; _rc=$?
  if [ "$_rc" -ne 0 ]; then
    _out="$MC_BOUNDED_OUT"
    [ "$_rc" -eq 124 ] && _out="timed out after ${NOTIFY_TIMEOUT}s (no output)"
    [ -z "$_out" ] && _out="no output (rc=$_rc)"
    if _mc_notice_suppressed; then
      log "${label} notice already in the inbox (duplicate title) — not re-sent"
    else
      log "WARNING: ${label} notice FAILED to send via ailang messages (rc=$_rc): $(printf '%s' "$_out" | tail -c 300 | tr '\n' ' ')"
    fi
  fi
  [ -n "${MISSION_GH_ISSUE:-}" ] || return 0      # unchanged: these sites were silent when unset
  _mc_bounded "$NOTIFY_TIMEOUT" gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" --body "$gbody"; _rc=$?
  if [ "$_rc" -ne 0 ]; then
    _out="$MC_BOUNDED_OUT"; [ "$_rc" -eq 124 ] && _out="timed out after ${NOTIFY_TIMEOUT}s (no output)"
    log "WARNING: ${label} notice FAILED to post to issue #${MISSION_GH_ISSUE} (rc=$_rc): $(printf '%s' "$_out" | tail -c 300 | tr '\n' ' ')"
  fi
  return 0
}
```

The three sites become one call each. Title and body strings move into the call **byte-for-byte**
from HEAD. That includes the multi-line `${_ref_reserve:+…}` GH body at A and the F8 literal
at C, which stays as a source literal in the call's 4th argument. The trailing `2>/dev/null`
redirections go away, because `_mc_bounded` already captures stdout+stderr into `MC_BOUNDED_OUT`
(the warning reason). Exit codes do not change: `exit 1` stays after A, and `exit "$RC"` stays
at the end of the final block. The helper always returns 0.

**Why a helper and not six inline wraps.** The warning, timeout-diagnosis and suppression logic
would otherwise be copied three times, and the CLAUDE.md "two near-identical emit blocks is how
one rots" comment above `_mc_notify` already records what that costs. With a helper, the source
guard (§5 arm G) has one canonical shape to recognise.

**Why not reuse `_mc_notify`.** `_mc_notify` takes ONE body for both channels, and these sites
use different bodies per channel. It date-stamps the title, which is a text change constraint 4
forbids. It also retries 3× with 5 s and 10 s backoff and spools, which can stretch the exit path
to roughly 3×(T+6)+15+T. One bounded shot per channel is the right shape for a process that is
about to exit. The spool is a preflight-drain feature, and the next fire re-derives these
conditions anyway: the refusal re-probes and the rc-fail episode re-evaluates.

### 3.2 Env-scoping decision: YES, use the `_mc_notify` form (`env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=${AILANG_MESSAGES_PROJECT:-ailang-multivac}`)

- Nothing on the launchd path exports either variable (§1). As written, the sites are bounded
  in no sense that matters: they deliver to a store nobody reads. Bounding them without scoping
  would leave a notice that is "fixed" and still invisible. That is the silent-fallback class
  (Critical Principle 2).
- Per-command `env`, never `export`, follows CLAUDE.md. `AILANG_STORAGE` stays put, so the
  coordinator and observatory stores do not move. This is the same form as `_mc_notify` and
  `_mc_drain_notices`. `env` is also the only form that works through `( exec "$@" )`.
- Honouring an ambient `AILANG_MESSAGES_PROJECT` via `:-` matches the existing two sites.
- This is store hygiene, not routing in the policy sense: no model, lane, ration or billing
  input changes. It does change where the notice is **readable**, so the changelog says so.

**Known consequence, recorded as a follow-up and not fixed here.** In prod Firestore,
`ailang messages send` rejects a title that already exists (rc=1, `duplicate message exists`).
Titles A/B/C are not date-stamped, so each distinct title lands once and later repeats are
refused. The local SQLite inbox has the same rule, so these titles have most likely been
rejected locally since their first send, and `2>/dev/null` hid that. The helper classifies
that refusal through `_mc_notice_suppressed` and logs it as "already in the inbox", not as a
WARNING. The fix is to date-stamp the messages titles via `_mc_notice_title`, as `_mc_notify`
does. That is a text change, constraint 4 forbids it here, so it goes to the backlog as
**`driver:exit-notice-titles-undated`**. The GH channel, which F8/gate0 reads, is unaffected.

### 3.3 SLOT NOTIFY (site S): add the same `env` scoping, keep `_mc_bounded 30` and `|| true`

This is one token per send line. It brings the only other driver send into the shape the store
guard enforces. `test_mission_heartbeat.sh:138` greps its stubbed `_mc_bounded` argv for
`^30 ailang messages send`. Update it to
`^30 env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=[^ ]* ailang messages send`.
This is the only change to that file. M3 can be split off cleanly if the evaluator rules it
out of scope. The store guard (arm G2) would then carry a single named exemption for the SLOT
NOTIFY block.

### 3.4 Extraction markers (comment-only, no behaviour)

- `# --- REFUSAL NOTICE START ---` immediately above `if ! select_model; then`, and
  `# --- REFUSAL NOTICE END ---` immediately below its closing `fi` (above `rm -f "$BLOCKED_FILE"`).
- `# --- FINAL RC NOTICE START ---` immediately above `if [ "$RC" -ne 0 ]; then`, and
  `# --- FINAL RC NOTICE END ---` immediately below `exit "$RC"`. Keep the existing
  `/^if \[ "\$RC" -ne 0 \]; then$/,/^exit "\$RC"$/` anchor valid. `test_controller_capacity.sh`
  uses it, and markers are added *around* it, not in place of it.

Verify that no existing awk range in `tools/launchd/test_*.sh` starts or ends on these new
lines: `grep -n 'REFUSAL NOTICE\|FINAL RC NOTICE' tools/launchd/test_*.sh` must return only the
new arms.

## 4. Per-file change list

| File | Change |
|------|--------|
| `tools/launchd/mission-control.sh` | + `_mc_exit_notice` after `_mc_notify`. Sites A/B/C → one `_mc_exit_notice` call each, strings byte-identical. Site S → `env` scoping. + 4 marker comments. |
| `tools/launchd/test_driver_notify.sh` | `DRV="${MC_NOTIFY_DRIVER:-…}"` override (for scratch-copy mutation drills, same as `MC_CAPACITY_DRIVER`). Extract `_mc_exit_notice` + the two marker blocks (+ FATAL-on-empty guard). New arms E1–E9 and guards G1–G3 (§5). |
| `tools/launchd/test_controller_capacity.sh` | + `awk '/^_mc_exit_notice\(\) \{/,/^\}$/' "$DRV" >> "$X/fns.sh"` plus a `grep -q '^_mc_exit_notice()'` extraction guard. Without it, `final.sh` calls an undefined function and the `killed post-record` / `FAILED (rc=1)` counts go to 0, which is loud but would fail for the wrong reason. |
| `tools/launchd/test_mission_heartbeat.sh` | :138 regex update (M3 only). |
| `changelogs/unreleased/2026-10-12-exit-path-notices-bounded.md` | `### Fixed` (§7). |
| `.ailang/state/sprints/…` | executor's sprint state/red-green log (optional). |

No new env var: `NOTIFY_TIMEOUT` / `MISSION_NOTIFY_TIMEOUT` already exist. No `make/test.mk`
change: both suites already run in `test-launchd-drivers` (:81 notify, capacity, heartbeat).

## 5. Test arms (all in `test_driver_notify.sh`, all under the existing `run_bounded` outer watchdog)

Common harness `run_exit BLOCK MODE` runs the extracted block in `/bin/bash -c` under `set -uo pipefail`. It does the following:
- sources `$MC_BND` and the extracted `_mc_exit_notice` + `notice_helpers.sh`
- sets `NOTIFY_TIMEOUT=${MISSION_NOTIFY_TIMEOUT:-30}` and `log()` → trace
- sets `MSG_FROM`, `MISSION_REPO`, `MISSION_GH_ISSUE=${ISSUE-635}`, `LOG`, `STATE_DIR` (per-arm `mktemp -d`) and `MISSION_NAME=v1`
- for the refusal block: stubs `select_model(){ return 1; }`, `_mc_load_ration(){ :; }` and
  `_mc_reset_hint(){ :; }`; sets `PREFS`, `CONTROLLER_FALLBACK` and `BLOCKED_FILE=$STATE_DIR/b`. These are function stubs, which is safe because they are called directly and never through `_mc_bounded`.
- for the final block: sets `RC`, `MC_PAUSED=0`, `MISSION_LOG_FILE` (with or without a `## ` record) and `pre_last_record=""`
- echoes the trace, then `EXIT_RC:$?`. The block's own `exit` ends the inner shell, so the
  outer harness captures the inner rc from `run_bounded`'s return and prints it.

Hang arms use `MISSION_NOTIFY_TIMEOUT=2 MC_BOUNDED_TERMINATION_GRACE=0`, which gives about 4 s
per bounded call and about 8 s per path. They use an outer watchdog of 30 s and assert
elapsed ≤ 14 s. That ceiling follows arm (5)'s lesson: slow macOS runners took 8 s on a 2 s
bound. An unbounded send trips the 30 s watchdog → rc 124 → FAIL. The test can never hang CI.

| Arm | Setup | Asserts | How shown RED pre-fix |
|-----|-------|---------|-----------------------|
| **E1** refusal hang | `STUB_HANG_AILANG=1 STUB_HANG_GH=1`, refusal block | returns before watchdog, elapsed ≤ 14 s, `EXIT_RC:1`, `LOG:WARNING: … FAILED to send` and `FAILED to post to issue #635` both contain `timed out after 2s` | M1 adds markers + arms on the *unfixed* sites: the bare send hangs → 30 s watchdog → FAIL |
| **E2** late-kill hang | same stubs, final block, RC=143, log gains `## ` | as E1, `EXIT_RC:143` | same |
| **E3** rc-fail hang | same stubs, final block, RC=1, no record | as E1, `EXIT_RC:1`, `*-rcfail.episode` = `1` | same |
| **E4** refusal normal | healthy stubs | trace has `AILANG:messages send controlplane` + `--title Mission iteration blocked: no usable model` + `GH:issue comment 635` + `**no usable controller**`; `EXIT_RC:1`; `$BLOCKED_FILE` exists | preservation control (green pre-fix by design). Mutation drill on a scratch copy via `MC_NOTIFY_DRIVER`: delete the gh call → RED |
| **E5** late-kill normal | healthy, RC=143, record | title `killed post-record (rc=143) — work landed`; GH body has `rc=143 after landing its record`; rcfail marker removed | mutation: swap the title text → RED |
| **E6** rc-fail normal | healthy, RC=1 | title `Mission iteration FAILED (rc=1)`; GH arg contains **`--body ⚠️ Mission iteration **FAILED to complete** (rc=1 — timeout or crash)`** (F8 prefix at offset 0 of the body) | mutation: drop the `⚠️ ` → RED; also covered by gate0's F8 control |
| **E7** store scoping + reason | `AILANG_RC=1 STUB_AILANG_OUT='Error: boom'`, each of the 3 paths | `LOG:WARNING:` line contains `store=<gcp> proj=<ailang-multivac>` and `Error: boom` (the stub prints its env to stdout, and the WARNING tails `MC_BOUNDED_OUT`) | pre-fix: no WARNING logged at all (`2>/dev/null`, no log) → RED; mutation: drop `env …` → `store=<>` → RED |
| **E8** episode gates preserved | refusal twice with the same `BLOCKED_FILE`; rc-fail twice with RC=1 | 2nd run: 0 `AILANG:` and 0 `GH:` records, suppression logged, exit codes unchanged | preservation; mutation: remove the `[ -f "$BLOCKED_FILE" ]` branch → RED |
| **E9** duplicate title is not a WARNING | `AILANG_RC=1 STUB_AILANG_OUT='duplicate message exists (ID: x)'` | `already in the inbox` logged, no `LOG:WARNING: … FAILED to send` | pre-fix: nothing logged → RED (`already in the inbox` absent) |
| **G1** source guard: bounded | awk joins `\`-continued lines of `$DRV`, drops lines whose first non-blank char is `#`, selects logical lines matching `ailang messages send` or `gh issue comment` | every selected logical line contains `_mc_bounded`; prints offending line numbers on fail | against HEAD: FAILS listing 2038, 2041, 2529, 2532, 2550, 2553 |
| **G2** source guard: store | same selection, `ailang messages send` only | every one contains `env AILANG_STORAGE_MESSAGING=gcp` | against HEAD: FAILS on the 3 bare sites + 2511 |
| **G3** F8 literal present | `grep -F '⚠️ Mission iteration **FAILED to complete** (rc=' "$DRV"` | exactly one match, inside an `_mc_exit_notice` call's GH-body argument | green/green tripwire (the gate0 suite's own F8 control also covers it) |

The G1 positive control must show the instrument can see a positive: the selection has to
include the three already-bounded helper sites (:239/:298/:334 equivalents). Assert
`selected ≥ 6` after the fix, so an empty selection cannot pass vacuously.

## 6. Milestones

1. **M1: red first.** Add the 4 markers, the `MC_NOTIFY_DRIVER` override and arms E1–E9 + G1–G3.
   Run the suite unpiped and record the RED lines (E1–E3 watchdog, E7, E9, G1, G2) in the
   sprint state. Do not commit a red suite to dev: M1 and M2 land in one commit, and the red log
   lives in `.ailang/state/sprints/`.
2. **M2: fix.** Add `_mc_exit_notice`, rewrite A/B/C and add the capacity-suite extraction.
   The suite goes GREEN. Run the mutation drills on a scratch copy
   (`cp tools/launchd/mission-control.sh "$TMPDIR/mut.sh"`, edit, then
   `MC_NOTIFY_DRIVER="$TMPDIR/mut.sh" /bin/bash tools/launchd/test_driver_notify.sh`). Never
   use `git checkout <file>`.
3. **M3: SLOT NOTIFY store scoping.** Make the env change at :2511 and update the heartbeat
   regex. G2 then has no exemption.

## 7. Changelog fragment

`changelogs/unreleased/2026-10-12-exit-path-notices-bounded.md`:

```markdown
### Fixed — mission driver exit-path notices are bounded and reach the canonical inbox
The no-usable-controller refusal, the late-kill-after-record notice and the rc-fail notice in
`tools/launchd/mission-control.sh` called `ailang messages send` / `gh issue comment` unbounded,
so a hung send wedged the exiting process (and with it the launchd job). They now go through
`_mc_exit_notice` (one `_mc_bounded "$NOTIFY_TIMEOUT"` shot per channel, non-fatal, failure
logged with its reason), with the message store scoped per command to prod Firestore exactly as
`_mc_notify` does. Under launchd they previously landed in the rig's local SQLite inbox. The slot
verdict notice gets the same store scoping. Notice titles, bodies and exit codes are unchanged.
```

## 8. Acceptance criteria

1. `make test-launchd-drivers` is green under `/bin/bash` 3.2.57, run **unpiped** (a pipe
   reports the pipe's exit code).
2. `bash -n tools/launchd/mission-control.sh` and `bash -n` on every edited test file both pass.
3. G1 and G2 report zero violations, and the selection count is ≥ 6.
4. E1–E3 each return in ≤ 14 s with both stubs hanging, with exit codes 1 / 143 / 1.
5. `scripts/test_mission_gate0_self_notices.sh` is green, so the F8 literal still extracts from the driver.
6. `git diff 06d524c59 -- tools/launchd/mission-control.sh` shows no change to any title or body
   string, any exit statement, or any routing/ration line. Reviewer check:
   `git diff -U0 06d524c59 -- tools/launchd/mission-control.sh | grep '^[-+].*--title\|^[-+].*exit '`
   lists only the moved title arguments, with identical text.
7. The diff touches only `tools/launchd/**`, `changelogs/**`, `design_docs/**` and `.ailang/state/sprints/**`.
8. The red→green evidence for every row in §5 is recorded, either from the M1 run or from a named mutation.

## 9. Verification commands

```bash
cd /Users/voightkampff/.ailang-worktrees/fleet-i33
/bin/bash --version | head -1                       # expect 3.2.57 on the rig
bash -n tools/launchd/mission-control.sh
for f in tools/launchd/test_driver_notify.sh tools/launchd/test_controller_capacity.sh tools/launchd/test_mission_heartbeat.sh; do bash -n "$f" || echo "SYNTAX FAIL $f"; done
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_driver_notify.sh
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_controller_capacity.sh
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_mission_heartbeat.sh
/bin/bash tools/launchd/lib/suite-env.sh scripts/test_mission_gate0_self_notices.sh
make test-launchd-drivers; echo "rc=$?"              # UNPIPED
grep -c -F '⚠️ Mission iteration **FAILED to complete** (rc=' tools/launchd/mission-control.sh   # expect 1
git diff --name-only 06d524c59
```

## 10. Out of scope and follow-ups to file

- **`driver:exit-notice-titles-undated`**: date-stamp the A/B/C messages titles via `_mc_notice_title`
  so a prod duplicate does not suppress them forever (§3.2). This is a text change.
- **`nightly-eval:unbounded-sends`**: `tools/launchd/nightly-eval.sh` has about 9 bare
  `ailang messages send` calls (:84, :101, :396, :430, :491, :572, :588, :604, :620). They are
  the same class, but not part of the mission driver.
- Retrying or spooling the exit notices is out of scope. One bounded shot per channel is enough
  (§3.1).
