# M-SMT-RECORD-DISCOVERY: SMT Record Type Discovery from Bodies and Return Types

**Status**: Planned
**Target**: v0.8.0
**Priority**: P1 (Medium) — extends correctness of existing SMT verification
**Estimated**: 2 days (4h implementation + 4h testing + 2h edge cases + buffer)
**Dependencies**: M-SMT-RECORDS (implemented), M-SMT-FRAGMENT-EXPANSION (implemented: B1 allowlist fix)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to deterministic semantics |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Expands the set of functions verifiable by local SMT checks |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | More functions verified automatically without human intervention |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No cost model changes |
| A10: Composability | +1 | Functions returning records compose with verification pipeline |
| A11: Structured Failure | +1 | Replaces opaque "unknown record type" ERROR with clean verification or SKIP |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine analysis capability

## Problem Statement

**Record types constructed in function bodies or return positions are invisible to the SMT encoder**, causing "unknown record type" errors at codegen time.

The root cause is in `collectAndDeclareRecordTypes` ([codegen.go:825-831](internal/smt/codegen.go#L825-L831)):

```go
func collectAndDeclareRecordTypes(params []FunctionParam, returnSort string, ctx *SMTContext, result *EncodeResult) {
    for _, p := range params {
        collectRecordType(p.Type, ctx, result)
    }
    // BUG: returnSort is accepted but NEVER used
    // BUG: function body is NEVER walked
    // BUG: ensures clause expressions are NEVER walked
}
```

When `encodeRecord` encounters a record construction at [codegen.go:882-885](internal/smt/codegen.go#L882-L885), it fails:

```
record construction: unknown record type with fields [x y] (not declared in function signature)
```

**Current State:**
- `collectAndDeclareRecordTypes` only walks function parameter types (line 828-830)
- The `returnSort` parameter is passed from `EncodeFunction` (line 111) but **never consumed**
- [verify.go:202-204](cmd/ailang/verify.go#L202-L204) already extracts `returnSort` from `fd.ReturnType` and passes it to `EncodeFunction`
- The existing `record_verify.ail` example works **by coincidence** — `moveRight` returns `{x: int, y: int}`, but the same record type appears in its parameter, so it's already discovered

**Impact:**
- Functions that construct NEW record types in their body produce ERROR instead of clean SKIP
- Functions returning record types different from their parameters fail verification
- The M-SMT-RECORDS sprint plan mentions "Walk function parameters and body for TRecord types" (M4) but body walking was never implemented
- Limits the practical utility of contract verification for record-heavy code

## Goals

**Primary Goal:** Discover all record types used in a function (parameters, return type, body, ensures clauses) before SMT encoding begins.

**Success Metrics:**
- Functions returning records not in parameters produce clean VERIFY/VIOLATION/SKIP (never ERROR)
- No regression on existing `record_verify.ail` (4 verified, 1 violation)
- New test cases covering return-only, body-only, and ensures-clause record types
- Zero new panics or ERRORs in the full contract verification suite

## Solution Design

### Overview

Expand `collectAndDeclareRecordTypes` to discover records from **four sources** (currently only source 1):

1. **Function parameter types** (already implemented)
2. **Return type annotation** (passed but unused — `returnSort` param)
3. **Function body expressions** (walk Core AST for `core.Record` nodes)
4. **Ensures clause expressions** (walk contract body for record constructions)

### Architecture

**Component 1: Return Type Discovery**

The return type is already available as a `types.Type` in `EncodeFunction`'s caller. Currently `returnSort` is a string ("Int", "Bool", etc.) — for records we need the original `types.Type` to extract `TRecord` for field information.

Two approaches:
- **A (preferred):** Pass the original `types.Type` alongside `returnSort` string
- **B:** Parse `returnSort` string back to discover record sort names and look up in a type registry

Approach A is cleaner — `verify.go` already has the `types.Type` from `astTypeToSMTSort`.

**Component 2: Body Expression Walking**

Walk the Core AST body to find all `core.Record` nodes. For each, extract the field types and call `collectRecordType`. This handles:
- Record literals constructed in the body: `{x: 5, y: 10}`
- Record literals in let bindings: `let p = {x: a, y: b} in ...`
- Record literals in if/match branches

The walker must handle all Core expression types (similar to `containsRef` in [encodable.go](internal/smt/encodable.go)).

**Component 3: Ensures Clause Walking**

Ensures clauses can reference `$result` which may be a record. The clause body is already a `core.CoreExpr`, so the same body walker handles this. However, ensures clauses may also construct intermediate records for comparison.

### Implementation Plan

**Phase 1: Return Type Discovery** (~2 hours)
- [ ] Add `returnType types.Type` parameter to `collectAndDeclareRecordTypes`
- [ ] Update `EncodeFunction` signature to accept optional return type
- [ ] Update `verify.go` to pass the return `types.Type` through
- [ ] Call `collectRecordType(returnType, ...)` in `collectAndDeclareRecordTypes`
- [ ] Tests: function with record return type not in params

**Phase 2: Body Expression Walking** (~3 hours)
- [ ] Implement `collectRecordTypesFromBody(body core.CoreExpr, ctx, result)` — walks Core AST
- [ ] Handle all Core expression types: Record, Let, LetRec, If, Match, App, Lambda, etc.
- [ ] Call from `collectAndDeclareRecordTypes` after param/return collection
- [ ] Tests: record constructed only in body, nested in let/if/match

**Phase 3: Ensures Clause Walking** (~1.5 hours)
- [ ] Walk ensures clause bodies for record types (reuse body walker)
- [ ] Pass contract expressions to `collectRecordTypesFromBody`
- [ ] Tests: ensures clause referencing record type not in params or body

**Phase 4: Integration & Edge Cases** (~1.5 hours)
- [ ] Integration test: full verify pipeline with return-only records
- [ ] Edge case: anonymous records (no TypeName) in return position
- [ ] Edge case: nested records (record field is another record)
- [ ] Edge case: record in callee return type (cross-function)
- [ ] Verify existing `record_verify.ail` unchanged
- [ ] Create new example: `record_discovery_verify.ail`

### Files to Modify/Create

**Modified files:**
- `internal/smt/codegen.go` — Expand `collectAndDeclareRecordTypes`, add body walker (~60 LOC)
- `cmd/ailang/verify.go` — Pass `types.Type` for return type through to encoder (~10 LOC)

**New files:**
- `internal/smt/codegen_test.go` — Add record discovery tests (~80 LOC)
- `examples/runnable/contracts/record_discovery_verify.ail` — New example (~30 LOC)

## Examples

### Example 1: Return-Only Record Type (currently fails)

```ailang
-- Function returns a record type not present in any parameter
fn makePoint(x: int, y: int) -> {x: int, y: int}
  requires x >= 0 && y >= 0
  ensures $result.x == x && $result.y == y
= {x: x, y: y}
```

**Before (ERROR):**
```
makePoint ... ERROR: record construction: unknown record type with fields [x y] (not declared in function signature)
```

**After (VERIFIED):**
```
makePoint ... VERIFIED (Z3: unsat in 0.05s)
```

### Example 2: Record Constructed in Body Only

```ailang
-- Record type appears only in a let binding, not in params or return annotation
fn distance(x1: int, y1: int, x2: int, y2: int) -> int
  requires x1 >= 0 && y1 >= 0 && x2 >= 0 && y2 >= 0
  ensures $result >= 0
= let delta = {dx: x2 - x1, dy: y2 - y1}
  in delta.dx * delta.dx + delta.dy * delta.dy
```

**Before:** ERROR (record `{dx: int, dy: int}` not discovered)
**After:** VERIFIED (body walker finds the record construction)

### Example 3: Different Record Types in Param vs Return

```ailang
fn transform(input: {a: int, b: int}) -> {x: int, y: int}
  requires input.a >= 0
  ensures $result.x >= 0
= {x: input.a + 1, y: input.b * 2}
```

**Before:** ERROR (return record `{x, y}` not discovered — only param record `{a, b}` is known)
**After:** VERIFIED (return type discovery finds the second record type)

## Success Criteria

- [ ] Functions returning record types not in parameters produce VERIFY/VIOLATION/SKIP (not ERROR)
- [ ] Functions with record construction only in body produce VERIFY/VIOLATION/SKIP (not ERROR)
- [ ] Ensures clauses referencing records produce correct results
- [ ] Existing `record_verify.ail` results unchanged (4 verified, 1 violation)
- [ ] New `record_discovery_verify.ail` example passes
- [ ] All existing SMT tests passing (`go test ./internal/smt/...`)
- [ ] No regressions in full test suite (`make test`)

## Testing Strategy

**Unit tests:**
- `collectAndDeclareRecordTypes` with return type only (no params)
- `collectRecordTypesFromBody` with Record in Let, If, Match branches
- `encodeRecord` succeeds for body-discovered record types
- Anonymous vs named records in discovery

**Integration tests:**
- Full `EncodeFunction` pipeline with return-only record
- Full verify pipeline (`ailang verify`) with new example file
- Cross-function verification where callee returns a record

**Manual testing:**
- `ailang verify examples/runnable/contracts/record_verify.ail` — unchanged results
- `ailang verify examples/runnable/contracts/record_discovery_verify.ail` — new cases pass

## Non-Goals

**Not in this feature:**
- **Module-level type declaration discovery** — Walking module `type` declarations to find record types defined at module scope but not used in the current function. This requires access to the full module's type environment, which the SMT encoder doesn't currently receive. Deferred to a future sprint.
- **Record update expressions** — `{r | x: newVal}` constructs are already handled by `encodeRecordUpdate` but could benefit from body walking too. Low priority since record updates typically use the same type as the base record.
- **Polymorphic record types** — Records with type variables in field types (e.g., `{x: a, y: b}`) cannot be SMT-encoded regardless of discovery. Correctly rejected by the fragment checker.

## Timeline

**Day 1** (4 hours):
- Phase 1: Return type discovery + tests
- Phase 2: Body expression walker + tests

**Day 2** (4 hours):
- Phase 3: Ensures clause walking + tests
- Phase 4: Integration, edge cases, example file
- Documentation updates

**Total: ~8 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Body walker misses a Core node type → record not discovered | Medium | Comprehensive walker covering all 20+ Core node types; use `containsRef` as template |
| Performance: walking large function bodies | Low | Record collection is O(n) in AST size, same as existing fragment checks |
| Anonymous records with same fields but different intent | Low | Already handled by `MapRecordSortName` which generates deterministic names from sorted fields |
| Return type not available as `types.Type` in all paths | Medium | Fallback: parse `returnSort` string if `types.Type` not available; but verify.go already has it |

## Related Documents

**Directly related:**
- [design_docs/planned/v0_8_0/m-smt-records-sprint-plan.md](design_docs/planned/v0_8_0/m-smt-records-sprint-plan.md) — Original record implementation sprint (M4 mentions body walking as deferred)
- [design_docs/planned/v0_8_0/m-smt-fragment-expansion.md](design_docs/planned/v0_8_0/m-smt-fragment-expansion.md) — Parent design doc for SMT fragment hardening

**Implemented (informed design):**
- [design_docs/implemented/v0_6_2/m-record-patterns.md](design_docs/implemented/v0_6_2/m-record-patterns.md) — Record pattern matching in Core AST
- [design_docs/implemented/v0_6_1/m-dx16-inline-record-match-arms.md](design_docs/implemented/v0_6_1/m-dx16-inline-record-match-arms.md) — Inline record construction in match arms

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [internal/smt/codegen.go](internal/smt/codegen.go) — `collectAndDeclareRecordTypes` (line 825), `encodeRecord` (line 882)
- [cmd/ailang/verify.go](cmd/ailang/verify.go) — Return type extraction (line 202)
- [internal/smt/encodable.go](internal/smt/encodable.go) — `containsRef` walker as template for body walker

## Future Work

- **Module-level type declaration scanning** — Access the module's type environment to discover record types defined via `type Point = {x: int, y: int}` declarations. Requires architectural change to pass module type info to the SMT encoder.
- **Cross-function record discovery** — When a callee returns a record type, discover it from the callee's return type annotation for the caller's verification context.
- **Record update discovery** — Walk `RecordUpdate` nodes in the body to discover the base record type (currently relies on the base record being discovered through other means).

---

**Document created**: 2026-02-13
**Last updated**: 2026-02-13
