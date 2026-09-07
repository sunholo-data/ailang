# M-LAUNCHD-NOTIFY-SUBSHELL-OBSERVATION: Restore Notification Test Observability and Bound Production Notification Calls

**Status**: Planned — mission iteration 347, revision under human ruling **D-60** (attended 2026-09-07, Mark Edmondson). This is **NO LONGER test-only**: the ruling overrides the earlier "separate production follow-up" deferral and folds production bounding INTO this design. **Lane:** Mission harness; test-observation repair **and** production bounded-call repair, both required by D-60.
**Target**: v0.35.2 · **Priority**: P0 (inherited dev-CI regression RED on `dev`) · **Estimated**: 4 hours (test repair + production bounding + mutation + evaluation)
**Dependencies**: existing notification repair `63a0d2b32`; base commit `81abc956d` (origin/dev).
**Created**: 2026-09-07 · **Revision**: iter-347, per D-60.

## Problem Statement

At base `81abc956d`, `make test-launchd-drivers` passes all 54 pin-root checks then fails the
`launchd drivers (bash 3.2)` notify suite **20 passed / 7 failed**, rc=1 (M1). The fixture is
`tools/launchd/test_driver_notify.sh`; the reported name `test_mission_control_notifications.sh`
does not exist at this base.

Commit `63a0d2b32` retained direct sends, added canonical message-store environment settings,
captured their diagnostic output, and added a drain for previously spooled failures. Its
`_out=$(ailang ... 2>&1)` executes the test's `ailang()` function in a subshell; that stub records
calls by assigning to `TRACE`, so the record disappears when the subshell exits (V5, G3). All seven
red checks depend on the missing AILANG call or its title.

This design has **two halves** that D-60 rules must land together:
1. **Test-observation repair** (retained from the iter-344 design): restore the seven assertions by
   making the fixture observe sends across shell boundaries via a file-backed trace, with the
   `ailang()`/`gh()` stubs made reachable through the production bounding helper (see Conflict
   Surface M3).
2. **Production bounding** (NEW, folded in by D-60): bound the three notification paths D-60 names —
   the drain-time send, the direct send, and the GitHub notice — with the existing `_mc_bounded`
   helper, so a hanging `ailang`/`gh` cannot stall a notification retry chain, a spool drain, or a
   fire's wrap-up.

## Verification Log

Rows M1–M5 are the controller's first-party measurements at `81abc956d`, reused verbatim; rows
V16+ are measured first-party in this iter-347 session at the same tree. All channel calls are
stubbed; no production notification was sent.

