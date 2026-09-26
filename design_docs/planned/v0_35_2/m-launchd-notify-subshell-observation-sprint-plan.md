# Sprint Plan — M-LAUNCHD-NOTIFY-SUBSHELL-OBSERVATION (V1 mission iteration 347)

**Status**: Planned, awaiting "execute sprint". Plans the APPROVED design
[`m-launchd-notify-subshell-observation.md`](m-launchd-notify-subshell-observation.md) (D-60 revision,
quorum round 3). No scope is added or removed here; every step traces to a named design section.
**Base**: `027bcb022` on branch `sprint/v1-iter347-launchd-notify-bounded` (= `origin/dev` `81abc956d`
+ approved design revision). **Baseline measured this planning session**:
`bash tools/launchd/test_driver_notify.sh` → `20 passed, 7 failed`, rc=1 (M1/V2 reproduced);
`/bin/bash` = 3.2.57(1) arm64.
**Machine-readable companion**: `sprint_v1_iter347_launchd_notify.json` (same milestones).

## Sequencing rationale — why M1 MUST land before M2 (the collision)

`_mc_bounded` (P:496) runs `( exec "$@" )` in a background subshell. `exec` resolves only external
binaries — measured M3: `( exec f )` on a shell function → `exec: f: not found`, rc=127. The current
fixture stubs `ailang()`/`gh()` as **shell functions**.

Therefore, if the production sends are wrapped in `_mc_bounded` **before** the fixture is migrated:

1. Every notify arm would `exec` the **real** `ailang`/`gh` on PATH — contacting prod Firestore and
   GitHub from CI (forbidden: tests must not touch Firestore/GitHub/providers/launchd), and
2. the stub-observation assertions would fail for the wrong reason (rc=127 / real-binary output), i.e.
   the suite **silently stops exercising the stubs** while still printing PASS/FAIL lines.

Conversely, the fixture migration is safe against **unbounded** production: the current direct send
`_out=$(ailang …)` is a command substitution, which resolves PATH executables fine. So the only
ordering in which the suite never silently stops exercising the stubs is:

**M1 fixture-first (PATH executable stubs + unified trace medium) → M2 production bounding →
M3 aggregate drain budget → M4 mutation sweep & full verification.**

M2 before M3 because M3's remaining-budget arithmetic is defined in terms of `NOTIFY_TIMEOUT`,
which M2 introduces; M3's acceptance lab reuses M2's bounded drain send.

## Budget

Flash-class executor, 30-minute cap. Estimate: M1 10m · M2 9m · M3 5m · M4 5m = **29m**. Suite wall
time grows by ~90–120s of real sleeps (retry back-off 5+10s per failing arm; hang labs at
`NOTIFY_TIMEOUT=2` cost ~5–7s per cutoff, ~28s for the full 3-retry timeout arm — ARM A/B measured).
Fits, tightly. **Cut order if the run is short:** (1) trim M4's mutation sweep to the three P0
mutants (direct-send bound removal, split-trace, drain-budget removal) and defer the rest to the
evaluator; (2) shrink M3's backlog lab from 5 rows to 3. Never cut M1/M2 assertion coverage.

## Global constraints (binding on the executor)

- Bash 3.2.57 only: no `declare -A`, no `${v,,}`, no GNU `timeout`, no new dependencies, no Go.
- No network: every lab resolves `ailang`/`gh` to the suite-owned PATH stubs; no Firestore, GitHub,
  model provider, or launchd contact. The stub `bin/` dir contains exactly `ailang` and `gh`.
- Do NOT stub `sleep`: `_mc_bounded`'s poll loop (`sleep 2`) and SIGKILL grace (`sleep 2`) are part of
  the measured cutoff arithmetic (V5c: bound 2s → cutoff 5s). Stubbing `sleep` would invalidate the
  7-second acceptance budget. This resolves the design's "sleep timing is asserted, not slept" risk
  note in favour of real sleeps with generous outer deadlines.
- The executor makes **NO git write operations** (no add/commit/stash/checkout); `git diff`/`git
  status` reads are fine. The controller commits.
