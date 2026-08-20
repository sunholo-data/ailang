# Mission Dashboard — V1

_Snapshot, overwritten each iteration. History lives in the charter STATUS block + `v1-mission-log.md`._

**Last iteration:** 238 · 2026-08-20 · **LANDED** · evaluator (sonnet) PASS **84/100**
**Latest release:** v0.33.1 · `dev` at `8e27b0a12`

## In flight / next
- **LC-1 `m-list-repr-spike`** — M1–M5 LANDED (`b32eef76b`). **M6 OWED**: the full 76-point ×
  5-trial matrix under M5's runner, the kill-criterion arithmetic, and the programme's
  **go/no-go**. Nothing else in the cons-cells programme may route before it.
- Provisional, NOT the verdict (M5 protocol, darwin/arm64 only): clause **(c)** C1 **1.946×** vs
  ≤2.5, C2K32 ≈**1.07×** · clause **(d)** all four arms **≈1.00×** vs ≤1.2 · clause **(a)**
  (iter-237) C1 **0.95×/1.08×** vs ≤1.5. Clause **(b)** not yet measured.
- Then: LC-2…LC-5 **iff** M6 says GO; otherwise STOP and re-open `D-19` with measurements.

## Blocked
- `m-wasm-deterministic-typecheck-budget` — waiting on `#662`'s reporter for per-module
  `typeCheckSteps`. Predicate re-run 2026-08-20: 1 comment, ours. **External; re-check each pick.**

## Loop / routing
- Controller opus · designer ROTATION (next: gemini) · planner+executor `codex:gpt-5.6-sol`
  (bucket reset 05:34 today) · evaluator **sonnet** (generator≠judge vs codex).
- **Skill edit `8e27b0a12`:** the Agent tool now **ACCEPTS** a `fable` pin — the stale
  "REJECTED" rule had been silently skipping the rotation's Fable designer slot.

## Parked on Mark
**None.** Decision ledger: **21 rows, ZERO OPEN.**
Carried, not a ledger row: rotate `AILANG_REGISTRY_API_KEY` (iter-232).

## Quota / cost
- Iteration 238 metered **$0.00** of $5. Quota buckets: opus, codex, sonnet, one bounded fable probe.
- No GPU, no `rig.lock` — the spike is pure CPU/heap.
