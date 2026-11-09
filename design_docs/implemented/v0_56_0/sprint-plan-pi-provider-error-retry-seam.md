# Sprint Plan: M-FLEET-PI-PROVIDER-ERROR-RETRY-SEAM

## Placement and Human-Decision Rules
This file is in the planned root as `sprint-plan-<slug>.md`. No decision is still OPEN, so there
are **no human-decision gates**. Rulings copied in, not to be reopened: retry policy (TRANSIENT_SIG
contents, TRANSIENT_RETRIES, TRANSIENT_BACKOFF) is routing/policy class (HD-2a) and stays as it is.
`.pi/extensions/**` is off limits (D-FLEET-3).

## Summary
Ticket `pi:provider-error:no-retry` (stapledon, filed 2026-10-07) asks for a bounded retry when the
provider returns `finish_reason=error`. **The driver already has that retry.** #1391 (`7eaf9bece`,
2026-09-29 07:24Z) added `Provider finish_reason: error` to `TRANSIENT_SIG`. That was 6.5 h after
the 00:52Z event and 8 days before the ticket was filed. What is missing is a test of the **seam**:
pi prints the error to stderr and exits 1, the driver's `2>&1` sends it into `$LAB/driver.log`, the
retry loop matches it, the same rung is retried, and the retry reuses the same `--session-id` with
the RESUME prompt. The only existing test (`test_driver_notify.sh` case 13) checks that the string
is a member of `TRANSIENT_SIG`. This sprint adds one end-to-end case to
`tools/launchd/test_controller_capacity.sh`. It is a test-only change and touches no driver behaviour.

Fleet iteration 70; base `96d69f05b`; branch `fleet/i70-provider-error-retry-seam`.
**Duration:** one executor run, about 45–60 min. **Risk:** low. **Dependencies:** none.

## Current Status Analysis
Measured by the planner, not copied from the controller:
- C1: `tools/launchd/mission-control.sh:1158-1160` has `TRANSIENT_RETRIES` (default 3 attempts in
  total), `TRANSIENT_BACKOFF` (default 45, multiplied by the attempt number) and `TRANSIENT_SIG`,
  which ends in `|Provider finish_reason: error`. The loop is at `:2552-2600`. It checks
  `RUNTIME_QUOTA_SIG` first, then `grep -qiE "$TRANSIENT_SIG"` over this attempt's `$LOG` lines.
  On a match it logs `transient API error (rc=$RC) attempt $attempt/$TRANSIENT_RETRIES — retrying in …`.
- C2: the pi launch is at `:2480`: `--session-id "$MC_PI_SESSION_ID" -p "$_mc_pi_prompt" < /dev/null ) >>"$LOG" 2>&1 &`.
  The prompt choice is at `:2468-2469`: `$MC_PI_RESUME_PROMPT` is used when `_mc_pi_session_exists`
  returns true. The full `-p` value is `<controller note>\n\n<prompt>` (`lib/pi-controller-note.sh`),
  so the prompt is its **last line**.
- C3: real pi 0.85.1 `dist/modes/print-mode.js:115-117` writes `errorMessage` to stderr with
  `console.error` and sets exit code 1. The stapledon session
  `~/.pi/agent/sessions/--Users-voightkampff-dev-sunholo-data-stapledons-godot--/2026-09-29T00-46-20-125Z_01a0eaa0-….jsonl`
  ends with `"stopReason":"error",…,"errorMessage":"Provider finish_reason: error"`. Its mtime is 02:52 +0200, which is 00:52Z.
- C4: pi stores a session as `<agent dir>/sessions/--<cwd slug>--/<ts>_<id>.jsonl`, using the
  **physical** `process.cwd()` (`dist/main.js:452`, `dist/core/session-manager.js:242-246`). It writes
  the file only once an assistant message exists (`session-manager.js:739-766`). An errored completion
  is an assistant message, so attempt 1 does create the file. `_mc_pi_session_exists` builds the slug
  from the **logical** `$REPO`.
  On macOS `$TMPDIR` (`/var/folders/…`) is a symlink to `/private/var/…`. So with a pi-faithful fake
  and the lab's current `ROOT="${TMPDIR:-/tmp}/…"`, the driver looks in the wrong directory and the
  resume assertion fails for a reason that has nothing to do with the seam. **The lab ROOT must be
  canonicalized with `pwd -P`.** Production `MISSION_WORKDIR`s (ailang, ailang-fleet,
  stapledons-godot, ailang-world, ailang-triage) are all physical, so nothing is affected live.
