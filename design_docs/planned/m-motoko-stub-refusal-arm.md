# M-MOTOKO-STUB-REFUSAL-ARM: pin the `(test stub)` refusal message behaviourally, symmetric with the other two branches

**Status**: NEW — awaiting design quorum. One sprint-sized item, ~1 iteration.
**Target**: v0.34.x (next motoko iteration)
**Priority**: P1 (loop-health — a hunk with no killer; the `(test stub)` suffix is held by a static grep and by nothing behavioural)
**Estimated**: 1 iteration (~1-2 hours implementation + mutation validation)
**Dependencies**: Row 6r of the motoko mission queue. No new dependency; the two shell files are self-contained and the suite is hermetic. This doc is the direct successor of the row 6n arc (`m-motoko-discovery-arm-discriminating-refusal.md` + its sprint plan), which introduced the three distinct messages and the wall-clock/node-ceiling arms but explicitly left the stub's message unasserted.

## Problem Statement

`tools/eval/motoko_connection_probe.sh` has three refusal branches inside `descendant_pids()`:

| probe line | branch | message |
|---|---|---|
| 184 | test-only `PROBE_TEST_DESCENDANT_FAILURE=1` | `process-tree discovery deadline expired (test stub)` |
| 189 | real in-loop wall clock | `process-tree discovery deadline expired (wall clock)` |
| 197 | node ceiling | `process-tree discovery exceeded $MAX_TREE_NODES nodes` |

The caller collapses every one of them (probe:220) into `instrument_failure "process-tree discovery failed"`.

Branches (2) and (3) each have a self-test arm asserting **their own** discriminating message:
- test:476 asserts `process-tree discovery deadline expired (wall clock)`
- test:731 asserts `process-tree discovery exceeded 3 nodes`

Branch (1) does **not**. The only arm that drives it — test:358-359,
`expect_failure "descendant discovery deadline refuses at the caller" "process-tree discovery failed" run_live PROBE_TEST_DESCENDANT_FAILURE=1` —
asserts the **caller's** wrapper message, not the stub's. So the `(test stub)` suffix, whose entire purpose is to make a stubbed refusal
distinguishable from a real one in a log, is held by nothing behavioural. The only thing holding its text is a **static grep** in the row 6n
sprint plan's acceptance table (`m-motoko-discovery-arm-discriminating-refusal-sprint-plan.md` line 155, AC A1.3: `grep -Fc 'process-tree
discovery deadline expired (test stub)'` → 1) — a claim about the file's text rather than about behaviour. The row 6n design doc itself
self-discloses this: *"The stub's message is never directly asserted (arm :357 asserts the caller wrapper), so changing it is safe."*

**Impact:** a future edit that strips or rewords the `(test stub)` suffix lands green (measured — M2 below). The loop's own rule says a hunk
with no killer is a finding rather than a failure; this row exists to give the hunk a killer.

## Goals

**Primary Goal:** Give branch (1) a self-test arm that asserts its **own** discriminating message
`process-tree discovery deadline expired (test stub)`, the symmetric treatment branches (2) and (3) already have, so the `(test stub)`
suffix is pinned behaviourally and the M2 mutant (strip the suffix) reds the suite.

**Secondary Goal:** Keep the three branches symmetric and the refusal-branch-count gate honest: adding a self-test **arm** must not move
`expected_refusal_branches` (28); only adding a **production branch** would.

**Non-Goal (scope discipline):** no production-semantics change to the probe. The stub still returns 1 and the caller still collapses it to
`process-tree discovery failed`. This row does **not** touch the neighbouring queue rows (see Residuals).

## High-Impact Decision

### (A) Add a stub-message arm, sitting BESIDE the existing caller-message arm — chosen

The controller's lean is (A), and I agree. Option (B) — declaring the branch behaviourally unpinned in a code comment — would name the gap
where a reader meets it, but it would not give the hunk a killer: the M2 mutant would still land green, and the `(test stub)` suffix would
still be held by nothing. (A) converts the static grep into a behavioural assertion, makes all three branches symmetric, and is cheap — the
stub short-circuits before the walk, so the new arm is deterministic and fast (no timing sensitivity, no flake surface).

**The new arm sits BESIDE, not in place of, the existing caller-message arm at :358.** The two assert two different strings on the same
stderr and each can fail for a reason the other cannot:

