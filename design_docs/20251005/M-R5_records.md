# M-R5: Records & Row Polymorphism

**Status**: 📋 Planned
**Priority**: P0 (CRITICAL - MUST SHIP)
**Estimated**: 500 LOC (350 impl + 150 tests)
**Duration**: 3 days
**Dependencies**: None
**Blocks**: Data modeling, structured data, APIs

## Problem Statement

**Current State**: Records partially work but have critical bugs.

### Problem 1: TRecord Unification Incomplete

```ailang
-- ⚠️ PARTIAL in v0.2.0
let user = {name: "Alice", age: 30} in user.name  -- Works sometimes
{x: 1}.x  -- Unification bug in some contexts
```

**Root Cause** (from CHANGELOG v0.2.0):
- `TRecord` has basic handler in unification
- Field-by-field unification incomplete
- No subset/subsumption rules
- Row variables not properly handled

**Evidence**:
```
TRecord Unification Support (internal/types/unification.go, ~40 LOC)
- Added handler for legacy *TRecord type in unification
- Fixed "unhandled type in unification" errors
- Improved record type checking with field-by-field unification
⚠️ Limitations: No row polymorphism, closed records only
```

### Problem 2: Row Polymorphism Not Implemented

```ailang
-- ❌ BROKEN in v0.2.0
-- Want: Polymorphic field access
func getName[r](obj: {name: string | r}) -> string {
  obj.name  -- Should work for ANY record with 'name' field
}

getName({name: "Alice", age: 30})  -- ❌ Type error
getName({name: "Bob", id: 123})    -- ❌ Type error
```

**Root Cause**:
- No row variables in type system
- Field access requires exact record type
- No subsumption: `{x:int, y:int}` doesn't unify with `{x:int}`

### Problem 3: Field Access Type Inference

```ailang
-- ⚠️ BROKEN in some cases
let r = {x: 1, y: 2} in r.x  -- Sometimes fails unification
```

**Root Cause**:
- Field access doesn't generate fresh row variable
- Unification fails when record type is inferred

## Goals

### Primary Goals (Must Achieve)
1. **TRecord unification works**: Field-by-field unification with proper rules
2. **Closed records usable**: `{x:int, y:int}` works reliably
3. **Field access works**: `record.field` type-checks correctly
4. **Subset unification**: `{x:1}` unifies with `{x:1, y:2}` (subsumption)
5. **Nested records work**: `{addr: {street: string}}.addr.street`

### Secondary Goals (Nice to Have - Partial OK)
6. **Row variables**: `{name:string | r}` representation
7. **Polymorphic access**: Functions with row-polymorphic parameters
8. **Row unification**: Unify `{x:int | r}` with `{x:int, y:int}`

**Note**: Full row polymorphism can be partial for v0.3.0. Basic closed records MUST work.

## Design

### Part 1: TRecord Unification (Must Have)

**Unification Rules**:

```
Rule 1: Exact match (closed records)
  {x:τ1, y:τ2} ~ {x:τ1', y:τ2'}
  if τ1 ~ τ1' and τ2 ~ τ2'

Rule 2: Subset subsumption (closed ⊆ open)
  {x:τ1} <: {x:τ1, y:τ2}
  (smaller record unifies with larger, used for field access)

Rule 3: Row extension (open records)
  {x:τ1 | ρ} ~ {x:τ1', y:τ2' | ρ'}
  if τ1 ~ τ1' and {| ρ} ~ {y:τ2' | ρ'}
```

**Implementation**:
```go
// internal/types/unification.go
func (u *Unifier) unifyRecord(r1, r2 *TRecord) error {
    // Case 1: Both closed - exact field match
    if r1.Row.Tail == nil && r2.Row.Tail == nil {
        return u.unifyClosedRecords(r1, r2)
    }

    // Case 2: One closed, one open - subsumption
    if r1.Row.Tail == nil && r2.Row.Tail != nil {
        return u.unifyClosed_Open(r1, r2)
    }

    // Case 3: Both open - row unification
    return u.unifyOpenRecords(r1, r2)
}

func (u *Unifier) unifyClosedRecords(r1, r2 *TRecord) error {
    // Check same fields exist
    if !sameFields(r1.Row.Labels, r2.Row.Labels) {
        return fmt.Errorf("record field mismatch: %v vs %v",
            fieldNames(r1), fieldNames(r2))
    }

    // Unify each field type
    for name, t1 := range r1.Row.Labels {
        t2 := r2.Row.Labels[name]
        if err := u.Unify(t1, t2); err != nil {
            return fmt.Errorf("field '%s': %w", name, err)
        }
    }
    return nil
}
```

