# Sprint Plan — M-MISSION-COST-CHAINS

**Design doc**: `design_docs/planned/v0_30_0/m-mission-cost-chains.md`
**Branch / worktree**: `sprint/m-mission-cost-chains` @ `.claude/worktrees/cost-chains` (base `origin/dev` @ `093f58860`)
**Planner**: sprint-planner (Opus), mission iter-100, 2026-07-24
**Executor / evaluator (next)**: Opus executor (this worktree) → Sonnet evaluator
**Verify profile**: `go-compiler` — `make quick-install && make build`, `make test`, targeted `go test ./internal/observatory/... ./internal/server/...`, plus `bash scripts/check_boundaries.sh`.
**Risk level**: MEDIUM (M1 lower than the doc assumed — wiring already exists; the real risk is the missing per-stage `model`).

---

## TL;DR for the controller

- **M1 is SMALLER than the doc's "expose/load provider cost specs into observatory" framing.** That wiring **already exists**: `internal/observatory/pricing.go` provides `CalculateCostFromTokens(model, tokensIn, tokensOut) float64`, importing the `models.yml` rate registry from `internal/eval_harness` (a legal apps→tools import; `check_boundaries.sh` does not police it). M1 is a **read-side rollup classification** using an EXISTING rate function.
- **The real M1 constraint (new, not in the doc):** `chain_stages` has **no `model` column** (confirmed: INSERT/SELECT columns in `store_chains.go:301` & `:349`; DDL predates migrate_v8; only `provider`, `cost`, `tokens_in/out`). To resolve a rate you need a model. The model is available per-stage only for **eval stages** (the `eval_assessment` JSON blob carries `Model`, `models_chains.go:138`) and on child **spans** (`spans.model`). M1 estimation is therefore only *possible* for stages where a model can be recovered — and Mark's rule already makes an unresolvable model → `unknown` (never faked). This is consistent, but the executor must implement estimation **per-stage in Go (not in the SQL aggregate)**, resolving the model from `eval_assessment` / spans, because `SUM(cs.cost)` in SQL cannot see a model.
- **Reality-check #1 (the key one): YES, quota-marking is structurally sound with NO migration.** Evidence below. Ship M1 + M2 + M3 in one sprint but **sequence M1 → M2 → M3** (M1 is independently landable and de-risks the rest).

---

## Reality-check results

### RC#1 — M2 quota-marking soundness (NO-migration): **YES, structurally sound.**
The mission-control skill's **METERED-SPEND LEDGER** (`.claude/skills/mission-control/SKILL.md:433–443`) is explicit:
> "keep a running per-iteration tally of METERED dollars (every codex run's reported cost, every managed_agents `CostUSD`, every quorum reviewer bill — **subscription/quota-bucket spend does NOT count**) … Record the final tally as a `metered=$X.XX` field in the evidence row."

And Gate 4 (`SKILL.md:409–411`) records the **routing-evidence row = `(role, model)`** actually used.

So per role the skill has:
- **Metered lanes** (codex / managed_agents / quorum reviewer): `role, provider, model, $` (dollars are reported; token counts are NOT separately tracked by the skill — only the dollar figure).
- **Quota lanes** (headless `claude -p`, Agent-tool sub-agents on Opus/Sonnet/Fable subscription): `role, model` only. **No dollars, and NO reportable token count** — subscription runs do not surface token usage to the skill.

Therefore when M2 (Gate 4) posts the iteration chain, quota-lane stages are posted `tokens=0, cost_usd=0` **because the skill literally has no token count to post**. M1's read-side gate (`tokens>0 && cost==0 → estimate`) then **structurally excludes** them: `tokens=0` → never estimated. **No `cost_status` / `quota_bucket` schema field is required for correctness.** This matches the doc's claim (M1 Design bullet, doc lines 84–91) and Mark's scoping decision.

**Belt-and-braces (recommended, still no migration):** M2 posts quota stages under `source = mission:v1/iter-<N>` and can carry the human-readable bucket in the existing free-text `agent_id` (e.g. `agent_id="sprint-executor (quota:opus)"`) OR in the `handoff_to`/`human_feedback` free-text fields — so the bucket is *visible* in `chains view` (M2 acceptance) without a new column. The `quota_bucket=<fable|opus|sonnet>` the doc mentions is a **display convention encoded in existing text fields**, not a DB column. This is the concrete no-migration exclusion the doc asked the planner to specify. **No escalation to Mark needed** — a no-migration path is proven to exist.

