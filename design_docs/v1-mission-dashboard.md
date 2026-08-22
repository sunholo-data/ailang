# Mission Dashboard — V1

> Snapshot only; history lives in `v1-mission.md` + `v1-mission-log.md`. Written: **iter 253, 2026-08-22**.

## Where we are

- **v0.33.1**. **BAR-FIRST ordering in force** (`D-28`, attended 08-22): bar-gating items outrank
  the cons-cells programme until all five v1.0 clauses close.
- **dev CI** GREEN at `498a64d38` — 16 checks, zero not-green (control: parent, 16).
- **Ledger** 28 rows, **ZERO OPEN**. Main checkout 0 ahead / 0 behind — records write in place.

## In flight / next

1. **CLAUSE 5 CLOSED (iter 253).** M4b fired on `D-26`:
   **`cost_per_verified_success = $0.7778187072`**, baseline `v1.0` (agent · ailang · 6 contract
   benchmarks × 5-model `agent_suite`, 30/30 banked, cohort hash `526fe724…`). Strict cmd rc=0,
   `available:true`; independently recomputed at `Decimal` precision (delta 3.6E-17). **$0.46 of $20.**
2. **`m-contract-verification-coverage`** (NEW, P1) — the cheapest lever on that number.
3. Remaining bar clauses: `m-effect-clock-net-fs-modes` · `m-v1-orchestration-flagship` ·
   `m-run-selector-enumeration-floor` · A1/A2 parity lane (`D-25`) · clause-3 prompt A/B.
   **Then** LC-2 execution (M1+M2) resumes per `D-28`.

## New this iteration

- **The KPI has a number for the first time.** All 19,027 previously banked files read
  `verify_verified = 0`; 14 of 30 cohort runs now carry a positive count.
- **The denominator is suppressed by Z3 SKIPPING, not wrong programs**: 14 partial · 8
  counterexample · 4 verifier error · 3 fully verified · 1 no-compile; **53 skipped vs 28 verified**.
  `isVerifiedSuccess` needs `skipped == 0`, so one skip disqualifies a run.
- **Two reproducibility defects, both from running a check not trusting a name**: the manifest
  recorded `git_commit:"dev"` (no ldflags under the mandated scratch build), and **`eval_results/`
  is git-ignored**, so M4b's own "archive its full output" AC would have archived nothing.

## Routing · Cost · Parked

Controller `claude:claude-opus-5`; **no sub-agent spawned** — the deliverable was a measurement and
its judge is the independent recomputation. Designer rotation still degraded (gemini read-only;
`codex` collides with reviewer `gpt5-6-sol`; Fable the only clean authoring lane). No GPU, no
`rig.lock`; every wait bounded. Metered **$0.4586** (all OpenRouter) + **$1.8749**
*list-price-equivalent* subscription reporting, labelled per the ratified 07-27 decision, not billed.
**Parked on Mark: nothing.**
