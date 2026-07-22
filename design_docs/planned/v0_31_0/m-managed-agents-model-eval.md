# M-MANAGED-AGENTS-MODEL-EVAL — Managed Agents as a Gemini agent-mode eval vessel

**Status**: Planned (BLOCKED on external rollout — see Verification Log V4)
**Target**: v0.31.0 (build) / activation gated on Google server rollout
**Priority**: P2 (Pi already covers the immediate need; this is a second, product-fidelity vessel)
**Estimated**: ~1.5 days build + ~0.5 day activation when unblocked
**Dependencies**: existing `internal/executor/managed_agents/` (M-MANAGED-AGENTS, v0.22.0); ADC auth

## Summary

Make Vertex **Managed Agents** able to run the antigravity sandbox on a **chosen
Gemini model** (`gemini-3.6-flash` / `gemini-3.5-flash` / `gemini-3.5-flash-lite`)
so we can measure **agent-mode model performance** through Google's own managed
harness. A completed spike (2026-07-22) proved the mechanism exists, is plain
REST, and is Go-implementable — but the field it requires (`agent_config`) is not
yet recognized by the server our project reaches. This doc specifies the Go build
so it lands behind a capability preflight and activates the moment Google ships
the field to us.

## Problem Statement

We want to compare Gemini models **in agent mode**, not just standard (0-shot)
mode. Today:

**Current State:**
- `internal/executor/managed_agents/` can only invoke the fixed public agent
  `antigravity-preview-05-2026` via the legacy flat `POST .../interactions` body,
  which has **no model field**. Whatever model that agent defaults to is the only
  one we can run. (Google's 2026-07-21 announcement flips that default to
  `gemini-3.6-flash`.)
- Standard-mode model choice already works via direct Vertex `generateContent`
  (that path carries the 3.6/3.5/flash-lite suite work landed 2026-07-21/22). It
  is NOT agent mode.
- Agent-mode model choice is available via **Pi** (`pi-gemini-3-flash-preview`
  exists; more can be added), which is the pragmatic vessel and remains the
  recommendation for pure model ranking.

**Impact:**
- We cannot currently answer "how does flash-lite vs 3.6-flash behave as an
  *agent* (multi-turn, tool-using) inside Google's managed sandbox?"
- Pi answers the neutral-harness version of that question but not the
  product-fidelity version (see Non-Goals / the confound note).

## Goals

**Primary Goal:** Run the antigravity managed-agent sandbox on an explicitly
chosen Gemini model, driven from our Go eval harness, so agent-mode results can be
banked per model exactly like standard-mode results.

**Success Metrics:**
- `ailang eval-suite --agent --models managed-gemini-3-5-flash-lite,...` runs the
  antigravity sandbox on the named model and banks a result with the correct
  `model` attribution and cost.
- A capability preflight cleanly reports "not available on this project" (and
  falls back / skips) when the server rejects `agent_config`, instead of silently
  running the default model and mislabeling it.
- At least two Gemini models compared in agent mode on ≥3 benchmarks once
  unblocked, with per-model turn count + cost banked.

## High-Impact Decisions

| Decision | Why it matters | Who decides | When | Change cost |
|---|---|---|---|---|
| Agent lifecycle: create-per-run vs create-once-and-reuse per model | Cost + rate limits; created agents are real resources needing cleanup | agent (impl) | design | med |
| models.yml surface: new `managed-gemini-*` entries vs a `managed_agent_model` field on existing entries | Determines how users select the vessel | human (Mark) | design | med |
| Fallback on unavailable `agent_config`: skip vs auto-route to Pi vs hard-error | Behavior when the server hasn't shipped the field | human (Mark) | design | low |

### Design Freeze (before sprint-executor starts)
- [ ] models.yml surface chosen (new entries vs field)
- [ ] Fallback behavior chosen (skip / Pi / error)
- [ ] Agent lifecycle chosen (per-run vs reuse), incl. cleanup guarantee

## Solution Design

### Overview

Move the managed-agents executor from "invoke the one public agent" to a
**two-phase** flow, exactly as captured from the SDK wire traffic:

1. **Create** a model-bound agent (idempotent per model):
   ```
   POST https://aiplatform.googleapis.com/v1beta1/projects/{proj}/locations/{loc}/agents
   { "base_agent": "antigravity-preview-05-2026",
     "agent_config": { "type": "antigravity", "model": "gemini-3.5-flash-lite" },
     "id": "ailang-eval-gemini-3-5-flash-lite" }
   ```
2. **Interact** against the returned agent id (the existing sandbox/streaming
   path, `agentInteraction.agent = <created id>`), then extract the artifact as
   the current bridge already does.
3. **Cleanup** (`DELETE .../agents/{id}`) or reuse across runs.