- C5: base run on the pristine worktree: `/bin/bash tools/launchd/test_controller_capacity.sh` gives
  rc 0, `15 cases passed, fail=0`, in about 14 s. Base shellcheck (0.11.0) on that file: **61** findings
  (`-f gcc | wc -l`) = SC1091×6, SC2015×2, SC2034×19, SC2129×1, SC2319×27, SC2329×6. 46 of them are
  warning-level or higher.
- C6: the planner built a prototype of this exact change in a scratch tree outside the worktree
  (`/tmp/i70proto`). Results: 16 passed, fail=0. The 15 existing PASS lines are identical to base
  (`diff` shows only the added line). Nothing was written under the real `~/.pi`. Every mutation in
  the table below was run there and behaved as listed.

**Capacity:** about 45 LOC of test plus about 8 LOC of changelog. This estimate comes from the
bounded ticket, not from historical velocity.

## Registry Reuse Audit
`M1_PROVIDER_ERROR_RETRY_SEAM`: **none**. This is shell test instrumentation with no package-like
capability and no `.ail` code, so the registry search and the AILANG syntax/example gates do not apply.

## Proposed Milestone
### M1_PROVIDER_ERROR_RETRY_SEAM — pin stderr → `$LOG` → retry → resume, end to end
**Files the executor may touch (exhaustive):**
- `tools/launchd/test_controller_capacity.sh`
- `changelogs/unreleased/2026-10-29-fleet-i70-provider-error-retry-seam.md` (create; opens with `### Changed`)
- this plan (completion notes only)

Do not change `mission-control.sh`, `lib/`, `.pi/`, the make wiring, or any other file. Every mutation runs on scratch copies.

**Tasks:**
1. **Hermetic lab.**
   - Add `MISSION_TRANSIENT_RETRIES` to the `unset` line, so an exported value cannot change the
     `attempt 1/3` text or the retry count.
   - Change `ROOT=` to `ROOT="$(cd "${TMPDIR:-/tmp}" && pwd -P)/ctl-capacity-$$"` (see C4).
   - In `runcase`, export `PI_CODING_AGENT_DIR="$LAB/pi-agent"` next to `MC_LAB`.
2. **Fake pi, extended minimally.** Do not change the `pi.argv` format, which stays one `--model`
   per line because `pi_count` depends on it.
   - Also parse `--session-id` and `-p`.
   - Per invocation: append the session id to `$MC_LAB/pi.sid`, and write the full `-p` value to
     `$MC_LAB/pi.p.<N>`, where N is the global invocation index (the line count of `pi.argv`).
   - When a session id is given, behave like real pi:
     - Fail loudly with exit 98 if `PI_CODING_AGENT_DIR` is unset, so the fake never falls back to the real `~/.pi`.
     - Compute the slug from `pwd -P` using pi's rule: drop the leading `/`, map `/\:` to `-`.
     - Append one JSON line to the existing `*_<sid>.jsonl`, or create `<ts>_<sid>.jsonl` if there is none.
   - Stream selector: `script()` takes an optional 6th argument `stderr`, which touches
     `<script>.stderr`. When that marker exists, the fake prints the text with `>&2`; otherwise it
     prints to stdout as before.
   - Reference (verified in the prototype):
     ```bash
     model=""; sid=""; prompt=""; prev=""
     for a in "$@"; do
       case "$prev" in --model) model="$a" ;; --session-id) sid="$a" ;; -p) prompt="$a" ;; esac
       prev="$a"
     done
     printf '%s\n' "$model" >> "$MC_LAB/pi.argv"
     inv=$(wc -l < "$MC_LAB/pi.argv" | tr -d ' ')
     printf '%s\n' "$sid" >> "$MC_LAB/pi.sid"; printf '%s' "$prompt" > "$MC_LAB/pi.p.$inv"
     # … existing key/n/f/rc/stamp lines unchanged …
     if [ -n "$sid" ]; then
       [ -n "${PI_CODING_AGENT_DIR:-}" ] || { echo "FAKE pi: PI_CODING_AGENT_DIR unset" >&2; exit 98; }
       cwd=$(pwd -P); slug=$(printf '%s' "${cwd#/}" | tr '/\\:' '---')
       sdir="$PI_CODING_AGENT_DIR/sessions/--$slug--"; mkdir -p "$sdir"
       set -- "$sdir"/*_"$sid".jsonl
       if [ -f "$1" ]; then sf="$1"; else sf="$sdir/$(date -u +%Y-%m-%dT%H-%M-%S)-000Z_$sid.jsonl"; fi
       printf '{"type":"message","inv":%s}\n' "$inv" >> "$sf"
     fi
     if [ -f "$f.stderr" ]; then sed -n '3,$p' "$f" >&2; else sed -n '3,$p' "$f"; fi
     ```