### RC#2 — rollup location + rate registry reachability: **CONFIRMED, with a premise correction.**
- Rollup computes cost via **SQL aggregation**, not Go: `GetChainStatusCounts` (`store_chains_query.go:178`, `COALESCE(SUM(total_cost),0)`) and `GetChainStatsByAgent` (`:216`, `COALESCE(SUM(cs.cost),0)`). CLI surface: `cmd/ailang/chains_stats.go` (`chainsStatsCommand`, prints `Total Cost: $%.4f`).
- **Rate registry is ALREADY wired into observatory.** `internal/observatory/pricing.go:69` `CalculateCostFromTokens(model, tokensIn, tokensOut) float64` loads `models.yml` via `eval_harness.LoadModelsConfig` and is already called from `otlp_receiver.go:359,582`. It follows the no-silent-fallback rule (returns `0.0` on unresolvable model — the executor must map that to `cost_status=unknown`, NOT `$0`). **Premise correction:** the doc's "M1 wiring task = expose/load provider cost specs into the observatory package" is **already done**; M1 does not need to touch `pkg.AIProviderCost` or `configdriven.computeCost`.

### RC#3 — write path + schema re-verification: **CONFIRMED.**
- Write path: `handleCreateChain` (`internal/server/handlers_chains.go:289`) → `obsBackend.CreateChain`; stages via `handleCreateStage` (`:330`) → `CreateStage`; metrics via `UpdateStageMetrics` (`store_chains.go:635`). Confirmed as the doc describes (doc verification-log row `handlers_chains.go:286`).
- `ChainStage` schema (`internal/observatory/models_chains.go:82`): `Cost float64` at `:111`, `TokensIn/TokensOut int` at `:112–113`. **Zero `cost_status` / `cost_estimated` / `quota_bucket` fields** — re-verified 2026-07-24. Also confirmed **no `model` field** on the stage row (see TL;DR constraint).