This is a superset of today's executor — the fixed-agent path stays as the
`agent_config`-unavailable fallback.

### Architecture

**Components:**
1. **Agent provisioner** (new, in `internal/executor/managed_agents/`): create /
   get / delete a model-bound agent keyed by model id; idempotent (`get` before
   `create`). Returns the agent resource id used by the interaction.
2. **Capability preflight** (new): one `create`/`get` probe that detects the
   `400 Unknown name "agent_config"` rejection and marks the vessel unavailable.
   Modeled on the bogus-field control from the spike (the rejection is byte-
   identical to an unknown field, so the probe must key on the exact field name).
3. **Executor wiring** (modify `managed_agents.go`): when a task carries a target
   model, provision+use a model-bound agent; else use the legacy fixed agent.
4. **models.yml + harness plumbing**: expose the vessel (surface TBD — see
   Design Freeze) and thread the model id to the executor.
5. **Cost model** (modify): today `CostModel()` hardcodes gemini-3-5-flash
   pricing as "the default agent's underlying model." Make it read the chosen
   model's pricing from models.yml (flash-lite $0.30/$2.50, 3.6 $1.50/$7.50).

### Implementation Plan

**Phase 1: Provisioner + preflight** (~4h)
- [ ] `agents_provision.go`: create/get/delete model-bound agent (REST, ADC auth,
      reuse the existing client + Api-Revision header handling)
- [ ] capability preflight probe + typed `ErrAgentConfigUnavailable`
- [ ] unit tests with a stub server (mirror `managed_agents_test.go` style)

**Phase 2: Executor + harness wiring** (~4h)
- [ ] thread target model → executor; branch fixed-agent vs model-bound
- [ ] models.yml surface (per Design Freeze) + cost model reads chosen pricing
- [ ] cleanup/reuse per lifecycle decision

**Phase 3: Activation + eval** (~3h, gated on V4 unblock)
- [ ] live smoke: provision flash-lite agent, run 1 benchmark end-to-end
- [ ] agent-mode compare ≥2 models on ≥3 benchmarks, bank results

### Files to Modify/Create

**New files:**
- `internal/executor/managed_agents/agents_provision.go` — create/get/delete + preflight (~180 LOC)
- `internal/executor/managed_agents/agents_provision_test.go` — stub-server tests (~150 LOC)

**Modified files:**
- `internal/executor/managed_agents/managed_agents.go` — branch on target model; use provisioned agent (~60 LOC)
- `internal/executor/managed_agents/managed_agents.go::CostModel` — read chosen-model pricing (~20 LOC)
- `internal/eval_harness/models.yml` — vessel entries/field (surface TBD)
- `internal/eval_harness/models.go` — thread model id to executor (~30 LOC)

## Examples

### Example 1: Provision + run flash-lite as an agent

**Before** (today — model is whatever antigravity defaults to):
```
ailang eval-suite --agent --models gemini-3-5-flash --benchmarks fizzbuzz
# runs antigravity's default model; cannot target flash-lite
```

**After** (model-bound agent):
```
ailang eval-suite --agent --models managed-gemini-3-5-flash-lite --benchmarks fizzbuzz
# provisions an antigravity agent with agent_config.model=gemini-3.5-flash-lite,
# runs the sandbox on it, banks result labeled with the correct model + pricing
```

### Example 2: Preflight when the field isn't shipped yet

```
$ ailang eval-suite --agent --models managed-gemini-3-5-flash-lite ...
managed_agents: agent_config not available on project ailang-dev
  (server rejects it as an unknown field across all revisions/regions).
  Falling back per config: <skip | route to pi-gemini-3-5-flash-lite | error>.
```

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | The model is set via `agent_config.model` at agent-CREATE time (not on the interaction) | Introspected `google.genai._gaos.types.agents.AgentConfig` → fields `{type:'antigravity', model, max_total_tokens}` | Confirmed |
| V2 | The mechanism is plain REST (Go-implementable, no Python needed) | Captured the SDK's actual httpx request: `POST .../v1beta1/projects/ailang-dev/locations/global/agents` with `{base_agent, agent_config:{type,model}}` | Confirmed |
| V3 | The `agents` create endpoint works on our project | Control: `POST .../agents` WITHOUT agent_config → `OK` (created `ailang-spike-probe`, then DELETEd, 404 confirms gone) | Confirmed |
| V4 | **BLOCKER**: server rejects `agent_config` on our reachable endpoint | `POST` with agent_config → `400 Unknown name "agent_config"` — **identical to a bogus field `totally_bogus_zzz`** — across v1beta1, Api-Revision {none,2026-05-20,06-20,07-01,07-20,07-22}, regions {global,us-central1,us-east4} | Confirmed BLOCKED (client ahead of backend) |
| V5 | Legacy fixed-agent path still works (fallback is real) | `POST .../interactions` (flat body, agent=antigravity) → streams | Confirmed |

