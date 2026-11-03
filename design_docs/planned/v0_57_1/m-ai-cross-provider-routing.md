# AI Cross-Provider Per-Call Routing + `callImageBase64Result`

**Status**: Planned
**Target**: v0.57.1
**Priority**: P1 (Medium-High) — unblocks a real consumer (stapledons-godot) that is currently bypassing `std/ai` with raw `std/net` HTTP
**Estimated**: 4–5 days (router + alternates: ~2.5 days; Result image op: ~1 day; tests/docs/example: ~1.5 days)
**Dependencies**: None — builds on surfaces that already shipped (#1496/#1500 per-call image models, #1495 `callSpeech`, #1497 `--ai-stub-fixtures`; see Verification Log V8)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Routing is a pure function of (model string, models.yml, credential presence) — resolved once at setup, no per-call ambient re-probing |
| A2: Replayability | 0 | No change to trace shape; `RecordAIEffect` already records model + routing metadata per call |
| A3: Effect Legibility | 0 | No new effect rows; everything stays inside the existing `! {AI}` / `! {AI, FS}` rows |
| A4: Explicit Authority | +1 | The per-call model string names the destination provider explicitly; alternates exist only where a credential is present, and the startup line announces them |
| A5: Bounded Verification | 0 | No typechecker change; `callImageBase64Result` type-checks locally like every std export |
| A6: Safe Concurrency | 0 | Handler table is built once before evaluation; dispatch stays single-threaded (existing AIContext contract) |
| A7: Machines First | +1 | Removes the need for AI-generated programs to hand-roll provider HTTP over `std/net` (the current workaround) — the routing decision becomes one typed call |
| A8: Minimal Syntax | +1 | No new syntax; one new std function + one new positional string param following the existing `model=""` convention |
| A9: Cost Visibility | +1 | Every cross-provider call lands on a handler whose lane was announced at startup; trace events + `LastRoutingMetadata` already carry cost/routing telemetry |
| A10: Composability | +1 | `step*()`, `callImage*` options-model, and the new third param all route through ONE router — no per-op routing logic |
| A11: Structured Failure | +1 | `callImageBase64Result` returns `Result[string, AIError]`; missing-credential routing refusals are typed `AIError`s, not aborts |
| A12: System Boundary | 0 | No new boundary crossings; same providers, same credentials, same process |

**Net Score: +10** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced (credential presence is read once at setup)
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted — a program can only reach a provider whose key the operator already exported; alternates never read `--ai-key-file`
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

## Problem Statement

**Reporter:** stapledons_godot (via `ailang messages`, source audited at v0.51.0; findings re-verified against v0.57.0 — see Verification Log).

stapledons-godot's AI service routes **text to OpenRouter (cheap models)** and **portraits/voice to Gemini**, in one process (Mark's "both providers" ruling). Today `--ai` binds exactly ONE provider per process, and every per-call escape hatch refuses to leave it:

**Current State (verified at v0.57.0, commit b23cadce):**

