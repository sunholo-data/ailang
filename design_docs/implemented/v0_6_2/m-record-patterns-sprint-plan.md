# M-RECORD-PATTERNS Sprint Plan

**Sprint ID**: RECPAT
**Duration**: 1 day (~4-5 hours)
**Design Doc**: [m-record-patterns.md](m-record-patterns.md)

## Discovery Summary

**Good news:** Much of the infrastructure already exists!

| Component | Status | Work Needed |
|-----------|--------|-------------|
| `ast.RecordPattern` | ✅ Exists | None |
| `ast.FieldPattern` | ✅ Exists | None |
| `core.RecordPattern` | ✅ Exists | None |
| `eval_patterns.go` | ✅ Handles RecordPattern | None |
| `parseRecordPattern()` | ❌ Stub (returns nil) | **Implement** |
| `elaboratePattern()` | ❌ No case for RecordPattern | **Add case** |

**Revised estimate**: ~175 LOC total (down from 250-350)

## Milestones

### M1: Parser Implementation (~60 LOC, 1.5 hours)

**File**: `internal/parser/parser_pattern.go`

**Tasks**:
- [ ] Implement `parseRecordPattern()`
- [ ] Handle shorthand: `{name}` → `FieldPattern{Name: "name", Pattern: Identifier{Name: "name"}}`
- [ ] Handle renaming: `{name: n}` → `FieldPattern{Name: "name", Pattern: Identifier{Name: "n"}}`
- [ ] Handle nested: `{user: {name}}` → nested RecordPattern
- [ ] Handle rest: `{name, ...}` → `Rest: true`

**Acceptance Criteria**:
- [ ] `{name}` parses to RecordPattern with 1 field
- [ ] `{name: n}` parses with renaming
- [ ] `{a, b, c}` parses multiple fields
- [ ] `{user: {name}}` parses nested patterns
- [ ] Parser tests pass

### M2: Elaboration (~15 LOC, 30 min)

**File**: `internal/elaborate/patterns.go`

**Tasks**:
- [ ] Add `case *ast.RecordPattern:` in `elaboratePattern()`
- [ ] Convert `ast.FieldPattern` to `core.RecordPattern.Fields` map

**Acceptance Criteria**:
- [ ] RecordPattern elaborates to core.RecordPattern
- [ ] Nested patterns elaborate correctly

### M3: Tests (~80 LOC, 1 hour)

**Files**:
- `internal/parser/record_pattern_test.go` (new)
- `internal/elaborate/patterns_test.go` (extend)

**Tasks**:
- [ ] Parser unit tests for all syntax forms
- [ ] Elaboration tests
- [ ] Integration test: full pipeline

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Coverage for edge cases

### M4: Examples & Documentation (~20 LOC, 30 min)

**Files**:
- `examples/runnable/record_patterns.ail` (new)
- Update example with "PATTERN MATCHING (Future)" comment

**Tasks**:
- [ ] Create working example file
- [ ] Update/remove "future syntax" comments
- [ ] Verify example runs

**Acceptance Criteria**:
- [ ] `make verify-examples` passes
- [ ] Example demonstrates all syntax forms

## Timeline

| Time | Milestone | LOC |
|------|-----------|-----|
| 0:00-1:30 | M1: Parser | 60 |
| 1:30-2:00 | M2: Elaboration | 15 |
| 2:00-3:00 | M3: Tests | 80 |
| 3:00-3:30 | M4: Examples | 20 |
| 3:30-4:00 | Final verification | - |

**Total**: ~4 hours, ~175 LOC

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Token position bugs | Medium | Low | Use DEBUG_PARSER=1 |
| Nested pattern edge cases | Low | Medium | Comprehensive tests |

## Success Criteria

- [ ] `match person { {name, age} => ... }` works
- [ ] `match r { {name: n} => n }` works
- [ ] All existing tests pass
- [ ] New example file passes verification
- [ ] No regressions in pattern matching

---

**Created**: 2025-12-25
