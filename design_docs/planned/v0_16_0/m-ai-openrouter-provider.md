# M-AI-OPENROUTER: OpenRouter Provider with Routable Model Resolution

**Status**: Planned
**Target**: v0.16.0
**Priority**: P1 (Medium)
**Estimated**: 4-6 days (~20-30 hours)
**Dependencies**: M-UNIFIED-AI-PROVIDERS (v0.5.10, complete), M-ARCH1 (v0.6.5, planned — not blocking)

## Framing

> **AILANG AI effects support provider-routed inference with replayable model resolution. OpenRouter is the first implementation of `AI[Routeable]`.**

OpenRouter is treated as a **provider backend**, not a new language feature. The AI effect already exists; OpenRouter is one more runtime implementation alongside `openai`, `anthropic`, `gemini`, `ollama`. The interesting design work is **how to keep dynamic provider routing replayable** without violating Axiom A1 (Determinism) or A2 (Replayability).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Routing is dynamic at runtime but resolved provider/model is captured in trace, making the call replayable from trace |
| A2: Replayability | +2 | Every routing decision (requested vs resolved model, fallback chain taken, cost) recorded in trace; replay reproduces exact call |
| A3: Effect Legibility | +1 | New `AI[Routeable]` row marker makes "may pick provider at runtime" visible in the type, not hidden in HTTP code |
| A4: Explicit Authority | +1 | API key / BYOK passed as capability; no ambient access; per-provider keys distinguishable |
| A5: Bounded Verification | 0 | Type-checking unchanged; provider adapter is opaque to type system |
| A6: Safe Concurrency | 0 | No concurrency changes; calls go through existing AI effect handler |
| A7: Machines First | +1 | One adapter unlocks ~100 models for agent benchmarking, reducing per-provider boilerplate |
| A8: Minimal Syntax | 0 | No new syntax; reuses `ai.complete(...)` and existing record-typed config |
| A9: Cost Visibility | +2 | OpenRouter usage accounting (prompt/completion/cached/reasoning tokens, $cost) maps directly into existing AI trace cost fields |
| A10: Composability | +1 | One adapter composes with all OpenRouter-routed models; routing policy is a record value, composes with existing config |
| A11: Structured Failure | +1 | Provider routing failures (no fallback satisfies capabilities, all upstreams 5xx) surface as typed `AIError` variants |
| A12: System Boundary | +1 | HTTPS call to `openrouter.ai/api/v1` is a single, declared boundary — same shape as existing OpenAI adapter |

**Net Score: +11** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — routing decisions are recorded in trace, making replay deterministic
- [x] A3 (Effects): All HTTP calls go through `AI` effect; no hidden side effects
- [x] A4 (Authority): API keys passed as capability, never ambient
- [x] A7 (Machines First): Reduces per-provider boilerplate, expands model coverage for agent eval

### Decision Thresholds

Net score +11 ≥ +2, no −1 on hard-violation axioms → **Proceed**.

## Problem Statement

**Current State:**
- AILANG ships dedicated provider adapters: `internal/ai/openai/`, `internal/ai/anthropic/`, `internal/ai/gemini/`, `internal/ai/ollama/`
- Adding a new model means adding (or extending) a provider, copying ~100-300 LOC of boilerplate (per M-ARCH1)
- Agent eval suite is bottlenecked on which providers we have integrated; cannot benchmark against e.g. DeepSeek, Qwen, Mistral, Llama-405B without writing a per-provider adapter
- No mechanism for "use cheapest model that supports structured outputs and tools" — every model choice is hardcoded in `models.yml`

**Impact:**
- **Eval coverage gap**: Cannot benchmark AILANG agent performance across the long tail of frontier models
- **Vendor lock-in risk**: Adding each new vendor is multi-day work; OpenRouter unlocks ~100 models behind one HTTP API
- **Cost visibility blind spot**: We cannot easily compare $/task across models without integrating each one
- **Routing-as-policy missing**: AILANG has no concept of "fall back to cheaper model on rate limit" — useful operationally, but currently impossible to express *and trace*

