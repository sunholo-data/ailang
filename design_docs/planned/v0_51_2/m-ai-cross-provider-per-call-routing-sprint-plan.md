# Sprint Plan: M-AI-CROSS-PROVIDER-PER-CALL-ROUTING

## Summary

Enable caller-selected provider routing for step and image operations under one AI capability, and expose catchable Result errors for both base64 image functions.

**Design:** [Approved design](m-ai-cross-provider-per-call-routing.md)
**Duration:** 5 working days, approximately 30 hours including integration buffer.
**Estimated LOC:** 650 implementation + 550 tests + 100 examples/docs = 1300 total.
**Risk level:** Medium, concentrated in credential isolation and trace attribution.
**Status:** Planned; design approval supplied by task-80bb467b handoff. This sprint plan awaits coordinator review before execution.
**Progress:** `.ailang/state/sprints/sprint_M-AI-CROSS-PROVIDER-PER-CALL-ROUTING.json`

## Current Status and Estimate Evidence

Planning verified source at `4ac769384e61a77e40126c77158a1069d7b1a4dd` on 2026-10-03. `std/VERSION` is v0.52.0; retain the approved design's v0_51_2 directory as its artifact identity, but use the current release changelog at execution time. No version bump is part of this sprint.

The same-provider resolver refusal remains in `cmd/ailang/ai_handlers.go`; the direct setup path still attaches no resolver. `internal/effects/ai.go` dispatches step operations through the bound handler and reads metadata from it. The two requested Result image builtins remain absent. Existing per-call image options.model support, provider factory, error classifiers, speech API and fixture stubs are reusable prerequisites. The design audits the step/cache/stream family, all image entry points, direct/config/stub setup and metadata together; it satisfies the systemic-analysis gate.

The skill's seven-day velocity script ran, but this checkout is shallow and exposes only one dependency-update commit. Its historical changelog grep yielded no usable recent LOC rate. No empirical LOC/day is claimed. The design estimates 1100 implementation/test LOC over four days; this plan allows 1200 implementation/test LOC plus 100 examples/docs and five days (25% schedule buffer). The resulting planning capacity of 260 LOC/day is an estimate, not measured velocity.

`go` is unavailable. The PATH binary reports v0.52.0, commit bf2436a-dirty, different from HEAD, and emits a stale-binary warning. Planning does not claim a test or coverage baseline. Executor must provision the supported Go toolchain, rebuild from this checkout and bank a baseline before implementation. Existing unrelated work must remain intact.

## Registry Reuse Audit

Ran `ailang pkg search ai` (24 matches) and `ailang pkg search image` (4 matches). Inspected `ailang pkg info` and `ailang pkg docs` for sunholo/gemini_files@0.2.1 and sunholo/gemini_live@0.5.0. The former uploads assets; the latter builds Gemini voice protocol messages. Neither can change host AIContext dispatch, credential routing, builtins or capability accounting. Use existing in-tree factory/handler/error utilities; add no package dependency.

| Milestone | Decision | Reason |
|---|---|---|
| M1 | none | Runtime handler selection and metadata require internal/effects changes; registry packages cannot implement them. |
| M2 | none | CLI provider factory and credential isolation require cmd/ailang integration. |
| M3 | none | Result effect operations and builtin registrations belong to the language host and std/ai. |
| M4 | none | Examples demonstrate the in-tree API; package upload/voice helpers do not implement or validate it. |

## Implementation Clarifications

The approved design's ProviderRouter(provider) sketch lacks the selected model's max_output_tokens and env_var, although its cache and factory rules need both. Before coding M1/M2, make an explicit route descriptor carry that data from model resolution to construction, or use an equivalent explicit internal contract. Do not use hidden mutable resolver state. Preserve the public AIHandler interface and legacy ModelResolver-only behavior. This is completion of the approved routing contract; if it requires a changed behavioral decision, return to design review.

Reuse the bound handler for same-provider calls where its construction settings match the requested route; document and test any required separate budget bucket. Cache identity must isolate differing credential sources as well as token budgets if models.yml entries for one provider name distinct env vars. Never reuse a handler authenticated with another model's credential lane. Bound --ai-key-file credentials may continue serving the bound provider, but must never enter a foreign provider's factory options.

