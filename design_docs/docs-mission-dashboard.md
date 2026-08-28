# Docs Mission Dashboard (snapshot — history lives in the charter + log)

**Last updated**: 2026-08-28T06:49Z, iteration 0.

## Status
Charter drafted attended with Mark 2026-08-28. **NOT YET RATIFIED.** Quorum blocked THREE times
(rounds in `.ailang/state/mission-quorum/docs-mission-*.json`). Two designer revisions applied (the
second folding in a mid-iteration human decision). No sprint has run — the charter's own gate
blocks all sprint routing until ratification lands. No reviewer has passed in any round yet;
objections have landed on a new surface each round rather than converging.

## Blocking on Mark
**D-DOCS-1 — RESOLVED** (attended, commit `29a467cac`): blast radius now includes `tools/`.
**D-DOCS-2 (open)**: should docs-1's full implementation protocol be specified in the charter now,
or deferred to its own sprint plan? **D-DOCS-3 (open)**: does an "unratified but operationally
live" doc need an explicit banner, or is that inherent to any doc mid-quorum-review? See
`design_docs/docs-mission-log.md` iteration 0 for full objection text and refutations.

## Queue (top = next, but ALL gated behind docs-0 ratifying)
1. `[NEXT]` docs-0 · ratify charter — **PARKED-needs-human-review**, blocked on D-DOCS-2/D-DOCS-3.
2. `[NEXT]` docs-1 · clause 7 · build the inbox-routing trigger — blast radius unblocked, still
   gated behind docs-0 ratifying.
3. `[NEXT]` docs-2 · clauses 1+3 · first `docs-sync` sweep.
4. `[PARKED]` docs-3 · clause 6 · benchmark surface audit.
5. `[PARKED]` docs-4 · clause 5 · taxonomy pass.

## Loop cadence + routing
launchd `dev.ailang.mission-docs`, every 6h, staggered against v1/world/motoko. Routing ladder is
cost-type ordered: subscription (`claude-sonnet-5`/`codex:gpt-5.6-luna`) → flat-rate (Ollama Cloud
`pi:ollama/*:cloud`) → metered OpenRouter twin. Evaluator vendor-disjoint from executor at every
rung. Metered ceiling $1/iteration (fleet default $5).

## Cost this iteration
$0.197 of $1 ceiling (three design-quorum rounds). Quota: sonnet (controller + 2 designer runs).

## Quota posture
No fallback triggered this iteration; no lane exhaustion observed.
