# Sprint Plan: M-OLLAMA-EMBED-RESULT-AI-EFFECT

**Design:** [m-ollama-embed-result-ai-effect.md](m-ollama-embed-result-ai-effect.md)
**Status:** Planned; implementation awaits coordinator plan approval.
**Duration:** 3 engineering days, approximately 8 hours including 25% integration buffer.
**Risk:** Medium — effect relabel and runtime budget/error boundaries.

## Scope and approval

Deliver recoverable text/media embedding calls, public std/embedding wrappers, AI capability and budget enforcement, and offline stub/fixture coverage. The handoff explicitly approves prior design work, including D1 (IO→AI) and D2 (string Err); unchecked design freeze boxes are stale. No implementation occurs in this planning task. Coordinator plan approval/merge starts the executor through the existing pipeline; do not dispatch a duplicate execution task.

The design target v0.54.8 is historical: std/VERSION is v0.54.12. Use the next available release when executing; do not bump versions in this sprint. Frozen historical prompts and changelogs stay untouched. Use an unreleased changelog fragment rather than the older cumulative file named by the design.

## Current status and estimate evidence

Verified: only two IO embedding builtins exist, with direct Ollama requests; std/embedding lacks wrappers. The design audits both text and media, registry names, consumers and AI precedents, so this is a systemic fix. Current sizes: builtin file 208 lines, fixture loader 381, std module 144. New companion files keep growth away from the 800-line limit.