Reset per-call metadata before resolution/dispatch so an error before dispatch cannot inherit the previous provider's route. Ensure routed Gemini and other handlers without OpenRouter-specific ResolvedRoute data still produce truthful provider/model trace identity; lastDispatched alone is insufficient evidence of this acceptance criterion.

## Milestones

### M1: Effect-context routing and metadata (~450 LOC)

**Estimate:** 230 implementation + 220 tests; 1.5 days / 9 hours.
**Dependencies:** None.
**Files:** `internal/effects/ai.go`, `internal/effects/ai_image.go`, existing routing/step/image tests; add focused `internal/effects/ai_routing_test.go` if useful.
**Example coverage:** Establish recording-handler tests for cases later demonstrated in `examples/runnable/ai_cross_provider.ail` (created in M4).

Add route classification, injected router and resolveTarget/resolveImageTarget. Route Step, StepWithCache, StepWithStream and all four existing image calls through the selected handler. Retain nil/legacy resolver paths and unknown-name passthrough. Rewrite options.model only, retaining other options. Thread actual handler identity into traces; restore bound metadata on bound calls, including speech.

- [ ] Same-provider, empty-default, unknown-name, nil-handler and legacy-resolver behavior have recording-handler tests.
- [ ] Cross-provider dispatch is proven for step/cache/stream and all four image operations in both provider directions.
- [ ] Image options preserve references and unrelated keys while replacing only the resolved model.
- [ ] Routed calls followed by bound calls or resolution failures expose the correct metadata, with no stale attribution.
- [ ] AI capability denial and the shared budget remain enforced across alternating providers.

**Risk:** Optional handler interfaces and metadata may differ. Exercise unsupported-capability errors and handlers without routing metadata.

### M2: CLI resolver and isolated lazy provider construction (~420 LOC)

**Estimate:** 230 implementation + 190 tests; 1.5 days / 9 hours.
**Dependencies:** M1.
**Files:** `cmd/ailang/ai_handlers.go`, focused CLI routing tests, stub setup paths found by executor; reuse `internal/ai/factory` and `internal/modelreg` without duplicating their logic.
**Example coverage:** Test the config/direct/stub setup used by `examples/runnable/ai_cross_provider.ail` (M4).

Classify explicit prefixes, registered friendly names, GuessProvider-recognized names and unknown names in the approved order. Resolve config-driven providers through the existing registry. Attach routing on config, direct, config-driven direct and both stub paths. Lazily construct/cache handlers using each target's factory lanes, readyForCalls, attribution and policy options. Carry model token/credential settings explicitly per the clarification above.

- [ ] Classification tables cover all conflict-surface rows from the design, including prefix stripping and friendly api_name resolution.
- [ ] Config and direct bindings both route; stub and fixture-stub routing requires no real keys or network.
- [ ] A missing target key returns non-retryable ProviderNotFound naming the required environment variable; no foreign request is sent.
- [ ] Key-file secrets and foreign model env_var values never enter target factory options or logs; test using dummy credentials and local recording factories.
- [ ] Cache hits construct once; differing token budgets or credential sources remain isolated and propagate the declared model budget.
- [ ] Routing policy/attribution and Google ADC/API-key/local/config-driven lane behavior retain existing semantics; unsupported policy errors remain explicit.

**Risk:** Factory probes can introduce network into tests. Inject construction/probe seams and use local mocks; never require production credentials for automated acceptance.

### M3: Result image operations and stdlib exports (~330 LOC)

**Estimate:** 190 implementation + 140 tests; 1 day / 6 hours.
**Dependencies:** M1, M2.
**Files:** `internal/effects/ai_image.go`, `internal/builtins/ai_image.go`, `std/ai.ail`, existing image effect/builtin tests.
**Example coverage:** Tests underpin `examples/runnable/ai_image_base64_result.ail` (M4).

Register callImageBase64Result(prompt, options) and callImageBase64WithRefsResult(prompt, refs, options), their builtins and std exports. Reuse existing classification and Result construction helpers. Use options.model for routing; no optional third model argument. Trace both success and typed error paths. File-writing variants retain their existing FS effect and abort semantics.

