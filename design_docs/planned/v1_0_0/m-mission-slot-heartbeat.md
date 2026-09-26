# M-MISSION-SLOT-HEARTBEAT: Per-Gate Heartbeat for Mission Slot Death Attribution

**Status**: Planned
**Target**: v1.0.0 (mission-loop tooling; no language surface)
**Priority**: P0 — ratified human ruling D-52 (Mark Edmondson, attended 2026-09-01)
**Estimated**: 3 days (one sprint, one iteration per the ruling)
**Dependencies**: None (touches only `tools/launchd/`, `make/test.mk`, `.claude/skills/mission-control/SKILL.md`)
**Axiom scoring**: N/A — harness/driver tooling, zero language, parser, typechecker, or codegen surface.
**Quorum triggers (attended checklist)**: none of the four fire (no design-freeze items, no shared-machinery override — every Conflict Surface row is "reuse" or "new namespaced file", no cost/KPI/banked-schema surface, all premises in-repo). This doc is produced by the unattended mission loop, so **quorum runs anyway** (unattended rule: always).

---

## Mandate

Charter `design_docs/v1-mission.md` D-52 (line 119), answered by Mark, attended, 2026-09-01:

> "(a) SPEND ONE ITERATION on it. Instrument the driver to write a per-gate heartbeat artifact so a
> slot that dies names the gate it died in. A ~40% reaped-slot rate is too high to keep absorbing by
> hand, and the recovery iterations cost more than the diagnosis will. **Report the MECHANISM when
> you have it, not just the rate** — the traces recover work but have never once explained it."

Measured basis (first-party at iteration 311): of iterations 296–310, six (299, 300, 302, 303, 306,
307) have **0** charter commits each (`git log -S "ITERATION <n>"`) — died mid-flight, not lost to
rotation (that half audited clean: 270 archive stamps). Recovery costs a whole later iteration each
time (308 spent itself recovering 306 and 307).

## Problem Statement

The mission loop fires `claude -p` (or codex/pi) controller sessions from
`tools/launchd/mission-control.sh` on a launchd timer. A controller session walks Gates 0→5 per
`.claude/skills/mission-control/SKILL.md`. Some slots die mid-flight and **exit rc=0** (V10), so:

- Neither `HARD_TIMEOUT` nor the stall watchdog fires — both are built to ignore clean exits.
- The driver logs `iteration complete (rc=0)` (line 1071, V9) — indistinguishable from success.
- The one attribution tell (`Background tasks still running after 600s`) is **suppressed by the
  very fix** that was installed against it (`CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS=0`, driver line
  850, V10) — the loop is blind to post-fix instances, and Standing rule 7 documents a post-fix
  death (iteration 176-era) that the ceiling did not prevent.
- No heartbeat/gate-stamp mechanism exists anywhere in `tools/launchd/` (V1).

So a dead slot leaves a plausible transcript, zero commits, zero charter rows — and **nothing that
names the gate it died in**. Six anecdotes, zero mechanisms. This design makes every slot's death
(or completion) classify itself in the driver's own log, and makes the rate derivable by one
command instead of a hand audit.

## Non-Goals

- Not a general observability platform. No dashboards, no Firestore, no new daemons.
- Not a fix for the reap itself — this is the *diagnosis* instrument D-52 bought. The fix is a
  later iteration, informed by the mechanism this reports.
- No migration of V1's legacy state paths (V6) — new files only.

## Related Documents

Searched `design_docs/{planned,implemented}` for `heartbeat` (V17; instrument's positive control =
it finds the rows below). None overlap this design; distinctions:

- `internal/coordinator/heartbeat_file.go` → `~/.ailang/state/worker_heartbeats.json` (V8) —
  **coordinator worker liveness**, a different plane and a different filename; no collision.
- `design_docs/planned/v1_1_0/global-collaboration-hub.md` — machine keep-alive heartbeats
  (Firestore), not mission slots.
- `design_docs/planned/v0_29_0/m-eval-stream-health-retry.md` — model stream liveness, explicitly
  notes no heartbeat protocol exists for ollama/opencode; unrelated plane.