- Record ACTUAL pass/fail counts and timings at each boundary; the counts below are predictions from
  the design's causal mapping, not measurements.

---

## M1 — Fixture observation repair: PATH stubs, unified `MC_TRACE_FILE`, RC capture

**Files**: `tools/launchd/test_driver_notify.sh` only. Production is untouched in this milestone.

**Design trace**: Half 2 steps 1–4; Conflict Surface rows 1–2; G1/G2/G3; mutation rows 5, 6, 9.

**Steps**:
1. Add a suite-owned `bin/` under `$LAB` containing executable `ailang` and `gh` scripts. Each
   appends one ordered `AILANG:`/`GH:` record (same argument forms the checks consume today) to
   `"$MC_TRACE_FILE"`; each honours `AILANG_RC`/`GH_RC`; the `ailang` stub also prints
   `store=<$AILANG_MESSAGES_STORE> proj=<$AILANG_MESSAGES_PROJECT>` on stdout so the child's env is
   observable through captured output (V4). Prepend `bin/` to an **exported** `PATH` and export
   `MC_TRACE_FILE` (fresh per arm) in `run()`, `run_drift()`, and the inline drift-j lab.
2. `log()` stays a function but appends its `LOG:` record to the SAME `"$MC_TRACE_FILE"` (G1 unified
   medium). Teardown in each lab loads the file into `TRACE` (`TRACE="$(cat "$MC_TRACE_FILE")"`)
   AFTER sourcing the block and BEFORE emitting the trace the 27 assertions read.
3. Fix the RC-after-`printf` defect (G2, T:52-55): capture `rc=$?` immediately after `. "$2"`, then
   emit the trace, then `echo "RC:$rc"`. (`run_drift`'s `DECISION_RC:$?` at T:88/:196 is already
   correct — leave it.)
4. New assertions (3):
   - `retry: attempts recorded across subshells (file counter)` — the `ailang` PATH stub increments a
     counter FILE; with `AILANG_RC=1`, assert 3 recorded attempts after the pin_block arm.
   - `RC capture: block failure visible, not masked by printf` — an arm whose sourced block aborts
     under `set -u` (drop the `STATE_DIR` assignment) must record `RC:1`, not `RC:0`.
   - `positive control: unified medium shows genuine AILANG fire (checkno would FAIL)` — on a firing
     pin_block arm, assert `$T` CONTAINS `AILANG:`; this is the tripwire proving the
     `no ailang call` / `lane SILENT when healthy` checknos read the unified medium (G1).

**Boundary expectation**: `==== 30 passed, 0 failed ====`, rc=0 (27 restored incl. the seven red
rows + 3 new; executor records the actual count). Red-at-base is the M1 measurement itself
(20/7, rc=1) — no separate red run needed.

**Acceptance** (red at milestone base → green after):
- `bash tools/launchd/test_driver_notify.sh` → base: `20 passed, 7 failed`, rc=1 → after: `N passed,
  0 failed`, rc=0 (N≈30).
- `bash tools/launchd/test_driver_notify.sh | grep -c "PASS: positive control: unified medium"` →
  base: `0` → after: `1`.
