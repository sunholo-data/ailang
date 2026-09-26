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

## STATUS 2026-09-26 — ITERATION 1: **P0 #1 LANDED**, `driver:slot-kill-leaves-orphan-descendants` ([#1325](https://github.com/sunholo-data/ailang/pull/1325), `e3dadcd07`)

Both watchdogs now reap the controller's whole process tree through `_mc_kill_tree`, which
snapshots before TERM, re-walks before KILL and skips recycled PIDs. A triggered watchdog is no
longer cancelled mid-grace. Evaluator (sonnet) PASS 87, 0 blocking. Ticket resolved. Thresholds
are untouched. Clause map: **1 product share** unmeasured (no product-loop window since the
charter); **2 turnaround** first datum ≈3h (filed 14:50Z); **3 one queue** MET (16 tickets, all in
`mission-fleet`); **4 idle is free** MET by construction (driver pre-check, iteration 0 dry run);
**5 no regressions** MET for this ticket, with one caveat: the done-gate's dry-run line cannot
reach `_mc_run_once` edits (UNINFORMATIVE, see log). Next: P0 #2, mechanical half.

## STATUS 2026-09-26 — ITERATION 0: **charter RATIFIED as written** (Mark, attended)

Mark: *"yes as written"* — the bar (clauses 1–5), Authority and Guardrails below stand unchanged.
Kill switch lifted the same session. The loop idles at zero cost (driver pre-check) until the first
ticket is filed to `mission-fleet`. Plane: `mission-fleet` is declared triage in prod
(ailang-multivac d2f277d, promoted 2026-09-26).

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
  `.claude/skills/sprint-*/**`, `.pi/extensions/**`, `scripts/hooks/**`, `scripts/mission_*`, its own
  `design_docs/fleet-mission*`), plus support paths (`design_docs/**`, `changelogs/**`,
  `CHANGELOG.md`, `docs/**`, `.ailang/state/sprints/**`, `internal/config/mission.go`, `make/test.mk`).
  **Anything else is refused**, including CI workflows, `go.mod`, the server, the UI and non-mission
  commands. A fix that genuinely needs one of those parks for Mark.
- **May not change** (language core, refused by the guard): `internal/{parser,lexer,ast,types,
  elaborate,core,eval,vm,codegen,effects,builtins,pipeline,runtime,link,iface}/**`, `std/**`,
  `examples/**`, `benchmarks/**`.
- **Parks for Mark (HD-2a, policy class)**: any fix that changes routing, lane order, quota or
  ration thresholds, or a billing guard. File a decision row with a recommendation; take the next
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
2. [NEXT] `stall-watchdog:kills-controller-on-long-drill`: **mechanical half only.** Count a live,
   progressing descendant as progress, as 4a86ea17b does for pi. **Changing the 600s threshold
   or the sample counts is POLICY: park it.** Cost world a whole slot today (04:45, rc 143).
3. `pi-runner:verdict-blind-to-commits-and-predirty`: the pi fallback lanes (now the tail after
   opus) report `empty_worktree` for executors that commit, so a working lane reads as dead.
4. `pi-runner:sandbox-extensions-not-wired`: pi roles run unfenced. Safety, not throughput.

**P1 — silent wedges and invisible failures**
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

**[PARKED] — routing policy, questions for Mark (HD-2a)**
- `resolver:planner-lane-field-missing-vs-spawn-pin`: should a design doc with no `planner_lane`
  field default to the mission's planner pin instead of `opus fail-closed`? Recommendation: yes.
  Almost no doc carries the field, so fail-closed is the common case, and it is hand-overridden
  every World fire.
- `spawn-pin-hook:no-fallback-mode`: should the spawn-pin hook allow the role's declared
  `MISSION_<ROLE>_FALLBACK` chain when the pinned designer is dead? Recommendation: yes, walking
  only the declared chain in order, so the pin still means something.

Standing parked item: the Phase 3a skill-resolution spike (design P1–P3), routed here only on
Mark's directive.

---
**Document created**: 2026-09-26. Iteration 0 ratifies it with Mark before any ticket routes.