| arm | drives | asserts | can fail for (unique to it) |
|---|---|---|---|
| :358 "descendant discovery deadline refuses at the caller" (existing) | `run_live PROBE_TEST_DESCENDANT_FAILURE=1` | `process-tree discovery failed` | the caller stops collapsing a `descendant_pids` failure into that wrapper — e.g. the `if ! pids=$(descendant_pids ...)` guard in `sample_tree` is removed and the probe proceeds with an empty pid scope, reddening `assert_pid_scope` with a *different* message; or the wrapper string is reworded. Does **not** fail if the stub's message text changes. |
| NEW "descendant discovery stub refusal carries its own message" | `run_live PROBE_TEST_DESCENDANT_FAILURE=1` | `process-tree discovery deadline expired (test stub)` | the stub's own message text changes — the `(test stub)` suffix is stripped (M2), or the message is reworded. Does **not** fail if the caller stops refusing (the stub message is still emitted to stderr before the caller's behaviour). |

Concretely, if the caller's guard is removed, the probe still exits 1 (via `assert_pid_scope` → `invalid empty or malformed pid scope: ${pids:-<empty>}` — the exact string carries a `: <pids-or-empty>` suffix, V15), so
stderr still carries `process-tree discovery deadline expired (test stub)` and the new arm stays green while the caller-message arm reds.
Conversely, if the stub's message is stripped, the caller-message arm stays green while the new arm reds. They are complementary, not
redundant; replacing one with the other would lose a distinct pin.

**First-party basis for "stderr carries BOTH strings":** established by direct reproduction (see Verification Log V7) — with
`PROBE_TEST_DESCENDANT_FAILURE=1` the probe's stderr contains both `process-tree discovery deadline expired (test stub)` (count 1) and
`INSTRUMENT FAILURE: process-tree discovery failed` (count 1). So a single `expect_failure` asserting the stub message is well-formed.

**Refusal-branch-count gate (M5):** `expected_refusal_branches=28` at test:763 counts **production** refusal branches in the probe
(`instrument_failure "` = 20, `|| usage$` = 5, `echo "process-tree discovery` = 3). This design adds a self-test **arm** in the test file
only; it adds no production branch, so the count stays 28 and the gate (arm 42) keeps passing. Adding a production branch would move it; this
design does not.

## Verification Log

Every row below is a claim about the codebase's current state, the command that produced it, and its observed output. Rows labelled
**controller-measured** are first-party measurements taken by the controller in this session (2026-09-02) and are cited as such; all other
rows I ran myself on this unmodified worktree (HEAD `7292ec780`).