| ID | Claim / command or read | Observed |
|---|---|---|
| M1 | (controller) `bash tools/launchd/test_driver_notify.sh` at 81abc956d | **20 passed, 7 failed**, rc=1. Seven named failures = the causal-mapping set: fires on both channels (ailang), titled as UNPINNED, lane fires on ailang, lane keeps its own title, drift-a, drift-c, drift-g. |
| M2 | (controller) `grep -n "ailang messages send\|gh issue comment" tools/launchd/mission-control.sh` | Drains send P:154 (`_mc_drain_notices`), direct send P:184 (`_mc_notify`), GH post P:198 (`_mc_notify`); **seven further unbounded sites** P:1470,1473,1486,1835,1838,1852,1855. Positive control of a bounded site: P:1817/1820 already read `_mc_bounded 30 ailang messages send …` and `_mc_bounded 30 gh issue comment …`. |
| M3 | (controller) `_mc_bounded` at P:496 runs `( exec "$@" ) >"$out_f" 2>&1 &` — `exec` bypasses shell functions, resolves an external binary. Measured: `f() { echo FUNCTION-CALLED; }; ( exec f )` → `/bin/bash: exec: f: not found` rc=127; direct `f` → `FUNCTION-CALLED` rc=0. | **THE COLLISION**: wrapping the notifies in `_mc_bounded` makes the suite's `ailang()`/`gh()` **function** stubs unreachable; the driver would exec the REAL binaries. `_mc_bounded` also runs in a background subshell, so it does NOT fix the lost-`TRACE` observation (a file spy is still required). `_mc_bounded` returns 124 on expiry, combined output in `$MC_BOUNDED_OUT` (`_mc_notify` today reads send output from command substitution). |
| M4 | (controller) read `_mc_bounded` P:487-512 | `PROBE_TIMEOUT` defaults to 120 (model probes, not messaging); the two existing bounded notification sites use **30s**; 124 on expiry (mirrors GNU `timeout`); combined stdout+stderr → `$MC_BOUNDED_OUT`; kill then SIGKILL after +2s. |
| M5 | (controller) read `_mc_notify` P:167-203 | Retries three times, `sleep 5` then `sleep 10`; per-call bound and retry count are different guarantees. |
| V1 | `git rev-parse HEAD`; `/bin/bash --version` | 81abc956dace8a9e297d26b232da3f0ce56b31a3; Bash 3.2.57 on arm64 Darwin. |
| V2 | re-ran `bash tools/launchd/test_driver_notify.sh` | 20 passed, 7 failed, rc=1; the seven named rows identical to M1. |
| V3 | re-ran M2 grep; read `_mc_drain_notices` P:145-165, `_mc_notify` P:167-203, `_mc_bounded` P:496-512, slot-notify P:1817/1820 | Line numbers and semantics match M2/M4/M5; refusal block P:1470/1473 ends in `exit 1`; model-change P:1486; post-record P:1835/1838 and rc-fail P:1852/1855 are episode-gated by marker files; slot-notify P:1808-1827 is the only already-bounded notify path. |
| V4 | M3 mechanism prototype: PATH executable stub + `_mc_bounded 30 env AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac stub …` (full helper reproduced) | rc=0; env vars reach the stub (`STORE=<gcp> PROJ=<ailang-multivac>`); stub's combined output lands in `MC_BOUNDED_OUT`; stub appends an `AILANG:` record to a file trace. Positive control that the exec chain and env passthrough both survive `_mc_bounded`. |
| V5 | V4 with a never-returning stub (`while :; do sleep 1; done`), `secs=2` | rc=124 after ~3s (designer reading). **CONTROLLER RE-MEASURED (V5c, below): 5s, not ~3s** — the designer's number is optimistic; use V5c. |
| V5c | (controller, first-party re-measure of V4/V5 at this tree) reproduced `_mc_bounded` verbatim via `awk '/^_mc_bounded\(\) \{/,/^\}/'`; ran (a) a PATH executable stub through `_mc_bounded 30 env AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac fakeailang …`, (b) a never-returning PATH stub with `secs=2`, (c) a shell FUNCTION as the command | (a) rc=0, `MC_BOUNDED_OUT=[STORE=<gcp> PROJ=<ailang-multivac>]`, file trace `AILANG:messages send controlplane body --title T` — V4 CONFIRMED. (b) rc=124 in **5 s wall-clock**, not ~3 s: `_mc_bounded`'s poll loop sleeps 2 s per turn and then grants a further 2 s before SIGKILL, so the observed cutoff is `bound + up to ~2 s poll granularity + 2 s grace`. **Any acceptance budget must be `bound + 4 s` at minimum, not `bound + 1 s`.** (c) rc=127, `exec: f: not found` — negative control, M3 CONFIRMED. |
| V6 | read `run()`/`run_drift()` in `test_driver_notify.sh` P:30-95 and assertions P:105-165 | Labs define `ailang()`/`gh()`/`log()` functions appending to `TRACE`, source `notify.sh` then the block in a subshell. log() is never wrapped in `_mc_bounded` (only `ailang`/`gh` are) — so log() need not route through the helper, but per G1 it must still append to `MC_TRACE_FILE` so the observation medium stays unified. 27 assertions include the seven red rows. |
| G1 | (controller, objection 1) `grep -n 'checkno' tools/launchd/test_driver_notify.sh` | **Two negative AILANG assertions read the captured trace `$T`:** T:121 `checkno "no ailang call" "$T" "AILANG:"` and T:144 `checkno "lane SILENT when healthy" "$T" "AILANG:"`. Both currently PASS (in the 20 green). Under a split trace (ailang/gh records → `MC_TRACE_FILE`, log-only → `$T`), they would keep reading `$T` and pass vacuously even if the AILANG record genuinely fired into the file — hiding a regression. Mandates exactly ONE unified observation medium. |
| G2 | (controller; RC-after-`printf` defect, re-measured at this base) `sed -n '45,57p' tools/launchd/test_driver_notify.sh`; shell control | `run()`'s emit sequence ends `. "$2"` then `printf "%s" "$TRACE"` then `echo "` / `RC:$?` (T:52-55) — so `RC:$?` is expanded AFTER `printf`. `/bin/bash -c 'false; actual=$?; printf "%s" ""; printf "actual=%s after_printf=%s\n" "$actual" "$?"'` → `actual=1 after_printf=0`: the block's real status is 1, the value the fixture records is 0. CONFIRMED. Positive control: `run_drift()` expands `DECISION_RC:$?` immediately after the decision block (T:88, :196) — the correct form already exists in this same file. |
| G3 | (controller; subshell `TRACE` loss, re-measured at this base) `/bin/bash -c 'TRACE=parent; ailang() { TRACE="sent"; }; r=$(ailang); printf "capture:[%s]\n" "$TRACE"; ailang; printf "direct:[%s]\n" "$TRACE"'` | `capture:[parent]` then `direct:[sent]`: an assignment made inside command substitution does NOT survive to the parent shell, while the identical direct call does. Positive control = the direct call. |

### Exact causal mapping of the seven baseline failures

