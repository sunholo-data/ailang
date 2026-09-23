# Sprint Plan: Type-Anchored Defaultable Record Fields

## Summary

Implement `default Type.field = expr` as a type-checked, cross-module record-construction feature. Omitted defaultable fields are completed during elaboration into full closed records; the evaluator continues to receive ordinary record literals and never supplies fallback values.

**Design doc:** `design_docs/planned/v1_1_0/m-record-defaultable-fields.md`  
**Issue:** #905  
**Target:** v1.1.0  
**Duration:** 5 engineering days  
**Estimated size:** ~1,200 LOC (implementation, tests, examples, and docs)  
**Dependencies:** Approved design document; no code dependency  
**Risk level:** High (parser, types, elaboration, and package-interface semantics)

## Current Status Analysis

### Verified baseline

- Closed record literals reject omitted fields in `internal/types/unification_records.go`.
- `ast.TypeDecl` and `ast.RecordType` retain the named record fields needed to validate anchors.
- Cross-module named-record information currently travels through `iface.Iface.TypeAliases`; interface JSON and cache keys already have explicit versioning and round-trip tests.
- `internal/types/json.go` now supports `KRow`, so the design document's earlier `RowVar` serialization observation must be re-tested rather than treated as an active defect.
- Existing full literals, open rows, nested records, ADTs, and Option-typed fields remain compatibility fixtures.

### Velocity and capacity

The seven-day history contains only the v0.42.0 release commit, so it provides no defensible LOC/day measurement. This plan uses the approved five-day estimate and a conservative capacity of ~240 LOC/day, including tests and documentation. Each semantics-sensitive milestone has its own test gate.

### Implementation decision resolved by this plan

Parse a standalone top-level declaration:

```ailang
default ExtensionHooks.on_pre_step = \_ctx _msgs. PassThrough
```

Each declaration elaborates to a deterministic hidden module binding. Interfaces carry metadata mapping the anchored `(type, field)` to that binding and its checked field type; they do not serialize source AST. At a contextually typed construction site, elaboration inserts references to these bindings for omitted defaultable fields. This representation preserves cross-package execution, cacheability, and ordinary evaluator semantics.

The first implementation supports defaults anchored to named closed record aliases. It rejects unknown types, non-record aliases, unknown fields, duplicate defaults, open record aliases, and parameterized anchors with coded diagnostics; parameterized defaults require a later design because instantiation and binding polymorphism need an explicit policy.

## Proposed Milestones

### M1: Syntax, AST, and declaration validation

**Estimated:** 90 implementation + 120 tests = 210 LOC  
**Duration:** 0.75 day  
**Files:** `internal/lexer/token.go`, lexer keyword tables, `internal/ast/ast_decl.go`, `internal/ast/print.go`, `internal/parser/parser_decl.go`, new/adjacent parser tests and goldens

**Tasks:**

- Reserve `default` and parse `default Type.field = expr` only at top level.
- Add `ast.DefaultFieldDecl` with anchor type, field, expression, export/module ownership, and positions.
- Add structured parser errors for incomplete anchors and declarations.
- Validate declaration shape before type checking: named closed record alias, existing field, no duplicate anchor, and no parameterized/open anchor in v1.

**Acceptance criteria:**

- Valid declarations parse and round-trip through AST printing/goldens.
- Invalid/malformed declarations produce stable coded diagnostics without panics or parser cascades.
- Existing use of identifier text containing `default` remains unaffected; existing parser suites pass.

**Risk:** Keyword routing can affect top-level recovery. Mitigate with declaration-boundary and multi-error fixtures.

### M2: Default registry and declaration-site type/effect checking

**Estimated:** 150 implementation + 130 tests = 280 LOC  
**Duration:** 1 day  
**Dependencies:** M1  
**Files:** `internal/types/typechecker*.go`, `internal/types/errors.go`, `internal/elaborate/file_funcs.go`, `internal/core/`, focused type/effect tests

**Tasks:**

- Resolve each anchor to its declared record field type and register it by canonical module/type/field identity.
- Type-check the default expression at its declaration site against the field type, including the field function's closed effect row.
- Lower each valid declaration to a deterministic collision-proof hidden binding, retaining source positions for diagnostics and tracing.
- Reject defaults that widen effects/capabilities or depend on an incompatible type.

**Acceptance criteria:**

- Correct pure and effectful function defaults type-check against their exact slots.
- Wrong value types and effect-row widening fail at the declaration, not at consumers.
- Duplicate, unknown-field, wrong-kind, open, and parameterized anchors have distinct structured errors.
- No evaluator change is required for declared defaults.

**Risk:** Type aliases and nominal ADTs share lookup paths. Mitigate by keying only validated record aliases and adding negative ADT tests.

### M3: Contextual record completion and deterministic trace

**Estimated:** 190 implementation + 160 tests = 350 LOC  
**Duration:** 1.5 days  
**Dependencies:** M2  
**Files:** record inference/checking paths in `internal/types/`, record lowering in `internal/elaborate/`, trace structures/output, unit and pipeline tests

**Tasks:**