| # | Claim | Command | Observed output |
|---|---|---|---|
| V1 | Baseline suite is green: 42 arms, 0 failures, rc=0, no skips. | `/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/m1_base.log 2>&1; echo rc=$?; grep -c '^ok ' /tmp/m1_base.log; grep -c '^not ok' /tmp/m1_base.log; tail -1 /tmp/m1_base.log; grep -ci '/skip/i' /tmp/m1_base.log` | rc=0; `^ok ` = 42; `^not ok` = 0; last line `PASS: 42 probe self-test arms ran`; skip = 0. (Controller-measured M1 agrees: rc=0, 42 ok, 0 not ok.) |
| V2 | The M2 mutant — strip `(test stub)` from branch (1) only — survives: suite stays green. | controller-measured M2: mutant LANDED (sha256 `f0b5e0249336` → `df45556cebcf`), PARSES (`bash -n` rc=0), effect asserted (`grep -Fc 'expired (test stub)'` 1→0 while `grep -Fc 'expired (wall clock)'` stays 1). | rc=0, 42 ok, 0 not ok. **The mutant survives — this is the defect.** |
| V3 | The same-scope known-positive control — strip `(wall clock)` from branch (2) only — reds the suite on the wall-clock arm by name. | controller-measured M3: mutant LANDED (`f0b5e0249336` → `645fbfc044c7`), PARSES, effect 1→0 on that suffix only. | rc=1, 32 ok, 1 not ok; failing arm by name: `not ok - descendant discovery refuses on the real wall-clock deadline lacked expected message: process-tree discovery deadline expired (wall clock)`. So the harness CAN red for exactly this class of edit; M2's green is a measurement, not a broken instrument. M2 and M3 differ in exit code (0 vs 1) — no false symmetry. |
| V4 | Instrument trap: `sed > file` creates mode 644, and the suite invokes `"$probe" --classify-fixture ...` directly at test:201, so an un-chmod'd mutant reds at the FIRST arm for its file mode, not for the mutation. | controller-measured M4; I additionally verified the direct invocation and the repo file mode: `sed -n '198,203p' tools/eval/test_motoko_connection_probe.sh`; `stat -f '%Sp %N' tools/eval/motoko_connection_probe.sh` | test:201 is `"$probe" --classify-fixture "$tmp_dir/or_ips" "$tmp_dir/lsof.fixture"`; repo probe mode is `-rwxr-xr-x` (755). Any mutation drill must `chmod 755` the mutant and read WHICH arm failed, never the exit code alone. |
| V5 | `expected_refusal_branches` is 28 and the gate passes at baseline. | `grep -n 'expected_refusal_branches' tools/eval/test_motoko_connection_probe.sh`; `grep -c 'instrument_failure "' tools/eval/motoko_connection_probe.sh`; `grep -cE '\|\| usage$' tools/eval/motoko_connection_probe.sh`; `grep -c 'echo "process-tree discovery' tools/eval/motoko_connection_probe.sh` | `expected_refusal_branches=28` at test:763; counts 20 + 5 + 3 = 28. The gate (arm 42) passes as part of V1. (Controller-measured M5 agrees.) |
| V6 | The three branches and the caller collapse exist at the stated lines with the stated messages. | `grep -n 'process-tree discovery deadline expired (test stub)\|process-tree discovery deadline expired (wall clock)\|process-tree discovery exceeded\|instrument_failure "process-tree discovery failed"' tools/eval/motoko_connection_probe.sh` | 184 `(test stub)`; 189 `(wall clock)`; 197 `exceeded $MAX_TREE_NODES nodes`; 220 `instrument_failure "process-tree discovery failed"`. |
| V7 | With `PROBE_TEST_DESCENDANT_FAILURE=1`, the probe's stderr carries BOTH the stub message and the caller wrapper. | Hermetic live-path reproduction (coreutils symlinked into a scratch `bin/`, stub `uname`/`dig`/`pgrep`/`lsof`/`ailang-stub`, `env PATH="$T/bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=4 PROBE_TEST_DESCENDANT_FAILURE=1 PROBE_STUB_STATE="$T/lane" /bin/bash tools/eval/motoko_connection_probe.sh treatment control "$T/out.json"`), then `grep -Fc` on captured stderr. | probe rc=1; stderr contains `process-tree discovery deadline expired (test stub)` (count 1) and `INSTRUMENT FAILURE: process-tree discovery failed` (count 1). |
| V8 | The existing caller-message arm at :358-359 asserts the caller wrapper, not the stub message. | `sed -n '358,359p' tools/eval/test_motoko_connection_probe.sh` | `expect_failure "descendant discovery deadline refuses at the caller" "process-tree discovery failed" \` / `  run_live PROBE_TEST_DESCENDANT_FAILURE=1` |
| V9 | The wall-clock arm (test:476) and node-ceiling arm (test:731) assert their own messages. | `grep -n 'real wall-clock deadline\|node-count ceiling' tools/eval/test_motoko_connection_probe.sh` | 476 `expect_failure "descendant discovery refuses on the real wall-clock deadline" "process-tree discovery deadline expired (wall clock)"`; 731 `expect_failure "descendant discovery refuses on the node-count ceiling" "process-tree discovery exceeded 3 nodes"` |
| V10 | Both shell files parse cleanly at base. | `bash -n tools/eval/motoko_connection_probe.sh; echo rc=$?`; `bash -n tools/eval/test_motoko_connection_probe.sh; echo rc=$?` | rc=0; rc=0. |
| V11 | The probe file's sha256 matches the controller's M2 baseline hash. | `shasum -a 256 tools/eval/motoko_connection_probe.sh` | `f0b5e02493369099f123c42107850fe062bf60d56ccabb2a7e4690d654aabc99` (matches controller's `f0b5e0249336`). |
| V12 | The static grep that currently holds the `(test stub)` text lives in the row 6n sprint plan's acceptance table. | `sed -n '140,170p' design_docs/planned/m-motoko-discovery-arm-discriminating-refusal-sprint-plan.md` | AC A1.3: `grep -Fc 'process-tree discovery deadline expired (test stub)' tools/eval/motoko_connection_probe.sh` → base 0, required after M1 **1**. |
| V13 | `expect_failure` matches with `grep -Fq -- "$expected" "$tmp_dir/stderr"` — a fixed-string SUBSTRING match over the ENTIRE captured stderr file, not a last-line match and not a structured field. | `sed -n '151,169p' tools/eval/test_motoko_connection_probe.sh` | `expect_failure() { local name=$1 expected=$2 rc; shift 2; run_bounded "$tmp_dir/stdout" "$tmp_dir/stderr" "$ARM_CAP_SECS" -- "$@"; rc=$?; ... if ! grep -Fq -- "$expected" "$tmp_dir/stderr"; then echo "not ok - $name lacked expected message: $expected" >&2; ... }` — the matcher is `grep -Fq -- "$expected" "$tmp_dir/stderr"`. |
| V14 | Behavioural, the stronger arm: an `expect_failure` asserting the stub's message PASSES on correct code and REDS on the M2 defect. | controller-measured (2026-09-02), throwaway test-side mutant run before routing: (arm 1) test file with the caller-message arm's expected string swapped to `process-tree discovery deadline expired (test stub)`, run against the PRISTINE probe; (arm 2) the SAME mutated test file run against the stripped-suffix probe (M2). | arm 1 → rc=0, 42 ok, 0 not ok (the assertion passes on correct code). arm 2 → rc=1, 24 ok, 1 not ok, failing arm by name: `not ok - descendant discovery deadline refuses at the caller lacked expected message: process-tree discovery deadline expired (test stub)`. Outcomes DIFFER (rc 0 vs 1) — a discriminating measurement, not a false symmetry. Together they establish that an `expect_failure` asserting the stub's message both passes on correct code and reds on the defect — exactly what the new arm must do. |
| V15 | `assert_pid_scope` exists, is called, and its refusal message carries a `: ${pids:-<empty>}` suffix. | `grep -n "assert_pid_scope" tools/eval/motoko_connection_probe.sh`; `grep -n "invalid empty or malformed pid scope" tools/eval/motoko_connection_probe.sh`; controls in the same call: `grep -c "instrument_failure" tools/eval/motoko_connection_probe.sh`; negative control `grep -c "assert_zzz_scope" tools/eval/motoko_connection_probe.sh` | 208 `assert_pid_scope() {`; 223 `assert_pid_scope "$pids"`; 210 `[[ "$pids" =~ ^[0-9]+(,[0-9]+)*$ ]] || instrument_failure "invalid empty or malformed pid scope: ${pids:-<empty>}"`. Controls: `instrument_failure` count = 21 (known-positive); `assert_zzz_scope` = 0 (negative). The zeros in this family are measurements, not a broken pattern. |
| V16 | The refusal-branch gate's anti-vacuity floor exists and `expected_refusal_branches=28`. | `grep -n "expected_refusal_branches=28" tools/eval/test_motoko_connection_probe.sh`; `grep -n "refusal-branch gate" tools/eval/test_motoko_connection_probe.sh`; `grep -n "counter matched nothing" tools/eval/test_motoko_connection_probe.sh` | `expected_refusal_branches=28` at test:763; `[[ -f "$probe" ]]` asserted at test:766 BEFORE any counter runs; anti-vacuity floor at test:771-774: `if (( actual_instrument_failures == 0 || actual_usage_refusals == 0 || actual_echo_refusals == 0 )); then echo "not ok - refusal-branch counter matched nothing; instrument failure, not a verdict" >&2; exit 1; fi`. |

**Empty/negative-result controls:** V2's "mutant survives" is a negative result (no red) — it is a controller-measured first-party run, not an
inference, and it is paired with V3, the same-scope known-positive control that proves the harness fires for exactly this class of edit.
V5's "no production branch added" is a design claim, not a measurement; the gate's own anti-vacuity floor (test:771-774, V16) refuses a zero
counter, so a broken counter cannot silently pass. V15's `assert_zzz_scope` = 0 is a negative control paired with the known-positive `instrument_failure` = 21, so the zeros in that family are measurements, not a broken pattern.

## Test Plan

Each row names the mutation it kills and the observable it reads. For each I state whether it is predicted to be a SOLE killer or a member
of a larger red set. **Every mutation drill must `chmod 755` the mutant copy and read WHICH arm failed, never the exit code alone (M4).**
The suite is run as `PROBE_UNDER_TEST=<mutant copy> /bin/bash tools/eval/test_motoko_connection_probe.sh` so the probe is swapped without
touching the tree.

**THE HARNESS IS FAIL-FAST, AND THAT ORDERS THIS TABLE** (`gemini-3-1-pro`, round 2, upheld). `expect_failure`
ends in `exit 1` on both of its failure paths (test:161 `unexpectedly succeeded`, test:166 `lacked expected message`), so the
first failing arm terminates the suite and every later arm is UNREACHED. The controller's own readings prove it rather than
assume it: V3 stops at 32 ok and V14 at 24 ok, both out of a 42-arm baseline. Since the new arm is inserted immediately AFTER
the caller-message arm at :358, any mutant that reds the caller arm masks the new one. This does not weaken T1 — T1's mutant
leaves the caller arm green, so the new arm is reached and is its sole killer — but it does mean a "both arms red" or "the
other arm stays green" prediction is unobservable in this harness, and the rows below say so.

| # | Mutation (named) | Observable it reads | Predicted killer(s) |
|---|---|---|---|
| T1 | **M2 — strip `(test stub)` from branch (1) only** (probe:184 → `process-tree discovery deadline expired`). This is the defect. | The NEW stub-message arm reds: `not ok - ... lacked expected message: process-tree discovery deadline expired (test stub)`. The caller-message arm (:358) stays green (it asserts `process-tree discovery failed`, unchanged); the refusal-branch count stays 28 (echo count still 3). | **SOLE killer** — the new arm is the only arm that reads the `(test stub)` string. This is the row's whole point. |
| T2 | **M3 — strip `(wall clock)` from branch (2) only** (probe:189). Known-positive control, controller-measured. | The wall-clock arm (test:476) reds by name. The new stub-message arm stays green. | Not the new arm's killer — the wall-clock arm is the sole killer. Included to prove the harness still fires for this class of edit after my change. |
| T3 | **Alter the stub message's `(test stub)` suffix** (probe:184 → `process-tree discovery deadline expired (stub)`). This is the substring-breaking form of the reword mutant. The append-shaped form (`... (test stub) X`) is REJECTED: `expect_failure` matches with `grep -Fq` fixed-string substring, so appending text leaves the expected string present and the arm still passes. Measured (controller, 2026-09-02) with E=`process-tree discovery deadline expired (test stub)`: `grep -Fq -- "$E"` rc=0 on a file containing E exactly, rc=0 on a file containing "E X", rc=1 on a file containing "...expired (stub)". A substring assertion is only broken by an edit that REMOVES or ALTERS the expected substring, never by one that ADDS around it. | The NEW stub-message arm reds: `not ok - ... lacked expected message: process-tree discovery deadline expired (test stub)`. Caller-message arm stays green. | **SOLE killer** — only the new arm reads the exact stub string. |
| T4 | **Remove the stub branch entirely** (delete the `if [[ "${PROBE_TEST_DESCENDANT_FAILURE:-0}" == 1 ]]` block at probe:183-186). | The caller-message arm (:358) reds first with `unexpectedly succeeded` and **aborts the suite before the new arm executes** — `expect_failure` ends in `exit 1` (test:161/166), so the harness is fail-fast and every arm after the first failure is unreached. | **Ordering-masked.** Applying `gemini-3-1-pro`'s round-2 fix verbatim: *"the caller-message arm reds and aborts the suite before the new arm executes (meaning the new arm is not reached, though the mutation is still caught)."* The mutation IS caught; the new arm is simply not the catcher. |
| T5 | **Alter the caller wrapper's asserted substring** (probe:220 → `instrument_failure "process-tree discovery FAILED"`). The append-shaped form (`... failed (X)`) is rejected for the same reason as T3: it keeps the asserted substring `process-tree discovery failed` present, so `grep -Fq` still matches and the caller arm would not red. | The caller-message arm (:358) reds and **aborts the suite, so the new arm's assertion is never reached** — the same fail-fast ordering as T4. | **Ordering-masked.** Applying `gemini-3-1-pro`'s round-2 fix verbatim: *"the caller-message arm reds and aborts the suite, so the new arm's assertion is never reached."* Note what this costs the doc's earlier argument: T5 can no longer be cited as evidence that the two arms discriminate different strings, because the second arm never runs. That claim now rests on **V13/V14** (the controller's two-arm behavioural measurement) alone, which is a measurement rather than a prediction. |
| T6 | **Add a production refusal branch** (e.g. a new `echo "process-tree discovery ..."` in `descendant_pids`). | The refusal-branch-count gate reds: `refusal-branch drift: probe has 29 refusal branches, this suite is written for 28`. | Not the new arm's killer — the count gate is the sole killer. Applying `gemini-3-1-pro`'s round-2 fix verbatim: *"correct T6 to note that adding an arm shifts the refusal-branch-count gate from arm 42 to arm 43."* The gate's arm NUMBER moves 42 → 43 because this change inserts one arm ahead of it; the gate's `expected_refusal_branches` VALUE stays 28, because a self-test arm is not a production branch. |

**Add-shaped mutants are vacuous under `grep -Fq`:** every `expect_failure` in this suite matches with `grep -Fq -- "$expected" "$tmp_dir/stderr"` — a fixed-string substring over the entire captured stderr (V13). A mutation that only ADDS text around an asserted substring (e.g. appending ` X`) leaves the expected string present, so the arm still passes; only an edit that REMOVES or ALTERS the expected substring can red it. T3 and T5 were both audited for this shape and rewritten to the substring-breaking form; T1/T2 (strip a suffix) and T4 (remove the branch → `unexpectedly succeeded`) already break the substring or the rc==0 check.

**Which write does each read?** T1/T3 read the stub's own message string (probe:184) — a value set alongside the stub branch, so the new arm
can fail for the reason it claims. T2 reads the wall-clock string (probe:189). T4 reads the presence of the stub branch itself. T5 reads the
caller wrapper (probe:220). T6 reads the production-branch count. None of the new arm's assertions observe a value set alongside the
mechanism it claims to pin.

## Acceptance Criteria

Every command below was run on this **unmodified** worktree and its base result recorded (see Verification Log). A gate already red at base
would measure the repo, not the change; none of these are.

| # | Command | Base (recorded) | Required after the change |
|---|---|---|---|
| AC1 | `bash -n tools/eval/motoko_connection_probe.sh; echo rc=$?` | rc=0 (V10) | rc=0 |
| AC2 | `bash -n tools/eval/test_motoko_connection_probe.sh; echo rc=$?` | rc=0 (V10) | rc=0 |
| AC3 | `/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/suite.log 2>&1; echo rc=$?; grep -c '^ok ' /tmp/suite.log; grep -c '^not ok' /tmp/suite.log; tail -1 /tmp/suite.log` | rc=0, 42 ok, 0 not ok, `PASS: 42 probe self-test arms ran` (V1) | rc=0, **43 ok** (the new arm adds one), 0 not ok, `PASS: 43 probe self-test arms ran` |
| AC4 | The refusal-branch-count gate (arm 42, part of the AC3 suite) | passes (V1, V5) | still passes with `expected_refusal_branches=28` unchanged — the change adds a test arm, not a production branch |
| AC5 | The new arm is the sole killer for the M2 mutant (strip `(test stub)`). | n/a (mutation drill, not a base measurement) | T1 reds the suite with the new arm as the failing arm by name |

AC3's 43-ok count is behavioural (the suite ran and passed one more arm), not a static grep for a string — the static-grep shape is the very
defect this row exists to fix. AC5 is the mutation drill that proves the new arm is non-vacuous; it is carried in the Test Plan and re-stated
here as the acceptance gate for the fix's purpose.

## Conflict Surface

Two files are touched: `tools/eval/motoko_connection_probe.sh` (read-only for this change — the new arm is in the test file) and
`tools/eval/test_motoko_connection_probe.sh` (one new `expect_failure` block inserted immediately after the existing caller-message arm at
:358-359). The probe file is not modified by this design, so the only collision surface is the test file.

Relevant history on these two files. **PROVENANCE VERIFIED BY COMMAND** (`gpt5-6-sol`, round 2, upheld: *"saying the
history was “read” is still assertion rather than verification"*). Its own proposed command was run —
`git show --stat --oneline <sha> -- tools/eval/motoko_connection_probe.sh tools/eval/test_motoko_connection_probe.sh` —
and the observed output is recorded per row below. Enumeration control: `git log --oneline -6 -- <the two files>` returns
`f5d031161, 20cce785e, 64ca81852, 4bd58bef6, b4da09b53, fd1fa9e01`, i.e. all three cited SHAs are among the most recent
commits to touch these files and none is invented. Negative control: the same `git show --stat` for `7292ec780` (HEAD, a
docs-only commit) lists **no files**, so an empty file list is a measurement and not a broken command.


- **#1008 (`64ca81852`, motoko iteration 32)** — VERIFIED: subject `test(probe): make the wall-clock discovery arm assert its own refusal (motoko iter-32, D4) (#1008)`; `motoko_connection_probe.sh` 4 ++--, `test_motoko_connection_probe.sh` 25 +++++-----, 2 files, 22 insertions / 7 deletions. — introduced the three distinct messages (probe:184/189/197), the wall-clock arm asserting its
  own message, the echo-shaped refusal counter, the suite-scope `PROBE_MAX_TREE_NODES` leak guard, and the `[[ -f "$probe" ]]` assertion. This
  is the commit that created the current shape and the self-disclosed stub gap.
- **#1013 (`20cce785e`, V1 iteration 317)** — VERIFIED: subject `fix(ci): give process-tree discovery its own deadline (reconciled onto dev) (#1013)`; `motoko_connection_probe.sh` 9 +++---, `test_motoko_connection_probe.sh` 35 +++----, 2 files, 24 insertions / 20 deletions. — de-raced `run_lane`'s single deadline: added `PROBE_TREE_DISCOVERY_SECS`, moved
  `expected_refusal_branches` 27→28 (re-derived as 20+5+3), and changed the wall-clock arm's env line and the bounded-termination arm's env
  line. A concurrent edit to the wall-clock arm's env line or to `expected_refusal_branches` would collide with this history.
- **#1020 (`f5d031161`, most recent)** — VERIFIED: subject `test(probe): give the process-tree de-race a killer, and guard the new knob at suite scope (#1020)`; `test_motoko_connection_probe.sh` ONLY, 1 file, 51 insertions / 7 deletions — so this commit did **not** touch the probe, which narrows its collision surface to the test file. — gave the de-race a killer (wall-clock arm lane deadline at `ARM_CAP_SECS + 30`, stub driver kept
  alive for the same duration) and added the suite-scope `PROBE_TREE_DISCOVERY_SECS` leak guard. It also added `PROBE_TEST_PGREP_LOOP_DELAY`
  to the pgrep stub.

**What a concurrent edit would collide with:** my insertion point is immediately after test:358-359, which is inside the block of
`run_live`-driven `expect_failure` arms (test:330-363). Any concurrent edit that adds/removes/reorders arms in that block, or that changes
`run_live`'s env line (test:319-321), would conflict. The wall-clock arm (test:476) and node-ceiling arm (test:731) are far from the
insertion point and are not touched. The refusal-branch-count gate (test:763-782) is not touched; my change must not move `expected_refusal_branches`. The probe file is untouched, so no collision there.

## Residuals / queue rows this does NOT fix

- **Row 6o** — the SIGKILL-escalation group form and the `REAL_LSOF` PATH assertion. Not touched; out of scope.
- **Row 6p** — deriving the node ceiling from an in-test stimulus. Not touched; out of scope.
- The wall-clock arm's accepted structural weakness (it can die on the arm cap rather than on a message mismatch under a full de-race
  revert) is documented in the test file's own comment and is **not** re-litigated here. The new stub-message arm has no such weakness: the
  stub short-circuits before the walk, so it is deterministic and fast.