Line numbers at base. `T`=`tools/launchd/test_driver_notify.sh`, `P`=`tools/launchd/mission-control.sh`. All rows use P:183-184, the captured direct send inside `_mc_notify`; P:185 accepts the stub's successful exit. The only AILANG observation in the relevant stubs is assignment to `TRACE` (T:40-41 / T:71-72). Capture loses that assignment as measured in G3 (subshell), so each failed predicate requires that lost record. Surviving controls in V2 pass because they read TRACE that survives (log, GH) or negate.

| Exact failing assertion | Lab / production emit | Lost value required; surviving control in V2 |
|---|---|---|
| `fires on both channels (ailang)` — T:114 | `run(pin_block)`; P:1578 | `AILANG:messages send controlplane`; adjacent GH post passes |
| `titled as UNPINNED` — T:116 | `run(pin_block)`; P:1578 | lowercase `driver ran UNPINNED` in AILANG title; GH body capitals differ so does not satisfy |
| `lane fires on ailang` — T:139 | `run(lane_block)`; P:1549 | `AILANG:messages send controlplane`; lane GH and log pass |
| `lane keeps its own title` — T:141 | `run(lane_block)`; P:1549 | lowercase `executor/planner lane degraded`; GH body initial caps differ |
| `drift-a` — T:148 | `run_drift(pinned,170,25,absent)`; P:1596 | ordered AILANG record before GH; DECISION_RC/state/GH/path survive |
| `drift-c` — T:152 | `run_drift(pinned,340,25,170)`; P:1596 | AILANG call record; DECISION_RC and state 340 survive |
| `drift-g` — T:160 | `run_drift(STALE,170,25,absent)`; P:1578 | lowercase `driver ran UNPINNED`; DECISION_RC and GH body survive |

## Goals and High-Impact Decisions

Two goals: (1) make the suite observe real notification calls across shell boundaries and restore its
27 checks incl. the seven red; (2) bound the three D-60 notification paths in production so a hang
cannot stall a retry/spool/wrap-up.

| Decision | Reason | Owner |
|---|---|---|
| Fold production bounding into THIS design (D-60) | Ruling overrules the iter-344 deferral; both halves must land together | Designer |
| Bound the three D-60 paths with `_mc_bounded`; use `NOTIFY_TIMEOUT=30` (env `MISSION_NOTIFY_TIMEOUT`) | Matches the two existing bounded notify sites (P:1817/1820); message-send/gh-comment lands well within 30s; `PROBE_TIMEOUT=120` is for model probes, not messaging (M4) | Designer |
| Convert the fixture's `ailang()`/`gh()` function stubs to executable PATH scripts, prefixed with a leading `env` for the store vars | `_mc_bounded` `( exec "$@" )` cannot reach function stubs and cannot parse `VAR=val` prefixes; PATH scripts + `env` are dependency-free and exercise the real background-subshell bounded path (V4) | Designer |
| Keep `log()` as a function, but unify it onto `MC_TRACE_FILE` | `_mc_notify` calls `log` directly, never via `_mc_bounded`, so it stays a function — but it must append to `MC_TRACE_FILE` and teardown must load the file into `TRACE` before the assertions, or the negative `checkno`s read a LOG-only `$T` and pass vacuously (G1) | Designer |
| Retain the file-backed trace (iter-344) | `_mc_bounded` runs sends in a background subshell; a var-based `TRACE` cannot survive (M3, G3) | Designer |
| Defer the seven non-D-60 call sites to a named queue row | They are episodic/terminal notices (marker-gated or immediate `exit`), not on the steady-state notify path, and each needs its own hanging-stub test (see Deferred) | Designer |

## Solution Design

### Half 1 — Production bounding (per D-60)

1. Add `NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"` next to `PROBE_TIMEOUT` (P:492). 30s matches
   the two established bounded notify sites (M2/M4); overridable for the hang-control test.
2. Wrap the **direct send** in `_mc_notify` (P:184). Replace the unbounded command substitution with
   `_mc_bounded "$NOTIFY_TIMEOUT" env AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT="${AILANG_MESSAGES_PROJECT:-ailang-multivac}" ailang messages send …`, then set `_out="$MC_BOUNDED_OUT"` and test the returned rc. Payload, retry count (3), and sleeps (5/10) are unchanged.
   **Error-tail diagnostic (G5):** on a NORMAL final failure the command returned output, so `_out`
   (from `MC_BOUNDED_OUT`) still holds it and the tail-300 log and the spool row are unchanged. BUT the
   earlier claim is WRONG for `rc=124`: a timed-out command produced no output, so `MC_BOUNDED_OUT` is
   empty and the WARNING would degrade to `FAILED … after 3 attempts:` with nothing after it — the
   exact blindness the `_mc_notify` comment (P:190) exists to prevent. So on `rc=124` the diagnostic is
   SYNTHESISED into the tail-300 log: `timed out after ${NOTIFY_TIMEOUT}s (no output)`. `rc=124` stays
   non-zero → a failed attempt → retried and spooled exactly like today.
