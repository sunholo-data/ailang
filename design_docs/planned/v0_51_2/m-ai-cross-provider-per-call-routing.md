# M-AI-CROSS-PROVIDER-PER-CALL-ROUTING: per-call cross-provider routing for step()/image calls + Result-returning image base64

**Status**: Planned
**Target**: v0.51.2
**Priority**: P1 (Medium) — unblocks the AI.5 workaround removal in stapledons-godot
**Estimated**: 4 days (~600 impl + ~500 test LOC)
**Dependencies**: [M-OPENROUTER-IMAGE-OUTPUT](../../implemented/v0_51_1/m-openrouter-image-output.md) (v0.51.1, landed — its per-call image `model` key and its same-provider `ModelResolver` are the base this doc generalizes)

**Created**: 2026-10-02

---

## Problem Statement

Stapledon's Voyage AI service (stapledons-godot) must route **text to OpenRouter (cheap) and
portraits/voice to Gemini, in one process** (Mark's "both providers" ruling). AILANG cannot do
this: the `--ai` flag binds exactly ONE handler to ONE provider for the whole process, and the
per-call model channel refuses to leave it.

Three findings, filed against the v0.51.0 binary (b99dd25) and re-verified against the v0.51.1
source tree (d980558a, this workspace) — see the Verification Log for the code citations:

1. **One handler per process; cross-provider per-call models are refused.**
   `cmd/ailang/ai_handlers.go:setupAIHandler` builds one `ai.NewHandler(client.Provider, model)`
   and stores it in one `effCtx.AI` (`internal/effects.AIContext`). The injected
   `makeModelResolver` (`cmd/ailang/ai_handlers.go:182-199`) resolves a per-call `step()` model
   via `models.yml` and returns `ai.CodeModelNotAllowed` when the model belongs to another
   provider. An id **unknown** to `models.yml` is passed through **unchanged to the bound
   provider** (`internal/modelreg/models.go:249-255` — `GetModel` is a plain map lookup; the
   resolver returns `model, nil` on lookup error). So `step("openrouter:z-ai/glm-5.3", …)` under
   `--ai gemini-2.5-flash-image` never reaches OpenRouter — the string is handed to the Gemini
   handler as a foreign model name, which fails as a confusing provider-side 400.

2. **Image calls are confined to the bound provider.** v0.51.1 (#1500/#1496, shipped) added a
   per-call `model` key to the image options JSON (`internal/ai/handler_image.go:generateImage`)
   and resolves it through the same `ModelResolver`
   (`internal/effects/ai_image.go:resolveImageOptions`), so a cross-provider image model fails
   with `ModelNotAllowed` exactly like `step()` (documented as Example 3 of
   M-OPENROUTER-IMAGE-OUTPUT). Note also: the per-call channel only exists where a resolver is
   attached — `setupAIHandlerDirect` (`cmd/ailang/ai_handlers.go:232-275`, the path an
   `models.yml`-unknown `--ai` model like `gemini-2.5-flash-image` takes, since no image/TTS
   model has a `models.yml` entry) attaches **no resolver at all**.

3. **`callImageBase64` has no Result variant: a provider failure aborts the program.**
   The op wraps any handler error in `E_AI_CALL_ERROR` and returns it as a raw Go error
   (`internal/effects/ai_image.go:206`), which the runtime treats as a host abort. Seven other
   std/ai functions already return `Result[_, AIError]` (`callResult`, `callJsonResult`,
   `callJsonSimpleResult`, `callSpeech`, `step`, `stepWithCache`, `stepWithStream`; plus
   `stepWithStreamRecorded`'s outcome field) — **zero of the four image functions do**. A
   portrait service cannot answer `provider_error` with a placeholder and keep serving.

**Current State (the measured cost, AI.5 workaround in the stapledons-godot tree):**
- `std/ai` is bound to Gemini for images and Gemini text; OpenRouter chat completions are
  hand-rolled over `std/net` with `Authorization: Bearer` headers; TTS went over `std/net` too
  (speech did not exist at v0.51.0; it shipped in v0.51.1 as #1495 — the workaround predates it).
- That parallel provider stack in user code bypasses everything the AI effect provides: typed
  `AIError`s, trace events (observatory spans), the AI capability budget, OpenRouter routing
  policy, stub/fixture-based testing, and `ClassifyError` retry semantics. It is a second
  provider implementation to maintain, and its failures are invisible to the observatory.

**Impact:**
- Any AILANG service that wants cheap text + specialized image/voice models in one process
  (stapledons-godot is the first; every "cheap model + image model" composition hits this).
- `callImageBase64`'s abort-on-failure hits any program that renders images as part of a larger
  flow — a single 429 from the image provider kills the whole process.
- v0.51.0-binary vs v0.51.1-source trap (measured this session): the findings were taken on the
  v0.51.0 binary while the tree is v0.51.1; two of them (#1495 speech, #1497
  `--ai-stub-fixtures` — `cmd/ailang/main_run.go:65`) are already fixed in source. This doc
  targets the v0.51.1 source and asks only for what remains: cross-provider routing and the
  image Result variant.

## Goals

**Primary Goal:** One AILANG process with one `--ai` binding can call different providers per
call when credentials for them are present, and image generation failures are catchable typed
errors — so the AI.5 std/net workaround can be deleted.

**Success Metrics:**
- `step("openrouter:vendor/model", …)` under `--ai gemini-2.5-flash-image` reaches OpenRouter
  when `OPENROUTER_API_KEY` is present (and returns a typed `Err(AIError)` naming the missing
  env var when it is not).
- The same routing works through the image options `{"model": …}` key, in both directions
  (Gemini images under an OpenRouter text binding; OpenRouter image models under a Gemini
  binding).
- `AI.callImageBase64Result(prompt, options) -> Result[string, AIError]` exists; a failing
  provider returns `Err`, the program continues (acceptance: stub-based example prints a
  fallback and exits 0).
- No new effect, no new capability, no new CLI flag: routing runs under the existing `AI`
  capability and one budget (acceptance: `make simplicity-audit` no new command/env routes).
- Every routed call names its provider in the trace event / routing metadata (acceptance:
  unit test asserts `LastRoutingMetadata` comes from the handler actually used).
- AI.5 workaround deletion is possible: the stapledons-godot service runs text+portraits+TTS
  through `std/ai` only (manual verification with real keys, both providers).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Routing lives in `AIContext` (`internal/effects`): a resolver *classifies* the model string into (provider, api_name), and a router picks the handler; NOT in a multi-handler composite inside `internal/ai` | The `ModelResolver` runs in `AIContext` **before** the handler sees the model, and the handler choice is the context's to make; a composite handler cannot avoid touching that contract anyway (V11) | agent | design | med |
| New `ModelRoute`/`RoutingModelResolver` types alongside the existing `ModelResolver`, not a signature change to it | Only `cmd/ailang` attaches resolvers today (V7); keeping the old type intact preserves stubs/tests/back-compat and makes the no-router behavior bit-for-bit today's | agent | design | med |
| Routed handlers are constructed lazily per provider, cached by `(provider, max_output_tokens bucket)`; "keys present" is decided at call time; a missing credential is a typed `Err(AIError{ProviderNotFound})` naming the env var, never a startup failure | Eager construction would refuse to start any process missing any provider key — a Gemini-only user must still run. Lazy + cached keeps routing deterministic within a run | agent | design | high |
| The routed handler's credential comes only from that provider's own lanes (env var / ADC / local); the `--ai-key-file` key of the bound provider is NEVER forwarded to a routed provider | A key-file or bound env var is a credential for ONE provider; forwarding it would send e.g. a ChatGPT OAuth token to OpenRouter (credential-leak / guaranteed 401) | human | design | high |
| The v0.51.1 cross-provider refusal (`ModelNotAllowed`, kept deliberately under D-8's one-key premise in M-OPENROUTER-IMAGE-OUTPUT) is REPLACED: a cross-provider per-call model routes when the target provider has a credential, otherwise fails `Err(AIError{ProviderNotFound})` | Mark's later "both providers" ruling supersedes the one-key premise; keeping the refusal makes the ask impossible | human | design | med |
| The per-call image model stays the options JSON `model` key; there is NO third `model?` parameter on `callImageBase64Result` | AILANG has no optional/default parameters (verified, V5) and `options.model` already exists (v0.51.1, #1500) — a third parameter would be a second spelling of one channel | agent | design | low |
| Result variants for the `! {AI}`-only image pair only (`callImageBase64Result`, `callImageBase64WithRefsResult`); the file-writing pair (`callImage`, `callImageWithRefs`) keeps `! {AI, FS}` and abort semantics | Their failure union includes mkdir/write failures, which are local deterministic FS errors, not provider errors — stuffing them into `AIError` would mislabel them | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Routing mechanism: classify in resolver, choose handler in `AIContext` (row 1)
- [x] Lazy construction + `(provider, budget)` cache + typed missing-key error (row 3)
- [x] Credential isolation: no cross-forwarding of the bound provider's key (row 4)
- [x] Supersede the v0.51.1 cross-provider refusal per the "both providers" ruling (row 5)
- [x] Image per-call channel remains `options.model`; no `model?` parameter (row 6)
- [x] Result-variant scope: the two base64 image functions only (row 7)

## Solution Design

### Overview

A per-call model string already exists on `step()`/`stepWithCache()`/`stepWithStream()` and in
the image options `model` key. This doc makes that string carry **provider intent**: an
explicit prefix (`openrouter:vendor/model`), a `models.yml` friendly name, or a
`GuessProvider`-recognizable name resolves to `(provider, api_name)`. The `AIContext` then
either stays on the bound handler (same provider / unknown name — today's behavior) or asks an
injected **provider router** for a handler of the target provider. The router, built in
`cmd/ailang`, reuses `factory.New` + `readyForCalls` + `ai.NewHandler` — the same construction
path as `--ai` — with per-provider credentials and a cache, so a routed OpenRouter call is
indistinguishable (wire, lane, telemetry) from a call on a process bound to OpenRouter.

Alongside the routing, `callImageBase64Result` and `callImageBase64WithRefsResult` bring the
image surface to the `Result[string, AIError]` shape every other modern std/ai function
already has.

### Architecture

**Components:**

1. **`internal/effects` — routing core.**

   ```go
   // ModelRoute is the classification of a per-call model string.
   type ModelRoute struct {
       APIName  string // what the target provider expects ("" only on error)
       Provider string // resolved provider key ("openrouter", "google", …); "" = unknown
   }
   // RoutingModelResolver classifies a per-call model. Supersedes ModelResolver
   // when attached; ModelResolver keeps its exact current semantics when it is
   // the only one attached (no routing, cross-provider refusal stays in the
   // injected resolver's hands).
   type RoutingModelResolver func(model string) (ModelRoute, error)
   // ProviderRouter returns a handler for a provider key ("openrouter", "google",
   // "ollama", a config-driven name). It may construct + cache lazily.
   type ProviderRouter func(provider string) (AIHandler, error)
   ```

   - `WithRoutingResolver(RoutingModelResolver)` and `WithProviderRouter(ProviderRouter)`
     on `AIContext`. When a routing resolver is attached it takes precedence over a plain
     `ModelResolver`.
   - `resolveTarget(model string) (handler AIHandler, apiName string, err error)`:
     `model == ""` → bound handler, bound default; resolver error → passthrough (typed);
     `route.Provider == ""` → bound handler with `route.APIName` (unknown-name passthrough,
     preserved); `route.Provider != ""` → `router(route.Provider)` → its handler; router
     error (missing credential) → typed `*ai.AIError` passthrough.
   - `Step`/`StepWithCache`/`StepWithStream` (`internal/effects/ai.go:203-238`) switch from
     `resolveModel` to `resolveTarget` and dispatch to the returned handler.
   - `resolveImageOptions` (`internal/effects/ai_image.go:38`) generalizes to
     `resolveImageTarget(options) (AIHandler, options string, error)`: same classification on
     the `model` key, options rewritten via `ai.SetImageOptionsModel` to the api_name for the
     target handler. All four image entry points plus the two new Result ops use it.
   - `LastRoutingMetadata` (`internal/effects/ai.go:143-149`) currently reads `c.handler` —
     the **bound** handler only. `AIContext` records the handler it last dispatched to
     (`c.lastDispatched`) and `LastRoutingMetadata` reads it, so a routed OpenRouter call's
     routing metadata/cost lands on the right provider in the trace event. Call/CallJson
     (no per-call model) set it to the bound handler, preserving today's values.
   - Budget and capability enforcement is unchanged: it is charged per effect op at the
     `EffContext` level (`internal/effects/context.go:393`), provider-agnostic — one `AI`
     capability, one budget, spanning providers.

2. **`cmd/ailang` — the resolver and the router.**

   - `makeModelResolver(boundProvider)` becomes `makeRoutingModelResolver()` —
     classification, in order:
     1. Explicit prefixes: `openrouter:` / `ollama:` / `ollama/` / `chatgpt/` → that provider,
        remainder as api_name (mirrors `setupAIHandlerDirect`'s `TrimPrefix` and
        `ai.GuessProvider`'s precedence, `internal/ai/config.go:51-100`).
     2. `models.yml` friendly name (`ResolveModelName`) → its provider + api_name.
     3. `ai.GuessProvider(model)` → provider + model as-is (handles `vendor/model` →
        OpenRouter, bare `claude-*`/`gpt-*`/`gemini-*`).
     4. Unknown (`""`) → `{Provider: "", APIName: model}` → bound passthrough, exactly today.
   - Attached on **both** paths: `setupAIHandlerFromConfig` (replacing the plain
     `WithModelResolver`, `cmd/ailang/ai_handlers.go:140`) and `setupAIHandlerDirect`
     (new — the direct path has no resolver today, so per-call names under a direct binding
     never resolved at all).
   - `makeProviderRouter(...)` closure: `map[routeKey]*ai.Handler` with
     `routeKey = (provider, max_output_tokens bucket)` where the bucket is the per-call
     model's declared `models.yml` `max_output_tokens` (0 = handler default 4096 —
     `declaredMaxOutputTokens` already exists for the direct path). Construction:
     `factory.New(provider, WithConfigDriven(LookupConfigDrivenProvider), …)` — **no**
     `WithAPIKey` from `--ai-key-file`, **no** bound model's `env_var` unless the routed model
     itself came from `models.yml` (then `WithAPIKeyEnv(model.EnvVar)`); then `readyForCalls`
     (lane announcement on stderr + ollama connection probe) and
     `ai.NewHandler(client.Provider, "", opts...)` carrying the same routing policy /
     attribution handler options as the bound handler. A `factory` error (missing key)
     becomes `ai.NewAIError(CodeProviderNotFound, …, false)` carrying the factory's message
     (which names the required env var).
   - Stub mode (`--ai-stub` / `--ai-stub-fixtures`): attach the routing resolver plus a router
     that maps **every** provider to the stub handler. The stub ignores the model, so
     behavior is unchanged for single-provider programs, while stub-driven E2E examples and
     fixture tests can exercise routing deterministically (V13's fixture discipline).

3. **Result-returning image ops.**

   - `internal/effects/ai_image.go`: register `AI.callImageBase64Result` and
     `AI.callImageBase64WithRefsResult`, mirroring `aiCallResult`
     (`internal/effects/ai_step.go:62-101`): on error `classifyOpError` →
     `makeAIErrorResultRecord`; on success `makeOkStringResult`; trace events
     `callImageBase64Result` / `callImageBase64WithRefsResult` with the `err:%s` prefix
     pattern; routing metadata from the handler actually used.
   - `internal/builtins/ai_image.go`: `_ai_call_image_base64_result` (2 args) and
     `_ai_call_image_base64_with_refs_result` (3 args), typed
     `(…) -> Result[string, AIError] ! {AI}` via `T.App("Result", T.String(),
     aiErrorRecordType(T))` — the exact shape `_ai_call_result` uses
     (`internal/builtins/ai_step.go:145-150`).
   - `std/ai.ail`: two new exports (see Examples).

4. **Docs / examples.** `examples/runnable/ai_cross_provider.ail` (stub-runnable, both
   directions), `examples/runnable/ai_image_base64_result.ail` (Err path serves a fallback),
   std/ai reference page, `docs/LIMITATIONS.md` if it lists the one-provider constraint.

**What does NOT change:**
- No provider wire protocol, no `AIHandler` interface signature (routed dispatch is inside
  `AIContext`; every existing handler — stub, real, serve-api — compiles untouched).
- `Call`/`CallJson`/`CallJsonSimple`/`callSpeech` behavior (no per-call model on those
  surfaces; speech options-model routing is a follow-up).
- The `ModelResolver` type and its contract when it is the only resolver attached.
- Capability model: one `AI` capability; budgets charged per op as today.
- `serve-api` (`cmd/ailang/serve_api.go:115`) flows through `setupAIHandler` and therefore
  gets routing with no further change.

### Implementation Plan

**Phase 1: routing core** (~1.5 days)
- [ ] `internal/effects`: `ModelRoute`, `RoutingModelResolver`, `ProviderRouter`,
      `WithRoutingResolver`, `WithProviderRouter`, `resolveTarget`, `resolveImageTarget`,
      `lastDispatched` metadata threading; step ops + image entry points dispatch through them
- [ ] `cmd/ailang`: `makeRoutingModelResolver`, `makeProviderRouter` (lazy `(provider,
      budget)` cache, credential isolation, `readyForCalls`), attach on both setup paths and
      in stub mode
- [ ] Unit tests for: same-provider unchanged, cross-provider routed, cross-provider without
      credential (typed `ProviderNotFound` naming env var), unknown-name passthrough,
      `LastRoutingMetadata` from the routed handler, key-file isolation, direct-path
      resolution

**Phase 2: Result image ops** (~0.5 day)
- [ ] `internal/effects/ai_image.go`: the two Result ops
- [ ] `internal/builtins/ai_image.go`: the two builtins
- [ ] `std/ai.ail`: the two exports + doc comments
- [ ] Tests: Ok path (stub PNG), Err path (failing handler → `Err(AIError)`), type shape via
      `ailang check` of a consumer module

**Phase 3: examples, docs, guardrails** (~1 day)
- [ ] `examples/runnable/ai_cross_provider.ail`, `examples/runnable/ai_image_base64_result.ail`
      (both run under `--ai-stub`)
- [ ] std/ai reference docs, `changelogs/v0.32-current.md` entry (behavior change: cross-provider
      per-call refusal → routing)
- [ ] `make test`, `make fmt`, `make check-boundaries` (routing code crosses
      cmd/ai ↔ effects; boundaries must stay clean)

### Files to Modify/Create

**New files:**
- `examples/runnable/ai_cross_provider.ail` — stub-runnable both-direction routing (~40 LOC)
- `examples/runnable/ai_image_base64_result.ail` — Err-path fallback (~30 LOC)

**Modified files:**
- `internal/effects/ai.go` — routing types + `resolveTarget` + step dispatch + metadata (~120 LOC)
- `internal/effects/ai_image.go` — `resolveImageTarget` + two Result ops (~110 LOC)
- `internal/effects/ai_test.go` / `ai_image_test.go` — routing tests (~200 LOC)
- `cmd/ailang/ai_handlers.go` — routing resolver + provider router + attachments (~150 LOC)
- `cmd/ailang/ai_handlers_test.go` — router/cache/isolation tests (~120 LOC)
- `internal/builtins/ai_image.go` — two Result builtins (~120 LOC)
- `std/ai.ail` — two exports + comments (~30 LOC)
- `changelogs/v0.32-current.md`, std/ai reference docs

## Examples

### Example 1: Voyage's service — text cheap on OpenRouter, portraits on Gemini, one process

**Before (AI.5 workaround, in tree today):**
```ailang
-- std/ai bound to the image model (--ai gemini-2.5-flash-image) for portraits;
-- OpenRouter text hand-rolled over std/net with Authorization: Bearer headers,
-- outside the AI effect: no typed errors, no trace, no budget, no stubs.
```

**After:**
```ailang
-- run: ailang run --caps AI,IO --ai openrouter/z-ai/glm-5.3 --entry main
import std/ai (step, callImageBase64Result, AIError, Message, ToolSchema)
import std/result (Ok)

export func main() -> () ! {AI, IO} {
  -- text: the bound handler, per-call "" = bound default
  let answer = step("", [{role: "user", content: "Roll an NPC", tool_calls: [], tool_call_id: "", images: []}], [])

  -- portrait: per-call model routes to the Gemini handler (GOOGLE_API_KEY/ADC present)
  let portrait = callImageBase64Result("portrait of a gruff dwarf", "{\"model\": \"gemini-2.5-flash-image\"}")
  -- both directions work: under --ai gemini-2.5-flash-image, the text call is
  -- step("openrouter:z-ai/glm-5.3", messages, [])
}
```

### Example 2: provider failure degrades instead of aborting

```ailang
import std/ai (callImageBase64Result, AIError)
import std/result (Result, Ok, Err)
import std/io (println)

export func main() -> () ! {AI, IO} {
  match callImageBase64Result("portrait", "{}") {
    Ok(json) => println("serving ${json}"),
    Err(e)  => println("image failed (${e.code}): ${e.message} — serving silhouette.svg")
  }
}
-- Before: the same provider failure printed
--   Error: E_AI_CALL_ERROR: [RateLimit] HTTP 429 …
-- and the process exited non-zero. After: exit 0, caller decides.
```

### Example 3: a cross-provider model with no key fails loud and typed

```ailang
-- under --ai gemini-2.5-flash-image with no OPENROUTER_API_KEY:
step("openrouter:z-ai/glm-5.3", messages, [])
-- Err(AIError{code: "ProviderNotFound",
--   message: "per-call model \"openrouter:z-ai/glm-5.3\" routes to provider \"openrouter\":
--   OPENROUTER_API_KEY environment variable required …",
--   retryable: false})
-- (v0.51.1: CodeModelNotAllowed "use --ai … to switch providers"; v0.51.0 binary:
-- the string was passed to the Gemini handler and failed as a provider-side 400.)
```

## Success Criteria

- [ ] `step("openrouter:z-ai/glm-5.3", …)` under a Gemini binding reaches the OpenRouter
      handler when the key is present (unit test with a recording router; manual E2E with real keys)
- [ ] Image options `{"model": "gemini-2.5-flash-image"}` under an OpenRouter binding reaches
      the Gemini handler (unit test; manual E2E)
- [ ] Missing credential for the routed provider returns `Err(AIError{ProviderNotFound})`
      naming the env var; the process does not abort and does not fail at startup (unit test)
- [ ] `AI.callImageBase64Result` / `AI.callImageBase64WithRefsResult` return
      `Ok(json-string)` / `Err(AIError)`; a failing handler yields exit 0 in the example (tests)
- [ ] `LastRoutingMetadata` after a routed call reflects the routed handler (unit test)
- [ ] `--ai-key-file` key never appears on a routed provider's construction (unit test asserts
      the factory options of the router)
- [ ] All existing ModelResolver/image/resolver tests pass unchanged
      (`TestAIContext_ModelResolver_*`, `ai_image_test.go` friendlyResolver suite)
- [ ] `examples/runnable/ai_*.ail` all still run under `--ai-stub` (fixtures list in Conflict Surface)
- [ ] `make test`, `make fmt`, `make check-boundaries` pass
- [ ] Documentation updated (std/ai reference, this doc's changelog entry)

## Conflict Surface

This change touches `internal/effects` (AIContext) — required analysis:

**1. Positions extended:**
- The per-call `model` argument (arg 0) of `step` / `stepWithCache` / `stepWithStream`.
- The options JSON `model` key of `callImage` / `callImageBase64` / `callImageWithRefs` /
  `callImageBase64WithRefs` (+ the two new Result variants).
- The resolver attachment surface of `AIContext` (`WithModelResolver` → + `WithRoutingResolver`
  / `WithProviderRouter`).

**2. Other valid constructs already living in those positions:**

| Model string | Resolves to | v0.51.1 behavior | After |
|---|---|---|---|
| `""` | bound default | bound handler | bound handler (unchanged; never routed) |
| `gemini-2.5-flash` (unknown to models.yml) | `{Provider: google, APIName: …}` | passthrough to bound | Google binding: unchanged; OpenRouter binding: **routes to Gemini if key** |
| friendly `models.yml` name, same provider | api_name | api_name rewrite | unchanged |
| friendly `models.yml` name, other provider | — | `ModelNotAllowed` | **routes to that provider if key**, else typed `ProviderNotFound` |
| `openrouter:vendor/model` | openrouter + remainder | passthrough to bound (foreign name) | **routes to OpenRouter** |
| `vendor/model` (known vendor prefix) | openrouter | passthrough to bound | routes to OpenRouter |
| `ollama:model` / `ollama/model` | ollama | passthrough / bound | routes to ollama (local lane, no key) |
| `chatgpt/…` | chatgpt (OAuth lane) | passthrough to bound | routes if the codex login exists |
| anything else | `{Provider: "", APIName: model}` | passthrough to bound | passthrough to bound (unchanged) |

**3. Disambiguation:** one deterministic classification order (explicit prefix → `models.yml`
→ `GuessProvider` → unknown), the same precedence `ai.GuessProvider` already uses for
`--ai` names (`internal/ai/config.go:51-100`: ollama before vendor-slash, chatgpt before
vendor-slash, `openrouter:` explicit, vendor-slash → OpenRouter before bare prefixes). The
same string always classifies identically in one binary, and the classification (provider +
api_name) is banked in the trace event.

**4. Programs that MUST still work post-change (regression fixtures — all exist, V14):**
- `internal/effects/ai_test.go`: `TestAIContext_ModelResolver_ResolvesFriendlyName`,
  `TestAIContext_ModelResolver_UnknownPassesThrough`,
  `TestAIContext_ModelResolver_EmptyModelSkipsResolver`,
  `TestAIContext_ModelResolver_NilResolverPassthrough` (the plain-resolver no-router path
  keeps byte-identical semantics)
- `internal/effects/ai_image_test.go` (`friendlyResolver` suite) — options without a `model`
  key never touch the resolver
- `examples/runnable/ai_tool_loop.ail`, `ai_image_generation.ail`, `ai_call_result.ail`,
  `ai_call_speech.ail`, `ai_caching.ail` — run unchanged under `--ai-stub`
- `--ai-stub-fixtures` replay behavior unchanged

**5. Deliberate changes (intentional incompatibilities):**
- Cross-provider per-call models no longer fail `ModelNotAllowed` when the target provider
  has a credential — the v0.51.1 refusal (kept deliberately under D-8's one-key premise,
  M-OPENROUTER-IMAGE-OUTPUT High-Impact row) is superseded by the "both providers" ruling.
  When no credential exists the error changes shape: `ModelNotAllowed` →
  `ProviderNotFound` (naming the env var) — message text changes, code changes, retryable
  stays false.
- The direct `--ai` path (no `models.yml` entry) gains a resolver it never had: per-call
  friendly names now resolve there too. A program that (improbably) relied on a friendly name
  passing through raw to the bound provider will see it resolved instead — that was never
  documented behavior (the resolver contract already treats friendly names as resolvable).

## Testing Strategy

**Unit tests:**
- `internal/effects`: routing resolver classification table (every Conflict Surface row);
  routed dispatch to a recording handler; router error → typed passthrough; plain-resolver
  no-router semantics unchanged; `LastRoutingMetadata` from routed handler; image options
  routing + `SetImageOptionsModel` rewrite on the target handler; Result image ops Ok/Err.
- `cmd/ailang`: router cache hits (same provider+bucket constructs once, `(provider, budget)`
  keying); credential isolation (no key-file key, no foreign env_var on routed construction);
  missing-key error carries the factory's env-var message; stub-mode router maps all
  providers to the stub; both setup paths attach the routing resolver.

**Integration tests:**
- `examples/runnable/ai_cross_provider.ail` and `ai_image_base64_result.ail` under
  `--ai-stub` (deterministic, no network) — and under `--ai-stub-fixtures` if a fixture pair
  is authored.

**Manual testing (real keys, both providers):**
- `--ai openrouter/z-ai/glm-5.3` + `step("", …)` (bound) + `callImageBase64Result(…,
  {"model": "gemini-2.5-flash-image"})` (routed) — check the observatory span attributes carry
  the Gemini provider for the image call and OpenRouter for the text call.
- Reverse direction: `--ai gemini-2.5-flash-image` + `step("openrouter:z-ai/glm-5.3", …)`.
- With `OPENROUTER_API_KEY` unset: the routed call returns `Err`, process exits 0.

## Deferred Decisions

- Exact stderr wording of the routed-construction lane announcement (`readyForCalls` reused
  vs a quieter variant naming "routed per-call") — agent may choose
- Cache key struct shape (`struct{provider string; budget int}` vs formatted string) — agent
  may choose
- Whether to expose the provider router through the embed/serveapi host API for hosts that
  construct handlers themselves — agent may choose (none of the current hosts need it; `serve-api`
  flows through `setupAIHandler`)
- Test and example file names — agent may choose
- Whether the stub-mode router also accepts a `--ai-stub-fixtures` per-provider fixture
  namespace — deferred; single-namespace fixtures first

## Non-Goals

**Not attempted in this feature:**
- Automatic failover / fallback across providers — same-weights retry is
  [m-provider-failover](../m-provider-failover.md); agent-side model walking is
  [m-secondary-model-fallback](../m-secondary-model-fallback.md). This doc's routing is
  **caller-chosen per call**; no automatic retries are added.
- Speech options-model routing (`callSpeech`'s `model` key) — speech is Gemini-only today; the
  same resolver generalization applies when a second TTS provider exists.
- Result variants for the file-writing image pair — their failure union includes local FS
  errors; see High-Impact row 7.
- Multiple `--ai` flags or per-provider `--caps` — one binding, one capability, one budget.
- OpenRouter's `/api/v1/images` dedicated API — chat-completions modalities cover the ask
  (M-OPENROUTER-IMAGE-OUTPUT).
- Any wire/protocol change to providers.

## Timeline

**Week 1** (4 days):
- Days 1-2: Phase 1 (routing core + cmd resolver/router + tests)
- Day 3: Phase 2 (Result image ops + std + tests)
- Day 4: Phase 3 (examples, docs, changelog, full `make test`, boundaries, manual E2E)

**Total: ~4 days** (2x the naive estimate, per convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| A program can now reach any provider whose credential is present in the process (wider than the one `--ai` provider) | Med | The operator's installed credentials define the reachable set (each required deliberate env/lane configuration); every routed construction announces its lane on stderr and every routed call banks provider + model in the trace; unset keys you don't want reachable; `--ai-key-file` is never forwarded |
| Behavior change: code relying on `ModelNotAllowed` to prevent cross-provider spends | Med | The refusal remains as a typed `Err` when no credential is present; the changelog entry names the supersession (D-8 one-key premise → both-providers ruling); trace records make accidental crossings auditable |
| Routed-handler budget pinning: first model to touch a provider fixes its `maxTokens` for later calls | Low | Cache keyed by `(provider, budget bucket)` — two models with different declared budgets get two handlers; deterministic per call sequence |
| Routing metadata mis-attribution (routed call's cost banked on the bound provider) | Med | `lastDispatched` threading + a unit test asserting `LastRoutingMetadata` reads the handler actually used |
| Stale binary trap: v0.51.0 `ailang` in PATH vs v0.51.1 std (measured this session — `ailang --version` reports b99dd25 while `std/VERSION` is v0.51.1) | Low | Rebuild before testing (`make quick-install`); the doc's claims cite source, and `ailang check` transcripts in the Verification Log note the binary version |
| Mid-program lazy construction noise (stderr lane announcements appear during the run, not at startup) | Low | Accepted (loud > silent); wording is a Deferred Decision |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Classification is a pure function of the model string (prefix → models.yml → GuessProvider → passthrough); handler construction is memoized, so the same call sequence takes the same routes; the route is banked per call |
| A2: Replayability | +1 | Every routed call records provider + api_name + routing metadata in its trace event; a replay sees exactly which provider answered |
| A3: Effect Legibility | +1 | No new effect; routing rides the existing `AI` effect and its per-op traces |
| A4: Explicit Authority | 0 | Reachable providers are those the operator explicitly installed credentials for, and each crossing is explicit in the program's model literal, per-call, and recorded — but no *new* per-provider declaration is required, so this is not a strict improvement in explicitness either; honest 0 |
| A5: Bounded Verification | +1 | No language semantics; `ailang check` covers the new std signatures; routing is unit-testable with a recording router (no network) |
| A6: Safe Concurrency | 0 | `AIContext` remains single-threaded per evaluation (documented on the type today); the router cache is append-only, same-thread |
| A7: Machines First | +1 | Removes an entire hand-rolled provider stack (AI.5) from a consumer program; typed `Result` errors instead of aborts and string parsing |
| A8: Minimal Syntax | +1 | Zero new syntax; two std functions follow the established `…Result` naming; no new flags |
| A9: Cost Visibility | +1 | One AI capability, one budget across providers; routed construction announces its billing lane; routed calls bank provider-true cost metadata (was: invisible std/net spend) |
| A10: Composability | +1 | Reuses `factory.New`, `readyForCalls`, `ai.NewHandler`, `SetImageOptionsModel`, `classifyOpError`, `makeAIErrorResultRecord` — no parallel machinery |
| A11: Structured Failure | +1 | Missing credential, refusal-when-unrouted, and provider failures all become typed `AIError`s with codes; image failures become `Err(AIError)` instead of host aborts |
| A12: System Boundary | +1 | Each provider crossing is per-call, explicit in the model string, announced, and banked — versus today's hidden workaround where the boundary is crossed in user std/net code with no record |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism; classification and memoized routes are deterministic within a run
- [x] A3 (Effects): no hidden side effects; construction announcements go to stderr, calls to declared effects
- [x] A4 (Authority): no credential is used that the operator did not explicitly configure; the `--ai-key-file` key is never forwarded to another provider
- [x] A7 (Machines First): the ask comes from a machine-facing service (typed errors, no aborts, traceable spend)

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |

## Verification Log

Every load-bearing claim above was checked against the v0.51.1 source tree (d980558a) or
with the installed `ailang` binary (v0.51.0, b99dd25-dirty — noted per row where the version
matters).

| # | Claim | Check | Result |
|---|---|---|---|
| V1 | Cross-provider per-call models are refused (`CodeModelNotAllowed`); the error tells the user to rebind `--ai` | Read `cmd/ailang/ai_handlers.go:182-199` (`makeModelResolver`) | Confirmed |
| V2 | Ids unknown to `models.yml` pass through to the bound provider unchanged | Read `internal/modelreg/models.go:249-255` (`GetModel` is a plain `c.Models[name]` lookup; the resolver returns `model, nil` on lookup error) + `internal/effects/ai.go:170-177` | Confirmed — so `step("openrouter:…")` under a Gemini binding hands a foreign name to Gemini |
| V3 | The v0.51.0-binary findings #1495 (speech) and #1497 (`--ai-stub-fixtures`) are already fixed in v0.51.1 source | `std/ai.ail:176` (`callSpeech`), `cmd/ailang/main_run.go:65` (`--ai-stub-fixtures` flag), `changelogs/v0.32-current.md` v0.51.1 entries; the installed binary (`ailang --version` → v0.51.0) lacks them — the coordinator's "run --help lists only -ai-stub" was measured on the stale binary | Confirmed (source ahead of binary) |
| V4 | Image per-call `model` key exists in v0.51.1 but cross-provider fails `ModelNotAllowed` | Read `internal/ai/handler_image.go` (`generateImage`, `ParseImageOptions` Model field), `internal/effects/ai_image.go:28-55` (`resolveImageOptions`), changelog v0.51.1 "#1500, #1496" | Confirmed |
| V5 | AILANG has no optional/default function parameters (grounds rejecting the ask's `model?` third parameter) | `ailang check` on `func f(x: int, y: int = 3) -> int = x + y` → `PAR_UNEXPECTED_TOKEN …:2:23: expected next token to be ), got = instead` (v0.51.0 binary; parse surface unchanged in v0.51.1) | Confirmed |
| V6 | The proposed std surface shape compiles: `Result[string, AIError] ! {AI}` with `match` Ok/Err + record field access | `ailang check` on a local module defining `type AIError = {code, message, retryable}`, a `! {AI}` function returning `Result[string, AIError]`, and a `match` with `${e.code}` interpolation → `✓ No errors found!` | Confirmed |
| V7 | Only `cmd/ailang` attaches a `ModelResolver` (changing the contract's consumers is cheap and safe) | `grep -rn "WithModelResolver" --include="*.go"` → one non-test hit, `cmd/ailang/ai_handlers.go:140` | Confirmed |
| V8 | No Result variant of any image function exists today | `grep -rn "callImageBase64Result\|callImageResult\|_ai_call_image.*result"` over `internal/ cmd/ std/` → empty | Confirmed |
| V9 | `callImageBase64` aborts the program on provider failure (no Result path) | Read `internal/effects/ai_image.go:193-206` (`aiCallImageBase64` wraps with `errAICallFmt` = `E_AI_CALL_ERROR: %w`, `internal/effects/ai.go:108-110`) | Confirmed |
| V10 | The resolver is attached only on the config path — the direct path (an `--ai` model unknown to `models.yml`, e.g. every image/TTS model: no `gemini-*image`/`tts` key exists in `models.yml`) has no resolver at all | Read `cmd/ailang/ai_handlers.go:140` (config path) vs `:232-275` (`setupAIHandlerDirect`, no `WithModelResolver`); `grep "gemini-2-5-flash-image\|tts" internal/modelreg/models.yml` → empty | Confirmed |
| V11 | A multi-handler composite in `internal/ai` cannot avoid touching the `AIContext`/resolver contract: the resolver runs before the handler sees the model (`internal/effects/ai.go:170-177, 203-238`) and the refusal lives in the resolver, not the handler | Read the dispatch chain `aiStep` → `ctx.AI.Step` → `resolveModel` → `handler.Step` | Confirmed |
| V12 | The typed-error vocabulary needed (`ProviderNotFound`, `AuthFailed`, `ModelNotAllowed`, `SchemaValidation`, `RateLimit`) already exists — no new error code is introduced | Read `internal/ai/errors.go:23-43` | Confirmed |
| V13 | Regression fixtures exist and assert what the doc says they assert | `internal/effects/ai_test.go:143-201` (five `TestAIContext_ModelResolver_*` functions), `internal/effects/ai_image_test.go:58-134` (`friendlyResolver` suite), `ls examples/runnable/ai_*.ail` (12 files, incl. `ai_tool_loop.ail`, `ai_image_generation.ail`) | Confirmed |
| V14 | Budget/capability enforcement is per-op and provider-agnostic (one budget across routed calls) | Read `internal/effects/context.go:393-408` (`RequireCapWithBudget` charges per logical op at the `EffContext` level, before any handler is chosen) | Confirmed |
| V15 | `LastRoutingMetadata` reads only the bound handler today (routed calls would mis-attribute cost) | Read `internal/effects/ai.go:143-149` | Confirmed |
| V16 | Credential lanes: `factory.New` resolves per-provider credentials and fails loudly naming the env var; `AILANG_AI_NO_ADC=1` is honored process-wide for routed Google construction | Read `internal/ai/factory/factory.go:83-86, 187-226` (`WithNoADC`, Google lane, `config.AINoADC()`), `:96-108` (`New`) | Confirmed |
| V17 | No planned/implemented doc already covers per-call cross-provider routing (duplicate gate) | `ailang docs search` (SimHash + neural→fallback) on "cross provider routing", "per-call model resolver", "multiple providers one process" → no topical match; nearest real neighbors are `m-provider-failover` (same-weights failover), `m-secondary-model-fallback` (executor model walking), `m-openrouter-image-output` (per-call model within the bound provider) — all cited in Related Documents as distinct | Confirmed (no duplicate) |
| V18 | `go test` baseline for affected packages | **Not run** — this workspace has no Go toolchain (`go: command not found`, no make/gcc). All code-level claims rest on source reads with file:line citations above; the sprint executor must run `make test` first as its baseline gate | Recorded gap, owned by sprint |

## Quorum (2026-10-02, attempted)

`ailang design-quorum` was run with the off-Anthropic pool
(`gpt6-1-sol`, `gemini-3-1-pro`, `oc-glm-5-3`) — **all three seats were ABSENT** for
environment reasons (`gpt6-1-sol`: auth; `gemini-3-1-pro`: Vertex 403 on project `ailang-dev` —
this workspace's ADC is not the quorum project; `oc-glm-5-3`: unreachable). The quorum degraded
to controller-only (recorded by name, never a silent pass — artifact:
`.ailang/state/mission-quorum/m-ai-cross-provider-per-call-routing-2026-10-02T19-24-31Z.json`,
mission-log block appended). Controller verdict: **pass** on the strength of the 18-row
Verification Log (source-cited premises + `ailang check` transcripts). **A re-quorum with live
reviewers is recommended before sprint execution** — this doc has not had an independent
off-Anthropic review.

## Related Documents

**Implemented (may inform design):**
- [M-OPENROUTER-IMAGE-OUTPUT](../../implemented/v0_51_1/m-openrouter-image-output.md) —
  per-call image `model` key + the same-provider resolver this doc generalizes; its D-2
  refusal is the decision superseded here
- [M-AI-IMAGE](../../implemented/v0_10_0/m-ai-image-generation.md) — the `callImage*` surface
- [M-AI-TOOL-LOOP](../../implemented/v0_17_0/m-ai-tool-loop.md) (cited via
  `internal/effects/ai_step.go` header) — the `step()` surface and `Result[StepResult, AIError]`
- [M-WASM-AI-STEP-BYO-KEY](../../implemented/v0_19_0/m-wasm-ai-step-byo-key.md) — host-side
  handler construction, a different lane of the same problem space

**Planned (check for overlap):**
- [m-provider-failover](../m-provider-failover.md) — same-weights transient failover; this doc
  is caller-chosen routing, no retry logic
- [m-secondary-model-fallback](../m-secondary-model-fallback.md) — executor model walking;
  unrelated surface, same "more than one model" family

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [std/ai source](/std/ai.ail) - the surface being extended
- [m-openrouter-image-output changelog entry](/changelogs/v0.32-current.md) - v0.51.1, the
  #1500/#1496 image model key
- Mark's "both providers" ruling (stapledons-godot AI service, recorded in the task brief;
  supersedes the D-8 one-key premise of 2026-10-02)

## Future Work

- Speech options-model routing through the same resolver (when a second TTS provider exists).
- Result variants for the file-writing image pair, once an honest error union for local FS
  failures is designed (or an `Err` code for `WriteFailed` is added).
- `--ai-stub-fixtures` per-provider fixture namespaces for multi-provider replay.
- Exposing the provider router to embed/serveapi hosts that construct their own handlers.

---

**Document created**: 2026-10-02
**Last updated**: 2026-10-02