1. **Per-call cross-provider models are refused.** `cmd/ailang/ai_handlers.go:188-204` (`makeModelResolver`, refusal at :200) and `cmd/ailang/ai_prepare.go:178-188` (`frozenModelResolver`, refusal at :185) return a typed `CodeModelNotAllowed` `AIError` whenever a `step()`/`stepWithCache()`/`stepWithStream()` model resolves (via models.yml) to a provider different from the `--ai` handler's. An id *unknown* to models.yml is passed through to the bound provider unchanged (V3, V4) — so `step("openrouter:vendor/model", …)` under `--ai gemini-2.5-flash` cannot reach OpenRouter at all.
2. **Image per-call models are locked to the bound provider too.** `internal/effects/ai_image.go:37-56` (`resolveImageOptions`) runs the options `"model"` key through the same resolver — same refusal (V5). The per-call model itself shipped (#1500/#1496), but only *within* the bound provider, so an image model still forces the whole-process binding to Gemini (gap G2 remainder, ailang#1496).
3. **Image ops abort the program on provider failure.** `AI.callImageBase64` returns `string ! {AI}`, not `Result` — `internal/effects/ai_image.go:132-147` (`aiCallImageBase64`) wraps handler errors into a plain op error, which aborts evaluation. There is no way to answer `provider_error` and continue. Every other AI surface already has a Result variant (`callResult`, `callJsonResult`, `callJsonSimpleResult`, `step*`, `callSpeech`); the image ops are the only non-Result AI calls left (V1, V2).

**Reporter's workaround in tree (AI.5, now largely obsolete at HEAD):** `std/ai` bound to Gemini for images and text; OpenRouter chat completions hand-rolled over `std/net` with `Authorization: Bearer`; TTS over `std/net`. Note at HEAD the reporter no longer needs the `std/net` TTS bypass (`callSpeech` shipped, ailang#1495 — V8), and `--ai-stub-fixtures` (ailang#1497) is available for offline tests.

**Impact:**
- **Who:** any AILANG program (or host embedding `internal/effects`) that wants mixed-provider economics — cheap text from one vendor, images/speech from another — in one deterministic process.
- **How significant:** blocker for the reporter (forced into `std/net` raw HTTP, losing capability gating, tracing, and routing telemetry); annoyance for everyone else (the `use --ai <model>` hint in the refusal message is the only recourse).

## Goals

**Primary Goal:** Let one `--ai`-bound process dispatch per-call AI work to any provider whose credential is present — via ONE routing mechanism shared by `step*()` and the image ops — and make image generation failure catchable via `Result`.

**Success Metrics:**
- `step("openrouter:vendor/model", …)` under `--ai gemini-…` reaches OpenRouter when `OPENROUTER_API_KEY` is present (typed refusal, naming the missing credential, when it is not).
- `AI.callImageBase64` options-model / the new third param reaches a Gemini alternate under an OpenRouter-bound process (and vice versa).
- A provider 5xx inside `callImageBase64Result` returns `Err(AIError{retryable})`; the program continues.
- Zero change to the `ModelResolver` contract and zero parser/typechecker/codegen change (all existing fixtures type-check unchanged — V11).
- One routing code path (not per-op): `step`, `stepWithCache`, `stepWithStream`, `resolveImageOptions`, and the new op all dispatch through the same router.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Alternate handlers are built from **env/ADC credential presence** at handler-setup time; no new CLI flag — key presence is the gate, a startup stderr line announces the live alternates | Defines the authority/cost surface (A4/A9): which vendors a program may bill; changing it later = re-litigating the capability model | human | design | high |
| New `effects.ModelRouter` type **supersedes** `ModelResolver` when attached; `ModelResolver` and its refusal contract stay untouched for embedders/tests | Keeps `internal/effects` exported API back-compat; wrong shape here breaks `internal/embed` consumers and every test that injects a resolver | agent | design | med |
| Router resolution order: models.yml lookup → explicit prefixes (`openrouter:`, `ollama:`, `ollama/`, `chatgpt/`) → OpenRouter `vendor/model` form → **passthrough to bound** (anything else) | This order *is* the observable semantics; reordering silently changes which handler a given string reaches | human | design | med |
| `callImageBase64Result(prompt, options, model)` — third positional `model: string` where `""` falls back to options `"model"` then the bound default; non-empty third param **overrides** options `"model"` | Freezes the std/ai API surface; a later signature change is a breaking release | human | design | med |
| Alternates are built **without** `CheckConnection` probes; a dead alternate surfaces its provider error on first use | Startup latency and determinism (no network at setup); "fail loudly at use" instead of eager probing | agent | design | low |
| `routingPolicy`/`attribution` are attached **only** to OpenRouter alternates; other providers get no policy | A non-zero policy on a non-OpenRouter handler is `ErrRoutingNotSupported` on first call (read: `WithRoutingPolicy` contract) | agent | compile | low |
| Missing-alternate refusals reuse `CodeModelNotAllowed` with a credential hint (no new error code) | Error codes are a shared namespace; adding one is a stability-surface decision | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Alternates gate = credential presence (env/ADC), announced on stderr at setup; no new CLI flag (D1)
- [x] `ModelRouter` supersedes `ModelResolver`; resolver contract unchanged (D2)
- [x] Resolution order fixed as listed (D3)
- [x] `callImageBase64Result(prompt, options, model)` surface + precedence (D4)
- [ ] Human sign-off on D1 (billing surface) and D4 (API surface) at design review — **sprint-executor pauses if unchecked**

## Solution Design

### Overview

The `--ai` flag keeps its meaning: it binds the **default** handler (the one `call`/`callJson` use, and the fallback for `model=""`). What changes is the **per-call** path: instead of resolving a model name only to refuse it when it names another vendor, the CLI builds a small table of **alternate handlers — one per provider whose credential is present in the environment** — plus a **router** that maps a per-call model string to `(apiName, handler)`. The `AIContext` dispatches `step*()` and the image ops through the router when one is attached; without one, behaviour is bit-for-bit today's.

Second, `std/ai` gains `callImageBase64Result(prompt, options, model) -> Result[string, AIError]`, the Result-returning variant of `callImageBase64`, constructed with the same helpers every other Result AI op uses (`makeOkStringResult` / `makeAIErrorResultRecord` / `classifyOpError`).

No parser, typechecker, or codegen change: the new op is a host-side effect op (`RegisterOp("AI", "callImageBase64Result", …)`) reached through the builtin → `effects.Call` registry path both engines already use (V10, V13).

### Architecture

**Components:**

1. **`effects.ModelRouter`** (`internal/effects/ai.go`) — new type:
   ```go
   // ModelRoute is where a per-call model string resolves to.
   type ModelRoute struct {
       APIName  string // provider-ready model id (prefix stripped / friendly name resolved)
       Provider string // canonical provider name; "" = the bound handler
   }

   // ModelRouter resolves a per-call model string to its destination handler.
   // Supersedes ModelResolver when attached (WithModelRouter); the ModelResolver
   // contract — including its cross-provider refusal — is unchanged for
   // embedders that do not attach a router.
   type ModelRouter func(model string) (route ModelRoute, handler AIHandler, err error)
   ```
   The router *returns the handler* so all routing knowledge (models.yml, prefix inference, alternates table, missing-credential errors) stays in `cmd/ailang`, where the factory and credentials live. `internal/effects` stays free of provider/factory imports.

   `AIContext` gains a `modelRouter ModelRouter` field + `WithModelRouter(r ModelRouter) *AIContext`. Dispatch sites (`Step`, `StepWithCache`, `StepWithStream`, and a new `routeImageModel` used by `CallImage*`/`resolveImageOptions`):
   - `model == ""` → bound handler default (unchanged short-circuit, before any routing);
   - router attached → `(apiName, handler, err) = c.modelRouter(model)`; dispatch `handler`;
   - no router → existing `resolveModel` path, byte-for-byte.

   `internal/effects/ai_lazy.go:46` currently copies `modelResolver` onto the resolved context — the router must be copied the same way (`resolved.WithModelRouter(c.modelRouter)`). Missing this copy silently drops routing on the `ailang run` lazy path (V7).

2. **Alternates builder + router** (`cmd/ailang/ai_handlers.go`, shared with `ai_prepare.go`):
   - **Probe list** (env-key providers + Google): `anthropic` (`ANTHROPIC_API_KEY`), `openai` (`OPENAI_API_KEY` or `OPENAI_BASE_URL` unauthenticated lane), `openrouter` (`OPENROUTER_API_KEY`), `google` (ADC, unless `--ai-no-adc`/`AILANG_AI_NO_ADC=1`, else `GOOGLE_API_KEY`/`GEMINI_API_KEY`), `lyceum` (`LYCEUM_API_KEY`), `zai` (`ZAI_API_KEY`). **Excluded:** `ollama` (local daemon, no key — bind `--ai ollama:…`), `chatgpt` (OAuth subscription lane — a user login, not a process key), `codex` (refused by factory), config-driven providers (no key semantics). Construction via `factory.New(provider)` with **env-only auth**: alternates never read `--ai-key-file` (that key belongs to the bound provider's lane, #1499).
   - A provider whose construction fails (no credential) is simply absent from the table — that *is* the "keys present" gate; the failure mode on use is loud (below).
   - Alternate handlers get `WithMaxTokens(row.MaxOutputTokens)` from the models.yml row **of the per-call model** when the router resolves one (0 → the 4096 default; mirrors `declaredMaxOutputTokens`'s reasoning), and `WithRoutingPolicy`/`WithAttribution` **only when the alternate is OpenRouter** (D6). Handlers may be cached per `(provider, apiName)` in the router closure.
   - **Startup announcement** (one line, stderr): `AI: cross-provider routing: openrouter(api-key), google(adc)` — cost visibility for an operator with several keys (D1).
   - The router resolves per-call strings in this order (D3):
     1. models.yml known (`ResolveModelName`) → `(api_name, provider)`;
     2. explicit prefixes `openrouter:` / `ollama:` / `ollama/` / `chatgpt/` → that provider, model = remainder;
     3. OpenRouter `vendor/model` form (contains `/` with a known vendor prefix) → openrouter, model unchanged;
     4. otherwise → passthrough to the bound handler unchanged (preserves today's unknown-id passthrough exactly, V4).
     For (1)–(3) pointing at a provider **without** an alternate handler → typed `CodeModelNotAllowed` `AIError` naming the provider and the missing credential (e.g. `set OPENROUTER_API_KEY or run --ai openrouter/<model>`). This keeps the old refusal semantics for keyless processes and upgrades only the message.
   - Both injection sites get the same router: `setupAIHandlerFromConfig`/`setupAIHandlerDirect` (exec/repl/eval) **and** `preparedRunAI.initialize` (the `ailang run` lazy path, which builds its context inside the closure — the alternates table must be built there too, from the same probe function).

3. **`callImageBase64Result`** (Result image op):
   - `internal/effects`: `RegisterOp("AI", "callImageBase64Result", aiCallImageBase64Result)`; type-checks 3 string args (`prompt`, `options`, `model`); routes `model` (D4 precedence: non-empty third param overrides options `"model"`, `""` → options `"model"`, both empty → bound default) via the same router; substitutes the resolved api_name into the options JSON (`ai.SetImageOptionsModel`); on handler error → `classifyOpError` → `makeAIErrorResultRecord` → `Err(AIError)`; on success → `makeOkStringResult`. Traces via `RecordAIEffect("callImageBase64Result", …)` like `callResult` does.
   - `internal/builtins/ai_image.go`: `_ai_call_image_base64_result` — `BuiltinSpec` mirroring `_ai_call_image_base64` (3 args, Effect `AI`, `(string, string, string) -> Result[string, AIError] ! {AI}`, `RequireCapWithBudget("AI", "")`), full `BuiltinMetadata` (Since v0.57.1, SeeAlso `callImageBase64`/`bytes.fromBase64`).
   - `std/ai.ail`: `export func callImageBase64Result(prompt: string, options: string, model: string) -> Result[string, AIError] ! {AI} = _ai_call_image_base64_result(prompt, options, model)` — placed next to `callImageBase64`, with the `model=""` convention documented exactly as `step` documents it (V2).
   - `--ai-stub` works unchanged: `StubAIHandler.CallImageBase64` already returns the stub PNG JSON; the op wraps it in `Ok`.

### Implementation Plan

**Phase 1: Router core (effects + cmd)** (~2.5 days)
- [ ] Add `ModelRoute`/`ModelRouter` + `WithModelRouter`; thread dispatch through `Step`/`StepWithCache`/`StepWithStream` and a new shared image-model route helper; copy the router in `ai_lazy.go ready()`
- [ ] Unit tests: router attached → dispatches alternate handler; no router → resolver path unchanged (incl. the two existing `CodeModelNotAllowed` tests, which must keep passing untouched); `model=""` short-circuit; lazy path retains router
- [ ] Build `cmd/ailang` alternates probe + router; replace the two resolver injections; startup announcement line; unit tests for the resolution order table (1)–(4) and the missing-credential refusal

**Phase 2: `callImageBase64Result`** (~1 day)
- [ ] Effects op + builtin + std export (D4 precedence)
- [ ] Stub path; `--ai-stub-fixtures` miss behaviour consistent with the other image ops
- [ ] Unit tests: Ok shape, Err classification (`AuthFailed`/`RateLimit`/…), third-param override, options-only model, both-empty bound default

**Phase 3: Tests, example, docs** (~1.5 days)
- [ ] New runnable example `examples/runnable/ai_cross_provider.ail` (mixed-provider: `step` on OpenRouter + `callImageBase64Result` portraits on Gemini, `match`-on-`Err` retry)
- [ ] Docs: `docs/docs/guides/ai-routing.md` (cross-provider section), `docs/docs/guides/ai-effect.mdx` surface list, `docs/docs/reference/stdlib.md` entry
- [ ] Regression pass: every fixture in Conflict Surface type-checks + runs under `--ai-stub` (V11)
- [ ] Changelog entry (changelogs/v0.32-current.md)

### Files to Modify/Create

**New files:**
- `cmd/ailang/ai_alternates.go` (~180 LOC) — probe list, alternates builder, `crossProviderRouter`, startup announcement
- `examples/runnable/ai_cross_provider.ail` (~120 LOC) — mixed-provider demo

**Modified files:**
- `internal/effects/ai.go` (+~70 LOC) — `ModelRoute`/`ModelRouter` types, `WithModelRouter`, dispatch in `Step`/`StepWithCache`/`StepWithStream`
- `internal/effects/ai_image.go` (+~60 LOC) — shared image-model routing through the router; new `aiCallImageBase64Result` op (+ registration)
- `internal/effects/ai_lazy.go` (+~5 LOC) — copy router onto resolved context
- `cmd/ailang/ai_handlers.go` (~±40 LOC) — inject router instead of resolver (both setup paths)
- `cmd/ailang/ai_prepare.go` (~±30 LOC) — same for the lazy `run` path
- `internal/builtins/ai_image.go` (+~80 LOC) — `_ai_call_image_base64_result` spec + impl
- `std/ai.ail` (+~15 LOC) — `callImageBase64Result` export + docs comment
- `internal/effects/ai_test.go`, `ai_image_test.go`, `ai_step_test.go` (+~150 LOC) — router dispatch tests
- `docs/docs/guides/ai-routing.md`, `docs/docs/reference/stdlib.md` (docs)

## Examples

### Example 1: stapledons-godot's mixed-provider service (the reporter's ask)

**Before (today, v0.57.0):** bind Gemini for images; hand-roll OpenRouter text over `std/net`:

```
ailang run --ai gemini-2.5-flash-image …   # text also goes to Gemini (expensive)
-- or --                                    # or bypass std/ai entirely:
let resp = post("https://openrouter.ai/api/v1/chat/completions",
                headers([("Authorization", "Bearer ${key}")]), body)   -- std/net, no tracing/caps
```

**After:**

```
export OPENROUTER_API_KEY=sk-or-…   # already exported on the service
ailang run --caps AI,IO --ai openrouter/deepseek/deepseek-v4-flash --entry main service.ail
# stderr: AI: Using OpenRouter (api-key)
#         AI: cross-provider routing: google(adc)
```

```ail
module service/portraits
import std/ai (step, callImageBase64Result, Message, ToolSchema)
import std/result (Result)

-- Cheap text stays on the bound OpenRouter handler (no model needed).
export func narrate(scenario: string) -> Result[string, AIError] ! {AI} =
  step("", [{ role: "user", content: scenario, tool_calls: [], tool_call_id: "", images: [] }], [])

-- Portraits reach the Gemini alternate by naming its model per call;
-- a provider 5xx is answerable, not fatal.
export func portrait(who: string) -> Result[string, AIError] ! {AI} =
  callImageBase64Result("a portrait of ${who}, oil on canvas", "{\"aspect_ratio\": \"1:1\"}", "gemini-2.5-flash-image")
```

### Example 2: the refusal gets better, not silently looser

Under a process with **no** OpenRouter key, `step("openrouter:anthropic/claude-sonnet-4.5", …)` still fails — but typed and actionable:

```
Err({code: "ModelNotAllowed", message: "per-call model "openrouter:anthropic/claude-sonnet-4.5"
     routes to provider "openrouter", which has no handler in this process (no OPENROUTER_API_KEY);
     export it or run --ai openrouter/anthropic/claude-sonnet-4.5", retryable: false})
```

## Success Criteria

- [ ] `step("openrouter:<vendor/model>", …)` under a non-OpenRouter `--ai` reaches the OpenRouter alternate when `OPENROUTER_API_KEY` is present; typed `ModelNotAllowed` refusal naming the missing env var when absent (unit test both)
- [ ] `callImageBase64` options-model and `callImageBase64Result` third param reach a cross-provider alternate (unit test with two fake handlers)
- [ ] `callImageBase64Result` returns `Ok(json)` on success and `Err(AIError)` on a handler error; program continues after `Err` (unit test + example under `--ai-stub` + `--ai-stub-fixtures`)
- [ ] Resolution-order table (models.yml → explicit prefixes → OR vendor form → bound passthrough) unit-tested, including unknown-id passthrough preservation (V4)
- [ ] Both existing `CodeModelNotAllowed` tests (`internal/effects/ai_image_test.go:122`, `ai_test.go:196`) pass **unmodified** (resolver contract untouched)
- [ ] Startup announcement lists live alternates once (no per-call noise)
- [ ] All fixtures in Conflict Surface type-check AND run under `--ai-stub`
- [ ] `make test-core`, `make fmt`, `make lint`, `make check-boundaries` pass
- [ ] Documentation updated (ai-routing.md, stdlib.md, changelog) and example added
- [ ] Engine parity: evaluator and strict VM agree — the op is host-side (builtin → `effects.Call` registry), no codegen surface; covered by the existing AI-op test pattern

## Testing Strategy

**Unit tests:**
- Router: each row of the resolution-order table; missing-alternate refusal message; `model=""` short-circuit; no-router fallback to resolver; alternates never read `--ai-key-file`; `--ai-no-adc` removes the google alternate
- Effects dispatch: `Step`/`StepWithCache`/`StepWithStream`/image ops pick the alternate handler; `LastRoutingMetadata` still reads from the handler that served the call
- New op: arg count/type errors, Ok/Err shapes, classify passthrough of pre-typed `*ai.AIError`

**Integration tests:**
- `ailang run --ai-stub --ai-stub-fixtures` end-to-end through the new example; fixture miss is a loud error (existing pattern)
- Lazy path (`NewLazyAIContext`) retains the router after `ready()`

**Regression-surface tests** (Conflict Surface filled in, so required):
- One run per fixture below under `--ai-stub`, pinning output where deterministic

**Manual testing:**
- Real two-key process (OpenRouter + Gemini): cross-provider `step` + image call; verify the OpenRouter Broadcast observability trace shows the call (second instrument, per CLAUDE.md §1)

## Non-Goals

**Not in this feature:**
- **Result variants for `callImage`/`callImage*WithRefs`** — FS-write errors aren't `AIError`-shaped; add later if a consumer needs them (Future Work)
- **Cross-provider `callSpeech`** — TTS is Google-only by capability (`ai/speech.go` returns a typed not-supported error for other providers, V12); routing can't fix a capability gap
- **Conflating this with `--routing-*` / `! {AI[mode=routeable]}`** — that axis is *within-OpenRouter* model fallback (`examples/ai_modes.ail`); cross-provider routing is orthogonal and requires no mode marker (per-call model strings name their destination explicitly)
- **Ollama/ChatGPT/OAuth alternates** — no env-key gate exists for those lanes (D1 rationale); bind `--ai` to them instead
- **Any CLI flag, syntax, or effect-row change** — none needed (A8/A3)
- **Provider failover between alternates** — the program names its destination deterministically; failover policy stays with `--routing-*` on OpenRouter

## Timeline

**Week 1** (3 days):
- Phase 1: router core + alternates (2.5 days)
- Phase 2 start: effects op (0.5 day)

**Week 2** (2 days):
- Phase 2 finish: builtin + std export (0.5 day)
- Phase 3: tests, example, docs, changelog (1.5 days)

**Total: ~5 days across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Lazy `run` path loses the router (the `ai_lazy.go` copy is easy to miss) | High | Explicit task + test in Phase 1; `ready()` copy is one line (V7) |
| Routing policy attached to a non-OpenRouter alternate → `ErrRoutingNotSupported` on first call | Medium | D6: policy/attribution only on openrouter alternates; unit test |
| Silent billing-surface expansion for operators with several keys | Medium | D1 startup announcement + refusal names the credential; `--ai-no-adc` and unsetting keys shrink the table; documented in ai-routing.md |
| Alternates table built per evaluation → nondeterminism if env changes mid-process | Low | Table is built once (setup or lazy-init) and never re-probed; per-call routing is pure lookup |
| `maxTokens` mismatch on alternates (reasoning models hitting `finish_reason=length`) | Medium | Router sets `WithMaxTokens` from the resolved models.yml row (Phase 1); mirrors `declaredMaxOutputTokens` trap (its comment documents the 2026-09-27 motoko A/B) |

## Conflict Surface

### Syntactic positions touched

- **None in the grammar.** No parser/lexer/AST/typechecker/codegen change. The touched surfaces are: the `std/ai` export list (one new function), the `AI` effect op registry (one new op name), the `AIContext` per-call dispatch (router supersedes resolver), and the two `cmd/ailang` setup paths.

### What else lives here

| Position | Existing valid form | Shape |
|----------|--------------------|-------|
| `AIContext.Step/StepWithCache/StepWithStream` per-call model param | resolver path (`resolveModel`) | `model → api_name` within bound provider; `CodeModelNotAllowed` otherwise |
| Same | `model=""` short-circuit | bound handler default, **before** any resolution |
| Image ops options `"model"` key (`resolveImageOptions`) | substitution within bound provider | same refusal for cross-provider |
| `callSpeech` options `"model"` key | Google-only TTS; typed not-supported for others | must NOT start routing (Non-Goal) |
| `! {AI}` / `! {AI[mode=fixed|routeable]}` rows | mode markers attesting OpenRouter runtime routing | unchanged; cross-provider needs no marker |
| Lazy `AIContext` (`NewLazyAIContext`) | resolver copied onto resolved context in `ready()` | router must be copied too (V7) |

### Disambiguation strategy

Router attached (`WithModelRouter` non-nil) → router path; nil → resolver path unchanged. `model==""` short-circuits to the bound default before either. The router's own disambiguation is the fixed resolution order (D3): a string that matches nothing inferable passes through to the bound handler *exactly as today* (V4) — so no existing program can change destination unless it names another provider explicitly.

### Programs that MUST still work

All verified type-checking today at v0.57.0 (V11):

- `examples/runnable/ai_caching.ail` — `stepWithCache("", …)` + `Result[StepResult, AIError]` match
- `examples/runnable/ai_tool_loop.ail` — `runTools` → `step` loop
- `examples/runnable/ai_image_generation.ail` — `callImage`/`callImageBase64`/`callImageWithRefs` + per-call options model (incl. the OpenRouter-bound image flow)
- `examples/runnable/ai_call_result.ail` — Result-call pattern the new op mirrors
- `examples/runnable/ai_call_speech.ail` — `callSpeech` options model (must not cross-route)
- `examples/ai_modes.ail` — effect modes + `--routing-*` OpenRouter policy (must keep its gate semantics)

### What deliberately changes

- Cross-provider per-call models under multi-key processes: routed (was: `CodeModelNotAllowed`). Single-key processes: identical refusal, **improved message** (names the missing credential).
- Unknown ids that today *look* prefixed (e.g. `step("ollama:x", …)` under `--ai gemini`): typed routing refusal with a bind hint (was: passthrough → bound provider's own `ModelNotFound`-shaped error).
- `std/ai` gains one export; no existing signature changes.

## Verification Log

Every "supports / doesn't support" claim above was checked live at v0.57.0 (b23cadce) on 2026-11-03:

| # | Claim | Evidence |
|---|-------|----------|
| V1 | `callImageBase64(prompt, options) -> string ! {AI}` exists; **no** Result variant exists | `std/ai.ail:409` (read); `ailang check` on a module importing `callImageBase64Result` → `Error: IMP010: symbol 'callImageBase64Result' not exported by 'std/ai' — did you mean 'callImageBase64'?` |
| V2 | `step*` return `Result[StepResult, AIError]`; `model=""` = bound default is the documented convention | `std/ai.ail:193-211` (read, "Pass model=\"\" to use the handler-bound model"); `ailang check examples/runnable/ai_caching.ail` → `✓ No errors found!` |
| V3 | Cross-provider per-call refusal exists at BOTH setup paths | `cmd/ailang/ai_handlers.go:188-204` (`makeModelResolver`, refusal at :200), `cmd/ailang/ai_prepare.go:178-188` (`frozenModelResolver`, refusal at :185) — both return `CodeModelNotAllowed` (read) |
| V4 | Unknown ids pass through to the bound provider | `makeModelResolver`: "Unknown to models.yml — assume it is already an api_name and let the provider be the final authority… return model, nil" (read) |
| V5 | Image options-model routes within bound provider only | `internal/effects/ai_image.go:37-56` `resolveImageOptions` runs `c.modelResolver(opts.Model)`, same refusal (read) |
| V6 | Existing tests pin the refusal and must keep passing unmodified | `internal/effects/ai_image_test.go:122-123`, `internal/effects/ai_test.go:196` assert `CodeModelNotAllowed` (read) |
| V7 | Lazy context copies the resolver in `ready()` — router needs the same copy | `internal/effects/ai_lazy.go:46` `resolved.WithModelResolver(c.modelResolver)` (read) |
| V8 | Reporter's v0.51.0 gaps #1495/#1497 already shipped at HEAD (doc must not re-plan them) | `std/ai.ail:177` `callSpeech` + `internal/ai/gemini/speech.go`; `ailang run --help` lists `-ai-stub-fixtures`; changelogs/v0.32-current.md:3687, :4031 |
| V9 | `CodeModelNotAllowed = "ModelNotAllowed"` is allocated; no `ProviderUnavailable`-style code exists to reuse | `internal/ai/errors.go:30` (read); `grep -rn "ProviderUnavailable\|ProviderNotConfigured" internal/ cmd/` → 0 hits |
| V10 | Result-construction + classification helpers for AI ops already exist | `internal/effects/ai_encode.go:48` (`makeOkStringResult`), `:104` (`makeAIErrorResultRecord`), `internal/effects/ai_step.go:37` (`classifyOpError`) (read) |
| V11 | Every fixture named in Conflict Surface type-checks today | `ailang check` on `ai_caching.ail`, `ai_image_generation.ail`, `ai_call_result.ail`, `ai_call_speech.ail` → all `✓ No errors found!` |
| V12 | `callSpeech` is Google-only; non-google providers get a typed not-supported error | `internal/ai/speech.go:77` `NewAIError(..., "callSpeech: text-to-speech is not supported by provider %q (supported: gemini)", ...)` (read) |
| V13 | New AI ops are host-side and need no parser/codegen change | builtin Impls route `→ effects.Call` → op registry (`internal/effects/ops.go:83`, `internal/builtins/ai_image.go:136` `aiCallImageBase64Impl`); strict-VM tests inject via the same `effects.RegisterOp` registry (`internal/vm/terminal_boundary_test.go:90`) (read) |
| V14 | `--ai-key-file`/`--ai-no-adc` exist and are bound-lane scoped (#1499) | `ailang run --help` (both listed); `cmd/ailang/ai_handlers.go:80` `cfg.resolveAuth()`, `factory.WithNoADC` docs (read) |
| V15 | Routing policy on non-OpenRouter handlers is refused on first call | `internal/ai/handler.go:52-59` `WithRoutingPolicy` contract: "other providers will reject a non-zero policy with `ErrRoutingNotSupported`" (read) |
| V16 | A `Result[Qualified.Type, …]` annotation does not parse — doc examples must use named imports | `ailang check` on `Result[AI.StepResult, AI.AIError]` → `PAR_UNEXPECTED_TOKEN … expected next token to be ], got .` (probe) |

## Deferred Decisions

The following are intentionally left open for the implementer:

- Alternate handler caching granularity (per-provider vs per-(provider, apiName)) — agent may choose, provided `maxTokens` comes from the resolved models.yml row
- Exact startup announcement wording/format — agent may choose (must be one line, stderr, lists lane names)
- Whether the router closure lives in `ai_alternates.go` or beside `makeModelResolver` — agent may choose
- Unit-test file placement for the new effects op (`ai_image_test.go` vs a new `ai_image_result_test.go`) — agent may choose
- Whether the example uses `--ai-stub-fixtures` too — agent may choose

## Related Documents

**Implemented (may inform design):**
- `changelogs/v0.32-current.md` — "OpenRouter image output and a per-call model for `callImage*` (#1500, #1496)", "reference-conditioned image generation (#1496)", "`std/ai.callSpeech` (#1495)", "`--ai-stub-fixtures` (#1497)"

**Planned (check for overlap):**
- `design_docs/planned/m-ai-call-context.md` — AI context plumbing; distinct (no cross-provider routing), coordinated via the same `AIContext`
- `design_docs/planned/m-provider-failover.md` — OpenRouter-internal fallback; orthogonal axis (see Non-Goals)

**Issues:** ailang#1496 (G2 image per-call model — partially shipped, remainder addressed here), ailang#1495/#1497 (shipped; reporter's v0.51.0 context). Reported by stapledons_godot via `ailang messages`.

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [docs/docs/guides/ai-routing.md](/docs/docs/guides/ai-routing.md) - OpenRouter runtime routing (the orthogonal axis)
- [docs/docs/guides/ai-effect.mdx](/docs/docs/guides/ai-effect.mdx) - AI effect surface
- `internal/effects/ai.go` - `ModelResolver` contract (unchanged by this design)
- `internal/ai/factory/factory.go` - credential lanes (`LaneAPIKey`, `LaneADC`, …) the alternates probe reuses

## Future Work

- Result variants for `callImage` / `callImage*WithRefs` (FS-error typing question first)
- `--ai-stub-fixtures`-driven end-to-end test of cross-provider *selection* under stub (needs per-provider stub handlers)
- Config-driven (`[[ai_provider]]`) alternates if a consumer asks
- Routing a whole *process lane* (not per-call) — e.g. `--ai-text`/`--ai-image` split flags; rejected for now (per-call strings already carry the destination)

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03