3. **One new case**, `provider-finish-reason-error-retries-and-resumes`, placed after `transient-unchanged`:
   - `CHAIN="$CH3"`
   - `script openrouter/a 1 1 - 'Provider finish_reason: error' stderr`
   - `script openrouter/a 2 0 complete done`
   - then `runcase` and the assertions in the acceptance list.
4. **Mutation drill** (table below): run every row and record killed/survived in the completion notes.
5. **Changelog fragment** under `### Changed — mission driver: provider finish_reason=error retry is now pinned end to end (2026-10-29)`.
   It names the ticket, says the retry already existed since #1391, and says this change is test-only.

**Why the 15 existing cases are unaffected:**
- None of them passes a 6th `script` argument, so their text still goes to stdout.
- `pi.argv` is byte-identical.
- The new `pi.sid`, `pi.p.N` and session files are read only by the new case.
- The session dir is per-lab, so no state crosses cases.
- The canonical ROOT changes no path that any assertion reads.
- One side effect: a second pi invocation inside an existing case (a retry, or a pi→pi re-walk) now
  receives the RESUME prompt, as it would in production. No existing assertion reads `-p`.
- Proof: `diff <(grep '^PASS' base.out) <(grep '^PASS' new.out)` must show exactly one added line.

**Acceptance criteria.** Each assertion reads something downstream of the mechanism, not a value set alongside it.
- [ ] A1: process rc 0 (`CASE_RC`).
- [ ] A2: `pi_count openrouter/a` = 2 and `pi_count openrouter/b` = 0: the same rung is retried and there is no re-walk.
- [ ] A3: `state_has "demoted=[]"`: no demotion.
- [ ] A4: `has_log "transient API error (rc=1) attempt 1/3"`.
- [ ] A5: `has_log 'Provider finish_reason: error'` (the stderr text reached `$LOG`), and it is absent from `$LAB/err` (it did not escape past the log).
- [ ] A6: `pi.sid` is exactly two lines, both `cap-session-$$`. This is the runcase's `MC_PI_SESSION_ID`; `$$` is the same in the subshell.
- [ ] A7: `tail -n 1 pi.p.1` = `fake prompt`, and `tail -n 1 pi.p.2` = `resume`.
- [ ] A8: `has_log "slot-verdict: COMPLETED"`.
- [ ] Stream: the fake emits the error with `>&2` (stderr marker). No in-suite assertion tells stderr from stdout. That is proven by mutation M2 being killed, and by fake mutation F2 surviving as expected.
- [ ] `/bin/bash tools/launchd/test_controller_capacity.sh` → `16 cases passed, fail=0`, with the 15 base PASS lines unchanged.
- [ ] `make test-launchd-drivers` rc 0. Run it unpiped: a piped make reports the pipe's exit code.
- [ ] `shellcheck -f gcc tools/launchd/test_controller_capacity.sh | wc -l` is no more than 61 plus the number of new `[ … ]; expect … $?` lines. Every added finding must be SC2319, which the suite's existing `expect "…" $?` idiom produces (the prototype gave 66 = 61 + 5 SC2319). No other rule code may increase.
- [ ] `ls -d ~/.pi/agent/sessions/*ctl-capacity* 2>/dev/null` prints nothing after the suite runs.
- [ ] `make check-changelog` passes and the fragment opens with `### `.
- [ ] Every row of the mutation table matches its expected outcome, recorded in the completion notes.

