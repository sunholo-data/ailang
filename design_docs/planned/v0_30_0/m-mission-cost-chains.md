# M-MISSION-COST-CHAINS: make `ailang chains` the mission's cost tracker — fix $0 attribution + ingest loop activity

**Status**: Planned (Mark 2026-07-18 — "lets keep an eye on these budgets, perhaps a cost
tracker. I think actually that should all appear in ailang chains CLI... dont know if all this
activity is still flowing in there" — verified: it is NOT)
**Baseline re-pinned 2026-07-24 (iteration 97 Gate-2 reality-check, binary `v0.30.0-147-g6ed26bebd`)**:
the original Defect-A HEADLINE ("Total Cost: $0.0000 everywhere") is now STALE — recent eval chains
DO attribute cost. Live: `chains stats --hours 48` = **$9.5879** over 30 chains; `--hours 336` =
**$291.19** over 213 chains. The eval sender began forwarding `CostUSD` via `43333e7a8` (2026-07-19,
a reasoning-token fix). The still-real M1 deliverable therefore NARROWS to the **rate-fallback
rollup** (token-bearing stages that lack a self-reported cost STILL show a misleading `$0.0000` — e.g.
`eval-agent:balanced_parens` 8.5M tokens → $0 over 14d — because the rollup never estimates from
tokens×rate). M2 (mission ingest) + M3 (`--by-mission`) verified fully UNBUILT (0 mission chains/14d;
`--by-mission` flag absent). All premise numbers below are pinned to this probe unless dated otherwise.
**Rev-1 2026-07-24 (iteration 97)**: resolved round-1 quorum objections (cost provenance,
pricing-registry verification, bounded+loud spool).
**Status → PARKED needs-human-review (iteration 97, 2026-07-24)**: re-quorum round-2 surfaced a
convergent soundness objection (M1's CLI-side cost inference would corrupt legitimately-free/quota
$0 stages — incl. M2's own quota-lane stages). Gate spent; genuine schema-design gap, not a verbatim
refinement → parked. See the ⛔ Human fork for Mark in the Quorum record. M2/M3 direction unobjected.
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

1. **Defect A — cost attribution has NO rollup fallback (partially fixed at source 2026-07-19).**
   *Original 2026-07-18 finding:* `chains stats --by-agent --hours 48` showed 45 chains,
   **50,301,564 tokens**, **Total Cost: $0.0000** — the eval sender omitted `CostUSD`.
   *Reality-check 2026-07-24:* the eval sender now forwards `CostUSD` (`43333e7a8`), so recent
   chains attribute cost ($9.59/48h). **The residual defect:** the rollup still has NO
   tokens×model-rate fallback, so ANY stage whose sender didn't self-report cost shows a misleading
   `$0.0000` despite real tokens (proven: `eval-agent:balanced_parens` 8.5M tokens → $0 over 14d;
   all pre-2026-07-19 banked chains likewise). A cost tracker that silently shows $0 for a
   token-bearing stage is still a false "free" signal — the fallback is the real fix.
2. **Gap B — the mission loop is invisible.** Chains are created only via the server HTTP API
   (`handleCreateChain`), which the eval harness/coordinator call. The mission's actual spenders
   never do: headless `claude -p` iterations (quota), Agent-tool sub-agents (quota), `codex exec`
   (metered $), `managed_agents` runs (metered $ — the $0.865 E2E and the $0.11 reuse experiment
   appear NOWHERE), `design-quorum` reviewer calls (metered cents). 48h of the heaviest fleet
   activity ever → zero chains from the mission.

## Design

### M1 — cost-attribution rollup fallback (Defect A residual)
The at-SOURCE half landed 2026-07-19 (`43333e7a8` — the eval sender forwards `Result.CostUSD`), so
NEW eval chains attribute cost. The remaining deliverable is the **rollup fallback**: when a stage
has tokens + a resolvable model but NO reported cost, compute cost from the model registry rate,
flagged `cost_estimated=true` (never silently, never overwriting a self-reported cost). This closes
the misleading `$0.0000`-with-tokens rows (pre-fix chains + any non-self-reporting sender).

**Cost provenance (Rev-1):** every stage in the rollup carries `cost_status ∈ {reported, estimated,
unknown}`. The stage schema (`ChainStage`, `internal/observatory/models_chains.go:82`, `Cost float64`
at ~:110) has NO provenance field today — $0 and "no data" are indistinguishable in storage — so M1
uses **CLI-side inference in the rollup** (no schema migration): `cost>0 → reported`;
`tokens>0 && cost==0 && model resolvable → estimated` (computed tokens×rate); `tokens>0 && cost==0
&& model unresolvable → unknown`. A persisted `cost_status` stage field can follow later (migration
would default existing rows to `reported` when cost>0 else `unknown` — back-compatible), but is NOT
required for M1. Prior art: `computeCost` (`internal/ai/configdriven/provider.go:279`) already
returns `""` (empty string, NOT 0.0) when no cost data is configured, deliberately distinguishing
no-cost-data from free — the rollup adopts the same distinction.
Acceptance: (a) `chains stats` shows a non-zero, `cost_estimated`-flagged cost for a token-bearing
stage that reported no cost; (b) a stage WITH a self-reported cost is unchanged (no double-count);
(c) a stage with tokens but no resolvable model is marked `cost_status=unknown` and is never
presented as $0.0000. Rollups report known reported cost, known estimated cost, and unknown-cost
stage/token counts separately; budget comparisons emit a visible incomplete-data warning or fail in
strict mode. No model or rate is guessed. A regression test asserts the rollup estimates rather
than returns $0 for the token-bearing/no-cost case, and asserts the unknown-model case surfaces as
`unknown`, not $0.

### M2 — mission-loop ingest (Gap B)
One chain per mission iteration; stages = the loop's units of spend. NO new infrastructure — POST
to the existing `handleCreateChain`/stage endpoints (server already runs on the rig; if the
server is down, buffer to a **strictly bounded** JSONL spool the next iteration flushes — hard cap
(max 100 entries / size cap), and every buffering event emits an explicit stderr warning so the
fallback is bounded and LOUD, not silent. On overflow, drop-oldest with a one-line stderr notice —
never grow without bound. Fail-soft: never block the loop on telemetry):
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
(fail-soft spool — bounded + loud: stderr warning on buffer, hard cap, drop-oldest on overflow);
double-count (the evidence row is the source of truth, the chain is its
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
| **Reality-check 2026-07-24 (iter-97, `v0.30.0-147-g6ed26bebd`):** recent eval chains NOW attribute cost | `chains stats --hours 48` = **$9.5879**/30 chains; `--hours 336` = **$291.19**/213 chains; the doc's cited `eval-agent:api_call_json` now shows $1.5457 | Defect-A HEADLINE STALE — at-source fix landed `43333e7a8` (2026-07-19) |
| **Residual M1:** rollup has no tokens×rate fallback | `chains stats --by-agent --hours 336`: `eval-agent:balanced_parens` 8.5M tokens → **$0.0000** (pre-fix + non-self-reporting senders show misleading $0) | Confirmed — rate-fallback UNBUILT |
| **M2:** mission loop still invisible | `chains stats --by-agent --hours 336` grep mission/quorum/codex/design-doc/iter-: **0 rows** | Confirmed — Gap B fully real |
| **M3:** `--by-mission` unbuilt | `chains stats --by-mission` → `flag provided but not defined: -by-mission` | Confirmed — flag absent |
| **Rev-1 (iter-97, `v0.30.0-147-g6ed26bebd`):** stage schema has NO cost-provenance field | `ChainStage` struct, `internal/observatory/models_chains.go:82` — `Cost float64` at ~:110; no cost-status field, so stored $0 and "no data" are indistinguishable | Confirmed — `cost_status` needs CLI-side inference (chosen for M1) or a schema migration (deferred; would default cost>0→`reported` else `unknown`) |
| **Rev-1:** codebase precedent for distinguishing no-cost-data from $0 | `computeCost`, `internal/ai/configdriven/provider.go:279` — returns `""` (empty string, NOT 0.0) when no cost data configured, deliberately "to distinguish no-cost-data from free" | Confirmed — provenance approach is consistent with existing code |
| **Rev-1:** pricing registry EXISTS and carries per-model rates | `pkg.AIProviderCost`, `internal/pkg/ai_provider.go:78` — `InputPer1MUSD`, `OutputPer1MUSD`, `PerCallUSD`, sourced from models.yml / provider specs; `computeCost` (`internal/ai/configdriven/provider.go:279`) already turns tokens+rate into a USD string | Confirmed — M1 wiring task = resolve a per-model rate from this registry in the observatory rollup (`internal/observatory/`), i.e. expose/load provider cost specs into the observatory package. NO rate is guessed — unresolvable model ⇒ `cost_status=unknown` per M1, never a fabricated rate |

## Quorum record
**Round 1 (iteration 96/97, both reviewers reject; design direction ACCEPTED by both):**
- **gpt5-6-sol** — M1 acceptance (c) "left at $0" re-preserved the false-free signal (unknown cost
  reading as $0 silently understates budget totals). → Fix: explicit `cost_status`
  {reported, estimated, unknown} provenance; unknown never presented as $0.0000; rollups split
  reported/estimated/unknown; strict-mode fail or loud warning on incomplete data (applied above,
  with schema/CLI verification rows).
- **gemini** — (1) the "model registry rate" premise was never verified in the log. → Fix:
  verification row proving `pkg.AIProviderCost` (`internal/pkg/ai_provider.go:78`) + `computeCost`
  exist and carry per-model rates. (2) The M2 JSONL spool was an unbounded silent fallback. → Fix:
  hard cap (max 100 entries / size cap), stderr warning on buffer, drop-oldest with notice on
  overflow (applied to M2 Design + Conflict surface).

**Round 2 (iteration 97, re-quorum — STILL BLOCKED; the round-1 fixes were ACCEPTED, but the
CLI-side-inference choice introduced a NEW convergent soundness objection both reviewers raised
independently):**
- **CONVERGENT (gpt5-6-sol + gemini):** M1's CLI-side inference rule
  (`tokens>0 && cost==0 && model resolvable → estimated`) **silently overwrites legitimately
  self-reported $0.00 costs** — 100%-cache-hit calls, free local models, **and M2's OWN quota-lane
  stages, which explicitly emit `cost_usd=0`**. Because the `float64` schema cannot distinguish an
  absent cost from a reported $0, CLI-side inference deterministically MIS-ATTRIBUTES free/quota
  activity as estimated metered spend (and risks double-counting). Rev-1's "avoid a migration" choice
  is unsound: correct provenance REQUIRES persistence, not inference.
- **Converged fix (both reviewers):** make provenance persistence part of M1 —
  add a nullable `Cost *float64` (null = no-data) **OR** an explicit persisted `cost_status` /
  `cost_source` field to the stage **write API + storage schema** now (drop the CLI-inference plan).
  New senders submit `reported`/`estimated`/`unknown` explicitly; **quota stages use `reported`
  with `cost_usd=0` + `quota_bucket`** (so free is not re-estimated). Migrate existing rows:
  `reported` when cost>0, `unknown` when 0; estimate ONLY legacy `unknown` rows (token-direction +
  exact model-rate match). Extend the conflict surface + verification log to the stage API, the DB
  migration, existing readers/writers, and backward compat. gemini adds: **verify `quota_bucket` can
  be stored in `ChainStage` (it is referenced by M2 but never confirmed to exist) or add it.**

**Verdict: PARKED needs-human-review (iteration 97).** The one-revision-one-requorum gate is spent
(Rev-1 → re-quorum). The residual objection is a **genuine schema-design gap** — it changes M1's
mechanism (persisted provenance + a DB migration + write-API change, a materially larger conflict
surface than the doc's ~0.5d estimate) and leaves an unresolved fork the reviewers offered as
alternatives (pointer-`Cost` vs a `cost_status` field). That is a controller-judgment / architecture
decision, NOT a reviewer-verbatim refinement — so it is NOT a clean case for the (unratified)
narrow-refinement carve-out (per the iter-96 precedent: genuine design gaps park; only
verbatim-applicable refinements fold). **The quorum did its job: it caught a real data-correctness
bug (free-$0/quota corruption) BEFORE any code was written — the collision is precisely between M1's
inference and M2's own quota-$0 stages.**

### ⛔ Human fork for Mark (unblocks routing)
1. **Pick the M1 provenance-persistence approach** (both reviewers require persistence, not inference):
   (a) `Cost *float64` pointer (null = no cost data); OR (b) an explicit `cost_status` /
   `cost_source` stage field. Either way M1 becomes a **schema-migration sprint** (stage write API +
   `ChainStage` schema + migration + readers/writers + back-compat), not the original CLI-only ~0.5d.
2. **Confirm/authorize adding `quota_bucket` to `ChainStage`** (M2 references it; not verified present).
3. Optionally **ratify the narrow-refinement carve-out** — but note it would NOT auto-fold THIS item
   (genuine design gap, not verbatim); it would help the OTHER parked narrow-refinement items
   (`m-budget-scoping-bug`, `m-effect-replay-contracts` premise rows).
Once (1)+(2) are decided, a future iteration folds the converged fix in one revision → sprint-planner
(opus) → executor (opus, worktree) → evaluator (sonnet). M2/M3 direction is unchanged and unobjected.

## Related
- `m-cost-per-success-kpi` (clause 5 — this is its data substrate; sequence THIS first)
- [m-mission-portability](m-mission-portability.md) — `mission:<name>/…` source naming keeps World in the same tracker
- Memory: `reference-headless-claude-billing-rig` (billing lanes), routing memory (evidence rows)

---
**Document created**: 2026-07-18 (interactive; expect quorum-at-pick)
