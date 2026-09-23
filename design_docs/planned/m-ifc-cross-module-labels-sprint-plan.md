# Sprint Plan: M-IFC-CROSS-MODULE — Preserve IFC Labels Across Module Boundaries

## Summary

Carry each exported function's IFC obligations and label-flow summary through the module interface, then enforce those facts in importing modules. This closes issue #1134 in both directions: imported sink refinements reject tainted arguments, and imported labelled/intrinsically-tainted returns remain tainted in the caller.

**Duration:** 4 days  
**Dependencies:** Approved `design_docs/planned/m-ifc-cross-module-labels.md`; shipped M-TAINT-TYPES/M-SECRET-EFFECT label lattice, `TLabelled` cache codec, and closure-laundering fix  
**Risk Level:** High — security-sensitive semantics and interface schema/cache invalidation

## Current Status Analysis

### Completed Foundations

- `CheckModuleIFC` already enforces sink refinements, return declassification, transparent propagation, and closure labels within one surface AST.
- `internal/iface.Builder` already reconstructs `TLabelled` parameter/return annotations from the AST and serializes interface types.
- Module cache keys already incorporate dependency interface digests.
- CLI module compilation and REPL loading both invoke the IFC gate, but neither provides imported IFC facts.

### Velocity and Capacity

The seven-day velocity script found only one recent design-only commit and no reliable implementation LOC baseline. Use the approved design's four-day estimate, with explicit security/cache test time rather than extrapolating an unsupported LOC/day rate.

**Planned capacity:** approximately 1,000 LOC across production code, tests, fixtures, and documentation.

### Remaining from the Design Doc

- Add a versioned, digest-bearing IFC projection to every exported interface item.
- Compute each export's intrinsic body label in the dependency's compilation context.
- Resolve imported call names to IFC summaries and enforce the same rules as local calls.
- Prove JSON/gob round trips, stale-schema rejection, cache invalidation, CLI/REPL parity, and the #1134 reproductions.

## Technical Guardrails

- Keep IFC as a surface-AST pass; do not change unification or allow `TLabelled` into CoreTI/codegen.
- Treat absent/old IFC metadata as a schema mismatch requiring rebuild, never as an unlabelled export.
- Preserve deterministic diagnostic order and name the caller-to-imported-callee edge.
- Cover all import spellings supported by `resolveModuleImports`: selective, aliased, and qualified references.
- Run `make check-boundaries` because the change spans `types`, `iface`, `pipeline`, and `repl` layers.

## Proposed Milestones

### M1: Versioned IFC Interface Projection (~260 LOC)

**Goal:** Define a serializable IFC summary on every `IfaceItem`, include it in JSON/gob/cache projections and the deterministic digest, and bump the interface schema to `ailang.iface/v2`.

**Estimated:** 120 LOC implementation + 140 LOC tests = 260 LOC  
**Duration:** Day 1  
**Dependencies:** None

**Files to update:**

- `internal/iface/iface.go`
- `internal/iface/json.go`
- `internal/iface/hash_projection.go`
- `internal/iface/builder.go`
- `internal/pipeline/cache_store.go`
- Interface JSON, digest, and cache round-trip tests under `internal/iface/` and `internal/pipeline/`

**Tasks:**

- Define exported summary/parameter structs carrying source label, forbidden label, declared return label, intrinsic label, and declassification authority.
- Emit an explicit summary for every function export, including the empty summary.
- Add the summary to compact JSON and cache gob representations and include it in `computeDigest`.
- Change newly built interfaces to schema v2 and reject readable v1/pre-summary artifacts through the existing rebuild/error path.

**Acceptance Criteria:**

- [ ] Every exported function has explicit IFC metadata after interface construction.
- [ ] JSON and gob/cache round trips preserve every IFC summary field.
- [ ] Changing only IFC metadata changes the interface digest and therefore the dependent module cache key.
- [ ] A v1 or otherwise summary-less readable interface cannot silently enter compilation as clean metadata.
- [ ] Serialization is deterministic and `go test ./internal/iface ./internal/pipeline` passes.

**Risks:** Existing digest golden tests may move. Update only goldens whose semantic projection intentionally changed and retain controls proving unrelated ordering stays stable.

### M2: Dependency-Side Intrinsic Summary Computation (~230 LOC)

**Goal:** Reuse the existing IFC label walk/fixpoint to compute exported function summaries while the dependency body is available.

**Estimated:** 100 LOC implementation + 130 LOC tests = 230 LOC  
**Duration:** Day 2  
**Dependencies:** M1

**Files to update:**

- `internal/types/ifc_check.go`
- `internal/types/ifc_check_test.go`
- `internal/types/ifc_closure_test.go`
- `internal/iface/builder.go` and focused builder tests

**Tasks:**

- Expose a narrow, immutable exported-summary API without exposing checker internals.
- Seed parameters at bottom when calculating `intrinsic`, preserving the current recursion guard/fixpoint and conservative closure behavior.
- Record declared parameter labels/refinements, return labels, and `Declassify` effects from the surface declaration.
- Build summaries only for exported functions and keep existing intramodule diagnostics unchanged.

**Acceptance Criteria:**

- [ ] An unannotated wrapper around `secret()` exports intrinsic label `secret`.
- [ ] Transparent forwarding does not invent an intrinsic label from bottom-seeded parameters.
- [ ] Explicit return labels and `Declassify` authority are represented distinctly.
- [ ] Recursive functions and closure-returning functions terminate and remain conservatively labelled.
- [ ] Existing single-module IFC tests pass unchanged.