## Platform Honesty

Every local reading in this doc is **darwin/arm64, GNU bash 3.2.57, this rig, with `/usr/sbin` present in PATH** (`command -p -v lsof` →
`/usr/sbin/lsof`, rc=0). The **windows and ubuntu CI legs are unrun locally**. I do not declare any branch "unreachable" or any ordering
"impossible" from this one host. The new arm is timing-independent (the stub returns before the walk), so it carries no new platform
sensitivity; but the suite as a whole still depends on `/usr/sbin` being on PATH for `lsof` (V1's iteration 317 recorded that a shell
without it dies early at "run_lane fixture arm requires real lsof"), so a run showing far fewer than 42 arms is a PATH problem, not a code
problem.

## Quorum Verification Log

**Round 1 — BLOCKED 3/3, zero absentees, $0.0628.** Three reviewers (gpt5-6-sol, gemini-3-1-pro, oc-glm-5-2) each raised a PREMISE objection — a claim about the codebase this doc had not established. The controller ran all three rather than forwarding them; two are UPHELD and one is REFUTED. The design direction — option (A), a new arm asserting the stub's own message, sitting BESIDE the existing caller-message arm — was not disputed by any reviewer and is unchanged.

| Reviewer | Objection (one line) | Disposition |
|---|---|---|
| gpt5-6-sol | T3's append-shaped mutant (`(test stub)` → `(test stub) X`) is vacuous: `expect_failure` uses `grep -Fq` fixed-string substring match, so appending text leaves the expected string present and the arm still passes. | **UPHELD — defect.** T3 replaced with the substring-breaking `(test stub)` → `(stub)` (measured: `grep -Fq` rc 0 on exact, rc 0 on "E X", rc 1 on "...(stub)"). T5 audited and fixed for the same shape. |
| gemini-3-1-pro | The "two arms fail independently" claim rested on unverified codebase facts (`assert_pid_scope`, the refusal-branch gate's anti-vacuity floor) with no Verification Log row. | **UPHELD — documentation.** Facts are true but unverified; added V15 (`assert_pid_scope` existence/call-site/exact message with `: ${pids:-<empty>}` suffix + controls) and V16 (anti-vacuity floor at test:771-774, `[[ -f ]]` at :766, `expected_refusal_branches=28` at :763). |
| oc-glm-5-2 | `expect_failure`'s matching semantics were unverified; the arm might fail if the harness matched only the last line or a structured field. | **REFUTED by measurement.** Source (test:151-169, V13) shows `grep -Fq -- "$expected" "$tmp_dir/stderr"` — a fixed-string substring over the entire stderr file; behavioural arms (V14) show the assertion PASSES on pristine code (rc=0, 42 ok) and REDS on the M2 defect (rc=1, 24 ok, 1 not ok) — a discriminating measurement. |

**Line-citation note:** the controller's round-1 note cited the anti-vacuity floor at test:772-776 and `[[ -f "$probe" ]]` at :767; the measured truth is test:771-774 and :766 (the doc's existing citations were already correct). The Verification Log rows above carry the measured numbers.