### RC#4 — premise corrections recorded
1. **M1 rate-wiring already exists** (`pricing.go`) — M1 shrinks to read-side classification. (RC#2)
2. **`chain_stages` has no `model` column** — estimation must be per-stage in Go, resolving model from `eval_assessment`/spans; unresolvable → `unknown`. (New constraint, not in doc.)
3. **Rollup is SQL-aggregate today** — estimation cannot live in the `SUM()`; the executor adds a Go post-pass over per-stage rows (or a new stats method that returns per-stage classification). This is a design constraint the doc did not spell out.
4. **Boundary is clear** — `eval_harness` is the *tools* layer; apps→tools is allowed and already exercised. No boundary work needed.

---

## Milestone breakdown (sequenced M1 → M2 → M3)

### M1 — cost-attribution rollup fallback (scoped inference)  ~0.5–0.75d · ~180 LOC (impl+tests)
**Deliverable:** a read-side cost classifier in `internal/observatory` that, per stage, computes
`cost_status ∈ {reported, estimated, quota, unknown}` under Mark's scoped rule, and a rollup that
reports reported / estimated / unknown totals + stage/token counts **separately**, surfaced through
`chains stats`.

**Implementation notes (binding):**
- Add a per-stage classifier function in observatory (e.g. `ClassifyStageCost(stage) (status, costUSD)`), using the EXISTING `CalculateCostFromTokens`. Resolve the model from, in order: `eval_assessment.Model` (JSON, already read on `GetChainStages`), then child `spans.model`; if none → `unknown`.
- Rule (verbatim from Mark's scoped decision, doc lines 76–82):
  - `cost>0` → **reported** (untouched).
  - `tokens>0 && cost==0 && model resolves to NON-ZERO metered rate` → **estimated** (`CalculateCostFromTokens`), flagged.
  - `tokens>0 && cost==0 && model unresolvable` → **unknown** (never `$0.0000`). Model resolving to a `$0` rate (free/local) rolls up to `$0` naturally, NOT faked as metered.
  - `tokens==0` (quota lanes post this) → **quota** / left untouched → NEVER estimated.
- Because the current rollup is a SQL `SUM`, add a Go rollup path that fetches per-stage rows and classifies, returning split totals. Extend `chainStatsResult` (`chains_stats.go`) with `estimated_cost`, `unknown_stage_count`, `cost_estimated` flag; print reported/estimated/unknown separately. Budget comparisons emit a **visible incomplete-data warning** (or fail in a `--strict` mode) when unknown stages exist — no silent $0.
- **No model or rate is guessed.** Unresolvable model ⇒ `unknown`.

**Acceptance (from doc §M1 a–d):**
- (a) `chains stats` shows a non-zero, `cost_estimated`-flagged cost for a token-bearing stage that reported no cost AND resolves to a non-zero metered rate.
- (b) a stage WITH a self-reported cost is unchanged (no double-count).
- (c) a stage with tokens but no resolvable model → `unknown` (and a `$0`-rate model → `$0`), never a fabricated metered `$0.0000`.
- (d) a quota/free-marked stage (`tokens==0`) is NEVER estimated.
- Rollup reports reported / estimated / unknown cost + stage/token counts separately; budget comparison warns or `--strict`-fails on incomplete data.
- Regression tests assert all four cases (estimate / unchanged-reported / unknown / not-estimate-quota).

**Files (est.):** `internal/observatory/pricing.go` or new `internal/observatory/cost_classify.go` (+test), `cmd/ailang/chains_stats.go`, `internal/observatory/store_chains_query.go` (per-stage fetch for rollup if needed).

---

### M2 — mission-loop ingest (Gap B)  ~0.75–1d · ~150 LOC + SKILL edit
**Deliverable:** one chain per mission iteration, posted by the mission-control SKILL Gate 4 to the
EXISTING `handleCreateChain` / `handleCreateStage` / `UpdateStageMetrics` endpoints; source
`mission:v1/iter-<N>` (portable `mission:<name>/iter-<N>`). No new storage, no new endpoints.

**Implementation notes (binding):**
- Gate 4 additive step: after writing the evidence row, POST the iteration chain (single source of truth = the evidence row → its projection).
- Stages = the loop's spend units `(role, provider, model, tokens?, $)` from what actually ran.
  - **Metered lanes** (codex / managed_agents / quorum): post `cost_usd=<metered $>`, model as run; `tokens` if known else 0.
  - **Quota lanes** (headless claude / Agent-tool subs): post `tokens=0, cost_usd=0`, and encode the bucket in an EXISTING free-text field (`agent_id="…(quota:opus)"` or `handoff_to`) — **no schema change** (RC#1). This structurally keeps M1 from estimating them.
- **Fail-soft + bounded + LOUD spool:** if the server is down, buffer to a JSONL spool with a **hard cap (≤100 entries / size cap)**; every buffer event emits an explicit `stderr` warning; on overflow **drop-oldest with a one-line stderr notice**. NEVER block or fail the iteration on telemetry. Next iteration flushes the spool.

**Acceptance (from doc §M2):**
- `ailang chains list` shows the iteration chain (source `mission:v1/iter-<N>`).
- `chains view <id>` shows role stages with `$` and the quota-bucket attribution (via the free-text field).
- E2E-class runs (gemini/codex) appear with their real metered cost.
- Spool is bounded+loud: a forced server-down path emits the stderr warning, respects the 100-entry cap, and drop-oldest fires with a notice (unit-testable on the spool component).

**Files (est.):** `.claude/skills/mission-control/SKILL.md` (Gate 4 additive posting step + bounded-spool spec); a small helper (shell in the skill, or a `cmd/ailang` subcommand `chains post-iteration` if the executor prefers a testable Go path — **preferred**, so the spool/cap logic has Go tests). If a Go subcommand is added: `cmd/ailang/chains_post.go` (+ bounded-spool component + test).

---

### M3 — budget rollup surface  ~0.25–0.5d · ~90 LOC (impl+tests)
**Deliverable:** `ailang chains stats --by-mission` (alias-friendly `--by-source-prefix`): per-mission
metered total vs `MISSION_METERED_BUDGET_USD`, per-bucket quota-stage counts, top-N most expensive
stages. CLI only (no dashboard).

**Implementation notes (binding):**
- Add `--by-mission` (and/or `--by-source-prefix <p>`) flag to `chains_stats.go`. Group chains by `source_ref` prefix `mission:*`. Reuse the M1 classifier so metered vs estimated vs quota are split.
- Show: per-mission metered total, comparison to `MISSION_METERED_BUDGET_USD` (env; warn if over), per-bucket quota-stage counts (parsed from the free-text bucket), top-N expensive stages.

**Acceptance (from doc §M3):**
- `ailang chains stats --by-mission` returns per-mission metered total, budget comparison, per-bucket quota counts, and top-N stages — in one command, no dashboard.
- `--by-mission` no longer errors `flag provided but not defined` (the doc's confirmed-absent state).
- Uses M1's split totals (metered/estimated/quota never conflated).

**Files (est.):** `cmd/ailang/chains_stats.go` (+ a query helper in `store_chains_query.go` for source-prefix grouping, +tests), `cmd/ailang/help.go` (flag docs).

---

## Day / milestone schedule (~1.5–2d)

| Day | Work |
|---|---|
| Day 1 AM | **M1**: `ClassifyStageCost` + Go rollup path + `chains_stats` split output; regression tests (4 cases). `make quick-install && go test ./internal/observatory/...`. |
| Day 1 PM | **M1** finish + `check_boundaries.sh`; **M2 start**: `chains post-iteration` Go helper + bounded/loud spool component + spool tests. |
| Day 2 AM | **M2**: SKILL.md Gate 4 additive posting step (metered $ + quota tokens=0 + bucket in free-text); `go test ./internal/server/...` for the write path. |
| Day 2 PM | **M3**: `--by-mission` flag + source-prefix grouping + budget compare + top-N; tests; help.go; full `make test`. Update CHANGELOG + move design doc to `implemented/` on green. |

---

## Success metrics / definition of done
- All milestone acceptance criteria pass with regression tests.
- `make build`, `make test`, `go test ./internal/observatory/... ./internal/server/...` green.
- `bash scripts/check_boundaries.sh` green (no new forbidden imports).
- No silent $0: unresolvable-cost stages surface as `unknown`; quota lanes surface as quota (never estimated, never faked).
- CHANGELOG.md updated (grouped); design doc moved `planned/ → implemented/v0_30/` on completion.
- No schema migration introduced (Mark constraint).

---

## Risks for the controller
1. **Per-stage model recovery (MEDIUM).** `chain_stages` has no `model` column. M1 estimation only works where a model is recoverable (eval_assessment JSON / spans). For mission stages, M2 supplies the model at post time (metered lanes carry it). If a stage has tokens but no recoverable model, M1 correctly yields `unknown` — this is by design, not a bug, but the *coverage* of "estimated" rows depends on model availability. Executor must not widen this to a guess.
2. **Rollup is SQL-aggregate today (LOW-MEDIUM).** Estimation cannot live in `SUM()`. The executor adds a Go classification pass over per-stage rows; watch performance on large windows (the codebase already has `SpanLite`/N+1 perf history — keep the per-stage fetch scoped to the time window).
3. **M2 is partly a SKILL.md edit (LOW).** The bounded-spool logic should live in **Go** (a `chains post-iteration` subcommand) so it is unit-testable; the SKILL should just invoke it. If the executor keeps the spool in shell, the evaluator cannot test the cap/drop-oldest — flag this preference.
4. **`MISSION_METERED_BUDGET_USD` env dependency (LOW).** M3 budget compare reads it (default $5 per SKILL:436). No silent fallback: if unset, show "budget unset" rather than compare against 0.

## Achievability
**All three milestones fit one sprint (~1.5–2d)** given M1 is smaller than the doc assumed (wiring exists). **Sequence M1 → M2 → M3**: M1 is independently landable and provides the classifier M2/M3 reuse. If time runs short, M1+M2 alone deliver the substrate (M3 is a thin CLI surface over M1's split totals) and can land in a follow-up — but the estimate supports all three.

---

SPRINT_PLAN_PATH: design_docs/planned/v0_30_0/m-mission-cost-chains-sprint-plan.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-MISSION-COST-CHAINS.json
