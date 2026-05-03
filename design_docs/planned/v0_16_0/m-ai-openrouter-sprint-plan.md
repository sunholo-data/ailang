# Sprint Plan: M-AI-OPENROUTER

## Summary

Add OpenRouter as an AILANG AI provider, with a new `AI[Routeable]` effect-row marker that captures dynamic provider routing in the replay trace. Unlocks ~100 models behind one HTTP API and lays foundation for cost-aware agent routing.

**Sprint ID:** M-AI-OPENROUTER
**Target Version:** v0.16.0
**Design Doc:** [design_docs/planned/v0_16_0/m-ai-openrouter-provider.md](m-ai-openrouter-provider.md)
**Duration:** 4 days (~26-34 hours, 2-3 calendar weeks if part-time)
**Dependencies:** None (M-UNIFIED-AI-PROVIDERS shipped in v0.5.10; M-ARCH1 base-class refactor is *not* blocking)
**Risk Level:** Medium

**All design-freeze decisions APPROVED by user (2026-05-03):**
- ✅ `AI[Routeable]` row marker on AI effect
- ✅ `ResolvedRoute` trace schema (requested/resolved model+provider, fallback chain, token + cost fields)
- ✅ Replay policy: pin-to-resolved by default, optional `--reroute` flag
- ✅ `AI[BYOK]` as separate row marker, stubbed in this milestone, full semantics deferred

## Current Status Analysis

### Completed Recently (last 14 days, informs velocity)
- ✅ **M-SMT-CROSS-MODULE-FUNCTIONS** (5 milestones, ~5 days): ImportedPrograms plumbing, inline imports, contract-as-spec fallback, examples
- ✅ **M-PKG-CASCADE-DETERMINISTIC-FIRST** (multi-phase): wrapper deterministic bump, template variables, conservative C-classification
- ✅ **M-AGENT-MCP** sprint (7/8 milestones): server-side filtering, per-module stdlib snapshots, MCP HTTP transport hardening
- ✅ Several telemetry / cascade fixes shipped

### Velocity
- **Recent average**: ~150-250 LOC/day for milestone-style work; multi-day milestones land in 1-3 calendar days
- **Estimated capacity for this sprint**: ~1,400-1,700 LOC (implementation + tests) over 4 working days
- **Confidence**: Medium — Phase 3 touches the type-system effect-row machinery, which is higher-risk than the adapter work

### Remaining from Design Doc
- ⏳ **M1: Thin OpenRouter adapter** (~600 LOC impl + 300 LOC tests = 900 LOC)
- ⏳ **M2: Routing policy IR** (~350 LOC impl + 150 LOC tests = 500 LOC)
- ⏳ **M3: AI[Routeable] + trace schema** (~400 LOC impl + 200 LOC tests = 600 LOC)
- ⏳ **M4: Eval, docs, release** (~150 LOC + docs + CHANGELOG)

**Total estimate:** ~2,150 LOC across 4 milestones, 4 working days.

## Proposed Milestones

### M1: Thin OpenRouter Adapter

**Goal:** Ship a working OpenRouter HTTP client implementing `internal/ai/Provider`, callable from CLI and eval harness, with no routing-policy support yet.

**Estimated:** ~600 LOC implementation + ~300 LOC tests = 900 LOC
**Duration:** 1 day (~6-8 hours)

**Tasks:**
- Day 1 (morning): Scaffold `internal/ai/openrouter/` from `internal/ai/openai/` template (`client.go`, `chat.go`, `handler.go`, `usage.go`)
- Day 1 (morning): Wire `NewClient(apiKey, ...ClientOption)`, `WithBaseURL`, `WithHTTPClient`, OTEL instrumentation
- Day 1 (afternoon): Implement Chat Completions request/response, including OpenRouter-specific `usage` block fields (cached_tokens, reasoning_tokens, total_cost)
- Day 1 (afternoon): Add HTTP-fixture unit tests covering happy path, 4xx auth error, 5xx transient, malformed JSON
- Day 1 (afternoon): Wire into `cmd/ailang/ai_handlers.go` (`--provider openrouter`) and `internal/eval_harness/api_openrouter.go`
- Day 1 (afternoon): Add 8-12 OpenRouter-routed model entries to `models.yml` spanning vendors/price tiers

**Files to create:**
- `internal/ai/openrouter/client.go` (~150 LOC)
- `internal/ai/openrouter/chat.go` (~200 LOC)
- `internal/ai/openrouter/usage.go` (~60 LOC)
- `internal/ai/openrouter/handler.go` (~80 LOC)
- `internal/ai/openrouter/client_test.go` (~150 LOC)
- `internal/ai/openrouter/chat_test.go` (~150 LOC)
- `internal/eval_harness/api_openrouter.go` (~100 LOC)

