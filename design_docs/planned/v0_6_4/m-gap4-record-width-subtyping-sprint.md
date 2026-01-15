# Sprint Plan: M-GAP4 - Record Width Subtyping via Row Polymorphism

## Summary
Enable explicit width subtyping for records using open record syntax (`{a: T | r}` or `{a: T, ..}`), allowing functions to accept records with additional fields beyond what they require.

**Duration:** 2-3 days (~12 hours)
**Dependencies:** None
**Risk Level:** Medium (type system changes require careful testing)

## Current Status Analysis

### Existing Infrastructure (Good News!)
After analyzing the codebase, AILANG already has substantial row polymorphism infrastructure:

- **`TRecordOpen`** type exists (`internal/types/types.go`) - open records with row variable
- **`unifyRecordOpen`** handles subsumption (`internal/types/unification_records.go:132`)
- **`unifyRows`** already handles field absorption into tails with occurs check (line 236-314)
- **`TRecord2`** with proper `Row` structure for new row-polymorphic records

### The Gap
The issue is:
1. `unifyRecord` (line 66-67) enforces exact field count for `TRecord` vs `TRecord`
2. User-facing syntax for `{a: T | r}` needs parser support verification
3. Sugar syntax `{a: T, ..}` not implemented
4. Error messages don't suggest open record syntax

### Design Doc Reference
- Design doc: `design_docs/planned/v0_6_4/m-gap4-record-width-subtyping.md`
- Approach: Explicit openness only (no implicit widening)
- Exact records remain exact, open records accept extras

### Velocity
- Recent stdlib additions: ~25 LOC/function
- Type system work: ~100-150 LOC/day based on recent unification_records.go additions
- Estimated capacity: ~150 LOC/day for complex type work

## Open Questions (Day 1 Investigation)

Before full implementation, need to answer:

1. **Parser status:** Does `{a: T | r}` parse to `TRecordOpen` today?
   - Check: `internal/parser/parser.go` for record type parsing
   - Test: Try parsing `{a: T | r}` in type position

2. **Which type is used:** Are record literals `TRecord` or `TRecord2`?
   - This affects which unification path is taken
   - Check: `internal/elaborate/elaborate.go` for record literal elaboration

3. **Integration point:** Does type inference generate `TRecordOpen` from annotations?
   - Check: `internal/types/infer.go` for annotation handling

4. **Runtime representation:** Can generated Go code handle open records?
   - Check: `internal/codegen/codegen.go` for record compilation

## Proposed Milestones

### Milestone 1: Investigation & Parser Verification
**Goal:** Answer open questions, verify parser supports row syntax
**Estimated:** ~2 hours
**Duration:** Day 1 morning

**Tasks:**
1. Test if `{a: T | r}` parses correctly in type position
2. Trace record literal elaboration path (TRecord vs TRecord2)
3. Check if `TRecordOpen` is ever generated from user code
4. Document current type flow for records

**Investigation Commands:**
```bash
# Test parsing
echo 'func f(x: {name: string | r}) -> string = x.name' | ailang check --stdin

# Search for TRecordOpen usage
grep -r "TRecordOpen" internal/

# Check parser record handling
grep -n "parseRecordType" internal/parser/
```

**Acceptance Criteria:**
- [ ] Document: Does `{a: T | r}` parse today? (Y/N)
- [ ] Document: Which record type is used for literals?
- [ ] Document: Integration point for open record syntax
- [ ] Updated design doc with findings

**Risks:**
- Parser may not support row syntax yet (then need parser work)
- Multiple record type representations may cause complexity

### Milestone 2: Enable Open Record Unification
**Goal:** Make `TRecordOpen ~ TRecord` unification work for subsumption
**Estimated:** ~80 LOC implementation + ~50 LOC tests = ~130 LOC
**Duration:** Day 1 afternoon + Day 2 morning

**Tasks:**
1. If parser support missing: Add `| r` syntax to record type parsing
2. Ensure `TRecordOpen` flows through type inference
3. Verify `unifyRecordOpen` handles all cases correctly
4. Add row variable solving (residuals → row tail)
5. Add comprehensive tests