3. Wrap the **drain-time send** in `_mc_drain_notices` (P:154) the same way: `_mc_bounded "$NOTIFY_TIMEOUT" env AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=… ailang messages send …`. On `rc=124` the row is treated as failed → re-appended to the spool (unchanged retention path).
4. Wrap the **GitHub notice** in `_mc_notify` (P:198) with `_mc_bounded "$NOTIFY_TIMEOUT" gh issue comment …`. GH is already one-shot (not retried); bounding does not change retry semantics.
5. `env` is required because `_mc_bounded` executes `( exec "$@" )`, and a `$@` prefixed with `VAR=x`
   would make `exec` treat `VAR=x` as the command name. `env` carries the two store vars down to the
   child while preserving the current scoping (the vars are never exported to the driver/coordinator;
   `AILANG_STORAGE` untouched).
6. **Aggregate drain budget (G4) and corrected worst-case latency (M5, G5).** Per-notice worst case,
   measured with polling/grace overhead: AILANG leg = `3×(bound+overhead) + sleeps + SIGKILL grace =
   3×(30+4) + 5 + 10 = 117 s` (G5: ~4s overhead per attempt — a 2s poll plus a further 2s pre-kill
   grace; V5c measured 5s for a 2s bound), NOT 105 s; per notice with the GH leg = `117 + (30+4) =
   ~151 s`, NOT ~135 s. This hard per-notice bound already beats today's unbounded form, but a DRAIN of
   `N` hanging rows must not cost `N × (bound + overhead)` inside the one preflight phase whose only
   backstop is the whole-slot `HARD_TIMEOUT` (G4). So the drain gets an explicit AGGREGATE budget
   `DRAIN_BUDGET` (env `MISSION_DRAIN_BUDGET`, default 90 s), justified by verification row G4: the
   preflight has NO phase-specific deadline — only `HARD_TIMEOUT=6 h` (P:795) — and the stall watchdog
   cannot rescue it (no controller child exists yet; slow-but-progressing is not "no progress").
   Before EACH drain attempt, require enough REMAINING budget for `NOTIFY_TIMEOUT` + termination
   overhead (~2 s SIGKILL grace); if insufficient, STOP draining, PRESERVE all unattempted rows in the
   spool, and emit an explicit DEFERRED-drain diagnostic (`notice spool: deferred <k> row(s), aggregate
   budget <BUDGET>s exhausted`) rather than starting a send certain to be cut off. Individual attempts
   keep reusing `_mc_bounded`; failed rows are re-appended exactly as today. A large-backlog acceptance
   case (hanging send stubs over an N-row spool) proves the whole drain returns within `DRAIN_BUDGET`
   and preserves both failed and unattempted rows; the mutation that removes the aggregate guard kills
   that assertion.

### Half 2 — Test-observation repair (retained, incl. M3 resolution)

1. In the send-capable `run()` and `run_drift()` labs, write the fixture stubs as executable scripts in
   a suite-owned temp `bin/` dir and prepend it to an **exported `PATH`** so `_mc_bounded`'s background
   `exec` resolves the fake `ailang` and `gh` instead of the real binaries (M3). `log()` stays a function (called directly, never through `_mc_bounded`) but, per G1, it also
   appends its records to `MC_TRACE_FILE`.
2. Each `ailang`/`gh` PATH script appends one ordered `AILANG:`/`GH:` record to a **per-arm trace FILE**
   (path via exported `MC_TRACE_FILE`), preserving every existing record prefix/argument form the checks
   consume. The stub also emits the env values (via a fake stdout line captured into `MC_BOUNDED_OUT`)
   so store/project assertions read `AILANG_MESSAGES_STORE=gcp` and the default `ailang-multivac`
   reaching the child (V4). This is where the subshell-lost-`TRACE` fix lands: `_mc_bounded` runs sends
   in a background subshell (M3), so only a file survives. `log()` appends to this SAME file, and the
   `run()`/`run_drift()` teardown loads `MC_TRACE_FILE` into the `TRACE` variable before the 27
   assertions evaluate, so positive and negative checks both read one unified medium (G1).
3. Add the sleep stub, retry/file-counter for attempt state, error-tail, spool, and drain coverage
   (direct-send retry/spool, `_mc_drain_notices` extraction via the guarded awk pattern) exactly as the
   iter-344 Solution Design specified — the drain/extraction tests are unchanged in intent.
4. Restore all seven assertions; add the production-bound assertions (next section). Capture the
   block's actual status immediately after sourcing (the `RC:$?`-after-`printf` defect, G2).
5. Wire check: exactly one top-level `_mc_drain_notices` invocation after the pin-decision end marker.

## Conflict Surface