## Quorum round 2 and the narrow-refinement carve-out (controller-applied, 2026-09-02)

Round 2 synthesis **BLOCKED**, $0.0875, 3/3 external reviewers present, `.synthesis.absent_reviewers` = `[]`
(cross-checked: `[.reviewers[]|select(.present==false)|.model]` = `[]`). Verdicts: `gpt5-6-sol` reject,
`gemini-3-1-pro` reject, **`oc-glm-5-2` pass** — its first pass, having been the round-1 rejecter on
`expect_failure` semantics, which the controller refuted by measurement.

**Objection SURFACES tracked per round** (V1 iteration 257's rule — the disposition turns on where objections
land, not on the round count). R1: test-plan mutant validity · verification-log completeness · harness
semantics. R2: conflict-surface provenance · test-plan fail-fast ordering. The objections are **SPREAD across
different surfaces, not localised onto one**, so this is not the SPLIT signal that rule describes; and one
reviewer flipped to pass.

**The revision and the one re-quorum are spent, so the disposition is the NARROW-REFINEMENT CARVE-OUT, not a
park.** Both surviving objections (a) carry a concrete reviewer-authored `proposed_fix` and (b) dispute no
design DIRECTION — one is attribution, one is a factual correction to predicted test outcomes. The carve-out
was first ratified for this mission at iteration 29. Fixes applied, VERBATIM where the reviewer supplied text:

