# Sprint Plan: M-BYTECODE-GETFIELD-SLOT-RESOLUTION

**Design doc:** [m-bytecode-getfield-slot-resolution.md](m-bytecode-getfield-slot-resolution.md)  
**Sprint ID:** `M-BYTECODE-GETFIELD-SLOT-RESOLUTION`  
**Tracking:** ailang#1354; Stapledon M1.6a  
**Target:** v0.47.2  
**Planned from:** v0.47.1 on 2026-09-28  
**Duration:** 2 working days (about 12–16 hours)  
**Dependencies:** Approved design document; no code dependency  
**Risk:** Medium-high. The root cause is localized, but the change crosses lower IR, multi-module lowering, and bytecode compilation on a P0 soundness path.

## Summary

Make record-field slot selection a deterministic function of the receiver's own type. Named records, aliases, imported records, and record updates will carry exact sorted field hints into bytecode compilation; unresolved reads use the existing runtime by-name path, and unresolved updates fail loudly instead of guessing from unrelated record types.

This sprint implements the approved design without changing the VM opcode/layout, evaluator, or language surface. It closes both corruption paths identified in the design: `GET_FIELD` wrong-slot reads and record updates rebuilt with the wrong record shape.

## Current Status and Planning Basis

- The design localizes Defect A to named-record type resolution in `internal/gen/lower/expr.go` and Defect B to module-wide record-type scans in `internal/bytecode/compiler/collections.go`.
- Existing infrastructure already provides `FieldAccess.KnownFields`, the `_record_get` runtime fallback, per-function `EvalOnly` degradation, sorted module traversal, and record type declarations.
- The tree was clean before these planning artifacts on the coordinator work branch. Repository version is v0.47.1; the planned target remains v0.47.2.
- The seven-day velocity script found only one recent design commit and no usable files-changed/LOC history. Therefore no numeric LOC/day claim is made. The two-day capacity is taken from the approved design's localized implementation estimate with verification buffer.
- Estimated changed code and tests: about 300 LOC (approximately 170 implementation/IR plumbing and 130 tests/fixtures), with deletion of the two ambiguous scan paths.

## Scope and Milestone Map

| Milestone | Goal | Estimate | Dependencies | Design criteria |
|---|---|---:|---|---|
| M1 | Build and test deterministic record-type/alias field resolution in lower | 75 LOC | none | AC4, AC8 foundation |
| M2 | Carry exact field sets through field access, record update, and merged multi-module lowering | 95 LOC | M1 | AC1–AC5, AC8 |
| M3 | Delete compiler guessing and enforce hint/by-name/loud-error behavior | 55 LOC | M2 | AC1–AC3, AC7 |
| M4 | Add CLI, compiler, golden, determinism, and regression gates | 75 LOC | M1–M3 | AC1–AC10 |

Total estimate: **300 LOC**, including tests and fixtures. Milestones are ordered because compiler behavior must not be tightened until lower reliably supplies named and imported receiver hints.

## Day 1 — Exact Type Information and Safe Compilation

### M1: Deterministic record-type and alias resolution

**Files to update:**

- `internal/gen/lower/program.go`
- `internal/gen/lower/expr.go`
- New or existing focused tests under `internal/gen/lower/`

**Tasks:**

1. Build a type-name-to-sorted-fields resolver from surface `TypeDecl`s.
2. Resolve direct named records and chained aliases with a cycle guard; return no hint for non-record `TCon` values.
3. Thread the resolver through lower alongside `CoreTypeInfo` and extend `recordFieldSet` to resolve named receivers.
4. Add table-driven tests for direct records, alias chains, cycles/non-record types, anonymous records, and deterministic sorting.

**Acceptance criteria:**

- Named record `TCon` receivers yield their own lexicographically sorted full field set.
- Alias chains resolve to the terminal record field set without looping.
- ADTs, unknown names, and cyclic/non-record aliases yield no record hint.
- Existing `TRecord` and `TRecord2` behavior remains green.

**Risk:** Surface AST type-declaration variants may not map one-to-one to lowered declarations. Mitigation: keep the resolver focused on record declarations and aliases, mirror the existing alias cycle semantics, and pin each declaration form with unit tests.

### M2: Propagate receiver-owned fields across all lowering paths

**Files to update:**

- `internal/gen/lower/program.go`
- `internal/gen/lower/expr.go`
- `internal/gen/stmt/stmt.go`
- `internal/runner/vm.go`
- Relevant lower/runner tests

**Tasks:**

1. Populate `FieldAccess.KnownFields` for named parameters, let-bound call results, function results, and aliases.
2. Add `KnownFields []string` to `stmt.RecordUpdate` and populate it from the base expression's receiver type.
3. Construct one merged type resolver for all loaded modules and make imported record declarations available while lowering each module.
4. Preserve deterministic sorted module traversal and avoid changing emitted type declarations or evaluator behavior.

**Acceptance criteria:**

- Single-file and imported `Motion.x`/`Vec3.x` accesses carry distinct correct field sets.
- Record updates carry the base record's complete sorted fields.
- Anonymous/row-polymorphic record hints still work.
- Tuple access and ADT switch binding lowering are unchanged.

