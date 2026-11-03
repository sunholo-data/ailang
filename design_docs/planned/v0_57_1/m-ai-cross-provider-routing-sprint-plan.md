# Sprint Plan: AI Cross-Provider Routing

Design: [m-ai-cross-provider-routing.md](m-ai-cross-provider-routing.md)

**Status:** Planned; implementation awaits sprint approval through the coordinator.
**Target:** v0.57.1. **Duration:** 5 engineering days, including approximately 25% uncertainty buffer. **Risk:** Medium.
**Estimate:** 1,160 LOC: 560 implementation/docs/example + 600 tests. Six bounded milestones, each targeting 60–90 minutes of concentrated executor work; remaining time covers investigation, builds, review and integration across sessions. Split a milestone before implementation if discovery exceeds that window.

## Scope and rulings

Implement per-call cross-provider step/image routing and the three-argument callImageBase64Result. Preserve the resolver contract, empty-model defaults, effect rows and grammar. Speech routing, new provider failover, OAuth/local alternates and other image Result APIs remain outside scope.

The coordinator handoff explicitly states “Previous work has been approved.” This records design sign-off for D1 (credential-gated billing surface) and D4 (three-argument API/precedence), satisfying the stale unchecked Design Freeze sign-off row. D2/D3/D5–D7 remain design rulings. There are no remaining human-decision gates; do not reopen these rulings. Approval of this plan is handled by the coordinator before executor dispatch, not self-approved here.

## Current implementation and velocity

Source version is v0.57.0; checkout HEAD d9be18f1. The history is shallow with only one visible commit; the seven-day velocity script reports no usable LOC metrics and N/A diff statistics. A measured LOC/day or historical completion rate cannot be claimed. Planning capacity is therefore an assumption: 232 LOC/day over five days, based on decomposition and the approved design's 4–5 day estimate.

Per-call image models, speech and fixture stubs already exist and are not reimplemented. Source confirms resolver-only dispatch and bound-handler metadata. Router propagation through lazy initialization and serving-handler metadata need explicit regression tests. Current principal file sizes: ai.go 506, ai_image.go 295, ai_lazy.go 65, ai_handlers.go 281, ai_prepare.go 219, builtin ai_image.go 248, std/ai.ail 427 lines. New companion files named below keep additions under the 800-line limit; put tests in separate ai_router_test.go, ai_alternates_test.go and ai_image_result_test.go rather than growing large existing tests.

## Registry reuse audit

Ran `ailang pkg search ai` and `ailang pkg search routing`. Inspected `sunholo/gemini_live@0.5.1` and `sunholo/decisions@0.4.0` using both pkg info and pkg docs. Gemini Live supplies protocol/TTS helpers; decisions is a Net/Env System One client explicitly outside AI budget accounting. Neither can implement Go host dispatch, credential factories or std/ai effect operations. All six milestones use **none**: extend existing compiler/runtime surfaces without new package dependencies. M6 uses existing std/ai and stub infrastructure; neither candidate substitutes for a mixed-provider AI-effect example.

## Milestones

### M1: Shared effects router (~200 LOC)

**Estimate:** 100 implementation/docs/example + 100 tests. **Executor budget:** 60–90 minutes.
**Dependencies:** none

- Add ModelRoute, ModelRouter and WithModelRouter in internal/effects/ai_router.go; retain ModelResolver contract.
- Route Step, StepWithCache and StepWithStream through one helper; empty model uses bound handler.
- Copy router in ai_lazy.go; track serving handler for LastRoutingMetadata and clear stale metadata after errors.

**Acceptance:**

- [ ] Fake handlers prove all three step variants select alternate; nil router retains existing behavior.
- [ ] Existing ModelNotAllowed resolver tests pass unmodified; lazy initialization retains routing.
- [ ] Alternate metadata is returned immediately after success; bound/default calls and failures cannot expose stale alternate metadata.

**Example coverage:** New `examples/runnable/ai_cross_provider.ail` in M6; existing caching/tool-loop examples exercise dispatch.

### M2: Credential lanes and model resolution (~230 LOC)

**Estimate:** 130 implementation/docs/example + 100 tests. **Executor budget:** 60–90 minutes.
**Dependencies:** M1

- Create cmd/ailang/ai_alternates.go and ai_alternates_test.go with injectable provider construction and credential probes.
- Snapshot env/ADC availability once; exclude Ollama, ChatGPT, Codex and config-driven alternates; never read bound key-file for alternates.
- Resolve models.yml, explicit prefixes, known OpenRouter vendor forms, then bound passthrough; enforce model-row max tokens and OpenRouter-only policy/attribution.