Seven-day velocity script found only the design-doc commit (#1802); the checkout has insufficient history and no usable LOC/day metrics. Do not claim measured throughput. Capacity is a conservative planning assumption: 1,000 changed/moved LOC over 3 days (about 334/day), including 420 test LOC; moved helper lines count toward review effort. Five milestones each fit a 60–90 minute executor run, with remaining time for integration/review.

## Registry reuse audit

Ran ailang pkg search embedding and ailang pkg search ollama; inspected pkg info and pkg docs for sunholo/email@0.9.0 and sunholo/ollama_stream@0.1.0. Email queries pre-indexed archives through DuckDB; ollama_stream implements chat streaming. Neither supplies language builtin authority, AI budget handling, embedding Result semantics, or fixture dispatch. All milestones use action none: extend existing core infrastructure, no registry dependency. Registry CLI warned binary may be stale; searches succeeded and are advisory, current repository code controls runtime planning.

## Milestones

### M1: Extract the Ollama request helper (~220 LOC)
**Estimate:** 140 implementation/docs + 80 tests; 60–90 minutes.
**Dependencies:** None
**Registry reuse:** none — existing core AI/Ollama path must enforce this behavior.

Move request and media encoding to internal/ai/ollama/embed.go returning []float64; preserve environment endpoint, 30/120 second timeouts and existing error prefixes. Move helper tests into internal/ai/ollama/embed_test.go; adapt existing live tests without requiring a live daemon.

**Acceptance criteria:**
- [x] HTTP test server verifies text/media payloads, empty input, empty response, server errors and float conversion offline.
- [x] Existing live dimensionality tests retain skip behavior when Ollama is unavailable.
- [x] go test ./internal/ai/ollama ./internal/builtins passes.

**Example coverage:** M5 supplies the runnable fallback and migrated text/media callers; earlier milestones use offline Go tests until wrappers land.

**Risk/mitigation:** Keep changes focused and run package-level offline regressions before dependent work.

### M2: Implement AI embedding ops and deterministic stubs (~290 LOC)
**Estimate:** 170 implementation/docs + 120 tests; 60–90 minutes.
**Dependencies:** M1
**Registry reuse:** none — existing core AI/Ollama path must enforce this behavior.

Create internal/effects/ai_embed.go and ai_embed_test.go. Register embedOllama/embedOllamaMedia, use the shared helper, and inspect resolved stub/fixture handlers without requiring a chat provider for real local calls. Add deterministic input-dependent eight-dimensional unit vectors and AI trace events.

**Acceptance criteria:**
- [x] effects.Call denies absent AI capability and zero budget before network access.
- [x] With AI and no chat handler, a fake Ollama server receives the call; an unrelated chat provider does not reroute it.
- [x] Stub vectors are repeatable, distinct across representative inputs, unit magnitude, and perform no network requests.
- [x] Text/media successes and failures emit bounded trace arguments without raw media bytes.

**Example coverage:** M5 supplies the runnable fallback and migrated text/media callers; earlier milestones use offline Go tests until wrappers land.

**Risk/mitigation:** Keep changes focused and run package-level offline regressions before dependent work.

### M3: Add embedding fixture replay (~180 LOC)
**Estimate:** 90 implementation/docs + 90 tests; 60–90 minutes.
**Dependencies:** M2
**Registry reuse:** none — existing core AI/Ollama path must enforce this behavior.

Extend internal/effects/ai_stub_fixtures.go with embed entries; put embed-only parsing/key/replay helpers in ai_stub_fixtures_embed.go and tests in ai_stub_fixtures_embed_test.go. Canonical compact sorted JSON uses model/text for text and audio/image presence booleans plus model/text for media. Require explicit media keys; no cross-op fallback. Validate numeric vectors and document the key and error format.

**Acceptance criteria:**
- [x] Embed success and error fixtures replay offline; missing fixture fails loudly.
- [x] Literal and sha256 lookup agree; text/media keys and empty media behavior are tested.
- [x] Existing text/image/speech fixture tests continue passing without format-version changes.

**Example coverage:** M5 supplies the runnable fallback and migrated text/media callers; earlier milestones use offline Go tests until wrappers land.

**Risk/mitigation:** Keep changes focused and run package-level offline regressions before dependent work.

### M4: Expose Result builtins and standard wrappers (~200 LOC)
**Estimate:** 100 implementation/docs + 100 tests; 60–90 minutes.
**Dependencies:** M2, M3
**Registry reuse:** none — existing core AI/Ollama path must enforce this behavior.

Update internal/builtins/ollama_embed.go for AI labeling and dispatch; add registrations and Result helpers in ollama_embed_result.go and focused tests in ollama_embed_result_test.go. Add four public wrappers to std/embedding.ail. Use existing ADT construction patterns. Capability/budget errors remain runtime errors; only provider/fixture failures become string Err values.

**Acceptance criteria:**
- [x] All four builtin registrations and type signatures require AI; wrappers type-check.
- [x] Text and media Result success yields Ok([float]); provider failure yields Err(string) with specified prefix; non-Result failures still abort.
- [x] End-to-end evaluator calls charge exactly one AI budget unit; capability and budget denial never become catchable Err.
- [x] Direct effects.Call budget tests and normal evaluator/builtin-scope tests cover distinct paths; avoid assuming direct Impl calls create a charge scope.

**Example coverage:** M5 supplies the runnable fallback and migrated text/media callers; earlier milestones use offline Go tests until wrappers land.

**Risk/mitigation:** Preserve authority failures outside Result wrapping and test real evaluator charge scopes.

### M5: Migrate examples and document graceful degradation (~110 LOC)
**Estimate:** 80 implementation/docs + 30 tests; 60–90 minutes.
**Dependencies:** M4
**Registry reuse:** none — existing core AI/Ollama path must enforce this behavior.

Migrate examples/reference/ollama_embed_test.ail and neural_semantic_search.ail to wrappers/AI rows; add examples/reference/ollama_embed_fallback.ail with success/error fixture files alongside it. Update semantic-search.md, semantic-caching-how-to.mdx, effects.md, std/embedding header, internal/builtins/sharedindex.go metadata and cmd/ailang/ai_handlers.go warning. Add changelogs/unreleased/2026-10-16-ollama-embed-result-ai-effect.md, opening with ### .

**Acceptance criteria:**
- [ ] Fallback example exits zero and demonstrates both success and Err branches using checked-in fixtures and --caps AI,IO --ai-stub.
- [ ] Legacy IO-only embedding row is rejected, IO-only capability cannot authorize AI entry, --caps auto follows the migrated row.
- [ ] CLI smoke verifies AI-only embedding works without --ai against a fake endpoint; live-down recovery uses a controlled unreachable loopback endpoint, not external service availability.
- [ ] make test-changed, make check-boundaries, make check-file-sizes, make check-changelog, make verify-stdlib and make verify-examples pass; report pre-existing failures separately.
- [ ] Format touched Go/AILANG files and run focused lint for touched packages; existing std/ai Result and speech stub regressions remain green.

**Example coverage:** Fallback example plus both migrated reference examples; live models are optional integration checks.

**Risk/mitigation:** Keep changes focused and run package-level offline regressions before dependent work.

## Day-by-day schedule

- Day 1: M1 and M2; prove network helper behavior and AI/stub dispatch independently.
- Day 2: M3 and M4; complete fixture replay and public runtime/std surface.
- Day 3: M5, focused regression gates, integration buffer and sprint-evaluator review.

## AILANG syntax and example gate

AILANG prompt version loaded: v0.16.7 (ailang prompt and ailang prompt --version-active). No new syntax is authored by this plan; executor must reload prompt before editing .ail files.

| Demo module | Contracts | Effects | Inline tests |
|---|---|---|---|
| ollama_embed_fallback.ail | skip: provider effects are validated through fixture replay rather than SMT | include: embedding helper ! {AI}, printing main ! {AI, IO} | include: pure Result-to-status formatter tests for Ok/Err |
| ollama_embed_test.ail | skip: vector inference depends on provider response | include: AI plus IO where printing is used | skip: existing integration demo; offline Go/fixture tests assert behavior |
| neural_semantic_search.ail | skip: provider and SharedIndex effects are not SMT contracts | include: AI, SharedIndex and IO wherever required by actual body | skip: stateful search demo; targeted regression run covers behavior |

## Validation and completion

Tests must prove all new dispatch/error branches, four builtin signatures, exact budget charge counts, and existing std/embedding pure codec/math behavior. No numerical coverage baseline was collected during planning; prioritize behavior assertions over a fabricated percentage. Use focused touched-package tests followed by make test-changed (includes test-core), boundary, size, changelog, stdlib and examples checks. CI owns the full suite; do not require make test in executor runs. Check available lint targets and use focused lint. Real Ollama dimensionality tests remain optional/skipped; CI uses fake servers and fixtures.

Error fixture payloads carry string messages; preserve supplied fixture messages verbatim, and preserve existing builtin prefixes on real transport errors. Resolve any design ambiguity about where Result is built by following existing ADT helpers without wrapping capability/budget failures. Stub selection must not accidentally force lazy chat provider initialization for a real embedding-only call.

Sprint JSON dependencies use exact milestone IDs. All passes/started/completed values begin null. Executor updates progress incrementally and finishes through sprint-evaluator. No additional package, provider abstraction, parser change, batch API or messaging embedder consolidation is in scope.

Execution deviation: the transport helper lives in `internal/ai/ollama/embed` to avoid the existing parent-package dependency through OpenAI and telemetry back to effects. No provider-layer refactor was needed.
