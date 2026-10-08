# WASM type-check budget is still wall-clock by default — the deterministic step ceiling (#662 ask 2) has no design doc

- **Date**: 2026-10-03
- **Class**: bug (hardware-dependent correctness)
- **Recommend**: design-doc
- **Searched**: `#662`, `m-wasm-deterministic-typecheck-budget`, `typeCheckSteps`, `step ceiling`, `wasm-typecheck` across design_docs/ (hits: `implemented/v0_29_0/m-wasm-typecheck-limits.md` — the original 2 s wall-clock guard; `deferred/m-wasm-typecheck-iterative.md` — the deferred refactor; `v1-mission.md:4781` — a queue row only, "BLOCKED ON EXTERNAL DATA"; no planned doc); `ailang-core-backlog.md` (no row)
- **Estimate**: omitted (design-doc)

Issue: [#662](https://github.com/sunholo-data/ailang/issues/662) — a real user's Firefox failed to load the deployed ailang-parse demo because the WASM type-checker's 2 s **wall-clock** budget tripped on a two-line function; under 2x CPU throttling 4 of 25 modules fail, under 4x 10 fail.

## State at origin/dev 790169359

PR #780 (`d5831af9b`) shipped asks 1, 3 and 4. The budget is host-settable (`ailangSetTypeCheckBudget`), and consumption is reported as `typeCheckMs`, `typeCheckSteps` and `budgetMs`. The error names the budget actually in force. The mechanism now lives in build-tag-free `internal/types/typecheck_budget.go`, so it is natively tested. **Ask 2 is not done.** The gate is still `b.now().After(b.deadline)` (typecheck_budget.go `check()`), `DefaultWasmTypeCheckBudget = 2 * time.Second` is unchanged, and `steps` is counted but never compared. Any host that does not call the setter, including the deployed demo unless docparse has since raised it, still has a hardware-dependent load outcome.

## Why it needs a doc, and what the doc must decide

1. **The ceiling value and how it is chosen.** The mission row blocks on the reporter sending per-module `typeCheckSteps`. The reporter is ailang-parse, which is `sunholo-data/docparse`, our own repo (29 `.ail` files locally). The measurement can be taken first-party: arm the budget natively in a test or a hidden flag (`begin()` and `check()` are not build-tagged) and record steps across docparse, `std/`, `examples/` and the citizen.ail pathological fixture. The doc should replace "wait for external data" with that measurement plan.
2. **Step gate versus wall-clock backstop.** Choose one: the step ceiling only; the step ceiling as the default plus the wall-clock limit as an opt-in host backstop; or both active. If both are active, the hardware dependence the issue reports remains.
3. **What a step is.** Today a step is a count of `inferCore` and `Unify` entries. If the gate becomes a compatibility contract, any type-checker refactor that changes the count also changes which programs load. Decide whether the ceiling is versioned with the binary and how much headroom it carries.
4. **Native parity.** Decide whether the native `ailang check` should report the same step count, so CI can warn when a module is at 80% of the WASM ceiling. This is ask 3 carried to the place where developers actually run checks.

Related: `implemented/v0_29_0/m-wasm-typecheck-limits.md` (prior art for the guard), `deferred/m-wasm-typecheck-iterative.md` (the refactor alternative).
