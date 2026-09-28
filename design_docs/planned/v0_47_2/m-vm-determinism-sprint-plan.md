# Sprint Plan: M-VM-DETERMINISM

**Design doc:** [m-vm-determinism.md](m-vm-determinism.md)  
**Issue:** [#1360](https://github.com/sunholo-data/ailang/issues/1360)  
**Target:** v0.47.2  
**Priority:** P0 — restores the VM/interpreter determinism guarantee and unblocks Stapledon's M1.6

## Summary

Remove order-sensitive type selection from the bytecode compiler, make unresolved record access and
record update match interpreter semantics, make ADT fallback ordering explicit, and install regression
guards for bytecode identity and compiler map iteration. The sprint is test-first: each affected path
gets a failing ambiguity fixture before its production change.

**Duration:** 4 days  
**Estimated size:** ~560 LOC (about 210 implementation/docs and 350 tests/fixtures)  
**Dependencies:** Approved `m-vm-determinism.md`; no code dependency  
**Risk level:** High — this changes record lowering, VM instruction semantics, and serialized bytecode

## Current Status Analysis

- The design doc identifies three order-sensitive sites in `internal/bytecode/compiler`: unresolved
  field indexing, record-update type guessing, and ADT inference.
- Existing by-name record lookup already supplies the correct field-access fallback.
- Record update has no equivalent VM operation, so it requires a new `OpUpdateRecord` instruction.
- Existing tests cover ordinary record update and record construction, but not ambiguous type sets,
  repeated compilation, or the game-shaped nested-JSON parity case.
- The isolated planning branch contains only the design-doc commit, so recent LOC/day is not a useful
  velocity signal. The four-day estimate comes from the design's file-level inventory plus a 25%
  validation/debugging allowance.

## Milestones

### M1: Red-first ambiguity fixtures and deterministic field access (~140 LOC)

**Duration:** Day 1  
**Dependencies:** None

**Tasks:**

- Add `internal/bytecode/compiler/determinism_test.go` with two record types whose shared field has
  different sorted indices; prove the unresolved access test fails or produces multiple outcomes on
  the unfixed compiler.
- Add an assertion for correct by-name value, not merely one stable bytecode image.
- Remove the `recordTypes` first-match scan from `lookupFieldIndex`; unresolved access must emit the
  existing `_record_get` path.
- Add ordered ADT registration and an ambiguity test proving repeated compiles choose source order.

**Acceptance criteria:**

- [ ] The unresolved-field regression is demonstrated red on the old implementation and green after the fix.
- [ ] 200 in-process compiles produce one serialized bytecode image and the expected field value.
- [ ] `inferADTFromCases` does not range over `adtTypes`; same-tag candidates resolve in source order.
- [ ] Focused compiler tests pass with `go test -count=1 ./internal/bytecode/compiler`.

**Risk:** A bytecode-only equality assertion could stabilize the wrong answer.  
**Mitigation:** Every determinism check also asserts an absolute semantic result.

### M2: Typed record-update lowering and runtime fallback (~210 LOC)

**Duration:** Day 2  
**Dependencies:** M1

**Tasks:**

- Add `KnownFields []string` to `stmt.RecordUpdate`, populate it through `recordFieldSet`, and sort
  update names during lowering.
- Replace `compileRecordUpdate`'s map scan: use `KnownFields` when available and emit the runtime
  fallback otherwise.
- Add `OpUpdateRecord` across `opcode.go`, compiler emission, VM dispatch, disassembly, and image
  validation/serialization using the established `OpMakeRecord` pseudo-constant encoding pattern.
- Add VM tests for override replacement, untouched-field preservation, empty updates, multiple
  overrides, missing/new fields as specified by interpreter behavior, and malformed encodings.

**Acceptance criteria:**

- [ ] `compileRecordUpdate` contains no first-match scan over registered record types.
- [ ] Known-shape updates retain the indexed fast path; unknown-shape updates preserve the runtime base shape.
- [ ] VM record-update results are identical to evaluator results for all unit cases.
- [ ] Opcode disassembly and image round-trip tests cover `OpUpdateRecord`.
- [ ] `go test -count=1 ./internal/gen/... ./internal/bytecode/... ./internal/vm/...` passes.

**Risk:** A new variable-length instruction can desynchronize the VM, disassembler, and image walker.  
**Mitigation:** Share the existing `OpMakeRecord` layout convention and test malformed/truncated images.

### M3: Systemic guard and game-shaped parity regression (~160 LOC)

**Duration:** Day 3  
**Dependencies:** M1, M2

**Tasks:**

- Add `internal/bytecode/compiler/maprange_guard_test.go` using Go AST/type information to reject
  order-sensitive map ranges in the compiler package; explicitly justify any allowlist entry.
- Extend the determinism fixture to exercise unresolved field access, unknown-base record update,
  and ambiguous ADT selection in repeated compiles.
- Add an in-repo parity fixture shaped like the reported flow: nested `heading` JSON, colliding
  record field names, record update carrying state, and success response.
- Wire the fixture into an existing Go/CLI parity test so it is executed by `go test ./...`; do not
  leave an unreferenced `.ail` file.

**Acceptance criteria:**

- [ ] The guard fails under a mutation that restores any of the three first-match map scans.
- [ ] The parity fixture produces byte-identical evaluator and VM stdout for 30 repeated VM runs.
- [ ] The fixture asserts tick/state advancement, preventing stable-but-wrong output from passing.
- [ ] The existing bytecode golden corpus has no unexplained diff.

**Risk:** A minimal fixture may omit the interaction that triggers the original failure.  
**Mitigation:** Preserve all measured ingredients: nested decode, colliding layouts, turn-result flow,
contract-bearing step/update, and response construction.

### M4: Validation, external repro, and release documentation (~50 LOC)

**Duration:** Day 4  
**Dependencies:** M3

**Tasks:**

- Run focused tests after each mutation, then `make test-core`, `make test`, `make parity`,
  `make check-boundaries`, `make simplicity-audit`, and formatting/lint targets.
- Rebuild the CLI and run the public Stapledon one-line and offaxis repro loops when the checkout is
  available; record hashes/counts and exact commit/binary version. If unavailable, record that the
  wired in-repo fixture is the acceptance proxy.
- Update the current changelog and `docs/LIMITATIONS.md` with the deferred cross-ADT tag-collision
  soundness limitation.
- Review all bytecode/golden changes; regenerate only intentional artifacts.

**Acceptance criteria:**

- [ ] All repository gates listed above pass without a simplicity or boundary regression.
- [ ] Rebuilt VM is 30/30 identical to the interpreter on the in-repo fixture.
- [ ] Public one-line and offaxis repros are stable when external validation is available.
- [ ] Changelog documents restored VM determinism; limitations document the deferred ADT issue.
- [ ] No unexplained golden-file changes remain.

**Risk:** External game validation may be unavailable on the executor host.  
**Mitigation:** The mandatory in-repo fixture is CI-gated; external validation is additional evidence,
not the only acceptance path.

## Day-by-Day Execution

| Day | Outcome | Primary files |
|---|---|---|
| 1 | Red fixture, correct unresolved access, ordered ADT fallback | `compiler/determinism_test.go`, `collections.go`, `switch.go`, `compiler.go` |
| 2 | Correct known/unknown record updates and complete opcode support | `stmt.go`, `lower/expr.go`, `collections.go`, `opcode.go`, `vm.go`, `disasm.go`, `image.go` |
| 3 | Static guard and continuously executed game-shaped parity fixture | `maprange_guard_test.go`, compiler/CLI parity tests, parity `.ail` fixture |
| 4 | Full gates, external repro evidence, changelog and limitation docs | changelog, `docs/LIMITATIONS.md` |

## Success Metrics

- 200 repeated in-process compiles: one bytecode image and correct absolute results.
- 30 repeated VM executions of the game-shaped fixture: byte-identical to the interpreter.
- Zero unjustified order-sensitive map ranges in `internal/bytecode/compiler`.
- Existing bytecode golden corpus passes with only reviewed intentional changes.
- `make test-core`, `make test`, `make parity`, `make check-boundaries`, and
  `make simplicity-audit` all pass.

## Execution Guardrails

- Do not broaden this sprint into cross-ADT tag identity or a whole-pipeline map audit.
- Preserve the indexed fast path only when lowerer-provided `KnownFields` proves the layout.
- Unknown layouts must use runtime record names; registered-type uniqueness is not proof of runtime type.
- Tests must assert semantics as well as repeatability, and new fixtures must be wired into CI.
- If opcode layout differs from the design, document the reason before expanding implementation scope.

## Handoff

After human approval, invoke `sprint-executor` with
`.ailang/state/sprints/sprint_M-VM-DETERMINISM.json`. Execution begins with M1's red-first evidence;
the executor must not mark M1 passing without recording how the pre-fix test failed.
