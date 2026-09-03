# M-MOTOKO-GROUP-KILL-AND-LSOF-CONTAINMENT: give the SIGKILL-escalation group form its own killer, and make the `REAL_LSOF` hijack fail loudly

**Status**: Planned (motoko mission iteration 34, queue row 6o)
**Target**: v0.34.x (next motoko iteration)
**Priority**: P1 (loop-health — one production kill site has zero discriminating killers; one measured PATH hole is held by a comment)
**Estimated**: 1 iteration (~2-3 hours implementation + mutation validation; every load-bearing mechanism is already prototyped and measured below)
**Dependencies**: Row 6o of the motoko mission queue. Both touched files are self-contained shell; the suite is hermetic. Direct successor of row 6i (`design_docs/implemented/v0_35_0/m-motoko-connection-probe-run-lane-harness.md`, the arm this doc extends) and row 6r (`design_docs/planned/m-motoko-stub-refusal-arm.md`, whose placement measurement this doc obeys). Base: `origin/dev` at `55891002f`, worktree `sprint/motoko-iter34-group-kill-lsof`, clean.

**Provenance convention.** Every number in this document was re-derived in this session on this worktree at `55891002f` (darwin/arm64, GNU bash 3.2.57, `/usr/sbin` on PATH). Nothing is transcribed from the queue row or from the controller's briefing; where the row's numbers are stale (it says 41 arms; HEAD has 43) the doc says so. Claims that were NOT measured are labelled **UNMEASURED** inline and collected in one section at the end. **Revision 1** (after quorum round 1, 2026-09-03) re-derived every number it added (V34-V41) on the same worktree at the same SHA; the earlier `/tmp` prototype copies had not survived, so nothing that needed the prototype was re-run, and those items stay UNMEASURED and say so.

---

## Problem Statement

`tools/eval/motoko_connection_probe.sh` (the production probe) bounds each eval lane in `run_lane()` with a two-stage process-group kill:

| probe line | stage | code |
|---|---|---|
| 252 | TERM, group form | `kill -TERM "-$pid" 2>/dev/null \|\| true` |
| 257 | 5-second grace window | `terminate_deadline=$(( $(date +%s) + 5 ))` |
| 261 | KILL escalation, group form | `kill -9 "-$pid" 2>/dev/null \|\| true` |
| 264 | KILL escalation, single-PID fallback (degraded branch) | `kill -9 "$pid" 2>/dev/null \|\| true` |
| 267 | refusal after the escalation | `instrument_failure "lane $lane exceeded its bounded termination deadline"` |
| 272 | refusal when TERM sufficed | `instrument_failure "lane $lane exceeded ${timeout_secs}s sampling deadline"` |

`tools/eval/test_motoko_connection_probe.sh` (the self-test suite, 43 arms at HEAD) pins this with arm 36, `production run_lane timeout kills wrapper grandchild` (test:536-682), landed by row 6i. Its acceptance mutant changes **both** group sites at once.

### (a) Only the TERM half of the group kill is pinned

