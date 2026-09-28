# Fleet Mission — the loop harness is fixed by one loop, so the product loops never have to

<!--
  From design_docs/mission-charter-TEMPLATE.md. Design: design_docs/planned/m-harness-mission-loop.md
  (HD-1..HD-6 ratified by Mark 2026-09-26). Iteration 0 = ratify this charter with Mark, attended.
-->

**Type**: Long-running mission (peer of [v1-mission.md](v1-mission.md), [docs-mission.md](docs-mission.md)); advanced by a scheduled
outer loop on the always-on rig. **It fires only when there is open work.** The driver exits
before any probe or spawn when `ailang mission ticket open --count` is 0.
**North star**: product loops spend their iterations on product work. Every loop-harness defect
they hit leaves them as a ticket and comes back as a fix.
**Traces to**: [PROGRAM.md](PROGRAM.md) and [M-HARNESS-MISSION-LOOP](planned/m-harness-mission-loop.md).
Measured basis: World iterations 164–180 were 12/13 `[HARNESS]`; V1 309–348 were 18/35.
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md), the SAME
unforked skill. Gate 2's **fleet branch** (`resources/gate-2-pick.md`) inverts admissibility for this
mission.
**Scheduling**: launchd `dev.ailang.mission-fleet`, registry entry [`missions/fleet.toml`](../missions/fleet.toml)
(interval 6h, boot offset 1680s). Billing guard as for every mission.
**Log**: [fleet-mission-log.md](fleet-mission-log.md), append-only, one entry per iteration.
**Human-facing reporting**: GitHub issue #1380 (rotated 2026-09-28 from #1321; live number in
`~/.ailang/state/mission-fleet-gh-issue`).

## Repo Profile (M-MISSION-PORTABILITY M2 — the per-mission values mission-control reads)

- **Repo slug**: `sunholo-data/ailang` (driver: `MISSION_REPO`)
- **Mission doc**: `design_docs/fleet-mission.md` (driver: `MISSION_DOC`)
- **Mission name / state namespace**: `fleet` (driver: `MISSION_NAME`; `~/.ailang/state/mission-fleet-*`)
- **Checkout**: work happens in the **pin worktree** `~/.ailang-driver-pin/fleet` (detached at the
  pinned `origin/dev`, fresh each fire). Every mission whose work repo IS `sunholo-data/ailang` runs
  this way (v1, docs, motoko; `pin-root.sh` `_set_pin_workdir`). The clone
  `~/dev/sunholo-data/ailang-fleet` is only launchd's working directory, as `ailang-docs` is for docs.
  Land changes through a branch or PR, never by editing the pin worktree in place.
- **Bookkeeping issue**: `#1380` (from `#1321`), rotates weekly; live number in `~/.ailang/state/mission-fleet-gh-issue`
- **CI workflows Gate 3b / Gate 1 poll**: `CI` (runs on every push; no push paths filter).
- **Verify profile**: `go-compiler`, plus the mission-loop-change pre-flight as the done-gate
  (below). Harness changes live in `tools/launchd/` (bash 3.2), `internal/mission/` and
  `cmd/ailang/mission*` (Go), and the mission skills.

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `fleet-mission-status-archive.md`.

## STATUS 2026-09-28 — ITERATION 6: iteration 5's two fixes **re-judged (PASS 92, PASS 96) and LANDED in [#1377](https://github.com/sunholo-data/ailang/pull/1377) (`eec86ca4f`)**; both tickets resolved

Iteration 5 (07:56 fire) built both fixes and crashed at Gate 3b on an API DNS error (`ENOTFOUND`,
slot verdict `CRASHED_at=gate-3b`) with no record. The 23:09 fire before it died in the Aqua-session
loss that its own pick is about. This iteration resumed rather than redid that work. **(1)** The
`blocking=all` ticket `rig:aqua-session-lost:windowserver-watchdog`: `cron-kicker.sh` now checks the
gui domain first, logs `SESSION-LOST` once and sends one bounded, fail-soft notice. On restore it
re-stamps state, and it derives mission labels from `missions/*.toml`, so fleet and stapledon are
covered. **(2)** P0 #4 `pi-runner:sandbox-extensions-not-wired` (D-FLEET-6): fail-closed extension,
`-e` wiring, rc 15/16/17. Fresh sonnet evaluators: **PASS 92** and **PASS 96**, 0 blocking. Each
single-fix PR conflicted on the changelog within minutes, and the scope guard refuses a rebased
push. So both commits land in one PR, with their entries at the END of `[Unreleased]`. #1377 is
MERGEABLE, and every check except `lint` is green or pending. `lint` is required and red on dev itself
since `1fcc479f1` (a direct push; `internal/executor/motoko/healthcheck.go` gofmt), which is outside
the fleet's scope, so it is sent to `mission-v1`. **Update, same iteration:** `96fd6c5e1` fixed dev `lint`. The record push re-tested, all 4 required
checks went green, and Mark merged #1377 at 15:26Z (`eec86ca4f`). On the merge commit, `test`, `lint` and
`launchd drivers (bash 3.2)` are all success. Both tickets are resolved with that SHA (19 open). Clause map: **1 product share** unmeasured; **2
turnaround** at risk (two tickets resolved; a product red delayed them, the third time); **3 one queue** MET
(19 open); **4 idle is free** MET; **5 no regressions** MET (nothing landed unverified).

