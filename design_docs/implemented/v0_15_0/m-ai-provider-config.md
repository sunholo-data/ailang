# M-AI-PROVIDER-CONFIG: Config-Driven AI Providers via Package Manifests

**Status**: Planned
**Target**: v0.15.0
**Priority**: P0 (architectural foundation — closes the AI extension surface, prerequisite for [M-AI-STREAMING-HELPER](../v0_17_0/m-ai-streaming-helper.md), prevents the next motoko-style binary fork)
**Estimated**: 3-4 days (~24-32 hours)
**Dependencies**: None blocking. Composes naturally with [M-ARCH1](../v0_13_0/m-arch1-ai-provider-base-class.md) (Go-side provider base class refactor).
**Author**: Claude + Mark
**Created**: 2026-05-04

---

## Executive Summary

AILANG advertises a package system, but the **AI effect** — the most-used and most-valuable effect — is **closed-extensible**. Adding a new provider today requires editing the hardcoded if/else chains in [cmd/ailang/exec.go:356-384](../../../cmd/ailang/exec.go#L356-L384) and [cmd/ailang/ai_handlers.go:88-207](../../../cmd/ailang/ai_handlers.go#L88-L207), submitting an upstream PR, and recompiling. There is no `RegisterProvider` function, no plugin loading, no extension hook. The five hardcoded providers (`openai`, `anthropic`, `gemini`, `ollama`, `openrouter`) are the entire universe.

This is exactly the pressure that produced the [arniwesth/ailang motoko fork](https://github.com/arniwesth/ailang/tree/motoko) (~1,500 LOC of Go to add OpenRouter routing + custom OpenAI base-URL — work that pre-dated v0.16.x's M-AI-OPENROUTER landing it upstream). Without this milestone, every new provider — vLLM, llama.cpp, Together, Groq direct, Anyscale, Fireworks, DeepInfra, Perplexity, Mistral native, Cohere, Bedrock, Vertex — produces the same fork pressure.

This milestone makes AI providers configurable via `[[ai_provider]]` blocks in package `ailang.toml`. The runtime scans installed packages at load time, harvests provider configs, and registers them as first-class providers. New providers ship as packages, integrate fully with the AI effect (budget tracking, AI cap gating, trace spans, error normalization), and require **zero Go code** for any HTTP-shaped provider that fits the declarative shape catalog.

**Scope (in priority order):**

1. **`[[ai_provider]]` schema** in `ailang.toml` — request shape, auth shape, cost declaration, capability flags, streaming params.
2. **Generic config-driven provider** in Go — one ~300-LOC implementation that consumes the schema and produces `internal/ai.Provider` instances. Runs through existing effect/budget/trace machinery.
3. **Package loader integration** — scans installed deps, harvests `[[ai_provider]]` blocks, registers in the AI handler.
4. **Request shape catalog** — `openai_chat`, `anthropic_messages`, `simple_completion` (Ollama-style). Escape hatch via `request_template`.
5. **Auth shape catalog** — `bearer`, `x-api-key`, `query-param`, `none`. Escape hatch via `auth_headers`.
6. **Example package** — `pkg/sunholo/ai_vllm` (or similar) — proves end-to-end flow without Go changes.
7. **Documentation** — `docs/docs/guides/custom-ai-providers.md`; updated external-consumers guide.

**Explicitly OUT OF SCOPE:**

- Custom auth flows (AWS SigV4 / Azure AD / OAuth) — stay Go-side, can ship as built-in providers
- Non-HTTP transports (gRPC, WebSocket-only) — stay Go-side
- Streaming for config-driven providers — folded into [M-AI-STREAMING-HELPER](../v0_17_0/m-ai-streaming-helper.md), depends on this milestone
- Per-package binary code execution / Go plugin loading / WASM modules — explicitly rejected (cross-platform pain, security, build complexity)
- AI provider plugin marketplace / discovery UI — separate v0.18+ design

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Provider behavior is determined by static TOML at package install — no new nondeterminism |
| A2: Replayability | +1 | Config-driven providers produce the same trace shape as built-in providers; replay works uniformly |
| A3: Effect Legibility | +1 | All AI providers (built-in and config-driven) flow through the same `! {AI}` effect — no second-class path |
| A4: Explicit Authority | +2 | **Critical fix**: closes the gap where adding a provider via `std/net.httpPost` would have bypassed AI cap gating. This milestone preserves authority across all providers |
| A5: Bounded Verification | 0 | Type-checking unaffected |
| A6: Safe Concurrency | 0 | None |
| A7: Machines First | +2 | Packages can ship custom providers without binary fork; AI agents can write provider configs as data, not Go code |
| A8: Minimal Syntax | 0 | No new AILANG syntax — only TOML schema |
| A9: Cost Visibility | +2 | **Critical fix**: per-provider cost declarations plumb into existing budget tracker. Without this, custom providers would have zero cost visibility |
| A10: Composability | +2 | Packages compose new AI capabilities into the core language without recompile |
| A11: Structured Failure | +1 | Per-shape error mapping normalizes diverse provider error formats to `AIError` |
| A12: System Boundary | +2 | Formalizes the AILANG↔external-AI-API boundary as configuration data; the boundary is auditable |

**Net Score: +13** → **Decision: ✅ Strong proceed**

### Hard Violation Check

- [x] A1 (Determinism): Config is static; no implicit nondeterminism
- [x] A3 (Effects): Config-driven providers preserve the AI effect contract
- [x] A4 (Authority): AI cap remains the gate; this milestone *preserves* authority that Option A (raw HTTP) would have broken
- [x] A7 (Machines First): The whole point — close the extension surface to unblock package-shipped providers

---

## Motivating Evidence

### The motoko fork (1,500+ LOC of Go to add a provider)

The arniwesth/ailang motoko fork added custom OpenAI base-URL routing, OpenRouter prefix routing, and provider-specific builtins by editing `internal/ai/openai/`, `internal/effects/`, and `internal/builtins/`. ~1,500 LOC across 6+ files. v0.16.x M-AI-OPENROUTER subsequently landed OpenRouter upstream, but only by repeating the same pattern: Go-side hardcoded provider, sixth entry in the if/else chain. The system did not become more open; it grew one more closed entry.

### BlackMage feedback (May 2026)

> *"The AI integration was in general the most confusing part — I was considering simply rolling my own. Ideally, adding a new provider should be easy, ie as a package. In particular, once you leave Anthropic/OpenAIs walled garden, there are a plethora of different endpoints/models — all behaving slightly different. Same goes with llama.cpp vs vLLM hosted models."*

This is one external consumer's report. The dynamic generalizes: the long tail of LLM providers (vLLM, llama.cpp, Together, Groq, Anyscale, Fireworks, DeepInfra, Perplexity, Mistral, Cohere) is broader than any single upstream PR cadence can absorb. Without an extension API, every consumer who needs one of these forks the binary or works around it via a non-AI HTTP path (which breaks budget, capability, and trace).

### The accidental discovery during M-AI-STREAMING-HELPER design

While designing [M-AI-STREAMING-HELPER](../v0_17_0/m-ai-streaming-helper.md), an early sketch routed token streaming through `std/stream` + `std/net` directly, taking `(baseUrl, apiKey, model)` from caller code. This **accidentally bypassed the AI effect entirely** — and would have shipped without budget tracking, AI cap gating, or trace integration. That's an architectural bug masquerading as an ergonomic feature: making a hard thing easy by abandoning the effect system. The fix is *not* to bolt the AI cap onto a Net path; it's to make the AI provider registry extensible so `std/ai.call("vllm-local/llama-3.1-70b", prompt)` works through the proper machinery.

### Relationship to M-EFFECT-REFINEMENT (Phase 1 shipped, AI port deferred)

[M-EFFECT-REFINEMENT Phase 1](../../implemented/v0_15_x/m-effect-refinement-phase1.md) (v0.15.0, ✅ shipped) ships parameterised effect syntax (`!{E[mode=X]}`) with Rand as the pilot. AI is **not** ported in Phase 1; bare `!{AI}` keeps working via back-compat aliasing. When the AI port lands (deferred phase, see [m-effect-refinement.md "Modal AI" example](../v1_0_0/m-effect-refinement.md)), the mode taxonomy will be:

- `!{AI[mode=fixed]}` — static-configured providers (openai/anthropic/gemini/ollama and **all config-driven providers from this milestone**)
- `!{AI[mode=routeable]}` — OpenRouter-style dynamic routing
- `!{AI[mode=replay-only]}` — deterministic offline replay
- `scope=byok` — orthogonal capability for bring-your-own-key

**Implication for this milestone**: config-driven providers are inherently `mode=fixed` — they declare a static endpoint/auth and reject `AIRoutingPolicy` (per the OpenRouter compatibility section above). When parameterised AI ships, `[[ai_provider]]` declarations auto-inherit `mode=fixed`; no schema change needed for Phase 1.

**Forward-compat hook**: schema v2 may add an optional `mode = "fixed" | "routeable" | "replay-only"` field on `[[ai_provider]]` if/when a config-driven provider needs to declare a non-default mode (e.g. a routing aggregator that ships as a package). Out of scope for v1; tracked in Future Work.

### Relationship to M-AI-OPENROUTER (already shipped)

[M-AI-OPENROUTER](../../implemented/v0_16_x/m-ai-openrouter-provider.md) (M1-M4 landed in v0.16.x) introduced [`AIRoutingPolicy`](../../../internal/ai/routing.go) — a **request-side IR** carrying provider order, fallback, capability requirements, price ceiling, and sort preference for OpenRouter's dynamic routing. This milestone (M-AI-PROVIDER-CONFIG) introduces **registration-side metadata** for static config-driven providers. The two are orthogonal and complementary:

- **Vocabulary alignment**: `AIProviderCapabilities` TOML keys match the stable wire identifiers of [`AICapability`](../../../internal/ai/routing.go) — `tool_calling`, `json_mode`, `streaming`, `vision`, `structured_outputs`. One vocabulary serves both registration declarations and request-time `AIRoutingPolicy.Require`.
- **Routing policy rejection**: per the OpenRouter design (line 6 of `routing.go`: *"all other providers must reject a non-zero policy with `ErrRoutingNotSupported` so callers cannot accidentally mask their intent"*), config-driven providers reject non-zero `AIRoutingPolicy` requests. Generic provider returns `ErrRoutingNotSupported` when the request carries `policy.HasRouting() == true`. No silent fallback; no implicit OpenRouter-style routing for config-driven providers.
- **Cost vs price**: registration-time `cost.input_per_1m_usd` (provider's static price metadata for budget tracker) and request-time `AIRoutingPolicy.MaxPricePerMTok` (caller's per-call price ceiling for OpenRouter to filter providers) are different concerns. Both can coexist on the same call.
- **Built-in OpenRouter stays built-in**: OpenRouter's routing logic, structured-output support, and provider-rejection semantics live in `internal/ai/openrouter/` — not migrated to config-driven. Per D4 (built-ins stay built-in).

### The package system already exists; the AI effect is the only major effect that ignores it

`std/fs`, `std/net`, `std/stream`, `std/ai`, `std/clock`, `std/random`, `std/io` — packages can extend behavior layered on top of all of them via pure AILANG. Only the AI effect has *closed-set* providers in the runtime. This is an inconsistency the package system was built to eliminate.

---

## Goals

**Primary Goal:** Make adding a new HTTP-shaped AI provider a **zero-Go-code, package-only** operation that integrates fully with AILANG's AI effect, budget tracking, capability gating, and trace system.

**Success Metrics:**

- A new package adding a vLLM endpoint (or any OpenAI-compatible endpoint) ships in **<50 LOC** total across `ailang.toml` + a one-line AILANG re-export module. No Go.
- Budget, AI cap, and trace spans behave identically for config-driven and built-in providers — verified by trace inspection in eval baselines.
- The `[[ai_provider]]` schema covers OpenAI-compat, Anthropic-shaped, and Ollama-shaped providers with no schema escape hatch needed for the long tail.
- BlackMage's stated friction (*"once you leave the walled garden, plethora of endpoints"*) becomes a documented one-paragraph recipe per provider, not a fork project.
- Eval baselines: at least one config-driven provider (e.g. local llama.cpp) is exercised in the standard suite.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `[[ai_provider]]` array vs single `[ai_provider]` | A package shipping multiple providers (e.g. one for vLLM-chat, one for vLLM-completion) is plausible; array is more flexible but adds parsing complexity | human | design | med |
| Request-shape catalog scope: 2, 3, or N shapes | More shapes = more code paths to test; too few = escape hatches everywhere | human | design | high |
| Auth-shape catalog scope | Custom auth (SigV4/Azure-AD/OAuth) vs strict bearer/header/query split — what fraction of providers fit the catalog? | human | design | high |
| Cost declaration: per-token / per-call / both | Built-in providers track per-token; some providers (e.g. fixed-fee endpoints) charge per call. Schema must accommodate without privileging one model | human | design | med |
| Routing: prefix match (`vllm/...`) vs declared model list vs both | Prefix is simple; declared list is precise; both adds flexibility but UX cost | human | design | med |
| Where the generic provider lives in Go: new `internal/ai/configdriven/` package vs extending base class | Affects code organization and how M-ARCH1 integrates | agent | design | low |
| Schema-version field in `[[ai_provider]]` | Migration path when v2 schema adds fields (e.g. tool-use templating later) | human | design | low |
| Whether to error or warn on unknown `request_shape` | Strict error = forces upgrade; warning = lenient but masks bugs | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Array form**: `[[ai_provider]]` (multiple per package). Locked.
- [x] **Request-shape catalog v1**: `openai_chat`, `anthropic_messages`, `simple_completion` (Ollama-style). Locked. Per Mark (2026-05-04): "We can assume Arni is ok with these three request shapes and will add more or edit later on" — schema versioning supports v2 additions; no need to gate v1 implementation on more shape data.
- [x] **Auth-shape catalog v1**: `bearer`, `x-api-key`, `query-param`, `none`. Custom auth declares as `auth_headers` (literal header dict with `${ENV_VAR}` interpolation). Locked.
- [x] **Cost declaration**: both per-token (`input_per_1m_usd`, `output_per_1m_usd`) and per-call (`per_call_usd`). Either or both populated; budget tracker uses what's available. Locked.
- [x] **Routing**: prefix-match by default (`<provider_name>/...`); optional `models` array for explicit allow-list. Locked.
- [x] **Escape hatch**: `request_template` (Go template) for shapes outside the v1 catalog; `auth_headers` for auth flows outside the v1 catalog. Locked.

**All Design Freeze items resolved.** Implementation may begin.

---

## Solution Design

### Schema (TOML, in package `ailang.toml`)

```toml
[package]
name = "sunholo/ai_vllm"
version = "0.1.0"
ailang = ">=0.15.0"

[[ai_provider]]
schema_version = 1
name = "vllm"                                     # used as routing prefix: call("vllm/llama-3.1-70b", ...)
endpoint = "http://localhost:8000/v1/chat/completions"
request_shape = "openai_chat"                     # or "anthropic_messages" | "simple_completion" | "custom"
response_path = "$.choices[0].message.content"    # JSONPath to extract result text
error_path = "$.error.message"                    # JSONPath for error normalization

# Auth: pick one shape
auth = { type = "bearer", env = "VLLM_API_KEY" }
# OR auth = { type = "x-api-key", env = "ANTHROPIC_API_KEY" }
# OR auth = { type = "query-param", name = "api_key", env = "GEMINI_API_KEY" }
# OR auth = { type = "none" }
# OR auth_headers = { Authorization = "Bearer ${VLLM_API_KEY}", X-Org = "${VLLM_ORG}" }  # escape

# Cost (either or both; budget tracker uses what's available)
cost = { input_per_1m_usd = 0.0, output_per_1m_usd = 0.0, currency = "USD" }
# OR cost = { per_call_usd = 0.001 }

# Capabilities (optional; routing/dispatch uses these for negotiation).
# Keys match the wire identifiers in internal/ai/routing.go AICapability so
# request-time AIRoutingPolicy.Require and registration-time declarations
# share one vocabulary. Valid keys: tool_calling, json_mode, streaming,
# vision, structured_outputs.
capabilities = { tool_calling = false, json_mode = true, streaming = true, vision = false, structured_outputs = true }

# Streaming (optional; consumed by M-AI-STREAMING-HELPER)
[ai_provider.streaming]
enabled = true
endpoint = "http://localhost:8000/v1/chat/completions"   # often same; field for cases where they differ
delta_path = "$.choices[0].delta.content"
reasoning_path = "$.choices[0].delta.reasoning_content"  # optional; for o1/DeepSeek-R1
done_sentinel = "[DONE]"                                  # OpenAI-style; Anthropic uses event types instead

# Models allow-list (optional; if absent, prefix-match accepts any)
[ai_provider.models]
allowed = ["llama-3.1-70b", "llama-3.1-8b", "qwen-2.5-72b"]
```

### Generic provider implementation (Go, ~300 LOC)

New file: `internal/ai/configdriven/provider.go`. Implements `internal/ai.Provider` interface. Behavior:

1. Receives the parsed `[[ai_provider]]` config at construction.
2. On `Call(req)`:
   - Builds request body from `request_shape` template (`openai_chat` → `{messages, model, ...}`; `anthropic_messages` → `{messages: [{role, content: [{type, text}]}], model, max_tokens, ...}`; `simple_completion` → `{prompt, model, ...}`).
   - Constructs HTTP request: URL from `endpoint`, headers from `auth` shape.
   - Resolves `${ENV_VAR}` interpolation in `auth_headers` against `os.Getenv` at call time (not load time — fresh env each call).
   - POSTs, handles HTTP errors → `AIError`.
   - On 2xx: extracts response via `response_path` (JSONPath). On 4xx/5xx: extracts via `error_path`, classifies as retryable per shape conventions.
   - Records cost via existing budget tracker using `cost.input_per_1m_usd` × token count (or `per_call_usd`).
   - Emits trace span identical in shape to built-in providers (`ai.call`, attributes for model, provider, tokens, latency, cost).
3. On `CallJson(req, schema)`: same but adds `response_format` per shape.
4. Capability checks: refuses calls that require capabilities the config flags as false (e.g. `tools = false` and the request includes tool definitions → `AIError{ code: "CapabilityNotSupported" }`).

### Package loader integration

`internal/pkg/loader.go` (existing) currently parses `[package]`, `[exports]`, `[dependencies]`, etc. Extend it to:

1. Parse `[[ai_provider]]` blocks during manifest load.
2. Pass harvested configs to a new `internal/ai.RegisterConfigDrivenProviders([]ConfigDrivenProviderSpec)` registration call after package resolution completes.
3. Conflict resolution: if two packages declare the same provider name, **error at load time** (don't silently override). User resolves by removing one or aliasing.
4. Validation: schema version, required fields, JSONPath syntactic validity (deep validation deferred to first call).

### Routing

Extend the existing model→provider resolution in [cmd/ailang/exec.go:356-384](../../../cmd/ailang/exec.go#L356-L384):

1. **Built-in providers** keep their hardcoded entries (`openai`, `anthropic`, `gemini`, `ollama`, `openrouter`).
2. **Config-driven providers** register a prefix matcher: `<provider_name>/...` → that provider. Optional `models.allowed` further constrains.
3. Resolution order: built-in providers checked first, then config-driven (alphabetical for determinism). On conflict, built-in wins (with warning).
4. CLI: `--ai vllm` works the same as `--ai openai`; `--model vllm/llama-3.1-70b` routes via the `vllm` package.

### Files to Modify/Create

**New files:**
- `internal/ai/configdriven/provider.go` — generic provider, ~300 LOC
- `internal/ai/configdriven/shapes.go` — request-shape templates (openai_chat, anthropic_messages, simple_completion), ~200 LOC
- `internal/ai/configdriven/auth.go` — auth-shape handlers, ~80 LOC
- `internal/ai/configdriven/jsonpath.go` (or use existing dep) — response/error extraction, ~50 LOC
- `internal/ai/configdriven/provider_test.go` — unit + integration with httptest.Server, ~400 LOC
- `internal/pkg/configdriven_provider.go` — schema + parser, ~80 LOC
- `internal/pkg/configdriven_provider_test.go` — schema parsing tests, ~150 LOC
- `examples/runnable/custom_provider_demo.ail` — end-to-end demo against echo-AI server, ~30 LOC
- `docs/docs/guides/custom-ai-providers.md` — comprehensive guide, ~250 lines
- *(Optional, separate repo)* `pkg/sunholo/ai_vllm/ailang.toml` + thin re-export module — example external package

**Modified files:**
- `internal/pkg/manifest.go` — extend `PackageManifest` struct with `[]ConfigDrivenProviderSpec`, ~30 LOC
- `internal/pkg/loader.go` — invoke `RegisterConfigDrivenProviders` after dependency resolution, ~20 LOC
- `internal/ai/handler.go` — accept config-driven provider registrations, ~50 LOC
- `cmd/ailang/exec.go` — extend provider resolution to consult config-driven registry, ~40 LOC
- `cmd/ailang/ai_handlers.go` — same as exec.go (parallel surface), ~40 LOC
- `docs/docs/guides/packages.md` — document the new manifest section
- `docs/docs/guides/external-consumers.md` (planned in m-external-consumer-dx.md) — cross-reference
- `CHANGELOG.md` — v0.15.0 entry

### Implementation Plan

**Day 1: Schema + parsing** (~6 hours)
- [ ] `ConfigDrivenProviderSpec` struct in `internal/pkg/`
- [ ] TOML parsing + validation (required fields, schema version, JSONPath syntax check)
- [ ] Schema parsing tests with golden TOML fixtures
- [ ] Conflict-detection test (two packages, same provider name)
- [ ] Manifest extension; loader hook (no-op registration)

**Day 2: Generic provider with `openai_chat` shape** (~8 hours)
- [ ] `internal/ai/configdriven/provider.go` implementing `Provider` interface
- [ ] `openai_chat` request shape template
- [ ] `bearer` auth shape
- [ ] `simple_completion` request shape (Ollama-style)
- [ ] Response extraction via JSONPath
- [ ] Error normalization
- [ ] Budget tracking integration (cost calculation, ledger update)
- [ ] Trace span emission
- [ ] Integration tests with httptest.Server (success, auth fail, 5xx retry, malformed response)

**Day 3: Anthropic shape + remaining auth + routing** (~6 hours)
- [ ] `anthropic_messages` request shape
- [ ] `x-api-key`, `query-param`, `none` auth shapes
- [ ] `auth_headers` escape with `${ENV_VAR}` interpolation
- [ ] Routing: prefix matcher in `cmd/ailang/exec.go` + `ai_handlers.go`
- [ ] `models.allowed` enforcement
- [ ] CLI `--ai <config-driven-name>` plumbed end-to-end
- [ ] Integration tests for Anthropic shape

**Day 4: Docs + example + eval baseline + release wiring** (~8 hours)
- [ ] `docs/docs/guides/custom-ai-providers.md` with three full recipes (vLLM, llama.cpp, Anthropic native)
- [ ] `examples/runnable/custom_provider_demo.ail`
- [ ] Update `docs/docs/guides/packages.md`
- [ ] Eval baseline: include one config-driven provider in the standard suite (gated on availability — runs locally, skipped in CI without endpoint)
- [ ] CHANGELOG entry
- [ ] Cross-reference from `m-external-consumer-dx.md` external-consumers guide
- [ ] Cross-reference from `m-ai-streaming-helper.md` (this is its prerequisite)

---

## Examples

### Example 1: Adding vLLM as a package (the headline win)

User runs a vLLM server locally serving `llama-3.1-70b`. They want it usable from AILANG with full budget/cap/trace integration.

**Before this milestone:** Fork `internal/ai/vllm/` from `internal/ai/openai/`, add to the if/else chain, recompile, ship a custom binary. ~200 LOC of Go, an upstream PR, or a permanent fork.

**After:**

```toml
# ai_vllm/ailang.toml
[package]
name = "sunholo/ai_vllm"
version = "0.1.0"
ailang = ">=0.15.0"

[[ai_provider]]
name = "vllm"
endpoint = "http://localhost:8000/v1/chat/completions"
request_shape = "openai_chat"
response_path = "$.choices[0].message.content"
auth = { type = "none" }
cost = { input_per_1m_usd = 0.0, output_per_1m_usd = 0.0 }
capabilities = { tool_calling = false, json_mode = true, streaming = true, vision = false, structured_outputs = false }
```

```bash
ailang install sunholo/ai_vllm@0.1.0
ailang run --caps AI --model vllm/llama-3.1-70b my_app.ail
```

In `my_app.ail`:

```ailang
import std/ai (call)

export func main() -> unit ! {AI, IO} {
  let response = call("Hello!");
  _io_println(response)
}
```

Total user-side delta: one TOML file. Budget, AI cap, trace spans all work identically to `--model openai/gpt-4o`.

### Example 2: llama.cpp local server with custom auth header

```toml
[[ai_provider]]
name = "llamacpp"
endpoint = "http://localhost:8080/completion"
request_shape = "simple_completion"
response_path = "$.content"
auth_headers = { X-Internal-Token = "${LLAMACPP_TOKEN}" }
cost = { per_call_usd = 0.0 }
```

### Example 3: Anthropic native (showing the milestone covers built-in patterns too)

A theoretical config-driven Anthropic that mirrors the built-in `anthropic` provider. Useful for testing the schema's expressiveness; in practice the built-in stays for production.

```toml
[[ai_provider]]
name = "anthropic-via-config"
endpoint = "https://api.anthropic.com/v1/messages"
request_shape = "anthropic_messages"
response_path = "$.content[0].text"
auth = { type = "x-api-key", env = "ANTHROPIC_API_KEY" }
auth_headers = { anthropic-version = "2023-06-01" }
cost = { input_per_1m_usd = 3.0, output_per_1m_usd = 15.0 }
capabilities = { tool_calling = true, json_mode = false, streaming = true, vision = true, structured_outputs = false }
```

---

## Success Criteria

- [ ] `make ci` passes
- [ ] `[[ai_provider]]` schema parses with full validation; golden tests cover all shapes
- [ ] Generic provider passes integration tests against mock OpenAI-shaped, Anthropic-shaped, and simple-completion servers
- [ ] Budget tracker shows cost for config-driven providers identical in shape to built-in providers
- [ ] AI cap gates config-driven providers (test: omit `--caps AI`, expect failure)
- [ ] Trace inspection: `ai.call` span present and identically structured for config-driven vs built-in
- [ ] Conflict detection: two packages with same provider name produces clear load-time error
- [ ] `examples/runnable/custom_provider_demo.ail` runs against echo-AI server end-to-end
- [ ] At least one external example package (e.g. `pkg/sunholo/ai_vllm`) demonstrates real-world flow
- [ ] `docs/docs/guides/custom-ai-providers.md` exists with three working recipes
- [ ] CHANGELOG.md v0.15.0 entry references this design doc and links to the new guide
- [ ] Eval baseline includes at least one config-driven provider exercise

---

## Testing Strategy

**Unit tests (Go):**
- TOML schema parsing for every documented field
- JSONPath validation (good + bad expressions)
- Auth-shape construction (headers correct for each type)
- Request-shape templating (each shape produces correct request body for representative inputs)
- Response/error extraction (each shape with golden response payloads)
- Cost calculation against mock token counts
- Conflict detection (duplicate provider names)
- Schema-version handling (current version accepted, future version rejected with clear message)

**Integration tests (Go + httptest.Server):**
- Full call flow: `Call(req)` → HTTP → response extraction → `Response{Text: ...}`
- Each request shape against a mock server returning shape-appropriate response
- Auth failure path returns `AIError{code: "AuthFailed", retryable: false}`
- 5xx with retry-after header → `AIError{retryable: true}`
- Budget exhaustion path
- Cap missing path
- Trace span emission verified via in-memory exporter

**Integration tests (AILANG end-to-end):**
- `examples/runnable/custom_provider_demo.ail` runs against httptest.Server-backed config provider
- Multi-package: install two packages each registering a provider; both work; conflict test with same name fails

**Manual testing:**
- Real vLLM endpoint (gated on env var)
- Real llama.cpp endpoint (gated on env var)
- Real Anthropic endpoint via config-driven path (compare trace shape to built-in)

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **JSONPath library choice** — agent picks (Go has 2-3 reasonable options; complexity vs feature tradeoff).
- **Configdriven package internal structure** — agent decides file split (`provider.go` / `shapes.go` / `auth.go` is a starting suggestion; reorganize freely).
- **Where in trace tier hierarchy to emit config-driven spans** — agent matches built-in provider tier; document if deviation chosen.
- **Allowlist for `${ENV_VAR}` patterns** — agent decides whether to restrict to `${[A-Z_]+}` or allow arbitrary `${...}` escapes.
- **Schema-version drift policy** — agent picks initial behavior (warn vs error vs reject); document.
- **Whether `auth_headers` interpolation supports nested fallbacks** (`${VAR:-default}`) — agent decides v1 scope.

---

## Non-Goals

- **Custom auth flows** (AWS SigV4, Azure AD, OAuth) — these stay Go-side. A future milestone may add a plugin point if demand emerges.
- **Non-HTTP transports** (gRPC, persistent WebSocket) — Go-side; out of scope.
- **Go plugin loading or WASM modules** — explicitly rejected. Cross-platform pain (Go's `plugin` package is Linux/macOS only with build-stamp version pinning), security review burden, build complexity. Config-driven covers the >90% case; custom providers stay built-in.
- **Streaming integration** — depends on this milestone but ships separately as [M-AI-STREAMING-HELPER](../v0_17_0/m-ai-streaming-helper.md). Schema includes streaming params; runtime support lands in the streaming milestone.
- **Provider-plugin marketplace / discovery UI** — separate v0.18+ design.
- **Tool-use / function-calling templating** — schema flags `capabilities.tools`, but the request-shape templating for tool definitions is deferred to a follow-on milestone.
- **Image input templating** — same; flagged in capabilities, schema work deferred.

---

## Timeline

**Day 1** (~6 hours): Schema + parsing + conflict detection
**Day 2** (~8 hours): Generic provider + `openai_chat` + `simple_completion` + bearer auth + budget/trace
**Day 3** (~6 hours): `anthropic_messages` + remaining auth shapes + routing
**Day 4** (~8 hours): Docs + example + eval baseline + cross-references + CHANGELOG

**Total: ~28 hours across 4 days**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Three request shapes are insufficient; long tail forces escape-hatch sprawl | High | Confirm with arni before implementation (Receive Path); add 4th shape only if evidence demands. `request_template` Go-template escape is the documented fallback for outliers |
| `auth_headers` env interpolation introduces injection risk | Med | Restrict `${...}` to `${[A-Z_][A-Z0-9_]*}` pattern; literal substitution only; document explicitly |
| Two packages declaring same provider name silently overwrite each other | Med | Hard error at load time, not warning. Documented |
| JSONPath errors at first call rather than load time degrade DX | Low | Lightweight syntactic validation at load; deep validation on first call with clear diagnostics |
| Cost declarations drift from real provider pricing | Low | Document that `cost` is best-effort; runtime never blocks on cost mismatch; expect package authors to update on price changes |
| Config-driven provider has subtle effect-system divergence (e.g. trace span attribute set differs) | Med | Snapshot test comparing trace span structure of `openai` (built-in) vs `anthropic-via-config` (config-driven) for an identical call shape |
| Schema v1 locks us out of features we want later (tool use, image input, batch) | Med | `schema_version = 1` field; v2 adds optional fields; v1 packages still work; documented migration path |
| Performance regression in package loading | Low | Provider config parsing is one-time at session start; benchmark in CI |

---

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_16_x/m-ai-openrouter-provider.md](../../implemented/v0_16_x/m-ai-openrouter-provider.md) — Latest example of adding a provider via the closed system. This milestone makes that work unnecessary for HTTP-shaped providers.
- [design_docs/implemented/v0_7_0/m-eval-ollama-local-models.md](../../implemented/v0_7_0/m-eval-ollama-local-models.md) (0.46 neural match) — Local-model integration; this milestone generalizes the pattern.
- [design_docs/implemented/v0_5_10/m-unified-ai-providers.md](../../implemented/v0_5_10/m-unified-ai-providers.md) — Established the `Provider` interface; config-driven provider implements it.

**Planned (companions):**
- [design_docs/planned/v0_13_0/m-arch1-ai-provider-base-class.md](../v0_13_0/m-arch1-ai-provider-base-class.md) (0.46 neural match) — **Complementary**: M-ARCH1 reduces Go-side provider boilerplate; this milestone obviates the need to write Go-side providers at all for the common case. M-ARCH1's BaseClient may serve as the foundation for the config-driven generic provider.
- [design_docs/planned/v0_17_0/m-ai-streaming-helper.md](../v0_17_0/m-ai-streaming-helper.md) — **Depends on this milestone**. Streaming params live in `[ai_provider.streaming]` block; runtime hook lands in the streaming milestone.
- [design_docs/planned/v0_17_0/m-external-consumer-dx.md](../v0_17_0/m-external-consumer-dx.md) — External-consumers guide must cross-reference `custom-ai-providers.md`.

---

## Receive Path

Already working (per [m-external-consumer-dx.md](../v0_17_0/m-external-consumer-dx.md) Receive Path section): `mcp.ailang.sunholo.com` → Pub/Sub → Firestore → SessionStart hook. External consumers can submit feedback via existing channels.

~~Open question to arni~~ — **No longer blocking.** Decision (Mark, 2026-05-04): proceed with the three v1 shapes (`openai_chat`, `anthropic_messages`, `simple_completion`). Schema versioning + `request_template` escape hatch absorb whatever arni's actual endpoint mix turns out to be; v2 can add named shapes if a clear pattern emerges from real usage. Iteration-over-survey.

When v0.15.0 ships, send arni a release note pointing at the new milestone and the example reference package — concrete artifacts solicit better feedback than scoping questions.

---

## Future Work

- **Schema v2: `mode` field on `[[ai_provider]]`** — once [M-EFFECT-REFINEMENT](../v1_0_0/m-effect-refinement.md) ports AI to parameterised effects, optional `mode = "fixed" | "routeable" | "replay-only"` declaration. v1 packages auto-inherit `mode=fixed` via the runtime mode dispatcher; v2 lets a routing-aggregator package declare `mode=routeable` explicitly.
- **M-AI-PROVIDER-PLUGIN-API (v0.18+)** — If config-driven shapes prove insufficient for a meaningful fraction of providers, design a real plugin extension point (likely WASM-based, not Go plugin). Out of scope here unless evidence demands it.
- **Schema v2** — tool-use templating, image input templating, batch endpoint templating, response-format templating beyond JSON. Add as `schema_version = 2` extending v1.
- **Provider marketplace / `ailang search --tag ai-provider`** — once the format is stable, build discovery on top.
- **Cost-aware routing** — given multiple providers serving the same model, route to cheapest available. Depends on cost declarations being present and accurate.
- **Provider health checks** — config-driven providers could declare a health endpoint; runtime warns on first call if unreachable.
- **Capability-based dispatch** — `call(prompt, requires=[Tools, JsonMode])` routes to any provider whose `capabilities` flags satisfy the requirement.

---

## References

- [Design Axioms](/docs/references/axioms)
- [M-AI-OPENROUTER design doc](../../implemented/v0_16_x/m-ai-openrouter-provider.md) — Latest hardcoded provider; motivates this milestone
- [arniwesth/ailang FORK.md](https://github.com/arniwesth/ailang/blob/motoko/FORK.md) — External-consumer evidence
- BlackMage feedback (May 2026) — User research driving scope; reproduced in Motivating Evidence above
- [internal/ai/](../../../internal/ai/) — Existing provider implementations
- [internal/pkg/manifest.go](../../../internal/pkg/manifest.go) — Package manifest schema (extended by this milestone)
- [cmd/ailang/exec.go:356-384](../../../cmd/ailang/exec.go#L356-L384) — The hardcoded provider chain this milestone opens

---

## Notes for the AI Implementer

Key constraints to keep in mind during implementation:

1. **Built-in providers stay built-in.** This milestone adds an *additional* registration path; it does not migrate `openai`/`anthropic`/`gemini`/`ollama`/`openrouter` to the config-driven system. Built-ins keep their per-provider Go packages because they need features (tool use, image input, OpenRouter routing logic) that aren't in the v1 schema.
2. **Effect machinery integration is the load-bearing requirement.** Budget, AI cap, trace spans must work identically. This is the failure mode that previously almost shipped (raw HTTP path bypassing AI cap). Snapshot-test trace structure against a built-in provider call.
3. **Hard error on conflicts.** Two packages declaring the same provider name = clear load-time error with both file locations, not silent override.
4. **Env-var interpolation is restricted.** `${[A-Z_][A-Z0-9_]*}` only. Literal substitution. No shell expansion, no command substitution, no nested fallbacks in v1.
5. **Schema version is required.** v1 packages declare `schema_version = 1`. Missing field = error. v2+ adds optional fields; v1 packages still work; runtime rejects packages declaring versions newer than it understands.
6. **Resist scope creep.** Tool-use templating, image input, batch — all flagged as Future Work. Don't expand the request shape templates beyond the three named shapes in v1.
7. **The example package is an acceptance criterion, not a nice-to-have.** Without an end-to-end demonstration that a third-party package can register a provider with zero Go code, this milestone has not shipped.

If implementation deviates from this doc, **update this doc first.** Decisions made in code without doc updates lose institutional memory.

---

**Document created**: 2026-05-04
**Last updated**: 2026-05-04

DESIGN_DOC_PATH: design_docs/planned/v0_15_0/m-ai-provider-config.md
