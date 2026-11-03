# Sprint Plan: M-PERSISTENT-SEMANTIC-STORE

**Design:** [Approved design](m-persistent-semantic-store.md)  
**Status:** Planned; implementation awaits sprint approval/coordinator merge.  
**Target:** v0.57.1  
**Duration:** 5 working days / 20 focused hours, including 5 hours contingency.  
**Risk:** Medium; SQLite parity, configuration provenance, TTL and handle lifetime.  
**Task:** task-d57f6f2f; design handoff task-d812326a. Work stays on current coordinator branch.

## Goal and scope

Persist SharedMem and SharedIndex across runs, expose provider-configured std/ai.embed with typed failures, and optionally cache embeddings in the SharedMem DB. Preserve in-memory defaults, namespaces, Strict ordering and capability enforcement. ANN, read-only Brain, remote stores and batch embedding are deferred by design D7.

## Evidence and estimates

Working tree was clean. Current source std version is v0.57.0. The 7-day velocity script found only b5f9b89d (2026-11-03, documentation) and no usable LOC metrics: this checkout cannot establish historical implementation velocity. Capacity below is a provisional engineering estimate, not a measured rate. Ten 60–90 minute executor milestones total 15 hours plus 5 hours contingency; five four-hour days. The design has inconsistent 5–6 day and 40-hour/two-week estimates; retain 40 hours as a conservative outer budget if integration exceeds contingency.

Verified seams: existing effects.Embedder and BrainCache interfaces, messaging provider factory, SQLite shared cache. Configuration loader defaults to Ollama and suppresses some load errors: simply calling it cannot meet the design's nil-unconfigured/no-silent-fallback requirements. M5 must distinguish explicit configuration and surface invalid requested config. BrainCache.PutVector has no TTL parameter: M8 must use the existing frame expiration path or a narrow adapter, without schema duplication. Expiry must be checked on lookup, since GC-on-open alone cannot expire a long-running cache.

File sizes: runner/run.go 738, effects/context.go 759, platform/sharedmem/sqlite.go 702 lines. Keep additions in handlers_persistent.go, handlers_embed.go, ai_embed.go, ai_embed_cache.go, sqlite_index.go and sqlite_index_search.go; keep edits to the large existing files minimal and below 800 lines. No compiler pass or syntax change is planned.

## Review decisions and terminal executor gates

Resolved D1–D7 are rulings, not approval gates: reuse registered opener, separate index table, effects.Embedder seam, AI Result error surface, opt-in cache in SharedMem DB, loud failures and deferred ANN/Brain.

Two decisions remain OPEN in the approved design's **Deferred Decisions** section:

1. Before M4: whether new flags are policy-overridable or operator-only (run_policy.go surface). STOP: commit current work, report `BLOCKED: persistence/embedding flag policy ruling required`, and end the run if no ruling has been recorded.
2. Before M5: OpenRouter embeddings support via base URL or provider case versus explicit deferral. STOP: commit current work, report `BLOCKED: OpenRouter embedding scope ruling required`, and end the run if no ruling has been recorded. Any endpoint compatibility research belongs to this review; this plan makes no unsupported endpoint claim. Gemini/OpenAI/Ollama are the approved existing provider paths.

Record review rulings here and in JSON before releasing the affected milestones. Do not silently assume that approval of the design resolved its explicitly deferred questions.

## Registry reuse audit

Executed `ailang pkg search semantic`, `embedding`, and `sqlite`. Results: sunholo/email@0.10.0 and world/core@0.1.1 for semantic; email for embedding; none for sqlite. Inspected both with pkg info and pkg docs (world/core has no AGENT.md). Email is a DuckDB email query layer requiring Ollama; world/core is pure graph semantics. Neither implements host effect persistence or provider wiring. Every milestone's decision is **none** for new registry dependencies; reuse existing Go backends, interfaces and factories. Per-milestone decisions and search evidence are recorded in JSON.

## AILANG syntax and example gate

