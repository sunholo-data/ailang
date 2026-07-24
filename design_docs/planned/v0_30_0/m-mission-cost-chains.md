# M-MISSION-COST-CHAINS: make `ailang chains` the mission's cost tracker — fix $0 attribution + ingest loop activity

**Status**: IN-SPRINT (iter-100, 2026-07-24 — Mark scoped-inference decision `4e1348adb` folded into
the M1 normative body by the controller, NO re-quorum; routed to sprint-planner). Originally Planned
(Mark 2026-07-18 — "lets keep an eye on these budgets, perhaps a cost tracker. I think actually that
should all appear in ailang chains CLI... dont know if all this activity is still flowing in there"
— verified: it is NOT)
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
**Status → DECIDED by Mark 2026-07-24 ("scoped inference") → route to sprint-planner**: M1's
tokens×rate estimation applies ONLY to stages with tokens>0 AND no self-reported cost AND no
quota-bucket marking. Quota-lane stages are $0-BY-DESIGN (`cost_status=quota`, never estimated,
never presented as unknown) — subscription burn is bucket-visible, not dollar-faked. A persisted
`cost_status` schema field may follow later as specced; not required for M1. This resolves the
round-2 soundness objection by scoping, not by schema migration. M2/M3 unchanged (unobjected).
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