**Files to modify:**
- `cmd/ailang/ai_handlers.go` (~20 LOC delta)
- `models.yml` (~80 lines added)

**Acceptance Criteria:**
- [ ] `go test ./internal/ai/openrouter/...` passes with ≥80% coverage
- [ ] `ailang --provider openrouter --model anthropic/claude-sonnet-4.5 ...` runs against live OpenRouter (manual smoke test)
- [ ] Eval harness can target `openrouter/<vendor>/<model>` strings
- [ ] Usage fields (prompt/completion/cached/reasoning tokens, cost_usd) populated in `Response.Usage`
- [ ] No regression: `make test` and `make verify-examples` pass
- [ ] Linting clean: `make lint` passes

**Risks:**
- OpenRouter API drift from OpenAI Chat Completions shape — Mitigation: version-pin via HTTP fixtures, document API version assumed
- Decimal type for cost may not match existing trace handling — Mitigation: read `internal/trace/events.go` first, use matching type

### M2: Routing Policy IR

**Goal:** Generalize `ai.Request` with optional `Routing *AIRoutingPolicy`. OpenRouter consumes the policy; other providers ignore it. AILANG-side `openrouter.provider({...})` constructor exposes the policy as a record value.

**Estimated:** ~350 LOC implementation + ~150 LOC tests = 500 LOC
**Duration:** 1 day (~6-8 hours)

**Tasks:**
- Day 2 (morning): Define `AIRoutingPolicy`, `RoutePreference` (Cheapest/Fastest/MostReliable), `AICapability` enum in `internal/ai/routing.go`
- Day 2 (morning): Extend `ai.Request` struct with `Routing *AIRoutingPolicy` and `OutputSchema *JSONSchema` (the latter stubbed for now)
- Day 2 (morning): OpenRouter adapter — translate `AIRoutingPolicy` → OpenRouter `provider` field (`order`, `allow_fallbacks`, `require_parameters`, `data_collection`)
- Day 2 (afternoon): Other providers (openai/anthropic/gemini/ollama): ignore `Routing` if nil; if non-nil with non-empty `Order`, return typed error `ErrRoutingNotSupported`
- Day 2 (afternoon): Author `stdlib/std/ai/providers/openrouter.ail` exposing `provider({model, route})` constructor
- Day 2 (afternoon): Tests for policy → OpenRouter `provider` field translation, including empty order, capability conflicts, missing prefer

**Files to create:**
- `internal/ai/routing.go` (~120 LOC)
- `internal/ai/openrouter/routing.go` (~80 LOC)
- `internal/ai/openrouter/routing_test.go` (~150 LOC)
- `stdlib/std/ai/providers/openrouter.ail` (~50 LOC)

**Files to modify:**
- `internal/ai/provider.go` (~30 LOC delta — add `Routing`, `OutputSchema` fields)
- `internal/ai/openai/client.go`, `internal/ai/anthropic/client.go`, `internal/ai/gemini/client.go`, `internal/ai/ollama/client.go` (~10 LOC each — error if routing non-nil with order)

**Acceptance Criteria:**
- [x] `AIRoutingPolicy`, `RoutePreference`, `AICapability` exported from `internal/ai`
- [x] `ai.Request.Routing` is optional pointer — nil for backward compat with all existing call sites
- [x] OpenRouter request body includes `provider: {order: [...], allow_fallbacks: bool, require_parameters: [...]}` when policy present
- [x] Other providers reject non-nil routing with typed error (`ai.ErrRoutingNotSupported`)
- [ ] ~~AILANG program calling `openrouter.provider({...route: {...}})` constructs valid `Provider` value~~ — **DEFERRED** (see below)
- [x] `go test ./internal/ai/...` passes; routing-translation tests cover edge cases
- [x] Linting clean
- [x] CLI plumbing: `--routing-fallback`, `--routing-require`, `--routing-prefer`, `--routing-max-price` flags on `ailang exec`