| Surface | Occupant to preserve | Resolution / verification |
|---|---|---|
| **exec-vs-function collision (M3)** | `_mc_bounded` `( exec "$@" )` bypasses functions | **PATH executable stubs** in the fixture (Half 2); `env` prefix in production (Half 1, step 5). Verified V4. Alternatives rejected: `export -f` (unreliable across `exec` in Bash 3.2; changes the shared helper), watcher-on-command-substitution (duplicates helper, does not fix function-bypass). |
| **split-trace / vacuous checkno (G1)** | `ailang`/`gh` records to `MC_TRACE_FILE` but `$T` comes from `log()`-only `TRACE` | Unify the medium: `log()` appends to `MC_TRACE_FILE`; teardown loads the file into `TRACE` before the 27 assertions; a positive-control arm forces the same `no ailang call`/`lane SILENT when healthy` checkno to FAIL (see Acceptance / Mutation). |
| **error-tail / `MC_BOUNDED_OUT` (M3)** | `_mc_notify` warns with last-300-bytes on send failure (assertions + spool depend on it) | `_out="$MC_BOUNDED_OUT"` after the bounded call; rc=124 → failure arm unchanged. Error-tail content is captured in the failing-send lab. |
| **env-prefix through `exec`** | per-command store/project scoping; `AILANG_STORAGE` untouched | leading `env` argument; verified reaching child (V4); scoping preserved (never exported). |
| **worst-case latency (M5)** | three retries + 5/10s sleeps, non-aborting | bounded ≤ ~151s/notice (corrected, G5); bounded drain per-row AND per-aggregate (step 6); budget-acceptable (see Design step 6). |
| **seven out-of-D-60 sites (M2)** | refusal P:1470/1473, model-change P:1486, post-record P:1835/1838, rc-fail P:1852/1855 | **Deferred to named queue row** with reasons (see Deferred); not left silent. |
| Captured stdout/stderr | `_out` (or `MC_BOUNDED_OUT`) holds provider output | separate file spy + diagnostic-tail assertions |
| Retry state | each bounded send forks a subshell | file counter; exact attempt/backoff checks |
| Spool filesystem | failures survive next drain | exact content/count, cross-namespace controls |
| Go-less Bash 3.2 CI | `/bin/bash`, no new deps | `/bin/bash`, `mktemp -d`, no `declare -A`/`timeout` |

Fixtures still work: pin/lane healthy/degraded, drift-a..j; all 54 `test_pin_root.sh`; every other
suite in `make test-launchd-drivers`. Intentional production change is bounded calls + `env` transport
only; no payload, retry count, message-store semantics, policy, or language/runtime change.

## Acceptance Criteria and Test Plan

**RED at base (non-vacuity controls):**
- Test half: `bash tools/launchd/test_driver_notify.sh` is RED at base (M1: 20 passed / 7 failed,
  rc=1) — the other twenty checks already pass, so the suite is not trivially all-fail and the seven
  named rows are the exact regression set.
- **Vacuous-checkno guard (unified medium, G1):** a positive-control arm in which the AILANG call
  genuinely fires (a live `ailang` PATH script) — the SAME `no ailang call`/`lane SILENT when healthy`
  checkno assertions MUST FAIL against it. This is how a vacuous `checkno` is DETECTED: it proves those
  negative assertions read `MC_TRACE_FILE` (loaded into `TRACE`), not an empty half of a split trace.
- Production half, **hang-cutoff control (separate from the test half):** a fixture `ailang`/`gh` stub
  that never returns (no output, `while :; do sleep 1; done`), with `MISSION_NOTIFY_TIMEOUT=2`, must be
  cut off at the bound. The bounded-notify lab runs the hang-stub through the real `_mc_bounded` and
  asserts the call returns within a **7 s** budget (`bound 2 s + ~2 s poll granularity + 2 s SIGKILL grace`, measured 5 s in V5c, +2 s headroom) with rc=124, and that the
  notify's failure arm spools a row after three retries. **This assertion FAILS if the `_mc_bounded`
  wrapper/timeout is removed** (the hang would not be cut off). An outer test-level date-loop deadline
  guards the lab so a removed bound fails cleanly rather than hanging CI.

**GREEN after implementation:**
- [ ] Original 27 assertions pass, including all seven restored rows; record new exact total.
- [ ] Negative `checkno` controls read the unified medium: `no ailang call` and `lane SILENT when
      healthy` FAIL on the genuine-fire positive-control arm and pass only when AILANG truly did not
      fire (G1).
- [ ] All `make test-launchd-drivers` suites pass; record exit code and elapsed time.
- [ ] New direct-send/store/retry/error-tail/spool/drain/wiring cases pass; every spy is local (file).
- [ ] **Production bound asserted:** for each of the three D-60 paths (direct, drain, GH), a
  never-returning stub with `MISSION_NOTIFY_TIMEOUT=2` is cut off at the bound (rc=124, measured 5 s — assert ≤7 s, V5c) and the
  failure path (spool / WARNING) still executes. Must fail if the bound is removed.