AILANG prompt version loaded: v0.16.11 (actual `ailang prompt --version-active` output). Loaded full `ailang prompt`; CLI warns it may be stale relative to source v0.57.0. Executor must build/use the source-matching CLI before syntax validation and record the refreshed prompt version; do not infer prompt version from std/VERSION.

| Showcase module | Contracts | Effects | Inline tests |
|---|---|---|---|
| examples/reference/persistent_semantic_store.ail | skip: external provider/store main cannot be proved as a pure contract | include: main() -> () ! {AI, IO, SharedMem, SharedIndex} | skip: deterministic stubbed integration tests cover external effects |
| examples/reference/persistent_semantic_query.ail | skip: external retrieval depends on persistent state | include: main() -> () ! {AI, IO, SharedMem, SharedIndex} | skip: cross-process fixtures assert retrieval results |

M1–M6 exercise existing examples/reference/sharedmem_cache.ail and examples/runnable/stdlib_sharedindex.ail as relevant. M7–M10 build the new showcase modules incrementally and preserve examples/reference/neural_semantic_search.ail. Showcase source is written only during execution after loading the matching prompt.

## Milestones

### M1: Persistent index opener and storage

**Estimate:** 180 implementation/docs + 100 tests = 280 LOC; 90 minutes. **Dependencies:** none.

**Files:** `internal/effects/sharedindex_backend.go`, `internal/platform/sharedmem/sqlite_index.go`, `sqlite_index_test.go`.

Register the opener at platform startup; use sqliteopen WAL/busy-timeout recipe and a separate sharedindex_entries table. Implement mutation, count and namespace methods, preserving signed simhash bits and atomic upserts.

**Acceptance:**

- [ ] Reopen preserves entries and namespace isolation; delete/count/namespaces match memory backend
- [ ] Existing brain_frames data remains readable; missing opener and failed open produce explicit errors

**Validation:** focused tests for the touched packages via `make test-changed`; no live provider required.

### M2: Persistent similarity queries

**Estimate:** 140 implementation/docs + 150 tests = 290 LOC; 90 minutes. **Dependencies:** M1.

**Files:** `internal/platform/sharedmem/sqlite_index_search.go`, `sqlite_index_search_test.go`.

Implement both query methods with lexicographic maxScan prefix and score DESC/key ASC sorting; reuse simhash.Cosine and existing encoders.

**Acceptance:**

- [ ] Differential fixtures match memory backend for ties, empty data, topK/maxScan boundaries and dimension mismatch
- [ ] Two independently opened connections share committed writes without namespace collisions

**Validation:** focused tests for the touched packages via `make test-changed`; no live provider required.

### M3: Runner store options and resource ownership

**Estimate:** 110 implementation/docs + 100 tests = 210 LOC; 90 minutes. **Dependencies:** M2.

**Files:** `internal/runner/handlers_persistent.go`, `handlers_persistent_test.go`, `internal/runner/run.go`, `batch.go`, `cmd/ailang/serve_api.go`.

Add DB options and setup error propagation; track ownership and close opened handles on normal exit, setup failure and execution failure; update every setup callsite.

**Acceptance:**

- [ ] No-path options keep existing in-memory behavior
- [ ] Two Run invocations share frames and index entries; same-file and separate-file modes work
- [ ] Batch/API contexts do not leak handles or close caller-owned resources

**Validation:** focused tests for the touched packages via `make test-changed`; no live provider required.

### M4: Persistence CLI and environment surface

**Estimate:** 90 implementation/docs + 90 tests = 180 LOC; 90 minutes. **Dependencies:** M3.

**Files:** `cmd/ailang/main_run.go`, `main_run_persistence_test.go`, `internal/config/shared_store.go`, `cmd/ailang/run_policy.go`.

Add sharedmem-db/sharedindex-db with flag-over-env precedence and capability validation; apply reviewed policy ruling.

**Acceptance:**

- [ ] Each path without its corresponding cap fails before execution; bad paths fail loudly
- [ ] Explicit flags override environment; policy flag inventory/hardening tests cover new flags