- `design_docs/implemented/v0_30_0/m-ailang-fmt.md`, **that doc's own Verification Log row V19**
  (m-ailang-fmt.md:371 — not this doc's V19) — corroborating survey that ad-hoc temp+rename is the
  repo's atomic-write pattern; re-measured first-party here as V23.

---

## Design Questions — Decisions

### Q1 — WHO writes each gate stamp?

Candidates, each with its failure mode stated:

**(a) Controller calls a helper script at each gate boundary** (skill edit).
*Failure mode*: can be forgotten — and is forgotten precisely by a session dying early or a
controller that skimmed the skill. Also blind to a slot that dies before Gate 0.
*Strengths*: only the controller knows gate boundaries; provider-agnostic (codex and pi
controllers run the same skill and the same shell — V16); the skill is the single shared runtime
contract for all four missions.

**(b) Claude Code hook in `.claude/settings.json`** — infrastructure exists (V15: SessionStart /
PreToolUse / PostToolUse / Stop hooks already route to `scripts/hooks/*`).
*Failure mode*: hooks map **tool events, not gates** — attribution would be a heuristic
reconstruction (which tool call "means" Gate 2?) that breaks every time the skill text changes.
Hooks also fire for *every* session in the repo (attended sessions, subagents via SubagentStart,
CI-adjacent runs), so stamps need session-discrimination logic; and hooks are **claude-only** —
the driver launches codex and pi controllers through the same code path (V16), which would get
zero coverage. Mechanical and unforgettable, but it answers "was the process alive?" not "which
gate?".

**(c) Driver post-processes the session transcript after exit.**
*Failure mode*: provider-dependent (claude JSONL under `~/.claude/projects/`, codex and pi have
their own formats), no session-id capture exists in the driver today, and transcripts do not mark
gates — same heuristic-reconstruction problem as (b), plus a fragile newest-file heuristic under
concurrent missions. Rejected.

**DECISION: (d) a minimal combination of (a) + a driver-written floor stamp, with (a)'s
forgetfulness converted into a loud, self-detecting failure (see Q6):**

1. **The DRIVER writes the first stamp** (`fired`) **per ATTEMPT, inside `_mc_run_once()`**,
   immediately before the provider launch. The driver retries the controller up to
   `TRANSIENT_RETRIES` (default 3) times per fire (V21), and everything attempt-scoped is already
   per-attempt there — watchdog PIDs and the pidfile are both refreshed per attempt (V21, driver
   lines 949/974) — so per-attempt is the *consistent* granularity, not a new concept. Round 1
   truncated per FIRE, which made a deterministically wrong attribution possible: attempt 1
   reaching `gate-4`, then attempt 2 dying before Gate 0, would read `REAPED at=gate-4`. Accurate
   gate attribution is the mandate, so that residual was not acceptable (round-2 fix, Finding 1).
   This covers the dies-before-Gate-0 case
   *without* controller cooperation: a slot whose artifact holds only `fired` died before Gate 0
   (or never loaded the skill) — and that case **does matter**: Gate 0 aborts are a designed path
   ("abort = exit silently", V11), so pre-Gate-0 death must be distinguishable from a legitimate
   abort. The skill's Gate-0 abort path therefore also stamps `abort <reason>`; a missing abort
   stamp degrades to a false REAPED-at-gate-0 — a false *alarm*, never false silence (named
   residual R3).
2. **The CONTROLLER stamps each gate boundary** via a new ~80-line helper
   `tools/launchd/mission-heartbeat.sh stamp <label>` (name unallocated, V2), one mandated first
   action per gate section in SKILL.md: `gate-0`, `gate-1`, `gate-2`, `gate-3`, `gate-3b`,
   `gate-4`, `gate-5`, and `complete` at the end of Gate 5 (gate set verified V11).
3. **The DRIVER classifies on exit** (Q4) by triangulating two independent instruments: exit rc
   (driver) and the stamp file (heartbeat). (Round 1 used a third — mission-log record
   advancement — dropped in round 2 because its working-tree reader is dead on the frozen pin,
   V24/V25; see Q4 and Finding 4.) (a)'s failure mode — a forgotten stamp — makes the two
   instruments *disagree* (rc=0 with no `complete` stamp), and the disagreement is itself a loud
   verdict (Q6), so forgetting is detected within one fire instead of silently rotting.

Option (b) is not taken even as a supplement: the driver's `fired` stamp already covers the only
case (a) cannot see, at zero heuristic cost and with codex/pi parity.

### Q2 — What survives a reap?

Each stamp is **one `printf` append (`>>`) at the moment the boundary is crossed** — a single
`write(2)` under `O_APPEND`, one line, well under 512 bytes, so it is atomic and durable the
instant it returns. Nothing is buffered, accumulated, or flushed at exit — a design that writes
the artifact at the end is *itself reaped with the slot* and records nothing, which is the exact
defect this replaces (and mutation `MUT-BUFFERED` in Q7 kills any regression to it). The driver's
`fired` stamp uses `>` (truncate) to reset the artifact **per attempt** (Q1 point 1; mutation
`MUT-STALE-ATTEMPT` kills a regression to per-fire reset).

**Superseded attempts are carried, not discarded**: before truncating for attempt N+1, the driver
appends one `verdict=RETRIED at=<last-stamp> rc=$RC attempt=N` row to the history file (Q5). A
retried attempt is by construction an rc≠0 exit with a transient signature, and its last stamp is
the only record of *where in the gate walk* transient failures strike — one append preserves that;
discarding would erase it for zero savings. The retry path is inside the loop, before the verdict
block, so RETRIED rows never race the fire's final verdict row.

### Q3 — Where does it live? Namespaced?

```
~/.ailang/state/mission-${MISSION_NAME}-heartbeat            # current attempt (truncated per attempt, Q1)
~/.ailang/state/mission-${MISSION_NAME}-slot-verdicts.log    # rolling history (bounded, Q5)
```

- `STATE_DIR="$HOME/.ailang/state"` is the driver's literal (line 59, V6). The listing (V7) shows
  the collision hazard is real: bare fleet-shared keys (`mission-gh-issue`, `mission-model-last`)
  coexist with namespaced ones (`mission-v1-gh-issue`, `mission-motoko-model-last`,
  `mission-world-designer-rotation`). V1's own driver block documents (V6) that the bare
  `mission-gh-issue` silently served a CLOSED thread to Gate 0 until iteration 282 deliberately
  broke the legacy-path rule and namespaced it. **Both new paths carry `${MISSION_NAME}` from day
  one, for V1 too** — new files have no bit-for-bit legacy constraint, so the legacy exemption
  does not apply.
- The helper resolves `MISSION_NAME` from the environment — the driver exports it (line 52, V5),
  and child controllers (claude/codex/pi) inherit it. `MISSION_NAME` **unset** = attended or
  non-mission session: the helper prints
  `mission-heartbeat: MISSION_NAME unset — not a scheduled mission slot; no stamp written` and
  exits 0. This is a *visible* no-op, not a silent fallback: it writes nothing (never a bare or
  empty-name path — `MUT-SHARED-PATH` asserts this), and the message lands in the session
  transcript. Stamping is meaningless without a driver to classify, so refusing loudly-on-stdout
  is the honest behavior.
- Tests override via `AILANG_STATE_DIR` (helper: `${AILANG_STATE_DIR:-$HOME/.ailang/state}`) so
  `make test-launchd-drivers` never touches live state (existing convention: test_driver_notify.sh
  uses a fresh `mktemp -d` STATE_DIR per arm, V14).

Stamp line format (tab-separated, one per line):
```
<epoch>	<ISO8601>	<label>	<attempt>	<optional note>
```
`<attempt>` is read from `MISSION_ATTEMPT` (the driver exports it before each `_mc_run_once`
invocation; `${MISSION_ATTEMPT:-1}` in the helper so attended/manual stamps are well-formed).
Labels are a closed set — `fired | gate-0 | gate-1 | gate-2 | gate-3 | gate-3b | gate-4 | gate-5 |
complete | abort` — validated by a `case` statement (bash 3.2: no `declare -A`, no `${v,,}`; V3);
an unknown label exits 2 loudly rather than writing garbage.

### Q4 — How does the driver classify on exit?

A new block in `mission-control.sh` between marker comments
`# --- SLOT VERDICT START ---` / `# --- SLOT VERDICT END ---` (markers so the test awk-extracts
the REAL block, per the directory's "a retyped copy tests the copy" convention, V14), executed on
**every** exit path — after the retry loop, **on the common exit path BEFORE the rc=0/rc≠0
split and BEFORE `rm -f "$PIDFILE"` (line 1037)** — i.e. **BEFORE
`rm -f "$PIDFILE"` (line 1037, V22)**. That ordering is load-bearing: removing the pidfile
releases the overlap guard (guard read at lines 763–768, V22), and `StartInterval` is 5400 s, so
a fire already waiting on the guard is free to proceed — and truncate the heartbeat — the instant
line 1037 runs. Round 1 specified the block *after* the removal, which raced exactly that
(round-2 fix, Finding 2).

**Two phases, split at the guard release:**

1. **Classify — under the guard.** Read the stamp file, compute the verdict, emit the
   `slot-verdict:` log line, append the history row (Q5). While holding the guard this phase must
   NOT block, wait, or touch the network: it is pure local reads of two small files (`tail`/`awk`)
   plus `O_APPEND` writes — bounded at milliseconds. No `ailang messages send`, no `gh`, no `git`.
2. **Notify — after `rm -f "$PIDFILE"`.** The episode-gated controlplane message and
   bookkeeping-issue comment use the already-computed verdict string; they can take network time
   because nothing they touch can be truncated by a successor (driver log and history file are
   append-only and mission-namespaced).

Inputs: `RC`, `MISSION_ATTEMPT`, and the stamp file. **Record-advancement is deliberately NOT an
input** (round-2 fix, Finding 4): this loop lands its mission-log record by committing in a
*separate* worktree and merging a PR to `origin/dev`, while the pin worktree is checked out once
at fire start (`pin-root.sh` sourced at lines 394–396, before the controller launch at 968; the
driver itself contains zero `git checkout`/`git fetch`, V24) — so a working-tree
`post_last_record` **equals `pre_last_record` even for a fully successful iteration** (reader
discrimination proven V25). Round 1's `COMPLETED` predicate required "record advanced = yes",
which was therefore unreachable: every healthy iteration would have classified
`COMPLETE-NO-RECORD` — a loud false alarm on the success path, firing every time, training its
reader to ignore it and destroying the Q6 floor. A `git fetch` on the exit path was rejected: it
would be a network call inside phase 1 (forbidden above), it has no clean bound on this rig (no
GNU timeout on macOS — the driver hand-rolls watchdogs for exactly that reason), and its failure
mode would have to degrade to yet another named verdict. The `complete` stamp already attests the
controller *reached the end of Gate 5*, which is the property the exit classifier can honestly
measure; record corroboration is routed to its own queue row (see Residuals R7).

Decision table — **positive evidence only; there is no default-to-COMPLETED branch**:

| rc | stamp evidence (this attempt's artifact) | verdict |
|---|---|---|
| any | artifact missing | `HEARTBEAT-MISSING` (loud: driver wrote `fired` itself, so absence = driver/state-dir fault) |
| 0 | last = `complete` | `COMPLETED` |
| 0 | last = `abort` | `ABORTED` (legitimate Gate-0-family exit) |
| 0 | last = `gate-N` | `REAPED at=gate-N` — **the D-52 deliverable** |
| 0 | only `fired` | `DIED-PRE-GATE-0` (also what a silently disabled stamping mechanism looks like — loud either way, Q6) |
| 143/137 | last = X | `KILLED at=X` (watchdog kill, now gate-attributed) |
| other ≠0 | last = X | `CRASHED at=X` (the existing rc≠0 branch and its late-kill sub-branch keep their behavior; the verdict adds gate attribution alongside) |
| — | anything else | `UNCLASSIFIED` (loud; never a checkmark) |

**Exact log line** (driver log, adjacent to the existing `iteration complete (rc=0)` /
`iteration exited rc=$RC` lines) — reports the attempt it classifies:

```
slot-verdict: REAPED at=gate-3 rc=0 attempt=2/3 stamps=5 last_age_s=412 elapsed_s=1840 mission=v1 hb=/Users/voightkampff/.ailang/state/mission-v1-heartbeat
```

(`last_age_s` = exit-time minus last-stamp epoch: distinguishes "stamped gate-3 then worked 30 min
before dying" from "died at the boundary". `elapsed_s` = fire-to-exit: distinguishes a 2-second
crash from a 40-minute unstamped run.)

**Missing artifact is a classification, not an error**: `HEARTBEAT-MISSING` logs the same
`slot-verdict:` line shape and still appends to the history file — the instrument's own failure is
a first-class row, never an unlogged exception.

On `REAPED`/`DIED-PRE-GATE-0`/`HEARTBEAT-MISSING`/`UNCLASSIFIED`, the driver additionally
notifies in phase 2, after the guard release. **Phase 2 performs no synchronous network I/O. It
appends a namespaced notification request to a bounded local queue and returns. A separately
invoked notifier may deliver it only through a verified wrapper with a fixed deadline and
process-group kill; timeout or delivery failure is recorded as `NOTIFY-TIMEOUT` or `NOTIFY-FAILED`
without changing the slot verdict.** (gpt5-6-sol round-2 `proposed_fix`, applied verbatim.)

The fix's fallback clause — *"if no existing bounded notifier can be verified, omit
notifications"* — does **not** trigger: `_mc_bounded()` already exists in this very script at
line **194** with a `date +%s` deadline, `kill` then `kill -9`, a `124` timeout return, and
`( exec "$@" )` so the kill reaches the real process rather than a subshell (8 call sites; V27).
Delivery keeps the episode gate: the `_rc_ep`-style marker `mission-${MISSION_NAME}-reaped.episode`
keyed on the verdict+gate string (clears on any `COMPLETED`) — same rationale as Mark's
2026-08-31 episode-gating ruling: first occurrence loud, repeats silent until the shape changes.

**Scope note, so this is not read as a wider promise:** the driver's *existing* exit-path notices
(lines 1046 and 1063) call `ailang messages send` **unbounded** today. That is a pre-existing
exposure this design neither introduces nor fixes; it is routed as **R8**, not absorbed.

### Q5 — How is the rate derivable?

One history line per *final* attempt appended to `mission-${MISSION_NAME}-slot-verdicts.log`,
plus one `verdict=RETRIED` line per superseded attempt (Q2) — so every attempt is accounted for
and a fire's rows are its RETRIED rows (0–2 of them) followed by its verdict row:

```
<ISO8601> verdict=REAPED at=gate-3 rc=0 attempt=2/3 elapsed_s=1840 stamps=5 controller=claude:claude-opus-5-20260115
```

**Bound: 200 lines** (≈5–7 weeks at V1's fire rate even with retry rows — comfortably spans a
charter audit window like 296–310 with 10x margin). **Eviction: on every append, rewrite via
`tail -n 200` to a temp file + `mv`** — the repo's established *ad-hoc* atomic pattern; no shared
helper exists. Measured first-party (V23): the temp-then-rename idiom is inlined at
`internal/coordinator/heartbeat_file.go:114`, `internal/eval_analysis/dashboard_io.go:196`,
`cmd/ailang/editor_vscode.go:260`, and in shell at
`tools/vscode-extension-build/bundle.sh:44` (`mv "$TMP" "$OUT"`); corroborated by
`m-ailang-fmt.md`'s own row V19 (m-ailang-fmt.md:371 — that doc's numbering, not this doc's V19).
Unbounded is not acceptable on this rig: `~/.ailang/state/` already carries the scar tissue of
exactly this class — `nightly-eval-history.jsonl.bak-iter119` / `.bak-iter128` are manual trims of
a file that grew without a bound (V18) — and the driver reads its own state files at fire time, so
an unbounded file is an unbounded read on the hot path.

Rate, one command, no hand audit:

```bash
awk '{ n[$2]++ } END { for (k in n) print k, n[k] }' ~/.ailang/state/mission-v1-slot-verdicts.log
# verdict=COMPLETED 9   verdict=REAPED 4   ...
```

### Q6 — Anti-vacuity floor: how is a dead heartbeat distinguishable from a healthy loop?

This is the load-bearing property, because a broken heartbeat fails as *silence* — the very thing
it detects. Three constructions, layered:

1. **Every completed iteration is a mandatory self-test of the instrument.** `COMPLETED` requires
   positive evidence from both live instruments (rc=0 AND the `complete` stamp) — never rc alone.
   A healthy loop whose stamping is silently disabled (skill edit reverted, helper deleted, path
   broken) is **impossible to classify as healthy**: its artifact holds only the driver's `fired`
   line, so it lands in `DIED-PRE-GATE-0`, which logs loudly and messages the controlplane — on
   the *very next fire*, whatever its outcome, because every fire exercises the same write path
   deaths depend on. (Round 1 used a third instrument, working-tree record-advancement, for this
   floor; V24–V26 showed that instrument is dead on the pin — it would have made the loud arm
   fire on *every healthy* iteration instead, which is the floor inverted. See Q4.)
2. **The driver writes `fired` itself**, so "artifact absent" always means the *driver's* write
   failed (`HEARTBEAT-MISSING`, loud) — controller non-cooperation can never present as an absent
   file.
3. **No default-pass branch exists.** The verdict `case` has no path to `COMPLETED` without
   positive stamps; its fall-through is `UNCLASSIFIED`, which is loud. An enumerator fed an empty
   set (`stamps=0`) can emit `DIED-PRE-GATE-0`, `HEARTBEAT-MISSING`, or `UNCLASSIFIED` — never a
   checkmark. Mutation `MUT-DEFAULT-COMPLETE` (Q7) pins this in CI.

### Q7 — Proof: simulated reap + one named mutation per refusal branch

New `tools/launchd/test_mission_heartbeat.sh`, wired into `make test-launchd-drivers`
(make/test.mk:40, V4) and therefore into the existing `launchd-drivers` CI job — which runs on
`macos-latest` under `/bin/bash` with a guard asserting `BASH_VERSINFO[0] -eq 3` (ci.yml:535, V4).
The inherited "tools/launchd has zero CI coverage" memory is **stale — re-checked and refuted**
(V4); bash-3.2 enforcement is already free.

Style per V14: the verdict block and the helper's stamp function are **awk-extracted from the real
files between the markers**, never retyped; extraction emptiness is itself a FATAL.

**Simulated reap (the headline test):** a fake controller script stamps `gate-0`, `gate-1`, writes
a marker that it is "inside gate-1 work", then `sleep 300`. The test `kill -9`s it mid-sleep
(SIGKILL = no traps, exactly like a harness reap of the process group), sets `RC=0`, leaves the
fake mission log unchanged, runs the extracted verdict block, and asserts the output contains
`REAPED at=gate-1` — the artifact must ALREADY name the gate at kill time, having been written BY
the mechanism, DURING the work, not alongside the test's assertion.

**Named mutations — each names the refusal branch it attacks and the single observable it moves:**

| Mutation | Edit applied to the extracted copy | Observable that must move (test fails) |
|---|---|---|
| `MUT-DEFAULT-COMPLETE` | make the verdict `case` fall through to `COMPLETED` | empty-stamp arm stops printing `UNCLASSIFIED`/`DIED-PRE-GATE-0`; asserts on those strings fail |
| `MUT-BUFFERED` | helper accumulates stamps in a variable, writes on exit | SIGKILL arm's artifact is empty at kill time → `REAPED at=gate-1` assertion fails |
| `MUT-SHARED-PATH` | drop `${MISSION_NAME}` from the artifact path | two-mission arm (stamp as `v1`, then as `world`) sees cross-written file; distinct-file assertion fails; also asserts NO file is created when `MISSION_NAME` is unset |
| `MUT-UNBOUNDED` | remove the `tail -n 200` trim | 250-append arm's `wc -l` ≠ 200 |
| `MUT-MISSING-IS-PASS` | treat absent artifact as `COMPLETED` | deleted-artifact arm stops printing `HEARTBEAT-MISSING` |
| `MUT-UNSTAMPED-IS-PASS` | drop the `complete`-stamp requirement (accept any last stamp on rc=0) | rc=0 + last=`gate-5` arm prints `COMPLETED` instead of `REAPED at=gate-5` |
| `MUT-STALE-ATTEMPT` | move the `fired` truncation out of `_mc_run_once` (per-fire reset) | retry arm (attempt 1 stamps `gate-4`, simulated transient rc≠0, attempt 2 dies pre-Gate-0) reads `REAPED at=gate-4` instead of `DIED-PRE-GATE-0 attempt=2` |

Mutations run as test arms against a sed-mutated copy of the *extracted* block (or helper): each
arm asserts the mutated copy FAILS the assertion the pristine copy passes — a mutation that
doesn't move its observable is a broken test, reported as such.

---

## Conflict Surface

What else touches the things this design touches (all decisions are "reuse" or "new namespaced
file" — nothing shared is overridden):

| Surface | Who else is there | Collision analysis |
|---|---|---|
| `~/.ailang/state/` | coordinator (`worker_heartbeats.json`, V8), all four mission drivers (namespaced + V1-legacy bare keys, V6/V7), quorum artifacts (`mission-quorum/`), eval history, DBs | Both new filenames embed `${MISSION_NAME}`; no existing key matches `mission-*-heartbeat` or `mission-*-slot-verdicts` (V1, V7). `worker_heartbeats.json` is a different filename written by Go code (V8) — zero overlap |
| Driver exit paths | rc≠0 messaging (episode-gated `_rc_ep`), late-kill-post-record branch, `rm -f "$PIDFILE"` line 1037 (V9, V22) | Verdict phase 1 (classify + log + history append) runs BEFORE `rm -f "$PIDFILE"`; phase 2 (notices) after. Neither modifies RC or any existing branch's condition; the late-kill branch keeps its behavior (its own record-instrument defect is pre-existing and routed, R7) |
| Concurrent sibling fire | motoko/world/docs drivers share the rig and the skill file | Distinct `MISSION_NAME` ⇒ distinct artifact + history files. Within one mission: truncation happens only inside `_mc_run_once`, i.e. while this slot holds the per-mission PIDFILE (V13, V22), and the verdict's reads + history append complete in phase 1, before `rm -f "$PIDFILE"` releases the guard — so a successor's truncation is *ordered after* this slot's last read. The ordering is what makes the no-race claim true; round 1 asserted it while specifying the opposite order (Finding 2) |
| Transient-retry loop (driver lines 1019–1034, V19, V21) | retries re-invoke `_mc_run_once` within one fire | `fired` truncation is per ATTEMPT (inside `_mc_run_once`, matching the per-attempt pidfile refresh and watchdogs, V21); superseded attempts leave one `RETRIED` history row each (Q2). The verdict classifies the final attempt's artifact and says so (`attempt=N/M`) |
| SKILL.md | shared by all four missions; three prior namespacing incidents recorded in it | The stamp instruction added per gate is `bash tools/launchd/mission-heartbeat.sh stamp gate-N` — no literal path, no mission name in the skill text; the helper derives everything from env (V5). Attended invocations of the skill hit the helper's visible no-op |
| `make test-launchd-drivers` / CI | six existing test scripts + one in tools/eval (V4) | Additive: one new script line in make/test.mk; the CI job already runs the target under bash 3.2 |

## Files to Modify

- `tools/launchd/mission-heartbeat.sh` — NEW, ~80 LOC. `stamp <label> [note]` subcommand; env
  resolution, label validation, visible no-op on unset `MISSION_NAME`, `AILANG_STATE_DIR` override.
  Pure bash 3.2 (case statements only, V3).
- `tools/launchd/mission-control.sh` — +~60 LOC: per-attempt `fired` truncate-write +
  `MISSION_ATTEMPT` export inside `_mc_run_once`; `RETRIED` history row in the retry path;
  `# --- SLOT VERDICT START/END ---` phase-1 block (classification, exact log line, bounded
  history append) invoked **on the common exit path BEFORE the rc=0/rc≠0 split and BEFORE `rm -f "$PIDFILE"` (line 1037)**; phase-2
  episode-gated notices after it. The rc=0 branch gains NO record snapshot — record-advancement
  is out of the predicate (Q4, Finding 4).
- `.claude/skills/mission-control/SKILL.md` — +~15 lines: one stamp instruction at the head of
  each gate section (V11), `stamp complete` at Gate 5's end, `stamp abort <reason>` on the Gate-0
  abort path, and a short Standing-rules note tying the mechanism to rule 7.
- `tools/launchd/test_mission_heartbeat.sh` — NEW, ~190 LOC (extraction + reap simulation + 7
  mutation arms + retry/attempt arm + namespacing/bound arms).
- `make/test.mk` — +1 line in `test-launchd-drivers` (line 40 block, V4).
- `CHANGELOG.md` — entry under Unreleased.

## Acceptance Criteria

Each is a command that can fail, baselined against untouched `dev` (every AC below is red at base:
the helper, test, make-line, and verdict strings do not exist there — V1, V2).

1. **Reap names its gate** (red at base: test file absent):
   `/bin/bash tools/launchd/test_mission_heartbeat.sh` exits 0, and its output contains
   `PASS: sigkill mid-gate-1 => REAPED at=gate-1`.
2. **Wired into the guarded suite** (red at base):
   `grep -q test_mission_heartbeat.sh make/test.mk && make test-launchd-drivers` exits 0.
3. **All seven mutations kill** (red at base): test output contains `mutations: 7/7 killed`; any
   surviving mutation is a test FAIL, not a warning.
4. **Namespacing** (red at base): test arm asserts stamps under `MISSION_NAME=v1` and
   `MISSION_NAME=world` land in two distinct files and that unset `MISSION_NAME` creates **no**
   file while printing the visible no-op line.
5. **Bound holds** (red at base): after 250 simulated appends,
   `wc -l < $STATE/mission-v1-slot-verdicts.log` prints exactly `200`.
6. **Live-fire anti-vacuity check** (post-merge, next scheduled V1 fire; red at base — the string
   exists nowhere): `grep -c 'slot-verdict: COMPLETED' /tmp/ailang-mission-control.log` ≥ 1 after
   the first *completed* iteration, with the paired control that
   `tail -1 ~/.ailang/state/mission-v1-heartbeat` shows label `complete`. (A first live fire that
   instead yields `REAPED at=gate-N` also passes THIS criterion's intent — the instrument spoke —
   and is itself the D-52 mechanism report.)
7. **Rate derivable** (post-merge): the Q5 `awk` one-liner over the verdicts file prints ≥1
   `verdict=` count line; empty output = FAIL (an empty enumeration is a claim, not a fact).

## Round 2 — quorum response

Round 1 returned **BLOCKED** (`gpt5-6-sol` reject, `gemini-3-1-pro` reject). **`oc-glm-5-2` was
ABSENT (reason `invalid`), so round 1 was decided at N−1.** Artifact:
`.ailang/state/mission-quorum/m-mission-slot-heartbeat-2026-09-01T08-35-13Z.json`. All four
findings below were re-measured first-party in this revision pass; none disputed the direction
(three-instrument→two-instrument classification, no-default-pass verdict switch, driver-written
`fired` stamp all retained).

| # | Objection (source) | Settling measurement | Change made |
|---|---|---|---|
| 1 | Heartbeat reset per FIRE, but the driver retries the controller up to 3× per fire — attempt 1 reaching `gate-4` + attempt 2 dying pre-Gate-0 reads `REAPED at=gate-4`, a deterministically wrong attribution (gpt5-6-sol) | V21: retry loop at ~1021–1033 invokes `_mc_run_once` (defined line 950) up to `TRANSIENT_RETRIES=3`; watchdogs and pidfile already per-attempt (lines 949/974) | `fired` truncation moved inside `_mc_run_once` (per ATTEMPT — the granularity the driver already uses); stamps carry `attempt=N` via exported `MISSION_ATTEMPT`; verdict line reports `attempt=N/M`; superseded attempts leave `RETRIED` history rows (Q1, Q2, Q5); new mutation `MUT-STALE-ATTEMPT` pins it. Round 1's V19 "accepted" note marked SUPERSEDED |
| 2 | Verdict block specified AFTER `rm -f "$PIDFILE"`, which releases the overlap guard — a waiting fire could truncate the heartbeat mid-read (gemini-3-1-pro) | V22: guard release at line 1037; guard read at 763–768; `StartInterval` 5400 | Verdict split into phase 1 (classify + log + history append, BEFORE line 1037, local-only, no network, bounded) and phase 2 (notices, after release); Conflict Surface row now states the *ordering* that makes the no-race claim true instead of asserting it (Q4) |
| 3 | Q5's atomic-pattern citation "m-ailang-fmt V19" collides with this doc's own V19 (the retry loop) — unreadable as written (gemini-3-1-pro) | V23: re-measured the precedent myself — `os.Rename` inlined at 3+ Go sites and `mv "$TMP" "$OUT"` at bundle.sh:44; m-ailang-fmt.md:371 is that doc's own V19 recording the same survey | Citation rewritten to name the file, line, and "that doc's numbering"; claim now rests on my own V23 row with the external row as corroboration (Q5, Related Documents) |
| 4 | "Record advanced" is a DEAD instrument on the pin — the pin worktree is frozen for the run while records land via PR to `origin/dev`, so round 1's `COMPLETED` predicate was unreachable and every healthy iteration would classify loud `COMPLETE-NO-RECORD` (round-2 designer directive, confirmed) | V24: zero `git checkout/fetch` in the driver (control 2), pin-root runs at 394–396 pre-launch; V25: working-tree read == `origin/dev` read == `## 314` while `origin/dev~3` == `## 313` (reader discriminates; the working-tree arm is the dead one); V26: late-kill branch fired 0 times vs 10 rc≠0 exits | Record-advancement dropped from the classification entirely (option b): `COMPLETED` = rc=0 + `complete` stamp; decision table, Q6 floor, mutations and ACs re-baselined; an exit-path `git fetch` rejected as an unbounded network call inside the guard-held phase. Vacuous-pass detection named as residual R6; the pre-existing dead late-kill detector filed as R7 with a queue-row recommendation, not absorbed into this sprint |

## Verification Log

All commands run 2026-09-01 in `/Users/voightkampff/.ailang-driver-pin/v1` (worktree at
`origin/dev`) on the driver-pin host (Darwin 25.5.0).

| # | Command | Observed output (verbatim or tightly excerpted) |
|---|---|---|
| V1 | `grep -rniE "heartbeat\|gate_stamp\|gate-stamp" tools/launchd/; echo "rc=$?"` with same-call controls `grep -rc "MISSION_NAME" tools/launchd/mission-control.sh` and fresh negative token `zzq_not_a_real_token_915`, plus `test -d tools/launchd` | 0 hits, `rc=1` (legitimate zero, not scope error); known-positive control `29`; negative control rc=1; `scope_ok`. **No heartbeat mechanism exists today** |
| V2 | `ls tools/launchd/mission-heartbeat.sh` | `No such file or directory` — helper name unallocated |
| V3 | `bash --version \| head -1` | `GNU bash, version 3.2.57(1)-release (arm64-apple-darwin25)` — bash-4 constructs forbidden; inherited claim CONFIRMED first-party |
| V4 | `grep -n "launchd-drivers:" .github/workflows/ci.yml`; `sed -n 40,47p make/test.mk`; `grep -rn "test-launchd-drivers" make/` | `535:  launchd-drivers:` — a dedicated CI job on `macos-latest` that asserts `/bin/bash` is 3.x then runs `make test-launchd-drivers`; target at `make/test.mk:40` runs 6 `tools/launchd/test_*.sh` scripts + `tools/eval/test_motoko_connection_probe.sh`. **Inherited "zero CI coverage" memory is STALE — refuted** |
| V5 | `grep -n "export MISSION_NAME" tools/launchd/mission-control.sh` | line 52: `export MISSION_NAME MISSION_REPO MISSION_DOC` — controllers (claude/codex/pi are child processes) inherit it |
| V6 | `sed -n 49,100p tools/launchd/mission-control.sh` | line 59 `STATE_DIR="$HOME/.ailang/state"`; lines 60–77: V1 uses LEGACY bare paths "bit-for-bit compat", EXCEPT `GH_ISSUE_FILE="$STATE_DIR/mission-v1-gh-issue"` — comment: "NAMESPACED, deliberately breaking the legacy-path rule above (iteration 282 found it) … V1 alone read the fleet-shared bare file, which holds a CLOSED thread" |
| V7 | `ls ~/.ailang/state/` | mixed namespacing observed: bare `mission-gh-issue`, `mission-model-last` alongside `mission-v1-gh-issue`, `mission-motoko-model-last`, `mission-world-designer-rotation`, …; also `worker_heartbeats.json`; no key matching `mission-*-heartbeat` or `*slot-verdicts*` |
| V8 | `grep -rn "worker_heartbeats" internal/ cmd/ tools/` | writer is `internal/coordinator/heartbeat_file.go:128` (`filepath.Join(stateDir, "worker_heartbeats.json")`) — coordinator worker liveness, distinct filename, no collision |
| V9 | `sed -n 1010,1075p tools/launchd/mission-control.sh` + `grep -n` anchors | `MISSION_LOG_FILE=` line 1014, `pre_last_record=` line 1015; `post_last_record=` line 1040 computed **only inside the rc≠0 branch**; rc=0 branch is 3 lines ending `log "iteration complete (rc=0)"` (line 1071) with **no record-advance check**; episode-gated `_rc_ep` messaging pattern present in rc≠0 branch |
| V10 | `sed -n 3780,3860p .claude/skills/mission-control/SKILL.md`; `grep -n "CLAUDE_CODE_PRINT_BG_WAIT_CEILING_MS" tools/launchd/mission-control.sh` | Standing rule 7: reaper "terminates still-running background tasks **600 s after the assistant's last turn ends** … exits **rc=0** … neither watchdog fires"; ceiling export at driver line 850 (`:-0`); rule 7 addendum: the ceiling "suppresses the very line it greps for — so a mission with the fix installed reads clean while dying exactly as before", and documents a post-fix death (`iteration complete (rc=0)` **nine minutes** in, executor still writing 12 min later) |
| V11 | `grep -n "^## Gate" .claude/skills/mission-control/SKILL.md` | Gate 0 PREFLIGHT (line 200, "abort = exit silently with a controlplane message"), Gate 1 OBSERVE (417), Gate 2 PICK+REALITY-CHECK (652), Gate 3 ROUTE+EXECUTE (2328), Gate 3b CI GREEN (3011), Gate 4 RECORD (3348), Gate 5 RETRO+REPORT (3610) — the closed stamp-label set |
| V12 | `grep -n '\| D-52 \|' design_docs/v1-mission.md` | line 119: RESOLVED, "**ANSWERED — (a) SPEND ONE ITERATION on it…**" (Mark Edmondson, attended 2026-09-01); measured basis: 6 of 296–310 with 0 charter commits |
| V13 | `sed -n 930,1010p tools/launchd/mission-control.sh` | per-mission `PIDFILE` written per attempt (`printf '%s\n' "$CONTROLLER_PID" > "$PIDFILE"`, "overlap guard reads this"); providers launched: `codex exec` / `pi` / `claude -p` — three provider paths, one driver |
| V14 | `head -60 tools/launchd/test_driver_notify.sh` | established convention: "blocks are awk-extracted from the file, never retyped: a retyped copy tests the copy"; empty extraction = FATAL; fresh `mktemp -d` STATE_DIR per arm |
| V15 | `grep -n '"hooks"' .claude/settings.json`; `sed -n 75,215p .claude/settings.json` | hooks infrastructure EXISTS (line 75): SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, PostToolUseFailure, SubagentStart/Stop, Stop → `scripts/hooks/*` — option (b) is mechanically available and was evaluated, not assumed absent |
| V16 | (same read as V13) | codex and pi controller paths receive no Claude Code hooks — hook-based stamping would cover only the claude provider |
| V17 | `grep -rniE "heartbeat" design_docs/planned/ design_docs/implemented/ \| grep -vi worker` | hits only in `m-message-plane-trust.md` (coordinator), `global-collaboration-hub.md` (machine keep-alive), `m-eval-stream-health-retry.md` (model streams), `m-ailang-fmt.md` V19 (atomic-write survey) — positive control for the instrument; **no doc covers mission-slot gate attribution**; duplicate gate: proceed |
| V18 | `ls ~/.ailang/state/` (V7 listing) | `nightly-eval-history.jsonl.bak-iter119`, `.bak-iter128` — manual trims of an unbounded state file; precedent for the 200-line bound |
| V19 | `sed -n 1019,1034p tools/launchd/mission-control.sh` | transient-retry loop re-invokes `_mc_run_once` up to `TRANSIENT_RETRIES` within one fire. *(Round 1 accepted cross-attempt stamp accumulation on this basis; SUPERSEDED in round 2 — the design now resets per attempt, V21 and Q1.)* |
| V20 | `wc -c tools/launchd/mission-control.sh` | `67759` — matches the mandate's measurement; same file, same revision |
| V21 | `grep -cE '_mc_run_once' tools/launchd/mission-control.sh` (negative control `_mc_never_defined_915`); `grep -nE '_mc_run_once'`; `sed -n 342,343p;949p;974p` | `3` (control `0`): definition line **950**, invocation line **1023** inside the `attempt=1` retry loop (~1021–1033); `TRANSIENT_RETRIES="${MISSION_TRANSIENT_RETRIES:-3}"` line 342 ("total attempts incl. the first"), `TRANSIENT_BACKOFF` 343; line 949 "Watchdogs are per-attempt (fresh PIDs each retry)"; line 974 pidfile "per-attempt: retries refresh it". **Up to 3 controller attempts per fire; attempt-scoped state is already per-attempt** |
| V22 | `grep -n 'rm -f "\$PIDFILE"' tools/launchd/mission-control.sh`; `sed -n 763,768p`; `grep -n 'PIDFILE='`; `grep -A1 StartInterval tools/launchd/dev.ailang.mission-control.plist` | guard-release `rm -f` at line **1037** ("this instance owns the run"); overlap guard reads the pidfile at lines **763–768** (stale-pid `rm -f` at 768); `PIDFILE="$STATE_DIR/mission-${MISSION_NAME}.pid"` line 81; `StartInterval` = **5400**. **Anything after line 1037 can race a successor fire** |
| V23 | `grep -rn "os.Rename" internal/ cmd/ailang/`; `grep -rnE 'mv (-f )?"?\$\{?(tmp\|TMP\|temp)' tools/ scripts/` | temp-then-rename inlined ad hoc, no shared helper: `internal/coordinator/heartbeat_file.go:114`, `internal/eval_analysis/dashboard_io.go:196`, `cmd/ailang/editor_vscode.go:260`, `cmd/ailang/fmt.go` (~226–260, self-describes "matches the inlined temp-file + os.Rename convention used elsewhere"); shell-side: `tools/vscode-extension-build/bundle.sh:44` `mv "$TMP" "$OUT"`. Matches m-ailang-fmt.md:371 (that doc's V19) |
| V24 | `grep -cE '^[^#]*git (checkout\|fetch)' tools/launchd/mission-control.sh` (same-file positive control `^[^#]*git ` = 2); same grep on `tools/launchd/lib/pin-root.sh` (negative control `zzq_915_none` = 0) | driver: **0** (control 2) — the driver never re-checks-out or fetches during a run; `pin-root.sh`: **2** (control 0), sourced at driver lines **394–396**, i.e. before the controller launch (`claude -p` at line 968). **The pin worktree is frozen for the whole run** |
| V25 | Two-arm reader test: `grep '^## ' design_docs/v1-mission-log.md \| tail -1` (A) vs `git show origin/dev:…` (B) vs `git show origin/dev~3:…` (C) | A = B = `## 314 — …`; C = `## 313 — …`. A==B only because the pin was just checked out; **B≠C proves the `git show <ref>:` reader genuinely discriminates** — the working-tree reader is the dead one, not the concept |
| V26 | `grep -c 'AFTER recording itself' /tmp/ailang-mission-control.log` with positive control `grep -c 'iteration exited rc='` and fresh negative token `zzq_control_915_none` | **0** (positive control **10**, negative control **0**) — the pre-existing late-kill branch has never fired in the whole driver log. Consistent with (not proof of) its record instrument being dead per V24/V25 |
| V27 | `grep -n '_mc_bounded()' tools/launchd/mission-control.sh`; `grep -c '_mc_bounded'` (negative control `_mc_never_915`); `sed -n 194,212p`; `grep -n 'ailang messages send controlplane'` | `_mc_bounded()` defined line **194**, **8** uses (negative control **0**): `deadline=$(( $(date +%s) + secs ))`, `kill` -> `sleep 2` -> `kill -9`, returns **124** on timeout, `( exec "$@" )` so the kill reaches the real process. **A verified bounded wrapper exists**, so gpt5-6-sol's omit-notifications fallback does NOT trigger. Existing exit-path sends at lines **1046**/**1063** are unbounded - pre-existing, routed as R8 |
| V28 | Controller measured its OWN behaviour mid-fire (iteration 315), answering oc-glm-5-2's challenge that V24 covers only the DRIVER: `git status --porcelain` in the pin; `git symbolic-ref -q HEAD`; the same in the record worktree; `grep -n 'worktree branched from'` in the running SKILL.md | The pin held **1** dirty path, and it is `?? design_docs/planned/v1_0_0/m-mission-slot-heartbeat.md` - the *design doc*, untracked. The charter/mission-log edit was in `/Users/voightkampff/dev/sunholo-data/.wt-iter315` (**1** modified: `design_docs/v1-mission.md`), a separate worktree, exactly as SKILL.md line **3425** instructs (*"record in a worktree branched from `origin/dev` ... and land it by PR"*). Positive control: planting a file moved the pin's count **1 -> 2**, removing it moved it back, so the reader sees dirt when it exists. **The pin is DETACHED** (`git symbolic-ref -q HEAD` non-zero), so a PR merge to `origin/dev` has no branch to advance in the pin and cannot rewrite its working tree. **oc-glm-5-2's alternative is REFUTED for the mission-log reader**: the controller writes *design docs* into the pin but never the record, so record-advancement stays out of the `COMPLETED` predicate |

Negative-existence claims sweep (per the skill's rule): "no heartbeat mechanism" → V1; "helper
name unallocated" → V2; "no duplicate design doc" → V17; "rc=0 branch has no record check" → V9;
"no key matching the new filenames" → V7; "driver has no git checkout/fetch during a run" → V24;
"late-kill branch has never fired" → V26. Each carries its row.

## Milestones

**Day 1 — M1: helper + driver.** `mission-heartbeat.sh` (stamp, validation, attempt field, no-op
branch); driver: per-attempt `fired` write + `MISSION_ATTEMPT` export, `RETRIED` history rows,
phase-1 verdict block with markers BEFORE the pidfile release, exact log line, bounded history,
phase-2 episode-gated notices; `bash -n` + manual dry-run (`MISSION_CONTROL_DRY_RUN` path
unaffected — verify).

**Day 2 — M2: skill edits + tests.** SKILL.md stamp instructions (8 labels + abort path + rule-7
note); `test_mission_heartbeat.sh` with the SIGKILL reap simulation and all 7 mutation arms;
`make/test.mk` wiring; green `make test-launchd-drivers` locally under `/bin/bash`.

**Day 3 — M3: land + live-fire observation.** CI green (Gate 3b: the `launchd-drivers` job);
AC-6/AC-7 on the next scheduled V1 fire; record the first verdict distribution in the mission log.
Per D-52's explicit ask, the first `REAPED at=gate-N` verdict is reported to Mark as a MECHANISM
statement (which gate, last_age, elapsed), not a rate.

## Residuals — named, not assumed

- **R1 (attribution granularity ≤ skill compliance):** a controller that skips one mid-gate stamp
  shifts attribution one gate early (death in gate-3 reads `REAPED at=gate-2`). Bounded by Q6-1:
  systematic non-stamping surfaces as a loud `DIED-PRE-GATE-0` on the next fire. Not closable
  without hooks-per-gate, which don't exist as a concept (V15 maps tools, not gates).
- **R2 (mechanism ≠ gate):** the artifact names WHERE, not WHY. `last_age_s` plus Gate 2's
  existing died-mid-flight traces narrow the why; if gate attribution alone under-determines the
  mechanism after ~10 verdicts, a follow-up (finer-grained stamps inside Gate 3, or transcript
  correlation) is a new, informed decision — explicitly out of this sprint.
- **R3 (unstamped legitimate aborts):** a Gate-0 abort path that fails to stamp `abort` reads as
  `REAPED at=gate-0` — a false alarm, never false silence; the episode gate caps its noise.
- **R4 (rig-copy skew):** the live plist runs the driver from the pin, and SKILL.md is read from
  the mission checkout — both update on the normal pin/checkout advance; until the pin advances
  past this commit, fires classify as before (no stamps → but also no verdict block, so no false
  verdicts). No flag-day.
- **R5 (codex/pi skill compliance):** non-claude controllers follow SKILL.md by instruction, not
  enforcement; their compliance with stamping is exactly as good as their compliance with the
  gates themselves — which is what the instrument measures anyway.
- **R6 (vacuous pass invisible at exit):** with record-advancement out of the predicate (Q4,
  Finding 4), a controller that skips Gate 4 yet stamps `complete` classifies `COMPLETED` at exit.
  This is detectable one iteration later (Gate 1 observes no new mission-log record) and would be
  restored at exit by the same bounded-fetch machinery as R7's queue row — knowingly deferred
  rather than silently assumed away. The alternative (keeping the dead working-tree instrument)
  produced a false alarm on *every* healthy iteration, which is strictly worse.
- **R7 — PRE-EXISTING DEFECT, routed as a queue row, NOT this sprint's scope:** the driver's
  late-kill branch (line ~1041, "AFTER recording itself") compares `pre_last_record` vs
  `post_last_record` read from the **pin's working tree**, which is frozen for the whole run
  (V24) while records land via PR to `origin/dev` (V25) — so its detector can essentially never
  fire, and has fired **0** times against 10 rc≠0 exits (V26). Consequence: every genuine
  late-kill-after-landed-work is misreported as a lost iteration (the exact iter-145 failure the
  branch was built to prevent). **Recommended queue row**: re-read the record via a *bounded*
  `git fetch` + `git show origin/dev:<log>` (the V25-proven reader) outside any guard-held
  section, with fetch failure degrading to a named "record-unverified" outcome, never a false
  alarm. This design merely *explains* the defect; per mission rules a pre-existing defect
  surfaced during design is a queue row, not a scope expansion.

- **R8 — PRE-EXISTING DEFECT, routed as a queue row, NOT this sprint's scope:** the driver's
  *existing* exit-path notifications at `tools/launchd/mission-control.sh:1046` and `:1063` call
  `ailang messages send controlplane` **unbounded**, on the exit path, with no deadline — the same
  class gpt5-6-sol's round-2 objection raises against phase 2. A bounded wrapper (`_mc_bounded()`,
  line 194, V27) is already present in the file and already used 8 times, so the fix is a wrapper
  swap, not new machinery. This design routes phase-2 notification through the bounded path and
  leaves the two pre-existing call sites alone; converting them is a separate, cheap queue row.
  Named rather than absorbed, per the mission's rule that a pre-existing defect surfaced during
  design is a queue row and not a scope expansion.

- **R9 — `UNCLASSIFIED` IS SPECIFIED AND UNREACHABLE; `CRASHED` ABSORBS IT. Found by the round-3
  evaluator, reproduced first-party, filed as a queue row.** This doc names `UNCLASSIFIED` five
  times — in the Q4 decision table's final row and, more importantly, in **Q6's anti-vacuity
  argument**, which reads *"its fall-through is `UNCLASSIFIED`, which is loud"*. The shipped
  classify `case` ends `*:*) _mc_slot_verdict="CRASHED at=..."`, and `*:*` matches unconditionally
  once a colon is present, so it absorbs both *"other ≠0"* and *"anything else"*. Measured:
  `grep -c UNCLASSIFIED` = **0** in `tools/launchd/mission-control.sh` against **5** in this doc
  (control: `CRASHED` = 1 in the driver).
  **The floor property still holds** — `CRASHED` is loud and is not a checkmark, so nothing passes
  vacuously — but the doc and the code disagree, and the distinct *"instrument confusion at rc=0"*
  signal the table specifies is silently relabelled as *"the process crashed"*. Exposure is narrow
  by construction: reaching it needs a heartbeat file whose last stamp is outside the closed label
  set, and the writer validates labels. Present unchanged since M2 (`6c53a0a20`). Fix is a queue
  row — either make the branch reachable or delete it from the table and say why — not a re-open.

- **R10 — the helper's unknown-label rejection has no test.** `mission-heartbeat.sh` refuses an
  out-of-enum label (`unknown label`, exit 2), and the suite never exercises it: `grep -c 'unknown
  label'` = **1** in the helper, **0** in `test_mission_heartbeat.sh` (control: the suite mentions
  `stamp` **22** times). This is rule 3j's shape — a refusal branch with no killer — on the one
  guard that keeps R9's unreachable state unreachable. Cheap queue row.

## Round 3 — narrow-refinement carve-out (controller, iteration 315)

Round 2 returned **BLOCKED, 3/3 reject**, with all three reviewers PRESENT (`oc-glm-5-2` was
restored after being `ABSENT (invalid)` in round 1, so round 2 was decided at full strength, not
N−1). Every remaining blocking objection carried a concrete reviewer-authored `proposed_fix`
(`has_fix=true` on all three) and none disputed the design DIRECTION — they were completeness,
placement and premise-verification objections. That is exactly the Gate-2 narrow-refinement
carve-out, so the controller applied the reviewers' **verbatim** fixes rather than parking, and
did NOT spend a second designer run (so the Fable diet is not exceeded).

| # | reviewer | objection | disposition |
|---|---|---|---|
| 1 | `gemini-3-1-pro` | Topological impossibility: the block cannot be *inside* the rc branches AND *before* `rm -f "$PIDFILE"`, because line 1037 is on the common path and the branch starts at 1039 | **CONFIRMED first-party** (`sed -n 1036,1041p`: 1037 `rm -f`, 1039 `if [ "$RC" -ne 0 ]`). Reviewer's fix applied verbatim at both sites (Q4 and Files to Modify) |
| 2 | `gpt5-6-sol` | Phase 2 permits unbounded network I/O on the exit path, violating the bounded-waits axiom | **VALID.** Fix applied verbatim. Its conditional fallback does not trigger: `_mc_bounded()` exists at line 194 with a deadline + `kill -9` + `exec` (V27) |
| 3 | `oc-glm-5-2` | Finding 4's premise is unverified for the CONTROLLER — V24 covers only the driver; if the controller merges its record-PR into the pin, record-advancement is not dead | **MEASURED AND REFUTED** (V28), which is the correct disposition for a premise objection: the controller measured its own mid-fire behaviour rather than arguing. The pin is DETACHED and holds only the untracked design doc; the record lives in a separate worktree per SKILL.md:3425. The reviewer was right that the arm was missing; it is now present, and it confirms the original conclusion |

**A note for whoever plans this doc.** Rounds 1 and 2 both blocked, and in both the objections were
about *where exactly a shell block goes* rather than about the mechanism. That is a signal about
this doc's granularity, not about the mechanism: a design doc that pins exact line numbers in a
67 KB shell script buys a new placement defect every round. The line anchors here are evidence of
where the seams are, not an instruction to patch by line number — the sprint should key off the
`# --- SLOT VERDICT START/END ---` markers and the named functions.

**DESIGN_DOC_PATH**: `design_docs/planned/v1_0_0/m-mission-slot-heartbeat.md`
