# Sprint Plan: M-SMT-RECORDS — Record Verification via SMT-LIB Datatypes

**Sprint ID**: M-SMT-RECORDS
**Design Doc**: [m-smt-fragment-expansion.md](m-smt-fragment-expansion.md) (Phase C)
**Target**: v0.8.0
**Duration**: 1 day (~6-8 hours)
**Risk Level**: Low-Medium

---

## Sprint Summary

Add record type verification to `ailang verify`. Records are pervasive in AILANG business logic and currently blocked from SMT verification. Z3's `declare-datatype` with named fields/accessors maps directly to AILANG records, making this a straightforward extension.

**Key insight**: SMT-LIB datatypes with a single constructor + named fields model records perfectly:
```smt2
(declare-datatype Point ((mk_Point (x Int) (y Int))))
```

---

## Current State

- **SMT implementation**: 2,041 LOC production / 2,256 LOC tests across 10 files in `internal/smt/`
- **Tests**: 103 tests, all passing
- **Record rejection points**:
  - `types.go:49-50` — `TRecord` returns error "record types cannot be encoded"
  - `codegen.go:250-254` — `Record` and `RecordAccess` return encoding errors
  - `encodable.go:406-411` — `Record`, `RecordAccess`, `RecordUpdate` return `true` (unencodable)
- **Reusable infrastructure**: `DeclareDatatype()` already supports named fields via `ADTVariant`/`ADTField`

---

## Milestones

### M1: Record Type Mapping — Map TRecord to SMT-LIB sort (~1.5h)

**What**: Extend `MapType()` in `types.go` to handle `TRecord` types by generating unique sort names and emitting `declare-datatype` declarations.

**Acceptance criteria**:
- `MapType(*TRecord{Fields: {x: int, y: int}})` returns `"Point"` (or generated name)
- `MapRecordType()` generates `(declare-datatype RecordName ((mk_RecordName (field1 Sort1) ...)))`
- Fields sorted alphabetically for deterministic output
- Named records (with `TypeName`) use that name; anonymous records get hash-based names
- Unit tests for simple records, multi-field records, records with ADT fields

**Files**:
- Modify `internal/smt/types.go` (+60 LOC)
- Add tests to `internal/smt/types_test.go` (+80 LOC)

**Estimated LOC**: ~140

---

### M2: Record Expression Encoding — Encode construction, access, update (~2h)

**What**: Add `Record`, `RecordAccess`, and `RecordUpdate` cases to `EncodeExpr()` in `codegen.go`.

**Acceptance criteria**:
- `Record{Fields: {x: 5, y: 10}}` encodes to `(mk_RecordName 5 10)` (alphabetical field order)
- `RecordAccess{Record: p, Field: "x"}` encodes to `(x p)` (accessor function call)
- `RecordUpdate{Base: p, Updates: {x: 20}}` encodes to `(mk_RecordName 20 (y p))` — reconstruct with updated fields
- Record type declarations emitted before function encoding
- Existing tests still pass (no regression)

**Files**:
- Modify `internal/smt/codegen.go` (+100 LOC)
- Add tests to `internal/smt/codegen_test.go` (+150 LOC)

**Estimated LOC**: ~250

---

### M3: Fragment Checker — Allow records in the decidable fragment (~30min)

**What**: Update `encodable.go` to stop rejecting records. Records with all-encodable field types should be accepted.

**Acceptance criteria**:
- `Record`, `RecordAccess`, `RecordUpdate` no longer auto-rejected
- Records with function-typed fields still rejected
- Records with string/list fields still rejected (until those phases land)
- Field types validated recursively
- All existing `IsSMTEncodable` tests pass + new record acceptance tests

**Files**:
- Modify `internal/smt/encodable.go` (+25 LOC)
- Add tests to `internal/smt/encodable_test.go` (+50 LOC)

**Estimated LOC**: ~75

---

### M4: Integration — End-to-end record verification + example (~1.5h)

**What**: Create `record_verify.ail` example demonstrating record contracts. Wire up type collection to emit record declarations in `EncodeFunction` and `verify.go`.

**Acceptance criteria**:
- `EncodeFunction` collects record types from function parameters and body
- Record type declarations emitted before ADT declarations
- New `examples/runnable/contracts/record_verify.ail` with 3+ verified functions
- Example includes: record construction, field access, ensures with record result fields
- `ailang verify record_verify.ail` succeeds
- `ailang run --caps IO --entry main record_verify.ail` succeeds
- `examples/manifest.json` updated

**Files**:
- Modify `internal/smt/codegen.go` (+30 LOC) — record type collection
- Modify `cmd/ailang/verify.go` (+15 LOC) — pass record types
- Create `examples/runnable/contracts/record_verify.ail` (~50 LOC)
- Modify `examples/manifest.json` (+10 LOC)

**Estimated LOC**: ~105

---

### M5: Documentation — Update contracts.mdx, CHANGELOG, decidable fragment (~45min)

**What**: Update docs to reflect records as now supported in the decidable fragment.

**Acceptance criteria**:
- `contracts.mdx` decidable fragment table updated (records: supported)
- Record verification section with example
- `CHANGELOG.md` updated with record verification entry
- Record limitations noted (no nested records with unsupported field types)

**Files**:
- Modify `docs/docs/guides/contracts.mdx` (+25 LOC)
- Modify `CHANGELOG.md` (+10 LOC)

**Estimated LOC**: ~35

---

## Success Metrics

| Metric | Target |
|--------|--------|
| All existing 103 SMT tests pass | ✅ |
| Record construction encodes correctly | ✅ |
| Record field access encodes correctly | ✅ |
| Record update encodes correctly | ✅ |
| `record_verify.ail` verifies with Z3 | ✅ |
| Total new LOC | ~605 (impl + tests) |

---

## Velocity Reference

- Phase A (Cross-Function Calls): ~780 LOC in ~3 hours
- This sprint: ~605 LOC estimated, similar complexity
- Estimated: 1 session (~6-8 hours)

---

## Technical Notes

### SMT-LIB Record Encoding Pattern

```smt2
; AILANG: type Point = { x: int, y: int }
(declare-datatype Point ((mk_Point (x Int) (y Int))))

; AILANG: let p = { x = 5, y = 10 }
(define-fun p () Point (mk_Point 5 10))

; AILANG: p.x
(x p)

; AILANG: { p with x = 20 }
(mk_Point 20 (y p))
```

### Record Type Collection

Records in Core AST carry type info via `CoreTypeInfo`. The encoder must:
1. Walk function parameters and body for `TRecord` types
2. Collect unique record type names
3. Emit `declare-datatype` for each unique record type
4. Handle named vs anonymous records

### Scoping Decisions

- **In scope**: Named record types, field access, functional update, records in contracts
- **Out of scope**: Row-polymorphic records (need monomorphization first), nested records with unsupported field types, record patterns in match expressions (defer to follow-up)
