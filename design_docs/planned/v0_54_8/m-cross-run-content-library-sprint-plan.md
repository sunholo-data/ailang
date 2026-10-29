# Sprint Plan: M-CROSS-RUN-CONTENT-LIBRARY

## Summary
Make generated content survive process restarts and remain discoverable through configured-provider embeddings. Implements the approved [design](m-cross-run-content-library.md) for v0.54.8.

**Status:** Planned; design approved by task-70cf8428 handoff. Sprint execution awaits coordinator plan approval.
**Duration:** 10 working days over two weeks; 18 hours milestone work plus 6 hours integration/uncertainty buffer = 24 hours. Each milestone targets one 60–90 minute executor run; stop and split rather than overflow it.
**Risk:** Medium, chiefly persistent-store error propagation and provider setup.
**Estimated LOC:** 1990, including implementation, tests, examples and docs.

## Current status and estimate basis
The clean checkout at 6c1ac816 contains the approved design but no implementation. SharedMem/SharedIndex setup remains in-memory; SharedIndex interface operations currently have no error returns. SQLite SharedCache, AI Result helpers and messaging embedders already exist. ANN, read-only brain/storage capabilities, decay helpers and migrations remain future work.

The velocity script was run for seven days. This checkout exposes only one recent design-doc commit, no usable diff history, and no recent LOC metrics; historical foundation changelog figures are not a measured current velocity. Planning assumption: 199 LOC per working day at 2.4 focused hours/day, with 33% time buffer over the 18-hour milestone subtotal. The design estimated 30 hours; reuse yields a provisional 24-hour estimate, with 30 hours as the conservative ceiling. Re-estimate after M3. No coverage baseline is claimed; measure changed-package coverage during execution and cover every new error/lifecycle branch rather than promise a repository-wide percentage.

## Registry reuse audit
Ran `ailang pkg search embeddings`, `sharedmem`, and `semantic`. `sharedmem` returned no candidates. Inspected `sunholo/email@0.9.0` with `pkg info` and `pkg docs`: domain-specific DuckDB + local Ollama tooling. Inspected `world/core@0.1.1`: pure world graph types/contracts, with no AGENT.md (`pkg docs` returned an explicit error). Neither replaces compiler/runtime capability wiring. All M1–M12 decisions are **none** for external packages; reuse existing internal implementations. The same audit is recorded per milestone in JSON.

## Execution constraints and design clarifications
- Keep core dependent only on abstract seams; cgo/SQLite stays in platform packages and provider resolution in CLI wiring.
- SharedIndex has void writes and slice-returning reads. M2 must add a narrow optional error-reporting adapter checked by effect handlers, or use an established equivalent, so DB failures become runtime errors. Never translate SQL failures into success/empty results. If this requires broad interface semantics changes, return to design review before implementing that expansion.
- Strict bounded scans select candidates in deterministic key order before scoring; compare with the existing contract and pin tested behavior. Float32 storage is lossy: use tolerance for backend parity, exact equality only for repeated cached stored vectors. Mixed-model namespaces remain operator-controlled; the approved index schema has no embed_model column despite a risk-table claim. Do not invent automatic migration or model metadata this sprint.
- Support real two-process sharing with WAL and timeout tests. The design's later “one run at a time” note is conservative guidance, not a substitute for its cross-process requirement.
- `internal/runner/run.go` is 730 lines and platform `sqlite.go` 702. Put storage setup in `handlers_shared_storage.go` and new index/cache files; use `run_shared_storage.go` for additional runner helpers if needed. Keep each source and test file under 800 lines.
- Existing binary is flagged stale; loaded teaching prompt active version is v0.16.7 while std/VERSION is v0.54.7. Executor rebuilds from checkout before validating new APIs; version changes follow repository lockstep policy and do not create releases.

## Milestones

### M1: Persistent SharedMem wiring (~180 LOC)

