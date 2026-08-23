# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) + `v1-mission-log.md`.*

**Iteration 259 · 2026-08-23 · dev green (16 checks, 0 not-green @ `bc3f80884`)**

## In flight
- **`m-verify-bounded-unrolling-false-counterexample`** — design doc LANDED, quorum ×2 + a
  restored absent reviewer, carve-out applied. **ROUTABLE TO SPRINT-PLANNER next iteration.**
  The verifier reports its own incompleteness as a `counterexample`, so correct recursive code
  is graded a verification FAILURE. Measured: **8 of 30 frozen-cohort runs**, all on
  `contract_sorted_merge` + `contract_bst_validate`, uniform across 4 model families.
  Fix needs **zero `internal/smt` changes** — the predicate is already computed.

## Next
1. Route the above to sprint-planner (`codex:gpt-5.6-sol`).
2. `m-benchmark-ensures-coverage` — Mark's `D-29` second clause; 4 of its 5 candidates stay
   blocked until the above lands (`minor3` is the one safe today).
3. `m-cohort-manifest-build-provenance` — awaiting the iter-257 decomposition split.

## Parked on Mark (3 open — see `DECISIONS FOR MARK` on #745)
- **`D-30`** — harness↔`ai-check` PATH version coupling. **Now blocking TWO docs
  independently**, and the newer one's skew direction is KPI *inflation*.
- **`D-31`** — designer rotation has ONE usable authoring lane (instance 6 this iteration).
- **`D-32`** *(new)* — should `inconclusive` be exempted from the effective KPI arm, as
  `not_applicable` is under your `D-29` ruling? Only thing that could move `$0.7778`.

## Loop
- Cadence: launchd, ~90 min. Controller opus · designer **rotation collapsed to fable** (D-31)
  · planner/executor `codex:gpt-5.6-sol` · evaluator sonnet.
- Metered **$0.2251** of $5 this iteration (quorum only). Quota: opus, fable ×2 (diet-compliant).
- Baseline KPI `cost_per_verified_success` = **$0.7778187072** (strict) / **$0.2121** (effective,
  per `D-29`). Unchanged this iteration, and unchangeable by the in-flight fix without `D-32`.