## Goals

**Primary Goal:** Add OpenRouter as a provider implementing the existing `internal/ai/` Provider interface, plus a new `AI[Routeable]` effect refinement that captures dynamic model resolution in the replay trace.

**Success Metrics:**
- One new provider package `internal/ai/openrouter/` (~400-700 LOC) routes to ~100 models via OpenAI-compatible Chat Completions API
- Eval harness can target `openrouter/<vendor>/<model>` strings without per-model adapter work
- Every OpenRouter call records: `requested_model`, `resolved_model`, `resolved_provider`, `fallback_chain`, prompt/completion/cached/reasoning tokens, $cost
- Replay of a routed trace uses `resolved_model` (not `requested_model`), making the replay deterministic even if upstream routing has since drifted
- AILANG programs can declare `! {AI[Routeable]}` to opt into runtime model selection; plain `! {AI}` rejects routable providers at typecheck or load time

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `AI[Routeable]` row marker vs runtime-only flag | Determines whether routing visibility is a type-system property (compile-time) or a runtime config flag. Type-level is more legible (A3) but touches type/effect rows | human | design | high |
| Trace schema for `resolved_*` fields | Affects every AI trace consumer (dashboard, replay engine, eval harness). Adding fields later means migrating historic traces | human | design | high |
| Replay policy: use `resolved_model` or re-route | If replay re-routes, replays drift over time as upstream availability changes. If replay pins `resolved_model`, we get determinism but may hit decommissioned models | human | design | med |
| Package location: `internal/ai/openrouter/` vs `internal/ai/providers/openrouter/` | Existing pattern is flat `internal/ai/<provider>/`. Sub-namespacing implies a future restructure | agent | design | low |
| Capability-requirement enforcement: at request build vs by upstream | If we enforce `require: ["structured_outputs"]` locally we need a capability table; if we delegate to OpenRouter their routing handles it but errors are vaguer | agent | compile | med |
| BYOK key-passing model | `AI[BYOK]` as a separate effect refinement, or a config-only flag? Effect-level is more legible but adds another row marker | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved by a human:

- [ ] **Effect-row design**: Confirm `AI[Routeable]` syntax/semantics (or pick alternative spelling like `AI<Routeable>` or `AI{routeable: true}`)
- [ ] **Trace schema**: Approve the new `resolved_*` and `fallback_chain` fields for the trace event format; coordinate with trace-debugger / dashboard owners
- [ ] **Replay policy**: Decide pin-to-resolved vs re-route-on-replay (recommendation: pin-to-resolved by default, optional `--reroute` for replay)
- [ ] **BYOK design**: Decide whether `AI[BYOK]` is a separate marker or folded into capability/config

## Solution Design

### Overview

Three layers, each independently reviewable:

1. **Thin OpenRouter provider** — Implements existing `ai.Provider` interface against OpenRouter's OpenAI-compatible `/api/v1/chat/completions` endpoint. No new types in `internal/ai/`. ~400 LOC.
2. **AIRequest/AIRoutingPolicy IR** — Generalize the request struct in `internal/ai/provider.go` to include optional `routing` and `outputSchema` fields. Existing providers ignore them; OpenRouter consumes them. ~150 LOC delta.
3. **`AI[Routeable]` effect refinement + trace schema** — Add a row marker on the `AI` effect that gates routable providers. Extend trace event schema with `resolved_*` fields. ~300 LOC across types/effects/trace.

### Architecture