### Part 2: Row Variables (Partial Implementation)

**Row Type Representation**:

```go
// internal/types/types.go
type Row struct {
    Kind   RowKind            // RecordRow or EffectRow
    Labels map[string]Type    // Field name → Field type
    Tail   *TVar              // Row variable (nil = closed row)
}

// Examples:
// Closed:  {x:int, y:int}     → Row{Labels: {x:int, y:int}, Tail: nil}
// Open:    {x:int | r}        → Row{Labels: {x:int}, Tail: TVar("r")}
```

**Field Access with Row Variable**:

```go
// internal/types/typechecker_core.go
func (tc *CoreTypeChecker) inferFieldAccess(record Expr, field string) (Type, error) {
    recordType := tc.infer(record)

    // Generate fresh row variable
    rowVar := tc.freshTVar()

    // Expected type: {field: α | ρ}
    fieldType := tc.freshTVar()
    expectedRecord := &TRecord{
        Row: &Row{
            Labels: map[string]Type{field: fieldType},
            Tail:   rowVar,
        },
    }

    // Unify with actual record type
    if err := tc.unify(recordType, expectedRecord); err != nil {
        return nil, fmt.Errorf("field '%s' not found: %w", field, err)
    }

    return fieldType, nil
}
```

### Part 3: Improved Error Messages

**Field Not Found**:
```
Type Error: Field access failed
  Record type: {x: int, y: int}
  Field: z

  Field 'z' not found in record.
  Available fields: x, y

  Hint: Check spelling or add field to record type.
```

**Field Type Mismatch**:
```
Type Error: Field type mismatch
  Record: {name: string, age: int}
  Field: age
  Expected: string
  Got: int

  Cannot unify field 'age' type.
```

## Implementation Plan

### Day 1: TRecord Unification (~200 LOC)

**Files to Modify**:
- `internal/types/unification.go` (~120 LOC)
- `internal/types/types.go` (~30 LOC)
- `internal/types/record_test.go` (~50 LOC new file)

**Tasks**:
1. Implement `unifyClosedRecords()` - exact field matching
2. Implement `unifyClosed_Open()` - subsumption (subset ⊆ superset)
3. Add field name validation and clear errors
4. Unit tests: closed record unification

**Test Cases**:
```go
// internal/types/record_test.go
func TestClosedRecordUnification(t *testing.T) {
    tests := []struct {
        name string
        r1   string
        r2   string
        ok   bool
    }{
        {"exact_match", "{x:int}", "{x:int}", true},
        {"field_mismatch", "{x:int}", "{y:int}", false},
        {"subset", "{x:int}", "{x:int, y:int}", true},  // subsumption
        {"type_mismatch", "{x:int}", "{x:string}", false},
    }
    // ...
}
```

### Day 2: Row Variables (~200 LOC)

**Files to Modify**:
- `internal/types/types.go` (~50 LOC)
- `internal/types/unification.go` (~80 LOC)
- `internal/types/typechecker_core.go` (~50 LOC)
- `internal/types/row_test.go` (~20 LOC new file)

**Tasks**:
1. Add `Tail *TVar` to Row struct (represent `{k:v | ρ}`)
2. Implement row variable unification (partial, best-effort)
3. Update field access to generate row variables
4. Unit tests: row variable basics

**Test Cases**:
```go
// internal/types/row_test.go
func TestRowVariables(t *testing.T) {
    tests := []struct {
        name string
        expr string
        typeStr string
    }{
        {"closed", "{x:1}", "{x:int}"},
        {"open", "{x:1 | r}", "{x:int | r}"},  // If parser supports
        {"access", "let r = {x:1, y:2} in r.x", "int"},
    }
    // ...
}
```

**Note**: Parser may not support `{| r}` syntax yet. That's OK - use internal representation only.

### Day 3: Field Access & Examples (~100 LOC)

**Files to Modify**:
- `internal/types/typechecker_core.go` (~40 LOC)
- `internal/types/errors.go` (~20 LOC)
- `examples/micro_record_person.ail` (~20 LOC new file)
- `examples/test_record_access.ail` (~20 LOC new file)

**Tasks**:
1. Fix field access type inference (use row variables)
2. Better error messages with field suggestions
3. Test nested records: `{addr: {street: string}}.addr.street`
4. Create example files

**Example Files**:
```ailang
// examples/micro_record_person.ail
module examples/micro_record_person

export type Person = {
  name: string,
  age: int
}

export func getName(p: Person) -> string {
  p.name
}

export func main() -> string {
  let alice = {name: "Alice", age: 30};
  getName(alice)  -- Returns: "Alice"
}
```

