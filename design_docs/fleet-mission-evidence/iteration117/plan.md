# Iteration 117 plan: motoko connection probe self-test flakes (signatures T and R)

Tickets: `ci:launchd-suite-flakes-openrouter-peer-and-gh-hang`, `ci:motoko-probe-refusing-live-path-unexpected-success`.
Worktree `/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter117-probe-flakes`, base `22a72e7c5` (origin/dev).
Planner host: darwin/arm64, 16 cores, `/bin/bash` = GNU bash 3.2.57(1)-release (arm64-apple-darwin25).

## Verdicts

| Sig | Controller hypothesis | Verdict | Fixed by |
|-----|-----------------------|---------|----------|
| T | SIGTERM lost in the timer child between fork and exec | **CONFIRMED**, mechanism pinned to bash 3.2's deferred terminating-signal handler. The suite's own `trap … EXIT` installs that handler. Reproduced at 1.07% to 10.2% with the suite's own functions, 0% with SIGKILL, and 0% with no EXIT trap. | `kill -KILL` for the timer injection, plus a guard: the timer arm runs with TERM ignored, so it fails every time if anyone reverts to TERM. |
| R | The `lsof` stub reads an empty lane file and falls into the OpenRouter `else` branch | **CONFIRMED with an amendment.** There are two windows, not one: W1, where the file has been created but not yet written (the file is new per pid, so the problem is creation, not truncation), and W2, where `rm -f` lands between the stub's `[[ -f ]]` check and its `cat`. Deterministic injection into either window reproduces both CI signatures byte-for-byte. I could not reproduce it naturally on this host. | The `lsof` stub now fails closed: a lane it cannot name samples no peer. A forced-read guard arm proves the window is reached. |

No probe change is needed (`tools/eval/motoko_connection_probe.sh` is untouched); both defects live in the self-test file.

## 1. Verification Log

Every row below was run by the planner. Scratch scripts lived under `.scratch-planner/`, which has now been deleted. The scripts are described inline so they can be rebuilt.

