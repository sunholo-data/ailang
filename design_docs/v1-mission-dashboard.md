# Mission Dashboard — V1

**Last updated: 2026-09-01, iteration 315.**

## Where the mission is

- **The charter now has a ratified finish line.** `D-51` (Mark, attended) made the countable unit
  *design docs remaining before v1.0.0*, scoped by the v1.0 bar's own cutoff rule against clauses
  2–5. **N = 10** — 7 existing docs + 3 NEW-DOC units — published in the charter this iteration
  with a 4-item **UNCLASSIFIED** bucket that would take N to 14 if Mark rules them all in.
- **Active work**: `m-mission-slot-heartbeat` (`D-52`) — a per-gate heartbeat so a reaped slot names
  the gate it died in, and the ~40% reaped rate becomes derivable. → PR #1005.
- **`m-registry-interface-hash-blind-to-signatures` Sprint 1: M5 (~1 day) still remains**; it is the
  next pick once #1005 lands. Sprint 2 (M6–M9) stays DEFERRED — its blast-radius measurement needs
  the live registry, which the loop's session cannot reach.

## Next picks (banked, in order)

1. **Sprint 1 M5** — signature-set classification with the `U` class, on the plan's 2×2 (which sides
   carry signatures), not D5's 1-D test, which would stall every cascade the day it lands.
2. **R7 + R8** (new, from this iteration's design review, both PRE-EXISTING and both cheap): the
   driver's late-kill record detector is structurally dead on a frozen pin worktree and has fired
   **0** times in the whole driver log; and two exit-path `ailang messages send` calls
   (`mission-control.sh:1046`/`:1063`) are unbounded, while `_mc_bounded()` already exists eight
   lines up and is used 8 times. R8 is a wrapper swap.
3. `m-registry-validator-unbounded-compile` — a public HTTP server compiles untrusted uploads with
   `exec.Command` (no deadline) at three sites. Confirmed at HEAD, pre-existing, security-shaped.
4. `m-weekly-sweep-orphans-2026-08-31` — triage-lite this week's zero-mention open issues;
   ghost-discipline each at HEAD before routing.

## Loop health

- **Cadence**: launchd, pinned worktree `~/.ailang-driver-pin/v1`, ~16 fires/day. **Routing**:
  controller opus · designer rotation (fable ↔ deepseek-v4-flash) · planner lane derived
  (`opus`, `fail-closed:planner-lane-field-missing`) · executor `codex:gpt-5.6-sol` · evaluator
  `sonnet`; generator≠judge enforced on **provider**.
- **Cost**: iteration 315 metered **$0.26** of the $5 ceiling — two quorum rounds; every other lane
  a quota bucket.
- **The judge earned its keep three times this iteration.** It failed the sprint at 64/100 and again
  at 66/100, and its worst catch was a *fabricated* measurement: the suite printed
  `mutations: 7/7 killed` from a hardcoded string gated on whether a marker comment existed — and
  the controller had quoted that line as evidence in its own gate re-run.
- **Standing divergence**: main checkout behind origin with 4 dirty files; routed around each
  iteration, reconcile is a human decision. The **running skill was 58 lines behind origin** at
  Gate 1 — missing exactly the attended-ledger rule this iteration then had to apply.
- **CI**: `SonarCloud` red on dev — **inherited** (failure on 5 of 6 commits walked back), not a
  required context, not a pick.

## Parked on Mark

**None.** `D-51` and `D-52` were both answered on 2026-09-01 through the attended ledger channel
(commit `878e0a5a0`, author `mark@aitanalabs.com` — provenance verified) and both are consumed here.
The ledger reads **52 rows, 0 OPEN**.

The one thing worth a look when convenient: the **4 UNCLASSIFIED items** in the charter's new Goal
block. They are not blocking — the loop proceeds on N = 10 — but they are the difference between
N = 10 and N = 14, and only you can rule on them.