**Estimate:** 100 implementation/docs/example + 80 tests; 1.5 hours.
**Dependencies:** None
**Files:** internal/runner/handlers_shared_storage.go; internal/runner/run.go; internal/runner/batch.go; cmd/ailang/main_run.go; cmd/ailang/serve_api.go; internal/runner/handlers_shared_storage_test.go.
**Task:** Add DB options and flag/env precedence, reuse OpenSharedCache, propagate setup errors and own connection cleanup across run, batch and serve-api.
**Example coverage:** Cross-run showcase in M11; existing sharedmem_cache.ail and stdlib_sharedindex.ail.
**Risk/mitigation:** SQLite lifecycle or contract drift: use temporary-file differential and failure tests.

- [ ] Flag overrides AILANG_SHAREDMEM_DB; missing SharedMem capability and failed DB open are fatal.
- [ ] No path preserves in-memory behavior; resources close on success, setup failure and runtime failure.
- [ ] Sequential subprocess put/get and concurrent CAS tests show durable cross-process state.
- [ ] make test-changed passes for touched packages plus test-core.

### M2: SharedIndex opener and SQLite writes (~210 LOC)

**Estimate:** 140 implementation/docs/example + 70 tests; 1.5 hours.
**Dependencies:** None
**Files:** internal/effects/sharedindex_backend.go; internal/platform/sharedmem/sqlite_sharedindex.go; internal/platform/sharedmem/sqlite_sharedindex_test.go; cmd/ailang/platform_init.go.
**Task:** Register a core opener and create namespace-key schema via sqliteopen. Implement upsert, embedding upsert, delete, count, namespaces and Close.
**Example coverage:** Cross-run showcase in M11; existing sharedmem_cache.ail and stdlib_sharedindex.ail.
**Risk/mitigation:** SQLite lifecycle or contract drift: use temporary-file differential and failure tests.

- [ ] Namespaces and files isolate entries; duplicate keys in different namespaces coexist.
- [ ] Reopening preserves metadata, signed SimHash bits and float32 encoded embeddings.
- [ ] Storage operation failures surface through the effect boundary; no empty-hit or successful-write disguise.
- [ ] make test-changed passes for touched packages plus test-core.

### M3: SharedIndex retrieval parity (~230 LOC)

**Estimate:** 130 implementation/docs/example + 100 tests; 1.5 hours.
**Dependencies:** M2
**Files:** internal/platform/sharedmem/sqlite_sharedindex_search.go; internal/platform/sharedmem/sqlite_sharedindex_search_test.go.
**Task:** Implement bounded namespace scans, SimHash and cosine ranking in Go using existing helpers; compare against in-memory fixtures.
**Example coverage:** Cross-run showcase in M11; existing sharedmem_cache.ail and stdlib_sharedindex.ail.
**Risk/mitigation:** SQLite lifecycle or contract drift: use temporary-file differential and failure tests.

- [ ] Strict order is score DESC then key ASC, including tied scores and bounded maxScan.
- [ ] topK and maxScan match existing contract; no-hit results are empty slices.
- [ ] Dimension mismatch, corrupt blobs, delete and embedding replacement have explicit tested behavior.
- [ ] make test-changed passes for touched packages plus test-core.

### M4: Persistent SharedIndex runner integration (~160 LOC)

**Estimate:** 80 implementation/docs/example + 80 tests; 1.5 hours.
**Dependencies:** M1, M3
**Files:** internal/runner/handlers_shared_storage.go; cmd/ailang/main_run.go; internal/runner/run.go; internal/runner/batch.go; cmd/ailang/serve_api.go; cmd/ailang/shared_storage_integration_test.go.
**Task:** Wire sharedindex flag/env and fatal cap gate through all setup callers; test the shared-file shape and preserve serve-api defaults.
**Example coverage:** Cross-run showcase in M11; existing sharedmem_cache.ail and stdlib_sharedindex.ail.
**Risk/mitigation:** SQLite lifecycle or contract drift: use temporary-file differential and failure tests.

- [ ] Flag overrides AILANG_SHAREDINDEX_DB; missing SharedIndex capability is fatal.
- [ ] Separate subprocesses upsert then find by SimHash and embedding across restart.
- [ ] SharedMem and SharedIndex coexist in one file; two files and namespaces remain isolated; serve-api SharedMem regression passes.
- [ ] make test-changed passes for touched packages plus test-core.

