# Fleet Mission — STATUS archive

Older STATUS stamps rotated out of [fleet-mission.md](fleet-mission.md) (newest 3 stay there). Append-only.

## STATUS 2026-09-26 — ITERATION 0: **charter RATIFIED as written** (Mark, attended)

Mark: *"yes as written"* — the bar (clauses 1–5), Authority and Guardrails below stand unchanged.
Kill switch lifted the same session. The loop idles at zero cost (driver pre-check) until the first
ticket is filed to `mission-fleet`. Plane: `mission-fleet` is declared triage in prod
(ailang-multivac d2f277d, promoted 2026-09-26).

## STATUS 2026-09-26 — ITERATION 1: **P0 #1 LANDED**, `driver:slot-kill-leaves-orphan-descendants` ([#1325](https://github.com/sunholo-data/ailang/pull/1325), `e3dadcd07`)

Both watchdogs now reap the controller's whole process tree through `_mc_kill_tree`, which
snapshots before TERM, re-walks before KILL and skips recycled PIDs. A triggered watchdog is no
longer cancelled mid-grace. Evaluator (sonnet) PASS 87, 0 blocking. Ticket resolved. Thresholds
are untouched. Clause map: **1 product share** unmeasured (no product-loop window since the
charter); **2 turnaround** first datum ≈3h (filed 14:50Z); **3 one queue** MET (16 tickets, all in
`mission-fleet`); **4 idle is free** MET by construction (driver pre-check, iteration 0 dry run);
**5 no regressions** MET for this ticket, with one caveat: the done-gate's dry-run line cannot
reach `_mc_run_once` edits (UNINFORMATIVE, see log). Next: P0 #2, mechanical half.

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