**Deferred to follow-up:**
- AILANG-side `stdlib/std/ai/providers/openrouter.ail` constructor. The current `std/ai.ail` exposes only `call`/`callJson`, with provider config purely host-side. Adding a value-level `provider({...})` requires new builtins and a new `ai.complete(req)` entry point — out of scope for M2. Tracked for v0.17.0.
- `MaxPricePerMTok` field forwarding. The IR carries the value but `translatePolicy` silently drops it — OpenRouter's per-call max-price filter lives under `transforms` which the design doc explicitly defers. Wiring is a single switch when the broader transforms work lands.
- `OutputSchema *JSONSchema` field on `ai.Request`. Not added in M2 — `ResponseSchema string` already exists and OpenRouter happily consumes it via the existing `response_format` path. M3 (or a later milestone) can promote to a structured type when needed.

**Risks:**
- AILANG-side record-typed `route` field — `stdlib/std/ai/providers/openrouter.ail` needs to type-check cleanly under existing AI module — Mitigation: mirror existing `provider({...})` patterns from openai.ail / anthropic.ail
- Other providers' routing-rejection error path may need a new typed `AIError` variant — Mitigation: add `AIError.RoutingNotSupported` in this milestone, keep simple

### M3: AI[Routeable] Effect Row + Trace Schema

**Goal:** Make routable providers visible in the type system via `AI[Routeable]` row marker. Capture every routing decision in the trace via `ResolvedRoute`. Replay engine pins to `resolved_model` by default with optional `--reroute`.

**Estimated:** ~400 LOC implementation + ~200 LOC tests = 600 LOC
**Duration:** 1 day (~6-8 hours)
**Risk:** Highest of the sprint — touches type/effect-row machinery and trace consumers.

**Tasks:**
- Day 3 (morning): Add `Routeable`, `BYOK` (stub), `ReplayOnly` (stub) row markers to `internal/types/effect_rows.go`
- Day 3 (morning): AI effect handler (`internal/effects/ai.go`) checks routable-provider capability against declared effect row; if routable provider used under plain `! {AI}`, return `AIError.RouteableProviderNotAllowed`
- Day 3 (morning): Provider `Capabilities()` method returns `{Routeable: true}` for OpenRouter when policy present
- Day 3 (afternoon): Extend AI trace event schema (`internal/trace/events.go`) with `ResolvedRoute` block — additive, zero-valued for non-routable calls
- Day 3 (afternoon): OpenRouter handler populates `ResolvedRoute` from response (`requested_model`, `resolved_model`, `resolved_provider`, `fallback_chain`, token fields, `cost_usd`)
- Day 3 (afternoon): Replay engine — when replaying a trace event with non-empty `ResolvedRoute`, call resolved model directly (bypass routing); add `--reroute` CLI flag for "what would happen now"
- Day 3 (afternoon): Coordinate with dashboard / trace-debugger — send handoff message describing new fields
- Day 3 (afternoon): Author `examples/ai_openrouter_routing.ail` demonstrating `! {AI[Routeable]}` declaration

**Files to create:**
- `internal/effects/ai_routeable.go` (~80 LOC)
- `examples/ai_openrouter_routing.ail` (~40 LOC)
- `internal/effects/ai_routeable_test.go` (~120 LOC)
- `internal/trace/resolved_route_test.go` (~80 LOC)

**Files to modify:**
- `internal/types/effect_rows.go` (~40 LOC delta — add 3 row markers)
- `internal/effects/ai.go` (~50 LOC delta — capability gate)
- `internal/trace/events.go` (~30 LOC delta — `ResolvedRoute` field)
- `internal/ai/openrouter/handler.go` (~30 LOC delta — populate ResolvedRoute)
- Replay engine (location TBD by impl) (~50 LOC delta)
- `cmd/ailang/` — add `--reroute` flag (~20 LOC delta)

**Acceptance Criteria:**
- [ ] `AI[Routeable]` parses and type-checks; declaring `! {AI[Routeable]}` is accepted
- [ ] Routable provider used under plain `! {AI}` produces typed error `AIError.RouteableProviderNotAllowed { required: "AI[Routeable]", declared: "AI" }`
- [ ] Every OpenRouter call writes a populated `ResolvedRoute` to the trace event (unit test asserts non-empty)
- [ ] Trace schema change is additive: existing non-routable trace events still parse correctly (backward compat test)
- [ ] Replay of a routed trace uses `resolved_model`, response matches modulo expected nondeterminism
- [ ] `--reroute` flag re-runs routing logic during replay (integration test)
- [ ] `examples/ai_openrouter_routing.ail` runs end-to-end, trace shows `resolved_model` populated
- [ ] Handoff message sent to dashboard / trace-debugger owners describing new fields

