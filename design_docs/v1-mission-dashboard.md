# Mission Dashboard — V1

**Snapshot** — overwritten every iteration; history lives in the charter STATUS block and the log.

## Iteration 295 — 2026-08-28

- **Release**: v0.34.0. `dev` CI **green** on all required contexts at `28014b1df`.
- **Standing non-required red**: `SonarCloud Code Analysis`, **inherited** since the `caea1f9e1`
  merge (green at `7dff0942d`). Gate conditions: `new_security_rating` 2 (needs 1),
  `new_coverage` 64.2% (needs 80). Named, not the pick.

## In flight / next

1. **[NEXT] m-openrouter-session-chain-registration** — **unblocked, scope settled, ~15 LOC.** `M-MISSION-LOOP-UNIFIED-TELEMETRY`
   M1's *read* side is live and tested; its *write* side was never built, so every OpenRouter Broadcast
   trace is still unjoinable. Measured: `sessions_keyed_by_a_chain_id` = **0** (controls 19262 / 18947);
   prod 72 h — 97 spans carry `session.id`, **0** resolve to a chain.
2. **m-prompt-freeze-mirror-all-versions** — design written (iter-293 planner), re-scope against `4d8705699`.
3. **m-git-binary-resolution-sweep** — has a doc, needs a quorum run before planning.

## Parked on Mark — 4 open decisions

`scripts/mission_decisions.sh --open` is authoritative (46 rows, valid). D-45 was filed this
iteration and **withdrawn by measurement** — the evaluator showed it was not a real decision.

- **D-46** — who reconciles the `M-MISSION-LOOP-UNIFIED-TELEMETRY` sprint JSON: `mine` or `loop`?
- **D-42** — standing authorisation to reconcile the main checkout against origin (17 behind today).
- **D-43** — should `std/string.charAt` itself become total (breaking)?
- **D-44** — `ai_check.go:289` verify blindness; fixing it moves a KPI with a banked baseline.

## Loop cadence + routing
- Driver runs **pinned** (`~/.ailang-driver-pin/v1`); running skill verified == origin each fire.
- controller `opus` · designer rotation `claude:claude-fable-5` → `pi:ollama/kimi-k3:cloud`
  (pointer unchanged; Fable diet unspent in 293/294/295) · planner lane derived per-pick ·
  executor `codex:gpt-5.6-sol` · evaluator `sonnet`, own worktree.
- **metered spend: $0.00** of the $5/iteration ceiling, three iterations running.

## Quota posture
Anthropic available (`MISSION_ANTHROPIC_AVAILABLE=1`); billing tripwire CLEAN every fire.