### M5: Typed AI embedding effect (~190 LOC)

**Estimate:** 100 implementation/docs/example + 90 tests; 1.5 hours.
**Dependencies:** None
**Files:** internal/effects/ai_embed.go; internal/effects/ai_embed_test.go; internal/effects/ai.go; internal/effects/ai_lazy.go.
**Task:** Add WithEmbedder and AI embed op, Result construction, classified errors and effect accounting; ensure lazy context initialization works.
**Example coverage:** Cross-run showcase in M11; cross_run_library.ail seed/query and existing legacy Ollama fixture.
**Risk/mitigation:** Configuration or integration drift: use deterministic stubs, counted providers and focused regression tests.

- [ ] Nil embedder returns ProviderNotFound; provider timeout/rate-limit errors preserve typed retryability.
- [ ] Each invocation including cache hits records AI embed accounting and respects existing AI budgets.
- [ ] Lazy and eager contexts produce the same embedding Result and malformed input handling.
- [ ] make test-changed passes for touched packages plus test-core.

### M6: Embedding builtin and stdlib surface (~160 LOC)

**Estimate:** 100 implementation/docs/example + 60 tests; 1.5 hours.
**Dependencies:** M5
**Files:** internal/builtins/ai_embed.go; internal/builtins/ai_embed_test.go; std/ai.ail; .stdlib-golden/; std/VERSION.
**Task:** Mirror _ai_call_result registration and metadata; export embed(string) returning Result[[float], AIError] with AI effect. Update approved stdlib interface goldens using existing tooling.
**Example coverage:** Cross-run showcase in M11; cross_run_library.ail seed/query and existing legacy Ollama fixture.
**Risk/mitigation:** Configuration or integration drift: use deterministic stubs, counted providers and focused regression tests.

- [ ] Builtin type and metadata agree with std/ai.embed; compiler checks import and AI effect requirement.
- [ ] Legacy _ollama_embed IO surface remains runnable with existing fixtures.
- [ ] make verify-stdlib passes after intentional golden refresh; follow repository version policy without independent release/tag actions.
- [ ] make test-changed passes for touched packages plus test-core.

### M7: Configured provider resolution (~200 LOC)

**Estimate:** 100 implementation/docs/example + 100 tests; 1.5 hours.
**Dependencies:** M5
**Files:** cmd/ailang/ai_embed_handlers.go; cmd/ailang/ai_embed_handlers_test.go; cmd/ailang/ai_handlers.go; internal/config/embed.go.
**Task:** Reuse messaging embedders: explicit embeddings config first, otherwise bound Gemini/OpenAI/Ollama; deterministic explicit ai-stub. Wire eager and lazy provider creation.
**Example coverage:** Cross-run showcase in M11; cross_run_library.ail seed/query and existing legacy Ollama fixture.
**Risk/mitigation:** Configuration or integration drift: use deterministic stubs, counted providers and focused regression tests.

- [ ] Explicit config wins; invalid explicit config surfaces an error and never silently derives.
- [ ] Gemini/OpenAI mock transports embed without Ollama; derived provider/model notice appears once.
- [ ] OpenRouter/Anthropic/configdriven do not implicitly derive; unconfigured embed returns ProviderNotFound.
- [ ] ai-stub yields stable dimensions/vectors and requires no credentials.
- [ ] make test-changed passes for touched packages plus test-core.

### M8: In-process embedding cache (~150 LOC)

**Estimate:** 80 implementation/docs/example + 70 tests; 1.5 hours.
**Dependencies:** M5
**Files:** internal/effects/embed_cache.go; internal/effects/embed_cache_test.go.
**Task:** Decorate the existing effects.Embedder interface with a thread-safe map keyed by model plus SHA256(text), including batch behavior and defensive copies.
**Example coverage:** Cross-run showcase in M11; cross_run_library.ail seed/query and existing legacy Ollama fixture.
**Risk/mitigation:** Configuration or integration drift: use deterministic stubs, counted providers and focused regression tests.