- [ ] Both APIs type-check as Result[string, AIError] with only the AI effect, using a freshly built compiler.
- [ ] Both return Ok for successful image output and typed Err for provider/routing/schema/reference failures, preserving code/message/retryable.
- [ ] A consumer handles Err, performs another operation and exits successfully rather than host-aborting.
- [ ] Ref-conditioned requests retain reference data and empty references match the no-ref path.
- [ ] Existing image API signatures, effects and failure behavior stay compatible.

**Risk:** Error wrapping may erase typed codes. Assert the language-level Result record, not merely the returned Go error.

### M4: Runnable examples, docs and acceptance gates (~100 LOC)

**Estimate:** 100 examples/docs; 1 day / 6 hours including integration buffer.
**Dependencies:** M1, M2, M3.
**Files:** Create `examples/runnable/ai_cross_provider.ail`, `examples/runnable/ai_image_base64_result.ail`; add deterministic success/error image fixtures in the existing fixture layout. Update `docs/docs/guides/ai-effect.mdx`, `docs/docs/guides/ai-routing.md`, relevant `docs/LIMITATIONS.md` statements and the current changelog.

Read `ailang prompt` before authoring .ail. Build examples from working in-tree idioms; type-check and run with fresh compiler. Use fixture stubs for the handled failure example: generic --ai-stub success alone cannot prove Err continuation. If existing fixture infrastructure cannot inject that error, use a minimal test seam under current stub flags rather than inventing a new flag. Pair stub examples with recording-handler tests because one stub cannot prove actual provider selection.

- [ ] Both new examples type-check and execute offline with exit 0; the error fixture demonstrably prints the fallback and continues.
- [ ] Documentation explains options.model, both Result functions, missing-key error changes, direct-path resolution and credential isolation.
- [ ] make test, make fmt, make lint, make check-boundaries and make simplicity-audit pass; new failure regressions are resolved and baseline failures reported separately.
- [ ] Focused coverage report records exercised routing, credential, budget and Result branches; no arbitrary global percentage is claimed.
- [ ] Optional authorized live smoke records both provider directions and provider/model trace identity, or is explicitly recorded as not run when keys are unavailable.

**Risk:** Missing local tools/binary drift. Rebuild before language validation and retain deterministic offline evidence as the acceptance gate.

## Day-by-Day Execution

| Day | Work | Exit evidence |
|---|---|---|
| 1 | Provision Go, rebuild, bank affected-package tests and full baseline; settle route descriptor; start M1. | Baseline results and step routing tests. |
| 2 | Finish M1 image/metadata/budget cases; start M2 classification and factory seams. | M1 pass and resolver matrix. |
| 3 | Finish M2 credential isolation/cache/setup tests. | M2 pass with no external credentials. |
| 4 | Complete M3 effect/builtin/std Result API and typed failure tests. | M3 pass and checked consumer module. |
| 5 | Complete M4 fixtures/examples/docs and full gates; optional live smoke. | Verified examples and recorded acceptance report. |

Run focused `go test ./internal/effects ./internal/builtins ./internal/ai/... ./cmd/ailang` as the changed milestone warrants, and run the full gates at integration. Confirm make targets from the repository's included make files. Fix pre-existing environment blockers before claiming acceptance; tool provisioning time beyond the buffer requires re-estimation.

## Scope, Dependencies and Handoff

No new AI effect, CLI flag, capability or provider protocol. No automatic failover, speech-model routing, or Result wrappers for file-writing image calls. Consequently the text/portrait portion of AI.5 can move to std/ai; a fully OpenRouter-bound text/image/TTS service is not promised because speech remains bound. Removing downstream stapledons-godot code is outside this repository sprint.

Issues #1495, #1496, #1497 and #1500 are landed prerequisite/context references, not unresolved sprint issues to auto-close. Record them as related issues only. Link any actual routing/Result issue if the coordinator supplies one; none is supplied in the handoff.

On sprint-plan approval, coordinator handoff starts sprint-executor using the plan and progress JSON, then sprint-evaluator checks all acceptance criteria. No executor is dispatched by this planning stage. Preserve the approved design and mission log; plan review/merge is the next gate.
