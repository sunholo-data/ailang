# M-MISSION-COST-CHAINS: make `ailang chains` the mission's cost tracker — fix $0 attribution + ingest loop activity

**Status**: Planned (Mark 2026-07-18 — "lets keep an eye on these budgets, perhaps a cost
tracker. I think actually that should all appear in ailang chains CLI... dont know if all this
activity is still flowing in there" — verified: it is NOT)
**Target**: v0.30.x — mission infrastructure / clause-5 substrate
**Priority**: P1½ — this is the DATA SUBSTRATE for clause 5's `m-cost-per-success-kpi` (the v1.0
headline "cost-per-verified-success" KPI cannot be credible while the canonical cost CLI shows
$0.0000 and the mission's own spend is invisible)
**Estimated**: ~1.5–2d (M1 cost-attribution fix ~0.5d; M2 mission ingest ~1d; M3 budget rollup ~0.5d)
**Dependencies**: `internal/server/handlers_chains.go` (`handleCreateChain` → obsBackend — the
existing write path); the metered-spend ledger + `metered=$` evidence-row field (landed
2026-07-18, `27cead433`); executor cost reporting (`Result.CostUSD` — codex/managed_agents both
populate it); quorum artifacts (per-reviewer cost in the JSON)
**Author**: interactive session (Fable) with Mark, 2026-07-18

---

## Problem statement (both halves verified live 2026-07-18)

1. **Defect A — cost attribution is unwired even where chains flow.** `ailang chains stats
   --by-agent --hours 48`: 45 chains, **50,301,564 tokens** tracked… **Total Cost: $0.0000**.
   Token counts arrive (e.g. `eval-agent:api_call_json` 1.5M in / 7.4K out) but no cost field is
   ever populated — either senders omit it or the rollup never computes from tokens × model rate.
   A cost tracker that always says $0 is worse than none (false "free" signal).
2. **Gap B — the mission loop is invisible.** Chains are created only via the server HTTP API
   (`handleCreateChain`), which the eval harness/coordinator call. The mission's actual spenders
   never do: headless `claude -p` iterations (quota), Agent-tool sub-agents (quota), `codex exec`
   (metered $), `managed_agents` runs (metered $ — the $0.865 E2E and the $0.11 reuse experiment
   appear NOWHERE), `design-quorum` reviewer calls (metered cents). 48h of the heaviest fleet
   activity ever → zero chains from the mission.

## Design

### M1 — fix cost attribution (Defect A)
Find where eval-harness chains get token counts but not cost; populate `CostUSD` at the SOURCE
(executors already compute `Result.CostUSD`; the eval sender must forward it) and add a rollup
fallback: when a stage has tokens + a known model, compute cost from the model registry rate —
flagged `cost_estimated=true`, never silently. Acceptance: `chains stats --hours 24` shows
non-zero, reconcilable cost for eval chains.

### M2 — mission-loop ingest (Gap B)
One chain per mission iteration; stages = the loop's units of spend. NO new infrastructure — POST
to the existing `handleCreateChain`/stage endpoints (server already runs on the rig; if the
server is down, buffer to a JSONL spool the next iteration flushes — fail-soft, never block the
loop on telemetry):
- The SKILL (Gate 4) posts the iteration chain: stages `(role, provider, model, tokens?, $)` from
  what actually ran — the same data already written to the evidence row (single source: write the
  row, then post it). Metered $ from executor results/quorum artifacts; quota-lane stages carry
  `cost_usd=0, quota_bucket=<fable|opus|sonnet>` so subscription burn is VISIBLE per-bucket even
  though it isn't dollars.
- `source` naming: `mission:v1/iter-<N>` (portability-ready: `mission:<name>/iter-<N>` — the
  Ailang World loop lands in the SAME tracker for free).
Acceptance: `ailang chains list` shows the iteration; `chains view` shows role stages with $ and
quota-bucket attribution; the E2E-class runs (gemini/codex) appear with their real cost.

### M3 — budget rollup surface
`ailang chains stats --by-mission` (or `--by-source-prefix`): per-mission daily/weekly metered
total vs `MISSION_METERED_BUDGET_USD`, per-bucket quota-stage counts, top-N most expensive
stages. This is Mark's "keep an eye on budgets" view — one command, no dashboards required
(dashboard consumption can come later via the existing benchmark-fetch route).

## Conflict surface
Touches the eval-harness chain sender (M1), mission-control SKILL Gate 4 (M2 — additive posting
step), `chains` CLI (M3 — additive flag). Must NOT: block or fail an iteration on telemetry
(fail-soft spool); double-count (the evidence row is the source of truth, the chain is its
projection); count subscription quota as dollars (buckets are tracked as buckets); invent a
second storage (observatory.db via the existing server API only).

## Non-goals
- Real-time mid-run cost streaming (managed_agents API can't; post-hoc per stage is enough).
- Provider-console spend caps (Mark-side backstop, separate).
- Replacing the evidence rows (they stay canonical; chains = queryable projection).
- Dashboard UI work (M3 is CLI; UI later rides the same data).

## Verification log
| Claim | Method | Result |
|---|---|---|
| Chains flow for eval but cost=$0 with tokens present | `chains stats --by-agent --hours 48`: 45 chains, 50.3M tokens, $0.0000 | Confirmed 2026-07-18 |
| Mission activity absent | `chains list --since 48h`: only `eval_suite:*` sources across the fleet-heaviest window | Confirmed |
| Write path = server HTTP API | `handlers_chains.go:286 handleCreateChain` → obsBackend | Confirmed |
| Executors report per-run cost | `Result.CostUSD` populated by codex + managed_agents (reuse test: $0.068/$0.039) | Confirmed |
| Evidence rows carry `metered=$` | skill Gate-3 ledger, commit `27cead433` | Confirmed |

## Related
- `m-cost-per-success-kpi` (clause 5 — this is its data substrate; sequence THIS first)
- [m-mission-portability](m-mission-portability.md) — `mission:<name>/…` source naming keeps World in the same tracker
- Memory: `reference-headless-claude-billing-rig` (billing lanes), routing memory (evidence rows)

---
**Document created**: 2026-07-18 (interactive; expect quorum-at-pick)