- [ ] Repeated model/text invokes provider once; changed model/text misses; failures are not cached.
- [ ] Caller mutation cannot corrupt cached vectors; EmbedBatch preserves order and caches individual successes.
- [ ] Concurrent access is race-free; returned dimension and model metadata match wrapped embedder.
- [ ] make test-changed passes for touched packages plus test-core.

### M9: SQLite embedding cache (~180 LOC)

**Estimate:** 100 implementation/docs/example + 80 tests; 1.5 hours.
**Dependencies:** M8
**Files:** internal/platform/sharedmem/embed_cache.go; internal/platform/sharedmem/embed_cache_test.go; internal/effects/embed_cache_backend.go; cmd/ailang/platform_init.go.
**Task:** Add optional persistent cache seam and dedicated embed_cache table via sqliteopen; store vectors with dimension metadata and close resources.
**Example coverage:** Cross-run showcase in M11; cross_run_library.ail seed/query and existing legacy Ollama fixture.
**Risk/mitigation:** SQLite lifecycle or contract drift: use temporary-file differential and failure tests.

- [ ] Reopened cache returns stored model/text vector with zero provider calls.
- [ ] Cache rows never appear in brain_frames or SharedMem keys; all three tables coexist.
- [ ] Corrupt/dimension-invalid rows and storage failures are surfaced explicitly; no silent persistent-store fallback.
- [ ] make test-changed passes for touched packages plus test-core.

### M10: Embedding cache CLI integration (~130 LOC)

**Estimate:** 60 implementation/docs/example + 70 tests; 1.5 hours.
**Dependencies:** M7, M9
**Files:** cmd/ailang/main_run.go; cmd/ailang/ai_embed_handlers.go; cmd/ailang/ai_embed_cache_integration_test.go; internal/runner/run.go.
**Task:** Wire embed-cache-db and AILANG_EMBED_CACHE_DB, flag precedence, AI capability validation and cleanup into embedding setup.
**Example coverage:** Cross-run showcase in M11; cross_run_library.ail seed/query and existing legacy Ollama fixture.
**Risk/mitigation:** Configuration or integration drift: use deterministic stubs, counted providers and focused regression tests.

- [ ] Flag beats env; path without AI capability or unopenable path fails loudly.
- [ ] Persistent cache remains opt-in; in-process caching is enabled for configured embedders.
- [ ] Restart test with counted mock provider proves persistent hits; effect budget still counts each call.
- [ ] make test-changed passes for touched packages plus test-core.

### M11: Cross-run showcase and regression fixtures (~120 LOC)

**Estimate:** 60 implementation/docs/example + 60 tests; 1.5 hours.
**Dependencies:** M4, M6, M10
**Files:** examples/reference/cross_run_library.ail; examples/manifest.json; cmd/ailang/cross_run_library_test.go.
**Task:** Create seed and query exported entries: store full content in SharedMem, embed/index under a namespace, then retrieve payloads in a second run. Use deterministic stub and fixed timestamps for automation.
**Example coverage:** Cross-run showcase in M11; cross_run_library.ail seed/query and existing legacy Ollama fixture.
**Risk/mitigation:** Configuration or integration drift: use deterministic stubs, counted providers and focused regression tests.

- [ ] ailang check and keyless ai-stub seed/query runs pass using one temporary DB for all flags.
- [ ] Process B retrieves process A content; fresh-file control returns no match.
- [ ] Manifest module metadata is accurate; existing SharedMem and SharedIndex runnable examples pass.
- [ ] make test-changed passes for touched packages plus test-core.

### M12: Operator documentation and final gates (~80 LOC)

**Estimate:** 60 implementation/docs/example + 20 tests; 1.5 hours.
**Dependencies:** M11
**Files:** docs/docs/guides/semantic-caching-how-to.mdx; cmd/ailang/help.go; internal/config/embed.go; changelogs/unreleased/2026-10-13-cross-run-content-library.md.
**Task:** Correct default persistence claim; document three flags/envs, explicit provider setup, model pinning, namespace/file isolation and two-run commands. Add changelog fragment opening with ###.
**Example coverage:** Cross-run showcase in M11; cross_run_library.ail seed/query and existing legacy Ollama fixture.
**Risk/mitigation:** Configuration or integration drift: use deterministic stubs, counted providers and focused regression tests.