```
internal/ai/
├── provider.go                     # MODIFIED: extend Request with Routing, OutputSchema
├── routing.go                      # NEW: AIRoutingPolicy, ResolvedRoute, capability enum
│
├── openrouter/                     # NEW: OpenRouter adapter
│   ├── client.go                   # HTTP client, auth, base URL config
│   ├── chat.go                     # Chat Completions endpoint, request shaping
│   ├── routing.go                  # Maps AIRoutingPolicy → OpenRouter `provider` field
│   ├── usage.go                    # Maps OpenRouter usage block → AILANG cost fields
│   ├── handler.go                  # implements effects.AIHandler
│   └── *_test.go
│
└── (existing providers: openai/, anthropic/, gemini/, ollama/)

internal/effects/
├── ai.go                           # MODIFIED: gate routable providers behind AI[Routeable] row
└── ai_routeable.go                 # NEW: row-marker type, capability check

internal/types/                     # MODIFIED if AI[Routeable] is row-level
└── effect_rows.go                  # Add Routeable, BYOK, ReplayOnly markers

internal/eval_harness/
└── api_openrouter.go               # NEW: thin wrapper for eval harness use

stdlib/std/ai/providers/
└── openrouter.ail                  # NEW: AILANG-side provider constructor

models.yml                          # MODIFIED: add openrouter/* entries
```

**Components:**

1. **OpenRouter HTTP adapter** — Borrows ~80% of OpenAI client code (path: `/v1/chat/completions`, OpenAI-compatible request/response shape). Differences: `provider` field for routing, expanded `usage` block (cached_tokens, reasoning_tokens, cost_usd), model strings of form `<vendor>/<model>`.

2. **AIRoutingPolicy IR** — A record type embedded in `ai.Request`:

   ```go
   type AIRoutingPolicy struct {
       Order            []string         // Preferred provider order (e.g., ["anthropic", "openai"])
       AllowFallback    bool             // If true, fall through Order on failure
       Require          []AICapability   // Hard requirements: STRUCTURED_OUTPUTS, TOOLS, VISION, etc.
       MaxPricePerMTok  *decimal.Decimal // Optional cap, USD per million tokens
       Prefer           RoutePreference  // CHEAPEST | FASTEST | MOST_RELIABLE
   }
   ```

3. **ResolvedRoute trace fields** — Added to every AI trace event (zero-valued for non-routable providers):

   ```go
   type ResolvedRoute struct {
       RequestedModel   string   // What user/program asked for, e.g., "openrouter/auto"
       ResolvedModel    string   // What OpenRouter actually used, e.g., "anthropic/claude-sonnet-4.5"
       ResolvedProvider string   // Underlying provider, e.g., "anthropic"
       FallbackChain    []string // Models tried, in order
       PromptTokens     int
       CompletionTokens int
       CachedTokens     int
       ReasoningTokens  int
       CostUSD          decimal.Decimal
   }
   ```

4. **`AI[Routeable]` row marker** — A new effect row that providers self-declare they need. The OpenRouter handler's `Capabilities()` method returns `{Routeable: true}`. The AI effect handler checks: if program declared `! {AI}` (no Routeable), reject routable providers at handler dispatch with a typed error. If program declared `! {AI[Routeable]}`, allow.

5. **Replay policy** — Default: replay uses `resolved_model` from the trace, calling that specific model directly (bypassing OpenRouter routing on replay). Optional `--reroute` flag re-runs routing logic, useful for "what would happen now" analysis but not deterministic.

### Implementation Plan

**Phase 1: Thin adapter** (~6-8 hours)
- [ ] Scaffold `internal/ai/openrouter/` from `internal/ai/openai/` template
- [ ] Implement `Client`, `ClientOption`, `NewClient(apiKey, ...)`
- [ ] Implement Chat Completions request/response (model, messages, system, max_tokens, temperature)
- [ ] Map OpenRouter `usage` block (incl. cached/reasoning tokens, cost) into existing `Response.Usage`
- [ ] Add `models.yml` entries for top-10 OpenRouter-routed models for eval coverage
- [ ] Unit tests with recorded HTTP fixtures (no live calls in CI)
- [ ] Wire into `cmd/ailang/ai_handlers.go` and `internal/eval_harness/`

