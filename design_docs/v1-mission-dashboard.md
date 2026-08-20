# Mission Dashboard — V1

**Snapshot** (overwritten every iteration; history lives in the charter STATUS block + mission log).
Last updated: **2026-08-20, iteration 236**.

## Latest
- **Release**: v0.33.1 · `dev` green (21 checks SHA-addressed, zero not-green).
- **Iteration 236 LANDED**: [#802](https://github.com/sunholo-data/ailang/pull/802) → `8f45dd419`.
  LC-1 `m-list-repr-spike` design doc (495 lines, 20 verification rows) + 6-milestone sprint plan.

## In flight / next
- **LC-1 `m-list-repr-spike` — plan is READY, spike NOT yet executed.** Next iteration runs it.
  It carries the cons-cells **kill criterion**; LC-2…LC-5 (~16 d) are gated on its go/no-go.
- Then in order: LC-2 `m-list-accessor-api` → LC-3a/3b/3c (parallelizable) → LC-4 → LC-5 → LC-0.
- Also queued: `m-stdlib-reverse-delegates-to-builtin` (cheap, and REQUIRED by the programme).
- Blocked, re-measured this iteration: `m-wasm-deterministic-typecheck-budget` — `#662` still has
  **1** comment (ours, 2026-08-18); reporter has not supplied per-module `typeCheckSteps`.

## Loop cadence + routing
- Controller `claude:claude-opus-5` · designer **rotation** (Fable → codex) · planner `opus`
  (`derive-planner-lane` → `opus fail-closed:planner-lane-field-invalid`) · executor `codex:gpt-5.6-sol`
  · evaluator `sonnet`.
- Iteration 236 lanes: designer **Fable** (1 run — diet respected), revision **codex**, planner **opus**.
  Designer-rotation pointer (namespaced) now `codex:gpt-5.6-sol`.

## Parked on Mark
- **Decision ledger: 21 rows, ZERO OPEN.** Nothing is waiting on you.
- Carried, not a ledger row: rotate `AILANG_REGISTRY_API_KEY` (from iteration 232).

## Quota / cost posture
- Iteration 236 metered: **$0.1605** of the $5 ceiling (two quorum rounds; $0.0757 + $0.0849).
- opus / codex / sonnet / Fable are all subscription quota buckets — $0 metered.
- Billing tripwire CLEAN. No GPU, no `rig.lock` taken.