| Reviewer | Objection surface | Disposition |
|---|---|---|
| `gpt5-6-sol` | Conflict Surface cited three SHAs without commands or observed outputs | **APPLIED.** Ran the reviewer's own `git show --stat --oneline <sha> -- <the two files>`; each bullet now carries its verified subject and diffstat, with an enumeration control and a negative control. The reviewer's alternative ("remove the unsupported historical attributions") was NOT taken, because the attributions turned out to be true and are useful. |
| `gemini-3-1-pro` | T4/T5 predict outcomes the fail-fast harness cannot produce; T6's arm number is stale | **APPLIED VERBATIM.** T4 and T5 rewritten in the reviewer's own words as ordering-masked; T6 notes the gate moves arm 42 → 43 while `expected_refusal_branches` stays 28. A new fail-fast preamble was added to the Test Plan, evidenced by `expect_failure`'s two `exit 1` paths and by the controller's own V3/V14 arm counts. |
| `oc-glm-5-2` | (passed) noted the literal insertion text is not written out | **NOT BLOCKING, and deliberately left to the sprint plan** — the exact `expect_failure` line is an implementation detail the planner and executor own, and AC3/AC5 are what make it non-vacuous. Recorded here so it is not silently dropped. |

**What this costs, stated rather than hidden:** T5 was previously cited as evidence that the two arms
discriminate different strings. Under fail-fast ordering it cannot be. That claim now rests on V13/V14 —
the controller's two-arm behavioural measurement — which is a measurement rather than a prediction, so the
argument is stronger than it was, not weaker.