- [ ] **Synthesised timeout diagnostic (G5):** a bounded direct send whose command times out (rc=124,
      no output — ARM B) yields a WARNING whose tail reads `timed out after ${NOTIFY_TIMEOUT}s (no
      output)`, NOT an empty `FAILED … after 3 attempts:`; the assertion FAILS if the synthesis is
      reverted to the raw (empty) `MC_BOUNDED_OUT` tail.
- [ ] **Aggregate drain budget (G4):** a large-backlog drain against hanging send stubs returns within
      the aggregate `DRAIN_BUDGET`, emits the deferred-drain diagnostic, and preserves BOTH the failed
      and the unattempted rows; FAILS if the aggregate guard is removed.
- [ ] Failed send and failed GH stay non-aborting, measured from the actual block status.
- [ ] Mutation controls below produce the named failures; clean copies parse; all applied mutations
  proven absent afterward; driver hash equals its initial value after re-run.

## Mutation Table

Run each mutation in a bounded temp copy; keep the production tree untouched; prove each landed by
diff/hash and `bash -n`.

| Mutant | Specific assertion it must kill |
|---|---|
| Remove `_mc_bounded`/`NOTIFY_TIMEOUT` on the direct send (revert to unbounded command substitution) | `production: hanging direct send is cut off at the bound` (hang not cut off; outer guard trips) |
| Remove `_mc_bounded` on the drain-time send | `production: hanging drain send is cut off and row retained` |
| Remove `_mc_bounded` on the GH notice | `production: hanging gh comment is cut off and warns` |
| Replace `env AILANG_MESSAGES_STORE=gcp …` with no store / `local` in the direct send | `direct send reaches child with store=gcp` (env/store assertion fails) |
| Keep the in-shell `ailang()` function stub instead of the PATH script | `fires on both channels (ailang)` AND `titled as UNPINNED` AND `lane fires on ailang` AND `lane keeps its own title` AND `drift-a` AND `drift-c` AND `drift-g` — all fail because `exec` cannot reach a function (M3 positive-proof) |
| Revert to a split trace: `ailang`/`gh` on `MC_TRACE_FILE`, `log()` on the `TRACE` var, and skip loading the file into `TRACE` in the teardown | `no ailang call` AND `lane SILENT when healthy` pass vacuously despite a genuinely fired AILANG record; the positive-control arm is the tripwire that flags the regression (G1) |
| Suppress re-appending a failed drain row | `drain: retained-row/recovery` assertion |
| Delete only the top-level drain invocation | `wiring: exactly one preflight drain call` |
| Replace file counter with a shell var for retry state | `retry: attempt-2/3 recorded across subshells` |
| Remove the aggregate drain-budget guard (`DRAIN_BUDGET` remaining-budget check before each row) | `drain: whole drain returns within aggregate budget and preserves unattempted rows` (a large-backlog hang exceeds budget and rows are neither attempted-bounded nor preserved) |
| Revert the `rc=124` SYNTHESISED diagnostic to the raw (possibly empty) `MC_BOUNDED_OUT` tail | `synthesised timeout diagnostic` — the `timed out after ${NOTIFY_TIMEOUT}s (no output)` text vanishes (G5, ARM B) |

Record exact outcomes, not predictions; re-run the clean suite; verify the driver hash is unchanged.

## Controller First-Party Rows G4, G5 (round-3 evidence)

Controller-measured at base `81abc956d`; production tree untouched. G4 justifies the aggregate drain
budget (Half 1 step 6); G5 is the load-bearing evidence for the `rc=124` synthesised diagnostic (Half 1
step 2) and the corrected latency arithmetic.

**G4 — the drain sits in the one phase with no preflight-specific deadline (controller, verified):**
- `_mc_drain_notices` is defined at `tools/launchd/mission-control.sh:145` and has exactly ONE top-level
  call, at `:972`, immediately after the `# --- DRIVER PIN DECISION END ---` marker — i.e. in the
  driver's PREFLIGHT, before the controller session starts.
- The ONLY deadline covering that phase is `HARD_TIMEOUT="${MISSION_TIMEOUT:-21600}"` (P:795 — 6 h for
  the whole slot). There is NO preflight-specific deadline at all.
- The stall watchdog cannot rescue it: `STALL_GRACE=2400` / `STALL_CHILD_AGE=2400` (`:802-803`); P:289
  *"stall watchdog has NO progress instrument … early kill DISABLED … HARD_TIMEOUT still applies"* —
  the preflight's situation (no controller child yet); slow-but-progressing is not "no progress".
- `notice-spool` has exactly two references (`:147` read, `:195` append). **No size cap, no row cap, no
  rotation** — unbounded by construction.
- **Failure mode (concrete):** an N-row spool of hanging sends costs `N × (bound + overhead)` inside the
  one phase whose only backstop kills the ENTIRE fire six hours later. Hence the aggregate budget.

