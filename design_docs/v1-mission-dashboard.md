# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the log.*
**Last updated: 2026-09-02, iteration 317.**

## Where we are against the bar
- **Goal (D-51, ratified attended 2026-09-01):** *design docs remaining before v1.0.0.*
- **N = 10 remaining (was 10, ±0 — goal unmoved).** Iteration 317 was HARNESS: it restored a red
  CI gate and pinned nothing new against the bar. A named **UNCLASSIFIED bucket of 4** would make
  it 14; ruling on those 4 is **D-53**, the only open decision.

## In flight
- **Nothing in sprint.** Sprint 1 of `m-registry-interface-hash-blind-to-signatures` completed at
  iteration 316 (M1–M5). Sprint 2 (M6–M9) stays **deferred** — its blast-radius precondition needs
  a live registry this loop cannot reach.

## Up next (banked, top of queue)
1. `m-docparse-v0340-reports-2026-09-01` — **VERIFY-then-route.** Three defects from a real
   downstream consumer at v0.34.0. The stale-iface-cache bug does **not** reproduce in the two
   obvious shapes (recorded so nobody re-buys the easy half); finish the repro before any sprint.
2. `m-probe-derace-has-no-killer` — the fix that cleared this iteration's dev red is pinned by
   nothing; a full revert passes 42/42. Small, and it protects work already shipped.
3. `m-message-watcher-windows-wallclock-flake` — same class as this red (an absolute wall-clock
   bound on a machine-speed-dependent stimulus); cheaper fixed alongside item 2.

## Loop health
- **dev CI: GREEN again** as of `20cce785e`. It was red on `75a2b8b40` (`launchd drivers (bash 3.2)`,
  a wall-clock race, now fixed) — V1 owns this repo, so the red outranked the queue.
- **SonarCloud red on 8 of 8 commits walked back** — inherited, not required, not caused by any
  recent merge. Recorded, deliberately not picked; nobody has looked at it in 8 commits.
- **Running skill was 147 lines behind origin** — the largest drift recorded; delta read and applied.
  Main checkout **22 behind / 9 ahead, 4 dirty files** incl. an uncommitted `SKILL.md` edit. The
  reconcile is still a **human decision**; the drift is one-way and growing.
- Reaped slots continue: a prior slot of *this* iteration number died holding a finished, green PR.

## Routing / cost
- Controller opus. **No designer, planner or executor ran** — the work was inherited from the dead
  slot. Evaluator `sonnet`, own worktree, PASS 87/100 zero blocking. generator≠judge held.
- **Metered $0.00** of the $5 ceiling. Every lane a quota bucket; no quorum round.

## Parked on Mark
- **D-53** — rule on the 4 UNCLASSIFIED docs (N=10 vs N=14). Loop recommends 1 and 2 IN, 3 and 4
  OUT → N=12. Default if unanswered: keep reporting N=10 with the bucket named. Nothing stalls.
