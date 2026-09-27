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
**Human-facing reporting**: GitHub issue #1321 (live number in
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
- **Bookkeeping issue**: `#1321`, rotates weekly; live number in `~/.ailang/state/mission-fleet-gh-issue`
- **CI workflows Gate 3b / Gate 1 poll**: `CI` (runs on every push; no push paths filter).
- **Verify profile**: `go-compiler`, plus the mission-loop-change pre-flight as the done-gate
  (below). Harness changes live in `tools/launchd/` (bash 3.2), `internal/mission/` and
  `cmd/ailang/mission*` (Go), and the mission skills.

---

## STATUS (rotation rule)

Newest **3** STATUS stamps live here; older ones move to `fleet-mission-status-archive.md`.

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

## STATUS 2026-09-27 — ITERATION 2: P0 #2 **PARKED (policy, D-FLEET-4)**; P0 #3 commit-blind half **built, evaluated PASS 97, merge blocked on the inherited dev red** ([#1329](https://github.com/sunholo-data/ailang/pull/1329))

**P0 #2** `stall-watchdog:kills-controller-on-long-drill`: the codex planner evaluated five progress
arms against the 09-26 04:45 World kill and the two reference wedges. None works without a new
numeric threshold, so it is policy (HD-2a). Decision row D-FLEET-4 carries the recommendation. Plan:
`design_docs/planned/sprint-plan-stall-descendant-progress.md`. **P0 #3**
`pi-runner:verdict-blind-to-commits-and-predirty`, **commit-blind half**: `mission_pi_run.sh` now
counts commits since launch as work, and a moved HEAD alone is not work. Evaluator (sonnet): round 1
FAIL 68 (HEAD_MOVED false greens), round 2 **PASS 97, 0 blocking**. `make test-launchd-drivers`
rc=0. **Not merged:** the required `test` check is red on dev at base (`internal/iface/builder.go`
801 > 800 lines since `79af650f7`, language core, handed to V1). Resume predicate: dev `test` green
→ re-run #1329 CI → squash-merge. The ticket stays open after the merge, because its pre-dirty
half (the D-58 design) remains. Clause map: **1 product share** unmeasured; **2
turnaround** at risk (P0 #2 now waits on Mark, P0 #3 waits on V1); **3 one queue** MET (16 tickets,
one new: `pi-runner:quota-429-reported-as-empty-worktree`); **4 idle is free** MET; **5 no regressions**
MET (nothing landed unverified).

## STATUS 2026-09-26 — ITERATION 1: **P0 #1 LANDED**, `driver:slot-kill-leaves-orphan-descendants` ([#1325](https://github.com/sunholo-data/ailang/pull/1325), `e3dadcd07`)

Both watchdogs now reap the controller's whole process tree through `_mc_kill_tree`, which
snapshots before TERM, re-walks before KILL and skips recycled PIDs. A triggered watchdog is no
longer cancelled mid-grace. Evaluator (sonnet) PASS 87, 0 blocking. Ticket resolved. Thresholds
are untouched. Clause map: **1 product share** unmeasured (no product-loop window since the
charter); **2 turnaround** first datum ≈3h (filed 14:50Z); **3 one queue** MET (16 tickets, all in
`mission-fleet`); **4 idle is free** MET by construction (driver pre-check, iteration 0 dry run);
**5 no regressions** MET for this ticket, with one caveat: the done-gate's dry-run line cannot
reach `_mc_run_once` edits (UNINFORMATIVE, see log). Next: P0 #2, mechanical half.

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
2. [NEXT — D-FLEET-4 RULED 2026-09-27: M1 (measure) only, threshold returns for approval] `stall-watchdog:kills-controller-on-long-drill`: **mechanical half only.** Count a live,
   progressing descendant as progress, as 4a86ea17b does for pi. **Changing the 600s threshold
   or the sample counts is POLICY: park it.** Cost world a whole slot today (04:45, rc 143).
3. [IN-SPRINT iter 2: commit-blind half built + PASS 97 in #1329, merge waits on dev `test` green; pre-dirty half = V1 D-58 design, not started] `pi-runner:verdict-blind-to-commits-and-predirty`: the pi fallback lanes (now the tail after
   opus) report `empty_worktree` for executors that commit, so a working lane reads as dead.
4. [UNPARKED — D-FLEET-6 RULED 2026-09-27: M1–M3 as one sprint; plan in #1330] `pi-runner:sandbox-extensions-not-wired`: pi roles run unfenced. Safety, not throughput. The extension itself fails open, so the fix needs `tools/pi-extensions/**`.

**P1 — silent wedges and invisible failures**
0. [IN-SPRINT iter 3: built + PASS 97 in #1330, merge waits on dev `test` green] **[DIRECTIVE, Mark via attended session 2026-09-26, from fleet iteration 1's own friction (a)]**
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
