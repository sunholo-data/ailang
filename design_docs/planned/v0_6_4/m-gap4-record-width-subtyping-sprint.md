# Sprint Plan: M-GAP4 Record Width Subtyping

## Summary
Implement record width subtyping via row polymorphism to allow functions to accept records with extra fields beyond what they require. This enables writing generic functions that operate on record subsets without repeating full record types.

**Duration:** 2.5 days (10 hours)
**Dependencies:** None (extends existing row polymorphism infrastructure)
**Risk Level:** Medium (type system changes require careful testing)

## Current Status Analysis

### Completed Recently
- v0.6.3: OpenAI Responses API Support (~190 LOC)
- v0.6.3: Enhanced OTEL tracing (~260 LOC)
- v0.6.2: OpenTelemetry Integration (~725 LOC)
- v0.6.2: GitHub-Driven Autonomous Workflow (~1,300 LOC)

### Velocity
- Recent average: ~200-300 LOC/day based on CHANGELOG entries
- Estimated capacity: ~300 LOC for this sprint (type system work is more complex)

### Current Implementation Status
- **TRecord2**: New row-polymorphic record type exists
- **TRecordOpen**: Open record type for subsumption exists
- **RowUnifier**: Row unification logic implemented
- **Gap**: `unifyRecord()` enforces exact field count matching (line 66-67)

```go
// Current behavior (blocking width subtyping):
if len(t1.Fields) != len(t2.Fields) {
    return nil, fmt.Errorf("record field count mismatch: %d vs %d", len(t1.Fields), len(t2.Fields))
}
```

### Files to Modify (from design doc)
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/types/unification_records.go` | Width subtyping in unification | ~50 |
| `internal/types/inference.go` | Implicit row variables for params | ~30 |
| `internal/types/types.go` or `types_v2.go` | Helper functions | ~20 |
| `internal/errors/codes.go` | Better error codes | ~20 |

## Proposed Milestones

### Milestone 1: Unification Width Subtyping (Day 1)
**Goal:** Modify `unifyRecord()` to allow records with extra fields when unifying at function boundaries.

**Estimated:** 80 LOC implementation + 60 LOC tests = 140 LOC total
**Duration:** 4 hours

**Tasks:**
1. **Hour 1-2:** Modify `unifyRecord()` in `unification_records.go`
   - Remove strict field count check
   - When t1 has more fields than t2, check t2 is a subset of t1
   - Create row extension for extra fields
   - Handle both directions (t1 > t2 and t2 > t1)

2. **Hour 3:** Update `unifyRows()` for width subtyping semantics
   - When unifying closed row with fewer fields against record with more fields
   - Extra fields captured in row variable

3. **Hour 4:** Write unit tests
   - Basic width subtyping: `{a, b, c}` unifies with `{a}`
   - Nested records with width subtyping
   - Lists of records with extra fields
   - Error cases: missing required fields

**Acceptance Criteria:**
- [ ] `{a: int, b: string}` unifies with `{a: int | r}` where r captures `{b: string}`
- [ ] Existing exact-match tests still pass
- [ ] TRecord2 row unification handles width subtyping
- [ ] All existing type inference tests pass

**Risks:**
- Breaking existing code relying on exact matching - Mitigation: Only apply at function boundaries, not type aliases

### Milestone 2: Implicit Row Variables for Parameters (Day 2)
**Goal:** Infer implicit row variables for record parameters so `{a: T}` in parameter position means "at least `a`".

**Estimated:** 60 LOC implementation + 50 LOC tests = 110 LOC total
**Duration:** 4 hours

**Tasks:**
1. **Hour 1-2:** Modify type inference in `typechecker_data.go`
   - When inferring record parameter types, add implicit row variable
   - `{a: T}` in parameter position becomes `{a: T | r}` with fresh `r`
   - Ensure explicit row syntax still works

2. **Hour 3:** Update elaboration to propagate row variables
   - Check `internal/elaborate/elaborate.go` for record handling
   - Ensure Core AST preserves row variable information

3. **Hour 4:** Write inference tests
   - Function with record parameter accepts wider records
   - Type aliases preserve exact semantics
   - Explicit row polymorphism still works as before

**Acceptance Criteria:**
- [ ] `func f(x: {name: string}) -> string` accepts `{name: "Alice", age: 30}`
- [ ] Type aliases remain exact: `type Person = {name: string}` means exactly those fields
- [ ] Explicit `{name: string | r}` syntax continues to work
- [ ] No inference regression on existing examples

**Risks:**
- Over-generalization making inference unpredictable - Mitigation: Only implicit at parameter boundaries

### Milestone 3: Error Messages & Documentation (Day 3)
**Goal:** Improve error messages for record mismatches and update documentation.

**Estimated:** 40 LOC implementation + 30 LOC tests = 70 LOC total
**Duration:** 2 hours

**Tasks:**
1. **Hour 1:** Improve error messages
   - Show which fields are missing (not just "field count mismatch")
   - Suggest using row polymorphism explicitly when appropriate
   - Add error code in `internal/errors/codes.go`

2. **Hour 2:** Create example file and update docs
   - Create `examples/runnable/record_width_subtyping.ail`
   - Update prompts/v0.6.5.md with width subtyping syntax
   - Update CHANGELOG.md with feature description

**Acceptance Criteria:**
- [ ] Error message shows missing fields: "record missing field 'name', has: {age: int}"
- [ ] `examples/runnable/record_width_subtyping.ail` passes verification
- [ ] Documentation updated
- [ ] CHANGELOG entry added

**Risks:**
- Low - Documentation and error messages are low-risk changes

## Success Metrics
- Test coverage: Maintain current coverage (no decrease)
- Examples passing: record_width_subtyping.ail works
- Documentation: prompts/v0.6.5.md updated, CHANGELOG entry
- All tests passing: `make test`
- All linting passing: `make lint`
- Existing record tests: No regressions

## Dependencies
- None - extends existing row polymorphism infrastructure

## Open Questions
1. **Should width subtyping apply to return types?**
   - Recommendation: No, only parameters (covariant position)

2. **Should pattern matching on records require exact match?**
   - Recommendation: Yes, patterns are more specific than parameters

## Implementation Notes

### Key Code Locations
- **Unification entry point:** `internal/types/unification_core.go:216-222`
- **Record unification:** `internal/types/unification_records.go:62-129` (TRecord)
- **Row unification:** `internal/types/row_unification.go:34-154`
- **TRecordOpen:** `internal/types/unification_records.go:131-233` (already supports subsumption)

### Existing Infrastructure to Leverage
- `TRecordOpen` already implements subsumption logic - can reuse patterns
- `RowUnifier.UnifyRows()` handles open/closed row combinations
- `RecordHasField()` and `RecordFieldType()` helpers exist

### Test File Locations
- `internal/types/record_unification_test.go` - add width subtyping tests
- `internal/types/row_unification_regression_test.go` - ensure no regressions

## Timeline Summary

| Day | Hours | Milestone | Deliverable |
|-----|-------|-----------|-------------|
| 1 | 4h | M1: Unification | Width subtyping in unifyRecord() |
| 2 | 4h | M2: Inference | Implicit row vars for params |
| 3 | 2h | M3: Polish | Error messages, docs, examples |

**Total:** 10 hours / 2.5 days
**Estimated LOC:** ~320 (implementation + tests)