**Acceptance:**

- [ ] Table tests pin precedence, friendly/api names, prefixes, unknown passthrough and same-provider routing.
- [ ] Missing credential returns typed ModelNotAllowed with provider-specific actionable hint.
- [ ] Fake factory tests cover API keys, unauthenticated OPENAI_BASE_URL, ADC/no-ADC, key-file isolation and zero CheckConnection calls.

**Example coverage:** New `examples/runnable/ai_cross_provider.ail` in M6; existing caching/tool-loop examples exercise dispatch.

### M3: CLI setup and lazy integration (~170 LOC)

**Estimate:** 70 implementation/docs/example + 100 tests. **Executor budget:** 60–90 minutes.
**Dependencies:** M1, M2

- Attach the same builder/router from setupAIHandlerFromConfig, setupAIHandlerDirect and preparedRunAI.initialize.
- Keep lazy initialization lazy; freeze alternates at initialization without per-call environment probes.
- Announce available alternates once on stderr; preserve bound auth and stub behavior.

**Acceptance:**

- [ ] Tests exercise direct/config setup and lazy run initialization using fakes without credentials or network.
- [ ] Mutating environment after initialization does not change routing; no duplicate startup line on repeated calls.
- [ ] Bound key-file/no-ADC semantics and provider policy gates remain unchanged.

**Example coverage:** New `examples/runnable/ai_cross_provider.ail` in M6; existing caching/tool-loop examples exercise dispatch.

### M4: Shared image dispatch (~160 LOC)

**Estimate:** 70 implementation/docs/example + 90 tests. **Executor budget:** 60–90 minutes.
**Dependencies:** M1

- Create internal/effects/ai_image_router.go; route options-model through the shared router and substitute resolved API name.
- Dispatch callImage, callImageBase64 and reference-conditioned variants to the selected handler, preserving FS and capability checks.
- Keep speech outside cross-provider routing.

**Acceptance:**

- [ ] Two fake handlers prove options-model routing across all four existing image variants.
- [ ] Empty model and no-router resolver behavior remain compatible; unsupported alternate capability fails explicitly.
- [ ] Reference arguments, output path and serving-handler metadata are preserved; speech remains bound.

**Example coverage:** New `examples/runnable/ai_cross_provider.ail` in M6; existing image-generation and call-result examples provide regression coverage.

### M5: Result image operation and std export (~260 LOC)

**Estimate:** 120 implementation/docs/example + 140 tests. **Executor budget:** 60–90 minutes.
**Dependencies:** M3, M4

- Create internal/effects/ai_image_result.go and ai_image_result_test.go; register callImageBase64Result using existing Result/error/trace helpers.
- Add three-string builtin and metadata in internal/builtins/ai_image.go; export function beside callImageBase64 in std/ai.ail.
- Apply explicit model over options model over bound default, keeping existing non-Result operations unchanged.

**Acceptance:**

- [ ] Ok contains image JSON; typed provider errors preserve codes/retryability; 5xx becomes retryable Err and evaluation continues.
- [ ] Arg validation, malformed options and all three precedence cases are tested; new builtin enforces AI budget/capability.
- [ ] Stub and strict fixture paths work; evaluator and strict VM agree through host-op registry; trace names new op and serving route.

**Example coverage:** New `examples/runnable/ai_cross_provider.ail` in M6; existing image-generation and call-result examples provide regression coverage.

### M6: Consumer example and focused release checks (~140 LOC)

**Estimate:** 70 implementation/docs/example + 70 tests. **Executor budget:** 60–90 minutes.
**Dependencies:** M5

- Create examples/runnable/ai_cross_provider.ail with cheap text plus Gemini image Result handling and bounded retry; update examples manifest using existing tooling.
- Update docs/docs/guides/ai-routing.md, docs/docs/guides/ai-effect.mdx and docs/docs/reference/stdlib.md.
- Add changelogs/unreleased/2026-11-03-ai-cross-provider-routing.md opening with ###; document credential billing surface and excluded lanes.

**Acceptance:**

- [ ] New example type-checks and runs offline under stub; deterministic error fixture proves continuation after Err.
- [ ] Conflict-surface examples type-check and run with appropriate caps/entrypoints and fixtures; preserve speech and AI mode gate behavior.
- [ ] Focused validation below passes; optional live two-key trace check is recorded separately and never required for offline completion.

