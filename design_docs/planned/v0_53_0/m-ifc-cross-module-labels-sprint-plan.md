# Sprint Plan: M-IFC-CROSS-MODULE

**Target:** v0.53.x (v0.54.0 if release sequencing requires it)
**Status:** Planned; execution awaits sprint approval
**Date:** 2026-10-08
**Issues:** Refs #1134; Refs #1132 (part 1 only)
**Design:** `design_docs/planned/m-ifc-cross-module-labels.md` (merged df8bc23ca)
**Scheduling approval:** Mark, 2026-10-08, P0 issue triage
**Duration:** 5 working days, approximately 30 focused hours
**Risk:** High — security-sensitive semantics and interface cache compatibility

## Goal and boundaries

Make imported sink refinements and return labels enforceable at compile time, using serializable per-export IFC summaries and the existing surface-AST checker. The same-module rejection must also occur when the sink moves to a library. Include intrinsic taint, explicit Declassify behavior, CLI/REPL parity, and cache safety. Keep label erasure in unification and CoreTI/codegen unchanged.

This implements #1134 and part 1 of #1132. Label-aware tracing, Z3 cross-module function contracts, and label-polymorphic inference remain separate work. Do not close #1132 after this sprint: its tracing half remains outstanding. No new issue is needed. The plan PR must use `Refs #1134` and `Refs #1132`; only the implementation PR may use `Closes #1134`, alongside `Refs #1132`.

## Current state and estimate

Maintainer triage reproduced the defect on origin/dev 658ff76a3: a same-module call errors while the lib/app version checks clean. This session inspected the implementation rather than repeating that live reproduction with the installed stale binary. Both `pipeline_module_compile.go` and `module_registry_load.go` call `CheckModuleIFC` without imports. The iface builder preserves TLabelled but omits sink refinements, and full cache JSON manually copies export fields without IFC metadata.

The design covers both missing sink checks and lost result taint, audits the existing sink helpers and label erasure, and proposes one summary mechanism for both. Its v0.39.0 target is stale against `std/VERSION` v0.52.5; schedule this plan for v0.53.x.

The velocity script found no usable recent LOC metrics. This is a shallow checkout exposing one recent design-only commit, so historical LOC/day is unavailable. Use the design's four-day estimate plus 25% security/cache buffer: five days, 1,000 estimated changed lines (550 implementation/documentation + 450 tests), a planning capacity of 200 LOC/day rather than a measured velocity.

## Registry reuse audit

Ran `ailang pkg search ifc`: one result, `sunholo/linkedin@0.5.1`. `pkg info sunholo/linkedin` identifies an experimental application client using IFC/Declassify; `pkg docs sunholo/linkedin` errors because it has no AGENT.md. It cannot supply compiler summaries, cache validation, or type-check enforcement. No package-like runtime capability is being added. M1–M4 each choose `none`: these changes belong in compiler/interface/REPL internals and their regression tests. No registry dependency or contribution is required.

## Milestones

### M1: Export IFC summaries and deterministic schema (~300 LOC)

**Estimate:** 180 implementation + 120 tests; 1.5 days / 9 hours
**Dependencies:** None

Add types-owned ImportedIFCSig/summary data so internal/types never imports internal/iface. Export a reusable analysis result from the checker, computing intrinsic body labels with parameters at bottom and imported summaries available. The iface builder attaches a non-nil summary to every export, including clean/non-function exports with explicit empty metadata. Capture parameter name, source label, forbidden label, declared return label, intrinsic label, and Declassify status. Preserve the lattice's full effective label representation; do not collapse joined labels into a lossy string. Use the existing fixpoint/memoization semantics, including local recursion and wrappers over imported producers.

Files: `internal/types/ifc_check.go`, a small `internal/types/ifc_summary.go` if needed, `internal/iface/iface.go`, `internal/iface/builder.go`, associated summary/builder tests. All interface creation paths, including builtin/synthetic exports, need valid summaries. Extend canonical digest input and compact/published JSON; bump schema to `ailang.iface/v2`.