**G5 — wrapping `_mc_notify`'s direct send (P:184) and GH notice (P:198) preserves retry/spool/error-tail
semantics under the real control flow (controller, verified).** Method: extracted `_mc_bounded` +
`_mc_notify` via the guarded `awk '/^_mc_notify\(\) \{/,/^\}/'` pattern, substitution on the EXTRACTED
copy only, PATH `ailang`/`gh` stubs + temp `STATE_DIR` + `log()` capture, real body run end to end. Two
arms:

- **ARM A** — `ailang` stub exits 1 immediately, `NOTIFY_TIMEOUT=30`: ELAPSED=24s · ailang attempts=**3**
  · gh calls=**1** · spool rows=**1**; spool row `2026-09-07T12:59:09Z<TAB>Test title<TAB>line one line
  two` (multiline body flattened); log `WARNING: testlabel notice FAILED to send via ailang messages
  after 3 attempts: stdout-noise SIMULATED-SEND-ERROR store=<gcp> proj=<ailang-multivac>` → (a) all
  three retries execute ✓, (b) error-tail carries `MC_BOUNDED_OUT` incl. the env reaching the child ✓,
  (c) spool row written on final failure ✓, (d) rc=124 flows into the retry/spool arm identically to a
  non-zero send rc ✓.
- **ARM B** — `ailang` stub never returns, `NOTIFY_TIMEOUT=2`: ELAPSED=28s · attempts=**3** · gh calls=**1**
  · spool rows=**1**; log = `WARNING: testlabel notice FAILED to send via ailang messages after 3
  attempts: ` — **the error tail is EMPTY**.

**THE DEFECT:** on a timeout the command produced no output, so `MC_BOUNDED_OUT` is empty and the WARNING
degrades to `FAILED … after 3 attempts:` with nothing after it — today's unbounded form cannot hit this,
because it only reaches the failure arm when the command actually returned and said something.
`_mc_notify`'s own comment (`P:190`) keeps the reason precisely because discarding the error is "the same
blindness that made an Anthropic rc=2 unexplainable for a whole day" — so the bounded substitution
silently reintroduces that blindness in exactly the case the bound was added for, and `rc=124` therefore
SYNTHESISES the `timed out after ${NOTIFY_TIMEOUT}s (no output)` diagnostic.

**Cosmetic (recorded, not designed around):** each cutoff prints `…/bounded.sh: line 1: <pid> Terminated:
15 ( exec "$@" ) > "$out_f" 2>&1` to stderr — job-control noise, three lines per timed-out notice.

## Timeline, Risks, and Deferred Decisions

One milestone: ~40m production bounding + env transport, ~40m fixture PATH-stub/trace migration, ~40m
new cases, ~40m mutations (incl. hang controls), ~40m full verification and evaluation buffer.

Risks: env passthrough regressing (guarded by store/project inside-stub assertions, V4); a hang-control
lab hanging CI after a bound is removed (guarded by an outer test-level deadline); accidentally vacuous
mutants (named assertions in the table). Sleep timing is asserted, not slept.

**Deferred — named queue row `M-LAUNCHD-NOTIFY-REMAINING-BOUNDING`** (controller adds to mission queue),
NOT silent: the seven non-D-60 call sites (M2) are out of scope for THIS milestone for stated reasons —
they are episodic/terminal, not the per-fire steady-state notify path:
- Refusal P:1470/1473 — guarded by `$BLOCKED_FILE` and immediately followed by `exit 1`; a hung send
  delays an intentional shutdown, not a continuing loop.
- Model-change P:1486 — fires only on controller-model transitions (rare), guarded by model-change
  logging; a hang delays only the announce.
- Post-record P:1835/1838 — fires only when rc≠0 AND a mission-log record landed (rare terminal case).
- rc-fail P:1852/1855 — episode-gated on rc value; at most once per identical-failure episode.
Each needs its own hanging-stub test and adds no CI gate to enforce it (none is exercised by the red
notify suite); bounding them is a coherent follow-up. This design does not claim them fixed.

## Axiom Compliance

Harness-scoped scoring; no language-support claim.

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | Stable observations from file state, independent of subshell variable lifetime |
| A2 Replayability | 0 | Existing production replay contract preserved |
| A3 Effect Legibility | +1 | Tests distinguish captured output from observable channel calls; bound rc=124 is legible |
| A4 Explicit Authority | 0 | All external channels remain stubbed (now via PATH scripts) |
| A5 Bounded Verification | **+1** | Production notification calls bounded (D-60); test runs have explicit budgets; hang-cutoff asserted |
| A6 Safe Concurrency | 0 | Per-arm isolation retained; `_mc_bounded` already background-safe |
| A7 Machines First | +1 | CI identifies broken behavior, not hidden spy state; bound regression is caught |
| A8 Minimal Syntax | 0 | No language change |
| A9 Cost Visibility | 0 | No billing change |
| A10 Composability | 0 | Existing test target retained |
| A11 Structured Failure | 0 | Failure contract preserved and checked (error-tail, spool) |
| A12 System Boundary | 0 | Notification payloads, retries, policy unchanged |