**Example coverage:** New `examples/runnable/ai_cross_provider.ail` in M6; existing image-generation and call-result examples provide regression coverage.

## Day-by-day schedule

- Day 1: M1; characterize resolver regressions, lazy lifecycle and metadata reset behavior.
- Day 2: M2; credential matrix and resolution-order table, with provider construction faked.
- Day 3: M3 and M4; setup integration and image variants; reserve time for factory/auth edge cases.
- Day 4: M5; effect/builtin/std export plus failure continuation and engine parity.
- Day 5: M6; docs, example, fixture checks and integration fixes. Buffer is included, not added to the estimate.

## AILANG syntax and example contract

AILANG prompt version loaded: v0.16.11 (`ailang prompt`, active version from `ailang prompt --version-active`). Prompt version is independent of repository version v0.57.0. CLI warns its binary may be stale: rebuild with `make build` before executing validation against changed source.

The new demo must import StepResult and AIError by name. `step` returns Result[StepResult, AIError], not Result[string, AIError]; correct the illustrative design's narrate annotation when producing the runnable example. Do not copy qualified type annotations or unverified syntax from the design.

| Demo checklist | Decision |
|---|---|
| contracts | skip: provider output is nondeterministic; no meaningful requires/ensures claim over generated text or images |
| effects | include: demo main() -> () ! {AI, IO}; AI helper wrappers explicitly ! {AI} |
| inline tests | include: pure retry-decision helper (retryable: bool, attemptsLeft: int) -> bool with tests for exhausted, retryable and permanent failure cases; run via existing inline-test tooling |

The std export is exactly callImageBase64Result(prompt: string, options: string, model: string) -> Result[string, AIError] ! {AI}; empty third argument permits options-model fallback. Demo retry must be bounded and handle Err without aborting. Stub smoke proves API/control flow; fake two-handler tests prove actual cross-provider selection (a single global stub does not).

## Validation and success metrics

Use `make test-changed` (touched Go packages plus test-core) after relevant milestones and targeted Go tests for effects, builtin registry, CLI setup and VM host-operation parity. Do not require the full Go suite in executor runs; CI owns it. No new repository-wide coverage threshold: cover every routing-table row, credential exclusion, model precedence, lazy path and Result branch with assertions.

At completion run `make fmt`, `make lint`, `make check-boundaries`, `make check-file-sizes`, `make check-changelog`, `make verify-stdlib` and `make verify-examples`. Update the example manifest with existing repository tooling. Record actual commands, outputs and any environment limitations rather than treating missing tools as passes.

Type-check and run the new demo plus design Conflict Surface programs: ai_caching, ai_tool_loop, ai_image_generation, ai_call_result, ai_call_speech (all examples/runnable), and examples/ai_modes.ail. Inspect each module's entrypoints/caps before running; use strict stub fixtures where its response shape demands them. Pin deterministic results; do not assume generic stub text satisfies tool/schema callers. Preserve mode markers/routing-policy gate behavior. Prove error continuation with a provider-error fixture using the existing fixture schema; a fixture miss must remain loud.

Optional attended live validation uses explicitly supplied OpenRouter/Gemini credentials and checks serving-provider metadata plus OpenRouter Broadcast traces. Offline tests must not discover real keys or send billable calls.

## Risks and execution notes

Auth discovery must not probe remote connectivity: exercise ADC detection through injectable factory boundaries. Capture credential presence at setup/lazy initialization once, and document that exact lifecycle. Do not silently suppress configuration errors as missing credentials; missing lanes become actionable typed refusals at use. A resolved known-model row supplies max output tokens; unknown models retain the existing default. Policies and attribution apply only to OpenRouter handlers.

For image calls check capability interfaces on the selected handler, including reference variants. Store routing metadata from the serving handler and reset it for later calls/errors to avoid stale cost attribution. Preserve bound-provider auth options and old resolver tests unchanged.

Only #1496 is linked as remaining implementation work. #1495, #1497, #1499 and #1500 are historical shipped context, not new closure targets. Implementation commits should use refs #1496; close only if the completed scope warrants it.

## Handoff

Machine progress: `.ailang/state/sprints/sprint_M-AI-CROSS-PROVIDER-ROUTING.json`. All passes begin null. The coordinator receives artifact paths from the planner's final response and dispatches execution after plan approval; do not start implementation or send a duplicate executor task from this planning stage.