| # | Claim | Command (abridged where noted) | Observed |
|---|-------|--------------------------------|----------|
| V1 | The T signature is real on CI (11-02) | `gh api --allow-escape-sequences repos/sunholo-data/ailang/actions/jobs/125416080744/logs` (run 41485837393), ANSI-stripped, grep `measurement failure evidence\|not ok\|bound derivation` | `# bound derivation: r=159/s r_real=168/s … scale=3`; `# measurement failure evidence mode=timer rc=0 refusal_count=0 derived_count=1`; `not ok - bound measurement refuses an unsuccessfully terminated timer (rc=0 expected_rc=73 refusal=0 derived=1)`. **Positive control in the same log:** `mode=exit1 rc=72 refusal_count=1` and `mode=nonexec rc=71 refusal_count=1`. |
| V2 | The R "unexpectedly succeeded" signature is real (10-15) | same API, run 38805887596 → job 116666641045 | `not ok - refusing live path refuses with the control-void message unexpectedly succeeded` (host `r=458/s`) |
| V3 | R's OpenRouter-leak variant is real (10-16) | same API, run 38914956664 → job 117020112147 | `lacked expected message: treatment verdict is void`. First lane peers `["127.0.0.1:11434","203.0.113.8:443"]`, second `["127.0.0.1:11434"]`, then `INSTRUMENT FAILURE: treatment peer set contains OpenRouter endpoint: 203.0.113.8:443` (host `r=483/s`) |
| V4 | The suite sets a top-level EXIT trap | `grep -n "^trap '" tools/eval/test_motoko_connection_probe.sh` | `7:trap '…; rm -rf "$tmp_dir"' EXIT` |
| V5 | Timer injection is a SIGTERM sent right after the fork | `grep -n 'kill -TERM "\$timer"' …test…` | `230:    kill -TERM "$timer" 2>/dev/null \|\| true` (directly after `228: sleep "$window" & timer=$!`) |
| V6 | bash 3.2: an EXIT trap in a non-interactive shell installs the terminating-signal handlers | Apple source `gh api -H "Accept: application/vnd.github.raw" repos/apple-oss-distributions/bash/contents/bash-3.2/trap.c` | `480: if (sig == EXIT_TRAP && interactive == 0)` / `481: initialize_terminating_signals ();` |
| V7 | That handler only records the signal unless `terminate_immediately` is set | same, `sig.c` lines 453-465 | `termsig_sighandler: terminating_signal = sig; if (terminate_immediately) {…termsig_handler (sig);} SIGRETURN (0);` The flag is acted on only at `QUIT`/`CHECK_TERMSIG` (`quit.h:34-52`). |
| V8 | Async function subshells (the run_bounded child) do not reset those handlers | same, `execute_cmd.c` | `3014: /* reset_terminating_signals (); */ /* XXX */` (contrast `1275: reset_terminating_signals ();` in `( … )` subshells). The disk-command child path (`execute_cmd.c:3704-3770`) has no `QUIT` between `make_child` and `shell_execve`. |
| V9 | Mechanism T reproduces with the suite's OWN functions | `t_realfn.sh`: `eval` of `run_bounded` and `measure_fork_rate` extracted verbatim from the suite by awk, top-level EXIT trap, `PROBE_SELFTEST_MEASUREMENT_FAILURE=timer`, loop `run_bounded o e 10 -- measure_fork_rate 1 stim` for N s | normal QoS, 90 s: **39 × rc=0, 3592 × rc=73 (1.07%)**. `taskpolicy -b`, 150 s: **72 × rc=0, 633 × rc=73 (10.2%)** |
| V10 | Discriminator: SIGKILL is never lost (same harness, same conditions) | `sed` the suite copy so line 230 reads `kill -KILL`, rerun V9 | normal QoS, 90 s: **0 / 8415**. `taskpolicy -b`, 150 s: **0 / 6749** |
| V11 | Discriminator: without the EXIT trap, SIGTERM is never lost | V9 with `NOTRAP=1` (no top-level trap), `taskpolicy -b`, 150 s | **0 / 6736** (rc=73 every trial) |
| V12 | One trial per fresh shell, which is the shape of the real arm | `t_fresh.sh`: a new `/bin/bash -c` per trial, same extracted functions, 120 s | TERM: **2 / 440 (0.45%)**. KILL: **0 / 433**. CI: 5 / ~819 ≈ 0.6% |
| V13 | The real full-suite timer arm is NOT reproduced locally (this is a negative result) | `t_arm_loop*.sh`: the arm's exact recursion `env -u … PROBE_SELFTEST_MEASUREMENT_FAILURE=timer PROBE_SELFTEST_DERIVATION_ONLY=1 /bin/bash <suite>` | quiet: 0/200 rc=0. `taskpolicy -b`, 300 s: 0/654 (all `rc=73 derived=0`). **Positive control:** the same harness shape (V9/V12) does reproduce. The full-suite rate on this host is below about 1/300. |
| V14 | R stub code: existence check, then a separate read, then unknown → OpenRouter | `grep -n` on the suite | `487: [[ -f "$lane_file" ]] \|\| exit 1`, `488: lane=$(cat "$lane_file")`, `492: else` → `203.0.113.8:443`. The driver writes at `505: echo "$lane" > "${PROBE_STUB_STATE}.$$"` (new file per pid) and removes it at `529: sleep …; 530: rm -f …` |
| V15 | The probe keeps whatever lsof prints, whatever its rc | `grep -n 'lsof -nP' tools/eval/motoko_connection_probe.sh` | `224: lsof … -p "$pids" 2>/dev/null >> "$raw_file" \|\| true`. A stub exit 1 prints nothing (harmless). An empty-lane read prints the header plus the OR peer with rc 0, and that gets classified. |
| V16 | Hermetic lanes sample only the driver pid (no descendants), and the refusal arm has a private state prefix | `grep -n 'pgrep -P' …probe…`; canonical pgrep stub prints only under `PROBE_TEST_PGREP_LOOP`; `grep -n '"\$probe" treatment treatment'` | `202: pgrep -P …` (stub returns nothing). Refusal arm: `PROBE_STUB_STATE="$tmp_dir/lane-refusal"` (line 597), so no lane file left over from another arm can reach it. This rules out cross-arm pid reuse. |
| V17 | W1 injection gives both CI signatures (original stubs) | `r_inject.sh <orig> <probe> W1 {1,2}`: a wrapper creates the driver's lane file empty, holds it 0.5 s, then `exec`s the verbatim stub (same pid), on lane 1 or lane 2 only. Refusal invocation `treatment treatment` | idx 1: `probe_rc=1`, peers `["127.0.0.1:11434","203.0.113.8:443"]` / `["127.0.0.1:11434"]`, `INSTRUMENT FAILURE: treatment peer set contains OpenRouter endpoint: 203.0.113.8:443`, **byte-identical to V3**. idx 2: `probe_rc=0`, which is **V2** ("unexpectedly succeeded") |
| V18 | W2 injection gives the same two signatures (original stubs) | `r_inject.sh … W2 {1,2}`: `sed` inserts, on the 2nd+ sample of the injected lane only, `while [[ -f "$lane_file" ]]; do sleep 0.05; done` between line 487 and line 488 | idx 1: identical to V17 idx 1. idx 2: `probe_rc=0`. |
| V19 | Fixed stub under the same injections | `r_inject.sh suite_fixed.sh … W1 1`, `W1 2`, `W1 1 "treatment control"`, `W2 1` | W1 idx 1 and idx 2 (refusal): `probe_rc=1 … treatment verdict is void` (correct). W1 idx 1 on the success lanes: `probe_rc=0`, control peers `["203.0.113.8:443"]`. W2: `original stub text not found (fixed stub has no gap)`. The fixed stub does a single `cat`, so the check-then-read gap no longer exists. |
| V20 | Natural R not reproduced on this host (negative result) | `r_sweep.sh`: verbatim stubs plus a diagnostic lsof stub that logs any empty read. `PROBE_TEST_DRIVER_SLEEP` swept 0.98 to 1.20 in 10 ms steps (18 runs), then 0.990 to 1.100 in 2 ms steps (56 runs). Also `PROBE_TEST_DRIVER_SLEEP=0` for 120 s (51 runs) | 74/74 `control_void`, **0 empty reads**. Sleep 0: 51/51 `empty_peer_set` (the driver was gone before the first sample). **Positive control:** the same diagnostic stub records empty reads under V17/V18 injection. |
| V21 | Phase: the window is ms-wide and only coincides by jitter | `r_phase.sh`: perl `Time::HiRes` stamps in the stubs, sleep 2 | lane 2: `drv-write 2.6812`, `lsof-test 2.6939` (13 ms later), then `3.8839`. `drv-rm 4.7208`, no sample within 0.8 s. The first sample lands about 10 ms after the lane write (that is W1's exposure under VM steal). Samples near the driver's end straddle `rm` only by jitter (W2). |
| V22 | Audit: no other signal is sent inside a fork→exec window | `grep -n kill` on both files | Every other `kill` is a deadline escalation sent ≥1 s after spawn (`run_bounded` 118/129, `cleanup_fixture_sleeps` 59/69, probe `run_lane` 252/261, nc listener 619/637). Line 237 (`kill "$timer"` on a stimulus failure) may also be lost, but its `wait` is bounded by the 1 s window and the result is discarded, so it returns 72 regardless. |
| V23 | Patched suite passes; baseline for comparison | `PROBE_UNDER_TEST=…/motoko_connection_probe.sh /bin/bash suite_fixed.sh` ×6; original ×1 | original: `rc=0 70s PASS: 59`. Fixed: **6/6** `rc=0`, 74–81 s, `PASS: 61` (`ok 32`/`ok 33` new, `ok 59 … timer` with `mode=timer rc=73 refusal_count=1 derived_count=0`) |
| V24 | The patch applies cleanly; bash 3.2 syntax is OK | `git apply --check fix.diff`; `/bin/bash -n suite_fixed.sh` | `apply-check rc=0`; `syntax-ok` |
| V25 | shellcheck delta | `shellcheck -s bash` on both files, diff the codes | one new **info-level** `SC2016` on the `/bin/bash -c '…$1…'` launcher (intentional single quotes). shellcheck does not gate this file; make only runs `bash -n` (`make/test.mk:148`). |
| V26 | Census and refusal-branch gates stay green | arm 60/61 output of V23 | `ok 60 - refusal-branch count … (28)`, `ok 61 - wall-clock literal census …` (no literal `PROBE_TIMEOUT_SECS=N` added; `bound_secs` count +1, which only has a minimum) |

Not re-verified by the planner: the controller's totals (5 T and 3 R occurrences across ~819 runs) and the claim of no failures in the last 300 runs. I verified three individual logs (V1–V3).

## 2. Mechanisms and reproduction

### T: lost timer SIGTERM

The causal chain, all line-anchored in V4–V8:
1. Suite line 7 sets `trap … EXIT`. bash 3.2 (`trap.c:480`) then calls `initialize_terminating_signals()`, which installs `termsig_sighandler` for SIGTERM.
2. `run_bounded` runs `measure_fork_rate` as an async function (`"$@" &`). In that child `reset_terminating_signals()` is commented out (`execute_cmd.c:3014`), so the handler survives into the subshell.
3. `sleep "$window" &` forks the timer child, which inherits the handler and keeps it until `execve` (the kernel resets caught signals to default only at exec).
4. Line 230 sends SIGTERM immediately. If it is delivered before `execve`, the handler only sets `terminating_signal` (`sig.c:456`). No `QUIT` runs before exec. Exec then drops the handler and the flag together, `sleep 1` runs to completion, `wait` returns 0, and the measurement goes on to derive bounds. That is the CI signature `rc=0 refusal=0 derived=1`.

| Instrument | Before (TERM) | After (KILL) |
|---|---|---|
| suite functions, repeated, normal QoS (V9/V10) | 39/3631 = 1.07% | 0/8415 |
| suite functions, repeated, `taskpolicy -b` | 72/705 = 10.2% | 0/6749 |
| suite functions, fresh shell per trial (V12) | 2/440 = 0.45% | 0/433 |
| no EXIT trap, TERM (V11), `taskpolicy -b` | 0/6736 | n/a |
| real full-suite arm (V13) | 0/854, **not reproduced locally** | n/a |
| guard (TERM ignored on entry), arm recursion ×20 | 20/20 `rc=0 derived=1` (mutant) | 20/20 `rc=73 refusal=1` |
| CI | 5/~819 ≈ 0.6% | (expected 0) |

Fix: inject the timer fault with SIGKILL, which can be neither caught nor ignored. The arm still proves exactly what it claims: a timer that did not exit 0 (here 137) makes the measurement refuse with rc 73 and "measurement timer exited". The deterministic guard is that the timer arm's recursion starts with SIGTERM ignored. That disposition is inherited across fork and exec, so any SIGTERM-based injection now leaves `sleep` running in every run. Bash's rule that "signals ignored on entry cannot be trapped" also keeps the suite's EXIT trap from overriding it.

### R: invented OpenRouter peer

The stub maps "not `treatment`" to `203.0.113.8:443`, and an empty read is "not `treatment`". An empty read happens in one of two windows:
- **W1**: the driver's `echo "$lane" > file` has created the file but not yet written it. The first sample lands about 10 ms after the lane write locally (V21), and it can fall inside that gap when the driver is descheduled.
- **W2**: the driver's closing `rm -f` lands between the stub's `[[ -f ]]` (line 487) and `cat` (line 488). That gap is one fork+exec of `cat`.

If the race hits lane 1 (treatment), the treatment peer set gets the OR peer, and you see V3. If it hits lane 2 (the "control" lane, which runs as treatment in the refusal arm), that lane looks like a valid control, so the probe exits 0 and you see V2. Injection reproduces both signatures from both windows, deterministically (V17, V18). The CI logs cannot tell W1 from W2, so the fix closes both. Natural reproduction failed on this host: 0/125 runs (V20). The argument for the fix therefore rests on the code plus deterministic injection, and the fix is risk-free for that reason: the fail-closed stub turns an unreadable lane into "no sample", which is exactly what the stub already did for a file that does not exist yet (line 487). Every lane still gets a mid-life sample from its 2 s driver. The success and refusal arms passed 6/6 (V23).

| Instrument | Before | After |
|---|---|---|
| W1 inject, lane 1 / lane 2 (refusal) | OR leak / unexpected success | `treatment verdict is void` both |
| W2 inject | OR leak / unexpected success | window structurally gone (single `cat`) |
| guard arm (forced W1, both lanes) | red under mutation M-R1 (exact V3 line) | 6/6 green, acks = 2 |
| natural, this host | 0/125 (not reproducible here) | n/a |
| CI | 3/~819 since 10-08 | (expected 0) |

## 3. The exact diff (one file: `tools/eval/test_motoko_connection_probe.sh`)

Apply with `git apply` from the worktree root. This was verified with `git apply --check` against `22a72e7c5`. It is bash 3.2 compatible: no arrays added, no `declare -A`, no `${v,,}`, no `mapfile`.

```diff
--- a/tools/eval/test_motoko_connection_probe.sh
+++ b/tools/eval/test_motoko_connection_probe.sh
@@ -227,7 +227,12 @@
   }
   sleep "$window" & timer=$!
   if [[ "${PROBE_SELFTEST_MEASUREMENT_FAILURE:-}" == timer ]]; then
-    kill -TERM "$timer" 2>/dev/null || true
+    # SIGKILL, not SIGTERM. The EXIT trap at the top of this file makes /bin/bash 3.2 install its
+    # deferred terminating-signal handler, and the timer child keeps that handler from fork until
+    # execve. A TERM that lands in that window only sets a flag that nothing reads before exec, so
+    # `sleep` runs the whole window and exits 0 (CI: rc=0 derived=1, about 1 run in 160). SIGKILL
+    # cannot be caught or ignored. The timer arm below starts with TERM ignored to pin this.
+    kill -KILL "$timer" 2>/dev/null || true
   fi
   while kill -0 "$timer" 2>/dev/null; do
     if "$stim" >/dev/null 2>&1; then
@@ -484,14 +489,20 @@
   shift
 done
 lane_file="${PROBE_STUB_STATE}.${pid}"
-[[ -f "$lane_file" ]] || exit 1
-lane=$(cat "$lane_file")
+# FAIL CLOSED. The driver creates its lane file before writing the lane and deletes it as it
+# exits, so a sample can read nothing at either end of the driver's life. The old stub sent
+# every lane that was not "treatment" to the OpenRouter peer, so that read invented an
+# OpenRouter connection: on CI the refusal arm passed when it should have refused, and
+# 203.0.113.8:443 leaked into a treatment lane. A lane it cannot name now samples no peer.
+lane=$(cat "$lane_file" 2>/dev/null) || exit 1
+[[ -n "$lane" || -z "${PROBE_TEST_EMPTY_LANE_ACK:-}" ]] || : > "$PROBE_TEST_EMPTY_LANE_ACK.$pid"
+case "$lane" in
+  treatment) peer="127.0.0.1:51000->127.0.0.1:11434" ;;
+  control) peer="192.0.2.1:51000->203.0.113.8:443" ;;
+  *) exit 1 ;;
+esac
 echo "COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME"
-if [[ "$lane" == treatment ]]; then
-  echo "stub $pid user 1u IPv4 0 0t0 TCP 127.0.0.1:51000->127.0.0.1:11434 (ESTABLISHED)"
-else
-  echo "stub $pid user 1u IPv4 0 0t0 TCP 192.0.2.1:51000->203.0.113.8:443 (ESTABLISHED)"
-fi
+echo "stub $pid user 1u IPv4 0 0t0 TCP $peer (ESTABLISHED)"
 EOF
 cat > "$live_bin/ailang-stub" <<'EOF'
 #!/bin/bash
@@ -502,6 +513,12 @@
   shift
 done
 [[ -z "${PROBE_TEST_MARKER:-}" ]] || printf 'ailang-stub lane=%s args=%s\n' "$lane" "$args" >> "$PROBE_TEST_MARKER"
+# Hold the lane file created-but-empty (the window above) until lsof has sampled it.
+# Bounded by the probe's lane deadline, which TERMs this untrapped stub.
+if [[ -n "${PROBE_TEST_EMPTY_LANE_ACK:-}" ]]; then
+  : > "${PROBE_STUB_STATE}.$$"
+  while [[ ! -e "$PROBE_TEST_EMPTY_LANE_ACK.$$" ]]; do sleep 0.05; done
+fi
 echo "$lane" > "${PROBE_STUB_STATE}.$$"
 echo "stub driver lane=$lane"
 # Shared by the bounded-termination arm and the tail SIGKILL-escalation fixture arm.
@@ -603,6 +620,25 @@
 # the retained log exists for.
 pass_arm "refusal path retains both lanes driver logs and lsof captures"
 
+# Pins the fail-closed lsof stub with a forced read rather than a 3-in-819 base rate: each driver
+# holds its lane file empty until lsof has read it empty, so both lanes are certain to take one
+# sample in the window. A stub that maps an unreadable lane to a peer puts 203.0.113.8:443 in the
+# treatment lane and the probe fails. The ack count proves the window was actually reached.
+empty_read_ack="$tmp_dir/empty-read-ack"
+expect_success "a lane file read before the driver writes it samples no peer" \
+  env PATH="$live_bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS="$(bound_secs 4)" \
+    PROBE_STUB_STATE="$tmp_dir/lane-empty-read" PROBE_TEST_EMPTY_LANE_ACK="$empty_read_ack" \
+    /bin/bash "$probe" treatment control "$tmp_dir/empty-read.json"
+empty_read_acks=0
+for ack in "$empty_read_ack".*; do
+  [[ -e "$ack" ]] && empty_read_acks=$((empty_read_acks + 1))
+done
+if (( empty_read_acks != 2 )); then
+  echo "not ok - empty lane-file window reached in $empty_read_acks lanes, expected 2" >&2
+  exit 1
+fi
+pass_arm "the empty lane-file window was sampled in both lanes"
+
 unwritable="$tmp_dir/not-a-directory"
 : > "$unwritable"
 expect_failure "JSON artifact write failure refuses" "could not write JSON artifact" \
@@ -1106,7 +1142,12 @@
 pass_arm "bound drift classifier is loud only when the end rate needs a higher scale"
 
 assert_measurement_failure() {
-  local name=$1 mode=$2 expected_rc=$3 expected_text=$4 rc refusal_count derived_count
+  local name=$1 mode=$2 expected_rc=$3 expected_text=$4 rc refusal_count derived_count term=default
+  # timer mode runs the recursion with SIGTERM IGNORED, which every child inherits through fork
+  # and exec. That turns the rare CI loss described in measure_fork_rate into a certainty: an
+  # injection that relies on SIGTERM leaves `sleep` running and this arm reds every time with
+  # the CI signature (rc=0 derived=1); one that cannot be caught or ignored refuses with 73.
+  [[ "$mode" == timer ]] && term=ignored
   # The forced-rate overrides MUST be cleared for these arms. They are the one
   # PROBE_SELFTEST_* pair that SHORT-CIRCUITS the measurement rather than steering it, so an
   # ambient value makes measure_rate_or_refuse unreachable and the injected fault is never
@@ -1117,6 +1158,7 @@
   # these overrides are LEGITIMATE at suite scope for every other arm; only this recursion
   # must not inherit them.
   run_bounded "$tmp_dir/measurement-$mode.out" "$tmp_dir/measurement-$mode.err" "$ARM_CAP_SECS" -- \
+    /bin/bash -c '[[ "$1" == ignored ]] && trap "" TERM; shift; exec "$@"' term-disposition "$term" \
     env -u PROBE_SELFTEST_FORK_RATE -u PROBE_SELFTEST_REAL_OP_RATE \
       PROBE_UNDER_TEST="$probe" PROBE_SELFTEST_MEASUREMENT_FAILURE="$mode" \
       PROBE_SELFTEST_DERIVATION_ONLY=1 /bin/bash "$0"
```

Optional and not tested by the planner: `# shellcheck disable=SC2016` on the line above `run_bounded` inside `assert_measurement_failure`, to silence the info-level V25 note. It is a comment only.

What does NOT change: the probe; what every existing arm asserts; `expected_refusal_branches` (28, probe-side); the timeout-literal census (still 5); and lane semantics. Only `treatment` and `control` are ever passed as lanes (`grep '"$probe" treatment'`: lines 536, 545, 547, 565, 584, 598, 610, 701, 802, 962, 966, 971), and they map to the same peers as before.

### Changelog fragment: `changelogs/unreleased/2026-11-17-motoko-probe-flakes.md`

```markdown
### Fixed — motoko connection probe self-test: two intermittent CI failures (2026-11-17)

`tools/eval/test_motoko_connection_probe.sh` failed about 1 run in 100 on the macOS
"launchd drivers (bash 3.2)" job, with two signatures. Both were defects in the test, not the probe.

- **Timer arm** (`rc=0 expected_rc=73 … derived=1`): the suite's EXIT trap makes `/bin/bash`
  3.2 install a deferred SIGTERM handler that the timer child keeps until `execve`. A
  SIGTERM sent right after the fork was recorded and then discarded at exec, so the timer ran
  to completion. The fault is now injected with SIGKILL, and the arm runs with SIGTERM ignored,
  so reverting to SIGTERM fails every time instead of once in a while.
- **Refusing live path** (`unexpectedly succeeded`, or `treatment peer set contains OpenRouter
  endpoint: 203.0.113.8:443`): the `lsof` stub treated an empty lane-file read, which happens
  while the driver is creating or deleting the file, as the OpenRouter lane. It now reports no
  peer for a lane it cannot name, and a new arm forces that read in both lanes.

Evidence: design_docs/fleet-mission-evidence/iteration117/plan.md.
```

## 4. Test and mutation plan (all run by the planner; each red is from a full-suite run of the mutant)

| ID | Mutation (applied to the fixed suite) | Must go red | Result |
|----|----------------------------------------|-------------|--------|
| M-T1 | line 235 `kill -KILL` → `kill -TERM` (reverts the T fix) | arm "bound measurement refuses an unsuccessfully terminated timer" | **RED** at arm 59: `not ok - … (rc=0 expected_rc=73 refusal=0 derived=1)`, exactly the CI signature. The arm's recursion run directly ×20: **20/20** `rc=0 derived=1`. |
| M-T1′ | M-T1 with the launcher's `term=default` (i.e. the old, unguarded arm) | (shows why the guard is needed) | 20/20 `rc=73` **green**. A reverted fix would be invisible without the TERM-ignored launcher. |
| M-R1 | lsof stub `*) exit 1 ;;` → `*) peer="192.0.2.1:51000->203.0.113.8:443" ;;` (reverts the R mapping only) | new arm "a lane file read before the driver writes it samples no peer" | **RED** at arm 32: `not ok - … failed` + `INSTRUMENT FAILURE: treatment peer set contains OpenRouter endpoint: 203.0.113.8:443` (the CI line) |
| M-R2 | whole lsof stub reverted to the original text (no fail-closed, no ack) | same arm | **RED** at arm 32: `INSTRUMENT FAILURE: lane treatment exceeded 4s sampling deadline` (the driver waits for an ack that never comes) |
| M-R3 | driver hold neutered (`if false; then`), so the guard never reaches the window | the vacuity check | **RED** after `ok 32`: `not ok - empty lane-file window reached in 0 lanes, expected 2` |
| (fixed) | none | none | 6/6 full-suite runs `rc=0`, `PASS: 61` |

These are the deterministic regression guards. T is pinned by the TERM-ignored launcher (M-T1 is red 100%, not at the 0.6% base rate). R is pinned by the `PROBE_TEST_EMPTY_LANE_ACK` handshake, which forces an empty read in both lanes regardless of host speed and is bounded by the lane deadline `bound_secs 4`. Anti-vacuity is the ack count (M-R3).

Executor re-run checklist. Mutate via a scratchpad copy, never `git checkout <file>`:
1. `cp tools/eval/test_motoko_connection_probe.sh <scratch>/m.sh`; apply the M-T1 sed (`s/kill -KILL "\$timer"/kill -TERM "$timer"/`); `PROBE_UNDER_TEST=$PWD/tools/eval/motoko_connection_probe.sh /bin/bash <scratch>/m.sh`. Expect rc 1 and the line above.
2. Same for M-R1 (`s/^  \*) exit 1 ;;$/  *) peer="192.0.2.1:51000->203.0.113.8:443" ;;/`) and M-R3.

## 5. Acceptance commands

Run them sequentially; never two suite copies at once, because the suite measures the host fork rate. Use `/bin/bash`.

```bash
cd /Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter117-probe-flakes
/bin/bash -n tools/eval/test_motoko_connection_probe.sh
for i in 1 2 3 4 5; do s=$(date +%s); /bin/bash tools/eval/test_motoko_connection_probe.sh >/dev/null 2>&1; echo "run$i rc=$? $(( $(date +%s)-s ))s"; done
# expect rc=0 x5, about 75-80 s each on this host (baseline 70 s; the new arm adds two 2 s lanes); last line "PASS: 61 probe self-test arms ran"
/bin/bash tools/eval/test_motoko_connection_probe.sh 2>&1 | grep -E 'mode=timer|ok 3[23] |PASS:'
# expect: "mode=timer rc=73 refusal_count=1 derived_count=0", ok 32/33 (new arms), "PASS: 61"
make test-launchd-drivers        # executor and controller; the planner did not run it (full launchd bundle)
make check-changelog             # fragment shape
```

A refusal reading "instrument failure, not a verdict" under heavy host load is not a verdict. Re-run on a quiet host.

## Contradictions and amendments to the controller's brief

1. **R has two windows, not one.** The lane file is new per pid, so W1 is create-then-write rather than truncate-then-write, and the `rm -f` race (W2) gives identical CI output. The fix covers both.
2. **T: the mechanism is the suite's own EXIT trap.** It is not a generic "bash resets dispositions before exec" issue. bash 3.2 installs a flag-only SIGTERM handler for non-interactive shells that have an EXIT trap, and async function subshells never reset it. Removing the trap removes the loss entirely (0/6736, V11).
3. **The full-suite T arm did not reproduce on this host** (0/854). The suite's own functions do (V9, V12), at rates that bracket CI's 0.6%.
4. **CI host speed varies about 3x.** The 11-02 T failure ran at `r=159/s` (scale 3). The October R failures ran at `r=458–483/s` (scale 1). R is therefore not exclusively a slow-host failure.
5. The base-rate totals and the claim of no failures in the last 300 runs were not re-verified (see the note under the log).