**Implementation Focus:**
```go
// Key changes to internal/types/unification_records.go

// Fix unifyRecordOpen to properly absorb residual fields
func (u *Unifier) unifyRecordOpen(t1 *TRecordOpen, t2 Type, sub Substitution) (Substitution, error) {
    // Existing code handles TRecord case (line 134-176)
    // Need to verify:
    // 1. Row variable solving works (remainingFields → row)
    // 2. Error messages suggest open record syntax
}
```

**Files to Modify:**
| File | Change |
|------|--------|
| `internal/parser/parser.go` | Row syntax parsing (if needed) |
| `internal/types/unification_records.go` | Row variable solving |
| `internal/types/unification_records_test.go` | New test cases |

**Acceptance Criteria:**
- [ ] `{a: T | r}` unifies with `{a: T, b: U}` yielding `r := {b: U}`
- [ ] `{a: T}` remains exact (rejects extra fields)
- [ ] Test file with 10+ test cases passes
- [ ] `make test` passes

**Risks:**
- Row variable solving complexity (existing code has TODOs at lines 167-173)
- Multiple record type representations (`TRecord`, `TRecord2`, `TRecordOpen`)

### Milestone 3: Sugar Syntax `{a: T, ..}` (Optional)
**Goal:** Add shorthand syntax for open records
**Estimated:** ~30 LOC parser + ~20 LOC tests = ~50 LOC
**Duration:** Day 2 afternoon (1-2 hours)

**Tasks:**
1. Add `DOTDOT` token to lexer (if not present)
2. Parse `{a: T, ..}` in record type position
3. Desugar to `TRecordOpen` with fresh row variable
4. Add tests for sugar syntax

**Acceptance Criteria:**
- [ ] `{a: T, ..}` parses and works identically to `{a: T | r}`
- [ ] Parser tests cover sugar syntax
- [ ] Documentation example uses both syntaxes

**Risks:**
- Low - straightforward parser addition
- Can be deferred if time constrained

### Milestone 4: Error Messages & Documentation
**Goal:** Improve DX with helpful error messages
**Estimated:** ~40 LOC errors + ~20 LOC docs = ~60 LOC
**Duration:** Day 3 (2-3 hours)

**Tasks:**
1. Add structured error for record mismatch:
   - List missing fields
   - List extra fields (when exact record rejects)
   - Suggest: "use `{field: T | r}` to accept extra fields"
2. Create example file `examples/runnable/record_width_subtyping.ail`
3. Update design doc status to Implemented
4. Update teaching prompt if needed

**Error Message Format:**
```
Type error: record field mismatch
  Expected: {name: string}
  Got: {name: string, age: int}

  Extra fields: age

  Hint: Use open record {name: string | r} to accept extra fields
```

**Acceptance Criteria:**
- [ ] Error messages list missing/extra fields
- [ ] Error messages suggest open record syntax
- [ ] Example file works: `examples/runnable/record_width_subtyping.ail`
- [ ] Design doc status updated
- [ ] `make lint` passes

**Risks:**
- None - straightforward improvements

## Success Metrics

- **Core functionality:** `{a: T | r}` unifies with `{a: T, b: U}` correctly
- **Exact records preserved:** `{a: T}` rejects extra fields
- **Test coverage:** 15+ new test cases
- **Examples passing:** `record_width_subtyping.ail` works
- **Documentation:** Design doc updated to Implemented
- **All tests passing:** `make test`
- **All linting passing:** `make lint`

## Dependencies

- None - builds on existing row polymorphism infrastructure

## Timeline

**Day 1:** Investigation + Core unification (~6 hours)
- Morning: Answer open questions, trace type flow
- Afternoon: Implement/fix row variable solving

**Day 2:** Testing + Sugar syntax (~4 hours)
- Morning: Comprehensive test suite
- Afternoon: `{a: T, ..}` sugar (optional)

**Day 3:** Error messages + Documentation (~2 hours)
- Error message improvements
- Example file and docs

## Notes

- Existing infrastructure is more complete than expected
- Key insight: `unifyRecordOpen` already exists (lines 132-232) but may have incomplete row variable solving (lines 167-173 have TODOs)
- `TRecord` vs `TRecord2` distinction needs investigation - may affect which path is used
- Sugar syntax is optional - can ship without `..` if time constrained
- Focus on explicit openness - no implicit widening per design doc

## Coordinator Integration

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_6_4/m-gap4-record-width-subtyping-sprint.md`