**Risks:**
- **High**: Effect-row changes ripple through type-checker — Mitigation: scope row markers as simple boolean flags, defer generalization; run `make test` after each row-marker addition
- **Med**: Trace schema change breaks dashboard consumers — Mitigation: additive-only, zero-valued for old data, send handoff message before merging
- **Med**: Replay-engine change is in unfamiliar code — Mitigation: read trace-debugger guide first, write integration test before changing engine

### M4: Eval, Docs, Release

**Goal:** Demonstrate cost-visibility win with a benchmark, ship docs, land CHANGELOG, move design doc to implemented/.

**Estimated:** ~150 LOC + docs
**Duration:** 0.5-1 day (~4-6 hours)

**Tasks:**
- Day 4 (morning): Author benchmark in `benchmarks/` running same prompt across 3+ OpenRouter-routed models, capturing cost/latency/quality
- Day 4 (morning): Run benchmark, capture result table (cost-per-task delta across vendors)
- Day 4 (afternoon): Write `docs/docs/guides/ai-routing.md` covering: `AI[Routeable]` semantics, OpenRouter setup, routing policy schema, replay behavior, BYOK stub status
- Day 4 (afternoon): CHANGELOG.md entry under v0.16.0 — link design doc + sprint plan
- Day 4 (afternoon): `make ci` green; `make verify-examples` green
- Day 4 (afternoon): Move `design_docs/planned/v0_16_0/m-ai-openrouter-*.md` → `design_docs/implemented/v0_16_x/`
- Day 4 (afternoon): Add implementation report to design doc (what was built, deviations, metrics)

**Files to create:**
- `benchmarks/openrouter_cost_comparison.{ail,json}` (~80 LOC)
- `docs/docs/guides/ai-routing.md` (~200 lines)

**Files to modify:**
- `CHANGELOG.md` — v0.16.0 entry
- Move design doc + sprint plan to `design_docs/implemented/v0_16_x/`

**Acceptance Criteria:**
- [ ] Benchmark runs same task across ≥3 OpenRouter-routed models, output shows cost/token deltas
- [ ] `docs/docs/guides/ai-routing.md` published, linked from docs nav
- [ ] CHANGELOG.md entry mentions both design doc and sprint plan paths
- [ ] `make ci` green (build, test, lint, verify-examples, file-size check)
- [ ] Design doc moved to `design_docs/implemented/v0_16_x/` with implementation report appended
- [ ] Sprint plan moved alongside

**Risks:**
- Live OpenRouter calls in benchmark cost real money — Mitigation: cap benchmark spend, gate behind explicit env var, don't run in CI

## Success Metrics

- **Test coverage**: ≥80% on `internal/ai/openrouter/`, no regression elsewhere
- **Examples passing**: `examples/ai_openrouter_routing.ail` runs; `make verify-examples` green
- **Documentation updated**:
  - `docs/docs/guides/ai-routing.md` (new)
  - `CHANGELOG.md` (v0.16.0 entry)
  - Design doc + sprint plan moved to `implemented/v0_16_x/`
- **Benchmark deliverable**: Cost-per-task table across ≥3 models, demonstrating cost-visibility win
- **Trace schema**: Every routed call writes `ResolvedRoute`; backward-compat test asserts old events still parse
- **All tests passing**: ✅ (`make test`)
- **All linting passing**: ✅ (`make lint`)

## Dependencies

- **External**: `OPENROUTER_API_KEY` set for live smoke tests and benchmark; CI uses HTTP fixtures only
- **Internal**: M-UNIFIED-AI-PROVIDERS (v0.5.10, ✅ shipped) — provides `internal/ai/Provider` interface this milestone implements
- **Coordination**: Dashboard / trace-debugger owners need notice of `ResolvedRoute` field addition (handoff message in M3)

## Open Questions

(None — all design-freeze decisions approved by user 2026-05-03)

## Notes

- M-ARCH1 (provider base-class refactor) is **not** blocking. Write OpenRouter adapter so it slots cleanly into the base class when M-ARCH1 lands later.
- `AI[BYOK]` and `AI[ReplayOnly]` row markers are stubbed in M3 (parse + accept) but full semantics deferred to follow-up design docs. Stub means: row marker exists in type machinery, but no provider currently requires it.
- Benchmark in M4 should pick models with meaningfully different cost tiers (e.g., `claude-sonnet-4.5` vs `gpt-4o-mini` vs `gemini-2.5-flash`) to make the cost-visibility story land.
- If Phase 3 (effect-row machinery) hits unexpected type-checker complexity, consider splitting M3 into M3a (trace schema, no row marker) and M3b (row marker) to ship a partial improvement.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_16_0/m-ai-openrouter-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-AI-OPENROUTER.json`
