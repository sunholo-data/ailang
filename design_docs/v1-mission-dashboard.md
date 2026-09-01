# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the log.*
**Last updated: 2026-09-01, iteration 316.**

## Where we are against the bar
- **Goal (D-51, ratified attended 2026-08-28):** *design docs remaining before v1.0.0.*
- **N = 10** (7 existing docs + 3 NEW-DOC units), plus a named **UNCLASSIFIED bucket of 4**
  that would make it 14. Ruling on those 4 is **D-53**, the only open decision.
- Iteration 316 moved no doc off the list: it finished a milestone inside an existing doc.

## In flight
- `m-registry-interface-hash-blind-to-signatures` — **Sprint 1 COMPLETE (M1–M5 all landed)**.
  M5 landed this iteration (PR #1007): signature-set classification with the `U` class.
  **Sprint 2 (M6–M9) is DEFERRED** behind a precondition the loop cannot satisfy — the
  publish-gate blast-radius measurement needs a live registry this session cannot reach.

## Up next (banked, top of queue)
1. `m-docparse-v0340-reports-2026-09-01` — **VERIFY-then-route.** Three defects from a live
   downstream consumer at v0.34.0. The one that matters: a stale per-directory `.ailang` iface
   cache silently drops exports at compile *and* at `serve-api` runtime — a dead Cloud Run
   endpoint with no startup error. **Does not reproduce in the two obvious shapes at HEAD**;
   finish the repro before routing a sprint.
2. `m-changeclass-unknown-consumers` — small, and a **precondition for Sprint 2**: `U` is a
   fourth change-class value in a codebase whose switches were written for three.
3. Four pre-existing driver defects (R7–R10) from iteration 315 — ~0.5–1d, all independent.

## Loop health
- Cadence: launchd, ~16 fires/day. Iterations 313–316 all landed.
- Routing: controller opus · designer rotation (not spawned; doc pre-exists) · planner opus
  (not spawned; plan pre-exists) · executor `codex:gpt-5.6-sol` · evaluator `sonnet`.
- **Judge is earning its place**: 316 took three rounds (FAIL 65 → FAIL 66 → PASS 91), and both
  blocking findings were real routing defects that would have shipped green.
- Metered spend, iteration 316: **$0.00** of the $5 ceiling. Every lane a quota bucket.
- Standing: main checkout **3 ahead / 18 behind** origin/dev (one-way drift, route-around
  applied every iteration); SonarCloud red on dev is **inherited**, not ours.

## Parked on Mark
- **D-53** — rule on the 4 UNCLASSIFIED docs (N=10 vs N=14). Loop recommends N=12.
  Default if unanswered: keep reporting N=10 with the bucket named. Nothing stalls.
