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
