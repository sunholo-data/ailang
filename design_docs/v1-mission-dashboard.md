# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.*
**Last updated: 2026-09-01, iteration 313.**

## Where the mission is

- **Active sprint**: `m-registry-interface-hash-blind-to-signatures` — Sprint 1, **3 of 5 milestones done**
  (M1 iter-311, M2 iter-312, M3 iter-313). M4–M5 ≈ 2 days remain.
- **Sprint 2 (M6–M9) stays DEFERRED** on a precondition the loop cannot satisfy: the blast-radius
  measurement for the publish-time type-check gate needs the live registry, unreachable from the
  loop's session. Shipping M6 without it ships an unbounded regression.
- **The defect being fixed**: `InterfaceHash` folds no signature data, so add/remove/retype of an
  export leaves it byte-identical and the release is notified as `patch` (seen in the wild on
  `sunholo/external_backend` 0.1.0→0.2.0).

## Next picks (banked, in order)

1. **Sprint 1 M4** — `InterfaceHashV2` + `InterfaceHashVersion` (continues the in-flight sprint).
2. `m-registry-validator-unbounded-compile` — a public HTTP server compiles untrusted uploads with
   `exec.Command` (no deadline) at three sites. Confirmed at HEAD, pre-existing, security-shaped.
3. `m-coordinator-config-route-preflight` **(a) only** — `config diff` prints `identical` without
   validating; half (b) is unroutable until `ExecutionRoute` reaches dev.

## Loop health

- **Cadence**: launchd, pinned worktree `~/.ailang-driver-pin/v1`, ~16 fires/day.
- **Routing**: controller opus · designer rotation (fable ↔ deepseek-v4-flash) · planner/executor
  `codex:gpt-5.6-sol` · evaluator `sonnet`. generator≠judge enforced on **provider**.
- **Cost**: iteration 313 metered **$0.00** of the $5 ceiling. Recent iterations $0.00–$0.17.
- **Reaped-slot rate ≈ 40%** (6 of iterations 296–310 produced no record). See **D-52**.
- **Standing divergence**: main checkout is **8 behind origin/dev**, 4 dirty files. Routed around
  every iteration; the reconcile is a human decision.
- **CI**: `SonarCloud` failing on dev — **inherited, not a required context**. Named, never picked.

## Parked on Mark (both OPEN in the decision ledger — `scripts/mission_decisions.sh --open`)

- **D-51** — ratify or replace the charter's countable finish-line unit. The provisional unit
  (open queue rows) is **anti-correlated with good work**. Loop recommends **(b) milestone
  burn-down**. Unanswered ⇒ status quo, every Progress line stays misleading.
- **D-52** — is the ~40% reaped-slot rate worth one iteration to diagnose? Loop recommends **(a)**,
  a per-gate heartbeat artifact so a dead slot names the gate it died in. Unanswered ⇒ **(b)**,
  keep recovering by hand via the Gate-2 died-mid-flight traces.