```ailang
// examples/test_record_access.ail
module examples/test_record_access

export func test_simple() -> int {
  {x: 1, y: 2}.x  -- Returns: 1
}

export func test_nested() -> string {
  let addr = {street: "Main St", city: "Boston"};
  let user = {name: "Bob", addr: addr};
  user.addr.street  -- Returns: "Main St"
}

export func main() -> string {
  test_nested()
}
```

## Acceptance Criteria

### Functional Requirements
- [ ] `{x:1, y:2}.x` returns 1
- [ ] Subset unification: `{x:1}` unifies with `{x:1, y:2}`
- [ ] Field type mismatch error: clear message with field name
- [ ] Missing field error: shows available fields
- [ ] Nested records: `{addr: {street: "Main"}}.addr.street` works
- [ ] Variable records: `let r = {x:1} in r.x` works

### Code Quality
- [ ] 100% test coverage for record unification
- [ ] Clear error messages with suggestions
- [ ] No regressions in existing tests

### Partial Row Polymorphism (Nice to Have)
- [ ] Row variables represented internally (Tail field)
- [ ] Field access generates row variables (best-effort)
- [ ] Partial unification for open records

**Note**: Full polymorphic functions `func[r](obj: {x:int | r})` can be incomplete. Basic field access MUST work.

## Risks & Mitigations

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **Row polymorphism too complex** | High | Medium | **Ship closed records only**; row vars internal-only |
| **Unification breaks existing code** | Medium | Low | Comprehensive test suite; subsumption opt-in |
| **Parser doesn't support row syntax** | Low | High | Use internal representation; defer syntax to v0.4.0 |
| **Performance regression** | Low | Low | Benchmark record-heavy code |

## Testing Strategy

### Unit Tests (~150 LOC)
- `internal/types/record_test.go`
  - Closed record unification (exact, subset)
  - Field type checking
  - Error messages and diagnostics

- `internal/types/row_test.go`
  - Row variable representation
  - Open record unification (best-effort)
  - Field access with row vars

### Integration Tests
- `examples/micro_record_person.ail` - Simple record usage
- `examples/test_record_access.ail` - Field access patterns
- `examples/test_record_nested.ail` - Nested records

### Regression Tests
- Re-run all existing record examples
- Check type inference doesn't break

## Success Metrics

| Metric | Target |
|--------|--------|
| **Record bugs fixed** | 3 (unification, subsumption, field access) |
| **Examples fixed** | +3-5 (record-using examples) |
| **Test coverage** | 100% for record unification |
| **Regressions** | 0 |

## Future Work (Deferred)

**v0.4.0 - Full Row Polymorphism**:
- Parser syntax: `{x:int | r}` in type annotations
- Polymorphic functions: `func[r](obj: {name:string | r})`
- Row unification algorithm (Rémy's algorithm)
- Scoped type variables with row kinds

**v0.4.0 - Record Extensions**:
- Record update syntax: `{user | age: 31}`
- Record concatenation: `{...r1, ...r2}`
- Record deletion: `{user \ age}`

**v0.5.0 - Structural Subtyping**:
- Width subtyping: more fields → fewer fields
- Depth subtyping: covariant field types
- Permutation: field order doesn't matter

## Implementation Notes

### Row Type Invariants

1. **Deterministic field order**: Always sort field names alphabetically
2. **No duplicate fields**: Parser/elaboration ensures uniqueness
3. **Tail variable**: `nil` = closed, `TVar(...)` = open
4. **Empty record**: `{}` has empty Labels, nil Tail

### Unification Algorithm

```
unify(T1, T2):
  case (TRecord(r1), TRecord(r2)):
    if closed(r1) && closed(r2):
      exact_match(r1, r2)  // All fields must match
    elif closed(r1) && open(r2):
      subsumption(r1, r2)  // r1 fields ⊆ r2 fields
    elif open(r1) && open(r2):
      row_unify(r1, r2)    // Best-effort row unification
    else:
      error("row kind mismatch")
```

### Error Code Taxonomy

- `TC_REC_001` - Field not found in record
- `TC_REC_002` - Field type mismatch
- `TC_REC_003` - Record type mismatch (different fields)
- `TC_REC_004` - Row variable unification failed

## References

- **CHANGELOG**: v0.2.0 - TRecord unification partial implementation
- **Issue**: Record field access bugs (user reports)
- **Design Doc**: `design_docs/planned/v0_3_0_implementation_plan.md`
- **Prior Art**: OCaml row polymorphism, PureScript records, Elm records
- **Algorithm**: Rémy's row polymorphism (simplified for v0.3.0)