- Detect a record literal checked against a named closed record alias before closed-row unification reports missing fields.
- Insert only missing fields that have registered defaults; preserve explicitly supplied values and source field order deterministically.
- Keep missing non-defaultable fields on the existing error path and keep extra/open-row behavior unchanged.
- Record anchor, inserted binding, and construction position in the existing deterministic elaboration/defaulting trace mechanism.
- Cover nested and annotated construction positions, function returns, let annotations, and imported function arguments.

**Acceptance criteria:**

- An override-only literal elaborates to the same full closed record shape as a handwritten literal.
- Explicit fields always win; defaults are never evaluated or selected by the runtime as fallbacks.
- Missing required fields still emit the existing missing-field diagnostic.
- Nested literals, full literals, open rows, record subsumption, and ADT constructor records retain current behavior.
- Trace output deterministically identifies every inserted field.

**Risk:** Bidirectional/contextual type information may not be available at every literal. Mitigate by limiting completion to sites with a resolved named expected type and failing normally elsewhere.

### M4: Cross-module interface, cache, and ABI fingerprinting

**Estimated:** 140 implementation + 120 tests = 260 LOC  
**Duration:** 1 day  
**Dependencies:** M2, M3  
**Files:** `internal/iface/iface.go`, `internal/iface/json.go`, `internal/iface/builder.go`, pipeline import plumbing, cache key/version files, interface/cache tests

**Tasks:**

- Add deterministic interface metadata for exported record-field defaults, pointing to hidden exported default bindings.
- Round-trip metadata and binding types through interface JSON and module caches; fail loudly on unsupported/corrupt entries.
- Include default metadata and implementation identity in the interface digest so adding, removing, or changing a default invalidates dependents.
- Bump the cache-key/schema version if the persisted interface shape requires it.
- Add a two-module fixture proving a consumer can omit a field defaulted by an imported ABI module.

**Acceptance criteria:**

- Cold-cache and warm-cache cross-module checks produce identical completed records and traces.
- Adding a default changes the interface digest; unrelated implementation changes retain existing digest policy.
- Missing referenced hidden bindings or malformed metadata are hard errors, never ignored fallbacks.
- `KRow`/`RowVar` interface serialization is re-tested; any current gap is fixed with round-trip coverage.

**Risk:** Hidden bindings can be omitted by export filtering. Mitigate with an explicit interface-builder path and end-to-end package fixtures.

### M5: Regression suite, user example, and documentation

**Estimated:** 25 implementation/docs + 75 tests/examples = 100 LOC  
**Duration:** 0.75 day  
**Dependencies:** M3, M4  
**Files:** `examples/record_defaultable_fields.ail`, `docs/docs/reference/language-syntax.md`, changelog, design status notes, existing record fixtures

**Tasks:**

- Add a runnable example with required identity fields, defaulted hooks, and an explicit override.
- Document syntax, construction-only semantics, restrictions, interface behavior, and minor/major ABI version guidance.
- Run the four design-doc regression fixtures plus parser, type, pipeline, interface/cache, boundary, example, and full test gates.
- Record implementation evidence in the design document; leave motoko scaffolder changes as the named follow-up.

**Acceptance criteria:**

- `ailang check` and `ailang run` pass for `examples/record_defaultable_fields.ail`.
- `examples/record_cons_pattern.ail`, `record_in_result.ail`, `record_list_extraction.ail`, and `extension_self_disable.ail` remain green.
- `make fmt`, `make test-core`, `make test`, `make lint`, `make check-boundaries`, and relevant example verification pass or document independently reproduced pre-existing failures.
- Documentation clearly states that adding a field with a default is additive while adding a required field remains breaking.

## Day-by-Day Plan

1. Day 1: M1, then begin M2 registry and anchor validation.
2. Day 2: Complete M2 and declaration-site effect/type tests.
3. Day 3: Implement M3 contextual completion and required-field non-regressions.
4. Day 4: Finish M3 tracing; implement M4 interface/cache plumbing and two-module tests.
5. Day 5: Complete M4 digest/cache gates, M5 example/docs, and full validation.

## Success Metrics

- One new top-level declaration form with coded parser/type diagnostics.
- Override-only construction works locally and across package interfaces with warm and cold caches.
- 100% branch coverage for default insertion decisions: explicit, defaulted, required-missing, extra, unknown, and invalid declaration.
- Default effect rows cannot exceed their slots; no new authority is granted.
- Every insertion is deterministic and trace-visible.
- Existing record/effect regression suites and repository quality gates pass.
- A runnable example and language reference document the shipped syntax.

## Dependencies and Execution Gates

- The design document is approved, but this is security- and semantics-sensitive work. The executor should preserve milestone boundaries and stop if M2 reveals that defaults cannot be checked before consumer elaboration.
- M4 is mandatory for issue #905; a local-only implementation does not satisfy the extension ABI use case.
- Do not update motoko extension templates in this sprint. That follow-up occurs only after the language feature is released.
- Implementation starts only after the user/coordinator explicitly says `execute sprint`.

## Open Questions Resolved for Execution

- **Syntax:** standalone `default Type.field = expr`.
- **Observation:** trace-only; completed fields are ordinary record fields downstream.
- **Export:** exported through interface metadata plus hidden bindings.
- **Interface compatibility:** adding/removing/changing default metadata changes the interface digest and invalidates dependents.
- **Parameterized record aliases:** rejected in v1 with a structured diagnostic, pending a separately designed instantiation policy.