**Phase 2: Routing policy IR** (~6-8 hours)
- [ ] Define `AIRoutingPolicy`, `RoutePreference`, `AICapability` enum in `internal/ai/routing.go`
- [ ] Extend `ai.Request` with optional `Routing *AIRoutingPolicy`
- [ ] OpenRouter adapter: translate `AIRoutingPolicy` → OpenRouter `provider` field (`order`, `allow_fallbacks`, `require_parameters`, `data_collection`)
- [ ] Other providers (openai/anthropic/gemini/ollama) ignore `Routing` (or error if non-nil and `len(Order) > 0`, depending on freeze decision)
- [ ] AILANG-side: `stdlib/std/ai/providers/openrouter.ail` exposes `provider({...})` constructor returning a configured `Provider` value
- [ ] Tests for routing policy translation and rejection paths

**Phase 3: AI[Routeable] effect + trace schema** (~6-8 hours)
- [ ] Add `Routeable` (and stub `BYOK`, `ReplayOnly`) row markers to effect-row representation
- [ ] AI effect handler checks routable-provider capability against declared effect row
- [ ] Extend AI trace event schema with `ResolvedRoute` block
- [ ] Update trace-debugger / dashboard consumers (or coordinate handoff)
- [ ] Replay engine: prefer `resolved_model` over `requested_model` when replaying a routable call
- [ ] Documentation: `docs/docs/guides/ai-routing.md` covering effect markers, replay semantics, OpenRouter setup
- [ ] Example: `examples/ai_openrouter_routing.ail` showing `AI[Routeable]` declaration and policy

### Files to Modify/Create

**New files:**
- `internal/ai/routing.go` — `AIRoutingPolicy`, `RoutePreference`, `AICapability` (~120 LOC)
- `internal/ai/openrouter/client.go` — HTTP client, options (~150 LOC)
- `internal/ai/openrouter/chat.go` — Chat Completions request/response (~200 LOC)
- `internal/ai/openrouter/routing.go` — Policy → OpenRouter `provider` field translation (~80 LOC)
- `internal/ai/openrouter/usage.go` — Usage/cost mapping (~60 LOC)
- `internal/ai/openrouter/handler.go` — `effects.AIHandler` impl (~80 LOC)
- `internal/ai/openrouter/*_test.go` — Unit tests with HTTP fixtures (~300 LOC)
- `internal/effects/ai_routeable.go` — Row marker + capability check (~80 LOC)
- `internal/eval_harness/api_openrouter.go` — Eval-harness wrapper (~100 LOC)
- `stdlib/std/ai/providers/openrouter.ail` — AILANG-side provider constructor (~50 LOC)
- `examples/ai_openrouter_routing.ail` — Example program (~40 LOC)
- `docs/docs/guides/ai-routing.md` — Routing guide (~200 lines)