## STATUS 2026-09-27 — ITERATION 4: P0 #2 M1 **measured**. On the rig, `ps -S` cannot see reaped-child CPU; `proc_pid_rusage` separates the drill from both wedges. Threshold returns to Mark as **D-FLEET-7**. Evaluator PASS 87.

#1329 and #1330 are **merged** (`c912320fe`, `4d8dff929`), so P0 #3's commit-blind half and P1 #0 have
landed. At Gate 1, dev `CI` was red on one `launchd drivers (bash 3.2)` timing assertion
(`test_driver_notify.sh`, "hanging gh comment", elapsed 8 s against a 7 s limit). A rerun was green:
that is the `ci:launchd-driver-suite-flakes` class (P2 #11), now reproduced once on dev.
**P0 #2 M1** (D-FLEET-4): a new `tools/launchd/measure_stall_cpu.sh` plus a `proc_rusage.py` helper.
Both are measurement only; the live watchdog is unchanged. **Finding:** on macOS 26.6.2, `ps -S` reads
`0:00.00` for a parent whose reaped child burned about 4 CPU-s, so the plan's candidate instrument
cannot work. Per ~121 s window, rusage (self plus reaped children) reads: drill **62.5–97.7**, w1-git
**0.73–1.72**, w1-gh **0.22–0.31**, w2 **0**. Idle `claude` roots accrue **0.57–1.60** in the same
window, so an arm must exclude the root. Recommendation in D-FLEET-7: 10 CPU-s per window over the
descendants only. Branch `fleet/iter4-stall-cpu-measure`; PR and merge at Gate 3b. Clause map: **1 product share**
unmeasured; **2 turnaround** at risk (P0 #2 back on Mark; P0 #4 routable); **3 one queue** MET (17
open, one new: `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive`); **4 idle is free**
MET; **5 no regressions** MET (nothing live changed).

## STATUS 2026-09-27 — ITERATION 3: P0 #4 **PARKED (D-FLEET-6: fix needs `tools/pi-extensions/**`)**, plan committed; P1 #0 driver-env scrub **built, evaluated PASS 97, merge blocked on the inherited dev red** ([#1330](https://github.com/sunholo-data/ailang/pull/1330))

**P0 #4** `pi-runner:sandbox-extensions-not-wired`: REAL at HEAD (`scripts/mission_pi_run.sh:165` passes no
`-e`). The codex planner verified P1–P5 and found more: `tools/pi-extensions/sandbox/index.ts` **fails
open**. When `SandboxManager.initialize` throws, bash runs unsandboxed (`:227`, `:290`, controller- and
evaluator-verified). Other findings: the rig's policy sits at a path pi 0.85.1 never reads, sandbox-runtime
resolves only from the main checkout, and linked-worktree commits would be fenced. So M1 must edit
`tools/pi-extensions/**`, which is outside the allowlist, and D-FLEET-3 already reserved this ticket for Mark.
Plan: `design_docs/planned/sprint-plan-pi-runner-sandbox-wiring.md`; decision **D-FLEET-6**. **P1 #0**
`tests:driver-env-leaks-into-launchd-suite` (Mark's directive): every `test-launchd-drivers` suite now runs
through the allowlisted `tools/launchd/lib/suite-env.sh`. In the live fire env, base rc=2 (routing 86/1)
and fix rc=0 (87/0). Evaluator (sonnet) **PASS 97, 0 blocking**; the CI `launchd drivers (bash 3.2)` job
is green. **Not merged:** `test` is red only on the inherited `builder.go` 801 > 800, and `test-windows`
is the inherited Go-suite red. Resume predicate, shared with #1329: dev `test` green → re-run CI →
squash-merge both. Clause map: **1 product share** unmeasured; **2 turnaround** at risk (three items wait:
P0 #2 and P0 #4 on Mark, #1329 and #1330 on V1's red); **3 one queue** MET (16 open tickets, none new);
**4 idle is free** MET; **5 no regressions** MET (nothing landed unverified).

## CURRENT GOAL

1. **Iteration 0 (definition)**: DONE 2026-09-26, ratified as written.
2. **Then**: each fire takes the top open ticket through the inner loop (design only when the fix
   warrants one, then plan, execute, evaluate), lands it on `dev`, and resolves it.

## The bar (RATIFIED 2026-09-26)

- **Clause 1 — product share**: `[HARNESS]`-tagged share of product-loop iterations ≤ 10% over a
  rolling 14 days (design goal 1; measured by header scan of each `<name>-mission-log.md`).
- **Clause 2 — turnaround**: median ticket filed → resolved ≤ 48h for non-policy tickets.
- **Clause 3 — one queue**: every escalation is findable in `mission-fleet` (no scattered issues).
- **Clause 4 — idle is free**: a fire with no open tickets spends zero controller tokens.
- **Clause 5 — no regressions shipped**: every resolved ticket passed the done-gate below.

## Authority (the write scope — enforced by `tools/launchd/githooks/pre-push`)

- **May change**, as an allowlist enforced by the guard: the loop harness (`tools/launchd/**`,
  `internal/mission/**`, `cmd/ailang/mission*`, `missions/**`, `.claude/skills/mission-*/**`,
  `.claude/skills/sprint-*/**`, `.pi/extensions/**`, `scripts/hooks/**`, `scripts/mission_*`,
  `scripts/test_mission_*` (D-FLEET-5), its own
  `design_docs/fleet-mission*`), plus support paths (`design_docs/**`, `changelogs/**`,
  `CHANGELOG.md`, `docs/**`, `.ailang/state/sprints/**`, `internal/config/mission.go`, `make/test.mk`,
  `tools/pi-extensions/sandbox/**` (D-FLEET-6)).
  **Anything else is refused**, including CI workflows, `go.mod`, the server, the UI and non-mission
  commands. A fix that genuinely needs one of those parks for Mark.
- **May not change** (language core, refused by the guard): `internal/{parser,lexer,ast,types,
  elaborate,core,eval,vm,codegen,effects,builtins,pipeline,runtime,link,iface}/**`, `std/**`,
  `examples/**`, `benchmarks/**`.
- **Parks for Mark (HD-2a, policy class)**: any fix that changes routing, lane order, quota or
  ration thresholds, or a billing guard. **Also any change to `.pi/extensions/**`** (D-FLEET-3): the
  pi EVAL harness loads the same extensions, so a fleet fix there can shift eval results unnoticed.
  Such a ticket parks with the proposed diff and its expected eval impact. File a decision row with a recommendation; take the next
  ticket.

## Guardrails (on top of the skill's Standing Rules)

- **Ticket-driven only.** The queue is `ailang mission ticket open --json` plus Mark's directives.
  No self-sourced audits, no "while I'm here". The 2026-09-21 attended hunt found harness defects
  faster than they could be fixed and never ran out.
- **Unread = open.** Never `ailang messages read` on `mission-fleet`. Close a ticket only with
  `ailang mission ticket resolve <signature> --resolution … --sha <commit on origin/dev>`.
- **Done-gate = the mission-loop-change skill's five-line pre-flight**: the surface that actually
  runs was edited; reach was confirmed (pushed, since every mission runs the pinned `origin/dev`);
  dry-run healthy **and** degraded (set `AILANG_DRIVER_PINNED=<sha>` so the pin cannot re-exec into
  another copy, design V15); `make test-launchd-drivers` green; nothing reloaded mid-iteration.
- **Every mission runs `origin/dev` until Phase 3b** (`harness-stable`). A fleet push reaches all
  loops on their next fire, so the done-gate is not optional.

## Routing policy

Shared per-role routing from `mission-control`. Overrides in `~/.config/ailang/mission-fleet.env`:

- **Executor default**: shared default (codex, then pi, then opus).
- **Evaluator**: shared default (`sonnet`); generator ≠ judge is enforced by the skill.
- **Opus before pi**: on (`MISSION_OPUS_BEFORE_PI=1`), same as World. Harness fixes are
  high-blast-radius, so capability beats flat-rate here.

## Decisions (policy rulings the fleet may act on; HD-2a)

| ID | Status | Ruling |
|---|---|---|
| D-FLEET-1 | RULED 2026-09-26 (Mark, attended: "yes") | Design docs without `planner_lane` default to the mission's planner pin, not `opus fail-closed`. |
| D-FLEET-2 | RULED 2026-09-26 (Mark, attended: "yes") | The spawn-pin hook may walk the role's DECLARED fallback chain, in order, when the pinned model is dead. No undeclared model is ever allowed. |
| D-FLEET-3 | RULED 2026-09-26 (Mark, attended: "do the follow ups") | Changes to `.pi/extensions/**` park for Mark with the diff and expected eval impact; the pi eval harness shares them. Affects `pi-runner:sandbox-extensions-not-wired` (P0 #4): the fleet designs it, and Mark approves before it lands. |
| D-FLEET-4 | RULED 2026-09-27 (Mark, attended: "go with the recommendations") | **YES to M1 only.** Measure `ps -S -o time` CPU deltas on the rig for the long-drill shape and both reference wedges (plan: `design_docs/planned/sprint-plan-stall-descendant-progress.md`), then bring a MEASURED threshold back as a new decision row. Until that row is ruled, thresholds and sample counts are unchanged; the provisional ≥2 CPU-s / 120 s is not live. |
| D-FLEET-5 | RULED 2026-09-27 (Mark, attended: "go with the recommendations") | **YES.** `scripts/test_mission_*` is a harness path: the fleet may change it, product loops are refused on it (`tools/launchd/githooks/pre-push`, `_scope_is_harness`). The fleet may now repair `scripts/test_mission_pi_run.sh` TEST 3. |
| D-FLEET-6 | RULED 2026-09-27 (Mark, attended: "go with the recommendations") | **YES to M1–M3 as one fleet sprint** (`design_docs/planned/sprint-plan-pi-runner-sandbox-wiring.md`): the extension fails closed and reads an explicit policy file; `scripts/mission_pi_run.sh` passes `-e` with typed verdicts rc 15/16/17; tests. The guard allows the fleet `tools/pi-extensions/sandbox/**` only; the rest of `tools/pi-extensions/**` stays outside its scope because the eval harness shares it. This ruling discharges D-FLEET-3 for this ticket. |
| D-FLEET-7 | **OPEN** 2026-09-27 (iteration 4; asks Mark) | **Stall-watchdog CPU arm: approve the measured threshold?** M1 found two things. On the rig, `ps -S` cannot see reaped-child CPU; `proc_pid_rusage` (`ri_child_*`) can. And an idle `claude` root accrues 0.57–1.60 CPU-s per 120 s. **Recommendation: YES to M2 with this arm:** count a window as progress when the cumulative rusage CPU of the controller's DESCENDANTS, excluding the root, grows by **≥10 CPU-s per 120-s sample**. Measured windows: drill 62.5–97.7, w1-git ≤1.72, w1-gh ≤0.31, w2 0 (`design_docs/planned/sprint-plan-stall-descendant-progress.md`, M1 result). It needs `python3` (ctypes) in the driver path. Sample counts and the 600 s budget stay unchanged. Risk: a W1 whose poll condition is itself heavy (for example a `go test`) would read live. **Default until answered:** nothing changes; the fleet takes the next ticket. |

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

The live tickets are **`ailang mission ticket open`**. This section is **Mark's triage order**
(attended, 2026-09-26) for the 16-ticket backfill. It **outranks** the `slots_lost` ranking,
because every backfilled ticket has exactly one occurrence, so that ranking degenerates to filing
order. New tickets filed after today rank by `slots_lost` BELOW this list unless they are
`blocking=all`, which jumps the queue. Always re-check a ticket at HEAD in Gate 2 before working it,
and resolve it as "already fixed" with evidence if it no longer reproduces.

**P0 — loops lose whole slots or run unsafe today**
1. [LANDED 2026-09-26 iter 1, #1325 `e3dadcd07`] `driver:slot-kill-leaves-orphan-descendants`: a killed slot leaves its descendants
   running (world 2026-09-26: the planner's harness ran 26 min past the kill). Leaks processes on a
   box with an OOM history.
2. [M1 DONE iter 4: `ps -S` blind on the rig; rusage separates drill (62.5–97.7 CPU-s per window) from w1 (≤1.72) and w2 (0). Threshold = **D-FLEET-7 OPEN**; M2 waits on it] `stall-watchdog:kills-controller-on-long-drill`: **mechanical half only.** Count a live,
   progressing descendant as progress, as 4a86ea17b does for pi. **Changing the 600s threshold
   or the sample counts is POLICY: park it.** Cost world a whole slot today (04:45, rc 143).
3. [commit-blind half LANDED #1329 `c912320fe`; pre-dirty half = V1 D-58 design, not started; ticket stays open] `pi-runner:verdict-blind-to-commits-and-predirty`: the pi fallback lanes (now the tail after
   opus) report `empty_worktree` for executors that commit, so a working lane reads as dead.
4. [LANDED 2026-09-28 iter 6, #1377 `eec86ca4f`; ticket resolved] `pi-runner:sandbox-extensions-not-wired`: pi roles run unfenced. Safety, not throughput. The extension itself fails open, so the fix needs `tools/pi-extensions/**`.

**Queue-jumper (`blocking=all`, filed 2026-09-28 by stapledon)**
- [LANDED 2026-09-28 iter 6, #1377 `eec86ca4f`; ticket resolved; kicker reach needs the main checkout updated] `rig:aqua-session-lost:windowserver-watchdog`: WindowServer was watchdog-killed and `gui/501` vanished, so every mission stopped for about 8 h and the kicker logged nothing. The fix detects and reports the class, and derives kicker labels from the registry. Recovery still needs a GUI login; the ticket asks only for "document or handle". **Reach:** the crontab runs the kicker from the main checkout, so the fix takes effect only once that checkout's `dev` includes the merge.

**P1 — silent wedges and invisible failures**
0. [LANDED #1330 `4d8dff929` (directive, not a ticket)] **[DIRECTIVE, Mark via attended session 2026-09-26, from fleet iteration 1's own friction (a)]**
   `tests:driver-env-leaks-into-launchd-suite`: inside a fire, the driver's exported `MISSION_*`
   env reaches `make test-launchd-drivers`, and `test_mission_routing` reads it as
   "unparsable-path-entry". So the fleet's own done-gate is red at base on every fire, and each
   iteration has to prove the red is not a regression. Fix it at the suite boundary (scrub mission
   env per suite, or in the make target), not per test. The `GIT_CONFIG_*` half was fixed attended
   in `test_mission_scope_guard.sh`.
0a. **[RULED D-FLEET-1]** `resolver:planner-lane-field-missing-vs-spawn-pin`: a design doc with no
   `planner_lane` field uses the mission's planner pin instead of `opus fail-closed`. It is
   hand-overridden on every World fire today.
0b. **[RULED D-FLEET-2]** `spawn-pin-hook:no-fallback-mode`: when the pinned role model is dead, the
   spawn-pin hook allows the role's declared `MISSION_<ROLE>_FALLBACK` chain, in order and nothing
   else, so the pin still means something.
5. `driver:exit-path-notices-unbounded`: unbounded sends on the exit path can hang a fire.
6. `gate0:driver-crash-notices-invisible`: loops cannot see their own crash notices.
7. `quorum:artifact-dir-cwd-relative`: reviewed docs read as unreviewed from pin worktrees.
8. `quorum:invalid-absent-on-quoted-literals`: a real reviewer counted absent.
9. `quorum:zero-signal-guard-vacuous-with-controller-verdict`: verify first (never re-checked).

**P2 — hygiene**
10. `mission:rotate-log-registry-cwd`
11. `ci:launchd-driver-suite-flakes`: verify it reproduces before working it.
12. `skills:agents-copies-stale`: mission-*/sprint-* copies only (the fleet's scope).
13. `driver:unclassified-state-unreachable`
14. `skill:heartbeat-relative-path-absent-in-world`: **likely stale**. World's slot verdicts
    show `stamps=9` on every completed 09-25/26 fire, so stamps land. Verify, then resolve as not
    reproducing.

**Ruled policy questions** — both moved into P1 above (items 0a and 0b; decisions D-FLEET-1 and D-FLEET-2).

**[DIRECTIVE, Mark 2026-09-26] After P1, before P2:** the Phase 3a skill-resolution spike
(design doc M-HARNESS-MISSION-LOOP, premises P1–P3). Measure how the claude, codex and pi controllers
each resolve skills, using marker skills in the pin worktree against conflicting user-level copies.
**Measurement only**: record the results in the design doc's Verification Log and park Phase 3b's
mechanism choice for Mark. This is what unblocks `harness-stable`.

---
**Document created**: 2026-09-26. Iteration 0 ratifies it with Mark before any ticket routes.