**Validation:** focused tests for the touched packages via `make test-changed`; no live provider required.

### M5: Provider-configured embedder setup

**Estimate:** 110 implementation/docs + 100 tests = 210 LOC; 90 minutes. **Dependencies:** M4.

**Files:** `internal/runner/handlers_embed.go`, `handlers_embed_test.go`, `internal/effects/context.go`, `cmd/ailang/main_run.go`.

Add EffContext Embedder and clone propagation; embed-provider/embed-model flags; reuse messaging factory outside core effects. Distinguish explicit config from the factory default Ollama provider.

**Acceptance:**

- [ ] No explicit embedding config leaves nil embedder without network calls or warnings
- [ ] Explicit unsupported provider/missing key fails startup; flag/env/YAML precedence tested
- [ ] Configured Gemini/OpenAI/Ollama select the requested model; reviewed OpenRouter ruling is reflected

**Validation:** focused tests for the touched packages via `make test-changed`; no live provider required.

### M6: Typed AI embed operation

**Estimate:** 110 implementation/docs + 120 tests = 230 LOC; 90 minutes. **Dependencies:** M5.

**Files:** `internal/effects/ai_embed.go`, `ai_embed_test.go`.

Register AI/embed through effects.Call and RequireCapWithBudget; classify transport failures into existing AIError records.

**Acceptance:**

- [ ] Nil embedder returns ProviderNotFound with retryable=false; stub vector succeeds
- [ ] Authentication, rate limit and timeout errors map correctly; capability and budget denial cannot call provider
- [ ] Trace and budget behavior matches existing AI operation precedent

**Validation:** focused tests for the touched packages via `make test-changed`; no live provider required.

### M7: Builtin and stdlib embedding export

**Estimate:** 90 implementation/docs + 100 tests = 190 LOC; 90 minutes. **Dependencies:** M6.

**Files:** `internal/builtins/ai_embed.go`, `ai_embed_test.go`, `std/ai.ail`.

Register _ai_embed with Result[[float], AIError] and export embed; convert float32 vectors to runtime float values without changing existing call APIs.

**Acceptance:**

- [ ] Import std/ai (embed) type-checks and returns Ok vector or typed Err with stub configuration
- [ ] Evaluator and strict VM produce equal values/errors for stub cases
- [ ] Legacy _ollama_embed and existing AI exports retain their behavior

**Validation:** focused tests for the touched packages via `make test-changed`; no live provider required.

### M8: Content-hash cache and expiry

**Estimate:** 140 implementation/docs + 120 tests = 260 LOC; 90 minutes. **Dependencies:** M7.

**Files:** `internal/effects/ai_embed_cache.go`, `ai_embed_cache_test.go`, `internal/runner/handlers_embed.go`, `cmd/ailang/main_run.go`.

Add embed-cache/embed-cache-ttl, Go-duration parsing and prerequisite validation. Cache in embed-cache namespace using sha256(resolved model identity + NUL + text); reuse BrainCache storage and existing expiration columns.

**Acceptance:**

- [ ] Repeated text and cross-run reuse call provider once; changed model misses cache
- [ ] Hit returns identical vector and still follows AI cap/budget gate; failures are never cached
- [ ] Expired rows are rejected at lookup as well as GC-on-open; enabled cache without persistent SharedMem and invalid TTL fail explicitly
- [ ] Cache entries never overwrite unrelated namespaces; cache errors surface explicitly

**Validation:** focused tests for the touched packages via `make test-changed`; no live provider required.

### M9: Process integration and contention

**Estimate:** 30 implementation/docs + 150 tests = 180 LOC; 90 minutes. **Dependencies:** M8.

**Files:** `internal/runner/persistent_store_integration_test.go`, `internal/platform/sharedmem/sqlite_index_concurrency_test.go`.

Use subprocesses with stub embeddings to prove real restart/process sharing; cover startup cleanup, lock contention, failed provider and TTL.