**Risk:** Bare names can collide across modules. Mitigation: follow the approved merged-declaration contract and existing canonical module ordering; add a multi-module fixture matching the Stapledon shape before compiler fallback removal.

### M3: Remove cross-type guessing from the bytecode compiler

**Files to update:**

- `internal/bytecode/compiler/collections.go`
- `internal/bytecode/compiler/collections_test.go`
- `internal/bytecode/compiler/multimodule_test.go`

**Tasks:**

1. Remove the `recordTypes` scan from field-index resolution.
2. Resolve direct `GET_FIELD` only from `KnownFields`; if no hint exists, use `_record_get` by name.
3. Treat a non-empty hint missing the requested field as a compiler inconsistency rather than silently routing through another type.
4. Remove the record-update type scan and rebuild only from `RecordUpdate.KnownFields`; return a compile error when the base shape is unavailable so existing `EvalOnly` handling remains loud under strict mode.
5. Update hand-built IR tests to state their record field hints explicitly.

**Acceptance criteria:**

- No Go map iteration participates in `GET_FIELD` or record-update shape selection.
- Empty field-access hints compile through `_record_get`.
- Record updates never borrow another registered type's field set.
- Tuple `_N` positional access remains direct and unchanged.

**Risk:** A missing lower hint can reduce bytecode coverage. Mitigation: reads retain a correct by-name VM path, updates degrade through the existing explicit compiler/EvalOnly contract, and M4 exercises all expected static paths.

## Day 2 — Regression Matrix and Release Gates

### M4: Prove semantic parity and determinism

**Files to update/create:**

- `cmd/ailang/run_bytecode_test.go`
- `internal/bytecode/compiler/collections_test.go`
- `internal/bytecode/compiler/multimodule_test.go`
- Focused fixtures/goldens under `tests/golden/bytecode/`

**Tasks:**

1. Add OOB and in-bounds shared-field fixtures, record-update shape coverage, alias-chain coverage, and the `Motion`/`Vec3` game-shape multi-module case.
2. Add a table-driven generality test with at least three record types and receiver flows through named parameters, let-bound call results, function results, and aliases.
3. Assert direct opcode indices when static resolution is available and `_record_get` when it is not.
4. Run strict bytecode fixtures ten consecutive times; compare interpreter and strict-bytecode values.
5. Run game-shape disassembly ten times and assert byte-identical instruction streams.
6. Run non-strict bytecode fixtures and assert stderr contains no `falling back to evaluator` warning.
7. Run the focused and repository regression gates below.

**Acceptance criteria:**

- Repro2 returns 13.0, repro5 returns 13.0, repro6 returns 99.0 with `MAKE_RECORD count=2`, and repro7 returns 10.0 in 10/10 strict runs.
- The game-shape `Motion.x` and `Vec3.x` fixture matches interpreter output in 10/10 strict runs.
- Ten disassemblies of the game-shape fixture are byte-identical.
- Plain bytecode execution of all new fixtures emits no evaluator-fallback warning.
- Interpreter outputs are unchanged.
- All focused compiler, CLI, golden, core, formatting, lint, and boundary gates pass.

**Risk:** A one-shot unit test can miss the former map-order nondeterminism. Mitigation: retain the explicit ten-process execution/disassembly checks and exact semantic assertions for the in-bounds corruption case.

## Verification Commands

The executor should adapt exact `-run` names to the committed test names while preserving these scopes and anti-vacuity checks.

```bash
go test ./internal/gen/lower/... -count=1
go test ./internal/bytecode/compiler/... -count=1
go test ./cmd/ailang -run 'Bytecode|StrictBytecode|GetField|RecordUpdate' -count=1 -v
make test-core
make fmt
make lint
make check-boundaries
```

For every named `go test -run` gate, verify the output contains the expected `=== RUN` lines so a missing/deleted test cannot pass vacuously. The CLI fixtures must additionally be invoked as separate processes for the 10/10 determinism checks.

## Success Metrics

- All ten design criteria AC1–AC10 are mapped to M1–M4 and pass.
- Four or more concrete regression fixtures cover OOB, in-bounds, update, alias, and imported game shapes.
- Direct static reads use receiver-owned indices; unresolved reads use by-name lookup; unresolved updates fail loudly.
- No VM opcode/layout or evaluator change.
- No fallback warning for the supported fixtures in normal bytecode mode.
- `make test-core`, focused package tests, formatting, lint, and architecture boundaries are green.

## Dependencies, Assumptions, and Handoff

- No external service, GPU, schema migration, or package dependency is required.
- ailang#1355 remains separate; this sprint removes one proven nondeterminism channel but does not claim to close unrelated VM/interpreter divergence.
- The plan does not authorize implementation by itself. Per repository workflow, execution starts only after the user says **execute sprint**, at which point `sprint-executor` should consume the JSON progress artifact.
- No open design question blocks execution; the approved design controls if implementation details conflict with this plan.