- `bash tools/launchd/test_driver_notify.sh | grep -c "PASS: RC capture"` → base: `0` → after: `1`.
- `make test-launchd-drivers` → base: rc≠0 (notify suite red; 54 pin-root checks pass first) →
  after: rc=0. (Red at base is carried by the same seven rows — declared: this command's redness is
  fully explained by M1's diff and nothing else.)

**Mutations (each anchored to M1's own diff — reverting M1 while keeping everything else re-reds
exactly these)**:
| Mutation | Kills assertion |
|---|---|
| Keep in-shell `ailang()`/`gh()` function stubs instead of PATH scripts | `fires on both channels (ailang)` |
| Split trace: `log()` stays on the `TRACE` var; teardown does not load `MC_TRACE_FILE` | `positive control: unified medium shows genuine AILANG fire (checkno would FAIL)` |
| Move `RC:$?` capture back after the `printf` | `RC capture: block failure visible, not masked by printf` |
| Replace the retry file counter with a shell variable | `retry: attempts recorded across subshells (file counter)` |

---

## M2 — Production bounding of the three D-60 paths + synthesised rc=124 diagnostic

**Files**: `tools/launchd/mission-control.sh` (P:492 `NOTIFY_TIMEOUT`, P:184 direct send, P:154
drain send, P:198 GH notice) and `tools/launchd/test_driver_notify.sh` (new labs/assertions FIRST).

**Design trace**: Half 1 steps 1–5; G5 (ARM A/B); Conflict Surface rows 3–5; mutation rows 1–4, 7, 8, 11.

**Steps (order matters WITHIN the milestone — tests first, no git available for red/green staging)**:
1. **Fixture first.** Extract `_mc_bounded` and `_mc_drain_notices` into the lab via the guarded awk
   patterns (`awk '/^_mc_bounded\(\) \{/,/^\}/'`, `awk '/^_mc_drain_notices\(\) \{/,/^\}/'`, with the
   same empty-extraction FATAL guard as the existing five). Add a never-returning stub variant
   (`while :; do sleep 1; done`, selected by env `STUB_HANG=1`) and the new arms below. **Run the
   suite now and record the RED**: each new production-bound assertion fails against unwrapped
   production — the direct-send hang lab trips its outer watchdog (see 3) and fails cleanly.
2. **Production.** Add `NOTIFY_TIMEOUT="${MISSION_NOTIFY_TIMEOUT:-30}"` beside `PROBE_TIMEOUT`
   (P:492). Wrap:
   - direct send (P:184): `_mc_bounded "$NOTIFY_TIMEOUT" env AILANG_MESSAGES_STORE=gcp
     AILANG_MESSAGES_PROJECT="${AILANG_MESSAGES_PROJECT:-ailang-multivac}" ailang messages send …`;
     capture `_rc=$?; _out="$MC_BOUNDED_OUT"`; success arm unchanged (3 attempts, sleeps 5/10).
   - **Synthesised diagnostic (G5 ARM B)**: in the failure arm, if `_rc -eq 124`, set
     `_out="timed out after ${NOTIFY_TIMEOUT}s (no output)"` BEFORE the tail-300 WARNING — a timed-out
     command produced no output, so the raw tail is empty.
   - drain send (P:154): same `_mc_bounded "$NOTIFY_TIMEOUT" env … ailang messages send …` form;
     rc=124 → row re-appended (retention path unchanged).
   - GH notice (P:198): `_mc_bounded "$NOTIFY_TIMEOUT" gh issue comment …` (one-shot semantics
     unchanged; the helper already captures output, so the `>/dev/null 2>&1` goes away).
3. **New arms/assertions** (all with `MISSION_NOTIFY_TIMEOUT=2`):
   - `production: hanging direct send is cut off at the bound` — single-shot
     `_mc_bounded 2 env … ailang …` against the hanging stub: assert rc=124 AND elapsed ≤ 7s
     (V5c: 5s = 2 bound + ~2 poll + 2 grace, +2 headroom). **Outer test-level deadline**: run the lab
     body in a background subshell, poll `kill -0` on a `date +%s` loop, `kill -9` at 15s and fail the
     assertion — a removed bound fails cleanly instead of hanging CI.
   - `production: direct send retries 3x then spools on timeout` — full `_mc_notify` with hanging
     `ailang` stub: outer deadline 60s (ARM B measured 28s); assert attempts=3 (file counter), spool
     rows=1, GH still called once.
   - `synthesised timeout diagnostic` — same arm: WARNING contains `timed out after 2s (no output)`,
     and does NOT end at a bare `after 3 attempts:`.
   - `production: hanging drain send is cut off and row retained` — one-row spool + hanging stub:
     drain returns (outer deadline 15s), the row is re-appended.
   - `production: hanging gh comment is cut off and warns` — `ailang` healthy, `gh` hanging: bounded
     cutoff ≤7s, WARNING posted, block still exits 0.
   - Guard (declared green-at-base/green-after, retained for its mutation row): `direct send reaches
     child with store=gcp` — failure-arm WARNING contains `store=<gcp> proj=<ailang-multivac>`.
     **This measures nothing as red→green evidence** (the unbounded `VAR=val` prefix also delivers
     the vars); it exists so the env-prefix mutation has a named victim.
   - Guard (green/green, regression tripwire only): `wiring: exactly one preflight drain call` —
     `awk '/^# --- DRIVER PIN DECISION END ---/,0' "$DRV" | grep -c '^_mc_drain_notices$'` = 1.
4. Run the suite; record GREEN.

**Boundary expectation**: `==== 37 passed, 0 failed ====`, rc=0 (30 + 7 new, of which 2 are declared
guards; executor records the actual).

**Acceptance** (red at M2 base = end of M1 → green after):
- After step 1 (fixture only): `bash tools/launchd/test_driver_notify.sh` → new `production:` arms
  FAIL (record exact count) → after step 2: `0 failed`, rc=0.
- `bash tools/launchd/test_driver_notify.sh | grep -c "PASS: synthesised timeout diagnostic"` →
  M2-base: `0` → after: `1`.
- `grep -c '_mc_bounded "$NOTIFY_TIMEOUT" env' tools/launchd/mission-control.sh` → M2-base: `0` →
  after: `2` (direct + drain).
- `grep -c '_mc_bounded "$NOTIFY_TIMEOUT" gh issue comment' tools/launchd/mission-control.sh` →
  M2-base: `0` → after: `1`.
- `bash -n tools/launchd/mission-control.sh tools/launchd/test_driver_notify.sh` → rc=0 (green/green
  parse guard; declared non-anchoring).

**Mutations (anchored to M2's own diff: revert only the P:154/184/198 wraps + synthesis while
keeping M1, and each named assertion re-fails)**:
| Mutation | Kills assertion |
|---|---|
| Remove `_mc_bounded`/`NOTIFY_TIMEOUT` on the direct send | `production: hanging direct send is cut off at the bound` |
| Remove `_mc_bounded` on the drain-time send | `production: hanging drain send is cut off and row retained` |
| Remove `_mc_bounded` on the GH notice | `production: hanging gh comment is cut off and warns` |
| Drop the `env AILANG_MESSAGES_STORE=gcp …` prefix from the bounded direct send | `direct send reaches child with store=gcp` |
| Revert the rc=124 synthesised diagnostic to the raw (empty) `MC_BOUNDED_OUT` tail | `synthesised timeout diagnostic` |

---

## M3 — Aggregate drain budget `DRAIN_BUDGET`

**Files**: `tools/launchd/mission-control.sh` (`_mc_drain_notices`, P:145-165) and
`tools/launchd/test_driver_notify.sh` (large-backlog lab).

**Design trace**: Half 1 step 6; G4; mutation row 10.

**Steps**:
1. **Fixture first.** Add the backlog arm: 5-row spool, hanging `ailang` stub,
   `MISSION_NOTIFY_TIMEOUT=2`, `MISSION_DRAIN_BUDGET=8`. Run → **record RED**: with per-call bounds
   but no aggregate guard the drain attempts all 5 rows (~5s each ≈ 25s > 8s budget) and emits no
   deferred diagnostic.
2. **Production.** In `_mc_drain_notices`: `DRAIN_BUDGET="${MISSION_DRAIN_BUDGET:-90}"`; record
   `drain_start=$(date +%s)` after the spool is claimed; **before each row's send** require
   `remaining = DRAIN_BUDGET - ($(date +%s) - drain_start)` ≥ `NOTIFY_TIMEOUT + 2`; if insufficient —
   STOP, append the current and ALL unattempted rows back to the spool unchanged, log
   `notice spool: deferred <k> row(s), aggregate budget <BUDGET>s exhausted`, and return 0. Failed
   rows keep the existing re-append path.
3. New assertions:
   - `drain: whole drain returns within aggregate budget and preserves unattempted rows` — elapsed
     ≤ 15s (outer deadline 30s), spool afterwards holds 5 rows (1 failed re-appended + 4
     unattempted), attempt counter = 1.
   - `drain: deferred-drain diagnostic names deferred row count` — log contains
     `deferred 4 row(s), aggregate budget 8s exhausted`.
4. Run the suite; record GREEN.

**Boundary expectation**: `==== 39 passed, 0 failed ====`, rc=0 (37 + 2; executor records actual).

**Acceptance** (red at M3 base = end of M2 → green after):
- After step 1: `bash tools/launchd/test_driver_notify.sh | grep -c "FAIL: drain: whole drain"` →
  `1` → after step 2: `0` (and the suite prints `0 failed`, rc=0).
- `grep -c 'MISSION_DRAIN_BUDGET' tools/launchd/mission-control.sh` → M3-base: `0` → after: ≥1.

**Mutations (anchored to M3's own diff — with M1+M2 kept and ONLY the guard removed, the backlog arm
reverts to the red state measured in step 1)**:
| Mutation | Kills assertion |
|---|---|
| Remove the `DRAIN_BUDGET` remaining-budget check before each row | `drain: whole drain returns within aggregate budget and preserves unattempted rows` |
| Attempt rows regardless / drop the deferred-drain diagnostic | `drain: deferred-drain diagnostic names deferred row count` |

---

## M4 — Mutation sweep + full verification (no new production/test diff)

**Files**: none modified — verification only (mutation copies under `mktemp -d`).

**Design trace**: Mutation Table (all 11 rows); Acceptance "GREEN after" block.

**Steps**:
1. Run the mutation sweep: for each mutation row from M1–M3 plus the two deferred-to-sweep rows
   below, apply it to a COPY of the tree files in a temp dir, prove the copy parses (`bash -n`),
   run the suite against the mutated copy, record that the named assertion FAILS and the failure is
   not a crash of the harness itself. Sweep additions:
   - "Suppress re-appending a failed drain row" kills
     `production: hanging drain send is cut off and row retained`.
   - "Delete only the top-level drain invocation" kills `wiring: exactly one preflight drain call`.
2. Full gate: `make test-launchd-drivers` → rc=0; record exit code and elapsed.
3. Determinism: run `bash tools/launchd/test_driver_notify.sh` twice on the clean tree; identical
   pass/fail counts; `git status --porcelain tools/launchd/` shows only the two intended files
   modified (read-only git).
4. Confirm no mutation text survives in the real tree (`grep` for each mutation marker) and
   `bash -n` both files.

**Boundary expectation**: suite count unchanged from M3 (`39 passed, 0 failed`); every mutant kills
its named assertion; `make test-launchd-drivers` rc=0.

**Acceptance**:
- `make test-launchd-drivers; echo rc=$?` → base of SPRINT: rc=1 → rc=0. (Red carried by M1's seven
  rows — declared, not independent evidence.)
- The sweep log shows 11/11 mutants killed their named assertions.
- `grep -c "124" tools/launchd/test_driver_notify.sh` ≥ 1 (hang labs exist) — green/green guard,
  declared.

**Mutations**: this milestone IS the sweep; its `mutations` array in the JSON lists all 11 rows with
their single named victim assertion each.

**Non-vacuity note**: M4 adds no diff, so it has no anchoring mutation of its own — its value is
re-running the M1–M3 mutations against the FINAL tree, catching a mutation that an earlier
milestone's base-state masked. Declared explicitly so no one counts M4 as assertion-anchored.

---

## Handoff

- Executor: load the `sprint-executor` skill; work milestone-by-milestone IN ORDER; record actual
  counts/timings into this plan's boundary lines (or the sprint JSON) as you go. **NO git writes —
  the controller commits.** Bounded-wait discipline: every hang lab carries its own outer
  `date +%s` watchdog; never end a run waiting on a notification.
- Out of scope (design "Deferred"): the seven non-D-60 call sites (P:1470/1473/1486/1835/1838/1852/
  1855) — named queue row `M-LAUNCHD-NOTIFY-REMAINING-BOUNDING`. Do not touch them.
- Evaluator: `sprint-evaluator` against the design's Acceptance Criteria + Mutation Table; verify
  the recorded boundary counts and the sweep log, and re-run `make test-launchd-drivers`
  first-party.
