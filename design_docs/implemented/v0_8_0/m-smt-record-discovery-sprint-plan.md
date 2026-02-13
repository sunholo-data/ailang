# Sprint Plan: M-SMT-RECORD-DISCOVERY

## Summary
Expand SMT record type discovery to find records from return types, function bodies, and ensures clauses — not just function parameters. Converts "unknown record type" ERRORs into clean VERIFY/VIOLATION/SKIP results.

**Duration:** 1 day (single session)
**Dependencies:** M-SMT-RECORDS (implemented), B1 allowlist fix (implemented this session)
**Risk Level:** Low — well-scoped, existing infrastructure, clear patterns to follow

## Current Status Analysis

### Completed Recently
- SMT fragment hardening (B1 allowlist, B3 float assertions, E1+E2 numeric conversions): ~130 LOC in current session
- SMT records implementation: ~200 LOC impl + ~200 LOC tests (v0.7.3)
- SMT strings/lists/cross-function: ~800 LOC impl + ~640 LOC tests (v0.7.3)

### Velocity
- Recent SMT work: ~2,695 insertions in last 10 commits to `internal/smt/`
- High familiarity with codegen.go, encodable.go, types.go from this session
- Estimate: 200 LOC/session for focused SMT work (implementation + tests)

### Remaining from Design Doc
- M1: Return type discovery (~30 LOC)
- M2: Body expression walking (~60 LOC)
- M3: Ensures clause walking + integration (~40 LOC)
- M4: Tests + example file (~100 LOC)

## Proposed Milestones

### Milestone 1: Return Type Discovery
**Goal:** Use the already-passed `returnSort` / return `types.Type` to discover record types from function return annotations.
**Estimated:** 30 LOC implementation + 30 LOC tests = 60 LOC
**Duration:** 30 min

**Tasks:**
- Add `returnType types.Type` parameter to `EncodeFunction` (via `EncodeFunctionOpts`)
- Pass `fd.ReturnType` converted to `types.Type` from `verify.go`
- Call `collectRecordType(returnType, ...)` in `collectAndDeclareRecordTypes`
- Unit test: function with record return type not in params → discovered

**Acceptance Criteria:**
- [ ] `collectAndDeclareRecordTypes` discovers record types from return annotation
- [ ] `EncodeFunction` accepts optional return `types.Type`
- [ ] Unit test passes: record in return only → VERIFY (not ERROR)
- [ ] Existing record_verify.ail results unchanged
- [ ] All tests passing, linting clean

**Risks:**
- `verify.go` uses `ast.Type` not `types.Type` — need a conversion. Mitigation: `astTypeToType` already exists at verify.go:470.

### Milestone 2: Body Expression Walking
**Goal:** Walk the Core AST function body to discover record types from `core.Record` nodes wherever they appear (let bindings, if branches, match arms, etc.).
**Estimated:** 60 LOC implementation + 40 LOC tests = 100 LOC
**Duration:** 45 min

**Tasks:**
- Implement `collectRecordTypesFromBody(body core.CoreExpr, ctx, result)` walker
- Handle all Core expression types: Record, Let, LetRec, If, Match, App, Lambda, BinOp, UnOp, Intrinsic, RecordAccess, RecordUpdate, List, Tuple
- Call from `collectAndDeclareRecordTypes` after param/return collection
- For `core.Record` nodes: infer field types from `CoreTypeInfo` or expression analysis, then call `collectRecordType`
- Unit tests: record in let binding, in if-then branch, in match arm

**Acceptance Criteria:**
- [ ] Body walker discovers `core.Record` nodes in all Core expression positions
- [ ] Record types from body are added to `activeRecordTypes` before encoding
- [ ] Unit tests pass: body-only record → VERIFY (not ERROR)
- [ ] All tests passing, linting clean

**Risks:**
- `core.Record` has `Fields map[string]core.CoreExpr` not `map[string]types.Type` — need to infer types from CoreTypeInfo or the record's own type annotation. Mitigation: use `CoreTypeInfo` lookup if available, or fall back to inferring from literal types in the record.

### Milestone 3: Ensures Clause Walking + Integration
**Goal:** Walk ensures clause expressions for record types, create integration test, and new example file.
**Estimated:** 40 LOC implementation + 60 LOC tests + 30 LOC example = 130 LOC
**Duration:** 45 min

**Tasks:**
- Pass ensures clause `contract.Expr` bodies through `collectRecordTypesFromBody`
- Create `examples/runnable/contracts/record_discovery_verify.ail` with 3 test cases:
  1. Return-only record type (makePoint)
  2. Body-only record in let binding (distance with delta record)
  3. Different param/return record types (transform)
- Run `ailang verify` on both old and new example files
- Run full `make test && make lint`

**Acceptance Criteria:**
- [ ] Ensures clauses with record references produce correct results
- [ ] New `record_discovery_verify.ail` example works
- [ ] Existing `record_verify.ail` unchanged (4 verified, 1 violation)
- [ ] `go test ./internal/smt/...` passes
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Body-only records may not have type annotations in Core AST — need runtime type inference from literal values. Mitigation: for records like `{x: 5, y: 10}`, field types can be inferred from int/float/string/bool literals.

## Success Metrics
- All existing SMT tests passing
- New tests for return/body/ensures record discovery
- `record_discovery_verify.ail` example passes verification
- No regressions in `record_verify.ail`
- `make test && make lint` clean

## Dependencies
- Z3 solver installed (for integration tests)
- B1 allowlist fix (completed this session)
- M-SMT-RECORDS (implemented in v0.7.3)

## Notes
- This sprint builds directly on infrastructure from the current session (B1/B3/E1/E2 fixes)
- The `containsRef` walker in encodable.go serves as the template for the body walker
- `collectRecordType` already handles nested records recursively — we just need to feed it more types
- Total estimated: ~290 LOC across 3 milestones in 1 day
