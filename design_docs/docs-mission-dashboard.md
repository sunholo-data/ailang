# Docs Mission Dashboard (snapshot — history lives in the charter + log)

**Last updated**: 2026-08-28T08:49Z, iteration 1.

## Status
Charter ratified attended 2026-08-28 (Mark). First real sprint (`docs-2`, first-ever `docs-sync`
sweep) LANDED this iteration: [PR #955](https://github.com/sunholo-data/ailang/pull/955) →
`a8f904aac`. Sprint-evaluator (sonnet) independently re-derived every load-bearing claim from
scratch — PASS 92/100, zero blocking.

## Blocking on Mark
Two NEW open decisions, both about this mission's own blast-radius allowlist blocking it from
fixing what it just found:
- **D-1**: widen `.claude/skills/docs-sync/*` into the allowlist so the mission can fix
  `check_examples.sh`'s absolute-path `MOD010` bug (found this iteration — it has been silently
  over-reporting broken examples: true split is 166 pass / 9 fail / 42 no-module vs the script's
  own unreliable 12/29/176).
- **D-2**: widen `docs/*` (currently single-level) to cover `docs/docs/**`/`docs/src/**`, where
  nearly all published content and a stale `v0.16.0` version reference actually live.
Full options and evidence: `design_docs/docs-mission.md`'s Human Decision Ledger (D-1, D-2) and
`design_docs/docs-mission-log.md` ITERATION 1.

## Queue (top = next)
1. `[LANDED]` docs-2 · clauses 1+3 · first `docs-sync` sweep.
2. `[NEXT]` docs-5 · clause 2 · fix the 9 genuinely-failing runnable examples (in scope, no
   allowlist change needed).
3. `[PARKED — needs Mark]` docs-6 · clause 1 · fix `check_examples.sh`'s absolute-path bug (D-1).
4. `[PARKED — needs Mark]` docs-7 · clause 1 · mission cannot edit its own published content (D-2).
5. `[PARKED]` docs-8 · clause 1 · 126 overdue planned design docs (aggregate triage).
6. `[NEXT]` docs-1 · clause 7 · build the inbox-routing trigger.
7. `[PARKED]` docs-3 · clause 6 · benchmark surface audit.
8. `[PARKED]` docs-4 · clause 5 · taxonomy pass (also blocked on D-2, nested paths).

## Loop cadence + routing
launchd `dev.ailang.mission-docs`, every 6h, staggered against v1/world/motoko. Routing ladder is
cost-type ordered: subscription (`claude-sonnet-5`/`codex:gpt-5.6-luna`) → flat-rate (Ollama Cloud
`pi:ollama/*:cloud`) → metered OpenRouter twin. Evaluator vendor-disjoint from executor at every
rung. Metered ceiling $1/iteration (fleet default $5).

## Cost this iteration
$0.00 of $1 ceiling — both codex runs (planner + executor) rode the subscription-lane rung
(`gpt-5.6-luna`), no flat-rate/metered fallback. Quota: sonnet (controller + evaluator).

## Quota posture
No fallback triggered this iteration; no lane exhaustion observed. Two pre-existing CI reds on
`sunholo-data/ailang` observed and handed to V1 (repo owner): a flaky `launchd drivers` timing
test (confirmed transient) and an inherited SonarCloud dev-branch quality-gate red (confirmed
pre-existing, unrelated to this mission's diff).