- [ ] Guide distinguishes in-memory defaults from opt-in durable stores and warns against CLI brain DB key collisions.
- [ ] OpenRouter users are told to configure a supported embedding provider separately; ANN and read-only capability remain deferred.
- [ ] Focused final validation and file-size, changelog, boundary, CLI-doc, stdlib and example gates pass.
- [ ] make test-changed passes for touched packages plus test-core.

## Day-by-day schedule

| Day | Work | Focus |
|---|---|---|
| 1 | M1 | SharedMem durability and cleanup |
| 2 | M2 | Index schema and writes |
| 3 | M3 | Search parity; re-estimate remaining work |
| 4 | M4 | Cross-process index and serve-api regression |
| 5 | M5 | Typed effect and accounting |
| 6 | M6, M7 | Public surface and provider resolution |
| 7 | M8 | In-process cache |
| 8 | M9, M10 | Persistent cache and CLI wiring |
| 9 | M11 | Keyless cross-run demonstration |
| 10 | M12 | Documentation and final validation |

Six hours of buffer spread across the schedule covers SQLite error adaptation, lifecycle integration and test failures. Days 6 and 8 can spill into buffer; milestone ordering is sequential even where dependencies permit independence.

## AILANG syntax and showcase gate

AILANG prompt version loaded: v0.16.7 (`ailang prompt` and `ailang prompt --version-active` run during planning). Rebuild and reload before implementation to resolve stale-binary warning.

| Module | Contracts | Effects | Inline tests |
|---|---|---|---|
| std/ai.ail embed wrapper | skip: external embedding cannot be proved by Z3 | include: embed(string) -> Result[[float], AIError] ! {AI} | skip: provider Results verified through Go effect tests |
| examples/reference/cross_run_library.ail | include: pure fixed timestamp helper -> int ! {}, ensures result >= 0; prove via ailang verify | include: seed/query entries -> () ! {IO, AI, SharedMem, SharedIndex} | include: fixed timestamp helper with input/output pairs; ailang test |

Example avoids time/network dependencies with ai-stub and a constant timestamp. Document real Gemini/OpenAI setup separately; credentials and network calls are optional operator smoke tests. No Ollama installation is required for mandatory validation.

## Validation and success metrics
Every milestone runs formatting on touched Go files and `make test-changed` (touched packages plus test-core). Confirm TEST_CHANGED_BASE resolves; use the starting commit if origin/dev is unavailable. Targeted tests cover subprocess persistence, concurrent CAS, namespace/file isolation, strict ordering/maxScan, corrupt data, typed errors, lazy initialization, provider precedence, caching and cleanup. Use changed-package coverage to inspect untested new branches; use race testing on new cache concurrency tests only.

Final executor gates: `make test-changed`, `make check-boundaries`, `make check-file-sizes`, `make check-changelog`, `make check-cli-docs`, `make verify-stdlib`, and `make verify-examples`; run scoped lint/vet on touched packages. Explicitly check, test and verify cross_run_library.ail and run seed then query in separate processes against a temp DB. The reference showcase is exercised explicitly because the standard runnable-example gate may not execute it. Retain existing sharedmem_cache, semantic_retrieval and stdlib_sharedindex fixtures; type-check neural_semantic_search without requiring its optional Ollama service. CI owns the full test suite; this overrides the design's broad local make-test requirement per current planner guidance.

Add `changelogs/unreleased/2026-10-13-cross-run-content-library.md` beginning with `### `. Update existing help/env documentation sources and refresh intentional stdlib goldens with established tools. No repository-wide coverage gate is introduced for this feature.

## Handoff

Progress JSON: `.ailang/state/sprints/sprint_M-CROSS-RUN-CONTENT-LIBRARY.json`. All passes/started/completed fields begin null, with complete acceptance criteria and registry decisions. No GitHub issue was identified; #1800 is the landed design PR, not an issue to auto-close. The coordinator publishes artifacts for plan review; merging the plan PR triggers sprint-executor per the sprint-planner coordinator workflow. Do not self-approve or start implementation in this planning stage.