**Mutation table.** bash 3.2.57. Mutate with perl on scratch copies only, never with bash
`${s//…}` over the 164 KB driver. Before each run, assert the mutation applied:
`[ "$(diff "$D" "$M/<id>.sh" | grep -c '^>')" -eq 1 ]`.

Setup:
```bash
W=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter70-provider-retry
D=$W/tools/launchd/mission-control.sh; T=$W/tools/launchd/test_controller_capacity.sh; M=$(mktemp -d)
run() { MC_CAPACITY_DRIVER="$1" /bin/bash "$T" > "$1.out" 2>&1; echo "rc=$? $(tail -1 "$1.out")"; grep unmet "$1.out"; }
```

| Row | Mutation | Command (then `run $M/<id>.sh`) | Killed by (prototype result) |
|---|---|---|---|
| M1 | delete `\|Provider finish_reason: error` from TRANSIENT_SIG | `perl -pe 's/\|Provider finish_reason: error"$/"/ if /^TRANSIENT_SIG=/' $D > $M/M1.sh` | A1 A2 A4 A6 A7(2) A8 |
| M2 | drop `2>&1` from the **pi** launch line only (the claude line keeps it) | `perl -pe 's/ < \/dev\/null \) >>"\$LOG" 2>&1 &$/ < \/dev\/null ) >>"\$LOG" &/ if /--session-id "\$MC_PI_SESSION_ID"/' $D > $M/M2.sh` | A1 A2 A4 **A5 (both halves)** A6 A7(2) A8. This kill is the stderr-path proof. |
| M3 | `_mc_pi_session_exists` always false | `perl -pe 's/^  ls "\$\{PI_CODING_AGENT_DIR.*$/  return 1/' $D > $M/M3.sh` | **A7(2) only**. The retry still happens; only the resume prompt is lost. |
| M4 | fresh `--session-id` per attempt | `perl -pe 's/--session-id "\$MC_PI_SESSION_ID"/--session-id "\$MC_PI_SESSION_ID-a\$attempt"/ if /^    \( cd "\$REPO" && pi --model/' $D > $M/M4.sh` | A6, A7(2) |
| M5 | TRANSIENT_RETRIES default 1 | `perl -pe 's/MISSION_TRANSIENT_RETRIES:-3/MISSION_TRANSIENT_RETRIES:-1/ if /^TRANSIENT_RETRIES=/' $D > $M/M5.sh` | A1 A2 A4 A6 A7(2) A8, plus the existing `transient-unchanged` |
| F1 | fake does not write the session file | scratch test tree (below): `perl -pe 's/^  printf .\{"type":"message","inv":%s\}\\n. "\$inv" >> "\$sf"$/  : no session write/'` | A7(2). Shows that A7 depends on the session file the fake creates. |
| F2 | fake prints the error to **stdout** | scratch test tree: `perl -pe 's/^if \[ -f "\$f.stderr" \]; then sed -n .3,\$p. "\$f" >&2;/if [ -f "\$f.stderr" ]; then sed -n "3,\\\$p" "\$f";/'` | **Survives (expected).** stdout is what `transient-unchanged` already covers. The stream is pinned only by M2, which F2 would make survive. |
| C1 | control: logical ROOT (undo task 1's `pwd -P`) | scratch test tree: `perl -pe 's/^ROOT="\$\(cd "\$\{TMPDIR:-\/tmp\}" && pwd -P\)\/ctl-capacity-\$\$"$/ROOT="\${TMPDIR:-\/tmp}\/ctl-capacity-\$\$"/'` | A7(2). Shows the canonical ROOT is load-bearing (C4). |

Scratch test tree for F1/F2/C1. The test derives `HERE`/`REPO_ROOT` from its own location, so do
not add a copy inside the worktree:
```bash
S=$(mktemp -d); mkdir -p "$S/tools/launchd"
ln -s "$D" "$S/tools/launchd/mission-control.sh"; ln -s "$W/tools/launchd/lib" "$S/tools/launchd/lib"
perl -pe '<row expr>' "$T" > "$S/tools/launchd/test_controller_capacity.sh"; /bin/bash "$S/tools/launchd/test_controller_capacity.sh"
```
The eight runs take about 14 s each and can run in parallel, because ROOT includes `$$`. Poll their
result files in a bounded `date +%s` loop.

**Risks:**
- The agent's Bash tool shell is zsh, which does not word-split and aborts on unmatched globs. Run
  the suite and every mutation command under `/bin/bash` (3.2).
- If a future driver change moves the prompt off the last line of `-p`, A7 breaks loudly. That is intended.

## Success Metrics
- 16/16 cases; 0 changes outside the three named files; mutation table reproduced (6 killed, F2 the expected survivor, C1 killed).

## Dependencies
- None. Base `96d69f05b`.

## Open Questions
- None open.
- Follow-up candidate, **not** in this sprint because it changes driver behaviour: `_mc_pi_session_exists`
  (and the pi arm of `_mc_progress_bytes`) slug the logical `$REPO`/`$PWD`, while pi uses the
  physical cwd. A symlinked `MISSION_WORKDIR` would silently lose resume and progress tracking.
  Production workdirs are all physical today.

## Notes
- The controller asked for no sprint JSON (plan file only), so the `create_sprint_json.sh` step is skipped on purpose.
- Real pi also prints `Warning: No project session found with id …` to stderr on a fresh id
  (`main.js:343`). The fake leaves this out: it matches no RUNTIME_QUOTA_SIG/TRANSIENT_SIG token, and leaving it out keeps the fake minimal.
- The companion ticket `stall:watchdog:slow-on-hung-tool` is out of scope.

## Execution notes (M1_PROVIDER_ERROR_RETRY_SEAM, 2026-10-29)
- **Done:** case `provider-finish-reason-error-retries-and-resumes` added after `transient-unchanged`; fake pi extended as specified (`pi.sid`, `pi.p.N`, session file, `script` 6th arg `stderr`); ROOT canonicalised with `pwd -P`; `PI_CODING_AGENT_DIR` exported per lab; `MISSION_TRANSIENT_RETRIES` unset. Changelog fragment written. No other file touched.
- **Gate 1:** `/bin/bash tools/launchd/test_controller_capacity.sh` rc 0, `16 cases passed, fail=0`. The 15 base PASS lines are unchanged (`diff` shows one added line).
- **Gate 2:** `make test-launchd-drivers` (unpiped) rc 0, `run-suites: all 52 suites passed`, `launchd drivers: tests + bash 3.2 syntax OK`. No flake this run.
- **Gate 3:** shellcheck 66 findings (base 61). The increase is 5 SC2319, as predicted. No other rule rose. The first draft also produced one SC1010 (a bare `done` argument to `script`); quoting it as `"done"` removed it.
- **Gate 4:** `make check-changelog` rc 0.
- **Gate 5:** `find ~/.pi -maxdepth 4 -name '*ctl-capacity*'` rc 0, empty output.
- **Mutations:** run on scratch copies with the 8-row setup from the table, every row asserted applied (one changed line) before running. The drill was run twice. The first run predates the SC1010 quoting fix. The table below is from the second run, on the final file. Both runs gave the same results.

| Row | Result | Red assertions |
|---|---|---|
| M1 drop `Provider finish_reason: error` from TRANSIENT_SIG | killed, 15/1 | A1 A2 A4 A6 A7(2) A8 |
| M2 drop `2>&1` on the pi launch | killed, 15/1 | A1 A2 A4 **A5 both halves** A6 A7(2) A8 |
| M3 `_mc_pi_session_exists` always false | killed, 15/1 | A7(2) only |
| M4 fresh `--session-id` per attempt | killed, 15/1 | A6 A7(2) |
| M5 TRANSIENT_RETRIES default 1 | killed, 14/1 | A1 A2 A4 A6 A7(2) A8, plus `transient-unchanged` |
| F1 fake writes no session file | killed, 15/1 | A7(2) |
| F2 fake prints the error to stdout | **survived** (16/0), as expected | none |
| C1 logical ROOT (no `pwd -P`) | killed, 15/1 | A7(2) |

Every row matches the table. F2 survives as the plan expects; the stream is pinned only by M2.
- **Deviations:** none from the plan's content. The only addition is the SC1010 quoting fix noted above, which the plan did not anticipate.