Example coverage: summary tests use the lib functions later published under `examples/runnable/ifc_cross_module/lib.ail`; no new standalone language feature is introduced.

- [ ] Every exported item has explicit IFC metadata; clean is distinguishable from missing.
- [ ] Source labels, sink refinements, declared/intrinsic returns, and Declassify survive summary construction.
- [ ] Local recursion and a wrapper around an imported secret producer retain intrinsic taint.
- [ ] Each security-relevant summary field changes the digest; export map order does not.
- [ ] TLabelled erasure and CoreTI/codegen behavior remain unchanged.

Risk: summary computation re-enters checking or introduces an import cycle. Keep the analysis reusable and data types in types; reuse the existing fixed-point rules.

### M2: Persist summaries and reject incompatible caches (~220 LOC)

**Estimate:** 100 implementation + 120 tests; 1 day / 6 hours
**Dependencies:** M1

Update full cache JSON DTOs and both marshal/unmarshal paths in `internal/pipeline/cache_store.go`, iface gob/JSON coverage, and validation in `internal/pipeline/cache_artifacts.go`. Audit cache keys and dependency digest propagation in `cache_dep_closure.go`. Existing artifact hashes authenticate stored bytes, but the current artifact reader does not explicitly reject a v1 IFC schema: do not assume a schema string bump alone forces recompilation. Add schema/summary validation and, if needed, a cache-key format bump. Old entries become an observable cache miss and recompile from source; if no source/valid interface is available, report an actionable error instead of accepting clean metadata. Malformed v2 metadata must be rejected too.

Files/tests: cache store/artifact tests, iface serialization tests, dependency invalidation integration tests. Example coverage uses M4's lib/app pair with cold and warm cache.

- [ ] Full cache JSON, compact/published JSON, and supported gob paths preserve every IFC summary field.
- [ ] A validly hashed v1 artifact cannot bypass IFC enforcement and causes recompilation.
- [ ] A v2 interface with absent or malformed metadata cannot be treated as clean.
- [ ] A library-only refinement/intrinsic/return change invalidates dependent cached artifacts.
- [ ] Cold and warm cache verdicts match, including a cached caller (audit whether IFC reruns on cache hits).

Risk: cached caller bypasses the checker. Prove key/dependency invalidation and cold/warm verdict parity rather than relying on the design's unverified assertion that checking runs every compile.

### M3: Enforce imported contracts in CLI and REPL (~320 LOC)

**Estimate:** 180 implementation + 140 tests; 1.5 days / 9 hours
**Dependencies:** M1, M2

Generalize CheckModuleIFC and all callers/tests. Resolve summaries with the same symbol identity rules as imported type/global refs: selective imports, renamed symbols, qualified module aliases, and local shadowing. Imported sinks use the same Check A as local sinks; declared/declassifying returns are authoritative and transparent results join intrinsic and argument labels. Preserve the conservative unknown closure/builtin path. Build summaries while compiling dependencies so a library calling another library also carries taint.

Files: `internal/types/ifc_check.go`, `internal/pipeline/pipeline_module_imports.go`, `pipeline_module_compile.go`, `internal/repl/module_registry_load.go`, relevant registry structures/interface persistence and tests. REPL currently checks before import resolution; move the gate after summaries are available and before publishing exports. Do not leave a failed module partly registered.

Examples: M4's library/app pair and positive declassification caller; use Go fixtures for negative calls, closure arguments, aliases, and recursive wrappers.

- [ ] #1134 two-module repro fails, naming caller location, canonical imported callee and forbidden parameter.
- [ ] Declared secret return and unannotated intrinsic-secret return both taint caller sinks.
- [ ] Imported transparent functions join argument labels, including closure body taint.
- [ ] A legal imported Declassify path passes; library return-label violations still fail.
- [ ] Selective, renamed, qualified, shadowed, and transitive wrapper cases resolve correctly.
- [ ] CLI and REPL agree for positive and negative fixtures; failed REPL loads publish nothing.