**Net +4** (A5 moved 0→+1 via D-60's production bounding). Hard gates A1/A3/A4/A7 have no negative
score.

## Quorum Verification Log

Round 1 of independent review BLOCKED this revision. Both objections were reproduced first-party by the
controller at base `81abc956d`; neither disputes the design direction, so this revision fixes both and
stops.

- **gemini-3-1-pro** — REJECT (split-brain trace). Surfaced on the Solution Design Half-2 (log-vs-file
  trace split), the Goals/Decisions `log()`-stub row, the Conflict Surface, the Acceptance Criteria, and
  the Verification Log (G1). **Response:** applied the reviewer's fix verbatim — `log()` now also appends
  to `MC_TRACE_FILE`, teardown loads the file into `TRACE` before the 27 assertions, the stale
  `log()`-stub decision row was replaced, and a vacuous-`checkno` positive-control detection arm was added
  to the Acceptance Criteria plus a matching Mutation Table row.
- **oc-glm-5-2** — REJECT (unverified V13/V14 dependency). Surfaced on the Problem Statement, the causal
  mapping, and Solution Design Half-2 step 4. **Response:** added controller-attributed first-party rows
  G2 (RC-after-`printf` defect) and G3 (subshell `TRACE` loss) measured at this base, and removed every
  remaining V13/V14/iter-344-as-evidence citation, replacing each with the new row IDs.
- **gpt6-astra** — ABSENT on budget; no verdict recorded.

**Round 2** — **gemini-3-1-pro**: **PASS** (flipped from round-1 REJECT; its round-1 fix worked and the
revision now passes). **gpt6-astra**: **REJECT** on the drain-budget surface (no aggregate drain bound;
a spool that is unbounded by construction drains inside a phase with no preflight-specific deadline —
see G4). **oc-glm-5-2**: **REJECT** on the real-call-site evidence surface (V4/V5c ran only a standalone
PATH-stub reproduction through a verbatim `_mc_bounded` copy, never the real `_mc_notify` body — see
G5). Note: glm's round-2 response failed to parse and was re-run alone at a raised cap.

**Round 3 — this narrow refinement**, applied under the mission's narrow-refinement carve-out: the two
reviewer-authored fixes were applied VERBATIM — astra's explicit AGGREGATE drain budget + before-each-row
remaining-budget check + stop-and-preserve-unattempted behaviour + deferred-drain diagnostic (+ row G4),
and glm's real-call-site evidence (row G5, both arms), the corrected error-tail claim, the synthesised
`rc=124` diagnostic, and the corrected latency arithmetic. The mechanism, the two-half structure, the
`env`-through-`exec` transport, the PATH-stub fixture migration, the unified `MC_TRACE_FILE` medium, and
rows M1–M5/V1–V6/V5c/G1–G3 are unchanged. **No further quorum round follows** — this revision routes
directly to the sprint planner.

## Related Documents and Handoff

- [Driver pin rollout](../m-driver-pin-rollout.md): caller-emits contract; not reopened.
- [Mission loop workbench](../v0_36_0/m-mission-loop-workbench.md): configuration registry, distinct.
- [Motoko refusal sprint plan](../m-motoko-discovery-arm-discriminating-refusal-sprint-plan.md): neural
  match 0.45; mutation discipline, different instrument.
- Historical commits `63a0d2b32` (production capture change) and `9f267cf1f` (prior fixture repair).

Design author changes only this document and makes no Git writes. Route to planner, executor, and
independent evaluator after quorum under D-60, which prefers this reliability repair early in the week.
Re-quorum must run against this revision; this document does not self-approve.

## Attended merge repair, 2026-09-07

While merging mission-runtime PR #1082, reproduced the inherited 20/7 failure from
GitHub job 101761650374. Applied the narrow observation repair: ordered file-backed
spies in both send-capable labs, actual block return-code capture, and a no-wait sleep
stub. All original 27 assertions pass; independent review PASS. Production driver is
unchanged. This resolves the seven observed failures but does not claim completion
of the broader retry/drain/environment coverage planned above.

**Superseded by iteration 347, and the sequence matters.** That attended repair landed on `dev` inside
PR #1082 at 14:15Z while V1 iteration 347 was mid-flight on the same item under human ruling **D-60**,
and the two are not composable: it keeps `ailang()`/`gh()` as in-shell FUNCTION stubs, which
`_mc_bounded` cannot reach at all (`( exec "$@" )` bypasses functions — row M3). So the fixture work in
this sprint SUPERSEDES it rather than duplicating it: PATH-executable stubs are a precondition of the
production bounding D-60 mandates, not a stylistic preference. Its two substantive additions are
carried forward: the `sleep` stub and file-backed `run_drift` spies. The same PR (`063df9917`) also
repaired the nine `test_mission_heartbeat.sh` arms this iteration had measured failing at base
`81abc956d`, so `make test-launchd-drivers` is rc=0 on `dev` — see the correction in the iteration-347
log entry.
