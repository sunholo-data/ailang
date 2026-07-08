# M-MOTOKO-CLOUD-MIGRATION: Replace `claude -p` OAuth Executors with motoko + OpenRouter in Production

**Status**: Planned
**Target**: v0.21.x (infrastructure work; AILANG core unchanged)
**Priority**: P0 — deadline-driven (Claude Code OAuth subscription stops counting `claude -p` headless usage in June 2026; current production agents are on a path to either break or bleed pay-per-token billing)
**Estimated**: ~5 days infra (terraform + config + Dockerfile) + 1 week pilot observation + 1 week phased rollout = **~3 calendar weeks before final cutover**
**Dependencies**:
- ✅ **M-MOTOKO-EXECUTOR-ADAPTER** ([v0.18.0](../v0_18_0/m-motoko-executor-adapter.md), shipped) — `internal/executor/motoko/` exists as an EXECUTOR_SHAPE-conformant subprocess executor with full JSONL parse + cost telemetry
- ✅ **motoko_agent itself** ([arniwesth/motoko_agent](https://github.com/arniwesth/motoko_agent)) — autonomous LLM-loop harness with bash + file ops + tests + extensions (`context_mode`, `exa_search`, `omnigraph`, `compose`, `mcp`), TUI, JSON profiles, `max_steps` up to 50. **Full agentic peer of `claude -p`**, not eval-only.
- ✅ **M-AI-OPENROUTER** — OpenRouter routing path exists; motoko routes every model via `openrouter/<provider>/<model>`
- ⚠️ **OpenRouter account capacity** — must verify Mark's account tier can handle autonomous-agent load (per-minute and daily rate limits) before pilot. **This is a Phase 0 spike, not an assumption.**
- ⚠️ **`agent-executor-motoko` Cloud Run Job** — does not exist yet in [ailang-multivac/terraform/cloud_run_jobs.tf](https://github.com/sunholo-data/ailang-multivac/blob/main/terraform/cloud_run_jobs.tf). To be created by this work.

**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-19

---

## Axiom Compliance

This is **deployment / infrastructure work**, not a language-level change. Most axioms are N/A; the relevant ones score net-positive.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to AILANG semantics |
| A2: Replayability | 0 | Existing trace pipeline (motoko adapter already emits JSONL → standard `Result`) unchanged |
| A3: Effect Legibility | 0 | No language-level effect change |
| A4: Explicit Authority | **+1** | Removes ambient OAuth-derived Anthropic authority from the agent container; replaces with explicit per-job OpenRouter API key binding. Cost fence (no `ANTHROPIC_API_KEY` in motoko Job) is preserved/strengthened. |
| A5: Bounded Verification | 0 | N/A |
| A6: Safe Concurrency | 0 | N/A (motoko adapter v0.18.2 already solved parallel-isolation) |
| A7: Machines First | 0 | N/A |
| A8: Minimal Syntax | 0 | N/A |
| A9: Cost Visibility | **+1** | Motoko emits per-step `cost_usd` from OpenRouter; current `claude -p` OAuth path reports cost as $0 (subscription-flat, no per-task billing signal). Migration **improves** per-task cost visibility. |
| A10: Composability | 0 | N/A |
| A11: Structured Failure | 0 | Both executors emit structured `Result` already |
| A12: System Boundary | **+1** | OAuth token in `~/.claude/.credentials.json` is an opaque shared subscription credential; per-job OpenRouter API key is a properly-scoped boundary credential |

**Net Score: +3** → **Decision: Proceed**

### Hard Violation Check

- [x] A1: No nondeterminism introduced (deployment change only)
- [x] A3: No hidden effects (motoko adapter already declares its subprocess + OpenRouter call as part of its capability surface)
- [x] A4: No ambient authority — strictly improves authority scoping
- [x] A7: No human-convenience-over-machine-analysis tradeoff

---

## Problem Statement

**Current state:**

Production AILANG cloud agents run via `claude -p` (Claude Code headless mode) using OAuth tokens minted from Mark's Claude Max subscription. The token is in Secret Manager as `{prefix}-claude-code-oauth-token`, bound into 3 Cloud Run Job variants ([cloud_run_jobs.tf:80-89, 371-378, 1609-1616](https://github.com/sunholo-data/ailang-multivac/blob/main/terraform/cloud_run_jobs.tf)):

| Job | Used By | Auth |
|-----|---------|------|
| `agent-executor` | Generic agent tasks (4 workspaces) | OAuth |
| `agent-executor-go` | sprint-executor (needs Go toolchain) | OAuth |
| `agent-executor-eval` | Multi-CLI benchmark runner | OAuth |

Subscription-based billing means a flat monthly fee absorbs hundreds of autonomous agent calls — the economic foundation of the current architecture.

**The change:**

From **June 2026**, Anthropic stops counting headless `claude -p` invocations against Claude Code subscriptions (per Mark — exact policy details TBC). After that date, the OAuth path either:
1. **Stops working entirely** → all cloud agents fail until migrated, OR
2. **Falls through to pay-per-token billing** → existing budgets ([config.cloud.yaml:967-983](https://github.com/sunholo-data/ailang-multivac/blob/main/config/config.cloud.yaml#L967-L983), daily $500 global, Claude $300) get burned in hours of autonomous activity.

**However, the Claude Max subscription includes $200/month of Anthropic API credits** (separate from the headless-mode allowance). These credits can be consumed via direct `ANTHROPIC_API_KEY` use — which is exactly what the **existing `agent-executor-apikey` Cloud Run Job already supports**. This materially changes the question:

- **If current monthly Claude Code consumption (token volume × Sonnet pricing) ≤ $200**, the included API credits can cover the post-June workload via the apikey path, and motoko migration becomes a strategic improvement, not a continuity requirement.
- **If consumption > $200**, we either bleed money on overage or must migrate the overflow (or all of it) to motoko + OpenRouter (likely cheaper per-token via GLM-5, comparable via Sonnet-on-OpenRouter).

**This means Phase 0 must now also measure current token consumption** to determine whether this migration is a P0 continuity project or a P2 strategic one. The infrastructure work below is the same either way — just the urgency, default model, and rollout pace change.

**Impact:**

- **Continuity risk** (only if consumption > $200/mo): 4 active workspaces depend on the OAuth path; without migration they hit budget caps after June
- **Cost risk**: depends entirely on the consumption measurement. Three plausible regimes:
  - **< $100/mo** → $200 credit comfortably covers; migration is strategic
  - **$100-200/mo** → covered today but no headroom; migration is prudent
  - **> $200/mo** → migration is necessary; motoko via OpenRouter likely cheapest path
- **Strategic** (regardless): motoko is AILANG's own native agent harness; eating our own dogfood retires a dependency on a third-party headless mode policy we don't control

---

## Goals

**Primary Goal:** Move all production AILANG cloud agents off `claude -p` (OAuth) and onto motoko (OpenRouter) **before the June 2026 Anthropic policy change**, with no service interruption and a verified cost envelope.

**Success Metrics:**

1. All 4 workspaces in [config.cloud.yaml](https://github.com/sunholo-data/ailang-multivac/blob/main/config/config.cloud.yaml) successfully dispatching to motoko in dev environment within 2 weeks of project start
2. Pilot workspace (sunholo-websites — lowest stakes) running on motoko in production for ≥ 7 consecutive days with task-success-rate within ±5pp of pre-migration baseline
3. Per-task cost in production motoko ≤ 1.5× current claude-OAuth marginal cost ($0). **Absolute floor: ≤ $1.50/task at p95**, hard daily cap $50/workspace during pilot, $200/workspace post-rollout (TBC after Phase 4 data)
4. `claude-code-oauth-token` Secret Manager entry deleted and the 3 OAuth-bound Cloud Run Job variants torn down by end of Phase 5 — verified by `terraform plan` showing zero references
5. Coordinator dispatch unchanged — task routing logic in [internal/coordinator/task_executor.go](../../../internal/coordinator/task_executor.go) requires zero code changes (auto-discovery via blank import already in place per EXECUTOR_SHAPE)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Whether to migrate at all** vs use existing `agent-executor-apikey` + $200/mo Claude Max API credit as the post-June path | If measured monthly consumption fits comfortably under $200, the apikey path is a 30-minute config flip vs a 3-week motoko rollout. Wrong call either way wastes weeks or burns budget. | **human** | **Phase 0 (after measurement)** | **n/a — gating decision** |
| Default model: `motoko-claude-sonnet-4-6` vs `motoko-glm-5` vs `motoko-claude-haiku-4-5` | Drives steady-state cost (Sonnet ~5x Haiku, GLM ~10x cheaper still). Also drives task-success-rate; wrong default = pilot fails for capability reasons that aren't really motoko's fault | **human** | design (Phase 0) | high (re-piloting takes a week) |
| Per-workspace model choice or global default | Per-workspace lets sunholo-websites use cheap GLM while ailang-core stays on Sonnet. Global default is simpler but blunter. | **human** | design (Phase 0) | med (config-only change, but affects budget math) |
| Keep `agent-executor-apikey` (pay-per-token claude) as permanent escape hatch, or delete it too | Retaining it means a fallback if motoko can't handle a specific task class; deleting it means full commitment | **human** | Phase 5 cutover | high (reversing means re-running Phase 2 infra work) |
| Whether to build `agent-executor-motoko-go` variant (Go toolchain in container) for sprint-executor | sprint-executor tasks build/test Go code. Without Go in the motoko image, sprint-executor migration blocks until built. | **agent** | Phase 2 implementation | med (one extra Dockerfile + Cloud Run Job resource) |
| Migration order across workspaces | Wrong order risks pilot signal contamination or worst-case data loss in higher-stakes workspaces | **human** | design (Phase 0) | low (just a sequencing call) |
| Whether motoko's extensions (`exa_search`, `omnigraph`, `mcp`) are wired in the cloud Job or stripped to bare-bones | Extensions add value but also add secrets (`EXA_API_KEY`) and surface area. Start bare or start full? | **agent** | Phase 2 | low |

### Design Freeze

Before implementation begins, these must be resolved (humans):

- [ ] **Measured monthly consumption** — actual $/mo number from Firestore. Determines whether this is P0 (>$200), P1 ($100-200), or P2 (<$100). **No other decision matters until this one is made.**
- [ ] **Go/no-go on migration** — if measurement says we fit comfortably under $200/mo, recommend defer motoko migration to Q4 2026 and use `agent-executor-apikey` + the included credit as the post-June path. Otherwise proceed.
- [ ] **Default model selected** (Sonnet vs GLM-5 vs Haiku) — recommend `motoko-claude-sonnet-4-6` for parity with current claude-OAuth quality during pilot; revisit cost optimization in Phase 6
- [ ] **Per-workspace vs global model choice** — recommend per-workspace (4 lines of yaml, gives optionality)
- [ ] **Escape-hatch policy** — recommend keep `agent-executor-apikey` for 90 days post-cutover, then re-evaluate
- [ ] **Migration order** — recommend: sunholo-websites → TwilightGame → stapledons_voyage → sunholo-data/ailang (highest stakes last)
- [ ] **OpenRouter account spike PASSED** — Phase 0 verification (rate limits, billing model, BYOK option) completes with green light

---

## Solution Design

### Overview

Three changes, in three places, sequenced over three weeks:

1. **ailang-multivac/terraform**: new `agent-executor-motoko` (and `-go` variant) Cloud Run Job, new `{prefix}-openrouter-api-key` secret. No deletions in this phase.
2. **ailang-multivac/config/config.cloud.yaml**: per-workspace model field flips from `model: sonnet` (or similar) to `model: motoko-claude-sonnet-4-6`, one workspace at a time
3. **ailang-multivac/docker**: extend `Dockerfile.agent` (or new `Dockerfile.agent-motoko`) to install the motoko binary + bun + Node + the 9 motoko extension packages

**Crucially, AILANG core requires zero changes.** The motoko executor adapter is already auto-discovered via blank import in [internal/coordinator/provider_executor.go](../../../internal/coordinator/provider_executor.go); the dispatcher already routes by workspace `model:` field to whichever executor handles that model.

### Architecture

**Components:**

1. **Cloud Run Job: `agent-executor-motoko`** — Mirrors `agent-executor-pi` pattern. Binds `OPENROUTER_API_KEY` (+ `EXA_API_KEY` if extensions enabled). **Does NOT bind `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN`** (cost fence, per [ailang-multivac CLAUDE.md §5](https://github.com/sunholo-data/ailang-multivac/blob/main/CLAUDE.md)).
2. **Cloud Run Job: `agent-executor-motoko-go`** — Same as above + Go toolchain for sprint-executor (parallels `agent-executor-go`).
3. **Secret: `{prefix}-openrouter-api-key`** — Mirror `anthropic-api-key` resource shape with `prevent_destroy = true`.
4. **Dockerfile.agent-motoko** — base image + ailang + motoko binary + bun + Node + extension packages installed via `ailang init motoko-extension` flow (or pre-built into the image).
5. **Config workspace mappings** — One line edit per workspace in `config.cloud.yaml`. The coordinator reads this at dispatch time; no rebuild needed.

### Implementation Plan

**Phase 0: Cost measurement & decision spike (~6 hours)**
- [ ] **Measure current monthly Claude Code consumption** — query Firestore task history for the last 30 days; sum input + output tokens per task; multiply by current Sonnet API pricing ($3/M in, $15/M out). Result: **estimated $/month if billed via API instead of subscription**. This is the single most important number in this design doc — it sets the urgency.
  - If ≤ $100/mo → migration is P2 strategic; can take 6+ months. Keep `agent-executor-apikey` as primary post-June path, motoko as future cost-optimization.
  - If $100-200/mo → migration is P1; aim for cutover in Q3 2026 for headroom.
  - If > $200/mo → migration is P0 as designed; cutover before June.
- [ ] Verify OpenRouter account rate-limits suffice for autonomous-agent burst load (synthetic 20-concurrent-task test from local motoko)
- [ ] Confirm OpenRouter billing model (pre-pay vs post-pay, BYOK Anthropic option)
- [ ] Confirm with Anthropic the **precise** scope of the June OAuth change AND confirm $200/mo API credit policy continues post-June (documented or by direct contact)
- [ ] Make Design Freeze decisions above (informed by the measurement)

**Phase 1: Dev-environment infra (~1 day)**
- [ ] Add `google_secret_manager_secret.openrouter_api_key` to `terraform/secrets.tf` (mirror `anthropic_api_key` shape)
- [ ] Add `google_cloud_run_v2_job.agent_executor_motoko` to `terraform/cloud_run_jobs.tf` — model env wiring per [motoko-executor-adapter.md](../v0_18_0/m-motoko-executor-adapter.md)
- [ ] Add `Dockerfile.agent-motoko` to ailang repo (since build context is ailang, per [multivac CLAUDE.md §3](https://github.com/sunholo-data/ailang-multivac/blob/main/CLAUDE.md))
- [ ] `make apply ENV=dev` and `make docker-build` — verify the Job manifest deploys cleanly
- [ ] Smoke test: trigger one task to motoko Job in dev, observe completion + cost in Firestore

**Phase 2: Pilot prep (~1 day)**
- [ ] Add `agent-executor-motoko-go` Job variant if sprint-executor migration is in scope (recommend yes)
- [ ] Wire OpenRouter cost rates into `models.yml` (mostly already done in v0.18.0; verify)
- [ ] Update `config/config.cloud.yaml` budget block: per-provider `openrouter: { daily_budget: 50.0, task_max_cost: 10.0, hard_limit: true }` for pilot
- [ ] Capture pre-migration baseline: 7-day rolling task success rate per workspace from Firestore — needed for ±5pp comparison in success metric 2

**Phase 3: Pilot — sunholo-websites in dev (~3 days observation)**
- [ ] Switch `sunholo-data/sunholo-websites` workspace `model:` field from `sonnet` → `motoko-claude-sonnet-4-6` in dev
- [ ] Run normal website-builder workload for 72 hours
- [ ] Compare success rate, cost, task latency against baseline

**Phase 4: Pilot — sunholo-websites in production (~7 days observation)**
- [ ] Promote config change to production (config-only deploy via `cloudbuild-config-only.yaml` fast path — ~3 min)
- [ ] Run for 7 days
- [ ] **Go/no-go decision**: does motoko meet budget envelope + success rate gate?

**Phase 5: Phased rollout (~5 days)**
- [ ] Day 1: TwilightGame → motoko (lower stakes, sprint workflow exercises full chain)
- [ ] Day 2: stapledons_voyage → motoko
- [ ] Day 3: sunholo-data/ailang → motoko (highest stakes, last)
- [ ] Day 4-5: Observation; revert any failures

**Phase 6: OAuth teardown (~2 hours, after 2 weeks clean operation)**
- [ ] Remove `agent-executor`, `agent-executor-go`, `agent-executor-eval` OAuth bindings (delete the `CLAUDE_CODE_OAUTH_TOKEN` env block from each; keep job resources only if needed for apikey variant fallback)
- [ ] If `agent-executor-apikey` retained: confirm it has no OAuth binding (per CLAUDE.md §5)
- [ ] Delete `google_secret_manager_secret.claude_code_oauth_token` resource (requires removing `prevent_destroy` first)
- [ ] Tag commit `motoko-cutover-complete`

### Files to Modify/Create

**New files (ailang-multivac):**
- `terraform/cloud_run_jobs.tf` — append `agent_executor_motoko` + `agent_executor_motoko_go` resources (~200 LOC each, mirror existing pattern)
- `terraform/secrets.tf` — append `openrouter_api_key` resource (~15 LOC)

**New files (ailang):**
- `docker/Dockerfile.agent-motoko` — based on existing `Dockerfile.agent`, swap `claude` CLI install for `motoko` binary + bun + Node + extension packages (~80 LOC)

**Modified files (ailang-multivac):**
- `config/config.cloud.yaml` — workspace `model:` field per workspace; new `openrouter:` budget block (~20 lines edited)
- `scripts/setup-secrets.sh` — add `openrouter-api-key` to the secret list (~3 lines)

**Modified files (ailang):**
- None expected. Verify by `grep -r CLAUDE_CODE_OAUTH internal/` in ailang and confirm only `claude` executor references it.

---

## Examples

### Example 1: Workspace config change (the actual cutover, one line per workspace)

**Before** ([config.cloud.yaml:951-964](https://github.com/sunholo-data/ailang-multivac/blob/main/config/config.cloud.yaml#L951-L964)):
```yaml
workspaces:
  - id: sunholo-data/sunholo-websites
    model: sonnet
    agents: [website-builder]
```

**After:**
```yaml
workspaces:
  - id: sunholo-data/sunholo-websites
    model: motoko-claude-sonnet-4-6
    agents: [website-builder]
```

That's it. Coordinator's `task_executor.go:selectProvider` dispatches by model prefix; `motoko-*` routes to the motoko executor, which dispatches to `agent-executor-motoko` Cloud Run Job, which has only `OPENROUTER_API_KEY` bound.

### Example 2: New Cloud Run Job resource shape (terraform/cloud_run_jobs.tf, abridged)

```hcl
resource "google_cloud_run_v2_job" "agent_executor_motoko" {
  provider = google-beta
  count    = var.bootstrap ? 0 : 1
  name     = "${var.prefix}-agent-executor-motoko"
  location = var.region

  template {
    template {
      service_account = google_service_account.agent.email
      timeout         = var.agent_timeout
      max_retries     = 1

      containers {
        image = "${local.image_base}/agent-motoko:${var.agent_image_tag}"

        # OpenRouter API key (sole LLM credential — no Anthropic, no Google direct)
        env {
          name = "OPENROUTER_API_KEY"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.openrouter_api_key.secret_id
              version = "latest"
            }
          }
        }

        # NO CLAUDE_CODE_OAUTH_TOKEN — physical cost fence
        # NO ANTHROPIC_API_KEY — physical cost fence
        # (Standard GITHUB_TOKEN, OTEL, AILANG_CLOUD_PROJECT blocks identical to agent_executor)
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].image,  # CI manages image tags
    ]
  }
}
```

---

## Success Criteria

- [ ] OpenRouter account spike passes (Phase 0)
- [ ] `agent-executor-motoko` Cloud Run Job deployed in dev and a smoke-test task completes with non-zero `cost_usd` in Firestore (Phase 1)
- [ ] sunholo-websites pilot in production runs ≥ 7 days with task success rate within ±5pp of baseline (Phase 4)
- [ ] Per-task p95 cost stays within $1.50, daily workspace spend within budget (Phase 4)
- [ ] All 4 workspaces switched to motoko in production with no rollbacks (Phase 5)
- [ ] `claude-code-oauth-token` secret deleted; `terraform plan` shows zero references (Phase 6)
- [ ] OAuth-bound Job resources removed; only `agent-executor-apikey` (if retained) and motoko variants remain (Phase 6)
- [ ] Updated [CLAUDE.md §5](https://github.com/sunholo-data/ailang-multivac/blob/main/CLAUDE.md) (ailang-multivac) to document OpenRouter as primary auth path
- [ ] Migration retro doc moved to `design_docs/implemented/v0_21_x/m-motoko-cloud-migration.md` with actual cost-per-task numbers

---

## Testing Strategy

**Unit tests:** None new. Motoko adapter has its own test suite ([internal/executor/motoko/](../../../internal/executor/motoko/)) and the Terraform changes are infrastructure-only.

**Integration tests:**
- Synthetic burst test against OpenRouter (Phase 0) — 20 concurrent task spawns from local motoko, verify no 429s
- Dev-environment smoke task post-Phase 1 — single end-to-end task with motoko Job, Firestore record correct
- Pilot workload (Phases 3-4) is the real integration test

**Manual testing:**
- Verify per-job env binding by inspecting deployed Job manifest (`gcloud run jobs describe ailang-dev-agent-executor-motoko --region=europe-west1`) and confirming **no** `CLAUDE_CODE_OAUTH_TOKEN` or `ANTHROPIC_API_KEY` appears
- Spot-check cost reporting: a known-easy task should report a known-low cost on motoko (not $0 like the OAuth path did)

---

## Deferred Decisions

- **Which motoko extensions to enable by default in cloud Jobs** — agent may choose between bare-bones (no extensions) and full (`context_mode + exa_search + omnigraph + mcp`). Default proposal: bare-bones for Phase 1-4, layer in extensions in Phase 6+.
- **Whether to swap default model to `motoko-glm-5` after pilot** — defer until Phase 4 cost data lands. Could be 10x cheaper if quality holds. Mark to decide.
- **Whether `agent-executor-eval` (benchmark runner) migrates in this work or separately** — recommend separately, since benchmarks intentionally exercise multiple executors and a forced-motoko benchmark loses cross-executor comparison value.
- **Per-workspace OpenRouter sub-budgets** — defer until 30 days of post-rollout data.

---

## Non-Goals

- **Building new motoko capabilities** — motoko is already a full agentic harness. This work is purely deployment + config.
- **Adding new AILANG executors** — no codex/gemini/pi changes.
- **Migrating local development workflows** — local `claude -p` still works for interactive coding; this is about the **cloud coordinator's** autonomous workloads.
- **Removing the `claude` executor from AILANG core** — keep it. It's still useful for local dev and remains available via the apikey path.
- **OpenRouter BYOK (bring-your-own-Anthropic-key) setup** — possible future optimization but not in scope here; OpenRouter's direct-billed Sonnet pricing is the assumed cost model.

---

## Timeline

| Calendar week | Phase | Hours |
|---------------|-------|-------|
| Week 1 (days 1-2) | Phase 0 + Phase 1: spike + dev infra | ~12h |
| Week 1 (days 3-4) | Phase 2: pilot prep + baseline capture | ~8h |
| Week 1-2 | Phase 3: dev pilot (3 days passive observation) | ~3h active |
| Week 2-3 | Phase 4: prod pilot (7 days passive observation) | ~3h active |
| Week 3-4 | Phase 5: phased rollout (5 days) | ~10h active |
| Week 5 | Phase 6: OAuth teardown after 2-week stability window | ~2h |

**Total active work: ~38 hours. Total elapsed time: ~5 calendar weeks** — must start by **end of April 2026** to comfortably land before June 2026 OAuth cutoff.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| OpenRouter rate-limits trip under autonomous-agent burst load | High — could break production mid-day | Phase 0 spike with 20-concurrent test. If fails, escalate to OpenRouter (account-tier upgrade) or fall back to apikey-direct routing |
| Per-task cost on motoko-Sonnet materially higher than ±1.5x baseline | High — could blow $500/day global budget in a single run | Hard daily caps in `config.cloud.yaml`; circuit breaker pattern (already exists in coordinator). Phase 4 cost gate is the go/no-go |
| motoko's agentic loop produces a worse PR/edit quality than claude -p for sprint-executor tasks | Med — sprint workflow degrades | Pilot lowest-stakes workspace first (sunholo-websites); keep `agent-executor-apikey` escape hatch for 90 days |
| Anthropic announces a different policy than expected and the OAuth path keeps working | Low — but means migration was unneeded | Migration is still worth doing for A4/A9/A12 axiom wins and to reduce dependency on third-party headless mode. No rollback needed; the new infra is cheaper to operate either way |
| **Phase 0 measurement underestimates real consumption** (e.g., misses retry storms or eval-suite runs) | High — could declare migration P2, then run out of $200 credit mid-July | Measure with 30-day lookback **and** include a worst-case month from history. Add a Firestore burn-rate monitor that alerts at $150/mo running rate even if measurement says we're under $200. |
| **$200 credit policy changes too** post-June | High — would invalidate P2 decision | Phase 0 task explicitly confirms credit policy directly with Anthropic. If unconfirmed, default to treating it as P1 (assume credit may not continue). |
| OpenRouter outage during pilot | Med — agents fail | Same blast radius as Anthropic outage today. Retained `agent-executor-apikey` provides multi-provider fallback if explicit failover is needed (out of scope; manual switch only) |
| Dockerfile bloat from bundling motoko extensions | Low — slower cold starts | Use multi-stage build; only install runtime deps in final stage. Pi adapter's Dockerfile is the precedent (already proven) |
| Cloud Build trigger paths don't pick up `Dockerfile.agent-motoko` in ailang | Low — image won't rebuild on push | Verify Cloud Build trigger `includedFiles` covers `docker/**` — already does per [multivac CLAUDE.md §11](https://github.com/sunholo-data/ailang-multivac/blob/main/CLAUDE.md) |

---

## Related Documents

<!-- Auto-populated by Ollama neural search on "motoko cloud migration" + manually curated -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_9_0/m-cloud-health.md](../../implemented/v0_9_0/m-cloud-health.md) (0.39) — cloud health check patterns; relevant for Phase 1 smoke testing
- [design_docs/implemented/v0_17_1/m-motoko-extension-integration-sprint-plan.md](../../implemented/v0_17_1/m-motoko-extension-integration-sprint-plan.md) (0.38) — motoko extension wiring; relevant for Phase 6 deferred decision on default extensions
- [design_docs/implemented/v0_9_0/m-cloud-dispatch.md](../../implemented/v0_9_0/m-cloud-dispatch.md) (0.37) — Cloud Run Jobs dispatch mechanics

**Planned (check for overlap):**
- [design_docs/planned/v0_18_0/m-motoko-executor-adapter.md](../v0_18_0/m-motoko-executor-adapter.md) (0.41) — **direct dependency**. Provides the adapter this migration depends on. The Cloud Run Job pattern referenced there (`agent-motoko`) is exactly what Phase 1 implements.
- [design_docs/planned/v0_19_0/m-motoko-ext-per-task-sprint-plan.md](../v0_19_0/m-motoko-ext-per-task-sprint-plan.md) (0.41) — per-task extension dispatch; informs Phase 6 deferred decisions

**Cross-repo references:**
- [ailang-multivac CLAUDE.md §5](https://github.com/sunholo-data/ailang-multivac/blob/main/CLAUDE.md) — **the rule this work supersedes** ("Agent Auth: OAuth Only, NEVER API Key"). Must update this file in Phase 6.
- [arniwesth/motoko_agent README](https://github.com/arniwesth/motoko_agent) — motoko's own capability surface
- [internal/executor/motoko/motoko.go](../../../internal/executor/motoko/motoko.go) — the adapter

---

## References

- [Design Axioms](/docs/references/axioms) — 12 non-negotiable principles
- [EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md) — two-pillar contract (Pillar 1 + Pillar 2) for executor adapters
- [Anthropic Claude Code subscription terms](https://www.anthropic.com/) — TBC, gather precise wording during Phase 0

---

## Future Work

- **`motoko-glm-5` as default model** — possibly 10x cost reduction if quality holds; investigate after 30 days of Sonnet-default production data
- **BYOK OpenRouter routing** — bring-your-own Anthropic key through OpenRouter for Sonnet workloads, getting OpenRouter's tooling + direct Anthropic billing terms; investigate in Q3 2026
- **Per-task extension auto-selection** — pre-existing [v0.19.0 design](../v0_19_0/m-motoko-ext-per-task-sprint-plan.md) becomes relevant once cloud motoko is the default
- **Decommission Claude OAuth path entirely** — depends on whether the Anthropic June change forces it or merely changes billing. If still functional post-June, keep apikey variant as backup. If broken, full decommission.

---

**Document created**: 2026-05-19
**Last updated**: 2026-05-19

---

**DESIGN_DOC_PATH**: `design_docs/planned/v0_21_0/m-motoko-cloud-migration.md`