**Risks:** Duplicating label evaluation could drift from enforcement. Use one shared summary computation path and test it against existing local-call behavior.

### M3: Import Resolution and Caller-Side Enforcement (~300 LOC)

**Goal:** Pass imported IFC signatures into `CheckModuleIFC` and apply local-equivalent sink and result-label rules at imported calls in both CLI and REPL compilation paths.

**Estimated:** 140 LOC implementation + 160 LOC tests = 300 LOC  
**Duration:** Day 3  
**Dependencies:** M1, M2

**Files to update:**

- `internal/types/ifc_check.go`
- `internal/types/errors.go`
- `internal/pipeline/pipeline_module_imports.go`
- `internal/pipeline/pipeline_module_compile.go`
- `internal/repl/module_registry_load.go`
- `internal/types/ifc_check_test.go`
- `internal/pipeline/ifc_cross_module_test.go` (new end-to-end fixture owner)
- REPL-focused IFC test near `internal/repl/module_registry_load.go`

**Tasks:**

- Add `ImportedIFCSig` input to the checker and populate it alongside external types/global refs for selective, qualified, and aliased imports.
- Run Check A against imported parameter refinements.
- Compute imported result labels as declared/declassified return labels or `join(intrinsic, argLabels)` for transparent functions.
- Extend diagnostics with the imported symbol and caller → callee edge while keeping existing coded error kinds.
- Wire the same summary map into REPL module loading.

**Acceptance Criteria:**

- [ ] The exact #1134 two-module sink repro fails with a diagnostic naming the imported edge and parameter.
- [ ] A library-labelled or intrinsically-secret return remains labelled and is rejected by a caller-local sink.
- [ ] Clean cross-module calls compile, including selective, module-alias, and qualified import forms.
- [ ] Explicit authorised declassification breaks the taint chain; an unauthorised return-label drop still fails.
- [ ] Closure-derived arguments retain their label when passed to an imported sink.
- [ ] CLI and REPL return the same verdict and error kind for equivalent modules.

**Risks:** Surface callee spelling may not match linker keys. Centralize name-to-summary registration beside `GlobalRefs` and lock all import forms with table-driven tests.

### M4: Security Regression Matrix, Docs, and Release Evidence (~210 LOC)

**Goal:** Complete cache/schema regression coverage, add a maintained two-module fixture, document the breaking enforcement change, and run repository-wide gates.

**Estimated:** 40 LOC implementation/docs + 170 LOC tests/fixtures = 210 LOC  
**Duration:** Day 4  
**Dependencies:** M3

**Files to create/update:**

- `internal/pipeline/testdata/ifc_cross_module/` (library/caller fixtures for rejected and accepted flows)
- `examples/runnable/secrets/README.md` and/or a linked multi-module example fixture
- Current release changelog entry
- Prompt source describing IFC label/refinement semantics, located via `ailang prompt` before editing
- Relevant cache artifact/schema tests

**Tasks:**

- Add positive and negative two-module fixtures covering sink, return, intrinsic, declassify, closure, and stale-cache cases.
- Confirm a changed dependency summary invalidates an already-built importer.
- Document that cross-module flows previously accepted may now be rejected and show the required declassification/library fix.
- Run formatting, focused tests, core tests, full tests, lint, and architecture boundaries.

**Acceptance Criteria:**

- [ ] The checked-in two-module fixture reproduces issue #1134 before the fix and rejects it after the fix.
- [ ] Rebuilding a dependency with only an IFC-summary change invalidates the importer cache.
- [ ] Prompt/docs/changelog describe cross-module enforcement and authorised declassification.
- [ ] `make fmt`, `make test-core`, `make test`, `make lint`, and `make check-boundaries` pass.
- [ ] No `TLabelled` reaches CoreTI or codegen, verified by existing boundaries/tests plus a focused regression assertion.

**Risks:** Full-suite failures may be unrelated in a busy worktree. Record focused-test evidence first, then separate pre-existing failures from sprint regressions without weakening gates.

## Day-by-Day Execution

| Day | Deliverable | Exit Evidence |
|---|---|---|
| 1 | Interface summary, v2 schema, serialization, digest | Round-trip tests; metadata-only digest delta; old-schema rejection |
| 2 | Shared dependency-side summary computation | Intrinsic/declassify/recursive/closure unit tests |
| 3 | Pipeline + REPL import wiring and enforcement | #1134 and mirror repros; clean/declassify/import-form controls |
| 4 | Cache matrix, fixtures, docs, full gates | Cache invalidation proof; documented example; repository gates |

## Success Metrics

- Issue #1134's imported sink repro is rejected with an actionable edge diagnostic.
- The mirror imported-return path is rejected in the caller.
- At least six cross-module security cases cover sink, declared return, intrinsic return, declassification, closure flow, and cache invalidation.
- CLI and REPL verdicts agree.
- Interface summaries round-trip and participate in digests; old summaries never silently widen labels.
- Existing intramodule IFC behavior and HM unification remain unchanged.

## Dependencies and Open Questions

No design question blocks execution; the approved design fixes the schema, conservative propagation, and surface-AST architecture. During implementation, the executor should verify the exact cache schema-validation entry point before changing it and should run `ailang prompt` before editing any `.ail` fixture or teaching prompt.

## Execution Handoff

Implementation requires the user/coordinator to say **execute sprint**. The sprint executor should use TDD, update only progress fields in the JSON tracker, and reference GitHub issues #1134 and #1132 without closing #1132's out-of-scope tracing work.