Mutating ONLY the group `-9` at probe:261 to the single-PID form leaves the suite **rc=0, 43 ok, 0 not ok, survivors=0** (V6). The row's diagnosis holds and was re-derived: the run_lane fixture's grandchild is a plain `sleep` (stub:301) with default signal dispositions, so it dies at the TERM stage; the probe shell reaps the dead wrapper, `kill -0 "$pid"` at probe:258 fails, the grace loop exits without ever reaching probe:261, and the lane refuses at probe:272 (`timeout=yes` in V6's evidence line is exactly the probe:272 message). The escalation stage is **dead for discrimination**: removing it, single-PID-ing it, or replacing it with `true` all land green. 6i's both-sites mutant reds on the TERM half alone (V7), so 6i's bar is honestly met — this is the gap that bar could not see.

### (b) `REAL_LSOF` containment is narrower than the code claimed

The suite's survivor oracle (`fixture_sleep_pids`, test:39-45) runs `"$REAL_LSOF" -a -c sleep -d cwd`, and `REAL_LSOF` is resolved at test:16 with `command -p -v lsof`, then only required to be an absolute executable path (test:18-21). On this shell `command -p` does **not** confine the lookup to `getconf PATH`: an `lsof` in a directory placed ahead of the standard path in the AMBIENT environment resolves as `REAL_LSOF` (V14: arm → the hijack; control, clean PATH → `/usr/sbin/lsof`; negative control, leading dir without an `lsof` → `/usr/sbin/lsof`). The suite's own stub PATH (`live_bin`) cannot reach the oracle because `REAL_LSOF` is resolved before `live_bin` exists — that defence holds — but an ambient hijack would make every survivor verdict (arms 35, 36 and the new arm) whatever the hijack prints. Commit `1caf02e44` narrowed the comment to what is defended and pointed at this row; nothing executable changed.

**Why both belong in one doc:** both are properties of the same survivor-oracle arm family in one file, and (b) is a precondition for trusting the survivor counts that (a) adds a new reader of.

---

## Goals

**Primary (a):** a self-test arm for which the single-PID mutant of probe:261 ALONE is red, by name, on a survivor count — while 6i's arm 36 keeps reddening on the TERM half (both-sites mutant and TERM-only mutant).

**Primary (b):** on Darwin, refuse at suite startup — before any arm — when the directory of the resolved `REAL_LSOF` is not one of the colon-separated entries of `getconf PATH`; and pin that refusal with an arm that reds if the gate is removed.

**Secondary:** no production-probe change; `expected_refusal_branches` stays 28; the suite's own wall-clock-bounded arms start at the same offset as at base (new forking arms go behind them); total suite time grows by roughly 11s (V23).

**Success metrics (all measured on the prototype, V23-V26):** suite 43 → 46 arms, rc=0 3/3 runs; `-9`-group-only mutant → rc=1 with the new kill arm the sole `not ok`; gate-removed suite → rc=1 with the hostile-PATH arm the sole `not ok`; 6i's mutant → still arm 36.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| D1 — (a) mechanism: TERM-immune fixture tree via the EXISTING `PROBE_TEST_IGNORE_TERM` knob, not a shortened grace window | Decides whether the production probe is touched at all, and whether the arm exercises the real 5s grace path | agent (measured) | design | low |
| D2 — (b) shape: ASSERT containment and hard-fail, rather than re-resolve `lsof` from `getconf PATH` ourselves | Fail-loud vs silent-fallback; the row scopes an assertion | agent | design | low |
| D3 — (b) pin: re-execute the suite (`$0`) under a hostile PATH with a containment-only marker, not a unit call of an extracted function | Only the re-exec proves the WIRING (that startup really calls the gate on the real resolution) | agent (measured) | design | low |
| D4 — placement: every new forking arm goes BEHIND arm 42, the last wall-clock-bounded arm | Iteration 33 measured 4 reds in 19 with a fork placed ahead of the bounded arms vs 0 in 5 behind | agent (measured by iteration 33; re-measured 3/3 here) | design | low |
| D5 — refactor: the 6i fixture block becomes one parameterised function `run_lane_fixture_arm`, called twice, instead of a second copy | Duplicating 145 lines would give the next edit two places to drift | agent | implementation | med |

### D1 — why the TERM-immune tree, and why not a shortened grace window

The row offers two mechanisms. The grace window is a hardcoded `+ 5` at probe:257 with **no knob** (V13: `PROBE_GRACE|PROBE_TERM_GRACE|GRACE_SECS` count 0, control `PROBE_TIMEOUT_SECS` count 3). Shortening it would mean (i) adding a production env var and its `instrument_failure` validation — moving `expected_refusal_branches` 28 → 29 and touching the probe — and (ii) it would not by itself create discrimination: with a plain-`sleep` grandchild the TERM stage still kills everything and the escalation is still never reached, whatever the grace length. Shortening only buys wall time.

The TERM-immune tree costs nothing in production code. The stub already carries `if [[ "${PROBE_TEST_IGNORE_TERM:-0}" == 1 ]]; then trap '' TERM; fi` at stub:293, **before** the grandchild path spawns `sleep … &` at stub:301 (V11), and an ignored disposition is inherited across fork and exec — measured directly: a `sleep` backgrounded by a bash that ran `trap '' TERM` is still alive after a direct `kill -TERM`, while the control without the trap is dead (V9). So one existing knob makes the whole fixture tree TERM-immune: the production TERM at probe:252 kills nothing, the wrapper stays alive so `kill -0 "$pid"` keeps succeeding through the grace loop, the 5s window expires, and only `kill -9 "-$pid"` at probe:261 clears the grandchild. Under the single-PID mutant the `-9` reaches the wrapper alone; the TERM-immune `sleep` is orphaned in the fixture cwd and the oracle counts it (V24: `survivors=1`).

Cost accepted: the arm spends the real 5s grace window (~9s total; V33), which is the price of exercising the real escalation path rather than a copy of it.

### D2 — assert, do not re-resolve

Iterating `getconf PATH` ourselves and taking the first executable `lsof` would be immune to the hijack and would need no arm. Rejected on the project's fail-loud rule: an `lsof` shadowing the standard path in the ambient environment of the CI shell is a symptom (a misconfigured or compromised runner), and silently routing around it hides the symptom from the one instrument that could report it. The row scopes an assertion; the assertion is what this doc specifies.

### D3 — re-exec, with a marker that doubles as the recursion guard

An arm that re-executes `$0` already exists (test:721, `env PROBE_SELFTEST_ARM_CAP_SECS=invalid /bin/bash "$0"`, V18), so the shape has precedent. The inner run must never reach its own arms: a marker env var (`PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY=1`) makes the suite `exit 0` immediately AFTER the gate, so a clean resolution reads as `unexpectedly succeeded` in the outer `expect_failure` and a hijack reads as the refusal. The marker exit sits outside the gate's `if` block on purpose: a mutant that deletes the gate still hits the marker exit (measured, V25) rather than running 43 arms one level deep. A second guard immediately before the arm refuses if the marker ever leaks past the top-of-file exit. **Revision 1 (quorum round 1, option A):** both inner invocations are additionally given an ABSENT probe on their own env line — `PROBE_UNDER_TEST="$tmp_dir/no-probe"`, a path nothing creates — so an inner run that ever passes the marker line reds at its arm 1 (`"$probe" --classify-fixture`, test:201) in ~20 ms instead of running 43 arms (V37). Nothing above the marker line reads `$probe` (V3, V19, V37), so the scoping is invisible on the green path. Independently of both guards, every re-exec arm is wall-clock bounded by `run_bounded` (V35) — see "Wall-clock bounding of the re-exec arms" below.

---

## Quorum trigger check

This is an unattended mission-loop doc, so the controller runs the quorum regardless. Of the four attended triggers: (1) design-freeze items — **none**, every decision above is agent-resolvable and measured; (2) overrides shared machinery — **no**, it reuses `run_bounded`, the existing stub knob and the existing `$0` re-exec shape; (3) cost/KPI/banked-schema — **no**; (4) external-system premises — **no**, `command -p` on bash 3.2.57 and `/usr/bin/getconf` are host facts measured on this host (the GitHub `macos-latest` runner's identical behaviour is UNMEASURED and listed as such).

### Quorum round 1 (2026-09-03) — BLOCKED; this is the one protocol revision

Three external reviewers: `gemini-3-1-pro` **PASS** (one non-blocking concern), `oc-glm-5-2` **REJECT**, `gpt5-6-sol` **REJECT**. `gpt5-6-sol` was initially ABSENT on budget; the controller re-ran it alone at a raised cap and its verdict counts. The controller measured both blocking premises first-party in this worktree before forwarding them; this revision applies those measurements and does not re-argue them. No reviewer disputed D1-D5 or the placement, so the direction is unchanged.

| Reviewer | Objection | Disposition | Where applied |
|---|---|---|---|
| `oc-glm-5-2` | AC2's probe sha256 was asserted but never measured in the Verification Log — it could be stale or transcribed | Premise (unmeasured) **UPHELD**; conclusion (may be stale) **REFUTED**: the hash reproduces, with a control that discriminates | V34; AC2 now cites it. Hash unchanged; AC2 not weakened |
| `gpt5-6-sol` | The `$0` re-exec arms are bounded in depth only, not time; a hung inner run cannot be terminated | Headline **REFUTED by construction** — every arm runs under `run_bounded`'s `date +%s` deadline with TERM→KILL escalation, and test:721 is precedent — but the doc described only the recursion guard, so the doc was at fault. The unstated **2.07× margin** on T7 was real and is closed by option A | V35-V39; "Wall-clock bounding of the re-exec arms"; D3, T7, AC8, AC13, Risks, UNMEASURED revised |
| `gemini-3-1-pro` | Un-normalised string equality could lock the suite out if a future macOS aliases a standard directory | Non-blocking; **APPLIED** as an explicit by-choice statement, with the failure mode reproduced and the operator recovery path stated | V40-V41; "Un-normalised BY CHOICE" bullet; Risks; UNMEASURED items 6 and 8 |

---

### Quorum round 2 (2026-09-03) — BLOCKED; resolved under the NARROW-REFINEMENT CARVE-OUT, not force-passed

Verdicts, all three external reviewers PRESENT: `gemini-3-1-pro` **pass** (second consecutive pass), `oc-glm-5-2` **reject**, `gpt5-6-sol` **reject**.
`gpt5-6-sol` was recorded ABSENT (budget) by the synthesis for the **second** round running — the self-selecting degrade this loop's rulebook names, since a reviewer drops on budget precisely when the doc has GROWN. It was re-run alone at a raised cap both times and its verdict counts; the quorum was **3/3 present**, never N−1. So the synthesis line `BLOCKED … gpt5-6-sol ABSENT` understates the round: it was a reject, not a hole.

The one designer revision and the one re-quorum the protocol allows are SPENT. Both surviving objections (a) carry a concrete reviewer-authored `proposed_fix` and (b) dispute no design DIRECTION — one is attribution, one is determinism — so the **narrow-refinement carve-out applies** (ratified for this mission at iteration 29) and the CONTROLLER applied the reviewers' own text VERBATIM. No controller-invented resolution was substituted, and no objection was overridden. The Fable diet is intact: one doc, one authoring run, one revision run.

| Reviewer | Objection | Surface | Controller's measurement | Disposition |
|---|---|---|---|---|
| `oc-glm-5-2` | test:539 / :681 / :740-741 / :743-752 are asserted in the Solution Design and Conflict Surface but no V-row outputs them; the `/tmp` prototype that relied on them no longer exists | attribution / provenance | Ran the reviewer's own three `sed -n` commands. **All four boundary claims are CORRECT; there is no drift to correct.** The objection is procedurally right and substantively refuted — the numbers were unverified, not wrong. | **APPLIED verbatim** as V42/V43/V44. Nothing else changed. |
| `gpt5-6-sol` | the gate runs `/usr/bin/getconf PATH` outside `run_bounded` with no deadline, so a hung `getconf` hangs the suite before arm 1; handling empty/non-zero does not handle non-termination | determinism / bounded-waits | **UPHELD, and the mechanism is confirmed:** `run_bounded` is defined at test:88 while the gate was specified at test:28, so it could not have called it (V45). The class is PRE-EXISTING — four unbounded external calls already run before arm 1 (V46) — but this design was adding a fifth. | **APPLIED verbatim**: gate relocated below test:88, run through `run_bounded` with an explicit `PROBE_GETCONF_CAP_SECS` cap, with distinct refusals for timeout / non-zero / empty, plus the reviewer's requested V-row (V45/V46) and mutation drill (T12). Bounding the four pre-existing calls is filed as a QUEUE ROW, not absorbed. |

Round-1 objections, for the record: `oc-glm-5-2`'s sha256 provenance (measured CORRECT, applied as V34) and `gpt5-6-sol`'s re-exec bounding (headline claim REFUTED by construction — the arms inherit `run_bounded`/`ARM_CAP_SECS` — with the real residual, a 2.07× margin, closed by option A at ~5,000×).
**Objection surfaces per round** (tracked per this loop's SPLIT rule): R1 = provenance + harness semantics; R2 = provenance + bounded-waits. Objections are SPREAD across surfaces and shrinking in scope round over round, and one reviewer has passed in both rounds — so this is a maturing doc, **not** a decomposition signal.

## Verification Log

Every row is a claim about the codebase's current state, the command that produced it, and its observed output. All rows were run in this session on the unmodified worktree at `55891002f` unless the row says it ran against a `/tmp` copy. Suite runs were executed strictly one at a time (the suite is load-sensitive; V23 records the load average).

| # | Claim | Command | Observed output |
|---|---|---|---|
| V1 | Worktree is the sprint branch at base, clean. | `git status --short \| head; git rev-parse --abbrev-ref HEAD; git rev-parse --short HEAD` | (no status lines); `sprint/motoko-iter34-group-kill-lsof`; `55891002f` |
| V2 | The kill sites are at probe:252 (group TERM), 261 (group `-9`), 264 (single-PID `-9`); the grace deadline at 257; the `kill -0` polls at 248/258. Known-positive control in the same sweep. | `grep -n 'kill -9\\\|kill -TERM\\\|kill -0' tools/eval/motoko_connection_probe.sh; grep -c PROBE_MAX_TREE_NODES tools/eval/motoko_connection_probe.sh` | 241 `kill -0 "-$pid"`; 248 `while kill -0 "$pid"`; 252 `kill -TERM "-$pid"`; 258 `while kill -0 "$pid"`; 261 `kill -9 "-$pid"`; 264 `kill -9 "$pid"`; control = 2 |
| V3 | `REAL_LSOF` appears at test:16, 18, 19, 20, 22, 30, 36, 42, 43, 664, 672 — resolution, case guard, Darwin requirement, comment, the oracle, and the evidence line. | `grep -n REAL_LSOF tools/eval/test_motoko_connection_probe.sh` | exactly those 11 line numbers; line 16 is `REAL_LSOF=$(command -p -v lsof 2>/dev/null \|\| true)` |
| V4 | Baseline suite at HEAD: rc=0, **43** ok (the row's "41" is stale), 0 not ok, 49s. | `/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/iter34-base.out 2> /tmp/iter34-base.err; echo rc=$?; grep -c '^ok ' …; grep -c 'not ok' …; tail -1 …` | `rc=0 elapsed=49s ok=43 notok=0`; `PASS: 43 probe self-test arms ran`; arm 36 = `ok 36 - production run_lane timeout kills wrapper grandchild`; evidence line `… survivors=0 cleanup=0 probe_rc=1 markers=yes real_lsof=/usr/sbin/lsof` |
| V5 | Both shell files parse at base; both are mode 755. | `bash -n` on each; `ls -l tools/eval/motoko_connection_probe.sh tools/eval/test_motoko_connection_probe.sh` | rc=0, rc=0; `-rwxr-xr-x` ×2 (313 and 795 lines) |
| V6 | **The defect (a), re-derived:** the group-`-9`-only mutant SURVIVES. Mutant hygiene: `cp -p` copy (mode preserved), `sed -i '' '261s/…/…/'`, `bash -n` rc=0, group `-9` count 1→0 while group TERM count stays 1, run via `PROBE_UNDER_TEST` so the tree is untouched. | `PROBE_UNDER_TEST=$m/probe-9only.sh /bin/bash tools/eval/test_motoko_connection_probe.sh` | diff = line 261 only; `M1: rc=0 elapsed=48s ok=43 notok=0`; evidence `timeout=yes … survivors=0 cleanup=0 probe_rc=1` (`timeout=yes` = the probe:272 sampling-deadline message, i.e. the escalation at 261 was never reached). Mutant sha256 `19bccf73…7f85ad` |
| V7 | 6i's both-sites mutant still reds on arm 36 with a survivor (the TERM half). | same shape, `sed -i '' -e '252s/…/…/' -e '261s/…/…/'`; both group counts → 0 | `M2: rc=1 elapsed=49s ok=35 notok=1`; `not ok - production run_lane timeout kills wrapper grandchild (outer_rc=0 survivors=1 cleanup=0 probe_rc=1)`. Mutant sha256 `143fcde3…33db2` |
| V8 | The TERM-group-only mutant (probe:252 alone) also reds on arm 36 — so arm 36 is the TERM half's killer and the new arm need not be. Run against the prototype suite (V23) so both arms were present. | `PROBE_UNDER_TEST=$m/probe-termonly.sh /bin/bash $m/suite-proto.sh` | `P5: rc=1 elapsed=45s ok=35 notok=1`; same `not ok` line as V7 (arm 36, `survivors=1`). Mutant sha256 `70f119ad…8840`; group `-9` count stayed 1, group TERM 1→0 |
| V9 | An ignored TERM disposition set by `trap '' TERM` in a bash is inherited by a `sleep` that bash backgrounds (fork + exec). Control: without the trap the sleep dies. | `/bin/bash -c 'trap "" TERM; sleep 300 & echo $! > f; wait' &` then `kill -TERM $(cat f); sleep 0.5; kill -0 …` / same without the trap | `sleep 4088 ALIVE after TERM (SIG_IGN inherited)` (`ps`: `4088 4084 SN sleep`); control: `sleep 4118 DEAD after TERM (control: default disposition)` |
| V10 | A child that has died is reaped by its bash parent without an explicit `wait`, so `kill -0` on it fails — which is why a wrapper that dies at TERM ends the probe:258 loop before 261. | `/bin/bash -c 'sleep 300 & c=$!; sleep 0.3; kill -9 $c; sleep 2; kill -0 $c …'` | `kill: (4240) - No such process`; `kill -0 on zombie 4240: rc=1`; `ps` shows no such pid. (Corroborated by V6's `timeout=yes`, the probe:272 path.) |
| V11 | `PROBE_TEST_IGNORE_TERM` is set at stub:293, BEFORE the grandchild path (stub:294-313) spawns `sleep … &` at stub:301; exactly one existing arm uses it (test:363). | `grep -n PROBE_TEST_IGNORE_TERM tools/eval/test_motoko_connection_probe.sh` | 293 `if [[ "${PROBE_TEST_IGNORE_TERM:-0}" == 1 ]]; then trap '' TERM; fi`; 363 `run_live PROBE_TEST_DRIVER_SLEEP=20 PROBE_TEST_IGNORE_TERM=1 PROBE_TIMEOUT_SECS=1 PROBE_TREE_DISCOVERY_SECS=30` |
| V12 | The existing bounded-termination arm (arm 27, test:362-363) asserts the MESSAGE only via `expect_failure`; it has no survivor oracle, which is why it cannot discriminate the group `-9` from the single-PID `-9` (V6). | `sed -n '362,363p' tools/eval/test_motoko_connection_probe.sh` | `expect_failure "bounded termination deadline refuses" "bounded termination deadline" \` / `run_live PROBE_TEST_DRIVER_SLEEP=20 PROBE_TEST_IGNORE_TERM=1 …` |
| V13 | The grace window is hardcoded; there is no env knob for it (negative-existence, with control). | `grep -n 'terminate_deadline=' …probe.sh; grep -c 'PROBE_GRACE\\\|PROBE_TERM_GRACE\\\|GRACE_SECS' …probe.sh; grep -c PROBE_TIMEOUT_SECS …probe.sh` | 257 `terminate_deadline=$(( $(date +%s) + 5 ))`; knob count **0**; control **3** |
| V14 | **The defect (b), re-derived under `/bin/bash`:** `command -p -v lsof` follows the ambient PATH. Arm, control, negative control, and the non-`env -i` shape the suite really runs under. | `/bin/bash -u -c '…'` with `env -i PATH="$(/usr/bin/getconf PATH)"`, `env -i PATH="$h:…"` (hostile dir holding an executable `lsof`), `env -i PATH="$n:…"` (empty dir), and `env PATH="$h:$PATH"` | `bash=3.2.57(1)-release Darwin arm64`; clean control → `/usr/sbin/lsof`; **hostile with lsof → `/tmp/hostile.COfTMy/lsof`**; hostile without lsof → `/usr/sbin/lsof`; ambient shape → `/tmp/hostile.COfTMy/lsof` |
| V15 | The proposed split-and-compare classifies correctly: `getconf PATH` splits into 4 entries; the hijack's directory matches none; `/usr/sbin` matches one. | `IFS=: read -ra ents <<< "$(/usr/bin/getconf PATH)"; for e in …; [[ "$e" == "${r%/*}" ]]` | `getconf PATH=[/usr/bin:/bin:/usr/sbin:/sbin]`; entries `/usr/bin`, `/bin`, `/usr/sbin`, `/sbin`; `hijack dir=/tmp/hostile.COfTMy contained=0`; `real dir=/usr/sbin contained=1` |
| V16 | `/usr/bin/getconf` and `/usr/sbin/lsof` exist as root-owned executables (so the gate can call getconf by absolute path). | `ls -l /usr/bin/getconf /usr/sbin/lsof` | `-rwxr-xr-x 1 root wheel 118288 … /usr/bin/getconf`; `-rwxr-xr-x 1 root wheel 307600 … /usr/sbin/lsof` |
| V17 | The five new identifiers are unallocated repo-wide (negative-existence, with control). | `grep -rn "$id" --include='*.sh' --include='*.go' --include='*.yml' --include='Makefile' . \| grep -v ./.git \| wc -l` for each | `PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY` 0, `run_lane_fixture_arm` 0, `standard_path_entries` 0, `real_lsof_contained` 0, `hostile_lsof_dir` 0; control `PROBE_SELFTEST_ARM_CAP_SECS` **4** |
| V18 | A `$0` re-exec arm already exists (precedent for D3). | `grep -n '/bin/bash "\$0"' tools/eval/test_motoko_connection_probe.sh` | 721 `env PROBE_SELFTEST_ARM_CAP_SECS=invalid /bin/bash "$0"` |
| V19 | Neither file currently CALLS getconf; the suite mentions it once, in the comment. | `grep -c getconf` on both; `grep -n getconf …test.sh` | suite 1 (line 36, comment), probe 0 |
| V20 | Trap (i): `expect_failure` matches with a fixed-string substring over the WHOLE captured stderr, so an append-shaped mutant is vacuous. | `sed -n '151,169p' tools/eval/test_motoko_connection_probe.sh` | test:163 `if ! grep -Fq -- "$expected" "$tmp_dir/stderr"; then`; failure paths at 159-162 (`unexpectedly succeeded`) and 163-167 (`lacked expected message`), both `exit 1` |
| V21 | Trap (ii): the suite invokes `"$probe"` directly at test:201, so a 644 mutant reds at arm 1 on file mode. `cp -p` preserves 755. | `sed -n '201p' …test.sh`; `ls -l $m/probe-9only.sh` | `"$probe" --classify-fixture "$tmp_dir/or_ips" "$tmp_dir/lsof.fixture" > "$tmp_dir/classified"`; mutant mode `-rwxr-xr-x` |
| V22 | The refusal-branch gate expects 28 = 20 + 5 + 3 production branches; this design adds none. | `grep -n 'expected_refusal_branches=' …test.sh; grep -c 'instrument_failure "' …probe.sh; grep -cE '\\\|\\\| usage$' …probe.sh; grep -c 'echo "process-tree discovery' …probe.sh` | test:770 `expected_refusal_branches=28`; 20 / 5 / 3 |
| V23 | **The prototype suite (both sub-items implemented as specified below) is green against the pristine probe, 3 of 3 runs, 46 arms.** | `PROBE_UNDER_TEST=$PWD/tools/eval/motoko_connection_probe.sh /bin/bash $m/suite-proto.sh` ×3, sequential | `P1: rc=0 elapsed=60s ok=46`; `P1b: rc=0 elapsed=60s ok=46 load={2.35 5.39 8.60}`; `P1c: rc=0 elapsed=62s ok=46 load={2.34 4.82 8.16}`; last line `PASS: 46 probe self-test arms ran`; both evidence lines `survivors=0 cleanup=0 probe_rc=1 … real_lsof=/usr/sbin/lsof` (fixtures 2861 and 2863); new arms `ok 43 - production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild`, `ok 44 - REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH`, `ok 45 - REAL_LSOF containment accepts a leading directory without an lsof`, `ok 46 - refusal-branch count …(28)`. Prototype sha256 `3fb0d84b…8b27` |
| V24 | **(a) has a sole killer:** prototype vs the group-`-9`-only mutant reds on the new arm by name with a survivor. | `PROBE_UNDER_TEST=$m/probe-9only.sh /bin/bash $m/suite-proto.sh` | `P2: rc=1 elapsed=68s ok=42 notok=1`; `not ok - production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild (outer_rc=0 survivors=1 cleanup=0 probe_rc=1)`; evidence: fixture-2861 `survivors=0`, fixture-2863 `timeout=yes … survivors=1 … outer_cap_fired=no` |
| V25 | **(b)'s pin has a killer:** the prototype with the containment gate DELETED (marker exit kept) reds on the hostile-PATH arm, and does not recurse. | python removed proto lines 30-52 (the gate block); `grep -c 'resolved outside getconf PATH'` proto 2 → nocontain 1; `bash -n` rc=0; `PROBE_UNDER_TEST=<pristine> /bin/bash $m/suite-proto-nocontain.sh` | `P4: rc=1 elapsed=58s ok=43 notok=1`; `not ok - REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH unexpectedly succeeded`. Mutant sha256 `70e42c3e…6753` |
| V26 | Fail-fast masking, measured: prototype vs 6i's both-sites mutant stops at arm 36; the new kill arm is unreached. | `PROBE_UNDER_TEST=$m/probe-both.sh /bin/bash $m/suite-proto.sh` | `P3: rc=1 elapsed=47s ok=35 notok=1`; `not ok - production run_lane timeout kills wrapper grandchild (… survivors=1 …)`; only the 2861 evidence line is printed |
| V27 | CI reaches this suite only through `make test-launchd-drivers` (`make/test.mk`), run by the `launchd drivers (bash 3.2)` job on `macos-latest`. The suite's filename appears in no workflow and not in the top-level Makefile (negative, with control). | `grep -rn motoko_connection_probe Makefile .github/` (0 hits); `grep -rn motoko_connection_probe_XYZ_NOPE …` rc=1 (negative control); `grep -n '^include' Makefile`; `awk '/^test-launchd-drivers:/…' make/test.mk`; `sed -n '554,573p' .github/workflows/ci.yml` | `make/test.mk:51 test-launchd-drivers:` → runs 8 launchd suites then `@/bin/bash tools/eval/test_motoko_connection_probe.sh`, then `bash -n` on both probe files; ci.yml:554 `name: launchd drivers (bash 3.2)`, `runs-on: macos-latest`, step `run: make test-launchd-drivers` |
| V28 | **Inherited base red, re-derived and NOT this doc's:** `tools/launchd/test_fmt_ab_schedule.sh` fails at base because `nightly-eval.sh` lost its markers (commit `c8c841e24`). | `grep -c FMT_AB_TESTABLE_FUNCTIONS tools/launchd/nightly-eval.sh` (control: same grep on the test); `/bin/bash tools/launchd/test_fmt_ab_schedule.sh` | marker count **0** (control 2); `rc=1`, `FAIL: instrument failure: FMT_AB_TESTABLE_FUNCTIONS marker extraction from …/tools/launchd/nightly-eval.sh produced no text`. `git show --stat c8c841e24` touches `tools/launchd/nightly-eval.sh` (283 lines) and neither probe file. So `make test-launchd-drivers` is red at base for a reason outside this doc; no acceptance criterion below depends on it |
| V29 | The bash-3.2 empty-array-under-`set -u` idiom the refactor needs works on this shell. | prototype's first call passes NO extra env (`run_lane_extra_env=()` then `env ${run_lane_extra_env[@]+"${run_lane_extra_env[@]}"} …`) | V23: arm 36 passed 3/3 under `/bin/bash` 3.2.57 with `set -u` |
| V30 | Fixture sleep durations in use are 2849 (orphan arm) and 2861 (run_lane arm); 2863 is free and distinct. | `grep -n '_fixture_secs=2' tools/eval/test_motoko_connection_probe.sh` | 506 `orphan_fixture_secs=2849`; 542 `run_lane_fixture_secs=2861` |
| V31 | Duplicate gate: no existing doc covers this topic (best neural score 0.38 < 0.45). The create script's search ran; its file write failed only because the tool cwd resets (it targeted `$HOME/design_docs/…`), so the file was written directly. | `create_planned_doc.sh m-motoko-group-kill-and-lsof-containment` | implemented neural top `m-executor-idle-timeout.md (0.35)`; planned neural top `m-list-repr-spike-sprint-plan.md (0.38)`; `line 209: /Users/voightkampff/design_docs/planned/…: No such file or directory` |
| V32 | The 41 → 43 arm growth the row does not know about came from iterations 32/33. | `git log --oneline -15 -- tools/eval/test_motoko_connection_probe.sh` | `115184a2e test(probe): pin the stub refusal branch's own message … (motoko row 6r) (#1027)`; `f5d031161 … give the process-tree de-race a killer … (#1020)`; `20cce785e fix(ci): give process-tree discovery its own deadline … (#1013)`; `64ca81852 … (motoko iter-32, D4) (#1008)`; `4bd58bef6 … (motoko row 6i) (#985)` |
| V33 | The new kill arm's wall-clock cost is the real grace window: suite 49s → 60s (+11s for one ~9s arm plus two sub-second re-exec arms); its 17s outer cap did not fire. | V4 vs V23 elapsed; V23/V24 evidence `outer_cap_fired=no` | 49s base; 60/60/62s prototype; `outer_cap_fired=no outer_cap_rc=0` on the 2863 fixture in every run |
| V34 | **(Revision 1)** AC2's pinned probe sha256 is measured, not transcribed, and the instrument discriminates (a different file gives a different hash). Reproduced first-party by the controller in the same worktree with identical output. | `shasum -a 256 tools/eval/motoko_connection_probe.sh; shasum -a 256 tools/eval/test_motoko_connection_probe.sh` | `f0b5e02493369099f123c42107850fe062bf60d56ccabb2a7e4690d654aabc99  tools/eval/motoko_connection_probe.sh`; control `e1d56346b08cd14dd16f2b826268f90c8b36a84f6399d8c1b8d7a4037ce71b58  tools/eval/test_motoko_connection_probe.sh` |
| V35 | **(Revision 1)** Every `expect_failure`/`expect_success` arm — a `$0` re-exec included — runs under `run_bounded` with a real wall-clock deadline and a TERM→KILL process-group escalation; the recursion guard is not the time bound. Negative control included. | `sed -n '9p;154p;174p;104p;111p;113p;118p;124p;131p;141p;149p;721p' tools/eval/test_motoko_connection_probe.sh; grep -c expect_failure …; grep -c expect_zzzz …` | 9 `ARM_CAP_SECS=${PROBE_SELFTEST_ARM_CAP_SECS:-120}`; 154 and 174 `run_bounded "$tmp_dir/stdout" "$tmp_dir/stderr" "$ARM_CAP_SECS" -- "$@"` (bodies of `expect_failure` and `expect_success`); 104 `deadline=$(( $(date +%s) + cap_secs ))`; 111 `if (( $(date +%s) > deadline )); then`; 113 `kill -TERM "-$pid"`; 118 `terminate_deadline=$(( $(date +%s) + 5 ))`; 124 `kill -9 "-$pid"`; 131 `return 199`; 141-149 `report_arm_cap()` … `exit 1`; 721 `env PROBE_SELFTEST_ARM_CAP_SECS=invalid /bin/bash "$0"` (pre-existing re-exec under the same machinery). `expect_failure` count **32**; invented token `expect_zzzz` **0** |
| V36 | **(Revision 1)** Green-path cost of a re-exec arm: one bash startup plus the gate is tens of milliseconds against a 120 s cap. | `env PROBE_SELFTEST_ARM_CAP_SECS=invalid /bin/bash tools/eval/test_motoko_connection_probe.sh` ×3, timed with `date +%s.%N` (the test:721 shape, exits at test:11); the gate body alone (`command -p -v lsof`, `/usr/bin/getconf PATH`, split, compare) ×3 | re-exec startup **15.6 / 11.1 / 9.4 ms** (rc=1 each); gate body **5.2 / 5.1 / 5.0 ms**. Margin ≈ 120 s / 0.02 s ≈ 6,000× |
| V37 | **(Revision 1) Option A, measured on HEAD:** an inner invocation given an absent `PROBE_UNDER_TEST` cannot run the suite — it reds at arm 1, deterministically, in ~20 ms. The identifier is unallocated and nothing above the marker's insertion point reads `$probe`. | `ap=$(mktemp -d /tmp/iter34-rev.XXXXXX); env PROBE_UNDER_TEST="$ap/no-probe" /bin/bash tools/eval/test_motoko_connection_probe.sh` ×2, timed; `grep -c no-probe …test.sh`; `sed -n '1,30p' …test.sh \| grep -c '\$probe'`; `grep -n PROBE_UNDER_TEST …test.sh` | rc=1, 0 ok, 1 not ok, **21.8 ms / 16.4 ms**; stderr exactly `…test_motoko_connection_probe.sh: line 201: /tmp/iter34-rev.DTYlD7/no-probe: No such file or directory` then `not ok - classification fixture: missing loopback	127.0.0.1:11434`; `no-probe` count **0**; `$probe` references in test:1-30 **0**; the only `PROBE_UNDER_TEST` reference is test:5. Control: the same suite with its real probe runs 43 arms in 46 s (V39) |
| V38 | **(Revision 1) Option B, measured on HEAD and rejected:** `PROBE_SELFTEST_ARM_CAP_SECS=1` on the arm line reaches the INNER suite's `ARM_CAP_SECS` (test:9), not the outer `run_bounded` cap (test:174); it bounds a leaked inner run to ~12 s but reds on whichever inner arm first exceeds 1 s — a load-dependent name — and re-parameterises `discovery_killer_lane_secs` (test:475). | `env PROBE_SELFTEST_ARM_CAP_SECS=1 /bin/bash tools/eval/test_motoko_connection_probe.sh`, timed | rc=1, **26 ok**, elapsed **12 s**; first red `not ok - bounded termination deadline refuses exceeded its 1s arm cap` (arm 27); last ok `ok 26 - lane sampling deadline refuses` |
| V39 | **(Revision 1)** Base suite at HEAD re-run today, for the margin arithmetic; the load is recorded. | `/bin/bash tools/eval/test_motoko_connection_probe.sh`, timed; `sysctl -n vm.loadavg` | rc=0, 43 ok, 0 not ok, **46 s**, load `{1.74 1.94 3.69}`; `PASS: 43 probe self-test arms ran`. With V4 (49 s), V23 (60/60/62 s, 46 arms) and V25 (58 s = arms 1-43 plus one sub-second arm — the exact set a leaked T7 inner run would have executed): 120 s / 58 s = **2.07×**; × 3.3-3.6 (iteration 32's load shift) = 191-209 s |
| V40 | **(Revision 1) `gemini-3-1-pro`'s class, reproduced:** the un-normalised comparison refuses alias spellings of a standard directory — a dot-segment and a symlink alias — loudly and naming both strings; the canonical spelling and a trailing slash are contained. | gate body under `env PATH="/usr/./sbin:$PATH" /bin/bash -u -c …`; `ln -s /usr/sbin $sl/sbin-alias; env PATH="$sl/sbin-alias:$PATH" …`; `env PATH="/usr/sbin/:$PATH" …`; control `env PATH="/usr/sbin:$PATH" …`; `stat -f %i` on both lsof paths; the gate's echo under the alias PATH; `/bin/realpath $sl/sbin-alias /usr/./sbin` | `/usr/./sbin` → resolved `/usr/./sbin/lsof`, dir `[/usr/./sbin]`, **contained=0**; symlink alias → `/tmp/iter34-alias.oC24v8/sbin-alias/lsof`, **contained=0**, same inode as `/usr/sbin/lsof` (1152921500312576118); trailing slash `/usr/sbin/` → bash collapses it to `/usr/sbin/lsof`, contained=1; control `/usr/sbin` → contained=1; `getconf=[/usr/bin:/bin:/usr/sbin:/sbin]` in every run. Message as printed: `not ok - REAL_LSOF resolved outside getconf PATH: /tmp/iter34-alias.oC24v8/sbin-alias/lsof is not in any of /usr/bin:/bin:/usr/sbin:/sbin; an ambient lsof would serve as the survivor oracle`. `/bin/realpath` exists (macOS 26.5.2, bash 3.2.57) and maps both aliases to `/usr/sbin` — not adopted |
| V41 | **(Revision 1)** The four `getconf PATH` entries are real directories on this host, not symlinks, so no listed entry is itself an alias today. | `ls -ld /usr/bin /bin /usr/sbin /sbin /private/usr` | all four `drwxr-xr-x root wheel`; `/private/usr: No such file or directory` |
| V42 | **(Revision 2, quorum round 2, `oc-glm-5-2`'s verbatim fix)** The hoist block's START boundary. The `skip_run_lane_fixture` guard occupies 536-538 and STAYS; the hoisted body begins at 539. The doc's `test:539` is CONFIRMED, no drift. | `sed -n '536,542p' tools/eval/test_motoko_connection_probe.sh` | `536: if (( skip_run_lane_fixture )); then` · `537: echo "UNINFORMATIVE: run_lane fixture arm requires real lsof for cwd survivor checks"` · `538: else` · `539: run_lane_timeout_secs=2` · `540: run_lane_ready_cap_secs=5` · `541: run_lane_outer_cap_secs=$(( run_lane_timeout_secs + 10 ))` · `542: run_lane_fixture_secs=2861` |
| V43 | **(Revision 2, `oc-glm-5-2`'s verbatim fix)** The hoist block's END boundary. `pass_arm` is the block's last statement at 681; the closing `fi` is 682 and is NOT hoisted. The doc's `test:681` is CONFIRMED, no drift. | `sed -n '676,682p' tools/eval/test_motoko_connection_probe.sh` | `678: echo "not ok - production run_lane timeout kills wrapper grandchild (outer_rc=… survivors=… cleanup=… probe_rc=…)" >&2` · `679: exit 1` · `680: fi` · `681: pass_arm "production run_lane timeout kills wrapper grandchild"` · `682: fi` |
| V44 | **(Revision 2, `oc-glm-5-2`'s verbatim fix)** Arm 42 and the insertion gap. Arm 42 occupies 740-741; 742 is blank; the suite-scope `PROBE_MAX_TREE_NODES` guard runs 743 (comment) to 752 (`fi`). Both `test:740-741` and `test:743-752` are CONFIRMED, no drift. | `sed -n '738,755p' tools/eval/test_motoko_connection_probe.sh` | `740: expect_failure "descendant discovery stub refusal carries its own message" "process-tree discovery deadline expired (test stub)" \` · `741: run_live PROBE_TEST_DESCENDANT_FAILURE=1` · `743: # The D4 ceiling override belongs on the wall-clock discovery arm's own env line and nowhere else.` · `749: if [[ -n "${PROBE_MAX_TREE_NODES:-}" ]]; then` · `752: fi` |
| V45 | **(Revision 2, quorum round 2, `gpt5-6-sol`'s verbatim fix)** `run_bounded` is DEFINED at 88 and the containment gate's original insertion point was test:28, i.e. ABOVE the definition — so the gate as first written could not have called it. Moving the gate below `run_bounded` is therefore necessary, not stylistic. | `grep -n '^run_bounded()' tools/eval/test_motoko_connection_probe.sh; grep -n '^ARM_CAP_SECS=' tools/eval/test_motoko_connection_probe.sh` | `88:run_bounded() {` · `9:ARM_CAP_SECS=${PROBE_SELFTEST_ARM_CAP_SECS:-120}` |
| V46 | **(Revision 2, `gpt5-6-sol`'s verbatim fix)** The unbounded-external-call-at-startup class is PRE-EXISTING and this design adds one more instance to it — so the fix is owed, and the class is wider than this doc. Four such calls already run before arm 1 at HEAD. | `sed -n '1,30p' tools/eval/test_motoko_connection_probe.sh \| grep -nE 'command -p\|getconf\|\$\('` | `4: script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)` · `6: tmp_dir=$(mktemp -d …)` · `15: host_os=$(uname -s 2>/dev/null …)` · `16: REAL_LSOF=$(command -p -v lsof 2>/dev/null \|\| true)`. Control: `grep -c 'command -p'` = **2**; `grep -c getconf` = **1** (a comment at HEAD, not a call). |

**Negative-result controls, stated:** V6 (mutant survives) is the defect and is paired with V7/V8 (same-scope mutants that DO red), so the green is a measurement. V13/V17/V19/V27/V35/V37 are absence claims, each paired with a known-positive control in the same call; V34 and V40 pair the measured value with a same-instrument control that gives a different answer. V25's "does not recurse" is evidenced by `elapsed=58s` (a recursion would have cost ≥ 60s more) and by the sole `not ok` being `unexpectedly succeeded`.

---

## Solution Design

### Overview

Two changes, both confined to `tools/eval/test_motoko_connection_probe.sh`. The production probe is **not** modified.

1. **(a)** Hoist the 6i fixture block (test:539-681) into `run_lane_fixture_arm <variant> <fixture_secs> <grace_allowance> <expected_refusal> <arm_name> [ENV=VAL…]`, call it once with 6i's parameters (byte-for-byte the same behaviour: arm 36 unchanged in name, message and fixture), and once more — behind arm 42 — with `PROBE_TEST_IGNORE_TERM=1`, a distinct fixture (`2863`), the probe:267 refusal string, and a 5s grace allowance on the outer cap.
2. **(b)** Insert a Darwin-only containment gate **after `run_bounded`'s definition closes (test:139) and before `report_arm_cap` (test:141)** — RELOCATED from test:28 by quorum round 2 so the gate can itself run under `run_bounded`; see §(b). It is followed by the marker early-exit; add two `$0` re-exec arms (hostile, benign) at the tail, guarded by a recursion refusal.

### (a) The parameterised fixture function

The body is test:539-681 verbatim with these and only these substitutions (the prototype applied them mechanically and asserted each count):

| test line (HEAD) | was | becomes |
|---|---|---|
| 542 | `run_lane_fixture_secs=2861` | `run_lane_fixture_secs=$fixture_secs` |
| 541 | `run_lane_outer_cap_secs=$(( run_lane_timeout_secs + 10 ))` | `run_lane_outer_cap_secs=$(( run_lane_timeout_secs + grace_allowance + 10 ))` |
| 548-554 | `"$tmp_dir/run-lane.<x>"` / `"$tmp_dir/run-lane-outer.<x>"` | `"$tmp_dir/run-lane-$variant.<x>"` / `"$tmp_dir/run-lane-$variant-outer.<x>"` |
| 563 | `PROBE_STUB_STATE="$tmp_dir/lane-run-lane"` | `PROBE_STUB_STATE="$tmp_dir/lane-run-lane-$variant"` |
| 561 | `env PATH="$live_bin" AILANG_BIN=ailang-stub \` | `env ${run_lane_extra_env[@]+"${run_lane_extra_env[@]}"} PATH="$live_bin" AILANG_BIN=ailang-stub \` |
| 658 | `grep -Fq -- "INSTRUMENT FAILURE: lane treatment exceeded ${run_lane_timeout_secs}s sampling deadline" \` | `grep -Fq -- "$expected_refusal" \` |
| 678, 681 | the literal arm name in the `not ok` echo and `pass_arm` | `$arm_name` |

Function prologue: `local variant=$1 fixture_secs=$2 grace_allowance=$3 expected_refusal=$4 arm_name=$5; shift 5; local run_lane_extra_env; run_lane_extra_env=("$@")`. The `${arr[@]+"${arr[@]}"}` form is required: bash 3.2 under `set -u` treats an empty array expansion as unbound (V29 shows the idiom working). `run_lane_timeout_secs=2` and `run_lane_ready_cap_secs=5` stay as they are inside the body.

The two calls:

```bash
run_lane_fixture_arm term 2861 0 "INSTRUMENT FAILURE: lane treatment exceeded 2s sampling deadline" \
  "production run_lane timeout kills wrapper grandchild"          # arm 36, unchanged, in its current position
…
run_lane_fixture_arm kill 2863 5 "INSTRUMENT FAILURE: lane treatment exceeded its bounded termination deadline" \
  "production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild" PROBE_TEST_IGNORE_TERM=1   # NEW, behind arm 42
```

Both calls sit inside `if (( skip_run_lane_fixture )); then echo "UNINFORMATIVE: …"; else …; fi` as the 6i block does today (test:536-538, 682); the new call gets its own guard at its own position, printing `UNINFORMATIVE: run_lane SIGKILL-escalation arm requires real lsof for cwd survivor checks` on non-Darwin.

**What the kill arm observes, in order (all from the existing harness, unchanged):** ready file written by the stub with wrapper/child pids and cwd; `kill -0` on the child before the timeout; the probe's rc non-zero; the probe:267 string present on stderr (`timeout=yes` now means the bounded-termination message); all seven markers including `fixture-lsof path=$REAL_LSOF cwd=<2863 dir>`; **survivors=0** by the cwd oracle; cleanup rc 0 and post-cleanup count 0; outer cap not fired. Under the single-PID mutant the survivor count is 1 (V24).

### (b) The containment gate, the marker exit, and the arms

**RELOCATED AND BOUNDED (quorum round 2, `gpt5-6-sol`'s verbatim proposed fix, applied under the narrow-refinement carve-out).** The gate was first specified at test:28. `run_bounded` is not defined until test:88 (V45), so at test:28 the gate had to call `/usr/bin/getconf PATH` directly — outside any deadline. Handling an empty result or a non-zero exit does not handle NON-TERMINATION, so a hung `getconf` would hang the whole suite before arm 1, which is standing rule 6's bounded-waits axiom violated at the one point the doc never analysed. The objection is UPHELD and the fix is the reviewer's own: **move gate execution below the existing `run_bounded` definition but still before the first arm and the marker exit; run `/usr/bin/getconf PATH` through `run_bounded` with a small explicit cap; capture its output; and refuse DISTINCTLY on timeout, on non-zero exit, and on empty output.** Scope, stated honestly: this class is PRE-EXISTING — `dirname`/`pwd`, `mktemp -d`, `uname -s` and `command -p -v lsof` all run unbounded before arm 1 at HEAD (V46). This design does not fix those; it declines to ADD a fifth. Bounding the four is filed as a queue row, not absorbed here (a pre-existing defect a reviewer surfaces is a queue row, not a revision).

Placement is therefore **after `run_bounded`'s definition (test:88) and before arm 1**, rather than after test:28. The `-z REAL_LSOF` Darwin refusal at test:22 is unchanged and still runs first:

```bash
if [[ "$host_os" == Darwin ]]; then
  # BOUNDED: the gate is the only external process this design adds at startup, and it goes
  # through the same run_bounded deadline every arm uses. GATE_CAP_SECS is small and explicit:
  # getconf is a libc lookup, so anything above ~1s is already pathological.
  gate_cap_secs=${PROBE_GETCONF_CAP_SECS:-5}
  run_bounded "$tmp_dir/gate.out" "$tmp_dir/gate.err" "$gate_cap_secs" -- /usr/bin/getconf PATH
  gate_rc=$?
  if (( gate_rc == 199 )); then
    echo "not ok - REAL_LSOF containment: /usr/bin/getconf PATH exceeded its ${gate_cap_secs}s cap; instrument failure, not a verdict" >&2
    exit 1
  fi
  if (( gate_rc != 0 )); then
    echo "not ok - REAL_LSOF containment: /usr/bin/getconf PATH exited ${gate_rc}; instrument failure, not a verdict" >&2
    exit 1
  fi
  standard_path=$(cat "$tmp_dir/gate.out")
  if [[ -z "$standard_path" ]]; then
    echo "not ok - REAL_LSOF containment: /usr/bin/getconf PATH produced no text; instrument failure, not a verdict" >&2
    exit 1
  fi
  real_lsof_dir=${REAL_LSOF%/*}
  real_lsof_contained=0
  IFS=: read -ra standard_path_entries <<< "$standard_path"
  for standard_path_entry in "${standard_path_entries[@]}"; do
    [[ "$standard_path_entry" == "$real_lsof_dir" ]] && real_lsof_contained=1
  done
  if (( ! real_lsof_contained )); then
    echo "not ok - REAL_LSOF resolved outside getconf PATH: $REAL_LSOF is not in any of $standard_path; an ambient lsof would serve as the survivor oracle" >&2
    exit 1
  fi
fi
if [[ "${PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY:-0}" == 1 ]]; then
  echo "REAL_LSOF containment check passed: $REAL_LSOF"
  exit 0
fi
```

**Precise semantics, as the row asks:**

- **How `getconf PATH` is obtained:** `/usr/bin/getconf PATH`, by absolute path, so the audit cannot itself be redirected by the PATH it audits (V16 shows the binary). An empty result is an instrument failure (exit 1, its own message), never a pass.
- **How it is split:** on `:` only, with `IFS=: read -ra` into an array (bash 3.2 supports `read -a` and `<<<`; V23 ran this under 3.2.57). No normalisation: no trailing-slash stripping, no symlink resolution, no `realpath`. On this host the entries are exactly `/usr/bin`, `/bin`, `/usr/sbin`, `/sbin` (V15).
- **What "inside" means:** `${REAL_LSOF%/*}` — the directory component of the already-absolute, already-executable path from test:16-21 — is **string-equal** to at least one entry. `/usr/sbin/lsof` → `/usr/sbin` → contained; `/tmp/hostile.X/lsof` → `/tmp/hostile.X` → not contained (V15). A symlink inside a standard directory pointing elsewhere is deliberately treated as contained: the standard directories are SIP-protected on Darwin, and resolving symlinks would widen the check into territory this row does not measure.
- **Un-normalised BY CHOICE, and what a spurious refusal looks like (quorum round 1, `gemini-3-1-pro`):** the comparison is literal string equality between the directory component of what `command -p -v` returned — which is the ambient PATH entry AS SPELLED — and the `getconf PATH` entries. Bash collapses a trailing slash itself (`/usr/sbin/` → `/usr/sbin/lsof`, contained; V40), but a dot-segment spelling (`/usr/./sbin`) or a symlink alias of a standard directory placed ahead of it resolves as `<alias>/lsof` and is refused (V40: both `contained=0`, the symlink case with the SAME inode as `/usr/sbin/lsof`). Today none of the four entries is itself a symlink (V41), so on a standard host the only way to trigger this is an ambient PATH entry that reaches ahead of the standard path under an alias spelling — which is exactly the ambient-reaches-ahead condition the gate exists to refuse, and D2 rejects resolving around it. If a future macOS makes a listed entry an alias (the `/var` → `/private/var` shape), the gate will refuse on a pristine host. The failure mode is a **hard-fail before arm 1**, never a silent pass, and it is diagnosable from its own message, which names the instrument and both strings: `not ok - REAL_LSOF resolved outside getconf PATH: <resolved lsof path> is not in any of <getconf PATH>; an ambient lsof would serve as the survivor oracle` (V40 shows it printed for the alias case). An operator recognises the spurious case by the resolved directory being an alias of one of the listed entries (`/bin/realpath` on both agrees, V40) and clears it either by fixing the PATH spelling in the caller's environment — no code change — or, if the OS changed, by widening the gate with a normalisation step under a new row, pinned by the same hostile/benign arms. Normalisation is not adopted here: `/bin/realpath` exists on this host (V40) but its presence on every supported macOS is UNMEASURED, and it would add a second executable to the trust base of an audit whose point is a minimal trust base, for a case that exists on no measured host.
- **When the hard-fail fires:** only when `host_os == Darwin` AND the directory is not an entry. It exits 1 before arm 1 with `not ok - REAL_LSOF resolved outside getconf PATH: <path> is not in any of <standard path>; an ambient lsof would serve as the survivor oracle`. The existing `-z REAL_LSOF` refusal at test:22-25 still fires first if nothing resolved.
- **Non-Darwin:** the gate is skipped entirely. `REAL_LSOF` resolution (test:16-21) is unchanged and may be empty; `skip_run_lane_fixture=1` already disables the survivor arms there (test:26-28); the two new re-exec arms print `UNINFORMATIVE: REAL_LSOF containment arms are Darwin-only, as is the gate they pin`. Nothing new can fail on Linux. (No Linux host was available — see UNMEASURED.)
- **The marker exit** is placed AFTER the gate and OUTSIDE its `if`, so a hijack refuses before the marker is reached, and a mutant that deletes the gate still exits 0 here instead of running the arms (V25).

The arms, placed at the tail (see Placement):

```bash
if [[ "${PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY:-0}" == 1 ]]; then
  echo "not ok - PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY leaked into the arm section; refusing to recurse" >&2
  exit 1
fi
if [[ "$host_os" == Darwin ]]; then
  hostile_lsof_dir="$tmp_dir/hostile-lsof"
  mkdir -p "$hostile_lsof_dir"
  printf '#!/bin/bash\nexit 1\n' > "$hostile_lsof_dir/lsof"
  chmod +x "$hostile_lsof_dir/lsof"
  # Both inner runs are given an ABSENT probe (nothing creates $tmp_dir/no-probe): an inner
  # run that ever passes the marker line reds at its arm 1 in ~20ms (V37) instead of
  # running 43 arms inside this arm's 120s run_bounded cap (V35). Per-arm env line only.
  expect_failure "REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH" "resolved outside getconf PATH" \
    env PATH="$hostile_lsof_dir:$PATH" PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY=1 PROBE_UNDER_TEST="$tmp_dir/no-probe" /bin/bash "$0"
  benign_dir="$tmp_dir/benign-dir"
  mkdir -p "$benign_dir"
  expect_success "REAL_LSOF containment accepts a leading directory without an lsof" \
    env PATH="$benign_dir:$PATH" PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY=1 PROBE_UNDER_TEST="$tmp_dir/no-probe" /bin/bash "$0"
else
  echo "UNINFORMATIVE: REAL_LSOF containment arms are Darwin-only, as is the gate they pin"
fi
```

The hostile arm reproduces V14's arm shape end to end (a real `command -p -v` under a real prepended directory) and reads the gate's own message. The benign arm is the same-shape negative control from V14 driven through the same hook; it proves the hostile arm's red is the gate and not the prepended directory breaking bash startup, and it exercises the marker exit on the green path. Each costs one bash startup plus the gate: 9-16 ms + 5 ms (V36).

### Wall-clock bounding of the re-exec arms (quorum round 1, `gpt5-6-sol`)

**(i) The bound.** Both new arms are ordinary `expect_failure` / `expect_success` arms, and both of those run their command through `run_bounded "$tmp_dir/stdout" "$tmp_dir/stderr" "$ARM_CAP_SECS" -- "$@"` (test:154 and test:174), where `ARM_CAP_SECS=${PROBE_SELFTEST_ARM_CAP_SECS:-120}` (test:9). `run_bounded` starts the command under `set -m` in its own process group, sets `deadline=$(( $(date +%s) + cap_secs ))` (test:104), polls with backoff, and on expiry (test:111) sends `kill -TERM "-$pid"` (test:113), waits a 5 s `terminate_deadline` (test:118), then `kill -9 "-$pid"` (test:124) and returns 199 (test:131), which the arm turns into `not ok - <arm> exceeded its 120s arm cap` plus the captured output tails and `exit 1` via `report_arm_cap` (test:141-149). V35 pins every one of those lines with a negative control. So the marker and the recursion guard bound DEPTH; the wall-clock bound is `run_bounded`'s, and a hung inner run IS terminated — TERM, then KILL, group-wide. The pre-existing arm at test:721 (`env PROBE_SELFTEST_ARM_CAP_SECS=invalid /bin/bash "$0"`, V18) has re-executed `$0` under exactly this machinery since it landed; the two new arms are a second and third caller of a bounded pattern, not a new pattern. The previous revision of this doc described only the recursion guard; that omission — not the design — is what the objection read.

**(ii) The margin, with numbers, and its resolution — option A.** On the green path an inner run exits at the marker line: one bash startup plus the gate, 9-16 ms + 5 ms (V36), inside a 120 s cap — a margin of roughly 6,000×. The exposure was T7: with the marker exit deleted, the benign arm's inner run would have executed arms 1-43 before reaching the tail guard — the arm set V25 measured at **58 s** (V23's full 46-arm suite: 60/60/62 s; base at V4/V39: 49/46 s) — inside a 120 s cap: a **2.07× margin**. Iteration 32 measured this host's load moving a comparable stimulus **3.3-3.6×**; 58 s × 3.3-3.6 = 191-209 s, past the cap. **Chosen: (A).** Both inner invocations carry `PROBE_UNDER_TEST="$tmp_dir/no-probe"` on their own env line — the per-arm env-line scoping iteration 32 established for `PROBE_MAX_TREE_NODES` (test:743-752 polices exactly that discipline) — naming a path nothing creates. Nothing above the marker line reads `$probe` (V37: 0 references in test:1-30), so the green path is byte-for-byte unchanged; an inner run that passes the marker line reaches arm 1's `"$probe" --classify-fixture` (test:201) and reds deterministically — measured on HEAD at **16-22 ms** with `not ok - classification fixture: missing loopback…` (V37). The leaked-inner cost falls from 58 s to ~0.02 s and the margin from 2.07× to roughly 5,000×; no load shift moves a 20 ms bash startup past 120 s. **(B) was measured and rejected** (V38): `PROBE_SELFTEST_ARM_CAP_SECS=1` on a `$0` line reaches the INNER suite's `ARM_CAP_SECS` (test:9), not the outer cap that bounds the arm (that stays the outer shell's `$ARM_CAP_SECS`, test:174); it bounds a leak to ~12 s but reds on whichever inner arm first exceeds 1 s (arm 27 at today's load — a load-dependent name) and re-parameterises `discovery_killer_lane_secs` (test:475). **(C) was rejected** because 2.07× sits inside the measured 3.3-3.6× load shift. **What exceeding the bound looks like:** on the green path the arm would red as `… exceeded its 120s arm cap` — a WRONG VERDICT (a green arm reported red), which is why the margin had to be four orders of magnitude, not two; it cannot happen short of a hung `bash` startup. On T7 the verdict is red either way (the mutant is a mutant), so a cap-shaped red would only have been a WRONG EXPLANATION (tails full of inner `ok N` lines instead of `refusing to recurse`); with option A it does not arise. **Residual:** a TRIPLE test-side mutant (marker exit, absent-probe scoping AND tail guard all removed) recurses depth-unbounded, each level inside its parent's 120 s + 5 s cap, and each level's `run_bounded` puts its child in a fresh process group so the outer KILL reaches only its direct child's group — UNMEASURED, not a Test Plan drill, and the same exposure test:721 has always had.

**(iii) AC8's `elapsed < 90s` was an observation, not a bound.** It is read after the run completes; it can show that no inner run happened, not stop one. The bound on every arm, the re-exec arms included, is `run_bounded`'s 120 s cap + 5 s TERM grace, per arm (V35). AC8 is reworded to say so.

### Files to Modify/Create

- `tools/eval/test_motoko_connection_probe.sh` (+~85 / −~10 LOC): containment gate + marker exit inserted after `run_bounded`'s definition (test:88) and before arm 1 — RELOCATED from line 28 by quorum round 2, see §(b); the 6i block hoisted into `run_lane_fixture_arm` with the seven substitutions above and one call in place; the kill-arm call, recursion guard and two containment arms (each carrying `PROBE_UNDER_TEST="$tmp_dir/no-probe"` on its own env line, option A) inserted after line 741 (after arm 42's `run_live PROBE_TEST_DESCENDANT_FAILURE=1`) and before the suite-scope `PROBE_MAX_TREE_NODES` guard at line 743-752. The prototype that measured all of this is 874 lines vs 795.
- `tools/eval/motoko_connection_probe.sh` — **not modified**. Its sha256 must still be `f0b5e02493369099f123c42107850fe062bf60d56ccabb2a7e4690d654aabc99` after the sprint — measured at base with a discriminating control in V34, not transcribed.
- `changelogs/v0.32-current.md` — one entry under the motoko/eval tooling heading.
- No new files.

---

## Arm placement and fail-fast ordering

The suite is fail-fast: `expect_failure` exits 1 on either failure path (V20) and every hand-rolled arm ends in `exit 1`, so the first red terminates the run and every later arm is unreached (V26 shows it: 35 ok then stop). Several arms are wall-clock bounded — the `PROBE_TIMEOUT_SECS=4` `run_live` arms (26-34, 36's readiness cap of 5s at test:540, the `refusing live path` arm) — and iteration 33 measured that inserting a forking arm AHEAD of them gave **4 reds in 19 runs** at position 26 against **0 in 5** at position 42 and **0 in 17** at base (`m-motoko-stub-refusal-arm.md`, "Arm PLACEMENT").

**Placement decision:** all three new arms go **after arm 42** (`descendant discovery stub refusal carries its own message`, test:740-741) and before the suite-scope env guards and the refusal-branch gate. Arm 42 was itself placed as "the last forking arm" by iteration 33; the new arms extend that tail, so every wall-clock-bounded arm begins at the same offset it has at base — the mechanism is unreachable by construction rather than argued by rate. The resulting order is 43 = kill arm (~9s, the only expensive one), 44 = hostile, 45 = benign, 46 = refusal-branch gate. Three prototype runs at this placement were 3/3 green at load 2.3-2.9 (V23).

**The honest residual:** the kill arm is itself wall-clock bounded (5s readiness cap, 17s outer cap) and shares arm 36's exposure to host contention; placing it last means no other arm is tipped by it, not that it cannot flake. That exposure is row 6p's (derive bounds from an in-test stimulus), not this row's.

**Code locality vs ordering:** the function is defined where the 6i block lives today (inside the first `skip_run_lane_fixture` guard) and the second call is ~60 lines later; a reader following arm 36 will find the function beside it and the second call named in the function's comment.

---

## Test Plan

Every mutant is a `cp -p` copy in `/tmp` (mode preserved — a `sed > file` mutant is 644 and reds at arm 1 for its mode, V21), is `bash -n` clean, has its intended effect asserted by a count that moves while a same-family count stays put, and is run with `PROBE_UNDER_TEST=<copy>` so the tree is never edited. Read WHICH arm failed, never the rc alone. Because `expect_failure` is a `grep -Fq` substring over the whole stderr (V20), a mutation must CHANGE or REMOVE the matched text; APPEND-shaped mutants are vacuous by construction and are not accepted as evidence.

| # | Mutation (exact) | Must red with `not ok` line containing | Killer status | Measured? |
|---|---|---|---|---|
| T1 | probe:261 `kill -9 "-$pid"` → `kill -9 "$pid"` (group `-9` count 1→0, group TERM stays 1) | `production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild (outer_rc=0 survivors=1 …)` | **SOLE killer** — arms 1-42 green, rc=1, 42 ok | **Yes, V24** |
| T2 | 6i's mutant: probe:252 AND 261 both → single-PID (both group counts → 0) | `production run_lane timeout kills wrapper grandchild (… survivors=1 …)` | arm 36 reds first and masks the new arm (fail-fast); 6i's coverage is intact | **Yes, V7 (base suite) and V26 (prototype)** |
| T3 | probe:252 `kill -TERM "-$pid"` → `kill -TERM "$pid"` only (group TERM 1→0, group `-9` stays 1) | `production run_lane timeout kills wrapper grandchild (… survivors=1 …)` | arm 36 sole killer for the TERM half; the new arm is unreached and would be green by construction (TERM is a no-op on a TERM-immune tree) | **Yes, V8** (the "would be green" half is UNMEASURED because fail-fast hides it) |
| T4 | Delete the containment gate block (the `if [[ "$host_os" == Darwin ]]; then … fi` holding `resolved outside getconf PATH`), keep the marker exit | `REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH unexpectedly succeeded` | **SOLE killer**, rc=1, 43 ok; no recursion (58s) | **Yes, V25** |
| T12 | **(Revision 2, `gpt5-6-sol`'s verbatim fix)** In a `/tmp` suite copy, replace `/usr/bin/getconf` in the gate line with a sleeping fixture (`"$tmp_dir/slow-getconf"`, a `#!/bin/sh` script running `sleep 30`, mode 755), and set `PROBE_GETCONF_CAP_SECS=2` | `REAL_LSOF containment: /usr/bin/getconf PATH exceeded its 2s cap; instrument failure, not a verdict`, with **zero** `ok N` lines before it (the refusal must precede arm 1) and total elapsed **< 10s** (proving the cap terminated it rather than the 30s sleep completing) | **SOLE killer** of the timeout branch; distinguishes timeout from the empty-output and non-zero branches, which T13/T14 cover | UNMEASURED — the drill is specified here and is a sprint deliverable |
| T5 | Change the gate's message text, e.g. `resolved outside getconf PATH` → `resolved OUTSIDE the standard path` (substring-breaking) | `… refuses an ambient lsof ahead of getconf PATH lacked expected message: resolved outside getconf PATH` | sole killer (arm 44) | UNMEASURED — predicted from V20's matcher |
| T6 | Invert the containment comparison `==` → `!=` in the `standard_path_entries` loop (test:169) | **CORRECTED after the evaluator measured it (iteration 34, non-blocking finding 1).** The doc PREDICTED a startup refusal before arm 1. It does not happen, and the reason is structural: the loop is a DISJUNCTION (`any entry equals`), so inverting it to `any entry differs` is satisfied by 3 of this host's 4 `getconf PATH` entries no matter what `REAL_LSOF` is — `real_lsof_contained` is set to 1 unconditionally and the gate is silently defeated. The mutation IS still caught, two arms later: arm 44 reds with `REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH unexpectedly succeeded`. | detected, but NOT by the branch the row named — the predicate is defeated, not tripped. A single-entry `getconf PATH` would make the original prediction true, which is why it read as obvious. | **Yes — measured by the iteration-34 evaluator (rc=1, 43 ok), and reproduced by the controller by inspection of test:167-174 against a 4-entry `getconf PATH` (`/usr/bin:/bin:/usr/sbin:/sbin`)** |
| T7 | Remove the marker early-exit alone (gate intact) | arm 44 stays green (the hijack refuses before the marker); arm 45's inner run passes the marker line, reaches arm 1 with the absent probe and reds there: `REAL_LSOF containment accepts a leading directory without an lsof failed` with `not ok - classification fixture: missing loopback` (and `no-probe: No such file or directory`) in the captured stderr; the tail guard is NOT reached | arm 45 sole killer; the leaked inner run costs ~20 ms (V37) inside the arm's 120 s `run_bounded` cap (V35) | Mechanism measured on HEAD (V37); the drill itself on the rebuilt prototype is UNMEASURED — the `/tmp` prototype did not survive. Run it in the sprint; expected outer elapsed ≈ 60 s. T4+T7 together is now cheap (both inner runs red at arm 1; arm 44 reds first as `lacked expected message`) but is not a listed drill |
| T8 | Drop `PROBE_TEST_IGNORE_TERM=1` from the kill arm's call (test-side self-check that the arm reaches the escalation) | `production run_lane SIGKILL escalation … (outer_rc=0 survivors=0 …)` — red because `timeout=no`: the probe emits the probe:272 sampling-deadline string, not probe:267's | arm 43 sole killer | UNMEASURED — predicted from V6 (that run IS this configuration on the base fixture) |
| T9 | Add a production refusal branch to the probe (e.g. a new `instrument_failure "…"`) | `refusal-branch drift: probe has 29 refusal branches, this suite is written for 28` | count gate (now arm 46) sole killer; unchanged behaviour, arm number moves 43 → 46 | UNMEASURED this session; the gate's own anti-vacuity floor is at test:778-782 |
| T10 | APPEND to the gate message (`… survivor oracle (x)`) | nothing — **vacuous by construction** (V20) | rejected as evidence | measured in principle by iteration 33 (`E X` still matches) |
| T11 | 6i's mutant B: neuter `set -m` in `run_lane` | arm 36 via the `INSTRUMENT DEGRADED` branch, per 6i's IV3 | arm 36; unchanged by this design | UNMEASURED this session; cited from 6i's doc |

**Which write does each read?** T1 reads probe:261 through the 2863 fixture's cwd oracle. T2/T3 read probe:252 through the 2861 fixture's oracle. T4/T5/T6 read the gate's presence, text and predicate through a real `command -p -v` under a real prepended directory. T7 reads the absent-probe scoping on the arm's own env line, through arm 1 of the inner run, with the tail guard as the depth backstop. T8 reads the arm's own env line. None of the new arms observes a value set alongside the mechanism it pins.

---

## Acceptance Criteria

Scoped, as instructed, to the suite and to targeted greps. **Deliberately NOT a criterion:** `make test-launchd-drivers`, the `test` CI check, and the `launchd drivers (bash 3.2)` CI check — all red at base for an unrelated, first-party-verified reason (V28, `c8c841e24`, owned by V1 with PR #1030 in flight). A gate red at base measures the repo, not the change.

| # | Command | Base (recorded) | Required after |
|---|---|---|---|
| AC1 | `bash -n tools/eval/motoko_connection_probe.sh; bash -n tools/eval/test_motoko_connection_probe.sh` | rc=0, rc=0 (V5) | rc=0, rc=0 |
| AC2 | `shasum -a 256 tools/eval/motoko_connection_probe.sh` | `f0b5e02493369099f123c42107850fe062bf60d56ccabb2a7e4690d654aabc99` — measured, with a discriminating control (V34) | unchanged — the probe is not touched |
| AC3 | `/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/s.out 2> /tmp/s.err; echo rc=$?; grep -c '^ok ' /tmp/s.out; cat /tmp/s.out /tmp/s.err \| grep -c 'not ok'; tail -1 /tmp/s.out` | rc=0, 43, 0, `PASS: 43 probe self-test arms ran` (V4) | rc=0, **46**, 0, `PASS: 46 probe self-test arms ran`, **three consecutive runs** (V23 shape) |
| AC4 | `grep -n '^ok 4[3-6]' /tmp/s.out` | n/a | `ok 43 - production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild`; `ok 44 - REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH`; `ok 45 - REAL_LSOF containment accepts a leading directory without an lsof`; `ok 46 - refusal-branch count still matches the set this suite covers (28)` |
| AC5 | `grep 'run_lane evidence' /tmp/s.out \| grep -o 'fixture-[0-9]*\\\|survivors=[0-9]*\\\|outer_cap_fired=[a-z]*' \| paste -sd' ' -` | one fixture, `fixture-2861 outer_cap_fired=no … survivors=0` | two: `fixture-2861 outer_cap_fired=no … survivors=0 fixture-2863 outer_cap_fired=no … survivors=0` |
| AC6 | T1 mutation drill (Test Plan) | mutant survives, rc=0 43 ok (V6) | rc=1, 42 ok, sole `not ok` is arm 43 with `survivors=1` |
| AC7 | T2 mutation drill | rc=1, 35 ok, arm 36 (V7) | rc=1, 35 ok, arm 36 — unchanged |
| AC8 | T4 mutation drill | n/a | rc=1, 43 ok, sole `not ok` is arm 44 `unexpectedly succeeded`; elapsed ≈ 58 s (V25), recorded as a post-hoc OBSERVATION that no inner run happened — not a bound. The bound is `run_bounded`'s 120 s + 5 s per arm (V35) |
| AC9 | `grep -c 'expected_refusal_branches=28' tools/eval/test_motoko_connection_probe.sh` | 1 (V22) | 1 |
| AC10 | `grep -c 'run_lane_fixture_arm' tools/eval/test_motoko_connection_probe.sh` | 0 (V17) | ≥ 3 (definition + two calls); and `grep -c 'run_lane_fixture_secs=2861'` = 0, `grep -c 'run_lane_fixture_secs=\$fixture_secs'` = 1 |
| AC11a | `grep -c -- '-- /usr/bin/getconf PATH' tools/eval/test_motoko_connection_probe.sh` | 0 hits (V19) | **exactly 1** — the single `run_bounded` invocation of the gate; its line number must be **>139 and <168** (after `run_bounded` closes, before the first arm) |
| AC11b | `grep -c '/usr/bin/getconf PATH' tools/eval/test_motoko_connection_probe.sh` | 0 hits (V19) | **exactly 4** — one `run_bounded` argument plus the three distinct refusal messages (timeout / non-zero / empty). **CORRECTED: the pre-relocation AC11 demanded 1 hit and would have failed a CORRECT implementation** — found by the sprint-planner and reproduced first-party by the controller (the literal appears 4× in the shipped gate). |
| AC12 | The three new arms appear AFTER `descendant discovery stub refusal carries its own message` in `/tmp/s.out` line order | n/a | line numbers strictly increasing 42 → 43 → 44 → 45 → 46 |
| AC13 | `grep -c 'PROBE_UNDER_TEST="\$tmp_dir/no-probe"' tools/eval/test_motoko_connection_probe.sh; grep -c 'no-probe' tools/eval/test_motoko_connection_probe.sh` | 0; 0 (V37 — identifier unallocated) | 2; 3 (both re-exec env lines plus the comment) — the scoping stays on the arm lines, never at suite scope |

---

## Conflict Surface

Two files are relevant: `tools/eval/motoko_connection_probe.sh` (read by the suite, NOT modified) and `tools/eval/test_motoko_connection_probe.sh` (modified). This is not a parser/typechecker change, but the section is written anyway because the suite is shared machinery with three sibling rows (6i, 6r, 6p) in the same file.

**What depends on the touched file, by command:**

- `make/test.mk:51-64` — VERIFIED (`awk '/^test-launchd-drivers:/…' make/test.mk`): `test-launchd-drivers` runs the suite as `@/bin/bash tools/eval/test_motoko_connection_probe.sh` after eight `tools/launchd/test_*.sh` suites, then `bash -n`s both probe files. The suite is the ninth step; the fifth (`test_fmt_ab_schedule.sh`) is red at base (V28), so the target never reaches this suite on dev today.
- `.github/workflows/ci.yml:554-573` — VERIFIED (`sed -n '554,573p'`): job `launchd drivers (bash 3.2)`, `runs-on: macos-latest`, asserts `/bin/bash` is 3.x, then `run: make test-launchd-drivers`. This is the ONLY CI leg that runs the suite (`grep -rn motoko_connection_probe .github/` = 0 hits; the wiring is through make, V27).
- `tools/eval/motoko_connection_probe.sh` — read via `$probe` / `PROBE_UNDER_TEST` (test:5); its refusal-branch count (28) is asserted by the suite's last arm. Untouched here.
- Design docs that cite suite line numbers: `m-motoko-stub-refusal-arm.md` (+ sprint plan), `m-motoko-discovery-arm-discriminating-refusal.md` (+ sprint plan), `m-motoko-connection-probe-run-lane-harness.md` (+ sprint plan), `m-motoko-fmt-remeasurement-instrument.md` — VERIFIED by `grep -rln motoko_connection_probe design_docs/`. Their cited line numbers (e.g. "test:476", "test:731") already drifted after #1020/#1027 and are historical; this change shifts everything after test:28 by ~30 lines and everything after test:539 by more. No doc is edited for that; the mission log records the shift.

**Recent history of the touched file, each verified with `git show --stat --format='%h %s' <sha>`:**

- `115184a2e` `test(probe): pin the stub refusal branch's own message, behind the bounded arms (motoko row 6r) (#1027)` — `test_motoko_connection_probe.sh` **7 +**; plus the 6r doc and sprint plan. Created arm 42 and the "last forking arm" tail this design extends. **Collision:** the new arms insert directly after its 2-line arm (test:740-741).
- `f5d031161` `test(probe): give the process-tree de-race a killer, and guard the new knob at suite scope (#1020)` — test file only, **58 +/−**. Added the suite-scope `PROBE_TREE_DISCOVERY_SECS` guard (test:754-764). **Collision:** the new arms must stay ABOVE that guard and the `PROBE_MAX_TREE_NODES` guard (test:743-752), and must not export either variable.
- `20cce785e` `fix(ci): give process-tree discovery its own deadline (reconciled onto dev) (#1013)` — probe **9 +/−**, test **35 +/−**. Set `expected_refusal_branches=28`. **Collision:** none unless a concurrent change moves the count.
- `64ca81852` `test(probe): make the wall-clock discovery arm assert its own refusal (motoko iter-32, D4) (#1008)` — probe **4 +/−**, test **25 +/−**, plus 6n docs. **Collision:** none; the wall-clock arm (test:476-481) is far from both insertion points.
- `4bd58bef6` `test(eval): pin the production run_lane process-group kill with a behavioural arm (motoko row 6i) (#985)` — test **256 +/−**, changelog, 6i docs. Authored the block this design hoists into a function. **Collision:** HIGH with any concurrent edit inside test:536-682; the hoist rewrites every line's indentation. Row 6p's future bound-derivation helper will touch `run_lane_ready_cap_secs` (test:540) inside this block — do 6p after this lands, not concurrently.
- `1caf02e44` `docs(eval): scope the REAL_LSOF immunity claim to the stub this suite installs` — test **10 +/−**, comment only (the diff was read: lines 30-38 replaced). **Collision:** the containment gate lands directly below that comment; the comment's last sentence ("tracked as charter row 6o") should be updated to point at the gate.
- `c8c841e24` `fix(eval): harness crashes were published as model failures` — touches `tools/launchd/nightly-eval.sh` (283 −), `os-rotation-filler.sh`, Go tests, models.yml, changelog, benchmarks JSON — **neither probe file**. This is the inherited-red cause (V28); no collision with this design.
- **Negative control:** `051c8d9c1` `docs(design): rulings D2/D3/D5 + sprint plan for M-COMPLETION-PATH-PARITY` — lists only `design_docs/planned/m-completion-path-parity*.md`; no relevant file. So an "untouched" reading from `git show --stat` is a measurement, not a broken command.

**Enumeration control:** `git log --oneline -15 -- tools/eval/test_motoko_connection_probe.sh` returns exactly the six probe-touching SHAs above plus `b4da09b53`, `fd1fa9e01`, `086b72184`, `c1950750c`, `684ebc23e` — none of the cited SHAs is invented and `1caf02e44` is confirmed by its own `git show`.

**Shared-machinery decisions (reuse vs override):** `run_bounded` — reuse, unchanged. `expect_failure`/`expect_success` — reuse. The stub `ailang-stub` — reuse; its `PROBE_TEST_IGNORE_TERM` knob gains a second caller and its comment should say so. `fixture_sleep_pids`/`cleanup_fixture_sleeps` — reuse, unchanged; they gain a third cwd (2863). `PROBE_UNDER_TEST` (test:5) — reuse; the two re-exec env lines give the inner run an absent value (option A), which is a per-arm assignment and never a suite-scope one. Nothing is overridden.

---

## Non-Goals and residuals

- **No production change.** The probe's grace window stays a hardcoded 5s; making it a knob was rejected in D1.
- **Row 6p** (derive wall-clock bounds from an in-test stimulus) is not touched; this doc adds one more arm that will benefit from it and says so.
- **Row 6s** (nothing in CI notices an arm disappearing) is not touched; note that AC4 pins the new arms by name only for this sprint's evaluator, not for CI.
- **The inherited `test-launchd-drivers` red** (V28) is V1's and is neither fixed nor worked around here.
- **Observed but out of scope (code reading, UNMEASURED):** on a non-Darwin host WITHOUT `lsof`, `REAL_LSOF` is empty and the orphan arm (test:503-531, arm 35) calls `"" -a -c sleep -d cwd`, which fails silently under `2>/dev/null` and counts 0 survivors vacuously. The suite's only CI leg is macOS (V27), so this cannot bite today; it is recorded so the next Linux leg does not inherit it unnoticed.
- The symlink-inside-a-standard-directory case is deliberately treated as contained, and an alias spelling OF a standard directory is deliberately refused (see "Un-normalised BY CHOICE" above, V40-V41).

---

## Platform honesty and UNMEASURED claims

Every measurement here is darwin/arm64, GNU bash 3.2.57, this rig, `/usr/sbin` on PATH, load average 2.3-2.9 during the prototype runs. The following are stated in this doc without a first-party measurement and are labelled as such wherever they appear:

1. **GitHub `macos-latest` runner behaviour** — that `/usr/bin/getconf PATH` returns the same four entries there and that `command -p -v` shows the same ambient-PATH behaviour. Both are platform properties of macOS/bash 3.2 and the CI job already asserts bash 3.x, but the runner was not exercised from this session. Consequence if wrong: the gate would refuse at startup on the runner with its own explicit message, which is fail-loud, not silent.
2. **Non-Darwin behaviour** of the new code paths — by construction they are skipped (`host_os != Darwin`), but no Linux host ran the modified suite.
3. **T5, T6, T8, T9, T11** in the Test Plan are predicted from measured mechanisms (V20, V15, V6, the count gate) but were not themselves run. **T7**'s mechanism IS measured on HEAD (V37: an absent-probe inner run reds at arm 1 in 16-22 ms) but the drill on the rebuilt prototype is not — the `/tmp` prototype did not survive to this revision. T1, T2, T3, T4 were run and are the load-bearing ones.
4. **The kill arm's flake rate under heavy load** — 3/3 at load ≤ 2.9 is not a rate; iteration 33's readiness-cap red at load 39-46 applies to this arm as much as to arm 36.
5. **The "would be green" half of T3** — fail-fast hides it; it follows from TERM being a no-op on a TERM-immune tree (V9), not from a run.
6. **The `macos-latest` runner's ambient PATH spelling** — whether any entry ahead of `/usr/sbin` is an alias spelling (V40's class) or holds a Homebrew `lsof`. Either refuses loudly at startup with the V40 message naming both strings; neither is silent.
7. **Triple test-side mutant recursion** (marker exit + absent-probe scoping + tail guard all removed) — depth-unbounded, each level inside its parent's 120 s + 5 s cap; not a Test Plan drill; the same exposure the pre-existing test:721 re-exec has always had.
8. **`/bin/realpath` on every supported macOS** — present here (V40); its portability is why normalisation is deferred rather than adopted.

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| The hoist changes arm 36's behaviour subtly | The substitutions are enumerated line by line; AC5 requires the 2861 evidence line unchanged; AC7 requires 6i's mutant to still red on arm 36 |
| The kill arm flakes under load | Placed last so it tips nothing else; readiness/outer caps unchanged in kind; residual is 6p's |
| A double test-side mutant (gate + marker both removed) recurses | Both inner invocations carry an absent probe (option A), so a leaked inner run reds at arm 1 in ~20 ms (V37); the tail recursion guard remains as the depth backstop; every arm sits under `run_bounded`'s 120 s + 5 s cap regardless (V35) |
| A future macOS aliases a standard directory, or an operator's ambient PATH spells one as an alias | The gate hard-fails before arm 1 with both strings in its message (V40); recognise it by the resolved directory being an alias of a listed entry; clear it by fixing the PATH spelling, or — if the OS did it — widening the gate under a new row pinned by the same two arms |
| The gate refuses on a legitimately unusual CI image | The refusal names the resolved path and the standard path; that is the intended loud failure, and the fix would be to the image, not the gate |
| Empty `getconf` output passes vacuously | Explicit instrument-failure branch on empty output |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | +1 | The escalation path is now exercised deterministically (TERM-immune tree) instead of never |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | +1 | The oracle's provenance (`REAL_LSOF`) is asserted, not assumed |
| A4: Explicit Authority | +1 | An ambient binary can no longer silently become the oracle |
| A5: Bounded Verification | +1 | Each half of the kill has its own local killer |
| A6: Safe Concurrency | 0 | Process-group semantics unchanged |
| A7: Machines First | 0 | Test tooling only |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | +11s suite time, stated |
| A10: Composability | +1 | One parameterised fixture function instead of two copies |
| A11: Structured Failure | +1 | New refusals carry their own distinct messages |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +7** → proceed. Hard-violation check: A1 no implicit nondeterminism; A3 no hidden side effects (the hostile stub lives in the suite's own `$tmp_dir`, removed by the existing EXIT trap at test:7); A4 no ambient access granted — the opposite; A7 not human-convenience over machine analysis.

---

## Related Documents

- `design_docs/implemented/v0_35_0/m-motoko-connection-probe-run-lane-harness.md` — row 6i: the arm this doc parameterises; its IV2 mutant is T2 here.
- `design_docs/planned/m-motoko-stub-refusal-arm.md` — row 6r: the placement measurement (4/19 vs 0/5) this doc obeys, and the `grep -Fq` / mode-644 traps.
- `design_docs/planned/m-motoko-discovery-arm-discriminating-refusal.md` — row 6n: the three-message refusal shape and the refusal-branch gate.
- `design_docs/motoko-mission.md` row 6o (charter line 1088) — the item; row 6p — the bound-derivation residual this doc hands one more arm to.
- Commit `1caf02e44` — the comment-only narrowing this doc turns into a gate.

The neural related-doc search (V31) surfaced nothing above 0.38; the documents above were found by `grep -rln motoko_connection_probe design_docs/`.