**Acceptance:**

- [ ] Writer process exits, reader retrieves memory and both index query forms
- [ ] Concurrent connections preserve committed state; WAL/busy-timeout and transaction behavior are exercised
- [ ] No credentials or running Ollama required for automated integration tests

**Validation:** focused tests for the touched packages via `make test-changed`; no live provider required.

### M10: Examples, guide and release fragment

**Estimate:** 110 implementation/docs + 20 tests = 130 LOC; 90 minutes. **Dependencies:** M9.

**Files:** `examples/reference/persistent_semantic_store.ail`, `examples/reference/persistent_semantic_query.ail`, `docs/docs/guides/semantic-caching-how-to.mdx`, `cmd/ailang/help.go`, `cmd/ailang/doctor_memory.go`, `changelogs/unreleased/2026-11-03-persistent-semantic-store.md`.

Correct persistence claims and document all six flags, two path env vars, existing embed config, TTL, caps, linear scans and two-run commands. Add changelog fragment opening with ### .

**Acceptance:**

- [ ] Both new examples check and run through the stub integration harness against one DB
- [ ] Help and guide match shipped flag defaults/prerequisites; ANN/read-only brain remain deferred
- [ ] make test-changed, check-file-sizes, check-changelog, verify-stdlib, verify-examples and check-boundaries pass; simplicity audit regression is reviewed

**Validation:** focused tests for the touched packages via `make test-changed`; no live provider required.

## Day-by-day schedule

| Day | Work | Focused effort |
|---|---|---|
| 1 | M1 storage; M2 deterministic queries | 3h + 1h buffer |
| 2 | M3 runner lifecycle; M4 CLI/env (policy ruling required) | 3h + 1h buffer |
| 3 | M5 config (OpenRouter ruling required); M6 typed operation | 3h + 1h buffer |
| 4 | M7 builtin/export; M8 cache/TTL | 3h + 1h buffer |
| 5 | M9 subprocess tests; M10 examples/docs/final gates | 3h + 1h buffer |

**Total estimate:** 2160 LOC including tests and documentation; provisional capacity 432 LOC/day. Each milestone remains independently reviewable. If 90 minutes is exceeded, commit tested partial work and report the remaining acceptance criteria for resumption.

## Validation and success metrics

Use `make test-changed` (touched packages plus core) after milestones. Final gates: `make check-file-sizes`, `make check-changelog`, `make verify-stdlib`, `make verify-examples`, `make check-boundaries`; run `make simplicity-audit` and explain any command/env-route changes against its baseline. Six new flags and two new path env vars must be documented (the design acceptance text mentioning four flags is stale). Run gofmt and focused lint checks for modified Go files. CI owns the full suite; executor does not run `make test`.

Acceptance is behavioral coverage of every new branch: nil/error/success embeddings, capability/budget denial, precedence/misconfiguration, persistent restart, namespace isolation, Strict/maxScan parity, handle cleanup and cache expiry. Do not claim a global coverage percentage from an unmeasured baseline. Run both new examples with a stub embedder integration harness and source-built `ailang check`; existing three examples remain check-clean. Real provider smoke is optional and credential-dependent, never a CI prerequisite.

## Risks and handoff

Cache keys use the resolved provider-qualified ModelName and NUL delimiter; assert model switching misses. The initial scope promises repeated sequential calls, not distributed single-flight. Separate table ownership avoids SharedMem/index collisions. No new imports from effects into messaging/platform: config wiring stays in runner/CLI and backend registration stays in platform startup.

JSON: `.ailang/state/sprints/sprint_M-PERSISTENT-SEMANTIC-STORE.json`. All features start passes=null. No concrete GitHub issue number exists in the handoff/design, so github_issues is empty rather than guessed. Coordinator approval/merge is the handoff to sprint-executor; do not send an independent execution message or begin implementation from this planning task. Executor must observe the two cited open-decision gates. After execution, sprint-evaluator checks the approved design and acceptance criteria.