**Modified files:**
- `internal/ai/provider.go` — Add `Routing`, `OutputSchema` to `Request` (~30 LOC delta)
- `internal/effects/ai.go` — Gate routable providers behind `AI[Routeable]` row (~50 LOC delta)
- `internal/types/effect_rows.go` — Add `Routeable` row marker (~40 LOC delta)
- `internal/trace/events.go` — Add `ResolvedRoute` field to AI events (~30 LOC delta)
- `cmd/ailang/ai_handlers.go` — Wire OpenRouter into CLI (~20 LOC delta)
- `models.yml` — Add openrouter/* model entries (~80 lines added)
- `CHANGELOG.md` — v0.16.0 section

**Total estimate**: ~1,400 new LOC, ~300 LOC modified, ~300 LOC tests.

## Examples

### Example 1: Plain (non-routable) call to a specific OpenRouter model

```ailang
import std/ai
import std/ai/providers/openrouter

func summarize(text: string) -> Result[string, AIError] ! {AI} =
  ai.complete({
    provider: openrouter.provider({
      model: "anthropic/claude-sonnet-4.5"
    }),
    system: "Summarize precisely.",
    input: text,
    maxTokens: 500
  })
```

Behavior: pinned model, no routing, replayable with the exact model. `! {AI}` is sufficient because no fallback is requested.

### Example 2: Routable call with fallback policy

```ailang
import std/ai
import std/ai/providers/openrouter

func summarize(text: string) -> Result[string, AIError] ! {AI[Routeable]} =
  ai.complete({
    provider: openrouter.provider({
      model: "openrouter/auto",
      route: {
        order:         ["anthropic", "openai", "google"],
        allowFallback: true,
        require:       ["structured_outputs"],
        prefer:        Cheapest
      }
    }),
    system: "Summarize precisely.",
    input: text,
    maxTokens: 500
  })
```

Behavior: OpenRouter selects from anthropic → openai → google in order, requiring structured-output capability. Trace records `requested_model = "openrouter/auto"`, `resolved_model = "anthropic/claude-sonnet-4.5"` (or whichever was chosen), full fallback chain, and cost. Effect row is `AI[Routeable]`.

### Example 3: Type error — routable provider used with plain AI effect

```ailang
func summarize(text: string) -> Result[string, AIError] ! {AI} =
  ai.complete({
    provider: openrouter.provider({
      model: "openrouter/auto",
      route: { allowFallback: true }
    }),
    -- ...
  })
```

Behavior: rejected at handler dispatch (or earlier, depending on freeze decision) with a typed error: `AIError.RouteableProviderNotAllowed { required: "AI[Routeable]", declared: "AI" }`.

### Example 4: Runtime config (TOML)

```toml
[ai.providers.openrouter]
base_url = "https://openrouter.ai/api/v1"
api_key_env = "OPENROUTER_API_KEY"
default_model = "openrouter/auto"

[ai.providers.openrouter.routing]
allow_fallback = true
require_structured_outputs = true
require_tool_calling = false
```

### Example 5: Trace event for a routed call

```json
{
  "event": "ai.complete",
  "provider": "openrouter",
  "route": {
    "requested_model":   "openrouter/auto",
    "resolved_model":    "anthropic/claude-sonnet-4.5",
    "resolved_provider": "anthropic",
    "fallback_chain":    ["anthropic/claude-sonnet-4.5"],
    "prompt_tokens":     1234,
    "completion_tokens": 312,
    "cached_tokens":     1000,
    "reasoning_tokens":  0,
    "cost_usd":          "0.00428"
  },
  "request_hash":  "sha256:...",
  "response_hash": "sha256:..."
}
```

## Success Criteria

- [ ] `internal/ai/openrouter/` package exists, implements `ai.Provider`, has ≥80% test coverage
- [ ] Eval harness can run benchmarks against `openrouter/<vendor>/<model>` model strings without per-model adapter work
- [ ] AI trace events include `ResolvedRoute` block for routable calls; zero-valued for fixed-model calls
- [ ] `AI[Routeable]` effect row exists; using a routable provider under plain `! {AI}` produces a typed error
- [ ] Replay of a routed trace uses `resolved_model` and produces matching response (modulo expected nondeterminism)
- [ ] Example `examples/ai_openrouter_routing.ail` runs end-to-end
- [ ] `make verify-examples` passes
- [ ] `make test` passes (no regression in existing providers)
- [ ] Documentation: `docs/docs/guides/ai-routing.md` published; CHANGELOG entry; design doc moved to `design_docs/implemented/v0_16_x/`
- [ ] At least one eval benchmark exists comparing same task across 3+ OpenRouter-routed models, demonstrating cost-visibility

## Testing Strategy

**Unit tests:**
- HTTP fixture tests for OpenRouter client (request shaping, response parsing, usage mapping)
- `AIRoutingPolicy` → OpenRouter `provider` field translation, including edge cases (empty order, conflicting capabilities, missing prefer)
- Capability gate: routable provider + plain `AI` effect → typed error
- Trace event population: every routable call has `ResolvedRoute`, every non-routable has zero-value
- Replay determinism: replay uses `resolved_model`, not `requested_model`

**Integration tests:**
- End-to-end with mock OpenRouter HTTP server (test harness)
- Eval harness running same prompt across 3 OpenRouter-routed models, validating cost/usage capture
- AILANG program with `! {AI[Routeable]}` runs through full type-check → handler → trace pipeline

**Manual testing:**
- Live call against real OpenRouter (`OPENROUTER_API_KEY` set, gated behind `make test-live` not in CI)
- Verify trace fields appear in dashboard / `ailang trace list`
- Run example file: `ailang run examples/ai_openrouter_routing.ail`

## Deferred Decisions

The following are intentionally left open for the implementer (agent latitude):

- **Streaming support** — Phase 1 is non-streaming. Streaming may be added in a follow-up; agent may stub `Stream()` returning `ErrNotImplemented`.
- **Capability table location** — Where the `AICapability` enum's actual capability strings live (a YAML file vs hardcoded const). Agent may choose; recommend YAML if more than ~10 capabilities are needed.
- **Decimal type for cost** — `decimal.Decimal` (shopspring) vs `*big.Rat` vs string. Agent may pick to match existing trace cost handling — check `internal/trace/events.go` first.
- **Model-string canonicalization** — Whether `models.yml` entries use `openrouter/anthropic/claude-sonnet-4.5` or `anthropic/claude-sonnet-4.5` with provider context. Agent may choose; document the choice.
- **OpenAI adapter generalization** — Whether to refactor `internal/ai/openai/` to share a base with OpenRouter (since both speak OpenAI Chat Completions). Agent may defer this to M-ARCH1; do **not** block this milestone on it.
- **Eval harness model list** — Which top-N OpenRouter models to add to `models.yml` for benchmark coverage. Agent may choose 8-12 representative models spanning vendors and price tiers.

## Non-Goals

- **Other routing meta-providers** — Together.ai, Replicate, Anyscale, etc. Out of scope; this milestone is OpenRouter-specific. The `AIRoutingPolicy` IR should be general enough to host them later but we do not implement them.
- **Streaming responses** — Deferred to a follow-up. Phase 1 is request/response only.
- **OpenRouter web search / image generation features** — Their preview features (web search tool, image inputs) deferred. Text-in / text-out only.
- **Fine-grained per-call key rotation / multi-key BYOK** — `AI[BYOK]` is stubbed as a row marker but full BYOK semantics deferred to a separate design doc.
- **Cost budget enforcement** — `MaxPricePerMTok` is captured in the policy and passed to OpenRouter, but AILANG-side budget enforcement (halt on exceed) is out of scope; trace records cost so a follow-up can build budgets on top.
- **OpenRouter `transforms` API** — middle-out compression etc.; not needed for v0.16.0 launch.

## Timeline

**Week 1** (~12-16 hours):
- Phase 1: Thin adapter complete, wired into CLI and eval harness
- Phase 2: AIRoutingPolicy IR, OpenRouter consumes it

**Week 2** (~10-12 hours):
- Phase 3: `AI[Routeable]` row marker, trace schema extension, replay-engine update
- Documentation, examples, CHANGELOG

**Week 3** (~4-6 hours):
- Eval-harness benchmark (cost comparison across 3+ models)
- Buffer for live-call testing, dashboard handoff
- Move design doc to `design_docs/implemented/v0_16_x/`

**Total: ~26-34 hours across 2-3 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Replay drift: `resolved_model` decommissioned by upstream, replay fails | Med | Default replay pins `resolved_model`, surfaces typed error if model gone. Optional `--reroute` for "what would happen now" |
| Trace schema change breaks dashboard / trace-debugger consumers | High | Coordinate with dashboard owner before Phase 3; new fields are additive (zero-valued for non-routable calls), no breakage for fixed-model traces |
| OpenRouter API drift from OpenAI Chat Completions shape | Low | Version-pin to current OpenRouter API in `client.go`; HTTP fixtures detect drift in CI |
| `AI[Routeable]` row marker collides with other planned effect-row work | Med | Coordinate with type-system owner; recommend simple boolean-flag row marker initially, generalize later if other markers (BYOK, ReplayOnly) need similar treatment |
| Hidden nondeterminism if `ResolvedRoute` not captured for some code path | High (axiom violation) | Make `ResolvedRoute` field non-optional in trace event for OpenRouter handler; unit test asserts every OpenRouter call writes it |
| Cost-field rounding inconsistencies between OpenRouter response and AILANG storage | Low | Use exact decimal type matching existing trace cost handling; round only at display time |

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_5_10/m-unified-ai-providers.md](../../implemented/v0_5_10/m-unified-ai-providers.md) — Existing `internal/ai/` Provider interface this milestone extends
- [design_docs/implemented/v0_10_0/m-ai-image-generation.md](../../implemented/v0_10_0/m-ai-image-generation.md) — Precedent for adding capabilities to AI providers
- [design_docs/implemented/v0_7_0/m-eval-ollama-local-models.md](../../implemented/v0_7_0/m-eval-ollama-local-models.md) — Precedent for new provider expanding eval coverage

**Planned (check for overlap):**
- [design_docs/planned/v0_13_0/m-arch1-ai-provider-base-class.md](../v0_13_0/m-arch1-ai-provider-base-class.md) — Provider base-class refactor; **not blocking**, but OpenRouter adapter should be written so it slots into the base class cleanly when M-ARCH1 lands
- [design_docs/planned/v0_13_0/m-arch5-error-handling-strategy.md](../v0_13_0/m-arch5-error-handling-strategy.md) — Error-typing strategy that `AIError.RouteableProviderNotAllowed` should follow
- [design_docs/planned/v1_0_0/m-agent-orchestration.md](../v1_0_0/m-agent-orchestration.md) — Agent orchestration may want to use `AI[Routeable]` for cost-aware routing

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) — Block-universe determinism (Axiom A1/A2 underpinnings)
- [OpenRouter API docs](https://openrouter.ai/docs) — Chat Completions, provider routing, BYOK
- `internal/ai/openai/` — Closest existing adapter; OpenRouter is mostly a fork of this
- `internal/effects/ai.go` — Existing AI effect handler this milestone extends
- `models.yml` — Model registry where OpenRouter-routed models will be added

## Future Work

- **`AI[BYOK]`** — Full bring-your-own-key semantics, including per-call key rotation and provider-key allowlists. Currently stubbed as a row marker only.
- **`AI[ReplayOnly]`** — An effect row that forbids live calls; response must come from trace/cache. Useful for deterministic test runs and offline agent eval. Stubbed here, designed in a follow-up.
- **Other routing providers** — Together.ai, Replicate, Anyscale adapters using the same `AIRoutingPolicy` IR.
- **Cost budgets** — AILANG-side `AIBudget` enforcement (halt or downgrade on exceed); trace already records cost so this is an additive layer.
- **Streaming** — `Stream()` method on the OpenRouter provider, with backpressure semantics consistent with other streaming providers.
- **Capability autodetection** — Query OpenRouter's `/api/v1/models` and auto-populate the capability table; today it is hand-maintained.
- **OpenAI adapter unification** — Once M-ARCH1 (provider base class) lands, refactor OpenRouter and OpenAI to share the OpenAI Chat Completions implementation.

---

**Document created**: 2026-05-03
**Last updated**: 2026-05-03

DESIGN_DOC_PATH: design_docs/planned/v0_16_0/m-ai-openrouter-provider.md
