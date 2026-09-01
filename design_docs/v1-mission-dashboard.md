# Mission Dashboard — V1

**Last updated: 2026-09-01, iteration 314.**

## Where the mission is

- **Active sprint**: `m-registry-interface-hash-blind-to-signatures` — Sprint 1, **4 of 5 milestones done**
  (M1 iter-311, M2 iter-312, M3 iter-313, M4 iter-314 → PR #1002). **M5 ≈ 1 day remains.**
- **Sprint 2 (M6–M9) stays DEFERRED**: its blast-radius measurement needs the live registry, which
  the loop's session cannot reach. Shipping M6 without it ships an unbounded regression.
- **The defect**: `InterfaceHash` folds no signature data, so add/remove/retype of an export leaves
  it byte-identical and the release ships as `patch` (seen on `sunholo/external_backend`).

## Next picks (banked, in order)

1. **Sprint 1 M5** — signature-set classification with the `U` class, on the plan's 2×2 (which
   sides carry signatures), not D5's 1-D test, which would stall every cascade the day it lands.
2. `m-registry-validator-unbounded-compile` — a public HTTP server compiles untrusted uploads with
   `exec.Command` (no deadline) at three sites. Confirmed at HEAD, pre-existing, security-shaped.
3. `m-weekly-sweep-orphans-2026-08-31` — triage-lite this week's 5 zero-mention open issues
   (#963, #962, #960, #959, #941); ghost-discipline each at HEAD before routing.

## Loop health

- **Cadence**: launchd, pinned worktree `~/.ailang-driver-pin/v1`, ~16 fires/day. **Routing**:
  controller opus · designer rotation (fable ↔ deepseek-v4-flash) · planner/executor
  `codex:gpt-5.6-sol` · evaluator `sonnet`; generator≠judge enforced on **provider**.
- **Cost**: iteration 314 metered **$0.00** of the $5 ceiling (recent: $0.00–$0.24). **Reaped-slot
  rate ≈ 40%** — 6 of iterations 296–310 produced no record. See **D-52**.
- **Standing divergence**: main checkout **11 behind / 3 ahead**, 4 dirty files; routed around each
  iteration, reconcile is a human decision. **CI**: `SonarCloud` red on dev — **inherited** (8 of 8
  commits walked back), not a required context.
## Parked on Mark (both OPEN in the decision ledger — `scripts/mission_decisions.sh --open`)

- **D-51** — ratify or replace the charter's countable finish-line unit. The provisional unit
  (open queue rows) is **anti-correlated with good work**. Loop recommends **(b) milestone
  burn-down**. Unanswered ⇒ status quo, every Progress line stays misleading.
- **D-52** — is the ~40% reaped-slot rate worth one iteration to diagnose? Loop recommends **(a)**, a
  per-gate heartbeat artifact so a dead slot names the gate it died in. Unanswered ⇒ **(b)**, keep
  recovering by hand via the Gate-2 died-mid-flight traces.