## Testing Strategy

**Unit tests:** provisioner create/get/delete against a stub server; preflight
correctly classifies the `Unknown name "agent_config"` rejection vs a real
success vs other 4xx.

**Integration tests:** gated behind a live flag (like existing
`managed_agents_live_test.go`); provision a flash-lite agent, run one interaction,
assert model attribution + cleanup.

**Manual:** re-run the V4 probe periodically; the day it returns non-`Unknown-name`,
the vessel is live — run the Phase 3 compare.

## Deferred Decisions

- Exact agent id naming scheme for provisioned agents — agent may choose (must be
  deterministic per model for idempotent reuse).
- Whether to reuse one agent across a whole suite run or per benchmark — agent may
  choose, guided by the lifecycle Design-Freeze decision.

## Non-Goals

- **Isolating model performance from the harness.** Managed agents measures
  **model + antigravity scaffolding together**. For neutral model ranking, Pi is
  the correct vessel and is unaffected by this doc. This vessel answers a
  *different* question: "how does model X do inside Google's managed product,"
  which is the point of building it — not a substitute for Pi.
- Custom (non-antigravity) agents / bring-your-own ADK agent (that's Agent
  Engine / `reasoningEngines`, out of scope).
- Fixing the managed-agents caching cost ($1.79/run — caching disabled when tools
  present); unchanged and orthogonal.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `agent_config` never rolls out to our project / needs preview enrollment | High | Build is cheap and behind a preflight; if it stays blocked, we've lost ~1.5 days and Pi still covers the need. Recheck V4 periodically; ask Google/PM re: preview enrollment for ailang-dev. |
| Model+scaffolding confound misread as pure model ranking | Med | Documented as an explicit Non-Goal; bank results under a `managed-` prefix so they're never pooled with Pi/standard rows. |
| Per-run agent creation hits rate/cost limits | Med | Lifecycle Design-Freeze decision (reuse per model); idempotent get-before-create. |
| Cost mislabeled (today's hardcoded gemini-3-5-flash pricing) | Med | Phase 2 makes CostModel read the chosen model's pricing. |

## Related Documents

**Implemented (may inform design):**
- [M-EVAL-AGENT-QUEUE](design_docs/implemented/v0_7_0/M-EVAL-AGENT-QUEUE.md) (0.45)
- [m-ollama-local-eval-sprint-plan](design_docs/implemented/v0_15_0/m-ollama-local-eval-sprint-plan.md) (0.43) — Pi/local agent-eval precedent
- M-MANAGED-AGENTS (v0.22.0) — the existing managed_agents executor this extends
- M-ANTIGRAVITY-CLI-MIGRATION (v0.22.0) — why Gemini CLI was retired; managed_agents is its replacement

**Planned (checked for overlap — distinct):**
- [m-eval-results-folder-structure](design_docs/planned/v0_29_0/m-eval-results-folder-structure.md) (0.45) — banking layout, not vessel mechanism

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is eval-harness infrastructure (no language-surface change), so most axioms
are N/A (0). Scored honestly rather than padded.

| Axiom | Score | Notes |
|-------|-------|-------|
| A1: Determinism | 0 | No language/runtime determinism impact; eval-side only |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect-system change |
| A4: Explicit Authority | +1 | Agent provisioning uses explicit ADC + named project; no new ambient authority |
| A5: Bounded Verification | 0 | N/A |
| A6: Safe Concurrency | 0 | N/A |
| A7: Machines First | +1 | Produces machine-banked agent-mode results per model; strengthens the eval data loop |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Fixes mislabeled cost (per-model pricing) and banks real agent-mode cost/turns |
| A10: Composability | 0 | N/A |
| A11: Structured Failure | +1 | Typed `ErrAgentConfigUnavailable` + loud preflight instead of silent wrong-model runs |
| A12: System Boundary | +1 | Managed-agent boundary is explicit; results namespaced `managed-` to avoid pooling |

**Net Score: +5** → **Decision: Move forward (build behind preflight)**

### Hard Violation Check
- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted (explicit ADC + project)
- [x] A7 (Machines First): optimizes machine-banked eval data, not human convenience

## Future Work

- If the confound matters, run a paired study: same benchmarks, same models, on
  managed-agents vs Pi, to quantify the antigravity-scaffolding delta.
- Extend the provisioner's `system_instruction`/`tools` to test AILANG-specific
  agent scaffolding inside the managed sandbox.

---

**Document created**: 2026-07-22
**Last updated**: 2026-07-22
**Spike reference**: memory `project-managed-agents-model-selection-spike`; wire calls captured via `google.genai._gaos` SDK

DESIGN_DOC_PATH: design_docs/planned/v0_31_0/m-managed-agents-model-eval.md
