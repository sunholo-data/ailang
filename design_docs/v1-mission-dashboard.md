# Mission Dashboard — V1

> Snapshot only; history in `v1-mission.md` + `v1-mission-log.md`. Written: **iter 254, 2026-08-23**.

## Where we are

- **v0.33.1**. **BAR-FIRST** (`D-28`, 08-22): bar items outrank cons-cells until all 5 clauses close.
- **dev CI** GREEN at `329b81544` — 16 checks, zero not-green.
- **Ledger** 29 rows, **ONE OPEN — `D-29`, the gate on clause 5's headline number.**

## In flight / next

1. **`m-contract-verification-coverage` — DIAGNOSED (iter 254), now BLOCKED ON `D-29`.**
   Iteration 253 routed it at the Z3 encoder. Measured, that is **wrong by 8:1**.
2. **Next un-blocked bar item: `m-effect-clock-net-fs-modes`** (clause 4) — needs no ruling.
3. Remaining: `m-v1-orchestration-flagship` · `m-run-selector-enumeration-floor` · A1/A2 (`D-25`) ·
   clause-3 prompt A/B. **Then** LC-2 execution resumes per `D-28`.

## New this iteration

- **The published `$0.7778` is suppressed by a predicate, not the encoder.** The 53 skips split
  **24 `no ensures clause` · 20 declaration-closure · 5 unencodable type · 4 unencodable builtin**.
- **The 24 are the benchmarks' OWN spec.** `isBST`/`encode`/`decode`/`toRoman`/`minor3` are declared
  `requires`-only, the contract sitting in separate proof functions; the models complied exactly and
  `skipped == 0` disqualifies them for it. Each is skipped in **all 5 models** — spec, not model.
- **Priced** (numerator frozen at `$2.3334561216`): **A as-published 3 → $0.7778** · **B exempt
  "nothing to verify" 11 → $0.2121** · **C exempt all skips 12 → $0.1945**. **8 of 9 recoverable
  runs are the PREDICATE; exactly 1 is the ENCODER.** Arm A reproduces the shipped KPI to the last
  digit, so all three are one instrument with one clause changed.
- **Routed** as an AILANG fix: split `skipped` into `skipped` vs `not_applicable` — correct either
  way, since the encoder figure a reader needs (**29**) hides inside a 53 that is 45% noise.

## Routing · Cost · Parked

Controller `claude:claude-opus-5`; **no sub-agent spawned** — the deliverable is a measurement over
frozen banked data, judged by exact reproduction of the published KPI. Designer rotation still
degraded (gemini read-only; `codex` collides with reviewer `gpt5-6-sol`). Metered **$0.00**.

**PARKED ON MARK — `D-29`:** should a function with **no `ensures` clause** count against
`isVerifiedSuccess`? **(a) exempt** → headline `$0.7778 → $0.2121`; **(b) keep strict** → add
`ensures` across the benchmark specs and re-run the cohort; **(c) publish both** strict/effective.