**Cost provenance — SCOPED INFERENCE (Mark decision 2026-07-24, `4e1348adb`; folded from the Status
header into this normative body by the iter-100 controller, NO re-quorum per the apply-verbatim
precedent iters 98/99):** the round-2 quorum's convergent objection was that *unscoped* CLI-side
inference would mis-attribute legitimately-free/quota `$0` stages as estimated metered spend (the
`float64` schema can't distinguish absent-cost from reported-$0). **Mark resolved this by SCOPING the
inference, NOT by a schema migration.** The stage schema (`ChainStage`,
`internal/observatory/models_chains.go:82`, `Cost float64` at ~:110) has NO provenance field today —
re-verified 2026-07-24 (iter-100): zero `cost_status`/`cost_estimated`/`quota_bucket` references
anywhere in `internal/observatory/` or `internal/server/`. M1 therefore computes `cost_status ∈
{reported, estimated, quota, unknown}` at **read time in the rollup** under Mark's SCOPED rule:
- `cost>0` → **reported** (untouched, never re-estimated).
- `tokens>0 && cost==0 && stage NOT quota/free-marked && model resolves to a NON-ZERO metered rate`
  → **estimated** (computed tokens×rate, flagged `cost_estimated=true`).
- `tokens>0 && cost==0 && model unresolvable` → **unknown** (never presented as `$0.0000`); a model
  that resolves to a **$0 rate** (free local models) naturally rolls up to `$0` WITHOUT being faked
  as metered.
- **Quota-lane stages are `$0`-BY-DESIGN (`cost_status=quota`)** — NEVER estimated, NEVER shown as
  unknown; subscription burn is bucket-visible (M2/M3), not dollar-faked.
- **Planner reality-check (the round-2 gemini point Mark's M1 decision did not spell out):** M1's
  soundness over a store that will ALSO hold M2's mission stages rests on quota lanes carrying NO
  token count reportable to the mission (Agent-tool / headless-`claude` subscription runs don't
  surface token usage) — so Mark's `tokens>0` gate excludes them **structurally**, no persisted
  marker required. The planner MUST verify this holds (M2 posts quota stages `tokens=0,
  cost_usd=0`); IF any quota stage can carry `tokens>0`, scope estimation by an explicit
  mission-posted quota signal with NO migration (source-prefix `mission:*` exclusion, or an existing
  field) — a persisted `cost_status`/`quota_bucket` column stays deferred (Mark: "may follow later;
  not required for M1").
Prior art: `computeCost` (`internal/ai/configdriven/provider.go:279`) already returns `""` (empty
string, NOT 0.0) when no cost data is configured — the rollup adopts the same
distinguish-no-data-from-free discipline.
Acceptance: (a) `chains stats` shows a non-zero, `cost_estimated`-flagged cost for a token-bearing
stage that reported no cost AND resolves to a non-zero metered rate; (b) a stage WITH a self-reported
cost is unchanged (no double-count); (c) a stage with tokens but no resolvable model is
`cost_status=unknown` (and a $0-rate model rolls up to `$0`), never a fabricated metered `$0.0000`;
(d) **a quota/free-marked stage is NEVER estimated** (Mark scoping — the round-2 soundness fix).
Rollups report reported / estimated / quota / unknown cost and stage/token counts separately; budget
comparisons emit a visible incomplete-data warning or fail in strict mode. No model or rate is
guessed. Regression tests assert: the rollup ESTIMATES (not $0) for the token-bearing/no-cost/
metered-rate case; surfaces the unresolvable-model case as `unknown` (not $0); and does NOT estimate
a quota/free-marked stage.

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

**Verdict: PARKED needs-human-review (iteration 97) → ✅ RESOLVED by Mark 2026-07-24 (`4e1348adb`).**
The one-revision-one-requorum gate was spent (Rev-1 → re-quorum), so the residual convergent objection
went to Mark as a design decision (correctly — it was a genuine schema-design fork, not a
reviewer-verbatim refinement, so NOT a carve-out case). **Mark chose "scoped inference": estimate ONLY
for token-bearing / no-cost / no-quota-bucket stages; quota lanes are `$0`-by-design (`cost_status=quota`);
no schema migration for M1.** This resolves the round-2 soundness objection **by scoping the inference
so it structurally never touches free/quota stages** (folded into the M1 Design section above),
overriding the reviewers' "persistence is REQUIRED" conclusion — Mark's prerogative as the human
principal. **The quorum still did its job: it caught the free-$0/quota-corruption collision BEFORE any
code was written, which is exactly what the M1 scoping now guards against.**

### ✅ RESOLVED by Mark 2026-07-24 — `4e1348adb` (was: ⛔ Human fork for Mark)
1. **M1 provenance approach** — the reviewers required persistence; **Mark chose SCOPED read-side
   inference instead** (no migration): estimate only for token-bearing / no-cost / **no-quota-bucket**
   stages; quota lanes `$0`-by-design (`cost_status=quota`). M1 stays a **CLI-side ~0.5d rollup fix**,
   not a schema-migration sprint. A persisted `cost_status` field may follow later, not now.
2. **`quota_bucket` on `ChainStage`** — **DEFERRED, not required for M1** (Mark: no schema migration).
   Re-verified 2026-07-24: the field does not exist. **M2 planner task:** post quota stages so M1's
   `tokens>0` gate excludes them structurally (quota lanes carry no reportable token count → post
   `tokens=0, cost_usd=0`), OR encode the bucket in an existing field / `mission:*` source prefix —
   no migration. Only escalate back to Mark if the planner proves a no-migration path impossible.
3. Narrow-refinement carve-out already **ratified** (Mark, iter-98 `m-budget-scoping-bug`); it did NOT
   apply here (genuine design gap, not verbatim) — Mark's direct decision unblocked this instead.
**Routing (this fold, iter-100):** controller folded the scoped fix (above), NO re-quorum →
sprint-planner (opus) → executor (opus, worktree) → evaluator (sonnet). M2/M3 direction unchanged and
unobjected.

## Related
- `m-cost-per-success-kpi` (clause 5 — this is its data substrate; sequence THIS first)
- [m-mission-portability](m-mission-portability.md) — `mission:<name>/…` source naming keeps World in the same tracker
- Memory: `reference-headless-claude-billing-rig` (billing lanes), routing memory (evidence rows)

---
**Document created**: 2026-07-18 (interactive; expect quorum-at-pick)