Risk: alias resolution bypasses contracts. Share/import the resolved identity map rather than implementing independent spelling guesses.

### M4: Examples, documentation and release validation (~160 LOC)

**Estimate:** 90 examples/documentation + 70 tests; 1 day / 6 hours
**Dependencies:** M1, M2, M3

Create `examples/runnable/ifc_cross_module/lib.ail`, `app.ail`, and `README.md` with a legal clean/declassified call and commands. Store illegal sink/return fixtures in Go integration tests, keeping runnable examples passing. Run `ailang prompt` before writing any .ail code and check every new runnable example with the freshly built binary.

Update `docs/docs/guides/ifc-labels.mdx`: remove the declaring-module-only limitation and cross-module sink disclaimer only after the regression matrix passes, replace them with enforced imported-contract behavior, and retain the independent Z3 limitation. Update current teaching prompt source as indicated by prompt-manager and add a breaking behavior/cache invalidation release note in the current changelog. Move the design to the implementation release directory on completion and preserve links.

- [ ] Runnable lib/app example checks successfully; deliberately illegal fixtures fail for IFC rather than unrelated errors.
- [ ] Guide no longer discloses the fixed cross-module limitation and still accurately describes Z3 limitations.
- [ ] Teaching prompt and release notes describe imported sink errors and one-time cache recompilation.
- [ ] Focused tests, full suite, lint, and architecture boundaries pass.
- [ ] sprint-evaluator checks the finished implementation against the design and these criteria.

Risk: documentation overclaims #1132 completion. State compile-time label enforcement only; tracing remains separate.

## Day-by-day execution

1. Read architecture and IFC tests; bank same-module/two-module regressions; implement summary data and digest tests (M1).
2. Finish M1 intrinsic analysis, implement cache codecs and schema validation (M2).
3. Finish cache invalidation regressions; implement pipeline import identity mapping and enforcement (M3).
4. Finish REPL wiring, alias/shadow/closure/transitive and Declassify test matrix (M3).
5. Verify examples, update guide/prompt/changelog, run release checks and sprint-evaluator (M4). The five-day budget includes the one-day buffer; do not add a sixth nominal day.

## Validation and success metrics

Run focused `go test ./internal/types ./internal/iface ./internal/pipeline ./internal/repl` with IFC, serialization, cache, and module fixtures. Use `go test -cover` for those packages to establish a baseline; require no package coverage regression and cover each new security branch with positive/negative assertions rather than claim an unmeasured global percentage. Finish `make test-core`, `make test`, `make lint`, and `make check-boundaries`; format edited Go files. Check runnable examples with a source-built binary. Documentation checks should use the existing docs build target.

The verdict matrix includes same/two-module sinks, declared/intrinsic secret returns, argument taint, imported wrappers, recursive local intrinsic flow, aliases, shadowing, closure arguments, valid/invalid declassification, clean exports, legacy/malformed interfaces, all serializer paths, cold/warm caller and dependency caches, and CLI/REPL parity. Record any failing baseline separately before implementation.

## Dependencies and handoff

Label lattice, TLabelled cache codec, closure-laundering fix and REPL gate already exist. No external package or new issue blocks work. The summary representation, REPL export storage, cache-hit behavior and all interface producers require execution-time audit; the tasks above explicitly cover those uncertainties.

Machine progress: `.ailang/state/sprints/sprint_M-IFC-CROSS-MODULE.json`. All milestone passes begin null. This task creates planning artifacts only. Coordinator approval/merge of the plan PR is the execution handoff; do not independently start sprint-executor before that approval. The implementation PR closes #1134 only after the entire acceptance matrix and docs update pass, and references #1132 for its partial delivery.
